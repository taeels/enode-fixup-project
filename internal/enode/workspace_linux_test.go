//go:build linux

package enode

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// 격리 노드의 Prepare (bake 유닛 · 계획 4.1 33번 · CG 물음 3 답 A · ADR-072 결정 3). 격리 노드는 저장소 확인 (읽기만)
// 만 하고 호스트에서 reset · clean · repo forall 을 돌리지 않는다. native 는 오늘과 같다. t.Setenv 를 쓰므로 t.Parallel
// 을 안 쓴다.

const prepareURL = "https://git.example/team/tree.git"

// fakeGitPATH 는 PATH 를 한 폴더로 바꾸고 argv 를 표지 파일에 한 줄씩 적는 가짜 git · repo 를 둔다. 가짜 git 은
// config --get remote.origin.url 에 url 을, rev-parse --abbrev-ref HEAD 에 main 을 답한다.
func fakeGitPATH(t *testing.T, url string) string {
	t.Helper()
	bin, marker := t.TempDir(), filepath.Join(t.TempDir(), "ran")
	git := "#!/bin/sh\necho \"git $*\" >> " + marker + "\ncase \"$*\" in\n" +
		"  \"config --get remote.origin.url\") echo " + url + " ;;\n" +
		"  \"rev-parse --abbrev-ref HEAD\") echo main ;;\nesac\nexit 0\n"
	repo := "#!/bin/sh\necho \"repo $*\" >> " + marker + "\nexit 0\n"
	for name, body := range map[string]string{"git": git, "repo": repo} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin)
	return marker
}

func ranLines(t *testing.T, marker string) []string {
	t.Helper()
	b, err := os.ReadFile(marker)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSpace(string(b)), "\n")
}

// shapedWorkspace 는 git 모양 (.git) 이나 repo 모양 (.repo/manifests) 의 워크스페이스다.
func shapedWorkspace(t *testing.T, shape string) (string, string) {
	t.Helper()
	ws := t.TempDir()
	if err := os.MkdirAll(filepath.Join(ws, shape), 0o755); err != nil {
		t.Fatal(err)
	}
	want := CanonicalRepoID(prepareURL)
	if shape == ".repo/manifests" {
		want += "#main"
	}
	return ws, want
}

func isolatedWorker(ws string) *Worker {
	return &Worker{Local: Local{Workspace: ws}, Runtime: newFakeRuntime(os.TempDir()), Log: discardLog()}
}

func TestPrepare_AnIsolatedNodeRunsNoHostGitWrite(t *testing.T) {
	for _, shape := range []string{".git", ".repo/manifests"} {
		t.Run(shape, func(t *testing.T) {
			marker := fakeGitPATH(t, prepareURL)
			ws, repo := shapedWorkspace(t, shape)
			w := isolatedWorker(ws)
			prep, err := w.Prepare(context.Background(), &WorkspaceSpec{Repo: repo}, discardLog())
			if err != nil || prep != PrepClean {
				t.Fatalf("Prepare = %q %v", prep, err)
			}
			want := []string{"git config --get remote.origin.url"}
			if shape == ".repo/manifests" {
				want = append(want, "git rev-parse --abbrev-ref HEAD")
			}
			if got := ranLines(t, marker); strings.Join(got, "|") != strings.Join(want, "|") {
				t.Fatalf("the host ran %q, want only the repository check %q", got, want)
			}
		})
	}
	t.Run("no repository in the contract", func(t *testing.T) {
		marker := fakeGitPATH(t, prepareURL)
		ws, _ := shapedWorkspace(t, ".git")
		prep, err := isolatedWorker(ws).Prepare(context.Background(), &WorkspaceSpec{}, discardLog())
		if err != nil || prep != PrepClean {
			t.Fatalf("Prepare = %q %v", prep, err)
		}
		if got := ranLines(t, marker); got != nil {
			t.Fatalf("the host ran %q", got)
		}
	})
}

// 격리 노드의 Prepare 는 lower 의 .git/config 가 적은 core.fsmonitor 를 실행하지 않는다 — 진짜 git. 대조군 — 같은
// 트리에서 git status 를 돌리면 표지가 생긴다 (fixture 가 살아 있다).
func TestPrepare_AnIsolatedNodeDoesNotRunTheLowersFsmonitor(t *testing.T) {
	for _, shape := range []string{"git", "repo"} {
		t.Run(shape, func(t *testing.T) {
			isolatedGitEnv(t)
			ws := t.TempDir()
			tree, want := ws, CanonicalRepoID(prepareURL)
			if shape == "repo" {
				tree, want = filepath.Join(ws, ".repo", "manifests"), want+"#main"
			}
			committedTree(t, tree, map[string]string{"default.xml": "<manifest/>\n"})
			hostGit(t, tree, "remote", "add", "origin", prepareURL)
			marker := filepath.Join(t.TempDir(), "ran")
			hook := filepath.Join(t.TempDir(), "hook")
			if err := os.WriteFile(hook, []byte("#!/bin/sh\necho \"$0 $*\" >> "+marker+"\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			hostGit(t, tree, "config", "core.fsmonitor", hook)
			write(t, tree, "default.xml", "<manifest changed/>\n")
			prep, err := isolatedWorker(ws).Prepare(context.Background(), &WorkspaceSpec{Repo: want}, discardLog())
			if err != nil || prep != PrepClean {
				t.Fatalf("Prepare = %q %v", prep, err)
			}
			if b, err := os.ReadFile(marker); !os.IsNotExist(err) {
				t.Fatalf("the host ran the lower's fsmonitor: %q", b)
			}
			cmd := exec.Command("git", "status", "--short")
			cmd.Dir = tree
			_ = cmd.Run()
			if _, err := os.Stat(marker); err != nil {
				t.Fatalf("git status did not run the fsmonitor; the fixture is dead: %v", err)
			}
		})
	}
}

// native 노드 (Runtime nil · NativeRuntime{}) 는 오늘처럼 reset --hard 다음 clean -df — repo 모양은 repo forall.
func TestPrepare_ANativeNodeStillResetsAndCleans(t *testing.T) {
	for name, rt := range map[string]StepRuntime{"no runtime": nil, "native": NativeRuntime{}} {
		for _, shape := range []string{".git", ".repo/manifests"} {
			t.Run(name+" "+shape, func(t *testing.T) {
				marker := fakeGitPATH(t, prepareURL)
				ws, repo := shapedWorkspace(t, shape)
				if shape == ".repo/manifests" {
					if err := os.MkdirAll(filepath.Join(ws, ".repo"), 0o755); err != nil {
						t.Fatal(err)
					}
				}
				w := &Worker{Local: Local{Workspace: ws}, Runtime: rt, Log: discardLog()}
				prep, err := w.Prepare(context.Background(), &WorkspaceSpec{Repo: repo}, discardLog())
				if err != nil || prep != PrepClean {
					t.Fatalf("Prepare = %q %v", prep, err)
				}
				got := strings.Join(ranLines(t, marker), "|")
				want := "git reset --hard|git clean -df"
				if shape == ".repo/manifests" {
					want = "repo forall -c git reset --hard|repo forall -c git clean -df"
				}
				if !strings.HasSuffix(got, want) {
					t.Fatalf("the host ran %q, want it to end with %q", got, want)
				}
			})
		}
	}
}

// 격리 노드도 다른 저장소면 오늘 문장 그대로 거절한다 — reset · clean 없이.
func TestPrepare_AnIsolatedNodeStillRefusesAnotherRepo(t *testing.T) {
	marker := fakeGitPATH(t, "https://git.example/other.git")
	ws, repo := shapedWorkspace(t, ".git")
	_, err := isolatedWorker(ws).Prepare(context.Background(), &WorkspaceSpec{Repo: repo}, discardLog())
	if err == nil || !strings.HasPrefix(err.Error(), "workspace repository mismatch: node has ") {
		t.Fatalf("Prepare = %v", err)
	}
	if got := ranLines(t, marker); len(got) != 1 {
		t.Fatalf("the host ran %q", got)
	}
}
