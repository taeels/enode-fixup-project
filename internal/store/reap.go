package store

import (
	"context"
	"encoding/json"
	"github.com/taeels/enode/internal/contract"
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
	// ★ 기한이 지난 되묻기를 먼저 정리한다 ★ (ADR-032 §2 — then:"fail").
	// 만료된 질문의 Run 을 정산하면 임대도 함께 풀리므로 아래 스캔과 안 겹친다.
	if runs, err := s.ExpireAsks(ctx); err != nil {
		log.Error("cannot expire asks", "err", err)
	} else {
		for _, runID := range runs {
			state, err := s.SettleIfDone(ctx, runID)
			if err != nil {
				log.Error("cannot settle after ask expiry", "run", runID, "err", err)
				continue
			}
			log.Warn("ask deadline passed; step marked failed", "run", runID, "state", state)
		}
	}

	// 만료된 임대를 가진 Run 을 먼저 실패시킨다. 이유를 남긴다 —
	// 왜 안 돌았는지가 없으면 껍데기가 재시도를 못 정한다 (ADR-005).
	//
	// ★ 사람의 답을 기다리는 Run 은 제외한다 ★ (ADR-047)
	//
	// ADR-032 는 되묻기에 ★ 없으면 무한 대기 ★ 를 못 박았다 —
	// "사람의 시간을 시스템이 짐작하지 않는다". 그런데 임대는 ★ 기계의 시간 ★ 으로
	// 만료된다(갱신 주기 × not_after_factor = 오늘 180초). 둘이 어긋나 있었다:
	//
	//	★ 실측 ★ 승인을 기다리던 Run 이, 노드가 조용해진 지 정확히 180초 만에
	//	         "reclaimed runs with expired leases" 로 FAILED 가 됐다. ★ 사람은 아직 보는 중 ★
	//
	// ★ 왜 노드 쪽 만료를 무시해도 되나 ★ — ASKED 인 동안에는 그 노드에서
	// ★ 아무것도 안 돌고 있다 ★. 임대가 지키는 것은 I1(한 노드에 한 Run)이고,
	// 실행이 없으면 ADR-008 이 이름 붙인 충돌("옛 Run 의 flash 가 아직 돌고 있다")도
	// 없다. 자원을 ★ 예약 ★ 해 둘 뿐이다.
	//
	// ★ 그럼 영원히 묶이지 않나 ★ — 기한은 ADR-032 가 이미 정한 자리에 있다:
	// ask.timeout 이다. 선언했으면 위의 ExpireAsks 가 정리하고, 안 했으면
	// ★ 무한 대기가 계약 저자의 선언 ★ 이다. 시스템이 3분으로 짐작하지 않는다.
	// 사람이 손으로 세우려면 runctl cancel 이 있다 (ADR-009).
	reason, _ := json.Marshal(map[string]any{"code": 410, "reason": "lease expired: renewal stopped"})
	tag, err := s.pool.Exec(ctx, `
		UPDATE runs SET state='FAILED', ended_at=now(), reject=$1
		 WHERE state NOT IN ('SUCCEEDED','FAILED')
		   AND run_id IN (SELECT run_id FROM leases WHERE not_after <= now())
		   AND NOT EXISTS (
		       SELECT 1 FROM steps st
		        WHERE st.run_id = runs.run_id AND st.state = 'ASKED')`, reason)
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
		   AND st.state IN ('PENDING','CLAIMED','ASKED')`); err != nil {
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
			log.Warn("reclaimed runs with expired leases", "runs", n)
		}
		// 회수된 Run 도 봉인한다 — ★ 왜 안 돌았는지가 Record 에 남아야 한다 ★ (ADR-005)
		if err := s.sealExpired(ctx, log); err != nil && log != nil {
			log.Error("cannot seal reclaimed run", "err", err)
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
				What: "ran", OK: false, Note: "lease expired: renewal stopped"}}}
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
			log.Error("reaper scan failed", "err", err)
		}
		t.Reset(every)
	}
}

// Cancel 은 사람이 Run 을 세우는 것이다 (ADR-009).
//
// ★ 새 상태를 만들지 않는다 ★ — * → FAILED 로 간다. 상태가 늘면 크래시 복구
// 검증 대상이 늘어난다는 QUEUED 기각 사유가 그대로 적용된다.
//
// ★ 취소는 회수가 아니다 ★ — 회수는 임대 만료가 한다. 다만 종료 상태이므로
// I2 를 지키려면 장부를 비워야 하고, 그 결과 ★ 다음 하트비트 응답의 임대 목록에서
// 빠지는 것이 곧 enode 에 대한 취소 통보 ★ 가 된다 (ADR-016).
// 그래서 취소가 enode 에 닿는 데 하트비트 주기만큼 지연되며, 그 창은 유계이되 0 이 아니다.
//
// 멱등이다 — 이미 종료됐으면 아무것도 하지 않고 그 상태를 돌려준다.
func (s *Store) Cancel(ctx context.Context, runID, by string) (string, error) {
	run, err := s.GetRun(ctx, runID)
	if err != nil {
		return "", err
	}
	if run.State == StateSucceeded || run.State == StateFailed {
		return run.State, nil
	}
	v := Verdict{State: StateFailed, Checks: []Check{{
		What: "cancelled", OK: false, Note: "cancelled by: " + by,
	}}}
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
		`UPDATE runs SET state='FAILED', verdict=$2, ended_at=now()
		  WHERE run_id=$1 AND state NOT IN ('SUCCEEDED','FAILED')`, runID, verdictJSON); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE steps SET state='FAILED', ended_at=now()
		  WHERE run_id=$1 AND state IN ('PENDING','CLAIMED','ASKED')`, runID); err != nil {
		return "", err
	}
	// ★ I2 ★ 그리고 이 삭제가 곧 enode 에 대한 취소 통보다 (ADR-016)
	if _, err := tx.Exec(ctx, `DELETE FROM leases WHERE run_id=$1`, runID); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	if err := s.sealRecord(ctx, runID, v); err != nil {
		return StateFailed, err
	}
	return StateFailed, nil
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
// hasFleetCondition 은 ★ 함대를 읽을 이유가 있는가 ★ 다 — 없으면 안 읽는다.
// 대부분의 Run 은 함대 조건이 없고, 그때 질의를 하나 아끼는 것보다
// ★ 「안 쓰는 것을 안 읽는다」가 봉인에도 맞다 ★.
func hasFleetCondition(c contract.Contract) bool {
	for _, cond := range c.SuccessWhen {
		if cond.IsFleet() {
			return true
		}
	}
	return false
}

func (s *Store) SettleIfDone(ctx context.Context, runID string) (string, error) {
	var pending, broke int
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE state IN ('PENDING','CLAIMED','ASKED')),
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
			What: "ran", OK: false, Note: "a step did not complete",
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
		// 재시도 소진 여부를 계약과 맞춰 표시한다 — within_attempts 가 이것을 본다.
		for i, st := range run.Contract.Steps {
			if st.MaxAttempts <= 0 {
				continue
			}
			r, ok := results[st.ID]
			if !ok {
				continue
			}
			_ = i
			r.Exhausted = r.Attempt+1 >= st.MaxAttempts
			results[st.ID] = r
		}

		// ★ VERIFYING — 계약 조건을 대조한다 ★
		if _, err := s.pool.Exec(ctx,
			`UPDATE runs SET state=$2 WHERE run_id=$1 AND state='RUNNING'`,
			runID, StateVerifying); err != nil {
			return "", err
		}
		// ★ 판정은 유효 계약을 본다 ★ (ADR-033) — 채택된 success_when 이 여기서
		// 효력을 낸다. 제출본만 보면 사람이 승인한 조건이 대조에서 빠진다.
		live, lerr := s.LiveContract(ctx, runID)
		if lerr != nil {
			return "", lerr
		}
		// ★ 함대를 여기서 한 번 읽는다 ★ (ADR-058)
		//
		// ★ 왜 이 시점인가 ★ — 우리는 수렴 시스템이 아니라 ★ 기록 시스템 ★ 이다
		// (ADR-005 · I4). 쿠버네티스처럼 조건이 맞을 때까지 기다리지 않는다.
		// Run 이 끝나는 그 순간의 함대가 ★ 판정의 대상이고 봉인의 대상 ★ 이다.
		//
		// ★ 못 읽으면 nil 을 넘긴다 ★ — Verify 가 그것을 「관측하지 못했다」로
		// 실패시킨다. ★ 조용히 참이 되지 않는다 ★.
		var fleet []contract.Advert
		if hasFleetCondition(live) {
			if ad, ferr := s.LiveAdverts(ctx); ferr == nil {
				fleet = ad
			} else {
				s.log().Warn("cannot read the fleet for verification; "+
					"fleet conditions will fail", "run", runID, "err", ferr)
			}
		}
		v = Verify(live, results, fleet)
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
