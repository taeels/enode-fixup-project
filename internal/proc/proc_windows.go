//go:build windows

package proc

import "syscall"

const (
	// 살았는지 묻는 데 필요한 최소 권한.
	processQueryLimitedInformation = 0x1000
	// GetExitCodeProcess 가 「아직 돈다」로 주는 값.
	stillActive = 259
)

// ProcessAlive 는 그 pid 가 아직 사는가다.
//
// os.FindProcess 로는 못 잰다 — 윈도우에서는 없는 pid 에도 성공한다.
// 핸들을 열어 종료 코드를 물어야 한다.
func ProcessAlive(pid int) bool {
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

// SignalStop 은 윈도우에서 곱게 멈추라고 말할 방법이 없다.
//
// 진짜 이유 — SIGTERM 의 대응은 CTRL_BREAK 인데, 그것은 콘솔을 통해서만
// 간다. 그런데 노드는 DETACHED_PROCESS 로 콘솔을 끊고 떠서(cmd/enodectl 의
// detachAttr) 그 길이 없다. 강제 종료로 잃는 것은 「즉시 반납」뿐이고
// 자원이 영영 묶이지는 않는다 — 임대는 하트비트가 끊기면 만료된다
// (ADR-008 · ADR-016).
func SignalStop(pid int) error {
	return SignalKill(pid)
}

// SignalKill 은 강제 종료다.
func SignalKill(pid int) error {
	h, err := syscall.OpenProcess(syscall.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return err
	}
	defer syscall.CloseHandle(h)
	return syscall.TerminateProcess(h, 1)
}

// OwnsConfig 는 그 pid 가 이 설정을 열고 있는지다.
//
// 유닉스판은 ps 로 명령줄을 읽어 설정 경로까지 맞춰 보지만, 윈도우에는 ps 가
// 없다. 그래서 여기서는 살아 있는지까지만 본다. 명령줄을 못 보므로 pid 가
// 재사용되면 틀릴 수 있으나, 중복 실행을 실제로 막는 것은 enode 의 flock
// 이므로 여기서 틀려도 두 노드가 서는 일은 없다.
func OwnsConfig(pid int, conf string) bool {
	_ = conf
	return ProcessAlive(pid)
}
