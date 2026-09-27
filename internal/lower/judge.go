package lower

import (
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"
)

// 판정이다 — 시스템 호출 없이 모든 플랫폼에서 빌드된다 (unit-of-work.md 10절). 값을 읽는 쪽은 _linux.go 다.

// keyOf 는 statfs 의 f_fsid 두 반쪽과 inode 에서 키를 짓는다. Val[0] 이 위 32비트다 — 이 차례여야
// `stat -f -c %i` 와 같은 글자가 된다 (반대로 붙이면 fdd76b7b3b167980 처럼 뒤집힌다).
func keyOf(val0, val1 int32, ino uint64) Key {
	return Key{FSID: uint64(uint32(val0))<<32 | uint64(uint32(val1)), Ino: ino}
}

// String 은 상태 자리의 이름이다 — "3b167980fdd76b7b-2905094" (16진 fsid · 10진 inode). 앞의 0 을 찍지 않는다 —
// `stat -f -c %i` 도 안 찍는다. 위 네 비트가 0 인 fsid 는 열다섯 자리 이하다.
func (k Key) String() string { return fmt.Sprintf("%x-%d", k.FSID, k.Ino) }

// fsidString 은 lower.json 의 fsid 글자다.
func (k Key) fsidString() string { return fmt.Sprintf("%x", k.FSID) }

// ParseKey 는 자리 이름을 키로 푼다. String 이 내는 글자만 받는다 — 대문자 · 앞의 0 · 부호는 거절한다.
func ParseKey(s string) (Key, error) {
	fsid, ino, ok := strings.Cut(s, "-")
	if !ok || fsid == "" || len(fsid) > 16 {
		return Key{}, fmt.Errorf("lower: %q is not a state directory name (<hex fsid>-<inode>)", s)
	}
	f, err := strconv.ParseUint(fsid, 16, 64)
	if err != nil {
		return Key{}, fmt.Errorf("lower: %q is not a state directory name: %w", s, err)
	}
	i, err := strconv.ParseUint(ino, 10, 64)
	if err != nil {
		return Key{}, fmt.Errorf("lower: %q is not a state directory name: %w", s, err)
	}
	k := Key{FSID: f, Ino: i}
	if k.String() != s {
		return Key{}, fmt.Errorf("lower: %q is not a state directory name; want %q", s, k.String())
	}
	return k, nil
}

// mkdev 는 major:minor 를 linux 의 st_dev 로 붙인다 (glibc 의 makedev · unix.Mkdev 와 같다).
func mkdev(major, minor uint32) uint64 {
	return uint64(major&0x00000fff)<<8 | uint64(major&0xfffff000)<<32 |
		uint64(minor&0x000000ff) | uint64(minor&0xffffff00)<<12
}

// devString 은 st_dev 를 사람이 읽는 major:minor 로 쓴다.
func devString(dev uint64) string {
	major := uint32((dev&0x00000000000fff00)>>8) | uint32((dev&0xfffff00000000000)>>32)
	minor := uint32(dev&0x00000000000000ff) | uint32((dev&0x00000ffffff00000)>>12)
	return fmt.Sprintf("%d:%d", major, minor)
}

// validNode 는 node_id 가 파일 이름이 될 수 있는가다 — [0-9a-z-] 만 (business-rules.md 7절).
func validNode(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for _, r := range id {
		if !nodeRune(r) {
			return false
		}
	}
	return true
}

func nodeRune(r rune) bool { return r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r == '-' }

// ── 신원 ────────────────────────────────────────────────────────────────

var errNotItsKey = errors.New("it does not match its key")

// matchesKey 는 lower.json 의 fsid · ino 가 자리 이름(키)과 같은가다.
func (rec *Identity) matchesKey(k Key) bool {
	return rec.Schema == schema && rec.FSID == k.fsidString() && rec.Ino == k.Ino
}

// identityVerdict 는 lower.json 과 지금 읽은 뿌리를 댄다 (business-rules.md 2절). rec 가 nil 이면 파일이
// 없다. 못 읽은 파일은 부르는 쪽이 Broken 으로 친다. phase 는 btime 이 다를 때만 본다.
func identityVerdict(rec *Identity, root Root, phase Phase) Verdict {
	switch {
	case rec == nil:
		return VerdictNew
	case !rec.matchesKey(root.Key):
		return VerdictBroken
	case rec.BirthNs == 0 || root.BirthNs == 0 || rec.BirthNs == root.BirthNs:
		return VerdictMatch
	case phase == PhaseCommitted:
		return VerdictReused
	}
	return VerdictForeign
}

// ── 권한 (계획 3.1) ──────────────────────────────────────────────────────

// st_mode 의 종류 비트. linux 의 값이다 — 이 판정을 부르는 것은 linux 파일뿐이다.
const (
	modeType    = 0o170000
	modeDir     = 0o040000
	modeRegular = 0o100000
	modeLoose   = 0o077 // 남에게 열린 비트 — 좁힌다
)

type permResult int

const (
	permOK           permResult = iota
	permLoose                   // 남에게 열린 비트가 있다 — 데몬은 좁히고 점검은 알린다
	permWrongKind               // 디렉터리여야 하는데 아니다 · 보통 파일이어야 하는데 아니다 (symlink 포함)
	permForeignOwner            // 주인이 노드 uid 가 아니다 — 특권 없이 고칠 수 없고 남이 놓았을 수 있다
)

// permVerdict 는 이미 있는 자리 하나를 본다. mode 는 st_mode 그대로, owner 는 그 주인, uid 는 노드 uid 다.
// 종류 · 주인 · 비트 차례로 본다 — 종류가 틀리면 주인을 볼 까닭이 없다.
func permVerdict(mode uint32, owner, uid int, dir bool) permResult {
	want := uint32(modeRegular)
	if dir {
		want = modeDir
	}
	switch {
	case mode&modeType != want:
		return permWrongKind
	case owner != uid:
		return permForeignOwner
	case mode&modeLoose != 0:
		return permLoose
	}
	return permOK
}

var (
	errNotDir     = errors.New("not a directory")
	errNotRegular = errors.New("not a regular file")
)

// permError 는 permVerdict 가 거절한 자리의 문구다.
func permError(p string, r permResult, owner, uid int, dir bool) error {
	switch r {
	case permWrongKind:
		if dir {
			return &pathError{Path: p, Err: errNotDir}
		}
		return &pathError{Path: p, Err: errNotRegular}
	case permForeignOwner:
		return &pathError{Path: p, Err: fmt.Errorf("owned by uid %d, not the node user %d", owner, uid)}
	}
	return nil
}

// ── mountinfo ───────────────────────────────────────────────────────────

// mountLine 은 /proc/<pid>/mountinfo 의 한 줄이다 (proc(5)).
type mountLine struct {
	ID     uint64
	Parent uint64
	Dev    uint64 // major:minor 를 mkdev 로 붙인 값
	Root   string // 그 filesystem 안에서 이 마운트의 뿌리
	Point  string // 마운트 자리
	FSType string
	Source string
	Super  string // super options — 쉼표로 나뉜 채 그대로
}

// parseMountinfo 는 mountinfo 한 벌을 푼다. 경로의 \040 같은 8진 탈출을 푼다. 선택 칸은 몇 개든 건너뛰고
// 구분자 "-" 뒤의 셋을 읽는다. 모양이 어긋난 줄이 있으면 오류다.
func parseMountinfo(b []byte) ([]mountLine, error) {
	var out []mountLine
	for n, line := range strings.Split(string(b), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		f := strings.Fields(line)
		sep := -1
		for i := 6; i < len(f); i++ {
			if f[i] == "-" {
				sep = i
				break
			}
		}
		if len(f) < 7 || sep < 0 || len(f) < sep+3 {
			return nil, fmt.Errorf("mountinfo line %d is malformed: %q", n+1, line)
		}
		id, err1 := strconv.ParseUint(f[0], 10, 64)
		parent, err2 := strconv.ParseUint(f[1], 10, 64)
		maj, min, ok := strings.Cut(f[2], ":")
		major, err3 := strconv.ParseUint(maj, 10, 32)
		minor, err4 := strconv.ParseUint(min, 10, 32)
		if err := errors.Join(err1, err2, err3, err4); err != nil || !ok {
			return nil, fmt.Errorf("mountinfo line %d is malformed: %q", n+1, line)
		}
		l := mountLine{ID: id, Parent: parent, Dev: mkdev(uint32(major), uint32(minor)),
			Root: unescapeMount(f[3]), Point: unescapeMount(f[4]), FSType: f[sep+1], Source: unescapeMount(f[sep+2])}
		if len(f) > sep+3 {
			l.Super = f[sep+3]
		}
		out = append(out, l)
	}
	return out, nil
}

// unescapeMount 는 커널이 8진으로 탈출한 글자(\040 공백 · \011 탭 · \012 줄바꿈 · \134 역빗금 · \054 쉼표)를 푼다.
func unescapeMount(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+3 < len(s) && isOctal(s[i+1]) && isOctal(s[i+2]) && isOctal(s[i+3]) {
			b.WriteByte((s[i+1]-'0')<<6 | (s[i+2]-'0')<<3 | (s[i+3] - '0'))
			i += 3
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func isOctal(c byte) bool { return c >= '0' && c <= '7' }

// mountByID 는 마운트 번호로 줄을 찾는다.
func mountByID(lines []mountLine, id uint64) (mountLine, bool) {
	if id == 0 {
		return mountLine{}, false
	}
	for _, l := range lines {
		if l.ID == id {
			return l, true
		}
	}
	return mountLine{}, false
}

// longestMount 는 경로 p 를 담은 마운트 중 마운트 자리가 가장 긴 것이다. 같은 자리에 겹친 마운트는
// 나중 줄이 위에 있다 (답 6 — STATX_MNT_ID 가 없는 커널의 대신).
func longestMount(lines []mountLine, p string) (mountLine, bool) {
	best, found := mountLine{}, false
	for _, l := range lines {
		if !within(p, l.Point) {
			continue
		}
		if !found || len(l.Point) >= len(best.Point) {
			best, found = l, true
		}
	}
	return best, found
}

// within 은 p 가 dir 이거나 그 안인가다. 둘 다 깨끗한 절대 경로다.
func within(p, dir string) bool {
	return p == dir || dir == "/" || strings.HasPrefix(p, dir+"/")
}

// inFS 는 마운트 m 을 지나 보이는 경로 p 가 그 filesystem 안에서 어느 경로인가다.
func inFS(m mountLine, p string) string {
	rest := strings.TrimPrefix(p, m.Point)
	if m.Point == "/" {
		rest = p
	}
	return path.Join(m.Root, rest)
}

// related 는 두 경로가 같거나 한쪽이 다른 쪽을 담는가다 (L 이거나 L 안이거나 L 을 담는다).
func related(a, b string) bool { return within(a, b) || within(b, a) }

// lowerDirs 는 overlay 의 super options 에서 아래층 경로를 모은다 — lowerdir= 를 : 로 나눈 것 ·
// lowerdir+= · datadir+=. 탈출한 쉼표 · 콜론 · 역빗금을 푼다.
func lowerDirs(super string) []string {
	var out []string
	for _, opt := range strings.Split(super, ",") {
		key, val, ok := strings.Cut(opt, "=")
		if !ok {
			continue
		}
		val = unescapeMount(val)
		switch key {
		case "lowerdir":
			out = append(out, splitColons(val)...)
		case "lowerdir+", "datadir+":
			out = append(out, val)
		}
	}
	return out
}

// splitColons 는 lowerdir 의 목록을 나눈다. \: 는 경로 안의 콜론이다. 빈 조각(:: 의 data 층 구분)은 버린다.
func splitColons(s string) []string {
	var out []string
	var cur strings.Builder
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '\\' && i+1 < len(s):
			cur.WriteByte(s[i+1])
			i++
		case s[i] == ':':
			if cur.Len() > 0 {
				out = append(out, cur.String())
			}
			cur.Reset()
		default:
			cur.WriteByte(s[i])
		}
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

// overlaysOn 은 한 namespace 의 mountinfo 에서 이 lower 에 닿는 overlay 를 고른다 (business-rules.md 8.2).
// dev 와 inPath 는 이 lower 의 (장치, filesystem 안의 경로 L) 이다. overlay 줄만 본다 — bind 줄만으로 보면
// lower 를 그대로 bind 한 호스트의 마운트(/work 모양)와 모든 namespace 의 / 가 걸린다.
func overlaysOn(lines []mountLine, dev uint64, inPath string) []Mount {
	var out []Mount
	for _, l := range lines {
		if l.FSType != "overlay" {
			continue
		}
		for _, p := range lowerDirs(l.Super) {
			p = path.Clean(p)
			m, ok := longestMount(lines, p)
			if !ok || m.Dev != dev {
				continue
			}
			if related(inFS(m, p), inPath) {
				out = append(out, Mount{MountPoint: l.Point, LowerDir: p})
				break
			}
		}
	}
	return out
}

// ── 점검 셋 (business-rules.md 12절) ─────────────────────────────────────

// 점검 셋의 문구 (business-rules.md 12.3).
const (
	scratchRequired = "same filesystem and mount as the workspace"
	scratchRemedy   = "put environment.scratch on the same mount as the workspace, outside it; " +
		"a bind alias of the workspace is a different mount"
	ownerRemedy = "run this node as the owner of the workspace; one lower is shared by one user"
)

// workspaceUnread 는 워크스페이스를 못 읽었을 때의 두 어긋남이다 — 설정이 쓸 수 없는 자리를 가리킨다.
func workspaceUnread(uid int, err error) []Finding {
	obs := "cannot read the workspace: " + err.Error()
	return []Finding{
		{Name: findingScratch, Required: scratchRequired, Observed: obs, Cause: CauseBinding, Remediation: scratchRemedy},
		{Name: findingOwnerUID, Required: fmt.Sprintf("uid %d (the node user)", uid), Observed: obs,
			Cause: CauseBinding, Remediation: ownerRemedy},
	}
}

// scratchFinding 은 binding.scratch_filesystem 이다. scratch 는 scratch 의 가장 가까운 있는 조상을 읽은 값이다.
// 마운트 번호가 한쪽이라도 0 이면 장치만 댄다.
func scratchFinding(scratch Root, scratchErr error, ws Root) Finding {
	f := Finding{Name: findingScratch, Required: scratchRequired, Cause: CauseBinding, Remediation: scratchRemedy}
	switch {
	case scratchErr != nil:
		f.Observed = "cannot read the scratch: " + scratchErr.Error()
	case scratch.Dev != ws.Dev:
		f.Observed = fmt.Sprintf("different filesystem (scratch dev %s, workspace dev %s)",
			devString(scratch.Dev), devString(ws.Dev))
	case scratch.MountID != 0 && ws.MountID != 0 && scratch.MountID != ws.MountID:
		f.Observed = fmt.Sprintf("same filesystem, different mount (scratch mount %d, workspace mount %d); "+
			"is the workspace a bind alias?", scratch.MountID, ws.MountID)
	default:
		return Finding{Name: f.Name, Required: f.Required, Observed: "same mount", OK: true}
	}
	return f
}

// ownerFinding 은 lower.owner_uid 다 — 한 lower 는 한 사용자가 쓴다 (결정 3-14).
func ownerFinding(ws Root, uid int) Finding {
	f := Finding{Name: findingOwnerUID, Required: fmt.Sprintf("uid %d (the node user)", uid),
		Observed: fmt.Sprintf("uid %d", ws.UID), OK: ws.UID == uid}
	if !f.OK {
		f.Cause, f.Remediation = CauseBinding, ownerRemedy
	}
	return f
}

// looseNote 는 Peek 이 느슨한 비트를 본 자리에 붙이는 한 마디다 — 점검은 고치지 않는다 (ADR-073).
const looseNote = "; the node narrows loose permissions on start"

// identityFinding 은 lower.identity 다. err 가 있으면 자리나 파일을 못 읽은 것이다.
func identityFinding(dir string, v Verdict, st State, err error, loose bool) Finding {
	f := Finding{Name: findingIdentity, Required: "the recorded identity of this lower", OK: true}
	switch {
	case err != nil:
		f.OK = false
		var pe *pathError
		if errors.As(err, &pe) {
			f.Observed = "cannot read " + pe.Path + ": " + pe.Err.Error()
		} else {
			f.Observed = "cannot read " + dir + ": " + err.Error()
		}
	case v == VerdictNew:
		f.Observed = "not recorded yet; the node records it on start"
	case v == VerdictMatch:
		f.Observed = "matches"
	case v == VerdictReused:
		f.Observed = "recorded for another directory with the same inode; the node rewrites it on start"
	case v == VerdictForeign:
		f.OK = false
		f.Observed = "recorded for another directory and a bake is " + string(st.Phase)
		if st.Owner != nil && st.Owner.Run != "" {
			f.Observed += " (run " + st.Owner.Run + ")"
		}
	default:
		f.OK = false
		f.Observed = "cannot read " + path.Join(dir, identityFile) + ": " + errNotItsKey.Error()
	}
	if loose {
		f.Observed += looseNote
	}
	if !f.OK {
		f.Cause = CauseState
		f.Remediation = "inspect " + dir + "; remove it only when no bake of that directory is left"
	}
	return f
}

// ── state.json · metadata 거르기 ─────────────────────────────────────────

// checkState 는 읽은 state.json 을 거른다. 모르는 phase 를 committed 로 읽지 않는다 (business-rules.md 3절).
func checkState(s State) error {
	if s.Schema != schema {
		return fmt.Errorf("schema %d is not %d", s.Schema, schema)
	}
	switch s.Phase {
	case PhaseCommitted, PhaseBuilding, PhasePending, PhaseMerging:
		return nil
	}
	return fmt.Errorf("unknown phase %q", s.Phase)
}

// checkMetadata 는 읽은 metadata 를 거른다 — schema 1 만 (business-rules.md 11절).
func checkMetadata(m *Metadata) error {
	if m.Schema != schema {
		return fmt.Errorf("schema %d is not %d", m.Schema, schema)
	}
	return nil
}
