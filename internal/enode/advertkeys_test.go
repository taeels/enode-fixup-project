package enode

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/lower"
)

// 광고 키의 표 시험 — 모든 플랫폼에서 돈다 (business-rules.md 11절).

func TestReservedKey(t *testing.T) {
	for k, want := range map[string]bool{
		"workspace.writes": true, "ir": true, "repo.built.config-a": true, "repo.built.": true,
		"bake.run": true, "bake.resumed": true,
		"repo": false, "arch": false, "irx": false, "bake": false, "repo.builtx": false, "harness.claude": false,
	} {
		if got := reservedKey(k); got != want {
			t.Errorf("reservedKey(%q) = %v, want %v", k, got, want)
		}
	}
}

// metadata 에서 키 — 어긋난 이름과 IR 은 그 키만 빼고, bake.run 이 상한을 넘으면 bake 둘을 뺀다.
func TestMetadataKeys(t *testing.T) {
	if keys, dropped := metadataKeys(nil); keys != nil || dropped != nil {
		t.Fatalf("no metadata = %v, %v", keys, dropped)
	}
	ir, bad := "your-ir-tag", "bad ir"
	md := &lower.Metadata{Source: lower.Source{IR: &ir},
		Builds: []lower.BuildRecord{{Name: "config-a"}, {Name: "Config_B"}, {Name: "config-c"}},
		Bake:   lower.BakeRecord{Run: "R-8790", Resumed: false}}
	keys, dropped := metadataKeys(md)
	want := map[string]string{"ir": "your-ir-tag", "repo.built.config-a": "yes", "repo.built.config-c": "yes",
		"bake.run": "R-8790", "bake.resumed": "false"}
	if !maps.Equal(keys, want) {
		t.Errorf("keys = %v, want %v", keys, want)
	}
	if len(dropped) != 1 || !strings.Contains(dropped[0], `"Config_B"`) {
		t.Errorf("dropped = %q", dropped)
	}

	md.Source.IR = &bad
	md.Bake = lower.BakeRecord{Run: strings.Repeat("r", 129), Resumed: true}
	keys, dropped = metadataKeys(md)
	if _, ok := keys["ir"]; ok || keys["bake.run"] != "" || keys["bake.resumed"] != "" || keys["repo.built.config-a"] != "yes" {
		t.Errorf("keys = %v", keys)
	}
	if len(dropped) != 3 || !strings.Contains(dropped[0], `ir "bad ir": ' ' is not allowed`) ||
		dropped[2] != "bake.run is longer than 128 bytes" {
		t.Errorf("dropped = %q", dropped)
	}

	empty := ""
	md = &lower.Metadata{Source: lower.Source{IR: &empty}, Bake: lower.BakeRecord{Run: strings.Repeat("r", 128), Resumed: true}}
	keys, dropped = metadataKeys(md)
	if len(dropped) != 0 || !maps.Equal(keys, map[string]string{"bake.run": strings.Repeat("r", 128), "bake.resumed": "true"}) {
		t.Errorf("keys = %v dropped %q", keys, dropped)
	}
}

// 복사본에 싣는다 — 원본은 그대로 · 라벨이 적은 예약 키는 빠진다 · 뺀 뒤 할 줄 아는 것이 없으면 그 능력을 뺀다 ·
// 모든 능력에 싣는다 · 능력이 0 이면 안 싣는다.
func TestAdvertCaps(t *testing.T) {
	caps := []contract.Capability{
		{Capability: contract.CapabilityAgentReason, Attrs: map[string]string{
			"os": "linux", "harness.claude": "2.1", "ir": "label-ir", "repo.built.x": "yes"}},
		{Capability: contract.CapabilityOrchestration, Attrs: map[string]string{"os": "linux", "harness.claude": "2.1"}},
		{Capability: contract.CapabilityAgentReason, Attrs: map[string]string{"os": "linux", "bake.run": "label"}},
	}
	before := []map[string]string{maps.Clone(caps[0].Attrs), maps.Clone(caps[1].Attrs), maps.Clone(caps[2].Attrs)}
	var ignored []string
	out := advertCaps(caps, "isolated", map[string]string{"ir": "your-ir-tag"}, func(k string) { ignored = append(ignored, k) })
	if len(out) != 2 {
		t.Fatalf("out = %+v", out)
	}
	for _, c := range out {
		if c.Attrs["workspace.writes"] != "isolated" || c.Attrs["ir"] != "your-ir-tag" || c.Attrs["repo.built.x"] != "" ||
			c.Attrs["harness.claude"] != "2.1" {
			t.Errorf("capability %s = %v", c.Capability, c.Attrs)
		}
	}
	slices.Sort(ignored)
	if !slices.Equal(ignored, []string{"bake.run", "ir", "repo.built.x"}) {
		t.Errorf("ignored = %q", ignored)
	}
	for i := range caps {
		if !maps.Equal(caps[i].Attrs, before[i]) {
			t.Errorf("the original capability %d changed: %v", i, caps[i].Attrs)
		}
	}
	if out := advertCaps(nil, "in-place", map[string]string{"ir": "x"}, nil); out != nil {
		t.Errorf("no capability = %+v", out)
	}
	if out := advertCaps([]contract.Capability{{Capability: "agent.reason", Attrs: map[string]string{"os": "linux"}}},
		"in-place", nil, nil); out != nil {
		t.Errorf("a capability that can do nothing = %+v", out)
	}
}
