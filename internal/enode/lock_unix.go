//go:build !windows

package enode

import (
	"os"
	"syscall"
)

// flock 은 프로세스가 죽으면 커널이 자동으로 푼다 — 잠금이 새지 않는다.
func lockFile(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
}

func unlockFile(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
