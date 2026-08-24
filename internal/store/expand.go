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

	// ★ 목표 미달을 ★ 무엇보다 먼저 ★ 본다 ★ (ADR-054)
	//
	// ★ 순서가 곧 설계다 ★ — 처음에는 이 검사를 빈 계획 검사 옆에 뒀는데,
	// 그러면 ★ 도달할 수 없었다 ★. 셋이 앞에서 막았다:
	//
	//	readPlan     "더 해도 소용없다" 면서 계획 파일을 낼 이유가 없다.
	//	             안 내면 ErrNoBlob 으로 그 단계가 FAILED 가 되고,
	//	             그러면 ★ SettleIfDone 이 Verify 를 아예 안 부른다 ★
	//	owed 검사    produces 로 목표를 못 박은 계약 — ★ ADR-049 가 권하는 바로 그 형태 ★ —
	//	             에서는 약속이 남아 있어 먼저 거절됐다.
	//	             ★ 「약속한 단계를 못 짓겠다」가 곧 목표 미달이다 ★
	//	모순 검사    계획이 있어야 도달하는 자리에 있었다
	//
	// ★ 그래서 여기가 맞는 자리다 ★ — 계획을 읽기도 전에, 약속을 묻기도 전에.
	if unmetReported(ctx, tx, runID, seq) {
		// ★ 계획을 함께 냈으면 모순이다 ★ — "더 해도 소용없다" 와
		// "이렇게 하면 된다" 는 함께 설 수 없다. 다만 ★ 빈 계획은 무해하다 ★:
		// 같은 말을 두 번 한 것뿐이다.
		if p, err := s.readPlan(runID, name); err == nil && len(p.Steps) > 0 {
			return fmt.Errorf("step %q: %s was written but the plan is not empty; "+
				"%s means nothing further will help — write one or the other, not both",
				st.ID, contract.UnmetName, contract.UnmetName)
		}
		// ★ 오류를 내지 않는다 ★ — 오류면 이 단계가 FAILED 가 되고
		// SettleIfDone 이 Verify 를 건너뛴다. 그러면 ★ 봉인되는 이유가
		// "완주하지 못했다" 로 바뀐다 ★ — 실제로는 완주했고 자기에게
		// 불리한 판단을 스스로 적었다 (ADR-004 가 가른 것이 다시 뭉개진다).
		s.log().Info("the step reported that the goal was not reached; "+
			"the contract is not extended", "run", runID, "step", st.ID)
		return skipAdopters(ctx, tx, runID, c.Steps, st.ID)
	}

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
	// ★ yolo 는 스스로 채택자다 ★ (ADR-061 §3) — 계약이 "묻지 않고 채택한다" 를
	// 미리 적었으므로 물어볼 ask 를 요구하지 않는다. 그 선언 자체가 사람의
	// 것이므로 ADR-033 의 "승인할 ask 가 없으면 거절한다" 를 우회하는 것이
	// 아니라 ★ 명시적으로 여는 다른 문 ★ 이다.
	if len(p.SuccessWhen) > 0 && st.Adopt != contract.AdoptYolo && !hasAdopter(c.Steps, st.ID) {
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

	// ★ 묻지 않고 채택한다 ★ (ADR-061 §3) — 계약이 adopt:"yolo" 를 달았으면
	// 제안이 ★ 이 트랜잭션 안에서 ★ 효력을 얻는다. 물어볼 ask 가 없으므로
	// 나중이 없다.
	//
	// ★ 판을 따로 붙인다 ★ — 제안(by: step:…)과 채택(by: contract:yolo)이
	// 한 판에 섞이면 ★ 저자와 승인자가 봉인에서 겹친다 ★ (ADR-033). 사람이
	// 답하는 경로에서 판이 둘인 것과 같은 모양을 지킨다.
	if st.Adopt == contract.AdoptYolo && len(ver.Proposed) > 0 {
		adopted := next
		adopted.SuccessWhen = append([]contract.Condition{}, next.SuccessWhen...)
		for _, cond := range ver.Proposed {
			if contract.HasCondition(adopted.SuccessWhen, cond) {
				continue
			}
			adopted.SuccessWhen = append(adopted.SuccessWhen, cond)
		}
		// ★ 채택 시점에 전체를 다시 검증한다 ★ — 사람이 답하는 경로와 같다.
		if err := adopted.Validate(); err != nil {
			return fmt.Errorf("adopting the proposal would make the contract invalid: %w", err)
		}
		prior = append(prior, ContractVersion{
			V:  len(prior) + 2,
			At: time.Now().UTC(),
			By: byYolo(),
			// ★ evidence 를 지어내지 않는다 ★ — 사람의 답 blob 이 없다.
			// 「없음」이 곧 "묻지 않았다" 는 값이다 (ADR-020 · ADR-058).
			Cause:    []string{ver.Evidence},
			Contract: adopted,
		})
		s.log().Info("proposal adopted without asking; the contract said so",
			"run", runID, "step", st.ID, "conditions", len(ver.Proposed))
	}

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

// byYolo 는 ★ 계약이 미리 맡긴 채택 ★ 이다 (ADR-061 §3).
// ★ 답이 없다 ★ — 그래서 answer:<ask> 가 아니라 계약 자신을 가리킨다.
func byYolo() string { return "contract:" + contract.AdoptYolo }

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

// unmetReported 는 그 단계가 ★ 목표 미달 ★ 을 보고했는지다 (ADR-054).
//
// ★ 산출물 목록에서 본다 ★ — _unmet 은 밑줄 예약이라 계약이 그 이름을 못 쓰고,
// 그래서 이 자리에 있는 것은 어댑터가 수확한 자백뿐이다.
func unmetReported(ctx context.Context, tx pgx.Tx, runID string, seq int) bool {
	var produced []string
	if err := tx.QueryRow(ctx,
		`SELECT coalesce(array(SELECT jsonb_array_elements_text(result->'produced')), '{}')
		   FROM steps WHERE run_id=$1 AND seq=$2`, runID, seq).Scan(&produced); err != nil {
		return false
	}
	for _, n := range produced {
		if n == contract.UnmetName {
			return true
		}
	}
	return false
}
