//go:build linux

package enode

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/taeels/enode/internal/lower"
)

// Baker · 치우는 몸통 · 기동 정리 (bake 유닛 · FD 엔티티 2 · 3절 · FD 규칙 11 · 12.1 · 12.2절). 몸통의 차례
// (옮기기 -> committed -> 굽기 Release -> DropBake -> held 비움) 에서 시험은 마지막 (held == nil) 을 기다린 뒤 앞의
// 것을 본다 (계획 5절 규칙 2).

// 고루틴 서른둘이 abandon 과 startMerging 을 함께 불러도 몸통은 한 번이다. 몸통이 돌았으면 startMerging 은 모두
// false 이고, startMerging 이 한 번이라도 true 면 몸통은 돌지 않았다. -race 로 돈다 (계획 5절 규칙 7).
func TestHeldBake_OneBodyUnderRace(t *testing.T) {
	f := newBakeFixture(t)
	h := f.hold(t, "r1")
	var writes atomic.Int32
	was := writeLowerState
	writeLowerState = func(b *lower.Bake, st lower.State) error { writes.Add(1); return was(b, st) }
	t.Cleanup(func() { writeLowerState = was })
	var started atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Go(func() {
			if i%2 == 0 {
				h.abandon(reasonBakeRunEnded, nil)
				return
			}
			if h.startMerging() {
				started.Add(1)
			}
		})
	}
	wg.Wait()
	switch {
	case writes.Load() > 1:
		t.Fatalf("the body ran %d times", writes.Load())
	case writes.Load() == 1 && started.Load() != 0:
		t.Fatalf("the body ran and %d merges started", started.Load())
	case writes.Load() == 0 && started.Load() == 0:
		t.Fatal("neither the body nor a merge ran")
	}
}

func TestHeldBake_Body(t *testing.T) {
	t.Run("after startMerging the body does nothing", func(t *testing.T) {
		f := newBakeFixture(t)
		h := f.hold(t, "r1")
		if !h.startMerging() {
			t.Fatal("startMerging = false")
		}
		h.abandon(reasonBakeRunEnded, nil)
		if f.held() != h || f.state(t).Phase != lower.PhasePending {
			t.Fatalf("the body ran after startMerging: held %v state %+v", f.held(), f.state(t))
		}
	})
	t.Run("after cancelMerging the body runs", func(t *testing.T) {
		f := newBakeFixture(t)
		h := f.hold(t, "r1")
		h.startMerging()
		h.cancelMerging()
		h.abandon(reasonBakeRunEnded, nil)
		if f.held() != nil || f.state(t).Phase != lower.PhaseCommitted {
			t.Fatalf("the body did not run after cancelMerging: %+v", f.state(t))
		}
	})
	t.Run("after the body startMerging is false", func(t *testing.T) {
		f := newBakeFixture(t)
		h := f.hold(t, "r1")
		h.abandon(reasonBakeRunEnded, nil)
		if h.startMerging() {
			t.Fatal("startMerging = true after the body")
		}
	})
	t.Run("the body moves, writes committed, releases, drops and clears", func(t *testing.T) {
		f := newBakeFixture(t)
		h := f.hold(t, "r1")
		pending := h.pendingDir()
		if f.guardBake() != "r1" {
			t.Fatalf("HoldBake did not take: %q", f.guardBake())
		}
		builds := h.draftBuilds()
		h.abandon(reasonBakeRunEnded, builds)
		if f.held() != nil {
			t.Fatal("held was not cleared")
		}
		if _, err := os.Stat(pending); !os.IsNotExist(err) {
			t.Fatalf("the pending directory stayed: %v", err)
		}
		if got := entries(t, filepath.Join(f.scratch, "trash")); len(got) != 1 || got[0] != filepath.Base(pending) {
			t.Fatalf("trash = %v", got)
		}
		st := f.state(t)
		la := st.LastAttempt
		if st.Phase != lower.PhaseCommitted || st.Owner != nil || st.PendingUpper != "" || la == nil || la.Run != "r1" ||
			la.Reason != reasonBakeRunEnded || !reflect.DeepEqual(la.Builds, builds) || la.At.IsZero() {
			t.Fatalf("state = %+v last_attempt %+v", st, la)
		}
		f.holdBakeLock(t) // 굽기 잠금이 풀렸다
		if f.guardBake() != "" {
			t.Fatal("DropBake was not called")
		}
		h.abandon("again", nil)
		if f.state(t).LastAttempt.Reason != reasonBakeRunEnded {
			t.Fatal("a second body ran")
		}
	})
	t.Run("a pending directory that cannot move still ends committed", func(t *testing.T) {
		f := newBakeFixture(t)
		h := f.hold(t, "r1")
		blockTrash(t, f.scratch)
		h.abandon(reasonBakeRunEnded, nil)
		if f.held() != nil || f.state(t).Phase != lower.PhaseCommitted {
			t.Fatalf("state = %+v", f.state(t))
		}
		if _, err := os.Stat(h.pendingDir()); err != nil {
			t.Fatalf("the pending directory should be left: %v", err)
		}
		if !f.nodeLogHas("bake: cannot move the pending upper to trash; it is left for the start-up cleanup") {
			t.Fatalf("node log = %s", f.nodeLog)
		}
	})
	t.Run("a committed that cannot be written still releases", func(t *testing.T) {
		f := newBakeFixture(t)
		h := f.hold(t, "r1")
		restore := f.failStateWrites(t)
		h.abandon(reasonBakeRunEnded, nil)
		restore()
		if f.held() != nil || f.state(t).Phase != lower.PhasePending {
			t.Fatalf("held %v state %+v", f.held(), f.state(t))
		}
		f.holdBakeLock(t)
		if !f.nodeLogHas("bake: cannot write the lower state while discarding the bake") {
			t.Fatalf("node log = %s", f.nodeLog)
		}
	})
	t.Run("held is cleared only for its own bake", func(t *testing.T) {
		f := newBakeFixture(t)
		h := f.hold(t, "r1")
		other := &heldBake{b: f.b, run: "r2"}
		f.b.mu.Lock()
		f.b.held = other
		f.b.mu.Unlock()
		h.abandon(reasonBakeRunEnded, nil)
		if f.held() != other {
			t.Fatal("the body cleared another bake")
		}
	})
}

// 기동 (FD 규칙 12.2 의 표) — committed · building · pending · TryBake 가 안 됨 · TryBake 오류 · Dir 이 nil ·
// guard 가 nil. 어느 경우든 OnStale 이 등록돼 있다. merging 줄은 재개 시험이 본다.
func TestStartBaker(t *testing.T) {
	registered := func(t *testing.T, g *LowerGuard) {
		t.Helper()
		g.mu.Lock()
		defer g.mu.Unlock()
		if g.onStale == nil {
			t.Fatal("OnStale is not registered")
		}
	}
	t.Run("committed moves this scratch's abandoned pending directories", func(t *testing.T) {
		f := newBakeFixture(t)
		left := f.pendingIn(t, f.scratch, "old", false)
		f.startBaker()
		registered(t, f.g)
		if _, err := os.Stat(left); !os.IsNotExist(err) {
			t.Fatalf("the abandoned directory stayed: %v", err)
		}
		if got := entries(t, filepath.Join(f.scratch, "trash")); len(got) != 1 {
			t.Fatalf("trash = %v", got)
		}
		f.holdBakeLock(t)
	})
	for _, phase := range []lower.Phase{lower.PhaseBuilding, lower.PhasePending} {
		t.Run(string(phase)+" is cleaned into the scratch it sits on", func(t *testing.T) {
			f := newBakeFixture(t)
			theirs := filepath.Join(filepath.Dir(f.ws), "their-scratch")
			if err := os.Mkdir(theirs, 0o700); err != nil {
				t.Fatal(err)
			}
			recorded := f.pendingIn(t, theirs, "R-dead", phase == lower.PhasePending)
			mine := f.pendingIn(t, f.scratch, "R-older", false)
			f.putState(t, lower.State{Phase: phase, Owner: &lower.Owner{Run: "R-dead", Node: "node-b"},
				PendingUpper: filepath.Join(recorded, "upper")})
			f.startBaker()
			registered(t, f.g)
			if got := entries(t, filepath.Join(theirs, "trash")); len(got) != 1 || got[0] != filepath.Base(recorded) {
				t.Fatalf("their trash = %v", got)
			}
			if got := entries(t, filepath.Join(f.scratch, "trash")); len(got) != 1 || got[0] != filepath.Base(mine) {
				t.Fatalf("my trash = %v", got)
			}
			st := f.state(t)
			la := st.LastAttempt
			if st.Phase != lower.PhaseCommitted || la == nil || la.Run != "R-dead" || la.Reason != abandonedReason(phase) {
				t.Fatalf("state = %+v %+v", st, la)
			}
			if (phase == lower.PhasePending) != (len(la.Builds) == 1) {
				t.Fatalf("builds = %+v; a pending has a draft, a building has none", la.Builds)
			}
			f.holdBakeLock(t)
			if !f.nodeLogHas("bake: cleaned a stale bake") {
				t.Fatalf("node log = %s", f.nodeLog)
			}
		})
	}
	t.Run("a recorded directory that is already gone counts as moved", func(t *testing.T) {
		f := newBakeFixture(t)
		gone := filepath.Join(f.scratch, pendingDirName, f.key(t), "bake-gone", "upper")
		f.putState(t, lower.State{Phase: lower.PhasePending, Owner: &lower.Owner{Run: "R-dead"}, PendingUpper: gone})
		f.startBaker()
		if st := f.state(t); st.Phase != lower.PhaseCommitted {
			t.Fatalf("state = %+v", st)
		}
	})
	t.Run("another node holds the bake lock", func(t *testing.T) {
		f := newBakeFixture(t)
		recorded := f.pendingIn(t, f.scratch, "R-live", true)
		f.putState(t, lower.State{Phase: lower.PhasePending, Owner: &lower.Owner{Run: "R-live"},
			PendingUpper: filepath.Join(recorded, "upper")})
		lock := f.holdBakeLock(t)
		f.startBaker()
		registered(t, f.g)
		if st := f.state(t); st.Phase != lower.PhasePending {
			t.Fatalf("a live bake was cleaned: %+v", st)
		}
		if _, err := os.Stat(recorded); err != nil {
			t.Fatalf("a live pending directory moved: %v", err)
		}
		_ = lock.Release()
	})
	t.Run("TryBake fails", func(t *testing.T) {
		f := newBakeFixture(t)
		p := filepath.Join(f.g.Dir().Path, "bake.lock")
		if err := os.Remove(p); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(p, 0o700); err != nil {
			t.Fatal(err)
		}
		f.startBaker()
		registered(t, f.g)
		if !f.nodeLogHas("bake: cannot take the bake lock; skipping the start-up cleanup") {
			t.Fatalf("node log = %s", f.nodeLog)
		}
	})
	t.Run("the state directory did not open", func(t *testing.T) {
		dir := t.TempDir()
		log := &lockedBuffer{}
		g := newLowerGuard(filepath.Join(dir, "lowers"), filepath.Join(dir, "missing"), Identity{NodeID: "n"},
			slogTo(log))
		g.start()
		b := StartBaker(context.Background(), g, t.TempDir(), Identity{NodeID: "n"}, "i", slogTo(log))
		defer b.Wait()
		registered(t, g)
		if !strings.Contains(log.String(), "bake: the lower state directory is not open; skipping the start-up cleanup") {
			t.Fatalf("log = %s", log)
		}
	})
	t.Run("no guard, no baker", func(t *testing.T) {
		if b := StartBaker(context.Background(), nil, t.TempDir(), Identity{}, "", discardLog()); b != nil {
			t.Fatalf("baker = %+v", b)
		}
		var b *Baker
		b.Wait()
	})
}

// 정리는 pending_upper 가 이 노드가 쓰는 모양이 아니면 그 경로를 옮기지 않는다 — 다른 lower 키 · pending 이 아닌
// 조각 · upper 가 아닌 끝. 상태만 committed + abandoned 이고 노드 로그에 경로 한 줄이다 (FD 규칙 12.2 끝 · 16.3).
func TestBakerClean_AnOddPendingPathIsLeft(t *testing.T) {
	for name, odd := range map[string]func(f *bakeFixture, t *testing.T) string{
		"another lower key": func(f *bakeFixture, t *testing.T) string {
			return filepath.Join(f.scratch, pendingDirName, "1-2", "bake-x", "upper")
		},
		"not a pending segment": func(f *bakeFixture, t *testing.T) string {
			return filepath.Join(f.scratch, "elsewhere", f.key(t), "bake-x", "upper")
		},
		// 자기 scratch 의 pending/<키>/ 아래는 모양과 무관하게 버려진 자리라 기동이 치운다 — 다른 scratch 에 둔다
		"not an upper": func(f *bakeFixture, t *testing.T) string {
			return filepath.Join(filepath.Dir(f.ws), "their-scratch", pendingDirName, f.key(t), "bake-x", "lower")
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := newBakeFixture(t)
			p := odd(f, t)
			write(t, p, "precious", "keep me")
			f.putState(t, lower.State{Phase: lower.PhasePending, Owner: &lower.Owner{Run: "R-odd"}, PendingUpper: p})
			f.startBaker()
			if b, err := os.ReadFile(filepath.Join(p, "precious")); err != nil || string(b) != "keep me" {
				t.Fatalf("the odd path was touched: %q %v", b, err)
			}
			st := f.state(t)
			if st.Phase != lower.PhaseCommitted || st.LastAttempt == nil ||
				st.LastAttempt.Reason != abandonedReason(lower.PhasePending) {
				t.Fatalf("state = %+v", st)
			}
			if !f.nodeLogHas("bake: the pending upper path does not look like one this node writes; leaving it") ||
				!f.nodeLogHas(p) {
				t.Fatalf("node log = %s", f.nodeLog)
			}
		})
	}
}

// build (git 모양 · repo 모양) · merge · 재개를 돌려도 호스트의 bash · sh · git · repo 가 한 번도 불리지 않는다 — PATH 를
// 표지 스크립트 폴더로 바꿔 본다. 같은 패키지 함수와 claim.go · finalize.go 의 길까지 덮는다 (계획 3.1 「행동 시험을
// 더하는 까닭」). t.Setenv 라 t.Parallel 을 안 쓴다.
func TestBakeFlows_RunNoHostProgram(t *testing.T) {
	f := newBakeFixture(t)
	useTestMergeHelper(t, "")
	marker := markerPATH(t, "bash", "sh", "git", "repo")
	f.succeed()
	f.w.execute(context.Background(), buildStepOf("r1"))
	f.w.execute(context.Background(), mergeStepOf("r1"))
	f.rt.answer(probeScript, probeRepo())
	f.rt.answer(pinCommand, pinWrites("<m/>"))
	f.w.Held.Add(Lease{RunID: "r2", Node: f.w.Ident.NodeID, NotAfter: time.Now().Add(time.Hour)})
	f.w.execute(context.Background(), buildStepOf("r2"))
	f.w.execute(context.Background(), mergeStepOf("r2"))
	pending := f.interrupted(t, lower.PhaseMerging, f.scratch, true, true)
	f.staleOnce(t)
	resumed(t, f, pending)
	f.m.mu.Lock()
	results := append([]Result(nil), f.m.results...)
	f.m.mu.Unlock()
	if len(results) != 4 {
		t.Fatalf("results = %+v", results)
	}
	for _, res := range results {
		if res.Error != "" {
			t.Fatalf("a step failed: %+v", res)
		}
	}
	if b, err := os.ReadFile(marker); !os.IsNotExist(err) {
		t.Fatalf("the bake ran a host program:\n%s", b)
	}
}
