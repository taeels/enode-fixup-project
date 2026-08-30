package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/taeels/enode/internal/enode"
)

// 기동을 안에서 부른다 - U8 이 main() 에서 뽑아낸 run() 과, 그것이 부르는
// runHookCmd · runSetupCmd 셋이다.
//
// entrypoint_unix_test.go 와 무엇이 다른가 - 그쪽은 프로세스 밖에서 보이는
// 것(종료 코드 · stderr 문구 · 시그널)을 자식 프로세스로 잡고, 이 파일은 같은
// 갈래가 지나가는 문장을 안에서 잡는다. 자식이 덮은 것은 이 패키지의
// 프로파일에 한 문장도 안 들어오므로 둘은 겹치지 않는다.
//
// 왜 빌드 태그가 없는가 - 여기에는 시그널도 자식 프로세스도 없다. 노드를
// 끝내는 것은 --once 이고 (docs/elastic-nodes.md), 그것은 순전히 프로토콜이라
// 어느 OS 에서나 같은 모양으로 돈다. entrypoint_unix_test.go 가 !windows 인
// 이유는 SIGTERM 과 프로세스 그룹이었고 그 이유가 여기에는 없다.
//
// 목을 세우는 자리는 하나뿐이다 - Mediator 다. 파일 시스템은 t.TempDir(),
// 신원은 진짜 git, 잠금은 진짜 flock 을 쓴다.

// ── 관용구 ───────────────────────────────────────────────────────────────

// captureOutput 은 os.Stdout 과 os.Stderr 를 파이프로 바꿔 fn 이 찍은 것을
// 돌려준다.
//
// run() 은 slog 를 os.Stderr 로 만들고 --version 은 fmt.Println 으로 쓴다.
// 출력 대상을 io.Writer 로 바꾸는 안은 cmd/runctl · cmd/enodectl 이 같은
// 이유로 이미 기각했다 - 이음매를 뽑는 일이 CLI 의 모양을 바꾸기 시작하면
// 무엇을 시험하는지가 흐려진다.
//
// 고루틴으로 비우는 이유 - 파이프 버퍼가 차면 fn 이 쓰기에서 멈추고,
// 멈추면 왜 멈췄는지 출력에 아무것도 안 남는다. 도는 노드를 잡는 이 파일에서
// 그 위험이 실제로 있다.
func captureOutput(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe for stdout: %v", err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe for stderr: %v", err)
	}
	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outW, errW

	var ob, eb bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); _, _ = io.Copy(&ob, outR) }()
	go func() { defer wg.Done(); _, _ = io.Copy(&eb, errR) }()

	func() {
		defer func() {
			os.Stdout, os.Stderr = oldOut, oldErr
			outW.Close()
			errW.Close()
		}()
		fn()
	}()
	wg.Wait()
	outR.Close()
	errR.Close()
	return ob.String(), eb.String()
}

// callRun 은 os.Args 를 갈아끼우고 run() 을 부른다.
//
// flag.CommandLine 을 매번 새것으로 바꾸는 이유가 둘이다. 첫째, run() 이 같은
// 이름의 플래그를 전역 집합에 다시 등록하므로 두 번째 호출이 "flag redefined"
// 로 패닉한다. 둘째, 생산의 flag.CommandLine 은 ExitOnError 라 파싱 실패가
// os.Exit(2) 로 테스트 바이너리를 통째로 데리고 나간다.
// cmd/runctl/subcommand_test.go 의 cli.exec 이 같은 이유로 같은 모양이고,
// U8 의 인계가 그것을 그대로 쓰라고 적었다.
func callRun(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	oldArgs, oldFlags := os.Args, flag.CommandLine
	fs := flag.NewFlagSet("enode", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	os.Args, flag.CommandLine = append([]string{"enode"}, args...), fs
	defer func() { os.Args, flag.CommandLine = oldArgs, oldFlags }()
	stdout, stderr = captureOutput(t, func() { code = run() })
	return code, stdout, stderr
}

// isolateNode 는 이 테스트만의 설정 · 상태 자리를 세우고 환경을 비운다.
//
// 격리는 생산 코드를 안 바꾸고 얻는다 - ConfDir 과 StateDir 이
// ENODE_CONFDIR · ENODE_STATEDIR 을 먼저 본다 (internal/enode/paths.go).
// ENODE_CONFIG 와 ENODE_TOKEN 을 함께 비우는 이유는 구체적이다: 둘 중 하나가
// 새어 들어오면 "설정을 못 찾았다" 갈래와 "토큰이 없다" 갈래가 조용히 다른
// 갈래가 된다.
func isolateNode(t *testing.T) string {
	t.Helper()
	conf := t.TempDir()
	t.Setenv("ENODE_CONFDIR", conf)
	t.Setenv("ENODE_STATEDIR", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("ENODE_CONFIG", "")
	t.Setenv("ENODE_TOKEN", "")
	return conf
}

// gitEmail 은 enode.Derive 가 읽는 전역 git 이메일을 고정한다.
//
// email 이 비면 이메일이 없는 기계를 만든다 - ADR-015 §1 이 조용한 대체를
// 금했으므로 그 자리에서 실패하는 것이 계약이고, 그 계약도 시험 대상이다.
func gitEmail(t *testing.T, email string) {
	t.Helper()
	path := os.DevNull
	if email != "" {
		path = filepath.Join(t.TempDir(), "gitconfig")
		if err := os.WriteFile(path, []byte("[user]\n\temail = "+email+"\n"), 0o600); err != nil {
			t.Fatalf("write a git config for the test: %v", err)
		}
	}
	t.Setenv("GIT_CONFIG_GLOBAL", path)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
}

// nodeConfig 는 설정 파일을 만들고 그 경로를 돌려준다.
func nodeConfig(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "local.yaml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

// noSystemNodeConfig 는 시스템 자리가 비어 있음을 단언한다.
//
// 스킵이 아니라 단언인 것이 일부러다. 이 기계에 그 파일이 있으면 "설정을 못
// 찾았다" 갈래가 아예 안 돌고 노드가 뜨려 든다. 경로를 박지 않고
// ConfigPaths() 의 마지막 자리를 꺼내는 이유는 그 목록이 곧 시험 대상이어서다.
func noSystemNodeConfig(t *testing.T) string {
	t.Helper()
	ps := enode.ConfigPaths()
	sys := ps[len(ps)-1]
	if _, err := os.Stat(sys); err == nil {
		t.Fatalf("%s exists on this machine; the no-config branch cannot be measured here", sys)
	}
	return sys
}

func wantIn(t *testing.T, what, got string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Fatalf("%s said %q, want %q in it", what, got, w)
		}
	}
}

// ── 플래그 파싱보다 앞에 있는 것들 ───────────────────────────────────────

func TestRun_VersionAnswersBeforeAnyConfigIsLooked(t *testing.T) {
	// ADR-056 - 설정 파일이 없어도 답해야 한다. 자기 갱신이 받아온 것이
	// 무엇인지를 이것으로 판정하므로, 설정이 없는 기계에서 종료 코드가 0 이
	// 아니면 그 판정 자체가 불가능해진다.
	isolateNode(t)
	noSystemNodeConfig(t)
	for _, spelling := range []string{"--version", "-version"} {
		t.Run(spelling, func(t *testing.T) {
			code, stdout, stderr := callRun(t, spelling)
			if code != 0 {
				t.Fatalf("exit code contract: enode %s = %d, want 0 (ADR-056); stderr %q",
					spelling, code, stderr)
			}
			if !strings.HasPrefix(stdout, "enode ") {
				t.Fatalf("enode %s printed %q, want it to start with the program name", spelling, stdout)
			}
		})
	}
}

func TestRun_SetupIsReachedBeforeAnyConfigIsLooked(t *testing.T) {
	// setup 은 설정이 없어서 부르는 명령이므로 설정을 찾다가 죽는 경로를
	// 지나가면 안 된다 (main.go). 알 수 없는 플래그를 주면 그 자리에서
	// 2 로 끝나는데, 그 2 자체가 "설정 검색을 안 지났다" 의 증거다.
	isolateNode(t)
	noSystemNodeConfig(t)
	code, _, stderr := callRun(t, "setup", "--no-such-flag")
	if code != 2 {
		t.Fatalf("exit code contract: enode setup with a bad flag = %d, want 2; stderr %q", code, stderr)
	}
	if strings.Contains(stderr, "no config file found") {
		t.Fatalf("setup went through the config search: %q", stderr)
	}
}

func TestRun_NoConfig_NamesEveryPlaceItLookedAndExitsOne(t *testing.T) {
	// ADR-015 §2 - 설정 파일이 곧 신원이다. 못 찾았을 때 어디를 봤는지
	// 말하지 않으면 사람이 엉뚱한 경로를 들여다본다.
	conf := isolateNode(t)
	sys := noSystemNodeConfig(t)
	code, _, stderr := callRun(t)
	if code != 1 {
		t.Fatalf("exit code contract: enode with no config = %d, want 1; stderr %q", code, stderr)
	}
	wantIn(t, "enode with no config", stderr,
		"no config file found",
		filepath.Join(conf, "local.yaml"),
		sys,
		"Create one and point --config at it",
		"mediator: http://<mediator-host>:8080",
		"The absolute path of this file is part of the node id")
}

func TestRun_UnknownWordIsAPositionalNotASubcommand(t *testing.T) {
	// enode 의 하위 명령은 hook · setup · --version 셋뿐이고 나머지는
	// 위치인자로 조용히 무시된다. 이것이 오늘의 계약이므로 못박는다.
	isolateNode(t)
	noSystemNodeConfig(t)
	bogusCode, _, bogusErr := callRun(t, "bogus")
	bareCode, _, bareErr := callRun(t)
	if bogusCode != bareCode {
		t.Fatalf("exit code contract: enode bogus = %d but bare enode = %d; an unknown word is a positional",
			bogusCode, bareCode)
	}
	if !strings.Contains(bogusErr, "no config file found") || !strings.Contains(bareErr, "no config file found") {
		t.Fatalf("enode bogus said %q and bare enode said %q; both should reach the same config search",
			bogusErr, bareErr)
	}
}

// ── 설정과 신원 ──────────────────────────────────────────────────────────

func TestRun_ConfigCannotBeRead_ExitsOneAndSaysWhichFile(t *testing.T) {
	// 못 찾은 것과 못 읽는 것을 가른다. --config 로 명시하면 없어도 조용히
	// 넘어가지 않는다 (internal/enode/paths.go).
	isolateNode(t)
	dir := t.TempDir()
	malformed := filepath.Join(dir, "malformed.yaml")
	if err := os.WriteFile(malformed, []byte(": : not yaml [\n"), 0o600); err != nil {
		t.Fatalf("write %s: %v", malformed, err)
	}
	for _, tc := range []struct{ name, path string }{
		{"a config file that is not there", filepath.Join(dir, "absent.yaml")},
		{"a config file that is not yaml", malformed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, _, stderr := callRun(t, "--config", tc.path)
			if code != 1 {
				t.Fatalf("exit code contract: enode --config %s = %d, want 1; stderr %q", tc.path, code, stderr)
			}
			wantIn(t, "enode --config "+tc.path, stderr, "cannot read config", tc.path)
		})
	}
}

func TestRun_MediatorAndTokenAreRequired_ExitsOne(t *testing.T) {
	// ADR-015 §1 - 조용한 대체를 하지 않는다. 주소도 토큰도 없이 뜨면 노드는
	// 아무 데도 못 붙은 채 살아 있게 되고, 그것이 가장 나쁜 상태다.
	for _, tc := range []struct{ name, body string }{
		{"neither", "workspace: /tmp\n"},
		{"only the mediator", "mediator: http://127.0.0.1:1\n"},
		{"only the token", "token: t\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolateNode(t)
			code, _, stderr := callRun(t, "--config", nodeConfig(t, tc.body))
			if code != 1 {
				t.Fatalf("exit code contract: a config with %s = %d, want 1; stderr %q", tc.name, code, stderr)
			}
			wantIn(t, "a config with "+tc.name, stderr, "mediator and token are required")
		})
	}
}

func TestRun_FlagsAndTheEnvironmentFillInWhatTheConfigLacks(t *testing.T) {
	// 여기 통과했다는 것을 무엇으로 보는가 - 그다음 갈림인 신원 유도에서
	// 죽는 것으로 본다. 이메일을 지운 기계에서 "cannot derive node identity"
	// 가 나오면 주소와 토큰은 이미 채워진 것이다. "mediator and token are
	// required" 가 나오면 채우기가 안 먹은 것이고, 두 메시지가 갈리므로
	// 이 단언은 그 자리를 정확히 가른다.
	for _, tc := range []struct {
		name, body string
		args       []string
		env        map[string]string
	}{
		{"--mediator fills in the address", "token: t\n", []string{"--mediator", "http://127.0.0.1:1"}, nil},
		{"--token fills in the token", "mediator: http://127.0.0.1:1\n", []string{"--token", "t"}, nil},
		{"$ENODE_TOKEN fills in the token", "mediator: http://127.0.0.1:1\n", nil,
			map[string]string{"ENODE_TOKEN": "from-the-environment"}},
		{"--token wins over $ENODE_TOKEN", "mediator: http://127.0.0.1:1\n", []string{"--token", "t"},
			map[string]string{"ENODE_TOKEN": "from-the-environment"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolateNode(t)
			gitEmail(t, "")
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			args := append([]string{"--config", nodeConfig(t, tc.body)}, tc.args...)
			code, _, stderr := callRun(t, args...)
			if code != 1 {
				t.Fatalf("exit code contract: no git email = %d, want 1; stderr %q", code, stderr)
			}
			if strings.Contains(stderr, "mediator and token are required") {
				t.Fatalf("%s did not reach the node: %q", tc.name, stderr)
			}
			wantIn(t, tc.name, stderr, "cannot derive node identity")
		})
	}
}

func TestRun_AnotherNodeHoldsTheLock_ExitsOne(t *testing.T) {
	// 중복 실행은 로컬에서 막는다 (ADR-015 §2) - Mediator 는 재시작과 중복을
	// 구분할 정보가 없다. 잠금을 먼저 잡아 두고 부르면 그 갈래가 선다.
	//
	// flock 은 열린 파일 기술자에 걸리지 프로세스에 걸리지 않는다. 그래서
	// 같은 프로세스 안에서 다른 기술자로 다시 잡으려 하면 실제로 막힌다 -
	// 자식 프로세스를 띄우지 않고도 이 계약을 잴 수 있는 이유다.
	isolateNode(t)
	gitEmail(t, "u11-node@example.invalid")
	cfg := nodeConfig(t, "mediator: http://127.0.0.1:1\ntoken: t\n")

	ident, err := enode.Derive(cfg)
	if err != nil {
		t.Fatalf("cannot derive the identity the node will derive: %v", err)
	}
	held, err := enode.Acquire(ident.Config)
	if err != nil {
		t.Fatalf("cannot take the lock before the node does: %v", err)
	}
	t.Cleanup(func() { _ = held.Release() })

	code, _, stderr := callRun(t, "--config", cfg)
	if code != 1 {
		t.Fatalf("exit code contract: a lock already held = %d, want 1; stderr %q", code, stderr)
	}
	wantIn(t, "a lock already held", stderr,
		"cannot acquire lock",
		"another enode is already running with this config")
}

// ── 도는 노드 ────────────────────────────────────────────────────────────

// fakeMediator 는 노드가 붙는 반대편이다.
//
// 첫 claim 에만 일감을 하나 주고 그 뒤로는 롱폴을 붙잡는다. 실어 보내는
// 임대의 not_after 는 이미 지난 시각이다 - Held.Add 가 "한 번은 집었다" 를
// 세우고, 그다음 execute 가 ADR-010 의 확인에서 멈춰 하네스를 안 띄운다.
// --once 의 조건(집은 적이 있고 지금 임대가 없다)을 그렇게 만든다.
type fakeMediator struct {
	srv *httptest.Server

	mu                  sync.Mutex
	adverts, claims     int
	auth, principal     string
	instance, userAgent string
}

func newFakeMediator(t *testing.T) *fakeMediator {
	t.Helper()
	m := &fakeMediator{}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		m.record(r)
		m.mu.Lock()
		m.adverts++
		m.mu.Unlock()
		m.writeJSON(t, w, map[string]any{"leases": []any{}, "renew_seconds": 0})
	})
	mux.HandleFunc("/v1/nodes/", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/claim") {
			http.NotFound(w, r)
			return
		}
		m.record(r)
		m.mu.Lock()
		m.claims++
		first := m.claims == 1
		m.mu.Unlock()
		if first {
			m.writeJSON(t, w, map[string]any{
				"step_id": "u11-run#01", "run_id": "u11-run", "seq": 1,
				"name": "noop", "kind": "command",
				"lease": map[string]any{
					"run_id": "u11-run", "node": "n", "capability": "agent.reason",
					"not_after": time.Now().Add(-time.Minute).Format(time.RFC3339Nano),
					"nonce":     "u11",
				},
			})
			return
		}
		// 롱폴이다 (ADR-029) - 노드가 멈출 때까지 잡고 있는다. 204 를 즉시
		// 돌려주면 워커가 그 사이를 꽉 채워 도느라 광고가 굶는다.
		<-r.Context().Done()
	})
	m.srv = httptest.NewServer(mux)
	t.Cleanup(m.srv.Close)
	return m
}

func (m *fakeMediator) record(r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.auth = r.Header.Get("Authorization")
	m.principal = r.Header.Get("X-Enode-Principal")
	if v := r.Header.Get("X-Enode-Instance"); v != "" {
		m.instance = v
	}
	m.userAgent = r.Header.Get("User-Agent")
}

func (m *fakeMediator) writeJSON(t *testing.T, w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Errorf("encoding the fake mediator response: %v", err)
	}
}

func (m *fakeMediator) counts() (adverts, claims int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.adverts, m.claims
}

func TestRun_AdvertisesThenExitsWhenTheRunIsDone(t *testing.T) {
	// 노드가 실제로 서는 유일한 갈래다 - 신원 · 잠금 · 이번 생의 표식 ·
	// 두 고루틴 · 종료까지 한 번에 지난다.
	//
	// 왜 --once 로 끝내는가 - 시그널 없이 되돌아오는 유일한 길이고
	// (docs/elastic-nodes.md §4.4), 시그널의 실제 도달은 프로세스 밖에서만
	// 관측되므로 entrypoint_unix_test.go 가 이미 그쪽을 들고 있다.
	//
	// 언제 끝나는가 - 광고가 "집은 적이 있는데 지금 임대가 없다" 를 처음
	// 보는 순간이다. claim 이 광고 사이 어디에 떨어지느냐에 따라 광고 한
	// 주기가 더 돌 수 있고, 그래서 주기를 짧게 준다. 붙잡히는 일은 없다 -
	// 임대는 한 번 집힌 뒤 다시 늘어나지 않으므로 다음 광고가 반드시 본다.
	for _, tc := range []struct {
		name        string
		readyFile   func(t *testing.T) string
		wantWritten bool
		wantLog     string
	}{
		{
			"it writes the ready file once the fleet knows about it",
			func(t *testing.T) string { return filepath.Join(t.TempDir(), "ready") },
			true,
			"msg=ready",
		},
		{
			"a ready file it cannot write does not stop the node",
			func(t *testing.T) string {
				// 파일 아래의 경로는 디렉터리가 아니라 못 만든다. 노드는
				// 이미 함대에 있으므로 막지 않는 것이 계약이다 (main.go).
				blocker := filepath.Join(t.TempDir(), "blocker")
				if err := os.WriteFile(blocker, []byte("not a directory\n"), 0o600); err != nil {
					t.Fatalf("write %s: %v", blocker, err)
				}
				return filepath.Join(blocker, "ready")
			},
			false,
			"cannot write ready file",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolateNode(t)
			gitEmail(t, "u11-node@example.invalid")
			m := newFakeMediator(t)
			ready := tc.readyFile(t)
			cfg := nodeConfig(t, "mediator: "+m.srv.URL+"\ntoken: u11-token\n")

			code, _, stderr := callRun(t, "--config", cfg, "--once", "--debug",
				"--every", "150ms", "--ready-file", ready)

			if code != 0 {
				t.Fatalf("exit code contract: a node that finished its run = %d, want 0; stderr %q", code, stderr)
			}
			wantIn(t, "a node that finished its run", stderr,
				"enode started",
				"lease is not valid; not running",
				"the run finished and no lease remains; exiting (--once)",
				"msg=stopped",
				tc.wantLog)

			adverts, claims := m.counts()
			if adverts < 1 {
				t.Fatal("the fleet never heard from the node; nothing could have matched it")
			}
			if claims < 1 {
				t.Fatalf("the node never claimed; --once cannot tell a finished run from a fresh start without it")
			}
			if m.auth != "Bearer u11-token" {
				t.Fatalf("the token crossed the boundary as %q, want %q", m.auth, "Bearer u11-token")
			}
			if m.principal != "u11-node@example.invalid" {
				t.Fatalf("the principal crossed the boundary as %q, want the git email", m.principal)
			}
			// 이번 생의 표식 (ADR-030) - 기동마다 새로 뽑히므로 값은 못
			// 박지만, 비어 있으면 유실된 claim 의 재전달과 재시작 판정이
			// 갈리지 않는다.
			if m.instance == "" {
				t.Fatal("the claim carried no X-Enode-Instance; ADR-030 needs it to tell a restart from a retry")
			}

			written, err := os.ReadFile(ready)
			if tc.wantWritten {
				if err != nil {
					t.Fatalf("read back %s: %v", ready, err)
				}
				if strings.TrimSpace(string(written)) == "" {
					t.Fatalf("%s is empty; it should name the node that is ready", ready)
				}
				if !strings.Contains(stderr, strings.TrimSpace(string(written))) {
					t.Fatalf("%s names %q but the log never mentions that node", ready, string(written))
				}
			} else if err == nil {
				t.Fatalf("%s was written although its directory is a file", ready)
			}
		})
	}
}

// ── 훅 ───────────────────────────────────────────────────────────────────

func TestRunHookCmd_WithoutStop_PrintsUsageAndExitsTwo(t *testing.T) {
	// 훅은 절대 하네스를 막지 않는다 - 그래서 무슨 일이 있어도 0 인데,
	// 사용법을 몰라 부른 것은 예외다 (hook.go). 이 2 가 0 으로 바뀌면
	// 오타를 낸 사람이 훅이 돌았다고 믿는다.
	for _, args := range [][]string{{"hook"}, {"hook", "bogus"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			isolateNode(t)
			noSystemNodeConfig(t)
			code, _, stderr := callRun(t, args...)
			if code != 2 {
				t.Fatalf("exit code contract: enode %v = %d, want 2; stderr %q", args, code, stderr)
			}
			wantIn(t, strings.Join(args, " "), stderr, "usage: enode hook stop --out <dir>")
		})
	}
}

func TestRunHookCmd_BadFlagsDoNotBlockTheHarness(t *testing.T) {
	// 인자가 이상해도 0 이다 (hook.go). 안전망이 정규 경로를 무너뜨리는 것이
	// 가장 나쁘다.
	var code int
	captureOutput(t, func() { code = runHookCmd([]string{"stop", "--no-such-flag"}) })
	if code != 0 {
		t.Fatalf("exit code contract: enode hook stop with a bad flag = %d, want 0", code)
	}
}

// withStdin 은 fn 이 도는 동안 os.Stdin 을 그 내용으로 바꾼다.
//
// runHookCmd 가 os.Stdin 을 그대로 RunStopHook 에 넘기고, RunStopHook 은
// 그것을 못 읽으면 그 자리에서 통과시킨다 (internal/enode/hook.go). 그래서
// 훅이 실제로 판정하는 갈래는 stdin 에 진짜 입력이 있을 때만 선다.
func withStdin(t *testing.T, content string, fn func()) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "stdin.json")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	f, err := os.Open(p)
	if err != nil {
		t.Fatalf("open %s: %v", p, err)
	}
	defer f.Close() //nolint:errcheck // 읽기만 했다
	old := os.Stdin
	os.Stdin = f
	defer func() { os.Stdin = old }()
	fn()
}

func TestRunHookCmd_BlocksOnAMissingOutputAndDropsEmptyNames(t *testing.T) {
	// 계약이 요구한 산출물이 없으면 훅이 되묻는다 (ADR-051). 이름 목록을
	// 쉼표로 가르는 자리에 빈 칸이 섞여도 그것이 이름이 되면 안 된다 -
	// 되면 훅이 "이름 없는 산출물이 없다" 로 영원히 막는다.
	out := t.TempDir()
	var code int
	var stdout string
	withStdin(t, `{"session_id":"s"}`+"\n", func() {
		stdout, _ = captureOutput(t, func() {
			code = runHookCmd([]string{"stop", "--out", out,
				"--expect", "plan.json, ,  ", "--roles", "builder, ,tester",
				"--workspace", t.TempDir()})
		})
	})
	if code != 0 {
		t.Fatalf("exit code contract: a blocking hook = %d, want 0; the hook never fails the harness", code)
	}
	if !strings.Contains(stdout, `"decision":"block"`) {
		t.Fatalf("the hook answered %q, want it to block on the missing output", stdout)
	}
	if !strings.Contains(stdout, "plan.json") {
		t.Fatalf("the hook answered %q, want the missing name in the reason", stdout)
	}
	if strings.Contains(stdout, `\"\"`) {
		t.Fatalf("the hook counted a blank as an output name: %q", stdout)
	}
}

func TestRunHookCmd_PassesWhenTheContractAsksForNothing(t *testing.T) {
	// 요구한 산출물이 없으면 짚을 것도 없다. 통과는 조용해야 한다 -
	// 아무것도 안 찍는 것이 하네스에게 "계속하라" 는 뜻이다.
	var code int
	var stdout, stderr string
	withStdin(t, `{"session_id":"s"}`+"\n", func() {
		stdout, stderr = captureOutput(t, func() {
			code = runHookCmd([]string{"stop", "--out", t.TempDir()})
		})
	})
	if code != 0 {
		t.Fatalf("exit code contract: a passing hook = %d, want 0", code)
	}
	if stdout != "" || stderr != "" {
		t.Fatalf("a passing hook said stdout %q stderr %q, want silence", stdout, stderr)
	}
}

func TestRunHookCmd_CannotWriteItsAnswer_StillDoesNotFail(t *testing.T) {
	// 훅이 답을 못 써도 0 이다. 그 자리가 실패로 바뀌면 하네스가 멈춰 서고,
	// 안전망이 정규 경로를 무너뜨리는 것이 가장 나쁘다 (hook.go).
	//
	// 닫힌 파이프로 몬다 - RunStopHook 이 오류를 내는 자리는 답을 쓰는
	// 곳뿐이므로, 막을 것이 있는 판을 세우고 출구를 닫는다.
	out := t.TempDir()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	r.Close() //nolint:errcheck // 쓰기가 실패하게 만드는 것이 목적이다
	w.Close() //nolint:errcheck

	var code int
	var stderr string
	withStdin(t, `{"session_id":"s"}`+"\n", func() {
		oldOut := os.Stdout
		os.Stdout = w
		defer func() { os.Stdout = oldOut }()
		_, stderr = captureOutput(t, func() {
			// captureOutput 이 os.Stderr 만 필요하다 - os.Stdout 은 방금
			// 닫힌 것으로 바꿔 두었고 captureOutput 이 그것을 다시
			// 되돌려 놓는다. 그래서 안쪽에서 한 번 더 갈아끼운다.
			os.Stdout = w
			code = runHookCmd([]string{"stop", "--out", out, "--expect", "plan.json"})
		})
	})
	if code != 0 {
		t.Fatalf("exit code contract: a hook that cannot write = %d, want 0", code)
	}
	wantIn(t, "a hook that cannot write", stderr, "hook error (ignored):")
}
