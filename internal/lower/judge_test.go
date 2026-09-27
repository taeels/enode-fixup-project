package lower

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// 판정의 표 시험 — 모든 플랫폼에서 돈다.

// 키 글자는 `stat -f -c %i` 와 같은 차례다 — Val[0] 이 위 32비트. Val 은 int32 라 음수가 올 수 있다.
func TestKeyString(t *testing.T) {
	hi, lo := uint32(0x3b167980), uint32(0xfdd76b7b)
	for _, c := range []struct {
		val0, val1 int32
		ino        uint64
		want       string
	}{
		{int32(hi), int32(lo), 2905094, "3b167980fdd76b7b-2905094"},
		{int32(lo), int32(hi), 1, "fdd76b7b3b167980-1"},
		{-1, -1, 0, "ffffffffffffffff-0"},
		{0, 1, 18446744073709551615, "0000000000000001-18446744073709551615"},
	} {
		k := keyOf(c.val0, c.val1, c.ino)
		if got := k.String(); got != c.want {
			t.Errorf("keyOf(%d, %d, %d) = %s, want %s", c.val0, c.val1, c.ino, got, c.want)
		}
		back, err := ParseKey(c.want)
		if err != nil || back != k {
			t.Errorf("ParseKey(%s) = %v, %v; want %v", c.want, back, err, k)
		}
	}
	if got := keyOf(int32(hi), int32(lo), 1).fsidString(); got != "3b167980fdd76b7b" {
		t.Errorf("fsidString = %s", got)
	}
}

func TestParseKeyRejects(t *testing.T) {
	for _, s := range []string{
		"", "3b167980fdd76b7b", "3b167980fdd76b7b-", "3B167980FDD76B7B-1", "3b167980fdd76b7-1",
		"3b167980fdd76b7bb-1", "zzzzzzzzzzzzzzzz-1", "3b167980fdd76b7b-01", "3b167980fdd76b7b-+1",
		"3b167980fdd76b7b--1", "3b167980fdd76b7b-1-2", "3b167980fdd76b7b-99999999999999999999",
	} {
		if k, err := ParseKey(s); err == nil {
			t.Errorf("ParseKey(%q) = %v, want an error", s, k)
		} else if !strings.Contains(err.Error(), "is not a state directory name") {
			t.Errorf("ParseKey(%q) error = %v", s, err)
		}
	}
}

func TestDevice(t *testing.T) {
	for _, c := range []struct {
		major, minor uint32
		want         string
	}{
		{8, 1, "8:1"}, {252, 15, "252:15"}, {0, 0, "0:0"}, {4095, 255, "4095:255"},
		{4096, 256, "4096:256"}, {1 << 20, 1 << 20, "1048576:1048576"},
	} {
		if got := devString(mkdev(c.major, c.minor)); got != c.want {
			t.Errorf("devString(mkdev(%d, %d)) = %s, want %s", c.major, c.minor, got, c.want)
		}
	}
	// linux 의 st_dev 와 같은 값이다 — 8:1 은 0x801
	if got := mkdev(8, 1); got != 0x801 {
		t.Errorf("mkdev(8, 1) = %#x, want 0x801", got)
	}
}

func TestValidNode(t *testing.T) {
	for id, want := range map[string]bool{
		"0a1b2c": true, "node-1": true, strings.Repeat("a", 128): true,
		"": false, "Node": false, "a_b": false, "a/b": false, "../x": false, "a.b": false,
		strings.Repeat("a", 129): false,
	} {
		if got := validNode(id); got != want {
			t.Errorf("validNode(%q) = %v, want %v", id, got, want)
		}
	}
}

// 신원 판정 다섯 (business-rules.md 2절). btime 이 한쪽이라도 0 이면 대조하지 않는다.
func TestIdentityVerdict(t *testing.T) {
	root := Root{Path: "/w", Key: Key{FSID: 0x3b167980fdd76b7b, Ino: 7}, BirthNs: 100, UID: 1000}
	rec := func(edit func(*Identity)) *Identity {
		r := &Identity{Schema: 1, FSID: "3b167980fdd76b7b", Ino: 7, BirthNs: 100}
		if edit != nil {
			edit(r)
		}
		return r
	}
	zeroRoot := root
	zeroRoot.BirthNs = 0
	for _, c := range []struct {
		name  string
		rec   *Identity
		root  Root
		phase Phase
		want  Verdict
	}{
		{"no file", nil, root, PhaseCommitted, VerdictNew},
		{"same", rec(nil), root, PhaseMerging, VerdictMatch},
		{"recorded btime zero", rec(func(r *Identity) { r.BirthNs = 0 }), root, PhasePending, VerdictMatch},
		{"root btime zero", rec(func(r *Identity) { r.BirthNs = 5 }), zeroRoot, PhasePending, VerdictMatch},
		{"reused committed", rec(func(r *Identity) { r.BirthNs = 5 }), root, PhaseCommitted, VerdictReused},
		{"foreign pending", rec(func(r *Identity) { r.BirthNs = 5 }), root, PhasePending, VerdictForeign},
		{"foreign building", rec(func(r *Identity) { r.BirthNs = 5 }), root, PhaseBuilding, VerdictForeign},
		{"foreign merging", rec(func(r *Identity) { r.BirthNs = 5 }), root, PhaseMerging, VerdictForeign},
		{"other fsid", rec(func(r *Identity) { r.FSID = "fdd76b7b3b167980" }), root, PhaseCommitted, VerdictBroken},
		{"other ino", rec(func(r *Identity) { r.Ino = 8 }), root, PhaseCommitted, VerdictBroken},
		{"other schema", rec(func(r *Identity) { r.Schema = 2 }), root, PhaseCommitted, VerdictBroken},
	} {
		if got := identityVerdict(c.rec, c.root, c.phase); got != c.want {
			t.Errorf("%s: verdict %d, want %d", c.name, got, c.want)
		}
	}
}

// 권한 표 (계획 3.1) — 종류 · 주인 · 비트 차례.
func TestPermVerdict(t *testing.T) {
	const uid = 1000
	for _, c := range []struct {
		name  string
		mode  uint32
		owner int
		dir   bool
		want  permResult
	}{
		{"dir 0700", modeDir | 0o700, uid, true, permOK},
		{"dir 0755", modeDir | 0o755, uid, true, permLoose},
		{"dir 0701", modeDir | 0o701, uid, true, permLoose},
		{"dir 0500", modeDir | 0o500, uid, true, permOK},
		{"dir of another user", modeDir | 0o700, 0, true, permForeignOwner},
		{"loose dir of another user", modeDir | 0o777, 0, true, permForeignOwner},
		{"file where a dir is wanted", modeRegular | 0o700, uid, true, permWrongKind},
		{"symlink where a dir is wanted", 0o120000 | 0o777, uid, true, permWrongKind},
		{"file 0600", modeRegular | 0o600, uid, false, permOK},
		{"file 0644", modeRegular | 0o644, uid, false, permLoose},
		{"file 0620", modeRegular | 0o620, uid, false, permLoose},
		{"file of another user", modeRegular | 0o600, 1001, false, permForeignOwner},
		{"dir where a file is wanted", modeDir | 0o600, uid, false, permWrongKind},
		{"fifo where a file is wanted", 0o010000 | 0o600, uid, false, permWrongKind},
	} {
		if got := permVerdict(c.mode, c.owner, uid, c.dir); got != c.want {
			t.Errorf("%s: %d, want %d", c.name, got, c.want)
		}
	}
	for _, c := range []struct {
		r    permResult
		dir  bool
		want string
	}{
		{permWrongKind, true, "lower: /p: not a directory"},
		{permWrongKind, false, "lower: /p: not a regular file"},
		{permForeignOwner, true, "lower: /p: owned by uid 0, not the node user 1000"},
	} {
		if err := permError("/p", c.r, 0, uid, c.dir); err == nil || err.Error() != c.want {
			t.Errorf("permError(%d) = %v, want %s", c.r, err, c.want)
		}
	}
	if err := permError("/p", permLoose, uid, uid, true); err != nil {
		t.Errorf("a loose place is not an error: %v", err)
	}
}

// mountinfo 의 모양 (proc(5)) — 선택 칸 · 구분자 · 8진 탈출.
func TestParseMountinfo(t *testing.T) {
	b := []byte(`22 1 252:1 / / rw,relatime shared:1 - ext4 /dev/vda1 rw
36 22 252:1 /home/u/my\040ws /work rw,relatime shared:2 master:1 - ext4 /dev/vda1 rw
40 22 0:50 / /tmp\011x rw - tmpfs tmp\134fs rw,size=10k

`)
	ls, err := parseMountinfo(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(ls) != 3 {
		t.Fatalf("parsed %d lines, want 3", len(ls))
	}
	if l := ls[1]; l.ID != 36 || l.Parent != 22 || l.Dev != mkdev(252, 1) || l.Root != "/home/u/my ws" ||
		l.Point != "/work" || l.FSType != "ext4" || l.Source != "/dev/vda1" || l.Super != "rw" {
		t.Errorf("line 2 = %+v", l)
	}
	if l := ls[2]; l.Point != "/tmp\tx" || l.Source != `tmp\fs` || l.Super != "rw,size=10k" {
		t.Errorf("line 3 = %+v", l)
	}
	for _, bad := range []string{
		"22 1 252:1 / / rw", "x 1 252:1 / / rw - ext4 /dev/vda1 rw", "22 1 252 / / rw - ext4 /dev/vda1 rw",
		"22 1 252:1 / / rw shared:1 ext4 /dev/vda1 rw", "22 1 252:1 / / rw - ext4",
	} {
		if _, err := parseMountinfo([]byte(bad)); err == nil || !strings.Contains(err.Error(), "malformed") {
			t.Errorf("parseMountinfo(%q) = %v, want malformed", bad, err)
		}
	}
	// super options 가 없는 줄도 받는다
	if ls, err := parseMountinfo([]byte("1 0 0:1 / / rw - rootfs rootfs")); err != nil || ls[0].Super != "" {
		t.Errorf("a line without super options = %v, %v", ls, err)
	}
	if got := unescapeMount(`a\040b\134c\054d\0`); got != `a b\c,d\0` {
		t.Errorf("unescapeMount = %q", got)
	}
}

// 이 lower 에 닿는 overlay 를 찾는 표 (business-logic-model.md 9.1). lower 는 장치 252:15 의 /home/u/ws 다.
func TestOverlaysOn(t *testing.T) {
	dev := mkdev(252, 15)
	const host = "22 1 252:15 / / rw - ext4 /dev/vda2 rw\n"
	// helper 모양 — 워크스페이스를 lower-ro 에 bind 하고 그 자리를 lowerdir 로 overlay
	const helper = host +
		"600 22 252:15 /home/u/ws /s/run/lower-ro ro - ext4 /dev/vda2 rw\n" +
		"601 22 0:77 / /s/run/merged rw - overlay overlay rw,lowerdir=/s/run/lower-ro,upperdir=/s/run/upper,workdir=/s/run/work,userxattr\n"
	for _, c := range []struct {
		name, mountinfo, want string
	}{
		{"helper bind and overlay", helper, "/s/run/merged"},
		{"host bind alias only", host + "36 22 252:15 /home/u/ws /work rw - ext4 /dev/vda2 rw\n", ""},
		{"lowerdir is the parent of the lower", host +
			"601 22 0:77 / /m rw - overlay overlay rw,lowerdir=/home/u,upperdir=/x/u,workdir=/x/w\n", "/m"},
		{"lowerdir is inside the lower", host +
			"601 22 0:77 / /m rw - overlay overlay rw,lowerdir=/home/u/ws/sub,upperdir=/x/u,workdir=/x/w\n", "/m"},
		{"a sibling of the lower", host +
			"601 22 0:77 / /m rw - overlay overlay rw,lowerdir=/home/u/ws2,upperdir=/x/u,workdir=/x/w\n", ""},
		{"second of many lowerdirs", host +
			"601 22 0:77 / /m rw - overlay overlay rw,lowerdir=/opt/a:/home/u/ws,upperdir=/x/u,workdir=/x/w\n", "/m"},
		{"escaped space", host +
			"600 22 252:15 /home/u/ws /s/my\\040ro ro - ext4 /dev/vda2 rw\n" +
			"601 22 0:77 / /m rw - overlay overlay rw,lowerdir=/s/my\\040ro,upperdir=/x/u,workdir=/x/w\n", "/m"},
		{"lowerdir+=", host +
			"601 22 0:77 / /m rw - overlay overlay rw,lowerdir+=/opt/a,lowerdir+=/home/u/ws,upperdir=/x/u,workdir=/x/w\n", "/m"},
		{"datadir+=", host +
			"601 22 0:77 / /m rw - overlay overlay rw,lowerdir+=/opt/a,datadir+=/home/u/ws/data,upperdir=/x/u,workdir=/x/w\n", "/m"},
		{"same path on another device", "22 1 252:16 / / rw - ext4 /dev/vda3 rw\n" +
			"601 22 0:77 / /m rw - overlay overlay rw,lowerdir=/home/u/ws,upperdir=/x/u,workdir=/x/w\n", ""},
		{"container rootfs overlay", host +
			"700 22 0:80 / /c rw - overlay overlay rw,lowerdir=/var/lib/rootfs,upperdir=/x/u,workdir=/x/w\n", ""},
	} {
		ls, err := parseMountinfo([]byte(c.mountinfo))
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		found := overlaysOn(ls, dev, "/home/u/ws")
		switch {
		case c.want == "" && len(found) != 0:
			t.Errorf("%s: found %+v, want none", c.name, found)
		case c.want != "" && (len(found) != 1 || found[0].MountPoint != c.want):
			t.Errorf("%s: found %+v, want %s", c.name, found, c.want)
		}
	}
	if got := splitColons(`a\:b:c::d:`); fmt.Sprint(got) != "[a:b c d]" {
		t.Errorf("splitColons = %q", got)
	}
}

func TestLongestMount(t *testing.T) {
	ls, err := parseMountinfo([]byte("1 0 8:1 / / rw - ext4 a rw\n2 1 8:2 / /home rw - ext4 b rw\n" +
		"3 2 8:3 / /home/u rw - ext4 c rw\n4 2 8:4 / /home/u rw - ext4 d rw\n"))
	if err != nil {
		t.Fatal(err)
	}
	for p, want := range map[string]uint64{"/": 1, "/etc": 1, "/home": 2, "/homes": 1, "/home/u/x": 4, "/home/v": 2} {
		if m, ok := longestMount(ls, p); !ok || m.ID != want {
			t.Errorf("longestMount(%s) = %d, want %d", p, m.ID, want)
		}
	}
	if _, ok := longestMount(ls[1:], "/etc"); ok {
		t.Error("a path outside every mount was found")
	}
	if m, ok := mountByID(ls, 3); !ok || m.Point != "/home/u" {
		t.Errorf("mountByID(3) = %+v", m)
	}
	for _, id := range []uint64{0, 9} {
		if _, ok := mountByID(ls, id); ok {
			t.Errorf("mountByID(%d) was found", id)
		}
	}
	if got := inFS(mountLine{Root: "/vol", Point: "/"}, "/a/b"); got != "/vol/a/b" {
		t.Errorf("inFS under / = %s", got)
	}
	if got := inFS(mountLine{Root: "/home/u/ws", Point: "/work"}, "/work/x"); got != "/home/u/ws/x" {
		t.Errorf("inFS under a bind = %s", got)
	}
}

// 점검 셋의 판정과 문구 (business-rules.md 12.3).
func TestFindings(t *testing.T) {
	ws := Root{Dev: mkdev(252, 15), MountID: 30, UID: 1000}
	same := ws
	sameDevOtherMount := Root{Dev: ws.Dev, MountID: 31}
	otherDev := Root{Dev: mkdev(0, 50), MountID: 40}
	unknownMount := Root{Dev: ws.Dev}
	for _, c := range []struct {
		name     string
		scratch  Root
		err      error
		ok       bool
		observed string
	}{
		{"same mount", same, nil, true, "same mount"},
		{"mount unknown", unknownMount, nil, true, "same mount"},
		{"other device", otherDev, nil, false, "different filesystem (scratch dev 0:50, workspace dev 252:15)"},
		{"bind alias", sameDevOtherMount, nil, false,
			"same filesystem, different mount (scratch mount 31, workspace mount 30); is the workspace a bind alias?"},
		{"unreadable", Root{}, errors.New("boom"), false, "cannot read the scratch: boom"},
	} {
		f := scratchFinding(c.scratch, c.err, ws)
		if f.Name != "binding.scratch_filesystem" || f.OK != c.ok || f.Observed != c.observed ||
			f.Required != "same filesystem and mount as the workspace" {
			t.Errorf("%s: %+v", c.name, f)
		}
		if !c.ok && (f.Cause != CauseBinding || !strings.Contains(f.Remediation, "a bind alias of the workspace is a different mount")) {
			t.Errorf("%s: cause %s remediation %q", c.name, f.Cause, f.Remediation)
		}
	}

	if f := ownerFinding(ws, 1000); !f.OK || f.Observed != "uid 1000" || f.Required != "uid 1000 (the node user)" {
		t.Errorf("owner same = %+v", f)
	}
	if f := ownerFinding(ws, 1001); f.OK || f.Cause != CauseBinding || f.Observed != "uid 1000" ||
		f.Remediation != "run this node as the owner of the workspace; one lower is shared by one user" {
		t.Errorf("owner other = %+v", f)
	}

	un := workspaceUnread(1000, errors.New("lower: lstat /w: no such file or directory"))
	if len(un) != 2 || un[0].OK || un[1].OK || un[0].Cause != CauseBinding || un[1].Cause != CauseBinding ||
		un[0].Observed != "cannot read the workspace: lower: lstat /w: no such file or directory" {
		t.Errorf("workspaceUnread = %+v", un)
	}

	const dir = "/h/lowers/k"
	pending := State{Phase: PhasePending, Owner: &Owner{Run: "R-1"}}
	for _, c := range []struct {
		name     string
		v        Verdict
		st       State
		err      error
		loose    bool
		ok       bool
		observed string
	}{
		{"new", VerdictNew, State{}, nil, false, true, "not recorded yet; the node records it on start"},
		{"match", VerdictMatch, State{}, nil, false, true, "matches"},
		{"match loose", VerdictMatch, State{}, nil, true, true, "matches; the node narrows loose permissions on start"},
		{"reused", VerdictReused, State{}, nil, false, true,
			"recorded for another directory with the same inode; the node rewrites it on start"},
		{"foreign", VerdictForeign, pending, nil, false, false, "recorded for another directory and a bake is pending (run R-1)"},
		{"foreign without owner", VerdictForeign, State{Phase: PhaseMerging}, nil, false, false,
			"recorded for another directory and a bake is merging"},
		{"broken key", VerdictBroken, State{}, nil, false, false, "cannot read /h/lowers/k/lower.json: it does not match its key"},
		{"unreadable file", VerdictBroken, State{}, &pathError{Path: dir + "/state.json", Err: errors.New("unexpected end of JSON input")},
			false, false, "cannot read /h/lowers/k/state.json: unexpected end of JSON input"},
		{"other error", VerdictBroken, State{}, errors.New("boom"), false, false, "cannot read /h/lowers/k: boom"},
	} {
		f := identityFinding(dir, c.v, c.st, c.err, c.loose)
		if f.Name != "lower.identity" || f.OK != c.ok || f.Observed != c.observed || f.Required != "the recorded identity of this lower" {
			t.Errorf("%s: %+v", c.name, f)
		}
		if !c.ok && (f.Cause != CauseState || f.Remediation != "inspect /h/lowers/k; remove it only when no bake of that directory is left") {
			t.Errorf("%s: cause %s remediation %q", c.name, f.Cause, f.Remediation)
		}
		if c.ok && (f.Cause != "" || f.Remediation != "") {
			t.Errorf("%s: a ready finding carries %+v", c.name, f)
		}
	}
}

func TestCheckStateAndMetadata(t *testing.T) {
	for _, c := range []struct {
		st   State
		want string
	}{
		{State{Schema: 1, Phase: PhaseCommitted}, ""},
		{State{Schema: 1, Phase: PhaseMerging}, ""},
		{State{Schema: 2, Phase: PhaseCommitted}, "schema 2 is not 1"},
		{State{Schema: 1, Phase: "done"}, `unknown phase "done"`},
		{State{Schema: 1}, `unknown phase ""`},
	} {
		err := checkState(c.st)
		if (c.want == "") != (err == nil) || err != nil && err.Error() != c.want {
			t.Errorf("checkState(%+v) = %v, want %q", c.st, err, c.want)
		}
	}
	if err := checkMetadata(&Metadata{Schema: 1}); err != nil {
		t.Error(err)
	}
	if err := checkMetadata(&Metadata{Schema: 0}); err == nil || err.Error() != "schema 0 is not 1" {
		t.Errorf("schema 0 = %v", err)
	}
	var none *Metadata
	if m := none.Marker(); m != (Marker{}) {
		t.Errorf("nil metadata marker = %+v", m)
	}
	at := time.Date(2026, 9, 27, 1, 2, 3, 0, time.UTC)
	if m := (&Metadata{Bake: BakeRecord{Run: "R-1", MergedAt: at}}).Marker(); m != (Marker{Run: "R-1", MergedAt: at}) {
		t.Errorf("marker = %+v", m)
	}
}

// 문구 — business-rules.md 13절의 internal/lower 줄.
func TestErrorMessages(t *testing.T) {
	if ErrUnsupported.Error() != "lower is supported on linux only" {
		t.Errorf("ErrUnsupported = %q", ErrUnsupported)
	}
	pe := &pathError{Path: "/p/lower.lock", Err: errNotRegular}
	if pe.Error() != "lower: /p/lower.lock: not a regular file" || !errors.Is(pe, errNotRegular) {
		t.Errorf("pathError = %q", pe)
	}
	if (&Shared{}).Recorded() || (*Shared)(nil).Recorded() {
		t.Error("a shared lock without a record says it is recorded")
	}
}
