package panel

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/enode"
	"github.com/taeels/enode/internal/runctl"
)

// fakeMediator stands in for the mediator. It answers the three routes the panel
// calls: GET /v1/nodes, GET /v1/runs/{id}, POST /v1/runs/{id}/cancel.
func fakeMediator(t *testing.T, nodeID string, withLease bool) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		row := map[string]any{"node_id": nodeID, "instance": "inst-1"}
		if withLease {
			row["lease"] = map[string]any{"run_id": "run-1", "not_after": time.Now().Add(time.Minute)}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"nodes": []any{row}})
	})
	mux.HandleFunc("/v1/runs/run-1/cancel", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"run_id": "run-1", "state": "FAILED"})
	})
	mux.HandleFunc("/v1/runs/run-1", func(w http.ResponseWriter, r *http.Request) {
		started := time.Now().Add(-time.Minute)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"run_id": "run-1", "state": "RUNNING",
			"steps": []any{map[string]any{"seq": 1, "id": "build", "state": "CLAIMED", "attempt": 1, "started_at": started}},
		})
	})
	return httptest.NewServer(mux)
}

// testServer builds a Server directly (bypassing New/Derive so tests do not need
// git) pointed at a fake mediator, with a temp config path.
func testServer(t *testing.T, med *httptest.Server, nodeID string) *Server {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "ws-a.yaml")
	if err := os.WriteFile(cfgPath, []byte("mediator: x\ntoken: y\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	base := ""
	if med != nil {
		base = med.URL
	}
	return &Server{
		cfg:    Config{Node: "ws-a", ConfigPath: cfgPath, Listen: "127.0.0.1:8081", MediatorBase: base, Token: "y"},
		client: &runctl.Client{Base: base, Token: "y", HTTP: &http.Client{Timeout: 2 * time.Second}},
		ident:  enode.Identity{NodeID: nodeID, Label: "someone@host:ws-a", Principal: "someone@host"},
	}
}

func TestNewRejectsLANWithoutToken(t *testing.T) {
	if _, err := New(Config{Listen: "0.0.0.0:8081"}); err == nil {
		t.Fatal("expected an error for LAN bind without a panel token")
	}
	if _, err := New(Config{Listen: "127.0.0.1:8081"}); err != nil {
		t.Fatalf("loopback bind should be allowed without a token: %v", err)
	}
	if _, err := New(Config{Listen: "192.168.1.5:8081", PanelToken: "t"}); err != nil {
		t.Fatalf("LAN bind with a token should be allowed: %v", err)
	}
}

func TestIsLoopback(t *testing.T) {
	cases := map[string]bool{
		"127.0.0.1:8081": true, "localhost:8081": true, "[::1]:8081": true,
		"127.0.0.2:9000": true, "127.0.0.1": true,
		"0.0.0.0:8081": false, "192.168.1.5:8081": false, "example.com:80": false,
	}
	for in, want := range cases {
		if got := isLoopback(in); got != want {
			t.Errorf("isLoopback(%q)=%v want %v", in, got, want)
		}
	}
}

func TestStateReadsLocalAndMediator(t *testing.T) {
	med := fakeMediator(t, "node-xyz", true)
	defer med.Close()
	s := testServer(t, med, "node-xyz")

	// local status file
	if err := enode.WriteStatus(s.cfg.ConfigPath, enode.Capabilities{
		Caps: []contract.Capability{{Capability: "agent.reason", Attrs: map[string]string{"harness": "claude"}}},
		At:   time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	// local policy file (drained)
	if err := enode.WritePolicyFile(s.cfg.ConfigPath, enode.Policy{Drain: contract.DrainGraceful}); err != nil {
		t.Fatal(err)
	}

	st := s.state(context.Background())
	if !st.Caps.Known || len(st.Caps.Caps) != 1 || st.Caps.At == nil {
		t.Errorf("caps not read from status file: %+v", st.Caps)
	}
	if st.Drain != contract.DrainGraceful {
		t.Errorf("drain=%q want graceful", st.Drain)
	}
	if !st.Mediator.Reachable || st.Identity.Instance != "inst-1" {
		t.Errorf("mediator row not read: reachable=%v instance=%q", st.Mediator.Reachable, st.Identity.Instance)
	}
	if !st.Work.HasLease || st.Work.RunID != "run-1" || st.Work.Step != "build" || st.Work.Attempt != 1 {
		t.Errorf("current work not filled: %+v", st.Work)
	}
}

func TestStateSurvivesMediatorDown(t *testing.T) {
	// no mediator: local sources still render, mediator marked unreachable
	s := testServer(t, nil, "node-xyz")
	if err := enode.WritePolicyFile(s.cfg.ConfigPath, enode.Policy{Drain: contract.DrainAtBoundary}); err != nil {
		t.Fatal(err)
	}
	st := s.state(context.Background())
	if st.Drain != contract.DrainAtBoundary {
		t.Errorf("drain=%q want at-boundary", st.Drain)
	}
	if st.Mediator.Reachable {
		t.Error("mediator should be unreachable")
	}
	if st.Caps.Known {
		t.Error("caps should be unknown without a status file")
	}
}

func TestStateNodeNotFound(t *testing.T) {
	// mediator answers, but the row is for a different node
	med := fakeMediator(t, "other-node", true)
	defer med.Close()
	s := testServer(t, med, "node-xyz")
	st := s.state(context.Background())
	if !st.Mediator.Reachable {
		t.Error("mediator answered, should be reachable")
	}
	if st.Work.HasLease || st.Identity.Instance != "" {
		t.Errorf("must not adopt a different node's row: %+v", st)
	}
}

func TestThisNodeBadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{not json"))
	}))
	defer srv.Close()
	s := testServer(t, nil, "n")
	s.client.Base = srv.URL
	if _, found, reachable := s.thisNode(context.Background()); found || !reachable {
		t.Error("bad json should yield no row but the mediator is reachable")
	}
}

func TestFillStepStatusError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"nodes": []any{
			map[string]any{"node_id": "node-xyz", "instance": "i", "lease": map[string]any{"run_id": "run-1"}},
		}})
	})
	mux.HandleFunc("/v1/runs/run-1", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"code":500,"reason":"boom"}}`, http.StatusInternalServerError)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	s := testServer(t, nil, "node-xyz")
	s.client.Base = srv.URL
	st := s.state(context.Background())
	if !st.Work.HasLease {
		t.Error("lease should be seen from the nodes row")
	}
	if st.Work.Step != "" {
		t.Errorf("step should be empty when the status hop fails: %q", st.Work.Step)
	}
}

func TestHandleStateJSON(t *testing.T) {
	med := fakeMediator(t, "node-xyz", false)
	defer med.Close()
	s := testServer(t, med, "node-xyz")
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/state")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var st State
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		t.Fatal(err)
	}
	if st.Node != "ws-a" || st.Identity.NodeID != "node-xyz" {
		t.Errorf("unexpected state: %+v", st)
	}
	if st.Work.HasLease {
		t.Error("no lease expected")
	}
}

func TestHandleDrainAndUndrain(t *testing.T) {
	s := testServer(t, nil, "n")
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	if _, err := http.Post(srv.URL+"/api/drain?mode=at-boundary", "", nil); err != nil {
		t.Fatal(err)
	}
	if p, _ := enode.ReadPolicyFile(s.cfg.ConfigPath); p.Drain != contract.DrainAtBoundary {
		t.Errorf("drain not written: %q", p.Drain)
	}
	if _, err := http.Post(srv.URL+"/api/undrain", "", nil); err != nil {
		t.Fatal(err)
	}
	if p, _ := enode.ReadPolicyFile(s.cfg.ConfigPath); p.Drain != "" {
		t.Errorf("undrain did not clear: %q", p.Drain)
	}
}

func TestHandleDrainDefaultAndInvalid(t *testing.T) {
	s := testServer(t, nil, "n")
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	// no mode -> graceful default
	if _, err := http.Post(srv.URL+"/api/drain", "", nil); err != nil {
		t.Fatal(err)
	}
	if p, _ := enode.ReadPolicyFile(s.cfg.ConfigPath); p.Drain != contract.DrainGraceful {
		t.Errorf("default mode not graceful: %q", p.Drain)
	}
	// invalid mode -> 400
	resp, err := http.Post(srv.URL+"/api/drain?mode=bogus", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid mode status=%d want 400", resp.StatusCode)
	}
}

func TestDrainPreservesPanelToken(t *testing.T) {
	s := testServer(t, nil, "n")
	if err := enode.WritePolicyFile(s.cfg.ConfigPath, enode.Policy{PanelToken: "secret"}); err != nil {
		t.Fatal(err)
	}
	if err := s.setDrain(contract.DrainGraceful); err != nil {
		t.Fatal(err)
	}
	p, _ := enode.ReadPolicyFile(s.cfg.ConfigPath)
	if p.PanelToken != "secret" || p.Drain != contract.DrainGraceful {
		t.Errorf("panel_token not preserved across drain toggle: %+v", p)
	}
}

func TestHandleStopAlreadyStopped(t *testing.T) {
	s := testServer(t, nil, "n")
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()
	resp, err := http.Post(srv.URL+"/api/stop", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out["stopped"] != true {
		t.Errorf("expected stopped=true for a node with no lock: %+v", out)
	}
}

func TestHandleStartDidNotComeUp(t *testing.T) {
	// A harmless spawn target that exists on both platforms and exits fast without
	// holding the node lock, so start reports "did not come up". The go tool with an
	// unknown flag prints an error and exits immediately.
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go is not on PATH")
	}
	s := testServer(t, nil, "n")
	s.startBin = goBin
	t.Setenv("ENODE_STATEDIR", t.TempDir())
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()
	resp, err := http.Post(srv.URL+"/api/start", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out["running"] != false {
		t.Errorf("expected running=false since the harmless binary does not hold the lock: %+v", out)
	}
}

func TestHandleLogs(t *testing.T) {
	s := testServer(t, nil, "ws-a")
	dir := t.TempDir()
	t.Setenv("ENODE_STATEDIR", dir)
	if err := os.WriteFile(filepath.Join(dir, "ws-a.log"), []byte("line one\nline two\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/logs")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := readAll(resp)
	if !strings.Contains(b, "line two") {
		t.Errorf("log tail missing content: %q", b)
	}

	// missing log -> 404
	t.Setenv("ENODE_STATEDIR", t.TempDir())
	resp2, err := http.Get(srv.URL + "/api/logs")
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusNotFound {
		t.Errorf("missing log status=%d want 404", resp2.StatusCode)
	}
}

func TestRequireTokenOnLAN(t *testing.T) {
	s := testServer(t, nil, "n")
	s.cfg.PanelToken = "secret"
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	// no token -> 401
	resp, err := http.Get(srv.URL + "/api/state")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("without token status=%d want 401", resp.StatusCode)
	}
	// with token -> 200
	req, _ := http.NewRequest("GET", srv.URL+"/api/state", nil)
	req.Header.Set("Authorization", "Bearer secret")
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("with token status=%d want 200", resp2.StatusCode)
	}
}

func TestHandleIndexServesButtons(t *testing.T) {
	s := testServer(t, nil, "n")
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := readAll(resp)
	for _, want := range []string{"호스트 제어판", "자원 회수 (drain)", "/api/state", "/api/drain", "탐지 능력"} {
		if !strings.Contains(b, want) {
			t.Errorf("index page missing %q", want)
		}
	}
}

func readAll(resp *http.Response) (string, error) {
	b, err := io.ReadAll(resp.Body)
	return string(b), err
}
