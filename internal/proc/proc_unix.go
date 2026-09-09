//go:build !windows

// Package proc 는 한 기계의 프로세스를 다루는 플랫폼 짝이다.
//
// net/http 를 안 쓴다 — 그것이 이 패키지를 가르는 이유다. cmd/enodectl 과
// internal/panel 이 이 셋을 같이 쓰는데, panel 은 net/http 서버라 이 함수들을
// panel 에 두면 cmd/enodectl 의 링크 그래프가 panel 을 거쳐 net/http 에 닿아
// enodectl.exe 의 심볼 상한(ci.yml:479)이 깨진다. 그래서 net/http 를 안 쓰는
// 이 패키지로 내려 둘이 같이 쓴다 (ADR-068 이 아니라 decisions §2 「제어판 서버 위치」).
package proc

import (
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// ProcessAlive 는 그 pid 가 아직 사는가다.
//
// 신호 0 은 보내지 않고 물어보기만 한다 — 권한이 있으면 nil 이다.
func ProcessAlive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}

// SignalStop 은 스스로 정리하고 끝나라다.
// enode 의 signal.NotifyContext 가 이것을 받아 임대를 놓고 나간다.
func SignalStop(pid int) error {
	return syscall.Kill(pid, syscall.SIGTERM)
}

// SignalKill 은 안 끝날 때의 마지막 수단이다.
func SignalKill(pid int) error {
	return syscall.Kill(pid, syscall.SIGKILL)
}

// OwnsConfig 는 그 pid 가 이 설정을 열고 있는지다.
//
// 이름이 아니라 명령줄을 본다 — pid 는 재사용되고, 실행파일 이름은 배포
// 방식에 따라 다르다(설치본은 enode, 묶음에서 바로 돌리면
// enode-linux-amd64). 우리가 묻는 것은 이 설정을 열고 있는가다.
func OwnsConfig(pid int, conf string) bool {
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "args=").Output()
	return err == nil && strings.Contains(string(out), conf)
}
