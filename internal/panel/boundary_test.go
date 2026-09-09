package panel_test

import (
	"os/exec"
	"strings"
	"testing"
)

// TestImportBoundaries 는 unit-of-work §5 · business-rules §3 의 임포트 경계를
// 지킨다: panel 은 store 도 api 도 안 딛고, 데몬(internal/enode)은 panel 을 안
// 딛는다. cmd/enode 는 panel 을 딛어도 된다(제어판 하위명령).
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

	panelDeps := deps("internal/panel")
	for _, forbidden := range []string{"internal/store", "internal/api"} {
		if has(panelDeps, forbidden) {
			t.Errorf("internal/panel must not import %s", forbidden)
		}
	}

	if has(deps("internal/enode"), "internal/panel") {
		t.Error("internal/enode (the daemon library) must not import internal/panel")
	}

	if !has(deps("cmd/enode"), "internal/panel") {
		t.Error("cmd/enode should import internal/panel (the panel subcommand)")
	}
}
