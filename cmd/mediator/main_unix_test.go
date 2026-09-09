//go:build !windows

package main

import (
	"context"
	"flag"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/taeels/enode/internal/config"
	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/store"
)

// 기동의 이음매를 안에서 부른다 - U8 이 main() 에서 뽑아낸 run · loadConfig ·
// bootstrapToken · openStore · openRecords · migrate · serve 일곱이다.
//
// entrypoint_unix_test.go 와 무엇이 다른가 - 그쪽은 프로세스 밖에서 보이는
// 것(종료 코드 · stderr 문구 · 리슨 주소 · 시그널)을 자식 프로세스로 잡고,
// 이 파일은 같은 갈래가 지나가는 문장을 안에서 잡는다. 자식이 덮은 것은
// 이 패키지의 프로파일에 한 문장도 안 들어오므로 둘은 겹치지 않는다.
//
// ok 가 false 인 이음매는 반환값이 아니라 찍힌 것을 본다. 오류를 돌려주고
// 부르는 쪽이 찍게 하는 안은 U8 이 기각했다 (main.go 의 loadConfig 주석) -
// 문구가 갈래마다 다르고 그 문구가 곧 계약이기 때문이다. 그래서 여기서도
// 반환값과 함께 stderr 를 건다.
//
// 왜 빌드 태그로 가르는가는 setup_unix_test.go 의 머리글에 적었다.

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
	fs := flag.NewFlagSet("mediator", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	os.Args, flag.CommandLine = append([]string{"mediator"}, args...), fs
	defer func() { os.Args, flag.CommandLine = oldArgs, oldFlags }()
	stdout, stderr = captureOutput(t, func() { code = run() })
	return code, stdout, stderr
}

// cleanEnv 는 mediator 가 파일보다 위에 두는 환경변수를 비운다.
//
// config.Load 는 DATABASE_URL 과 ENODE_MEDIATOR_TOKEN 이 파일을 이기게 한다
// (ADR-015 §4). 이 테스트를 돌리는 기계에 둘 중 하나라도 있으면 "설정을 못
// 찾았다" 갈래와 "토큰이 없다" 갈래가 조용히 다른 갈래가 된다.
func cleanEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"DATABASE_URL", "ENODE_MEDIATOR_TOKEN", "ENODE_MEDIATOR_CONFIG"} {
		t.Setenv(k, "")
	}
}

// logTo 는 지금의 os.Stderr 로 찍는 로거다. captureOutput 안에서 만들어야
// 파이프로 바뀐 os.Stderr 를 잡는다 - run() 이 하는 것과 같은 모양이다.
func logTo() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

// awaitDial 은 그 주소가 연결을 받을 때까지 기다린다.
//
// 서기 전에 취소하면 Shutdown 이 무엇을 닫는지가 실행마다 갈린다. 리슨이
// 실제로 섰다는 것을 확인하고 나서 취소하는 것이 그 경합을 없앤다.
func awaitDial(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			conn.Close() //nolint:errcheck // 살아 있는지만 봤다
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("nothing ever accepted on %s: %v", addr, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// ── 플래그 파싱보다 앞에 있는 것들 ───────────────────────────────────────

func TestRun_VersionAnswersOnAllThreeSpellings(t *testing.T) {
	// ADR-056 - 설정도 DB 도 없이 답한다. 세 철자를 다 거는 것은 main.go 의
	// switch 가 셋을 다 받기 때문이고, 하나가 빠지면 그것으로 스크립트를 짠
	// 사람이 조용히 다른 갈래로 간다.
	cleanEnv(t)
	for _, spelling := range []string{"--version", "-version", "version"} {
		t.Run(spelling, func(t *testing.T) {
			code, stdout, stderr := callRun(t, spelling)
			if code != 0 {
				t.Fatalf("exit code contract: mediator %s = %d, want 0 (ADR-056); stderr %q",
					spelling, code, stderr)
			}
			if !strings.HasPrefix(stdout, "mediator ") {
				t.Fatalf("mediator %s printed %q, want it to start with the program name", spelling, stdout)
			}
		})
	}
}

func TestRun_SetupIsReachedBeforeAnyConfigIsLoaded(t *testing.T) {
	// setup 은 설정도 DB 도 없는 상태에서 불린다. Load 를 지나면 "설정을 못
	// 찾았다" 로 죽고, 그러면 설정을 만들라고 있는 명령이 설정이 없어서
	// 못 돈다 (main.go 의 주석이 그 이유를 적는다).
	cleanEnv(t)
	code, _, stderr := callRun(t, "setup", "--check", "--admin-url",
		"postgres://nobody:nobody@127.0.0.1:1/postgres?sslmode=disable")
	if code != 1 {
		t.Fatalf("exit code contract: mediator setup against a dead port = %d, want 1; stderr %q", code, stderr)
	}
	if !strings.Contains(stderr, "cannot reach postgres:") {
		t.Fatalf("stderr wording contract: got %q, want the setup failure, not a config failure", stderr)
	}
	if strings.Contains(stderr, "no config file found") {
		t.Fatalf("setup went through the config search: %q", stderr)
	}
}

// ── 설정 ─────────────────────────────────────────────────────────────────

func TestLoadConfig_CannotReadIt_SaysWhichFile(t *testing.T) {
	cleanEnv(t)
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
			var ok bool
			var file string
			_, stderr := captureOutput(t, func() {
				_, file, ok = loadConfig(tc.path, logTo())
			})
			if ok {
				t.Fatalf("loadConfig(%s) said ok; want it to refuse", tc.path)
			}
			if file != "" {
				t.Fatalf("loadConfig(%s) named %q as the file it read; want none", tc.path, file)
			}
			if !strings.Contains(stderr, "cannot read config") || !strings.Contains(stderr, tc.path) {
				t.Fatalf("stderr wording contract: got %q, want %q with the offending path named",
					stderr, "cannot read config")
			}
		})
	}
}

func TestLoadConfig_NoFileFound_NamesEveryPlaceItLookedAndPointsAtSetup(t *testing.T) {
	// 못 찾은 것과 토큰만 빠진 것을 가르는 것이 기동의 첫 갈림이다. 예전에는
	// 둘이 같은 메시지로 나와 사람이 토큰만 들여다봤다 (main.go).
	cleanEnv(t)
	noSystemConfig(t)
	e := isolate(t)
	t.Setenv("HOME", e.home)

	var ok bool
	_, stderr := captureOutput(t, func() { _, _, ok = loadConfig("", logTo()) })

	if ok {
		t.Fatal("loadConfig with no config anywhere said ok; want it to refuse")
	}
	for _, want := range []string{
		"no config file found",
		e.userConfigPath(),
		"/etc/enode-mediator/config.yaml",
		"Run `mediator setup` to create one.",
	} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr wording contract: got %q, want %q in it", stderr, want)
		}
	}
}

func TestLoadConfig_EnvTokenStandsInForTheFile(t *testing.T) {
	// 파일이 없어도 $ENODE_MEDIATOR_TOKEN 이 있으면 돈다 (ADR-015 §4 - 비밀은
	// 환경변수가 이긴다). 이 갈래가 막히면 파일 없이 컨테이너로 띄우는 길이
	// 통째로 사라진다.
	cleanEnv(t)
	noSystemConfig(t)
	e := isolate(t)
	t.Setenv("HOME", e.home)
	t.Setenv("ENODE_MEDIATOR_TOKEN", "from-the-environment")

	var ok bool
	var cfg config.Config
	var file string
	_, stderr := captureOutput(t, func() { cfg, file, ok = loadConfig("", logTo()) })

	if !ok {
		t.Fatalf("loadConfig with a token in the environment refused; stderr %q", stderr)
	}
	if file != "" {
		t.Fatalf("loadConfig named %q as the file it read; there is no file", file)
	}
	if cfg.Token != "from-the-environment" {
		t.Fatalf("the token is %q, want the one from the environment", cfg.Token)
	}
}

func TestLoadConfig_ReturnsTheFileItRead(t *testing.T) {
	cleanEnv(t)
	cfgPath := mediatorConfig(t, "127.0.0.1:8080", "postgres:///nope", filepath.Join(t.TempDir(), "artifacts"))

	var ok bool
	var cfg config.Config
	var file string
	_, stderr := captureOutput(t, func() { cfg, file, ok = loadConfig(cfgPath, logTo()) })

	if !ok {
		t.Fatalf("loadConfig on a good config refused; stderr %q", stderr)
	}
	if file != cfgPath {
		t.Fatalf("loadConfig named %q as the file it read, want %q", file, cfgPath)
	}
	if cfg.Listen != "127.0.0.1:8080" || cfg.Token != "u8-token" {
		t.Fatalf("the config it returned is listen=%q token=%q, want the values in the file", cfg.Listen, cfg.Token)
	}
}

// ── 토큰 ─────────────────────────────────────────────────────────────────

func TestBootstrapToken_GeneratesOneAndWritesItBack(t *testing.T) {
	// ADR-015 §1 은 인증을 건너뛰는 것을 막는 것이지 부트스트랩을 막는 것이
	// 아니다. 조용한 대체가 아니라는 것이 이 갈래의 계약이므로 셋을 다 본다:
	// 만들었다는 로그 · 화면의 값 · 파일에 남은 값.
	cleanEnv(t)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("listen: 127.0.0.1:8080\n"), 0o600); err != nil {
		t.Fatalf("write %s: %v", cfgPath, err)
	}
	cfg := config.Default()

	var ok bool
	_, stderr := captureOutput(t, func() { ok = bootstrapToken(&cfg, cfgPath, logTo()) })

	if !ok {
		t.Fatalf("bootstrapToken refused although it could write; stderr %q", stderr)
	}
	if cfg.Token == "" {
		t.Fatal("bootstrapToken said ok but left the token empty")
	}
	if !strings.Contains(stderr, "generated a token and wrote it to the config") {
		t.Fatalf("stderr wording contract: got %q, want the generation announced", stderr)
	}
	if !strings.Contains(stderr, "different token is rejected with 401") {
		t.Fatalf("stderr wording contract: got %q, want the token printed with what it is for", stderr)
	}
	written, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read back %s: %v", cfgPath, err)
	}
	if !strings.Contains(string(written), "token: "+cfg.Token) {
		t.Fatalf("the token in memory is %q but %s holds %q", cfg.Token, cfgPath, string(written))
	}
}

func TestBootstrapToken_LeavesAnExistingTokenAlone(t *testing.T) {
	// 있는 토큰을 갈아치우면 이미 그 값으로 도는 노드 전부가 401 이 된다.
	cleanEnv(t)
	cfgPath := mediatorConfig(t, "127.0.0.1:8080", "postgres:///nope", t.TempDir())
	cfg := config.Default()
	cfg.Token = "already-here"

	var ok bool
	_, stderr := captureOutput(t, func() { ok = bootstrapToken(&cfg, cfgPath, logTo()) })

	if !ok {
		t.Fatalf("bootstrapToken refused a config that already has a token; stderr %q", stderr)
	}
	if cfg.Token != "already-here" {
		t.Fatalf("the token became %q; an existing token must survive", cfg.Token)
	}
	if strings.Contains(stderr, "generated a token") {
		t.Fatalf("it announced a generation that must not have happened: %q", stderr)
	}
}

func TestBootstrapToken_NoPlaceToPutOne_Refuses(t *testing.T) {
	// ADR-015 - 조용한 대체를 하지 않는다. 만들 자리조차 없으면 그 자리에서
	// 죽는다. 두 갈래 다 같은 자리로 떨어져야 한다: 파일이 아예 없는 경우와,
	// 이름은 있는데 못 쓰는 경우.
	cleanEnv(t)
	for _, tc := range []struct{ name, file string }{
		{"no config file at all", ""},
		{"a config file that is not there", filepath.Join(t.TempDir(), "absent.yaml")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Default()
			var ok bool
			_, stderr := captureOutput(t, func() { ok = bootstrapToken(&cfg, tc.file, logTo()) })
			if ok {
				t.Fatalf("bootstrapToken said ok with %q as the config file and no token", tc.file)
			}
			if !strings.Contains(stderr, "no token: set token in the config file or $ENODE_MEDIATOR_TOKEN") {
				t.Fatalf("stderr wording contract: got %q, want the token demand", stderr)
			}
		})
	}
}

// ── 저장소 ───────────────────────────────────────────────────────────────

func TestOpenStore_UnreachableDatabase_Refuses(t *testing.T) {
	cfg := config.Default()
	cfg.Database.URL = "postgres://nobody:nobody@127.0.0.1:1/nope?sslmode=disable"

	var st *store.Store
	var ok bool
	_, stderr := captureOutput(t, func() {
		st, ok = openStore(context.Background(), cfg, logTo())
	})

	if ok || st != nil {
		t.Fatalf("openStore against a dead port returned (%v, %v); want (nil, false)", st, ok)
	}
	if !strings.Contains(stderr, "cannot open database") {
		t.Fatalf("stderr wording contract: got %q, want %q in it", stderr, "cannot open database")
	}
}

func TestOpenStore_OpensTheDatabaseTheConfigNames(t *testing.T) {
	cfg := config.Default()
	cfg.Database.URL = scratchDB(t)

	var st *store.Store
	var ok bool
	_, stderr := captureOutput(t, func() {
		st, ok = openStore(context.Background(), cfg, logTo())
	})

	if !ok || st == nil {
		t.Fatalf("openStore on a real database returned (%v, %v); stderr %q", st, ok, stderr)
	}
	t.Cleanup(st.Close)
}

func TestOpenRecords_WiresEverythingTheContractDoesNotCarry(t *testing.T) {
	// Run Record 는 DB 가 아니라 파일시스템에 산다 (ADR-015 §3), 그리고
	// 계약이 얼마나 자랄 수 있는가는 계약 밖에서 정한다 (ADR-031). 배선이
	// 빠지면 그 한도가 0 이 되어 계약이 한 번도 못 자란다.
	root := filepath.Join(t.TempDir(), "artifacts", "deeper")
	cfg := config.Default()
	cfg.Artifacts.Root = root
	cfg.Contract.MaxVersions = 7
	cfg.Lease.MaxPerRun = 3
	cfg.Notify.AsksURL = "http://127.0.0.1:9/hook"

	ctx := context.Background()
	st, err := store.Open(ctx, scratchDB(t))
	if err != nil {
		t.Fatalf("cannot open the scratch database: %v", err)
	}
	t.Cleanup(st.Close)

	var ok bool
	_, stderr := captureOutput(t, func() { ok = openRecords(st, cfg, logTo()) })

	if !ok {
		t.Fatalf("openRecords refused a writable root; stderr %q", stderr)
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("openRecords said ok but %s is not there: %v", root, err)
	}
	if st.Records == nil {
		t.Fatal("openRecords left st.Records nil; the run record has nowhere to live (ADR-015 §3)")
	}
	if st.Log == nil {
		t.Fatal("openRecords left st.Log nil; a rollback would then be invisible from outside")
	}
	if st.MaxContractVersions != 7 || st.MaxLeasesPerRun != 3 {
		t.Fatalf("the limits are contract=%d leases=%d, want 7 and 3 (ADR-031)",
			st.MaxContractVersions, st.MaxLeasesPerRun)
	}
	if st.NotifyURL != cfg.Notify.AsksURL {
		t.Fatalf("the asks webhook is %q, want %q", st.NotifyURL, cfg.Notify.AsksURL)
	}
}

func TestOpenRecords_RootCannotBeCreated_Refuses(t *testing.T) {
	// ADR-015 §3 - 그 자리를 못 만들면 I4(봉인)를 강제할 데가 없으므로
	// 뜨면 안 된다.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("not a directory\n"), 0o600); err != nil {
		t.Fatalf("write %s: %v", blocker, err)
	}
	cfg := config.Default()
	cfg.Artifacts.Root = filepath.Join(blocker, "artifacts")

	ctx := context.Background()
	st, err := store.Open(ctx, scratchDB(t))
	if err != nil {
		t.Fatalf("cannot open the scratch database: %v", err)
	}
	t.Cleanup(st.Close)

	var ok bool
	_, stderr := captureOutput(t, func() { ok = openRecords(st, cfg, logTo()) })

	if ok {
		t.Fatal("openRecords said ok for a root it cannot create")
	}
	if !strings.Contains(stderr, "cannot create artifacts directory") {
		t.Fatalf("stderr wording contract: got %q, want %q in it", stderr, "cannot create artifacts directory")
	}
	if st.Records != nil {
		t.Fatal("openRecords wired the record root although it could not create it")
	}
}

func TestMigrate_AppliesTheSchemaAndIsIdempotent(t *testing.T) {
	// 멱등이라 매 기동마다 돈다 (main.go). 두 번 돌려 둘 다 참이어야
	// 그 주장이 성립한다.
	ctx := context.Background()
	st, err := store.Open(ctx, scratchDB(t))
	if err != nil {
		t.Fatalf("cannot open the scratch database: %v", err)
	}
	t.Cleanup(st.Close)

	var first, second bool
	_, stderr := captureOutput(t, func() {
		first = migrate(ctx, st, logTo())
		second = migrate(ctx, st, logTo())
	})
	if !first || !second {
		t.Fatalf("migrate returned %v then %v on a real database; stderr %q", first, second, stderr)
	}
}

func TestMigrate_SchemaCannotBeApplied_Refuses(t *testing.T) {
	// 닫힌 풀로 몬다 - 판을 지우는 안보다 경합이 없다.
	ctx := context.Background()
	st, err := store.Open(ctx, scratchDB(t))
	if err != nil {
		t.Fatalf("cannot open the scratch database: %v", err)
	}
	st.Close()

	var ok bool
	_, stderr := captureOutput(t, func() { ok = migrate(ctx, st, logTo()) })

	if ok {
		t.Fatal("migrate said ok through a closed pool")
	}
	if !strings.Contains(stderr, "cannot apply database schema") {
		t.Fatalf("stderr wording contract: got %q, want %q in it", stderr, "cannot apply database schema")
	}
}

// ── 서버 ─────────────────────────────────────────────────────────────────

func TestServe_ShutsDownOnTheContextAndReturnsZero(t *testing.T) {
	// 정상 종료 경로의 표식이 "shutting down" 이다 (main.go 의 serve 주석).
	// 그 줄과 종료 코드 0 이 함께 나오는 것이 이 갈래의 계약이다.
	addr := "127.0.0.1:" + strconv.Itoa(freePort(t))
	srv := &http.Server{Addr: addr, Handler: http.NewServeMux(), ReadHeaderTimeout: 10 * time.Second}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var code int
	_, stderr := captureOutput(t, func() {
		done := make(chan struct{})
		go func() { defer close(done); code = serve(ctx, srv, addr, logTo()) }()
		awaitDial(t, addr)
		cancel()
		<-done
	})

	if code != 0 {
		t.Fatalf("exit code contract: a cancelled context = %d, want 0; stderr %q", code, stderr)
	}
	for _, want := range []string{"mediator started", "listen=" + addr, "shutting down"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr wording contract: got %q, want %q in it", stderr, want)
		}
	}
	if strings.Contains(stderr, "server stopped") {
		t.Fatalf("a clean shutdown said %q; that line belongs to the failure path only", stderr)
	}
}

func TestServe_BindFailure_ReturnsOneAndNeverSaysShuttingDown(t *testing.T) {
	// 오류 채널을 ctx.Done() 과 같은 자리에서 받으면 실패한 바인드가 곱게
	// 끝난 것처럼 보인다. "shutting down" 이 안 나오는 것이 그 구분이다.
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("cannot occupy a port for the test: %v", err)
	}
	defer occupied.Close() //nolint:errcheck // 테스트가 끝나면 커널이 거둔다
	addr := occupied.Addr().String()

	srv := &http.Server{Addr: addr, Handler: http.NewServeMux(), ReadHeaderTimeout: 10 * time.Second}
	var code int
	_, stderr := captureOutput(t, func() {
		code = serve(context.Background(), srv, addr, logTo())
	})

	if code != 1 {
		t.Fatalf("exit code contract: a port already in use = %d, want 1; stderr %q", code, stderr)
	}
	for _, want := range []string{"server stopped", "address already in use"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr wording contract: got %q, want %q in it", stderr, want)
		}
	}
	if strings.Contains(stderr, "shutting down") {
		t.Fatalf("a failed bind said %q; that line belongs to the signal path only", stderr)
	}
}

// ── 기동 전체 ────────────────────────────────────────────────────────────

func TestRun_GoesAllTheWayToServeAndCarriesTheBindFailureOut(t *testing.T) {
	// 이 갈래가 run() 의 몸통을 통째로 지난다 - 설정 · 토큰 · DB · 아티팩트
	// 루트 · 스키마 · 감시자 · 서버. 바인드로 끝내는 이유는 시그널 없이
	// 되돌아오는 유일한 길이어서다.
	//
	// 종료 코드가 serve 에서 run() 으로 실제로 흘러나오는지가 여기서 보인다.
	// U8 이 고루틴 안의 os.Exit(1) 을 오류 채널로 옮긴 것이 바꾼 자리다.
	cleanEnv(t)
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("cannot occupy a port for the test: %v", err)
	}
	defer occupied.Close() //nolint:errcheck // 테스트가 끝나면 커널이 거둔다
	addr := occupied.Addr().String()

	root := filepath.Join(t.TempDir(), "artifacts")
	cfgPath := mediatorConfig(t, addr, scratchDB(t), root)

	code, _, stderr := callRun(t, "--config", cfgPath)

	if code != 1 {
		t.Fatalf("exit code contract: a port already in use = %d, want 1; stderr %q", code, stderr)
	}
	for _, want := range []string{"mediator started", "listen=" + addr, "server stopped"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr wording contract: got %q, want %q in it", stderr, want)
		}
	}
	if strings.Contains(stderr, "shutting down") {
		t.Fatalf("a failed bind said %q; that line belongs to the signal path only", stderr)
	}
	// 여기까지 왔다는 것은 스키마와 아티팩트 루트를 지났다는 뜻이다.
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("run() reached the server without creating %s: %v", root, err)
	}
}

func TestRun_StopsAtTheFirstThingItCannotDo(t *testing.T) {
	// 갈래마다 종료 코드는 같은 1 이고 다른 것은 문구뿐이다. 그 문구가 곧
	// 계약이므로 (main.go 의 loadConfig 주석) 갈래별로 무엇을 말하는지를 건다.
	cleanEnv(t)
	noSystemConfig(t)

	t.Run("no config anywhere", func(t *testing.T) {
		cleanEnv(t)
		e := isolate(t)
		t.Setenv("HOME", e.home)
		code, _, stderr := callRun(t)
		if code != 1 {
			t.Fatalf("exit code contract: no config = %d, want 1; stderr %q", code, stderr)
		}
		if !strings.Contains(stderr, "no config file found") {
			t.Fatalf("stderr wording contract: got %q, want %q in it", stderr, "no config file found")
		}
	})

	t.Run("a config with no token and nowhere to put one", func(t *testing.T) {
		// 설정은 찾았는데 토큰이 없고 만들어 넣을 수도 없는 판이다. 읽기만
		// 되는 디렉터리가 그것을 만든다 - config.EnsureToken 은 파일을 읽고
		// 같은 디렉터리에 임시 파일을 만들어 옮기므로, 디렉터리에 쓰기가
		// 없으면 읽기는 되고 쓰기만 막힌다.
		cleanEnv(t)
		if os.Geteuid() == 0 {
			t.Fatalf("this test runs as root, and root writes into a 0500 directory; " +
				"the no-token branch cannot be measured here")
		}
		dir := filepath.Join(t.TempDir(), "readonly")
		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
		// 되돌리는 정리를 먼저 등록한다 - t.Cleanup 은 LIFO 라 TempDir 이
		// 등록한 삭제보다 나중에 등록한 이것이 먼저 돈다.
		t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
		cfgPath := filepath.Join(dir, "config.yaml")
		if err := os.WriteFile(cfgPath, []byte("listen: 127.0.0.1:1\n"), 0o600); err != nil {
			t.Fatalf("write %s: %v", cfgPath, err)
		}
		if err := os.Chmod(dir, 0o500); err != nil {
			t.Fatalf("chmod %s: %v", dir, err)
		}

		code, _, stderr := callRun(t, "--config", cfgPath)
		if code != 1 {
			t.Fatalf("exit code contract: a config with no token = %d, want 1; stderr %q", code, stderr)
		}
		if !strings.Contains(stderr, "no token: set token in the config file or $ENODE_MEDIATOR_TOKEN") {
			t.Fatalf("stderr wording contract: got %q, want the token demand", stderr)
		}
	})

	t.Run("a database that is not there", func(t *testing.T) {
		cleanEnv(t)
		cfgPath := mediatorConfig(t, "127.0.0.1:1",
			"postgres://nobody:nobody@127.0.0.1:1/nope?sslmode=disable", t.TempDir())
		code, _, stderr := callRun(t, "--config", cfgPath)
		if code != 1 {
			t.Fatalf("exit code contract: an unreachable database = %d, want 1; stderr %q", code, stderr)
		}
		if !strings.Contains(stderr, "cannot open database") {
			t.Fatalf("stderr wording contract: got %q, want %q in it", stderr, "cannot open database")
		}
	})

	t.Run("an artifacts root under a file", func(t *testing.T) {
		cleanEnv(t)
		blocker := filepath.Join(t.TempDir(), "blocker")
		if err := os.WriteFile(blocker, []byte("not a directory\n"), 0o600); err != nil {
			t.Fatalf("write %s: %v", blocker, err)
		}
		cfgPath := mediatorConfig(t, "127.0.0.1:"+strconv.Itoa(freePort(t)), scratchDB(t),
			filepath.Join(blocker, "artifacts"))
		code, _, stderr := callRun(t, "--config", cfgPath)
		if code != 1 {
			t.Fatalf("exit code contract: an artifacts root under a file = %d, want 1; stderr %q", code, stderr)
		}
		if !strings.Contains(stderr, "cannot create artifacts directory") {
			t.Fatalf("stderr wording contract: got %q, want %q in it", stderr, "cannot create artifacts directory")
		}
	})
}

// ── 기동 시 큐 훑기 (ADR-064 §6) ─────────────────────────────────────────

// queuedContract 는 요구 하나 · 단계 하나의 최소 계약이다.
func queuedContract() contract.Contract {
	return contract.Contract{
		Requires: []contract.Require{{As: "b", Capability: contract.CapabilityAgentReason,
			Attrs: map[string]string{"harness": "claude"}}},
		Steps: []contract.Step{{ID: "one", Uses: "b", Run: []string{"true"}}},
	}
}

// 죽어 있는 동안 풀린 것을 기동이 줍는다 — 광고 없이 넣어 QUEUED 로 두고,
// 노드를 들인 뒤 기동 훑기를 부르면 RUNNING 이다.
func TestWakeQueuedAtStart_PromotesWhatWasFreedWhileDown(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(ctx, scratchDB(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if promoted, err := st.CreateQueuedRun(ctx, store.Run{RunID: "w", Principal: "p", Contract: queuedContract()}); err != nil || promoted {
		t.Fatalf("queueing without a fleet: promoted=%v err=%v", promoted, err)
	}
	adv := contract.Advert{NodeID: "n1", Label: "n1", Capabilities: []contract.Capability{{
		Capability: contract.CapabilityAgentReason, Attrs: map[string]string{"harness": "claude"}}}}
	if _, err := st.UpsertAdvert(ctx, adv, "p", time.Minute); err != nil {
		t.Fatal(err)
	}

	_, stderr := captureOutput(t, func() { wakeQueuedAtStart(ctx, st, logTo()) })

	if !strings.Contains(stderr, "promoted queued runs at start") {
		t.Fatalf("stderr wording contract: got %q, want the promotion line in it", stderr)
	}
	run, err := st.GetRun(ctx, "w")
	if err != nil || run.State != store.StateRunning {
		t.Fatalf("w after the start wake: %v %v, want RUNNING", run, err)
	}
}

// 훑기가 실패해도 기동은 계속한다 — 오류 한 줄이 남고 돌아온다.
func TestWakeQueuedAtStart_FailureIsLoggedAndBootContinues(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(ctx, scratchDB(t))
	if err != nil {
		t.Fatal(err)
	}
	st.Close() // 닫힌 풀 — 훑기의 Begin 이 실패한다

	_, stderr := captureOutput(t, func() { wakeQueuedAtStart(ctx, st, logTo()) })

	if !strings.Contains(stderr, "cannot wake the queue at start") {
		t.Fatalf("stderr wording contract: got %q, want the failure line in it", stderr)
	}
}
