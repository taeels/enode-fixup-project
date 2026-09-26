# `merge-rules` — 도메인 엔티티

대기 upper 를 lower 에 합치는 규칙의 타입과 칸이다. 입력은 계획의 답 열(`construction/plans/merge-rules-functional-design-plan.md`
2절)이다. 규칙과 오류 문구는 `business-rules.md`, 흐름과 시험은 `business-logic-model.md` 에 있다.

```text
   답   1 A   이 유닛은 마운트를 확인하지 않는다.  부르는 쪽이 쥔 배타 잠금이 「마운트 0」의 증거다.
              옛 노드의 틈은 lower-state 유닛에 넘긴다
        2 A   연산 한 번 = lower 나 upper 를 바꾸는 호출 하나.  시험은 1번째부터 마지막 호출까지 모두 끊는다
        3 A   순수 Go 모형이 기대값을 낸다.  integration 시험이 모형을 커널의 merged view 에 댄다
        4 A   lower 에도 있는 디렉터리는 소유 · mode · 시각을 맞춘다
        5 A   user.overlay.origin · user.overlay.impure 는 지우지 않고 항목과 함께 옮긴다
        6 A   lower 쪽 항목은 부르는 쪽이 채운 함수로 버린다.  merge-helper 가 scratch.Trash.Move 로 채운다
        7 A   디렉터리 fd (열어 둔 디렉터리의 번호) 에 기대어 내려간다.  lower 의 symlink 는 디렉터리로 치지 않는다
        8 A   upper 에 metacopy · redirect · xattr whiteout · 값이 y 가 아닌 opaque 가 있으면 시작하지 않는다
        9 A   첫 오류에서 멈춘다.  ctx 는 호출 사이에서 본다.  그때까지 센 수와 오류를 함께 돌려준다
```

**새 패키지는 하나다** — `internal/merge` (Application Design 의 자리 · `components.md` 2.2). 표준 라이브러리와
`golang.org/x/sys` 만 쓴다. `internal/scratch` 도 임포트하지 않는다 — 새 패키지 셋은 서로 임포트하지 않는다 (`components.md`
2절 · 답 6). Mediator 는 이 패키지를 링크하지 않는다.

```text
   merge.go          Paths · Options · Op · Call · Result · 오류 타입 · 패키지 문서            모든 플랫폼
   decide.go         Kind · LowerKind · Marks · classify · decide  (순수 함수)              모든 플랫폼
   merge_linux.go    Preflight · Apply — 디렉터리 fd 로 걷는 호출                            linux
   merge_other.go    Preflight · Apply 가 ErrUnsupported 를 돌려준다                        linux 밖
```

파일 이름과 나눔은 Code Generation 이 바꿀 수 있다. 지키는 것은 둘이다 — 판정(`classify` · `decide`)이 시스템 호출 없이
모든 플랫폼에서 빌드되고, 시스템 호출은 `_linux.go` 에만 있다 (`unit-of-work.md` 10절 · `runc_overlay_other.go` 선례).

---

## 1. 겉면 — `Paths` · `Options` · `Preflight` · `Apply`

```go
// Paths 는 합치기의 세 뿌리다. 셋은 노드 설정이 준 경로라 symlink 를 풀어 쓴다.
// 뿌리 아래에서는 symlink 를 따라가지 않는다 (답 7).
type Paths struct {
	Upper string // 대기 upper. 합치기가 그 안을 비운다. 뿌리 자신은 남긴다 — 치우는 것은 부르는 쪽이다
	Lower string
	Trash string // <scratch>/trash. 같은 filesystem 확인에만 쓴다. 옮기는 일은 Options.Discard 가 한다
}

// Options 는 부르는 쪽이 채우는 자리다.
type Options struct {
	// Discard 는 lower 쪽 항목 하나를 버린다. 인자는 절대 경로다 — 푼 lower 뿌리 + 상대 경로.
	// merge-helper 가 scratch.Trash.Move 로 채운다 (답 6 · trash 유닛이 넘긴 약속).
	// 비어 있으면 Apply 가 아무것도 바꾸지 않고 오류를 돌려준다.
	Discard func(lowerPath string) error

	// OnOp 는 호출 하나가 성공한 뒤마다 불린다. 오류를 돌려주면 Apply 가 거기서 멈추고 그 오류를 돌려준다.
	// 재개 시험이 N 번째 호출 뒤에 끊는 자리다 (답 2 · 조각 7). 비어 있어도 된다.
	OnOp func(Op) error
}

// Preflight 는 시작 전 확인이다 (FR-7). 하나라도 어긋나면 *PreflightError 를 돌려주고, 아무것도 바꾸지 않는다.
// 「이 lower 의 overlay 마운트 0」은 보지 않는다 — 부르는 쪽이 쥔 lower 배타 잠금이 그 증거다 (답 1).
func Preflight(p Paths) error

// Apply 는 upper 를 lower 에 합친다. 끊긴 뒤 다시 불러도 한 번에 끝낸 것과 같다 (결정 3-5).
// 첫 오류에서 멈추고, 그때까지 센 Result 를 오류와 함께 돌려준다 (답 9).
// Preflight 를 부르지 않는다 — 부르는 쪽이 처음과 재개 때 모두 Preflight 뒤에 부른다.
func Apply(ctx context.Context, p Paths, opt Options) (Result, error)
```

- **Apply 도 금지 표시를 본다.** 항목마다 표시를 읽어 판정하므로, 금지 표시를 만나면 그 항목에 손대기 전에 멈춘다
  (`business-rules.md` 2절). Preflight 를 건너뛰고 불러도 lower 를 망가뜨리지 않는다. 다만 그 앞의 항목은 이미 합쳐져
  있다 — 그래서 Preflight 가 먼저다
- **Application Design 과 다른 점 셋** (`component-methods.md` 2절).
  - `Classify(path string, fi fs.FileInfo)` 를 겉면에서 뺐다. 걷기가 디렉터리 fd 로 읽으므로 경로를 받는 판정은 쓰이지
    않는다. 판정은 안쪽 순수 함수 `classify` 다 (3절)
  - `Options` 에 `Discard` 를 더했다 (답 6)
  - `Result` 의 다섯 칸을 ADR-077 §12 의 표에 맞춰 나눴다 (2절)

---

## 2. 호출과 결과 — `Call` · `Op` · `Result` (답 2)

```go
// Call 은 lower 나 upper 를 바꾸는 호출 하나다. 재개 시험이 끊는 단위다 (답 2).
type Call int

const (
	CallDiscard     Call = iota + 1 // lower 쪽 항목을 버린다 (Options.Discard)
	CallUnlink                      // upper 의 whiteout 을 지운다
	CallClearOpaque                 // upper 디렉터리의 user.overlay.opaque 를 지운다
	CallRename                      // upper 항목을 lower 의 같은 경로로 옮긴다
	CallStashMtime                  // upper 디렉터리의 mtime 을 그 디렉터리의 xattr 로 적어 둔다 (5절)
	CallChown                       // lower 디렉터리의 소유를 맞춘다
	CallChmod                       // lower 디렉터리의 mode 를 맞춘다
	CallChtimes                     // lower 디렉터리의 mtime 을 맞춘다
	CallRmdir                       // 빈 upper 디렉터리를 지운다
)

// String 은 로그와 오류에 쓰는 이름이다 — "discard" · "unlink" · "clear-opaque" · "rename" · "stash-mtime" ·
// "chown" · "chmod" · "chtimes" · "rmdir".
func (c Call) String() string

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

	Discarded int // lower 쪽을 버린 수 (CallDiscard 가 성공한 수)
}
```

**ADR-077 §12 의 이름과의 대응.**

```text
   §12 의 이름            Result 칸
   기존 파일 대체          Replaced
   새 파일                Added
   새 디렉터리            NewDirs
   opaque                OpaqueDirs
   whiteout              Whiteouts
   trash                 Discarded
   속성을 맞춘 디렉터리     MergedDirs
   (§12 에 없음)          TypeChanged — 표에 없던 줄 (FR-7).  이 항목은 Replaced · NewDirs 에 겹쳐 세지 않는다
```

**§12 의 「연산 59,030」과 `Ops` 는 단위가 다르다.** 시제품은 항목 하나에 한 번(trash 옮기기만 따로 한 번) 셌다.
`Ops` 는 호출마다 센다 — 양쪽 디렉터리 하나가 stash-mtime · chown · chmod · chtimes · rmdir 다섯 번이다. 같은 하루치 upper
에서 `Ops` 가 §12 의 값보다 크게 나온다.

---

## 3. 판정 — `Kind` · `LowerKind` · `Marks` (답 7 · 8)

시스템 호출이 없는 순수 함수다. 모든 플랫폼에서 빌드되고, 기본 `go test` 의 표 시험이 덮는다.

```go
// Kind 는 upper 항목 하나의 종류다.
type Kind int

const (
	KindOther     Kind = iota + 1 // 파일 · symlink · 그 밖 (fifo · socket · 0/0 이 아닌 장치)
	KindDir                       // opaque 가 아닌 디렉터리
	KindOpaqueDir                 // user.overlay.opaque 의 값이 정확히 "y" 인 디렉터리
	KindWhiteout                  // 문자 장치 0/0
)

// LowerKind 는 같은 경로의 lower 쪽이다. lstat 으로 본다 — symlink 는 무엇을 가리키든 LowerOther 다 (답 7).
type LowerKind int

const (
	LowerAbsent LowerKind = iota + 1
	LowerDir
	LowerOther
)

// Marks 는 upper 항목에서 읽은 overlay 표시다. 이름은 정확히 맞춘다 — 앞부분(user.overlay.)으로 보지 않는다.
// escape 한 이름(user.overlay.overlay.*)은 세션이 붙인 보통 xattr 이라 여기 들지 않는다 (계획 1.7).
type Marks struct {
	Opaque   []byte // user.overlay.opaque 의 값. 없으면 nil
	Whiteout bool   // user.overlay.whiteout 이 있다
	Metacopy bool   // user.overlay.metacopy 가 있다
	Redirect bool   // user.overlay.redirect 가 있다
}

// classify 는 upper 항목 하나를 판정한다. mode 는 st_mode (종류 비트 포함), rdev 는 st_rdev 다.
// 금지 표시면 *MarkError 다 (business-rules.md 2절). root 는 upper 뿌리인가다 — 뿌리에는 opaque 가 있으면 안 된다.
func classify(mode uint32, rdev uint64, m Marks, root bool) (Kind, error)

// step 은 (upper 종류, lower 종류) 한 칸의 할 일이다. business-rules.md 1절의 표를 그대로 옮긴 것이다.
type step struct {
	discard bool  // 먼저 lower 쪽을 버린다
	clear   bool  // 그다음 upper 의 opaque 를 지운다
	last    Call  // CallRename · CallUnlink.  양쪽 디렉터리면 0 — 안으로 들어간다
	replace bool  // last 가 CallRename 일 때 대체를 허락하나 (파일 · symlink).  디렉터리를 옮길 때는 덮어쓰지 않는다
	tally   tally // Result 의 일곱 칸 중 어느 것
}

func decide(k Kind, l LowerKind) step
```

---

## 4. 오류 — 문구는 영어다 (`CONVENTIONS.md` 2.1)

```go
// Check 는 시작 전 확인의 어느 줄이 어긋났나다. 부르는 쪽(bake)이 이것으로 상태를 어디로 돌릴지 정한다.
type Check string

const (
	CheckPath       Check = "path"        // 절대 경로가 아니다 · 풀 수 없다
	CheckDirectory  Check = "directory"   // 진짜 디렉터리가 아니다 (없음 포함)
	CheckOverlap    Check = "overlap"     // 셋 중 하나가 다른 하나의 안에 있다
	CheckFilesystem Check = "filesystem"  // st_dev 가 셋이 같지 않다
	CheckMark       Check = "mark"        // upper 에 금지 표시가 있다
)

// PreflightError 는 시작 전 확인이 어긋난 까닭이다.
type PreflightError struct {
	Check Check
	Path  string // 뿌리의 경로, 또는 (mark 면) upper 기준 상대 경로
	Err   error  // mark 면 *MarkError, 그 밖에는 원래 오류나 설명
}

// MarkError 는 upper 항목에 있으면 안 되는 표시다.
type MarkError struct {
	Path  string // upper 기준 상대 경로. 뿌리면 "."
	Name  string // "user.overlay.metacopy" · "user.overlay.redirect" · "user.overlay.whiteout" · "user.overlay.opaque"
	Value []byte // opaque 의 값
}

// OpError 는 호출 하나가 실패한 자리다. Discard 가 돌려준 오류도 이것으로 감싼다.
type OpError struct {
	Op  Op
	Err error
}

// ErrUnsupported 는 linux 밖에서 Preflight 와 Apply 가 돌려준다.
var ErrUnsupported = errors.New("merge is supported on linux only")
```

- 셋 다 `Unwrap` 을 갖는다. 문구는 `business-rules.md` 10절이다
- Apply 가 걷다 만난 금지 표시는 `*MarkError` 를 그대로 돌려준다 (`merge: ` 를 앞에 붙인다). Preflight 는 같은 것을
  `*PreflightError{Check: CheckMark}` 로 감싼다

---

## 5. 재개가 쓰는 표지 — `user.enode.merge-mtime`

```text
   자리     양쪽에 있는 디렉터리의 upper 쪽 (그 디렉터리 자신)
   값       들어가기 전의 mtime.  10진 나노초 (예: 1727361234123456789)
   쓰는 때   안의 항목에 처음 손대기 전.  이미 있으면 쓰지 않고 그 값을 읽는다
   사라지는 때 그 디렉터리를 rmdir 할 때 — lower 로 가지 않는다
```

- 이름이 `user.overlay.` 로 시작하지 않는다 — overlay 표시가 아니고, 커널이 읽지 않는다. 시작 전 확인도 보지 않는다
- 왜 있나는 `business-rules.md` 5절이다 — 안의 항목을 옮기면 upper 디렉터리의 mtime 이 바뀐다. 적어 두지 않으면 재개 때
  맞출 값이 없다

---

## 6. 시험 모형 (답 3) — `_test.go` 안

```go
// entry 는 목록의 한 줄이다. 모형과 실제 트리를 같은 모양으로 적어 비교한다.
type entry struct {
	Path   string // 뿌리 기준 상대 경로
	Type   byte   // 'd' 디렉터리 · 'f' 파일 · 'l' symlink · 'o' 그 밖
	Mode   uint32 // 권한 비트 07777
	UID    uint32
	GID    uint32
	Size   int64  // 'f'
	Mtime  int64  // 'f' · 'd' — 나노초
	Target string // 'l'
	Ino    uint64 // 'f' · 'l' · 'o' — 옮겼지 복사하지 않았다는 증거. 커널의 merged view 와 댈 때는 뺀다
}

// view 는 합치기 전의 lower 와 upper 트리(표시 포함)로 overlay 가 보일 모습(merged view)을 낸다.
// Apply 와 다른 길로 짠다 — 호출을 흉내 내지 않고, 경로마다 「누가 보이나」를 정한다 (business-logic-model.md 4.1).
func view(lower, upper tree) []entry
```

- 모형은 경로마다 「upper 에 항목이 있나 · 그것이 whiteout 인가 · opaque 인가 · 양쪽이 디렉터리인가」만으로 보일 것을
  정한다. `decide` 를 부르지 않는다 — 같은 틀린 규칙을 두 번 쓰면 시험이 초록이 되기 때문이다
- 디렉터리 `Mtime` — 양쪽에 있는 디렉터리는 upper 쪽의 합치기 전 값, 한쪽에만 있는 디렉터리는 그쪽 값

---

## 7. linux 와 그 밖

```text
   linux       Preflight · Apply 의 시스템 호출 (x/sys/unix)
                 걷기        Openat (O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC) · Fstatat (AT_SYMLINK_NOFOLLOW) · 이름 읽기
                 옮기기      Renameat (파일 · symlink — 대체) · Renameat2 (RENAME_NOREPLACE — 디렉터리) · Unlinkat
                 표시        디렉터리는 연 fd 로 Flistxattr · Fgetxattr · Fremovexattr · Fsetxattr
                            디렉터리가 아닌 항목은 /proc/self/fd/<부모 fd>/<이름> 에 Llistxattr · Lgetxattr
                 속성        Fchown · Fchmod · UtimesNanoAt (부모 fd · 이름 · AT_SYMLINK_NOFOLLOW · atime 은 UTIME_OMIT)
   linux 밖     ErrUnsupported
   32비트 arm   Stat_t 의 Timespec 칸이 int32 이고 Rdev · Dev 의 타입이 플랫폼마다 다르다.
               unix.TimespecToNsec · NsecToTimespec 로 옮기고, rdev 는 uint64 로 바꿔 classify 에 넘긴다
```

- `getxattrat` 같은 fd 기준 xattr 호출(커널 6.13 부터)에는 기대지 않는다 — CI 의 커널과 이 기계(6.5)가 그보다 낮다
- `/proc/self/fd/<부모 fd>` 는 부모 디렉터리로 곧장 이어지는 링크라 경로를 이어 붙이지 않는다. helper 의 마운트 namespace
  에도 `/proc` 가 있다 (`unshare --mount` 는 호스트의 마운트를 물려받는다)
