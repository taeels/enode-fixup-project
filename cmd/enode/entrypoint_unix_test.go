//go:build !windows

package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// 프로세스 밖에서 보이는 계약만 본다 - 종료 코드 · stderr 문구 · 시그널
// 처리. (리슨 주소는 이 바이너리에 없다. Mediator 쪽이 든다.)
//
// 왜 함수를 안 부르고 바이너리를 짓는가 - 이 파일이 쓰인 시점에 main() 은
// 아직 쪼개지지 않았고, 쪼개는 것이 이 단위의 일이다. 특성화 테스트는 그
// 수술의 안전망이므로 수술 전 코드에 대해 초록이어야 하고, 그러려면 이음매가
// 필요 없는 자리에서 재야 한다. 프로세스 경계가 그 자리다.
//
// 추출이 끝난 뒤에도 이 파일이 남는 이유는 따로 있다 - os.Exit 이 내는 종료
// 코드와 SIGTERM 의 도달은 인프로세스 호출로는 애초에 관측되지 않는다.
// run() 을 직접 부르는 테스트는 "run 이 1 을 돌려줬다" 까지만 말하고
// "프로세스가 1 로 끝났다" 는 말하지 못한다. 그 둘 사이가 정확히 이 단위가
// 건드리는 자리다.
//
// 커버리지를 목표로 하지 않는다 - 자식 프로세스가 덮은 것은 이 패키지의
// 프로파일에 안 들어간다. 이 패키지를 하한 위로 올리는 것은 U11 의 일이고
// 이 파일은 그 수술이 계약을 안 깼다는 것만 말한다.
//
// 왜 빌드 태그로 가르는가 - 이 저장소는 OS 분기를 빌드 태그 쌍으로만 한다
// (lock_unix.go/lock_windows.go). cmd/enodectl/lifecycle_unix_test.go 가
// 같은 이유로 같은 모양을 세웠다. 런타임 t.Skip 으로 가르는 안을 기각했다 -
// CI 의 스킵 감시가 전 패키지를 보고 .ci-allowed-skips 는 비어 있다.
//
// 목을 세우지 않는다. 경계가 셋뿐이고 셋 다 실물로 선다 - 파일 시스템은
// t.TempDir(), 신원은 진짜 git, 프로세스는 진짜 자식.

// ── 바이너리 ─────────────────────────────────────────────────────────────

// enodeBin 은 이 패키지를 지어 둔 자리다. 경로가 유일하다는 것이 아래
// survivors 의 전제다 - 그 문자열을 argv 에 들고 있는 프로세스는 우리 것뿐이다.
var enodeBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "enode-entrypoint-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot make a directory for the binary under test: %v\n", err)
		os.Exit(1)
	}
	enodeBin = filepath.Join(dir, "enode")
	build := exec.Command("go", "build", "-o", enodeBin, ".")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "cannot build the binary under test: %v\n", err)
		_ = os.RemoveAll(dir)
		os.Exit(1)
	}
	// 실행 전 pid 집합. 경로가 방금 만들어졌으므로 비어 있어야 하고, 안
	// 비었으면 유일성 전제가 틀린 것이라 아래의 고아 판정도 못 믿는다.
	if before := survivors(); len(before) > 0 {
		fmt.Fprintf(os.Stderr, "pid %v already carries %s before any test ran\n", before, enodeBin)
		_ = os.RemoveAll(dir)
		os.Exit(1)
	}

	code := m.Run()

	// 실행 후 pid 집합. 띄운 것을 그룹째 거두지 못했으면 그 잔재가 다음
	// 실행의 판정을 바꾼다 - 잠금 파일 하나로 노드가 갈리는 바이너리라
	// 살아남은 자식 하나가 다음 회차의 "cannot acquire lock" 을 만든다.
	if after := survivors(); len(after) > 0 {
		fmt.Fprintf(os.Stderr, "orphan contract: pid %v still carries %s after the suite\n", after, enodeBin)
		code = 1
	}
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// survivors 는 이 테스트가 지은 바이너리를 아직 들고 있는 pid 들이다.
func survivors() []int {
	out, err := exec.Command("ps", "-eo", "pid=,args=").Output()
	if err != nil {
		return nil
	}
	var pids []int
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, enodeBin) {
			continue
		}
		f := strings.Fields(line)
		if len(f) == 0 {
			continue
		}
		if pid, err := strconv.Atoi(f[0]); err == nil {
			pids = append(pids, pid)
		}
	}
	return pids
}

// ── 자식을 어떻게 띄우는가 ───────────────────────────────────────────────

// childEnv 는 자식에게 줄 환경을 빈 것에서 쌓는다.
//
// os.Environ() 을 거르지 않고 빈 것에서 허용목록으로 쌓는 것은 이 저장소의
// 관행 그대로다 (internal/enode/env.go 의 harnessEnv). 여기서 그 모양을
// 쓰는 이유는 이 테스트를 돌리는 사람의 기계에 ENODE_CONFIG 나 ENODE_CONFDIR
// 이 있으면 "설정을 못 찾았다" 갈래가 조용히 다른 갈래가 되기 때문이다.
type childEnv struct {
	home  string
	extra map[string]string
}

// isolate 는 이 테스트만의 HOME 을 준다.
//
// ConfigPaths() 의 첫 자리가 $HOME/.config/enode/local.yaml 이므로 HOME 하나로
// 사용자 자리가 통째로 갈린다. 시스템 자리(/etc/enode/local.yaml)는 환경으로
// 못 옮기므로 아래 noSystemConfig 가 그 자리가 비어 있음을 단언한다.
func isolate(t *testing.T) *childEnv {
	t.Helper()
	return &childEnv{home: t.TempDir(), extra: map[string]string{}}
}

// gitIdentity 는 enode.Derive 가 읽는 전역 git 이메일을 고정한다.
//
// email 이 비면 이메일이 없는 기계를 만든다 - ADR-015 §1 이 조용한 대체를
// 금했으므로 그 자리에서 죽는 것이 계약이고, 그 계약도 시험 대상이다.
func (e *childEnv) gitIdentity(t *testing.T, email string) *childEnv {
	t.Helper()
	path := os.DevNull
	if email != "" {
		path = filepath.Join(t.TempDir(), "gitconfig")
		if err := os.WriteFile(path, []byte("[user]\n\temail = "+email+"\n"), 0o600); err != nil {
			t.Fatalf("write a git config for the test: %v", err)
		}
	}
	e.extra["GIT_CONFIG_GLOBAL"] = path
	e.extra["GIT_CONFIG_NOSYSTEM"] = "1"
	return e
}

func (e *childEnv) slice() []string {
	env := []string{"HOME=" + e.home, "PATH=" + os.Getenv("PATH")}
	for k, v := range e.extra {
		env = append(env, k+"="+v)
	}
	return env
}

// userConfigPath 는 이 환경에서 enode 가 첫 번째로 볼 자리다.
func (e *childEnv) userConfigPath() string {
	return filepath.Join(e.home, ".config", "enode", "local.yaml")
}

// noSystemConfig 는 시스템 자리가 비어 있음을 단언한다.
//
// 스킵이 아니라 단언인 것이 일부러다. 이 기계에 /etc/enode/local.yaml 이
// 있으면 "설정을 못 찾았다" 갈래가 아예 안 돌고 노드가 뜨려 든다 - 조용히
// 다른 것을 재는 대신 무엇 때문에 못 재는지를 말하고 죽는 편이 낫다.
func noSystemConfig(t *testing.T) {
	t.Helper()
	const p = "/etc/enode/local.yaml"
	if _, err := os.Stat(p); err == nil {
		t.Fatalf("%s exists on this machine; the no-config branch cannot be measured here", p)
	}
}

// writeConfig 는 설정 파일을 만들고 그 경로를 돌려준다.
func writeConfig(t *testing.T, dir, name, body string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

// syncBuf 는 도는 자식의 출력을 읽으면서 모으는 버퍼다.
type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// child 는 도는 자식 하나다.
type child struct {
	t              *testing.T
	cmd            *exec.Cmd
	stdout, stderr *syncBuf
}

// start 는 자식에게 자기 프로세스 그룹을 주고 띄운다.
//
// 그룹을 주는 이유 - 셸이나 자식의 자식까지 그룹째 거두려면 음수 pid 로 한
// 번에 보내야 한다. cmd/enodectl/lifecycle_unix_test.go 가 같은 함정을 먼저
// 밟았고 같은 처방을 썼다.
func start(t *testing.T, e *childEnv, args ...string) *child {
	t.Helper()
	c := &child{t: t, cmd: exec.Command(enodeBin, args...), stdout: &syncBuf{}, stderr: &syncBuf{}}
	c.cmd.Env = e.slice()
	c.cmd.Stdout = c.stdout
	c.cmd.Stderr = c.stderr
	c.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := c.cmd.Start(); err != nil {
		t.Fatalf("cannot start %s %v: %v", enodeBin, args, err)
	}
	t.Cleanup(c.reap)
	return c
}

// reap 은 그 프로세스와 그 아래를 통째로 거둔다.
func (c *child) reap() {
	if c.cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-c.cmd.Process.Pid, syscall.SIGKILL)
	_ = c.cmd.Process.Kill()
	_, _ = c.cmd.Process.Wait()
}

// awaitStderr 는 stderr 에 그 문구가 나올 때까지 기다린다.
func (c *child) awaitStderr(want string) {
	c.t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		if strings.Contains(c.stderr.String(), want) {
			return
		}
		if time.Now().After(deadline) {
			c.t.Fatalf("%q never appeared on stderr; it said %q", want, c.stderr.String())
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// wait 은 자식이 끝나기를 기다리고 종료 코드를 돌려준다.
func (c *child) wait() int {
	c.t.Helper()
	done := make(chan error, 1)
	go func() { done <- c.cmd.Wait() }()
	select {
	case err := <-done:
		if err == nil {
			return 0
		}
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return ee.ExitCode()
		}
		c.t.Fatalf("waiting for the child: %v", err)
		return -1
	case <-time.After(60 * time.Second):
		c.reap()
		c.t.Fatalf("the child never exited; stderr so far: %q", c.stderr.String())
		return -1
	}
}

// exec1 은 한 번 돌리고 끝날 때까지 기다린다.
func exec1(t *testing.T, e *childEnv, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	c := start(t, e, args...)
	code = c.wait()
	return code, c.stdout.String(), c.stderr.String()
}

// ── 종료 코드와 stderr ───────────────────────────────────────────────────

func TestVersion_AnswersBeforeAnyConfigIsLooked(t *testing.T) {
	// ADR-056 - --version 은 설정 파일이 없어도 답해야 한다. 자기 갱신이
	// 받아온 것이 무엇인지를 이것으로 판정하므로, 설정이 없는 기계에서
	// 종료 코드가 0 이 아니면 그 판정 자체가 불가능해진다.
	noSystemConfig(t)
	for _, spelling := range []string{"--version", "-version"} {
		t.Run(spelling, func(t *testing.T) {
			code, stdout, stderr := exec1(t, isolate(t), spelling)
			if code != 0 {
				t.Fatalf("exit code contract: enode %s = %d, want 0 (ADR-056); stderr %q",
					spelling, code, stderr)
			}
			if !strings.HasPrefix(stdout, "enode ") {
				t.Fatalf("enode %s printed %q on stdout, want it to start with the program name", spelling, stdout)
			}
		})
	}
}

func TestNoConfig_NamesEveryPlaceItLookedAndExitsOne(t *testing.T) {
	// ADR-015 §2 - 설정 파일이 곧 신원이다. 못 찾았을 때 어디를 봤는지
	// 말하지 않으면 사람이 엉뚱한 경로를 들여다본다.
	noSystemConfig(t)
	e := isolate(t)
	code, _, stderr := exec1(t, e)
	if code != 1 {
		t.Fatalf("exit code contract: enode with no config = %d, want 1; stderr %q", code, stderr)
	}
	for _, want := range []string{
		"no config file found",
		e.userConfigPath(),
		"/etc/enode/local.yaml",
		"Create one and point --config at it",
		"mediator: http://<mediator-host>:8080",
		"The absolute path of this file is part of the node id",
	} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr wording contract: enode with no config said %q, want %q in it", stderr, want)
		}
	}
}

func TestUnknownSubcommand_IsNotASubcommandAtAll(t *testing.T) {
	// enode 의 하위 명령은 hook · setup · --version 셋뿐이고 나머지는
	// 위치인자로 조용히 무시된다. 이것이 오늘의 계약이므로 못박는다 -
	// 추출하면서 여기에 usage 를 붙이고 싶어지는 자리이고, 붙이면
	// AC3.2.5 의 "신규 기능 0" 을 넘는다.
	noSystemConfig(t)
	bogusCode, _, bogusErr := exec1(t, isolate(t), "bogus")
	bareCode, _, bareErr := exec1(t, isolate(t))
	if bogusCode != bareCode {
		t.Fatalf("exit code contract: enode bogus = %d but bare enode = %d; an unknown word is a positional, not an error of its own",
			bogusCode, bareCode)
	}
	if !strings.Contains(bogusErr, "no config file found") || !strings.Contains(bareErr, "no config file found") {
		t.Fatalf("enode bogus said %q and bare enode said %q; both should reach the same config search",
			bogusErr, bareErr)
	}
}

func TestConfigCannotBeRead_ExitsOneAndSaysWhichFile(t *testing.T) {
	// 못 찾은 것과 못 읽는 것을 가르는 것이 main 의 첫 갈림이다. --config 로
	// 명시하면 없어도 조용히 넘어가지 않는다 (internal/enode/paths.go).
	dir := t.TempDir()
	missing := filepath.Join(dir, "absent.yaml")
	malformed := writeConfig(t, dir, "malformed.yaml", ": : not yaml [\n")
	for _, tc := range []struct {
		name, path string
	}{
		{"a config file that is not there", missing},
		{"a config file that is not yaml", malformed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, _, stderr := exec1(t, isolate(t), "--config", tc.path)
			if code != 1 {
				t.Fatalf("exit code contract: enode --config %s = %d, want 1; stderr %q", tc.path, code, stderr)
			}
			if !strings.Contains(stderr, "cannot read config") {
				t.Fatalf("stderr wording contract: got %q, want %q in it", stderr, "cannot read config")
			}
			if !strings.Contains(stderr, tc.path) {
				t.Fatalf("stderr wording contract: got %q, want the offending path %q named", stderr, tc.path)
			}
		})
	}
}

func TestMediatorAndTokenAreRequired_ExitsOne(t *testing.T) {
	// ADR-015 §1 - 조용한 대체를 하지 않는다. 주소도 토큰도 없이 뜨면
	// 노드는 아무 데도 못 붙은 채 살아 있게 되고, 그것이 가장 나쁜 상태다.
	cfg := writeConfig(t, t.TempDir(), "local.yaml", "workspace: /tmp\n")
	code, _, stderr := exec1(t, isolate(t), "--config", cfg)
	if code != 1 {
		t.Fatalf("exit code contract: a config with neither mediator nor token = %d, want 1; stderr %q", code, stderr)
	}
	if !strings.Contains(stderr, "mediator and token are required") {
		t.Fatalf("stderr wording contract: got %q, want %q in it", stderr, "mediator and token are required")
	}
}

func TestIdentityCannotBeDerived_ExitsOne(t *testing.T) {
	// ADR-015 §1 다시 - 이메일이 없으면 그 자리에서 죽는다. 대체 신원을
	// 만들면 그 노드는 재시작마다 다른 노드가 된다 (ADR-017).
	cfg := writeConfig(t, t.TempDir(), "local.yaml", "mediator: http://127.0.0.1:1\ntoken: t\n")
	e := isolate(t).gitIdentity(t, "")
	code, _, stderr := exec1(t, e, "--config", cfg)
	if code != 1 {
		t.Fatalf("exit code contract: no git email = %d, want 1; stderr %q", code, stderr)
	}
	if !strings.Contains(stderr, "cannot derive node identity") {
		t.Fatalf("stderr wording contract: got %q, want %q in it", stderr, "cannot derive node identity")
	}
}

func TestHookWithoutStop_PrintsUsageAndExitsTwo(t *testing.T) {
	// hook 은 절대 하네스를 막지 않는다 - 그래서 무슨 일이 있어도 0 인데,
	// 사용법을 몰라 부른 것은 예외다 (hook.go). 이 2 가 0 으로 바뀌면
	// 오타를 낸 사람이 훅이 돌았다고 믿는다.
	for _, args := range [][]string{{"hook"}, {"hook", "bogus"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			code, _, stderr := exec1(t, isolate(t), args...)
			if code != 2 {
				t.Fatalf("exit code contract: enode %v = %d, want 2; stderr %q", args, code, stderr)
			}
			if !strings.Contains(stderr, "usage: enode hook stop --out <dir>") {
				t.Fatalf("stderr wording contract: got %q, want the hook usage line", stderr)
			}
		})
	}
}

// ── 시그널 ───────────────────────────────────────────────────────────────

func TestSigterm_StopsCleanlyWithExitZero(t *testing.T) {
	// signal.NotifyContext 가 잡고, 두 고루틴이 ctx 로 끝나고, main 이
	// "stopped" 를 찍고 0 으로 끝난다. Mediator 에 못 붙어도 그렇다 -
	// 실패한 하트비트 하나는 중단 신호가 아니다 (ADR-016).
	//
	// 이 테스트가 특히 무거운 것이 아니라 무거워야 한다. 추출은 os.Exit 을
	// return 으로 바꾸는 일이고, 시그널 경로는 그 return 이 실제로 프로세스의
	// 종료 코드가 되는지를 보는 유일한 자리다.
	dir := t.TempDir()
	cfg := writeConfig(t, dir, "local.yaml", "mediator: http://127.0.0.1:1\ntoken: t\n")
	e := isolate(t).gitIdentity(t, "u8-node@example.invalid")

	c := start(t, e, "--config", cfg)
	c.awaitStderr("enode started")
	if err := c.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("cannot send SIGTERM: %v", err)
	}
	if code := c.wait(); code != 0 {
		t.Fatalf("signal contract: SIGTERM made enode exit %d, want 0; stderr %q", code, c.stderr.String())
	}
	if !strings.Contains(c.stderr.String(), "msg=stopped") {
		t.Fatalf("stderr wording contract: got %q, want the final %q line", c.stderr.String(), "msg=stopped")
	}
	// 잠금은 커널이 푼다 (flock) - 프로세스가 끝났으면 같은 설정으로 다시
	// 뜰 수 있어야 한다. 못 뜨면 노드 하나가 재시작으로 영영 죽는다.
	c2 := start(t, e, "--config", cfg)
	c2.awaitStderr("enode started")
	if err := c2.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("cannot send SIGTERM to the second run: %v", err)
	}
	if code := c2.wait(); code != 0 {
		t.Fatalf("lock contract: the second run after a clean stop exited %d, want 0; stderr %q",
			code, c2.stderr.String())
	}
}
