// Command enode 는 실행 노드다.
//
//	enode --config /etc/enode/ws-a.yaml
//
// 설정 파일이 곧 신원이다 (ADR-015 §2) — 한 기계에서 둘을 띄우려면
// 설정이 이미 둘이어야 한다. 같은 설정으로 두 번 띄우면 그 자리에서 죽는다.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/taeels/enode/internal/build"
	"github.com/taeels/enode/internal/enode"
)

func main() {
	// 하위 명령이 하나 있다 — `enode hook stop` (R5③ · R6).
	// 훅을 별도 스크립트로 두지 않고 enode 자신이 되는 이유는 hook.go 에 적었다.
	if len(os.Args) > 1 && os.Args[1] == "hook" {
		os.Exit(runHookCmd(os.Args[2:]))
	}
	// --version 은 플래그 파싱보다 앞이다 (ADR-056) — 설정 파일이 없어도
	// 답해야 한다. 자기 갱신이 받아온 것이 무엇인지를 이것으로 판정한다.
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-version") {
		fmt.Println(build.Version("enode"))
		return
	}

	// 기본값을 비운다 — 자리는 enode.ConfigPaths() 가 안다. 유닉스만 아는
	// 경로를 여기 박아 두면 윈도우에서 "/etc/enode/local.yaml 이 없다" 로
	// 죽는다. rc8·rc9 가 Mediator 에서 밟은 것과 같은 자리다.
	cfgPath := flag.String("config", "", "path to the config file (also determines node identity)")
	mediator := flag.String("mediator", "", "mediator address (overrides config)")
	token := flag.String("token", "", "auth token (overrides config)")
	every := flag.Duration("every", 60*time.Second, "interval for advertise, heartbeat and lease renewal")
	debug := flag.Bool("debug", false, "")
	// 탄력 노드를 위한 둘 (docs/elastic-nodes.md)
	//
	// 지금까지 enode 는 상주를 전제했다 — 사람이 띄우고 계속 돈다.
	// 이슈마다 · 요청마다 노드가 서는 형태에서는 뜨는 것과 끝나는 것을
	// 띄운 쪽이 알아야 한다.
	once := flag.Bool("once", false,
		"exit after the run this node worked on reaches a terminal state")
	readyFile := flag.String("ready-file", "",
		"write this file once the first advertisement succeeds")
	flag.Parse()

	lvl := slog.LevelInfo
	if *debug {
		lvl = slog.LevelDebug
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl}))

	// 설정을 못 찾은 것과 못 읽는 것을 가른다. 못 찾았으면 어디를 봤는지
	// 말한다 — 안 말하면 사람이 엉뚱한 경로를 들여다본다.
	confPath, tried := enode.ResolveConfig(*cfgPath)
	if confPath == "" {
		log.Error("no config file found", "tried", strings.Join(tried, ", "))
		fmt.Fprintf(os.Stderr, "\nCreate one and point --config at it. The minimum is:\n\n%s\n"+
			"The token must match the mediator's; a node with a different token is\nrejected with 401. "+
			"The absolute path of this file is part of the node id,\nso moving it makes a different node.\n",
			enode.SampleLocal())
		os.Exit(1)
	}
	local, err := enode.LoadLocal(confPath)
	if err != nil {
		log.Error("cannot read config", "path", confPath, "err", err)
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

	// 신원. 이메일이 없으면 그 자리에서 죽는다 — 조용한 대체를 안 한다.
	ident, err := enode.Derive(confPath)
	if err != nil {
		log.Error("cannot derive node identity", "err", err)
		os.Exit(1)
	}

	// 중복 실행 방지는 로컬에서 — Mediator 는 재시작과 중복을 구분할 수 없다.
	lock, err := enode.Acquire(ident.Config)
	if err != nil {
		log.Error("cannot acquire lock", "err", err)
		os.Exit(1)
	}
	defer lock.Release()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 이번 생의 표식 (ADR-030) — 기동마다 새로 뽑는다. NodeID 는 신원이라
	// 재시작해도 같지만, 이것은 재시작하면 달라지는 것이 존재 이유다.
	inst := make([]byte, 8)
	if _, err := rand.Read(inst); err != nil {
		log.Error("cannot generate instance id", "err", err)
		os.Exit(1)
	}

	client := &enode.Client{
		Base: local.Mediator, Token: local.Token, Principal: ident.Principal,
		Instance: hex.EncodeToString(inst),
		HTTP:     &http.Client{Timeout: 30 * time.Second},
		// 롱폴은 타임아웃을 두지 않는다 (ADR-029) — 수명은 ctx 가 관리한다.
		// 짧은 타임아웃으로 걸면 그 시간에 끊고, 끊는 순간 서버가 집으면
		// 응답이 유실되어 그 단계를 아무도 안 돌린다 (claim 은 비멱등이다).
		Poll: &http.Client{},
	}
	caps := enode.Detect(local, log)
	log.Info("enode started",
		"node", ident.NodeID, "label", ident.Label,
		"config", ident.Config, "caps", caps)

	// 두 연결을 동시에 든다 (ADR-016)
	//   claim   롱폴 — 일을 기다린다. 서버가 대기 시간을 정한다
	//   nodes   짧은 주기 — 살아 있다고 말하고 권한을 받는다
	// 클라이언트도 둘이다 (ADR-029) — 롱폴에 짧은 타임아웃을 걸면
	// 그 시간에 끊고 응답이 유실된다.
	held := enode.NewHeld()

	adv := &enode.Advertiser{
		Client: client, Ident: ident, Local: local, Every: *every, Log: log,
		OnLeases: held.Set, // 응답이 임대의 갱신이자 취소 통보다 통째로 교체한다
	}

	// 떴다는 신호 — 첫 광고가 성공한 뒤 한 번 (docs/elastic-nodes.md §4.4).
	//
	// 프로세스가 뜬 것과 함대에 등록된 것은 다르다. 그 사이에 Run 을 내면
	// 매칭이 422 로 거절한다. 띄운 쪽이 이 파일을 기다리면 폴링도 ·
	// 「몇 초」라는 짐작도 필요 없다. k8s 면 readiness probe 가 이것을 본다.
	if *readyFile != "" {
		adv.OnReady = func() {
			if err := os.WriteFile(*readyFile, []byte(ident.NodeID+"\n"), 0o644); err != nil {
				// 막지 않는다 — 노드는 이미 함대에 있다. 못 알린 것뿐이다.
				log.Warn("cannot write ready file", "path", *readyFile, "err", err)
				return
			}
			log.Info("ready", "file", *readyFile, "node", ident.NodeID)
		}
	}

	// 일이 끝나면 종료한다 (--once)
	//
	// 경계를 어디에 두는가 — 워커는 단계를 보고, Run 의 종료는
	// Mediator 가 안다. 그런데 I2 가 그 다리다:
	// "종료 상태(SUCCEEDED/FAILED)에서는 Mediator 의 점유 장부가 비어 있다".
	// 그리고 RenewLeases 는 끝난 Run 의 임대를 안 돌려준다.
	//
	//	한 번이라도 임대를 받았고 · 지금 하나도 없다        ⇒ 그 Run 이 끝났다
	//
	// 시작 직후와 구별해야 한다 — 그때도 임대가 0 이다. 그래서 sawLease 가 있다.
	// 단계 사이에는 안 빈다 — 임대는 Run 이 도는 내내 유지된다.
	if *once {
		adv.OnLeases = func(ls []enode.Lease) {
			held.Set(ls)
			// 일을 집은 적이 있고 지금 임대가 하나도 없다 ⇒ 그 Run 이 끝났다.
			//
			// 「집은 적」은 claim 에서 센다 — 광고 주기(기본 60초)보다 짧은
			// Run 은 광고가 임대를 한 번도 못 본다. 실측에서 밟았다.
			if len(ls) == 0 && held.EverHeld() {
				log.Info("the run finished and no lease remains; exiting (--once)")
				stop()
			}
		}
	}
	worker := &enode.Worker{Client: client, Ident: ident, Local: local, Held: held, Log: log}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); adv.Run(ctx) }()
	go func() { defer wg.Done(); worker.Run(ctx) }()
	wg.Wait()
	log.Info("stopped")
}
