package enode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/taeels/enode/internal/scratch"
)

// checkpoint 블록 (business-rules.md 10절) — 0 은 기본값 · 틀린 값은 노드를 안 띄운다.
func TestCheckpointConfig_Validate(t *testing.T) {
	for _, tc := range []struct {
		name, block, err string
		want             scratch.Policy
	}{
		{"no block", "", "", scratch.Policy{Mode: scratch.ModeOnFailure, TTL: 48 * time.Hour, CapacityPercent: 20,
			MaxBytes: 32 << 30, MinFree: 10 << 30}},
		{"zeros are defaults", "checkpoint: {policy: '', ttl_hours: 0, capacity_percent: 0, max_gb: 0}", "",
			scratch.Policy{Mode: scratch.ModeOnFailure, TTL: 48 * time.Hour, CapacityPercent: 20, MaxBytes: 32 << 30,
				MinFree: 10 << 30}},
		{"written values", "checkpoint: {policy: always, ttl_hours: 72, capacity_percent: 5, max_gb: 1, max_total_inodes: 5000000}",
			"", scratch.Policy{Mode: scratch.ModeAlways, TTL: 72 * time.Hour, CapacityPercent: 5, MaxBytes: 1 << 30,
				MaxTotalInodes: 5000000, MinFree: 10 << 30}},
		{"off", "checkpoint: {policy: off}", "", scratch.Policy{Mode: scratch.ModeOff, TTL: 48 * time.Hour,
			CapacityPercent: 20, MaxBytes: 32 << 30, MinFree: 10 << 30}},
		{"unknown policy", "checkpoint: {policy: sometimes}", `checkpoint: policy "sometimes" is not off, on-failure or always`, scratch.Policy{}},
		{"negative ttl", "checkpoint: {ttl_hours: -1}", "checkpoint: ttl_hours must be at least 1", scratch.Policy{}},
		{"share over 100", "checkpoint: {capacity_percent: 101}", "checkpoint: capacity_percent must be between 1 and 100", scratch.Policy{}},
		{"negative share", "checkpoint: {capacity_percent: -5}", "checkpoint: capacity_percent must be between 1 and 100", scratch.Policy{}},
		{"negative max_gb", "checkpoint: {max_gb: -1}", "checkpoint: max_gb must be at least 1", scratch.Policy{}},
		{"negative inodes", "checkpoint: {max_total_inodes: -1}", "checkpoint: max_total_inodes must be at least 1", scratch.Policy{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "node.yaml")
			body := "mediator: http://m:8080\ntoken: t\n" + tc.block + "\n"
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			l, err := LoadLocal(path)
			if tc.err != "" {
				if err == nil || !strings.HasSuffix(err.Error(), ": "+tc.err) || !strings.HasPrefix(err.Error(), "config "+path) {
					t.Fatalf("err = %v, want %q", err, tc.err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got := l.CheckpointPolicy(); got != tc.want {
				t.Fatalf("policy = %+v, want %+v", got, tc.want)
			}
		})
	}
}
