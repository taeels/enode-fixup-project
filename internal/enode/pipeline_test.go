package enode

import (
	"encoding/json"
	"strings"
	"testing"
)

// 계약에서 프롬프트까지 한 번에 지난다 (ADR-057)
//
// 이 시험이 없어서 놓쳤다 — 기존 시험은 buildPrompt 에 req 를 문자열
// 인자로 직접 넘긴다. 프롬프트에 그것이 실리는지는 재지만 계약의 어느
// 필드가 req 가 되는지 는 한 번도 안 쟀다.
//
//	실측 (vm-scratch-6) 오케스트레이터가 과제를 agent.task 에 적었다.
//	AgentParams 에 그 필드가 없어 통째로 버려졌고, in.prompt 가 비어
//	"### 요청" 이 빈 절로 나갔다. 전체 테스트가 통과한 채로.
//
// 같은 병이 verdict_test 에서도 지적됐다(Verify 를 직접 부르고 applyExpands 를
// 안 지난다). 단위는 재는데 배관은 안 쟀다.
func TestContractFieldsReachThePrompt(t *testing.T) {
	// Mediator 가 claim 응답으로 내려보내는 것과 같은 모양의 JSON.
	// 손으로 Step 을 만들지 않는다 — 그러면 json 태그를 안 지난다.
	raw := `{
	  "step_id":"r#03","run_id":"r","seq":3,"name":"install_it","uses":"mac","kind":"agent",
	  "agent":{"max_turns":40,"ask":"never"},
	  "in":{"prompt":"PROMPT_MARKER","from":["survey"]},
	  "out":["install_report"],
	  "feedback":["survey"]
	}`
	var step Step
	if err := json.Unmarshal([]byte(raw), &step); err != nil {
		t.Fatal(err)
	}

	// 계약이 적은 것이 어댑터에 도착했는가
	if step.In.Prompt != "PROMPT_MARKER" {
		t.Fatalf("in.prompt did not arrive: %q", step.In.Prompt)
	}
	if len(step.In.From) != 1 || step.In.From[0] != "survey" {
		t.Fatalf("in.from did not arrive: %v", step.In.From)
	}
	if len(step.Feedback) != 1 || step.Feedback[0] != "survey" {
		t.Fatalf("feedback did not arrive: %v", step.Feedback)
	}

	// 그리고 프롬프트에 실렸는가
	p := buildPrompt(step.In.Prompt, "/o", step.Out, step.Schema,
		map[string]string{"survey": "SURVEY_BODY"}, step.Attempt, step.Expands,
		step.Roles, step.RoleAttrs, step.Owed, step.Standing, nil, nil, step.Goal, step.EnvelopeKey)
	for _, want := range []string{"PROMPT_MARKER", "SURVEY_BODY", "install_report"} {
		if !strings.Contains(p, want) {
			t.Fatalf("%q is missing from the prompt", want)
		}
	}
	// 요청 절이 비면 안 된다 — 그것이 실측에서 밟은 모양이다.
	i := strings.Index(p, "### 요청")
	if i < 0 || strings.TrimSpace(p[i+len("### 요청"):]) == "" {
		t.Fatal("the request section is empty")
	}
}

// 계획을 짓는 단계는 문법과 모양을 함께 받는다 (ADR-057)
//
// 문법은 무엇을 지켜야 하는지, 모양은 필드가 어떻게 생겼는지다.
// 모양이 없어서 계획이 이름에서 유추했고 agent.task 를 지어냈다.
func TestAPlanStepReceivesTheStepShape(t *testing.T) {
	got := buildPrompt("계획을 짜라", "/o", []string{"plan"}, nil, nil, 0, true,
		nil, nil, nil, nil, nil, nil, "목표", "")
	for _, want := range []string{
		"The shape of a step",
		"in.prompt",                 // 과제가 가는 자리
		"execution parameters only", // agent 가 담는 것
		"Without it the agent gets an empty request",
		"in.from and feedback are not the same",
		"Do not invent output names", // 원장을 쓰라는 말
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("%q is missing from the plan prompt", want)
		}
	}
	// 평범한 단계에는 안 싣는다 — 그 단계는 계획을 안 짓는다.
	got = buildPrompt("일해라", "/o", []string{"x"}, nil, nil, 0, false,
		nil, nil, nil, nil, nil, nil, "", "")
	if strings.Contains(got, "The shape of a step") {
		t.Fatal("plan grammar rode along on an ordinary step")
	}
}
