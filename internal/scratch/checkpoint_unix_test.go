//go:build unix

package scratch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// testStore 는 임시 scratch 의 spool 이다. 여유는 넉넉하고 inode 수는 안 낸다 — 판정의 몫은 각 시험이 정한다.
func testStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s := &Store{Dir: SpoolIn(dir), Scratch: dir, Trash: TrashIn(dir),
		Policy: Policy{Mode: ModeOnFailure, TTL: time.Hour, CapacityPercent: 20, MaxBytes: 32 * gib},
		Stat:   func() (Filesystem, error) { return Filesystem{Known: true, Free: 1000 * gib}, nil }}
	if check, err := s.Open(); err != nil || !check.Created || check.Refused != "" {
		t.Fatalf("open = %+v %v", check, err)
	}
	return s
}

func reserve(t *testing.T, s *Store) *Reservation {
	t.Helper()
	r, err := s.Reserve(Entry{Node: "n1", Run: "run-1", Seq: 3, Step: "build", Attempt: 1, Runtime: "runc-overlay",
		Format: FormatOverlayUpper, Scope: ScopeWorkspaceUpper, Guarantee: GuaranteeInspectOnly})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// kept 는 upper 하나를 든 확정된 보존본이다.
func kept(t *testing.T, s *Store) *Reservation {
	t.Helper()
	r := reserve(t, s)
	if err := os.MkdirAll(filepath.Join(r.Upper(), "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.Upper(), "sub", "built"), []byte("built"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Commit(r, r.Entry()); err != nil {
		t.Fatal(err)
	}
	return r
}

func record(t *testing.T, s *Store, id string) Entry {
	t.Helper()
	e, err := readRecord(filepath.Join(s.Dir, id))
	if err != nil {
		t.Fatal(err)
	}
	return *e
}

func modeOf(t *testing.T, path string) os.FileMode {
	t.Helper()
	fi, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	return fi.Mode().Perm()
}

// NFR C8 — 느슨한 비트는 좁히고, symlink · 디렉터리 아님 · 남의 것은 고치지 않고 거절한다.
func TestOpenSpool_NarrowsAndRefuses(t *testing.T) {
	t.Run("narrows", func(t *testing.T) {
		dir := t.TempDir()
		spool := SpoolIn(dir)
		if err := os.Mkdir(spool, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(spool, 0o755); err != nil {
			t.Fatal(err)
		}
		s := &Store{Dir: spool, Scratch: dir}
		check, err := s.Open()
		if err != nil || !check.Narrowed || check.From != 0o755 || check.Refused != "" || modeOf(t, spool) != 0o700 {
			t.Fatalf("open = %+v %v · mode %o", check, err, modeOf(t, spool))
		}
	})
	refuse := func(t *testing.T, setup func(dir, spool string), want string) {
		dir := t.TempDir()
		spool := SpoolIn(dir)
		setup(dir, spool)
		s := &Store{Dir: spool, Scratch: dir, Trash: TrashIn(dir)}
		check, err := s.Open()
		if err != nil || check.Refused != want || s.Refused() != want {
			t.Fatalf("open = %+v %v, want refused %q", check, err, want)
		}
		if _, err := s.Reserve(Entry{}); !errors.Is(err, ErrRefused) {
			t.Fatalf("reserve on a refused spool = %v", err)
		}
	}
	t.Run("symlink", func(t *testing.T) {
		refuse(t, func(dir, spool string) {
			elsewhere := filepath.Join(dir, "elsewhere")
			if err := os.Mkdir(elsewhere, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(elsewhere, spool); err != nil {
				t.Fatal(err)
			}
		}, "symlink")
	})
	t.Run("not a directory", func(t *testing.T) {
		refuse(t, func(_, spool string) {
			if err := os.WriteFile(spool, nil, 0o600); err != nil {
				t.Fatal(err)
			}
		}, "not a directory")
	})
	t.Run("owned by another uid", func(t *testing.T) {
		orig := spoolUID
		spoolUID = func() int { return os.Getuid() + 1 }
		t.Cleanup(func() { spoolUID = orig })
		refuse(t, func(_, spool string) {
			if err := os.Mkdir(spool, 0o700); err != nil {
				t.Fatal(err)
			}
		}, fmt.Sprintf("owned by uid %d", os.Getuid()))
	})
}

// NFR C1 — spool · 항목 0700 · 기록과 요약 0600 은 umask 와 무관하다.
func TestSpoolModes_IgnoreUmask(t *testing.T) {
	old := syscall.Umask(0)
	t.Cleanup(func() { syscall.Umask(old) })
	s := testStore(t)
	r := kept(t, s)
	if _, err := s.Settle(context.Background()); err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]os.FileMode{
		s.Dir:                                 0o700,
		r.Dir:                                 0o700,
		filepath.Join(r.Dir, recordName):      0o600,
		filepath.Join(r.Dir, SessionLockName): 0o600,
		filepath.Join(s.Dir, summaryName):     0o600,
	} {
		if got := modeOf(t, path); got != want {
			t.Fatalf("%s is %o, want %o", filepath.Base(path), got, want)
		}
	}
}

// NFR C7 — 기록은 symlink 를 안 따라가고 · 보통 파일이고 · 1 MiB 까지만 읽는다.
func TestRecord_ReadLimits(t *testing.T) {
	dir := t.TempDir()
	if err := writeRecord(dir, Entry{ID: "3f9a1c0b7d2e", State: EntryKept}); err != nil {
		t.Fatal(err)
	}
	if e, err := readRecord(dir); err != nil || e.ID != "3f9a1c0b7d2e" {
		t.Fatalf("read = %+v %v", e, err)
	}
	link := t.TempDir()
	if err := os.Symlink(filepath.Join(dir, recordName), filepath.Join(link, recordName)); err != nil {
		t.Fatal(err)
	}
	if _, err := readRecord(link); err == nil {
		t.Fatal("a record behind a symlink was read")
	}
	big := t.TempDir()
	if err := os.WriteFile(filepath.Join(big, recordName), []byte(`{"id":"x","pad":"`+strings.Repeat("a", recordLimit)+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readRecord(big); err == nil {
		t.Fatal("a record over 1 MiB was read")
	}
	notFile := t.TempDir()
	if err := os.Mkdir(filepath.Join(notFile, recordName), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := readRecord(notFile); err == nil {
		t.Fatal("a directory was read as a record")
	}
	empty := t.TempDir()
	if err := os.WriteFile(filepath.Join(empty, recordName), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readRecord(empty); err == nil {
		t.Fatal("an empty record was read")
	}
}

func TestReserveCommitDispose(t *testing.T) {
	s := testStore(t)
	r := reserve(t, s)
	if !ValidID(r.ID) || r.Upper() != filepath.Join(s.Dir, r.ID, upperName) {
		t.Fatalf("reservation = %+v", r)
	}
	if e := record(t, s, r.ID); e.State != EntryReserved || e.Run != "run-1" || e.Schema != recordSchema {
		t.Fatalf("reserved record = %+v", e)
	}
	if exists, free := probeLock(filepath.Join(r.Dir, SessionLockName)); !exists || free {
		t.Fatalf("the entry lock is not held: exists %v free %v", exists, free)
	}
	if _, err := os.Stat(r.Upper()); !os.IsNotExist(err) {
		t.Fatalf("the keep path must not exist yet: %v", err)
	}

	// 확정 — kept · 잡은 시각과 만료 시각 · 잠금이 풀린다
	e := r.Entry()
	e.Lower, e.Environment, e.Head, e.IR = "fsid-1/ino-2", "env-1", "abc", "ir-1"
	got, err := s.Commit(r, e)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != EntryKept || got.CapturedAt == nil || got.ExpiresAt == nil ||
		got.ExpiresAt.Sub(*got.CapturedAt) != time.Hour || got.Lower != "fsid-1/ino-2" {
		t.Fatalf("committed = %+v", got)
	}
	if _, free := probeLock(filepath.Join(r.Dir, SessionLockName)); !free {
		t.Fatal("commit did not release the entry lock")
	}
	if err := s.Dispose(r); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(r.Dir); err != nil {
		t.Fatalf("dispose removed a committed checkpoint: %v", err)
	}

	// 버림 — 빈 예약은 폴더째 지운다
	d := reserve(t, s)
	if err := s.Dispose(d); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(d.Dir); !os.IsNotExist(err) {
		t.Fatalf("a disposed reservation is left: %v", err)
	}
	// 무엇이 더 들어 있으면 trash 로 간다
	m := reserve(t, s)
	if err := os.Mkdir(m.Upper(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := s.Dispose(m); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(s.Trash.Dir, m.ID, upperName)); err != nil {
		t.Fatalf("a non-empty reservation did not go to trash: %v", err)
	}
	// Abandon — 항목 폴더째 trash
	a := reserve(t, s)
	if err := s.Abandon(a); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(s.Trash.Dir, a.ID, recordName)); err != nil {
		t.Fatalf("an abandoned reservation is not in trash: %v", err)
	}
}

func TestReserve_FailsWithoutASpool(t *testing.T) {
	dir := t.TempDir()
	s := &Store{Dir: SpoolIn(dir), Scratch: dir}
	if _, err := s.Reserve(Entry{}); err == nil || !strings.HasPrefix(err.Error(), "open the spool: ") {
		t.Fatalf("reserve without a spool = %v", err)
	}
}

// NFR Design D2 — 측정은 spool 잠금 밖이고, 그새 퇴출된 항목의 측정값은 버린다.
func TestSettle_MeasuresOutsideTheLock(t *testing.T) {
	s := testStore(t)
	first, second := kept(t, s), kept(t, s)
	var lockFree []bool
	s.Measure = func(_ context.Context, root, entry string) (Size, error) {
		_, free := probeLock(filepath.Join(s.Dir, spoolLockName))
		lockFree = append(lockFree, free)
		if filepath.Base(root) == second.ID {
			// 형제의 판정이 그새 퇴출했다
			e := record(t, s, second.ID)
			e.State, e.Evicted = EntryEvicted, EvictOverCapacity
			if err := writeRecord(root, e); err != nil {
				t.Fatal(err)
			}
		}
		if entry != upperName {
			t.Fatalf("measured %q, want the upper", entry)
		}
		return Size{Bytes: 4096, Entries: 3}, nil
	}
	res, err := s.Settle(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(lockFree) != 2 || !lockFree[0] || !lockFree[1] {
		t.Fatalf("the spool lock was held while measuring: %v", lockFree)
	}
	if e := record(t, s, first.ID); e.Bytes == nil || *e.Bytes != 4096 || *e.Inodes != 3 || e.MeasuredAt == nil {
		t.Fatalf("first = %+v", e)
	}
	if e := record(t, s, second.ID); e.Bytes != nil || e.State != EntryEvicted {
		t.Fatalf("a value for an entry evicted meanwhile was written: %+v", e)
	}
	if res.Usage.Checkpoints != 1 || res.Usage.Bytes != 4096 || res.Usage.Unsized != 0 || s.Kept() != 4096 {
		t.Fatalf("usage = %+v kept %d", res.Usage, s.Kept())
	}
}

func TestSettle_LaunchErrorStops(t *testing.T) {
	s := testStore(t)
	kept(t, s)
	kept(t, s)
	calls := 0
	s.Measure = func(context.Context, string, string) (Size, error) {
		calls++
		return Size{}, &LaunchError{Err: errors.New("unshare failed")}
	}
	res, err := s.Settle(context.Background())
	if err != nil || res.Launch == nil || calls != 1 || res.Usage.Unsized != 2 {
		t.Fatalf("settle = %+v %v after %d calls", res, err, calls)
	}
}

func TestSettle_EvictsAndExpires(t *testing.T) {
	s := testStore(t)
	clock := time.Now().UTC()
	s.Now = func() time.Time { return clock }
	big, small, old := kept(t, s), kept(t, s), kept(t, s)
	s.Measure = func(_ context.Context, root, _ string) (Size, error) {
		if filepath.Base(root) == big.ID {
			return Size{Bytes: 33 * gib, Entries: 1}, nil
		}
		return Size{Bytes: gib, Entries: 1}, nil
	}
	// old 는 이미 만료되었다
	e := record(t, s, old.ID)
	past := clock.Add(-time.Minute)
	e.ExpiresAt = &past
	if err := writeRecord(old.Dir, e); err != nil {
		t.Fatal(err)
	}
	res, err := s.Settle(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Expired) != 1 || res.Expired[0] != old.ID || !res.Moved {
		t.Fatalf("expired = %v", res.Expired)
	}
	if _, err := os.Stat(old.Dir); !os.IsNotExist(err) {
		t.Fatalf("an expired entry is still in the spool: %v", err)
	}
	if len(res.Evicted) != 1 || res.Evicted[0] != (Eviction{big.ID, EvictLargerThanMaxGB}) {
		t.Fatalf("evicted = %v", res.Evicted)
	}
	if e := record(t, s, big.ID); e.State != EntryEvicted || e.Evicted != EvictLargerThanMaxGB || e.EvictedAt == nil {
		t.Fatalf("evicted record = %+v", e)
	}
	if _, err := os.Stat(big.Upper()); !os.IsNotExist(err) {
		t.Fatalf("the evicted upper is still in the spool: %v", err)
	}
	if e := record(t, s, small.ID); e.State != EntryKept {
		t.Fatalf("small = %+v", e)
	}
	if res.Usage.Checkpoints != 1 || res.Usage.Bytes != gib {
		t.Fatalf("usage = %+v", res.Usage)
	}
	// 만료된 퇴출 기록은 다음 판정이 폴더째 거둔다
	clock = clock.Add(2 * time.Hour)
	res, err = s.Settle(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Expired) != 2 {
		t.Fatalf("expired after the ttl = %v", res.Expired)
	}
}

func TestUsageSummary_RoundTrip(t *testing.T) {
	s := testStore(t)
	if s.Kept() != 0 {
		t.Fatal("no summary must read as nothing kept")
	}
	at := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	if err := writeSummary(s.Dir, Summary{Bytes: 41 * gib, Inodes: 812345, At: at}); err != nil {
		t.Fatal(err)
	}
	sum, err := readSummary(s.Dir)
	if err != nil || sum.Bytes != 41*gib || sum.Inodes != 812345 || !sum.At.Equal(at) || s.Kept() != 41*gib {
		t.Fatalf("summary = %+v %v", sum, err)
	}
}

// 도는 단계의 예약은 목록에 없고, 주인 없는 예약은 incomplete 다 (NFR Design 답 1).
func TestList_HidesHeldReservations(t *testing.T) {
	s := testStore(t)
	held := reserve(t, s)
	k := kept(t, s)
	ownerless := reserve(t, s)
	ownerless.mu.Lock()
	ownerless.releaseLocked() // 잠금만 놓는다 — 데몬이 죽은 모양
	ownerless.mu.Unlock()
	list, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]Listed{}
	for _, l := range list {
		ids[l.ID] = l
	}
	if _, ok := ids[held.ID]; ok || len(list) != 2 {
		t.Fatalf("list = %+v", list)
	}
	if l := ids[ownerless.ID]; !l.Incomplete || l.State != EntryReserved {
		t.Fatalf("ownerless = %+v", l)
	}
	if l := ids[k.ID]; l.Incomplete || l.Path != k.Upper() {
		t.Fatalf("kept = %+v", l)
	}
	if got, err := s.Lookup(held.ID); err != nil || got != nil {
		t.Fatalf("lookup of a running reservation = %+v %v", got, err)
	}
	if got, err := s.Lookup(k.ID); err != nil || got == nil || got.ID != k.ID {
		t.Fatalf("lookup = %+v %v", got, err)
	}
	if got, err := s.Lookup("../escaped"); err != nil || got != nil {
		t.Fatalf("lookup of a bad id = %+v %v", got, err)
	}
}

func TestReconcile(t *testing.T) {
	s := testStore(t)
	held := reserve(t, s)
	k := kept(t, s)
	ownerless := reserve(t, s)
	ownerless.mu.Lock()
	ownerless.releaseLocked()
	ownerless.mu.Unlock()
	if err := os.Mkdir(filepath.Join(s.Dir, "notes"), 0o700); err != nil {
		t.Fatal(err)
	}
	res, err := s.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Trashed) != 1 || res.Trashed[0] != ownerless.ID {
		t.Fatalf("trashed = %v", res.Trashed)
	}
	if len(res.Marked) != 1 || res.Marked[0] != k.ID || record(t, s, k.ID).Report != ReportUnknown {
		t.Fatalf("marked = %v", res.Marked)
	}
	if len(res.Unknown) != 1 || res.Unknown[0] != "notes" {
		t.Fatalf("unknown = %v", res.Unknown)
	}
	if _, err := os.Stat(held.Dir); err != nil {
		t.Fatalf("a running reservation was touched: %v", err)
	}
}

func TestReport(t *testing.T) {
	s := testStore(t)
	k := kept(t, s)
	if err := s.Report(k.ID, ReportDelivered); err != nil {
		t.Fatal(err)
	}
	if e := record(t, s, k.ID); e.Report != ReportDelivered {
		t.Fatalf("report = %q", e.Report)
	}
	if err := s.Report("000000000000", ReportDelivered); err != nil {
		t.Fatalf("a report for a gone entry = %v", err)
	}
	if ok, err := s.SameFilesystem(); err != nil || !ok {
		t.Fatalf("same filesystem = %v %v", ok, err)
	}
}
