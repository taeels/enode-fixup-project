package ui_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/taeels/enode/internal/api/ui"
)

func get(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	ui.Handler().ServeHTTP(rec, req)
	return rec
}

func TestHandlerServesLanding(t *testing.T) {
	rec := get(t, "/ui/")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /ui/ status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `data-testid="landing-guest-login-button"`) {
		t.Fatal("landing page is missing the guest login button")
	}
	if !strings.Contains(rec.Body.String(), `data-testid="landing-admin-token-input"`) {
		t.Fatal("landing page is missing the admin token placeholder")
	}
}

func TestHandlerServesCardnews(t *testing.T) {
	rec := get(t, "/ui/cardnews/")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /ui/cardnews/ status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, testID := range []string{
		"cardnews-close-button",
		"cardnews-prev-button",
		"cardnews-next-button",
		"cardnews-progress-dot-0",
		"cardnews-progress-dot-3",
		"cardnews-progress-dot-4",
		"cardnews-story-video",
		"cardnews-illustration-0",
		"cardnews-illustration-1",
		"cardnews-illustration-2",
		"cardnews-illustration-3",
	} {
		if !strings.Contains(body, `data-testid="`+testID+`"`) {
			t.Errorf("cardnews page is missing element %q", testID)
		}
	}
}

func TestHandlerServesDemo(t *testing.T) {
	rec := get(t, "/ui/demo/")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /ui/demo/ status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `data-testid="demo-replay-cardnews-link"`) {
		t.Fatal("demo page is missing the replay link")
	}
}

func TestHandlerServesStaticAssets(t *testing.T) {
	paths := []string{
		"/ui/landing.css",
		"/ui/landing.js",
		"/ui/shared/guest.js",
		"/ui/cardnews/cardnews.js",
		"/ui/cardnews/cardnews.css",
		"/ui/cardnews/card5.mp4",
		"/ui/cardnews/card1.png",
		"/ui/cardnews/card2.png",
		"/ui/cardnews/card3.png",
		"/ui/cardnews/card4.png",
		"/ui/demo/demo.js",
		"/ui/demo/demo.css",
	}
	for _, p := range paths {
		rec := get(t, p)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s status = %d, want 200", p, rec.Code)
		}
	}
}

func TestHandlerUnknownPathIs404(t *testing.T) {
	rec := get(t, "/ui/does-not-exist")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /ui/does-not-exist status = %d, want 404", rec.Code)
	}
}

func TestHandlerNoDirectoryListing(t *testing.T) {
	// shared/ 에 index.html 이 있어야 디렉터리 목록 대신 그것을 낸다
	// (nfr-design.md 「SECURITY-09 하드닝」).
	rec := get(t, "/ui/shared/")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /ui/shared/ status = %d, want 200", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "Directory listing") {
		t.Fatal("shared/ served a directory listing instead of index.html")
	}
}

func TestHandlerSecurityHeaders(t *testing.T) {
	var settings struct {
		Webcam *struct {
			EmbedURL string `json:"embedUrl"`
		} `json:"webcam"`
	}
	if err := json.Unmarshal(get(t, "/ui/demo/settings.json").Body.Bytes(), &settings); err != nil {
		t.Fatal(err)
	}
	demoCSP := "default-src 'self'"
	if settings.Webcam != nil {
		embed, err := url.Parse(settings.Webcam.EmbedURL)
		if err != nil || embed.Scheme != "https" || embed.Host == "" {
			t.Fatal("configured webcam must use a public HTTPS embed URL")
		}
		demoCSP += "; frame-src https://" + strings.TrimSuffix(embed.Host, ":443")
	}
	want := map[string]string{
		"Content-Security-Policy":   "default-src 'self'",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
	}
	for _, path := range []string{"/ui/", "/ui/cardnews/", "/ui/demo/", "/ui/demo/index.html", "/ui/fleet/", "/ui/does-not-exist"} {
		rec := get(t, path)
		for header, value := range want {
			if header == "Content-Security-Policy" && (path == "/ui/demo/" || path == "/ui/demo/index.html") {
				value = demoCSP
			}
			if got := rec.Header().Get(header); got != value {
				t.Errorf("GET %s header %s = %q, want %q", path, header, got, value)
			}
		}
	}
}
