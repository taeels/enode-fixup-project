package store

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"
)

// Reap 은 만료된 임대를 회수한다 (ADR-008).
//
// ★ 시간이 감시자다 ★ — 감시자를 새로 만들지 않는다.
// Mediator 가 죽어 있는 동안 갱신이 멈추고, 재시작하면 not_after 가 이미 지나
// 여기서 회수된다. ADR-008 이 "주기 스캔 + Mediator 재시작 시 스캔" 이라 한 그대로다.
//
// 이것이 O6("Run 도중 Mediator 를 죽였다 살리면 자원이 풀린다")의 구현이고,
// I2("종료 상태에서 점유 장부가 비어 있다")를 지키는 장치다.
func (s *Store) Reap(ctx context.Context, log *slog.Logger) (int, error) {
	// 만료된 임대를 가진 Run 을 먼저 실패시킨다. 이유를 남긴다 —
	// 왜 안 돌았는지가 없으면 껍데기가 재시도를 못 정한다 (ADR-005).
	reason, _ := json.Marshal(map[string]any{"code": 410, "reason": "임대 만료 — 갱신이 끊겼다"})
	tag, err := s.pool.Exec(ctx, `
		UPDATE runs SET state='FAILED', ended_at=now(), reject=$1
		 WHERE state NOT IN ('SUCCEEDED','FAILED')
		   AND run_id IN (SELECT run_id FROM leases WHERE not_after <= now())`, reason)
	if err != nil {
		return 0, err
	}
	n := int(tag.RowsAffected())

	// 종료한 Run 의 붙잡힌 단계를 정리한다. 보고가 못 들어온 것들이다 —
	// enode 는 임대가 끝나 중단했는데 Mediator 가 죽어 있어 못 알렸을 수 있다.
	if _, err := s.pool.Exec(ctx, `
		UPDATE steps st SET state='FAILED', ended_at=now()
		  FROM runs r
		 WHERE r.run_id = st.run_id AND r.state = 'FAILED'
		   AND st.state IN ('PENDING','CLAIMED')`); err != nil {
		return n, err
	}

	// 종료한 Run 의 임대를 전부 해제한다 — ★ I2 ★
	if _, err := s.pool.Exec(ctx, `
		DELETE FROM leases l USING runs r
		 WHERE r.run_id = l.run_id AND r.state IN ('SUCCEEDED','FAILED')`); err != nil {
		return n, err
	}
	if n > 0 {
		if log != nil {
			log.Warn("임대 만료로 Run 을 회수했다", "runs", n)
		}
		// 회수된 Run 도 봉인한다 — ★ 왜 안 돌았는지가 Record 에 남아야 한다 ★ (ADR-005)
		if err := s.sealExpired(ctx, log); err != nil && log != nil {
			log.Error("회수된 Run 봉인 실패", "err", err)
		}
	}
	return n, nil
}

// sealExpired 는 아직 안 봉인된 종료 Run 을 봉인한다.
// Mediator 가 죽어 있는 동안 끝난 것들이 여기서 잡힌다.
func (s *Store) sealExpired(ctx context.Context, log *slog.Logger) error {
	if s.Records == nil {
		return nil
	}
	rows, err := s.pool.Query(ctx,
		`SELECT run_id, verdict FROM runs WHERE state IN ('SUCCEEDED','FAILED') AND ended_at > now() - interval '1 hour'`)
	if err != nil {
		return err
	}
	type item struct {
		id string
		v  Verdict
	}
	var items []item
	for rows.Next() {
		var id string
		var raw []byte
		if err := rows.Scan(&id, &raw); err != nil {
			rows.Close()
			return err
		}
		var v Verdict
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &v)
		}
		if v.State == "" {
			v = Verdict{State: StateFailed, Checks: []Check{{
				What: "ran", OK: false, Note: "임대 만료 — 갱신이 끊겼다"}}}
		}
		items = append(items, item{id, v})
	}
	rows.Close()
	for _, it := range items {
		if s.Records.Sealed(it.id) {
			continue
		}
		if err := s.sealRecord(ctx, it.id, it.v); err != nil {
			return err
		}
	}
	return nil
}

// RunReaper 는 주기 스캔이다. 시작할 때 한 번 먼저 돈다 (재시작 스캔).
func (s *Store) RunReaper(ctx context.Context, every time.Duration, log *slog.Logger) {
	t := time.NewTimer(0)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		if _, err := s.Reap(ctx, log); err != nil && ctx.Err() == nil {
			log.Error("회수 스캔 실패", "err", err)
		}
		t.Reset(every)
	}
}

// SettleIfDone 은 모든 단계가 끝났으면 Run 을 종료시킨다.
//
// 상태 전이는 INVARIANTS §1.1 그대로다:
//
//	RUNNING → FAILED     ★ 단계가 완주하지 못했다 ★ (못 띄웠거나 중단됐다)
//	                     계약을 대조할 재료가 없으므로 VERIFYING 을 안 거친다
//	RUNNING → VERIFYING  마지막 단계까지 전부 완주했다
//	VERIFYING → SUCCEEDED / FAILED   ⑩ 의 대조 결과
//
// 종료 시 임대를 해제한다 — ★ I2 ★
func (s *Store) SettleIfDone(ctx context.Context, runID string) (string, error) {
	var pending, broke int
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE state IN ('PENDING','CLAIMED')),
		       count(*) FILTER (WHERE state = 'FAILED')
		  FROM steps WHERE run_id = $1`, runID).Scan(&pending, &broke); err != nil {
		return "", err
	}
	if broke == 0 && pending > 0 {
		return "", nil // 아직 진행 중
	}

	var v Verdict
	if broke > 0 {
		// 완주하지 못한 단계가 있다. 재개하지 않는다 —
		// RUNNING → RUNNING 이 멱등이 아니기 때문이다 (INVARIANTS §2).
		v = Verdict{State: StateFailed, Checks: []Check{{
			What: "ran", OK: false, Note: "단계가 완주하지 못했다",
		}}}
	} else {
		run, err := s.GetRun(ctx, runID)
		if err != nil {
			return "", err
		}
		results, err := s.StepResults(ctx, runID)
		if err != nil {
			return "", err
		}
		// ★ VERIFYING — 계약 조건을 대조한다 ★
		if _, err := s.pool.Exec(ctx,
			`UPDATE runs SET state=$2 WHERE run_id=$1 AND state='RUNNING'`,
			runID, StateVerifying); err != nil {
			return "", err
		}
		v = Verify(run.Contract, results)
	}

	verdictJSON, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx,
		`UPDATE runs SET state=$2, verdict=$3, ended_at=now()
		  WHERE run_id=$1 AND state NOT IN ('SUCCEEDED','FAILED')`,
		runID, v.State, verdictJSON); err != nil {
		return "", err
	}
	// ★ I2 — 종료 상태에서 점유 장부가 비어 있다 ★
	if _, err := tx.Exec(ctx, `DELETE FROM leases WHERE run_id = $1`, runID); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	// ★ 종료 상태에 이르렀으므로 봉인한다 ★ (I4) — 이후 변경되지 않는다.
	if err := s.sealRecord(ctx, runID, v); err != nil {
		return v.State, err
	}
	return v.State, nil
}
