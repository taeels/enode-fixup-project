package lower

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// fastPoll 은 배타 대기의 간격을 줄인다. 시험이 끝나면 되돌린다.
func fastPoll(t *testing.T) {
	t.Helper()
	old := pollEvery
	pollEvery = 2 * time.Millisecond
	t.Cleanup(func() { pollEvery = old })
}

// sibling 은 같은 자리를 따로 연 Dir 이다 — 한 프로세스에서 형제를 흉내 낸다 (flock 은 열린 파일마다다).
func sibling(t *testing.T, f fixture) *Dir {
	t.Helper()
	d, err := Peek(f.lowers, f.root)
	if err != nil || d == nil {
		t.Fatalf("Peek = %v, %v", d, err)
	}
	return d
}

func mustShared(t *testing.T, d *Dir, h Holder) *Shared {
	t.Helper()
	s, ok, err := d.TryShared(h)
	if err != nil || !ok {
		t.Fatalf("TryShared(%s) = %v, %v", h.Node, ok, err)
	}
	return s
}

// 공유 + 공유 · 공유 중 TryBake · 공유 중 Exclusive 는 기다리다 공유를 놓으면 잡힌다 (FD 흐름 9.1).
func TestLocks(t *testing.T) {
	fastPoll(t)
	f := newFixture(t)
	a, b, c := f.open(t), sibling(t, f), sibling(t, f)

	sa := mustShared(t, a, Holder{Node: "node-a", Role: RoleCandidate})
	sb := mustShared(t, b, Holder{Node: "node-b", Role: RoleRun, Run: "R-1"})
	bake, ok, err := c.TryBake()
	if err != nil || !ok {
		t.Fatalf("TryBake under shared locks = %v, %v", ok, err)
	}
	if _, ok, err := a.TryBake(); ok || err != nil {
		t.Fatalf("the second TryBake = %v, %v; want false", ok, err)
	}

	got := make(chan error, 1)
	var ex *Exclusive
	var seen []Waiting
	var mu sync.Mutex
	go func() {
		var err error
		ex, err = c.Exclusive(context.Background(), time.Millisecond, func(w Waiting) {
			mu.Lock()
			seen = append(seen, w)
			mu.Unlock()
		})
		got <- err
	}()
	select {
	case err := <-got:
		t.Fatalf("Exclusive did not wait for two shared locks: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
	if err := sa.Release(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-got:
		t.Fatalf("Exclusive did not wait for the second shared lock: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
	// 놓기 전의 마지막 관찰을 본다. Release 는 기록을 먼저 닫고 lower.lock 을 나중에 닫으므로
	// 그 사이의 관찰은 Unnamed 일 수 있다.
	mu.Lock()
	if len(seen) == 0 {
		mu.Unlock()
		t.Fatal("watch was never called while two shared locks were held")
	}
	last := seen[len(seen)-1]
	mu.Unlock()
	_ = sb.Release()
	if err := <-got; err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	if len(seen) == 0 || len(seen[0].Holders) != 2 || seen[0].Unnamed ||
		seen[0].Holders[0].Node != "node-a" || seen[0].Holders[1].Run != "R-1" {
		t.Errorf("watch saw %+v", seen)
	}
	mu.Unlock()
	if len(last.Holders) != 1 || last.Holders[0].Node != "node-b" {
		t.Errorf("after node-a released, watch saw %+v", last)
	}
	// 배타가 쥐어져 있으면 공유는 안 잡힌다 — 형제는 bake 출처로 drain 한다
	if _, ok, err := a.TryShared(Holder{Node: "node-a"}); ok || err != nil {
		t.Fatalf("TryShared under an exclusive lock = %v, %v", ok, err)
	}
	_ = ex.Release()
	_ = ex.Release()
	_ = bake.Release()
	_ = bake.Release()
	_ = (*Exclusive)(nil).Release()
	_ = (*Bake)(nil).Release()
	_ = (*Shared)(nil).Release()
}

// 배타는 ctx 의 마감에서 멈춘다 — bake 가 merge_wait_timeout 으로 적는다.
func TestExclusiveDeadline(t *testing.T) {
	fastPoll(t)
	f := newFixture(t)
	a, b := f.open(t), sibling(t, f)
	s := mustShared(t, a, Holder{Node: "node-a"})
	defer func() { _ = s.Release() }()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := b.Exclusive(ctx, time.Hour, nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Exclusive past its deadline = %v", err)
	}
}

// smoke 의 공유 — 배타가 쥐어져 있으면 다시 보며 기다리고, 기다리는 동안 state.json 을 넘긴다.
// 기록을 안 남기므로 배타를 기다리는 쪽은 Unnamed 로 본다.
func TestWaitShared(t *testing.T) {
	fastPoll(t)
	f := newFixture(t)
	a, b := f.open(t), sibling(t, f)
	bake, _, _ := a.TryBake()
	defer func() { _ = bake.Release() }()
	if err := bake.WriteState(State{Phase: PhaseMerging, Owner: &Owner{Run: "R-5", Node: "baker"}, Since: time.Now()}); err != nil {
		t.Fatal(err)
	}
	ex, err := a.Exclusive(context.Background(), time.Hour, nil)
	if err != nil {
		t.Fatal(err)
	}
	var notices []State
	got := make(chan *Shared, 1)
	go func() {
		s, err := b.WaitShared(context.Background(), time.Millisecond, func(st State) { notices = append(notices, st) })
		if err != nil {
			t.Error(err)
		}
		got <- s
	}()
	time.Sleep(20 * time.Millisecond)
	_ = ex.Release()
	s := <-got
	if s == nil || s.Recorded() {
		t.Fatalf("WaitShared = %+v", s)
	}
	if len(notices) == 0 || notices[0].Owner == nil || notices[0].Owner.Run != "R-5" {
		t.Errorf("notices = %+v", notices)
	}
	// 기록 없는 쥔 쪽 — 배타를 기다리는 쪽은 Unnamed 로 본다
	var w Waiting
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := a.Exclusive(ctx, time.Hour, func(x Waiting) { w = x }); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Exclusive over a smoke = %v", err)
	}
	if !w.Unnamed || len(w.Holders) != 0 {
		t.Errorf("waiting = %+v; want unnamed", w)
	}
	_ = s.Release()

	// 마감이면 멈춘다
	ex, _ = a.Exclusive(context.Background(), time.Hour, nil)
	ctx, cancel = context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := b.WaitShared(ctx, time.Millisecond, nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("WaitShared past its deadline = %v", err)
	}
	_ = ex.Release()

	// lower.lock 이 없으면 잠그지 않는다 (계획 4절 ⑰)
	if err := os.Remove(filepath.Join(a.Path, lowerLock)); err != nil {
		t.Fatal(err)
	}
	if s, err := b.WaitShared(context.Background(), time.Millisecond, nil); s != nil || err != nil {
		t.Errorf("WaitShared without lower.lock = %+v, %v", s, err)
	}
}

// 쥔 사람 기록 — 살아 있는 것만 · Release 뒤 죽은 기록 · 기록이 살아 있으면 lower.lock 이 막혀 있다 ·
// 이름이 규칙 밖인 node 는 기록 없이 쥔다 · Update 는 바뀐 칸이 있을 때만 쓴다.
func TestHolders(t *testing.T) {
	fastPoll(t)
	f := newFixture(t)
	a, b := f.open(t), sibling(t, f)
	since := time.Date(2026, 9, 27, 4, 0, 0, 0, time.UTC)
	sa := mustShared(t, a, Holder{Node: "node-a", Label: "box-a", Role: RoleCandidate, Since: since})
	sb := mustShared(t, b, Holder{Node: "node-b", Role: RoleCandidate, Since: since})
	if !sa.Recorded() {
		t.Fatal("node-a is not recorded")
	}
	hs, err := a.Holders()
	if err != nil || len(hs) != 2 || hs[0].Label != "box-a" || hs[0].PID != os.Getpid() || !hs[0].Since.Equal(since) {
		t.Fatalf("Holders = %+v, %v", hs, err)
	}
	// 기록이 살아 있으면 lower.lock 이 막혀 있다 — 잡을 때 lower.lock 다음 기록이다
	bk, ok, _ := b.TryBake()
	if !ok {
		t.Fatal("TryBake")
	}
	defer func() { _ = bk.Release() }()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, err := b.Exclusive(ctx, time.Hour, nil); err == nil {
		t.Fatal("an exclusive lock was taken while a live record exists")
	}
	_ = sb.Release()
	if hs, _ := a.Holders(); len(hs) != 1 || hs[0].Node != "node-a" {
		t.Fatalf("after Release = %+v", hs)
	}
	if _, err := os.Stat(filepath.Join(a.Path, holdersDir, "node-b.json")); err != nil {
		t.Errorf("a dead record was removed: %v", err)
	}

	// Update — 바뀐 칸이 있을 때만 쓴다
	p := filepath.Join(a.Path, holdersDir, "node-a.json")
	before, _ := os.Stat(p)
	if err := sa.Update(Holder{Node: "someone-else", Label: "box-a", Role: RoleCandidate, Since: since}); err != nil {
		t.Fatal(err)
	}
	if after, _ := os.Stat(p); !os.SameFile(before, after) {
		t.Error("Update rewrote an unchanged record")
	}
	if err := sa.Update(Holder{Label: "box-a", Role: RoleRun, Run: "R-2", Since: since, Acks: 1}); err != nil {
		t.Fatal(err)
	}
	if hs, _ := a.Holders(); len(hs) != 1 || hs[0].Node != "node-a" || hs[0].Role != RoleRun || hs[0].Run != "R-2" || hs[0].Acks != 1 {
		t.Errorf("after Update = %+v", hs)
	}
	_ = sa.Release()
	if hs, _ := a.Holders(); len(hs) != 0 {
		t.Errorf("all released, yet %+v", hs)
	}

	// 이름이 규칙 밖 — 공유는 쥐고 기록은 없다
	s := mustShared(t, a, Holder{Node: "Node/../x"})
	if s.Recorded() {
		t.Error("an invalid node id was recorded")
	}
	if err := s.Update(Holder{Role: RoleRun}); err != nil {
		t.Error(err)
	}
	if hs, _ := a.Holders(); len(hs) != 0 {
		t.Errorf("an unrecorded holder is listed: %+v", hs)
	}
	_ = s.Release()

	// 같은 node_id 의 기록 잠금을 남이 쥐고 있으면 기록 없이 공유만 쥔다
	s1 := mustShared(t, a, Holder{Node: "node-c"})
	s2 := mustShared(t, b, Holder{Node: "node-c"})
	if !s1.Recorded() || s2.Recorded() {
		t.Errorf("recorded %v %v; want true false", s1.Recorded(), s2.Recorded())
	}
	_ = s1.Release()
	_ = s2.Release()

	// 점으로 시작하는 이름 · .json 이 아닌 이름 · 깨진 기록은 건너뛴다
	dir := filepath.Join(a.Path, holdersDir)
	writeFile(t, filepath.Join(dir, ".node-d.json.tmp-1"), "{")
	writeFile(t, filepath.Join(dir, "notes.txt"), "x")
	s3 := mustShared(t, a, Holder{Node: "node-e"})
	writeFile(t, filepath.Join(dir, "node-e.json"), "{")
	if hs, err := a.Holders(); err != nil || len(hs) != 0 {
		t.Errorf("Holders = %+v, %v", hs, err)
	}
	_ = s3.Release()
	// holders 가 없으면 아무도 없다
	_ = os.RemoveAll(dir)
	if hs, err := a.Holders(); hs != nil || err != nil {
		t.Errorf("Holders without the directory = %+v, %v", hs, err)
	}
	if _, _, err := a.TryShared(Holder{Node: "node-f"}); err == nil {
		t.Error("TryShared recorded into a missing holders directory")
	}
}
