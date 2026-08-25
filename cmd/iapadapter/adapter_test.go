package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestParseAnswer_SplitsApproveFromReject(t *testing.T) {
	cases := []struct {
		name    string
		prompt  string
		verdict string
		note    string
	}{
		{
			name:    "a bare ok",
			prompt:  "The comment that mentioned you: ok",
			verdict: "ok",
		},
		{
			name:    "what follows again is the reason",
			prompt:  "The comment that mentioned you: again. 3단계가 워크스페이스 밖에 쓴다.",
			verdict: "again",
			note:    "3단계가 워크스페이스 밖에 쓴다.",
		},
		{
			// 스레드가 통째로 실려 온다 (실측 §10.5.2) — 그래서 표지 뒤만 본다.
			// 표지 앞의 우리 질문에도 "ok" 라는 낱말이 들어 있다.
			name: "not fooled by the thread context",
			prompt: "Someone answered your comment on issue EP-2\n" +
				"The comments above it in the thread, oldest first:\n" +
				"  enode fleet: 승인이면 ok 로, 다시 지어야 하면 again 으로 답해 주세요.\n" +
				"The comment that mentioned you: again 표를 더 자세히",
			verdict: "again",
			note:    "표를 더 자세히",
		},
		{
			name:    "approval written in korean",
			prompt:  "The comment that mentioned you: 승인",
			verdict: "ok",
		},
		{
			name:   "an unreadable verdict yields an empty value",
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

func TestRunIDPrefix_DerivedFromTheIssue(t *testing.T) {
	// 표가 아니라 유도다 — 어댑터가 죽어도 같은 규칙이 같은 이름을 준다.
	if got := runIDPrefix("EP-2"); got != "itsaplan-EP-2-" {
		t.Fatalf("prefix = %q", got)
	}
	// 접두사가 다른 이슈를 안 삼켜야 한다.
	if strings.HasPrefix("itsaplan-EP-20-1", runIDPrefix("EP-2")) {
		t.Fatal("the EP-2 prefix swallows EP-20")
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

func TestBuildContract_IsAFixedTemplate(t *testing.T) {
	rr := &RunnerRun{ID: 7, Trigger: "delegation", Prompt: "handing over the work", IssueIdentifier: "EP-2"}
	issue := &Issue{ID: 2, Title: "title", Description: "body"}

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

	// 이름표가 붙는다 — 이것이 없으면 남의 이슈의 오케스트레이터가 걸린다.
	reqs := c["requires"].([]any)
	planner := reqs[0].(map[string]any)
	if planner["capability"] != "orchestration" || planner["issue"] != "EP-2" {
		t.Fatalf("planner = %v", planner)
	}
	worker := reqs[1].(map[string]any)
	if worker["capability"] != "agent.reason" || worker["harness"] != "claude" {
		t.Fatalf("worker = %v", worker)
	}

	// 원장이 Work 범위여야 답을 이전 Run 에서 찾을 수 있다 (ADR-040 §3.4).
	if c["ledger"].(map[string]any)["scope"] != "work" {
		t.Fatalf("ledger = %v", c["ledger"])
	}

	steps := c["steps"].([]any)
	plan := steps[0].(map[string]any)
	if plan["expands"] != true {
		t.Fatal("the plan step is not expands")
	}
	if prods := plan["produces"].([]any); len(prods) != 1 || prods[0] != "report" {
		t.Fatalf("produces = %v", plan["produces"])
	}

	gate := steps[1].(map[string]any)
	ask := gate["ask"].(map[string]any)
	if ask["adopts"] != "plan" || ask["adopt_when"] != "ok" {
		t.Fatalf("ask = %v", ask)
	}
	// 계약이 목적지를 안 적는다 (ADR-062) — dispatch 가 있으면 앞단이
	// 뒷단의 계획 모양을 단정하게 되고, 그것이 promised-1 을 죽였다.
	if _, has := gate["dispatch"]; has {
		t.Fatal("the contract uses dispatch — ADR-062 removed it")
	}
}

func TestBuildContract_JudgesFactsNotGraphPositions(t *testing.T) {
	rr := &RunnerRun{ID: 1, IssueIdentifier: "EP-9"}
	raw, err := BuildContract(testConfig(), "itsaplan-EP-9-1", "EP-9", nil, rr)
	if err != nil {
		t.Fatal(err)
	}
	var c map[string]any
	_ = json.Unmarshal(raw, &c)
	sw := c["success_when"].([]any)
	if len(sw) != 2 {
		t.Fatalf("success_when has %d entries", len(sw))
	}
	last := sw[1].(map[string]any)
	// report 앞에 몇 단계가 있든 상관없다 — 그것이 produces 가 괜찮았던 이유다.
	if last["step"] != "report" {
		t.Fatalf("the last condition does not look at report: %v", last)
	}
}

func TestRenderQuestion_SpellsOutHowToAnswer(t *testing.T) {
	// It's a Plan 에 폼이 없다 — 스키마 강제는 우리 쪽에 남으므로
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
			t.Fatalf("the comment does not contain %q:\n%s", want, got)
		}
	}
}

// 두 번째 실측이 잡은 결함이다 (2026-08-24, EP-2) — 중복 표지를 run_id 로
// 삼았더니 질문 코멘트가 결과 코멘트로 오인됐다. 둘 다 꼬리표에 run_id 를
// 달기 때문이다. 그래서 복구 경로가 「이미 넘겼다」로 판단하고 조용히 지나갔다.
func TestResultMarker_DoesNotCollideWithTheQuestionComment(t *testing.T) {
	const runID = "itsaplan-EP-2-1"
	ask := &AskView{RunID: runID, Seq: 2, Step: "gate", Prompt: "진행할까요"}
	question := RenderQuestion(ask, runID)

	// 질문에도 run_id 는 들어 있다 — 사람이 어느 Run 인지 알아야 하므로 맞다.
	if !strings.Contains(question, runID) {
		t.Fatal("the question carries no run_id")
	}
	// 그런데 결과 표지는 없어야 한다.
	if strings.Contains(question, ResultMarker(runID)) {
		t.Fatalf("the question comment carries the result marker:\n%s", question)
	}
}

func TestOneLine_DoesNotPourOutTheSeal(t *testing.T) {
	// agent_run.output 은 지울 수 있고 스키마도 없다 (ADR-040 §1).
	run := &RunView{RunID: "itsaplan-EP-2-1", State: "SUCCEEDED"}
	run.Verdict.Checks = []Check{{OK: true}, {OK: false}}

	got := oneLine(run, "itsaplan-EP-2-1")
	if !strings.Contains(got, "1/2") || !strings.Contains(got, "itsaplan-EP-2-1") {
		t.Fatalf("one-line summary = %q", got)
	}
	if len(got) > 200 {
		t.Fatalf("not one line (%d chars)", len(got))
	}
}

// 첫 실측이 잡은 결함이다 (2026-08-24, EP-2) — verdict.checks 의 want·got 은
// produced 검사에서 문자열 배열이고 exit_code 검사에서 숫자다. []string 으로
// 받았더니 Run 조회가 통째로 실패했고, 어댑터가 끝난 Run 을 영영 못 넘겼다.
// 조용히 멈추는 종류의 고장이라 시험으로 못 박는다.
func TestRunView_CheckWantGotArePolymorphic(t *testing.T) {
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
		t.Fatalf("cannot read the run lookup: %v", err)
	}
	if len(run.Verdict.Checks) != 3 {
		t.Fatalf("%d checks", len(run.Verdict.Checks))
	}

	// 렌더러도 형을 단정하면 안 된다 — 어휘가 창발한다 (ADR-012).
	if got := describe(run.Verdict.Checks[0].Want); got != "`plan`" {
		t.Fatalf("array want = %q", got)
	}
	if got := describe(run.Verdict.Checks[1].Want); got != "`0`" {
		t.Fatalf("number want = %q", got)
	}
	if got := describe(run.Verdict.Checks[2].Want); !strings.Contains(got, "agent.reason") {
		t.Fatalf("object want = %q", got)
	}
	if got := describe(nil); got != "" {
		t.Fatalf("empty want = %q", got)
	}
}

// 세 번째 실측이 잡은 결함이다 (2026-08-24) — 되묻기로 손을 떼는 경로는
// 오케스트레이터를 죽이면 안 되므로 Stop 을 안 불렀는데, Wait 가 Stop 에만
// 있었다. 그래서 그 판마다 좀비가 하나씩 쌓였다 (EP-3·EP-4·EP-5 에서 셋).
// 거두는 자리를 시작 직후 고루틴 하나로 옮기고, Detach 도 그것을 기다린다.
func TestOrchestrator_ReapsTheChildOnEveryPath(t *testing.T) {
	for _, tc := range []struct {
		name string
		end  func(o *Orchestrator)
	}{
		{"Stop reaps", func(o *Orchestrator) { o.Stop() }},
		{"Detach reaps too", func(o *Orchestrator) { o.Detach() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			cmd := exec.Command("/bin/sh", "-c", "exit 0")
			if err := cmd.Start(); err != nil {
				t.Skipf("cannot spawn a child: %v", err)
			}
			o := &Orchestrator{
				IssueKey: "EP-9", Dir: dir, cmd: cmd,
				log:  slog.New(slog.NewTextHandler(io.Discard, nil)),
				done: make(chan struct{}),
			}
			go func() { _ = cmd.Wait(); close(o.done) }()

			tc.end(o)

			// done 이 닫혔다 = Wait 가 불렸다 = 좀비가 안 남는다.
			select {
			case <-o.done:
			case <-time.After(5 * time.Second):
				t.Fatal("the child was not reaped — a zombie remains")
			}
		})
	}
}

func TestRunView_Terminal(t *testing.T) {
	for _, s := range []string{"SUCCEEDED", "FAILED", "CANCELLED", "EXPIRED"} {
		if !(&RunView{State: s}).Terminal() {
			t.Fatalf("%s is reported as not terminal", s)
		}
	}
	// ASKED 는 종료가 아니다 — 사람이 답하는 동안 Run 은 살아 있다 (ADR-047).
	for _, s := range []string{"PENDING", "RUNNING", "ASKED"} {
		if (&RunView{State: s}).Terminal() {
			t.Fatalf("%s is reported as terminal", s)
		}
	}
}
