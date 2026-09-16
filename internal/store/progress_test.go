package store

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/taeels/enode/internal/record"
)

// 이 파일은 고아 진행 트리 쓸기의 정책을 잰다 (N2 · R24).
//
// 기제는 internal/record 가 지고 그쪽 시험이 잰다 — 어느 디렉터리를 통째로
// 지우고 어느 파일만 나이로 지우는가. 여기서 재는 것은 그 기제에 무엇이
// 들어가는가다: 살아 있는 Run 이 무엇인지는 DB 만 알고, 그것을 아는 쪽이
// store 다. 그래서 이 시험들은 전부 Reap 을 통과해서 잰다.
//
// 슬립이 0 이다. 나이는 os.Chtimes 로 파일의 mtime 을 과거로 당겨 만든다 —
// 여섯 시간을 기다리는 시험은 아무도 안 돌린다.

// progressTestStore 는 쓸기를 잴 수 있는 Store 를 연다.
// 판은 이 시험만의 것이고 (reapTestStore), 기록 뿌리는 임시 디렉터리다.
func progressTestStore(t *testing.T) (*Store, *record.Store) {
	t.Helper()
	st := reapTestStore(t)
	root := t.TempDir()
	// 봉인된 기록은 0444 · 0555 라 t.TempDir 이 못 지운다. 봉인이 삭제까지
	// 막는 것 자체가 I4 가 산다는 증거이므로, 권한을 되돌리는 정리를 건다.
	// t.Cleanup 은 역순이라 이것이 TempDir 의 정리보다 먼저 돈다.
	t.Cleanup(func() {
		_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return nil //nolint:nilerr // 정리다. 판정을 안 바꾼다
			}
			if d.IsDir() {
				_ = os.Chmod(p, 0o700)
			} else {
				_ = os.Chmod(p, 0o600)
			}
			return nil
		})
	})
	rec := record.New(root)
	st.Records = rec
	return st, rec
}

// seedRun 은 Run 행 하나를 심는다. state 가 살았는지 죽었는지를 가른다.
func seedRun(t *testing.T, st *Store, runID, state string) {
	t.Helper()
	if _, err := st.pool.Exec(context.Background(), `
		INSERT INTO runs (run_id, state, principal, contract)
		VALUES ($1, $2, 'test', '{}'::jsonb)`, runID, state); err != nil {
		t.Fatalf("cannot seed the run %s: %v", runID, err)
	}
}

// writeProgress 는 진행 파일 하나를 실제 경로로 쓴다.
func writeProgress(t *testing.T, rec *record.Store, runID string, seq int, name, body string) {
	t.Helper()
	if _, err := rec.AppendProgress(runID, seq, name, 1, strings.NewReader(body), 1<<20); err != nil {
		t.Fatalf("cannot write the progress file for %s: %v", runID, err)
	}
}

// progressTree 는 그 Run 의 진행 디렉터리 경로다.
// run_id 에 경로 문자를 안 쓰므로 safe 가 그대로 통과한다.
func progressTree(rec *record.Store, runID string) string {
	return filepath.Join(rec.Root, "progress", "run-"+runID)
}

func treeExists(t *testing.T, rec *record.Store, runID string) bool {
	t.Helper()
	_, err := os.Stat(progressTree(rec, runID))
	return err == nil
}

func fileExists(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Stat(path)
	return err == nil
}

// testLogger 는 남긴 줄을 그대로 모으는 로거다. 질의가 돌았는지를 로그로
// 재는 자리에서 쓴다.
func testLogger(w io.Writer) *slog.Logger {
	return slog.New(slog.NewTextHandler(w, nil))
}

// ageFile 은 파일의 mtime 을 과거로 당긴다. 슬립 대신이다.
func ageFile(t *testing.T, path string, by time.Duration) {
	t.Helper()
	old := time.Now().Add(-by)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatalf("cannot age %s: %v", path, err)
	}
}

// V4 · R37 — 종료한 Run 의 트리는 유예 0 으로 통째로 사라진다.
func TestReapSweepsTheTreeOfATerminalRun(t *testing.T) {
	st, rec := progressTestStore(t)
	seedRun(t, st, "dead-run", StateSucceeded)
	writeProgress(t, rec, "dead-run", 1, "build", "raw\n")
	if !treeExists(t, rec, "dead-run") {
		t.Fatal("the fixture did not create the progress tree")
	}

	if _, err := st.Reap(context.Background(), quietLogger()); err != nil {
		t.Fatalf("Reap: %v", err)
	}
	if treeExists(t, rec, "dead-run") {
		t.Fatal("the tree of a terminal run survived one Reap")
	}
}

// R38 — DB 에 행이 아예 없는 Run 의 트리도 걷힌다.
// 제출이 반쯤 죽은 자리다. 살아 있는 집합에 없으면 고아다.
func TestReapSweepsATreeWithNoRunRow(t *testing.T) {
	st, rec := progressTestStore(t)
	writeProgress(t, rec, "ghost-run", 1, "build", "raw\n")

	if _, err := st.Reap(context.Background(), quietLogger()); err != nil {
		t.Fatalf("Reap: %v", err)
	}
	if treeExists(t, rec, "ghost-run") {
		t.Fatal("the tree of a run with no database row survived the sweep")
	}
}

// V5 — 살아 있는 Run 은 파일마다 나이로 가른다.
// 늙은 파일만 걷히고 방금 쓴 파일은 그대로다. 트리 자체도 남는다.
func TestReapSweepsAgedFilesButKeepsFreshOnesOfTheSameRun(t *testing.T) {
	st, rec := progressTestStore(t)
	seedRun(t, st, "live-run", StateRunning)
	writeProgress(t, rec, "live-run", 1, "old", "raw\n")
	writeProgress(t, rec, "live-run", 2, "new", "raw\n")

	dir := progressTree(rec, "live-run")
	old := filepath.Join(dir, "01-old.1.log")
	fresh := filepath.Join(dir, "02-new.1.log")
	if !fileExists(t, old) || !fileExists(t, fresh) {
		t.Fatalf("the fixture did not create both files in %s", dir)
	}
	// 상한보다 한 시간 더 늙힌다.
	ageFile(t, old, record.ProgressMaxAge+time.Hour)

	if _, err := st.Reap(context.Background(), quietLogger()); err != nil {
		t.Fatalf("Reap: %v", err)
	}
	if fileExists(t, old) {
		t.Fatal("the aged file of a live run survived the sweep")
	}
	if !fileExists(t, fresh) {
		t.Fatal("the sweep took a freshly written file of the same live run")
	}
	if !treeExists(t, rec, "live-run") {
		t.Fatal("the sweep took the whole tree of a live run")
	}
}

// R37 의 순서 — 상태를 먼저 보고 종료가 아닐 때만 나이를 본다.
//
// 종료한 Run 의 방금 쓴 파일도 걷힌다. 나이를 먼저 보는 구현은 여기서
// 빨개진다 — 그 구현은 이 파일을 여섯 시간 더 남긴다.
func TestReapSweepsAFreshFileOfATerminalRun(t *testing.T) {
	st, rec := progressTestStore(t)
	seedRun(t, st, "just-ended", StateFailed)
	writeProgress(t, rec, "just-ended", 1, "build", "raw\n")

	if _, err := st.Reap(context.Background(), quietLogger()); err != nil {
		t.Fatalf("Reap: %v", err)
	}
	if treeExists(t, rec, "just-ended") {
		t.Fatal("a freshly written tree of a terminal run survived: the age was checked before the state")
	}
}

// R39 · R35 — 쓸기는 진행 트리만 만진다. Record 는 그대로다.
func TestTheSweepLeavesTheRecordAlone(t *testing.T) {
	st, rec := progressTestStore(t)
	seedRun(t, st, "sealed-run", StateSucceeded)
	if err := rec.Open("sealed-run"); err != nil {
		t.Fatalf("cannot open the record: %v", err)
	}
	writeProgress(t, rec, "sealed-run", 1, "build", "raw\n")
	if err := rec.Seal("sealed-run",
		map[string]any{"run_id": "sealed-run"},
		map[string]any{"state": "SUCCEEDED"}, nil); err != nil {
		t.Fatalf("cannot seal the record: %v", err)
	}
	// 봉인 뒤에 늦은 청크가 트리를 다시 만든 모양을 만든다.
	writeProgress(t, rec, "sealed-run", 1, "build", "late\n")

	if _, err := st.Reap(context.Background(), quietLogger()); err != nil {
		t.Fatalf("Reap: %v", err)
	}
	if treeExists(t, rec, "sealed-run") {
		t.Fatal("the late tree of a terminal run survived the sweep")
	}
	if !rec.Sealed("sealed-run") {
		t.Fatal("the sweep unsealed the record")
	}
	var buf strings.Builder
	if err := rec.Tar("sealed-run", &buf); err != nil {
		t.Fatalf("the record is not readable after the sweep: %v", err)
	}
	if strings.Contains(buf.String(), "progress") {
		t.Fatal("the sealed bundle carries a progress entry")
	}
}

// V2 — 쓸기는 회수가 0 인 주기에도 돈다.
//
// sealExpired 안에 두면 이 시험이 빨개진다. 그쪽은 n > 0 뒤라 회수가 없는
// 주기에는 아예 안 돌고, 고아는 회수와 무관하게 생긴다.
func TestTheSweepRunsInACycleThatReclaimsNothing(t *testing.T) {
	st, rec := progressTestStore(t)
	// 만료된 임대를 하나도 안 심는다. 회수가 0 이다.
	writeProgress(t, rec, "orphan-run", 1, "build", "raw\n")

	n, err := st.Reap(context.Background(), quietLogger())
	if err != nil {
		t.Fatalf("Reap: %v", err)
	}
	if n != 0 {
		t.Fatalf("this fixture must reclaim nothing: got %d", n)
	}
	if treeExists(t, rec, "orphan-run") {
		t.Fatal("the sweep did not run in a cycle that reclaimed nothing")
	}
}

// R23 — 봉인 경로의 진행 트리 걷기는 Seal 보다 앞이다.
//
// StepFiles 가 실패하면 Seal 이 아예 안 불린다. 그 경로에서도 트리가 걷혀야
// 한다 — 안 그러면 봉인이 못 선 Run 의 트리를 아무도 안 지운다.
// StepFiles 는 nodes 에 LEFT JOIN 하므로 그 테이블을 감추면 그 질의만 깨진다.
// GetRun 은 runs 만 읽어 그대로 선다.
func TestSealRecordDropsTheTreeEvenWhenStepFilesFails(t *testing.T) {
	st, rec := progressTestStore(t)
	ctx := context.Background()
	seedRun(t, st, "half-sealed", StateFailed)
	writeProgress(t, rec, "half-sealed", 1, "build", "raw\n")

	if _, err := st.pool.Exec(ctx, `ALTER TABLE nodes RENAME TO nodes_hidden`); err != nil {
		t.Fatalf("cannot hide the nodes table: %v", err)
	}

	err := st.sealRecord(ctx, "half-sealed", Verdict{State: StateFailed})
	if err == nil {
		t.Fatal("sealRecord: got nil error, want the StepFiles failure surfaced")
	}
	if rec.Sealed("half-sealed") {
		t.Fatal("the record was sealed even though StepFiles failed")
	}
	if treeExists(t, rec, "half-sealed") {
		t.Fatal("the progress tree survived a seal path that never reached Seal")
	}
}

// 계획 8.1 — 진행 트리가 없으면 살아 있는 Run 질의가 아예 안 돈다.
//
// runs.state 에 인덱스가 없어 그 질의는 전수 스캔이다. U4 와 U7 이 서기 전에는
// 진행 트리가 언제나 0 이므로 주기마다 도는 스캔을 하나도 만들면 안 된다.
//
// 어떻게 재나 — 취소된 컨텍스트로 부른다. 질의가 돌면 취소로 실패해 로그가
// 한 줄 난다. 디렉터리 검사가 먼저 막으면 로그가 0 줄이다. 짝을 함께 둔다:
// 트리가 있으면 같은 컨텍스트에서 그 로그가 실제로 난다.
func TestTheSweepSkipsTheQueryWithoutATree(t *testing.T) {
	st, rec := progressTestStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var log strings.Builder
	st.sweepProgress(ctx, testLogger(&log))
	if log.Len() != 0 {
		t.Fatalf("the sweep queried the database with no progress tree: %s", log.String())
	}

	// 짝 — 트리가 있으면 같은 컨텍스트에서 질의가 돌고 실패가 보인다.
	writeProgress(t, rec, "some-run", 1, "build", "raw\n")
	st.sweepProgress(ctx, testLogger(&log))
	if log.Len() == 0 {
		t.Fatal("with a progress tree the sweep must reach the query; the directory check is not what gated it")
	}
}
