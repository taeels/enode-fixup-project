package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// LeaseRow 는 enode 에게 내려보내는 허가 아티팩트다 (ADR-010).
type LeaseRow struct {
	RunID      string    `json:"run_id"`
	Node       string    `json:"node"`
	Capability string    `json:"capability"`
	NotAfter   time.Time `json:"not_after"`
	Nonce      string    `json:"nonce"`
}

// RenewLeases 는 그 노드가 든 임대를 갱신하고 ★ 전부 ★ 돌려준다 (ADR-016).
//
// 이 반환값이 곧 갱신이자 취소 통보다:
//
//	갱신          새 not_after 가 담긴다
//	취소 · 회수   그 임대가 ★ 목록에서 빠진다 ★ → enode 가 다음 단계를 시작하지 않는다
//	Mediator 사망 ★ 응답 자체가 없다 ★ → not_after 가 지나 enode 가 스스로 멈춘다
//
// ★ 델타가 아니라 전부인 것이 핵심이다 ★ — 목록에 없는 것이 곧 없는 것이라
// 취소를 알리는 별도 신호가 필요 없고, 놓칠 이벤트도 없다.
func (s *Store) RenewLeases(ctx context.Context, nodeID string, ttl time.Duration) ([]LeaseRow, error) {
	rows, err := s.pool.Query(ctx, `
		UPDATE leases l
		   SET not_after = now() + $2::interval
		  FROM runs r
		 WHERE l.node_id = $1
		   AND r.run_id = l.run_id
		   AND r.state NOT IN ('SUCCEEDED','FAILED')   -- 끝난 Run 의 임대는 갱신하지 않는다
	 RETURNING l.run_id, l.node_id, l.not_after, l.nonce`,
		nodeID, ttl.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LeaseRow{}
	for rows.Next() {
		var l LeaseRow
		if err := rows.Scan(&l.RunID, &l.Node, &l.NotAfter, &l.Nonce); err != nil {
			return nil, err
		}
		l.Capability = "agent.reason" // ADR-019 — 어휘는 하나뿐이다
		out = append(out, l)
	}
	return out, rows.Err()
}

// Claimed 는 claim 이 돌려주는 할 일 하나다.
type Claimed struct {
	StepID    string          `json:"step_id"`
	RunID     string          `json:"run_id"`
	Seq       int             `json:"seq"`
	Name      string          `json:"name"`
	Uses      string          `json:"uses"`
	Kind      string          `json:"kind"`
	Agent     json.RawMessage `json:"agent,omitempty"`
	Run       []string        `json:"run,omitempty"`
	Workspace json.RawMessage `json:"workspace,omitempty"`
	In        json.RawMessage `json:"in,omitempty"`
	Out       []string        `json:"out,omitempty"`
	Lease     LeaseRow        `json:"lease"`
}

var ErrNoWork = errors.New("할 일이 없다")

// ClaimStep 은 이 노드가 할 단계 하나를 집는다.
//
// ★ SELECT … FOR UPDATE SKIP LOCKED 가 배분 그 자체다 ★ (ADR-015 §3) —
// 여러 노드가 동시에 당겨도 한 단계는 정확히 한 노드에만 간다.
// 손으로 짤 잠금이 없다.
//
// ※ 할당(CreateRun)의 기본키 충돌 + 롤백과는 ★ 다른 기계 ★ 다.
//
//	저쪽은 여러 자원을 한꺼번에 잡거나 전부 포기하는 것이고,
//	이쪽은 대기열에서 하나를 집는 것이다.
//
// 앞 단계가 전부 DONE 이어야 집을 수 있다 — Mediator 가 시퀀서이기 때문이다
// (ADR-014 결정 1). 순서는 계약에 있고 여기서 강제된다.
func (s *Store) ClaimStep(ctx context.Context, nodeID string) (*Claimed, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var c Claimed
	var contractJSON []byte
	err = tx.QueryRow(ctx, `
		SELECT s.run_id, s.seq, s.name, s.uses, s.kind, r.contract
		  FROM steps s
		  JOIN runs r ON r.run_id = s.run_id
		 WHERE s.node_id = $1
		   AND s.state = 'PENDING'
		   AND r.state = 'RUNNING'
		   AND NOT EXISTS (
		       SELECT 1 FROM steps p
		        WHERE p.run_id = s.run_id AND p.seq < s.seq AND p.state <> 'DONE')
		 ORDER BY s.run_id, s.seq
		   FOR UPDATE OF s SKIP LOCKED
		 LIMIT 1`, nodeID).
		Scan(&c.RunID, &c.Seq, &c.Name, &c.Uses, &c.Kind, &contractJSON)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoWork
	}
	if err != nil {
		return nil, err
	}

	// 허가 아티팩트를 함께 내려보낸다 (ADR-010). enode 는 단계를 시작하기 전에
	// not_after 를 확인하고, 지났으면 실행하지 않는다.
	if err := tx.QueryRow(ctx,
		`SELECT run_id, node_id, not_after, nonce FROM leases WHERE node_id = $1 AND run_id = $2`,
		nodeID, c.RunID).
		Scan(&c.Lease.RunID, &c.Lease.Node, &c.Lease.NotAfter, &c.Lease.Nonce); err != nil {
		return nil, fmt.Errorf("임대가 없는데 단계가 배정돼 있다 (%s#%d): %w", c.RunID, c.Seq, err)
	}
	c.Lease.Capability = "agent.reason"

	if _, err := tx.Exec(ctx,
		`UPDATE steps SET state='CLAIMED', started_at=now() WHERE run_id=$1 AND seq=$2`,
		c.RunID, c.Seq); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	c.StepID = fmt.Sprintf("%s#%02d", c.RunID, c.Seq)
	fillFromContract(&c, contractJSON)
	return &c, nil
}

// fillFromContract 는 계약의 단계 정의를 응답에 싣는다.
// 계약을 통째로 저장해둔 덕에 별도 컬럼이 필요 없다 (ADR-020 이 스키마를
// 인라인으로 둔 것과 같은 이유 — 흩어놓으면 나중에 못 모은다).
func fillFromContract(c *Claimed, contractJSON []byte) {
	var raw struct {
		Steps []struct {
			ID        string          `json:"id"`
			Agent     json.RawMessage `json:"agent"`
			Run       []string        `json:"run"`
			Workspace json.RawMessage `json:"workspace"`
			In        json.RawMessage `json:"in"`
			Out       []string        `json:"out"`
		} `json:"steps"`
	}
	if json.Unmarshal(contractJSON, &raw) != nil || c.Seq-1 >= len(raw.Steps) {
		return
	}
	st := raw.Steps[c.Seq-1]
	c.Agent, c.Run, c.Workspace, c.In, c.Out = st.Agent, st.Run, st.Workspace, st.In, st.Out
}

// StepResult 는 enode 가 보고하는 것이다.
type StepResult struct {
	ExitCode *int            `json:"exit_code,omitempty"` // ★ 명령 단계만 ★ (ADR-019)
	Produced []string        `json:"produced,omitempty"`
	Harness  json.RawMessage `json:"harness,omitempty"` // agent 단계만 (ADR-020)
}

// ReportStep 은 단계를 끝낸다. 성공이면 DONE, 아니면 FAILED 다.
// ★ 이 보고를 받은 Mediator 가 다음 단계를 만든다 ★ (ADR-014 결정 1) —
// 다음 claim 이 집을 수 있게 되는 것이 그 형태다.
func (s *Store) ReportStep(ctx context.Context, runID string, seq int, nodeID string, ok bool, res StepResult) error {
	resJSON, err := json.Marshal(res)
	if err != nil {
		return err
	}
	state := "FAILED"
	if ok {
		state = "DONE"
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE steps SET state=$4, ended_at=now(), result=$5
		 WHERE run_id=$1 AND seq=$2 AND node_id=$3 AND state='CLAIMED'`,
		runID, seq, nodeID, state, resJSON)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("집지 않은 단계를 보고했다 (%s#%d)", runID, seq)
	}
	return nil
}
