//go:build unix

package scratch

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// SessionLockName 은 작업 폴더 안의 잠금 파일 이름이다.
const SessionLockName = ".enode-session.lock"

// orphanAge 는 잠금 파일이 없는 작업 폴더를 남은 것으로 치는 나이다. 이 유닛 전의 enode 가
// 남긴 폴더이고, MkdirTemp 와 잠금 사이의 짧은 창을 비킨다 (business-rules.md 3절).
const orphanAge = time.Hour

// SessionLock 은 작업 폴더를 만든 프로세스가 쥐는 잠금이다. 프로세스가 죽으면 커널이 푼다.
type SessionLock struct {
	f *os.File
}

// HoldSession 은 작업 폴더의 잠금을 쥔다 (flock · 배타 · 기다리지 않음).
//
// 잠금은 작업 폴더를 따라 trash 로 간다 — 파일이 아니라 열린 파일에 걸린다. 그래서 닫기는
// rename 뒤에 놓는다. fd 는 close-on-exec 라 helper 가 물려받지 않는다.
func HoldSession(runRoot string) (*SessionLock, error) {
	f, err := os.OpenFile(filepath.Join(runRoot, SessionLockName), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create session lock: %w", err)
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("hold session lock: %w", err)
	}
	return &SessionLock{f: f}, nil
}

// Release 는 잠금을 놓는다. 두 번 불러도 된다.
func (l *SessionLock) Release() error {
	if l == nil || l.f == nil {
		return nil
	}
	err := l.f.Close()
	l.f = nil
	return err
}

// Orphans 는 scratch 에서 쥔 프로세스가 없는 작업 폴더를 찾는다 (prefix 는 "enode-runc-").
//
//	잠금 파일이 있고 쥘 수 있다        남은 것이다.  쥔 뒤 곧바로 놓는다
//	잠금 파일이 있고 못 쥔다          살아 있다 — 다른 데몬의 단계나 도는 env check 의 smoke
//	잠금 파일이 없고 now 보다 1시간 전  남은 것이다 (이 유닛 전의 enode)
//	잠금 파일이 없고 1시간 안          skipped — 막 만든 폴더일 수 있다.  다음 기동이 다시 본다
//
// 나이는 폴더의 mtime 으로 잰다. now 는 인자다 — 시험이 시각을 준다.
func Orphans(scratchDir, prefix string, now time.Time) (orphans, skipped []string, err error) {
	ents, err := os.ReadDir(scratchDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	for _, e := range ents {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), prefix) {
			continue
		}
		dir := filepath.Join(scratchDir, e.Name())
		switch sessionState(dir, now) {
		case sessionOrphaned:
			orphans = append(orphans, dir)
		case sessionYoung:
			skipped = append(skipped, dir)
		}
	}
	return orphans, skipped, nil
}

type sessionKind int

const (
	sessionAlive sessionKind = iota
	sessionOrphaned
	sessionYoung
)

// sessionState 는 작업 폴더 하나를 본다. 모르는 오류는 살아 있는 쪽으로 친다 — 틀리면
// 도는 단계의 폴더를 옮기게 되고, 그 반대는 다음 기동이 다시 본다.
func sessionState(dir string, now time.Time) sessionKind {
	f, err := os.OpenFile(filepath.Join(dir, SessionLockName), os.O_RDWR, 0)
	if errors.Is(err, fs.ErrNotExist) {
		fi, err := os.Lstat(dir)
		if err != nil {
			return sessionAlive
		}
		if now.Sub(fi.ModTime()) >= orphanAge {
			return sessionOrphaned
		}
		return sessionYoung
	}
	if err != nil {
		return sessionAlive
	}
	defer f.Close()
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return sessionAlive
	}
	return sessionOrphaned
}
