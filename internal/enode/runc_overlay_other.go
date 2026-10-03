//go:build !linux

package enode

import (
	"context"
	"errors"
	"io"

	execenv "github.com/taeels/enode/internal/environment"
	"github.com/taeels/enode/internal/scratch"
)

type RuncOverlayRuntime struct{}

func NewRuncOverlayRuntime(execenv.Document, execenv.Binding, execenv.Manifest) (*RuncOverlayRuntime, error) {
	return nil, errors.New("runc-overlay is supported only on Linux")
}

func (*RuncOverlayRuntime) Open(context.Context, RuntimeSpec) (StepSession, error) {
	return nil, errors.New("runc-overlay is supported only on Linux")
}

// Capability 는 linux 판과 같은 값이다 — 짓지 못하므로 광고에 오르지 않는다.
func (*RuncOverlayRuntime) Capability() RuntimeCapability {
	return RuntimeCapability{Writes: writesIsolated, Capture: CaptureSupport{Supported: true,
		Scope: scratch.ScopeWorkspaceUpper, Guarantee: scratch.GuaranteeInspectOnly}}
}

func RunRuncOverlayHelper(io.Reader, io.Writer, io.Writer) int { return 1 }

// Verify 는 linux 밖에서 native 만 받는다. type 은 lowercheck.go 에 있다.
func (ExecutionRuntimeVerifier) Verify(_ context.Context, doc execenv.Document, _ execenv.Binding, _ string, _ execenv.Manifest) error {
	if doc.Profile.Runtime.Driver == "native" {
		return nil
	}
	return errors.New("runc-overlay is supported only on Linux")
}
