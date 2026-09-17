package panel

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 다섯이 모든 응답에 붙는다. 문서만이 아니라 /api/* 에도 붙는 이유가 nosniff 다 -
// JSON 이라 브라우저가 문서로 안 읽는다는 전제를 안 믿는 헤더이고, 이 회차가
// 그 JSON 에 하네스 바이트를 싣는다.
func TestSecurityHeadersAreOnEveryResponse(t *testing.T) {
	s := testServer(t, nil, "n")
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	want := map[string]string{
		"Content-Security-Policy":   "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
	}
	for _, path := range []string{"/", "/api/transcript", "/api/state"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close() //nolint:errcheck
		for k, v := range want {
			if got := resp.Header.Get(k); got != v {
				t.Errorf("%s: header %s = %q, want %q", path, k, got, v)
			}
		}
	}
}

// CSP 가 /ui/ 의 것과 갈린 자리를 고정한다. 이 페이지는 통짜 인라인이라
// default-src 'self' 하나로는 스타일과 스크립트가 전부 막힌다 - 요구가 적은
// "같은 다섯 줄" 에서 이 한 줄만 갈렸고, 그 어긋남을 여기서 센다.
func TestPanelCSPAllowsItsOwnInlineButNothingOutbound(t *testing.T) {
	s := testServer(t, nil, "n")
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close() //nolint:errcheck
	csp := resp.Header.Get("Content-Security-Policy")

	// 인라인이 살아야 페이지가 뜬다.
	for _, need := range []string{"style-src 'self' 'unsafe-inline'", "script-src 'self' 'unsafe-inline'"} {
		if !strings.Contains(csp, need) {
			t.Errorf("the panel is one inline page, so the CSP must carry %q: %q", need, csp)
		}
	}
	// 나가는 길은 막혀 있어야 한다 - connect-src 와 img-src 가 default-src 를
	// 물려받으므로 default-src 가 'self' 인 것이 그 자물쇠다.
	if !strings.HasPrefix(csp, "default-src 'self'") {
		t.Errorf("default-src must stay 'self' so injected code cannot phone out: %q", csp)
	}
	for _, banned := range []string{"connect-src", "img-src", "default-src *", "'unsafe-eval'"} {
		if strings.Contains(csp, banned) {
			t.Errorf("the CSP must not widen %q: %q", banned, csp)
		}
	}
}

// 401 에도 붙는다 - 브라우저가 그리는 문서라 같은 헤더가 필요하다.
func TestSecurityHeadersSurviveTheTokenWall(t *testing.T) {
	s := testServer(t, nil, "n")
	s.cfg.PanelToken = "secret"
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401 without the token, got %d", resp.StatusCode)
	}
	if resp.Header.Get("X-Frame-Options") != "DENY" {
		t.Error("a 401 is still a document the browser renders")
	}
}

// 이 페이지가 밖에서 가져오는 것이 0 이라야 위 CSP 가 실제로 성립한다.
// 하나라도 들어오면 페이지가 조용히 반쯤 깨지고, 그 깨짐은 CSP 위반이라
// 화면에 오류로 안 뜬다.
func TestPanelPageFetchesNothingFromOutside(t *testing.T) {
	for _, banned := range []string{"https://", "http://", "@import", "src=\"//", "data:"} {
		if strings.Contains(indexHTML, banned) {
			t.Errorf("the panel page must not reach outside its own origin, but it contains %q", banned)
		}
	}
}
