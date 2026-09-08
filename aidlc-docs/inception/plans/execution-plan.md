# Workflow Planning — v1-run-dhseo-cardnews

## 1. 이번 실행이 도는 단계

```text
   Workspace Detection       완료 (1fe2145 시점)
   Reverse Engineering       완료 (1fe2145 시점, 이관) — 다시 안 돎
   Requirements Analysis     완료 — 여덟째 기능만 이 실행이 민다
   User Stories              완료 — 3.4 만. 신규 사용자 대면 기능
   Workflow Planning         이 문서
   Application Design        실행 — 3.4 는 신규 컴포넌트다 (internal/api/ui)
   Units Generation          실행 — 단일 유닛 cardnews-guest-login
     Functional Design       실행 — 사전 화면 디자인이 없어 이 유닛이 직접 낸다
     NFR Requirements        실행 — security-baseline 이 새 표면에 걸린다
     NFR Design              실행 — 위와 짝
     Infrastructure Design   SKIP — 새 인프라 없음(정적 파일 + mux 등록 한 줄)
     Code Generation         실행
     Build and Test          실행
   Operations                PLACEHOLDER (판 자체가 그렇다)
```

## 2. 왜 이 조합인가

- **Application Design 을 켠다** — `application-design.md` 의 실행 조건
  「새 컴포넌트/서비스가 필요한가」에 해당한다. `internal/api/ui` 는
  이 저장소에 없던 패키지다
- **Units Generation 을 켜지만 유닛은 하나다** — 이 실행의 스코프가
  3.4.1 하나뿐이라 분해할 다른 유닛이 없다. 기존 일곱 기능의 유닛
  분해는 다른 세션/유닛의 몫이라 여기서 안 낸다
- **Infrastructure Design 을 끈다** — 새 배포 대상 · 새 클라우드 자원 ·
  새 바이너리가 없다. 기존 `cmd/mediator` 프로세스 안에서 정적 파일
  핸들러 하나가 늘 뿐이다
- **Functional Design 이 화면 디자인까지 낸다** — 착수 요청이 명시한
  대로, 이 축은 `design/enode-ux.pen` 의 사전 디자인이 없다. Application
  Design 이 컴포넌트 경계(랜딩/카드뉴스/데모 세 번들)를 정하고,
  Functional Design 이 그 안의 와이어프레임·카드 문안·상호작용을 낸다

## 3. 유닛 병렬성

이 실행은 유닛이 하나라 CONVENTIONS.md 3.1 의 「의존 그래프에 따른 병렬
착수」가 적용될 자리가 없다. `decisions.md` 8.5 가 3.1.1(중앙 현황판) 축과
만나는 파일 접점(`internal/api/ui` 패키지 · `api.go` 의 등록 줄 하나)을
이미 적어 뒀다 — 그 축이 나중에 별도 유닛으로 붙을 때 참조한다.

## 4. 게이트

`scene-gates.md` CP8 이 이 실행의 유일한 조각 게이트다. 「먼저 서는
기능」이 없어(표 CP8 행) 다른 조각의 완료를 안 기다리고 언제든 돈다.
CP4(장면 전체)의 조건이 아니다.
