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
func TestCollect_적은_경로를_옮긴다(t *testing.T) {
	ws, out := t.TempDir(), t.TempDir()
	mk(t, ws, "drivers/spi/spi-bcm2835.ko", "모듈")
	mk(t, ws, "build.log", "빌드로그")

	got, notes := collectDeclared(ws, out, map[string]string{
		"artifact":  "drivers/spi/spi-bcm2835.ko",
		"build_log": "build.log",
	})
	if len(notes) != 0 {
		t.Fatalf("이유가 붙었다: %+v", notes)
	}
	if strings.Join(got, ",") != "artifact,build_log" {
		t.Fatalf("걷은 것: %v", got)
	}
	b, err := os.ReadFile(filepath.Join(out, "artifact"))
	if err != nil || string(b) != "모듈" {
		t.Fatalf("내용이 안 맞다: %q %v", b, err)
	}
}

// ★ 워크스페이스 밖을 못 가리킨다 ★
//
// acceptEdits + --add-dir 로 모델이 쓸 수 있는 곳을 좁혀놨는데, collect 가
// 아무 데나 가리키면 ★ 계약이 그 경계를 우회한다 ★. 계약은 노드 주인이 아닌
// 사람이 낸다.
func TestCollect_바깥을_못_가리킨다(t *testing.T) {
	ws, out := t.TempDir(), t.TempDir()
	secret := filepath.Join(t.TempDir(), "비밀.txt")
	if err := os.WriteFile(secret, []byte("비밀-청록"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, pat := range []string{
		secret,             // 절대경로
		"../비밀.txt",        // .. 탈출
		"../../etc/passwd", // 더 깊은 탈출
		"a/../../비밀.txt",   // 중간에 섞인 탈출
	} {
		got, notes := collectDeclared(ws, out, map[string]string{"x": pat})
		if len(got) != 0 {
			t.Fatalf("★ %q 를 걷었다 ★", pat)
		}
		if len(notes) != 1 {
			t.Fatalf("%q — 이유를 안 남겼다: %+v", pat, notes)
		}
	}
	if n := len(harvest(out)); n != 0 {
		t.Fatalf("★ $OUT 에 뭔가 들어갔다 ★: %v", harvest(out))
	}
}

// ★ 심링크를 안 따라간다 ★ — 글롭은 워크스페이스 안이어도 가리키는 곳은 밖일 수 있다.
func TestCollect_심링크를_안_따라간다(t *testing.T) {
	ws, out := t.TempDir(), t.TempDir()
	secret := filepath.Join(t.TempDir(), "비밀.txt")
	if err := os.WriteFile(secret, []byte("비밀-청록"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(ws, "미끼")); err != nil {
		t.Skip("심링크를 못 만든다")
	}

	got, notes := collectDeclared(ws, out, map[string]string{"x": "미끼"})
	if len(got) != 0 {
		b, _ := os.ReadFile(filepath.Join(out, "x"))
		t.Fatalf("★ 심링크를 따라가 밖을 걷었다 ★: %q", b)
	}
	if len(notes) != 1 || !strings.Contains(notes[0].Why, "맞는 파일이 없다") {
		t.Fatalf("이유: %+v", notes)
	}
}

// ★ 여럿이 맞으면 안 걷고 목록을 남긴다 ★ — 하나의 blob 이름에 여럿을 넣으면
// 소비자가 예측을 못 한다 (어떨 땐 .ko, 어떨 땐 묶음).
func TestCollect_여럿이면_안_걷고_알린다(t *testing.T) {
	ws, out := t.TempDir(), t.TempDir()
	mk(t, ws, "m/a.ko", "1")
	mk(t, ws, "m/b.ko", "2")

	got, notes := collectDeclared(ws, out, map[string]string{"modules": "m/*.ko"})
	if len(got) != 0 {
		t.Fatalf("걷으면 안 된다: %v", got)
	}
	if len(notes) != 1 || !strings.Contains(notes[0].Why, "2개가 맞는다") {
		t.Fatalf("이유가 부실하다: %+v", notes)
	}
	if !strings.Contains(notes[0].Why, "a.ko") || !strings.Contains(notes[0].Why, "b.ko") {
		t.Fatalf("★ 무엇이 맞았는지 안 알려줬다 ★: %s", notes[0].Why)
	}
}

// 하나만 맞는 글롭은 걷는다.
func TestCollect_글롭이_하나면_걷는다(t *testing.T) {
	ws, out := t.TempDir(), t.TempDir()
	mk(t, ws, "arch/arm/boot/zImage", "커널")
	got, notes := collectDeclared(ws, out, map[string]string{"kernel": "arch/*/boot/zImage"})
	if len(got) != 1 || len(notes) != 0 {
		t.Fatalf("got=%v notes=%+v", got, notes)
	}
}

// ★ 스크립트가 직접 낸 것이 우선 ★ — collect 는 보조다.
func TestCollect_이미_있으면_안_덮는다(t *testing.T) {
	ws, out := t.TempDir(), t.TempDir()
	mk(t, ws, "a.bin", "워크스페이스것")
	if err := os.WriteFile(filepath.Join(out, "artifact"), []byte("스크립트것"), 0o644); err != nil {
		t.Fatal(err)
	}
	collectDeclared(ws, out, map[string]string{"artifact": "a.bin"})
	b, _ := os.ReadFile(filepath.Join(out, "artifact"))
	if string(b) != "스크립트것" {
		t.Fatalf("★ 덮어썼다 ★: %q", b)
	}
}

// 없으면 ★ 왜 없는지 ★ 를 남긴다 — 새 실패 경로는 안 만든다.
func TestCollect_없으면_이유를_남긴다(t *testing.T) {
	ws, out := t.TempDir(), t.TempDir()
	got, notes := collectDeclared(ws, out, map[string]string{"artifact": "없는/경로.ko"})
	if len(got) != 0 || len(notes) != 1 {
		t.Fatalf("got=%v notes=%+v", got, notes)
	}
	if !strings.Contains(notes[0].Why, "없는/경로.ko") {
		t.Fatalf("★ 어느 경로를 못 찾았는지 안 적었다 ★: %s", notes[0].Why)
	}
}

// 디렉터리는 안 걷는다.
func TestCollect_디렉터리는_안_걷는다(t *testing.T) {
	ws, out := t.TempDir(), t.TempDir()
	if err := os.MkdirAll(filepath.Join(ws, "build"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, notes := collectDeclared(ws, out, map[string]string{"x": "build"})
	if len(got) != 0 || len(notes) != 1 {
		t.Fatalf("got=%v notes=%+v", got, notes)
	}
}

// 못 걷은 이유가 ★ 기록에 실린다 ★ — 없으면 사람이 계약과 트리를 대조해야 한다.
func TestCollect_이유가_기록에_실린다(t *testing.T) {
	out := t.TempDir()
	writeChangedNote(out, []string{"artifact"}, Stamp{},
		[]collectNote{{"artifact", `"arch/arm/boot/zImage" 에 맞는 파일이 없다`}}, testLog())

	b, err := os.ReadFile(filepath.Join(out, changedName))
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if !strings.Contains(got, "collect 가 못 걷은 것") || !strings.Contains(got, "zImage") {
		t.Fatalf("이유가 안 실렸다:\n%s", got)
	}
}
