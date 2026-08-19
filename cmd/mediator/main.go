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
	if err := st.Migrate(ctx); err != nil {
		log.Error("스키마 적용 실패", "err", err)
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           api.New(st, cfg, log).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
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
