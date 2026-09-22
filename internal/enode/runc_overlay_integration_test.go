//go:build linux && integration

package enode

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	execenv "github.com/taeels/enode/internal/environment"
)

func TestRuncOverlayRuntimeIntegration(t *testing.T) {
	rootfs := os.Getenv("ENODE_RUNC_ROOTFS")
	testRoot := os.Getenv("ENODE_RUNC_TEST_ROOT")
	helper := os.Getenv("ENODE_RUNC_HELPER")
	if rootfs == "" || testRoot == "" || helper == "" {
		t.Skip("set ENODE_RUNC_ROOTFS, ENODE_RUNC_TEST_ROOT, and ENODE_RUNC_HELPER for the real namespace gate")
	}
	doc, err := execenv.Parse([]byte(`api_version: enode.dev/v1alpha1
kind: execution-environment
name: runc-integration
host:
  provider: apt
  packages: [debootstrap, runc, uidmap, util-linux]
  require: {subuid_size: 65536, subgid_size: 65536, unprivileged_userns: true}
rootfs:
  builder: debootstrap
  release: noble
  arch: amd64
  apt: {components: [main], packages: [bash]}
  locale: C.UTF-8
  user: {name: sunny, uid: 1000, gid: 1000}
runtime:
  driver: runc-overlay
  workspace_target: /work
  tmp: {size: 64MiB, executable: true}
  credentials: {}
verify: {executables: [bash], locale: C.UTF-8}
`))
	if err != nil {
		t.Fatal(err)
	}
	binding := execenv.Binding{
		Scratch: filepath.Join(testRoot, "scratch"), Workspace: filepath.Join(testRoot, "workspace"),
		Store: filepath.Join(testRoot, "store"),
	}
	for _, dir := range []string{binding.Scratch, binding.Workspace} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	manifest := execenv.Manifest{Profile: execenv.ManifestProfile{Name: doc.Profile.Name, SHA256: doc.SHA256}}
	runtimeImpl, err := newRuncOverlayRuntime(doc, binding, manifest, rootfs)
	if err != nil {
		t.Fatal(err)
	}
	runtimeImpl.helper = helper
	in, out := filepath.Join(testRoot, "in"), filepath.Join(testRoot, "out")
	for _, dir := range []string{in, out} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	probeInput := filepath.Join(in, "probe")
	_ = os.Remove(probeInput)
	if err := os.WriteFile(probeInput, []byte("input\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	session, err := runtimeImpl.Open(context.Background(), RuntimeSpec{Dir: binding.Workspace, In: in, Out: out})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code, runErr := session.Run(context.Background(), ProcessSpec{
		Argv: []string{"/bin/sh", "-c", `set -eu
test "$(id -u)" = 1000
test "$(id -g)" = 1000
test "$PWD" = /work
test "$(cat "$IN/probe")" = input
! (printf bad > "$IN/probe") 2>/dev/null
printf workspace > .gate-marker
printf output > "$OUT/probe"
printf '#!/bin/sh\nexit 0\n' > /tmp/gate-exec
chmod +x /tmp/gate-exec
/tmp/gate-exec
`},
		Env: []string{"IN=" + runtimeInTarget, "OUT=" + runtimeOutTarget}, Stdout: &stdout, Stderr: &stderr,
	})
	if runErr != nil || code != 0 {
		_ = session.Close()
		t.Fatalf("run code=%d err=%v stdout=%q stderr=%q", code, runErr, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(binding.Workspace, ".gate-marker")); !os.IsNotExist(err) {
		_ = session.Close()
		t.Fatalf("overlay write reached the original workspace: %v", err)
	}
	if b, err := os.ReadFile(filepath.Join(out, "probe")); err != nil || string(b) != "output" {
		_ = session.Close()
		t.Fatalf("$OUT projection body=%q err=%v", b, err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
}
