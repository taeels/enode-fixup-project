package lower

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

// ForeignMounts 는 같은 uid 의 프로세스를 마운트 namespace 마다 한 번씩 읽어 이 lower 에 닿는 overlay 를
// 찾는다 (business-rules.md 8.2). 증거가 아니라 그물이다 — 0 이어도 배타 잠금 없이 합치지 않는다. 부르는 쪽
// (bake 의 merge 단계와 기동 때 재개)이 배타를 잡은 뒤 · merging 을 적기 전에 부른다.
func ForeignMounts(root Root) (MountScan, error) { return scanMounts("/proc", root, os.Getuid()) }

// scanMounts 는 proc 아래를 훑는다. 시험은 가짜 /proc 을 준다.
//
//	이 lower 의 자리   <proc>/self/mountinfo 에서 lower 를 담은 마운트 — (장치, filesystem 안의 경로 L)
//	훑는 프로세스       주인이 이 uid 인 pid.  ns/mnt 링크마다 한 번 읽는다
//	못 읽은 것         권한이면 Unreadable 로 센다.  사라졌거나 좀비면 (ns 링크가 없다) 건너뛴다
func scanMounts(proc string, root Root, uid int) (MountScan, error) {
	var scan MountScan
	self := filepath.Join(proc, "self", "mountinfo")
	b, err := os.ReadFile(self)
	if err != nil {
		return scan, openErr(self, err)
	}
	lines, err := parseMountinfo(b)
	if err != nil {
		return scan, &pathError{Path: self, Err: err}
	}
	m, ok := mountByID(lines, root.MountID)
	if !ok {
		m, ok = longestMount(lines, root.Path)
	}
	if !ok {
		return scan, &pathError{Path: root.Path, Err: fmt.Errorf("no mount holds it in %s", self)}
	}
	dev, inPath := m.Dev, inFS(m, root.Path)

	ents, err := os.ReadDir(proc)
	if err != nil {
		return scan, openErr(proc, err)
	}
	seen := map[string]bool{}
	for _, e := range ents {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid <= 0 {
			continue
		}
		dir := filepath.Join(proc, e.Name())
		switch owned(dir, uid) {
		case ownerOther:
			continue
		case ownerHidden:
			scan.Unreadable++
			continue
		}
		ns, err := os.Readlink(filepath.Join(dir, "ns", "mnt"))
		if err != nil {
			if denied(err) {
				scan.Unreadable++
			}
			continue // 사라졌거나 좀비 — 마운트를 쥐지 않는다
		}
		if seen[ns] {
			continue
		}
		info, err := os.ReadFile(filepath.Join(dir, "mountinfo"))
		if err != nil {
			if denied(err) {
				scan.Unreadable++
			}
			continue // 그 사이 끝났다 — 같은 namespace 의 다른 프로세스가 읽힐 수 있다
		}
		seen[ns] = true
		scan.Namespaces++
		ls, err := parseMountinfo(info)
		if err != nil {
			continue
		}
		for _, found := range overlaysOn(ls, dev, inPath) {
			found.PID, found.Namespace = pid, ns
			scan.Found = append(scan.Found, found)
		}
	}
	return scan, nil
}

type ownerKind int

const (
	ownerSelf   ownerKind = iota // 이 uid 의 프로세스
	ownerOther                   // 다른 사용자 — 훑지 않는다 (한 lower 는 한 사용자가 쓴다)
	ownerHidden                  // 이 uid 인데 dumpable 이 꺼져 /proc 이 root 소유로 보인다 — 못 읽는다
)

// owned 는 /proc/<pid> 의 주인으로 그 프로세스가 이 uid 의 것인가를 본다.
func owned(dir string, uid int) ownerKind {
	fi, err := os.Stat(dir)
	if err != nil {
		return ownerOther
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return ownerOther
	}
	return ownerOf(int(st.Uid), uid, func() []byte {
		b, _ := os.ReadFile(filepath.Join(dir, "status"))
		return b
	})
}

// ownerOf 는 /proc/<pid> 의 주인(dirUID)으로 판정한다. dumpable 이 꺼진 프로세스는 /proc/<pid> 가 root 소유로
// 보이므로 status 의 Uid 줄(실제 uid)로 한 번 더 본다 — 그때만 status 를 읽는다.
func ownerOf(dirUID, uid int, status func() []byte) ownerKind {
	switch {
	case dirUID == uid:
		return ownerSelf
	case dirUID != 0 || uid == 0:
		return ownerOther
	}
	for _, line := range strings.Split(string(status()), "\n") {
		if rest, ok := strings.CutPrefix(line, "Uid:"); ok {
			if f := strings.Fields(rest); len(f) > 0 && f[0] == strconv.Itoa(uid) {
				return ownerHidden
			}
			return ownerOther
		}
	}
	return ownerOther
}

// denied 는 권한 때문에 못 읽은 것인가다.
func denied(err error) bool {
	return errors.Is(err, unix.EACCES) || errors.Is(err, unix.EPERM)
}
