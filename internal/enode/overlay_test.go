package enode

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestOverlayProbeNeedsAWorkspace 는 잴 자리가 없으면 안 재는지를 본다.
//
// 임시 디렉터리에 재면 안 된다 — 컨테이너의 /var/tmp 는 도커 자기 overlay2 라
// 윗 층을 못 두고, 실제로는 되는데 안 된다고 답하게 된다.
func TestOverlayProbeNeedsAWorkspace(t *testing.T) {
	attrs, err := (overlayFP{}).Probe(context.Background(), Local{}, testLog())
	if err != nil {
		t.Fatalf("a node without a workspace must not fail: %v", err)
	}
	if len(attrs) != 0 {
		t.Errorf("a node without a workspace advertised %v", attrs)
	}
}

// TestOverlayProbeSaysOneOfTheLadder 는 답이 사다리 위의 값인지를 본다.
//
// 어느 칸이 나오는지는 그 기계가 정한다 — 기본 도커면 아무것도 안 나오고,
// cap-add SYS_ADMIN 이면 userns 가, 특권이 있으면 kernel 이 나온다.
// 그래서 값을 못 박지 않고 어휘만 잰다.
func TestOverlayProbeSaysOneOfTheLadder(t *testing.T) {
	ws := t.TempDir()
	attrs, err := (overlayFP{}).Probe(context.Background(), Local{Workspace: ws}, testLog())
	if err != nil {
		t.Fatalf("probing must not fail: %v", err)
	}
	if len(attrs) == 0 {
		t.Log("this machine cannot build an overlay; the attribute is absent, which is the answer")
		return
	}
	t.Logf("this machine builds overlays with %q", attrs["overlay"])
	switch attrs["overlay"] {
	case overlayKernel, overlayUserns, overlayFuse:
	default:
		t.Errorf("overlay is %q, which is not on the ladder", attrs["overlay"])
	}
	if len(attrs) != 1 {
		t.Errorf("the probe advertised more than the one attribute: %v", attrs)
	}
}

// TestOverlayProbeLeavesNothingBehind 는 잰 자리를 치우는지를 본다.
//
// 워크스페이스 안에 만들면 clean -df 가 지우고 관찰 범위에도 들어가므로
// 옆에 만드는데, 옆에 남겨두면 그 디렉터리가 매 탐지마다 쌓인다.
func TestOverlayProbeLeavesNothingBehind(t *testing.T) {
	parent := t.TempDir()
	ws := filepath.Join(parent, "workspace")
	if err := os.MkdirAll(ws, 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := (overlayFP{}).Probe(context.Background(), Local{Workspace: ws}, testLog()); err != nil {
		t.Fatalf("probing must not fail: %v", err)
	}

	inside, err := os.ReadDir(ws)
	if err != nil {
		t.Fatal(err)
	}
	if len(inside) != 0 {
		t.Errorf("the probe wrote into the workspace: %v", inside)
	}
	beside, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range beside {
		if e.Name() != "workspace" {
			t.Errorf("the probe left %q next to the workspace", e.Name())
		}
	}
}
