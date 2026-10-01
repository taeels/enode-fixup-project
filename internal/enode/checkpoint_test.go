//go:build linux

package enode

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/scratch"
)

// 실패한 단계의 보존을 Worker 에 잇는 자리 (checkpoint 유닛) — 판정의 차례 · keep 의 결과 · receipt · 버림 · 보고의 성패.

// testKeeper 는 임시 scratch 의 spool 을 든 Keeper 다. scratch 의 경로를 dir 로 받는다 — 경로 표지 시험이 쓴다.
func testKeeper(t *testing.T, dir string, mode scratch.Mode) (*CheckpointKeeper, *scratch.Store) {
	t.Helper()
	st := &scratch.Store{Dir: scratch.SpoolIn(dir), Scratch: dir, Trash: scratch.TrashIn(dir),
		Policy: scratch.Policy{Mode: mode, TTL: time.Hour, CapacityPercent: 20, MaxBytes: 32 * int64(gib), MinFree: gib},
		Stat:   func() (scratch.Filesystem, error) { return scratch.Filesystem{Known: true, Free: 1000 * gib}, nil }}
	if _, err := st.Open(); err != nil {
		t.Fatal(err)
	}
	return &CheckpointKeeper{Policy: st.Policy, Store: st, Support: overlayCapture, Node: "n1",
		Runtime: "runc-overlay"}, st
}

// spoolRuntime 은 보존의 keep 에 정해진 결과로 답하는 세션을 연다 — runc-overlay 의 Close 대신이다. 명령은 native 로 돈다.
type spoolRuntime struct{ mode string }

func (spoolRuntime) Capability() RuntimeCapability {
	return RuntimeCapability{Writes: writesIsolated, Capture: overlayCapture}
}

func (r spoolRuntime) Open(ctx context.Context, spec RuntimeSpec) (StepSession, error) {
	s, err := NativeRuntime{}.Open(ctx, spec)
	return &spoolSession{StepSession: s, mode: r.mode}, err
}

type spoolSession struct {
	StepSession
	mode string
}

func (s *spoolSession) Close(ctx context.Context, keep Keep) error {
	err := s.StepSession.Close(ctx, keep)
	if keep.Result == nil {
		return err
	}
	moved := func() {
		_ = os.MkdirAll(filepath.Join(keep.Upper, "sub"), 0o700)
		_ = os.WriteFile(filepath.Join(keep.Upper, "sub", "built"), []byte("built"), 0o600)
		keep.Result.Moved = true
	}
	switch s.mode {
	case "late":
		keep.Result.Late = true
	case "exists":
		keep.Result.Err = &os.LinkError{Op: "rename", Old: "/run/root/upper", New: keep.Upper, Err: syscall.EEXIST}
	case "abort":
		keep.Result.Err = errKeepAborted
	case "close-error":
		moved()
		return errors.New("unmount failed")
	default:
		moved()
	}
	return err
}

func failingStep() *Step {
	step := runStep("sh", "-c", "exit 1")
	step.Name = "build"
	return step
}

func spoolEntries(t *testing.T, st *scratch.Store) []string {
	t.Helper()
	ents, err := os.ReadDir(st.Dir)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, e := range ents {
		if scratch.ValidID(e.Name()) {
			ids = append(ids, e.Name())
		}
	}
	return ids
}

// 규칙 1절의 표 — 굽기 build 는 failEnd 만 · 성공 끝은 always 여도 보존하지 않는다.
func TestCheckpointRequested(t *testing.T) {
	for _, tc := range []struct {
		mode         scratch.Mode
		failed, bake bool
		want         bool
	}{
		{scratch.ModeOff, true, false, false},
		{scratch.ModeOnFailure, false, false, false},
		{scratch.ModeOnFailure, true, false, true},
		{scratch.ModeAlways, false, false, true},
		{scratch.ModeAlways, true, false, true},
		{scratch.ModeOnFailure, true, true, true},
		{scratch.ModeAlways, false, true, false},
		{scratch.ModeOff, true, true, false},
	} {
		k := &CheckpointKeeper{Policy: scratch.Policy{Mode: tc.mode}}
		if got := k.requested(closeFacts{failed: tc.failed, bake: tc.bake}); got != tc.want {
			t.Fatalf("%+v: requested = %v", tc, got)
		}
	}
}

// 어휘가 두 벌인 자리 (domain-entities.md 8절) — scratch 의 글자가 contract 와 같다.
func TestCheckpointVocabulary_MatchesContract(t *testing.T) {
	pairs := [][2]string{
		{scratch.StateNotRequested, contract.CaptureNotRequested},
		{scratch.StateUnsupported, contract.CaptureUnsupported},
		{scratch.StateRejected, contract.CaptureRejected},
		{scratch.StateCaptured, contract.CaptureCaptured},
		{scratch.StateFailed, contract.CaptureFailed},
	}
	for _, p := range pairs {
		if p[0] != p[1] {
			t.Fatalf("%q != %q", p[0], p[1])
		}
	}
}

// NFR C6 — 계약에 보존 칸이 없다. 정책은 노드 소유자의 것이다 (결정 2-11).
func TestContractHasNoCheckpointField(t *testing.T) {
	seen := map[reflect.Type]bool{}
	var walk func(reflect.Type, string)
	walk = func(ty reflect.Type, path string) {
		for ty.Kind() == reflect.Pointer || ty.Kind() == reflect.Slice || ty.Kind() == reflect.Map {
			ty = ty.Elem()
		}
		if ty.Kind() != reflect.Struct || seen[ty] {
			return
		}
		seen[ty] = true
		for i := 0; i < ty.NumField(); i++ {
			f := ty.Field(i)
			name := strings.ToLower(strings.Split(f.Tag.Get("json"), ",")[0] + f.Name)
			if strings.Contains(name, "checkpoint") {
				t.Fatalf("the contract has a checkpoint field at %s.%s", path, f.Name)
			}
			walk(f.Type, path+"."+f.Name)
		}
	}
	walk(reflect.TypeOf(contract.Contract{}), "Contract")
	walk(reflect.TypeOf(Step{}), "Step")
}

// NFR C3 — 오류 문장은 errno 의 글만이다. 경로를 담은 오류에서 경로가 안 나온다.
func TestCheckpointDetail_ErrnoOnly(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want string
	}{
		{&fs.PathError{Op: "mkdir", Path: "/home/me/scratch/spool/x", Err: syscall.EACCES}, "permission denied"},
		{&os.LinkError{Op: "rename", Old: "/a", New: "/home/me/scratch/spool/x/upper", Err: syscall.EEXIST}, "file exists"},
		{errors.New("open /home/me/scratch/spool: broken"), "unexpected error"},
		{nil, "no reservation was made"},
	} {
		if got := errnoText(tc.err); got != tc.want || strings.Contains(got, "/") {
			t.Fatalf("errnoText(%v) = %q", tc.err, got)
		}
	}
}

// keep 의 결과마다 receipt · diagnostics 가 맞고, finalize 칸 · error · exit 은 그대로다 (규칙 2 · 3절 · FR-10).
func TestCloseOut_KeepResults(t *testing.T) {
	for _, tc := range []struct {
		mode, state, reason, detail string
	}{
		{"", contract.CaptureCaptured, "", ""},
		{"late", contract.CaptureRejected, scratch.ReasonLeaseBudget, noteBudgetBefore},
		{"exists", contract.CaptureFailed, scratch.ReasonIO, noteMove + "file exists"},
		{"abort", contract.CaptureFailed, scratch.ReasonIO, noteAborted},
	} {
		t.Run("mode "+tc.mode, func(t *testing.T) {
			m, w := finalizeWorker(t)
			k, st := testKeeper(t, t.TempDir(), scratch.ModeOnFailure)
			w.Runtime, w.Checkpoints = spoolRuntime{mode: tc.mode}, k
			w.execute(context.Background(), failingStep())

			res := m.only(t)
			c := res.CheckpointCapture
			if c == nil || c.State != tc.state || c.Reason != tc.reason || res.Diagnostics.Checkpoint != tc.detail {
				t.Fatalf("capture = %+v detail %q", c, res.Diagnostics.Checkpoint)
			}
			if res.Finalize != contract.StageOK || res.Error != "" || res.ExitCode == nil || *res.ExitCode != 1 {
				t.Fatalf("the capture changed the step: finalize %q error %q exit %v", res.Finalize, res.Error, res.ExitCode)
			}
			ids := spoolEntries(t, st)
			if tc.state == contract.CaptureCaptured {
				if len(ids) != 1 || c.ID != ids[0] || c.Node != "n1" || c.Scope != "workspace-upper" ||
					c.Guarantee != "inspect-only" || c.ExpiresAt == nil {
					t.Fatalf("captured = %+v · spool %v", c, ids)
				}
				l, err := st.Lookup(c.ID)
				if err != nil || l == nil || l.State != scratch.EntryKept || l.Report != scratch.ReportDelivered ||
					l.Run != "r1" || l.Step != "build" {
					t.Fatalf("record = %+v %v", l, err)
				}
				return
			}
			if len(ids) != 0 {
				t.Fatalf("a capture that did not happen left %v in the spool", ids)
			}
		})
	}
}

// 다른 Close 오류는 오늘처럼 finalize 칸 error 다 — 보존이 성공해도 (규칙 3절 · ADR-076 §4).
func TestCloseOut_OtherCloseErrorsStillFail(t *testing.T) {
	m, w := finalizeWorker(t)
	k, _ := testKeeper(t, t.TempDir(), scratch.ModeOnFailure)
	w.Runtime, w.Checkpoints = spoolRuntime{mode: "close-error"}, k
	w.execute(context.Background(), failingStep())
	res := m.only(t)
	if res.Finalize != contract.StageError || !strings.Contains(res.Error, "runtime cleanup: unmount failed") {
		t.Fatalf("finalize %q error %q", res.Finalize, res.Error)
	}
	if res.CheckpointCapture == nil || res.CheckpointCapture.State != contract.CaptureCaptured {
		t.Fatalf("capture = %+v", res.CheckpointCapture)
	}
}

// 성공한 단계는 not_requested 이고 예약은 보고 뒤에 버린다 (FD 답 1 · NFR Design 답 1).
func TestReserve_DisposedWhenNotRequested(t *testing.T) {
	m, w := finalizeWorker(t)
	k, st := testKeeper(t, t.TempDir(), scratch.ModeOnFailure)
	w.Runtime, w.Checkpoints = spoolRuntime{}, k
	var during []string
	checkpointPause = func(at string) {
		if at == "reserved" {
			during = spoolEntries(t, st)
		}
	}
	t.Cleanup(func() { checkpointPause = func(string) {} })
	w.execute(context.Background(), runStep("sh", "-c", "exit 0"))
	res := m.only(t)
	if res.CheckpointCapture == nil || res.CheckpointCapture.State != contract.CaptureNotRequested ||
		res.Diagnostics.Checkpoint != "" {
		t.Fatalf("capture = %+v", res.CheckpointCapture)
	}
	if len(during) != 1 {
		t.Fatalf("no reservation was made at open: %v", during)
	}
	if ids := spoolEntries(t, st); len(ids) != 0 {
		t.Fatalf("the reservation of a step that succeeded was not disposed: %v", ids)
	}
}

// 세션을 열 때의 예약이 실패해도 단계는 돌고, 닫을 때 요구하면 그 errno 로 알린다 (규칙 2절 ⑤).
func TestReserve_FailureReportedAtClose(t *testing.T) {
	m, w := finalizeWorker(t)
	k, st := testKeeper(t, t.TempDir(), scratch.ModeOnFailure)
	if err := os.Chmod(st.Dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(st.Dir, 0o700) })
	w.Runtime, w.Checkpoints = spoolRuntime{}, k
	w.execute(context.Background(), failingStep())
	res := m.only(t)
	c := res.CheckpointCapture
	if c == nil || c.State != contract.CaptureFailed || c.Reason != scratch.ReasonIO ||
		res.Diagnostics.Checkpoint != noteReserve+"permission denied" || res.ExitCode == nil || *res.ExitCode != 1 {
		t.Fatalf("capture = %+v detail %q", c, res.Diagnostics.Checkpoint)
	}
}

// NFR P2 — 느린 예약과 느린 확정을 끼워도 finalize 판정은 그대로다. 예약은 명령 앞이고 확정은 closedAt 뒤다. 확정이
// 마감을 넘기면 보존만 버린다 (failed · lease_budget).
func TestCheckpoint_SlowReservationKeepsFinalize(t *testing.T) {
	m, w := finalizeWorker(t)
	k, st := testKeeper(t, t.TempDir(), scratch.ModeOnFailure)
	w.Runtime, w.Checkpoints = spoolRuntime{}, k
	w.budgets = func(*Step) (time.Duration, time.Duration) { return 100 * time.Millisecond, time.Minute }
	checkpointPause = func(string) { time.Sleep(300 * time.Millisecond) }
	t.Cleanup(func() { checkpointPause = func(string) {} })
	w.execute(context.Background(), failingStep())
	res := m.only(t)
	if res.Finalize != contract.StageOK || res.Reason != "" || res.Error != "" || res.ExitCode == nil || *res.ExitCode != 1 {
		t.Fatalf("a slow capture changed the step: finalize %q reason %q error %q", res.Finalize, res.Reason, res.Error)
	}
	c := res.CheckpointCapture
	if c == nil || c.State != contract.CaptureFailed || c.Reason != scratch.ReasonLeaseBudget ||
		res.Diagnostics.Checkpoint != noteBudgetAfter {
		t.Fatalf("capture = %+v detail %q", c, res.Diagnostics.Checkpoint)
	}
	if ids := spoolEntries(t, st); len(ids) != 0 {
		t.Fatalf("a discarded capture is left in the spool: %v", ids)
	}
}

// NFR C2 — host 경로가 result 와 올린 단계 로그에 없다. scratch 경로에 표지 글자를 넣고 규칙 2절의 갈래를 모두 돈다.
func TestCheckpoint_NoHostPathLeaves(t *testing.T) {
	const marker = "PATHMARK7f3a"
	type setup func(t *testing.T, w *Worker, k *CheckpointKeeper, st *scratch.Store)
	cases := map[string]struct {
		mode  string
		setup setup
		state string
	}{
		"captured": {"", nil, contract.CaptureCaptured},
		"late":     {"late", nil, contract.CaptureRejected},
		"exists":   {"exists", nil, contract.CaptureFailed},
		"abort":    {"abort", nil, contract.CaptureFailed},
		"native": {"", func(t *testing.T, w *Worker, k *CheckpointKeeper, st *scratch.Store) {
			k.Support = NativeRuntime{}.Capability().Capture
		}, contract.CaptureUnsupported},
		"cross filesystem": {"", func(t *testing.T, w *Worker, k *CheckpointKeeper, st *scratch.Store) {
			st.Scratch = "/proc"
		}, contract.CaptureUnsupported},
		"free space": {"", func(t *testing.T, w *Worker, k *CheckpointKeeper, st *scratch.Store) {
			st.Stat = func() (scratch.Filesystem, error) { return scratch.Filesystem{Known: true}, nil }
		}, contract.CaptureRejected},
		"quota": {"", func(t *testing.T, w *Worker, k *CheckpointKeeper, st *scratch.Store) {
			r, err := st.Reserve(scratch.Entry{Run: "old"})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := st.Commit(r, r.Entry()); err != nil {
				t.Fatal(err)
			}
			st.Measure = func(context.Context, string, string) (scratch.Size, error) {
				return scratch.Size{Bytes: 30 * int64(gib), Entries: 1}, nil
			}
			if _, err := st.Settle(context.Background()); err != nil {
				t.Fatal(err)
			}
			st.Stat = func() (scratch.Filesystem, error) { return scratch.Filesystem{Known: true, Free: 100 * gib}, nil }
		}, contract.CaptureRejected},
		"reservation failed": {"", func(t *testing.T, w *Worker, k *CheckpointKeeper, st *scratch.Store) {
			if err := os.Chmod(st.Dir, 0o500); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(st.Dir, 0o700) })
		}, contract.CaptureFailed},
		"refused spool": {"", func(t *testing.T, w *Worker, k *CheckpointKeeper, st *scratch.Store) {
			dir := filepath.Dir(st.Dir)
			if err := os.Rename(st.Dir, filepath.Join(dir, "elsewhere")); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.Join(dir, "elsewhere"), st.Dir); err != nil {
				t.Fatal(err)
			}
			if check, err := st.Open(); err != nil || check.Refused != "symlink" {
				t.Fatalf("open = %+v %v", check, err)
			}
		}, contract.CaptureFailed},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			m, w := finalizeWorker(t)
			dir := filepath.Join(t.TempDir(), marker, "scratch")
			k, st := testKeeper(t, dir, scratch.ModeOnFailure)
			w.Runtime, w.Checkpoints = spoolRuntime{mode: tc.mode}, k
			if tc.setup != nil {
				tc.setup(t, w, k, st)
			}
			w.execute(context.Background(), failingStep())
			res := m.only(t)
			if res.CheckpointCapture == nil || res.CheckpointCapture.State != tc.state {
				t.Fatalf("capture = %+v detail %q", res.CheckpointCapture, res.Diagnostics.Checkpoint)
			}
			if tc.state != contract.CaptureCaptured && res.Diagnostics.Checkpoint == "" {
				t.Fatal("a capture that did not happen has no reason in diagnostics")
			}
			body, err := json.Marshal(res)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(body), marker) {
				t.Fatalf("the result carries a host path: %s", body)
			}
			if log := m.logOf("build"); strings.Contains(log, marker) {
				t.Fatalf("the step log carries a host path:\n%s", log)
			}
		})
	}
}

// 보고의 네 끝이 기록의 report 칸에 남는다 (규칙 8절).
func TestReport_RecordsOutcome(t *testing.T) {
	k, st := testKeeper(t, t.TempDir(), scratch.ModeOnFailure)
	capture := func(t *testing.T) *contract.CheckpointCapture {
		r, err := st.Reserve(scratch.Entry{Run: "r1"})
		if err != nil {
			t.Fatal(err)
		}
		e, err := st.Commit(r, r.Entry())
		if err != nil {
			t.Fatal(err)
		}
		return &contract.CheckpointCapture{State: contract.CaptureCaptured, ID: e.ID}
	}
	reportOf := func(t *testing.T, id string) string {
		l, err := st.Lookup(id)
		if err != nil || l == nil {
			t.Fatalf("lookup = %+v %v", l, err)
		}
		return l.Report
	}
	rejecting := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "the step is not claimed", http.StatusConflict)
	}))
	t.Cleanup(rejecting.Close)
	gone := httptest.NewServer(http.NotFoundHandler())
	gone.Close()

	for _, tc := range []struct {
		name, want string
		worker     func(t *testing.T) (*Worker, context.Context)
	}{
		{"delivered", scratch.ReportDelivered, func(t *testing.T) (*Worker, context.Context) {
			_, w := finalizeWorker(t)
			return w, context.Background()
		}},
		{"rejected", scratch.ReportRejected, func(t *testing.T) (*Worker, context.Context) {
			_, w := finalizeWorker(t)
			w.Client.Base = rejecting.URL
			return w, context.Background()
		}},
		{"lease ended", scratch.ReportLeaseEnded, func(t *testing.T) (*Worker, context.Context) {
			m := newMediator(t)
			w := newWorker(m)
			w.Client.Base = gone.URL
			return w, context.Background()
		}},
		{"stopped", scratch.ReportUnknown, func(t *testing.T) (*Worker, context.Context) {
			_, w := finalizeWorker(t)
			w.Client.Base = gone.URL
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return w, ctx
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, ctx := tc.worker(t)
			w.Checkpoints = k
			c := capture(t)
			w.report(ctx, runStep(), Result{Node: "n1", CheckpointCapture: c})
			if got := reportOf(t, c.ID); got != tc.want {
				t.Fatalf("report = %q, want %q", got, tc.want)
			}
		})
	}
}

// 확정의 기록 쓰기가 실패하면 failed(io) 이고 errno 만 남는다 — 예약은 버린다 (규칙 2절 ⑥).
func TestCheckpointFinish_RecordFailure(t *testing.T) {
	k, st := testKeeper(t, t.TempDir(), scratch.ModeOnFailure)
	slot := k.Reserve(runStep())
	plan := k.Decide(closeFacts{step: runStep(), failed: true, deadline: time.Now().Add(time.Hour), slot: slot})
	if plan == nil || plan.decided != nil || plan.keep.Upper != slot.resv.Upper() {
		t.Fatalf("plan = %+v", plan)
	}
	plan.result.Moved = true
	// 기록을 쓸 자리가 사라졌다 — 확정이 실패한다
	if err := os.RemoveAll(slot.resv.Dir); err != nil {
		t.Fatal(err)
	}
	c, detail := k.Finish(plan)
	if c.State != contract.CaptureFailed || c.Reason != scratch.ReasonIO || detail != noteRecord+"no such file or directory" {
		t.Fatalf("capture = %+v detail %q", c, detail)
	}
	if ids := spoolEntries(t, st); len(ids) != 0 {
		t.Fatalf("the failed capture is left in the spool: %v", ids)
	}
	var nilKeeper *CheckpointKeeper
	if c, d := nilKeeper.Finish(nil); c != nil || d != "" || nilKeeper.Decide(closeFacts{}) != nil ||
		nilKeeper.Reserve(runStep()) != nil {
		t.Fatal("a nil keeper must leave everything as today")
	}
	if got := (*capturePlan)(nil).Keep(Keep{Upper: "pending"}); got.Upper != "pending" {
		t.Fatalf("a nil plan changed the keep: %+v", got)
	}
}

// 기동 조정 뒤 판정 — 주인 없는 예약은 trash 로 가고 측정 전 보존본은 측정한다. 판정이 상태 파일의 spool 칸을 채운다
// (business-rules.md 6 · 9절).
func TestKeeperRun_ReconcileThenSettle(t *testing.T) {
	k, st := testKeeper(t, t.TempDir(), scratch.ModeOnFailure)
	config := filepath.Join(t.TempDir(), "node.yaml")
	k.Status = NewStatusBook(config, nil)
	ownerless, err := st.Reserve(scratch.Entry{Run: "r0"})
	if err != nil {
		t.Fatal(err)
	}
	record, err := os.ReadFile(filepath.Join(ownerless.Dir, "checkpoint.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Dispose(ownerless); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(ownerless.Dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string][]byte{"checkpoint.json": record, scratch.SessionLockName: nil} {
		if err := os.WriteFile(filepath.Join(ownerless.Dir, name), body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	kept, err := st.Reserve(scratch.Entry{Run: "r1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.Commit(kept, kept.Entry()); err != nil {
		t.Fatal(err)
	}
	st.Measure = func(context.Context, string, string) (scratch.Size, error) {
		return scratch.Size{Bytes: 4096, Entries: 3}, nil
	}
	trashed := make(chan struct{}, 4)
	k.OnTrash = func() { trashed <- struct{}{} }
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { k.Run(ctx); close(done) }()
	deadline := time.Now().Add(10 * time.Second)
	for {
		got, err := ReadStatus(config)
		if err == nil && got.Scratch != nil && got.Scratch.Checkpoints == 1 && got.Scratch.SpoolBytes == 4096 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the spool usage never reached the status file: %+v %v", got.Scratch, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := os.Stat(filepath.Join(st.Trash.Dir, ownerless.ID)); err != nil {
		t.Fatalf("the ownerless reservation is not in trash: %v", err)
	}
	select {
	case <-trashed:
	default:
		t.Fatal("the deleter was not woken after the reconcile moved an entry")
	}
	cancel()
	<-done
}

// 판정 중에 온 깸은 끝난 뒤 한 번으로 합친다 (NFR Design D4). 도는 동안의 버림과 보고의 성패는 판정이 한다.
func TestKeeperRun_KickCoalesces(t *testing.T) {
	k, st := testKeeper(t, t.TempDir(), scratch.ModeOnFailure)
	k.Every = time.Hour
	passes := make(chan struct{}, 100)
	release := make(chan struct{})
	first := true
	st.Stat = func() (scratch.Filesystem, error) {
		passes <- struct{}{}
		if first {
			first = false
			<-release
		}
		return scratch.Filesystem{Known: true, Free: 1000 * gib}, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { k.Run(ctx); close(done) }()
	<-passes // 첫 판정이 막혀 있다
	for i := 0; i < 10; i++ {
		k.Kick()
	}
	slot := k.Reserve(runStep())
	k.Dispose(slot)
	close(release)
	<-passes // 깸 열 번이 판정 한 번이 된다
	time.Sleep(100 * time.Millisecond)
	if n := len(passes); n != 0 {
		t.Fatalf("%d more passes after ten kicks", n)
	}
	if ids := spoolEntries(t, st); len(ids) != 0 {
		t.Fatalf("a disposal queued while running was not done: %v", ids)
	}
	cancel()
	<-done
}

func TestStartScratch_WiresAfterReport(t *testing.T) {
	logs := func() (*slog.Logger, *strings.Builder) {
		var b strings.Builder
		return slog.New(slog.NewTextHandler(&lockedWriter{w: &b}, nil)), &b
	}
	t.Run("runc-overlay", func(t *testing.T) {
		dir := t.TempDir()
		config := filepath.Join(dir, "node.yaml")
		log, buf := logs()
		ctx, cancel := context.WithCancel(context.Background())
		work := StartScratch(ctx, ScratchSetup{Scratch: filepath.Join(dir, "scratch"), Config: config,
			Local: Local{MinFreeGB: 10}, Runtime: spoolRuntime{}, Node: "n1", Status: NewStatusBook(config, nil), Log: log})
		if work.Keeper == nil || work.Keeper.Store == nil || work.AfterReport == nil {
			t.Fatalf("work = %+v", work)
		}
		work.AfterReport()
		cancel()
		work.Wait()
		got, err := ReadStatus(config)
		if err != nil || got.Checkpoint == nil ||
			*got.Checkpoint != (CheckpointStatus{Policy: "on-failure", TTLHours: 48, CapacityPercent: 20}) {
			t.Fatalf("status = %+v %v", got.Checkpoint, err)
		}
		want := "checkpoint policy on-failure, ttl 48h, capacity 20 percent of kept plus free space; change it under checkpoint: in " + config
		if !strings.Contains(buf.String(), want) {
			t.Fatalf("the startup line is missing:\n%s", buf.String())
		}
		if fi, err := os.Stat(scratch.SpoolIn(filepath.Join(dir, "scratch"))); err != nil || fi.Mode().Perm() != 0o700 {
			t.Fatalf("spool = %v %v", fi, err)
		}
	})
	t.Run("native", func(t *testing.T) {
		config := filepath.Join(t.TempDir(), "node.yaml")
		log, buf := logs()
		work := StartScratch(context.Background(), ScratchSetup{Config: config, Status: NewStatusBook(config, nil), Log: log,
			Local: Local{Checkpoint: &CheckpointConfig{Policy: "always"}}})
		if work.Keeper == nil || work.Keeper.Store != nil || work.AfterReport != nil || work.Keeper.Support.Supported {
			t.Fatalf("work = %+v", work)
		}
		work.Wait()
		got, err := ReadStatus(config)
		if err != nil || got.Checkpoint == nil || got.Checkpoint.Unsupported != "runtime" || got.Checkpoint.Policy != "always" {
			t.Fatalf("status = %+v %v", got.Checkpoint, err)
		}
		if !strings.Contains(buf.String(), "checkpoint: this node runs steps without an isolated upper and keeps nothing (runtime)") {
			t.Fatalf("the native line is missing:\n%s", buf.String())
		}
	})
	t.Run("off · narrowed · refused", func(t *testing.T) {
		for _, tc := range []struct {
			name  string
			setup func(spool string)
			want  string
		}{
			{"off", nil, "checkpoint policy off; failed steps leave nothing behind"},
			{"narrowed", func(spool string) {
				_ = os.Mkdir(spool, 0o700)
				_ = os.Chmod(spool, 0o755)
			}, `msg="checkpoint spool permissions narrowed to 0700" mode=755`},
			{"refused", func(spool string) {
				_ = os.WriteFile(spool, nil, 0o600)
			}, `msg="the checkpoint spool is not a private directory of this node; nothing will be kept" why="not a directory"`},
		} {
			dir := t.TempDir()
			scratchDir := filepath.Join(dir, "scratch")
			if err := os.MkdirAll(scratchDir, 0o700); err != nil {
				t.Fatal(err)
			}
			if tc.setup != nil {
				tc.setup(scratch.SpoolIn(scratchDir))
			}
			local := Local{}
			if tc.name == "off" {
				local.Checkpoint = &CheckpointConfig{Policy: "off"}
			}
			log, buf := logs()
			ctx, cancel := context.WithCancel(context.Background())
			work := StartScratch(ctx, ScratchSetup{Scratch: scratchDir, Local: local, Runtime: spoolRuntime{},
				Log: log})
			cancel()
			work.Wait()
			if !strings.Contains(buf.String(), tc.want) {
				t.Fatalf("%s: the line is missing:\n%s", tc.name, buf.String())
			}
		}
	})
}

// lockedWriter 는 고루틴 여럿이 쓰는 로그를 시험이 읽게 한다.
type lockedWriter struct {
	mu sync.Mutex
	w  *strings.Builder
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}
