# AI-DLC State Tracking — v3-run-harness-components

## Project Information
- **Project Type**: Brownfield
- **Start Date**: 2026-09-11T13:13:54Z
- **Current Stage**: INCEPTION — Units Generation 완료 · **검증 루프 진행 중** (11차까지)
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
- **Mediator 라우트**: 26 (`internal/api/*.go` 의 `mux.HandleFunc` + `mux.Handle(`). 옛 셈법인 `api.go` 한 파일의 `HandleFunc` 만 세면 17 이 나오고 아홉을 놓친다
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

## 이 회차가 `decisions.md` 에 더한 행 — 열아홉
셋은 Requirements 승인 뒤에, 둘은 Application Design 이, **열넷은 2026-09-12 의
검증 루프 열한 바퀴**가 실었다.

```text
   ①  알려진 키 검증은 이미 있다        3.5 의 범위가 목록 추가로 준다
   ②  팩 tar 풀기의 경로 검증           절대경로 · .. · 심볼릭 링크 거부 · 크기 상한
   ③  팩의 기대 다이제스트는 이월       기록으로 족한 근거 셋
   ④  계장 디렉터리를 못 만들면 단계 실패  Application Design Q2.  오늘 동작이 바뀐다
   ⑤  탐지기 순회로 바꾼다              Q3 = B (뒤집힘).  Fingerprinter · 종류 셋
   ⑥  팩의 서버도 agent.mcp 필터를 탄다   문서 셋과 셋이 정반대였다
   ⑦  팩이 노드 선언 이름을 덮으면 거절    소유권이 뒤집히는 경로를 막는다
   ⑧  훅 설정을 가짜 홈 안으로          features.md 3.1 의 요구대로 되돌렸다
   ⑨  실패 등급의 기본이 치명이다        빠뜨림이 닫히는 쪽으로 틀린다
   ⑩  계장 보존 스위치를 안 둔다         한 번 넣었다 뺐다.  3.1 의 보안 요구를 뚫는다
   ⑪  보조 등급은 하나다                기준 시각은 Instrument 앞이라 안 닿는다
   ⑫  agent.pack 이 미래 결합점이다      submit --pack 이 와도 계약 어휘는 안 는다
   ⑬  열거 면은 runctl capabilities 다   계약 작성자가 보는 면.  nodes 는 운영자 면
   ⑭  mcpUp 이 못 잡는 것 셋            뜨나는 존재이지 동작이 아니다
   ⑮  Argv 에 stream-json 한 줄         게이트가 복제본 대신 실물을 잰다
   ⑯  ParseClaude 가 type 을 본다        ⑮ 이 연 fail-open 을 막는다
   ⑰  봉인 기록의 자격증명 누출        준수로 만든다 (사용자 결정)
   ⑱  도구 사건은 껍데기만 남긴다        logs/ 를 허용목록으로 거른다
   ⑲  stream-json 동안 링을 닫는다       tee 가 선별 앞이라 ⑱ 만으로는 안 닫힌다
```

열아홉 다 `requirements/harness-components/decisions.md` **6절**에 실렸고,
**6.1 이 뒤집힌 옛 문자열을 세는 열쇠 표**를 든다 — 진행자가 유닛 병합 전에
돈다. 도는 스크립트가 없어 CA0 의 기계 검사가 아니다.
기존 1 ~ 5절의 번호는 다른 문서가 참조하므로 안 건드리고, **2절의 낡은 행에는
「6절이 뒤집었다」 꼬리표를 달았다** — 권장값 표가 구현자에게 먼저 읽히기 때문이다.

## Execution Plan Summary
정본은 `inception/plans/execution-plan.md` (2026-09-11T13:59:38Z).

- **전체 단계**: 13 (Inception 7 · Construction 5 · Operations 1)
- **실행**: Application Design · Units Generation · Functional Design ·
  Code Generation · Build and Test
- **스킵 셋**: NFR Requirements · NFR Design · Infrastructure Design. **User Stories 는 2026-09-12 에 최소 형태로 되살렸다**
- **위험도**: High · 되돌리기 Moderate · 검사 복잡도 Complex

스킵의 근거를 한 줄로.

```text
   User Stories            **되살렸다** (2026-09-12).  스킵 근거가 규칙의 SKIP 조건에
                           안 걸렸고 하류(units-generation · functional-design)가
                           story map 을 읽는다.  최소 형태로 돈다 — 행복 경로는 안 쓰고
                           저작 경로와 오류 경로만 짓는다
   NFR Requirements        requirements.md 4절이 Comprehensive 로 이미 닫았고
                           이 회차가 그 값을 안 바꾼다.  차단 확장의 집행은 단계마다 그대로 돈다
   NFR Design              NFR Requirements 를 건너뛰므로 넘길 패턴이 없다
   Infrastructure Design   배포 모형이 안 바뀐다.  새 포트 0 · 새 전송 0 · 클라우드 자원 0
```

값이 안 정해진 자리는 스킵과 함께 사라지지 않고 옮겨 적었다 — SEC-A 의 크기 ·
개수 상한은 팩 유닛의 Functional Design 으로, 4.4 의 「광고 루프가 탐지를 직접
안 부른다」는 Application Design D2 로.

## 설계 검증 (2026-09-12) — 에이전트 일곱

승인 뒤 검증을 돌렸다. 축 셋(코드 대조 · 정본 대조 · 요구 팩과 내부 일관성)과
단계별 반대 심문 넷이다. **「답이 전부 권장안」은 거짓이었다** — `requirement-verification-questions.md` 의 Q1 · Q2 는 권장이 A 인데 답이 B 였다. 쏠린 것은 `Other` 선택지가 없던 질문 파일들이고, 그 원인은 구조다 —
질문을 내는 단계 셋(User Stories · NFR Requirements · NFR Design)을 스킵했고,
Q1 은 규칙이 정한 선택지에만 반대 근거가 붙어 있었고, Q3 은 네 번째 길을 안 보였다.

```text
   뒤집힌 결정     Q3 = A -> B.  근거 둘이 다 오독이었다
   설계 결함        decisions.md 6절 ⑥ ~ ⑭ 와 application-design.md 4.3 ~ 4.6
   규칙 위반 셋     User Stories 스킵 · NFR Requirements 스킵 ·
                  Workspace Detection 이 규칙 분기 대신 질문으로 돌린 것
   집행자 0 명     scene-gates.md 2절 머리를 회차가 안 옮겨 적었다.
                  unit-of-work-dependency.md 8절이 그 배정을 진다
   측정 오류 셋     하네스 exec 만 한 자리다 (internal/enode 에 여덟) ·
                  라우트는 17 이 아니라 26 · 매처가 분류에서 빠졌다
```

검증 도구를 `.claude/agents/aidlc-verify.md` 로 굳혔다 — 다음 회차가 같은
프롬프트를 다시 짓지 않는다.

## Application Design 이 닫을 넷 (D1 ~ D4)
`requirements.md` 8절이 명시로 넘긴 미결이다.

| | 물음 | 어디서 왔나 |
|---|---|---|
| D1 | `Fixed()` 가 계장 디렉터리를 어떻게 아나 | requirements.md 2.2 |
| D2 | Detector 가 MCP 를 어떻게 드나. 광고 루프와 어떻게 끊나 | ADR-035 §4.2 · requirements.md 4.4 |
| D3 | 허용목록을 누가 쓰나. 팩 펴기와 같은 자리인가 | requirements.md 8절 |
| D4 | 게이트가 transcript 링을 쓸 수 있나 | requirements.md 2.3 · 5.1 |

**넷 다 닫혔다** (2026-09-12T00:38:40Z). 답은 전부 권장(A)이었다.

```text
   D1   Fixed(dir string) map[string]string 으로 인터페이스를 바꾼다 (Q1 = A)
   D2   costlyAttrs 를 Fingerprinter 순회로 바꾼다 (Q3 = B · 2026-09-12 뒤집힘)
   D3   결정(resolveComponents)과 쓰기(Instrument)를 가른다 (Q4 = A)
   D4   2026-09-12 에 뒤집혔다 (decisions.md 6절 ⑮).  Argv 에
        --output-format stream-json --verbose 한 줄을 U1 이 가져온다 —
        그러지 않으면 CA1 · CA4 · CA5 가 재려는 줄이 이 회차에 없다.
        게이트는 logs/ 경로다.  Decode 와 tee 는 짝 팩의 것으로 둔다
```

설계가 답 밖에서 더 정한 둘 — `Instrument` 의 오류를 등급으로 가르고(뒤에 ⑨ · ⑪ 이 방향을 뒤집었다. 그때는 `errComponents`
만 치명), `Instrument` 를 언제나 부른다. 근거는 `application-design.md` 4절.

## 만지는 경로가 셋으로 줄었다
`component-methods.md` 7절의 측정 — **`cmd/runctl` 의 diff 가 0 이다.**
`runctl example` 이 `contract.ExampleNames()` 로 임베드 FS 를 읽으므로 예시는
`internal/contract/examples/mcp.json` 하나로 족하고 `runctl schema steps` 는
구조체에서 뽑는다. Units Generation 의 파일 행렬이 이 값을 쓴다.

## 짝 팩과 겹치는 파일은 둘이다
`constraints.md` 접점 표의 여섯 줄 중 남는 것은 **둘**이다 — `runner.go` 와
`claude.go` 의 `Argv` (`component-dependency.md` 5절). `Argv` 는 Q5 = A 로 한 번
빠졌다가 ⑮ 로 되돌아왔다. 이 팩이 플래그 한 줄을 더하고 짝 팩이 그 위에서
출력 처리를 자라게 하는 모양이라 병합이 기계적이다.

## Stage Progress

### INCEPTION PHASE
- [x] Workspace Detection — 2026-09-11T13:13:54Z
- [x] Reverse Engineering — 부분 재측정 (네 경로 · Q1=B). 전면 재실행은 안 함
- [x] Requirements Analysis — 승인됨 (2026-09-11T13:59:38Z · 사용자가 Workflow Planning 을 지시)
- [x] User Stories — **EXECUTE (minimal)** — 2026-09-12 에 되살렸다. 스킵이 규칙의 SKIP 조건 여섯 중 0 에 걸렸다
- [x] Workflow Planning — 승인됨 (2026-09-11T23:43:28Z · 커밋 88dc120)
- [x] Application Design — 승인됨 (2026-09-12T00:41:18Z · 커밋 3012a92). 답은 전부 A
- [x] Units Generation — 완료 (2026-09-12). 답 다섯이 전부 A. 유닛 다섯 · 파일 행렬을 냈다

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

## Units Generation 이 낸 것 (2026-09-12)
정본은 `inception/application-design/` 의 넷이다 — `unit-of-work.md` ·
`unit-of-work-dependency.md` · `unit-of-work-story-map.md` ·
`unit-of-work-file-matrix.md`.

### 유닛 다섯 · 직렬

| | 유닛 | 맡는 기능 | 닫는 게이트 | 선행 |
|---|---|---|---|---|
| U1 | `isolation` | 3.1 · 3.2 의 최소 | CA1 | 없음 |
| U2 | `contract-vocab` | 3.5 | CA3 의 절반 | 없음 |
| U3 | `advert` | 3.3 | CA2 · CA3 완결 | U1 · U2 |
| U4 | `sources` | 3.4 · 3.2 의 완성 | CA4 | U1 · U2 · U3 |
| U5 | `pack` | 3.6 · 3.7 | CA5 · CA6 | U1 · U2 · U4 |

착수 순서는 U1 · U2 · U3 · U4 · U5 다. **모든 유닛의 완료 조건에 CA0 가 들어간다** —
`internal/enode` 의 커버리지 80% 를 유닛 단위로 집행한다.

### 행렬이 센 것

```text
   만지는 파일     14   새 파일 둘(mcp.go · examples/mcp.json) · 고치는 파일 열둘
   만지는 패키지    3   internal/enode · internal/contract · cmd/iapadapter
   접점 파일 넷     mcp.go(U1·U3·U4·U5) · runner.go(U1·U4·U5) ·
                   claude.go(U1·U5) · harness.go(U1·U5).  직렬이라 충돌이 아니다
   짝 팩과 겹치는 것  runner.go 하나.  팩 단위로 이 팩이 먼저 전부 들어간다
   cmd/runctl       소스 diff 0.  다만 shape_test.go 가 새 예시 위로 돈다
```

### 설계 요약 표에 없던 파일 둘 — 행렬이 세웠다

```text
   internal/enode/claim.go   U4.  Job{...} 리터럴의 유일한 제품 코드 자리(765).
                             NodeMCP 를 안 실으면 노드 선언이 조용히 안 실린다
   internal/enode/agent.go   U2.  AgentParams(36)와 parseAgentParams(500).
                             components.md 1.6 은 적었고 요약 표만 빠졌다
```

## Current Status
- **Lifecycle Phase**: INCEPTION 완료 · 다음은 CONSTRUCTION
- **Current Stage**: Units Generation 완료
- **Next Stage**: CONSTRUCTION — U1 `isolation` 의 Functional Design
- **Status**: 승인 대기

## 다음 세션이 CONSTRUCTION 에서 쓸 입력
```text
   첫 유닛          U1 isolation.  선행 없음.  CA1 을 닫는다
   브랜치           unit/isolation 을 v3-run-harness-components 에서 딴다.
                   게이트가 초록이면 PR 로 main (CONVENTIONS 3.1 · 3.3)
   문서 루트         aidlc-docs/taeels/construction/isolation/
   FD 가 닫을 것     mcp.json 의 필드 표현 (빈 파일의 모양)
   완료 조건         unit-of-work.md 1절.  CA1 은 OAuth 노드와 게이트웨이 노드
                   둘 다에서 사람이 한 번 띄워야 초록이다
   같이 도는 것      CA0 — 커버리지 80% 를 이 유닛에서 이미 잰다
```
