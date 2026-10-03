# `checkpoint` — Code Generation 계획

**유닛** `checkpoint` (실패한 단계 보존 · 한 줄 순서의 여덟째) · **브랜치** `unit/checkpoint` · **기준** `03b6eef` (NFR Design 커밋) ·
**맡는 조각** 9 (보존 · 사람 · SunnyVM) · **병합 조건** 조각 9 가 초록

**이 계획이 이 유닛 Code Generation 의 기준이다.** 여기 없는 것은 짓지 않는다. 설계는 아래 입력의 일곱 문서다 — 「FD 규칙 · 엔티티 ·
흐름 N절」은 `construction/checkpoint/functional-design/` 의 셋, 「NFR N절」은 `nfr-requirements/nfr-requirements.md`, 「패턴 N절」 ·
「구성 N절」은 `nfr-design/` 의 둘이다. 내용을 여기에 옮겨 적지 않는다.

- **작성 시각** 2026-09-30T23:24:02Z · 규칙 `.aidlc/aidlc-rules/aws-aidlc-rule-details/construction/code-generation.md` Step 1 ~ 5
- **입력** FD 셋 · NFR 둘 (`nfr-requirements.md` · `tech-stack-decisions.md`) · NFR Design 둘 (`nfr-design-patterns.md` ·
  `logical-components.md`) · 유닛 정의 `unit-of-work.md` 8절 · 파일 행렬 1.3 의 ⑧ 열 · 팩 `scene-gates.md` 조각 9

---

## 1. 유닛 맥락

**완성하는 스토리** — US-2 (trash 와 보존본이 차지한 양 · spool 몫) · US-5 (실패한 단계의 upper 가 48시간 남는다) · US-11 (captured 의 뜻).
**완료 조건** 2 · 3 ③ · 7. **요구** FR-10. 노드 쪽만 바뀐다 — Mediator 는 코드를 안 고치고 다시 짓기만 한다 (FD 흐름 9절).

**새 이름**

```text
   internal/scratch   Mode · Policy · Capture · Entry · Store · Reservation · SpoolUsage · 상태 다섯과 원인 여섯의 상수 · NewID ·
                      OpenSpool · ReadRecord · WriteRecord · DecideAdmission · PlanSettle · PlanReconcile · ReadUsage · WriteUsage
   internal/enode     CheckpointConfig · CheckpointKeeper (Reserve · Decide · Finish · Dispose · Reported · Run · Kick) ·
                      CaptureSupport · KeepResult · CheckpointStatus · RunCheckpointCmd · StartScratch
                      Keep.By · Keep.Result · RuntimeCapability.Capture · Result.CheckpointCapture · Worker.Checkpoints
   internal/contract  Diagnostics.Checkpoint
   internal/panel     State.Checkpoint
```

**경계** — `internal/scratch` 는 표준 라이브러리와 `golang.org/x/sys` 만 쓴다 (`boundary_test.go:102` · 봉인 그대로 초록이어야 한다).
`go.mod` 를 안 움직인다. **크로스 빌드** — flock · lstat · `O_NOFOLLOW` · statfs 의 inode · `Renameat2` 는 unix 전용이라 `_unix.go` 와
`_other.go` 짝으로 두고 `_other` 는 「지원하지 않음」을 돌려준다 (유닛 정의 10절). linux/arm 은 32비트 — statfs 칸의 타입을 lower-state 가
쓴 방식대로 바꿔 담는다.

**기대는 것 (다른 유닛)** — `scratch.Trash` · `Measure` · `Deleter` (trash) · `Keep{Upper}` 와 `closeOut` 의 `closing` (bake) ·
`lower.ReadRoot` · `lower.ReadMetadata` (lower-state) · `contract.CheckpointCapture` (step-phase). 모두 `main` 에 있다.

---

## 2. 파일

| 파일 | 새 · 고침 | 행렬 | 까닭 |
|---|---|---|---|
| `internal/scratch/checkpoint.go` | 새 | 안 | 타입 · 상수 · ID · 판정과 계획의 순수 함수 · 요약 (엔티티 1 · 3 · 4절 · 규칙 2 · 5 · 6 · 9절) |
| `internal/scratch/checkpoint_unix.go` · `checkpoint_other.go` | 새 | 안 | spool 자리 확인 · 잠금 · 기록 읽기와 쓰기 · Store 의 파일 일 (규칙 11절 · 패턴 2 · 3절) |
| `internal/scratch/checkpoint_test.go` · `checkpoint_unix_test.go` | 새 | 시험 | 6절 |
| `internal/scratch/deleter.go` | 고침 | **밖** (trash 의 파일) | `Usage` 에 spool 칸 넷 (엔티티 7절) — `Usage` 가 이 파일에 있다 |
| `internal/scratch/remove_unix.go` · `remove_other.go` | 고침 | **밖** (FD 흐름 9절) | `Measure` 가 spool 뿌리도 받는다 (흐름 5절) |
| `internal/contract/result.go` | 고침 | **밖** (FD 흐름 9절) | `Diagnostics.Checkpoint` (FD 답 9) |
| `internal/enode/checkpoint.go` | 새 | **밖** (FD 흐름 9절) | Keeper · 규칙 1절 · 옮겨 담기 · 문장 · 판정 고루틴 · `StartScratch` (3.2) |
| `internal/enode/checkpoint_cmd.go` | 새 | **밖** (FD 흐름 9절) | `RunCheckpointCmd` (규칙 13절) |
| `internal/enode/checkpoint_test.go` · `checkpoint_cmd_test.go` · `config_checkpoint_test.go` | 새 | 시험 | 6절 |
| `internal/enode/runtime.go` | 고침 | 안 | `Keep.By` · `Keep.Result` · `KeepResult` · `CaptureSupport` · native 의 Capture (엔티티 5절) |
| `internal/enode/runc_overlay_linux.go` · `runc_overlay_other.go` | 고침 | 안 | Close 의 keep (By · Result · 오류에 안 실음) · Capability (규칙 3절) |
| `internal/enode/claim.go` | 고침 | 안 | `Result.CheckpointCapture` · 세션을 연 뒤 예약 · `closing` 칸 · closeOut 의 차례 · report 의 성패 · 예약 버리기 (흐름 1 · 3절) |
| `internal/enode/bake_build.go` | 고침 | **밖** (FD 흐름 9절) | 세션을 연 뒤 예약 · `failEnd` 의 `closing` 칸 (흐름 2절) |
| `internal/enode/trash_linux.go` · `trash_other.go` | 고침 | **밖** (FD 흐름 9절) | trash-helper 의 측정 동작과 그 launcher (흐름 5절 · 패턴 4절) |
| `internal/enode/config.go` | 고침 | 안 | `Local.Checkpoint` · 검증 (규칙 10절) |
| `internal/enode/status.go` | 고침 | 안 | `Status.Checkpoint` · `SetSpool` · `SetScratch` 가 spool 칸을 지킨다 (엔티티 7절) |
| `internal/panel/view.go` · `page.go` | 고침 | 안 · **밖** (`page.go` · FD 흐름 9절) | `State.Checkpoint` · 제어판 줄 (규칙 12절) |
| `cmd/enode/main.go` | 고침 | 안 | `checkpoint` 입구 한 줄 · runc-overlay 기동을 `StartScratch` 한 줄로 (3.2) |
| `scripts/finalize-bake/slice-9.sh` | 새 | 안 (조각 스크립트) | 7절 |
| 시험 고침 — `runc_overlay_linux_test.go` · `finalize_worker_test.go` · `status_test.go` · `trash_test.go` · `runc_overlay_integration_test.go` · `internal/panel/panel_test.go` · `cmd/enode/main_test.go` · `internal/contract` 의 result 시험 | 고침 | 시험 | 6절 |

**행렬 밖은 열이다** — FD 흐름 9절의 여섯 (`result.go` · `bake_build.go` · `page.go` · `trash_linux.go` · `trash_other.go` · `remove_unix.go`,
새 파일 둘) 에 이 계획이 둘을 더한다 — `internal/scratch/deleter.go` (`Usage` 의 자리 · FD 에서 빠졌다) 와 `remove_other.go` (짝).

---

## 3. 기준선 · 코드 검사 · 이 계획이 정한 것

### 3.1 기준선 (2026-10-01 · 이 기계 · `03b6eef`)

CI 의 커버리지 명령 (`ci.yml:267` · 시험 DB · 가짜 claude 스텁 · `-coverpkg=./...`) 과 `:269` ~ `:300` 의 awk 그대로. 작업 트리는 안 바뀌었다.

| 패키지 | 커버리지 | 문장 |
|---|---|---|
| `cmd/enode` | 80.8% | 198 / 245 |
| `internal/enode` | 86.7% | 4,584 / 5,286 |
| `internal/panel` | 89.1% | 245 / 275 |
| `internal/contract` | 92.5% | 760 / 822 |
| `internal/scratch` | 93.5% | 231 / 247 |
| 전체 | 87.7% | 12,554 / 14,319 |

**흔들리는 시험 하나** — 첫 전체 실행에서 `TestFinalize_ACommandKilledByASignal` 이 빨갛다 (「exit report = []」). 혼자 30 번은 모두 초록 ·
둘째 전체 실행은 초록이다. 종료 보고는 goroutine 이 보내고 `reporter.Stop()` 이 첫 요청 앞에서 끊을 수 있다 (`finalize.go:155-193`) —
설계대로 「유실돼도 result 가 같은 사실을 싣는다」 인데 시험은 보고가 정확히 하나라고 본다. finalize 유닛의 시험이다 — 물음 1.

### 3.2 `cmd/enode` 의 여유

80% 까지 남은 문장이 둘이다 (198 - 0.8 x 245 = 2). runc-overlay 기동 갈래는 기본 `go test` 에서 안 닿는다 (trash `code-summary.md:146-149`).

- **runc-overlay 기동을 `internal/enode` 의 `StartScratch` 하나로 옮긴다** — 기동 청소 · 삭제자 · Keeper · 판정 고루틴 · `AfterReport`
  합치기. `main.go` 에는 부르는 줄 하나. trash 요약이 적어 둔 길이다. `StartScratch` 는 가짜 Launch 로 기본 `go test` 에서 덮는다
- `checkpoint` 입구는 `runtime-helper` · `trash-helper` 옆의 한 줄이고 `main_test.go` 가 덮는다
- Step 16 에서 병합 전과 뒤를 적는다. 80% 아래면 병합하지 않는다

### 3.3 코드 검사 (유닛 정의 0절 · CI 와 같은 명령)

| 검사 | 명령 |
|---|---|
| 포맷 · vet · 빌드 | `gofmt -l .` · `go vet ./...` · `go vet -tags integration ./internal/enode/ ./internal/scratch/` · `go build ./...` |
| 시험 | `eval "$(scripts/testdb.sh)"` · 스텁 PATH (`ci.yml:63-70`) · `go test ./... -count=1` — 실패 0 · 스킵 0 |
| 커버리지 | `ci.yml:267` 의 명령과 awk — 패키지마다 80% 이상 |
| 스킵 감시 | `ci.yml:341` 의 스텝 그대로 |
| 경계 | `internal/panel/boundary_test.go` (기본 `go test` 안) |
| 크로스 빌드 | `ci.yml:456-458` 의 셋 |
| 심볼 상한 | `ci.yml:481-483` — `enodectl.exe` 의 crypto/tls T 10 · net/http T 50 |
| 장식 문자 | `ci.yml:79` (U+2605 0) · `go run ./scripts/glyphscan.go` |
| 린트 | `golangci-lint run ./...` — 수가 Step 1 에서 적은 기준선보다 늘지 않는다 |
| 조각 0 | 위 셋 초록 · `grep -c 'mux.HandleFunc' internal/api/api.go` 가 19 그대로 |

### 3.4 이 계획이 정한 것 (FD · NFR 에 적히지 않은 자리)

| # | 정한 것 | 까닭 |
|---|---|---|
| ① | lower 신원은 `lower.ReadRoot(워크스페이스)` 의 키 (filesystem 번호 · inode), head 와 ir 은 `lower.ReadMetadata`. 둘 다 읽기만 · Decide 에서 | FD 엔티티 3절의 「lower-state 의 Identity」는 `lower.json` 의 내용이다 — 워크스페이스마다 늘 있는 값은 Root 키다. `lowerguard.go` 를 안 고친다 |
| ② | environment 는 `Worker.RuntimeRecord.PreparedEnvironment` | 굽기의 초안이 같은 값을 쓴다 (`bake_build.go:484`) |
| ③ | 판정 고루틴과 예약 버리기는 Keeper 안, 측정 launcher 는 `TrashLauncher` 옆의 `MeasureLauncher` | 구성 1절 |
| ④ | Keeper 가 nil 이면 오늘과 같다 (칸 없음 · 예약 없음) | FD 규칙 4절 · 시험과 옛 조립이 그대로 돈다 |
| ⑤ | runc-overlay 기동을 `StartScratch` 로 (3.2) | `cmd/enode` 여유 |

---

## 4. 단계 — 열일곱

### Step 1 — 기준선과 흔들리는 시험

- [x] `unit/checkpoint` 가 `03b6eef` 위에 있다 · 3.1 의 숫자를 다시 적는다 · `golangci-lint` 경고 수를 적는다
- [x] 물음 1 의 답대로 `TestFinalize_ACommandKilledByASignal` 을 다룬다

### Step 2 — `internal/contract` (FD 답 9)

- [x] `result.go` — ``Diagnostics.Checkpoint string `json:"checkpoint,omitempty"` ``
- [x] 시험 — JSON 왕복 · 빈 값이면 칸이 없다 · `store.StepResult` 가 봉인으로 되푼다

### Step 3 — `internal/scratch` 의 타입과 순수 함수 (엔티티 1 · 3 · 4절 · 규칙 2 · 5 · 6 · 9절 · 패턴 2 · 3절)

- [x] `checkpoint.go` — `Mode` · `Policy` · `Capture` · `Entry` · 상태와 원인 상수 · `NewID` (crypto/rand 6 바이트 · hex) · `DecideAdmission`
      (규칙 2절 ② ~ ④ 를 사실로 · 상수 시간) · `PlanSettle` (만료 · 하나의 상한 · 두 몫 · 오래된 것부터 · 전체 inode 0 이면 건너뜀) ·
      `PlanReconcile` (규칙 9절의 여섯 줄) · 사유 글 (`larger than max_gb` · `over capacity_percent` · `over max_total_inodes`)
- [x] `deleter.go` — `Usage` 에 `SpoolBytes` · `Checkpoints` · `SpoolUnsized` · `SpoolMeasuredAt`
- [x] 시험 — 6절의 `internal/scratch` 순수 함수 줄

### Step 4 — Store 의 파일 일 (규칙 9 · 11절 · NFR C1 · C7 · C8 · 패턴 1 ~ 3절)

- [x] `checkpoint_unix.go` — `OpenSpool` (lstat · 없으면 0700 · 느슨한 비트 fchmod · symlink · 디렉터리 아님 · 남의 것은 거절) · `Reserve`
      (`O_NOFOLLOW` · Mkdir · 항목 잠금 · `reserved` 기록) · `Commit` · `Abandon` (`Trash.Move`) · `Dispose` · `Reported` ·
      `ReadRecord` / `WriteRecord` (1 MiB · 보통 파일 · 임시 파일과 rename · 0600 · fsync 없음) · `ReadUsage` / `WriteUsage` · spool 잠금 ·
      `Settle` (읽기 → 잠금 밖 측정 → 다시 읽고 적기) · `Reconcile` · `List` (쥐어진 `reserved` 는 뺀다 · `LOCK_NB` 시험)
- [x] `checkpoint_other.go` — 「지원하지 않음」
- [x] 시험 — 6절의 `internal/scratch` 파일 줄

### Step 5 — 측정 입구 (흐름 5절 · 패턴 4절)

- [x] `remove_unix.go` · `remove_other.go` — `Measure` 가 받는 뿌리를 trash 로 못 박지 않는다 (경계 그대로) — 고칠 것이 없었다. 이미 어느 뿌리든
      `O_NOFOLLOW` 로 열어 받는다 (code-summary 「계획과 다른 자리」)
- [x] `trash_linux.go` — trash-helper 의 측정 동작 (지우지 않는다 · idle · CPU 19) · `MeasureLauncher(spool)` · `trash_other.go` 짝
- [x] 시험 — `trash_test.go` 에 측정 동작 (namespace 없이 부른다) · argv 모양

### Step 6 — 런타임의 keep 과 보존 지원 (엔티티 5절 · 규칙 3절)

- [x] `runtime.go` — `Keep.By` · `Keep.Result` · `KeepResult` · `CaptureSupport` · `RuntimeCapability.Capture` · native 는 `runtime`
- [x] `runc_overlay_linux.go` — Close 에서 helper 가 끝난 뒤 `By` 가 지났으면 옮기지 않고 `Late` · rename 실패와 abort 는 `Err` ·
      `Result` 가 있으면 Close 오류에 안 싣는다 · 없으면 (굽기) 오늘 그대로 · Capability 에 Capture
- [x] `runc_overlay_other.go` — linux 판과 같은 Capability
- [x] 시험 — 6절의 런타임 줄 (가짜 helper)

### Step 7 — Keeper (규칙 1 · 2 · 8 · 11 · 14절 · 패턴 1절)

- [x] `checkpoint.go` (enode) — `CheckpointKeeper` · 설정에서 `scratch.Policy` · `Reserve(step)` (세션을 연 뒤 · 실패는 errno 만) ·
      `Decide(facts)` (규칙 1절 → 받아들임 · 요약 읽기 · 신원 3.4 ① ②) · `Finish` (closedAt 뒤 · 규칙 2절 ⑥ ⑦) · `Dispose` · `Reported` ·
      `scratch.Capture` → `contract.CheckpointCapture` · `diagnostics.checkpoint` 문장 (규칙 2절 · errno 만) · 노드 로그 (규칙 14절)
- [x] 시험 — 6절의 Keeper 줄

### Step 8 — Worker 에 잇는다 (흐름 1 · 2 · 3절 · 규칙 4절)

- [x] `claim.go` — `Result.CheckpointCapture` · `Worker.Checkpoints` · 세션을 연 뒤 (`:721`) `Reserve` · `closing` 에 `failed` · `bake` ·
      예약 · `closeOut` 차례 (Finalize → diagnostics → Decide → `Close(Keep{Upper, By, Result})` → closedAt → Finish) · 칸이 없는 끝과 요구하지
      않은 끝은 보고 뒤 `Dispose` · `report` 의 네 끝을 `Reported` 로
- [x] `bake_build.go` — 세션을 연 뒤 (`:202`) `Reserve` · `failEnd` 는 `closing{after, failed: true, bake: true}` · 성공 끝과 `stopEarly` 는 `Dispose`
- [x] 시험 — 6절의 closeOut 줄 · P2 · C2

### Step 9 — 설정 (규칙 10절 · NFR 1절)

- [x] `config.go` — `Local.Checkpoint *CheckpointConfig` · 기본 (on-failure · 48 · 20 · 32 · 몫) · 검증 다섯 줄의 영어 문장
- [x] 시험 — `config_checkpoint_test.go` 의 표

### Step 10 — 상태 파일과 제어판 (규칙 12절)

- [x] `status.go` — `Status.Checkpoint` · `SetSpool` · `SetScratch` 는 spool 칸을 안 덮는다 · `sameUsage` 에 새 칸
- [x] `view.go` — `State.Checkpoint` (상태 파일에서) · `page.go` — spool 줄 · 정책 줄 · 안내 줄 · off · native (규칙 12절의 한국어 초안)
- [x] 시험 — `status_test.go` · `panel_test.go`

### Step 11 — 조회 명령 (규칙 13절 · FD 답 6 · 7 · 8)

- [x] `checkpoint_cmd.go` — `RunCheckpointCmd(args, stdout, stderr) int` · `--config` · `--json` · 목록 · show · 모르는 ID exit 1 · scratch 없음
- [x] `cmd/enode/main.go` — `checkpoint` 입구 한 줄 (플래그 앞)
- [x] 시험 — `checkpoint_cmd_test.go` (출력 글자 그대로) · `main_test.go` (입구가 갈린다)

### Step 12 — 기동과 판정 고루틴 (흐름 4절 · 규칙 6 · 9 · 12절 · 패턴 4절)

- [x] `checkpoint.go` (enode) — `Run` (조정 → 판정 → 깸 · 10분) · `Kick` (자리 하나짜리 채널) · 기동 로그 세 가지 · 상태 파일의 블록 ·
      `StartScratch` (기동 청소 · 삭제자 · Keeper · 고루틴 · `AfterReport` 에 삭제자와 Keeper 의 Kick)
- [x] `cmd/enode/main.go` — runc-overlay 갈래를 `StartScratch` 한 줄로 · native 는 상태 파일의 블록과 로그 한 줄
- [x] 시험 — 6절의 기동 줄

### Step 13 — 경계와 어휘

- [x] `boundary_test.go` 가 그대로 초록이다 (`internal/scratch` 가 새로 부르는 것은 표준 라이브러리뿐) · 두 벌의 어휘 시험 (엔티티 8절)

### Step 14 — integration 시험 (사람 · SunnyVM · CI 밖)

- [x] `runc_overlay_integration_test.go` — 6절의 integration 줄 · `go vet -tags integration` 로 컴파일만 여기서 확인

### Step 15 — 조각 9 스크립트 (7절)

- [x] `scripts/finalize-bake/slice-9.sh` · `bash -n` · 사내 이름 0 · 출력 영어 · 주석 한국어

### Step 16 — 코드 검사 (3.3)

- [x] 3.3 의 표 전부 · 커버리지 병합 전과 뒤 (`cmd/enode` · `internal/enode` · `internal/scratch` · `internal/panel` · `internal/contract`)
- [x] 흔들림 — 새 시험을 `-count=20` · `go test -race` (`internal/enode` · `internal/scratch`)
- [x] `git status` 로 2절 밖의 파일이 없는지 · `cmd/enodectl/probe.lock` 이 바뀌면 되돌린다

### Step 17 — 요약 · 상태 · 감사 · 커밋

- [x] `construction/checkpoint/code/code-summary.md` — 파일 · 규칙의 자리 · 3.4 · 계획과 다른 자리 · 코드 검사 숫자 · 조각 9 를 돌리는 법 ·
      넘기는 것 · 정본 되돌림 (FD 흐름 12절 · NFR 8절)
- [x] 표기 검사 · 말투 · 사내 이름 · 이 계획의 체크박스 · `aidlc-state.md` · `audit.md` (진행자)
- [x] 한 커밋 — 승인 뒤 (CONVENTIONS 3.3)

---

## 5. 하지 않는 것

```text
   Mediator 코드 · 매칭 · 광고 어휘         안 바뀐다 (FD 규칙 4절 · NFR C6)
   운영자의 명시 요청 정책                   짓지 않는다 (FD 흐름 12절)
   bake 규칙 3 의 근거 문장                  진행자 (FD 흐름 11절)
   조각 9 를 돌리는 일                       사람의 조각이다.  스크립트만 준비한다
   정본 되돌림                              진행자가 올린다 (FD 흐름 12절 · NFR 8절)
   PR 과 병합                               조각 9 가 초록인 뒤.  올리기 전에 묻는다
```

---

## 6. 시험 목록

**기본 `go test` (CI)** — 규칙마다 표 시험.

| 자리 | 시험 이름 | 보는 것 |
|---|---|---|
| scratch 순수 | `TestDecideAdmission_Order` | 규칙 2절 ② ~ ④ 의 차례 · 여유가 몫보다 먼저 |
| | `TestNewID_ShapeAndCollision` | 12 hex · 겹치면 다시 |
| | `TestPlanSettle` | 만료 · `max_gb` · 몫에서 오래된 것부터 (새 것도) · inode 몫 · 전체 inode 0 이면 건너뜀 · 몫은 한 번 |
| | `TestPlanReconcile` | 규칙 9절 여섯 줄 (잠금 파일 없음과 1시간 · 빈 기록 · 1 MiB 넘음 · 형제의 잠금 · 만료 · `unknown`) |
| scratch 파일 | `TestOpenSpool_NarrowsAndRefuses` | C8 — 느슨한 비트 · symlink · 파일 · 남의 것 (남의 것은 uid 를 바꿀 수 없어 판정 함수로) |
| | `TestSpoolModes_IgnoreUmask` | C1 — umask 0 에서 0700 · 0600 |
| | `TestRecord_ReadLimits` | C7 — symlink · 1 MiB · 보통 파일 아님 |
| | `TestReserveCommitDispose` | 예약 · 확정 · 버림 · 겹침 |
| | `TestSettle_MeasuresOutsideTheLock` | 측정하는 동안 spool 잠금을 다른 fd 로 쥘 수 있다 · 그새 퇴출된 항목의 값은 버린다 |
| | `TestSettle_LaunchErrorStops` · `TestUsageSummary_RoundTrip` · `TestList_HidesHeldReservations` | |
| 런타임 | `TestRuncOverlayClose_KeepLatePastBy` · `TestRuncOverlayClose_KeepRenameFails` · `TestRuncOverlayClose_KeepNotInCloseError` · `TestRuncOverlayClose_PendingKeepUnchanged` | 규칙 3절 — 가짜 helper |
| Keeper | `TestCheckpointRequested` | 규칙 1절 표 (명령 · agent · 굽기 x off · on-failure · always) |
| | `TestCheckpointVocabulary_MatchesContract` · `TestContractHasNoCheckpointField` | 엔티티 8절 · C6 (reflect) |
| | `TestCheckpointDetail_ErrnoOnly` | C3 — `PathError` · `LinkError` 에서 경로가 안 나온다 |
| closeOut | `TestCloseOut_KeepResults` | 옮김 · 늦음 · 실패마다 receipt · diagnostics · finalize 칸 그대로 |
| | `TestCloseOut_OtherCloseErrorsStillFail` | 규칙 3절 — unmount · trash 옮기기 오류는 finalize 칸 error |
| | **`TestCheckpoint_SlowReservationKeepsFinalize`** | P2 — 느린 예약 · 느린 확정을 끼운 가짜 store, 짧은 Finalize 예산 → finalize `ok` · error 없음 · exit 그대로 |
| | **`TestCheckpoint_NoHostPathLeaves`** | C2 — 표지 글자를 담은 scratch 경로로 규칙 2절의 갈래 전부 → result JSON · 올린 단계 로그에 그 글자가 없다 |
| | `TestReserve_DisposedWhenNotRequested` · `TestReserve_FailureReportedAtClose` · `TestReport_RecordsOutcome` | 패턴 1절 · 규칙 8절의 네 끝 |
| 설정 · 상태 · 제어판 | `TestCheckpointConfig_Validate` · `TestStatus_SetSpoolKeepsTrash` · `TestPanel_CheckpointLines` | 규칙 10 · 12절 |
| 조회 | `TestCheckpointCmd_List` · `_Show` · `_JSON` · `_UnknownID` · `_NoScratch` | 규칙 13절 글자 그대로 |
| 기동 | `TestKeeperRun_ReconcileThenSettle` · `TestKeeperRun_KickCoalesces` · `TestStartScratch_WiresAfterReport` · `TestMain_CheckpointEntry` | 흐름 4절 · 패턴 4절 |
| contract | `TestDiagnostics_CheckpointRoundTrip` | Step 2 |
| trash-helper | `TestTrashHelper_MeasureOnly` | 지우지 않는다 |

**integration 태그 (사람 · SunnyVM · CI 밖)**

| 시험 | 보는 것 |
|---|---|
| `TestIntegrationCheckpoint_CapturesSubuidAndWhiteouts` | 실제 upper (subordinate uid 항목 · whiteout · opaque) 가 spool 로 옮겨지고 그대로다 |
| `TestIntegrationCheckpoint_MeasureInHelper` | helper 가 spool 의 항목을 측정한 값이 `du` 의 블록 합과 같다 |
| `TestIntegrationCheckpoint_UnshareReadsRootFiles` | C4 — `show` 의 안내 명령으로 컨테이너 root 가 쓴 0600 파일이 읽힌다 (NFR 5절의 잔여) |
| `TestIntegrationCheckpoint_WindowIndependentOfSize` | P1 · P3 — 큰 upper 와 빈 upper 의 창 차이 · 한가할 때 p99 기록 |

---

## 7. 조각 9 — `scripts/finalize-bake/slice-9.sh`

`slice-4.sh` 의 모양을 따른다 — 이미 도는 노드를 쓰고 · 판정하지 않고 무엇을 볼지 출력한 뒤 Enter 를 기다리고 · 데몬 재시작과 설정 고침은
사람에게 맡긴다 (`todo`). 입력 — `M` · `T` · `NODE_CONFIG` (runc-overlay 노드 · 조각용 설정 `ttl_hours: 1` · `max_gb: 1`) · `WS` ·
`NATIVE_CONFIG` · `NATIVE_WS` · `ENODE` (조회 명령) · `SLICE_DIR` · `PAUSE`.

**차례 — TTL 을 먼저 건다.** 4번은 한 시간과 10분을 기다려야 하므로 첫 보존본을 가장 먼저 잡고, 기다리는 동안 나머지를 돈다.

| 차례 | FD 흐름 7절 | 하는 일 | 보는 것 |
|---|---|---|---|
| 1 | 4 · 1 | exit 1 단계와 exit 0 단계를 낸다 · 앞의 ID 와 시각을 적는다 (TTL 시계 시작) | `captured` · ID · `expires_at` 이 한 시간 뒤 · 뒤는 `not_requested` |
| 2 | 2 | `enode checkpoint list` · `show <ID>` | `kept` · 보고 뒤 크기 · `delivered` · path · open · discard 줄 |
| 3 | 3 | 1 GiB 넘는 upper 를 남기고 exit 1 | receipt `captured` · 몇 초 뒤 `list` 에 `evicted: larger than max_gb` |
| 4 | 5 | native 노드에 exit 1 단계 | `unsupported`(`runtime`) · 문장 |
| 5 | 8 | 주인 없는 `reserved` 항목을 만들고 데몬 재시작 (todo) · 긴 단계를 도는 중에 `list` | 재시작 뒤 그 항목이 trash 로 가고 삭제자가 지운다 · 도는 단계의 예약은 목록에 없다 |
| 6 | 7 | 설정을 `policy: off` 로 재시작 (todo) · 1 의 실패 계약을 다시 | exit_code · produced · finalize 칸이 1 과 같다 · `not_requested` |
| 7 | 6 | 설정을 on-failure · `min_free_gb` 를 지금 여유보다 1 GB 작게 두고 재시작 (todo) · 단계가 2 GB 를 쓰고 exit 1 | `rejected`(`free_space`) · 문장. 미리 여유를 채우면 노드가 drain 해 단계를 못 집는다 · 끝나면 설정을 되돌린다 (todo) |
| 8 | 4 | 1 의 `expires_at` 과 10분이 지날 때까지 기다린다 | `list` 에 없음 · `show <ID>` exit 1 · Record 는 `captured` 그대로 |

기다림 뒤 전체는 약 한 시간 반이다. 출력은 영어, 주석은 한국어. `bake-common.sh` 의 `look` · `todo` · `api` 를 쓴다.

---

## 8. 물음

답을 `[Answer]:` 뒤에 적는다. 권장을 **A** 에 둔다.

**내기 전에 대 본 것**

| 흠 | 대 본 결과 |
|---|---|
| 이미 정한 것을 다시 묻기 | 파일 · 차례 · 시험 · 3.4 는 FD · NFR · NFR Design 과 코드에서 한 가지로 나온다. 묻지 않는다 |
| 측정 없는 근거 | 물음 1 은 3.1 의 두 번의 전체 실행과 혼자 30 번에 기댄다 |
| 앞 유닛 넘김과 어긋남 | 물음 1 의 A 는 finalize 유닛의 시험 파일 하나를 고친다 — 제품 코드는 안 고친다 |
| 요구를 빼는 선택지 | 두 선택지 모두 코드 검사 (실패 0) 를 지킨다 — B 는 다시 돌려서 지킨다 |

### Question 1 — 흔들리는 finalize 시험 (`TestFinalize_ACommandKilledByASignal`)

이 기계의 전체 실행 두 번 가운데 한 번 빨갛고 혼자는 30 번 모두 초록이다. 종료 보고는 설계대로 잃을 수 있는데 (`finalize.go:188` 「result 가
같은 사실을 싣는다」) 시험이 보고가 정확히 하나라고 본다. 이 유닛의 병합 조건은 코드 검사의 실패 0 이다.

A) 이 유닛의 Step 1 에서 시험만 고친다 — 종료 보고가 0 또는 1 개이고, 1 개면 signal 9 인지 본다. `finalize_worker_test.go` 한 곳 · 행렬 밖 ·
제품 코드 불변

B) 고치지 않는다 — 빨가면 CI 를 다시 돌리고 진행자에게 넘긴다

C) Other (please describe after [Answer]: tag below)

**권장 A.** 이 유닛이 closeOut 에 일을 더하므로 그 시험의 창이 더 흔들릴 수 있다. 고침은 시험의 기대를 설계에 맞출 뿐이다.

[Answer]: A

---

## 9. 확장 준수 — Code Generation

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. NFR C1 ~ C8 은 6절의 시험이 확인한다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 규칙마다 표 시험 |
