// Package merge 는 굽기의 대기 upper 를 lower 에 합치는 규칙이다 (ADR-077 §4 · requirements.md FR-7).
//
// 담는 것 — 시작 전 확인(Preflight)과 합치기(Apply). upper 를 걸으며 항목마다 lower 에
// rename 으로 적용하고, 표시(whiteout · opaque)는 그 효과를 적용한 뒤에만 치운다. 그래서
// 어느 호출 뒤에서 끊겨도 같은 절차를 다시 부르면 한 번에 끝낸 것과 같다 (결정 3-5).
//
// 모르는 것 — lower 의 상태 파일 · 잠금 · 광고 · namespace 를 여는 일 · Run · trash 의 이름
// 규칙. lower 쪽을 버리는 일은 부르는 쪽이 Options.Discard 로 채운다 (merge-helper 가
// scratch.Trash.Move 로). 부르는 쪽은 lower 의 배타 잠금을 쥐고 온다 — 그것이 「이 lower 의
// overlay 마운트 0」의 증거다.
//
// 표준 라이브러리와 golang.org/x/sys 만 쓴다 (경계 시험의 봉인). Mediator 는 이 패키지를
// 링크하지 않는다.
package merge

import (
	"errors"
	"fmt"
	"strconv"
)

// Paths 는 합치기의 세 뿌리다. 셋은 노드 설정이 준 경로라 symlink 를 풀어 쓴다.
// 뿌리 아래에서는 symlink 를 따라가지 않는다.
type Paths struct {
	Upper string // 대기 upper. 합치기가 그 안을 비운다. 뿌리 자신은 부르는 쪽이 치운다
	Lower string
	Trash string // <scratch>/trash. 같은 filesystem 확인에만 쓴다. 옮기는 일은 Options.Discard 다
}

// Options 는 부르는 쪽이 채우는 자리다.
type Options struct {
	// Discard 는 lower 쪽 항목 하나를 버린다. 인자는 푼 lower 뿌리에 상대 경로를 이은 절대
	// 경로다. 비어 있으면 Apply 가 아무것도 바꾸지 않고 오류를 돌려준다.
	Discard func(lowerPath string) error

	// OnOp 는 호출 하나가 성공한 뒤마다 불린다. 오류를 돌려주면 Apply 가 거기서 멈추고 그
	// 오류를 그대로 돌려준다. 재개 시험이 N 번째 호출 뒤에 끊는 자리다. 비어 있어도 된다.
	OnOp func(Op) error
}

// Call 은 lower 나 upper 를 바꾸는 호출 하나다. 재개 시험이 끊는 단위다.
type Call int

const (
	CallDiscard     Call = iota + 1 // lower 쪽 항목을 버린다 (Options.Discard)
	CallUnlink                      // upper 의 whiteout 을 지운다
	CallClearOpaque                 // upper 디렉터리의 user.overlay.opaque 를 지운다
	CallRename                      // upper 항목을 lower 의 같은 경로로 옮긴다
	CallStashMtime                  // upper 디렉터리의 mtime 을 그 디렉터리의 xattr 로 적어 둔다
	CallChown                       // lower 디렉터리의 소유를 맞춘다
	CallChmod                       // lower 디렉터리의 mode 를 맞춘다
	CallChtimes                     // lower 디렉터리의 mtime 을 맞춘다
	CallRmdir                       // 빈 upper 디렉터리를 지운다
)

var callNames = [...]string{
	CallDiscard:     "discard",
	CallUnlink:      "unlink",
	CallClearOpaque: "clear-opaque",
	CallRename:      "rename",
	CallStashMtime:  "stash-mtime",
	CallChown:       "chown",
	CallChmod:       "chmod",
	CallChtimes:     "chtimes",
	CallRmdir:       "rmdir",
}

// String 은 로그와 오류에 쓰는 이름이다.
func (c Call) String() string {
	if c > 0 && int(c) < len(callNames) {
		return callNames[c]
	}
	return "call(" + strconv.Itoa(int(c)) + ")"
}

// Op 는 OnOp 가 받는 것이다.
type Op struct {
	Call Call
	Path string // upper 뿌리 기준 상대 경로. 구분자는 "/"
}

// Result 는 그 Apply 호출이 한 일이다. 재개로 다시 부르면 남은 일만 센다.
type Result struct {
	Ops int // 성공한 호출 수 — OnOp 가 불린 수와 같다

	// upper 항목 하나가 끝날 때 아래 일곱 칸 중 꼭 하나가 는다.
	Replaced    int // lower 의 파일 · symlink 를 대체했다
	Added       int // lower 에 없던 파일 · symlink 를 들였다
	NewDirs     int // lower 에 없던 디렉터리를 하위 트리째 한 번에 들였다
	OpaqueDirs  int
	Whiteouts   int
	TypeChanged int // 종류가 바뀐 항목 (FR-7)
	MergedDirs  int // 양쪽에 있는 디렉터리 — 안으로 들어가 속성을 맞추고 upper 쪽을 지웠다

	Discarded int // lower 쪽을 버린 수
}

// tally 는 Result 의 일곱 칸 중 어느 것이 느나다.
type tally int

const (
	tallyNone tally = iota
	tallyReplaced
	tallyAdded
	tallyNewDir
	tallyOpaqueDir
	tallyWhiteout
	tallyTypeChanged
	tallyMergedDir
)

func (r *Result) count(t tally) {
	switch t {
	case tallyReplaced:
		r.Replaced++
	case tallyAdded:
		r.Added++
	case tallyNewDir:
		r.NewDirs++
	case tallyOpaqueDir:
		r.OpaqueDirs++
	case tallyWhiteout:
		r.Whiteouts++
	case tallyTypeChanged:
		r.TypeChanged++
	case tallyMergedDir:
		r.MergedDirs++
	}
}

// Check 는 시작 전 확인의 어느 줄이 어긋났나다. 부르는 쪽이 이것으로 상태를 어디로 돌릴지 정한다.
type Check string

const (
	CheckPath       Check = "path"       // 절대 경로가 아니다 · 풀 수 없다
	CheckDirectory  Check = "directory"  // 진짜 디렉터리가 아니다 (없음 포함)
	CheckOverlap    Check = "overlap"    // 셋 중 하나가 다른 하나와 같거나 그 안에 있다
	CheckFilesystem Check = "filesystem" // st_dev 가 셋이 같지 않다
	CheckMark       Check = "mark"       // upper 에 금지 표시가 있다
)

// PreflightError 는 시작 전 확인이 어긋난 까닭이다.
type PreflightError struct {
	Check Check
	Path  string // 뿌리의 경로, 또는 (mark 면) upper 기준 상대 경로
	Err   error  // mark 면 *MarkError
}

func (e *PreflightError) Error() string {
	switch e.Check {
	case CheckPath, CheckDirectory:
		return fmt.Sprintf("merge preflight: %s: %v", e.Path, e.Err)
	default:
		return "merge preflight: " + e.Err.Error()
	}
}

func (e *PreflightError) Unwrap() error { return e.Err }

// MarkError 는 upper 항목에 있으면 안 되는 표시다.
type MarkError struct {
	Path  string // upper 기준 상대 경로. 뿌리면 "."
	Name  string // 표시의 xattr 이름
	Value []byte // opaque 의 값
}

func (e *MarkError) Error() string {
	switch {
	case e.Name == xattrOpaque && e.Path == ".":
		return "upper entry . carries " + xattrOpaque + "; the upper root cannot be opaque"
	case e.Name == xattrOpaque:
		return fmt.Sprintf("upper entry %s has %s=%q; only \"y\" on a directory is read as opaque",
			e.Path, xattrOpaque, e.Value)
	case e.Name == xattrWhiteout:
		return "upper entry " + e.Path + " carries " + xattrWhiteout +
			"; only 0/0 character devices are read as whiteouts"
	default:
		return "upper entry " + e.Path + " carries " + e.Name
	}
}

// OpError 는 호출 하나가 실패한 자리다. Discard 가 돌려준 오류도 이것으로 감싼다.
type OpError struct {
	Op  Op
	Err error
}

func (e *OpError) Error() string {
	return fmt.Sprintf("merge: %s %s: %v", e.Op.Call, e.Op.Path, e.Err)
}

func (e *OpError) Unwrap() error { return e.Err }

// ErrUnsupported 는 linux 밖에서 Preflight 와 Apply 가 돌려준다.
var ErrUnsupported = errors.New("merge is supported on linux only")

var (
	errNoDiscard = errors.New("merge: no discard function")
	errNotEmpty  = errors.New("merge: upper is not empty after the walk")
)

// stashXattr 는 양쪽 디렉터리의 upper 쪽에 들어가기 전의 mtime 을 적어 두는 자리다
// (business-rules.md 5절). 안의 항목을 옮기면 upper 디렉터리의 mtime 이 바뀌므로, 재개 때
// 맞출 값을 여기서 읽는다. overlay 표시가 아니다 — 이름이 user.overlay. 로 시작하지 않는다.
// 그 디렉터리는 rmdir 로 사라지므로 lower 로 가지 않는다.
const stashXattr = "user.enode.merge-mtime"
