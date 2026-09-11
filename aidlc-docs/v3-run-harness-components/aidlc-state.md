# AI-DLC State Tracking — v3-run-harness-components

## Project Information
- **Project Type**: Brownfield
- **Start Date**: 2026-09-11T13:13:54Z
- **Current Stage**: INCEPTION — Requirements Analysis (질문 대기)
- **AI-DLC Version**: 1.0.1 (`.aidlc/aidlc-rules/`)
- **Run Branch**: `v3-run-harness-components` (Inception 산출물이 여기 직렬로 쌓인다. CONVENTIONS.md 3.1)
- **문서 루트**: `aidlc-docs/v3-run-harness-components/` (CLAUDE.md 의 회차별 layering)

## Workspace State
- **Existing Code**: Yes
- **Programming Languages**: Go (module `github.com/taeels/enode`)
- **Build System**: Go modules. CI 는 GitHub Actions
- **Project Structure**: 멀티 바이너리 모노레포 — `cmd/{mediator,enode,enodectl,runctl,iapadapter}` 다섯 + `internal/*` 공용 패키지. 설계 정본은 서브모듈 `enode-design/`
- **Workspace Root**: /home/sunny/enode-fixup-project
- **Go 소스**: 190 파일 (`cmd/` · `internal/`)
- **Mediator 라우트**: 17 (`grep -c 'mux.HandleFunc' internal/api/api.go`)
- **enode-design 핀**: `29c89cd` · `origin/main` 과의 거리 0

## Reverse Engineering
공용이다 — `aidlc-docs/inception/reverse-engineering/` (CLAUDE.md). 이 회차가
새로 만들지 않는다.

- **산출물 시각**: 2026-09-08T07:17:58Z (여덟 문서)
- **그 뒤 코드 이동**: `cmd/` · `internal/` 에 커밋 43. 라우트 15 -> 17 · Go 파일 143 -> 190
- **이 팩이 만지는 경로의 이동**: `internal/enode` 4 · `internal/contract` 4 · `cmd/iapadapter` 0 · `cmd/runctl` 0. 움직인 32 는 `internal/api` 로 이 팩의 밖이다
- **판정**: 사용자 확인 대기 (Q1). 팩이 만지는 경로는 거의 안 움직였고 움직인 곳은 이 팩 밖이라 갱신 없이 진행하는 쪽을 권한다

## Code Location Rules
- **Application Code**: 저장소 뿌리 (`cmd/`, `internal/`). aidlc-docs/ 에는 안 들어온다
- **Documentation**: `aidlc-docs/v3-run-harness-components/` 만
- **Structure patterns**: See code-generation.md Critical Rules

## Requirements Pack (입력)
이 회차는 요구 팩이 이미 있다. Requirements Analysis 가 이것을 입력으로 읽는다.
- `requirements/harness-components/features.md` — 기능 일곱 (3.1 ~ 3.7)
- `requirements/harness-components/decisions.md` — 결정표 (1 ~ 5절)
- `requirements/harness-components/scene-gates.md` — 장면 조각 게이트 CA0 ~ CA6 (수용 기준)
- `requirements/harness-components/canon.md` — enode-design 정본과의 연결
- `requirements/harness-components/constraints.md` — 제외 여덟 범주 · 구조 불변식 · 접점

짝 팩은 `requirements/transcript/` (`v3-run-transcript`)이고 독립이다. 접점은
`constraints.md` 의 접점 절이 적는다.

## Extension Configuration
`decisions.md` §1 이 사용자 결정(2026-09-11)으로 이미 닫았다. Requirements
Analysis Step 5.1 에서 재확인만 한다.

| Extension | Enabled | Decided At |
|---|---|---|
| security-baseline | Yes | decisions.md §1 (사용자 결정 2026-09-11) |
| resiliency-baseline | No | decisions.md §1 |
| property-based-testing | No | decisions.md §1 |

취급: `decisions.md` §3 이 SECURITY 규칙마다 처리를 미리 적었다 — 새 표면(가짜
홈 · 허용목록 · 노드 선언 · 계약 키)에만 걸고 기존 코드 사실은 기록만 한다.

## Stage Progress

### INCEPTION PHASE
- [x] Workspace Detection — 2026-09-11T13:13:54Z
- [x] Reverse Engineering — SKIP (공용 산출물 사용. 신선도 판정은 Q1)
- [ ] Requirements Analysis — 진행 중 (질문 파일 제출 · 답 대기)
- [ ] User Stories — 미정 (Workflow Planning 이 정한다)
- [ ] Workflow Planning
- [ ] Application Design — 미정
- [ ] Units Generation — 미정

### CONSTRUCTION PHASE
- [ ] 유닛별 단계 — Units Generation 뒤에 채운다
- [ ] Build and Test
