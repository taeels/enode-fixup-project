// Package api 는 Mediator 의 HTTP 표면이다.
//
// 정본은 enode-design 저장소의 protocol/mediator-api.md 다.
// 라우팅은 표준 라이브러리만 쓴다 — Go 1.22+ 의 ServeMux 가 메서드와 경로
// 패턴을 받으므로 라우터 의존성이 필요 없다.
package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/taeels/enode/internal/config"
	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/match"
	"github.com/taeels/enode/internal/record"
	"github.com/taeels/enode/internal/schema"
	"github.com/taeels/enode/internal/store"
)

type Server struct {
	st      *store.Store
	records *record.Store
	cfg     config.Config
	log     *slog.Logger
}

func New(st *store.Store, cfg config.Config, log *slog.Logger) *Server {
	return &Server{st: st, records: st.Records, cfg: cfg, log: log}
}

// needRecords 는 Record 저장소 없이 호출된 경우를 막는다.
// cmd/mediator 는 항상 붙이지만, 없으면 ★ 패닉이 아니라 503 ★ 이어야 한다.
func (s *Server) needRecords(w http.ResponseWriter) bool {
	if s.records == nil {
		fail(w, 503, "record store is not configured")
		return false
	}
	return true
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/nodes", s.auth(s.postNodes))
	mux.HandleFunc("POST /v1/nodes/{id}/claim", s.auth(s.postClaim))
	mux.HandleFunc("POST /v1/runs/{run}/steps/{seq}/result", s.auth(s.postResult))
	mux.HandleFunc("POST /v1/runs", s.auth(s.postRuns))
	mux.HandleFunc("POST /v1/runs/dry-run", s.auth(s.postDryRun))
	mux.HandleFunc("GET /v1/runs/{id}", s.auth(s.getRun))
	mux.HandleFunc("GET /v1/capabilities", s.auth(s.getCapabilities))
	mux.HandleFunc("GET /v1/asks", s.auth(s.getAsks))
	mux.HandleFunc("POST /v1/runs/{run}/steps/{seq}/answer", s.auth(s.postAnswer))
	mux.HandleFunc("GET /v1/runs/{id}/ledger", s.auth(s.getLedger))
	mux.HandleFunc("GET /v1/runs/{id}/record", s.auth(s.getRecord))
	mux.HandleFunc("POST /v1/runs/{id}/cancel", s.auth(s.postCancel))
	mux.HandleFunc("PUT /v1/runs/{run}/steps/{seq}/log", s.auth(s.putLog))
	mux.HandleFunc("PUT /v1/runs/{run}/steps/{seq}/blob/{name}", s.auth(s.putBlob))
	mux.HandleFunc("GET /v1/runs/{run}/blob/{name}", s.auth(s.getBlob))
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
			fail(w, 401, "missing or invalid token")
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
	Verdict  *store.Verdict   `json:"verdict,omitempty"` // ⑩ 의 대조 결과
	// Steps 는 ★ 실행 중 관측 ★ 이다 (ADR-025). 폭이 1 을 넘으면 여러 가지가
	// 각각 다른 상태에 있고, GET record 는 종료 전이면 409 다(I4).
	// ★ 새 표면을 만들지 않고 이미 있는 조회를 넓힌다 ★ — 표면 개수가 비용이다.
	Steps []store.StepView `json:"steps,omitempty"`
	// Warnings 는 ★ 받았지만 뜻대로 안 돌 것 ★ 이다 (ADR-061 §2).
	// 제출을 막지 않는다 — 계약 저자가 읽고 고칠 자리다.
	Warnings []string `json:"warnings,omitempty"`
}

// view 는 Run 하나를 밖에서 읽는 형태로 만든다.
//
// ★ 단계를 못 읽어도 Run 상태는 준다 ★ — 관측이 조회를 막으면 안 된다.
// 배정 전(ALLOCATING)이면 단계가 없는 것이 정상이고, 그때는 빈 채로 나간다.
func (s *Server) view(ctx context.Context, r *store.Run) runView {
	steps, err := s.st.Steps(ctx, r.RunID)
	if err != nil {
		s.log.Error("cannot query steps", "run", r.RunID, "err", err)
	}
	return runView{RunID: r.RunID, State: r.State, Assigned: r.Assigned,
		Reject: r.Reject, Verdict: r.Verdict, Steps: steps}
}

// ── POST /v1/nodes — 광고 + 하트비트 ──────────────────────────────────────
//
// ★ 응답이 임대의 갱신이자 취소 통보다 ★ (ADR-016).
// 목록은 델타가 아니라 ★ 전부 ★ 이므로 목록에 없는 것이 곧 없는 것이다.
type advertResponse struct {
	Leases []store.LeaseRow `json:"leases"`
	// RenewSeconds 는 ★ 다음에 언제 다시 말할지 ★ 다 (ADR-028).
	//
	// ★ 만료를 계산하는 쪽이 주기도 말한다 ★ — 그러지 않으면 같은 하나를
	// 두 곳에서 정하게 되고, 어긋나면 ★ 노드가 조용히 함대에서 사라진다 ★:
	// 광고는 만료됐는데 claim 은 롱폴이라 계속 돌아서 ★ 기존 Run 은 멀쩡하고
	// 새 Run 만 422 ★ 를 받는다. 아무도 경고하지 않는다.
	RenewSeconds int `json:"renew_seconds"`
}

func (s *Server) postNodes(w http.ResponseWriter, r *http.Request) {
	var a contract.Advert
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		fail(w, 400, "cannot parse advertisement: "+err.Error())
		return
	}
	if a.NodeID == "" {
		fail(w, 400, "node_id is missing")
		return
	}
	// 광고는 만료된다 (ADR-012). 만료 = 갱신 주기의 배수다 (ADR-016).
	// ★ 그 주기를 응답으로 내려보낸다 ★ — 노드가 자기 플래그로 정하면
	// 이 계산과 어긋날 수 있다 (ADR-028).
	ttl := time.Duration(s.cfg.Lease.RenewSeconds*s.cfg.Lease.NotAfterFactor) * time.Second
	if err := s.st.UpsertAdvert(r.Context(), a, principal(r), ttl); err != nil {
		s.log.Error("cannot store advertisement", "node", a.NodeID, "err", err)
		fail(w, 503, "store failed")
		return
	}
	// ★ 재시작 판정이 임대 갱신보다 먼저다 ★ (ADR-030) — 다른 생이 집어둔
	// 단계를 실패시키고 그 Run 을 정산하면 임대가 함께 풀리므로, 아래 갱신
	// 응답에서 그 임대가 ★ 빠진 채로 ★ 나간다. 노드는 목록에 없는 것을 보고
	// 남은 일이 없음을 안다 — 새 통보 채널이 아니라 ADR-016 의 그 규칙이다.
	if a.Instance != "" {
		runs, err := s.st.FailRestarted(r.Context(), a.NodeID, a.Instance)
		if err != nil {
			s.log.Error("cannot detect node restart", "node", a.NodeID, "err", err)
		}
		for _, runID := range runs {
			state, err := s.st.SettleIfDone(r.Context(), runID)
			if err != nil {
				s.log.Error("cannot settle after node restart", "run", runID, "err", err)
				continue
			}
			s.log.Warn("node restarted; in-flight steps marked failed",
				"node", a.NodeID, "run", runID, "state", state)
		}
	}
	// ★ 살아 있다고 말하면 살아 있을 권한을 받는다 ★
	// not_after 는 갱신 주기의 배수로 준다 — 하트비트를 한 번 놓쳐도 안 죽게
	// (ADR-016: "실패한 하트비트 하나는 중단 신호가 아니다").
	leaseTTL := time.Duration(s.cfg.Lease.RenewSeconds*s.cfg.Lease.NotAfterFactor) * time.Second
	leases, err := s.st.RenewLeases(r.Context(), a.NodeID, leaseTTL)
	if err != nil {
		s.log.Error("cannot renew leases", "node", a.NodeID, "err", err)
		fail(w, 503, "renew failed")
		return
	}
	write(w, 200, advertResponse{Leases: leases, RenewSeconds: s.cfg.Lease.RenewSeconds})
}

// ── POST /v1/nodes/{id}/claim — ★ 유일한 비멱등 지점 ★ ────────────────────
//
// 롱폴이다. 할 일이 없으면 시간이 다 될 때까지 기다렸다가 204 로 답한다.
// enode 는 204 를 정상으로 보고 즉시 다시 건다.
//
// ※ MVP 는 짧은 주기로 DB 를 다시 본다. ADR-015 가 적어둔 LISTEN/NOTIFY 는
//
//	노드가 늘어 폴링이 부담이 될 때의 최적화다. 지금은 넷이다.
func (s *Server) postClaim(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")
	// ★ 어느 「생」이 묻는가 ★ (ADR-030) — 같은 생이 다시 물으면 들고 있던 것을
	// 재전달한다. 헤더가 없으면 옛 enode 다: 오늘 그대로 동작한다.
	instance := r.Header.Get("X-Enode-Instance")
	deadline := time.Now().Add(time.Duration(s.cfg.Claim.LongPollSeconds) * time.Second)
	tick := time.NewTicker(time.Second)
	defer tick.Stop()

	for {
		c, err := s.st.ClaimStep(r.Context(), nodeID, instance)
		switch {
		case err == nil:
			write(w, 200, c)
			return
		case !errors.Is(err, store.ErrNoWork):
			s.log.Error("claim failed", "node", nodeID, "err", err)
			fail(w, 503, "claim failed")
			return
		}
		if time.Now().After(deadline) {
			w.WriteHeader(204)
			return
		}
		select {
		case <-r.Context().Done(): // 클라이언트가 끊었다 — 정상이다 (ADR-015 §5)
			return
		case <-tick.C:
		}
	}
}

// ── POST /v1/runs/{run}/steps/{seq}/result ───────────────────────────────
//
// ★ 이 보고를 받은 Mediator 가 다음 단계를 만든다 ★ (ADR-014 결정 1).
// INVARIANTS §2 의 RUNNING → RUNNING 이 여기서 일어나며 주체는 Mediator 다.
//
// ★ 경로가 두 세그먼트인 이유 ★ — step_id 를 run_id#NN 한 덩어리로 URL 에 넣으면
// '#' 이 프래그먼트 구분자라 서버까지 오지 않는다. 이스케이프 규칙을 넷이 기억하게
// 하는 것보다 복합키를 경로로 쪼개는 편이 틀릴 여지가 없다.
// run_id#NN 은 ★ 사람이 읽고 Record 에 남는 표기 ★ 로만 쓴다.
func (s *Server) postResult(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("run")
	seq, err := strconv.Atoi(r.PathValue("seq"))
	if err != nil || seq <= 0 {
		fail(w, 400, "invalid step sequence: "+r.PathValue("seq"))
		return
	}
	var body struct {
		Node string `json:"node"`
		store.StepResult
		Produced []string `json:"produced"`
		Error    string   `json:"error"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		fail(w, 400, "cannot parse result: "+err.Error())
		return
	}
	res := body.StepResult
	res.Produced, res.Error = body.Produced, body.Error

	// ★ 여기서 성패를 판정하지 않는다 ★ — 완주했는지만 본다.
	// exit_code 2 로 끝난 빌드도 완주한 것이고, 그게 성공인지는 success_when 이
	// 판정한다 (ADR-004 · I3). 여기서 가로채면 O4 가 성립하지 않는다.
	completed := res.Error == ""
	// ★ 되돌림 판단이 ReportStep 안으로 들어갔다 ★ (2026-08-21 실측) —
	// 밖에서 하면 그 사이에 이 단계의 효과(갈림길 · 자원 · 계획)가 적용되고,
	// 되돌려도 ★ 그것들은 안 돌아온다 ★.
	rolled, err := s.st.ReportStep(r.Context(), runID, seq, body.Node, completed, res)
	if err != nil {
		fail(w, 409, err.Error())
		return
	}
	// ★ 되돌려졌으면 아직 진행 중이므로 정산하지 않는다 ★.
	if rolled {
		write(w, 200, map[string]any{"run_id": runID, "seq": seq, "rolled_back": true})
		return
	}
	state, err := s.st.SettleIfDone(r.Context(), runID)
	if err != nil {
		s.log.Error("settle failed", "run", runID, "err", err)
		fail(w, 503, "settle failed")
		return
	}
	if state != "" {
		s.log.Info("run finished", "run", runID, "state", state)
	}
	write(w, 200, map[string]any{"run_id": runID, "seq": seq, "run_state": state})
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
		fail(w, 400, "cannot parse contract: "+err.Error())
		return
	}
	if err := c.Validate(); err != nil {
		fail(w, 400, err.Error())
		return
	}
	// ★ 경고는 막지 않는다 ★ (ADR-061 §2.3) — 응답에 싣고 로그에 남긴다.
	warnings := contract.Warnings(c)
	for _, wmsg := range warnings {
		s.log.Warn("contract warning", "run", c.RunID, "warning", wmsg)
	}
	ctx := r.Context()

	// ★ 같은 run_id 재제출은 200 + 기존 Run ★ (INVARIANTS §2 첫 행).
	// run_id 가 (change-id, patchset) 에서 결정적으로 유도되므로 폴링 커서가 필요 없다.
	if !dry {
		if existing, err := s.st.GetRun(ctx, c.RunID); err == nil {
			write(w, 200, s.view(ctx, existing))
			return
		} else if !errors.Is(err, store.ErrNotFound) {
			s.log.Error("cannot query run", "run", c.RunID, "err", err)
			fail(w, 503, "query failed")
			return
		}
	}

	adverts, err := s.st.LiveAdverts(ctx)
	if err != nil {
		s.log.Error("cannot query advertisements", "err", err)
		fail(w, 503, "query failed")
		return
	}

	busy := map[string]bool{}
	if !dry {
		if busy, err = s.st.BusyNodes(ctx); err != nil {
			s.log.Error("cannot query leases", "err", err)
			fail(w, 503, "query failed")
			return
		}
	}

	assign, rej := match.Match(c.Requires, adverts, busy)
	if rej != nil {
		if !dry {
			// 거절도 기록한다 — 왜 안 돌았는지가 없으면 껍데기가 재시도를 못 정한다.
			run := store.Run{RunID: c.RunID, Principal: principal(r), Contract: c, Reject: rej}
			if err := s.st.CreateRejectedRun(ctx, run); err != nil {
				s.log.Error("cannot record rejection", "run", c.RunID, "err", err)
			}
		}
		fail(w, rej.Code, rej.Reason)
		return
	}
	// ★ 폭의 상한 ★ (ADR-024 §4.2) — 한 Run 이 동시에 쥘 수 있는 노드 수.
	// t=0 에 이미 넘는 요구는 ★ 영구 거절(422) ★ 이다: 다시 내도 같기 때문이다.
	// 실행 중 획득이 넘는 것은 여기가 아니라 acquire 가 "unavailable" 로 낸다.
	if max := s.cfg.Lease.MaxPerRun; max > 0 {
		nodes := map[string]bool{}
		for _, a := range assign {
			for _, n := range a.Nodes {
				nodes[n] = true
			}
		}
		if len(nodes) > max {
			rej := &match.Reject{Code: match.CodeNoCandidate,
				Reason: fmt.Sprintf("requires exceeds the width limit: %d nodes, limit %d", len(nodes), max)}
			if !dry {
				run := store.Run{RunID: c.RunID, Principal: principal(r), Contract: c, Reject: rej}
				if err := s.st.CreateRejectedRun(ctx, run); err != nil {
					s.log.Error("cannot record rejection", "run", c.RunID, "err", err)
				}
			}
			fail(w, rej.Code, rej.Reason)
			return
		}
	}
	if dry {
		write(w, 200, runView{RunID: c.RunID, State: "DRY_RUN", Assigned: label(assign, adverts),
			Warnings: warnings})
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
		fail(w, match.CodeAllBusy, "another run took the node during allocation")
		return
	case err != nil:
		// 같은 run_id 로 동시에 들어온 경우도 여기로 온다 — 다시 읽어 200 으로 답한다.
		if existing, gerr := s.st.GetRun(ctx, c.RunID); gerr == nil {
			write(w, 200, s.view(ctx, existing))
			return
		}
		s.log.Error("cannot create run", "run", c.RunID, "err", err)
		fail(w, 503, "create failed")
		return
	}
	v := s.view(ctx, &run)
	v.Warnings = warnings
	write(w, 201, v)
}

// ── GET /v1/runs/{id} ────────────────────────────────────────────────────

func (s *Server) getRun(w http.ResponseWriter, r *http.Request) {
	run, err := s.st.GetRun(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		fail(w, 404, "no such run")
		return
	}
	if err != nil {
		s.log.Error("cannot query run", "err", err)
		fail(w, 503, "query failed")
		return
	}
	write(w, 200, s.view(r.Context(), run))
}

// ── GET /v1/asks — 인박스 ────────────────────────────────────────────────
//
// ★ 폴링 인박스가 정본이다 ★ (ADR-032 §4 · Airflow 모범). 대기 중인 것만 든다 —
// 답한 것은 봉인에 있다. can_answer 는 ★ 관점 필드 ★ 다: 보는 사람 기준으로
// 서버가 채운다 (GitHub 의 current_user_can_approve 모범).
func (s *Server) getAsks(w http.ResponseWriter, r *http.Request) {
	asks, err := s.st.PendingAsks(r.Context())
	if err != nil {
		s.log.Error("cannot query inbox", "err", err)
		fail(w, 503, "query failed")
		return
	}
	me := principal(r)
	for i := range asks {
		asks[i].CanAnswer = len(asks[i].Answerers) == 0
		for _, a := range asks[i].Answerers {
			if a == me {
				asks[i].CanAnswer = true
				break
			}
		}
	}
	write(w, 200, map[string]any{"asks": asks})
}

// ── POST /v1/runs/{run}/steps/{seq}/answer ───────────────────────────────
//
// ★ 답은 주소 있는 단일 쓰기다 ★ (ADR-032 §1②) — 본문이 곧 답이고 산출물이 된다.
// 스키마 위반이면 422 로 저장되지 않고 ★ 질문은 열린 채 남는다 ★ — 다시 답하면 된다.
func (s *Server) postAnswer(w http.ResponseWriter, r *http.Request) {
	if !s.needRecords(w) {
		return
	}
	runID := r.PathValue("run")
	seq, err := strconv.Atoi(r.PathValue("seq"))
	if err != nil || seq <= 0 {
		fail(w, 400, "invalid step sequence")
		return
	}
	limit := s.cfg.Artifacts.MaxBlobBytes
	body, err := io.ReadAll(io.LimitReader(r.Body, limit+1))
	if err != nil {
		fail(w, 503, "read failed")
		return
	}
	if int64(len(body)) > limit {
		fail(w, 413, "answer exceeds the size limit")
		return
	}
	rolled, err := s.st.AnswerStep(r.Context(), runID, seq, principal(r), body, limit)
	var sv *store.SchemaViolation
	switch {
	case errors.Is(err, store.ErrNoAsk):
		fail(w, 409, "step is not awaiting an answer: unknown, already answered, or expired")
		return
	case errors.Is(err, store.ErrNotAnswerer):
		fail(w, 403, "principal is not an allowed answerer")
		return
	case errors.As(err, &sv):
		fail(w, 422, sv.Error())
		return
	case err != nil:
		s.log.Error("cannot process answer", "run", runID, "seq", seq, "err", err)
		fail(w, 503, "cannot process answer")
		return
	}
	if rolled {
		write(w, 200, map[string]any{"run_id": runID, "seq": seq, "rolled_back": true})
		return
	}
	state, err := s.st.SettleIfDone(r.Context(), runID)
	if err != nil {
		s.log.Error("settle failed", "run", runID, "err", err)
		fail(w, 503, "settle failed")
		return
	}
	write(w, 200, map[string]any{"run_id": runID, "seq": seq, "run_state": state})
}

// ── GET /v1/runs/{id}/ledger ─────────────────────────────────────────────
//
// ★ 원장은 목록이다. 본문이 아니다 ★ (ADR-023 §6.3).
//
// 원장 전체를 하네스에 깔면 컨텍스트가 터지고 비용이 든다. 그래서 여기서는
// 메타만 주고, 본문이 필요하면 ★ 이미 있는 blob 경로 ★ 로 가져간다 —
// ★ 새 표면이 하나이고 새 의미가 0 개다 ★.
//
// ★ 종료 전에도 답한다 ★ — 이것은 Record 가 아니다 (ADR-025 와 같은 이유).
// 그래서 GET record 의 409 를 우회하지 않는다: 여기서 나가는 것은
// ★ 무엇이 있는가 ★ 이지 무슨 일이 있었나가 아니다.
func (s *Server) getLedger(w http.ResponseWriter, r *http.Request) {
	entries, err := s.st.Ledger(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		fail(w, 404, "no such run")
		return
	}
	if err != nil {
		s.log.Error("cannot query ledger", "run", r.PathValue("id"), "err", err)
		fail(w, 503, "query failed")
		return
	}
	write(w, 200, map[string]any{"entries": entries})
}

// ── GET /v1/capabilities ─────────────────────────────────────────────────
//
// ★ 이것이 우리 층의 tools/list 다 ★ (ADR-012 가 MCP 에 물어보던 것의 대칭).
// 다만 돌려주는 것은 모델이 읽는 산문이 아니라 ★ 스케줄러가 평가하는 술어 ★ 다.
//
// ADR-012 가 어휘를 창발시켰기 때문에 이것이 필요하다 — capability 이름과 속성이
// 중앙에 선언되지 않고 enode 광고로만 존재하므로, 읽는 경로가 없으면
// ★ 계약을 쓰는 쪽이 문자열을 추측한다 ★. 계약은 사람이 아니라 에이전트가 쓴다.
func (s *Server) getCapabilities(w http.ResponseWriter, r *http.Request) {
	caps, err := s.st.Capabilities(r.Context())
	if err != nil {
		s.log.Error("cannot query capabilities", "err", err)
		fail(w, 503, "query failed")
		return
	}
	write(w, 200, map[string]any{"capabilities": caps})
}

// ── POST /v1/runs/{id}/cancel ────────────────────────────────────────────
//
// ADR-009. 멱등이며 이미 종료면 200 이다.
// X-Enode-Principal 은 ★ 누가 취소했는지 기록 ★ 하는 데만 쓴다 —
// MVP 는 신뢰 경계가 하나라 유효한 토큰을 가진 자는 누구나 취소할 수 있다.
func (s *Server) postCancel(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	state, err := s.st.Cancel(r.Context(), runID, principal(r))
	if errors.Is(err, store.ErrNotFound) {
		fail(w, 404, "no such run")
		return
	}
	if err != nil {
		s.log.Error("cancel failed", "run", runID, "err", err)
		fail(w, 503, "cancel failed")
		return
	}
	s.log.Info("cancelled", "run", runID, "by", principal(r))
	write(w, 200, map[string]any{"run_id": runID, "state": state})
}

// ── PUT /v1/runs/{run}/steps/{seq}/log ───────────────────────────────────
//
// 그 단계가 뱉은 것을 원문 그대로 남긴다 (ADR-005 의 logs/).
// blob 과 자리가 다르다 — blob 은 단계 ★ 사이 ★ 를 오가고 다음 단계가 읽지만,
// log 는 그 단계가 ★ 뱉은 것 ★ 으로 아무도 읽지 않고 기록에만 남는다.
//
// result 보다 ★ 먼저 ★ 올린다 — 단계가 실패해도 로그는 남아야 한다.
func (s *Server) putLog(w http.ResponseWriter, r *http.Request) {
	if !s.needRecords(w) {
		return
	}
	runID := r.PathValue("run")
	seq, err := strconv.Atoi(r.PathValue("seq"))
	if err != nil || seq <= 0 {
		fail(w, 400, "invalid step sequence")
		return
	}
	run, err := s.st.GetRun(r.Context(), runID)
	if errors.Is(err, store.ErrNotFound) {
		fail(w, 404, "no such run")
		return
	}
	if err != nil {
		fail(w, 503, "query failed")
		return
	}
	// ★ I4 — 봉인된 것에는 못 쓴다 ★
	if run.State == store.StateSucceeded || run.State == store.StateFailed {
		fail(w, 410, "run has already finished")
		return
	}
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "step"
	}
	if _, err := s.records.AppendLog(runID, seq, name, r.Body, s.cfg.Artifacts.MaxBlobBytes); err != nil {
		s.log.Error("cannot store log", "run", runID, "seq", seq, "err", err)
		fail(w, 503, "store failed")
		return
	}
	w.WriteHeader(204)
}

// ── blob — 단계 사이를 오가는 산출물 (run-contract §4 별 모양) ────────────
//
// ★ 경로가 비대칭인 이유 ★
//
//	PUT  /v1/runs/{run}/steps/{seq}/blob/{name}   생산자는 자기가 몇 번째인지 안다
//	GET  /v1/runs/{run}/blob/{name}               소비자는 이름만 안다 — 최신을 준다
//
// 계약에서 parent_build 와 patch_build 가 ★ 둘 다 artifact 를 낸다 ★.
// 이름만으로 키를 잡으면 뒤엣것이 앞엣것을 덮어 차분 반증의 두 아티팩트를
// 봉인된 기록에서 구분할 수 없게 된다.
//
// ★ 그리고 여기가 스키마를 검증하는 자리다 ★ (ADR-020) —
// 어긴 산출물은 저장하지 않으므로 produced 가 불만족이 되고,
// success_when 에 schema_ok 같은 새 조건이 생기지 않는다.
func (s *Server) putBlob(w http.ResponseWriter, r *http.Request) {
	if !s.needRecords(w) {
		return
	}
	runID, name := r.PathValue("run"), r.PathValue("name")
	seq, err := strconv.Atoi(r.PathValue("seq"))
	if err != nil || seq <= 0 {
		fail(w, 400, "invalid step sequence")
		return
	}
	run, err := s.st.GetRun(r.Context(), runID)
	if errors.Is(err, store.ErrNotFound) {
		fail(w, 404, "no such run")
		return
	}
	if err != nil {
		fail(w, 503, "query failed")
		return
	}
	// ★ I4 — 봉인된 것에는 못 쓴다 ★
	if run.State == store.StateSucceeded || run.State == store.StateFailed {
		fail(w, 410, "run has already finished")
		return
	}

	limit := s.cfg.Artifacts.MaxBlobBytes
	// 회차는 ★ Mediator 가 안다 ★ — 클라이언트가 보내지 않는다.
	attempt, err := s.st.StepAttempt(r.Context(), runID, seq)
	if err != nil {
		fail(w, 404, "no such step")
		return
	}
	// ★ 늘어난 계약의 스키마도 본다 ★ — 제출본만 보면 계획이 지은 단계의
	// 산출물이 ★ 검증 없이 ★ 저장된다 (「실행하는 쪽은 늘어난 계약을 봐야
	// 한다」의 같은 계열 — ADR-030 커밋이 찾은 결의 연장).
	live, err := s.st.LiveContract(r.Context(), runID)
	if err != nil {
		fail(w, 503, "query failed")
		return
	}
	sch := schemaFor(live, seq, name)
	if sch != nil {
		// 스키마가 걸린 산출물은 검증해야 하므로 먼저 읽는다.
		// 형식 검증 대상이라 상한 안쪽이라는 전제가 있다.
		body, err := io.ReadAll(io.LimitReader(r.Body, limit+1))
		if err != nil {
			fail(w, 503, "read failed")
			return
		}
		if int64(len(body)) > limit {
			fail(w, 413, "blob exceeds the size limit")
			return
		}
		if vs := schema.Validate(sch, body); len(vs) > 0 {
			// ★ 어긴 산출물은 저장하지 않는다 ★ → produced 불만족 → 단계 실패
			// 위반 내역이 feedback 으로 되먹여진다 (ADR-013 의 루프)
			parts := make([]string, 0, len(vs))
			for _, v := range vs {
				parts = append(parts, v.String())
			}
			fail(w, 422, "schema violation: "+strings.Join(parts, " / "))
			return
		}
		if _, err := s.records.WriteBlob(runID, seq, attempt, name, bytes.NewReader(body), limit); err != nil {
			s.log.Error("cannot store blob", "run", runID, "name", name, "err", err)
			fail(w, 503, "store failed")
			return
		}
		w.WriteHeader(204)
		return
	}

	if _, err := s.records.WriteBlob(runID, seq, attempt, name, r.Body, limit); err != nil {
		if errors.Is(err, record.ErrTooBig) {
			// ★ 잘라 저장하지 않는다 ★ — 잘린 산출물은 산출물이 아니다.
			// (로그는 잘라 표시한다. 자리가 다르다.)
			fail(w, 413, "blob exceeds the size limit")
			return
		}
		s.log.Error("cannot store blob", "run", runID, "name", name, "err", err)
		fail(w, 503, "store failed")
		return
	}
	w.WriteHeader(204)
}

func (s *Server) getBlob(w http.ResponseWriter, r *http.Request) {
	if !s.needRecords(w) {
		return
	}
	runID, name := r.PathValue("run"), r.PathValue("name")
	// ★ ADR-018 — 나중에 여기서 302 로 저장소를 가리킨다 ★
	// 클라이언트는 리다이렉트를 따라야 하고, 상한과 저장 위치는 표면의 약속이 아니다.
	// 지금은 직접 서빙한다. enode 코드는 그때도 안 바뀐다.
	f, size, err := s.records.OpenBlob(runID, name)
	if err != nil {
		fail(w, 404, "no such blob")
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	_, _ = io.Copy(w, f)
}

// schemaFor 는 그 단계가 그 이름에 스키마를 달았는지 본다 (ADR-020).
func schemaFor(c contract.Contract, seq int, name string) any {
	if seq-1 < 0 || seq-1 >= len(c.Steps) {
		return nil
	}
	return c.Steps[seq-1].Schema[name]
}

// ── GET /v1/runs/{id}/record ─────────────────────────────────────────────
//
// 봉인된 Record 를 tar 로 돌려준다.
// ★ 성질 4(자기충족)가 전송 형식까지 정한다 ★ — 묶음 하나를 받아 풀면 전부 있다.
//
// 종료 전에 부르면 409 다 — ★ 봉인되지 않은 것은 Record 가 아니다 ★ (I4).
func (s *Server) getRecord(w http.ResponseWriter, r *http.Request) {
	if !s.needRecords(w) {
		return
	}
	runID := r.PathValue("id")
	if _, err := s.st.GetRun(r.Context(), runID); errors.Is(err, store.ErrNotFound) {
		fail(w, 404, "no such run")
		return
	}
	if !s.records.Sealed(runID) {
		fail(w, 409, "run is not sealed yet")
		return
	}
	w.Header().Set("Content-Type", "application/x-tar")
	w.Header().Set("Content-Disposition", `attachment; filename="run-`+runID+`.tar"`)
	if err := s.records.Tar(runID, w); err != nil {
		s.log.Error("cannot export record", "run", runID, "err", err)
	}
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
