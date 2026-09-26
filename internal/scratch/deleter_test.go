package scratch

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// lockedLog 는 삭제자의 노드 로그를 모은다.
type lockedLog struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedLog) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

func (l *lockedLog) count(s string) int { return strings.Count(l.String(), s) }

// fakeLauncher 는 namespace 없이 항목을 지운다. fail 에 적힌 항목은 그 오류를 한 번 돌려준다.
type fakeLauncher struct {
	trash Trash
	mu    sync.Mutex
	calls []string
	fail  map[string][]error
	hold  chan struct{} // 있으면 측정 뒤 여기서 기다린다
}

func (f *fakeLauncher) launch(ctx context.Context, entry string, measured func(Size)) error {
	f.mu.Lock()
	f.calls = append(f.calls, entry)
	var err error
	if errs := f.fail[entry]; len(errs) > 0 {
		err, f.fail[entry] = errs[0], errs[1:]
	}
	hold := f.hold
	f.mu.Unlock()
	var launch *LaunchError
	if errors.As(err, &launch) {
		return err
	}
	measured(Size{Bytes: 100, Entries: 2})
	if hold != nil {
		select {
		case <-hold:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(f.trash.Dir, entry))
}

func (f *fakeLauncher) called() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.calls...)
}

type deleterRig struct {
	d       *Deleter
	launch  *fakeLauncher
	log     *lockedLog
	mu      sync.Mutex
	changes []Usage
	cancel  context.CancelFunc
	done    chan struct{}
}

func newDeleterRig(t *testing.T, every time.Duration, entries ...string) *deleterRig {
	t.Helper()
	trash := TrashIn(t.TempDir())
	for _, e := range entries {
		write(t, filepath.Join(trash.Dir, e, "f"), "x")
	}
	r := &deleterRig{launch: &fakeLauncher{trash: trash, fail: map[string][]error{}}, log: &lockedLog{}}
	r.d = &Deleter{Trash: trash, Launch: r.launch.launch, Every: every,
		Log: slog.New(slog.NewTextHandler(r.log, &slog.HandlerOptions{Level: slog.LevelDebug})),
		Changed: func(u Usage) {
			r.mu.Lock()
			r.changes = append(r.changes, u)
			r.mu.Unlock()
		}}
	return r
}

func (r *deleterRig) start(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel, r.done = cancel, make(chan struct{})
	go func() { defer close(r.done); r.d.Run(ctx) }()
	t.Cleanup(r.stop)
}

func (r *deleterRig) stop() {
	r.cancel()
	<-r.done
}

func (r *deleterRig) seen() []Usage {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Usage(nil), r.changes...)
}

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// 기동하면 곧바로 한 번 돌아 trash 를 비운다. 항목마다 측정이 Usage 에 들어가고
// 끝나면 지우는 중이 꺼진다.
func TestDeleterEmptiesTheTrashAtStart(t *testing.T) {
	r := newDeleterRig(t, time.Hour, "a", "b")
	r.start(t)
	eventually(t, "the first pass", func() bool {
		u := r.d.Usage()
		return len(r.launch.called()) == 2 && u.TrashEntries == 0 && !u.Deleting
	})
	if got := r.launch.called(); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("calls = %v", got)
	}
	names, _ := r.d.Trash.Entries()
	if len(names) != 0 {
		t.Fatalf("trash still holds %v", names)
	}
	seen := r.seen()
	first := seen[0]
	if first.TrashEntries != 2 || first.TrashUnsized != 2 || !first.Deleting {
		t.Fatalf("first change = %+v", first)
	}
	var measured bool
	for _, u := range seen {
		if u.TrashBytes == 100 && u.TrashUnsized == 1 && u.Deleting {
			measured = true
		}
		if u.MeasuredAt.IsZero() {
			t.Fatalf("a change without its time: %+v", u)
		}
	}
	if !measured {
		t.Fatalf("the measurement never reached Usage: %+v", seen)
	}
	if n := r.log.count("trash entry removed"); n != 2 {
		t.Fatalf("removed lines = %d\n%s", n, r.log)
	}
}

// 지우는 동안의 Usage — 측정한 것은 bytes 에, 아직 안 한 것은 unsized 에.
func TestDeleterShowsWhatIsBeingDeleted(t *testing.T) {
	r := newDeleterRig(t, time.Hour, "a", "b")
	r.launch.hold = make(chan struct{})
	r.start(t)
	eventually(t, "the measurement", func() bool { return r.d.Usage().TrashBytes == 100 })
	u := r.d.Usage()
	if u.TrashEntries != 2 || u.TrashUnsized != 1 || !u.Deleting {
		t.Fatalf("usage while deleting = %+v", u)
	}
	close(r.launch.hold)
	eventually(t, "the end of the pass", func() bool { u := r.d.Usage(); return u.TrashEntries == 0 && !u.Deleting })
}

// 실패한 항목은 trash 에 남고 Every 뒤에 다시 한다.
func TestDeleterRetriesAFailedEntry(t *testing.T) {
	r := newDeleterRig(t, 20*time.Millisecond, "bad", "good")
	r.launch.fail["bad"] = []error{errors.New("permission denied")}
	r.start(t)
	eventually(t, "the retry", func() bool { return r.d.Usage().TrashEntries == 0 && len(r.launch.called()) == 3 })
	if got := r.launch.called(); !reflect.DeepEqual(got, []string{"bad", "good", "bad"}) {
		t.Fatalf("calls = %v", got)
	}
	if n := r.log.count("cannot remove trash entry; it stays in trash"); n != 1 {
		t.Fatalf("failure lines = %d\n%s", n, r.log)
	}
}

// 남긴 경로가 있으면 그 사실을 적는다. 항목은 남는다.
func TestDeleterLogsWhatWasLeftOnAnotherFilesystem(t *testing.T) {
	r := newDeleterRig(t, time.Hour, "mounted")
	r.launch.fail["mounted"] = []error{&LeftError{Left: []string{"mounted/upper/mnt"}}}
	r.start(t)
	eventually(t, "the left line", func() bool {
		return r.log.count("trash entry left in part; it reaches into another filesystem") == 1
	})
	if !strings.Contains(r.log.String(), "mounted/upper/mnt") {
		t.Fatalf("the left path is not in the log:\n%s", r.log)
	}
	eventually(t, "the end of the pass", func() bool { return !r.d.Usage().Deleting })
	if u := r.d.Usage(); u.TrashEntries != 1 || u.TrashUnsized != 0 || u.TrashBytes != 100 {
		t.Fatalf("usage = %+v", u)
	}
	if !strings.Contains((&LeftError{Left: []string{"a", "b"}}).Error(), "2 paths") {
		t.Fatal("left error text")
	}
}

// helper 를 못 띄우면 그 깸을 멈춘다. 같은 원인은 한 번만 적는다.
func TestDeleterStopsThePassWhenTheHelperCannotStart(t *testing.T) {
	r := newDeleterRig(t, 20*time.Millisecond, "a", "b")
	cause := &LaunchError{Err: errors.New(`exec: "unshare": executable file not found in $PATH`)}
	r.launch.fail["a"] = []error{cause, cause, cause}
	r.start(t)
	eventually(t, "three failed passes and a good one", func() bool {
		return r.d.Usage().TrashEntries == 0 && len(r.launch.called()) >= 5
	})
	// 앞의 세 깸은 a 에서 멈췄다 — b 를 안 열었다
	if got := r.launch.called()[:4]; !reflect.DeepEqual(got, []string{"a", "a", "a", "a"}) {
		t.Fatalf("calls = %v", r.launch.called())
	}
	if n := r.log.count("cannot start the trash helper; trash is not emptied"); n != 1 {
		t.Fatalf("launch lines = %d\n%s", n, r.log)
	}
	if !errors.Is(cause, cause.Err) || cause.Error() != cause.Err.Error() {
		t.Fatal("launch error does not unwrap")
	}
}

// 항목이 없으면 Every 에 깨지 않는다. Kick 은 막지 않고 겹치면 한 번으로 합친다.
func TestDeleterSleepsOnAnEmptyTrashUntilKicked(t *testing.T) {
	r := newDeleterRig(t, 5*time.Millisecond)
	r.d.Kick() // Run 앞의 Kick 도 막지 않는다
	r.d.Kick()
	r.start(t)
	eventually(t, "the start and the queued kick", func() bool { return len(r.seen()) == 2 })
	time.Sleep(50 * time.Millisecond) // Every 가 열 번 지나도 안 깬다
	if n := len(r.seen()); n != 2 {
		t.Fatalf("woke %d times on an empty trash", n)
	}
	write(t, filepath.Join(r.d.Trash.Dir, "late", "f"), "x")
	r.d.Kick()
	eventually(t, "the kicked pass", func() bool { return r.d.Usage().TrashEntries == 0 && len(r.launch.called()) == 1 })
}

// ctx 가 끝나면 도는 helper 를 멈추고 돌아온다. 반쯤 지운 항목은 trash 에 남는다.
func TestDeleterStopsWithItsContext(t *testing.T) {
	r := newDeleterRig(t, time.Hour, "a", "b")
	r.launch.hold = make(chan struct{})
	r.start(t)
	eventually(t, "the first launch", func() bool { return len(r.launch.called()) == 1 })
	r.stop()
	if got := r.launch.called(); !reflect.DeepEqual(got, []string{"a"}) {
		t.Fatalf("calls after stop = %v", got)
	}
	names, _ := r.d.Trash.Entries()
	if !reflect.DeepEqual(names, []string{"a", "b"}) {
		t.Fatalf("trash after stop = %v", names)
	}
	r.start(t) // 두 번째 Run 도 같은 채널을 쓴다
	close(r.launch.hold)
	eventually(t, "the next start", func() bool { return r.d.Usage().TrashEntries == 0 })
}

// trash 를 못 읽으면 원인을 한 번 적고 다음 깸을 기다린다. 로그가 없어도 돈다.
func TestDeleterWaitsOnAnUnreadableTrash(t *testing.T) {
	file := filepath.Join(t.TempDir(), "trash")
	write(t, file, "not a directory")
	logs := &lockedLog{}
	d := &Deleter{Trash: Trash{Dir: file}, Every: 5 * time.Millisecond,
		Launch: func(context.Context, string, func(Size)) error { return nil },
		Log:    slog.New(slog.NewTextHandler(logs, nil))}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	d.Run(ctx)
	if n := logs.count("cannot list trash; trash is not emptied"); n != 1 {
		t.Fatalf("list lines = %d\n%s", n, logs)
	}
	quiet := &Deleter{Trash: TrashIn(t.TempDir()), Launch: d.Launch}
	ctx2, cancel2 := context.WithCancel(context.Background())
	cancel2()
	quiet.Run(ctx2)
	quiet.log().Info("a nil logger discards")
}
