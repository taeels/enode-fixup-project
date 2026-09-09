package panel

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/taeels/enode/internal/enode"
)

func TestHandleTranscript(t *testing.T) {
	s := testServer(t, nil, "n")
	// no ring yet -> available:false
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()
	resp, _ := http.Get(srv.URL + "/api/transcript")
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	resp.Body.Close()
	if out["available"] != false {
		t.Fatalf("no ring should give available=false: %+v", out)
	}

	// write a ring beside the config, then it should read
	r, err := enode.OpenRing(enode.TranscriptPath(s.cfg.ConfigPath), 4096)
	if err != nil {
		t.Fatal(err)
	}
	r.Write([]byte("agent is thinking..."))
	r.Close() //nolint:errcheck

	resp2, _ := http.Get(srv.URL + "/api/transcript")
	var out2 map[string]any
	_ = json.NewDecoder(resp2.Body).Decode(&out2)
	resp2.Body.Close()
	if out2["available"] != true || out2["data"] != "agent is thinking..." {
		t.Fatalf("ring content not served: %+v", out2)
	}
}

func TestHandleRunsFiltersByNode(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/runs", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"runs": []any{
			map[string]any{"run_id": "mine", "state": "SUCCEEDED",
				"assigned": []any{map[string]any{"nodes": []any{map[string]any{"node": "node-xyz"}}}}},
			map[string]any{"run_id": "theirs", "state": "FAILED",
				"assigned": []any{map[string]any{"nodes": []any{map[string]any{"node": "other"}}}}},
		}})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	s := testServer(t, nil, "node-xyz")
	s.client.Base = srv.URL

	psrv := httptest.NewServer(s.Handler())
	defer psrv.Close()
	resp, err := http.Get(psrv.URL + "/api/runs")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out struct {
		Runs []pastRun `json:"runs"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if len(out.Runs) != 1 || out.Runs[0].RunID != "mine" {
		t.Fatalf("expected only this node's run: %+v", out.Runs)
	}
}

func TestHandleRecordExtractsLogs(t *testing.T) {
	// build a tar with logs/01-build.log and a non-log entry
	var tarbuf bytes.Buffer
	tw := tar.NewWriter(&tarbuf)
	write := func(name, body string) {
		_ = tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body))})
		_, _ = tw.Write([]byte(body))
	}
	write("logs/01-build.log", "compiling the kernel")
	write("verdict.json", "{}")
	_ = tw.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/runs/r1/record", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(tarbuf.Bytes())
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	s := testServer(t, nil, "n")
	s.client.Base = srv.URL

	psrv := httptest.NewServer(s.Handler())
	defer psrv.Close()
	resp, err := http.Get(psrv.URL + "/api/record?run=r1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out struct {
		Logs []recordLog `json:"logs"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if len(out.Logs) != 1 || out.Logs[0].Name != "logs/01-build.log" || out.Logs[0].Content != "compiling the kernel" {
		t.Fatalf("record did not extract logs/*.log only: %+v", out.Logs)
	}
}

func TestHandleRecordNeedsRun(t *testing.T) {
	s := testServer(t, nil, "n")
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()
	resp, _ := http.Get(srv.URL + "/api/record")
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing run should be 400, got %d", resp.StatusCode)
	}
}
