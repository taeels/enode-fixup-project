# Component Dependency — 3.4 온보딩 카드뉴스 + Guest Login

## 의존 그래프

```text
   internal/api.Server  ---(import, mux 등록 한 줄)-->  internal/api/ui (C-1)

   브라우저 쪽 (파일 경계 = 컴포넌트 경계, JS import 가 아니라 전체
   페이지 이동으로 연결된다):

   LandingPage (C-2)  --script src-->  guest.js (C-5)
        |
        | window.location.href (전체 페이지 이동)
        v
   CardNewsScreen (C-3)  --script src-->  guest.js (C-5)
        |
        | window.location.href (전체 페이지 이동, 완료 시)
        v
   DemoPlaceholder (C-4)
        |
        | <a href> 앵커 (재열람, 스크립트 없음)
        v
   CardNewsScreen (C-3, ?replay=1)
```

## 통신 패턴

- **Go -> Go**: `internal/api` 가 `internal/api/ui` 를 임포트한다. 반대
  방향은 없다(순환 없음). `internal/api/ui` 는 `internal/store` ·
  `internal/match` · `internal/record` 어느 것도 임포트하지 않는다 —
  구조 불변식(`constraints.md`) 그대로
- **브라우저 내부**: 세 화면(C-2 · C-3 · C-4)은 **서로를 import 하지
  않는다.** 이동은 전부 `window.location.href` 전체 페이지 네비게이션
  이거나 `<a href>` 다 — 번들 경계가 파일 경계와 정확히 겹친다
- **공유 코드**: `guest.js` (C-5) 하나만 셋이 `<script src="/ui/shared/
  guest.js">` 로 공유한다. 순수 함수 모음이고 DOM 을 만지지 않는다 —
  화면마다 자기 DOM 갱신은 자기 파일이 한다
- **서버 호출**: 브라우저 쪽 어느 파일도 `fetch`/`XMLHttpRequest` 를
  쓰지 않는다. 전부 정적 파일 GET 하나(`/ui/...`)로 끝난다

## 3.1.1(중앙 현황판) 축과의 접점

`decisions.md` 8.5 가 정본이다. 접점은 둘 — `internal/api/ui` 패키지
자체(3.1.1 축이 나중에 `static/index.html` 의 관리자 로그인 placeholder
를 실제 동작으로 채운다)와 `internal/api/api.go` 의 `GET /ui/` 등록
줄(둘 다 같은 줄을 원하므로 파일 행렬 접점). 이 실행이 먼저 세우고,
3.1.1 축이 병합될 때 그 자리를 이어받는다.
