package contract

import (
	"errors"
	"fmt"
	"time"
)

// 결과 어휘 — 노드가 Mediator 에 보고하는 것의 모양이다 (ADR-075 · ADR-076 · ADR-077).
//
// 여기 두는 까닭은 노드와 Mediator 가 함께 가져오는 패키지이기 때문이다. 광고
// 어휘(Advert · Policy)가 이미 이 패키지에 있다. 노드 쪽 새 패키지 셋(lower ·
// merge · scratch)은 표준 라이브러리만 쓰는 규칙을 지키려고 자기 타입을 그대로
// 두고, internal/enode 가 보고할 때 이 타입으로 옮겨 담는다.
//
// 이 파일의 모양은 더하기만 한다. 뒤 유닛이 칸을 더할 수 있다. 있는 칸의 이름이나
// 뜻을 바꾸려면 step-phase 의 Functional Design (domain-entities.md 2절)을 함께
// 고친다 — 봉인된 Record 를 읽는 쪽이 옛 이름을 안다.

// ── 종료 보고 ────────────────────────────────────────────────────────────

// Exited 는 명령 종료 보고의 본문이다 (ADR-075 결정 7 · POST .../exited).
//
// 노드가 보내고 Mediator 가 받는다. 한 정의를 둘이 쓴다. 판정 재료가 아니다 —
// Mediator 는 이것으로 phase 를 finalizing 으로 옮길 뿐이고, 단계를 끝내는 것은
// 여전히 result 하나다 (결정 1-7 · 1-9).
type Exited struct {
	Node string `json:"node"`
	// Instance 는 노드의 「이번 생」이다 (ADR-030). 단계를 집은 생과 대조한다 —
	// 재시작한 노드가 앞 생의 종료를 보고하지 못한다.
	Instance string  `json:"instance"`
	Attempt  int     `json:"attempt"`
	Outcome  Outcome `json:"outcome"`
	// ExitedAt 은 노드 시계다. Mediator 는 이 값을 고치지 않고 phase_since 에 적는다.
	ExitedAt time.Time `json:"exited_at"`
}

// Outcome 은 명령이 어떻게 끝났나다.
type Outcome struct {
	Kind string `json:"kind"` // exit | signal | timeout
	// Code 는 exit 면 반드시 있다. signal 이면 신호 번호이고 없어도 된다.
	// timeout 이면 없어도 된다.
	Code *int `json:"code,omitempty"`
}

const (
	OutcomeExit    = "exit"
	OutcomeSignal  = "signal"
	OutcomeTimeout = "timeout"
)

// Check 는 본문의 형식만 본다. 틀리면 Mediator 가 400 으로 돌려준다.
//
// 값의 뜻(미래 시각 · started_at 보다 앞선 시각)은 안 본다. 두 기계의 시계를
// Mediator 가 대조하지 않는다 — 칸마다 시계를 하나로 정했을 뿐이다.
// JSON 과 시각 문자열의 파싱은 받는 쪽이 먼저 한다. 이것은 풀린 값만 본다.
func (e Exited) Check() error {
	switch {
	case e.Node == "":
		return errors.New("exited: node is empty")
	case e.Instance == "":
		return errors.New("exited: instance is empty; the report is matched against the instance that claimed the step")
	case e.Attempt < 0:
		return errors.New("exited: attempt must not be negative")
	case e.ExitedAt.IsZero():
		return errors.New("exited: exited_at is missing")
	}
	switch e.Outcome.Kind {
	case OutcomeExit:
		if e.Outcome.Code == nil {
			return errors.New("exited: outcome.kind exit needs a code")
		}
	case OutcomeSignal, OutcomeTimeout:
	default:
		return fmt.Errorf("exited: outcome.kind %q is not exit, signal or timeout", e.Outcome.Kind)
	}
	return nil
}

// ── result 에 더하는 칸 ───────────────────────────────────────────────────

// Stage 는 결과를 확정하는 구간과 올리는 구간이 어떻게 끝났나다 (FR-3).
type Stage string

const (
	StageOK      Stage = "ok"
	StageTimeout Stage = "timeout"
	StageError   Stage = "error"
)

// 원인 코드다 (ADR-075 §10.2 · ADR-077 §6 · §7). 열린 어휘다 — 뒤 유닛이 더한다.
const (
	ReasonFinalizeTimeout  = "finalize_timeout"
	ReasonUploadTimeout    = "upload_timeout"
	ReasonMergeWaitTimeout = "merge_wait_timeout"
	ReasonBakeInProgress   = "bake_in_progress"
)

// Diagnostics 는 result 의 진단 칸이다 (Application Design Q5 · FR-1).
//
// 판정 재료가 아니다 — 계약 작성자가 읽는다. success_when 은 이 칸을 안 본다.
type Diagnostics struct {
	// Missing 은 out 에 적었는데 안 나온 이름이다.
	Missing []string `json:"missing,omitempty"`
	// Collect 는 collect 가 못 걷은 것과 그 이유다.
	Collect []CollectNote `json:"collect,omitempty"`
	// Changes 는 바뀐 파일을 확인했는가다 — measured · not_measured · partial.
	// omitempty 가 아니다. not_measured 는 「바뀐 파일이 없다」와 다르고, 칸이
	// 빠지면 읽는 쪽이 둘을 구별하지 못한다.
	Changes string `json:"changes"`
	// Effect 는 이 단계가 따른 effect 다. 기본값을 채운 뒤의 값이다.
	Effect Effect `json:"effect"`
	// Discovered 는 discover 가 찾은 경로다. produced 가 아니다.
	Discovered []string `json:"discovered,omitempty"`
	// DiscoveryLimit 은 discover 가 닿은 상한이다 — visits · time · memory · size.
	// 닿았을 때만 있다.
	DiscoveryLimit string `json:"discovery_limit,omitempty"`
}

// CollectNote 는 collect 가 못 걷은 이름 하나와 그 이유다.
// 오늘의 enode.HarvestNote 와 같은 뜻이다.
type CollectNote struct {
	Name string `json:"name"`
	Why  string `json:"why"`
}

const (
	ChangesMeasured = "measured"
	// ChangesNotMeasured 는 「바뀐 파일이 없다」가 아니다 (FR-1). 확인하지 않았다는 뜻이다.
	ChangesNotMeasured = "not_measured"
	// ChangesPartial 은 상한에 닿아 일부만 확인했다는 뜻이다.
	ChangesPartial = "partial"

	LimitVisits = "visits"
	LimitTime   = "time"
	LimitMemory = "memory"
	LimitSize   = "size"
)

// CheckpointCapture 는 실패한 단계의 보존 상태다 (ADR-076 §2).
// 상태는 닫힌 집합이고 원인은 열린 집합이다.
type CheckpointCapture struct {
	State  string `json:"state"` // not_requested | unsupported | rejected | captured | failed
	Reason string `json:"reason,omitempty"`
	// 아래는 captured 일 때만 있다.
	ID        string     `json:"id,omitempty"`
	Scope     string     `json:"scope,omitempty"`     // workspace-upper
	Guarantee string     `json:"guarantee,omitempty"` // inspect-only
	Node      string     `json:"node,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

const (
	CaptureNotRequested = "not_requested"
	CaptureUnsupported  = "unsupported"
	CaptureRejected     = "rejected"
	CaptureCaptured     = "captured"
	CaptureFailed       = "failed"
)

// BuildManifest 는 굽기 build 단계가 아는 것이다 (ADR-077 §5 의 metadata 중 build 몫).
type BuildManifest struct {
	Sync   BuildRecord   `json:"sync"`
	Builds []BuildRecord `json:"builds"`
	// Head 는 sync 뒤의 manifest HEAD 다.
	Head string `json:"head"`
	// IR 은 HEAD 에 정확히 붙은 태그다. 없으면 null 이다.
	IR *string `json:"ir"`
	// Pinned 는 manifest 파일의 지문이다. manifest 가 없는 저장소면 null 이다.
	Pinned *Pinned `json:"pinned"`
}

// BuildRecord 는 명령 하나를 돈 기록이다. sync 도 같은 모양이다.
//
// 계약의 Build 와 다르다 — Build 는 요청({name, command})이고 이것은 사실이다.
type BuildRecord struct {
	Name       string    `json:"name"`
	Command    string    `json:"command"`
	StartedAt  time.Time `json:"started_at"`  // 노드 시계
	FinishedAt time.Time `json:"finished_at"` // 노드 시계
	ExitCode   int       `json:"exit_code"`
}

// Pinned 는 고정한 manifest 파일과 그 sha256 이다.
type Pinned struct {
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
}

// MergeResult 는 굽기 merge 단계가 아는 것이다 (ADR-077 §5 · §7 · §12).
type MergeResult struct {
	// IR 은 합친 뒤 lower 의 ir 이다. 없으면 null 이다.
	IR *string `json:"ir"`
	// PreviousIR 은 합치기 전 lower 의 ir 이다. 없으면 null 이다.
	PreviousIR *string   `json:"previous_ir"`
	MergedAt   time.Time `json:"merged_at"` // 노드 시계
	// Resumed 는 앞서 끊긴 합치기를 이어서 끝냈는가다.
	Resumed bool     `json:"resumed"`
	Ops     MergeOps `json:"ops"`
}

// MergeOps 는 합치기 연산의 셈이다 — ADR-077 §12 실측의 일곱 갈래.
type MergeOps struct {
	Replaced  int `json:"replaced"`  // 기존 파일을 대체
	Created   int `json:"created"`   // 새 파일
	Dirs      int `json:"dirs"`      // 새 디렉터리
	Opaque    int `json:"opaque"`    // opaque 디렉터리
	Whiteouts int `json:"whiteouts"` // 지운 항목
	Trashed   int `json:"trashed"`   // lower 에서 trash 로 옮긴 것
	Attrs     int `json:"attrs"`     // 속성을 맞춘 디렉터리
}
