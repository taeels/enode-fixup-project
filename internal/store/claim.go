package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/taeels/enode/internal/contract"
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
	StepID    string            `json:"step_id"`
	RunID     string            `json:"run_id"`
	Seq       int               `json:"seq"`
	Name      string            `json:"name"`
	Uses      string            `json:"uses"`
	Kind      string            `json:"kind"`
	Agent     json.RawMessage   `json:"agent,omitempty"`
	Run       []string          `json:"run,omitempty"`
	Env       []string          `json:"env,omitempty"`     // 통과시킬 환경변수 ★ 이름 ★
	Collect   map[string]string `json:"collect,omitempty"` // 이름 → 워크스페이스 상대경로
	Workspace json.RawMessage   `json:"workspace,omitempty"`
	In        json.RawMessage   `json:"in,omitempty"`
	Out       []string          `json:"out,omitempty"`
	// Schema 는 어댑터가 ★ 프롬프트에 심는 데 ★ 쓴다 (ADR-020).
	// 최종 검증은 Mediator 가 PUT blob 에서 한다 — 강제 지점은 하나다.
	Schema  json.RawMessage `json:"schema,omitempty"`
	Attempt int             `json:"attempt,omitempty"` // 0 부터. 재시도면 1 이상.
	// Requester 는 ★ runctl 로 요청한 사람 ★ 이다 (runs.principal, ADR-015 §1).
	// 지금은 아무도 안 본다 — R2(하네스가 누구 신원으로 도는가)가 쓸 재료다.
	// 미리 싣는 이유는, 나중에 필요해졌을 때 ★ 이 표면을 고치지 않기 위해서 ★ 다.
	Requester string   `json:"requester,omitempty"`
	Feedback  []string `json:"feedback,omitempty"`
	// Ledger 는 ★ 그 시점 원장의 목록 ★ 이다 — 계약이 see.ledger:"list" 라고
	// 했을 때만 실린다 (ADR-023 §6.4). ★ 본문이 아니라 목록이다 ★:
	// enode 가 $IN 에 파일 하나로 깔고, 본문이 필요하면 in.from 이 가져온다.
	Ledger []LedgerEntry `json:"ledger,omitempty"`
	Lease  LeaseRow      `json:"lease"`
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
// ★ needs 가 전부 끝나야 집을 수 있다 ★ — Mediator 가 시퀀서이기 때문이다
// (ADR-014 결정 1). 순서는 계약에 있고 여기서 강제된다.
//
// ★ 배분 정책은 여기 없고, 앞으로도 안 생긴다 ★ (ADR-023 §5) —
// 단계의 node_id 는 CreateRun 이 t=0 에 확정하고 이 질의는 WHERE s.node_id = $1 로
// 자기 몫만 본다. 즉 ★ 당기기가 이미 배분이다 ★. 실행 가능한 단계가 셋인데
// 노드가 둘이어도 고를 일이 없다 — 애초에 각자 자기 것만 보인다.
func (s *Store) ClaimStep(ctx context.Context, nodeID, instance string) (*Claimed, error) {
	// ★ 같은 생이 다시 물으면, 들고 있던 것을 먼저 돌려준다 ★ (ADR-030).
	//
	// 워커는 직렬이다 — claim 은 워커가 한가할 때만 온다. 그리고 완주한 단계의
	// 보고는 ★ 닿을 때까지 다시 보내므로 ★ (enode 쪽 report 재시도), 보고가 밀린
	// 동안에도 워커는 한가해지지 않는다. ⇒ ★ 한가한 워커가 같은 생으로 다시
	// 물었는데 CLAIMED 가 남아 있다면, 그것은 「시작도 못 한 단계」다 ★ —
	// claim 응답이 유실된 것이고, 재전달해도 두 번 실행이 아니다.
	//
	// 생이 다르면 재전달하지 않는다 — 그쪽은 재시작이고, 하트비트가 판정한다
	// (FailRestarted). 생이 비었으면 옛 enode 다 — 오늘 그대로 동작한다.
	if instance != "" {
		if c, err := s.redeliver(ctx, nodeID, instance); err != nil || c != nil {
			return c, err
		}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var c Claimed
	var contractJSON []byte
	err = tx.QueryRow(ctx, `
		SELECT s.run_id, s.seq, s.name, s.uses, s.kind, s.attempt,
		       coalesce(r.contract_versions -> -1, r.contract), r.principal
		  FROM steps s
		  JOIN runs r ON r.run_id = s.run_id
		 WHERE s.node_id = $1
		   AND s.state = 'PENDING'
		   AND r.state = 'RUNNING'
		   AND NOT EXISTS (
		       -- ★ 술어가 여기 하나뿐이고, 그것이 폭이다 ★ (ADR-023 §4).
		       -- 오늘까지는 p.seq < s.seq 였다 — ★ 목록에서의 위치가 의존 ★ 이라
		       -- 한 번에 하나만 돌았다. 이제 ★ 선언된 간선이 의존 ★ 이므로,
		       -- 서로 안 가리키는 단계들은 ★ 동시에 집힌다 ★.
		       -- needs 를 안 적은 계약은 [직전 단계] 로 채워져 들어오므로
		       -- ★ 여기는 한 형태만 안다 ★ (CreateRun 이 정규화한다).
		       --
		       -- ★ SKIPPED 도 끝난 것이다 ★ (ADR-022 §7.2) — dispatch 가 안 간 쪽을
		       -- 여기 남겨두면 뒤 단계가 ★ 영원히 안 집힌다 ★.
		       SELECT 1 FROM steps p
		        WHERE p.run_id = s.run_id AND p.name = ANY(s.needs)
		          AND p.state NOT IN ('DONE','SKIPPED'))
		 ORDER BY s.run_id, s.seq
		   FOR UPDATE OF s SKIP LOCKED
		 LIMIT 1`, nodeID).
		Scan(&c.RunID, &c.Seq, &c.Name, &c.Uses, &c.Kind, &c.Attempt, &contractJSON, &c.Requester)
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

	// ★ 어느 생이 집었는지를 함께 적는다 ★ (ADR-030) — 재전달과 재시작 판정의 재료다.
	if _, err := tx.Exec(ctx,
		`UPDATE steps SET state='CLAIMED', started_at=now(), claimed_instance=$3
		  WHERE run_id=$1 AND seq=$2`,
		c.RunID, c.Seq, nullable(instance)); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	c.StepID = fmt.Sprintf("%s#%02d", c.RunID, c.Seq)
	fillFromContract(&c, contractJSON)
	s.stampLedger(ctx, &c, contractJSON)
	return &c, nil
}

// stampLedger 는 ★ 워터마크를 남기고, 계약이 원하면 목록을 함께 내려보낸다 ★
// (ADR-023 §6.4 자리 2·3).
//
// ★ 트랜잭션 밖에서 한다 ★ — 원장은 파일시스템과 (scope:"work" 면) 다른 행을
// 읽으므로 claim 의 잠금 안에서 부르면 커넥션이 서로를 기다릴 수 있다.
// 집은 직후이고 그 단계는 아직 시작 전이므로 ★ "시작할 때" 와 같은 시점 ★ 이다.
//
// ★ 실패해도 단계를 막지 않는다 ★ — 워터마크는 기록이지 실행 조건이 아니다.
func (s *Store) stampLedger(ctx context.Context, c *Claimed, contractJSON []byte) {
	entries, err := s.Ledger(ctx, c.RunID)
	if err != nil {
		return
	}
	at := make([]string, 0, len(entries))
	for _, e := range entries {
		at = append(at, fmt.Sprintf("%02d.%d-%s", e.Seq, e.Attempt, e.Name))
	}
	if _, err := s.pool.Exec(ctx,
		`UPDATE steps SET ledger_at=$3 WHERE run_id=$1 AND seq=$2`,
		c.RunID, c.Seq, at); err != nil {
		return
	}
	// ★ 심는 것은 계약이 그러라고 할 때뿐이다 ★ — 기본은 안 심는다(오늘 동작).
	var raw struct {
		Steps []struct {
			See *contract.See `json:"see"`
		} `json:"steps"`
	}
	if json.Unmarshal(contractJSON, &raw) != nil || c.Seq-1 >= len(raw.Steps) {
		return
	}
	if see := raw.Steps[c.Seq-1].See; see != nil && see.Ledger == contract.SeeList {
		c.Ledger = entries
	}
}

// redeliver 는 이 노드의 ★ 이번 생 ★ 이 집어놓고 못 받은 단계를 다시 준다 (ADR-030).
//
// ★ 상태를 안 바꾼다 ★ — 이미 CLAIMED 다. 그래서 몇 번을 다시 물어도 같은
// 답이고, 죽은 연결에 전달돼 또 유실되어도 다음 물음이 또 받는다.
// started_at 과 워터마크만 갱신한다 — 실행은 이 전달 뒤에 시작되므로
// "시작할 때 원장에 있던 것" 이라는 뜻(성질 4)이 그대로 산다.
func (s *Store) redeliver(ctx context.Context, nodeID, instance string) (*Claimed, error) {
	var c Claimed
	var contractJSON []byte
	err := s.pool.QueryRow(ctx, `
		SELECT s.run_id, s.seq, s.name, s.uses, s.kind, s.attempt,
		       coalesce(r.contract_versions -> -1, r.contract), r.principal
		  FROM steps s
		  JOIN runs r ON r.run_id = s.run_id
		  JOIN leases l ON l.node_id = s.node_id AND l.run_id = s.run_id
		 WHERE s.node_id = $1 AND s.state = 'CLAIMED'
		   AND s.claimed_instance = $2
		   AND r.state = 'RUNNING' AND l.not_after > now()
		 ORDER BY s.run_id, s.seq LIMIT 1`, nodeID, instance).
		Scan(&c.RunID, &c.Seq, &c.Name, &c.Uses, &c.Kind, &c.Attempt, &contractJSON, &c.Requester)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := s.pool.QueryRow(ctx,
		`SELECT run_id, node_id, not_after, nonce FROM leases WHERE node_id = $1 AND run_id = $2`,
		nodeID, c.RunID).
		Scan(&c.Lease.RunID, &c.Lease.Node, &c.Lease.NotAfter, &c.Lease.Nonce); err != nil {
		return nil, err
	}
	c.Lease.Capability = "agent.reason"
	if _, err := s.pool.Exec(ctx,
		`UPDATE steps SET started_at=now() WHERE run_id=$1 AND seq=$2`,
		c.RunID, c.Seq); err != nil {
		return nil, err
	}
	c.StepID = fmt.Sprintf("%s#%02d", c.RunID, c.Seq)
	fillFromContract(&c, contractJSON)
	s.stampLedger(ctx, &c, contractJSON)
	s.log().Info("재전달 — 같은 생이 다시 물었다", "node", nodeID, "step", c.StepID)
	return &c, nil
}

// FailRestarted 는 ★ 다른 생이 집어둔 단계 ★ 를 실패시킨다 (ADR-030).
//
// 하트비트에 새 생이 실려 오면, 옛 생이 집어둔 CLAIMED 단계는 ★ 어디까지
// 실행됐는지 알 수 없다 ★ — 시작 전이었는지 절반 돌았는지 노드 자신도 모른다
// (재시작으로 기억이 없다). INVARIANTS §2 의 "재실행하지 않는다" 그대로,
// ★ 다시 돌리는 대신 실패시킨다 ★. 보드를 절반 구운 단계를 또 굽지 않는다.
//
// 반환값은 손댄 Run 들이다 — 호출자가 각각을 정산한다(SettleIfDone).
// 이것이 없으면 그 단계는 ★ 영구히 CLAIMED ★ 다: 노드는 살아 하트비트를
// 보내니 임대가 안 죽고, 임대가 살아 있으니 회수도 손대지 않는다.
func (s *Store) FailRestarted(ctx context.Context, nodeID, instance string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		UPDATE steps s SET state='FAILED', ended_at=now(),
		       result = coalesce(result,'{}'::jsonb) || $3::jsonb
		  FROM runs r
		 WHERE r.run_id = s.run_id AND r.state = 'RUNNING'
		   AND s.node_id = $1 AND s.state = 'CLAIMED'
		   AND coalesce(s.claimed_instance,'') NOT IN ('', $2)
		 RETURNING s.run_id`,
		nodeID, instance,
		mustJSON(map[string]string{"error": "노드가 재시작해 진행 중이던 단계를 신뢰할 수 없다"}))
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

// fillFromContract 는 계약의 단계 정의를 응답에 싣는다.
// 계약을 통째로 저장해둔 덕에 별도 컬럼이 필요 없다 (ADR-020 이 스키마를
// 인라인으로 둔 것과 같은 이유 — 흩어놓으면 나중에 못 모은다).
func fillFromContract(c *Claimed, contractJSON []byte) {
	var raw struct {
		Steps []struct {
			ID        string            `json:"id"`
			Agent     json.RawMessage   `json:"agent"`
			Run       []string          `json:"run"`
			Env       []string          `json:"env"`
			Collect   map[string]string `json:"collect"`
			Workspace json.RawMessage   `json:"workspace"`
			In        json.RawMessage   `json:"in"`
			Out       []string          `json:"out"`
			Schema    json.RawMessage   `json:"schema"`
			Feedback  []string          `json:"feedback"`
		} `json:"steps"`
	}
	if json.Unmarshal(contractJSON, &raw) != nil || c.Seq-1 >= len(raw.Steps) {
		return
	}
	st := raw.Steps[c.Seq-1]
	c.Agent, c.Run, c.Workspace, c.In, c.Out = st.Agent, st.Run, st.Workspace, st.In, st.Out
	c.Schema, c.Feedback, c.Env, c.Collect = st.Schema, st.Feedback, st.Env, st.Collect
}

// StepResult 는 enode 가 보고하는 것이다.
type StepResult struct {
	ExitCode *int            `json:"exit_code,omitempty"` // ★ 명령 단계만 ★ (ADR-019)
	Produced []string        `json:"produced,omitempty"`
	Harness  json.RawMessage `json:"harness,omitempty"` // agent 단계만 (ADR-020)
	// Attempt · Exhausted 는 재시도 루프의 결과다 (DB 에서 채운다).
	Attempt   int  `json:"attempt,omitempty"`
	Exhausted bool `json:"-"`
	// Skipped 는 ★ 그 단계가 실행되지 않았다 ★ 는 뜻이다 (dispatch — ADR-022 §7.2).
	// 결과가 없는 것과 ★ 다르다 ★ — 결과가 없으면 크래시일 수 있고,
	// 건너뛴 것은 경로가 갈렸을 뿐이다. Verify 가 둘을 갈라 본다.
	// 와이어로 안 나간다 — 노드가 보고하는 값이 아니라 DB 에서 채우는 것이다.
	Skipped bool `json:"-"`
	// AnsweredBy 는 ★ 누가 답했는가 ★ 다 (ADR-032) — ask 단계에만 채워진다.
	// 조사에서 예외 없는 공통분모가 「답에는 항상 who/when」이었다.
	AnsweredBy string `json:"answered_by,omitempty"`
	// Error 는 ★ 완주하지 못한 ★ 경우다 — 프로세스를 못 띄웠거나 임대가 끝나
	// 중단됐거나. 비어 있으면 완주한 것이고, 종료코드가 무엇이든 DONE 이다.
	Error string `json:"error,omitempty"`
}

// ReportStep 은 단계를 끝낸다.
//
// ★ "완주" 와 "성공" 은 다르다 ★
//
//	DONE    프로세스가 끝나고 결과를 보고했다. ★ 종료코드가 무엇이든 ★.
//	FAILED  아예 못 돌았다 — 프로세스를 못 띄웠거나 임대가 끝나 중단됐다.
//
// exit_code 2 로 끝난 빌드는 ★ 완주한 것 ★ 이고, 그게 성공인지는
// success_when 이 판정한다 (ADR-004 · I3). 여기서 판정하면 계약이 할 일을
// 코드가 가로채는 것이고, 그러면 O4("테스트가 실패했는데 Run 은 성공")가 성립하지 않는다.
// ★ 이 보고를 받은 Mediator 가 다음 단계를 만든다 ★ (ADR-014 결정 1) —
// 다음 claim 이 집을 수 있게 되는 것이 그 형태다.
// ★ 되돌릴지를 효과보다 먼저 본다 ★ (2026-08-21 실측이 찾았다)
//
// 단계는 종료코드가 무엇이든 ★ 완주하면 DONE ★ 이다(ADR-004). 그래서 실패한
// 빌드도 DONE 이고, 그 순간 뒤 단계들의 효과가 적용될 자격을 얻는다. 그런데
// ★ 되돌릴지는 그 뒤에 판단했다 ★ — 그 사이에 갈림길이 닫히고 자원이 잡혔다.
//
//	★ 실측 ★  build 가 1 회차에 exit 1 → DONE → ★ acquire 가 즉시 돌고 ★
//	          그 다음에야 loop 이 구간을 되돌렸다. 획득은 취소되지 않는다.
//
// ★ claim 은 노드가 당기므로 시간차가 있어 대부분 안 겹쳤다 ★ —
// Mediator 가 즉시 수행하는 acquire 가 생기면서 이 경쟁이 ★ 항상 지는 쪽 ★ 이 됐다.
// ⇒ 순서를 뒤집는다: ★ 되돌리면 효과를 아예 적용하지 않는다 ★.
//
// 반환값은 「되돌렸다」이고, 그러면 Run 은 아직 진행 중이므로 정산하지 않는다.
func (s *Store) ReportStep(ctx context.Context, runID string, seq int, nodeID string, ok bool, res StepResult) (bool, error) {
	resJSON, err := json.Marshal(res)
	if err != nil {
		return false, err
	}
	state := StepFailed
	if ok {
		state = StepDone
	}
	// ★ 트랜잭션이다 ★ — 단계를 끝내는 것과 갈림길을 닫는 것이 함께 일어나야 한다.
	// 따로 하면 그 사이에 claim 이 들어와 ★ 안 간 경로가 집힌다 ★.
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	tag, err := tx.Exec(ctx, `
		UPDATE steps SET state=$4, ended_at=now(), result=$5
		 WHERE run_id=$1 AND seq=$2 AND node_id=$3 AND state='CLAIMED'`,
		runID, seq, nodeID, state, resJSON)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, fmt.Errorf("집지 않은 단계를 보고했다 (%s#%d)", runID, seq)
	}
	if ok {
		// ★ 되돌릴지가 먼저다 ★ — 되돌리면 이 단계의 효과는 없던 일이 된다.
		rolled, err := s.rollBack(ctx, tx, runID, seq)
		if err != nil {
			return false, err
		}
		if rolled {
			return true, tx.Commit(ctx)
		}
	}
	var raisedAsks []AskEvent
	if ok {
		// ★ 이름을 못 고르거나 계획이 유효하지 않으면 그 단계가 FAILED 다 ★ —
		// 결과가 나쁜 것이 아니라 계약이 요구한 것을 못 낸 것이므로
		// "완주하지 못함" 과 같은 자리다.
		//
		// ★ 늘리는 것이 고르는 것보다 먼저다 ★ — 계획이 지은 단계가 생긴 뒤라야
		// 분기가 그것을 목적지로 찾을 수 있다.
		asks, err := s.afterStep(ctx, tx, runID, seq)
		if err != nil {
			if _, e := tx.Exec(ctx, `
				UPDATE steps SET state=$3, result = coalesce(result,'{}'::jsonb) || $4::jsonb
				 WHERE run_id=$1 AND seq=$2`,
				runID, seq, StepFailed,
				mustJSON(map[string]string{"error": err.Error()})); e != nil {
				return false, e
			}
			if err := tx.Commit(ctx); err != nil {
				return false, err
			}
			return false, nil // ★ 보고 자체는 받았다 ★ — 노드에 오류를 되던지지 않는다
		}
		raisedAsks = asks
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	// ★ 알림은 커밋 뒤에만 ★ (ADR-032 §4)
	s.PushAsks(raisedAsks)
	return false, nil
}

// applyStepEffects 는 한 단계가 끝나면서 ★ 계약 · 단계 목록 · 점유에 미치는 것 ★ 을
// 적용한다 — 계획을 붙이고(expands), 갈림길을 닫고(dispatch), 자원을 놓는다(release).
//
// 셋 다 「Mediator 가 다음 단계를 만든다」(ADR-014 결정 1)의 일부이고,
// ReportStep 의 ★ 한 트랜잭션 안에서 ★ 일어나야 한다 — 따로 하면 그 사이에
// claim 이 들어와 안 간 경로를 집거나 아직 안 검증된 단계를 집는다.
// afterStep 은 한 단계가 끝난 뒤의 전부다 — 그 단계의 효과를 적용하고,
// 그로 인해 ★ 실행 가능해진 획득 단계 ★ 를 Mediator 가 수행한다.
//
// ★ 획득을 마지막에 두는 이유 ★ — 그것이 needs 를 보고 고르므로, 앞의 효과
// (건너뜀 전파 · 지어진 단계)가 먼저 반영돼 있어야 같은 그림을 본다.
func (s *Store) afterStep(ctx context.Context, tx pgx.Tx, runID string, seq int) ([]AskEvent, error) {
	if err := s.applyStepEffects(ctx, tx, runID, seq); err != nil {
		return nil, err
	}
	if err := s.runAcquires(ctx, tx, runID); err != nil {
		return nil, err
	}
	// ★ 되묻기는 맨 뒤다 ★ — 앞의 효과(전파·획득)가 반영된 그림을 보고
	// needs 가 찬 질문을 올린다 (ADR-032). 올린 것은 ★ 커밋 뒤 알림 ★ 의 재료다.
	return s.raiseAsks(ctx, tx, runID)
}

func (s *Store) applyStepEffects(ctx context.Context, tx pgx.Tx, runID string, seq int) error {
	if err := s.applyExpands(ctx, tx, runID, seq); err != nil {
		return err
	}
	if err := s.applyDispatch(ctx, tx, runID, seq); err != nil {
		return err
	}
	// ★ 놓는 것은 맨 마지막이다 ★ — 되돌릴 수 없으므로, 앞의 둘이 실패해
	// 이 단계가 FAILED 가 되는 경우에는 놓지 않는다.
	return s.applyRelease(ctx, tx, runID, seq)
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

// StepAttempt 는 그 단계가 몇 번째 시도인지다. 산출물 이름에 들어간다.
func (s *Store) StepAttempt(ctx context.Context, runID string, seq int) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx,
		`SELECT attempt FROM steps WHERE run_id=$1 AND seq=$2`, runID, seq).Scan(&n)
	return n, err
}
