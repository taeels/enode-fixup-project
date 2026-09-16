package panel_test

import (
	"os/exec"
	"strings"
	"testing"
)

// TestImportBoundaries 는 unit-of-work §5 · business-rules §3 의 임포트 경계를
// 지킨다: panel 은 store 도 api 도 안 딛고, 데몬(internal/enode)은 panel 을 안
// 딛는다. cmd/enode 는 panel 을 딛어도 된다(제어판 하위명령).
//
// 이 파일은 이름과 디렉터리만 panel 이지 내용은 저장소 전체의 경계 검사기다 —
// internal/panel 을 한 번도 참조하지 않는다. internal/transcript 의 금지 넷도
// 여기서 잰다. 검사기를 두 벌로 두면 한쪽만 고쳐도 둘 다 초록이라 갈린 것을
// 아무도 못 본다.
//
// go list -deps 를 쓰고 -test 를 안 쓴다. -test 는 목록을 .test 패키지로
// 흐리고, 시험 임포트는 제품 바이너리에 안 실린다. 그래서 이 검사가 못 보는
// 것이 하나 있다 — internal/transcript/*_test.go 가 internal/enode 를
// 임포트해도 여기는 초록이다. 그 자리는 사람이 진다.
func TestImportBoundaries(t *testing.T) {
	const mod = "github.com/taeels/enode/"
	deps := func(pkg string) string {
		out, err := exec.Command("go", "list", "-deps", mod+pkg).CombinedOutput()
		if err != nil {
			t.Fatalf("go list -deps %s: %v\n%s", pkg, err, out)
		}
		return string(out)
	}
	has := func(all, pkg string) bool {
		for _, line := range strings.Split(all, "\n") {
			if strings.TrimSpace(line) == mod+pkg {
				return true
			}
		}
		return false
	}

	// 금지 여덟 줄. 표 하나라 줄을 더하는 것이 한 행이다.
	//
	// 오늘 넷이 자동으로 초록인 것이 쓸모없다는 뜻이 아니다 — 값은 나중에
	// 누가 임포트를 더했을 때 빨개지는 데 있다.
	for _, rule := range []struct{ from, to string }{
		{"internal/panel", "internal/store"},
		{"internal/panel", "internal/api"},
		{"internal/enode", "internal/panel"},
		{"internal/api/ui", "internal/store"},
		{"internal/transcript", "internal/enode"},
		{"internal/transcript", "internal/api"},
		{"internal/transcript", "internal/store"},
		{"internal/transcript", "internal/panel"},
	} {
		if has(deps(rule.from), rule.to) {
			t.Errorf("%s must not import %s", rule.from, rule.to)
		}
	}

	// 봉인 하나 — 금지 넷보다 강하다.
	//
	// 금지는 이름을 아는 넷만 막고 봉인은 전부 막는다: 새 내부 패키지든
	// 새 외부 모듈이든 같다. 넷을 그래도 두는 이유는 실패 메시지가 어느
	// 줄인지를 말해야 하기 때문이다. 두 벌이 아니라 다른 단언 둘이다.
	//
	// 표준 라이브러리는 첫 경로 조각에 점이 없다 (encoding/json · unicode/utf8).
	// 점이 있으면 모듈 경로다 (github.com/... · gopkg.in/...).
	for _, dep := range strings.Split(deps("internal/transcript"), "\n") {
		dep = strings.TrimSpace(dep)
		if dep == "" || dep == mod+"internal/transcript" {
			continue
		}
		if first, _, _ := strings.Cut(dep, "/"); strings.Contains(first, ".") {
			t.Errorf("internal/transcript must depend on the standard library only, but it imports %s", dep)
		}
	}

	// 「있어야 한다」는 금지 표가 못 담는 모양이라 표 밖에 그대로 둔다.
	if !has(deps("cmd/enode"), "internal/panel") {
		t.Error("cmd/enode should import internal/panel (the panel subcommand)")
	}
}
