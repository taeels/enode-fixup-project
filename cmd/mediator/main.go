// Command mediator 는 매칭 · 권한 · 바인딩 · Run Record 를 맡는다 (ADR-002).
//
// ★ 실행은 하지 않는다 ★ — 실행은 enode 의 몫이다.
package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/taeels/enode/internal/api"
	"github.com/taeels/enode/internal/config"
	"github.com/taeels/enode/internal/record"
	"github.com/taeels/enode/internal/store"
)

func main() {
	cfgPath := flag.String("config", "", "설정 파일 (기본: ADR-015 §4 의 우선순위)")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Error("설정을 읽을 수 없다", "err", err)
		os.Exit(1)
	}
	if cfg.Token == "" {
		// ADR-015 의 원칙 — 조용한 대체를 하지 않는다. 없으면 그 자리에서 죽는다.
		log.Error("토큰이 없다. config 의 token 또는 $ENODE_MEDIATOR_TOKEN 을 설정하라")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := store.Open(ctx, cfg.Database.URL)
	if err != nil {
		log.Error("DB 를 열 수 없다", "err", err)
		os.Exit(1)
	}
	defer st.Close()
	// Run Record 는 ★ DB 가 아니라 파일시스템 ★ 에 산다 (ADR-015 §3) —
	// I4(봉인)를 파일시스템은 강제할 수 있고 행은 못 한다.
	if err := os.MkdirAll(cfg.Artifacts.Root, 0o755); err != nil {
		log.Error("아티팩트 디렉터리를 만들 수 없다", "root", cfg.Artifacts.Root, "err", err)
		os.Exit(1)
	}
	st.Records = record.New(cfg.Artifacts.Root)
	if err := st.Migrate(ctx); err != nil {
		log.Error("스키마 적용 실패", "err", err)
		os.Exit(1)
	}

	// ★ 시간이 감시자다 ★ (ADR-008) — 시작할 때 한 번 먼저 돈다(재시작 스캔).
	// Mediator 가 죽어 있는 동안 갱신이 멈추고, 재시작하면 not_after 가 지나
	// 여기서 회수된다. 이것이 O6 의 구현이다.
	go st.RunReaper(ctx, time.Duration(cfg.Lease.RenewSeconds)*time.Second, log)

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           api.New(st, cfg, log).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		// ★ WriteTimeout 을 걸지 않는다 ★ — claim 이 최대 2시간 매달리는
		// 롱폴이기 때문이다 (ADR-015 §5). 걸면 정상 대기가 끊긴다.
	}
	go func() {
		log.Info("Mediator 시작", "listen", cfg.Listen)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("서버가 죽었다", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("종료 중")
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdown)
}
