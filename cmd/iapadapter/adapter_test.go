package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseAnswer_승인과_거절을_가른다(t *testing.T) {
	cases := []struct {
		name    string
		prompt  string
		verdict string
		note    string
	}{
		{
			name:    "ok 한 마디",
			prompt:  "The comment that mentioned you: ok",
			verdict: "ok",
		},
		{
			name:    "again 뒤가 이유다",
			prompt:  "The comment that mentioned you: again. 3단계가 워크스페이스 밖에 쓴다.",
			verdict: "again",
			note:    "3단계가 워크스페이스 밖에 쓴다.",
		},
		{
			// ★ 스레드가 통째로 실려 온다 ★ (실측 §10.5.2) — 그래서 표지 뒤만 본다.
			// 표지 앞의 우리 질문에도 "ok" 라는 낱말이 들어 있다.
			name: "스레드 맥락에 낚이지 않는다",
			prompt: "Someone answered your comment on issue EP-2\n" +
				"The comments above it in the thread, oldest first:\n" +
				"  enode fleet: 승인이면 ok 로, 다시 지어야 하면 again 으로 답해 주세요.\n" +
				"The comment that mentioned you: again 표를 더 자세히",
			verdict: "again",
			note:    "표를 더 자세히",
		},
		{
			name:    "한국어 승인",
			prompt:  "The comment that mentioned you: 승인",
			verdict: "ok",
		},
		{
			name:   "판정을 못 읽으면 빈 값이다",
			prompt: "The comment that mentioned you: 음 글쎄요",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v, n := parseAnswer(c.prompt)
			if v != c.verdict {
				t.Fatalf("verdict = %q, want %q", v, c.verdict)
			}
			if n != c.note {
				t.Fatalf("note = %q, want %q", n, c.note)
			}
		})
	}
}

func TestRunIDPrefix_이슈에서_유도된다(t *testing.T) {
	// ★ 표가 아니라 유도다 ★ — 어댑터가 죽어도 같은 규칙이 같은 이름을 준다.
	if got := runIDPrefix("EP-2"); got != "itsaplan-EP-2-" {
		t.Fatalf("prefix = %q", got)
	}
	// 접두사가 다른 이슈를 안 삼켜야 한다.
	if strings.HasPrefix("itsaplan-EP-20-1", runIDPrefix("EP-2")) {
		t.Fatal("EP-2 의 접두사가 EP-20 을 삼킨다")
	}
}

func testConfig() *Config {
	return &Config{
		Executor: ExecutorConfig{
			As:    "worker",
			Attrs: map[string]string{"harness": "claude", "repo": "mirror"},
		},
	}
}

func TestBuildContract_고정_템플릿이다(t *testing.T) {
	rr := &RunnerRun{ID: 7, Trigger: "delegation", Prompt: "일을 맡깁니다", IssueIdentifier: "EP-2"}
	issue := &Issue{ID: 2, Title: "제목", Description: "본문"}

	raw, err := BuildContract(testConfig(), "itsaplan-EP-2-1", "EP-2", issue, rr)
	if err != nil {
		t.Fatal(err)
	}
	var c map[string]any
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatal(err)
	}

	if c["run_id"] != "itsaplan-EP-2-1" {
		t.Fatalf("run_id = %v", c["run_id"])
	}
	work := c["work"].(map[string]any)
	if work["system"] != "itsaplan" || work["change_id"] != "EP-2" {
		t.Fatalf("work = %v", work)
	}

	// ★ 이름표가 붙는다 ★ — 이것이 없으면 남의 이슈의 오케스트레이터가 걸린다.
	reqs := c["requires"].([]any)
	planner := reqs[0].(map[string]any)
	if planner["capability"] != "orchestration" || planner["issue"] != "EP-2" {
		t.Fatalf("planner = %v", planner)
	}
	worker := reqs[1].(map[string]any)
	if worker["capability"] != "agent.reason" || worker["harness"] != "claude" {
		t.Fatalf("worker = %v", worker)
	}

	// ★ 원장이 Work 범위여야 답을 이전 Run 에서 찾을 수 있다 ★ (ADR-040 §3.4).
	if c["ledger"].(map[string]any)["scope"] != "work" {
		t.Fatalf("ledger = %v", c["ledger"])
	}

	steps := c["steps"].([]any)
	plan := steps[0].(map[string]any)
	if plan["expands"] != true {
		t.Fatal("plan 단계가 expands 가 아니다")
	}
	if prods := plan["produces"].([]any); len(prods) != 1 || prods[0] != "report" {
		t.Fatalf("produces = %v", plan["produces"])
	}

	gate := steps[1].(map[string]any)
	ask := gate["ask"].(map[string]any)
	if ask["adopts"] != "plan" || ask["adopt_when"] != "ok" {
		t.Fatalf("ask = %v", ask)
	}
	// ★ 계약이 목적지를 안 적는다 ★ (ADR-062) — dispatch 가 있으면 앞단이
	// 뒷단의 계획 모양을 단정하게 되고, 그것이 promised-1 을 죽였다.
	if _, has := gate["dispatch"]; has {
		t.Fatal("계약이 dispatch 를 쓴다 — ADR-062 가 없앤 것이다")
	}
}

func TestBuildContract_판정은_그래프의_자리가_아니라_사실을_묻는다(t *testing.T) {
	rr := &RunnerRun{ID: 1, IssueIdentifier: "EP-9"}
	raw, err := BuildContract(testConfig(), "itsaplan-EP-9-1", "EP-9", nil, rr)
	if err != nil {
		t.Fatal(err)
	}
	var c map[string]any
	_ = json.Unmarshal(raw, &c)
	sw := c["success_when"].([]any)
	if len(sw) != 2 {
		t.Fatalf("success_when 이 %d 개다", len(sw))
	}
	last := sw[1].(map[string]any)
	// report 앞에 몇 단계가 있든 상관없다 — 그것이 produces 가 괜찮았던 이유다.
	if last["step"] != "report" {
		t.Fatalf("마지막 조건이 report 를 안 본다: %v", last)
	}
}

func TestRenderQuestion_답하는_방법을_말로_적는다(t *testing.T) {
	// ★ It's a Plan 에 폼이 없다 ★ — 스키마 강제는 우리 쪽에 남으므로
	// 코멘트가 답의 형태를 말로 알려줘야 한다.
	ask := &AskView{
		RunID:  "itsaplan-EP-2-1",
		Seq:    2,
		Step:   "gate",
		Prompt: "이 계획으로 진행할까요",
		Shown: []struct {
			Name    string          `json:"name"`
			Content json.RawMessage `json:"content"`
		}{{Name: "plan", Content: json.RawMessage(`{"steps":[]}`)}},
	}
	got := RenderQuestion(ask, "itsaplan-EP-2-1")
	for _, want := range []string{"이 계획으로 진행할까요", "plan", "`ok`", "`again`", "itsaplan-EP-2-1"} {
		if !strings.Contains(got, want) {
			t.Fatalf("코멘트에 %q 가 없다:\n%s", want, got)
		}
	}
}

func TestOneLine_봉인을_붓지_않는다(t *testing.T) {
	// agent_run.output 은 지울 수 있고 스키마도 없다 (ADR-040 §1).
	run := &RunView{RunID: "itsaplan-EP-2-1", State: "SUCCEEDED"}
	run.Verdict.Checks = []Check{{OK: true}, {OK: false}}

	got := oneLine(run, "itsaplan-EP-2-1")
	if !strings.Contains(got, "1/2") || !strings.Contains(got, "itsaplan-EP-2-1") {
		t.Fatalf("한 줄 요약 = %q", got)
	}
	if len(got) > 200 {
		t.Fatalf("한 줄이 아니다 (%d 자)", len(got))
	}
}

// ★ 첫 실측이 잡은 결함이다 ★ (2026-08-24, EP-2) — verdict.checks 의 want·got 은
// produced 검사에서 문자열 배열이고 exit_code 검사에서 숫자다. []string 으로
// 받았더니 Run 조회가 통째로 실패했고, 어댑터가 ★ 끝난 Run 을 영영 못 넘겼다 ★.
// 조용히 멈추는 종류의 고장이라 시험으로 못 박는다.
func TestRunView_판정_검사의_want_got_은_다형이다(t *testing.T) {
	raw := `{
	  "run_id": "itsaplan-EP-2-1",
	  "state": "SUCCEEDED",
	  "verdict": { "state": "SUCCEEDED", "checks": [
	    { "step": "plan",    "what": "produced",  "want": ["plan"], "got": ["plan"], "ok": true },
	    { "step": "collect", "what": "exit_code", "want": 0,        "got": 0,        "ok": true },
	    { "step": "fleet",   "what": "fleet_has", "want": {"capability": "agent.reason"}, "ok": false,
	      "note": "함대에 후보가 없다" }
	  ] }
	}`
	var run RunView
	if err := json.Unmarshal([]byte(raw), &run); err != nil {
		t.Fatalf("Run 조회를 못 읽는다: %v", err)
	}
	if len(run.Verdict.Checks) != 3 {
		t.Fatalf("검사가 %d 개다", len(run.Verdict.Checks))
	}

	// 렌더러도 형을 단정하면 안 된다 — 어휘가 창발한다 (ADR-012).
	if got := describe(run.Verdict.Checks[0].Want); got != "`plan`" {
		t.Fatalf("배열 want = %q", got)
	}
	if got := describe(run.Verdict.Checks[1].Want); got != "`0`" {
		t.Fatalf("숫자 want = %q", got)
	}
	if got := describe(run.Verdict.Checks[2].Want); !strings.Contains(got, "agent.reason") {
		t.Fatalf("객체 want = %q", got)
	}
	if got := describe(nil); got != "" {
		t.Fatalf("빈 want = %q", got)
	}
}

func TestRunView_Terminal(t *testing.T) {
	for _, s := range []string{"SUCCEEDED", "FAILED", "CANCELLED", "EXPIRED"} {
		if !(&RunView{State: s}).Terminal() {
			t.Fatalf("%s 가 종료가 아니라고 한다", s)
		}
	}
	// ★ ASKED 는 종료가 아니다 ★ — 사람이 답하는 동안 Run 은 살아 있다 (ADR-047).
	for _, s := range []string{"PENDING", "RUNNING", "ASKED"} {
		if (&RunView{State: s}).Terminal() {
			t.Fatalf("%s 가 종료라고 한다", s)
		}
	}
}
