package record

import (
	"archive/tar"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// 기본 상한. 10 MiB 는 설정의 MaxBlobBytes 기본값이다.
const bigLimit = 10 << 20

func put(t *testing.T, s *Store, runID string, seq int, name string, attempt int, body string) Progress {
	t.Helper()
	return putLimit(t, s, runID, seq, name, attempt, body, bigLimit)
}

func putLimit(t *testing.T, s *Store, runID string, seq int, name string,
	attempt int, body string, limit int64) Progress {
	t.Helper()
	p, err := s.AppendProgress(runID, seq, name, attempt, strings.NewReader(body), limit)
	if err != nil {
		t.Fatalf("AppendProgress(%s seq=%d name=%s attempt=%d): %v", runID, seq, name, attempt, err)
	}
	return p
}

func get(t *testing.T, s *Store, runID string, seq int, name string, from int64) (Progress, string) {
	t.Helper()
	p, body, err := s.ReadProgress(runID, seq, name, from)
	if err != nil {
		t.Fatalf("ReadProgress(%s seq=%d name=%s from=%d): %v", runID, seq, name, from, err)
	}
	b, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("cannot read the progress body: %v", err)
	}
	if err := body.Close(); err != nil {
		t.Fatalf("cannot close the progress body: %v", err)
	}
	return p, string(b)
}

func progressFilePath(s *Store, runID string, seq int, name string, attempt int) string {
	return filepath.Join(s.progressDir(runID), progressFile(seq, name, attempt))
}

// marks 는 본문에 든 표시 줄의 수다.
func marks(body string) int {
	return strings.Count(body, `"type":"`+cappedType+`"`)
}

// lastLine 은 마지막 줄이다. 끝의 개행 하나는 줄이 아니다.
func lastLine(body string) string {
	s := strings.TrimSuffix(body, "\n")
	if i := strings.LastIndexByte(s, '\n'); i >= 0 {
		return s[i+1:]
	}
	return s
}

// ── Step 12 — 자리 · 이름 · 시도 · 뮤텍스 ────────────────────────────────

// R1 — 진행 트리는 Record 의 형제다. 기록 안이 아니다.
func TestProgressLivesBesideTheRecord(t *testing.T) {
	s := newStore(t)
	if err := s.Open("r"); err != nil {
		t.Fatal(err)
	}
	put(t, s, "r", 1, "build", 1, "hello\n")

	want := filepath.Join(s.Root, "progress", "run-r", "01-build.1.log")
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("the progress file is not at <Root>/progress/run-r: %v", err)
	}
	var inRecord []string
	err := filepath.Walk(s.dir("r"), func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !fi.IsDir() {
			inRecord = append(inRecord, p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(inRecord) != 0 {
		t.Fatalf("the record directory must hold no progress files: got %v", inRecord)
	}
}

// R2 · R6 — 이름을 오른쪽에서 읽는다. 단계 이름에 점이 있어도 안 깨진다.
func TestProgressNameIsReadFromTheRight(t *testing.T) {
	if got := progressFile(1, "build.step", 2); got != "01-build.step.2.log" {
		t.Fatalf("progressFile: got %q, want %q", got, "01-build.step.2.log")
	}
	seq, attempt, name, ok := parseProgressName("01-build.step.2.log")
	if !ok {
		t.Fatal("parseProgressName refused a name it wrote itself")
	}
	if seq != 1 || attempt != 2 || name != "build.step" {
		t.Fatalf("parseProgressName: got seq=%d attempt=%d name=%q, want 1/2/build.step",
			seq, attempt, name)
	}
}

// R4 — 진행 파일은 원문이라 logs/ 보다 좁게 연다.
func TestProgressPermissionsAreTighterThanLogs(t *testing.T) {
	s := newStore(t)
	put(t, s, "r", 1, "build", 1, "x\n")

	fi, err := os.Stat(progressFilePath(s, "r", 1, "build", 1))
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0o600 {
		t.Fatalf("progress file mode: got %o, want 600", got)
	}
	di, err := os.Stat(s.progressDir("r"))
	if err != nil {
		t.Fatal(err)
	}
	if got := di.Mode().Perm(); got != 0o700 {
		t.Fatalf("progress directory mode: got %o, want 700", got)
	}
}

// R3 — 자리는 게으르게 난다. Open 은 진행 트리를 안 만든다.
func TestOpenDoesNotCreateTheProgressTree(t *testing.T) {
	s := newStore(t)
	if err := s.Open("r"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(s.Root, "progress")); !os.IsNotExist(err) {
		t.Fatalf("Store.Open created the progress root: err=%v", err)
	}
	if s.HasProgress() {
		t.Fatal("HasProgress is true before any chunk arrived")
	}
	put(t, s, "r", 1, "build", 1, "x\n")
	if !s.HasProgress() {
		t.Fatal("HasProgress is false after a chunk arrived")
	}
}

// R7 · R8 — 같은 시도는 이어 붙고, 늦게 온 앞 시도는 0 바이트다.
func TestAnOlderAttemptWritesNothing(t *testing.T) {
	s := newStore(t)
	put(t, s, "r", 1, "build", 2, "aa\n")
	p := put(t, s, "r", 1, "build", 2, "bb\n")
	if p.Total != 6 || p.Attempt != 2 {
		t.Fatalf("the same attempt must append: got total=%d attempt=%d, want 6/2", p.Total, p.Attempt)
	}
	late := put(t, s, "r", 1, "build", 1, "cc\n")
	if late.Total != 6 || late.Attempt != 2 {
		t.Fatalf("a late chunk from an older attempt must change nothing: got total=%d attempt=%d, want 6/2",
			late.Total, late.Attempt)
	}
	_, body := get(t, s, "r", 1, "build", 0)
	if strings.Contains(body, "cc") {
		t.Fatalf("the older attempt leaked into the current file: %q", body)
	}
}

// R9 · R16 · R17 — 새 시도가 앞 시도를 걷고, 읽는 쪽은 언제나 가장 큰 시도다.
func TestANewAttemptSweepsTheOlderFile(t *testing.T) {
	s := newStore(t)
	put(t, s, "r", 1, "build", 1, "old\n")
	put(t, s, "r", 1, "build", 2, "new\n")

	if _, err := os.Stat(progressFilePath(s, "r", 1, "build", 1)); !os.IsNotExist(err) {
		t.Fatalf("the older attempt file is still on disk: err=%v", err)
	}
	p, body := get(t, s, "r", 1, "build", 0)
	if p.Attempt != 2 || body != "new\n" {
		t.Fatalf("ReadProgress must show the newest attempt only: got attempt=%d body=%q", p.Attempt, body)
	}
}

// R18 — 아직 아무것도 안 온 단계는 빈 것이지 없는 것이 아니다.
func TestAMissingStepIsNotAnError(t *testing.T) {
	s := newStore(t)
	p, body := get(t, s, "r", 3, "never", 0)
	if p != (Progress{}) || body != "" {
		t.Fatalf("a step with no chunk yet: got %+v body=%q, want the zero progress and an empty body", p, body)
	}
	// 트리는 있고 그 단계만 없는 자리도 같다.
	put(t, s, "r", 1, "build", 1, "x\n")
	p, body = get(t, s, "r", 3, "never", 0)
	if p != (Progress{}) || body != "" {
		t.Fatalf("a missing step inside a live tree: got %+v body=%q", p, body)
	}
}

// R19 · R20 — 되돌아가는 경로 둘의 표시가 다르다.
func TestTheTwoBackwardPathsLookDifferent(t *testing.T) {
	s := newStore(t)
	put(t, s, "r", 1, "build", 1, "aaaaaaaa\n")

	// from 이 Total 보다 커도 오류가 아니다 (R19).
	p, body := get(t, s, "r", 1, "build", 999)
	if p.Total != 9 || body != "" {
		t.Fatalf("from beyond Total: got total=%d body=%q, want 9 and an empty body", p.Total, body)
	}
	// 시도가 바뀌면 Attempt 가 바뀐다.
	put(t, s, "r", 1, "build", 2, "b\n")
	p, _ = get(t, s, "r", 1, "build", 0)
	if p.Attempt != 2 || p.Total != 2 {
		t.Fatalf("a new attempt: got attempt=%d total=%d, want 2/2", p.Attempt, p.Total)
	}
	// 트리가 걷히면 둘이 함께 0 이다 (R20).
	if err := s.DropProgress("r"); err != nil {
		t.Fatal(err)
	}
	p, body = get(t, s, "r", 1, "build", 0)
	if p.Total != 0 || p.Attempt != 0 || body != "" {
		t.Fatalf("after the tree was dropped: got total=%d attempt=%d body=%q, want 0/0 and an empty body",
			p.Total, p.Attempt, body)
	}
}

// R21 — 드롭은 멱등이다.
func TestDropProgressIsIdempotent(t *testing.T) {
	s := newStore(t)
	put(t, s, "r", 1, "build", 1, "x\n")
	for i := 0; i < 3; i++ {
		if err := s.DropProgress("r"); err != nil {
			t.Fatalf("DropProgress call %d: %v", i+1, err)
		}
	}
	if _, err := os.Stat(s.progressDir("r")); !os.IsNotExist(err) {
		t.Fatalf("the progress tree survived the drop: err=%v", err)
	}
}

// blockingReader 는 풀어줄 때까지 잠금을 쥔 채 멈춰 있는 청크다.
type blockingReader struct {
	release <-chan struct{}
	data    string
	done    bool
}

func (r *blockingReader) Read(p []byte) (int, error) {
	<-r.release
	if r.done {
		return 0, io.EOF
	}
	r.done = true
	return copy(p, r.data), nil
}

// R29 — 키가 다르면 안 기다린다.
func TestDifferentStepsDoNotWaitOnEachOther(t *testing.T) {
	s := newStore(t)
	release := make(chan struct{})
	held := make(chan struct{})
	go func() {
		defer close(held)
		_, _ = s.AppendProgress("r", 1, "slow", 1,
			&blockingReader{release: release, data: "x\n"}, bigLimit)
	}()
	// 느린 쪽이 자물쇠를 쥐었다는 신호는 파일이다 — 파일을 여는 것이
	// 잠금 안이고 그 뒤에 읽기가 멈춘다.
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(progressFilePath(s, "r", 1, "slow", 1)); err == nil {
			break
		}
		if time.Now().After(deadline) {
			close(release)
			t.Fatal("the slow write never took its lock")
		}
		time.Sleep(time.Millisecond)
	}

	done := make(chan error, 1)
	go func() {
		_, err := s.AppendProgress("r", 2, "fast", 1, strings.NewReader("y\n"), bigLimit)
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("the other step failed: %v", err)
		}
	case <-time.After(10 * time.Second):
		close(release)
		t.Fatal("a write to a different step waited on another step's lock")
	}
	close(release)
	<-held
}

// R28 · R30 · R31 — 같은 키의 동시 쓰기와 드롭이 섞여도 안 깨진다.
// go test -race 로 재는 자리다.
func TestConcurrentWritesAndDropsAreSafe(t *testing.T) {
	s := newStore(t)
	var mu sync.Mutex
	var errs []error
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 25; j++ {
				_, err := s.AppendProgress("r", 1, "build", 1, strings.NewReader("line\n"), bigLimit)
				if err != nil {
					mu.Lock()
					errs = append(errs, err)
					mu.Unlock()
				}
			}
		}()
	}
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 25; j++ {
				if err := s.DropProgress("r"); err != nil {
					mu.Lock()
					errs = append(errs, err)
					mu.Unlock()
				}
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 25; j++ {
			_, body, err := s.ReadProgress("r", 1, "build", 0)
			if err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
				continue
			}
			_, _ = io.ReadAll(body)
			_ = body.Close()
		}
	}()
	wg.Wait()
	if len(errs) != 0 {
		t.Fatalf("concurrent progress calls failed: %v", errs[0])
	}
	// 참조가 0 이 되면 잠금 맵도 비어야 한다 (R30).
	s.progress.mu.Lock()
	left := len(s.progress.runs)
	s.progress.mu.Unlock()
	if left != 0 {
		t.Fatalf("the lock map kept %d entries after every caller left", left)
	}
}

// R32 — 경로 탈출은 진행 트리에서도 막힌다.
func TestProgressPathTraversalIsBlocked(t *testing.T) {
	s := newStore(t)
	put(t, s, "../../etc/passwd", 1, "../../etc/shadow", 1, "x\n")

	want := filepath.Join(s.Root, "progress", "run-____etc_passwd", "01-____etc_shadow.1.log")
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("the escaping id was not folded into one path component: %v", err)
	}
	outside := filepath.Join(filepath.Dir(s.Root), "etc")
	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		t.Fatalf("something was written outside the store root: err=%v", err)
	}
}

// R6 · R34 — 못 읽는 이름은 조용히 건너뛴다.
func TestAnUnreadableNameIsSkipped(t *testing.T) {
	s := newStore(t)
	put(t, s, "r", 1, "build", 1, "x\n")
	dir := s.progressDir("r")
	for _, junk := range []string{"garbage.txt", "01-build.log", "aa-build.1.log", "01-build.x.log", "nodash.1.log"} {
		if err := os.WriteFile(filepath.Join(dir, junk), []byte("junk\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	p := put(t, s, "r", 1, "build", 1, "y\n")
	if p.Total != 4 {
		t.Fatalf("junk in the tree changed the write: got total=%d, want 4", p.Total)
	}
	p, body := get(t, s, "r", 1, "build", 0)
	if p.Attempt != 1 || body != "x\ny\n" {
		t.Fatalf("junk in the tree changed the read: got attempt=%d body=%q", p.Attempt, body)
	}
	n, err := s.SweepProgress([]string{"r"}, ProgressMaxAge, time.Now())
	if err != nil || n != 0 {
		t.Fatalf("the sweep tripped over junk: got n=%d err=%v", n, err)
	}
}

// ── Step 13 — 상한 · 개행 보장 · 틈 넷 ──────────────────────────────────

// R10 — 상한 안의 청크는 통째로 들어간다. 개행에 안 맞춘다.
func TestAChunkWithinTheLimitGoesInWhole(t *testing.T) {
	s := newStore(t)
	p := putLimit(t, s, "r", 1, "build", 1, "half a li", 200)
	if p.Total != 9 || p.Capped {
		t.Fatalf("a chunk within the limit: got total=%d capped=%v, want 9/false", p.Total, p.Capped)
	}
	_, body := get(t, s, "r", 1, "build", 0)
	if body != "half a li" {
		t.Fatalf("the chunk was reshaped: got %q", body)
	}
}

// R11 — 넘기면 예산 안의 마지막 개행까지만 남는다. 반쪽 줄이 0 이다.
func TestOverflowKeepsOnlyWholeLines(t *testing.T) {
	s := newStore(t)
	p := putLimit(t, s, "r", 1, "build", 1, "aaaa\nbbbb\ncc", 10)
	_, body := get(t, s, "r", 1, "build", 0)
	if strings.Contains(body, "cc") {
		t.Fatalf("a half line survived the cut: %q", body)
	}
	if !strings.HasPrefix(body, "aaaa\nbbbb\n") {
		t.Fatalf("the whole lines within the budget were not kept: %q", body)
	}
	if !p.Capped || marks(body) != 1 {
		t.Fatalf("the mark is missing: capped=%v marks=%d", p.Capped, marks(body))
	}
}

// R14 의 핵심 — 판정은 마지막 줄이지 크기가 아니다.
//
// 개행 없는 큰 청크 하나로 넘기면 예산 안에 개행이 없어 0 바이트가 남고,
// 그 파일은 Total 이 상한보다 작은데 이미 닿은 것이다. 크기 비교로 적은
// 구현은 여기서 빨개진다.
func TestCappedIsJudgedByTheMarkNotTheSize(t *testing.T) {
	s := newStore(t)
	p := putLimit(t, s, "r", 1, "build", 1, strings.Repeat("x", 250), 200)
	if !p.Capped {
		t.Fatal("Capped is false after the limit was reached")
	}
	if p.Total >= 200 {
		t.Fatalf("this fixture must leave Total below the limit: got total=%d, want < 200", p.Total)
	}
	_, body := get(t, s, "r", 1, "build", 0)
	if marks(body) != 1 || lastLine(body) != string(cappedMarker(200)[:len(cappedMarker(200))-1]) {
		t.Fatalf("the file must hold exactly one mark as its last line: %q", body)
	}
}

// R12 · R15 — 표시 줄은 정확히 한 번이고 언제나 마지막이다.
func TestTheMarkAppearsOnceAndStaysLast(t *testing.T) {
	s := newStore(t)
	putLimit(t, s, "r", 1, "build", 1, "aaaa\nbbbb\ncc", 10)
	_, first := get(t, s, "r", 1, "build", 0)

	for i := 0; i < 3; i++ {
		p := putLimit(t, s, "r", 1, "build", 1, "more\n", 10)
		if !p.Capped {
			t.Fatalf("call %d: Capped went back to false", i+1)
		}
	}
	p, body := get(t, s, "r", 1, "build", 0)
	if body != first {
		t.Fatalf("the file grew after the limit was reached:\n got %q\nwant %q", body, first)
	}
	if marks(body) != 1 {
		t.Fatalf("the mark is not exactly once: %d", marks(body))
	}
	if p.Total != int64(len(body)) {
		t.Fatalf("Total drifted from the file: got %d, want %d", p.Total, len(body))
	}
}

// R43 — 표시 줄 앞의 개행을 보장한다. 걷으면 이 시험이 빨개진다.
func TestTheMarkIsPrecededByANewline(t *testing.T) {
	s := newStore(t)
	// 꼬리가 반쪽 줄인 파일을 만든다 — 도는 동안의 정상이다.
	putLimit(t, s, "r", 1, "build", 1, "aaaa\nbb", 200)
	// 개행 없는 큰 청크로 넘긴다. 예산 안에 개행이 없어 0 바이트가 남는다.
	p := putLimit(t, s, "r", 1, "build", 1, strings.Repeat("x", 300), 200)
	if !p.Capped {
		t.Fatal("Capped is false: the mark did not become the last line")
	}
	_, body := get(t, s, "r", 1, "build", 0)
	lines := strings.Split(strings.TrimSuffix(body, "\n"), "\n")
	if len(lines) != 3 || lines[0] != "aaaa" || lines[1] != "bb" {
		t.Fatalf("the half line was not closed on its own line: %q", body)
	}
	var m cappedMark
	if err := json.Unmarshal([]byte(lines[2]), &m); err != nil {
		t.Fatalf("the last line is not a whole mark: %q (%v)", lines[2], err)
	}
	if m.Type != cappedType || m.Bytes != 200 {
		t.Fatalf("the mark carries the wrong values: %+v", m)
	}
}

// 틈 넷 — 어느 자리에서 죽어도 다음 호출이 「표시 줄이 온전히 마지막 줄」로
// 수렴한다 (nfr-design-patterns.md 1.3).
func TestTheFourGapsConverge(t *testing.T) {
	const limit = 20
	full := string(cappedMarker(limit))
	// 틈마다 청크가 다르다. 수렴하려면 그 호출이 상한을 넘겨야 하고, 넘기는
	// 자리는 씨앗의 크기가 정한다 — 첫째 틈만 씨앗이 상한 아래라 청크가
	// 크로싱을 만들어야 한다. 나머지 셋은 씨앗이 이미 상한 위다.
	cases := []struct {
		gap   string
		seed  string
		chunk string
		want  string
	}{
		// ③ 앞에서 죽었다 — 파일은 상한 아래이고 표시 줄이 없다. 다음 호출이
		// 같은 판정을 다시 해서 ③ ~ ⑤ 를 돈다. 그래서 이 청크는 상한을
		// 넘겨야 하고, 예산 안에 개행이 있어 ③ 이 실제로 자른다 — want 가
		// x 를 하나도 안 담는 것이 그 자름이다 (R11).
		{"before the cut write", "aaaa\n", "bbbb\n" + strings.Repeat("x", 20),
			"aaaa\nbbbb\n" + full},
		// ③ 과 ④ 사이 — 예산이 0 이라 ③ 이 0 바이트고 ④ ⑤ 만 돈다.
		{"between the cut write and the newline guard", "aaaaaaaaa\nbbbbbbbbb\n", "zzzz\n",
			"aaaaaaaaa\nbbbbbbbbb\n" + full},
		// ⑤ 중간 — 표시 줄이 반쪽으로 남았다. ④ 가 그 조각을 개행으로 닫는다.
		{"inside the mark write", "aaaa\n" + `{"type":"enode.cap`, "zzzz\n",
			"aaaa\n" + `{"type":"enode.cap` + "\n" + full},
		// ⑤ 뒤 — 이미 온전하다. R15 로 더 안 쓴다.
		{"after the mark write", "aaaa\n" + full, "zzzz\n",
			"aaaa\n" + full},
	}
	for _, c := range cases {
		t.Run(c.gap, func(t *testing.T) {
			s := newStore(t)
			dir := s.progressDir("r")
			if err := os.MkdirAll(dir, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, progressFile(1, "build", 1)),
				[]byte(c.seed), 0o600); err != nil {
				t.Fatal(err)
			}
			p := putLimit(t, s, "r", 1, "build", 1, c.chunk, limit)
			_, body := get(t, s, "r", 1, "build", 0)
			if !p.Capped {
				t.Fatalf("the gap did not converge: capped=false body=%q", body)
			}
			if marks(body) != 1 {
				t.Fatalf("the mark is not exactly once: %d in %q", marks(body), body)
			}
			if lastLine(body) != strings.TrimSuffix(full, "\n") {
				t.Fatalf("the last line is not a whole mark: %q", body)
			}
			if body != c.want {
				t.Fatalf("the file did not converge to the expected bytes:\n got %q\nwant %q", body, c.want)
			}
			if p.Total != int64(len(body)) {
				t.Fatalf("Total drifted from the file: got %d, want %d", p.Total, len(body))
			}
		})
	}
}

// R45 — Total 은 상한보다 클 수 있다. 그것이 정상이다.
//
// 이 시험이 없으면 다음 사람이 Total <= limit 을 불변식으로 적는다.
// 상한은 원문에 거는 것이고 표시 줄과 R43 의 개행은 시스템이 더한 것이다.
func TestTotalMayExceedTheLimit(t *testing.T) {
	s := newStore(t)
	const limit = 20
	p := putLimit(t, s, "r", 1, "build", 1, "aaaaaaaaa\nbbbbbbbbb\nccc\n", limit)
	if !p.Capped {
		t.Fatal("the fixture did not reach the limit")
	}
	if p.Total <= limit {
		t.Fatalf("Total must be allowed past the limit: got %d, want > %d", p.Total, limit)
	}
}

// R13 — Total 은 언제나 파일의 크기다. 표시 줄도 R43 의 개행도 든다.
func TestTotalMatchesTheFileSize(t *testing.T) {
	s := newStore(t)
	const limit = 60
	bodies := []string{"a\n", "bb", strings.Repeat("c", 100), "d\n"}
	for i, b := range bodies {
		p := putLimit(t, s, "r", 1, "build", 1, b, limit)
		fi, err := os.Stat(progressFilePath(s, "r", 1, "build", 1))
		if err != nil {
			t.Fatal(err)
		}
		if p.Total != fi.Size() {
			t.Fatalf("write %d: Total=%d but the file is %d bytes", i, p.Total, fi.Size())
		}
	}
}

// R27 · D4 — AppendLog 은 이번 호출의 바이트로 판정하고 총 길이를 낸다.
func TestAppendLogJudgesByWrittenAndReturnsTheTotal(t *testing.T) {
	s := newStore(t)
	if err := s.Open("r"); err != nil {
		t.Fatal(err)
	}
	const limit = 10
	n1, err := s.AppendLog("r", 1, "build", strings.NewReader("aaaaa"), limit)
	if err != nil {
		t.Fatal(err)
	}
	if n1 != 5 {
		t.Fatalf("the first half: got %d, want the total 5", n1)
	}
	n2, err := s.AppendLog("r", 1, "build", strings.NewReader("bbbbb"), limit)
	if err != nil {
		t.Fatal(err)
	}
	if n2 != 10 {
		t.Fatalf("the second half: got %d, want the total 10", n2)
	}
	b, err := os.ReadFile(filepath.Join(s.dir("r"), "logs", "01-build.log"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "truncated") {
		t.Fatalf("two halves that add up to the limit were marked as truncated: %q", b)
	}
	// 한 번에 상한을 넘긴 로그는 그대로 잘린 것으로 표시된다.
	n3, err := s.AppendLog("r", 2, "big", strings.NewReader(strings.Repeat("x", 100)), limit)
	if err != nil {
		t.Fatal(err)
	}
	big, err := os.ReadFile(filepath.Join(s.dir("r"), "logs", "02-big.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(big), "truncated") {
		t.Fatalf("a log over the limit lost its truncation mark: %q", big)
	}
	if n3 != int64(len(big)) {
		t.Fatalf("the return must be the total length: got %d, want %d", n3, len(big))
	}
}

// 이음매 — 표시 줄의 바이트를 글자 그대로 못 박는다 (계획 16.2 · 16.3).
// U1 의 파서가 이 글자를 읽는다. 필드 이름이나 순서나 타입이 바뀌면 빨개진다.
func TestTheCappedMarkIsByteForByte(t *testing.T) {
	const want = `{"type":"enode.capped","bytes":10485760}` + "\n"
	got := string(cappedMarker(10 << 20))
	if got != want {
		t.Fatalf("the seam changed:\n got %q\nwant %q", got, want)
	}
	if len(got) != 41 {
		t.Fatalf("the mark is %d bytes, want 41 (40 plus the newline)", len(got))
	}
	// Total 이 그만큼 는다.
	s := newStore(t)
	const limit = 20
	putLimit(t, s, "r", 1, "build", 1, "aaaaaaaaa\nbbbbbbbbb\n", limit)
	p := putLimit(t, s, "r", 1, "build", 1, "z", limit)
	if p.Total != 20+int64(len(string(cappedMarker(limit)))) {
		t.Fatalf("Total did not grow by the mark: got %d", p.Total)
	}
}

// ── Step 14 — 봉인 경로 불변식 ─────────────────────────────────────────

// R35 — 봉인된 묶음에 원문이 실리지 않는다.
func TestTarHasNoProgressEntries(t *testing.T) {
	s := newStore(t)
	if err := s.Open("r"); err != nil {
		t.Fatal(err)
	}
	put(t, s, "r", 1, "build", 1, "raw transcript\n")
	if err := s.Seal("r", map[string]any{"run_id": "r"}, map[string]any{"state": "SUCCEEDED"}, steps()); err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	if err := s.Tar("r", &buf); err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(strings.NewReader(buf.String()))
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(h.Name, "progress") {
			t.Fatalf("the sealed bundle carries a progress entry: %s", h.Name)
		}
	}
}

// R36 — 봉인은 진행 트리를 얼리지 않는다. 늦게 온 청크도 지울 수 있다.
func TestTheProgressTreeStaysWritableAfterSealing(t *testing.T) {
	s := newStore(t)
	if err := s.Seal("r", map[string]any{"run_id": "r"}, map[string]any{"state": "SUCCEEDED"}, steps()); err != nil {
		t.Fatal(err)
	}
	put(t, s, "r", 1, "build", 1, "late\n")
	put(t, s, "r", 1, "build", 1, "later\n")
	if err := s.DropProgress("r"); err != nil {
		t.Fatalf("the progress tree could not be removed after sealing: %v", err)
	}
}

// R22 · R47 — 봉인이 진행 트리를 지운다. 조기 반환보다 앞이라 이미 봉인된
// Run 에 다시 불러도 늦은 청크의 트리를 지운다.
func TestSealDropsTheProgressTreeEvenWhenAlreadySealed(t *testing.T) {
	s := newStore(t)
	if err := s.Open("r"); err != nil {
		t.Fatal(err)
	}
	put(t, s, "r", 1, "build", 1, "raw\n")
	if err := s.Seal("r", map[string]any{"run_id": "r"}, map[string]any{"state": "SUCCEEDED"}, steps()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.progressDir("r")); !os.IsNotExist(err) {
		t.Fatalf("sealing left the progress tree behind: err=%v", err)
	}
	// 늦게 온 청크가 트리를 다시 만든다.
	put(t, s, "r", 1, "build", 1, "late\n")
	if _, err := os.Stat(s.progressDir("r")); err != nil {
		t.Fatalf("the late chunk did not land: %v", err)
	}
	if err := s.Seal("r", map[string]any{"run_id": "r"}, map[string]any{"state": "SUCCEEDED"}, steps()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.progressDir("r")); !os.IsNotExist(err) {
		t.Fatalf("the second seal left the late tree behind: err=%v", err)
	}
}

// R48 — 진행 트리를 못 지워도 봉인은 선다.
//
// 권한으로 실패를 만들지 않는다 — root 로 돌면 안 발동한다. progress 자리에
// 파일 하나를 놓으면 그 아래를 지우는 것이 ENOTDIR 로 실패한다.
func TestSealSurvivesADropFailure(t *testing.T) {
	s := newStore(t)
	if err := os.MkdirAll(s.Root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.Root, "progress"), []byte("not a directory\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := s.DropProgress("r"); err == nil {
		t.Fatal("the fixture did not make DropProgress fail")
	}
	if err := s.Seal("r", map[string]any{"run_id": "r"}, map[string]any{"state": "SUCCEEDED"}, steps()); err != nil {
		t.Fatalf("a failed drop blocked the seal: %v", err)
	}
	if !s.Sealed("r") {
		t.Fatal("the record was not sealed")
	}
}

// 자리를 못 만들거나 파일을 못 여는 갈래도 오류로 나온다.
func TestAppendProgressSurfacesFilesystemFailures(t *testing.T) {
	s := newStore(t)
	if err := os.WriteFile(filepath.Join(s.Root, "progress"), []byte("not a directory\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AppendProgress("r", 1, "build", 1, strings.NewReader("x\n"), bigLimit); err == nil {
		t.Fatal("AppendProgress swallowed a directory creation failure")
	}
	if _, _, err := s.ReadProgress("r", 1, "build", 0); err != nil {
		t.Fatalf("ReadProgress must stay quiet when there is no tree: %v", err)
	}
	if err := os.Remove(filepath.Join(s.Root, "progress")); err != nil {
		t.Fatal(err)
	}

	// 파일 이름 자리에 디렉터리를 놓으면 여는 것이 실패한다.
	dir := s.progressDir("r")
	if err := os.MkdirAll(filepath.Join(dir, progressFile(1, "build", 1)), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AppendProgress("r", 1, "build", 1, strings.NewReader("x\n"), bigLimit); err == nil {
		t.Fatal("AppendProgress swallowed an open failure")
	}
}

// ── 쓸기 — 기제만 여기서 잰다. 정책(살아 있는 Run)은 internal/store 다 ──

func TestSweepProgressIsQuietWithoutATree(t *testing.T) {
	s := newStore(t)
	n, err := s.SweepProgress(nil, ProgressMaxAge, time.Now())
	if n != 0 || err != nil {
		t.Fatalf("an empty store: got n=%d err=%v, want 0/nil", n, err)
	}
	if s.HasProgress() {
		t.Fatal("HasProgress is true with no progress root")
	}
}

func TestSweepProgressTakesOrphansWholeAndAgedFilesOneByOne(t *testing.T) {
	s := newStore(t)
	put(t, s, "live", 1, "old", 1, "x\n")
	put(t, s, "live", 2, "fresh", 1, "y\n")
	put(t, s, "gone", 1, "build", 1, "z\n")

	// 한 파일만 늙게 만든다.
	old := progressFilePath(s, "live", 1, "old", 1)
	past := time.Now().Add(-7 * time.Hour)
	if err := os.Chtimes(old, past, past); err != nil {
		t.Fatal(err)
	}
	n, err := s.SweepProgress([]string{"live"}, ProgressMaxAge, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("the sweep took %d items, want 2 (one orphan tree and one aged file)", n)
	}
	if _, err := os.Stat(s.progressDir("gone")); !os.IsNotExist(err) {
		t.Fatalf("the orphan tree survived: err=%v", err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("the aged file survived: err=%v", err)
	}
	if _, err := os.Stat(progressFilePath(s, "live", 2, "fresh", 1)); err != nil {
		t.Fatalf("the fresh file of a live run was taken: %v", err)
	}
}
