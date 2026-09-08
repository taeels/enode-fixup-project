# 서비스 — 정의와 오케스트레이션

AI-DLC Application Design 산출물이다. 컴포넌트가 **무엇인지**는
`components.md`, **어떤 겉면인지**는 `component-methods.md` 가 진다. 여기는
그것들이 **어떻게 엮여 도는지** — 시퀀싱과 오케스트레이션이다. 값은 팩을
가리킨다.

이 저장소는 서비스 계층을 별도 패키지로 두지 않는다 — 오케스트레이션은
`internal/api` 핸들러와 `internal/store` 트랜잭션, 그리고 각 데몬/서버의 진입점이
진다. 아래는 그 흐름을 서비스 단위로 적은 것이다.

---

## 1. 관측 서비스 — 함대와 Run 을 보인다

**책임**: 함대 스냅숏(`GET /v1/nodes`)과 Run 목록(`GET /v1/runs`)을 낸다.
현황판·제어판·MCP 가 전부 이 둘을 딛는다 (임계 경로 · `execution-plan.md`).

```text
   getNodes (api/nodes.go)
     store.Nodes(ctx)  ->  expires_at > now() 인 광고만
                           node_id · label · instance · capabilities ·
                           seen_at · expires_at · lease{run_id,not_after} · draining
     principal 안 냄 · 필터 없음 (ADR-065)

   getRuns (api/runs.go)
     store.Runs(ctx, {state, since, work, limit=100})  ->  created_at 역순
                           run_id · state · verdict · work_id · created_at ·
                           ended_at · assigned · submitter
     페이지네이션 없음 (decisions §2)

   getRun (기존 확장)
     runView 에 requires · as (ADR-069) · steps[].chosen (ADR-060) 추가
```

**같은 자료가 세 표면으로 나간다.** 현황판(브라우저)은 두 라우트를 직접 부르고,
제어판은 `runctl.Client.Nodes`/`Status` 로, MCP 는 `fleet.list`/`runs.list` 로
같은 응답을 낸다 — CP5 가 「글자까지 같다」를 잰다.

---

## 2. QUEUED 서비스 — 죽지 않고 기다린다

**책임**: 점유 실패를 `FAILED` 가 아니라 `QUEUED` 로 받는다 (`3.2.1` · `ADR-064`).

```text
   submit (api.go, if !dry)
     1  store.DrainingNodes 를 busy 에 합친다 (draining 제외)
     2  match.Match(requires, adverts, busy)
     3a 매처 거절 CodeAllBusy (api.go:398~408)   -> CreateQueuedRun -> 202 QUEUED
     3b 매칭됐으나 ErrNodeTaken 롤백 (api.go:457~463) -> CreateQueuedRun -> 202 QUEUED
     3c 매칭 성공                                 -> CreateRun(RUNNING) -> 201
     3d CodeNoCandidate (함대에 없음)             -> CreateRejectedRun(FAILED) -> 422
```

**전이**: `ALLOCATING -> QUEUED` (`INVARIANTS §2`). `QUEUED` 는 임대도 노드도
안 쥔다. 빠져나가는 길은 `cancel` 뿐 — 대기 상한 없음 (`decisions §1`).

**두 자리의 차이**: 3a 는 오늘도 `runs` 행을 남기고(FAILED), 3b 는 트랜잭션이
통째로 롤백돼 **행조차 안 남는다**. 둘 다 `CreateQueuedRun` 으로 QUEUED 행을
남기게 만드는 것이 이 서비스의 일이다. 3b 의 tx 재구성 세부는 FD.

---

## 3. WakeQueued 서비스 — 자원이 풀리면 대기 Run 을 보낸다 (Q4 = A)

**책임**: 임대가 지워지는 모든 지점 뒤에서 대기 Run 을 다시 매칭한다.
**부르는 요청의 트랜잭션 안에서 동기로** 돈다 (`decisions §1` — 비동기면 CP2 가
경쟁이 된다).

여섯 호출 지점:

```text
   지점                              임대가 지워지는 이유
   postResult                        단계 결과 보고 후 종료 정산
   postCancel                        사람이 Run 을 세움
   Reap (만료 회수)                   not_after 지난 임대 회수
   FailRestarted -> SettleIfDone     재기동 감지 (postNodes 안)
   applyStepEffects 부분 반납          단계의 release 역할 부분 해제
   drain 해제를 받은 광고 처리          UpsertAdvert 가 draining 해제를 봄
```

각 지점의 트랜잭션 안에서 `store.WakeQueued(ctx, tx)` 를 부른다. 그것은 QUEUED
를 FIFO 로 훑어 지금 매칭되는 것을 `RUNNING` 으로 올리고 올린 `run_id` 들을
돌려준다. `Reap`·drain 해제는 여러 임대를 한꺼번에 지울 수 있어 **FIFO 전체
훑기**가 필요하다 (Q4 근거).

```text
   크래시 복구   기동 시 cmd/mediator/main.go 가 WakeQueuedNow 를 한 번 부른다
                (자기 tx 를 열어 WakeQueued 를 부른다).  QUEUED 는 재기동 후에도
                참이므로 (ADR-064 §6 · decisions §2)
```

**CP2 동기 보장**: `a` 가 끝나면 `postResult` 트랜잭션 안에서 `WakeQueued` 가
`b` 를 `RUNNING` 으로 올린 뒤 커밋한다. 그래서 커밋 직후 `runctl status <b>` 가
`RUNNING` 을 본다.

---

## 4. drain 서비스 — 소유자가 자원을 돌려받는다

**책임**: 노드 소유자가 새 임대를 막고 도는 것을 안전하게 닫는다 (`3.2.2` ·
`ADR-063`). **중앙 라우트 신설 없음** — 경로는 광고뿐이다.

```text
   정책 왕복 (graceful · at-boundary 공통)
     제어판(정책 파일 쓰기)  ->  데몬이 광고 직전에 읽어 광고에 실음  ->
     UpsertAdvert 가 nodes 열에 복사본 저장  ->  광고 응답이 draining 을 되돌림  ->
     Worker 가 응답의 값을 받음
     매칭 제외: submit 이 draining 노드를 busy 에 합침 (2절).  광고는 계속 보냄

   graceful (기본)
     새 임대만 막는다.  도는 Run 은 자연 종료를 기다린다

   at-boundary
     도는 단계는 끝까지 간다.  결과 보고 후:
     postResult 끝에서 그 노드가 at-boundary 면 store.Cancel(run, "drain:<node_id>")
     ->  Run 전체 FAILED · verdict 가 drain:<node_id> · 끝난 단계 산출은 Record 에 남음
     ->  같은 트랜잭션에서 WakeQueued (3절)

   해제
     소유자가 명시적으로 푼다.  다음 광고에서 후보로 돌아가고 WakeQueued 가
     대기 Run 을 보낸다.  자동 복귀 없음 (decisions §1)
```

정책 파일의 위치·형식·enum 값과 at-boundary 취소의 정확한 트랜잭션 경계는 FD.

---

## 5. 호스트 제어판 서비스

**책임**: 자기 노드 하나를 보고 통제한다 (`3.1.2` · `internal/panel`).

```text
   기동          enodectl serve <name>  ->  enode 제어판 하위명령 exec (위임) ->
                 panel.New(cfg).Handler() 를 127.0.0.1:8081 로 serve
   현재 작업      runctl.Client.Nodes -> 자기 node_id 행의 lease -> Status(run_id)
                 데몬 로그를 안 읽는다 (decisions §2)
   drain 토글     정책 파일 쓰기 (4절의 정책 왕복을 연다)
   프로세스 제어   proc 원장 (processAlive/signalStop/ownsConfig)
                 stop: POST /v1/runs/{id}/cancel 먼저 -> 데몬 정지 (decisions §7.2)
                 재시작 = stop 뒤 start (새 하위명령 아님)
   Mediator 상태  마지막 응답 시각.  끊기면 drain 통보가 늦어진다는 문구
```

---

## 6. 트랜스크립트 서비스 — 도는 것과 지난 것 (`3.1.3`)

**책임**: 하네스가 지금 뱉는 글자와 지난 작업의 봉인 기록을 제어판에 보인다.
**DB 를 안 만진다** (`decisions §6`).

```text
   도는 것 (노드 로컬)
     runner.go/claim.go 가 하네스 stdout/stderr 를 io.MultiWriter 로 링 파일에 tee
     ->  제어판이 링 파일을 1초마다 읽어 카드에 그림 (Mediator 폴링 5초와 별개)
     ->  512 KiB WriteAt · 앞부터 밀림 · 다음 단계 시작 때 비움 (decisions §6.2·§6.3)

   지난 것 (Mediator 가 이미 가짐)
     제어판이 GET /v1/runs 를 받아 assigned[].nodes[].node 가 자기 node_id 인 행만
     화면에서 거른다 (Mediator 필터 신설 0)
     ->  한 줄 누르면 verdict.checks + GET /v1/runs/{id}/record 의 tar 에서
         logs/NN-<step>.log 를 꺼낸다
```

링 파일의 tee 지점·머리/몸통 설계·감김 로직은 FD (`decisions §6.3`).

---

## 7. 되묻기 서비스 — 사람이 병목일 때 보인다 (`3.2.3`)

**책임**: 되묻기 걸린 Run 을 현황판이 보이고, 답은 `runctl`·MCP 로 받는다.
**중앙은 답을 안 받는다** (`decisions §7.2`).

```text
   보이기 (중앙)   GET /v1/asks (기존) 를 현황판이 읽어 카드의 여섯째 상태로 그림.
                  카운트다운이 not_after 가 아니라 ask.deadline (없으면 무한).
                  새 라우트 0 · 답 안 받음
   답하기          runctl asks/answer (기존) · MCP asks.list/run.answer (2절 도구 표)
                  GET /v1/asks 와 답 라우트를 감싼다.  새 의미 0
```

Mediator·DB 무변경. `internal/mcp` 의 도구 둘이 이 서비스에 걸린다.

---

## 8. MCP 서비스 — 사용자 Claude 가 붙는다 (`3.3.1`)

**책임**: stdio JSON-RPC 로 도구 열 개를 낸다. 전부 기존 REST 를 감싼다.

```text
   기동      runctl mcp  ->  mcp.New(runctl.Client{env 토큰}).Serve(ctx, stdin, stdout)
   루프      initialize · tools/list(열 개) · tools/call -> 디스패치 표 -> Client
   글자 일치  fleet.list/runs.list 는 Client.Nodes/Runs 의 원문을 그대로 실음 (Q3)
   토큰      ENODE_MEDIATOR · ENODE_TOKEN 환경변수.  도구 인자로 안 받음
```

---

## 9. 데모 제출 서비스 — 공개 게스트가 고정 시나리오를 낸다 (`3.4` · Q5 = A)

**책임**: 공개·무인증 브라우저가 고정 시나리오만 안전하게 낸다. SECURITY-08
수락 위험을 가둠 셋으로 닫는다 (`decisions §8`).

```text
   postDemo (api/demo.go)
     1  본문의 시나리오 id 가 allow-list 에 있나 — 없으면 거부 (임의 계약 불가)
     2  submitter(표시 라벨)를 받는다 — 인증 아님 (ADR-015)
     3  internal/contract 의 example 픽스처로 고정 계약을 만든다 (decisions §8.3)
     4  서버측 Mediator 토큰을 주입해 내부 submit (2절) — 브라우저에 실 토큰 없음
     5  제출되면 작업 그래프(S7)로 전환 · RUN 카드에 submitter (프론트는 api/ui)

   가둠 셋 (SECURITY-08)
     일회용 데모 Mediator · 서버측 allow-list · 브라우저에 실 토큰 없음
```

시나리오 둘 — ① rpi LED 토글 ② mac wav 를 rpi 스피커 재생 (cross-node 음원은
봉인 Record blob 을 딛는다 · `decisions §8.3`).

---

## 10. 오케스트레이션 지점 요약

```text
   임계 경로        관측 API (getNodes · getRuns).  대부분의 게이트가 딛는다
   동기 지점        WakeQueued 여섯 — 부르는 트랜잭션 안 (CP2)
   접점 (직렬 병합)  api.go 등록 줄 · aidlc-docs 상태 파일 둘 · design/
   통신 수단        HTTP (panel/mcp/browser -> Mediator) · stdio (mcp) ·
                    파일 tee (enode -> 링 -> panel) · 프로세스 신호 (proc)
```

유닛 분해와 병합 순서는 **Units Generation** 이 낸 의존 그래프가 정한다
(`constraints` 구조 불변식). 이 문서는 흐름만 적고 유닛을 안 가른다.
