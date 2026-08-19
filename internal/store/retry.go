package store

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/taeels/enode/internal/contract"
)

// MaybeRetry 는 ⑥ 의 재시도 루프를 돈다 (ADR-013).
//
// ★ 루프를 도는 주체는 Mediator 다 ★ (ADR-014 결정 1) —
// 루프가 agent enode 와 build enode 를 오가는데 둘은 서로를 모른다.
// 이것이 시퀀서 결정의 결정적 근거였고, 여기가 그것이 실물이 되는 자리다.
//
//	write_test(agent) ──▶ Mediator ──▶ parent_build(builder)
//	       ▲                                  │ 실패
//	       └──── feedback: build_log ─────────┘
//
// ★ INVARIANTS §2 의 "재실행하지 않는다" 와 부딪히지 않는가 ★
// 그건 ★ 크래시 후 재개 ★ 에 대한 것이다. 이 루프는 계약이 미리 선언한 것이고,
// 계약 작성자가 반복해도 안전한 단계만 넣을 책임을 진다 —
// 보드를 두 번 flash 하는 단계를 검증자로 쓰면 안 된다.
func (s *Store) MaybeRetry(ctx context.Context, runID string, doneSeq int, log *slog.Logger) (bool, error) {
	run, err := s.GetRun(ctx, runID)
	if err != nil {
		return false, err
	}
	if doneSeq-1 < 0 || doneSeq-1 >= len(run.Contract.Steps) {
		return false, nil
	}
	validator := run.Contract.Steps[doneSeq-1]

	// 이 단계를 검증자로 쓰는 앞 단계를 찾는다.
	target, targetSeq := contract.Step{}, 0
	for i, st := range run.Contract.Steps {
		if st.ValidateWith == validator.ID && i+1 < doneSeq {
			target, targetSeq = st, i+1
		}
	}
	if targetSeq == 0 {
		return false, nil
	}

	// 검증자가 통과했으면 아무것도 안 한다.
	ok, err := s.stepPassed(ctx, runID, doneSeq)
	if err != nil || ok {
		return false, err
	}

	var attempt int
	if err := s.pool.QueryRow(ctx,
		`SELECT attempt FROM steps WHERE run_id=$1 AND seq=$2`, runID, targetSeq).Scan(&attempt); err != nil {
		return false, err
	}
	max := target.MaxAttempts
	if max <= 0 {
		max = 1
	}
	if attempt+1 >= max {
		// ★ 소진했다. 재시도하지 않는다. ★
		// 성패는 여기서 정하지 않는다 — success_when 의 within_attempts 가 판정한다
		// (ADR-004: 계약에 선언된 기계적 조건으로만).
		log.Warn("재시도 소진", "run", runID, "step", target.ID, "attempts", attempt+1)
		return false, nil
	}

	// ★ 되먹여 다시 돌린다 ★ — 대상과 검증자를 함께 PENDING 으로 되돌린다.
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx,
		`UPDATE steps SET state='PENDING', attempt=attempt+1, result=NULL,
		        started_at=NULL, ended_at=NULL
		  WHERE run_id=$1 AND seq=$2`, runID, targetSeq); err != nil {
		return false, err
	}
	// ★ 검증자의 회차도 올린다 ★ — 다시 도는 것이 사실이고,
	// 산출물 최신성 순서가 (회차, 순번) 이라 한 회차 안에서 순번이 맞아야 한다.
	if _, err := tx.Exec(ctx,
		`UPDATE steps SET state='PENDING', attempt=attempt+1, result=NULL,
		        started_at=NULL, ended_at=NULL
		  WHERE run_id=$1 AND seq=$2`, runID, doneSeq); err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	log.Info("검증 실패 — 되먹여 재시도", "run", runID, "step", target.ID,
		"validator", validator.ID, "attempt", attempt+2, "max", max)
	return true, nil
}

// stepPassed 는 검증자가 통과했는지다.
// ★ 여기서 Run 의 성패를 정하지 않는다 ★ — 루프를 돌지 말지만 정한다.
// 성패는 끝에서 success_when 이 정한다 (ADR-004).
func (s *Store) stepPassed(ctx context.Context, runID string, seq int) (bool, error) {
	var state string
	var raw []byte
	if err := s.pool.QueryRow(ctx,
		`SELECT state, result FROM steps WHERE run_id=$1 AND seq=$2`, runID, seq).
		Scan(&state, &raw); err != nil {
		return false, err
	}
	if state != "DONE" || len(raw) == 0 {
		return false, nil
	}
	var r StepResult
	if err := json.Unmarshal(raw, &r); err != nil {
		return false, err
	}
	if r.ExitCode != nil {
		return *r.ExitCode == 0, nil
	}
	return len(r.Produced) > 0, nil
}
