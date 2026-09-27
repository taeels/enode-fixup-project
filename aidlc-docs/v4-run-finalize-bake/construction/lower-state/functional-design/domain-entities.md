# `lower-state` — 도메인 엔티티

같은 기계의 형제 노드가 한 아래층(lower)을 함께 쓰기 위한 타입과 칸이다. 입력은 계획의 답 아홉과 3절(묻지 않고 정한 것)
(`construction/plans/lower-state-functional-design-plan.md`)이다. 규칙과 문구는 `business-rules.md`, 흐름과 시험은
`business-logic-model.md` 에 있다.

```text
   답   1 A   공유 잠금은 drain 이 받아 적히고 임대가 0 인 응답을 두 번 연속 받으면 놓는다.  놓은 뒤에 보인 임대는
              공유를 다시 잡고, 그 사이 lower 가 바뀌었으면 그 Run 의 단계를 lower_changed 로 실패시킨다
        2 A   실행 중 획득(acquire)이 drain 을 본다 — internal/store/acquire.go 한 줄 (파일 행렬 밖 · Mediator)
        3 A   쥔 사람 기록은 노드마다 잠금 파일과 기록 파일 한 쌍.  잠금이 쥐어져 있으면 살아 있는 기록이다
        4 A   배타 잠금이 「마운트 0」의 증거 (그대로).  배타를 잡은 뒤 같은 uid 의 마운트를 훑는 그물을 더하고,
              한 lower 의 노드를 모두 새 판으로 올린 뒤에 굽는다는 규칙을 둔다
        5 A   준비도 점검의 smoke 도 공유 잠금을 잡는다.  합치는 중이면 기다린다
        6 A   scratch 와 워크스페이스는 st_dev 와 마운트까지 같아야 ready.  merge Preflight 의 같은 확인은 bake 에 넘긴다
        7 A   lower.json 에 생성 시각(btime)을 더해 inode 재사용을 잡는다
        8 A   scratch 의 filesystem 과 소유 uid 는 invalid, lower.json 신원은 external-blocked.  host 점검 뒤 · smoke 앞
        9 A   상태 자리(lowers)는 ENODE_STATEDIR 를 따르지 않는다 — 늘 $HOME/.local/state/enode/lowers
```

**새 패키지는 하나다** — `internal/lower` (Application Design 의 자리 · `components.md` 2.1). 표준 라이브러리와
`golang.org/x/sys` 만 쓴다. `internal/scratch` · `internal/merge` · `internal/environment` 를 임포트하지 않는다. Mediator 는 이
패키지를 링크하지 않는다.

```text
   internal/lower
     lower.go           Key · Root · Identity · Phase · State · Owner · LastAttempt · Holder · Role ·           모든 플랫폼
                        Metadata · Marker · Finding · Cause · 오류 · 패키지 문서
     judge.go           키 문자열 · 신원 판정 · mountinfo 풀기와 overlay 찾기 · 점검 판정 (순수 함수)           모든 플랫폼
     root_linux.go      ReadRoot (statfs · statx)                                                           linux
     dir_linux.go       Open · Peek · 상태와 lower.json 과 metadata 읽고 쓰기                                  linux
     lock_linux.go      Shared · Exclusive · Bake · Holders · WaitShared                                   linux
     mounts_linux.go    ForeignMounts (/proc 훑기)                                                          linux
     check_linux.go     Check (점검 셋)                                                                     linux
     lower_other.go     ErrUnsupported                                                                     linux 밖

   internal/enode       lowerguard.go (새) · advertise.go · leases.go · claim.go · runtime.go · policy.go · paths.go ·
                        runc_overlay_linux.go · runc_overlay_other.go
   그 밖                internal/environment/check.go · internal/panel/page.go · internal/panel/boundary_test.go ·
                        cmd/enode/main.go · cmd/enode/environment.go · internal/contract/result.go ·
                        internal/store/acquire.go · internal/store/claim.go
```

파일 이름과 나눔은 Code Generation 이 바꿀 수 있다. 지키는 것은 둘이다 — 판정(`judge.go`)은 시스템 호출 없이 모든 플랫폼에서
빌드되고, 시스템 호출은 `_linux.go` 에만 있다 (`unit-of-work.md` 10절). 파일 행렬 밖의 파일은 `business-logic-model.md` 11절이다.

---

## 1. 신원 — `Key` · `Root`

```go
// Key 는 lower 루트 디렉터리의 신원이다. statfs 의 f_fsid 와 inode (결정 3-14 · bind 별칭도 같은 키).
type Key struct {
	// FSID 는 f_fsid.Val[0]<<32 | Val[1] 이다. %016x 로 찍으면 `stat -f -c %i` 와 같은 글자가 된다 (계획 2.2).
	// Val 은 int32 둘이라 uint32 로 바꾼 뒤 붙인다.
	FSID uint64
	Ino  uint64
}

// String 은 상태 자리의 이름이다 — "3b167980fdd76b7b-2905094" (16 자리 16진 · 10진 inode).
func (k Key) String() string
func ParseKey(s string) (Key, error)

// Root 는 lower 루트를 한 번 읽은 값이다. statfs 한 번과 statx 한 번이다.
type Root struct {
	Path    string // symlink 를 푼 경로 (filepath.EvalSymlinks)
	Key     Key
	BirthNs int64  // statx 의 btime.  0 이면 이 filesystem 이 알려 주지 않았다 — 신원 대조를 건너뛴다 (답 7)
	Dev     uint64 // st_dev (major:minor)
	MountID uint64 // statx 의 STATX_MNT_ID.  0 이면 커널이 안 줬다 — /proc/self/mountinfo 에서 찾는다 (답 6)
	UID     int    // 소유 uid
}

func ReadRoot(path string) (Root, error)
```

- `Key.FSID` 의 타입은 Application Design 과 같다 (`uint64`). 두 반쪽을 붙이는 차례를 정했다 — 반대로 붙이면 `stat -f` 와 순서가
  뒤집혀 사람이 자리를 못 찾는다 (계획 2.2 의 `fdd76b7b3b167980`)
- `Root` 는 새 타입이다. 키 · 점검 셋 · 마운트 훑기가 같은 한 번의 읽기에서 값을 얻는다

---

## 2. 상태 자리 — `Dir` · `Identity` (답 7 · 9)

```go
// Dir 은 <lowers>/<key>/ 다. lowers 는 부르는 쪽이 준다 — 제품은 LowersDir(), 시험은 임시 폴더 (답 9).
type Dir struct {
	Root Root
	Path string
}

// Open 은 노드 데몬이 부른다 (기동 · 못 열었으면 광고 주기마다 다시). 자리 · holders/ (0700) 와
// lower.lock · bake.lock (0600) 을 만들고, lower.json 을 대조한 뒤 쓴다 (business-rules.md 2절).
func Open(lowers string, root Root, now time.Time) (*Dir, error)

// Peek 은 env check 와 smoke 가 부른다. 아무것도 만들지 않는다 (계획 3.2). 자리가 없으면 nil, nil 이다.
func Peek(lowers string, root Root) (*Dir, error)

// Identity 는 lower.json 이다.
type Identity struct {
	Schema   int       `json:"schema"`   // 1
	FSID     string    `json:"fsid"`     // Key.FSID 의 16 자리 16진 — stat -f -c %i 와 같은 글자
	Ino      uint64    `json:"ino"`
	BirthNs  int64     `json:"birth_ns"` // 답 7.  0 이면 모름
	Paths    []string  `json:"paths"`    // 이 lower 를 본 경로들 (별칭).  정보다 — 대조하지 않는다
	OwnerUID int       `json:"owner_uid"`
	SeenAt   time.Time `json:"seen_at"`  // 마지막으로 쓴 때 (노드 시계)
}

// Verdict 는 lower.json 과 지금 읽은 뿌리를 댄 결과다. 순수 함수 identityVerdict 가 낸다.
type Verdict int

const (
	VerdictNew     Verdict = iota + 1 // lower.json 이 없다 — 처음 본 lower
	VerdictMatch                      // 같다.  경로가 새면 Paths 에 더한다
	VerdictReused                     // btime 이 다르고 상태가 committed — 같은 inode 번호의 다른 디렉터리.  새로 쓴다
	VerdictForeign                    // btime 이 다르고 상태가 committed 가 아니다 — 끊긴 굽기가 다른 디렉터리의 것
	VerdictBroken                     // 못 읽는다 · fsid 나 ino 가 자리 이름과 다르다
)

func identityVerdict(rec *Identity, root Root, phase Phase) Verdict
```

- **JSON 의 fsid 를 16진 글자로 적는다.** Application Design 은 `uint64` 숫자였다. 사람이 `stat -f -c %i` 와 대 보는 값이고,
  JSON 숫자로 두면 53비트를 넘는 값을 자바스크립트(제어판)가 틀리게 읽는다
- **`BirthNs` 를 더했다** (답 7). 둘 중 하나라도 0 이면 대조하지 않는다

---

## 3. 상태 — `Phase` · `State` (계획 3.1)

```go
type Phase string

const (
	PhaseCommitted Phase = "committed"
	PhaseBuilding  Phase = "building"
	PhasePending   Phase = "pending"
	PhaseMerging   Phase = "merging"
)

// State 는 state.json 이다. lower 밖에 둔다 (결정 3-15). 칸은 Application Design 그대로다.
type State struct {
	Schema       int          `json:"schema"`
	Phase        Phase        `json:"phase"`
	Owner        *Owner       `json:"owner,omitempty"`         // committed 가 아닐 때
	PendingUpper string       `json:"pending_upper,omitempty"` // pending · merging 일 때 대기 upper 의 자리
	Since        time.Time    `json:"since"`                   // 이 phase 가 된 때 (노드 시계)
	LastAttempt  *LastAttempt `json:"last_attempt,omitempty"`  // 실패한 굽기 (결정 3-18)
}

type Owner struct {
	Run      string `json:"run"`
	Step     int    `json:"step"`
	Node     string `json:"node"`
	Instance string `json:"instance"`
}

type LastAttempt struct {
	Run    string        `json:"run"`
	At     time.Time     `json:"at"`
	Reason string        `json:"reason"`
	Builds []BuildRecord `json:"builds,omitempty"`
}

// ReadState 는 누구나 부른다 — 잠금 없이 읽는다 (계획 2.5).  파일이 없으면 committed 다.
func (d *Dir) ReadState() (State, error)

// WriteState 는 굽기 잠금을 쥔 쪽만 부른다 — 그래서 *Bake 의 메서드다.
func (b *Bake) WriteState(s State) error
```

- **`WriteState` 가 `*Dir` 에서 `*Bake` 로 옮겼다** (Application Design 과 다른 점). state.json 을 쓰는 쪽이 굽기 잠금의 주인
  하나라는 규칙(계획 3.1)을 타입이 지킨다
- 파일이 없으면 committed 로 읽는다 — 굽기가 한 번도 없었던 lower 다

---

## 4. 잠금 — `Shared` · `Exclusive` · `Bake` · `Holder` (답 1 · 3 · 5)

```go
type Role string

const (
	RoleCandidate Role = "candidate" // drain 없이 광고하는 중 — 매칭 후보
	RoleRun       Role = "run"       // prepare 가 아닌 Run 의 임대를 쥐었다
)

// Holder 는 holders/<node_id>.json 이다 (답 3). holders/<node_id>.lock 을 배타로 쥔 동안만 살아 있는 기록이다.
type Holder struct {
	Node  string    `json:"node"`          // node_id
	Label string    `json:"label"`         // 사람이 읽는 이름 — 대기 로그에 쓴다
	Role  Role      `json:"role"`
	Run   string    `json:"run,omitempty"` // RoleRun 일 때
	Since time.Time `json:"since"`         // 이 역할이 된 때 (노드 시계)
	Acks  int       `json:"acks"`          // drain 이 받아 적히고 임대가 0 인 응답을 연속 몇 번 받았나 (0 · 1) — 답 1
	PID   int       `json:"pid"`
}

// TryShared 는 lower.lock 을 공유로 잡고 기록을 남긴다. 막히지 않는다 — 배타가 쥐어져 있으면 false 다.
func (d *Dir) TryShared(h Holder) (*Shared, bool, error)
func (s *Shared) Update(h Holder) error // 역할 바꾸기 — 기록만 다시 쓴다
func (s *Shared) Release() error

// WaitShared 는 smoke 가 쓴다 (답 5). 기록을 남기지 않는다. 배타가 쥐어져 있으면 every 마다 다시 보며 ctx 까지
// 기다리고, 기다리는 동안 notice 에 state.json 을 넘긴다 (누가 합치나를 한 줄로 쓰게).
func (d *Dir) WaitShared(ctx context.Context, every time.Duration, notice func(State)) (*Shared, error)

// Waiting 은 배타를 기다리는 동안 watch 가 받는 것이다.
type Waiting struct {
	Holders []Holder // 살아 있는 기록
	Unnamed bool     // 잠금이 막혀 있는데 살아 있는 기록으로 다 설명되지 않을 수 있다 — 기록이 0 인데 막혔다
}

// Exclusive 는 lower.lock 의 배타를 ctx 가 끝날 때까지 기다린다. 1초마다 LOCK_EX|LOCK_NB 를 다시 걸고,
// every 마다 watch 를 부른다. 마감이면 ctx.Err() 를 돌려준다.
func (d *Dir) Exclusive(ctx context.Context, every time.Duration, watch func(Waiting)) (*Exclusive, error)
func (e *Exclusive) Release() error

// TryBake 는 bake.lock 을 배타로 잡는다. 주인이 살아 있으면 false 다 (결정 3-12).
func (d *Dir) TryBake() (*Bake, bool, error)
func (b *Bake) Release() error

// Holders 는 살아 있는 기록만 돌려준다 (business-rules.md 7절).
func (d *Dir) Holders() ([]Holder, error)
```

- **Application Design 과 다른 점.** `Exclusive` 가 기다리는 주기(`every`)를 받고 `watch` 가 `Waiting` 을 받는다 — 기록 없는
  쥔 쪽(smoke · 옛 판의 프로세스)이 있을 수 있어서다. `Holder` 에 `Label` · `Acks` · `PID` 를 더했다 — `Acks` 로 대기 로그가
  「그 후보가 언제 놓나」를 말한다. `WaitShared` 는 새 메서드다
- 모든 잠금 파일과 기록 파일은 `O_CLOEXEC` 로 연다 (계획 2.1 여섯째 줄)

---

## 5. metadata — `Metadata` · `Marker`

```go
// Metadata 는 <lower>/.enode-metadata.json 이다 (결정 3-16).  칸은 component-methods.md 1.4 그대로다
// (Source · Pinned · BuildRecord · BakeRecord).  이 유닛은 읽기를 쓰고, 쓰기는 bake 가 부른다.
func ReadMetadata(lowerRoot string) (*Metadata, error) // 없으면 nil, nil
func WriteMetadata(lowerRoot string, m Metadata) error

// Marker 는 lower 의 내용이 마지막 합치기 뒤로 그대로인지 보는 표지다 (답 1).  metadata 가 없으면 제로값이다.
type Marker struct {
	Run      string
	MergedAt time.Time
}

func (m *Metadata) Marker() Marker // nil 수신자는 제로값
```

---

## 6. 마운트 훑기 — `ForeignMounts` (답 4)

```go
// Mount 는 이 lower 를 아래층으로 쓰는 overlay 마운트 하나다 — 다른 프로세스의 마운트 namespace 에서 찾았다.
type Mount struct {
	PID        int    // 그 namespace 를 처음 읽은 프로세스
	Namespace  string // /proc/<pid>/ns/mnt 의 링크 값 — mnt:[4026532...]
	MountPoint string // overlay 의 마운트 자리 (그 namespace 안의 경로)
	LowerDir   string // lowerdir 중 이 lower 에 닿는 것
}

type MountScan struct {
	Found      []Mount
	Namespaces int // 읽은 마운트 namespace 수
	Unreadable int // 권한 때문에 못 읽은 같은 uid 프로세스 (dumpable 이 꺼진 것) — 그물의 빈틈
}

// ForeignMounts 는 같은 uid 의 프로세스를 마운트 namespace 마다 한 번씩 읽어 이 lower 에 닿는 overlay 를 찾는다.
// 증거가 아니라 그물이다 — 0 이어도 배타 잠금 없이 합치지 않는다 (business-rules.md 8절).
func ForeignMounts(root Root) (MountScan, error)

// 판정은 순수 함수다 — mountinfo 한 벌을 풀고 (dev, fs 안의 경로) 로 이 lower 에 닿는 overlay 를 고른다.
func parseMountinfo(b []byte) ([]mountLine, error)
func overlaysOn(lines []mountLine, dev uint64, pathInFS string) []Mount
```

---

## 7. 점검 — `Finding` (답 6 · 7 · 8)

```go
// Cause 는 어긋난 까닭의 갈래다. internal/enode 가 environment.State 로 옮긴다 (답 8).
type Cause string

const (
	CauseBinding Cause = "binding" // 설정이 쓸 수 없는 자리를 가리킨다 -> invalid
	CauseState   Cause = "state"   // profile 밖에 남은 상태가 막는다 -> external-blocked
)

// Finding 은 environment.Fact 로 옮겨질 결과다. 이 패키지는 environment 를 모른다 (components.md 2.1).
type Finding struct {
	Name        string // "binding.scratch_filesystem" | "lower.owner_uid" | "lower.identity"
	Required    string
	Observed    string
	OK          bool
	Cause       Cause  // OK 가 거짓일 때
	Remediation string
}

// Check 는 점검 셋이다. 아무것도 만들지 않는다 (Peek).
func Check(lowers, lowerRoot, scratch string, uid int) []Finding
```

- **`Cause` 와 `Remediation` 을 더했다.** State 는 `internal/environment` 의 어휘라 이 패키지가 모른다. 까닭의 갈래만 내고
  옮기는 쪽이 State 를 고른다

---

## 8. `internal/enode` — 런타임 능력 · drain 출처 · 원인 코드

```go
// RuntimeCapability 는 런타임이 광고에 내는 것이다 (component-methods.md 4.1).
type RuntimeCapability struct {
	Writes string // "isolated" | "in-place" — 광고 workspace.writes
	// 보존 지원(Capture)은 checkpoint 유닛이 더한다
}

// StepRuntime 에 메서드 하나를 더한다.  NativeRuntime 은 in-place, RuncOverlayRuntime 은 isolated.
//   Capability() RuntimeCapability
// 런타임이 없는 노드(environment 블록 없음)는 in-place 로 광고한다 (ADR-077 §8).

// drain 출처의 새 Kind 둘 (policy.go 의 DrainOwner · DrainDisk 옆)
const (
	DrainBake  = "bake"  // 이 lower 의 굽기가 pending · merging 이거나 합치는 중이라 공유를 못 잡았다
	DrainLower = "lower" // 상태 자리를 못 열었거나 state.json 을 못 읽었다
)

// 광고 키 (예약 — 라벨이 못 덮는다)
const (
	KeyWorkspaceWrites = "workspace.writes"
	KeyIR              = "ir"
	KeyRepoBuilt       = "repo.built." // 뒤에 구성 이름
	KeyBakeRun         = "bake.run"
	KeyBakeResumed     = "bake.resumed" // "true" | "false"
)

// LowersDir 는 $HOME/.local/state/enode/lowers 다 (paths.go · 답 9). ENODE_STATEDIR 를 따르지 않는다.
func LowersDir() (string, error)
```

- **`Capability()` 를 세션이 아니라 런타임에 둔다** (Application Design 과 다른 점 — `component-methods.md` 4.1 은 `StepSession`
  에 두었다). 광고는 세션 밖에서 돈다

`internal/contract/result.go` 의 원인 코드에 하나를 더한다 (답 1).

```go
ReasonLowerChanged = "lower_changed" // 놓은 뒤에 보인 임대의 Run 이 바뀐 lower 위에서 돌 뻔했다
```

같은 패키지의 `bake.go` 에 있는 구성 이름 판정(`buildName`)과 IR 판정(`irProblem`)을 내보낸다 — metadata 에서 읽은 이름과 IR 을
계약과 같은 규칙으로 거른다 (`business-rules.md` 11절). 규칙을 두 벌로 두지 않는다.

```go
func ValidBuildName(n string) bool // buildName 그대로
func IRProblem(ir string) string   // irProblem 그대로 — "" 면 괜찮다
```

---

## 9. `LowerGuard` — 후보 잠금 (답 1 · Application Design Q1)

```go
// LowerGuard 는 이 노드의 공유 잠금을 쥐고 놓는다. Advertiser(광고 주기)와 Worker(claim)가 나눠 쓴다.
// runc-overlay 노드만 만든다 — 그 밖의 노드는 nil 이고 nil 은 아무것도 안 한다.
type LowerGuard struct {
	/* mu · lowers · root · node · label · log · now
	   dir *lower.Dir · openErr error                 못 열었으면 광고 주기마다 다시 연다
	   shared *lower.Shared                          쥐고 있나
	   streak int                                    drain 이 받아 적히고 임대가 0 인 응답이 연속 몇 번
	   released bool · mark lower.Marker             놓았나 · 놓을 때의 표지 (답 1)
	   refused map[string]bool                       lower_changed 로 실패시킬 Run
	   stepping int                                  도는 단계 수 — 0 이 아니면 놓지 않는다
	   bake *bakeHold                                이 프로세스가 쥔 굽기 잠금의 Run 과 치우는 몸통 (bake 가 등록) */
}

// BeforeAdvert 는 광고 직전이다. others 는 소유자 · 여유 부족 출처다. 이 노드가 더할 출처(bake · lower)와
// 광고 키(ir · repo.built.* · bake.*)를 돌려준다. 실을 drain 이 없으면 여기서 공유를 잡는다.
func (g *LowerGuard) BeforeAdvert(others []DrainSource) (own []DrainSource, keys map[string]string)

// AfterResponse 는 광고 응답 뒤다. 울타리를 세고, 놓은 뒤에 보인 임대를 다루고, 굽기 Run 의 임대가 사라졌는지 본다.
func (g *LowerGuard) AfterResponse(drain string, leases []Lease)

// OnClaim 은 claim 한 단계를 돌리기 전이다. prepare 단계(와 merge 단계)면 공유를 놓는다 (결정 3-9).
// 거절할 Run 이면 *LowerChangedError 를 돌려준다 — Worker 가 그 단계를 lower_changed 로 보고한다.
func (g *LowerGuard) OnClaim(step *Step) error

// StepDone 은 Worker 가 단계의 세션을 닫은 뒤 부른다 — OnClaim 과 짝이다.
func (g *LowerGuard) StepDone(step *Step)

// HoldBake · DropBake 는 bake 유닛이 부른다. 쥔 굽기 잠금의 Run 과, 합치기 없이 끝났을 때 치우는 몸통이다 (business-rules.md 10절).
func (g *LowerGuard) HoldBake(run string, abandon func())
func (g *LowerGuard) DropBake()
```

- **Application Design 과 다른 점.** `BeforeAdvert` 가 소유자 정책과 여유를 직접 받지 않고 이미 만든 출처 목록을 받는다 —
  trash 유닛이 그 합치기를 `Advertiser.drain` 에 두었다. `OnClaim` 이 오류를 돌려주고, `StepDone` · `HoldBake` · `DropBake` 를
  더했다

---

## 10. `internal/environment` — 이음매 (답 8)

```go
// FactSource 는 verifier 가 더 낼 Fact 가 있을 때 구현한다 (component-methods.md 6절 그대로).
type FactSource interface {
	Facts(context.Context, Document, Binding) []Fact
}
```

`internal/enode` 의 `ExecutionRuntimeVerifier` 가 구현한다. runc-overlay 가 아니면 nil 을 돌려준다. 같은 verifier 에 칸 하나를
더한다 — `Notice io.Writer` (smoke 가 합치기를 기다린다는 줄을 쓰는 자리 · nil 이면 안 쓴다).

---

## 11. linux 와 그 밖

```text
   linux        statfs · statx (STATX_INO | STATX_BTIME | STATX_MNT_ID) · flock (LOCK_SH · LOCK_EX · LOCK_NB) ·
                /proc/<pid>/ns/mnt · /proc/<pid>/mountinfo · /proc/self/mountinfo
   linux 밖      ErrUnsupported.  LowerGuard 는 만들지 않는다 (runc-overlay 가 linux 만이다)
   32비트 arm    Statfs_t.Fsid.Val 은 [2]int32 · Stat_t.Ino · Statx_t.Ino 는 uint64 (계획 2.10).  uint32 로 바꿔 붙인다
```

- `STATX_MNT_ID` 는 커널 5.8 부터다. 없으면 `/proc/self/mountinfo` 에서 경로가 가장 길게 겹치는 마운트를 찾는다 (답 6)
- btime 을 안 주는 filesystem 은 `BirthNs` 가 0 이다 — 신원 대조를 건너뛴다 (답 7)
