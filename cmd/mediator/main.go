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

func main() {
	cfgPath := flag.String("config", "", "path to the config file")
	// --version 과 setup 은 플래그 파싱보다 앞이다 (ADR-056).
	// setup 은 설정도 DB 도 없는 상태에서 불리므로 Load 를 지나면 안 된다.
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-version", "version":
			fmt.Println(build.Version("mediator"))
			return
		case "setup":
			os.Exit(runSetup(os.Args[2:]))
		}
	}
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Error("cannot read config", "err", err)
		os.Exit(1)
	}
	// 설정을 못 찾은 것과 토큰만 빠진 것을 가른다.
	//
	// 예전에는 둘이 같은 메시지로 나왔다. 못 찾아도 조용히 기본값으로 갔고
	// 그다음 줄에서 "토큰이 없다" 로 죽으니, 사람이 토큰만 들여다봤다.
	// 진짜 원인은 그 위에 있었다.
	cfgFile, tried, _ := config.Resolve(*cfgPath)
	if cfgFile == "" && os.Getenv("ENODE_MEDIATOR_TOKEN") == "" {
		log.Error("no config file found", "tried", strings.Join(tried, ", "))
		fmt.Fprintf(os.Stderr, "\nRun `mediator setup` to create one.\n")
		os.Exit(1)
	}
	if cfg.Token == "" {
		// 파일은 있는데 값만 비었으면 만들어 넣는다. 조용한 대체가 아니다 —
		// 만들고, 파일에 남기고, 만들었다고 말한다 (ADR-015 §1 은 인증을
		// 건너뛰는 것을 막는 것이지 부트스트랩을 막는 것이 아니다).
		if cfgFile != "" {
			tok, terr := config.EnsureToken(cfgFile)
			if terr == nil && tok != "" {
				cfg.Token = tok
				log.Info("generated a token and wrote it to the config", "config", cfgFile)
				fmt.Fprintf(os.Stderr, "\ntoken  %s\n"+
					"       Every node needs this exact value; a node with a\n"+
					"       different token is rejected with 401.\n\n", tok)
			}
		}
	}
	if cfg.Token == "" {
		// ADR-015 의 원칙 — 조용한 대체를 하지 않는다. 없으면 그 자리에서 죽는다.
		log.Error("no token: set token in the config file or $ENODE_MEDIATOR_TOKEN")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := store.Open(ctx, cfg.Database.URL)
	if err != nil {
		log.Error("cannot open database", "err", err)
		os.Exit(1)
	}
	defer st.Close()
	// Run Record 는 DB 가 아니라 파일시스템에 산다 (ADR-015 §3) —
	// I4(봉인)를 파일시스템은 강제할 수 있고 행은 못 한다.
	if err := os.MkdirAll(cfg.Artifacts.Root, 0o755); err != nil {
		log.Error("cannot create artifacts directory", "root", cfg.Artifacts.Root, "err", err)
		os.Exit(1)
	}
	st.Records = record.New(cfg.Artifacts.Root)
	// 되돌림은 밖에서 안 보이는 판단이다 — 로그가 없으면 왜 같은 단계가
	// 두 번 도는지 운영자가 알 길이 없다.
	st.Log = log
	// 계약이 얼마나 자랄 수 있는가 — 계약 밖에서 정한다 (ADR-031).
	st.MaxContractVersions = cfg.Contract.MaxVersions
	st.MaxLeasesPerRun = cfg.Lease.MaxPerRun
	st.NotifyURL = cfg.Notify.AsksURL
	if err := st.Migrate(ctx); err != nil {
		log.Error("cannot apply database schema", "err", err)
		os.Exit(1)
	}

	// 시간이 감시자다 (ADR-008) — 시작할 때 한 번 먼저 돈다(재시작 스캔).
	// Mediator 가 죽어 있는 동안 갱신이 멈추고, 재시작하면 not_after 가 지나
	// 여기서 회수된다. 이것이 O6 의 구현이다.
	go st.RunReaper(ctx, time.Duration(cfg.Lease.RenewSeconds)*time.Second, log)

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           api.New(st, cfg, log).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		// WriteTimeout 을 걸지 않는다 — claim 이 최대 2시간 매달리는
		// 롱폴이기 때문이다 (ADR-015 §5). 걸면 정상 대기가 끊긴다.
	}
	go func() {
		log.Info("mediator started", "listen", cfg.Listen)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server stopped", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdown)
}
