// Command enode 는 실행 노드다.
//
//	enode --config /etc/enode/ws-a.yaml
//
// ★ 설정 파일이 곧 신원이다 ★ (ADR-015 §2) — 한 기계에서 둘을 띄우려면
// 설정이 이미 둘이어야 한다. 같은 설정으로 두 번 띄우면 그 자리에서 죽는다.
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

	"github.com/taeels/enode/internal/enode"
)

func main() {
	cfgPath := flag.String("config", "/etc/enode/local.yaml", "설정 파일. ★ 이 경로가 신원의 일부다 ★")
	mediator := flag.String("mediator", "", "Mediator 주소 (설정을 덮어쓴다)")
	token := flag.String("token", "", "토큰 (설정을 덮어쓴다)")
	every := flag.Duration("every", 60*time.Second, "광고 · 하트비트 · 임대 갱신 주기 (한 값이다)")
	debug := flag.Bool("debug", false, "")
	flag.Parse()

	lvl := slog.LevelInfo
	if *debug {
		lvl = slog.LevelDebug
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl}))

	local, err := enode.LoadLocal(*cfgPath)
	if err != nil {
		log.Error("설정을 읽을 수 없다", "err", err)
		os.Exit(1)
	}
	if *mediator != "" {
		local.Mediator = *mediator
	}
	if *token != "" {
		local.Token = *token
	}
	if v := os.Getenv("ENODE_TOKEN"); v != "" && *token == "" {
		local.Token = v
	}
	if local.Mediator == "" || local.Token == "" {
		log.Error("mediator 와 token 이 필요하다")
		os.Exit(1)
	}

	// 신원. 이메일이 없으면 ★ 그 자리에서 죽는다 ★ — 조용한 대체를 안 한다.
	ident, err := enode.Derive(*cfgPath)
	if err != nil {
		log.Error("신원을 계산할 수 없다", "err", err)
		os.Exit(1)
	}

	// ★ 중복 실행 방지는 로컬에서 ★ — Mediator 는 재시작과 중복을 구분할 수 없다.
	lock, err := enode.Acquire(ident.Config)
	if err != nil {
		log.Error("잠금 실패", "err", err)
		os.Exit(1)
	}
	defer lock.Release()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client := &enode.Client{
		Base: local.Mediator, Token: local.Token, Principal: ident.Principal,
		HTTP: &http.Client{Timeout: 30 * time.Second},
	}
	caps := enode.Detect(local, log)
	log.Info("enode 시작",
		"node", ident.NodeID, "label", ident.Label,
		"config", ident.Config, "caps", caps)

	adv := &enode.Advertiser{Client: client, Ident: ident, Local: local, Every: *every, Log: log}
	adv.Run(ctx)
	log.Info("종료")
}
