# Construction 상태 — taeels (최태양)

담당 유닛 — **obs**(W0 · CP1 · 토대) · **mcp**(W1 · CP5 · +CP7 도구).
자기 브랜치에서 작업하고 PR 로 main 에 병합한다(게이트 초록 뒤). 산출물은 이
디렉터리 `aidlc-docs/taeels/` 아래.

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
