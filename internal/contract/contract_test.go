package contract

import (
	"encoding/json"
	"errors"
	"testing"
)

// 속성은 as/capability/count 와 같은 층의 형제 키로 온다.
// 이 왕복이 깨지면 계약 문서와 코드가 갈라진 것이다.
func TestRequireRoundTrip(t *testing.T) {
	in := []byte(`{"as":"builder","capability":"agent.reason","arch":"armv7","repo":"gerrit.corp/kernel/linux","count":3}`)
	var r Require
	if err := json.Unmarshal(in, &r); err != nil {
		t.Fatal(err)
	}
	if r.As != "builder" || r.Capability != CapabilityAgentReason || r.Count != 3 {
		t.Fatalf("구조 키가 안 걸렸다: %+v", r)
	}
	if r.Attrs["arch"] != "armv7" || r.Attrs["repo"] != "gerrit.corp/kernel/linux" {
		t.Fatalf("속성이 안 걸렸다: %+v", r.Attrs)
	}
	if len(r.Attrs) != 2 {
		t.Fatalf("구조 키가 속성으로 샜다: %+v", r.Attrs)
	}

	out, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var back Require
	if err := json.Unmarshal(out, &back); err != nil {
		t.Fatal(err)
	}
	if back.As != r.As || back.Count != r.Count || len(back.Attrs) != len(r.Attrs) {
		t.Fatalf("왕복이 깨졌다: %+v → %+v", r, back)
	}
}

// ADR-011 — 라벨 매칭 계열이므로 속성 값은 문자열뿐이다.
// 숫자나 객체를 허용하면 범위 비교로 미끄러지고 그게 표현식 언어의 시작이다.
func TestRequireRejectsNonStringAttr(t *testing.T) {
	for _, in := range []string{
		`{"as":"b","capability":"agent.reason","free_gb":40}`,
		`{"as":"b","capability":"agent.reason","arch":{"in":["armv7"]}}`,
	} {
		var r Require
		if err := json.Unmarshal([]byte(in), &r); err == nil {
			t.Fatalf("문자열 아닌 속성을 받아버렸다: %s → %+v", in, r)
		}
	}
}

// ADR-019 결정 3 — 단계는 두 종류다. 암묵을 남기지 않는다.
func TestStepKind(t *testing.T) {
	cases := []struct {
		name string
		s    Step
		want StepKind
		err  bool
	}{
		{"agent 단계", Step{ID: "h", Agent: map[string]interface{}{"model": "x"}}, KindAgent, false},
		{"명령 단계", Step{ID: "b", Run: []string{"make", "-j8"}}, KindRun, false},
		{"둘 다 있으면 틀렸다", Step{ID: "x", Agent: map[string]interface{}{}, Run: []string{"make"}}, KindUnknown, true},
		{"둘 다 없으면 틀렸다", Step{ID: "y"}, KindUnknown, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := c.s.Kind()
			if (err != nil) != c.err {
				t.Fatalf("err=%v, 기대 err=%v", err, c.err)
			}
			if got != c.want {
				t.Fatalf("kind=%v, 기대 %v", got, c.want)
			}
		})
	}
}

func agentStep(id, uses string) Step {
	return Step{ID: id, Uses: uses, Agent: map[string]interface{}{"ask": "never"}, Out: []string{id}}
}
func runStep(id, uses string) Step {
	return Step{ID: id, Uses: uses, Run: []string{"true"}, Out: []string{id}}
}
func req(as string) Require {
	return Require{As: as, Capability: CapabilityAgentReason, Attrs: map[string]string{}}
}
func zero() *int { z := 0; return &z }

func TestValidate(t *testing.T) {
	base := func() Contract {
		return Contract{
			RunID:    "gerrit-1-ps1",
			Requires: []Require{req("brain"), req("builder")},
			Steps:    []Step{agentStep("hypothesis", "brain"), runStep("build", "builder")},
			SuccessWhen: []Condition{
				{Step: "hypothesis", Produced: []string{"hypothesis"}},
				{Step: "build", ExitCode: zero()},
			},
		}
	}

	if err := base().Validate(); err != nil {
		t.Fatalf("정상 계약이 거절됐다: %v", err)
	}

	cases := []struct {
		name string
		mut  func(*Contract)
		want error
	}{
		{"run_id 없음", func(c *Contract) { c.RunID = "" }, ErrNoRunID},
		{"requires 비었음", func(c *Contract) { c.Requires = nil }, ErrNoRequires},
		{"steps 비었음", func(c *Contract) { c.Steps = nil }, ErrNoSteps},
		// ADR-019 결정 1 — 어휘는 agent.reason 하나뿐이다.
		{"옛 capability", func(c *Contract) { c.Requires[0].Capability = "build.linux" }, ErrUnknownCap},
		{"as 중복", func(c *Contract) { c.Requires[1].As = "brain" }, ErrDupAs},
		{"step id 중복", func(c *Contract) { c.Steps[1].ID = "hypothesis" }, ErrDupStepID},
		// ★ ADR-004 를 지키는 한 줄 ★ — claude 는 헛소리를 하고도 0 으로 끝난다.
		{"agent 단계에 exit_code", func(c *Contract) {
			c.SuccessWhen[0].ExitCode = zero()
		}, ErrExitOnAgent},
		{"없는 단계를 가리킴", func(c *Contract) { c.SuccessWhen[0].Step = "nope" }, ErrCondUnknownID},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := base()
			tc.mut(&c)
			err := c.Validate()
			if err == nil {
				t.Fatalf("통과해버렸다")
			}
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("err=%v, 기대 %v", err, tc.want)
			}
		})
	}
}

// ADR-011 — 부분집합 일치. 노드에 여분 속성이 있는 건 상관없다.
func TestCapabilitySatisfies(t *testing.T) {
	node := Capability{
		Capability: CapabilityAgentReason,
		Attrs: map[string]string{
			"harness": "claude", "repo": "gerrit.corp/kernel/linux",
			"arch": "armv7", "board": "SoC-X", "tag": "board-042",
		},
	}
	cases := []struct {
		name  string
		attrs map[string]string
		want  bool
	}{
		{"속성 없는 요구는 아무나", map[string]string{}, true},
		{"부분집합", map[string]string{"arch": "armv7"}, true},
		{"여럿 부분집합", map[string]string{"arch": "armv7", "repo": "gerrit.corp/kernel/linux"}, true},
		{"값이 다르면 불일치", map[string]string{"arch": "x86_64"}, false},
		{"없는 키면 불일치", map[string]string{"machine": "qemu"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := Require{Capability: CapabilityAgentReason, Attrs: c.attrs}
			if got := node.Satisfies(r); got != c.want {
				t.Fatalf("got=%v 기대 %v", got, c.want)
			}
		})
	}
}

// ★ orchestration 은 평범한 Run 에 안 끌려간다 ★ (ADR-022 §5.2)
//
// 어휘를 나눈 목적이 ★ 배제 ★ 다. capability 는 완전일치라 서로를 안 만족시킨다.
func TestCapability_어휘가_서로를_배제한다(t *testing.T) {
	orch := Capability{Capability: CapabilityOrchestration,
		Attrs: map[string]string{"harness": "claude"}}
	agent := Capability{Capability: CapabilityAgentReason,
		Attrs: map[string]string{"harness": "claude"}}

	// ★ 핵심 ★ — 평범한 Run 이 오케스트레이터를 못 잡는다
	if orch.Satisfies(Require{Capability: CapabilityAgentReason}) {
		t.Fatal("★ 평범한 Run 이 오케스트레이터를 잡아간다 ★")
	}
	// 속성까지 같아도 마찬가지다 — 부분집합 매칭이 capability 를 못 넘는다
	if orch.Satisfies(Require{Capability: CapabilityAgentReason,
		Attrs: map[string]string{"harness": "claude"}}) {
		t.Fatal("★ 속성 매칭이 capability 경계를 넘었다 ★")
	}
	// 반대도 성립한다
	if agent.Satisfies(Require{Capability: CapabilityOrchestration}) {
		t.Fatal("평범한 노드가 오케스트레이션 요구를 만족시켰다")
	}
	// 제 짝은 만족시킨다
	if !orch.Satisfies(Require{Capability: CapabilityOrchestration,
		Attrs: map[string]string{"harness": "claude"}}) {
		t.Fatal("제 짝을 못 만족시킨다")
	}
}

// ★ 모르는 capability 는 거절한다 ★ — 열린 어휘가 아니다.
// 오타가 조용히 통과하면 계약 저자가 422 대신 "후보 없음" 을 보게 된다.
func TestValidate_모르는_capability는_거절한다(t *testing.T) {
	c := Contract{
		RunID:    "r1",
		Requires: []Require{{As: "x", Capability: "orchestraton"}}, // 오타
		Steps:    []Step{{ID: "s", Uses: "x", Run: []string{"true"}}},
	}
	if err := c.Validate(); err == nil {
		t.Fatal("★ 오타가 통과했다 ★")
	}
	c.Requires[0].Capability = CapabilityOrchestration
	if err := c.Validate(); err != nil {
		t.Fatalf("정상 어휘가 거절됐다: %v", err)
	}
}

// ★ dispatch 검증 — DAG 를 정적으로 확인한다 ★ (ADR-022 §7.2)
//
// 뒤로 못 가면 ★ 종료가 계약 검증 단계에서 보장된다 ★. 실행 중에 무한 루프를
// 발견하는 것과 제출 시점에 400 을 받는 것은 다르다.
func TestValidate_dispatch(t *testing.T) {
	mk := func(d *Dispatch) Contract {
		return Contract{
			RunID:    "r1",
			Requires: []Require{{As: "b", Capability: CapabilityAgentReason}},
			Steps: []Step{
				{ID: "triage", Uses: "b", Agent: map[string]interface{}{}, Out: []string{"route"}, Dispatch: d},
				{ID: "full", Uses: "b", Run: []string{"true"}},
				{ID: "quick", Uses: "b", Run: []string{"true"}},
			},
		}
	}
	ok := &Dispatch{From: "route.next", To: []string{"full", "quick"}}
	if err := mk(ok).Validate(); err != nil {
		t.Fatalf("정상 dispatch 가 거절됐다: %v", err)
	}

	for name, d := range map[string]*Dispatch{
		"from 이 비었다":     {To: []string{"full", "quick"}},
		"안 내는 산출물을 가리킨다": {From: "없는것.next", To: []string{"full", "quick"}},
		"갈림길이 하나다":       {From: "route.next", To: []string{"full"}},
		"없는 단계를 가리킨다":    {From: "route.next", To: []string{"full", "없는것"}},
		"중복이 있다":         {From: "route.next", To: []string{"full", "full"}},
		"★ 자기를 가리킨다 ★":   {From: "route.next", To: []string{"full", "triage"}},
	} {
		if err := mk(d).Validate(); err == nil {
			t.Fatalf("%s — 통과했다: %+v", name, d)
		}
	}
}
