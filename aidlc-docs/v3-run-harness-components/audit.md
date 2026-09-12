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
