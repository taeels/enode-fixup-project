package api_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/taeels/enode/internal/api"
	"github.com/taeels/enode/internal/config"
	"github.com/taeels/enode/internal/store"
)

func TestEntryRedirectOnlyMatchesBrowserRoot(t *testing.T) {
	for _, demo := range []bool{false, true} {
		name := "fleet"
		if demo {
			name = "demo"
		}
		t.Run(name, func(t *testing.T) {
			cfg := config.Default()
			cfg.Demo = demo
			cfg.Token = "entry-test-token"
			handler := api.New(&store.Store{}, cfg, slog.New(slog.NewTextHandler(io.Discard, nil))).Handler()
			for _, method := range []string{http.MethodGet, http.MethodHead} {
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, httptest.NewRequest(method, "/", nil))
				if w.Code != http.StatusFound || w.Header().Get("Location") != "/ui/" {
					t.Fatalf("%s /: got %d location=%q, want 302 /ui/", method, w.Code, w.Header().Get("Location"))
				}
			}
			for _, tc := range []struct {
				method string
				path   string
				status int
			}{
				{http.MethodGet, "/ui/", http.StatusOK},
				{http.MethodGet, "/missing", http.StatusNotFound},
				{http.MethodGet, "/v1/missing", http.StatusNotFound},
				{http.MethodPost, "/", http.StatusMethodNotAllowed},
				{http.MethodPost, "/v1/runs", http.StatusUnauthorized},
			} {
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
				if w.Code != tc.status || w.Header().Get("Location") != "" {
					t.Fatalf("%s %s: got %d location=%q, want %d without redirect", tc.method, tc.path, w.Code, w.Header().Get("Location"), tc.status)
				}
			}
		})
	}
}
