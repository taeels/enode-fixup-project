//go:build !windows

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// 프로세스 밖에서 보이는 계약만 본다 - 종료 코드 · stderr 문구 · 리슨 주소 ·
// 시그널 처리. 넷 다 AC3.2.5 가 이름으로 든 것이고, 넷 다 이 파일이 든다.
//
// 왜 함수를 안 부르고 바이너리를 짓는가 - 이 파일이 쓰인 시점에 main() 은
// 아직 쪼개지지 않았고, 쪼개는 것이 이 단위의 일이다. 특성화 테스트는 그
// 수술의 안전망이므로 수술 전 코드에 대해 초록이어야 하고, 그러려면 이음매가
// 필요 없는 자리에서 재야 한다.
//
// 추출이 끝난 뒤에도 이 파일이 남는 이유는 따로 있다 - 알려진 델타 ②가
// ListenAndServe 고루틴의 os.Exit(1) 을 오류 채널로 옮긴다. 그 변경이 바꾸는
// 것은 종료의 순서이고 안 바꿔야 하는 것은 종료 코드와 stderr 인데, 그 구분은
// 프로세스 경계에서만 관측된다. run() 을 직접 부르는 테스트는 "run 이 1 을
// 돌려줬다" 까지만 말한다.
//
// 커버리지를 목표로 하지 않는다 - 자식 프로세스가 덮은 것은 이 패키지의
// 프로파일에 안 들어간다. 이 패키지를 하한 위로 올리는 것은 U11 의 일이다.
//
// 왜 빌드 태그로 가르는가 - 이 저장소는 OS 분기를 빌드 태그 쌍으로만 한다
// (internal/enode/lock_unix.go/lock_windows.go). 런타임 t.Skip 으로 가르는
// 안을 기각했다 - CI 의 스킵 감시가 전 패키지를 보고 .ci-allowed-skips 는
// 비어 있다.
//
// 목을 세우지 않는다. 경계가 셋뿐이고 셋 다 실물로 선다 - 파일 시스템은
// t.TempDir(), 프로세스는 진짜 자식, Postgres 는 진짜 Postgres.

// ── 바이너리 ─────────────────────────────────────────────────────────────

// mediatorBin 은 이 패키지를 지어 둔 자리다. 경로가 유일하다는 것이 아래
// survivors 의 전제다 - 그 문자열을 argv 에 들고 있는 프로세스는 우리 것뿐이다.
var mediatorBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "mediator-entrypoint-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot make a directory for the binary under test: %v\n", err)
		os.Exit(1)
	}
	mediatorBin = filepath.Join(dir, "mediator")
	build := exec.Command("go", "build", "-o", mediatorBin, ".")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "cannot build the binary under test: %v\n", err)
		_ = os.RemoveAll(dir)
		os.Exit(1)
	}
	if before := survivors(); len(before) > 0 {
		fmt.Fprintf(os.Stderr, "pid %v already carries %s before any test ran\n", before, mediatorBin)
		_ = os.RemoveAll(dir)
		os.Exit(1)
	}

	code := m.Run()

	// 살아남은 자식 하나가 다음 회차의 리슨 주소를 점거한다 - 바인드 실패
	// 갈래와 리슨 갈래가 서로의 결과를 바꾸는 모양이 정확히 그것이다.
	if after := survivors(); len(after) > 0 {
		fmt.Fprintf(os.Stderr, "orphan contract: pid %v still carries %s after the suite\n", after, mediatorBin)
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
		if !strings.Contains(line, mediatorBin) {
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
// 관행 그대로다 (internal/enode/env.go 의 harnessEnv). 여기서 그 모양이
// 필요한 이유는 구체적이다 - 이 테스트를 돌리는 환경에는
// ENODE_TEST_DATABASE_URL 이 반드시 있고, DATABASE_URL 이나
// ENODE_MEDIATOR_TOKEN 이 새어 들어가면 "설정을 못 찾았다" 갈래와
// "토큰이 없다" 갈래가 조용히 다른 갈래가 된다.
type childEnv struct {
	home  string
	extra map[string]string
}

// isolate 는 이 테스트만의 HOME 을 준다.
//
// config.Paths() 의 첫 자리가 $HOME/.config/enode-mediator/config.yaml 이므로
// HOME 하나로 사용자 자리가 통째로 갈린다. 시스템 자리는 환경으로 못 옮기므로
// 아래 noSystemConfig 가 그 자리가 비어 있음을 단언한다.
func isolate(t *testing.T) *childEnv {
	t.Helper()
	return &childEnv{home: t.TempDir(), extra: map[string]string{}}
}

func (e *childEnv) slice() []string {
	env := []string{"HOME=" + e.home, "PATH=" + os.Getenv("PATH")}
	for k, v := range e.extra {
		env = append(env, k+"="+v)
	}
	return env
}

// userConfigPath 는 이 환경에서 mediator 가 첫 번째로 볼 자리다.
func (e *childEnv) userConfigPath() string {
	return filepath.Join(e.home, ".config", "enode-mediator", "config.yaml")
}

// noSystemConfig 는 시스템 자리가 비어 있음을 단언한다.
//
// 스킵이 아니라 단언인 것이 일부러다. 이 기계에
// /etc/enode-mediator/config.yaml 이 있으면 "설정을 못 찾았다" 갈래가 아예
// 안 돌고, 조용히 다른 것을 재게 된다.
func noSystemConfig(t *testing.T) {
	t.Helper()
	const p = "/etc/enode-mediator/config.yaml"
	if _, err := os.Stat(p); err == nil {
		t.Fatalf("%s exists on this machine; the no-config branch cannot be measured here", p)
	}
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
func start(t *testing.T, e *childEnv, args ...string) *child {
	t.Helper()
	c := &child{t: t, cmd: exec.Command(mediatorBin, args...), stdout: &syncBuf{}, stderr: &syncBuf{}}
	c.cmd.Env = e.slice()
	c.cmd.Stdout = c.stdout
	c.cmd.Stderr = c.stderr
	c.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := c.cmd.Start(); err != nil {
		t.Fatalf("cannot start %s %v: %v", mediatorBin, args, err)
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

// ── Postgres ─────────────────────────────────────────────────────────────

// scratchDB 는 이 테스트만의 데이터베이스를 만들고 그 URL 을 돌려준다.
//
// 공용 테스트 DB 를 그대로 쓰는 안을 기각했다. mediator 는 뜨자마자
// RunReaper 를 t=0 에 한 번 돌리고(재시작 스캔), 그 스캔은 만료된 임대를
// 회수한다. go test ./... 는 패키지별 테스트 바이너리를 병렬로 돌리므로
// internal/api 의 통합 테스트가 만든 임대를 이 테스트의 mediator 가 거둘 수
// 있다. 두 패키지의 테스트가 서로의 결과를 바꾸는 모양이고, 그것은 아무도
// 재현하지 못하는 실패다. 판 하나를 따로 파면 그 경로가 아예 없어진다.
func scratchDB(t *testing.T) string {
	t.Helper()
	base := os.Getenv("ENODE_TEST_DATABASE_URL")
	if base == "" {
		// 스킵하지 않는다 - CI 의 스킵 감시가 전 패키지를 보고
		// .ci-allowed-skips 는 비어 있다. 조건을 못 갖췄으면 조용히
		// 사라지는 대신 무엇이 없는지 말하고 죽는다.
		t.Fatal("ENODE_TEST_DATABASE_URL is unset; see scripts/testdb.sh. " +
			"the mediator entrypoint contract cannot be measured without a real postgres")
	}
	name := fmt.Sprintf("u8_mediator_%d_%d", os.Getpid(), time.Now().UnixNano()%1e9)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	admin, err := pgx.Connect(ctx, base)
	if err != nil {
		t.Fatalf("cannot reach the test postgres at ENODE_TEST_DATABASE_URL: %v", err)
	}
	defer admin.Close(ctx) //nolint:errcheck // 닫기 실패는 판정을 바꾸지 않는다
	if _, err := admin.Exec(ctx, `CREATE DATABASE "`+name+`"`); err != nil {
		t.Fatalf("cannot create the scratch database %s: %v", name, err)
	}
	t.Cleanup(func() {
		dctx, dcancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer dcancel()
		drop, err := pgx.Connect(dctx, base)
		if err != nil {
			return
		}
		defer drop.Close(dctx) //nolint:errcheck // 닫기 실패는 판정을 바꾸지 않는다
		_, _ = drop.Exec(dctx, `DROP DATABASE IF EXISTS "`+name+`" WITH (FORCE)`)
	})
	return replaceDBName(t, base, name)
}

// replaceDBName 은 URL 의 데이터베이스 이름만 갈아 끼운다.
func replaceDBName(t *testing.T, raw, name string) string {
	t.Helper()
	cfg, err := pgx.ParseConfig(raw)
	if err != nil {
		t.Fatalf("cannot parse ENODE_TEST_DATABASE_URL: %v", err)
	}
	host := cfg.Host
	if !strings.HasPrefix(host, "/") {
		host = net.JoinHostPort(cfg.Host, strconv.Itoa(int(cfg.Port)))
	}
	u := "postgres://" + cfg.User
	if cfg.Password != "" {
		u += ":" + cfg.Password
	}
	return u + "@" + host + "/" + name + "?sslmode=disable"
}

// freePort 는 지금 비어 있는 포트 하나를 돌려준다.
//
// 열었다 닫는 것 말고 다른 방법이 없다 - 커널이 정한 것을 물어봐야 그것이
// 비어 있다는 근거가 된다. 닫은 뒤 누가 채갈 창이 이론상 남지만, 그 창에서
// 실패하면 "address already in use" 로 시끄럽게 실패하지 조용히 다른 것을
// 재지는 않는다.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("cannot find a free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	if err := l.Close(); err != nil {
		t.Fatalf("cannot release the probe listener: %v", err)
	}
	return port
}

// mediatorConfig 는 뜰 수 있는 설정 파일을 만든다.
func mediatorConfig(t *testing.T, listen, dbURL, artifacts string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.yaml")
	body := "listen: " + listen + "\n" +
		"token: u8-token\n" +
		"database:\n  url: " + dbURL + "\n" +
		"artifacts:\n  root: " + artifacts + "\n"
	// 0600 으로 쓴다 - 0644 면 config.Load 가 stderr 에 권고 한 줄을 더
	// 얹고, 그 줄이 stderr 단언의 잡음이 된다.
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

// ── 종료 코드와 stderr ───────────────────────────────────────────────────

func TestVersion_AnswersOnAllThreeSpellingsBeforeAnythingIsLoaded(t *testing.T) {
	// ADR-056 - --version 은 설정도 DB 도 없는 상태에서 답해야 한다.
	// 세 철자를 다 거는 것은 main.go 의 switch 가 셋을 다 받기 때문이고,
	// 하나가 빠지면 그것으로 스크립트를 짠 사람이 조용히 다른 갈래로 간다.
	noSystemConfig(t)
	for _, spelling := range []string{"--version", "-version", "version"} {
		t.Run(spelling, func(t *testing.T) {
			code, stdout, stderr := exec1(t, isolate(t), spelling)
			if code != 0 {
				t.Fatalf("exit code contract: mediator %s = %d, want 0 (ADR-056); stderr %q",
					spelling, code, stderr)
			}
			if !strings.HasPrefix(stdout, "mediator ") {
				t.Fatalf("mediator %s printed %q on stdout, want it to start with the program name",
					spelling, stdout)
			}
		})
	}
}

func TestNoConfig_NamesEveryPlaceItLookedAndPointsAtSetup(t *testing.T) {
	// 설정을 못 찾은 것과 토큰만 빠진 것을 가르는 것이 main 의 첫 갈림이다.
	// 예전에는 둘이 같은 메시지로 나와 사람이 토큰만 들여다봤다 (main.go).
	noSystemConfig(t)
	e := isolate(t)
	code, _, stderr := exec1(t, e)
	if code != 1 {
		t.Fatalf("exit code contract: mediator with no config = %d, want 1; stderr %q", code, stderr)
	}
	for _, want := range []string{
		"no config file found",
		e.userConfigPath(),
		"/etc/enode-mediator/config.yaml",
		"Run `mediator setup` to create one.",
	} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr wording contract: mediator with no config said %q, want %q in it", stderr, want)
		}
	}
}

func TestConfigCannotBeRead_ExitsOneAndSaysWhichFile(t *testing.T) {
	dir := t.TempDir()
	malformed := filepath.Join(dir, "malformed.yaml")
	if err := os.WriteFile(malformed, []byte(": : not yaml [\n"), 0o600); err != nil {
		t.Fatalf("write %s: %v", malformed, err)
	}
	for _, tc := range []struct {
		name, path string
	}{
		{"a config file that is not there", filepath.Join(dir, "absent.yaml")},
		{"a config file that is not yaml", malformed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, _, stderr := exec1(t, isolate(t), "--config", tc.path)
			if code != 1 {
				t.Fatalf("exit code contract: mediator --config %s = %d, want 1; stderr %q",
					tc.path, code, stderr)
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

func TestTokenBootstrap_GeneratesOneAndWritesItBack(t *testing.T) {
	// ADR-015 §1 은 인증을 건너뛰는 것을 막는 것이지 부트스트랩을 막는 것이
	// 아니다 - 파일은 있는데 값만 비었으면 만들어 넣고, 넣었다고 말한다.
	// 조용한 대체가 아니라는 것이 이 갈래의 계약이므로 셋을 다 본다:
	// 만들었다는 로그 · 화면에 찍힌 값 · 파일에 남은 값.
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	body := "listen: 127.0.0.1:1\n" +
		"database:\n  url: postgres://nobody:nobody@127.0.0.1:1/nope?sslmode=disable\n"
	if err := os.WriteFile(cfg, []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", cfg, err)
	}

	code, _, stderr := exec1(t, isolate(t), "--config", cfg)
	// 토큰을 만든 뒤에도 DB 가 없으므로 1 로 끝난다. 그 1 이 여기서
	// 확인하는 것은 "부트스트랩이 종료 코드를 삼키지 않는다" 다.
	if code != 1 {
		t.Fatalf("exit code contract: token bootstrap then an unreachable database = %d, want 1; stderr %q",
			code, stderr)
	}
	if !strings.Contains(stderr, "generated a token and wrote it to the config") {
		t.Fatalf("stderr wording contract: got %q, want the generation announced", stderr)
	}
	if !strings.Contains(stderr, "different token is rejected with 401") {
		t.Fatalf("stderr wording contract: got %q, want the token printed with what it is for", stderr)
	}
	written, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("read back %s: %v", cfg, err)
	}
	if !strings.Contains(string(written), "token:") {
		t.Fatalf("the generated token was never written to %s; it said %q", cfg, string(written))
	}
}

func TestDatabaseUnreachable_ExitsOne(t *testing.T) {
	cfg := mediatorConfig(t, "127.0.0.1:1",
		"postgres://nobody:nobody@127.0.0.1:1/nope?sslmode=disable",
		filepath.Join(t.TempDir(), "artifacts"))
	code, _, stderr := exec1(t, isolate(t), "--config", cfg)
	if code != 1 {
		t.Fatalf("exit code contract: an unreachable database = %d, want 1; stderr %q", code, stderr)
	}
	if !strings.Contains(stderr, "cannot open database") {
		t.Fatalf("stderr wording contract: got %q, want %q in it", stderr, "cannot open database")
	}
}

func TestArtifactsRootCannotBeCreated_ExitsOne(t *testing.T) {
	// ADR-015 §3 - Run Record 는 파일시스템에 산다. 그 자리를 못 만들면
	// I4(봉인)를 강제할 데가 없으므로 뜨면 안 된다.
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("not a directory\n"), 0o600); err != nil {
		t.Fatalf("write %s: %v", blocker, err)
	}
	cfg := mediatorConfig(t, "127.0.0.1:"+strconv.Itoa(freePort(t)), scratchDB(t),
		filepath.Join(blocker, "artifacts"))
	code, _, stderr := exec1(t, isolate(t), "--config", cfg)
	if code != 1 {
		t.Fatalf("exit code contract: an artifacts root under a file = %d, want 1; stderr %q", code, stderr)
	}
	if !strings.Contains(stderr, "cannot create artifacts directory") {
		t.Fatalf("stderr wording contract: got %q, want %q in it", stderr, "cannot create artifacts directory")
	}
}

// ── 리슨 주소 ────────────────────────────────────────────────────────────

func TestListen_BindsTheAddressTheConfigNames(t *testing.T) {
	// 리슨 주소는 설정이 정하고 아무도 덮어쓰지 않는다. 로그만 보는 것으로는
	// 모자라다 - 찍는 것과 바인드하는 것이 갈리면 로그가 거짓말을 한다.
	// 그래서 찍힌 주소로 실제로 붙어 본다.
	port := freePort(t)
	addr := "127.0.0.1:" + strconv.Itoa(port)
	cfg := mediatorConfig(t, addr, scratchDB(t), filepath.Join(t.TempDir(), "artifacts"))

	c := start(t, isolate(t), "--config", cfg)
	c.awaitStderr("mediator started")
	if !strings.Contains(c.stderr.String(), "listen="+addr) {
		t.Fatalf("listen address contract: startup log %q does not name listen=%s", c.stderr.String(), addr)
	}
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		t.Fatalf("listen address contract: nothing accepts on %s although the log says it started: %v", addr, err)
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("closing the probe connection: %v", err)
	}
	if err := c.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("cannot send SIGTERM: %v", err)
	}
	if code := c.wait(); code != 0 {
		t.Fatalf("exit code contract: SIGTERM after a good start = %d, want 0; stderr %q", code, c.stderr.String())
	}
}

func TestBindFailure_ExitsOneAndNeverSaysShuttingDown(t *testing.T) {
	// 알려진 델타 ②가 정확히 이 갈래를 건드린다. 오늘은 ListenAndServe 가
	// 도는 고루틴 안에서 os.Exit(1) 이 나고, 그래서 defer 도 Shutdown 도
	// 안 돈다. 추출하면 그 자리가 오류 채널이 되고 종료가 main 으로 돌아온다.
	//
	// 바뀌어도 되는 것은 순서이고 안 바뀌어야 하는 것은 관측이다. 그래서
	// 셋을 건다 - 종료 코드 1 · "server stopped" · 그리고 "shutting down"
	// 이 나오지 않을 것. 마지막 것이 이 테스트의 날이다: 오류 채널을
	// ctx.Done() 과 같은 자리에서 받으면 정상 종료 경로로 새기 쉽고, 그러면
	// 바인드 실패가 곱게 끝난 것처럼 보인다.
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("cannot occupy a port for the test: %v", err)
	}
	defer occupied.Close() //nolint:errcheck // 테스트가 끝나면 커널이 거둔다
	addr := occupied.Addr().String()

	cfg := mediatorConfig(t, addr, scratchDB(t), filepath.Join(t.TempDir(), "artifacts"))
	code, _, stderr := exec1(t, isolate(t), "--config", cfg)
	if code != 1 {
		t.Fatalf("exit code contract: a port already in use = %d, want 1; stderr %q", code, stderr)
	}
	if !strings.Contains(stderr, "server stopped") {
		t.Fatalf("stderr wording contract: got %q, want %q in it", stderr, "server stopped")
	}
	if !strings.Contains(stderr, "address already in use") {
		t.Fatalf("stderr wording contract: got %q, want the bind error surfaced", stderr)
	}
	if strings.Contains(stderr, "shutting down") {
		t.Fatalf("shutdown contract: a failed bind said %q; that line belongs to the signal path only", stderr)
	}
}

// ── 시그널 ───────────────────────────────────────────────────────────────

func TestSigterm_ShutsDownGracefullyWithExitZero(t *testing.T) {
	// signal.NotifyContext 가 잡고, main 이 "shutting down" 을 찍고,
	// srv.Shutdown 이 10초 안에 끝나고, 프로세스가 0 으로 끝난다.
	//
	// 추출은 os.Exit 을 return 으로 바꾸는 일이고, 시그널 경로는 그 return 이
	// 실제로 프로세스의 종료 코드가 되는지를 보는 유일한 자리다.
	for _, sig := range []syscall.Signal{syscall.SIGTERM, syscall.SIGINT} {
		t.Run(sig.String(), func(t *testing.T) {
			addr := "127.0.0.1:" + strconv.Itoa(freePort(t))
			cfg := mediatorConfig(t, addr, scratchDB(t), filepath.Join(t.TempDir(), "artifacts"))
			c := start(t, isolate(t), "--config", cfg)
			c.awaitStderr("mediator started")
			if err := c.cmd.Process.Signal(sig); err != nil {
				t.Fatalf("cannot send %s: %v", sig, err)
			}
			c.awaitStderr("shutting down")
			if code := c.wait(); code != 0 {
				t.Fatalf("signal contract: %s made the mediator exit %d, want 0; stderr %q",
					sig, code, c.stderr.String())
			}
		})
	}
}
