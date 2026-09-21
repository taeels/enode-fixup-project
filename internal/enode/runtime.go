package enode

import (
	"context"
	"io"
	"os/exec"
	"sync"

	execenv "github.com/taeels/enode/internal/environment"
)

type RuntimeSpec struct {
	RunID  string
	StepID string
	Dir    string
	In     string
	Out    string
	Record *execenv.Record
}

type ProcessSpec struct {
	Argv   []string
	Dir    string
	Env    []string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type RuntimePaths struct {
	Dir string
	In  string
	Out string
}

// FrameworkProjectionSpec은 node-local framework source만 나른다. container
// target과 mount option은 runtime role별 고정값이며 caller가 정하지 않는다.
type FrameworkProjectionSpec struct {
	HarnessExecutable string
	EnodeExecutable   string
	Instrumentation   string
	CredentialHelper  string
}

// FrameworkProjection은 process와 instrumentation이 실제로 써야 할 runtime
// visible path다. native에서는 source와 같고 runc-overlay에서는 /run/enode 아래
// 고정 target을 돌려준다.
type FrameworkProjection struct {
	HarnessExecutable string
	EnodeExecutable   string
	Instrumentation   string
	CredentialHelper  string
}

// HarvestSpec은 새 Run 계약이 아니다. Worker가 이미 받은 workspace/collect/
// check_changed와 내부 stamp를 session 안으로 옮긴 값이다. 비어 있으면 직접
// $OUT만 기존 경로로 봉인되며 계약 제출자가 추가로 선언할 것은 없다.
type HarvestSpec struct {
	Workspace  string
	Out        string
	RecordDiff bool
	Discover   bool
	Collect    map[string]string
	Check      []string
	Stamp      Stamp
}

type HarvestResult struct {
	Changed      []string
	Collected    []string
	Notes        []HarvestNote
	Workspace    []Changed
	WorkspaceN   int
	DiffBytes    int
	DiffError    string
	ChangedError string
}

type StepRuntime interface {
	Open(context.Context, RuntimeSpec) (StepSession, error)
}

type StepSession interface {
	Paths() RuntimePaths
	Project(context.Context, FrameworkProjectionSpec) (FrameworkProjection, error)
	Run(context.Context, ProcessSpec) (int, error)
	Harvest(context.Context, HarvestSpec) (HarvestResult, error)
	Close() error
	Environment() *execenv.Record
}

// NativeRuntime은 기존 exec.Cmd 동작을 StepRuntime 계약 뒤에 보존한다.
type NativeRuntime struct{}

func (NativeRuntime) Open(_ context.Context, spec RuntimeSpec) (StepSession, error) {
	return &nativeSession{spec: spec, record: spec.Record}, nil
}

type nativeSession struct {
	spec   RuntimeSpec
	record *execenv.Record
	once   sync.Once
}

func (s *nativeSession) Paths() RuntimePaths {
	return RuntimePaths{Dir: s.spec.Dir, In: s.spec.In, Out: s.spec.Out}
}

func (s *nativeSession) Project(_ context.Context, spec FrameworkProjectionSpec) (FrameworkProjection, error) {
	return FrameworkProjection{
		HarnessExecutable: spec.HarnessExecutable,
		EnodeExecutable:   spec.EnodeExecutable,
		Instrumentation:   spec.Instrumentation,
		CredentialHelper:  spec.CredentialHelper,
	}, nil
}

func (s *nativeSession) Run(ctx context.Context, spec ProcessSpec) (int, error) {
	if len(spec.Argv) == 0 {
		return -1, exec.ErrNotFound
	}
	cmd := child(exec.CommandContext(ctx, spec.Argv[0], spec.Argv[1:]...))
	cmd.Dir, cmd.Env = spec.Dir, spec.Env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = spec.Stdin, spec.Stdout, spec.Stderr
	err := cmd.Run()
	if cmd.ProcessState == nil {
		return -1, err
	}
	return cmd.ProcessState.ExitCode(), err
}

func (s *nativeSession) Harvest(ctx context.Context, spec HarvestSpec) (HarvestResult, error) {
	var result HarvestResult
	stamp := spec.Stamp
	stamp.Root = spec.Workspace
	if spec.RecordDiff && spec.Workspace != "" {
		n, err := writeWorkspaceDiff(ctx, spec.Workspace, spec.Out, maxBlobBytes)
		result.DiffBytes = n
		if err != nil {
			result.DiffError = err.Error()
		}
	}
	result.Collected, result.Notes = collectDeclared(spec.Workspace, spec.Out, spec.Collect)
	result.Changed = CheckChanged(stamp, spec.Check)
	if spec.Discover && !stamp.At.IsZero() && stamp.Root != "" {
		found, total, err := changedSince(stamp, 2000)
		result.Workspace, result.WorkspaceN = found, total
		if err != nil {
			result.ChangedError = err.Error()
		}
	}
	return result, nil
}

func (s *nativeSession) Close() error {
	s.once.Do(func() {})
	return nil
}

func (s *nativeSession) Environment() *execenv.Record { return s.record }

type onceSession struct {
	StepSession
	once sync.Once
	err  error
}

func manageSession(session StepSession) StepSession {
	return &onceSession{StepSession: session}
}

func (s *onceSession) Close() error {
	s.once.Do(func() { s.err = s.StepSession.Close() })
	return s.err
}
