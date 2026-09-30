# 컴포넌트 겉면 — 메서드와 타입

**시그니처와 입출력 타입까지다.** 규칙(전이 · 경계 · 순서의 세부)은 Functional Design 이
유닛마다 닫는다. 이름은 코드가 굳힐 때 바뀔 수 있고, 바뀌면 이 문서가 아니라 유닛의
code 요약이 새 이름을 진다.

- **작성 시각**: 2026-09-24T06:33:21Z · 답 일곱(전부 A)을 딛는다

주석은 한국어, 문자열 값은 영어다 (CONVENTIONS 2절).

---

## 1. `internal/lower`

### 1.1 신원과 상태 자리

```go
// Key 는 lower 루트 디렉터리의 신원이다. statfs 의 f_fsid 와 inode.
// bind 별칭도 같은 키가 된다 (결정 3-14).
type Key struct {
	FSID uint64 // f_fsid.Val[0]<<32 | Val[1] — %x 가 stat -f -c %i 와 같다 (lower-state FD · 앞의 0 없이)
	Ino  uint64
}

func KeyOf(root string) (Key, error)
func (k Key) String() string // "<fsid>-<ino>"
// (lower-state 코드 · 2026-09-27 — KeyOf 대신 ReadRoot(path) (Root, error) 가 키 · btime · 장치 · 마운트 번호 · 주인을
//  한 번에 읽는다.  ParseKey 는 String 이 내는 글자만 받는다)

// Dir 은 ~/.local/state/enode/lowers/<key>/ 다. 노드 사용자 전용 권한 (5.3).
type Dir struct {
	Lower string // 이 노드가 본 lower 루트 경로
	Key   Key
	Path  string // 상태 자리
}

func Open(stateRoot, lowerRoot string) (*Dir, error) // 자리를 만들고 lower.json 에 경로를 더한다

// Identity 는 lower.json 이다. 신원 대조용.
// (lower-state FD 답 7 · 2026-09-26 고침 — fsid 는 16진 글자, btime 을 더했다.
//  Dir · Open 의 겉면도 바뀌었다 — 그 FD 의 domain-entities.md 1 · 2절)
type Identity struct {
	Schema   int       `json:"schema"`
	FSID     string    `json:"fsid"`     // 16진 (앞의 0 없이) — stat -f -c %i 와 같은 글자
	Ino      uint64    `json:"ino"`
	BirthNs  int64     `json:"birth_ns"` // statx btime — inode 재사용을 잡는다
	Paths    []string  `json:"paths"`
	OwnerUID int       `json:"owner_uid"`
	SeenAt   time.Time `json:"seen_at"`
}

func (d *Dir) Identity() (Identity, error)
// (lower-state 코드 — Open(lowers, root, now) · Peek(lowers, root) 는 FD 그대로.  Dir 에 Notes (Open 이 고친 것 —
//  좁힌 권한 · 새로 쓴 신원) 와 Loose (Peek 이 본 느슨한 비트) 를 더했다 — 이 패키지는 로그를 모르고 부르는 쪽이 쓴다)
```

### 1.2 상태 파일

```go
type Phase string

const (
	PhaseCommitted Phase = "committed"
	PhaseBuilding  Phase = "building"
	PhasePending   Phase = "pending"
	PhaseMerging   Phase = "merging"
)

type Owner struct {
	Run      string `json:"run"`
	Step     int    `json:"step"`
	Node     string `json:"node"`
	Instance string `json:"instance"`
}

// State 는 state.json 이다. lower 밖에 둔다 (결정 3-15).
type State struct {
	Schema       int          `json:"schema"`
	Phase        Phase        `json:"phase"`
	Owner        *Owner       `json:"owner,omitempty"`
	PendingUpper string       `json:"pending_upper,omitempty"` // 대기 upper 의 자리
	Since        time.Time    `json:"since"`
	LastAttempt  *LastAttempt `json:"last_attempt,omitempty"`   // 실패한 굽기 (결정 3-18)
}

type LastAttempt struct {
	Run    string        `json:"run"`
	At     time.Time     `json:"at"`
	Reason string        `json:"reason"`
	Builds []BuildRecord `json:"builds,omitempty"`
}

func (d *Dir) ReadState() (State, error)
func (b *Bake) WriteState(State) error // 굽기 잠금의 주인만 쓴다.  임시 파일 · fsync · rename (lower-state FD 에서 *Dir 에서 옮겼다)
```

### 1.3 잠금과 쥔 사람 기록 (Q1 · Q4)

```go
// Holder 는 공유 잠금 옆의 기록이다. 살아 있는 기록만 읽힌다.
type Holder struct {
	Node  string    `json:"node"`
	Role  string    `json:"role"` // "candidate" | "run"
	Run   string    `json:"run,omitempty"`
	Since time.Time `json:"since"`
}

type Shared struct{ /* lower.lock 의 공유 flock 과 기록 */ }

func (d *Dir) TryShared(h Holder) (*Shared, bool, error) // 막히지 않는다. 합치기 중이면 false
func (s *Shared) Update(h Holder) error                   // 후보 -> Run 역할 바꾸기
func (s *Shared) Release() error

type Exclusive struct{ /* lower.lock 의 배타 flock */ }

// Exclusive 는 ctx 의 마감(merge 대기 상한)까지 기다린다.
// watch 는 기다리는 동안 주기마다 불린다 — 쥔 사람을 로그로 쓰는 자리 (Q4).
// (lower-state FD 고침 — every 를 받고 watch 가 Waiting{Holders, Unnamed} 를 받는다.  Holder 에 Label · Acks · PID)
func (d *Dir) Exclusive(ctx context.Context, every time.Duration, watch func(Waiting)) (*Exclusive, error)
func (e *Exclusive) Release() error

type Bake struct{ /* bake.lock 의 배타 flock */ }

func (d *Dir) TryBake() (*Bake, bool, error) // 주인이 살아 있으면 false (결정 3-12)
func (b *Bake) Release() error

func (d *Dir) Holders() ([]Holder, error)
// (lower-state 코드 — func (s *Shared) Recorded() bool 을 더했다.  node_id 가 파일 이름이 될 수 없거나 같은 이름의 기록
//  잠금을 남이 쥐면 기록 없이 공유만 쥔다 · WaitShared(ctx, every, notice) 는 기록 없는 공유다)
```

### 1.4 metadata

```go
// Metadata 는 <lower>/.enode-metadata.json 이다 (결정 3-16).
// 합치기 창 안에서, 합치기의 마지막 동작으로만 쓴다.
type Metadata struct {
	Schema          int           `json:"schema"`
	Source          Source        `json:"source"`
	Builds          []BuildRecord `json:"builds"`
	Environment     string        `json:"environment"`
	WorkspaceTarget string        `json:"workspace_target"`
	Bake            BakeRecord    `json:"bake"`
}

type Source struct {
	URL         string    `json:"url"`
	Branch      string    `json:"branch"`
	RepoID      string    `json:"repo_id"`
	Head        string    `json:"head"`
	IR          *string   `json:"ir"`                   // 없으면 null
	Pinned      *Pinned   `json:"pinned"`
	SyncCommand string    `json:"sync_command"`
	SyncedAt    time.Time `json:"synced_at"`
}

type Pinned struct {
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
}

type BuildRecord struct {
	Name       string    `json:"name"`
	Command    string    `json:"command"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	ExitCode   int       `json:"exit_code"`
}

type BakeRecord struct {
	Run        string    `json:"run"`
	Node       string    `json:"node"`
	MergedAt   time.Time `json:"merged_at"`
	Resumed    bool      `json:"resumed"`
	PreviousIR *string   `json:"previous_ir"`
}

func ReadMetadata(lowerRoot string) (*Metadata, error) // 없으면 nil 과 nil
func WriteMetadata(lowerRoot string, m Metadata) error
```

처음 판은 여기에 `ir_reason`(IR 을 유도하지 못한 이유)을 더했다. Units Generation 2.3 이
IR 을 계약이 적게 바꿔 없앴다 — build 가 대조에 성공해야 합치므로 `ir` 은 늘 계약의 값이다.
어긋난 이유(HEAD 의 태그와 커밋)는 build 단계 결과가 담는다 (완료 조건 10).

### 1.5 env check 의 확인 셋

```go
// Finding 은 environment.Fact 로 옮겨질 결과다. 이 패키지는 environment 를 모른다.
type Finding struct {
	Name     string // "binding.scratch_filesystem" | "lower.owner_uid" | "lower.identity"
	Required string
	Observed string
	OK       bool
	// (lower-state FD 답 8 · 더함) Cause — binding 이면 invalid, state 면 external-blocked 로 옮긴다.  Remediation
}

func Check(stateRoot, lowerRoot, scratch string, uid int) []Finding
// (lower-state 코드 — 첫 인자는 상태 자리의 뿌리 lowers 다.  "" 면 lower.identity 를 안 낸다.  home 은 부르는 쪽
//  (internal/enode 의 Facts) 이 LowersDir 로 찾고, 못 찾으면 그쪽이 lower.identity 를 external-blocked 로 낸다)
```

---

## 2. `internal/merge`

```go
type Paths struct {
	Upper string // 대기 upper
	Lower string
	Trash string // <scratch>/trash — lower 와 같은 filesystem 이어야 한다
}

// Preflight 는 시작 전 확인이다. 하나라도 어긋나면 시작하지 않는다 (FR-7).
// 「이 lower 의 마운트 0」은 부르는 쪽이 쥔 배타 잠금이 증거를 진다.
func Preflight(p Paths) error

type Kind int // 파일 · symlink · 새 디렉터리 · 양쪽 디렉터리 · whiteout · opaque · 종류가 바뀐 항목

type Op struct {
	Kind Kind
	Path string // upper 기준 상대 경로
}

type Result struct {
	Renamed     int
	Merged      int // 안으로 들어간 디렉터리
	Whiteouts   int
	Opaques     int
	TypeChanged int
}

// Options.OnOp 는 연산마다 불린다. 오류를 돌려주면 거기서 멈춘다.
// 시험이 1 ~ 30번째 연산 뒤 끊고 다시 돌리는 자리다 (5.6 · 조각 7).
type Options struct {
	OnOp func(Op) error
}

// Apply 는 끊긴 뒤 다시 불러도 한 번에 끝낸 것과 같다 (결정 3-5).
func Apply(ctx context.Context, p Paths, opt Options) (Result, error)

// Classify 는 upper 의 항목 하나가 무엇인지 읽는다.
// whiteout 은 문자 장치 0/0, opaque 는 user.overlay.opaque 값 y. xattr 형식의 whiteout 은 거절한다.
// (merge-rules FD 답 8 · 2026-09-26 고침. 겉면은 FD 의 domain-entities.md 1절이 바꿨다 — Classify 는 안쪽 순수 함수가 됐다)
func Classify(path string, fi fs.FileInfo) (Kind, error)
```

---

## 3. `internal/scratch`

### 3.1 trash 와 배경 삭제자

```go
type Trash struct {
	Dir string // <scratch>/trash
}

// Move 는 rename 한 번이다. 크기와 무관하다 (5.4).
func (t Trash) Move(path string) (string, error)

// Remove 는 helper 안에서 도는 삭제다. symlink 를 안 따라가고 trash 밖을 안 지운다 (5.3).
func Remove(trashDir, entry string) error

type Deleter struct {
	Trash  Trash
	Launch func(ctx context.Context, entry string) error // trash-helper 를 연다 (enode 가 채운다)
	Log    *slog.Logger
}

func (d *Deleter) Run(ctx context.Context) // 시작 때 한 번 돌고 Kick 마다 깬다
func (d *Deleter) Kick()
func (d *Deleter) Usage() Usage

type Usage struct {
	TrashBytes   int64     `yaml:"trash_bytes" json:"trash_bytes"`
	TrashEntries int       `yaml:"trash_entries" json:"trash_entries"`
	SpoolBytes   int64     `yaml:"spool_bytes" json:"spool_bytes"`
	Checkpoints  int       `yaml:"checkpoints" json:"checkpoints"`
	Deleting     bool      `yaml:"deleting" json:"deleting"`
	MeasuredAt   time.Time `yaml:"measured_at" json:"measured_at"` // 늦은 값이다 (Q3)
}
```

### 3.2 Checkpoint Store

```go
type Mode string // "off" | "on-failure" | "always"

type Policy struct {
	Mode      Mode
	TTL       time.Duration // 기본 48시간
	Share     float64       // (보존 총량 + 여유) 의 몫.  기본 0.2
	MaxBytes  int64         // checkpoint 하나의 상한 — 값은 NFR N2
	MaxInodes int64         // 값은 NFR N2
}

// Capture 는 receipt 의 checkpoint_capture 다 (ADR-076 §2 · 설계가 답한 ⑧).
type Capture struct {
	State     string     `json:"state"` // not_requested | unsupported | rejected | captured | failed
	Reason    string     `json:"reason,omitempty"`
	ID        string     `json:"id,omitempty"`
	Scope     string     `json:"scope,omitempty"`     // workspace-upper
	Guarantee string     `json:"guarantee,omitempty"` // inspect-only
	Node      string     `json:"node,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type Store struct {
	Spool  string
	Policy Policy
	Trash  Trash
}

// Admit 는 보고 전의 받아들임이다. 상수 시간만 본다 (결정 2-7).
func (s *Store) Admit(free, minFree uint64) (ok bool, reason string)

// Keep 은 upper 를 spool 로 rename 하고 미완료 기록을 남긴다.
func (s *Store) Keep(upper string, meta Meta) (Capture, error)

func (s *Store) Settle(ctx context.Context) error    // 보고 뒤 — 크기 판정 · 퇴출 · TTL
func (s *Store) Reconcile(ctx context.Context) error // 데몬 시작 때 — 미완료와 만료
func (s *Store) List() ([]Entry, error)              // enode checkpoint 가 읽는다
```

---

## 4. `internal/enode`

### 4.1 세션의 새 수명

```go
// Finalize 는 Harvest 를 대신한다 (ADR-075 §9).
type FinalizeSpec struct {
	Out       string
	Effect    contract.Effect
	Collect   map[string]string
	Check     []string       // ADR-037 — 지목한 경로만 stat
	Stamp     Stamp
	Discover  *DiscoverSpec  // 명시로 켠 bounded discovery (FR-1)
	Changeset bool           // effect 가 edit 일 때만
	Produce   *ProduceSpec   // producer adapter (⑪)
}

type FinalizeResult struct {
	Changed     []string
	Collected   []string
	Diagnostics Diagnostics
	Changeset   *ChangesetDescriptor
	Discovery   *DiscoveryResult // 상한에 닿았으면 Partial
}

// Diagnostics 는 result 의 진단 칸이다 (Q5).
type Diagnostics struct {
	Missing []string      `json:"missing,omitempty"`
	Collect []HarvestNote `json:"collect,omitempty"`
	Changes string        `json:"changes"` // measured | not_measured | partial
	Effect  string        `json:"effect"`
}

// Keep 은 닫을 때 upper 의 행선지다.
type Keep struct {
	Upper string // "" 면 trash · 경로면 그 자리로 rename (대기 자리 또는 spool)
}

type RuntimeCapability struct {
	Writes  string        // "isolated" | "in-place" — 광고 workspace.writes
	Capture CaptureSupport // 지원 · 범위 · 보장 · 안 되는 이유
}

type StepSession interface {
	Paths() RuntimePaths
	Project(context.Context, FrameworkProjectionSpec) (FrameworkProjection, error)
	Run(context.Context, ProcessSpec) (int, error)
	Finalize(context.Context, FinalizeSpec) (FinalizeResult, error)
	Close(context.Context, Keep) error // 나머지 runRoot 는 trash 로.  native 는 Keep.Upper 가 늘 비어 있다
	Environment() *execenv.Record
}

// Capability 는 세션이 아니라 런타임이 낸다 — 광고는 세션 밖에서 돈다 (lower-state FD · 2026-09-26 고침).
//   StepRuntime 에  Capability() RuntimeCapability
// (lower-state 코드 — func WorkspaceWrites(rt StepRuntime) string.  nil 이나 값 없는 런타임은 in-place)
```

### 4.2 Worker 와 Client

```go
// Exited 는 명령 종료 보고다 (FR-2 · mediator-api.md:450).
type Exited struct {
	Node     string    `json:"node"`
	Instance string    `json:"instance"`
	Attempt  int       `json:"attempt"`
	Outcome  Outcome   `json:"outcome"`
	ExitedAt time.Time `json:"exited_at"`
}

type Outcome struct {
	Kind string `json:"kind"` // exit | signal | timeout
	Code *int   `json:"code,omitempty"`
}

func (c *Client) Exited(ctx context.Context, runID string, seq int, e Exited) error

// Client.Upload 는 요청마다의 제한이 없는 client 다. 마감은 업로드 예산의 context 가 준다 (1.9).
// PutBlob 과 UploadLog 가 이것을 쓴다.

// Result 에 더하는 칸 — 정본 result wire (mediator-api.md:471 ~ :476) 와 receipt.
//   ExitedAt    *time.Time
//   FinalizedAt *time.Time
//   Finalize    string                // ok | timeout | error
//   Upload      string                // ok | timeout | error
//   Reason      string                // finalize_timeout | upload_timeout | merge_wait_timeout | bake_in_progress | lower_changed (lower-state FD 답 1) |
//                                     ir_mismatch (bake FD 물음 1 답 B · 2026-09-27 더함 — DONE 에 실린다)
//   Diagnostics *Diagnostics
//   Checkpoint  *scratch.Capture       // receipt 의 checkpoint_capture
//   Changeset   *ChangesetDescriptor
//   Build       *BuildManifest         // build 단계
//   Merge       *MergeResult           // merge 단계 — ir · resumed · 셈
```

```go
// (bake 코드 · CG 계획 4절 2번 — 겉면은 runAgentStep 과 같은 모양이다.  runCtx 는 임대의 context · ctx 는 데몬의 context ·
//  dir · in · out 은 그 단계의 워크스페이스 · $IN · $OUT.  execute 가 빈 argv 확인 앞에서 kind 로 나눠 부른다.  두 함수의 첫 줄이
//  WorkspaceWrites(w.Runtime) 가 isolated 가 아니면 거절한다 — Bake 가 있어도.  Runtime 이 nil 이면 native 로 보고 거절한다)
func (w *Worker) runBuildStep(runCtx, ctx context.Context, step *Step, dir, in, out string, log *slog.Logger)
func (w *Worker) runMergeStep(runCtx, ctx context.Context, step *Step, log *slog.Logger)

// (bake 코드) Worker 에 Bake *Baker 칸 — 기동이 func StartBaker(ctx context.Context, guard *LowerGuard, scratchDir string,
//  ident Identity, instance string, log *slog.Logger) *Baker 로 짓고 (guard 가 nil 이면 nil) 끝에 Baker.Wait() 로 배경 재개를 기다린다

type BuildManifest struct {
	Sync   lower.BuildRecord   `json:"sync"`
	Builds []lower.BuildRecord `json:"builds"`
	Head   string              `json:"head"`
	IR     *string             `json:"ir"`
	Pinned *lower.Pinned       `json:"pinned"`
	// HeadTags 는 sync 뒤 HEAD 에 붙은 태그 전부다 — 대조 전이거나 대조를 못 했으면 null · 태그가 없으면 [] (bake FD 물음 1 답 B ·
	// 되물음 5 답 A · 2026-09-27 더함)
	HeadTags []string          `json:"head_tags"`
}

type ChangesetDescriptor struct {
	Base     string `json:"base"`   // 정확한 base — git HEAD
	Digest   string `json:"digest"`
	Size     int64  `json:"size"`
	Complete bool   `json:"complete"` // 10 MiB 를 넘으면 false 이고 patch 를 안 올린다
}
```

### 4.3 광고와 후보 잠금 (Q1 · Q3)

```go
// DrainSource 는 drain 의 출처 하나다 (완료 조건 1).
type DrainSource struct {
	Kind   string `yaml:"kind" json:"kind"`     // owner | disk | bake
	Mode   string `yaml:"mode" json:"mode"`     // graceful | at-boundary
	Detail string `yaml:"detail" json:"detail"` // free 7 GB < min 10 GB · lower pending (run R-8790)
	Owner  bool   `yaml:"owner" json:"owner"`   // 소유자가 풀 수 있나
}

// LowerGuard 는 후보 잠금을 쥐고 놓는다. Advertiser 와 Worker 가 나눠 쓴다.
type LowerGuard struct{ /* lower.Dir · lower.Shared · 받아 적힌 drain 의 연속 횟수 */ }

// BeforeAdvert 는 광고 직전이다. 소유자 정책과 스스로의 사유를 합쳐 실을 drain 을 낸다.
// 후보면 공유 잠금을 쥐고, 못 쥐면(합치기 중) drain 을 싣는다.
// (lower-state FD 고침 — trash 유닛이 소유자 · 여유 부족 출처를 Advertiser 에 두어, 이미 만든 출처를 받고 자기 출처와 광고 키를 낸다)
func (g *LowerGuard) BeforeAdvert(others []DrainSource) (own []DrainSource, keys map[string]string)

// AfterResponse 는 광고 응답 뒤다. drain 이 받아 적혔고 임대가 없는 응답을 두 번 연속 받고 도는 단계가 없으면 놓는다
// (lower-state FD 답 1).  놓은 뒤에 보인 임대 · 합치기 없이 끝난 굽기도 여기서 본다.
func (g *LowerGuard) AfterResponse(drain string, leases []Lease)

// OnClaim 은 claim 직후다. prepare 단계면 놓는다 (결정 3-9).
func (g *LowerGuard) OnClaim(step *Step) error // 거절할 Run 이면 lower_changed 로 보고하게 오류 (lower-state FD 답 1)
func (g *LowerGuard) StepDone(step *Step)       // 세션을 닫은 뒤.  lower-state FD 가 더했다 · HoldBake · DropBake 도
// (bake FD 더함 · 되물음 1 답 A) func (g *LowerGuard) OnStale(f func(lower.State)) — state 가 committed 가 아닌 광고에서 부른다 ·
//  func (g *LowerGuard) Dir() *lower.Dir — 연 상태 자리 (못 열었으면 nil)
// (bake 코드 · CG 계획 4절 18번) guard 는 BeforeAdvert 안에서 ReadState 가 된 뒤 phase 가 building · pending · merging 이고 이
//  프로세스가 굽기를 쥐지 않았으면 (HoldBake 부터 DropBake 까지 밖) g.mu 아래에서 f(st) 를 곧바로 부른다 — 고루틴을 띄우지 않는다.
//  f 는 막지 않아야 한다 (Baker.onStale 은 판단만 하고 배경 일을 wg 로 연다).  f 가 nil 이거나 ReadState 가 실패하면 안 부른다.
//  둘 다 nil 수신자에 안전하다

// (lower-state 코드 — 짓는 것은 func StartLowerGuard(lowerRoot string, ident Identity, log *slog.Logger) *LowerGuard.
//  lowerRoot 가 "" 면 nil 이고 nil 은 아무것도 안 한다.  거절의 오류는 *LowerChangedError.  Advertiser 에 Guard · Writes,
//  Worker 에 Guard 칸)

// Status 에 더하는 칸 (상태 파일 · 제어판이 읽는다)
//   Drain   DrainStatus   // Effective · Sources
//   Scratch scratch.Usage
```

### 4.4 helper 입구와 env check

```go
func RunTrashHelper(args []string, errOut io.Writer) int // trash 항목 하나 — scratch.Remove
func RunMergeHelper(in io.Reader, out, errOut io.Writer) int // merge.Preflight 와 merge.Apply
// (bake 코드 · CG 계획 4절 16번) 입력 한 줄 · 출력 한 줄의 JSON — 요청은 op (preflight | apply) 와 upper · lower · trash,
//  응답은 result (apply 가 센 것) · error · check (시작 전 확인의 어긋남) · kind (preflight | op | io).  exit 0 은 오류가 없었다 ·
//  1 은 그 밖이다.
//  linux 밖은 exit 1 과 merge-helper is supported on linux only.  호스트는 unshare --user --map-root-user --map-auto --fork
//  --kill-child -- <enode> merge-helper 로 연다 (--mount 없음)

// ExecutionRuntimeVerifier 가 environment.FactSource 를 구현한다.
func (ExecutionRuntimeVerifier) Facts(ctx context.Context, doc execenv.Document, b execenv.Binding) []execenv.Fact
// (lower-state 코드 — ExecutionRuntimeVerifier{Notice io.Writer} : smoke 가 합치기를 기다린다는 줄의 자리.  enodectl env 는
//  표준 오류, 데몬 기동은 func LogNotice(log *slog.Logger) io.Writer — 줄마다 노드 로그의 Info 한 줄)
```

---

## 5. `internal/contract`

```go
type Effect string

const (
	EffectRead    Effect = "read"
	EffectEdit    Effect = "edit"
	EffectBuild   Effect = "build"
	EffectPrepare Effect = "prepare"
)

// Step 에 더하는 칸
//   Effect   Effect    `json:"effect,omitempty"`
//   Budget   *Budget   `json:"budget,omitempty"`
//   Sync     string    `json:"sync,omitempty"`
//   Builds   []Build   `json:"builds,omitempty"`
//   IR       string    `json:"ir,omitempty"`       // 구울 IR 태그의 값.  prepare 단계에 필수 (Units Generation 2.3 · 칸 이름은 Functional Design)
//   Merge    *Merge    `json:"merge,omitempty"`
//   Discover *Discover `json:"discover,omitempty"`
//   Produce  *Produce  `json:"produce,omitempty"`

type Budget struct {
	Finalize string `json:"finalize,omitempty"` // Go duration.  1분 아래는 400
	Upload   string `json:"upload,omitempty"`   // Go duration.  0 이하는 400
}

type Build struct {
	Name    string `json:"name"`    // [a-z0-9-]+ · 계약 안에서 겹치지 않는다
	Command string `json:"command"`
}

type Merge struct {
	Wait string `json:"wait,omitempty"` // Go duration.  기본 4시간
}

type Produce struct {
	Adapter string `json:"adapter"` // 이름/판 — 광고 producer.<이름>=<판> 과 맞춘다
	Target  string `json:"target"`
}

// 기존 StepKind 상수 블록의 끝(KindAcquire 뒤)에 둘을 더한다.
//   KindBuild   sync 와 builds 가 있다.  effect 는 prepare 여야 한다
//   KindMerge   merge 가 있다.  명령이 아니라 노드의 내장 단계

func (s Step) EffectOrDefault() Effect // 안 적으면 run 은 build, agent 는 edit
func (s Step) Budgets() (finalize, upload time.Duration)
```

---

## 6. `internal/environment`

```go
// FactSource 는 verifier 가 더 낼 Fact 가 있을 때 구현한다.
// 이 패키지는 구현을 모른다 — RuntimeVerifier 와 같은 이음매다 (check.go:82).
type FactSource interface {
	Facts(context.Context, Document, Binding) []Fact
}
```

`CheckWithRuntime` 은 verifier 가 이것을 구현하면 그 Fact 를 더한다. 어느 단계에서 더하고
어느 State 로 적는지는 Functional Design 이다.

---

## 7. `internal/store` 와 `internal/api`

```go
// schema.sql 에 더하는 줄
//   ALTER TABLE steps ADD COLUMN IF NOT EXISTS phase text;
//   ALTER TABLE steps ADD COLUMN IF NOT EXISTS phase_since timestamptz;
//   ALTER TABLE steps ADD COLUMN IF NOT EXISTS exit jsonb;

type ExitedReport struct {
	Node     string
	Instance string
	Attempt  int
	Outcome  json.RawMessage
	ExitedAt time.Time
}

// MarkExited 는 종료 보고의 수락이다. 판정이 아니다 (결정 1-7).
// 인스턴스가 다르면 ErrNotClaimant.  종결됐거나 같은 키면 조용히 성공.
func (s *Store) MarkExited(ctx context.Context, runID string, seq int, r ExitedReport) error

// StepView 에 더하는 칸
//   Phase      string          `json:"phase,omitempty"`
//   PhaseSince *time.Time      `json:"phase_since,omitempty"`
//   Exit       json.RawMessage `json:"exit,omitempty"`

// RequireView 에 더하는 칸 (QUEUED 일 때만 · 완료 조건 4)
//   Candidates *Candidates `json:"candidates,omitempty"`
type Candidates struct {
	Live     int `json:"live"`     // 요구를 만족하는 살아 있는 노드
	Busy     int `json:"busy"`     // 그중 다른 Run 에 묶인 것
	Draining int `json:"draining"` // 그중 drain 인 것
}

func (s *Store) CandidatesFor(ctx context.Context, reqs []contract.Require) ([]Candidates, error)
```

```go
// api.go — 등록 한 줄
//   mux.HandleFunc("POST /v1/runs/{run}/steps/{seq}/exited", s.auth(s.postExited))
func (s *Server) postExited(w http.ResponseWriter, r *http.Request)
```

---

## 8. `internal/record` 와 `internal/panel`

```go
// StepFile 에 더하는 칸 — 노드 시계다.  어느 칸이 어느 시계인지 적는 규칙은 Functional Design (7절 ②)
//   ExitedAt    string `json:"exited_at,omitempty"`
//   FinalizedAt string `json:"finalized_at,omitempty"`
```

```go
// panel.State 에 더하는 칸 (Q3)
//   DrainSources []enode.DrainSource `json:"drain_sources"`
//   Scratch      *scratch.Usage      `json:"scratch,omitempty"`
// Drain 은 정책 파일이 아니라 상태 파일의 effective 를 보인다.  소유자 것은 따로 표시한다
```

`internal/panel` 이 `internal/scratch` 를 임포트하는 것은 경계 표의 금지와 부딪치지 않는다
(`component-dependency.md` 2절).

---

## 9. `cmd/enode`

```text
   enode trash-helper <trash> <entry>     unshare 뒤에서 도는 삭제
   enode merge-helper                     unshare 뒤에서 도는 합치기.  경로 셋을 stdin 으로 받는다
   enode checkpoint list | show <id>      노드의 보존본.  host 경로는 소유자 화면에만 (5.3)
```
