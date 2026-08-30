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

// 판 번호가 박힌 묶음은 판 번호와 커밋을 **둘 다** 든다.
//
// deb·rpm·msi·pkg 는 판 번호를 요구하므로 태그에서 지은 묶음만 Release 를
// 채운다. 그때 커밋을 떨어뜨리면 "내가 1.2.3 을 깔았는데 이게 어느 커밋인가"
// 를 되짚을 길이 없어지고, 판 번호를 안 실으면 사람이 자기가 깐 것을 못
// 알아본다. 그래서 둘 다여야 한다.
func TestReleaseLineCarriesBothTheReleaseAndTheCommit(t *testing.T) {
	oldC, oldR := Commit, Release
	defer func() { Commit, Release = oldC, oldR }()

	Commit = "3f9a1c8b2e07"
	Release = "1.2.3"
	got := Version("enode")

	for _, want := range []string{
		"enode",        // which binary
		"1.2.3",        // which release — this is what the package manager installed
		"3f9a1c8b2e07", // which commit — still recoverable from a released build
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("%q is missing from the release version line: %s", want, got)
		}
	}
	// 판 번호가 커밋보다 앞이다 — 사람이 먼저 찾는 것이 판 번호다.
	if strings.Index(got, "1.2.3") > strings.Index(got, "3f9a1c8b2e07") {
		t.Fatalf("the release must come before the commit, got: %s", got)
	}
	// 커밋은 괄호 안이다 — 판 번호와 눈으로 구분된다.
	if !strings.Contains(got, "(3f9a1c8b2e07)") {
		t.Fatalf("the commit is not parenthesised, so it reads as part of the release: %s", got)
	}
	if strings.Contains(got, "\n") {
		t.Fatalf("more than one line: %q", got)
	}
}

// 판 번호가 없으면 그 자리가 아예 없다 — 빈 괄호나 빈 칸을 남기지 않는다.
//
// 커밋이 곧 신원이라는 이 저장소의 선택은 Release 를 도입해도 안 바뀐다.
func TestNoReleaseLeavesNoEmptySlot(t *testing.T) {
	oldC, oldR := Commit, Release
	defer func() { Commit, Release = oldC, oldR }()

	Commit = "3f9a1c8b2e07"
	Release = ""
	got := Version("enode")

	if strings.Contains(got, "(") || strings.Contains(got, ")") {
		t.Fatalf("an empty release still left its parentheses behind: %s", got)
	}
	if strings.Contains(got, "  ") {
		t.Fatalf("an empty release left a double space: %q", got)
	}
	if !strings.Contains(got, "3f9a1c8b2e07") {
		t.Fatalf("the commit is missing, so the build has no identity at all: %s", got)
	}
}

// 커밋도 날짜도 안 박힌 개발 빌드가 빈 칸을 내지 않는다.
//
// 갱신 스크립트가 이 줄을 파싱한다 — 빈 자리는 "받아온 것이 없다" 와
// 구별되지 않는다.
func TestUnstampedBuildSaysUnknownRatherThanNothing(t *testing.T) {
	oldC, oldD, oldR := Commit, Date, Release
	defer func() { Commit, Date, Release = oldC, oldD, oldR }()

	Commit, Date, Release = "", "", ""
	got := Version("mediator")

	// fromVCS() 가 빈 문자열을 주는 개발 빌드에서도 자리가 "unknown" 으로 찬다.
	if !strings.Contains(got, "unknown") {
		t.Fatalf("an unstamped build must say unknown, not leave the slot empty: %q", got)
	}
	if strings.Contains(got, "  ") {
		t.Fatalf("an unstamped build left a double space: %q", got)
	}
	if !strings.HasPrefix(got, "mediator ") {
		t.Fatalf("the line must start with the command name, got: %q", got)
	}
}
