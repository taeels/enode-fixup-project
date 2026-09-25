# `step-phase` — Functional Design 계획

**유닛** `step-phase` (Mediator 진행 구간) · **브랜치** `unit/step-phase` · **담당** taeels ·
**회차** `v4-run-finalize-bake` (굽기) · **순서** 여덟 중 둘째 · **선행** contract-grammar (`main`
`9b40cd0` 에 병합) · **맡는 조각** 0 (기동이 안 깨졌다) · **완료 조건** 4 · **스토리** US-7

입력은 유닛 정의(`unit-of-work.md` 2절) · 요구(`requirements.md` FR-2) · 설계
(`application-design/components.md` 3.3 ~ 3.5 · `services.md` 6절 · `component-methods.md` 7 · 8절) ·
실행 계획 7절 ② (두 시계) · 정본 `enode-design/protocol/mediator-api.md` 의 exited 절(:446)과
진행 조회 절(:208) · 앞 유닛이 넘긴 일(`construction/contract-grammar/functional-design/business-logic-model.md`
8절)이다. 이 계획은 그 위에서 **Mediator 가 단계의 진행 구간을 어떻게 적고 보이나**만 짓는다.
코드는 다음 단계다.

---

## 0. 이 단계가 닫는 것과 안 닫는 것

```text
   닫는다     종료 보고(POST .../exited)의 수락 규칙 — 무엇이 맞으면 받고, 무엇이 다르면
             거절하고, 무엇은 조용히 200 인가.  본문의 모양과 검사
             phase 의 전이 — 어느 사건이 어느 값으로 바꾸고, 되돌림과 종결 뒤에 무엇이 남나
             두 시계 — 진행 조회와 Record 의 칸마다 어느 시계인가 (실행 계획 7절 ②)
             result 에 더하는 칸의 모양과, Mediator 가 그 값을 어떻게 다루나
             QUEUED 사유 — 요구 줄마다 후보 수 셋의 정의와 계산
             Claimed 에 계약의 새 칸 일곱을 싣는 모양 (contract-grammar 가 넘긴 일)

   안 닫는다   노드가 exited 를 언제 · 어떻게 보내고 재시도하나       finalize 유닛
             예산을 노드가 지키고 reason 을 채우는 일               finalize 유닛
             result 의 진단 · 보존 · build · merge 칸의 안쪽 모양    finalize · checkpoint · bake 유닛
             merge 단계가 합치기를 어떻게 알리나                   bake 유닛
             result 보고의 인스턴스 대조                           잔여 (FR-2 · 5.3)
             internal/match 의 변경                               하지 않는다 (유닛 정의)
             보안과 성능의 값 (인스턴스 대조의 강도 · 조회 비용)     이 유닛의 NFR Requirements
```

---

## 1. 실측 — 코드가 지금 어떻게 생겼나 (2026-09-25 · `9b40cd0`)

### 1.1 단계의 행과 시각

- `steps` 표는 `state` (PENDING · CLAIMED · DONE · FAILED · SKIPPED) 와 `started_at` · `ended_at` 를
  가진다 (`internal/store/schema.sql:71`). CHECK 제약을 일부러 안 건다 — 어휘가 늘 때
  마이그레이션을 강요하지 않으려고. 새 칸도 같은 방식이면 된다 (`ALTER TABLE … ADD COLUMN IF NOT EXISTS`)
- `started_at` 은 claim 때 Mediator 의 `now()` (`claim.go:321`), `ended_at` 은 result 를 받을 때
  Mediator 의 `now()` (`claim.go:783`). 둘 다 **Mediator 시계**다
- 재전달(같은 생이 다시 물음 · `claim.go:510`)은 `started_at` 을 `now()` 로 다시 적는다
- 단계를 PENDING 으로 되돌리는 자리가 **셋** 있다 — loop 과 validate_with 의 되돌림
  (`rollback.go:76` · `:130`)과 계획 거절의 되감기(`ask.go:660`). 셋 다 `result` · `started_at` ·
  `ended_at` 을 지운다. **파일 행렬에 없던 자리다** — 새 칸도 여기서 지워야 한다

### 1.2 claim 과 result

- claim 은 `node_id` 가 맞고 PENDING 이고 needs 가 끝난 단계를 집어 CLAIMED 로 바꾸고
  `claimed_instance` 를 적는다 (`claim.go:321`). ask · acquire 는 노드가 안 집는다
- result 는 본문의 `node` 로만 대조한다 — `node_id` 가 같고 CLAIMED 여야 받는다. 아니면 409
  「reported a step that was not claimed」 (`claim.go:782` · `api.go:427`). 인증은 공용 bearer 토큰이다
- result 본문은 `store.StepResult` 를 그대로 품고 `steps.result` (jsonb) 로 들어간다. **봉인할 때
  `StepResult` 로 다시 푼다** (`seal.go:139`) — `StepResult` 에 없는 칸은 Record 에서 사라진다
- 노드는 result 를 **임대가 죽을 때까지 재시도**한다. 그래서 result 를 400 으로 거절하면 그
  보고는 영영 안 들어간다

### 1.3 진행 조회와 QUEUED

- `GET /v1/runs/{id}` 가 단계마다 `StepView` 를 싣는다 (`observe.go:21`). 본문과 판정은 안 싣는다
  (I4 — 봉인 전에는 Record 가 없다). 화면이 폴링한다 (`api.go:704`)
- QUEUED 인 Run 은 제출 때 후보가 다 점유돼 기다리는 Run 이다 (`api.go:556` · ADR-064). 제출과
  대기열 훑기는 **점유 집합(`busyIn`)과 drain 집합(`drainingIn`)을 합쳐** 매처에 넘긴다
  (`queue.go:233` · `api.go:540`). 그래서 밖에서는 둘이 같아 보인다 — US-7 의 출발점이다
- 매처 `match.Match` 는 배정 아니면 거절 하나를 준다. **수를 돌려주지 않는다** — 수는 거절 문구의
  글자 안에만 있다 (`match.go:100` · `:118`). 수를 세는 함수 `countSatisfying` 는 패키지 밖에 안 보인다.
  매처가 쓰는 판단은 `contract.Advert.Satisfies` 하나다
- `observe.go:85` 의 주석 — 「관측 경로에서 매처를 안 부른다 (ADR-065)」. ADR-065 가 막은 것은
  노드 목록의 `free` · 순위 · 필터다 (확인 뒤 행동 경쟁). QUEUED 사유는 배정이 아니라 설명이지만,
  같은 경쟁을 부르지 않게 모양을 골라야 한다 (물음 6)

### 1.4 앞 유닛이 넘긴 것

- `Claimed` (`claim.go:66`) 는 노드가 받는 유일한 단계 모양이다. 계약의 칸을 `fillFromContract`
  (`claim.go:599`) 가 지역 구조체로 풀어 옮긴다. contract-grammar 가 더한 칸 일곱(effect · budget ·
  discover · sync · builds · ir · merge)은 **아직 안 실린다**. 지금 build · merge 단계를 노드가 집으면
  「run step has an empty argv」로 실패한다 (`internal/enode/claim.go:625`)
- `steps.kind` 에 `build` · `merge` 가 들어올 수 있다 (contract-grammar). claim 이 merge 종류를
  알아보는 재료다

### 1.5 정본이 이미 정한 것

```text
   exited 본문          { outcome: { kind: exit | signal | timeout, code }, exited_at, attempt }
   받으면               phase = finalizing · phase_since = exited_at.  state 는 CLAIMED 그대로 · 임대도 그대로
   판정이 아니다         success_when 대조 · 다음 단계 · 정산을 안 한다.  종결 전이는 result 하나
   멱등                 같은 단계 · attempt · 노드 instance 의 재전송은 200.  이미 종결된 단계면 200
   유실                 result 가 같은 사실(exited_at · finalize · upload)을 다시 싣는다
   보내지 않는 노드       phase 가 running 에 머문다
   merge                claim 되는 순간 waiting (결정 1-12)
   exit                 exited 가 나른 사실이지 판정이 아니다
```

정본 절 제목의 경로(`/v1/steps/{id}/exited`)와 표의 경로(`/v1/runs/{run}/steps/{seq}/exited`)가
다르다. 표의 것을 쓴다 (Application Design 이 정했다 · 정본 되돌림 목록에 있다).

---

## 2. 물음 일곱

답을 `[Answer]:` 뒤에 적는다. 권장을 **A** 에 둔다. 기호 옆에 뜻을 적었다.

### Question 1 — 종료 보고를 받을지 · 거절할지 · 조용히 넘길지

정본은 「다르면 거절」과 「재전송 · 늦은 도착 · 종결된 단계는 200」만 적었다. 경우마다 정한다.
「키」는 Run · seq · attempt · 노드 · instance 다.

```text
   경우                                              A (권장)
   CLAIMED · 노드 · instance · attempt 가 맞고 running   받는다.  finalizing · phase_since · exit 를 적는다.  200
   같은 키의 재전송 (이미 finalizing)                  아무것도 안 바꾼다.  200.  처음 값이 남는다
   같은 키인데 outcome 이나 exited_at 이 다르다        처음 값이 남는다.  200.  로그에 경고 한 줄
   이미 종결 (DONE · FAILED · SKIPPED)                아무것도 안 바꾼다.  200
   attempt 가 지금보다 작다 (되돌림 뒤 옛 회차)          아무것도 안 바꾼다.  200 — 늦은 도착이다
   attempt 가 지금보다 크다                           거절 — 아직 없는 회차다
   CLAIMED 인데 노드가 다르다                          거절
   CLAIMED 인데 instance 가 다르다 (재시작한 생)        거절
   PENDING (아직 안 집혔다)                            거절 — 집지 않은 단계의 종료는 없다
   merge 단계 (waiting)                              아무것도 안 바꾼다.  200 — merge 는 명령이 없다
   ask · acquire 단계                                 거절 — 노드가 수행하지 않는다
   Run 이 RUNNING 이 아니다 (취소 · 종결)              아무것도 안 바꾼다.  200
   Run 이나 단계가 없다                                404
```

거절의 응답 코드 — A 는 **409** 다. result 가 「claim 하지 않은 단계」에 이미 409 를 쓴다
(`api.go:427`). 노드는 409 를 받으면 exited 재시도를 멈춘다 (finalize 유닛이 따른다).

A) 위 표 그대로 · 거절은 409

B) 위 표 그대로 · 거절은 403 (다른 생 · 다른 노드는 권한 문제로 본다)

C) 거절 없이 모두 200 으로 받고 맞지 않으면 무시한다 (가장 관대함. 대신 잘못 보낸 노드가 모른다)

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2 — 종료 보고의 본문과 검사

정본 본문은 `outcome` · `exited_at` · `attempt` 다. Application Design 이 대조를 위해 `node` ·
`instance` 를 더했다 (result 가 이미 `node` 를 본문에 싣는 방식과 같다).

A) 다섯 칸 `node` · `instance` · `attempt` · `outcome` · `exited_at` 을 받는다. 400 조건 —
   JSON 이 아님 · `node` 나 `instance` 가 빔 · `attempt` 가 음수 · `exited_at` 이 RFC 3339 가 아님 ·
   `outcome.kind` 가 exit · signal · timeout 밖 · kind 가 exit 인데 `code` 가 없음. signal 과 timeout 의
   `code` 는 있어도 되고 없어도 된다. `exited_at` 의 값(미래 · 과거)은 검사하지 않는다 (물음 3).
   `exit` 칸에는 `outcome` 만 그대로 적는다 — `exited_at` 은 `phase_since` 에 이미 있다

B) A 와 같되 `exited_at` 이 Mediator 시계로 미래(예: 5분 넘게 앞섬)면 400

C) A 와 같되 `instance` 를 비워도 받는다 — `claimed_instance` 가 비어 있는 옛 생과 맞추려고.
   (옛 노드는 exited 를 아예 안 보내므로 실제로 쓰일 일은 없다)

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3 — 두 시계

한 단계의 줄에 두 시계가 섞인다. `started_at` · `ended_at` 은 Mediator 가 적고, `exited_at` 은
노드가 보낸다 (실행 계획 7절 ②). US-8 의 상한 「`phase_since` + finalize 예산 + upload 예산」은
노드가 자기 시계로 예산을 지키므로 노드 시계로 맞다.

A) **칸마다 시계를 하나로 고정하고 문서에 적는다. 보정하지 않는다.**

```text
   진행 조회 · Record 의 칸     시계         적는 쪽
   started_at · ended_at       Mediator     claim · result 를 받은 때 (오늘 그대로)
   phase_since (running)       Mediator     claim 때 — started_at 과 같은 값
   phase_since (waiting)       Mediator     claim 때 — started_at 과 같은 값
   phase_since (finalizing)    노드          exited 의 exited_at
   exited_at · finalized_at    노드          result 가 싣는다 (Record 의 단계 기록)
```

   phase_since 는 phase 에 따라 시계가 바뀐다. 그 한 줄을 `StepView` 의 godoc 과 정본에 적는다.
   두 시계의 차이를 재거나 고치지 않는다 — NTP 가 맞춘 기계라고 본다

B) phase_since 를 언제나 Mediator 시계로 한다 — finalizing 이면 exited 를 **받은** 때.
   노드의 exited_at 은 `exit` 안에 따로 싣는다. 한 줄이 한 시계가 되지만 정본(:462)과 다르고,
   US-8 의 상한이 전송 지연만큼 늦게 나온다

C) A 와 같되 진행 조회에 칸 하나(`phase_clock`: node · mediator)를 더해 시계를 값으로 알린다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 4 — phase 의 전이와 종결 뒤에 남는 것

```text
   사건                       phase        phase_since      exit
   claim (merge 가 아님)       running      claim 시각         -
   claim (merge)              waiting      claim 시각         -
   재전달 (같은 생)            그대로 running · waiting 로 다시 적고 claim 시각도 다시 (started_at 과 같이)
   exited 수락                 finalizing   exited_at         outcome
   되돌림 · 되감기 (PENDING)    지운다        지운다             지운다 (result 도 지운다 — 오늘 그대로)
   result (종결)               ?            ?                 ?
   재시작 · 임대 만료 · 취소     ?            ?                 ?
```

마지막 두 줄이 물음이다. services.md 의 「Finalize 중 노드가 죽으면 Record 에 명령은 끝났고
Finalize 는 못 끝났다는 사실이 exit 와 함께 남는다」를 어디에 남기나.

A) **칸을 지우지 않는다 — 마지막 값이 남는다.** 진행 조회는 `phase` · `phase_since` 를 CLAIMED
   일 때만 싣고, `exit` 는 값이 있으면 상태와 무관하게 싣는다. Record 의 단계 기록은 `exit` 와
   마지막 phase(`last_phase`)를 싣는다 — 결과 없이 FAILED 로 끝난 단계가 `last_phase: finalizing` 과
   `exit` 를 가지면 「명령은 끝났고 Finalize 는 못 끝났다」다. 종결 전이가 phase 를 건드리지 않으므로
   종결 경로 다섯(result · 재시작 · 임대 만료 · 취소 · 계획 거절)을 하나씩 안 고친다

B) result 가 오면 phase 를 지운다. 나머지 종결 경로는 남긴다. 진행 조회는 값이 있으면 싣는다.
   Record 는 `exit` 만 싣는다 — phase 는 싣지 않는다

C) A 와 같되 Record 에 `last_phase` 를 싣지 않는다 — `exit` 가 있고 result 의 `finalized_at` 이 없으면
   그것으로 「Finalize 를 못 끝냈다」를 읽는다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 5 — result 에 더하는 칸의 모양

유닛 정의는 「결과 보고가 새 칸을 담아 봉인한다 — 종료 시각 · 결과 확정 시각 · 예산 결과 · 원인 코드 ·
진단 · 보존 상태 · build manifest · merge 결과」다. 그런데 진단 · 보존 · build · merge 의 안쪽 모양은
뒤 유닛(finalize · checkpoint · bake)이 정한다. 그리고 봉인이 `StepResult` 로 다시 풀므로(1.2)
`StepResult` 에 없는 칸은 Record 에서 사라진다.

A) **두 층으로 나눈다.**

```text
   이 유닛이 모양까지 정한다     exited_at · finalized_at (시각)
                               finalize · upload (ok · timeout · error)
                               reason (finalize_timeout · upload_timeout · merge_wait_timeout ·
                                       bake_in_progress — ADR-075 · ADR-077 의 목록)
   이 유닛은 자리만 둔다         diagnostics · checkpoint · changeset · build · merge
                               json.RawMessage 로 받아 그대로 봉인한다.  Mediator 는 안 읽는다.
                               안쪽 모양은 그 칸을 채우는 유닛이 노드 쪽 타입으로 정한다
```

   값 검사는 하지 않는다 — 모르는 `finalize` 값이나 `reason` 도 그대로 봉인한다. result 를 400 으로
   거절하면 노드가 임대가 죽을 때까지 재시도하다 보고를 잃는다 (1.2). 뒤 유닛이 `claim.go` 를 다시
   안 만지므로 파일 행렬의 「claim.go 는 step-phase 만」이 지켜진다

B) 이 유닛은 시각 둘 · finalize · upload · reason 다섯만 더한다. 나머지는 그 칸을 채우는 유닛이
   `StepResult` 에 직접 더한다 (그 유닛들이 `claim.go` 를 고친다 — 파일 행렬을 고친다)

C) 여덟 칸을 모두 이 유닛에서 타입으로 정한다 (뒤 유닛의 설계를 앞당긴다)

D) Other (please describe after [Answer]: tag below)

[Answer]: C

### Question 6 — QUEUED 사유: 후보 수 셋

완료 조건 4 는 「대기가 다른 Run 때문인지 drain 때문인지」다. Application Design 은 요구 줄마다
`live` · `busy` · `draining` 셋을 실었다. 세는 방식과 겹침을 정한다.

A) **배타로 센다 · 매처가 쓰는 판단(`Advert.Satisfies`)을 그대로 쓴다.**

```text
   live       그 요구를 만족하는 살아 있는 광고의 수 (만료 안 됨)
   busy       live 중 임대를 쥔 것 — 다른 Run 이 맡았다
   draining   live 중 임대가 없는데 drain 이 걸린 것 — drain 만 아니면 받을 수 있었다
```

   busy 를 먼저 센다. 점유된 노드는 drain 이 없어도 못 받으므로, `draining` 이 0 보다 크면 곧 「drain
   때문에 기다린다」로 읽힌다. `match.Match` 를 부르지 않는다 — 수를 안 돌려주고 첫 실패에서 멈춘다
   (1.3). `internal/match` 는 안 바뀐다. 남는 수(`live - busy - draining`)를 칸으로 내지 않는다
   (ADR-065 가 기각한 `free`). 응답에 DB 시각 한 칸(`candidates_at`)을 붙인다 — 관측은 시각과 함께다.
   QUEUED 인 Run 의 `GET /v1/runs/{id}` 에만 싣는다. 목록(`GET /v1/runs`)과 RUNNING 에는 없다.
   세다가 실패하면 칸을 빼고 `warnings` 에 한 줄을 적는다 (requires 와 같은 방식)

B) 겹침을 허용한다 — busy 와 draining 을 따로 센다 (임대를 쥐고 drain 도 걸린 노드는 둘 다에 든다)

C) Application Design 의 문장대로 `match.Match` 를 점유 집합과 drain 집합으로 따로 두 번 부르고,
   거절 문구에서 수를 읽는다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 7 — `Claimed` 에 계약의 새 칸 일곱을 싣는 모양 (contract-grammar 가 넘긴 일)

A) **계약의 칸을 그대로 싣는다.** `Claimed` 에 `effect` · `budget` · `discover` · `sync` · `builds` ·
   `ir` · `merge` 를 contract 패키지의 타입 그대로 더하고, `fillFromContract` 가 계약에서 옮긴다.
   기본값은 채우지 않는다 — 노드가 `contract.Step` 의 메서드(`EffectOrDefault` · `Budgets` ·
   `MergeWait`)로 채운다. Mediator 와 노드가 같은 상수를 읽는다 (contract-grammar 의 권장).
   재전달도 같은 함수를 지나므로 같은 칸이 실린다

B) Mediator 가 기본값을 채워 싣는다 — 노드는 적힌 값만 읽는다. 옛 Mediator 에서 온 claim 과 새
   Mediator 에서 온 claim 이 같은 단계에 다른 값을 줄 수 있다

C) 이 유닛에서 싣지 않는다 — 칸을 처음 읽는 유닛(finalize · bake)이 싣는다 (`claim.go` 를 그 유닛들도
   고친다)

D) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## 3. 산출물 계획 (체크박스)

답이 들어오고 모호함이 풀린 뒤에 채운다. 자리는
`aidlc-docs/v4-run-finalize-bake/construction/step-phase/functional-design/` 이다.

- [x] 답을 읽고 모호함을 확인한다 — 있으면 되물음 파일을 만든다 (2026-09-25T08:30:40Z · 물음 넷 — `step-phase-functional-design-clarification-questions.md` · 답 모두 A)
- [x] `domain-entities.md` — `steps` 의 새 칸 셋 · phase 어휘 · exited 본문 · `ExitedReport` ·
      `StepResult` 의 새 칸 · `StepView` · `RequireView` 의 새 칸 · `Candidates` · `Claimed` 의 새 칸 ·
      Record 단계 기록의 새 칸 · 칸마다의 시계
- [x] `business-rules.md` — 종료 보고의 수락 표와 응답 · 본문 검사와 영어 문구 · phase 전이 표 ·
      되돌림 셋에서 지우는 칸 · result 새 칸의 처리 · 후보 수의 정의 · 진행 조회에 싣는 조건
- [x] `business-logic-model.md` — `MarkExited` 의 흐름(한 문장의 UPDATE 로 멱등을 지키는 법) ·
      claim 과 재전달의 phase · 봉인까지의 흐름 · `CandidatesFor` 의 계산 · 진행 조회의 조립 ·
      라우트 등록 18 -> 19 · 옛 노드와의 공존 · 조각 0
- [x] 파일 행렬 밖 자리 — `rollback.go` · `ask.go` (되돌림 셋) · `seal.go` (Record) · `store.go`
      (수를 세는 쿼리)가 필요한지 확인해 적는다
- [x] 다른 유닛에 넘기는 것 — finalize (exited 보내기 · 409 에서 멈춤 · result 새 칸 채우기) ·
      checkpoint · bake (RawMessage 칸의 안쪽 모양 · merge 의 waiting)
- [x] 정본 되돌림 목록 — mediator-api.md exited 절(경로 · 본문의 node · instance · 거절 코드) ·
      진행 조회 절(phase_since 의 시계 · candidates) · result 절(새 칸)
- [x] 표기 검사 (`enode-design/scripts/emphasis-check.py`) · 사용자가 싫어한 말투 검사 · 사내 이름 검사

---

## 4. 확장 준수 — 이 단계

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표의 첫 줄(종료 보고의 인스턴스 대조)은 물음 1 · 2 가 닫고, 강도는 이 유닛의 NFR Requirements 가 본다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 저장소의 표 시험 관례를 따른다 |
