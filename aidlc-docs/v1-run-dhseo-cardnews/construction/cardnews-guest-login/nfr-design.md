# NFR Design — cardnews-guest-login

`nfr-requirements.md` 의 "적용" 항목마다 구체 구현을 정한다.

## SECURITY-04 HTTP 보안 헤더

`internal/api/ui` 의 `Handler()` 가 모든 응답 앞에 미들웨어를 둔다.

```go
h.Set("Content-Security-Policy", "default-src 'self'")
h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
h.Set("X-Content-Type-Options", "nosniff")
h.Set("X-Frame-Options", "DENY")
h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
```

- `default-src 'self'` — 이 표면은 외부 스크립트/폰트/CDN 을 하나도
  안 쓰므로(SECURITY-10 이 이미 새 의존 0 을 요구) 가장 좁은 정책이
  그대로 맞는다. `unsafe-inline`/`unsafe-eval` 없음
- HSTS — Mediator 가 지금 평문 HTTP 다(`decisions.md` §3
  SECURITY-01, 기존 코드 사실 · 이 유닛이 안 고친다). 그래도 규칙은
  "새 표면에 적용" 이라 헤더는 낸다 — TLS 뒤에 서면 즉시 효과가
  생기고, 평문일 때는 브라우저가 무시할 뿐 해가 없다
- `X-Frame-Options: DENY` — 이 표면을 다른 페이지 안에 iframe 으로
  끼워 넣는 시나리오가 없다(문서화된 의도)

테스트: `ui_test.go` 가 `GET /ui/` · `GET /ui/cardnews/` · `GET
/ui/demo/` 응답 헤더 다섯을 전부 검증한다.

## SECURITY-08 접근 제어 (문서화된 예외)

`GET /ui/*` 는 `s.auth(...)` 로 안 감싼다 — `internal/api/api.go` 의
다른 열다섯 라우트와 다르다. 이 예외는:

- `enode-features.md` §3.4.1 보안 요구사항이 요구한 것이다(공개 정적
  표면)
- 3.1.1(중앙 현황판)이 이미 같은 결정을 내렸다(`GET /ui/` 무인증,
  `enode-features.md` §3.1.1) — 이 유닛이 새로 여는 예외가 아니라
  이미 있는 결정을 앞당겨 구현하는 것이다
- 이 라우트로 들어오는 요청은 **읽기만** 한다 — 상태를 바꾸는 어떤
  동작도 없다(BR-1 ~ BR-6). "쓰기 없는 공개 표면" 이 이 예외의 좁은
  범위다

## SECURITY-09 하드닝

- **디렉터리 목록 금지**: `static/` 의 서빙 대상 디렉터리(루트 ·
  `cardnews/` · `demo/` · `shared/`) 전부에 `index.html` 을 둔다.
  `http.FileServerFS` 는 `index.html` 이 있으면 그것을 내고 목록을
  안 만든다. `shared/` 는 `guest.js` 하나만 있고 디렉터리로 직접
  요청될 일이 없지만, 방어적으로 빈 `index.html`(204 대신 최소
  텍스트)을 둔다
- **에러 응답**: `http.FileServerFS` 의 기본 404 응답은 경로 하나만
  돌려주고 스택 트레이스나 내부 파일시스템 구조를 안 보인다 — 별도
  에러 핸들러가 필요 없다

## SECURITY-11 보안 설계 (오용 사례)

오용 시나리오: 누군가 브라우저 개발자 도구로 `enode.guest.id` 값을
읽거나 조작해 서버에 다른 사람인 척 요청을 보내려 시도한다.

**막히는 이유**: 애초에 그 값을 받는 서버 엔드포인트가 없다(BR-6).
`guest.js` 가 그 값을 어떤 요청에도 안 싣는다는 사실 자체가 이
오용을 무력화한다 — "검증해서 막는" 것이 아니라 "표면이 없어서 안
통하는" 설계다.

## SECURITY-15 예외 처리

`guest.js` 의 저장소 접근 셋(`hasOnboarded` · `markOnboarded` ·
`guestName`) 전부 `try { ... } catch { ... }` 로 감싼다 — 세부는
`business-rules.md` BR-5. `ui_test.go` 는 Go 쪽만 검증한다(핸들러가
패닉 없이 404/200 을 낸다) — 브라우저 `localStorage` 예외는 Go 테스트
대상이 아니다(도구가 다르다). CP8 의 실동작 확인(스크립트로 저장소를
지운 뒤 재확인)이 그 경로를 사람이 확인하는 지점이다.

## 커버리지 하한 80%

`internal/api/ui` 는 DB 를 안 쓰므로 `ui_test.go` 가 `ENODE_TEST_
DATABASE_URL` 없이도 전부 돈다(`decisions.md` 2절의 원칙 그대로).
`Handler()` 의 모든 분기(랜딩/카드뉴스/데모 서빙, 알 수 없는 경로의
404, 헤더 설정)를 테스트가 지나가면 이런 얇은 패키지에서 80% 는
자연히 넘는다 — 별도 커버리지 전용 코드를 안 만든다.
