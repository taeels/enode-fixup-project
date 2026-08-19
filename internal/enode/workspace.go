package enode

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// WorkspaceSpec 은 계약의 steps[].workspace 다 (ADR-017 결정 5).
type WorkspaceSpec struct {
	Repo string `json:"repo"`
	Rev  string `json:"rev"`
}

// Prepare 는 ①사출의 앞부분이다 — ★ 작업공간을 그 리비전으로 세운다 ★.
//
// 순서: sanitize → 리비전 확인 → (그다음 호출자가 $IN 을 깐다) → 기동
//
// ★ 저장소를 받지 않는다 ★ — 노드가 이미 갖고 있고 그것이 매칭 조건이었다
// (ADR-017: 저장소는 GB 라 10 MiB blob 을 못 지나간다).
func (w *Worker) Prepare(ctx context.Context, spec *WorkspaceSpec, log *slog.Logger) error {
	if spec == nil || spec.Repo == "" {
		return nil // 워크스페이스가 필요 없는 단계
	}
	dir := w.Local.Workspace
	if dir == "" {
		return fmt.Errorf("계약이 워크스페이스를 요구하는데 이 노드에는 없다")
	}

	// 매칭이 이미 걸렀지만 확인한다 — 다른 저장소를 빌드하면 조용히 틀린 결과가 나온다.
	if got := DetectRepo(dir); got != spec.Repo {
		return fmt.Errorf("워크스페이스의 저장소가 다르다: %q 인데 계약은 %q", got, spec.Repo)
	}

	// ★ 순서가 셋이고 뒤바꾸면 안 된다 ★
	//
	//	① reset --hard   추적 파일의 변경을 버린다 — 안 하면 checkout 이 거절된다
	//	② checkout       목표 리비전으로 옮긴다
	//	③ clean -df      ★ 목표 리비전의 .gitignore 로 ★ 청소한다
	//
	// ③ 을 ② 앞에 두면 ★ 이전 리비전의 무시 규칙으로 청소 ★ 하게 되고,
	// 그 리비전에 .gitignore 가 없거나 다르면 ★ 데워둔 빌드 캐시가 날아간다 ★.
	// 실측에서 밟았다 — ADR-017 이 -x 를 뺀 이유가 순서에도 걸려 있었다.
	start := time.Now()
	if err := w.reset(ctx, dir); err != nil {
		return err
	}
	if spec.Rev != "" {
		if err := w.checkout(ctx, dir, spec.Rev, log); err != nil {
			return err
		}
	}
	if err := w.clean(ctx, dir); err != nil {
		return err
	}
	log.Info("워크스페이스 준비", "repo", spec.Repo, "rev", spec.Rev,
		"took", time.Since(start).Round(time.Millisecond))
	return nil
}

// reset 은 추적 파일의 변경을 버린다. checkout 이 거절되지 않게 하는 것이 목적이다.
func (w *Worker) reset(ctx context.Context, dir string) error {
	if isRepo(dir) {
		return run(ctx, dir, "repo", "forall", "-c", "git reset --hard")
	}
	if err := run(ctx, dir, "git", "reset", "--hard"); err != nil {
		return fmt.Errorf("sanitize(reset): %w", err)
	}
	return nil
}

// clean 은 ★ 알려진 상태 ★ 를 만든다. 깨끗함을 만드는 것이 아니다 (ADR-017 결정 4).
//
//	✗ git clean -xdf   ★ 무시되는 파일까지 지운다 = 데워둔 빌드 캐시가 날아간다 ★
//	                   ADR-007 이 준비물로 잡았고 INVARIANTS §3.1 이 "증분 빌드면
//	                   몇 분" 을 전제했는데 둘 다 무너진다
//	○ git clean -df    추적 안 되는 것만. 무시되는 산출물은 남는다.
func (w *Worker) clean(ctx context.Context, dir string) error {
	if isRepo(dir) {
		return run(ctx, dir, "repo", "forall", "-c", "git clean -df")
	}
	if err := run(ctx, dir, "git", "clean", "-df"); err != nil { // ★ -x 없이 ★
		return fmt.Errorf("sanitize(clean): %w", err)
	}
	return nil
}

// ★ repo sync 는 sanitize 가 아니다 ★ — 리비전을 바꾸는 것이고 시간이 다른 자릿수다.
func isRepo(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".repo"))
	return err == nil
}

// checkout 은 그 리비전으로 세운다. 로컬에 없으면 ★ 받아온다 ★.
//
// ADR-017 이 [미정] 으로 남긴 자리다 — 새 패치는 로컬에 없는 것이 정상이므로
// 실패시키면 본편 시나리오가 아예 안 돈다. 대신 시간이 드는데,
// ★ 임대가 그 시간을 묶는다 ★ (not_after).
func (w *Worker) checkout(ctx context.Context, dir, rev string, log *slog.Logger) error {
	if err := run(ctx, dir, "git", "rev-parse", "--verify", rev+"^{commit}"); err != nil {
		log.Info("리비전이 로컬에 없다 — 받아온다", "rev", rev)
		if err := run(ctx, dir, "git", "fetch", "--quiet", "origin", rev); err != nil {
			// 브랜치명이나 태그일 수도 있다. 전체 fetch 로 한 번 더.
			if err2 := run(ctx, dir, "git", "fetch", "--quiet", "--all"); err2 != nil {
				return fmt.Errorf("리비전 %s 를 못 받았다: %w", rev, err)
			}
		}
	}
	if err := run(ctx, dir, "git", "checkout", "--quiet", "--detach", rev); err != nil {
		return fmt.Errorf("checkout %s: %w", rev, err)
	}
	return nil
}

func run(ctx context.Context, dir string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if len(msg) > 400 {
			msg = msg[:400] + "…"
		}
		return fmt.Errorf("%s: %s", name, msg)
	}
	return nil
}

func parseWorkspace(raw json.RawMessage) (*WorkspaceSpec, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var s WorkspaceSpec
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	return &s, nil
}
