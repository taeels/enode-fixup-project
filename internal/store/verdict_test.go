package store

import "testing"

import "github.com/taeels/enode/internal/contract"

func i(n int) *int { return &n }

func c(conds ...contract.Condition) contract.Contract {
	return contract.Contract{SuccessWhen: conds}
}

// ★ 이 파일이 ADR-004 를 기계적으로 지킨다 ★
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
			// ★ O4 — 시연에서 가장 설명이 필요한 장면 ★
			// 테스트가 FAIL 을 뱉어도 계약이 "나왔는가" 만 물으면 Run 은 성공이다.
			// 결과값을 통과 기준에 넣으면 ★ 회귀를 증명한 Run 이 FAILED ★ 가 되어
			// ADR-004 의 네 결과표가 뒤집힌다.
			name:    "테스트가 실패했는데 Run 은 성공",
			con:     c(contract.Condition{Step: "observe", Produced: []string{"test_result"}}),
			results: map[string]StepResult{"observe": {ExitCode: i(0), Produced: []string{"test_result", "serial_log"}}},
			want:    StateSucceeded,
		},
		{
			name:    "산출물이 없으면 실패",
			con:     c(contract.Condition{Step: "observe", Produced: []string{"test_result"}}),
			results: map[string]StepResult{"observe": {ExitCode: i(0), Produced: []string{"serial_log"}}},
			want:    StateFailed,
		},
		{
			name:    "exit_code 가 다르면 실패",
			con:     c(contract.Condition{Step: "build", ExitCode: i(0)}),
			results: map[string]StepResult{"build": {ExitCode: i(2), Produced: []string{"build_log"}}},
			want:    StateFailed,
		},
		{
			// ★ 계약이 안 물으면 안 본다 ★ (I3) — 종료코드가 2 여도 조건에 없으면 성공이다.
			name:    "묻지 않은 것은 판정하지 않는다",
			con:     c(contract.Condition{Step: "build", Produced: []string{"build_log"}}),
			results: map[string]StepResult{"build": {ExitCode: i(2), Produced: []string{"build_log"}}},
			want:    StateSucceeded,
		},
		{
			name:    "단계가 결과를 안 남기면 실패",
			con:     c(contract.Condition{Step: "missing", Produced: []string{"x"}}),
			results: map[string]StepResult{},
			want:    StateFailed,
		},
		{
			name: "조건이 여럿이면 전부 만족해야 한다",
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
			name:    "조건이 하나도 없으면 성공",
			con:     c(),
			results: map[string]StepResult{},
			want:    StateSucceeded,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := Verify(tc.con, tc.results)
			if v.State != tc.want {
				t.Fatalf("state=%s 기대 %s · checks=%+v", v.State, tc.want, v.Checks)
			}
		})
	}
}

// verdict 는 ★ 무엇을 왜 ★ 를 남겨야 한다 (ADR-005: 실패 원인이 Record 에 있다).
func TestVerdictRecordsWantAndGot(t *testing.T) {
	v := Verify(
		c(contract.Condition{Step: "build", ExitCode: i(0), Produced: []string{"build_log", "artifact"}}),
		map[string]StepResult{"build": {ExitCode: i(2), Produced: []string{"build_log"}}},
	)
	if len(v.Checks) != 2 {
		t.Fatalf("검사 %d 개 기대 2: %+v", len(v.Checks), v.Checks)
	}
	var sawExit, sawMissing bool
	for _, ch := range v.Checks {
		if ch.What == "exit_code" && ch.Want == 0 && ch.Got == 2 && !ch.OK {
			sawExit = true
		}
		if ch.What == "produced" && !ch.OK && ch.Note != "" {
			sawMissing = true // "없는 것: [artifact]"
		}
	}
	if !sawExit || !sawMissing {
		t.Fatalf("★ 왜 실패했는지가 안 남았다 ★: %+v", v.Checks)
	}
}

// ★ 건너뛴 단계의 조건은 공허하게 참이다 ★ (ADR-022 §7.2 · B2)
//
// dispatch 로 경로가 갈리면 안 간 쪽의 단계는 실행되지 않는다. 그 조건을 실패로
// 치면 ★ 안 간 경로가 Run 을 죽인다 ★ — 계약 저자가 경로별로 조건을 나눠 쓸
// 방법이 없다(경로는 실행 시 정해진다).
func TestVerify_건너뛴_단계는_안_묻는다(t *testing.T) {
	con := contract.Contract{SuccessWhen: []contract.Condition{
		{Step: "taken", Produced: []string{"report"}},
		{Step: "not_taken", ExitCode: i(0)},
	}}
	v := Verify(con, map[string]StepResult{
		"taken":     {Produced: []string{"report"}},
		"not_taken": {Skipped: true},
	})
	if v.State != StateSucceeded {
		t.Fatalf("★ 안 간 경로가 Run 을 죽였다 ★: %+v", v)
	}
	var skipped *Check
	for i := range v.Checks {
		if v.Checks[i].What == "skipped" {
			skipped = &v.Checks[i]
		}
	}
	if skipped == nil || !skipped.OK {
		t.Fatalf("건너뛴 것이 기록에 안 남았다: %+v", v.Checks)
	}
}

// ★ 크래시와 건너뜀은 다르다 ★ — 결과가 없는 것은 여전히 실패다.
func TestVerify_결과_없음은_여전히_실패다(t *testing.T) {
	con := contract.Contract{SuccessWhen: []contract.Condition{
		{Step: "ran", Produced: []string{"x"}},
		{Step: "crashed", ExitCode: i(0)},
	}}
	v := Verify(con, map[string]StepResult{"ran": {Produced: []string{"x"}}})
	if v.State != StateFailed {
		t.Fatalf("★ 크래시를 건너뜀으로 봤다 ★: %+v", v)
	}
}

// ★ 공허한 참을 막는다 ★ — 전부 건너뛰면 SUCCEEDED 가 되면 안 된다.
//
// 「검증이 불필요한 패치」도 ★ 그 경로에 보고 단계가 있어야 ★ 하고
// 거기에 조건이 걸려야 한다. 그게 이 규칙이 요구하는 것이다.
func TestVerify_전부_건너뛰면_실패다(t *testing.T) {
	con := contract.Contract{SuccessWhen: []contract.Condition{
		{Step: "a", ExitCode: i(0)},
		{Step: "b", ExitCode: i(0)},
	}}
	v := Verify(con, map[string]StepResult{
		"a": {Skipped: true}, "b": {Skipped: true},
	})
	if v.State != StateFailed {
		t.Fatalf("★ 아무것도 안 하고 SUCCEEDED 가 됐다 ★: %+v", v)
	}
	last := v.Checks[len(v.Checks)-1]
	if last.What != "any" || last.OK {
		t.Fatalf("이유가 안 남았다: %+v", v.Checks)
	}
}

// ★ 「목표에 못 닿았다」는 기준을 이긴다 ★ (ADR-054)
//
// ★ 실측 ★ (vm-scratch-5) vm_node_up 이 exit 0 · produced[vm_caps] 로 두 조건을
// 다 만족했다. vm_caps 본문은 "enode binary in vm: exit status 1" 이었고
// 함대에 VM 노드는 없었다 — ★ 「아무것도 없다」를 적은 보고서도 존재하는 파일이다 ★.
func Test목표미달이_통과를_뒤집는다(t *testing.T) {
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
	if v := Verify(c, base); v.State != StateSucceeded {
		t.Fatalf("★ 기준이 맞는데 실패했다 ★: %+v", v)
	}

	// ② ★ 계획 단계가 목표 미달을 보고하면 뒤집힌다 ★
	unmet := map[string]StepResult{
		"work":   {ExitCode: &zero, Produced: []string{"caps"}},
		"replan": {Produced: []string{"plan2", contract.UnmetName}},
	}
	v := Verify(c, unmet)
	if v.State != StateFailed {
		t.Fatalf("★ 목표 미달을 보고했는데 통과했다 ★: %+v", v)
	}
	// ★ 무엇이 뒤집었는지가 기록에 남아야 한다 ★
	found := false
	for _, ch := range v.Checks {
		if ch.What == contract.UnmetName && ch.Step == "replan" {
			found = true
		}
	}
	if !found {
		t.Fatalf("★ 뒤집은 이유가 Checks 에 없다 ★: %+v", v.Checks)
	}
	// ★ 나머지 대조도 그대로 남는다 ★ — 기록이지 단축 평가가 아니다.
	if len(v.Checks) < 3 {
		t.Fatalf("★ 대조를 건너뛰었다 ★: %+v", v.Checks)
	}
}

// ★ 방향은 한쪽뿐이다 ★ (ADR-054) — 실패할 Run 을 통과시키지 못한다.
// 이것이 ADR-037(판정 술어를 에이전트가 저작하지 못하게 한다)을 안 깨는 이유다.
func Test목표미달은_실패를_되살리지_못한다(t *testing.T) {
	zero, one := 0, 1
	c := contract.Contract{Steps: []contract.Step{
		{ID: "work", Uses: "n", Run: []string{"true"}, Out: []string{"caps"}},
	}, SuccessWhen: []contract.Condition{{Step: "work", ExitCode: &zero}}}

	// 기준이 안 맞는다 — 그리고 _unmet 도 없다. ★ 그래도 실패다 ★.
	v := Verify(c, map[string]StepResult{"work": {ExitCode: &one}})
	if v.State != StateFailed {
		t.Fatalf("★ 종료코드가 틀렸는데 통과했다 ★: %+v", v)
	}
}

// ★ 목표 미달은 계획을 짓는 단계만 말할 수 있다 ★ (ADR-054 §2.2)
//
// 처음에는 이것이 ★ 프롬프트 문구로만 ★ 있었고 코드에 없었다. 그러면 계약의
// 어떤 단계든 — 명령 단계까지 — $OUT 에 그 이름의 파일이 하나 있으면
// ★ Run 전체가 실패했다 ★. 평범한 단계는 자기 일만 알므로, 그 자리를 주면
// 자기가 막힌 것을 Run 전체의 실패로 선언한다. 그 말은 _cannot 의 자리다.
func Test평범한_단계의_목표미달은_무시된다(t *testing.T) {
	zero := 0
	c := contract.Contract{Steps: []contract.Step{
		{ID: "work", Uses: "n", Run: []string{"true"}, Out: []string{"caps"}},
		{ID: "replan", Uses: "n", Expands: true, Out: []string{"plan2"}},
	}, SuccessWhen: []contract.Condition{{Step: "work", ExitCode: &zero}}}

	v := Verify(c, map[string]StepResult{
		"work":   {ExitCode: &zero, Produced: []string{"caps", contract.UnmetName}},
		"replan": {Produced: []string{"plan2"}},
	})
	if v.State != StateSucceeded {
		t.Fatalf("★ 평범한 단계의 _unmet 이 Run 을 죽였다 ★: %+v", v)
	}
	// ★ 조용히 버리지 않는다 ★ — 왜 안 먹혔는지가 기록에 남아야 한다.
	found := false
	for _, ch := range v.Checks {
		if ch.Step == "work" && ch.What == contract.UnmetName {
			found = true
			if !ch.OK {
				t.Fatal("★ 무시하기로 해놓고 실패로 셌다 ★")
			}
		}
	}
	if !found {
		t.Fatalf("★ 무시한 사실이 Checks 에 없다 ★: %+v", v.Checks)
	}
}

// ★ 종료코드는 명령 단계의 것이다 ★ (ADR-019)
//
// 처음에는 agent 만 막았다. ask 와 acquire 도 프로세스가 없으므로 조건이
// 조용히 통과하면 Verify 가 got=-1 로 비교해 ★ 언제나 거짓 ★ 이 되고,
// 계약 저자는 자기가 무엇을 잘못 적었는지 못 본다.
func Test종료코드_조건은_명령단계만(t *testing.T) {
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
			t.Fatalf("★ %s 단계에 종료코드 조건이 통과했다 ★", tc.name)
		}
	}
}
