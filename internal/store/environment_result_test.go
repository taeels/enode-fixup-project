package store

import (
	"encoding/json"
	"testing"

	execenv "github.com/taeels/enode/internal/environment"
)

func TestStepResultPreservesExecutionEnvironmentIdentity(t *testing.T) {
	want := StepResult{Environment: &execenv.Record{
		ProfileSHA256: "profile", PreparedEnvironment: "sha256:prepared",
		Runtime: "runc-overlay", WorkspaceTarget: "/work", UID: 1000, GID: 1000,
		SSH: "readonly", TmpSize: "256MiB", TmpExecutable: true,
	}}
	b, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var got StepResult
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Environment == nil || got.Environment.PreparedEnvironment != "sha256:prepared" ||
		got.Environment.Runtime != "runc-overlay" {
		t.Fatalf("environment identity did not round-trip: %s", b)
	}
}
