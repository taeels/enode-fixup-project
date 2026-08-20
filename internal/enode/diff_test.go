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
	git(t, dir, "commit", "--quiet", "-m", "초기")
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

// 아무것도 안 바뀌었으면 ★ 산출물도 없다 ★ — 빈 diff 를 올리지 않는다.
func TestDiff_변화가_없으면_없다(t *testing.T) {
	d, err := workspaceDiff(context.Background(), gitInit(t), maxBlobBytes)
	if err != nil || len(d) != 0 {
		t.Fatalf("빈 diff 가 아니다: %d바이트 %v", len(d), err)
	}
}

// 추적 파일의 수정이 잡힌다.
func TestDiff_수정을_잡는다(t *testing.T) {
	dir := gitInit(t)
	write(t, dir, "keep.c", "int x = 2;\n")
	d, err := workspaceDiff(context.Background(), dir, maxBlobBytes)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(d), "-int x = 1;") || !strings.Contains(string(d), "+int x = 2;") {
		t.Fatalf("수정이 안 잡혔다:\n%s", d)
	}
}

// ★ 추적 안 된 새 파일도 잡힌다 ★ — 에이전트가 만든 것이 보통 이쪽이다.
func TestDiff_새_파일을_잡는다(t *testing.T) {
	dir := gitInit(t)
	write(t, dir, "new.c", "int y;\n")
	d, err := workspaceDiff(context.Background(), dir, maxBlobBytes)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(d), "new.c") || !strings.Contains(string(d), "+int y;") {
		t.Fatalf("새 파일이 안 잡혔다:\n%s", d)
	}
}

// 삭제도 잡힌다.
func TestDiff_삭제를_잡는다(t *testing.T) {
	dir := gitInit(t)
	if err := os.Remove(filepath.Join(dir, "keep.c")); err != nil {
		t.Fatal(err)
	}
	d, _ := workspaceDiff(context.Background(), dir, maxBlobBytes)
	if !strings.Contains(string(d), "-int x = 1;") {
		t.Fatalf("삭제가 안 잡혔다:\n%s", d)
	}
}

// ★ 무시되는 것은 안 걷는다 ★ — 데워둔 빌드 캐시가 diff 에 섞이면 못 쓴다.
// clean 에서 -x 를 뺀 것과 ★ 같은 규칙 ★ 이다 (ADR-017 결정 4).
func TestDiff_무시되는_것은_안_걷는다(t *testing.T) {
	dir := gitInit(t)
	write(t, dir, "build/큰것.bin", strings.Repeat("x", 5000))
	write(t, dir, "a.o", "오브젝트")
	write(t, dir, "진짜.c", "int z;\n")

	d, err := workspaceDiff(context.Background(), dir, maxBlobBytes)
	if err != nil {
		t.Fatal(err)
	}
	s := string(d)
	for _, bad := range []string{"build/", "a.o"} {
		if strings.Contains(s, bad) {
			t.Fatalf("★ 무시되는 것이 diff 에 들어왔다: %s ★\n%s", bad, s)
		}
	}
	if !strings.Contains(s, "진짜.c") {
		t.Fatalf("진짜 변경이 빠졌다:\n%s", s)
	}
}

// ★ 진짜 인덱스를 안 건드린다 ★ — 에이전트가 stage 해둔 것을 지우면
// 뒤따르는 명령 단계의 git 동작이 조용히 달라진다.
func TestDiff_인덱스를_안_건드린다(t *testing.T) {
	dir := gitInit(t)
	write(t, dir, "staged.c", "int s;\n")
	git(t, dir, "add", "staged.c")

	before := status(t, dir)
	if _, err := workspaceDiff(context.Background(), dir, maxBlobBytes); err != nil {
		t.Fatal(err)
	}
	if after := status(t, dir); after != before {
		t.Fatalf("★ 인덱스가 바뀌었다 ★\n전: %q\n후: %q", before, after)
	}
	if !strings.HasPrefix(before, "A ") {
		t.Fatalf("전제가 틀렸다 — staged 상태가 아니다: %q", before)
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

// ★ 상한을 넘으면 잘라서 주지 않는다 ★ — 잘린 산출물은 산출물이 아니다.
// 대신 ★ 완전한 요약 ★ 인 --stat 으로 갈아끼운다.
func TestDiff_상한을_넘으면_요약한다(t *testing.T) {
	dir := gitInit(t)
	write(t, dir, "커진다.c", strings.Repeat("아주 긴 줄이다\n", 500))

	const limit = 1000
	d, err := workspaceDiff(context.Background(), dir, limit)
	if err != nil {
		t.Fatal(err)
	}
	s := string(d)
	if !strings.Contains(s, "상한을 넘어 요약으로 대체했다") {
		t.Fatalf("요약으로 안 바뀌었다:\n%s", s)
	}
	if strings.Contains(s, "+아주 긴 줄이다") {
		t.Fatal("★ 잘린 diff 본문이 섞였다 ★ — 요약은 요약이어야 한다")
	}
	if !strings.Contains(s, "커진다.c") {
		t.Fatalf("요약에 파일 이름이 없다:\n%s", s)
	}
}

// $OUT 에 놓으면 ★ 기존 ④수확이 그대로 걷는다 ★ — 새 전송 경로가 없다.
func TestDiff_OUT에_놓인다(t *testing.T) {
	dir := gitInit(t)
	out := t.TempDir()
	write(t, dir, "keep.c", "int x = 9;\n")

	n, err := writeWorkspaceDiff(context.Background(), dir, out, maxBlobBytes)
	if err != nil || n == 0 {
		t.Fatalf("%d %v", n, err)
	}
	names := harvest(out)
	if len(names) != 1 || names[0] != diffName {
		t.Fatalf("수확이 못 걷는다: %v", names)
	}
}

// ★ 한글 경로가 이스케이프되면 안 된다 ★
//
// git 은 core.quotePath 가 기본 true 라 "\354\247\204\354\247\234.c" 로 찍는다.
// 실측에서 diff 전체가 그렇게 나왔다. ★ 우리 사용자는 한국어 커널 개발자다 ★ —
// 사람이 못 읽는 기록은 ADR-005 성질 4(자기충족)를 못 지킨다.
func TestDiff_한글_경로가_읽힌다(t *testing.T) {
	dir := gitInit(t)
	write(t, dir, "드라이버/스핀들.c", "int 속도;\n")

	d, err := workspaceDiff(context.Background(), dir, maxBlobBytes)
	if err != nil {
		t.Fatal(err)
	}
	s := string(d)
	if strings.Contains(s, `\3`) {
		t.Fatalf("★ 경로가 이스케이프됐다 ★ (core.quotePath):\n%s", s)
	}
	if !strings.Contains(s, "드라이버/스핀들.c") || !strings.Contains(s, "+int 속도;") {
		t.Fatalf("한글이 그대로 안 나온다:\n%s", s)
	}
}

// ★ 삭제는 `git diff` 로 안 보인다 ★ — add -N . 이 삭제를 인덱스에 반영하기 때문.
// 이 시험은 누가 `diff HEAD` 를 `diff` 로 되돌리면 터진다.
func TestDiff_삭제와_추가가_한_diff에(t *testing.T) {
	dir := gitInit(t)
	if err := os.Remove(filepath.Join(dir, "keep.c")); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "새것.c", "int n;\n")

	s := string(mustDiff(t, dir))
	if !strings.Contains(s, "deleted file") {
		t.Fatalf("★ 삭제가 빠졌다 ★ — diff HEAD 가 아니라 diff 를 쓰고 있나:\n%s", s)
	}
	if !strings.Contains(s, "new file") {
		t.Fatalf("추가가 빠졌다:\n%s", s)
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
