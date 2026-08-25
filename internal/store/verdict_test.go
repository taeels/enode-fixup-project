package store

import (
	"strings"
	"testing"

	"github.com/taeels/enode/internal/contract"
)

func i(n int) *int { return &n }

func c(conds ...contract.Condition) contract.Contract {
	return contract.Contract{SuccessWhen: conds}
}

// 이 파일이 ADR-004 를 기계적으로 지킨다
//
//	"결과가 나쁜 것은 실패가 아니고, 결과가 없는 것이 실패다"
func TestVerify(t *testing.T) {
	cases := []struct {
		name    string
		con     contract.Contract
		results map[string]StepResult
		want    string
	}{
		{
			// O4 — 시연에서 가장 설명이 필요한 장면
			// 테스트가 FAIL 을 뱉어도 계약이 "나왔는가" 만 물으면 Run 은 성공이다.
			// 결과값을 통과 기준에 넣으면 회귀를 증명한 Run 이 FAILED가 되어
			// ADR-004 의 네 결과표가 뒤집힌다.
			name:    "the test failed but the run succeeded",
			con:     c(contract.Condition{Step: "observe", Produced: []string{"test_result"}}),
			results: map[string]StepResult{"observe": {ExitCode: i(0), Produced: []string{"test_result", "serial_log"}}},
			want:    StateSucceeded,
		},
		{
			name:    "no artifact means failure",
			con:     c(contract.Condition{Step: "observe", Produced: []string{"test_result"}}),
			results: map[string]StepResult{"observe": {ExitCode: i(0), Produced: []string{"serial_log"}}},
			want:    StateFailed,
		},
		{
			name:    "a different exit_code means failure",
			con:     c(contract.Condition{Step: "build", ExitCode: i(0)}),
			results: map[string]StepResult{"build": {ExitCode: i(2), Produced: []string{"build_log"}}},
			want:    StateFailed,
		},
		{
			// 계약이 안 물으면 안 본다 (I3) — 종료코드가 2 여도 조건에 없으면 성공이다.
			name:    "what was not asked is not judged",
			con:     c(contract.Condition{Step: "build", Produced: []string{"build_log"}}),
			results: map[string]StepResult{"build": {ExitCode: i(2), Produced: []string{"build_log"}}},
			want:    StateSucceeded,
		},
		{
			name:    "a step that leaves no result means failure",
			con:     c(contract.Condition{Step: "missing", Produced: []string{"x"}}),
			results: map[string]StepResult{},
			want:    StateFailed,
		},
		{
			name: "several conditions must all hold",
			con: c(
				contract.Condition{Step: "build", ExitCode: i(0)},
				contract.Condition{Step: "observe", Produced: []string{"test_result"}},
			),
			results: map[string]StepResult{
				"build":   {ExitCode: i(0)},
				"observe": {ExitCode: i(0), Produced: []string{"serial_log"}},
			},
			want: StateFailed,
		},
		{
			name:    "no conditions at all means success",
			con:     c(),
			results: map[string]StepResult{},
			want:    StateSucceeded,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := Verify(tc.con, tc.results, nil)
			if v.State != tc.want {
				t.Fatalf("state=%s, want %s · checks=%+v", v.State, tc.want, v.Checks)
			}
		})
	}
}

// verdict 는 무엇을 왜를 남겨야 한다 (ADR-005: 실패 원인이 Record 에 있다).
func TestVerdictRecordsWantAndGot(t *testing.T) {
	v := Verify(
		c(contract.Condition{Step: "build", ExitCode: i(0), Produced: []string{"build_log", "artifact"}}),
		map[string]StepResult{"build": {ExitCode: i(2), Produced: []string{"build_log"}}},
		nil)
	if len(v.Checks) != 2 {
		t.Fatalf("%d checks, want 2: %+v", len(v.Checks), v.Checks)
	}
	var sawExit, sawMissing bool
	for _, ch := range v.Checks {
		if ch.What == "exit_code" && ch.Want == 0 && ch.Got == 2 && !ch.OK {
			sawExit = true
		}
		if ch.What == "produced" && !ch.OK && ch.Note != "" {
			sawMissing = true // "missing: [artifact]"
		}
	}
	if !sawExit || !sawMissing {
		t.Fatalf("why it failed was not recorded: %+v", v.Checks)
	}
}

// 건너뛴 단계의 조건은 공허하게 참이다 (ADR-022 §7.2 · B2)
//
// dispatch 로 경로가 갈리면 안 간 쪽의 단계는 실행되지 않는다. 그 조건을 실패로
// 치면 안 간 경로가 Run 을 죽인다 — 계약 저자가 경로별로 조건을 나눠 쓸
// 방법이 없다(경로는 실행 시 정해진다).
func TestVerify_SkippedStepsAreNotAsked(t *testing.T) {
	con := contract.Contract{SuccessWhen: []contract.Condition{
		{Step: "taken", Produced: []string{"report"}},
		{Step: "not_taken", ExitCode: i(0)},
	}}
	v := Verify(con, map[string]StepResult{
		"taken":     {Produced: []string{"report"}},
		"not_taken": {Skipped: true},
	}, nil)
	if v.State != StateSucceeded {
		t.Fatalf("an untaken branch killed the run: %+v", v)
	}
	var skipped *Check
	for i := range v.Checks {
		if v.Checks[i].What == "skipped" {
			skipped = &v.Checks[i]
		}
	}
	if skipped == nil || !skipped.OK {
		t.Fatalf("the skip was not recorded: %+v", v.Checks)
	}
}

// 크래시와 건너뜀은 다르다 — 결과가 없는 것은 여전히 실패다.
func TestVerify_NoResultIsStillFailure(t *testing.T) {
	con := contract.Contract{SuccessWhen: []contract.Condition{
		{Step: "ran", Produced: []string{"x"}},
		{Step: "crashed", ExitCode: i(0)},
	}}
	v := Verify(con, map[string]StepResult{"ran": {Produced: []string{"x"}}}, nil)
	if v.State != StateFailed {
		t.Fatalf("a crash was read as a skip: %+v", v)
	}
}

// 공허한 참을 막는다 — 전부 건너뛰면 SUCCEEDED 가 되면 안 된다.
//
// 「검증이 불필요한 패치」도 그 경로에 보고 단계가 있어야 하고
// 거기에 조건이 걸려야 한다. 그게 이 규칙이 요구하는 것이다.
func TestVerify_SkippingEverythingIsFailure(t *testing.T) {
	con := contract.Contract{SuccessWhen: []contract.Condition{
		{Step: "a", ExitCode: i(0)},
		{Step: "b", ExitCode: i(0)},
	}}
	v := Verify(con, map[string]StepResult{
		"a": {Skipped: true}, "b": {Skipped: true},
	}, nil)
	if v.State != StateFailed {
		t.Fatalf("SUCCEEDED without doing anything: %+v", v)
	}
	last := v.Checks[len(v.Checks)-1]
	if last.What != "any" || last.OK {
		t.Fatalf("no reason was recorded: %+v", v.Checks)
	}
}

// 「목표에 못 닿았다」는 기준을 이긴다 (ADR-054)
//
// 실측 (vm-scratch-5) vm_node_up 이 exit 0 · produced[vm_caps] 로 두 조건을
// 다 만족했다. vm_caps 본문은 "enode binary in vm: exit status 1" 이었고
// 함대에 VM 노드는 없었다 — 「아무것도 없다」를 적은 보고서도 존재하는 파일이다.
func TestGoalUnmetOverturnsAPass(t *testing.T) {
	zero := 0
	c := contract.Contract{Steps: []contract.Step{
		{ID: "work", Uses: "n", Run: []string{"true"}, Out: []string{"caps"}},
		{ID: "replan", Uses: "n", Expands: true, Out: []string{"plan2"}},
	}, SuccessWhen: []contract.Condition{
		{Step: "work", ExitCode: &zero, Produced: []string{"caps"}},
	}}

	// ① 기준이 만족되면 성공이다 (오늘 그대로)
	base := map[string]StepResult{
		"work":   {ExitCode: &zero, Produced: []string{"caps"}},
		"replan": {Produced: []string{"plan2"}},
	}
	if v := Verify(c, base, nil); v.State != StateSucceeded {
		t.Fatalf("failed even though the criteria held: %+v", v)
	}

	// ② 계획 단계가 목표 미달을 보고하면 뒤집힌다
	unmet := map[string]StepResult{
		"work":   {ExitCode: &zero, Produced: []string{"caps"}},
		"replan": {Produced: []string{"plan2", contract.UnmetName}},
	}
	v := Verify(c, unmet, nil)
	if v.State != StateFailed {
		t.Fatalf("passed even though goal-unmet was reported: %+v", v)
	}
	// 무엇이 뒤집었는지가 기록에 남아야 한다
	found := false
	for _, ch := range v.Checks {
		if ch.What == contract.UnmetName && ch.Step == "replan" {
			found = true
		}
	}
	if !found {
		t.Fatalf("the reason for overturning is missing from Checks: %+v", v.Checks)
	}
	// 나머지 대조도 그대로 남는다 — 기록이지 단축 평가가 아니다.
	if len(v.Checks) < 3 {
		t.Fatalf("the comparison was skipped: %+v", v.Checks)
	}
}

// 방향은 한쪽뿐이다 (ADR-054) — 실패할 Run 을 통과시키지 못한다.
// 이것이 ADR-037(판정 술어를 에이전트가 저작하지 못하게 한다)을 안 깨는 이유다.
func TestGoalUnmetCannotReviveAFailure(t *testing.T) {
	zero, one := 0, 1
	c := contract.Contract{Steps: []contract.Step{
		{ID: "work", Uses: "n", Run: []string{"true"}, Out: []string{"caps"}},
	}, SuccessWhen: []contract.Condition{{Step: "work", ExitCode: &zero}}}

	// 기준이 안 맞는다 — 그리고 _unmet 도 없다. 그래도 실패다.
	v := Verify(c, map[string]StepResult{"work": {ExitCode: &one}}, nil)
	if v.State != StateFailed {
		t.Fatalf("passed with the wrong exit code: %+v", v)
	}
}

// 목표 미달은 계획을 짓는 단계만 말할 수 있다 (ADR-054 §2.2)
//
// 처음에는 이것이 프롬프트 문구로만 있었고 코드에 없었다. 그러면 계약의
// 어떤 단계든 — 명령 단계까지 — $OUT 에 그 이름의 파일이 하나 있으면
// Run 전체가 실패했다. 평범한 단계는 자기 일만 알므로, 그 자리를 주면
// 자기가 막힌 것을 Run 전체의 실패로 선언한다. 그 말은 _cannot 의 자리다.
func TestGoalUnmetOnAnOrdinaryStepIsIgnored(t *testing.T) {
	zero := 0
	c := contract.Contract{Steps: []contract.Step{
		{ID: "work", Uses: "n", Run: []string{"true"}, Out: []string{"caps"}},
		{ID: "replan", Uses: "n", Expands: true, Out: []string{"plan2"}},
	}, SuccessWhen: []contract.Condition{{Step: "work", ExitCode: &zero}}}

	v := Verify(c, map[string]StepResult{
		"work":   {ExitCode: &zero, Produced: []string{"caps", contract.UnmetName}},
		"replan": {Produced: []string{"plan2"}},
	}, nil)
	if v.State != StateSucceeded {
		t.Fatalf("_unmet on an ordinary step killed the run: %+v", v)
	}
	// 조용히 버리지 않는다 — 왜 안 먹혔는지가 기록에 남아야 한다.
	found := false
	for _, ch := range v.Checks {
		if ch.Step == "work" && ch.What == contract.UnmetName {
			found = true
			if !ch.OK {
				t.Fatal("counted as a failure after deciding to ignore it")
			}
		}
	}
	if !found {
		t.Fatalf("the fact that it was ignored is missing from Checks: %+v", v.Checks)
	}
}

// 종료코드는 명령 단계의 것이다 (ADR-019)
//
// 처음에는 agent 만 막았다. ask 와 acquire 도 프로세스가 없으므로 조건이
// 조용히 통과하면 Verify 가 got=-1 로 비교해 언제나 거짓이 되고,
// 계약 저자는 자기가 무엇을 잘못 적었는지 못 본다.
func TestExitCodeConditionsAreCommandStepsOnly(t *testing.T) {
	zero := 0
	for _, tc := range []struct {
		name string
		step contract.Step
	}{
		{"agent", contract.Step{ID: "s", Uses: "n",
			Agent: map[string]interface{}{},
			In:    map[string]interface{}{"prompt": "x"},
			Out:   []string{"o"}}},
		{"ask", contract.Step{ID: "s", Ask: &contract.Ask{Prompt: "?"}, Out: []string{"o"}}},
	} {
		c := contract.Contract{
			RunID:       "r",
			Requires:    []contract.Require{{As: "n", Capability: contract.CapabilityAgentReason}},
			Steps:       []contract.Step{tc.step},
			SuccessWhen: []contract.Condition{{Step: "s", ExitCode: &zero}},
		}
		if err := c.Validate(); err == nil {
			t.Fatalf("an exit-code condition passed on a %s step", tc.name)
		}
	}
}

// 함대 술어 (ADR-058)
//
// vm-scratch 일곱 판의 목표는 "노드가 함대에 능력을 광고하며 선다" 인데
// 계약이 그것을 표현할 수단이 없었다. 그래서 대리를 세 번 갈아탔고
// (exit_code → produced → _unmet) 갈 때마다 새 결함이 났다.
func TestFleetPredicate(t *testing.T) {
	fleet := func(specs ...map[string]string) []contract.Advert {
		var out []contract.Advert
		for i, sp := range specs {
			out = append(out, contract.Advert{
				NodeID: string(rune('a' + i)),
				Capabilities: []contract.Capability{
					{Capability: contract.CapabilityAgentReason, Attrs: sp}},
			})
		}
		return out
	}
	linux := map[string]string{"os": "linux", "host_arch": "arm64", "harness": "claude"}
	mac := map[string]string{"os": "darwin", "host_arch": "arm64", "harness": "claude"}

	con := contract.Contract{
		Steps: []contract.Step{{ID: "a", Uses: "n", Run: []string{"true"}, Out: []string{"o"}}},
		SuccessWhen: []contract.Condition{{
			FleetHas: &contract.Require{
				Capability: contract.CapabilityAgentReason,
				Attrs:      map[string]string{"os": "linux"}},
		}},
	}
	res := map[string]StepResult{"a": {Produced: []string{"o"}}}

	// ① 그런 노드가 있으면 성공
	if v := Verify(con, res, fleet(linux, mac)); v.State != StateSucceeded {
		t.Fatalf("failed even though a linux node exists: %+v", v)
	}
	// ② 없으면 실패 — 단계는 다 성공했는데도
	v := Verify(con, res, fleet(mac))
	if v.State != StateFailed {
		t.Fatalf("passed even though no linux node exists: %+v", v)
	}
	// 무엇을 요구했는지가 기록에 남는다 — 속성 어휘가 창발하므로(ADR-012)
	// 안 보이면 왜 실패했는지 못 찾는다.
	found := false
	for _, ch := range v.Checks {
		if ch.Step == "(fleet)" && strings.Contains(ch.What, "os=linux") {
			found = true
		}
	}
	if !found {
		t.Fatalf("what was required is missing from Checks: %+v", v.Checks)
	}

	// ③ 함대를 관측 못 했으면 실패다 — 조용히 참이 되지 않는다.
	if v := Verify(con, res, nil); v.State != StateFailed {
		t.Fatalf("passed without ever seeing the fleet: %+v", v)
	}

	// ④ min_count
	two := con
	two.SuccessWhen = []contract.Condition{{
		FleetHas: &contract.Require{Capability: contract.CapabilityAgentReason,
			Attrs: map[string]string{"host_arch": "arm64"}},
		MinCount: 3,
	}}
	if v := Verify(two, res, fleet(linux, mac)); v.State != StateFailed {
		t.Fatalf("a condition asking for three passed with only two: %+v", v)
	}

	// ⑤ 관측한 함대가 봉인된다 — Verify 는 순수 함수이고 함대는 인자다.
	// 그 인자를 안 남기면 봉인된 묶음만 보고 판정을 재현할 수 없다.
	got := Verify(con, res, fleet(linux))
	if len(got.Fleet) != 1 {
		t.Fatalf("the observed fleet was not kept in the verdict: %+v", got.Fleet)
	}

	// ⑥ 함대 조건이 없으면 함대를 안 봉인한다 — 안 쓰는 것을 안 남긴다.
	plain := contract.Contract{
		Steps:       con.Steps,
		SuccessWhen: []contract.Condition{{Step: "a", Produced: []string{"o"}}},
	}
	if v := Verify(plain, res, fleet(linux)); len(v.Fleet) != 0 {
		t.Fatal("sealed a fleet nobody asked about")
	}
}

// 고른 목적지가 안 돌았으면 공허하게 참이 아니다 (ADR-060 §3)
//
// third-run-1 의 판정 쪽 재현이다. 목표 단계가 SKIPPED 인데 절차 단계 둘이
// 조건을 채워서 Run 이 SUCCEEDED 로 봉인됐다 — evaluated 하한도 0 이 아니라
// 발동하지 못했다.
func TestAChosenStepThatNeverRanIsGoalUnmet(t *testing.T) {
	c := contract.Contract{
		Steps: []contract.Step{
			{ID: "plan"}, {ID: "approve"}, {ID: "final"},
		},
		SuccessWhen: []contract.Condition{
			{Step: "plan", Produced: []string{"plan"}},
			{Step: "approve", Produced: []string{"approval"}},
			{Step: "final", Produced: []string{"verify_result"}},
		},
	}
	base := map[string]StepResult{
		"plan":    {Produced: []string{"plan"}},
		"approve": {Produced: []string{"approval"}},
	}

	// ① 안 골라서 건너뛰었다 — 공허하게 참이 옳다 (경로가 갈렸다)
	notChosen := map[string]StepResult{}
	for k, v := range base {
		notChosen[k] = v
	}
	notChosen["final"] = StepResult{Skipped: true}
	if got := Verify(c, notChosen, nil); got.State != StateSucceeded {
		t.Fatalf("an unchosen SKIPPED must pass: %s", got.State)
	}

	// ② 골랐는데도 건너뛰었다 — 목표 판정이 통째로 빠졌다
	chosen := map[string]StepResult{}
	for k, v := range base {
		chosen[k] = v
	}
	chosen["final"] = StepResult{Skipped: true, Chosen: true}
	got := Verify(c, chosen, nil)
	if got.State != StateFailed {
		t.Fatalf("passed even though the chosen destination never ran: %s — "+
			"this is how third-run-1 got sealed as SUCCEEDED", got.State)
	}
}
