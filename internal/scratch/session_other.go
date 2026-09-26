//go:build !unix

package scratch

import "time"

// SessionLockName 은 작업 폴더 안의 잠금 파일 이름이다.
const SessionLockName = ".enode-session.lock"

// SessionLock 은 이 OS 에서 할 일이 없다 — runc-overlay 세션이 linux 에만 있다.
type SessionLock struct{}

// HoldSession 은 할 일이 없는 잠금을 돌려준다.
func HoldSession(string) (*SessionLock, error) { return &SessionLock{}, nil }

// Release 는 할 일이 없다.
func (*SessionLock) Release() error { return nil }

// Orphans 는 빈 목록이다 — 남을 작업 폴더가 없다.
func Orphans(string, string, time.Time) (orphans, skipped []string, err error) {
	return nil, nil, nil
}
