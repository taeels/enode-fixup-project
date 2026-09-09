# queue — Code Generation 계획

AI-DLC Construction · **queue 유닛**(W1 · CP2)의 Code Generation Part 1 이다. 코드가
아니라 **무엇을 어느 순서로 짓고 무엇으로 재는가**의 계획이다. **Part 2 는 이 문서만
본다** — 여기 없는 것은 짓지 않는다.

정본 — 이 유닛의 Functional Design 셋(`aidlc-docs/shin-son/construction/queue/functional-design/`) ·
NFR Requirements 둘(`.../nfr-requirements/`) · 팩 `requirements/` · `CONVENTIONS.md`.
**어긋나면 `enode-design` 이 이긴다.**

---

## 0. 유닛 맥락

```text
   지는 기능      3.2.1 대기열 (enode-features)
   재는 게이트    CP2 「기다린다」 (+ CP0 회귀)
   의존           obs — 병합됨 (0a159a4).  nodes.draining · Run.Submitter · requires 를 그대로 읽는다
   딛는 유닛      drain(W2 · 같은 담당) — WakeQueued 를 at-boundary 취소 뒤에 부른다
                  demo-back(W2 · runixs) — submit 의 202 경로를 탄다
   브랜치         unit/queue.  접점 셋(store.go · api.go · main.go)은 진행자 직렬 병합
   시험 환경      진짜 Postgres.  NFR 답 3=A — 사용자가 깐다.  DSN 이 오기 전엔 go build · go vet 까지
```

### 0.1 이 계획이 지고 가는 것 — NFR Design 이 SKIP 이다

값은 FD · NFR 이 다 적었다. 여기가 정하는 것은 **어떻게**뿐이다 — 함수 몸통의 순서 ·
파일 안의 자리 · 시험의 이름. 4절이 그 자리다.

---

## 1. 갈래 셋 — 파일이 안 겹친다

```text
   갈래   패키지                 파일                                          의존
   ────   ────────────────────   ───────────────────────────────────────────   ──────────
   A      internal/store         store.go · reap.go · claim.go · schema.sql ·   없음
                                 queue_test.go(신규)
   B      internal/api           api.go · api_test.go(셋 수정) ·                A 의 겉면(3절)
                                 queue_test.go(신규)
   C      cmd/mediator           main.go · main_unix_test.go                    A 의 겉면(3절)
   —      internal/match         match.go 주석 한 줄                            없음
```

**순서는 A → B → C** 다. 한 손이라 병렬이 아니고, B·C 는 A 의 겉면을 부른다.
3절이 그 겉면을 못 박아 A 를 짓는 동안 B·C 의 글자가 안 흔들린다.

---

## 2. 실행 순서 (체크박스)

- [x] 2.1 갈래 A — store (4.A)
- [x] 2.2 갈래 B — api (4.B)
- [x] 2.3 갈래 C — mediator (4.C) · match 주석
- [x] 2.4 `go build ./...` · `go vet ./...` · `gofmt` · glyphscan · emphasis-check
- [ ] 2.5 (미실행 — DSN 없음 · 2026-09-09T00:33:58Z) DSN 이 있으면 `go test ./internal/store ./internal/api ./cmd/mediator` · 커버리지 (6절).
      없으면 「미실행」으로 적고 사용자에게 알린다 — 통과라고 적지 않는다
- [x] 2.6 코드 요약(냄) · 상태 · 감사 갱신.  커밋은 승인 뒤 — 코드 요약 `construction/queue/code/code-summary.md` · `aidlc-state.md` · `audit.md` · 커밋

---

## 3. 못 박는 겉면 — B · C 가 이 글자를 본다

`domain-entities.md` 3절에서 왔고, **둘이 더 넓어졌다** (3.1). 여기서 짐작하지 않는다.

```go
// internal/store — 새로 든다
const StateQueued = "QUEUED"

// Woken 은 한 훑기가 승격한 것과 그때 올라간 되묻기다.
// 되묻기 알림은 커밋 뒤에만 쏜다 (ADR-032 §4) — 부르는 쪽이 커밋한 뒤 Notify 한다.
type Woken struct {
    Runs []string   // 승격된 run_id · 도착순
    asks []AskEvent // 승격이 올린 첫 단계 되묻기.  Notify 가 쏜다
}
func (s *Store) Notify(w Woken)                       // s.PushAsks(w.asks)

func (s *Store) CreateQueuedRun(ctx context.Context, r Run) (promoted bool, err error)
func (s *Store) WakeQueued(ctx context.Context, tx pgx.Tx) (Woken, error)
func (s *Store) WakeQueuedNow(ctx context.Context) ([]string, error)   // 자기 tx · 커밋 · Notify 까지
func (s *Store) DrainingNodes(ctx context.Context) (map[string]bool, error)

// Store 필드 하나
LeaseTTL time.Duration   // 승격이 만드는 임대의 not_after.  cmd/mediator 가 cfg.Lease.TTLSeconds 로 채운다
```

### 3.1 FD 의 겉면에서 넓어진 둘 — 왜

```text
   WakeQueued 가 []string 이 아니라 Woken 이다
      CreateRun 의 몸통에 raiseAsks 가 있다 — 첫 단계가 되묻기인 계약은 만들자마자 묻는다.
      승격이 같은 몸통을 쓰므로(promoteIn) 승격도 묻는다.  그런데 알림은 커밋 뒤에만
      쏘고(ADR-032 §4) 커밋은 부르는 쪽(SettleIfDone · Cancel · ReportStep)이 한다.
      그래서 올라간 질문을 부르는 쪽에 돌려줘야 한다.  []string 만 돌려주면 승격된 Run 의
      첫 되묻기가 영영 안 울린다

   nonce 를 옮기지 않는다
      FD 는 api 의 nonce() 를 store 로 옮긴다고 적었는데, store 에 이미 같은 함수가 있다
      (acquire.go:243 · 획득이 쓴다).  승격은 그것을 부르고 api 의 것은 그대로 둔다 —
      한 벌이 아니라 이미 두 벌이었고, 이 유닛이 세 벌을 안 만든다는 뜻이다.
      두 벌을 하나로 접는 것은 이 유닛의 일이 아니다 (api 는 접점 · 최소 diff)
```

---

## 4. 갈래별 산출

### 4.A internal/store

- [x] A1 `store.go` — `StateQueued` 상수 · `Store.LeaseTTL` 필드 · advisory lock 키 상수와
      「이 저장소의 advisory lock 키」 주석 목록(첫 항목) · `Woken` · `Notify`
- [x] A2 `store.go` — `createRunIn(ctx, tx, r, grants, steps) ([]AskEvent, error)`:
      `CreateRun` 의 tx 몸통(runs INSERT 는 빼고 leases · steps · runAcquires · raiseAsks ·
      Records.Open)을 뽑는다. `CreateRun` 은 runs INSERT 뒤 그것을 부르고 커밋 · PushAsks —
      **동작 무변경**. 기존 시험이 그것을 잰다
- [x] A3 `store.go` — `promoteIn(ctx, tx, runID, c contract.Contract, assign, adverts)`:
      `UPDATE runs SET state='RUNNING', assigned=$2 WHERE run_id=$1 AND state='QUEUED'`
      (0행이면 오류 — 잠금 아래서는 안 난다) → grants(`LeaseTTL` · store 의 `nonce()`) →
      `createRunIn`. `label` 은 api 의 것과 같은 규칙 — assigned 의 `{node,label}` 을 광고에서 채운다
- [x] A4 `store.go` — `liveAdvertsIn(ctx, tx)` · `drainingIn(ctx, tx)` · `DrainingNodes(ctx)`:
      `LiveAdverts` 와 SQL 을 공유한다(`querier` 인터페이스 — `pool` 과 `pgx.Tx` 가 둘 다 만족).
      draining 은 `SELECT node_id FROM nodes WHERE draining <> '' AND expires_at > now()`
- [x] A5 `store.go` — `wakeQueuedIn(ctx, tx) (Woken, error)`: 잠금 → QUEUED 행 FOR UPDATE
      (created_at, run_id) → 광고 · busy(`busyIn` ∪ `drainingIn`) → 행마다 `match.Match` →
      폭 상한(`MaxLeasesPerRun`) 검사 → `promoteIn` → busy 갱신. 0행이면 잠금 뒤 바로 반환
- [x] A6 `store.go` — `WakeQueued`(감싸기) · `WakeQueuedNow`(Begin · wake · Commit · Notify ·
      Runs 반환) · `CreateQueuedRun`(Begin · 잠금 · runs INSERT `state='QUEUED'` · wake ·
      Commit · Notify · `promoted = 내 run_id ∈ Runs`). unique 위반은 그대로 오류(호출자가 200 갈래)
- [x] A7 `reap.go` — `SettleIfDone` 첫 줄에 보호: `runs.state = 'RUNNING'` 이 아니면 `""` 반환
      (FD rules §2). `SettleIfDone` · `Cancel` 의 `DELETE FROM leases` 뒤 · Commit 앞에
      `WakeQueued` — Woken 을 받아 Commit 뒤 `Notify`. `Reap` 의 leases DELETE 뒤에
      `WakeQueuedNow` (오류는 로그 · 회수 수는 그대로 반환)
- [x] A8 `claim.go` — `applyRelease` 반환을 `(freed int, err error)` 로. `applyStepEffects` 가
      그것을 올리고, `afterStep` 이 `freed > 0` 이면 `WakeQueued` — Woken 의 asks 를 자기
      raisedAsks 에 합쳐 `ReportStep` 의 커밋 뒤 `PushAsks` 한 번에 탄다
- [x] A9 `schema.sql` — `CREATE INDEX IF NOT EXISTS runs_queued_idx ON runs (created_at)
      WHERE state = 'QUEUED';` 주석에 왜(빈 큐의 훑기가 매 종료마다 돈다)
- [x] A10 `queue_test.go`(신규 · `scratchDB` 재사용) — 시험 아홉 (6절)

### 4.B internal/api

- [x] B1 `api.go` `submit` — `if !dry` 안에서 `busy` 에 `DrainingNodes` 합침(api.go:477~483).
      매처 거절 갈래: `rej.Code == match.CodeAllBusy && !dry` → `CreateQueuedRun` →
      `promoted` 면 `GetRun` 다시 읽어 201, 아니면 202 + `view` (+warnings). 오류는 기존
      `create failed` 503 갈래와 같은 모양(같은 run_id 경쟁은 200). `ErrNodeTaken` 갈래도
      같은 함수 — 409 `fail` 이 사라진다
- [x] B2 `api.go` `postNodes` — `UpsertAdvert` 의 `prevDrain` 을 받아 `prev != "" &&
      store.DrainPolicy(a.Policy.Drain) == ""` 이면 `WakeQueuedNow` (오류는 로그 · 응답은 그대로).
      자리는 재기동 판정 뒤 · `RenewLeases` 앞
- [x] B3 `api_test.go` — 409 를 기대하던 셋을 202 + QUEUED 로 (FD logic §3.7):
      `TestSubmitRejectCodes` · 경쟁 시험(진 쪽 202 · 「세 번째 요구」 202) · 회수 뒤
      「409 -> 201」 주석과 단언. **그 밖의 줄은 안 만진다** — 3,400줄이고 모두가 딛는다
- [x] B4 `queue_test.go`(신규 · `newServerFast`) — 시험 일곱 (6절)

### 4.C cmd/mediator · internal/match

- [x] C1 `main.go` — `openRecords` 에 `st.LeaseTTL = time.Duration(cfg.Lease.TTLSeconds) * time.Second`.
      `wakeQueuedAtStart(ctx, st, log)` 함수 하나: `WakeQueuedNow` → 승격 있으면 Info ·
      오류면 Error 로그 · **항상 계속**(NFR 답 2=A). 자리는 `migrate` 뒤 · reaper 앞
- [x] C2 `main_unix_test.go` — `wakeQueuedAtStart` 시험 둘(6절)
- [x] C3 `internal/match/match.go` — 주석 「큐가 없으므로(ADR-002 가 QUEUED 를 기각)」를
      ADR-064 정정으로. 코드 무변경

---

## 5. 짓지 않는 것

```text
   READY · 우선순위 · 상한 · 정책 틀        정본 · constraints §2
   광고마다 훑기                            지점 여섯 + 기동 (NFR 2.2)
   api 의 nonce 를 옮기는 것                 3.1
   internal/api/ui · runs.go · nodes.go    obs · ui 의 파일.  안 연다
   cmd/runctl                               2xx 성공.  무변경
   drain 정책 파일 · at-boundary 취소        W2
   .coverage-contract.yml                   안 고친다 (obs 와 같다)
   lock_timeout · statement_timeout         NFR tech-stack 4절
```

---

## 6. 시험 — 이름과 자리

전부 진짜 Postgres. `t.Skip` 을 새로 안 만든다 (store 는 `t.Fatal` · api 는 기존 `newServer` 의
Skip 규약 그대로).

```text
   internal/store/queue_test.go  (scratchDB)
     TestQueue_AllBusyBecomesQueuedWithNoLeaseAndNoStep          QUEUED 행 · leases 0 · steps 0
     TestQueue_CreateQueuedRunPromotesAtOnceWhenTheNodeIsFree    첫 훑기 → promoted=true · RUNNING
     TestQueue_WakeOnSettlePromotesTheOldestFirst                a 정산 → b(먼저 온 것) RUNNING · c QUEUED
     TestQueue_WakeSkipsTheHeadAndPromotesALaterFit              맨 앞이 못 가도 뒤가 간다 (FIFO 전체 훑기)
     TestQueue_DrainingNodesAreNotCandidates                     draining 노드는 승격 후보 아님 · 해제 후 승격
     TestQueue_CancelOfAQueuedRunSealsWithZeroSteps              FAILED · cancelled · 봉인 · 임대 0
     TestQueue_ReapLeavesQueuedRunsAlone                         만료 회수가 QUEUED 를 안 건드림
     TestQueue_SettleIfDoneIgnoresAQueuedRun                     보호 — "" 반환 · 상태 그대로
     TestQueue_ConcurrentEnqueueAndReleaseNeverStrands           고루틴 둘 · 잠금 · 끝에 QUEUED 0 또는 RUNNING

   internal/api/queue_test.go  (newServerFast)
     TestSubmit_SecondRunOnABusyNodeIs202Queued                  CP2 첫 줄
     TestSubmit_QueuedRunAppearsInGetRunWithRequires             GET /v1/runs/{id} state · requires(obs)
     TestSubmit_WhenTheFirstRunEndsTheQueuedOneIsRunning          postResult → 같은 요청 뒤 RUNNING (동기)
     TestSubmit_CancelReleasesAndWakes                           postCancel → 대기 Run RUNNING
     TestSubmit_DryRunIgnoresBusyAndDraining                     dry-run 은 200 · 202 없음
     TestSubmit_UnmatchableIsStill422Failed                      함대에 없으면 422 · FAILED
     TestSubmit_DrainReleaseInAdvertWakes                        prev at-boundary → "" 광고 뒤 승격

   cmd/mediator/main_unix_test.go
     TestWakeQueuedAtStart_PromotesWhatWasFreedWhileDown         DB 에 QUEUED 심고 부르면 RUNNING
     TestWakeQueuedAtStart_FailureIsLoggedAndBootContinues       닫힌 풀 → Error 로그 · 반환은 계속
```

**커버리지** — `internal/store` · `internal/api` · `cmd/mediator` 셋 다 하한 80% 를 잰다
(`go test ./... -coverpkg=./... -coverprofile` + ci.yml:269 의 awk 그대로). 새 문장은
전부 위 시험이 덮는다.

---

## 7. 재는 순서 (CP0 · CP2)

```text
   CP0   eval "$(scripts/testdb.sh)" && go test ./... -count=1
         커버리지 awk · go run ./scripts/glyphscan.go · gofmt -l · go vet ./... · 크로스 빌드
         (심볼 상한 둘은 cmd/enodectl 이 이 패키지들을 안 딛어 안 움직인다 — obs 실측)
   CP2   scene-gates 3절 — Mediator 하나 · 노드 하나 · 계약 셋(runctl example)
         runctl submit a → RUNNING · curl -i POST b → 202 · runctl status b → QUEUED ·
         a 끝난 뒤 status b → RUNNING · submit c(없는 능력) → 422 · status c → FAILED
         화면 항목(QUEUED 행 · chosen)은 ui(runixs)의 몫 — 여기서는 API 로 잰다
```

CP2 는 실제 노드가 필요하다(`enode` 프로세스). 이 기계에서 Mediator 와 enode 를 함께
띄워 잰다 — 시험 DB 와 **다른 판**을 쓴다 (testdb.sh 주석 · 실제로 밟은 사고).

---

## 8. 진행자에게 남기는 표시

```text
   ①  접점 셋을 만진다 — store.go(+reap.go · claim.go) · api.go · main.go.  schema.sql 한 줄.
      직렬 병합의 순서는 W1 셋(queue · mcp · ui) 안에서 진행자가 정한다
   ②  api_test.go 의 셋을 고친다.  「기존 라우트 회귀」의 뜻이 「팩이 바꾼 것 빼고」가 된다
   ③  WakeQueued 의 반환이 Woken 이다 (3.1).  drain(W2)이 부를 때 그 모양이다
   ④  DSN 이 오기 전엔 시험을 못 돌린다.  Part 2 완료 메시지에 「미실행」을 그대로 적는다
```
