# Code Generation Summary — cardnews-guest-login

## 신규 파일 (Application Code)

```text
   internal/api/ui/ui.go                       패키지 진입점 — embed + Handler()
   internal/api/ui/ui_test.go                   핸들러 테스트
   internal/api/ui/static/index.html            LandingPage
   internal/api/ui/static/landing.css
   internal/api/ui/static/landing.js
   internal/api/ui/static/cardnews/index.html   CardNewsScreen
   internal/api/ui/static/cardnews/cardnews.css
   internal/api/ui/static/cardnews/cardnews.js
   internal/api/ui/static/demo/index.html       DemoPlaceholder
   internal/api/ui/static/demo/demo.css
   internal/api/ui/static/demo/demo.js
   internal/api/ui/static/shared/guest.js       GuestIdentity (공유)
   internal/api/ui/static/shared/index.html     디렉터리 목록 방지용 자리표시
```

## 수정 파일 (Brownfield)

```text
   internal/api/api.go   import 한 줄("github.com/taeels/enode/internal/api/ui")
                          + mux 등록 한 줄(mux.Handle("/ui/", ui.Handler()))
                          이 두 줄 밖에는 diff 가 없다 — 기존 라우트 15개
                          그대로, s.auth 로 감싼 핸들러 어느 것도 안 고쳤다
```

## CSP 준수를 위한 구현 결정 (Functional Design 이후 정한 것)

`frontend-components.md` 의 컴포넌트 트리는 `onclick="..."` 인라인
핸들러로 의도를 보였으나, `nfr-design.md` 가 `Content-Security-Policy:
default-src 'self'` 를 요구하고 그 정책은 `unsafe-inline` 없이
인라인 이벤트 핸들러를 막는다. 그래서 실제 구현은 전부
`addEventListener` 로 바꿨다 — `landing.js` · `cardnews.js` 가 DOM
로드 뒤 셀렉터로 요소를 찾아 이벤트를 붙인다. `data-testid` 계약과
동작은 설계 문서 그대로이고 바뀐 것은 배선 방식뿐이다.

## 유닛이 만지지 않은 것

- `internal/store` · `internal/match` · `internal/record` — 어느 것도
  임포트하지 않는다(구조 불변식)
- `internal/api` 의 기존 핸들러 15개 — 등록 줄 둘을 빼면 diff 없음
- `go.mod` · `go.sum` — 새 의존 0
- `design/` — pen 파일도 exports 도 안 건드렸다(진행자 전용 자원)
