package enode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPackagedExamplesParse 는 배포 묶음의 예시 설정이 살아 있는지를 잰다.
//
// 예시는 사람이 복사해서 쓰는 첫 파일이라 Local 이 바뀌면 가장 먼저 썩는다.
// 그런데 썩은 예시는 맥에서야 "설정 %s: yaml: ..." 로 나타나고, 그 시점의 사람은
// 자기 오타를 의심한다. 조용한 드리프트를 여기서 잡는다.
func TestPackagedExamplesParse(t *testing.T) {
	dir := filepath.Join("..", "..", "packaging", "macos", "examples")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("no packaging examples: %v", err)
	}

	var seen int
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		seen++
		path := filepath.Join(dir, e.Name())
		t.Run(e.Name(), func(t *testing.T) {
			l, err := LoadLocal(path)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			// 예시는 반드시 채워야 할 자리를 비워두면 안 된다 —
			// 비어 있으면 사람이 무엇을 채워야 하는지 모른다.
			if l.Mediator == "" {
				t.Error("mediator is empty")
			}
			if l.Token == "" {
				t.Error("token is empty (even a placeholder must be there)")
			}
			// MinFreeGB 는 LoadLocal 이 0 이면 10 으로 채운다 — 그 기본이 살아 있는지.
			if l.MinFreeGB <= 0 {
				t.Errorf("min_free_gb is %d", l.MinFreeGB)
			}
		})
	}
	if seen == 0 {
		t.Fatal("there is not a single example")
	}
}

// TestPackagedExamplesDeclareArchWhenToolchainIsUndetectable 은
// Zephyr 예시가 arch 를 손으로 적고 있는지를 잰다.
//
// detectArch() 는 arm-linux-gnueabihf-gcc · aarch64-linux-gnu-gcc 둘만 본다.
// Zephyr SDK(arm-zephyr-eabi-gcc)는 그 목록에 없으므로 설정이 메워야 한다.
// 나중에 detectArch 가 Zephyr 를 알게 되면 이 테스트가 그 사실을 알려준다.
func TestPackagedExamplesDeclareArchWhenToolchainIsUndetectable(t *testing.T) {
	for _, name := range []string{"zephyr.yaml", "qemu.yaml"} {
		path := filepath.Join("..", "..", "packaging", "macos", "examples", name)
		l, err := LoadLocal(path)
		if err != nil {
			t.Skipf("%s is missing: %v", name, err)
		}
		if l.Arch == "" {
			t.Errorf("%s: arch is empty. autodetection does not know the Zephyr toolchain, "+
				"so the config must write it (Local.Arch is the place)", name)
		}
	}
}
