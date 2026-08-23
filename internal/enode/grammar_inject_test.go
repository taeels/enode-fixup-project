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
	got := buildPrompt("계획을 짜라", "/o", []string{"plan"}, nil, nil, 0, true)
	if !strings.Contains(got, contract.Grammar) {
		t.Fatal("★ expands 단계인데 계약 문법이 안 실렸다 ★")
	}
	// 평범한 에이전트 단계
	got = buildPrompt("가설을 내라", "/o", []string{"hypothesis"}, nil, nil, 0, false)
	if strings.Contains(got, contract.Grammar) {
		t.Fatal("★ 평범한 단계에 계약 문법이 실렸다 — 쓸 데가 없다 ★")
	}
	// ★ 배출 규약은 둘 다 실린다 ★ — 층이 다르다.
	if !strings.Contains(got, "배출 규약") {
		t.Fatal("배출 규약이 빠졌다")
	}
}
