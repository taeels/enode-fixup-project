package ui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebcamOrigin(t *testing.T) {
	for _, tc := range []struct{ url, origin string }{
		{"https://player.example.com/embed/123?autoplay=1", "https://player.example.com"},
		{"https://PLAYER.example.com:443/embed", "https://player.example.com"},
		{"https://player.example.com:8443/embed", "https://player.example.com:8443"},
	} {
		data, _ := json.Marshal(map[string]any{"webcam": map[string]string{"embedUrl": tc.url}})
		got, err := webcamOrigin(data)
		if err != nil || got != tc.origin {
			t.Fatalf("webcamOrigin(%q) = %q, %v", tc.url, got, err)
		}
	}
	for _, input := range []string{`{}`, `null`, `[]`, `{"webcam":{}}`, `{"webcam":"invalid"}`, `{"webcam":`, `{"webcam":{"embedUrl":7}}`} {
		if _, err := webcamOrigin([]byte(input)); err == nil {
			t.Errorf("accepted invalid settings %s", input)
		}
	}
	for _, value := range []string{"http://player.example.com/", "https://user:pass@player.example.com/", "https://127.0.0.1/", "https://localhost/", "https://camera.local/", "https://camera.internal/", "https://camera.localhost/", "https://player.example.com/?token=secret", "https://player.example.com/#", "https://player.example.com/#secret", "https://player.example.com:0/", "https://player.example.com:65536/", "https://player.example.com:bad/", "https://player.example.com/ bad", "https://player.example.com/\\bad", "https://player.example.com/?x=%zz", "https://player.example.com/?x=1;y=2", "https://player.example.com/?X-Amz-Credential=secret"} {
		data, _ := json.Marshal(map[string]any{"webcam": map[string]string{"embedUrl": value}})
		if _, err := webcamOrigin(data); err == nil {
			t.Errorf("accepted invalid URL %q", value)
		}
	}
	if origin, err := webcamOrigin([]byte(`{"webcam":null}`)); err != nil || origin != "" {
		t.Fatalf("disabled webcam = %q, %v", origin, err)
	}
}

func TestDemoCSPUsesOnlyConfiguredOrigin(t *testing.T) {
	h := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }), "https://player.example.com")
	for _, p := range []string{"/demo/", "/demo/index.html", "/fleet/", "/cardnews/", "/", "/demo/demo.js"} {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest("GET", p, nil))
		want := "default-src 'self'"
		if p == "/demo/" || p == "/demo/index.html" {
			want += "; frame-src https://player.example.com"
		}
		if got := r.Header().Get("Content-Security-Policy"); got != want {
			t.Errorf("CSP for %s = %q", p, got)
		}
		if r.Header().Get("X-Frame-Options") != "DENY" {
			t.Fatal("frame embedding protection changed")
		}
	}
}
