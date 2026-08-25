package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/taeels/enode/internal/contract"
	"io"
	"strings"
)

// LeaseRow 는 enode 에게 내려보내는 허가 아티팩트다 (ADR-010).
type LeaseRow struct {
	RunID      string    `json:"run_id"`
	Node       string    `json:"node"`
	Capability string    `json:"capability"`
	NotAfter   time.Time `json:"not_after"`
	Nonce      string    `json:"nonce"`
}

// RenewLeases 는 그 노드가 든 임대를 갱신하고 전부 돌려준다 (ADR-016).
//
// 이 반환값이 곧 갱신이자 취소 통보다:
//
//	갱신          새 not_after 가 담긴다
//	취소 · 회수   그 임대가 목록에서 빠진다 → enode 가 다음 단계를 시작하지 않는다
//	Mediator 사망 응답 자체가 없다 → not_after 가 지나 enode 가 스스로 멈춘다
//
// 델타가 아니라 전부인 것이 핵심이다 — 목록에 없는 것이 곧 없는 것이라
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
	StepID  string            `json:"step_id"`
	RunID   string            `json:"run_id"`
	Seq     int               `json:"seq"`
	Name    string            `json:"name"`
	Uses    string            `json:"uses"`
	Kind    string            `json:"kind"`
	Agent   json.RawMessage   `json:"agent,omitempty"`
	Run     []string          `json:"run,omitempty"`
	Env     []string          `json:"env,omitempty"`     // 통과시킬 환경변수 이름
	Collect map[string]string `json:"collect,omitempty"` // 이름 → 워크스페이스 상대경로
	// CheckChanged 는 이 단계가 바꿨는지 확인할 경로들이다 (ADR-037).
	// success_when 의 changed 조건에서 Mediator 가 뽑아 싣는다 — 노드는
	// 관찰만 대신하고 판정 조건은 모른다.
	CheckChanged []string        `json:"check_changed,omitempty"`
	Workspace    json.RawMessage `json:"workspace,omitempty"`
	In           json.RawMessage `json:"in,omitempty"`
	Out          []string        `json:"out,omitempty"`
	// Schema 는 어댑터가 프롬프트에 심는 데 쓴다 (ADR-020).
	// 최종 검증은 Mediator 가 PUT blob 에서 한다 — 강제 지점은 하나다.
	Schema json.RawMessage `json:"schema,omitempty"`
	// Roles 는 계획이 uses 에 쓸 수 있는 이름들이다 (ADR-045).
	//
	// 왜 필요한가 — 문법은 "uses 를 적는다" 까지만 말한다. 무엇을 적는지는
	// 그 계약의 requires 에 있고, 계획을 짓는 쪽은 그것을 볼 수 없다.
	// ADR-012 가 적은 문장이 한 층 위에서 되풀이된다 —
	// 읽는 경로가 없으면 계약을 쓰는 쪽이 문자열을 추측한다.
	// 실측에서 밟았다 (colima-enode-1: replan_1 이 uses:"claude" 를 지어냈다).
	//
	// expands 단계에만 싣는다 — 다른 단계는 자기 uses 만 알면 되고,
	// 남의 역할 이름을 아는 것은 그 단계에 쓸 데가 없다.
	Roles []string `json:"roles,omitempty"`
	// RoleAttrs 는 그 역할에 배정된 노드가 무엇을 광고하는가다 (ADR-055).
	//
	// 역할 이름만으로는 명령을 못 짓는다 — 계획은 "mac" 이라는 이름은
	// 알지만 그것이 어떤 기계인지 모른다. 그래서 매 판 uname · sw_vers 를
	// 돌리는 조사 단계를 지었고, 그 답을 보려면 판이 하나 더 들었다.
	//
	// ADR-045(문법) · ADR-049(목표) · ADR-052(이미 선 단계)와 같은 자리다 —
	// 아는 쪽이 적어준다. 매처가 이미 이 값으로 노드를 골랐다.
	RoleAttrs map[string]map[string]string `json:"role_attrs,omitempty"`
	// Owed 는 계약이 약속했는데 아직 안 지어진 단계다 (ADR-049).
	// 이것이 곧 목표다 — success_when 이 이미 그 이름을 가리키고 있다.
	Owed []OwedStep `json:"owed,omitempty"`
	// Standing 은 이미 계약에 서 있는 단계와 그 결말이다 (ADR-052).
	//
	// owed 의 반대쪽이다 — owed 는 「아직 안 지어진 것」을 나르고,
	// 이것은 「이미 지어진 것」을 나른다. 계획은 자기가 어디에 붙는지
	// 모른 채 지어 왔다.
	//
	//	실측 (vm-scratch-3) 재계획이 이미 끝난 조사 단계를 또 짓고,
	//	이미 있는 이름(replan_1 · approve_replan_1)을 다시 썼다.
	//	되먹임으로 조사 결과는 받았는데 그것이 계약의 어디에 있는지는 몰랐다.
	//
	// 결말까지 싣는다 — 실패한 단계를 고치는 것과 없는 단계를 짓는 것은
	// 다른 일이고, 무엇이 실패했는지는 시스템만 안다.
	Standing []StandingStep `json:"standing,omitempty"`
	// Goal 은 이 Run 이 처음 받은 목표다 (ADR-049).
	//
	// 왜 필요한가 — 계획이 지은 재계획 단계의 in.prompt 가 비면
	// 재료를 받고도 무엇을 향해 지을지 모른다. 실측에서 밟았다:
	// 재계획이 "요청 섹션이 비어 있다. 목표는 어디에도 명시돼 있지 않다" 며
	// _cannot 을 냈다. 목표는 v1 의 expands 단계에만 있었고 다음 판으로
	// 전달되는 경로가 없었다. 아는 쪽이 적어준다 (ADR-045 와 같은 자리).
	Goal string `json:"goal,omitempty"`
	// Rejected 는 왜 되돌아왔는가다 (ADR-062).
	//
	// 거절은 분기가 아니라 되돌림이다 — 계획을 지은 단계가 다시 돈다.
	// 그런데 그 단계의 in 은 처음 그대로 라서, 아무것도 안 하면
	// 같은 계획을 다시 짓는다. 실측에서 밟았다(rewind-1): 2 회차가
	// 지적을 그대로 무시하고 첫 계획과 같은 것을 냈다.
	//
	// 앞단이 in.from 을 적을 필요가 없다 — 되돌린 주체가 시스템이므로
	// 이유도 시스템이 안다. goal · owed · standing 과 같은 자리다(ADR-045).
	Rejected []Rejection `json:"rejected,omitempty"`
	// Expands 는 이 단계가 계약을 짓는 단계인가다 (ADR-045).
	//
	// 노드가 알아야 하는 이유 — 어댑터가 프롬프트에 계약 문법을 심는다.
	// 그 전에는 사람이 매 판 Validate() 를 자연어로 번역해 넣었고, 번역이
	// 축약되고 퇴행하고 모순됐다. outContract 를 심는 것과 같은 자리다.
	Expands bool `json:"expands,omitempty"`
	// EnvelopeKey 는 되먹임 봉투의 열쇠다 (ADR-050).
	//
	// 어댑터가 앞 단계의 도구 출력을 프롬프트에 실을 때, 그 출력을 감싸는
	// 구분자에 이 값을 붙인다. 격을 선언하는 것은 봉투의 안내문이고,
	// 열쇠는 그 봉투를 앞 단계가 흉내내지 못하게 한다.
	//
	// 집을 때 뽑는다 — 앞 단계가 산출물을 쓰던 시점에 이 값은 아직
	// 존재하지 않았다. 재전달(ADR-030)이면 행에 남은 것을 그대로 쓴다.
	// 비어 있으면 어댑터가 열쇠 없이 봉투만 씌운다 — 오늘 그대로 동작한다.
	EnvelopeKey string `json:"envelope_key,omitempty"`
	Attempt     int    `json:"attempt,omitempty"` // 0 부터. 재시도면 1 이상.
	// Requester 는 runctl 로 요청한 사람이다 (runs.principal, ADR-015 §1).
	// 지금은 아무도 안 본다 — R2(하네스가 누구 신원으로 도는가)가 쓸 재료다.
	// 미리 싣는 이유는, 나중에 필요해졌을 때 이 표면을 고치지 않기 위해서다.
	Requester string   `json:"requester,omitempty"`
	Feedback  []string `json:"feedback,omitempty"`
	// Ledger 는 그 시점 원장의 목록이다 — 계약이 see.ledger:"list" 라고
	// 했을 때만 실린다 (ADR-023 §6.4). 본문이 아니라 목록이다:
	// enode 가 $IN 에 파일 하나로 깔고, 본문이 필요하면 in.from 이 가져온다.
	Ledger []LedgerEntry `json:"ledger,omitempty"`
	Lease  LeaseRow      `json:"lease"`
}

// Rejection 은 거절 한 번이다 (ADR-062).
//
// 답 본문을 그대로 나른다 — ADR-032 가 「답은 산출물이다」로 정한 대로
// verdict 옆에 note 같은 필드가 함께 있고, 그 글이 곧 이유다. 시스템이
// 요약하지 않는다 — 무엇이 이유인지는 읽는 쪽이 정한다.
type Rejection struct {
	// By 는 답한 사람이다.
	By string `json:"by"`
	// At 은 답한 시각이다.
	At time.Time `json:"at,omitempty"`
	// Answer 는 답 전문이다 (JSON).
	Answer json.RawMessage `json:"answer"`
}

// OwedStep 은 아직 안 지어진 약속 하나다 (ADR-049).
//
// 이름만으로는 부족하다 — 계약이 그 단계에 건 판정이 단계의 종류를
// 정한다. success_when 이 exit_code 로 판정하면 그것은 명령 단계여야 하고
// (ADR-019: agent 단계에 exit_code 를 못 건다), 계획이 agent 로 지으면
// 확장된 계약이 유효하지 않아 통째로 거절된다. 그런데 계획을 짓는 쪽은
// success_when 을 볼 수 없다 — 벽을 보지 못한 채 부딪힌다.
//
// ADR-045 가 역할 어휘에 대해, ADR-049 가 목표에 대해 적은 것과 같은 자리다:
// 아는 쪽이 적어준다. 조건을 구조 그대로 싣고 문장으로 만드는 것은
// 프롬프트를 짓는 쪽(어댑터)이 한다 — 표현이 Mediator 로 새지 않는다.
type OwedStep struct {
	Name string `json:"name"`
	// When 은 계약이 이 이름에 건 판정 조건들이다. 비어 있을 수 있다 —
	// produces 로 약속만 하고 판정은 안 걸 수도 있기 때문이다.
	When []contract.Condition `json:"when,omitempty"`
}

// StandingStep 은 이미 선 단계 하나다 (ADR-052).
type StandingStep struct {
	Name  string `json:"name"`
	State string `json:"state"`
	// ExitCode 는 명령 단계가 끝났을 때만 있다 — 완주와 성공은 다르므로
	// DONE 이면서 0 이 아닐 수 있고, 그 자리가 재계획이 봐야 할 곳이다.
	ExitCode *int `json:"exit_code,omitempty"`
}

var ErrNoWork = errors.New("no work available")

// newEnvelopeKey 는 되먹임 봉투의 구분자에 붙일 열쇠를 뽑는다 (ADR-050).
//
// 48비트면 충분하다 — 막으려는 것은 무차별 대입이 아니라 앞 단계가
// 다음 단계의 구분자를 미리 적어두는 것 이고, 그 단계는 값을 볼 기회가
// 한 번도 없다. 짧게 두는 이유는 프롬프트에 봉투마다 두 번씩 실리기 때문이다.
//
// 못 뽑으면 빈 값을 돌려준다 — 봉투 자체는 열쇠 없이도 격을 선언한다.
// 여기서 실패했다고 단계를 못 돌게 하는 것은 강화 장치가 본체를 막는 것이다.
func newEnvelopeKey() string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return ""
	}
	return hex.EncodeToString(b[:])
}

// ClaimStep 은 이 노드가 할 단계 하나를 집는다.
//
// SELECT … FOR UPDATE SKIP LOCKED 가 배분 그 자체다 (ADR-015 §3) —
// 여러 노드가 동시에 당겨도 한 단계는 정확히 한 노드에만 간다.
// 손으로 짤 잠금이 없다.
//
// ※ 할당(CreateRun)의 기본키 충돌 + 롤백과는 다른 기계다.
//
//	저쪽은 여러 자원을 한꺼번에 잡거나 전부 포기하는 것이고,
//	이쪽은 대기열에서 하나를 집는 것이다.
//
// needs 가 전부 끝나야 집을 수 있다 — Mediator 가 시퀀서이기 때문이다
// (ADR-014 결정 1). 순서는 계약에 있고 여기서 강제된다.
//
// 배분 정책은 여기 없고, 앞으로도 안 생긴다 (ADR-023 §5) —
// 단계의 node_id 는 CreateRun 이 t=0 에 확정하고 이 질의는 WHERE s.node_id = $1 로
// 자기 몫만 본다. 즉 당기기가 이미 배분이다. 실행 가능한 단계가 셋인데
// 노드가 둘이어도 고를 일이 없다 — 애초에 각자 자기 것만 보인다.
func (s *Store) ClaimStep(ctx context.Context, nodeID, instance string) (*Claimed, error) {
	// 같은 생이 다시 물으면, 들고 있던 것을 먼저 돌려준다 (ADR-030).
	//
	// 워커는 직렬이다 — claim 은 워커가 한가할 때만 온다. 그리고 완주한 단계의
	// 보고는 닿을 때까지 다시 보내므로 (enode 쪽 report 재시도), 보고가 밀린
	// 동안에도 워커는 한가해지지 않는다. ⇒ 한가한 워커가 같은 생으로 다시
	// 물었는데 CLAIMED 가 남아 있다면, 그것은 「시작도 못 한 단계」다 —
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
		       -- 술어가 여기 하나뿐이고, 그것이 폭이다 (ADR-023 §4).
		       -- 오늘까지는 p.seq < s.seq 였다 — 목록에서의 위치가 의존이라
		       -- 한 번에 하나만 돌았다. 이제 선언된 간선이 의존 이므로,
		       -- 서로 안 가리키는 단계들은 동시에 집힌다.
		       -- needs 를 안 적은 계약은 [직전 단계] 로 채워져 들어오므로
		       -- 여기는 한 형태만 안다 (CreateRun 이 정규화한다).
		       --
		       -- SKIPPED 도 끝난 것이다 (ADR-022 §7.2) — dispatch 가 안 간 쪽을
		       -- 여기 남겨두면 뒤 단계가 영원히 안 집힌다.
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
		return nil, fmt.Errorf("step is assigned but has no lease (%s#%d): %w", c.RunID, c.Seq, err)
	}
	c.Lease.Capability = "agent.reason"

	// 어느 생이 집었는지를 함께 적는다 (ADR-030) — 재전달과 재시작 판정의 재료다.
	//
	// 봉투의 열쇠도 여기서 뽑는다 (ADR-050) — 집는 순간이 곧 「앞 단계가
	// 더는 손댈 수 없게 된 시점」이다. 재시도면 여기를 다시 지나므로
	// 회차마다 새 열쇠가 된다: 앞 회차의 자백을 되먹일 때, 그것을 쓴
	// 것이 자기 자신이어도 그때 본 열쇠는 이미 쓸모가 없다.
	c.EnvelopeKey = newEnvelopeKey()
	if _, err := tx.Exec(ctx,
		`UPDATE steps SET state='CLAIMED', started_at=now(), claimed_instance=$3,
		        envelope_key=$4
		  WHERE run_id=$1 AND seq=$2`,
		c.RunID, c.Seq, nullable(instance), nullable(c.EnvelopeKey)); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	c.StepID = fmt.Sprintf("%s#%02d", c.RunID, c.Seq)
	fillFromContract(&c, contractJSON)
	s.stampStanding(ctx, &c)
	s.stampRejections(ctx, &c)
	s.stampRoleAttrs(ctx, &c)
	s.stampLedger(ctx, &c, contractJSON)
	return &c, nil
}

// stampRoleAttrs 는 각 역할이 어느 기계에 앉았는지를 싣는다 (ADR-055).
//
// 계획을 짓는 단계에만 — 다른 단계는 자기 기계에서 돌므로 그것을
// 알 필요가 없고, 남의 기계 사정은 그 단계에 쓸 데가 없다.
//
// 배정은 t=0 에 확정돼 있다 (ADR-015 §3) — steps.node_id 가 이미 있고
// nodes 행에 그 노드의 마지막 광고가 있다. 새로 물어보지 않는다.
func (s *Store) stampRoleAttrs(ctx context.Context, c *Claimed) {
	if !c.Expands || len(c.Roles) == 0 {
		return
	}
	// 원천은 runs.assigned 다 — steps 가 아니다.
	//
	// 처음에는 steps 를 돌았고, 그것이 이 결정을 무력화했다:
	// requires 에 선언됐지만 아직 단계가 없는 역할은 steps 에 행이 없다.
	// 그런데 계획이 명령을 지어 붙일 대상이 바로 그 역할들이다 —
	// ADR-049 가 권하는 형태(requires 로 함대를 선언하고 produces 로 약속만
	// 걸어 계획이 단계를 짓는다)에서는 그것이 기본이다.
	// ⇒ 이름만 실린 채 나가고 계획은 다시 조사 단계를 지어 판을 하나 먹는다 —
	//   이 결정이 없애려던 바로 그 동작이다.
	var assignedJSON []byte
	if err := s.pool.QueryRow(ctx,
		`SELECT assigned FROM runs WHERE run_id = $1`, c.RunID).Scan(&assignedJSON); err != nil {
		return
	}
	var assigned []Assigned
	if len(assignedJSON) == 0 || json.Unmarshal(assignedJSON, &assigned) != nil {
		return
	}
	ids := map[string][]string{}
	for _, a := range assigned {
		for _, n := range a.Nodes {
			ids[a.As] = append(ids[a.As], n.Node)
		}
	}
	if len(ids) == 0 {
		return
	}
	// 노드의 마지막 광고를 읽는다 — 배정은 t=0 에 확정돼 있지만(node_id)
	// 속성은 광고마다 통째로 덮인다 (UpsertAdvert). 그래서 여기서 읽는 것은
	// 「배정 시점의 값」이 아니라 「지금 값」이고, 둘이 다를 수 있다.
	// 지금 값이 맞는 값이다 — 계획은 지금 명령을 짓는다. 다만 그래서
	// 같은 단계를 재전달해도 프롬프트가 달라질 수 있다 (I4 가 재현을
	// 보장하는 것은 봉인된 기록이지 프롬프트의 불변이 아니다).
	all := map[string][]string{}
	for role, nodes := range ids {
		all[role] = nodes
	}
	out := map[string]map[string]string{}
	for role, nodes := range all {
		merged := map[string]string{}
		for _, id := range nodes {
			var capsJSON []byte
			if err := s.pool.QueryRow(ctx,
				`SELECT capabilities FROM nodes WHERE node_id = $1`, id).Scan(&capsJSON); err != nil {
				continue
			}
			var caps []contract.Capability
			if json.Unmarshal(capsJSON, &caps) != nil {
				continue
			}
			// 능력이 여럿이면 속성을 합친다 — 오늘 어휘는 하나뿐이라
			// 실제로는 한 벌이고, 늘어도 이 자리가 안 바뀐다.
			for _, cp := range caps {
				for k, v := range cp.Attrs {
					merged[k] = v
				}
			}
		}
		if len(merged) > 0 {
			out[role] = merged
		}
	}
	if len(out) > 0 {
		c.RoleAttrs = out
	}
}

// stampStanding 은 이미 선 단계와 그 결말을 싣는다 (ADR-052).
//
// 계획을 짓는 단계에만 — 다른 단계는 남의 상태를 알 필요가 없고,
// 알면 그것으로 자기 판정을 흉내낼 여지만 생긴다 (ADR-037).
//
// 트랜잭션 밖이다 — stampLedger 와 같은 이유이고, 집은 직후라
// 「시작할 때의 그림」과 같은 시점이다. 실패해도 단계를 막지 않는다.
func (s *Store) stampStanding(ctx context.Context, c *Claimed) {
	if !c.Expands {
		return
	}
	rows, err := s.pool.Query(ctx,
		`SELECT name, state, result->>'exit_code'
		   FROM steps WHERE run_id=$1 ORDER BY seq`, c.RunID)
	if err != nil {
		return
	}
	defer rows.Close()
	var out []StandingStep
	for rows.Next() {
		var st StandingStep
		var code *string
		if err := rows.Scan(&st.Name, &st.State, &code); err != nil {
			return
		}
		if code != nil {
			if n, err := strconv.Atoi(*code); err == nil {
				st.ExitCode = &n
			}
		}
		out = append(out, st)
	}
	if rows.Err() == nil {
		c.Standing = out
	}
}

// stampLedger 는 워터마크를 남기고, 계약이 원하면 목록을 함께 내려보낸다
// (ADR-023 §6.4 자리 2·3).
//
// 트랜잭션 밖에서 한다 — 원장은 파일시스템과 (scope:"work" 면) 다른 행을
// 읽으므로 claim 의 잠금 안에서 부르면 커넥션이 서로를 기다릴 수 있다.
// 집은 직후이고 그 단계는 아직 시작 전이므로 "시작할 때" 와 같은 시점이다.
//
// 실패해도 단계를 막지 않는다 — 워터마크는 기록이지 실행 조건이 아니다.
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
	// 심는 것은 계약이 그러라고 할 때뿐이다 — 기본은 안 심는다(오늘 동작).
	var raw struct {
		Steps []struct {
			See *contract.See `json:"see"`
		} `json:"steps"`
	}
	if json.Unmarshal(contractJSON, &raw) != nil || c.Seq-1 >= len(raw.Steps) {
		return
	}
	// 계획을 짓는 단계에는 기본으로 켠다 (ADR-057)
	//
	// ADR-023 §6.4 는 "없으면 오늘 그대로" 로 열어두고 계약이 요구할 때만
	// 실었다. 그런데 여섯 판 동안 see.ledger 를 쓴 계약이 0 건이다 —
	// 문법이 그 기계의 존재를 안 가르쳤기 때문이다.
	//
	// 계획을 짓는 쪽은 「무엇이 이미 나와 있는지」를 항상 알아야 한다.
	// goal · owed · standing 을 기본으로 싣는 것과 같은 자리다: 모르면
	// 이름을 지어내거나 틀린 자리에 적는다 (vm-scratch-6 이 그랬다).
	//
	// 목록이지 본문이 아니다 — 크기 걱정(§6.3)은 그대로 지킨다.
	// 다른 단계는 계약이 요구할 때만 받는다(오늘 그대로).
	see := raw.Steps[c.Seq-1].See
	if c.Expands || (see != nil && see.Ledger == contract.SeeList) {
		c.Ledger = entries
	}
}

// redeliver 는 이 노드의 이번 생이 집어놓고 못 받은 단계를 다시 준다 (ADR-030).
//
// 상태를 안 바꾼다 — 이미 CLAIMED 다. 그래서 몇 번을 다시 물어도 같은
// 답이고, 죽은 연결에 전달돼 또 유실되어도 다음 물음이 또 받는다.
// started_at 과 워터마크만 갱신한다 — 실행은 이 전달 뒤에 시작되므로
// "시작할 때 원장에 있던 것" 이라는 뜻(성질 4)이 그대로 산다.
func (s *Store) redeliver(ctx context.Context, nodeID, instance string) (*Claimed, error) {
	var c Claimed
	var contractJSON []byte
	err := s.pool.QueryRow(ctx, `
		SELECT s.run_id, s.seq, s.name, s.uses, s.kind, s.attempt,
		       coalesce(r.contract_versions -> -1, r.contract), r.principal,
		       coalesce(s.envelope_key, '')
		  FROM steps s
		  JOIN runs r ON r.run_id = s.run_id
		  JOIN leases l ON l.node_id = s.node_id AND l.run_id = s.run_id
		 WHERE s.node_id = $1 AND s.state = 'CLAIMED'
		   AND s.claimed_instance = $2
		   AND r.state = 'RUNNING' AND l.not_after > now()
		 ORDER BY s.run_id, s.seq LIMIT 1`, nodeID, instance).
		Scan(&c.RunID, &c.Seq, &c.Name, &c.Uses, &c.Kind, &c.Attempt, &contractJSON, &c.Requester,
			&c.EnvelopeKey)
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
	s.stampStanding(ctx, &c)
	// 재전달도 같은 프롬프트를 만들어야 한다 (ADR-030) — 하나라도 빠지면
	// 같은 단계가 다른 프롬프트를 받는다.
	s.stampRejections(ctx, &c)
	s.stampRoleAttrs(ctx, &c)
	s.stampLedger(ctx, &c, contractJSON)
	s.log().Info("redelivering to the same instance", "node", nodeID, "step", c.StepID)
	return &c, nil
}

// FailRestarted 는 다른 생이 집어둔 단계를 실패시킨다 (ADR-030).
//
// 하트비트에 새 생이 실려 오면, 옛 생이 집어둔 CLAIMED 단계는 어디까지
// 실행됐는지 알 수 없다 — 시작 전이었는지 절반 돌았는지 노드 자신도 모른다
// (재시작으로 기억이 없다). INVARIANTS §2 의 "재실행하지 않는다" 그대로,
// 다시 돌리는 대신 실패시킨다. 보드를 절반 구운 단계를 또 굽지 않는다.
//
// 반환값은 손댄 Run 들이다 — 호출자가 각각을 정산한다(SettleIfDone).
// 이것이 없으면 그 단계는 영구히 CLAIMED 다: 노드는 살아 하트비트를
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
		mustJSON(map[string]string{"error": "node restarted; in-flight step cannot be trusted"}))
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
			Expands   bool              `json:"expands"`
			Produces  []string          `json:"produces"`
		} `json:"steps"`
		// 판정 조건에서 「확인할 경로」만 뽑아 싣는다 (ADR-037).
		SuccessWhen []struct {
			Step    string   `json:"step"`
			Changed []string `json:"changed"`
		} `json:"success_when"`
		Requires []struct {
			As string `json:"as"`
		} `json:"requires"`
	}
	if json.Unmarshal(contractJSON, &raw) != nil || c.Seq-1 >= len(raw.Steps) {
		return
	}
	st := raw.Steps[c.Seq-1]
	c.Agent, c.Run, c.Workspace, c.In, c.Out = st.Agent, st.Run, st.Workspace, st.In, st.Out
	c.Schema, c.Feedback, c.Env, c.Collect = st.Schema, st.Feedback, st.Env, st.Collect
	c.Expands = st.Expands
	// 계획을 짓는 단계에만 어휘를 실어준다 (ADR-045).
	// acquire 로 실행 중에 생기는 역할도 이 계약의 역할이다 (ADR-022 §7.5).
	if c.Expands {
		for _, r := range raw.Requires {
			if r.As != "" {
				c.Roles = append(c.Roles, r.As)
			}
		}
		// 약속했는데 아직 없는 이름 — 이것이 이 판이 향할 곳이다 (ADR-049)
		have := map[string]bool{}
		for _, x := range raw.Steps {
			have[x.ID] = true
		}
		for _, x := range raw.Steps {
			for _, n := range x.Produces {
				if have[n] {
					continue
				}
				// 그 이름에 걸린 판정을 함께 싣는다 — 단계의 종류가
				// 거기서 정해진다. 계획은 success_when 을 볼 수 없다.
				c.Owed = append(c.Owed, OwedStep{Name: n, When: condsFor(contractJSON, n)})
			}
		}
		// 처음 받은 목표를 나른다 — v1 의 첫 expands 단계가 받은 프롬프트다.
		// 그 단계 자신이면 자기 프롬프트가 이미 in 에 있으므로 안 싣는다.
		for _, x := range raw.Steps {
			if x.Expands && x.ID != st.ID {
				var in struct {
					Prompt string `json:"prompt"`
				}
				if json.Unmarshal(x.In, &in) == nil && in.Prompt != "" {
					c.Goal = in.Prompt
					break
				}
			}
		}
	}
	// 확인할 경로를 실어 보낸다 (ADR-037) — out 이 「무엇을 낼 것인가」를
	// 싣는 것과 같은 자리다. 노드는 판정 조건을 모른다 — 관찰만 대신하고
	// 대조는 Verify 가 한다 (ADR-005 조립자=평가자).
	// 계약 저자는 success_when 에만 적는다 — 두 곳에 안 적는다.
	seen := map[string]bool{}
	for _, cond := range raw.SuccessWhen {
		if cond.Step != st.ID {
			continue
		}
		for _, p := range cond.Changed {
			if !seen[p] {
				seen[p] = true
				c.CheckChanged = append(c.CheckChanged, p)
			}
		}
	}
}

// condsFor 는 그 단계 이름에 걸린 success_when 조건들을 원형 그대로 뽑는다
// (ADR-049 보강). 문장으로 만들지 않는다 — 표현은 프롬프트를 짓는 쪽의 몫이다.
func condsFor(contractJSON []byte, name string) []contract.Condition {
	var raw struct {
		SuccessWhen []contract.Condition `json:"success_when"`
	}
	if json.Unmarshal(contractJSON, &raw) != nil {
		return nil
	}
	var out []contract.Condition
	for _, cond := range raw.SuccessWhen {
		if cond.Step == name {
			out = append(out, cond)
		}
	}
	return out
}

// StepResult 는 enode 가 보고하는 것이다.
type StepResult struct {
	ExitCode *int            `json:"exit_code,omitempty"` // 명령 단계만 (ADR-019)
	Produced []string        `json:"produced,omitempty"`
	Harness  json.RawMessage `json:"harness,omitempty"` // agent 단계만 (ADR-020)
	// Workspace 는 어떤 상태의 워크스페이스에서 돌았는가다 (ADR-036).
	// "clean" 은 되돌린 자리, "unprepared" 는 되돌리지 않은 자리다 —
	// 저장소가 없어 reset·clean 을 할 수 없는 워크스페이스가 그렇다.
	// 판정에는 안 쓴다 (I3 — 기계적 조건만). 재현 실패의 원인을 찾기 위한 기록이다.
	Workspace string `json:"workspace,omitempty"`
	// Changed 는 실제로 바뀐 것으로 관측된 경로다 (ADR-037).
	// 노드가 확인해 보고하고 Verify 가 대조한다 — produced 와 같은 모양이다.
	Changed []string `json:"changed,omitempty"`
	// Attempt · Exhausted 는 재시도 루프의 결과다 (DB 에서 채운다).
	Attempt   int  `json:"attempt,omitempty"`
	Exhausted bool `json:"-"`
	// Skipped 는 그 단계가 실행되지 않았다는 뜻이다 (dispatch — ADR-022 §7.2).
	// 결과가 없는 것과 다르다 — 결과가 없으면 크래시일 수 있고,
	// 건너뛴 것은 경로가 갈렸을 뿐이다. Verify 가 둘을 갈라 본다.
	// 와이어로 안 나간다 — 노드가 보고하는 값이 아니라 DB 에서 채우는 것이다.
	Skipped bool `json:"-"`
	// Chosen 은 갈림길이 이 단계를 골랐던 적이 있는가다 (ADR-060 §3).
	// SKIPPED 와 함께 서면 골랐는데 못 닿았다는 뜻이고, 그것은 경로가
	// 갈린 것이 아니라 목표 미달이다. 와이어로 안 나간다 — DB 에서 채운다.
	Chosen bool `json:"-"`
	// AnsweredBy 는 누가 답했는가다 (ADR-032) — ask 단계에만 채워진다.
	// 조사에서 예외 없는 공통분모가 「답에는 항상 who/when」이었다.
	AnsweredBy string `json:"answered_by,omitempty"`
	// Error 는 완주하지 못한 경우다 — 프로세스를 못 띄웠거나 임대가 끝나
	// 중단됐거나. 비어 있으면 완주한 것이고, 종료코드가 무엇이든 DONE 이다.
	Error string `json:"error,omitempty"`
}

// ReportStep 은 단계를 끝낸다.
//
// "완주" 와 "성공" 은 다르다
//
//	DONE    프로세스가 끝나고 결과를 보고했다. 종료코드가 무엇이든.
//	FAILED  아예 못 돌았다 — 프로세스를 못 띄웠거나 임대가 끝나 중단됐다.
//
// exit_code 2 로 끝난 빌드는 완주한 것이고, 그게 성공인지는
// success_when 이 판정한다 (ADR-004 · I3). 여기서 판정하면 계약이 할 일을
// 코드가 가로채는 것이고, 그러면 O4("테스트가 실패했는데 Run 은 성공")가 성립하지 않는다.
// 이 보고를 받은 Mediator 가 다음 단계를 만든다 (ADR-014 결정 1) —
// 다음 claim 이 집을 수 있게 되는 것이 그 형태다.
// 되돌릴지를 효과보다 먼저 본다 (2026-08-21 실측이 찾았다)
//
// 단계는 종료코드가 무엇이든 완주하면 DONE 이다(ADR-004). 그래서 실패한
// 빌드도 DONE 이고, 그 순간 뒤 단계들의 효과가 적용될 자격을 얻는다. 그런데
// 되돌릴지는 그 뒤에 판단했다 — 그 사이에 갈림길이 닫히고 자원이 잡혔다.
//
//	실측        build 가 1 회차에 exit 1 → DONE → acquire 가 즉시 돌고
//	          그 다음에야 loop 이 구간을 되돌렸다. 획득은 취소되지 않는다.
//
// claim 은 노드가 당기므로 시간차가 있어 대부분 안 겹쳤다 —
// Mediator 가 즉시 수행하는 acquire 가 생기면서 이 경쟁이 항상 지는 쪽이 됐다.
// ⇒ 순서를 뒤집는다: 되돌리면 효과를 아예 적용하지 않는다.
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
	// 트랜잭션이다 — 단계를 끝내는 것과 갈림길을 닫는 것이 함께 일어나야 한다.
	// 따로 하면 그 사이에 claim 이 들어와 안 간 경로가 집힌다.
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
		return false, fmt.Errorf("reported a step that was not claimed (%s#%d)", runID, seq)
	}
	if ok {
		// 되돌릴지가 먼저다 — 되돌리면 이 단계의 효과는 없던 일이 된다.
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
		// 이름을 못 고르거나 계획이 유효하지 않으면 그 단계가 FAILED 다 —
		// 결과가 나쁜 것이 아니라 계약이 요구한 것을 못 낸 것이므로
		// "완주하지 못함" 과 같은 자리다.
		//
		// 늘리는 것이 고르는 것보다 먼저다 — 계획이 지은 단계가 생긴 뒤라야
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
			return false, nil // 보고 자체는 받았다 — 노드에 오류를 되던지지 않는다
		}
		raisedAsks = asks
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	// 알림은 커밋 뒤에만 (ADR-032 §4)
	s.PushAsks(raisedAsks)
	return false, nil
}

// applyStepEffects 는 한 단계가 끝나면서 계약 · 단계 목록 · 점유에 미치는 것을
// 적용한다 — 계획을 붙이고(expands), 갈림길을 닫고(dispatch), 자원을 놓는다(release).
//
// 셋 다 「Mediator 가 다음 단계를 만든다」(ADR-014 결정 1)의 일부이고,
// ReportStep 의 한 트랜잭션 안에서 일어나야 한다 — 따로 하면 그 사이에
// claim 이 들어와 안 간 경로를 집거나 아직 안 검증된 단계를 집는다.
// afterStep 은 한 단계가 끝난 뒤의 전부다 — 그 단계의 효과를 적용하고,
// 그로 인해 실행 가능해진 획득 단계를 Mediator 가 수행한다.
//
// 획득을 마지막에 두는 이유 — 그것이 needs 를 보고 고르므로, 앞의 효과
// (건너뜀 전파 · 지어진 단계)가 먼저 반영돼 있어야 같은 그림을 본다.
func (s *Store) afterStep(ctx context.Context, tx pgx.Tx, runID string, seq int) ([]AskEvent, error) {
	if err := s.applyStepEffects(ctx, tx, runID, seq); err != nil {
		return nil, err
	}
	if err := s.runAcquires(ctx, tx, runID); err != nil {
		return nil, err
	}
	// 되묻기는 맨 뒤다 — 앞의 효과(전파·획득)가 반영된 그림을 보고
	// needs 가 찬 질문을 올린다 (ADR-032). 올린 것은 커밋 뒤 알림의 재료다.
	return s.raiseAsks(ctx, tx, runID)
}

func (s *Store) applyStepEffects(ctx context.Context, tx pgx.Tx, runID string, seq int) error {
	if err := s.applyExpands(ctx, tx, runID, seq); err != nil {
		return err
	}
	if err := s.applyDispatch(ctx, tx, runID, seq); err != nil {
		return err
	}
	// 놓는 것은 맨 마지막이다 — 되돌릴 수 없으므로, 앞의 둘이 실패해
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

// stampRejections 는 왜 되돌아왔는가를 싣는다 (ADR-062).
//
// 계획을 짓는 단계에만 — 되돌림은 그 단계로 오고, 다른 단계는 남의 거절을
// 알 필요가 없다. stampStanding 과 같은 규칙이다.
//
// 트랜잭션 밖이다 — 실패해도 단계를 막지 않는다. 이유가 없으면 계획이
// 처음처럼 짓는데, 그것이 오늘 동작이므로 없으면 오늘 그대로다.
//
// 판의 열에서 읽는다 — retirePlan 이 붙인 판(by: answer:…)이 evidence 로
// 사람의 답 blob을 가리킨다. 그 판이 곧 "여기서 물러났다" 는 기록이고,
// 답 본문이 곧 이유다. 별도 장부를 안 만든다.
func (s *Store) stampRejections(ctx context.Context, c *Claimed) {
	if !c.Expands || s.Records == nil {
		return
	}
	var raw []byte
	if err := s.pool.QueryRow(ctx,
		`SELECT contract_versions FROM runs WHERE run_id=$1`, c.RunID).Scan(&raw); err != nil {
		return
	}
	var vers []ContractVersion
	if len(raw) == 0 || json.Unmarshal(raw, &vers) != nil {
		return
	}
	var out []Rejection
	for i, v := range vers {
		// 물린 판은 답으로 붙었고 계획 blob 을 cause 로 든다 (retirePlan).
		if !strings.HasPrefix(v.By, "answer:") || len(v.Cause) == 0 || v.Evidence == "" {
			continue
		}
		// 이 단계가 지은 계획이 물린 것만 — 계약에 expands 가 여럿일 수 있다.
		if i == 0 || vers[i-1].By != byStep(c.Name) {
			continue
		}
		body, err := s.readRecordBlob(c.RunID, v.Evidence)
		if err != nil {
			continue
		}
		out = append(out, Rejection{At: v.At, Answer: body})
	}
	c.Rejected = out
}

// readRecordBlob 은 Record 안의 경로("blobs/02.0-approval")를 읽는다.
func (s *Store) readRecordBlob(runID, path string) ([]byte, error) {
	name := path
	if i := strings.LastIndexByte(name, '/'); i >= 0 {
		name = name[i+1:]
	}
	// "02.0-approval" → "approval"
	if i := strings.IndexByte(name, '-'); i >= 0 {
		name = name[i+1:]
	}
	rc, _, err := s.Records.OpenBlob(runID, name)
	if err != nil {
		return nil, err
	}
	defer rc.Close() //nolint:errcheck
	return io.ReadAll(rc)
}
