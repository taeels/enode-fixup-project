package enode

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/lower"
	"github.com/taeels/enode/internal/merge"
)

// 굽기 규칙의 표 시험 — 시스템 호출이 없어 모든 플랫폼에서 돈다 (FD 흐름 8.1 의 「순수 함수」).

const (
	headA = "1111111111111111111111111111111111111111"
	headB = "2222222222222222222222222222222222222222"
)

func TestIRVerdict(t *testing.T) {
	cases := []struct {
		name string
		p    irProbe
		want irOutcome
		text string
	}{
		{"match with two tags", irProbe{Mode: "git", Head: headA, Tagged: headA, Tags: []string{"ir-1", "ir-1-rc"}},
			irMatch, "ir ir-1 matches HEAD " + headA + " (tags at HEAD: ir-1 ir-1-rc)"},
		// annotated 와 가벼운 태그는 ^{commit} 로 벗겨 같은 값이 된다 — 판정은 커밋만 본다
		{"match on repo", irProbe{Mode: "repo", Head: headA, Tagged: headA, Tags: []string{"ir-1"}},
			irMatch, "ir ir-1 matches HEAD " + headA + " (tags at HEAD: ir-1)"},
		{"not local", irProbe{Mode: "git", Head: headA, Tags: []string{}},
			irNotLocal, "ir ir-1 is not in the local repository after sync; the sync command must fetch that tag " +
				"(HEAD is " + headA + ", tags at HEAD: none)"},
		{"another commit", irProbe{Mode: "git", Head: headA, Tagged: headB, Tags: []string{"other"}},
			irElsewhere, "ir ir-1 points at " + headB + ", but sync left HEAD at " + headA + " (tags at HEAD: other)"},
		{"no git", irProbe{Mode: "none"}, irUnverified,
			"cannot verify ir: the workspace has neither .repo/manifests nor .git"},
		{"git failed", irProbe{Mode: "git", Exit: 128, Stderr: "fatal: invalid gitfile format: .git"}, irUnverified,
			"cannot verify ir: git exited 128: fatal: invalid gitfile format: .git"},
		{"git failed silently", irProbe{Mode: "git", Exit: 1}, irUnverified, "cannot verify ir: git exited 1"},
		{"cannot enter manifests", irProbe{Mode: "repo", Exit: 125, Stderr: "sh: 1: cd: can't cd to .repo/manifests"},
			irUnverified, "cannot verify ir: cannot enter .repo/manifests: sh: 1: cd: can't cd to .repo/manifests"},
		{"printed nothing", irProbe{}, irUnverified, "cannot verify ir: the check printed no HEAD"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, text := irVerdict("ir-1", tc.p)
			if got != tc.want || text != tc.text {
				t.Fatalf("irVerdict = %v %q\nwant %v %q", got, text, tc.want, tc.text)
			}
		})
	}
}

// 계획 3절 측정 3 의 다섯 모양 — 줄 끝 공백 · 모르는 줄은 버린다 · detached 는 branch "" · origin 없음은 url "".
func TestParseProbe(t *testing.T) {
	cases := []struct {
		name   string
		out    string
		exit   int
		stderr string
		want   irProbe
	}{
		{"init and fetch without origin, detached",
			"mode=git\nhead=" + headA + "\ntagged=" + headA + "\ntag=ir-1\ntag=ir-2\nurl=\nbranch=HEAD\n", 0, "",
			irProbe{Mode: "git", Head: headA, Tagged: headA, Tags: []string{"ir-1", "ir-2"}}},
		{"origin and a branch, no tags",
			"mode=git \r\nhead=" + headA + "\ntagged=\nurl=https://u:p@h/x\nbranch=main\nnoise\nwho=me\n", 0, "",
			irProbe{Mode: "git", Head: headA, Tags: []string{}, URL: "https://u:p@h/x", Branch: "main"}},
		{"no repository", "mode=none\n", 0, "", irProbe{Mode: "none"}},
		{"repo shape", "mode=repo\nhead=" + headA + "\ntagged=" + headA + "\ntag=ir-1\nurl=https://h/m\nbranch=stable\n",
			0, "", irProbe{Mode: "repo", Head: headA, Tagged: headA, Tags: []string{"ir-1"}, URL: "https://h/m",
				Branch: "stable"}},
		{"a broken .git", "mode=git\n", 128, "warning: x\nfatal: invalid gitfile format: .git\n\n",
			irProbe{Mode: "git", Exit: 128, Stderr: "fatal: invalid gitfile format: .git"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseProbe(tc.out, tc.exit, tc.stderr)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("parseProbe = %#v\nwant %#v", got, tc.want)
			}
		})
	}
}

func TestLastLine(t *testing.T) {
	if got := lastLine("a\nb\n\n  "); got != "b" {
		t.Fatalf("lastLine = %q", got)
	}
	long := strings.Repeat("x", tailMax+10) + "END"
	if got := lastLine(long); len(got) != tailMax || !strings.HasSuffix(got, "END") {
		t.Fatalf("lastLine kept %d bytes ending %q", len(got), got[len(got)-3:])
	}
	if got := lastLine(""); got != "" {
		t.Fatalf("lastLine of nothing = %q", got)
	}
}

// bash 확인의 갈래 넷 (결정 45) — 0 은 통과 · 1 이상은 문장 · 음수는 세션 오류.
func TestBashCheckText(t *testing.T) {
	if got := bashCheckText(0, nil); got != "" {
		t.Fatalf("exit 0 = %q", got)
	}
	for _, code := range []int{1, 127} {
		want := "cannot run the bake commands: the prepared rootfs has no bash, or sh cannot run there (exit " +
			map[int]string{1: "1", 127: "127"}[code] + ")"
		if got := bashCheckText(code, nil); got != want {
			t.Fatalf("exit %d = %q", code, got)
		}
	}
	if got := bashCheckText(-1, errors.New("runtime run: EOF: helper died")); got != "runtime run: EOF: helper died" {
		t.Fatalf("a session error = %q", got)
	}
	if got := bashCheckText(-1, errors.New("canceled")); got != "runtime run: canceled" {
		t.Fatalf("a bare session error = %q", got)
	}
}

// last_attempt 의 reason 글자 (결정 49 · FD 규칙 14절).
func TestAttemptReason(t *testing.T) {
	cases := map[string]string{
		attemptReason(contract.ReasonIRMismatch, "x"):            "ir_mismatch",
		attemptReason("", exitReason("sync", 1)):                 "sync exited 1",
		attemptReason("", exitReason(buildLabel("config-a"), 2)): "build config-a exited 2",
		attemptReason("", "cannot verify ir: git exited 128"):    "cannot verify ir: git exited 128",
		attemptReason("", "node stopped"):                        "node stopped",
		abandonedReason(lower.PhasePending):                      "abandoned: no process held the bake while it was pending",
		abandonedReason(lower.PhaseBuilding):                     "abandoned: no process held the bake while it was building",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("reason = %q, want %q", got, want)
		}
	}
}

func TestBuildManifestOf(t *testing.T) {
	at := time.Date(2026, 9, 27, 5, 0, 0, 0, time.UTC)
	ir := "ir-1"
	sync := lower.BuildRecord{Name: "sync", Command: "git fetch", StartedAt: at, FinishedAt: at.Add(time.Minute)}
	builds := []lower.BuildRecord{{Name: "config-a", Command: "make", StartedAt: at, FinishedAt: at, ExitCode: 0}}
	m := buildManifestOf(sync, builds, headA, &ir, &lower.Pinned{File: pinnedFile, SHA256: "abc"}, []string{"ir-1"})
	if m.Sync.Command != "git fetch" || len(m.Builds) != 1 || m.Builds[0].Name != "config-a" || m.Head != headA ||
		*m.IR != ir || m.Pinned.SHA256 != "abc" || !reflect.DeepEqual(m.HeadTags, []string{"ir-1"}) {
		t.Fatalf("manifest = %+v", m)
	}
	// 대조를 못 했으면 head 는 "" · head_tags 는 null · builds 는 [] 다
	none := buildManifestOf(sync, nil, "", nil, nil, nil)
	b, _ := json.Marshal(none)
	for _, want := range []string{`"head":""`, `"head_tags":null`, `"builds":[]`, `"ir":null`, `"pinned":null`} {
		if !strings.Contains(string(b), want) {
			t.Fatalf("%s does not carry %s", b, want)
		}
	}
	// 태그가 없으면 []
	if b, _ := json.Marshal(buildManifestOf(sync, nil, headA, nil, nil, []string{})); !strings.Contains(string(b), `"head_tags":[]`) {
		t.Fatalf("no tags = %s", b)
	}
}

func TestMergeOpsOf(t *testing.T) {
	got := mergeOpsOf(merge.Result{Ops: 99, Replaced: 1, TypeChanged: 2, Added: 3, NewDirs: 4, OpaqueDirs: 5,
		Whiteouts: 6, MergedDirs: 7, Discarded: 8})
	want := contract.MergeOps{Replaced: 3, Created: 3, Dirs: 4, Opaque: 5, Whiteouts: 6, Trashed: 8, Attrs: 7}
	if got != want {
		t.Fatalf("mergeOpsOf = %+v, want %+v", got, want)
	}
	if line := mergedLine(merge.Result{Ops: 2, Added: 1, Discarded: 1}, 1500*time.Millisecond); line !=
		"bake: merged: ops 2, replaced 0, added 1, new dirs 0, opaque dirs 0, whiteouts 0, type changed 0, "+
			"merged dirs 0, discarded 1 in 1.5s" {
		t.Fatalf("merged line = %q", line)
	}
}

// fullDraft 는 칸이 모두 찬 초안이다.
func fullDraft() Draft {
	at := time.Date(2026, 9, 27, 5, 0, 0, 0, time.UTC)
	ir, prev := "ir-2", "ir-1"
	return Draft{Schema: draftSchema, Run: "run-bake", Node: "node-k",
		Source: lower.Source{URL: "https://h/x", Branch: "main", RepoID: "h/x", Head: headA, IR: &ir,
			Pinned: &lower.Pinned{File: pinnedFile, SHA256: "abc"}, SyncCommand: "git fetch", SyncedAt: at},
		Builds:      []lower.BuildRecord{{Name: "config-a", Command: "make", StartedAt: at, FinishedAt: at}},
		Environment: "prep-1", WorkspaceTarget: "/work", PreviousIR: &prev}
}

// zeroFields 는 구조체에서 빈 칸의 이름을 모은다 — 옮기다 빠진 칸을 찾는다.
func zeroFields(prefix string, v reflect.Value) []string {
	var out []string
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return []string{prefix}
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct || v.Type() == reflect.TypeOf(time.Time{}) {
		if v.IsZero() {
			out = append(out, prefix)
		}
		return out
	}
	for i := 0; i < v.NumField(); i++ {
		out = append(out, zeroFields(prefix+"."+v.Type().Field(i).Name, v.Field(i))...)
	}
	return out
}

// metadataOf 는 초안의 칸을 모두 옮기고 bake 다섯을 채운다 — source 여덟 · builds · environment · workspace_target ·
// bake (run · node · merged_at · resumed · previous_ir) (FD 규칙 3.1 · 팩 결정 3-16 · 조각 6 의 「필드가 다 있다」).
func TestMetadataOf(t *testing.T) {
	d := fullDraft()
	at := time.Date(2026, 9, 27, 6, 0, 0, 0, time.FixedZone("x", 3600))
	m := metadataOf(d, "run-bake", "node-k", at, false)
	if m.Bake.Resumed || m.Bake.Node != "node-k" || m.Bake.Run != "run-bake" || !m.Bake.MergedAt.Equal(at) ||
		m.Bake.MergedAt.Location() != time.UTC || *m.Bake.PreviousIR != "ir-1" {
		t.Fatalf("merge step metadata = %+v", m.Bake)
	}
	if z := zeroFields("m", reflect.ValueOf(m)); !reflect.DeepEqual(z, []string{"m.Schema", "m.Bake.Resumed"}) {
		t.Fatalf("empty fields = %v", z)
	}
	r := metadataOf(d, d.Run, d.Node, at, true)
	if !r.Bake.Resumed || r.Bake.Node != d.Node {
		t.Fatalf("resume metadata = %+v", r.Bake)
	}
	if z := zeroFields("r", reflect.ValueOf(r)); !reflect.DeepEqual(z, []string{"r.Schema"}) {
		t.Fatalf("empty fields = %v", z)
	}
	if !reflect.DeepEqual(r.Source, d.Source) || !reflect.DeepEqual(r.Builds, d.Builds) ||
		r.Environment != d.Environment || r.WorkspaceTarget != d.WorkspaceTarget {
		t.Fatalf("metadata did not carry the draft: %+v", r)
	}
}

// 대기 로그 (TestWaitLog) — 처음 · 모습 바뀜 · 5분 · Unnamed · 재개는 마감 줄 없음 · 4시간에 48 번 · 남은 시간의 글자.
func TestWaitLog(t *testing.T) {
	t0 := time.Date(2026, 9, 27, 5, 8, 0, 0, time.UTC)
	since := time.Date(2026, 9, 27, 4, 10, 0, 0, time.UTC)
	run := lower.Waiting{Holders: []lower.Holder{{Node: "a1", Label: "box-a", Role: lower.RoleRun, Run: "R-8790", Since: since}}}
	l := waitLog{deadline: t0.Add(3*time.Hour + 52*time.Minute), every: 5 * time.Minute}
	got := l.lines(t0, run)
	want := []string{
		"waiting for the lower lock; deadline 2026-09-27T09:00:00Z node clock (3h52m left)",
		"  node box-a holds it for run R-8790 since 2026-09-27T04:10:00Z",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("first lines = %q", got)
	}
	if got := l.lines(t0.Add(10*time.Second), run); got != nil {
		t.Fatalf("an unchanged shape wrote %q", got)
	}
	cand := lower.Waiting{Holders: append(run.Holders, lower.Holder{Node: "b1", Role: lower.RoleCandidate, Acks: 1})}
	got = l.lines(t0.Add(20*time.Second), cand)
	if len(got) != 3 || got[2] != "  node b1 holds it as a candidate; its drain was acknowledged 1 of 2 times" {
		t.Fatalf("a new holder = %q", got)
	}
	if got := l.lines(t0.Add(5*time.Minute+19*time.Second), cand); got != nil {
		t.Fatalf("wrote before five minutes: %q", got)
	}
	if got := l.lines(t0.Add(5*time.Minute+20*time.Second), cand); len(got) != 3 {
		t.Fatalf("did not write after five minutes: %q", got)
	}
	unnamed := l.lines(t0.Add(6*time.Minute), lower.Waiting{Unnamed: true})
	if len(unnamed) != 2 || unnamed[1] != "  a holder left no record (an env check smoke, or an older enode that "+
		"takes no lower lock)" {
		t.Fatalf("unnamed = %q", unnamed)
	}

	// 재개는 마감이 없다 — 머리 줄이 다르다
	r := waitLog{resume: "R-1", every: 5 * time.Minute}
	if got := r.lines(t0, run); got[0] != "waiting for the lower lock to resume the merge of run R-1" {
		t.Fatalf("resume head = %q", got[0])
	}

	// 4시간 · 10초마다 · 모습이 안 바뀌면 48 번
	four := waitLog{deadline: t0.Add(4 * time.Hour), every: 5 * time.Minute}
	heads := 0
	for at := t0; at.Before(t0.Add(4 * time.Hour)); at = at.Add(10 * time.Second) {
		if lines := four.lines(at, run); lines != nil {
			heads++
		}
	}
	if heads != 48 {
		t.Fatalf("four hours wrote %d times, want 48", heads)
	}
}

func TestLeftText(t *testing.T) {
	for d, want := range map[time.Duration]string{
		3*time.Hour + 52*time.Minute + 10*time.Second: "3h52m",
		4 * time.Hour:                         "4h0m",
		90 * time.Second:                      "2m",
		59*time.Second + 400*time.Millisecond: "59s",
		-time.Second:                          "0s",
	} {
		if got := leftText(d); got != want {
			t.Errorf("leftText(%s) = %q, want %q", d, got, want)
		}
	}
	if got := tookLine(41*time.Minute + 12*time.Second + 300*time.Millisecond); got != "took the lower lock after 41m12s" {
		t.Fatalf("took = %q", got)
	}
}

// pending_upper 의 모양 (TestPendingShape) — 키는 열다섯 자리 (러너의 fsid 는 위 네 비트가 0) 와 열여섯 자리 둘 다.
func TestPendingShape(t *testing.T) {
	for _, key := range []string{
		lower.Key{FSID: 0x35b60f8473d0c15, Ino: 2}.String(),
		lower.Key{FSID: 0xa35b60f8473d0c15, Ino: 1234567}.String(),
	} {
		t.Run(key, func(t *testing.T) {
			ok := "/home/u/scratch/pending/" + key + "/bake-123/upper"
			if s, good := pendingShape(ok, key); !good || s != "/home/u/scratch" {
				t.Fatalf("pendingShape(%q) = %q, %v", ok, s, good)
			}
			for _, bad := range []string{
				"/home/u/scratch/pending/1-2/bake-123/upper",              // 다른 키
				"home/u/scratch/pending/" + key + "/bake-123/upper",       // 상대 경로
				"/home/u/scratch/pending/" + key + "/bake-123/lower",      // upper 가 아닌 끝
				"/home/u/scratch/pending/" + key + "/../x/bake-123/upper", // .. 이 든 경로
				"/home/u/scratch/pending/" + key + "//bake-123/upper",     // Clean 이 아닌 글자
				"/home/u/scratch/pending/" + key + "/bake-123/upper/",     // Clean 이 아닌 글자
				"/home/u/scratch/elsewhere/" + key + "/bake-123/upper",    // pending 이 아닌 조각
				"/pending/" + key + "/bake-123/upper",                     // scratch 가 뿌리
				"/home/u/scratch/pending/" + key + "/upper",               // 이름이 없다
				"",
			} {
				if s, good := pendingShape(bad, key); good {
					t.Errorf("pendingShape(%q) = %q, true", bad, s)
				}
			}
		})
	}
}

func TestStripPassword(t *testing.T) {
	for in, want := range map[string]string{
		"https://u:p@h/x":         "https://u@h/x",
		"https://u@h/x":           "https://u@h/x",
		"https://h/x":             "https://h/x",
		"ssh://git:secret@h:22/x": "ssh://git@h:22/x",
		"git@h:x.git":             "git@h:x.git",
		"/srv/mirror/x.git":       "/srv/mirror/x.git",
		"":                        "",
		"https://u:p@h/%zz":       "",
		"not a scheme://u:p@h/x":  "not a scheme://u:p@h/x",
	} {
		if got := stripPassword(in); got != want {
			t.Errorf("stripPassword(%q) = %q, want %q", in, got, want)
		}
	}
}

// repoIDOf 는 DetectRepo 의 규칙을 대조 출력에 옮긴다 — 부르지는 않는다.
func TestRepoIDOf(t *testing.T) {
	cases := []struct {
		p    irProbe
		want string
	}{
		{irProbe{Mode: "repo", URL: "https://h.example/platform/manifest.git", Branch: "stable"},
			CanonicalRepoID("https://h.example/platform/manifest.git") + "#stable"},
		{irProbe{Mode: "repo", URL: "https://h.example/platform/manifest.git"},
			CanonicalRepoID("https://h.example/platform/manifest.git")},
		{irProbe{Mode: "git", URL: "https://u:p@h.example/x.git", Branch: "main"}, CanonicalRepoID("https://h.example/x.git")},
		{irProbe{Mode: "git", Branch: "main"}, ""},
	}
	for _, tc := range cases {
		if got := repoIDOf(tc.p); got != tc.want || (tc.want != "" && got == "") {
			t.Errorf("repoIDOf(%+v) = %q, want %q", tc.p, got, tc.want)
		}
	}
}

func TestHelperErrorOf(t *testing.T) {
	if err := helperErrorOf(mergeHelperResponse{Result: &merge.Result{}}); err != nil {
		t.Fatalf("no error = %v", err)
	}
	var pe *helperPreflightError
	err := helperErrorOf(mergeHelperResponse{Error: "merge preflight: upper entry a carries user.overlay.metacopy",
		Check: "mark", Kind: "preflight"})
	if !errors.As(err, &pe) || pe.Check != merge.CheckMark ||
		err.Error() != "merge preflight: upper entry a carries user.overlay.metacopy" {
		t.Fatalf("preflight = %#v", err)
	}
	var oe *helperOpError
	if err := helperErrorOf(mergeHelperResponse{Error: "merge: rename a: boom", Kind: "op"}); !errors.As(err, &oe) ||
		err.Error() != "merge: rename a: boom" {
		t.Fatalf("op = %#v", err)
	}
	io := helperErrorOf(mergeHelperResponse{Error: "merge preflight: read a: permission denied", Kind: "io"})
	if io == nil || errors.As(io, &pe) || errors.As(io, &oe) {
		t.Fatalf("io = %#v", io)
	}

	// helper 쪽의 짝 — 갈래를 같은 글자로 옮긴다
	for _, tc := range []struct {
		err  error
		kind string
	}{
		{&merge.PreflightError{Check: merge.CheckMount, Err: errors.New("x")}, "preflight"},
		{&merge.OpError{Op: merge.Op{Call: merge.CallRename, Path: "a"}, Err: errors.New("boom")}, "op"},
		{errors.New("merge preflight: read .: EIO"), "io"},
	} {
		r := helperResponseOf(nil, tc.err)
		if r.Kind != tc.kind || r.Error != tc.err.Error() {
			t.Errorf("helperResponseOf(%v) = %+v", tc.err, r)
		}
		back := helperErrorOf(r)
		if back.Error() != tc.err.Error() {
			t.Errorf("round trip = %v", back)
		}
	}
	if r := helperResponseOf(&merge.Result{Ops: 3}, nil); r.Error != "" || r.Result.Ops != 3 {
		t.Fatalf("a clean apply = %+v", r)
	}
}

// 끝났나 — bake.run 과 synced_at · head 가 모두 같을 때만. run_id 만 같으면 끝나지 않은 것이다 (FD 규칙 12.4 끝).
func TestFinished(t *testing.T) {
	d := fullDraft()
	md := metadataOf(d, d.Run, d.Node, time.Now(), true)
	if !finished(&md, d.Run, d) {
		t.Fatal("the same bake reads as unfinished")
	}
	if finished(nil, d.Run, d) {
		t.Fatal("no metadata reads as finished")
	}
	other := md
	other.Source.SyncedAt = d.Source.SyncedAt.Add(time.Second)
	if finished(&other, d.Run, d) {
		t.Fatal("a rewritten run_id with another sync reads as finished")
	}
	other = md
	other.Source.Head = headB
	if finished(&other, d.Run, d) {
		t.Fatal("another HEAD reads as finished")
	}
	if finished(&md, "other-run", d) {
		t.Fatal("another run reads as finished")
	}
}

// merge claim 의 표 (FD 규칙 7.1) — 위에서부터 처음 맞는 줄.
func TestMergeClaim(t *testing.T) {
	owner := func(run string) *lower.Owner { return &lower.Owner{Run: run, Node: "k"} }
	discarded := &lower.LastAttempt{Run: "R", Reason: abandonedReason(lower.PhasePending)}
	cases := []struct {
		name string
		held bool
		st   lower.State
		want claimVerdict
	}{
		{"held and pending by this run", true, lower.State{Phase: lower.PhasePending, Owner: owner("R")}, claimProceed},
		{"held but merging", true, lower.State{Phase: lower.PhaseMerging, Owner: owner("R")}, claimMismatch},
		{"held but another run", true, lower.State{Phase: lower.PhasePending, Owner: owner("S")}, claimMismatch},
		{"held but committed", true, lower.State{Phase: lower.PhaseCommitted}, claimMismatch},
		{"restarted while pending", false, lower.State{Phase: lower.PhasePending, Owner: owner("R")}, claimRestartLeft},
		{"restarted while merging", false, lower.State{Phase: lower.PhaseMerging, Owner: owner("R")}, claimRestartLeft},
		{"discarded while pending", false, lower.State{Phase: lower.PhaseCommitted, LastAttempt: discarded},
			claimRestartDiscarded},
		// 다른 굽기가 building 을 써도 last_attempt 는 남는다 (결정 49)
		{"discarded, another bake building", false, lower.State{Phase: lower.PhaseBuilding, Owner: owner("S"),
			LastAttempt: discarded}, claimRestartDiscarded},
		// 재시작 줄이 합칠 것 없음보다 먼저다 — 주인이 이 Run 인 pending 은 last_attempt 가 무엇이든 left
		{"left before discarded", false, lower.State{Phase: lower.PhasePending, Owner: owner("R"),
			LastAttempt: discarded}, claimRestartLeft},
		{"a command failed", false, lower.State{Phase: lower.PhaseCommitted,
			LastAttempt: &lower.LastAttempt{Run: "R", Reason: "sync exited 1"}}, claimNothing},
		{"ir mismatch", false, lower.State{Phase: lower.PhaseCommitted,
			LastAttempt: &lower.LastAttempt{Run: "R", Reason: contract.ReasonIRMismatch}}, claimNothing},
		{"another bake", false, lower.State{Phase: lower.PhasePending, Owner: owner("S")}, claimNothing},
		// 재시작 줄은 pending 갈래뿐이다 (결정 52)
		{"building by this run", false, lower.State{Phase: lower.PhaseBuilding, Owner: owner("R")}, claimNothing},
		{"a building was cleaned", false, lower.State{Phase: lower.PhaseCommitted,
			LastAttempt: &lower.LastAttempt{Run: "R", Reason: abandonedReason(lower.PhaseBuilding)}}, claimNothing},
		{"another run's discarded pending", false, lower.State{Phase: lower.PhaseCommitted,
			LastAttempt: &lower.LastAttempt{Run: "S", Reason: abandonedReason(lower.PhasePending)}}, claimNothing},
		{"never baked", false, lower.State{Phase: lower.PhaseCommitted}, claimNothing},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := mergeClaim(tc.held, tc.st, "R"); got != tc.want {
				t.Fatalf("mergeClaim = %v, want %v", got, tc.want)
			}
		})
	}
}

// 단계 보고와 단계 로그의 글자 (FD 규칙 16절).
func TestBakeTexts(t *testing.T) {
	since := time.Date(2026, 9, 27, 4, 10, 0, 0, time.UTC)
	cases := map[string]string{
		bakeInProgressText(lower.State{Phase: lower.PhasePending, Since: since,
			Owner: &lower.Owner{Run: "R-1", Node: "node-k"}}): "another bake holds this lower: run R-1 on node node-k " +
			"since 2026-09-27T04:10:00Z (pending); submit the bake again later",
		bakeInProgressText(lower.State{Phase: lower.PhaseCommitted}): "another bake holds this lower: run unknown " +
			"(committed); submit the bake again later",
		resumeOpenedText("R-1"): "an interrupted merge of run R-1 is left on this lower; this node started resuming " +
			"it; submit the bake again after it finishes",
		mismatchText(lower.State{Phase: lower.PhaseMerging, Owner: &lower.Owner{Run: "R-1"}}): "the lower state " +
			"does not match this bake: merging (run R-1)",
		mismatchText(lower.State{Phase: lower.PhaseCommitted}): "the lower state does not match this bake: committed " +
			"(run unknown)",
		mergeWaitText(contract.DefaultMergeWait): "gave up waiting for the lower lock after 4h0m0s (merge.wait)",
		mergeWaitText(2 * time.Second):           "gave up waiting for the lower lock after 2s (merge.wait)",
		mergeStoppedText(errors.New("merge: rename a: boom")): "merge stopped: merge: rename a: boom; the lower " +
			"stays merging and a node on this lower resumes it",
		cannotOpenState(errors.New("lower: /x: permission denied")): "cannot open the lower state directory: " +
			"lower: /x: permission denied",
		pinFailedText(1, "error: no manifest"): "cannot pin the manifest: repo manifest -r exited 1: error: no manifest",
		startedLine("sync"):                    "bake: sync started",
		exitedLine("build config-a", 0, 41*time.Minute+45*time.Second+400*time.Millisecond): "bake: build config-a " +
			"exited 0 after 41m45s",
		skipLine("config-b", "config-a failed"): "bake: skipping config-b; config-a failed",
		refusedText: "a bake step needs a node that writes to an upper (workspace.writes=isolated); this node " +
			"writes to its workspace in place",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("text = %q\nwant  %q", got, want)
		}
	}
}
