# queue — 코드 요약

AI-DLC Construction · **queue 유닛**(W1 · CP2)의 Code Generation Part 2 산출물이다.
코드는 저장소 루트에 있고 여기는 **무엇이 어디에 섰고 무엇으로 쟀는가**다. 계획은
`aidlc-docs/shin-son/construction/plans/queue-code-generation-plan.md`.

---

## 1. 만진 파일

```text
   갈래   파일                                  무엇
   ────   ───────────────────────────────────   ───────────────────────────────────────────
   A      internal/store/queue.go                신규 · 대기열 기계 전부 — 잠금 키 · Woken · Notify ·
                                                 DrainingNodes · CreateQueuedRun · WakeQueued ·
                                                 WakeQueuedNow · wakeQueuedIn · promoteIn · labelAssigned
          internal/store/store.go                StateQueued · LeaseTTL · querier · liveAdvertsIn ·
                                                 createRunIn (CreateRun 의 몸통 추출 · 동작 무변경)
          internal/store/reap.go                 SettleIfDone 의 RUNNING 보호 + 깨우기 · Cancel 깨우기 ·
                                                 Reap 이 지운 임대가 있을 때 WakeQueuedNow
          internal/store/claim.go                afterStep 이 부분 반납 뒤 깨운다 · applyStepEffects 반환
          internal/store/release.go              applyRelease 가 지운 임대 수를 돌려준다
          internal/store/schema.sql              runs_queued_idx 부분 인덱스 한 줄
          internal/store/queue_test.go           신규 · 시험 아홉

   B      internal/api/api.go                    submit 의 자리 둘 → enqueue · busy 에 DrainingNodes 합침 ·
                                                 postNodes 의 drain 해제 → WakeQueuedNow
          internal/api/api_test.go               409 를 기대하던 셋 → 202 QUEUED (그 밖의 줄 무변경)
          internal/api/queue_test.go             신규 · 시험 일곱

   C      cmd/mediator/main.go                   wakeQueuedAtStart · LeaseTTL 배선
          cmd/mediator/main_unix_test.go         시험 둘 (이어 붙임)
   —      internal/match/match.go                주석 한 줄 (ADR-064 정정).  코드 무변경
```

**계획에서 벗어난 것 하나** — 큐 기계를 `store.go` 가 아니라 새 파일 `queue.go` 에
뒀다. `store.go` 는 접점이라 diff 가 작을수록 진행자의 직렬 병합이 싸다. 같은
패키지 안이므로 파일 행렬의 「internal/store」 칸은 그대로다.

## 2. 계획이 정한 것이 코드에서 어떻게 섰나

```text
   넣기와 첫 훑기 한 tx      CreateQueuedRun — Begin · pg_advisory_xact_lock · INSERT QUEUED ·
                            wakeQueuedIn · Commit · Notify.  promoted 는 내 run_id 가 승격 목록에
                            있는가다.  api 의 enqueue 가 promoted 면 다시 읽어 201, 아니면 202

   FIFO 전체 훑기            wakeQueuedIn — QUEUED 를 created_at, run_id 순으로 FOR UPDATE SKIP LOCKED.
                            행마다 match.Match · 폭 상한 · promoteIn.  앞 행이 잡은 노드는 뒤 행의 busy

   SKIP LOCKED (계획 밖 결정)  취소는 runs 행을 먼저 잠그고 임대를 지운 뒤 advisory lock 을 기다린다.
                            훑기가 그 행을 FOR UPDATE 로 기다리면 둘이 서로를 기다린다 — 교착.
                            잠긴 QUEUED 행은 지금 취소되는 중이라 건너뛰는 것이 맞고, 그 취소가
                            끝나며 다시 훑는다.  queue.go 의 wakeQueuedIn 주석에 적었다

   승격 = CreateRun 의 몸통   createRunIn 을 뽑아 둘이 쓴다 — 임대 · 단계 · 첫 획득 · 첫 되묻기 ·
                            Record 디렉터리.  제출로 도는 Run 과 기다렸다 도는 Run 이 같은 코드다

   Woken 과 Notify           승격이 올린 첫 되묻기는 부르는 쪽이 커밋한 뒤 Notify 한다 (ADR-032 §4).
                            afterStep 은 자기 raisedAsks 에 합쳐 ReportStep 의 PushAsks 한 번에 탄다

   지점 여섯 + 기동           SettleIfDone · Cancel — 같은 tx 의 DELETE leases 뒤 · Commit 앞
                            Reap — DELETE 의 RowsAffected > 0 일 때만 WakeQueuedNow (주기에 안 얹는다)
                            postNodes 재기동 감지 — SettleIfDone 안 (새 코드 0)
                            afterStep — applyRelease 가 지운 게 있을 때 (획득 뒤 · 되묻기 뒤)
                            postNodes drain 해제 — prevDrain != "" && 지금 "" 이면 WakeQueuedNow
                            기동 — main.go migrate 뒤 wakeQueuedAtStart.  실패는 Error 로그 · 계속

   SettleIfDone 보호         state 가 RUNNING · VERIFYING 이 아니면 "" — QUEUED 의 단계 0 이
                            「전부 끝났다」로 읽히지 않는다 (FD rules §2)

   LeaseTTL                  Store.LeaseTTL — cmd/mediator 의 openRecords 가 cfg.Lease.TTLSeconds 로 채운다.
                            0 이면 한 시간(설정 기본값과 같다).  nonce 는 store 의 것을 쓴다
```

## 3. 재는 것 — 결과 (2026-09-09T00:47:40Z · PostgreSQL 18.6 Postgres.app · 포트 55434)

```text
   go build ./...                       통과
   go vet (store · api · mediator · match)  통과
   gofmt -l                             빈 출력
   go run ./scripts/glyphscan.go        87 파일 · 장식 문자 0
   크로스 빌드 4 (linux/arm64 · linux/amd64 · windows/amd64 · darwin/arm64)   통과
   go test ./internal/store ./internal/api ./cmd/mediator -count=1            전부 통과 (시험 열여덟 포함)
   go test ./... -coverpkg=./... (ci.yml:269 의 awk)   열여섯 패키지 전부 하한 80% 통과 · 전체 87.0%
                                        internal/api 80.6% (408/506) · internal/store 82.1% · cmd/mediator 96.4%
   스킵                                  0
```

**전체 병렬 실행에서 넷이 한 번 깨졌고 단독 재실행에서 전부 통과했다** —
`cmd/enodectl` 의 `TestCmdStart_` 셋과 `cmd/iapadapter` 의
`TestAdapter_HandleNewWorkDoesNotSubmitBeforeTheFleetSeesTheNode`. 넷 다 1초 안에
프로세스나 함대가 서길 기다리는 시험이고 이 유닛이 만지지 않은 패키지다(`main`
워크트리에서도 같은 조건에서 통과). 기록만 한다 — 이 유닛의 것이 아니다.

**첫 실행에서 잡은 것 둘** (같은 커밋에서 고쳤다)
- 시험 픽스처의 단계가 `Agent: {}` 라 `runs.contract` 의 JSON 왕복에서 빠져 종류를
  잃었다 — `Run: ["true"]` 로. 승격이 계약을 DB 에서 다시 읽는다는 사실을 시험이 잡았다.
- 409 를 기대하는 네 번째 시험 `TestRelease_OnceReleasedAnotherCanTake` — 놓기 전엔
  202 로 기다리고, 놓으면 기다리던 것이 같은 요청 안에서 잡는 것으로 고쳤다.
  부분 반납 지점의 깨우기를 재는 시험이 됐다.

**internal/api 가 80.6% 로 하한에 가깝다.** 이 유닛이 더한 문장은 전부 덮였고
여유는 3 문장이다 — 뒤에 이 패키지를 만지는 유닛(demo-back · mcp)이 시험 없이
문장을 더하면 미달이 된다. 진행자에게 넘긴다(5절).

**CP2 는 아직이다** — 실제 노드가 필요하다. 결과는 이 절 아래에 잇는다.

## 4. 시험 — 이름과 재는 것

계획 6절의 열여덟이 그대로 섰다.

```text
   internal/store/queue_test.go
     AllBusyBecomesQueuedWithNoLeaseAndNoStep · CreateQueuedRunPromotesAtOnceWhenTheNodeIsFree ·
     WakeOnSettlePromotesTheOldestFirst (claim → report → settle) · WakeSkipsTheHeadAndPromotesALaterFit ·
     DrainingNodesAreNotCandidates · CancelOfAQueuedRunSealsWithZeroSteps · ReapLeavesQueuedRunsAloneAndWakes ·
     SettleIfDoneIgnoresAQueuedRun · ConcurrentEnqueueAndReleaseNeverStrands (여덟 회차 · 고루틴 둘)
   internal/api/queue_test.go
     SecondRunOnABusyNodeIs202Queued · QueuedRunAppearsInGetRunWithRequires (+재제출 200) ·
     WhenTheFirstRunEndsTheQueuedOneIsRunning (result 응답 직후 · 동기) · CancelReleasesAndWakes (+QUEUED 취소) ·
     DryRunIgnoresBusyAndDraining · UnmatchableIsStill422Failed · DrainReleaseInAdvertWakes
   cmd/mediator/main_unix_test.go
     WakeQueuedAtStart_PromotesWhatWasFreedWhileDown · WakeQueuedAtStart_FailureIsLoggedAndBootContinues
```

## 5. 진행자에게 남기는 것

```text
   ①  접점 — store.go(작은 diff) · reap.go · claim.go · release.go · api.go · main.go · schema.sql 한 줄
   ②  api_test.go 의 셋이 202 를 기대한다.  「기존 라우트 회귀」의 뜻이 「팩이 바꾼 것 빼고」다
   ③  WakeQueued 의 반환이 Woken 이다.  drain(W2)이 at-boundary 취소 뒤에 부를 때 그 모양이다 —
      다만 Cancel 이 이미 안에서 깨우므로 drain 은 Cancel 만 부르면 된다
   ④  internal/api 커버리지 80.6% — 여유 3 문장.  뒤 유닛이 시험 없이 문장을 더하면 미달
```
