# `step-phase` — Code Generation 계획

**유닛** `step-phase` (한 줄 순서의 둘째 · Mediator 진행 구간) · **브랜치** `unit/step-phase` ·
**기준** `242ca33` (Functional Design 커밋) · **맡는 조각** 0 (기동이 안 깨졌다) ·
**병합 조건** 조각 0

**이 계획이 이 유닛 Code Generation 의 기준이다.** 여기 없는 것은 짓지 않는다. 설계는
`construction/step-phase/functional-design/` 의 셋이다 — 타입은 `domain-entities.md`, 규칙과 응답 문구는
`business-rules.md`, 순서와 흐름은 `business-logic-model.md`. 아래의 **FD** 는 Functional Design 산출물의
줄임이다 — 「FD 규칙 1절」은 `business-rules.md` 1절, 「FD 흐름 4절」은 `business-logic-model.md` 4절,
「FD 엔티티 2절」은 `domain-entities.md` 2절을 뜻한다.

- **작성 시각**: 2026-09-25T08:52:58Z 이후 (Functional Design 승인 뒤)
- **입력**: FD 셋 · `unit-of-work.md` 2절 · `unit-of-work-file-matrix.md` 1.1절 · `unit-of-work-story-map.md` ·
  `scene-gates.md` 조각 0 · 요구 문서 `requirements.md` 6절 (조각 0 의 라우트 수)
- **NFR 두 단계** (NFR Requirements · NFR Design — 비기능 요구와 그 설계): 사용자 결정으로 건너뛰었다
  (2026-09-25T08:52:58Z). 그 둘이 보려던 것을 이 계획의 3절이 받는다

---

## 1. 유닛 맥락

**하는 일** — Mediator 가 단계의 진행 구간을 적고 보이게 한다. 노드 쪽은 한 줄도 안 바뀐다. 노드가
종료 보고를 보내고 result 의 새 칸을 채우는 일은 뒤 유닛(finalize · checkpoint · bake)의 것이다.

```text
   steps 의 새 칸 셋          phase · phase_since · exit
   종료 보고 받기             POST /v1/runs/{run}/steps/{seq}/exited — 수락 표 열셋 (FD 규칙 1절)
   result 의 새 칸 아홉       exited_at · finalized_at · finalize · upload · reason · diagnostics ·
                             checkpoint_capture · build · merge.  값으로 거절하지 않는다
   진행 조회                  단계마다 phase · phase_since · exit.  QUEUED 면 요구 줄마다 후보 수 셋
   Claimed 의 새 칸 일곱      effect · budget · discover · sync · builds · ir · merge (U1 이 넘긴 일)
   Record 의 새 칸 넷         exited_at · finalized_at · exit · last_phase
```

**완성하는 스토리** — US-7 (계약 작성자가 QUEUED 인 자기 Run 의 후보가 점유돼서인지 drain 중이라서인지
안다) 과 완료 조건 4 (같은 뜻 · `stories.md` 2절). **받치는 스토리** — US-8 (finalizing 에 오래 있을 때의
상한 — `phase_since` 를 이 유닛이 낸다. 완성은 finalize).

**앞 유닛** — U1 `contract-grammar` (계약 문법. PR #61 로 병합). 쓰는 것은 `contract.KindMerge` · `Effect` ·
`Budget` · `Build` · `Merge` 다. **뒤 유닛에 넘기는 것** — FD 흐름 10절 그대로다.

```text
   finalize     exited 를 보낸다 · 200 과 409 에서 멈춘다 · 404 와 405 면 그 Run 동안 다시 안 보낸다 ·
                result 에 exited_at · finalized_at · finalize · upload · reason · diagnostics 를 채운다
   checkpoint   result 에 checkpoint_capture 를 채운다
   bake         build 단계의 exited · result 에 build 와 merge 를 채운다
```

**새로 내보내는 이름**

```text
   internal/contract (새 파일 result.go)
     타입     Exited · Outcome · Stage · Diagnostics · CollectNote · CheckpointCapture ·
              BuildManifest · BuildRecord · Pinned · MergeResult · MergeOps
     상수     OutcomeExit · OutcomeSignal · OutcomeTimeout · StageOK · StageTimeout · StageError ·
              ReasonFinalizeTimeout · ReasonUploadTimeout · ReasonMergeWaitTimeout · ReasonBakeInProgress ·
              ChangesMeasured · ChangesNotMeasured · ChangesPartial ·
              LimitVisits · LimitTime · LimitMemory · LimitSize ·
              CaptureNotRequested · CaptureUnsupported · CaptureRejected · CaptureCaptured · CaptureFailed
     메서드   Exited.Check
   internal/store
     상수     PhaseRunning · PhaseFinalizing · PhaseWaiting
     오류     ErrNoSuchStep · ErrExitRejected
     타입     Candidates
     메서드   Store.MarkExited · Store.CandidatesFor · StepResult.OutOfVocabulary (4절 ②)
```

**자료** — `steps` 표에 칸 셋 (`ALTER TABLE steps ADD COLUMN IF NOT EXISTS`). 값 제약(CHECK)은 안 건다
(FD 엔티티 1절). 옛 행은 셋 다 NULL 이다. `phase` 와 `exit` 는 Postgres 17 의 예약어가 아니다 —
`pg_get_keywords()` 로 확인했다 (2026-09-25).

**경계** — `internal/contract/result.go` 는 표준 라이브러리 `time` · `errors` · `fmt` 만 쓴다.
`internal/record` 는 오늘처럼 표준 라이브러리만 쓴다 — 새 칸 `exit` 는 `json.RawMessage` 로 받는다
(contract 를 가져오지 않는다). 코드 경계 시험의 금지 표가 바뀌지 않는다.

---

## 2. 고치는 파일

| 파일 | 새 · 고침 | 행렬 | 무엇 |
|---|---|---|---|
| `internal/contract/result.go` | 새 | 밖 (FD 흐름 9절) | 결과 어휘 · `Exited.Check` |
| `internal/store/schema.sql` | 고침 | 있음 | 칸 셋 |
| `internal/store/claim.go` | 고침 | 있음 | phase 상수 · claim 과 재전달이 phase 를 적는다 · `MarkExited` 와 분류 · `StepResult` 새 칸 아홉과 `OutOfVocabulary` · `Claimed` 칸 일곱 · `fillFromContract` |
| `internal/store/rollback.go` | 고침 | 밖 (FD 흐름 9절) | 되돌림 둘이 칸 셋을 지운다 |
| `internal/store/ask.go` | 고침 | 밖 (FD 흐름 9절) | 되감기 하나가 칸 셋을 지운다 |
| `internal/store/observe.go` | 고침 | 있음 | `StepView` 칸 셋 · `RequireView.Candidates` |
| `internal/store/queue.go` | 고침 | 있음 | `Candidates` · `CandidatesFor` |
| `internal/store/seal.go` | 고침 | 밖 (FD 흐름 9절) | `StepFiles` 가 Record 의 새 칸 넷을 채운다 |
| `internal/record/record.go` | 고침 | 있음 | `StepFile` 칸 넷 |
| `internal/api/api.go` | 고침 | 있음 | 라우트 한 줄 · `postExited` · `postResult` 의 경고 로그 · `runView.CandidatesAt` · `getRun` 의 후보 수 |
| `internal/contract/result_test.go` | 새 | 시험 | 본문 검사 표 · JSON 칸 이름 |
| `internal/store/exited_test.go` | 새 | 시험 | 수락 표 · 동시 재전송 · phase 의 한 생애 · 되돌림 · Record · 어휘 밖 값 |
| `internal/store/candidates_test.go` | 새 | 시험 | 후보 수 셋 · 배타 · 셈의 시각 · 비용 벤치마크 (3절) |
| `internal/api/exited_test.go` | 새 | 시험 | HTTP 로 400 · 404 · 409 · 200 · 조각 0 의 라우트 수 · result 새 칸의 봉인 · QUEUED 조회의 후보 수 · Claimed 칸 일곱 |

**행렬 밖 파일 다섯** — 넷은 FD 흐름 9절이 이미 적었다 (`result.go` · `rollback.go` · `ask.go` · `seal.go`).
시험 파일은 행렬이 다루지 않는다. 이 계획이 더하는 행렬 밖 파일은 없다.

**시험 파일을 새로 두는 까닭** — 오늘 `internal/api/api_test.go` 가 3,404 줄이다. 이 유닛의 시험을 주제별
새 파일에 두고, 도우미(`newServerFast` · `do` · `advert` · `contractJSON`)는 같은 패키지의 것을 그대로 쓴다.

---

## 3. NFR 두 단계를 건너뛰어서 이 계획이 받는 것

유닛 정의(`unit-of-work.md` 2절)가 NFR 에 적은 것은 둘이었다.

**① 보안 — 종료 보고의 인스턴스 대조** (요구 팩 보안 표의 첫 줄). FD 규칙 1절 10 · 11 (다른 노드 ·
같은 노드의 다른 인스턴스)이 409 로 거절한다. 이 계획은 두 줄을 시험으로 확인한다 (Step 12) — 특히
「같은 노드가 재시작한 뒤 새 인스턴스가 옛 단계의 종료를 보고하면 409」. 조각 2 (보인다)가 사람이 볼
때 이 줄을 다시 확인한다 (finalize 유닛의 조각).

**② 성능 — 대기 중인 Run 을 조회할 때마다의 셈.** 유닛 정의는 「매칭을 두 번 부른다」로 적었고, FD 흐름
4절이 `match.Match` 대신 `contract.Advert.Satisfies` 를 바로 쓰도록 바꿨다. 조회 한 번의 비용은 세 표
(광고 · 임대 · 노드)를 한 번씩 읽는 것과 메모리 안의 (요구 줄 수 x 광고 수) 번 비교다. 대기열 훑기 한 번과
같은 양이다.

- 상한을 새로 두지 않는다 — 대기열 훑기가 이미 같은 일을 제출과 정산마다 하고, 그 자리에도 상한이 없다
- 벤치마크 하나를 둔다 — `BenchmarkCandidatesFor` (`candidates_test.go`). 광고 50 · 요구 줄 3 · 임대 10 ·
  drain 5 로 한 번 돌리고 숫자(ns/op)를 `code-summary.md` 에 적는다 (Step 13). 기본 `go test` 에서는 안 돈다
  (벤치마크는 `-bench` 를 줄 때만 돈다) — 스킵 수에 안 잡힌다
- 목록 `GET /v1/runs` 와 제출 응답에는 셈이 붙지 않는다 (FD 규칙 7절). 셈은 QUEUED 인 Run 하나를 조회할
  때만 일어난다

---

## 4. 이 계획이 정한 것 — FD 에 적히지 않은 자리

**① 409 의 문구를 싣는 오류 모양.** FD 는 `ErrExitRejected` 하나와 경우마다의 문구를 적었다. 문구가
응답의 reason 에 그대로 가야 하므로 store 가 문구를 담은 작은 오류 타입을 돌려준다.

```go
// exitRejection 은 409 의 문구를 나른다.  errors.Is(err, ErrExitRejected) 가 참이다.
type exitRejection struct{ reason string }

func (e exitRejection) Error() string        { return e.reason }
func (e exitRejection) Is(target error) bool { return target == ErrExitRejected }
```

api 는 `errors.Is` 로 409 를 고르고 `err.Error()` 를 reason 에 싣는다. 머리말을 안 붙인다 — 문구가 FD
규칙 1절의 표 그대로 나간다.

**② 어휘 밖 값을 찾는 자리.** FD 규칙 5절은 「어휘 밖의 값은 로그에 경고 한 줄」을 정했고 자리는
`postResult` 다 (FD 흐름 5절). 판단은 `store.StepResult` 의 메서드 하나에 둔다 — 값의 어휘는 contract 의
상수이고 칸은 StepResult 의 것이라 둘을 함께 보는 자리가 여기다.

```go
// OutOfVocabulary 는 어휘 밖 값을 (칸 이름, 값) 쌍으로 돌려준다.  거절 재료가 아니다 — 로그에만 쓴다.
// 보는 칸 다섯: finalize · upload · reason · checkpoint_capture.state · diagnostics.changes
func (r StepResult) OutOfVocabulary() [][2]string
```

빈 값은 어휘 밖이 아니다 (안 적은 것이다). api 는 쌍마다 `result carries an unknown value` 경고 한 줄을
남긴다 — 칸 이름과 값, Run 과 seq 를 속성으로 단다.

**③ 같은 키의 재전송에서 본문이 다를 때의 경고 문구** (FD 규칙 1절 12). 분류가 이미 `steps.exit` 와
`phase_since` 를 읽으므로 store 가 낸다 (`Store.Log` — Mediator 가 기동 때 채운다 · `cmd/mediator/main.go:187`).
문구는 `exit report repeated with a different outcome or time; the first one stands`.

**④ 저장소 오류의 응답 코드.** FD 는 400 · 404 · 409 · 200 만 적었다. DB 오류는 503 `query failed` 로 낸다
(`getRun` 과 같다). finalize 는 5xx 를 재시도하므로 (FD 흐름 10절) 409 로 내면 재시도가 멈춘다.

**⑤ `exited_at` 을 적는 모양.** `phase_since` 는 `timestamptz` 이고 본문의 `exited_at` 을 그대로 넣는다.
Record 의 `exited_at` · `finalized_at` 은 오늘의 `started_at` 과 같은 문자열 모양(UTC · RFC 3339 나노초)이다.

---

## 5. 단계 — 열여섯

### Step 1 — 결과 어휘 `internal/contract/result.go` (FD 엔티티 2절)

- [x] 타입과 상수를 FD 엔티티 2.1 · 2.2 그대로 둔다. JSON 칸 이름과 `omitempty` 도 그대로. 주석은 한국어
      (CONVENTIONS 2.2) · 장식 문자 0
- [x] 파일 머리 주석 — 「이 파일의 모양은 더하기만 한다. 있는 칸의 이름이나 뜻을 바꾸려면 FD 엔티티 2절을
      함께 고친다」 (되물음 3 = A)
- [x] `Exited.Check` — FD 규칙 2절의 검사를 표 순서로. 문구는 표의 영어 그대로. 파싱 오류(JSON · RFC 3339)와
      seq 는 api 가 본다 (Check 는 풀린 값만 본다)
- [x] `result_test.go` — 규칙마다 거절 한 줄과 받는 본문 셋(exit + code · signal 코드 없음 · timeout).
      타입마다 JSON 을 한 번 묶어 FD 의 칸 이름이 나오는지. `Changes` 는 `omitempty` 가 아니므로 빈 값도
      나온다는 것을 확인한다 (FD 엔티티 2.2 그대로)

### Step 2 — 스키마 칸 셋과 phase 상수 (FD 엔티티 1절)

- [x] `schema.sql` — `chosen` 칸 뒤에 `ALTER TABLE steps ADD COLUMN IF NOT EXISTS` 셋. 주석 한 덩어리 —
      무엇이고, 어느 사건이 적고, 종결 전이는 안 건드리며, CHECK 를 안 거는 까닭
- [x] `claim.go` — `PhaseRunning` · `PhaseFinalizing` · `PhaseWaiting`
- [x] 기동이 두 번 돌아도 같은지 — 시험마다 `Migrate` 를 부르므로 기존 시험이 이미 확인한다

### Step 3 — claim 과 재전달이 phase 를 적는다 (FD 규칙 4절 · FD 흐름 2절)

- [x] claim 의 UPDATE (`claim.go:321`) — `phase = CASE WHEN kind = 'merge' THEN 'waiting' ELSE 'running' END,
      phase_since = now(), exit = NULL`. `started_at = now()` 과 한 문장이라 두 시각이 같다
- [x] 재전달의 UPDATE (`claim.go:540`) — 같은 세 칸
- [x] result 문장(`claim.go:783`)과 종결 경로(`FailRestarted` · 임대 만료 · 취소 · 계획 거절)는 안 고친다

### Step 4 — 되돌림 셋이 칸을 지운다 (FD 규칙 4절)

- [x] `rollback.go:76` (loop 구간) · `rollback.go:130` (검증 실패) · `ask.go:660` (계획 거절의 되감기) —
      각 문장에 `phase = NULL, phase_since = NULL, exit = NULL`

### Step 5 — 종료 보고의 수락 `MarkExited` (FD 규칙 1절 · FD 흐름 1절)

- [x] `ErrNoSuchStep` · `ErrExitRejected` · `exitRejection` (4절 ①)
- [x] 수락 — FD 흐름 1절의 UPDATE 한 문장. 1 행이면 `true`. 트랜잭션을 안 연다 · 알림 없음
- [x] 0 행이면 분류 — `runs.state` · `steps.state` · `kind` · `attempt` · `node_id` · `claimed_instance` ·
      `phase` · `exit` · `phase_since` 를 한 번 읽고 FD 규칙 1절 1 ~ 12 의 처음 맞는 줄. Run 이 없으면
      `ErrNotFound` · 단계가 없으면 `ErrNoSuchStep` · 409 줄은 문구를 담은 `exitRejection` ·
      200 줄은 `false, nil`
- [x] 12 에서 본문의 outcome 이나 exited_at 이 처음 값과 다르면 경고 한 줄 (4절 ③)
- [x] godoc 첫 줄 — 「종료 보고의 수락이다. 판정이 아니다 (결정 1-7)」

### Step 6 — 라우트와 핸들러 `postExited` (FD 흐름 1 · 7절)

- [x] `api.go` 등록 — `mux.HandleFunc("POST /v1/runs/{run}/steps/{seq}/exited", s.auth(s.postExited))`.
      result 줄 바로 뒤. `mux.HandleFunc` 18 -> 19
- [x] 순서 — seq 파싱(`invalid step sequence: %s`) -> 본문 파싱(`cannot parse exited: %v`) -> `Check` ->
      `MarkExited`. 응답 — 200 `{run_id, seq, accepted}` · 404 `no such run` · 404 `no such step` ·
      409 문구 그대로 · 503 `query failed` (4절 ④)
- [x] 정산 · drain 경계 · 알림을 부르지 않는다 (FD 규칙 원칙 둘)

### Step 7 — result 의 새 칸 아홉 (FD 엔티티 3절 · FD 규칙 5절)

- [x] `StepResult` 에 칸 아홉 — FD 엔티티 3절의 표 그대로. 모두 `omitempty` · 한 덩어리 주석 (누가 채우나 ·
      판정 재료가 아니다 · 봉인이 다시 풀므로 여기 있어야 Record 에 남는다)
- [x] `OutOfVocabulary` (4절 ②)
- [x] `postResult` — 파싱 뒤, `ReportStep` 앞에서 쌍마다 경고 한 줄. 거절하지 않는다

### Step 8 — 진행 조회의 단계 칸 셋 (FD 엔티티 4.1 · FD 규칙 7절)

- [x] `StepView` 에 `Phase` · `PhaseSince` · `Exit *contract.Outcome`. godoc 에 한 줄 —
      「`phase_since` 는 phase 에 따라 시계가 바뀐다 — running · waiting 은 Mediator, finalizing 은 노드」
      (FD 규칙 3절)
- [x] `Steps` 의 SELECT 에 셋을 더한다. 상태가 CLAIMED 가 아니면 `Phase` · `PhaseSince` 를 비운다.
      `Exit` 는 값이 있으면 상태와 무관하게

### Step 9 — 대기 사유의 후보 수 (FD 엔티티 4.2 · FD 규칙 8절 · FD 흐름 4절)

- [x] `queue.go` — `Candidates` 와 `CandidatesFor`. 읽기 전용 REPEATABLE READ 트랜잭션 하나 —
      `SELECT now()` · `liveAdvertsIn` · `busyIn` · `drainingIn` (새 SQL 없음). 요구 줄마다 배타로 센다 —
      busy 먼저, 그다음 draining. godoc 에 「매처를 부르지 않고 매처가 쓰는 판단 하나를 빌린다 —
      ADR-065 의 관측 경로 규칙은 그대로 참이다」
- [x] `observe.go` — `RequireView.Candidates *Candidates json:"candidates,omitempty"`
- [x] `api.go` — `runView.CandidatesAt *time.Time json:"candidates_at,omitempty"`. `getRun` 에서 Run 이
      QUEUED 이고 계약을 읽었을 때만 부른다. 성공이면 `requires[i].candidates` 와 `candidates_at`, 실패면
      둘을 빼고 로그 한 줄과 warnings 에 `candidate counts could not be read; candidates is omitted`
- [x] 제출 응답(`view`)과 목록에는 안 붙는다

### Step 10 — `Claimed` 의 새 칸 일곱 (FD 엔티티 5절 · FD 규칙 9절)

- [x] `Claimed` 에 칸 일곱 — contract 의 타입 그대로 · 모두 `omitempty`. 한 덩어리 주석 (기본값을 안 채운다 —
      노드가 `EffectOrDefault` · `Budgets` · `MergeWait` 로 채운다)
- [x] `fillFromContract` 의 지역 구조체에 같은 일곱을 더하고 옮긴다. 지역 구조체를 `contract.Step` 으로
      바꾸지 않는다

### Step 11 — Record 의 단계 기록 (FD 엔티티 6절 · FD 규칙 6절 · FD 흐름 5절)

- [x] `record.StepFile` 에 `ExitedAt string` · `FinalizedAt string` · `Exit json.RawMessage` ·
      `LastPhase string`. 모두 `omitempty`. 주석에 칸마다의 시계
- [x] `StepFiles` 의 SELECT 에 `phase` · `phase_since` · `exit`. `exited_at` 은 result 의 값, 없고 exit 가
      있으면 `phase_since` · `finalized_at` 은 result 의 값 · `exit` 는 칸 그대로 · `last_phase` 는 칸 그대로

### Step 12 — 저장소 시험 (`exited_test.go` · `candidates_test.go`)

- [x] 수락 표 열셋 — 줄마다 한 경우. 409 는 문구의 핵심 구절, 200 은 `accepted`, 행이 안 바뀐 줄은
      칸 셋을 다시 읽어 확인한다. 10 (다른 노드)과 11 (같은 노드의 새 인스턴스)을 따로 둔다 (3절 ①)
- [x] 13 의 NULL 줄 — claim 뒤 SQL 로 phase 를 NULL 로 되돌려 이 코드 전의 행을 흉내낸다
- [x] 같은 키의 재전송 둘을 동시에 — 하나만 `true`, 칸에는 처음 값. 다른 본문의 재전송이면 경고 로그 한 줄
      (`Store.Log` 에 기록용 handler)
- [x] phase 의 한 생애 — claim running · exited finalizing · result 뒤 DONE 에도 finalizing 이 남는다 ·
      `examples/bake.json` 으로 세운 Run 에서 build 를 끝낸 뒤 merge 의 claim 이 waiting · 재전달이
      running 과 새 `phase_since` 로 다시 적는다
- [x] 되돌림 셋 — 각 경로 뒤 칸 셋이 NULL
- [x] Record — result 에 `exited_at` 이 있으면 그 값 · 없고 exited 를 받았으면 `phase_since` · 둘 다 없으면
      없음 · `finalized_at` · `exit` · `last_phase`. 옛 노드의 흐름(exited 없음)은 `last_phase: running` 만
- [x] result 새 칸이 봉인에 그대로 남는다 — 아홉 칸을 모두 채운 result 를 보고하고 `StepFiles` 로 다시 읽는다
- [x] `OutOfVocabulary` — 칸 다섯마다 어휘 밖 값 하나 · 어휘 안 값과 빈 값은 안 나온다
- [x] 후보 수 — busy 와 drain 이 겹친 노드는 busy 로만 · 두 요구 줄을 함께 만족하는 노드는 두 줄에 다 ·
      만료된 광고는 live 에 없음 · `candidates_at` 이 DB 시각
- [x] `Claimed` 칸 일곱 — `bake.json` 의 build 와 merge 를 claim 해 칸이 JSON 이름 그대로 실리는지. 안 적은
      계약의 claim 응답에 새 칸 이름이 안 나오는지. 재전달도 같은 칸

### Step 13 — HTTP 시험과 비용 측정 (`internal/api/exited_test.go` · 3절 ②)

- [x] 400 줄마다 하나 · 404 둘 · 409 하나 · 200 의 `accepted` true 와 false · 인증 없으면 401
- [x] `GET /v1/runs/{id}` — CLAIMED 단계에 phase 와 phase_since · DONE 단계에는 없고 exit 만 · QUEUED Run 에
      `candidates` 와 `candidates_at` · RUNNING Run 에는 없음 · 제출 응답에 없음
- [x] result 에 어휘 밖 값을 담아도 200 이고 봉인된 Record 에 그 값이 남는다
- [x] `BenchmarkCandidatesFor` — `candidates_test.go`. 한 번 돌리고 숫자를 `code-summary.md` 에 적는다

### Step 14 — 코드 검사와 조각 0 (`unit-of-work.md` 0절 · CI 와 같은 명령)

- [x] `gofmt -l .` 빈 출력 · `go vet ./...` · `go build ./...`
- [x] `eval "$(scripts/testdb.sh)"` 뒤 `go test ./... -count=1` — 실패 0 · 스킵 0
- [x] `go test ./... -count=1 -coverpkg=./... -coverprofile=...` — 패키지마다 80% 이상. 새 함수의 줄마다 시험이
      닿는지 `go tool cover -func` 로 본다
- [x] 크로스 빌드 셋 — `GOOS=windows GOARCH=amd64` · `GOOS=linux GOARCH=arm GOARM=7` · `GOOS=darwin GOARCH=arm64`
- [x] `enodectl.exe` 의 심볼 상한 (crypto/tls 10 · net/http 50)
- [x] U+2605 0 (`grep -rlIP '\x{2605}'`) · `go run ./scripts/glyphscan.go` (Go 문자열의 장식 문자)
- [x] 코드 경계 시험이 지나는지 (`go test ./...` 안에 있다)
- [x] **조각 0** — 위의 셋(build · vet · test)이 통과하고 `grep -c 'mux.HandleFunc' internal/api/api.go` 가 19.
      ADR-073 (제품 게이트)의 기존 시험이 회귀하지 않았는지는 `go test ./...` 가 확인한다
- [x] `golangci-lint` 경고 수가 `main` 보다 늘지 않는다 (U1 이 병합 전에 밟은 자리 · `8e634bc`)
- [x] `git status` 로 2절 밖의 파일이 없는지. `cmd/enodectl/probe.lock` 이 바뀌면 되돌린다 (Reverse
      Engineering 이 적은 알려진 부채)

### Step 15 — 코드 요약 문서

- [x] `construction/step-phase/code/code-summary.md` — 고친 파일과 새 파일 · 새 이름 · 규칙이 어느 함수에
      있나 · 4절의 결정 다섯 · 코드 검사 결과(숫자) · 벤치마크 숫자 · 뒤 유닛에 넘기는 것 · 정본에 되돌려
      올릴 것 (FD 흐름 11절 · 진행자가 올린다)
- [x] 표기 검사 — `enode-design/scripts/emphasis-check.py` 를 새 문서와 이 계획에 돌린다. 사용자가 싫어한 말투 ·
      사내 이름 검사

### Step 16 — 상태 · 감사 · 커밋

- [x] 이 계획의 체크박스를 채운다 (단계를 끝낸 그 자리에서)
- [x] `aidlc-state.md` 의 U2 절 · `audit.md` 에 완료와 승인 요청을 적는다
- [x] 한 커밋 — 코드 · 시험 · 이 계획 · code-summary · 상태 · 감사 (CONVENTIONS 3.4). 승인 뒤에 넣는다
      (CONVENTIONS 3.3)

---

## 6. 이 단계가 하지 않는 것

```text
   노드가 exited 를 보내는 일 · result 새 칸을 채우는 일     finalize · checkpoint · bake
   result 보고의 인스턴스 대조                            잔여 (FR-2 · 요구 문서 5.3)
   internal/match 변경                                   유닛 정의 그대로 안 바꾼다
   Mediator 화면(internal/api/ui)에 phase 를 그리는 일       유닛 정의에 없다
   공용 Reverse Engineering 문서의 라우트 목록             다음 전면 갱신이 다시 센다.  유닛이 공용 문서를 안 고친다
   정본(enode-design) 되돌림                              진행자가 올린다 (FD 흐름 11절)
   PR 과 병합                                            조각 0 이 초록인 뒤.  밖으로 나가는 일이라 올리기 전에 묻는다
```

---

## 7. 확장 준수 — Code Generation

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표의 첫 줄(종료 보고의 인스턴스 대조)은 FD 규칙 1절 10 · 11 이 닫고 Step 12 가 시험으로 확인한다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 규칙마다 표 시험 한 줄 (Step 12 · 13) |
