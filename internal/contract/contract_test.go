package contract

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
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
		t.Fatalf("the structural keys did not land: %+v", r)
	}
	if r.Attrs["arch"] != "armv7" || r.Attrs["repo"] != "gerrit.corp/kernel/linux" {
		t.Fatalf("the attributes did not land: %+v", r.Attrs)
	}
	if len(r.Attrs) != 2 {
		t.Fatalf("a structural key leaked into the attributes: %+v", r.Attrs)
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
		t.Fatalf("the round trip broke: %+v → %+v", r, back)
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
			t.Fatalf("accepted a non-string attribute: %s → %+v", in, r)
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
		{"agent step", Step{ID: "h", Agent: map[string]interface{}{"model": "x"}}, KindAgent, false},
		{"command step", Step{ID: "b", Run: []string{"make", "-j8"}}, KindRun, false},
		{"both is wrong", Step{ID: "x", Agent: map[string]interface{}{}, Run: []string{"make"}}, KindUnknown, true},
		{"neither is wrong", Step{ID: "y"}, KindUnknown, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := c.s.Kind()
			if (err != nil) != c.err {
				t.Fatalf("err=%v, want err=%v", err, c.err)
			}
			if got != c.want {
				t.Fatalf("kind=%v, want %v", got, c.want)
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
		t.Fatalf("a valid contract was rejected: %v", err)
	}

	cases := []struct {
		name string
		mut  func(*Contract)
		want error
	}{
		{"no run_id", func(c *Contract) { c.RunID = "" }, ErrNoRunID},
		{"requires empty", func(c *Contract) { c.Requires = nil }, ErrNoRequires},
		{"steps empty", func(c *Contract) { c.Steps = nil }, ErrNoSteps},
		// ADR-019 결정 1 — 어휘는 agent.reason 하나뿐이다.
		{"old capability", func(c *Contract) { c.Requires[0].Capability = "build.linux" }, ErrUnknownCap},
		{"duplicate as", func(c *Contract) { c.Requires[1].As = "brain" }, ErrDupAs},
		{"duplicate step id", func(c *Contract) { c.Steps[1].ID = "hypothesis" }, ErrDupStepID},
		// ADR-004 를 지키는 한 줄 — claude 는 헛소리를 하고도 0 으로 끝난다.
		{"exit_code on an agent step", func(c *Contract) {
			c.SuccessWhen[0].ExitCode = zero()
		}, ErrExitOnAgent},
		{"points at a step that does not exist", func(c *Contract) { c.SuccessWhen[0].Step = "nope" }, ErrCondUnknownID},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := base()
			tc.mut(&c)
			err := c.Validate()
			if err == nil {
				t.Fatalf("it passed")
			}
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("err=%v, want %v", err, tc.want)
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
		{"a requirement with no attributes takes anyone", map[string]string{}, true},
		{"subset", map[string]string{"arch": "armv7"}, true},
		{"multi-key subset", map[string]string{"arch": "armv7", "repo": "gerrit.corp/kernel/linux"}, true},
		{"a different value does not match", map[string]string{"arch": "x86_64"}, false},
		{"a missing key does not match", map[string]string{"machine": "qemu"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := Require{Capability: CapabilityAgentReason, Attrs: c.attrs}
			if got := node.Satisfies(r); got != c.want {
				t.Fatalf("got=%v, want %v", got, c.want)
			}
		})
	}
}

// orchestration 은 평범한 Run 에 안 끌려간다 (ADR-022 §5.2)
//
// 어휘를 나눈 목적이 배제다. capability 는 완전일치라 서로를 안 만족시킨다.
func TestCapability_TheTwoWordsExcludeEachOther(t *testing.T) {
	orch := Capability{Capability: CapabilityOrchestration,
		Attrs: map[string]string{"harness": "claude"}}
	agent := Capability{Capability: CapabilityAgentReason,
		Attrs: map[string]string{"harness": "claude"}}

	// 핵심 — 평범한 Run 이 오케스트레이터를 못 잡는다
	if orch.Satisfies(Require{Capability: CapabilityAgentReason}) {
		t.Fatal("an ordinary run grabs the orchestrator")
	}
	// 속성까지 같아도 마찬가지다 — 부분집합 매칭이 capability 를 못 넘는다
	if orch.Satisfies(Require{Capability: CapabilityAgentReason,
		Attrs: map[string]string{"harness": "claude"}}) {
		t.Fatal("attribute matching crossed the capability boundary")
	}
	// 반대도 성립한다
	if agent.Satisfies(Require{Capability: CapabilityOrchestration}) {
		t.Fatal("an ordinary node satisfied an orchestration requirement")
	}
	// 제 짝은 만족시킨다
	if !orch.Satisfies(Require{Capability: CapabilityOrchestration,
		Attrs: map[string]string{"harness": "claude"}}) {
		t.Fatal("it fails to satisfy its own match")
	}
}

// 모르는 capability 는 거절한다 — 열린 어휘가 아니다.
// 오타가 조용히 통과하면 계약 저자가 422 대신 "후보 없음" 을 보게 된다.
func TestValidate_RejectsAnUnknownCapability(t *testing.T) {
	c := Contract{
		RunID:    "r1",
		Requires: []Require{{As: "x", Capability: "orchestraton"}}, // typo
		Steps:    []Step{{ID: "s", Uses: "x", Run: []string{"true"}}},
	}
	if err := c.Validate(); err == nil {
		t.Fatal("the typo passed")
	}
	c.Requires[0].Capability = CapabilityOrchestration
	if err := c.Validate(); err != nil {
		t.Fatalf("a valid keyword was rejected: %v", err)
	}
}

// dispatch 검증 — DAG 를 정적으로 확인한다 (ADR-022 §7.2)
//
// 뒤로 못 가면 종료가 계약 검증 단계에서 보장된다. 실행 중에 무한 루프를
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
		t.Fatalf("a valid dispatch was rejected: %v", err)
	}

	for name, d := range map[string]*Dispatch{
		"from is empty":                         {To: []string{"full", "quick"}},
		"points at an artifact nobody produces": {From: "nosuch.next", To: []string{"full", "quick"}},
		"only one branch":                       {From: "route.next", To: []string{"full"}},
		"points at a step that does not exist":  {From: "route.next", To: []string{"full", "nosuch"}},
		"has a duplicate":                       {From: "route.next", To: []string{"full", "full"}},
		"points at itself":                      {From: "route.next", To: []string{"full", "triage"}},
	} {
		if err := mk(d).Validate(); err == nil {
			t.Fatalf("%s — it passed: %+v", name, d)
		}
	}
}

// needs 검증 — dispatch 와 같은 것을 지킨다 (ADR-023 §4.3)
//
// 간선이 전부 뒤를 향하면 그래프가 DAG 이고, 그래서 종료가 제출 시점에
// 정적으로 보장된다. 뒤로 가야 하는 것은 의존이 아니라 반복이다.
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
			t.Fatalf("valid needs %v were rejected: %v", ok, err)
		}
	}
	for name, bad := range map[string][]string{
		"points at a step that does not exist": {"nosuch"},
		"has a duplicate":                      {"one", "one"},
		"points at itself":                     {"three"},
	} {
		if err := mk(bad).Validate(); err == nil {
			t.Fatalf("%s — it passed: %v", name, bad)
		}
	}
	// 뒤를 가리킨다 — 위상순서를 깨는 계약은 제출 시점에 막힌다.
	back := Contract{
		RunID:    "r1",
		Requires: []Require{{As: "b", Capability: CapabilityAgentReason}},
		Steps: []Step{
			{ID: "one", Uses: "b", Run: []string{"true"}, Needs: []string{"two"}},
			{ID: "two", Uses: "b", Run: []string{"true"}},
		},
	}
	if err := back.Validate(); err == nil {
		t.Fatal("a backward needs passed — the DAG is not held")
	}
}

// 기본값은 한 곳에서 채운다 — 읽는 쪽이 여럿이라 각자 알게 두면 어긋난다.
func TestNeedsOf_Defaults(t *testing.T) {
	steps := []Step{
		{ID: "one"},
		{ID: "two"},                      // unwritten → [previous]
		{ID: "three", Needs: []string{}}, // an empty array is a declaration → waits for nothing
		{ID: "four", Needs: []string{"one"}},
	}
	for i, want := range [][]string{{}, {"one"}, {}, {"one"}} {
		got := NeedsOf(steps, i)
		if len(got) != len(want) || (len(want) == 1 && got[0] != want[0]) {
			t.Fatalf("steps[%d] = %v — want %v", i, got, want)
		}
	}
}

// 분기 목적지는 형제다 — 직전 단계가 아니라 분기를 낸 단계 다음이다.
//
// 이것을 빠뜨리면 to: ["full","quick"] 에서 quick 의 기본값이 [full] 이 되어
// 형제가 사슬로 이어지고, full 이 SKIPPED 가 되는 순간 전파가
// 살아 있어야 할 가지까지 죽인다. 구현이 이 빈틈을 찾았다.
func TestNeedsOf_BranchDestinationsPointAtTheBranchStep(t *testing.T) {
	steps := []Step{
		{ID: "triage", Dispatch: &Dispatch{From: "route.next", To: []string{"full", "quick"}}},
		{ID: "full"},
		{ID: "quick"},
	}
	for i, name := range []string{"full", "quick"} {
		got := NeedsOf(steps, i+1)
		if len(got) != 1 || got[0] != "triage" {
			t.Fatalf("default needs of %s are %v — want [triage] "+
				"(chaining siblings lets propagation kill a live branch)", name, got)
		}
	}
}

// expands 검증 — 「Run 은 하나다」의 완화를 유계로 묶는다 (ADR-022 §6.3)
//
// 임의 확장이 아니라 한 단계가 한 번이다. 여러 번은 재계획(P6)이고
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
		t.Fatalf("a valid expands was rejected: %v", err)
	}
	// 둘 이상이어도 된다 (ADR-031) — 그것이 재계획이다.
	// 유한성은 계약이 아니라 시스템의 판 개수 상한이 준다.
	two := []Step{
		{ID: "plan", Uses: "b", Agent: map[string]interface{}{},
			Out: []string{"plan"}, Schema: sch, Expands: true},
		{ID: "replan", Uses: "b", Agent: map[string]interface{}{},
			Out: []string{"plan"}, Schema: sch, Expands: true},
	}
	if err := mk(two).Validate(); err != nil {
		t.Fatalf("a replanning contract was rejected: %v", err)
	}

	for name, steps := range map[string][]Step{
		"no schema": {
			{ID: "plan", Uses: "b", Agent: map[string]interface{}{},
				Out: []string{"plan"}, Expands: true},
		},
		"no artifact": {
			{ID: "plan", Uses: "b", Agent: map[string]interface{}{}, Expands: true},
		},
		"two artifacts": {
			{ID: "plan", Uses: "b", Agent: map[string]interface{}{},
				Out: []string{"plan", "other"}, Schema: sch, Expands: true},
		},
	} {
		if err := mk(steps).Validate(); err == nil {
			t.Fatalf("%s — it passed", name)
		}
	}
}

// 동일성은 우리가 정의하지 않는다 (ADR-023 §6.5.2)
//
// 그 시스템이 「하나의 변경」이라 부르는 것의 식별자를 받아 적는다.
// ⇒ patchset 2 와 3 은 같은 Work 다 — 그래야 "지난번에 이 지적을 했는데
// 안 고쳤다" 가 성립한다.
func TestWorkKey_PatchsetsOfOneChangeAreOneWork(t *testing.T) {
	ps2 := Work{System: "gerrit", ChangeID: "12345", Patchset: 2, PatchRev: "aaa"}
	ps3 := Work{System: "gerrit", ChangeID: "12345", Patchset: 3, PatchRev: "bbb"}
	if ps2.Key() != ps3.Key() {
		t.Fatalf("patchsets split the work: %q vs %q — "+
			"work 1:N run does not hold", ps2.Key(), ps3.Key())
	}
	if ps2.Key() != "gerrit:12345" {
		t.Fatalf("the derived key is %q", ps2.Key())
	}
	// 다른 change 는 다른 Work — abandon 후 새로 올린 경우가 여기다.
	other := Work{System: "gerrit", ChangeID: "99999", Patchset: 1}
	if other.Key() == ps2.Key() {
		t.Fatal("two different changes became one work")
	}
	// 적으면 그것을 쓴다 — 그 시스템의 변경 단위가 change_id 와 다를 때.
	explicit := Work{System: "gerrit", ChangeID: "12345",
		ID: &WorkID{System: "gerrit", ChangeID: "branchy-777"}}
	if explicit.Key() != "gerrit:branchy-777" {
		t.Fatalf("the explicit key was ignored: %q", explicit.Key())
	}
	// work 를 안 채운 계약(시험용 계약들)은 키가 없다 — "" 와 "없다" 를 섞지 않는다.
	if (Work{}).Key() != "" {
		t.Fatal("an empty work produced a key")
	}
}

// 시야의 검증 — 모르는 값을 조용히 무시하지 않는다 (ADR-023 §6.4)
func TestValidate_Visibility(t *testing.T) {
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
		t.Fatalf("scope work was rejected: %v", err)
	}
	if err := mk(func(c *Contract) { c.Steps[1].See = &See{Ledger: SeeList} }).Validate(); err != nil {
		t.Fatalf("see list was rejected: %v", err)
	}
	// in.from 이 실존 산출물을 가리키면 통과한다.
	if err := mk(func(c *Contract) {
		c.Steps[1].In = map[string]interface{}{"from": []interface{}{"one"}}
	}).Validate(); err != nil {
		t.Fatalf("a valid in.from was rejected: %v", err)
	}

	for name, f := range map[string]func(*Contract){
		"unknown scope": func(c *Contract) { c.Ledger = &Ledger{Scope: "global"} },
		"unknown see":   func(c *Contract) { c.Steps[1].See = &See{Ledger: "all"} },
		"see.from does not exist yet": func(c *Contract) {
			c.Steps[1].See = &See{From: []string{"one"}}
		},
		"the underscore is reserved": func(c *Contract) { c.Steps[0].Out = []string{"_ledger.json"} },
		"in.from names something nobody produces": func(c *Contract) {
			c.Steps[1].In = map[string]interface{}{"from": []interface{}{"nosuch"}}
		},
	} {
		if err := mk(f).Validate(); err == nil {
			t.Fatalf("%s — it passed", name)
		}
	}
}

// release 검증 — 폭이 열리면서 조건이 강해졌다 (ADR-022 §7.4 + ADR-023)
//
// 순차였다면 "뒤에서 안 쓰면 된다" 로 족했다. 병렬에서는 순서가 정해지지 않은
// 단계가 동시에 돌 수 있고, 그 단계가 놓아버린 역할을 쓰면 실행 중에 임대가
// 사라진다. ⇒ 그 역할을 쓰는 모든 단계가 놓는 단계의 조상이어야 한다.
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

	// 조상이면 통과 — first 가 second 보다 확실히 앞선다.
	if err := mk([]Step{
		run("first", "b", nil),
		run("second", "c", nil, "b"),
	}).Validate(); err != nil {
		t.Fatalf("a valid release was rejected: %v", err)
	}

	// 뒤에서 쓰면 거절
	if err := mk([]Step{
		run("drop", "c", nil, "b"),
		run("later", "b", nil),
	}).Validate(); err == nil {
		t.Fatal("a contract using a released resource later passed")
	}

	// 순서가 안 정해졌으면 거절 — 동시에 돌 수 있으므로 조상이 아니다.
	// sibling 은 needs 가 비어 있어 drop 과 순서 관계가 없다.
	if err := mk([]Step{
		run("gate", "c", nil),
		run("sibling", "b", []string{}),
		run("drop", "c", []string{"gate"}, "b"),
	}).Validate(); err == nil {
		t.Fatal("a contract releasing a resource a concurrent step uses passed — " +
			"the lease disappears mid-run")
	}

	// 없는 역할 · 중복
	if err := mk([]Step{run("one", "b", nil, "norole")}).Validate(); err == nil {
		t.Fatal("a contract releasing a role that does not exist passed")
	}
	if err := mk([]Step{
		run("first", "b", nil),
		run("second", "c", nil, "b", "b"),
	}).Validate(); err == nil {
		t.Fatal("a duplicate release passed")
	}
}

// acquire 검증 — 잡기 전에는 못 쓴다 (ADR-022 §7.5 · ADR-024)
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
	// 잡은 뒤에 쓰면 통과
	if err := mk([]Step{
		acq("board"),
		{ID: "use", Uses: "board", Run: []string{"true"}},
		{ID: "other", Uses: "b", Run: []string{"true"}},
	}).Validate(); err != nil {
		t.Fatalf("a valid acquire was rejected: %v", err)
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
		"used before it is acquired": {
			{ID: "use", Uses: "board", Run: []string{"true"}},
			acq("board"),
			{ID: "other", Uses: "b", Run: []string{"true"}},
		},
		"already in requires": full(&Acquire{
			Want: want("b"), Acquired: "use", Unavailable: "other"}),
		"no destination": full(&Acquire{Want: want("board")}),
		"both destinations are the same": full(&Acquire{
			Want: want("board"), Acquired: "use", Unavailable: "use"}),
		"points at a step that does not exist": full(&Acquire{
			Want: want("board"), Acquired: "use", Unavailable: "nosuch"}),
		"uses is present": {
			{ID: "try", Uses: "b", Acquire: &Acquire{
				Want: want("board"), Acquired: "use", Unavailable: "other"}},
			{ID: "use", Uses: "board", Run: []string{"true"}},
			{ID: "other", Uses: "b", Run: []string{"true"}},
		},
		"unknown capability": full(&Acquire{
			Want:     &Require{As: "board", Capability: "board.flash"},
			Acquired: "use", Unavailable: "other"}),
	} {
		if err := mk(steps).Validate(); err == nil {
			t.Fatalf("%s — it passed", name)
		}
	}
}

// loop 검증 — 뒤로 가는 유일한 간선 (ADR-026)
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
		t.Fatalf("a valid loop was rejected: %v", err)
	}

	for name, f := range map[string]func([]Step) []Step{
		"goes forward": func(s []Step) []Step {
			s[0].Loop = &Loop{BackTo: "check", Max: 3, Until: Condition{ExitCode: &zero}}
			return s
		},
		"goes to a step that does not exist": func(s []Step) []Step {
			s[1].Loop = &Loop{BackTo: "nosuch", Max: 3, Until: Condition{ExitCode: &zero}}
			return s
		},
		"max is 1": func(s []Step) []Step {
			s[1].Loop = &Loop{BackTo: "write", Max: 1, Until: Condition{ExitCode: &zero}}
			return s
		},
		"until is empty": func(s []Step) []Step {
			s[1].Loop = &Loop{BackTo: "write", Max: 3}
			return s
		},
		"until names a step": func(s []Step) []Step {
			s[1].Loop = &Loop{BackTo: "write", Max: 3,
				Until: Condition{Step: "write", ExitCode: &zero}}
			return s
		},
		"together with validate_with": func(s []Step) []Step {
			s[0].ValidateWith = "check"
			s[1].Loop = &Loop{BackTo: "write", Max: 3, Until: Condition{ExitCode: &zero}}
			s[1].ValidateWith = "write"
			return s
		},
		"releases inside the span": func(s []Step) []Step {
			s[0].Release = []string{"b"}
			s[1].Loop = &Loop{BackTo: "write", Max: 3, Until: Condition{ExitCode: &zero}}
			return s
		},
		"exit_code on an agent step": func(s []Step) []Step {
			s[1] = Step{ID: "check", Uses: "b", Agent: map[string]interface{}{},
				Loop: &Loop{BackTo: "write", Max: 3, Until: Condition{ExitCode: &zero}}}
			return s
		},
	} {
		if err := mk(f).Validate(); err == nil {
			t.Fatalf("%s — it passed", name)
		}
	}

	// 중첩은 아직 없다
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
		t.Fatal("a nested loop passed")
	}
}

// 시연 계약이 실제로 유효한가
//
// testdata/demo.json 은 enode-design/protocol/run-contract.md §2.0 에서
// 그대로 뽑은 것 이다. 정본에 적힌 계약이 코드가 받는 계약과 어긋나면
// 시연 당일에 알게 된다 — 그것을 여기서 막는다.
//
// 이 시험이 깨지면 둘 중 하나다: 문서가 낡았거나, 검증이 문서를 배신했거나.
// 어느 쪽이든 고쳐야 한다.
func TestValidate_TheDemoContract(t *testing.T) {
	raw, err := os.ReadFile("testdata/demo.json")
	if err != nil {
		t.Fatal(err)
	}
	var c Contract
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatalf("the canonical contract does not even parse: %v", err)
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("the canonical contract is rejected: %v", err)
	}

	// 병렬 둘이 실제로 드러나는가 — 이 계약의 값이 거기 있다.
	needs := map[string][]string{}
	for i, st := range c.Steps {
		needs[st.ID] = NeedsOf(c.Steps, i)
	}
	if len(needs["hypothesis"]) != 0 || len(needs["baseline_build"]) != 0 {
		t.Fatalf("the two entry points did not split: %v %v",
			needs["hypothesis"], needs["baseline_build"])
	}
	if len(needs["write_test"]) != 2 {
		t.Fatalf("the join waits on only one side: %v", needs["write_test"])
	}
	// patch_build 는 parent_build 만 기다린다 — parent_observe 와 동시에 돈다.
	// write_test 가 아니다 — parent_build 에 loop 이 있어서, write_test 만
	// 기다리면 반복 도중의 test_source를 쓴다.
	if len(needs["patch_build"]) != 1 || needs["patch_build"][0] != "parent_build" {
		t.Fatalf("the patch build has the wrong dependency: %v", needs["patch_build"])
	}
	if len(needs["parent_observe"]) != 1 || needs["parent_observe"][0] != "parent_build" {
		t.Fatalf("the parent observation has the wrong dependency: %v", needs["parent_observe"])
	}
	// patch_observe 는 보드가 하나라는 사실을 계약에 적어둔 자리다.
	if len(needs["patch_observe"]) != 2 {
		t.Fatalf("the order of the two observations is unwritten: %v — "+
			"the diff breaks the day there are two boards", needs["patch_observe"])
	}
}

// ask 검증 — 질문의 형태가 곧 스키마이고, 평면 폼만 허용한다 (ADR-032)
func TestValidate_ask(t *testing.T) {
	form := map[string]interface{}{
		"type": "object", "required": []interface{}{"verdict"},
		"properties": map[string]interface{}{
			"verdict": map[string]interface{}{"enum": []interface{}{"approve", "reject"}},
			"note":    map[string]interface{}{"type": "string"}}}
	mk := func(f func(*Step)) Contract {
		st := Step{ID: "gate", Ask: &Ask{Prompt: "approve?"},
			Out: []string{"decision"}, Schema: map[string]interface{}{"decision": form}}
		f(&st)
		return Contract{
			RunID:    "r1",
			Requires: []Require{{As: "b", Capability: CapabilityAgentReason}},
			Steps:    []Step{st, {ID: "next", Uses: "b", Run: []string{"true"}}},
		}
	}
	if err := mk(func(*Step) {}).Validate(); err != nil {
		t.Fatalf("a valid ask was rejected: %v", err)
	}
	if err := mk(func(st *Step) {
		st.Ask.Timeout = &AskTimeout{After: "72h", Then: "fail"}
	}).Validate(); err != nil {
		t.Fatalf("an ask with a deadline was rejected: %v", err)
	}

	for name, f := range map[string]func(*Step){
		"uses is present":   func(st *Step) { st.Uses = "b" },
		"prompt is missing": func(st *Step) { st.Ask.Prompt = "" },
		"schema is missing": func(st *Step) { st.Schema = nil },
		"nested object": func(st *Step) {
			st.Schema = map[string]interface{}{"decision": map[string]interface{}{
				"type": "object", "properties": map[string]interface{}{
					"inner": map[string]interface{}{"type": "object"}}}}
		},
		"array": func(st *Step) {
			st.Schema = map[string]interface{}{"decision": map[string]interface{}{
				"type": "object", "properties": map[string]interface{}{
					"list": map[string]interface{}{"type": "array"}}}}
		},
		"then is default — not yet": func(st *Step) {
			st.Ask.Timeout = &AskTimeout{After: "1h", Then: "default"}
		},
		"the deadline does not parse": func(st *Step) {
			st.Ask.Timeout = &AskTimeout{After: "three days", Then: "fail"}
		},
	} {
		if err := mk(f).Validate(); err == nil {
			t.Fatalf("%s — it passed", name)
		}
	}
}

// adopts 검증 — 채택의 어휘를 못 박는다 (ADR-033)
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
		t.Fatalf("a valid adopts was rejected: %v", err)
	}
	if err := mk("nosuch", verdictSch("approve", "reject")).Validate(); err == nil {
		t.Fatal("adopts pointing at a step that does not exist passed")
	}
	if err := mk("gate", verdictSch("approve", "reject")).Validate(); err == nil {
		t.Fatal("adopts pointing at a non-expands step passed")
	}
	// verdict 어휘가 없으면 채택이 기계적으로 못 갈린다
	if err := mk("plan", verdictSch("yes", "no")).Validate(); err == nil {
		t.Fatal("a schema without approve/reject passed")
	}
}

// 사고를 낸 실제 계약이 이제 거부된다 (ADR-060 §2)
//
// third-run-1 의 봉인된 v3 그대로다. 계획이 재시도 루프를 loop 없이 선형으로
// 펴고 final_verify 를 세 분기의 공통 출구로 삼았는데, 그 needs 는 사슬의
// 끝(build_4)만 가리켜서 어느 분기로도 못 닿았다. 그런데 계약은 통과했고
// Run 은 목표 판정 없이 SUCCEEDED 로 봉인됐다.
func TestThirdRun1_RejectsAnUnreachableExit(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "third-run-1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var c Contract
	if err := json.Unmarshal(b, &c); err != nil {
		t.Fatal(err)
	}
	err = c.Validate()
	if err == nil {
		t.Fatal("it passed — the reachability check misses the accident contract")
	}
	if !strings.Contains(err.Error(), "cannot be reached") {
		t.Fatalf("rejected for a different reason: %v", err)
	}
	// 오류가 갈 곳을 가리켜야 한다 — 계획이 이 문장을 읽고 고친다.
	if !strings.Contains(err.Error(), "loop") {
		t.Fatalf("it does not point at the loop: %v", err)
	}
}

// dispatch.to 가 약속한 이름을 가리킬 수 있다 (ADR-062)
//
// 계획 위임에서 승인 경로의 목적지는 계획이 짓는다. 그런데 dispatch.to 는
// 제출 시점에 실존하는 단계만 가리킬 수 있었고, 그래서 ADR-061 이 요구한
// "거절에 갈 곳을 준다" 를 쓰려면 뜻 없는 더미 단계를 지어야 했다.
//
// ADR-049 가 success_when 에 대해 연 것과 같은 자리다.
func TestDispatch_MayPointAtAPromisedName(t *testing.T) {
	body := []byte(`{
	  "run_id":"p1",
	  "requires":[{"as":"b","capability":"agent.reason","role":"x"}],
	  "steps":[
	    {"id":"plan","uses":"b","agent":{"ask":"never"},"out":["plan"],
	     "expands":true,"produces":["report"],
	     "schema":{"plan":{"type":"object","required":["steps"]}}},
	    {"id":"approve","needs":["plan"],
	     "ask":{"prompt":"?","adopts":"plan","adopt_when":"report"},
	     "out":["decision"],
	     "schema":{"decision":{"type":"object","required":["verdict"],
	       "properties":{"verdict":{"enum":["report","replan"]}}}},
	     "dispatch":{"from":"decision.verdict","to":["report","replan"]}},
	    {"id":"replan","uses":"b","needs":["approve"],"agent":{"ask":"never"},
	     "out":["plan2"],"expands":true,"produces":["report"],
	     "schema":{"plan2":{"type":"object","required":["steps"]}}}
	  ],
	  "success_when":[{"step":"report","produced":["summary"]}]}`)
	var c Contract
	if err := json.Unmarshal(body, &c); err != nil {
		t.Fatal(err)
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("rejected while pointing at a promised name: %v", err)
	}
}

// 약속하지 않은 이름은 여전히 거절한다 — 완화가 새어나가지 않는다.
func TestDispatch_RejectsAnUnpromisedName(t *testing.T) {
	body := []byte(`{
	  "run_id":"p2",
	  "requires":[{"as":"b","capability":"agent.reason","role":"x"}],
	  "steps":[
	    {"id":"first","uses":"b","run":["true"],"out":["found"],
	     "schema":{"found":{"type":"object","required":["next"],
	       "properties":{"next":{"enum":["there","here"]}}}},
	     "dispatch":{"from":"found.next","to":["there","here"]}},
	    {"id":"here","uses":"b","run":["true"],"out":["here"]}
	  ],
	  "success_when":[{"step":"here","exit_code":0}]}`)
	var c Contract
	if err := json.Unmarshal(body, &c); err != nil {
		t.Fatal(err)
	}
	err := c.Validate()
	if err == nil {
		t.Fatal("a name nobody promised passed")
	}
	if !strings.Contains(err.Error(), "unknown step") {
		t.Fatalf("rejected for a different reason: %v", err)
	}
}

// agent.mcp · agent.pack 의 값 검사 (features.md 3.5 · US-7)
//
// 문법 대조가 없는 둘을 여기서 잰다 — R3(빈 이름)과 R6(in.from 의 타입)이다.
// 문법에 별 문장을 안 만든 이유는 R2 · R5 의 문장이 값의 모양을 이미 말하고,
// 쪼개면 계획이 읽는 비용만 늘기 때문이다.
func TestValidate_agentComponents(t *testing.T) {
	// 앞 단계가 팩 blob 을 낸다 — in.from 의 이름은 어느 단계가 내는 것이어야
	// 한다는 정적 검사가 이미 있다 (ADR-023 §6.2.1).
	step := func(agent map[string]interface{}, in map[string]interface{}) Contract {
		return Contract{
			RunID:    "r",
			Requires: []Require{req("brain")},
			Steps: []Step{
				{ID: "fetch", Uses: "brain", Run: []string{"true"}, Out: []string{"kernel-review"}},
				{ID: "a", Uses: "brain", Needs: []string{"fetch"},
					Agent: agent, In: in, Out: []string{"x"}},
			},
			SuccessWhen: []Condition{
				{Step: "fetch", ExitCode: zero()},
				{Step: "a", Produced: []string{"x"}},
			},
		}
	}

	// 되는 것부터 — 둘을 제대로 적은 계약은 통과한다.
	ok := step(
		map[string]interface{}{"mcp": []interface{}{"probe"}, "pack": "kernel-review"},
		map[string]interface{}{"prompt": "p", "from": []interface{}{"kernel-review"}},
	)
	if err := ok.Validate(); err != nil {
		t.Fatalf("a valid contract was rejected: %v", err)
	}

	// 빈 배열과 부재가 같다 — 둘 다 허용목록이 빈다는 같은 뜻이다.
	empty := step(map[string]interface{}{"mcp": []interface{}{}}, nil)
	if err := empty.Validate(); err != nil {
		t.Fatalf("an empty agent.mcp must be allowed; it means the allowlist is empty: %v", err)
	}

	cases := []struct {
		name  string
		agent map[string]interface{}
		in    map[string]interface{}
		want  string
	}{
		{
			// 빈 이름은 어느 출처에도 없다 — 노드에서 "없는 서버" 로 죽을
			// 것을 제출에서 같은 답으로 준다.
			name:  "an empty server name",
			agent: map[string]interface{}{"mcp": []interface{}{"probe", ""}},
			want:  "agent.mcp[1]",
		},
		{
			name:  "a server name that is not a string",
			agent: map[string]interface{}{"mcp": []interface{}{1}},
			want:  "agent.mcp[0]",
		},
		{
			name:  "an empty pack name",
			agent: map[string]interface{}{"pack": ""},
			in:    map[string]interface{}{"prompt": "p"},
			want:  "agent.pack must be a blob name",
		},
		{
			// in.from 의 타입이 서야 팩 대조를 할 수 있다.
			name:  "in.from is not an array",
			agent: map[string]interface{}{"pack": "kernel-review"},
			in:    map[string]interface{}{"from": "kernel-review"},
			want:  "in.from must be an array of artifact names",
		},
		{
			// 팩이 없어도 in.from 의 타입은 본다.
			name:  "in.from is not an array, with no pack in play",
			agent: map[string]interface{}{"ask": "never"},
			in:    map[string]interface{}{"from": "kernel-review"},
			want:  "in.from must be an array of artifact names",
		},
	}
	// R5 가 실제로 닫는 구멍 — in.from 을 아예 빼면 오늘은 통과한다.
	//
	// 이름을 틀리게 적은 경우는 이미 in.from 의 정적 검사가 400 으로 막았다
	// (위 step 의 주석). 남아 있던 것은 안 적은 경우이고, 그때 단계는 팩 없이
	// 돌아 0 으로 끝날 수 있었다.
	forgot := step(map[string]interface{}{"pack": "kernel-review"},
		map[string]interface{}{"prompt": "p"})
	err := forgot.Validate()
	if err == nil {
		t.Fatal("a pack with no in.from must be rejected; the step would run with no pack")
	}
	if !strings.Contains(err.Error(), "in.from does not carry it") {
		t.Fatalf("err=%v, want it to name in.from as the place to fix", err)
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := step(tc.agent, tc.in).Validate()
			if err == nil {
				t.Fatalf("it passed")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err=%v, want it to mention %q", err, tc.want)
			}
		})
	}
}
