# obs — Functional Design 계획

AI-DLC Construction · **obs 유닛**(W0 · CP1 · 토대)의 Functional Design Part 1 이다.
설계 본문이 아니라 **무엇을 설계할지의 계획과 물음**이다. 답이 오면 그 값으로
`aidlc-docs/taeels/construction/obs/functional-design/` 아래 산출물 셋을 낸다.

정본 — 유닛 정의 `aidlc-docs/v1-run-dhseo/inception/application-design/unit-of-work.md` §1 ·
의존·파일 행렬 `.../unit-of-work-dependency.md` · 게이트 `.../unit-of-work-story-map.md` ·
팩 `requirements/`(enode-features 3.1.1 · decisions 1~3절 · constraints · scene-gates CP1) ·
설계 정본 `enode-design/adr/ADR-065` · `ADR-069` · `ADR-060` ·
`enode-design/protocol/mediator-api.md`.

---

## 0. 이 유닛이 지는 것 (한눈)

```text
   신규 라우트 둘    GET /v1/nodes            ADR-065 응답 모양 그대로
                    GET /v1/runs             decisions 2절 응답 모양 · 필터 넷
   기존 라우트 확장  GET /v1/runs/{id}        requires · as (ADR-069) · steps[].chosen (ADR-060)
                    POST /v1/nodes           정책 복사와 되돌림.  이 유닛의 유일한 쓰기
   store 읽기       Nodes · Runs · StepView.chosen · getRun 의 requires
   store 쓰기       UpsertAdvert — 어휘 검사 · 이전 draining 반환
   store 열         draining (nodes) · submitter (runs).  둘 다 additive
   광고 필드        contract.Advert 에 Policy{Drain}
   runctl.Client    Nodes · Runs — 원문 JSON 반환 (mcp 글자 일치 · panel 조회)
                    Run.Requires · Step.Chosen · Step.StartedAt — 파싱 타입도 넓힌다
   설정             ENODE_DEMO_MODE (기본 거짓).  데모 인스턴스의 읽기 셋 무인증

   지는 기능  3.1.1 (관측 API 부분) + 3.4.4 (submitter 컬럼·목록 노출)
   재는 게이트 CP1 (+ CP0 회귀)
   의존       없다.  이것이 병합되어야 W1(queue · mcp · ui)이 선다
```

---

## 1. 착수 전에 실측한 것 (기존 코드)

계획을 세우기 전에 저장소를 읽었다. **유닛 정의와 어긋나는 자리 하나**를 찾았고
그것이 아래 물음 1 이다.

```text
   확인한 사실                                                      자리
   ─────────────────────────────────────────────────────────────   ────────────────────────────
   mux 등록 라우트가 열다섯이다.  회귀 대상이 그 열다섯이다           internal/api/api.go:62~76
   nodes 표에 draining 열이 없다.  유닛 정의는 "열은 이미 있음"       internal/store/schema.sql
     이라고 적었으나 이 저장소에는 없다                              (nodes 표 · ALTER 로도 없음)
   runs 표에 submitter 열이 없다 (obs 가 additive 로 더한다)         internal/store/schema.sql
   steps.chosen 열은 있다.  StepView 가 그것을 안 싣는다             schema.sql 끝 · observe.go:21~38
   runView 에 requires 도 as 도 없다                                internal/api/api.go:156~169
   contract.Require 의 MarshalJSON 이 attrs 를 형제 키로 편다        internal/contract/contract.go:350
     (as · capability · count 를 뺀 나머지가 속성)
   runctl.Client 에 Nodes 도 Runs 도 없다                            internal/runctl/client.go
   contract.Advert 에 정책(draining) 필드가 없다 — drain 유닛 몫      internal/contract/advert.go:9
   internal/api/ui 와 /ui/ 마운트는 이미 있다 (cardnews 병합)         internal/api/api.go:80
   테스트는 진짜 Postgres 를 쓴다.  CI 가 postgres 서비스를 붙여      internal/api/api_test.go:29~
     t.Skip 이 CI 에서 발동하지 않는다                               .ci-allowed-skips
```

`sandbox` 라는 **키**는 이 저장소의 Go 코드 어디에도 없다 (`grep` 실측 0건).
그러나 **그 값이 탈 자리는 이미 있다** — 노드 설정의 `labels` 가 광고 속성에
합쳐져 `nodes.capabilities` 에 앉고, 이 유닛이 `capabilities` 를 그대로 내므로
`capabilities[].attrs.sandbox` 로 화면에 닿는다. 코드 변경 0 이다.

그래서 obs 는 `GET /v1/nodes` 에 `sandbox` **필드**를 안 만들고, 출처가 어느 쪽으로
결정되든 **이 라우트를 다시 열 이유가 없다.** 출처 결정의 담당은 로스터 §6 이
**ui · runixs** 로 두었다. (앞 판은 이것을 「출처가 결정되면 이 라우트가 함께
열린다」로 적었다 — 2차 검토가 접었다. 산출물의 정본은 `business-rules.md` §5 다.)

---

## 2. 물음 다섯

답은 `[Answer]:` 뒤에 글자 하나로 적는다. 계약에 걸리는 것만 물었다 —
스키마 소유 · 응답 모양 · 등록 줄이다.

**셋은 계획 단계에서, 둘은 하류 검토 뒤에 물었다** (7절). 넷째와 다섯째는
사용자 규칙의 「단계마다 셋까지」를 넘지만, 검토가 새로 드러낸 계약이라
안 묻고 정하면 W3 에 되열린다.

### 물음 1
`nodes.draining` 열이 이 저장소에 없다. CP1 은 `GET /v1/nodes` 에 `draining` 이
실려 있을 것을 요구하고(`scene-gates.md` CP1 · `ADR-065` 2026-09-04 개정), 유닛
정의는 그 열을 **drain 유닛의 「이미 있음」**으로 적었다. 누가 만드나.

A) obs 가 `schema.sql` 에 additive 로 더하고 (`text NOT NULL DEFAULT ''`) 읽기만 한다.
   쓰는 것(`UpsertAdvert` 의 복사)은 drain 유닛이 그 위에 얹는다.
   W0 이 먼저 병합되므로 drain 이 딛을 자리가 미리 선다

B) drain 유닛이 열을 만들고, obs 는 그때까지 응답에 상수 `""` 를 낸다.
   store 접점 충돌이 줄지만 CP1 은 drain 병합 전까지 `draining` 을 실증하지 못한다

C) obs 가 열도 만들고 `UpsertAdvert` 의 복사까지 넣는다.
   그러면 drain 유닛은 광고에 정책을 싣는 쪽(`internal/enode`)만 남는다

X) Other (please describe after [Answer]: tag below)

[Answer]: C

### 물음 2
`GET /v1/runs/{id}` 에 더하는 `requires[]` 의 **속성 표기**다. 기존
`contract.Require.MarshalJSON` 은 속성을 형제 키로 편다(`{"as","capability","arch":"armv7"}`).
`ADR-069` §4 의 예시는 `attrs` 로 감싼다(`{"as","capability","attrs":{...}}`). 둘이 다르다.

A) `ADR-069` §4 예시 그대로 — 관측 전용 뷰 타입을 새로 두고 `attrs` 로 감싼다.
   읽는 쪽(화면 · MCP)이 예약 키를 몰라도 속성을 뽑는다. 계약 입력과는 다른 글자다

B) `contract.Require` 를 그대로 마셜 — 계약 입력과 응답이 같은 글자다.
   읽는 쪽이 `as` · `capability` · `count` 를 빼고 나머지를 속성으로 읽어야 한다

X) Other (please describe after [Answer]: tag below)

[Answer]: A

### 물음 3
`submitter`(게스트 로그인 이름 · `decisions.md` §8.6)를 obs 가 어디에 노출하나.
열을 더하고 읽는 것은 obs, 쓰는 것은 demo-back 이다.

A) `GET /v1/runs` 목록 행에만. 언제나 싣는다 (없으면 `"submitter": ""`).
   `verdict` · `ended_at` 이 종료 전에 `null` 로 그대로 나가는 것과 같은 규칙이다

B) 목록 행과 `GET /v1/runs/{id}` 상세 둘 다. 언제나 싣는다.
   RUN 카드가 상세로 들어가도 이름이 이어진다

C) 목록 행에만. 값이 있을 때만 싣는다 (`omitempty`).
   실 함대 Run 의 행이 오늘 모양 그대로 남는다

X) Other (please describe after [Answer]: tag below)

[Answer]: A

### 물음 4
데모 모드에는 S0(토큰 입력)이 없다 — 「토큰 입력 대신 Guest 로그인」이다. 그런데
데모 대시보드가 3D 함대뷰 · run 목록 · 작업 그래프를 그리려면 이 유닛의 잠긴
라우트를 불러야 한다. 브라우저가 무엇으로 읽나.

A) 데모 인스턴스에서만 읽기 셋을 무인증으로 등록한다. 쓰기는 그대로 잠긴다.
   가둠 셋의 「브라우저에 실 토큰 없음」이 그대로 산다

B) demo-back 이 무인증 읽기 프록시를 낸다. obs 는 무변경이지만 관측 표면이
   두 벌이 되고 demo-back 의 범위가 넓어진다

C) 지금 정하지 않고 열린 미정으로 진행자에게 넘긴다. 어느 갈래로 닫히든
   W3 에 obs 의 등록 줄과 auth 규칙을 다시 연다

X) Other (please describe after [Answer]: tag below)

[Answer]: A

### 물음 5
팩 3.1.1 은 S1 노드 카드에 「지금 무슨 단계」를 그리라고 적었는데
`GET /v1/nodes` 는 `lease{run_id, not_after}` 까지만 준다. 단계 이름은
`GET /v1/runs/{id}` 의 `steps[]` 에만 있다.

A) 두 홉을 수락하고 FD 에 계약으로 적는다. 응답 모양은 `ADR-065` 그대로 두고,
   임대된 노드만 `lease.run_id` 로 상세를 한 번 더 부른다

B) `lease` 에 `CLAIMED` 단계의 이름 · 회차 · 시작 시각을 얹는다. 호출이 하나로
   끝나지만 `ADR-065` 예시에 없는 필드가 늘고 관측 표면의 책임 경계를 넘는다

X) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## 3. Functional Design 실행 계획

답 다섯이 오면 아래를 순서대로 낸다. 각 항목은 산출물의 절 하나에 대응한다.
(물음 4·5 는 1차 검토 뒤에 늘었다 — 7절. 아래 목록도 그만큼 늘었다.)

### 3.1 답 반영

- [x] 물음 다섯의 답을 읽고 어긋남·모호함을 검사한다. 모호하면 clarification 파일을 낸다
      — 답은 C · A · A · A · A. 물음 3 은 사용자가 추가 설명을 요청해 한 번
      더 설명하고 다시 물었다. **답 4=A 는 `ADR-065` 2절의 인증 줄을 벗어난다** —
      2차 검토가 찾았고 사용자가 A 를 유지하기로 정했다. 어긋남을 산출물에 기록한다
      (`business-logic-model.md` §7.1 · `business-rules.md` §1 · §6)
- [x] 답을 `aidlc-docs/taeels/audit.md` 에 원문 그대로 기록한다
- [x] 답이 팩의 권장값을 벗어나면 근거를 한 줄로 적는다 — 벗어난 값 없음. 답 C 가
      파일 행렬을 넓히므로 그것을 6절에 적었다

### 3.2 domain-entities.md — 관측 표면이 쓰는 것

- [x] `NodeView` — `node_id` · `label` · `instance` · `capabilities` · `seen_at` ·
      `expires_at` · `lease{run_id, not_after}` · `draining`. `principal` 은 안 낸다 (ADR-065)
- [x] `RunRow` — `run_id` · `state` · `verdict` · `work_id` · `created_at` ·
      `ended_at` · `assigned` (+ `submitter` · 물음 3)
- [x] `RunFilter` — `state` · `since` · `work` · `limit`. 어느 열을 보는지 못 박는다
- [x] `StepView` 확장 — `chosen` (ADR-060). 언제나 싣는다. `false` 가 사라지면
      「안 고른 SKIPPED」와 「고른 뒤 SKIPPED」가 다시 한 글자가 된다
- [x] `requires[]` 항목 (물음 2 의 답 모양) 과 `steps[].uses` 의 이음
- [x] `runs.submitter` 열의 형(型) · 기본값 · additive 마이그레이션 자리
- [x] (물음 1 이 A·C 면) `nodes.draining` 열의 형 · 기본값 · 값 어휘(`""`·`graceful`·`at-boundary`)
- [x] 표마다 「어느 기존 열에서 나오나」를 적는다. 새 저장소는 없다 (3.1.1 데이터 관리)

### 3.3 business-logic-model.md — 읽기 경로의 흐름

- [x] `GET /v1/nodes` — `Store.Nodes` 한 질의(노드 + 임대 LEFT JOIN) ·
      `expires_at > now()` 만 · `observed_at` 을 DB 시계에서 낸다 (임대 시각과 같은 시계)
- [x] `GET /v1/runs` — `Store.Runs(RunFilter)` · `created_at` 내림차순 · `limit` 기본 100
- [x] `GET /v1/runs/{id}` 확장 — 지금 유효한 계약(`LiveContract` · 마지막 판)에서
      `requires` 를 꺼낸다 (ADR-069 §4 「계약이 여러 판이면 마지막 판」)
- [x] `Steps` 질의에 `chosen` 을 더한다 — 열은 이미 있다
- [x] `runctl.Client.Nodes` · `Runs` — 원문 `json.RawMessage` 를 그대로 돌려준다.
      파싱을 안 하는 이유는 mcp 의 글자 일치(CP5)다
- [x] 어느 흐름에도 매처를 안 태운다 (ADR-065 · ADR-069 §5 — 재료를 주고 판정은 안 준다)
- [x] 흐름 넷을 ASCII 다이어그램 하나로 그린다 (`common/ascii-diagram-standards.md`)

### 3.4 business-rules.md — 규칙과 검증

- [x] 인증 — 실 함대는 두 라우트 다 `s.auth` 를 거친다. 지금 표면과 같은 규칙.
      **데모 인스턴스만 읽기 셋 셋이 무인증이다**(답 4=A) — 그 어긋남을 SECURITY-12
      칸에 조건부로 적는다
- [x] 입력 검증 (SECURITY-08) — `limit` 의 상한과 위반 처리 · `state` 값 · `since` 시각
      형식 · 알 수 없는 질의 인자의 취급. **상한 값은 팩이 안 정했으므로 여기서
      정하고 근거를 적는다**
- [x] 필터가 보는 열을 못 박는다 — `state`=`runs.state` · `since`=`runs.created_at`
      하한 · `work`=`runs.work_id` 일치. `principal` 필터는 없다 (ADR-015 §1)
- [x] 만료된 광고는 `GET /v1/nodes` 에 안 나온다. draining 노드는 만료 전이면 나온다
- [x] 오류 코드 — 질의 인자가 나쁘면 `400`, 저장소가 죽으면 `503`. 기존 `fail` 규약
- [x] 빈 결과는 `200` + 빈 배열이다. `null` 이 아니다 (화면이 분기하지 않게)
- [x] `GET /v1/nodes` 는 필터를 안 준다 (ADR-065 §2 · 여유 질의를 안 만든다)
- [x] 페이지네이션 없음 — `limit` 뿐 (decisions 2절)

### 3.5 완료 조건

- [x] CP1 의 `curl` 줄(`scene-gates.md`:150)이 무엇을 요구하는지 그대로 옮기고
      각 요구를 위 규칙에 잇는다 — 노드 여럿 · 임대 하나 · draining 하나 · 상태 넷의 Run
- [x] CP0 회귀 목록 — 기존 열다섯 라우트 · 포맷 · vet · glyphscan · 커버리지 하한 ·
      스킵 감시 · 크로스 빌드
- [x] 접점 둘(`internal/store/store.go` · `internal/api/api.go`)에서 이 유닛이
      만지는 줄을 미리 세어 둔다. `api.go` 는 등록 두 줄과 `getRun` 확장뿐이다
      (constraints 「api.go 는 등록 줄만」)
- [x] 커버리지 — 새 패키지가 없으므로 기존 `internal/api` · `internal/store` ·
      `internal/runctl` 의 하한 80% 를 재측정한다

### 3.6 검토

- [x] 산출물 셋이 `CONVENTIONS.md` 1절(강조는 굵게만 · 장식 문자 금지)을 지키는지 본다
- [x] `content-validation.md` 로 다이어그램과 특수문자를 검사한다
- [x] security-baseline 준수 요약을 낸다 (SECURITY-08 · SECURITY-12 는 적용 ·
      SECURITY-01 · 03 · 07 은 기존 코드 사실로 기록만 — decisions 3절)
- [ ] 완료 메시지를 내고 승인을 기다린다. 승인 전에 코드를 안 쓴다

---

### 3.7 답 4·5 와 두 차례 검토가 늘린 것

§3.2~§3.4 는 답 셋을 전제로 쓴 목록이라 아래가 빠져 있었다. 산출물에는 있고
이 목록이 그것을 뒤늦게 진다.

- [x] `POST /v1/nodes` 의 왕복 셋 — 받고 · 어휘를 보고 · 되돌린다
      (`business-logic-model.md` §5). 답 1=C 가 이 흐름을 obs 로 들였다
- [x] `contract.Advert` 의 `Policy{Drain}` 과 세 자리의 이름
      (`domain-entities.md` §7.3)
- [x] `UpsertAdvert` 가 이전 `draining` 을 돌려준다 — 0행 갈래를 `coalesce` 로 접는다
      (`domain-entities.md` §8)
- [x] `submitter` 를 쓰는 입구 넷 — 열 · 컨텍스트 키 · `submit()` 리터럴 셋 ·
      두 생성자 (`domain-entities.md` §7.2)
- [x] `assigned` 가 jsonb `null` 로 들어오면 안 된다는 계약을 queue 에 넘긴다
      (`domain-entities.md` §3.1)
- [x] 데모 인스턴스의 무인증 읽기 등록 · 나가는 값의 목록 · 스위치 이름과 기본값 ·
      정본을 벗어난다는 기록 (`business-logic-model.md` §7)
- [x] `runctl.Step.StartedAt` — panel(W3)이 되열지 않게 W0 이 세운다
      (`domain-entities.md` §5)
- [x] 병합 조건을 못 박는다 — §3 curl 두 줄 + CP0 (`business-rules.md` §7.1)

---

## 4. 낼 파일

```text
   aidlc-docs/taeels/construction/obs/functional-design/
     domain-entities.md        관측 표면이 쓰는 것 (표 · 열 출처)
     business-logic-model.md   읽기 경로 넷의 흐름
     business-rules.md         규칙 · 검증 · 오류 · 완료 조건
```

화면이 없는 유닛이라 `frontend-components.md` 는 안 낸다 — 화면은 ui 유닛(runixs)이 진다.

---

## 5. 이 유닛이 안 하는 것

```text
   QUEUED 상태를 만드는 것          queue (W1).  obs 는 목록이 그 값을 낼 수 있게만 한다
   drain 정책을 광고에 싣는 것       drain (W2) 의 internal/enode 쪽.  답 1=C 로
                                   중앙의 복사·되돌림·어휘 검사는 obs 가 진다
   submitter 값을 넣는 것           demo-back (W2).  obs 는 열과 쓰는 입구와 읽기를 진다
                                   (컨텍스트 키 · submit() 리터럴 셋 · 두 생성자)
   데모 쓰기 라우트                  demo-back (W2).  obs 는 읽기 셋의 등록만 가른다
   화면                            ui (W1·W3)
   MCP 도구                        mcp (W1) — 같은 담당이지만 다음 유닛이다
   판정 · 매처 · 자연어 설명         ADR-065 · ADR-069 §5 가 막았다
   sandbox 필드를 새로 만드는 것      자리가 이미 있다 — capabilities[].attrs 로 닿는다.
                                   출처 결정은 ui 의 열린 미정 (roster §6) 이고
                                   어느 쪽으로 닫혀도 obs 는 안 열린다 (§1)
```

---

## 6. 답이 부른 것 (2026-09-08)

```text
   물음 1 = C   obs 가 nodes.draining 열을 만들고 UpsertAdvert 의 복사까지 넣는다
   물음 2 = A   requires[] 는 ADR-069 §4 그대로 — 관측 전용 뷰에 attrs 를 감싼다
   물음 3 = A   submitter 는 GET /v1/runs 목록 행에만 · 언제나 (빈 값도 키로)
   물음 4 = A   데모 인스턴스에서만 읽기 셋 셋을 무인증으로 등록한다.
                ADR-065 2절의 인증 줄을 벗어난다 — 어긋남을 산출물에 기록한다
   물음 5 = A   카드의 단계는 두 홉이다.  GET /v1/nodes 를 안 넓힌다
```

### 6.1 답 C 가 파일 행렬을 넓힌다

`UpsertAdvert` 가 복사할 값이 광고에 없다 — `contract.Advert`(advert.go:9)에
정책 필드가 없다. 그래서 이 답을 이행하려면 obs 가 **`internal/contract/advert.go`
에 필드 하나**를 더해야 한다. 그 파일은 어느 유닛의 행렬에도 없다.

```text
   행렬에 더하는 줄   internal/contract/advert.go   W(obs)   단독 obs · 필드 하나
```

**모양은 정본이 이미 정했다** — `ADR-063` §6 과 `mediator-api.md` 가 광고 본문을
`"policy": {"drain": ""}` 으로 적어 두었다. 그래서 `Policy{Drain string}` 을
`policy` 로 감싼다. (앞 판은 「값 하나이므로 구조체로 안 감싼다」로 `Draining string`
을 골랐다 — 1차 검토가 접었다. 감싸는 모양을 obs 가 짐작하는 것이 아니라 정본이
정해 둔 것이다. 지금 필드 수는 여전히 하나다.) 산출물의 정본은
`domain-entities.md` §7.3 이다.

**drain 담당(shin-son)이 딛는 자리가 이것으로 바뀐다** — 열도 광고 필드도
Mediator 쪽 복사도 W0 에 이미 서 있고, drain 은 `internal/enode` 의 정책 파일
읽기와 광고 적재 · 광고 응답 되돌림 · 두 모드 · at-boundary 취소만 진다.
`CONVENTIONS.md` 3.5 의 「행렬 밖 파일을 만진 diff」로 가져간다.

---

## 7. 하류 병렬 검토 (2026-09-08)

토대 유닛이라 하류에서 모순이 나오면 회귀가 비싸다. 사용자 지시로 검토를 넷으로
갈라 병렬로 돌렸다 — **queue·demo-back** · **drain·panel·transcript** ·
**ui·mcp** · **코드·스키마·CI 정합**. 각자 하류 담당의 입장에서 적대적으로 읽고
파일·줄 근거를 요구했다.

### 7.1 고친 것

```text
   무엇                                     겹친 수   고친 자리
   ──────────────────────────────────────   ───────   ────────────────────────
   광고 본문이 policy.drain 이다 (정본)         2      domain-entities §7.3
                                                     business-logic-model §5
   SECURITY-08 근거가 거짓이었다               2      business-rules §2.5 · §6
     (「본문이 없다」로 준수를 닫았다)
   정책 값 enum 검증이 주인이 없었다            2      business-rules §2.5
   observed_at 이 빈 함대에서 값이 없다         1      business-logic-model §2
   requires 를 어디서 채우나 (view 호출자 넷)   2      business-logic-model §4
   UpsertAdvert 가 이전 값을 안 돌려준다        2      domain-entities §8
   assigned 가 null 로 샌다 (정본은 [])        3      domain-entities §3
   instance · work_id 가 nullable 이다        1      domain-entities §2 · §3
   submitter 의 쓰는 입구가 없었다              1      domain-entities §7.2
   응답에 drain 되돌림이 주인이 없었다          1      domain-entities §7.3
   모르는 질의 인자 — nodes 는 400 이 정본      2      business-rules §2.4
   CP1/CP2 지분을 착각했다                     2      business-rules §7.1 · §7.2
   게이트 인용이 §5 가 아니라 §3 이다           1      business-rules §7.1
   RunsQuery 가 정의되지 않았다                3      domain-entities §4
   not_after 가 과거일 수 있다 (ADR-047)       1      domain-entities §2.1
   runctl.Run·Step 에 requires·chosen 없음    2      domain-entities §5 · §8
   접점 목록에 observe.go·schema.sql 누락      1      business-rules §7.5
   커버리지 수를 안 적었다                     1      business-rules §7.6
   count 규칙 표현이 틀렸다 (0 이면 생략)       1      domain-entities §6
   since 경계가 열려 있었다 (포함으로 못박음)   1      business-rules §2.2
   limit 빈 문자열이 분류 안 됐다              1      business-rules §2.1
   sandbox 문장이 「자리도 없다」로 읽혔다      1      business-rules §5
```

### 7.2 검토가 새로 연 물음 둘

물음 4(데모 읽기 인증)와 물음 5(카드의 단계)는 계획 단계에 없던 것이다.
둘 다 이 유닛의 등록 줄과 응답 모양에 걸리므로 물어서 닫았다 — 2절.

### 7.3 구멍이 아니었던 것 (반증을 적는다)

지어낸 문제를 안 남기기 위해 **검토가 반증한 것**을 적는다.

```text
   panel 의 principal 표시      enode.Derive 가 로컬에서 낸다.  constraints §4 가
                              GET /v1/nodes 에 principal 을 내는 것을 명시적으로 제외
   transcript 의 node 거르기    assigned 로 충분하다.  claim.go 가 「원천은 runs.assigned
                              다 — steps 가 아니다」로 못 박았고 applyRelease 가 안 지운다
   requires 추출               contract_versions 의 판이 계약을 익명으로 품어 평평하게
                              마셜된다.  판이 쌓여도 requires 가 안 사라진다
   심볼 상한                   enodectl 링크 그래프에 store·api·runctl 이 없다.
                              실측 net/http 6 · crypto/tls 1
   chosen 열                   dispatch.go 가 이미 SET chosen = true 를 쓴다.
                              상수 false 가 아니라 진짜 값이 나온다
   매처와의 중복               LiveAdverts 가 정책을 안 실어 겹칠 자리가 없다.
                              draining 은 queue 가 busy 로 합친다
   state 어휘 무검사            runs.state 에 CHECK 가 없다.  queue 를 안 막는다
   계획 1절의 실측 열두 줄       전부 맞았다.  하나도 틀리지 않았다
```

### 7.4 남긴 표시 (진행자)

```text
   ADR-068 의 At        panel 의 열린 미정.  광고로 닫히면 이 유닛 표면 넷이 함께 열린다
                        business-rules §5
   카드의 단계 정본       팩 3.1.1 은 단계를 요구하고 design/README §2 는 안 적었다.
                        어느 쪽이 중앙 카드의 정본인가 — domain-entities §2.2
   행렬 밖 파일 넷        observe.go · schema.sql · contract/advert.go · config/config.go
                        business-rules §7.5
```

---

## 8. 2차 병렬 검토 (2026-09-08)

1차(7절)는 **하류 담당별**로 갈랐다. 같은 축을 다시 돌리면 같은 것이 다시 나올
뿐이라 2차는 축을 바꿔 넷으로 갈랐다 — **문서 셋 내부 정합** · **정본 인용 대조** ·
**코드 실측과 구현 가능성** · **게이트 실증 가능성과 재개방 비용**. 원시 발견
일흔둘을 겹침으로 합쳐 마흔 남짓이 남았다.

### 8.1 실행으로 확정된 치명 둘

에이전트 둘이 PostgreSQL 17 에 직접 쳐서 재현했다. 의견이 아니다.

```text
   UpsertAdvert 의 CTE 가 첫 광고에서 NULL 을 낸다        겹친 수 3 (둘이 실행)
     WITH old 가 0행이라 RETURNING 의 스칼라 부질의가 SQL NULL.
     prevDrain string 스캔이 죽고 api.go 가 503 을 낸다.
     node_id 가 PK 이므로 그 경우는 정확히 「이 노드의 첫 광고」다 —
     어떤 노드도 함대에 못 들어오고, 시험은 매번 Truncate 하므로
     광고 109 자리가 전부 첫 삽입이다.  CP0 부터 빨갛다
     고침   RETURNING coalesce((SELECT draining FROM old), '')
            부질의 안에 coalesce 를 넣으면 0행에서 여전히 NULL 이다
     자리   domain-entities §8

   coalesce(assigned,'[]'::jsonb) 가 jsonb null 을 안 접는다   겹친 수 1 (실행)
     'null'::jsonb IS NULL 은 거짓이라 그대로 통과한다.
     오늘 그것을 넣는 자리는 없고, CreateQueuedRun(W1)이 제로값을
     쓰면 생긴다.  깨지는 것은 CP2 의 화면 판정이다
     고침   쓰는 쪽의 계약으로 적는다 — 열을 생략하거나 [] 로 넣는다
     자리   domain-entities §3.1 · business-rules §7.2
```

### 8.2 계약이 갈려 구현자가 반대로 짓던 자리

```text
   무엇                                겹친 수   어떻게 닫았나
   ─────────────────────────────────   ───────   ──────────────────────────────
   observed_at 의 조리법이 둘이었다        3       1행 CTE 를 FROM 에 두고 노드를
     (CROSS JOIN 대 별도 SELECT now())            LEFT JOIN 한다.  행이 0 이어도
                                                 시각이 서고 질의는 하나다
   정책 어휘 검사의 주인 (store 대 api)     2       store.  api 의 커버리지 여유가
                                                 5.6 문장뿐이다
   requires 의 omitempty 가 읽기 실패를    1       warnings 에 한 줄 싣는다.
     생략과 한 글자로 만든다                       runView 에 이미 있는 필드다
   빈 값 규칙이 limit 에만 있었다           1       넷에 같은 규칙을 붙였다
   POST /v1/nodes 응답 drain 의 생략 규칙   1       언제나 싣는다.  요청의 omitzero 와
                                                 반대 방향이다
   RunFilter · RunsQuery 의 Go 형이 없었다  1       형을 적었다.  Since 는 time.Time
                                                 이고 빈 값 판정이 IsZero 다
   RequireView 의 생략 규칙이 count 에만    1       count 만 생략.  attrs 는 비면 {}
   work_id 는 coalesce 인데 필터는 원열     1       비대칭을 적었다.  인덱스를 지킨다
```

### 8.3 하류가 obs 를 되열 자리 — 미리 막았다

```text
   submit() 의 「한 줄」이 세 자리였다      거절 경로 둘은 CreateRejectedRun 을 타고
                                        그 INSERT 에 submitter 가 없다.  데모에서
                                        가장 흔한 실패가 422 이고 그것이 CP9 의 행이다
   panel(W3)이 runctl.Step 을 되연다      S3 가 시작 시각을 요구하는데 Step 에 없다.
                                        담당이 갈리므로 W0 이 StartedAt 을 세운다
   config.go 가 단독이 아니다             demo-back 이 같은 스위치를 읽는다.
                                        접점으로 승격 — 진행자 직렬 대상
   UpsertAdvert 가 tx 를 안 받는다        queue 가 postNodes 를 tx 로 두르면 시그니처를
                                        다시 연다.  prevDrain 을 미리 세운 근거가
                                        그 자리에서 무효다 — 열린 채로 남긴다 (8.5)
```

### 8.4 게이트 — 병합 조건을 못 박았다

`scene-gates.md` 의 CP1 판정 재료 「상태 넷의 Run」 중 **넷째가 `QUEUED` 이고 그것은
W1 이다.** 저장소가 실제로 저장하는 상태는 `RUNNING` · `SUCCEEDED` · `FAILED` 셋뿐이고
(`RESOLVING` · `ALLOCATING` 은 코드에 0건, `VERIFYING` 은 한 함수 안에서 덮인다)
`enode-features.md` 3.2.1 이 그것을 정본으로 적었다.

**사용자 결정 — obs 의 병합 조건은 `scene-gates.md` §3 의 curl 두 줄 + CP0 이다.**
상태 셋으로 재고 넷째는 CP2(W1)가 잰다. CP1 전체는 화면 다섯이 ui 몫이라 W0 에
초록이 될 수 없다(ui 가 obs 병합 뒤에 착수하므로 순환이다) — 이 부분 통과를
진행자가 승인한다. `business-rules.md` §7.1 에 적었다.

같은 절에서 함께 고친 것 — 게이트 `curl` 에서 지웠던 `Authorization` 헤더를
정본대로 되돌렸고, `draining` 노드를 만드는 절차(실 데몬과 `node_id` 를 나눌 것 ·
180초 안에 재전송 · 그 노드는 임대를 못 쥔다)를 적었다. CP0 목록에는 빠져 있던
`crypto/tls` 상한 10 과 `probe.lock` 을 더했다.

### 8.5 진행자에게 넘기는 것 (2차에서 는 것)

```text
   답 4=A 가 ADR-065 2절을 벗어난다      정본 둘이 어긋나는 자리다(팩의 데모 전환에는
                                       S0 가 없다).  canon 서열대로면 enode-design 이
                                       이긴다.  사용자가 A 를 유지하기로 정했고,
                                       ADR-065 를 개정할지 데모를 정본 밖 예외로 둘지는
                                       진행자가 정한다.  BLM §7.1

   무인증 GET /v1/nodes 가 label 을 낸다  ADR-065 가 principal 을 안 내는 대신 「식별은
                                       label 에 있다」로 옮겼고 그 실체가
                                       <이메일 로컬파트>@<호스트명>:<워크스페이스> 다.
                                       게스트는 가명인데 노드 소유자는 실명이다.
                                       응답을 안 바꾸고 데모 함대의 label 로 닫는다 —
                                       누구 기계로 세우느냐가 노출의 크기다.  BLM §7.2

   submitter 가 설계 정본에 없다          enode-design 에 그 낱말이 0건이고 canon §4 의
                                       「팩이 정본에 더하는 것」 둘에도 없다.
                                       mediator-api 와 ADR-064 §4 를 누가 언제 고쳐
                                       올릴지가 안 정해졌다.  domain-entities §3.2

   ADR-063 §4 가 그 대가를 이름으로 거절   「저절로 풀리면 소유자가 되찾은 보드를 다음
                                       광고 주기에 도로 빌려주는 셈」.  옛 판 enode 가
                                       걸린 drain 을 지우는 것이 그것이다.  반대로
                                       가면 소유자가 파일에서 지워도 안 풀린다 —
                                       어느 쪽이든 정본 하나를 접는다.
                                       obs 는 ADR-012 쪽으로 가되 저울을 적었다.
                                       domain-entities §7.3

   오류 코드 넷이 정본 표 밖이다           mediator-api §1.2 는 400 을 「계약 문법」,
                                       503 을 「기동 미완」으로 적었다.  질의 인자
                                       검증과 실행 중 실패는 그 표에 없다.
                                       business-rules §4

   UpsertAdvert 의 tx 인자                queue(W1)의 설계에 달렸다.  필요하면 3접점
                                       파일의 시그니처를 W1 에 다시 연다
```

앞 판의 표시 셋(`ADR-068` 의 `At` · 카드의 단계 정본 · 행렬 밖 파일)은 그대로 산다.
**`sandbox` 는 표시에서 뺀다** — 자리가 이미 있고 어느 쪽으로 닫혀도 obs 를 안 연다.

### 8.6 반증 — 2차가 확인하고 안 고친 것

```text
   기존 열다섯 라우트가 안 깨진다          DisallowUnknownFields 가 저장소에 0건이고
                                       api_test 는 필요한 필드만 뽑아 단언한다.
                                       drain 키와 chosen:false 추가가 골든을 안 깬다 —
                                       마셜 골든 파일 자체가 없다
   ALTER 둘이 도는 DB 에 선다             Store.Migrate 가 schema.sql 을 통째로 Exec 하고
                                       mediator 가 기동마다 부른다.  steps.chosen 이
                                       이미 같은 패턴이다
   CTE 스냅숏 주장이 참이다               둘째 호출에서 갱신 전 값이 나온다.
                                       RETURNING OLD.* 은 PG18 부터라 17 에서는 CTE 가
                                       유일한 길이다 — 설계는 옳고 0행 갈래만 빠졌다
   LEFT JOIN 이 노드를 복제하지 않는다     leases.node_id 가 PRIMARY KEY 다
   omitzero 가 산다                      go.mod 가 go 1.26 이다
   데모 무인증 갈래가 배선상 가능하다       s.auth 는 통짜 미들웨어가 아니라 라우트마다
                                       감싸는 함수다
   정본 인용에 지어낸 것이 0건이다         수십 건을 원문과 대조했고 mediator-api 의
                                       한 줄은 글자까지 같았다
   표기 규약 통과                        emphasis-check.py 지적 0건
   커버리지 산술이 맞다                   0.2n + 여유 공식까지 검산했다.  다만 실측이
                                       스냅숏과 한 문장 다르다 — 328/403 · 여유 5.6
   계획 §1 의 실측 열두 줄                열하나가 정확하고 하나가 1행 어긋났다
                                       (observe.go:20~38 -> 21~38).  1차의 「전부
                                       맞았다」는 11/12 였다
```

### 8.7 1차 고침이 이 계획으로 안 따라왔다

문서 셋에는 1차 22건이 빠짐없이 반영돼 있는데 **이 계획만 옛 결론을 지고 있었다** —
§5 의 「obs 는 읽기만 한다」와 「열과 읽기만 진다」, §3.4 의 「두 라우트 다 `s.auth`」,
§6.1 의 `Draining string`, §1·§5 의 sandbox, §3·§6 이 답 셋에서 멈춘 것, §0 이
쓰기와 열 하나를 안 실은 것. 전부 이 판에서 고쳤다.

「안 하는 것」 목록은 **다른 담당이 경계를 읽으러 오는 자리**다. 문서 루트가
담당별이라 shin-son 도 runixs 도 산출물 셋을 안 읽고 이 계획만 볼 수 있다.
