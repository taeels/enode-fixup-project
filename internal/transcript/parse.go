package transcript

import (
	"bytes"
	"encoding/json"
	"strings"
	"unicode/utf8"
)

// 표시 상한 둘. 숫자는 팩의 것이고 단위(바이트)와 룬 경계 규칙은 이 유닛이
// 정했다. 한국어는 UTF-8 에서 글자당 3 바이트라 같은 숫자가 3 배 다른 값이다.
//
// 자르는 자리가 하나다 - 화면은 받은 Text 를 그대로 그린다. 상한이 두 자리에
// 있으면 갈리고, as=events 의 응답 크기가 파서에서 이미 묶여야 부르는 쪽이
// 그 부하를 계산할 수 있다.
const (
	maxToolUseText    = 200
	maxToolResultText = 500
)

// Parse 는 하네스 stdout 한 뭉치를 사건 열로 읽는다.
//
// truncated 는 앞이 잘려 왔다는 뜻이다 - 링이 감기면 첫 줄이 반만 남는다.
//
// 시계를 안 받는다. 같은 입력에 언제나 같은 Result 를 낸다 - 경과와 갱신
// 시각은 화면이 뺀다. 로거도 안 받고 error 도 안 돌려준다 - 못 읽은 줄은
// raw 나 plain 으로 남고 오류로 단계를 죽이는 경로가 이 패키지에 0 이다.
//
// 파이프라인이 여덟이고 순서가 값이다. 머리 · 꼬리 자르기가 줄 나누기보다
// 앞인 것이 특히 그렇다 - 뒤에 두면 잘린 조각이 줄로 세어져 Lines 가 부풀고,
// 그 수로 세는 불변식이 거짓이 된다.
func Parse(b []byte, truncated bool) Result {
	var r Result

	// 1. 머리 자르기. 버린 바이트 수를 Head 에 적는다.
	if truncated {
		i := bytes.IndexByte(b, '\n')
		if i < 0 {
			// 개행이 하나도 없다 - 줄 하나가 통째로 반만 온 것이라 읽을
			// 것이 0 이다.
			return Result{}
		}
		r.Head = i + 1
		b = b[i+1:]
	}

	// 2. 꼬리 자르기. 개행으로 안 끝나면 마지막 조각을 떼고 Partial 에 적는다.
	if n := len(b); n > 0 && b[n-1] != '\n' {
		if i := bytes.LastIndexByte(b, '\n'); i < 0 {
			r.Partial = n
			b = nil
		} else {
			r.Partial = n - (i + 1)
			b = b[:i+1]
		}
	}

	// 3. 줄 나누기. selectLogs 와 같은 함수다.
	lines := SplitLines(b)
	r.Lines = len(lines)

	// 4 ~ 6. 줄마다 판정하고 사건을 짓는다.
	for i, ln := range lines {
		evs, agg := lineToEvents(ln, i+1, r.Elided == nil)
		if agg != nil {
			r.Elided = agg
			continue
		}
		r.Events = append(r.Events, evs...)
	}

	// 7. 붙이기.
	attachNames(r.Events)

	// 8. 세기.
	for i := range r.Events {
		if r.Events[i].Kind == KindRaw {
			r.Raw++
		}
	}
	return r
}

// lineToEvents 는 줄 하나를 사건 열로 읽는다. 집계 줄이면 사건이 0 이고 둘째
// 값이 그 집계다.
//
// 읽은 줄마다 사건 하나 이상을 내거나 집계 줄이다. 사건을 0 개 내고 넘어가는
// 줄이 없다 - 파서가 사건을 고치거나 버리는 것을 팩이 제외 기능으로 적었다.
func lineToEvents(ln []byte, no int, wantAggregate bool) ([]Event, *Elided) {
	obj, typ, ok := ParseLine(ln)
	if !ok {
		// 사다리 1. JSON 객체가 아니거나 type 이 문자열이 아니다.
		//
		// raw 가 아니라 text 다. 그런 줄이 오늘 세 경로에 있고(명령 단계의
		// 링과 로그, 선별본 꼬리의 stderr) 셋 다 사람이 읽으라고 있는
		// 글자다. raw 로 떨어뜨리면 명령 단계에서 읽을 수 있는 문장이 안
		// 흐른다. 대가는 text 가 두 사실(에이전트가 말했다 · JSON 이
		// 아니었다)을 지게 되는 것이고, 가르는 것은 Sub 하나다.
		return []Event{{Kind: KindText, Sub: "plain", Line: no, Text: string(ln)}}, nil
	}
	evs, agg := jsonLineEvents(obj, typ, ln, no, wantAggregate)
	if len(evs) > 0 {
		// Tokens 는 그 줄이 낸 첫 사건에만 싣는다 - 와이어의 usage 는 줄
		// 하나의 예산이지 블록 하나의 예산이 아니다. 사건마다 복사하면
		// 화면이 합계를 내다가 같은 값을 여러 번 더한다.
		evs[0].Tokens = lineTokens(obj)
	}
	return evs, agg
}

// jsonLineEvents 는 사다리의 2 · 3 · 4 다.
func jsonLineEvents(obj Fields, typ string, ln []byte, no int, wantAggregate bool) ([]Event, *Elided) {
	// 사다리 2. enode 가 찍은 표시 줄이다. 3 · 4 보다 위에 둔다.
	//
	// 아래로 내리면 capped 가 사상표에서 못 찾아 raw 로 떨어지고, raw 의
	// Text 는 줄 원문이라 화면이 「상한에 닿아 멈췄다」 대신 JSON 한 줄을
	// 그대로 그린다. 그것은 「단계가 멈춘 줄 안다」는 오독을 못 막는다.
	if strings.HasPrefix(typ, enodePrefix) {
		switch typ {
		case elidedType:
			// 파일 전체의 집계라 어느 줄에 있든 뜻이 같다. 사건이 아니다.
			//
			// 둘째 표시 줄은 집계가 아니라 raw 로 떨어진다 - 집계가 한
			// 자리뿐이므로 둘을 담을 데가 없고, 버리면 줄 수가 안 맞는다.
			// 모양이 다른 줄도 같은 자리로 간다.
			var mk elidedMark
			if wantAggregate && json.Unmarshal(ln, &mk) == nil {
				return nil, &Elided{Events: mk.Events, Bytes: mk.Bytes}
			}
		case cappedType:
			// 위치가 값이다. 「그 사건 다음부터 없다」가 전부이고, 집계로
			// 빼면 사건 열 어디서 끊겼는지를 잃는다.
			//
			// bytes 는 닿은 상한이지 총 길이가 아니다. 총 길이로 고쳐 읽지
			// 않는다 - 표시 줄도 총 길이에 들어가므로 자기가 든 총 길이를
			// 담을 수 없다.
			return []Event{{
				Kind: KindCapped,
				Line: no,
				Info: Info{Bytes: int64Of(obj, "bytes")},
			}}, nil
		}
		// 모르는 enode.* 는 raw 다. 아는 둘에만 자리를 준다 - 뜻을 모르는
		// 줄에 화면이 문장을 지어 붙이면 그것이 거짓이 된다.
		return []Event{rawEvent(typ, ln, no)}, nil
	}

	// 사다리 3 · 4.
	switch typ {
	case "assistant", "user":
		// 껍데기 판정을 이 둘에만 건다. 그 둘만 Shell 이 본문을 걷는
		// 대상이고, 그 둘의 원문에는 message 키가 언제나 있으며 그 둘의
		// 껍데기에는 절대 없다 - logShell 의 필드 다섯에 message 가 없다.
		return messageEvents(obj, no), nil
	case "system":
		if String(obj, "subtype") == "init" {
			return []Event{initEvent(obj, no)}, nil
		}
		return []Event{rawEvent(systemSub(obj), ln, no)}, nil
	case "result":
		return []Event{resultEvent(obj, no)}, nil
	}
	// 사상표에 없다. 버리지 않고 raw 로 남긴다.
	return []Event{rawEvent(typ, ln, no)}, nil
}

// messageEvents 는 assistant · user 줄 하나를 사건 열로 읽는다.
//
// 원문 줄기와 껍데기 줄기가 여기서 갈린다. 가르는 것은 message 키 하나다.
func messageEvents(obj Fields, no int) []Event {
	raw, hasMessage := obj["message"]
	if !hasMessage {
		return shellEvents(obj, no)
	}

	var msg struct {
		Content []Fields `json:"content"`
	}
	// 모양이 다르면 블록이 0 이다. 아래의 되돌림이 그 줄을 사건 하나로 세운다.
	_ = json.Unmarshal(raw, &msg)

	var evs []Event
	for _, blk := range msg.Content {
		// 블록의 종류로 가른다. 줄의 종류로 다시 가르지 않는다 - 실측에서
		// assistant 는 tool_result 를 안 들고 user 는 tool_use 를 안 들어
		// 결과가 같고, 줄로 가르면 안 본 모양 하나가 본문을 통째로 지운다.
		switch String(blk, "type") {
		case "text":
			evs = append(evs, Event{Kind: KindText, Line: no, Text: String(blk, "text")})
		case "thinking":
			// text 와 같은 Kind 로 두되 Sub 로 가른다. 어휘가 닫혀 있어
			// 같은 Kind 이고, 화면이 생각을 기본으로 접을 수 있어야 해서
			// 갈라 적는다 - 접는 일은 화면의 몫이고 파서는 근거만 준다.
			evs = append(evs, Event{Kind: KindText, Sub: "thinking", Line: no, Text: String(blk, "thinking")})
		case "tool_use":
			ev := Event{Kind: KindToolUse, Line: no, Name: String(blk, "name"), ID: String(blk, "id")}
			ev.Text, ev.Cut = cutTo(compactJSON(blk["input"]), maxToolUseText)
			evs = append(evs, ev)
		case "tool_result":
			ev := Event{Kind: KindToolResult, Line: no, ID: String(blk, "tool_use_id")}
			ok := true
			// 성공하면 is_error 키가 아예 없다 (실측).
			if v, has := Bool(blk, "is_error"); has && v {
				ok = false
			}
			ev.OK = &ok
			// content 가 문자열이면 그대로 쓰고 그 밖의 모양이면 비운다 -
			// 아는 모양이 아니면 없는 것으로 본다. 실측한 한 턴에서 그
			// 값은 문자열이었다.
			ev.Text, ev.Cut = cutTo(String(blk, "content"), maxToolResultText)
			evs = append(evs, ev)
		}
		// 모르는 블록 종류는 건너뛴다. 버리는 것이 아니다 - 그 줄의 원문은
		// 원문 토글이 언제나 든다.
	}
	if len(evs) == 0 {
		// 블록에서 사건이 하나도 안 나온 줄도 사건 하나를 낸다.
		// {"type":"assistant"} 가 사건 0 이 되면 「아무 말도 안 했다」와
		// 「걷혔다」가 화면에서 같아 보인다.
		evs = append(evs, Event{Kind: KindText, Line: no})
	}
	return evs
}

// shellEvents 는 껍데기 줄 하나를 사건 열로 읽는다.
//
// Shell 이 "tools" 로 쓴 것을 여기서 "tools" 로 읽는다. 두 줄이 같은 패키지에
// 있어 한쪽만 고치면 왕복 시험이 그 자리에서 빨개진다.
func shellEvents(obj Fields, no int) []Event {
	var evs []Event
	var tools []string
	if json.Unmarshal(obj["tools"], &tools) == nil {
		for _, name := range tools {
			evs = append(evs, Event{Kind: KindToolUse, Line: no, Name: name, Shell: true})
		}
	}
	if v, has := Bool(obj, "ok"); has {
		ok := v
		evs = append(evs, Event{Kind: KindToolResult, Line: no, OK: &ok, Shell: true})
	}
	if len(evs) == 0 {
		evs = append(evs, Event{Kind: KindText, Line: no, Shell: true})
	}
	return evs
}

// initEvent 는 첫 system/init 줄의 값을 Info 에 담는다.
//
// 버전의 키가 claude_code_version 인 것은 실측으로 고른 것이다
// (claude 2.1.271 · 2026-09-15). 줄에 없으면 빈 문자열이다.
func initEvent(obj Fields, no int) Event {
	info := Info{
		Model:   String(obj, "model"),
		Version: String(obj, "claude_code_version"),
	}
	// 도구 수는 배열의 길이다. 이름을 안 읽는다 - 화면이 세기만 한다.
	var tools []json.RawMessage
	if json.Unmarshal(obj["tools"], &tools) == nil {
		info.Tools = len(tools)
	}
	// mcp_servers 는 배열이고 맵이 아니다 (실측).
	var servers []Server
	if json.Unmarshal(obj["mcp_servers"], &servers) == nil {
		info.Servers = servers
	}
	return Event{Kind: KindInit, Line: no, Info: info}
}

// resultEvent 는 마지막 result 줄의 값을 Info 에 담는다.
//
// Reason 을 HarnessResult 의 어휘로 안 옮긴다. 그 번역은 ParseClaude 가 하고
// 그 자리는 이 패키지 밖이다 - 두 벌로 두면 갈린다.
func resultEvent(obj Fields, no int) Event {
	sub := String(obj, "subtype")
	info := Info{Reason: sub}
	if n, has := Int(obj, "num_turns"); has {
		info.Turns = n
	}
	if f, has := floatOf(obj, "total_cost_usd"); has {
		info.CostUSD = f
	}
	return Event{Kind: KindResult, Sub: sub, Line: no, Info: info}
}

// rawEvent 는 Kind 로 사상되지 않은 줄을 원문 그대로 든다.
func rawEvent(sub string, ln []byte, no int) Event {
	return Event{Kind: KindRaw, Sub: sub, Line: no, Text: string(ln)}
}

// systemSub 는 init 이 아닌 system 줄의 Sub 다.
func systemSub(obj Fields) string {
	sub := String(obj, "subtype")
	if sub == "" {
		return "system"
	}
	return "system/" + sub
}

// lineTokens 는 줄 하나의 예산 신호를 집는다.
//
// 원문 줄과 껍데기 줄의 키가 다르므로 둘 다 본다 - 한 줄이 둘 다 들 수는
// 없다. Shell 이 싣는 것과 같은 것을 같은 이름으로 읽어야 링에서 읽든
// 봉인된 logs 에서 읽든 같은 값이 선다.
func lineTokens(obj Fields) map[string]int {
	var tk map[string]int
	var msg struct {
		Usage Fields `json:"usage"`
	}
	if json.Unmarshal(obj["message"], &msg) == nil {
		tk = usageTokens(msg.Usage)
	}
	if tk == nil {
		var shell map[string]int
		if json.Unmarshal(obj["tokens"], &shell) == nil && len(shell) > 0 {
			tk = shell
		}
	}
	// usage 밖의 정수 하나다. Shell 이 thinking 키로 싣는 그것을 여기서
	// 같은 이름으로 읽는다.
	if n, has := Int(obj, "estimated_tokens"); has {
		if tk == nil {
			tk = map[string]int{}
		}
		tk["thinking"] = n
	}
	return tk
}

// attachNames 는 tool_result 사건의 Name 을 같은 ID 의 tool_use 에서 채운다.
//
// 짝이 없으면 빈 채로 선다. 그대로 사건이다.
//
// 자리로 짝짓지 않는다 - 「가장 가까운 앞의 tool_use」로 붙이면 도구가 둘인
// 턴에서 절반이 틀리고, 틀린 붙이기는 안 붙인 것보다 나쁘다. 읽는 사람이
// 그 줄을 근거로 쓴다.
//
// 껍데기 줄기에서는 안 돈다. logShell 에 tool_use_id 자리가 없어 ID 가 아예
// 비고, ok 가 사건 하나에 하나뿐이라 도구가 여럿이면 어느 것의 결과인지
// 알 수 없다.
//
// 배치는 안 한다. 어느 호출의 결과인가를 값으로 주고, 겹쳐 그리는 것은
// 카드의 것이다.
func attachNames(evs []Event) {
	var names map[string]string
	for i := range evs {
		e := &evs[i]
		if e.Kind != KindToolUse || e.Shell || e.ID == "" || e.Name == "" {
			continue
		}
		if names == nil {
			names = make(map[string]string)
		}
		names[e.ID] = e.Name
	}
	if names == nil {
		return
	}
	for i := range evs {
		e := &evs[i]
		if e.Kind != KindToolResult || e.Shell || e.ID == "" || e.Name != "" {
			continue
		}
		if name, has := names[e.ID]; has {
			e.Name = name
		}
	}
}

// cutTo 는 s 를 limit 바이트 이하로 자르되 마지막 룬을 안 쪼갠다. 둘째 값은
// 잘려 나간 바이트 수다.
//
// 상한 자리에서 시작해 그 자리의 바이트가 UTF-8 이어바이트인 동안 뒤로
// 물린다. 룬의 최대 길이가 4 바이트라 올바른 입력에서는 최대 세 번이다.
// 앞이 전부 이어바이트인 입력(이미 깨진 것)이면 빈 문자열이 되고 둘째 값이
// 전체 길이다 - 멈추지 않는 경로가 0 이다.
//
// 고치지 않는다. 원래 깨져 있던 바이트는 깨진 채로 남는다 - 치환도 마스킹도
// 안 한다. 자르기는 앞에서부터 남기는 것이고 무엇을 가릴지의 규칙이 0 이다.
//
// len(첫째 값) + 둘째 값 == len(s) 가 언제나 참이다. 입력의 유효성과 무관한
// 유일한 자르기 불변식이고, 입력이 올바른 UTF-8 일 때만 첫째 값도 올바른
// UTF-8 이다.
func cutTo(s string, limit int) (string, int) {
	if len(s) <= limit {
		return s, 0
	}
	n := limit
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n], len(s) - n
}

// compactJSON 은 도구 입력을 공백 없는 JSON 으로 다시 적는다.
//
// any 로 풀지 않는다 - 그러면 도구 입력 본문이 파싱 시점에 통째로 메모리로
// 올라오고, 그것이 R2.1 이 막은 모양이다. Compact 는 바이트를 훑으며 공백만
// 뺀다. 잘못된 UTF-8 은 그대로 지난다 - 고치는 것이 파서의 일이 아니다.
func compactJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var buf bytes.Buffer
	if json.Compact(&buf, raw) != nil {
		return string(raw)
	}
	return buf.String()
}

// int64Of 와 floatOf 는 String · Bool · Int 와 같은 규율을 다른 폭으로 든다 -
// 아는 모양이 아니면 없는 것으로 본다. 공개 표면을 안 늘리려고 비공개다.
func int64Of(obj Fields, key string) int64 {
	var n int64
	if json.Unmarshal(obj[key], &n) != nil {
		return 0
	}
	return n
}

func floatOf(obj Fields, key string) (float64, bool) {
	var f float64
	if json.Unmarshal(obj[key], &f) != nil {
		return 0, false
	}
	return f, true
}
