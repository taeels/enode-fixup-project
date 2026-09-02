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

// Prepare 는 ①사출의 앞부분이다 — 작업공간을 그 리비전으로 세운다.
//
// 순서: sanitize → 리비전 확인 → (그다음 호출자가 $IN 을 깐다) → 기동
//
// 저장소를 받지 않는다 — 노드가 이미 갖고 있고 그것이 매칭 조건이었다
// (ADR-017: 저장소는 GB 라 10 MiB blob 을 못 지나간다).
func (w *Worker) Prepare(ctx context.Context, spec *WorkspaceSpec, log *slog.Logger) (Prep, error) {
	if spec == nil {
		return PrepNone, nil // 워크스페이스를 안 쓰는 단계
	}
	if spec.Repo == "" {
		// 준비하지 않은 워크스페이스에서 돈다 (ADR-036).
		//
		// 저장소가 없으면 되돌릴 수단이 없다 — git 워크스페이스는 reset·clean 으로
		// 「알려진 상태」가 되지만, 풀어놓은 소스 트리에는 그런 것이 없다.
		// 워크스페이스는 고정 경로를 재사용 하므로 이전 단계가 남긴 것이 그대로 남는다.
		//
		// 조용히 넘기지 않고 사실로 남긴다 — 안 남기면 나중에 "왜 결과가 다르지" 의
		// 원인을 못 찾는다. 봉인이 자기충족이려면 이것이 거기 있어야 한다 (ADR-005 성질 4).
		if w.Local.Workspace == "" {
			return PrepNone, nil
		}
		log.Warn("workspace not prepared: no repository, so nothing was reset",
			"dir", w.Local.Workspace)
		return PrepUnprepared, nil
	}
	dir := w.Local.Workspace
	if dir == "" {
		return PrepNone, fmt.Errorf("the contract requires a workspace but this node has none")
	}

	// 매칭이 이미 걸렀지만 확인한다 — 다른 저장소를 빌드하면 조용히 틀린 결과가 나온다.
	if got := DetectRepo(dir); got != spec.Repo {
		return PrepNone, fmt.Errorf("workspace repository mismatch: node has %q, contract wants %q", got, spec.Repo)
	}

	// 순서가 셋이고 뒤바꾸면 안 된다
	//
	//	① reset --hard   추적 파일의 변경을 버린다 — 안 하면 checkout 이 거절된다
	//	② checkout       목표 리비전으로 옮긴다
	//	③ clean -df      목표 리비전의 .gitignore 로 청소한다
	//
	// ③ 을 ② 앞에 두면 이전 리비전의 무시 규칙으로 청소 하게 되고,
	// 그 리비전에 .gitignore 가 없거나 다르면 데워둔 빌드 캐시가 날아간다.
	// 실측에서 밟았다 — ADR-017 이 -x 를 뺀 이유가 순서에도 걸려 있었다.
	start := time.Now()
	if err := w.reset(ctx, dir); err != nil {
		return PrepNone, err
	}
	if spec.Rev != "" {
		if err := w.checkout(ctx, dir, spec.Rev, log); err != nil {
			return PrepNone, err
		}
	}
	if err := w.clean(ctx, dir); err != nil {
		return PrepNone, err
	}
	log.Info("workspace prepared", "repo", spec.Repo, "rev", spec.Rev,
		"took", time.Since(start).Round(time.Millisecond))
	return PrepClean, nil
}

// Prep 은 워크스페이스가 어떤 상태에서 이 단계가 돌았는가다 (ADR-036).
//
// 부재로 두지 않는다 — ADR-020 이 "가설 없음" 을 status:none 이라는 값으로
// 만든 것과 같은 이유다. 준비 안 함과 준비 실패와 워크스페이스 없음이
// 봉인에서 구분되지 않으면 재현 실패의 원인을 못 찾는다.
type Prep string

const (
	PrepNone       Prep = ""           // 워크스페이스를 안 쓰는 단계
	PrepClean      Prep = "clean"      // reset · checkout · clean 을 마쳤다
	PrepUnprepared Prep = "unprepared" // 되돌리지 않았다 — 저장소가 없다
)

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

// clean 은 알려진 상태를 만든다. 깨끗함을 만드는 것이 아니다 (ADR-017 결정 4).
//
//	✗ git clean -xdf   무시되는 파일까지 지운다 = 데워둔 빌드 캐시가 날아간다
//	                   ADR-007 이 준비물로 잡았고 INVARIANTS §3.1 이 "증분 빌드면
//	                   몇 분" 을 전제했는데 둘 다 무너진다
//	○ git clean -df    추적 안 되는 것만. 무시되는 산출물은 남는다.
func (w *Worker) clean(ctx context.Context, dir string) error {
	if isRepo(dir) {
		return run(ctx, dir, "repo", "forall", "-c", "git clean -df")
	}
	if err := run(ctx, dir, "git", "clean", "-df"); err != nil { // -x 없이
		return fmt.Errorf("sanitize(clean): %w", err)
	}
	return nil
}

// repo sync 는 sanitize 가 아니다 — 리비전을 바꾸는 것이고 시간이 다른 자릿수다.
func isRepo(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".repo"))
	return err == nil
}

// checkout 은 그 리비전으로 세운다. 로컬에 없으면 받아온다.
//
// ADR-017 이 [미정] 으로 남긴 자리다 — 새 패치는 로컬에 없는 것이 정상이므로
// 실패시키면 본편 시나리오가 아예 안 돈다. 대신 시간이 드는데,
// 임대가 그 시간을 묶는다 (not_after).
func (w *Worker) checkout(ctx context.Context, dir, rev string, log *slog.Logger) error {
	if err := run(ctx, dir, "git", "rev-parse", "--verify", rev+"^{commit}"); err != nil {
		log.Info("revision not present locally; fetching", "rev", rev)
		if err := run(ctx, dir, "git", "fetch", "--quiet", "origin", rev); err != nil {
			// 브랜치명이나 태그일 수도 있다. 전체 fetch 로 한 번 더.
			if err2 := run(ctx, dir, "git", "fetch", "--quiet", "--all"); err2 != nil {
				return fmt.Errorf("cannot fetch revision %s: %w", rev, err)
			}
		}
	}
	if err := run(ctx, dir, "git", "checkout", "--quiet", "--detach", rev); err != nil {
		return fmt.Errorf("checkout %s: %w", rev, err)
	}
	return nil
}

func run(ctx context.Context, dir string, name string, args ...string) error {
	cmd := noConsole(exec.CommandContext(ctx, name, args...))
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
