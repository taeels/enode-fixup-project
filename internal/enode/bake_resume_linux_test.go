//go:build linux

package enode

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/lower"
	"golang.org/x/sys/unix"
)

// 재개 · 광고 주기의 정리 (bake 유닛 · FD 규칙 10 · 12.3 · 12.4 · 12.5 · 13절 · FD 흐름 6절). 배경 일은 Baker 의 wg 로
// 기다린다 (계획 5절 규칙 3). onStale 은 시험이 직접 부른다.

// interrupted 는 주인이 죽은 굽기를 짓는다 — 대기 자리 (upper 에 built/out.bin · 초안이면 R-dead 의 초안) 와 그
// phase 의 state.json. 돌려주는 것은 대기 자리다.
func (f *bakeFixture) interrupted(t *testing.T, phase lower.Phase, scratchDir string, draft, upper bool) string {
	t.Helper()
	pending := f.pendingIn(t, scratchDir, "R-dead", draft)
	if !upper {
		if err := os.RemoveAll(filepath.Join(pending, "upper")); err != nil {
			t.Fatal(err)
		}
	}
	f.putState(t, lower.State{Phase: phase, Owner: &lower.Owner{Run: "R-dead", Node: "node-k"},
		PendingUpper: filepath.Join(pending, "upper")})
	return pending
}

// stale 은 광고 주기 한 번이다 — onStale 을 부르고 배경 일을 기다린다.
func (f *bakeFixture) staleOnce(t *testing.T) {
	t.Helper()
	f.b.onStale(f.state(t))
	f.b.wg.Wait()
}

func (f *bakeFixture) retryAt() time.Time {
	f.b.mu.Lock()
	defer f.b.mu.Unlock()
	return f.b.retryAt
}

// failedResume 은 재개가 실패한 모양이다 — merging 그대로 · 두 잠금이 풀림 · retryAt 이 10분 뒤.
func failedResume(t *testing.T, f *bakeFixture) {
	t.Helper()
	if st := f.state(t); st.Phase != lower.PhaseMerging || st.Owner.Run != "R-dead" {
		t.Fatalf("state = %+v", st)
	}
	if at := f.retryAt(); time.Until(at) < 9*time.Minute || time.Until(at) > 11*time.Minute {
		t.Fatalf("retryAt is %s away", time.Until(at))
	}
	if !f.nodeLogHas("resume failed; trying again in 10m") {
		t.Fatalf("node log = %s", f.nodeLog)
	}
	released(t, f)
}

// resumed 는 재개가 끝낸 모양이다 — committed · 대기 자리가 trash · 두 잠금이 풀림 · 노드 로그 한 줄.
func resumed(t *testing.T, f *bakeFixture, pending string) {
	t.Helper()
	if st := f.state(t); st.Phase != lower.PhaseCommitted || st.LastAttempt != nil {
		t.Fatalf("state = %+v", st)
	}
	if _, err := os.Stat(pending); !os.IsNotExist(err) {
		t.Fatalf("the pending directory stayed: %v", err)
	}
	if !f.nodeLogHas("bake: resumed the merge of run R-dead") || !f.retryAt().IsZero() {
		t.Fatalf("node log = %s", f.nodeLog)
	}
	released(t, f)
}

// 워크스페이스가 lower 를 가리키는 symlink 인 형제가 끊긴 합치기를 한 번에 끝낸다 — 10분 재시도 없이. 조각 7 에서
// symlink 형제의 재개가 합친 뒤 metadata 쓰기에서 ENOTDIR 로 멈추고 10분 뒤의 재개가 committed 로 옮겼다.
func TestResume_ALinkedSiblingFinishesAtOnce(t *testing.T) {
	f := newBakeFixtureAt(t, true)
	useTestMergeHelper(t, "")
	pending := f.interrupted(t, lower.PhaseMerging, f.scratch, true, true)
	f.staleOnce(t)
	resumed(t, f, pending)
	if b, _ := os.ReadFile(filepath.Join(f.ws, "built", "out.bin")); string(b) != "out" {
		t.Fatalf("the upper was not merged: %q", b)
	}
	md, err := lower.ReadMetadata(f.ws)
	if err != nil || md == nil || !md.Bake.Resumed || md.Bake.Run != "R-dead" {
		t.Fatalf("metadata = %+v %v", md, err)
	}
	if fi, err := os.Lstat(f.nodeWS); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("the workspace symlink was replaced: %v %v", fi, err)
	}
}

// 끝났는지 먼저 본다 (FD 규칙 12.4 의 넷) · run_id 만 같은 metadata 는 끝나지 않은 것이다.
func TestResume_WhatIsLeft(t *testing.T) {
	t.Run("no draft fails and stays merging", func(t *testing.T) {
		f := newBakeFixture(t)
		useTestMergeHelper(t, "")
		f.interrupted(t, lower.PhaseMerging, f.scratch, false, true)
		f.staleOnce(t)
		failedResume(t, f)
		if !f.nodeLogHas("cannot read the bake draft") {
			t.Fatalf("node log = %s", f.nodeLog)
		}
	})
	t.Run("already finished writes committed only", func(t *testing.T) {
		f := newBakeFixture(t)
		useTestMergeHelper(t, "")
		pending := f.interrupted(t, lower.PhaseMerging, f.scratch, true, true)
		d := fullDraft()
		d.Run = "R-dead"
		if err := lower.WriteMetadata(f.ws, metadataOf(d, "R-dead", "node-k", time.Now(), false)); err != nil {
			t.Fatal(err)
		}
		f.staleOnce(t)
		resumed(t, f, pending)
		if _, err := os.Stat(filepath.Join(f.ws, "built")); !os.IsNotExist(err) {
			t.Fatalf("a finished merge was applied again: %v", err)
		}
	})
	t.Run("no upper root starts at the metadata", func(t *testing.T) {
		f := newBakeFixture(t)
		useTestMergeHelper(t, "")
		pending := f.interrupted(t, lower.PhaseMerging, f.scratch, true, false)
		f.staleOnce(t)
		resumed(t, f, pending)
		if md, _ := lower.ReadMetadata(f.ws); md == nil || !md.Bake.Resumed || md.Bake.Run != "R-dead" {
			t.Fatalf("metadata = %+v", md)
		}
	})
	t.Run("an upper root is merged, even an empty one", func(t *testing.T) {
		f := newBakeFixture(t)
		useTestMergeHelper(t, "")
		pending := f.interrupted(t, lower.PhaseMerging, f.scratch, true, true)
		f.staleOnce(t)
		resumed(t, f, pending)
		if b, _ := os.ReadFile(filepath.Join(f.ws, "built", "out.bin")); string(b) != "out" {
			t.Fatalf("the upper was not merged: %q", b)
		}
		f2 := newBakeFixture(t)
		p2 := f2.interrupted(t, lower.PhaseMerging, f2.scratch, true, false)
		if err := os.Mkdir(filepath.Join(p2, "upper"), 0o700); err != nil {
			t.Fatal(err)
		}
		f2.staleOnce(t)
		resumed(t, f2, p2)
	})
	t.Run("the same run id with another sync is not finished", func(t *testing.T) {
		f := newBakeFixture(t)
		useTestMergeHelper(t, "")
		pending := f.interrupted(t, lower.PhaseMerging, f.scratch, true, true)
		d := fullDraft()
		d.Source.SyncedAt = d.Source.SyncedAt.Add(-time.Hour)
		if err := lower.WriteMetadata(f.ws, metadataOf(d, "R-dead", "node-k", time.Now(), false)); err != nil {
			t.Fatal(err)
		}
		f.staleOnce(t)
		resumed(t, f, pending)
		if _, err := os.Stat(filepath.Join(f.ws, "built", "out.bin")); err != nil {
			t.Fatalf("a rewritten run id stopped the merge: %v", err)
		}
	})
}

// 재개의 metadata — resumed true · node 는 초안의 Node · previous_ir 은 초안 · run 은 주인 Run (FD 규칙 12.5).
func TestResume_Metadata(t *testing.T) {
	f := newBakeFixture(t)
	useTestMergeHelper(t, "")
	f.interrupted(t, lower.PhaseMerging, f.scratch, true, true)
	f.staleOnce(t)
	md, err := lower.ReadMetadata(f.ws)
	if err != nil || md == nil {
		t.Fatal(err)
	}
	d := fullDraft()
	if !md.Bake.Resumed || md.Bake.Run != "R-dead" || md.Bake.Node != d.Node || *md.Bake.PreviousIR != *d.PreviousIR ||
		md.Source.Head != d.Source.Head || md.Environment != d.Environment {
		t.Fatalf("metadata = %+v bake %+v", md, md.Bake)
	}
}

// 모양이 틀린 pending_upper 는 실패다 — 옮기지도 합치지도 않는다 (계획 4.1 19번).
func TestResume_AnOddPendingPathIsNotMerged(t *testing.T) {
	f := newBakeFixture(t)
	useTestMergeHelper(t, "")
	odd := filepath.Join(filepath.Dir(f.ws), "elsewhere", "upper")
	write(t, odd, "precious", "keep")
	f.putState(t, lower.State{Phase: lower.PhaseMerging, Owner: &lower.Owner{Run: "R-dead"}, PendingUpper: odd})
	f.staleOnce(t)
	failedResume(t, f)
	if b, _ := os.ReadFile(filepath.Join(odd, "precious")); string(b) != "keep" {
		t.Fatal("the odd path was touched")
	}
	if _, err := os.Stat(filepath.Join(f.ws, "precious")); !os.IsNotExist(err) {
		t.Fatal("the odd path was merged")
	}
}

// 재개의 시작 전 확인이 어긋나거나 helper 를 못 띄워도 upper 를 버리지 않는다 — merging 그대로 · 10분 뒤 (FD 규칙
// 10절 표 · 답 2). merge 단계는 같은 결과에서 upper 를 버린다.
func TestResume_PreflightFailuresKeepTheUpper(t *testing.T) {
	t.Run("a mark in the upper", func(t *testing.T) {
		f := newBakeFixture(t)
		useTestMergeHelper(t, "")
		pending := f.interrupted(t, lower.PhaseMerging, f.scratch, true, true)
		if err := unix.Lsetxattr(filepath.Join(pending, "upper", "built", "out.bin"), "user.overlay.metacopy", nil, 0); err != nil {
			t.Fatal(err)
		}
		f.staleOnce(t)
		failedResume(t, f)
		if _, err := os.Stat(filepath.Join(pending, "upper", "built", "out.bin")); err != nil {
			t.Fatalf("the upper went: %v", err)
		}
	})
	t.Run("the helper cannot start", func(t *testing.T) {
		f := newBakeFixture(t)
		was := mergeHelperCommand
		mergeHelperCommand = func() ([]string, error) { return []string{filepath.Join(t.TempDir(), "no-such")}, nil }
		t.Cleanup(func() { mergeHelperCommand = was })
		pending := f.interrupted(t, lower.PhaseMerging, f.scratch, true, true)
		f.staleOnce(t)
		failedResume(t, f)
		if _, err := os.Stat(filepath.Join(pending, "upper")); err != nil {
			t.Fatalf("the upper went: %v", err)
		}
	})
}

// 같은 오류가 되풀이되면 로그는 한 번이고, 오류가 바뀌면 한 번 더다.
func TestResume_TheSameErrorLogsOnce(t *testing.T) {
	f := newBakeFixture(t)
	useTestMergeHelper(t, "")
	f.interrupted(t, lower.PhaseMerging, f.scratch, false, true)
	again := func() {
		f.b.mu.Lock()
		f.b.retryAt = time.Time{}
		f.b.mu.Unlock()
		f.staleOnce(t)
	}
	f.staleOnce(t)
	again()
	if n := countIn(f.nodeLog, "resume failed; trying again in 10m"); n != 1 {
		t.Fatalf("logged %d times:\n%s", n, f.nodeLog)
	}
	d := fullDraft()
	d.Schema = 7
	if err := writeDraft(filepath.Dir(f.state(t).PendingUpper), d); err != nil {
		t.Fatal(err)
	}
	again()
	if n := countIn(f.nodeLog, "resume failed; trying again in 10m"); n != 2 {
		t.Fatalf("a changed error logged %d times in all:\n%s", n, f.nodeLog)
	}
}

// onStale 이 아무것도 안 하는 때 — 이 프로세스가 굽기를 쥠 · 재개 중 · retryAt 전 · TryBake 가 안 됨 (형제가 쥠) ·
// TryBake 오류 (로그 한 줄 · 간격 없음) · 잠금 뒤 다시 읽은 state 가 committed (놓는다).
func TestOnStale_WhenItDoesNothing(t *testing.T) {
	untouched := func(t *testing.T, f *bakeFixture) {
		t.Helper()
		if st := f.state(t); st.Phase != lower.PhaseMerging {
			t.Fatalf("state = %+v", st)
		}
	}
	t.Run("this process holds a bake", func(t *testing.T) {
		f := newBakeFixture(t)
		f.interrupted(t, lower.PhaseMerging, f.scratch, true, true)
		f.b.mu.Lock()
		f.b.held = &heldBake{b: f.b, run: "R-mine"}
		f.b.mu.Unlock()
		f.staleOnce(t)
		untouched(t, f)
		f.b.mu.Lock()
		f.b.held = nil
		f.b.mu.Unlock()
	})
	t.Run("already resuming", func(t *testing.T) {
		f := newBakeFixture(t)
		f.interrupted(t, lower.PhaseMerging, f.scratch, true, true)
		f.b.mu.Lock()
		f.b.resuming = true
		f.b.mu.Unlock()
		f.staleOnce(t)
		untouched(t, f)
		f.b.mu.Lock()
		f.b.resuming = false
		f.b.mu.Unlock()
	})
	t.Run("before retryAt", func(t *testing.T) {
		f := newBakeFixture(t)
		f.interrupted(t, lower.PhaseMerging, f.scratch, true, true)
		f.b.mu.Lock()
		f.b.retryAt = time.Now().Add(time.Minute)
		f.b.mu.Unlock()
		f.staleOnce(t)
		untouched(t, f)
	})
	t.Run("a sibling holds the bake lock", func(t *testing.T) {
		f := newBakeFixture(t)
		f.interrupted(t, lower.PhaseMerging, f.scratch, true, true)
		lock := f.holdBakeLock(t)
		f.staleOnce(t)
		untouched(t, f)
		if !f.retryAt().IsZero() || f.nodeLogHas("resume failed") {
			t.Fatal("a live owner counted as a failure")
		}
		_ = lock.Release()
	})
	t.Run("TryBake fails", func(t *testing.T) {
		f := newBakeFixture(t)
		f.interrupted(t, lower.PhaseMerging, f.scratch, true, true)
		p := filepath.Join(f.g.Dir().Path, "bake.lock")
		if err := os.Remove(p); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(p, 0o700); err != nil {
			t.Fatal(err)
		}
		f.b.onStale(lower.State{})
		f.b.wg.Wait()
		f.b.onStale(lower.State{})
		f.b.wg.Wait()
		if n := countIn(f.nodeLog, "bake: cannot take the bake lock"); n != 1 || !f.retryAt().IsZero() {
			t.Fatalf("logged %d times, retryAt %v", n, f.retryAt())
		}
	})
	t.Run("the state is committed again after the lock", func(t *testing.T) {
		f := newBakeFixture(t)
		f.b.onStale(lower.State{Phase: lower.PhaseMerging})
		f.b.wg.Wait()
		if st := f.state(t); st.Phase != lower.PhaseCommitted || f.nodeLogHas("resuming") {
			t.Fatalf("state = %+v", st)
		}
		f.holdBakeLock(t)
	})
}

// onStale 은 곧바로 돌아온다 — 배경 일이 막힌 채 onStale 과 BeforeAdvert 가 돌아온다 · 막힌 동안의 다음 onStale 은
// 아무것도 안 연다 (resuming) · 풀면 끝나고 Baker.Wait 가 돌아온다 · Wait 뒤의 onStale 은 아무것도 안 연다 (closed).
func TestOnStale_ReturnsAtOnce(t *testing.T) {
	f := newBakeFixture(t)
	useTestMergeHelper(t, "")
	pending := f.interrupted(t, lower.PhaseMerging, f.scratch, true, true)
	block, entered := make(chan struct{}), make(chan struct{}, 4)
	was := foreignMounts
	foreignMounts = func(lower.Root) (lower.MountScan, error) {
		entered <- struct{}{}
		<-block
		return lower.MountScan{}, nil
	}
	t.Cleanup(func() { foreignMounts = was })
	began := time.Now()
	f.g.BeforeAdvert(nil) // g.mu 아래에서 onStale 을 곧바로 부른다
	if took := time.Since(began); took > time.Second {
		t.Fatalf("BeforeAdvert waited %s", took)
	}
	<-entered
	f.g.BeforeAdvert(nil)
	f.b.onStale(lower.State{Phase: lower.PhaseMerging})
	select {
	case <-entered:
		t.Fatal("a second background task opened while one was running")
	case <-time.After(50 * time.Millisecond):
	}
	close(block)
	f.b.Wait()
	resumed(t, f, pending)
	f.b.onStale(lower.State{Phase: lower.PhasePending})
	f.b.mu.Lock()
	opened := f.b.resuming
	f.b.mu.Unlock()
	if opened {
		t.Fatal("onStale opened work after Wait")
	}
}

// 광고 주기의 정리 — 형제가 굽기 잠금을 쥔 채 pending 이면 아무것도 안 한다 · 그 Bake 를 놓으면 (주인이 죽음)
// committed · abandoned pending · 대기 자리를 그 scratch 의 trash 로. 기록된 자리가 없어도 · 못 옮겨도 committed 다.
// committed 를 못 쓰면 retryAt 과 잠금 Release. building (주인이 죽음) 도 같은 표로 치운다 (되물음 1 답 A).
func TestOnStale_CleansAPendingWhoseOwnerDied(t *testing.T) {
	t.Run("a live owner, then a dead one", func(t *testing.T) {
		f := newBakeFixture(t)
		theirs := filepath.Join(filepath.Dir(f.ws), "their-scratch")
		if err := os.Mkdir(theirs, 0o700); err != nil {
			t.Fatal(err)
		}
		pending := f.interrupted(t, lower.PhasePending, theirs, true, true)
		lock := f.holdBakeLock(t)
		f.staleOnce(t)
		if st := f.state(t); st.Phase != lower.PhasePending {
			t.Fatalf("a live bake was cleaned: %+v", st)
		}
		_ = lock.Release() // 주인이 죽었다
		f.staleOnce(t)
		st := f.state(t)
		if st.Phase != lower.PhaseCommitted || st.LastAttempt.Reason != abandonedReason(lower.PhasePending) ||
			st.LastAttempt.Run != "R-dead" || len(st.LastAttempt.Builds) != 1 {
			t.Fatalf("state = %+v %+v", st, st.LastAttempt)
		}
		if got := entries(t, filepath.Join(theirs, "trash")); len(got) != 1 || got[0] != filepath.Base(pending) {
			t.Fatalf("their trash = %v", got)
		}
		f.holdBakeLock(t)
	})
	t.Run("the recorded directory is already gone", func(t *testing.T) {
		f := newBakeFixture(t)
		pending := f.interrupted(t, lower.PhasePending, f.scratch, true, true)
		if err := os.RemoveAll(pending); err != nil {
			t.Fatal(err)
		}
		f.staleOnce(t)
		if st := f.state(t); st.Phase != lower.PhaseCommitted {
			t.Fatalf("state = %+v", st)
		}
	})
	t.Run("it cannot be moved", func(t *testing.T) {
		f := newBakeFixture(t)
		f.interrupted(t, lower.PhasePending, f.scratch, true, true)
		blockTrash(t, f.scratch)
		f.staleOnce(t)
		if st := f.state(t); st.Phase != lower.PhaseCommitted || !f.nodeLogHas("cannot move the pending upper to trash") {
			t.Fatalf("state = %+v log %s", st, f.nodeLog)
		}
	})
	t.Run("committed cannot be written", func(t *testing.T) {
		f := newBakeFixture(t)
		f.interrupted(t, lower.PhasePending, f.scratch, true, true)
		restore := f.failStateWrites(t)
		f.staleOnce(t)
		restore()
		if st := f.state(t); st.Phase != lower.PhasePending {
			t.Fatalf("state = %+v", st)
		}
		if at := f.retryAt(); time.Until(at) < 9*time.Minute {
			t.Fatalf("retryAt = %v", at)
		}
		f.holdBakeLock(t)
	})
	t.Run("a building whose owner died", func(t *testing.T) {
		f := newBakeFixture(t)
		f.interrupted(t, lower.PhaseBuilding, f.scratch, false, true)
		f.staleOnce(t)
		st := f.state(t)
		if st.Phase != lower.PhaseCommitted || st.LastAttempt.Reason != abandonedReason(lower.PhaseBuilding) ||
			len(st.LastAttempt.Builds) != 0 {
			t.Fatalf("state = %+v %+v", st, st.LastAttempt)
		}
	})
}

// 재개는 DropBake · HoldBake 를 안 부른다 — guard 의 표지가 그대로라 합치기 전에 매칭된 늦은 임대가 lower_changed
// 로 거절된다 (결정 1 · 27).
func TestResume_DoesNotDropTheBake(t *testing.T) {
	f := newBakeFixture(t)
	useTestMergeHelper(t, "")
	mark := f.g.mark
	f.interrupted(t, lower.PhaseMerging, f.scratch, true, true)
	f.staleOnce(t)
	if f.g.mark != mark || f.guardBake() != "" {
		t.Fatalf("the resume moved the guard's mark: %+v -> %+v", mark, f.g.mark)
	}
	f.g.AfterResponse("", []Lease{{RunID: "R-late", Node: "node-a"}})
	if err := f.g.OnClaim(stepOf("R-late", "run", "")); err == nil {
		t.Fatal("a late lease ran on the merged lower")
	} else {
		var changed *LowerChangedError
		if !errors.As(err, &changed) {
			t.Fatalf("OnClaim = %v", err)
		}
	}
	f.g.StepDone(nil)
}

// merge 단계의 Apply 가 멈춘 뒤의 onStale 은 곧바로 재개한다 — retryAt 이 없다 (답 2).
func TestResume_AfterAStoppedMergeRunsAtOnce(t *testing.T) {
	f := mergeFixture(t)
	useTestMergeHelper(t, "apply-fails")
	f.w.execute(context.Background(), mergeStepOf("r1"))
	if st := f.state(t); st.Phase != lower.PhaseMerging {
		t.Fatalf("state = %+v", st)
	}
	useTestMergeHelper(t, "")
	f.staleOnce(t)
	if st := f.state(t); st.Phase != lower.PhaseCommitted {
		t.Fatalf("state = %+v log %s", st, f.nodeLog)
	}
	if md, _ := lower.ReadMetadata(f.ws); md == nil || !md.Bake.Resumed || md.Bake.Run != "r1" || md.Bake.Node != "node-a" {
		t.Fatalf("metadata = %+v", md)
	}
}

// 재개의 대기 줄은 노드 로그다 (마감 줄 없음) · 데몬이 멈추면 두 잠금을 놓고 merging 그대로.
func TestResume_WaitsInTheNodeLog(t *testing.T) {
	f := newBakeFixture(t)
	useTestMergeHelper(t, "")
	f.interrupted(t, lower.PhaseMerging, f.scratch, true, true)
	s, _, _ := f.sibling(t).TryShared(lower.Holder{Node: "node-b", Label: "box-b", Role: lower.RoleCandidate})
	t.Cleanup(func() { _ = s.Release() })
	f.b.onStale(lower.State{})
	waitFor(t, func() bool { return f.nodeLogHas("waiting for the lower lock to resume the merge of run R-dead") },
		"no resume wait line")
	if f.nodeLogHas("deadline") || !f.nodeLogHas("node box-b holds it as a candidate") {
		t.Fatalf("node log = %s", f.nodeLog)
	}
	f.stop()
	f.b.wg.Wait()
	if st := f.state(t); st.Phase != lower.PhaseMerging || f.nodeLogHas("resume failed") {
		t.Fatalf("state = %+v log %s", st, f.nodeLog)
	}
	_ = s.Release()
	released(t, f)
}

// 재개를 여는 셋 — 기동의 merging 줄 (배경 · Baker.Wait) · build claim 이 낡은 merging 을 만나면 잡은 잠금을 넘겨
// 재개를 열고 build 는 둘째 문장의 bake_in_progress (되물음 4 답 A) · 광고 주기 (위의 시험들).
func TestResume_TheOpeners(t *testing.T) {
	t.Run("start", func(t *testing.T) {
		f := newBakeFixture(t)
		useTestMergeHelper(t, "")
		mine := f.pendingIn(t, f.scratch, "R-older", false) // 기록된 자리 밖의 버려진 자리
		pending := f.interrupted(t, lower.PhaseMerging, f.scratch, true, true)
		f.b = f.startBaker()
		f.b.wg.Wait()
		resumed(t, f, pending)
		if _, err := os.Stat(mine); !os.IsNotExist(err) {
			t.Fatalf("an abandoned directory stayed: %v", err)
		}
		if !f.nodeLogHas("from=start") {
			t.Fatalf("node log = %s", f.nodeLog)
		}
	})
	t.Run("build claim", func(t *testing.T) {
		f := newBakeFixture(t)
		useTestMergeHelper(t, "")
		pending := f.interrupted(t, lower.PhaseMerging, f.scratch, true, true)
		f.w.execute(context.Background(), buildStepOf("r1"))
		res := f.m.only(t)
		if res.Reason != contract.ReasonBakeInProgress || res.Error != resumeOpenedText("R-dead") {
			t.Fatalf("result = %+v", res)
		}
		f.b.wg.Wait()
		resumed(t, f, pending)
		if !f.nodeLogHas(`from="build claim"`) || len(f.rt.ran()) != 0 {
			t.Fatalf("node log = %s commands %v", f.nodeLog, f.rt.scripts())
		}
	})
}

// BenchmarkOnStaleWhileTheOwnerLives 는 주인이 살아 있는 동안 광고마다의 배경 일 한 번이다 — TryBake 헛시도 (계획 3.2 ·
// 측정 2 의 7.7 ~ 8.4 µs 근처). 기본 go test 에서는 안 돈다.
func BenchmarkOnStaleWhileTheOwnerLives(b *testing.B) {
	f := benchBakeFixture(b)
	root, err := lower.ReadRoot(f.ws)
	if err != nil {
		b.Fatal(err)
	}
	sib, err := lower.Open(f.lowers, root, time.Now())
	if err != nil {
		b.Fatal(err)
	}
	lock, ok, err := sib.TryBake()
	if err != nil || !ok {
		b.Fatalf("TryBake = %v %v", ok, err)
	}
	defer func() { _ = lock.Release() }()
	st := lower.State{Phase: lower.PhasePending}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.b.onStale(st)
		f.b.wg.Wait()
	}
}

// BenchmarkStaleCleanup 은 주인이 죽은 pending 의 정리 한 번이다 — 옮기기 + committed (1.5 ms 근처).
func BenchmarkStaleCleanup(b *testing.B) {
	f := benchBakeFixture(b)
	dir := f.g.Dir()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		pending, err := makePending(f.scratch, dir.Root.Key.String())
		if err != nil {
			b.Fatal(err)
		}
		lock, _, _ := dir.TryBake()
		st := lower.State{Phase: lower.PhasePending, Owner: &lower.Owner{Run: "R"}, PendingUpper: filepath.Join(pending, "upper"),
			Since: time.Now()}
		if err := lock.WriteState(st); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		if _, err := f.b.clean(dir, lock, st, fromAdvert); err != nil {
			b.Fatal(err)
		}
		_ = lock.Release()
	}
}

// benchBakeFixture 는 벤치마크의 굽는 노드다 — 가짜 Mediator 없이 lower 자리 · Baker 만.
func benchBakeFixture(b *testing.B) *bakeFixture {
	b.Helper()
	dir := b.TempDir()
	f := &bakeFixture{guardFixture: guardFixture{ws: filepath.Join(dir, "ws"), lowers: filepath.Join(dir, "lowers"),
		log: &lockedBuffer{}}, nodeLog: &lockedBuffer{}}
	for _, d := range []string{f.ws, filepath.Join(dir, "scratch")} {
		if err := os.Mkdir(d, 0o700); err != nil {
			b.Fatal(err)
		}
	}
	f.scratch = filepath.Join(dir, "scratch")
	f.log = slogTo(f.nodeLog)
	f.g = newLowerGuard(f.lowers, f.ws, Identity{NodeID: "node-a"}, f.log)
	f.g.start()
	f.b = StartBaker(context.Background(), f.g, f.scratch, Identity{NodeID: "node-a"}, "i", f.log)
	b.Cleanup(f.b.Wait)
	return f
}

// 재개의 그물 — 찾으면 배타를 놓고 foreignRetry 뒤 다시 기다린다 (찾은 것은 실패가 아니다) · 오류면 경고 뒤 배타만으로
// 합친다 (FD 규칙 8.4 · 12.3 의 2).
func TestResume_TheNet(t *testing.T) {
	swap := func(t *testing.T, f func(lower.Root) (lower.MountScan, error)) {
		was, retry := foreignMounts, foreignRetry
		foreignMounts, foreignRetry = f, 10*time.Millisecond
		t.Cleanup(func() { foreignMounts, foreignRetry = was, retry })
	}
	t.Run("found, then clear", func(t *testing.T) {
		f := newBakeFixture(t)
		useTestMergeHelper(t, "")
		pending := f.interrupted(t, lower.PhaseMerging, f.scratch, true, true)
		calls := 0
		swap(t, func(lower.Root) (lower.MountScan, error) {
			calls++
			if calls == 1 {
				return lower.MountScan{Found: []lower.Mount{{PID: 7, Namespace: "mnt:[1]", MountPoint: "/m", LowerDir: "/l"}}}, nil
			}
			return lower.MountScan{}, nil
		})
		f.staleOnce(t)
		resumed(t, f, pending)
		if calls != 2 || !f.nodeLogHas("found 1 overlay mount of this lower in another mount namespace") {
			t.Fatalf("scans %d, node log %s", calls, f.nodeLog)
		}
	})
	t.Run("an error merges on the lock", func(t *testing.T) {
		f := newBakeFixture(t)
		useTestMergeHelper(t, "")
		pending := f.interrupted(t, lower.PhaseMerging, f.scratch, true, true)
		swap(t, func(lower.Root) (lower.MountScan, error) { return lower.MountScan{}, errors.New("no mountinfo") })
		f.staleOnce(t)
		resumed(t, f, pending)
		if !f.nodeLogHas("cannot scan mounts of other processes; merging on the lower lock alone") {
			t.Fatalf("node log = %s", f.nodeLog)
		}
	})
	t.Run("the daemon stops while the net waits", func(t *testing.T) {
		f := newBakeFixture(t)
		f.interrupted(t, lower.PhaseMerging, f.scratch, true, true)
		swap(t, func(lower.Root) (lower.MountScan, error) {
			f.stop()
			return lower.MountScan{Found: []lower.Mount{{PID: 7}}}, nil
		})
		foreignRetry = time.Hour
		f.staleOnce(t)
		if st := f.state(t); st.Phase != lower.PhaseMerging || f.nodeLogHas("resume failed") {
			t.Fatalf("state = %+v log %s", st, f.nodeLog)
		}
		released(t, f)
	})
}

// 광고 주기의 배경 일이 state.json 을 못 읽으면 잠금을 놓고 로그 한 줄이다 — 간격은 걸지 않는다. 자리를 못 연 노드는
// 아무것도 안 한다.
func TestOnStale_UnreadableStateAndNoDirectory(t *testing.T) {
	f := newBakeFixture(t)
	if err := os.WriteFile(filepath.Join(f.g.Dir().Path, "state.json"), []byte(`{"schema":1,"phase":"odd"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	f.b.onStale(lower.State{})
	f.b.wg.Wait()
	if !f.nodeLogHas("bake: cannot read the lower state") || !f.retryAt().IsZero() {
		t.Fatalf("node log = %s", f.nodeLog)
	}
	f.holdBakeLock(t)

	dir := t.TempDir()
	g := newLowerGuard(filepath.Join(dir, "lowers"), filepath.Join(dir, "missing"), Identity{NodeID: "n"}, f.log)
	g.start()
	b := StartBaker(context.Background(), g, dir, Identity{NodeID: "n"}, "i", f.log)
	b.onStale(lower.State{Phase: lower.PhasePending})
	b.Wait()
}
