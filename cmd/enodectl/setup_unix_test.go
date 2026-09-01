//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 여기서 지키는 것은 setup 의 동작이 아니다 — 그것은
// internal/enode/setup_test.go 가 덮는다. 여기서 지키는 것은 enodectl 이 그
// 코드를 링크하지 않는다는 것과, 사람이 친 인자가 그대로 건너간다는 것이다.
//
// 갈래 파일인 이유는 가짜 실행 파일을 sh 스크립트로 두기 때문이다
// (cmd/iapadapter/orchestrator_unix_test.go 선례).
func TestCmdSetup_HandsOffToEnode(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "fake-enode")
	out := filepath.Join(dir, "args")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + out + "\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ENODE_BIN", bin)

	if err := cmdSetup([]string{"win-builder", "-check"}); err != nil {
		t.Fatalf("cmdSetup = %v, want nil", err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	// setup 이 맨 앞에 붙고 나머지는 순서대로 간다. 순서가 흐트러지면
	// 이름을 위치 인자로 받는 쪽이 플래그를 이름으로 읽는다.
	if want := "setup\nwin-builder\n-check\n"; string(got) != want {
		t.Errorf("enode got %q, want %q", got, want)
	}
}

func TestCmdSetup_NamesTheBinaryItCouldNotFind(t *testing.T) {
	// 없는 자리를 가리키면 무엇을 고쳐야 하는지 말해야 한다. 넘기는 구조는
	// 넘길 상대가 없을 때 조용히 실패하기 쉽다.
	missing := filepath.Join(t.TempDir(), "not-here")
	t.Setenv("ENODE_BIN", missing)
	err := cmdSetup(nil)
	if err == nil {
		t.Fatal("cmdSetup = nil, want an error naming the missing binary")
	}
	if !strings.Contains(err.Error(), "not-here") {
		t.Errorf("error = %v, want it to name the path", err)
	}
}
