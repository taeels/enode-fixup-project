//go:build integration && linux

package merge

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

// 사람이 SunnyVM 에서 도는 시험이다 (integration 태그 · CI 밖 · business-logic-model.md 4.2).
// 기본 go test 가 못 보는 셋을 본다 — 커널이 만든 진짜 표시, namespace 안의 권한, 읽기 전용
// 디렉터리. 버려도 되는 자리(t.TempDir)에서만 돈다. TMPDIR 를 ext4 에 두고 돌린다.
//
//	go test -c -tags integration -o merge.it ./internal/merge
//	TMPDIR=$HOME/merge-it-tmp ./merge.it -test.run TestMergeIntegration -test.v

const itChild = "ENODE_MERGE_IT_CHILD"

// TestMergeIntegration 은 자기 바이너리를 helper 와 같은 매핑의 namespace 에서 다시 실행한다.
// uid 매핑을 못 하면 실패한다 — 건너뛰지 않는다.
func TestMergeIntegration(t *testing.T) {
	if os.Getenv(itChild) == "" {
		unshare, err := exec.LookPath("unshare")
		if err != nil {
			t.Fatalf("unshare is required: %v", err)
		}
		cmd := exec.Command(unshare, "--user", "--map-root-user", "--map-auto", "--mount",
			os.Args[0], "-test.run=^TestMergeIntegration$", "-test.count=1", "-test.v")
		cmd.Env = append(os.Environ(), itChild+"=1")
		out, err := cmd.CombinedOutput()
		t.Logf("namespace child:\n%s", out)
		if err != nil {
			t.Fatalf("namespace child failed: %v", err)
		}
		return
	}
	if os.Getuid() != 0 {
		t.Fatalf("the child is not root in its namespace (uid %d)", os.Getuid())
	}

	t.Run("once", func(t *testing.T) {
		s := newScenario(t)
		want := expect(t, s.f)
		res, err := func() (Result, error) {
			if err := Preflight(s.f.p); err != nil {
				return Result{}, err
			}
			return Apply(context.Background(), s.f.p, Options{Discard: s.f.trash.discard})
		}()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("result %+v", res)
		verify(t, s.f, want)
		for _, rel := range []string{"ro", "rodeep", "rodeep/sub"} {
			var st unix.Stat_t
			if err := unix.Lstat(filepath.Join(s.f.p.Lower, rel), &st); err != nil || st.Mode&0o7777 != 0o555 {
				t.Errorf("lower %s mode = %o (%v); want 555 left as it was", rel, st.Mode&0o7777, err)
			}
		}
		s.nextOverlay(t)
		if !t.Failed() {
			t.Run("resume", func(t *testing.T) { resumeAll(t, res.Ops) })
		}
	})
}

// resumeAll 은 진짜 표시의 트리를 k = 1 .. n 마다 새로 짓고 끊고 다시 부른다.
func resumeAll(t *testing.T, n int) {
	for k := 1; k <= n; k++ {
		t.Run(fmt.Sprintf("after-%02d", k), func(t *testing.T) {
			s := newScenario(t)
			want := expect(t, s.f)
			calls := 0
			_, err := Apply(context.Background(), s.f.p, Options{Discard: s.f.trash.discard, OnOp: func(Op) error {
				if calls++; calls == k {
					return errStop
				}
				return nil
			}})
			if !errors.Is(err, errStop) {
				t.Fatalf("interrupted Apply = %v; want errStop", err)
			}
			if _, err := Apply(context.Background(), s.f.p, Options{Discard: s.f.trash.discard}); err != nil {
				t.Fatal(err)
			}
			verify(t, s.f, want)
		})
	}
}

type scenario struct {
	f      *fixture
	work   string
	merged string
}

// newScenario 는 lower 를 짓고, 진짜 overlay 세션 안에서 toy-test.sh 의 동작을 해 커널이 표시를
// 만들게 하고, 세션 안에서 본 모습(merged view)을 모형에 댄 뒤 마운트를 내린다.
func newScenario(t *testing.T) *scenario {
	t.Helper()
	f := newFixture(t)
	// 권한 000 · 0555 가 남은 폴더를 t.TempDir 이 지울 수 있게 푼다 — Cleanup 은 거꾸로 돈다
	t.Cleanup(func() { openUp(f.base) })
	s := &scenario{f: f, work: filepath.Join(f.base, "work"), merged: filepath.Join(f.base, "merged")}
	f.must(os.Mkdir(s.work, 0o755))
	f.must(os.Mkdir(s.merged, 0o755))

	f.file("lower", "a.txt", "lower a\n")
	f.dir("lower", "keep", 0o755)
	f.file("lower", "keep/untouched.txt", "u\n")
	f.dir("lower", "d1", 0o755)
	f.file("lower", "d1/b.txt", "b\n")
	f.file("lower", "d1/c.txt", "c\n")
	f.dir("lower", "d2", 0o755)
	f.dir("lower", "d2/x", 0o755)
	f.file("lower", "d2/x/y.txt", "y\n")
	f.dir("lower", "d3", 0o755)
	f.dir("lower", "d3/deep", 0o755)
	f.file("lower", "d3/deep/q.txt", "q\n")
	f.file("lower", "f_to_dir", "f\n")
	f.dir("lower", "d_to_file", 0o755)
	f.file("lower", "d_to_file/inner", "i\n")
	f.symlink("lower", "link", "a.txt")
	f.symlink("lower", "linkdir", "d1")
	f.dir("lower", "ro", 0o755)
	f.file("lower", "ro/old.txt", "old\n")
	f.must(os.Chmod(f.at("lower", "ro"), 0o555))
	f.dir("lower", "rodeep", 0o755)
	f.dir("lower", "rodeep/sub", 0o755)
	f.file("lower", "rodeep/sub/s.txt", "s\n")
	f.must(os.Chmod(f.at("lower", "rodeep/sub"), 0o555))
	f.must(os.Chmod(f.at("lower", "rodeep"), 0o555))
	f.dir("lower", "gone", 0o755)
	f.file("lower", "gone/g.txt", "g\n")
	f.must(os.Chmod(f.at("lower", "gone"), 0))

	opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s,userxattr", f.p.Lower, f.p.Upper, s.work)
	if err := unix.Mount("overlay", s.merged, "overlay", 0, opts); err != nil {
		t.Fatalf("mount overlay: %v", err)
	}
	m := func(rel string) string { return filepath.Join(s.merged, filepath.FromSlash(rel)) }
	for _, step := range []func() error{
		func() error { return appendFile(m("a.txt"), "more\n") },
		func() error { return os.WriteFile(m("new.txt"), []byte("n\n"), 0o644) },
		func() error { return os.Remove(m("d1/c.txt")) },
		func() error { return os.Chmod(m("d1"), 0o700) },
		func() error { return os.RemoveAll(m("d2")) },
		func() error { return os.Mkdir(m("d2"), 0o755) },
		func() error { return os.WriteFile(m("d2/z.txt"), []byte("z\n"), 0o644) },
		func() error { return os.RemoveAll(m("d3")) }, // mv d3 d3moved — redirect 없이 통째 복사 (EXDEV)
		func() error { return os.MkdirAll(m("d3moved/deep"), 0o755) },
		func() error { return os.WriteFile(m("d3moved/deep/q.txt"), []byte("q\n"), 0o644) },
		func() error { return os.Remove(m("f_to_dir")) },
		func() error { return os.Mkdir(m("f_to_dir"), 0o755) },
		func() error { return os.WriteFile(m("f_to_dir/g.txt"), []byte("g\n"), 0o644) },
		func() error { return os.RemoveAll(m("d_to_file")) },
		func() error { return os.WriteFile(m("d_to_file"), []byte("now a file\n"), 0o644) },
		func() error { return os.Remove(m("link")) },
		func() error { return os.Symlink("new.txt", m("link")) },
		func() error { return os.Remove(m("linkdir")) },
		func() error { return os.Mkdir(m("linkdir"), 0o755) },
		func() error { return os.WriteFile(m("linkdir/m.txt"), []byte("m\n"), 0o644) },
		func() error { return os.WriteFile(m("ro/new.txt"), []byte("in a read-only dir\n"), 0o644) },
		func() error { return os.WriteFile(m("rodeep/sub/t.txt"), []byte("t\n"), 0o644) },
		func() error { return os.RemoveAll(m("gone")) },
		func() error { return os.MkdirAll(m("brandnew/sub"), 0o755) },
		func() error { return os.WriteFile(m("brandnew/sub/k.txt"), []byte("k\n"), 0o644) },
	} {
		if err := step(); err != nil {
			_ = unix.Unmount(s.merged, 0)
			t.Fatalf("session step: %v", err)
		}
	}
	kernel := dropIno(listing(t, s.merged))
	if err := unix.Unmount(s.merged, 0); err != nil {
		t.Fatalf("unmount: %v", err)
	}
	model, _ := view(snapshot(t, f.p.Lower), snapshot(t, f.p.Upper))
	if d := diffEntries(dropIno(model), kernel); d != "" {
		t.Fatalf("the model differs from the kernel's merged view:\n%s", d)
	}
	return s
}

// nextOverlay 는 합친 lower 위에 새 overlay 를 올려 목록이 lower 와 같고 덧붙이기가 되는지 본다.
func (s *scenario) nextOverlay(t *testing.T) {
	t.Helper()
	up, work := filepath.Join(s.f.base, "upper2"), filepath.Join(s.f.base, "work2")
	s.f.must(os.Mkdir(up, 0o755))
	s.f.must(os.Mkdir(work, 0o755))
	opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s,userxattr", s.f.p.Lower, up, work)
	if err := unix.Mount("overlay", s.merged, "overlay", 0, opts); err != nil {
		t.Fatalf("mount the next overlay: %v", err)
	}
	defer func() { _ = unix.Unmount(s.merged, 0) }()
	if d := diffEntries(dropIno(listing(t, s.f.p.Lower)), dropIno(listing(t, s.merged))); d != "" {
		t.Errorf("the next overlay does not show the merged lower:\n%s", d)
	}
	if err := appendFile(filepath.Join(s.merged, "new.txt"), "again\n"); err != nil {
		t.Errorf("copy-up on the next overlay: %v", err)
	}
}

func appendFile(path, s string) error {
	fh, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return err
	}
	_, err = fh.WriteString(s)
	if cerr := fh.Close(); err == nil {
		err = cerr
	}
	return err
}

func dropIno(es []entry) []entry {
	out := make([]entry, len(es))
	for i, e := range es {
		e.Ino = 0
		out[i] = e
	}
	return out
}

// openUp 은 뿌리 아래 디렉터리를 모두 0700 으로 푼다 — 지우기 전에.
func openUp(root string) {
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if d != nil && d.IsDir() {
			_ = os.Chmod(p, 0o700)
		}
		return nil
	})
}
