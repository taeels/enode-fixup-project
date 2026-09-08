# Construction 상태 — taeels (최태양)

담당 유닛 — **obs**(W0 · CP1 · 토대) · **mcp**(W1 · CP5 · +CP7 도구).
자기 브랜치에서 작업하고 PR 로 main 에 병합한다(게이트 초록 뒤). 산출물은 이
디렉터리 `aidlc-docs/taeels/` 아래.

## 유닛

- [ ] obs 관측 API — CP1 · 의존 없음(토대). **먼저 병합돼야 W1(queue·mcp·ui)이 선다**
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
   Functional Design      2차 병렬 검토(축 넷)를 반영했다 — 승인을 기다린다
                          construction/obs/functional-design/ (domain-entities ·
                          business-logic-model · business-rules)
                          답 다섯 C·A·A·A·A 는 계획 2절 · 파장은 6절
                          1차 검토(하류 담당별) 7절 · 2차 검토(정합·정본·코드·게이트) 8절
                          실행으로 확정된 치명 둘을 고쳤다 — UpsertAdvert CTE 의 0행
                          NULL · coalesce 가 못 접는 jsonb null
   NFR Requirements       대기
   NFR Design             대기
   Infrastructure Design  건너뛴다 (배포·인프라 변경 없음 · constraints §6)
   Code Generation        대기
```

브랜치 `unit/obs`. 감사 로그는 `aidlc-docs/taeels/audit.md`.

## 다음

**obs Functional Design 승인 뒤 NFR Requirements** (토대라 먼저 병합돼야 나머지
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
