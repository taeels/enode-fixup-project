# 컴포넌트 의존 — 행렬과 통신 패턴

AI-DLC Application Design 산출물이다. 어느 컴포넌트가 어느 것을 임포트하고 어떤
수단으로 이야기하는지를 적는다. **임포트 금지 넷을 표에 박는다** — 검사기가 이
표를 읽는다 (`constraints` 구조 불변식).

---

## 1. 의존 행렬 (임포트 방향)

행이 열을 임포트한다. `O` 는 임포트, 빈칸은 안 함, `X` 는 **금지**(검사기가
잰다), `HTTP`·`stdio`·`file` 은 임포트가 아니라 런타임 통신이다.

```text
   임포트하는 쪽 \ 대상   store  api  match  record  contract  runctl  enode  panel  proc  api/ui  mcp
   ────────────────────  ─────  ───  ─────  ──────  ────────  ──────  ─────  ─────  ────  ──────  ───
   cmd/mediator            O     O                                                          O
   internal/api            O          O      O        O                                     
   internal/api/ui                                                                           (정적)
   internal/panel          X     X                             O                     O       
   internal/mcp                                                O                             
   internal/proc                                                                             
   cmd/enodectl                                                              (exec)    O       
   cmd/enode                                                          O       O
   internal/enode                                                                 X          
   internal/runctl        (없음 — 순수 HTTP 클라이언트)
```

**금지 넷 (X)**:

```text
   internal/panel   -> internal/store    화면·제어판은 store 를 직접 안 부른다
   internal/panel   -> internal/api       Mediator 는 HTTP 로만 본다
   internal/api/ui  -> internal/store     화면은 정적 파일이다
   internal/enode   -> internal/panel     데몬은 제어판을 모른다 (반대는 허용)
```

**허용이 중요한 자리**:

```text
   cmd/enode  -> internal/panel   허용.  제어판 서버를 enode 바이너리가 띄운다.
                                  금지는 internal/enode -> panel 한 방향뿐이라
                                  cmd/enode 는 규칙 안 (constraints)
   internal/panel -> internal/runctl   허용.  runctl 은 store·api 가 아니다
   internal/mcp   -> internal/runctl   허용.  같은 이유
   cmd/enodectl · internal/panel -> internal/proc   허용.  proc 는 net/http 를
                                  안 물어 enodectl.exe 심볼 상한이 안 깨진다 (Q1)
```

---

## 2. 심볼 상한과 링크 그래프 (차단 게이트)

`enodectl.exe` 의 `net/http` T 심볼 ≤50 · `crypto/tls` ≤10 이 차단이다
(`ci.yml:485`). 링크 그래프가 이 상한을 정한다.

```text
   cmd/enodectl  ->  internal/proc      net/http 없음.  안전
   cmd/enodectl  =/=> internal/panel    임포트 안 함 (serve 는 exec 위임)
   cmd/enodectl  ->  (serve) exec enode  링크가 아니라 프로세스 실행.  심볼 0

   cmd/enode     ->  internal/panel  ->  net/http    enode.exe 는 상한 없음.  안전
```

`internal/proc`(Q1=A) 가 이 게이트의 핵심이다 — 프로세스 원장을 `net/http` 없는
패키지에 둬서 `cmd/enodectl` 이 임포트해도 상한이 안 깨진다.

---

## 3. 통신 패턴 (임포트가 아닌 런타임)

```text
   수단          경로                                     쓰는 자리
   ───────       ──────────────────────────────────       ────────────────────
   HTTP (out)    internal/panel  ->  Mediator             현재·지난 작업 · cancel
   HTTP (out)    internal/mcp    ->  Mediator             도구 열 개
   HTTP (out)    브라우저 (api/ui) ->  Mediator            GET /v1/nodes · /v1/runs
   HTTP (out)    internal/enode  ->  Mediator             광고 · claim · 결과 (기존)
   stdio         사용자 Claude   <-> internal/mcp          JSON-RPC 2.0
   file tee      internal/enode  ->  링 파일  ->  panel     하네스 출력 (1초 폴링)
   process       internal/proc   ->  데몬 프로세스          alive · stop 신호
   exec          cmd/enodectl serve -> cmd/enode 제어판     위임 (stdio 잇기)
   embed mount   cmd/mediator    ->  internal/api/ui        /ui/ 정적
   out-of-band   웹캠 브로드캐스트 ->  브라우저 임베드         Mediator 밖 (decisions §8)
```

**제어판은 Mediator 를 HTTP 로만 본다** — 임포트가 아니라 통신이다. 그래서
`internal/panel -> internal/store` 금지가 지켜지면서도 관측이 된다.

---

## 4. 데이터 흐름 — 관측 (임계 경로)

```text
   광고 (기존)
     enode  --POST /v1/nodes-->  api.postNodes  -->  store.UpsertAdvert  -->  nodes 표

   현황판 (신규)
     browser --GET /v1/nodes-->  api.getNodes   -->  store.Nodes    -->  스냅숏 JSON
     browser --GET /v1/runs -->  api.getRuns    -->  store.Runs     -->  목록 JSON

   제어판 (신규)
     panel  --runctl.Client.Nodes-->  같은 GET /v1/nodes  -->  자기 행 lease
            --runctl.Client.Status-->  GET /v1/runs/{id}   -->  CLAIMED 단계

   MCP (신규)
     Claude --fleet.list--> mcp --Client.Nodes--> GET /v1/nodes --원문 그대로--> Claude
     (글자까지 같다 · CP5 · Q3)
```

세 표면이 **같은 두 라우트**를 딛는다. 그래서 자료가 안 갈린다.

---

## 5. 데이터 흐름 — QUEUED 와 깨우기

```text
   제출 (신규 202 분기)
     submit --if !dry--> busy += DrainingNodes --> match.Match
        점유 실패 --> store.CreateQueuedRun --> runs(QUEUED) --> 202
        매칭 성공 --> store.CreateRun       --> runs(RUNNING) --> 201

   깨우기 (동기 · 여섯 지점)
     임대 삭제 지점 (postResult 등) --tx--> store.WakeQueued(ctx, tx)
        --> QUEUED 를 FIFO 로 훑어 매칭 --> RUNNING 승격 --> 올린 run_id[]
        --> 같은 tx 커밋 --> status <b> 가 RUNNING 을 봄 (CP2)
```

---

## 6. 데이터 흐름 — drain 과 트랜스크립트

```text
   drain (중앙 라우트 없음 · 광고 경로)
     panel(정책 파일) --> enode(광고에 실음) --POST /v1/nodes-->
     store.UpsertAdvert(nodes.draining) --광고 응답--> Worker
     매칭 제외: submit 이 busy 에 합침
     at-boundary: postResult 끝 --> store.Cancel(drain:<node_id>) --> WakeQueued

   트랜스크립트 (DB 안 만짐)
     enode(하네스) --io.MultiWriter--> 링 파일 --1초 읽기--> panel 카드 (도는 것)
     panel --GET /v1/runs (node_id 필터)--> 목록
           --GET /v1/runs/{id}/record--> tar 에서 logs/NN-*.log (지난 것)
```

---

## 7. 접점 — 유닛 둘 이상이 만진다 (직렬 병합)

의존 그래프와 별개로, 파일 수준에서 여러 유닛이 만지는 자리다. Units Generation
이 파일 행렬로 확정한다 — 이 문서는 미리 정하지 않는다 (`constraints`).

```text
   internal/api/api.go   등록 줄만.  새 핸들러는 새 파일 (nodes.go·runs.go·demo.go)
   internal/store        여러 유닛이 스키마·상태를 만질 수 있다 (QUEUED·submitter·chosen)
   aidlc-docs 상태 파일 둘  진행자가 병합 뒤 (audit.md · aidlc-state.md)
   design/               진행자·shonsin (pen 을 유닛이 안 건드림)
```

---

## 8. 요약 — 의존이 만드는 성질

```text
   한 방향 흐름     cmd -> internal.  internal 끼리는 금지 넷으로 순환을 막는다
   HTTP 격리       panel·mcp 가 Mediator 를 통신으로만 봐서 store 결합이 없다
   심볼 격리       proc 가 net/http 를 안 물어 enodectl.exe 가 깨끗하다 (Q1)
   자료 단일화     세 관측 표면이 같은 두 라우트를 딛어 CP5 가 성립한다
   동기 보장       WakeQueued 가 부르는 tx 안에서 돌아 CP2 가 성립한다
```
