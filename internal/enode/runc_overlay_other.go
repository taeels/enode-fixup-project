//go:build !linux

package enode

import (
	"context"
	"errors"
	"io"

	execenv "github.com/taeels/enode/internal/environment"
)

type RuncOverlayRuntime struct{}

func NewRuncOverlayRuntime(execenv.Document, execenv.Binding, execenv.Manifest) (*RuncOverlayRuntime, error) {
	return nil, errors.New("runc-overlay is supported only on Linux")
}

func (*RuncOverlayRuntime) Open(context.Context, RuntimeSpec) (StepSession, error) {
	return nil, errors.New("runc-overlay is supported only on Linux")
}

func RunRuncOverlayHelper(io.Reader, io.Writer, io.Writer) int { return 1 }

type ExecutionRuntimeVerifier struct{}

func (ExecutionRuntimeVerifier) Verify(_ context.Context, doc execenv.Document, _ execenv.Binding, _ string, _ execenv.Manifest) error {
	if doc.Profile.Runtime.Driver == "native" {
		return nil
	}
	return errors.New("runc-overlay is supported only on Linux")
}
