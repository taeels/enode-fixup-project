# Component Methods — 3.4 온보딩 카드뉴스 + Guest Login

상세 비즈니스 규칙(카드 넘김 조건 등)은 Functional Design
(`aidlc-docs/construction/cardnews-guest-login/functional-design/`)이 낸다.
여기는 메서드/함수 서명과 입출력 타입까지만 정한다.

## C-1 UIHandler (Go)

```go
// Handler 는 GET /ui/ 아래 랜딩 · 카드뉴스 · 데모 세 번들을 낸다.
func Handler() http.Handler
```

- 입력: 없음(패키지 수준 embed.FS 를 감싼다)
- 출력: `http.Handler`. 내부에서 `http.StripPrefix("/ui/", ...)` 와
  보안 헤더 미들웨어를 두른 `http.FileServerFS` 를 합성한다
- 에러: 없음(임베드 실패는 컴파일 타임/초기화 타임에 패닉 — 정적 자산
  자체가 빌드에 포함되므로 런타임에 없어질 수 없다)

## C-2 LandingPage (JS, `landing.js`)

```js
// enterGuest() 는 Guest Login 버튼의 클릭 핸들러다.
// 첫 방문이면 카드뉴스로, 아니면 데모 자리로 이동한다.
function enterGuest()
```

- 입력: 없음(DOM 이벤트에서 호출)
- 출력: 없음 — `window.location.href` 이동이 부수효과
- 의존: `guest.hasOnboarded()` (C-5)

## C-3 CardNewsScreen (JS, `cardnews.js`)

```js
// renderCard(index) 는 index 번 카드를 활성 카드로 그린다.
function renderCard(index)

// next() / prev() 는 카드 인덱스를 옮기고 renderCard 를 부른다.
// 마지막 카드에서 next() 를 부르면 finish() 로 넘어간다.
function next()
function prev()

// finish() 는 첫 방문 완료를 기록하고 데모 자리로 이동한다.
// 닫기 버튼과 마지막 카드의 next() 가 둘 다 이 함수로 모인다.
function finish()
```

- 입력: `index` (0 이상 카드 수 미만의 정수)
- 출력: 없음 — DOM 갱신 또는 `window.location.href` 이동이 부수효과
- 의존: `guest.markOnboarded()` (C-5)

## C-4 DemoPlaceholder

정적 HTML 하나 — 컴포넌트 메서드가 없다. 카드뉴스 재열람 링크는
`href="/ui/cardnews/?replay=1"` 앵커 태그다(스크립트 없음).

## C-5 GuestIdentity (JS, `guest.js`)

```js
// hasOnboarded() 는 첫 방문 완료 여부를 읽는다.
function hasOnboarded()  // -> boolean

// markOnboarded() 는 첫 방문 완료를 기록한다.
function markOnboarded()  // -> void

// guestName() 은 저장된 이름을 읽거나, 없으면 새로 만들어 저장하고
// 돌려준다.
function guestName()  // -> string
```

- 저장소: `window.localStorage` 하나만 쓴다. 실패(사생활 모드 등)를
  던지지 않고 메모리 폴백으로 삼는다 — 세부는 Functional Design
