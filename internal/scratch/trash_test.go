package scratch

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func write(t *testing.T, path, body string) {
	t.Helper()
	mkdir(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// 옮기기는 rename 한 번이다 — 옮긴 뒤 같은 inode 이고 원래 자리에 없다 (business-rules.md 1절).
func TestMoveRenamesOnceAndCreatesTheTrash(t *testing.T) {
	root := t.TempDir()
	run := filepath.Join(root, "enode-runc-1")
	write(t, filepath.Join(run, "upper", "big"), "data")
	before, err := os.Stat(run)
	if err != nil {
		t.Fatal(err)
	}
	trash := TrashIn(root)
	name, err := trash.Move(run)
	if err != nil {
		t.Fatal(err)
	}
	if name != "enode-runc-1" {
		t.Fatalf("name = %q", name)
	}
	if _, err := os.Stat(run); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("the work folder is still in place: %v", err)
	}
	after, err := os.Stat(filepath.Join(root, "trash", name))
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) {
		t.Fatal("the move was not a rename of the same inode")
	}
	fi, err := os.Stat(trash.Dir)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o700 {
		t.Fatalf("trash mode = %v, want 0700", fi.Mode().Perm())
	}
	if b, err := os.ReadFile(filepath.Join(trash.Dir, name, "upper", "big")); err != nil || string(b) != "data" {
		t.Fatalf("content did not travel: %q %v", b, err)
	}
}

// 이름이 겹치면 -1 · -2 를 붙인다.
func TestMoveNumbersAClashingName(t *testing.T) {
	root := t.TempDir()
	trash := TrashIn(root)
	var got []string
	for i := 0; i < 3; i++ {
		run := filepath.Join(root, "enode-runc-x")
		write(t, filepath.Join(run, "f"), "x")
		name, err := trash.Move(run)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, name)
	}
	if want := []string{"enode-runc-x", "enode-runc-x-1", "enode-runc-x-2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("names = %v, want %v", got, want)
	}
	entries, err := trash.Entries()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"enode-runc-x", "enode-runc-x-1", "enode-runc-x-2"}; !reflect.DeepEqual(entries, want) {
		t.Fatalf("entries = %v, want %v", entries, want)
	}
}

// 없는 경로는 오류이고 아무것도 지우지 않는다 — 지우기로 떨어지지 않는다.
func TestMoveFailsWithoutRemovingAnything(t *testing.T) {
	root := t.TempDir()
	trash := TrashIn(root)
	if _, err := trash.Move(filepath.Join(root, "missing")); err == nil {
		t.Fatal("moved a path that does not exist")
	}
	// trash 를 만들 수 없는 자리 — 파일이 그 이름을 쥐고 있다
	blocked := Trash{Dir: filepath.Join(root, "file", "trash")}
	write(t, filepath.Join(root, "file"), "x")
	run := filepath.Join(root, "enode-runc-2")
	write(t, filepath.Join(run, "keep"), "x")
	if _, err := blocked.Move(run); err == nil || !strings.Contains(err.Error(), "create trash") {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(run, "keep")); err != nil {
		t.Fatalf("a failed move removed the folder: %v", err)
	}
	// 옮길 자리에 다른 종류가 있다 — 이름 검사를 지나도 rename 이 실패하면 그대로 돌려준다
	odd := Trash{Dir: filepath.Join(root, "odd")}
	mkdir(t, odd.Dir)
	if err := os.Chmod(odd.Dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(odd.Dir, 0o700) })
	if os.Geteuid() != 0 {
		if _, err := odd.Move(run); err == nil {
			t.Fatal("moved into a trash it cannot write")
		}
		if _, err := os.Stat(filepath.Join(run, "keep")); err != nil {
			t.Fatalf("a failed move removed the folder: %v", err)
		}
	}
}

func TestEntriesOfAMissingTrashIsEmpty(t *testing.T) {
	names, err := TrashIn(t.TempDir()).Entries()
	if err != nil || len(names) != 0 {
		t.Fatalf("names = %v err = %v", names, err)
	}
	file := filepath.Join(t.TempDir(), "f")
	write(t, file, "x")
	if _, err := (Trash{Dir: file}).Entries(); err == nil {
		t.Fatal("listed a trash that is a file")
	}
}

// helper 는 trash 의 바로 아래 이름 하나만 받는다 (business-rules.md 5절).
func TestCheckEntryRejectsNamesAndTrashes(t *testing.T) {
	root := t.TempDir()
	trash := filepath.Join(root, "trash")
	mkdir(t, trash)
	for _, name := range []string{"", ".", "..", "a/b", "../x", `a\b`} {
		err := CheckEntry(trash, name)
		if err == nil || !strings.Contains(err.Error(), "invalid trash entry name") {
			t.Errorf("CheckEntry(%q) = %v", name, err)
		}
	}
	if err := CheckEntry(trash, "enode-runc-1"); err != nil {
		t.Fatalf("a plain name was rejected: %v", err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(trash, link); err != nil {
		t.Fatal(err)
	}
	if err := CheckEntry(link, "x"); err == nil || !strings.Contains(err.Error(), "trash is not a plain directory") {
		t.Fatalf("a symlinked trash was accepted: %v", err)
	}
	file := filepath.Join(root, "file")
	write(t, file, "x")
	if err := CheckEntry(file, "x"); err == nil || !strings.Contains(err.Error(), "trash is not a plain directory") {
		t.Fatalf("a file trash was accepted: %v", err)
	}
	if err := CheckEntry(filepath.Join(root, "missing"), "x"); err == nil {
		t.Fatal("a missing trash was accepted")
	}
}
