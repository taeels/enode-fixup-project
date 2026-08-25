package enode

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitInit(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	git(t, dir, "config", "commit.gpgsign", "false")
	write(t, dir, "keep.c", "int x = 1;\n")
	write(t, dir, ".gitignore", "build/\n*.o\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "--quiet", "-m", "initial")
	return dir
}

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// 아무것도 안 바뀌었으면 산출물도 없다 — 빈 diff 를 올리지 않는다.
func TestDiff_NoChangeMeansNoDiff(t *testing.T) {
	d, err := workspaceDiff(context.Background(), gitInit(t), maxBlobBytes)
	if err != nil || len(d) != 0 {
		t.Fatalf("not an empty diff: %d bytes %v", len(d), err)
	}
}

// 추적 파일의 수정이 잡힌다.
func TestDiff_CatchesAnEdit(t *testing.T) {
	dir := gitInit(t)
	write(t, dir, "keep.c", "int x = 2;\n")
	d, err := workspaceDiff(context.Background(), dir, maxBlobBytes)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(d), "-int x = 1;") || !strings.Contains(string(d), "+int x = 2;") {
		t.Fatalf("the edit was not caught:\n%s", d)
	}
}

// 추적 안 된 새 파일도 잡힌다 — 에이전트가 만든 것이 보통 이쪽이다.
func TestDiff_CatchesANewFile(t *testing.T) {
	dir := gitInit(t)
	write(t, dir, "new.c", "int y;\n")
	d, err := workspaceDiff(context.Background(), dir, maxBlobBytes)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(d), "new.c") || !strings.Contains(string(d), "+int y;") {
		t.Fatalf("the new file was not caught:\n%s", d)
	}
}

// 삭제도 잡힌다.
func TestDiff_CatchesADeletion(t *testing.T) {
	dir := gitInit(t)
	if err := os.Remove(filepath.Join(dir, "keep.c")); err != nil {
		t.Fatal(err)
	}
	d, _ := workspaceDiff(context.Background(), dir, maxBlobBytes)
	if !strings.Contains(string(d), "-int x = 1;") {
		t.Fatalf("the deletion was not caught:\n%s", d)
	}
}

// 무시되는 것은 안 걷는다 — 데워둔 빌드 캐시가 diff 에 섞이면 못 쓴다.
// clean 에서 -x 를 뺀 것과 같은 규칙이다 (ADR-017 결정 4).
func TestDiff_DoesNotCollectIgnoredFiles(t *testing.T) {
	dir := gitInit(t)
	write(t, dir, "build/big.bin", strings.Repeat("x", 5000))
	write(t, dir, "a.o", "object")
	write(t, dir, "real.c", "int z;\n")

	d, err := workspaceDiff(context.Background(), dir, maxBlobBytes)
	if err != nil {
		t.Fatal(err)
	}
	s := string(d)
	for _, bad := range []string{"build/", "a.o"} {
		if strings.Contains(s, bad) {
			t.Fatalf("an ignored file got into the diff: %s\n%s", bad, s)
		}
	}
	if !strings.Contains(s, "real.c") {
		t.Fatalf("the real change is missing:\n%s", s)
	}
}

// 진짜 인덱스를 안 건드린다 — 에이전트가 stage 해둔 것을 지우면
// 뒤따르는 명령 단계의 git 동작이 조용히 달라진다.
func TestDiff_LeavesTheIndexAlone(t *testing.T) {
	dir := gitInit(t)
	write(t, dir, "staged.c", "int s;\n")
	git(t, dir, "add", "staged.c")

	before := status(t, dir)
	if _, err := workspaceDiff(context.Background(), dir, maxBlobBytes); err != nil {
		t.Fatal(err)
	}
	if after := status(t, dir); after != before {
		t.Fatalf("the index changed\nbefore: %q\nafter:  %q", before, after)
	}
	if !strings.HasPrefix(before, "A ") {
		t.Fatalf("the premise is wrong — nothing is staged: %q", before)
	}
}

func status(t *testing.T, dir string) string {
	t.Helper()
	c := exec.Command("git", "status", "--porcelain")
	c.Dir = dir
	out, err := c.Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// 상한을 넘으면 잘라서 주지 않는다 — 잘린 산출물은 산출물이 아니다.
// 대신 완전한 요약인 --stat 으로 갈아끼운다.
func TestDiff_SummarizesOverTheLimit(t *testing.T) {
	dir := gitInit(t)
	write(t, dir, "grows.c", strings.Repeat("a very long line\n", 500))

	const limit = 1000
	d, err := workspaceDiff(context.Background(), dir, limit)
	if err != nil {
		t.Fatal(err)
	}
	s := string(d)
	if !strings.Contains(s, "exceeded the size limit") {
		t.Fatalf("it did not turn into a summary:\n%s", s)
	}
	if strings.Contains(s, "+a very long line") {
		t.Fatal("a truncated diff body got mixed in — a summary must be a summary")
	}
	if !strings.Contains(s, "grows.c") {
		t.Fatalf("the summary carries no file name:\n%s", s)
	}
}

// $OUT 에 놓으면 기존 ④수확이 그대로 걷는다 — 새 전송 경로가 없다.
func TestDiff_LandsInOUT(t *testing.T) {
	dir := gitInit(t)
	out := t.TempDir()
	write(t, dir, "keep.c", "int x = 9;\n")

	n, err := writeWorkspaceDiff(context.Background(), dir, out, maxBlobBytes)
	if err != nil || n == 0 {
		t.Fatalf("%d %v", n, err)
	}
	names := harvest(out)
	if len(names) != 1 || names[0] != diffName {
		t.Fatalf("the harvest cannot pick it up: %v", names)
	}
}

// 한글 경로가 이스케이프되면 안 된다
//
// git 은 core.quotePath 가 기본 true 라 "\354\247\204\354\247\234.c" 로 찍는다.
// 실측에서 diff 전체가 그렇게 나왔다. 우리 사용자는 한국어 커널 개발자다 —
// 사람이 못 읽는 기록은 ADR-005 성질 4(자기충족)를 못 지킨다.
func TestDiff_ReadsNonASCIIPaths(t *testing.T) {
	dir := gitInit(t)
	write(t, dir, "드라이버/스핀들.c", "int 속도;\n")

	d, err := workspaceDiff(context.Background(), dir, maxBlobBytes)
	if err != nil {
		t.Fatal(err)
	}
	s := string(d)
	if strings.Contains(s, `\3`) {
		t.Fatalf("the path was escaped (core.quotePath):\n%s", s)
	}
	if !strings.Contains(s, "드라이버/스핀들.c") || !strings.Contains(s, "+int 속도;") {
		t.Fatalf("the non-ASCII text does not come through as-is:\n%s", s)
	}
}

// 삭제는 `git diff` 로 안 보인다 — add -N . 이 삭제를 인덱스에 반영하기 때문.
// 이 시험은 누가 `diff HEAD` 를 `diff` 로 되돌리면 터진다.
func TestDiff_DeletionAndAdditionInOneDiff(t *testing.T) {
	dir := gitInit(t)
	if err := os.Remove(filepath.Join(dir, "keep.c")); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "fresh.c", "int n;\n")

	s := string(mustDiff(t, dir))
	if !strings.Contains(s, "deleted file") {
		t.Fatalf("the deletion is missing — is this diff instead of diff HEAD?\n%s", s)
	}
	if !strings.Contains(s, "new file") {
		t.Fatalf("the addition is missing:\n%s", s)
	}
}

func mustDiff(t *testing.T, dir string) []byte {
	t.Helper()
	d, err := workspaceDiff(context.Background(), dir, maxBlobBytes)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// 추적 바이너리의 내용이 사라지면 안 된다
//
// --binary 없이는 "Binary files a/x and b/x differ" 한 줄로 줄어
// 봉인된 기록에서 내용이 사라진다. diff 라고 이름 붙여놓고
// 적용 불가능한 것을 남기면 그건 기록이 아니다.
func TestDiff_BinaryContentSurvives(t *testing.T) {
	dir := gitInit(t)
	bin := filepath.Join(dir, "fw.bin")
	if err := os.WriteFile(bin, []byte{0, 1, 2, 'f', 'w', 0}, 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", "fw.bin")
	git(t, dir, "commit", "--quiet", "-m", "firmware")
	if err := os.WriteFile(bin, []byte{0, 1, 2, 'F', 'W', '2', 0}, 0o644); err != nil {
		t.Fatal(err)
	}

	s := string(mustDiff(t, dir))
	if strings.Contains(s, "Binary files") {
		t.Fatalf("binary content disappeared — is --binary missing?\n%s", s)
	}
	if !strings.Contains(s, "GIT binary patch") {
		t.Fatalf("not an appliable binary patch:\n%s", s)
	}
}
