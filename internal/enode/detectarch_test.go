package enode

import (
	"os"
	"path/filepath"
	"testing"
)

// fakeToolchains 는 PATH 에 가짜 크로스 컴파일러를 놓는다.
func fakeToolchains(t *testing.T, names ...string) {
	t.Helper()
	dir := t.TempDir()
	for _, n := range names {
		p := filepath.Join(dir, n)
		if err := os.WriteFile(p, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}
	t.Setenv("PATH", dir)
}

// TestDetectArchsIsStableWhenBothToolchainsExist 는 맵 순회가 낳던 뒤집힘을 잰다.
//
// 고치기 전에는 둘 다 깔린 기계가 광고할 때마다 다른 arch 를 냈다. Go 의 맵
// 순회가 무작위이기 때문이고, 그러면 같은 계약이 하트비트마다 붙었다 떨어졌다
// 한다. 여러 번 불러서 같은 답이 나오는지를 본다.
func TestDetectArchsIsStableWhenBothToolchainsExist(t *testing.T) {
	fakeToolchains(t, "aarch64-linux-gnu-gcc", "arm-linux-gnueabihf-gcc")

	want := []string{"arm64", "armv7"}
	for i := 0; i < 50; i++ {
		got := detectArchs(Local{})
		if len(got) != len(want) {
			t.Fatalf("run %d: got %v, want %v", i, got, want)
		}
		for j := range want {
			if got[j] != want[j] {
				t.Fatalf("run %d: got %v, want %v", i, got, want)
			}
		}
	}
}

// TestDetectArchsReportsEveryToolchain 은 할 줄 아는 것을 다 말하는지를 잰다.
func TestDetectArchsReportsEveryToolchain(t *testing.T) {
	fakeToolchains(t, "arm-linux-gnueabihf-gcc")
	if got := detectArchs(Local{}); len(got) != 1 || got[0] != "armv7" {
		t.Errorf("one toolchain: got %v, want [armv7]", got)
	}

	fakeToolchains(t, "aarch64-linux-gnu-gcc", "arm-linux-gnueabihf-gcc")
	if got := detectArchs(Local{}); len(got) != 2 {
		t.Errorf("two toolchains: got %v, want both", got)
	}
}

// TestDetectArchsLetsTheConfigWin 은 Zephyr 처럼 탐지가 모르는 툴체인의 자리다.
func TestDetectArchsLetsTheConfigWin(t *testing.T) {
	fakeToolchains(t, "aarch64-linux-gnu-gcc")
	got := detectArchs(Local{Arch: "arm-zephyr-eabi"})
	if len(got) != 1 || got[0] != "arm-zephyr-eabi" {
		t.Errorf("got %v, want [arm-zephyr-eabi]", got)
	}
}

// TestCheapAttrsSpreadsArchIntoKeys 는 집합이 값이 아니라 키로 나가는지를 잰다.
func TestCheapAttrsSpreadsArchIntoKeys(t *testing.T) {
	fakeToolchains(t, "aarch64-linux-gnu-gcc", "arm-linux-gnueabihf-gcc")

	attrs := cheapAttrs(Local{}, testLog())
	if attrs["arch"] != "arm64" {
		t.Errorf("arch is %q, want arm64 (the first of the fixed order)", attrs["arch"])
	}
	for _, k := range []string{"arch.arm64", "arch.armv7"} {
		if attrs[k] != "yes" {
			t.Errorf("%s is %q, want yes", k, attrs[k])
		}
	}
}

// TestOrchestrationDropsEveryArchKey 는 광고를 좁히는 방어가 편 키에도
// 걸리는지를 잰다. 하나라도 남으면 빌드 계약이 그 키로 이 노드를 고른다.
func TestOrchestrationDropsEveryArchKey(t *testing.T) {
	fakeToolchains(t, "aarch64-linux-gnu-gcc", "arm-linux-gnueabihf-gcc")

	l := Local{Orchestration: true}
	log := testLog()
	caps := capabilities(l, log, map[string]string{"harness": "claude"}, cheapAttrs(l, log))
	if len(caps) == 0 {
		t.Fatal("an orchestration node with a harness must still advertise")
	}
	for _, c := range caps {
		for k := range c.Attrs {
			if k == "arch" || len(k) > len(archPrefix) && k[:len(archPrefix)] == archPrefix {
				t.Errorf("orchestration node still advertises %q", k)
			}
		}
	}
}
