package store

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/match"
)

// 획득 결과의 어휘. ★ 계약이 dispatch.to 에 적는 이름이 이것이다 ★ (ADR-022 §7.5).
const (
	Acquired    = "acquired"
	Unavailable = "unavailable"
)

// runAcquires 는 지금 수행할 수 있는 획득 단계들을 처리한다 (ADR-022 §7.5 · ADR-024).
//
// ★ 이 단계는 노드에 안 간다 ★ — 잡기 전이므로 uses 가 없고 claim 이 집지 않는다.
// Mediator 가 수행하고, 그래서 「Mediator 가 다음 단계를 만든다」(ADR-014 결정 1)의
// 연장이다. ★ 배분 정책은 여전히 0 개다 ★ — 매처가 고르고 기본키가 강제한다.
//
// ★ I5 를 안 깬다 ★ — ADR-024 가 「요구 자원」을 한 획득 요청의 범위로 정했다.
// 오늘은 요청 하나에 자원 하나이므로 ★ 전부-아니면-전무가 자명하다 ★.
// 실패해도 ★ 이미 쥔 것은 안 놓는다 ★ — 부분 점유가 아니라 정상 점유다.
//
// ★ 실패는 중단이 아니라 값이다 ★ — 결과를 산출물로 내고 dispatch 가 읽는다.
func (s *Store) runAcquires(ctx context.Context, tx pgx.Tx, runID string) error {
	// 한 획득이 끝나면 그 뒤의 획득이 실행 가능해질 수 있으므로 더 없을 때까지 돈다.
	// 매 회차가 PENDING 을 하나씩 줄이므로 ★ 단계 수 안에 멈춘다 ★.
	for {
		seq, err := s.nextAcquire(ctx, tx, runID)
		if err != nil {
			return err
		}
		if seq == 0 {
			return nil
		}
		if err := s.doAcquire(ctx, tx, runID, seq); err != nil {
			return err
		}
	}
}

// nextAcquire 는 ★ needs 가 전부 끝난 ★ 획득 단계 하나를 고른다. 게이트의 술어와
// 같은 것을 본다 — 순서를 정하는 규칙이 두 벌이 되면 안 된다.
func (s *Store) nextAcquire(ctx context.Context, tx pgx.Tx, runID string) (int, error) {
	var seq int
	err := tx.QueryRow(ctx, `
		SELECT s.seq FROM steps s
		 WHERE s.run_id = $1 AND s.state = 'PENDING' AND s.kind = 'acquire'
		   AND NOT EXISTS (
		       SELECT 1 FROM steps p
		        WHERE p.run_id = s.run_id AND p.name = ANY(s.needs)
		          AND p.state NOT IN ('DONE','SKIPPED'))
		 ORDER BY s.seq LIMIT 1`, runID).Scan(&seq)
	if err == pgx.ErrNoRows {
		return 0, nil
	}
	return seq, err
}

func (s *Store) doAcquire(ctx context.Context, tx pgx.Tx, runID string, seq int) error {
	var raw, assignedJSON []byte
	if err := tx.QueryRow(ctx,
		`SELECT contract, assigned FROM runs WHERE run_id=$1`, runID).
		Scan(&raw, &assignedJSON); err != nil {
		return err
	}
	var c contract.Contract
	if err := json.Unmarshal(raw, &c); err != nil {
		return err
	}
	if seq < 1 || seq > len(c.Steps) || c.Steps[seq-1].Acquire == nil {
		return fmt.Errorf("%s#%d 이 획득 단계가 아니다", runID, seq)
	}
	st := c.Steps[seq-1]

	node, label, err := s.tryGrab(ctx, tx, runID, *st.Acquire.Want)
	if err != nil {
		return err
	}
	state, taken, skipped := Acquired, st.Acquire.Acquired, st.Acquire.Unavailable
	if node == "" {
		state, taken, skipped = Unavailable, st.Acquire.Unavailable, st.Acquire.Acquired
	}
	// ★ 결과를 산출물로 남긴다 ★ — 봉인에 "그때 자원이 있었나" 가 남는다(성질 4).
	// ★ 판정 재료가 아니라 기록이다 ★ — 분기는 이미 위에서 정해졌고, 계약이
	// out 에 이 이름을 적으면 success_when 으로도 쓸 수 있다.
	body, err := json.Marshal(map[string]string{"state": state, "node": node})
	if err != nil {
		return err
	}
	if s.Records != nil {
		if _, err := s.Records.WriteBlob(runID, seq, 0, st.ID,
			bytes.NewReader(body), int64(len(body))); err != nil {
			return err
		}
	}
	if node != "" {
		if err := s.bindRole(ctx, tx, runID, st.Acquire.Want.As, node, label, assignedJSON); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE steps SET state='DONE', started_at=now(), ended_at=now(),
		       uses=$3, node_id=$4, result=$5
		 WHERE run_id=$1 AND seq=$2`,
		runID, seq, st.Acquire.Want.As, nullable(node),
		mustJSON(StepResult{Produced: []string{st.ID}})); err != nil {
		return err
	}
	// ★ 안 간 쪽을 닫는다 ★ — 분기와 같은 규칙이고, 같은 전파를 쓴다.
	// PENDING 인 것만 바꾼다: 이미 돈 것을 되돌리지 않는다.
	if _, err := tx.Exec(ctx, `
		UPDATE steps SET state=$3, ended_at=now()
		 WHERE run_id=$1 AND name=$2 AND state='PENDING'`,
		runID, skipped, StepSkipped); err != nil {
		return err
	}
	_ = taken // 간 쪽은 그대로 둔다 — 게이트가 알아서 집는다
	return propagateSkips(ctx, tx, runID)
}

// tryGrab 은 자원 하나를 잡아본다. ★ 못 잡는 것은 오류가 아니다 ★ —
// 빈 노드 이름으로 돌아오고 그것이 "unavailable" 이 된다.
//
// ★ 충돌은 저장 계층이 막는다 ★ — leases(node_id) 기본키가 I1 이고(ADR-019),
// 애플리케이션 로직이 아니라 그 제약이 경쟁을 판정한다. 그래서 세이브포인트로
// 감싸 ★ 충돌이 바깥 트랜잭션을 죽이지 않게 ★ 한다.
func (s *Store) tryGrab(ctx context.Context, tx pgx.Tx, runID string,
	want contract.Require) (node, label string, err error) {
	adverts, err := s.LiveAdverts(ctx)
	if err != nil {
		return "", "", err
	}
	busy, err := s.busyIn(ctx, tx)
	if err != nil {
		return "", "", err
	}
	assign, rej := match.Match([]contract.Require{want}, adverts, busy)
	if rej != nil || len(assign) == 0 || len(assign[0].Nodes) == 0 {
		return "", "", nil // ★ 후보가 없거나 전부 점유됨 ★ — 값으로 돌려준다
	}
	node = assign[0].Nodes[0]
	for _, a := range adverts {
		if a.NodeID == node {
			label = a.Label
			break
		}
	}
	sp, err := tx.Begin(ctx)
	if err != nil {
		return "", "", err
	}
	// ★ 만료는 같은 Run 의 임대에서 물려받는다 ★ — 하나만 먼저 끝나면 그 노드를
	// 잃고, 그러면 이 Run 의 자원이 시간에 따라 갈라진다. 설정값을 여기서 다시
	// 읽으면 ★ 임대 수명이 두 곳에서 정해진다 ★ — 하트비트가 곧 각자 갱신하므로
	// 초기값은 형제와 같기만 하면 된다 (ADR-008 · ADR-016).
	_, err = sp.Exec(ctx, `
		INSERT INTO leases (node_id, run_id, not_after, nonce)
		 SELECT $1, $2, max(not_after), $3 FROM leases WHERE run_id = $2`,
		node, runID, nonce())
	if err != nil {
		_ = sp.Rollback(ctx)
		if isUniqueViolation(err) {
			return "", "", nil // ★ 그 사이에 남이 채갔다 ★ — 이것도 값이다
		}
		return "", "", err
	}
	return node, label, sp.Commit(ctx)
}

// busyIn 은 ★ 이 트랜잭션이 보는 ★ 점유 장부다. 바깥의 BusyNodes 와 같은 것을
// 읽지만 트랜잭션 안에서 봐야 방금 놓은 것(release)이 반영된다.
func (s *Store) busyIn(ctx context.Context, tx pgx.Tx) (map[string]bool, error) {
	rows, err := tx.Query(ctx, `SELECT node_id FROM leases`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	busy := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		busy[id] = true
	}
	return busy, rows.Err()
}

// bindRole 은 잡은 노드를 그 역할에 묶는다 — assigned 에 붙이고,
// ★ 그 역할을 쓰는 단계들의 node_id 를 채운다 ★.
//
// 안 채우면 claim 의 `WHERE s.node_id = $1` 이 그 단계를 영영 못 찾는다:
// CreateRun 때는 이 역할이 없었으므로 빈 채로 들어와 있다.
func (s *Store) bindRole(ctx context.Context, tx pgx.Tx, runID, as, node, label string,
	assignedJSON []byte) error {
	var assigned []Assigned
	if len(assignedJSON) > 0 {
		if err := json.Unmarshal(assignedJSON, &assigned); err != nil {
			return err
		}
	}
	assigned = append(assigned, Assigned{As: as, Nodes: []NodeRef{{Node: node, Label: label}}})
	next, err := json.Marshal(assigned)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE runs SET assigned=$2 WHERE run_id=$1`, runID, next); err != nil {
		return err
	}
	// ★ 빈 문자열도 「아직 없다」 다 ★ — CreateRun 이 그 역할을 몰랐을 때
	// nodeOf 가 빈 값을 넣었다. NULL 만 보면 이 단계들을 영영 못 채운다.
	_, err = tx.Exec(ctx,
		`UPDATE steps SET node_id=$3
		  WHERE run_id=$1 AND uses=$2 AND coalesce(node_id,'') = ''`,
		runID, as, node)
	return err
}

// nonce 는 허가 아티팩트의 일회용 값이다 (ADR-010).
func nonce() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
