//go:build windows

package enode

import (
	"os"

	"golang.org/x/sys/windows"
)

// lockOffset 은 잠금을 파일 내용 밖에 둔다.
//
// 왜 0 이 아닌가 — 윈도우의 LockFileEx 는 강제적 잠금이다. 잠긴 바이트
// 범위는 다른 프로세스가 쓰지 못할 뿐 아니라 읽지도 못한다. 그런데 이
// 파일의 앞부분에는 pid 가 적혀 있고, enodectl 이 그것을 읽어 노드가 도는지
// 판단한다(pidOf).
//
//	실측 (2026-09-06) 윈도우에서 enode.exe 가 멀쩡히 도는데 enodectl 이
//	그 노드를 stopped 로 봤다. start 는 "another enode is already running"
//	으로 거절하고 목록은 stopped 라고 하는, 서로 어긋나는 상태였다.
//	os.ReadFile 이 0번 바이트에서 ERROR_LOCK_VIOLATION 을 맞고 있었다.
//
// 유닉스의 flock 은 권고적이라 잠근 파일도 그냥 읽힌다. 그래서 리눅스와
// 맥에서는 한 번도 안 걸렸다 — child.go 의 콘솔 문제와 같은 성질이다.
//
// 파일 끝 너머를 잠근다. 윈도우는 존재하지 않는 범위에도 잠금을 걸 수
// 있고, 이것은 파일 잠금을 데이터와 분리하는 통상적인 방법이다.
// 1<<62 은 실제 내용과 절대 안 겹치면서 오버플로도 안 나는 자리다.
const lockOffset = int64(1) << 62

func overlappedAtLockOffset() *windows.Overlapped {
	return &windows.Overlapped{
		Offset:     uint32(lockOffset & 0xFFFFFFFF),
		OffsetHigh: uint32(lockOffset >> 32),
	}
}

// LockFileEx 도 핸들이 닫히면 커널이 푼다.
// ADR-015 가 윈도우 enode 를 전제하므로 여기가 비면 안 된다.
func lockFile(f *os.File) error {
	return windows.LockFileEx(windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0, 1, 0, overlappedAtLockOffset())
}

func unlockFile(f *os.File) error {
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0,
		overlappedAtLockOffset())
}
