package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/match"
)

// 대기열 (ADR-064).
//
// 점유 실패를 죽이지 않고 QUEUED 로 받아 두었다가, 임대가 지워지는 지점마다
// 큐의 앞부터 다시 매칭해 승격한다. 큐는 표가 아니라 runs.state 의 값 하나다 —
// 임대도 노드도 쥐지 않으므로 재기동 후에도 그대로 참이고(INVARIANTS §1.1),
// 되돌릴 것이 없다.
//
// 깨우는 자리는 주기가 아니라 지점이다 — 정산(SettleIfDone) · 취소(Cancel) ·
// 만료 회수(Reap) · 부분 반납(afterStep) · drain 해제를 받은 광고 · 기동 한 번.
// 요청 안의 것은 그 요청의 트랜잭션 안에서 동기로 돈다. CP2 가 「a 가 끝난 뒤
// status <b> 가 RUNNING」을 요구하므로 비동기면 그 줄이 경쟁이 된다.

// 이 저장소의 advisory lock 키 목록.
//
// pg_advisory_xact_lock 은 int64 하나로 잠금을 가른다. 키가 겹치면 서로 모르는
// 두 자리가 같은 줄에 서므로, 새 키는 여기 한 줄로 등재한다.
//
//	queueLockKey   대기열 변경 — 넣기 · 훑기 · 승격
const queueLockKey int64 = 0x656e6f6465000001

// Woken 은 한 훑기가 승격한 Run 들과 그때 올라간 되묻기다.
//
// 승격은 CreateRun 과 같은 몸통을 쓰므로 첫 단계가 되묻기인 계약은 승격되는
// 순간 묻는다. 알림은 커밋 뒤에만 쏘고(ADR-032 §4) 커밋은 부르는 쪽이 하므로,
// 올라간 질문을 돌려주고 부르는 쪽이 커밋한 뒤 Notify 한다.
type Woken struct {
	// Runs 는 승격된 run_id 다. 도착순이다.
	Runs []string
	asks []AskEvent
}

// Notify 는 승격이 올린 되묻기를 알린다. 커밋 뒤에만 부른다.
func (s *Store) Notify(w Woken) { s.PushAsks(w.asks) }

// DrainingNodes 는 소유자가 drain 을 건 노드다 (ADR-063). 제출이 busy 에 합친다 —
// 점유된 노드와 같은 편이고, dry-run 은 busy 를 안 보듯 이것도 안 본다.
func (s *Store) DrainingNodes(ctx context.Context) (map[string]bool, error) {
	return drainingIn(ctx, s.pool)
}

func drainingIn(ctx context.Context, q querier) (map[string]bool, error) {
	rows, err := q.Query(ctx,
		`SELECT node_id FROM nodes WHERE draining <> '' AND expires_at > now()`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

// leaseTTL 은 승격이 발급하는 임대의 수명이다. 안 채워졌으면 설정 기본값과 같다.
func (s *Store) leaseTTL() time.Duration {
	if s.LeaseTTL > 0 {
		return s.LeaseTTL
	}
	return time.Hour
}

// CreateQueuedRun 은 점유 실패한 Run 을 QUEUED 로 넣고, 같은 트랜잭션에서 한 번
// 훑는다. promoted 가 참이면 그 자리에서 RUNNING 이 됐다 — 호출자는 202 가
// 아니라 201 로 답해야 한다.
//
// 넣기와 첫 훑기가 한 트랜잭션이고 같은 advisory lock 아래인 것이 경쟁 창을
// 닫는 방법이다. 따로 하면 「임대를 지운 쪽은 아직 커밋 안 된 이 행을 못 보고,
// 이 쪽은 아직 커밋 안 된 그 삭제를 못 보는」 순간이 생겨 Run 이 다음 해제까지
// 선다. 잠금이 둘을 줄 세우므로 하나는 반드시 상대를 본다.
//
// 같은 run_id 가 이미 있으면 기본키 위반이 그대로 오류다 — 호출자가 다시 읽어
// 200 으로 답한다 (제출 경로의 경쟁 재제출과 같은 갈래).
func (s *Store) CreateQueuedRun(ctx context.Context, r Run) (promoted bool, err error) {
	contractJSON, err := json.Marshal(r.Contract)
	if err != nil {
		return false, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // 커밋됐으면 무해하다
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, queueLockKey); err != nil {
		return false, err
	}
	// assigned · reject · verdict 는 안 넣는다 — SQL NULL 이어야 읽기 쪽의
	// coalesce 가 [] 로 접고, 거절이 아니므로 reject 도 비어 있어야 한다.
	if _, err := tx.Exec(ctx,
		`INSERT INTO runs (run_id, state, principal, contract, work_id, submitter)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		r.RunID, StateQueued, r.Principal, contractJSON,
		nullable(r.Contract.Work.Key()), r.Submitter); err != nil {
		return false, err
	}
	s.log().Info("queued: every candidate node is held", "run", r.RunID)
	w, err := s.wakeQueuedIn(ctx, tx)
	if err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	s.Notify(w)
	for _, id := range w.Runs {
		if id == r.RunID {
			return true, nil
		}
	}
	return false, nil
}

// WakeQueued 는 부르는 트랜잭션 안에서 큐를 훑어 승격한다. 임대를 지운 직후,
// 커밋 전에 부른다. 돌려받은 Woken 은 커밋 뒤 Notify 한다.
func (s *Store) WakeQueued(ctx context.Context, tx pgx.Tx) (Woken, error) {
	return s.wakeQueuedIn(ctx, tx)
}

// WakeQueuedNow 는 자기 트랜잭션을 열어 훑는다 — 기동 · 만료 회수 · drain 해제처럼
// 감싸는 요청 트랜잭션이 없는 자리다. 승격된 run_id 를 돌려준다.
func (s *Store) WakeQueuedNow(ctx context.Context) ([]string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	w, err := s.wakeQueuedIn(ctx, tx)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	s.Notify(w)
	return w.Runs, nil
}

// queuedRow 는 훑기가 읽는 한 줄이다.
type queuedRow struct {
	runID    string
	contract contract.Contract
}

// wakeQueuedIn 은 훑기와 승격의 몸통이다.
//
// 순서는 도착순 하나다 (ADR-064 §2.1 — FIFO). 전체를 훑고 못 가는 것은 건너뛴다 —
// 맨 앞이 보드를 기다린다고 뒤의 다른 자원 요구까지 세우지 않는다. 같은 자원을
// 두고 겨루면 도착순이다. 한 훑기 안에서 앞 행이 잡은 노드는 뒤 행의 busy 에 든다.
//
// 승격 실패는 오류가 아니다. 광고가 만료돼 후보가 0 이 된 Run 도 QUEUED 로
// 남는다 — 노드는 돌아올 수 있고, 시간이 보내지 않는다. 출구는 취소뿐이다.
//
// SKIP LOCKED 인 이유 — 취소는 runs 행을 먼저 잠그고(UPDATE) 임대를 지운 뒤
// 이 잠금을 기다린다. 훑기가 그 행을 FOR UPDATE 로 기다리면 둘이 서로를 기다린다.
// 잠긴 QUEUED 행은 지금 취소되는 중이므로 건너뛰는 것이 맞고, 취소가 끝난 뒤
// 그 취소가 다시 훑는다.
func (s *Store) wakeQueuedIn(ctx context.Context, tx pgx.Tx) (Woken, error) {
	var w Woken
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, queueLockKey); err != nil {
		return w, err
	}
	rows, err := tx.Query(ctx, `
		SELECT run_id, `+liveContract+` FROM runs
		 WHERE state = $1
		 ORDER BY created_at, run_id
		 FOR UPDATE SKIP LOCKED`, StateQueued)
	if err != nil {
		return w, err
	}
	var queued []queuedRow
	for rows.Next() {
		var q queuedRow
		var raw []byte
		if err := rows.Scan(&q.runID, &raw); err != nil {
			rows.Close()
			return w, err
		}
		if err := json.Unmarshal(raw, &q.contract); err != nil {
			rows.Close()
			return w, err
		}
		queued = append(queued, q)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return w, err
	}
	if len(queued) == 0 {
		return w, nil // 빠른 길 — 대기가 없는 함대에서 비용은 위의 한 줄이다
	}

	adverts, err := liveAdvertsIn(ctx, tx)
	if err != nil {
		return w, err
	}
	busy, err := s.busyIn(ctx, tx)
	if err != nil {
		return w, err
	}
	draining, err := drainingIn(ctx, tx)
	if err != nil {
		return w, err
	}
	for id := range draining {
		busy[id] = true
	}

	for _, q := range queued {
		assign, rej := match.Match(q.contract.Requires, adverts, busy)
		if rej != nil {
			continue
		}
		nodes := map[string]bool{}
		for _, a := range assign {
			for _, n := range a.Nodes {
				nodes[n] = true
			}
		}
		// 폭의 상한 (ADR-024 §4.2). 제출이 이미 422 로 걸렀지만 재계획으로
		// 요구가 늘 수 있다 — 넘으면 승격하지 않고 남긴다.
		if s.MaxLeasesPerRun > 0 && len(nodes) > s.MaxLeasesPerRun {
			continue
		}
		asks, err := s.promoteIn(ctx, tx, q.runID, q.contract, assign, adverts)
		if err != nil {
			return w, err
		}
		for n := range nodes {
			busy[n] = true
		}
		// 커밋 전의 로그다. 이 트랜잭션이 되돌아가면 부르는 쪽이 오류를 남기므로
		// 두 줄을 같이 읽으면 사실이 선다.
		s.log().Info("promoted from queue", "run", q.runID)
		w.Runs = append(w.Runs, q.runID)
		w.asks = append(w.asks, asks...)
	}
	return w, nil
}

// promoteIn 은 QUEUED 행 하나를 RUNNING 으로 올리고 CreateRun 의 몸통을 붙인다.
func (s *Store) promoteIn(ctx context.Context, tx pgx.Tx, runID string,
	c contract.Contract, assign []match.Assignment, adverts []contract.Advert) ([]AskEvent, error) {
	assigned := labelAssigned(assign, adverts)
	assignedJSON, err := json.Marshal(assigned)
	if err != nil {
		return nil, err
	}
	tag, err := tx.Exec(ctx,
		`UPDATE runs SET state=$2, assigned=$3 WHERE run_id=$1 AND state=$4`,
		runID, StateRunning, assignedJSON, StateQueued)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		// 잠금 아래서는 안 난다 — 나면 훑기 밖에서 누가 이 행을 바꾼 것이다.
		return nil, fmt.Errorf("promoting a run that is no longer queued: %s", runID)
	}
	notAfter := time.Now().Add(s.leaseTTL())
	var grants []LeaseGrant
	seen := map[string]bool{}
	for _, a := range assign {
		for _, n := range a.Nodes {
			if seen[n] { // 한 노드가 여러 역할을 맡으면 임대는 하나다 (ADR-019 결정 2)
				continue
			}
			seen[n] = true
			grants = append(grants, LeaseGrant{NodeID: n, NotAfter: notAfter, Nonce: nonce()})
		}
	}
	r := Run{RunID: runID, State: StateRunning, Contract: c, Assigned: assigned}
	return s.createRunIn(ctx, tx, r, grants, c.Steps)
}

// labelAssigned 는 배정에 광고의 라벨을 붙인다. api 의 label 과 같은 규칙이다 —
// Record 가 노드 귀속을 라벨로 읽으므로 승격도 같은 모양을 남겨야 한다.
func labelAssigned(as []match.Assignment, adverts []contract.Advert) []Assigned {
	byID := map[string]string{}
	for _, a := range adverts {
		byID[a.NodeID] = a.Label
	}
	out := make([]Assigned, 0, len(as))
	for _, a := range as {
		refs := make([]NodeRef, 0, len(a.Nodes))
		for _, n := range a.Nodes {
			refs = append(refs, NodeRef{Node: n, Label: byID[n]})
		}
		out = append(out, Assigned{As: a.As, Nodes: refs})
	}
	return out
}
