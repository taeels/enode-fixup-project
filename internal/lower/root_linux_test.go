package lower

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// fixture 는 시험 하나의 lower 루트와 상태 자리의 뿌리다. 둘 다 임시 폴더 안이다.
type fixture struct {
	dir    string // 임시 폴더
	lower  string // lower 루트 (워크스페이스)
	lowers string // 상태 자리의 뿌리 — 아직 없다
	root   Root
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	dir := t.TempDir()
	f := fixture{dir: dir, lower: filepath.Join(dir, "ws"), lowers: filepath.Join(dir, "state", "enode", "lowers")}
	if err := os.Mkdir(f.lower, 0o755); err != nil {
		t.Fatal(err)
	}
	root, err := ReadRoot(f.lower)
	if err != nil {
		t.Fatal(err)
	}
	f.root = root
	return f
}

func (f fixture) open(t *testing.T) *Dir {
	t.Helper()
	d, err := Open(f.lowers, f.root, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// 임시 폴더의 뿌리를 읽는다 — 키는 stat 과 같은 글자이고 btime 과 마운트 번호가 있다.
func TestReadRoot(t *testing.T) {
	f := newFixture(t)
	var st unix.Stat_t
	if err := unix.Stat(f.lower, &st); err != nil {
		t.Fatal(err)
	}
	var sfs unix.Statfs_t
	if err := unix.Statfs(f.lower, &sfs); err != nil {
		t.Fatal(err)
	}
	r := f.root
	real, _ := filepath.EvalSymlinks(f.lower)
	if r.Path != real || r.Key.Ino != st.Ino || r.Dev != st.Dev || r.UID != os.Getuid() {
		t.Fatalf("root = %+v, stat ino %d dev %d", r, st.Ino, st.Dev)
	}
	if r.MountID == 0 {
		t.Errorf("no mount id for %s", r.Path)
	}
	// btime 은 ext4 와 tmpfs 가 준다 (계획 3절 — 이 기계의 /tmp · /dev/shm). 다른 filesystem 이면 0 일 수 있다.
	const ext4, tmpfs = 0xef53, 0x01021994
	if (sfs.Type == ext4 || sfs.Type == tmpfs) && r.BirthNs == 0 {
		t.Errorf("no birth time on filesystem type %#x", sfs.Type)
	}
	if sfs.Type != ext4 && sfs.Type != tmpfs {
		t.Logf("filesystem type %#x, birth time %d", sfs.Type, r.BirthNs)
	}
	// 키의 fsid 는 `stat -f -c %i` 와 같은 글자다 — 사람이 그 명령으로 자리를 찾는다.
	if out, err := exec.Command("stat", "-f", "-c", "%i", f.lower).Output(); err == nil {
		if got := strings.TrimSpace(string(out)); got != r.Key.fsidString() {
			t.Errorf("stat -f -c %%i = %s, key fsid %s", got, r.Key.fsidString())
		}
	} else {
		t.Logf("stat -f is unavailable here: %v", err)
	}
	// STATX_MNT_ID 가 없는 커널의 대신 — mountinfo 에서 가장 길게 겹치는 마운트가 같은 번호다.
	if got := mountIDOf(selfMountinfo, r.Path); got != r.MountID {
		t.Errorf("mountinfo says mount %d, statx %d", got, r.MountID)
	}
	if got := mountIDOf(filepath.Join(f.dir, "none"), r.Path); got != 0 {
		t.Errorf("a missing mountinfo gave %d", got)
	}
	bad := filepath.Join(f.dir, "bad")
	if err := os.WriteFile(bad, []byte("not mountinfo\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := mountIDOf(bad, r.Path); got != 0 {
		t.Errorf("a malformed mountinfo gave %d", got)
	}
	empty := filepath.Join(f.dir, "empty")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if got := mountIDOf(empty, r.Path); got != 0 {
		t.Errorf("an empty mountinfo gave %d", got)
	}
}

// symlink 로 준 뿌리는 풀린다 — 같은 키다. 디렉터리가 아니거나 없으면 오류다.
func TestReadRootResolvesAndRejects(t *testing.T) {
	f := newFixture(t)
	link := filepath.Join(f.dir, "link")
	if err := os.Symlink(f.lower, link); err != nil {
		t.Fatal(err)
	}
	r, err := ReadRoot(link)
	if err != nil || r.Key != f.root.Key || r.Path != f.root.Path {
		t.Fatalf("ReadRoot(symlink) = %+v, %v; want %+v", r, err, f.root)
	}
	file := filepath.Join(f.dir, "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadRoot(file); err == nil || err.Error() != "lower: "+file+": not a directory" {
		t.Errorf("ReadRoot(file) = %v", err)
	}
	if _, err := ReadRoot(filepath.Join(f.dir, "nope")); err == nil || !errors.Is(err, os.ErrNotExist) {
		t.Errorf("ReadRoot(missing) = %v", err)
	}
}

// ext4 는 지운 디렉터리의 inode 번호를 곧바로 다시 쓴다 (FD 계획 2.3). 같은 번호가 나오면 Reused 와 Foreign 을
// 진짜 자리로 본다. 안 나오면 판정 표(judge_test.go)만으로 본다 — 스킵하지 않는다.
func TestInodeReuse(t *testing.T) {
	f := newFixture(t)
	d := f.open(t)
	b, ok, err := d.TryBake()
	if err != nil || !ok {
		t.Fatalf("TryBake = %v, %v", ok, err)
	}
	failed := &LastAttempt{Run: "R-old", At: time.Now().UTC(), Reason: "build failed"}
	if err := b.WriteState(State{Phase: PhaseCommitted, Since: time.Now(), LastAttempt: failed}); err != nil {
		t.Fatal(err)
	}
	_ = b.Release()

	recreate := func() (Root, bool) {
		t.Helper()
		if err := os.Remove(f.lower); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(f.lower, 0o755); err != nil {
			t.Fatal(err)
		}
		r, err := ReadRoot(f.lower)
		if err != nil {
			t.Fatal(err)
		}
		return r, r.Key == f.root.Key && r.BirthNs != 0 && r.BirthNs != f.root.BirthNs
	}
	r2, reused := recreate()
	if !reused {
		t.Logf("the inode was not reused here (%v -> %v); the verdict table covers Reused and Foreign", f.root.Key, r2.Key)
		return
	}
	d2, err := Open(f.lowers, r2, time.Now())
	if err != nil {
		t.Fatalf("Open after the inode was reused: %v", err)
	}
	if len(d2.Notes) == 0 || !strings.Contains(strings.Join(d2.Notes, "\n"), "recorded another directory with the same inode; rewrote it") {
		t.Errorf("notes = %q", d2.Notes)
	}
	rec, err := d2.readIdentity()
	if err != nil || rec.BirthNs != r2.BirthNs {
		t.Errorf("lower.json = %+v, %v; want birth %d", rec, err, r2.BirthNs)
	}
	if st, err := d2.ReadState(); err != nil || st.LastAttempt != nil {
		t.Errorf("the old directory's last_attempt is left: %+v, %v", st, err)
	}

	// 끊긴 굽기가 남은 채 다른 디렉터리를 만나면 Foreign — 자리를 열지 않는다.
	b, ok, err = d2.TryBake()
	if err != nil || !ok {
		t.Fatal(err)
	}
	if err := b.WriteState(State{Phase: PhasePending, Owner: &Owner{Run: "R-9"}, Since: time.Now()}); err != nil {
		t.Fatal(err)
	}
	_ = b.Release()
	f.root = r2
	r3, reused := recreate()
	if !reused {
		t.Logf("the inode was not reused the second time (%v -> %v)", r2.Key, r3.Key)
		return
	}
	want := "records a different directory (birth time differs) and its state is pending"
	if _, err := Open(f.lowers, r3, time.Now()); err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Open over a pending bake of another directory = %v", err)
	}
	if got := Check(f.lowers, f.lower, f.dir, os.Getuid()); len(got) != 3 || got[2].OK ||
		got[2].Observed != "recorded for another directory and a bake is pending (run R-9)" {
		t.Errorf("Check = %+v", got)
	}
}
