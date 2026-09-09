//go:build !windows

package panel

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/taeels/enode/internal/proc"
)

// ownedProcess spawns a process whose command line contains the config path, so
// proc.OwnsConfig (which reads the args via ps on unix) reports it as owning the
// node. It writes the lock file so proc.PidFromLock finds it.
func ownedProcess(t *testing.T, cfgPath string) int {
	t.Helper()
	// A shell may exec its final sleep on macOS and lose cfgPath from argv.
	// Keep a dedicated test process whose arguments stay stable until stopped.
	cmd := exec.Command(os.Args[0], "-test.run=^TestPanelOwnedProcessHelper$", "--", cfgPath)
	cmd.Env = append(os.Environ(), "ENODE_PANEL_OWNED_PROCESS_TEST=1")
	if err := cmd.Start(); err != nil {
		t.Fatalf("cannot spawn a helper process: %v", err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() })
	pid := cmd.Process.Pid
	if err := os.WriteFile(cfgPath+".lock", []byte(strconv.Itoa(pid)), 0o600); err != nil {
		t.Fatal(err)
	}
	if proc.PidFromLock(cfgPath) != pid {
		t.Fatal("helper process did not retain its config argument")
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

func TestPanelOwnedProcessHelper(t *testing.T) {
	if os.Getenv("ENODE_PANEL_OWNED_PROCESS_TEST") != "1" {
		return
	}
	time.Sleep(30 * time.Second)
	os.Exit(0)
}
