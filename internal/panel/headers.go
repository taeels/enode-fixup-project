package panel

import "net/http"

// securityHeaders 는 제어판의 모든 응답에 보안 헤더 다섯을 건다.
//
// 왜 지금 생겼나 - 이 회차 전까지 제어판은 자기 화면만 그렸다. 이제 이 페이지에
// 하네스의 출력을 그리고, 그것은 신뢰할 수 없는 입력이다. 위험을 들여오는 쪽이
// 닫는다.
//
// 왜 internal/api/ui 의 것을 안 쓰나 - 경계 검사가 internal/panel ->
// internal/api 를 금지한다 (boundary_test.go). 다시 쓰는 것이 두 벌이 되는 것과
// 다른 이유가 아래 CSP 다 - 값이 애초에 갈렸으므로 같은 것 두 벌이 아니라
// 다른 것 둘이다.
//
// 왜 GET / 만이 아니라 전부인가 - nosniff 때문이다. /api/* 가 JSON 이라
// 브라우저가 문서로 안 읽는다는 전제를 세울 수 있는데, nosniff 가 정확히 그
// 전제를 안 믿는 헤더다. 그리고 이 회차가 그 JSON 에 하네스 바이트를 싣는다.
// 한 줄이고 공짜라 안 뺀다.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		// CSP 만 /ui/ 와 갈린다. 거기는 static/ 아래 파일을 따로 내므로
		// default-src 'self' 하나로 선다. 이 페이지는 통짜 인라인이라
		// (page.go 의 indexHTML 하나에 style 과 script 와 onclick 이 산다)
		// 그 값을 글자 그대로 걸면 페이지가 죽는다 - 스타일이 빠지고 버튼이
		// 전부 안 눌린다.
		//
		// 갈린 값으로도 남는 것이 있다 - connect-src 와 img-src 와 font-src 가
		// default-src 를 물려받으므로, 주입이 일어나도 밖으로 못 보낸다.
		// 잃는 것은 주입된 스크립트 자체를 막는 힘이고, 그 자리는 화면이
		// 본문을 textContent 로만 그리는 것 하나가 진다.
		h.Set("Content-Security-Policy",
			"default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'")
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}
