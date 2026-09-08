// Package ui 는 GET /ui/ 아래 정적 화면을 낸다 — 랜딩 · 카드뉴스 · 데모
// 세 번들이다.
//
// internal/store 를 참조하지 않는다(구조 불변식, constraints.md —
// "internal/api/ui -> internal/store 금지. 화면은 정적 파일이다").
package ui

import (
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
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
	// 새 모듈 디렉터리도 목록을 노출하지 않는다.
	files := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if name == "" {
			name = "."
		}
		if info, err := fs.Stat(sub, name); err == nil && info.IsDir() {
			if _, err := fs.Stat(sub, path.Join(name, "index.html")); err != nil {
				http.NotFound(w, r)
				return
			}
		}
		fileServer.ServeHTTP(w, r)
	})
	return http.StripPrefix("/ui/", withSecurityHeaders(files))
}

// withSecurityHeaders 는 HTML 을 내는 모든 응답에 다섯 헤더를 싣는다.
func withSecurityHeaders(next http.Handler) http.Handler {
	settings, _ := staticFiles.ReadFile("static/demo/settings.json")
	origin, _ := webcamOrigin(settings) // 잘못된 공개 설정은 외부 프레임을 허용하지 않는다.
	return securityHeaders(next, origin)
}

func securityHeaders(next http.Handler, origin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'self'")
		if origin != "" && (r.URL.Path == "demo/" || r.URL.Path == "demo/index.html" || r.URL.Path == "/demo/" || r.URL.Path == "/demo/index.html") {
			h.Set("Content-Security-Policy", "default-src 'self'; frame-src "+origin)
		}
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

var publicHost = regexp.MustCompile(`^[a-z0-9]+(?:[a-z0-9-]*[a-z0-9])?(?:\.[a-z0-9]+(?:[a-z0-9-]*[a-z0-9])?)+$`)
var numericHost = regexp.MustCompile(`^[0-9.]+$`)
var privateQuery = regexp.MustCompile(`(?i)token|secret|password|authorization|signature|credential|api.?key|^sig$|^key$|^auth$`)

// webcamOrigin은 브라우저와 같은 embed 설정에서 허용 origin 하나만 추출한다.
func webcamOrigin(data []byte) (string, error) {
	invalid := errors.New("invalid public webcam settings")
	var settings struct {
		Webcam json.RawMessage `json:"webcam"`
	}
	if json.Unmarshal(data, &settings) != nil || len(settings.Webcam) == 0 {
		return "", invalid
	}
	if string(settings.Webcam) == "null" {
		return "", nil
	}
	var camera struct {
		EmbedURL string `json:"embedUrl"`
	}
	if json.Unmarshal(settings.Webcam, &camera) != nil {
		return "", invalid
	}
	raw := camera.EmbedURL
	if !strings.HasPrefix(raw, "https://") || len(raw) > 4096 {
		return "", invalid
	}
	for _, c := range raw {
		if c < 33 || c > 126 || c == '\\' {
			return "", invalid
		}
	}
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || u.Fragment != "" || strings.Contains(raw, "#") {
		return "", invalid
	}
	host := strings.ToLower(u.Hostname())
	if !publicHost.MatchString(host) || numericHost.MatchString(host) || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") {
		return "", invalid
	}
	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return "", invalid
	}
	for key := range query {
		if privateQuery.MatchString(key) {
			return "", invalid
		}
	}
	origin := "https://" + host
	if port := u.Port(); port != "" {
		p, err := strconv.Atoi(port)
		if err != nil || p < 1 || p > 65535 {
			return "", invalid
		}
		if p != 443 {
			origin += ":" + strconv.Itoa(p)
		}
	}
	return origin, nil
}
