package enode

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	execenv "github.com/taeels/enode/internal/environment"
)

type trackingRuntime struct {
	opens    int
	runs     int
	closes   int
	commands [][]string
	record   *execenv.Record
}

func (r *trackingRuntime) Open(context.Context, RuntimeSpec) (StepSession, error) {
	r.opens++
	return &trackingSession{owner: r}, nil
}

type trackingSession struct{ owner *trackingRuntime }

func (s *trackingSession) Run(_ context.Context, spec ProcessSpec) (int, error) {
	s.owner.runs++
	s.owner.commands = append(s.owner.commands, append([]string(nil), spec.Argv...))
	if spec.Stdout != nil && len(spec.Argv) > 0 && strings.Contains(spec.Argv[0], "claude") {
		_, _ = io.WriteString(spec.Stdout, "{\"type\":\"result\",\"subtype\":\"success\",\"num_turns\":1}\n")
	}
	return 0, nil
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
			if runtime.opens != 1 || runtime.runs != 1 || runtime.closes != 1 {
				t.Fatalf("want one Open/Run/Close, got %d/%d/%d", runtime.opens, runtime.runs, runtime.closes)
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
