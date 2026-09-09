//go:build !windows

package proc

import (
	"os/exec"
	"testing"
)

func TestSignalStopUnix(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot spawn sleep: %v", err)
	}
	pid := cmd.Process.Pid
	if !ProcessAlive(pid) {
		t.Fatal("spawned process should be alive")
	}
	// OwnsConfig reads the command line via ps; the args contain "sleep".
	if !OwnsConfig(pid, "sleep") {
		t.Error("OwnsConfig should see 'sleep' in the process args")
	}
	if OwnsConfig(pid, "zzz-not-in-args") {
		t.Error("OwnsConfig should not match an absent string")
	}
	if err := SignalStop(pid); err != nil {
		t.Fatalf("SignalStop: %v", err)
	}
	if err := cmd.Wait(); err == nil {
		t.Error("the process should have exited via the signal")
	}
}

func TestSignalKillUnix(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot spawn sleep: %v", err)
	}
	if err := SignalKill(cmd.Process.Pid); err != nil {
		t.Fatalf("SignalKill: %v", err)
	}
	if err := cmd.Wait(); err == nil {
		t.Error("the process should have been killed")
	}
}
