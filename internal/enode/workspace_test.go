package enode

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@e",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@e")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %s", args, out)
	}
	return string(out)
}

// ★ 순서가 셋이고 뒤바꾸면 안 된다 ★ (ADR-017 결정 4 + ADR-021)
//
//	① reset --hard  추적 변경을 버린다 — 안 하면 checkout 이 거절된다
//	② checkout      목표 리비전으로
//	③ clean -df     ★ 목표 리비전의 .gitignore 로 ★ 청소한다
//
// ③ 을 ② 앞에 두면 이전 리비전의 무시 규칙으로 청소하게 되고,
// ★ 데워둔 빌드 캐시가 날아간다 ★. 실측에서 밟았다.
func TestPrepareKeepsBuildCacheDropsJunk(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	git(t, dir, "remote", "add", "origin", "ssh://git@gerrit.corp:29418/kernel/linux")
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.o\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "drv.c"), []byte("parent\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-qm", "parent")
	rev := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))

	// 워크스페이스를 더럽힌다
	os.WriteFile(filepath.Join(dir, "drv.o"), []byte("warm cache"), 0o644) // ★ 무시됨 — 살아야 한다
	os.WriteFile(filepath.Join(dir, "junk.c"), []byte("junk"), 0o644)      // 추적 안 됨 — 죽어야 한다
	os.WriteFile(filepath.Join(dir, "drv.c"), []byte("tampered\n"), 0o644) // 추적 변경 — 되돌려져야 한다

	w := &Worker{Local: Local{Workspace: dir}}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := w.Prepare(context.Background(),
		&WorkspaceSpec{Repo: "gerrit.corp/kernel/linux", Rev: rev}, log); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "drv.o")); err != nil {
		t.Fatal("★ 데워둔 빌드 캐시가 날아갔다 ★ — ADR-007 준비물과 §3.1 전제가 무너진다")
	}
	if _, err := os.Stat(filepath.Join(dir, "junk.c")); err == nil {
		t.Fatal("추적 안 되는 쓰레기가 남았다 — 알려진 상태가 아니다")
	}
	b, _ := os.ReadFile(filepath.Join(dir, "drv.c"))
	if string(b) != "parent\n" {
		t.Fatalf("추적 파일의 오염이 안 되돌려졌다: %q", b)
	}
}

// 계약이 다른 저장소를 가리키면 거절한다 — 조용히 틀린 것을 빌드하면 안 된다.
func TestPrepareRefusesWrongRepo(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	git(t, dir, "remote", "add", "origin", "https://other.corp/x/y")
	w := &Worker{Local: Local{Workspace: dir}}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	err := w.Prepare(context.Background(),
		&WorkspaceSpec{Repo: "gerrit.corp/kernel/linux"}, log)
	if err == nil {
		t.Fatal("★ 다른 저장소인데 통과했다 ★")
	}
}

// 워크스페이스가 없는 노드에 워크스페이스 단계가 오면 거절한다.
func TestPrepareRefusesWithoutWorkspace(t *testing.T) {
	w := &Worker{Local: Local{}}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := w.Prepare(context.Background(),
		&WorkspaceSpec{Repo: "x/y"}, log); err == nil {
		t.Fatal("워크스페이스가 없는데 통과했다")
	}
	// 워크스페이스를 요구하지 않는 단계는 그냥 지나간다
	if err := w.Prepare(context.Background(), nil, log); err != nil {
		t.Fatalf("워크스페이스가 필요 없는 단계가 거절됐다: %v", err)
	}
}
