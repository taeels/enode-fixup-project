package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// renderMediator 는 RenderResult · summaryOf 가 읽는 두 라우트를 세운다.
func renderMediator(t *testing.T, h http.HandlerFunc) *Mediator {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return NewMediator(srv.URL, "fleet-token", "")
}

// notFound 는 원장도 산출물도 없는 Mediator 다.
func notFound(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotFound)
}

func runViewFrom(t *testing.T, raw string) *RunView {
	t.Helper()
	var run RunView
	if err := json.Unmarshal([]byte(raw), &run); err != nil {
		t.Fatalf("cannot read the run lookup: %v", err)
	}
	return &run
}

// 머리글이 상태를 그대로 옮긴다 — 사람이 코멘트 첫 줄만 보고 판정을 안다.
func TestRenderResult_HeadlineFollowsTheRunState(t *testing.T) {
	for _, tc := range []struct {
		state string
		head  string
	}{
		{"SUCCEEDED", "끝났습니다"},
		{"FAILED", "실패했습니다"},
		{"CANCELLED", "취소됐습니다"},
		{"EXPIRED", "시간이 다 됐습니다"},
	} {
		t.Run(tc.state, func(t *testing.T) {
			m := renderMediator(t, notFound)
			run := &RunView{RunID: "itsaplan-EP-2-1", State: tc.state}
			got := RenderResult(context.Background(), m, run, "")
			first := strings.SplitN(got, "\n", 2)[0]
			if !strings.Contains(first, tc.head) {
				t.Fatalf("headline = %q, want it to say %q", first, tc.head)
			}
			// 기계가 읽는 상태 이름도 남는다 — 사람 말과 코드가 둘 다 필요하다.
			if !strings.Contains(first, "`"+tc.state+"`") {
				t.Fatalf("headline = %q lost the state code", first)
			}
		})
	}
}

// 어느 단계가 어느 노드에서 돌았는지는 여기 표로만 보인다 (comment.go 머리 주석) —
// 러너의 존재감은 last_seen_at 한 칸이라 UI 로는 초록불 하나다.
func TestRenderResult_ShowsWhichNodeRanWhichStep(t *testing.T) {
	m := renderMediator(t, notFound)
	run := runViewFrom(t, `{
	  "run_id": "itsaplan-EP-2-1",
	  "state": "SUCCEEDED",
	  "assigned": [{"as":"worker","nodes":[{"node":"n-1","label":"taeels@host:orch-EP-2"}]}],
	  "steps": [
	    {"seq":1,"id":"plan","state":"SUCCEEDED","uses":"planner","node":"n-1",
	     "started_at":"2026-08-24T00:00:00Z","ended_at":"2026-08-24T00:00:01.234Z"},
	    {"seq":2,"id":"gate","state":"SUCCEEDED","uses":"","node":""},
	    {"seq":3,"id":"build","state":"FAILED","uses":"worker","node":"n-9"}
	  ]
	}`)

	got := RenderResult(context.Background(), m, run, "")

	if !strings.Contains(got, "| # | 단계 | 상태 | 역할 | 노드 | 걸린 시간 |") {
		t.Fatalf("the step table is missing:\n%s", got)
	}
	// ① 라벨이 있으면 사람이 읽는 이름으로 바뀐다.
	if !strings.Contains(got, "taeels@host:orch-EP-2") {
		t.Fatalf("the node label was not substituted:\n%s", got)
	}
	// ② 라벨이 없으면 노드 id 를 그대로 쓴다 — 빈 칸보다 낫다.
	if !strings.Contains(got, "n-9") {
		t.Fatalf("an unlabelled node lost its id:\n%s", got)
	}
	// ③ 노드가 아예 없는 단계는 대시로 닫는다.
	if !strings.Contains(got, "| 2 | `gate` | SUCCEEDED | (사람) | — | — |") {
		t.Fatalf("a human step is not rendered as (사람) with dashes:\n%s", got)
	}
	// 걸린 시간은 100ms 로 반올림한다.
	if !strings.Contains(got, "1.2s") {
		t.Fatalf("the elapsed time is missing:\n%s", got)
	}
}

// 무엇을 왜 통과·실패했나 — 실패한 것만 「실제」를 붙인다. 통과한 검사에
// 실제값을 붙이면 표가 두 배로 길어지고 읽을 이유가 없다.
func TestRenderResult_ExplainsWhyEachCheckPassedOrFailed(t *testing.T) {
	m := renderMediator(t, notFound)
	run := runViewFrom(t, `{
	  "run_id": "itsaplan-EP-2-1",
	  "state": "FAILED",
	  "verdict": {"state":"FAILED","checks":[
	    {"step":"plan","what":"produced","want":["plan"],"got":["plan"],"ok":true},
	    {"step":"build","what":"exit_code","want":0,"got":2,"ok":false,"note":"the compiler refused"},
	    {"step":"gate","what":"answered","ok":true}
	  ]}
	}`)

	got := RenderResult(context.Background(), m, run, "")

	if !strings.Contains(got, "- 통과 `plan` — produced (요구 `plan`)") {
		t.Fatalf("a passing check does not read cleanly:\n%s", got)
	}
	if !strings.Contains(got, "- 실패 `build` — exit_code (요구 `0` · 실제 `2`) — the compiler refused") {
		t.Fatalf("a failing check lost its want/got/note:\n%s", got)
	}
	// want 가 없으면 괄호 자체가 안 붙는다.
	if !strings.Contains(got, "- 통과 `gate` — answered\n") {
		t.Fatalf("a check with no want grew an empty bracket:\n%s", got)
	}
	// 통과한 검사에는 실제값이 안 붙는다.
	if strings.Contains(got, "요구 `plan` · 실제") {
		t.Fatalf("a passing check printed its actual value:\n%s", got)
	}
}

// 원장은 blobs/ 에서 유도된다 (ADR-040 §3.3 정정) — seq 다음 attempt 순으로
// 안정 정렬해야 재시도가 원래 시도 아래에 붙는다.
func TestRenderResult_ListsTheLedgerInOrder(t *testing.T) {
	m := renderMediator(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/ledger") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(`{"entries":[
		  {"seq":3,"attempt":1,"name":"report","by":"worker","bytes":12,"schema_ok":null},
		  {"seq":1,"attempt":2,"name":"plan","by":"planner","bytes":200,"schema_ok":false},
		  {"seq":1,"attempt":1,"name":"plan","by":"planner","bytes":180,"schema_ok":true}]}`))
	})
	run := &RunView{RunID: "itsaplan-EP-2-1", State: "SUCCEEDED"}

	got := RenderResult(context.Background(), m, run, "")

	if !strings.Contains(got, "<summary><b>남은 산출물</b></summary>") {
		t.Fatalf("the ledger section is missing:\n%s", got)
	}
	rows := []string{
		"| `plan` | planner | 1 | 180 | 통과 |",
		"| `plan` | planner | 2 | 200 | 실패 |",
		"| `report` | worker | 1 | 12 | — |",
	}
	at := -1
	for _, row := range rows {
		i := strings.Index(got, row)
		if i < 0 {
			t.Fatalf("the ledger row %q is missing:\n%s", row, got)
		}
		if i < at {
			t.Fatalf("the ledger is not sorted by seq then attempt:\n%s", got)
		}
		at = i
	}
}

// 원장을 못 읽어도 코멘트는 나가야 한다 — 판정은 이미 끝났고 사람은 결과를
// 봐야 한다 (ADR-040 §6 의 부분 성공).
func TestRenderResult_SurvivesALedgerFailure(t *testing.T) {
	m := renderMediator(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	run := &RunView{RunID: "itsaplan-EP-2-1", State: "SUCCEEDED"}

	got := RenderResult(context.Background(), m, run, "the work is done")

	if strings.Contains(got, "남은 산출물") {
		t.Fatalf("an unreadable ledger produced an empty section:\n%s", got)
	}
	if !strings.Contains(got, "the work is done") {
		t.Fatalf("the summary was dropped along with the ledger:\n%s", got)
	}
	if !strings.Contains(got, ResultMarker("itsaplan-EP-2-1")) {
		t.Fatalf("the result marker was dropped:\n%s", got)
	}
}

// 봉인의 정본은 우리 쪽에 남는다 (ADR-005 성질 4) — 코멘트는 사본이 아니라
// 요약이고, 어디서 꺼내는지만 적는다.
func TestRenderResult_PointsAtTheRecordInsteadOfPouringIt(t *testing.T) {
	m := renderMediator(t, notFound)
	run := &RunView{RunID: "itsaplan-EP-2-1", State: "SUCCEEDED"}

	got := RenderResult(context.Background(), m, run, "")

	if !strings.Contains(got, "GET /v1/runs/itsaplan-EP-2-1/record") {
		t.Fatalf("the comment does not say where the record is:\n%s", got)
	}
	// 표지가 있어야 복구 경로가 「이미 넘겼다」를 안다 (EP-2 실측).
	if !strings.Contains(got, "enode:result:itsaplan-EP-2-1") {
		t.Fatalf("the comment carries no result marker:\n%s", got)
	}
}

// 빈 절은 아예 안 낸다 — 표 머리만 있고 줄이 없는 표가 코멘트에 남으면
// 사람이 「무언가 잘못됐다」로 읽는다.
func TestRenderResult_OmitsTheEmptySections(t *testing.T) {
	m := renderMediator(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"entries":[]}`))
	})
	run := &RunView{RunID: "itsaplan-EP-2-1", State: "SUCCEEDED"}

	got := RenderResult(context.Background(), m, run, "   ")

	for _, unwanted := range []string{"| # | 단계 |", "**판정**", "남은 산출물"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("an empty %q section was rendered:\n%s", unwanted, got)
		}
	}
	// 공백만 있는 요약도 안 싣는다.
	if strings.Contains(got, "\n\n   \n") {
		t.Fatalf("a whitespace-only summary was rendered:\n%s", got)
	}
}

// 어댑터가 산출물의 필드 이름을 알면 계획의 스키마를 아는 것이 된다 —
// 그래서 이름이 아니라 값의 성질로 가른다.
func TestSummaryOf_SplitsProseFromTheRest(t *testing.T) {
	longProse := "첫 줄입니다\n" + strings.Repeat("본문이 이어집니다. ", 40)
	for _, tc := range []struct {
		name    string
		status  int
		blob    string
		want    []string
		notWant []string
	}{
		{name: "no summary artefact at all", status: 404, want: []string{""}},
		{name: "an empty artefact", status: 200, blob: "", want: []string{""}},
		{
			name: "plain prose is carried as it is", status: 200,
			blob: "  the plan went through as written  ",
			want: []string{"the plan went through as written"},
			// 글은 접지 않는다 — 사람이 읽으라고 낸 것이다.
			notWant: []string{"<details>"},
		},
		{
			name: "a JSON array is not an object, so it is carried as it is", status: 200,
			blob: `["a","b"]`,
			want: []string{`["a","b"]`},
		},
		{
			name: "a long multi-line string is unfolded as prose", status: 200,
			blob:    `{"narrative": ` + mustJSON(longProse) + `}`,
			want:    []string{"첫 줄입니다"},
			notWant: []string{"<details>"},
		},
		{
			name: "short values are folded away under report 가 낸 것", status: 200,
			blob: `{"exit_code": 0, "files": ["a.go"]}`,
			want: []string{"<summary><b>report 가 낸 것</b></summary>", `"exit_code": 0`},
		},
		{
			name: "with prose present the fold is labelled 산출물 원본", status: 200,
			blob: `{"narrative": ` + mustJSON(longProse) + `, "exit_code": 0}`,
			want: []string{"첫 줄입니다", "<summary><b>산출물 원본</b></summary>", `"exit_code": 0`},
		},
		{
			name: "a short single-line string is not prose", status: 200,
			blob:    `{"note": "done"}`,
			want:    []string{"<details>", `"note": "done"`},
			notWant: []string{"\ndone\n"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := renderMediator(t, func(w http.ResponseWriter, r *http.Request) {
				if !strings.HasSuffix(r.URL.Path, "/blob/summary") {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.blob))
			})
			got := summaryOf(context.Background(), m, "itsaplan-EP-2-1")
			for _, want := range tc.want {
				if want == "" {
					if got != "" {
						t.Fatalf("summary = %q, want empty", got)
					}
					continue
				}
				if !strings.Contains(got, want) {
					t.Fatalf("summary is missing %q:\n%s", want, got)
				}
			}
			for _, no := range tc.notWant {
				if strings.Contains(got, no) {
					t.Fatalf("summary should not contain %q:\n%s", no, got)
				}
			}
		})
	}
}

func mustJSON(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// 잘랐다는 사실을 적는다 — 안 적으면 사람이 잘린 글을 전문으로 읽는다.
func TestClip_SaysWhereItCut(t *testing.T) {
	if got := clip("short", 10); got != "short" {
		t.Fatalf("a short string was touched: %q", got)
	}
	if got := clip("exactly-10", 10); got != "exactly-10" {
		t.Fatalf("a string at the limit was cut: %q", got)
	}
	got := clip(strings.Repeat("x", 50), 10)
	if !strings.HasPrefix(got, strings.Repeat("x", 10)) {
		t.Fatalf("the kept prefix is wrong: %q", got)
	}
	if !strings.Contains(got, "잘렸습니다") {
		t.Fatalf("the cut is not announced: %q", got)
	}
	if !strings.Contains(got, "봉인된 Record") {
		t.Fatalf("the cut does not say where the whole thing is: %q", got)
	}
}

func TestPrettyJSON_IndentsWhatItCanAndKeepsTheRest(t *testing.T) {
	got := prettyJSON(json.RawMessage(`{"b":2,"a":1}`))
	if !strings.Contains(got, "\n  \"a\": 1") {
		t.Fatalf("valid JSON was not indented: %q", got)
	}
	// 못 읽으면 원문을 그대로 보인다 — 사람이 무엇이 왔는지는 봐야 한다.
	if got := prettyJSON(json.RawMessage("  not json at all  ")); got != "not json at all" {
		t.Fatalf("unreadable content = %q, want the trimmed raw text", got)
	}
	// 8000자를 넘으면 자른다 — 코멘트 하나가 이슈를 삼키면 안 된다.
	big, err := json.Marshal(strings.Repeat("y", 9000))
	if err != nil {
		t.Fatal(err)
	}
	got = prettyJSON(big)
	if len(got) > 8100 {
		t.Fatalf("an oversized artefact was not clipped (%d chars)", len(got))
	}
	if !strings.Contains(got, "잘렸습니다") {
		t.Fatalf("the clip is not announced: %q", got[len(got)-40:])
	}
}

// 형을 단정하지 않는다 — 어휘가 창발하므로 (ADR-012) 열거하면 새 조건이
// 생길 때마다 어댑터가 조용히 못 읽는다.
func TestDescribe_HandlesEveryShapeAConditionMayTake(t *testing.T) {
	for _, tc := range []struct {
		name string
		v    any
		want string
	}{
		{"nothing", nil, ""},
		{"a string", "plan", "`plan`"},
		{"an integral number", float64(7), "`7`"},
		{"a fractional number", 1.5, "`1.5`"},
		{"a boolean", true, "`true`"},
		{"an empty list", []any{}, ""},
		{"a list", []any{"a", "b"}, "`a`, `b`"},
		{"a list with a hole", []any{nil, "b"}, "`b`"},
		{"anything else falls back to JSON", map[string]any{"capability": "agent.reason"},
			"`{\"capability\":\"agent.reason\"}`"},
		{"a value JSON cannot express", make(chan int), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := describe(tc.v); got != tc.want {
				t.Fatalf("describe(%v) = %q, want %q", tc.v, got, tc.want)
			}
		})
	}
}

// 못 재면 대시다 — 0초로 적으면 「빨랐다」로 읽힌다.
func TestTook_DashesWhatItCannotMeasure(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start string
		end   string
		want  string
	}{
		{"a step that never started", "", "2026-08-24T00:00:01Z", "—"},
		{"a step that has not ended", "2026-08-24T00:00:00Z", "", "—"},
		{"an unreadable start", "yesterday", "2026-08-24T00:00:01Z", "—"},
		{"an unreadable end", "2026-08-24T00:00:00Z", "soon", "—"},
		{"clocks that disagree", "2026-08-24T00:00:05Z", "2026-08-24T00:00:00Z", "—"},
		{"a measurable step", "2026-08-24T00:00:00Z", "2026-08-24T00:00:01.234Z", "1.2s"},
		{"rounding up", "2026-08-24T00:00:00Z", "2026-08-24T00:00:02.96Z", "3s"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := took(tc.start, tc.end); got != tc.want {
				t.Fatalf("took(%q, %q) = %q, want %q", tc.start, tc.end, got, tc.want)
			}
		})
	}
}

// 무엇을 승인하는지 여기서 보세요 — 보여줄 것과 채택될 것이 둘 다 접힌 채
// 코멘트에 실린다. 안 실리면 사람이 이슈 밖으로 나가야 답할 수 있다.
func TestRenderQuestion_UnfoldsWhatIsBeingApproved(t *testing.T) {
	ask := &AskView{
		RunID: "itsaplan-EP-2-1", Seq: 4, Step: "gate", Prompt: "  이 계획으로 진행할까요  ",
		Shown: []struct {
			Name    string          `json:"name"`
			Content json.RawMessage `json:"content"`
		}{
			{Name: "plan", Content: json.RawMessage(`{"steps":["a"]}`)},
			{Name: "budget", Content: json.RawMessage(`{"minutes":30}`)},
		},
		Proposes: json.RawMessage(`{"verdict":"ok"}`),
	}

	got := RenderQuestion(ask, "itsaplan-EP-2-1")

	for _, want := range []string{
		"<summary><b>plan</b> — 무엇을 승인하는지 여기서 보세요</summary>",
		"<summary><b>budget</b> — 무엇을 승인하는지 여기서 보세요</summary>",
		"**이 답이 채택하면 판정 기준이 되는 것**",
		"\"verdict\": \"ok\"",
		"step `gate` (seq 4)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("the question comment is missing %q:\n%s", want, got)
		}
	}
	// 앞뒤 공백은 떼고 싣는다.
	if strings.Contains(got, "\n  이 계획으로") {
		t.Fatalf("the prompt kept its padding:\n%s", got)
	}
}

// proposes 가 없으면 그 절을 아예 안 낸다. null 이라는 글자가 코멘트에 뜨면
// 사람이 무엇을 승인하는지 되레 헷갈린다.
func TestRenderQuestion_OmitsAnAbsentProposal(t *testing.T) {
	for _, tc := range []struct {
		name     string
		proposes json.RawMessage
	}{
		{"no proposal at all", nil},
		{"an explicit null", json.RawMessage("null")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ask := &AskView{RunID: "r", Seq: 1, Step: "gate", Prompt: "p", Proposes: tc.proposes}
			got := RenderQuestion(ask, "r")
			if strings.Contains(got, "판정 기준이 되는 것") {
				t.Fatalf("an absent proposal grew a section:\n%s", got)
			}
			// 답하는 방법은 언제나 남는다 — 폼이 없으므로 말로 적는 수밖에 없다.
			if !strings.Contains(got, "**답하는 방법**") {
				t.Fatalf("the question does not say how to answer:\n%s", got)
			}
		})
	}
}
