package store

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/taeels/enode/internal/contract"
)

// plan 은 expands 단계가 내는 산출물의 형태다 (ADR-022 §6.3 갈래 A).
//
// ★ 배열이 아니라 객체다 ★ — dispatch.from 이 "route.next" 로 객체 경로를 쓰는
// 것과 같은 결이고, 계획에 다른 것(근거·비용)을 붙일 자리가 남는다.
type plan struct {
	Steps []contract.Step `json:"steps"`
	// SuccessWhen 은 ★ 제안 ★ 이다 (ADR-033) — 효력이 없다.
	//
	// 판정 기준의 저자는 기계(계획)일 수 있으나, 그것이 효력을 얻는 유일한
	// 길은 ★ 목표를 준 사람의 답 ★ 이다 — 이 expands 단계를 adopts 로 지목한
	// ask 가 계약에 있어야 하고, 그 답이 approve 여야 채택된다.
	// ★ 승인할 ask 가 없으면 여전히 거절한다 ★ — 조용히 무시하면 계약 저자는
	// 자기 기준이 걷힌 줄 알고, 그 사실이 Record 를 열어야 드러난다.
	SuccessWhen []contract.Condition `json:"success_when,omitempty"`
}

// applyExpands 는 보고를 마친 단계의 계획을 계약에 붙인다 (ADR-022 §6.3 · P4).
//
// ★ 이것이 ①계획 위임이 실물이 되는 자리다 ★ — 오케스트레이터가 나머지 단계를
// 짓고, Mediator 는 그것을 ★ 검증해서 받는다 ★. 판정이 아니라 검증이다:
// 무엇이 좋은 계획인지 안 보고, ★ 유효한 계약인지만 ★ 본다 (ADR-004 를 안 건드린다).
//
// ★ 붙이는 것이 셋이다 ★
//
//	① 계약의 열에 v2 를 append   앞 판을 안 고친다 (성질 1)
//	② steps 행을 늘린다          claim 이 집을 수 있게 되는 것이 그 형태다
//	③ ★ 늘어난 계약 전체를 다시 검증한다 ★
//	   ⇒ 역할이 requires 에 있는가 · 간선이 DAG 인가 · 스키마가 경계를 안 넘는가
//	   ⇒ ★ 종료가 append 시점에 다시 정적으로 보장된다 ★
//
// ★ 자원은 여전히 사용자가 선언한다 ★ (갈래 A) — 계획이 requires 에 없는 역할을
// 쓰면 검증이 거절한다. 자원까지 위임하는 것은 P5(acquire) 이고 그것은 I5 를 건드린다.
//
// ★ 못 붙이면 그 단계가 FAILED 다 ★ — dispatch 가 이름을 못 고를 때와 같은 자리다.
// 결과가 나쁜 것이 아니라 ★ 계약이 요구한 것을 못 낸 것 ★ 이다.
func (s *Store) applyExpands(ctx context.Context, tx pgx.Tx, runID string, seq int) error {
	var raw []byte
	var assignedJSON []byte
	var versions []byte
	if err := tx.QueryRow(ctx,
		`SELECT `+liveContract+`, assigned, contract_versions FROM runs WHERE run_id=$1`, runID).
		Scan(&raw, &assignedJSON, &versions); err != nil {
		return err
	}
	var c contract.Contract
	if err := json.Unmarshal(raw, &c); err != nil {
		return err
	}
	if seq < 1 || seq > len(c.Steps) {
		return nil
	}
	st := c.Steps[seq-1]
	if !st.Expands {
		return nil
	}

	// ★ 한 단계는 한 번만 늘린다 ★ — 재시도(attempt)나 반복(loop)이 같은 단계를
	// 다시 보고해도 계약이 두 번 자라지 않는다.
	var prior []ContractVersion
	if len(versions) > 0 {
		if err := json.Unmarshal(versions, &prior); err != nil {
			return err
		}
	}
	for _, v := range prior {
		if v.By == byStep(st.ID) {
			return nil
		}
	}
	// ★ 깊이 상한 — 종료 보장이 여기 걸린다 ★ (ADR-031).
	//
	// 계약 하나에 expands 가 여럿일 수 있고 지어진 단계가 또 expands 를 들 수
	// 있으므로, ★ 계약은 원리적으로 무한히 자랄 수 있다 ★. 그것을 막는 것이
	// 판의 개수 상한이고, ★ 계약이 못 건드리는 자리에 둔다 ★ — 계약을 짓는 것이
	// 기계이기 때문이다. 넘으면 그 단계가 FAILED 다: 계획을 받아들일 수 없다는
	// 것이지 계획이 나쁘다는 판정이 아니다 (ADR-004 를 안 건드린다).
	if max := s.maxContractVersions(); len(prior)+1 >= max {
		return fmt.Errorf("step %q: contract version limit (%d) reached; "+
			"the contract cannot grow further", st.ID, max)
	}

	attempt, err := stepAttemptTx(ctx, tx, runID, seq)
	if err != nil {
		return err
	}
	name := st.Out[0] // 검증이 정확히 하나임을 보장한다
	p, err := s.readPlan(runID, name)
	if err != nil {
		return fmt.Errorf("step %q: %w", st.ID, err)
	}
	// ★ 빈 계획은 값이다 ★ (ADR-043)
	//
	// ★ 왜 에러가 아닌가 ★ — 재계획 단계는 needs 로만 이어져 ★ 조건부가 아니다 ★.
	// 앞이 성공해도 돈다. 그때 「고칠 것이 없다」를 낼 방법이 없으면 계획은
	// ★ 반드시 다음 판을 잇게 되고 ★, 그 사슬은 max_versions 상한에 걸려서만
	// 끝난다 — 즉 ★ 성공한 일이 FAILED 로 끝난다 ★. 실측에서 밟았다
	// (zephyr-setup-9: zephyr.elf 를 링크했는데 Run 이 FAILED).
	//
	// ADR-020 이 「가설 없음」을 부재가 아니라 status:none 이라는 ★ 값 ★ 으로 만든 것과
	// 같은 자리다. 부재(파일 없음)는 크래시와 구분되지 않지만, ★ 빈 배열은 판단이다 ★.
	// ★ 약속한 이름을 지었는가 ★ (ADR-049)
	//
	// 계약이 produces 로 "계획은 이 단계를 반드시 짓는다" 를 선언했으면,
	// ★ 그것이 곧 목표다 ★ — success_when 이 이미 그 이름을 가리키고 있다.
	// 빈 계획으로 끝내려면 ★ 약속이 이미 지어져 있어야 한다 ★.
	built := map[string]bool{}
	for _, st := range c.Steps {
		built[st.ID] = true
	}
	for _, ns := range p.Steps {
		built[ns.ID] = true
	}
	var owed []string
	for _, n := range st.Produces {
		if !built[n] {
			owed = append(owed, n)
		}
	}
	if len(owed) > 0 {
		return fmt.Errorf("step %q: the plan did not build the promised steps: %v; "+
			"the goal is not yet in place", st.ID, owed)
	}

	if len(p.Steps) == 0 {
		if len(p.SuccessWhen) > 0 {
			// 늘릴 단계가 없는데 판정할 것이 있다면 계획이 자기모순이다.
			return fmt.Errorf("step %q: plan has no steps but declares success_when; "+
				"there is nothing to judge", st.ID)
		}
		s.log().Info("plan is empty; nothing to fix, contract not extended",
			"run", runID, "step", st.ID)
		// ★ 승인할 것이 없으므로 그 ask 를 건너뛴다 ★ — 안 그러면 사람이
		// 「빈 계획을 승인하라」는 질문을 받고, 그 질문이 Run 을 붙잡는다.
		// SKIPPED 는 종료 상태이면서 실패가 아니고, 그 단계의 조건은
		// ★ 공허하게 참 ★ 이다 (verdict.go). 뒷단계는 needs 를 따라 전파된다.
		return skipAdopters(ctx, tx, runID, c.Steps, st.ID)
	}
	if len(p.SuccessWhen) > 0 && !hasAdopter(c.Steps, st.ID) {
		return fmt.Errorf("step %q: the plan declares success_when but no ask adopts it; "+
			"an ask with adopts pointing at this step is required for the criteria to take effect", st.ID)
	}

	// ★ v2 를 만든다 — 차분이 아니라 전문이다 ★ (성질 4: 각 판이 그 자체로 완결).
	next := c
	next.Steps = append(append([]contract.Step{}, c.Steps...), p.Steps...)
	if err := next.Validate(); err != nil {
		return fmt.Errorf("step %q: the extended contract is not valid: %w", st.ID, err)
	}
	// ★ 제안도 지금 검증한다 ★ (ADR-044)
	//
	// 예전에는 proposed 를 그냥 저장하고 ★ 승인 답이 들어올 때에야 ★ 유효성을 봤다.
	// 그러면 사람이 계획을 다 읽은 뒤에 터지고 ★ 회복 경로가 없다 ★:
	// approve 는 같은 제안이라 또 거절되고, reject 는 늘어난 단계 전체를
	// 무판정으로 돌린다. ★ 답할 수 있는 유일한 답이 나쁜 답이 된다 ★.
	// 실측에서 밟았다 (zephyr-setup-8: agent 단계에 exit_code 를 건 제안).
	//
	// ★ 새 규칙이 하나도 안 는다 ★ — 같은 Validate() 를 제안을 얹어 한 번 더 부른다.
	if len(p.SuccessWhen) > 0 {
		withProposed := next
		withProposed.SuccessWhen = append(
			append([]contract.Condition{}, next.SuccessWhen...), p.SuccessWhen...)
		if err := withProposed.Validate(); err != nil {
			return fmt.Errorf("step %q: the success_when proposed by the plan is not valid: %w",
				st.ID, err)
		}
	}

	ver := ContractVersion{
		V:  len(prior) + 2, // v1 은 runs.contract 가 든다
		At: time.Now().UTC(),
		By: byStep(st.ID),
		// ★ 무엇을 보고 지었나 ★ — 스키마 검증을 통과한 그 산출물이다 (ADR-020).
		Evidence: fmt.Sprintf("blobs/%02d.%d-%s", seq, attempt, name),
		Contract: next,
	}
	// ★ 계획이 지은 success_when 은 「제안」으로만 싣는다 ★ (ADR-033) —
	// 이 판의 유효 조건(Contract.SuccessWhen)에는 안 들어간다. 채택은 답이 한다.
	ver.Proposed = p.SuccessWhen
	// ★ 재계획에서만 cause 를 채운다 ★ (ADR-031) — 첫 판은 「처음 지은 것」이라
	// 바꾼 근거가 없다. 두 번째부터는 ★ 무엇을 보고 다시 짰나 ★ 가 남아야
	// "왜 이 경로로 갔나" 를 봉인된 묶음만 보고 따라갈 수 있다 (성질 4).
	if len(prior) > 0 {
		ver.Cause = causeOf(c.Steps, seq-1)
	}
	prior = append(prior, ver)
	nextJSON, err := json.Marshal(prior)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE runs SET contract_versions=$2 WHERE run_id=$1`, runID, nextJSON); err != nil {
		return err
	}

	// 역할 → 노드. 매처가 t=0 에 이미 정했다 — ★ 계획은 자원을 못 정한다 ★.
	var assigned []Assigned
	if len(assignedJSON) > 0 {
		if err := json.Unmarshal(assignedJSON, &assigned); err != nil {
			return err
		}
	}
	nodeOf := map[string]string{}
	for _, a := range assigned {
		if len(a.Nodes) > 0 {
			nodeOf[a.As] = a.Nodes[0].Node
		}
	}
	// ★ needs 는 늘어난 계약 전체 기준으로 채운다 ★ — 새 단계의 "직전" 은
	// 계획 단계일 수도 있고 새로 지어진 앞 단계일 수도 있다.
	for i := len(c.Steps); i < len(next.Steps); i++ {
		ns := next.Steps[i]
		kind, err := ns.Kind()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO steps (run_id, seq, name, uses, kind, state, node_id, needs)
			 VALUES ($1,$2,$3,$4,$5,'PENDING',$6,$7)`,
			runID, i+1, ns.ID, ns.Uses, kind.String(), nodeOf[ns.Uses],
			contract.NeedsOf(next.Steps, i)); err != nil {
			return err
		}
	}
	return nil
}

// maxContractVersions 는 계약의 열이 가질 수 있는 판의 개수다 (ADR-031).
// ★ 설정이 없으면 2 ★ — 계획 위임 한 번까지. 오늘 동작과 같다.
func (s *Store) maxContractVersions() int {
	if s.MaxContractVersions > 0 {
		return s.MaxContractVersions
	}
	return 2
}

// causeOf 는 그 계획 단계가 ★ 무엇을 보고 지었나 ★ 를 단계 기록 경로로 준다.
//
// needs 가 곧 그 답이다 — 이 단계는 그것들이 끝나야 돌았고, 그 결과를 보고 지었다.
// ★ 여럿일 수 있다 ★: 계획을 다시 짜는 단계가 여러 앞 단계를 기다리는 것은
// 흔하고, 하나만 적으면 근거가 잘린다.
func causeOf(steps []contract.Step, i int) []string {
	index := map[string]int{}
	for k, st := range steps {
		index[st.ID] = k
	}
	var out []string
	for _, name := range contract.NeedsOf(steps, i) {
		if j, ok := index[name]; ok {
			out = append(out, fmt.Sprintf("steps/%02d-%s.json", j+1, name))
		}
	}
	return out
}

// byStep 은 계약 한 판을 지은 단계의 이름이다 (seal.go 의 By* 어휘와 같은 자리).
func byStep(id string) string { return "step:" + id }

// byAnswer 는 ★ 사람의 답이 채택한 판 ★ 이다 (ADR-033).
func byAnswer(id string) string { return "answer:" + id }

// hasAdopter 는 이 expands 단계를 adopts 로 지목한 ask 가 있는지 본다.
// skipAdopters 는 ★ 채택할 것이 없어진 ask 를 건너뛴다 ★ (ADR-043).
//
// 빈 계획을 낸 expands 단계를 adopts 로 지목한 ask 는 물을 것이 없다.
// PENDING 인 것만 바꾼다 — 이미 답이 온 것을 되돌리지 않는다.
func skipAdopters(ctx context.Context, tx pgx.Tx, runID string,
	steps []contract.Step, id string) error {
	var names []string
	for _, st := range steps {
		if st.Ask != nil && st.Ask.Adopts == id {
			names = append(names, st.ID)
		}
	}
	if len(names) == 0 {
		return nil
	}
	if _, err := tx.Exec(ctx, `
		UPDATE steps SET state=$3, ended_at=now()
		 WHERE run_id=$1 AND name = ANY($2) AND state='PENDING'`,
		runID, names, StepSkipped); err != nil {
		return err
	}
	// ★ 뒷단계는 needs 를 따라 전파된다 ★ — dispatch 가 쓰는 것과 같은 기계다.
	return propagateSkips(ctx, tx, runID)
}

func hasAdopter(steps []contract.Step, id string) bool {
	for _, st := range steps {
		if st.Ask != nil && st.Ask.Adopts == id {
			return true
		}
	}
	return false
}

func stepAttemptTx(ctx context.Context, tx pgx.Tx, runID string, seq int) (int, error) {
	var n int
	err := tx.QueryRow(ctx,
		`SELECT attempt FROM steps WHERE run_id=$1 AND seq=$2`, runID, seq).Scan(&n)
	return n, err
}

// readPlan 은 계획 산출물을 읽는다. ★ 형태 검증은 이미 끝나 있다 ★ —
// PUT blob 이 스키마로 거른다 (ADR-020). 여기서 다시 보는 것은 계약으로서의
// 유효성이고 그것은 Validate 가 한다.
func (s *Store) readPlan(runID, name string) (*plan, error) {
	if s.Records == nil {
		return nil, fmt.Errorf("record store is not configured")
	}
	rc, _, err := s.Records.OpenBlob(runID, name)
	if err != nil {
		return nil, fmt.Errorf("cannot read plan %q: %w", name, err)
	}
	defer rc.Close() //nolint:errcheck
	b, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	var p plan
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, fmt.Errorf("plan %q is not an object with steps: %w", name, err)
	}
	return &p, nil
}
