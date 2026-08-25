//go:build !windows

package main

import "syscall"

// 프로세스를 다루는 네 가지만 플랫폼마다 다르다. 나머지는 전부 공통이다.
//
// 왜 가르나 — enodectl 이 syscall.Kill 과 SysProcAttr.Setsid 를 직접 부르는
// 바람에 윈도우로 빌드가 안 됐다. CI 의 cross 잡이 GOOS=windows 로
// `go build ./...` 를 돌려 그것을 막고 있었는데, 그 커밋들이 아직 푸시되지
// 않아 한 번도 안 돌았다 (실측 2026-08-24). ADR-015 가 Go 를 고른 이유 자체가
// "리눅스 CI 에서 윈도우 실행파일이 나온다" 이므로, 여기가 막히면 윈도우 배포
// 계획이 통째로 무너진다.

// processAlive 는 그 pid 가 아직 사는가다.
//
// 신호 0 은 보내지 않고 물어보기만 한다 — 권한이 있으면 nil 이다.
func processAlive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}

// detachAttr 는 부모에서 떼어낸다 — enodectl 이 끝나도 노드는 살아 있어야 한다.
func detachAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}

// signalStop 은 스스로 정리하고 끝나라다.
// enode 의 signal.NotifyContext 가 이것을 받아 임대를 놓고 나간다.
func signalStop(pid int) error {
	return syscall.Kill(pid, syscall.SIGTERM)
}

// signalKill 은 안 끝날 때의 마지막 수단이다.
func signalKill(pid int) error {
	return syscall.Kill(pid, syscall.SIGKILL)
}
