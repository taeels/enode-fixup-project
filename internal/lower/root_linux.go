package lower

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// selfMountinfo 는 이 프로세스의 마운트 표다. 시험이 바꾸지 않는다 — 가짜 /proc 은 ForeignMounts 쪽만 쓴다.
const selfMountinfo = "/proc/self/mountinfo"

// ReadRoot 는 lower 루트를 한 번 읽는다 — symlink 를 풀고 · 디렉터리인지 보고 · statfs 와 statx 한 번씩.
// 커널이 마운트 번호를 안 주면(5.8 전) /proc/self/mountinfo 에서 경로가 가장 길게 겹치는 마운트를 찾는다 (답 6).
func ReadRoot(path string) (Root, error) {
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return Root{}, fmt.Errorf("lower: %w", err)
	}
	var sx unix.Statx_t
	mask := unix.STATX_TYPE | unix.STATX_UID | unix.STATX_INO | unix.STATX_BTIME | unix.STATX_MNT_ID
	if err := unix.Statx(unix.AT_FDCWD, real, unix.AT_SYMLINK_NOFOLLOW, mask, &sx); err != nil {
		return Root{}, &pathError{Path: real, Err: err}
	}
	if uint32(sx.Mode)&modeType != modeDir {
		return Root{}, &pathError{Path: path, Err: errNotDir}
	}
	var fs unix.Statfs_t
	if err := unix.Statfs(real, &fs); err != nil {
		return Root{}, &pathError{Path: real, Err: err}
	}
	r := Root{Path: real, Key: keyOf(fs.Fsid.Val[0], fs.Fsid.Val[1], sx.Ino),
		Dev: mkdev(sx.Dev_major, sx.Dev_minor), UID: int(sx.Uid)}
	if sx.Mask&unix.STATX_BTIME != 0 {
		r.BirthNs = sx.Btime.Sec*1e9 + int64(sx.Btime.Nsec)
	}
	if sx.Mask&unix.STATX_MNT_ID != 0 {
		r.MountID = sx.Mnt_id
	}
	if r.MountID == 0 {
		r.MountID = mountIDOf(selfMountinfo, real)
	}
	return r, nil
}

// mountIDOf 는 mountinfo 에서 경로 p 를 담은 마운트의 번호다. 못 찾으면 0 이다 — 점검이 장치만 댄다.
func mountIDOf(mountinfo, p string) uint64 {
	b, err := os.ReadFile(mountinfo)
	if err != nil {
		return 0
	}
	lines, err := parseMountinfo(b)
	if err != nil {
		return 0
	}
	if m, ok := longestMount(lines, p); ok {
		return m.ID
	}
	return 0
}
