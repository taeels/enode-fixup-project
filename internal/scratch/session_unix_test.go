//go:build unix

package scratch

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestHoldSessionLocksUntilRelease(t *testing.T) {
	run := t.TempDir()
	lock, err := HoldSession(run)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := HoldSession(run); err == nil || !strings.Contains(err.Error(), "hold session lock") {
		t.Fatalf("a second holder got the lock: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("release is not idempotent: %v", err)
	}
	again, err := HoldSession(run)
	if err != nil {
		t.Fatalf("the lock was not released: %v", err)
	}
	_ = again.Release()
	if err := (*SessionLock)(nil).Release(); err != nil {
		t.Fatal(err)
	}
	if _, err := HoldSession(filepath.Join(run, "missing")); err == nil || !strings.Contains(err.Error(), "create session lock") {
		t.Fatalf("err = %v", err)
	}
}

// 잠금을 쥔 폴더는 안 나온다 · 풀린 잠금은 나온다 · 잠금 파일이 없으면 1시간 규칙이다.
// flock 은 열린 파일마다이므로 같은 프로세스의 다른 fd 로 「다른 프로세스가 쥐었다」를 만든다.
func TestOrphansFindsSessionsNobodyHolds(t *testing.T) {
	scratch := t.TempDir()
	now := time.Now()
	held := filepath.Join(scratch, "enode-runc-held")
	released := filepath.Join(scratch, "enode-runc-released")
	oldBare := filepath.Join(scratch, "enode-runc-old")
	youngBare := filepath.Join(scratch, "enode-runc-young")
	other := filepath.Join(scratch, "enode-smoke-io-1")
	for _, d := range []string{held, released, oldBare, youngBare, other, filepath.Join(scratch, "trash")} {
		mkdir(t, d)
	}
	write(t, filepath.Join(scratch, "enode-runc-file"), "not a folder")
	lock, err := HoldSession(held)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = lock.Release() }()
	gone, err := HoldSession(released)
	if err != nil {
		t.Fatal(err)
	}
	_ = gone.Release()
	past := now.Add(-2 * time.Hour)
	if err := os.Chtimes(oldBare, past, past); err != nil {
		t.Fatal(err)
	}
	orphans, skipped, err := Orphans(scratch, "enode-runc-", now)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{oldBare, released}; !reflect.DeepEqual(orphans, want) {
		t.Fatalf("orphans = %v, want %v", orphans, want)
	}
	if want := []string{youngBare}; !reflect.DeepEqual(skipped, want) {
		t.Fatalf("skipped = %v, want %v", skipped, want)
	}
	// 옮겨 보기 전의 판정이 잠금을 쥔 채로 남지 않는다 — 판정한 뒤에도 다시 쥘 수 있다
	again, err := HoldSession(released)
	if err != nil {
		t.Fatalf("the judgment kept the lock: %v", err)
	}
	_ = again.Release()
	// 1시간이 지나면 잠금 파일이 없는 폴더도 남은 것이다
	orphans, _, _ = Orphans(scratch, "enode-runc-", now.Add(2*time.Hour))
	if len(orphans) != 3 {
		t.Fatalf("orphans an hour later = %v", orphans)
	}
}

func TestOrphansOfAMissingScratchIsEmpty(t *testing.T) {
	orphans, skipped, err := Orphans(filepath.Join(t.TempDir(), "missing"), "enode-runc-", time.Now())
	if err != nil || orphans != nil || skipped != nil {
		t.Fatalf("orphans = %v skipped = %v err = %v", orphans, skipped, err)
	}
	file := filepath.Join(t.TempDir(), "f")
	write(t, file, "x")
	if _, _, err := Orphans(file, "enode-runc-", time.Now()); err == nil {
		t.Fatal("read a scratch that is a file")
	}
}

// 잠금 파일을 못 여는 폴더는 살아 있는 쪽으로 친다 — 틀리면 도는 단계를 옮긴다.
func TestOrphansLeavesWhatItCannotJudge(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root opens any lock file")
	}
	scratch := t.TempDir()
	run := filepath.Join(scratch, "enode-runc-closed")
	mkdir(t, run)
	write(t, filepath.Join(run, SessionLockName), "")
	if err := os.Chmod(filepath.Join(run, SessionLockName), 0o000); err != nil {
		t.Fatal(err)
	}
	orphans, skipped, err := Orphans(scratch, "enode-runc-", time.Now())
	if err != nil || len(orphans) != 0 || len(skipped) != 0 {
		t.Fatalf("orphans = %v skipped = %v err = %v", orphans, skipped, err)
	}
	if got := sessionState(filepath.Join(scratch, "missing"), time.Now()); got != sessionAlive {
		t.Fatalf("a vanished folder = %v", got)
	}
}
