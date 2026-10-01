package scratch

// Checkpoint Store — 실패한 단계의 upper 를 노드 안에 잠시 남기는 자리다 (ADR-076 §5 · checkpoint 유닛).
//
// 이 파일은 어휘와 판정이다 — 무엇을 받아들이고, 무엇을 퇴출하고, 기동 때 무엇을 거두나. 파일을 만지는 일은
// checkpoint_unix.go 에 있다. 단계의 결과 · Mediator · lower 를 모른다 — receipt 로 옮겨 담는 일과 신원은
// internal/enode 가 한다 (boundary_test.go 의 봉인).

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// SpoolName 은 scratch 안의 보존 자리 이름이다. 설정으로 못 바꾼다 (business-rules.md 11절).
const SpoolName = "spool"

// SpoolIn 은 scratch 의 spool 이다. 자리의 규칙이 한 곳에 있게 여기서만 짓는다.
func SpoolIn(scratchDir string) string { return filepath.Join(scratchDir, SpoolName) }

// 기록 · 요약 · 잠금의 이름과 읽는 상한 (domain-entities.md 4절 · NFR C7).
const (
	recordName    = "checkpoint.json"
	summaryName   = "usage.json"
	spoolLockName = ".lock"
	upperName     = "upper"
	recordLimit   = 1 << 20
	gib           = 1 << 30
)

// Mode 는 노드 소유자의 보존 정책이다 (ADR-076 §4 · 결정 2-11). 계약은 못 바꾼다.
type Mode string

const (
	ModeOff       Mode = "off"
	ModeOnFailure Mode = "on-failure"
	ModeAlways    Mode = "always"
)

// Valid 는 아는 값인가다.
func (m Mode) Valid() bool { return m == ModeOff || m == ModeOnFailure || m == ModeAlways }

// Policy 는 Store 가 받는 값이다. enode 가 설정의 checkpoint 블록과 min_free_gb 로 채운다.
type Policy struct {
	Mode            Mode
	TTL             time.Duration
	CapacityPercent int
	MaxBytes        int64  // 보존본 하나의 상한 (max_gb x 2^30)
	MaxTotalInodes  int64  // 0 이면 CapacityPercent 의 몫을 inode 에도 건다 (business-rules.md 10절)
	MinFree         uint64 // min_free_gb x 2^30
}

// receipt 의 상태 다섯과 원인 여섯 (ADR-076 §2 · 결정 2-2 · 2-3). 상태는 contract 에도 있다 — 이 패키지는 contract 를
// 못 부르므로 같은 글자를 둔다. 두 벌이 같은지는 internal/enode 의 시험이 본다.
const (
	StateNotRequested = "not_requested"
	StateUnsupported  = "unsupported"
	StateRejected     = "rejected"
	StateCaptured     = "captured"
	StateFailed       = "failed"

	ReasonRuntime         = "runtime"
	ReasonCrossFilesystem = "cross_filesystem"
	ReasonQuota           = "quota"
	ReasonFreeSpace       = "free_space"
	ReasonLeaseBudget     = "lease_budget"
	ReasonIO              = "io"
)

// 보존본의 범위와 보장 · 형식 (ADR-076 §3 · §5).
const (
	ScopeWorkspaceUpper  = "workspace-upper"
	GuaranteeInspectOnly = "inspect-only"
	FormatOverlayUpper   = "overlay-upper"
)

// 기록의 상태 셋과 보고의 성패 넷 · 퇴출 사유 셋 (domain-entities.md 3절 · FD 답 7 · 10).
const (
	EntryReserved = "reserved"
	EntryKept     = "kept"
	EntryEvicted  = "evicted"

	ReportDelivered  = "delivered"
	ReportRejected   = "rejected"
	ReportLeaseEnded = "lease_ended"
	ReportUnknown    = "unknown"

	EvictLargerThanMaxGB = "larger than max_gb"
	EvictOverCapacity    = "over capacity_percent"
	EvictOverTotalInodes = "over max_total_inodes"
)

// Capture 는 보존 한 번의 결과다 — receipt 의 checkpoint_capture 와 diagnostics 의 문장이 된다. enode 가
// contract.CheckpointCapture 로 옮겨 담는다.
type Capture struct {
	State     string
	Reason    string
	ID        string
	Scope     string
	Guarantee string
	Node      string
	ExpiresAt *time.Time
	// Detail 은 diagnostics.checkpoint 의 문장이다. host 경로를 싣지 않는다.
	Detail string
}

// Entry 는 항목 폴더 안의 기록이다 (checkpoint.json · 0600). 조회의 --json 이 이 칸들을 그대로 낸다.
type Entry struct {
	Schema      int        `json:"schema"`
	ID          string     `json:"id"`
	State       string     `json:"state"`
	Node        string     `json:"node"`
	Run         string     `json:"run"`
	Seq         int        `json:"seq"`
	Step        string     `json:"step"`
	Attempt     int        `json:"attempt"`
	Runtime     string     `json:"runtime"`
	Format      string     `json:"format"`
	Scope       string     `json:"scope"`
	Guarantee   string     `json:"guarantee"`
	Lower       string     `json:"lower,omitempty"`
	Environment string     `json:"environment,omitempty"`
	Head        string     `json:"head,omitempty"`
	IR          string     `json:"ir,omitempty"`
	CapturedAt  *time.Time `json:"captured_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Bytes       *int64     `json:"bytes,omitempty"`
	Inodes      *int64     `json:"inodes,omitempty"`
	MeasuredAt  *time.Time `json:"measured_at,omitempty"`
	Report      string     `json:"report,omitempty"`
	Evicted     string     `json:"evicted,omitempty"`
	EvictedAt   *time.Time `json:"evicted_at,omitempty"`
}

// recordSchema 는 기록의 판이다.
const recordSchema = 1

// Summary 는 spool 의 요약이다 (usage.json · NFR Design D3). 판정이 끝날 때 쓰고 받아들임이 읽는다 — 어느 노드의
// 판정이든 spool 전체를 읽고 쓰므로 scratch 를 나눠 쓰는 형제의 보존본이 든다.
type Summary struct {
	Bytes  int64     `json:"bytes"`
	Inodes int64     `json:"inodes"`
	At     time.Time `json:"at"`
}

// SpoolUsage 는 상태 파일의 spool 칸이다 (domain-entities.md 7절).
type SpoolUsage struct {
	Bytes       int64
	Checkpoints int
	Unsized     int
	At          time.Time
}

// NewID 는 불투명 ID 다 — crypto/rand 의 6 바이트를 hex 로 쓴 12 글자 (FD 답 8). 한 노드의 spool 안에서만
// 유일하면 된다 — 겹침은 만들 때 Mkdir 이 본다.
func NewID() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ValidID 는 ID 모양인가다 — 소문자 hex 12 글자. 조정은 이 모양이 아닌 폴더를 건드리지 않는다.
func ValidID(name string) bool {
	if len(name) != 12 {
		return false
	}
	for _, r := range name {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

// Admission 은 보고 전 받아들임이 보는 두 값이다 (결정 2-7 · business-rules.md 5절). 둘 다 upper 크기와 무관하다.
type Admission struct {
	Free      uint64 // statfs 의 여유
	FreeKnown bool   // 여유를 못 쟀으면 둘 다 거르지 않는다 — 광고의 여유 부족 drain 과 같은 규칙
	Kept      int64  // 요약의 보존 총량
}

// Admit 은 규칙 2절의 ④ 다. 여유 하한을 먼저 본다 — 노드 전체의 여유를 보존 몫보다 먼저 지킨다. 받아들이면
// ok 이고, 거절하면 그 결과다.
func (p Policy) Admit(a Admission) (Capture, bool) {
	if !a.FreeKnown {
		return Capture{}, true
	}
	if a.Free < p.MinFree {
		return Capture{State: StateRejected, Reason: ReasonFreeSpace,
			Detail: fmt.Sprintf("free space %.1f GiB is below min_free_gb %d", float64(a.Free)/gib, p.MinFree/gib)}, false
	}
	if a.Kept > 0 && float64(a.Kept) >= share(p.CapacityPercent, a.Kept, int64(a.Free)) {
		return Capture{State: StateRejected, Reason: ReasonQuota,
			Detail: fmt.Sprintf("kept checkpoints already use %.1f GiB, capacity_percent %d of kept plus free space",
				float64(a.Kept)/gib, p.CapacityPercent)}, false
	}
	return Capture{}, true
}

// share 는 몫이다 — percent x (보존 + 여유).
func share(percent int, kept, free int64) float64 {
	return float64(percent) / 100 * float64(kept+free)
}

// Filesystem 은 보고 뒤 판정이 보는 scratch 의 filesystem 이다 (statfs).
type Filesystem struct {
	// Known 이 거짓이면 statfs 를 못 했다 — 두 몫을 안 본다. 여유 0 으로 읽으면 보존본을 모두 퇴출한다.
	Known      bool
	Free       uint64
	FilesTotal uint64 // 0 이면 inode 수를 내지 않는 filesystem 이다 (btrfs) — inode 한도를 안 본다
	FilesFree  uint64
}

// Eviction 은 퇴출 하나다.
type Eviction struct {
	ID     string
	Reason string
}

// SettlePlan 은 보고 뒤 판정이 할 일이다 — 만료는 항목 폴더째, 퇴출은 upper 만 trash 로 간다.
type SettlePlan struct {
	Expire []string
	Evict  []Eviction
}

// PlanSettle 은 규칙 6절의 1 · 3 ~ 6 이다. entries 는 읽을 수 있는 기록 전부다 (reserved 는 안 본다).
//
// 몫은 한 번 정한다 — 퇴출한 것은 여유로 돌아올 것이므로 판정 동안 그대로다. 오래된 것부터 퇴출하고 새 것도
// 예외가 아니다. 측정 전 항목은 합에 0 으로 들고 퇴출하지 않는다 — 빼도 합이 줄지 않는다.
func (p Policy) PlanSettle(entries []Entry, fs Filesystem, now time.Time) SettlePlan {
	var plan SettlePlan
	var kept []Entry
	for _, e := range entries {
		switch {
		case e.State != EntryKept && e.State != EntryEvicted:
		case e.ExpiresAt != nil && !now.Before(*e.ExpiresAt):
			plan.Expire = append(plan.Expire, e.ID)
		case e.State == EntryKept:
			kept = append(kept, e)
		}
	}
	sort.SliceStable(kept, func(i, j int) bool { return capturedAt(kept[i]).Before(capturedAt(kept[j])) })

	evicted := map[string]bool{}
	evict := func(e Entry, reason string) {
		evicted[e.ID] = true
		plan.Evict = append(plan.Evict, Eviction{ID: e.ID, Reason: reason})
	}
	var bytes, inodes int64
	for _, e := range kept {
		if e.Bytes == nil {
			continue
		}
		if p.MaxBytes > 0 && *e.Bytes > p.MaxBytes {
			evict(e, EvictLargerThanMaxGB)
			continue
		}
		bytes += *e.Bytes
		if e.Inodes != nil {
			inodes += *e.Inodes
		}
	}

	if !fs.Known {
		return plan
	}
	limit := share(p.CapacityPercent, bytes, int64(fs.Free))
	for _, e := range kept {
		if float64(bytes) <= limit {
			break
		}
		if evicted[e.ID] || e.Bytes == nil {
			continue
		}
		evict(e, EvictOverCapacity)
		bytes -= *e.Bytes
		if e.Inodes != nil {
			inodes -= *e.Inodes
		}
	}

	if fs.FilesTotal == 0 {
		return plan
	}
	inodeLimit := float64(p.MaxTotalInodes)
	if p.MaxTotalInodes == 0 {
		inodeLimit = share(p.CapacityPercent, inodes, int64(fs.FilesFree))
	}
	for _, e := range kept {
		if float64(inodes) <= inodeLimit {
			break
		}
		if evicted[e.ID] || e.Inodes == nil {
			continue
		}
		evict(e, EvictOverTotalInodes)
		inodes -= *e.Inodes
	}
	return plan
}

func capturedAt(e Entry) time.Time {
	if e.CapturedAt == nil {
		return time.Time{}
	}
	return *e.CapturedAt
}

// Found 는 조정이 spool 에서 본 것 하나다 — 파일을 보는 일은 checkpoint_unix.go 가 한다.
type Found struct {
	Name     string
	Dir      bool
	LockFile bool   // 항목 잠금 파일이 있다
	LockFree bool   // 그 잠금을 쥘 수 있었다 — 주인이 없다
	Record   *Entry // 없거나 · 비었거나 · 읽을 수 없으면 nil
	Age      time.Duration
}

// Reconciliation 은 조정이 항목 하나에 할 일이다.
type Reconciliation int

const (
	Leave Reconciliation = iota
	TrashIt
	MarkUnknown
	WarnUnknown
)

// orphanEntryAge 는 잠금 파일조차 없는 항목 폴더를 주인 없는 것으로 치는 나이다 — 예약의 mkdir 와 잠금 사이를
// 비킨다 (trash 의 기동 청소와 같은 값 · business-rules.md 9절).
const orphanEntryAge = time.Hour

// PlanReconcile 은 규칙 9절의 표다.
func PlanReconcile(f Found, now time.Time) Reconciliation {
	if !f.Dir {
		return Leave // .lock · usage.json · 사람이 둔 파일
	}
	if !ValidID(f.Name) {
		return WarnUnknown
	}
	switch {
	case !f.LockFile && f.Age < orphanEntryAge:
		return Leave
	case f.LockFile && !f.LockFree:
		return Leave // 도는 단계의 예약이거나 형제가 쓰는 중이다
	}
	e := f.Record
	switch {
	case e == nil || e.State == EntryReserved:
		return TrashIt
	case e.ExpiresAt != nil && !now.Before(*e.ExpiresAt):
		return TrashIt
	case e.State == EntryKept && e.Report == "":
		return MarkUnknown
	}
	return Leave
}

// Store 는 spool 하나다 (<scratch>/spool). scratch 를 나눠 쓰는 형제 노드가 같은 spool 을 함께 쓴다 — 판정과 조정은
// spool 잠금으로 한 번에 하나이고, 도는 예약은 항목 잠금이 지킨다 (logical-components.md 3절).
type Store struct {
	Dir     string // <scratch>/spool
	Scratch string // upper 가 있는 자리 — filesystem 번호를 대 본다
	Trash   Trash
	Policy  Policy
	// Measure 는 항목 하나를 helper 안에서 걷는다 (enode 가 채운다 · subordinate uid 소유 항목이 있다). root 는
	// 항목 폴더이고 entry 는 "upper" 다. helper 를 못 띄우면 *LaunchError 다.
	Measure func(ctx context.Context, root, entry string) (Size, error)
	// Stat 은 scratch 의 statfs 다. nil 이면 statfs 를 부른다 — 시험이 바꿔 끼운다.
	Stat func() (Filesystem, error)
	// Now 는 시계다. nil 이면 time.Now 의 UTC.
	Now func() time.Time

	refused string
}

// Refused 는 기동 때 spool 자리를 거절한 까닭이다 — symlink · not a directory · owned by uid N. 비면 받아들였다.
func (s *Store) Refused() string { return s.refused }

func (s *Store) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now().UTC()
}

// SpoolCheck 는 기동 때 spool 자리를 본 결과다 (business-rules.md 11절 · NFR C8).
type SpoolCheck struct {
	Created  bool
	Narrowed bool
	From     os.FileMode // 좁히기 전의 권한
	Refused  string
}

// ErrRefused 는 거절한 spool 에 예약하려 한 것이다.
var ErrRefused = errors.New("the spool is not a private directory of this node")

// Reservation 은 세션을 열 때 잡은 자리다 (NFR Design 답 1). 항목 잠금을 쥐고 있다 — 확정이나 버림이 놓는다.
type Reservation struct {
	ID  string
	Dir string

	mu    sync.Mutex
	lock  *os.File
	entry Entry
	done  bool
}

// Upper 는 Close 가 upper 를 옮길 자리다 — 없어야 한다 (RENAME_NOREPLACE).
func (r *Reservation) Upper() string { return filepath.Join(r.Dir, upperName) }

// Entry 는 예약할 때 적은 기록이다.
func (r *Reservation) Entry() Entry {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.entry
}

// Listed 는 조회의 한 줄이다 — 기록에 자리를 더한다. 도는 단계의 예약은 목록에 없다.
type Listed struct {
	Entry
	Path       string    `json:"path"`
	Incomplete bool      `json:"-"` // 주인 없는 reserved
	Since      time.Time `json:"-"` // 줄 세우는 시각 — 확정한 시각이나 폴더를 만든 시각
}

// SettleResult 는 보고 뒤 판정 한 번의 결과다. 노드 로그와 상태 파일이 받는다.
type SettleResult struct {
	Expired []string
	Evicted []Eviction
	Usage   SpoolUsage
	Moved   bool             // trash 에 넣었다 — 삭제자를 깨운다
	Failed  map[string]error // 측정하지 못한 항목 — 다음 판정에서 다시 걷는다
	Launch  error            // helper 를 못 띄웠다 — 판정의 측정을 멈췄다
}

// ReconcileResult 는 기동 조정의 결과다.
type ReconcileResult struct {
	Trashed []string
	Marked  []string // report 칸을 unknown 으로 채웠다
	Unknown []string // ID 모양이 아닌 폴더 — 건드리지 않았다
}
