package enode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ★ $OUT 을 문자 그대로 주면 모델이 확장하지 않는다 ★
//
// 실물 claude 에서 밟았다 — 6턴을 쓰고도 아무 파일도 안 만들었다.
// 어댑터는 경로를 아는데 모델은 모른다. ★ 아는 쪽이 적어준다. ★
func TestPromptCarriesLiteralPath(t *testing.T) {
	out := "/tmp/enode-out-123"
	p := buildPrompt("회귀를 짚어라", out, []string{"hypothesis"}, nil, nil, 0, false, nil, nil, nil, "", "")
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
	p := buildPrompt("요청", "/o", []string{"hypothesis"}, sch, nil, 0, false, nil, nil, nil, "", "")
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
		map[string]string{"build_log": "error: undefined reference"}, 1, false, nil, nil, nil, "", "")
	fb := strings.Index(p, "error: undefined reference")
	req := strings.Index(p, "REQUEST_MARKER")
	if fb < 0 || req < 0 || fb > req {
		t.Fatalf("되먹임이 요청 앞에 없다: fb=%d req=%d", fb, req)
	}
	if !strings.Contains(p, "앞 시도가 실패했다") {
		t.Fatal("재시도라는 것이 안 보인다")
	}
	// 1회차에는 되먹임이 없다
	if strings.Contains(buildPrompt("R", "/o", []string{"x"}, nil, nil, 0, false, nil, nil, nil, "", ""), "앞 시도가 실패했다") {
		t.Fatal("첫 시도인데 재시도 문구가 붙었다")
	}
}

// ★ 실패 차선은 스키마가 있든 없든 항상 붙는다 ★ (ADR-038)
func TestBuildPrompt_실패차선(t *testing.T) {
	// 스키마 없는 단계
	p := buildPrompt("빌드해라", "/o", []string{"log"}, nil, nil, 0, false, nil, nil, nil, "", "")
	if !strings.Contains(p, "/o/_cannot") {
		t.Fatalf("★ 스키마 없는 단계에 차선이 없다 ★:\n%s", p)
	}
	if !strings.Contains(p, "성공을 주장하지 말고") {
		t.Fatalf("★ 거짓 성공을 막는 문구가 없다 ★")
	}
	// ★ 대체물이 아니라는 것도 말해야 한다 ★ — 안 그러면 _cannot 만 내고 끝낸다
	if !strings.Contains(p, "대체물이 아니다") {
		t.Fatalf("★ 「단계는 그대로 실패한다」가 없다 ★")
	}

	// 스키마 있는 단계에도 붙는다 (ADR-020 문구와 ★ 함께 ★)
	sch := map[string]json.RawMessage{"r": json.RawMessage(`{"type":"object"}`)}
	p = buildPrompt("리뷰해라", "/o", []string{"r"}, sch, nil, 0, false, nil, nil, nil, "", "")
	if !strings.Contains(p, "/o/_cannot") || !strings.Contains(p, "부재는 크래시와 구분되지 않는다") {
		t.Fatalf("★ 둘이 함께 있어야 한다 ★:\n%s", p)
	}
}

// ★ 자백을 읽으면 ok 가 cannot 이 된다 ★ (ADR-038)
func TestReadCannot(t *testing.T) {
	out := t.TempDir()
	if _, ok := readCannot(out); ok {
		t.Fatal("없는데 있다고 했다")
	}
	if err := os.WriteFile(filepath.Join(out, cannotName),
		[]byte("  툴체인이 없다  \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	why, ok := readCannot(out)
	if !ok || why != "툴체인이 없다" {
		t.Fatalf("★ 이유를 못 읽었다 ★: %q %v", why, ok)
	}
	// ★ 빈 파일도 자백이다 ★
	if err := os.WriteFile(filepath.Join(out, cannotName), []byte("\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if why, ok := readCannot(out); !ok || why == "" {
		t.Fatalf("★ 빈 자백이 무시됐다 ★: %q %v", why, ok)
	}
}

// ★ cannot 은 완주다 ★ — 크래시가 아니라 정직한 보고이므로 산출물을 믿을 수 있다.
func TestReasonCannot_완주로_친다(t *testing.T) {
	if !ReasonCannot.Completed() {
		t.Fatal("★ cannot 이 완주가 아니면 산출물을 통째로 버린다 ★ — 정직한 보고인데")
	}
	// 크래시 계열은 그대로 완주가 아니다.
	for _, r := range []Reason{ReasonError, ReasonTimeout} {
		if r.Completed() {
			t.Fatalf("%q 가 완주가 됐다", r)
		}
	}
}

// ★ 되먹임은 회차와 무관하게 실린다 ★ (ADR-048)
//
// 예전에는 attempt > 0 일 때만 실었다. 그래서 계획이 지은 재계획 단계가
// ★ 앞 단계 로그를 하나도 못 봤다 ★ — expands 로 붙은 단계는 attempt 0 이다.
// 실측에서 밟았다: 재계획 에이전트가 "요청 섹션이 비어 있고 입력 디렉터리도
// 비어 있어 무엇을 고칠지 모르겠다" 며 _cannot 을 남겼다.
func Test되먹임은_첫_시도에도_실린다(t *testing.T) {
	fb := map[string]string{"build_log": "error: 뭔가 터졌다"}

	// ★ attempt 0 — 계획이 지은 재계획 단계의 자리 ★
	got := buildPrompt("다시 짜라", "/o", []string{"plan2"}, nil, fb, 0, true, []string{"a"}, nil, nil, "", "")
	if !strings.Contains(got, "error: 뭔가 터졌다") {
		t.Fatal("★ 첫 시도인데 되먹임이 안 실렸다 ★ — 재계획이 로그를 못 본다")
	}
	// ★ 실패했다고 단정하지 않는다 ★ — 성공한 로그를 보고 판단하는 자리이기도 하다.
	if strings.Contains(got, "앞 시도가 실패했다") {
		t.Fatal("★ attempt 0 인데 「앞 시도가 실패했다」라고 적었다 ★")
	}
	if !strings.Contains(got, "앞 단계들이 남긴 것") {
		t.Fatalf("제목이 없다: %s", got)
	}

	// ★ attempt > 0 — 재시도. 앞 시도의 나가 남긴 것이다 ★
	got = buildPrompt("고쳐라", "/o", []string{"x"}, nil, fb, 2, false, nil, nil, nil, "", "")
	if !strings.Contains(got, "앞 시도가 실패했다 (2회차)") {
		t.Fatalf("재시도 제목이 없다: %s", got)
	}
}
