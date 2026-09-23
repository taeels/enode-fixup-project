//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnvironmentCommandDelegatesToTheSiblingEnode(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args")
	bin := filepath.Join(dir, "enode")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$ENODE_TEST_ARGS\"\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ENODE_BIN", bin)
	configDir := filepath.Join(dir, "config")
	t.Setenv("ENODE_CONFDIR", configDir)
	t.Setenv("ENODE_TEST_ARGS", argsFile)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "builder.yaml"), []byte("workspace: /work\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := cmdEnvironment([]string{"check", "builder", "--json"}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	want := "env\ncheck\n--config\n" + filepath.Join(configDir, "builder.yaml") + "\n--json\n"
	if string(b) != want {
		t.Fatalf("delegated argv=%q want=%q", b, want)
	}
}

func TestEnvironmentCommandRejectsInvalidInputsAndMissingEnode(t *testing.T) {
	if err := cmdEnvironment(nil); err == nil || !strings.Contains(err.Error(), "usage") {
		t.Fatalf("invalid usage error=%v", err)
	}
	if err := cmdEnvironment([]string{"check", "../escape"}); err == nil {
		t.Fatal("accepted an unsafe node name")
	}
	root := t.TempDir()
	configDir := filepath.Join(root, "config")
	t.Setenv("ENODE_CONFDIR", configDir)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "builder.yaml"), []byte("workspace: /work\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ENODE_BIN", filepath.Join(root, "missing"))
	if err := cmdEnvironment([]string{"apply", "builder"}); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("missing enode error=%v", err)
	}
	dir := t.TempDir()
	t.Setenv("ENODE_BIN", dir)
	if err := cmdEnvironment([]string{"check", "builder"}); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("directory enode error=%v", err)
	}
}
