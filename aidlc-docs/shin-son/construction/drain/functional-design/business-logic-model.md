# drain — Business Logic Model

유닛 **drain**(W2 · CP3)의 비즈니스 로직이다. 정본은 `ADR-063` §2·§3·§4·§6 ·
`INVARIANTS` §2(`* -> FAILED`) · `mediator-api.md` §3. 값은 `enode-features.md` 3.2.2 ·
`decisions.md` §1. 여기는 **코드가 어느 자리에서 어떤 순서로 무엇을 하는가**만 적는다.
규칙은 `business-rules.md`, 자료 모양은 `domain-entities.md`.

계획과 답 — `construction/plans/drain-functional-design-plan.md` (Q1 = A · Q2 = A · Q3 = A).

---

## 1. 이미 서 있는 것 — 이 유닛은 나머지 셋만 짓는다

```text
   있다   nodes.draining 열 · DrainPolicy 어휘 접기 · UpsertAdvert 의 정책 복사와 이전 값 ·
          advertResponse.drain(언제나) · GET /v1/nodes 의 draining                       obs
   있다   submit 이 if !dry 안에서 DrainingNodes 를 busy 에 합침 (후보 제외 · dry-run 무시) ·
          Cancel 이 안에서 큐를 깨움 · postNodes 가 drain 해제(이전 != "" · 지금 "")에 WakeQueuedNow   queue
   짓는다  ① 노드 — 정책 파일을 읽어 광고에 싣는다
          ② 노드 — 응답의 drain 을 Worker 가 안다 · at-boundary 면 더 집지 않는다
          ③ Mediator — 결과 보고 끝에서 at-boundary 노드의 Run 을 drain:<node_id> 로 취소한다
```

## 2. 흐름 하나 — 정책이 파일에서 중앙까지 (①)

```text
   소유자   <dir>/<stem>.policy.yaml 에 drain: at-boundary 를 쓴다 (손 · 또는 panel 의 토글)
   Advertiser.Run  매 광고 직전
     policy := ReadPolicy(PolicyPath(a.Ident.Config))      파일 하나 읽기.  캐시 없음
     ad.Policy = contract.Policy{Drain: policy.Drain}      "" 이면 omitzero 로 키가 안 나간다
     resp := Advertise(ad)                                 서버가 nodes.draining 에 복사 · 응답 drain 에 되돌림
     a.Held.SetDrain(resp.Drain)                           (Q2 = A · Held 가 잇는다)
     바뀌었으면 Info "drain acknowledged by mediator" mode=<값>
```

`ReadPolicy` 의 갈래 —

```text
   파일 없음                    Policy{} · 로그 없음.  안 걸린 것이 기본이다
   파싱 실패 · 어휘 밖           Policy{} · Warn 한 줄 (값 원문을 적는다 — 소유자의 기계 · 소유자의 파일).
                                같은 원인이 이어지면 다시 안 찍는다 (마지막으로 찍은 원인을 기억)
   유닉스 · 그룹·타인 쓰기 비트    Warn 한 줄 · 값은 그대로 쓴다 (Q3 = A).  윈도우는 안 본다
   읽기 오류 (권한 등)            Policy{} · Warn.  광고는 나간다 — 하트비트를 겸한다
```

## 3. 흐름 둘 — 응답이 Worker 의 태도를 바꾼다 (②)

```text
   Worker.Run  claim 직전
     if w.Held.Drain() == at-boundary
         Info "draining at-boundary; not claiming" (바뀔 때 한 번)
         광고 주기(기본 60초 · Advertiser 가 Mediator 값으로 바꾼 것)만큼 기다렸다 다시 본다
         → 도는 단계가 있으면 그 단계는 이 분기 밖에서 이미 끝까지 간다 (execute 는 claim 뒤다)
     else  오늘 그대로 Claim
```

**graceful 은 Worker 를 안 바꾼다** — 「새 임대만 막는다」이고 임대는 Mediator 의 매칭이
막는다. 도는 Run 의 남은 단계는 이 노드가 계속 집는다 — 그것이 「끝까지 간다」다.

기다리는 길이는 `Advertiser.Every` 와 같은 값이어야 하는데 둘은 다른 고루틴이다 —
`Held` 에 `RenewEvery` 도 함께 둔다(Advertiser 가 응답에서 채운다). 값이 아직 없으면 5초.

## 4. 흐름 셋 — Mediator 가 경계에서 닫는다 (③)

`internal/api/api.go` `postResult` 의 오늘 순서 뒤에 갈래 하나가 붙는다.

```text
   ReportStep (tx 1)         단계 결과 · 효과 · 부분 반납 → 큐 깨우기            오늘 그대로
   rolled == true            응답 rolled_back 을 내기 전에 [drain] 을 먼저 본다   — 되돌린 단계가 다시 돌면 안 된다
   SettleIfDone (tx 2)       마지막 단계면 종료.  state != "" 면 취소할 것이 없다   오늘 그대로
   [drain]  state == ""      drain, err := s.st.NodeDrain(ctx, body.Node)         SELECT draining FROM nodes WHERE node_id
            drain == at-boundary
                              cancelled, err := s.st.Cancel(ctx, runID, "drain:"+body.Node)   (tx 3 · 안에서 큐 깨우기)
                              Info "run closed at a step boundary; the node is draining" run node
                              state = cancelled (FAILED)
   응답                      {run_id, seq, run_state: state}  — 취소됐으면 FAILED 가 실린다
```

```text
   NodeDrain 실패             Error 로그 · 취소 없이 응답 200.  다음 결과 보고가 다시 본다
   Cancel 실패                Error 로그 · 응답 200 (보고는 받았다).  Worker 가 안 집으므로 그 Run 은 이 노드의
                             다음 단계에서 멈춘다 — 다음 결과 보고(다른 노드의 단계)나 소유자의 stop 이 닫는다.  기록
   여러 노드 Run              Run 전체가 FAILED (I5).  다른 노드가 그 사이 집은 단계는 취소로 FAILED 가 된다
   graceful · ""              갈래에 안 든다.  새 코드가 안 돈다
```

**왜 세 tx 인가** — 셋은 이미 각자의 함수와 커밋을 갖고 있고(ReportStep 은 되돌림 판단
때문에 안에서 커밋한다), 취소는 정산 결과를 본 뒤의 판단이다. tx 2 와 tx 3 사이의 창
(claim 이 다음 단계를 집는다)은 **Worker 가 닫는다** — 이 노드는 at-boundary 를 안
순간부터 claim 을 안 건다(3절). 남는 것은 여러 노드 Run 의 다른 노드뿐이고 그 단계는
취소로 닫힌다 — 정본이 Run 전체를 닫으라 했다.

## 5. 해제 · 두 국면

```text
   해제       소유자가 파일의 drain 을 지운다 → 다음 광고에 policy 가 안 실린다 → UpsertAdvert 가
              "" 로 덮고 이전 값을 돌려준다 → postNodes 가 WakeQueuedNow (queue) → 대기 Run 이 RUNNING.
              이 유닛의 새 코드 0.  Worker 는 Held.Drain() 이 "" 로 바뀌는 것을 보고 다시 집는다
   두 국면     「draining · 진행 중」(임대 있음) · 「draining · 대기 중」(임대 0).  GET /v1/nodes 의
              draining 과 lease 로 화면(ui)이 가른다.  이 유닛은 열을 안 더한다
```

## 6. 기존 코드에 닿는 것 — 파일 행렬

```text
   internal/enode/policy.go     신규 — PolicyPath · ReadPolicy · Policy 구조 · 권한 경고
   internal/enode/advertise.go  Advertiser.Held 필드 · 광고 직전 읽기 · AdvertResponse.Drain · Held.SetDrain/SetRenew
   internal/enode/leases.go     Held 에 drain · renewEvery 한 칸씩
   internal/enode/claim.go      Worker.Run 의 claim 직전 분기 (transcript 의 tee 자리와 다른 줄 · 조율)
   internal/api/api.go          postResult 의 [drain] 갈래
   internal/store/queue.go      NodeDrain(ctx, nodeID) (string, error)
   cmd/enode/main.go            Advertiser 리터럴에 Held: held 한 줄 (Q2 = A · panel 의 파일 · 조율)
```
