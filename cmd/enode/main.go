// Command enode 는 실행 노드다.
//
//	enode --config /etc/enode/ws-a.yaml
//
// ★ 설정 파일이 곧 신원이다 ★ (ADR-015 §2) — 한 기계에서 둘을 띄우려면
// 설정이 이미 둘이어야 한다. 같은 설정으로 두 번 띄우면 그 자리에서 죽는다.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/taeels/enode/internal/enode"
)

func main() {
	// ★ 하위 명령이 하나 있다 — `enode hook stop` ★ (R5③ · R6).
	// 훅을 별도 스크립트로 두지 않고 enode 자신이 되는 이유는 hook.go 에 적었다.
	if len(os.Args) > 1 && os.Args[1] == "hook" {
		os.Exit(runHookCmd(os.Args[2:]))
	}

	cfgPath := flag.String("config", "/etc/enode/local.yaml", "path to the config file (also determines node identity)")
	mediator := flag.String("mediator", "", "mediator address (overrides config)")
	token := flag.String("token", "", "auth token (overrides config)")
	every := flag.Duration("every", 60*time.Second, "interval for advertise, heartbeat and lease renewal")
	debug := flag.Bool("debug", false, "")
	flag.Parse()

	lvl := slog.LevelInfo
	if *debug {
		lvl = slog.LevelDebug
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl}))

	local, err := enode.LoadLocal(*cfgPath)
	if err != nil {
		log.Error("cannot read config", "err", err)
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
		log.Error("mediator and token are required")
		os.Exit(1)
	}

	// 신원. 이메일이 없으면 ★ 그 자리에서 죽는다 ★ — 조용한 대체를 안 한다.
	ident, err := enode.Derive(*cfgPath)
	if err != nil {
		log.Error("cannot derive node identity", "err", err)
		os.Exit(1)
	}

	// ★ 중복 실행 방지는 로컬에서 ★ — Mediator 는 재시작과 중복을 구분할 수 없다.
	lock, err := enode.Acquire(ident.Config)
	if err != nil {
		log.Error("cannot acquire lock", "err", err)
		os.Exit(1)
	}
	defer lock.Release()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ★ 이번 생의 표식 ★ (ADR-030) — 기동마다 새로 뽑는다. NodeID 는 신원이라
	// 재시작해도 같지만, 이것은 ★ 재시작하면 달라지는 것 ★ 이 존재 이유다.
	inst := make([]byte, 8)
	if _, err := rand.Read(inst); err != nil {
		log.Error("cannot generate instance id", "err", err)
		os.Exit(1)
	}

	client := &enode.Client{
		Base: local.Mediator, Token: local.Token, Principal: ident.Principal,
		Instance: hex.EncodeToString(inst),
		HTTP:     &http.Client{Timeout: 30 * time.Second},
		// ★ 롱폴은 타임아웃을 두지 않는다 ★ (ADR-029) — 수명은 ctx 가 관리한다.
		// 짧은 타임아웃으로 걸면 그 시간에 끊고, ★ 끊는 순간 서버가 집으면
		// 응답이 유실되어 그 단계를 아무도 안 돌린다 ★ (claim 은 비멱등이다).
		Poll: &http.Client{},
	}
	caps := enode.Detect(local, log)
	log.Info("enode started",
		"node", ident.NodeID, "label", ident.Label,
		"config", ident.Config, "caps", caps)

	// ★ 두 연결을 동시에 든다 ★ (ADR-016)
	//   claim   롱폴 — 일을 기다린다. ★ 서버가 대기 시간을 정한다 ★
	//   nodes   짧은 주기 — 살아 있다고 말하고 권한을 받는다
	// ★ 클라이언트도 둘이다 ★ (ADR-029) — 롱폴에 짧은 타임아웃을 걸면
	// 그 시간에 끊고 응답이 유실된다.
	held := enode.NewHeld()

	adv := &enode.Advertiser{
		Client: client, Ident: ident, Local: local, Every: *every, Log: log,
		OnLeases: held.Set, // ★ 응답이 임대의 갱신이자 취소 통보다 ★ 통째로 교체한다
	}
	worker := &enode.Worker{Client: client, Ident: ident, Local: local, Held: held, Log: log}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); adv.Run(ctx) }()
	go func() { defer wg.Done(); worker.Run(ctx) }()
	wg.Wait()
	log.Info("stopped")
}
