# drain Functional Design 계획 — shin-son

AI-DLC Construction · 유닛 **drain**(W2 · CP3 · 의존 queue)의 Functional Design 계획이다.
**정본은 `ADR-063` §2·§3·§4·§6 · `INVARIANTS` §2(`* -> FAILED` 행) · `mediator-api.md`
§3(광고 본문 `policy` · 응답 `drain`)** 이고, 값은 요구 팩(`enode-features.md` 3.2.2 ·
`decisions.md` §1)이 닫았다. 유닛 정의는 `unit-of-work.md` §3. 이 계획은 값을 다시 적지
않고, **코드가 어느 자리에서 무엇을 하는가**와 계약에 걸리는 미정 셋만 낸다.

브랜치 `unit/drain` (`main` `310c22d` — queue 병합 PR #5 — 에서). 문서 루트 `aidlc-docs/shin-son/`.

---

## 0. 지금 서 있는 자리 — 이미 서 있는 것이 많다

```text
   obs 가 낸 것        nodes.draining 열 · store.DrainPolicy(어휘 셋) · UpsertAdvert 가 정책을
                       복사하고 이전 값을 돌려줌 · advertResponse.Drain(언제나 싣는다) ·
                       contract.Policy{Drain} (광고 본문 policy.drain · omitzero) · GET /v1/nodes 의 draining
   queue 가 낸 것       DrainingNodes — submit 이 if !dry 안에서 busy 에 합친다 (draining 노드 후보 제외) ·
                       Cancel 이 안에서 큐를 깨운다 · postNodes 가 drain 해제(이전 값 != "" · 지금 "")를
                       보고 WakeQueuedNow — 「해제 후 대기 Run 이 RUNNING」이 이미 선다
   남은 것 = 이 유닛     ① 노드가 정책 파일을 읽어 광고에 싣는다 (지금 광고는 policy 를 안 보낸다)
                       ② 응답의 drain 을 Worker 가 안다 · at-boundary 면 더 집지 않는다
                       ③ Mediator 가 postResult 끝에서 at-boundary 노드의 Run 을 취소한다 (drain:<node_id>)
   이 기계              Postgres.app 18.6 · 55434 (queue 와 같다).  CP3 는 enode_cp2 판을 다시 쓴다
```

## 1. 이 단계가 정하는 것 · 안 정하는 것

**정한다.**

```text
   정책 파일의 위치 · 이름 · 형식 · enum · 없을 때 · 틀렸을 때 · 권한     (decisions §1 「모델 몫」)
   광고 경로 — 어느 고루틴이 언제 읽고 어디에 싣나 · 응답이 Worker 로 가는 길
   at-boundary 의 tx 경계 — postResult 안 어디서 취소하나 · 되돌림(rolled) 때 · 마지막 단계 때
   Worker 의 태도 — at-boundary 면 claim 을 안 건다.  graceful 은 그대로
   해제 — 파일에서 지우면 다음 광고가 나른다.  이미 있는 깨우기 지점이 받는다
```

**안 정한다.**

```text
   중앙 라우트 · 소유권 판정 · 자동 복귀 · 지연 상한      정본 · constraints
   제어판의 drain 토글 (파일을 쓰는 쪽)                   panel · nacl1119.  이 FD 가 정한 형식을 쓴다
   현황판의 배지 (C4 · C5)                               ui · runixs.  GET /v1/nodes 의 draining 과 lease 로 가른다
   링 파일 tee (internal/enode/runner.go · claim.go 의 다른 자리)   transcript · nacl1119
```

---

## 2. 산출물 계획 — 생성 완료 (2026-09-09T01:58:09Z · A · A · A)

- [x] `construction/drain/functional-design/business-logic-model.md`
  - [x] 정책 파일 읽기 — 경로 유도 · 파싱 · 접기 · 로그 규칙
  - [x] 광고 경로 — Advertiser 가 매 광고 직전 읽는다 · 응답 drain → Held → Worker
  - [x] Worker — at-boundary 면 claim 대신 기다린다 · 도는 단계는 끝까지 · 보고는 그대로
  - [x] Mediator postResult — 정산 뒤 · Run 이 아직 살아 있고 노드가 at-boundary 면 Cancel(drain:<node_id>)
  - [x] 두 모드 · 해제 · dry-run 의 표
- [x] `construction/drain/functional-design/business-rules.md`
  - [x] 정책 규칙 (정본은 파일 · 어휘 셋 · 기본 graceful · 만료 없음 · 소유자만 푼다)
  - [x] at-boundary 규칙 (단계 도중 안 끊는다 · Run 전체 FAILED · 산출은 Record · verdict 문구)
  - [x] 파일 규칙 (위치 · 형식 · 없음 = "" · 틀림 = "" + 로그 · 권한)
  - [x] 보안 확장 — SECURITY-05(파일 값 검증) · 08(소유 = 파일 쓰기) · 09(권한) · 15 · 나머지 N/A
- [x] `construction/drain/functional-design/domain-entities.md`
  - [x] 정책 파일 스키마 · `contract.Policy` · `AdvertResponse.Drain` · `Held` 의 drain · `Store.NodeDrain`
  - [x] 파일 행렬 재확인 — internal/enode(advertise.go · claim.go · policy.go 신규) · api.go · store 한 함수 · cmd/enode(질문 2)
- [x] 검증 — 표기 규약 · 확장 준수 요약 · 정본과 어긋남 0

---

## 3. 설계 초안 (검토용)

### 3.1 정책 파일

```text
   위치      노드 설정 파일 옆 — 설정이 <dir>/<stem>.yaml 이면 정책은 <dir>/<stem>.policy.yaml   (질문 1)
             설정 경로가 곧 노드 신원(ADR-017)이라 정책도 그 옆에 하나씩 — 한 기계에 노드 여럿이면 각각
   형식      yaml 한 키
               drain: graceful        # 또는 at-boundary.  키가 없거나 파일이 없으면 안 걸린 것
   어휘      "" · graceful · at-boundary  (contract.Policy · store.DrainPolicy 와 같다)
   없음      정책 없음 — 광고에 policy 키가 안 나간다 (omitzero)
   틀림      파싱 실패 · 어휘 밖 → "" 로 접고 노드 로그에 Warn 한 줄 (값을 적는다 — 소유자의 기계 · 소유자의 파일).
             값이 바뀔 때만 찍는다 — 매 광고마다 찍으면 5초에 한 줄이 쌓인다
   권한      (질문 3)
   쓰는 쪽   소유자 손 · 제어판의 토글(panel).  이 유닛은 쓰지 않는다 — CP3 는 손으로 쓴다
```

### 3.2 광고 경로 — 읽는 자리와 나르는 길

```text
   Advertiser.Run  매 광고 직전  policy := ReadPolicy(policyPath(Ident.Config))     Caps 처럼 「읽기만」 · 파일 하나라 싸다
                                 ad.Policy = contract.Policy{Drain: policy}
                   응답          resp.Drain (AdvertResponse 에 필드 추가 — 서버가 언제나 싣는다)
                                 → Held.SetDrain(resp.Drain)                          (질문 2 — 배선)
                                 바뀌면 Info "drain acknowledged by mediator" mode=
   Worker.Run      claim 직전    if Held.Drain() == at-boundary → claim 을 안 걸고 광고 주기만큼 기다린다
                                 graceful · "" → 오늘 그대로
```

**왜 Worker 가 안 집는가** — Mediator 가 경계에서 Run 을 닫으므로 이 노드로 올 단계는
없다. 그래도 안 거는 이유는 둘이다: 취소가 실패한 경우(3.4)에 이 노드가 다음 단계를
집어 「경계에서 놓는다」가 거짓이 되지 않게, 그리고 「더 집지 않는다」가 문서가 아니라
코드에 있게. 응답의 값을 쓰는 이유 — 파일을 썼다고 중앙이 알았다는 뜻은 아니다.
중앙이 받아 적은 값(통보)이 Worker 의 것이다 (enode-features 3.2.2 「데몬은 응답의 값을 Worker 에」).

### 3.3 두 모드 · 해제 · dry-run

```text
   graceful      새 임대만 막는다 — 매칭 제외 (queue 가 이미).  도는 Run 은 끝까지.  Worker 그대로
   at-boundary   + 결과 보고마다 Mediator 가 그 Run 을 취소 (3.4) · Worker 는 더 안 집는다
   해제          파일의 drain 을 지운다/비운다 → 다음 광고 → UpsertAdvert 이전 값 != "" · 지금 "" →
                 WakeQueuedNow (queue 가 이미).  자동 복귀 없음 — 임대가 0 이 돼도 파일이 정본이다
   dry-run       draining 을 안 본다 (queue 가 이미 · ADR-063 §6).  이 유닛은 손대지 않는다
```

### 3.4 Mediator — at-boundary 의 tx 경계

```text
   postResult
     ReportStep (tx 1 · 커밋)          단계 결과 · 효과 · 부분 반납 → 큐 깨우기 (queue)
     SettleIfDone (tx 2 · 커밋)         마지막 단계였으면 종료 · 임대 0 · 큐 깨우기.  state != "" 면 여기서 끝 — 취소할 것이 없다
     [drain]  state == "" (아직 산다)   drain := NodeDrain(body.Node)            한 줄 SELECT
              && drain == at-boundary  Cancel(runID, "drain:"+node)  (tx 3 · 커밋 · 안에서 큐 깨우기)
                                       Info "run cancelled at a step boundary; the node is draining"
     응답 run_state 에 그 결과를 싣는다 — 취소됐으면 FAILED
```

셋을 한 tx 로 안 묶는 이유 — 셋은 이미 각자의 함수와 커밋이 있고(ReportStep 은 되돌림
판단 때문에 안에서 커밋한다), 취소는 정산 결과를 본 뒤의 판단이다. 창은 있다 — tx 2 와
tx 3 사이에 claim 이 다음 단계를 집을 수 있다. **그 창을 Worker 가 닫는다** (3.2 — 이
노드는 at-boundary 를 안 순간부터 claim 을 안 건다). 여러 노드에 걸친 Run 이면 다른
노드가 집을 수 있고, 그 단계는 취소로 FAILED 가 된다 — Run 전체가 닫히는 것이 정본이다.

```text
   rolled == true (되돌림)   그래도 취소한다 — 경계는 경계다.  되돌린 단계가 다시 돌면 「더 집지 않는다」가 깨진다
   Cancel 실패 (DB 오류)      Error 로그 · 응답은 200 (보고는 받았다).  다음 결과 보고나 소유자의 stop 이 잡는다 —
                            Worker 가 안 집으므로 그 Run 은 다음 노드 단계에서 멈춰 있다.  기록해 둔다
   verdict                   Cancel 그대로 — checks[0] = {what:"cancelled", ok:false, note:"cancelled by: drain:<node_id>"}
                            목록의 verdict 로 「drain 으로 닫힘」을 가른다 (decisions §2 GET /v1/runs 행)
```

### 3.5 파일 행렬 — 만지는 것

```text
   internal/enode/policy.go      신규 — 경로 유도 · 읽기 · 접기 · 로그.  transcript(runner.go · claim.go 의 tee)와 다른 파일
   internal/enode/advertise.go   Policy 읽어 싣기 · AdvertResponse.Drain · Held 로 나름
   internal/enode/leases.go      Held 에 drain 한 칸 (SetDrain · Drain)
   internal/enode/claim.go       Worker.Run 의 claim 직전 분기 한 곳 — transcript 가 만지는 자리(runner 의 tee)와 다른 줄.  조율
   internal/api/api.go           postResult 끝의 [drain] 갈래
   internal/store/queue.go       NodeDrain(ctx, nodeID) (string, error) 한 함수 (store 접점이나 파일은 queue 의 것)
   cmd/enode/main.go             질문 2 — Advertiser 에 Held 를 넘기는 한 줄.  행렬상 panel(nacl1119)의 파일
   시험                          internal/enode(정책 읽기 · 광고에 실림 · Worker 가 안 집음 · DB 없음) ·
                                 internal/api(at-boundary 취소 · graceful 무취소 · 해제 후 승격 · DB)
```

---

## 4. 물음 없이 정한 것 — 근거와 함께

```text
   Worker 는 at-boundary 면 claim 을 안 건다   3.2.  graceful 은 그대로 — 「새 임대만 막는다」
   되돌림 뒤에도 취소한다                      3.4.  경계는 경계다
   verdict 는 Cancel 그대로                    decisions §1 이 store.Cancel(run, "drain:<node_id>") 로 못 박았다
   정책 값 틀림은 "" + Warn                     광고는 하트비트를 겸한다 — 값 하나로 노드를 세우지 않는다 (ADR-016 · obs 와 같은 결)
   매 광고마다 읽는다                          파일 하나 · 5초 · 「광고 직전에 읽는다」가 정본.  캐시 없음
   해제 · dry-run · 매칭 제외는 새 코드 0        queue · obs 가 이미 냈다
   중앙은 판정하지 않는다                       파일이 있는 기계에 접속한 사람이 소유자 (ADR-063 §3)
```

---

## 5. 질문 — 계약에 걸리는 셋

`[Answer]:` 뒤에 글자를 적어 달라. 셋 다 **다른 담당의 파일이나 계약**에 닿는다.

## Question 1
정책 파일의 위치와 이름. panel(nacl1119)의 drain 토글이 이 파일을 **쓰고**, 정본이
「enode 설정 옆 · 같은 형식(yaml)」까지만 정했다. 한 기계에 노드가 여럿일 수 있다
(`ENODE_CONFDIR` · `<Name>.yaml`).

A) 설정 파일 옆 · 같은 줄기 — `<dir>/<stem>.yaml` 의 정책은 `<dir>/<stem>.policy.yaml`.
   노드마다 하나 · 설정 경로가 신원이라 정책도 그 옆에 (권장)

B) 설정 디렉터리에 하나 — `ConfDir()/policy.yaml`. 기계의 노드 전부가 같은 정책을 본다

C) 설정 파일 안의 키 — `local.yaml` 에 `policy: {drain: ...}`. 파일 하나. 다만 설정은
   신원의 일부라 제어판이 그 파일을 고쳐 쓰게 된다

D) Other (please describe after [Answer]: A tag below)

[Answer]:

## Question 2
광고 응답의 `drain` 을 Worker 에 넘기는 배선. `Advertiser` 와 `Worker` 는 다른 고루틴이고
둘을 잇는 것은 `Held` 뿐인데, `Advertiser` 는 `Held` 를 모른다 (`OnLeases: held.Set` 만 받는다).
잇는 한 줄은 `cmd/enode/main.go` 에 있고 그 파일은 행렬상 panel(nacl1119)의 것이다.

A) `cmd/enode/main.go` 에 한 줄 — `Advertiser{..., Held: held}`. `Held` 에 drain 한 칸.
   행렬 밖 한 줄이라 진행자와 nacl1119 에 조율로 적는다 (권장 — 정본 「응답의 값을 Worker 에」 그대로)

B) `cmd/enode` 를 안 만진다 — `Advertiser.OnLeases` 의 시그니처를 `func([]Lease, string)` 로
   넓힌다. 그러면 `held.Set` 이 못 들어가 결국 `cmd/enode` 가 바뀐다. 사실상 A 와 같다

C) Worker 가 정책 파일을 직접 읽는다 — 응답이 아니라 파일이 Worker 의 기준. `cmd/enode`
   무변경. 다만 「중앙이 받아 적은 값」이 아니라 「내가 쓴 값」으로 안 집게 되어 정본 문장을 벗어난다

D) Other (please describe after [Answer]: A tag below)

[Answer]:

## Question 3
정책 파일의 권한. `decisions.md` §3 이 「정책 파일 권한」을 보안 확장의 새 표면으로 적었고,
6.3 이 「유닉스 0600 · 윈도우는 디렉터리 ACL」을 링 파일과 같은 자리로 적었다. 다른 사용자가
쓸 수 있는 파일이면 그 사람이 남의 노드를 뺄 수 있다.

A) 읽을 때 권한을 본다 — 유닉스에서 그룹·타인 쓰기 비트가 있으면 Warn 한 줄, 값은 그대로
   쓴다. 윈도우는 안 본다. drain 은 파괴가 아니라 회수라 막지는 않는다 (권장)

B) 그룹·타인 쓰기 비트가 있으면 파일을 무시한다 (""). 안전한 쪽이나, 소유자가 권한을 모르면
   「걸었는데 안 걸린다」가 된다

C) 안 본다. 파일이 그 기계에 있다는 것이 소유 증명이고 권한은 운영자의 것이다

D) Other (please describe after [Answer]: A tag below)

[Answer]:

---

## 6. 다음

답 → FD 산출물 셋 → 승인 · 커밋 → NFR Requirements(짧게 · 새 HTTP 표면 0 · 파일 표면 하나) →
Code Generation 계획 → 코드 · 시험 → CP0 · CP3(단계 둘 이상인 계약 · 정책 파일에 at-boundary ·
`GET /v1/nodes` draining · 경계 뒤 lease 없음 · 새 계약 202 QUEUED · FAILED + verdict drain:<node_id> ·
풀면 대기 Run RUNNING) → PR.
