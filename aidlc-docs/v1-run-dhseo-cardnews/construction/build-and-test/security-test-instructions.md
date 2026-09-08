# Security Test Instructions — cardnews-guest-login

`security-baseline` 이 켜져 있다. `nfr-requirements.md` 가 규칙별
적용 여부를, `nfr-design.md` 가 구현을 정했다. 여기는 그것을 확인하는
절차다.

## SECURITY-04 HTTP 보안 헤더

```bash
go test ./internal/api/ui/... -run TestHandlerSecurityHeaders -v
```

**실행 결과**: 통과. 랜딩·카드뉴스·데모·404 응답 넷 모두에서 다섯
헤더(CSP · HSTS · X-Content-Type-Options · X-Frame-Options ·
Referrer-Policy)가 정확한 값으로 실린다(`ui_test.go`).

## SECURITY-09 하드닝 (디렉터리 목록)

```bash
go test ./internal/api/ui/... -run TestHandlerNoDirectoryListing -v
```

**실행 결과**: 통과. `shared/` 에 `index.html` 이 있어 목록 대신 그것을
낸다.

## SECURITY-06 (BR-6) Guest 식별이 서버로 안 나간다

자동화된 검사가 아니라 **코드 리뷰로 확인**한다 — 만들지 않은 네트워크
호출은 정적 분석 대상이 없다.

```bash
grep -n "fetch\|XMLHttpRequest" internal/api/ui/static/shared/guest.js \
  internal/api/ui/static/landing.js \
  internal/api/ui/static/cardnews/cardnews.js \
  internal/api/ui/static/demo/demo.js
```

**실행 결과**: 매치 0건. 네 파일 어디에도 네트워크 호출이 없다.

## CSP 가 인라인 스크립트를 막는지

와이어프레임(`frontend-components.md`)의 `onclick=` 표기를 실제
구현에서 `addEventListener` 로 바꾼 이유가 이것이다(`code/summary.md`).
모든 `<script>` 태그가 `src=` 외부 파일이고 인라인 스크립트 블록이
없음을 확인한다.

```bash
grep -n "<script>" internal/api/ui/static/index.html \
  internal/api/ui/static/cardnews/index.html \
  internal/api/ui/static/demo/index.html
```

**실행 결과**: 매치 0건 — 인라인 `<script>` 블록이 없다. 전부
`<script src="...">` 다.
