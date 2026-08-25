//go:build windows

package main

import "syscall"

// 윈도우판. 셋은 제대로 되고 하나는 못 한다 — 아래에 진짜 이유를 적는다.

const (
	// 프로세스 생성 플래그 (Win32 procthread).
	createNewProcessGroup = 0x00000200 // CREATE_NEW_PROCESS_GROUP
	detachedProcess       = 0x00000008 // DETACHED_PROCESS
	// GetExitCodeProcess 가 「아직 돈다」로 주는 값.
	stillActive = 259
	// 살았는지 묻는 데 필요한 최소 권한.
	processQueryLimitedInformation = 0x1000
)

// processAlive 는 그 pid 가 아직 사는가다.
//
// os.FindProcess 로는 못 잰다 — 윈도우에서는 없는 pid 에도 성공한다.
// 핸들을 열어 종료 코드를 물어야 한다.
func processAlive(pid int) bool {
	h, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(h)
	var code uint32
	if err := syscall.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	return code == stillActive
}

// detachAttr 는 부모에서 떼어낸다 — Setsid 의 윈도우 대응이다.
// DETACHED_PROCESS 로 콘솔을 끊고, 새 프로세스 그룹으로 부모의 Ctrl 신호를 피한다.
func detachAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: createNewProcessGroup | detachedProcess}
}

// signalStop 은 윈도우에서 곱게 멈추라고 말할 방법이 없다.
//
// 진짜 이유 — SIGTERM 의 대응은 CTRL_BREAK 인데, 그것은 콘솔을 통해서만
// 간다. 그런데 detachAttr 가 DETACHED_PROCESS 로 콘솔을 끊어 놨다 —
// 끊지 않으면 enodectl 의 콘솔이 닫힐 때 노드가 함께 죽는다. 둘 다는 안 된다.
// 떼어내는 쪽을 골랐다: 노드는 오래 살아야 하는 것이고, 곱게 멈추는 것은
// 대체 수단이 있다 — 임대는 하트비트가 끊기면 만료되므로(ADR-008 · ADR-016)
// 강제 종료로 잃는 것은 「즉시 반납」뿐이고 자원이 영영 묶이지는 않는다.
//
// 되돌리는 조건 — 윈도우 노드가 실물로 서고, 강제 종료로 놓친 임대가
// 실제로 관측되면 그때 다시 본다. 그때의 답은 아마 서비스로 등록하는 것이다
// (SCM 이 정지 신호를 제대로 보낸다).
func signalStop(pid int) error {
	return signalKill(pid)
}

// signalKill 은 강제 종료다.
func signalKill(pid int) error {
	h, err := syscall.OpenProcess(syscall.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return err
	}
	defer syscall.CloseHandle(h)
	return syscall.TerminateProcess(h, 1)
}
