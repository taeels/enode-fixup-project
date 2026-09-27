package lower

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 이 프로세스로 훑는다 — 이 lower 를 마운트한 곳이 없고, 이 프로세스의 namespace 는 읽힌다.
func TestForeignMountsThisProcess(t *testing.T) {
	f := newFixture(t)
	scan, err := ForeignMounts(f.root)
	if err != nil {
		t.Fatal(err)
	}
	if len(scan.Found) != 0 || scan.Namespaces < 1 {
		t.Fatalf("scan = %+v", scan)
	}
	t.Logf("namespaces %d, unreadable %d", scan.Namespaces, scan.Unreadable)
}

// fakeProc 은 가짜 /proc 이다. pid 마다 ns/mnt 링크 · mountinfo · status 를 둔다.
type fakeProc struct {
	t    *testing.T
	root string
}

func newFakeProc(t *testing.T, self string) fakeProc {
	p := fakeProc{t: t, root: t.TempDir()}
	p.file("self/mountinfo", self)
	return p
}

func (p fakeProc) file(rel, body string) {
	p.t.Helper()
	path := filepath.Join(p.root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		p.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		p.t.Fatal(err)
	}
}

// pid 는 프로세스 하나를 둔다. ns 가 ""면 좀비다 (ns 링크가 없다).
func (p fakeProc) pid(pid, ns, mountinfo string) {
	p.t.Helper()
	dir := filepath.Join(p.root, pid)
	if err := os.MkdirAll(filepath.Join(dir, "ns"), 0o700); err != nil {
		p.t.Fatal(err)
	}
	if ns != "" {
		if err := os.Symlink(ns, filepath.Join(dir, "ns", "mnt")); err != nil {
			p.t.Fatal(err)
		}
	}
	if mountinfo != "" {
		p.file(filepath.Join(pid, "mountinfo"), mountinfo)
	}
}

// 가짜 /proc 을 훑는다 — helper 모양은 찾고 · 같은 namespace 는 한 번만 읽고 · 좀비와 사라진 프로세스는
// 건너뛰고 · 권한으로 못 읽은 것은 센다.
func TestScanMounts(t *testing.T) {
	const host = "22 1 252:15 / / rw - ext4 /dev/vda2 rw\n"
	helper := host +
		"600 22 252:15 /home/u/ws /s/run/lower-ro ro - ext4 /dev/vda2 rw\n" +
		"601 22 0:77 / /s/run/merged rw - overlay overlay rw,lowerdir=/s/run/lower-ro,upperdir=/s/u,workdir=/s/w\n"
	root := Root{Path: "/home/u/ws", MountID: 22}
	p := newFakeProc(t, host)
	p.pid("100", "mnt:[4026532001]", helper)
	p.pid("101", "mnt:[4026532001]", helper) // 같은 namespace — 한 번만
	p.pid("102", "", "")                     // 좀비
	p.pid("103", "mnt:[4026531841]", host+"36 22 252:15 /home/u/ws /work rw - ext4 /dev/vda2 rw\n")
	p.pid("104", "mnt:[4026532002]", "")     // 그 사이 끝났다 — mountinfo 가 없다
	p.pid("105", "mnt:[4026532003]", "junk") // 모양이 어긋난 mountinfo
	p.pid("106", "mnt:[4026532004]", helper)
	p.file("abc/mountinfo", helper) // 숫자가 아닌 이름
	if os.Getuid() != 0 {
		if err := os.Chmod(filepath.Join(p.root, "106", "mountinfo"), 0); err != nil {
			t.Fatal(err)
		}
	}

	scan, err := scanMounts(p.root, root, os.Getuid())
	if err != nil {
		t.Fatal(err)
	}
	if len(scan.Found) != 1 || scan.Found[0].PID != 100 || scan.Found[0].Namespace != "mnt:[4026532001]" ||
		scan.Found[0].MountPoint != "/s/run/merged" || scan.Found[0].LowerDir != "/s/run/lower-ro" {
		t.Errorf("found = %+v", scan.Found)
	}
	wantNS, wantUnreadable := 3, 1
	if os.Getuid() == 0 {
		wantNS, wantUnreadable = 4, 0
	}
	if scan.Namespaces != wantNS || scan.Unreadable != wantUnreadable {
		t.Errorf("namespaces %d unreadable %d, want %d %d", scan.Namespaces, scan.Unreadable, wantNS, wantUnreadable)
	}
	// 다른 uid 로 훑으면 이 프로세스들은 남의 것이다
	if scan, err := scanMounts(p.root, root, os.Getuid()+1); err != nil || scan.Namespaces != 0 || len(scan.Found) != 0 {
		t.Errorf("another uid = %+v, %v", scan, err)
	}
	// 마운트 번호가 없으면 경로로 찾는다
	root.MountID = 0
	if scan, err := scanMounts(p.root, root, os.Getuid()); err != nil || len(scan.Found) != 1 {
		t.Errorf("without a mount id = %+v, %v", scan, err)
	}
}

func TestScanMountsErrors(t *testing.T) {
	root := Root{Path: "/home/u/ws", MountID: 22}
	if _, err := scanMounts(t.TempDir(), root, 0); err == nil || !strings.Contains(err.Error(), "self/mountinfo") {
		t.Errorf("no self mountinfo = %v", err)
	}
	p := newFakeProc(t, "junk")
	if _, err := scanMounts(p.root, root, 0); err == nil || !strings.Contains(err.Error(), "malformed") {
		t.Errorf("a malformed self mountinfo = %v", err)
	}
	p = newFakeProc(t, "30 1 8:1 / /data rw - ext4 /dev/sdb rw\n")
	if _, err := scanMounts(p.root, root, 0); err == nil || !strings.Contains(err.Error(), "no mount holds it") {
		t.Errorf("no mount holds the lower = %v", err)
	}
}

// /proc/<pid> 의 주인 — dumpable 이 꺼지면 root 소유로 보이고, 그때만 status 로 실제 uid 를 본다.
func TestOwnerOf(t *testing.T) {
	status := func(s string) func() []byte { return func() []byte { return []byte(s) } }
	never := func() []byte { t.Fatal("status was read"); return nil }
	for _, c := range []struct {
		name        string
		dirUID, uid int
		status      func() []byte
		want        ownerKind
	}{
		{"ours", 1000, 1000, never, ownerSelf},
		{"another user", 1001, 1000, never, ownerOther},
		{"root and we are root", 0, 0, never, ownerSelf},
		{"hidden ours", 0, 1000, status("Name:\tx\nUid:\t1000\t1000\t1000\t1000\n"), ownerHidden},
		{"hidden root", 0, 1000, status("Uid:\t0\t0\t0\t0\n"), ownerOther},
		{"no status", 0, 1000, status(""), ownerOther},
		{"no uid in status", 0, 1000, status("Uid:\n"), ownerOther},
	} {
		if got := ownerOf(c.dirUID, c.uid, c.status); got != c.want {
			t.Errorf("%s: %d, want %d", c.name, got, c.want)
		}
	}
	if got := owned(filepath.Join(t.TempDir(), "gone"), os.Getuid()); got != ownerOther {
		t.Errorf("a vanished process = %d", got)
	}
}
