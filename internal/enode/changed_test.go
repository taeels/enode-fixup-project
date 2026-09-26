package enode

import (
	"context"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/taeels/enode/internal/contract"
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

// ── 명시 훑기 (discover) ─────────────────────────────────────────
//
// 계약이 discover 를 켠 단계에서만 돈다. 상한 넷 중 먼저 닿은 하나를 적고 멈춘다
// (business-rules.md 5.2). 시험은 상한을 작게 준다.

func bigLimits() discoverLimits {
	return discoverLimits{visits: 1 << 30, held: 1 << 30, paths: 1 << 30, bytes: 1 << 30, time: time.Hour}
}

// tree 는 기준 시각 뒤의 파일 n 개와 앞의 파일 old 개를 만든다. 잠들지 않고 mtime 을 옮긴다.
func tree(t *testing.T, n, old int) (string, Stamp) {
	t.Helper()
	dir := t.TempDir()
	at := time.Now().Add(-time.Hour)
	for i := 0; i < n; i++ {
		write(t, dir, filepath.Join("new", strconv.Itoa(i%7), "f"+strconv.Itoa(i)), strings.Repeat("x", i+1))
	}
	for i := 0; i < old; i++ {
		p := filepath.Join(dir, "old", "f"+strconv.Itoa(i))
		write(t, dir, filepath.Join("old", "f"+strconv.Itoa(i)), "o")
		if err := os.Chtimes(p, at.Add(-time.Hour), at.Add(-time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	return dir, Stamp{At: at, Root: dir}
}

// 다 훑으면 기준 시각 뒤의 보통 파일만, 큰 것부터 나온다. 상한은 안 적힌다.
func TestWalkWorkspace_ListsFilesAfterTheStampLargestFirst(t *testing.T) {
	dir, s := tree(t, 20, 5)
	write(t, dir, ".git/objects/x", "git")
	write(t, dir, ".repo/manifest.xml", "repo")
	d := walkWorkspace(context.Background(), s, bigLimits())
	if d.Total != 20 || len(d.Paths) != 20 || d.Limit != "" {
		t.Fatalf("total %d paths %d limit %q, want 20 · 20 · none", d.Total, len(d.Paths), d.Limit)
	}
	if d.Paths[0].Size != 20 || d.Paths[19].Size != 1 {
		t.Errorf("not largest first: first %+v last %+v", d.Paths[0], d.Paths[19])
	}
	for _, p := range d.Paths {
		if strings.HasPrefix(p.Path, ".git") || strings.HasPrefix(p.Path, ".repo") || strings.HasPrefix(p.Path, "old") {
			t.Errorf("%s must not be listed", p.Path)
		}
	}
	if d.Visited == 0 {
		t.Error("visits were not counted")
	}
}

// 상한마다 — 닿은 하나가 적히고 목록은 부분이다.
func TestWalkWorkspace_StopsAtEachLimit(t *testing.T) {
	dir, s := tree(t, 50, 0)
	cases := []struct {
		name  string
		tweak func(*discoverLimits)
		want  string
	}{
		{"visits", func(l *discoverLimits) { l.visits = 10 }, contract.LimitVisits},
		{"time", func(l *discoverLimits) { l.time = time.Nanosecond }, contract.LimitTime},
		{"memory", func(l *discoverLimits) { l.held = 5 }, contract.LimitMemory},
		{"size by count", func(l *discoverLimits) { l.paths = 3 }, contract.LimitSize},
		{"size by bytes", func(l *discoverLimits) { l.bytes = 30 }, contract.LimitSize},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l := bigLimits()
			c.tweak(&l)
			d := walkWorkspace(context.Background(), Stamp{At: s.At, Root: dir}, l)
			if d.Limit != c.want {
				t.Fatalf("limit = %q, want %q", d.Limit, c.want)
			}
			if len(d.Paths) >= 50 {
				t.Errorf("a limited walk listed everything (%d)", len(d.Paths))
			}
		})
	}
}

// 먼저 닿은 하나만 적는다 — 방문 상한에 멈춘 걷기는 결과 크기에 걸려도 size 가 아니다.
func TestWalkWorkspace_OnlyTheFirstLimitIsRecorded(t *testing.T) {
	_, s := tree(t, 50, 0)
	l := bigLimits()
	l.visits, l.paths = 30, 1
	d := walkWorkspace(context.Background(), s, l)
	if d.Limit != contract.LimitVisits || len(d.Paths) != 1 {
		t.Fatalf("limit %q paths %d, want visits · 1", d.Limit, len(d.Paths))
	}
	if d.Visited != 30 {
		t.Errorf("visited %d, want exactly the visit limit", d.Visited)
	}
}

// Finalize 의 마감이 지나면 멈춘다. 그것도 시간 상한이라 time 으로 적고 목록은 부분이다.
func TestWalkWorkspace_StopsAtTheDeadline(t *testing.T) {
	_, s := tree(t, 10, 0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	d := walkWorkspace(ctx, s, bigLimits())
	if d.Limit != contract.LimitTime || d.Total != 0 || d.Visited != 1 {
		t.Fatalf("limit %q total %d visited %d, want time · 0 · 1", d.Limit, d.Total, d.Visited)
	}
}

// overlay — upper 에 있는 것이 곧 이 단계가 쓴 것이다. 기준 시각을 안 보고,
// whiteout 은 목록에 안 넣고 센다.
func TestWalkUpper_ListsTheUpperAndCountsWhiteouts(t *testing.T) {
	upper := t.TempDir()
	write(t, upper, "src/new.c", "int x;")
	write(t, upper, "build/zImage", strings.Repeat("k", 100))
	write(t, upper, "src/gone.c", "") // 가짜 whiteout — 이름으로 판정한다
	write(t, upper, "src/also-gone.h", "")
	write(t, upper, ".git/index", "i")
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(filepath.Join(upper, "src/new.c"), old, old); err != nil {
		t.Fatal(err)
	}
	fake := func(d fs.DirEntry) bool { return strings.Contains(d.Name(), "gone") }
	d := walkUpper(context.Background(), upper, bigLimits(), fake)
	if d.Total != 2 || d.Deleted != 2 {
		t.Fatalf("total %d deleted %d, want 2 · 2 (%+v)", d.Total, d.Deleted, d.Paths)
	}
	if d.Paths[0].Path != filepath.Join("build", "zImage") {
		t.Errorf("largest first: got %+v", d.Paths)
	}
}

// upper 도 상한을 따른다.
func TestWalkUpper_StopsAtALimit(t *testing.T) {
	upper := t.TempDir()
	for i := 0; i < 20; i++ {
		write(t, upper, "f"+strconv.Itoa(i), "x")
	}
	l := bigLimits()
	l.held = 4
	d := walkUpper(context.Background(), upper, l, func(fs.DirEntry) bool { return false })
	if d.Limit != contract.LimitMemory || d.Total != 4 {
		t.Fatalf("limit %q total %d, want memory · 4", d.Limit, d.Total)
	}
}

// 시간 상한은 Finalize 예산의 절반이다. 0 이면 기본 예산의 절반.
func TestDefaultDiscoverLimits(t *testing.T) {
	if got := defaultDiscoverLimits(0).time; got != 30*time.Second {
		t.Errorf("default time = %v, want 30s", got)
	}
	l := defaultDiscoverLimits(discoverTime(5 * time.Minute))
	if l.time != 150*time.Second || l.visits != 2_000_000 || l.held != 200_000 ||
		l.paths != 2_000 || l.bytes != 256<<10 {
		t.Errorf("limits = %+v", l)
	}
}

// BenchmarkWalkWorkspace 는 이 디스크에서 초당 몇 항목을 훑나다 (계획 3절 ①).
// 30초의 시간 상한과 2,000,000 의 방문 상한 중 무엇이 먼저 닿나를 여기서 읽는다.
//
//	go test ./internal/enode/ -run '^$' -bench WalkWorkspace -benchtime 5x
func BenchmarkWalkWorkspace(b *testing.B) {
	dir := b.TempDir()
	for i := 0; i < 100; i++ {
		sub := filepath.Join(dir, strconv.Itoa(i))
		if err := os.MkdirAll(sub, 0o755); err != nil {
			b.Fatal(err)
		}
		for j := 0; j < 1000; j++ {
			if err := os.WriteFile(filepath.Join(sub, strconv.Itoa(j)), nil, 0o644); err != nil {
				b.Fatal(err)
			}
		}
	}
	s := Stamp{At: time.Now().Add(time.Hour), Root: dir} // 아무것도 안 걸린다 — 방문만 센다
	b.ResetTimer()
	var visited int
	for i := 0; i < b.N; i++ {
		visited = walkWorkspace(context.Background(), s, defaultDiscoverLimits(0)).Visited
	}
	b.ReportMetric(float64(visited)*float64(b.N)/b.Elapsed().Seconds(), "visits/s")
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
