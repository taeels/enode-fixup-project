package enode

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/scratch"
)

func statusBook(t *testing.T) (*StatusBook, string, *lockedBuffer) {
	t.Helper()
	cfg := filepath.Join(t.TempDir(), "local.yaml")
	logs := &lockedBuffer{}
	return NewStatusBook(cfg, slog.New(slog.NewTextHandler(logs, nil))), cfg, logs
}

func statusTime(t *testing.T, cfg string) time.Time {
	t.Helper()
	fi, err := os.Stat(StatusPath(cfg))
	if err != nil {
		t.Fatal(err)
	}
	return fi.ModTime()
}

// 같은 값이면 파일이 안 바뀐다. 칸 하나만 바뀌어도 파일 전체를 다시 쓴다.
func TestStatusBookWritesOnlyWhenSomethingChanged(t *testing.T) {
	b, cfg, _ := statusBook(t)
	at := time.Now().UTC().Truncate(time.Second)
	caps := Capabilities{Caps: []contract.Capability{{Capability: "agent.reason"}}, At: at}
	b.SetCaps(caps)
	disk, _ := diskDrain(0, 10, false)
	drain := DrainStatus{Effective: contract.DrainGraceful, Sources: []DrainSource{disk}, At: at}
	b.SetDrain(drain)
	usage := scratch.Usage{TrashBytes: 9 << 30, TrashEntries: 2, TrashUnsized: 1, Deleting: true, MeasuredAt: at}
	b.SetScratch(usage)

	// 쓴 시각을 옛날로 돌려 두고 같은 값을 다시 준다 — 안 써야 한다
	old := time.Now().Add(-time.Hour).Truncate(time.Second)
	if err := os.Chtimes(StatusPath(cfg), old, old); err != nil {
		t.Fatal(err)
	}
	later := drain
	later.At = at.Add(time.Minute) // 다음 광고 — 값이 같으면 At 은 정한 광고의 것으로 남는다
	b.SetCaps(caps)
	b.SetDrain(later)
	b.SetScratch(usage)
	if !statusTime(t, cfg).Equal(old) {
		t.Fatal("the same values rewrote the status file")
	}

	s, err := ReadStatus(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Caps) != 1 || !s.At.Equal(at) || s.Drain == nil || !s.Drain.At.Equal(at) ||
		s.Drain.Sources[0].Detail != "free 0 GB < min 10 GB" || s.Scratch == nil || !sameUsage(*s.Scratch, usage) {
		t.Fatalf("status = %+v drain %+v scratch %+v", s, s.Drain, s.Scratch)
	}

	// 칸 하나만 바뀐다 — 파일을 쓰고 다른 칸은 그대로다
	usage.Deleting, usage.MeasuredAt = false, at.Add(time.Second)
	b.SetScratch(usage)
	if statusTime(t, cfg).Equal(old) {
		t.Fatal("a changed scratch did not write the file")
	}
	s, _ = ReadStatus(cfg)
	if s.Scratch.Deleting || s.Drain == nil || len(s.Caps) != 1 {
		t.Fatalf("status after one change = %+v", s)
	}
	// 출처가 바뀌어도 쓴다 — 실린 값이 같아도
	owner, _ := OwnerDrain(contract.DrainGraceful)
	b.SetDrain(DrainStatus{Effective: contract.DrainGraceful, Sources: []DrainSource{owner, disk}, At: at})
	s, _ = ReadStatus(cfg)
	if len(s.Drain.Sources) != 2 {
		t.Fatalf("a new source was not written: %+v", s.Drain)
	}
}

// 광고와 삭제자가 동시에 불러도 서로의 칸을 안 덮는다.
func TestStatusBookKeepsEveryFieldUnderConcurrentWriters(t *testing.T) {
	b, cfg, _ := statusBook(t)
	start := time.Now().UTC().Truncate(time.Second)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			b.SetCaps(Capabilities{Caps: []contract.Capability{{Capability: "agent.reason"}}, At: start.Add(time.Duration(i) * time.Second)})
			mode := contract.DrainGraceful
			if i%2 == 0 {
				mode = contract.DrainNone
			}
			b.SetDrain(DrainStatus{Effective: mode, At: start})
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			b.SetScratch(scratch.Usage{TrashEntries: i, MeasuredAt: start.Add(time.Duration(i) * time.Second)})
		}
	}()
	wg.Wait()
	s, err := ReadStatus(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !s.At.Equal(start.Add(49*time.Second)) || s.Drain == nil || s.Drain.Effective != contract.DrainGraceful ||
		s.Scratch == nil || s.Scratch.TrashEntries != 49 {
		t.Fatalf("a writer lost its field: at %v drain %+v scratch %+v", s.At, s.Drain, s.Scratch)
	}
}

// 옛 상태 파일(칸 둘)이 그대로 읽힌다 — 새 칸은 nil 이다.
func TestAnOldStatusFileStillReads(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "local.yaml")
	old := "caps:\n  - capability: agent.reason\n    attrs:\n      harness: claude\nat: 2026-09-20T10:00:00Z\n"
	if err := os.WriteFile(StatusPath(cfg), []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := ReadStatus(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Caps) != 1 || s.At.IsZero() || s.Drain != nil || s.Scratch != nil {
		t.Fatalf("status = %+v", s)
	}
}

// 못 쓰면 원인이 바뀔 때만 경고하고, 다음 부름이 값이 같아도 다시 쓴다. nil 은 아무것도 안 한다.
func TestStatusBookRetriesAFailedWriteAndWarnsOnce(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "missing", "local.yaml") // 디렉터리가 없어 못 쓴다
	logs := &lockedBuffer{}
	b := NewStatusBook(cfg, slog.New(slog.NewTextHandler(logs, nil)))
	usage := scratch.Usage{TrashEntries: 1, MeasuredAt: time.Now().UTC()}
	b.SetScratch(usage)
	b.SetScratch(usage)
	if n := strings.Count(logs.String(), "cannot write the status file; the panel will show capabilities as unknown"); n != 1 {
		t.Fatalf("warned %d times:\n%s", n, logs)
	}
	if err := os.MkdirAll(filepath.Dir(cfg), 0o700); err != nil {
		t.Fatal(err)
	}
	b.SetScratch(usage) // 값은 같지만 앞의 쓰기가 실패했다
	s, err := ReadStatus(cfg)
	if err != nil || s.Scratch == nil || s.Scratch.TrashEntries != 1 {
		t.Fatalf("the retry did not write: %+v %v", s, err)
	}
	var none *StatusBook
	none.SetCaps(Capabilities{At: time.Now()})
	none.SetDrain(DrainStatus{})
	none.SetScratch(usage)
	if NewStatusBook(cfg, nil).log == nil {
		t.Fatal("a nil logger was kept")
	}
}
