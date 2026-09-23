package enode

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadExecutionEnvironmentResolvesProfileFromTheNodeConfig(t *testing.T) {
	dir := t.TempDir()
	profile := filepath.Join(dir, "profile.yaml")
	body := `api_version: enode.dev/v1alpha1
kind: execution-environment
name: test
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
  credentials: {ssh: readonly}
verify: {executables: [gcc], locale: en_US.UTF-8}
`
	if err := os.WriteFile(profile, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "node.yaml")
	doc, binding, err := LoadExecutionEnvironment(configPath, Local{
		Workspace:   "/workspace",
		Environment: &EnvironmentBinding{Profile: "profile.yaml", Store: "/store", Scratch: "/scratch"},
		Credentials: CredentialBinding{SSHDir: "/ssh"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if doc.Path != profile || binding.Store != "/store" || binding.SSHDir != "/ssh" {
		t.Fatalf("wrong profile or binding: %+v %+v", doc, binding)
	}
}
