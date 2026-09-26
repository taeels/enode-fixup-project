package merge

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
)

// 시스템 호출 없이 판정하는 자리다. 모든 플랫폼에서 빌드되고 표 시험이 덮는다.

// st_mode 의 종류 비트. x/sys/unix 는 linux 밖에서 없으므로 여기 둔다.
const (
	modeType = 0o170000
	modeDir  = 0o040000
	modeChr  = 0o020000
)

// overlay 표시의 이름. 정확히 맞춘다 — 앞부분(user.overlay.)으로 보지 않는다. escape 한
// 이름(user.overlay.overlay.*)은 세션이 붙인 보통 xattr 이다.
const (
	xattrOpaque   = "user.overlay.opaque"
	xattrWhiteout = "user.overlay.whiteout"
	xattrMetacopy = "user.overlay.metacopy"
	xattrRedirect = "user.overlay.redirect"
)

// Kind 는 upper 항목 하나의 종류다.
type Kind int

const (
	KindOther     Kind = iota + 1 // 파일 · symlink · 그 밖 (fifo · socket · 0/0 이 아닌 장치)
	KindDir                       // opaque 가 아닌 디렉터리
	KindOpaqueDir                 // user.overlay.opaque 의 값이 정확히 "y" 인 디렉터리
	KindWhiteout                  // 문자 장치 0/0
)

// LowerKind 는 같은 경로의 lower 쪽이다. lstat 으로 본다 — symlink 는 무엇을 가리키든
// LowerOther 다.
type LowerKind int

const (
	LowerAbsent LowerKind = iota + 1
	LowerDir
	LowerOther
)

// Marks 는 upper 항목에서 읽은 overlay 표시다.
type Marks struct {
	Opaque   []byte // user.overlay.opaque 의 값. 없으면 nil
	Whiteout bool
	Metacopy bool
	Redirect bool
}

// splitNames 는 listxattr 가 준 버퍼(이름마다 NUL 로 끝난다)를 이름 목록으로 나눈다.
func splitNames(buf []byte) []string {
	var names []string
	for _, b := range bytes.Split(buf, []byte{0}) {
		if len(b) > 0 {
			names = append(names, string(b))
		}
	}
	return names
}

// marksFrom 은 xattr 이름 목록에서 표시를 고른다. opaque 가 있을 때만 그 값을 읽는다.
func marksFrom(names []string, opaque func() ([]byte, error)) (Marks, error) {
	var m Marks
	for _, n := range names {
		switch n {
		case xattrOpaque:
			v, err := opaque()
			if err != nil {
				return Marks{}, err
			}
			if v == nil {
				v = []byte{}
			}
			m.Opaque = v
		case xattrWhiteout:
			m.Whiteout = true
		case xattrMetacopy:
			m.Metacopy = true
		case xattrRedirect:
			m.Redirect = true
		}
	}
	return m, nil
}

// classify 는 upper 항목 하나를 판정한다 (business-rules.md 2절). rel 은 upper 뿌리 기준
// 상대 경로이고 뿌리면 "." 다. mode 는 st_mode, rdev 는 st_rdev 다. 금지 표시면 *MarkError 다.
func classify(rel string, mode uint32, rdev uint64, m Marks) (Kind, error) {
	isDir := mode&modeType == modeDir
	switch {
	case m.Metacopy:
		return 0, &MarkError{Path: rel, Name: xattrMetacopy}
	case m.Redirect:
		return 0, &MarkError{Path: rel, Name: xattrRedirect}
	case m.Whiteout:
		return 0, &MarkError{Path: rel, Name: xattrWhiteout}
	case m.Opaque != nil && (rel == "." || !isDir || string(m.Opaque) != "y"):
		return 0, &MarkError{Path: rel, Name: xattrOpaque, Value: m.Opaque}
	}
	switch {
	case mode&modeType == modeChr && rdev == 0:
		return KindWhiteout, nil
	case isDir && m.Opaque != nil:
		return KindOpaqueDir, nil
	case isDir:
		return KindDir, nil
	default:
		return KindOther, nil
	}
}

// lowerKindOf 는 lstat 의 st_mode 로 lower 쪽 종류를 낸다. symlink 는 디렉터리가 아니다.
func lowerKindOf(mode uint32) LowerKind {
	if mode&modeType == modeDir {
		return LowerDir
	}
	return LowerOther
}

// step 은 (upper 종류, lower 종류) 한 칸의 할 일이다.
type step struct {
	discard bool  // 먼저 lower 쪽을 버린다
	clear   bool  // 그다음 upper 의 opaque 를 지운다
	last    Call  // CallRename · CallUnlink. 0 이면 양쪽 디렉터리 — 안으로 들어간다
	replace bool  // last 가 CallRename 일 때 대체를 허락하나 (파일 · symlink). 디렉터리는 덮어쓰지 않는다
	tally   tally // 항목이 끝나면 느는 칸
}

// decide 는 business-rules.md 1절의 표다.
func decide(k Kind, l LowerKind) step {
	lowerThere := l != LowerAbsent
	switch k {
	case KindWhiteout:
		return step{discard: lowerThere, last: CallUnlink, tally: tallyWhiteout}
	case KindOpaqueDir:
		return step{discard: lowerThere, clear: true, last: CallRename, tally: tallyOpaqueDir}
	case KindDir:
		switch l {
		case LowerDir:
			return step{tally: tallyMergedDir}
		case LowerOther:
			return step{discard: true, last: CallRename, tally: tallyTypeChanged}
		default:
			return step{last: CallRename, tally: tallyNewDir}
		}
	default:
		switch l {
		case LowerDir:
			return step{discard: true, last: CallRename, replace: true, tally: tallyTypeChanged}
		case LowerOther:
			return step{last: CallRename, replace: true, tally: tallyReplaced}
		default:
			return step{last: CallRename, replace: true, tally: tallyAdded}
		}
	}
}

// checkOverlap 는 푼 세 뿌리 중 어느 것도 다른 것과 같거나 그 안에 있지 않은지 본다.
// 경로 조각 단위로 앞부분을 비교한다 — /a/up 은 /a/upper 의 앞이 아니다.
func checkOverlap(upper, lower, trash string) error {
	roots := [...]string{upper, lower, trash}
	for i := range roots {
		for j := i + 1; j < len(roots); j++ {
			if within(roots[i], roots[j]) || within(roots[j], roots[i]) {
				return &PreflightError{Check: CheckOverlap, Path: roots[i],
					Err: fmt.Errorf("%s and %s overlap", roots[i], roots[j])}
			}
		}
	}
	return nil
}

// within 은 child 가 parent 와 같거나 그 아래인가다. 둘 다 정리된 절대 경로다.
func within(parent, child string) bool {
	if parent == child {
		return true
	}
	sep := string(filepath.Separator)
	if strings.HasSuffix(parent, sep) {
		return strings.HasPrefix(child, parent)
	}
	return strings.HasPrefix(child, parent+sep)
}

// checkDevices 는 세 뿌리의 st_dev 가 같은지 본다. 다르면 도중의 rename 이 EXDEV 로 실패한다.
func checkDevices(upper, lower, trash uint64) error {
	if upper == lower && lower == trash {
		return nil
	}
	return &PreflightError{Check: CheckFilesystem, Err: fmt.Errorf(
		"upper, lower and trash must share one filesystem (upper dev %d, lower dev %d, trash dev %d)",
		upper, lower, trash)}
}

// joinRel 은 상대 경로에 이름 하나를 잇는다. 뿌리 "." 아래는 이름 그대로다.
func joinRel(rel, name string) string {
	if rel == "." {
		return name
	}
	return rel + "/" + name
}
