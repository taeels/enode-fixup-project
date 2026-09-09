package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/taeels/enode/internal/config"
	"github.com/taeels/enode/internal/record"
	"github.com/taeels/enode/internal/store"
)

// demo-back이 인수할 내부 접점이다. 공개 데모 라우트나 하드웨어 픽스처를
// 대신하지 않고, 실제 submitterKey부터 큐 생성·재접수·승격까지 검증한다.
func TestDemoQueue_SubmitterSurvivesAdmissionRetryAndPromotion(t *testing.T) {
	dsn := os.Getenv("ENODE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("ENODE_TEST_DATABASE_URL is unset; use scripts/testdb.sh")
	}
	ctx := context.Background()
	st, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := st.Truncate(ctx); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	t.Cleanup(func() {
		_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return os.Chmod(path, 0o755)
			}
			return os.Chmod(path, 0o644)
		})
	})
	st.Records = record.New(root)
	cfg := config.Default()
	cfg.Token = "demo-intake-test-token"
	cfg.Demo = true
	s := New(st, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	handler := s.Handler()

	call := func(method, path, body, name string, internal bool, want int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Authorization", "Bearer "+cfg.Token)
		w := httptest.NewRecorder()
		if internal {
			r = r.WithContext(context.WithValue(r.Context(), submitterKey, name))
			s.auth(s.postRuns)(w, r)
		} else {
			// 공개 읽기와 일반 제출은 표시 이름 헤더를 신뢰하지 않는다.
			r.Header.Set("X-Enode-Submitter", "untrusted-header")
			if method == http.MethodGet {
				r.Header.Del("Authorization")
			}
			handler.ServeHTTP(w, r)
		}
		if w.Code != want {
			t.Fatalf("%s %s: got %d %s, want %d", method, path, w.Code, w.Body.String(), want)
		}
		return w
	}
	contractBody := func(id string) string {
		return `{"run_id":"` + id + `","requires":[{"as":"worker","capability":"agent.reason","role":"demo-intake"}],"steps":[{"id":"step","uses":"worker","run":["true"]}]}`
	}
	assertRow := func(id, state, name string) {
		t.Helper()
		w := call("GET", "/v1/runs", "", "", false, 200)
		var response runsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		count := 0
		for _, row := range response.Runs {
			if row.RunID != id {
				continue
			}
			count++
			if row.State != state || row.Submitter != name {
				t.Fatalf("run %s: state=%s submitter=%q, want %s %q", id, row.State, row.Submitter, state, name)
			}
		}
		if count != 1 {
			t.Fatalf("run %s: got %d rows, want 1", id, count)
		}
	}

	call("POST", "/v1/nodes", `{"node_id":"demo-intake-node","label":"Demo intake node","capabilities":[{"capability":"agent.reason","attrs":{"role":"demo-intake"}}]}`, "", false, 200)
	call("POST", "/v1/runs", contractBody("demo-intake-holder"), "guest-first-visitor", true, 201)
	assertRow("demo-intake-holder", store.StateRunning, "guest-first-visitor")
	// 한글 두 단어도 내부 전달과 DB를 지나 원문 그대로 남아야 한다.
	name := "\ubc1d\uc740 \uc218\ub2ec"
	call("POST", "/v1/runs", contractBody("demo-intake-queued"), name, true, 202)
	assertRow("demo-intake-queued", store.StateQueued, name)
	call("POST", "/v1/runs", contractBody("demo-intake-queued"), "guest-retry-name", true, 200)
	assertRow("demo-intake-queued", store.StateQueued, name)
	call("POST", "/v1/runs", contractBody("demo-intake-no-context"), "", false, 202)
	assertRow("demo-intake-no-context", store.StateQueued, "")
	call("POST", "/v1/runs/demo-intake-holder/cancel", "", "", false, 200)
	assertRow("demo-intake-queued", store.StateRunning, name)
	call("POST", "/v1/runs", contractBody("demo-intake-queued"), "guest-retry-name", true, 200)
	assertRow("demo-intake-queued", store.StateRunning, name)
}
