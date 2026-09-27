package lower

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

// 자리와 파일을 여는 법 (계획 3.1 · 4절 ① ② ③).
//
//	디렉터리   O_DIRECTORY|O_NOFOLLOW|O_CLOEXEC 로 열고 fstat — symlink 면 열리지 않는다 (MkdirAll 은 따라간다)
//	잠금 파일  os.OpenFile(O_RDWR|O_CREATE|O_NOFOLLOW, 0600) — Go 가 O_CLOEXEC 를 붙여 자식에게 안 샌다
//	JSON 읽기  O_NOFOLLOW · 보통 파일 · 주인 · 1 MiB 까지
//	JSON 쓰기  같은 디렉터리의 CreateTemp (쓰는 쪽마다 이름이 다르다) 뒤 rename.  덮어쓰면 늘 0600 이다
//
// 주인이 다른 것과 symlink 는 거절한다 — 특권 없이 고칠 수 없고 남이 놓았을 수 있다. 남에게 열린 비트는
// 데몬이 좁힌다 — 자리는 데몬의 것이다.

const dirFlags = unix.O_RDONLY | unix.O_DIRECTORY | unix.O_NOFOLLOW | unix.O_CLOEXEC

// checkFd 는 연 자리 하나를 본다. narrow 가 참이면 남에게 열린 비트를 좁히고 좁혔는지를 돌려준다.
// 거짓이면 좁히지 않고 loose 로 알린다.
func checkFd(fd int, p string, uid int, dir, narrow bool) (loose bool, err error) {
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return false, &pathError{Path: p, Err: err}
	}
	r := permVerdict(st.Mode, int(st.Uid), uid, dir)
	if err := permError(p, r, int(st.Uid), uid, dir); err != nil {
		return false, err
	}
	if r != permLoose {
		return false, nil
	}
	if !narrow {
		return true, nil
	}
	mode := uint32(0o600)
	if dir {
		mode = 0o700
	}
	if err := unix.Fchmod(fd, mode); err != nil {
		return true, &pathError{Path: p, Err: err}
	}
	return true, nil
}

// openErr 는 여는 호출의 오류를 자리의 문구로 바꾼다. symlink 는 ELOOP 나 ENOTDIR 로 온다.
func openErr(p string, err error) error {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		err = pe.Err
	}
	return &pathError{Path: p, Err: err}
}

// ensureDir 는 부모 fd 아래 name 을 0700 으로 만들고(있으면 그대로) 다시 열어 본다. 부모 fd 기준으로
// 만들고 열어서 경로의 중간이 바뀌어도 밖에 만들지 않는다. 연 fd 를 돌려준다 — 닫는 것은 부르는 쪽이다.
func ensureDir(parent int, name, p string, uid int, notes *[]string) (int, error) {
	if err := unix.Mkdirat(parent, name, 0o700); err != nil && !errors.Is(err, unix.EEXIST) {
		return -1, &pathError{Path: p, Err: err}
	}
	fd, err := unix.Openat(parent, name, dirFlags, 0)
	if err != nil {
		return -1, &pathError{Path: p, Err: err}
	}
	narrowed, err := checkFd(fd, p, uid, true, true)
	if err != nil {
		_ = unix.Close(fd)
		return -1, err
	}
	if narrowed {
		*notes = append(*notes, "narrowed loose permissions on "+p+" to 0700")
	}
	return fd, nil
}

// peekDir 은 이미 있는 자리를 연다. 없으면 fd -1 과 nil 이다. 만들지 않고 고치지 않는다.
func peekDir(parent int, name, p string, uid int) (fd int, loose bool, err error) {
	fd, err = unix.Openat(parent, name, dirFlags, 0)
	if errors.Is(err, unix.ENOENT) {
		return -1, false, nil
	}
	if err != nil {
		return -1, false, &pathError{Path: p, Err: err}
	}
	loose, err = checkFd(fd, p, uid, true, false)
	if err != nil {
		_ = unix.Close(fd)
		return -1, false, err
	}
	return fd, loose, nil
}

// openLock 은 잠금 파일을 연다 — 없으면 0600 으로 만든다. symlink 면 ELOOP 이고 아무것도 안 만든다 (측정 3).
// 보통 파일 · 주인을 보고 남에게 열린 비트를 좁힌다. 좁혔는지를 함께 돌려준다.
func openLock(p string, uid int) (*os.File, bool, error) {
	f, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, false, openErr(p, err)
	}
	narrowed, err := checkFd(int(f.Fd()), p, uid, false, true)
	if err != nil {
		f.Close()
		return nil, false, err
	}
	return f, narrowed, nil
}

// openLockRO 는 이미 있는 잠금 파일을 읽기로 연다 — 만들지 않는다 (smoke 의 WaitShared · 기록 읽기).
// 없으면 fs.ErrNotExist 를 담은 오류다. flock 은 연 방식과 무관하게 걸린다 (측정 8).
func openLockRO(p string, uid int) (*os.File, error) {
	f, err := os.OpenFile(p, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, openErr(p, err)
	}
	if _, err := checkFd(int(f.Fd()), p, uid, false, false); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

// readJSON 은 자리 안의 JSON 하나를 읽는다. uid 가 음수면 주인을 보지 않는다 (lower 뿌리의 metadata).
// 없으면 fs.ErrNotExist 를 담은 오류다.
func readJSON(p string, uid int, v any) error {
	f, err := os.OpenFile(p, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return openErr(p, err)
	}
	defer f.Close()
	var st unix.Stat_t
	if err := unix.Fstat(int(f.Fd()), &st); err != nil {
		return &pathError{Path: p, Err: err}
	}
	owner := uid
	if owner < 0 {
		owner = int(st.Uid)
	}
	if r := permVerdict(st.Mode&^modeLoose, int(st.Uid), owner, false); r != permOK {
		return permError(p, r, int(st.Uid), owner, false)
	}
	if st.Size > maxJSON {
		return &pathError{Path: p, Err: fmt.Errorf("larger than %d bytes", maxJSON)}
	}
	b, err := io.ReadAll(io.LimitReader(f, maxJSON+1))
	if err != nil {
		return &pathError{Path: p, Err: err}
	}
	if len(b) > maxJSON {
		return &pathError{Path: p, Err: fmt.Errorf("larger than %d bytes", maxJSON)}
	}
	if err := json.Unmarshal(b, v); err != nil {
		return &pathError{Path: p, Err: err}
	}
	return nil
}

// tempPattern 은 쓰는 중인 임시 파일의 이름이다 — .<파일>.tmp-<무작위>. 점으로 시작하므로 기록을 읽는
// 쪽이 건너뛴다. 죽은 쓰기가 남긴 것은 지우지 않는다 — 지우는 쪽이 다른 데몬의 쓰는 중인 파일과 경쟁한다.
func tempPattern(name string) string {
	if strings.HasPrefix(name, ".") {
		return name + ".tmp-*"
	}
	return "." + name + ".tmp-*"
}

// writeJSON 은 dir 안의 name 을 임시 파일 + rename 으로 바꾼다. 읽는 쪽은 반쯤 쓴 파일을 못 본다 (측정 6).
// durable 이면 파일과 디렉터리를 fsync 한다 — state.json · lower.json · metadata. 쥔 사람 기록은 안 한다
// (살아 있음은 잠금이 말한다). perm 이 0 이면 CreateTemp 의 0600 그대로다.
func writeJSON(dir, name string, v any, durable bool, perm os.FileMode) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	final := filepath.Join(dir, name)
	f, err := os.CreateTemp(dir, tempPattern(name))
	if err != nil {
		return openErr(final, err)
	}
	tmp := f.Name()
	fail := func(err error) error {
		f.Close()
		_ = os.Remove(tmp)
		return &pathError{Path: final, Err: err}
	}
	if _, err := f.Write(b); err != nil {
		return fail(err)
	}
	if perm != 0 {
		if err := f.Chmod(perm); err != nil {
			return fail(err)
		}
	}
	if durable {
		if err := f.Sync(); err != nil {
			return fail(err)
		}
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return &pathError{Path: final, Err: err}
	}
	if err := os.Rename(tmp, final); err != nil {
		_ = os.Remove(tmp)
		return openErr(final, err)
	}
	if durable {
		return syncDir(dir)
	}
	return nil
}

// syncDir 는 rename 을 디스크에 남긴다.
func syncDir(dir string) error {
	fd, err := unix.Open(dir, dirFlags, 0)
	if err != nil {
		return &pathError{Path: dir, Err: err}
	}
	defer func() { _ = unix.Close(fd) }()
	if err := unix.Fsync(fd); err != nil {
		return &pathError{Path: dir, Err: err}
	}
	return nil
}
