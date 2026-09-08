# queue Functional Design 계획 — shin-son

AI-DLC Construction · 유닛 **queue**(W1 · CP2 · 의존 obs)의 Functional Design 계획이다.
**정본은 `ADR-064` · `INVARIANTS` §1.1·§2 · `mediator-api.md` §1·§2** 이고, 값은 요구
팩(`enode-features.md` 3.2.1 · `decisions.md` §1·§2·§7)이 이미 닫았다. 유닛 정의는
`aidlc-docs/v1-run-dhseo/inception/application-design/unit-of-work.md` §2. 이 계획은
값을 다시 적지 않고, **코드가 어느 자리에서 무엇을 하는가**와 계약에 걸리는 미정 셋만
낸다.

브랜치 `unit/queue` (`main` `0a159a4` — obs 병합 PR #4 — 위로 옮겼다). 문서 루트 `aidlc-docs/shin-son/`.

---

## 0. 지금 서 있는 자리

```text
   선행 obs (taeels · W0)     main 에 병합됐다 (PR #4 · 0a159a4).  이 브랜치를 그 위로 옮겼다
   nodes.draining 열           obs 가 더했다 — 어휘 셋 "" · graceful · at-boundary 는
                              store.DrainPolicy 가 접는다.  UpsertAdvert 가 이전 drain 값을
                              돌려준다 (해제를 본 사람이 생겼다 — 3.4 의 여섯째 지점)
   runs.submitter · Run.Submitter   obs 가 더했다.  QUEUED 행도 그대로 싣는다
   GET /v1/runs/{id} 의 requires    obs 가 더했다 (ADR-069).  QUEUED 가 무엇을 기다리는지는 이걸로 본다
   로컬 Postgres              없다 (docker · brew 도 없다).  docs/testdb-setup.md 4절이 안내다.
                              Code Generation 전에 깐다 — 시험이 하드 실패한다
   runctl 과 202              손댈 것 없음 — 2xx 를 성공으로 본다 (client.go:70).  ADR-064 §2
   재배정 (2026-09-08)         shin-son 은 queue · drain.  panel · transcript 는 nacl1119
```

## 1. 이 단계가 정하는 것 · 안 정하는 것

**정한다.**

```text
   QUEUED 의 저장 모양          runs 행에 무엇이 있고 무엇이 비어 있나 (임대 0 · 단계 ?)
   두 자리의 ALLOCATING -> QUEUED   매처 거절(CodeAllBusy) · ErrNodeTaken 롤백
   WakeQueued 알고리즘          훑는 순서 · 재매칭 입력 · 승격 tx · 경쟁 창 닫기
   여섯 지점의 tx 경계           어느 tx 안에서 부르나.  Reap 은 tx 가 없다
   submit 의 202 분기 · dry-run   if !dry 안에서 DrainingNodes 를 busy 에 합친다
   기동 wake                   cmd/mediator/main.go 의 자리
   기존 테스트의 변화            409 를 기대하던 셋이 202 가 된다
```

**안 정한다** — 팩이 값으로 닫았거나 다른 유닛 몫이다.

```text
   202 · FIFO · 상한 없음 · 취소만 출구 · READY 없음 · 크래시 복구   팩 · ADR-064
   GET /v1/runs 목록 · GET /v1/runs/{id} 의 requires                 obs
   draining 을 광고에 싣고 nodes 에 복사 · at-boundary 취소            drain (W2)
   우선순위 · 기아 · 정책 틀                                          이월 (constraints §2)
```

---

## 2. 산출물 계획 — 생성 완료 (2026-09-08T17:32:59Z · Q1=A · Q2=A)

`functional-design.md` Step 6 의 셋. 프런트엔드 없음(frontend-components 안 낸다).

- [x] `construction/queue/functional-design/business-logic-model.md`
  - [x] 제출 경로 — 매처 거절 자리와 ErrNodeTaken 자리에서 `QUEUED` 로 가는 흐름
  - [x] `WakeQueued` — 잠금 · 훑기 · 재매칭 · 승격 · 반환
  - [x] 여섯 지점 + 기동의 호출 자리와 tx 경계 표
  - [x] 승격 실패(광고 만료 · 여전히 busy · draining)의 처리
- [x] `construction/queue/functional-design/business-rules.md`
  - [x] 상태 규칙 — `QUEUED` 에서 참인 것 셋(INVARIANTS §1.1) · 출구 둘(승격 · 취소)
  - [x] 응답 규칙 — 202 본문 · 422 유지 · 200 멱등 · dry-run 은 busy 도 draining 도 안 본다
  - [x] 순서 규칙 — FIFO 의 뜻(도착순 · 전체 훑기 · 못 가면 건너뜀)
  - [x] 보안 확장 — SECURITY-05(입력 검증) · 15(fail-safe) 해당 여부 · 나머지 N/A 근거
- [x] `construction/queue/functional-design/domain-entities.md`
  - [x] `runs` 행의 `QUEUED` 모양 (state · assigned · reject · steps · leases)
  - [x] `Store` 겉면 넷 — `CreateQueuedRun` · `WakeQueued` · `WakeQueuedNow` · `DrainingNodes`
  - [x] 파일 행렬 재확인 — 만지는 파일 셋 · 안 만지는 것(`internal/match` · `cmd/runctl`)
- [x] 검증 — 표기 규약(강조는 굵게만 · 코드에 장식 문자 없음) · 확장 준수 요약 · 정본과 어긋남 0

---

## 3. 설계 초안 (검토용)

### 3.1 `QUEUED` 의 저장 모양

```text
   runs.state      'QUEUED'
   runs.assigned   NULL          배정 전이다.  승격 때 채운다
   runs.reject     NULL          거절이 아니다.  reject 는 422 와 회수(410)의 자리
   runs.work_id    유도값         CreateRun 과 같은 규칙
   leases          0 행          INVARIANTS §1.1 · I5
   steps           질문 3        0 행(승격 때) 또는 PENDING 미리 만들기
```

`GET /v1/runs/{id}` 는 오늘의 `view()` 그대로 — `state:"QUEUED"` · `assigned` 비움.
무슨 능력을 기다리는지(`requires`)는 obs 가 `runView` 에 더한다(ADR-069).

### 3.2 두 자리 — `ALLOCATING -> QUEUED`

```text
   자리 1  매처 거절 rej.Code == CodeAllBusy       api.go submit, match.Match 뒤
           422(CodeNoCandidate) 는 그대로 CreateRejectedRun -> FAILED
   자리 2  CreateRun 이 ErrNodeTaken              api.go submit, CreateRun 뒤
           tx 가 전부 롤백됐다(I5).  지금은 runs 행조차 없다
```

둘 다 `CreateQueuedRun` 을 부르고 `202` 로 답한다. 폭 상한(MaxPerRun) 거절은
422 그대로 — 기다려도 안 된다.

### 3.3 `WakeQueued(ctx, tx)` — 부르는 tx 안에서 동기

```text
   1  pg_advisory_xact_lock(<queue 상수>)    큐 변경을 직렬화한다.  아래 3.5 의 창
   2  QUEUED 행을 created_at, run_id 순으로 읽는다  FOR UPDATE
   3  광고 = LiveAdverts(tx) · busy = busyIn(tx) ∪ DrainingNodes(tx)
   4  행마다 match.Match(requires, adverts, busy)
        배정됨   -> 승격: UPDATE runs SET state='RUNNING', assigned=...
                         INSERT leases · INSERT steps (CreateRun 의 몸통을 tx 버전으로 뽑는다)
                         busy 에 그 노드를 더하고 다음 행으로
        거절됨   -> 그대로 QUEUED.  409 든 422 든 상태를 안 바꾼다 (3.4 · 팩 「출구는 cancel 뿐」)
   5  승격한 run_id 열을 돌려준다.  호출자가 로그를 찍는다
```

`match.Match` 시그니처 무변경 — 호출자가 busy 를 합친다(유닛 정본). 승격 실패는
오류가 아니다. 광고가 만료돼 후보가 0 이 된 Run 도 `QUEUED` 로 남는다 — 노드는
돌아올 수 있고, 시간이 보내지 않는다(INVARIANTS §2 개정).

### 3.4 여섯 지점 + 기동 — 호출 자리와 tx 경계

```text
   지점                       임대를 지우는 코드                     tx     WakeQueued 자리
   ─────────────────────────  ──────────────────────────────────  ─────  ─────────────────────────
   postResult                 SettleIfDone  (reap.go DELETE leases)   있음   같은 tx, DELETE 뒤
   postCancel                 Cancel        (reap.go DELETE leases)   있음   같은 tx, DELETE 뒤
   Reap 만료 회수              Reap          (pool.Exec 연쇄)          없음   DELETE 뒤 WakeQueuedNow (자기 tx)
   postNodes 재기동 감지        FailRestarted -> SettleIfDone          있음   SettleIfDone 안 (위와 같은 코드)
   applyStepEffects 부분 반납   applyRelease  (ReportStep 의 tx)       있음   applyRelease 가 지운 게 있을 때, 같은 tx
   drain 해제를 받은 광고 처리   UpsertAdvert 의 이전 drain 값 (obs)      없음   postNodes: prev != "" 이고 지금 "" 이면 WakeQueuedNow
   기동 한 번                  (없음 — 죽어 있는 동안 풀린 것)          —      main.go migrate 뒤 · reaper 전 WakeQueuedNow
```

Reap 은 요청이 아니라 감시자라 CP2 의 경쟁 조건(「a 가 끝난 뒤 status <b> -> RUNNING」)
밖이다 — 자기 tx 로 충분하다. `SettleIfDone` 안에 두면 postResult 와 재기동 감지 둘이
한 자리로 덮인다.

### 3.5 경쟁 창 — 왜 잠금인가

```text
   T1  임대를 지운다  -> QUEUED 를 훑는다   (아직 커밋 안 된 T2 의 행은 안 보인다)
   T2  QUEUED 를 넣는다 -> 임대를 본다      (아직 커밋 안 된 T1 의 삭제는 안 보인다)
   둘 다 못 보면 그 Run 은 다음 임대 해제까지 선다
```

`CreateQueuedRun` 도 같은 advisory xact lock 을 잡고, 넣은 뒤 **같은 tx 에서 한 번
`WakeQueued`** 를 돈다. 잠금이 T1·T2 를 직렬화하므로 둘 중 하나는 반드시 본다.
잠금은 xact 범위라 커밋·롤백에 풀린다 — 누수 없음.

### 3.6 submit 의 분기 · dry-run

```text
   if !dry {
       busy = BusyNodes(ctx)
       busy ∪= DrainingNodes(ctx)      <- 여기 (api.go:477~483 · decisions §2 「dry-run 과 draining」)
   }
   match.Match(...)
   rej != nil && rej.Code == CodeAllBusy && !dry   -> CreateQueuedRun -> 202 view
   rej != nil (그 외)                              -> 오늘 그대로 (422 FAILED · dry 는 코드만)
   CreateRun == ErrNodeTaken                        -> CreateQueuedRun -> 202 view
```

dry-run 은 busy 를 안 보므로 202 가 나올 수 없다(`mediator-api` §2 dry-run 표).

### 3.7 기존 테스트의 변화

`internal/api/api_test.go` 에서 409 를 기대하는 자리 셋 — `TestSubmitRejectCodes`(둘째
Run) · 경쟁 테스트(진 쪽 · 세 번째 요구) · 회수 뒤 「409 -> 201」 주석. 셋 다 `202` +
`QUEUED` 가 되고, 경쟁 테스트는 「이긴 하나 + 나머지 QUEUED」로 바뀐다. `internal/match`
의 주석(「ADR-002 가 QUEUED 를 기각」)은 ADR-064 정정대로 고친다 — 코드는 무변경.

---

## 4. 물음 없이 정한 것 — 근거와 함께

착수 지침(「권장값을 벗어나야 하면 묻지 말고 근거를 적어라」)대로, 팩과 정본에서
유도되는 것은 묻지 않는다. 틀렸으면 아래 질문 파일의 Other 로 적어 달라.

```text
   FIFO 전체 훑기     도착순으로 전부 보고, 맞는 것은 승격하고 못 가는 것은 건너뛴다.
                     맨 앞이 보드를 기다린다고 뒤의 mac 요구까지 세우지 않는다.
                     같은 자원을 두고 겨루면 도착순이다.  기아는 ADR-064 §6 이월 항목
                     (유닛 정본 「FIFO 전체 훑기(Reap·drain 해제 대응)」)
   광고 도착은 지점이 아니다   깨우는 자리는 팩의 여섯 + 기동.  광고마다 훑지 않는다
                     (「주기에 얹지 않고 지점에 건다」).  노드가 재시작 없이 광고만
                     만료됐다 돌아오는 경우는 다음 해제 지점까지 선다 — 기록해 둔다
   취소는 Cancel 그대로   QUEUED 도 * -> FAILED 취소 경로.  verdict 「cancelled by」 ·
                     봉인까지 오늘 코드 그대로(단계 0 인 Record).  왜 안 돌았는지가
                     Record 에 남아야 한다(ADR-005).  Seal 이 단계 0 을 받는지는 코드 단계 검증
   202 본문 = view     201 과 같은 모양 · state QUEUED · assigned 비움 (mediator-api §2)
   reject 는 NULL     QUEUED 는 거절이 아니다.  CodeAllBusy 는 매처 안의 이름으로만 남는다
   승격 실패는 QUEUED 유지   422 가 됐어도 안 닫는다.  출구는 cancel 뿐 (decisions §1)
   Reap 은 QUEUED 를 안 건드린다   만료 회수는 leases 가 있는 Run 만 본다.  확인했다
   draining 열은 obs 의 것      처음 계획의 질문 1 이었다.  obs 가 더해 닫혔다 (2026-09-09)
```

---

## 5. 질문 — 계약에 걸리는 둘

`[Answer]:` 뒤에 글자를 적어 달라. Other 면 그 뒤에 설명을.

## Question 1
유닛 정본의 겉면은 `CreateQueuedRun(ctx, tx, contract.Contract, submitter string) error`
다. 그런데 submit 은 tx 를 안 열고(CreateRun · CreateRejectedRun 이 자기 tx 를 연다),
경쟁 창(3.5)을 닫으려면 넣기와 첫 훑기가 한 tx 에 있어야 한다. 서명을 어떻게 두나?

A) `CreateQueuedRun(ctx, run store.Run) error` — store 가 자기 tx 를 열고 잠금 · INSERT ·
   첫 `WakeQueued` 를 그 안에서 한다. CreateRun · CreateRejectedRun 과 같은 결.
   submitter 는 `Run` 의 필드로 든다(obs 가 열을 더한 뒤) (권장)

B) 정본 그대로 — submit 이 tx 를 열어 넘긴다. api.go 가 tx 를 다루는 첫 자리가 된다

C) Other (please describe after [Answer]: A tag below)

[Answer]:

## Question 2
`QUEUED` 인 Run 의 `steps` 행을 언제 만드나? `GET /v1/runs/{id}` 의 `steps[]` 모양과
claim 안전성에 걸린다.

A) 승격 때 — QUEUED 는 runs 행 하나뿐(임대 0 · 단계 0). 상세의 `steps` 는 비어 있다.
   `view()` 주석 「배정 전이면 단계가 없는 것이 정상」 그대로. 승격이 CreateRun 의 몸통을
   그대로 쓴다 (권장)

B) 제출 때 PENDING 으로 미리 만든다(node_id 비움) — 화면이 계획된 단계를 미리 보인다.
   claim 경로가 임대 없는 단계를 안 집는지 검증이 따른다

C) Other (please describe after [Answer]: A tag below)

[Answer]:

---

## 6. 다음

답 -> 모호하면 후속 질문 파일 -> FD 산출물 셋 생성 -> 완료 메시지(2-옵션) -> 승인 ->
커밋(「단계 승인마다」) -> NFR Requirements 여부 판단(신규 표면 없음 · 보안 확장 취급표가
이미 있으므로 짧게) -> Code Generation 계획(obs 병합 뒤 rebase).
