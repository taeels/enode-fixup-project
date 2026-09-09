# queue — Business Logic Model

유닛 **queue**(W1 · CP2)의 비즈니스 로직이다. 정본은 `ADR-064` · `INVARIANTS` §1.1·§2 ·
`mediator-api.md` §1·§2. 값은 `enode-features.md` 3.2.1 과 `decisions.md` §1·§2 가
닫았고, 여기는 **코드가 어느 자리에서 어떤 순서로 무엇을 하는가**만 적는다. 규칙은
`business-rules.md`, 자료 모양은 `domain-entities.md`.

계획과 답 — `construction/plans/queue-functional-design-plan.md` (Q1 = A · Q2 = A).

---

## 1. 흐름 하나 — 제출이 대기가 되는 두 자리

`internal/api/api.go` `submit(w, r, dry)` 의 오늘 순서에 분기 둘이 든다. 순서 자체는
안 바뀐다.

```text
   ① 계약 파싱 · Validate · 경고           오늘 그대로
   ② 같은 run_id 재제출 -> 200 기존 Run      오늘 그대로 (QUEUED 도 「기존 Run」이다)
   ③ adverts = LiveAdverts
   ④ if !dry:  busy = BusyNodes ∪ DrainingNodes        <- 합치는 자리 (3절)
   ⑤ assign, rej = match.Match(requires, adverts, busy)
   ⑥ rej != nil:
        dry                     -> 오늘 그대로 (코드만 낸다 · 202 는 못 나온다)
        rej.Code == 422         -> CreateRejectedRun -> FAILED · 422      오늘 그대로
        rej.Code == 409(AllBusy) -> CreateQueuedRun -> 202 view           자리 1
   ⑦ 폭 상한(MaxPerRun) 초과   -> 422 FAILED                              오늘 그대로
   ⑧ dry -> 200 DRY_RUN                                                   오늘 그대로
   ⑨ grants 만들고 CreateRun(RUNNING)
        ErrNodeTaken            -> CreateQueuedRun -> 202 view           자리 2
        그 외 오류              -> 오늘 그대로 (같은 run_id 경쟁은 200 · 아니면 503)
   ⑩ 201 view
```

자리 2 의 `CreateRun` 은 이미 전부 롤백된 뒤다(I5). `runs` 행이 없으므로 이 자리에서
새로 넣는다. 자리 1·2 가 같은 함수를 부르고 같은 응답을 낸다 — 호출자가 둘을
구분할 이유가 없다.

응답 본문은 `s.view(ctx, &run)` 그대로 — `state:"QUEUED"` · `assigned` 비움 · `steps`
비움(Q2 = A) · `warnings` 는 201 과 같이 싣는다.

## 2. `CreateQueuedRun(ctx, run Run) error` — 넣기와 첫 훑기가 한 tx (Q1 = A)

```text
   1  tx = Begin
   2  pg_advisory_xact_lock(queueLockKey)            큐 변경의 직렬화 (business-rules §4)
   3  INSERT INTO runs (run_id, state='QUEUED', principal, contract, work_id, submitter)
        assigned · reject · verdict · ended_at 은 안 넣는다 (SQL NULL)
        같은 run_id 가 이미 있으면 unique 위반 -> 그대로 오류.  submit 의 ⑨ 갈래처럼
        호출자가 GetRun 으로 다시 읽어 200 을 낸다 (경쟁 재제출)
   4  promoted = wakeQueuedIn(ctx, tx)                 방금 넣은 행도 후보다 (3절)
   5  Commit
   6  promoted 에 이 run_id 가 있으면 호출자에게 알린다 (반환값 · 아래)
```

**반환.** `error` 하나로는 「넣었는데 그 자리에서 바로 승격됐다」를 못 낸다. 그 경우
응답은 `201` + `RUNNING` 이어야 한다 — 202 를 내고 나면 껍데기가 QUEUED 를 읽는데
DB 는 RUNNING 이다(거짓은 아니나 응답이 낡는다). 그래서 겉면은
`CreateQueuedRun(ctx, run) (promoted bool, err error)` 로 한 칸 넓힌다. submit 은
`promoted` 면 `GetRun` 으로 다시 읽어 `201` 을, 아니면 `202` 를 낸다.

이 갈래가 실제로 도는 때는 드물다 — ⑤에서 busy 였던 노드가 ③~⑨ 사이에 풀린 경우다.
드물어서 안 만들면 그 경우가 「첫 훑기가 봤는데 응답이 몰랐다」가 된다.

## 3. `wakeQueuedIn(ctx, tx) ([]string, error)` — 훑기와 승격

`WakeQueued(ctx, tx)` 의 몸통이다. 겉면 둘이 이것을 감싼다.

```text
   WakeQueued(ctx, tx)      부르는 쪽의 tx 안에서 (동기)        SettleIfDone · Cancel · ReportStep
   WakeQueuedNow(ctx)       자기 tx 를 열고 부른다               기동 · Reap · postNodes drain 해제
```

```text
   1  pg_advisory_xact_lock(queueLockKey)
   2  rows = SELECT run_id, contract(live) FROM runs
              WHERE state = 'QUEUED' ORDER BY created_at, run_id  FOR UPDATE
      (0 행이면 여기서 끝 — 이것이 빠른 길이고, 대기가 없는 함대에서 비용은 이 한 줄이다)
   3  adverts = liveAdvertsIn(tx)          LiveAdverts 의 tx 판 (같은 SQL · querier 공유)
      busy    = busyIn(tx) ∪ drainingIn(tx)
   4  for each row (도착순):
        assign, rej = match.Match(row.requires, adverts, busy)
        rej != nil          -> continue            (QUEUED 유지.  422 여도 안 닫는다 · rules §2)
        폭 상한 초과          -> continue            (submit 이 이미 422 로 걸렀지만 재계획으로 늘 수 있다)
        promoteIn(tx, row, assign, adverts)         (아래)
        busy ∪= assign 의 노드들                    (다음 행이 같은 노드를 못 잡게)
        promoted = append(promoted, run_id)
   5  return promoted
```

**`promoteIn`** — `CreateRun` 의 tx 몸통을 함수로 뽑아 둘이 같은 코드를 쓴다.

```text
   UPDATE runs SET state='RUNNING', assigned=$assigned WHERE run_id=$1 AND state='QUEUED'
   INSERT INTO leases (node_id, run_id, not_after, nonce)   노드마다 하나 (역할 여럿이어도 임대는 하나)
        unique 위반이면 ErrNodeTaken -> 승격 전체를 tx 오류로 올린다 (rules §4 — 잠금 아래서는 안 난다)
   INSERT INTO steps (...)  CreateRun 과 같은 규칙 — needs 정규화 · node_id 는 역할의 첫 노드
```

`not_after` 와 `nonce` 는 오늘 submit 이 만드는 것과 같은 규칙이다 — TTL 은
`cfg.Lease.TTLSeconds`. store 는 cfg 를 안 보므로 `Store.LeaseTTL` 필드를 하나 두고
`cmd/mediator` 의 `openRecords` 가 채운다(`MaxLeasesPerRun` 과 같은 자리). nonce 는
`api.nonce()` 와 같은 함수를 store 로 옮긴다 — 두 벌이 되면 어긋난다.

## 4. 여섯 지점 + 기동 — 어디서 부르나

| 지점 | 임대를 지우는 코드 | 부르는 방법 | 비고 |
|---|---|---|---|
| postResult | `SettleIfDone` 의 tx (DELETE leases 뒤) | `WakeQueued(ctx, tx)` | CP2 의 「a 끝나면 b RUNNING」이 여기다 |
| postCancel | `Cancel` 의 tx (DELETE leases 뒤) | `WakeQueued(ctx, tx)` | QUEUED 를 취소해도 돈다 — 무해 |
| Reap 만료 회수 | `Reap` (pool.Exec 연쇄 · tx 없음) | DELETE 뒤 `WakeQueuedNow(ctx)` | 감시자라 요청 경쟁 밖 |
| postNodes 재기동 감지 | `FailRestarted` -> `SettleIfDone` | 위 SettleIfDone 안 | 새 코드 0 |
| applyStepEffects 부분 반납 | `applyRelease` (ReportStep 의 tx) | 지운 게 있을 때만 `WakeQueued(ctx, tx)` | `applyRelease` 가 지운 노드 수를 돌려주게 한 칸 넓힌다 |
| drain 해제를 받은 광고 | `UpsertAdvert` 의 이전 값(obs) | `prev != "" && now == ""` 면 `WakeQueuedNow` | postNodes · RenewLeases 앞 |
| 기동 한 번 | 없음 (죽어 있는 동안 풀린 것) | `main.go` migrate 뒤 · reaper 전 `WakeQueuedNow` | 실패해도 기동은 계속 — 로그만 |

`SettleIfDone` 안의 자리는 커밋 앞이다 — 승격이 정산과 같은 tx 에 들어 「a 는 끝났는데
b 는 아직」인 순간이 밖에서 안 보인다. `Cancel` 도 같다.

승격된 run_id 는 호출한 자리에서 `log.Info("promoted from queue", "run", id)` 로 남긴다.
계약 본문은 안 찍는다(SECURITY-03 · obs 가 같은 규칙을 썼다).

## 5. 승격 실패 · 예외 흐름

```text
   후보가 0 (광고 만료 · 유일 노드 drain)     QUEUED 유지.  다음 지점에서 다시 본다
   여전히 전부 busy                         QUEUED 유지
   폭 상한 초과 (재계획으로 requires 가 늘었다)  QUEUED 유지 (submit 의 422 와 다르게 기록은 안 남긴다 — 다음 훑기에 또 본다)
   promoteIn 의 DB 오류                     tx 롤백 -> 이 훑기의 승격 전부 취소.  호출자(정산·취소)도 함께 롤백
                                            — 정산이 실패로 돌아가는 것이 맞다: 임대는 안 지워졌고 다시 온다
   취소 (POST /v1/runs/{id}/cancel)          Cancel 그대로: FAILED · verdict cancelled by · 임대 0 · 봉인.
                                            단계 0 인 Record 가 생긴다 (rules §3)
   Reap                                     QUEUED 는 leases 가 없어 만료 회수 대상이 아니다.  안 건드린다
```

**정산의 롤백이 큐 때문에 나는 것**은 받아들인다 — 큐가 정산을 막는 것이 아니라,
정산 tx 가 통째로 다시 오면 다음 시도가 같은 자리를 다시 훑는다. 갈라서 「정산은
커밋하고 승격만 버린다」로 하면 그 임대 해제를 본 사람이 없어져 3.5 의 창이 열린다.

## 6. 기존 코드에 닿는 것 — 파일 행렬 안

```text
   internal/store/store.go       상수 StateQueued · CreateQueuedRun · WakeQueued · WakeQueuedNow ·
                                 DrainingNodes · promoteIn (CreateRun 몸통 추출) · LeaseTTL · nonce
   internal/store/reap.go        SettleIfDone · Cancel 에 WakeQueued 한 줄씩 · Reap 끝에 WakeQueuedNow ·
                                 SettleIfDone 에 QUEUED 보호 (rules §2)
   internal/store/claim.go       applyRelease 반환 넓힘 · ReportStep 에 조건부 WakeQueued
   internal/api/api.go           submit 의 자리 1·2 와 ④ 합치기 · postNodes 의 drain 해제 wake
   cmd/mediator/main.go          기동 WakeQueuedNow · LeaseTTL 배선
   internal/match/match.go       코드 무변경.  주석의 「ADR-002 가 QUEUED 를 기각」만 ADR-064 정정으로
```

`internal/match` 시그니처 무변경 · `cmd/runctl` 무변경(2xx 성공) · `internal/api/runs.go`
무변경(`state=QUEUED` 필터는 열 값 그대로 탄다).
