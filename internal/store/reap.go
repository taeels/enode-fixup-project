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
	if n > 0 && log != nil {
		log.Warn("임대 만료로 Run 을 회수했다", "runs", n)
	}
	return n, nil
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
// S4 는 여기까지만 한다 — 계약 조건 대조(⑩)와 Record 봉인(⑪)은 S5·S6 이다.
// 지금은 ★ 단계가 하나라도 실패하면 FAILED, 전부 DONE 이면 SUCCEEDED ★ 로 두고,
// 종료 시 임대를 해제해 I2 를 지킨다.
func (s *Store) SettleIfDone(ctx context.Context, runID string) (string, error) {
	var pending, failed int
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE state IN ('PENDING','CLAIMED')),
		       count(*) FILTER (WHERE state = 'FAILED')
		  FROM steps WHERE run_id = $1`, runID).Scan(&pending, &failed); err != nil {
		return "", err
	}
	switch {
	case failed > 0:
		// 실패한 단계가 있으면 남은 단계를 돌릴 이유가 없다.
		// RUNNING → RUNNING 이 멱등이 아니므로 재개하지 않는다 (INVARIANTS §2).
	case pending > 0:
		return "", nil // 아직 진행 중
	}
	state := StateSucceeded
	if failed > 0 {
		state = StateFailed
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx,
		`UPDATE runs SET state=$2, ended_at=now() WHERE run_id=$1 AND state NOT IN ('SUCCEEDED','FAILED')`,
		runID, state); err != nil {
		return "", err
	}
	// ★ I2 — 종료 상태에서 점유 장부가 비어 있다 ★
	if _, err := tx.Exec(ctx, `DELETE FROM leases WHERE run_id = $1`, runID); err != nil {
		return "", err
	}
	return state, tx.Commit(ctx)
}
