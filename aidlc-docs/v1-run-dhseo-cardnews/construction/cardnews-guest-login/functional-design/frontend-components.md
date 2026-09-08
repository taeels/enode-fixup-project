# Frontend Components — cardnews-guest-login

착수 시점에는 사전 화면 디자인이 없는 축이었다(`decisions.md` 8절
머리) — 그래서 이 문서가 레이아웃 · 카드 순서 · 전환 방식을 정하는
디자인 산출물로 시작했다. `design/README.md` 의 표기 관례(화면 번호
S0~S5 · 컴포넌트 C1~C5)는 기존 일곱 축의 것이라 여기서는 새 이름을
쓴다(아래 §0).

2026-09-08 에 `design/enode-cardnews.pen` 이 새로 생겼다(§2.1 카드
그림). **이제부터 이 유닛도 `design/README.md` §7 의 규칙을 따른다 —
"화면이 있는 유닛은 이 문서와 pen 을 읽는다."** 이 문서가 최초
디자인 산출물이었다는 사실은 남지만, 카드 문안 · 그림 · 진행 표시
개수의 정본은 이제 이 문서와 `design/enode-cardnews.pen` 둘이 같이
진다 — 한쪽만 고치면 갈라진다. 다음에 AI-DLC 로 이 유닛을 다시 태워
코드를 낼 때는 반드시 `design/enode-cardnews.pen` 을 함께 읽어서
카드 그림(§2.2)과 5번째 카드(§2.6)를 코드에 반영해야 한다 — 5번째
카드는 pen 에는 정적 목업(재생 아이콘 placeholder)으로만 있고, 실제
동작(영상 재생)은 이 문서 §2.6 이 정본이다.

## 0. 파일 · 라우트 지도

```text
   internal/api/ui/
     ui.go                     embed.FS + Handler() + 보안 헤더
     ui_test.go
     static/
       index.html              GET /ui/                (LandingPage)
       landing.css
       landing.js
       cardnews/
         index.html            GET /ui/cardnews/        (CardNewsScreen)
         cardnews.css
         cardnews.js
       demo/
         index.html            GET /ui/demo/            (DemoPlaceholder)
         demo.css
       shared/
         guest.js               GuestIdentity (세 화면이 공유)
```

`http.FileServerFS` 가 확장자로 정적 파일을 그대로 서빙하므로 `.css` ·
`.js` 도 각자의 경로(`GET /ui/landing.css` 등)로 열린다 — 별도 등록이
없다.

## 1. 랜딩 (`static/index.html`)

### 1.1 와이어프레임

```text
   +--------------------------------------------------------------+
   |  enode                                                        |
   |                                                                |
   |   +----------------------------+   +------------------------+ |
   |   |  관리자용                    |   |  둘러보러 오셨나요?      | |
   |   |                              |   |                        | |
   |   |  Mediator 토큰으로 로그인합니다 |   |  Guest 로 들어가면      | |
   |   |                              |   |  enode 가 왜 필요한지     | |
   |   |  [ 토큰 입력 자리 ]           |   |  카드로 먼저 보여드려요   | |
   |   |  (이 유닛의 스코프 밖 —       |   |                        | |
   |   |   자리표시만 그린다)          |   |  [  Guest 로 둘러보기  ] | |
   |   +----------------------------+   +------------------------+ |
   +--------------------------------------------------------------+
```

두 영역은 형제 `<section>` 이고 CSS 로만 나란히 놓인다(flex). 서로의
DOM 을 참조하지 않는다.

### 1.2 컴포넌트 트리

```text
   <body>
     <header>            "enode" 워드마크. 정적 텍스트
     <main>
       <section id="admin-login">     placeholder. <form> 없음
         <h2>관리자용</h2>
         <p>안내 문구</p>
         <input disabled data-testid="landing-admin-token-input"
                placeholder="Mediator 토큰 (준비 중)">
       <section id="guest-entry">
         <h2>둘러보러 오셨나요?</h2>
         <p>안내 문구</p>
         <button data-testid="landing-guest-login-button"
                 onclick="enterGuest()">Guest 로 둘러보기</button>
     <script src="/ui/shared/guest.js">
     <script src="/ui/landing.js">
```

### 1.3 상태 · 상호작용

- 로컬 상태 없음(리액트 등 프레임워크 없이 순수 DOM)
- 유일한 이벤트: `guest-login-button` 클릭 -> `enterGuest()`
  (`business-rules.md` BR-1)
- `admin-login` 의 `<input>` 은 `disabled` 다 — 제출도, 어떤 네트워크
  호출도 일으키지 않는다. **관리자 화면을 손대지 않는다는 요구를
  코드 수준에서 강제**하는 방법이다(동작이 없으므로 회귀시킬 것이 없다)

## 2. 카드뉴스 (`static/cardnews/index.html`)

### 2.1 와이어프레임

카드마다 사진(들)이 title 과 본문 사이에 들어간다. `design/
enode-cardnews.pen` 이 이 그림의 원본이다(문서 머리 §0 참고) —
아래는 그 그림을 코드 관점에서 옮긴 것이다.

```text
   +--------------------------------------------------------------+
   |  [ x 닫기 ]                                            1 / 5  |
   |                                                                |
   |                 enode 는 왜 필요할까요?                        |
   |          [ 사진 — 여러 보드에 둘러싸인 개발자 ]                 |
   |          개별로 관리하는 개발보드를,                            |
   |          팀이 필요할 때 나눠 씁니다.                            |
   |                                                                |
   |     [ < 이전 ]        ● ○ ○ ○ ○      [ 다음 > ]                |
   +--------------------------------------------------------------+
```

카드 4 는 사진이 둘이다(폰 사진 + 실제 보드 사진, 나란히):

```text
   +--------------------------------------------------------------+
   |  [ x 닫기 ]                                            4 / 5  |
   |                                                                |
   |                 enode 가 그 다리를 놓습니다                    |
   |          [ 사진 — 폰 ]      [ 사진 — 개발보드 ]                 |
   |          원하는 작업을 지원 가능한 보드, 서버, 시료 등           |
   |          가용자원을 살펴보세요                                  |
   |                                                                |
   |     [ < 이전 ]        ○ ○ ○ ● ○      [ 다음 > ]                |
   +--------------------------------------------------------------+
```

마지막 카드(5/5)는 사진 대신 영상이고, "다음" 버튼 자리가 "현황판
보기" 로 바뀐다. 본문 문구는 없다 — 제목과 영상뿐이다:

```text
   +--------------------------------------------------------------+
   |  [ x 닫기 ]                                            5 / 5  |
   |                                                                |
   |          enode 와 함께한 미래는 이렇게 일 합니다                |
   |          [ 영상 — story2.mp4. 자동재생 · 음소거 · 반복 ]        |
   |                                                                |
   |     [ < 이전 ]        ○ ○ ○ ○ ●      [ 현황판 보기 ]           |
   +--------------------------------------------------------------+
```

### 2.2 카드 문안과 그림 (다섯 장)

카드 1~4 의 사진은 `design/assets-cardnews/` 의 크롭 이미지를 그대로
쓴다(동료가 만든 실사 스크린샷에서 뜬 것 — AI 로 다시 그리지 않는다,
`audit.md` 참고). 카드 5 의 영상은 `story2.mp4` 다.

| # | 제목 | 본문 | 사진/영상 |
|---|---|---|---|
| 1 | enode 는 왜 필요할까요? | 개별로 관리하는 개발보드를, 팀이 필요할 때 나눠 씁니다. | `card1.png` |
| 2 | 갤럭시 S2 버그를, 오늘 고쳐야 한다면 | 갤럭시 S26 이 나온 지금도 그 시절 버그는 그 시절 보드에서만 재현됩니다. 그런 보드는 대개 한 대, 한 자리뿐입니다. | `card2.png` |
| 3 | 옆 팀 보드가 비어 있다면 | 여러 보드를 동시에 써야 하는 날, 지금 비어 있는 동료의 보드를 곧장 빌려 쓸 수 있다면 훨씬 빨리 끝납니다. | `card3.png` |
| 4 | enode 가 그 다리를 놓습니다 | 원하는 작업을 지원 가능한 보드, 서버, 시료 등 가용자원을 살펴보세요 | `card4.png` + `card4-board.png` |
| 5 | enode 와 함께한 미래는 이렇게 일 합니다 | (없음) | `story2.mp4` |

### 2.3 컴포넌트 트리

```text
   <body>
     <header>
       <button data-testid="cardnews-close-button" onclick="finish()">닫기</button>
       <span id="progress-text">1 / 5</span>
     <main id="card-stage">
       <article class="card" data-index="0">
         <h2>...</h2>
         <img data-testid="cardnews-illustration-0">
         <p>...</p>
       <article class="card" data-index="1">
         <h2>...</h2>
         <img data-testid="cardnews-illustration-1">
         <p>...</p>
       <article class="card" data-index="2">
         <h2>...</h2>
         <img data-testid="cardnews-illustration-2">
         <p>...</p>
       <article class="card" data-index="3">
         <h2>...</h2>
         <div class="card-image-row">
           <img data-testid="cardnews-illustration-3">
           <img data-testid="cardnews-illustration-3b">
         <p>...</p>
       <article class="card card-video" data-index="4">
         <h2>...</h2>
         <video data-testid="cardnews-story-video" src="/ui/cardnews/story2.mp4"
                autoplay muted loop playsinline controls>
     <footer>
       <button data-testid="cardnews-prev-button" onclick="prev()">이전</button>
       <nav id="progress-dots">
         <span data-testid="cardnews-progress-dot-0" class="dot">
         <span data-testid="cardnews-progress-dot-1" class="dot">
         <span data-testid="cardnews-progress-dot-2" class="dot">
         <span data-testid="cardnews-progress-dot-3" class="dot">
         <span data-testid="cardnews-progress-dot-4" class="dot">
       <button data-testid="cardnews-next-button" onclick="next()">다음</button>
     <script src="/ui/shared/guest.js">
     <script src="/ui/cardnews/cardnews.js">
```

다섯 `<article class="card">` 는 처음부터 DOM 에 전부 있다(서버 호출로
카드를 더 받아오지 않는다 — BR-4). `renderCard(index)` 는 `.card` 에
`.active` 클래스를 토글할 뿐 DOM 을 새로 만들지 않는다. 카드 개수는
`cardnews.js` 가 DOM 에서 동적으로 세므로("다음"/"현황판 보기" 전환,
진행 표시 모두 `cards.length` 기준) 카드를 더하거나 빼도 이 파일의
로직은 바뀌지 않는다.

### 2.4 전환 애니메이션

`decisions.md` 8.2 가 CSS `transform` + `transition` 으로 닫았다.

```css
#card-stage { position: relative; overflow: hidden; }
.card {
  position: absolute; inset: 0;
  transform: translateX(100%);
  transition: transform 240ms ease;
}
.card.active { transform: translateX(0); }
.card.exiting-left  { transform: translateX(-100%); }
```

`renderCard(index)` 가 하는 일: 이전 활성 카드에는 이동 방향에 맞는
`exiting-left`/기본(오른쪽 대기) 클래스를 주고, 새 카드에 `active` 를
준다. 새 JS 애니메이션 라이브러리를 받지 않는다(요구 그대로).

### 2.5 상호작용 · 접근성

- 클릭: 닫기 · 이전 · 다음 버튼(§2.3)
- 키보드: `ArrowRight` -> `next()`, `ArrowLeft` -> `prev()`, `Escape` ->
  `finish()`. `document` 레벨 `keydown` 리스너 하나
- 진행 점(`progress-dots`)은 읽기 전용 표시다 — 클릭해서 특정 카드로
  건너뛰는 기능은 이번 스코프가 아니다(요구에 없다. 넣으면 새 상호작용을
  임의로 추가하는 것이라 안 만든다)
- `aria-live="polite"` 를 `#progress-text` 에 달아 카드가 바뀔 때
  스크린리더가 "2 / 5" 를 읽게 한다
- 카드 5 는 `<video autoplay muted loop playsinline controls>` 다.
  음소거 자동재생만 브라우저가 사용자 조작 없이 허용하므로 `muted` 를
  뺄 수 없다 — 소리는 `controls` 로 사용자가 직접 켠다

## 3. 데모 현황판 자리표시 (`static/demo/index.html`)

### 3.1 와이어프레임

```text
   +--------------------------------------------------------------+
   |  enode 현황판                                                 |
   |                                                                |
   |            (placeholder)                                      |
   |     다음 라운드에서 이 화면을 채웁니다.                          |
   |                                                                |
   |     Guest — guest-vivid-otter 로 둘러보는 중                   |
   |                                                                |
   |     [ 카드뉴스 다시 보기 ]                                      |
   +--------------------------------------------------------------+
```

### 3.2 컴포넌트 트리

```text
   <body>
     <header><h1>enode 현황판</h1></header>
     <main>
       <p class="placeholder-notice">다음 라운드에서 이 화면을 채웁니다.</p>
       <p id="guest-badge"></p>          <!-- guest.js 가 이름을 채운다 -->
       <a data-testid="demo-replay-cardnews-link" href="/ui/cardnews/">
         카드뉴스 다시 보기
       </a>
     <script src="/ui/shared/guest.js">
     <script src="/ui/demo/demo.js">     <!-- guest-badge 채우기 한 줄뿐 -->
```

재열람 링크는 평범한 `<a href>` 다 — 클릭 핸들러가 없다
(`business-logic-model.md` 의 단순화).

## 4. 폼 검증 · API 통합

- 폼이 없다(관리자 토큰 입력은 `disabled`). 검증 규칙 없음
- API 호출이 없다 — 이 유닛의 어떤 파일도 `fetch`/`XMLHttpRequest` 를
  쓰지 않는다. 백엔드 통합 지점은 정적 파일 서빙 하나(`GET /ui/*`)뿐

## 5. data-testid 목록 (Automation Friendly Code Rules)

```text
   landing-admin-token-input
   landing-guest-login-button
   cardnews-close-button
   cardnews-prev-button
   cardnews-next-button
   cardnews-progress-dot-0 ~ cardnews-progress-dot-4
   cardnews-illustration-0 ~ cardnews-illustration-3
   cardnews-illustration-3b
   cardnews-story-video
   demo-replay-cardnews-link
```
