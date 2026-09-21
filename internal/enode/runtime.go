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

type StepRuntime interface {
	Open(context.Context, RuntimeSpec) (StepSession, error)
}

type StepSession interface {
	Run(context.Context, ProcessSpec) (int, error)
	Close() error
	Environment() *execenv.Record
}

// NativeRuntime은 기존 exec.Cmd 동작을 StepRuntime 계약 뒤에 보존한다.
type NativeRuntime struct{}

func (NativeRuntime) Open(_ context.Context, spec RuntimeSpec) (StepSession, error) {
	return &nativeSession{record: spec.Record}, nil
}

type nativeSession struct {
	record *execenv.Record
	once   sync.Once
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
