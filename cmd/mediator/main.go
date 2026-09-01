// Command mediator 는 매칭 · 권한 · 바인딩 · Run Record 를 맡는다 (ADR-002).
//
// 실행은 하지 않는다 — 실행은 enode 의 몫이다.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/taeels/enode/internal/api"
	"github.com/taeels/enode/internal/build"
	"github.com/taeels/enode/internal/config"
	"github.com/taeels/enode/internal/record"
	"github.com/taeels/enode/internal/store"
)

func main() { os.Exit(run()) }

// run 은 main 의 몸통이다. 종료 코드를 돌려주고 os.Exit 을 직접 부르지 않는다.
//
// 왜 가르는가 — 이 모양이 아니면 기동 경로에 테스트 가능한 이음매가 없다.
// 이 패키지의 211 문장 중 57 이 main() 안에 있었고, setup.go 를 전부 덮어도
// 154 라 패키지 하한 80%(169 문장)에 15 문장이 모자란다. 즉 이 추출은 취향이
// 아니라 산술이다.
//
// 무엇이 달라지는가 — os.Exit 은 defer 를 안 돌리고 return 은 돌린다.
// 아래 defer stop() 과 defer st.Close() 가 이제 실패 경로에서도 돈다
// (예전에는 아티팩트 디렉터리 실패 · 마이그레이션 실패 · 바인드 실패에서
// 프로세스가 그대로 사라졌다). 밖에서 보이는 것은 같다 — 종료 코드도
// stderr 문구도 그대로이고, 달라지는 것은 커널이 거두던 것을 코드가 먼저
// 놓는다는 것뿐이다. entrypoint_unix_test.go 가 그 계약을 프로세스 경계에서
// 잡고 있다.
func run() int {
	cfgPath := flag.String("config", "", "path to the config file")
	// --version 과 setup 은 플래그 파싱보다 앞이다 (ADR-056).
	// setup 은 설정도 DB 도 없는 상태에서 불리므로 Load 를 지나면 안 된다.
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-version", "version":
			fmt.Println(build.Version("mediator"))
			return 0
		case "setup":
			return runSetup(os.Args[2:])
		}
	}
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, cfgFile, ok := loadConfig(*cfgPath, log)
	if !ok {
		return 1
	}
	if !bootstrapToken(&cfg, cfgFile, log) {
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, ok := openStore(ctx, cfg, log)
	if !ok {
		return 1
	}
	defer st.Close()
	if !openRecords(st, cfg, log) {
		return 1
	}
	if !migrate(ctx, st, log) {
		return 1
	}

	// 시간이 감시자다 (ADR-008) — 시작할 때 한 번 먼저 돈다(재시작 스캔).
	// Mediator 가 죽어 있는 동안 갱신이 멈추고, 재시작하면 not_after 가 지나
	// 여기서 회수된다. 이것이 O6 의 구현이다.
	reaper := make(chan struct{})
	go func() {
		defer close(reaper)
		st.RunReaper(ctx, time.Duration(cfg.Lease.RenewSeconds)*time.Second, log)
	}()
	// 감시자를 세우고 나서 풀을 닫는다. defer 는 LIFO 이므로 이것이 위의
	// st.Close() 보다 먼저 돈다.
	//
	// 왜 필요해졌나 — 실측이다. os.Exit 을 쓰던 동안에는 프로세스가 그
	// 자리에서 사라져 감시자가 무엇을 하든 보이지 않았다. return 으로 바꾸니
	// 바인드 실패 경로에서 st.Close() 가 먼저 돌고, 재시작 스캔 중이던
	// 감시자가 `reaper scan failed err="closed pool"` 을 ERROR 로 찍었다.
	// 종료 코드도 문구도 그대로였지만 stderr 에 없던 줄이 하나 생겼고, 그것도
	// 경합에 따라 났다 안 났다 했다. 쥐고 있는 쪽을 먼저 세우는 것이 순서다.
	defer func() {
		stop()
		<-reaper
	}()

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           api.New(st, cfg, log).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		// WriteTimeout 을 걸지 않는다 — claim 이 최대 2시간 매달리는
		// 롱폴이기 때문이다 (ADR-015 §5). 걸면 정상 대기가 끊긴다.
	}
	return serve(ctx, srv, cfg.Listen, log)
}

// loadConfig 는 설정을 읽고, 못 찾은 것과 못 읽는 것을 가른다.
//
// 예전에는 둘이 같은 메시지로 나왔다. 못 찾아도 조용히 기본값으로 갔고
// 그다음 줄에서 "토큰이 없다" 로 죽으니, 사람이 토큰만 들여다봤다.
// 진짜 원인은 그 위에 있었다.
//
// ok 가 false 면 이미 이유를 로그와 stderr 에 냈다 — 부르는 쪽은 종료 코드만
// 정한다. 오류를 돌려주고 부르는 쪽이 찍게 하는 안을 기각했다: 문구가 갈래마다
// 다르고(하나는 찾아본 자리를 열거하고 하나는 setup 을 가리킨다) 그 문구가
// 곧 계약이므로, 갈래와 문구를 떼면 한쪽만 낡는다.
func loadConfig(cfgPath string, log *slog.Logger) (config.Config, string, bool) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Error("cannot read config", "err", err)
		return cfg, "", false
	}
	cfgFile, tried, _ := config.Resolve(cfgPath)
	if cfgFile == "" && os.Getenv("ENODE_MEDIATOR_TOKEN") == "" {
		log.Error("no config file found", "tried", strings.Join(tried, ", "))
		fmt.Fprintf(os.Stderr, "\nRun `mediator setup` to create one.\n")
		return cfg, "", false
	}
	return cfg, cfgFile, true
}

// bootstrapToken 은 토큰이 비었으면 만들어 넣는다.
//
// 조용한 대체가 아니다 — 만들고, 파일에 남기고, 만들었다고 말한다 (ADR-015 §1
// 은 인증을 건너뛰는 것을 막는 것이지 부트스트랩을 막는 것이 아니다).
// 만들 자리조차 없으면 그 자리에서 죽는다.
func bootstrapToken(cfg *config.Config, cfgFile string, log *slog.Logger) bool {
	if cfg.Token == "" && cfgFile != "" {
		tok, terr := config.EnsureToken(cfgFile)
		if terr == nil && tok != "" {
			cfg.Token = tok
			log.Info("generated a token and wrote it to the config", "config", cfgFile)
			fmt.Fprintf(os.Stderr, "\ntoken  %s\n"+
				"       Every node needs this exact value; a node with a\n"+
				"       different token is rejected with 401.\n\n", tok)
		}
	}
	if cfg.Token == "" {
		// ADR-015 의 원칙 — 조용한 대체를 하지 않는다. 없으면 그 자리에서 죽는다.
		log.Error("no token: set token in the config file or $ENODE_MEDIATOR_TOKEN")
		return false
	}
	return true
}

// openStore 는 DB 를 연다. 여는 데 실패하면 닫을 것도 없다.
func openStore(ctx context.Context, cfg config.Config, log *slog.Logger) (*store.Store, bool) {
	st, err := store.Open(ctx, cfg.Database.URL)
	if err != nil {
		log.Error("cannot open database", "err", err)
		return nil, false
	}
	return st, true
}

// openRecords 는 저장소의 파일시스템 쪽을 열고 계약 밖의 값들을 배선한다.
//
// Run Record 는 DB 가 아니라 파일시스템에 산다 (ADR-015 §3) — I4(봉인)를
// 파일시스템은 강제할 수 있고 행은 못 한다. 그래서 이 자리가 DB 를 여는 것과
// 짝이지 곁다리가 아니다.
func openRecords(st *store.Store, cfg config.Config, log *slog.Logger) bool {
	if err := os.MkdirAll(cfg.Artifacts.Root, 0o755); err != nil {
		log.Error("cannot create artifacts directory", "root", cfg.Artifacts.Root, "err", err)
		return false
	}
	st.Records = record.New(cfg.Artifacts.Root)
	// 되돌림은 밖에서 안 보이는 판단이다 — 로그가 없으면 왜 같은 단계가
	// 두 번 도는지 운영자가 알 길이 없다.
	st.Log = log
	// 계약이 얼마나 자랄 수 있는가 — 계약 밖에서 정한다 (ADR-031).
	st.MaxContractVersions = cfg.Contract.MaxVersions
	st.MaxLeasesPerRun = cfg.Lease.MaxPerRun
	st.NotifyURL = cfg.Notify.AsksURL
	return true
}

// migrate 는 스키마를 올린다. 멱등이라 매 기동마다 돈다.
func migrate(ctx context.Context, st *store.Store, log *slog.Logger) bool {
	if err := st.Migrate(ctx); err != nil {
		log.Error("cannot apply database schema", "err", err)
		return false
	}
	return true
}

// serve 는 서버를 띄우고, 시그널과 바인드 실패 중 먼저 오는 것으로 끝난다.
//
// 예전에는 ListenAndServe 를 도는 고루틴이 실패하면 그 안에서 os.Exit(1) 을
// 불렀다. 그 자리를 오류 채널로 옮긴 이유는 os.Exit 이 고루틴에서 불리면
// main 의 defer 가 하나도 안 돌기 때문이다 — 이 패키지의 이음매를 뽑고도
// 그 한 줄이 남아 있으면 「종료 코드는 run() 이 정한다」가 거짓이 된다.
//
// 바꾸지 않은 것이 관측이다. 바인드 실패는 여전히 "mediator started" 다음에
// "server stopped" 를 찍고 1 로 끝나며, 이 경로에는 "shutting down" 도
// Shutdown 도 없다. 그것이 정상 종료 경로의 표식이기 때문이다.
//
// 버퍼를 1 로 두는 이유 — 시그널로 끝나는 경로에서는 아무도 이 채널을 안
// 읽는다. 버퍼가 없으면 고루틴이 보내기에서 영영 멈춰 샌다.
func serve(ctx context.Context, srv *http.Server, listen string, log *slog.Logger) int {
	failed := make(chan error, 1)
	go func() {
		log.Info("mediator started", "listen", listen)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			failed <- err
		}
	}()

	select {
	case err := <-failed:
		log.Error("server stopped", "err", err)
		return 1
	case <-ctx.Done():
		log.Info("shutting down")
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
		return 0
	}
}
