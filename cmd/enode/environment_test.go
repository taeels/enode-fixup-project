package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	execenv "github.com/taeels/enode/internal/environment"
)

const cliEnvironmentProfile = `api_version: enode.dev/v1alpha1
kind: execution-environment
name: cli-test
host:
  provider: apt
  packages: [debootstrap, runc, uidmap, util-linux]
  require: {subuid_size: 65536, subgid_size: 65536, unprivileged_userns: true}
rootfs:
  builder: debootstrap
  release: noble
  arch: amd64
  apt: {components: [main], packages: [gcc]}
  locale: en_US.UTF-8
  user: {name: enode, uid: 1000, gid: 1000}
runtime:
  driver: runc-overlay
  workspace_target: /work
  tmp: {size: 256MiB, executable: true}
  credentials: {}
verify: {executables: [gcc], locale: en_US.UTF-8}
`

func writeEnvironmentCLIConfig(t *testing.T) (config, store string) {
	t.Helper()
	dir := t.TempDir()
	profile := filepath.Join(dir, "profile.yaml")
	if err := os.WriteFile(profile, []byte(cliEnvironmentProfile), 0o644); err != nil {
		t.Fatal(err)
	}
	store = filepath.Join(dir, "store")
	config = filepath.Join(dir, "node.yaml")
	body := "workspace: " + filepath.Join(dir, "workspace") + "\n" +
		"environment:\n" +
		"  profile: profile.yaml\n" +
		"  store: " + store + "\n" +
		"  scratch: " + filepath.Join(dir, "scratch") + "\n"
	if err := os.WriteFile(config, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return config, store
}

func TestEnvironmentCheckReportsAnUnlinkedProductRuntime(t *testing.T) {
	config, _ := writeEnvironmentCLIConfig(t)
	var code int
	stdout, _ := captureOutput(t, func() {
		code = runEnvironmentCmd([]string{"check", "--config", config, "--json"})
	})
	if code != 2 {
		t.Fatalf("want non-ready exit 2, got %d: %s", code, stdout)
	}
	var report execenv.Report
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("decode report: %v: %s", err, stdout)
	}
	if report.State != execenv.StateUnsupported {
		t.Fatalf("check claimed %s: %+v", report.State, report.Facts)
	}
	found := false
	for _, fact := range report.Facts {
		if fact.Name == "runtime.product_driver" && strings.Contains(fact.Observed, "not linked") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing runtime wiring fact: %+v", report.Facts)
	}
}

func TestEnvironmentApplyDoesNotMutateWhenTheRuntimeIsUnlinked(t *testing.T) {
	config, store := writeEnvironmentCLIConfig(t)
	var code int
	_, stderr := captureOutput(t, func() {
		code = runEnvironmentCmd([]string{"apply", "--config", config})
	})
	if code != 1 || !strings.Contains(stderr, "unsupported") {
		t.Fatalf("want fail-closed apply, code=%d stderr=%q", code, stderr)
	}
	if _, err := os.Stat(store); !os.IsNotExist(err) {
		t.Fatalf("unsupported apply mutated the store: %v", err)
	}
}
