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
| U2 | `contract-vocab` | 3.5 | CA3 의 절반 | 없음 | 대기 |
| U3 | `advert` | 3.3 | CA2 · CA3 완결 | U1 · U2 | 대기 |
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
