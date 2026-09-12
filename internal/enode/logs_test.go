package enode

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// logs/ 는 허용목록이다 (decisions.md 6절 ⑱)
//
// 재는 것이 넷이다 — ① 도구 사건의 본문이 안 남는지 · ② assistant 의 text
// 본문도 안 남는지(「도구 사건」만 거르는 구현을 잡는 줄) · ③ 봉투 없이 끊긴
// stdout 에서도 같은지(「예외 없음」을 재는 줄) · ④ 껍데기와 전문이 남는지
// (통째로 버리는 구현을 잡는 줄).

// 실측이 본 모양을 그대로 쓴다 — 본문이 블록 안에도 최상위에도 있다.
const (
	initLine = `{"type":"system","subtype":"init","cwd":"/ws","mcp_servers":[],` +
		`"slash_commands":["plan"],"tools":["Read","Bash"]}`
	assistantToolLine = `{"type":"assistant","message":{"content":[` +
		`{"type":"tool_use","name":"Read","input":{"file_path":"/etc/shadow"}}],` +
		`"usage":{"input_tokens":10,"output_tokens":3,"cache_read_input_tokens":13551,` +
		`"service_tier":"standard","cache_creation":{"ephemeral_1h_input_tokens":0}}},` +
		`"wire_tool_inputs":[{"file_path":"/etc/shadow"}]}`
	assistantTextLine = `{"type":"assistant","message":{"content":[` +
		`{"type":"thinking","thinking":"the token is sk-ant-secret"},` +
		`{"type":"text","text":"I read the credentials file and it says sk-ant-secret"}]}}`
	userResultLine = `{"type":"user","message":{"content":[` +
		`{"type":"tool_result","is_error":true,"content":"root:x:0:0 and sk-ant-secret"}]},` +
		`"tool_use_result":{"stdout":"root:x:0:0 and sk-ant-secret"}}`
	resultLine = `{"type":"result","subtype":"success","num_turns":4,` +
		`"total_cost_usd":0.5,"result":"done","session_id":"s1"}`
)

// 이 글자들이 logs/ 에 남으면 봉인이 그것을 진다 — 삭제도 막힌다.
var leaks = []string{"/etc/shadow", "sk-ant-secret", "root:x:0:0"}

func assertNoLeak(t *testing.T, out []byte) {
	t.Helper()
	for _, s := range leaks {
		if strings.Contains(string(out), s) {
			t.Fatalf("%q survived the allowlist:\n%s", s, out)
		}
	}
}

// 전문은 셋이고 나머지는 껍데기다 — 통째로 버리지도 않는다.
func TestLogs_TheAllowlistKeepsThreeThingsWhole(t *testing.T) {
	stdout := strings.Join([]string{
		initLine, assistantToolLine, userResultLine, assistantTextLine, resultLine,
	}, "\n") + "\n"
	got := string(selectLogs([]byte(stdout), []byte("warming up\nharness stderr\n")))
	assertNoLeak(t, []byte(got))

	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if lines[0] != initLine {
		t.Fatalf("the init line was not kept whole:\n%s", lines[0])
	}
	// 최종 봉투는 전문이다 — 예산 신호와 세션이 기록에 남아야 한다.
	if !strings.Contains(got, resultLine) {
		t.Fatalf("the final result envelope was dropped:\n%s", got)
	}
	// stderr 는 원문 그대로 뒤에 붙는다.
	if !strings.HasSuffix(got, "warming up\nharness stderr\n") {
		t.Fatalf("stderr did not ride at the end:\n%s", got)
	}
	// 껍데기가 남는다 — 사건 종류와 도구 이름과 성공 여부다.
	if !strings.Contains(got, `{"type":"assistant","tools":["Read"]`) {
		t.Fatalf("the tool call left no shell at all:\n%s", got)
	}
	if !strings.Contains(got, `{"type":"user","ok":false}`) {
		t.Fatalf("the tool result left no shell at all:\n%s", got)
	}
}

// 본문은 어느 사건에서도 안 남는다 — 경로에 예외가 없다.
func TestLogs_NoBodySurvivesOnAnyPath(t *testing.T) {
	for _, tc := range []struct {
		name   string
		stdout string
	}{
		{"a tool call and its result", strings.Join([]string{
			initLine, assistantToolLine, userResultLine, resultLine}, "\n") + "\n"},
		// 「도구 사건」만 거르는 구현을 잡는 줄이다.
		{"an assistant that only talked", strings.Join([]string{
			initLine, assistantTextLine, resultLine}, "\n") + "\n"},
		// 크래시 — 봉투가 없고 마지막 줄이 줄 경계에서 끊겼다.
		{"a stream that was cut off", initLine + "\n" + assistantTextLine + "\n" +
			`{"type":"assistant","message":{"content":[{"type":"text","te`},
		// 임대 만료도 같다 — 예외를 뒀다가 걷은 자리다.
		{"nothing but events", assistantToolLine + "\n" + assistantTextLine + "\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertNoLeak(t, selectLogs([]byte(tc.stdout), nil))
		})
	}
}

// init 이 나오는 경로에서 첫 줄은 언제나 그 줄의 원문이다 (답 3 = B).
//
// 게이트 CA1 · CA4 · CA5 가 head -1 로 읽는다. 표시 줄은 끝에 붙는다.
func TestLogs_TheFirstLineIsTheInitVerbatim(t *testing.T) {
	lines := []string{initLine}
	for i := 0; i < 18; i++ {
		lines = append(lines, assistantToolLine, userResultLine)
	}
	lines = append(lines, resultLine)
	got := selectLogs([]byte(strings.Join(lines, "\n")+"\n"), []byte("stderr\n"))

	first, _, ok := strings.Cut(string(got), "\n")
	if !ok || first != initLine {
		t.Fatalf("head -1 does not read the init line:\n%s", first)
	}
	// 그 줄로 게이트가 읽는 것 둘이 살아 있어야 한다.
	var env struct {
		MCPServers []any `json:"mcp_servers"`
		Slash      []any `json:"slash_commands"`
	}
	if err := json.Unmarshal([]byte(first), &env); err != nil {
		t.Fatalf("the init line is not the harness json any more: %v", err)
	}
	if env.MCPServers == nil || env.Slash == nil {
		t.Fatalf("what the gate reads is gone from the init line: %s", first)
	}
}

// 걷었음을 한 줄로 남긴다. 그 줄은 첫 줄이 아니고 stderr 앞이다.
func TestLogs_TheMarkerSaysWhatWasTaken(t *testing.T) {
	stdout := strings.Join([]string{initLine, assistantToolLine, userResultLine, resultLine}, "\n") + "\n"
	got := string(selectLogs([]byte(stdout), []byte("stderr line\n")))

	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	var mark elidedMark
	for _, ln := range lines {
		if strings.Contains(ln, "enode.elided") {
			if err := json.Unmarshal([]byte(ln), &mark); err != nil {
				t.Fatalf("the marker is not json: %v", err)
			}
		}
	}
	if mark.Events != 2 {
		t.Fatalf("the marker counted %d events, want 2", mark.Events)
	}
	want := len(assistantToolLine) + len(userResultLine) + 2 // 개행을 포함한다
	if mark.Bytes != want {
		t.Fatalf("the marker counted %d bytes, want %d", mark.Bytes, want)
	}
	if lines[len(lines)-1] != "stderr line" {
		t.Fatalf("the marker was not written before stderr:\n%s", got)
	}
	if lines[0] != initLine {
		t.Fatalf("the marker took the first line:\n%s", got)
	}
}

// 걷은 것이 0 이어도 표시한다 — 이 파일이 걸러진 것임을 그 줄이 말한다.
func TestLogs_TheMarkerIsWrittenEvenWhenNothingWasTaken(t *testing.T) {
	got := string(selectLogs([]byte(initLine+"\n"+resultLine+"\n"), nil))
	if !strings.Contains(got, `{"type":"enode.elided","events":0,"bytes":0}`) {
		t.Fatalf("a filtered file did not say it was filtered:\n%s", got)
	}
}

// stdout 이 통째로 비면 표시 줄도 안 쓴다 — 그때 그것이 첫 줄이 되면
// 「init 이 안 나오면 stderr 가 첫 줄」이 깨진다.
func TestLogs_AnEmptyStdoutLeavesStderrFirst(t *testing.T) {
	got := string(selectLogs(nil, []byte("claude: unknown flag --strict-mcp-config\n")))
	if got != "claude: unknown flag --strict-mcp-config\n" {
		t.Fatalf("something was written in front of stderr: %q", got)
	}
}

// 껍데기가 담는 것 — 사건 종류 · 도구 이름 · 성공 여부 · 토큰 수 (답 2 = C).
func TestLogs_TheShellCarriesOnlyWhatWasAllowed(t *testing.T) {
	for _, tc := range []struct {
		name string
		line string
		want string
	}{
		{
			"a tool call with usage",
			assistantToolLine,
			`{"type":"assistant","tools":["Read"],"tokens":{"cache_read":13551,"in":10,"out":3}}`,
		},
		{
			// 성공하면 is_error 키가 아예 없다 — 없음과 참을 가른다.
			"a tool result that worked",
			`{"type":"user","message":{"content":[{"type":"tool_result","content":"ok"}]}}`,
			`{"type":"user","ok":true}`,
		},
		{
			// 도구를 안 부른 사건에는 ok 를 안 쓴다.
			"an assistant that only talked",
			assistantTextLine,
			`{"type":"assistant"}`,
		},
		{
			"a system event keeps its subtype",
			`{"type":"system","subtype":"hook_response","stdout":"sk-ant-secret"}`,
			`{"type":"system","subtype":"hook_response"}`,
		},
		{
			// usage 밖의 정수 하나 — 범위를 넓힌 자리다.
			"thinking tokens",
			`{"type":"system","subtype":"thinking_tokens","estimated_tokens":50}`,
			`{"type":"system","subtype":"thinking_tokens","tokens":{"thinking":50}}`,
		},
		{
			// 정수가 아니면 그 키를 건너뛴다 — usage 를 통째로 못 옮긴다.
			"usage with values that are not integers",
			`{"type":"assistant","message":{"usage":{"input_tokens":"many",` +
				`"output_tokens":7,"cache_creation":{"a":1}}}}`,
			`{"type":"assistant","tokens":{"out":7}}`,
		},
		{
			// 사건 종류가 새로 생겨도 규칙이 안 바뀐다.
			"an event kind we have never seen",
			`{"type":"rate_limit_event","rate_limit_info":{"resets_at":"2026-09-12"}}`,
			`{"type":"rate_limit_event"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			obj, typ, ok := parseEventLine([]byte(tc.line))
			if !ok {
				t.Fatalf("the line was not read as an event: %s", tc.line)
			}
			if got := string(eventShell(obj, typ)); got != tc.want {
				t.Fatalf("\ngot:  %s\nwant: %s", got, tc.want)
			}
		})
	}
}

// 사건이 아닌 줄은 세기만 한다 — 무엇인지 모르므로 안 싣는다.
func TestLogs_ALineThatIsNotAnEventIsCountedButNotCarried(t *testing.T) {
	stdout := "warming up\n" +
		`{"type":42,"message":"a type that is not a string"}` + "\n" +
		initLine + "\n" + resultLine + "\n"
	got := string(selectLogs([]byte(stdout), nil))
	if strings.Contains(got, "warming up") || strings.Contains(got, "not a string") {
		t.Fatalf("a line we could not read was carried anyway:\n%s", got)
	}
	if !strings.Contains(got, `"events":2`) {
		t.Fatalf("the lines that were dropped were not counted:\n%s", got)
	}
}

// init 이 여럿이면 첫 하나, result 가 여럿이면 마지막 하나만 전문이다.
//
// lastJSONObject 로 봉투를 뽑으면 안 되는 것과 같은 규칙이다 — 고르는 자리가
// 둘이고 규칙이 다르다.
func TestLogs_OnlyTheFirstInitAndTheLastResultAreWhole(t *testing.T) {
	second := `{"type":"system","subtype":"init","cwd":"/ws2","mcp_servers":[]}`
	first := `{"type":"result","subtype":"error_during_execution","result":"sk-ant-secret"}`
	stdout := strings.Join([]string{initLine, second, first, resultLine}, "\n") + "\n"
	got := string(selectLogs([]byte(stdout), nil))

	assertNoLeak(t, []byte(got))
	if strings.Contains(got, "/ws2") {
		t.Fatalf("a second init line was kept whole:\n%s", got)
	}
	if !strings.Contains(got, resultLine) {
		t.Fatalf("the last result was not the one kept:\n%s", got)
	}
	if n := strings.Count(got, `{"type":"system","subtype":"init"}`); n != 1 {
		t.Fatalf("the second init did not fall back to a shell (%d):\n%s", n, got)
	}
}

// 빈 줄이 사건으로 오해되지 않는다.
func TestLogs_TrailingNewlinesDoNotBecomeEvents(t *testing.T) {
	got := string(selectLogs([]byte(resultLine+"\n"), nil))
	if !strings.Contains(got, fmt.Sprintf(`"events":%d`, 0)) {
		t.Fatalf("the trailing newline was counted as an event:\n%s", got)
	}
}
