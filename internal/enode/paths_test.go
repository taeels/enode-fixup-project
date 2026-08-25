package enode

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// rc9 가 밟은 자리다 — enode.exe 가 윈도우에서 /etc/enode/local.yaml 을
// 찾다가 죽었다. 기본 경로가 유닉스만 알면 크로스 빌드가 통과해도 안 돈다.
func TestConfigPaths_CarryNoForeignRoot(t *testing.T) {
	for _, p := range ConfigPaths() {
		if !filepath.IsAbs(p) {
			t.Fatalf("%q is not absolute", p)
		}
		if runtime.GOOS == "windows" && strings.HasPrefix(p, "/etc") {
			t.Fatalf("%q is a unix path on windows", p)
		}
	}
}

// 사용자 자리가 시스템 자리보다 앞이다 — 시연에서 노드가 발표자 노트북에
// 평범한 사용자로 sudo 없이 떠야 한다 (ADR-007 D3).
func TestConfigPaths_UserBeforeSystem(t *testing.T) {
	t.Setenv("ENODE_CONFDIR", "")
	ps := ConfigPaths()
	if len(ps) < 2 {
		t.Fatalf("want a user path and a system path: %v", ps)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory")
	}
	if !strings.HasPrefix(ps[0], home) {
		t.Fatalf("the first path %q does not sit under the home directory", ps[0])
	}
}

func TestResolveConfig_ExplicitPathWins(t *testing.T) {
	want := filepath.Join(t.TempDir(), "somewhere.yaml")
	got, tried := ResolveConfig(want)
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if tried != nil {
		t.Fatalf("an explicit path must not search: %v", tried)
	}
}

// 못 찾으면 찾아본 자리를 함께 낸다. 그것이 없으면 사람이 어디를 볼지 모른다.
func TestResolveConfig_ReportsWhereItLooked(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ENODE_CONFDIR", dir)
	t.Setenv("ENODE_CONFIG", "")

	got, tried := ResolveConfig("")
	if got != "" {
		t.Fatalf("found %q in an empty directory", got)
	}
	if len(tried) == 0 {
		t.Fatal("it did not say where it looked")
	}

	// 그 자리에 두면 찾아야 한다.
	want := filepath.Join(dir, "local.yaml")
	if err := os.WriteFile(want, []byte("mediator: http://x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, _ := ResolveConfig(""); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// ENODE_CONFDIR 은 enodectl 이 이미 열어 둔 이름이다. 둘이 같은 자리를
// 봐야 enodectl start 로 띄운 노드를 enode 가 혼자서도 찾는다.
func TestConfDir_HonoursTheEnvironment(t *testing.T) {
	t.Setenv("ENODE_CONFDIR", "/tmp/somewhere-else")
	if ConfDir() != "/tmp/somewhere-else" {
		t.Fatalf("ConfDir = %q", ConfDir())
	}
	if ConfigPaths()[0] != filepath.Join("/tmp/somewhere-else", "local.yaml") {
		t.Fatalf("ConfigPaths[0] = %q", ConfigPaths()[0])
	}
}
