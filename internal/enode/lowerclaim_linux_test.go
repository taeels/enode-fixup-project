package enode

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/taeels/enode/internal/contract"
	execenv "github.com/taeels/enode/internal/environment"
	"github.com/taeels/enode/internal/lower"
)

// claim 과 lower 공유 잠금 (FD 흐름 3절 · 계획 4절 ⑨ ⑩) — 시험 Mediator 와 임시 폴더의 진짜 잠금으로 돈다.

// releasedGuard 는 공유를 쥐었다가 울타리 두 번으로 놓은 LowerGuard 다.
func releasedGuard(t *testing.T) guardFixture {
	t.Helper()
	f := newGuardFixture(t)
	f.g.BeforeAdvert(nil)
	f.g.AfterResponse("graceful", nil)
	f.g.AfterResponse("graceful", nil)
	if f.g.shared != nil {
		t.Fatal("not released")
	}
	return f
}

// 거절 보고의 모양 — reason lower_changed 와 그 문구. 준비도 세션도 명령도 없고 exited 도 없다.
func TestLowerClaim_RefusedRunIsReportedWithoutRunning(t *testing.T) {
	f := releasedGuard(t)
	f.merged(t, "R-9") // 놓은 뒤에 합쳐졌다
	m, w := finalizeWorker(t)
	rt := &trackingRuntime{}
	w.Runtime, w.Guard = rt, f.g
	w.RuntimeRecord = &execenv.Record{Runtime: "runc-overlay"}
	step := runStep("sh", "-c", "exit 0")
	step.Workspace = []byte(`{"revision":"HEAD"}`)
	w.execute(context.Background(), step)

	res := m.only(t)
	if res.Reason != contract.ReasonLowerChanged ||
		res.Error != "the lower changed after this run was matched on this node; resubmit the run" ||
		res.ExitCode != nil || res.Workspace != "" || res.ExitedAt != nil || res.Finalize != "" {
		t.Fatalf("result = %+v", res)
	}
	if res.Environment == nil || res.Environment.Runtime != "runc-overlay" {
		t.Fatalf("environment = %+v", res.Environment)
	}
	if rt.opens != 0 || len(m.exits()) != 0 {
		t.Fatalf("the refused step ran: opens %d exits %d", rt.opens, len(m.exits()))
	}
	if f.g.stepping != 0 {
		t.Fatalf("StepDone was not paired: stepping %d", f.g.stepping)
	}
}

// 상태 자리를 못 연 노드는 prepare 가 아닌 단계를 돌리지 않는다 — 원인 코드는 달지 않는다.
func TestLowerClaim_UnopenedStateIsReportedWithoutAReason(t *testing.T) {
	dir := t.TempDir()
	lowers := filepath.Join(dir, "lowers")
	if err := os.WriteFile(lowers, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	g := newLowerGuard(lowers, dir, Identity{NodeID: "n1"}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	g.start()
	m, w := finalizeWorker(t)
	rt := &trackingRuntime{}
	w.Runtime, w.Guard = rt, g
	w.execute(context.Background(), runStep("sh", "-c", "exit 0"))
	res := m.only(t)
	if res.Reason != "" || !strings.HasPrefix(res.Error, "cannot open the lower state directory: ") || rt.opens != 0 {
		t.Fatalf("result = %+v opens %d", res, rt.opens)
	}
}

// prepare 단계를 claim 하면 곧바로 놓는다 — 굽기 Run 의 세션은 공유 대신 굽기 잠금이 지킨다.
func TestLowerClaim_PrepareStepReleases(t *testing.T) {
	f := newGuardFixture(t)
	f.g.BeforeAdvert(nil)
	m, w := finalizeWorker(t)
	w.Runtime, w.Guard = &trackingRuntime{}, f.g
	step := runStep("/fake/build")
	step.Effect = contract.EffectPrepare
	w.execute(context.Background(), step)
	if res := m.only(t); res.Error != "" || res.Reason != "" {
		t.Fatalf("result = %+v", res)
	}
	if hs := f.holders(t); len(hs) != 0 {
		t.Fatalf("the prepare claim kept the lower lock: %+v", hs)
	}
}

// panicRuntime 은 세션을 열다 패닉을 낸다.
type panicRuntime struct{}

func (panicRuntime) Open(context.Context, RuntimeSpec) (StepSession, error) { panic("boom") }
func (panicRuntime) Capability() RuntimeCapability                          { return RuntimeCapability{} }

// 패닉에도 StepDone 이 돈다 — 도는 단계로 남아 놓는 울타리를 영영 막지 않는다.
func TestLowerClaim_PanicStillPairsStepDone(t *testing.T) {
	f := newGuardFixture(t)
	f.g.BeforeAdvert(nil)
	m, w := finalizeWorker(t)
	w.Runtime, w.Guard = panicRuntime{}, f.g
	w.safeExecute(context.Background(), runStep("sh"))
	if res := m.only(t); !strings.HasPrefix(res.Error, "adapter panic: boom") {
		t.Fatalf("result = %+v", res)
	}
	f.g.AfterResponse("graceful", nil)
	f.g.AfterResponse("graceful", nil)
	if hs := f.holders(t); len(hs) != 0 {
		t.Fatalf("a panicked step still counts as running: %+v", hs)
	}
}

// blockingRuntime 은 명령이 release 가 닫힐 때까지 도는 세션을 연다.
type blockingRuntime struct {
	started chan struct{}
	release chan struct{}
}

func (r blockingRuntime) Open(ctx context.Context, spec RuntimeSpec) (StepSession, error) {
	s, err := NativeRuntime{}.Open(ctx, spec)
	return &blockingSession{StepSession: s, r: r}, err
}

func (blockingRuntime) Capability() RuntimeCapability { return RuntimeCapability{Writes: "isolated"} }

type blockingSession struct {
	StepSession
	r blockingRuntime
}

func (s *blockingSession) Run(context.Context, ProcessSpec) (int, error) {
	close(s.r.started)
	<-s.r.release
	return 0, nil
}

// 도는 단계가 있는 동안은 응답 두 번에도 안 놓는다. 세션을 닫은 뒤의 응답에서 놓는다.
func TestLowerClaim_RunningStepHoldsOverTwoResponses(t *testing.T) {
	f := newGuardFixture(t)
	f.g.BeforeAdvert(nil)
	f.g.AfterResponse("", lease("r1"))
	m, w := finalizeWorker(t)
	rt := blockingRuntime{started: make(chan struct{}), release: make(chan struct{})}
	w.Runtime, w.Guard = rt, f.g
	done := make(chan struct{})
	go func() {
		w.execute(context.Background(), runStep("/fake/long"))
		close(done)
	}()
	<-rt.started
	if hs := f.holders(t); len(hs) != 1 || hs[0].Role != lower.RoleRun || hs[0].Run != "r1" {
		t.Fatalf("holders while running = %+v", hs)
	}
	f.g.AfterResponse("graceful", nil)
	f.g.AfterResponse("graceful", nil)
	if len(f.holders(t)) != 1 {
		t.Fatal("released while the session was open")
	}
	close(rt.release)
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the step did not finish")
	}
	if res := m.only(t); res.Error != "" {
		t.Fatalf("result = %+v", res)
	}
	f.g.AfterResponse("graceful", nil)
	if hs := f.holders(t); len(hs) != 0 {
		t.Fatalf("not released after the session closed: %+v", hs)
	}
}
