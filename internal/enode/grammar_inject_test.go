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
	got := buildPrompt("계획을 짜라", "/o", []string{"plan"}, nil, nil, 0, true, nil)
	if !strings.Contains(got, contract.Grammar) {
		t.Fatal("★ expands 단계인데 계약 문법이 안 실렸다 ★")
	}
	// 평범한 에이전트 단계
	got = buildPrompt("가설을 내라", "/o", []string{"hypothesis"}, nil, nil, 0, false, nil)
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
		[]string{"planner", "mac"})
	for _, want := range []string{"planner", "mac", "uses 에 쓸 수 있는 역할"} {
		if !strings.Contains(got, want) {
			t.Fatalf("★ %q 가 프롬프트에 없다 ★", want)
		}
	}
	// ★ 평범한 단계에는 안 싣는다 ★ — 남의 역할 이름은 그 단계에 쓸 데가 없다.
	got = buildPrompt("가설을 내라", "/o", []string{"h"}, nil, nil, 0, false,
		[]string{"planner", "mac"})
	if strings.Contains(got, "uses 에 쓸 수 있는 역할") {
		t.Fatal("★ 평범한 단계에 역할 어휘가 실렸다 ★")
	}
}
