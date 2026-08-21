package store

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/taeels/enode/internal/contract"
)

// MaybeLoop 은 구간 반복을 돈다 (ADR-026).
//
// ★ 뒤로 가는 간선이다 ★ — "이 단계가 끝났는데 until 이 불만족이면 back_to 부터
// 다시". 구간 [back_to … 이 단계] 는 ★ 블록을 안 적어도 정해진다 ★:
// seq 가 이미 위상순서이므로 두 끝이 구간을 준다.
//
// ★ 새 실행 기계가 0 개다 ★ — MaybeRetry(validate_with)가 대상과 검증자 ★ 둘 ★ 을
// 되돌리는데, 여기서는 되돌릴 범위가 ★ 구간 ★ 이 된다. 그것이 §7.3 이 말한 일반화다.
//
// ★ I3 를 안 건드린다 ★ — until 은 ★ 되돌릴지 ★ 를 정하지 성패를 안 정한다.
// 소진해도 FAILED 가 아니고 그냥 진행한다. 판정은 success_when 이 한다.
func (s *Store) MaybeLoop(ctx context.Context, runID string, doneSeq int,
	log *slog.Logger) (bool, error) {
	run, err := s.GetRun(ctx, runID)
	if err != nil {
		return false, err
	}
	if doneSeq-1 < 0 || doneSeq-1 >= len(run.Contract.Steps) {
		return false, nil
	}
	st := run.Contract.Steps[doneSeq-1]
	if st.Loop == nil {
		return false, nil
	}
	from := 0
	for i, other := range run.Contract.Steps {
		if other.ID == st.Loop.BackTo {
			from = i + 1
			break
		}
	}
	if from == 0 {
		return false, nil // 검증이 막았어야 한다
	}

	done, err := s.untilMet(ctx, runID, doneSeq, st.Loop.Until)
	if err != nil || done {
		return false, err
	}
	var attempt int
	if err := s.pool.QueryRow(ctx,
		`SELECT attempt FROM steps WHERE run_id=$1 AND seq=$2`, runID, doneSeq).
		Scan(&attempt); err != nil {
		return false, err
	}
	if attempt+1 >= st.Loop.Max {
		// ★ 소진했다. 그냥 진행한다 ★ — 성패는 success_when 이 정한다(ADR-004).
		log.Warn("반복 소진", "run", runID, "step", st.ID, "attempts", attempt+1)
		return false, nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	// ★ 구간을 통째로 되돌린다 ★.
	//
	// ★ SKIPPED 도 되돌린다 ★ — 다음 회차가 ★ 다른 경로를 고를 수 있어야 ★ 하고,
	// 그것이 되먹임의 값이다.
	// ★ 획득 단계는 안 되돌린다 ★ — 이미 잡은 임대가 살아 있고, 다시 잡을 이유가
	// 없다. ⇒ 되돌림의 대상은 ★ 노드가 도는 단계 ★ 뿐이다.
	// ★ 회차를 함께 올린다 ★ — 산출물 최신성 순서가 (회차, 순번)이라
	// 한 회차 안에서 순번이 맞아야 한다 (ADR-021).
	if _, err := tx.Exec(ctx, `
		UPDATE steps SET state='PENDING', attempt=$4, result=NULL,
		       started_at=NULL, ended_at=NULL, ledger_at=NULL
		 WHERE run_id=$1 AND seq BETWEEN $2 AND $3 AND kind <> 'acquire'`,
		runID, from, doneSeq, attempt+1); err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	log.Info("반복 — 구간을 되돌린다", "run", runID, "step", st.ID,
		"back_to", st.Loop.BackTo, "attempt", attempt+2, "max", st.Loop.Max)
	return true, nil
}

// untilMet 은 그만 돌 조건이 찼는지 본다.
// ★ 판정이 아니라 「되돌릴지」의 물음이다 ★ — 어휘는 success_when 과 같지만
// 결과가 성패로 가지 않는다.
func (s *Store) untilMet(ctx context.Context, runID string, seq int,
	until contract.Condition) (bool, error) {
	var raw []byte
	if err := s.pool.QueryRow(ctx,
		`SELECT result FROM steps WHERE run_id=$1 AND seq=$2`, runID, seq).Scan(&raw); err != nil {
		return false, err
	}
	if len(raw) == 0 {
		return false, nil // 결과가 없으면 안 찬 것이다
	}
	var res StepResult
	if err := json.Unmarshal(raw, &res); err != nil {
		return false, err
	}
	if until.ExitCode != nil {
		if res.ExitCode == nil || *res.ExitCode != *until.ExitCode {
			return false, nil
		}
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
