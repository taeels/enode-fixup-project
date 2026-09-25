# `finalize` — 도메인 엔티티

노드가 명령이 끝난 뒤의 구간(결과 확정 · 닫기 · 업로드 · 보고)을 닫는 데 쓰는 칸과 타입이다. 입력은
계획의 답 아홉(`construction/plans/finalize-functional-design-plan.md` 2절)이다. 규칙은
`business-rules.md`, 흐름은 `business-logic-model.md` 에 있다.

```text
   답   1 A  Finalize 예산은 Finalize 까지.  닫기는 trash 전까지 예산 밖.  로그를 먼저 올린다
        2 A  runCtx 에 마감 · overlay 는 마감을 요청에 싣는다 · 넘어도 닫기와 업로드와 보고는 계속
        3 A  프로세스가 떴고 끝났으면 보낸다 · agent 는 runner.go 의 콜백 · 단계 단위로 멈춘다
        4 A  agent 가 완주 못 해도 discover 는 돈다
        5 A  상한 넷 · 시간은 Finalize 예산의 절반 · overlay 는 upper 만 훑는다
        6 A  workspace.changed 를 없앤다 · diagnostics 칸과 단계 로그 끝
        7 A  기본값을 쓴 run 단계와 discover 를 안 켠 agent 단계의 로그 끝에 안내 한 줄
        8 A  업로드 client · 흘려 보내기 · 크기 사전 검사 없음 · 재시도 없음
        9 A  조각 1 은 go test (방문 수) 와 스크립트 (300만 파일) 둘
```

**타입은 새로 짓지 않는 쪽을 먼저 본다.** 결과의 모양(`contract.Exited` · `Outcome` · `Stage` ·
`Diagnostics` · 원인 코드)과 계약의 칸(`contract.Effect` · `Budget` 과 그 메서드)은 앞 두 유닛이 이미
지었다. 이 유닛은 그것을 노드 쪽에서 채우고, 노드 안의 세션 겉면만 새로 짓는다.

---

## 1. 노드 `Step` 의 새 칸 셋 (`internal/enode/claim.go`)

claim 응답(`store.Claimed`)이 이미 싣는 칸이다 (step-phase). 노드의 `Step` 이 받게 한다. JSON 이름은
Claimed 와 같다.

| 칸 | 타입 | 없을 때 |
|---|---|---|
| `Effect` `json:"effect,omitempty"` | `contract.Effect` | 종류의 기본값 — run 은 build, agent 는 edit |
| `Budget` `json:"budget,omitempty"` | `*contract.Budget` | 두 예산 모두 기본값 — Finalize 1분 · 업로드 3분 |
| `Discover` `json:"discover,omitempty"` | `bool` | 안 훑는다 |

- **기본값은 계약 패키지의 메서드로 채운다** — `contract.Step.EffectOrDefault` 와 `Budgets`. 노드는
  종류를 정하는 칸(run 이면 `Run`, agent 면 `Agent`)과 이 세 칸만 채운 `contract.Step` 을 만들어 부른다.
  Mediator 와 노드가 같은 상수를 읽는다 (contract-grammar 가 권한 모양)
- 옛 Mediator 는 세 칸을 안 싣는다. 그때도 기본값이 채워지므로 동작은 이 유닛의 기본값 그대로다
- bake 유닛이 더할 칸(`sync` · `builds` · `ir` · `merge`)은 이 유닛이 받지 않는다

---

## 2. 세션 겉면 — `Harvest` 를 `Finalize` 로 (`internal/enode/runtime.go`)

ADR-075 §9 가 「contract effect 를 정한 뒤 한 번에 바꾸고, 그때 `HarvestSpec` 의 칸마다 호환을 판정한다」
고 적었다. effect 가 contract-grammar 에서 생겼으므로 이 유닛이 바꾼다.

### 2.1 `FinalizeSpec`

```go
// FinalizeSpec 은 Worker 가 세션에 넘기는 결과 확정의 입력이다.  새 Run 계약이 아니다.
type FinalizeSpec struct {
	Workspace string            // native 에서만.  overlay 는 helper 가 merged 로 채운다
	Out       string
	Effect    contract.Effect   // 기본값을 채운 뒤의 값.  진단 칸에 그대로 실린다
	Diff      bool              // workspace.diff 를 만드나 — business-rules.md 1절의 표
	Collect   map[string]string // collect 를 할 때만.  agent 가 완주 못 하면 비운다
	Check     []string          // 지목 경로 (ADR-037).  늘 있다
	Stamp     Stamp
	Discover  bool
	// Deadline 은 Finalize 예산의 마감이다 (답 2 = A).  overlay helper 에 요청과 함께 가서
	// helper 가 자기 ctx 를 만든다.  native 에서는 ctx 의 마감과 같다.
	Deadline time.Time
}
```

`HarvestSpec` 과의 대응 — 칸마다 판정했다.

```text
   HarvestSpec       FinalizeSpec     판정
   Workspace         Workspace        그대로
   Out               Out              그대로
   RecordDiff        Diff             이름만 바꾼다.  changeset 이 아니다 (Git changeset adapter 는 순연)
   Discover          Discover         뜻이 바뀐다 — 늘 켜던 전체 훑기가 아니라 계약이 켠 명시 훑기
   Collect           Collect          그대로
   Check             Check            그대로
   Stamp             Stamp            그대로
   (없음)             Effect           새로.  진단 칸의 effect
   (없음)             Deadline         새로.  overlay 에 마감을 나른다
```

`component-methods.md` 4.1 의 스케치에 있던 `Changeset` · `Produce` 는 넣지 않는다 — 둘 다 순연된
adapter 의 자리다 (FR-11).

### 2.2 `FinalizeResult`

```go
type FinalizeResult struct {
	Changed   []string      // 지목 경로 중 바뀐 것 (ADR-037).  오늘 그대로
	Collected []string
	Notes     []HarvestNote // collect 가 못 걷은 것.  진단 칸의 collect 로 옮겨 담는다
	DiffBytes int
	DiffError string
	Discovery *Discovery    // Discover 가 켜졌을 때만
}

// Discovery 는 명시 훑기의 결과다.  produced 가 아니다 (ADR-075 §8).
type Discovery struct {
	Paths   []string // 큰 파일부터.  상한 안쪽만
	Total   int      // 찾은 파일 수 (목록에서 자른 것 포함)
	Deleted int      // overlay upper 의 whiteout 수.  목록에 안 넣는다
	Visited int      // 방문한 항목 수 — 조각 1 의 시험이 센다
	Limit   string   // 닿은 상한 — contract.LimitVisits · LimitTime · LimitMemory · LimitSize.  안 닿았으면 ""
	Skipped string   // 못 훑은 이유 — 워크스페이스가 없는 노드.  훑었으면 ""
}
```

`HarvestResult` 의 `Workspace` · `WorkspaceN` · `ChangedError` 는 `Discovery` 가 대신한다.

### 2.3 `Keep` 과 `StepSession`

```go
// Keep 은 닫을 때 upper 의 행선지다.  이 유닛에서는 늘 비어 있다 — 쓰는 것은
// trash (trash 로) · bake (대기 자리) · checkpoint (spool) 유닛이다.
type Keep struct {
	Upper string // "" 면 버린다.  경로면 그 자리로 rename
}

type StepSession interface {
	Paths() RuntimePaths
	Project(context.Context, FrameworkProjectionSpec) (FrameworkProjection, error)
	Run(context.Context, ProcessSpec) (int, error)
	Finalize(context.Context, FinalizeSpec) (FinalizeResult, error)
	Close(context.Context, Keep) error
	Environment() *execenv.Record
}
```

- `Close` 의 동작은 이 유닛에서 오늘과 같다 — overlay 는 helper 를 닫고 runRoot 를 지운다. ctx 는
  받기만 한다 (닫기가 예산 밖이다 · 답 1). trash 유닛이 ctx 와 Keep 을 쓰기 시작한다
- `onceSession` 도 새 `Close` 를 한 번만 부른다. 두 번째부터는 첫 오류를 돌려준다 (오늘 그대로)
- `RuntimeCapability` (`component-methods.md` 4.1) 는 이 유닛이 짓지 않는다 — lower-state · checkpoint 의 것

---

## 3. 명시 훑기의 상한 (`internal/enode/changed.go` · 답 5 = A)

```go
const (
	discoverMaxVisits = 2_000_000 // 방문한 항목 (디렉터리 포함)
	discoverMaxHeld   = 200_000   // 들고 있는 항목 — 메모리 상한 대신.  오늘 changedSince 의 값
	discoverMaxPaths  = 2_000     // diagnostics.discovered 의 경로 수
	discoverMaxBytes  = 256 << 10 // diagnostics.discovered 의 경로 글자 합
)

// 시간 상한은 상수가 아니다 — Finalize 예산의 절반이다.  기본 30초.
func discoverTime(finalize time.Duration) time.Duration { return finalize / 2 }
```

값의 크기(느린 디스크에서 30초에 몇 항목을 도나)는 이 유닛의 NFR Requirements 가 확인한다.

---

## 4. 노드 `Result` 의 새 칸 여섯 (`internal/enode/claim.go`)

Mediator 의 `store.StepResult` 가 이미 받는 칸이다 (step-phase). 이름과 타입을 그대로 맞춘다.

| 칸 | 타입 | JSON | 뜻 |
|---|---|---|---|
| `ExitedAt` | `*time.Time` | `exited_at,omitempty` | 종료 status 를 받은 순간의 노드 시계. 종료 보고의 `exited_at` 과 같은 값 |
| `FinalizedAt` | `*time.Time` | `finalized_at,omitempty` | 닫기가 끝난 순간의 노드 시계 (답 1) |
| `Finalize` | `contract.Stage` | `finalize,omitempty` | ok · timeout · error |
| `Upload` | `contract.Stage` | `upload,omitempty` | ok · timeout · error |
| `Reason` | `string` | `reason,omitempty` | `finalize_timeout` · `upload_timeout` |
| `Diagnostics` | `*contract.Diagnostics` | `diagnostics,omitempty` | 진단 칸 (답 6) |

- **여섯은 Finalize 를 돈 단계에만 있다.** 명령 앞에서 실패한 단계(워크스페이스 준비 · `$IN` ·
  runtime open · 빈 argv)와 임대가 끝나 Finalize 를 건너뛴 단계에는 없다. 칸이 없다는 것이 곧
  「그 구간에 닿지 않았다」다
- `checkpoint_capture` · `build` · `merge` 는 이 유닛이 안 채운다 (checkpoint · bake)
- `exit_code` 는 오늘 그대로다. 종료 보고의 `outcome` 을 result 에 따로 싣지 않는다 — step-phase 의
  StepResult 에 그 칸이 없고, exit code 가 같은 사실을 나른다

---

## 5. 종료 보고를 보내는 쪽

### 5.1 `Client.Exited` (`internal/enode/claim.go`)

```go
// Exited 는 명령 종료 보고다 (POST /v1/runs/{run}/steps/{seq}/exited).  30초 client 를 쓴다.
// 돌려주는 것 — 멈출지(stop)와 오류.  stop 이 참이면 다시 보내지 않는다.
func (c *Client) Exited(ctx context.Context, runID string, seq int, e contract.Exited) (stop bool, err error)
```

본문은 `contract.Exited` 그대로다 — `node` 는 `Ident.NodeID`, `instance` 는 `Client.Instance`,
`attempt` 는 `Step.Attempt`.

### 5.2 `exitReporter`

```go
// exitReporter 는 종료 보고 한 건을 goroutine 에서 보낸다.  Finalize 를 막지 않는다.
type exitReporter struct { /* 보낼 본문 · 멈춤 신호 · 끝남 신호 */ }

func (w *Worker) startExitReport(ctx context.Context, step *Step, e contract.Exited) *exitReporter
// Stop 은 result 를 보내기 직전에 부른다.  재시도를 그만두고 goroutine 이 끝나기를 기다린다.
func (r *exitReporter) Stop()
```

### 5.3 outcome 을 정하는 순수 함수

```go
// exitOutcome 은 session.Run 이 돌려준 것에서 종료 보고의 outcome 을 만든다.
// ok 가 거짓이면 보내지 않는다 — 프로세스가 뜨지 않았다.
func exitOutcome(code int, runErr error) (o contract.Outcome, ok bool)
```

### 5.4 `Job.Exited` (`internal/enode/runner.go` · 파일 행렬 밖 · 답 3 = A)

```go
type Job struct {
	// …
	// Exited 는 하네스 프로세스가 끝난 순간에 한 번 불린다.  nil 이면 안 부른다.
	// 봉투 해석과 버전 확인 앞이다 — exited_at 이 프로세스가 끝난 때를 가리키게.
	Exited func(code int, runErr error, at time.Time)
}
```

---

## 6. 업로드 client (`internal/enode` · `cmd/enode/main.go` · 답 8 = A)

```go
type Client struct {
	// … 오늘의 칸
	// Upload 는 요청마다의 제한이 없는 client 다.  마감은 업로드 예산의 ctx 가 준다.
	// PutBlob 과 UploadLog 가 쓴다.  nil 이면 HTTP 를 쓴다 (시험 · 옛 조립).
	Upload *http.Client
}

// PutBlob 은 산출물을 흘려 보낸다.  size 는 Content-Length 다.  body 를 통째로 읽지 않는다.
func (c *Client) PutBlob(ctx context.Context, runID string, seq int, name string, body io.Reader, size int64) error
```

- `UploadLog` 의 모양은 그대로다 (`[]byte`). 단계 로그는 노드가 이미 메모리에 들고 있다
- `cmd/enode/main.go` 가 `Upload: &http.Client{}` 를 채운다. 롱폴 client 와 같은 모양이지만 칸을 나눈다 —
  쓰임이 다르고, 한쪽을 조이면 다른 쪽이 따라 조여지면 안 된다
- 거절(413 · 422)과 전송 실패를 나누는 오류 타입을 둔다 — `upload` 칸의 error 는 전송 실패만이다

```go
// BlobRejected 는 Mediator 가 받고서 거절한 산출물이다 (4xx).  업로드 실패가 아니다.
type BlobRejected struct{ Status, Body string }
```

---

## 7. helper 요청 (`internal/enode/runc_overlay_linux.go`)

```text
   오늘                        이 유닛
   op "harvest"                op "finalize"
   request.Harvest             request.Finalize   *FinalizeSpec   (Deadline 을 싣는다)
   response.Harvest            response.Finalize  *FinalizeResult
   op "harvested"              op "finalized"
```

helper 는 같은 바이너리가 띄우므로 옛 이름과 함께 살 까닭이 없다. 이름을 바꾼다.

---

## 8. 영어 문구

단계 로그 끝의 줄과 error 칸의 문구는 `business-rules.md` 에 모았다. 모두 밖으로 나가는 문자열이라
영어다 (`CONVENTIONS.md` 2.1).
