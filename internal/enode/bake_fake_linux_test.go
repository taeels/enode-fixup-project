//go:build linux

package enode

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/taeels/enode/internal/contract"
	execenv "github.com/taeels/enode/internal/environment"
	"github.com/taeels/enode/internal/lower"
	"golang.org/x/sys/unix"
)

// 굽기 시험의 공용 틀 (bake 유닛 · 계획 4.1 23번). 가짜 런타임 (isolated) · 임시 폴더의 진짜 lower 자리와 잠금 ·
// 가짜 Mediator · 잠금 핸들을 t.Cleanup 으로 놓는 도우미다. namespace 없는 merge-helper 는 mergehelper_linux_test.go.

// fakeAnswer 는 가짜 세션이 argv 하나에 내는 답이다.
type fakeAnswer struct {
	exit   int
	stdout string
	stderr string
	err    error              // exit 가 음수일 때의 오류
	wait   bool               // ctx 가 끝날 때까지 돌아오지 않는다 (임대 · 데몬)
	do     func(upper string) // 명령이 upper 에 쓰는 일 (pinned 파일 · symlink · FIFO)
	doOut  func(out string)   // 명령이 세션의 $OUT (호스트 쪽 폴더) 에 쓰는 일
	after  func()             // 답한 뒤에 부른다 (시험이 흐름 사이에 끼운다)
}

// fakeRuntime 은 isolated 를 광고하는 가짜 런타임이다. 받은 ProcessSpec 을 모두 적고, 세션마다 upper 로 쓸 임시
// 폴더를 만들어 Close(Keep{Upper}) 에서 rename 한다.
type fakeRuntime struct {
	scratch string

	mu       sync.Mutex
	answers  map[string]fakeAnswer // argv 의 마지막 칸 (셸에 넘긴 글) 이 열쇠다
	specs    []ProcessSpec
	keeps    []Keep
	openErr  error
	closeErr error
	finalize func(context.Context) (FinalizeResult, error)
	uppers   []string
}

func newFakeRuntime(scratchDir string) *fakeRuntime {
	return &fakeRuntime{scratch: scratchDir, answers: map[string]fakeAnswer{}}
}

func (r *fakeRuntime) Capability() RuntimeCapability {
	return RuntimeCapability{Writes: writesIsolated}
}

func (r *fakeRuntime) answer(script string, a fakeAnswer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.answers[script] = a
}

func (r *fakeRuntime) ran() []ProcessSpec {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]ProcessSpec(nil), r.specs...)
}

// scripts 는 돈 argv 의 마지막 칸들이다 — 명령의 차례를 본다.
func (r *fakeRuntime) scripts() []string {
	var out []string
	for _, s := range r.ran() {
		out = append(out, s.Argv[len(s.Argv)-1])
	}
	return out
}

func (r *fakeRuntime) Open(_ context.Context, spec RuntimeSpec) (StepSession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.openErr != nil {
		return nil, r.openErr
	}
	root, err := os.MkdirTemp(r.scratch, runcSessionPrefix)
	if err != nil {
		return nil, err
	}
	if err := os.Mkdir(filepath.Join(root, "upper"), 0o755); err != nil {
		return nil, err
	}
	r.uppers = append(r.uppers, filepath.Join(root, "upper"))
	return &fakeSession{r: r, root: root, out: spec.Out, record: &execenv.Record{Runtime: "runc-overlay"}}, nil
}

type fakeSession struct {
	r      *fakeRuntime
	root   string
	out    string
	record *execenv.Record
}

func (s *fakeSession) Paths() RuntimePaths {
	return RuntimePaths{Dir: "/work", In: runtimeInTarget, Out: runtimeOutTarget}
}

func (s *fakeSession) Project(context.Context, FrameworkProjectionSpec) (FrameworkProjection, error) {
	return FrameworkProjection{}, nil
}

func (s *fakeSession) Run(ctx context.Context, spec ProcessSpec) (int, error) {
	s.r.mu.Lock()
	s.r.specs = append(s.r.specs, spec)
	a, ok := s.r.answers[spec.Argv[len(spec.Argv)-1]]
	s.r.mu.Unlock()
	if !ok {
		return 0, nil
	}
	if a.after != nil {
		defer a.after()
	}
	if a.do != nil {
		a.do(filepath.Join(s.root, "upper"))
	}
	if a.doOut != nil {
		a.doOut(s.out)
	}
	if a.wait {
		<-ctx.Done()
		return -1, fmt.Errorf("runtime run: %w", ctx.Err())
	}
	if spec.Stdout != nil && a.stdout != "" {
		_, _ = spec.Stdout.Write([]byte(a.stdout))
	}
	if spec.Stderr != nil && a.stderr != "" {
		_, _ = spec.Stderr.Write([]byte(a.stderr))
	}
	if a.exit < 0 {
		return -1, a.err
	}
	return a.exit, nil
}

func (s *fakeSession) Finalize(ctx context.Context, _ FinalizeSpec) (FinalizeResult, error) {
	s.r.mu.Lock()
	f := s.r.finalize
	s.r.mu.Unlock()
	if f != nil {
		return f(ctx)
	}
	return FinalizeResult{}, nil
}

// Close 는 runc-overlay 처럼 Keep.Upper 로 upper 를 한 번 옮긴다 — 대상이 있으면 덮지 않는다.
func (s *fakeSession) Close(_ context.Context, keep Keep) error {
	s.r.mu.Lock()
	s.r.keeps = append(s.r.keeps, keep)
	closeErr := s.r.closeErr
	s.r.mu.Unlock()
	if closeErr != nil {
		_ = os.RemoveAll(s.root)
		return closeErr
	}
	var err error
	if keep.Upper != "" {
		err = unix.Renameat2(unix.AT_FDCWD, filepath.Join(s.root, "upper"), unix.AT_FDCWD, keep.Upper, unix.RENAME_NOREPLACE)
		if err != nil {
			err = fmt.Errorf("keep the upper: %w", err)
		}
	}
	_ = os.RemoveAll(s.root)
	return err
}

func (s *fakeSession) Environment() *execenv.Record { return s.record }

// bakeFixture 는 굽는 노드 하나다 — 진짜 lower 자리 · Baker · 가짜 런타임 · 가짜 Mediator 와 Worker.
type bakeFixture struct {
	guardFixture
	nodeWS  string // 노드 설정의 워크스페이스 글자 — 보통은 ws 와 같고 linked 면 ws 를 가리키는 symlink 다
	t       *testing.T
	scratch string
	nodeLog *lockedBuffer
	log     *slog.Logger
	ctx     context.Context
	stop    context.CancelFunc
	b       *Baker
	rt      *fakeRuntime
	m       *mediator
	w       *Worker
}

func newBakeFixture(t *testing.T) *bakeFixture {
	t.Helper()
	return newBakeFixtureAt(t, false)
}

// newBakeFixtureAt 의 linked 가 참이면 노드의 워크스페이스 (설정의 ws 글자) 가 lower 를 가리키는 symlink 다 — 조각 7
// 의 symlink 형제 모양. f.ws 는 늘 진짜 lower 다 (시험이 목록과 metadata 를 거기서 읽는다).
func newBakeFixtureAt(t *testing.T, linked bool) *bakeFixture {
	t.Helper()
	f := &bakeFixture{guardFixture: newGuardFixture(t), t: t, nodeLog: &lockedBuffer{}}
	f.nodeWS = f.ws
	if linked {
		f.nodeWS = filepath.Join(filepath.Dir(f.ws), "ws-link")
		if err := os.Symlink(f.ws, f.nodeWS); err != nil {
			t.Fatal(err)
		}
		f.g = newLowerGuard(f.lowers, f.nodeWS, Identity{NodeID: "node-a", Label: "box-a"},
			slog.New(slog.NewTextHandler(f.guardFixture.log, nil)))
		f.g.start()
	}
	f.scratch = filepath.Join(filepath.Dir(f.ws), "scratch")
	if err := os.Mkdir(f.scratch, 0o700); err != nil {
		t.Fatal(err)
	}
	f.log = slog.New(slog.NewTextHandler(f.nodeLog, nil))
	f.ctx, f.stop = context.WithCancel(context.Background())
	t.Cleanup(f.stop)
	f.b = f.startBaker()
	f.rt = newFakeRuntime(f.scratch)
	f.m, f.w = finalizeWorker(t)
	f.w.Log = f.log
	f.w.Runtime = f.rt
	f.w.RuntimeRecord = &execenv.Record{Runtime: "runc-overlay", PreparedEnvironment: "prep-1", WorkspaceTarget: "/work"}
	f.w.Guard = f.g
	f.w.Bake = f.b
	f.w.Local = Local{Workspace: f.nodeWS}
	return f
}

// startBaker 는 이 자리에 Baker 를 새로 짓는다 — 재시작 장면은 옛 굽기를 놓고 새 LowerGuard 와 Baker 로 짓는다.
// 끝나면 배경 일을 기다리고 Baker 가 쥔 굽기 잠금을 놓는다 (계획 5절 규칙 9).
func (f *bakeFixture) startBaker() *Baker {
	b := StartBaker(f.ctx, f.g, f.scratch, Identity{NodeID: "node-a", Label: "box-a"}, "inst-1", f.log)
	f.t.Cleanup(func() {
		f.stop()
		b.Wait()
		b.mu.Lock()
		h := b.held
		b.mu.Unlock()
		if h != nil {
			_ = h.lock.Release()
		}
	})
	return b
}

// key 는 이 lower 의 키다 — 늘 ReadRoot 로 얻는다 (계획 5절 「키 글자를 박지 않는다」).
func (f *bakeFixture) key(t *testing.T) string {
	t.Helper()
	root, err := lower.ReadRoot(f.ws)
	if err != nil {
		t.Fatal(err)
	}
	return root.Key.String()
}

// holdBakeLock 은 형제 Dir 로 bake.lock 을 쥔다 — 주인이 살아 있는 굽기를 흉내 낸다. 끝나면 놓는다.
func (f *bakeFixture) holdBakeLock(t *testing.T) *lower.Bake {
	t.Helper()
	lock, ok, err := f.sibling(t).TryBake()
	if err != nil || !ok {
		t.Fatalf("TryBake = %v, %v", ok, err)
	}
	t.Cleanup(func() { _ = lock.Release() })
	return lock
}

// state 는 지금 state.json 이다.
func (f *bakeFixture) state(t *testing.T) lower.State {
	t.Helper()
	st, err := f.sibling(t).ReadState()
	if err != nil {
		t.Fatal(err)
	}
	return st
}

// putState 는 형제처럼 굽기 잠금을 잠깐 쥐고 state.json 을 쓴다 — 낡은 상태를 흉내 낸다.
func (f *bakeFixture) putState(t *testing.T, st lower.State) {
	t.Helper()
	lock, ok, err := f.sibling(t).TryBake()
	if err != nil || !ok {
		t.Fatalf("TryBake = %v, %v", ok, err)
	}
	defer func() { _ = lock.Release() }()
	if st.Since.IsZero() {
		st.Since = time.Now().UTC()
	}
	if err := lock.WriteState(st); err != nil {
		t.Fatal(err)
	}
}

// pendingIn 은 scratch 에 대기 자리 하나를 만든다 — upper 안에 파일 하나 · 초안을 쓰면 builds 가 든다.
func (f *bakeFixture) pendingIn(t *testing.T, scratchDir, run string, draft bool) string {
	t.Helper()
	dir, err := makePending(scratchDir, f.key(t))
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "upper"), "built/out.bin", "out")
	if draft {
		d := fullDraft()
		d.Run = run
		if err := writeDraft(dir, d); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// guardBake 는 guard 가 쥔 굽기의 Run 이다 — HoldBake 부터 DropBake 까지.
func (f *bakeFixture) guardBake() string {
	f.g.mu.Lock()
	defer f.g.mu.Unlock()
	if f.g.bake == nil {
		return ""
	}
	return f.g.bake.run
}

// held 는 Baker 가 쥔 굽기다.
func (f *bakeFixture) held() *heldBake {
	f.b.mu.Lock()
	defer f.b.mu.Unlock()
	return f.b.held
}

// hold 는 build 단계가 pending 을 쓴 모양을 짓는다 — 굽기 잠금 · 대기 자리 · pending · HoldBake. merge 단계를
// 거치지 않고 몸통과 표지를 보는 시험이 쓴다.
func (f *bakeFixture) hold(t *testing.T, run string) *heldBake {
	t.Helper()
	dir := f.g.Dir()
	lock, ok, err := dir.TryBake()
	if err != nil || !ok {
		t.Fatalf("TryBake = %v, %v", ok, err)
	}
	t.Cleanup(func() { _ = lock.Release() })
	h := &heldBake{b: f.b, run: run, step: 1, dir: dir, lock: lock}
	h.setPending(f.pendingIn(t, f.scratch, run, true))
	d := fullDraft()
	d.Run = run
	h.setDraft(d)
	if err := lock.WriteState(lower.State{Phase: lower.PhasePending, Since: time.Now().UTC(),
		Owner: &lower.Owner{Run: run, Step: 1, Node: "node-a", Instance: "inst-1"}, PendingUpper: filepath.Join(h.pending, "upper")}); err != nil {
		t.Fatal(err)
	}
	f.b.mu.Lock()
	f.b.held = h
	f.b.mu.Unlock()
	f.g.HoldBake(run, func() { h.abandon(reasonBakeRunEnded, h.draftBuilds()) })
	return h
}

// failStateWrites 는 state.json 쓰기를 실패시킨다 — 자리 디렉터리를 0500 으로. root 는 막히지 않으므로 쓰기 함수
// 값을 바꿔 끼운다 (계획 5절 「root 로 도는 경우」). 되돌리는 함수를 돌려준다.
func (f *bakeFixture) failStateWrites(t *testing.T) func() {
	t.Helper()
	if os.Getuid() == 0 {
		was := writeLowerState
		writeLowerState = func(*lower.Bake, lower.State) error { return errors.New("injected: read-only file system") }
		restore := func() { writeLowerState = was }
		t.Cleanup(restore)
		return restore
	}
	p := f.g.Dir().Path
	if err := os.Chmod(p, 0o500); err != nil {
		t.Fatal(err)
	}
	restore := func() { _ = os.Chmod(p, 0o700) }
	t.Cleanup(restore)
	return restore
}

// blockTrash 는 scratch 의 trash 자리를 파일로 막는다 — 대기 자리를 옮길 수 없다.
func blockTrash(t *testing.T, scratchDir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(scratchDir, "trash"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
}

// entries 는 디렉터리의 이름들이다. 없으면 nil.
func entries(t *testing.T, dir string) []string {
	t.Helper()
	ents, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	var out []string
	for _, e := range ents {
		out = append(out, e.Name())
	}
	return out
}

// 굽기 계약의 단계

const (
	syncCmd = "sync the tree to $ENODE_IR"
	buildA  = "build config-a"
	buildB  = "build config-b"
	bakeIR  = "ir-1"
)

func buildStepOf(run string) *Step {
	return &Step{RunID: run, Seq: 1, StepID: run + "#01", Name: "build", Kind: "build", Effect: contract.EffectPrepare,
		Sync: syncCmd, IR: bakeIR, Builds: []contract.Build{{Name: "config-a", Command: buildA},
			{Name: "config-b", Command: buildB}}}
}

func mergeStepOf(run string) *Step {
	return &Step{RunID: run, Seq: 2, StepID: run + "#02", Name: "merge", Kind: "merge", Merge: &contract.Merge{}}
}

// probeMatch 는 대조가 맞은 git 모양의 출력이다 — origin 에 비밀번호가 든 url.
func probeMatch() fakeAnswer {
	return fakeAnswer{stdout: "mode=git\nhead=" + headA + "\ntagged=" + headA + "\ntag=" + bakeIR +
		"\ntag=zz-other\nurl=https://u:secret@git.example/x.git\nbranch=main\n"}
}

// probeRepo 는 대조가 맞은 repo 모양의 출력이다.
func probeRepo() fakeAnswer {
	return fakeAnswer{stdout: "mode=repo\nhead=" + headA + "\ntagged=" + headA + "\ntag=" + bakeIR +
		"\nurl=https://git.example/manifest.git\nbranch=stable\n"}
}

// pinWrites 는 repo manifest -r 이 upper 에 고정 파일을 쓰는 모양이다.
func pinWrites(body string) fakeAnswer {
	return fakeAnswer{do: func(upper string) {
		_ = os.WriteFile(filepath.Join(upper, pinnedFile), []byte(body), 0o644)
	}}
}

// succeed 는 성공하는 굽기의 답을 둔다 — git 모양.
func (f *bakeFixture) succeed() {
	f.rt.answer(probeScript, probeMatch())
}

// buildOK 는 성공하는 build 단계 하나를 돌리고 결과를 돌려준다.
func (f *bakeFixture) buildOK(t *testing.T, run string) Result {
	t.Helper()
	f.succeed()
	f.w.execute(context.Background(), buildStepOf(run))
	res := f.m.only(t)
	if res.Error != "" || f.state(t).Phase != lower.PhasePending {
		t.Fatalf("the build did not pend: %+v, state %+v", res, f.state(t))
	}
	return res
}

// lastResult 는 가짜 Mediator 가 마지막으로 받은 결과다.
func (m *mediator) lastResult(t *testing.T) Result {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.results) == 0 {
		t.Fatal("no result was reported")
	}
	return m.results[len(m.results)-1]
}

// progressText 는 진행 청크 전부다.
func (m *mediator) progressText() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return string(m.progress)
}

// uploaded 는 올라간 blob 이름들이다.
func (m *mediator) uploaded() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []string
	for name := range m.blobs {
		out = append(out, name)
	}
	return out
}

// slogTo 는 w 로 쓰는 노드 로그다.
func slogTo(w io.Writer) *slog.Logger { return slog.New(slog.NewTextHandler(w, nil)) }

// nodeLogHas 는 노드 로그에 그 글이 있는가다.
func (f *bakeFixture) nodeLogHas(s string) bool { return strings.Contains(f.nodeLog.String(), s) }
