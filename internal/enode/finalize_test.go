package enode

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/taeels/enode/internal/contract"
)

// 명령이 끝난 뒤의 구간을 닫는 규칙 — finalize.go 의 순수 함수들이다.
//
// 규칙마다 표 한 줄이다. 흐름(goroutine · HTTP · 세션)은 finalize_worker_test.go 가
// 시험 서버와 가짜 세션으로 덮는다.

// effect 표 (business-rules.md 1절). diff 는 edit 이고 오늘 조건이 맞을 때만,
// 명시 훑기는 discover 일 때만, 지목 경로 stat 은 늘 한다.
func TestFinalizeSpecFor_FollowsTheEffectTable(t *testing.T) {
	ws := json.RawMessage(`{"ref":"main"}`)
	withWorkspace := Local{Workspace: "/ws"}
	cases := []struct {
		name      string
		step      *Step
		completed bool
		local     Local
		effect    contract.Effect
		diff      bool
		collect   bool
	}{
		{"run defaults to build and returns no diff",
			&Step{Kind: "run", Run: []string{"make"}, Workspace: ws}, true, withWorkspace, contract.EffectBuild, false, true},
		{"run read returns no diff",
			&Step{Kind: "run", Run: []string{"make"}, Workspace: ws, Effect: contract.EffectRead}, true, withWorkspace, contract.EffectRead, false, true},
		{"run edit keeps today's diff",
			&Step{Kind: "run", Run: []string{"fmt"}, Workspace: ws, Effect: contract.EffectEdit}, true, withWorkspace, contract.EffectEdit, true, true},
		{"agent defaults to edit and keeps today's diff",
			&Step{Kind: "agent", Workspace: ws}, true, withWorkspace, contract.EffectEdit, true, true},
		{"agent read returns no diff",
			&Step{Kind: "agent", Workspace: ws, Effect: contract.EffectRead}, true, withWorkspace, contract.EffectRead, false, true},
		{"edit without a node workspace returns no diff",
			&Step{Kind: "run", Run: []string{"fmt"}, Workspace: ws, Effect: contract.EffectEdit}, true, Local{}, contract.EffectEdit, false, true},
		{"edit without a contract workspace returns no diff",
			&Step{Kind: "run", Run: []string{"fmt"}, Effect: contract.EffectEdit}, true, withWorkspace, contract.EffectEdit, false, true},
		{"an agent that did not complete collects nothing and returns no diff",
			&Step{Kind: "agent", Workspace: ws}, false, withWorkspace, contract.EffectEdit, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.step.Collect = map[string]string{"artifact": "out/*.bin"}
			c.step.CheckChanged = []string{"a", "b"}
			spec := finalizeSpecFor(c.step, c.completed, c.local)
			if spec.Effect != c.effect {
				t.Errorf("effect = %q, want %q", spec.Effect, c.effect)
			}
			if spec.Diff != c.diff {
				t.Errorf("diff = %v, want %v", spec.Diff, c.diff)
			}
			if got := spec.Collect != nil; got != c.collect {
				t.Errorf("collect set = %v, want %v", got, c.collect)
			}
			if len(spec.Check) != 2 {
				t.Errorf("named paths must always be checked (ADR-037), got %v", spec.Check)
			}
			if spec.Discover {
				t.Error("discover must stay off unless the contract turns it on")
			}
		})
	}
}

// 명시 훑기는 effect 와 무관하고, agent 가 완주하지 못했어도 켜진다.
func TestFinalizeSpecFor_DiscoverFollowsTheContractOnly(t *testing.T) {
	for _, s := range []*Step{
		{Kind: "run", Run: []string{"make"}, Discover: true},
		{Kind: "agent", Discover: true, Effect: contract.EffectRead},
	} {
		for _, completed := range []bool{true, false} {
			if !finalizeSpecFor(s, completed, Local{}).Discover {
				t.Errorf("kind %s completed %v: discover was requested but is off", s.Kind, completed)
			}
		}
	}
}

// 예산 — 계약이 안 적으면 기본값, 적으면 그 값. 옛 Mediator 는 칸을 안 싣는다.
func TestStepBudgets_DefaultsAndContractValues(t *testing.T) {
	f, u := stepBudgets(&Step{Kind: "run", Run: []string{"make"}})
	if f != time.Minute || u != 3*time.Minute {
		t.Errorf("defaults = %v · %v, want 1m · 3m", f, u)
	}
	f, u = stepBudgets(&Step{Kind: "run", Run: []string{"make"},
		Budget: &contract.Budget{Finalize: "5m", Upload: "10m"}})
	if f != 5*time.Minute || u != 10*time.Minute {
		t.Errorf("contract values = %v · %v, want 5m · 10m", f, u)
	}
	f, u = stepBudgets(&Step{Kind: "agent", Budget: &contract.Budget{Upload: "90s"}})
	if f != time.Minute || u != 90*time.Second {
		t.Errorf("partial budget = %v · %v, want 1m · 1m30s", f, u)
	}
}

// 옛 Mediator 의 claim 응답 — effect · budget · discover 가 없다. 풀면 기본값이다.
func TestStep_AnOldClaimDecodesToDefaults(t *testing.T) {
	var s Step
	if err := json.Unmarshal([]byte(`{"run_id":"r1","seq":1,"kind":"run","run":["make"]}`), &s); err != nil {
		t.Fatal(err)
	}
	spec := finalizeSpecFor(&s, true, Local{Workspace: "/ws"})
	if spec.Effect != contract.EffectBuild || spec.Diff || spec.Discover {
		t.Errorf("old claim = effect %q diff %v discover %v, want build · false · false",
			spec.Effect, spec.Diff, spec.Discover)
	}
	var n Step
	if err := json.Unmarshal([]byte(`{"run_id":"r1","seq":1,"kind":"run","run":["make"],
		"effect":"edit","budget":{"finalize":"2m"},"discover":true}`), &n); err != nil {
		t.Fatal(err)
	}
	if n.Effect != contract.EffectEdit || n.Budget == nil || n.Budget.Finalize != "2m" || !n.Discover {
		t.Errorf("new claim did not decode: %+v %+v %v", n.Effect, n.Budget, n.Discover)
	}
}

// ── 종료 보고 ────────────────────────────────────────────────────

// 보내는 조건 (business-rules.md 4.1). native 의 signal 은 진짜 프로세스가 필요해
// worker_unix_test.go 에 있다.
func TestExitOutcome(t *testing.T) {
	cases := []struct {
		name   string
		code   int
		err    error
		ok     bool
		kind   string
		wantNo int
	}{
		{"exit 0", 0, nil, true, contract.OutcomeExit, 0},
		{"exit 1 is still an exit", 1, errors.New("exit status 1"), true, contract.OutcomeExit, 1},
		{"overlay reports a signal as 128+n", 137, nil, true, contract.OutcomeExit, 137},
		{"a process that never started", -1, errors.New("exec: not found"), false, "", 0},
		{"no reason at all", -1, nil, false, "", 0},
	}
	for _, c := range cases {
		o, ok := exitOutcome(c.code, c.err)
		if ok != c.ok || o.Kind != c.kind || (ok && *o.Code != c.wantNo) {
			t.Errorf("%s: outcome %+v ok %v", c.name, o, ok)
		}
		if ok {
			e := contract.Exited{Node: "n", Instance: "i", Outcome: o, ExitedAt: time.Now()}
			if err := e.Check(); err != nil {
				t.Errorf("%s: the mediator would reject it: %v", c.name, err)
			}
		}
	}
}

// 응답마다 멈출지와 로그 (business-rules.md 4.2).
func TestExitedStopAndLog(t *testing.T) {
	cases := []struct {
		code  int
		stop  bool
		level slog.Level
		msg   string
	}{
		{200, true, slog.LevelDebug, "exit reported"},
		{400, true, slog.LevelError, "exit report was malformed; not retrying"},
		{404, true, slog.LevelInfo, "the mediator does not accept exit reports for this step; not retrying"},
		{409, true, slog.LevelWarn, "exit report rejected; not retrying"},
		{410, true, slog.LevelWarn, "exit report rejected; not retrying"},
		{503, false, slog.LevelDebug, "exit report failed; retrying"},
		{0, false, slog.LevelDebug, "exit report failed; retrying"},
	}
	for _, c := range cases {
		if c.code != 0 && exitedStop(c.code) != c.stop {
			t.Errorf("%d: stop = %v", c.code, !c.stop)
		}
		if level, msg := exitReportLog(c.code); level != c.level || msg != c.msg {
			t.Errorf("%d: log = %v %q", c.code, level, msg)
		}
	}
}

func TestExitBackoff(t *testing.T) {
	want := []time.Duration{1, 2, 4, 8, 10, 10, 10}
	for n, w := range want {
		if got := exitBackoff(n); got != w*time.Second {
			t.Errorf("n=%d: %v, want %v", n, got, w*time.Second)
		}
	}
}

// exitServer 는 종료 보고만 받는 시험 서버다. codes 를 차례로 답하고 다 쓰면 마지막을 되풀이한다.
type exitServer struct {
	mu     sync.Mutex
	codes  []int
	bodies []contract.Exited
	hang   chan struct{}
}

func newExitServer(t *testing.T, codes ...int) (*exitServer, *Worker) {
	t.Helper()
	es := &exitServer{codes: codes}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/v1/runs/r1/steps/1/exited") || r.Method != "POST" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var e contract.Exited
		body, _ := io.ReadAll(r.Body) // 다 읽어야 끊긴 연결이 r.Context() 에 닿는다
		_ = json.Unmarshal(body, &e)
		es.mu.Lock()
		es.bodies = append(es.bodies, e)
		code := es.codes[0]
		if len(es.codes) > 1 {
			es.codes = es.codes[1:]
		}
		hang := es.hang
		es.mu.Unlock()
		if hang != nil {
			select {
			case <-hang:
			case <-r.Context().Done():
			}
		}
		w.WriteHeader(code)
	}))
	t.Cleanup(srv.Close)
	w := &Worker{
		Client: &Client{Base: srv.URL, Token: "t", Principal: "p", Instance: "inst-1", HTTP: srv.Client()},
		Ident:  Identity{NodeID: "n1"}, Held: NewHeld(), Log: discardLog(),
		exitWait: func(int) time.Duration { return time.Millisecond },
	}
	return es, w
}

func (es *exitServer) sent() []contract.Exited {
	es.mu.Lock()
	defer es.mu.Unlock()
	return append([]contract.Exited(nil), es.bodies...)
}

func exitBody() contract.Exited {
	code := 1
	return contract.Exited{Node: "n1", Instance: "inst-1", Attempt: 2,
		Outcome: contract.Outcome{Kind: contract.OutcomeExit, Code: &code}, ExitedAt: time.Now().UTC()}
}

func waitSent(t *testing.T, es *exitServer, n int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for len(es.sent()) < n {
		if time.Now().After(deadline) {
			t.Fatalf("sent %d reports, want %d", len(es.sent()), n)
		}
		time.Sleep(time.Millisecond)
	}
}

// 200 · 404 · 409 에서 멈추고, 5xx 에서 다시 보낸다.
func TestExitReporter_StopsOnAnAnswerAndRetriesOnFailure(t *testing.T) {
	for _, c := range []struct {
		name  string
		codes []int
		want  int
	}{
		{"accepted", []int{200}, 1},
		{"old mediator", []int{404}, 1},
		{"rejected", []int{409}, 1},
		{"retried until accepted", []int{503, 502, 200}, 3},
	} {
		t.Run(c.name, func(t *testing.T) {
			es, w := newExitServer(t, c.codes...)
			r := w.startExitReport(context.Background(), &Step{RunID: "r1", Seq: 1, StepID: "r1#01"}, exitBody())
			waitSent(t, es, c.want)
			<-r.done // 스스로 끝난다
			r.Stop()
			if got := len(es.sent()); got != c.want {
				t.Fatalf("sent %d, want %d", got, c.want)
			}
			if err := es.sent()[0].Check(); err != nil {
				t.Fatalf("the body would be rejected: %v", err)
			}
		})
	}
}

// result 를 보내기 직전의 Stop 은 도는 요청과 재전송을 그만둔다 — Mediator 가 답을
// 늦춰도 result 가 기다리지 않는다.
func TestExitReporter_StopEndsAHangingRequest(t *testing.T) {
	es, w := newExitServer(t, 200)
	es.hang = make(chan struct{})
	defer close(es.hang)
	r := w.startExitReport(context.Background(), &Step{RunID: "r1", Seq: 1, StepID: "r1#01"}, exitBody())
	waitSent(t, es, 1)
	began := time.Now()
	r.Stop()
	r.Stop() // 두 번 불러도 된다
	if took := time.Since(began); took > time.Second {
		t.Fatalf("stop waited %v for a hanging mediator", took)
	}
}

// instance 가 없는 Client 는 보내지 않는다 — Mediator 가 400 으로 돌려줄 것이다.
func TestExitReporter_NoInstanceNoReport(t *testing.T) {
	es, w := newExitServer(t, 200)
	w.Client.Instance = ""
	r := w.startExitReport(context.Background(), &Step{RunID: "r1", Seq: 1}, exitBody())
	r.Stop()
	if r != nil || len(es.sent()) != 0 {
		t.Fatalf("a report went out without an instance: %v", es.sent())
	}
}

// ── 판정 칸 · 진단 · 로그 끝 ─────────────────────────────────────────

// settle — 경로마다 한 줄 (business-rules.md 3.2 · 9절).
func TestSettle(t *testing.T) {
	base := settleIn{upload: contract.StageOK, finalizeBudget: time.Minute, uploadBudget: 3 * time.Minute}
	with := func(f func(*settleIn)) settleIn { in := base; f(&in); return in }
	cases := []struct {
		name         string
		in           settleIn
		fin, up      contract.Stage
		reason, text string
	}{
		{"normal", base, contract.StageOK, contract.StageOK, "", ""},
		{"finalize error", with(func(i *settleIn) { i.finalizeErr = errors.New("merged view is gone") }),
			contract.StageError, contract.StageOK, "", "runtime finalize: merged view is gone"},
		{"cleanup error", with(func(i *settleIn) { i.closeErr = errors.New("busy") }),
			contract.StageError, contract.StageOK, "", "runtime cleanup: busy"},
		{"finalize over budget", with(func(i *settleIn) { i.finalizeErr = context.DeadlineExceeded }),
			contract.StageTimeout, contract.StageOK, contract.ReasonFinalizeTimeout, "finalize budget of 1m0s exceeded"},
		{"both over budget", with(func(i *settleIn) {
			i.finalizeErr, i.upload = context.DeadlineExceeded, contract.StageTimeout
		}), contract.StageTimeout, contract.StageTimeout, contract.ReasonFinalizeTimeout,
			"finalize budget of 1m0s exceeded; upload budget of 3m0s exceeded"},
		{"upload over budget", with(func(i *settleIn) { i.upload = contract.StageTimeout }),
			contract.StageOK, contract.StageTimeout, contract.ReasonUploadTimeout, "upload budget of 3m0s exceeded"},
		{"a transfer failed", with(func(i *settleIn) { i.upload = contract.StageError }),
			contract.StageOK, contract.StageError, "", ""},
		{"the lease ended during finalize", with(func(i *settleIn) {
			i.finalizeErr, i.leaseEnded = context.DeadlineExceeded, true
		}), contract.StageError, contract.StageOK, "", "runtime finalize: context deadline exceeded"},
		{"the lease ended during upload", with(func(i *settleIn) {
			i.upload, i.leaseEnded = contract.StageTimeout, true
		}), contract.StageOK, contract.StageError, "", ""},
	}
	for _, c := range cases {
		fin, up, reason, text := settle(c.in)
		if fin != c.fin || up != c.up || reason != c.reason || text != c.text {
			t.Errorf("%s: got %q %q %q %q, want %q %q %q %q",
				c.name, fin, up, reason, text, c.fin, c.up, c.reason, c.text)
		}
	}
}

// changes 의 값 넷 (business-rules.md 5.3). not_measured 는 「바뀐 파일이 없다」가 아니다.
func TestDiagnosticsFor_Changes(t *testing.T) {
	out := t.TempDir()
	step := &Step{Kind: "run", Run: []string{"make"}}
	cases := []struct {
		name  string
		disc  *Discovery
		want  string
		limit string
		paths int
	}{
		{"discover off", nil, contract.ChangesNotMeasured, "", 0},
		{"no workspace", &Discovery{Skipped: discoverNoWorkspace}, contract.ChangesNotMeasured, "", 0},
		{"stopped at a limit", &Discovery{Total: 9, Paths: []Changed{{Path: "a"}}, Limit: contract.LimitVisits},
			contract.ChangesPartial, contract.LimitVisits, 1},
		{"walked it all", &Discovery{Total: 2, Paths: []Changed{{Path: "a"}, {Path: "b"}}},
			contract.ChangesMeasured, "", 2},
	}
	for _, c := range cases {
		d := diagnosticsFor(step, out, contract.EffectBuild, FinalizeResult{Discovery: c.disc})
		if d.Changes != c.want || d.DiscoveryLimit != c.limit || len(d.Discovered) != c.paths || d.Effect != contract.EffectBuild {
			t.Errorf("%s: %+v", c.name, d)
		}
	}
	b, _ := json.Marshal(diagnosticsFor(step, out, contract.EffectBuild, FinalizeResult{}))
	if !strings.Contains(string(b), `"changes":"not_measured"`) {
		t.Errorf("changes must always be on the wire: %s", b)
	}
}

// missing 은 $OUT 을 직접 읽는다. .part 로 끝나는 반쪽은 그 이름이 아니다.
func TestDiagnosticsFor_MissingAndCollect(t *testing.T) {
	out := t.TempDir()
	for _, n := range []string{"kept", "half.part"} {
		if err := os.WriteFile(filepath.Join(out, n), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	step := &Step{Kind: "run", Run: []string{"make"}, Out: []string{"kept", "half", "never"}}
	d := diagnosticsFor(step, out, contract.EffectBuild, FinalizeResult{
		Notes: []HarvestNote{{"half", "no file matches \"x\""}}})
	if strings.Join(d.Missing, ",") != "half,never" {
		t.Errorf("missing = %v", d.Missing)
	}
	if len(d.Collect) != 1 || d.Collect[0].Name != "half" {
		t.Errorf("collect = %+v", d.Collect)
	}
}

// 단계 로그 끝의 줄 (business-rules.md 6.3 · 7절).
func TestLogTail(t *testing.T) {
	run := &Step{Kind: "run", Run: []string{"make"}}
	edit := &Step{Kind: "run", Run: []string{"fmt"}, Effect: contract.EffectEdit}
	agent := &Step{Kind: "agent"}
	discovering := &Step{Kind: "agent", Discover: true}
	cases := []struct {
		name    string
		step    *Step
		diag    *contract.Diagnostics
		fin     FinalizeResult
		timeout bool
		want    []string
		not     []string
	}{
		{"the default run step is told what changed", run, &contract.Diagnostics{}, FinalizeResult{}, false,
			[]string{"enode: " + noticeRunDefault}, []string{"agent steps"}},
		{"a step that wrote its effect is not told", edit, &contract.Diagnostics{}, FinalizeResult{}, false,
			nil, []string{"default effect", "agent steps"}},
		{"agent steps are told about workspace.changed", agent, &contract.Diagnostics{}, FinalizeResult{}, false,
			[]string{"enode: " + noticeAgentChanged}, []string{"default effect"}},
		{"missing · collect · diff", edit, &contract.Diagnostics{
			Missing: []string{"a", "b"}, Collect: []contract.CollectNote{{Name: "a", Why: "no file matches"}}},
			FinalizeResult{DiffError: "git: not a repository"}, false, []string{
				"enode: required by the contract but missing from $OUT: a, b\n",
				"enode: collect could not gather a: no file matches\n",
				"enode: workspace.diff was not produced: git: not a repository\n"}, nil},
		{"a full walk with deletions", discovering, &contract.Diagnostics{}, FinalizeResult{Discovery: &Discovery{
			Total: 2, Deleted: 3, Paths: []Changed{{Path: "vmlinux", Size: 2 << 20}, {Path: "x.o", Size: 10}}}}, false,
			[]string{"enode: discover listed 2 files created or modified by this step, and 3 deletions\n",
				"enode:   vmlinux  2.0 MiB\n", "enode:   x.o  10 B\n"}, []string{"partial", "no longer return"}},
		{"a partial walk", discovering, &contract.Diagnostics{}, FinalizeResult{Discovery: &Discovery{
			Total: 7, Limit: contract.LimitTime, Paths: []Changed{{Path: "a"}}}}, false,
			[]string{"discover listed 7 files created or modified by this step before it stopped at the time limit; the list is partial\n"}, nil},
		{"no workspace", discovering, &contract.Diagnostics{}, FinalizeResult{Discovery: &Discovery{Skipped: discoverNoWorkspace}}, false,
			[]string{"enode: discover was requested but this node has no workspace directory; nothing was listed\n"}, nil},
		{"over the finalize budget", edit, &contract.Diagnostics{}, FinalizeResult{}, true,
			[]string{"enode: finalize budget of 1m0s exceeded\n"}, nil},
	}
	for _, c := range cases {
		got := logTail(c.step, c.diag, c.fin, c.timeout, time.Minute)
		for _, w := range c.want {
			if !strings.Contains(got, w) {
				t.Errorf("%s: missing %q in\n%s", c.name, w, got)
			}
		}
		for _, n := range c.not {
			if strings.Contains(got, n) {
				t.Errorf("%s: unexpected %q in\n%s", c.name, n, got)
			}
		}
		if strings.Contains(got, "no files changed") {
			t.Errorf("%s: FR-1 violated — an unmeasured list reported as no change:\n%s", c.name, got)
		}
	}
}

// 목록은 스무 줄까지다. 나머지는 diagnostics 칸에 있다.
func TestLogTail_ListsAtMostTwentyPaths(t *testing.T) {
	var paths []Changed
	for i := 0; i < 50; i++ {
		paths = append(paths, Changed{Path: "p", Size: 1})
	}
	got := logTail(&Step{Kind: "agent", Discover: true}, &contract.Diagnostics{},
		FinalizeResult{Discovery: &Discovery{Total: 50, Paths: paths}}, false, time.Minute)
	if n := strings.Count(got, "enode:   p  1 B"); n != logTailList {
		t.Fatalf("listed %d paths, want %d", n, logTailList)
	}
}

func TestWithTail(t *testing.T) {
	if got := string(withTail([]byte("no newline"), "enode: x\n")); got != "no newline\nenode: x\n" {
		t.Errorf("got %q", got)
	}
	if got := string(withTail([]byte("line\n"), "enode: x\n")); got != "line\nenode: x\n" {
		t.Errorf("got %q", got)
	}
	if got := string(withTail(nil, "enode: x\n")); got != "enode: x\n" {
		t.Errorf("got %q", got)
	}
}

// diff 는 edit 일 때만 만들고, 마감은 각 자리 뒤에서 본다 (runtime.go 의 finalizeLocal).
func TestFinalizeLocal_DiffAndTheDeadline(t *testing.T) {
	ws, out := gitInit(t), t.TempDir()
	write(t, ws, "keep.c", "int x = 2;\n")
	res, err := (&nativeSession{}).Finalize(context.Background(), FinalizeSpec{Workspace: ws, Out: out, Diff: true})
	if err != nil || res.DiffBytes == 0 {
		t.Fatalf("an edit step lost its diff: %+v %v", res, err)
	}
	if _, err := os.Stat(filepath.Join(out, "workspace.diff")); err != nil {
		t.Fatalf("workspace.diff is not in $OUT: %v", err)
	}
	notGit := t.TempDir()
	res, _ = (&nativeSession{}).Finalize(context.Background(), FinalizeSpec{Workspace: notGit, Out: t.TempDir(), Diff: true})
	if res.DiffError == "" {
		t.Fatal("a failed diff was not written down")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err = (&nativeSession{}).Finalize(ctx, FinalizeSpec{Workspace: ws, Out: t.TempDir(), Diff: true,
		Check: []string{"keep.c"}, Discover: true, Stamp: Stamp{Root: ws}})
	if !errors.Is(err, context.Canceled) || res.Changed != nil || res.DiffBytes != 0 || res.Discovery != nil {
		t.Fatalf("a finished budget still worked: %+v %v", res, err)
	}
}

// 노드 로그 — finalized 한 줄과 사실마다의 경고.
func TestLogFinalize(t *testing.T) {
	var buf strings.Builder
	log := slog.New(slog.NewTextHandler(&buf, nil))
	spec := FinalizeSpec{Effect: contract.EffectEdit, Check: []string{"a", "b"}, Discover: true}
	fin := FinalizeResult{Collected: []string{"k"}, Notes: []HarvestNote{{"x", "no file matches"}},
		DiffError: "not a git repository", Discovery: &Discovery{Visited: 42, Limit: contract.LimitVisits}}
	logFinalize(log, spec, fin, &contract.Diagnostics{Missing: []string{"x"}}, 1500*time.Millisecond)
	got := buf.String()
	for _, w := range []string{"msg=finalized", "effect=edit", "checked=2", "discover=true", "visited=42",
		"limit=visits", "took=1.5s", "required outputs are missing", "collect failed", "cannot collect workspace diff",
		"msg=collected"} {
		if !strings.Contains(got, w) {
			t.Errorf("%q missing in\n%s", w, got)
		}
	}
	buf.Reset()
	logFinalize(log, spec, FinalizeResult{DiffBytes: 10}, &contract.Diagnostics{}, time.Millisecond)
	if !strings.Contains(buf.String(), "msg=\"workspace diff\" bytes=10") {
		t.Errorf("the diff size is not logged:\n%s", buf.String())
	}
}

func TestBlobRejected_Error(t *testing.T) {
	if got := (&BlobRejected{Status: "413 Request Entity Too Large"}).Error(); got != "413 Request Entity Too Large" {
		t.Errorf("got %q", got)
	}
	if got := (&BlobRejected{Status: "422", Body: "schema"}).Error(); got != "422: schema" {
		t.Errorf("got %q", got)
	}
}
