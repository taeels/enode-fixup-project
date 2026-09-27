// Package lower 는 같은 기계의 형제 노드가 한 아래층(lower)을 함께 쓰기 위한 상태와 잠금이다
// (ADR-077 §5 · §6 · requirements.md FR-8).
//
// 담는 것 — lower 루트의 신원(Key · Root) · 상태 자리 <lowers>/<key>/ 와 그 안의 파일 (lower.json ·
// state.json · 잠금 셋 · 쥔 사람 기록) · <lower>/.enode-metadata.json 읽기와 쓰기 · 다른 프로세스의
// 마운트 훑기 · 준비도 점검 셋의 판정.
//
// 모르는 것 — 준비도 점검의 State 어휘(internal/environment) · 계약의 규칙(internal/contract) · 합치기
// (internal/merge) · trash(internal/scratch) · 광고와 drain · Run 의 흐름 · home 디렉터리. 상태 자리의
// 뿌리(lowers)는 부르는 쪽이 준다 — 제품은 $HOME/.local/state/enode/lowers, 시험은 임시 폴더다.
//
// 판정(judge.go)은 모든 플랫폼에서 빌드되고, 시스템 호출(statfs · statx · flock · /proc)은 _linux.go 에만
// 있다. linux 밖은 같은 겉면이 ErrUnsupported 를 돌려준다.
//
// 표준 라이브러리와 golang.org/x/sys 만 쓴다 (경계 시험의 봉인). Mediator 는 이 패키지를 링크하지 않는다.
package lower

import (
	"errors"
	"os"
	"time"
)

// ErrUnsupported 는 linux 밖에서 돌려주는 오류다. 굽기와 runc-overlay 는 linux 만 한다.
var ErrUnsupported = errors.New("lower is supported on linux only")

// 상태 자리 안의 이름들 (business-rules.md 1절).
const (
	identityFile = "lower.json"
	stateFile    = "state.json"
	lowerLock    = "lower.lock"
	bakeLock     = "bake.lock"
	holdersDir   = "holders"

	// metadataFile 은 lower 뿌리의 metadata 이름이다 (결정 3-16).
	metadataFile = ".enode-metadata.json"

	// schema 는 이 패키지가 쓰는 세 파일(lower.json · state.json · metadata)의 판이다.
	schema = 1
	// maxJSON 은 읽는 JSON 파일 하나의 상한이다. 넘으면 못 읽은 것으로 친다.
	maxJSON = 1 << 20
)

// Key 는 lower 루트 디렉터리의 신원이다. statfs 의 f_fsid 와 inode (결정 3-14 · bind 별칭도 같은 키).
type Key struct {
	// FSID 는 f_fsid.Val[0]<<32 | Val[1] 이다. %x 로 찍으면 `stat -f -c %i` 와 같은 글자가 된다 (둘 다 앞의 0 을 안 찍는다).
	// Val 은 int32 둘이라 uint32 로 바꾼 뒤 붙인다 (32비트 arm 도 같다).
	FSID uint64
	Ino  uint64
}

// Root 는 lower 루트를 한 번 읽은 값이다. statfs 한 번과 statx 한 번이다.
type Root struct {
	Path    string // symlink 를 푼 경로
	Key     Key
	BirthNs int64  // statx 의 btime.  0 이면 이 filesystem 이 알려 주지 않았다 — 신원 대조를 건너뛴다
	Dev     uint64 // st_dev (major:minor 를 linux 의 방식으로 붙인 값)
	MountID uint64 // statx 의 STATX_MNT_ID.  커널이 안 주면 /proc/self/mountinfo 에서 찾은 값.  0 이면 모른다
	UID     int    // 소유 uid
}

// Dir 은 <lowers>/<key>/ 다. 노드 데몬만 만든다 (Open). 점검과 smoke 는 만들지 않고 본다 (Peek).
type Dir struct {
	Root Root
	Path string

	// Notes 는 Open 이 자리를 고친 것이다 — 좁힌 권한 · 새로 쓴 신원 · 못 지운 last_attempt. 영어 한 줄씩이고
	// 부르는 쪽이 로그에 쓴다.
	Notes []string
	// Loose 는 Peek 이 본, 남에게 열린 비트가 있는 자리다. Peek 은 고치지 않는다 (ADR-073).
	Loose []string

	uid int // 노드 uid — 자리와 파일의 주인이어야 한다
}

// Identity 는 lower.json 이다.
type Identity struct {
	Schema   int       `json:"schema"`
	FSID     string    `json:"fsid"` // Key.FSID 의 16진 (앞의 0 없이) — stat -f -c %i 와 같은 글자
	Ino      uint64    `json:"ino"`
	BirthNs  int64     `json:"birth_ns"` // 0 이면 모름
	Paths    []string  `json:"paths"`    // 이 lower 를 본 경로들 (별칭).  정보다 — 대조하지 않는다
	OwnerUID int       `json:"owner_uid"`
	SeenAt   time.Time `json:"seen_at"` // 마지막으로 쓴 때 (노드 시계)
}

// Verdict 는 lower.json 과 지금 읽은 뿌리를 댄 결과다 (business-rules.md 2절).
type Verdict int

const (
	VerdictNew     Verdict = iota + 1 // lower.json 이 없다 — 처음 본 lower
	VerdictMatch                      // 같다.  경로가 새면 Paths 에 더한다
	VerdictReused                     // btime 이 다르고 상태가 committed — 같은 inode 번호의 다른 디렉터리.  새로 쓴다
	VerdictForeign                    // btime 이 다르고 상태가 committed 가 아니다 — 끊긴 굽기가 다른 디렉터리의 것
	VerdictBroken                     // 못 읽는다 · fsid 나 ino 가 자리 이름과 다르다
)

// Phase 는 state.json 의 상태 넷이다 (business-rules.md 3절).
type Phase string

const (
	PhaseCommitted Phase = "committed"
	PhaseBuilding  Phase = "building"
	PhasePending   Phase = "pending"
	PhaseMerging   Phase = "merging"
)

// State 는 state.json 이다. lower 밖에 둔다 (결정 3-15). 쓰는 쪽은 굽기 잠금의 주인 하나다 (Bake.WriteState).
type State struct {
	Schema       int          `json:"schema"`
	Phase        Phase        `json:"phase"`
	Owner        *Owner       `json:"owner,omitempty"`         // committed 가 아닐 때
	PendingUpper string       `json:"pending_upper,omitempty"` // pending · merging 일 때 대기 upper 의 자리
	Since        time.Time    `json:"since"`                   // 이 phase 가 된 때 (노드 시계)
	LastAttempt  *LastAttempt `json:"last_attempt,omitempty"`  // 실패한 굽기 (결정 3-18)
}

// Owner 는 committed 가 아닌 상태를 만든 굽기다.
type Owner struct {
	Run      string `json:"run"`
	Step     int    `json:"step"`
	Node     string `json:"node"`
	Instance string `json:"instance"`
}

// LastAttempt 는 실패한 굽기의 기록이다.
type LastAttempt struct {
	Run    string        `json:"run"`
	At     time.Time     `json:"at"`
	Reason string        `json:"reason"`
	Builds []BuildRecord `json:"builds,omitempty"`
}

// BuildRecord 는 명령 하나를 돈 기록이다. JSON 칸은 contract.BuildRecord 와 같다 — 이 패키지는 계약을
// 임포트하지 않으므로 따로 둔다 (봉인). 같은 모양인지는 internal/enode 의 시험이 본다.
type BuildRecord struct {
	Name       string    `json:"name"`
	Command    string    `json:"command"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	ExitCode   int       `json:"exit_code"`
}

// Role 은 쥔 사람 기록의 역할이다.
type Role string

const (
	RoleCandidate Role = "candidate" // drain 없이 광고하는 중 — 매칭 후보
	RoleRun       Role = "run"       // prepare 가 아닌 Run 의 임대를 쥐었다
)

// Holder 는 holders/<node_id>.json 이다 (답 3). holders/<node_id>.lock 을 배타로 쥔 동안만 살아 있는 기록이다.
type Holder struct {
	Node  string    `json:"node"`          // node_id
	Label string    `json:"label"`         // 사람이 읽는 이름 — 대기 로그에 쓴다
	Role  Role      `json:"role"`          // candidate | run
	Run   string    `json:"run,omitempty"` // RoleRun 일 때
	Since time.Time `json:"since"`         // 이 역할이 된 때 (노드 시계)
	Acks  int       `json:"acks"`          // drain 이 받아 적히고 임대가 0 인 응답을 연속 몇 번 받았나 — 답 1
	PID   int       `json:"pid"`
}

// Shared 는 lower.lock 의 공유 잠금과 쥔 사람 기록이다. 기록이 없는 공유(smoke)도 있다.
type Shared struct {
	d    *Dir
	lock *os.File // lower.lock
	rec  *os.File // holders/<node>.lock — 기록을 안 쓰면 nil
	h    Holder
}

// Recorded 는 쥔 사람 기록을 남겼는가다. node_id 가 파일 이름이 될 수 없거나 기록 잠금을 못 쥐면
// 기록 없이 공유만 쥔다 (business-rules.md 7절).
func (s *Shared) Recorded() bool { return s != nil && s.rec != nil }

// Exclusive 는 lower.lock 의 배타 잠금이다. 잡히면 이 lower 를 마운트한 자리가 0 이다 (business-rules.md 8.1).
type Exclusive struct {
	lock *os.File
}

// Bake 는 bake.lock 의 배타 잠금이다. 주인이 살아 있다는 증거다 (결정 3-12). state.json 은 이것을 쥔 쪽만 쓴다.
type Bake struct {
	d    *Dir
	lock *os.File
}

// Waiting 은 배타를 기다리는 동안 watch 가 받는 것이다.
type Waiting struct {
	Holders []Holder // 살아 있는 기록
	Unnamed bool     // 잠금이 막혀 있는데 살아 있는 기록이 0 이다 — 기록 없는 쥔 쪽 (smoke · 옛 판의 프로세스)
}

// Metadata 는 <lower>/.enode-metadata.json 이다 (결정 3-16 · component-methods.md 1.4).
// 합치기의 마지막 동작으로만 쓴다 (bake 유닛). 이 유닛은 읽어 광고 키를 짓는다.
type Metadata struct {
	Schema          int           `json:"schema"`
	Source          Source        `json:"source"`
	Builds          []BuildRecord `json:"builds"`
	Environment     string        `json:"environment"`
	WorkspaceTarget string        `json:"workspace_target"`
	Bake            BakeRecord    `json:"bake"`
}

// Source 는 합친 lower 의 출처다.
type Source struct {
	URL         string    `json:"url"`
	Branch      string    `json:"branch"`
	RepoID      string    `json:"repo_id"`
	Head        string    `json:"head"`
	IR          *string   `json:"ir"` // 없으면 null
	Pinned      *Pinned   `json:"pinned"`
	SyncCommand string    `json:"sync_command"`
	SyncedAt    time.Time `json:"synced_at"`
}

// Pinned 는 고정한 manifest 파일과 그 sha256 이다. JSON 칸은 contract.Pinned 와 같다.
type Pinned struct {
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
}

// BakeRecord 는 이 lower 를 마지막으로 합친 굽기다.
type BakeRecord struct {
	Run        string    `json:"run"`
	Node       string    `json:"node"`
	MergedAt   time.Time `json:"merged_at"`
	Resumed    bool      `json:"resumed"`
	PreviousIR *string   `json:"previous_ir"`
}

// Marker 는 lower 의 내용이 마지막 합치기 뒤로 그대로인지 보는 표지다 (답 1). metadata 가 없으면 제로값이다.
type Marker struct {
	Run      string
	MergedAt time.Time
}

// Marker 는 이 metadata 의 표지다. nil 수신자는 제로값이다.
func (m *Metadata) Marker() Marker {
	if m == nil {
		return Marker{}
	}
	return Marker{Run: m.Bake.Run, MergedAt: m.Bake.MergedAt}
}

// Mount 는 이 lower 를 아래층으로 쓰는 overlay 마운트 하나다 — 다른 프로세스의 마운트 namespace 에서 찾았다.
type Mount struct {
	PID        int    // 그 namespace 를 처음 읽은 프로세스
	Namespace  string // /proc/<pid>/ns/mnt 의 링크 값 — mnt:[4026532...]
	MountPoint string // overlay 의 마운트 자리 (그 namespace 안의 경로)
	LowerDir   string // lowerdir 중 이 lower 에 닿는 것
}

// MountScan 은 ForeignMounts 가 훑은 결과다.
type MountScan struct {
	Found      []Mount
	Namespaces int // 읽은 마운트 namespace 수
	Unreadable int // 권한 때문에 못 읽은 같은 uid 프로세스 (dumpable 이 꺼진 것) — 그물의 빈틈
}

// Cause 는 점검이 어긋난 까닭의 갈래다. internal/enode 가 environment.State 로 옮긴다 (답 8).
type Cause string

const (
	CauseBinding Cause = "binding" // 설정이 쓸 수 없는 자리를 가리킨다 -> invalid
	CauseState   Cause = "state"   // profile 밖에 남은 상태가 막는다 -> external-blocked
)

// Finding 은 environment.Fact 로 옮겨질 결과다. 이 패키지는 environment 를 모른다.
type Finding struct {
	Name        string // "binding.scratch_filesystem" | "lower.owner_uid" | "lower.identity"
	Required    string
	Observed    string
	OK          bool
	Cause       Cause // OK 가 거짓일 때
	Remediation string
}

// 점검 셋의 이름 (component-methods.md 1.5).
const (
	findingScratch  = "binding.scratch_filesystem"
	findingOwnerUID = "lower.owner_uid"
	findingIdentity = "lower.identity"
)

// pathError 는 자리 안의 파일이나 디렉터리 하나를 못 쓴 까닭이다. 문구는 "lower: <path>: <why>" 다.
// 점검은 Path 와 Err 를 나눠 「cannot read <path>: <why>」로 적는다.
type pathError struct {
	Path string
	Err  error
}

func (e *pathError) Error() string { return "lower: " + e.Path + ": " + e.Err.Error() }
func (e *pathError) Unwrap() error { return e.Err }
