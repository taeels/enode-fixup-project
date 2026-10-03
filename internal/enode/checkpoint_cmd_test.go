//go:build linux

package enode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/taeels/enode/internal/scratch"
)

// cmdSpool 은 조회가 읽을 spool 이다 — 보관 하나 (측정함 · 보고 닿음) · 퇴출 하나 · 주인 없는 예약 하나 · 도는 예약 하나.
func cmdSpool(t *testing.T) (config string, st *scratch.Store, kept, evicted, ownerless, held *scratch.Reservation) {
	t.Helper()
	dir := t.TempDir()
	config = filepath.Join(dir, "node.yaml")
	body := fmt.Sprintf("mediator: http://m:8080\ntoken: t\nenvironment: {profile: p.yaml, store: %s, scratch: %s}\n",
		filepath.Join(dir, "store"), filepath.Join(dir, "scratch"))
	if err := os.WriteFile(config, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	scratchDir := filepath.Join(dir, "scratch")
	clock := time.Date(2026, 9, 30, 13, 2, 11, 0, time.UTC)
	st = &scratch.Store{Dir: scratch.SpoolIn(scratchDir), Scratch: scratchDir, Trash: scratch.TrashIn(scratchDir),
		Policy: scratch.Policy{Mode: scratch.ModeOnFailure, TTL: 48 * time.Hour, CapacityPercent: 20, MaxBytes: 1 << 30},
		Stat:   func() (scratch.Filesystem, error) { return scratch.Filesystem{Known: true, Free: 1 << 40}, nil },
		Now:    func() time.Time { return clock }}
	if _, err := st.Open(); err != nil {
		t.Fatal(err)
	}
	make1 := func(seq int, step string) *scratch.Reservation {
		r, err := st.Reserve(scratch.Entry{Node: "n1", Run: "r-7a1c", Seq: seq, Step: step, Attempt: 1,
			Runtime: "runc-overlay", Format: scratch.FormatOverlayUpper, Scope: scratch.ScopeWorkspaceUpper,
			Guarantee: scratch.GuaranteeInspectOnly})
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	kept = make1(3, "build")
	if err := os.Mkdir(kept.Upper(), 0o700); err != nil {
		t.Fatal(err)
	}
	e := kept.Entry()
	e.Lower, e.Environment, e.Head, e.IR = "fd00-1234", "env-1", "b34499f", "ir-2"
	if _, err := st.Commit(kept, e); err != nil {
		t.Fatal(err)
	}
	clock = clock.Add(time.Minute)
	evicted = make1(5, "test")
	if err := os.Mkdir(evicted.Upper(), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Commit(evicted, evicted.Entry()); err != nil {
		t.Fatal(err)
	}
	st.Measure = func(_ context.Context, root, _ string) (scratch.Size, error) {
		if filepath.Base(root) == evicted.ID {
			return scratch.Size{Bytes: 31 << 30, Entries: 9}, nil
		}
		return scratch.Size{Bytes: 4096, Entries: 3}, nil
	}
	if _, err := st.Settle(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := st.Report(kept.ID, scratch.ReportDelivered); err != nil {
		t.Fatal(err)
	}
	ownerless = make1(7, "agent")
	ownerlessLock := filepath.Join(ownerless.Dir, scratch.SessionLockName)
	// 데몬이 죽은 모양 — 기록과 잠금 파일은 남고 잠금은 풀렸다. 예약을 버린 뒤 그 둘을 다시 놓는다
	rec, err := os.ReadFile(filepath.Join(ownerless.Dir, "checkpoint.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Dispose(ownerless); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(ownerless.Dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ownerless.Dir, "checkpoint.json"), rec, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ownerlessLock, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	held = make1(9, "deploy")
	t.Cleanup(func() { _ = st.Dispose(held) })
	return config, st, kept, evicted, ownerless, held
}

func runCheckpointCmd(args ...string) (int, string, string) {
	var out, errOut bytes.Buffer
	code := RunCheckpointCmd(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestCheckpointCmd_List(t *testing.T) {
	config, _, kept, evicted, ownerless, held := cmdSpool(t)
	code, out, stderr := runCheckpointCmd("list", "--config", config)
	if code != 0 || stderr != "" {
		t.Fatalf("list = %d · %q", code, stderr)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	header := regexp.MustCompile(`^ID +STATE +RUN +STEP +CAPTURED +EXPIRES +SIZE +REPORT$`)
	if len(lines) != 4 || !header.MatchString(lines[0]) {
		t.Fatalf("list:\n%s", out)
	}
	for i, want := range []string{
		`^` + kept.ID + ` +kept +r-7a1c +3 build +2026-09-30T13:02:11Z +2026-10-02T13:02:11Z +4\.0 KiB +delivered$`,
		`^` + evicted.ID + ` +evicted: larger than max_gb +r-7a1c +5 test +2026-09-30T13:03:11Z +2026-10-02T13:03:11Z +31\.0 GiB +-$`,
		`^` + ownerless.ID + ` +incomplete +r-7a1c +7 agent +- +- +- +-$`,
	} {
		if !regexp.MustCompile(want).MatchString(lines[i+1]) {
			t.Fatalf("line %d = %q, want %s\n%s", i+1, lines[i+1], want, out)
		}
	}
	if strings.Contains(out, held.ID) {
		t.Fatalf("a running reservation is listed:\n%s", out)
	}
}

func TestCheckpointCmd_Show(t *testing.T) {
	config, st, kept, evicted, _, _ := cmdSpool(t)
	code, out, stderr := runCheckpointCmd("show", kept.ID, "--config", config)
	if code != 0 || stderr != "" {
		t.Fatalf("show = %d · %q", code, stderr)
	}
	spoolDir := filepath.Join(st.Dir, kept.ID)
	want := strings.Join([]string{
		"id           " + kept.ID,
		"state        kept",
		"node         n1",
		"run          r-7a1c   step 3 build   attempt 1",
		"runtime      runc-overlay (overlay-upper)",
		"scope        workspace-upper, inspect-only",
		"lower        fd00-1234",
		"environment  env-1",
		"head         b34499f   ir ir-2",
		"captured     2026-09-30T13:02:11Z",
		"expires      2026-10-02T13:02:11Z",
		"size         4.0 KiB, 3 entries (measured 2026-09-30T13:03:11Z)",
		"report       delivered",
		"path         " + filepath.Join(spoolDir, "upper"),
		"open         entries written as the container root belong to subordinate uids; read them inside",
		"             unshare --user --map-root-user --map-auto (the same mapping the helpers use)",
		"discard      move " + spoolDir + " into " + st.Trash.Dir + "/; the background deleter removes it",
	}, "\n") + "\n"
	if out != want {
		t.Fatalf("show:\n%s\nwant:\n%s", out, want)
	}
	code, out, _ = runCheckpointCmd("show", "--config", config, evicted.ID)
	if code != 0 || !strings.Contains(out, "evicted      larger than max_gb at 2026-09-30T13:03:11Z; the tree is gone and this "+
		"record stays until 2026-10-02T13:03:11Z\n") || strings.Contains(out, "path ") {
		t.Fatalf("show evicted = %d:\n%s", code, out)
	}
}

func TestCheckpointCmd_JSON(t *testing.T) {
	config, _, kept, _, _, _ := cmdSpool(t)
	code, out, _ := runCheckpointCmd("list", "--config", config, "--json")
	var list []map[string]any
	if code != 0 || json.Unmarshal([]byte(out), &list) != nil || len(list) != 3 || list[0]["id"] != kept.ID ||
		list[0]["path"] == nil || list[0]["report"] != "delivered" {
		t.Fatalf("list --json = %d:\n%s", code, out)
	}
	code, out, _ = runCheckpointCmd("show", kept.ID, "--json", "--config", config)
	var one map[string]any
	if code != 0 || json.Unmarshal([]byte(out), &one) != nil || one["id"] != kept.ID || one["state"] != "kept" {
		t.Fatalf("show --json = %d:\n%s", code, out)
	}
}

// 모르는 ID · 만료된 ID · 도는 단계의 예약은 같은 문장이다 (FD 답 7).
func TestCheckpointCmd_UnknownID(t *testing.T) {
	config, _, _, _, _, held := cmdSpool(t)
	for _, id := range []string{"000000000000", held.ID, "../escaped"} {
		code, out, stderr := runCheckpointCmd("show", id, "--config", config)
		if code != 1 || out != "" ||
			stderr != "no checkpoint "+id+" on this node: it expired or was never captured here\n" {
			t.Fatalf("show %s = %d · %q · %q", id, code, out, stderr)
		}
	}
}

func TestCheckpointCmd_NoScratch(t *testing.T) {
	config := filepath.Join(t.TempDir(), "node.yaml")
	if err := os.WriteFile(config, []byte("mediator: http://m:8080\ntoken: t\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runCheckpointCmd("list", "--config", config)
	if code != 0 || out != "this node has no scratch; checkpoints need the runc-overlay runtime\n" {
		t.Fatalf("list on a native node = %d · %q", code, out)
	}
	empty := filepath.Join(t.TempDir(), "node.yaml")
	body := fmt.Sprintf("mediator: http://m:8080\ntoken: t\nenvironment: {scratch: %s}\n", t.TempDir())
	if err := os.WriteFile(empty, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, out, _ := runCheckpointCmd("list", "--config", empty); code != 0 || out != "no checkpoints on this node\n" {
		t.Fatalf("list on an empty spool = %d · %q", code, out)
	}
	for _, args := range [][]string{{}, {"drop"}, {"show"}, {"list", "extra", "more"}} {
		if code, _, stderr := runCheckpointCmd(args...); code != 2 || !strings.Contains(stderr, checkpointUsage) {
			t.Fatalf("%v = %d · %q", args, code, stderr)
		}
	}
}
