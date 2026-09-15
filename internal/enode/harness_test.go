package enode

import (
	"strings"
	"testing"
)

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

// 크래시가 성공으로 안 봉인된다 (decisions.md 6절 ⑯)
//
// stream-json 아래서 하네스가 중간에 죽으면 lastJSONObject 가 집는 마지막
// 완결 객체가 assistant 사건이다. 그것을 봉투로 읽으면 Subtype 도 IsError 도
// 없어 default 로 떨어지고 ReasonOK 가 된다 — Completed() 가 참이라 반쯤 쓴
// $OUT 이 수확되고, 기록에 남는 서명은 reason=ok · turns=0 · cost_usd=0 이다.
//
// 입력은 줄 경계에서 끊긴 stdout 이어야 한다. 객체 중간에서 끊으면
// lastJSONObject 가 } 로 안 끝나 오늘 코드도 이미 harness_error 라
// 이 검사를 안 재는 시험이 된다.
func TestParseClaude_ACrashIsNotSealedAsSuccess(t *testing.T) {
	stdout := `{"type":"system","subtype":"init","mcp_servers":[]}` + "\n" +
		`{"type":"assistant","message":{"content":[{"type":"text","text":"sk-ant-secret"}]},` +
		`"session_id":"s1"}` + "\n"
	h := ParseClaude([]byte(stdout), 0)
	if h.Reason != ReasonError {
		t.Fatalf("a crashed harness was normalised as %s", h.Reason)
	}
	if h.Reason.Completed() {
		t.Fatal("a half-written $OUT would be harvested from this run")
	}
	// Message 는 봉인에 들어간다 — 봉투의 type 만 싣고 원문 줄은 안 싣는다.
	if h.Message != "assistant" {
		t.Fatalf("the message is not the envelope type: %q", h.Message)
	}
	if strings.Contains(h.Message, "sk-ant-secret") {
		t.Fatalf("the raw line was sealed into the record: %q", h.Message)
	}
	// 봉투가 아닌 것에서도 읽을 수 있던 값은 읽는다.
	if h.Session != "s1" {
		t.Fatalf("the session was dropped on the error path: %q", h.Session)
	}
}

// type 이 아예 없는 봉투는 빈 Message 로 안 나간다.
//
// 빈 Message 는 runner.go 가 err.Error() 로 덮는 자리라 값이 갈린다.
func TestParseClaude_AnEnvelopeWithNoTypeSaysSo(t *testing.T) {
	h := ParseClaude([]byte(`{"subtype":"success","num_turns":14,"total_cost_usd":0.83}`), 0)
	if h.Reason != ReasonError || h.Message != "no result envelope type" {
		t.Fatalf("%+v", h)
	}
	// 예산 신호는 그래도 남는다 — 봉투에 있던 값이다.
	if h.Turns != 14 || h.CostUSD != 0.83 {
		t.Fatalf("the budget signal was dropped on the error path: %+v", h)
	}
}
