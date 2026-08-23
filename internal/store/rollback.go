package store

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"github.com/taeels/enode/internal/contract"
)

// rollBack 은 ★ 이 단계가 끝났으니 되돌릴 것이 있는가 ★ 를 본다.
//
// ★ 효과보다 먼저 돈다 ★ (ReportStep 참조) — 되돌릴 단계의 갈림길이 닫히거나
// 자원이 잡히면 되돌려도 그것들이 안 돌아온다.
//
// 되돌리는 문법이 둘이고 ★ 되돌리는 범위가 다르다 ★:
//
//	loop          이 단계 → back_to 까지의 ★ 구간 ★        (ADR-026)
//	validate_with 이 단계를 검증자로 쓰는 앞 단계와 ★ 둘 ★  (ADR-013)
//
// 한 단계에 둘 다 있으면 계약 검증이 400 으로 막으므로 여기서는 순서만 정한다.
func (s *Store) rollBack(ctx context.Context, tx pgx.Tx, runID string, seq int) (bool, error) {
	var raw []byte
	if err := tx.QueryRow(ctx, `SELECT `+liveContract+` FROM runs WHERE run_id=$1`, runID).
		Scan(&raw); err != nil {
		return false, err
	}
	var c contract.Contract
	if err := json.Unmarshal(raw, &c); err != nil {
		return false, err
	}
	if seq < 1 || seq > len(c.Steps) {
		return false, nil
	}
	if done, err := s.loopBack(ctx, tx, runID, seq, c); err != nil || done {
		return done, err
	}
	return s.validateBack(ctx, tx, runID, seq, c)
}

// loopBack 은 구간 반복이다 (ADR-026).
func (s *Store) loopBack(ctx context.Context, tx pgx.Tx, runID string, seq int,
	c contract.Contract) (bool, error) {
	st := c.Steps[seq-1]
	if st.Loop == nil {
		return false, nil
	}
	from := 0
	for i, other := range c.Steps {
		if other.ID == st.Loop.BackTo {
			from = i + 1
			break
		}
	}
	if from == 0 {
		return false, nil // 검증이 막았어야 한다
	}
	met, err := s.untilMet(ctx, tx, runID, seq, st.Loop.Until)
	if err != nil || met {
		return false, err
	}
	var attempt int
	if err := tx.QueryRow(ctx,
		`SELECT attempt FROM steps WHERE run_id=$1 AND seq=$2`, runID, seq).
		Scan(&attempt); err != nil {
		return false, err
	}
	if attempt+1 >= st.Loop.Max {
		// ★ 소진했다. 그냥 진행한다 ★ — 성패는 success_when 이 정한다(I3).
		s.log().Warn("loop exhausted", "run", runID, "step", st.ID, "attempts", attempt+1)
		return false, nil
	}
	// ★ 구간을 통째로 되돌린다 ★ — SKIPPED 도 되돌려 다음 회차가 다른 경로를
	// 고를 수 있게 하고, ★ 획득 단계는 안 되돌린다 ★ (임대가 살아 있다).
	if _, err := tx.Exec(ctx, `
		UPDATE steps SET state='PENDING', attempt=$4, result=NULL,
		       started_at=NULL, ended_at=NULL, ledger_at=NULL
		 WHERE run_id=$1 AND seq BETWEEN $2 AND $3 AND kind <> 'acquire'`,
		runID, from, seq, attempt+1); err != nil {
		return false, err
	}
	s.log().Info("loop: rolling back the range", "run", runID, "step", st.ID,
		"back_to", st.Loop.BackTo, "attempt", attempt+2, "max", st.Loop.Max)
	return true, nil
}

// validateBack 은 ⑥ 의 되먹임 재시도다 (ADR-013).
//
// ★ 루프를 도는 주체는 Mediator 다 ★ (ADR-014 결정 1) — 루프가 agent enode 와
// build enode 를 오가는데 둘은 서로를 모른다. 여기가 그 결정이 실물이 되는 자리다.
//
// ★ INVARIANTS §2 의 "재실행하지 않는다" 와 부딪히지 않는다 ★ — 그것은
// 크래시 후 재개에 대한 것이다. 이 루프는 계약이 미리 선언한 것이고,
// 반복해도 안전한 단계만 넣을 책임은 계약 저자에게 있다.
func (s *Store) validateBack(ctx context.Context, tx pgx.Tx, runID string, seq int,
	c contract.Contract) (bool, error) {
	validator := c.Steps[seq-1]
	target, targetSeq := contract.Step{}, 0
	for i, st := range c.Steps {
		if st.ValidateWith == validator.ID && i+1 < seq {
			target, targetSeq = st, i+1
		}
	}
	if targetSeq == 0 {
		return false, nil
	}
	passed, err := s.stepPassedTx(ctx, tx, runID, seq)
	if err != nil || passed {
		return false, err
	}
	var attempt int
	if err := tx.QueryRow(ctx,
		`SELECT attempt FROM steps WHERE run_id=$1 AND seq=$2`, runID, targetSeq).
		Scan(&attempt); err != nil {
		return false, err
	}
	max := target.MaxAttempts
	if max <= 0 {
		max = 1
	}
	if attempt+1 >= max {
		// ★ 소진했다 ★ — 성패는 success_when 의 within_attempts 가 판정한다.
		s.log().Warn("attempts exhausted", "run", runID, "step", target.ID, "attempts", attempt+1)
		return false, nil
	}
	// ★ 대상과 검증자를 함께 되돌린다 ★ — 검증자의 회차도 올린다.
	// 산출물 최신성 순서가 (회차, 순번)이라 한 회차 안에서 순번이 맞아야 한다.
	for _, n := range []int{targetSeq, seq} {
		if _, err := tx.Exec(ctx, `
			UPDATE steps SET state='PENDING', attempt=$3, result=NULL,
			       started_at=NULL, ended_at=NULL, ledger_at=NULL
			 WHERE run_id=$1 AND seq=$2`, runID, n, attempt+1); err != nil {
			return false, err
		}
	}
	s.log().Info("validation failed; retrying with feedback", "run", runID, "step", target.ID,
		"validator", validator.ID, "attempt", attempt+2, "max", max)
	return true, nil
}

func (s *Store) stepPassedTx(ctx context.Context, tx pgx.Tx, runID string, seq int) (bool, error) {
	var state string
	var raw []byte
	if err := tx.QueryRow(ctx,
		`SELECT state, result FROM steps WHERE run_id=$1 AND seq=$2`, runID, seq).
		Scan(&state, &raw); err != nil {
		return false, err
	}
	if state != StepDone || len(raw) == 0 {
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

// untilMet 은 그만 돌 조건이 찼는지 본다.
// ★ 판정이 아니라 「되돌릴지」의 물음이다 ★ — 어휘는 success_when 과 같지만
// 결과가 성패로 가지 않는다.
func (s *Store) untilMet(ctx context.Context, tx pgx.Tx, runID string, seq int,
	until contract.Condition) (bool, error) {
	var raw []byte
	if err := tx.QueryRow(ctx,
		`SELECT result FROM steps WHERE run_id=$1 AND seq=$2`, runID, seq).Scan(&raw); err != nil {
		return false, err
	}
	if len(raw) == 0 {
		return false, nil
	}
	var res StepResult
	if err := json.Unmarshal(raw, &res); err != nil {
		return false, err
	}
	if until.ExitCode != nil && (res.ExitCode == nil || *res.ExitCode != *until.ExitCode) {
		return false, nil
	}
	for _, want := range until.Produced {
		found := false
		for _, got := range res.Produced {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			return false, nil
		}
	}
	return true, nil
}
