package enode

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	execenv "github.com/taeels/enode/internal/environment"
)

type projectionHarness struct {
	hostInstrumentation string
	self                string
	hook                HookArgs
	argvIO              IOPaths
}

func (*projectionHarness) Name() string                         { return "projection" }
func (*projectionHarness) Env() []string                        { return nil }
func (*projectionHarness) Usable(context.Context, string) error { return nil }
func (*projectionHarness) Version(context.Context, string) (string, error) {
	return "", errors.New("not relevant")
}
func (*projectionHarness) Decode(io.Reader, int, func(Event)) HarnessResult {
	return HarnessResult{Reason: ReasonOK}
}
func (h *projectionHarness) Argv(_ AgentParams, paths IOPaths) []string {
	h.argvIO = paths
	return []string{"--out", paths.Out}
}
func (*projectionHarness) Fixed(dir string) map[string]string {
	return map[string]string{"HARNESS_HOME": filepath.Join(dir, "home")}
}
func (h *projectionHarness) Instrument(dir, self string, hook HookArgs, _ Components) ([]string, error) {
	h.hostInstrumentation, h.self, h.hook = dir, self, hook
	return []string{"--settings", filepath.Join(dir, "settings.json")}, nil
}

type projectingSession struct {
	paths   RuntimePaths
	process ProcessSpec
	source  FrameworkProjectionSpec
}

func (s *projectingSession) Paths() RuntimePaths { return s.paths }
func (s *projectingSession) Project(_ context.Context, source FrameworkProjectionSpec) (FrameworkProjection, error) {
	s.source = source
	return FrameworkProjection{
		HarnessExecutable: "/run/enode/bin/harness",
		EnodeExecutable:   "/run/enode/bin/enode",
		Instrumentation:   "/run/enode/instrumentation",
		CredentialHelper:  "/run/enode/bin/auth-helper",
	}, nil
}
func (s *projectingSession) Run(_ context.Context, process ProcessSpec) (int, error) {
	s.process = process
	return 0, nil
}
func (*projectingSession) Finalize(context.Context, FinalizeSpec) (FinalizeResult, error) {
	return FinalizeResult{}, nil
}
func (*projectingSession) Close(context.Context, Keep) error { return nil }
func (*projectingSession) Environment() *execenv.Record      { return nil }

func TestRunnerUsesOnlyRuntimeVisibleProjectionTargets(t *testing.T) {
	host := IOPaths{Dir: t.TempDir(), In: t.TempDir(), Out: t.TempDir()}
	session := &projectingSession{paths: RuntimePaths{
		Dir: "/workspace", In: "/run/enode/in", Out: "/run/enode/out",
	}}
	harness := &projectionHarness{}
	_, result := runHarness(context.Background(), harness, "/host/bin/harness", Job{
		IO: host, Session: session,
	})
	if result.Reason != ReasonOK {
		t.Fatalf("projected harness failed: %+v", result)
	}
	if session.source.HarnessExecutable != "/host/bin/harness" || session.source.Instrumentation == "" {
		t.Fatalf("framework sources were not typed before projection: %+v", session.source)
	}
	if session.process.Argv[0] != "/run/enode/bin/harness" || session.process.Dir != "/workspace" {
		t.Fatalf("process used host paths: %+v", session.process)
	}
	wantArgs := []string{"--out", "/run/enode/out", "--settings", "/run/enode/instrumentation/settings.json"}
	if !slices.Equal(session.process.Argv[1:], wantArgs) {
		t.Fatalf("projected argv=%v want=%v", session.process.Argv[1:], wantArgs)
	}
	joinedEnv := strings.Join(session.process.Env, "\n")
	for _, want := range []string{
		"IN=/run/enode/in", "OUT=/run/enode/out",
		"HARNESS_HOME=/run/enode/instrumentation/home",
	} {
		if !strings.Contains(joinedEnv, want) {
			t.Fatalf("runtime env lacks %q: %s", want, joinedEnv)
		}
	}
	if harness.self != "/run/enode/bin/enode" || harness.hook.Out != "/run/enode/out" ||
		harness.hook.Workspace != "/workspace" {
		t.Fatalf("instrumentation received host paths: self=%q hook=%+v", harness.self, harness.hook)
	}
	// The exact OS temp root is intentionally not part of the contract; only
	// assert that the adapter did not write into the container-only target.
	if strings.HasPrefix(harness.hostInstrumentation, "/run/enode/") {
		t.Fatalf("adapter wrote files into a container-only target: %s", harness.hostInstrumentation)
	}
}
