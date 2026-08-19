package enode

import "testing"

// ★ ADR-013 결정 3 의 부분 정정을 기계적으로 지킨다 ★
//
//	종료코드는 계약이 될 수 없지만 ★ 구조화된 봉투는 될 수 있다 ★.
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
			"정상", `{"type":"result","subtype":"success","is_error":false,
			         "num_turns":14,"total_cost_usd":0.83,"result":"…"}`,
			0, ReasonOK, true,
		},
		{
			// ★ 상한 소진은 완주다 ★ — 필요한 걸 다 냈으면 produced 가 판정한다.
			// 다만 Record 에 남는다 (예산 신호).
			"턴 소진", `{"type":"result","subtype":"error_max_turns","is_error":true,"num_turns":20}`,
			1, ReasonMaxTurns, true,
		},
		{
			"토큰 소진", `{"type":"result","subtype":"error_max_tokens","is_error":true}`,
			1, ReasonMaxTokens, true,
		},
		{
			// ★ 크래시는 완주가 아니다 ★ — 반쯤 쓴 파일을 남길 수 있어 산출물을 믿을 수 없다
			"봉투가 없다", "Traceback…\nsegfault\n", 139, ReasonError, false,
		},
		{
			"봉투가 깨졌다", "{not json", 0, ReasonError, false,
		},
		{
			"하네스 오류", `{"type":"result","subtype":"error_during_execution","is_error":true}`,
			1, ReasonError, false,
		},
		{
			// ★ 종료코드 0 을 믿지 않는다 ★ — 로그가 섞여도 봉투를 집는다
			"로그가 섞여 있다",
			"준비 중…\n도구 호출\n" + `{"type":"result","subtype":"success","num_turns":3}`,
			0, ReasonOK, true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := ParseClaude([]byte(c.stdout), c.exit)
			if h.Reason != c.want {
				t.Fatalf("reason=%s 기대 %s (%s)", h.Reason, c.want, h.Message)
			}
			if h.Reason.Completed() != c.completed {
				t.Fatalf("completed=%v 기대 %v", h.Reason.Completed(), c.completed)
			}
		})
	}
}

// 비용과 턴 수가 Record 에 남아야 한다 — not_after 는 시간만 묶고 비용을 안 묶는다 (ADR-013).
func TestHarnessRecordsBudget(t *testing.T) {
	h := ParseClaude([]byte(`{"subtype":"success","num_turns":14,"total_cost_usd":0.83}`), 0)
	if h.Turns != 14 || h.CostUSD != 0.83 {
		t.Fatalf("예산 신호가 안 남았다: %+v", h)
	}
}
