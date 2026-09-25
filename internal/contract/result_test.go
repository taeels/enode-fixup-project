package contract

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

// 종료 보고의 본문 검사 — 규칙마다 거절 한 줄과 받는 본문 셋 (step-phase FD 규칙 2절).
func TestExited_Check(t *testing.T) {
	code := func(n int) *int { return &n }
	at := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	good := func() Exited {
		return Exited{Node: "n1", Instance: "i1", Attempt: 0,
			Outcome: Outcome{Kind: OutcomeExit, Code: code(0)}, ExitedAt: at}
	}
	cases := []struct {
		name string
		edit func(*Exited)
		want string
	}{
		{"exit with a code", func(e *Exited) {}, ""},
		{"exit with a non-zero code", func(e *Exited) { e.Outcome.Code = code(2) }, ""},
		{"signal without a code", func(e *Exited) { e.Outcome = Outcome{Kind: OutcomeSignal} }, ""},
		{"signal with a number", func(e *Exited) { e.Outcome = Outcome{Kind: OutcomeSignal, Code: code(9)} }, ""},
		{"timeout without a code", func(e *Exited) { e.Outcome = Outcome{Kind: OutcomeTimeout} }, ""},
		{"a retry attempt", func(e *Exited) { e.Attempt = 3 }, ""},

		{"empty node", func(e *Exited) { e.Node = "" }, "exited: node is empty"},
		{"empty instance", func(e *Exited) { e.Instance = "" },
			"exited: instance is empty; the report is matched against the instance that claimed the step"},
		{"negative attempt", func(e *Exited) { e.Attempt = -1 }, "exited: attempt must not be negative"},
		{"no exited_at", func(e *Exited) { e.ExitedAt = time.Time{} }, "exited: exited_at is missing"},
		{"unknown kind", func(e *Exited) { e.Outcome.Kind = "crash" },
			`exited: outcome.kind "crash" is not exit, signal or timeout`},
		{"empty kind", func(e *Exited) { e.Outcome.Kind = "" },
			`exited: outcome.kind "" is not exit, signal or timeout`},
		{"exit without a code", func(e *Exited) { e.Outcome.Code = nil }, "exited: outcome.kind exit needs a code"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := good()
			tc.edit(&e)
			err := e.Check()
			if tc.want == "" {
				if err != nil {
					t.Fatalf("Check = %v, want nil", err)
				}
				return
			}
			if err == nil || err.Error() != tc.want {
				t.Fatalf("Check = %v, want %q", err, tc.want)
			}
		})
	}
}

// 종료 보고는 RFC 3339 시각으로 오간다 — 노드가 묶은 것을 Mediator 가 같은 값으로 푼다.
func TestExited_RoundTrip(t *testing.T) {
	var e Exited
	body := `{"node":"n1","instance":"i1","attempt":1,
	  "outcome":{"kind":"exit","code":2},"exited_at":"2026-09-25T09:00:00.5Z"}`
	if err := json.Unmarshal([]byte(body), &e); err != nil {
		t.Fatal(err)
	}
	if err := e.Check(); err != nil {
		t.Fatalf("Check = %v", err)
	}
	if *e.Outcome.Code != 2 || e.ExitedAt.Nanosecond() != 5e8 {
		t.Fatalf("decoded %+v", e)
	}
	b, err := json.Marshal(Outcome{Kind: OutcomeTimeout})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"kind":"timeout"}` {
		t.Fatalf("an outcome without a code carries %s", b)
	}
}

// jsonKeys 는 v 를 묶었을 때 가장 바깥 객체의 키다.
func jsonKeys(t *testing.T, v any) []string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// 결과 어휘의 칸 이름은 FD 엔티티 2절 그대로다. 봉인된 Record 를 읽는 쪽이 이 이름을
// 알므로 바뀌면 이 시험이 깨져야 한다 — 이 파일의 모양은 더하기만 한다.
func TestResultVocabulary_FieldNames(t *testing.T) {
	ir := "ir-1"
	at := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		v    any
		want string
	}{
		{"exited", Exited{}, "attempt exited_at instance node outcome"},
		{"empty diagnostics", Diagnostics{}, "changes effect"},
		{"full diagnostics", Diagnostics{Missing: []string{"a"}, Collect: []CollectNote{{"a", "b"}},
			Changes: ChangesPartial, Effect: EffectEdit, Discovered: []string{"x"}, DiscoveryLimit: LimitVisits},
			"changes collect discovered discovery_limit effect missing"},
		{"collect note", CollectNote{}, "name why"},
		{"empty capture", CheckpointCapture{State: CaptureNotRequested}, "state"},
		{"captured", CheckpointCapture{State: CaptureCaptured, Reason: "r", ID: "c1", Scope: "workspace-upper",
			Guarantee: "inspect-only", Node: "n1", ExpiresAt: &at},
			"expires_at guarantee id node reason scope state"},
		{"build manifest", BuildManifest{}, "builds head ir pinned sync"},
		{"build record", BuildRecord{}, "command exit_code finished_at name started_at"},
		{"pinned", Pinned{}, "file sha256"},
		{"merge result", MergeResult{IR: &ir}, "ir merged_at ops previous_ir resumed"},
		{"merge ops", MergeOps{}, "attrs created dirs opaque replaced trashed whiteouts"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := strings.Join(jsonKeys(t, tc.v), " "); got != tc.want {
				t.Fatalf("keys = %q, want %q", got, tc.want)
			}
		})
	}
}

// not_measured 는 「바뀐 파일이 없다」와 다르다 — changes 칸은 빈 값이어도 빠지지 않는다.
// ir 과 pinned 는 없으면 null 로 나간다 — 「태그가 없다」가 칸의 부재와 섞이지 않는다.
func TestResultVocabulary_KeysThatStayWhenEmpty(t *testing.T) {
	b, err := json.Marshal(struct {
		D Diagnostics   `json:"d"`
		B BuildManifest `json:"b"`
		M MergeResult   `json:"m"`
	}{})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"changes":""`, `"ir":null`, `"pinned":null`, `"previous_ir":null`} {
		if !strings.Contains(string(b), want) {
			t.Fatalf("%s does not carry %s", b, want)
		}
	}
}

// Stage 는 문자열 그대로 오간다.
func TestStage_RoundTrip(t *testing.T) {
	var got struct {
		F Stage `json:"f"`
	}
	if err := json.Unmarshal([]byte(`{"f":"timeout"}`), &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.F, StageTimeout) {
		t.Fatalf("stage = %q", got.F)
	}
}
