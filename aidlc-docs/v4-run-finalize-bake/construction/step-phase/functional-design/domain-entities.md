# `step-phase` — 도메인 엔티티

Mediator 가 단계의 진행 구간을 적고 보이는 데 쓰는 칸과 타입이다. 입력은 계획의 답 일곱과 되물음 넷
(`construction/plans/step-phase-functional-design-plan.md` 2절 · `...-clarification-questions.md`)이다.
규칙은 `business-rules.md`, 흐름은 `business-logic-model.md` 에 있다.

```text
   답      1 A  수락 표 · 거절은 409          2 A  본문 다섯 칸 · 형식만 검사
           3 A  칸마다 시계 하나 · 보정 없음   4 A  칸을 안 지운다 · Record 에 exit 와 last_phase
           5 C  result 의 새 칸을 모두 타입으로  6 A  후보 수는 배타 · Satisfies · free 없음
           7 A  Claimed 에 계약 칸을 그대로
   되물음   1 A  타입은 internal/contract       2 A  changeset 은 뺀다
           3 A  표의 모양 그대로 · 더하기만 한다  4 A  값으로 400 하지 않는다
```

---

## 1. `steps` 표의 새 칸 셋

`ALTER TABLE steps ADD COLUMN IF NOT EXISTS …` 로 붙인다. CHECK 를 걸지 않는다 — `state` 와 같은
이유다 (`schema.sql:77` — 어휘가 늘 때 마이그레이션을 강요하지 않는다). 옛 행은 셋 다 NULL 이다.

| 칸 | 타입 | 뜻 | 적는 사건 |
|---|---|---|---|
| `phase` | `text` | CLAIMED 안의 구간 — `running` · `finalizing` · `waiting` | claim · 재전달 · exited 수락 |
| `phase_since` | `timestamptz` | 그 구간이 시작된 시각 | 같은 사건 |
| `exit` | `jsonb` | exited 가 나른 `outcome` 그대로 — `{kind, code}` | exited 수락 |

- phase 는 **열린 어휘**다 (ADR-075 §10.3). 뒤에 값을 더해도 스키마와 판정이 안 바뀐다
- `exit` 는 판정이 아니다 (정본 `mediator-api.md:210`). Verify 는 이 칸을 안 읽는다
- NULL 은 「적은 적이 없다」다 — 이 코드 전에 claim 된 단계 · 아직 안 집힌 단계 · 되돌려진 단계

phase 의 값은 `internal/store` 의 상수다.

```go
const (
	PhaseRunning    = "running"    // 명령이 돈다.  claim 때
	PhaseFinalizing = "finalizing" // 명령은 끝났고 결과를 확정하는 중이다.  exited 수락 때
	PhaseWaiting    = "waiting"    // 다른 것을 기다린다.  merge 단계의 claim 때 (결정 1-12)
)
```

---

## 2. 결과 어휘 — `internal/contract/result.go` (새 파일 · 되물음 1 = A)

노드와 Mediator 가 함께 가져오는 패키지에 둔다. 광고 어휘(`advert.go` 의 `Advert` · `Policy`)가 이미
거기 있다. 새 패키지 셋(lower · merge · scratch)은 표준 라이브러리만 쓰는 규칙을 지키려고 자기 타입을
그대로 두고, `internal/enode` 가 보고할 때 이 타입으로 옮겨 담는다.

**이 파일의 모양은 더하기만 한다** (되물음 3 = A). 뒤 유닛이 칸을 더할 수 있다. 있는 칸의 이름이나
뜻을 바꾸려면 이 문서를 함께 고친다 — 봉인된 Record 를 읽는 쪽이 옛 이름을 안다.

### 2.1 종료 보고

```go
// Exited 는 명령 종료 보고의 본문이다 (ADR-075 결정 7 · POST .../exited).
// 노드가 보내고 Mediator 가 받는다.  한 정의를 둘이 쓴다.
type Exited struct {
	Node     string    `json:"node"`
	Instance string    `json:"instance"` // 노드의 「생」 (ADR-030)
	Attempt  int       `json:"attempt"`
	Outcome  Outcome   `json:"outcome"`
	ExitedAt time.Time `json:"exited_at"` // 노드 시계
}

// Outcome 은 명령이 어떻게 끝났나다.
type Outcome struct {
	Kind string `json:"kind"`           // exit | signal | timeout
	Code *int   `json:"code,omitempty"` // exit 면 반드시 · signal 이면 신호 번호 · timeout 이면 없어도 된다
}

const (
	OutcomeExit    = "exit"
	OutcomeSignal  = "signal"
	OutcomeTimeout = "timeout"
)

// Check 는 본문의 형식만 본다 (400 의 근거).  값의 뜻(미래 시각 등)은 안 본다.
func (e Exited) Check() error
```

### 2.2 result 에 더하는 칸의 타입

```go
// Stage 는 결과를 확정하고 올리는 두 구간의 끝이다 (FR-3).
type Stage string

const (
	StageOK      Stage = "ok"
	StageTimeout Stage = "timeout"
	StageError   Stage = "error"
)

// 원인 코드 (ADR-075 §10.2 · ADR-077 §6 · §7).  열린 어휘다.
const (
	ReasonFinalizeTimeout  = "finalize_timeout"
	ReasonUploadTimeout    = "upload_timeout"
	ReasonMergeWaitTimeout = "merge_wait_timeout"
	ReasonBakeInProgress   = "bake_in_progress"
)

// Diagnostics 는 result 의 진단 칸이다 (Application Design Q5 · FR-1).
// 판정 재료가 아니다 — 사람이 읽는다.
type Diagnostics struct {
	Missing        []string      `json:"missing,omitempty"`         // out 에 적었는데 안 나온 이름
	Collect        []CollectNote `json:"collect,omitempty"`         // collect 가 못 걷은 이유
	Changes        string        `json:"changes"`                   // measured | not_measured | partial
	Effect         Effect        `json:"effect"`                    // 이 단계가 따른 effect
	Discovered     []string      `json:"discovered,omitempty"`      // discover 가 찾은 경로.  produced 가 아니다
	DiscoveryLimit string        `json:"discovery_limit,omitempty"` // visits | time | memory | size.  닿았을 때만
}

// CollectNote 는 오늘의 enode.HarvestNote 와 같은 뜻이다 (이름 · 이유).
type CollectNote struct {
	Name string `json:"name"`
	Why  string `json:"why"`
}

const (
	ChangesMeasured    = "measured"
	ChangesNotMeasured = "not_measured" // 「바뀐 파일이 없다」가 아니다 (FR-1)
	ChangesPartial     = "partial"      // 상한에 닿았다

	LimitVisits = "visits"
	LimitTime   = "time"
	LimitMemory = "memory"
	LimitSize   = "size"
)

// CheckpointCapture 는 receipt 의 checkpoint_capture 다 (ADR-076 §2).
// 상태는 닫힌 집합이고 원인은 열린 집합이다.
type CheckpointCapture struct {
	State     string     `json:"state"`               // not_requested | unsupported | rejected | captured | failed
	Reason    string     `json:"reason,omitempty"`
	ID        string     `json:"id,omitempty"`        // captured 일 때만
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
	Head   string        `json:"head"`   // sync 뒤 manifest HEAD
	IR     *string       `json:"ir"`     // HEAD 에 정확히 붙은 태그.  없으면 null
	Pinned *Pinned       `json:"pinned"` // manifest 가 없는 저장소면 null
}

// BuildRecord 는 명령 하나의 기록이다.  sync 도 같은 모양이다.
type BuildRecord struct {
	Name       string    `json:"name"`
	Command    string    `json:"command"`
	StartedAt  time.Time `json:"started_at"`  // 노드 시계
	FinishedAt time.Time `json:"finished_at"` // 노드 시계
	ExitCode   int       `json:"exit_code"`
}

type Pinned struct {
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
}

// MergeResult 는 굽기 merge 단계가 아는 것이다 (ADR-077 §5 · §7 · §12).
type MergeResult struct {
	IR         *string   `json:"ir"`          // 합친 뒤 lower 의 ir.  없으면 null
	PreviousIR *string   `json:"previous_ir"` // 합치기 전 lower 의 ir.  없으면 null
	MergedAt   time.Time `json:"merged_at"`   // 노드 시계
	Resumed    bool      `json:"resumed"`     // 앞서 끊긴 합치기를 이어 끝냈나
	Ops        MergeOps  `json:"ops"`
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
```

`contract.Build` (U1 · 계약의 구성 하나 `{name, command}`)와 `contract.BuildRecord` (결과 · 명령 하나를
돈 기록)는 다른 것이다. 이름이 비슷하지만 앞은 요청이고 뒤는 사실이다.

**changeset 칸은 없다** (되물음 2 = A). 채울 결과 adapter 가 순연됐다 (Units Generation Q3 = A).

---

## 3. `store.StepResult` 에 더하는 칸 (답 5 = C)

오늘 모양 그대로 평평하게 더한다 (정본 `mediator-api.md:471` 의 result 가 평평하다). 모두 `omitempty`.

| JSON 칸 | Go 타입 | 뜻 |
|---|---|---|
| `exited_at` | `*time.Time` | 명령이 끝난 시각 (노드 시계). exited 가 유실돼도 같은 사실 |
| `finalized_at` | `*time.Time` | 결과 확정이 끝난 시각 (노드 시계) |
| `finalize` | `contract.Stage` | 결과 확정 구간의 끝 — ok · timeout · error |
| `upload` | `contract.Stage` | 올리기 구간의 끝 — ok · timeout · error |
| `reason` | `string` | 원인 코드 |
| `diagnostics` | `*contract.Diagnostics` | 진단 |
| `checkpoint_capture` | `*contract.CheckpointCapture` | 보존 상태 |
| `build` | `*contract.BuildManifest` | 굽기 build 단계만 |
| `merge` | `*contract.MergeResult` | 굽기 merge 단계만 |

봉인이 `steps.result` 를 `StepResult` 로 다시 풀므로 (`seal.go:139`) 여기 있어야 Record 에 남는다.

---

## 4. 진행 조회 — `StepView` · `RequireView` · `runView`

### 4.1 `StepView` 에 더하는 칸

| JSON 칸 | Go 타입 | 싣는 때 |
|---|---|---|
| `phase` | `string` | 단계가 CLAIMED 이고 값이 있을 때 (답 4 = A) |
| `phase_since` | `*time.Time` | 같은 때 |
| `exit` | `*contract.Outcome` | 값이 있으면 상태와 무관하게 |

### 4.2 대기 사유 (답 6 = A)

```go
// Candidates 는 QUEUED 인 Run 의 요구 줄 하나에 대한 후보 수다 (완료 조건 4).
// 배타로 센다: live = busy + draining + (나머지).  나머지는 칸으로 내지 않는다 (ADR-065 의 free).
type Candidates struct {
	Live     int `json:"live"`     // 그 요구를 만족하는 살아 있는 광고
	Busy     int `json:"busy"`     // live 중 임대를 쥔 것 — 다른 Run 이 맡았다
	Draining int `json:"draining"` // live 중 임대가 없는데 drain 이 걸린 것
}
```

- `RequireView` 에 `Candidates *Candidates json:"candidates,omitempty"` 를 더한다
- `runView` 에 `CandidatesAt *time.Time json:"candidates_at,omitempty"` 를 더한다 — 셈을 한 DB 시각이다
- 둘 다 QUEUED 인 Run 의 `GET /v1/runs/{id}` 에서만 나온다

---

## 5. `Claimed` 에 더하는 칸 (답 7 = A · contract-grammar 가 넘긴 일)

계약의 칸을 contract 의 타입 그대로 옮긴다. 기본값은 채우지 않는다.

| JSON 칸 | Go 타입 |
|---|---|
| `effect` | `contract.Effect` |
| `budget` | `*contract.Budget` |
| `discover` | `bool` |
| `sync` | `string` |
| `builds` | `[]contract.Build` |
| `ir` | `string` |
| `merge` | `*contract.Merge` |

노드는 기본값을 `contract.Step` 의 메서드로 채운다 — `EffectOrDefault` · `Budgets` · `MergeWait`.
Mediator 와 노드가 같은 상수를 읽는다.

---

## 6. Record 의 단계 기록 — `record.StepFile` 에 더하는 칸 (답 3 · 4 = A)

| JSON 칸 | 뜻 | 시계 | 어디서 오나 |
|---|---|---|---|
| `exited_at` | 명령이 끝난 시각 | 노드 | result 의 `exited_at`. 없으면 exited 로 받은 `phase_since` |
| `finalized_at` | 결과 확정이 끝난 시각 | 노드 | result 의 `finalized_at` |
| `exit` | exited 가 나른 outcome | - | `steps.exit` |
| `last_phase` | 단계가 끝날 때의 phase | - | `steps.phase` |

오늘의 `started_at` · `ended_at` 은 그대로 **Mediator 시계**다. 한 기록 안에서 칸마다 시계가 정해져
있고, 그 표가 `business-rules.md` 3절이다.

---

## 7. 이 유닛이 안 만드는 것

```text
   노드가 exited 를 보내는 일 · result 의 새 칸을 채우는 일      finalize · checkpoint · bake
   새 패키지의 타입에서 contract 타입으로 옮겨 담는 일           그 칸을 채우는 유닛
   changeset 칸                                              순연 (되물음 2 = A)
   result 보고의 인스턴스 대조                                잔여 (FR-2 · 5.3)
   Mediator 화면(internal/api/ui)에 phase 를 그리는 일           유닛 정의에 없다.  진행 조회 JSON 으로 충분하다
```
