//go:build linux

package merge

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"golang.org/x/sys/unix"
)

// helper 안(runtime 과 같은 uid 매핑의 user namespace)에서 돈다. upper 에 노드 uid 가 못 걷는
// 항목이 생길 수 있고, 권한 비트가 없는 디렉터리를 옮기는 일은 namespace 안의 root 만 한다.
// 권한 비트를 풀었다 되돌리지 않는다 — 그 사이에 끊기면 lower 에 풀린 권한이 남는다.

const openDirFlags = unix.O_RDONLY | unix.O_DIRECTORY | unix.O_NOFOLLOW | unix.O_CLOEXEC

// Preflight 는 시작 전 확인이다 (FR-7 · business-rules.md 3절). 하나라도 어긋나면
// *PreflightError 를 돌려주고, 아무것도 바꾸지 않는다. 걷다가 읽지 못한 것은 확인이 아니라
// 입출력 오류라 그대로 돌려준다.
//
// 「이 lower 의 overlay 마운트 0」은 보지 않는다 — 부르는 쪽이 쥔 lower 배타 잠금이 그 증거다.
func Preflight(p Paths) error {
	r, err := resolve(p)
	if err != nil {
		return err
	}
	fd, err := unix.Open(r.upper, openDirFlags, 0)
	if err != nil {
		return fmt.Errorf("merge preflight: open upper: %w", err)
	}
	defer func() { _ = unix.Close(fd) }()
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return fmt.Errorf("merge preflight: read .: %w", err)
	}
	m, err := marksOfDir(fd)
	if err != nil {
		return fmt.Errorf("merge preflight: read .: %w", err)
	}
	if _, err := classify(".", st.Mode, uint64(st.Rdev), m); err != nil {
		return &PreflightError{Check: CheckMark, Path: ".", Err: err}
	}
	return scan(fd, ".")
}

// roots 는 푼 세 뿌리다.
type roots struct {
	upper, lower, trash string
}

// resolve 는 시작 전 확인의 1 ~ 4 다 — 절대 경로 · 진짜 디렉터리 · 겹침 · 같은 filesystem.
func resolve(p Paths) (roots, error) {
	in := [...]string{p.Upper, p.Lower, p.Trash}
	var out [3]string
	var dev [3]uint64
	for i, path := range in {
		if !filepath.IsAbs(path) {
			return roots{}, &PreflightError{Check: CheckPath, Path: path, Err: errors.New("path is not absolute")}
		}
		real, err := filepath.EvalSymlinks(path)
		if errors.Is(err, fs.ErrNotExist) {
			return roots{}, &PreflightError{Check: CheckDirectory, Path: path, Err: err}
		}
		if err != nil {
			return roots{}, &PreflightError{Check: CheckPath, Path: path, Err: err}
		}
		var st unix.Stat_t
		if err := unix.Lstat(real, &st); err != nil {
			return roots{}, &PreflightError{Check: CheckDirectory, Path: path, Err: err}
		}
		if st.Mode&modeType != modeDir {
			return roots{}, &PreflightError{Check: CheckDirectory, Path: path, Err: errors.New("not a directory")}
		}
		out[i], dev[i] = real, uint64(st.Dev)
	}
	if err := checkOverlap(out[0], out[1], out[2]); err != nil {
		return roots{}, err
	}
	if err := checkDevices(dev[0], dev[1], dev[2]); err != nil {
		return roots{}, err
	}
	return roots{upper: out[0], lower: out[1], trash: out[2]}, nil
}

// scan 은 upper 를 fd 로 걸으며 금지 표시를 찾는다. 첫 금지에서 멈춘다.
func scan(dirfd int, rel string) error {
	names, err := readNames(dirfd)
	if err != nil {
		return fmt.Errorf("merge preflight: read %s: %w", rel, err)
	}
	for _, name := range names {
		child := joinRel(rel, name)
		var st unix.Stat_t
		if err := unix.Fstatat(dirfd, name, &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
			return fmt.Errorf("merge preflight: read %s: %w", child, err)
		}
		if st.Mode&modeType != modeDir {
			m, err := marksAt(dirfd, name)
			if err != nil {
				return fmt.Errorf("merge preflight: read %s: %w", child, err)
			}
			if _, err := classify(child, st.Mode, uint64(st.Rdev), m); err != nil {
				return &PreflightError{Check: CheckMark, Path: child, Err: err}
			}
			continue
		}
		if err := scanDir(dirfd, name, child, st.Mode); err != nil {
			return err
		}
	}
	return nil
}

func scanDir(parent int, name, rel string, mode uint32) error {
	fd, err := unix.Openat(parent, name, openDirFlags, 0)
	if err != nil {
		return fmt.Errorf("merge preflight: read %s: %w", rel, err)
	}
	defer func() { _ = unix.Close(fd) }()
	m, err := marksOfDir(fd)
	if err != nil {
		return fmt.Errorf("merge preflight: read %s: %w", rel, err)
	}
	if _, err := classify(rel, mode, 0, m); err != nil {
		return &PreflightError{Check: CheckMark, Path: rel, Err: err}
	}
	return scan(fd, rel)
}

// Apply 는 upper 를 lower 에 합친다 (business-rules.md 1 · 4 ~ 9절). 끊긴 뒤 다시 불러도
// 한 번에 끝낸 것과 같다. 첫 오류에서 멈추고, 그때까지 센 Result 를 오류와 함께 돌려준다.
// Preflight 를 부르지 않는다 — 부르는 쪽이 처음과 재개 때 모두 Preflight 뒤에 부른다. 걷다가
// 금지 표시를 만나면 그 항목에 손대기 전에 멈춘다.
func Apply(ctx context.Context, p Paths, opt Options) (Result, error) {
	if opt.Discard == nil {
		return Result{}, errNoDiscard
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	upper, err := filepath.EvalSymlinks(p.Upper)
	if err != nil {
		return Result{}, fmt.Errorf("merge: open upper: %w", err)
	}
	lower, err := filepath.EvalSymlinks(p.Lower)
	if err != nil {
		return Result{}, fmt.Errorf("merge: open lower: %w", err)
	}
	ufd, err := unix.Open(upper, openDirFlags, 0)
	if err != nil {
		return Result{}, fmt.Errorf("merge: open upper: %w", err)
	}
	defer func() { _ = unix.Close(ufd) }()
	lfd, err := unix.Open(lower, openDirFlags, 0)
	if err != nil {
		return Result{}, fmt.Errorf("merge: open lower: %w", err)
	}
	defer func() { _ = unix.Close(lfd) }()

	var st unix.Stat_t
	if err := unix.Fstat(ufd, &st); err != nil {
		return Result{}, fmt.Errorf("merge: read .: %w", err)
	}
	m, err := marksOfDir(ufd)
	if err != nil {
		return Result{}, fmt.Errorf("merge: read .: %w", err)
	}
	if _, err := classify(".", st.Mode, uint64(st.Rdev), m); err != nil {
		return Result{}, fmt.Errorf("merge: %w", err)
	}

	w := &walker{ctx: ctx, opt: opt, lower: lower}
	if err := w.walk(ufd, lfd, "."); err != nil {
		return w.res, err
	}
	names, err := readNames(ufd)
	if err != nil {
		return w.res, fmt.Errorf("merge: read .: %w", err)
	}
	if len(names) > 0 {
		return w.res, errNotEmpty
	}
	return w.res, nil
}

type walker struct {
	ctx   context.Context
	opt   Options
	lower string // 푼 lower 뿌리 — Discard 의 경로를 짓는다
	res   Result
}

// walk 는 디렉터리 하나 안의 upper 항목을 이름의 바이트 순으로 합친다. 같은 트리면 호출 차례가
// 같다.
func (w *walker) walk(ufd, lfd int, rel string) error {
	names, err := readNames(ufd)
	if err != nil {
		return fmt.Errorf("merge: read %s: %w", rel, err)
	}
	for _, name := range names {
		if err := w.item(ufd, lfd, name, joinRel(rel, name)); err != nil {
			return err
		}
	}
	return nil
}

// item 은 upper 항목 하나를 판정하고 표의 차례대로 호출한다.
func (w *walker) item(ufd, lfd int, name, rel string) error {
	var ust unix.Stat_t
	if err := unix.Fstatat(ufd, name, &ust, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return fmt.Errorf("merge: read %s: %w", rel, err)
	}
	dfd := -1
	var m Marks
	var err error
	if ust.Mode&modeType == modeDir {
		if dfd, err = unix.Openat(ufd, name, openDirFlags, 0); err != nil {
			return fmt.Errorf("merge: read %s: %w", rel, err)
		}
		defer func() { _ = unix.Close(dfd) }()
		m, err = marksOfDir(dfd)
	} else {
		m, err = marksAt(ufd, name)
	}
	if err != nil {
		return fmt.Errorf("merge: read %s: %w", rel, err)
	}
	kind, err := classify(rel, ust.Mode, uint64(ust.Rdev), m)
	if err != nil {
		return fmt.Errorf("merge: %w", err)
	}
	var lst unix.Stat_t
	lk := LowerAbsent
	switch err := unix.Fstatat(lfd, name, &lst, unix.AT_SYMLINK_NOFOLLOW); {
	case err == nil:
		lk = lowerKindOf(lst.Mode)
	case !errors.Is(err, unix.ENOENT):
		return fmt.Errorf("merge: read lower %s: %w", rel, err)
	}

	s := decide(kind, lk)
	if s.last == 0 {
		return w.mergeDir(ufd, lfd, dfd, name, rel, ust.Mtim.Nano())
	}
	if s.discard {
		path := filepath.Join(w.lower, filepath.FromSlash(rel))
		if err := w.call(CallDiscard, rel, tallyNone, func() error { return w.opt.Discard(path) }); err != nil {
			return err
		}
	}
	if s.clear {
		if err := w.call(CallClearOpaque, rel, tallyNone, func() error {
			return unix.Fremovexattr(dfd, xattrOpaque)
		}); err != nil {
			return err
		}
	}
	switch {
	case s.last == CallUnlink:
		return w.call(CallUnlink, rel, s.tally, func() error { return unix.Unlinkat(ufd, name, 0) })
	case s.replace:
		return w.call(CallRename, rel, s.tally, func() error { return unix.Renameat(ufd, name, lfd, name) })
	default:
		// 디렉터리는 덮어쓰지 않는다. 판정은 「없다」인데 무엇이 있으면 오류로 멈춘다
		return w.call(CallRename, rel, s.tally, func() error {
			return unix.Renameat2(ufd, name, lfd, name, unix.RENAME_NOREPLACE)
		})
	}
}

// mergeDir 은 양쪽에 있는 디렉터리다 — 들어가기 전 mtime 을 적어 두고, 안을 합치고, 소유 ·
// mode · mtime 을 맞추고, 빈 upper 쪽을 지운다 (business-rules.md 5절).
func (w *walker) mergeDir(ufd, lfd, dfd int, name, rel string, mtime int64) error {
	ldfd, err := unix.Openat(lfd, name, openDirFlags, 0)
	if err != nil {
		return fmt.Errorf("merge: read lower %s: %w", rel, err)
	}
	defer func() { _ = unix.Close(ldfd) }()

	stashed, ok, err := readStash(dfd)
	if err != nil {
		return fmt.Errorf("merge: read %s: %w", rel, err)
	}
	if ok {
		mtime = stashed
	} else {
		v := []byte(strconv.FormatInt(mtime, 10))
		if err := w.call(CallStashMtime, rel, tallyNone, func() error {
			return unix.Fsetxattr(dfd, stashXattr, v, 0)
		}); err != nil {
			return err
		}
	}

	if err := w.walk(dfd, ldfd, rel); err != nil {
		return err
	}

	var st unix.Stat_t
	if err := unix.Fstat(dfd, &st); err != nil {
		return fmt.Errorf("merge: read %s: %w", rel, err)
	}
	if err := w.call(CallChown, rel, tallyNone, func() error {
		return unix.Fchown(ldfd, int(st.Uid), int(st.Gid))
	}); err != nil {
		return err
	}
	if err := w.call(CallChmod, rel, tallyNone, func() error {
		return unix.Fchmod(ldfd, uint32(st.Mode)&0o7777)
	}); err != nil {
		return err
	}
	times := []unix.Timespec{{Nsec: unix.UTIME_OMIT}, unix.NsecToTimespec(mtime)}
	if err := w.call(CallChtimes, rel, tallyNone, func() error {
		return unix.UtimesNanoAt(lfd, name, times, unix.AT_SYMLINK_NOFOLLOW)
	}); err != nil {
		return err
	}
	return w.call(CallRmdir, rel, tallyMergedDir, func() error {
		return unix.Unlinkat(ufd, name, unix.AT_REMOVEDIR)
	})
}

// call 은 호출 하나다. 성공하면 세고 OnOp 를 부르고 ctx 를 본다. 실패하면 되돌리지 않고
// 멈춘다 — 오류로 멈춘 자리와 죽어서 멈춘 자리가 같은 모양으로 남는다.
func (w *walker) call(c Call, rel string, t tally, fn func() error) error {
	if err := fn(); err != nil {
		return &OpError{Op: Op{Call: c, Path: rel}, Err: err}
	}
	w.res.Ops++
	if c == CallDiscard {
		w.res.Discarded++
	}
	w.res.count(t)
	if w.opt.OnOp != nil {
		if err := w.opt.OnOp(Op{Call: c, Path: rel}); err != nil {
			return err
		}
	}
	return w.ctx.Err()
}

// readNames 는 디렉터리 안의 이름을 바이트 순으로 낸다. 새 fd 로 읽는다 — 같은 fd 를 다시
// 읽을 때 읽기 자리가 끝에 가 있지 않게.
func readNames(dirfd int) ([]string, error) {
	fd, err := unix.Openat(dirfd, ".", openDirFlags, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), ".")
	names, err := f.Readdirnames(-1)
	_ = f.Close()
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	return names, nil
}

// marksOfDir 은 연 디렉터리 fd 로 표시를 읽는다.
func marksOfDir(fd int) (Marks, error) {
	names, err := xattrNames(func(b []byte) (int, error) { return unix.Flistxattr(fd, b) })
	if err != nil {
		return Marks{}, err
	}
	return marksFrom(names, func() ([]byte, error) {
		return xattrValue(func(b []byte) (int, error) { return unix.Fgetxattr(fd, xattrOpaque, b) })
	})
}

// marksAt 은 디렉터리가 아닌 항목의 표시를 부모 fd 아래의 이름으로 읽는다.
// /proc/self/fd/<부모> 는 부모 디렉터리로 곧장 이어지는 링크라 경로를 이어 붙이지 않는다.
// l 계열이라 마지막 조각이 symlink 여도 따라가지 않는다.
func marksAt(parent int, name string) (Marks, error) {
	path := "/proc/self/fd/" + strconv.Itoa(parent) + "/" + name
	names, err := xattrNames(func(b []byte) (int, error) { return unix.Llistxattr(path, b) })
	if err != nil {
		return Marks{}, err
	}
	return marksFrom(names, func() ([]byte, error) {
		return xattrValue(func(b []byte) (int, error) { return unix.Lgetxattr(path, xattrOpaque, b) })
	})
}

// readStash 는 stashXattr 를 읽는다. 없으면 ok 가 false 다.
func readStash(fd int) (int64, bool, error) {
	v, err := xattrValue(func(b []byte) (int, error) { return unix.Fgetxattr(fd, stashXattr, b) })
	if err != nil || v == nil {
		return 0, false, err
	}
	n, err := strconv.ParseInt(string(v), 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("bad %s %q", stashXattr, v)
	}
	return n, true, nil
}

// xattrNames 는 listxattr 를 크기부터 묻고 읽는다. 사이에 늘었으면(ERANGE) 다시 묻는다.
// xattr 을 지원하지 않는 filesystem 이면 빈 목록이다.
func xattrNames(list func([]byte) (int, error)) ([]string, error) {
	for {
		n, err := list(nil)
		if errors.Is(err, unix.ENOTSUP) {
			return nil, nil
		}
		if err != nil || n == 0 {
			return nil, err
		}
		buf := make([]byte, n)
		n, err = list(buf)
		if errors.Is(err, unix.ERANGE) {
			continue
		}
		if err != nil {
			return nil, err
		}
		return splitNames(buf[:n]), nil
	}
}

// xattrValue 는 getxattr 를 크기부터 묻고 읽는다. 없으면 nil 이다. 있는데 비었으면 빈 조각이다.
func xattrValue(get func([]byte) (int, error)) ([]byte, error) {
	for {
		n, err := get(nil)
		if errors.Is(err, unix.ENODATA) || errors.Is(err, unix.ENOTSUP) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		buf := make([]byte, n)
		n, err = get(buf)
		if errors.Is(err, unix.ERANGE) {
			continue
		}
		if errors.Is(err, unix.ENODATA) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		return buf[:n], nil
	}
}
