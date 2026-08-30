//go:build !windows

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// 노드의 수명주기 - 무엇이 도는가, 어떻게 띄우고 멈추는가, 로그를 어떻게
// 보는가. main_test.go 가 보는 자리·이름·신원과 달리 이쪽은 전부 유닉스의
// 물건에 매여 있다: ps 로 명령줄을 읽고, SIGTERM 으로 곱게 멈추라 말하고,
// setsid 로 부모에서 떼어낸다.
//
// 왜 빌드 태그로 가르는가 - 이 저장소는 OS 분기를 빌드 태그 쌍으로만 한다
// (lock_unix.go/lock_windows.go · proc_unix.go/proc_windows.go). 런타임에
// t.Skip 으로 가르는 안을 기각했다: CI 의 스킵 감시가 전 패키지를 보고,
// 스킵은 "테스트가 돌아 통과했다" 와 "CI 가 초록이다" 를 갈라 놓는다.
// 짝이 되는 _windows 파일이 없는 것은 짝이 필요 없어서다 - 컴파일이
// 요구하는 것은 생산 코드의 짝이지 테스트의 짝이 아니다.
//
// 목을 세우지 않는다. 경계가 셋뿐이고 셋 다 실물로 선다 - 파일 시스템은
// t.TempDir(), 프로세스는 진짜 자식, HTTP 는 httptest.

// reap 은 그 프로세스와 그 아래를 통째로 거둔다.
//
// 셸만 죽이는 것으로는 모자라다 - `sleep 60` 은 별개 프로세스라 부모가
// 죽어도 최대 1분을 더 산다. 그 유령이 남으면 caffeinated 처럼 기계 전체의
// 프로세스 목록을 훑는 함수가 앞선 실행의 잔재를 보고, 그러면 이 파일의
// 테스트가 서로의 결과를 바꾼다. 그래서 자식에게 자기 프로세스 그룹을
// 주고(Setpgid) 그룹째 죽인다.
func reap(t *testing.T, c *exec.Cmd) {
	t.Helper()
	if c.Process == nil {
		return
	}
	// 음수 pid 는 프로세스 그룹이다. Setpgid 로 띄웠으므로 그룹 id 가 곧 pid 다.
	_ = syscall.Kill(-c.Process.Pid, syscall.SIGKILL)
	_ = c.Process.Kill()
	_, _ = c.Process.Wait()
}

// holdOpen 은 표시 문자열을 자기 명령줄에 들고 사는 프로세스를 하나 띄운다.
//
// pidOf 가 보는 것은 잠금 파일의 존재가 아니라 "그 pid 가 살아 있고 이
// 설정을 열고 있는가" 다 - enode 는 끝나도 잠금 파일을 안 지우기 때문이다
// (flock 은 커널이 푼다). 그래서 가짜 pid 로는 이 함수를 시험할 수 없고,
// 실제로 그 경로를 argv 에 들고 있는 프로세스가 있어야 한다.
//
// `sleep 60; exit 0` 처럼 명령을 둘로 두는 것과 $0 자리에 표시를 주는 것이
// 둘 다 의도다 - 셸이 단일 명령을 exec 로 갈아치우면 argv 에서 표시가
// 사라진다. 아래에서 ps 로 실제로 보이는지 확인하고 나서 돌려준다.
func holdOpen(t *testing.T, marker string, ignoreTerm bool) int {
	t.Helper()
	body := "sleep 60; exit 0"
	if ignoreTerm {
		body = "trap '' TERM; " + body
	}
	c := exec.Command("/bin/sh", "-c", body, marker)
	// 자기 프로세스 그룹을 준다 - reap 이 그 아래까지 거둘 수 있게.
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := c.Start(); err != nil {
		t.Fatalf("cannot spawn a child holding %q: %v", marker, err)
	}
	pid := c.Process.Pid
	t.Cleanup(func() { reap(t, c) })
	deadline := time.Now().Add(5 * time.Second)
	for {
		out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "args=").Output()
		if err == nil && strings.Contains(string(out), marker) {
			return pid
		}
		if time.Now().After(deadline) {
			t.Fatalf("pid %d never showed %q in its command line (ps said %q); "+
				"the shell replaced itself and the marker was lost", pid, marker, out)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// lockFor 는 <conf>.lock 에 pid 를 적는다. enode 가 뜨면서 하는 일과 같다.
//
// flock 은 흉내 내지 않는다 - pidOf 가 읽는 것은 파일의 내용이고, 잠금까지
// 흉내 내면 이 테스트가 커널을 시험하게 된다.
func lockFor(t *testing.T, name string, pid int) {
	t.Helper()
	if err := os.WriteFile(confOf(name)+".lock", []byte(strconv.Itoa(pid)+"\n"), 0o600); err != nil {
		t.Fatalf("write the lock file for %s: %v", name, err)
	}
}

// pathWith 는 이름을 든 도구들만 실제로 찾을 수 있는 PATH 를 만든다.
//
// 왜 진짜 PATH 를 안 쓰는가 - "claude 가 PATH 에 없으면 경고한다" 를 시험
// 하려는데, 이 기계에 claude 가 깔려 있으면 그 경로가 영영 안 돌고 없으면
// 반대쪽이 안 돈다. 즉 판정이 개발자의 기계에 달린다. 필요한 것만 골라
// 담은 디렉터리 하나로 두 경우를 이 파일이 결정한다.
func pathWith(t *testing.T, tools ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range tools {
		real, err := exec.LookPath(name)
		if err != nil {
			t.Fatalf("this machine has no %s; the enodectl tests need it: %v", name, err)
		}
		if err := os.Symlink(real, filepath.Join(dir, name)); err != nil {
			t.Fatalf("link %s into the test PATH: %v", name, err)
		}
	}
	t.Setenv("PATH", dir)
	return dir
}

// stubIn 은 자기가 어떻게 불렸는지를 적고 끝나는 실행파일을 만든다.
func stubIn(t *testing.T, dir, name string) string {
	t.Helper()
	record := filepath.Join(t.TempDir(), name+".argv")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" > " + record + "\nexit 0\n"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
		t.Fatalf("write the %s stub: %v", name, err)
	}
	return record
}

// fakeEnode 는 enode 인 척하는 실행파일이다.
//
// cmdStart 가 "떴다" 로 판정하는 조건이 pidOf 의 조건과 같다 - 잠금 파일에
// pid 가 있고 그 pid 가 살아서 이 설정을 열고 있어야 한다. 그래서 가짜도
// 그 둘을 실제로 한다: 자기 pid 를 <conf>.lock 에 쓰고 산다. 자기 argv 에
// --config <경로> 가 이미 들어 있으므로 ps 쪽 조건은 저절로 맞는다.
func fakeEnode(t *testing.T, comesUp bool) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "enode")
	body := "#!/bin/sh\n" +
		"echo \"fake enode: $*\"\n"
	if comesUp {
		body += "echo $$ > \"$2.lock\"\nsleep 60\nexit 0\n"
	} else {
		body += "echo 'config rejected: mediator is required' >&2\nexit 1\n"
	}
	if err := os.WriteFile(bin, []byte(body), 0o755); err != nil {
		t.Fatalf("write the fake enode: %v", err)
	}
	t.Setenv("ENODE_BIN", bin)
	return bin
}

// reapNode 는 잠금 파일이 가리키는 노드를 그 아래까지 거둔다.
//
// pidOf 를 안 쓰고 잠금 파일을 직접 읽는 이유 - 셸이 이미 멈춘 뒤에도 그
// 아래 sleep 이 살아 있는데, 그때 pidOf 는 0 을 돌려주므로 정확히 치워야
// 할 순간에 아무것도 안 치운다. cmdStart 가 노드를 setsid 로 떼어내므로
// 그룹 id 가 곧 그 pid 이고, 음수로 보내면 그룹 전체에 닿는다.
func reapNode(t *testing.T, name string) {
	t.Helper()
	b, err := os.ReadFile(confOf(name) + ".lock")
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(strings.SplitN(string(b), "\n", 2)[0]))
	if err != nil || pid <= 0 {
		return
	}
	_ = syscall.Kill(-pid, syscall.SIGKILL)
	_ = syscall.Kill(pid, syscall.SIGKILL)
}

// ── 도는가 ───────────────────────────────────────────────────────────────

func TestPidOf_AsksWhetherThatProcessIsHoldingThisConfig(t *testing.T) {
	isolate(t)
	writeConfig(t, "zephyr")
	conf := confOf("zephyr")

	if got := pidOf("zephyr"); got != 0 {
		t.Fatalf("pidOf with no lock file = %d, want 0", got)
	}

	for _, bad := range []string{"", "not-a-number\n", "0\n", "-3\n"} {
		if err := os.WriteFile(conf+".lock", []byte(bad), 0o600); err != nil {
			t.Fatalf("write the lock file: %v", err)
		}
		if got := pidOf("zephyr"); got != 0 {
			t.Fatalf("pidOf with lock contents %q = %d, want 0", bad, got)
		}
	}

	// 살아 있고 이 설정을 열고 있다.
	pid := holdOpen(t, conf, false)
	lockFor(t, "zephyr", pid)
	if got := pidOf("zephyr"); got != pid {
		t.Fatalf("pidOf = %d, want %d - the process is alive and holds this config", got, pid)
	}

	// pid 는 재사용된다. 이름이 아니라 명령줄을 보는 이유가 이것이다 -
	// 그 자리에 다른 프로그램이 들어와 있으면 우리 노드가 아니다.
	other := holdOpen(t, "some-other-program", false)
	lockFor(t, "zephyr", other)
	if got := pidOf("zephyr"); got != 0 {
		t.Fatalf("pidOf = %d, want 0 - that pid is alive but holds a different config", got)
	}
}

func TestPidOf_ForgetsADeadNodeEvenThoughTheLockFileRemains(t *testing.T) {
	// enode 는 끝나도 잠금 파일을 안 지운다 - flock 은 커널이 푼다.
	// 파일의 존재만으로 "도는 중" 이라고 읽으면 죽은 노드가 영영 도는
	// 것으로 보인다.
	isolate(t)
	writeConfig(t, "gone")
	c := exec.Command("/bin/sh", "-c", "exit 0", confOf("gone"))
	if err := c.Start(); err != nil {
		t.Fatalf("spawn: %v", err)
	}
	pid := c.Process.Pid
	if err := c.Wait(); err != nil {
		t.Fatalf("wait: %v", err)
	}
	lockFor(t, "gone", pid)
	if got := pidOf("gone"); got != 0 {
		t.Fatalf("pidOf = %d, want 0 - the lock file outlives the process", got)
	}
}

// ── caffeinate 가 붙어 있는가 ────────────────────────────────────────────

func TestCaffeinated_LooksForTheHolderOfThatPid(t *testing.T) {
	if got := caffeinated(424242); got {
		t.Fatal("caffeinated on a pid nothing holds = true, want false")
	}

	// argv[0] 을 caffeinate 로 두고 -w <pid> 를 들려 띄운다. caffeinated 가
	// 세는 것이 정확히 그 모양이다.
	held := 4242
	c := &exec.Cmd{
		Path: "/bin/sh",
		Args: []string{
			"caffeinate", "-c", "sleep 60; exit 0", "held", "-i", "-s", "-w", strconv.Itoa(held)},
		SysProcAttr: &syscall.SysProcAttr{Setpgid: true},
	}
	if err := c.Start(); err != nil {
		t.Fatalf("spawn a fake caffeinate: %v", err)
	}
	t.Cleanup(func() { reap(t, c) })
	deadline := time.Now().Add(5 * time.Second)
	for !caffeinated(held) {
		if time.Now().After(deadline) {
			t.Fatalf("caffeinated(%d) stayed false while a caffeinate holding it was running", held)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if caffeinated(held + 1) {
		t.Fatalf("caffeinated(%d) = true, want false - it must match the held pid, not any caffeinate", held+1)
	}
}

func TestCaffeinated_SaysNoWhenItCannotAsk(t *testing.T) {
	// ps 가 없으면 "안 붙어 있다" 로 답한다. 여기서 죽으면 list 가 통째로
	// 안 나온다 - 잠자기 방지는 부가 정보이지 목록의 전제가 아니다.
	t.Setenv("PATH", t.TempDir())
	if caffeinated(1) {
		t.Fatal("caffeinated with no ps on PATH = true, want false")
	}
}

// ── enode 를 어디서 찾는가 ───────────────────────────────────────────────

func TestEnodeBin_PrefersTheEnvironmentThenPathThenTheDefault(t *testing.T) {
	dir := pathWith(t)
	want := filepath.Join(dir, "enode")
	if err := os.WriteFile(want, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write a fake enode on PATH: %v", err)
	}

	t.Setenv("ENODE_BIN", "/opt/enode/bin/enode")
	if got := enodeBin(); got != "/opt/enode/bin/enode" {
		t.Fatalf("enodeBin() = %q, want ENODE_BIN to win", got)
	}

	t.Setenv("ENODE_BIN", "")
	if got := enodeBin(); got != want {
		t.Fatalf("enodeBin() = %q, want the one found on PATH (%q)", got, want)
	}

	// PATH 에도 없으면 설치 기본 자리를 말한다 - "못 찾겠다" 로 끝내면
	// cmdStart 의 오류가 무엇을 깔아야 하는지 안 알려준다.
	if err := os.Remove(want); err != nil {
		t.Fatalf("remove the fake enode: %v", err)
	}
	got := enodeBin()
	if !strings.HasSuffix(got, filepath.Join(".local", "bin", "enode")) {
		t.Fatalf("enodeBin() = %q, want the ~/.local/bin default", got)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("enodeBin() = %q, want an absolute path", got)
	}
}

// ── 띄우기 ───────────────────────────────────────────────────────────────

func TestCmdStart_RefusesBeforeItSpawnsAnything(t *testing.T) {
	isolate(t)
	writeConfig(t, "zephyr")
	gitIdentity(t, "node-test@example.invalid")
	pathWith(t, "ps", "sleep", "git")

	t.Run("a name is required", func(t *testing.T) {
		if err := cmdStart(nil); err == nil {
			t.Fatal("cmdStart(nil) = nil, want an error asking for a name")
		}
	})

	t.Run("the binary is missing", func(t *testing.T) {
		t.Setenv("ENODE_BIN", filepath.Join(t.TempDir(), "absent"))
		err := cmdStart([]string{"zephyr"})
		if err == nil || !strings.Contains(err.Error(), "enode binary not found") {
			t.Fatalf("cmdStart = %v, want it to name the binary it could not find", err)
		}
	})

	t.Run("the binary is a directory", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "enode")
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		t.Setenv("ENODE_BIN", dir)
		err := cmdStart([]string{"zephyr"})
		if err == nil || !strings.Contains(err.Error(), "enode binary not found") {
			t.Fatalf("cmdStart = %v, want a directory to be refused like a missing binary", err)
		}
	})

	t.Run("it is already running", func(t *testing.T) {
		fakeEnode(t, true)
		pid := holdOpen(t, confOf("zephyr"), false)
		lockFor(t, "zephyr", pid)
		t.Cleanup(func() { _ = os.Remove(confOf("zephyr") + ".lock") })
		err := cmdStart([]string{"zephyr"})
		if err == nil || !strings.Contains(err.Error(), "already running") {
			t.Fatalf("cmdStart = %v, want it refused because a node already holds this config", err)
		}
		// enode 의 flock 이 두 번째를 거절한다는 사실이 오류문에 적혀 있어야
		// 사람이 "그럼 왜 안 뜨나" 를 다시 묻지 않는다.
		if !strings.Contains(err.Error(), strconv.Itoa(pid)) {
			t.Fatalf("cmdStart error = %q, want the pid that holds it", err)
		}
	})

	t.Run("the state directory cannot be made", func(t *testing.T) {
		blocker := filepath.Join(t.TempDir(), "not-a-directory")
		if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
			t.Fatalf("write the blocker: %v", err)
		}
		t.Setenv("ENODE_STATEDIR", filepath.Join(blocker, "state"))
		fakeEnode(t, true)
		if err := cmdStart([]string{"zephyr"}); err == nil {
			t.Fatal("cmdStart = nil, want the mkdir failure surfaced")
		}
	})

	t.Run("the log file cannot be opened", func(t *testing.T) {
		state := t.TempDir()
		t.Setenv("ENODE_STATEDIR", state)
		if err := os.Mkdir(filepath.Join(state, "zephyr.log"), 0o755); err != nil {
			t.Fatalf("mkdir the blocking log: %v", err)
		}
		fakeEnode(t, true)
		if err := cmdStart([]string{"zephyr"}); err == nil {
			t.Fatal("cmdStart = nil, want the log open failure surfaced")
		}
	})

	t.Run("the binary cannot be executed", func(t *testing.T) {
		bin := filepath.Join(t.TempDir(), "enode")
		// 셸이 없는 인터프리터를 가리킨다 - 실행 권한과 무관하게 execve 가
		// 실패하므로 러너가 root 여도 같은 결과가 나온다.
		if err := os.WriteFile(bin, []byte("#!/nonexistent/interpreter\n"), 0o755); err != nil {
			t.Fatalf("write the unrunnable binary: %v", err)
		}
		t.Setenv("ENODE_BIN", bin)
		if err := cmdStart([]string{"zephyr"}); err == nil {
			t.Fatal("cmdStart = nil, want the exec failure surfaced")
		}
	})
}

func TestCmdStart_BringsTheNodeUpAndSaysWhereItsLogIs(t *testing.T) {
	isolate(t)
	writeConfig(t, "zephyr")
	gitIdentity(t, "node-test@example.invalid")
	dir := pathWith(t, "ps", "sleep", "git")
	// claude 가 PATH 에 있으면 경고가 안 나와야 한다.
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write the claude stub: %v", err)
	}
	fakeEnode(t, true)
	t.Cleanup(func() { reapNode(t, "zephyr") })

	stdout, stderr := captureOutput(t, func() {
		if err := cmdStart([]string{"zephyr", "--verbose"}); err != nil {
			t.Errorf("cmdStart = %v, want nil", err)
		}
	})
	if !strings.Contains(stdout, "up") || !strings.Contains(stdout, "zephyr") {
		t.Fatalf("stdout = %q, want the node reported up by name", stdout)
	}
	if !strings.Contains(stdout, logOf("zephyr")) {
		t.Fatalf("stdout = %q, want the log path - it is where the next question is answered", stdout)
	}
	if strings.Contains(stderr, "claude is not on PATH") {
		t.Fatalf("stderr = %q, want no harness warning when claude is on PATH", stderr)
	}
	pid := pidOf("zephyr")
	if pid == 0 {
		t.Fatal("pidOf = 0 after a successful start; the node did not actually come up")
	}
	if !strings.Contains(stdout, strconv.Itoa(pid)) {
		t.Fatalf("stdout = %q, want the pid %d in it", stdout, pid)
	}
	// 노드는 enodectl 이 끝나도 살아 있어야 한다 - 부모에서 떼어냈으므로
	// 우리 프로세스 그룹의 자식이 아니다.
	log, err := os.ReadFile(logOf("zephyr"))
	if err != nil {
		t.Fatalf("read the log: %v", err)
	}
	if !strings.Contains(string(log), "start =====") {
		t.Fatalf("log = %q, want the start banner appended", log)
	}
	if !strings.Contains(string(log), "--config") || !strings.Contains(string(log), "--verbose") {
		t.Fatalf("log = %q, want the extra arguments passed through to the node", log)
	}
}

func TestCmdStart_WarnsWhenTheHarnessIsNotOnPath(t *testing.T) {
	// 맥에서 가장 잘 밟는 자리다 - launchd 로 띄우면 PATH 가 최소 집합이라
	// claude 를 못 찾고, 그러면 harness 가 광고에서 조용히 빠져 계약이 422 를
	// 받는다. 원인이 안 보이는 것이 이 경고가 막으려는 것이다.
	isolate(t)
	writeConfig(t, "zephyr")
	gitIdentity(t, "node-test@example.invalid")
	pathWith(t, "ps", "sleep", "git")
	fakeEnode(t, true)
	t.Cleanup(func() { reapNode(t, "zephyr") })

	_, stderr := captureOutput(t, func() {
		if err := cmdStart([]string{"zephyr"}); err != nil {
			t.Errorf("cmdStart = %v, want nil - a missing harness is a warning, not a failure", err)
		}
	})
	if !strings.Contains(stderr, "claude is not on PATH") {
		t.Fatalf("stderr = %q, want the harness warning", stderr)
	}
	if !strings.Contains(stderr, "will not advertise a harness") {
		t.Fatalf("stderr = %q, want the consequence spelled out, not just the fact", stderr)
	}
}

func TestCmdStart_ShowsTheLogWhenTheNodeDoesNotComeUp(t *testing.T) {
	isolate(t)
	writeConfig(t, "zephyr")
	gitIdentity(t, "node-test@example.invalid")
	pathWith(t, "ps", "sleep", "git")
	fakeEnode(t, false)

	_, stderr := captureOutput(t, func() {
		err := cmdStart([]string{"zephyr"})
		if err == nil || !strings.Contains(err.Error(), "start failed") {
			t.Errorf("cmdStart = %v, want a start failure", err)
		}
	})
	if !strings.Contains(stderr, "did not come up") {
		t.Fatalf("stderr = %q, want it to say the node did not come up", stderr)
	}
	// 로그의 꼬리를 함께 낸다 - 안 내면 사람이 그 파일을 손으로 찾아야
	// 하고, 실패의 이유가 거기 있다.
	if !strings.Contains(stderr, "mediator is required") {
		t.Fatalf("stderr = %q, want the tail of the log that says why", stderr)
	}
}

// ── 멈추기 ───────────────────────────────────────────────────────────────

func TestCmdStop_IsQuietAboutANodeThatIsAlreadyStopped(t *testing.T) {
	isolate(t)
	writeConfig(t, "zephyr")
	stdout, _ := captureOutput(t, func() {
		if err := cmdStop([]string{"zephyr"}); err != nil {
			t.Errorf("cmdStop = %v, want nil - stopping a stopped node is not a failure", err)
		}
	})
	if !strings.Contains(stdout, "already stopped: zephyr") {
		t.Fatalf("stdout = %q, want it to say the node was already stopped", stdout)
	}
	if err := cmdStop(nil); err == nil {
		t.Fatal("cmdStop(nil) = nil, want an error asking for a name")
	}
}

func TestCmdStop_AsksTheNodeToTidyUpAndLeave(t *testing.T) {
	// SIGTERM 이면 enode 의 signal.NotifyContext 가 받아 임대를 놓고 나간다.
	isolate(t)
	writeConfig(t, "zephyr")
	pathWith(t, "ps", "sleep")
	pid := holdOpen(t, confOf("zephyr"), false)
	lockFor(t, "zephyr", pid)

	stdout, stderr := captureOutput(t, func() {
		if err := cmdStop([]string{"zephyr"}); err != nil {
			t.Errorf("cmdStop = %v, want nil", err)
		}
	})
	if !strings.Contains(stdout, "stopped: zephyr") {
		t.Fatalf("stdout = %q, want the node reported stopped", stdout)
	}
	if strings.Contains(stderr, "killing") {
		t.Fatalf("stderr = %q, want no kill - the node answered SIGTERM", stderr)
	}
	if got := pidOf("zephyr"); got != 0 {
		t.Fatalf("pidOf = %d after stop, want 0", got)
	}
}

func TestCmdStop_KillsANodeThatWillNotLeave(t *testing.T) {
	// 실측 10초가 드는 유일한 테스트다. 이 대기가 곧 계약이므로 줄일 수
	// 없다 - "10초 안에 안 나가면 죽인다" 를 5초로 재면 다른 약속을 재는
	// 것이 된다.
	isolate(t)
	writeConfig(t, "stubborn")
	pathWith(t, "ps", "sleep")
	pid := holdOpen(t, confOf("stubborn"), true)
	lockFor(t, "stubborn", pid)

	start := time.Now()
	_, stderr := captureOutput(t, func() {
		if err := cmdStop([]string{"stubborn"}); err != nil {
			t.Errorf("cmdStop = %v, want nil - the kill is the documented last resort", err)
		}
	})
	if elapsed := time.Since(start); elapsed < 9*time.Second {
		t.Fatalf("cmdStop gave up after %v; it promises to wait 10s before killing", elapsed)
	}
	if !strings.Contains(stderr, "did not exit within 10s") {
		t.Fatalf("stderr = %q, want the warning that the node is being killed", stderr)
	}
	if !strings.Contains(stderr, strconv.Itoa(pid)) {
		t.Fatalf("stderr = %q, want the pid it killed", stderr)
	}
	deadline := time.Now().Add(5 * time.Second)
	for pidOf("stubborn") != 0 {
		if time.Now().After(deadline) {
			t.Fatal("the node survived the kill")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// ── 로그 보기 ────────────────────────────────────────────────────────────

func TestCmdLogs_NeedsANameAndALogThatExists(t *testing.T) {
	isolate(t)
	writeConfig(t, "zephyr")
	if err := cmdLogs(nil); err == nil {
		t.Fatal("cmdLogs(nil) = nil, want an error asking for a name")
	}
	err := cmdLogs([]string{"zephyr"})
	if err == nil || !strings.Contains(err.Error(), "no log:") {
		t.Fatalf("cmdLogs = %v, want it to name the log file that is not there", err)
	}
}

func TestCmdLogs_PrintsTheTailAndHandsFollowingToTail(t *testing.T) {
	isolate(t)
	writeConfig(t, "zephyr")
	var b strings.Builder
	for i := 1; i <= 60; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	if err := os.WriteFile(logOf("zephyr"), []byte(b.String()), 0o600); err != nil {
		t.Fatalf("write the log: %v", err)
	}

	stdout, _ := captureOutput(t, func() {
		if err := cmdLogs([]string{"zephyr"}); err != nil {
			t.Errorf("cmdLogs = %v, want nil", err)
		}
	})
	if !strings.Contains(stdout, "line 60") {
		t.Fatalf("stdout = %q, want the last line", stdout)
	}
	if strings.Contains(stdout, "line 10\n") {
		t.Fatalf("stdout = %q, want at most the last 50 lines", stdout)
	}

	// -f 는 우리가 흉내 내지 않고 tail 에 넘긴다. 흉내 내면 회전과 절단을
	// 다시 구현하게 된다.
	dir := pathWith(t, "ps")
	record := stubIn(t, dir, "tail")
	captureOutput(t, func() {
		if err := cmdLogs([]string{"zephyr", "-f"}); err != nil {
			t.Errorf("cmdLogs -f = %v, want nil", err)
		}
	})
	argv, err := os.ReadFile(record)
	if err != nil {
		t.Fatalf("the tail stub was never run: %v", err)
	}
	if !strings.Contains(string(argv), "-f") || !strings.Contains(string(argv), logOf("zephyr")) {
		t.Fatalf("tail was called with %q, want -f and the log path", argv)
	}
}

// ── 상태 ─────────────────────────────────────────────────────────────────

func TestCmdStatus_AddsTheRecentLogOfEveryRunningNode(t *testing.T) {
	isolate(t)
	gitIdentity(t, "node-test@example.invalid")
	writeConfig(t, "running")
	writeConfig(t, "idle")
	pathWith(t, "ps", "sleep", "git")
	pid := holdOpen(t, confOf("running"), false)
	lockFor(t, "running", pid)
	if err := os.WriteFile(logOf("running"), []byte("a\nb\nc\nd\ne\nf\ng\n"), 0o600); err != nil {
		t.Fatalf("write the log: %v", err)
	}
	if err := os.WriteFile(logOf("idle"), []byte("stale\n"), 0o600); err != nil {
		t.Fatalf("write the idle log: %v", err)
	}

	stdout, _ := captureOutput(t, func() {
		if err := cmdStatus(); err != nil {
			t.Errorf("cmdStatus = %v, want nil", err)
		}
	})
	if !strings.Contains(stdout, "running pid="+strconv.Itoa(pid)) {
		t.Fatalf("stdout = %q, want the live node shown with its pid", stdout)
	}
	if !strings.Contains(stdout, "   g\n") {
		t.Fatalf("stdout = %q, want the indented tail of the running node's log", stdout)
	}
	if strings.Contains(stdout, "   b\n") {
		t.Fatalf("stdout = %q, want at most the last five log lines", stdout)
	}
	// 안 도는 노드의 로그는 안 낸다 - status 는 지금 무슨 일이 벌어지는가다.
	if strings.Contains(stdout, "stale") {
		t.Fatalf("stdout = %q, want no log from the node that is not running", stdout)
	}
}

// ── 갈래 ─────────────────────────────────────────────────────────────────

func TestDispatch_EverySubcommandReachesItsHandler(t *testing.T) {
	// main 은 갈래일 뿐이지만, 갈래가 틀리면 그 아래가 전부 안 닿는다.
	// 여기서 os.Exit 로 끝나는 갈래(알 수 없는 명령)는 뺀다 - 그것을 부르면
	// 테스트 바이너리가 통째로 나간다.
	isolate(t)
	writeConfig(t, "zephyr")
	gitIdentity(t, "node-test@example.invalid")
	dir := pathWith(t, "ps", "sleep", "git")
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write the claude stub: %v", err)
	}
	if err := os.WriteFile(logOf("zephyr"), []byte("a log line\n"), 0o600); err != nil {
		t.Fatalf("write the log: %v", err)
	}
	fakeEnode(t, true)
	t.Cleanup(func() { reapNode(t, "zephyr") })
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// 순서가 있다 - start 로 띄운 것을 stop 이 거둔다.
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"no subcommand falls through to status", nil, "config dir:"},
		{"list", []string{"list"}, "zephyr"},
		{"id", []string{"id", "zephyr"}, "node_id"},
		{"logs", []string{"logs", "zephyr"}, "a log line"},
		{"status", []string{"status"}, "config dir:"},
		{"help", []string{"help"}, "enodectl setup"},
		{"-h", []string{"-h"}, "enodectl setup"},
		{"--help", []string{"--help"}, "enodectl setup"},
		{"version", []string{"version"}, "enodectl"},
		{"--version", []string{"--version"}, "enodectl"},
		{"-version", []string{"-version"}, "enodectl"},
		{"setup", []string{"setup", "probe", "-check", "-yes", "-mediator", srv.URL, "-token", "t"}, "reachable"},
		{"start", []string{"start", "zephyr"}, "up"},
		{"stop", []string{"stop", "zephyr"}, "stopped: zephyr"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			old := os.Args
			os.Args = append([]string{"enodectl"}, tc.args...)
			defer func() { os.Args = old }()
			stdout, _ := captureOutput(t, main)
			if !strings.Contains(stdout, tc.want) {
				t.Fatalf("enodectl %q printed %q, want %q in it", tc.args, stdout, tc.want)
			}
		})
	}
	// setup --check 는 여기서도 아무것도 안 써야 한다.
	if _, err := os.Stat(confOf("probe")); err == nil {
		t.Fatalf("the setup dispatch wrote %s; --check writes nothing", confOf("probe"))
	}
}
