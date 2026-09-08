# Requirements Analysis — v1-run-dhseo-cardnews

## 0. 실행 방식 메모

이 실행은 무인터랙티브 원격 세션이다. `requirement-verification-questions.md`
를 별도로 내고 사람의 `[Answer]:` 를 기다리는 대신, 이 문서 안에 질문과
스스로 낸 답을 함께 적는다(`aidlc-docs/v1-run-dhseo-cardnews/audit.md` 의 「실행 방식 적응」 항목
참조). 계약에 걸리는 결정은 `requirements/decisions.md` 에 근거와 함께
남겼다 — 이 문서는 그것을 다시 베끼지 않고 가리킨다.

## 1. 의도 분석

- **사용자 요청**: `v1-run-dhseo` 브랜치 커밋 `1fe2145`(Reverse Engineering
  직후, 대회 데모 전환 이전)에서 새 토픽 브랜치 `v1-run-dhseo-cardnews` 를
  따고, 기존 요구 팩(일곱 기능)에 여덟째 기능 — 온보딩 카드뉴스 +
  Guest Login 진입점 — 을 더해 Requirements Analysis 부터 실제 웹페이지
  구현(Code Generation)까지 이어간다. 이 기능은 사전 화면 디자인이 없어
  Application Design/Functional Design 이 직접 낸다.
- **요청 유형**: 신규 기능 (브라운필드에 새 표면을 더한다)
- **범위 추정**: 여덟째 기능은 **단일 신규 컴포넌트**(`internal/api/ui`) —
  기존 일곱 기능(다중 컴포넌트에 걸친 시스템 전체 변경)과는 독립이다. 이
  실행이 실제로 설계·구현하는 것은 여덟째 기능 하나뿐이고, 기존 일곱은
  입력 팩으로만 참조한다(다른 유닛이 병렬로 맡는다)
- **복잡도 추정**: 단순 ~ 보통. 함대 상태를 안 만지고 서버 로직이 거의 없다
  (정적 파일 서빙 하나). 복잡도는 화면 쪽에 있다 — 독립된 화면 셋(랜딩 ·
  카드뉴스 · 데모 placeholder)의 경계를 컴포넌트/번들 단위로 지키는 것

## 2. 입력 — 이미 있는 요구 팩

브라운필드 Reverse Engineering 은 `1fe2145` 시점에 이미 끝나 있다
(`aidlc-docs/inception/reverse-engineering/` 아홉 산출물). 이번 회차는
다시 돌지 않는다.

```text
   requirements/enode-features.md   기능 여덟 (4판).  목적 · 요구사항 · 수용 기준
   requirements/decisions.md        결정표.  1~7절 기존 · 8절 이번 회차
   requirements/scene-gates.md      장면 조각 게이트 CP0 ~ CP8.  수용 기준의 실동작판
   requirements/canon.md            enode-design 정본과의 연결
   requirements/constraints.md      안 만드는 것 여덟 범주 · 구조 불변식
   design/                          화면 아홉 장(기존 일곱 축).  3.4 는 안 담는다
```

### 2.1 기존 일곱 (요약 — 전문은 `enode-features.md` §3.1 ~ §3.3)

| 절 | 기능 | 이번 실행의 관계 |
|---|---|---|
| 3.1.1 | 중앙 현황판 | 입력으로만 참조. 구현 안 함 — 다른 유닛 몫 |
| 3.1.2 | 호스트 제어판 | 입력으로만 참조 |
| 3.1.3 | 트랜스크립트 | 입력으로만 참조 |
| 3.2.1 | 대기열 | 입력으로만 참조 |
| 3.2.2 | drain | 입력으로만 참조 |
| 3.2.3 | 되묻기 | 입력으로만 참조 |
| 3.3.1 | Mediator MCP | 입력으로만 참조 |

이 일곱은 이미 완결된 요구사항 문서를 갖고 있다. 이 실행은 그것을 다시
분석하거나 고치지 않는다 — Units Generation 이 이번 회차의 유닛 하나
(3.4.1)만 낸다.

### 2.2 여덟째 — 이번 실행이 설계·구현하는 것

**3.4.1 온보딩 카드뉴스 + Guest Login 진입점.** 전문은
`requirements/enode-features.md` §3.4.1 이 정본이다. 아래는 그 기능
요구사항의 완결성 점검이다.

## 3. 완결성 점검 — 카테고리별

### 3.1 기능 요구사항 — 명확

- Guest 로그인 버튼 -> 카드 시퀀스 -> 닫기 -> 데모 현황판 이동 (흐름 고정)
- 재방문 시 카드 생략 (로컬 저장소 판정)
- 카드 전환 애니메이션 (효과는 Functional Design)
- 재열람 경로 (`decisions.md` 8.2 가 닫았다 — 데모 placeholder 의 링크)

**질문과 답 (이 실행이 스스로 냄)**:

> [Q] 카드가 몇 장이어야 하는가? 착수 요청은 "소재만 준다"고 했다.
> [A] 넉 장 — 소개 1 · 시나리오 2(구형 보드 희소성 · 동료 보드 공유) ·
> 마무리 1. 근거는 `decisions.md` 8.2 표 — 관람객이 다 넘기지 않고 닫을
> 확률이 카드 수에 비례해 커진다.

> [Q] "카드를 닫는다"의 UI 동작이 두 가지다 — 끝까지 넘기는 것과 도중에
> 닫는 것. 도중에 닫아도 데모 현황판으로 가는가?
> [A] 그렇다. 수용 기준 초안이 "끝까지 넘기거나 닫기" 를 대등하게 적었다.
> 어느 쪽이든 종료 동작은 하나 — 카드뉴스 DOM 을 걷고 데모 현황판으로
> 이동하며 `enode.guest.onboarded` 를 세운다. 닫기 버튼은 매 카드에 있다.

### 3.2 비기능 요구사항 — 명확

`constraints.md` 의 차단 게이트 다섯 중 이 표면에 걸리는 것은 커버리지
하한 80%(`internal/api/ui`)와 장식 문자 상한 0 뿐이다. 상세는
`enode-features.md` §3.4.1 비기능 요구사항.

### 3.3 사용자 시나리오 — 명확 (배경 시나리오가 카드 카피의 소재로 이미 주어짐)

관람객이 랜딩에서 Guest Login 을 누른다 -> 카드 넉 장을 본다(또는 도중에
닫는다) -> 데모 현황판 placeholder 를 본다 -> (선택) 나중에 재열람 링크로
카드를 다시 본다. 에지 케이스 — 재방문(로컬 저장소가 남아 있음)에는
카드 없이 곧장 데모 현황판.

### 3.4 비즈니스 맥락 — 명확

목적 · 대상(대회·행사 관람객) · 성공 기준(카드를 끝까지 봤을 때 데모
현황판까지 요소 유출 없이 도달)이 착수 요청과 수용 기준 초안에 이미 있다.

### 3.5 기술 맥락 — 명확

- 통합 지점: `GET /ui/` 정적 서빙 하나. `internal/api/api.go` 의 mux 등록
  줄 하나만 는다
- 데이터: 없음. 클라이언트 로컬 저장소만(서버로 안 나간다)
- 시스템 경계: `internal/api/ui -> internal/store` 금지(기존 구조
  불변식) — 이 기능은 애초에 store 를 부를 이유가 없다

### 3.6 품질 속성 — 명확

- 신뢰성: 정적 파일이라 실패 모드가 거의 없다(404 뿐)
- 유지보수성: 번들 셋(랜딩 · 카드뉴스 · 데모)이 파일 경계로 분리 — 한
  쪽을 고쳐도 다른 쪽 파일이 안 바뀐다
- 테스트 가능성: 서버 쪽은 Go 테스트(핸들러 · 헤더). 화면 쪽은 CP8 의
  실동작 확인(사람이 브라우저로) — `scene-gates.md` §3 CP8
- 접근성: 카드 넘김에 키보드 조작(화살표 · Tab)을 낸다. 명세는
  Functional Design(`frontend-components.md`)

## 4. 확장 opt-in 재확인

`decisions.md` §1 이 착수 전(2026-09-04)에 이미 닫았다. 이 여덟째 기능도
같은 결정 아래 있다.

| Extension | Enabled | Decided At |
|---|---|---|
| security-baseline | Yes | decisions.md §1 (2026-09-04) · 8.3 이 3.4 표면에 적용을 재확인 |
| resiliency-baseline | No | decisions.md §1 |
| property-based-testing | No | decisions.md §1 |

## 5. 요약

이번 Requirements Analysis 는 여덟째 기능 하나를 새로 완결시켰다 — 기존
일곱은 손대지 않고 그대로 입력으로 참조한다. 여덟째 기능의 계약(라우트
하나 · 패키지 하나 · 저장소 키 둘 · 재열람 경로)은 `decisions.md` §8 이
전부 닫았고, 남은 미정은 없다. 카드 문안 · 정확한 와이어프레임 ·
컴포넌트 트리는 Application Design 과 유닛 `cardnews-guest-login` 의
Functional Design 이 낸다.
