package merge

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// 판정 표 시험 — 시스템 호출이 없어 모든 플랫폼에서 돈다.

const (
	modeFile = 0o100644
	modeLink = 0o120777
	modeFifo = 0o010644
)

func TestClassify(t *testing.T) {
	dir := uint32(modeDir | 0o755)
	chr := uint32(modeChr | 0o644)
	for _, tc := range []struct {
		name string
		rel  string
		mode uint32
		rdev uint64
		m    Marks
		want Kind
		mark string // 금지면 MarkError.Name
	}{
		{name: "file", rel: "a", mode: modeFile, want: KindOther},
		{name: "symlink", rel: "a", mode: modeLink, want: KindOther},
		{name: "fifo", rel: "a", mode: modeFifo, want: KindOther},
		{name: "device that is not 0/0", rel: "a", mode: chr, rdev: 0x0103, want: KindOther},
		{name: "whiteout", rel: "a", mode: chr, want: KindWhiteout},
		{name: "directory", rel: "d", mode: dir, want: KindDir},
		{name: "opaque directory", rel: "d", mode: dir, m: Marks{Opaque: []byte("y")}, want: KindOpaqueDir},
		{name: "root directory", rel: ".", mode: dir, want: KindDir},
		{name: "metacopy", rel: "a", mode: modeFile, m: Marks{Metacopy: true}, mark: xattrMetacopy},
		{name: "redirect", rel: "d", mode: dir, m: Marks{Redirect: true}, mark: xattrRedirect},
		{name: "xattr whiteout", rel: "a", mode: modeFile, m: Marks{Whiteout: true}, mark: xattrWhiteout},
		{name: "opaque x", rel: "d", mode: dir, m: Marks{Opaque: []byte("x")}, mark: xattrOpaque},
		{name: "opaque empty", rel: "d", mode: dir, m: Marks{Opaque: []byte{}}, mark: xattrOpaque},
		{name: "opaque on a file", rel: "a", mode: modeFile, m: Marks{Opaque: []byte("y")}, mark: xattrOpaque},
		{name: "opaque root", rel: ".", mode: dir, m: Marks{Opaque: []byte("y")}, mark: xattrOpaque},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := classify(tc.rel, tc.mode, tc.rdev, tc.m)
			if tc.mark != "" {
				var me *MarkError
				if !errors.As(err, &me) || me.Name != tc.mark || me.Path != tc.rel {
					t.Fatalf("classify = %v, %v; want a MarkError for %s at %s", got, err, tc.mark, tc.rel)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("classify = %v, %v; want %v", got, err, tc.want)
			}
		})
	}
}

func TestMarksFrom(t *testing.T) {
	names := splitNames([]byte("user.overlay.overlay.opaque\x00user.overlay.origin\x00user.overlay.opaque\x00" +
		"user.overlay.metacopy\x00user.overlay.redirect\x00user.overlay.whiteout\x00user.enode.merge-mtime\x00"))
	if len(names) != 7 {
		t.Fatalf("splitNames = %q; want 7 names", names)
	}
	m, err := marksFrom(names, func() ([]byte, error) { return []byte("y"), nil })
	if err != nil {
		t.Fatal(err)
	}
	want := Marks{Opaque: []byte("y"), Whiteout: true, Metacopy: true, Redirect: true}
	if !reflect.DeepEqual(m, want) {
		t.Fatalf("marksFrom = %+v; want %+v", m, want)
	}

	// escape 한 이름과 origin 은 표시가 아니다
	m, err = marksFrom(splitNames([]byte("user.overlay.overlay.opaque\x00user.overlay.origin\x00")),
		func() ([]byte, error) { t.Fatal("opaque value read without the opaque name"); return nil, nil })
	if err != nil || !reflect.DeepEqual(m, Marks{}) {
		t.Fatalf("marksFrom = %+v, %v; want no marks", m, err)
	}

	// 값을 못 읽으면 오류 · 빈 값은 nil 이 아니다
	boom := errors.New("boom")
	if _, err := marksFrom([]string{xattrOpaque}, func() ([]byte, error) { return nil, boom }); !errors.Is(err, boom) {
		t.Fatalf("marksFrom error = %v; want boom", err)
	}
	m, _ = marksFrom([]string{xattrOpaque}, func() ([]byte, error) { return nil, nil })
	if m.Opaque == nil || len(m.Opaque) != 0 {
		t.Fatalf("empty opaque value = %#v; want an empty, non-nil slice", m.Opaque)
	}
	if got := splitNames(nil); got != nil {
		t.Fatalf("splitNames(nil) = %q; want nil", got)
	}
}

func TestLowerKindOf(t *testing.T) {
	if lowerKindOf(modeDir|0o755) != LowerDir || lowerKindOf(modeLink) != LowerOther || lowerKindOf(modeFile) != LowerOther {
		t.Fatal("lowerKindOf: a directory is LowerDir; a symlink and a file are LowerOther")
	}
}

// TestDecide 는 business-rules.md 1절의 표다 — 종류 넷 x lower 셋.
func TestDecide(t *testing.T) {
	type row struct {
		k    Kind
		l    LowerKind
		want step
	}
	rename, unlink := CallRename, CallUnlink
	for _, r := range []row{
		{KindOther, LowerAbsent, step{last: rename, replace: true, tally: tallyAdded}},
		{KindOther, LowerOther, step{last: rename, replace: true, tally: tallyReplaced}},
		{KindOther, LowerDir, step{discard: true, last: rename, replace: true, tally: tallyTypeChanged}},
		{KindDir, LowerAbsent, step{last: rename, tally: tallyNewDir}},
		{KindDir, LowerDir, step{tally: tallyMergedDir}},
		{KindDir, LowerOther, step{discard: true, last: rename, tally: tallyTypeChanged}},
		{KindOpaqueDir, LowerAbsent, step{clear: true, last: rename, tally: tallyOpaqueDir}},
		{KindOpaqueDir, LowerDir, step{discard: true, clear: true, last: rename, tally: tallyOpaqueDir}},
		{KindOpaqueDir, LowerOther, step{discard: true, clear: true, last: rename, tally: tallyOpaqueDir}},
		{KindWhiteout, LowerAbsent, step{last: unlink, tally: tallyWhiteout}},
		{KindWhiteout, LowerDir, step{discard: true, last: unlink, tally: tallyWhiteout}},
		{KindWhiteout, LowerOther, step{discard: true, last: unlink, tally: tallyWhiteout}},
	} {
		if got := decide(r.k, r.l); got != r.want {
			t.Errorf("decide(%v, %v) = %+v; want %+v", r.k, r.l, got, r.want)
		}
	}
}

func TestCheckOverlap(t *testing.T) {
	for _, tc := range []struct {
		upper, lower, trash string
		overlap             bool
	}{
		{"/s/upper", "/w/lower", "/s/trash", false},
		{"/a/up", "/a/upper", "/a/trash", false}, // 앞부분만 같은 형제
		{"/w/lower", "/w/lower", "/s/trash", true},
		{"/w/lower/up", "/w/lower", "/s/trash", true},
		{"/s/upper", "/w", "/w/trash", true},
		{"/s/upper", "/", "/s/trash", true},
	} {
		err := checkOverlap(tc.upper, tc.lower, tc.trash)
		var pe *PreflightError
		if got := errors.As(err, &pe) && pe.Check == CheckOverlap; got != tc.overlap {
			t.Errorf("checkOverlap(%s, %s, %s) = %v; want overlap %v", tc.upper, tc.lower, tc.trash, err, tc.overlap)
		}
		if err != nil && !strings.Contains(err.Error(), " overlap") {
			t.Errorf("message %q does not say overlap", err)
		}
	}
}

func TestCheckDevices(t *testing.T) {
	if err := checkDevices(7, 7, 7); err != nil {
		t.Fatalf("same device: %v", err)
	}
	for _, d := range [][3]uint64{{7, 8, 7}, {7, 7, 8}, {8, 7, 7}} {
		err := checkDevices(d[0], d[1], d[2])
		var pe *PreflightError
		if !errors.As(err, &pe) || pe.Check != CheckFilesystem {
			t.Fatalf("checkDevices(%v) = %v; want a filesystem PreflightError", d, err)
		}
	}
}

func TestCallString(t *testing.T) {
	want := []string{"discard", "unlink", "clear-opaque", "rename", "stash-mtime", "chown", "chmod", "chtimes", "rmdir"}
	for i, w := range want {
		if got := Call(i + 1).String(); got != w {
			t.Errorf("Call(%d) = %q; want %q", i+1, got, w)
		}
	}
	if got := Call(0).String(); got != "call(0)" {
		t.Errorf("Call(0) = %q", got)
	}
	if got := Call(99).String(); got != "call(99)" {
		t.Errorf("Call(99) = %q", got)
	}
}

// TestErrorMessages 는 business-rules.md 10절의 문구다.
func TestErrorMessages(t *testing.T) {
	boom := errors.New("boom")
	for _, tc := range []struct {
		err  error
		want string
	}{
		{&PreflightError{Check: CheckPath, Path: "up", Err: errors.New("path is not absolute")},
			"merge preflight: up: path is not absolute"},
		{&PreflightError{Check: CheckDirectory, Path: "/s/trash", Err: errors.New("not a directory")},
			"merge preflight: /s/trash: not a directory"},
		{checkOverlap("/w/lower", "/w/lower", "/s/trash"), "merge preflight: /w/lower and /w/lower overlap"},
		{checkDevices(1, 2, 1),
			"merge preflight: upper, lower and trash must share one filesystem (upper dev 1, lower dev 2, trash dev 1)"},
		{&PreflightError{Check: CheckMark, Path: "a/b", Err: &MarkError{Path: "a/b", Name: xattrMetacopy}},
			"merge preflight: upper entry a/b carries user.overlay.metacopy"},
		{&MarkError{Path: "a", Name: xattrRedirect}, "upper entry a carries user.overlay.redirect"},
		{&MarkError{Path: "a", Name: xattrWhiteout},
			"upper entry a carries user.overlay.whiteout; only 0/0 character devices are read as whiteouts"},
		{&MarkError{Path: "d", Name: xattrOpaque, Value: []byte("x")},
			`upper entry d has user.overlay.opaque="x"; only "y" on a directory is read as opaque`},
		{&MarkError{Path: ".", Name: xattrOpaque, Value: []byte("y")},
			"upper entry . carries user.overlay.opaque; the upper root cannot be opaque"},
		{&OpError{Op: Op{Call: CallRename, Path: "d1/a.txt"}, Err: boom}, "merge: rename d1/a.txt: boom"},
		{errNoDiscard, "merge: no discard function"},
		{errNotEmpty, "merge: upper is not empty after the walk"},
		{ErrUnsupported, "merge is supported on linux only"},
	} {
		if got := tc.err.Error(); got != tc.want {
			t.Errorf("error = %q; want %q", got, tc.want)
		}
	}
	pe := &PreflightError{Check: CheckMark, Err: &MarkError{Path: "a", Name: xattrMetacopy}}
	var me *MarkError
	if !errors.As(pe, &me) {
		t.Error("PreflightError does not unwrap to its MarkError")
	}
	if !errors.Is(&OpError{Err: boom}, boom) {
		t.Error("OpError does not unwrap to its cause")
	}
}

func TestResultCount(t *testing.T) {
	var r Result
	for _, tl := range []tally{tallyNone, tallyReplaced, tallyAdded, tallyNewDir, tallyOpaqueDir,
		tallyWhiteout, tallyTypeChanged, tallyMergedDir} {
		r.count(tl)
	}
	want := Result{Replaced: 1, Added: 1, NewDirs: 1, OpaqueDirs: 1, Whiteouts: 1, TypeChanged: 1, MergedDirs: 1}
	if r != want {
		t.Fatalf("count = %+v; want %+v", r, want)
	}
}

func TestJoinRel(t *testing.T) {
	if joinRel(".", "a") != "a" || joinRel("a", "b") != "a/b" {
		t.Fatal("joinRel: the root has no prefix; below it names join with /")
	}
}
