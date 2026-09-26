//go:build unix

package scratch

import (
	"io/fs"
	"os"

	"golang.org/x/sys/unix"
)

// Measure 는 지우기 전에 항목을 한 번 걷는다 (답 5 = A). 블록 수 x 512 의 합과 항목 수
// (디렉터리 포함)다. Remove 와 같은 경계를 지킨다 — trash 와 다른 filesystem 은 안 센다.
//
// helper 안(runtime 과 같은 uid 매핑의 user namespace)에서 돈다. 노드 uid 는 subordinate
// uid 소유 디렉터리와 권한 000 디렉터리 안을 못 걷는다.
func Measure(trashDir, entry string) (Size, error) {
	w, fd, err := openTrash(trashDir, nil)
	if err != nil {
		return Size{}, err
	}
	defer func() { _ = unix.Close(fd) }()
	w.measure(fd, entry, entry)
	return w.size, w.err
}

// Remove 는 항목을 지운다 (business-rules.md 5절).
//
//	걷기는 lstat 으로만 본다.  디렉터리는 O_NOFOLLOW 로 열고 그 fd 에 기대어 내려간다
//	symlink 는 링크 자체만 지운다.  가리키는 곳으로 안 들어간다
//	권한이 모자란 디렉터리(overlay 의 work/work 는 000)는 들어가기 전에 0700 으로 푼다
//	trash 와 st_dev 가 다른 것은 안 들어가고 left 에 남긴다 — 남은 마운트로 다른
//	filesystem 을 지우는 일을 막는다.  그러면 그 항목은 다 못 지운 것이다
//
// 한 곳이 실패해도 나머지는 지운다. 돌려주는 오류는 첫 실패다.
func Remove(trashDir, entry string) (left []string, err error) {
	return remove(trashDir, entry, nil)
}

// crossesFn 은 걷는 경로가 trash 와 다른 filesystem 인가다. 시험이 바꿔 끼운다 — 시험은
// 다른 st_dev 를 만들 수 없다.
type crossesFn func(rel string, dev uint64) bool

func remove(trashDir, entry string, crosses crossesFn) ([]string, error) {
	w, fd, err := openTrash(trashDir, crosses)
	if err != nil {
		return nil, err
	}
	defer func() { _ = unix.Close(fd) }()
	w.remove(fd, entry, entry)
	return w.left, w.err
}

// openTrash 는 trash 를 symlink 를 안 따라 연다. crosses 가 nil 이면 trash 의 st_dev 와 다른가로 판정한다.
func openTrash(trashDir string, crosses crossesFn) (*walker, int, error) {
	fd, err := unix.Open(trashDir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, -1, &fs.PathError{Op: "open", Path: trashDir, Err: err}
	}
	if crosses == nil {
		var st unix.Stat_t
		if err := unix.Fstat(fd, &st); err != nil {
			_ = unix.Close(fd)
			return nil, -1, &fs.PathError{Op: "stat", Path: trashDir, Err: err}
		}
		dev := uint64(st.Dev)
		crosses = func(_ string, d uint64) bool { return d != dev }
	}
	return &walker{crosses: crosses}, fd, nil
}

// walker 는 Measure 와 Remove 가 함께 쓰는 걷기다. 경로는 trash 기준의 상대 경로로만 적는다.
type walker struct {
	crosses crossesFn
	size    Size
	left    []string
	err     error
}

func (w *walker) note(op, rel string, err error) {
	if w.err == nil {
		w.err = &fs.PathError{Op: op, Path: rel, Err: err}
	}
}

func (w *walker) measure(parent int, name, rel string) {
	var st unix.Stat_t
	if err := unix.Fstatat(parent, name, &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		if err != unix.ENOENT {
			w.note("lstat", rel, err)
		}
		return
	}
	if w.crosses(rel, uint64(st.Dev)) {
		return
	}
	w.size.Bytes += int64(st.Blocks) * 512
	w.size.Entries++
	if st.Mode&unix.S_IFMT == unix.S_IFDIR {
		w.children(parent, name, rel, uint32(st.Mode), w.measure)
	}
}

func (w *walker) remove(parent int, name, rel string) {
	var st unix.Stat_t
	if err := unix.Fstatat(parent, name, &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		if err != unix.ENOENT {
			w.note("lstat", rel, err)
		}
		return
	}
	if w.crosses(rel, uint64(st.Dev)) {
		w.left = append(w.left, rel)
		return
	}
	if st.Mode&unix.S_IFMT != unix.S_IFDIR {
		if err := unix.Unlinkat(parent, name, 0); err != nil && err != unix.ENOENT {
			w.note("remove", rel, err)
		}
		return
	}
	before := len(w.left)
	w.children(parent, name, rel, uint32(st.Mode), w.remove)
	if err := unix.Unlinkat(parent, name, unix.AT_REMOVEDIR); err != nil && err != unix.ENOENT {
		// 안에 남긴 다른 filesystem 이 있으면 비지 않은 것이 맞다 — 오류가 아니다
		if len(w.left) > before && (err == unix.ENOTEMPTY || err == unix.EEXIST) {
			return
		}
		w.note("remove", rel, err)
	}
}

// children 은 디렉터리 안의 이름마다 visit 을 부른다. 권한이 모자라면 먼저 0700 으로 푼다 —
// namespace 안의 root 가 권한 검사를 넘을 수 있다는 가정에 기대지 않는다.
//
// chmod 는 마지막 조각이 symlink 로 바뀌어 있으면 그것을 따른다. fstatat 과 그 사이에 바꿔
// 끼울 프로세스가 없다 — 단계의 프로세스는 닫기에서 다 죽었고 trash 는 노드만 쓴다. 들어가는
// open 은 O_NOFOLLOW 라 symlink 로 내려가지 않는다.
func (w *walker) children(parent int, name, rel string, mode uint32, visit func(int, string, string)) {
	if mode&0o700 != 0o700 {
		if err := unix.Fchmodat(parent, name, 0o700, 0); err != nil {
			w.note("chmod", rel, err)
			return
		}
	}
	fd, err := unix.Openat(parent, name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		w.note("open", rel, err)
		return
	}
	dir := os.NewFile(uintptr(fd), rel)
	defer dir.Close()
	names, err := dir.Readdirnames(-1)
	if err != nil {
		w.note("readdir", rel, err)
	}
	for _, n := range names {
		visit(fd, n, rel+"/"+n)
	}
}
