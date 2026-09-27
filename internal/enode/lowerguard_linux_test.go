package enode

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/lower"
)

// LowerGuard 의 시험 — 임시 폴더의 진짜 자리와 잠금으로 돈다 (FD 흐름 9.1 의 LowerGuard 줄).

// countIn 은 로그에 그 줄이 몇 번 나왔나다.
func countIn(l *lockedBuffer, msg string) int { return strings.Count(l.String(), msg) }

type guardFixture struct {
	ws, lowers string
	g          *LowerGuard
	log        *lockedBuffer
}

func newGuardFixture(t *testing.T) guardFixture {
	t.Helper()
	dir := t.TempDir()
	f := guardFixture{ws: filepath.Join(dir, "ws"), lowers: filepath.Join(dir, "lowers"), log: &lockedBuffer{}}
	if err := os.Mkdir(f.ws, 0o755); err != nil {
		t.Fatal(err)
	}
	f.g = newLowerGuard(f.lowers, f.ws, Identity{NodeID: "node-a", Label: "box-a"},
		slog.New(slog.NewTextHandler(f.log, nil)))
	f.g.start()
	return f
}

// sibling 은 같은 자리를 따로 연 Dir 이다 — 형제 노드 · 굽는 노드 · merge 를 흉내 낸다.
func (f guardFixture) sibling(t *testing.T) *lower.Dir {
	t.Helper()
	root, err := lower.ReadRoot(f.ws)
	if err != nil {
		t.Fatal(err)
	}
	d, err := lower.Open(f.lowers, root, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func (f guardFixture) holders(t *testing.T) []lower.Holder {
	t.Helper()
	hs, err := f.sibling(t).Holders()
	if err != nil {
		t.Fatal(err)
	}
	return hs
}

// writeState 는 굽는 노드처럼 굽기 잠금을 쥐고 state.json 을 쓴다.
func (f guardFixture) writeState(t *testing.T, phase lower.Phase, run string) {
	t.Helper()
	b, ok, err := f.sibling(t).TryBake()
	if err != nil || !ok {
		t.Fatalf("TryBake = %v, %v", ok, err)
	}
	defer func() { _ = b.Release() }()
	st := lower.State{Phase: phase, Since: time.Now()}
	if phase != lower.PhaseCommitted {
		st.Owner = &lower.Owner{Run: run, Node: "baker"}
	}
	if err := b.WriteState(st); err != nil {
		t.Fatal(err)
	}
}

// merged 는 합치기가 끝난 것처럼 metadata 를 쓴다 — 표지가 바뀐다.
func (f guardFixture) merged(t *testing.T, run string) {
	t.Helper()
	ir := "your-ir-tag"
	if err := lower.WriteMetadata(f.ws, lower.Metadata{Source: lower.Source{IR: &ir},
		Builds: []lower.BuildRecord{{Name: "config-a"}}, Bake: lower.BakeRecord{Run: run, MergedAt: time.Now().UTC()}}); err != nil {
		t.Fatal(err)
	}
}

func stepOf(run, kind string, effect contract.Effect) *Step {
	return &Step{RunID: run, StepID: run + "#01", Kind: kind, Effect: effect}
}

var lease = func(run string) []Lease { return []Lease{{RunID: run, Node: "node-a"}} }

// 후보면 쥐고, drain 이 받아 적히고 임대가 0 인 응답 두 번 뒤에 놓는다. 한 번이면 안 놓는다.
func TestLowerGuard_CandidateHoldsAndReleasesAfterTwoAcks(t *testing.T) {
	f := newGuardFixture(t)
	own, keys := f.g.BeforeAdvert(nil)
	if len(own) != 0 || keys != nil {
		t.Fatalf("a fresh candidate = %+v, %v", own, keys)
	}
	if hs := f.holders(t); len(hs) != 1 || hs[0].Role != lower.RoleCandidate || hs[0].Label != "box-a" {
		t.Fatalf("holders = %+v", hs)
	}
	f.g.AfterResponse("graceful", nil)
	f.g.BeforeAdvert([]DrainSource{{Kind: DrainOwner, Mode: "graceful"}})
	if hs := f.holders(t); len(hs) != 1 || hs[0].Acks != 1 {
		t.Fatalf("after one ack = %+v", hs)
	}
	// drain 이 풀린 응답 — 셈이 0 으로
	f.g.AfterResponse("", nil)
	f.g.AfterResponse("graceful", nil)
	if hs := f.holders(t); len(hs) != 1 {
		t.Fatalf("released after a broken streak: %+v", hs)
	}
	// 광고가 실패하면 AfterResponse 가 없다 — 셈은 그대로이고 다음 응답이 둘째다
	f.g.BeforeAdvert([]DrainSource{{Kind: DrainOwner, Mode: "graceful"}})
	f.g.AfterResponse("at-boundary", nil)
	if hs := f.holders(t); len(hs) != 0 {
		t.Fatalf("not released after two acks: %+v", hs)
	}
	if n := countIn(f.log, "released the lower lock: drain acknowledged twice and no lease"); n != 1 {
		t.Errorf("release logged %d times:\n%s", n, f.log)
	}
	// drain 이 남아 있으면 다시 쥐지 않는다 · 풀리면 다시 쥔다
	if own, _ := f.g.BeforeAdvert([]DrainSource{{Kind: DrainOwner, Mode: "graceful"}}); len(own) != 0 || len(f.holders(t)) != 0 {
		t.Fatalf("took the lock under a drain: %+v", own)
	}
	f.g.BeforeAdvert(nil)
	if hs := f.holders(t); len(hs) != 1 || countIn(f.log, "took the lower lock again") != 1 {
		t.Fatalf("not taken again: %+v\n%s", hs, f.log)
	}
}

// 도는 단계가 있으면 놓지 않는다 — 세션이 열린 채 놓으면 마운트 0 의 증거가 깨진다.
func TestLowerGuard_RunningStepKeepsTheLock(t *testing.T) {
	f := newGuardFixture(t)
	f.g.BeforeAdvert(nil)
	f.g.AfterResponse("", lease("R-1"))
	step := stepOf("R-1", "run", "")
	if err := f.g.OnClaim(step); err != nil {
		t.Fatal(err)
	}
	if hs := f.holders(t); len(hs) != 1 || hs[0].Role != lower.RoleRun || hs[0].Run != "R-1" {
		t.Fatalf("holders = %+v", hs)
	}
	// 취소된 Run 의 임대는 목록에서 먼저 빠진다 — 세션은 아직 열려 있다
	f.g.AfterResponse("graceful", nil)
	f.g.AfterResponse("graceful", nil)
	f.g.AfterResponse("graceful", nil)
	if hs := f.holders(t); len(hs) != 1 {
		t.Fatalf("released while a step runs: %+v", hs)
	}
	f.g.StepDone(step)
	f.g.AfterResponse("graceful", nil)
	if hs := f.holders(t); len(hs) != 0 {
		t.Fatalf("not released after the step: %+v", hs)
	}
}

// 임대가 0 이면 역할을 candidate 로 · Run 을 비운다 (계획 4절 ⑫).
func TestLowerGuard_NoLeaseMakesACandidate(t *testing.T) {
	f := newGuardFixture(t)
	f.g.BeforeAdvert(nil)
	f.g.AfterResponse("", lease("R-1"))
	if hs := f.holders(t); hs[0].Role != lower.RoleRun || hs[0].Run != "R-1" {
		t.Fatalf("holders = %+v", hs)
	}
	f.g.AfterResponse("", nil)
	if hs := f.holders(t); hs[0].Role != lower.RoleCandidate || hs[0].Run != "" {
		t.Fatalf("holders = %+v", hs)
	}
}

// pending 이면 bake 출처이고 공유를 새로 안 잡는다 — 기다리는 배타가 굶지 않는다 (계획 3.2 다섯째 줄).
func TestLowerGuard_PendingDrainsAndTheMergeGetsTheLock(t *testing.T) {
	f := newGuardFixture(t)
	f.g.BeforeAdvert(nil)
	f.writeState(t, lower.PhasePending, "R-9")
	merge := f.sibling(t)
	got := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		ex, err := merge.Exclusive(ctx, time.Hour, nil)
		if err == nil {
			_ = ex.Release()
		}
		got <- err
	}()
	for range 2 {
		own, _ := f.g.BeforeAdvert(nil)
		if len(own) != 1 || own[0].Kind != DrainBake || own[0].Detail != "lower pending a merge (run R-9)" ||
			own[0].Owner || own[0].Mode != "graceful" {
			t.Fatalf("own = %+v", own)
		}
		f.g.AfterResponse("graceful", nil)
	}
	if own, _ := f.g.BeforeAdvert(nil); len(own) != 1 {
		t.Fatalf("own = %+v", own)
	}
	if err := <-got; err != nil {
		t.Fatalf("the merge starved: %v", err)
	}
	if n := countIn(f.log, "the lower is pending a merge; draining this node"); n != 1 {
		t.Errorf("the pending line was logged %d times", n)
	}
	f.writeState(t, lower.PhaseMerging, "R-9")
	if own, _ := f.g.BeforeAdvert(nil); len(own) != 1 || own[0].Detail != "lower being merged (run R-9)" {
		t.Fatalf("merging own = %+v", own)
	}
	f.writeState(t, lower.PhaseCommitted, "")
	f.merged(t, "R-9")
	own, keys := f.g.BeforeAdvert(nil)
	if len(own) != 0 || keys["ir"] != "your-ir-tag" || keys["bake.run"] != "R-9" || keys["repo.built.config-a"] != "yes" {
		t.Fatalf("after the merge = %+v, %v", own, keys)
	}
	if len(f.holders(t)) != 1 {
		t.Fatal("not taken after the merge")
	}
}

// 배타가 쥐어져 있으면 공유를 못 잡고 bake 출처 (being merged) 다.
func TestLowerGuard_HeldExclusiveDrains(t *testing.T) {
	f := newGuardFixture(t)
	ex, err := f.sibling(t).Exclusive(context.Background(), time.Hour, nil)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		own, _ := f.g.BeforeAdvert(nil)
		if len(own) != 1 || own[0].Kind != DrainBake || own[0].Detail != "lower is being merged" {
			t.Fatalf("own = %+v", own)
		}
	}
	if n := countIn(f.log, "the lower lock is held by a merge; draining this node"); n != 1 {
		t.Errorf("logged %d times", n)
	}
	_ = ex.Release()
	if own, _ := f.g.BeforeAdvert(nil); len(own) != 0 {
		t.Fatalf("own = %+v", own)
	}
}

// 자리를 못 열면 lower 출처이고 다음 광고에 다시 연다. state.json 을 못 읽어도 lower 출처다.
func TestLowerGuard_UnopenedStateDirectory(t *testing.T) {
	dir := t.TempDir()
	ws, lowers := filepath.Join(dir, "ws"), filepath.Join(dir, "lowers")
	if err := os.Mkdir(ws, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lowers, nil, 0o600); err != nil { // lowers 자리에 파일
		t.Fatal(err)
	}
	log := &lockedBuffer{}
	g := newLowerGuard(lowers, ws, Identity{NodeID: "node-a"}, slog.New(slog.NewTextHandler(log, nil)))
	g.start()
	for range 2 {
		own, _ := g.BeforeAdvert(nil)
		if len(own) != 1 || own[0].Kind != DrainLower || !strings.HasPrefix(own[0].Detail, "cannot open the lower state: ") {
			t.Fatalf("own = %+v", own)
		}
	}
	if n := countIn(log, "cannot open the lower state; draining this node"); n != 1 {
		t.Errorf("logged %d times", n)
	}
	// 자리를 못 연 노드는 prepare 가 아닌 단계를 돌리지 않는다 — 원인 코드는 달지 않는다
	err := g.OnClaim(stepOf("R-1", "run", ""))
	var changed *LowerChangedError
	if err == nil || errors.As(err, &changed) || !strings.HasPrefix(err.Error(), "cannot open the lower state directory: ") {
		t.Fatalf("OnClaim = %v", err)
	}
	g.StepDone(nil)
	if err := os.Remove(lowers); err != nil {
		t.Fatal(err)
	}
	if own, _ := g.BeforeAdvert(nil); len(own) != 0 || g.shared == nil {
		t.Fatalf("not reopened: %+v", own)
	}
	// state.json 이 깨졌다
	if err := os.WriteFile(filepath.Join(g.dir.Path, "state.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if own, _ := g.BeforeAdvert(nil); len(own) != 1 || own[0].Kind != DrainLower {
		t.Fatalf("broken state = %+v", own)
	}

	// home 을 못 찾았다
	g2 := newLowerGuard("", ws, Identity{NodeID: "node-b"}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	g2.homeErr = errors.New("$HOME is not defined")
	g2.start()
	if own, _ := g2.BeforeAdvert(nil); len(own) != 1 || own[0].Detail != "cannot find the home directory: $HOME is not defined" {
		t.Fatalf("home = %+v", own)
	}
	if err := g2.OnClaim(stepOf("R-1", "run", "")); err == nil || err.Error() != "cannot open the lower state directory: $HOME is not defined" {
		t.Fatalf("OnClaim without a home = %v", err)
	}
}

// 늦게 보인 임대 네 장면 (FD 흐름 4절). 놓은 뒤 · 기동 뒤 첫 응답도 같은 길이다.
func TestLowerGuard_LateLeases(t *testing.T) {
	released := func(t *testing.T) guardFixture {
		f := newGuardFixture(t)
		f.g.BeforeAdvert(nil)
		f.g.AfterResponse("graceful", nil)
		f.g.AfterResponse("graceful", nil)
		if f.g.shared != nil {
			t.Fatal("not released")
		}
		return f
	}

	t.Run("pending: taken again", func(t *testing.T) {
		f := released(t)
		f.writeState(t, lower.PhasePending, "R-9")
		f.g.AfterResponse("graceful", lease("R-1"))
		if hs := f.holders(t); len(hs) != 1 || hs[0].Role != lower.RoleRun || hs[0].Run != "R-1" {
			t.Fatalf("holders = %+v", hs)
		}
		if err := f.g.OnClaim(stepOf("R-1", "run", "")); err != nil {
			t.Fatal(err)
		}
		if n := countIn(f.log, "a lease arrived after the lower lock was released; taking it again"); n != 1 {
			t.Errorf("logged %d times", n)
		}
	})
	t.Run("merging: refused", func(t *testing.T) {
		f := released(t)
		f.writeState(t, lower.PhaseMerging, "R-9")
		err := f.g.OnClaim(stepOf("R-1", "run", ""))
		var changed *LowerChangedError
		if !errors.As(err, &changed) || changed.Run != "R-1" ||
			err.Error() != "the lower changed after this run was matched on this node; resubmit the run" {
			t.Fatalf("OnClaim = %v", err)
		}
		if !strings.Contains(f.log.String(), "why=\"the lower is being merged\"") {
			t.Errorf("log:\n%s", f.log)
		}
	})
	t.Run("merged after the release: refused", func(t *testing.T) {
		f := released(t)
		f.merged(t, "R-9")
		f.g.AfterResponse("graceful", lease("R-1"))
		if len(f.holders(t)) != 0 {
			t.Fatal("took the lock over a changed lower")
		}
		if err := f.g.OnClaim(stepOf("R-1", "run", "")); !errors.As(err, new(*LowerChangedError)) {
			t.Fatalf("OnClaim = %v", err)
		}
		// 거절 표는 그 Run 의 임대가 사라지면 지운다
		f.g.AfterResponse("graceful", nil)
		if f.g.refused["R-1"] {
			t.Error("the refusal outlived the lease")
		}
		if n := countIn(f.log, "refusing a run: the lower changed after it was matched"); n != 1 {
			t.Errorf("logged %d times", n)
		}
	})
	t.Run("exclusive held: refused", func(t *testing.T) {
		f := released(t)
		ex, err := f.sibling(t).Exclusive(context.Background(), time.Hour, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = ex.Release() }()
		if err := f.g.OnClaim(stepOf("R-1", "run", "")); !errors.As(err, new(*LowerChangedError)) {
			t.Fatalf("OnClaim = %v", err)
		}
	})
	t.Run("first response after start: already held", func(t *testing.T) {
		f := newGuardFixture(t)
		f.g.BeforeAdvert(nil)
		f.g.AfterResponse("", lease("R-0"))
		if hs := f.holders(t); len(hs) != 1 || hs[0].Run != "R-0" {
			t.Fatalf("holders = %+v", hs)
		}
	})
	t.Run("first response after start: not held", func(t *testing.T) {
		f := newGuardFixture(t)
		f.g.AfterResponse("graceful", lease("R-0"))
		if hs := f.holders(t); len(hs) != 1 || hs[0].Run != "R-0" {
			t.Fatalf("holders = %+v", hs)
		}
	})
}

// prepare 단계와 merge 단계를 claim 하면 곧바로 놓고, 거절하지 않는다 (결정 3-9).
func TestLowerGuard_BakeStepsReleaseAndAreNotRefused(t *testing.T) {
	f := newGuardFixture(t)
	f.g.BeforeAdvert(nil)
	f.g.AfterResponse("", lease("R-B"))
	build := stepOf("R-B", "build", contract.EffectPrepare)
	if err := f.g.OnClaim(build); err != nil {
		t.Fatal(err)
	}
	if hs := f.holders(t); len(hs) != 0 {
		t.Fatalf("the build claim kept the lock: %+v", hs)
	}
	// 굽기 Run 의 임대는 늦게 보인 임대가 아니다
	f.g.AfterResponse("", lease("R-B"))
	if len(f.holders(t)) != 0 || f.g.refused["R-B"] {
		t.Fatal("the bake run's own lease was treated as late")
	}
	f.g.StepDone(build)
	// 거절 표에 있어도 merge 단계는 돈다
	f.g.refused["R-B"] = true
	if err := f.g.OnClaim(stepOf("R-B", "merge", "")); err != nil {
		t.Fatalf("a merge step was refused: %v", err)
	}
	for _, s := range []*Step{stepOf("x", "build", ""), stepOf("x", "merge", ""), stepOf("x", "run", contract.EffectPrepare)} {
		if !bakeStep(s) {
			t.Errorf("bakeStep(%s, %s) = false", s.Kind, s.Effect)
		}
	}
	if bakeStep(stepOf("x", "agent", contract.EffectEdit)) {
		t.Error("an agent step is a bake step")
	}
}

// HoldBake 부터 DropBake 까지 공유를 쥐지 않는다 (계획 4절 ⑪). HoldBake 는 쥔 공유를 놓는다.
func TestLowerGuard_HoldBakeKeepsTheLockOff(t *testing.T) {
	f := newGuardFixture(t)
	f.g.BeforeAdvert(nil)
	f.g.HoldBake("R-B", nil)
	if len(f.holders(t)) != 0 {
		t.Fatal("HoldBake kept the shared lock")
	}
	if own, _ := f.g.BeforeAdvert(nil); len(own) != 0 || len(f.holders(t)) != 0 {
		t.Fatalf("took the lock while baking: %+v", own)
	}
	f.g.AfterResponse("", lease("R-B"))
	if len(f.holders(t)) != 0 {
		t.Fatal("the bake run's lease took the lock")
	}
	f.merged(t, "R-B")
	f.g.DropBake()
	if f.g.mark.Run != "R-B" {
		t.Errorf("DropBake did not take the new mark: %+v", f.g.mark)
	}
	f.g.BeforeAdvert(nil)
	if len(f.holders(t)) != 1 {
		t.Fatal("not taken after DropBake")
	}
}

// 합치기 없이 끝난 굽기 — pending 이면 abandon 을 한 번 부른다. building · merging 이면 안 부른다.
func TestLowerGuard_BakeWithoutAMerge(t *testing.T) {
	for _, c := range []struct {
		phase lower.Phase
		calls int
	}{
		{lower.PhasePending, 1},
		{lower.PhaseBuilding, 0},
		{lower.PhaseMerging, 0},
	} {
		t.Run(string(c.phase), func(t *testing.T) {
			f := newGuardFixture(t)
			f.writeState(t, c.phase, "R-B")
			called := make(chan struct{}, 4)
			f.g.HoldBake("R-B", func() { called <- struct{}{} })
			f.g.AfterResponse("", lease("R-B")) // 아직 있다
			f.g.AfterResponse("", nil)          // 사라졌다
			f.g.AfterResponse("", nil)
			time.Sleep(20 * time.Millisecond)
			if len(called) != c.calls {
				t.Fatalf("abandon ran %d times, want %d", len(called), c.calls)
			}
			if c.calls == 1 && (f.g.bake != nil || countIn(f.log, "discarding the upper") != 1) {
				t.Errorf("bake %+v\n%s", f.g.bake, f.log)
			}
		})
	}
}

// nil 수신자는 아무것도 안 한다 — native 노드 · 시험.
func TestLowerGuard_Nil(t *testing.T) {
	var g *LowerGuard
	if own, keys := g.BeforeAdvert([]DrainSource{{Kind: DrainOwner}}); own != nil || keys != nil {
		t.Fatal("a nil guard returned something")
	}
	g.AfterResponse("graceful", nil)
	if err := g.OnClaim(stepOf("R", "run", "")); err != nil {
		t.Fatal(err)
	}
	g.StepDone(nil)
	g.HoldBake("R", nil)
	g.DropBake()
	if StartLowerGuard("", Identity{}, nil) != nil {
		t.Fatal("a node without a lower root got a guard")
	}
}

// StartLowerGuard 는 $HOME 아래 자리를 연다. 연 뒤 놓은 상태로 시작한다.
func TestStartLowerGuard(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	ws := filepath.Join(home, "ws")
	if err := os.Mkdir(ws, 0o755); err != nil {
		t.Fatal(err)
	}
	g := StartLowerGuard(ws, Identity{NodeID: "node-a"}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if g == nil || g.dir == nil || !strings.HasPrefix(g.dir.Path, filepath.Join(home, ".local", "state", "enode", "lowers")) ||
		!g.released || g.shared != nil {
		t.Fatalf("guard = %+v", g)
	}
	t.Setenv("HOME", "")
	g = StartLowerGuard(ws, Identity{NodeID: "node-a"}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if own, _ := g.BeforeAdvert(nil); len(own) != 1 || !strings.HasPrefix(own[0].Detail, "cannot find the home directory: ") {
		t.Fatalf("without HOME = %+v", own)
	}
}

// 광고 본문에 굽기 출처와 metadata 키가 실리고, 실패한 광고는 놓는 셈을 안 바꾼다 (business-rules.md 5.2).
// 진짜 잠금이 필요해 이 파일(linux)에 둔다 — drain_test.go 의 광고 시험과 같은 서버를 쓴다.
func TestDrain_BakeSourceKeysAndFailedAdverts(t *testing.T) {
	var failing atomic.Bool
	srv, seen := bodyServer(t, failing.Load)
	defer srv.Close()
	dir := t.TempDir()
	ws, lowers := filepath.Join(dir, "ws"), filepath.Join(dir, "lowers")
	if err := os.Mkdir(ws, 0o755); err != nil {
		t.Fatal(err)
	}
	ir := "your-ir-tag"
	if err := lower.WriteMetadata(ws, lower.Metadata{Source: lower.Source{IR: &ir},
		Builds: []lower.BuildRecord{{Name: "config-a"}}, Bake: lower.BakeRecord{Run: "R-1"}}); err != nil {
		t.Fatal(err)
	}
	f := guardFixture{ws: ws, lowers: lowers, log: &lockedBuffer{}}
	f.g = newLowerGuard(lowers, ws, Identity{NodeID: "n1"}, slog.New(slog.NewTextHandler(f.log, nil)))
	f.g.start()
	cfg := filepath.Join(dir, "local.yaml")
	a := &Advertiser{
		Client: &Client{Base: srv.URL, Token: "t", Principal: "p", HTTP: srv.Client()},
		Ident:  Identity{NodeID: "n1", Label: "l", Config: cfg},
		Caps: func() Capabilities {
			return Capabilities{Caps: []contract.Capability{{Capability: contract.CapabilityAgentReason,
				Attrs: map[string]string{"harness.claude": "2.1"}}}}
		},
		Every:  time.Millisecond,
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		Guard:  f.g,
		Writes: "isolated",
	}
	// 첫 광고가 공유를 쥐고 metadata 키를 싣는다
	runAdverts(t, a, 1, drainsOf(seen))
	attrs := seen()[0].Capabilities[0].Attrs
	if attrs["ir"] != "your-ir-tag" || attrs["repo.built.config-a"] != "yes" || attrs["bake.run"] != "R-1" ||
		attrs["bake.resumed"] != "false" || attrs["workspace.writes"] != "isolated" {
		t.Fatalf("attrs = %v", attrs)
	}
	if len(f.holders(t)) != 1 {
		t.Fatal("the first advert did not take the lower lock")
	}
	// 굽는 노드가 pending 을 쓴다 — 광고가 실패하는 동안은 놓지 않는다
	f.writeState(t, lower.PhasePending, "R-2")
	failing.Store(true)
	n := len(seen())
	runAdverts(t, a, n+4, drainsOf(seen))
	if len(f.holders(t)) != 1 {
		t.Fatal("failed adverts released the lower lock")
	}
	if d := seen()[n].Policy.Drain; d != contract.DrainGraceful {
		t.Fatalf("a pending lower did not drain: %q", d)
	}
	failing.Store(false)
	n = len(seen())
	runAdverts(t, a, n+3, drainsOf(seen), func() bool { return len(f.holders(t)) == 0 })
	status, err := ReadStatus(cfg)
	if err != nil || status.Drain == nil || len(status.Drain.Sources) != 1 || status.Drain.Sources[0].Kind != DrainBake ||
		status.Drain.Sources[0].Detail != "lower pending a merge (run R-2)" {
		t.Fatalf("status drain = %+v err = %v", status.Drain, err)
	}
}

// 깨진 metadata 면 네 키를 다 안 싣고 원인이 바뀔 때 한 줄 · 어긋난 값은 그 키만 빼고 한 줄 (business-rules.md 11절).
func TestLowerGuard_MetadataKeys(t *testing.T) {
	f := newGuardFixture(t)
	p := filepath.Join(f.ws, ".enode-metadata.json")
	if err := os.WriteFile(p, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, keys := f.g.BeforeAdvert(nil); keys != nil {
			t.Fatalf("keys from broken metadata = %v", keys)
		}
	}
	if n := countIn(f.log, "cannot read .enode-metadata.json; not advertising ir, repo.built and bake keys"); n != 1 {
		t.Errorf("logged %d times", n)
	}
	bad := "bad ir"
	if err := lower.WriteMetadata(f.ws, lower.Metadata{Source: lower.Source{IR: &bad},
		Builds: []lower.BuildRecord{{Name: "config-a"}}}); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, keys := f.g.BeforeAdvert(nil); len(keys) != 1 || keys["repo.built.config-a"] != "yes" {
			t.Fatalf("keys = %v", keys)
		}
	}
	if n := countIn(f.log, "ignoring values in .enode-metadata.json; not advertising those keys"); n != 1 {
		t.Errorf("logged %d times:\n%s", n, f.log)
	}
}

// BenchmarkBeforeAdvert 는 광고 한 번의 몫이다 — state.json 읽기 · 쥔 사람 기록 맞추기 · metadata 읽기 (계획 3.2 ·
// FD 흐름 12절이 NFR 에 둔 「광고마다 metadata 읽기」). 임시 폴더의 진짜 자리에서 공유를 쥔 채 측정한다.
func BenchmarkBeforeAdvert(b *testing.B) {
	dir := b.TempDir()
	ws := filepath.Join(dir, "ws")
	if err := os.Mkdir(ws, 0o755); err != nil {
		b.Fatal(err)
	}
	ir := "your-ir-tag"
	if err := lower.WriteMetadata(ws, lower.Metadata{Source: lower.Source{IR: &ir},
		Builds: []lower.BuildRecord{{Name: "config-a"}, {Name: "config-b"}}, Bake: lower.BakeRecord{Run: "R-1"}}); err != nil {
		b.Fatal(err)
	}
	g := newLowerGuard(filepath.Join(dir, "lowers"), ws, Identity{NodeID: "node-a", Label: "a"},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	g.start()
	if own, keys := g.BeforeAdvert(nil); len(own) != 0 || len(keys) != 5 || g.shared == nil {
		b.Fatalf("setup = %+v %v", own, keys)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.BeforeAdvert(nil)
	}
}
