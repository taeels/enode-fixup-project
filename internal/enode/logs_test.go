package enode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/taeels/enode/internal/transcript"
)

// line 은 픽스처 줄 하나를 읽고 꼬리 개행을 뗀다.
//
// 파일이 한 벌이다 — internal/transcript 의 같은 도우미가 같은 디렉터리를
// 읽는다. 두 벌로 두면 한쪽만 고쳐도 둘 다 초록이라 갈린 것을 아무도 못 본다.
//
// 상대 경로의 대가를 이름으로 적는다 — 파일 이름이 바뀌면 컴파일이 아니라
// 실행 때 깨진다. go list 가 못 보는 결합이라 실패 메시지가 경로를 싣는다.
func line(name string) string {
	path := filepath.Join("..", "transcript", "testdata", "lines", name)
	b, err := os.ReadFile(path)
	if err != nil {
		panic("fixture is missing: " + path + ": " + err.Error())
	}
	return strings.TrimRight(string(b), "\n")
}

// logs/ 는 허용목록이다 (decisions.md 6절 ⑱)
//
// 재는 것이 넷이다 — ① 도구 사건의 본문이 안 남는지 · ② assistant 의 text
// 본문도 안 남는지(「도구 사건」만 거르는 구현을 잡는 줄) · ③ 봉투 없이 끊긴
// stdout 에서도 같은지(「예외 없음」을 재는 줄) · ④ 껍데기와 전문이 남는지
// (통째로 버리는 구현을 잡는 줄).

// 실측이 본 모양을 그대로 쓴다 — 본문이 블록 안에도 최상위에도 있다.
//
// 상수가 아니라 파일이다. internal/transcript 의 시험이 같은 줄을 읽으므로
// 글자가 두 자리에 있으면 갈린다 — 갈려도 양쪽이 초록이라 아무도 못 본다.
var (
	initLine          = line("init.json")
	assistantToolLine = line("assistant-tool.json")
	assistantTextLine = line("assistant-text.json")
	userResultLine    = line("user-result.json")
	resultLine        = line("result.json")
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
	var mark transcript.Elided
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
