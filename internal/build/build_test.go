package build

import (
	"runtime"
	"strings"
	"testing"
)

// 버전 한 줄에 무엇이 있어야 하는가
//
// 자기 갱신이 이 줄을 파싱해 「받아온 것이 기대한 커밋인가」를 판정한다.
// 형태가 흔들리면 그 판정이 조용히 통과한다.
func TestVersionLineHasEverythingNeeded(t *testing.T) {
	old := Commit
	defer func() { Commit = old }()

	Commit = "3f9a1c8b2e07"
	got := Version("enode")
	for _, want := range []string{
		"enode",           // which binary
		"3f9a1c8b2e07",    // which commit — the update check reads this
		runtime.GOOS,      // which machine — keeps a darwin binary out of a linux box
		runtime.GOARCH,    //
		runtime.Version(), // which toolchain
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("%q is missing from the version line: %s", want, got)
		}
	}
	// 한 줄이어야 한다 — 스크립트가 읽는다.
	if strings.Contains(got, "\n") {
		t.Fatalf("more than one line: %q", got)
	}
}

// 커밋을 안 박아도 무언가 나온다 — 개발 빌드가 조용히 빈 줄을 내면
// 갱신 스크립트가 "받아온 것이 없다" 와 구별하지 못한다.
func TestNoCommitStillNotAnEmptyLine(t *testing.T) {
	old := Commit
	defer func() { Commit = old }()
	Commit = ""
	if got := Version("runctl"); strings.TrimSpace(got) == "" || !strings.Contains(got, "runctl") {
		t.Fatalf("useless line when the commit is absent: %q", got)
	}
}
