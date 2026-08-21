package contract

import (
	"encoding/json"
	"errors"
	"os"
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

// ★ needs 검증 — dispatch 와 같은 것을 지킨다 ★ (ADR-023 §4.3)
//
// 간선이 전부 뒤를 향하면 그래프가 DAG 이고, 그래서 ★ 종료가 제출 시점에
// 정적으로 보장된다 ★. 뒤로 가야 하는 것은 의존이 아니라 반복이다.
func TestValidate_needs(t *testing.T) {
	mk := func(needs []string) Contract {
		return Contract{
			RunID:    "r1",
			Requires: []Require{{As: "b", Capability: CapabilityAgentReason}},
			Steps: []Step{
				{ID: "one", Uses: "b", Run: []string{"true"}},
				{ID: "two", Uses: "b", Run: []string{"true"}},
				{ID: "three", Uses: "b", Run: []string{"true"}, Needs: needs},
			},
		}
	}
	for _, ok := range [][]string{nil, {}, {"one"}, {"one", "two"}} {
		if err := mk(ok).Validate(); err != nil {
			t.Fatalf("정상 needs %v 가 거절됐다: %v", ok, err)
		}
	}
	for name, bad := range map[string][]string{
		"없는 단계를 가리킨다":  {"없는것"},
		"중복이 있다":       {"one", "one"},
		"★ 자기를 가리킨다 ★": {"three"},
	} {
		if err := mk(bad).Validate(); err == nil {
			t.Fatalf("%s — 통과했다: %v", name, bad)
		}
	}
	// ★ 뒤를 가리킨다 ★ — 위상순서를 깨는 계약은 제출 시점에 막힌다.
	back := Contract{
		RunID:    "r1",
		Requires: []Require{{As: "b", Capability: CapabilityAgentReason}},
		Steps: []Step{
			{ID: "one", Uses: "b", Run: []string{"true"}, Needs: []string{"two"}},
			{ID: "two", Uses: "b", Run: []string{"true"}},
		},
	}
	if err := back.Validate(); err == nil {
		t.Fatal("★ 뒤를 가리키는 needs 가 통과했다 ★ — DAG 가 안 지켜진다")
	}
}

// ★ 기본값은 한 곳에서 채운다 ★ — 읽는 쪽이 여럿이라 각자 알게 두면 어긋난다.
func TestNeedsOf_기본값(t *testing.T) {
	steps := []Step{
		{ID: "one"},
		{ID: "two"},                      // 안 적었다 → [직전]
		{ID: "three", Needs: []string{}}, // ★ 빈 배열은 선언이다 ★ → 안 기다린다
		{ID: "four", Needs: []string{"one"}},
	}
	for i, want := range [][]string{{}, {"one"}, {}, {"one"}} {
		got := NeedsOf(steps, i)
		if len(got) != len(want) || (len(want) == 1 && got[0] != want[0]) {
			t.Fatalf("steps[%d] = %v — %v 여야 한다", i, got, want)
		}
	}
}

// ★ 분기 목적지는 형제다 ★ — 직전 단계가 아니라 분기를 낸 단계 다음이다.
//
// 이것을 빠뜨리면 to: ["full","quick"] 에서 quick 의 기본값이 [full] 이 되어
// ★ 형제가 사슬로 이어지고 ★, full 이 SKIPPED 가 되는 순간 전파가
// ★ 살아 있어야 할 가지까지 죽인다 ★. 구현이 이 빈틈을 찾았다.
func TestNeedsOf_분기_목적지는_분기_단계를_가리킨다(t *testing.T) {
	steps := []Step{
		{ID: "triage", Dispatch: &Dispatch{From: "route.next", To: []string{"full", "quick"}}},
		{ID: "full"},
		{ID: "quick"},
	}
	for i, name := range []string{"full", "quick"} {
		got := NeedsOf(steps, i+1)
		if len(got) != 1 || got[0] != "triage" {
			t.Fatalf("%s 의 기본 needs 가 %v 다 — [triage] 여야 한다 "+
				"(형제를 사슬로 이으면 전파가 산 가지를 죽인다)", name, got)
		}
	}
}

// ★ expands 검증 — 「Run 은 하나다」의 완화를 유계로 묶는다 ★ (ADR-022 §6.3)
//
// 임의 확장이 아니라 ★ 한 단계가 한 번 ★ 이다. 여러 번은 재계획(P6)이고
// 그때는 깊이 상한이 따라온다.
func TestValidate_expands(t *testing.T) {
	mk := func(steps []Step) Contract {
		return Contract{
			RunID:    "r1",
			Requires: []Require{{As: "b", Capability: CapabilityAgentReason}},
			Steps:    steps,
		}
	}
	sch := map[string]interface{}{"plan": map[string]interface{}{"type": "object"}}
	ok := []Step{{ID: "plan", Uses: "b", Agent: map[string]interface{}{},
		Out: []string{"plan"}, Schema: sch, Expands: true}}
	if err := mk(ok).Validate(); err != nil {
		t.Fatalf("정상 expands 가 거절됐다: %v", err)
	}
	// ★ 둘 이상이어도 된다 ★ (ADR-031) — 그것이 재계획이다.
	// 유한성은 계약이 아니라 ★ 시스템의 판 개수 상한 ★ 이 준다.
	two := []Step{
		{ID: "plan", Uses: "b", Agent: map[string]interface{}{},
			Out: []string{"plan"}, Schema: sch, Expands: true},
		{ID: "replan", Uses: "b", Agent: map[string]interface{}{},
			Out: []string{"plan"}, Schema: sch, Expands: true},
	}
	if err := mk(two).Validate(); err != nil {
		t.Fatalf("★ 재계획하는 계약이 거절됐다 ★: %v", err)
	}

	for name, steps := range map[string][]Step{
		"★ 스키마가 없다 ★": {
			{ID: "plan", Uses: "b", Agent: map[string]interface{}{},
				Out: []string{"plan"}, Expands: true},
		},
		"산출물이 없다": {
			{ID: "plan", Uses: "b", Agent: map[string]interface{}{}, Expands: true},
		},
		"산출물이 둘이다": {
			{ID: "plan", Uses: "b", Agent: map[string]interface{}{},
				Out: []string{"plan", "other"}, Schema: sch, Expands: true},
		},
	} {
		if err := mk(steps).Validate(); err == nil {
			t.Fatalf("%s — 통과했다", name)
		}
	}
}

// ★ 동일성은 우리가 정의하지 않는다 ★ (ADR-023 §6.5.2)
//
// 그 시스템이 「하나의 변경」이라 부르는 것의 식별자를 받아 적는다.
// ⇒ ★ patchset 2 와 3 은 같은 Work 다 ★ — 그래야 "지난번에 이 지적을 했는데
// 안 고쳤다" 가 성립한다.
func TestWorkKey_같은_변경의_패치셋들은_같은_Work다(t *testing.T) {
	ps2 := Work{System: "gerrit", ChangeID: "12345", Patchset: 2, PatchRev: "aaa"}
	ps3 := Work{System: "gerrit", ChangeID: "12345", Patchset: 3, PatchRev: "bbb"}
	if ps2.Key() != ps3.Key() {
		t.Fatalf("★ 패치셋이 Work 를 가른다 ★: %q vs %q — "+
			"Work 1:N Run 이 성립하지 않는다", ps2.Key(), ps3.Key())
	}
	if ps2.Key() != "gerrit:12345" {
		t.Fatalf("유도된 키가 %q 다", ps2.Key())
	}
	// ★ 다른 change 는 다른 Work ★ — abandon 후 새로 올린 경우가 여기다.
	other := Work{System: "gerrit", ChangeID: "99999", Patchset: 1}
	if other.Key() == ps2.Key() {
		t.Fatal("다른 change 가 같은 Work 가 됐다")
	}
	// ★ 적으면 그것을 쓴다 ★ — 그 시스템의 변경 단위가 change_id 와 다를 때.
	explicit := Work{System: "gerrit", ChangeID: "12345",
		ID: &WorkID{System: "gerrit", ChangeID: "브랜치별-777"}}
	if explicit.Key() != "gerrit:브랜치별-777" {
		t.Fatalf("적어준 키를 안 썼다: %q", explicit.Key())
	}
	// work 를 안 채운 계약(시험용 계약들)은 키가 없다 — "" 와 "없다" 를 섞지 않는다.
	if (Work{}).Key() != "" {
		t.Fatal("빈 Work 가 키를 만들었다")
	}
}

// ★ 시야의 검증 — 모르는 값을 조용히 무시하지 않는다 ★ (ADR-023 §6.4)
func TestValidate_시야(t *testing.T) {
	mk := func(f func(*Contract)) Contract {
		c := Contract{
			RunID:    "r1",
			Requires: []Require{{As: "b", Capability: CapabilityAgentReason}},
			Steps: []Step{
				{ID: "one", Uses: "b", Run: []string{"true"}, Out: []string{"one"}},
				{ID: "two", Uses: "b", Run: []string{"true"}},
			},
		}
		f(&c)
		return c
	}
	if err := mk(func(c *Contract) { c.Ledger = &Ledger{Scope: ScopeWork} }).Validate(); err != nil {
		t.Fatalf("scope work 가 거절됐다: %v", err)
	}
	if err := mk(func(c *Contract) { c.Steps[1].See = &See{Ledger: SeeList} }).Validate(); err != nil {
		t.Fatalf("see list 가 거절됐다: %v", err)
	}
	// in.from 이 실존 산출물을 가리키면 통과한다.
	if err := mk(func(c *Contract) {
		c.Steps[1].In = map[string]interface{}{"from": []interface{}{"one"}}
	}).Validate(); err != nil {
		t.Fatalf("정상 in.from 이 거절됐다: %v", err)
	}

	for name, f := range map[string]func(*Contract){
		"★ 모르는 scope ★": func(c *Contract) { c.Ledger = &Ledger{Scope: "global"} },
		"★ 모르는 see ★":   func(c *Contract) { c.Steps[1].See = &See{Ledger: "all"} },
		"★ see.from 은 아직 없다 ★": func(c *Contract) {
			c.Steps[1].See = &See{From: []string{"one"}}
		},
		"★ 밑줄은 예약이다 ★": func(c *Contract) { c.Steps[0].Out = []string{"_ledger.json"} },
		"★ 아무도 안 내는 것을 in.from 에 ★": func(c *Contract) {
			c.Steps[1].In = map[string]interface{}{"from": []interface{}{"없는것"}}
		},
	} {
		if err := mk(f).Validate(); err == nil {
			t.Fatalf("%s — 통과했다", name)
		}
	}
}

// ★ release 검증 — 폭이 열리면서 조건이 강해졌다 ★ (ADR-022 §7.4 + ADR-023)
//
// 순차였다면 "뒤에서 안 쓰면 된다" 로 족했다. 병렬에서는 ★ 순서가 정해지지 않은
// 단계가 동시에 돌 수 있고 ★, 그 단계가 놓아버린 역할을 쓰면 실행 중에 임대가
// 사라진다. ⇒ ★ 그 역할을 쓰는 모든 단계가 놓는 단계의 조상이어야 한다 ★.
func TestValidate_release(t *testing.T) {
	mk := func(steps []Step) Contract {
		return Contract{
			RunID: "r1",
			Requires: []Require{
				{As: "b", Capability: CapabilityAgentReason},
				{As: "c", Capability: CapabilityAgentReason}},
			Steps: steps,
		}
	}
	run := func(id, uses string, needs []string, release ...string) Step {
		return Step{ID: id, Uses: uses, Run: []string{"true"}, Needs: needs, Release: release}
	}

	// ★ 조상이면 통과 ★ — first 가 second 보다 확실히 앞선다.
	if err := mk([]Step{
		run("first", "b", nil),
		run("second", "c", nil, "b"),
	}).Validate(); err != nil {
		t.Fatalf("정상 release 가 거절됐다: %v", err)
	}

	// ★ 뒤에서 쓰면 거절 ★
	if err := mk([]Step{
		run("drop", "c", nil, "b"),
		run("later", "b", nil),
	}).Validate(); err == nil {
		t.Fatal("★ 놓은 자원을 뒤에서 쓰는 계약이 통과했다 ★")
	}

	// ★ 순서가 안 정해졌으면 거절 ★ — 동시에 돌 수 있으므로 조상이 아니다.
	// sibling 은 needs 가 비어 있어 drop 과 순서 관계가 없다.
	if err := mk([]Step{
		run("gate", "c", nil),
		run("sibling", "b", []string{}),
		run("drop", "c", []string{"gate"}, "b"),
	}).Validate(); err == nil {
		t.Fatal("★ 동시에 돌 수 있는 단계의 자원을 놓는 계약이 통과했다 ★ — " +
			"실행 중에 임대가 사라진다")
	}

	// 없는 역할 · 중복
	if err := mk([]Step{run("one", "b", nil, "없는역할")}).Validate(); err == nil {
		t.Fatal("없는 역할을 놓는 계약이 통과했다")
	}
	if err := mk([]Step{
		run("first", "b", nil),
		run("second", "c", nil, "b", "b"),
	}).Validate(); err == nil {
		t.Fatal("중복 release 가 통과했다")
	}
}

// ★ acquire 검증 — 잡기 전에는 못 쓴다 ★ (ADR-022 §7.5 · ADR-024)
func TestValidate_acquire(t *testing.T) {
	acq := func(as string) Step {
		return Step{ID: "try", Acquire: &Acquire{
			Want:     &Require{As: as, Capability: CapabilityAgentReason},
			Acquired: "use", Unavailable: "other"}}
	}
	mk := func(steps []Step) Contract {
		return Contract{
			RunID:    "r1",
			Requires: []Require{{As: "b", Capability: CapabilityAgentReason}},
			Steps:    steps,
		}
	}
	// ★ 잡은 뒤에 쓰면 통과 ★
	if err := mk([]Step{
		acq("board"),
		{ID: "use", Uses: "board", Run: []string{"true"}},
		{ID: "other", Uses: "b", Run: []string{"true"}},
	}).Validate(); err != nil {
		t.Fatalf("정상 acquire 가 거절됐다: %v", err)
	}

	full := func(a *Acquire) []Step {
		return []Step{
			{ID: "try", Acquire: a},
			{ID: "use", Uses: "board", Run: []string{"true"}},
			{ID: "other", Uses: "b", Run: []string{"true"}},
		}
	}
	want := func(as string) *Require { return &Require{As: as, Capability: CapabilityAgentReason} }

	for name, steps := range map[string][]Step{
		"★ 잡기 전에 쓴다 ★": {
			{ID: "use", Uses: "board", Run: []string{"true"}},
			acq("board"),
			{ID: "other", Uses: "b", Run: []string{"true"}},
		},
		"★ requires 에 이미 있다 ★": full(&Acquire{
			Want: want("b"), Acquired: "use", Unavailable: "other"}),
		"★ 목적지가 없다 ★": full(&Acquire{Want: want("board")}),
		"★ 두 목적지가 같다 ★": full(&Acquire{
			Want: want("board"), Acquired: "use", Unavailable: "use"}),
		"★ 없는 단계를 가리킨다 ★": full(&Acquire{
			Want: want("board"), Acquired: "use", Unavailable: "없는것"}),
		"★ uses 가 있다 ★": {
			{ID: "try", Uses: "b", Acquire: &Acquire{
				Want: want("board"), Acquired: "use", Unavailable: "other"}},
			{ID: "use", Uses: "board", Run: []string{"true"}},
			{ID: "other", Uses: "b", Run: []string{"true"}},
		},
		"★ 모르는 capability ★": full(&Acquire{
			Want:     &Require{As: "board", Capability: "board.flash"},
			Acquired: "use", Unavailable: "other"}),
	} {
		if err := mk(steps).Validate(); err == nil {
			t.Fatalf("%s — 통과했다", name)
		}
	}
}

// ★ loop 검증 — 뒤로 가는 유일한 간선 ★ (ADR-026)
func TestValidate_loop(t *testing.T) {
	zero := 0
	mk := func(f func([]Step) []Step) Contract {
		steps := []Step{
			{ID: "write", Uses: "b", Run: []string{"true"}},
			{ID: "check", Uses: "b", Run: []string{"true"}},
		}
		return Contract{
			RunID:    "r1",
			Requires: []Require{{As: "b", Capability: CapabilityAgentReason}},
			Steps:    f(steps),
		}
	}
	ok := mk(func(s []Step) []Step {
		s[1].Loop = &Loop{BackTo: "write", Max: 3, Until: Condition{ExitCode: &zero}}
		return s
	})
	if err := ok.Validate(); err != nil {
		t.Fatalf("정상 loop 이 거절됐다: %v", err)
	}

	for name, f := range map[string]func([]Step) []Step{
		"★ 앞으로 간다 ★": func(s []Step) []Step {
			s[0].Loop = &Loop{BackTo: "check", Max: 3, Until: Condition{ExitCode: &zero}}
			return s
		},
		"★ 없는 단계로 간다 ★": func(s []Step) []Step {
			s[1].Loop = &Loop{BackTo: "없는것", Max: 3, Until: Condition{ExitCode: &zero}}
			return s
		},
		"★ max 가 1 이다 ★": func(s []Step) []Step {
			s[1].Loop = &Loop{BackTo: "write", Max: 1, Until: Condition{ExitCode: &zero}}
			return s
		},
		"★ until 이 비었다 ★": func(s []Step) []Step {
			s[1].Loop = &Loop{BackTo: "write", Max: 3}
			return s
		},
		"★ until 에 step 을 적었다 ★": func(s []Step) []Step {
			s[1].Loop = &Loop{BackTo: "write", Max: 3,
				Until: Condition{Step: "write", ExitCode: &zero}}
			return s
		},
		"★ validate_with 와 함께 ★": func(s []Step) []Step {
			s[0].ValidateWith = "check"
			s[1].Loop = &Loop{BackTo: "write", Max: 3, Until: Condition{ExitCode: &zero}}
			s[1].ValidateWith = "write"
			return s
		},
		"★ 구간 안에서 놓는다 ★": func(s []Step) []Step {
			s[0].Release = []string{"b"}
			s[1].Loop = &Loop{BackTo: "write", Max: 3, Until: Condition{ExitCode: &zero}}
			return s
		},
		"★ agent 단계에 exit_code ★": func(s []Step) []Step {
			s[1] = Step{ID: "check", Uses: "b", Agent: map[string]interface{}{},
				Loop: &Loop{BackTo: "write", Max: 3, Until: Condition{ExitCode: &zero}}}
			return s
		},
	} {
		if err := mk(f).Validate(); err == nil {
			t.Fatalf("%s — 통과했다", name)
		}
	}

	// ★ 중첩은 아직 없다 ★
	nested := Contract{
		RunID:    "r1",
		Requires: []Require{{As: "b", Capability: CapabilityAgentReason}},
		Steps: []Step{
			{ID: "a", Uses: "b", Run: []string{"true"}},
			{ID: "inner", Uses: "b", Run: []string{"true"},
				Loop: &Loop{BackTo: "a", Max: 2, Until: Condition{ExitCode: &zero}}},
			{ID: "outer", Uses: "b", Run: []string{"true"},
				Loop: &Loop{BackTo: "a", Max: 2, Until: Condition{ExitCode: &zero}}},
		},
	}
	if err := nested.Validate(); err == nil {
		t.Fatal("★ 중첩 loop 이 통과했다 ★")
	}
}

// ★ 시연 계약이 실제로 유효한가 ★
//
// testdata/demo.json 은 ★ enode-design/protocol/run-contract.md §2.0 에서
// 그대로 뽑은 것 ★ 이다. 정본에 적힌 계약이 코드가 받는 계약과 어긋나면
// ★ 시연 당일에 알게 된다 ★ — 그것을 여기서 막는다.
//
// 이 시험이 깨지면 둘 중 하나다: 문서가 낡았거나, 검증이 문서를 배신했거나.
// ★ 어느 쪽이든 고쳐야 한다 ★.
func TestValidate_시연계약(t *testing.T) {
	raw, err := os.ReadFile("testdata/demo.json")
	if err != nil {
		t.Fatal(err)
	}
	var c Contract
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatalf("★ 정본의 계약이 파싱조차 안 된다 ★: %v", err)
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("★ 정본의 계약이 거절된다 ★: %v", err)
	}

	// ★ 병렬 둘이 실제로 드러나는가 ★ — 이 계약의 값이 거기 있다.
	needs := map[string][]string{}
	for i, st := range c.Steps {
		needs[st.ID] = NeedsOf(c.Steps, i)
	}
	if len(needs["hypothesis"]) != 0 || len(needs["baseline_build"]) != 0 {
		t.Fatalf("★ 시작점 둘이 안 갈렸다 ★: %v %v",
			needs["hypothesis"], needs["baseline_build"])
	}
	if len(needs["write_test"]) != 2 {
		t.Fatalf("★ 합류가 한쪽만 기다린다 ★: %v", needs["write_test"])
	}
	// patch_build 는 parent_build 만 기다린다 — parent_observe 와 ★ 동시에 돈다 ★.
	// ★ write_test 가 아니다 ★ — parent_build 에 loop 이 있어서, write_test 만
	// 기다리면 ★ 반복 도중의 test_source ★ 를 쓴다.
	if len(needs["patch_build"]) != 1 || needs["patch_build"][0] != "parent_build" {
		t.Fatalf("★ 패치 빌드의 의존이 틀렸다 ★: %v", needs["patch_build"])
	}
	if len(needs["parent_observe"]) != 1 || needs["parent_observe"][0] != "parent_build" {
		t.Fatalf("★ 부모 관찰의 의존이 틀렸다 ★: %v", needs["parent_observe"])
	}
	// patch_observe 는 ★ 보드가 하나라는 사실 ★ 을 계약에 적어둔 자리다.
	if len(needs["patch_observe"]) != 2 {
		t.Fatalf("★ 두 관찰의 순서가 안 적혔다 ★: %v — "+
			"보드가 둘이 되는 날 차분이 깨진다", needs["patch_observe"])
	}
}

// ★ ask 검증 — 질문의 형태가 곧 스키마이고, 평면 폼만 허용한다 ★ (ADR-032)
func TestValidate_ask(t *testing.T) {
	form := map[string]interface{}{
		"type": "object", "required": []interface{}{"verdict"},
		"properties": map[string]interface{}{
			"verdict": map[string]interface{}{"enum": []interface{}{"approve", "reject"}},
			"note":    map[string]interface{}{"type": "string"}}}
	mk := func(f func(*Step)) Contract {
		st := Step{ID: "gate", Ask: &Ask{Prompt: "승인?"},
			Out: []string{"decision"}, Schema: map[string]interface{}{"decision": form}}
		f(&st)
		return Contract{
			RunID:    "r1",
			Requires: []Require{{As: "b", Capability: CapabilityAgentReason}},
			Steps:    []Step{st, {ID: "next", Uses: "b", Run: []string{"true"}}},
		}
	}
	if err := mk(func(*Step) {}).Validate(); err != nil {
		t.Fatalf("정상 ask 가 거절됐다: %v", err)
	}
	if err := mk(func(st *Step) {
		st.Ask.Timeout = &AskTimeout{After: "72h", Then: "fail"}
	}).Validate(); err != nil {
		t.Fatalf("기한 있는 ask 가 거절됐다: %v", err)
	}

	for name, f := range map[string]func(*Step){
		"★ uses 가 있다 ★":   func(st *Step) { st.Uses = "b" },
		"★ prompt 가 없다 ★": func(st *Step) { st.Ask.Prompt = "" },
		"★ 스키마가 없다 ★":     func(st *Step) { st.Schema = nil },
		"★ 중첩 객체 ★": func(st *Step) {
			st.Schema = map[string]interface{}{"decision": map[string]interface{}{
				"type": "object", "properties": map[string]interface{}{
					"inner": map[string]interface{}{"type": "object"}}}}
		},
		"★ 배열 ★": func(st *Step) {
			st.Schema = map[string]interface{}{"decision": map[string]interface{}{
				"type": "object", "properties": map[string]interface{}{
					"list": map[string]interface{}{"type": "array"}}}}
		},
		"★ then 이 default — 아직 없다 ★": func(st *Step) {
			st.Ask.Timeout = &AskTimeout{After: "1h", Then: "default"}
		},
		"★ 기한을 못 읽는다 ★": func(st *Step) {
			st.Ask.Timeout = &AskTimeout{After: "사흘", Then: "fail"}
		},
	} {
		if err := mk(f).Validate(); err == nil {
			t.Fatalf("%s — 통과했다", name)
		}
	}
}

// ★ adopts 검증 — 채택의 어휘를 못 박는다 ★ (ADR-033)
func TestValidate_adopts(t *testing.T) {
	planSch := map[string]interface{}{"plan": map[string]interface{}{"type": "object"}}
	verdictSch := func(opts ...string) map[string]interface{} {
		e := make([]interface{}, len(opts))
		for i, o := range opts {
			e[i] = o
		}
		return map[string]interface{}{"decision": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"verdict": map[string]interface{}{"enum": e}}}}
	}
	mk := func(adopts string, sch map[string]interface{}) Contract {
		return Contract{
			RunID:    "r1",
			Requires: []Require{{As: "b", Capability: CapabilityAgentReason}},
			Steps: []Step{
				{ID: "plan", Uses: "b", Agent: map[string]interface{}{},
					Out: []string{"plan"}, Schema: planSch, Expands: true},
				{ID: "gate", Ask: &Ask{Prompt: "?", Adopts: adopts},
					Out: []string{"decision"}, Schema: sch},
			},
		}
	}
	if err := mk("plan", verdictSch("approve", "reject")).Validate(); err != nil {
		t.Fatalf("정상 adopts 가 거절됐다: %v", err)
	}
	if err := mk("없는것", verdictSch("approve", "reject")).Validate(); err == nil {
		t.Fatal("없는 단계를 adopts 하는데 통과했다")
	}
	if err := mk("gate", verdictSch("approve", "reject")).Validate(); err == nil {
		t.Fatal("expands 아닌 단계를 adopts 하는데 통과했다")
	}
	// ★ verdict 어휘가 없으면 채택이 기계적으로 못 갈린다 ★
	if err := mk("plan", verdictSch("yes", "no")).Validate(); err == nil {
		t.Fatal("approve/reject 없는 스키마가 통과했다")
	}
}
