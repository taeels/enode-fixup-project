package enode

import (
	"strings"
	"testing"

	"github.com/taeels/enode/internal/contract"
)

// 계약을 짓는 단계에만 문법이 실린다 (ADR-045)
//
// 평범한 에이전트 단계에까지 실으면 프롬프트만 길어지고 그 단계는
// 계약을 짓지 않으므로 쓸 데가 없다. 반대로 expands 단계에 안 실으면
// 사람이 다시 자연어로 번역하게 된다 — 그것이 아홉 판의 사고였다.
func TestGrammarRidesOnlyOnAPlanStep(t *testing.T) {
	// 계획을 짓는 단계
	got := buildPrompt("build a plan", "/o", []string{"plan"}, nil, nil, 0, true, nil, nil, nil, nil, nil, nil, "", "")
	if !strings.Contains(got, contract.Grammar) {
		t.Fatal("an expands step, yet the contract grammar did not ride along")
	}
	// 평범한 에이전트 단계
	got = buildPrompt("produce a hypothesis", "/o", []string{"hypothesis"}, nil, nil, 0, false, nil, nil, nil, nil, nil, nil, "", "")
	if strings.Contains(got, contract.Grammar) {
		t.Fatal("the contract grammar rode along on an ordinary step — it has no use there")
	}
	// 배출 규약은 둘 다 실린다 — 층이 다르다.
	if !strings.Contains(got, "Output contract") {
		t.Fatal("the output contract is missing")
	}
}

// 문법은 uses 의 문법을 가르치고 어휘는 안 가르친다 (ADR-045)
//
// 역할 이름은 그 계약의 requires 에 있고 계획을 짓는 쪽은 그것을 못 본다.
// 안 실어주면 문자열을 추측한다 — 실측에서 밟았다
// (colima-enode-1: replan_1 이 uses:"claude" 를 지어내 계획 전체가 거절됐다).
// ADR-012 가 capability 어휘에 대해 적은 문장이 한 층 위에서 되풀이된 것이다.
func TestAPlanStepCarriesTheRoleVocabulary(t *testing.T) {
	got := buildPrompt("build a plan", "/o", []string{"plan"}, nil, nil, 0, true,
		[]string{"planner", "mac"}, nil, nil, nil, nil, nil, "", "")
	for _, want := range []string{"planner", "mac", "the only roles you may put in uses"} {
		if !strings.Contains(got, want) {
			t.Fatalf("%q is missing from the prompt", want)
		}
	}
	// 평범한 단계에는 안 싣는다 — 남의 역할 이름은 그 단계에 쓸 데가 없다.
	got = buildPrompt("produce a hypothesis", "/o", []string{"h"}, nil, nil, 0, false,
		[]string{"planner", "mac"}, nil, nil, nil, nil, nil, "", "")
	if strings.Contains(got, "the only roles you may put in uses") {
		t.Fatal("the role vocabulary rode along on an ordinary step")
	}
}

// 목표와 약속이 재계획에 실린다 (ADR-049)
//
// 계획이 지은 재계획 단계의 in.prompt 는 비어 있을 수 있다. 그러면
// 재료를 받고도 무엇을 향해 지을지 모른다 — 실측에서 밟았다:
// "요청 섹션이 비어 있다. 목표는 어디에도 명시돼 있지 않다" 며 _cannot 을 냈고,
// 빈 계획이 나가 아무것도 안 했는데 Run 이 SUCCEEDED로 끝났다.
func TestAPlanStepCarriesTheGoalAndThePromises(t *testing.T) {
	got := buildPrompt("", "/o", []string{"plan2"}, nil, nil, 0, true,
		[]string{"planner", "mac"}, nil, []OwedStep{{Name: "vm_node_up"}}, nil, nil, nil, "stand the VM up as a node", "")
	for _, want := range []string{
		"the goal this run was given at the start", "stand the VM up as a node",
		"the contract promised but nobody has built yet", "vm_node_up",
		"you cannot produce an empty plan",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("%q is missing from the prompt", want)
		}
	}
	// 평범한 단계에는 안 싣는다 — 계약을 짓지 않는 단계에는 쓸 데가 없다.
	got = buildPrompt("do the work", "/o", []string{"x"}, nil, nil, 0, false,
		nil, nil, []OwedStep{{Name: "vm_node_up"}}, nil, nil, nil, "stand the VM up as a node", "")
	if strings.Contains(got, "given at the start") || strings.Contains(got, "vm_node_up") {
		t.Fatal("the goal and the promises rode along on an ordinary step")
	}
}

// 약속된 단계의 「종류」가 실린다 (ADR-049 보강)
//
// 계약이 exit_code 로 판정하는 단계를 계획이 agent 로 지으면 확장된 계약
// 전체가 거절된다 (ADR-019). 그런데 계획을 짓는 쪽은 success_when 을
// 볼 수 없다 — 벽을 보지 못한 채 부딪히고, 재계획은 계획이 짓는 것이라
// 아직 존재하지도 않는다. 즉 수렴이 아니라 벽이다.
func TestTheVerdictOnAPromisedStepRidesAlong(t *testing.T) {
	zero := 0
	owed := []OwedStep{{
		Name: "vm_node_up",
		When: []contract.Condition{{Step: "vm_node_up", ExitCode: &zero}},
	}}
	got := buildPrompt("", "/o", []string{"plan"}, nil, nil, 0, true,
		[]string{"planner", "mac"}, nil, owed, nil, nil, nil, "stand the VM up as a node", "")
	for _, want := range []string{
		"vm_node_up",
		"judged by exit code 0",
		"it must be a command step (run)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("%q is missing from the prompt\n%s", want, got)
		}
	}

	// produced 조건이면 낼 이름을 적어준다 — 그것도 계획이 못 보는 값이다.
	owed = []OwedStep{{
		Name: "vm_caps",
		When: []contract.Condition{{Step: "vm_caps", Produced: []string{"caps.txt"}}},
	}}
	got = buildPrompt("", "/o", []string{"plan"}, nil, nil, 0, true,
		nil, nil, owed, nil, nil, nil, "the goal", "")
	if !strings.Contains(got, "caps.txt") {
		t.Fatal("the name of the artifact to produce did not ride along")
	}

	// 조건이 없으면 아무 말도 안 한다 — produces 로 약속만 하고
	// 판정은 안 걸 수도 있다. 없는 요구를 지어내지 않는다.
	got = buildPrompt("", "/o", []string{"plan"}, nil, nil, 0, true,
		nil, nil, []OwedStep{{Name: "just_a_name"}}, nil, nil, nil, "the goal", "")
	if strings.Contains(got, "it must be a command step (run)") ||
		strings.Contains(got, "this step is judged by exit code") {
		t.Fatal("invented a verdict that was never attached")
	}
}

// 이미 선 단계를 알려준다 (ADR-052)
//
// owed 의 반대쪽이다. 실측(vm-scratch-3): 재계획이 이미 끝난 조사를 또 짓고,
// 이미 있는 이름(replan_1)을 다시 썼다. 되먹임으로 조사 결과는 받았는데
// 그것이 계약의 어디에 있는지는 몰랐다.
func TestAPlanStepCarriesTheStepsAlreadyStanding(t *testing.T) {
	one := 1
	standing := []StandingStep{
		{Name: "plan_setup", State: "DONE"},
		{Name: "mac_survey_1", State: "DONE"},
		{Name: "vm_node_up", State: "DONE", ExitCode: &one},
		{Name: "replan_1", State: "CLAIMED"},
	}
	got := buildPrompt("", "/o", []string{"plan2"}, nil, nil, 0, true,
		nil, nil, nil, standing, nil, nil, "the goal", "")
	for _, want := range []string{
		"steps already standing in the contract",
		"mac_survey_1", "replan_1",
		"exit code 1", "(failed)", // completing and succeeding are different
		"building a new step with the same",
		"Do not rebuild work that is already done",
		"build a new step that fixes it",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("%q is missing from the prompt\n%s", want, got)
		}
	}
	// 성공한 단계에 실패 표시를 붙이지 않는다
	if strings.Contains(got, "plan_setup") &&
		strings.Contains(got, "plan_setup           DONE  exit code") {
		t.Fatal("invented an exit code for a step that has none")
	}

	// 평범한 단계에는 안 싣는다 — 남의 상태는 그 단계에 쓸 데가 없고,
	// 알면 그것으로 자기 판정을 흉내낼 여지만 생긴다 (ADR-037).
	got = buildPrompt("do the work", "/o", []string{"x"}, nil, nil, 0, false,
		nil, nil, nil, standing, nil, nil, "the goal", "")
	if strings.Contains(got, "steps already standing in the contract") {
		t.Fatal("another step's state rode along on an ordinary step")
	}
}

// 역할 옆에 그 기계의 사실이 실린다 (ADR-055)
//
// 실측(vm-scratch-1..5): 계획이 매 판 uname · sw_vers · ls 를 돌려 이것을
// 알아냈고, 그 답을 보려면 판이 하나 더 들었다. 매처는 이미 이 값으로
// 노드를 골랐는데 계획만 못 봤다.
func TestAPlanStepCarriesTheMachineFactsOfEachRole(t *testing.T) {
	attrs := map[string]map[string]string{
		"mac": {"os": "darwin", "host_arch": "arm64",
			"ws": "/Users/x/enode-ws", "harness": "claude", "arch": "arm64"},
	}
	got := buildPrompt("", "/o", []string{"plan"}, nil, nil, 0, true,
		[]string{"planner", "mac"}, attrs, nil, nil, nil, nil, "the goal", "")
	for _, want := range []string{
		"os=darwin", "host_arch=arm64", "ws=/Users/x/enode-ws",
		"Do not go survey what you already know",
		"arch is the build target, not the machine",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("%q is missing from the prompt\n%s", want, got)
		}
	}
	// 순서가 고정이다 — 맵을 그대로 돌면 프롬프트가 매번 달라진다.
	first := attrLine(attrs["mac"])
	for i := 0; i < 20; i++ {
		if attrLine(attrs["mac"]) != first {
			t.Fatal("the attribute line wobbles in order — one contract would give different prompts")
		}
	}

	// 속성이 없는 역할은 이름만 — 없는 사실을 지어내지 않는다.
	got = buildPrompt("", "/o", []string{"plan"}, nil, nil, 0, true,
		[]string{"planner"}, nil, nil, nil, nil, nil, "the goal", "")
	if strings.Contains(got, "Do not go survey what you already know") {
		t.Fatal("wrote the guidance although no facts rode along")
	}

	// 평범한 단계에는 안 싣는다
	got = buildPrompt("do the work", "/o", []string{"x"}, nil, nil, 0, false,
		[]string{"mac"}, attrs, nil, nil, nil, nil, "the goal", "")
	if strings.Contains(got, "os=darwin") {
		t.Fatal("another machine's facts rode along on an ordinary step")
	}
}

// 계획을 짓는 단계에만 목표 미달의 자리를 준다 (ADR-054)
func TestTheGoalUnmetLaneExistsOnlyOnAPlanStep(t *testing.T) {
	got := buildPrompt("build a plan", "/o", []string{"plan"}, nil, nil, 0, true,
		nil, nil, nil, nil, nil, nil, "the goal", "")
	for _, want := range []string{
		"When the goal was not reached", "_unmet",
		"can only fail a run that would have passed",
		"together with a plan",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("%q is missing from the prompt", want)
		}
	}
	// 평범한 단계는 목표를 모른다 — 그 자리를 주면 자기 일만 보고 Run 을 죽인다.
	got = buildPrompt("do the work", "/o", []string{"x"}, nil, nil, 0, false,
		nil, nil, nil, nil, nil, nil, "the goal", "")
	if strings.Contains(got, "_unmet") {
		t.Fatal("gave an ordinary step the goal-unmet lane")
	}
	// 그래도 _cannot 은 있다 — 그 단계를 못 하겠다는 것은 누구나 말한다.
	if !strings.Contains(got, "_cannot") {
		t.Fatal("the failure lane disappeared")
	}
}
