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

## 새 토픽 브랜치 — 갈라진 지점 (사람이 손으로 적음)
**Timestamp**: 2026-09-08T08:07:14Z
**User Input**: (분기 명령 — 그대로)

```
git fetch origin v1-run-dhseo
git switch -c v1-run-dhseo-cardnews 1fe2145
```

**AI Response**: `v1-run-dhseo-cardnews` 를 `v1-run-dhseo` 의 팁(`da73c53`,
대회 데모 전환)이 아니라 그보다 앞선 `1fe2145`(Adding R/E, 대회 데모 전환
이전)에서 땄다. `git log --oneline` 로 확인 — `1fe2145` 는 `06215ff` (병합
커밋) 바로 다음이고, `da73c53` 는 그 앞선 요구 팩 회차가 대회 데모로 전환한
자리라 이 브랜치는 그것을 안 딛는다. 나중에 `da73c53` 계열과 합칠 때 이
사실이 필요하므로 여기 첫 항목으로 남긴다. Reverse Engineering 산출물 아홉은
`1fe2145` 시점에 이미 있고 다시 돌리지 않는다.

**Context**: 새 실행 브랜치 착수. 분기점 기록 — 병합 시 참조용.

---

## 여덟째 기능 요청 — 온보딩 카드뉴스 + Guest Login 진입점
**Timestamp**: 2026-09-08T08:07:14Z
**User Input**: "프로젝트 위치 C:\Users\MSI\workspace\enode-fixup-project

enode-fixup-project 의 v1-run-dhseo 브랜치, 커밋 1fe2145(Adding R/E, 대회
데모 전환 이전) 지점에서 새 토픽 브랜치를 딴다:

  git fetch origin v1-run-dhseo
  git switch -c v1-run-dhseo-cardnews 1fe2145

Reverse Engineering 은 이미 이 지점에 끝나 있다(다시 돌지 마라). 이 지점의
requirements/ 는 아직 대회 데모 전환 이전 — dhseo 장면 기준 일곱 기능
팩이다(enode-features.md 3판 · decisions.md 1~7절 · scene-gates.md CP0~CP7).
v1-run-dhseo 팁에는 그 뒤(da73c53)에 다른 세션이 대회 데모 전환을 얹었지만
이 브랜치는 그것을 안 딛는다.

기존 요구 팩(requirements/enode-features.md 등 다섯 파일 · design/)은 그대로
입력이다. 거기에 아래 여덟째 기능을 더해서 Requirements Analysis 부터
진행해라. 번호는 3.1.4 든 다른 자리든 이 단계가 정해라.

디자인이 사전에 없다 — 이번 라운드가 직접 만든다
  기존 아홉 장은 design/enode-ux.pen 으로 대회 전에 사람이 이미 그려서
  요구 팩의 입력으로 들어왔다. 카드뉴스는 그런 사전 디자인이 없다.
  Requirements Analysis 다음부터 — Application Design 이든, 카드뉴스를
  맡는 유닛의 Functional Design 이든 — 화면 디자인 산출물(레이아웃 · 카드
  순서 · 전환 방식)을 이 실행이 스스로 내고, 그 위에서 실제 웹페이지
  구현(Code Generation)까지 이어가라. 디자인을 외부에서 받아오는 단계가
  없다. 산출물 형식은 design/README.md 같은 규칙 문서 + 와이어프레임
  수준이어도 된다. Penpot 을 새로 요구하지 않는다.

---

새 기능 — 온보딩 카드뉴스 + Guest Login 진입점

목적
개발보드를 다루지 않는 방문자도 enode 가 왜 필요한지 배경부터 이해하게
한다. 대회·행사의 관람객이 대상이다.

배경 시나리오 (카드 카피의 소재. 문안 자체는 이 실행의 Functional Design 이
직접 확정한다)
  1. 갤럭시 S26 까지 나온 지금, 갤럭시 S2 에서 난 이슈를 재현·수정해야 하는
     상황 — 필요한 구형 보드가 희귀하고 특정 자리에 한 대뿐이다
  2. 여러 개발보드를 동시에 써야 하는 업무에서, 동료의 보드를 자유롭게 빌려
     쓸 수 있으면 훨씬 빨리 끝난다
  같은 결의 시나리오를 더 넣어도 된다. 요구 팩은 소재만 준다.

진입 흐름 (사용자 확정 — 그대로 값으로 박는다)
  Guest 로그인 버튼을 누른다  ->  카드 시퀀스(온보딩)가 뜬다  ->  카드를
  닫는다(끝까지 넘기거나 닫기)  ->  데모용 enode 현황판 자리로 이동한다.

  카드 시퀀스는 페이지 로드 시점이 아니라 Guest 로그인 클릭이 트리거다.
  재방문(localStorage 에 첫 방문 완료 기록)시 Guest 로그인을 누르면 카드
  시퀀스 없이 곧장 데모용 현황판 자리로 간다.

범위 경계 — 이번 라운드가 반드시 지킬 선
  - 관리자 화면(S0 Mediator 토큰 입력 -> S1 현황판)은 손대지 않는다.
    기존 보안 요구사항 그대로 유지 — 개정 없음. Guest Login 은 그 옆에
    서는 완전히 별도의 진입 경로다
  - 카드뉴스 화면은 독립된 화면이다. 데모 현황판 구현에 카드뉴스가
    조건부로 섞여 들어가는 구조는 금지한다
  - Guest Login 버튼 + 카드뉴스 화면 + (카드뉴스를 닫았을 때 도착하는)
    데모 현황판으로의 이동/라우팅까지가 이번 라운드다. 그 데모 현황판
    화면 자체의 내용(무엇을 보여주는가)은 이번 스코프가 아니다 — 지금은
    자리표시(placeholder)로 두고 다음 라운드가 채운다

기능 요구사항
  - Guest Login 버튼은 새로 만든다 — 기존 S0(관리자 토큰 입력) 화면과는
    별개 자리다. 정확한 위치(랜딩 화면에 나란히 두는지, 별도 화면인지)는
    이 실행의 디자인 몫이다
  - 카드 전환에 애니메이션을 쓴다. 구체 효과는 디자인 몫이다
  - 카드뉴스를 다시 보고 싶을 때의 경로 — 미정. 이 실행이 정한다

UI/UX 요구사항
  - 카드뉴스 화면은 internal/api/ui 가 담당하는 정적 영역에 살아도 되지만,
    컴포넌트/번들 단위로 데모 현황판과 분리한다 (구조 불변식 —
    internal/api/ui -> internal/store 금지, 화면은 정적 파일이라는 기존
    규칙과 별개로 이번 요구가 추가로 거는 제약)
  - 새 백엔드 라우트는 최소화한다. Guest Login 이 세션/식별을 어떻게
    표시할지(랜덤 이름 등)는 이 실행이 정하되, 그것이 인증으로 격상되지
    않게 한다 — ADR-015 의 principal = 식별, 인증 아님 원칙을 유지한다

데이터 관리
  새 저장소 없음. 클라이언트 로컬 저장소(첫 방문 판정)만.

수용 기준 초안
  - 랜딩 화면에 Guest Login 버튼이 있고, 기존 Mediator 토큰 입력과는
    분리돼 있다
  - Guest Login 을 처음 누르면 카드뉴스 화면이 뜬다(데모 현황판이 같이
    뜨거나 뒤에 걸쳐 있으면 안 된다)
  - 카드를 끝까지 넘기면 데모 현황판 자리로 이동한다 — 이 시점부터 카드
    뉴스 화면의 어떤 요소도 안 남는다. 그 자리의 실제 내용은 placeholder
    로 남아도 통과다
  - 재방문 시 Guest Login 을 누르면 카드뉴스 없이 곧장 그 자리로 간다
  - 관리자 흐름(Mediator 토큰 -> S1)은 기존 그대로 동작한다 — 회귀 없음

---

이 여덟째 기능을 기존 일곱과 함께 Requirements Analysis 에 넣어라. 확장
opt-in(security-baseline 켬)은 이미 결정돼 있으니 이 새 표면에도 그대로
적용된다.

브랜치·감사 로그·게이트 관례는 v1-run-dhseo 킥오프 때와 같다(audit.md 의
최초 항목을 읽어라) — 여기서 다시 안 적는다. 다만 이 브랜치는 v1-run-dhseo
의 팁이 아니라 1fe2145 에서 갈라졌다는 사실을 audit.md 첫 항목에 명시해라
— 나중에 대회 데모 전환(da73c53)과 합칠 때 그 기록이 필요하다."

**AI Response**: 요청을 기록했다. 이 세션은 원격 · 무인터랙티브 실행이다 —
질문 파일을 내고 사람의 `[Answer]:` 입력을 기다리는 v1 의 표준 게이트 방식이
안 통한다. 그래서 이번 라운드는 진행자(coordinator)와 실행자를 겸해 각 단계의
질문에 스스로 답하고 그 근거를 `decisions.md` · 각 단계 산출물에 결정으로
남기는 방식으로 돈다 — `decisions.md` 가 이미 앞 회차들에서 쓰던 「진행자
결정 + 근거」 패턴 그대로다. 사람이 다시 열어 반려할 수 있도록 매 단계
산출물과 근거를 빠짐없이 남긴다. 이 적응을 아래에 별도 항목으로 기록한다.

**Context**: INCEPTION - Requirements Analysis 착수. 여덟째 기능 요청 원문.

---

## 실행 방식 적응 — 무인터랙티브 세션
**Timestamp**: 2026-09-08T08:07:14Z
**User Input**: (없음 — 실행 환경의 사실)
**AI Response**: v1 규칙의 각 단계는 질문 파일을 내고 사람의 답을 기다리는
게이트를 명시한다 (`requirements-analysis.md` Step 6 의 게이트 등). 이 세션은
그 대기가 응답을 받을 수 없는 원격 실행이다. 그래서 다음으로 적응한다 —

  1. 단계마다 여는 질문은 산출물 안에 「질문과 그 답」으로 남긴다 (별도
     대기 파일을 내고 멈추지 않는다)
  2. 계약(파일 경로 · 함수 서명 · 응답 모양)에 걸리는 결정은 `decisions.md`
     에 행으로 적어 근거를 남긴다 — 팩의 기존 관례와 같다
  3. 각 단계 완료 메시지의 승인 게이트는 자기 승인으로 남기고 「자기 승인 —
     무인터랙티브」로 표시한다. 사람이 나중에 반려하면 그 단계로 돌아온다
  4. Extension 강제(security-baseline)와 구조 불변식(constraints.md) 은
     그대로 차단성으로 지킨다 — 이것은 적응 대상이 아니다

**Context**: 방법론 적응. `aidlc-docs/README.md` 의 협업 관례와 어긋나지
않는다 — 여기는 병렬 유닛이 아니라 단일 실행자가 전 단계를 순서대로 도는
경우다.

---

## Requirements Analysis 완료 — 자기 승인
**Timestamp**: 2026-09-08T08:20:00Z
**User Input**: (무인터랙티브 — 2절 적응에 따라 자기 승인)
**AI Response**: `aidlc-docs/inception/requirements/requirements.md` 를 냈다.
기존 일곱 기능은 입력으로만 참조하고(다른 유닛 몫), 이 실행이 실제로 미는
것은 3.4.1 하나임을 명시했다. 완결성 점검(기능·비기능·시나리오·비즈니스·
기술·품질 속성 여섯 카테고리)에서 뜬 질문 둘(카드 매수, 도중에 닫을 때의
동작)을 문서 안에서 스스로 답했다 — 둘 다 계약을 안 바꾸는 낮은 위험도
질문이고 근거를 남겼다. 확장 opt-in(security-baseline 켬)을 재확인했다.
**자기 승인 — 무인터랙티브.** Requirements Analysis 완료로 표시하고
User Stories 로 넘어간다.
**Context**: INCEPTION - Requirements Analysis 완료. `aidlc-state.md` 갱신.

---

## User Stories 완료 — 자기 승인
**Timestamp**: 2026-09-08T08:25:00Z
**User Input**: (무인터랙티브 — 자기 승인)
**AI Response**: 3.4 만 다룬다 — 워크플로 기준으로 「신규 사용자 대면
기능」이라 실행 대상이다. 페르소나 하나(관람객)와 스토리 다섯(US-1 ~
US-4 가 CP8 을 구성하고 US-5 는 관리자 흐름 회귀 확인)을 냈다. 기존
일곱 기능의 스토리는 이 실행의 스코프 밖이라 만들지 않는다. **자기
승인 — 무인터랙티브.** Workflow Planning 으로 넘어간다.
**Context**: INCEPTION - User Stories 완료.

---
