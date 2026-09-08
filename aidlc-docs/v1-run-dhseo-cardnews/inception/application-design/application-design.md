# Application Design — 3.4 온보딩 카드뉴스 + Guest Login

이 문서는 `components.md` · `component-methods.md` · `services.md` ·
`component-dependency.md` 를 한 장으로 묶는다. 전문은 각 파일에 있다.

## 요약

- **컴포넌트 다섯**: `UIHandler`(Go, 정적 파일 서빙) · `LandingPage` ·
  `CardNewsScreen` · `DemoPlaceholder`(정적 화면 셋) · `GuestIdentity`
  (공유 로컬 저장소 모듈)
- **서비스 하나**: `StaticAssetService` — 오케스트레이션이 아니라
  파일 서빙 하나뿐이다. Guest 세션을 서버에 두지 않는다(의도적으로
  안 만든 서비스)
- **의존**: `internal/api -> internal/api/ui` (Go, mux 등록 한 줄).
  브라우저 쪽 세 화면은 서로 import 하지 않고 전체 페이지 이동으로만
  연결된다 — 번들 경계 = 파일 경계
- **접점**: 3.1.1(중앙 현황판) 축과 `internal/api/ui` 패키지 · `GET
  /ui/` 등록 줄에서 만난다(`decisions.md` 8.5)

## 설계가 요구를 어떻게 만족하는가

| 요구 (`enode-features.md` §3.4.1) | 설계 반영 |
|---|---|
| 카드뉴스는 독립 화면, 데모와 조건부 결합 금지 | `CardNewsScreen` 과 `DemoPlaceholder` 가 별도 HTML/CSS/JS 파일 — 공유는 순수 로컬 저장소 함수(`guest.js`) 하나뿐 |
| internal/api/ui 정적 영역 + 번들 분리 | `static/{index.html, cardnews/, demo/, shared/}` 네 디렉터리 |
| 새 백엔드 라우트 최소화 | `GET /ui/` 하나. 그 아래는 전부 정적 파일 경로 — 별도 핸들러 0 |
| Guest 식별 = 인증 아님 | `GuestIdentity` 가 서버 호출을 하지 않는다 — 네트워크 계층 자체가 없다 |
| 관리자 화면을 손대지 않는다 | `LandingPage` 는 관리자 로그인 자리를 placeholder 로만 잡는다. `internal/api` 의 기존 핸들러 15개를 하나도 안 고친다 |

## 다음 단계

Units Generation 이 이 컴포넌트 다섯을 유닛 하나(`cardnews-guest-login`)
로 묶는다 — 서로 강하게 결합돼 있어(랜딩 없이 카드뉴스로 못 들어가고,
카드뉴스 없이 데모로 가는 유일한 경로가 랜딩이다) 나눌 이유가 없다.
