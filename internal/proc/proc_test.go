package proc

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestProcessAliveSelf(t *testing.T) {
	if !ProcessAlive(os.Getpid()) {
		t.Fatal("the test process should be alive")
	}
}

func TestProcessAliveDead(t *testing.T) {
	// a pid this large is almost certainly not a live process on either platform
	if ProcessAlive(1 << 30) {
		t.Fatal("a huge pid should not be reported alive")
	}
}

func TestDetachAttr(t *testing.T) {
	if DetachAttr() == nil {
		t.Fatal("DetachAttr must return a non-nil SysProcAttr")
	}
}

func TestPidFromLockMissing(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "ws-a.yaml")
	if got := PidFromLock(cfg); got != 0 {
		t.Fatalf("no lock file should give 0, got %d", got)
	}
}

func TestPidFromLockGarbage(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "ws-a.yaml")
	if err := os.WriteFile(cfg+".lock", []byte("not-a-number\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := PidFromLock(cfg); got != 0 {
		t.Fatalf("garbage lock should give 0, got %d", got)
	}
}

func TestPidFromLockDeadPid(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "ws-a.yaml")
	// a parseable but dead pid: OwnsConfig is reached and returns false
	if err := os.WriteFile(cfg+".lock", []byte(strconv.Itoa(1<<30)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := PidFromLock(cfg); got != 0 {
		t.Fatalf("dead pid should give 0, got %d", got)
	}
}
