//go:build windows

package proc

import (
	"os/exec"
	"testing"
)

func TestSignalKillWindows(t *testing.T) {
	cmd := exec.Command("ping", "-n", "30", "127.0.0.1")
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot spawn ping: %v", err)
	}
	pid := cmd.Process.Pid
	if !ProcessAlive(pid) {
		t.Fatal("spawned process should be alive")
	}
	// On windows OwnsConfig can only tell liveness (no ps to read the args).
	if !OwnsConfig(pid, "anything") {
		t.Error("OwnsConfig should report a live process on windows")
	}
	// SignalStop delegates to SignalKill on windows.
	if err := SignalStop(pid); err != nil {
		t.Fatalf("SignalStop: %v", err)
	}
	_ = cmd.Wait()
	if ProcessAlive(pid) {
		t.Error("the process should be gone after SignalStop")
	}
}
