package enode

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	execenv "github.com/taeels/enode/internal/environment"
	"github.com/taeels/enode/internal/lower"
)

// 준비도 점검 셋과 smoke 의 잠금 (FD 흐름 8절 · 계획 4절 ⑦ ⑯ ⑰).

// homeFixture 는 $HOME 을 임시 폴더로 옮기고 그 아래 워크스페이스와 scratch 를 둔다.
type homeFixture struct {
	home, ws, scratch, lowers string
}

func newHomeFixture(t *testing.T) homeFixture {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	f := homeFixture{home: home, ws: filepath.Join(home, "ws"), scratch: filepath.Join(home, "scratch"),
		lowers: filepath.Join(home, ".local", "state", "enode", "lowers")}
	if err := os.Mkdir(f.ws, 0o755); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f homeFixture) dir(t *testing.T) *lower.Dir {
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

func runcDoc(driver string) execenv.Document {
	var doc execenv.Document
	doc.Profile.Runtime.Driver = driver
	return doc
}

func factNamed(t *testing.T, facts []execenv.Fact, name string) execenv.Fact {
	t.Helper()
	for _, f := range facts {
		if f.Name == name {
			return f
		}
	}
	t.Fatalf("no %s in %+v", name, facts)
	return execenv.Fact{}
}

// runc-overlay 만 셋을 낸다. 같은 마운트 · 노드 uid · 신원 — 모두 ready 이고 Source 는 /runtime 이다.
func TestFacts_ReadyAndNativeHasNone(t *testing.T) {
	f := newHomeFixture(t)
	b := execenv.Binding{Workspace: f.ws, Scratch: f.scratch}
	v := ExecutionRuntimeVerifier{}
	if got := v.Facts(context.Background(), runcDoc("native"), b); got != nil {
		t.Fatalf("native = %+v", got)
	}
	facts := v.Facts(context.Background(), runcDoc("runc-overlay"), b)
	if len(facts) != 3 {
		t.Fatalf("facts = %+v", facts)
	}
	for _, x := range facts {
		if x.State != execenv.StateReady || x.Source != "/runtime" {
			t.Errorf("fact = %+v", x)
		}
	}
	if got := factNamed(t, facts, "lower.identity").Observed; got != "not recorded yet; the node records it on start" {
		t.Errorf("identity = %s", got)
	}
	if _, err := os.Lstat(f.lowers); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the check made the state directory: %v", err)
	}
}

// 다른 filesystem 은 invalid · 다른 디렉터리의 끊긴 굽기는 external-blocked · home 을 못 찾으면 external-blocked.
func TestFacts_NotReady(t *testing.T) {
	f := newHomeFixture(t)
	v := ExecutionRuntimeVerifier{}
	if shm, err := os.MkdirTemp("/dev/shm", "enode-check-"); err == nil {
		defer func() { _ = os.RemoveAll(shm) }()
		fs := factNamed(t, v.Facts(context.Background(), runcDoc("runc-overlay"), execenv.Binding{Workspace: f.ws, Scratch: shm}),
			"binding.scratch_filesystem")
		if !strings.HasPrefix(fs.Observed, "same mount") && (fs.State != execenv.StateInvalid ||
			!strings.HasPrefix(fs.Observed, "different filesystem")) {
			t.Errorf("scratch on /dev/shm = %+v", fs)
		}
	} else {
		t.Logf("no /dev/shm here (%v); TestFactOf covers the mapping", err)
	}

	d := f.dir(t)
	if d.Root.BirthNs == 0 {
		t.Log("this filesystem gives no birth time; the identity table in internal/lower covers Foreign")
		return
	}
	b, _, err := d.TryBake()
	if err != nil {
		t.Fatal(err)
	}
	if err := b.WriteState(lower.State{Phase: lower.PhasePending, Owner: &lower.Owner{Run: "R-7"}, Since: time.Now()}); err != nil {
		t.Fatal(err)
	}
	_ = b.Release()
	p := filepath.Join(d.Path, "lower.json")
	var rec lower.Identity
	if body, err := os.ReadFile(p); err != nil || json.Unmarshal(body, &rec) != nil {
		t.Fatalf("lower.json: %v", err)
	}
	rec.BirthNs++ // 다른 디렉터리의 기록
	body, _ := json.Marshal(rec)
	if err := os.WriteFile(p, body, 0o600); err != nil {
		t.Fatal(err)
	}
	id := factNamed(t, v.Facts(context.Background(), runcDoc("runc-overlay"), execenv.Binding{Workspace: f.ws, Scratch: f.scratch}),
		"lower.identity")
	if id.State != execenv.StateExternalBlocked || id.Observed != "recorded for another directory and a bake is pending (run R-7)" ||
		!strings.HasPrefix(id.Remediation, "inspect "+d.Path) {
		t.Errorf("identity = %+v", id)
	}

	t.Setenv("HOME", "")
	facts := v.Facts(context.Background(), runcDoc("runc-overlay"), execenv.Binding{Workspace: f.ws, Scratch: f.scratch})
	id = factNamed(t, facts, "lower.identity")
	if len(facts) != 3 || id.State != execenv.StateExternalBlocked || !strings.HasPrefix(id.Observed, "cannot find the home directory: ") {
		t.Errorf("without HOME = %+v", facts)
	}
}

// 까닭의 갈래를 State 로 — binding 은 invalid, state 는 external-blocked (답 8).
func TestFactOf(t *testing.T) {
	for _, c := range []struct {
		f    lower.Finding
		want execenv.State
	}{
		{lower.Finding{OK: true}, execenv.StateReady},
		{lower.Finding{Cause: lower.CauseBinding}, execenv.StateInvalid},
		{lower.Finding{Cause: lower.CauseState}, execenv.StateExternalBlocked},
	} {
		if got := factOf(c.f); got.State != c.want || got.Source != "/runtime" {
			t.Errorf("factOf(%+v) = %+v", c.f, got)
		}
	}
}

// fastSmoke 는 smoke 의 다시 보는 간격을 줄인다.
func fastSmoke(t *testing.T) {
	t.Helper()
	poll, notice := smokePoll, smokeNotice
	smokePoll = 2 * time.Millisecond
	t.Cleanup(func() { smokePoll, smokeNotice = poll, notice })
}

// smoke 의 잠금 — 배타를 쥔 Dir 이 놓으면 풀린다 · Notice 는 처음 한 줄 · 자리가 없거나 lower.lock 이 없으면 안 잠근다.
func TestSmokeLock(t *testing.T) {
	fastSmoke(t)
	f := newHomeFixture(t)
	if s, err := smokeLock(context.Background(), f.ws, nil); s != nil || err != nil {
		t.Fatalf("no state directory = %+v, %v", s, err)
	}
	d := f.dir(t)
	b, _, _ := d.TryBake()
	if err := b.WriteState(lower.State{Phase: lower.PhaseMerging, Owner: &lower.Owner{Run: "R-3", Node: "baker"},
		Since: time.Date(2026, 9, 27, 6, 0, 0, 0, time.UTC)}); err != nil {
		t.Fatal(err)
	}
	_ = b.Release()
	ex, err := d.Exclusive(context.Background(), time.Hour, nil)
	if err != nil {
		t.Fatal(err)
	}
	var notice lockedBuffer
	got := make(chan *lower.Shared, 1)
	go func() {
		s, err := smokeLock(context.Background(), f.ws, &notice)
		if err != nil {
			t.Error(err)
		}
		got <- s
	}()
	time.Sleep(30 * time.Millisecond)
	_ = ex.Release()
	s := <-got
	if s == nil {
		t.Fatal("the smoke did not take the lower lock")
	}
	want := "env check: waiting for the lower merge to finish (run R-3 on node baker, since 2026-09-27T06:00:00Z)\n"
	if notice.String() != want {
		t.Errorf("notice = %q", notice.String())
	}
	// 쥔 동안은 merge 가 배타를 못 잡는다 — smoke 는 기록 없는 쥔 쪽이다
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	var w lower.Waiting
	if _, err := d.Exclusive(ctx, time.Hour, func(x lower.Waiting) { w = x }); !errors.Is(err, context.DeadlineExceeded) || !w.Unnamed {
		t.Fatalf("Exclusive over a smoke = %v %+v", err, w)
	}
	_ = s.Release()

	// 마감이면 smoke 오류다
	ex, _ = d.Exclusive(context.Background(), time.Hour, nil)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel2()
	if _, err := smokeLock(ctx2, f.ws, nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("smoke past its deadline = %v", err)
	}
	_ = ex.Release()

	// lower.lock 이 없으면 안 잠근다 — 데몬 기동 때 점검이 Open 보다 먼저 돈다
	if err := os.Remove(filepath.Join(d.Path, "lower.lock")); err != nil {
		t.Fatal(err)
	}
	if s, err := smokeLock(context.Background(), f.ws, nil); s != nil || err != nil {
		t.Errorf("no lower.lock = %+v, %v", s, err)
	}
	// 워크스페이스를 못 읽으면 · home 을 못 찾으면 smoke 오류다
	if _, err := smokeLock(context.Background(), filepath.Join(f.home, "gone"), nil); err == nil {
		t.Error("smoke over a missing workspace took no error")
	}
	t.Setenv("HOME", "")
	if _, err := smokeLock(context.Background(), f.ws, nil); err == nil || !strings.HasPrefix(err.Error(), "cannot find the home directory: ") {
		t.Errorf("smoke without HOME = %v", err)
	}
}

// LogNotice 는 줄마다 Info 한 번이다. 끝나지 않은 줄은 다음 쓰기를 기다린다.
func TestLogNotice(t *testing.T) {
	var buf lockedBuffer
	w := LogNotice(slog.New(slog.NewTextHandler(&buf, nil)))
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := w.Write([]byte("first line\n")); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if n, err := w.Write([]byte("second\n\nthi")); err != nil || n != len("second\n\nthi") {
		t.Fatalf("Write = %d, %v", n, err)
	}
	if strings.Count(buf.String(), "level=INFO") != 3 || strings.Contains(buf.String(), "thi") {
		t.Fatalf("log = %s", buf.String())
	}
	_, _ = w.Write([]byte("rd\n"))
	if !strings.Contains(buf.String(), `msg=third`) || strings.Count(buf.String(), "level=INFO") != 4 {
		t.Fatalf("log = %s", buf.String())
	}
}
