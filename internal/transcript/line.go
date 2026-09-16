package transcript

import (
	"bytes"
	"encoding/json"
)

// 이 파일의 다섯은 internal/enode/runner.go 에서 몸통 그대로 옮겨 온 것이다.
// 이름만 바뀌었다 - 근거 주석도 함께 왔다. 옮기기에 모양 바꾸기를 안 섞는다.

// SplitLines 는 stdout 을 줄로 가른다. 마지막 빈 조각은 버린다.
func SplitLines(b []byte) [][]byte {
	if len(b) == 0 {
		return nil
	}
	lines := bytes.Split(b, []byte("\n"))
	if n := len(lines); n > 0 && len(lines[n-1]) == 0 {
		lines = lines[:n-1]
	}
	return lines
}

// ParseLine 은 줄 하나를 사건으로 읽는다.
//
// 구조체로 한 번에 안 받는다 — 모르는 필드의 모양 하나가 줄 전체를
// 떨어뜨리기 때문이다. type 을 먼저 집고 아는 키만 따로 읽으면 모르는 것은
// 안 실리고 아는 것은 남는다. 허용목록의 규율이 여기서도 같다.
func ParseLine(line []byte) (Fields, string, bool) {
	var obj Fields
	if json.Unmarshal(line, &obj) != nil {
		return nil, "", false
	}
	typ := String(obj, "type")
	if typ == "" {
		// type 이 없거나 문자열이 아니다 — 무엇인지 모르므로 안 싣는다.
		return nil, "", false
	}
	return obj, typ, true
}

// String · Bool · Int 는 아는 키 하나를 아는 모양으로만 읽는다.
// 모양이 다르면 없는 것으로 본다 — 판정을 짐작으로 메우지 않는다.
//
// 셋이 한 벌이라 함께 공개한다. 안 공개하면 호출자가 json.Unmarshal 로 같은
// 일을 다시 짓고, 그것이 두 벌이다.
func String(obj Fields, key string) string {
	var s string
	if json.Unmarshal(obj[key], &s) != nil {
		return ""
	}
	return s
}

func Bool(obj Fields, key string) (bool, bool) {
	var v bool
	if json.Unmarshal(obj[key], &v) != nil {
		return false, false
	}
	return v, true
}

func Int(obj Fields, key string) (int, bool) {
	var n int
	if json.Unmarshal(obj[key], &n) != nil {
		return 0, false
	}
	return n, true
}
