package environment

import (
	"context"
	"encoding/json"
	"os/exec"
)

func commandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

func jsonUnmarshal(b []byte, value any) error { return json.Unmarshal(b, value) }
