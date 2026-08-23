package build

import (
	"runtime"
	"strings"
	"testing"
)

// ★ 버전 한 줄에 무엇이 있어야 하는가 ★
//
// 자기 갱신이 이 줄을 파싱해 「받아온 것이 기대한 커밋인가」를 판정한다.
// 형태가 흔들리면 그 판정이 조용히 통과한다.
func Test버전줄에_필요한_것이_다_있다(t *testing.T) {
	old := Commit
	defer func() { Commit = old }()

	Commit = "3f9a1c8b2e07"
	got := Version("enode")
	for _, want := range []string{
		"enode",           // 어느 실행파일인가
		"3f9a1c8b2e07",    // ★ 어느 커밋인가 ★ — 갱신 검증이 이것을 본다
		runtime.GOOS,      // ★ 어느 기계용인가 ★ — darwin 바이너리를 리눅스에 넣는 사고를 막는다
		runtime.GOARCH,    //
		runtime.Version(), // 어느 툴체인인가
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("★ %q 가 버전 줄에 없다 ★: %s", want, got)
		}
	}
	// ★ 한 줄이어야 한다 ★ — 스크립트가 읽는다.
	if strings.Contains(got, "\n") {
		t.Fatalf("★ 여러 줄이다 ★: %q", got)
	}
}

// ★ 커밋을 안 박아도 무언가 나온다 ★ — 개발 빌드가 조용히 빈 줄을 내면
// 갱신 스크립트가 "받아온 것이 없다" 와 구별하지 못한다.
func Test커밋이_없어도_빈_줄이_아니다(t *testing.T) {
	old := Commit
	defer func() { Commit = old }()
	Commit = ""
	if got := Version("runctl"); strings.TrimSpace(got) == "" || !strings.Contains(got, "runctl") {
		t.Fatalf("★ 커밋이 없을 때 쓸모없는 줄이 나온다 ★: %q", got)
	}
}
