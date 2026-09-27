package lower

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func mode(t *testing.T, p string) os.FileMode {
	t.Helper()
	fi, err := os.Lstat(p)
	if err != nil {
		t.Fatal(err)
	}
	return fi.Mode().Perm()
}

// 새 자리의 권한은 umask 와 무관하게 0700 · 0600 이다 (계획 3.1 · 측정 1).
func TestOpenModesIgnoreUmask(t *testing.T) {
	for _, mask := range []int{0o000, 0o002, 0o022, 0o077} {
		old := syscall.Umask(mask)
		f := newFixture(t)
		d := f.open(t)
		syscall.Umask(old)
		for _, p := range []string{f.lowers, d.Path, filepath.Join(d.Path, holdersDir)} {
			if m := mode(t, p); m != 0o700 {
				t.Errorf("umask %04o: %s is %04o, want 0700", mask, p, m)
			}
		}
		for _, name := range []string{lowerLock, bakeLock, identityFile} {
			if m := mode(t, filepath.Join(d.Path, name)); m != 0o600 {
				t.Errorf("umask %04o: %s is %04o, want 0600", mask, name, m)
			}
		}
		if len(d.Notes) != 0 {
			t.Errorf("umask %04o: a fresh place has notes %q", mask, d.Notes)
		}
	}
}

// 남에게 열린 자리는 데몬이 좁히고 한 줄씩 남긴다. Peek 은 좁히지 않고 알린다 (ADR-073).
func TestLoosePermissions(t *testing.T) {
	f := newFixture(t)
	d := f.open(t)
	holders := filepath.Join(d.Path, holdersDir)
	lock := filepath.Join(d.Path, lowerLock)
	for p, m := range map[string]os.FileMode{f.lowers: 0o755, d.Path: 0o755, holders: 0o777, lock: 0o644} {
		if err := os.Chmod(p, m); err != nil {
			t.Fatal(err)
		}
	}
	p, err := Peek(f.lowers, f.root)
	if err != nil || p == nil {
		t.Fatal(err)
	}
	if len(p.Loose) != 4 || mode(t, f.lowers) != 0o755 || mode(t, lock) != 0o644 {
		t.Errorf("Peek loose = %q; lowers %04o lock %04o", p.Loose, mode(t, f.lowers), mode(t, lock))
	}
	got := Check(f.lowers, f.lower, f.dir, os.Getuid())
	if len(got) != 3 || !got[2].OK || got[2].Observed != "matches; the node narrows loose permissions on start" {
		t.Errorf("Check identity = %+v", got)
	}

	d2 := f.open(t)
	if len(d2.Notes) != 4 {
		t.Errorf("notes = %q", d2.Notes)
	}
	for _, p := range []string{f.lowers, d.Path, holders} {
		if m := mode(t, p); m != 0o700 {
			t.Errorf("%s is %04o after Open", p, m)
		}
	}
	if m := mode(t, lock); m != 0o600 {
		t.Errorf("lower.lock is %04o after Open", m)
	}
	if !strings.Contains(strings.Join(d2.Notes, "\n"), "narrowed loose permissions on "+lock+" to 0600") {
		t.Errorf("notes = %q", d2.Notes)
	}
	// 잠금 파일은 쥘 때도 좁힌다
	if err := os.Chmod(lock, 0o666); err != nil {
		t.Fatal(err)
	}
	s, ok, err := d2.TryShared(Holder{Node: "n1"})
	if err != nil || !ok {
		t.Fatal(err)
	}
	_ = s.Release()
	if m := mode(t, lock); m != 0o600 {
		t.Errorf("lower.lock is %04o after TryShared", m)
	}
}

// symlink 는 어느 층에서든 거절하고 밖에 아무것도 만들지 않는다 — 가리키는 곳이 없는 symlink 도 (측정 2 · 3).
func TestSymlinksAreRefused(t *testing.T) {
	for _, c := range []struct {
		name string
		link func(f fixture, d *Dir) string // 바꿔 끼울 자리 — 이 자리를 지우고 밖을 가리키는 symlink 로 만든다
	}{
		{"lowers", func(f fixture, d *Dir) string { return f.lowers }},
		{"key", func(f fixture, d *Dir) string { return d.Path }},
		{"holders", func(f fixture, d *Dir) string { return filepath.Join(d.Path, holdersDir) }},
		{"lower.lock", func(f fixture, d *Dir) string { return filepath.Join(d.Path, lowerLock) }},
		{"bake.lock", func(f fixture, d *Dir) string { return filepath.Join(d.Path, bakeLock) }},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newFixture(t)
			d := f.open(t)
			place := c.link(f, d)
			if err := os.RemoveAll(place); err != nil {
				t.Fatal(err)
			}
			outside := filepath.Join(f.dir, "outside")
			if err := os.Symlink(outside, place); err != nil {
				t.Fatal(err)
			}
			if _, err := Open(f.lowers, f.root, time.Now()); err == nil {
				t.Fatal("Open followed a symlink")
			}
			if _, err := os.Lstat(outside); !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("Open made something outside: %v", err)
			}
			if d, err := Peek(f.lowers, f.root); err == nil {
				t.Fatalf("Peek followed a symlink: %+v", d)
			}
			// 밖이 있어도 같다 — 디렉터리를 가리키면 열리지 않고, 파일을 가리키면 ELOOP 다
			if err := os.Mkdir(outside, 0o700); err != nil {
				t.Fatal(err)
			}
			if _, err := Open(f.lowers, f.root, time.Now()); err == nil {
				t.Fatal("Open followed a symlink to a directory")
			}
			if ents, _ := os.ReadDir(outside); len(ents) != 0 {
				t.Fatalf("Open made %v outside", ents)
			}
		})
	}
	// 잠금을 쥐는 자리에서도 같다
	f := newFixture(t)
	d := f.open(t)
	lock := filepath.Join(d.Path, lowerLock)
	_ = os.Remove(lock)
	if err := os.Symlink(filepath.Join(f.dir, "gone"), lock); err != nil {
		t.Fatal(err)
	}
	if _, _, err := d.TryShared(Holder{Node: "n1"}); err == nil || !strings.Contains(err.Error(), "too many levels of symbolic links") {
		t.Errorf("TryShared through a symlink = %v", err)
	}
	if _, err := d.WaitShared(t.Context(), time.Millisecond, nil); err == nil {
		t.Error("WaitShared followed a symlink")
	}
	if _, err := d.Exclusive(t.Context(), time.Millisecond, nil); err == nil {
		t.Error("Exclusive followed a symlink")
	}
	if _, _, err := d.TryBake(); err != nil {
		t.Errorf("bake.lock is untouched, yet TryBake = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(f.dir, "gone")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a lock made something outside: %v", err)
	}
}

// 주인이 다른 자리 (계획 4절 ⑳). 특권 없이 남의 파일을 만들 수 없어서 root 소유의 / 를 lowers 로 넘긴다 —
// Open 이 아무것도 만들지 않고 거절한다. root 로 돌면 / 가 자기 것이라 표 시험(TestPermVerdict)만 본다.
func TestForeignOwnerIsRefused(t *testing.T) {
	if os.Getuid() == 0 {
		t.Log("running as root; / is ours, so TestPermVerdict covers the foreign owner")
		return
	}
	f := newFixture(t)
	want := "lower: /: owned by uid 0, not the node user"
	if _, err := Open("/", f.root, time.Now()); err == nil || !strings.HasPrefix(err.Error(), want) {
		t.Fatalf("Open under / = %v", err)
	}
	if _, err := os.Lstat(filepath.Join("/", f.root.Key.String())); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Open made something under /: %v", err)
	}
	if _, err := Peek("/", f.root); err == nil || !strings.HasPrefix(err.Error(), want) {
		t.Fatalf("Peek under / = %v", err)
	}
	got := Check("/", f.lower, f.dir, os.Getuid())
	if len(got) != 3 || got[2].OK || !strings.HasPrefix(got[2].Observed, "cannot read /: owned by uid 0") {
		t.Errorf("Check identity = %+v", got)
	}
}

// 읽는 파일의 종류와 크기 — 디렉터리 · 1 MiB 넘는 파일은 못 읽은 것이다.
func TestReadJSONLimits(t *testing.T) {
	f := newFixture(t)
	d := f.open(t)
	p := filepath.Join(d.Path, stateFile)
	if err := os.Mkdir(p, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := d.ReadState(); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Errorf("state.json as a directory = %v", err)
	}
	_ = os.Remove(p)
	writeFile(t, p, `{"schema":1,"phase":"committed","pending_upper":"`+strings.Repeat("x", maxJSON)+`"}`)
	if _, err := d.ReadState(); err == nil || !strings.Contains(err.Error(), "larger than") {
		t.Errorf("a large state.json = %v", err)
	}
	// 0644 인 기록은 읽는다 — 읽기는 좁히지 않는다
	writeFile(t, p, `{"schema":1,"phase":"building"}`)
	_ = os.Chmod(p, 0o644)
	if st, err := d.ReadState(); err != nil || st.Phase != PhaseBuilding {
		t.Errorf("a 0644 state.json = %+v, %v", st, err)
	}
	// 자리 밖의 디렉터리로 쓰기를 부르면 그 자리의 오류다
	if err := writeJSON(filepath.Join(f.dir, "nope"), "x.json", 1, false, 0); err == nil || !strings.Contains(err.Error(), "nope") {
		t.Errorf("writeJSON into a missing directory = %v", err)
	}
	if err := writeJSON(f.dir, "x.json", func() {}, false, 0); err == nil {
		t.Error("writeJSON of a function succeeded")
	}
	if err := syncDir(filepath.Join(f.dir, "nope")); err == nil {
		t.Error("syncDir of a missing directory succeeded")
	}
}
