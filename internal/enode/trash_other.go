//go:build !linux

package enode

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/taeels/enode/internal/scratch"
)

// errTrashHelperLinux 는 trash-helper 가 없는 OS 의 답이다 — runc-overlay 노드가 linux 에만 있다.
var errTrashHelperLinux = errors.New("trash helper is only available on linux")

// RunTrashHelper 는 이 OS 에서 할 일이 없다.
func RunTrashHelper(_ []string, _, errOut io.Writer) int {
	fmt.Fprintln(errOut, errTrashHelperLinux)
	return 1
}

// TrashLauncher 는 helper 를 못 띄운다는 답만 한다.
func TrashLauncher(scratch.Trash) func(context.Context, string, func(scratch.Size)) error {
	return func(context.Context, string, func(scratch.Size)) error {
		return &scratch.LaunchError{Err: errTrashHelperLinux}
	}
}

// SweepOrphanSessions 는 할 일이 없다 — 남을 작업 폴더가 없다.
func SweepOrphanSessions(string, *slog.Logger) {}
