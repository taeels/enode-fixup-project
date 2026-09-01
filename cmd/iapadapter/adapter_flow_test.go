package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// 어댑터의 흐름은 두 바깥(It's a Plan · Mediator)을 상대로만 관찰된다 —
// 어느 코멘트가 나갔고 어느 칸으로 옮겼고 러너에 무엇을 보고했나.
// 그래서 대역 둘을 세우고 그 셋을 단언한다.

func def(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// iapFake 는 어댑터가 부르는 It's a Plan 라우트를 전부 받는 대역이다.
type iapFake struct {
	mu sync.Mutex

	queue   []string       // claim 이 순서대로 내줄 본문
	issue   string         // GET /issues/{id}
	feed    string         // GET /issues/{id}/feed
	columns string         // GET /projects/{key}
	status  map[string]int // 경로 접두어 -> 강제 상태 코드
	// issueStatus 는 이슈 본문 조회에만 걸린다 — 접두어로 누르면 같은
	// /issues/{id}/ 아래의 코멘트까지 같이 죽어 실패 경로가 안 보인다.
	issueStatus int

	comments   []string
	results    []string
	moved      []int
	heartbeats int
	claims     int
	nextID     int
}

func (f *iapFake) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	raw, _ := io.ReadAll(r.Body)
	p := r.URL.Path
	if p == "/agent-runs/claim" {
		f.claims++
	}
	for pre, code := range f.status {
		if strings.HasPrefix(p, pre) {
			w.WriteHeader(code)
			_, _ = w.Write([]byte("forced by the test"))
			return
		}
	}
	switch {
	case p == "/agent-runs/claim":
		if len(f.queue) == 0 {
			_, _ = w.Write([]byte(`{"run":null}`))
			return
		}
		body := f.queue[0]
		f.queue = f.queue[1:]
		_, _ = w.Write([]byte(body))
	case strings.HasSuffix(p, "/heartbeat"):
		f.heartbeats++
		_, _ = w.Write([]byte(`{}`))
	case strings.HasSuffix(p, "/result"):
		f.results = append(f.results, string(raw))
		_, _ = w.Write([]byte(`{}`))
	case strings.HasSuffix(p, "/comments"):
		f.comments = append(f.comments, string(raw))
		f.nextID++
		fmt.Fprintf(w, `{"id":%d}`, f.nextID)
	case strings.HasPrefix(p, "/projects/"):
		_, _ = w.Write([]byte(def(f.columns, `{"columns":[]}`)))
	case r.Method == http.MethodPatch && strings.HasPrefix(p, "/issues/"):
		var b struct {
			ColumnID int `json:"columnId"`
		}
		_ = json.Unmarshal(raw, &b)
		f.moved = append(f.moved, b.ColumnID)
		_, _ = w.Write([]byte(`{}`))
	case strings.HasSuffix(p, "/feed"):
		_, _ = w.Write([]byte(def(f.feed, `{"items":[]}`)))
	case strings.HasPrefix(p, "/issues/"):
		if f.issueStatus != 0 {
			w.WriteHeader(f.issueStatus)
			return
		}
		_, _ = w.Write([]byte(def(f.issue, `{}`)))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (f *iapFake) snapshot() ([]string, []string, []int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string{}, f.comments...), append([]string{}, f.results...),
		append([]int{}, f.moved...), f.heartbeats
}

func (f *iapFake) claimCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.claims
}

// medFake 는 우리 쪽 Mediator 의 대역이다.
type medFake struct {
	mu sync.Mutex

	runs map[string]string // run_id -> RunView 본문 (없으면 404)
	// onSubmit 은 계약이 접수된 뒤에야 나타나는 Run 이다 — 실물이 그렇다.
	// 미리 심어 두면 nextRunID 의 탐침이 그것을 지난 세대로 읽고 한 칸
	// 위의 이름을 고르며, follow 가 없는 Run 을 영원히 조회한다.
	onSubmit     map[string]string
	submitStatus int
	asks         string
	ledger       string
	blob         string
	caps         string
	status       map[string]int
	failRunOnce  int // GetRun 을 이 횟수만큼 500 으로 떨어뜨린다

	submitted []string
	answers   []string
	outbound  []string
}

func (m *medFake) serve(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	raw, _ := io.ReadAll(r.Body)
	p := r.URL.Path
	for pre, code := range m.status {
		if strings.HasPrefix(p, pre) {
			w.WriteHeader(code)
			_, _ = w.Write([]byte("forced by the test"))
			return
		}
	}
	switch {
	case p == "/v1/runs" && r.Method == http.MethodPost:
		m.submitted = append(m.submitted, string(raw))
		if m.submitStatus != 0 {
			w.WriteHeader(m.submitStatus)
			_, _ = w.Write([]byte("worker: need 1, fleet has 0"))
			return
		}
		if m.runs == nil {
			m.runs = map[string]string{}
		}
		for k, v := range m.onSubmit {
			m.runs[k] = v
		}
		w.WriteHeader(http.StatusCreated)
	case p == "/v1/capabilities":
		_, _ = w.Write([]byte(def(m.caps, `{"capabilities":[]}`)))
	case p == "/v1/asks":
		_, _ = w.Write([]byte(def(m.asks, `{"asks":[]}`)))
	case strings.HasSuffix(p, "/answer"):
		m.answers = append(m.answers, string(raw))
		w.WriteHeader(http.StatusNoContent)
	case strings.Contains(p, "/steps/") && strings.Contains(p, "/blob/"):
		m.outbound = append(m.outbound, string(raw))
		w.WriteHeader(http.StatusCreated)
	case strings.Contains(p, "/blob/"):
		if m.blob == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(m.blob))
	case strings.HasSuffix(p, "/ledger"):
		_, _ = w.Write([]byte(def(m.ledger, `{"entries":[]}`)))
	case strings.HasPrefix(p, "/v1/runs/"):
		if m.failRunOnce > 0 {
			m.failRunOnce--
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		body, ok := m.runs[strings.TrimPrefix(p, "/v1/runs/")]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(body))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (m *medFake) snapshot() ([]string, []string, []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string{}, m.submitted...), append([]string{}, m.answers...),
		append([]string{}, m.outbound...)
}

// flowConfig 는 흐름 시험의 기본 설정이다.
//
// heartbeat_seconds 는 0 이면 안 된다 — time.NewTicker(0) 이 패닉이고,
// 실물에서는 LoadConfig 가 60 으로 채운다.
func flowConfig() *Config {
	return &Config{
		ItsAPlan: ItsAPlanConfig{ProjectKey: "EP", PollSeconds: 0, HeartbeatSeconds: 60},
		Mediator: MediatorConfig{SubmitWaitSeconds: 0},
		Transition: TransitionConfig{
			OnSuccess: "Done", OnFailure: "Blocked", OnAsk: "Waiting", OnStart: "Doing",
		},
	}
}

func newFlowAdapter(t *testing.T, f *iapFake, m *medFake, cfg *Config) *Adapter {
	t.Helper()
	is := httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(is.Close)
	ms := httptest.NewServer(http.HandlerFunc(m.serve))
	t.Cleanup(ms.Close)
	cfg.Mediator.URL = ms.URL
	return &Adapter{
		cfg:     cfg,
		iap:     NewIAP(is.URL, "agent-key"),
		med:     NewMediator(ms.URL, "fleet-token", cfg.Mediator.Principal),
		log:     quietLog(),
		columns: map[string]int{"Done": 11, "Blocked": 12, "Waiting": 13, "Doing": 14},
	}
}

func issueRun(id int, trigger string) *RunnerRun {
	issue := 2
	return &RunnerRun{ID: id, Trigger: trigger, IssueID: &issue, IssueIdentifier: "EP-2"}
}

func lastResult(t *testing.T, f *iapFake) map[string]any {
	t.Helper()
	_, results, _, _ := f.snapshot()
	if len(results) == 0 {
		t.Fatal("nothing was reported to the runner - the lease would simply expire")
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(results[len(results)-1]), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// 칸 이름을 id 로 바꾸기 위해 프로젝트를 읽는다 — project_key 가 없으면
// 옮길 수가 없고, 그것은 뜨는 시점에 알아야 한다.
func TestAdapter_LoadColumnsNeedsAProjectKey(t *testing.T) {
	t.Run("no project key", func(t *testing.T) {
		a := newFlowAdapter(t, &iapFake{}, &medFake{}, &Config{})
		err := a.loadColumns(context.Background())
		if err == nil || !strings.Contains(err.Error(), "project_key") {
			t.Fatalf("err = %v, want it to name itsaplan.project_key", err)
		}
	})
	t.Run("the project cannot be read", func(t *testing.T) {
		f := &iapFake{status: map[string]int{"/projects/": 500}}
		a := newFlowAdapter(t, f, &medFake{}, flowConfig())
		if err := a.loadColumns(context.Background()); err == nil {
			t.Fatal("an unreadable project was accepted")
		}
	})
	t.Run("the column names become ids", func(t *testing.T) {
		f := &iapFake{columns: `{"columns":[{"id":11,"name":"Done"},{"id":14,"name":"Doing"}]}`}
		a := newFlowAdapter(t, f, &medFake{}, flowConfig())
		a.columns = nil
		if err := a.loadColumns(context.Background()); err != nil {
			t.Fatal(err)
		}
		if a.columns["Done"] != 11 || a.columns["Doing"] != 14 {
			t.Fatalf("columns = %v", a.columns)
		}
	})
}

// 이름이 비었거나 모르면 안 옮긴다 — 모르는 이름으로 PATCH 를 내면
// 트래커가 거절하고 그 거절이 실패로 보고된다.
func TestAdapter_MoveOnlyMovesWhatItCanName(t *testing.T) {
	for _, tc := range []struct {
		name    string
		column  string
		known   map[string]int
		wantIDs []int
	}{
		{"an empty transition means do not move", "", map[string]int{"Done": 11}, nil},
		{"no column list at all", "Done", nil, nil},
		{"a name the project does not have", "Shipped", map[string]int{"Done": 11}, nil},
		{"a name it knows", "Done", map[string]int{"Done": 11}, []int{11}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &iapFake{}
			a := newFlowAdapter(t, f, &medFake{}, flowConfig())
			a.columns = tc.known
			a.move(context.Background(), 2, tc.column, quietLog())
			_, _, moved, _ := f.snapshot()
			if len(moved) != len(tc.wantIDs) {
				t.Fatalf("moved %v, want %v", moved, tc.wantIDs)
			}
			for i := range tc.wantIDs {
				if moved[i] != tc.wantIDs[i] {
					t.Fatalf("moved to %d, want %d", moved[i], tc.wantIDs[i])
				}
			}
		})
	}
}

// 실패해도 되돌릴 것이 없다 (ADR-040 §6) — 부분 성공이고 어댑터는 계속 간다.
func TestAdapter_MoveTreatsAFailedMoveAsPartialSuccess(t *testing.T) {
	f := &iapFake{status: map[string]int{"/issues/2": 500}}
	a := newFlowAdapter(t, f, &medFake{}, flowConfig())
	a.move(context.Background(), 2, "Done", quietLog())
	// 여기까지 왔다는 것이 계약이다 — 칸 이동 실패가 흐름을 끊지 않는다.
	if _, _, moved, _ := f.snapshot(); len(moved) != 0 {
		t.Fatalf("a rejected move was recorded as done: %v", moved)
	}
}

func TestAdapter_FailOutTellsTheIssueAndTheRunner(t *testing.T) {
	t.Run("a job attached to an issue", func(t *testing.T) {
		f := &iapFake{}
		a := newFlowAdapter(t, f, &medFake{}, flowConfig())
		a.failOut(context.Background(), issueRun(7, "delegation"), "the orchestrator did not start")

		comments, _, moved, _ := f.snapshot()
		if len(comments) != 1 || !strings.Contains(comments[0], "시작하지 못했습니다") {
			t.Fatalf("comments = %v", comments)
		}
		if !strings.Contains(comments[0], "the orchestrator did not start") {
			t.Fatalf("the reason never reached the issue: %v", comments[0])
		}
		if len(moved) != 1 || moved[0] != 12 {
			t.Fatalf("moved = %v, want the failure column", moved)
		}
		res := lastResult(t, f)
		if res["status"] != "failed" {
			t.Fatalf("status = %v, want failed", res["status"])
		}
		if res["error"] != "the orchestrator did not start" {
			t.Fatalf("error = %v", res["error"])
		}
	})

	t.Run("a job with no issue still closes the runner side", func(t *testing.T) {
		f := &iapFake{}
		a := newFlowAdapter(t, f, &medFake{}, flowConfig())
		a.failOut(context.Background(), &RunnerRun{ID: 7}, "no issue here")

		comments, _, moved, _ := f.snapshot()
		if len(comments) != 0 || len(moved) != 0 {
			t.Fatalf("it wrote to an issue it does not have: %v %v", comments, moved)
		}
		if lastResult(t, f)["status"] != "failed" {
			t.Fatal("the runner was left waiting for its lease to expire")
		}
	})
}

// 끝난 Run 을 밖으로 넘긴다 — 셋으로 갈린다 (ADR-040 §1):
// 기계가 읽는 것 · 사람이 읽는 것 · 보드가 읽는 것.
func TestAdapter_FinishSplitsIntoThree(t *testing.T) {
	for _, tc := range []struct {
		state   string
		column  int
		status  string
		wantErr bool
	}{
		{"SUCCEEDED", 11, "success", false},
		{"FAILED", 12, "failed", true},
		{"EXPIRED", 12, "failed", true},
	} {
		t.Run(tc.state, func(t *testing.T) {
			f := &iapFake{}
			a := newFlowAdapter(t, f, &medFake{}, flowConfig())
			run := runViewFrom(t, `{"run_id":"itsaplan-EP-2-1","state":"`+tc.state+`",
			  "verdict":{"state":"`+tc.state+`","checks":[{"step":"plan","what":"produced","ok":true}]}}`)

			a.finish(context.Background(), issueRun(7, "delegation"), run, quietLog())

			comments, _, moved, _ := f.snapshot()
			// ② 사람이 읽는 것 — 표지가 있어야 복구 경로가 중복을 안 만든다.
			if len(comments) != 1 || !strings.Contains(comments[0], ResultMarker("itsaplan-EP-2-1")) {
				t.Fatalf("the result comment is missing or unmarked: %v", comments)
			}
			// ③ 보드가 읽는 것
			if len(moved) != 1 || moved[0] != tc.column {
				t.Fatalf("moved = %v, want column %d", moved, tc.column)
			}
			// ① 기계가 읽는 것 — 우리 완주 판정이 그대로 사상된다
			res := lastResult(t, f)
			if res["status"] != tc.status {
				t.Fatalf("status = %v, want %v", res["status"], tc.status)
			}
			out, _ := res["output"].(string)
			if !strings.Contains(out, tc.state) || !strings.Contains(out, "1/1") {
				t.Fatalf("output = %q, want the one-line verdict", out)
			}
			if _, has := res["error"]; has != tc.wantErr {
				t.Fatalf("error present = %v, want %v", has, tc.wantErr)
			}
			if tc.wantErr && !strings.Contains(res["error"].(string), tc.state) {
				t.Fatalf("error = %v, want it to name the end state", res["error"])
			}
		})
	}
}

// 코멘트가 실패해도 판정은 넘어가야 한다 — 사람이 못 읽는 것과 기계가 못
// 읽는 것은 다른 손실이고, 하나가 다른 하나를 막으면 안 된다.
func TestAdapter_FinishSurvivesAFailedComment(t *testing.T) {
	f := &iapFake{status: map[string]int{"/issues/2/comments": 500}}
	a := newFlowAdapter(t, f, &medFake{}, flowConfig())
	run := &RunView{RunID: "itsaplan-EP-2-1", State: "SUCCEEDED"}

	a.finish(context.Background(), issueRun(7, "delegation"), run, quietLog())

	if lastResult(t, f)["status"] != "success" {
		t.Fatal("a failed comment swallowed the verdict")
	}
}

// 실행을 붙잡고 기다리면 실패로 끝난다 (integration §6.2 의 함정) —
// It's a Plan 의 리스는 300초이고 사람은 그보다 오래 걸린다. 그래서
// result 는 success 다: 실패가 아니라 손을 뗀 것 (ADR-040 §4).
func TestAdapter_HandOffLetsGoWithoutFailing(t *testing.T) {
	f := &iapFake{}
	m := &medFake{}
	a := newFlowAdapter(t, f, m, flowConfig())
	run := &RunView{RunID: "itsaplan-EP-2-1", State: "ASKED"}
	ask := &AskView{RunID: "itsaplan-EP-2-1", Seq: 4, Step: "gate", Prompt: "진행할까요"}

	a.handOff(context.Background(), issueRun(7, "delegation"), run, ask, quietLog())

	comments, _, moved, _ := f.snapshot()
	if len(comments) != 1 || !strings.Contains(comments[0], "확인이 필요합니다") {
		t.Fatalf("the question was not relayed: %v", comments)
	}
	if len(moved) != 1 || moved[0] != 13 {
		t.Fatalf("moved = %v, want the ask column", moved)
	}
	res := lastResult(t, f)
	if res["status"] != "success" {
		t.Fatalf("status = %v - letting go was reported as a failure", res["status"])
	}
	if out, _ := res["output"].(string); !strings.Contains(out, "itsaplan-EP-2-1") {
		t.Fatalf("output = %q, want the run it is waiting on", out)
	}

	// 밖에 남긴 것도 산출물이다 (ADR-040 §3.3) — 어댑터의 표가 아니라 blob 이다.
	_, _, outbound := m.snapshot()
	if len(outbound) != 1 {
		t.Fatalf("_outbound was not recorded: %v", outbound)
	}
	var ob map[string]any
	if err := json.Unmarshal([]byte(outbound[0]), &ob); err != nil {
		t.Fatal(err)
	}
	if ob["system"] != "itsaplan" || ob["step"] != "gate" || ob["seq"] != float64(4) {
		t.Fatalf("_outbound = %v", ob)
	}
	// 답을 맞추는 재료가 이 코멘트 번호다 (실측 ③ - 상관관계 식별자가 없다).
	if ob["comment"] != float64(1) || ob["issue"] != float64(2) {
		t.Fatalf("_outbound cannot be matched back to the comment: %v", ob)
	}
}

func TestAdapter_HandOffFailsWhenTheQuestionCannotBeRelayed(t *testing.T) {
	f := &iapFake{status: map[string]int{"/issues/2/comments": 500}}
	m := &medFake{}
	a := newFlowAdapter(t, f, m, flowConfig())
	run := &RunView{RunID: "itsaplan-EP-2-1", State: "ASKED"}
	ask := &AskView{RunID: "itsaplan-EP-2-1", Seq: 4, Step: "gate", Prompt: "진행할까요"}

	a.handOff(context.Background(), issueRun(7, "delegation"), run, ask, quietLog())

	res := lastResult(t, f)
	if res["status"] != "failed" {
		t.Fatalf("status = %v - a question nobody can see was reported as handed off", res["status"])
	}
	_, _, moved, _ := f.snapshot()
	if len(moved) != 0 {
		t.Fatalf("the issue was moved to the ask column with no question on it: %v", moved)
	}
	if _, _, outbound := m.snapshot(); len(outbound) != 0 {
		t.Fatalf("_outbound was recorded for a question that never went out: %v", outbound)
	}
}

// 막지 않는다 — 질문은 이미 나갔다. 다만 답을 맞출 재료가 약해진다.
func TestAdapter_HandOffDoesNotBlockOnAWeakOutboundRecord(t *testing.T) {
	f := &iapFake{}
	m := &medFake{status: map[string]int{"/v1/runs/itsaplan-EP-2-1/steps/": 500}}
	a := newFlowAdapter(t, f, m, flowConfig())
	run := &RunView{RunID: "itsaplan-EP-2-1", State: "ASKED"}
	ask := &AskView{RunID: "itsaplan-EP-2-1", Seq: 4, Step: "gate", Prompt: "진행할까요"}

	a.handOff(context.Background(), issueRun(7, "delegation"), run, ask, quietLog())

	if lastResult(t, f)["status"] != "success" {
		t.Fatal("a weak _outbound record turned a relayed question into a failure")
	}
	if _, _, moved, _ := f.snapshot(); len(moved) != 1 {
		t.Fatalf("moved = %v, want the ask column anyway", moved)
	}
}

// 표가 아니라 탐침이다 — 어댑터가 죽어도, 다시 떠도, 같은 규칙이 같은 답을 준다.
func TestAdapter_NextRunIDProbesInsteadOfKeepingATable(t *testing.T) {
	terminal := `{"run_id":"x","state":"SUCCEEDED"}`
	t.Run("a fresh issue starts at generation 1", func(t *testing.T) {
		a := newFlowAdapter(t, &iapFake{}, &medFake{}, flowConfig())
		id, err := a.nextRunID(context.Background(), "EP-2")
		if err != nil {
			t.Fatal(err)
		}
		if id != "itsaplan-EP-2-1" {
			t.Fatalf("run_id = %q", id)
		}
	})
	t.Run("finished generations are skipped", func(t *testing.T) {
		m := &medFake{runs: map[string]string{
			"itsaplan-EP-2-1": terminal, "itsaplan-EP-2-2": terminal}}
		a := newFlowAdapter(t, &iapFake{}, m, flowConfig())
		id, err := a.nextRunID(context.Background(), "EP-2")
		if err != nil {
			t.Fatal(err)
		}
		if id != "itsaplan-EP-2-3" {
			t.Fatalf("run_id = %q, want the first free generation", id)
		}
	})
	t.Run("one issue holds at most one open run", func(t *testing.T) {
		m := &medFake{runs: map[string]string{
			"itsaplan-EP-2-1": `{"run_id":"x","state":"RUNNING"}`}}
		a := newFlowAdapter(t, &iapFake{}, m, flowConfig())
		_, err := a.nextRunID(context.Background(), "EP-2")
		if err == nil {
			t.Fatal("a second run was started while the first was still open")
		}
		if !strings.Contains(err.Error(), "itsaplan-EP-2-1") || !strings.Contains(err.Error(), "RUNNING") {
			t.Fatalf("err = %v, want it to name the open run and its state", err)
		}
	})
	t.Run("a lookup failure is not a free name", func(t *testing.T) {
		m := &medFake{status: map[string]int{"/v1/runs/": 503}}
		a := newFlowAdapter(t, &iapFake{}, m, flowConfig())
		if _, err := a.nextRunID(context.Background(), "EP-2"); err == nil {
			t.Fatal("an unreadable Mediator handed out a run_id")
		}
	})
	t.Run("the probe has a ceiling", func(t *testing.T) {
		runs := map[string]string{}
		for n := 1; n <= 50; n++ {
			runs[fmt.Sprintf("itsaplan-EP-2-%d", n)] = terminal
		}
		m := &medFake{runs: runs}
		a := newFlowAdapter(t, &iapFake{}, m, flowConfig())
		_, err := a.nextRunID(context.Background(), "EP-2")
		if err == nil || !strings.Contains(err.Error(), "50 generations") {
			t.Fatalf("err = %v, want the ceiling", err)
		}
	})
}

func TestAdapter_LatestRunFindsTheNewestGeneration(t *testing.T) {
	t.Run("it walks up to the last one that exists", func(t *testing.T) {
		m := &medFake{runs: map[string]string{
			"itsaplan-EP-2-1": `{"run_id":"itsaplan-EP-2-1","state":"SUCCEEDED"}`,
			"itsaplan-EP-2-2": `{"run_id":"itsaplan-EP-2-2","state":"FAILED"}`,
		}}
		a := newFlowAdapter(t, &iapFake{}, m, flowConfig())
		id, run := a.latestRun(context.Background(), "EP-2")
		if id != "itsaplan-EP-2-2" || run == nil || run.State != "FAILED" {
			t.Fatalf("latestRun = %q %+v", id, run)
		}
	})
	t.Run("an issue with no runs yields nothing", func(t *testing.T) {
		a := newFlowAdapter(t, &iapFake{}, &medFake{}, flowConfig())
		id, run := a.latestRun(context.Background(), "EP-2")
		if id != "" || run != nil {
			t.Fatalf("latestRun = %q %+v, want nothing", id, run)
		}
	})
}

// 어댑터가 표를 안 든다 — 인박스를 훑고, run_id 가 이 이슈에서 유도된
// 이름인지로 가른다.
func TestAdapter_FindOpenAskBelongsToThisIssue(t *testing.T) {
	t.Run("the newest generation of this issue wins", func(t *testing.T) {
		m := &medFake{asks: `{"asks":[
		  {"run_id":"itsaplan-EP-3-1","seq":1,"step":"gate"},
		  {"run_id":"itsaplan-EP-2-1","seq":2,"step":"gate"},
		  {"run_id":"itsaplan-EP-2-2","seq":9,"step":"gate"}]}`}
		a := newFlowAdapter(t, &iapFake{}, m, flowConfig())
		runID, ask, err := a.findOpenAsk(context.Background(), "EP-2")
		if err != nil {
			t.Fatal(err)
		}
		if runID != "itsaplan-EP-2-2" || ask == nil || ask.Seq != 9 {
			t.Fatalf("findOpenAsk = %q %+v", runID, ask)
		}
	})
	t.Run("the prefix does not swallow a longer issue key", func(t *testing.T) {
		m := &medFake{asks: `{"asks":[{"run_id":"itsaplan-EP-20-1","seq":1,"step":"gate"}]}`}
		a := newFlowAdapter(t, &iapFake{}, m, flowConfig())
		runID, ask, err := a.findOpenAsk(context.Background(), "EP-2")
		if err != nil {
			t.Fatal(err)
		}
		if ask != nil {
			t.Fatalf("EP-2 picked up EP-20's question: %q %+v", runID, ask)
		}
	})
	t.Run("an unreadable inbox is not an empty inbox", func(t *testing.T) {
		m := &medFake{status: map[string]int{"/v1/asks": 500}}
		a := newFlowAdapter(t, &iapFake{}, m, flowConfig())
		if _, _, err := a.findOpenAsk(context.Background(), "EP-2"); err == nil {
			t.Fatal("an unreadable inbox was reported as having no open question")
		}
	})
}

// Mediator 와 노드는 어댑터를 모르므로 Run 은 어댑터 없이도 끝난다
// (adapter-example §2) — 그래서 끝난 것을 다시 찾아 넘긴다.
func TestAdapter_ReportUnreportedClosesTheGapTheAdapterLeft(t *testing.T) {
	const done = `{"run_id":"itsaplan-EP-2-1","state":"SUCCEEDED"}`
	marker := ResultMarker("itsaplan-EP-2-1")
	for _, tc := range []struct {
		name       string
		runs       map[string]string
		feed       string
		feedStatus int
		want       bool
		wantPost   bool
	}{
		{name: "no run on this issue at all", want: false},
		{
			name: "the run is still going",
			runs: map[string]string{"itsaplan-EP-2-1": `{"run_id":"itsaplan-EP-2-1","state":"RUNNING"}`},
		},
		{
			name: "the feed cannot be read, so duplication is unknown",
			runs: map[string]string{"itsaplan-EP-2-1": done}, feedStatus: 500,
		},
		{
			name: "it was already reported",
			runs: map[string]string{"itsaplan-EP-2-1": done},
			feed: `{"items":[{"kind":"comment","body":"` + marker + `"}]}`,
		},
		{
			name: "a finished run nobody reported",
			runs: map[string]string{"itsaplan-EP-2-1": done},
			want: true, wantPost: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &iapFake{feed: tc.feed}
			if tc.feedStatus != 0 {
				f.status = map[string]int{"/issues/2/feed": tc.feedStatus}
			}
			a := newFlowAdapter(t, f, &medFake{runs: tc.runs}, flowConfig())

			got := a.reportUnreported(context.Background(), issueRun(7, "mention"), "EP-2", quietLog())

			if got != tc.want {
				t.Fatalf("reportUnreported = %v, want %v", got, tc.want)
			}
			comments, results, _, _ := f.snapshot()
			if tc.wantPost {
				if len(comments) != 1 || len(results) != 1 {
					t.Fatalf("the finished run was not handed over: %v %v", comments, results)
				}
			} else if len(comments) != 0 || len(results) != 0 {
				t.Fatalf("it reported something it should not have: %v %v", comments, results)
			}
		})
	}
}

func TestAdapter_HandleAnswerWithNothingOpen(t *testing.T) {
	f := &iapFake{}
	a := newFlowAdapter(t, f, &medFake{}, flowConfig())

	a.handleAnswer(context.Background(), issueRun(7, "mention"))

	res := lastResult(t, f)
	// 실패가 아니다 — 답할 것이 없었을 뿐이다.
	if res["status"] != "success" {
		t.Fatalf("status = %v", res["status"])
	}
	if out, _ := res["output"].(string); !strings.Contains(out, "열린 질문이 없어") {
		t.Fatalf("output = %q", out)
	}
}

// 못 맞추면 질문은 열린 채 남는다 — 그래서 사람에게 형태를 다시 알려준다.
func TestAdapter_HandleAnswerCannotReadTheVerdict(t *testing.T) {
	f := &iapFake{}
	m := &medFake{asks: `{"asks":[{"run_id":"itsaplan-EP-2-1","seq":4,"step":"gate"}]}`}
	a := newFlowAdapter(t, f, m, flowConfig())
	rr := issueRun(7, "mention")
	rr.Prompt = "The comment that mentioned you: 음 글쎄요"

	a.handleAnswer(context.Background(), rr)

	comments, _, _, _ := f.snapshot()
	if len(comments) != 1 || !strings.Contains(comments[0], "`ok`") {
		t.Fatalf("the human was not told how to answer: %v", comments)
	}
	if _, answers, _ := m.snapshot(); len(answers) != 0 {
		t.Fatalf("an unreadable answer was submitted anyway: %v", answers)
	}
	if lastResult(t, f)["status"] != "success" {
		t.Fatal("an unreadable answer was reported as a runner failure")
	}
}

// 스키마 위반이면 422 이고 질문은 열린 채 남는다 — 그 사실이 사람에게
// 그대로 가야 다시 답할 수 있다.
func TestAdapter_HandleAnswerSurfacesARejectedAnswer(t *testing.T) {
	f := &iapFake{}
	m := &medFake{
		asks:   `{"asks":[{"run_id":"itsaplan-EP-2-1","seq":4,"step":"gate"}]}`,
		status: map[string]int{"/v1/runs/itsaplan-EP-2-1/steps/4/answer": 422},
	}
	a := newFlowAdapter(t, f, m, flowConfig())
	rr := issueRun(7, "mention")
	rr.Prompt = "The comment that mentioned you: ok"

	a.handleAnswer(context.Background(), rr)

	comments, _, _, _ := f.snapshot()
	if len(comments) != 1 || !strings.Contains(comments[0], "답을 계약에 넣지 못했습니다") {
		t.Fatalf("the rejection never reached the issue: %v", comments)
	}
	if lastResult(t, f)["status"] != "failed" {
		t.Fatal("a rejected answer was reported as success")
	}
}

// handle 은 mention 을 답 경로로, 그 밖을 새 일감 경로로 보낸다.
func TestAdapter_HandleRefusesAJobWithNoIssue(t *testing.T) {
	for _, tc := range []struct {
		name string
		rr   *RunnerRun
	}{
		{"no issue id", &RunnerRun{ID: 7, IssueIdentifier: "EP-2"}},
		{"no issue identifier", func() *RunnerRun { r := issueRun(7, "delegation"); r.IssueIdentifier = ""; return r }()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &iapFake{}
			a := newFlowAdapter(t, f, &medFake{}, flowConfig())
			a.handle(context.Background(), tc.rr)
			res := lastResult(t, f)
			if res["status"] != "failed" {
				t.Fatalf("status = %v", res["status"])
			}
			if e, _ := res["error"].(string); !strings.Contains(e, "issueId") {
				t.Fatalf("error = %q, want it to name what was missing", e)
			}
		})
	}
}

func TestAdapter_HandleRoutesMentionsToTheAnswerPath(t *testing.T) {
	f := &iapFake{}
	a := newFlowAdapter(t, f, &medFake{}, flowConfig())

	a.handle(context.Background(), issueRun(7, "mention"))

	if out, _ := lastResult(t, f)["output"].(string); !strings.Contains(out, "열린 질문이 없어") {
		t.Fatalf("a mention did not reach the answer path: %q", out)
	}
}

// 어댑터가 안에서 터져도 러너를 매달아 두지 않는다 — 안 그러면 리스가
// 만료될 때까지 같은 일감이 조용히 갇힌다.
func TestAdapter_HandleReportsItsOwnPanic(t *testing.T) {
	f := &iapFake{}
	is := httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(is.Close)
	// med 가 없으면 handleNewWork 의 첫 조회에서 터진다.
	a := &Adapter{cfg: flowConfig(), iap: NewIAP(is.URL, "k"), log: quietLog()}

	a.handle(context.Background(), issueRun(7, "delegation"))

	res := lastResult(t, f)
	if res["status"] != "failed" {
		t.Fatalf("status = %v", res["status"])
	}
	if e, _ := res["error"].(string); !strings.Contains(e, "어댑터 내부 오류") {
		t.Fatalf("error = %q, want the panic to be named", e)
	}
}

// 하트비트를 끝까지 돌린다 — 리스가 만료되면 같은 일감이 다시 나온다.
func TestAdapter_HeartbeatRenewsTheLeaseUntilItIsToldToStop(t *testing.T) {
	f := &iapFake{}
	cfg := flowConfig()
	cfg.ItsAPlan.HeartbeatSeconds = 1
	a := newFlowAdapter(t, f, &medFake{}, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { a.heartbeat(ctx, 7); close(done) }()

	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, _, _, hb := f.snapshot(); hb > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the lease was never renewed")
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the heartbeat outlived its job")
	}
}

// 하트비트가 실패해도 멈추지 않는다 — 한 번의 왕복 실패로 리스를 포기하면
// 일감이 통째로 다시 나온다.
func TestAdapter_HeartbeatKeepsGoingAfterAFailedRenewal(t *testing.T) {
	f := &iapFake{status: map[string]int{"/agent-runs/7/heartbeat": 500}}
	cfg := flowConfig()
	cfg.ItsAPlan.HeartbeatSeconds = 1
	a := newFlowAdapter(t, f, &medFake{}, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()
	a.heartbeat(ctx, 7)
	// 돌아왔다는 것이 계약이다 — 실패한 갱신이 고루틴을 죽이지 않았다.
}

func TestAdapter_RunDrainsTheQueue(t *testing.T) {
	t.Run("an empty queue with --once returns at once", func(t *testing.T) {
		f := &iapFake{}
		a := newFlowAdapter(t, f, &medFake{}, flowConfig())
		a.Run(context.Background(), true)
		a.wg.Wait()
		if _, results, _, _ := f.snapshot(); len(results) != 0 {
			t.Fatalf("an empty queue produced a report: %v", results)
		}
	})

	t.Run("a job is handled and then --once returns", func(t *testing.T) {
		f := &iapFake{queue: []string{`{"run":{"id":7,"trigger":"delegation"}}`}}
		a := newFlowAdapter(t, f, &medFake{}, flowConfig())
		a.Run(context.Background(), true)
		a.wg.Wait()
		res := lastResult(t, f)
		if res["status"] != "failed" {
			t.Fatalf("the pulled job was not handled: %v", res)
		}
	})

	t.Run("an already cancelled adapter pulls nothing", func(t *testing.T) {
		f := &iapFake{queue: []string{`{"run":{"id":7,"trigger":"delegation"}}`}}
		a := newFlowAdapter(t, f, &medFake{}, flowConfig())
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		a.Run(ctx, false)
		a.wg.Wait()
		if _, results, _, _ := f.snapshot(); len(results) != 0 {
			t.Fatalf("a cancelled adapter still handled a job: %v", results)
		}
	})

	// 한 번의 claim 실패로 어댑터가 멈추면 안 된다 — 트래커가 잠시 흔들린
	// 것뿐일 수 있고, 그때 큐에 남은 일감은 아무도 안 집는다.
	t.Run("a claim failure is retried until the adapter goes down", func(t *testing.T) {
		f := &iapFake{status: map[string]int{"/agent-runs/claim": 503}}
		cfg := flowConfig()
		cfg.ItsAPlan.PollSeconds = 1 // 0 이면 sleep 의 두 갈래가 경합한다
		a := newFlowAdapter(t, f, &medFake{}, cfg)
		ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
		defer cancel()
		a.Run(ctx, false)
		a.wg.Wait()
		if n := f.claimCount(); n < 2 {
			t.Fatalf("claimed %d times, want it to come back after the failure", n)
		}
	})

	// 빈 큐도 마찬가지다 — 한 번 비었다고 그만두면 어댑터가 한 번만 산다.
	t.Run("an empty queue without --once waits and comes back", func(t *testing.T) {
		f := &iapFake{}
		cfg := flowConfig()
		cfg.ItsAPlan.PollSeconds = 1
		a := newFlowAdapter(t, f, &medFake{}, cfg)
		ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
		defer cancel()
		a.Run(ctx, false)
		a.wg.Wait()
		if n := f.claimCount(); n < 2 {
			t.Fatalf("polled %d times, want it to poll again after an empty queue", n)
		}
	})
}

// follow 는 되묻기로 손을 떼면 true 를 돌려준다 — 그러면 오케스트레이터를
// 안 죽인다. 답이 오면 그 자리에서 이어야 하기 때문이다 (ADR-047).
func TestAdapter_FollowDecidesWhetherToKeepTheOrchestrator(t *testing.T) {
	t.Run("an adapter going down keeps the orchestrator alive", func(t *testing.T) {
		a := newFlowAdapter(t, &iapFake{}, &medFake{}, flowConfig())
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if !a.follow(ctx, issueRun(7, "delegation"), "itsaplan-EP-2-1", "EP-2", quietLog()) {
			t.Fatal("a live Run's orchestrator would be killed on shutdown")
		}
	})

	t.Run("a terminal run is handed over and the orchestrator is released", func(t *testing.T) {
		f := &iapFake{}
		m := &medFake{
			// 첫 조회는 못 읽는다 — 한 번의 실패가 추적을 끝내면 안 된다.
			failRunOnce: 1,
			runs: map[string]string{"itsaplan-EP-2-1": `{"run_id":"itsaplan-EP-2-1",
			  "state":"SUCCEEDED","verdict":{"state":"SUCCEEDED","checks":[]}}`},
		}
		a := newFlowAdapter(t, f, m, flowConfig())

		if a.follow(context.Background(), issueRun(7, "delegation"), "itsaplan-EP-2-1", "EP-2", quietLog()) {
			t.Fatal("a finished Run still held its orchestrator open")
		}
		comments, _, moved, _ := f.snapshot()
		if len(comments) != 1 || len(moved) != 1 || moved[0] != 11 {
			t.Fatalf("the finished Run was not handed over: %v %v", comments, moved)
		}
	})

	t.Run("an open question hands off and keeps the orchestrator", func(t *testing.T) {
		f := &iapFake{}
		m := &medFake{
			runs: map[string]string{"itsaplan-EP-2-1": `{"run_id":"itsaplan-EP-2-1","state":"ASKED"}`},
			asks: `{"asks":[{"run_id":"itsaplan-EP-2-1","seq":4,"step":"gate","prompt":"진행할까요"}]}`,
		}
		a := newFlowAdapter(t, f, m, flowConfig())

		if !a.follow(context.Background(), issueRun(7, "delegation"), "itsaplan-EP-2-1", "EP-2", quietLog()) {
			t.Fatal("the orchestrator was released while a human was still answering")
		}
		comments, _, moved, _ := f.snapshot()
		if len(comments) != 1 || !strings.Contains(comments[0], "확인이 필요합니다") {
			t.Fatalf("the question was not relayed: %v", comments)
		}
		if len(moved) != 1 || moved[0] != 13 {
			t.Fatalf("moved = %v, want the ask column", moved)
		}
	})
}

func TestHead_MarksWhereItCut(t *testing.T) {
	if got := head("short", 10); got != "short" {
		t.Fatalf("head = %q", got)
	}
	if got := head("exactly-10", 10); got != "exactly-10" {
		t.Fatalf("a string at the limit was cut: %q", got)
	}
	got := head(strings.Repeat("x", 30), 10)
	if !strings.HasPrefix(got, strings.Repeat("x", 10)) || !strings.HasSuffix(got, "…") {
		t.Fatalf("head = %q", got)
	}
}

// 기다림은 어댑터가 내려가면 끝난다 — 안 그러면 종료가 poll 간격만큼 늦고,
// claim 이 계속 실패하는 동안 고리를 못 벗어난다.
func TestSleep_StopsWhenTheAdapterGoesDown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if sleep(ctx, time.Minute) {
		t.Fatal("a cancelled adapter waited out its poll interval")
	}
	if !sleep(context.Background(), time.Millisecond) {
		t.Fatal("a normal wait reported a shutdown")
	}
}

// 이유는 계획을 다시 짓는 단계로 그대로 전달된다 — 그래서 상한이 있다.
func TestParseAnswer_ClipsAReasonThatWouldNotFit(t *testing.T) {
	long := strings.Repeat("가", 5000)
	v, note := parseAnswer("The comment that mentioned you: again " + long)
	if v != "again" {
		t.Fatalf("verdict = %q", v)
	}
	if len(note) != 4000 {
		t.Fatalf("note is %d bytes, want it clipped to 4000", len(note))
	}
}

// 함대가 끝내 안 비는데 어댑터가 내려가면 그 자리에서 그만둔다 — 상한까지
// 붙잡고 있으면 종료가 submit_wait_seconds 만큼 늦는다.
func TestAdapter_SubmitStopsWhenTheAdapterGoesDown(t *testing.T) {
	m := &medFake{submitStatus: 409}
	cfg := flowConfig()
	cfg.ItsAPlan.PollSeconds = 1
	cfg.Mediator.SubmitWaitSeconds = 600
	a := newFlowAdapter(t, &iapFake{}, m, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := a.submit(ctx, "itsaplan-EP-2-1", []byte(`{}`), quietLog())
	if err == nil {
		t.Fatal("a cancelled adapter kept resubmitting")
	}
}

// 열린 질문이 없다고 할 일이 없는 것은 아니다 — 어댑터가 죽거나 재시작하면
// 이미 끝난 Run 의 결과가 아직 안 나갔을 수 있다.
func TestAdapter_HandleAnswerReportsAFinishedRunInstead(t *testing.T) {
	f := &iapFake{}
	m := &medFake{runs: map[string]string{
		"itsaplan-EP-2-1": `{"run_id":"itsaplan-EP-2-1","state":"SUCCEEDED"}`}}
	a := newFlowAdapter(t, f, m, flowConfig())

	a.handleAnswer(context.Background(), issueRun(7, "mention"))

	comments, results, moved, _ := f.snapshot()
	if len(comments) != 1 || !strings.Contains(comments[0], ResultMarker("itsaplan-EP-2-1")) {
		t.Fatalf("the finished run was not handed over: %v", comments)
	}
	if len(moved) != 1 || moved[0] != 11 {
		t.Fatalf("moved = %v, want the success column", moved)
	}
	// 「열린 질문이 없었다」로 닫지 않는다 — 결과를 넘겼으므로 그것이 보고다.
	if len(results) != 1 {
		t.Fatalf("results = %v, want exactly the handover report", results)
	}
	if out, _ := lastResult(t, f)["output"].(string); strings.Contains(out, "열린 질문이 없어") {
		t.Fatalf("output = %q - the handover was reported as nothing to do", out)
	}
}

// 러너 보고가 실패해도 사람과 보드는 이미 받았다 — 하나가 다른 둘을
// 되돌리지 않는다 (ADR-040 §6).
func TestAdapter_ReportFailuresDoNotUndoWhatAlreadyWentOut(t *testing.T) {
	t.Run("finish", func(t *testing.T) {
		f := &iapFake{status: map[string]int{"/agent-runs/7/result": 500}}
		a := newFlowAdapter(t, f, &medFake{}, flowConfig())
		run := &RunView{RunID: "itsaplan-EP-2-1", State: "SUCCEEDED"}
		a.finish(context.Background(), issueRun(7, "delegation"), run, quietLog())
		comments, _, moved, _ := f.snapshot()
		if len(comments) != 1 || len(moved) != 1 {
			t.Fatalf("a failed runner report undid the comment or the move: %v %v", comments, moved)
		}
	})
	t.Run("handOff", func(t *testing.T) {
		f := &iapFake{status: map[string]int{"/agent-runs/7/result": 500}}
		m := &medFake{}
		a := newFlowAdapter(t, f, m, flowConfig())
		run := &RunView{RunID: "itsaplan-EP-2-1", State: "ASKED"}
		ask := &AskView{RunID: "itsaplan-EP-2-1", Seq: 4, Step: "gate", Prompt: "진행할까요"}
		a.handOff(context.Background(), issueRun(7, "delegation"), run, ask, quietLog())
		comments, _, moved, _ := f.snapshot()
		if len(comments) != 1 || len(moved) != 1 {
			t.Fatalf("a failed runner report undid the question or the move: %v %v", comments, moved)
		}
		if _, _, outbound := m.snapshot(); len(outbound) != 1 {
			t.Fatalf("_outbound = %v", outbound)
		}
	})
}

// 진행 중인데 아직 되묻기가 없는 것은 흔한 상태다 — 그때 손을 떼면
// 오케스트레이터가 살아 있는 채로 Run 이 버려진다.
func TestAdapter_FollowKeepsWatchingWhileThereIsNoQuestionYet(t *testing.T) {
	f := &iapFake{}
	m := &medFake{
		runs: map[string]string{"itsaplan-EP-2-1": `{"run_id":"itsaplan-EP-2-1","state":"RUNNING"}`},
	}
	a := newFlowAdapter(t, f, m, flowConfig())

	// 첫 바퀴에는 인박스가 비어 있다가 그 다음에 질문이 열린다.
	var once sync.Once
	base := m.serve
	is := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/asks" {
			opened := false
			once.Do(func() { opened = true })
			if opened {
				_, _ = w.Write([]byte(`{"asks":[]}`))
				return
			}
			_, _ = w.Write([]byte(`{"asks":[{"run_id":"itsaplan-EP-2-1","seq":4,"step":"gate"}]}`))
			return
		}
		base(w, r)
	}))
	t.Cleanup(is.Close)
	a.med = NewMediator(is.URL, "fleet-token", "")

	if !a.follow(context.Background(), issueRun(7, "delegation"), "itsaplan-EP-2-1", "EP-2", quietLog()) {
		t.Fatal("the adapter let go of a Run that was still running")
	}
	comments, _, _, _ := f.snapshot()
	if len(comments) != 1 {
		t.Fatalf("comments = %v, want exactly the question once it opened", comments)
	}
}

// 답이 들어가면 계약이 그 자리에서 이어진다 — 어댑터는 이슈를 다시 작업
// 칸으로 옮기고 Run 을 끝까지 따라간다. 답만 넣고 손을 떼면 그 Run 의
// 결과는 영영 이슈로 안 나간다.
func TestAdapter_HandleAnswerSubmitsAndPicksTheRunBackUp(t *testing.T) {
	f := &iapFake{}
	m := &medFake{
		asks: `{"asks":[{"run_id":"itsaplan-EP-2-1","seq":4,"step":"gate"}]}`,
		runs: map[string]string{"itsaplan-EP-2-1": `{"run_id":"itsaplan-EP-2-1",
		  "state":"SUCCEEDED","verdict":{"state":"SUCCEEDED","checks":[]}}`},
	}
	a := newFlowAdapter(t, f, m, flowConfig())
	rr := issueRun(7, "mention")
	rr.Prompt = "The comment that mentioned you: again 표를 더 자세히"

	a.handleAnswer(context.Background(), rr)

	_, answers, _ := m.snapshot()
	if len(answers) != 1 {
		t.Fatalf("answers = %v, want exactly one", answers)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(answers[0]), &body); err != nil {
		t.Fatal(err)
	}
	if body["verdict"] != "again" {
		t.Fatalf("verdict = %v", body["verdict"])
	}
	// 그 이유는 계획을 다시 짓는 단계로 그대로 전달된다.
	if body["note"] != "표를 더 자세히" {
		t.Fatalf("note = %v", body["note"])
	}

	comments, _, moved, _ := f.snapshot()
	// 작업 칸으로 되돌린 뒤, 끝나면 성공 칸으로 간다.
	if len(moved) != 2 || moved[0] != 14 || moved[1] != 11 {
		t.Fatalf("moved = %v, want start then success", moved)
	}
	if len(comments) != 1 || !strings.Contains(comments[0], ResultMarker("itsaplan-EP-2-1")) {
		t.Fatalf("the run was answered but never reported: %v", comments)
	}
}
