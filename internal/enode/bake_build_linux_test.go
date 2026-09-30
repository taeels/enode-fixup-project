//go:build linux

package enode

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/lower"
)

// build 단계 (bake 유닛 · FD 규칙 1 ~ 6 · 15 · 16절 · FD 흐름 2 · 7절). 가짜 런타임 · 진짜 lower 자리 · 가짜 Mediator.

// 분기와 거절 (FD 규칙 1절)

// 제자리에 쓰는 노드는 거절한다 — Bake 가 nil · Bake 가 있고 Runtime 이 nil · Bake 가 있고 NativeRuntime{}. 뒤의
// 둘은 NativeRuntime 으로 떨어뜨리는 되돌림을 잡는다. 세션을 열지 않고 상태를 건드리지 않는다. 빈 argv 확인에 닿지
// 않는다 — build 단계에는 run 이 없다.
func TestBakeBuild_ANodeWithoutAnUpperRefuses(t *testing.T) {
	for name, set := range map[string]func(f *bakeFixture){
		"no baker":                  func(f *bakeFixture) { f.w.Bake = nil },
		"a baker but no runtime":    func(f *bakeFixture) { f.w.Runtime = nil },
		"a baker and a native node": func(f *bakeFixture) { f.w.Runtime = NativeRuntime{} },
	} {
		t.Run(name, func(t *testing.T) {
			f := newBakeFixture(t)
			set(f)
			marker := markerPATH(t, "bash", "sh")
			for _, step := range []*Step{buildStepOf("r1"), mergeStepOf("r1")} {
				f.w.execute(context.Background(), step)
				res := f.m.lastResult(t)
				if res.Error != refusedText || res.ExitCode != nil || res.Reason != "" {
					t.Fatalf("%s result = %+v", step.Kind, res)
				}
			}
			if len(f.rt.ran()) != 0 || len(f.m.exits()) != 0 {
				t.Fatalf("something ran: %v %v", f.rt.scripts(), f.m.exits())
			}
			if b, err := os.ReadFile(marker); !os.IsNotExist(err) {
				t.Fatalf("a host program ran: %q", b)
			}
			if st := f.state(t); st.Phase != lower.PhaseCommitted || st.LastAttempt != nil {
				t.Fatalf("state = %+v", st)
			}
		})
	}
}

// 상태 자리 · 잠금 파일 · state.json 을 못 읽으면 bake_in_progress 가 아니라 cannot open 이고 잡은 잠금을 놓는다 (결정 29).
func TestBakeBuild_TheStateCannotBeRead(t *testing.T) {
	t.Run("the directory did not open", func(t *testing.T) {
		f := newBakeFixture(t)
		dir := t.TempDir()
		g := newLowerGuard(filepath.Join(dir, "lowers"), filepath.Join(dir, "missing"), Identity{NodeID: "n"}, f.log)
		g.start()
		f.w.Bake = StartBaker(context.Background(), g, f.scratch, Identity{NodeID: "n"}, "i", f.log)
		f.w.execute(context.Background(), buildStepOf("r1"))
		res := f.m.only(t)
		if !strings.HasPrefix(res.Error, "cannot open the lower state directory: ") || res.Reason != "" {
			t.Fatalf("result = %+v", res)
		}
	})
	t.Run("the bake lock cannot be opened", func(t *testing.T) {
		f := newBakeFixture(t)
		p := filepath.Join(f.g.Dir().Path, "bake.lock")
		if err := os.Remove(p); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(p, 0o700); err != nil {
			t.Fatal(err)
		}
		f.w.execute(context.Background(), buildStepOf("r1"))
		if res := f.m.only(t); !strings.HasPrefix(res.Error, "cannot open the lower state directory: ") || res.Reason != "" {
			t.Fatalf("result = %+v", res)
		}
	})
	t.Run("state.json cannot be read", func(t *testing.T) {
		f := newBakeFixture(t)
		if err := os.WriteFile(filepath.Join(f.g.Dir().Path, "state.json"), []byte(`{"schema":1,"phase":"weird"}`), 0o600); err != nil {
			t.Fatal(err)
		}
		f.w.execute(context.Background(), buildStepOf("r1"))
		if res := f.m.only(t); !strings.HasPrefix(res.Error, "cannot open the lower state directory: ") || res.Reason != "" {
			t.Fatalf("result = %+v", res)
		}
		if f.held() != nil {
			t.Fatal("a failed claim kept the bake")
		}
		f.holdBakeLock(t) // 잡은 잠금을 놓았다
	})
}

// 굽기 잠금의 주인이 살아 있으면 bake_in_progress 와 주인 문장이다 (US-16). 이 프로세스가 쥔 굽기 · 재개 중도 같다.
func TestBakeBuild_AnotherBakeHoldsTheLower(t *testing.T) {
	t.Run("a live owner", func(t *testing.T) {
		f := newBakeFixture(t)
		since := time.Date(2026, 9, 27, 4, 10, 0, 0, time.UTC)
		st := lower.State{Phase: lower.PhasePending, Owner: &lower.Owner{Run: "R-other", Node: "node-b"}, Since: since,
			PendingUpper: "/x/pending/k/n/upper"}
		f.putState(t, st)
		f.holdBakeLock(t)
		f.w.execute(context.Background(), buildStepOf("r1"))
		res := f.m.only(t)
		if res.Reason != contract.ReasonBakeInProgress || res.Error != bakeInProgressText(f.state(t)) ||
			!strings.Contains(res.Error, "run R-other on node node-b since 2026-09-27T04:10:00Z (pending)") {
			t.Fatalf("result = %+v", res)
		}
		if len(f.rt.ran()) != 0 {
			t.Fatalf("commands ran: %v", f.rt.scripts())
		}
	})
	t.Run("this process holds a bake", func(t *testing.T) {
		f := newBakeFixture(t)
		f.hold(t, "R-mine")
		f.w.execute(context.Background(), buildStepOf("r1"))
		if res := f.m.only(t); res.Reason != contract.ReasonBakeInProgress {
			t.Fatalf("result = %+v", res)
		}
	})
	t.Run("this process is resuming", func(t *testing.T) {
		f := newBakeFixture(t)
		f.b.mu.Lock()
		f.b.resuming = true
		f.b.mu.Unlock()
		f.w.execute(context.Background(), buildStepOf("r1"))
		if res := f.m.only(t); res.Reason != contract.ReasonBakeInProgress {
			t.Fatalf("result = %+v", res)
		}
		f.b.mu.Lock()
		f.b.resuming = false
		f.b.mu.Unlock()
	})
	t.Run("a stale pending is cleaned and the bake goes on", func(t *testing.T) {
		f := newBakeFixture(t)
		theirs := filepath.Join(filepath.Dir(f.ws), "their-scratch")
		if err := os.Mkdir(theirs, 0o700); err != nil {
			t.Fatal(err)
		}
		old := f.pendingIn(t, theirs, "R-dead", true)
		f.putState(t, lower.State{Phase: lower.PhasePending, Owner: &lower.Owner{Run: "R-dead"},
			PendingUpper: filepath.Join(old, "upper")})
		f.buildOK(t, "r1")
		if got := entries(t, filepath.Join(theirs, "trash")); len(got) != 1 || got[0] != filepath.Base(old) {
			t.Fatalf("their trash = %v", got)
		}
		st := f.state(t)
		if st.Owner.Run != "r1" || st.LastAttempt == nil || st.LastAttempt.Run != "R-dead" ||
			st.LastAttempt.Reason != abandonedReason(lower.PhasePending) {
			t.Fatalf("the pending state did not carry the abandoned record: %+v %+v", st, st.LastAttempt)
		}
	})
}

// 성공 (FD 규칙 3 · 3.1 · 15절)

func TestBakeBuild_Succeeds(t *testing.T) {
	f := newBakeFixture(t)
	prev := "ir-0"
	if err := lower.WriteMetadata(f.ws, lower.Metadata{Source: lower.Source{IR: &prev},
		Bake: lower.BakeRecord{Run: "R-0"}}); err != nil {
		t.Fatal(err)
	}
	var building lower.State
	f.rt.answer(syncCmd, fakeAnswer{stdout: "fetched\n", after: func() { building = f.state(t) }})
	f.rt.answer(buildA, fakeAnswer{stdout: "built a\n", do: func(upper string) { write(t, upper, "out/a.bin", "a") }})
	f.succeed()
	f.w.execute(context.Background(), buildStepOf("r1"))
	res := f.m.only(t)

	if got := f.rt.scripts(); !reflect.DeepEqual(got, []string{bashCheck, syncCmd, probeScript, buildA, buildB}) {
		t.Fatalf("commands = %q", got)
	}
	if res.Error != "" || res.ExitCode == nil || *res.ExitCode != 0 || res.Finalize != contract.StageOK ||
		!reflect.DeepEqual(res.Produced, []string{contract.ArtifactManifest}) || res.Reason != "" {
		t.Fatalf("result = %+v", res)
	}
	if ex := f.m.exits(); len(ex) != 1 || *ex[0].Outcome.Code != 0 || !ex[0].ExitedAt.Equal(*res.ExitedAt) {
		t.Fatalf("exited = %+v", ex)
	}
	// building 때 owner · pending_upper · since 가 적힌다 (FD 규칙 2절)
	if building.Phase != lower.PhaseBuilding || building.Owner == nil || building.Owner.Run != "r1" ||
		building.Owner.Node != "node-a" || building.Owner.Instance != "inst-1" || building.PendingUpper == "" ||
		building.Since.IsZero() {
		t.Fatalf("building = %+v", building)
	}
	st := f.state(t)
	if st.Phase != lower.PhasePending || st.PendingUpper != building.PendingUpper || *st.Owner != *building.Owner {
		t.Fatalf("pending = %+v", st)
	}
	pending := filepath.Dir(st.PendingUpper)
	if b, err := os.ReadFile(filepath.Join(st.PendingUpper, "out", "a.bin")); err != nil || string(b) != "a" {
		t.Fatalf("the upper did not reach the pending directory: %q %v", b, err)
	}
	if keeps := f.rt.keeps; len(keeps) == 0 || keeps[0].Upper != st.PendingUpper {
		t.Fatalf("the first close did not keep the upper: %+v", keeps)
	}
	// 초안의 칸 전부 (0600)
	if m := mode(t, filepath.Join(pending, draftName)); m != 0o600 {
		t.Fatalf("the draft is %o", m)
	}
	d, err := readDraft(pending, os.Getuid())
	if err != nil {
		t.Fatal(err)
	}
	want := lower.Source{URL: "https://u@git.example/x.git", Branch: "main",
		RepoID: CanonicalRepoID("https://git.example/x.git"), Head: headA, SyncCommand: syncCmd}
	if d.Source.URL != want.URL || d.Source.Branch != want.Branch || d.Source.RepoID != want.RepoID ||
		d.Source.Head != want.Head || d.Source.SyncCommand != want.SyncCommand || *d.Source.IR != bakeIR ||
		d.Source.Pinned != nil || d.Source.SyncedAt.IsZero() || d.Run != "r1" || d.Node != "node-a" ||
		d.Environment != "prep-1" || d.WorkspaceTarget != "/work" || d.PreviousIR == nil || *d.PreviousIR != "ir-0" ||
		len(d.Builds) != 2 || d.Builds[1].Name != "config-b" {
		t.Fatalf("draft = %+v source %+v", d, d.Source)
	}
	// HoldBake 뒤 공유를 안 잡는다
	if f.guardBake() != "r1" {
		t.Fatalf("HoldBake = %q", f.guardBake())
	}
	f.g.BeforeAdvert(nil)
	if f.g.shared != nil {
		t.Fatal("the guard took the shared lock after HoldBake")
	}
	// 올라간 이름은 manifest 하나 · 내용은 build 칸
	var m contract.BuildManifest
	body, _ := f.m.blob(contract.ArtifactManifest)
	if err := json.Unmarshal(body, &m); err != nil || len(f.m.uploaded()) != 1 {
		t.Fatalf("uploaded %v manifest %s %v", f.m.uploaded(), body, err)
	}
	if !reflect.DeepEqual(&m, res.Build) || m.Head != headA || *m.IR != bakeIR ||
		!reflect.DeepEqual(m.HeadTags, []string{bakeIR, "zz-other"}) || m.Sync.Command != syncCmd || len(m.Builds) != 2 {
		t.Fatalf("manifest = %+v\nresult build = %+v", m, res.Build)
	}
	// 단계 로그의 머리와 끝 줄
	log := f.m.logOf("build")
	for _, line := range []string{"bake: sync started\n", "bake: sync exited 0 after 0s\n",
		"bake: ir ir-1 matches HEAD " + headA + " (tags at HEAD: ir-1 zz-other)\n",
		"bake: build config-a started\n", "bake: build config-a exited 0 after 0s\n",
		"bake: build config-b exited 0 after 0s\n", pendingLine + "\n", "fetched\n", "built a\n"} {
		if !strings.Contains(log, line) {
			t.Fatalf("the step log misses %q:\n%s", line, log)
		}
	}
	if !f.nodeLogHas("bake: holding the lower for run r1") {
		t.Fatalf("node log = %s", f.nodeLog)
	}
}

// 마감 직전에 pending 을 쓴 build 는 finalize ok 인 DONE 이다 — after 가 late false 를 말하고 pending 과 HoldBake 가
// 살아 있다 (계획 4.1 3번).
func TestBakeBuild_PendingJustBeforeTheDeadlineIsOnTime(t *testing.T) {
	f := newBakeFixture(t)
	f.w.budgets = func(*Step) (time.Duration, time.Duration) { return time.Second, time.Minute }
	f.rt.finalize = func(ctx context.Context) (FinalizeResult, error) {
		dl, _ := ctx.Deadline()
		time.Sleep(time.Until(dl) - 300*time.Millisecond)
		return FinalizeResult{}, nil
	}
	res := f.buildOK(t, "r1")
	if res.Finalize != contract.StageOK || res.Error != "" || f.guardBake() != "r1" {
		t.Fatalf("result = %+v", res)
	}
}

// repo 모양 — pinned 의 argv · 닫은 뒤 대기 자리에서 읽은 sha256 이 초안과 manifest 에 든다.
func TestBakeBuild_TheRepoShapePinsTheManifest(t *testing.T) {
	f := newBakeFixture(t)
	f.rt.answer(probeScript, probeRepo())
	f.rt.answer(pinCommand, pinWrites("<manifest/>\n"))
	f.w.execute(context.Background(), buildStepOf("r1"))
	res := f.m.only(t)
	if got := f.rt.scripts(); !reflect.DeepEqual(got, []string{bashCheck, syncCmd, probeScript, pinCommand, buildA, buildB}) {
		t.Fatalf("commands = %q", got)
	}
	sum := sha256.Sum256([]byte("<manifest/>\n"))
	want := hex.EncodeToString(sum[:])
	if res.Error != "" || res.Build.Pinned == nil || res.Build.Pinned.SHA256 != want || res.Build.Pinned.File != pinnedFile {
		t.Fatalf("result = %+v build %+v", res, res.Build)
	}
	d, err := readDraft(filepath.Dir(f.state(t).PendingUpper), os.Getuid())
	if err != nil || d.Source.Pinned.SHA256 != want || d.Source.RepoID != CanonicalRepoID("https://git.example/manifest.git")+"#stable" {
		t.Fatalf("draft = %+v %v", d.Source, err)
	}
	if !strings.Contains(f.m.logOf("build"), pinnedLine+"\n") {
		t.Fatalf("step log = %s", f.m.logOf("build"))
	}
}

// builds 가 pinned 파일을 symlink 나 FIFO 로 바꾸면 제한 시간 안에 FAILED cannot pin the manifest · 몸통 · committed 다.
// 호스트 파일의 내용을 읽지 않는다 (계획 4.1 30번).
func TestBakeBuild_APinnedFileThatIsNotRegularFails(t *testing.T) {
	host := filepath.Join(t.TempDir(), "host-secret")
	if err := os.WriteFile(host, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, swap := range map[string]func(p string) error{
		"symlink": func(p string) error { _ = os.Remove(p); return os.Symlink(host, p) },
		"fifo":    func(p string) error { _ = os.Remove(p); return syscall.Mkfifo(p, 0o600) },
	} {
		t.Run(name, func(t *testing.T) {
			f := newBakeFixture(t)
			f.rt.answer(probeScript, probeRepo())
			f.rt.answer(pinCommand, pinWrites("<manifest/>\n"))
			f.rt.answer(buildB, fakeAnswer{do: func(upper string) {
				if err := swap(filepath.Join(upper, pinnedFile)); err != nil {
					t.Error(err)
				}
			}})
			var done atomic.Bool
			go func() {
				f.w.execute(context.Background(), buildStepOf("r1"))
				done.Store(true)
			}()
			waitFor(t, done.Load, "the build step did not return")
			res := f.m.only(t)
			if !strings.Contains(res.Error, "cannot pin the manifest: ") || res.ExitCode == nil || *res.ExitCode != 0 ||
				res.Finalize != contract.StageError {
				t.Fatalf("result = %+v", res)
			}
			st := f.state(t)
			if st.Phase != lower.PhaseCommitted || !strings.HasPrefix(st.LastAttempt.Reason, "cannot pin the manifest: ") {
				t.Fatalf("state = %+v %+v", st, st.LastAttempt)
			}
			if f.guardBake() != "" || f.held() != nil {
				t.Fatal("the bake was left held")
			}
		})
	}
}

// 단계 로그와 진행 청크에 host 경로 (대기 자리 · scratch) 가 없다 — 노드 로그에만 있다 (FD 규칙 16.2 끝).
func TestBakeBuild_TheStepLogCarriesNoHostPath(t *testing.T) {
	for name, fail := range map[string]bool{"success": false, "a failed build": true} {
		t.Run(name, func(t *testing.T) {
			f := newBakeFixture(t)
			if fail {
				f.rt.answer(buildA, fakeAnswer{exit: 2})
			}
			f.succeed()
			f.w.execute(context.Background(), buildStepOf("r1"))
			if log, progress := f.m.logOf("build"), f.m.progressText(); strings.Contains(log, f.scratch) ||
				strings.Contains(progress, f.scratch) || log == "" || progress == "" {
				t.Fatalf("a host path reached the step log:\n%s\n---\n%s", log, progress)
			}
			if !f.nodeLogHas(f.scratch) {
				t.Fatalf("the node log does not say where the pending directory is: %s", f.nodeLog)
			}
		})
	}
}

// 실패 (FD 규칙 6절 표 · 줄마다 부분 시험 하나)

// failCase 는 한 줄이다. want 는 결과 · state · 단계 로그를 본다.
type failCase struct {
	name  string
	setup func(t *testing.T, f *bakeFixture)
	check func(t *testing.T, f *bakeFixture, res Result)
}

// endedCommitted 는 합칠 upper 를 남기지 않고 끝난 모양이다 — committed · last_attempt · 대기 자리가 trash ·
// held 와 guard 의 굽기가 빈다 · 다음 굽기가 거절되지 않는다.
func endedCommitted(t *testing.T, f *bakeFixture, reason string) lower.State {
	t.Helper()
	st := f.state(t)
	if st.Phase != lower.PhaseCommitted || st.LastAttempt == nil || st.LastAttempt.Run != "r1" ||
		!strings.HasPrefix(st.LastAttempt.Reason, reason) {
		t.Fatalf("state = %+v last_attempt %+v; want %q", st, st.LastAttempt, reason)
	}
	if f.held() != nil || f.guardBake() != "" {
		t.Fatal("the bake is still held")
	}
	if ents := entries(t, filepath.Join(f.scratch, pendingDirName, f.key(t))); len(ents) != 0 {
		t.Fatalf("a pending directory stayed: %v", ents)
	}
	return st
}

func noExit(t *testing.T, f *bakeFixture, res Result) {
	t.Helper()
	if res.ExitCode != nil || len(f.m.exits()) != 0 || res.ExitedAt != nil {
		t.Fatalf("an interrupted step reported an exit: %+v %+v", res, f.m.exits())
	}
}

func exitOf(t *testing.T, f *bakeFixture, res Result, code int) {
	t.Helper()
	ex := f.m.exits()
	if res.ExitCode == nil || *res.ExitCode != code || len(ex) != 1 || *ex[0].Outcome.Code != code {
		t.Fatalf("exit_code %v exited %+v; want %d", res.ExitCode, ex, code)
	}
}

func TestBakeBuild_Failures(t *testing.T) {
	cases := []failCase{
		{"bash check exit 1", func(t *testing.T, f *bakeFixture) {
			f.rt.answer(bashCheck, fakeAnswer{exit: 1})
		}, func(t *testing.T, f *bakeFixture, res Result) {
			want := "cannot run the bake commands: the prepared rootfs has no bash, or sh cannot run there (exit 1)"
			if res.Error != want || res.Build != nil {
				t.Fatalf("result = %+v", res)
			}
			noExit(t, f, res)
			endedCommitted(t, f, want)
			if got := f.rt.scripts(); !reflect.DeepEqual(got, []string{bashCheck}) {
				t.Fatalf("a contract command ran: %q", got)
			}
		}},
		{"bash check exit 127", func(t *testing.T, f *bakeFixture) {
			f.rt.answer(bashCheck, fakeAnswer{exit: 127})
		}, func(t *testing.T, f *bakeFixture, res Result) {
			if !strings.HasSuffix(res.Error, "(exit 127)") {
				t.Fatalf("result = %+v", res)
			}
			noExit(t, f, res)
		}},
		{"bash check session error", func(t *testing.T, f *bakeFixture) {
			f.rt.answer(bashCheck, fakeAnswer{exit: -1, err: errors.New("runtime run: EOF")})
		}, func(t *testing.T, f *bakeFixture, res Result) {
			if res.Error != "runtime run: EOF" {
				t.Fatalf("result = %+v", res)
			}
			noExit(t, f, res)
			endedCommitted(t, f, "runtime run: EOF")
		}},
		{"the lease ends during the bash check", func(t *testing.T, f *bakeFixture) {
			f.rt.answer(bashCheck, fakeAnswer{wait: true})
			go func() {
				waitFor(t, func() bool { return len(f.rt.ran()) == 1 }, "the check did not start")
				f.w.Held.Set(nil)
			}()
		}, func(t *testing.T, f *bakeFixture, res Result) {
			if res.Error != "aborted: lease expired" {
				t.Fatalf("result = %+v", res)
			}
			noExit(t, f, res)
			endedCommitted(t, f, "aborted: lease expired")
		}},
		{"the session does not open", func(t *testing.T, f *bakeFixture) {
			f.rt.openErr = errors.New("prepared rootfs target /work is missing")
		}, func(t *testing.T, f *bakeFixture, res Result) {
			if res.Error != "runtime open: prepared rootfs target /work is missing" {
				t.Fatalf("result = %+v", res)
			}
			noExit(t, f, res)
			endedCommitted(t, f, "runtime open: ")
		}},
		{"sync fails", func(t *testing.T, f *bakeFixture) {
			f.rt.answer(syncCmd, fakeAnswer{exit: 1, stderr: "fatal: no such ref\n"})
		}, func(t *testing.T, f *bakeFixture, res Result) {
			if res.Error != "" || res.Reason != "" || res.Produced != nil {
				t.Fatalf("result = %+v", res)
			}
			exitOf(t, f, res, 1)
			if res.Build == nil || res.Build.Sync.ExitCode != 1 || len(res.Build.Builds) != 0 || res.Build.Head != "" ||
				res.Build.HeadTags != nil {
				t.Fatalf("build = %+v", res.Build)
			}
			st := endedCommitted(t, f, "sync exited 1")
			if len(st.LastAttempt.Builds) != 0 {
				t.Fatalf("builds = %+v", st.LastAttempt.Builds)
			}
			log := f.m.logOf("build")
			for _, line := range []string{"bake: skipping config-a; sync failed\n", "bake: skipping config-b; sync failed\n",
				"fatal: no such ref\n"} {
				if !strings.Contains(log, line) {
					t.Fatalf("step log misses %q:\n%s", line, log)
				}
			}
			if got := f.rt.scripts(); len(got) != 2 {
				t.Fatalf("commands after a failed sync: %q", got)
			}
			if len(f.m.uploaded()) != 0 {
				t.Fatalf("uploaded %v", f.m.uploaded())
			}
		}},
		{"a build fails and the rest are skipped", func(t *testing.T, f *bakeFixture) {
			f.succeed()
			f.rt.answer(buildA, fakeAnswer{exit: 2})
		}, func(t *testing.T, f *bakeFixture, res Result) {
			if res.Error != "" {
				t.Fatalf("result = %+v", res)
			}
			exitOf(t, f, res, 2)
			st := endedCommitted(t, f, "build config-a exited 2")
			if len(st.LastAttempt.Builds) != 1 || st.LastAttempt.Builds[0].ExitCode != 2 {
				t.Fatalf("builds = %+v", st.LastAttempt.Builds)
			}
			if !strings.Contains(f.m.logOf("build"), "bake: skipping config-b; config-a failed\n") {
				t.Fatalf("step log = %s", f.m.logOf("build"))
			}
			if got := f.rt.scripts(); got[len(got)-1] != buildA {
				t.Fatalf("a build ran after the first failure: %q", got)
			}
		}},
		{"ir is not local", func(t *testing.T, f *bakeFixture) {
			f.rt.answer(probeScript, fakeAnswer{stdout: "mode=git\nhead=" + headA + "\ntagged=\nurl=\nbranch=main\n"})
		}, func(t *testing.T, f *bakeFixture, res Result) {
			irMismatch(t, f, res, "bake: ir ir-1 is not in the local repository after sync; the sync command must "+
				"fetch that tag (HEAD is "+headA+", tags at HEAD: none)", []string{})
		}},
		{"ir points elsewhere", func(t *testing.T, f *bakeFixture) {
			f.rt.answer(probeScript, fakeAnswer{stdout: "mode=git\nhead=" + headA + "\ntagged=" + headB +
				"\ntag=other\nurl=\nbranch=\n"})
		}, func(t *testing.T, f *bakeFixture, res Result) {
			irMismatch(t, f, res, "bake: ir ir-1 points at "+headB+", but sync left HEAD at "+headA+
				" (tags at HEAD: other)", []string{"other"})
		}},
		{"ir cannot be verified", func(t *testing.T, f *bakeFixture) {
			f.rt.answer(probeScript, fakeAnswer{stdout: "mode=none\n"})
		}, func(t *testing.T, f *bakeFixture, res Result) {
			want := "cannot verify ir: the workspace has neither .repo/manifests nor .git"
			if res.Error != want || res.Reason != "" || res.Build.Head != "" || res.Build.HeadTags != nil {
				t.Fatalf("result = %+v build %+v", res, res.Build)
			}
			exitOf(t, f, res, 0)
			endedCommitted(t, f, want)
		}},
		{"pinning fails", func(t *testing.T, f *bakeFixture) {
			f.rt.answer(probeScript, probeRepo())
			f.rt.answer(pinCommand, fakeAnswer{exit: 1, stderr: "error: manifest is broken\n"})
		}, func(t *testing.T, f *bakeFixture, res Result) {
			want := "cannot pin the manifest: repo manifest -r exited 1: error: manifest is broken"
			if res.Error != want {
				t.Fatalf("result = %+v", res)
			}
			exitOf(t, f, res, 0)
			endedCommitted(t, f, want)
			if got := f.rt.scripts(); got[len(got)-1] != pinCommand {
				t.Fatalf("a build ran after a failed pin: %q", got)
			}
		}},
		{"the lease ends during sync", func(t *testing.T, f *bakeFixture) {
			f.rt.answer(syncCmd, fakeAnswer{wait: true})
			go func() {
				waitFor(t, func() bool { return len(f.rt.ran()) == 2 }, "sync did not start")
				f.w.Held.Set(nil)
			}()
		}, func(t *testing.T, f *bakeFixture, res Result) {
			if res.Error != "aborted: lease expired" || res.Build != nil {
				t.Fatalf("result = %+v", res)
			}
			noExit(t, f, res)
			endedCommitted(t, f, "aborted: lease expired")
		}},
		{"the helper dies during a build", func(t *testing.T, f *bakeFixture) {
			f.succeed()
			f.rt.answer(buildA, fakeAnswer{exit: -1, err: errors.New("runtime run: unexpected EOF")})
		}, func(t *testing.T, f *bakeFixture, res Result) {
			if res.Error != "runtime run: unexpected EOF" || res.Build == nil || res.Build.Sync.ExitCode != 0 {
				t.Fatalf("result = %+v", res)
			}
			noExit(t, f, res)
			endedCommitted(t, f, "runtime run: unexpected EOF")
		}},
		{"the upper cannot be kept", func(t *testing.T, f *bakeFixture) {
			f.succeed()
			f.rt.answer(buildB, fakeAnswer{after: func() {
				if err := os.Mkdir(f.state(t).PendingUpper, 0o700); err != nil {
					t.Error(err)
				}
			}})
		}, func(t *testing.T, f *bakeFixture, res Result) {
			if !strings.HasPrefix(res.Error, "runtime cleanup: keep the upper: ") || res.Finalize != contract.StageError {
				t.Fatalf("result = %+v", res)
			}
			exitOf(t, f, res, 0)
			endedCommitted(t, f, "runtime cleanup: keep the upper: ")
		}},
		{"finalize runs past the deadline before pending", func(t *testing.T, f *bakeFixture) {
			f.succeed()
			f.w.budgets = func(*Step) (time.Duration, time.Duration) { return 50 * time.Millisecond, time.Minute }
			f.rt.finalize = func(ctx context.Context) (FinalizeResult, error) {
				dl, _ := ctx.Deadline()
				time.Sleep(time.Until(dl) + 20*time.Millisecond)
				return FinalizeResult{}, nil
			}
		}, func(t *testing.T, f *bakeFixture, res Result) {
			if res.Reason != contract.ReasonFinalizeTimeout || res.Finalize != contract.StageTimeout {
				t.Fatalf("result = %+v", res)
			}
			exitOf(t, f, res, 0)
			endedCommitted(t, f, contract.ReasonFinalizeTimeout)
		}},
		{"finalize is cut at the deadline", func(t *testing.T, f *bakeFixture) {
			f.succeed()
			f.w.budgets = func(*Step) (time.Duration, time.Duration) { return 50 * time.Millisecond, time.Minute }
			f.rt.finalize = func(ctx context.Context) (FinalizeResult, error) {
				<-ctx.Done()
				return FinalizeResult{}, ctx.Err()
			}
		}, func(t *testing.T, f *bakeFixture, res Result) {
			if res.Reason != contract.ReasonFinalizeTimeout {
				t.Fatalf("result = %+v", res)
			}
			endedCommitted(t, f, contract.ReasonFinalizeTimeout)
		}},
		{"pending cannot be written", func(t *testing.T, f *bakeFixture) {
			f.succeed()
			was := writeLowerState
			writeLowerState = func(b *lower.Bake, st lower.State) error {
				if st.Phase == lower.PhasePending {
					return errors.New("injected: no space left on device")
				}
				return was(b, st)
			}
			t.Cleanup(func() { writeLowerState = was })
		}, func(t *testing.T, f *bakeFixture, res Result) {
			want := "cannot write the lower state: injected: no space left on device"
			if res.Error != want || res.Finalize != contract.StageError {
				t.Fatalf("result = %+v", res)
			}
			exitOf(t, f, res, 0)
			endedCommitted(t, f, want)
			writeLowerState = func(b *lower.Bake, st lower.State) error { return b.WriteState(st) }
		}},
		{"the draft cannot be written", func(t *testing.T, f *bakeFixture) {
			f.succeed()
			f.rt.answer(buildB, fakeAnswer{after: func() {
				// bake.json 자리를 디렉터리가 쥐었다 — root 도 rename 이 실패한다
				if err := os.Mkdir(filepath.Join(filepath.Dir(f.state(t).PendingUpper), draftName), 0o700); err != nil {
					t.Error(err)
				}
			}})
		}, func(t *testing.T, f *bakeFixture, res Result) {
			if !strings.HasPrefix(res.Error, "cannot write the bake draft: ") {
				t.Fatalf("result = %+v", res)
			}
			exitOf(t, f, res, 0)
			endedCommitted(t, f, "cannot write the bake draft: ")
		}},
		{"the upload budget runs out after pending", func(t *testing.T, f *bakeFixture) {
			f.succeed()
			f.m.putHold[contract.ArtifactManifest] = true
			f.w.budgets = func(*Step) (time.Duration, time.Duration) { return time.Minute, 200 * time.Millisecond }
		}, func(t *testing.T, f *bakeFixture, res Result) {
			if res.Reason != contract.ReasonUploadTimeout || res.Upload != contract.StageTimeout {
				t.Fatalf("result = %+v", res)
			}
			exitOf(t, f, res, 0)
			endedCommitted(t, f, contract.ReasonUploadTimeout)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newBakeFixture(t)
			tc.setup(t, f)
			f.w.execute(context.Background(), buildStepOf("r1"))
			tc.check(t, f, f.m.only(t))
			// 다음 굽기가 bake_in_progress 로 거절되지 않는다
			if tc.name != "the upload budget runs out after pending" {
				assertNextBakeRuns(t, f)
			}
		})
	}
}

// irMismatch 는 IR 어긋남의 모양이다 — DONE · exit 0 (sync) · reason ir_mismatch · 구성마다 skipping 줄 · 문장은 단계
// 로그 끝 · head · head_tags · manifest 없음 · last_attempt ir_mismatch (물음 1 답 B · 결정 22 · 55).
func irMismatch(t *testing.T, f *bakeFixture, res Result, sentence string, tags []string) {
	t.Helper()
	if res.Error != "" || res.Reason != contract.ReasonIRMismatch || res.Produced != nil {
		t.Fatalf("result = %+v", res)
	}
	exitOf(t, f, res, 0)
	if res.Build.Head != headA || res.Build.IR != nil || !reflect.DeepEqual(res.Build.HeadTags, tags) ||
		len(res.Build.Builds) != 0 {
		t.Fatalf("build = %+v", res.Build)
	}
	endedCommitted(t, f, contract.ReasonIRMismatch)
	log := f.m.logOf("build")
	for _, line := range []string{"bake: skipping config-a; ir ir-1 did not match\n",
		"bake: skipping config-b; ir ir-1 did not match\n", sentence + "\n"} {
		if !strings.Contains(log, line) {
			t.Fatalf("step log misses %q:\n%s", line, log)
		}
	}
	if strings.Index(log, sentence) < strings.Index(log, "skipping config-b") {
		t.Fatalf("the sentence is not after the skipping lines:\n%s", log)
	}
}

// assertNextBakeRuns 는 held 와 guard 의 굽기가 비어 다음 굽기가 bake_in_progress 로 거절되지 않는 것을 본다.
func assertNextBakeRuns(t *testing.T, f *bakeFixture) {
	t.Helper()
	f.rt.mu.Lock()
	f.rt.answers = map[string]fakeAnswer{probeScript: probeMatch()}
	f.rt.openErr, f.rt.finalize = nil, nil
	f.rt.mu.Unlock()
	f.w.budgets = nil
	f.w.Held.Add(Lease{RunID: "r2", Node: f.w.Ident.NodeID, NotAfter: time.Now().Add(time.Hour)})
	next := buildStepOf("r2")
	f.w.execute(context.Background(), next)
	if res := f.m.lastResult(t); res.Reason == contract.ReasonBakeInProgress || res.Error != "" {
		t.Fatalf("the next bake = %+v", res)
	}
}

// 명령 중 데몬이 멈추면 보고하지 않는다 — last_attempt 는 node stopped (FD 규칙 6절).
func TestBakeBuild_TheNodeStopsDuringACommand(t *testing.T) {
	f := newBakeFixture(t)
	f.rt.answer(syncCmd, fakeAnswer{wait: true})
	ctx, stop := context.WithCancel(context.Background())
	go func() {
		waitFor(t, func() bool { return len(f.rt.ran()) == 2 }, "sync did not start")
		stop()
	}()
	f.w.execute(ctx, buildStepOf("r1"))
	if n := f.m.reports(); n != 0 {
		t.Fatalf("a stopped node reported %d times", n)
	}
	endedCommitted(t, f, "node stopped")
}

// held 가 있는 동안의 오류 길 (결정 44) — 대기 자리 만들기 · building 쓰기가 실패해도 몸통이 돌고 held 와 g.bake 가
// 빈다. 다음 굽기가 bake_in_progress 로 거절되지 않는다.
func TestBakeBuild_ErrorsWhileHeldRunTheBody(t *testing.T) {
	t.Run("the pending directory", func(t *testing.T) {
		f := newBakeFixture(t)
		if err := os.WriteFile(filepath.Join(f.scratch, pendingDirName), nil, 0o600); err != nil {
			t.Fatal(err)
		}
		f.w.execute(context.Background(), buildStepOf("r1"))
		res := f.m.only(t)
		if !strings.HasPrefix(res.Error, "cannot create the pending upper directory: ") {
			t.Fatalf("result = %+v", res)
		}
		st := f.state(t)
		if st.Phase != lower.PhaseCommitted || !strings.HasPrefix(st.LastAttempt.Reason, "cannot create the pending") ||
			f.held() != nil {
			t.Fatalf("state = %+v", st)
		}
		if err := os.Remove(filepath.Join(f.scratch, pendingDirName)); err != nil {
			t.Fatal(err)
		}
		assertNextBakeRuns(t, f)
	})
	t.Run("the building state", func(t *testing.T) {
		f := newBakeFixture(t)
		was := writeLowerState
		writeLowerState = func(b *lower.Bake, st lower.State) error {
			if st.Phase == lower.PhaseBuilding {
				return errors.New("injected: read-only file system")
			}
			return was(b, st)
		}
		f.w.execute(context.Background(), buildStepOf("r1"))
		writeLowerState = was
		if res := f.m.only(t); res.Error != "cannot write the lower state: injected: read-only file system" {
			t.Fatalf("result = %+v", res)
		}
		if f.held() != nil || f.guardBake() != "" {
			t.Fatal("the bake is still held")
		}
		assertNextBakeRuns(t, f)
	})
}

// 계약의 명령은 격리 실행 환경 안에서만 (계획 3.1 · Step 16)

// 받은 argv 의 차례와 모양 — 계약의 명령은 bash -c · 노드의 고정 한 줄은 sh -c (pinned 만 bash) · 작업 폴더는 세션의
// 워크스페이스 자리다. repo 모양은 pinned 가 대조 뒤 · builds 앞이다.
func TestBakeBuild_CommandsGoThroughTheSession(t *testing.T) {
	for name, repo := range map[string]bool{"git": false, "repo": true} {
		t.Run(name, func(t *testing.T) {
			f := newBakeFixture(t)
			want := [][]string{{"sh", "-c", bashCheck}, {"bash", "-c", syncCmd}, {"sh", "-c", probeScript}}
			if repo {
				f.rt.answer(probeScript, probeRepo())
				f.rt.answer(pinCommand, pinWrites("<m/>"))
				want = append(want, []string{"bash", "-c", pinCommand})
			} else {
				f.succeed()
			}
			want = append(want, []string{"bash", "-c", buildA}, []string{"bash", "-c", buildB})
			f.w.execute(context.Background(), buildStepOf("r1"))
			if res := f.m.only(t); res.Error != "" {
				t.Fatalf("result = %+v", res)
			}
			ran := f.rt.ran()
			if len(ran) != len(want) {
				t.Fatalf("ran %d commands, want %d: %q", len(ran), len(want), f.rt.scripts())
			}
			for i, spec := range ran {
				if !reflect.DeepEqual(spec.Argv, want[i]) || spec.Dir != "/work" {
					t.Fatalf("command %d = %q in %q, want %q in /work", i, spec.Argv, spec.Dir, want[i])
				}
			}
		})
	}
}

// 계약이 env 로 부르지 않은 호스트 환경 변수는 명령에 닿지 않는다. 계약이 부른 이름은 닿는다 — 노드 비밀의 이름도
// 부르면 닿는 것은 잔여다 (계획 8절 · CG 물음 2 답).
func TestBakeBuild_UnnamedHostVariablesDoNotReachTheCommands(t *testing.T) {
	t.Setenv("ENODE_BAKE_HOST_SECRET", "must-not-leak")
	t.Setenv("CROSS_COMPILE", "aarch64-linux-gnu-")
	t.Setenv("ENODE_TOKEN", "node-token")
	t.Setenv("CONTRACT_NAMED", "named")
	f := newBakeFixture(t)
	f.succeed()
	step := buildStepOf("r1")
	step.Env = []string{"CONTRACT_NAMED", "ENODE_TOKEN"}
	f.w.execute(context.Background(), step)
	for _, spec := range f.rt.ran() {
		env := strings.Join(spec.Env, "\n")
		if strings.Contains(env, "must-not-leak") {
			t.Fatalf("an unnamed host variable reached %q", spec.Argv)
		}
		for _, want := range []string{"CROSS_COMPILE=aarch64-linux-gnu-", "CONTRACT_NAMED=named", "ENODE_TOKEN=node-token",
			"OUT=" + runtimeOutTarget, "IN=" + runtimeInTarget, contract.EnvIR + "=" + bakeIR} {
			if !strings.Contains(env+"\n", want+"\n") {
				t.Fatalf("%q misses %s in its environment:\n%s", spec.Argv, want, env)
			}
		}
	}
}

// IR 은 환경 변수 ENODE_IR 로만 간다 — 셸 글자에 넣지 않는다. 받은 argv 어디에도 IR 글자가 없다.
func TestIRProbe_TheIRGoesOnlyThroughTheEnvironment(t *testing.T) {
	f := newBakeFixture(t)
	f.succeed()
	step := buildStepOf("r1")
	step.IR = "rel-2026.09-distinct"
	f.w.execute(context.Background(), step)
	probes := 0
	for _, spec := range f.rt.ran() {
		if spec.Argv[len(spec.Argv)-1] != probeScript {
			continue
		}
		probes++
		for _, a := range spec.Argv {
			if strings.Contains(a, step.IR) {
				t.Fatalf("the IR reached the shell text: %q", spec.Argv)
			}
		}
		if !strings.Contains(strings.Join(spec.Env, "\n")+"\n", contract.EnvIR+"="+step.IR+"\n") {
			t.Fatalf("the probe did not get %s: %q", contract.EnvIR, spec.Env)
		}
	}
	if probes != 1 {
		t.Fatalf("the probe ran %d times", probes)
	}
}

// build 의 $OUT 은 노드의 것이다 (계획 4.1 5번) — 명령이 세션의 $OUT 에 쓴 것은 올리지 않는다. 실패면 명령이 쓴
// manifest 가 produced 에 없고, 성공이면 올라간 이름은 노드의 manifest 하나이고 내용은 노드의 build 칸이다.
func TestBakeBuild_TheCommandsOutIsNotUploaded(t *testing.T) {
	forge := func(out string) {
		_ = os.WriteFile(filepath.Join(out, contract.ArtifactManifest), []byte(`{"forged":true}`), 0o644)
		_ = os.WriteFile(filepath.Join(out, "other"), []byte("x"), 0o644)
	}
	t.Run("a failed build", func(t *testing.T) {
		f := newBakeFixture(t)
		f.rt.answer(syncCmd, fakeAnswer{exit: 1, doOut: forge})
		f.w.execute(context.Background(), buildStepOf("r1"))
		if res := f.m.only(t); res.Produced != nil || len(f.m.uploaded()) != 0 {
			t.Fatalf("produced %v uploaded %v", res.Produced, f.m.uploaded())
		}
	})
	t.Run("a successful build", func(t *testing.T) {
		f := newBakeFixture(t)
		f.succeed()
		f.rt.answer(buildB, fakeAnswer{doOut: forge})
		f.w.execute(context.Background(), buildStepOf("r1"))
		res := f.m.only(t)
		if !reflect.DeepEqual(res.Produced, []string{contract.ArtifactManifest}) || len(f.m.uploaded()) != 1 {
			t.Fatalf("produced %v uploaded %v", res.Produced, f.m.uploaded())
		}
		body, _ := f.m.blob(contract.ArtifactManifest)
		var m contract.BuildManifest
		if err := json.Unmarshal(body, &m); err != nil || strings.Contains(string(body), "forged") || m.Head != headA {
			t.Fatalf("the uploaded manifest = %s %v", body, err)
		}
	})
}
