package contract

import (
	"encoding/json"
	"fmt"
	"sort"
)

// PlanDoc 은 expands 단계가 내는 산출물의 형태다.
// store 의 plan 과 같은 모양이지만 여기는 계약 패키지라 훅도 쓸 수 있다.
type PlanDoc struct {
	Steps       []Step      `json:"steps"`
	SuccessWhen []Condition `json:"success_when,omitempty"`
}

// CheckPlan 은 계획을 그 자체로 검증한다 (ADR-046).
//
// 왜 필요한가 — 프롬프트로는 안 닫힌다
//
// 계약 문법을 프롬프트에 심어도(ADR-045) 모델이 어긴다. 실측이 그랬다:
//
//	10차  uses 에 없는 역할 "claude" 를 지어냈다   → 계획 전체가 거절됐다
//	11차  schema 를 산출물 이름으로 키잉 안 했다   → 계획 전체가 거절됐다
//
// 그때는 이미 늦다 — 하네스가 끝난 뒤라 고칠 기회가 없고, 계획 한 판
// (약 $0.2~0.45)과 사람의 검토가 통째로 버려진다.
//
// 그래서 하네스가 끝나기 전에 짚는다 — Stop 훅이 이것을 부르고,
// 어겼으면 종료를 막고 오류 문장 그대로 되먹인다. 모델은 같은 세션에서
// 고친다. hook.go 가 적어둔 세 겹 중 ③ 보조의 자리다.
//
// 진실의 원천이 둘이 되지 않는다 — 같은 Validate() 를 부른다.
// 훅은 빠르고 불완전 하고, applyExpands 가 느리고 완전하다.
//
//	훅이 잡는 것    계획 자체의 모양 — 역할·필드·조건·스키마·DAG
//	훅이 못 잡는 것 이 계약에 붙는가 — 부모 단계와의 id 충돌 ·
//	                needs 가 가리키는 부모 단계의 실존 · 판의 개수 상한
//	                ⇒ 그것은 applyExpands 가 본다 (ADR-044). 권위는 거기 있다
//
// roles 는 uses 에 쓸 수 있는 이름이다 (ADR-045). 비면 역할 검사를 건너뛴다 —
// 모르는 것으로 막지 않는다.
func CheckPlan(raw []byte, roles []string) error {
	var p PlanDoc
	if err := json.Unmarshal(raw, &p); err != nil {
		return fmt.Errorf("plan is not valid JSON: %w", err)
	}
	// 빈 계획은 값이다 (ADR-043) — 검사할 것이 없다.
	if len(p.Steps) == 0 {
		if len(p.SuccessWhen) > 0 {
			return fmt.Errorf("plan has no steps but declares success_when; there is nothing to judge")
		}
		return nil
	}
	// 역할을 모르면 역할로 막지 않는다 — 모르는 것으로 막는 것이 가장 나쁜
	// 안전망이다. 계획이 쓴 이름을 전부 선언한 것으로 치고 나머지만 본다.
	if len(roles) == 0 {
		seen := map[string]bool{}
		for _, st := range p.Steps {
			if st.Uses != "" && !seen[st.Uses] {
				seen[st.Uses] = true
				roles = append(roles, st.Uses)
			}
		}
		if len(roles) == 0 {
			roles = []string{"_"}
		}
		sort.Strings(roles)
	}

	c := Contract{RunID: "_plan", Steps: p.Steps, SuccessWhen: p.SuccessWhen}
	for _, r := range roles {
		c.Requires = append(c.Requires, Require{As: r, Capability: CapabilityAgentReason})
	}
	// 계획 밖을 가리키는 needs 는 그루터기로 채운다
	//
	// 계획의 첫 단계는 보통 부모의 승인 단계를 needs 로 잡는다 — 그 단계는
	// 여기 없다. 그것을 오류로 치면 정당한 계획을 막는다.
	// 그루터기를 앞에 놓아 참조만 성립시키고, 실존 여부는 applyExpands 가 본다.
	c.Steps = append(stubsFor(p.Steps, roles[0]), p.Steps...)
	// success_when 도 계획 밖을 가리킬 수 있다 — 같은 이유로 그루터기가 받는다.

	if err := c.Validate(); err != nil {
		return err
	}
	return nil
}

// stubsFor 는 계획 안에서 정의되지 않은 needs·adopts 대상을 채운다.
func stubsFor(steps []Step, role string) []Step {
	in := map[string]bool{}
	for _, st := range steps {
		in[st.ID] = true
	}
	want := map[string]bool{}
	add := func(n string) {
		if n != "" && !in[n] {
			want[n] = true
		}
	}
	for _, st := range steps {
		for _, n := range st.Needs {
			add(n)
		}
		if st.Ask != nil {
			add(st.Ask.Adopts)
		}
		if st.Loop != nil {
			add(st.Loop.BackTo)
		}
	}
	// 결정적 순서 — 오류 메시지가 실행마다 달라지면 안 된다.
	names := make([]string, 0, len(want))
	for n := range want {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]Step, 0, len(names))
	for _, n := range names {
		out = append(out, Step{ID: n, Uses: role, Run: []string{"true"}, Needs: []string{}})
	}
	return out
}
