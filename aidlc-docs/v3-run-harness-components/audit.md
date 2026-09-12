# AI-DLC Audit Log — v3-run-harness-components

모든 사용자 입력과 결정을 시간순으로 기록한다. 사용자 원문은 요약하지 않고
그대로 싣는다. 타임스탬프는 `date -u +%Y-%m-%dT%H:%M:%SZ` (ISO 8601 UTC).

이 파일은 이 회차의 것이다 (CLAUDE.md 의 회차별 layering). 루트
`aidlc-docs/audit.md` 에는 안 쓴다.

---

## Workspace Detection — 초기 사용자 요청
**Timestamp**: 2026-09-11T13:13:54Z
**User Input**: "하네스 구성요소 aidlc 하자"
**AI Response**: 워크스페이스를 조사하고 이 회차를 연다. 요구 팩은
`requirements/harness-components/` 다섯 파일이고 회차 브랜치는
`v3-run-harness-components` (커밋 `7d92bfd`)다. 문서 루트는
`aidlc-docs/v3-run-harness-components/` 로 잡는다.
**Context**: INCEPTION · Workspace Detection 착수. AI-DLC 1.0.1
(`.aidlc/aidlc-rules/`). Office Task `7397d2e0` 에 바인딩함.

---

## Workspace Detection — 완료
**Timestamp**: 2026-09-11T13:16:20Z
**User Input**: (없음 — 이 단계는 승인 게이트가 없다)
**AI Response**: 브라운필드 확인. Go 모노레포 · 실행파일 다섯 · Go 파일 190 ·
Mediator 라우트 17 · enode-design 핀 29c89cd (origin/main 과 거리 0).
공용 R/E 산출물이 `aidlc-docs/inception/reverse-engineering/` 에 있으므로
Reverse Engineering 은 안 돈다. 다만 산출물 시각(2026-09-08T07:17:58Z) 이후
`cmd/` · `internal/` 에 커밋 43 이 들어와 신선도가 걸린다 — 경로별로 세면
움직인 32 가 `internal/api` 로 이 팩 밖이다. 판정을 Requirements Analysis
질문 1 로 넘긴다.
**Context**: INCEPTION · Workspace Detection 완료. 다음은 Requirements Analysis

---

## Requirements Analysis — 착수와 팩 실측
**Timestamp**: 2026-09-11T13:16:20Z
**User Input**: (없음)
**AI Response**: 팩 다섯 파일을 입력으로 읽고, `features.md` 1.3 이 적어둔 세는
명령을 실제로 돌렸다. 셋은 맞고 하나가 틀렸다 — `agent` 맵의 알려진 키 검증은
이미 있다 (`internal/contract/contract.go:902` `agentKeys` · `knownKeys`,
ADR-057). 3.5 의 그 항목은 목록에 `mcp` · `pack` 두 이름을 더하는 일로 줄어든다.
범위가 주는 방향이라 팩의 게이트 규칙대로 묻지 않고 `decisions.md` 에 행으로
적는다. 확장 opt-in 은 `decisions.md` §1 이 이미 닫았다 (security-baseline 켬 ·
나머지 둘 끔) — Step 5.1 에서 재확인만 하고 `security-baseline.md` 규칙 파일을
읽는다.
**Context**: INCEPTION · Requirements Analysis Step 1 ~ 5.1

---

## Requirements Analysis — 질문 제출
**Timestamp**: 2026-09-11T13:16:20Z
**User Input**: (없음 — 답 대기)
**AI Response**: `aidlc-docs/v3-run-harness-components/inception/requirements/requirement-verification-questions.md`
에 질문 셋을 냈다. Q1 공용 R/E 신선도 · Q2 회차의 끝과 손의 수 · Q3 짝
팩(transcript)과의 순서. 팩이 값으로 닫은 자리는 안 묻는다.
**Context**: INCEPTION · Requirements Analysis Step 6 게이트 — 답 대기

---

## Requirements Analysis — 질문 답 수신
**Timestamp**: 2026-09-11T13:32:23Z
**User Input**: "답했다."
질문 파일의 답 원문 (`[Answer]:` 태그 그대로, 덧붙인 설명 없음)
- Q1 공용 R/E 를 갱신하나 -> "B"
- Q2 이 회차를 어디까지 도나, 그리고 몇 손인가 -> "B"
- Q3 짝 팩(transcript)과의 순서 -> "A"
**AI Response**: 셋 다 모호함이 없어 추가 질문을 내지 않는다. 답이 정하는 것 —
(Q1=B) 이 팩이 딛는 네 경로(`internal/enode` · `internal/contract` ·
`cmd/iapadapter` · `cmd/runctl`)만 공용 R/E 에서 다시 재고 나머지는 그대로 둔다.
(Q2=B) Inception 을 돌고 그대로 Construction 까지 한 손(taeels)으로 간다 —
유닛을 직렬로 민다. 담당 배정과 병렬 손은 없다.
(Q3=A) 이 팩을 먼저 끝내고 transcript 를 뒤에 둔다. 게이트의 `init` 줄은
`logs/` 가 아니라 사람이 직접 띄우는 경로로 잰다 (`scene-gates.md` 3절).
**Context**: INCEPTION · Requirements Analysis Step 6 게이트 통과

---

## Reverse Engineering — 부분 재측정 (Q1=B 의 실행)
**Timestamp**: 2026-09-11T13:38:14Z
**User Input**: (없음)
**AI Response**: 기준선 `06215ff` 와 HEAD 를 대서 네 경로만 다시 쟀다.
`internal/enode` 비테스트 소스 일곱이 바뀌고 셋이 새 파일(`policy.go` ·
`status.go` · `transcript.go`) · `internal/contract/advert.go` 에 Policy 와
Drain 어휘 셋 · `cmd/iapadapter` 와 `cmd/runctl` 은 무변경. 공용 R/E 의
`code-structure.md` · `component-inventory.md` 에 부분 재측정 절을 더하고
해당 줄을 고쳤고, `reverse-engineering-timestamp.md` 에 재측정 기록을 남겼다.
안 잰 자리의 낡음(라우트 15 -> 17 · 패키지 10 -> 12 · Go 파일 143 -> 190)은
본문을 안 고치고 관측값으로만 적었다.
**Context**: INCEPTION · Reverse Engineering 부분 재측정 완료

---

## Requirements Analysis — 산출물 완료
**Timestamp**: 2026-09-11T13:38:14Z
**User Input**: (없음 — 승인 대기)
**AI Response**: `requirements.md` 를 냈다. 깊이는 Comprehensive.
FR-1 ~ FR-7 이 팩 3.1 ~ 3.7 과 짝이고, 실측 교정 둘(2.1 알려진 키 검증은 이미
있다 · 2.2 계장 임시 디렉터리는 이미 있으나 `Fixed()` 가 그 경로를 모른다)과
보안 요구 둘(SEC-A 팩 tar 경로 검증 · SEC-B 팩 다이제스트는 기록이지 검증이
아님)을 더했다. security-baseline 열다섯 규칙의 적용 여부를 표로 냈다 —
`decisions.md` §3 이 다섯을 미리 적었고 이 회차가 SECURITY-05 · SECURITY-13
둘을 더한다. `decisions.md` 에 더할 행 셋을 7절에 모았다.
**Context**: INCEPTION · Requirements Analysis Step 7 ~ 9. 승인 대기

---
## Requirements Analysis — 승인
**Timestamp**: 2026-09-11T13:59:38Z
**User Input**: "workflow planning 하자. v3-run-harness-components"
**AI Response**: 다음 단계를 지시한 것을 Requirements Analysis 승인으로 읽었다.
승인의 귀결로 `requirements.md` 7절이 예약해 둔 행 셋을
`requirements/harness-components/decisions.md` 에 6절(2026-09-11 날짜 절)로
실었다 — ① 알려진 키 검증은 이미 있다 · ② 팩 tar 풀기의 경로 검증 · ③ 팩의
기대 다이제스트는 이월. 기존 1 ~ 5절의 번호는 안 건드렸다 (다른 문서가 4절과
5절을 번호로 참조한다).
**Context**: INCEPTION · Requirements Analysis Step 10 승인 · decisions.md 갱신

---

## Workflow Planning — 산출물 완료
**Timestamp**: 2026-09-11T13:59:38Z
**User Input**: (없음 — 위 지시로 착수)
**AI Response**: `inception/plans/execution-plan.md` 를 냈다.
실행 다섯 · 스킵 넷으로 갈랐다.

실행 — Application Design (`requirements.md` 8절의 D1 ~ D4 를 닫는다) ·
Units Generation (파일 행렬 필수) · Functional Design (유닛마다 · 새 파일 형식
넷) · Code Generation (유닛마다) · Build and Test (조각 게이트 CA0 ~ CA6 이 곧
시험 계획이다).

스킵 — User Stories (`scene-gates.md` 의 조각 일곱이 실행 명령으로 적힌 수용
기준이고 한 손 회차라 공유 이해의 값도 없다) · NFR Requirements
(`requirements.md` 4절이 Comprehensive 로 이미 닫았고 값을 안 바꾼다.
차단 확장의 집행은 스킵과 무관하게 단계마다 돈다) · NFR Design (앞을 건너뛰므로
넘길 패턴이 없다) · Infrastructure Design (새 포트 0 · 새 전송 0 · 새 라우트 0 ·
클라우드 자원 0).

값이 안 정해진 자리는 스킵과 함께 안 버리고 옮겨 적었다 — SEC-A 의 크기 · 개수
상한은 팩 유닛의 Functional Design 으로, 4.4 의 「광고 루프가 탐지를 직접 안
부른다」는 Application Design D2 로.

패키지 순서는 `internal/contract` 가 계약 키를 읽는 유닛보다 먼저 서고,
가짜 홈과 허용목록의 뼈대는 계약 키 없이 `internal/enode` 안에서 먼저 설 수
있다고 적었다. **유닛 순서는 안 박았다** — 팩이 유닛 분해를 안 주므로
Units Generation 이 낸다 (`CONVENTIONS.md` 3.1).

위험도 High · 되돌리기 Moderate · 검사 복잡도 Complex.
security-baseline 준수 요약을 7절에 실었다 — SECURITY-11 · SECURITY-05 준수,
나머지 열셋은 계획 문서라 N/A.
**Context**: INCEPTION · Workflow Planning Step 7 ~ 9. 승인 대기

---
## Workflow Planning — 승인
**Timestamp**: 2026-09-11T23:43:28Z
**User Input**: "승인. 커밋하고 Application Design 가자"
**AI Response**: 실행 계획을 커밋했다 (`88dc120`). Inception 은 회차 브랜치 위에서
직렬로 돌므로 유닛 브랜치를 안 땄다. 이어서 Application Design 을 착수한다.
**Context**: INCEPTION · Workflow Planning Step 10 승인 · 커밋

---

## Application Design — 계획 산출 · 답 대기
**Timestamp**: 2026-09-11T23:43:28Z
**User Input**: (없음 — 위 지시로 착수)
**AI Response**: `inception/plans/application-design-plan.md` 를 냈다.
질문을 짓기 전에 D1 ~ D4 가 걸리는 코드를 먼저 읽었고, 실측 넷을 1절에 적었다.

실측이 바꾼 것 셋.

D4 는 **답이 나왔다**. `claim.go:367` 이 트랜스크립트 링을 설정만 있으면 언제나
열지만, `claude.go` 의 `Argv` 가 `--output-format json` 이라 `system/init` 줄이
아예 안 나온다. 링은 나를 것이 없다. 게이트는 `scene-gates.md` 3절의 사람 경로
그대로다. 질문으로 안 내고 뒤집기 선택지만 Q5 로 물었다.

D2 는 **이름이 부딪친다**. `decisions.md` 2절의 권장값은 「`Detector` 인터페이스를
세운다」인데 `internal/enode` 에 이미 `Detector` 가 있고 그것은 종류가 아니라
시계다 (`ADR-068`). 그 권장이 지키려던 값(광고 루프가 탐지에서 안 멈춘다)은
`Detector.Capabilities()` 가 외부 프로세스를 안 띄우는 것으로 이미 서 있다.

D1 은 **인터페이스 모양보다 실패 규칙이 먼저다**. `runner.go:63` 의 계장
디렉터리가 if 블록 안에서만 살아 `tmp` 를 끌어올리는 것은 기계적이지만,
FR-1 의 「계장 실패와 독립」이 말하는 실패는 `Instrument` 실패이고
`MkdirTemp` 실패는 팩도 요구도 안 적었다. Q2 로 물었다.

넷째 실측 — `hook.go:346` 의 `gatewayAuthFields()` 가 `os.UserHomeDir()` 를
읽으므로 `CLAUDE_CONFIG_DIR` 을 바꿔도 게이트웨이 인증은 안 끊긴다. 가짜 홈이
끊는 유일한 인증 경로는 OAuth 의 `.credentials.json` 이고, 그래서 CA1 이 노드
둘에서 각각 돌아야 한다.

질문 다섯을 `[Answer]:` 태그로 냈다 — Q1 `Fixed` 인터페이스 · Q2 계장
디렉터리 생성 실패 시의 처리 · Q3 MCP 탐지가 앉는 자리 · Q4 허용목록과 팩의
소유자와 실패 전파 · Q5 출력 형식을 이 팩이 가져오나.
**Context**: INCEPTION · Application Design Step 1 ~ 6. 답 대기

---
## Application Design — 질문 다섯의 답
**Timestamp**: 2026-09-12T00:38:40Z
**User Input**: "전부 권장대로 할게"
**AI Response**: Q1 ~ Q5 를 모두 A 로 읽었다. 모호한 답이 없어 후속 질문을 안
냈다 (규칙 8 · 9 의 게이트 통과). 계획 문서 4절의 `[Answer]:` 다섯을 A 로 채웠다.
**Context**: INCEPTION · Application Design Step 7 ~ 9

---

## Application Design — 산출물 완료
**Timestamp**: 2026-09-12T00:38:40Z
**User Input**: (없음 — 승인 대기)
**AI Response**: `inception/application-design/` 에 다섯을 냈다 —
`components.md` · `component-methods.md` · `services.md` ·
`component-dependency.md` · 통합본 `application-design.md`.

설계의 뼈대는 **결정과 쓰기를 가른 것**이다 (Q4 = A). `resolveComponents` 가
파일을 하나도 안 만지고 무엇을 열지 정하며, `Instrument` 는 받은 것만 쓴다.
그 분리가 커버리지 80% 를 하네스 실행파일 없이 채울 수 있게 만든다.

산출물이 답 밖에서 더 정한 것 둘 — ① `Instrument` 의 오류를 등급으로 가른다
(`errComponents` 로 감싼 것만 치명. 훅 실패는 오늘처럼 보조). ②
`Instrument` 를 언제나 부른다 (오늘은 `os.Executable()` 이 비면 통째로
건너뛰는데, 그 경로로 가면 허용목록이 조용히 안 쓰인다). 둘 다 조용히 열리는
길을 막는 것이고 근거를 `application-design.md` 4절에 적었다.

측정이 하나를 줄였다 — **`cmd/runctl` 의 diff 가 0 이다.** `runctl example` 이
`contract.ExampleNames()` 로 임베드 FS 를 읽으므로 예시는
`internal/contract/examples/mcp.json` 하나로 족하고, `runctl schema steps` 는
구조체에서 뽑는다. 만지는 경로가 넷에서 셋으로 준다.

Q5 = A 가 짝 팩과의 접점도 줄였다 — `constraints.md` 접점 표의 여섯 줄 중
실제로 겹치는 것은 `runner.go` 하나뿐이다 (`claude.go` 의 `Argv` 를 이 팩이
안 건드리므로).

`decisions.md` 6절에 행 둘을 더했다 — ④ 계장 디렉터리를 못 만들면 단계 실패
(오늘 동작이 바뀐다) · ⑤ `Detector` 인터페이스를 안 세운다 (2절 권장값에서
벗어난다). 같은 절의 맺음 문단에 D1 의 답을 실었다.
**Context**: INCEPTION · Application Design Step 10 ~ 12. 승인 대기

---
## Application Design — 승인
**Timestamp**: 2026-09-12T00:41:18Z
**User Input**: "커밋해. 다음 세션에서 유닛 만들거야"
**AI Response**: 산출물 다섯과 `decisions.md` 행 둘을 커밋했다 (`3012a92`).
Units Generation 은 다음 세션에서 돈다. 그 세션이 바로 쓸 입력(만질 경로 셋 ·
착수 순서의 뿌리 · 빌드 시점 의존 하나 · 접점 하나)을 `aidlc-state.md` 의
Current Status 아래에 모았다.
**Context**: INCEPTION · Application Design Step 13 ~ 15 승인 · 커밋 · 세션 경계

---
## Units Generation — 착수
**Timestamp**: 2026-09-12T01:07:00Z
**User Input**: "승인. 넘어가자"
**AI Response**: Application Design 승인으로 읽고 Units Generation Part 1
(Planning) 을 시작했다. 질문을 짓기 전에 코드를 먼저 읽어 설계의 요약 표
(`application-design.md` 3절)와 실제 호출자를 대조했고 **표에 없는 파일 둘**을
찾았다 — `internal/enode/claim.go` (`Job` 이 조립되는 유일한 자리. `NodeMCP` 를
안 실으면 노드 선언이 조용히 안 실린다)와 `internal/enode/agent.go`
(`AgentParams` 와 `parseAgentParams` 가 사는 자리. `components.md` 1.6 은
적었고 요약 표만 빠졌다). `cmd/runctl` 의 소스 diff 0 은 맞으나
`cmd/runctl/shape_test.go` 의 `TestExamples_LintClean` 이 새 예시 위로 돌므로
행렬에 「닿는 시험」 열을 따로 둔다.
**Context**: INCEPTION · Units Generation Part 1 Step 1 ~ 3. 착수 전 실측

---
## 협업 규칙 정리 — Units Generation Q4 가 연 것
**Timestamp**: 2026-09-12T01:22:00Z
**User Input**: "대회장에서 협업 규칙 적다가 빠트린 것 같은데 지금은 대회 끝나서 정리가 필요하다"
**User Input**: "레이어링 구조가 아직 쓸만하니 고려해서 고쳐놔"
**AI Response**: Q4 가 낸 갈림(`CONVENTIONS.md` 3.1 은 유닛이 회차 브랜치로
모인다고, `CLAUDE.md` 는 PR 로 `main` 에 간다고 적었다)을 git 이력으로 판정했다.
`origin/v1-run-dhseo` 가 `unit/obs` · `panel` · `drain` · `transcript` 의 조상이고
PR 스물여섯이 전부 `main` 으로 갔다 — **두 문서가 각각 절반씩 맞았다.** 회차
브랜치도 Inception 이 닫히면 PR 로 올리는 것으로 고쳤다 (대회 때는 회차의
Inception 커밋이 첫 유닛 PR 에 실려 닿았고, 그 경로는 Construction 이 없는 회차에서
끊긴다). 레이어링은 유지하고 넓혔다 — 3.2 가 적던 flat 경로 셋은 저장소에 존재하지
않았고, 소유자별 루트로 다시 적으니 부딪히는 자리가 셋에서 `design/` 하나로 줄었다.
`roster` 5절의 병합 정책 사본을 없애고 정본을 가리키게 했다 (두 벌이었고 그래서
갈렸다). 파일 다섯을 커밋 `5cbf171` 로 넣었다.
**Context**: INCEPTION · Units Generation Part 1 Step 7 ~ 8 (Q4 의 모호를 규칙으로 해소)

---
## Units Generation — 답 다섯과 생성
**Timestamp**: 2026-09-12T01:34:00Z
**User Input**: "권장안으로 결정하고 닫는다."
**AI Response**: Q1 ~ Q5 를 전부 A 로 적었다. 모호한 답이 없어 후속 질문을 안 냈다
(규칙 7 · 8 의 게이트 통과). 4절 체크박스 열여덟을 전부 완료로 표시하고 산출물
넷을 냈다 — `unit-of-work.md` · `unit-of-work-dependency.md` ·
`unit-of-work-story-map.md` · `unit-of-work-file-matrix.md`.

유닛 다섯을 직렬로 낸다 — `isolation`(CA1) · `contract-vocab`(CA3 절반) ·
`advert`(CA2 · CA3) · `sources`(CA4) · `pack`(CA5 · CA6). 기능 일곱 중 사상되지
않은 것이 0 이고 담당 없는 게이트가 0 이다. 3.2 만 유닛 둘에 쪼개졌다 — 쓰는 자리
(U1)와 정하는 자리(U4)가 이 설계의 뼈대이기 때문이다. CA3 만 유닛 둘에 걸린다 —
400 과 lint 는 U2 가, requires 매칭과 422 는 U3 이 닫는다.

**파일 행렬이 설계 요약 표에 없던 파일 둘을 세웠다** — `internal/enode/claim.go`
(`Job{...}` 리터럴의 유일한 제품 코드 자리. `NodeMCP` 를 안 실으면 노드 선언이
조용히 안 실린다)와 `internal/enode/agent.go` (`AgentParams` 와 `parseAgentParams`).
`cmd/runctl` 의 소스 diff 0 은 맞으나 `shape_test.go` 가 새 예시 위로 돌므로
「닿는 시험」을 별도 열로 적었다.

생성이 답 밖에서 더 정한 것 셋 — ① `Local.MCP` 는 U3 이, `Job.NodeMCP` 와 `claim.go`
의 싣는 줄은 U4 가 낸다 (읽는 쪽 없는 필드를 먼저 안 세운다). ② `resolveComponents`
를 U1 이 빈 경우로 세워 `runner.go` 의 호출 자리가 U1 에서 최종형이 된다.
③ CA3 을 유닛 둘로 쪼개 적는다.
**Context**: INCEPTION · Units Generation Part 1 Step 7 ~ 11 · Part 2 Step 12 ~ 16

---
## 설계 검증 — 에이전트 일곱
**Timestamp**: 2026-09-12T02:10:00Z
**User Input**: "에이전트 풀어서 설계 검증 돌려."
**User Input**: "제안대로. 지금까지 결정 모두 제안대로 한 것 같은데. 걸어온 단계별로 검증 에이전트를 구동해라"
**User Input**: "문서 정합 수준을 검증하는 에이전트를 하나 생성. 이 에이전트는 자율적으로 서브에이전트를 소환해 문서 수준을 검증할 수 있어야 한다. 방금 네가 한 일을 에이전트 하나가 전담"
**AI Response**: 검증 축 셋(코드 대조 · 정본 대조 · 요구 팩과 내부 일관성)과
단계별 반대 심문 넷을 돌렸다. **사용자의 쏠림 지적이 입증됐다** — 질문을 내는
단계 셋을 스킵했고, Q1 은 규칙이 정한 선택지에만 반대 근거를 붙였고, Q3 은
「플래그 한 줄만 가져온다」는 네 번째 길을 안 보였고, PBT 는 선택지 B(Partial)가
사용자 앞에 놓인 적이 없다. Q3 을 A 에서 B 로 뒤집었다 — 근거 둘이 다 오독이었고
`ADR-035:246-248` 이 이 회차를 시점으로 지목한 문장이 안 읽혔다.
검증 도구를 `.claude/agents/aidlc-verify.md` 로 굳혔다.
**Context**: INCEPTION · Units Generation 승인 뒤 검증

---
## 문서 수정 1차
**Timestamp**: 2026-09-12T02:40:00Z
**User Input**: "/goal 방금 만든 에이전트가 승인 할 때까지 문서 수정을 반복해라"
**AI Response**: 1차 수정을 넣었다. `decisions.md` 6절에 행 다섯(⑥ ~ ⑩)을 더하고
⑤ 를 뒤집었다. 설계 결함 여섯을 `application-design.md` 4.3 ~ 4.6 으로 닫았다 —
팩 서버도 `agent.mcp` 필터를 탄다 · 팩이 노드 선언 이름을 덮으면 거절한다 ·
실패 등급의 기본이 치명이다(`errAux` 로 감싼 둘만 보조) · `Instrument` 의 불변식
둘 · 격리의 확실한 겹은 하나다 · 계장 보존 스위치. 훅 설정을 요구대로 가짜 홈
안으로 되돌렸다. Q3 = B 를 산출물 다섯과 유닛 문서 넷에 반영했다. 측정 오류 셋과
인용 주소를 고쳤고, 스킵 둘의 근거를 「규칙에서 벗어난다」로 정직하게 다시 적었다.
집행자 배정을 `unit-of-work-dependency.md` 8절로 세웠다.
**Context**: INCEPTION · 검증 루프 1차

---
## 문서 수정 2차 — 검증이 「넘기면 안 된다」로 판정한 뒤
**Timestamp**: 2026-09-12T04:30:00Z
**User Input**: (질문 셋에 대한 답) "스위치를 이 회차에서 뺀다" · "한 손을 유지하고
진행자가 집행한다" · "최소 형태로 돌린다"
**AI Response**: 1차 수정이 **연 구멍 둘을 되돌렸다.** ⑩ 계장 보존 스위치를 뺐다 —
`features.md` 3.1 의 「복사한 자격증명은 단계 끝에 지워진다」(SECURITY-12)를 뚫었기
때문이다. 대가로 눈 검증이 복제본을 잰다는 한계가 남고 그것을 명시로 적었다.
SECURITY-11 판정을 다시 냈다 — 4.6 의 「확실한 겹은 하나」가 측정이 아니라 가설이었고
(`features.md` 1.2 에 「가짜 홈만 -> 계정 커넥터 차단」 실측 행이 있다. 다만 로그아웃
홈이었다), 규칙 원문이 「단일 방어선이 아닐 것」이라 그 문장을 근거로 준수를 주장하면
자기모순이다. **겹을 둘로 두고 첫 겹을 재측정 항목으로 내렸다** — CA1 이 로그인된
홈에서 잰다.

집행자를 닫았다 — 한 손을 유지하고 **진행자가 CA1 ~ CA6 을 집행한다**
(`unit-of-work-dependency.md` 8.1 의 배정표). User Stories 를 최소 형태로 되살렸다 —
행복 경로는 안 쓰고 저작 경로와 오류 경로만 짓는다. 스토리 열 중 셋이 새 완료 조건을
낳았다. `decisions.md` 6절에 ⑪ ~ ⑭ 를 더했다 — 보조 등급은 하나다(기준 시각은
`writeStamp` 가 `Instrument` 앞에서 불려 구조적으로 안 닿는다) · `agent.pack` 이
미래 결합점이다 · 열거 면은 `runctl capabilities` 다 · `mcpUp` 이 못 잡는 것 셋.

⑥ 과 ⑦ 을 앞 단계 문서와 게이트로 내렸다 — `features.md` 3.2 · 3.3 · 3.6 ·
`scene-gates.md` 장면 ⑤ · CA1 · CA3 · CA5. 라우트를 **26** 으로 굳혔다 (24 도 틀렸다 —
`api.go` 의 `mux.Handle` 이 셋이다). `Fingerprinter.Probe` 에 logger 를 넣고 `repoFP` 의
`WorkspaceID` fallback 을 명시했다.
**Context**: INCEPTION · 검증 루프 2차

---
## 문서 수정 3차 — 인용하는 자리까지 함께 고친다
**Timestamp**: 2026-09-12T14:10:00Z
**User Input**: "플래그 한 줄만 U1 이 가져온다"
**AI Response**: 3차 검증이 구조적 원인을 짚었다 — **2차가 결정을 낸 문서만 고치고
그 결정을 인용하는 문서를 안 고쳤다.** `92697aa` 의 변경 목록에 회차의
`requirements.md` 가 없어서 결함 절반이 한 겹 아래로 내려갔고, 그래서 최소 수정
집합이 여덟에서 일곱으로밖에 안 줄었다. 이번에는 고친 결정마다 전수 `grep` 으로
인용 자리를 세어 함께 고쳤다.

FR-2 · 4.3 을 팩과 맞췄고, 팩 `decisions.md` 2절의 낡은 행 다섯에 「6절이
뒤집었다」 꼬리표를 달았다 — 권장값 표가 구현자에게 먼저 읽힌다. 보조 등급에
셋째 불변식(조기 반환 금지)을 세웠다. 보조 등급이 둘이 아니라 하나임을 하류에
내렸다. Q3 뒤집기를 상태 파일과 story map 에 내리고 옛 이름을 현재 값 자리에서
전부 걷었다. 라우트 세는 명령을 다섯 자리에서 걷었다.

**⑮ 를 더했다** — 사용자 결정으로 `Argv` 에 `--output-format stream-json --verbose`
한 줄을 U1 이 가져온다. 게이트 CA1 · CA4 · CA5 와 장면 ⑤ 가 전부 `system/init`
줄을 재는데 이 회차에 그 줄이 존재하지 않았고, 문서가 적어 둔 대안 둘이 다 안 섰다
(짝 팩은 병합 순서와 모순 · 프로브 산출은 내는 유닛이 0). **대가는 짝 팩과의
접점이 하나에서 둘로 느는 것**이고 그 값을 접점 표 · 행렬 · 상태에 함께 적었다.
게이트가 이제 복제본이 아니라 실물을 잰다.

User Stories Part 1 을 뒤늦게 닫았다 — 평가서와 생성 계획. **Part 2 가 먼저
돌았다는 절차 결함을 기록으로 남겼다.** 그리고 질문 형식 결함을 실측으로 적었다 —
`Other` 가 있던 질문 파일은 답이 권장을 벗어났고(B · B) 없던 파일은 전부 A 였다.
**「답이 열셋 전부 권장안이었다」는 앞선 진단은 거짓이다.**
**Context**: INCEPTION · 검증 루프 3차

---
## 문서 수정 4차 — 내 결정이 연 코드 구멍을 막는다
**Timestamp**: 2026-09-12T16:20:00Z
**User Input**: (4차 검증 판정 뒤 진행. ⑮ 은 사용자가 이미 결정한 값이므로
되돌리지 않고 안전하게 구현하는 쪽으로 갔다)
**AI Response**: 4차 검증이 **⑮ 이 코드 결함을 열었다**고 판정했고 직접 확인해
맞았다. `harness.go:131-143` 의 `lastJSONObject` 는 줄 단위가 아니라 출력 끝에서
유효한 마지막 JSON 객체를 집고, `:111-121` 의 switch 는 `Subtype` 과 `IsError` 만
본다. `claudeEnvelope.Type` 은 선언만 있고 읽는 자리가 0 이다. 그래서 `stream-json`
아래서 하네스가 중간에 죽으면 `{"type":"assistant",...}` 가 봉투로 오인돼
`ReasonOK` 로 떨어진다 — **크래시가 성공으로 봉인된다.** `harness.go:98-99` 의
「종료코드 0 을 믿지 않는다」가 무효가 되고, 같은 회차 ⑨ 가 세운 「실수는 닫히는
쪽으로」를 정면으로 어긴다.

**⑯ 을 더해 막았다** — `ParseClaude` 가 `type` 을 보고 `"result"` 가 아니면
`ReasonError` 다. U1 이 그 한 줄과 시험을 진다. ⑮ 의 근거 칸에 정본을 달았다 —
`agent-runtime.md:327-328` 이 「MVP 는 `--output-format json` 그대로」로 값을 박았고
R4 는 안 열렸으므로 **정본 MVP 값을 R4 없이 뒤집는 것**이다. 5절 되돌리기 목록에
올렸다.

SECURITY-03 · SECURITY-12 를 ⑮ 뒤 표면으로 다시 판정했다 — `logs/` 가 봉인인데
`stream-json` 이 도구 입력과 결과를 거기 싣는다. ⑩ 이 디스크 경계로 막은 값이
다른 문으로 나간다. 준수에서 **재판정**으로 내리고 U1 의 FD 로 넘겼다.

**방식을 바꿨다.** 4차가 「이 셋을 안 바꾸면 5차도 일곱을 낸다」로 진단했다 —
수정 회차가 새 결정을 낳고, 전수 grep 이 손과 기억으로 돌고, 산출물이 팩 값을
다시 적는다. `decisions.md` **6.1 에 뒤집힌 옛 문자열의 열쇠 표**를 세우고 CA0 가
그것을 돌게 했다. `CONVENTIONS.md` 1.4 의 「세는 것으로 지킨다」 그대로다.
**Context**: INCEPTION · 검증 루프 4차

---
## 문서 수정 5차 — 내가 만든 장치가 거짓 초록을 냈다
**Timestamp**: 2026-09-12T06:10:00Z
**User Input**: (5차 검증 판정 뒤 진행. 사용자 결정 대기 항목 없음)
**AI Response**: 5차 검증이 **6.1 열쇠 표가 거짓 초록을 낸다**고 측정으로 증명했다.
열한 열쇠 중 둘이 저장소 역사에 존재한 적 없는 문자열이었다 — **백틱이 빠져서**
`git log -S` 가 그 열쇠를 세운 커밋 하나만 낸다. **손과 기억을 없애려고 만든
장치가 손과 기억으로 쓰였다.** 그 둘이 덮어야 할 실물 아홉 자리가 살아 있었다.

표를 다시 세웠다 — 규칙 셋을 박았다. ① 열쇠를 지어내지 않는다 (`git log -S` 가
두 커밋 이상을 내야 실재한 열쇠다) · ② 어형 변이를 함께 적는다 · ③ **집행자는
사람이다** (도는 스크립트가 없는 채로 기계라고 적으면 그것이 또 거짓 초록이다).
CA0 의 기계 검사에서 뺐다. 오래 가는 답이 스크립트라는 것과 그 자리(`scripts/`)를
적었다.

⑯ 을 값으로 못 박았다 — **switch 앞에서** `type` 을 보고, `ReasonError` 때도
`Turns` · `CostUSD` · `Session` 을 채운다. switch 뒤에 두면 `subtype` 에 `token` 이
든 사건이 `max_tokens` 로 떨어져 `Completed()` 가 참이 된다. U1 의 시험 입력도
고쳤다 — **줄 경계에서 끊긴** stdout 이어야 ⑯ 을 잰다. 객체 중간에서 끊으면
오늘 코드도 이미 `harness_error` 라 ⑯ 을 안 재는 시험이었다.

**⑰ 을 더해 SECURITY-03 을 지금 판정했다** — 비준수로 적고 받아들인다. 「재판정
대기」는 규칙이 허용하는 셋(준수 · 비준수 · N/A)에 없다. 표면이 이 팩보다 먼저
있고 ⑮ 이 넓힌다. 닫는 길 둘을 5절 되돌리기로 올렸다. SECURITY-12 의 재판정은
**규칙 오태그였으므로 철회**했다 — 그 규칙은 사용자 인증이고 계장의 자격증명
파일 복사는 거기 안 걸린다.

⑮ 을 팩 정본 둘에 내렸다 (`constraints.md` 접점 표 · `features.md` 3.7). 접점 표가
급했다 — 파일 행렬이 그것을 정본으로 가리키는데 U1 의 `Argv` 작업을 짝 팩에
배정하고 있었다. 살아남은 어형을 전부 걷었다 (접점 「하나」 · 「봉투가 안
흔들린다」 · 사람 경로 · `scene-gates` 의 자기모순 한 칸).
**Context**: INCEPTION · 검증 루프 5차

---
## 문서 수정 6차 — 코드 층이 처음으로 깨끗해졌다
**Timestamp**: 2026-09-12T07:40:00Z
**User Input**: (6차 검증 판정 뒤 진행. ⑰ 만 사용자 결정으로 남긴다)
**AI Response**: 6차가 **넷**을 냈고 그중 셋을 닫았다. 최소 수정 집합 추이는
여덟 → 일곱 → 일곱 → 다섯 → 넷이고 **이번에 코드가 바뀌는 자리가 0 이 됐다.**

**내가 세 문서에 적은 거짓을 고쳤다** — ⑰ 이 「닫는 길 둘을 5절 되돌리기 목록에
올린다」고 했는데 5절에 없었다. `ADR-005` 항목을 실제로 올려 참으로 만들었다.

게이트 경로를 하나로 맞췄다 — `requirements.md` §5.1 이 「`logs/` 가 아니라 사람이
직접 띄워서 읽는다」로 ⑮ 의 정반대를 지시하고 있었고, **출처로 인용한
`scene-gates.md` 3절이 지금 그 경로를 명시로 배제한다.** FR-7 의 「짝 팩의 몫이고」도
함께 고쳤다. 5차 항목 3 의 남은 둘이었다.

6.1 장치를 집행 가능한 형태로 세웠다 — **검사 범위**를 못 박고(표 자신 · `audit.md` ·
`aidlc-state.md` 의 경위 · 계획 · 제품 코드 · 다른 회차 · 서브모듈을 뺀다), 이번에
뒤집은 값 넷에 열쇠를 더했다(접점 「셋」 · SECURITY-03 준수 · SECURITY-12 오태그 ·
CA0 기계 검사). 집행자를 사람으로 내린 것을 `scene-gates.md` CA0 와 `aidlc-state.md`
에도 내렸다 — 커밋 메시지만 「뺐다」고 하고 팩 정본은 안 따라왔었다.

행렬에 `harness_test.go` 행과 U1 의 `ParseClaude` 를 세웠다. 다음 단계가 U1 의
Functional Design 이라 그것이 먼저 필요하다.

**⑰ 은 문서 수정으로 못 닫는다.** `security-baseline.md:21` 이 비준수를 blocking 으로
규정하고 `:11-12` 가 「`Continue to Next Stage` 를 제시하지 말라」고 적는다.
「받아들인다」는 규칙 어휘가 아니라 이 회차가 지어낸 것이다. 규칙이 주는 길 셋 중
어느 것도 에이전트가 혼자 정할 값이 아니라 **미결로 적고 사용자에게 묻는다.**
**Context**: INCEPTION · 검증 루프 6차

---
