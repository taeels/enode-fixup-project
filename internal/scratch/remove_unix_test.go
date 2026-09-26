//go:build unix

package scratch

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

// fakeTree 는 보통 권한으로 만들 수 있는 가짜 작업 폴더다 — overlay 의 모양을 흉내 낸다.
//
//	entry/upper/a.txt           파일
//	entry/upper/sub/b.bin       파일
//	entry/upper/escape          trash 밖의 파일을 가리키는 symlink
//	entry/work/work             권한 000 디렉터리 (overlay 가 남기는 모양)
//	entry/work/work/inner       그 안의 파일
func fakeTree(t *testing.T) (trash, entry, outside string) {
	t.Helper()
	root := t.TempDir()
	trash = filepath.Join(root, "trash")
	entry = "enode-runc-1"
	base := filepath.Join(trash, entry)
	write(t, filepath.Join(base, "upper", "a.txt"), strings.Repeat("a", 5000))
	write(t, filepath.Join(base, "upper", "sub", "b.bin"), "b")
	outside = filepath.Join(root, "outside.txt")
	write(t, outside, "keep me")
	if err := os.Symlink(outside, filepath.Join(base, "upper", "escape")); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(base, "work", "work", "inner"), "i")
	if err := os.Chmod(filepath.Join(base, "work", "work"), 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(base, "work", "work"), 0o700) })
	return trash, entry, outside
}

// 측정은 블록 수 x 512 의 합과 항목 수(디렉터리 포함)다. lstat 으로 센다.
func TestMeasureSumsBlocksAndCountsEntries(t *testing.T) {
	trash, entry, _ := fakeTree(t)
	// 기대값은 같은 규칙으로 따로 센다 — 000 디렉터리는 풀어 둔다
	_ = os.Chmod(filepath.Join(trash, entry, "work", "work"), 0o700)
	var want Size
	err := filepath.WalkDir(filepath.Join(trash, entry), func(p string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		var st unix.Stat_t
		if err := unix.Lstat(p, &st); err != nil {
			return err
		}
		want.Bytes += int64(st.Blocks) * 512
		want.Entries++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = os.Chmod(filepath.Join(trash, entry, "work", "work"), 0o000)
	got, err := Measure(trash, entry)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("measure = %+v, want %+v", got, want)
	}
	if got.Entries != 9 {
		t.Fatalf("entries = %d, want 9 (entry upper a.txt sub b.bin escape work work inner)", got.Entries)
	}
	if got.Bytes < 5000 {
		t.Fatalf("bytes = %d, want at least the 5000-byte file", got.Bytes)
	}
}

// 지우기는 symlink 를 안 따라가고 권한 000 디렉터리를 풀어 지운다.
func TestRemoveDeletesTheEntryAndNothingOutside(t *testing.T) {
	trash, entry, outside := fakeTree(t)
	left, err := Remove(trash, entry)
	if err != nil || left != nil {
		t.Fatalf("left = %v err = %v", left, err)
	}
	if _, err := os.Lstat(filepath.Join(trash, entry)); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("the entry survived: %v", err)
	}
	if b, err := os.ReadFile(outside); err != nil || string(b) != "keep me" {
		t.Fatalf("the symlink target outside trash was touched: %q %v", b, err)
	}
	if _, err := os.Stat(trash); err != nil {
		t.Fatalf("trash itself went away: %v", err)
	}
	// 없는 항목은 지울 것이 없다 — 오류가 아니다
	if left, err := Remove(trash, entry); err != nil || left != nil {
		t.Fatalf("second remove: left = %v err = %v", left, err)
	}
}

// symlink 인 항목은 링크만 지운다 — 가리키는 디렉터리 안으로 안 들어간다.
func TestRemoveOfASymlinkEntryRemovesOnlyTheLink(t *testing.T) {
	root := t.TempDir()
	trash := filepath.Join(root, "trash")
	target := filepath.Join(root, "target")
	write(t, filepath.Join(target, "precious"), "x")
	mkdir(t, trash)
	if err := os.Symlink(target, filepath.Join(trash, "link")); err != nil {
		t.Fatal(err)
	}
	size, err := Measure(trash, "link")
	if err != nil || size.Entries != 1 {
		t.Fatalf("measure followed the link: %+v %v", size, err)
	}
	if _, err := Remove(trash, "link"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(trash, "link")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("the link survived: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "precious")); err != nil {
		t.Fatalf("the link target was removed: %v", err)
	}
}

// trash 와 다른 filesystem 은 안 들어가고 left 에 남긴다. 시험은 다른 st_dev 를 만들 수
// 없으므로 판정 함수를 바꿔 끼운다. 그 부모는 비지 않은 채 남고, 그것은 오류가 아니다.
func TestRemoveLeavesAnotherFilesystemInPlace(t *testing.T) {
	trash, entry, _ := fakeTree(t)
	mount := entry + "/upper/sub"
	crosses := func(rel string, _ uint64) bool { return rel == mount }
	left, err := remove(trash, entry, crosses)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{mount}; !reflect.DeepEqual(left, want) {
		t.Fatalf("left = %v, want %v", left, want)
	}
	if _, err := os.Stat(filepath.Join(trash, entry, "upper", "sub", "b.bin")); err != nil {
		t.Fatalf("the other filesystem was entered: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(trash, entry, "upper", "a.txt")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("the rest was not removed: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(trash, entry, "work")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("a sibling tree was not removed: %v", err)
	}
	// 측정도 같은 경계다 — 남길 것은 안 센다
	w, fd, err := openTrash(trash, crosses)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = unix.Close(fd) }()
	w.measure(fd, entry, entry)
	if w.err != nil || w.size.Entries != 2 {
		t.Fatalf("measure across the boundary = %+v %v, want entry and upper", w.size, w.err)
	}
}

// 기본 판정은 trash 의 st_dev 와 같은가다 — 같은 filesystem 의 항목은 다 지운다.
func TestRemoveWithTheDefaultBoundaryStaysOnTheTrashDevice(t *testing.T) {
	trash, entry, _ := fakeTree(t)
	w, fd, err := openTrash(trash, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = unix.Close(fd) }()
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		t.Fatal(err)
	}
	if w.crosses(entry, uint64(st.Dev)) || !w.crosses(entry, uint64(st.Dev)+1) {
		t.Fatal("the default boundary is not the trash device")
	}
}

func TestMeasureAndRemoveRefuseASymlinkedTrash(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "real")
	write(t, filepath.Join(real, "x", "f"), "x")
	link := filepath.Join(root, "trash")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Measure(link, "x"); err == nil {
		t.Fatal("measured through a symlinked trash")
	}
	if _, err := Remove(link, "x"); err == nil {
		t.Fatal("removed through a symlinked trash")
	}
	if _, err := os.Stat(filepath.Join(real, "x", "f")); err != nil {
		t.Fatalf("the symlinked trash was emptied: %v", err)
	}
}

// 쓰기 권한이 모자란 디렉터리도 들어가기 전에 풀어 지운다 (0500 은 0700 이 아니다).
func TestRemoveUnlocksAReadOnlyDirectory(t *testing.T) {
	root := t.TempDir()
	trash := filepath.Join(root, "trash")
	write(t, filepath.Join(trash, "e", "a", "f"), "x")
	write(t, filepath.Join(trash, "e", "b", "g"), "x")
	if err := os.Chmod(filepath.Join(trash, "e", "a"), 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(trash, "e", "a"), 0o700) })
	if left, err := Remove(trash, "e"); err != nil || left != nil {
		t.Fatalf("left = %v err = %v", left, err)
	}
	if _, err := os.Lstat(filepath.Join(trash, "e")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("the entry survived: %v", err)
	}
}

// 지우지 못한 곳이 있으면 그 앞에서 지울 수 있는 것은 지우고 그 실패를 돌려준다. 항목은
// trash 에 남고 다음 깸에 다시 한다.
func TestRemoveReportsAFailureAfterRemovingWhatItCould(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores permissions")
	}
	trash := filepath.Join(t.TempDir(), "trash")
	write(t, filepath.Join(trash, "e2", "f"), "x")
	// trash 를 읽기 전용으로 두면 항목 안은 지워지고 항목 자체만 못 지운다
	if err := os.Chmod(trash, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(trash, 0o700) })
	left, err := Remove(trash, "e2")
	if err == nil || left != nil {
		t.Fatalf("left = %v err = %v", left, err)
	}
	var pe *fs.PathError
	if !errors.As(err, &pe) || pe.Path != "e2" || pe.Op != "remove" {
		t.Fatalf("err = %#v", err)
	}
	if _, err := os.Lstat(filepath.Join(trash, "e2", "f")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("the files inside were not removed before the failure: %v", err)
	}
}

// BenchmarkRemove 는 N1 (한 번에 지우는 양과 속도) 의 측정이다 — 디렉터리 100 x 파일 1,500 =
// 150,000 개를 측정하고 지운다. 기본 go test 에서는 안 돈다.
//
//	go test ./internal/scratch -run '^$' -bench BenchmarkRemove -benchtime 1x
func BenchmarkRemove(b *testing.B) {
	const dirs, files = 100, 1500
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		trash := filepath.Join(b.TempDir(), "trash")
		for d := 0; d < dirs; d++ {
			dir := filepath.Join(trash, "entry", fmt.Sprintf("d%03d", d))
			if err := os.MkdirAll(dir, 0o755); err != nil {
				b.Fatal(err)
			}
			for f := 0; f < files; f++ {
				if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%04d", f)), []byte("x"), 0o644); err != nil {
					b.Fatal(err)
				}
			}
		}
		b.StartTimer()
		size, err := Measure(trash, "entry")
		if err != nil {
			b.Fatal(err)
		}
		if _, err := Remove(trash, "entry"); err != nil {
			b.Fatal(err)
		}
		b.ReportMetric(float64(size.Entries), "entries/op")
	}
}
