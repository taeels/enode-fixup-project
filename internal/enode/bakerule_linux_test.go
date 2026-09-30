//go:build linux

package enode

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// IR 대조의 셸 한 줄을 호스트 sh 와 git 으로 다섯 모양에 돌린다 (계획 3절 측정 3 · 4.1 8번). git 환경은 러너의 전역
// 설정과 기본 브랜치 이름에 기대지 않는다 (계획 5절).
func TestProbeScript_OnAHostShell(t *testing.T) {
	isolatedGitEnv(t)
	probe := func(t *testing.T, ws string) irProbe {
		t.Helper()
		cmd := exec.Command("sh", "-c", probeScript)
		cmd.Dir = ws
		cmd.Env = append(os.Environ(), "ENODE_IR=ir-1")
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		_ = cmd.Run()
		return parseProbe(stdout.String(), cmd.ProcessState.ExitCode(), stderr.String())
	}
	head := func(t *testing.T, dir string) string {
		t.Helper()
		cmd := exec.Command("git", "rev-parse", "HEAD")
		cmd.Dir = dir
		b, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(b))
	}
	// origin 은 태그 둘 (annotated ir-1 · 가벼운 zz-light) 이 한 커밋에 붙은 저장소다
	origin := filepath.Join(t.TempDir(), "origin")
	committedTree(t, origin, map[string]string{"README": "r\n"})
	hostGit(t, origin, "tag", "-a", "ir-1", "-m", "ir")
	hostGit(t, origin, "tag", "zz-light")
	tagged := head(t, origin)

	t.Run("init and fetch without origin, detached", func(t *testing.T) {
		ws := t.TempDir()
		hostGit(t, ws, "init", "-q", "-b", "main")
		hostGit(t, ws, "fetch", "-q", origin, "+refs/tags/*:refs/tags/*")
		hostGit(t, ws, "checkout", "-q", "--detach", "ir-1")
		p := probe(t, ws)
		want := irProbe{Mode: "git", Head: tagged, Tagged: tagged, Tags: []string{"ir-1", "zz-light"}}
		if !reflect.DeepEqual(p, want) {
			t.Fatalf("probe = %#v", p)
		}
		if got, _ := irVerdict("ir-1", p); got != irMatch {
			t.Fatalf("verdict = %v", got)
		}
	})
	t.Run("origin and a branch", func(t *testing.T) {
		ws := filepath.Join(t.TempDir(), "ws")
		hostGit(t, filepath.Dir(ws), "clone", "-q", origin, ws)
		write(t, ws, "next", "n\n")
		hostGit(t, ws, "add", "next")
		hostGit(t, ws, "commit", "-q", "-m", "next")
		p := probe(t, ws)
		if p.Mode != "git" || p.URL != origin || p.Branch != "main" || p.Tagged != tagged || p.Head == tagged ||
			p.Tags == nil || len(p.Tags) != 0 {
			t.Fatalf("probe = %#v", p)
		}
		if got, _ := irVerdict("ir-1", p); got != irElsewhere {
			t.Fatalf("verdict = %v", got)
		}
	})
	t.Run("no repository", func(t *testing.T) {
		p := probe(t, t.TempDir())
		if p.Mode != "none" || p.Exit != 0 {
			t.Fatalf("probe = %#v", p)
		}
	})
	t.Run("repo shape reads .repo/manifests", func(t *testing.T) {
		ws := t.TempDir()
		manifests := filepath.Join(ws, ".repo", "manifests")
		if err := os.MkdirAll(filepath.Dir(manifests), 0o755); err != nil {
			t.Fatal(err)
		}
		hostGit(t, filepath.Dir(manifests), "clone", "-q", origin, manifests)
		hostGit(t, manifests, "checkout", "-q", "--detach", "ir-1")
		p := probe(t, ws)
		if p.Mode != "repo" || p.Head != tagged || p.Tagged != tagged || p.URL != origin || p.Branch != "" {
			t.Fatalf("probe = %#v", p)
		}
	})
	t.Run("a broken .git", func(t *testing.T) {
		ws := t.TempDir()
		write(t, ws, ".git", "not a gitfile\n")
		p := probe(t, ws)
		if p.Mode != "git" || p.Exit == 0 || !strings.HasPrefix(p.Stderr, "fatal:") {
			t.Fatalf("probe = %#v", p)
		}
		if got, text := irVerdict("ir-1", p); got != irUnverified || !strings.HasPrefix(text, "cannot verify ir: git exited ") {
			t.Fatalf("verdict = %v %q", got, text)
		}
	})
	t.Run("the parent repository is not read", func(t *testing.T) {
		parent := t.TempDir()
		committedTree(t, parent, map[string]string{"x": "x\n"})
		ws := filepath.Join(parent, "ws")
		if err := os.Mkdir(ws, 0o755); err != nil {
			t.Fatal(err)
		}
		if p := probe(t, ws); p.Mode != "none" {
			t.Fatalf("probe = %#v", p)
		}
	})
}
