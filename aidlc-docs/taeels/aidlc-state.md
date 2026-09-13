# Construction 상태 — taeels (최태양)

이 문서 루트는 **회차가 아니라 담당으로 갈린다** (`CLAUDE.md` 의 문서 루트 규약).
그래서 회차가 바뀌어도 주소가 안 바뀌고, 여기 회차 둘의 Construction 이 쌓인다.

```text
   v1-run-dhseo                obs · mcp        아래 「회차 v1」
   v3-run-harness-components   isolation 외 넷   아래 「회차 v3」
```

자기 브랜치에서 작업하고 PR 로 main 에 병합한다(게이트 초록 뒤). 산출물은 이
디렉터리 `aidlc-docs/taeels/` 아래.

---

# 회차 v1 — v1-run-dhseo

담당 유닛 — **obs**(W0 · CP1 · 토대) · **mcp**(W1 · CP5 · +CP7 도구).

## 유닛

- [x] obs 관측 API — CP1 · 의존 없음(토대). **먼저 병합돼야 W1(queue·mcp·ui)이 선다**
  - GET /v1/nodes · GET /v1/runs · getRun 확장(requires·as·chosen) · store 읽기 ·
    submitter 컬럼 · runctl Client.Nodes·Runs
- [ ] mcp — CP5 (+CP7 도구 asks.list·run.answer) · 의존 obs
  - internal/mcp(stdio JSON-RPC) · runctl mcp · 도구 열 · 글자 일치 passthrough

## 선행 · 공용

- 유닛 정본 `aidlc-docs/v1-run-dhseo/inception/application-design/unit-of-work.md`
- 의존·파일 행렬 `.../unit-of-work-dependency.md` · 게이트 `.../unit-of-work-story-map.md`
- 배정 `aidlc-docs/construction-roster.md` · 팩 `requirements/` · RE `aidlc-docs/inception/reverse-engineering/`

## 단계 진행 — obs

```text
   Functional Design      승인 2026-09-08T14:15:57Z (사용자가 다음 단계를 지시)
                          construction/obs/functional-design/ (domain-entities ·
                          business-logic-model · business-rules)
                          답 다섯 C·A·A·A·A 는 계획 2절 · 파장은 6절
                          1차 검토(하류 담당별) 7절 · 2차 검토(정합·정본·코드·게이트) 8절
                          실행으로 확정된 치명 둘을 고쳤다 — UpsertAdvert CTE 의 0행
                          NULL · coalesce 가 못 접는 jsonb null
   NFR Requirements       승인 2026-09-08 (사용자가 「이제 코드 작성해라」로
                          다음 단계를 지시했고 그것이 이 단계의 승인을 겸한다)
                          construction/obs/nfr-requirements/ (nfr-requirements ·
                          tech-stack-decisions) · 계획 construction/plans/
                          obs-nfr-requirements-plan.md
                          회차 계획은 이 단계를 SKIP 으로 승인받았고 사용자가
                          다시 들였다.  옮겨 적기가 아닌 자리 셋이 물음 셋이었다.
                          답 B·B·A — 셋 다 obs 안으로 들어왔다
                            1=B  무인증 읽기 셋에 요청 한도.  전역 · 초당 120 ·
                                 버스트 240 · 429 + Retry-After.  직접 짠다(의존 0)
                            2=B  runs_created_idx (created_at) 를 schema.sql 에
                            3=A  읽기 셋 셋에 Cache-Control: no-store
                          SECURITY 열다섯을 판정했다 — 조건부 둘(08 · 12)이
                          데모 무인증이라는 같은 사실에서 나온다
   NFR Design             건너뛴다 (회차 계획 SKIP).  답 1=B 가 남긴 「어떻게」는
                          Code Generation 계획 5절이 졌다 — 자료구조(토큰 버킷
                          하나)와 거는 자리(read 래퍼).  값은 nfr-requirements 3절
   Infrastructure Design  건너뛴다 (배포·인프라 변경 없음 · constraints §6)
   Code Generation        완료 2026-09-09.  계획 construction/plans/
                          obs-code-generation-plan.md · 요약 construction/obs/code/
                          code-summary.md
                          병렬 세 갈래를 파일 축으로 갈랐다 (겹치는 파일 0) —
                          A store+contract · B api+config · C runctl
                          CP0 전부 초록 · 커버리지 16개 패키지 미달 0 ·
                          internal/api 가 81.4% -> 82.9% 로 하한에서 멀어졌다
                          CP1 은 합성 함대로 부분 측정 — curl 두 줄이 초록이나
                          실 데몬 함대의 재측정은 게이트 집행자의 몫이다
```

**앞 판이 회차 계획과 어긋나 있었다** — 이 표가 NFR Requirements 와 NFR Design 을
「대기」로 적었는데 `aidlc-docs/v1-run-dhseo/inception/plans/execution-plan.md` 는
둘 다 **SKIP** 이다. 담당 문서는 여기서 정리했다. **회차 계획과 회차 상태 파일의
SKIP 줄을 고칠지는 진행자의 것이다** — 계획 7절의 표시 ①.

브랜치 `unit/obs`. 감사 로그는 `aidlc-docs/taeels/audit.md`.

## 다음

**obs 의 단계가 다 돌았다. 남은 것은 병합이다** (토대라 먼저 병합돼야 나머지
웨이브가 선다). 접점 `internal/store` · `internal/api/api.go` 는 진행자 직렬 병합.
**행렬 밖 파일이 넷으로 늘었다** — `internal/store/observe.go` ·
`internal/store/schema.sql` · `internal/contract/advert.go` ·
`internal/config/config.go`. 계획 6.1 · business-rules §7.5.
그중 `config.go` 는 단독이 아니라 **접점**이다 — demo-back 이 같은 스위치를 읽는다.
`internal/runctl/client.go` 도 「단독 obs」가 아니다 — mcp(W1)와 panel(W3)이 만진다.

**병합 조건이 정해졌다 (사용자 답)** — `scene-gates.md` §3 의 curl 두 줄 + CP0.
CP1 전체는 화면 다섯이 ui 몫이고 ui 는 obs 병합 뒤에 착수하므로 W0 에 초록이 될 수
없다. 이 부분 통과를 진행자가 승인한다. 상태 넷 중 넷째(`QUEUED`)는 CP2(W1)가 잰다.

**진행자에게 넘기는 것이 여섯이다** — 계획 §8.5. 무거운 셋: 답 4=A 가 `ADR-065`
2절의 인증 줄을 벗어난다(사용자가 A 유지 · 어긋남 기록) · 무인증 `GET /v1/nodes` 가
`label` 로 노드 소유자의 이메일 로컬파트와 호스트명을 낸다 · `submitter` 가
`enode-design` 에 0건이라 정본 개정의 주인이 없다.

**NFR Requirements 가 여섯을 더 남겼다** — `nfr-requirements.md` §6. 무거운 셋:
`429` 가 정본 코드 표(`mediator-api.md` 1.2 의 여덟)에 없다(다섯째 표시다) ·
SECURITY-11 의 첫 판정이 이 유닛에서 나므로 demo-back(W2)과 ui 가 그 표를 이어
써야 한다 · 전역 한도의 대가로 한 클라이언트가 나머지를 굶길 수 있다(IP 로 나누면
NAT 뒤의 관객 전체가 막혀 CP10 을 직접 깨므로 이쪽을 수락했다).

**답 셋이 만지는 파일을 넓혔다** — `internal/api` 에 한도 파일 하나가 신규로 생기고
`schema.sql` 의 변경이 셋이 된다(ALTER 둘 + `CREATE INDEX` 하나). 둘 다 이미
접점이므로 행렬이 넓어지지는 않는다.


---

# 회차 v3 — v3-run-harness-components

담당은 taeels 하나다 (회차 확인 질문 Q2 = B). 유닛 다섯을 **직렬로** 돈다.
회차 정본은 `aidlc-docs/v3-run-harness-components/` 이고 요구 팩은
`requirements/harness-components/` 다.

```text
   착수 순서   U1 isolation -> U2 contract-vocab -> U3 advert
               -> U4 sources -> U5 pack
```

| | 유닛 | 맡는 기능 | 닫는 게이트 | 선행 | 상태 |
|---|---|---|---|---|---|
| U1 | `isolation` | 3.1 · 3.2 의 최소 | CA1 | 없음 | **코드 완료 · CA1 대기** |
| U2 | `contract-vocab` | 3.5 | CA3 의 절반 | 없음 | **코드 완료 · CA3 앞 절반 초록 · 병합 대기** |
| U3 | `advert` | 3.3 | CA2 · CA3 완결 | U1 · U2 | **FD 계획 · 답 일곱 대기** |
| U4 | `sources` | 3.4 · 3.2 의 완성 | CA4 | U1 · U2 · U3 | 대기 |
| U5 | `pack` | 3.6 · 3.7 | CA5 · CA6 | U1 · U2 · U4 | 대기 |

**모든 유닛의 완료 조건에 CA0 가 들어간다** — `internal/enode` 커버리지 80% 를
유닛 단위로 집행한다 (`unit-of-work.md` 6절).

## 단계 진행 — U1 `isolation`

브랜치 `unit/isolation` (회차 브랜치에서 땄다 · `CONVENTIONS.md` 3.1).

```text
   Functional Design      승인 2026-09-12 (사용자가 「isolation 코드 써」로 다음
                          단계를 지시했고 그것이 이 단계의 승인을 겸한다).  산출물 셋을
                          construction/isolation/functional-design/ 에 냈다
                          (domain-entities · business-logic-model · business-rules).
                          계획과 답은 construction/plans/isolation-functional-design-plan.md

                          계획을 짓기 전에 decisions.md 6절 ⑱ 이 U1 에 남긴
                          실측을 먼저 돌렸다 — stream-json 의 사건 종류 다섯이다
                          (system · assistant · user · rate_limit_event · result).
                          그 실측이 문서의 전제 하나를 뒤집었다 — 「첫 줄은
                          system/init」이 참이 아니었다.  SessionStart 훅이 있으면
                          init 앞에 사건이 온다

                          답 다섯 A · C · B · A · A.  둘이 권장과 갈렸다 —
                          2=C 는 껍데기에 토큰 수를 싣고(usage 를 통째로 못 옮긴다.
                          안에 문자열 둘과 객체 하나가 있어 이름으로 넷만 집는다),
                          3=B 는 「첫 줄은 init」을 불변식으로 유지한다.
                          B 의 잔여 위험을 business-rules 6.3 이 이름으로 적었다 —
                          하네스가 init 앞에 사건을 내면 게이트의 head -1 이
                          엉뚱한 줄을 읽고, 최악은 mcp_servers 키가 없어
                          「비어 있다」로 오독되는 것이다

                          남은 것 하나였던 decisions.md 6절의 실측 행은
                          Code Generation 이 실었다 (6절 ⑳)

   NFR Requirements       SKIP (회차 실행 계획)
   NFR Design             SKIP
   Infrastructure Design  SKIP
   Code Generation        완료 2026-09-12.  계획 construction/plans/
                          isolation-code-generation-plan.md · 요약
                          construction/isolation/code/code-summary.md

                          병렬로 안 갈랐다 — 다섯 파일이 한 시그니처 변경을 함께
                          받아 갈래마다 컴파일되는 시점이 하나뿐이다 (계획 3절).
                          제품 파일은 행렬 그대로 다섯이고 행렬 밖은 새 시험 파일
                          셋뿐이다 (행렬 1절에 행으로 더했다)

                          CA0 이 전부 초록이다 — 시험 · vet · glyphscan · 포맷 ·
                          워킹트리 청결 · 크로스 빌드와 심볼 상한(6/50).  커버리지
                          미달 0 이고 전체 87.0% · internal/enode 85.6% 다

                          초록이 문다는 뜻은 아니라서 변이 넷을 돌렸다.  셋은
                          제대로 빨개졌고 **하나가 구멍을 찾았다** — selectLogs 를
                          원문으로 되돌려도 아무것도 안 빨개졌다.  logs_test.go 가
                          함수를 직접 부르고 배선을 안 쟀기 때문이다.  배선을 재는
                          시험을 더했다 (code-summary 3.1)

                          행렬이 예고한 worker_unix_test.go 는 안 빨개졌다 —
                          stubHarness 가 이미 type:"result" 봉투를 찍는다
```

## 이 유닛이 회차 밖으로 낸 것

```text
   decisions.md 6절   사건 종류 실측 행 ⑳ 을 실었다.  굳힌 것 셋과 뒤집은 전제
                      하나를 함께 적었고, --mcp-config 가 가변인자임도 --help 로 쟀다
   파일 행렬           새 시험 파일 셋을 1절에 행으로 더했다 (그 문서 6절 ②)
   scene-gates.md     안 고쳤다 — 답 3 = B 라 게이트 명령이 안 바뀐다
   짝 팩과의 접점      claude.go 의 Argv · runner.go.  이 팩이 먼저 전부 들어간다
```

## 다음 — U1

**남은 것은 CA1 과 병합이다.** CA1 은 사람이 재고 **집행자는 이 유닛을 구현하지
않은 사람**이다 (`scene-gates.md` 2절 머리) — 개인 MCP 서버와 계정 커넥터가 있는
기계에서, OAuth 노드와 게이트웨이 노드 둘 다에서 잰다. 초록인 뒤에 `unit/isolation`
을 PR 로 `main` 에 올린다 (`CONVENTIONS.md` 3.3).

**진행자에게 넘기는 것이 다섯이다** — `code-summary.md` 6절. 무거운 둘:
`decisions.md` 6.1 의 문자열 검사 스크립트가 **어느 유닛에도 배정되지 않았고**
(오늘은 사람이 손으로 돈다) · R3 과 R6 의 등급이 같은 논거 위에서 갈린다(8.1).

---

## 단계 진행 — U2 `contract-vocab`

브랜치 `unit/contract-vocab`. **회차 브랜치가 아니라 `unit/isolation` 에서 땄다** —
U1 이 아직 `main` 에 없고 `aidlc-state.md` 가 자동 병합이 안 되는 파일이라
회차에서 따면 U1 의 절과 U2 의 절이 파일 꼬리에서 각자 자라 부딪친다. 근거와
대가는 계획 6절. 코드는 안 부딪친다 — 행렬이 두 유닛의 파일 교집합을 0 으로 센다.

```text
   Functional Design      승인 2026-09-13 (사용자가 「승인」으로 닫았다).
                          계획과 답 일곱은 construction/plans/
                          contract-vocab-functional-design-plan.md · 산출물 셋은
                          construction/contract-vocab/functional-design/

                          계획을 짓기 전에 만지는 자리를 코드로 읽어 갈린 문장
                          셋을 찾았다 — ① features.md 3.5 의 「runctl schema
                          steps 가 새 agent 키를 저절로 낸다」가 거짓이다
                          (Step.Agent 가 map 이라 printFields 가 하위 키를 안
                          찍고 cmd/runctl 은 agentKeys 를 import 하지 않는다)
                          ② blob 이름 공간은 Run 하나에 평평해서 in.from 의 점은
                          이름의 글자다 ③ 그래서 scene-gates.md CA5 의 명령이
                          안 돈다 (첫 단계가 $OUT/pack 을 내는데 둘째가
                          in.from: ["fetch.pack"] 을 적는다)

                          행렬 밖 파일 하나가 이 유닛의 것이었다 —
                          internal/contract/planshape.go.  PlanShape 의
                          「agent 는 다섯 — nothing else」가 agentKeys 가 는 뒤
                          계획에게 거짓을 가르친다

                          답 일곱 A · A · A · A · A · A · B.  여섯이 권장이고
                          하나(7=B)가 GLOSSARY.md 를 이 유닛 밖으로 뺐다.
                          답 1=A 와 2=A 가 검증 자리를 Mediator 로 올렸다 —
                          agent.mcp · agent.pack 의 타입과, agent.pack 이
                          in.from 에 있는지를 contract.Validate 가 본다.
                          계획이 지은 단계도 같은 검사를 받는다
                          (checkplan.go:82 · expand.go:206)

                          답 2=A 가 CA5 의 값을 바꿨다 — 이름 불일치가 조용한
                          실패가 아니라 제출 400 이 된다.  고치지 않으면 그
                          게이트를 시작조차 못 한다.  그래서 답 6=A 로
                          scene-gates.md 까지 고친다 (승인 뒤 · 팩 문서다)

                          예시를 오늘 코드에 실측으로 걸었다 — 유일한 거절이
                          unknown field "mcp" 이고 그것을 뺀 같은 계약은
                          통과한다.  예시는 agentKeys 가 는 커밋에서 초록이 된다

   NFR Requirements       SKIP (회차 실행 계획)
   NFR Design             SKIP
   Infrastructure Design  SKIP
   Code Generation        완료 2026-09-13.  계획 construction/plans/
                          contract-vocab-code-generation-plan.md · 요약
                          construction/contract-vocab/code/code-summary.md

                          갈래를 안 갈랐다 — 제품 파일 다섯 중 넷이 한 패키지이고
                          예시가 agentKeys 와 같은 커밋이어야 해서 갈래가 서로를
                          기다린다 (계획 3절)

                          CA0 이 전부 초록이다 — 시험 · 커버리지 · vet · 포맷 ·
                          glyphscan · 크로스 빌드 · 심볼 상한 · 라우트 26 ·
                          워킹트리 청결.  미달 0 이고 전체 87.1% ·
                          internal/contract 89.3% · internal/enode 85.6%

                          CA3 의 앞 절반을 runctl lint 로 쟀다 (그 명령이
                          Validate 를 부른다) — 모르는 키의 허용 목록이 일곱으로
                          함께 늘고, agent.mcp 를 문자열로 적은 계약과
                          agent.pack 만 적은 계약이 문장으로 거절된다

                          변이 다섯을 돌려 다 빨개졌다.  U1 과 달리 구멍이 0 인
                          이유는 규칙이 전부 순수 함수 하나를 지나서다 — 배선이
                          한 자리이고 변이 ② 가 그 자리를 직접 잰다

                          **실측이 FD 의 주장 하나를 뒤집었다** — CA5 의 이름
                          불일치는 조용한 실패가 아니다.  Validate 에 in.from 의
                          정적 검사가 이미 있어 (contract.go:1109-1126) 오늘도
                          제출에서 거절된다.  게이트가 안 도는 것은 그대로 참이고
                          이유가 그 거절이다 — 문구가 팩을 안 가리켜서 게이트를
                          돌리는 사람이 팩 코드의 결함으로 읽는다.  R5 가 닫는
                          것은 in.from 을 아예 안 적은 경우 하나로 좁아졌다.
                          고친 자리 넷과 코드 주석 하나는 code-summary 5절
```

## 이 유닛이 회차 밖으로 낸 것

```text
   decisions.md 6절    실측 행 ㉑ — 열거 면 · blob 이름 공간 · CA5 를 한 행에
   scene-gates.md      CA5 의 명령을 고쳤다 (out: ["pack"] · in.from: ["pack"])
   파일 행렬            planshape.go 행.  만지는 파일 15 -> 16
   짝 팩과의 접점       0.  이 유닛은 runner.go 도 claude.go 도 안 만진다
```

## 다음 — U2

**남은 것은 병합이다.** CA3 의 앞 절반이 초록이고 (그 조각 전체는 U3 에서
초록이 된다 — `unit-of-work-story-map.md` 2.1) CA0 이 전부 초록이다. 사람이
실제로 띄워야 하는 게이트가 이 유닛에는 없다.

**병합 순서가 U1 과 엮인다** — 이 브랜치는 `unit/isolation` 위에 섰으므로
U1 이 먼저 `main` 에 들어가야 이 PR 의 차이가 U2 것만 남는다. U1 의 CA1 이
아직 사람 대기다.

**진행자에게 넘기는 것 다섯** — `code-summary.md` 7절. 무거운 셋:
`agent` 의 나머지 다섯 키는 타입 검사가 여전히 노드뿐이라 **비대칭이 남고**
(답 1=A 가 범위를 정했다) · 회차 문서 루트의 두 값이 낡았다(`decisions.md` 의
행이 스물하나이고 요약 표 밖 파일이 셋이다 — 그 루트는 회차 진행자의 것이라
안 고쳤다) · `GLOSSARY.md` 의 푼 말이 아직 없다.

---

## 단계 진행 — U3 `advert`

브랜치 `unit/advert`. **회차 브랜치가 아니라 `unit/contract-vocab` 에서 땄다** —
선행이 U1 · U2 둘이라 회차에서 따면 그 둘이 없는 나무 위에서 게이트를 재게 되고,
`aidlc-state.md` 가 자동 병합이 안 되는 파일이라 세 유닛의 절이 파일 꼬리에서
각자 자란다. 근거는 계획 6절.

```text
   Functional Design      승인 2026-09-13 (사용자가 「승인」으로 닫았다).
                          계획과 답 일곱은 construction/plans/
                          advert-functional-design-plan.md · 산출물 셋은
                          construction/advert/functional-design/
                          (domain-entities · business-rules · business-logic-model)

                          답 일곱 A · A · A · A · A · B · B.  다섯이 권장이고
                          둘이 갈렸다 — 6=B 가 mcp.<이름> 만으로는 광고하지
                          않게 하고(하네스가 없으면 물 것이 없다),
                          7=B 가 현황판에 새 키의 이름을 준다

                          6=B 가 완료 조건 하나와 어긋난다 — unit-of-work.md
                          3절이 「cheapAttrs · capabilities · Detector 에 diff 0」
                          인데 hasCapability 를 고치므로 capabilities 의 글자는
                          그대로이고 동작이 바뀐다.  business-logic-model 7절이
                          그 어긋남을 표로 적었다.  대가는 답 6 의 B 가 미리 적었고
                          사용자가 그것을 보고 골랐다

                          6=B 의 침묵을 R12 가 막는다 — mcp: 를 적었는데 능력이
                          하나도 안 서면 노드 로그에 사유를 낸다.  안 그러면
                          소유자가 함대에서 자기 노드가 빈 것만 보고 멈춘다

                          3=A 가 env 를 이름에서 이름으로 못 박고 허용목록에
                          ${이름} 참조로 낸다.  그 꼴을 하네스가 펴는지는 실측이
                          0 이라 Code Generation 이 잰다 — 안 펴면 env 를 안 쓰는
                          쪽으로 가고 「값은 파일에 안 실린다」는 어느 쪽이든 산다

                          2=A 가 파일 행렬 2.1 의 한 줄을 거짓으로 만든다 —
                          이 유닛이 U1 의 allowlistEntry 를 만진다.  7=B 가
                          internal/api/ui 의 0 을 깬다.  둘 다 승인 뒤에 행렬에 싣는다

                          계획을 짓기 전에 만지는 자리를 코드로 읽고 돌렸다.
                          갈린 자리 셋을 찾았다 —
                          ① decisions.md 2절의 「그 밖의 키는 그대로 허용목록에
                          옮긴다」가 오늘 코드로 성립하지 않는다.  yaml 이 모르는
                          키를 말없이 버리므로 옮길 맵이 애초에 안 생긴다.
                          그리고 U1 의 allowlistEntry 이음매 주석과 파일 행렬
                          2.1 의 「U3 은 U1 의 것을 안 고친다」가 같은 자리를
                          두고 서로 다르게 말한다
                          ② env 가 이름인지 값인지가 요구와 형식에서 갈린다 —
                          features.md 3.2 는 「값은 적지 않는다」인데 형식은
                          map[string]string 이고 allowlistEntry 가 통째로 옮긴다.
                          ADR-035 §4.4 의 예시에는 env 가 아예 없다
                          ③ 현황판이 새 키를 원문으로 낸다 (attributeName 의 표
                          밖이라) — 같은 사실이 실행 도구 = claude 와
                          harness.claude = 1 두 줄로 보인다

                          yaml 실측 열일곱을 돌려 검증 범위를 쟀다.  yaml 이
                          잡는 것은 종류뿐이고 일곱이 그대로 통과한다 —
                          command: 5 가 "5" 가 되고, 빈 서버 · 빈 이름 ·
                          command 와 url 이 둘 다인 선언이 다 선다

                          동작 중립의 근거 셋도 쟀다 — harnesses 가 하나라
                          break 를 걷어도 결과가 안 바뀌고, Detector 의 시계가
                          CA2 의 「탐지 주기 뒤」를 배선 0 으로 세우며,
                          hasCapability 가 mcp.<이름> 하나로 참이 되어
                          하네스 없는 노드가 광고를 낸다 (물음 6)

   NFR Requirements       SKIP (회차 실행 계획)
   NFR Design             SKIP
   Infrastructure Design  SKIP
   Code Generation        완료 2026-09-13.  계획 construction/plans/
                          advert-code-generation-plan.md · 요약
                          construction/advert/code/code-summary.md

                          갈래를 안 갈랐다 — Go 셋이 서로를 기다려 컴파일되는
                          시점이 하나뿐이다.  format.mjs 는 진짜 독립이지만
                          여섯 줄이라 갈래 비용이 이득보다 크다

                          CA0 이 전부 초록이다 — 시험 18 패키지 · 커버리지
                          미달 0(전체 87.2% · internal/enode 85.6% -> 86.0%) ·
                          vet · 포맷 · glyphscan · 크로스 빌드 셋 · 심볼 상한 ·
                          라우트 26 · ui 시험 75 · 워킹트리 청결

                          실측이 FD 의 미정 하나를 닫았다 — claude 2.1.266 이
                          mcp.json 의 env 에서 ${이름} 을 편다.  그래서 참조 꼴이
                          확정이고 「안 펴면 env 를 안 쓴다」는 대안이 닫혔다

                          CA3 의 뒤 절반을 합성 함대로 쟀다 — runctl capabilities
                          에 mcp.probe 와 harness.claude 가 나오고, requires 에
                          그 키를 적은 계약이 그 노드에만 가고, 없는 키가 422 다.
                          셋 다 코드 0 으로 닫혔다.  덤으로 희소성 정렬도 섰다 —
                          mcp 를 안 요구하는 계약은 흔한 노드로 간다

                          변이 다섯을 돌려 다 빨개졌다.  구멍 0 이다.  다만 옛
                          시험 둘이 확정 빨강이라 함께 고쳤다 (TestDetectEmpty 의
                          허용 키 목록 · allowlistEntry 의 env 단언).  둘 다
                          행렬 밖이라 행렬 3절에 행으로 더했다

                          시험 픽스처 하나가 검사의 한계를 드러냈다 — env 값
                          ghp_secret 이 환경변수 이름의 꼴을 만족해 통과한다.
                          FD 가 이미 적은 한계이고, 그것을 재는 시험을 따로 더했다
```

---

## 단계 진행 — U4 `sources`

브랜치 `unit/sources`. **회차 브랜치가 아니라 `unit/advert` 에서 땄다** —
선행이 U1 · U2 · U3 셋이라 회차에서 따면 그 셋이 없는 나무 위에서 CA4 를 재게
되고, `aidlc-state.md` 가 자동 병합이 안 되는 파일이라 네 유닛의 절이 파일
꼬리에서 각자 자란다. 근거는 계획 6절.

```text
   Functional Design      승인 2026-09-13 (사용자가 「승인, 코드 써」로 닫았다).
                          계획과 답 일곱은 construction/plans/
                          sources-functional-design-plan.md · 산출물 셋은
                          construction/sources/functional-design/
                          (domain-entities · business-rules · business-logic-model)

                          답 일곱 전부 A 다.  계획을 짓기 전에 만지는 자리를
                          코드로 읽고 claude 2.1.266 으로 돌려 갈린 자리 넷을
                          찾았다 —
                          ① 워크스페이스 .mcp.json 은 하네스가 스스로 쓰는
                          파일이라 우리 어휘에 없는 키 셋을 담는다 (type ·
                          headers · 값으로서의 env).  claude mcp add 가 실제로
                          쓴 파일을 읽어 쟀다
                          ② 그것을 MCPServer 로 받으면 셋을 잃거나 망친다 —
                          Extra 가 yaml 전용이라 type 과 headers 가 사라지고,
                          envRefs 가 값을 두 번 감싸 ${${X}} 가 되며, 종류 오류
                          하나가 파일 전체를 죽인다
                          ③ url 만 적힌 항목을 하네스가 말없이 버린다.
                          mcp_servers 목록에 이름조차 안 나오고 failed 로도
                          안 나타난다.  type 이나 command 가 있으면 선다.
                          그것이 ADR-035 §4.4 예시 그대로 적은 원격 노드 선언의
                          오늘 모양이라, 광고는 서고 서버는 안 열린다
                          ④ ${} 는 한 겹만 펴진다 — ②의 이중 감싸기가 오류가
                          아니라 조용한 손상인 이유다

                          답 2=A 가 Components.Servers 의 타입을 최종 허용목록
                          항목으로 올렸다.  출처마다 어휘가 다르므로 한 형식으로
                          둘을 못 담는다 — 노드 것은 allowlistEntry 로 짓고
                          워크스페이스 것은 원문 그대로 옮긴다.  번역하는 코드가
                          없으므로 ②의 손상 셋이 구조적으로 못 생긴다

                          답 1=A 와 4=A 가 만나 규칙이 하나로 줄었다.  4=A 의
                          「요청이 있을 때만 치명」이 1=A 아래서 조건문이 아니라
                          호출 자리로 보장된다 — 요청이 0 이면 파일을 아예 안 연다

                          답 3=A 가 type 을 채우는 한 줄을 들였다.  그것이
                          ③ 의 침묵을 막는 유일한 자리이고, 종류를 잘못 채운
                          경우는 failed 로 보인다 — 침묵이 아니라 실패다

                          한계 넷을 business-rules 10절이 이름으로 졌다.
                          무거운 것은 원격 인증이다 — credential 은 파일에 안
                          나가므로 선언만으로 원격 MCP 가 도는 경로가 이 회차에
                          없다.  소유자가 headers 를 Extra 로 적어야 돈다

   NFR Requirements       SKIP (회차 실행 계획)
   NFR Design             SKIP
   Infrastructure Design  SKIP
   Code Generation        완료 2026-09-13.  계획 construction/plans/
                          sources-code-generation-plan.md · 요약
                          construction/sources/code/code-summary.md

                          갈래를 안 갈랐다 — 제품 파일 셋이 한 타입 변경을
                          함께 받아 컴파일되는 시점이 하나뿐이다

                          CA0 이 전부 초록이다 — 시험 18 패키지 · 커버리지
                          미달 0(전체 87.3% · internal/enode 86.0% -> 86.4%) ·
                          스킵 0 · vet · 포맷 · glyphscan · 크로스 빌드 셋 ·
                          심볼 상한 · 라우트 26 · ui 시험 75 · 워킹트리 청결

                          실측을 제품 코드가 낸 파일로 했다 — resolveComponents
                          와 writeMCPAllowlist 가 쓴 mcp.json 을 claude 2.1.266 에
                          물리니 넷이 전부 mcp_servers 에 섰다.  R9(종류 채우기)를
                          안 넣었으면 gerrit 이 빠진 채로 초록이었다.  워크스페이스
                          항목의 headers 도 한 글자도 안 바뀌고 살아 나갔다

                          CA4 의 셋째 줄(없는 이름)은 코드로 닫혔다 — 스텁 하네스가
                          마커 파일을 안 남기는 것으로 「안 떴다」를 잰다.  앞 두 줄은
                          같은 시험이 그 단계가 실제로 쓴 허용목록을 $OUT 으로 받아
                          읽지만, 실 함대의 init 줄은 집행자의 몫이다

                          변이 여섯을 돌려 다 빨개졌다.  구멍 0 이다 — 셋은 순수
                          시험과 배선 시험이 함께 빨개졌다

                          실측이 계획의 문장 둘을 고쳤다 — 「종류 오류 하나가 파일
                          전체를 죽인다」가 원문으로 받는 판에서는 안 일어나고
                          (답 2=A 가 덤으로 닫았다), features.md 3.2 에는 종류가
                          이미 있었다.  FD 의 한 줄도 고쳤다 — 종류를 채우는 자리는
                          allowlistEntry 가 아니라 합친 뒤의 공용 자리다
```

## 이 유닛이 회차 밖으로 낸 것

```text
   decisions.md 6절   실측 행 ㉓ 넷을 한 행에
   decisions.md 2절   허용목록 행에 종류 한 줄
   features.md 3.2    종류가 type 키라는 줄 · 원문 그대로 옮긴다는 줄
   파일 행렬           1절 · 2.1 · 2.2 · 3절 · 4.1 의 다섯 자리
   scene-gates.md     안 고쳤다 — CA4 의 명령이 안 바뀐다
   짝 팩과의 접점       runner.go 하나.  Job 의 필드만 더했다
```

## 다음 — U4

**남은 것은 CA4 와 병합이다.** 실 함대에서 `runctl record` 로 푼 `logs/` 의
첫 줄을 읽는 것이 사람의 몫이고 집행자는 이 유닛을 구현하지 않은 사람이다.
병합 순서는 U1 · U2 · U3 뒤다 — 이 브랜치가 그 위에 섰다.

**진행자에게 넘기는 것 다섯** — `code-summary.md` 8절. 무거운 셋:
원격 노드 선언만으로는 인증이 안 실린다(정본 개정 후보다) · 깨진 워크스페이스
파일이 MCP 를 쓰는 단계를 전부 죽인다(답 4=A 의 대가) · Notes 가 노드 로그에만
남아 봉인을 읽는 사람은 겹침과 빠짐을 못 본다.
