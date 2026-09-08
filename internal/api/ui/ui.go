// Package ui 는 GET /ui/ 아래 정적 화면을 낸다 — 랜딩 · 카드뉴스 · 데모
// 세 번들이다.
//
// internal/store 를 참조하지 않는다(구조 불변식, constraints.md —
// "internal/api/ui -> internal/store 금지. 화면은 정적 파일이다").
package ui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static
var staticFiles embed.FS

// Handler 는 /ui/ 아래 정적 파일을 낸다. 모든 응답에 SECURITY-04 보안
// 헤더를 싣는다(nfr-design.md 「SECURITY-04 HTTP 보안 헤더」).
//
// 무인증이다 — enode-features.md §3.4.1 의 보안 요구사항이 이 표면을
// 명시적 public 으로 정했다(nfr-design.md 「SECURITY-08」).
func Handler() http.Handler {
	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		// static/ 은 //go:embed 로 빌드에 포함되므로 이 경로가 실패하는
		// 것은 컴파일 타임에 이미 걸렸을 실수뿐이다 — 런타임 방어가 아니다.
		panic(err)
	}
	fileServer := http.FileServerFS(sub)
	return http.StripPrefix("/ui/", withSecurityHeaders(fileServer))
}

// withSecurityHeaders 는 HTML 을 내는 모든 응답에 다섯 헤더를 싣는다.
func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'self'")
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}
