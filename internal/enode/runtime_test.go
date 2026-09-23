package enode

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	execenv "github.com/taeels/enode/internal/environment"
)

type trackingRuntime struct {
	opens    int
	projects int
	runs     int
	harvests int
	closes   int
	commands [][]string
	record   *execenv.Record
}

func (r *trackingRuntime) Open(_ context.Context, spec RuntimeSpec) (StepSession, error) {
	r.opens++
	return &trackingSession{owner: r, spec: spec}, nil
}

type trackingSession struct {
	owner *trackingRuntime
	spec  RuntimeSpec
}

func (s *trackingSession) Paths() RuntimePaths {
	return RuntimePaths{Dir: s.spec.Dir, In: s.spec.In, Out: s.spec.Out}
}

func (s *trackingSession) Project(_ context.Context, spec FrameworkProjectionSpec) (FrameworkProjection, error) {
	s.owner.projects++
	return FrameworkProjection{
		HarnessExecutable: spec.HarnessExecutable,
		EnodeExecutable:   spec.EnodeExecutable,
		Instrumentation:   spec.Instrumentation,
		CredentialHelper:  spec.CredentialHelper,
	}, nil
}

func (s *trackingSession) Run(_ context.Context, spec ProcessSpec) (int, error) {
	s.owner.runs++
	s.owner.commands = append(s.owner.commands, append([]string(nil), spec.Argv...))
	if spec.Stdout != nil && len(spec.Argv) > 0 && strings.Contains(spec.Argv[0], "claude") {
		_, _ = io.WriteString(spec.Stdout, "{\"type\":\"result\",\"subtype\":\"success\",\"num_turns\":1}\n")
	}
	return 0, nil
}

func (s *trackingSession) Harvest(_ context.Context, spec HarvestSpec) (HarvestResult, error) {
	s.owner.harvests++
	return (&nativeSession{}).Harvest(context.Background(), spec)
}

func (s *trackingSession) Close() error {
	s.owner.closes++
	if s.owner.closes > s.owner.opens {
		return errors.New("session closed more than once")
	}
	return nil
}

func (s *trackingSession) Environment() *execenv.Record { return s.owner.record }

func TestWorkerRoutesCommandAndAgentThroughStepRuntime(t *testing.T) {
	for _, kind := range []string{"command", "agent"} {
		t.Run(kind, func(t *testing.T) {
			m := newMediator(t)
			w := newWorker(m)
			holdLease(w)
			runtime := &trackingRuntime{record: &execenv.Record{
				ProfileSHA256: "profile", PreparedEnvironment: "prepared", Runtime: "test-runtime",
			}}
			w.Runtime = runtime
			var step *Step
			if kind == "agent" {
				w.Local.HarnessBin = "/fake/claude"
				step = agentStep(`{}`)
			} else {
				step = runStep("/fake/build")
			}
			w.execute(context.Background(), step)
			res := m.only(t)
			if res.Error != "" {
				t.Fatalf("runtime path failed: %s", res.Error)
			}
			wantProjects := 0
			if kind == "agent" {
				wantProjects = 1
			}
			if runtime.opens != 1 || runtime.projects != wantProjects || runtime.runs != 1 ||
				runtime.harvests != 1 || runtime.closes != 1 {
				t.Fatalf("want Open/Project/Run/Harvest/Close 1/%d/1/1/1, got %d/%d/%d/%d/%d",
					wantProjects, runtime.opens, runtime.projects, runtime.runs, runtime.harvests, runtime.closes)
			}
			if res.Environment == nil || res.Environment.PreparedEnvironment != "prepared" {
				t.Fatalf("runtime identity did not reach the result: %+v", res.Environment)
			}
		})
	}
}

func TestNativeRuntimePreservesExitCodeAndOutput(t *testing.T) {
	session, err := (NativeRuntime{}).Open(context.Background(), RuntimeSpec{})
	if err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	code, runErr := session.Run(context.Background(), ProcessSpec{
		Argv: []string{"sh", "-c", "printf native; exit 7"}, Stdout: &out, Stderr: &out,
	})
	if code != 7 || runErr == nil || out.String() != "native" {
		t.Fatalf("native behavior changed: code=%d err=%v out=%q", code, runErr, out.String())
	}
}

func TestNativeHarvestDiscoversChangesWithoutNewContractDeclarations(t *testing.T) {
	workspace, out := t.TempDir(), t.TempDir()
	stamp := Stamp{At: time.Now().Add(-time.Second), Root: workspace}
	if err := os.WriteFile(filepath.Join(workspace, "discovered-by-runtime.txt"), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "direct-output"), []byte("already in OUT"), 0o644); err != nil {
		t.Fatal(err)
	}
	session, err := (NativeRuntime{}).Open(context.Background(), RuntimeSpec{})
	if err != nil {
		t.Fatal(err)
	}
	result, err := session.Harvest(context.Background(), HarvestSpec{
		Workspace: workspace, Out: out, Stamp: stamp, Discover: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.WorkspaceN == 0 || len(result.Workspace) == 0 ||
		result.Workspace[0].Path != "discovered-by-runtime.txt" {
		t.Fatalf("automatic workspace discovery was lost: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(out, "direct-output")); err != nil {
		t.Fatalf("direct $OUT stopped being the default result path: %v", err)
	}
	if len(result.Collected) != 0 || len(result.Notes) != 0 {
		t.Fatalf("empty optional collect became a requirement: %+v", result)
	}
}
