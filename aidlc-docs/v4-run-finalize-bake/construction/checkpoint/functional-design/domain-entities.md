# `checkpoint` — 도메인 엔티티

실패한 단계의 upper 를 노드 안에 잠시 남기는 자리의 칸과 타입이다. 입력은 계획
(`construction/plans/checkpoint-functional-design-plan.md`) 의 3절과 답 열이다. 규칙은 `business-rules.md`, 흐름은
`business-logic-model.md` 에 있다. 작성 2026-09-30T13:46:54Z.

**답 N** 은 계획 물음 N 의 답이다. 열 모두 A 다.

| 답 | 한 줄 |
|---|---|
| 1 | on-failure 의 실패 — 노드가 아는 실패 전부 (exit ≠ 0 · signal · 하네스 미완주 · 빠진 산출물 · Finalize 오류) |
| 2 | 정책을 먼저 본다 — 요구하지 않았으면 `not_requested`, 요구했을 때만 `unsupported` |
| 3 | 굽기 build 는 build 끝의 실패 넷만 보존한다. 대기 자리 뒤와 merge 단계는 보존하지 않는다 |
| 4 | 설정 키는 단위를 이름에 — `policy` · `ttl_hours` · `capacity_percent` · `max_gb` · `max_total_inodes` |
| 5 | 상태 파일과 제어판에 양과 실효 정책 · 기동 로그 한 줄 |
| 6 | 새 명령 없이 `show` 가 여는 법과 버리는 법을 알린다 |
| 7 | 퇴출된 것은 만료 시각까지 기록이 남고, 만료되면 기록째 없앤다 |
| 8 | ID 는 무작위 12 hex |
| 9 | 실패 상세는 `diagnostics.checkpoint` 문장 하나 |
| 10 | 보고가 닿지 않은 보존본도 TTL 까지 남기고, store 가 보고의 성패를 적는다 |

**새 패키지는 없다.** Store 는 `internal/scratch` 의 새 파일이다 (봉인 — 표준 라이브러리와 `golang.org/x/sys` 만 ·
`internal/panel/boundary_test.go:102`). `contract` · `lower` 를 모르므로 신원과 receipt 의 옮겨 담기는 `internal/enode` 가 한다.

---

## 1. 정책과 설정 (답 4 · 계획 3절 8번)

```go
// internal/enode/config.go — Local 에 더한다. 블록이 없으면 nil 이고 기본값을 쓴다.
type CheckpointConfig struct {
	Policy          string `yaml:"policy,omitempty"`           // off | on-failure | always.  비면 on-failure
	TTLHours        int    `yaml:"ttl_hours,omitempty"`        // 0 이면 48
	CapacityPercent int    `yaml:"capacity_percent,omitempty"` // 0 이면 20.  1 ~ 100
	MaxGB           int    `yaml:"max_gb,omitempty"`           // 보존본 하나의 상한.  0 이면 N2 의 기본값
	MaxTotalInodes  int64  `yaml:"max_total_inodes,omitempty"` // 보존본 전체의 inode 한도.  0 이면 N2 의 기본값
}
```

```go
// internal/scratch — Store 가 받는 값. enode 가 CheckpointConfig 와 min_free_gb 로 채운다.
type Mode string

const (
	ModeOff       Mode = "off"
	ModeOnFailure Mode = "on-failure"
	ModeAlways    Mode = "always"
)

type Policy struct {
	Mode            Mode
	TTL             time.Duration
	CapacityPercent int
	MaxBytes        int64  // max_gb x 2^30
	MaxTotalInodes  int64
	MinFree         uint64 // min_free_gb x 2^30 (advertise.go:336 과 같은 단위)
}
```

N2 (NFR 값 둘째 — 하나의 상한과 inode 한도의 값) 는 NFR Requirements 가 정한다. 여기는 이름과 「0 이면 기본값」 만이다.

---

## 2. receipt 와 diagnostics (`internal/contract/result.go`)

**receipt** 는 오늘의 `CheckpointCapture` 그대로다 (`result.go:147-166` · 칸 일곱). `internal/enode` 의 `Result` 에 칸 하나를
더한다 — ``CheckpointCapture *contract.CheckpointCapture `json:"checkpoint_capture,omitempty"` `` (이름과 모양은
`store.StepResult` 와 같다 · `store/claim.go:933`).

| 칸 | 값 | 있는 때 |
|---|---|---|
| `state` | `not_requested` · `unsupported` · `rejected` · `captured` · `failed` | 늘 |
| `reason` | `runtime` · `cross_filesystem` · `quota` · `free_space` · `lease_budget` · `io` | `unsupported` · `rejected` · `failed` |
| `id` | 12 hex (답 8) | `captured` |
| `scope` · `guarantee` | `workspace-upper` · `inspect-only` | `captured` |
| `node` | 노드 ID (`Ident.NodeID`) | `captured` |
| `expires_at` | 잡은 시각 + TTL | `captured` |

**diagnostics** 에 칸 하나를 더한다 (답 9 · 더하기만 · 행렬 밖).

```go
// Checkpoint 는 보존하지 못한 까닭이다 — 영어 문장 하나. captured · not_requested 면 없다.
// host 경로를 싣지 않는다 (requirements.md 5.3).
Checkpoint string `json:"checkpoint,omitempty"`
```

---

## 3. store 기록과 항목의 생애

기록은 항목 폴더 안의 `checkpoint.json` 이다 (0600). 조회의 `--json` 이 이 칸들에 `path` 하나를 더해 그대로 낸다.

```go
type Entry struct {
	Schema      int        `json:"schema"` // 1
	ID          string     `json:"id"`
	State       string     `json:"state"` // reserved | kept | evicted
	Node        string     `json:"node"`
	Run         string     `json:"run"`
	Seq         int        `json:"seq"`
	Step        string     `json:"step"`
	Attempt     int        `json:"attempt"`
	Runtime     string     `json:"runtime"`   // runc-overlay
	Format      string     `json:"format"`    // overlay-upper
	Scope       string     `json:"scope"`     // workspace-upper
	Guarantee   string     `json:"guarantee"` // inspect-only
	Lower       string     `json:"lower,omitempty"`       // lower 신원 (lower-state 의 Identity)
	Environment string     `json:"environment,omitempty"` // 준비된 실행 환경의 식별자 (execenv.Record)
	Head        string     `json:"head,omitempty"`        // lower metadata 가 있으면
	IR          string     `json:"ir,omitempty"`
	CapturedAt  *time.Time `json:"captured_at,omitempty"` // kept 부터
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Bytes       *int64     `json:"bytes,omitempty"`  // 측정 전이면 없다
	Inodes      *int64     `json:"inodes,omitempty"` // 측정한 항목 수 (디렉터리 포함 · scratch.Size.Entries)
	MeasuredAt  *time.Time `json:"measured_at,omitempty"`
	Report      string     `json:"report,omitempty"`  // delivered | rejected | lease_ended | unknown (답 10)
	Evicted     string     `json:"evicted,omitempty"` // 퇴출 사유 (답 7)
	EvictedAt   *time.Time `json:"evicted_at,omitempty"`
}
```

| 상태 | 뜻 | 트리 | 들어오는 때 | 나가는 때 |
|---|---|---|---|---|
| `reserved` | 예약 — 옮기는 중이거나 확정 전에 멈췄다 | 없거나 `upper/` | 보고 전 판정이 받아들임 | 확정 → `kept` · 버림이나 조정 → 자리째 trash |
| `kept` | 보관 중 | `upper/` | 확정 | 퇴출 → `evicted` · 만료 → 자리째 trash |
| `evicted` | 트리는 trash 로 갔고 기록만 남았다 | 없음 | 보고 뒤 판정 | 만료 → 자리째 trash |

만료된 항목은 기록째 없다 (답 7). `reserved` 는 조회에 `incomplete` 로 보인다 (규칙 13절).

**보고의 성패** (답 10) — `delivered` 보고가 닿았다 · `rejected` Mediator 가 4xx 로 거절했다 · `lease_ended` 임대가 끝나 보내기를
멈췄다 · `unknown` 데몬이 멈췄거나 성패를 적기 전에 죽었다. `Worker.report` 의 네 끝이다 (`claim.go:405-433`).

---

## 4. spool 의 자리 (계획 3절 10번)

```text
   <scratch>/spool/                 0700  노드 uid.  고정 — 설정으로 못 바꾼다
     .lock                          보고 뒤 판정과 조정이 쥐는 spool 잠금 (flock)
     <ID>/                          0700  노드 uid.  이름이 곧 ID
       .enode-session.lock          reserved 동안 쥐는 항목 잠금 (scratch.HoldSession 과 같은 모양)
       checkpoint.json              기록 (0600)
       upper/                       runRoot/upper 가 rename 한 번으로 들어온다
```

ID 는 `crypto/rand` 의 6 바이트를 hex 로 쓴 12 글자다 (답 8). 폴더를 `Mkdir` 로 만들어 겹침을 확인한다 — 이미 있으면 새로 뽑는다.

---

## 5. 런타임과 세션 (`internal/enode/runtime.go` · `runc_overlay_linux.go`)

```go
// RuntimeCapability 에 더한다 (lower-state 가 넘긴 칸).
type CaptureSupport struct {
	Supported bool
	Scope     string // workspace-upper
	Guarantee string // inspect-only
	Reason    string // 못 하면 runtime
}

// Keep 에 두 칸을 더한다. 둘 다 spool 로 옮길 때만 쓴다 — 굽기의 대기 자리는 오늘 그대로다.
type Keep struct {
	Upper  string
	By     time.Time   // 비어 있지 않으면, helper 가 끝난 뒤 이 시각이 지났을 때 옮기지 않는다
	Result *KeepResult // 있으면 옮기기의 결과를 여기에 적고 Close 의 오류에 싣지 않는다
}

type KeepResult struct {
	Moved bool
	Late  bool  // By 가 지나 옮기지 않았다
	Err   error // 세션이 abort 됐거나 rename 이 실패했다
}
```

| 런타임 | `Capture` |
|---|---|
| `NativeRuntime` | `Supported: false · Reason: runtime` |
| `RuncOverlayRuntime` (linux) | `Supported: true · workspace-upper · inspect-only` |
| `RuncOverlayRuntime` (linux 밖 · `runc_overlay_other.go`) | linux 판과 같은 값 — 짓지 못하므로 쓰이지 않는다 (그 파일의 `Capability` 주석과 같은 까닭) |

---

## 6. Worker 쪽

| 칸 | 자리 | 뜻 |
|---|---|---|
| `Worker.Checkpoints *CheckpointKeeper` | `internal/enode` 새 파일 | Store 와 정책을 들고 판정과 옮겨 담기를 한다. nil 이면 칸을 안 싣는다 (시험과 옛 조립) |
| `closing.failed bool` · `closing.bake bool` | `claim.go:853` | 부르는 쪽이 아는 실패와 굽기 build 인지 (규칙 1절) |
| `Result.CheckpointCapture` | `claim.go:192` | 2절 |
| 보고의 성패 | `Worker.report` | 네 끝을 `Reported(ID, 성패)` 로 store 에 적는다 |

---

## 7. 상태 파일과 제어판 (답 5)

```go
// scratch.Usage 에 더한다 — 앞의 둘은 component-methods.md:278 의 칸이다.
SpoolBytes      int64     `yaml:"spool_bytes" json:"spool_bytes"`             // kept 항목의 측정한 합
Checkpoints     int       `yaml:"checkpoints" json:"checkpoints"`             // kept 항목 수
SpoolUnsized    int       `yaml:"spool_unsized" json:"spool_unsized"`         // 그 가운데 측정 전
SpoolMeasuredAt time.Time `yaml:"spool_measured_at" json:"spool_measured_at"` // 늦은 값이다 (Application Design Q3)

// enode.Status 와 panel.State 에 더한다. native 노드도 싣는다.
type CheckpointStatus struct {
	Policy          string `yaml:"policy" json:"policy"`
	TTLHours        int    `yaml:"ttl_hours" json:"ttl_hours"`
	CapacityPercent int    `yaml:"capacity_percent" json:"capacity_percent"`
	Unsupported     string `yaml:"unsupported,omitempty" json:"unsupported,omitempty"` // native 면 runtime
}
```

`Usage` 는 두 곳이 쓴다 — 삭제자는 trash 칸, Store 는 spool 칸. `StatusBook` 에 `SetSpool` 을 더하고, 각자 자기 칸만 바꾼다.

---

## 8. 어휘가 두 벌인 자리

상태 다섯은 `contract` 에 있고 (`result.go:161-165`), `internal/scratch` 는 `contract` 를 못 부른다. 그래서 `scratch` 가 같은 글자의
상태 다섯과 원인 여섯을 둔다. 원인은 `scratch` 에만 있다 — Mediator 는 원인을 안 본다 (`store/claim.go:965`). 두 벌이 같은
글자인지는 `internal/enode` 의 시험 한 줄이 확인한다.
