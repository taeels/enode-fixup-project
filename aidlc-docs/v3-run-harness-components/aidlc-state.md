# AI-DLC State Tracking — v3-run-harness-components

## Project Information
- **Project Type**: Brownfield
- **Start Date**: 2026-09-11T13:13:54Z
- **Current Stage**: INCEPTION — Workflow Planning 완료 · 승인 대기. 다음은 Application Design
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
- **판정**: 질문 1 의 답 B — **네 경로만 다시 쟀다** (2026-09-11T13:13:54Z). `code-structure.md` · `component-inventory.md` · `reverse-engineering-timestamp.md` 에 실었다. 바뀐 것은 `internal/enode` 의 새 파일 셋(`policy.go` · `status.go` · `transcript.go`)과 `internal/contract/advert.go` 뿐이고 `cmd/iapadapter` · `cmd/runctl` 은 무변경이다. 안 잰 자리의 낡음은 관측값으로만 적었다

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

## 확인 질문의 답 (2026-09-11)
| | 질문 | 답 | 정해진 것 |
|---|---|---|---|
| Q1 | 공용 R/E 를 갱신하나 | B | 이 팩이 딛는 네 경로만 다시 잰다 |
| Q2 | 회차를 어디까지 · 몇 손 | B | Inception 을 돌고 Construction 까지 한 손(taeels)으로 간다. 유닛 직렬 |
| Q3 | 짝 팩과의 순서 | A | 이 팩 먼저. 게이트의 `init` 줄은 사람이 직접 띄워 읽는다 |

## 이 회차가 `decisions.md` 에 더할 행
승인 뒤 진행자가 `requirements/harness-components/decisions.md` 에 2026-09-11
날짜 절로 싣는다. 근거는 `requirements.md` 7절.

```text
   ①  알려진 키 검증은 이미 있다        3.5 의 범위가 목록 추가로 준다
   ②  팩 tar 풀기의 경로 검증           절대경로 · .. · 심볼릭 링크 거부 · 크기 상한
   ③  팩의 기대 다이제스트는 이월       기록으로 족한 근거 셋
```

## Execution Plan Summary
정본은 `inception/plans/execution-plan.md` (2026-09-11T13:59:38Z).

- **전체 단계**: 13 (Inception 7 · Construction 5 · Operations 1)
- **실행**: Application Design · Units Generation · Functional Design ·
  Code Generation · Build and Test
- **스킵 넷**: User Stories · NFR Requirements · NFR Design · Infrastructure Design
- **위험도**: High · 되돌리기 Moderate · 검사 복잡도 Complex

스킵의 근거를 한 줄로.

```text
   User Stories            scene-gates.md 의 조각 일곱이 실행 명령으로 적힌 수용 기준이다.
                           스토리는 그것을 약한 형태로 다시 쓰는 일이 된다
   NFR Requirements        requirements.md 4절이 Comprehensive 로 이미 닫았고
                           이 회차가 그 값을 안 바꾼다.  차단 확장의 집행은 단계마다 그대로 돈다
   NFR Design              NFR Requirements 를 건너뛰므로 넘길 패턴이 없다
   Infrastructure Design   배포 모형이 안 바뀐다.  새 포트 0 · 새 전송 0 · 클라우드 자원 0
```

값이 안 정해진 자리는 스킵과 함께 사라지지 않고 옮겨 적었다 — SEC-A 의 크기 ·
개수 상한은 팩 유닛의 Functional Design 으로, 4.4 의 「광고 루프가 탐지를 직접
안 부른다」는 Application Design D2 로.

## Application Design 이 닫을 넷 (D1 ~ D4)
`requirements.md` 8절이 명시로 넘긴 미결이다.

| | 물음 | 어디서 왔나 |
|---|---|---|
| D1 | `Fixed()` 가 계장 디렉터리를 어떻게 아나 | requirements.md 2.2 |
| D2 | Detector 가 MCP 를 어떻게 드나. 광고 루프와 어떻게 끊나 | ADR-035 §4.2 · requirements.md 4.4 |
| D3 | 허용목록을 누가 쓰나. 팩 펴기와 같은 자리인가 | requirements.md 8절 |
| D4 | 게이트가 transcript 링을 쓸 수 있나 | requirements.md 2.3 · 5.1 |

## Stage Progress

### INCEPTION PHASE
- [x] Workspace Detection — 2026-09-11T13:13:54Z
- [x] Reverse Engineering — 부분 재측정 (네 경로 · Q1=B). 전면 재실행은 안 함
- [x] Requirements Analysis — 승인됨 (2026-09-11T13:59:38Z · 사용자가 Workflow Planning 을 지시)
- [x] User Stories — SKIP (실행 계획 3절의 근거 둘)
- [x] Workflow Planning — 산출물 완료 · 승인 대기
- [ ] Application Design — EXECUTE (D1 ~ D4 를 닫는다)
- [ ] Units Generation — EXECUTE (파일 행렬 필수)

### CONSTRUCTION PHASE
담당은 taeels 하나다 (Q2=B). 문서 루트는 `aidlc-docs/taeels/` 이고 유닛은
`unit/<유닛>` 브랜치에서 직렬로 돈다.
- [ ] Functional Design — EXECUTE (유닛마다 · 형식을 안 만드는 유닛은 그 자리에서 스킵)
- [ ] NFR Requirements — SKIP
- [ ] NFR Design — SKIP
- [ ] Infrastructure Design — SKIP
- [ ] Code Generation — EXECUTE (유닛마다 · 계획 뒤 생성)
- [ ] Build and Test — EXECUTE (조각 게이트 CA0 ~ CA6)

### OPERATIONS PHASE
- [ ] Operations — PLACEHOLDER

## Current Status
- **Lifecycle Phase**: INCEPTION
- **Current Stage**: Workflow Planning 완료
- **Next Stage**: Application Design
- **Status**: 승인 대기
