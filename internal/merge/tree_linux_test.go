//go:build linux

package merge

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

// 시험 도우미 — 가짜 트리를 짓고, 트리를 목록으로 읽고, 모형(view)으로 기대값을 낸다
// (business-logic-model.md 4.1 · domain-entities.md 6절).

// fixture 는 시험 하나의 폴더다. lower · upper · trash 와, lower 밖을 가리키는 symlink 의
// 과녁인 outside 를 둔다. 넷 다 같은 filesystem 이다.
type fixture struct {
	t       testing.TB
	base    string
	p       Paths
	outside string
	trash   *trashRec
}

func newFixture(t testing.TB) *fixture {
	t.Helper()
	base := t.TempDir()
	f := &fixture{t: t, base: base, p: Paths{
		Upper: filepath.Join(base, "upper"),
		Lower: filepath.Join(base, "lower"),
		Trash: filepath.Join(base, "trash"),
	}, outside: filepath.Join(base, "outside")}
	for _, d := range []string{f.p.Upper, f.p.Lower, f.p.Trash, f.outside} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	f.trash = &trashRec{dir: f.p.Trash}
	return f
}

func (f *fixture) at(side, rel string) string {
	root := map[string]string{"upper": f.p.Upper, "lower": f.p.Lower, "outside": f.outside}[side]
	if root == "" {
		f.t.Fatalf("unknown side %q", side)
	}
	return filepath.Join(root, filepath.FromSlash(rel))
}

func (f *fixture) must(err error) {
	f.t.Helper()
	if err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) dir(side, rel string, mode os.FileMode) {
	f.t.Helper()
	p := f.at(side, rel)
	f.must(os.Mkdir(p, 0o700))
	f.must(os.Chmod(p, mode)) // umask 와 무관하게
}

func (f *fixture) file(side, rel, content string) {
	f.t.Helper()
	f.must(os.WriteFile(f.at(side, rel), []byte(content), 0o644))
}

func (f *fixture) symlink(side, rel, target string) {
	f.t.Helper()
	f.must(os.Symlink(target, f.at(side, rel)))
}

func (f *fixture) hardlink(side, rel, target string) {
	f.t.Helper()
	f.must(os.Link(f.at(side, target), f.at(side, rel)))
}

// whiteout 은 커널이 만드는 것과 같은 문자 장치 0/0 이다. 특권 없이 만들어진다 (계획 1.4).
// 못 만들면 시험이 실패한다 — 건너뛰지 않는다.
func (f *fixture) whiteout(side, rel string) {
	f.t.Helper()
	if err := unix.Mknod(f.at(side, rel), unix.S_IFCHR, 0); err != nil {
		f.t.Fatalf("cannot make a fake whiteout (mknod c 0 0) without privilege: %v", err)
	}
}

// xattr 은 user xattr 하나를 붙인다. 못 붙이면 시험이 실패한다 — 건너뛰지 않는다.
func (f *fixture) xattr(side, rel, name, value string) {
	f.t.Helper()
	if err := unix.Lsetxattr(f.at(side, rel), name, []byte(value), 0); err != nil {
		f.t.Fatalf("cannot set %s on %s without privilege: %v", name, rel, err)
	}
}

func (f *fixture) opaque(side, rel string) { f.xattr(side, rel, xattrOpaque, "y") }

// stampDirs 는 두 트리의 디렉터리마다 다른 mtime 을 준다. 안에 항목을 다 만든 뒤에 부른다 —
// 항목을 만들면 부모의 mtime 이 바뀐다. upper 와 lower 가 같은 값을 갖지 않게 한다.
func (f *fixture) stampDirs() {
	f.t.Helper()
	sec := int64(1_000_000_000)
	for _, root := range []string{f.p.Lower, f.p.Upper} {
		f.must(filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil || !d.IsDir() || p == root {
				return err
			}
			sec += 1000
			ts := []unix.Timespec{unix.NsecToTimespec(sec * 1e9), unix.NsecToTimespec(sec*1e9 + 123)}
			return unix.UtimesNanoAt(unix.AT_FDCWD, p, ts, unix.AT_SYMLINK_NOFOLLOW)
		}))
	}
}

// mainFixture 는 표의 줄마다 하나 이상을 담은 가짜 트리다 (business-logic-model.md 4.1 ·
// 계획 3절의 escape).
func mainFixture(t testing.TB) *fixture {
	f := newFixture(t)
	f.file("outside", "secret.txt", "outside\n")

	// lower
	f.file("lower", "a.txt", "lower a\n")
	f.symlink("lower", "link", "a.txt")
	f.dir("lower", "d1", 0o755)
	f.file("lower", "d1/b.txt", "b\n")
	f.file("lower", "d1/c.txt", "c\n")
	f.dir("lower", "d1/sub", 0o755)
	f.file("lower", "d1/sub/s.txt", "s\n")
	f.dir("lower", "d2", 0o755)
	f.dir("lower", "d2/x", 0o755)
	f.file("lower", "d2/x/y.txt", "y\n")
	f.dir("lower", "d3", 0o755)
	f.dir("lower", "d3/deep", 0o755)
	f.file("lower", "d3/deep/q.txt", "q\n")
	f.file("lower", "f_to_dir", "f\n")
	f.file("lower", "f_to_plain", "f\n")
	f.dir("lower", "d_to_file", 0o755)
	f.file("lower", "d_to_file/inner", "i\n")
	f.symlink("lower", "linkdir", "d1")
	f.symlink("lower", "escape", f.outside)
	f.dir("lower", "keep", 0o755)
	f.file("lower", "keep/untouched.txt", "u\n")

	// upper
	f.file("upper", "a.txt", "upper a\n")
	f.file("upper", "new.txt", "n\n")
	f.symlink("upper", "link", "new.txt")
	f.dir("upper", "d1", 0o700)
	f.xattr("upper", "d1", "user.overlay.origin", "fh")
	f.xattr("upper", "d1", "user.overlay.impure", "y")
	f.whiteout("upper", "d1/c.txt")
	f.dir("upper", "d1/sub", 0o750)
	f.file("upper", "d1/sub/t.txt", "t\n")
	f.dir("upper", "d2", 0o755)
	f.opaque("upper", "d2")
	f.file("upper", "d2/z.txt", "z\n")
	f.whiteout("upper", "d3")
	f.dir("upper", "d3moved", 0o755)
	f.dir("upper", "d3moved/deep", 0o755)
	f.file("upper", "d3moved/deep/q.txt", "q\n")
	f.dir("upper", "f_to_dir", 0o755)
	f.opaque("upper", "f_to_dir")
	f.file("upper", "f_to_dir/g.txt", "g\n")
	f.dir("upper", "f_to_plain", 0o755)
	f.file("upper", "f_to_plain/h.txt", "h\n")
	f.file("upper", "d_to_file", "now a file\n")
	f.dir("upper", "linkdir", 0o755)
	f.file("upper", "linkdir/m.txt", "m\n")
	f.dir("upper", "escape", 0o755)
	f.file("upper", "escape/e.txt", "e\n")
	f.dir("upper", "brandnew", 0o755)
	f.dir("upper", "brandnew/sub", 0o755)
	f.file("upper", "brandnew/sub/k.txt", "k\n")
	f.whiteout("upper", "wh_absent")
	f.dir("upper", "opq_absent", 0o755)
	f.opaque("upper", "opq_absent")
	f.file("upper", "opq_absent/o.txt", "o\n")
	f.file("upper", "hl1", "hard\n")
	f.hardlink("upper", "hl2", "hl1")
	f.file("upper", "esc.txt", "escaped\n")
	f.xattr("upper", "esc.txt", "user.overlay.overlay.opaque", "y")
	f.file("upper", "origin.txt", "copied up\n")
	f.xattr("upper", "origin.txt", "user.overlay.origin", "fh")

	f.stampDirs()
	return f
}

// mainResult 는 mainFixture 를 한 번에 합쳤을 때의 Result 다. Ops 는 호출마다 센 수다 —
// 파일 대체 · 들임 8 · 새 디렉터리 2 · opaque 8 (3 + 3 + 2) · whiteout 5 (2 + 2 + 1) ·
// 종류가 바뀐 항목 8 (4 x 2) · 양쪽 디렉터리 10 (2 x 5).
var mainResult = Result{Ops: 41, Replaced: 2, Added: 6, NewDirs: 2, OpaqueDirs: 3, Whiteouts: 3,
	TypeChanged: 4, MergedDirs: 2, Discarded: 8}

// trashRec 은 시험의 Discard 다. 버릴 항목을 시험 trash 로 옮긴다 (이름은 순번). 재개로 다시
// 부를 때도 같은 것을 쓴다.
type trashRec struct {
	dir  string
	n    int
	fail int // 남은 실패 수 — 오류 시험이 쓴다
}

var errDiscard = fmt.Errorf("fake discard failure")

func (tr *trashRec) discard(path string) error {
	if tr.fail > 0 {
		tr.fail--
		return errDiscard
	}
	tr.n++
	return os.Rename(path, filepath.Join(tr.dir, strconv.Itoa(tr.n)))
}

// entry 는 목록의 한 줄이다. 디렉터리 크기 · atime · ctime 은 안 본다.
type entry struct {
	Path   string
	Type   byte // 'd' 'f' 'l' 'o'
	Mode   uint32
	UID    uint32
	GID    uint32
	Size   int64
	Mtime  int64
	Target string
	Ino    uint64 // 디렉터리가 아닌 항목만 — 옮겼지 복사하지 않았다는 증거
}

func (e entry) String() string {
	return fmt.Sprintf("%s %c %04o %d:%d size=%d mtime=%d target=%q ino=%d",
		e.Path, e.Type, e.Mode, e.UID, e.GID, e.Size, e.Mtime, e.Target, e.Ino)
}

// node 는 트리를 읽어 둔 한 항목이다.
type node struct {
	e        entry
	ino      uint64 // 디렉터리도 — 버린 모임을 댈 때 쓴다
	dir      bool
	whiteout bool
	opaque   bool
	children []string
}

// snapshot 은 뿌리 아래를 lstat 으로 읽는다. 뿌리는 "." 이다.
func snapshot(t testing.TB, root string) map[string]node {
	t.Helper()
	tree := map[string]node{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		var st unix.Stat_t
		if err := unix.Lstat(p, &st); err != nil {
			return err
		}
		n := node{ino: st.Ino, e: entry{Path: rel, Mode: st.Mode & 0o7777, UID: st.Uid, GID: st.Gid}}
		switch st.Mode & unix.S_IFMT {
		case unix.S_IFDIR:
			n.dir, n.e.Type, n.e.Mtime = true, 'd', st.Mtim.Nano()
			v := make([]byte, 8)
			if sz, err := unix.Lgetxattr(p, xattrOpaque, v); err == nil && string(v[:sz]) == "y" {
				n.opaque = true
			}
			ents, err := os.ReadDir(p)
			if err != nil {
				return err
			}
			for _, c := range ents {
				n.children = append(n.children, c.Name())
			}
		case unix.S_IFREG:
			n.e.Type, n.e.Size, n.e.Mtime, n.e.Ino = 'f', st.Size, st.Mtim.Nano(), st.Ino
		case unix.S_IFLNK:
			target, err := os.Readlink(p)
			if err != nil {
				return err
			}
			n.e.Type, n.e.Target, n.e.Ino = 'l', target, st.Ino
		default:
			n.e.Type, n.e.Ino = 'o', st.Ino
			n.whiteout = st.Mode&unix.S_IFMT == unix.S_IFCHR && st.Rdev == 0
		}
		tree[rel] = n
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

// listing 은 뿌리 아래의 목록이다 (뿌리 빼고 · 경로순).
func listing(t testing.TB, root string) []entry {
	t.Helper()
	var out []entry
	for rel, n := range snapshot(t, root) {
		if rel != "." {
			out = append(out, n.e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// view 는 합치기 전의 lower 와 upper 로 overlay 가 보일 모습(merged view)과, 가려져 버려질
// lower 항목의 inode 를 낸다. Apply 와 다른 길로 짠다 — 호출을 흉내 내지 않고 경로마다
// 「누가 보이나」를 정한다. decide 를 부르지 않는다.
//
//	upper 에 항목이 있으면 그것이 보인다.  whiteout 이면 아무것도 안 보인다
//	양쪽이 진짜 디렉터리이고 upper 가 opaque 가 아니면 안을 합쳐 보인다 (속성은 upper 의 것)
//	upper 에 없으면 lower 의 것이 보인다
//	보이지 않게 된 lower 항목 중, 같은 경로의 upper 파일에 덮여 사라지는 것(파일 위의 파일)
//	말고는 버려진다
func view(lower, upper map[string]node) (entries []entry, discarded []uint64) {
	var visit func(rel string, lowerOn, upperOn bool)
	visit = func(rel string, lowerOn, upperOn bool) {
		seen := map[string]bool{}
		var names []string
		add := func(ns []string) {
			for _, n := range ns {
				if !seen[n] {
					seen[n] = true
					names = append(names, n)
				}
			}
		}
		if upperOn {
			add(upper[rel].children)
		}
		if lowerOn {
			add(lower[rel].children)
		}
		sort.Strings(names)
		for _, name := range names {
			c := joinRel(rel, name)
			u, uok := upper[c]
			l, lok := lower[c]
			uok, lok = uok && upperOn, lok && lowerOn
			switch {
			case uok && u.whiteout:
				if lok {
					discarded = append(discarded, l.ino)
				}
			case uok && u.dir:
				entries = append(entries, u.e)
				both := lok && l.dir && !u.opaque
				if lok && !both {
					discarded = append(discarded, l.ino)
				}
				visit(c, both, true)
			case uok:
				entries = append(entries, u.e)
				if lok && l.dir {
					discarded = append(discarded, l.ino)
				}
			case lok:
				entries = append(entries, l.e)
				if l.dir {
					visit(c, true, false)
				}
			}
		}
	}
	visit(".", true, true)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	sort.Slice(discarded, func(i, j int) bool { return discarded[i] < discarded[j] })
	return entries, discarded
}

// trashInodes 는 시험 trash 바로 아래 항목의 inode 다.
func trashInodes(t testing.TB, dir string) []uint64 {
	t.Helper()
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []uint64
	for _, e := range ents {
		var st unix.Stat_t
		if err := unix.Lstat(filepath.Join(dir, e.Name()), &st); err != nil {
			t.Fatal(err)
		}
		out = append(out, st.Ino)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// diffEntries 는 두 목록이 다른 줄만 적는다. 같으면 빈 문자열이다.
func diffEntries(want, got []entry) string {
	w := map[string]entry{}
	for _, e := range want {
		w[e.Path] = e
	}
	g := map[string]entry{}
	for _, e := range got {
		g[e.Path] = e
	}
	var lines []string
	for _, e := range want {
		if ge, ok := g[e.Path]; !ok {
			lines = append(lines, "missing: "+e.String())
		} else if ge != e {
			lines = append(lines, "want:    "+e.String(), "got:     "+ge.String())
		}
	}
	for _, e := range got {
		if _, ok := w[e.Path]; !ok {
			lines = append(lines, "extra:   "+e.String())
		}
	}
	return strings.Join(lines, "\n")
}

// expectation 은 합치기 전에 읽어 둔 기대값이다.
type expectation struct {
	lower     []entry
	discarded []uint64
	outside   []entry
}

func expect(t testing.TB, f *fixture) expectation {
	t.Helper()
	lower, discarded := view(snapshot(t, f.p.Lower), snapshot(t, f.p.Upper))
	return expectation{lower: lower, discarded: discarded, outside: listing(t, f.outside)}
}

// verify 는 합친 뒤의 모습을 기대값에 댄다 — lower 목록 (디렉터리 mtime · inode 포함) ·
// 버린 모임 · 빈 upper · lower 밖이 그대로 · 재개 표지가 lower 에 없다.
func verify(t testing.TB, f *fixture, want expectation) {
	t.Helper()
	if d := diffEntries(want.lower, listing(t, f.p.Lower)); d != "" {
		t.Errorf("merged lower differs from the model:\n%s", d)
	}
	if got := trashInodes(t, f.p.Trash); fmt.Sprint(got) != fmt.Sprint(want.discarded) {
		t.Errorf("discarded inodes = %v; want %v", got, want.discarded)
	}
	if left := listing(t, f.p.Upper); len(left) != 0 {
		t.Errorf("upper is not empty: %v", left)
	}
	if d := diffEntries(want.outside, listing(t, f.outside)); d != "" {
		t.Errorf("the directory outside lower changed:\n%s", d)
	}
	for rel, n := range snapshot(t, f.p.Lower) {
		if !n.dir {
			continue
		}
		v := make([]byte, 64)
		if _, err := unix.Lgetxattr(filepath.Join(f.p.Lower, rel), stashXattr, v); err == nil {
			t.Errorf("lower %s carries %s", rel, stashXattr)
		}
	}
}
