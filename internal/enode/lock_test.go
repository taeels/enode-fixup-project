package enode

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// 잠금이 pid 를 가리면 안 된다.
//
// enodectl 은 노드가 도는지를 이 파일의 첫 줄로 판단한다(pidOf). 그래서
// 잠금과 pid 가 같은 바이트를 쓰면 「잠갔다」가 「안 돈다」로 읽힌다.
//
//	실측 (2026-09-06) 윈도우에서 enode.exe 가 멀쩡히 도는데 enodectl 이
//	그 노드를 stopped 로 봤다. start 는 "another enode is already running"
//	으로 거절하고 목록은 stopped 라고 하는, 서로 어긋나는 상태였다.
//	LockFileEx 가 강제적 잠금이라 os.ReadFile 이 0번 바이트에서
//	ERROR_LOCK_VIOLATION 을 맞고 있었다.
//
// 이 시험은 리눅스와 맥에서 그냥 통과한다 — flock 은 권고적이라 잠근 파일도
// 읽힌다. 그래도 여기 두는 이유는 의도를 코드로 남기기 위해서다.
// 윈도우에서 도는 순간 이것이 그 자리를 잡는다.
func TestTheLockDoesNotHideThePid(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "local.yaml")
	if err := os.WriteFile(cfg, []byte("mediator: x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	lock, err := Acquire(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release() //nolint:errcheck

	// enodectl 의 pidOf 가 하는 일을 그대로 한다.
	b, err := os.ReadFile(cfg + ".lock")
	if err != nil {
		t.Fatalf("the lock file cannot be read while held, so enodectl sees a "+
			"running node as stopped: %v", err)
	}
	line := strings.TrimSpace(strings.SplitN(string(b), "\n", 2)[0])
	pid, err := strconv.Atoi(line)
	if err != nil {
		t.Fatalf("the first line is not a pid: %q", line)
	}
	if pid != os.Getpid() {
		t.Fatalf("the lock names pid %d, but this process is %d", pid, os.Getpid())
	}
}
