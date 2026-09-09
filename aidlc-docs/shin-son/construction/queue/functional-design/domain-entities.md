# queue — Domain Entities

새 표는 없다. **어휘 하나(`QUEUED`)와 `Store` 겉면 넷**이 는다. 스키마는 obs 가
`nodes.draining` · `runs.submitter` 를 이미 더했으므로 이 유닛의 `schema.sql` 변경은 0 이다.

---

## 1. `runs` 행 — `QUEUED` 일 때의 모양

```text
   열                QUEUED                       승격 뒤 (RUNNING)            비고
   ────────────────  ───────────────────────────  ──────────────────────────  ──────────────────────
   run_id            제출값                        같음
   state             'QUEUED'                     'RUNNING'                   StateQueued 상수
   principal         헤더의 자기 신고               같음
   submitter         게스트 이름 또는 ""            같음                        obs 의 열.  CreateRun 과 같은 규칙
   contract          제출 전문                     같음                        재계획 판은 contract_versions
   work_id           유도값                        같음
   assigned          NULL                         [{as, nodes:[{node,label}]}] 읽기는 [] 로 접힌다(obs)
   reject            NULL                         NULL                        거절이 아니다
   verdict           NULL                         NULL
   created_at        제출 시각                     같음                        FIFO 의 키
   ended_at          NULL                         NULL
   leases            0 행                          노드마다 1 행                INVARIANTS §1.1
   steps             0 행 (Q2 = A)                 계약의 단계 전부 PENDING     CreateRun 과 같은 규칙
```

취소된 `QUEUED` — `state='FAILED'` · `verdict` cancelled · `ended_at` · 봉인. `Cancel`
그대로다.

## 2. 상태 어휘

```go
// INVARIANTS §1.1 그대로. QUEUED 는 ADR-064 (2026-09-04) 로 올라왔다.
const (
    StateResolving  = "RESOLVING"
    StateAllocating = "ALLOCATING"
    StateQueued     = "QUEUED"      // 새로 든다.  임대 0 · 노드 0 · 재기동 후에도 참
    StateRunning    = "RUNNING"
    StateVerifying  = "VERIFYING"
    StateSucceeded  = "SUCCEEDED"
    StateFailed     = "FAILED"
)
```

CHECK 제약은 안 건다 — `steps.state` · `nodes.draining` 과 같은 이유(어휘가 늘 때
마이그레이션을 강요하지 않는다).

## 3. `Store` 겉면 — 넷 (+ 내부 셋)

유닛 정본 §2 의 겉면에서 **`CreateQueuedRun` 하나가 바뀐다** (Q1 = A). 나머지는 그대로.

```go
// 대기 Run 을 넣고 같은 tx 에서 한 번 훑는다.  promoted 가 참이면 그 자리에서 RUNNING 이 됐다.
func (s *Store) CreateQueuedRun(ctx context.Context, r Run) (promoted bool, err error)

// 부르는 tx 안에서 큐를 훑어 승격한다.  임대가 지워진 직후, 커밋 전에 부른다.
func (s *Store) WakeQueued(ctx context.Context, tx pgx.Tx) ([]string, error)

// 자기 tx 를 열어 훑는다 — 기동 · Reap · drain 해제.
func (s *Store) WakeQueuedNow(ctx context.Context) ([]string, error)

// draining 이 비어 있지 않은 노드.  submit 이 busy 에 합친다 (if !dry 안).
func (s *Store) DrainingNodes(ctx context.Context) (map[string]bool, error)
```

```text
   유닛 정본                                                      이 FD                                   왜
   CreateQueuedRun(ctx, tx, contract.Contract, submitter) error   CreateQueuedRun(ctx, Run) (bool, error)  Q1=A: store 가 tx·잠금·첫 훑기를 진다.
                                                                                                          submitter 는 Run.Submitter(obs).
                                                                                                          bool 은 「바로 승격됨」(rules §3)
   DrainingNodes(ctx, tx)                                         DrainingNodes(ctx)                       submit 은 tx 가 없다.  tx 판은 내부
                                                                                                          drainingIn(ctx, tx) 로 둔다 (busyIn 과 짝)
```

내부(비공개) —

```text
   wakeQueuedIn(ctx, tx) ([]string, error)      훑기·승격의 몸통.  WakeQueued · WakeQueuedNow · CreateQueuedRun 이 공유
   promoteIn(ctx, tx, run, assign, adverts)     UPDATE runs · INSERT leases · INSERT steps.  CreateRun 의 몸통을 뽑아 둘이 쓴다
   drainingIn(ctx, tx) · liveAdvertsIn(ctx, tx) busyIn 과 같은 결의 tx 판.  pool 판과 SQL 을 공유한다 (querier 인터페이스)
   queueLockKey (int64 상수)                    pg_advisory_xact_lock 의 키.  이 저장소에서 유일해야 한다 — 상수 하나에 주석으로 등재
```

`Store` 필드 하나 — `LeaseTTL time.Duration`. 승격이 만드는 임대의 `not_after` 에 쓴다.
`cmd/mediator` 의 `openRecords` 가 `cfg.Lease.TTLSeconds` 로 채운다(`MaxLeasesPerRun`
옆). `nonce()` 는 `internal/api` 에서 `internal/store` 로 옮기고 api 가 그것을 부른다 —
한 벌이다.

## 4. `applyRelease` 의 반환 — 한 칸 넓힌다

```go
// 지금:  func (s *Store) applyRelease(ctx, tx, runID, seq) error
// FD:    func (s *Store) applyRelease(ctx, tx, runID, seq) (freed int, err error)
```

`ReportStep` 이 `freed > 0` 일 때만 `WakeQueued(ctx, tx)` 를 부른다. 놓은 게 없으면
훑지 않는다 — 대기가 있어도 풀린 자원이 없으니 결과가 같다.

## 5. 응답 모양 — 바뀌는 것 없음

`runView` 는 obs 가 넓힌 그대로다. `QUEUED` 응답의 예 —

```jsonc
// POST /v1/runs -> 202
{ "run_id": "demo-42", "state": "QUEUED" }
// GET /v1/runs/demo-42 -> 200   (requires 는 obs · steps 는 승격 전엔 없다)
{ "run_id": "demo-42", "state": "QUEUED",
  "requires": [ { "as": "b", "capability": "agent.reason", "attrs": { "board": "rpi5" } } ] }
// GET /v1/runs?state=QUEUED   (obs 의 목록 · assigned 는 [] 로 접힌다)
```

## 6. 파일 행렬 — 재확인

```text
   internal/store/store.go     W    StateQueued · CreateQueuedRun · WakeQueued · WakeQueuedNow · DrainingNodes ·
                                    promoteIn · liveAdvertsIn · LeaseTTL · nonce
   internal/store/reap.go      W    SettleIfDone · Cancel 의 wake 한 줄 · SettleIfDone QUEUED 보호 · Reap 끝 WakeQueuedNow
   internal/store/claim.go     W    applyRelease 반환 · ReportStep 조건부 wake
   internal/api/api.go         W    submit 자리 둘 + busy 합침 · postNodes drain 해제 wake · nonce 호출 교체
   cmd/mediator/main.go        W    기동 WakeQueuedNow · LeaseTTL 배선
   internal/match/match.go     주석만
   schema.sql                  무변경 (obs 가 열을 더했다)
   테스트  internal/store (queue_test.go 신규) · internal/api (409 -> 202 셋 수정 + CP2 시나리오) · cmd/mediator (기동 wake)
```

행렬 밖 파일은 없다. `internal/store/claim.go` 는 행렬의 「internal/store」 안이고,
transcript(nacl1119)가 만지는 `internal/enode/claim.go` 와는 다른 파일이다.
