//go:build !windows

package enode

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/taeels/enode/internal/contract"
)

// 명령이 끝난 뒤의 구간을 Worker 흐름으로 본다 — 조각 1 (걷지 않는다) · 조각 3 (예산)의
// 기계 부분과 경로마다의 result 칸 (business-rules.md 9절).
//
// 진짜 프로세스를 띄우므로 worker_unix_test.go 와 같은 태그다.

// finalizeWorker 는 종료 보고를 보내는 Worker 다 — instance 가 있고 재전송을 기다리지 않는다.
func finalizeWorker(t *testing.T) (*mediator, *Worker) {
	t.Helper()
	m := newMediator(t)
	w := newWorker(m)
	w.Client.Instance = "inst-1"
	w.exitWait = func(int) time.Duration { return time.Millisecond }
	holdLease(w)
	return m, w
}

// oldTree 는 파일 n 개가 있는 워크스페이스다. mtime 을 한 시간 전으로 옮겨 이 단계가
// 쓴 것과 나눈다.
func oldTree(t *testing.T, n int) string {
	t.Helper()
	ws := t.TempDir()
	old := time.Now().Add(-time.Hour)
	for i := 0; i < n; i++ {
		p := filepath.Join(ws, "f"+strconv.Itoa(i))
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, old, old); err != nil {
			t.Fatal(err)
		}
	}
	return ws
}

// countFS 는 결과 확정이 파일시스템에 닿는 두 자리를 센다 (changed.go 의 statPath · walkDir).
func countFS(t *testing.T) (stats, walks *atomic.Int32) {
	t.Helper()
	stats, walks = &atomic.Int32{}, &atomic.Int32{}
	origStat, origWalk := statPath, walkDir
	statPath = func(p string) (os.FileInfo, error) { stats.Add(1); return origStat(p) }
	walkDir = func(root string, fn fs.WalkDirFunc) error { walks.Add(1); return origWalk(root, fn) }
	t.Cleanup(func() { statPath, walkDir = origStat, origWalk })
	return stats, walks
}

// ── 조각 1 — 걷지 않는다 ──────────────────────────────────────────

// build effect 단계는 워크스페이스를 걷지 않는다. 지목 경로 N 개만 stat 하고,
// $OUT 의 이름은 그대로 올라간다. 기록에 「바뀐 파일이 없다」가 없다.
func TestFinalize_ABuildStepDoesNotWalkTheWorkspace(t *testing.T) {
	m, w := finalizeWorker(t)
	w.Local.Workspace = oldTree(t, 500)
	stats, walks := countFS(t)

	step := runStep("sh", "-c", `touch f1 f2 f3 && printf ok > "$OUT/result"`)
	step.CheckChanged = []string{"f1", "f2", "f3"}
	step.Out = []string{"result"}
	w.execute(context.Background(), step)

	res := m.only(t)
	if res.Error != "" || res.ExitCode == nil || *res.ExitCode != 0 {
		t.Fatalf("the step did not complete: %+v", res)
	}
	if n := walks.Load(); n != 0 {
		t.Fatalf("slice 1 violated: a build step walked the workspace %d times", n)
	}
	if n := stats.Load(); n != 3 {
		t.Fatalf("slice 1 violated: %d stats for 3 named paths", n)
	}
	if strings.Join(res.Changed, ",") != "f1,f2,f3" {
		t.Fatalf("the named paths were not checked: %v", res.Changed)
	}
	if body, ok := m.blob("result"); !ok || string(body) != "ok" || !listed(res.Produced, "result") {
		t.Fatalf("the named output was not sealed: %q %v", body, res.Produced)
	}
	if _, ok := m.blob("workspace.changed"); ok {
		t.Fatal("FR-1 violated: workspace.changed is still produced")
	}
	tail := m.logOf("build")
	if strings.Contains(tail, "no files changed") || !strings.Contains(tail, "enode: "+noticeRunDefault) {
		t.Fatalf("the step log is wrong:\n%s", tail)
	}
	d := res.Diagnostics
	if d == nil || d.Changes != contract.ChangesNotMeasured || d.Effect != contract.EffectBuild || d.Discovered != nil {
		t.Fatalf("diagnostics = %+v", d)
	}
	if res.ExitedAt == nil || res.FinalizedAt == nil || res.FinalizedAt.Before(*res.ExitedAt) {
		t.Fatalf("the two times are missing or out of order: %v %v", res.ExitedAt, res.FinalizedAt)
	}
	if res.Finalize != contract.StageOK || res.Upload != contract.StageOK || res.Reason != "" {
		t.Fatalf("stages = %q %q %q", res.Finalize, res.Upload, res.Reason)
	}
	exits := m.exits()
	if len(exits) != 1 || exits[0].Instance != "inst-1" || exits[0].Node != "n1" ||
		exits[0].Outcome.Kind != contract.OutcomeExit || *exits[0].Outcome.Code != 0 ||
		!exits[0].ExitedAt.Equal(*res.ExitedAt) {
		t.Fatalf("exit report = %+v, result exited_at %v", exits, res.ExitedAt)
	}
}

// discover 를 켠 단계는 훑는다 — 이 단계가 쓴 것만 목록에 있고 changes 는 measured.
func TestFinalize_DiscoverListsWhatTheStepWrote(t *testing.T) {
	m, w := finalizeWorker(t)
	w.Local.Workspace = oldTree(t, 50)
	_, walks := countFS(t)

	step := runStep("sh", "-c", `printf new > f7 && printf made > built.bin`)
	step.Discover = true
	w.execute(context.Background(), step)

	res := m.only(t)
	d := res.Diagnostics
	if walks.Load() != 1 || d == nil || d.Changes != contract.ChangesMeasured ||
		strings.Join(d.Discovered, ",") != "built.bin,f7" {
		t.Fatalf("walks %d diagnostics %+v", walks.Load(), d)
	}
	tail := m.logOf("build")
	if !strings.Contains(tail, "enode: discover listed 2 files created or modified by this step\n") ||
		strings.Contains(tail, noticeRunDefault) {
		t.Fatalf("the step log is wrong:\n%s", tail)
	}
}

// ── 조각 3 — 예산 ─────────────────────────────────────────────────

// stuckRuntime 은 Finalize 가 마감까지 안 끝나는 세션을 연다. 나머지는 native 다.
type stuckRuntime struct{ spec *FinalizeSpec }

func (r stuckRuntime) Open(ctx context.Context, spec RuntimeSpec) (StepSession, error) {
	s, err := NativeRuntime{}.Open(ctx, spec)
	return &stuckSession{StepSession: s, seen: r.spec}, err
}

type stuckSession struct {
	StepSession
	seen *FinalizeSpec
}

func (s *stuckSession) Finalize(ctx context.Context, spec FinalizeSpec) (FinalizeResult, error) {
	if s.seen != nil {
		*s.seen = spec
		return s.StepSession.Finalize(ctx, spec)
	}
	<-ctx.Done()
	return FinalizeResult{}, ctx.Err()
}

// Finalize 예산을 넘기면 finalize_timeout. 업로드와 보고는 계속하고, 명령의 exit code 는
// 그대로 남는다 — 명령 실패와 원인이 나뉜다.
func TestFinalize_OverTheFinalizeBudget(t *testing.T) {
	m, w := finalizeWorker(t)
	w.Runtime = stuckRuntime{}
	w.budgets = func(*Step) (time.Duration, time.Duration) { return 50 * time.Millisecond, time.Minute }

	step := runStep("sh", "-c", `printf ok > "$OUT/result"; exit 1`)
	w.execute(context.Background(), step)

	res := m.only(t)
	if res.Finalize != contract.StageTimeout || res.Reason != contract.ReasonFinalizeTimeout ||
		res.Error != "finalize budget of 50ms exceeded" {
		t.Fatalf("finalize %q reason %q error %q", res.Finalize, res.Reason, res.Error)
	}
	if res.ExitCode == nil || *res.ExitCode != 1 {
		t.Fatalf("the command's exit code was lost: %v", res.ExitCode)
	}
	if res.Upload != contract.StageOK || !listed(res.Produced, "result") {
		t.Fatalf("the upload did not go on: %q %v", res.Upload, res.Produced)
	}
	if d := res.Diagnostics; d == nil || d.Changes != contract.ChangesNotMeasured {
		t.Fatalf("diagnostics = %+v", d)
	}
	if tail := m.logOf("build"); !strings.Contains(tail, "enode: finalize budget of 50ms exceeded\n") {
		t.Fatalf("the step log does not say it:\n%s", tail)
	}
}

// 업로드 예산을 넘기면 upload_timeout. 남은 이름은 produced 에 없다.
func TestFinalize_OverTheUploadBudget(t *testing.T) {
	m, w := finalizeWorker(t)
	w.budgets = func(*Step) (time.Duration, time.Duration) { return time.Minute, 300 * time.Millisecond }
	m.putHold["b"] = true

	w.execute(context.Background(), runStep("sh", "-c", `for n in a b c; do printf $n > "$OUT/$n"; done`))

	res := m.only(t)
	if res.Upload != contract.StageTimeout || res.Reason != contract.ReasonUploadTimeout ||
		res.Error != "upload budget of 300ms exceeded" {
		t.Fatalf("upload %q reason %q error %q", res.Upload, res.Reason, res.Error)
	}
	if strings.Join(res.Produced, ",") != "a" {
		t.Fatalf("produced %v, want only the name before the deadline", res.Produced)
	}
	if res.ExitCode == nil || *res.ExitCode != 0 || res.Finalize != contract.StageOK {
		t.Fatalf("exit %v finalize %q", res.ExitCode, res.Finalize)
	}
}

// 계약이 늘린 예산이 그 값으로 잡힌다 — 마감은 exited_at + finalize, 훑기 시간은 그 절반.
func TestFinalize_TheContractBudgetSetsTheDeadline(t *testing.T) {
	m, w := finalizeWorker(t)
	var seen FinalizeSpec
	w.Runtime = stuckRuntime{spec: &seen}
	step := runStep("true")
	step.Budget = &contract.Budget{Finalize: "5m", Upload: "10m"}
	w.execute(context.Background(), step)

	res := m.only(t)
	if got := seen.Deadline.Sub(*res.ExitedAt); got != 5*time.Minute {
		t.Fatalf("deadline is exited_at + %v, want 5m", got)
	}
	if seen.DiscoverFor != 150*time.Second {
		t.Fatalf("discover time = %v, want half of 5m", seen.DiscoverFor)
	}
}

// ── 종료 보고 ─────────────────────────────────────────────────────

// 옛 Mediator 는 exited 에 404 를 준다. 그 단계는 다시 안 보내고 정상으로 끝난다.
func TestFinalize_AnOldMediatorGetsOneReport(t *testing.T) {
	m, w := finalizeWorker(t)
	m.exitedCodes = []int{404}
	w.execute(context.Background(), runStep("true"))

	if res := m.only(t); res.Error != "" || res.ExitCode == nil {
		t.Fatalf("an old mediator broke the step: %+v", res)
	}
	if n := len(m.exits()); n != 1 {
		t.Fatalf("sent %d exit reports to an old mediator, want 1", n)
	}
}

// 5xx 가 이어지는 동안 result 가 나가면 재전송이 멈춘다.
func TestFinalize_TheResultEndsTheRetries(t *testing.T) {
	m, w := finalizeWorker(t)
	m.exitedCodes = make([]int, 1_000_000)
	for i := range m.exitedCodes {
		m.exitedCodes[i] = 503
	}
	w.execute(context.Background(), runStep("true"))
	m.only(t)
	before := len(m.exits())
	time.Sleep(50 * time.Millisecond)
	if after := len(m.exits()); after != before {
		t.Fatalf("exit reports went on after the result: %d -> %d", before, after)
	}
}

// ── 경로마다의 새 칸 (business-rules.md 9절) ──────────────────────────

// 프로세스가 뜨지 않았으면 종료 보고도 Finalize 도 없다. 단계 로그는 올린다.
func TestFinalize_AProcessThatNeverStartedHasNoNewFields(t *testing.T) {
	m, w := finalizeWorker(t)
	w.execute(context.Background(), runStep(filepath.Join(t.TempDir(), "never-launched")))

	res := m.only(t)
	if res.Error == "" || res.ExitCode != nil {
		t.Fatalf("a command that never ran was reported as run: %+v", res)
	}
	if res.ExitedAt != nil || res.FinalizedAt != nil || res.Finalize != "" || res.Upload != "" || res.Diagnostics != nil {
		t.Fatalf("new fields on a step that never reached finalize: %+v", res)
	}
	if n := len(m.exits()); n != 0 {
		t.Fatalf("%d exit reports for a process that never started", n)
	}
}

// native 에서 signal 로 죽은 명령 — 종료 보고는 signal 로 보내고, result 는 오늘처럼
// 완주가 아니다. Finalize 는 돈다.
func TestFinalize_ACommandKilledByASignal(t *testing.T) {
	m, w := finalizeWorker(t)
	w.execute(context.Background(), runStep("sh", "-c", `kill -KILL $$`))

	res := m.only(t)
	if res.Error != "signal: killed" || res.ExitCode != nil {
		t.Fatalf("error %q exit %v", res.Error, res.ExitCode)
	}
	if res.ExitedAt == nil || res.FinalizedAt == nil || res.Finalize != contract.StageOK {
		t.Fatalf("finalize did not run: %+v", res)
	}
	exits := m.exits()
	if len(exits) != 1 || exits[0].Outcome.Kind != contract.OutcomeSignal || *exits[0].Outcome.Code != 9 {
		t.Fatalf("exit report = %+v", exits)
	}
}

// 임대가 끝나 죽인 명령에는 종료 보고도 새 칸도 없다.
func TestFinalize_ALeaseEndSkipsFinalize(t *testing.T) {
	m, w := finalizeWorker(t)
	marker := filepath.Join(t.TempDir(), "started")
	sh := script(t, "long", `printf 'started\n' > "$1"
exec sleep 30
`)
	go func() {
		waitForFile(t, marker)
		w.Held.Set(nil)
	}()
	w.execute(context.Background(), runStep(sh, marker))

	res := m.only(t)
	if res.Error != "aborted: lease expired" || res.ExitedAt != nil || res.Finalize != "" || res.Diagnostics != nil {
		t.Fatalf("result = %+v", res)
	}
	if n := len(m.exits()); n != 0 {
		t.Fatalf("%d exit reports for a command the lease killed", n)
	}
}

// ── agent 단계 ────────────────────────────────────────────────────

// 완주하지 못한 하네스에도 discover 를 켰으면 훑는다 — 실패한 sandbox 를 조사하는
// 진단이다. collect 와 산출물은 없다. 단계 로그는 올라간다.
func TestFinalize_AnAgentThatDidNotCompleteStillDiscovers(t *testing.T) {
	m, w := finalizeWorker(t)
	w.Local.Workspace = oldTree(t, 10)
	w.Local.HarnessBin = stubHarness(t, `printf half > half.c
printf 'half written\n' > "$OUT/plan.json"
printf '{"type":"result","subtype":"error_during_execution","is_error":true}\n'
`)
	step := agentStep(`{}`)
	step.Discover = true
	step.Out = []string{"plan.json"}
	step.Collect = map[string]string{"extra": "f1"}
	w.execute(context.Background(), step)

	res := m.only(t)
	if !strings.HasPrefix(res.Error, "harness: ") || len(res.Produced) != 0 {
		t.Fatalf("error %q produced %v", res.Error, res.Produced)
	}
	d := res.Diagnostics
	if d == nil || d.Changes != contract.ChangesMeasured || !listed(d.Discovered, "half.c") || len(d.Collect) != 0 {
		t.Fatalf("diagnostics = %+v", d)
	}
	if res.ExitedAt == nil || len(m.exits()) != 1 {
		t.Fatalf("the harness exit was not reported: %v %v", res.ExitedAt, m.exits())
	}
	if tail := m.logOf("plan"); !strings.Contains(tail, "enode: discover listed") {
		t.Fatalf("the step log did not arrive with its tail:\n%s", tail)
	}
}

// failingRuntime 은 Finalize 가 오류를 내는 세션을 연다.
type failingRuntime struct{}

func (failingRuntime) Open(ctx context.Context, spec RuntimeSpec) (StepSession, error) {
	s, err := NativeRuntime{}.Open(ctx, spec)
	return &failingSession{s}, err
}

type failingSession struct{ StepSession }

func (failingSession) Finalize(context.Context, FinalizeSpec) (FinalizeResult, error) {
	return FinalizeResult{}, errors.New("merged view is gone")
}

// agent 에서 Finalize 오류는 finalize: error 로 적고 업로드와 보고를 계속한다 — 명령 단계와
// 같은 모양이다. 오늘은 곧바로 보고해 단계 로그와 산출물이 빠졌다. 안내 줄이 끝에 있다.
func TestFinalize_AnAgentFinalizeErrorStillUploads(t *testing.T) {
	m, w := finalizeWorker(t)
	w.Runtime = failingRuntime{}
	w.Local.HarnessBin = stubHarness(t, `printf 'the plan\n' > "$OUT/plan.json"
printf '{"type":"result","subtype":"success","num_turns":1}\n'
`)
	step := agentStep(`{}`)
	step.Out = []string{"plan.json"}
	w.execute(context.Background(), step)

	res := m.only(t)
	if res.Finalize != contract.StageError || res.Error != "runtime finalize: merged view is gone" {
		t.Fatalf("finalize %q error %q", res.Finalize, res.Error)
	}
	if !listed(res.Produced, "plan.json") {
		t.Fatalf("the artifact was dropped: %v", res.Produced)
	}
	if tail := m.logOf("plan"); !strings.Contains(tail, "enode: "+noticeAgentChanged) {
		t.Fatalf("the agent notice is missing:\n%s", tail)
	}
}

// ── 측정 — helper 여유 5초 (계획 3절 ②) ─────────────────────────────

// 마감이 지난 뒤 Finalize 가 돌아오기까지다. helper 는 같은 Finalize 를 돌므로 여기서
// 측정한다 — 큰 트리를 훑는 중과 큰 파일을 collect 하는 중. 1초 안쪽이면 5초가 넉넉하다.
func TestFinalize_ReturnsSoonAfterItsDeadline(t *testing.T) {
	ws := t.TempDir()
	for i := 0; i < 200; i++ {
		sub := filepath.Join(ws, "d"+strconv.Itoa(i))
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatal(err)
		}
		for j := 0; j < 200; j++ {
			if err := os.WriteFile(filepath.Join(sub, strconv.Itoa(j)), nil, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	big := filepath.Join(ws, "big.bin")
	if err := os.WriteFile(big, make([]byte, 256<<20), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name string
		spec FinalizeSpec
	}{
		{"walking", FinalizeSpec{Discover: true}},
		{"collecting", FinalizeSpec{Collect: map[string]string{"big": "big.bin"}}},
	} {
		c.spec.Workspace, c.spec.Out = ws, t.TempDir()
		c.spec.Stamp = Stamp{At: time.Now().Add(time.Hour), Root: ws}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
		deadline, _ := ctx.Deadline()
		_, err := (&nativeSession{}).Finalize(ctx, c.spec)
		late := time.Since(deadline)
		cancel()
		t.Logf("%s: returned %v after the deadline (err %v)", c.name, late, err)
		if late > time.Second {
			t.Errorf("%s: returned %v after the deadline", c.name, late)
		}
	}
}

// Result 의 새 칸은 store.StepResult 와 같은 이름으로 나간다.
func TestResult_TheNewFieldsUseTheMediatorNames(t *testing.T) {
	now := time.Now().UTC()
	b, _ := json.Marshal(Result{ExitedAt: &now, FinalizedAt: &now, Finalize: contract.StageOK,
		Upload: contract.StageTimeout, Reason: contract.ReasonUploadTimeout,
		Diagnostics: &contract.Diagnostics{Changes: contract.ChangesNotMeasured}})
	for _, k := range []string{`"exited_at"`, `"finalized_at"`, `"finalize":"ok"`, `"upload":"timeout"`,
		`"reason":"upload_timeout"`, `"diagnostics":{`} {
		if !strings.Contains(string(b), k) {
			t.Errorf("%s missing in %s", k, b)
		}
	}
	b, _ = json.Marshal(Result{Node: "n"})
	for _, k := range []string{"exited_at", "finalized_at", "finalize", "upload", "reason", "diagnostics"} {
		if strings.Contains(string(b), k) {
			t.Errorf("an old-shaped result carries %s: %s", k, b)
		}
	}
}

// collect 가 못 걷어도 단계를 안 죽인다 — 판정은 success_when 이 한다
// (ADR-004 · I3). 대신 왜 못 걷었는지를 진단 칸과 단계 로그 끝이 적는다.
// workspace.changed 는 더 안 낸다 (FR-1 · US-10).
func TestExecute_ACollectFailureIsWrittenDownNotThrown(t *testing.T) {
	m := newMediator(t)
	w := newWorker(m)
	holdLease(w)
	w.Local.Workspace = t.TempDir()

	step := runStep("true")
	step.Collect = map[string]string{"artifact": "arch/arm/boot/zImage"}
	step.Out = []string{"artifact"}
	w.execute(context.Background(), step)

	res := m.only(t)
	if res.Error != "" || res.ExitCode == nil {
		t.Fatalf("a collect failure killed the step: %+v", res)
	}
	if _, ok := m.blob("workspace.changed"); ok {
		t.Fatal("FR-1 violated: workspace.changed is still produced")
	}
	d := res.Diagnostics
	if d == nil || len(d.Collect) != 1 || !strings.Contains(d.Collect[0].Why, "no file matches") ||
		len(d.Missing) != 1 || d.Missing[0] != "artifact" {
		t.Fatalf("the diagnostics do not say why collect failed: %+v", d)
	}
	tail := m.logOf("build")
	if !strings.Contains(tail, "enode: collect could not gather artifact: no file matches") ||
		!strings.Contains(tail, "enode: required by the contract but missing from $OUT: artifact") {
		t.Fatalf("the step log does not say what the contract wanted:\n%s", tail)
	}
	if strings.Contains(tail, "no files changed") {
		t.Fatalf("FR-1 violated: an unmeasured list was reported as no change:\n%s", tail)
	}
}
