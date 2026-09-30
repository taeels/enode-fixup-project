//go:build linux

package enode

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/lower"
	"golang.org/x/sys/unix"
)

// merge 단계 (bake 유닛 · FD 규칙 7 · 7.1 · 8 · 10 · 11절 · FD 흐름 3 · 4 · 7절). 진짜 lower 자리 · namespace 없는
// merge-helper · 가짜 Mediator. 대기 자리는 가짜 런타임의 build 단계가 만든다.

// mergeFixture 는 build 가 pending 을 쓴 굽는 노드다. lower 에 a.txt · d/old 가 있고, upper 는 a.txt 를 바꾸고 new.txt
// 를 더하고 d 를 파일로 바꾼다 (lower 의 d 가 trash 로 간다).
func mergeFixture(t *testing.T) *bakeFixture {
	t.Helper()
	return mergeFixtureAt(t, false)
}

// mergeFixtureAt 의 linked 는 newBakeFixtureAt 과 같다 — 노드의 워크스페이스가 lower 를 가리키는 symlink 다.
func mergeFixtureAt(t *testing.T, linked bool) *bakeFixture {
	t.Helper()
	f := newBakeFixtureAt(t, linked)
	useTestMergeHelper(t, "")
	write(t, f.ws, "a.txt", "old a")
	write(t, f.ws, "d/old", "old")
	f.rt.answer(buildA, fakeAnswer{do: func(upper string) {
		write(t, upper, "a.txt", "new a")
		write(t, upper, "new.txt", "new")
		write(t, upper, "d", "now a file")
	}})
	f.buildOK(t, "r1")
	return f
}

// released 는 두 잠금이 풀렸는지 본다 — 형제가 굽기 잠금과 lower 배타를 곧바로 잡는다.
func released(t *testing.T, f *bakeFixture) {
	t.Helper()
	f.holdBakeLock(t)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	ex, err := f.sibling(t).Exclusive(ctx, time.Hour, nil)
	if err != nil {
		t.Fatalf("the lower lock is still held: %v", err)
	}
	_ = ex.Release()
}

func TestMergeStep_Merges(t *testing.T) {
	f := mergeFixture(t)
	pending := filepath.Dir(f.state(t).PendingUpper)
	markBefore := f.g.mark
	f.w.execute(context.Background(), mergeStepOf("r1"))
	res := f.m.lastResult(t)
	if res.Error != "" || res.Reason != "" || !reflect.DeepEqual(res.Produced, []string{contract.ArtifactMerged}) ||
		res.Upload != contract.StageOK {
		t.Fatalf("result = %+v", res)
	}
	if res.ExitCode != nil || res.ExitedAt != nil || res.FinalizedAt != nil || res.Finalize != "" || res.Build != nil {
		t.Fatalf("a merge step reported command fields: %+v", res)
	}
	if n := len(f.m.exits()); n != 1 {
		t.Fatalf("the merge step sent an exit report (%d in all)", n)
	}
	for name, want := range map[string]string{"a.txt": "new a", "new.txt": "new", "d": "now a file"} {
		if b, _ := os.ReadFile(filepath.Join(f.ws, name)); string(b) != want {
			t.Fatalf("lower %s = %q", name, b)
		}
	}
	trash := filepath.Join(f.scratch, "trash")
	if m := mode(t, trash); m != 0o700 {
		t.Fatalf("trash is %o", m)
	}
	if got := entries(t, trash); !reflect.DeepEqual(got, []string{filepath.Base(pending), "d"}) {
		t.Fatalf("trash = %v", got)
	}
	md, err := lower.ReadMetadata(f.ws)
	if err != nil || md == nil {
		t.Fatalf("metadata = %v %v", md, err)
	}
	if md.Bake.Run != "r1" || md.Bake.Node != "node-a" || md.Bake.Resumed || md.Bake.MergedAt.IsZero() ||
		*md.Source.IR != bakeIR || md.Source.Head != headA || md.Source.URL != "https://u@git.example/x.git" ||
		md.Source.SyncCommand != syncCmd || len(md.Builds) != 2 || md.Environment != "prep-1" ||
		md.WorkspaceTarget != "/work" || md.Bake.PreviousIR != nil {
		t.Fatalf("metadata = %+v bake %+v", md, md.Bake)
	}
	if st := f.state(t); st.Phase != lower.PhaseCommitted || st.LastAttempt != nil || st.Owner != nil {
		t.Fatalf("state = %+v", st)
	}
	if f.held() != nil || f.guardBake() != "" || f.g.mark == markBefore || f.g.mark.Run != "r1" {
		t.Fatalf("held %v guard %q mark %+v", f.held(), f.guardBake(), f.g.mark)
	}
	released(t, f)
	var got contract.MergeResult
	body, _ := f.m.blob(contract.ArtifactMerged)
	if err := json.Unmarshal(body, &got); err != nil || !reflect.DeepEqual(&got, res.Merge) {
		t.Fatalf("merged = %s %v; result %+v", body, err, res.Merge)
	}
	if res.Merge.Resumed || *res.Merge.IR != bakeIR || res.Merge.Ops.Created != 1 || res.Merge.Ops.Replaced != 2 ||
		res.Merge.Ops.Trashed != 1 {
		t.Fatalf("merge = %+v", res.Merge)
	}
	if log := f.m.logOf("merge"); !strings.Contains(log, "bake: merged: ops ") || strings.Contains(log, f.scratch) {
		t.Fatalf("merge step log = %s", log)
	}
}

// 워크스페이스가 lower 를 가리키는 symlink 인 노드도 합치고 metadata 를 쓰고 committed 로 끝난다. lower 파일 일은
// 상태 자리를 연 때 symlink 를 푼 lower 뿌리 (lower.Dir 의 Root.Path) 로 한다 — lower.WriteMetadata 는 O_NOFOLLOW 로
// 뿌리를 열어 fsync 하므로 설정의 symlink 글자로는 합친 뒤에 ENOTDIR 로 멈췄다 (조각 7 에서 찾았다).
func TestMergeStep_ALinkedWorkspaceMerges(t *testing.T) {
	f := mergeFixtureAt(t, true)
	if fi, err := os.Lstat(f.nodeWS); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("the node workspace is not a symlink: %v %v", fi, err)
	}
	f.w.execute(context.Background(), mergeStepOf("r1"))
	res := f.m.lastResult(t)
	if res.Error != "" || res.Reason != "" || res.Merge == nil || !reflect.DeepEqual(res.Produced, []string{contract.ArtifactMerged}) {
		t.Fatalf("result = %+v", res)
	}
	if b, _ := os.ReadFile(filepath.Join(f.ws, "new.txt")); string(b) != "new" {
		t.Fatalf("lower new.txt = %q", b)
	}
	md, err := lower.ReadMetadata(f.ws)
	if err != nil || md == nil || md.Bake.Run != "r1" || md.Bake.Resumed {
		t.Fatalf("metadata = %+v %v", md, err)
	}
	if fi, err := os.Lstat(f.nodeWS); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("the workspace symlink was replaced: %v %v", fi, err)
	}
	if st := f.state(t); st.Phase != lower.PhaseCommitted || st.LastAttempt != nil {
		t.Fatalf("state = %+v", st)
	}
	if f.held() != nil || f.guardBake() != "" || f.g.mark.Run != "r1" {
		t.Fatalf("held %v guard %q mark %+v", f.held(), f.guardBake(), f.g.mark)
	}
	released(t, f)
}

// 형제가 공유를 쥔 동안 기다린다 — 단계 로그와 진행 청크에 쥔 쪽 줄이 있고, 형제가 놓으면 합친다.
func TestMergeStep_WaitsForTheSibling(t *testing.T) {
	f := mergeFixture(t)
	since := time.Date(2026, 9, 27, 4, 10, 0, 0, time.UTC)
	s, ok, err := f.sibling(t).TryShared(lower.Holder{Node: "node-b", Label: "box-b", Role: lower.RoleRun,
		Run: "R-sib", Since: since})
	if err != nil || !ok {
		t.Fatalf("TryShared = %v %v", ok, err)
	}
	t.Cleanup(func() { _ = s.Release() })
	holder := "  node box-b holds it for run R-sib since 2026-09-27T04:10:00Z"
	go func() {
		waitFor(t, func() bool { return strings.Contains(f.m.progressText(), holder) }, "no holder line in progress")
		_ = s.Release()
	}()
	f.w.execute(context.Background(), mergeStepOf("r1"))
	res := f.m.lastResult(t)
	if res.Error != "" || res.Merge == nil {
		t.Fatalf("result = %+v", res)
	}
	log := f.m.logOf("merge")
	if !strings.Contains(log, "waiting for the lower lock; deadline ") || !strings.Contains(log, holder+"\n") ||
		!strings.Contains(log, "took the lower lock after ") {
		t.Fatalf("merge step log = %s", log)
	}
	if !f.nodeLogHas(`event=took`) || !f.nodeLogHas(`event=waiting`) {
		t.Fatalf("node log = %s", f.nodeLog)
	}
}

// claim 확인 (FD 규칙 7.1)

func TestMergeStep_ClaimCheck(t *testing.T) {
	t.Run("held but the state is not this bake's", func(t *testing.T) {
		f := mergeFixture(t)
		if err := f.held().lock.WriteState(lower.State{Phase: lower.PhaseCommitted, Since: time.Now()}); err != nil {
			t.Fatal(err)
		}
		f.w.execute(context.Background(), mergeStepOf("r1"))
		res := f.m.lastResult(t)
		if res.Error != "the lower state does not match this bake: committed (run unknown)" {
			t.Fatalf("result = %+v", res)
		}
		if st := f.state(t); st.LastAttempt != nil {
			t.Fatalf("the mismatch wrote a last_attempt: %+v", st)
		}
		if f.held() != nil || f.guardBake() != "" {
			t.Fatal("the bake is still held")
		}
		f.holdBakeLock(t)
	})
	t.Run("a restart left the pending for the cleanup", func(t *testing.T) {
		f := mergeFixture(t)
		// 새 Baker 는 굽기 잠금을 못 잡아 정리가 못 돌았다 — 옛 잠금은 옛 프로세스의 것이다
		f.w.Bake = f.startBaker()
		f.w.execute(context.Background(), mergeStepOf("r1"))
		if res := f.m.lastResult(t); res.Error != restartLeftText {
			t.Fatalf("result = %+v", res)
		}
	})
	t.Run("a restart discarded the pending", func(t *testing.T) {
		f := mergeFixture(t)
		restart(t, f)
		if st := f.state(t); st.Phase != lower.PhaseCommitted || st.LastAttempt.Reason != abandonedReason(lower.PhasePending) {
			t.Fatalf("the start-up cleanup did not run: %+v", st)
		}
		// 다른 굽기가 building 을 써도 last_attempt 는 남는다 (결정 49)
		st := f.state(t)
		f.putState(t, lower.State{Phase: lower.PhaseBuilding, Owner: &lower.Owner{Run: "R-other"},
			PendingUpper: "/x", LastAttempt: st.LastAttempt})
		f.w.execute(context.Background(), mergeStepOf("r1"))
		if res := f.m.lastResult(t); res.Error != restartDiscardedText {
			t.Fatalf("result = %+v", res)
		}
	})
	nothing := func(t *testing.T, f *bakeFixture) {
		t.Helper()
		f.w.execute(context.Background(), mergeStepOf("r1"))
		res := f.m.lastResult(t)
		if res.Error != "" || res.Merge != nil || res.Produced != nil {
			t.Fatalf("result = %+v", res)
		}
		if !strings.Contains(f.m.logOf("merge"), nothingToMergeLine+"\n") {
			t.Fatalf("merge step log = %s", f.m.logOf("merge"))
		}
	}
	t.Run("a failed command leaves nothing to merge", func(t *testing.T) {
		f := newBakeFixture(t)
		f.rt.answer(syncCmd, fakeAnswer{exit: 1})
		f.w.execute(context.Background(), buildStepOf("r1"))
		nothing(t, f)
	})
	t.Run("an ir mismatch leaves nothing to merge", func(t *testing.T) {
		f := newBakeFixture(t)
		f.rt.answer(probeScript, fakeAnswer{stdout: "mode=git\nhead=" + headA + "\ntagged=\n"})
		f.w.execute(context.Background(), buildStepOf("r1"))
		nothing(t, f)
	})
	t.Run("another bake's pending", func(t *testing.T) {
		f := newBakeFixture(t)
		f.putState(t, lower.State{Phase: lower.PhasePending, Owner: &lower.Owner{Run: "R-other"}, PendingUpper: "/x"})
		nothing(t, f)
	})
	t.Run("a building of this run whose committed was not written", func(t *testing.T) {
		f := newBakeFixture(t)
		f.putState(t, lower.State{Phase: lower.PhaseBuilding, Owner: &lower.Owner{Run: "r1"}, PendingUpper: "/x"})
		nothing(t, f)
	})
	t.Run("a building that was cleaned", func(t *testing.T) {
		f := newBakeFixture(t)
		f.putState(t, lower.State{Phase: lower.PhaseCommitted,
			LastAttempt: &lower.LastAttempt{Run: "r1", Reason: abandonedReason(lower.PhaseBuilding)}})
		nothing(t, f)
	})
}

// restart 는 build 보고 뒤의 재시작이다 — 옛 굽기 잠금을 놓고 (프로세스가 죽었다) 새 LowerGuard 와 Baker 로 짓는다.
// 기동 정리가 pending 을 치운다.
func restart(t *testing.T, f *bakeFixture) {
	t.Helper()
	if h := f.held(); h != nil {
		_ = h.lock.Release()
	}
	f.g = newLowerGuard(f.lowers, f.ws, Identity{NodeID: "node-a", Label: "box-a"}, f.log)
	f.g.start()
	f.b = f.startBaker()
	f.w.Guard, f.w.Bake = f.g, f.b
}

// 그물 (FD 규칙 8.4)

func TestMergeStep_TheNet(t *testing.T) {
	swap := func(t *testing.T, f func(lower.Root) (lower.MountScan, error)) {
		was, retry := foreignMounts, foreignRetry
		foreignMounts, foreignRetry = f, 10*time.Millisecond
		t.Cleanup(func() { foreignMounts, foreignRetry = was, retry })
	}
	t.Run("found releases, waits and tries again", func(t *testing.T) {
		f := mergeFixture(t)
		var mu sync.Mutex
		calls := 0
		swap(t, func(lower.Root) (lower.MountScan, error) {
			mu.Lock()
			defer mu.Unlock()
			calls++
			if calls == 1 {
				return lower.MountScan{Found: []lower.Mount{{PID: 41822, Namespace: "mnt:[4026532871]",
					MountPoint: "/s/enode-runc-x/merged", LowerDir: "/s/enode-runc-x/lower-ro"}}, Unreadable: 2}, nil
			}
			return lower.MountScan{}, nil
		})
		f.w.execute(context.Background(), mergeStepOf("r1"))
		if res := f.m.lastResult(t); res.Error != "" || res.Merge == nil || calls != 2 {
			t.Fatalf("result = %+v after %d scans", res, calls)
		}
		log := f.m.logOf("merge")
		for _, line := range []string{"2 processes could not be read while scanning mounts\n",
			"found 1 overlay mount of this lower in another mount namespace; releasing the lock and waiting 0.01s\n",
			"  pid 41822 namespace mnt:[4026532871] mount /s/enode-runc-x/merged lowerdir /s/enode-runc-x/lower-ro\n"} {
			if !strings.Contains(log, line) {
				t.Fatalf("merge step log misses %q:\n%s", line, log)
			}
		}
	})
	t.Run("an error warns and merges on the lock", func(t *testing.T) {
		f := mergeFixture(t)
		swap(t, func(lower.Root) (lower.MountScan, error) { return lower.MountScan{}, errors.New("no mountinfo") })
		f.w.execute(context.Background(), mergeStepOf("r1"))
		if res := f.m.lastResult(t); res.Error != "" || res.Merge == nil {
			t.Fatalf("result = %+v", res)
		}
		if !strings.Contains(f.m.logOf("merge"), "cannot scan mounts of other processes: no mountinfo; merging on the "+
			"lower lock alone\n") {
			t.Fatalf("merge step log = %s", f.m.logOf("merge"))
		}
	})
}

// 시작 전 확인 · held 동안의 오류 · merging 뒤 (FD 규칙 10 · 11 · 7절)

func TestMergeStep_Failures(t *testing.T) {
	t.Run("preflight mismatch", func(t *testing.T) {
		f := newBakeFixture(t)
		useTestMergeHelper(t, "")
		f.rt.answer(buildA, fakeAnswer{do: func(upper string) {
			write(t, upper, "marked", "x")
			if err := unix.Lsetxattr(filepath.Join(upper, "marked"), "user.overlay.metacopy", nil, 0); err != nil {
				t.Error(err)
			}
		}})
		f.buildOK(t, "r1")
		f.w.execute(context.Background(), mergeStepOf("r1"))
		res := f.m.lastResult(t)
		if !strings.HasPrefix(res.Error, "merge preflight: upper entry marked carries user.overlay.metacopy") {
			t.Fatalf("result = %+v", res)
		}
		st := endedCommitted(t, f, "merge preflight: ")
		if _, err := os.Stat(filepath.Join(f.ws, "marked")); !os.IsNotExist(err) || st.LastAttempt.Builds == nil {
			t.Fatalf("the lower changed or the builds were lost: %v %+v", err, st.LastAttempt)
		}
		released(t, f)
	})
	t.Run("the body moved the pending directory first", func(t *testing.T) {
		f := mergeFixture(t)
		h := f.held()
		was := foreignMounts
		foreignMounts = func(lower.Root) (lower.MountScan, error) {
			h.abandon(reasonBakeRunEnded, h.draftBuilds())
			return lower.MountScan{}, nil
		}
		t.Cleanup(func() { foreignMounts = was })
		f.w.execute(context.Background(), mergeStepOf("r1"))
		res := f.m.lastResult(t)
		if !strings.HasPrefix(res.Error, "merge preflight: ") || !strings.Contains(res.Error, "no such file or directory") {
			t.Fatalf("result = %+v", res)
		}
		if st := f.state(t); st.LastAttempt.Reason != reasonBakeRunEnded {
			t.Fatalf("a second body ran: %+v", st.LastAttempt)
		}
		released(t, f)
	})
	t.Run("state.json cannot be read", func(t *testing.T) {
		f := mergeFixture(t)
		if err := os.WriteFile(filepath.Join(f.g.Dir().Path, "state.json"), []byte(`{"schema":1,"phase":"odd"}`), 0o600); err != nil {
			t.Fatal(err)
		}
		f.w.execute(context.Background(), mergeStepOf("r1"))
		if res := f.m.lastResult(t); !strings.HasPrefix(res.Error, "cannot open the lower state directory: ") {
			t.Fatalf("result = %+v", res)
		}
		endedCommitted(t, f, "cannot open the lower state directory: ")
		assertNextBakeRuns(t, f)
	})
	t.Run("the trash cannot be made", func(t *testing.T) {
		f := mergeFixture(t)
		blockTrash(t, f.scratch)
		f.w.execute(context.Background(), mergeStepOf("r1"))
		if res := f.m.lastResult(t); !strings.HasPrefix(res.Error, "cannot create the trash of the pending upper: ") {
			t.Fatalf("result = %+v", res)
		}
		st := f.state(t)
		if st.Phase != lower.PhaseCommitted || f.held() != nil || f.guardBake() != "" {
			t.Fatalf("state = %+v", st)
		}
		released(t, f)
	})
	t.Run("merging cannot be written", func(t *testing.T) {
		f := mergeFixture(t)
		was := writeLowerState
		writeLowerState = func(b *lower.Bake, st lower.State) error {
			if st.Phase == lower.PhaseMerging {
				return errors.New("injected: read-only file system")
			}
			return was(b, st)
		}
		t.Cleanup(func() { writeLowerState = was })
		f.w.execute(context.Background(), mergeStepOf("r1"))
		if res := f.m.lastResult(t); res.Error != "cannot write the lower state: injected: read-only file system" {
			t.Fatalf("result = %+v", res)
		}
		endedCommitted(t, f, "cannot write the lower state: ")
		released(t, f)
	})
	t.Run("apply stops and the lower stays merging", func(t *testing.T) {
		f := mergeFixture(t)
		useTestMergeHelper(t, "apply-fails")
		f.w.execute(context.Background(), mergeStepOf("r1"))
		res := f.m.lastResult(t)
		if res.Error != "merge stopped: merge: rename a.txt: permission denied; the lower stays merging and a node on "+
			"this lower resumes it" || res.Reason != "" {
			t.Fatalf("result = %+v", res)
		}
		if st := f.state(t); st.Phase != lower.PhaseMerging || st.Owner.Run != "r1" {
			t.Fatalf("state = %+v", st)
		}
		if f.held() != nil || f.guardBake() != "" {
			t.Fatal("the bake is still held")
		}
		f.b.mu.Lock()
		retry := f.b.retryAt
		f.b.mu.Unlock()
		if !retry.IsZero() {
			t.Fatalf("a stopped merge set retryAt %v", retry)
		}
		released(t, f)
	})
	t.Run("committed cannot be written after the metadata", func(t *testing.T) {
		f := mergeFixture(t)
		pending := filepath.Dir(f.state(t).PendingUpper)
		was := writeLowerState
		writeLowerState = func(b *lower.Bake, st lower.State) error {
			if st.Phase == lower.PhaseCommitted {
				return errors.New("injected: disk quota exceeded")
			}
			return was(b, st)
		}
		t.Cleanup(func() { writeLowerState = was })
		f.w.execute(context.Background(), mergeStepOf("r1"))
		writeLowerState = was
		res := f.m.lastResult(t)
		if res.Error != "" || res.Merge == nil {
			t.Fatalf("result = %+v", res)
		}
		if st := f.state(t); st.Phase != lower.PhaseMerging {
			t.Fatalf("state = %+v", st)
		}
		if _, err := os.Stat(filepath.Join(pending, draftName)); err != nil {
			t.Fatalf("the draft went: %v", err)
		}
		if !strings.Contains(f.m.logOf("merge"), "the merge finished but cleaning up failed: injected: disk quota "+
			"exceeded; the next resume writes committed\n") {
			t.Fatalf("merge step log = %s", f.m.logOf("merge"))
		}
		released(t, f)
	})
	t.Run("the pending directory cannot move after committed", func(t *testing.T) {
		f := mergeFixture(t)
		pending := filepath.Dir(f.state(t).PendingUpper)
		was := movePending
		movePending = func(string) error { return errors.New("injected: cross-device link") }
		t.Cleanup(func() { movePending = was })
		f.w.execute(context.Background(), mergeStepOf("r1"))
		movePending = was
		if res := f.m.lastResult(t); res.Error != "" || res.Merge == nil {
			t.Fatalf("result = %+v", res)
		}
		if st := f.state(t); st.Phase != lower.PhaseCommitted {
			t.Fatalf("state = %+v", st)
		}
		if _, err := os.Stat(pending); err != nil || !f.nodeLogHas("bake: cannot move the pending upper to trash") {
			t.Fatalf("pending %v node log %s", err, f.nodeLog)
		}
	})
	t.Run("the upload budget runs out after the merge", func(t *testing.T) {
		f := mergeFixture(t)
		f.m.putHold[contract.ArtifactMerged] = true
		f.w.budgets = func(*Step) (time.Duration, time.Duration) { return time.Minute, 200 * time.Millisecond }
		f.w.execute(context.Background(), mergeStepOf("r1"))
		res := f.m.lastResult(t)
		if res.Error != "upload budget of 200ms exceeded" || res.Reason != contract.ReasonUploadTimeout ||
			res.Upload != contract.StageTimeout || res.Merge == nil {
			t.Fatalf("result = %+v", res)
		}
		md, _ := lower.ReadMetadata(f.ws)
		if st := f.state(t); st.Phase != lower.PhaseCommitted || md == nil || md.Bake.Run != "r1" {
			t.Fatalf("state = %+v metadata %+v", st, md)
		}
		released(t, f)
		if !f.nodeLogHas("upload budget exceeded; not uploaded") {
			t.Fatalf("node log = %s", f.nodeLog)
		}
	})
	t.Run("the lease disappears while waiting", func(t *testing.T) {
		f := mergeFixture(t)
		s, _, _ := f.sibling(t).TryShared(lower.Holder{Node: "node-b", Role: lower.RoleCandidate})
		t.Cleanup(func() { _ = s.Release() })
		go func() {
			waitFor(t, func() bool { return strings.Contains(f.m.progressText(), "waiting for the lower lock") },
				"the merge did not wait")
			f.w.Held.Set(nil)
		}()
		f.w.execute(context.Background(), mergeStepOf("r1"))
		if res := f.m.lastResult(t); res.Error != "aborted: lease expired" {
			t.Fatalf("result = %+v", res)
		}
		endedCommitted(t, f, "aborted: lease expired")
	})
	t.Run("the node stops while waiting", func(t *testing.T) {
		f := mergeFixture(t)
		s, _, _ := f.sibling(t).TryShared(lower.Holder{Node: "node-b", Role: lower.RoleCandidate})
		t.Cleanup(func() { _ = s.Release() })
		ctx, stop := context.WithCancel(context.Background())
		go func() {
			waitFor(t, func() bool { return strings.Contains(f.m.progressText(), "waiting for the lower lock") },
				"the merge did not wait")
			stop()
		}()
		f.w.execute(ctx, mergeStepOf("r1"))
		if n := f.m.reports(); n != 1 {
			t.Fatalf("a stopped node reported: %d reports", n)
		}
		endedCommitted(t, f, "node stopped")
	})
	t.Run("the body ran before startMerging", func(t *testing.T) {
		f := mergeFixture(t)
		h := f.held()
		beforeStartMerging = func() { h.abandon(reasonBakeRunEnded, nil) }
		t.Cleanup(func() { beforeStartMerging = nil })
		f.w.execute(context.Background(), mergeStepOf("r1"))
		if st := f.state(t); st.Phase != lower.PhaseCommitted || st.LastAttempt.Reason != reasonBakeRunEnded {
			t.Fatalf("merging was written after the body: %+v", st)
		}
		if _, err := os.Stat(filepath.Join(f.ws, "new.txt")); !os.IsNotExist(err) {
			t.Fatalf("the merge ran after the body: %v", err)
		}
		released(t, f)
	})
}

// 단계 로그와 진행 청크에 대기 자리 경로가 없다 — 그물 줄만 마운트 자리를 보인다 (FD 규칙 16.2 끝).
func TestMergeStep_TheStepLogCarriesNoHostPath(t *testing.T) {
	f := mergeFixture(t)
	s, _, _ := f.sibling(t).TryShared(lower.Holder{Node: "node-b", Role: lower.RoleCandidate})
	go func() {
		waitFor(t, func() bool { return strings.Contains(f.m.progressText(), "holds it as a candidate") }, "no wait")
		_ = s.Release()
	}()
	f.w.execute(context.Background(), mergeStepOf("r1"))
	if log, progress := f.m.logOf("merge"), f.m.progressText(); strings.Contains(log, f.scratch) ||
		strings.Contains(progress, f.scratch) || !strings.Contains(log, "took the lower lock") {
		t.Fatalf("a host path reached the merge step log:\n%s\n---\n%s", log, progress)
	}
}

// recordingHandler 는 slog 의 레코드를 그대로 모은다 — 시각을 자르지 않고 본다.
type recordingHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *recordingHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *recordingHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r.Clone())
	return nil
}
func (h *recordingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *recordingHandler) WithGroup(string) slog.Handler      { return h }

// gaveUpAt 은 노드 로그의 마감 줄의 시각이다.
func (h *recordingHandler) gaveUpAt() (time.Time, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, r := range h.records {
		gaveUp := false
		r.Attrs(func(a slog.Attr) bool {
			gaveUp = gaveUp || (a.Key == "event" && a.Value.String() == "gave up")
			return true
		})
		if r.Message == "waiting for the lower lock (merge step)" && gaveUp {
			return r.Time, true
		}
	}
	return time.Time{}, false
}

// 조각 5 의 (2) — 형제가 공유를 쥔 채 merge.wait 300ms · 900ms 인 merge 단계 둘이 각각 claim 시각 + wait 에서
// merge_wait_timeout 으로 끝난다. 포기한 순간 (노드 로그의 마감 줄) 이 시작 시각 + wait 이상이고 + wait + 300 ms
// 미만이다 (계획 5절 규칙 5 · 결정 37). 그 뒤 upper 가 trash · state 가 committed · 기다리는 동안 쥔 쪽 줄이 있다.
func TestMergeStep_WaitsForTheContractValue(t *testing.T) {
	for _, wait := range []time.Duration{300 * time.Millisecond, 900 * time.Millisecond} {
		t.Run(wait.String(), func(t *testing.T) {
			f := mergeFixture(t)
			rec := &recordingHandler{}
			f.w.Log = slog.New(rec)
			s, _, _ := f.sibling(t).TryShared(lower.Holder{Node: "node-b", Label: "box-b", Role: lower.RoleRun, Run: "R-sib",
				Since: time.Now()})
			t.Cleanup(func() { _ = s.Release() })
			step := mergeStepOf("r1")
			step.Merge = &contract.Merge{Wait: wait.String()}
			start := time.Now()
			f.w.execute(context.Background(), step)
			res := f.m.lastResult(t)
			if res.Reason != contract.ReasonMergeWaitTimeout || res.Error != mergeWaitText(wait) {
				t.Fatalf("result = %+v", res)
			}
			at, ok := rec.gaveUpAt()
			if !ok {
				t.Fatal("no deadline line in the node log")
			}
			if lo, hi := start.Add(wait), start.Add(wait+300*time.Millisecond); at.Before(lo) || !at.Before(hi) {
				t.Fatalf("gave up at %s after the start, want [%s, %s)", at.Sub(start), wait, wait+300*time.Millisecond)
			}
			endedCommitted(t, f, contract.ReasonMergeWaitTimeout)
			if !strings.Contains(f.m.logOf("merge"), "  node box-b holds it for run R-sib since ") {
				t.Fatalf("merge step log = %s", f.m.logOf("merge"))
			}
		})
	}
}

// BenchmarkMergeStepFixedCost 는 배타를 쥔 뒤 합치기가 드는 고정 비용이다 — 빈 upper · namespace 없는 helper · 배타가
// 비어 있다. 그물 · helper 두 번 (preflight · apply) · state 쓰기 둘 (merging · committed) · metadata · 대기 자리 옮기기
// (계획 3.2). 업로드와 보고는 뺀다. 기본 go test 에서는 안 돈다.
func BenchmarkMergeStepFixedCost(b *testing.B) {
	useTestMergeHelper(b, "")
	f := benchBakeFixture(b)
	dir := f.g.Dir()
	key := dir.Root.Key.String()
	d := fullDraft()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		pending, err := makePending(f.scratch, key)
		if err != nil {
			b.Fatal(err)
		}
		upper := filepath.Join(pending, "upper")
		if err := os.Mkdir(upper, 0o700); err != nil {
			b.Fatal(err)
		}
		lock, ok, err := dir.TryBake()
		if err != nil || !ok {
			b.Fatalf("TryBake = %v %v", ok, err)
		}
		trash := filepath.Join(f.scratch, "trash")
		b.StartTimer()

		ex, err := dir.Exclusive(context.Background(), time.Hour, nil)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := foreignMounts(dir.Root); err != nil {
			b.Fatal(err)
		}
		if err := os.MkdirAll(trash, 0o700); err != nil {
			b.Fatal(err)
		}
		req := mergeHelperRequest{Op: "preflight", Upper: upper, Lower: f.ws, Trash: trash}
		if _, err := callMergeHelper(req); err != nil {
			b.Fatal(err)
		}
		if err := lock.WriteState(lower.State{Phase: lower.PhaseMerging, Owner: &lower.Owner{Run: "R"},
			PendingUpper: upper, Since: time.Now()}); err != nil {
			b.Fatal(err)
		}
		req.Op = "apply"
		if _, err := callMergeHelper(req); err != nil {
			b.Fatal(err)
		}
		if err := os.Remove(upper); err != nil {
			b.Fatal(err)
		}
		if err := lower.WriteMetadata(f.ws, metadataOf(d, "R", "node-a", time.Now(), false)); err != nil {
			b.Fatal(err)
		}
		if err := lock.WriteState(lower.State{Phase: lower.PhaseCommitted, Since: time.Now()}); err != nil {
			b.Fatal(err)
		}
		if err := discardPending(pending); err != nil {
			b.Fatal(err)
		}
		_ = ex.Release()
		_ = lock.Release()
	}
}
