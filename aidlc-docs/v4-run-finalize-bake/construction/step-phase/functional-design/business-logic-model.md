# `step-phase` — 흐름

타입은 `domain-entities.md`, 규칙과 문구는 `business-rules.md` 에 있다. 여기는 그 규칙이 어느 자리에서
어느 순서로 도나, 옛 노드 · 옛 Mediator 와 어떻게 함께 사나, 그리고 다른 유닛에 무엇을 넘기나다.

---

## 1. 종료 보고의 흐름

```text
   노드                         api.postExited                      store.MarkExited
   POST .../exited  ---------->  seq 파싱 -> 본문 파싱 -> Check     ->  UPDATE 한 문장 (수락 조건 전부)
                                (400 이면 여기서 끝)                   1 행이면  accepted = true
                                                                     0 행이면  분류 SELECT 한 번
                                                                              -> 200 (받지 않음) · 409 · 404
                    <----------  200 {run_id, seq, accepted}  |  409  |  404
```

**수락은 한 문장이다.** 받을 조건(`business-rules.md` 1절 13)을 전부 `WHERE` 에 둔다.

```sql
UPDATE steps s
   SET phase = 'finalizing', phase_since = $exited_at, exit = $outcome
  FROM runs r
 WHERE s.run_id = $1 AND s.seq = $2 AND r.run_id = s.run_id
   AND r.state = 'RUNNING'
   AND s.state = 'CLAIMED'
   AND s.kind NOT IN ('merge', 'ask', 'acquire')
   AND s.node_id = $node
   AND coalesce(s.claimed_instance, '') = $instance
   AND s.attempt = $attempt
   AND coalesce(s.phase, 'running') = 'running'
```

- 같은 키의 재전송이 동시에 둘 와도 한 문장만 행을 바꾼다 — 행 잠금이 둘째를 기다리게 하고, 둘째는
  `phase = 'finalizing'` 을 보고 0 행이 된다. 처음 값이 남는다
- 0 행이면 **분류만** 한다. `runs.state` · `steps.state` · `kind` · `attempt` · `node_id` ·
  `claimed_instance` · `phase` · `exit` · `phase_since` 를 한 번 읽어 1절 표의 1 ~ 12 중 처음 맞는 줄을
  고른다. 분류는 행을 바꾸지 않으므로 그 사이에 상태가 바뀌어도 해가 없다 — 응답 코드만 그 순간의 것이다
- 12 (같은 키 · 이미 finalizing)에서 본문이 처음 값과 다르면 경고 로그 한 줄
- **트랜잭션을 안 연다.** 판정 · 다음 단계 · 정산 · 임대를 안 건드리므로 묶을 것이 없다
  (결정 1-7 · 1-8). 알림(`PushAsks`)도 없다

`store` 가 돌려주는 것 — 받았나(`bool`)와 오류 셋 중 하나. api 가 응답 코드로 옮긴다.

```go
var (
	ErrNoSuchStep    = errors.New("no such step")            // 404.  Run 이 없으면 오늘의 ErrNotFound
	ErrExitRejected  = errors.New("exit report rejected")     // 409.  문구는 business-rules.md 1절
)

// MarkExited 는 종료 보고의 수락이다.  판정이 아니다 (결정 1-7).
func (s *Store) MarkExited(ctx context.Context, runID string, seq int, e contract.Exited) (accepted bool, err error)
```

---

## 2. phase 의 한 생애

```text
   PENDING ──claim──> CLAIMED/running ──exited──> CLAIMED/finalizing ──result──> DONE · FAILED
      ^                  │    (merge 면 waiting)          │                    (phase 는 그대로 남는다)
      │                  │                               │
      └──되돌림 · 되감기──┴───────────────────────────────┘   재시작 · 임대 만료 · 취소 · 계획 거절
         (phase · phase_since · exit 를 지운다)                  -> FAILED (phase 그대로)
```

- claim 문장(`claim.go:321`)에 `phase = CASE WHEN kind = 'merge' THEN 'waiting' ELSE 'running' END,
  phase_since = now(), exit = NULL` 을 더한다. `started_at = now()` 과 같은 문장이라 두 시각이 같다
- 재전달 문장(`claim.go:540` 의 `UPDATE steps SET started_at=now()`)에 같은 세 칸을 더한다
- 되돌림 셋의 문장에 `phase = NULL, phase_since = NULL, exit = NULL` 을 더한다
- result 문장(`claim.go:783`)은 안 고친다 — phase 를 안 건드리는 것이 규칙이다

**옛 노드** — exited 를 안 보내므로 running 에서 result 로 바로 간다. Record 에는 `last_phase: running`
만 남고 `exit` · `exited_at` 이 없다 (result 도 안 싣기 때문이다). 정본의 「보내지 않는 노드에서는
phase 가 running 에 머문다」 그대로다.

---

## 3. 진행 조회의 조립 — `GET /v1/runs/{id}`

```text
   getRun
     GetRun                        오늘 그대로
     view -> Steps                 SELECT 에 phase · phase_since · exit 를 더한다.
                                   CLAIMED 가 아니면 phase · phase_since 를 비워 낸다
     LiveContract -> RequiresOf    오늘 그대로
     Run 이 QUEUED 면
       CandidatesFor(requires)     한 스냅샷에서 셋을 세고 DB 시각을 준다 (4절)
       성공 -> requires[i].candidates · candidates_at
       실패 -> 둘을 빼고 warnings 에 한 줄
```

`view` 는 제출 응답(`postRuns` · 200 · 202)에도 쓰인다. 후보 수는 `getRun` 에서만 붙인다 — 제출 응답에
안 싣는다 (`business-rules.md` 7절).

---

## 4. 후보 수 — `CandidatesFor`

```go
// CandidatesFor 는 요구 줄마다 후보 수 셋과 그 셈의 DB 시각을 준다 (완료 조건 4).
// 배정이 아니다 — match.Match 를 부르지 않는다.
func (s *Store) CandidatesFor(ctx context.Context, reqs []contract.Require) ([]Candidates, time.Time, error)
```

```text
   읽기 전용 트랜잭션 (REPEATABLE READ) 하나
     SELECT now()                         candidates_at
     liveAdvertsIn(tx)                    오늘의 문장 그대로 — 만료 안 된 광고
     busyIn(tx)                           오늘의 문장 그대로 — leases 의 node_id
     drainingIn(tx)                       오늘의 문장 그대로 — nodes.draining <> ''
   요구 줄 r 마다
     for a in adverts: if a.Satisfies(r)
         live++
         if busy[a.NodeID]           busy++
         else if draining[a.NodeID]  draining++
```

- 문장 셋은 대기열 훑기(`queue.go:225` ~ `:233`)가 이미 쓰는 것이다. 새 SQL 이 없다
- REPEATABLE READ 라 세 읽기가 한 시점을 본다. `now()` 는 트랜잭션 시작 시각이라 그 시점과 같다
- 자리는 `internal/store/queue.go` 다 (파일 행렬 「queue.go — 대기 사유의 후보 수」)
- `observe.go:85` 의 「관측 경로에서 매처를 안 부른다 (ADR-065)」는 그대로 참이다. 매처를 부르지 않고
  매처가 쓰는 판단 하나를 빌린다. 이 뜻을 `CandidatesFor` 의 godoc 에 한 줄로 적는다
- 비용 — QUEUED 인 Run 하나를 조회할 때마다 광고 · 임대 · 노드 세 표를 한 번씩 읽는다. 대기열 훑기
  한 번과 같은 양이다. 값(몇 번까지 괜찮나)은 이 유닛의 NFR Requirements 가 본다

---

## 5. result 에서 Record 까지

```text
   노드 POST .../result    본문 = {node} + StepResult (새 칸 아홉 포함)
   api.postResult          파싱 (모양이 틀리면 400 — 오늘 그대로)
                           어휘 밖 값 -> 경고 로그 (거절 안 함)
   store.ReportStep        오늘 그대로 — steps.result 에 JSON 을 적는다.  phase 는 안 건드린다
   ...
   봉인  StepFiles         SELECT 에 phase · phase_since · exit 를 더한다
                           result 를 StepResult 로 푼다 (새 칸이 있으므로 남는다)
                           exited_at    = result.exited_at ?? (exit 가 있으면 phase_since)
                           finalized_at = result.finalized_at
                           exit         = steps.exit
                           last_phase   = steps.phase
```

어휘 밖 값의 경고는 `postResult` 에서 한 번 낸다. 봉인 때는 다시 안 낸다.

---

## 6. `Claimed` 에 계약 칸을 싣기

```text
   ClaimStep · redeliver -> fillFromContract(c, contractJSON)
     지역 구조체 raw.Steps[] 에 칸 일곱을 더한다 (contract 의 타입 그대로)
     c.Effect, c.Budget, c.Discover, c.Sync, c.Builds, c.IR, c.Merge = st....
```

지역 구조체를 `contract.Step` 으로 바꾸지 않는다 — 오늘 그 함수가 필요한 칸만 풀어 쓰는 방식을 따른다.
칸 일곱이 JSON 이름 그대로 옮겨지는지를 시험이 확인한다.

---

## 7. 라우트와 조각 0

```go
// api.go — 등록 한 줄.  mux.HandleFunc 18 -> 19 · internal/api 전체 등록 27 -> 28
mux.HandleFunc("POST /v1/runs/{run}/steps/{seq}/exited", s.auth(s.postExited))
```

- 경로는 정본 표의 것이다 (`mediator-api.md:286`). 정본 절 제목(`/v1/steps/{id}/exited`)은 되돌림 목록에 있다
- **조각 0** (기동이 안 깨졌다) — `go build ./...` · `go vet ./...` · `go test ./...` 가 통과하고
  ADR-073 의 기존 게이트가 회귀하지 않는다. 라우트 수를 세는 시험은 없다 — 수는 요구 문서(FR-2)가 적는다
- 스키마의 새 칸은 기동 때 `schema.sql` 이 붙인다 (`IF NOT EXISTS` 라 여러 번 돌아도 같다)

---

## 8. 함께 사는 법

```text
   옛 노드 + 새 Mediator     exited 를 안 보낸다 -> phase 가 running 에 머문다.  result 의 새 칸이 없다.
                            Record 에 last_phase: running 만 남는다.  아무것도 안 깨진다
   새 노드 + 옛 Mediator     exited 가 404 (라우트 없음).  노드는 404 · 405 를 「지원 안 함」으로 보고
                            그 Run 동안 다시 안 보낸다 (finalize 유닛).  result 의 새 칸은 옛 Mediator 가
                            모르는 칸이라 JSON 파싱이 버리고 봉인에도 없다
   새 노드 + 새 Mediator     전부
```

Mediator 를 먼저 올려도, 노드를 먼저 올려도 안 깨진다 (Application Design 4절 「옛 노드」 줄).

---

## 9. 파일 행렬 밖의 자리

행렬(`unit-of-work-file-matrix.md` 1.1절)은 이 유닛에 `schema.sql` · `claim.go` · `observe.go` · `queue.go` ·
`api.go` · `record.go` 를 두었다. 설계가 더한 자리는 넷이다. Code Generation 계획이 이 표로 적는다.

| 파일 | 왜 |
|---|---|
| `internal/contract/result.go` (새) | 결과 어휘 (되물음 1 = A). contract 는 U1 의 패키지이고 병합됐다 — 새 파일이라 부딪치지 않는다 |
| `internal/store/rollback.go` | 되돌림 둘에서 새 칸 셋을 지운다 (`:76` · `:130`) |
| `internal/store/ask.go` | 되감기 하나에서 새 칸 셋을 지운다 (`:660`) |
| `internal/store/seal.go` | Record 의 단계 기록에 새 칸 넷 (`StepFiles`) |

---

## 10. 다른 유닛에 넘기는 것

```text
   finalize     exited 를 보낸다 — 종료 status 가 정해진 뒤 · 결과 확정 전 · 한 번.
                  200 이면 멈춘다 (accepted 가 false 여도).  409 이면 멈춘다.  404 · 405 면 그 Run 동안
                  다시 안 보낸다.  그 밖(5xx · 끊김)은 결과 확정과 나란히 재시도한다 (services.md)
                result 에 exited_at · finalized_at · finalize · upload · reason · diagnostics 를 채운다.
                  diagnostics 는 contract.Diagnostics 로 옮겨 담는다 (enode.HarvestNote -> CollectNote)
   checkpoint   result 에 checkpoint_capture 를 채운다.  scratch.Capture -> contract.CheckpointCapture
   bake         build 단계의 exited — 마지막 명령이 끝난 뒤 한 번 (받는 쪽은 run 과 같다).
                merge 단계는 exited 를 안 보낸다 (보내도 200 · 받지 않음).
                result 에 build (contract.BuildManifest) · merge (contract.MergeResult) 를 채운다.
                  lower.BuildRecord · lower.Pinned -> contract 타입.  실패한 build 에도 build 를 채울지는
                  bake 가 정한다 (타입은 막지 않는다)
   모두         contract/result.go 의 모양은 더하기만 한다 — 있는 칸을 바꾸려면 이 문서를 고친다
```

---

## 11. 정본에 되돌려 올리는 것

진행자가 `enode-design` 에 올린다.

```text
   mediator-api.md exited 절     제목의 경로를 /v1/runs/{run}/steps/{seq}/exited 로 ·
                                본문에 node · instance · 응답 {accepted} · 거절 409 와 그 경우 ·
                                merge 단계는 받지 않고 200 · 400 의 조건
   mediator-api.md 진행 조회 절   phase · phase_since 는 CLAIMED 일 때만 · phase_since 의 시계가
                                phase 에 따라 바뀐다 · exit 는 상태와 무관 · QUEUED 의 candidates 셋과
                                candidates_at (배타 · free 없음)
   mediator-api.md result 절     새 칸 아홉 (exited_at 외 여덟) 과 그 타입 · 값으로 거절 안 함
   ADR-075 §4 의 봉투            receipt 묶음이 wire 에서는 평평한 칸이 됐다 (exited_at · finalized_at ·
                                checkpoint_capture)
   Record 형식 문서              단계 기록의 exited_at · finalized_at · exit · last_phase 와 칸마다의 시계
```

---

## 12. 확장 준수 — Functional Design

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표의 첫 줄(종료 보고의 인스턴스 대조)은 1절 10 · 11 이 닫는다. 강도는 NFR Requirements 가 본다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 규칙마다 표 시험 한 줄 |

다음 단계는 이 유닛의 NFR Requirements (최소)다 — 종료 보고의 인스턴스 대조와 대기 조회의 비용.
