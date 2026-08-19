// Package api 는 Mediator 의 HTTP 표면이다.
//
// 정본은 enode-design 저장소의 protocol/mediator-api.md 다.
// 라우팅은 표준 라이브러리만 쓴다 — Go 1.22+ 의 ServeMux 가 메서드와 경로
// 패턴을 받으므로 라우터 의존성이 필요 없다.
package api

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/taeels/enode/internal/config"
	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/match"
	"github.com/taeels/enode/internal/store"
)

type Server struct {
	st  *store.Store
	cfg config.Config
	log *slog.Logger
}

func New(st *store.Store, cfg config.Config, log *slog.Logger) *Server {
	return &Server{st: st, cfg: cfg, log: log}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/nodes", s.auth(s.postNodes))
	mux.HandleFunc("POST /v1/runs", s.auth(s.postRuns))
	mux.HandleFunc("POST /v1/runs/dry-run", s.auth(s.postDryRun))
	mux.HandleFunc("GET /v1/runs/{id}", s.auth(s.getRun))
	return mux
}

// ── 인증과 식별 (ADR-015 §1) ──────────────────────────────────────────────
//
//	Authorization: Bearer <token>   ★ 인증 ★  붙어도 되는가
//	X-Enode-Principal: <email>      ★ 식별 ★  누구의 것인가. ★ 검증하지 않는다 ★
//
// 이메일에 권한을 걸지 않는다 — ~/.gitconfig 는 사용자가 쓰는 파일이라 자기 신고다.

type ctxKey int

const principalKey ctxKey = 1

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const prefix = "Bearer "
		got := r.Header.Get("Authorization")
		if s.cfg.Token == "" || len(got) <= len(prefix) ||
			subtle.ConstantTimeCompare([]byte(got[len(prefix):]), []byte(s.cfg.Token)) != 1 {
			fail(w, 401, "토큰이 없거나 틀렸다")
			return
		}
		ctx := context.WithValue(r.Context(), principalKey, r.Header.Get("X-Enode-Principal"))
		next(w, r.WithContext(ctx))
	}
}

func principal(r *http.Request) string {
	v, _ := r.Context().Value(principalKey).(string)
	return v
}

// ── 응답 ─────────────────────────────────────────────────────────────────

type errBody struct {
	Error struct {
		Code   int    `json:"code"`
		Reason string `json:"reason"`
	} `json:"error"`
}

func fail(w http.ResponseWriter, code int, reason string) {
	var b errBody
	b.Error.Code, b.Error.Reason = code, reason
	write(w, code, b)
}

func write(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

type runView struct {
	RunID    string           `json:"run_id"`
	State    string           `json:"state"`
	Assigned []store.Assigned `json:"assigned,omitempty"`
	Reject   *match.Reject    `json:"reject,omitempty"`
}

func view(r *store.Run) runView {
	return runView{RunID: r.RunID, State: r.State, Assigned: r.Assigned, Reject: r.Reject}
}

// ── POST /v1/nodes — 광고 + 하트비트 ──────────────────────────────────────
//
// S2 는 쓰기 경로만 만든다. 응답의 leases(임대 갱신·취소 통보, ADR-016)는
// ★ 자리를 지금 만들되 채우는 것은 S4 ★ 다 — 그래야 enode 쪽 계약이 안 바뀐다.
type advertResponse struct {
	Leases []any `json:"leases"`
}

func (s *Server) postNodes(w http.ResponseWriter, r *http.Request) {
	var a contract.Advert
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		fail(w, 400, "광고를 읽을 수 없다: "+err.Error())
		return
	}
	if a.NodeID == "" {
		fail(w, 400, "node_id 가 없다")
		return
	}
	// 광고는 만료된다 (ADR-012). 만료 = 하트비트 주기와 같은 값이다 (ADR-016).
	ttl := time.Duration(s.cfg.Lease.RenewSeconds*s.cfg.Lease.NotAfterFactor) * time.Second
	if err := s.st.UpsertAdvert(r.Context(), a, principal(r), ttl); err != nil {
		s.log.Error("광고 저장 실패", "node", a.NodeID, "err", err)
		fail(w, 503, "저장 실패")
		return
	}
	write(w, 200, advertResponse{Leases: []any{}})
}

// ── POST /v1/runs ────────────────────────────────────────────────────────

func (s *Server) postRuns(w http.ResponseWriter, r *http.Request) { s.submit(w, r, false) }

// dry-run 은 ★ 본문도 매처도 같고 점유만 안 한다 ★ (ADR-014 결정 3).
// 그래서 409 가 나오지 않는다 — 점유를 보지 않기 때문이다.
// 존재는 답하고 여유는 답하지 않는다.
func (s *Server) postDryRun(w http.ResponseWriter, r *http.Request) { s.submit(w, r, true) }

func (s *Server) submit(w http.ResponseWriter, r *http.Request, dry bool) {
	var c contract.Contract
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		fail(w, 400, "계약을 읽을 수 없다: "+err.Error())
		return
	}
	if err := c.Validate(); err != nil {
		fail(w, 400, err.Error())
		return
	}
	ctx := r.Context()

	// ★ 같은 run_id 재제출은 200 + 기존 Run ★ (INVARIANTS §2 첫 행).
	// run_id 가 (change-id, patchset) 에서 결정적으로 유도되므로 폴링 커서가 필요 없다.
	if !dry {
		if existing, err := s.st.GetRun(ctx, c.RunID); err == nil {
			write(w, 200, view(existing))
			return
		} else if !errors.Is(err, store.ErrNotFound) {
			s.log.Error("Run 조회 실패", "run", c.RunID, "err", err)
			fail(w, 503, "조회 실패")
			return
		}
	}

	adverts, err := s.st.LiveAdverts(ctx)
	if err != nil {
		s.log.Error("광고 조회 실패", "err", err)
		fail(w, 503, "조회 실패")
		return
	}

	busy := map[string]bool{}
	if !dry {
		if busy, err = s.st.BusyNodes(ctx); err != nil {
			s.log.Error("점유 장부 조회 실패", "err", err)
			fail(w, 503, "조회 실패")
			return
		}
	}

	assign, rej := match.Match(c.Requires, adverts, busy)
	if rej != nil {
		if !dry {
			// 거절도 기록한다 — 왜 안 돌았는지가 없으면 껍데기가 재시도를 못 정한다.
			run := store.Run{RunID: c.RunID, Principal: principal(r), Contract: c, Reject: rej}
			if err := s.st.CreateRejectedRun(ctx, run); err != nil {
				s.log.Error("거절 기록 실패", "run", c.RunID, "err", err)
			}
		}
		fail(w, rej.Code, rej.Reason)
		return
	}
	if dry {
		write(w, 200, runView{RunID: c.RunID, State: "DRY_RUN", Assigned: label(assign, adverts)})
		return
	}

	// ③ 바인딩 — 임대를 발급한다 (ADR-010 허가 아티팩트).
	notAfter := time.Now().Add(time.Duration(s.cfg.Lease.TTLSeconds) * time.Second)
	var grants []store.LeaseGrant
	seen := map[string]bool{}
	for _, a := range assign {
		for _, n := range a.Nodes {
			if seen[n] { // 한 노드가 여러 역할을 맡으면 임대는 하나다 (ADR-019 결정 2)
				continue
			}
			seen[n] = true
			grants = append(grants, store.LeaseGrant{NodeID: n, NotAfter: notAfter, Nonce: nonce()})
		}
	}

	run := store.Run{
		RunID: c.RunID, State: store.StateRunning, Principal: principal(r),
		Contract: c, Assigned: label(assign, adverts),
	}
	err = s.st.CreateRun(ctx, run, grants, c.Steps)
	switch {
	case errors.Is(err, store.ErrNodeTaken):
		// ★ I5 ★ 그 사이 다른 Run 이 가져갔다. 트랜잭션이 전부 롤백했으므로
		// 손으로 해제할 것이 없다. 일시적 실패이므로 409 다.
		fail(w, match.CodeAllBusy, "배정 중 다른 Run 이 노드를 가져갔다")
		return
	case err != nil:
		// 같은 run_id 로 동시에 들어온 경우도 여기로 온다 — 다시 읽어 200 으로 답한다.
		if existing, gerr := s.st.GetRun(ctx, c.RunID); gerr == nil {
			write(w, 200, view(existing))
			return
		}
		s.log.Error("Run 생성 실패", "run", c.RunID, "err", err)
		fail(w, 503, "생성 실패")
		return
	}
	write(w, 201, view(&run))
}

// ── GET /v1/runs/{id} ────────────────────────────────────────────────────

func (s *Server) getRun(w http.ResponseWriter, r *http.Request) {
	run, err := s.st.GetRun(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		fail(w, 404, "그런 Run 이 없다")
		return
	}
	if err != nil {
		s.log.Error("Run 조회 실패", "err", err)
		fail(w, 503, "조회 실패")
		return
	}
	write(w, 200, view(run))
}

// label 은 node_id 에 사람이 읽는 이름을 붙인다.
// Record 의 노드 귀속이 해시로만 남으면 Case D 의 "서로 다른 기계였다" 를
// 사람이 못 읽는다 (ADR-015 §2).
func label(as []match.Assignment, adverts []contract.Advert) []store.Assigned {
	byID := map[string]string{}
	for _, a := range adverts {
		byID[a.NodeID] = a.Label
	}
	out := make([]store.Assigned, 0, len(as))
	for _, a := range as {
		refs := make([]store.NodeRef, 0, len(a.Nodes))
		for _, n := range a.Nodes {
			refs = append(refs, store.NodeRef{Node: n, Label: byID[n]})
		}
		out = append(out, store.Assigned{As: a.As, Nodes: refs})
	}
	return out
}

func nonce() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
