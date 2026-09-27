package lower

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func findingNamed(t *testing.T, fs []Finding, name string) Finding {
	t.Helper()
	for _, f := range fs {
		if f.Name == name {
			return f
		}
	}
	t.Fatalf("no %s in %+v", name, fs)
	return Finding{}
}

// scratch 가 워크스페이스와 같은 폴더 아래 · 아직 없는 scratch · uid 가 다름 · lowers 가 "" 면 신원을 안 낸다.
func TestCheck(t *testing.T) {
	f := newFixture(t)
	scratch := filepath.Join(f.dir, "scratch")
	if err := os.Mkdir(scratch, 0o700); err != nil {
		t.Fatal(err)
	}
	fs := Check(f.lowers, f.lower, scratch, os.Getuid())
	if len(fs) != 3 {
		t.Fatalf("Check = %+v", fs)
	}
	for _, x := range fs {
		if !x.OK {
			t.Errorf("not ready: %+v", x)
		}
	}
	if got := findingNamed(t, fs, "binding.scratch_filesystem").Observed; got != "same mount" {
		t.Errorf("scratch = %s", got)
	}
	if got := findingNamed(t, fs, "lower.identity").Observed; got != "not recorded yet; the node records it on start" {
		t.Errorf("identity before Open = %s", got)
	}
	if _, err := os.Lstat(f.lowers); !os.IsNotExist(err) {
		t.Fatalf("Check made the state directory: %v", err)
	}

	// 아직 없는 scratch 는 가장 가까운 있는 조상으로 본다
	fs = Check("", f.lower, filepath.Join(f.dir, "a", "b", "scratch"), os.Getuid())
	if len(fs) != 2 || !fs[0].OK || !fs[1].OK {
		t.Errorf("a scratch not made yet = %+v", fs)
	}
	if got := existingAncestor("/"); got != "/" {
		t.Errorf("existingAncestor(/) = %s", got)
	}

	// 다른 uid 가 돌리면 — 한 lower 는 한 사용자가 쓴다
	o := findingNamed(t, Check("", f.lower, scratch, os.Getuid()+1), "lower.owner_uid")
	if o.OK || o.Cause != CauseBinding {
		t.Errorf("owner = %+v", o)
	}

	// 신원 판정의 연결 — Open 뒤 matches
	f.open(t)
	if got := findingNamed(t, Check(f.lowers, f.lower, scratch, os.Getuid()), "lower.identity"); !got.OK || got.Observed != "matches" {
		t.Errorf("identity after Open = %+v", got)
	}
	// 깨진 lower.json 은 external-blocked 로 옮겨질 어긋남이다
	d, _ := Peek(f.lowers, f.root)
	writeFile(t, filepath.Join(d.Path, identityFile), "{")
	got := findingNamed(t, Check(f.lowers, f.lower, scratch, os.Getuid()), "lower.identity")
	if got.OK || got.Cause != CauseState || !strings.HasPrefix(got.Observed, "cannot read "+filepath.Join(d.Path, identityFile)+": ") ||
		got.Remediation != "inspect "+d.Path+"; remove it only when no bake of that directory is left" {
		t.Errorf("identity of a broken record = %+v", got)
	}

	// 워크스페이스를 못 읽으면 앞의 둘이 어긋난다
	fs = Check(f.lowers, filepath.Join(f.dir, "gone"), scratch, os.Getuid())
	if len(fs) != 2 || fs[0].OK || fs[1].OK || !strings.HasPrefix(fs[0].Observed, "cannot read the workspace: ") {
		t.Errorf("a missing workspace = %+v", fs)
	}
	// scratch 자리를 못 읽으면 (파일 아래) 어긋난다
	file := filepath.Join(f.dir, "file")
	writeFile(t, file, "")
	if got := findingNamed(t, Check("", f.lower, filepath.Join(file, "scratch"), os.Getuid()), "binding.scratch_filesystem"); got.OK {
		t.Errorf("a scratch under a file = %+v", got)
	}
}

// 다른 filesystem — /dev/shm 이 tmpfs 로 있으면 진짜로 본다. 없으면 판정 표(TestFindings)가 본다.
func TestCheckOtherFilesystem(t *testing.T) {
	f := newFixture(t)
	shm, err := os.MkdirTemp("/dev/shm", "enode-lower-test-")
	if err != nil {
		t.Logf("no /dev/shm here (%v); TestFindings covers a different filesystem", err)
		return
	}
	defer func() { _ = os.RemoveAll(shm) }()
	sc, err := ReadRoot(shm)
	if err != nil {
		t.Fatal(err)
	}
	got := findingNamed(t, Check("", f.lower, shm, os.Getuid()), "binding.scratch_filesystem")
	if sc.Dev == f.root.Dev {
		t.Logf("/dev/shm shares the device of the temporary directory; observed %q", got.Observed)
		return
	}
	if got.OK || got.Cause != CauseBinding || !strings.HasPrefix(got.Observed, "different filesystem (scratch dev ") {
		t.Errorf("scratch on /dev/shm = %+v", got)
	}
}
