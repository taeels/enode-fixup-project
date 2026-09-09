//go:build windows

package panel

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"testing"

	"github.com/taeels/enode/internal/proc"
)

// ownedProcess spawns a live process and writes the lock file. On windows
// proc.OwnsConfig reports liveness (there is no ps to read the args), so
// PidFromLock finds it by pid alone.
func ownedProcess(t *testing.T, cfgPath string) int {
	t.Helper()
	cmd := exec.Command("ping", "-n", "30", "127.0.0.1")
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot spawn a helper process: %v", err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() })
	pid := cmd.Process.Pid
	if err := os.WriteFile(cfgPath+".lock", []byte(strconv.Itoa(pid)), 0o600); err != nil {
		t.Fatal(err)
	}
	if proc.PidFromLock(cfgPath) != pid {
		t.Skip("liveness-based ownership is not available here")
	}
	return pid
}

func TestHandleStopCancelsThenStops(t *testing.T) {
	med := fakeMediator(t, "node-xyz", true) // has lease run-1
	defer med.Close()
	s := testServer(t, med, "node-xyz")
	ownedProcess(t, s.cfg.ConfigPath)

	srv := httptest.NewServer(s.Handler())
	defer srv.Close()
	resp, err := http.Post(srv.URL+"/api/stop", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out["cancelled"] != true || out["mediator_reachable"] != true {
		t.Errorf("expected cancel-first with a reachable mediator: %+v", out)
	}
}

func TestHandleStopMediatorDown(t *testing.T) {
	s := testServer(t, nil, "node-xyz") // no mediator
	ownedProcess(t, s.cfg.ConfigPath)

	srv := httptest.NewServer(s.Handler())
	defer srv.Close()
	resp, err := http.Post(srv.URL+"/api/stop", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out["mediator_reachable"] != false {
		t.Errorf("expected mediator unreachable: %+v", out)
	}
}
