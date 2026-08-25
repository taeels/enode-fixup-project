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

// 이게 이 파일이 존재하는 이유다
//
// git status 는 .gitignore 를 지켜서 빌드 산출물을 정확히 가린다.
// 빌드 단계는 산출물이 전부 무시 목록에 있으므로, git 만 보면
// 아무 일도 안 한 것처럼 보인다.
func TestChanged_CatchesIgnoredBuildArtifacts(t *testing.T) {
	dir := gitInit(t) // .gitignore 에 build/ 와 *.o 가 있다
	// stampNow 는 1초를 빼둔다 (mtime 해상도가 초 단위인 파일시스템 대비).
	// 그래서 방금 만든 파일이 걸린다 — 시험에서는 쉬었다 찍는다.
	time.Sleep(1100 * time.Millisecond)
	s := stampNow(dir)
	time.Sleep(1100 * time.Millisecond)

	write(t, dir, "build/zImage", strings.Repeat("k", 4096))
	write(t, dir, "drivers/spi.o", "obj")
	write(t, dir, "drivers/spi.ko", "module")
	write(t, dir, "tracked.c", "int x;")

	found, total, err := changedSince(s, 100)
	if err != nil {
		t.Fatal(err)
	}
	if total != 4 {
		t.Fatalf("want 4 (including the ignored ones): %d — %v", total, found)
	}
	names := map[string]bool{}
	for _, c := range found {
		names[filepath.ToSlash(c.Path)] = true
	}
	for _, need := range []string{"build/zImage", "drivers/spi.o", "drivers/spi.ko", "tracked.c"} {
		if !names[need] {
			t.Fatalf("missed %s — the point is to see what git cannot: %v", need, names)
		}
	}
	// 확인: git 은 정말 못 본다 (이 시험의 전제)
	out, _ := gitOut(t.Context(), dir, nil, "status", "--porcelain", "-uall")
	if strings.Contains(string(out), "zImage") {
		t.Fatal("the premise is wrong — if git sees zImage this code is unnecessary")
	}
}

// 기준보다 오래된 것은 안 잡는다 — 데워둔 빌드 캐시가 매번 딸려오면
// 목록이 쓸모없어진다 (ADR-007 이 준비물로 잡은 그 캐시다).
func TestChanged_LeavesTheWarmedCacheAlone(t *testing.T) {
	dir := gitInit(t)
	write(t, dir, "build/cache.o", "stale") // what sanitize leaves behind
	time.Sleep(1100 * time.Millisecond)
	s := stampNow(dir) // sanitize 직후에 찍는다
	time.Sleep(1100 * time.Millisecond)
	write(t, dir, "build/fresh.o", "made-this-time")

	found, total, _ := changedSince(s, 100)
	if total != 1 {
		t.Fatalf("only the 1 made by this step must be caught: %d — %v", total, found)
	}
	if !strings.Contains(found[0].Path, "fresh") {
		t.Fatalf("caught the wrong thing: %v", found)
	}
}

// .git 안은 안 본다 — 인덱스·로그가 계속 바뀌어 목록을 덮는다.
func TestChanged_DoesNotLookInsideGit(t *testing.T) {
	dir := gitInit(t)
	time.Sleep(1100 * time.Millisecond)
	s := stampNow(dir)
	time.Sleep(1100 * time.Millisecond)
	write(t, dir, "real.c", "x")
	git(t, dir, "add", "real.c") // .git/index changes

	found, total, _ := changedSince(s, 100)
	if total != 1 || !strings.Contains(found[0].Path, "real.c") {
		t.Fatalf("the inside of .git got mixed in: %d — %v", total, found)
	}
}

// 수만 개를 그대로 넘기지 않는다 — 커널 빌드가 그렇다.
// 자르되 자른 사실을 숨기지 않는다.
func TestChanged_SummarizesWhenThereAreMany(t *testing.T) {
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
		t.Fatalf("total=%d found=%d — count them all but truncate the list", total, len(found))
	}
	sum := summarize(found, total, 5)
	if !strings.Contains(sum, "files created or modified by this step: 51") || !strings.Contains(sum, "(showing 10)") {
		t.Fatalf("the truncation was not disclosed:\n%s", sum)
	}
	if !strings.Contains(sum, "vmlinux") {
		t.Fatalf("the big ones must come first — they are likely the final artifacts:\n%s", sum)
	}
	if !strings.Contains(sum, ".o ") {
		t.Fatalf("no per-kind tally:\n%s", sum)
	}
}

// 훅이 기준 시각을 받아 빌드 산출물을 짚어준다.
func TestHook_ReportsBuildArtifacts(t *testing.T) {
	dir := gitInit(t)
	out, inst := t.TempDir(), t.TempDir()
	time.Sleep(1100 * time.Millisecond)
	s := stampNow(dir)
	time.Sleep(1100 * time.Millisecond)
	write(t, dir, "build/zImage", strings.Repeat("k", 5000)) // .gitignore 안

	sp := filepath.Join(inst, "stamp")
	if err := writeStamp(sp, s); err != nil {
		t.Fatal(err)
	}
	o := hookRun(t, HookArgs{Out: out, Workspace: dir, Expect: []string{"kernel"}, Stamp: sp},
		StopInput{})
	if o.Decision != "block" {
		t.Fatalf("it did not block: %+v", o)
	}
	if !strings.Contains(o.Reason, "zImage") {
		t.Fatalf("the hook did not see the build artifacts — this is what looking only at git gives:\n%s", o.Reason)
	}
}

// 기준 시각이 없으면 조용히 그 부분만 빠진다 — 훅이 죽지 않는다.
func TestHook_RunsWithoutAStamp(t *testing.T) {
	out := t.TempDir()
	o := hookRun(t, HookArgs{Out: out, Expect: []string{"x"}}, StopInput{})
	if o.Decision != "block" || !strings.Contains(o.Reason, "x") {
		t.Fatalf("%+v", o)
	}
}

// 명령 단계에는 훅이 없다 — 기록이 대신 말해야 한다
//
// 빌드·플래시가 전부 명령 단계이고 ADR-019 로 같은 노드의 cap 아래 들어왔다.
// 되물을 상대가 스크립트라 훅을 못 쓴다. 그러면 기록이
// "무엇을 만들었고 무엇을 안 냈나" 를 스스로 말해야 한다.
func TestNote_RecordsWhatWasMadeAndWhatWasNotProduced(t *testing.T) {
	dir := gitInit(t)
	out := t.TempDir()
	time.Sleep(1100 * time.Millisecond)
	s := stampNow(dir)
	time.Sleep(1100 * time.Millisecond)
	// 빌드가 산출물을 만들었다 — 전부 .gitignore 안이다
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
		t.Fatalf("what was not produced is not pointed out:\n%s", got)
	}
	if !strings.Contains(got, "vmlinux") {
		t.Fatalf("what was made is not recorded — the diff alone looks like nothing happened:\n%s", got)
	}
	if strings.Contains(got, "실패") {
		t.Fatalf("the record passed judgment — success_when does the judging (ADR-004·I3):\n%s", got)
	}
}

// 다 냈으면 「안 낸 것」 절이 없다.
func TestNote_SaysNothingWhenAllWasProduced(t *testing.T) {
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
	if strings.Contains(string(b), "missing") {
		t.Fatalf("pointed something out although everything was produced:\n%s", b)
	}
}

// 보드 단계 — 파일시스템에 흔적이 없다
//
// diff 도 changed 도 비어 있는 것이 정상이다. 시리얼 출력이 산출물이고
// 그건 단계가 직접 $OUT 에 옮겨야 한다. 기록이 그 사실을 말해줘야
// 사람이 "왜 아무것도 안 걷혔지" 를 안 헤맨다.
func TestNote_ExplainsAStepThatLeftNoTrace(t *testing.T) {
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
		t.Fatalf("what was not produced is not pointed out:\n%s", got)
	}
	if !strings.Contains(got, "no files changed") {
		t.Fatalf("the absence of any trace is not recorded — board steps look like this:\n%s", got)
	}
}

func testLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// 지목된 경로만 stat 한다 (ADR-037) — 전체를 걷지 않는다.
func TestCheckChanged(t *testing.T) {
	root := t.TempDir()
	old := filepath.Join(root, "old.c")
	if err := os.WriteFile(old, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// 실물과 같은 방식으로 잡는다 — stampNow 가 1초를 빼는 이유가
	// "파일시스템 mtime 해상도가 초 단위인 경우" 이고, 직접 만들면 그 보정이 빠져
	// 방금 쓴 파일이 기준보다 과거로 보인다 (이 시험이 실제로 그걸 밟았다).
	stamp := stampNow(root)
	past := stamp.At.Add(-time.Hour)
	if err := os.Chtimes(old, past, past); err != nil {
		t.Fatal(err)
	}
	// 기준 이후에 쓰인 파일
	fresh := filepath.Join(root, "sub", "fresh.c")
	if err := os.MkdirAll(filepath.Dir(fresh), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fresh, []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "adir"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := CheckChanged(stamp, []string{"sub/fresh.c", "old.c", "nosuch.c", "adir"})
	if len(got) != 1 || got[0] != "sub/fresh.c" {
		t.Fatalf("only what changed may come back: %v", got)
	}
	// 기준이 없으면 아무것도 안 본다 — 워크스페이스 없는 단계.
	if got := CheckChanged(Stamp{}, []string{"sub/fresh.c"}); got != nil {
		t.Fatalf("no baseline, yet %v", got)
	}
	if got := CheckChanged(stamp, nil); got != nil {
		t.Fatalf("nothing was required, yet %v", got)
	}
}
