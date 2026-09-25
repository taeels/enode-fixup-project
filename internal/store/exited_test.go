package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/taeels/enode/internal/contract"
)

// 이 파일은 step-phase 유닛의 저장소 쪽을 확인한다 — 종료 보고의 수락 표,
// phase 의 한 생애, 되돌림 셋, Record 의 새 칸, 어휘 밖 값, Claimed 의 계약 칸.
// 판은 obsStore 로 판다 — 이 패키지의 다른 시험과 같은 이유다.

const xInst = "life-1"

// xRun 은 계약 c 의 모든 역할을 노드 하나에 앉힌 RUNNING Run 이다.
// 임대는 노드마다 하나라(leases 의 기본 키) Run 마다 노드를 따로 준다.
func xRun(t *testing.T, st *Store, runID, node string, c contract.Contract) {
	t.Helper()
	c.RunID = runID
	var as []Assigned
	for _, r := range c.Requires {
		as = append(as, Assigned{As: r.As, Nodes: []NodeRef{{Node: node, Label: node + " label"}}})
	}
	r := Run{RunID: runID, State: StateRunning, Principal: "p", Contract: c, Assigned: as}
	grants := []LeaseGrant{{NodeID: node, NotAfter: time.Now().Add(time.Hour), Nonce: "n"}}
	if err := st.CreateRun(context.Background(), r, grants, c.Steps); err != nil {
		t.Fatalf("cannot create run %s: %v", runID, err)
	}
}

// xTwoSteps 는 명령 단계 둘(s1 · s2)이 이어진 계약이다.
func xTwoSteps() contract.Contract {
	return contract.Contract{
		Requires: []contract.Require{{As: "b", Capability: contract.CapabilityAgentReason}},
		Steps: []contract.Step{
			{ID: "s1", Uses: "b", Run: []string{"true"}},
			{ID: "s2", Uses: "b", Run: []string{"true"}},
		},
	}
}

// xBake 는 제품의 굽기 예시 그대로다 (build · merge).
func xBake(t *testing.T) contract.Contract {
	t.Helper()
	raw, err := contract.Example("bake")
	if err != nil {
		t.Fatal(err)
	}
	var c contract.Contract
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatal(err)
	}
	return c
}

func xClaim(t *testing.T, st *Store, node, inst string) *Claimed {
	t.Helper()
	c, err := st.ClaimStep(context.Background(), node, inst)
	if err != nil {
		t.Fatalf("claim by %s: %v", node, err)
	}
	return c
}

func xReport(t *testing.T, st *Store, runID string, seq int, node string, code int, extra StepResult) {
	t.Helper()
	extra.ExitCode = &code
	if _, err := st.ReportStep(context.Background(), runID, seq, node, true, extra); err != nil {
		t.Fatalf("report %s#%d: %v", runID, seq, err)
	}
}

func xExit(node, inst string, attempt, code int, at time.Time) contract.Exited {
	return contract.Exited{Node: node, Instance: inst, Attempt: attempt,
		Outcome: contract.Outcome{Kind: contract.OutcomeExit, Code: &code}, ExitedAt: at}
}

// xRow 는 한 단계의 새 칸 셋이다. NULL 은 빈 값과 nil 로 온다.
type xRow struct {
	Phase string
	Since *time.Time
	Exit  []byte
}

func xRead(t *testing.T, st *Store, runID string, seq int) xRow {
	t.Helper()
	var r xRow
	var phase *string
	if err := st.pool.QueryRow(context.Background(),
		`SELECT phase, phase_since, exit FROM steps WHERE run_id=$1 AND seq=$2`, runID, seq).
		Scan(&phase, &r.Since, &r.Exit); err != nil {
		t.Fatalf("read %s#%d: %v", runID, seq, err)
	}
	if phase != nil {
		r.Phase = *phase
	}
	return r
}

func xExec(t *testing.T, st *Store, sql string, args ...any) {
	t.Helper()
	if _, err := st.pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

// 수락 표 열셋 — 위에서부터 처음 맞는 줄이 결과다 (step-phase FD 규칙 1절).
// 받지 않은 줄은 칸 셋이 그대로인지도 본다.
func TestMarkExited_TheAcceptanceTable(t *testing.T) {
	st := obsStore(t)
	ctx := context.Background()
	at := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)

	cases := []struct {
		name string
		// setup 은 Run 을 세우고 보낼 (run, seq, 본문) 을 준다.
		setup    func(t *testing.T, run, node string) (string, int, contract.Exited)
		accepted bool
		err      error
		reason   string // 409 의 문구 전체
		// phase 는 보고 뒤 그 단계의 phase 다. 빈 값이면 보지 않는다.
		phase string
	}{
		{name: "1 the run does not exist", err: ErrNotFound,
			setup: func(t *testing.T, run, node string) (string, int, contract.Exited) {
				return "no-such-run", 1, xExit(node, xInst, 0, 0, at)
			}},
		{name: "2 the step does not exist", err: ErrNoSuchStep,
			setup: func(t *testing.T, run, node string) (string, int, contract.Exited) {
				xRun(t, st, run, node, xTwoSteps())
				return run, 9, xExit(node, xInst, 0, 0, at)
			}},
		{name: "3 the run is cancelled", phase: PhaseRunning,
			setup: func(t *testing.T, run, node string) (string, int, contract.Exited) {
				xRun(t, st, run, node, xTwoSteps())
				xClaim(t, st, node, xInst)
				if _, err := st.Cancel(ctx, run, "someone"); err != nil {
					t.Fatal(err)
				}
				return run, 1, xExit(node, xInst, 0, 0, at)
			}},
		{name: "4 the step is done", phase: PhaseRunning,
			setup: func(t *testing.T, run, node string) (string, int, contract.Exited) {
				xRun(t, st, run, node, xTwoSteps())
				xClaim(t, st, node, xInst)
				xReport(t, st, run, 1, node, 0, StepResult{})
				return run, 1, xExit(node, xInst, 0, 0, at)
			}},
		{name: "5 an ask step", err: ErrExitRejected,
			reason: "step %s is an ask step; only a step that runs a command reports an exit",
			setup: func(t *testing.T, run, node string) (string, int, contract.Exited) {
				xRun(t, st, run, node, xTwoSteps())
				// 종류만 바꿔 분류를 본다 — ask 단계를 세우는 문법은 이 유닛의 것이 아니다.
				xExec(t, st, `UPDATE steps SET kind='ask' WHERE run_id=$1 AND seq=1`, run)
				return run, 1, xExit(node, xInst, 0, 0, at)
			}},
		{name: "5 an acquire step", err: ErrExitRejected,
			reason: "step %s is an acquire step; only a step that runs a command reports an exit",
			setup: func(t *testing.T, run, node string) (string, int, contract.Exited) {
				xRun(t, st, run, node, xTwoSteps())
				xExec(t, st, `UPDATE steps SET kind='acquire' WHERE run_id=$1 AND seq=1`, run)
				return run, 1, xExit(node, xInst, 0, 0, at)
			}},
		{name: "6 a merge step", phase: PhaseWaiting,
			setup: func(t *testing.T, run, node string) (string, int, contract.Exited) {
				xRun(t, st, run, node, xBake(t))
				xClaim(t, st, node, xInst)
				xReport(t, st, run, 1, node, 0, StepResult{})
				if c := xClaim(t, st, node, xInst); c.Kind != "merge" {
					t.Fatalf("claimed %s, want the merge step", c.Kind)
				}
				return run, 2, xExit(node, xInst, 0, 0, at)
			}},
		{name: "7 a late report of an earlier attempt", phase: PhaseRunning,
			setup: func(t *testing.T, run, node string) (string, int, contract.Exited) {
				xRun(t, st, run, node, xTwoSteps())
				xClaim(t, st, node, xInst)
				xExec(t, st, `UPDATE steps SET attempt=1 WHERE run_id=$1 AND seq=1`, run)
				return run, 1, xExit(node, xInst, 0, 0, at)
			}},
		{name: "8 an attempt that has not started", err: ErrExitRejected,
			reason: "step %s attempt 1 has not started; the current attempt is 0", phase: PhaseRunning,
			setup: func(t *testing.T, run, node string) (string, int, contract.Exited) {
				xRun(t, st, run, node, xTwoSteps())
				xClaim(t, st, node, xInst)
				return run, 1, xExit(node, xInst, 1, 0, at)
			}},
		{name: "9 a step that is not claimed", err: ErrExitRejected,
			reason: "step %s is PENDING, not claimed; nothing ran that could exit",
			setup: func(t *testing.T, run, node string) (string, int, contract.Exited) {
				xRun(t, st, run, node, xTwoSteps())
				return run, 1, xExit(node, xInst, 0, 0, at)
			}},
		{name: "10 another node", err: ErrExitRejected,
			reason: "step %s is claimed by another node", phase: PhaseRunning,
			setup: func(t *testing.T, run, node string) (string, int, contract.Exited) {
				xRun(t, st, run, node, xTwoSteps())
				xClaim(t, st, node, xInst)
				return run, 1, xExit("someone-else", xInst, 0, 0, at)
			}},
		{name: "11 a restarted node reports the exit of an earlier life", err: ErrExitRejected,
			reason: "step %s is claimed by another instance of this node; " +
				"a restarted node cannot report the exit of an earlier life", phase: PhaseRunning,
			setup: func(t *testing.T, run, node string) (string, int, contract.Exited) {
				xRun(t, st, run, node, xTwoSteps())
				xClaim(t, st, node, xInst)
				return run, 1, xExit(node, "life-2", 0, 0, at)
			}},
		{name: "12 the same report again", phase: PhaseFinalizing,
			setup: func(t *testing.T, run, node string) (string, int, contract.Exited) {
				xRun(t, st, run, node, xTwoSteps())
				xClaim(t, st, node, xInst)
				e := xExit(node, xInst, 0, 0, at)
				if ok, err := st.MarkExited(ctx, run, 1, e); !ok || err != nil {
					t.Fatalf("first report = %v, %v", ok, err)
				}
				return run, 1, e
			}},
		{name: "13 accepted", accepted: true, phase: PhaseFinalizing,
			setup: func(t *testing.T, run, node string) (string, int, contract.Exited) {
				xRun(t, st, run, node, xTwoSteps())
				xClaim(t, st, node, xInst)
				return run, 1, xExit(node, xInst, 0, 0, at)
			}},
		{name: "13 accepted on a row claimed before this code", accepted: true, phase: PhaseFinalizing,
			setup: func(t *testing.T, run, node string) (string, int, contract.Exited) {
				xRun(t, st, run, node, xTwoSteps())
				xClaim(t, st, node, xInst)
				xExec(t, st, `UPDATE steps SET phase=NULL, phase_since=NULL WHERE run_id=$1 AND seq=1`, run)
				return run, 1, xExit(node, xInst, 0, 0, at)
			}},
	}
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			run := "run-table-" + string(rune('a'+i))
			node := "node-table-" + string(rune('a'+i))
			runID, seq, e := tc.setup(t, run, node)
			before := xRow{}
			if tc.err != ErrNotFound && tc.err != ErrNoSuchStep {
				before = xRead(t, st, runID, seq)
			}
			ok, err := st.MarkExited(ctx, runID, seq, e)
			if ok != tc.accepted {
				t.Fatalf("accepted = %v, want %v (err %v)", ok, tc.accepted, err)
			}
			if !errors.Is(err, tc.err) || (tc.err == nil && err != nil) {
				t.Fatalf("err = %v, want %v", err, tc.err)
			}
			if tc.reason != "" {
				if want := strings.Replace(tc.reason, "%s", stepID(runID, seq), 1); err.Error() != want {
					t.Fatalf("reason = %q, want %q", err.Error(), want)
				}
			}
			if tc.err == ErrNotFound || tc.err == ErrNoSuchStep {
				return
			}
			after := xRead(t, st, runID, seq)
			if tc.phase != "" && after.Phase != tc.phase {
				t.Fatalf("phase = %q, want %q", after.Phase, tc.phase)
			}
			if tc.accepted {
				if after.Since == nil || !after.Since.Equal(at) {
					t.Fatalf("phase_since = %v, want the node's exited_at %v", after.Since, at)
				}
				var o contract.Outcome
				if err := json.Unmarshal(after.Exit, &o); err != nil || o.Kind != "exit" || *o.Code != 0 {
					t.Fatalf("exit = %s (%v)", after.Exit, err)
				}
				return
			}
			if before.Phase != after.Phase || !bytes.Equal(before.Exit, after.Exit) ||
				(before.Since == nil) != (after.Since == nil) ||
				(before.Since != nil && !before.Since.Equal(*after.Since)) {
				t.Fatalf("a report that was not accepted changed the row: %+v -> %+v", before, after)
			}
		})
	}
}

// 같은 키의 재전송 둘이 동시에 와도 하나만 받는다. 처음 값이 남고, 본문이 다른
// 재전송은 경고 한 줄을 남긴다.
func TestMarkExited_ARepeatKeepsTheFirstValue(t *testing.T) {
	st := obsStore(t)
	var logs bytes.Buffer
	st.Log = slog.New(slog.NewTextHandler(&logs, nil))
	ctx := context.Background()
	at := time.Date(2026, 9, 25, 9, 0, 0, 123456789, time.UTC)

	xRun(t, st, "run-repeat", "node-repeat", xTwoSteps())
	xClaim(t, st, "node-repeat", xInst)
	e := xExit("node-repeat", xInst, 0, 3, at)

	var wg sync.WaitGroup
	results := make([]bool, 8)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ok, err := st.MarkExited(ctx, "run-repeat", 1, e)
			if err != nil {
				t.Errorf("report %d: %v", i, err)
			}
			results[i] = ok
		}(i)
	}
	wg.Wait()
	n := 0
	for _, ok := range results {
		if ok {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("%d concurrent reports were accepted, want exactly 1", n)
	}
	if strings.Contains(logs.String(), "different outcome") {
		t.Fatalf("the same report warned: %s", logs.String())
	}

	other := xExit("node-repeat", xInst, 0, 7, at.Add(time.Second))
	if ok, err := st.MarkExited(ctx, "run-repeat", 1, other); ok || err != nil {
		t.Fatalf("a different repeat = %v, %v; want not accepted and no error", ok, err)
	}
	row := xRead(t, st, "run-repeat", 1)
	var o contract.Outcome
	if err := json.Unmarshal(row.Exit, &o); err != nil || *o.Code != 3 {
		t.Fatalf("exit = %s; the first value must stand", row.Exit)
	}
	if !row.Since.Equal(at.Truncate(time.Microsecond)) {
		t.Fatalf("phase_since = %v; the first value must stand", row.Since)
	}
	if !strings.Contains(logs.String(),
		"exit report repeated with a different outcome or time; the first one stands") {
		t.Fatalf("no warning for a different repeat: %s", logs.String())
	}
}

// phase 의 한 생애 — claim running · 종료 보고 finalizing · result 뒤에도 남는다.
// 진행 조회는 CLAIMED 일 때만 phase 를 싣고 exit 는 상태와 무관하게 싣는다.
func TestPhase_OneLife(t *testing.T) {
	st := obsStore(t)
	ctx := context.Background()
	xRun(t, st, "run-life", "node-life", xTwoSteps())

	xClaim(t, st, "node-life", xInst)
	row := xRead(t, st, "run-life", 1)
	var started time.Time
	if err := st.pool.QueryRow(ctx, `SELECT started_at FROM steps WHERE run_id='run-life' AND seq=1`).
		Scan(&started); err != nil {
		t.Fatal(err)
	}
	if row.Phase != PhaseRunning || row.Since == nil || !row.Since.Equal(started) || row.Exit != nil {
		t.Fatalf("after claim %+v, want running since started_at %v and no exit", row, started)
	}
	view := xView(t, st, "run-life", 1)
	if view.Phase != PhaseRunning || view.PhaseSince == nil || view.Exit != nil {
		t.Fatalf("view after claim %+v", view)
	}

	at := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	if ok, err := st.MarkExited(ctx, "run-life", 1, xExit("node-life", xInst, 0, 2, at)); !ok || err != nil {
		t.Fatalf("exited = %v, %v", ok, err)
	}
	view = xView(t, st, "run-life", 1)
	if view.Phase != PhaseFinalizing || !view.PhaseSince.Equal(at) || view.Exit == nil || *view.Exit.Code != 2 {
		t.Fatalf("view after exited %+v", view)
	}

	xReport(t, st, "run-life", 1, "node-life", 2, StepResult{})
	if row := xRead(t, st, "run-life", 1); row.Phase != PhaseFinalizing || row.Exit == nil {
		t.Fatalf("a result changed the phase columns: %+v", row)
	}
	view = xView(t, st, "run-life", 1)
	if view.State != StepDone || view.Phase != "" || view.PhaseSince != nil {
		t.Fatalf("a finished step carries a phase: %+v", view)
	}
	if view.Exit == nil || view.Exit.Kind != "exit" {
		t.Fatalf("a finished step lost its exit: %+v", view)
	}
}

func xView(t *testing.T, st *Store, runID string, seq int) StepView {
	t.Helper()
	steps, err := st.Steps(context.Background(), runID)
	if err != nil {
		t.Fatal(err)
	}
	return steps[seq-1]
}

// 재전달은 명령이 안 돈 단계다 — claim 과 같이 running 과 새 phase_since 를 적는다.
// merge 단계의 claim 은 waiting 이다.
func TestPhase_RedeliveryAndMerge(t *testing.T) {
	st := obsStore(t)
	ctx := context.Background()

	xRun(t, st, "run-redeliver", "node-redeliver", xTwoSteps())
	xClaim(t, st, "node-redeliver", xInst)
	past := time.Now().Add(-time.Hour)
	xExec(t, st, `UPDATE steps SET phase='finalizing', phase_since=$2, exit='{"kind":"exit","code":0}'
	               WHERE run_id=$1 AND seq=1`, "run-redeliver", past)
	again := xClaim(t, st, "node-redeliver", xInst)
	if again.Seq != 1 {
		t.Fatalf("redelivered seq %d", again.Seq)
	}
	row := xRead(t, st, "run-redeliver", 1)
	if row.Phase != PhaseRunning || row.Since == nil || !row.Since.After(past) || row.Exit != nil {
		t.Fatalf("after redelivery %+v", row)
	}

	xRun(t, st, "run-merge", "node-merge", xBake(t))
	build := xClaim(t, st, "node-merge", xInst)
	if build.Kind != "build" || xRead(t, st, "run-merge", 1).Phase != PhaseRunning {
		t.Fatalf("the build step is %s / %q", build.Kind, xRead(t, st, "run-merge", 1).Phase)
	}
	if ok, err := st.MarkExited(ctx, "run-merge", 1,
		xExit("node-merge", xInst, 0, 0, time.Now())); !ok || err != nil {
		t.Fatalf("a build step takes an exit report: %v, %v", ok, err)
	}
	xReport(t, st, "run-merge", 1, "node-merge", 0, StepResult{})
	xClaim(t, st, "node-merge", xInst)
	if row := xRead(t, st, "run-merge", 2); row.Phase != PhaseWaiting || row.Since == nil {
		t.Fatalf("the merge step is %+v, want waiting", row)
	}
}

// 단계를 PENDING 으로 되돌리는 세 자리가 칸 셋을 지운다 — 안 지우면 다음 회차가
// 옛 회차의 finalizing 을 물려받는다.
func TestPhase_RewindsClearTheColumns(t *testing.T) {
	st := obsStore(t)
	ctx := context.Background()
	zero := 0
	at := time.Now()

	// loop — 뒤 단계가 조건을 못 채워 구간이 되돌아간다 (rollback.go loopBack).
	loop := xTwoSteps()
	loop.Steps[1].Loop = &contract.Loop{BackTo: "s1", Max: 3, Until: contract.Condition{ExitCode: &zero}}
	xRun(t, st, "run-loop", "node-loop", loop)
	for seq, code := range []int{0, 1} {
		xClaim(t, st, "node-loop", xInst)
		if ok, err := st.MarkExited(ctx, "run-loop", seq+1, xExit("node-loop", xInst, 0, code, at)); !ok || err != nil {
			t.Fatalf("exited s%d: %v, %v", seq+1, ok, err)
		}
		xReport(t, st, "run-loop", seq+1, "node-loop", code, StepResult{})
	}
	xWantCleared(t, st, "run-loop", 1, 2)

	// validate_with — 검증자가 실패해 대상과 검증자가 되돌아간다 (rollback.go validateBack).
	val := xTwoSteps()
	val.Steps[0].ValidateWith, val.Steps[0].MaxAttempts = "s2", 2
	xRun(t, st, "run-validate", "node-validate", val)
	for seq, code := range []int{0, 1} {
		xClaim(t, st, "node-validate", xInst)
		if ok, err := st.MarkExited(ctx, "run-validate", seq+1,
			xExit("node-validate", xInst, 0, code, at)); !ok || err != nil {
			t.Fatalf("exited s%d: %v, %v", seq+1, ok, err)
		}
		xReport(t, st, "run-validate", seq+1, "node-validate", code, StepResult{})
	}
	xWantCleared(t, st, "run-validate", 1, 2)

	// 계획 거절의 되감기 (ask.go rewindToPlanner) — 문장만 본다. 계획과 승인을 세우는
	// 문법은 이 유닛의 것이 아니다.
	plan := contract.Contract{
		Requires: []contract.Require{{As: "b", Capability: contract.CapabilityAgentReason}},
		Steps: []contract.Step{
			{ID: "plan", Uses: "b", Run: []string{"true"}},
			{ID: "work", Uses: "b", Run: []string{"true"}},
			{ID: "approve", Uses: "b", Run: []string{"true"}},
		},
	}
	xRun(t, st, "run-rewind", "node-rewind", plan)
	xExec(t, st, `UPDATE steps SET state='DONE', phase='finalizing', phase_since=now(),
	               exit='{"kind":"exit","code":0}' WHERE run_id='run-rewind'`)
	tx, err := st.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ask := contract.Step{ID: "approve", Ask: &contract.Ask{Adopts: "plan"}}
	if back, err := st.rewindToPlanner(ctx, tx, "run-rewind", 3, ask, plan); !back || err != nil {
		t.Fatalf("rewind = %v, %v", back, err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	xWantCleared(t, st, "run-rewind", 1, 2, 3)
}

func xWantCleared(t *testing.T, st *Store, runID string, seqs ...int) {
	t.Helper()
	for _, seq := range seqs {
		var state string
		if err := st.pool.QueryRow(context.Background(),
			`SELECT state FROM steps WHERE run_id=$1 AND seq=$2`, runID, seq).Scan(&state); err != nil {
			t.Fatal(err)
		}
		if state != StepPending {
			t.Fatalf("%s#%d is %s; the rewind did not happen", runID, seq, state)
		}
		if row := xRead(t, st, runID, seq); row.Phase != "" || row.Since != nil || row.Exit != nil {
			t.Fatalf("%s#%d kept %+v after the rewind", runID, seq, row)
		}
	}
}

// Record 의 새 칸 넷 — exited_at 은 result 의 값이 먼저이고, 없으면 종료 보고로 받은
// phase_since 다. 옛 노드(종료 보고 없음)는 last_phase running 만 남긴다.
func TestStepFiles_TheExitColumns(t *testing.T) {
	st := obsStore(t)
	ctx := context.Background()
	exited := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	fromResult := exited.Add(-time.Second)
	finalized := exited.Add(30 * time.Second)

	xRun(t, st, "run-record", "node-record", xTwoSteps())
	// s1 — 종료 보고와 result 둘 다 시각을 나른다. result 의 값이 이긴다.
	xClaim(t, st, "node-record", xInst)
	if ok, err := st.MarkExited(ctx, "run-record", 1, xExit("node-record", xInst, 0, 0, exited)); !ok || err != nil {
		t.Fatalf("exited = %v, %v", ok, err)
	}
	xReport(t, st, "run-record", 1, "node-record", 0,
		StepResult{ExitedAt: &fromResult, FinalizedAt: &finalized})
	// s2 — 종료 보고만 있다. result 에 시각이 없다.
	xClaim(t, st, "node-record", xInst)
	if ok, err := st.MarkExited(ctx, "run-record", 2, xExit("node-record", xInst, 0, 1, exited)); !ok || err != nil {
		t.Fatalf("exited = %v, %v", ok, err)
	}
	xReport(t, st, "run-record", 2, "node-record", 1, StepResult{})

	xRun(t, st, "run-old-node", "node-old", xTwoSteps())
	xClaim(t, st, "node-old", xInst)
	xReport(t, st, "run-old-node", 1, "node-old", 0, StepResult{})

	files, err := st.StepFiles(ctx, "run-record")
	if err != nil {
		t.Fatal(err)
	}
	s1, s2 := files[0], files[1]
	if s1.ExitedAt != fromResult.Format(time.RFC3339Nano) || s1.FinalizedAt != finalized.Format(time.RFC3339Nano) {
		t.Fatalf("s1 exited_at %q finalized_at %q", s1.ExitedAt, s1.FinalizedAt)
	}
	if s1.LastPhase != PhaseFinalizing || !strings.Contains(string(s1.Exit), `"kind": "exit"`) &&
		!strings.Contains(string(s1.Exit), `"kind":"exit"`) {
		t.Fatalf("s1 last_phase %q exit %s", s1.LastPhase, s1.Exit)
	}
	if s2.ExitedAt != exited.Format(time.RFC3339Nano) || s2.FinalizedAt != "" {
		t.Fatalf("s2 exited_at %q finalized_at %q; want the exit report's time", s2.ExitedAt, s2.FinalizedAt)
	}

	old, err := st.StepFiles(ctx, "run-old-node")
	if err != nil {
		t.Fatal(err)
	}
	if o := old[0]; o.LastPhase != PhaseRunning || o.Exit != nil || o.ExitedAt != "" || o.FinalizedAt != "" {
		t.Fatalf("an old node's step %+v", o)
	}
	if p := old[1]; p.LastPhase != "" || p.Exit != nil {
		t.Fatalf("a step that never ran carries %+v", p)
	}
}

// result 의 새 칸 아홉은 값으로 거절하지 않고 봉인에 그대로 남는다 — 어휘 밖 값도.
func TestStepFiles_TheNewResultFieldsSurviveTheSeal(t *testing.T) {
	st := obsStore(t)
	ctx := context.Background()
	at := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	ir := "your-ir-tag"
	res := StepResult{
		ExitedAt: &at, FinalizedAt: &at,
		Finalize: contract.StageOK, Upload: "slow", Reason: "something_new",
		Diagnostics: &contract.Diagnostics{Changes: contract.ChangesNotMeasured, Effect: contract.EffectBuild,
			Missing: []string{"report"}, Collect: []contract.CollectNote{{Name: "log", Why: "not found"}}},
		CheckpointCapture: &contract.CheckpointCapture{State: contract.CaptureNotRequested},
		Build: &contract.BuildManifest{Head: "abc", IR: &ir,
			Builds: []contract.BuildRecord{{Name: "config-a", Command: "<cmd>", ExitCode: 0}}},
		Merge: &contract.MergeResult{IR: &ir, MergedAt: at, Ops: contract.MergeOps{Created: 3}},
	}
	xRun(t, st, "run-seal", "node-seal", xTwoSteps())
	xClaim(t, st, "node-seal", xInst)
	xReport(t, st, "run-seal", 1, "node-seal", 0, res)

	files, err := st.StepFiles(ctx, "run-seal")
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(files[0].Result)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"finalize":"ok"`, `"upload":"slow"`, `"reason":"something_new"`,
		`"changes":"not_measured"`, `"checkpoint_capture":{"state":"not_requested"}`, `"head":"abc"`,
		`"ir":"your-ir-tag"`, `"created":3`, `"exited_at":"2026-09-25T09:00:00Z"`,
		`"finalized_at":"2026-09-25T09:00:00Z"`, `"why":"not found"`} {
		if !strings.Contains(string(got), want) {
			t.Fatalf("the sealed result %s lost %s", got, want)
		}
	}
}

// 어휘 밖 값은 칸 다섯에서만 찾는다. 빈 값은 안 적은 것이다.
func TestStepResult_OutOfVocabulary(t *testing.T) {
	if got := (StepResult{}).OutOfVocabulary(); len(got) != 0 {
		t.Fatalf("an empty result = %v", got)
	}
	known := StepResult{Finalize: contract.StageOK, Upload: contract.StageTimeout,
		Reason:            contract.ReasonMergeWaitTimeout,
		CheckpointCapture: &contract.CheckpointCapture{State: contract.CaptureCaptured, Reason: "anything"},
		Diagnostics:       &contract.Diagnostics{Changes: contract.ChangesPartial, DiscoveryLimit: "anything"}}
	if got := known.OutOfVocabulary(); len(got) != 0 {
		t.Fatalf("known values = %v", got)
	}
	unknown := StepResult{Finalize: "a", Upload: "b", Reason: "c",
		CheckpointCapture: &contract.CheckpointCapture{State: "d"},
		Diagnostics:       &contract.Diagnostics{Changes: "e"}}
	got := unknown.OutOfVocabulary()
	want := [][2]string{{"finalize", "a"}, {"upload", "b"}, {"reason", "c"},
		{"checkpoint_capture.state", "d"}, {"diagnostics.changes", "e"}}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

// Claimed 는 계약의 새 칸 일곱을 JSON 이름 그대로 싣는다. 안 적은 계약에는 새 이름이
// 안 나온다. 재전달도 같은 칸을 싣는다.
func TestClaimed_CarriesTheContractFields(t *testing.T) {
	st := obsStore(t)

	xRun(t, st, "run-claim-bake", "node-claim-bake", xBake(t))
	build := xClaim(t, st, "node-claim-bake", xInst)
	if build.Effect != contract.EffectPrepare || build.IR != "your-ir-tag" || build.Sync == "" ||
		len(build.Builds) != 2 || build.Builds[1].Name != "config-b" {
		t.Fatalf("the build step claimed %+v", build)
	}
	again := xClaim(t, st, "node-claim-bake", xInst)
	if again.IR != build.IR || len(again.Builds) != 2 {
		t.Fatalf("a redelivery lost the fields: %+v", again)
	}
	xReport(t, st, "run-claim-bake", 1, "node-claim-bake", 0, StepResult{})
	merge := xClaim(t, st, "node-claim-bake", xInst)
	if merge.Merge == nil || merge.Merge.Wait != "4h" {
		t.Fatalf("the merge step claimed %+v", merge.Merge)
	}
	b, _ := json.Marshal(build)
	for _, key := range []string{`"effect":"prepare"`, `"sync":`, `"builds":[{"name":"config-a"`, `"ir":"your-ir-tag"`} {
		if !strings.Contains(string(b), key) {
			t.Fatalf("the claim %s lacks %s", b, key)
		}
	}
	if m, _ := json.Marshal(merge); !strings.Contains(string(m), `"merge":{"wait":"4h"}`) {
		t.Fatalf("the merge claim %s", m)
	}

	edit := xTwoSteps()
	edit.Steps[0].Effect = contract.EffectEdit
	edit.Steps[0].Budget = &contract.Budget{Finalize: "2m"}
	edit.Steps[0].Discover = true
	xRun(t, st, "run-claim-edit", "node-claim-edit", edit)
	c := xClaim(t, st, "node-claim-edit", xInst)
	cb, _ := json.Marshal(c)
	for _, key := range []string{`"effect":"edit"`, `"budget":{"finalize":"2m"}`, `"discover":true`} {
		if !strings.Contains(string(cb), key) {
			t.Fatalf("the claim %s lacks %s", cb, key)
		}
	}

	xRun(t, st, "run-claim-plain", "node-claim-plain", xTwoSteps())
	plain, _ := json.Marshal(xClaim(t, st, "node-claim-plain", xInst))
	for _, key := range []string{`"effect"`, `"budget"`, `"discover"`, `"sync"`, `"builds"`, `"ir"`, `"merge"`} {
		if strings.Contains(string(plain), key) {
			t.Fatalf("a contract that wrote nothing new carries %s: %s", key, plain)
		}
	}
}
