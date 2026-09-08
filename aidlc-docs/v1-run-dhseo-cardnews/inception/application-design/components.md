# Components — 3.4 온보딩 카드뉴스 + Guest Login

기존 일곱 기능의 컴포넌트는 이 실행의 스코프 밖이라 다루지 않는다.

## C-1 UIHandler (Go, `internal/api/ui`)

- **책임**: 랜딩 · 카드뉴스 · 데모 세 번들을 담은 정적 파일을 `embed.FS`
  로 안고, `GET /ui/` 아래에서 서빙하는 `http.Handler` 하나를 낸다. 모든
  HTML 응답에 SECURITY-04 헤더를 붙인다
- **책임 밖**: 어떤 비즈니스 로직도 없다. `internal/store` 를 참조하지
  않는다(구조 불변식). 인증하지 않는다 — 이 표면 전체가 무인증
- **인터페이스**: `func Handler() http.Handler`

## C-2 LandingPage (정적, `static/index.html` + `landing.js` + `landing.css`)

- **책임**: 랜딩 화면을 그린다 — 관리자 로그인 자리(placeholder, 이
  유닛은 동작을 안 채운다)와 Guest Login 버튼을 시각적으로 분리해 보인다.
  Guest Login 클릭 시 `GuestIdentity` 로 첫 방문 여부를 물어 카드뉴스
  또는 데모 자리로 이동한다
- **책임 밖**: 관리자 로그인의 실제 동작(토큰 검증 · `GET /v1/nodes` 호출)

## C-3 CardNewsScreen (정적, `static/cardnews/*`)

- **책임**: 카드 넉 장을 순서대로 보이고 슬라이드 애니메이션으로 전환한다.
  다음/이전/닫기/진행 점 상호작용을 처리한다. 종료(끝까지 넘김 또는
  닫기) 시 `GuestIdentity` 에 첫 방문 완료를 기록하고 데모 자리로
  이동한다. `?replay=1` 쿼리가 있으면 완료 기록 여부와 무관하게 처음부터
  보인다
- **책임 밖**: 데모 현황판의 내용. 관리자 로그인

## C-4 DemoPlaceholder (정적, `static/demo/*`)

- **책임**: 자리표시 화면 하나를 보인다 — "다음 라운드가 채운다" 안내와
  카드뉴스 재열람 링크
- **책임 밖**: 실제 함대 데이터 표시(3.1.1 축의 몫)

## C-5 GuestIdentity (정적 JS 모듈, `static/shared/guest.js`)

- **책임**: 클라이언트 로컬 저장소에서 Guest 식별 이름을 읽거나 새로
  만들고, 첫 방문 완료 플래그를 읽고 쓴다. **네트워크 호출을 하지 않는다**
  — 이 모듈에 `fetch`/`XMLHttpRequest` 가 없다
- **책임 밖**: 서버에 식별을 전달하는 것. 인증

`static/shared/`는 세 번들이 공유하는 유일한 코드다 — 순수 로컬 저장소
접근 함수만 담아 "컴포넌트/번들 단위로 분리한다"는 요구를 어기지 않는다
(레이아웃 · 카드 콘텐츠 · 스타일은 번들마다 독립).
