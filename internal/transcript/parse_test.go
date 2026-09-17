package transcript

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

// join 은 줄들을 개행으로 잇고 꼬리 개행을 붙인다. 온전한 입력의 모양이다.
func join(lines ...string) []byte {
	return []byte(strings.Join(lines, "\n") + "\n")
}

// toolResultLine 은 본문 하나를 든 tool_result 줄을 짓는다.
//
// 손으로 JSON 을 쓰지 않는다 — 600 바이트짜리 본문을 문자열 리터럴로 적으면
// 이스케이프가 섞여 재려던 길이가 아니게 된다.
func toolResultLine(t *testing.T, content string) string {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"type": "user",
		"message": map[string]any{
			"content": []any{
				map[string]any{"type": "tool_result", "content": content},
			},
		},
	})
	if err != nil {
		t.Fatalf("could not build the fixture line: %v", err)
	}
	return string(b)
}

// textBlockLine 은 본문 하나를 든 assistant/text 줄을 짓는다.
func textBlockLine(t *testing.T, text string) string {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"content": []any{map[string]any{"type": "text", "text": text}},
		},
	})
	if err != nil {
		t.Fatalf("could not build the fixture line: %v", err)
	}
	return string(b)
}

// linesThatMadeEvents 는 사건을 낸 줄 번호의 가짓수다.
func linesThatMadeEvents(r Result) int {
	seen := map[int]bool{}
	for _, e := range r.Events {
		seen[e.Line] = true
	}
	return len(seen)
}

// R1 — 줄을 안 버린다. 읽은 줄은 사건을 내거나 집계다. 셋째 길이 없다.
func TestParse_EveryLineIsEitherAnEventOrTheAggregate(t *testing.T) {
	lines := []string{
		line("init.json"),
		line("assistant-tool.json"),
		line("user-result.json"),
		line("elided.json"),
		line("plain.txt"),
		// 블록이 0 인 줄도 여기 있다 — 그 되돌림이 빠지면 이 줄이 통째로
		// 사라지고 아래의 셈이 안 맞는다.
		line("assistant-no-blocks.json"),
		line("result.json"),
	}
	r := Parse(join(lines...), false)

	if r.Lines != len(lines) {
		t.Fatalf("read %d lines, want %d", r.Lines, len(lines))
	}
	aggregate := 0
	if r.Elided != nil {
		aggregate = 1
	}
	if got := linesThatMadeEvents(r) + aggregate; got != r.Lines {
		t.Fatalf("%d lines made events or the aggregate, but %d were read", got, r.Lines)
	}

	// 줄 하나를 지우면 Lines 가 정확히 하나 준다. 같은 목록에서 덜어내므로
	// 위의 목록을 고쳐도 이 셈이 안 갈린다.
	shorter := Parse(join(lines[:len(lines)-1]...), false)
	if shorter.Lines != r.Lines-1 {
		t.Fatalf("dropping one line moved Lines from %d to %d", r.Lines, shorter.Lines)
	}
}

// R2 — 아는 키를 아는 모양으로만 읽는다. usage 에 문자열과 객체가 섞여 있다.
func TestParse_TokensTakeOnlyTheIntegers(t *testing.T) {
	r := Parse(join(line("assistant-tool.json")), false)
	if len(r.Events) != 1 {
		t.Fatalf("got %d events, want 1", len(r.Events))
	}
	want := map[string]int{"in": 10, "out": 3, "cache_read": 13551}
	got := r.Events[0].Tokens
	if len(got) != len(want) {
		t.Fatalf("got tokens %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("token %q is %d, want %d", k, got[k], v)
		}
	}
	// service_tier 와 cache_creation 은 정수가 아니라 안 실린다.
	for _, k := range []string{"service_tier", "inference_geo", "cache_creation"} {
		if _, has := got[k]; has {
			t.Fatalf("a value that is not an integer was carried: %q", k)
		}
	}
}

// R3 — 사상표에 없는 줄도 버리지 않는다. 어느 자리로 떨어지는지가 규칙이다.
func TestParse_TheLadderPutsEveryLineSomewhere(t *testing.T) {
	for _, tc := range []struct {
		name string
		line string
		kind Kind
		sub  string
	}{
		{"a line that is not json", line("plain.txt"), KindText, "plain"},
		{"a type that is not a string", line("type-not-string.json"), KindText, "plain"},
		{"a harness type we do not map", line("rate-limit.json"), KindRaw, "rate_limit_event"},
		{"a system line that is not init", line("hook-response.json"), KindRaw, "system/hook_response"},
		{"our own capped marker", line("capped.json"), KindCapped, ""},
		{"a marker of ours we do not know", line("unknown-enode.json"), KindRaw, "enode.wibble"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := Parse(join(tc.line), false)
			if len(r.Events) != 1 {
				t.Fatalf("got %d events, want 1", len(r.Events))
			}
			e := r.Events[0]
			if e.Kind != tc.kind || e.Sub != tc.sub {
				t.Fatalf("got kind %q sub %q, want %q %q", e.Kind, e.Sub, tc.kind, tc.sub)
			}
			// 못 읽은 줄은 원문을 그대로 든다 — US-4 가 그것을 읽는다.
			if tc.kind == KindRaw || tc.sub == "plain" {
				if e.Text != tc.line {
					t.Fatalf("the line was not carried verbatim:\ngot:  %s\nwant: %s", e.Text, tc.line)
				}
			}
		})
	}
}

// R3.9 — capped 의 Bytes 는 줄이 말한 그대로다. 총 길이로 고쳐 읽지 않는다.
func TestParse_CappedCarriesTheLimitItWasGiven(t *testing.T) {
	r := Parse(join(line("init.json"), line("capped.json")), false)
	var capped *Event
	for i := range r.Events {
		if r.Events[i].Kind == KindCapped {
			capped = &r.Events[i]
		}
	}
	if capped == nil {
		t.Fatalf("the capped marker did not become an event: %+v", r.Events)
	}
	if capped.Info.Bytes != 10485760 {
		t.Fatalf("Bytes is %d, want the 10 MiB the line carried", capped.Info.Bytes)
	}
	// 위치가 값이다 — 집계로 빼면 사건 열 어디서 끊겼는지를 잃는다.
	if capped.Line != 2 {
		t.Fatalf("the capped event landed on line %d, want 2", capped.Line)
	}
}

// R4 — 껍데기와 원문을 Shell 하나로 가른다.
func TestParse_ShellTellsTheTwoStemsApart(t *testing.T) {
	whole := line("assistant-tool.json")
	// 껍데기 줄기는 이제 옛 기록의 것이다 (ADR-071). Shell 이 짓는 줄에는
	// message 가 있으므로 원문 줄기로 간다 — 그래서 짓는 쪽을 부르지 않고
	// 이미 봉인된 모양의 픽스처를 쓴다. 이 시험이 지키는 것이 그것이다:
	// 0444 로 굳은 옛 logs/ 가 오늘 코드로도 읽힌다.
	shell := line("shell-assistant.json")

	from := func(ln string) Event {
		r := Parse(join(ln), false)
		if len(r.Events) != 1 {
			t.Fatalf("got %d events from %s, want 1", len(r.Events), ln)
		}
		return r.Events[0]
	}
	if e := from(whole); e.Shell {
		t.Fatalf("the verbatim line was read as a shell: %+v", e)
	}
	if e := from(shell); !e.Shell {
		t.Fatalf("the shell line was read as verbatim: %+v", e)
	}
	// 껍데기에도 도구 이름은 선다 — US-10 이 그것을 읽는다.
	if e := from(shell); e.Name != "Read" {
		t.Fatalf("the shell lost the tool name: %+v", e)
	}
}

// R4.3 — 껍데기 판정은 assistant 와 user 에만 건다.
//
// 넓히면 result 줄의 message 없음이 껍데기로 읽혀 Shell 이 참이 된다.
func TestParse_OnlyAssistantAndUserCanBeShells(t *testing.T) {
	r := Parse(join(line("result.json"), line("rate-limit.json")), false)
	for _, e := range r.Events {
		if e.Shell {
			t.Fatalf("a line that is not assistant or user was read as a shell: %+v", e)
		}
	}
}

// R5 — 블록이 0 인 줄도 사건 하나다.
//
// 사건 0 이 되면 「아무 말도 안 했다」와 「걷혔다」가 화면에서 같아 보인다.
// 줄기가 둘이라 되돌림도 둘이다. 한쪽만 재면 다른 쪽을 지워도 초록이다 —
// 변이 ⑤ 가 그것을 실제로 보였다.
func TestParse_ALineWithNoBlocksStillMakesOneEvent(t *testing.T) {
	for _, tc := range []struct {
		name  string
		line  string
		shell bool
	}{
		// 껍데기 줄기 — message 키가 아예 없다.
		{"a shell with nothing in it", `{"type":"assistant"}`, true},
		// 원문 줄기 — message 는 있고 블록이 0 이다.
		{"a message with an empty block list", `{"type":"assistant","message":{"content":[]}}`, false},
		// 원문 줄기 — 블록은 있으나 아는 종류가 0 이다.
		{"a message with only blocks we do not know", line("assistant-no-blocks.json"), false},
		// 원문 줄기 — content 의 모양이 아예 다르다.
		{"a message whose content is not a list", `{"type":"assistant","message":{"content":"hi"}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := Parse(join(tc.line), false)
			if len(r.Events) != 1 {
				t.Fatalf("got %d events, want 1 — the line was dropped", len(r.Events))
			}
			e := r.Events[0]
			if e.Kind != KindText || e.Text != "" {
				t.Fatalf("got %+v, want an empty text event", e)
			}
			if e.Shell != tc.shell {
				t.Fatalf("got Shell=%v, want %v — the wrong stem made this event", e.Shell, tc.shell)
			}
		})
	}
}

// R6 — usage 는 줄 하나의 예산이다. 사건마다 복사하면 합계가 부푼다.
func TestParse_TokensRideOnTheFirstEventOfTheLineOnly(t *testing.T) {
	ln := `{"type":"assistant","message":{"content":[` +
		`{"type":"tool_use","id":"a","name":"Read","input":{}},` +
		`{"type":"tool_use","id":"b","name":"Bash","input":{}}],` +
		`"usage":{"input_tokens":7,"output_tokens":2}}}`
	r := Parse(join(ln), false)
	if len(r.Events) != 2 {
		t.Fatalf("got %d events, want 2", len(r.Events))
	}
	if r.Events[0].Tokens["in"] != 7 {
		t.Fatalf("the first event did not carry the budget: %+v", r.Events[0].Tokens)
	}
	if r.Events[1].Tokens != nil {
		t.Fatalf("the budget was copied onto the second event: %+v", r.Events[1].Tokens)
	}
}

// R7 — 개행으로 안 끝나면 그 꼬리는 「버렸다」가 아니라 「안 읽었다」다.
func TestParse_AnUnterminatedTailIsNotReadYet(t *testing.T) {
	// 사건 하나를 내는 줄을 쓴다 — assistant 는 블록마다 사건이라 차이가
	// 하나가 아니고, 그러면 이 시험이 재려던 것이 흐려진다.
	whole := line("result.json")
	complete := Parse([]byte(line("init.json")+"\n"+whole+"\n"), false)
	cut := Parse([]byte(line("init.json")+"\n"+whole), false)

	if len(cut.Events) != len(complete.Events)-1 {
		t.Fatalf("the unterminated line made %d events, the terminated one %d",
			len(cut.Events), len(complete.Events))
	}
	if cut.Partial != len(whole) {
		t.Fatalf("Partial is %d, want the %d bytes of the tail", cut.Partial, len(whole))
	}
	if complete.Partial != 0 {
		t.Fatalf("a terminated input reported a partial tail: %d", complete.Partial)
	}
}

// R7.1 — truncated 면 첫 개행까지를 버리고 그 수를 Head 에 적는다.
func TestParse_ATruncatedHeadIsDroppedAndCounted(t *testing.T) {
	head := `ssage":{"content":[]}}`
	in := []byte(head + "\n" + line("result.json") + "\n")
	r := Parse(in, true)

	if r.Head != len(head)+1 {
		t.Fatalf("Head is %d, want %d", r.Head, len(head)+1)
	}
	if r.Lines != 1 {
		t.Fatalf("read %d lines, want 1", r.Lines)
	}
	// 개행이 하나도 없으면 읽을 것이 0 이다.
	if empty := Parse([]byte("half a line with no newline"), true); empty.Lines != 0 ||
		len(empty.Events) != 0 || empty.Head != 0 {
		t.Fatalf("a headless truncated input was not empty: %+v", empty)
	}
}

// R8 ① — 도구 결과는 500 바이트에서 잘린다.
func TestParse_AToolResultIsCutAtFiveHundredBytes(t *testing.T) {
	body := strings.Repeat("a", 600)
	r := Parse(join(toolResultLine(t, body)), false)
	if len(r.Events) != 1 {
		t.Fatalf("got %d events, want 1", len(r.Events))
	}
	e := r.Events[0]
	if len(e.Text) != 500 || e.Cut != 100 {
		t.Fatalf("len(Text)=%d Cut=%d, want 500 and 100", len(e.Text), e.Cut)
	}
	if len(e.Text)+e.Cut != len(body) {
		t.Fatalf("len(Text)+Cut is %d, want the original %d", len(e.Text)+e.Cut, len(body))
	}
}

// R8 ③ — 상한이 룬 가운데로 떨어지면 룬 앞으로 물러난다.
//
// 한국어는 UTF-8 에서 글자당 3 바이트다. 498 바이트 뒤에 한글을 두면 500 번째
// 바이트가 그 룬의 세 번째 바이트이고, 바이트로 그냥 자르면 거기서 쪼개진다.
func TestParse_TheCutStepsBackToARuneBoundary(t *testing.T) {
	// 바이트가 한 자리다 — 같은 파일이 FuzzParse 의 시드이기도 하다.
	// 시드로도 두는 이유는 그것이 없으면 F2 의 전제(Cut > 0)가 코퍼스에서
	// 한 번도 안 서서 그 불변식이 공허해지기 때문이다. 실측으로 그랬다.
	body := strings.Repeat("a", 498) + "한국"
	if len(body) != 504 {
		t.Fatalf("the fixture is %d bytes, want 504", len(body))
	}
	if utf8.RuneStart(body[500]) {
		t.Fatalf("byte 500 is a rune start, so this test measures nothing")
	}

	r := Parse(join(line("long-tool-result.json")), false)
	e := r.Events[0]
	if len(e.Text) != 498 {
		t.Fatalf("len(Text)=%d, want 498 — the cut did not step back", len(e.Text))
	}
	if len(e.Text)+e.Cut != len(body) {
		t.Fatalf("len(Text)+Cut is %d, want the original %d", len(e.Text)+e.Cut, len(body))
	}
	if !utf8.ValidString(e.Text) {
		t.Fatalf("the cut text is not valid utf-8: %q", e.Text)
	}
}

// R8 ② · R8.3 — 상한이 걸리는 Kind 는 둘뿐이다.
func TestParse_TextAndRawAreNeverCut(t *testing.T) {
	body := strings.Repeat("b", 600)

	r := Parse(join(textBlockLine(t, body)), false)
	if e := r.Events[0]; e.Cut != 0 || len(e.Text) != 600 {
		t.Fatalf("a text block was cut: len=%d Cut=%d", len(e.Text), e.Cut)
	}

	// raw 와 plain 도 줄 원문을 통째로 든다.
	long := strings.Repeat("c", 700)
	if e := Parse(join(long), false).Events[0]; e.Cut != 0 || e.Text != long {
		t.Fatalf("a plain line was cut: len=%d Cut=%d", len(e.Text), e.Cut)
	}
}

// R8.4 — 원래 깨진 바이트는 깨진 채로 남는다. 치환도 마스킹도 없다.
func TestParse_BrokenBytesAreLeftBroken(t *testing.T) {
	// tool_use 의 입력은 json.RawMessage 를 compact 한 것이라 잘못된 UTF-8 이
	// 살아남는다. 파서가 그것을 고치면 그것이 마스킹이다.
	ln := "{\"type\":\"assistant\",\"message\":{\"content\":[" +
		"{\"type\":\"tool_use\",\"name\":\"Read\",\"input\":{\"k\":\"\xff\xfe\"}}]}}"
	r := Parse(join(ln), false)
	if len(r.Events) != 1 {
		t.Fatalf("got %d events, want 1", len(r.Events))
	}
	if !strings.Contains(r.Events[0].Text, "\xff\xfe") {
		t.Fatalf("the broken bytes were rewritten: %q", r.Events[0].Text)
	}
}

// R10.2 — 어떤 입력에도 안 멈춘다. 오류를 안 돌려주므로 남는 경로가 패닉뿐이다.
func TestParse_NothingPanics(t *testing.T) {
	huge := strings.Repeat("x", 1<<20)
	for _, tc := range []struct {
		name string
		in   []byte
	}{
		{"no bytes at all", nil},
		{"newlines only", []byte("\n\n\n")},
		{"json that stops halfway", []byte(`{"type":"assistant","message":{"con` + "\n")},
		{"one enormous line", []byte(huge + "\n")},
		{"a line that starts mid-rune", []byte("\x80\x80\x80\n")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			Parse(tc.in, false)
			Parse(tc.in, true)
		})
	}
}

// R12 — 붙는 경로 하나.
func TestParse_AToolResultTakesItsNameFromTheCall(t *testing.T) {
	r := Parse(join(line("assistant-tool-id.json"), line("user-result-id.json")), false)

	var result *Event
	for i := range r.Events {
		if r.Events[i].Kind == KindToolResult {
			result = &r.Events[i]
		}
	}
	if result == nil {
		t.Fatalf("there was no tool result event: %+v", r.Events)
	}
	if result.Name != "Read" {
		t.Fatalf("the name was not attached: %+v", result)
	}
}

// R12.2 · R12.3 — 못 붙이는 경로 셋. 붙이지 않고 그대로 사건이다.
func TestParse_WhenThereIsNoPairTheNameStaysEmpty(t *testing.T) {
	for _, tc := range []struct {
		name  string
		lines []string
	}{
		// ① 링이 감겨 호출 줄이 잘렸다. ID 는 있고 짝이 없다.
		{"the call line is not in the input", []string{line("user-result-id.json")}},
		// ② 껍데기 줄기. ID 가 아예 없고 ok 가 사건 하나에 하나뿐이다.
		{"a shell stem", []string{
			line("shell-assistant.json"),
			`{"type":"user","ok":true}`,
		}},
		// ③ tool_use_id 가 줄에 없다.
		{"the result carries no id", []string{
			line("assistant-tool-id.json"),
			line("user-result.json"),
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := Parse(join(tc.lines...), false)
			for _, e := range r.Events {
				if e.Kind == KindToolResult && e.Name != "" {
					t.Fatalf("a name was attached with no pair to attach it from: %+v", e)
				}
			}
		})
	}
}

// 배열 꼴 content — 실측이 2026-09-16 에 이 모양을 처음 보였다.
//
// 이미지를 읽는 턴에서 content 가 배열로 오고 그 안에 image 블록 하나뿐이다.
// text 원소가 0 이라 건질 본문이 없다. 아는 모양이 아니면 없는 것으로 본다.
func TestParse_AToolResultWhoseContentIsAnArrayCarriesNoText(t *testing.T) {
	r := Parse(join(line("user-result-array.json")), false)
	if len(r.Events) != 1 {
		t.Fatalf("got %d events, want 1", len(r.Events))
	}
	e := r.Events[0]
	if e.Kind != KindToolResult {
		t.Fatalf("got kind %q, want tool_result", e.Kind)
	}
	if e.Text != "" || e.Cut != 0 {
		t.Fatalf("an array content was read as a body: Text=%q Cut=%d", e.Text, e.Cut)
	}
	// 그래도 사건이고 id 는 선다 — 줄을 안 버린다.
	if e.ID != "toolu_01Image" {
		t.Fatalf("the tool_use_id was lost: %+v", e)
	}
}

// 두 번째 표시 줄은 집계가 아니라 raw 다 — 집계가 한 자리뿐이다.
func TestParse_OnlyTheFirstElidedMarkerIsTheAggregate(t *testing.T) {
	r := Parse(join(line("elided.json"), line("elided.json")), false)
	if r.Elided == nil {
		t.Fatalf("the first marker did not become the aggregate")
	}
	if r.Elided.Events != 2 || r.Elided.Bytes != 345 {
		t.Fatalf("the aggregate is %+v, want events 2 bytes 345", *r.Elided)
	}
	if r.Raw != 1 {
		t.Fatalf("the second marker made %d raw events, want 1", r.Raw)
	}
	if r.Lines != 2 {
		t.Fatalf("read %d lines, want 2 — both markers count", r.Lines)
	}
}

// init 과 result 의 값이 Info 로 선다.
func TestParse_TheEnvelopesCarryTheirValues(t *testing.T) {
	r := Parse(join(line("init.json"), line("result.json")), false)
	if len(r.Events) != 2 {
		t.Fatalf("got %d events, want 2", len(r.Events))
	}
	init, res := r.Events[0], r.Events[1]
	if init.Kind != KindInit || init.Info.Tools != 2 {
		t.Fatalf("the init event is %+v", init)
	}
	if init.Info.Servers == nil || len(init.Info.Servers) != 0 {
		t.Fatalf("mcp_servers should be an empty slice, got %+v", init.Info.Servers)
	}
	if res.Kind != KindResult || res.Sub != "success" || res.Info.Turns != 4 {
		t.Fatalf("the result event is %+v", res)
	}
	if res.Info.CostUSD != 0.5 {
		t.Fatalf("the cost is %v, want 0.5", res.Info.CostUSD)
	}
}

// 선 위에서 events 는 언제나 배열이다. 짝 팩의 verdict.checks 와 같은 자리이고
// 같은 방법으로 닫았다 - 타입이 자기 선 위 모양을 진다.
//
// 빈 입력은 예외가 아니라 흔한 상태다: 아직 아무도 아무 말도 안 한 단계.
// 그 자리가 null 로 나가면 사건 배열을 검사하는 화면이 계약 위반으로 거절하고,
// 카드가 "아직 첫 글자 전" 대신 오류로 선다.
func TestEventsMarshalAsAnArrayEvenWhenEmpty(t *testing.T) {
	for name, r := range map[string]Result{
		"zero value":   {},
		"parsed empty": Parse(nil, false),
		"empty bytes":  Parse([]byte{}, false),
	} {
		b, err := json.Marshal(r)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !bytes.Contains(b, []byte(`"events":[]`)) {
			t.Errorf("%s: events is not an empty array on the wire: %s", name, b)
		}
		var back struct {
			Events []Event `json:"events"`
		}
		if err := json.Unmarshal(b, &back); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if back.Events == nil {
			t.Errorf("%s: a reader still sees null", name)
		}
	}
	// 사건이 있으면 그대로다 - 별칭 타입이 나머지 필드를 안 잃는다.
	full := Parse([]byte("{\"type\":\"system\",\"subtype\":\"init\",\"model\":\"m\"}\n"), false)
	b, err := json.Marshal(full)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(b, []byte(`"events":[]`)) || !bytes.Contains(b, []byte(`"lines":1`)) {
		t.Errorf("a non-empty result lost something: %s", b)
	}
}
