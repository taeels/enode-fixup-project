package enode

import (
	"encoding/json"
	"strings"
	"testing"
)

// ★ $OUT 을 문자 그대로 주면 모델이 확장하지 않는다 ★
//
// 실물 claude 에서 밟았다 — 6턴을 쓰고도 아무 파일도 안 만들었다.
// 어댑터는 경로를 아는데 모델은 모른다. ★ 아는 쪽이 적어준다. ★
func TestPromptCarriesLiteralPath(t *testing.T) {
	out := "/tmp/enode-out-123"
	p := buildPrompt("회귀를 짚어라", out, []string{"hypothesis"}, nil, nil, 0)
	if !strings.Contains(p, out+"/hypothesis") {
		t.Fatalf("★ 실제 경로가 안 들어갔다 ★:\n%s", p)
	}
	if strings.Contains(p, "$OUT/") {
		t.Fatalf("★ 확장 안 되는 $OUT 이 남았다 ★:\n%s", p)
	}
}

// 스키마가 있으면 ★ 못 하겠다를 값으로 말할 수 있어야 한다 ★ (ADR-020).
func TestPromptCarriesSchemaAndHonestNone(t *testing.T) {
	sch := map[string]json.RawMessage{
		"hypothesis": json.RawMessage(`{"type":"object","required":["status"]}`),
	}
	p := buildPrompt("요청", "/o", []string{"hypothesis"}, sch, nil, 0)
	if !strings.Contains(p, `"required":["status"]`) {
		t.Fatal("스키마가 프롬프트에 안 실렸다")
	}
	if !strings.Contains(p, "부재는 크래시와 구분되지 않는다") {
		t.Fatal("★ 결론이 없을 때 값으로 말하라는 지시가 없다 ★")
	}
}

// 되먹임은 ★ 요청 바로 앞 ★ 에 온다 — 무엇을 고쳐야 하는지가 가장 가깝게 놓인다.
func TestPromptPutsFeedbackJustBeforeRequest(t *testing.T) {
	p := buildPrompt("REQUEST_MARKER", "/o", []string{"x"}, nil,
		map[string]string{"build_log": "error: undefined reference"}, 1)
	fb := strings.Index(p, "error: undefined reference")
	req := strings.Index(p, "REQUEST_MARKER")
	if fb < 0 || req < 0 || fb > req {
		t.Fatalf("되먹임이 요청 앞에 없다: fb=%d req=%d", fb, req)
	}
	if !strings.Contains(p, "앞 시도가 실패했다") {
		t.Fatal("재시도라는 것이 안 보인다")
	}
	// 1회차에는 되먹임이 없다
	if strings.Contains(buildPrompt("R", "/o", []string{"x"}, nil, nil, 0), "앞 시도가 실패했다") {
		t.Fatal("첫 시도인데 재시도 문구가 붙었다")
	}
}
