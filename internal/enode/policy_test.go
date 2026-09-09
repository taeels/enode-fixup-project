package enode

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/taeels/enode/internal/contract"
)

func policyLog() (*slog.Logger, *bytes.Buffer) {
	var b bytes.Buffer
	return slog.New(slog.NewTextHandler(&b, nil)), &b
}

func TestPolicyPath_SitsNextToTheConfigWithTheSameStem(t *testing.T) {
	if got := PolicyPath("/etc/enode/local.yaml"); got != "/etc/enode/local.policy.yaml" {
		t.Fatalf("PolicyPath = %q", got)
	}
	if got := PolicyPath("/x/board-02.yml"); got != "/x/board-02.policy.yaml" {
		t.Fatalf("PolicyPath = %q", got)
	}
}

func TestPolicy_MissingFileMeansNotDrainingAndSaysNothing(t *testing.T) {
	log, out := policyLog()
	r := &policyReader{Path: filepath.Join(t.TempDir(), "local.policy.yaml"), Log: log}
	if p := r.Read(); p.Drain != contract.DrainNone {
		t.Fatalf("missing file read as %q", p.Drain)
	}
	if out.Len() != 0 {
		t.Fatalf("a missing policy file was logged: %s", out.String())
	}
}

func TestPolicy_ReadsTheTwoModesAndIgnoresUnknownKeys(t *testing.T) {
	log, _ := policyLog()
	path := filepath.Join(t.TempDir(), "local.policy.yaml")
	r := &policyReader{Path: path, Log: log}
	for _, mode := range []string{contract.DrainGraceful, contract.DrainAtBoundary, contract.DrainNone} {
		if err := os.WriteFile(path, []byte("drain: "+mode+"\npanel_token: later\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if p := r.Read(); p.Drain != mode {
			t.Fatalf("wrote %q, read %q", mode, p.Drain)
		}
	}
}

func TestPolicy_BadValueFoldsToNoneAndWarnsOncePerCause(t *testing.T) {
	log, out := policyLog()
	path := filepath.Join(t.TempDir(), "local.policy.yaml")
	r := &policyReader{Path: path, Log: log}
	if err := os.WriteFile(path, []byte("drain: sideways\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if p := r.Read(); p.Drain != contract.DrainNone {
			t.Fatalf("an unknown value was not folded: %q", p.Drain)
		}
	}
	if n := strings.Count(out.String(), "unknown drain policy"); n != 1 {
		t.Fatalf("the same cause was logged %d times, want once: %s", n, out.String())
	}
	if !strings.Contains(out.String(), "sideways") {
		t.Fatal("the offending value is not in the node log; the owner cannot fix what they cannot see")
	}
	// 원인이 바뀌면 다시 찍는다
	if err := os.WriteFile(path, []byte("drain: [\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if p := r.Read(); p.Drain != contract.DrainNone {
		t.Fatalf("a broken file was not folded: %q", p.Drain)
	}
	if !strings.Contains(out.String(), "cannot parse the policy file") {
		t.Fatalf("a new cause was not logged: %s", out.String())
	}
	// 고치면 조용히 돌아온다
	if err := os.WriteFile(path, []byte("drain: graceful\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	before := out.Len()
	if p := r.Read(); p.Drain != contract.DrainGraceful {
		t.Fatalf("the fixed file was not read: %q", p.Drain)
	}
	if out.Len() != before {
		t.Fatalf("recovery was logged: %s", out.String()[before:])
	}
}

func TestPolicy_LooseFilePermissionsAreWarnedOnce(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mode bits are not the permission model on windows")
	}
	log, out := policyLog()
	path := filepath.Join(t.TempDir(), "local.policy.yaml")
	r := &policyReader{Path: path, Log: log}
	if err := os.WriteFile(path, []byte("drain: at-boundary\n"), 0o666); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o666); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if p := r.Read(); p.Drain != contract.DrainAtBoundary {
			t.Fatalf("a loose file was refused: %q — drain is reclaim, not destruction", p.Drain)
		}
	}
	if n := strings.Count(out.String(), "writable by others"); n != 1 {
		t.Fatalf("permission warning logged %d times, want once: %s", n, out.String())
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	r.Read()
	if err := os.Chmod(path, 0o666); err != nil {
		t.Fatal(err)
	}
	r.Read()
	if n := strings.Count(out.String(), "writable by others"); n != 2 {
		t.Fatalf("a repeated loosening was not warned again: %d", n)
	}
}
