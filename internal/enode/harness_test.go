package enode

import "testing"

// ADR-013 결정 3 의 부분 정정을 기계적으로 지킨다
//
//	종료코드는 계약이 될 수 없지만 구조화된 봉투는 될 수 있다.
//	claude 는 헛소리를 하고도 0 으로 끝나지만 error_max_turns 로는 끝나지 않는다.
func TestParseClaude(t *testing.T) {
	cases := []struct {
		name      string
		stdout    string
		exit      int
		want      Reason
		completed bool
	}{
		{
			"normal", `{"type":"result","subtype":"success","is_error":false,
			         "num_turns":14,"total_cost_usd":0.83,"result":"…"}`,
			0, ReasonOK, true,
		},
		{
			// 상한 소진은 완주다 — 필요한 걸 다 냈으면 produced 가 판정한다.
			// 다만 Record 에 남는다 (예산 신호).
			"turns exhausted", `{"type":"result","subtype":"error_max_turns","is_error":true,"num_turns":20}`,
			1, ReasonMaxTurns, true,
		},
		{
			"tokens exhausted", `{"type":"result","subtype":"error_max_tokens","is_error":true}`,
			1, ReasonMaxTokens, true,
		},
		{
			// 크래시는 완주가 아니다 — 반쯤 쓴 파일을 남길 수 있어 산출물을 믿을 수 없다
			"no envelope", "Traceback…\nsegfault\n", 139, ReasonError, false,
		},
		{
			"broken envelope", "{not json", 0, ReasonError, false,
		},
		{
			"harness error", `{"type":"result","subtype":"error_during_execution","is_error":true}`,
			1, ReasonError, false,
		},
		{
			// 종료코드 0 을 믿지 않는다 — 로그가 섞여도 봉투를 집는다
			"logs are mixed in",
			"warming up…\ntool call\n" + `{"type":"result","subtype":"success","num_turns":3}`,
			0, ReasonOK, true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := ParseClaude([]byte(c.stdout), c.exit)
			if h.Reason != c.want {
				t.Fatalf("reason=%s, want %s (%s)", h.Reason, c.want, h.Message)
			}
			if h.Reason.Completed() != c.completed {
				t.Fatalf("completed=%v, want %v", h.Reason.Completed(), c.completed)
			}
		})
	}
}

// 비용과 턴 수가 Record 에 남아야 한다 — not_after 는 시간만 묶고 비용을 안 묶는다 (ADR-013).
func TestHarnessRecordsBudget(t *testing.T) {
	h := ParseClaude([]byte(`{"subtype":"success","num_turns":14,"total_cost_usd":0.83}`), 0)
	if h.Turns != 14 || h.CostUSD != 0.83 {
		t.Fatalf("the budget signal was not kept: %+v", h)
	}
}

// 봉투의 session_id 를 읽어놓고 버리지 않는다 — R4 가 여기 걸린다.
func TestParseClaude_PassesTheSessionThrough(t *testing.T) {
	env := `{"type":"result","subtype":"success","is_error":false,` +
		`"num_turns":3,"total_cost_usd":0.01,"session_id":"abc-123","result":"ok"}`
	h := ParseClaude([]byte(env), 0)
	if h.Session != "abc-123" {
		t.Fatalf("the session was not passed through: %q", h.Session)
	}
	if h.Reason != ReasonOK {
		t.Fatalf("reason=%v", h.Reason)
	}
}

// 봉투에 session_id 가 없어도 나머지는 그대로 산다.
func TestParseClaude_NoSessionIsFine(t *testing.T) {
	h := ParseClaude([]byte(`{"type":"result","subtype":"success","num_turns":1}`), 0)
	if h.Session != "" || h.Reason != ReasonOK || h.Turns != 1 {
		t.Fatalf("%+v", h)
	}
}
