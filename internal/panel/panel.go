// Package panel 은 노드 소유자가 자기 기계에서 자기 노드 하나를 보고 통제하는
// 제어판 서버다 (enode-features 3.1.2 · unit-of-work §5).
//
// 무상태다 — 값을 셋에서 그때그때 읽어 그린다: 로컬 파일(신원·정책·상태 파일) ·
// 로컬 프로세스(internal/proc) · Mediator 조회(runctl.Client). 하나가 죽어도
// 나머지는 산다.
//
// 제어판은 데몬과 다른 프로세스다 — enodectl serve 가 enode 제어판 하위명령으로
// exec 위임하고, 그 하위명령이 이 패키지를 net/http 로 띄운다. 데몬 실행파일에
// HTTP 를 직접 안 넣는 것은 enodectl.exe 심볼 상한 때문이다 (decisions §2).
package panel

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/taeels/enode/internal/enode"
	"github.com/taeels/enode/internal/runctl"
)

// Config 는 제어판 서버의 입력이다. 겉면은 unit-of-work §5 가 얼렸고,
// ConfigPath 를 더한다 — 정책·상태 파일이 그 옆에 살아 경로 없이는 drain 토글도
// 탐지 능력 읽기도 못 한다.
type Config struct {
	Node         string // 노드 이름 (표시 · serve <name> 가 넘긴다)
	ConfigPath   string // 이 노드의 설정 파일 경로. 정책·상태 파일이 그 옆이다
	Listen       string // 바인딩. 기본 127.0.0.1:8081
	MediatorBase string // runctl.Client.Base
	Token        string // runctl.Client.Token
	PanelToken   string // LAN 노출 시 필수 (정책 파일에서 읽어 넣는다)
}

// Server 는 제어판이다.
type Server struct {
	cfg    Config
	client *runctl.Client
	ident  enode.Identity // 로컬에서 낸 신원. Derive 실패해도 나머지는 그린다

	mu             sync.Mutex
	lastMediatorOK time.Time // Mediator 마지막 성공 응답 시각

	// startBin 은 start 가 띄우는 실행파일이다. 비면 os.Executable() —
	// 제어판 자신이 enode(enode panel)이므로 자기를 --config 로 다시 띄운다.
	// 시험이 무해한 실행파일로 갈아끼우는 이음매다.
	startBin string
}

// New 는 제어판 서버를 만든다.
//
// LAN 노출인데 panel_token 이 비면 거부한다 — 그 기계에 접속한 것이 소유의
// 증거인데(ADR-063 §3) LAN 은 그 전제가 깨지므로 토큰이 필수다. Mediator 토큰을
// 재사용하지 않는다 — 함대 전체의 신뢰 경계라 기계 하나의 제어판에 안 흘린다
// (decisions 「LAN 노출의 토큰」).
func New(cfg Config) (*Server, error) {
	if !isLoopback(cfg.Listen) && cfg.PanelToken == "" {
		return nil, fmt.Errorf("panel is exposed on %s but no panel_token is set in the policy file; refusing to start", cfg.Listen)
	}
	ident, _ := enode.Derive(cfg.ConfigPath) // 실패해도 로컬 화면은 그린다
	c := &runctl.Client{
		Base:      cfg.MediatorBase,
		Token:     cfg.Token,
		Principal: ident.Principal,
		// 제어판은 롱폴을 안 쓴다 — 짧은 타임아웃으로 Mediator 불통이 화면을 멈추지 않게.
		HTTP: &http.Client{Timeout: 3 * time.Second},
	}
	return &Server{cfg: cfg, client: c, ident: ident}, nil
}

// Handler 는 라우트를 낸다.
//
// panel_token 이 있으면(LAN) 모든 요청에 Bearer 토큰을 요구한다. loopback 기본은
// 인증이 없다 — 그 기계에 접속한 것이 소유의 증거다.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("GET /api/state", s.handleState)
	mux.HandleFunc("POST /api/drain", s.handleDrain)
	mux.HandleFunc("POST /api/undrain", s.handleUndrain)
	mux.HandleFunc("POST /api/stop", s.handleStop)
	mux.HandleFunc("POST /api/start", s.handleStart)
	mux.HandleFunc("GET /api/logs", s.handleLogs)
	mux.HandleFunc("GET /api/transcript", s.handleTranscript)
	mux.HandleFunc("GET /api/runs", s.handleRuns)
	mux.HandleFunc("GET /api/record", s.handleRecord)
	if s.cfg.PanelToken != "" {
		return s.requireToken(mux)
	}
	return mux
}

// requireToken 은 Authorization: Bearer <panel_token> 을 요구한다 (LAN 노출 시).
func (s *Server) requireToken(next http.Handler) http.Handler {
	want := "Bearer " + s.cfg.PanelToken
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != want {
			http.Error(w, "panel token required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// isLoopback 은 바인딩 호스트가 이 기계 안쪽인지다.
//
// 안쪽이 아니면 LAN 노출이고 panel_token 이 필수가 된다. 포트가 없거나 호스트가
// 비면(예: ":8081") 모든 인터페이스에 붙는 것이라 loopback 이 아니다.
func isLoopback(listen string) bool {
	host, _, err := net.SplitHostPort(listen)
	if err != nil {
		host = listen // 포트 없는 값이 오면 그대로 호스트로 본다
	}
	switch strings.ToLower(host) {
	case "127.0.0.1", "localhost", "::1":
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

// markMediator 는 Mediator 호출 결과를 기억한다 — 성공하면 시각을 갱신한다.
func (s *Server) markMediator(ok bool) {
	if !ok {
		return
	}
	s.mu.Lock()
	s.lastMediatorOK = time.Now()
	s.mu.Unlock()
}

func (s *Server) mediatorLast() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastMediatorOK
}
