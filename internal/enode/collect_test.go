package enode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mk(t *testing.T, dir, rel, body string) string {
	t.Helper()
	write(t, dir, rel, body)
	return filepath.Join(dir, rel)
}

// 평범한 경우 — 계약이 경로를 적으면 $OUT 으로 옮긴다.
func TestCollect_MovesTheDeclaredPaths(t *testing.T) {
	ws, out := t.TempDir(), t.TempDir()
	mk(t, ws, "drivers/spi/spi-bcm2835.ko", "module")
	mk(t, ws, "build.log", "buildlog")

	got, notes := collectDeclared(ws, out, map[string]string{
		"artifact":  "drivers/spi/spi-bcm2835.ko",
		"build_log": "build.log",
	})
	if len(notes) != 0 {
		t.Fatalf("a note was attached: %+v", notes)
	}
	if strings.Join(got, ",") != "artifact,build_log" {
		t.Fatalf("harvested: %v", got)
	}
	b, err := os.ReadFile(filepath.Join(out, "artifact"))
	if err != nil || string(b) != "module" {
		t.Fatalf("content does not match: %q %v", b, err)
	}
}

// 워크스페이스 밖을 못 가리킨다
//
// (ADR-042 이전 표현: acceptEdits + --add-dir) 모델이 쓸 수 있는 곳과 collect 가
// 아무 데나 가리키면 계약이 그 경계를 우회한다. 계약은 노드 주인이 아닌
// 사람이 낸다.
func TestCollect_CannotPointOutside(t *testing.T) {
	ws, out := t.TempDir(), t.TempDir()
	secret := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(secret, []byte("secret-teal"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, pat := range []string{
		secret,               // 절대경로
		"../secret.txt",      // .. escape
		"../../etc/passwd",   // 더 깊은 탈출
		"a/../../secret.txt", // escape mixed into the middle
	} {
		got, notes := collectDeclared(ws, out, map[string]string{"x": pat})
		if len(got) != 0 {
			t.Fatalf("harvested %q", pat)
		}
		if len(notes) != 1 {
			t.Fatalf("%q — no note was left: %+v", pat, notes)
		}
	}
	if n := len(harvest(out)); n != 0 {
		t.Fatalf("something got into $OUT: %v", harvest(out))
	}
}

// 심링크를 안 따라간다 — 글롭은 워크스페이스 안이어도 가리키는 곳은 밖일 수 있다.
func TestCollect_DoesNotFollowASymlink(t *testing.T) {
	ws, out := t.TempDir(), t.TempDir()
	secret := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(secret, []byte("secret-teal"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(ws, "bait")); err != nil {
		t.Skip("cannot create a symlink")
	}

	got, notes := collectDeclared(ws, out, map[string]string{"x": "bait"})
	if len(got) != 0 {
		b, _ := os.ReadFile(filepath.Join(out, "x"))
		t.Fatalf("followed a symlink and harvested outside: %q", b)
	}
	if len(notes) != 1 || !strings.Contains(notes[0].Why, "no file matches") {
		t.Fatalf("notes: %+v", notes)
	}
}

// 부모가 심링크여도 안 걷는다
//
// 최종 항목만 Lstat 으로 보면 통과해 버린다 — 심링크는 부모 쪽에 있기 때문이다.
// git 이 심링크를 담을 수 있으므로 리뷰 대상 코드가 스스로 통로를 놓을 수 있다.
func TestCollect_DoesNotFollowASymlinkedParent(t *testing.T) {
	ws, out := t.TempDir(), t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "shadow"), []byte("secret-teal"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(ws, "x")); err != nil {
		t.Skip("cannot create a symlink")
	}

	got, notes := collectDeclared(ws, out, map[string]string{"leak": "x/shadow"})
	if len(got) != 0 {
		b, _ := os.ReadFile(filepath.Join(out, "leak"))
		t.Fatalf("passed through a symlinked parent and harvested outside: %q", b)
	}
	if len(notes) != 1 || !strings.Contains(notes[0].Why, "no file matches") {
		t.Fatalf("notes: %+v", notes)
	}
}

// 워크스페이스 자신이 심링크 아래 있어도 걷는다 — 음성 대조.
// 양쪽을 다 풀지 않으면 실경로 비교가 정상 산출물까지 떨어뜨린다.
func TestCollect_HarvestsEvenIfTheWorkspaceIsASymlink(t *testing.T) {
	actual, out := t.TempDir(), t.TempDir()
	mk(t, actual, "a.ko", "module")
	link := filepath.Join(t.TempDir(), "ws")
	if err := os.Symlink(actual, link); err != nil {
		t.Skip("cannot create a symlink")
	}

	got, notes := collectDeclared(link, out, map[string]string{"m": "a.ko"})
	if len(got) != 1 || len(notes) != 0 {
		t.Fatalf("dropped a valid artifact: got=%v notes=%+v", got, notes)
	}
	b, err := os.ReadFile(filepath.Join(out, "m"))
	if err != nil || string(b) != "module" {
		t.Fatalf("content does not match: %q %v", b, err)
	}
}

// 여럿이 맞으면 안 걷고 목록을 남긴다 — 하나의 blob 이름에 여럿을 넣으면
// 소비자가 예측을 못 한다 (어떨 땐 .ko, 어떨 땐 묶음).
func TestCollect_SeveralMatchesHarvestNothingAndSayWhy(t *testing.T) {
	ws, out := t.TempDir(), t.TempDir()
	mk(t, ws, "m/a.ko", "1")
	mk(t, ws, "m/b.ko", "2")

	got, notes := collectDeclared(ws, out, map[string]string{"modules": "m/*.ko"})
	if len(got) != 0 {
		t.Fatalf("must not harvest: %v", got)
	}
	if len(notes) != 1 || !strings.Contains(notes[0].Why, "matches 2 files") {
		t.Fatalf("the note is too thin: %+v", notes)
	}
	if !strings.Contains(notes[0].Why, "a.ko") || !strings.Contains(notes[0].Why, "b.ko") {
		t.Fatalf("it does not say what matched: %s", notes[0].Why)
	}
}

// 하나만 맞는 글롭은 걷는다.
func TestCollect_ASingleGlobMatchIsHarvested(t *testing.T) {
	ws, out := t.TempDir(), t.TempDir()
	mk(t, ws, "arch/arm/boot/zImage", "kernel")
	got, notes := collectDeclared(ws, out, map[string]string{"kernel": "arch/*/boot/zImage"})
	if len(got) != 1 || len(notes) != 0 {
		t.Fatalf("got=%v notes=%+v", got, notes)
	}
}

// 스크립트가 직접 낸 것이 우선 — collect 는 보조다.
func TestCollect_DoesNotOverwriteWhatExists(t *testing.T) {
	ws, out := t.TempDir(), t.TempDir()
	mk(t, ws, "a.bin", "from-workspace")
	if err := os.WriteFile(filepath.Join(out, "artifact"), []byte("from-script"), 0o644); err != nil {
		t.Fatal(err)
	}
	collectDeclared(ws, out, map[string]string{"artifact": "a.bin"})
	b, _ := os.ReadFile(filepath.Join(out, "artifact"))
	if string(b) != "from-script" {
		t.Fatalf("overwritten: %q", b)
	}
}

// 없으면 왜 없는지를 남긴다 — 새 실패 경로는 안 만든다.
func TestCollect_LeavesANoteWhenNothingMatches(t *testing.T) {
	ws, out := t.TempDir(), t.TempDir()
	got, notes := collectDeclared(ws, out, map[string]string{"artifact": "missing/path.ko"})
	if len(got) != 0 || len(notes) != 1 {
		t.Fatalf("got=%v notes=%+v", got, notes)
	}
	if !strings.Contains(notes[0].Why, "missing/path.ko") {
		t.Fatalf("it does not say which path was missing: %s", notes[0].Why)
	}
}

// 디렉터리는 안 걷는다.
func TestCollect_DoesNotHarvestDirectories(t *testing.T) {
	ws, out := t.TempDir(), t.TempDir()
	if err := os.MkdirAll(filepath.Join(ws, "build"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, notes := collectDeclared(ws, out, map[string]string{"x": "build"})
	if len(got) != 0 || len(notes) != 1 {
		t.Fatalf("got=%v notes=%+v", got, notes)
	}
}

// 못 걷은 이유가 기록에 실린다 — 없으면 사람이 계약과 트리를 대조해야 한다.
func TestCollect_TheNoteReachesTheRecord(t *testing.T) {
	out := t.TempDir()
	writeChangedNote(out, []string{"artifact"}, Stamp{},
		[]collectNote{{"artifact", `no file matches "arch/arm/boot/zImage"`}}, testLog())

	b, err := os.ReadFile(filepath.Join(out, changedName))
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if !strings.Contains(got, "collect could not gather") || !strings.Contains(got, "zImage") {
		t.Fatalf("the note did not ride along:\n%s", got)
	}
}

// $IN 은 읽기 전용이다
//
// $IN 은 읽기만 필요한데 쓰기가 열려 있으므로, 훅이 못 보는 쓰기가
// 생긴다. 시연에 대입하면 ④의 리뷰 대상 diff · ⑥의 되먹인 빌드 로그다.
func TestInputLock_CannotBeEdited(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores permissions")
	}
	in := t.TempDir()
	if err := os.WriteFile(filepath.Join(in, "diff"), []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	if why := sealInput(in); why != "" {
		t.Fatalf("could not lock: %s", why)
	}
	t.Cleanup(func() { _ = os.Chmod(in, 0o700) })

	// ① 내용을 못 바꾼다
	if err := os.WriteFile(filepath.Join(in, "diff"), []byte("forged"), 0o644); err == nil {
		t.Fatal("the input was rewritten")
	}
	// ② 새 파일을 못 만든다
	if err := os.WriteFile(filepath.Join(in, "newfile"), []byte("x"), 0o644); err == nil {
		t.Fatal("a new file was created in $IN")
	}
	// ③ 지우고 다시 만들기도 막힌다 — 파일만 잠그면 이 길이 열린다
	if err := os.Remove(filepath.Join(in, "diff")); err == nil {
		t.Fatal("the input was deleted")
	}
	// ④ 읽기는 된다 — 잠금이 단계를 깨면 안 된다
	b, err := os.ReadFile(filepath.Join(in, "diff"))
	if err != nil || string(b) != "original" {
		t.Fatalf("reading broke: %q %v", b, err)
	}
}
