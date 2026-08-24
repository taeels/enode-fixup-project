// Package build 는 ★ 이 실행파일이 무엇인가 ★ 를 담는다.
//
// ★ 왜 필요한가 ★ — 두 곳에서 막혔다.
//
//	★ 자기 갱신 ★   새 바이너리를 받아 두었을 때 ★ 그것이 새것인지 ★ 확인할
//	                방법이 없었다. 실행해 보는 것 말고는 없고, 실행하면
//	                노드가 하나 더 뜬다
//	★ 조사 ★        실측(vm-scratch-5)에서 계획이 `enode --version` 을 불렀다가
//	                "flag provided but not defined" 로 막혔다. ★ 표준 CLI 관례를
//	                어긴 것 ★ 이고, 도구가 자기를 설명하지 못한 것이다
//
// ★ 광고에는 안 싣는다 ★ — 매처는 동등 비교뿐이라 버전 문자열은 매칭에
// 못 쓰고 공간만 더럽힌다 (detect.go 가 하네스 버전에 대해 적은 것과 같다).
// 필요한 곳은 ★ 기록과 검증 ★ 이다.
package build

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

// Commit 은 ★ 링커가 박는다 ★ — packaging/macos/build.sh 의 -ldflags.
// 안 박으면 아래 fromVCS() 가 Go 의 VCS 스탬프에서 읽는다.
//
// Release 는 ★ 설치본이 붙는 자리 ★ 다 (deb·rpm·msi·pkg 는 전부 판 번호를
// 요구한다). ★ 비어 있으면 오늘 그대로다 ★ — 커밋이 곧 신원이라는 이 저장소의
// 선택은 안 바꾼다. 태그에서 지은 묶음만 이것을 채우고, 그때는 사람이
// "내가 1.2.3 을 깔았는데 --version 이 해시를 말한다" 로 헷갈리지 않는다.
var (
	Commit  = ""
	Date    = ""
	Release = ""
)

// Version 은 한 줄이다. `<cmd> --version` 이 이것을 찍는다.
//
//	enode 3f9a1c8b2e07 2026-08-23T09:12:00Z linux/amd64 go1.24.0
func Version(cmd string) string {
	c := Commit
	if c == "" {
		c = fromVCS()
	}
	if c == "" {
		c = "unknown"
	}
	d := Date
	if d == "" {
		d = "unknown"
	}
	if Release != "" {
		return fmt.Sprintf("%s %s (%s) %s %s/%s %s",
			cmd, Release, c, d, runtime.GOOS, runtime.GOARCH, runtime.Version())
	}
	return fmt.Sprintf("%s %s %s %s/%s %s",
		cmd, c, d, runtime.GOOS, runtime.GOARCH, runtime.Version())
}

// fromVCS 는 ★ go build 가 자동으로 박는 것 ★ 을 읽는다 — -ldflags 없이
// 빌드했을 때(개발 중)도 무언가 나오게 한다.
func fromVCS() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			if len(s.Value) > 12 {
				return s.Value[:12]
			}
			return s.Value
		}
	}
	return ""
}
