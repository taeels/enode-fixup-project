# AI-DLC Audit Log

모든 사용자 입력과 결정을 시간순으로 기록한다. 사용자 원문은 요약하지 않고 그대로 싣는다.
타임스탬프는 `date -u +%Y-%m-%dT%H:%M:%SZ` (ISO 8601 UTC).

---

## Workspace Detection — 초기 사용자 요청
**Timestamp**: 2026-09-08T05:36:09Z
**User Input**: "enode 에 기능을 더한다. 요구사항은 이미 문서다.

  requirements/decisions.md        정해진 값과 근거
  requirements/enode-features.md   기능 일곱. 목적 · 요구사항 · 수용 기준
  requirements/scene-gates.md      장면 조각 게이트. 이것이 수용 기준이다
  requirements/canon.md            설계 정본과의 연결. 어느 ADR 을 읽을지 적혀 있다
  requirements/constraints.md      안 만드는 것과 구조 불변식
  design/README.md                 화면 아홉 장의 규칙
  design/exports/*.png             그 화면의 그림. .pen 은 못 여니 이것을 본다

설계 정본은 서브모듈 enode-design/ 이고, 어긋나면 protocol/INVARIANTS.md 가
이긴다. aidlc-docs/ 는 비어 있다.

값은 decisions.md 가 다 갖고 있고 미정은 하나뿐이다 (enode-features.md 5절).
그래서 물을 것이 거의 없다 — 물어야 한다면 계약(함수 서명 · 파일 경로 ·
응답 모양)에 걸리는 것만, 단계마다 셋까지. 권장값을 벗어나야 하면 묻지 말고
근거를 decisions.md 에 행으로 적어라.

유닛 분해는 팩이 주지 않는다. Units Generation 이 정하되 유닛마다 어느 유닛에
의존하는지를 적어라 — Construction 을 그 그래프대로 병렬로 돌린다. 화면이 있는
유닛은 완료 조건에 그 화면의 버튼을 전부 나열한다.

완료는 scene-gates.md 가 잰다. 3절의 명령을 실제로 돌리고, 화면이 있는 조각은
2.1 의 눈으로 보는 항목을 따로 본다 — API 검증은 화면 없이도 초록이 된다.
조각이 초록이 아니면 그 유닛을 병합하지 않는다.

커밋은 토픽 브랜치 위에서 한다. Inception 은 v1-run-<이름> 하나에서 직렬로,
Construction 은 유닛마다 unit/<유닛> 을 따서 병렬로 간다. 단계 승인마다
커밋하고, 유닛은 조각이 초록인 뒤에 v1-run 으로 병합한다.

병렬이 부딪히는 자리가 셋이다. 유닛 브랜치는 여기를 안 건드린다.
  aidlc-docs/audit.md · aidlc-state.md   각각 한 장이다. 진행자가 합친다
  design/                                 그림은 진행자만 고친다
유닛 둘이 같은 파일을 만져야 하면 (internal/api/api.go 의 등록 줄이 그 자리다)
파일 행렬에 표시해라. 그 파일은 진행자가 직렬로 병합한다.

타임스탬프는 date -u +%Y-%m-%dT%H:%M:%SZ 로 적는다."
**AI Response**: 요청을 기록했다. 요구 팩 다섯 파일과 `design/README.md`, 화면 그림, AI-DLC 규칙(v1.0.1)을 읽었다. Workspace Detection 을 실행했다 — 브라운필드 확정, `aidlc-docs/` 비어 있음, RE 산출물 없음. Inception 실행 브랜치 `v1-run-dhseo` 를 `main` 에서 땄다(CONVENTIONS.md 3.1). `aidlc-state.md` 를 만들고 다음 단계로 Reverse Engineering 을 시작했다.
**Context**: INCEPTION - Workspace Detection. 초기 요청 로그. 브라운필드 감지.

---

## Workspace Detection — 확장 opt-in (사전 확정)
**Timestamp**: 2026-09-08T05:36:09Z
**User Input**: (요구 팩 `decisions.md` §1 이 착수 전에 확정 — 2026-09-04 사용자 결정)
**AI Response**: 확장 3종의 enable 상태를 `aidlc-state.md` 의 Extension Configuration 에 기록했다 — security-baseline 켬, resiliency-baseline 끔, property-based-testing 끔. Requirements Analysis Step 5.1 에서 재확인한다. security-baseline 전문을 로드했다(켜져 있으므로).
**Context**: INCEPTION - Workspace Detection. 확장 결정은 요구 팩이 이미 닫음.

---

## Reverse Engineering — 완료 · 검증
**Timestamp**: 2026-09-08T07:17:58Z
**User Input**: (없음 — RE 는 분석 단계라 착수에 사용자 입력이 필요 없다. 완료 게이트만 승인 대상)
**AI Response**: 브라운필드 RE 를 워크플로로 돌렸다 — 리더 7(Mediator API · store/DB · match/contract · node daemon · node control · build/CI · enode-design 정본)이 각 서브시스템을 훑으며 팩이 인용한 코드 주장을 검증했고, 라이터 6 이 산출물 8개를 규약(한국어 산문 · 장식 문자 금지)대로 썼다. 13/13 에이전트 성공 · 오류 0. 검증: `emphasis-check.py` 통과(위반 0) · U+2605/이모지 없음 · 코드펜스 짝 맞음 · Mermaid 4개 유효 · 라우트 15개가 `internal/api/api.go:61-75` 원본과 정확히 일치. 산출물은 `aidlc-docs/inception/reverse-engineering/` (business-overview · architecture · code-structure · api-documentation · component-inventory · technology-stack · dependencies · code-quality-assessment). 파일은 이미 커밋 `1fe2145` 가 실었다(다른 세션이 "Adding R/E" 로). 이 항목에서 RE 를 완료로 표시하고 timestamp 파일을 더한다. 병렬 사실: 이 브랜치에 shonsin 이 `cc04508`(S6·S7 3D 탑뷰 시안, `.pen`)를 붙였다.
**Context**: INCEPTION - Reverse Engineering 완료. 다음 단계 Requirements Analysis.

---

## Requirements Analysis — 현장 워크숍 요구 추가 (대회 데모로 전환)
**Timestamp**: 2026-09-08T07:17:58Z
**User Input**: "pen.dev 업데이트 작업은 shonsin이 착수했다. 우리는 요구사항 업데이트 한다. led 데모는 취약하다. h/w가 필요한 개발환경이 없는 사람이 많을 수 있음.
현장에서 워크숍 진행해서 다음과 같이 결정했다.

추가 요구사항을 만든 이유 - 대회장에서 알게 된 사실
```
평가 구분 : AI 40%, 상호 투표 60%
6개 항목 100점 만점.
AI-DLC 활용도 25점
문제 정의 및 해결 20점
창의성 15점
완성도 15점
사용성 15점
유지보수성 및 보안 10점
AI 에이전트 점수와 사람 심사위원 점수가 독립적으로 산출됩니다. 최종 점수는 이 AI 심사가 40%, 참가자 상호평가가 60%입니다. 심사위원은 따로 점수를 매기지 않고, AI 가 낸 필러 점수를 필러당 ±5점 안에서 보정합니다.

참가자 투표
운영진이 투표를 열면 계정마다 투표 5개가 생깁니다. 갤러리를 돌아다니며 다른 팀 프로젝트에 나눠 붙이면 되고, 한 팀에 한 개까지, 자기 팀에는 투표할 수 없습니다. 한 투표는 투표가 열려 있는 동안 다시 취소할 수 있습니다.
가장 많은 표를 받은 팀이 투표 몫 60점을 모두 가져가고, 나머지 팀은 그 팀 대비 득표 비율만큼 받습니다. 누가 몇 표를 받았는지는 진행 중에 공개되지 않습니다.
```

첫 방문때 온보딩 시나리오를 강제하자.
- 보드를 enode를 써서 효율적으로 운용할 수 있다는 사실을 각인시킬 수 있음. (이 시스템이 왜 필요한지)
  - 설명 방식을 애니메이션을 가미한 카드 나열로 한다.
- 첫 방문 이후로는 나타나지 않음. 바로 데모용 mediator 현황판으로 이동.

mediator 현황판을 대중에게 공개되는 demo page로 전환한다.
- Mediator 현황판을 데모용으로 개조한다. 그래서 \"Guest 로그인\" 버튼이 필요함. (지금은 Token 입력이 필요함) -> Guest 로그인하면 랜덤한 이름이 선택된다. (멍청한 어피치 등등 2단어)
- 여기도 첫 방문한 경우 가이드투어 진행. 웹캠, 노드 현황판, 현재 작업 목록, \"새 작업\".
- \"새 작업\" 을 누르면 화면을 덮는 모달 창에서 run을 제출할 수 있음.
- 메인 함대 현황에 pen.dev에서 개발중인 3d 뷰를 적용
- \"새 작업\"이 제출되면 화면은 \"작업 그래프\" 화면으로 넘어감
- \"작업 그래프\" 에도 pen.dev에서 만든 3d 뷰를 적용함.
- \"작업 그래프\" 화면이 run 목록을 가리지 않게 함.

시연용 시나리오
- 참가 enode
  - rpi - windows pc(nacl1119@windows). rpi와 ssh로 연결되어 있음.
  - mac (shinson@mac) - higgsfield mcp를 가진 환경. run을 요청한 사람의 이름을 \"xxx님 환영합니다\" wav로 추출함. 그리고 봉인에 artifact로 전달함.
- 시연용 mediator 현황판에서 run은 고정된 시나리오만을 낼 수 있음.
  - 1. Rpi \"LED blink pattern toggle. 1 초 heart beat, persistent on 두개
  - 2. Rpi 스피커를 통해 mac에서 전달받은 음원 재생
     - rpi 안에 음원이 없어서, shon-sin 노트북에서 음원을 받아와야 한다.
- run을 제출한 사람의 이름을 적어야 함.
  - 이러면 현황판에서 작동중인 RUN 카드에 누가 낸 run인지 보여야 함.
- 웹캠으로 실물 보드 모습을 비춰줘야함. 웹캠은 고정된 노트북에서 화상을 전송함. webrtc 셋업이 필요?? 검토좀.
- 웹캠은 mediator 현황판 좌하단에 floating 형태로 보여지며 resize 가능함. (현황판 레이아웃에 영향을 미치지 않음.)"
**AI Response**: 원문을 그대로 기록했다. 이것은 요구 팩에 여덟째 축(시연)을 더하는 큰 범위 확장이고 dhseo 장면에서 대회 데모로의 전환이다. 기존 팩·정본·제약과 부딪히는 자리를 짚고(중앙 읽기전용 반전 · security-baseline 차단성 · GET /v1/runs 최소열 확장 · constraints §5/§6 의 중계 금지), 웹캠 전송을 검토했으며, 계약·범위·보안 태세에 걸리는 갈림 넷을 `aidlc-docs/inception/requirements/requirement-verification-questions.md` 에 담았다. 답을 받은 뒤 `decisions.md` 에 날짜 절 · `enode-features.md §4` 범위 · `scene-gates.md` 의 새 CP · `constraints.md` 로 태운다. 채점 맥락(AI-DLC 활용도 25 · 사용성 15 · 완성도 15 · 창의성 15 · 유지보수성/보안 10, 상호투표 60%)을 우선순위 판단에 넣었다.
**Context**: INCEPTION - Requirements Analysis. 사용자 요구 추가. 질문 파일 대기.

---

## Requirements Analysis — 질문 답변 (대회 데모 전환 확정)
**Timestamp**: 2026-09-08T07:33:17Z
**User Input**: "답했다." (`requirement-verification-questions.md` 에 기입 — Q1: A + "시연 시나리오는 두개이다. 1: LED 토글, 2: run제출자 환영합니다 wav 재생." · Q2: A · Q3: A · Q4: A · Veto 전부 비움)
**AI Response**: 답을 분석했다 — 모순·모호 없음. Q1 의 시나리오 둘 클라리피케이션은 프레이밍(LED = 시나리오 ①, welcome-wav = 시나리오 ②)과 일치한다. 전환을 팩에 태웠다 — `decisions.md §8`(닫은 값 넷 + 진행자 기본 결정 + 수락 위험 SECURITY-08) · `enode-features.md §1.5·§3.4·§4`(가치 고정점 전환 · 시연 다섯 · 범위) · `scene-gates.md §5`(대회 데모 장면 + CP8~CP11) · `constraints.md`(데모 표면 경계 · 수락 위험 · out-of-band 웹캠). AI-DLC 산출물 `requirements.md` 를 냈다. Requirements Analysis 완료 게이트로 간다.
**Context**: INCEPTION - Requirements Analysis 완료. 다음 Workflow Planning.

---

## Requirements Analysis — 노트 파일에서 추가 요구 발견 (pen.dev 노드 강조)
**Timestamp**: 2026-09-08T07:33:17Z
**User Input**: (루트의 untracked `시연시나리오변경.md` 끝에 채팅에 없던 블록 — "pendev 요구 추가. 각 노드를 나타낼때 강조필요한 내용. capability · label · sandbox")
**AI Response**: 채팅 메시지에는 없고 팀 노트 파일에만 있던 요구다. 노드 표현(카드 C1 · 3D 함대뷰 S6)이 `capability`·`label`·`sandbox` 셋을 강조하는 것으로 `enode-features.md` 3.4.2 에 태웠다. `sandbox` 출처가 계약에 걸려 `decisions.md` §8.5 에 미정으로 적었다. 채팅에 없던 것이라 완료 게이트에서 사용자 확인을 요청한다.
**Context**: INCEPTION - Requirements Analysis. 파생 요구 반영 · 확인 대기.

---

## Requirements Analysis — 승인 (게이트 통과)
**Timestamp**: 2026-09-08T07:50:41Z
**User Input**: "샌드박스는 표시만 할게. 승인한다. 커밋후 푸시해. 다음 세션에서 이어서 간다."
**AI Response**: sandbox 를 표시만으로 닫았다(`decisions.md §8.5`) — 격리 구현은 범위 밖, 화면은 있는 값을 보여만 준다. Requirements Analysis 를 완료로 표시했다. `requirements/` 팩 4파일 + `aidlc-docs/` 를 커밋하고 `origin/v1-run-dhseo` 로 푸시했다(사용자 명시 허가). 다음 세션은 Workflow Planning 부터 — User Stories SKIP 권장. 세션 핸드오프를 `aidlc-state.md` 에 남겼다.
**Context**: INCEPTION - Requirements Analysis 완료·승인. 다음 Workflow Planning. 세션 종료.

---
