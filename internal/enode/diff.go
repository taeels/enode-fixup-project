package enode

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// R5② — 워크스페이스 변경을 diff 로 걷는다
//
// `ADR-017` 결정 6 이 *"에이전트 산출물은 트리가 아니라 diff 로 나간다"* 고
// 정해놨는데 구현이 그걸 안 하고 있었다. 에이전트가 워크스페이스를 고치면
// 아무것도 안 걷혔다. 이건 안전망이 아니라 빠진 정규 경로다.
//
// fs watch 가 아니라 git 인 이유
//
//	✗ 파일시스템 watch   빌드 산출물이 쏟아져 신호 대 잡음이 나쁘고,
//	                     .gitignore 를 다시 구현하게 된다
//	○ git               무시 규칙을 이미 지킨다 — 우리가 clean 에서 -x 를
//	                     빼고 지켜온 그 규칙 그대로다 (ADR-017 결정 4)
//
// 그리고 `Prepare` 의 sanitize 가 여기서 두 번째 값을 한다 —
// 시작점이 알려진 상태라야 끝점의 차분이 의미가 있다.

// diffNote 는 상한을 넘었을 때 요약 앞에 붙는 머리말이다.
const diffNote = "# workspace diff exceeded the size limit and was replaced by a summary\n" +
	"# the full diff is not in this record; below is git diff --stat\n" +
	"# original size: %d bytes / limit: %d bytes\n#\n"

// workspaceDiff 는 단계가 워크스페이스에 남긴 변경을 diff 로 만든다.
//
// 진짜 인덱스를 안 건드린다 — 임시 인덱스에 read-tree 하고 거기에
// `add -N` 한다. 진짜 인덱스를 쓰면 에이전트가 일부러 stage 해둔 것을
// 우리가 지우게 되고, 뒤따르는 명령 단계의 `git commit` 이 조용히 달라진다.
//
// --binary 를 준다 — 없으면 추적 바이너리가 "Binary files … differ" 한 줄로
// 줄어 내용이 봉인된 기록에서 사라진다. 실측에서 확인했다. diff 라고
// 이름 붙여놓고 적용 불가능한 것을 남기면 그건 기록이 아니다.
//
//	add -N (intent-to-add)  추적 안 된 새 파일도 diff 에 실린다.
//	                        .gitignore 는 그대로 지켜진다.
//	git diff                추적 파일의 수정·삭제 + 위의 새 파일
//
// 상한을 넘으면 잘라서 주지 않는다 — 잘린 산출물은 산출물이 아니다
// (Mediator 의 putBlob 이 413 으로 같은 말을 한다). 대신 완전한 요약인
// --stat 으로 갈아끼운다. 성질 4(자기충족)를 지키면서 거짓말을 안 하는 방법이다.
func workspaceDiff(ctx context.Context, dir string, limit int64) ([]byte, error) {
	if dir == "" {
		return nil, nil
	}
	if isRepo(dir) {
		return repoDiff(ctx, dir, limit)
	}

	idx, err := os.CreateTemp("", "enode-index-*")
	if err != nil {
		return nil, err
	}
	idx.Close()                 //nolint:errcheck
	defer os.Remove(idx.Name()) //nolint:errcheck
	// 워크스페이스 밖에 둔다 — 안에 두면 자기가 추적 안 된 파일로 잡힌다.
	env := []string{"GIT_INDEX_FILE=" + idx.Name()}

	if _, err := gitOut(ctx, dir, env, "read-tree", "HEAD"); err != nil {
		return nil, fmt.Errorf("cannot prepare temporary index: %w", err)
	}
	// add -N . 은 순수 intent-to-add 가 아니다 — 실측에서 알았다.
	// `.` 를 주면 삭제까지 인덱스에 반영 하므로, 그 뒤의 `git diff`
	// (작업트리 vs 인덱스) 로는 삭제가 안 보인다. 그래서 아래는 `diff HEAD` 다.
	if _, err := gitOut(ctx, dir, env, "add", "-N", "."); err != nil {
		return nil, fmt.Errorf("intent-to-add: %w", err)
	}
	d, err := gitOut(ctx, dir, env, "diff", "--binary", "HEAD")
	if err != nil {
		return nil, err
	}
	if len(d) == 0 {
		return nil, nil // 아무것도 안 바뀌었으면 산출물도 없다
	}
	if int64(len(d)) <= limit {
		return d, nil
	}
	stat, err := gitOut(ctx, dir, env, "diff", "HEAD", "--stat")
	if err != nil {
		return nil, err
	}
	head := fmt.Sprintf(diffNote, len(d), limit)
	return append([]byte(head), stat...), nil
}

// repoDiffScript · repoStatScript 는 .repo 트리의 프로젝트마다 도는 글이다 — 프로젝트마다 자기 임시 인덱스를
// 쓴다 (진짜 인덱스를 안 건드리는 이유는 단일 git 쪽과 같다). 호스트의 repoDiff 와 세션 안의 sessionDiffScript 가
// 같은 글자를 쓰도록 패키지 상수로 둔다 (bake 유닛 · 계획 4.1 34번).
const (
	repoDiffScript = `i=$(mktemp); export GIT_INDEX_FILE="$i"; ` +
		`git read-tree HEAD 2>/dev/null && git add -N . >/dev/null 2>&1; ` +
		`d=$(git -c core.quotePath=false diff --binary HEAD); rm -f "$i"; ` +
		`if [ -n "$d" ]; then echo "### $REPO_PATH"; echo "$d"; fi`
	// 요약도 프로젝트별로 모은다.
	repoStatScript = `i=$(mktemp); export GIT_INDEX_FILE="$i"; ` +
		`git read-tree HEAD 2>/dev/null && git add -N . >/dev/null 2>&1; ` +
		`s=$(git -c core.quotePath=false diff HEAD --stat); rm -f "$i"; ` +
		`if [ -n "$s" ]; then echo "### $REPO_PATH"; echo "$s"; fi`
)

// repoDiff 는 .repo 트리다 — 프로젝트마다 돌며 앞에 경로를 붙인다.
//
// 실물 manifest 로는 아직 못 돌려봤다. 형태는 위와 같다.
func repoDiff(ctx context.Context, dir string, limit int64) ([]byte, error) {
	out, err := gitOutName(ctx, dir, nil, "repo", "forall", "-c", repoDiffScript)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 || int64(len(out)) <= limit {
		return out, nil
	}
	stat, err := gitOutName(ctx, dir, nil, "repo", "forall", "-c", repoStatScript)
	if err != nil {
		return nil, err
	}
	return append([]byte(fmt.Sprintf(diffNote, len(out), limit)), stat...), nil
}

func gitOut(ctx context.Context, dir string, env []string, args ...string) ([]byte, error) {
	return gitOutName(ctx, dir, env, "git", args...)
}

// gitOutName 은 git(또는 repo)을 돌리고 stdout 을 돌려준다.
//
// core.quotePath=false 를 여기서 붙인다 — 호출부마다 붙이면 새 git 호출을
// 더할 때 다시 밟는다. 실제로 밟았다: diff 에서 고쳐놓고 `git status` 를 쓰는
// 훅에서 똑같이 깨졌다. 한 곳에서 지킨다 — exec 을 runner 하나로 모은 것과
// 같은 원칙이다.
//
// git 은 이 설정이 기본 true 라 ASCII 밖 경로를 "\354\247\204…" 로 찍는다.
// 우리 사용자는 한국어 커널 개발자다 — 사람이 못 읽는 기록은
// ADR-005 성질 4(자기충족)를 못 지킨다.
func gitOutName(ctx context.Context, dir string, env []string, name string, args ...string) ([]byte, error) {
	if name == "git" {
		args = append([]string{"-c", "core.quotePath=false"}, args...)
	}
	cmd := child(exec.CommandContext(ctx, name, args...))
	cmd.Dir = dir
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if len(msg) > 300 {
			msg = msg[:300] + "…"
		}
		return nil, fmt.Errorf("%s %s: %s", name, strings.Join(args, " "), msg)
	}
	return stdout.Bytes(), nil
}

// diffName 은 워크스페이스 diff 가 실리는 산출물 이름이다.
//
// 계약이 선언하지 않아도 나온다 — 그게 요점이다. 모델이 선언을 빠뜨려도
// 무엇이 바뀌었는지는 기록에 남는다. 이름이 고정이라 계약이 참조할 수도 있다.
const diffName = "workspace.diff"

// maxBlobBytes 는 Mediator 가 권위다 (ADR-015 §3, config 의 max_blob_bytes).
// 여기 값은 「전체를 보낼지 요약으로 갈지」를 미리 정하는 데만 쓴다.
// 어긋나면 Mediator 가 413 으로 막으므로 틀려도 안전한 쪽으로 틀린다.
const maxBlobBytes = 10 << 20

// 격리 노드의 diff — 세션 안에서 만든다 (bake 유닛 · 계획 4.1 34번 · CG 물음 5 답)
//
// runc-overlay 노드의 Finalize 는 workspace.diff 를 준비된 rootfs 의 git · repo 로 세션 안에서 만든다. 호스트의
// runtime helper (컨테이너 밖 · 노드 uid) 가 merged view 에서 git 을 돌리면 그 단계가 쓴 .git/config 와 굽기 뒤
// lower 에 남은 계약의 설정 (core.fsmonitor · filter 의 clean · .repo/repo 의 런처) 을 호스트가 실행한다.
// native 노드는 오늘처럼 workspaceDiff 의 호스트 git 이다 — 계약의 명령이 이미 호스트에서 돈다.

// sessionDiffScript 는 세션 안에서 도는 고정 글자다 — 계약은 못 바꾼다. 차례와 글자는 workspaceDiff 그대로다
// (임시 인덱스 · read-tree HEAD · add -N . · diff --binary · 요약은 diff HEAD --stat · 모든 git 에 core.quotePath=false).
// repo 모양은 repoDiffScript · repoStatScript 를 환경 변수로 받아 repo forall -c 에 넘긴다. 출력은 stdout 흐름으로만
// 호스트에 온다 — 호스트가 컨테이너가 쓴 파일을 읽지 않는다.
//
//	exit 125    준비된 rootfs 에 git 이 없다 (repo 모양이어도 먼저)
//	exit 124    repo 모양인데 repo 가 없다
//	exit 121    임시 인덱스를 못 만들었다 · read-tree HEAD 가 실패했다
//	exit 122    add -N . 이 실패했다
const sessionDiffScript = `export LC_ALL=C GIT_TERMINAL_PROMPT=0
GIT_CEILING_DIRECTORIES=$(cd .. && pwd); export GIT_CEILING_DIRECTORIES
command -v git >/dev/null 2>&1 || exit 125
if [ -e .repo ]; then
  command -v repo >/dev/null 2>&1 || exit 124
  if [ "$ENODE_DIFF_MODE" = stat ]; then exec repo forall -c "$ENODE_REPO_STAT_SCRIPT"; fi
  exec repo forall -c "$ENODE_REPO_DIFF_SCRIPT"
fi
i=$(mktemp) || exit 121
trap 'rm -f "$i"' EXIT
GIT_INDEX_FILE=$i; export GIT_INDEX_FILE
git -c core.quotePath=false read-tree HEAD >/dev/null || exit 121
git -c core.quotePath=false add -N . >/dev/null || exit 122
if [ "$ENODE_DIFF_MODE" = stat ]; then git -c core.quotePath=false diff HEAD --stat
else git -c core.quotePath=false diff --binary HEAD; fi
`

// sessionDiffEnv 는 세션 안의 diff 가 받는 환경이다 — 호스트 환경 변수를 넘기지 않는다.
func sessionDiffEnv(mode string) []string {
	return []string{"ENODE_DIFF_MODE=" + mode, "ENODE_REPO_DIFF_SCRIPT=" + repoDiffScript,
		"ENODE_REPO_STAT_SCRIPT=" + repoStatScript}
}

// diffLimit 은 세션 안 diff 의 상한이다 — 넘으면 요약으로 갈아끼운다. 제품은 maxBlobBytes 이고 시험이 줄인다.
var diffLimit int64 = maxBlobBytes

// sessionDiffError 는 세션 안 diff 의 exit 와 stderr 의 마지막 줄을 DiffError 글자로 옮긴다. 진단이다 —
// 단계는 이것으로 실패하지 않는다 (finalize.go 의 logTail 이 단계 로그 끝에 적는다).
func sessionDiffError(code int, last string) string {
	switch code {
	case 125:
		return "the prepared rootfs has no git"
	case 124:
		return "the prepared rootfs has no repo"
	case 121:
		return joinCause("cannot prepare temporary index", last)
	case 122:
		return joinCause("intent-to-add", last)
	}
	return joinCause(fmt.Sprintf("workspace diff exited %d", code), last)
}

// placeWorkspaceDiff 는 세션이 stdout 으로 보낸 diff 를 호스트의 $OUT 에 놓는다 — 새 임시 파일 (O_EXCL) 에 쓰고
// fsync 뒤 workspace.diff 로 rename 한다. 단계가 그 이름에 미리 둔 symlink · FIFO 를 따라가지 않는다 (rename 은
// 그 이름을 갈아끼운다). 그 이름이 디렉터리면 rename 이 실패한다. 임시 파일은 .enode- 로 시작해 올라가지 않는다.
func placeWorkspaceDiff(out string, b []byte) error {
	fail := func(err error) error { return fmt.Errorf("cannot place %s: %w", diffName, err) }
	f, err := os.CreateTemp(out, ".enode-diff-*")
	if err != nil {
		return fail(err)
	}
	tmp := f.Name()
	_, err = f.Write(b)
	if err == nil {
		err = f.Chmod(0o644)
	}
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp, filepath.Join(out, diffName))
	}
	if err != nil {
		_ = os.Remove(tmp)
		return fail(err)
	}
	return nil
}

// sessionDiff 는 sessionDiffScript 를 돈 결과를 workspace.diff 의 바이트로 만든다 — run 이 그 글을 한 모드로 돌린다
// (세션이면 세션의 Run, 시험이면 호스트 sh). 전체가 diffLimit 을 넘으면 stat 모드로 한 번 더 돌아 diffNote 머리와
// 요약을 잇는다 — 오늘 workspaceDiff 와 같은 바이트다. 실패는 DiffError 의 글자로 돌려준다. 빈 diff 는 nil 이다.
func sessionDiff(run func(mode string, stdout, stderr io.Writer) (int, error)) ([]byte, string) {
	once := func(mode string) (*diffBuffer, string) {
		// 요약은 호스트처럼 자르지 않는다 — 상한은 Mediator 의 blob 상한이다
		limit := diffLimit
		if mode == "stat" {
			limit = maxBlobBytes
		}
		stdout, stderr := &diffBuffer{limit: limit}, &tailBuffer{max: 4 << 10}
		code, err := run(mode, stdout, stderr)
		switch {
		case code < 0:
			return nil, runtimeRunText(err)
		case code > 0:
			return nil, sessionDiffError(code, lastLine(stderr.String()))
		}
		return stdout, ""
	}
	d, why := once("diff")
	if why != "" || d.total == 0 {
		return nil, why
	}
	if d.total <= diffLimit {
		return d.buf.Bytes(), ""
	}
	stat, why := once("stat")
	if why != "" {
		return nil, why
	}
	return append([]byte(fmt.Sprintf(diffNote, d.total, diffLimit)), stat.buf.Bytes()...), ""
}

// diffBuffer 는 세션 안 diff 의 stdout 이다 — limit 바이트까지 담고 전체 길이는 센다. 넘으면 요약으로 간다.
type diffBuffer struct {
	limit int64
	total int64
	buf   bytes.Buffer
}

func (b *diffBuffer) Write(p []byte) (int, error) {
	b.total += int64(len(p))
	if room := b.limit - int64(b.buf.Len()); room > 0 {
		b.buf.Write(p[:min(int64(len(p)), room)])
	}
	return len(p), nil
}

// tailBuffer 는 stderr 의 뒤쪽 max 바이트만 남긴다 — 문장에는 마지막 줄만 싣는다.
type tailBuffer struct {
	max int
	b   []byte
}

func (t *tailBuffer) Write(p []byte) (int, error) {
	t.b = append(t.b, p...)
	if len(t.b) > t.max {
		t.b = append([]byte(nil), t.b[len(t.b)-t.max:]...)
	}
	return len(p), nil
}

func (t *tailBuffer) String() string { return string(t.b) }

// writeWorkspaceDiff 는 diff 를 $OUT 에 놓는다.
//
// 새 전송 경로를 안 만든다 — $OUT 에 놓으면 기존 ④수확이 그대로 걷어
// 올린다. 상한 초과는 Mediator 가 413 으로 막고 그건 이미 있는 규칙이다.
func writeWorkspaceDiff(ctx context.Context, dir, out string, limit int64) (int, error) {
	d, err := workspaceDiff(ctx, dir, limit)
	if err != nil || len(d) == 0 {
		return 0, err
	}
	return len(d), os.WriteFile(filepath.Join(out, diffName), d, 0o644)
}
