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

## 다음

**obs Functional Design 부터** (토대라 먼저 병합돼야 나머지 웨이브가 선다). 접점
`internal/store` · `internal/api/api.go` 는 진행자 직렬 병합.
