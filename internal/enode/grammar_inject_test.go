package enode

import (
	"strings"
	"testing"

	"github.com/taeels/enode/internal/contract"
)

// ★ 계약을 짓는 단계에만 문법이 실린다 ★ (ADR-045)
//
// 평범한 에이전트 단계에까지 실으면 ★ 프롬프트만 길어지고 ★ 그 단계는
// 계약을 짓지 않으므로 쓸 데가 없다. 반대로 expands 단계에 안 실으면
// 사람이 다시 자연어로 번역하게 된다 — 그것이 아홉 판의 사고였다.
func Test문법은_계획단계에만_실린다(t *testing.T) {
	// 계획을 짓는 단계
	got := buildPrompt("계획을 짜라", "/o", []string{"plan"}, nil, nil, 0, true, nil, nil, "", "")
	if !strings.Contains(got, contract.Grammar) {
		t.Fatal("★ expands 단계인데 계약 문법이 안 실렸다 ★")
	}
	// 평범한 에이전트 단계
	got = buildPrompt("가설을 내라", "/o", []string{"hypothesis"}, nil, nil, 0, false, nil, nil, "", "")
	if strings.Contains(got, contract.Grammar) {
		t.Fatal("★ 평범한 단계에 계약 문법이 실렸다 — 쓸 데가 없다 ★")
	}
	// ★ 배출 규약은 둘 다 실린다 ★ — 층이 다르다.
	if !strings.Contains(got, "배출 규약") {
		t.Fatal("배출 규약이 빠졌다")
	}
}

// ★ 문법은 uses 의 문법을 가르치고 어휘는 안 가르친다 ★ (ADR-045)
//
// 역할 이름은 그 계약의 requires 에 있고 계획을 짓는 쪽은 그것을 못 본다.
// 안 실어주면 ★ 문자열을 추측한다 ★ — 실측에서 밟았다
// (colima-enode-1: replan_1 이 uses:"claude" 를 지어내 계획 전체가 거절됐다).
// ADR-012 가 capability 어휘에 대해 적은 문장이 한 층 위에서 되풀이된 것이다.
func Test계획단계에_역할_어휘가_실린다(t *testing.T) {
	got := buildPrompt("계획을 짜라", "/o", []string{"plan"}, nil, nil, 0, true,
		[]string{"planner", "mac"}, nil, "", "")
	for _, want := range []string{"planner", "mac", "uses 에 쓸 수 있는 역할"} {
		if !strings.Contains(got, want) {
			t.Fatalf("★ %q 가 프롬프트에 없다 ★", want)
		}
	}
	// ★ 평범한 단계에는 안 싣는다 ★ — 남의 역할 이름은 그 단계에 쓸 데가 없다.
	got = buildPrompt("가설을 내라", "/o", []string{"h"}, nil, nil, 0, false,
		[]string{"planner", "mac"}, nil, "", "")
	if strings.Contains(got, "uses 에 쓸 수 있는 역할") {
		t.Fatal("★ 평범한 단계에 역할 어휘가 실렸다 ★")
	}
}

// ★ 목표와 약속이 재계획에 실린다 ★ (ADR-049)
//
// 계획이 지은 재계획 단계의 in.prompt 는 비어 있을 수 있다. 그러면
// ★ 재료를 받고도 무엇을 향해 지을지 모른다 ★ — 실측에서 밟았다:
// "요청 섹션이 비어 있다. 목표는 어디에도 명시돼 있지 않다" 며 _cannot 을 냈고,
// 빈 계획이 나가 ★ 아무것도 안 했는데 Run 이 SUCCEEDED ★ 로 끝났다.
func Test계획단계에_목표와_약속이_실린다(t *testing.T) {
	got := buildPrompt("", "/o", []string{"plan2"}, nil, nil, 0, true,
		[]string{"planner", "mac"}, []OwedStep{{Name: "vm_node_up"}}, "VM 을 노드로 세워라", "")
	for _, want := range []string{
		"이 Run 이 처음 받은 목표", "VM 을 노드로 세워라",
		"약속했는데 아직 안 지어진 단계", "vm_node_up",
		"빈 계획을 낼 수 없다",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("★ %q 가 프롬프트에 없다 ★", want)
		}
	}
	// ★ 평범한 단계에는 안 싣는다 ★ — 계약을 짓지 않는 단계에는 쓸 데가 없다.
	got = buildPrompt("일해라", "/o", []string{"x"}, nil, nil, 0, false,
		nil, []OwedStep{{Name: "vm_node_up"}}, "VM 을 노드로 세워라", "")
	if strings.Contains(got, "처음 받은 목표") || strings.Contains(got, "vm_node_up") {
		t.Fatal("★ 평범한 단계에 목표·약속이 실렸다 ★")
	}
}

// ★ 약속된 단계의 「종류」가 실린다 ★ (ADR-049 보강)
//
// 계약이 exit_code 로 판정하는 단계를 계획이 agent 로 지으면 ★ 확장된 계약
// 전체가 거절된다 ★ (ADR-019). 그런데 계획을 짓는 쪽은 success_when 을
// 볼 수 없다 — ★ 벽을 보지 못한 채 부딪히고, 재계획은 계획이 짓는 것이라
// 아직 존재하지도 않는다 ★. 즉 수렴이 아니라 벽이다.
func Test약속된_단계의_판정이_실린다(t *testing.T) {
	zero := 0
	owed := []OwedStep{{
		Name: "vm_node_up",
		When: []contract.Condition{{Step: "vm_node_up", ExitCode: &zero}},
	}}
	got := buildPrompt("", "/o", []string{"plan"}, nil, nil, 0, true,
		[]string{"planner", "mac"}, owed, "VM 을 노드로 세워라", "")
	for _, want := range []string{
		"vm_node_up",
		"종료코드 0 으로 판정된다",
		"명령 단계(run)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("★ %q 가 프롬프트에 없다 ★\n%s", want, got)
		}
	}

	// ★ produced 조건이면 낼 이름을 적어준다 ★ — 그것도 계획이 못 보는 값이다.
	owed = []OwedStep{{
		Name: "vm_caps",
		When: []contract.Condition{{Step: "vm_caps", Produced: []string{"caps.txt"}}},
	}}
	got = buildPrompt("", "/o", []string{"plan"}, nil, nil, 0, true,
		nil, owed, "목표", "")
	if !strings.Contains(got, "caps.txt") {
		t.Fatal("★ 내야 할 산출물 이름이 안 실렸다 ★")
	}

	// ★ 조건이 없으면 아무 말도 안 한다 ★ — produces 로 약속만 하고
	// 판정은 안 걸 수도 있다. 없는 요구를 지어내지 않는다.
	got = buildPrompt("", "/o", []string{"plan"}, nil, nil, 0, true,
		nil, []OwedStep{{Name: "just_a_name"}}, "목표", "")
	if strings.Contains(got, "명령 단계(run)") || strings.Contains(got, "판정된다") {
		t.Fatal("★ 걸리지 않은 판정을 지어냈다 ★")
	}
}
