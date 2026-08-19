//go:build windows

package enode

import (
	"os"

	"golang.org/x/sys/windows"
)

// LockFileEx 도 핸들이 닫히면 커널이 푼다.
// ADR-015 가 윈도우 enode 를 전제하므로 여기가 비면 안 된다.
func lockFile(f *os.File) error {
	ol := new(windows.Overlapped)
	return windows.LockFileEx(windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0, 1, 0, ol)
}

func unlockFile(f *os.File) error {
	ol := new(windows.Overlapped)
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, ol)
}
