# Construction 상태 — taeels (최태양)

이 문서 루트는 **회차가 아니라 담당으로 갈린다** (`CLAUDE.md` 의 문서 루트 규약).
그래서 회차가 바뀌어도 주소가 안 바뀌고, 여기 회차 둘의 Construction 이 쌓인다.

```text
   v1-run-dhseo                obs · mcp         아래 「회차 v1」
   v3-run-harness-components   isolation 외 넷    아래 「회차 v3 — harness-components」
   v3-run-transcript           transcript 외 일곱  아래 「회차 v3 — transcript」
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
| U1 | `isolation` | 3.1 · 3.2 의 최소 | CA1 | 없음 | **병합됨 (PR #33) · CA1 초록** |
| U2 | `contract-vocab` | 3.5 | CA3 의 절반 | 없음 | **병합됨 (PR #34) · CA3 초록** |
| U3 | `advert` | 3.3 | CA2 · CA3 완결 | U1 · U2 | **병합됨 (PR #35) · CA2 · CA3 초록** |
| U4 | `sources` | 3.4 · 3.2 의 완성 | CA4 | U1 · U2 · U3 | **병합됨 (PR #36) · CA4 초록** |
| U5 | `pack` | 3.6 · 3.7 | CA5 · CA6 | U1 · U2 · U4 | **CA5 · CA6 초록 · 병합 대기** |

**게이트 일곱이 전부 초록이다** (2026-09-15 · 진행자가 사내에서 `unit/pack`
브랜치로 집행했다). 유닛 넷은 그 앞에 병합됐고 — 「병합됨」이 서명이 아니었던
자리다 — 서명이 뒤늦게 붙었다. `unit/pack` 은 서명 뒤에 올린다
(`CONVENTIONS.md` 3.3 의 병합 지점). 서명 기록은 `audit.md` 의 CA0 ~ CA6 항목.

```text
   CA0   기계.  유닛 다섯 전부에서 초록
   CA1   OAuth 절반과 게이트웨이 절반 둘 다.  게이트웨이는 사내 기계가 냈다
   CA2   광고에 실린다 · 실행파일을 치우면 빠지고 사유가 남는다 ·
         되돌리면 5분 뒤 복귀 (-detect-every 의 기본값)
         S1 — 현황판 상세 패널에 「MCP 서버 · omab 있음」 ·
         제어판 탐지 능력(읽기 전용) 칸에 mcp.omab=1.
         현황판의 그 칩은 **상세 패널**에 있고 함대 격자의 카드 면에는 없다 —
         카드는 identity.mjs 의 고정 키 목록으로 짓고 그 목록에 mcp.* 가 없다.
         카드 면에 올리는 것은 이 회차의 파일 행렬 밖이다
   CA3   422(없는 키) · 400(모르는 에이전트 키) · 400(mcp 를 문자열로) · lint · dry-run
   CA4   셋 중 요청한 하나만 열린다 · 워크스페이스 것도 열린다 ·
         없는 이름은 하네스가 안 뜨고 그 사유로 실패한다
   CA5   pack:hello 가 skills 와 slash_commands 에 · probe4 가 failed 로 ·
         다이제스트 일치
   CA6   사내 MCP omab 이 connected · harness.pack 이 blob 과 다이제스트 일치
```

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

**남은 것은 CA1 의 게이트웨이 절반이다.** CA1 은 사람이 재고 **집행자는 이 유닛을
구현하지 않은 사람**이다 (`scene-gates.md` 2절 머리 · 회차는 진행자로 닫았다) —
개인 MCP 서버와 계정 커넥터가 있는 기계에서, OAuth 노드와 게이트웨이 노드 둘 다에서
잰다. `unit/isolation` 은 이미 `main` 에 있다 (PR #33 · 게이트 앞에 병합됐다).

**OAuth 절반을 합성 함대로 돌려 초록이다** (2026-09-14 · `audit.md` 의 CA1 항목).
개인 서버 일곱(claude.ai 계정 커넥터 넷 포함)이 붙은 로그인된 기계에서 `agent.mcp`
없는 단계를 돌렸고 `init` 줄의 `mcp_servers` 가 `[]` 다. 같은 단계가 `Not logged in`
없이 돌았고 산출물이 나왔다 — **두 겹이 동시에 섰다.**

**게이트웨이 절반은 사내에서 잰다.** 이 기계의 `~/.claude/settings.json` 에
`apiKeyHelper` 도 `env` 도 없어 `gatewayAuthFields()` 가 옮길 필드가 0 이고,
가짜 게이트웨이로 돌리면 API 호출에서 죽어 판정이 안 선다.

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

**CA4 의 세 줄을 합성 함대로 돌려 다 초록이었다** (2026-09-13 · `code-summary.md`
4절). Mediator 를 시험 DB 위에 띄우고 노드 하나를 `probe` · `probe2` 선언과
워크스페이스 `.mcp.json`(`probe3`)으로 세웠으며 하네스는 실물 `claude 2.1.266` 이다.
판정은 `runctl record` 로 푼 `logs/` 의 첫 줄이고, ①은 `probe` 하나 ·
②는 `probe3` · ③은 단계 FAILED 와 0 바이트 로그다.

**그 실측은 게이트의 서명이 아니다** — `scene-gates.md` 2절 머리가 집행자를
이 유닛을 구현하지 않은 사람으로 못 박는다. 남은 것은 다른 손이 같은 세 줄을
보는 일과 병합이다. 병합 순서는 U1 · U2 · U3 뒤다 — 이 브랜치가 그 위에 섰다.

**진행자에게 넘기는 것 다섯** — `code-summary.md` 8절. 무거운 셋:
원격 노드 선언만으로는 인증이 안 실린다(정본 개정 후보다) · 깨진 워크스페이스
파일이 MCP 를 쓰는 단계를 전부 죽인다(답 4=A 의 대가) · Notes 가 노드 로그에만
남아 봉인을 읽는 사람은 겹침과 빠짐을 못 본다.

---

## 단계 진행 — U5 `pack`

브랜치 `unit/pack`. **`main` 에서 땄다** — U1 ~ U4 가 PR #33 ~ #36 으로 전부
들어와 있어 U4 처럼 앞 유닛 브랜치에서 딸 이유가 없다. 회차의 Inception 산출물도
같은 나무에 있다. 근거는 계획 6절.

```text
   Functional Design      승인 2026-09-14 (사용자가 「64메가 이외 모두 a로 한다」로
                          닫았다).  계획과 답 여덟은 construction/plans/
                          pack-functional-design-plan.md · 산출물 셋은
                          construction/pack/functional-design/
                          (domain-entities · business-rules · business-logic-model)

                          답 여덟 전부 A 다.  값 하나만 사용자가 올렸다 —
                          PackLimits.MaxBytes 를 32 MiB 에서 64 MiB 로

                          계획을 짓기 전에 만지는 자리를 코드로 읽고 실물
                          claude 2.1.270 으로 다섯을 쟀다.  그 실측이 유닛의
                          뼈대를 바꿨다 —
                          ① 오늘의 플래그(--setting-sources "")로는 가짜 홈의
                          skills/ 와 agents/ 를 하네스가 아예 안 읽는다.  파일은
                          생기는데 init 줄의 skills 에도 agents 에도
                          slash_commands 에도 안 나타난다.  unit-of-work.md 5절의
                          「Instrument 가 <dir>/home/skills/ 를 편다」가 펴기만
                          하고 안 읽히는 자리였고, 그대로면 CA5 가 확정 빨강이며
                          원인이 팩 코드가 아니라 플래그다
                          ② 길이 둘이고 둘 다 선다 — --setting-sources user 는
                          CLAUDE_CONFIG_DIR 이 user 범위를 통째로 옮겨서 사람의
                          ~/.claude/skills 수십과 agents 둘이 하나도 안 샜다.
                          --plugin-dir <계장>/pack 은 --setting-sources "" 를 그대로
                          둔 채 실리고 plugin.json 이 없어도 되며 이름이
                          pack:<이름> 으로 namespace 된다
                          ③ 워크스페이스 .claude/skills/ 는 어느 경우에도 안 읽힌다 —
                          features.md 5절과 decisions.md 2절의 열린 미정이
                          「안 읽는다.  팩으로 나른다」로 닫힌다
                          ④ --plugin-dir 에 없는 경로를 주면 종료코드 0 에 stderr 도
                          없이 조용히 무시된다.  빠짐을 우리가 잡아야 한다
                          ⑤ 가짜 홈의 settings.json 이 user 설정과 --settings 로
                          두 번 실려도 Stop 훅은 한 번만 뛴다

                          답 1=A 가 격리를 안 건드리는 길을 골랐다 —
                          --plugin-dir 은 「아무것도 읽지 마라 + 우리가 지은 이
                          자리만」이라 --strict-mcp-config --mcp-config=<경로> 와
                          같은 문장이고, 팩이 없는 단계의 argv 가 오늘과 한 글자도
                          안 달라 CA1 의 서명이 그대로 산다.  대가는 스킬 이름이
                          pack:hello 가 되는 것이고 CA5 의 문장이 함께 바뀐다

                          답 3=A 가 가장자리를 runner.go 로 정했다 (claim.go 는
                          이 유닛의 파일 행렬 밖이다).  resolveComponents 는
                          검증을 통과한 *Pack 을 받아 필터와 충돌만 본다 —
                          「정하는 함수는 파일을 안 만진다」가 그대로 산다

                          답 4=A 의 64 MiB 는 받은 바이트와 푼 바이트 둘 다에
                          걸린다.  전송 상한 10 MiB 의 여섯 배라 gzip 을 투명하게
                          받아도(답 5=A) 압축률 6.4 까지가 그 안이다

                          답 6=A 가 노드 선언 충돌의 범위를 요청된 이름으로
                          좁혔다.  요청 안 한 이름은 어디에도 안 실리므로 뒤집을
                          것이 없고, 전부 보면 실행 위험 0 인 팩까지 죽인다.
                          CA5 의 충돌 줄에 agent.mcp 한 줄을 더해야 한다

   NFR Requirements       SKIP (회차 실행 계획)
   NFR Design             SKIP
   Infrastructure Design  SKIP
   Code Generation        승인 2026-09-14 (사용자가 「승인」으로 계획을 닫았고
                          단계 열셋을 그대로 돌렸다).  계획은 construction/plans/
                          pack-code-generation-plan.md · 요약은
                          construction/pack/code/code-summary.md

                          제품 파일 여섯 · 새 제품 파일 0 · 새 시험 파일 하나
                          (pack_test.go).  claim.go 는 소스 diff 0 이다 —
                          답 3=A 가 가장자리를 runner.go 로 정했기 때문이다

                          CA0 전부 초록.  internal/enode 커버리지가 표준 명령으로
                          87.0% 다 (U4 뒤 86.4% 에서 올랐다).  라우트 26 그대로

                          변이 일곱을 넣어 전부 빨개지는 것을 봤다.  다섯과
                          여섯과 일곱은 순수 시험과 배선 시험이 함께 빨개졌다

                          실측 하나를 더 했다 — 우리 코드가 지은 <계장>/pack 을
                          claude 2.1.270 이 --plugin-dir= 로 읽는다.  skills 에
                          pack:hello · agents 에 pack:helper · slash_commands 에
                          pack:hello · plugins 에 pack@inline · mcp_servers 에
                          probe4.  이 단계가 새로 잰 것은 등호 형태다 — 이 자리
                          바로 뒤에 --strict-mcp-config 가 따라붙는데 가변인자면
                          그것을 삼킨다.  안 삼켰다

                          계획에 없던 것을 둘 더했다 — ① command 도 url 도 없는
                          팩 항목의 거절 (U4 가 워크스페이스에서 이미 거절한 같은
                          실패이고, 하네스의 침묵이 출처를 안 가린다)
                          ② Notes 복사를 걸음 0 의 오류보다 앞으로 (거절과 함께
                          규약 밖 항목의 이름이 사라지면 안 된다)
```

## 이 유닛이 회차 밖으로 낸 것

Code Generation 이 실었다 (U4 와 같은 자리다).

```text
   decisions.md 6절    실측 행 ㉔ — 위 다섯에 등호 형태까지 한 행에
   decisions.md 2절    워크스페이스 .claude/skills 행이 값이 됐다 ·
                       팩의 형식 행에 gzip 과 상한
   features.md 5절     미정이 둘에서 하나로
   features.md 3.6     펴는 자리가 <계장>/pack 과 --plugin-dir= 로
   unit-of-work.md 5절  claude.go 줄 · 완료 조건의 「판정」이 「확인」으로
   scene-gates.md      CA5 의 표 한 줄과 3절 명령 넷
   component-methods.md 2.1 의 Components.Servers(U4 가 남긴 낡은 줄) ·
                       Pack.MCP · PackFile.Mode · PackLimits 의 값
   파일 행렬            새 시험 파일 행 · resolve_test.go 의 U5 칸 ·
                       2.1 · 2.2 · 2.3 의 U5 칸
```

## 다음 — U5

**남은 것은 병합이다.** CA5 와 CA6 이 2026-09-15 에 초록이 됐다 — 진행자가
사내에서 `unit/pack` 브랜치로 집행했고, CA5 는 `pack:hello` 가 `skills` 와
`slash_commands` 에 서고 `probe4` 가 `failed` 로 나타나며 다이제스트가 맞는 것을,
CA6 은 사내 MCP `omab` 이 **connected** 로 뜨고 `harness.pack` 이 같은 Run 의
blob 과 맞는 것을 냈다.

**CA6 이 실물 MCP 로 섰다는 것이 값이다.** 스크래치 게이트들은 가짜 서버(`true`)라
`mcp_servers` 에 `failed` 로 나타나는 것이 판정 재료였는데, 사내에서는 뜨기만 하는
것이 아니라 MCP 로 답했다. 그리고 `SHA256` 을 **받은 바이트**의 것으로 정한 U5 의
판단(봉인을 읽는 사람이 blob 을 받아 `sha256sum` 으로 맞출 수 있어야 한다)이
실물 대조로 확인됐다.

`unit/pack` 을 PR 로 `main` 에 올린다 (`CONVENTIONS.md` 3.3).


---

# 회차 v3 — v3-run-transcript

담당은 taeels 하나다 (회차 질문 3 = B). 유닛 **여덟**을 **웨이브 다섯**으로 돈다.
회차 정본은 `aidlc-docs/v3-run-transcript/` 이고 요구 팩은
`requirements/transcript/` 다. Inception 이 2026-09-15T08:32:42Z 에 닫혔다.

## 착수 배치 — 여덟 직렬을 다섯 웨이브로

회차의 `unit-of-work-dependency.md` 3절은 착수 순서를 **여덟 직렬**로 적었다.
그 문서가 근거로 댄 것은 의존이 아니라 **한 손**이다 — 「한 손이므로 웨이브는
병렬 기회가 아니라 순서의 하한이다」. 사용자가 병렬을 지시해
(2026-09-15T08:33:10Z 「병렬로 돌릴 수 있으면 돌려」) 그 전제를 걷고 의존 행렬과
파일 행렬에 다시 댔다.

**가르는 기준이 둘이다** — 코드 의존과 **같은 파일**. 둘째가 없으면 의존 0 인
U1 과 U2 를 같은 웨이브에 넣게 되는데 둘 다 `runner.go` 를 만진다.

```text
   W-a   U1 transcript   병렬  U3 progress-store    의존 0 · 파일 겹침 0
   W-b   U2 node-stream  병렬  U4 log-api           enode 대 api.  **CB0**
   W-c   U5 panel-live   단독                       **CB1**
   W-d   U6 panel-past   병렬  U7 chunk-push        panel·runctl 대 enode
   W-e   U8 fleet-card   단독                       **CB4 · CB6**
```

**U5 를 단독으로 두는 것이 이 배치의 값이다.** CB1 은 뒤의 셋(U6 · U7 · U8)이
전부 딛는 게이트다. 그것을 보기 전에 셋을 지으면 빨갰을 때 되돌릴 것이 셋이 되고,
그것이 앞 팩의 CP6 이 눈을 늦게 떠서 이 팩이 생긴 바로 그 실패다. **병렬은
되돌릴 것이 안 느는 자리에서만 쓴다.**

**W-a 가 U1 · U2 가 아니라 U1 · U3 인 이유**가 파일 행렬 2절이다 — U1 이
`runner.go` 에서 셋을 덜어내고 U2 가 그 뒤에 tee 를 잇는다. 같은 웨이브에 넣으면
U2 가 U1 의 변경 위에서 재작업한다.

## 유닛

| | 유닛 | 맡는 기능 | 지는 게이트 | 선행 | 웨이브 | 상태 |
|---|---|---|---|---|---|---|
| U1 | `transcript` | FR-3 | (코드만) | 없음 | W-a | **병합됨 (PR #40)** |
| U2 | `node-stream` | FR-1 · FR-2 | (코드만) | U1 (파일) | W-b | **병합됨 (PR #43)** |
| U3 | `progress-store` | FR-5 (med) | (코드만) | 없음 | W-a | **병합됨 (PR #41)** |
| U4 | `log-api` | FR-6 · FR-5 (api) | **CB0** | U1 · U3 | W-b | **병합됨 (PR #44) · CB0 초록** |
| U5 | `panel-live` | FR-4 (절반) | **CB1** | U1 · U2 | W-c | **도는 중** |
| U6 | `panel-past` | FR-4 (나머지) | **CB2** | U4 · CB1 | W-d | 대기 |
| U7 | `chunk-push` | FR-5 (노드) | **CB3** | U2 · U4 · CB1 | W-d | 대기 |
| U8 | `fleet-card` | FR-7 | **CB4** · **CB6** | U4 · CB3 | W-e | 대기 |

**NFR Requirements 를 도는 유닛이 여섯이고 스킵이 둘이다** (U2 · U6). 회차 계획이
「새 표면을 만드는 유닛만 돈다」로 걸었다. **N1 은 U4 가 · N2 는 U3 가 진다.**

## 선행 · 공용

- 유닛 정본 `aidlc-docs/v3-run-transcript/inception/application-design/unit-of-work.md`
- 의존 `.../unit-of-work-dependency.md` · 파일 행렬 `.../unit-of-work-file-matrix.md`
- 게이트 사상 `.../unit-of-work-story-map.md` · 요구 `.../requirements/requirements.md`
- 팩 `requirements/transcript/` · RE `aidlc-docs/inception/reverse-engineering/`

## 단계 진행 — W-a (U1 `transcript` · U3 `progress-store` 병렬)

브랜치 `unit/transcript` · `unit/progress-store`. 둘 다 회차 브랜치에서 땄고
**worktree 를 갈라 동시에 돌렸다** (`/home/sunny/enode-wt/`). 앞선 회차가
브랜치를 앞 유닛 위에 쌓은 것과 다르다 — W-a 는 파일 교집합이 0 이라 쌓을
이유가 없다.

```text
   Functional Design      승인 2026-09-15T10:45:00Z (사용자 「승인」)
                          계획 둘 · 산출물 여섯.  커밋 edda804 · ac3acac (Part 1) ·
                          d20c26c (U3) · adc93d0 (U1)
                          물음 스물하나 + 막힌 뒤 낸 둘 = **스물셋.  전부 닫혔다**
                          답 — 스물하나가 A (사용자 「권장대로」) · 둘이 B

                          **둘 다 계획을 짓기 전에 코드를 읽었고 문서가 코드와
                          갈린 자리를 여덟씩 찾았다.** 진행자가 표본 일곱을 코드에
                          대고 확인했고 일곱 다 맞았다

                          무거운 셋 — ① U1 이 옮길 범위가 문서의 셋이 아니라
                          **일곱 + 타입 둘**이다 (components.md 와 unit-of-work.md
                          가 함께 틀렸다) ② `go list -deps` 가 시험 임포트를 안
                          보므로 경계 검사를 그 명령으로만 세우면 시험이 금지된
                          패키지를 임포트해도 초록이다 ③ N2 의 오늘 값이
                          「상한이 없다」가 아니라 **「지우는 코드가 0」**이다

                          **U3 의 모순 검사가 막는 것 하나를 찾아 멈췄다** —
                          Q3 = A 와 Q4 = A 를 합치면 진행 파일 안에 상한 도달의
                          흔적이 0 이라 GET 이 못 가른다.  NC-4 와 US-7 이 그
                          자리에서 죽는다.  사용자가 상한 값(10 MiB)을 묻고
                          표시 줄을 골랐다 (물음 10 = B · 11 = B)

                          같은 검사가 앞서 적은 구멍 하나를 **지웠다** —
                          `Records` 가 nil 이면 `needRecords` 가 503 을 내므로
                          진행 트리가 애초에 안 생긴다.  N2 의 구멍은 **둘**이다

                          **교차 검사가 하나 더 잡았다** — 같은 필드를 U3 은
                          상한으로 U1 은 총 길이로 정의했다.  상한이 맞고 취향이
                          아니라 기계로 갈린다: 표시 줄이 총 길이에 드는 이상
                          **자기가 든 총 길이를 담을 수 없다.**  U1 을 고쳤다

   NFR Requirements       승인 2026-09-15T12:45:00Z (사용자 「승인」).  둘 다 돌았다
                          계획 둘 · 산출물 넷.  커밋 ab819f5 · bd8cc0f (Part 1) ·
                          5e9219b (U3) · 8fc8886 (U1) · b58906f (회차와 팩)
                          물음 일곱 전부 A (사용자 「권장대로」) · 모순 0 ·
                          **SECURITY 미준수 0 · 빈 칸 0**

                          **N2 가 값을 얻었다** — 진행 파일의 **마지막 쓰기**로부터
                          6시간.  상수다.  집행은 Reap() 안이고 새 타이머가 0 이다.
                          회차의 미결 「① 에 상한이 없다」가 여기서 닫혔다

                          **봉투를 다시 계산했더니 기대와 달랐다** — 6시간이 자르는
                          것은 노출 시간이고 디스크가 아니다.  Run 이 그보다 짧게
                          돌면 최악 10 GiB 가 그대로 선다.  없앤 것이 아니라
                          받아들인 것이라 규칙으로 적었다

                          **U3 가 자기 판정을 교정했다** — SECURITY-14 의 넷 중
                          디스크 용량을 말하는 줄이 0 이다.  01 은 준수로 안 굳혔다:
                          6시간이 닫은 것은 「기간에 값이 없다」이고 「평문이다」는
                          그대로다

                          **U1 이 자기가 FD 에서 만든 갈림을 찾아 기록했다** —
                          팩은 「200자」인데 FD 가 「200 바이트」로 적고 「값은 팩의
                          것」이라 달았다.  한글은 3바이트라 3배 다르고, 바이트로
                          자르면 룬이 쪼개져 encoding/json 이 U+FFFD 로 바꾼다.
                          숫자는 팩 · 단위는 U1 로 갈라 적었다

                          **교차 검사 0** — U3 의 NFR 이 파서 표면을 안 건드린다
   NFR Design             승인 2026-09-15T13:35:00Z (사용자 「승인한다. 코드 쓰자」)
                          **물음 0** 으로 둘 다 산출물까지
                          커밋 5cf1571 (U1) · ef4d766 (U3)
                          패턴 다섯 중 W-a 가 셋을 진다 — U1 이 ④ · U3 가 ①의
                          찍는 쪽과 ⑤.  ② ③ 은 U7 · U8 의 것이고 **안 지는 것도
                          이름으로 적었다**

                          U1 — 규율이 깨지는 길 일곱 중 **여섯을 기계가 막고
                          하나는 사람이 막는다**.  이음매 하나를 박았다:
                          enode.capped 는 짓는 쪽이 U3 이라 왕복 시험이 못 잡는다
                          U3 — 설계가 순서 하나를 새로 더했다 (**개행 보장**).
                          없으면 표시 줄이 반쪽에 붙어 Capped 가 영영 참이 안 되고
                          표시 줄이 무한히 쌓인다
                          U3 의 실측이 회차 계획을 고쳤다 — Seal 은 tar 를 안 짓는다
   Infrastructure Design  SKIP (회차 계획)
   Code Generation        Part 1 (계획) 승인 2026-09-15T14:20:00Z (사용자 「넘어가지」)
                          Part 2 (생성) 2026-09-16.  승인 2026-09-16T01:08:12Z
                          (사용자 「병합하고 w-b 가자」).  **코드가 섰다**
                          U1 커밋 넷 c95d5bf · 583ebf6 · 0510edb · 864d5e0
                          U3 커밋 다섯 bdb13a3 · 26bf7b4 · efc0042 · ebb7d7a · 73fdeb3
                          체크박스 U1 97 중 94 · U3 122 중 121 (남긴 것은
                          채우면 거짓 초록인 자리들이다)

                          **구독 만료로 한 번 끊겼다** — 재개 지점을 에이전트의
                          말이 아니라 저장소 상태로 쟀다.  커밋 안 된 것을 안
                          버렸고 U1 790 줄 · U3 1,507 줄이 그대로 살았다

                          진행자가 게이트를 다시 쟀다 — 전체 시험 초록 ·
                          커버리지 미달 **0** (U1 87.5% · transcript 94.4%.
                          U3 87.4% · record 84.4%) · 라우트 17 · gofmt 빔 ·
                          diff 0 경로 전부 0 · 파일 교집합 0

                          **이음매를 진행자가 실제로 왕복시켰다** — W-a 가 병렬의
                          대가로 남긴 자리다.  U3 의 cappedMark 41 바이트를 U1 의
                          Parse 에 먹였고 KindCapped 와 Bytes 10485760 이 섰다

                          실측이 계획을 뒤집은 것 셋 — 물음 1 의 답(content 가
                          배열로 온다.  A 로 지었고 실측이 받친다) · U3 의 빨간
                          시험은 시험이 틀렸다(제품 0 줄) · 최악 봉투가 112 MB 가
                          아니라 **2.3 GB** 다 (개행만 든 1 바이트 줄도 사건이다)

                          진행자가 고친 것 둘 — U1 주석의 장식 문자 하나(규약 1.2.
                          에이전트는 0 이라 보고했다) · probe.lock 되돌리기
                          커밋 9d907da · c4e6eec (U1) · f962d26 · 53ef404 (U3)
                          Step 열아홉씩 · 체크박스 아흔여섯(U1) · 백스물둘(U3)

                          **U1 이 불변식 F2 가 거짓인 것을 찾았다** — tool_use 의
                          Text 는 RawMessage 를 다시 마샬한 것이라 compact 만
                          지난다.  json.Marshal(json.RawMessage) 가 잘못된 UTF-8 을
                          그대로 낸다 (진행자가 go1.26.6 으로 재현).  F2 에 조건을
                          달아야 Part 2 가 첫 변이에서 안 멈춘다.  값은 안 바뀐다

                          진행자 표본 검증이 유닛마다 하나씩 되돌렸다 —
                          U1 의 셈(표는 여덟인데 글자가 일곱.  앞 문서에 틀린 값이
                          두 벌) · U3 의 게이트 7(스물한 자리 중 여섯은 렌더링되는
                          자리라 위반이 아니다.  위반은 열다섯 · 파일 여덟)

                          **U3 가 진행자의 셈을 되돌렸다** — 「여덟 중 일곱이 행렬
                          밖」이 아니라 여덟 전부가 밖이다.  짐작으로 안 따랐다

                          기준선 실측 — go build · vet · test 전부 초록.  단 postgres
                          가 서야 한다 (셋이 URL 없이 실패).  scripts/testdb.sh
```

## 이 웨이브가 회차 밖으로 낼 것 — 진행자의 몫

**U3 의 Code Generation 전에 서야 하는 둘이 있다.** 기준선인 채로 병합하면
그 자리에서 빨개진다.

```text
   때가 박힌 것   unit-of-work-file-matrix.md 5절   internal/store diff 0 이 거짓이 된다
                 execution-plan.md 6절 품질 게이트 2  같은 이유.  contract 는 0 그대로
                 unit-of-work-file-matrix.md 1절   U1 의 「경계 검사 두 줄」이 넷이다
                 component-methods.md 1.2          map[string]any -> Fields

   그 밖          components.md 1절 · application-design.md D1 의 줄 번호와 D4 의 근거
                 requirements.md 5.1 (testdata) · FR-3 의 사건 종류 · 2.4 의 낡은 인용
                 user-stories.md US-4 의 확인 글자
                 팩 transcript/decisions.md 2절 — raw 정의와 사건 종류
                 unit-of-work.md U1 절의 「셋」
                 GLOSSARY.md 의 CB · N1 · N2 — **푼 말을 아무도 안 짐작했다**
```

## 이 웨이브가 배운 것 — 병렬의 대가 하나

**유닛 경계를 넘는 답의 조합은 어느 한 유닛의 모순 검사도 못 본다.** AI-DLC 의
Functional Design Step 5 는 그 유닛의 답끼리만 댄다. 같은 날 두 번 났고 **두 번
다 두 유닛이 각자 모순 0 이었다.** 병렬로 돌리면 **웨이브를 닫는 자리에 유닛
사이를 대 보는 검사가 따로 있어야 한다** — 이 회차는 진행자가 그것을 졌다.

**한 벌로 못 만드는 값은 맞대는 절차가 따로 있어야 한다.** `enode.elided` 는
짓는 함수가 같은 패키지라 왕복 시험이 잡지만 `enode.capped` 는 찍는 쪽이 U3 ·
읽는 쪽이 U1 이라 한 패키지 시험으로 안 잡힌다. Code Generation 이 맞댈 것을
값 이름까지 적어 뒀다 (`business-rules.md` 16.1).

## W-a 가 닫혔다 — 병합 2026-09-16

승인 뒤 **합친 나무에서 게이트를 다시 쟀다.** 유닛 둘의 값은 각자의 브랜치에서 잰
것이고, 합치면 같은 값이라는 보장이 없다 — 그 보장을 안 믿고 쟀다.

```text
   합본        origin/main + v3-run-transcript + unit/transcript + unit/progress-store
               충돌 0 · 세 갈래의 파일 교집합 0

   초록인 것    build · vet · gofmt · glyphscan(109 파일) ·
               시험 19 패키지 전부 · **스킵 0**
               커버리지 정본 명령으로 19 패키지 **미달 0** — 전체 87.5%
               transcript 94.4% · record 84.4% · store 82.5% · enode 86.7%
               라우트 17 그대로 (18 은 U4 의 몫 · CB0)
               windows/amd64 크로스 빌드 · net/http T 6 · crypto/tls T 1

   PR 셋       #39 회차 브랜치 -> #40 U1 -> #41 U3.  이 순서로 올렸다
               회차가 먼저다 — 유닛 정의와 파일 행렬이 거기 있다
```

**앞 단계가 「못 갈랐다」고 적은 자리를 다시 쟀다** — `internal/api` 의 deadlock
(40P01). 합본에서 **일곱 번 다 초록이고 재현 0 이다.** 재현이 0 인 것은 없어진 것과
다르므로 사라졌다고 안 적는다. 다음 웨이브가 `internal/api` 를 만지므로 (U4) 거기서
다시 본다.

**`cmd/enodectl/probe.lock` 이 시험을 돌 때마다 바뀐다** — CP0 의 「시험이 추적 파일을
안 고친다」를 그대로 깬다. 이 유닛들이 만든 것이 아니고 (둘 다 `cmd/enodectl` 을 안
만진다) 앞선 회차들도 겪었다. 되돌리고 **기준선의 성질로 적는다** — 고치는 것은 이
회차의 파일 행렬 밖이다.

## 다음 — W-b (U2 `node-stream` · U4 `log-api` 병렬)

```text
   U2 node-stream   FR-1 · FR-2   게이트 (코드만)   선행 U1 (파일)
   U4 log-api       FR-6 · FR-5   **CB0**          선행 U1 · U3
```

**둘 다 W-a 의 코드를 딛는다** — U2 는 U1 이 `runner.go` 에서 덜어낸 자리에 tee 를
잇고, U4 는 U1 의 파서와 U3 의 진행 파일을 둘 다 읽는다. 그래서 브랜치를 회차
브랜치가 아니라 **병합된 `main` 에서 딴다** — 회차 브랜치는 코드를 0 줄 싣는다.

**가르는 축은 enode 대 api 다** — U2 가 `internal/enode`, U4 가 `internal/api`.
파일 행렬이 교집합을 0 으로 뒀는지가 착수 전에 볼 첫 자리다.

**NFR Requirements 는 U4 만 돈다** (U2 는 회차 계획 SKIP — 새 표면을 안 만든다).
**N1 을 U4 가 진다.**

**CB0 이 이 웨이브에서 처음 값을 얻는다** — `grep -c 'mux.HandleFunc' internal/api/api.go`
가 17 에서 **18** 이 된다. 오늘 17 인 것을 합본에서 쟀다.

## 단계 진행 — W-b (U2 `node-stream` · U4 `log-api` 병렬)

브랜치 `unit/node-stream` · `unit/log-api`. 둘 다 **병합된 `main` 에서 땄다** —
W-a 와 다르다. 회차 브랜치가 아닌 이유는 둘 다 W-a 의 코드를 딛기 때문이다.
worktree 를 갈라 동시에 돌린다 (`/home/sunny/enode-wt/`).

```text
   Functional Design      Part 1 (계획) 2026-09-16.  커밋 77eb3d2 (U2) · f8c7b74 (U4)
                          Part 2 (산출) 승인 2026-09-16T02:05:00Z (사용자 「권장대로」)
                          커밋 b115b25 (U2) · 0d9f14c (U4).  산출물 여섯
                          물음 열다섯 (U2 일곱 · U4 여덟) **전부 A · 갈린 답 0**

                          **둘 다 계획을 짓기 전에 코드를 읽었고 문서가 코드와
                          갈린 자리를 다섯 찾았다** — U2 셋 · U4 둘

                          무거운 셋 — ① component-methods 대로 Decode 안만
                          고치면 사건이 흐르는 시점이 안 바뀐다.  부르는 자리가
                          runner.go:209 로 cmd.Run 뒤다 ② 봉인된 logs/ 를 여는
                          공개 겉면이 internal/record 에 0 이고 safe() 가 비공개다
                          ③ 데모 모드의 read 래퍼가 무인증이라 requirements 5.4
                          잔여 ② 의 「토큰 하나」 전제가 데모에는 없다

                          **모순 검사가 유닛마다 하나씩 막았다.**
                          U2 — 답 1 = A 와 FR-1 을 합치면 배출이 두 벌이다.
                          한 단계의 사건이 두 번 난다.  U7 이 업로더를 물리면
                          같은 바이트가 두 번 올라간다.  흘리는 쪽이 낸다 —
                          runner 가 Decode 에 넘기는 emit 을 no-op 으로 둔다
                          U4 — 답 6 = A 와 as=events 를 합치면 폴링이 깨진다.
                          사건 배열이라 읽은 바이트를 셀 수 없다.  **상한에 걸릴 때
                          마지막 개행에서 끊는다** — 헤더를 다섯째로 안 늘려도 닫힌다

                          **교차 검사가 하나 더 잡았다** — Source: progress 에
                          Bytes 0 이 나가는 길이 셋이다 (안 왔다 · 6시간 쓸기가
                          걷었다 · 봉인의 창).  셋이 한 값으로 나가고 셋이 다른
                          문서에 살아 어느 유닛의 모순 검사도 못 본다

                          **갈릴 자리 넷을 이름으로 적었다** — 어휘(정본 하나로 맞다) ·
                          총 길이(Ring.Total 대 Progress.Total 이 같은 단어 다른 값) ·
                          시도(gen 과 attempt 가 같은 수가 아니다) ·
                          상한의 단위 셋 (512 KiB · 10 MiB · 1 MiB)
   NFR Requirements       **SKIP** (2026-09-16T02:20:00Z · 사용자 「nfr 단계를
                          모두 스킵하고 다음으로 간다」).  **회차 계획과 갈린다** —
                          계획은 EXECUTE 이고 U4 가 N1 을 지도록 걸려 있었다.
                          어긋남으로 적는다.  U2 는 원래 SKIP 이었다
   NFR Design             **SKIP** (같은 지시)
   Infrastructure Design  SKIP (회차 계획)
   Code Generation        Part 1 (계획) 승인 2026-09-16 (사용자 「승인. 넘어가자」)
                          커밋 ec8d7c5 (U2 · Step 열둘 · 체크박스 54) ·
                          24b531f (U4 · Step 열셋 · 체크박스 58).  **물음 0**
                          Part 2 (생성) 승인 2026-09-16T05:40:00Z (사용자
                          「w-b pr둘고 메인에 올리고 w-c 진행」)
                          커밋 0b66780 (U2) · 9685f52 (U4)
                          체크박스 **U2 54 중 54 · U4 58 중 58**

                          진행자가 합본에서 다시 쟀다 — 충돌 0 · 파일 교집합 0 ·
                          시험 19 패키지 초록 · 스킵 0 · 커버리지 미달 0 (87.5%) ·
                          **라우트 18 (CB0)** · gofmt 빔 · glyphscan 111 파일 0

                          **이음매를 웨이브 닫는 자리에서 맞댔다** — 노드가 찍는
                          종류와 선 위로 나가는 종류가 같은 글자다.  여섯이 같고
                          그중 capped 는 찍는 쪽(U3) · 읽는 쪽(U1) · 흘리는 쪽(U2) ·
                          내는 쪽(U4) 넷을 지나 같은 글자로 나온다

                          실측이 계획을 고친 것 여덟 (U2 다섯 · U4 넷 · 하나 겹침).
                          무거운 셋 — ① Decode 에 no-op 을 넘기면 final 이 사라진다
                          (봉투를 읽고 나는 판정이라 배출기가 못 낸다.  거르개가 답)
                          ② transcript.Parse 는 개행 없는 줄을 안 읽는다.  두 자리
                          다 개행을 붙여야 한다 ③ 태그를 달 타입이 다섯이다 —
                          「안 단다」의 근거였던 대소문자 관용은 **읽을 때만** 있다

                          변이 열하나 전부 빨강.  둘이 값이 있었다 — U2 의 다섯째는
                          실패가 아니라 **멈춤**으로 빨갰고(대기열 상한을 빼면 Write 가
                          영영 막힌다), U4 의 넷째는 **처음에 살아남았다**.  폴링이
                          규칙대로 돌면 그 갈래를 한 번도 안 밟기 때문이고, 밟는 길
                          둘을 재는 시험을 따로 짓고 나서야 빨개졌다

                          **사고 하나** — 변이를 git checkout 으로 되돌리다 커밋 안 된
                          runner.go 를 지웠다.  백업에서 되살렸다.  그 사고가 시험의
                          구멍을 드러냈다: 배출기가 없는데도 수를 세는 시험이
                          초록이었다.  **수를 세는 시험은 시점을 못 잰다**

                          앞 팩의 시험 하나를 뒤집었다 — 링이 한 바이트도 안 받는
                          것을 재던 시험(decisions.md 6절 ⑲)을 본문 글자까지
                          확인하는 것으로 바꿨다
                          CB0 의 앞 값을 합본에서 쟀다 — 라우트 17.  U4 가 18 로 만든다
                          **NFR 스킵이 남긴 값 둘을 이 계획이 졌다** (U4 계획 0절).
                          1 MiB 는 internal/api/log.go 의 상수다 — 조절 손잡이가
                          아니라 보호라 설정 키를 안 만든다.
                          **N1 은 안 닫고 봉투를 산수로 남겼다** — 미는 쪽 S/2 ·
                          당기는 쪽 V x C / 2 · 천장은 데모의 전역 한도 120 req/s.
                          아픈 자리가 그 천장이다: 이 폴링 라우트가 GET /v1/nodes ·
                          GET /v1/runs 와 한 바구니를 나눠 쓴다 (ratelimit.go:19-20).
                          v1 의 obs 가 이미 적은 잔여에 이 회차가 폴링을 더한다.
                          한도는 안 바꾸고 **U8 에 「보이는 카드만 폴링한다」를 넘긴다**

                          계획이 실측으로 찾은 것 둘 — harness.go 가 파일 행렬에
                          없다 (EventKind 와 Event 가 거기 산다) ·
                          internal/api/log_test.go 가 이미 쓰여 GET 의 시험을
                          getlog_test.go 로 가른다
```

## 이 웨이브가 회차 밖으로 낼 것 — 여섯

```text
   때가 박힌 것   unit-of-work-file-matrix.md 1절  internal/record/record.go 가 U3 하나다.
                                                U4 가 OpenLog 를 더한다.  충돌은 0
                 unit-of-work-file-matrix.md 1절  internal/transcript/** 가 U1 하나다.
                                                U4 가 json 태그를 단다.  동작 diff 0
                 component-methods.md 4.1       응답 상한과 개행 규칙이 없다.
                                                헤더 넷만으로는 폴링이 안 닫힌다

   그 밖          unit-of-work.md U2 절          「Decode 도 파서를 안 쓴다」가 거짓이다
                 component-methods.md 3절       Decode 안만 고치면 시점이 안 바뀐다 ·
                                                transcript() 의 네 갈래는 U7 의 것이다
                 requirements.md 5.4 잔여 ②      토큰이 있다는 전제로 쓰였다.
                                                데모에는 없다 — 잔여가 둘로 갈린다
```

## W-b 가 닫혔다 — 병합 2026-09-16

W-a 와 같은 규율로 **합친 나무에서 게이트를 다시 쟀다.** 각 브랜치의 값이 합쳐진
나무에서도 같다는 보장이 없으므로 그 보장을 안 믿는다.

```text
   합본        origin/main + v3-run-transcript + unit/node-stream + unit/log-api
               충돌 0 · 세 갈래의 파일 교집합 0

   초록인 것    build · vet · gofmt 빔 · glyphscan 111 파일 0
               시험 19 패키지 전부 · **스킵 0**
               커버리지 정본 awk 로 19 패키지 **미달 0** — 전체 7722/8824 = 87.5%
               transcript 95.8% · record 84.4% · enode 87.0% · api 82.3%
               windows/amd64 크로스 빌드 · net/http T 6 · crypto/tls T 1
               워킹트리 청결 (probe.lock 은 아래)

   CB0        `grep -c 'mux.HandleFunc' internal/api/api.go` 가 **18** 이다.
               앞 값 17 을 W-a 의 합본에서 쟀고 U4 가 하나를 더했다.
               CB0 의 가르는 조건이 그 수이므로 이 웨이브에서 값을 얻었다

   PR 셋       #42 회차 브랜치 -> #43 U2 -> #44 U4.  W-a 와 같은 순서다
               회차가 먼저다 — 웨이브의 상태와 감사가 거기 있다
```

**커버리지가 W-a 의 합본과 같은 87.5% 다.** 코드가 4,366줄 늘었는데 전체 비율이
안 움직인 것은 늘어난 줄의 덮인 비율이 기준선과 거의 같기 때문이다. 패키지로
보면 `internal/transcript` 가 94.4% -> **95.8%**, `internal/enode` 가 86.7% ->
**87.0%**, `internal/api` 가 82.9% -> **82.3%** 다. **api 만 내려갔다** — U4 가
`log.go` 238줄을 새로 들였고 그 파일의 갈래 몇이 폴링의 규칙적인 경로에서 안
밟힌다. 하한 80% 에서 2.3 포인트 위라 막지는 않으나 **U6 이 같은 패키지를
만지므로 거기서 다시 본다.**

**`cmd/enodectl/probe.lock` 이 또 바뀌었다.** W-a 가 기준선의 성질로 적은 그대로다
— 이 유닛들이 만든 것이 아니고 (둘 다 `cmd/enodectl` 을 안 만진다) 되돌리면 워킹
트리가 깨끗하다. 고치는 것은 이 회차의 파일 행렬 밖이다. **두 웨이브 연속으로
같은 자리라 성질이 확인됐다.**

**`internal/api` 의 deadlock(40P01)을 이 합본에서도 봤다** — W-a 가 「재현 0」으로
적고 「다음 웨이브가 `internal/api` 를 만지므로 거기서 다시 본다」고 넘긴 자리다.
U4 가 그 패키지에 405줄을 더한 뒤에도 **재현이 0 이다.** 여전히 없어졌다고 안
적는다 — U6 이 같은 패키지를 만지므로 한 번 더 본다.

## 다음 — W-c (U5 `panel-live` 단독)

```text
   U5 panel-live   FR-4 (절반)   지는 게이트 **CB1**   선행 U1 · U2
```

**이 웨이브가 하나인 것이 배치의 값이다.** CB1 은 뒤의 셋(U6 · U7 · U8)이 전부
딛는 게이트다. 그것을 눈으로 보기 전에 셋을 지으면 빨갰을 때 되돌릴 것이 셋이
되고, 그것이 앞 팩의 CP6 이 눈을 늦게 떠서 이 팩이 생긴 바로 그 실패다.

**CB1 은 사람이 실제 하네스로 보는 게이트다** (`scene-gates.md` §4 — 눈 검증을
보류로 안 넘긴다). 코드 게이트가 초록이라는 것으로 대신하지 않는다.

**U5 는 Mediator 를 안 탄다** — 링을 직접 읽는다. 그래서 U4 의 라우트가 아니라
U2 의 tee 와 U1 의 파서를 딛는다. 브랜치는 **병합된 `main` 에서 딴다**.

**정본에 갈린 자리가 하나 있다 — U5 의 NFR 요구.** `unit-of-work.md` 의 U5 절은
**스킵**이라 적고 그 근거로 「출처만 바뀐다 · 기존 `do()` 를 탄다」를 댔는데,
**그것은 U6 의 근거다** (같은 문서의 마지막 요약 블록이 그 문장을 U6 에 붙인다).
같은 문서 9절의 표는 U5 를 **돈다**로 적는다. 한 문서 안에서 두 값이다.
회차 밖으로 낼 것에 더한다. 실무로는 사용자가 W-b 에서 NFR 단계를 전부 스킵으로
지시했으므로 **W-c 에도 그 지시가 이어지는지를 단계 앞에서 묻는다.**

## 단계 진행 — U7 `chunk-push` (웨이브 순서를 바꿔 단독으로 돈다)

브랜치 `unit/chunk-push`. **병합된 `main` (05ee710) 에서 땄다.**

**회차 계획과 갈린다 — 어긋남으로 적는다.** 계획의 W-d 는 U6 · U7 병렬이고
그 앞에 W-c 의 CB1 이 있었다. 사용자가 순서를 바꿨다 — **중앙(Mediator)에 보이는
실시간이 이 회차의 목표인데 그것을 채우는 것이 U7 · U8 이기 때문이다.**
의존 행렬이 **U7 의 U5 의존을 게이트로만** 적었고 코드로는 U2 · U4 인데 둘 다
병합됐다. **CB1 은 U8 착수 앞에서 CB3 와 함께 잰다** — 둘 다 로그인된 노드에서
계약을 돌려야 하는 게이트라 한 자리에서 보는 것이 싸다.

```text
   Functional Design      Part 1 (계획) 2026-09-16.  커밋 f8a7dc0 · 물음 여덟
                          Part 2 (산출) 승인 2026-09-16T10:30:00Z (사용자 「권장대로」)
                          커밋 69fb139.  **여덟이 전부 A · 갈린 답 0**

                          **실측이 이 유닛을 통째로 막는 것을 찾았고 탐침으로 쟀다** —
                          노드의 첫 시도는 `attempt 0` 인데 (`store/claim.go:156`
                          「0 부터. 재시도면 1 이상」) U4 가 병합한 `putProgress` 가
                          `attempt <= 0` 을 400 으로 막는다.
                          탐침 — `attempt=0` -> **400** · `attempt=1` -> 200.
                          **첫 시도가 대부분이라 이대로면 한 바이트도 안 올라가고
                          CB3 이 통째로 빨갛다.** U4 가 못 본 이유는
                          `getlog_test.go` 의 진행 청크 시험 **다섯이 전부
                          `attempt=1`** 이라 그 갈래를 한 번도 안 밟기 때문이다 —
                          **W-b 의 U4 변이 넷째와 같은 모양이다.**
                          `record` 계층은 0 을 정상으로 다루므로 **막는 것은
                          API 검증 한 줄뿐이다**

                          **모순 검사가 하나를 막았다 — 「멈춘다」가 자기를 알릴 수
                          없다.** 버퍼 넘침도 멈춤이고 상한도 멈춤인데, **버퍼가
                          넘치는 조건이 「Mediator 에 못 닿는다」라 못 닿는 채로
                          표시를 보낼 수 없다.** 상한은 반대다 — 서버가 알고 서버가
                          박는다 (`enode.capped`).  **그 비대칭이 답이다**:
                          `capped` 줄이 있으면 상한이고 아무 줄 없이 끝났으면
                          노드가 포기한 것이다.  **연결이 돌아와도 재개하지
                          않는다** — 버린 바이트가 없어 재개하면 표시 없는 구멍이
                          생긴다 (답 3 = B 를 기각한 바로 그 이유)

                          **값 둘을 이 단계가 졌다** (NFR 스킵) — 버퍼 상한
                          **1 MiB** (CB3 의 10초 정지가 들어가야 해서 링의
                          512 KiB 보다 키웠다.  **CB3 이 그 값을 밟는다**) ·
                          오프셋은 **보내는 고루틴의 지역 변수**로만 둔다
                          (공유 상태가 줄고 `Write` 가 못 막히는 것이 구조로 선다)

                          **교차 검사** — 이 유닛이 새 type 을 **0 개** 들인다.
                          어휘가 안 넓어져 U8 이 그릴 것이 안 는다
   NFR Requirements       **SKIP.** 회차 계획은 U7 을 EXECUTE 로 적는다
                          (「청크 푸시가 새 표면이다」).  **어긋남으로 적는다** —
                          사용자가 이 회차에 스킵을 지시했고 값은 설계와 계획이 졌다
   NFR Design             **SKIP** (같은 지시)
   Infrastructure Design  SKIP (회차 계획 · 배포 변경 0)
   Code Generation        Part 1 (계획) 2026-09-16.  계획 construction/plans/
                          chunk-push-code-generation-plan.md
                          **Step 열둘 · 체크박스 쉰 · 물음 0**
                          Part 2 (생성) 승인 2026-09-16 (사용자 「승인」)
                          커밋 22b14b5 · **체크박스 50 중 50** · 요약
                          construction/chunk-push/code/code-summary.md

                          계획이 실측으로 둘을 찾았다 — `step` 이 배선 두 자리 다
                          스코프에 있어 넷을 따로 나를 필요가 0 이다 ·
                          `Client` 가 `advertise.go` 에 사는데 **메서드를
                          `upload.go` 에 두면 행렬이 준 파일 둘만 만진다**

                          **재전송 간격 하나를 계획이 마저 졌다** — 주기 그대로
                          2초이고 백오프를 안 둔다.  버퍼가 1 MiB 라 재전송이
                          오래 못 가고, 백오프를 두면 **멈추기까지의 시간이 늘어
                          CB3 의 10초 창을 오히려 좁힌다**
```

## U7 의 코드가 섰다 — 그리고 CB3 의 첫 줄을 실측했다

```text
   커밋      22b14b5 (unit/chunk-push).  뒤에 main 을 머지해 2f56034 이 됐다
   체크박스   50 중 50
   게이트     시험 19 패키지 초록 · 스킵 0 · 커버리지 미달 0 (7845/8954 = 87.6%)
             internal/enode 87.0% -> **87.4%** · 라우트 api.go 18 그대로
             gofmt 0 줄 · glyphscan 112 파일 0 · 크로스 빌드 OK
             diff 0 — record · transcript · panel · store · go.mod · go.sum
```

**한 줄을 고쳤고 그것이 시험 하나를 뒤집게 했다.** `api.go` 의 `attempt <= 0` 을
`< 0` 으로 바꾸니 U4 의 `TestPutLog_TheProgressBranch` 가 빨개졌다 — **그 시험의
400 목록에 `attempt=0` 이 들어 있었다.** 노드의 첫 시도를 거절하는 것이 정본으로
박혀 있던 것이다. `attempt=-1` 로 뒤집고 0 이 저장되고 읽히는 것을 재는 시험을 더했다.

**실측이 진짜 결함 하나를 잡았다** — 전송 중인 바이트가 버퍼 상한 계산에서 빠져
있어 **실제 메모리가 상한의 두 배**가 될 수 있었다. `take` 가 버퍼를 떼어 가면
그동안 `u.buf` 가 비어 보인다. 버퍼 상한 시험이 「넘칠 만큼 썼는데 안 넘쳤다」로
잡았고 `inflight` 필드를 셈에 넣었다.

**변이 일곱 중 다섯이 바로 빨갰고 둘이 살아남았다.** ⑤ 는 규칙이 공허했다 —
새 시도를 보면 **멈추므로** 그 뒤에 오프셋을 읽는 자리가 0 이다. 시험을 억지로
짓지 않고 **`business-rules` 의 R8 을 걷었다.** ⑦ 은 순서를 재는 시험을 안 지어
살아남았고, 지으면서 **가짜 Mediator 가 진행 청크와 선별본을 한 자리에 덮고
있던 것**도 드러났다.

### CB3 의 첫 줄이 초록이다 — 실 하네스로 쟀다 (2026-09-16)

배포본으로 계약 하나를 실제로 돌렸다 (`cb3-live-1` · 에이전트 단계 하나).

```text
   도는 동안   source=progress   bytes 30307 -> 71730 -> 76271   **attempt=0**
   끝난 뒤     source=sealed     bytes 6785
   as=events  사건 29 · lines 30 · head 0 · partial 0
              init 1 · text 6 · tool_use 6 · tool_result 6 · result 1 · raw 9
   도구 이름   Bash · Read · Read · Read · Bash · Write  (CB1 이 재는 것의 재료다)
   판정       SUCCEEDED · checks 가 배열이고 조건 하나가 담겼다
```

**`attempt=0` 이 받아진 것이 이 회차에서 고친 자리다.** 고치기 전에는 여기서
전부 400 이라 한 바이트도 안 올라갔다.

**`raw` 아홉은 설계대로다** — `rate_limit_event` 셋과 `system/thinking_tokens`
여섯이다. 파서의 어휘가 닫혀 있어 모르는 `type` 은 버리지도 자리를 지어 주지도
않고 `raw` 로 떨어진다. **다만 서른 줄 중 아홉이라 화면에 그대로 그리면 카드의
3분의 1 이 장부 줄이 된다** — U5 · U8 의 그리는 규칙이 이것을 어떻게 다룰지는
안 정했다. **회차 밖으로 낸다.**

### 봉인 뒤의 줄도 쟀다 — 같은 바이트다

```text
   runctl record cb3-live-1 -o r.tar   17408 바이트 · rc 0
   tar 안                             run-cb3-live-1/logs/01-summarize.log 외 셋
   GET (봉인 뒤) 과 cmp               **6785 대 6785 · 같은 바이트다**
```

### CB3 이 아직 안 잰 것 — 하나

```text
   Mediator 를 10초 멈췄다 켜기   버퍼 상한 1 MiB 가 그 10초를 받는지 ·
                               다음 청크가 이어 붙고 두 벌이 안 생기는지.
                               **안 쟀다.**  시험으로는 초록이나 실물로는 안 밟았다
```

**앞 판에 「`runctl record` 가 tar 를 안 낸다」고 적었다가 지웠다 — 제가 틀렸다.**
환경변수를 안 내보낸 셸에서 돌리고 `2>/dev/null` 로 오류를 삼켰다. 도구는 멀쩡하다.

## 다음 세션이 서 있는 자리 (2026-09-16 저녁)

```text
   main              f6458b2.  U1 · U2 · U3 · U4 병합됨 + verdict 수정
   unit/panel-live   2456ded.  U5 코드 섰음 · **미병합** · CB1 안 쟀다
   unit/chunk-push   2f56034.  U7 코드 섰음 + main 머지 · **미병합** · 배포본이 이것이다
   안 지은 것         U6 panel-past · U8 fleet-card
```

**웨이브 순서를 사용자가 바꿨다** — 회차 계획의 W-c(U5) -> W-d(U6·U7) -> W-e(U8)
대신 U5 를 미뤄 두고 U7 을 먼저 돌렸다. 중앙에 보이는 실시간이 목표인데 그것을
채우는 것이 U7 · U8 이기 때문이다. 의존 행렬이 U7 의 U5 의존을 **게이트로만**
적었고 코드로는 U2 · U4 라 가능했다.

**화면은 아직 아무것도 안 바뀌었다.** 데이터는 흐르는데 (CB3 의 첫 줄과 마지막
줄이 초록) 그리는 유닛이 둘 다 안 붙어 있다 — U5 는 미병합이고 U8 은 미착수다.
배포본을 열어도 제어판은 앞 판 그대로 JSON 원문을 보이고 중앙에는 카드가 없다.

**남은 게이트**

```text
   CB1   흐른다      **안 쟀다.**  U5 가 진다.  사람이 실 하네스로 본다
   CB2   읽힌다      U6 이 진다.  안 지었다
   CB3   올라간다    **첫 줄과 마지막 줄 초록.**  10초 정지만 안 쟀다
   CB4   보인다      U8 이 진다.  안 지었다
   CB6   한 장면     U8 이 진다
```

## U8 이 착수하자마자 밟을 자리 — 카드가 URL 을 못 만든다

**실측으로 밟았다 (2026-09-16 · `cb3-live-1`).**

```text
   GET .../steps/{seq}/log 는 ?name= 을 받고 없으면 "step" 으로 기본값을 쓴다
   Run 상세의 단계 객체에는 **name 이 없다** — 있는 것은 id 다

   {"seq":1,"id":"summarize","state":"DONE","uses":"brain", ...}

   name 없이        -> 200  X-Enode-Log-Bytes: 0      **빈 카드**
   name=summarize  -> 200  X-Enode-Log-Bytes: 6785   내용이 온다
```

**404 가 아니라 200 에 0 바이트라 더 아프다** — 화면이 「에이전트가 아무 말도
안 했다」로 그리고, 보는 사람도 짓는 사람도 어디가 틀렸는지 못 짚는다.

**닫는 법은 화면 쪽이다.** 상세의 `id` 가 곧 로그의 이름이다 — `StepView.ID` 가
`steps.name` 을 그대로 싣는다 (`observe.go:54` 의 `SELECT seq, name, ...`).
확인했다: DB 의 `steps.name` 이 `summarize` 이고 여러 단계짜리 Run 에서도
(`probe_exec` · `probe_exec2` · ...) 같다.

```text
   권하는 것   U8 이 steps[].id 를 ?name= 으로 넘긴다.  U8 안에서 닫히고
              「ui.go 의 diff 가 0 이다」와도 맞는다
   안 권하는 것  서버의 기본값을 그 단계의 실제 이름으로 바꾸는 것 —
              Go 를 건드리게 되고 U8 의 범위가 넓어진다
```

**회차 밖으로 낼 것에 더한다** — `component-methods.md` 의 `GET log` 절이
`name` 의 기본값을 `step` 으로 적었는데, **그 기본값이 맞는 Run 이 실제로 몇이나
되는지 아무도 안 셌다.** 계약이 단계 이름을 지으므로 대개 안 맞는다.

## 회차 밖의 일 — 이 세션이 한 것

```text
   배포          CT103(이 기계)의 enode-dev 를 0.1.0-rc3 (2f56034e1f64) 로 올렸다.
                mediator · enode · runctl · enodectl 넷.  앞 판은 .bak- 로 옆에 있다
                설정과 토큰은 안 건드렸다.  노드 exec 가 함대에 붙어 있다
   NPM          mediator.taeels.duckdns.org (ID 19) · dashboard (ID 20) 가 이미
                8080 을 가리키고 인증서가 12월까지다.  새로 만든 것 0
   verdict 수정   PR #45 로 main 에 병합.  브랜치 삭제됨.  회차 밖이라 유닛 문서에
                안 적는다 — 근거는 audit.md 와 그 PR
```

**배포본이 `main` 에 없는 커밋이다.** `unit/chunk-push` 나무여야 하는 이유가
있다 — `main` 의 mediator 로는 노드의 첫 시도(`attempt 0`) 청크가 전부 400 이라
실시간 경로가 죽는다. 그 한 줄 수정이 U7 과 같은 브랜치에 있다.
