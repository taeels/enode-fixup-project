package enode

import (
	"bytes"
	"context"
	"fmt"
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

// repoDiff 는 .repo 트리다 — 프로젝트마다 돌며 앞에 경로를 붙인다.
//
// 실물 manifest 로는 아직 못 돌려봤다. 형태는 위와 같다.
func repoDiff(ctx context.Context, dir string, limit int64) ([]byte, error) {
	// 프로젝트마다 자기 임시 인덱스를 쓴다 — 진짜 인덱스를 안 건드리는
	// 이유는 단일 git 쪽과 같다.
	const script = `i=$(mktemp); export GIT_INDEX_FILE="$i"; ` +
		`git read-tree HEAD 2>/dev/null && git add -N . >/dev/null 2>&1; ` +
		`d=$(git -c core.quotePath=false diff --binary HEAD); rm -f "$i"; ` +
		`if [ -n "$d" ]; then echo "### $REPO_PATH"; echo "$d"; fi`
	out, err := gitOutName(ctx, dir, nil, "repo", "forall", "-c", script)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 || int64(len(out)) <= limit {
		return out, nil
	}
	// 요약도 프로젝트별로 모은다.
	const statScript = `i=$(mktemp); export GIT_INDEX_FILE="$i"; ` +
		`git read-tree HEAD 2>/dev/null && git add -N . >/dev/null 2>&1; ` +
		`s=$(git -c core.quotePath=false diff HEAD --stat); rm -f "$i"; ` +
		`if [ -n "$s" ]; then echo "### $REPO_PATH"; echo "$s"; fi`
	stat, err := gitOutName(ctx, dir, nil, "repo", "forall", "-c", statScript)
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
	cmd := exec.CommandContext(ctx, name, args...)
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
