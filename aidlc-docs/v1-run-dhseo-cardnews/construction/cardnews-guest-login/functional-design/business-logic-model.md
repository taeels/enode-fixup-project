# Business Logic Model — cardnews-guest-login

## 상태 기계 (브라우저 쪽, 서버는 상태가 없다)

```text
   LANDING --(Guest Login 클릭, hasOnboarded()==false)--> CARD[0]
   LANDING --(Guest Login 클릭, hasOnboarded()==true)---> DEMO

   CARD[i] --(next(), i < 3)--------> CARD[i+1]
   CARD[i] --(next(), i == 3)-------> finish() --> DEMO
   CARD[i] --(prev(), i > 0)--------> CARD[i-1]
   CARD[i] --(prev(), i == 0)-------> CARD[0]  (변화 없음)
   CARD[i] --(닫기)-----------------> finish() --> DEMO

   DEMO --(재열람 링크 클릭)---------> CARD[0]  (새 페이지 로드)
```

각 상태는 **별도 HTML 페이지**다 — SPA 라우터가 없다. `LANDING` =
`static/index.html`, `CARD[i]` = `static/cardnews/index.html` 안의 DOM
상태(페이지 자체는 하나), `DEMO` = `static/demo/index.html`. 상태 전이
셋(landing->cardnews, cardnews->demo, demo->cardnews)은 전부
`window.location.href` 전체 페이지 이동이고, `CARD[i]` 내부의 네 상태
(카드 0~3)만 같은 페이지 안에서 DOM 갱신으로 돈다.

## finish() 가 유일한 종료 경로

「끝까지 넘기기」와「닫기」가 사용자에게는 다른 동작이지만 로직에서는
같은 함수 `finish()` 로 모인다 — 둘 다 `markOnboarded()` 를 부르고
`DEMO` 로 이동한다. 갈래를 안 두는 이유: 요구사항(수용 기준 초안)이
"끝까지 넘기거나 닫기" 를 대등하게 적었고, 어느 경로든 종료 뒤의
불변식("이 시점부터 카드뉴스 화면의 어떤 요소도 안 남는다")이 같기
때문이다.

## 카드뉴스 화면은 onboarded 플래그를 읽지 않는다

`Application Design` 의 `component-methods.md` 는 `?replay=1` 쿼리로
재생을 "강제"하는 것처럼 적었으나, Functional Design 에서 단순화한다 —
**`CardNewsScreen` 은 애초에 `hasOnboarded()` 를 절대 안 읽는다.** 로드
되면 무조건 카드 0번부터 보인다. 판정(재방문이면 카드를 생략한다)은
오직 `LandingPage` 의 `Guest Login` 클릭 핸들러 하나에만 있다.

이유:
- 카드뉴스 페이지를 직접 열었을 때(북마크 · 재열람 링크) 카드가 보이는
  것이 자연스럽다 — "재열람" 자체가 다시 보고 싶어서 누른 것이다
- 판정 지점이 하나(`enterGuest()`)뿐이면 "언제 카드를 건너뛰는가" 라는
  질문에 답이 하나뿐이다. 카드뉴스 페이지 자신도 판정하게 하면 두 곳의
  판정이 어긋날 경우(예: 플래그를 지우는 타이밍)를 따로 다뤄야 한다
- 데모 현황판의 재열람 링크는 `href="/ui/cardnews/"` 로 충분하다 —
  쿼리 매개변수도, 그것을 읽는 코드도 필요 없다

`decisions.md` 8.2 의 "강제 재생 쿼리로 다시 튼다" 는 문장은 이
단순화로 여전히 참이다 — 카드뉴스가 애초에 무조건 처음부터 보이므로
"강제"할 것이 없어졌을 뿐이다. 계약(재열람이 된다)은 그대로 지킨다.
