//go:build linux

package enode

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"

	"github.com/taeels/enode/internal/lower"
)

// 대기 자리와 초안 파일 (bake 유닛 · FD 엔티티 4절 · 계획 4.1 7 · 30 · 32번).

func mode(t *testing.T, p string) os.FileMode {
	t.Helper()
	fi, err := os.Lstat(p)
	if err != nil {
		t.Fatal(err)
	}
	return fi.Mode().Perm()
}

// 초안은 0600 이고 대기 자리의 세 층은 0700 이다. 쓴 것을 그대로 되읽는다.
func TestDraft_IsPrivate(t *testing.T) {
	scratchDir := t.TempDir()
	key := lower.Key{FSID: 0x35b60f8473d0c15, Ino: 2}.String()
	dir, err := makePending(scratchDir, key)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{filepath.Join(scratchDir, pendingDirName), filepath.Join(scratchDir, pendingDirName, key), dir} {
		if m := mode(t, p); m != 0o700 {
			t.Fatalf("%s is %o, want 700", p, m)
		}
	}
	if s, ok := pendingShape(filepath.Join(dir, "upper"), key); !ok || s != scratchDir {
		t.Fatalf("a pending directory this node made does not have its shape: %q %v", s, ok)
	}
	d := fullDraft()
	if err := writeDraft(dir, d); err != nil {
		t.Fatal(err)
	}
	if m := mode(t, filepath.Join(dir, draftName)); m != 0o600 {
		t.Fatalf("the draft is %o, want 600", m)
	}
	got, err := readDraft(dir, os.Getuid())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Source.Pinned, d.Source.Pinned) || got.Run != d.Run || got.Node != d.Node ||
		*got.PreviousIR != *d.PreviousIR || !got.Source.SyncedAt.Equal(d.Source.SyncedAt) {
		t.Fatalf("read back %+v", got)
	}
	if ents, _ := os.ReadDir(dir); len(ents) != 1 {
		t.Fatalf("a temporary file was left: %v", ents)
	}
}

// 초안 읽기가 거절하는 것 — symlink · 보통 파일이 아님 · 1 MiB 넘음 · schema 가 1 이 아님 · 주인이 다름.
func TestDraft_RefusesWhatItShouldNotRead(t *testing.T) {
	good := func(t *testing.T) string {
		dir := t.TempDir()
		if err := writeDraft(dir, fullDraft()); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	cases := []struct {
		name  string
		setup func(t *testing.T) (string, int)
		want  string
	}{
		{"symlink", func(t *testing.T) (string, int) {
			host := filepath.Join(good(t), draftName)
			dir := t.TempDir()
			if err := os.Symlink(host, filepath.Join(dir, draftName)); err != nil {
				t.Fatal(err)
			}
			return dir, os.Getuid()
		}, "not a regular file"},
		{"a directory", func(t *testing.T) (string, int) {
			dir := t.TempDir()
			if err := os.Mkdir(filepath.Join(dir, draftName), 0o700); err != nil {
				t.Fatal(err)
			}
			return dir, os.Getuid()
		}, "not a regular file"},
		{"larger than 1 MiB", func(t *testing.T) (string, int) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, draftName), make([]byte, draftMax+1), 0o600); err != nil {
				t.Fatal(err)
			}
			return dir, os.Getuid()
		}, "larger than"},
		{"schema 2", func(t *testing.T) (string, int) {
			dir := t.TempDir()
			d := fullDraft()
			d.Schema = 2
			if err := writeDraft(dir, d); err != nil {
				t.Fatal(err)
			}
			return dir, os.Getuid()
		}, "schema 2"},
		{"another owner", func(t *testing.T) (string, int) { return good(t), os.Getuid() + 1 }, "is owned by uid"},
		{"missing", func(t *testing.T) (string, int) { return t.TempDir(), os.Getuid() }, "no such file"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir, uid := tc.setup(t)
			_, err := readDraft(dir, uid)
			if err == nil || !strings.HasPrefix(err.Error(), "cannot read the bake draft: ") ||
				!strings.Contains(err.Error(), tc.want) {
				t.Fatalf("readDraft = %v, want %q", err, tc.want)
			}
		})
	}
}

// 대기 자리는 그 자리가 있는 scratch 의 trash 로 간다 — scratch 가 둘이면 각자의 trash 로. 없는 자리는 옮겨진
// 것이다. 자기 scratch 의 나머지는 keep 밖이 모두 옮겨진다.
func TestDiscardPending_GoesToItsOwnScratch(t *testing.T) {
	key := "1-2"
	mine, theirs := t.TempDir(), t.TempDir()
	a, err := makePending(mine, key)
	if err != nil {
		t.Fatal(err)
	}
	b, err := makePending(theirs, key)
	if err != nil {
		t.Fatal(err)
	}
	if err := discardPending(b); err != nil {
		t.Fatal(err)
	}
	if ents, _ := os.ReadDir(filepath.Join(theirs, "trash")); len(ents) != 1 || ents[0].Name() != filepath.Base(b) {
		t.Fatalf("their trash = %v", ents)
	}
	if _, err := os.Stat(filepath.Join(mine, "trash")); !os.IsNotExist(err) {
		t.Fatalf("my trash was touched: %v", err)
	}
	if err := discardPending(b); err != nil {
		t.Fatalf("a moved pending directory is not moved again: %v", err)
	}
	c, _ := makePending(mine, key)
	moved, err := sweepPending(mine, key, a)
	if err != nil || len(moved) != 1 || moved[0] != c {
		t.Fatalf("sweep = %v, %v", moved, err)
	}
	if _, err := os.Stat(a); err != nil {
		t.Fatalf("the kept directory went: %v", err)
	}
	if moved, err := sweepPending(t.TempDir(), key, ""); moved != nil || err != nil {
		t.Fatalf("an empty scratch = %v, %v", moved, err)
	}
}

// 고정 manifest 의 sha256 은 보통 파일일 때만 읽는다 — symlink 가 가리키는 호스트 파일의 내용을 읽지 않고, FIFO
// 에서 멈추지 않고, 상한을 넘으면 거절한다. 모두 제한 시간 안에 돌아온다 (계획 4.1 30번).
func TestReadPinned(t *testing.T) {
	dir := t.TempDir()
	plain := filepath.Join(dir, "plain.xml")
	if err := os.WriteFile(plain, []byte("<manifest/>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("<manifest/>\n"))
	if got, err := readPinned(plain); err != nil || got != hex.EncodeToString(sum[:]) {
		t.Fatalf("readPinned = %q, %v", got, err)
	}
	host := filepath.Join(t.TempDir(), "host-secret")
	if err := os.WriteFile(host, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.xml")
	if err := os.Symlink(host, link); err != nil {
		t.Fatal(err)
	}
	fifo := filepath.Join(dir, "fifo.xml")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	sparse := filepath.Join(dir, "big.xml")
	f, err := os.Create(sparse)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(pinnedMax + 1); err != nil {
		t.Fatal(err)
	}
	f.Close()
	for p, want := range map[string]string{
		link: "not a regular file", fifo: "not a regular file", sparse: "larger than",
		filepath.Join(dir, "missing.xml"): "no such file",
	} {
		var done atomic.Bool
		var got string
		var gotErr error
		go func() {
			got, gotErr = readPinned(p)
			done.Store(true)
		}()
		waitFor(t, done.Load, "readPinned did not return for "+p)
		if gotErr == nil || !strings.HasPrefix(gotErr.Error(), "cannot pin the manifest: ") ||
			!strings.Contains(gotErr.Error(), want) || got != "" {
			t.Fatalf("readPinned(%s) = %q, %v; want %q", filepath.Base(p), got, gotErr, want)
		}
	}
}

// 파일 일의 실패 갈래 — 쓸 자리가 없는 초안 · JSON 이 아닌 초안 · 옮길 수 없는 버려진 자리와 읽을 수 없는 키 자리.
func TestBakefile_Failures(t *testing.T) {
	if err := writeDraft(filepath.Join(t.TempDir(), "missing"), fullDraft()); err == nil ||
		!strings.HasPrefix(err.Error(), "cannot write the bake draft: ") {
		t.Fatalf("writeDraft into nothing = %v", err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, draftName), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readDraft(dir, os.Getuid()); err == nil || !strings.HasPrefix(err.Error(), "cannot read the bake draft: ") {
		t.Fatalf("readDraft of garbage = %v", err)
	}
	scratchDir := t.TempDir()
	stuck, err := makePending(scratchDir, "1-2")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scratchDir, "trash"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if moved, err := sweepPending(scratchDir, "1-2", ""); err == nil || len(moved) != 0 {
		t.Fatalf("sweep with a blocked trash = %v, %v", moved, err)
	}
	if _, err := os.Stat(stuck); err != nil {
		t.Fatalf("the stuck directory went: %v", err)
	}
	fileKey := t.TempDir()
	if err := os.MkdirAll(filepath.Join(fileKey, pendingDirName), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fileKey, pendingDirName, "3-4"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := sweepPending(fileKey, "3-4", ""); err == nil {
		t.Fatal("a key that is a file swept clean")
	}
	if _, err := makePending(fileKey, "3-4"); err == nil || !strings.HasPrefix(err.Error(), "cannot create the pending upper directory: ") {
		t.Fatalf("makePending over a file = %v", err)
	}
}

// Lstat 과 Open 사이에 바뀐 파일을 읽지 않는다 — symlink 로 바꿔 끼우면 가리키는 호스트 파일의 내용을 읽지 않고,
// 같은 파일이 상한 밖으로 자라면 거절한다 (계획 4.1 7 · 30번). 틈은 afterLstat 이 연다.
func TestOpenPlain_AFileChangedAfterTheCheck(t *testing.T) {
	swapIn := func(t *testing.T, f func(p string)) {
		was := afterLstat
		afterLstat = f
		t.Cleanup(func() { afterLstat = was })
	}
	host := filepath.Join(t.TempDir(), "host-secret")
	if err := os.WriteFile(host, []byte(`{"schema":1,"run":"host"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	toLink := func(p string) {
		if err := os.Remove(p); err != nil {
			t.Error(err)
		}
		if err := os.Symlink(host, p); err != nil {
			t.Error(err)
		}
	}
	grow := func(size int64) func(string) {
		return func(p string) {
			if err := os.Truncate(p, size); err != nil {
				t.Error(err)
			}
		}
	}
	draftDir := func(t *testing.T) string {
		dir := t.TempDir()
		if err := writeDraft(dir, fullDraft()); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	t.Run("a draft swapped for a symlink", func(t *testing.T) {
		dir := draftDir(t)
		swapIn(t, toLink)
		d, err := readDraft(dir, os.Getuid())
		if err == nil || !strings.Contains(err.Error(), "changed while it was opened") || d.Run != "" {
			t.Fatalf("readDraft = %+v, %v", d, err)
		}
	})
	t.Run("a draft that grew", func(t *testing.T) {
		dir := draftDir(t)
		swapIn(t, grow(draftMax+10))
		if _, err := readDraft(dir, os.Getuid()); err == nil || !strings.Contains(err.Error(), "larger than") {
			t.Fatalf("readDraft = %v", err)
		}
	})
	pinned := func(t *testing.T) string {
		p := filepath.Join(t.TempDir(), pinnedFile)
		if err := os.WriteFile(p, []byte("<manifest/>\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	t.Run("a pinned manifest swapped for a symlink", func(t *testing.T) {
		p := pinned(t)
		swapIn(t, toLink)
		if got, err := readPinned(p); err == nil || !strings.Contains(err.Error(), "changed while it was opened") || got != "" {
			t.Fatalf("readPinned = %q, %v", got, err)
		}
	})
	t.Run("a pinned manifest that grew", func(t *testing.T) {
		p := pinned(t)
		swapIn(t, grow(pinnedMax+1))
		if got, err := readPinned(p); err == nil || !strings.Contains(err.Error(), "larger than") || got != "" {
			t.Fatalf("readPinned = %q, %v", got, err)
		}
	})
}

// 열거나 만들 수 없는 자리 — Lstat 뒤에 사라진 초안 (afterLstat 이 지운다 · root 여도 같은 갈래) · 이름을 만들 수
// 없는 키 자리 (0500). root 는 모드를 넘으므로 키 자리의 확인만 없다 — 건너뛰지 않는다 (CI 의 스킵 감시 · 5절 규칙 10).
func TestBakefile_OpenAndCreateFailures(t *testing.T) {
	dir := t.TempDir()
	if err := writeDraft(dir, fullDraft()); err != nil {
		t.Fatal(err)
	}
	was := afterLstat
	afterLstat = func(p string) {
		if err := os.Remove(p); err != nil {
			t.Error(err)
		}
	}
	_, err := readDraft(dir, os.Getuid())
	afterLstat = was
	if err == nil || !strings.HasPrefix(err.Error(), "cannot read the bake draft: ") || !strings.Contains(err.Error(), "no such file") {
		t.Fatalf("readDraft of a draft that went = %v", err)
	}
	if os.Geteuid() == 0 {
		t.Log("root creates past the modes; the closed key directory is not checked")
		return
	}
	scratchDir := t.TempDir()
	keyDir := filepath.Join(scratchDir, pendingDirName, "5-6")
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(keyDir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(keyDir, 0o700) })
	if _, err := makePending(scratchDir, "5-6"); err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("makePending under a closed key = %v", err)
	}
}
