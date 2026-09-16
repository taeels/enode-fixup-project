package transcript

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// line 은 픽스처 줄 하나를 읽고 꼬리 개행을 뗀다.
//
// 파일이 한 벌이다 — internal/enode 의 같은 도우미가 같은 디렉터리를 읽는다.
// 글자를 두 자리에 두면 갈리고, 갈려도 양쪽이 초록이라 아무도 못 본다.
func line(name string) string {
	path := filepath.Join("testdata", "lines", name)
	b, err := os.ReadFile(path)
	if err != nil {
		panic("fixture is missing: " + path + ": " + err.Error())
	}
	return strings.TrimRight(string(b), "\n")
}

// 이 글자들이 껍데기에 남으면 봉인이 그것을 진다 — 삭제도 막힌다.
//
// 퍼즈가 이것을 못 닫는다. 무엇이 본문인지 아는 것은 사람이고, 퍼즈는
// 「패닉하지 않는다」까지만 안다.
var leaks = []string{"/etc/shadow", "sk-ant-secret", "root:x:0:0"}

func assertNoLeak(t *testing.T, out []byte) {
	t.Helper()
	for _, s := range leaks {
		if strings.Contains(string(out), s) {
			t.Fatalf("%q survived the allowlist:\n%s", s, out)
		}
	}
}

// 껍데기가 담는 것 — 사건 종류 · 도구 이름 · 성공 여부 · 토큰 수.
//
// internal/enode/logs_test.go 에서 그대로 옮겨 왔다. 기대 문자열을 한 글자도
// 안 바꿨다 — 이미 봉인된 Run 의 logs/ 가 이 바이트로 굳어 있고 0444 라
// 고칠 수 없다. 옮긴 이유는 이 시험이 Shell 과 ParseLine 을 직접 부르는데
// 그 둘이 이제 이 패키지에 살기 때문이다.
func TestShell_CarriesOnlyWhatWasAllowed(t *testing.T) {
	for _, tc := range []struct {
		name string
		line string
		want string
	}{
		{
			"a tool call with usage",
			line("assistant-tool.json"),
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
			line("assistant-text.json"),
			`{"type":"assistant"}`,
		},
		{
			"a system event keeps its subtype",
			line("hook-response.json"),
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
			line("rate-limit.json"),
			`{"type":"rate_limit_event"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			obj, typ, ok := ParseLine([]byte(tc.line))
			if !ok {
				t.Fatalf("the line was not read as an event: %s", tc.line)
			}
			if got := string(Shell(obj, typ)); got != tc.want {
				t.Fatalf("\ngot:  %s\nwant: %s", got, tc.want)
			}
		})
	}
}

// 본문은 어느 사건에서도 껍데기에 안 실린다.
//
// 위의 표가 바이트를 통째로 맞대는 데 비해 이쪽은 고정 문자열 셋만 본다.
// 표는 모양이 바뀌면 빨개지고 이쪽은 본문이 새면 빨개진다 — 다른 것을 잰다.
func TestShell_NoBodySurvivesOnAnyPath(t *testing.T) {
	for _, name := range []string{
		"assistant-tool.json",
		"assistant-text.json",
		"user-result.json",
		"hook-response.json",
		"user-result-id.json",
	} {
		t.Run(name, func(t *testing.T) {
			obj, typ, ok := ParseLine([]byte(line(name)))
			if !ok {
				t.Fatalf("the fixture was not read as an event: %s", name)
			}
			assertNoLeak(t, Shell(obj, typ))
		})
	}
}

// ElidedMarker 의 바이트도 굳어 있다 — 걷은 것이 0 일 때의 모양까지 그렇다.
func TestElidedMarker_WritesTheSameBytesItAlwaysDid(t *testing.T) {
	if got := string(ElidedMarker(0, 0)); got != `{"type":"enode.elided","events":0,"bytes":0}` {
		t.Fatalf("the marker bytes changed: %s", got)
	}
	if got := string(ElidedMarker(2, 345)); got != `{"type":"enode.elided","events":2,"bytes":345}` {
		t.Fatalf("the marker bytes changed: %s", got)
	}
}
