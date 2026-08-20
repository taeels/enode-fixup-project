package enode

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ★ 이게 이 파일이 존재하는 이유다 ★
//
// git status 는 .gitignore 를 지켜서 빌드 산출물을 ★ 정확히 가린다 ★.
// 빌드 단계는 산출물이 전부 무시 목록에 있으므로, git 만 보면
// ★ 아무 일도 안 한 것처럼 보인다 ★.
func TestChanged_무시되는_빌드산출물을_잡는다(t *testing.T) {
	dir := gitInit(t) // .gitignore 에 build/ 와 *.o 가 있다
	// ★ stampNow 는 1초를 빼둔다 ★ (mtime 해상도가 초 단위인 파일시스템 대비).
	// 그래서 방금 만든 파일이 걸린다 — 시험에서는 쉬었다 찍는다.
	time.Sleep(1100 * time.Millisecond)
	s := stampNow(dir)
	time.Sleep(1100 * time.Millisecond)

	write(t, dir, "build/zImage", strings.Repeat("k", 4096))
	write(t, dir, "drivers/spi.o", "obj")
	write(t, dir, "drivers/spi.ko", "module")
	write(t, dir, "추적됨.c", "int x;")

	found, total, err := changedSince(s, 100)
	if err != nil {
		t.Fatal(err)
	}
	if total != 4 {
		t.Fatalf("★ 4개여야 한다 (무시되는 것 포함) ★: %d — %v", total, found)
	}
	names := map[string]bool{}
	for _, c := range found {
		names[filepath.ToSlash(c.Path)] = true
	}
	for _, need := range []string{"build/zImage", "drivers/spi.o", "drivers/spi.ko", "추적됨.c"} {
		if !names[need] {
			t.Fatalf("★ %s 를 못 잡았다 ★ — git 이 못 보는 것을 보는 게 목적이다: %v", need, names)
		}
	}
	// 확인: git 은 정말 못 본다 (이 시험의 전제)
	out, _ := gitOut(t.Context(), dir, nil, "status", "--porcelain", "-uall")
	if strings.Contains(string(out), "zImage") {
		t.Fatal("전제가 틀렸다 — git 이 zImage 를 본다면 이 코드가 필요없다")
	}
}

// ★ 기준보다 오래된 것은 안 잡는다 ★ — 데워둔 빌드 캐시가 매번 딸려오면
// 목록이 쓸모없어진다 (ADR-007 이 준비물로 잡은 그 캐시다).
func TestChanged_데워둔_캐시는_안_잡는다(t *testing.T) {
	dir := gitInit(t)
	write(t, dir, "build/캐시.o", "오래된것") // sanitize 가 남기는 것
	time.Sleep(1100 * time.Millisecond)
	s := stampNow(dir) // ★ sanitize 직후에 찍는다 ★
	time.Sleep(1100 * time.Millisecond)
	write(t, dir, "build/새것.o", "이번에만든것")

	found, total, _ := changedSince(s, 100)
	if total != 1 {
		t.Fatalf("★ 이번 단계가 만든 1개만 잡혀야 한다 ★: %d — %v", total, found)
	}
	if !strings.Contains(found[0].Path, "새것") {
		t.Fatalf("엉뚱한 걸 잡았다: %v", found)
	}
}

// .git 안은 안 본다 — 인덱스·로그가 계속 바뀌어 목록을 덮는다.
func TestChanged_git내부는_안_본다(t *testing.T) {
	dir := gitInit(t)
	time.Sleep(1100 * time.Millisecond)
	s := stampNow(dir)
	time.Sleep(1100 * time.Millisecond)
	write(t, dir, "진짜.c", "x")
	git(t, dir, "add", "진짜.c") // .git/index 가 바뀐다

	found, total, _ := changedSince(s, 100)
	if total != 1 || !strings.Contains(found[0].Path, "진짜.c") {
		t.Fatalf("★ .git 내부가 섞였다 ★: %d — %v", total, found)
	}
}

// ★ 수만 개를 그대로 넘기지 않는다 ★ — 커널 빌드가 그렇다.
// 자르되 ★ 자른 사실을 숨기지 않는다 ★.
func TestChanged_많으면_요약한다(t *testing.T) {
	dir := t.TempDir()
	time.Sleep(1100 * time.Millisecond)
	s := stampNow(dir)
	time.Sleep(1100 * time.Millisecond)
	for i := 0; i < 50; i++ {
		write(t, dir, filepath.Join("o", string(rune('a'+i%26))+string(rune('a'+i/26))+".o"), "x")
	}
	write(t, dir, "vmlinux", strings.Repeat("B", 9000)) // 큰 것

	found, total, _ := changedSince(s, 10)
	if total != 51 || len(found) != 10 {
		t.Fatalf("total=%d found=%d — 세되 자르는 게 맞다", total, len(found))
	}
	sum := summarize(found, total, 5)
	if !strings.Contains(sum, "51개") || !strings.Contains(sum, "그중 10개만") {
		t.Fatalf("★ 자른 사실을 안 밝혔다 ★:\n%s", sum)
	}
	if !strings.Contains(sum, "vmlinux") {
		t.Fatalf("★ 큰 것이 먼저 나와야 한다 ★ — 최종 산출물일 가능성이 높다:\n%s", sum)
	}
	if !strings.Contains(sum, ".o ") {
		t.Fatalf("종류별 집계가 없다:\n%s", sum)
	}
}

// 훅이 기준 시각을 받아 ★ 빌드 산출물을 짚어준다 ★.
func TestHook_빌드산출물을_알려준다(t *testing.T) {
	dir := gitInit(t)
	out, inst := t.TempDir(), t.TempDir()
	time.Sleep(1100 * time.Millisecond)
	s := stampNow(dir)
	time.Sleep(1100 * time.Millisecond)
	write(t, dir, "build/zImage", strings.Repeat("k", 5000)) // ★ .gitignore 안 ★

	sp := filepath.Join(inst, "stamp")
	if err := writeStamp(sp, s); err != nil {
		t.Fatal(err)
	}
	o := hookRun(t, HookArgs{Out: out, Workspace: dir, Expect: []string{"커널"}, Stamp: sp},
		StopInput{})
	if o.Decision != "block" {
		t.Fatalf("안 막았다: %+v", o)
	}
	if !strings.Contains(o.Reason, "zImage") {
		t.Fatalf("★ 훅이 빌드 산출물을 못 봤다 ★ — git 만 보면 이렇게 된다:\n%s", o.Reason)
	}
}

// 기준 시각이 없으면 ★ 조용히 그 부분만 빠진다 ★ — 훅이 죽지 않는다.
func TestHook_기준시각이_없어도_돈다(t *testing.T) {
	out := t.TempDir()
	o := hookRun(t, HookArgs{Out: out, Expect: []string{"x"}}, StopInput{})
	if o.Decision != "block" || !strings.Contains(o.Reason, "x") {
		t.Fatalf("%+v", o)
	}
}

// ★ 명령 단계에는 훅이 없다 — 기록이 대신 말해야 한다 ★
//
// 빌드·플래시가 전부 명령 단계이고 ADR-019 로 같은 노드의 cap 아래 들어왔다.
// 되물을 상대가 스크립트라 훅을 못 쓴다. 그러면 기록이
// "무엇을 만들었고 무엇을 안 냈나" 를 스스로 말해야 한다.
func TestNote_만든것과_안낸것을_같이_적는다(t *testing.T) {
	dir := gitInit(t)
	out := t.TempDir()
	time.Sleep(1100 * time.Millisecond)
	s := stampNow(dir)
	time.Sleep(1100 * time.Millisecond)
	// 빌드가 산출물을 만들었다 — ★ 전부 .gitignore 안이다 ★
	write(t, dir, "build/vmlinux", strings.Repeat("v", 9000))
	write(t, dir, "drivers/spi.ko", strings.Repeat("k", 3000))
	write(t, dir, "drivers/spi.o", "obj")
	// 그런데 $OUT 으로 안 옮겼다

	writeChangedNote(out, []string{"artifact", "build_log"}, s, nil, testLog())

	b, err := os.ReadFile(filepath.Join(out, changedName))
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if !strings.Contains(got, "artifact") || !strings.Contains(got, "build_log") {
		t.Fatalf("★ 안 낸 것을 안 짚었다 ★:\n%s", got)
	}
	if !strings.Contains(got, "vmlinux") {
		t.Fatalf("★ 만든 것을 안 적었다 — diff 만 보면 아무것도 안 한 것처럼 보인다 ★:\n%s", got)
	}
	if strings.Contains(got, "실패") {
		t.Fatalf("★ 기록이 판정했다 ★ — 판정은 success_when 이 한다 (ADR-004·I3):\n%s", got)
	}
}

// 다 냈으면 「안 낸 것」 절이 없다.
func TestNote_다_냈으면_안_짚는다(t *testing.T) {
	dir, out := gitInit(t), t.TempDir()
	if err := os.WriteFile(filepath.Join(out, "artifact"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1100 * time.Millisecond)
	s := stampNow(dir)
	time.Sleep(1100 * time.Millisecond)
	write(t, dir, "build/x.o", "o")

	writeChangedNote(out, []string{"artifact"}, s, nil, testLog())
	b, _ := os.ReadFile(filepath.Join(out, changedName))
	if strings.Contains(string(b), "없는 것") {
		t.Fatalf("다 냈는데 짚었다:\n%s", b)
	}
}

// ★ 보드 단계 — 파일시스템에 흔적이 없다 ★
//
// diff 도 changed 도 비어 있는 것이 ★ 정상 ★ 이다. 시리얼 출력이 산출물이고
// 그건 단계가 직접 $OUT 에 옮겨야 한다. 기록이 그 사실을 말해줘야
// 사람이 "왜 아무것도 안 걷혔지" 를 안 헤맨다.
func TestNote_흔적이_없는_단계도_설명한다(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	time.Sleep(1100 * time.Millisecond)
	s := stampNow(dir)

	writeChangedNote(out, []string{"kunit_result"}, s, nil, testLog())
	b, err := os.ReadFile(filepath.Join(out, changedName))
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if !strings.Contains(got, "kunit_result") {
		t.Fatalf("안 낸 것을 안 짚었다:\n%s", got)
	}
	if !strings.Contains(got, "바뀐 파일이 없다") {
		t.Fatalf("★ 흔적이 없다는 사실을 안 적었다 ★ — 보드 단계가 이렇다:\n%s", got)
	}
}

func testLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }
