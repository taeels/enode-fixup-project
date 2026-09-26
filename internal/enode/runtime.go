package enode

import (
	"context"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/taeels/enode/internal/contract"
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

// FinalizeSpec 은 Worker 가 세션에 넘기는 결과 확정의 입력이다. 새 Run 계약이 아니다 —
// Worker 가 이미 받은 collect · check_changed · effect · discover 와 내부 stamp 를
// 세션 안으로 옮긴 값이다 (ADR-075 §9 · business-rules.md 1절).
//
// HarvestSpec 을 칸마다 판정해 바꿨다 (domain-entities.md 2.1). RecordDiff 는 Diff 로
// 이름만 바뀌었고, Discover 는 늘 켜던 전체 훑기가 아니라 계약이 켠 명시 훑기가 됐다.
type FinalizeSpec struct {
	Workspace string // native 에서만.  overlay 는 helper 가 merged 로 채운다
	Out       string
	Effect    contract.Effect   // 기본값을 채운 뒤의 값
	Diff      bool              // workspace.diff 를 만드나
	Collect   map[string]string // collect 를 할 때만.  agent 가 완주 못 하면 비운다
	Check     []string          // 지목 경로 (ADR-037).  늘 있다
	Stamp     Stamp             // Root 가 비면 노드에 워크스페이스가 없다 — 명시 훑기를 건너뛴다
	Discover  bool
	// DiscoverFor 는 명시 훑기의 시간 상한이다 — Finalize 예산의 절반 (business-rules.md 5.2).
	// helper 는 Deadline 만으로는 예산을 모르므로 Worker 가 셈해 싣는다. 0 이면 기본 예산의 절반.
	DiscoverFor time.Duration
	// Deadline 은 Finalize 예산의 마감이다. overlay helper 에 요청과 함께 가서 helper 가
	// 자기 ctx 를 만든다 — helper 는 요청을 하나씩 처리해 Worker 의 ctx 를 볼 수 없다.
	// native 에서는 ctx 의 마감과 같다.
	Deadline time.Time
}

// FinalizeResult 는 결과 확정이 모은 사실이다. 판정이 아니다 (ADR-004 · I3).
type FinalizeResult struct {
	Changed   []string      // 지목 경로 중 바뀐 것 (ADR-037)
	Collected []string      // collect 가 $OUT 으로 옮긴 이름
	Notes     []HarvestNote // collect 가 못 걷은 것.  진단 칸의 collect 로 옮겨 담는다
	DiffBytes int
	DiffError string
	Discovery *Discovery // Discover 가 켜졌을 때만
}

// Discovery 는 명시 훑기의 결과다. produced 가 아니다 (ADR-075 §8).
type Discovery struct {
	Paths   []Changed // 큰 파일부터.  결과 크기 상한 안쪽만
	Total   int       // 찾은 파일 수 (목록에서 자른 것 포함)
	Deleted int       // overlay upper 의 whiteout 수.  목록에 안 넣는다
	Visited int       // 방문한 항목 수 (디렉터리 포함)
	Limit   string    // 닿은 상한 — contract.LimitVisits · LimitTime · LimitMemory · LimitSize.  안 닿았으면 ""
	Skipped string    // 못 훑은 이유.  훑었으면 ""
}

// Keep 은 닫을 때 upper 의 행선지다. 이 유닛에서는 늘 비어 있다 — 쓰는 것은
// trash (trash 로) · bake (대기 자리) · checkpoint (spool) 유닛이다.
type Keep struct {
	Upper string // "" 면 버린다
}

type StepRuntime interface {
	Open(context.Context, RuntimeSpec) (StepSession, error)
}

type StepSession interface {
	Paths() RuntimePaths
	Project(context.Context, FrameworkProjectionSpec) (FrameworkProjection, error)
	Run(context.Context, ProcessSpec) (int, error)
	Finalize(context.Context, FinalizeSpec) (FinalizeResult, error)
	// Close 는 세션을 닫는다. ctx 는 이 유닛에서 받기만 한다 — 닫기는 trash 유닛이
	// rename 으로 바꾸기 전까지 Finalize 예산 밖이다 (business-logic-model.md 7절).
	Close(context.Context, Keep) error
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

func (s *nativeSession) Finalize(ctx context.Context, spec FinalizeSpec) (FinalizeResult, error) {
	return finalizeLocal(ctx, spec, func(ctx context.Context, limits discoverLimits) *Discovery {
		stamp := spec.Stamp
		stamp.Root = spec.Workspace
		return walkWorkspace(ctx, stamp, limits)
	})
}

// discoverNoWorkspace 는 명시 훑기를 켰는데 노드에 워크스페이스가 없을 때의 이유다.
const discoverNoWorkspace = "this node has no workspace directory"

// finalizeLocal 은 native 세션과 overlay helper 가 함께 쓰는 결과 확정이다.
// 둘이 다른 것은 명시 훑기의 곳 하나다 — native 는 워크스페이스, overlay 는 upper.
//
// 순서는 collect · 지목 경로 stat · diff · 명시 훑기다. stat 이 판정 재료이고 값이
// 싸므로 diff 앞에 둔다 — 느린 diff 가 마감을 넘겨도 changed 는 남는다.
// 마감이 지나면 그 자리에서 멈추고 ctx 의 오류를 돌려준다. 그때까지 모은 것은 남긴다.
func finalizeLocal(ctx context.Context, spec FinalizeSpec, discover func(context.Context, discoverLimits) *Discovery) (FinalizeResult, error) {
	var result FinalizeResult
	stamp := spec.Stamp
	stamp.Root = spec.Workspace
	result.Collected, result.Notes = collectDeclared(ctx, spec.Workspace, spec.Out, spec.Collect)
	if err := ctx.Err(); err != nil {
		return result, err
	}
	result.Changed = CheckChanged(stamp, spec.Check)
	if spec.Diff && spec.Workspace != "" {
		n, err := writeWorkspaceDiff(ctx, spec.Workspace, spec.Out, maxBlobBytes)
		result.DiffBytes = n
		if err != nil {
			result.DiffError = err.Error()
		}
		if err := ctx.Err(); err != nil {
			return result, err
		}
	}
	if spec.Discover {
		if spec.Stamp.Root == "" {
			result.Discovery = &Discovery{Skipped: discoverNoWorkspace}
		} else {
			result.Discovery = discover(ctx, defaultDiscoverLimits(spec.DiscoverFor))
		}
	}
	return result, ctx.Err()
}

func (s *nativeSession) Close(context.Context, Keep) error {
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

func (s *onceSession) Close(ctx context.Context, keep Keep) error {
	s.once.Do(func() { s.err = s.StepSession.Close(ctx, keep) })
	return s.err
}
