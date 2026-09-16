package transcript

import "encoding/json"

// 이 파일의 다섯도 internal/enode/runner.go 에서 몸통 그대로 옮겨 온 것이다.
// 옮기면서 바꾼 것은 하나다 - enode.elided 의 글자가 상수로 올라갔다.
// 짓는 쪽만 있을 때는 리터럴이 한 자리였지만 읽는 쪽(parse.go)이 생기면서
// 두 자리가 될 뻔했다. 바이트 출력은 한 글자도 안 바뀐다.

const (
	// enodePrefix 가 붙은 type 은 enode 가 찍은 표시 줄이다.
	//
	// 점을 넣는다 — 실측한 하네스의 type 은 전부 홑단어라
	// (system · assistant · user · result · rate_limit_event) 부딪칠 수 없다.
	enodePrefix = "enode."

	// elidedType 은 걷었음을 표시하는 줄의 type 이다. selectLogs 가 찍는다.
	elidedType = enodePrefix + "elided"

	// cappedType 은 진행 파일이 상한에 닿아 멈춘 자리를 표시하는 줄의 type 이다.
	//
	// 찍는 쪽이 이 패키지 밖(internal/record)이라 왕복 시험이 못 잡는다.
	// 그래서 글자가 여기 한 자리에 있고, 맞대는 일은 사람이 한다.
	cappedType = enodePrefix + "capped"
)

// logShell 은 사건 하나가 logs/ 에 남기는 전부다.
//
// 필드를 짓는 쪽이 허용목록이다 — 원본에서 지우는 것이 아니라 새 객체를
// 지으므로 하네스가 필드를 늘려도 안 샌다. 본문은 어느 사건에서도 안 남는다:
// assistant 의 text 도 thinking 도 도구 결과도 같다.
//
// 시각을 안 넣는다 — exec 이 끝난 뒤 한 번에 선별하므로 사건마다의 시각을
// 못 찍고, 넣으면 모든 줄이 같은 값이라 정보량이 0 이다.
//
// 필드 다섯과 태그를 안 바꾼다 — 이미 봉인된 Run 의 logs/ 를 못 읽게 된다.
// 봉인은 0444 로 굳어 고칠 수 없다.
type logShell struct {
	Type    string         `json:"type"`
	Subtype string         `json:"subtype,omitempty"`
	Tools   []string       `json:"tools,omitempty"`
	OK      *bool          `json:"ok,omitempty"`
	Tokens  map[string]int `json:"tokens,omitempty"`
}

// elidedMark 는 걷었음을 표시하는 줄이다.
//
// 짓는 쪽(ElidedMarker)과 읽는 쪽(Parse)이 같은 구조체를 본다. 오늘까지는
// 짓는 쪽만 있어 갈릴 수 없었을 뿐이고, 읽는 쪽이 생기는 순간 키 이름이 두
// 자리에 적힐 뻔했다.
type elidedMark struct {
	Type   string `json:"type"`
	Events int    `json:"events"`
	Bytes  int    `json:"bytes"`
}

// ElidedMarker 는 걷은 양을 줄 하나로 짓는다. 마샬이 실패하면 nil 이다.
//
// 인자가 정수 둘인 것은 호출자(selectLogs)가 정수 둘을 손에 들고 있기
// 때문이다. Elided 하나로 묶으면 옮기기에 모양 바꾸기가 섞인다.
func ElidedMarker(events, bytes int) []byte {
	b, err := json.Marshal(elidedMark{Type: elidedType, Events: events, Bytes: bytes})
	if err != nil {
		return nil
	}
	return b
}

// Shell 은 사건 하나를 껍데기 한 줄로 짓는다. 마샬이 실패하면 nil 이다.
func Shell(obj Fields, typ string) []byte {
	sh := logShell{Type: typ}
	if typ == "system" {
		sh.Subtype = String(obj, "subtype")
	}
	var msg struct {
		Content []Fields `json:"content"`
		Usage   Fields   `json:"usage"`
	}
	if json.Unmarshal(obj["message"], &msg) == nil {
		ok, sawResult := true, false
		for _, blk := range msg.Content {
			switch String(blk, "type") {
			case "tool_use":
				// 배열로 둔다 — 실측은 사건마다 블록 하나였지만 그것이
				// 보증은 아니다. 하나일 때도 배열이면 뒤에 모양이 안 갈린다.
				sh.Tools = append(sh.Tools, String(blk, "name"))
			case "tool_result":
				sawResult = true
				// 성공하면 is_error 키가 아예 없다 (실측).
				if v, has := Bool(blk, "is_error"); has && v {
					ok = false
				}
			}
		}
		if sawResult {
			// 있을 때만 쓴다 — 도구를 안 부른 사건에 ok: true 를 박으면
			// 「성공한 도구가 있었다」로 읽힌다. 없음과 참을 가른다.
			sh.OK = &ok
		}
		sh.Tokens = usageTokens(msg.Usage)
	}
	// system/thinking_tokens 의 estimated_tokens — usage 밖의 정수 하나다.
	// 같은 종류의 값이라(본문이 없는 정수 하나) 싣는 쪽으로 정했고,
	// 범위를 넓힌 자리라 이름으로 적는다. 빼려면 이 블록을 지운다.
	if n, has := Int(obj, "estimated_tokens"); has {
		if sh.Tokens == nil {
			sh.Tokens = map[string]int{}
		}
		sh.Tokens["thinking"] = n
	}
	b, err := json.Marshal(sh)
	if err != nil {
		return nil
	}
	return b
}

// usageTokens 는 usage 에서 정수 넷만 집는다.
//
// 통째로 못 옮긴다 — 실측이 usage 안에 문자열 둘(service_tier ·
// inference_geo)과 객체 하나(cache_creation)를 보였다. 그래서 이름으로 집고,
// 정수가 아니면 그 키를 건너뛴다. 모르는 것은 안 싣는다.
func usageTokens(usage Fields) map[string]int {
	if len(usage) == 0 {
		return nil
	}
	names := [...]struct{ out, in string }{
		{"in", "input_tokens"},
		{"out", "output_tokens"},
		{"cache_write", "cache_creation_input_tokens"},
		{"cache_read", "cache_read_input_tokens"},
	}
	var tk map[string]int
	for _, n := range names {
		v, has := Int(usage, n.in)
		if !has {
			continue
		}
		if tk == nil {
			tk = map[string]int{}
		}
		tk[n.out] = v
	}
	return tk
}
