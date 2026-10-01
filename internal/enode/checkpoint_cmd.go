package enode

// enode checkpoint list | show <ID> — 노드의 보존본 조회 (checkpoint 유닛 · business-rules.md 13절 · FD 답 6 · 7 · 8).
//
// 데몬을 띄우지 않는다. spool 의 기록만 읽고 spool 잠금을 쥐지 않는다. host 경로는 이 소유자 화면에만 나간다
// (requirements.md 5.3) — receipt · diagnostics · 단계 로그 · 광고에는 없다. 출력은 영어다 (CONVENTIONS 2.1).

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/taeels/enode/internal/scratch"
)

const checkpointUsage = "usage: enode checkpoint list|show [ID] --config PATH [--json]"

// RunCheckpointCmd 는 enode checkpoint 의 몸통이다. 종료 코드를 돌려준다 — 0 이면 답했다 · 1 이면 못 찾았거나 못
// 읽었다 · 2 이면 쓰는 법이 틀렸다.
func RunCheckpointCmd(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || (args[0] != "list" && args[0] != "show") {
		fmt.Fprintln(stderr, checkpointUsage)
		return 2
	}
	action := args[0]
	fs := flag.NewFlagSet("checkpoint "+action, flag.ContinueOnError)
	fs.SetOutput(stderr)
	configPath := fs.String("config", "", "node config path")
	asJSON := fs.Bool("json", false, "print the records as JSON")
	// ID 는 플래그 앞에도 뒤에도 올 수 있다 — flag 는 첫 위치 인자에서 멈추므로 두 번 읽는다
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	var id string
	if rest := fs.Args(); len(rest) > 0 {
		id = rest[0]
		if err := fs.Parse(rest[1:]); err != nil || fs.NArg() > 0 {
			fmt.Fprintln(stderr, checkpointUsage)
			return 2
		}
	}
	if (action == "show") != (id != "") {
		fmt.Fprintln(stderr, checkpointUsage)
		return 2
	}

	resolved, tried := ResolveConfig(*configPath)
	if resolved == "" {
		fmt.Fprintf(stderr, "error: no config file found (tried %v)\n", tried)
		return 1
	}
	local, err := LoadLocal(resolved)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	if local.Environment == nil || local.Environment.Scratch == "" {
		fmt.Fprintln(stdout, "this node has no scratch; checkpoints need the runc-overlay runtime")
		if action == "show" {
			return 1
		}
		return 0
	}
	scratchDir := local.Environment.Scratch
	store := &scratch.Store{Dir: scratch.SpoolIn(scratchDir), Scratch: scratchDir}

	if action == "list" {
		entries, err := store.List()
		if err != nil {
			fmt.Fprintf(stderr, "error: cannot read the spool: %v\n", err)
			return 1
		}
		if *asJSON {
			return writeJSONOut(stdout, stderr, nonNil(entries))
		}
		if len(entries) == 0 {
			fmt.Fprintln(stdout, "no checkpoints on this node")
			return 0
		}
		writeCheckpointList(stdout, entries)
		return 0
	}

	got, err := store.Lookup(id)
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot read the spool: %v\n", err)
		return 1
	}
	if got == nil {
		fmt.Fprintf(stderr, "no checkpoint %s on this node: it expired or was never captured here\n", id)
		return 1
	}
	if *asJSON {
		return writeJSONOut(stdout, stderr, got)
	}
	writeCheckpointShow(stdout, *got, scratchDir)
	return 0
}

func nonNil(l []scratch.Listed) []scratch.Listed {
	if l == nil {
		return []scratch.Listed{}
	}
	return l
}

func writeJSONOut(stdout, stderr io.Writer, v any) int {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, string(b))
	return 0
}

// writeCheckpointList 는 한 줄에 하나다 — 오래된 것부터 (Store.List 의 차례).
func writeCheckpointList(w io.Writer, entries []scratch.Listed) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tSTATE\tRUN\tSTEP\tCAPTURED\tEXPIRES\tSIZE\tREPORT")
	for _, e := range entries {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", e.ID, listedState(e), dash(e.Run), stepLabel(e.Entry),
			stamp(e.CapturedAt), stamp(e.ExpiresAt), sizeLabel(e.Entry), dash(e.Report))
	}
	_ = tw.Flush()
}

// writeCheckpointShow 는 칸마다 한 줄이다. 기록에 없는 칸은 줄째 뺀다.
func writeCheckpointShow(w io.Writer, e scratch.Listed, scratchDir string) {
	line := func(key, value string) {
		if value != "" {
			fmt.Fprintf(w, "%-12s %s\n", key, value)
		}
	}
	line("id", e.ID)
	line("state", listedState(e))
	line("node", e.Node)
	if e.Run != "" {
		line("run", fmt.Sprintf("%s   step %s   attempt %d", e.Run, stepLabel(e.Entry), e.Attempt))
	}
	if e.Runtime != "" {
		line("runtime", fmt.Sprintf("%s (%s)", e.Runtime, e.Format))
	}
	if e.Scope != "" {
		line("scope", e.Scope+", "+e.Guarantee)
	}
	line("lower", e.Lower)
	line("environment", e.Environment)
	switch {
	case e.Head != "" && e.IR != "":
		line("head", e.Head+"   ir "+e.IR)
	case e.Head != "":
		line("head", e.Head)
	case e.IR != "":
		line("ir", e.IR)
	}
	line("captured", optStamp(e.CapturedAt))
	line("expires", optStamp(e.ExpiresAt))
	if !e.Incomplete {
		if e.Bytes != nil {
			size := humanBytes(*e.Bytes)
			if e.Inodes != nil {
				size += fmt.Sprintf(", %d entries", *e.Inodes)
			}
			if e.MeasuredAt != nil {
				size += " (measured " + e.MeasuredAt.UTC().Format(time.RFC3339) + ")"
			}
			line("size", size)
		} else if e.State == scratch.EntryKept {
			line("size", "unmeasured")
		}
	}
	line("report", e.Report)
	if e.State == scratch.EntryEvicted {
		line("evicted", fmt.Sprintf("%s at %s; the tree is gone and this record stays until %s",
			e.Evicted, optStamp(e.EvictedAt), optStamp(e.ExpiresAt)))
		return
	}
	dir := filepath.Dir(e.Path)
	line("path", e.Path)
	if !e.Incomplete {
		fmt.Fprintf(w, "%-12s %s\n%-12s %s\n", "open",
			"entries written as the container root belong to subordinate uids; read them inside", "",
			"unshare --user --map-root-user --map-auto (the same mapping the helpers use)")
	}
	line("discard", fmt.Sprintf("move %s into %s/; the background deleter removes it", dir,
		scratch.TrashIn(scratchDir).Dir))
}

func listedState(e scratch.Listed) string {
	switch {
	case e.Incomplete:
		return "incomplete"
	case e.State == scratch.EntryEvicted && e.Evicted != "":
		return "evicted: " + e.Evicted
	}
	return e.State
}

func stepLabel(e scratch.Entry) string {
	if e.Step == "" {
		return "-"
	}
	return fmt.Sprintf("%d %s", e.Seq, e.Step)
}

func sizeLabel(e scratch.Entry) string {
	switch {
	case e.Bytes != nil:
		return humanBytes(*e.Bytes)
	case e.State == scratch.EntryKept:
		return "unmeasured"
	}
	return "-"
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func stamp(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.UTC().Format(time.RFC3339)
}

func optStamp(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// humanBytes 는 GiB 이 기본이고, 작은 보존본은 MiB · KiB 로 적는다 — 0.0 GiB 로 적으면 크기를 못 읽는다.
func humanBytes(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GiB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(n)/(1<<20))
	}
	return fmt.Sprintf("%.1f KiB", float64(n)/(1<<10))
}
