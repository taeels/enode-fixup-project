package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/schema"
)

// raiseAsks 는 ★ needs 가 찬 되묻기 단계를 ASKED 로 올린다 ★ (ADR-032).
//
// ★ 대기는 일급 상태다 ★ — 조사한 15개 시스템 전부의 공통점이고, 자원을 쥔 채
// 기다리는 것이 안티패턴의 교과서였다. ASKED 단계는 노드에 안 가고(claim 이
// node_id 로 거른다) 자원을 안 잡는다. 발견은 인박스(PendingAsks)가 한다.
func (s *Store) raiseAsks(ctx context.Context, tx pgx.Tx, runID string) error {
	rows, err := tx.Query(ctx, `
		SELECT s.seq FROM steps s
		 WHERE s.run_id = $1 AND s.state = 'PENDING' AND s.kind = 'ask'
		   AND NOT EXISTS (
		       SELECT 1 FROM steps p
		        WHERE p.run_id = s.run_id AND p.name = ANY(s.needs)
		          AND p.state NOT IN ('DONE','SKIPPED'))`, runID)
	if err != nil {
		return err
	}
	var seqs []int
	for rows.Next() {
		var n int
		if err := rows.Scan(&n); err != nil {
			rows.Close()
			return err
		}
		seqs = append(seqs, n)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(seqs) == 0 {
		return nil
	}
	c, err := s.liveContractIn(ctx, tx, runID)
	if err != nil {
		return err
	}
	for _, seq := range seqs {
		if seq < 1 || seq > len(c.Steps) || c.Steps[seq-1].Ask == nil {
			continue
		}
		var deadline *time.Time
		if t := c.Steps[seq-1].Ask.Timeout; t != nil {
			if d, err := time.ParseDuration(t.After); err == nil {
				at := time.Now().UTC().Add(d)
				deadline = &at
			}
		}
		if _, err := tx.Exec(ctx, `
			UPDATE steps SET state='ASKED', started_at=now(), ask_deadline=$3
			 WHERE run_id=$1 AND seq=$2 AND state='PENDING'`,
			runID, seq, deadline); err != nil {
			return err
		}
		s.log().Info("되묻기 — 답을 기다린다", "run", runID, "seq", seq,
			"step", c.Steps[seq-1].ID)
	}
	return nil
}

// AskView 는 인박스의 항목 하나다 (ADR-032 §4).
// ★ 대기 중인 것만 든다 ★ — 답한 것은 봉인에 있다.
type AskView struct {
	RunID     string          `json:"run_id"`
	Seq       int             `json:"seq"`
	Step      string          `json:"step"`
	Prompt    string          `json:"prompt"`
	Schema    json.RawMessage `json:"schema"`
	Answerers []string        `json:"answerers,omitempty"`
	AskedAt   *time.Time      `json:"asked_at,omitempty"`
	Deadline  *time.Time      `json:"deadline,omitempty"`
	// CanAnswer 는 ★ 관점 필드 ★ 다 — 보는 사람 기준으로 서버가 채운다
	// (GitHub 의 current_user_can_approve 모범). 저장되는 값이 아니다.
	CanAnswer bool `json:"can_answer"`
}

// PendingAsks 는 답을 기다리는 되묻기 전부다 — 인박스의 정본 표면.
func (s *Store) PendingAsks(ctx context.Context) ([]AskView, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT s.run_id, s.seq, s.name, s.started_at, s.ask_deadline,
		       coalesce(r.contract_versions -> -1, r.contract)
		  FROM steps s JOIN runs r ON r.run_id = s.run_id
		 WHERE s.state = 'ASKED' AND r.state = 'RUNNING'
		 ORDER BY s.started_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AskView{}
	for rows.Next() {
		var v AskView
		var raw []byte
		if err := rows.Scan(&v.RunID, &v.Seq, &v.Step, &v.AskedAt, &v.Deadline, &raw); err != nil {
			return nil, err
		}
		var c contract.Contract
		if json.Unmarshal(raw, &c) != nil || v.Seq < 1 || v.Seq > len(c.Steps) {
			continue
		}
		st := c.Steps[v.Seq-1]
		if st.Ask == nil {
			continue
		}
		v.Prompt = st.Ask.Prompt
		v.Answerers = st.Ask.Answerers
		if len(st.Out) == 1 {
			if b, err := json.Marshal(st.Schema[st.Out[0]]); err == nil {
				v.Schema = b
			}
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

var (
	// ErrNoAsk 는 답을 기다리는 단계가 아니다 — 없거나, 이미 답했거나, ask 가 아니다.
	ErrNoAsk = errors.New("답을 기다리는 단계가 아니다")
	// ErrNotAnswerer 는 answerers 에 없는 사람이다.
	ErrNotAnswerer = errors.New("이 질문에 답할 수 있는 사람이 아니다")
)

// SchemaViolation 은 답이 질문의 형태를 어겼다 — PUT blob 의 422 와 같은 자리다.
type SchemaViolation struct{ Details string }

func (e *SchemaViolation) Error() string { return "스키마 위반 — " + e.Details }

// AnswerStep 은 ★ 답을 받는다 ★ (ADR-032 §1②) — 주소 있는 단일 쓰기.
//
// 답은 산출물이다: 스키마 검증을 통과해야 저장되고(강제 지점 하나 — ADR-020),
// 단계가 DONE 이 되면서 ★ answered_by · answered_at ★ 이 기록에 남고,
// dispatch · 전파 · 다음 되묻기까지 보고와 같은 트랜잭션 경로를 탄다.
func (s *Store) AnswerStep(ctx context.Context, runID string, seq int,
	principal string, body []byte, limit int64) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var state string
	var attempt int
	err = tx.QueryRow(ctx, `
		SELECT s.state, s.attempt FROM steps s JOIN runs r ON r.run_id = s.run_id
		 WHERE s.run_id=$1 AND s.seq=$2 AND s.kind='ask' AND r.state='RUNNING'
		 FOR UPDATE OF s`, runID, seq).Scan(&state, &attempt)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && state != "ASKED") {
		return false, ErrNoAsk
	}
	if err != nil {
		return false, err
	}
	c, err := s.liveContractIn(ctx, tx, runID)
	if err != nil {
		return false, err
	}
	if seq < 1 || seq > len(c.Steps) || c.Steps[seq-1].Ask == nil {
		return false, ErrNoAsk
	}
	st := c.Steps[seq-1]
	if as := st.Ask.Answerers; len(as) > 0 {
		ok := false
		for _, a := range as {
			if a == principal {
				ok = true
				break
			}
		}
		if !ok {
			return false, ErrNotAnswerer
		}
	}
	name := st.Out[0]
	if vs := schema.Validate(st.Schema[name], body); len(vs) > 0 {
		parts := make([]string, 0, len(vs))
		for _, v := range vs {
			parts = append(parts, v.String())
		}
		// ★ 어긴 답은 저장하지 않는다 ★ — 질문은 그대로 열려 있다. 다시 답하면 된다.
		return false, &SchemaViolation{Details: strings.Join(parts, " / ")}
	}
	if s.Records == nil {
		return false, fmt.Errorf("기록 저장소가 없다")
	}
	if _, err := s.Records.WriteBlob(runID, seq, attempt, name,
		bytes.NewReader(body), limit); err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE steps SET state='DONE', ended_at=now(), result=$3
		 WHERE run_id=$1 AND seq=$2`,
		runID, seq, mustJSON(StepResult{
			Produced: []string{name}, AnsweredBy: principal})); err != nil {
		return false, err
	}
	s.log().Info("되묻기 — 답을 받았다", "run", runID, "step", st.ID, "by", principal)
	// ★ 보고와 같은 길이다 ★ — 되돌림을 먼저 보고, 안 되돌리면 효과를 적용한다.
	rolled, err := s.rollBack(ctx, tx, runID, seq)
	if err != nil {
		return false, err
	}
	if rolled {
		return true, tx.Commit(ctx)
	}
	if err := s.afterStep(ctx, tx, runID, seq); err != nil {
		return false, err
	}
	return false, tx.Commit(ctx)
}

// liveContractIn 은 트랜잭션 안에서 지금 유효한 계약을 읽는다.
func (s *Store) liveContractIn(ctx context.Context, tx pgx.Tx, runID string) (contract.Contract, error) {
	var raw []byte
	var c contract.Contract
	if err := tx.QueryRow(ctx,
		`SELECT `+liveContract+` FROM runs WHERE run_id=$1`, runID).Scan(&raw); err != nil {
		return c, err
	}
	return c, json.Unmarshal(raw, &c)
}

// ExpireAsks 는 기한이 지난 되묻기를 실패시킨다 (ADR-032 §2 — then:"fail").
// 회수(reap) 주기에서 돈다 — ★ 여기서도 시간이 감시자다 ★.
func (s *Store) ExpireAsks(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		UPDATE steps s SET state='FAILED', ended_at=now(),
		       result = coalesce(result,'{}'::jsonb) || $1::jsonb
		  FROM runs r
		 WHERE r.run_id = s.run_id AND r.state = 'RUNNING'
		   AND s.state = 'ASKED' AND s.ask_deadline IS NOT NULL
		   AND s.ask_deadline <= now()
		 RETURNING s.run_id`,
		mustJSON(map[string]string{"error": "되묻기 기한이 지났다"}))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	seen := map[string]bool{}
	var runs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if !seen[id] {
			seen[id] = true
			runs = append(runs, id)
		}
	}
	return runs, rows.Err()
}
