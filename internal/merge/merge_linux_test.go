//go:build linux

package merge

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"golang.org/x/sys/unix"
)

var errStop = errors.New("stop here")

// TestApplyOnce 는 가짜 트리를 한 번에 합친다 (business-logic-model.md 4.1 「한 번에」).
func TestApplyOnce(t *testing.T) {
	f := mainFixture(t)
	want := expect(t, f)
	calls := 0
	res, err := Apply(context.Background(), f.p, Options{
		Discard: f.trash.discard,
		OnOp:    func(Op) error { calls++; return nil },
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if res != mainResult {
		t.Errorf("Result = %+v; want %+v", res, mainResult)
	}
	if calls != res.Ops {
		t.Errorf("OnOp was called %d times; Result.Ops = %d", calls, res.Ops)
	}
	verify(t, f, want)

	// 종류가 바뀐 항목 · 속성 · 옮겨 간 xattr (business-rules.md 1 · 2 · 5절)
	for rel, mode := range map[string]uint32{"escape": 0o755, "linkdir": 0o755, "d1": 0o700, "d1/sub": 0o750} {
		var st unix.Stat_t
		if err := unix.Lstat(filepath.Join(f.p.Lower, rel), &st); err != nil {
			t.Fatal(err)
		}
		if st.Mode&unix.S_IFMT != unix.S_IFDIR || st.Mode&0o7777 != mode {
			t.Errorf("lower %s mode = %o; want a directory with %o", rel, st.Mode, mode)
		}
	}
	for rel, name := range map[string]string{"esc.txt": "user.overlay.overlay.opaque", "origin.txt": "user.overlay.origin"} {
		v := make([]byte, 16)
		if _, err := unix.Lgetxattr(filepath.Join(f.p.Lower, rel), name, v); err != nil {
			t.Errorf("lower %s lost %s: %v", rel, name, err)
		}
	}
	for _, rel := range []string{"d2", "f_to_dir", "opq_absent"} {
		v := make([]byte, 8)
		if _, err := unix.Lgetxattr(filepath.Join(f.p.Lower, rel), xattrOpaque, v); !errors.Is(err, unix.ENODATA) {
			t.Errorf("lower %s still carries %s (err %v)", rel, xattrOpaque, err)
		}
	}

	// 다시 부르면 할 일이 없다
	res, err = Apply(context.Background(), f.p, Options{Discard: f.trash.discard})
	if err != nil || res != (Result{}) {
		t.Fatalf("second Apply = %+v, %v; want nothing to do", res, err)
	}
}

// TestApplyResume 는 1번째부터 마지막 호출까지 모든 호출 뒤에서 끊고 다시 부른다 (답 2 · 조각 7 의
// 기계 부분). 결과는 매번 모형과 같아야 한다 — 디렉터리 mtime 과 inode 까지.
func TestApplyResume(t *testing.T) {
	n := mainResult.Ops
	for k := 1; k <= n; k++ {
		t.Run(fmt.Sprintf("after-%02d", k), func(t *testing.T) {
			f := mainFixture(t)
			want := expect(t, f)
			calls := 0
			var last Op
			first, err := Apply(context.Background(), f.p, Options{
				Discard: f.trash.discard,
				OnOp: func(op Op) error {
					calls++
					if calls == k {
						last = op
						return errStop
					}
					return nil
				},
			})
			if !errors.Is(err, errStop) {
				t.Fatalf("interrupted Apply = %v; want errStop after call %d", err, k)
			}
			if first.Ops != k {
				t.Fatalf("interrupted Apply counted %d calls; want %d (stopped after %v)", first.Ops, k, last)
			}
			rest, err := Apply(context.Background(), f.p, Options{Discard: f.trash.discard})
			if err != nil {
				t.Fatalf("resumed Apply after %v: %v", last, err)
			}
			if first.Discarded+rest.Discarded != mainResult.Discarded {
				t.Errorf("discarded %d + %d; want %d", first.Discarded, rest.Discarded, mainResult.Discarded)
			}
			verify(t, f, want)
		})
	}
}

func TestApplyContext(t *testing.T) {
	t.Run("cancelled between calls", func(t *testing.T) {
		f := mainFixture(t)
		want := expect(t, f)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		calls := 0
		_, err := Apply(ctx, f.p, Options{Discard: f.trash.discard, OnOp: func(Op) error {
			if calls++; calls == 20 {
				cancel()
			}
			return nil
		}})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Apply = %v; want context.Canceled", err)
		}
		if _, err := Apply(context.Background(), f.p, Options{Discard: f.trash.discard}); err != nil {
			t.Fatal(err)
		}
		verify(t, f, want)
	})
	t.Run("cancelled before the walk", func(t *testing.T) {
		f := mainFixture(t)
		before := listing(t, f.p.Upper)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := Apply(ctx, f.p, Options{Discard: f.trash.discard}); !errors.Is(err, context.Canceled) {
			t.Fatalf("Apply = %v; want context.Canceled", err)
		}
		if d := diffEntries(before, listing(t, f.p.Upper)); d != "" {
			t.Fatalf("upper changed:\n%s", d)
		}
	})
}

// TestApplyDiscardError 는 Discard 가 실패하면 그 자리에서 멈추고, 다시 부르면 잇는다.
func TestApplyDiscardError(t *testing.T) {
	f := mainFixture(t)
	want := expect(t, f)
	f.trash.fail = 1
	res, err := Apply(context.Background(), f.p, Options{Discard: f.trash.discard})
	var oe *OpError
	if !errors.As(err, &oe) || !errors.Is(err, errDiscard) {
		t.Fatalf("Apply = %v; want an OpError around the discard failure", err)
	}
	// 이름순으로 처음 버리는 것은 d1 안의 whiteout 이다 — 그 앞에 d1 의 stash-mtime 하나와 파일 들임
	if oe.Op != (Op{Call: CallDiscard, Path: "d1/c.txt"}) {
		t.Errorf("failed op = %+v; want discard d1/c.txt", oe.Op)
	}
	if res.Discarded != 0 || res.Ops == 0 {
		t.Errorf("Result = %+v; want the calls before the failure and no discard", res)
	}
	if _, err := Apply(context.Background(), f.p, Options{Discard: f.trash.discard}); err != nil {
		t.Fatal(err)
	}
	verify(t, f, want)
}

func TestApplyNoDiscard(t *testing.T) {
	f := mainFixture(t)
	before := listing(t, f.p.Upper)
	if _, err := Apply(context.Background(), f.p, Options{}); !errors.Is(err, errNoDiscard) {
		t.Fatalf("Apply = %v; want errNoDiscard", err)
	}
	if d := diffEntries(before, listing(t, f.p.Upper)); d != "" {
		t.Fatalf("upper changed:\n%s", d)
	}
}

// TestApplyStopsAtMark 는 Preflight 를 건너뛰고 불러도 금지 표시 앞에서 멈추는지 본다.
func TestApplyStopsAtMark(t *testing.T) {
	f := newFixture(t)
	f.file("upper", "a.txt", "a\n")
	f.file("upper", "m.txt", "m\n")
	f.xattr("upper", "m.txt", xattrMetacopy, "")
	f.file("upper", "z.txt", "z\n")
	_, err := Apply(context.Background(), f.p, Options{Discard: f.trash.discard})
	var me *MarkError
	if !errors.As(err, &me) || me.Path != "m.txt" || me.Name != xattrMetacopy {
		t.Fatalf("Apply = %v; want a MarkError for m.txt", err)
	}
	for rel, where := range map[string]string{"a.txt": f.p.Lower, "m.txt": f.p.Upper, "z.txt": f.p.Upper} {
		if _, err := os.Lstat(filepath.Join(where, rel)); err != nil {
			t.Errorf("%s is not in %s: %v", rel, where, err)
		}
	}

	f = newFixture(t)
	f.opaque("upper", ".")
	if _, err := Apply(context.Background(), f.p, Options{Discard: f.trash.discard}); !errors.As(err, &me) || me.Path != "." {
		t.Fatalf("Apply on an opaque upper root = %v; want a MarkError for .", err)
	}
}

// TestApplyNoReplace 는 디렉터리를 옮기는 rename 이 덮어쓰지 않는지 본다 — 판정은 「없다」였는데
// 사이에 lower 에 빈 디렉터리가 생기면 조용히 대체하지 않고 멈춘다.
func TestApplyNoReplace(t *testing.T) {
	f := newFixture(t)
	f.dir("upper", "opq", 0o755)
	f.opaque("upper", "opq")
	f.file("upper", "opq/o.txt", "o\n")
	_, err := Apply(context.Background(), f.p, Options{Discard: f.trash.discard, OnOp: func(op Op) error {
		if op.Call == CallClearOpaque {
			return os.Mkdir(filepath.Join(f.p.Lower, "opq"), 0o755)
		}
		return nil
	}})
	var oe *OpError
	if !errors.As(err, &oe) || oe.Op.Call != CallRename || !errors.Is(err, unix.EEXIST) {
		t.Fatalf("Apply = %v; want a rename OpError with EEXIST", err)
	}
	if _, err := os.Lstat(filepath.Join(f.p.Upper, "opq", "o.txt")); err != nil {
		t.Fatalf("upper opq was moved: %v", err)
	}
}

// TestApplyNewDirOnce 는 새 디렉터리가 하위 트리가 커도 rename 한 번인지 본다 (requirements.md 5.4).
func TestApplyNewDirOnce(t *testing.T) {
	f := newFixture(t)
	f.dir("upper", "big", 0o755)
	for i := 0; i < 1000; i++ {
		f.file("upper", "big/"+strconv.Itoa(i), "x")
	}
	res, err := Apply(context.Background(), f.p, Options{Discard: f.trash.discard})
	if err != nil {
		t.Fatal(err)
	}
	if res != (Result{Ops: 1, NewDirs: 1}) {
		t.Fatalf("Result = %+v; want one rename", res)
	}
}

// TestApplyBadStash 는 재개 표지가 읽히지 않으면 멈추는지 본다.
func TestApplyBadStash(t *testing.T) {
	f := newFixture(t)
	f.dir("lower", "d", 0o755)
	f.dir("upper", "d", 0o755)
	f.xattr("upper", "d", stashXattr, "not a number")
	_, err := Apply(context.Background(), f.p, Options{Discard: f.trash.discard})
	if err == nil || err.Error() != `merge: read d: bad user.enode.merge-mtime "not a number"` {
		t.Fatalf("Apply = %v; want a bad stash error", err)
	}
}

// TestApplySymlinkedRoots 는 설정이 준 뿌리가 symlink 여도 풀어 쓰는지 본다.
func TestApplySymlinkedRoots(t *testing.T) {
	f := mainFixture(t)
	want := expect(t, f)
	links := Paths{}
	for _, l := range []struct {
		dst  *string
		name string
		to   string
	}{{&links.Upper, "u", f.p.Upper}, {&links.Lower, "l", f.p.Lower}, {&links.Trash, "t", f.p.Trash}} {
		*l.dst = filepath.Join(f.base, l.name)
		if err := os.Symlink(l.to, *l.dst); err != nil {
			t.Fatal(err)
		}
	}
	if err := Preflight(links); err != nil {
		t.Fatalf("Preflight: %v", err)
	}
	if _, err := Apply(context.Background(), links, Options{Discard: f.trash.discard}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	verify(t, f, want)
}

// TestPreflight 는 시작 전 확인의 줄마다 한 번씩 어긋나게 한다 (business-rules.md 3절).
func TestPreflight(t *testing.T) {
	t.Run("passes the main fixture and changes nothing", func(t *testing.T) {
		f := mainFixture(t)
		upper, lower := listing(t, f.p.Upper), listing(t, f.p.Lower)
		if err := Preflight(f.p); err != nil {
			t.Fatalf("Preflight: %v", err)
		}
		if d := diffEntries(upper, listing(t, f.p.Upper)) + diffEntries(lower, listing(t, f.p.Lower)); d != "" {
			t.Fatalf("Preflight changed the trees:\n%s", d)
		}
	})
	for _, tc := range []struct {
		name  string
		setup func(f *fixture) Paths
		check Check
		path  string
	}{
		{"relative path", func(f *fixture) Paths { p := f.p; p.Upper = "upper"; return p }, CheckPath, "upper"},
		{"missing trash", func(f *fixture) Paths { f.must(os.Remove(f.p.Trash)); return f.p }, CheckDirectory, ""},
		{"lower is a file", func(f *fixture) Paths {
			p := f.p
			p.Lower = filepath.Join(f.base, "file")
			f.must(os.WriteFile(p.Lower, nil, 0o644))
			return p
		}, CheckDirectory, ""},
		{"trash inside lower", func(f *fixture) Paths {
			p := f.p
			p.Trash = filepath.Join(f.p.Lower, "trash")
			f.must(os.Mkdir(p.Trash, 0o755))
			return p
		}, CheckOverlap, ""},
		{"metacopy", func(f *fixture) Paths {
			f.dir("upper", "d", 0o755)
			f.file("upper", "d/m", "m")
			f.xattr("upper", "d/m", xattrMetacopy, "")
			return f.p
		}, CheckMark, "d/m"},
		{"redirect", func(f *fixture) Paths {
			f.dir("upper", "r", 0o755)
			f.xattr("upper", "r", xattrRedirect, "/elsewhere")
			return f.p
		}, CheckMark, "r"},
		{"xattr whiteout", func(f *fixture) Paths {
			f.file("upper", "w", "")
			f.xattr("upper", "w", xattrWhiteout, "y")
			return f.p
		}, CheckMark, "w"},
		{"opaque x", func(f *fixture) Paths {
			f.dir("upper", "x", 0o755)
			f.xattr("upper", "x", xattrOpaque, "x")
			return f.p
		}, CheckMark, "x"},
		{"opaque on a file", func(f *fixture) Paths {
			f.file("upper", "o", "o")
			f.opaque("upper", "o")
			return f.p
		}, CheckMark, "o"},
		{"opaque root", func(f *fixture) Paths { f.opaque("upper", "."); return f.p }, CheckMark, "."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			err := Preflight(tc.setup(f))
			var pe *PreflightError
			if !errors.As(err, &pe) || pe.Check != tc.check {
				t.Fatalf("Preflight = %v; want a %s PreflightError", err, tc.check)
			}
			if tc.path != "" && pe.Path != tc.path {
				t.Fatalf("Preflight path = %q; want %q", pe.Path, tc.path)
			}
		})
	}
}

// BenchmarkPreflight 는 upper 15만 항목(디렉터리 100 x 파일 1,500)을 걷는 시간이다. ADR-077 §12 의
// 금지 xattr 사전 검사(147,893 파일 1.00 초)와 댄다 (계획 3절). 기본 go test 에서는 안 돈다.
func BenchmarkPreflight(b *testing.B) {
	f := newFixture(b)
	for d := 0; d < 100; d++ {
		dir := "d" + strconv.Itoa(d)
		f.dir("upper", dir, 0o755)
		for i := 0; i < 1500; i++ {
			f.file("upper", dir+"/"+strconv.Itoa(i), "")
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := Preflight(f.p); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkApply 는 ADR-077 §12 의 하루치 합치기(B)에 가까운 모양이다 — 양쪽 디렉터리 3,300 ·
// 파일 대체 52,800 · 새 파일 1,600 · 새 디렉터리 160 · opaque 200 · whiteout 330. §12 의
// 1.49 초와 댄다 (계획 3절). 반복마다 트리를 새로 짓고, 짓는 시간은 뺀다.
func BenchmarkApply(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		f := dailyFixture(b)
		b.StartTimer()
		res, err := Apply(context.Background(), f.p, Options{Discard: f.trash.discard})
		if err != nil {
			b.Fatal(err)
		}
		b.StopTimer()
		b.ReportMetric(float64(res.Ops), "ops")
		b.ReportMetric(float64(res.Replaced+res.Added+res.NewDirs+res.OpaqueDirs+res.Whiteouts+
			res.TypeChanged+res.MergedDirs), "entries")
		b.StartTimer()
	}
}

func dailyFixture(b *testing.B) *fixture {
	f := newFixture(b)
	both := func(rel string) {
		f.dir("lower", rel, 0o755)
		f.dir("upper", rel, 0o755)
	}
	for top := 0; top < 33; top++ {
		wd := "w" + strconv.Itoa(top)
		both(wd)
		for sub := 0; sub < 100; sub++ {
			d := wd + "/" + strconv.Itoa(sub)
			both(d)
			for i := 0; i < 16; i++ {
				name := d + "/f" + strconv.Itoa(i)
				f.file("lower", name, "old")
				f.file("upper", name, "new")
			}
			n := top*100 + sub
			switch {
			case n < 1600: // 새 파일
				f.file("upper", d+"/added", "a")
			}
			switch {
			case n < 160: // 새 디렉터리
				f.dir("upper", d+"/newdir", 0o755)
				f.file("upper", d+"/newdir/x", "x")
			case n < 360: // opaque
				f.dir("lower", d+"/opq", 0o755)
				f.file("lower", d+"/opq/old", "o")
				f.dir("upper", d+"/opq", 0o755)
				f.opaque("upper", d+"/opq")
				f.file("upper", d+"/opq/new", "n")
			case n < 690: // whiteout
				f.file("lower", d+"/gone", "g")
				f.whiteout("upper", d+"/gone")
			}
		}
	}
	return f
}
