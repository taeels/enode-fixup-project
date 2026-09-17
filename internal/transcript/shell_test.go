package transcript

import (
	"encoding/json"
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

// ADR-071 이 본문을 되살린 뒤로도 어느 경로로도 안 나가는 글자들이다.
//
// 목록이 줄었다 — /etc/shadow 와 sk-ant-secret 은 이제 남는다. 앞의 것은
// 도구 인자(에이전트가 무엇을 읽었나 · ADR-017 §7 ①)이고 뒤의 것은
// 에이전트가 스스로 말한 것이라, 둘 다 ADR-071 이 남기기로 한 자리다.
// 남은 하나는 생각의 서명이다 — 읽을 것이 0 인데 블록 중 가장 크다.
//
// 퍼즈가 이것을 못 닫는다. 무엇이 본문인지 아는 것은 사람이고, 퍼즈는
// 「패닉하지 않는다」까지만 안다.
var leaks = []string{"signature", "wire_tool_inputs", "tool_use_result"}

func assertNoLeak(t *testing.T, out []byte) {
	t.Helper()
	for _, s := range leaks {
		if strings.Contains(string(out), s) {
			t.Fatalf("%q survived the allowlist:\n%s", s, out)
		}
	}
}

// 줄여 쓴 원문이 담는 것 — 껍데기 다섯에 본문이 더해진다 (ADR-071).
//
// 기대 문자열이 이 ADR 에서 바뀌었다. 바꿔도 되는 이유는 더하는 쪽이기
// 때문이다 — 필드 다섯과 태그는 그대로고 message 가 붙는다. 옛 기록에는
// 그 키가 없으므로 파서가 껍데기 줄기로 가고 (parse.go 의 messageEvents),
// 0444 로 굳은 logs/ 가 계속 읽힌다.
func TestShell_CarriesOnlyWhatWasAllowed(t *testing.T) {
	for _, tc := range []struct {
		name string
		line string
		want string
	}{
		{
			// 최상위 wire_tool_inputs 는 안 따라온다 — 같은 것의 두 벌이다.
			"a tool call with usage",
			line("assistant-tool.json"),
			`{"type":"assistant","tools":["Read"],"tokens":{"cache_read":13551,"in":10,"out":3},` +
				`"message":{"content":[{"type":"tool_use","name":"Read","input":{"file_path":"/etc/shadow"}}]}}`,
		},
		{
			// 성공하면 is_error 키가 아예 없다 — 없음과 참을 가른다.
			"a tool result that worked",
			`{"type":"user","message":{"content":[{"type":"tool_result","content":"ok"}]}}`,
			`{"type":"user","ok":true,"message":{"content":[{"type":"tool_result","content":"ok"}]}}`,
		},
		{
			// 생각은 안 싣는다. 말은 싣는다 — 같은 줄에서 갈린다.
			"an assistant that only talked",
			line("assistant-text.json"),
			`{"type":"assistant","message":{"content":[{"type":"text",` +
				`"text":"I read the credentials file and it says sk-ant-secret"}]}}`,
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

// 생각의 서명과 최상위 두 벌은 어느 사건에서도 안 실린다.
//
// 위의 표가 바이트를 통째로 맞대는 데 비해 이쪽은 고정 문자열 셋만 본다.
// 표는 모양이 바뀌면 빨개지고 이쪽은 안 나가야 할 것이 새면 빨개진다 —
// 다른 것을 잰다.
func TestShell_WhatMustNotSurviveDoesNot(t *testing.T) {
	for _, name := range []string{
		"assistant-tool.json",
		"assistant-text.json",
		"user-result.json",
		"hook-response.json",
		"user-result-id.json",
		"assistant-tool-id.json",
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

// 생각 블록은 통째로 안 남는다 — 본문이 있어도 그렇다 (ADR-071 2절).
//
// 위의 서명 검사와 다른 것을 잰다. 저쪽은 서명이라는 글자가 새는지를 보고
// 이쪽은 블록 자체가 사라지는지를 본다. 하네스가 서명 키 이름을 바꾸는 날
// 저쪽만으로는 안 걸린다.
func TestShell_ThinkingIsNotCarried(t *testing.T) {
	const ln = `{"type":"assistant","message":{"content":[` +
		`{"type":"thinking","thinking":"a thought","signature":"AAAA"},` +
		`{"type":"text","text":"a word"}]}}`
	obj, typ, ok := ParseLine([]byte(ln))
	if !ok {
		t.Fatalf("the line was not read as an event")
	}
	got := string(Shell(obj, typ))
	if strings.Contains(got, "thinking") || strings.Contains(got, "a thought") {
		t.Fatalf("the thinking block was carried: %s", got)
	}
	if !strings.Contains(got, "a word") {
		t.Fatalf("the text block went with it: %s", got)
	}
}

// 남는 본문은 유계다 — 도구 결과도 도구 인자도 상한 안이다 (ADR-071 2절).
//
// 상한이 없으면 Write 한 번이 봉인을 채운다. 인자는 자를 때 JSON 문자열이
// 되므로 「유효한 JSON 으로 남는가」를 함께 잰다 — 깨지면 파서가 그 줄을
// 통째로 raw 로 떨어뜨려 도구 이름까지 잃는다.
func TestShell_TheBodyThatSurvivesIsBounded(t *testing.T) {
	big := strings.Repeat("a", 4000)
	for _, tc := range []struct {
		name string
		line string
	}{
		{
			"a tool result longer than the cap",
			`{"type":"user","message":{"content":[{"type":"tool_result","content":"` + big + `"}]}}`,
		},
		{
			"a tool input longer than the cap",
			`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Write",` +
				`"input":{"file_path":"/tmp/x","content":"` + big + `"}}]}}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			obj, typ, ok := ParseLine([]byte(tc.line))
			if !ok {
				t.Fatalf("the line was not read as an event")
			}
			out := Shell(obj, typ)
			// 줄 전체가 아니라 블록의 본문 필드를 잰다 — 전체를 세면 키
			// 이름의 글자가 섞여 상한이 한두 바이트씩 틀리게 읽힌다.
			var got struct {
				Message struct {
					Content []map[string]json.RawMessage `json:"content"`
				} `json:"message"`
			}
			if err := json.Unmarshal(out, &got); err != nil {
				t.Fatalf("the reduced line is not valid JSON: %v\n%s", err, out)
			}
			if len(got.Message.Content) != 1 {
				t.Fatalf("got %d blocks, want 1: %s", len(got.Message.Content), out)
			}
			for _, key := range []string{"content", "input"} {
				v, has := got.Message.Content[0][key]
				if !has {
					continue
				}
				// 적힌 바이트가 아니라 담긴 바이트를 잰다 — 문자열로
				// 자를 때 따옴표가 이스케이프되어 적힌 쪽이 더 길다.
				n := len(v)
				var s string
				if json.Unmarshal(v, &s) == nil {
					n = len(s)
				}
				if n > MaxSealedToolResult {
					t.Fatalf("%s kept %d bytes, the cap is %d", key, n, MaxSealedToolResult)
				}
				if !strings.Contains(string(v), "aaaa") {
					t.Fatalf("%s lost the body entirely: %s", key, v)
				}
			}
			// 파서가 그 줄을 다시 읽는다 — 사건 하나이고 raw 가 아니다.
			r := Parse(append(out, '\n'), false)
			if r.Raw != 0 {
				t.Fatalf("the reduced line came back as raw: %s", out)
			}
			if len(r.Events) != 1 {
				t.Fatalf("got %d events, want 1: %s", len(r.Events), out)
			}
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
