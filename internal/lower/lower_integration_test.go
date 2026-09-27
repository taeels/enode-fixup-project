//go:build integration && linux

package lower

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// 사람이 도는 시험이다 (integration 태그 · CI 밖 · business-logic-model.md 9.2). 기본 go test 가 못 보는 둘을 본다 —
// 다른 프로세스의 마운트 namespace 에 걸린 진짜 overlay 와 진짜 bind 별칭. 버려도 되는 자리(t.TempDir)에서만 돈다.
// 이 결과로 조각 8 (사람 · bake 유닛) 을 대신하지 않는다.
//
//	go test -c -tags integration -o lower.it ./internal/lower
//	./lower.it -test.run Integration -test.v

// itChild 는 namespace 안에서 다시 실행된 자식의 할 일이다 — "mount|<ws>|<base>" · "alias|<ws>|<base>".
const itChild = "ENODE_LOWER_IT_CHILD"

// itRun 은 자기 바이너리를 helper 와 같은 모양의 namespace (unshare --user --map-root-user --mount) 에서 다시 띄운다.
func itRun(t *testing.T, test, spec string) *exec.Cmd {
	t.Helper()
	unshare, err := exec.LookPath("unshare")
	if err != nil {
		t.Fatalf("unshare is required: %v", err)
	}
	cmd := exec.Command(unshare, "--user", "--map-root-user", "--mount",
		os.Args[0], "-test.run=^"+test+"$", "-test.count=1", "-test.v")
	cmd.Env = append(os.Environ(), itChild+"="+spec)
	return cmd
}

// itSpec 는 자식이 받은 할 일을 푼다. 부모면 ok 가 거짓이다.
func itSpec(t *testing.T, kind string) (ws, base string, ok bool) {
	spec := os.Getenv(itChild)
	if spec == "" {
		return "", "", false
	}
	parts := strings.Split(spec, "|")
	if len(parts) != 3 || parts[0] != kind {
		t.Fatalf("bad child spec %q", spec)
	}
	if err := unix.Mount("", "/", "", unix.MS_REC|unix.MS_PRIVATE, ""); err != nil {
		t.Fatalf("make the mount namespace private: %v", err)
	}
	return parts[1], parts[2], true
}

// 다른 프로세스가 helper 모양 (bind + overlay) 으로 lower 를 마운트하면 ForeignMounts 가 찾고, 끝나면 0 이다.
// 같은 namespace 의 bind 별칭만으로는 찾지 않는다.
func TestForeignMountsIntegration(t *testing.T) {
	if ws, base, child := itSpec(t, "mount"); child {
		lowerRO, alias := filepath.Join(base, "lower-ro"), filepath.Join(base, "alias")
		for _, m := range []struct{ src, dst string }{{ws, lowerRO}, {ws, alias}} {
			if err := unix.Mount(m.src, m.dst, "", unix.MS_BIND|unix.MS_REC, ""); err != nil {
				t.Fatalf("bind %s: %v", m.dst, err)
			}
		}
		opts := "lowerdir=" + lowerRO + ",upperdir=" + filepath.Join(base, "upper") + ",workdir=" + filepath.Join(base, "work")
		if err := unix.Mount("overlay", filepath.Join(base, "merged"), "overlay", 0, opts); err != nil {
			t.Fatalf("mount overlay: %v", err)
		}
		os.Stdout.WriteString("mounted\n")
		_, _ = io.Copy(io.Discard, os.Stdin) // 부모가 stdin 을 닫을 때까지 쥔다
		return
	}

	base := t.TempDir()
	ws := filepath.Join(base, "ws")
	for _, d := range []string{ws, filepath.Join(base, "lower-ro"), filepath.Join(base, "alias"), filepath.Join(base, "upper"),
		filepath.Join(base, "work"), filepath.Join(base, "merged")} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	root, err := ReadRoot(ws)
	if err != nil {
		t.Fatal(err)
	}
	if scan, err := ForeignMounts(root); err != nil || len(scan.Found) != 0 {
		t.Fatalf("before the child = %+v, %v", scan, err)
	}

	cmd := itRun(t, "TestForeignMountsIntegration", "mount|"+ws+"|"+base)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	lines := bufio.NewScanner(stdout)
	for lines.Scan() && lines.Text() != "mounted" {
		t.Logf("child: %s", lines.Text())
	}
	began := time.Now()
	scan, err := ForeignMounts(root)
	took := time.Since(began)
	stdin.Close()
	rest, _ := io.ReadAll(stdout)
	waitErr := cmd.Wait()
	t.Logf("child output after mounting:\n%s", rest)
	if waitErr != nil {
		t.Fatalf("the namespace child failed: %v", waitErr)
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("scan %+v in %v", scan, took)
	if len(scan.Found) != 1 || scan.Found[0].PID != cmd.Process.Pid ||
		scan.Found[0].MountPoint != filepath.Join(base, "merged") || scan.Found[0].LowerDir != filepath.Join(base, "lower-ro") ||
		scan.Namespaces < 2 {
		t.Fatalf("found = %+v", scan)
	}
	if scan, err := ForeignMounts(root); err != nil || len(scan.Found) != 0 {
		t.Fatalf("after the child = %+v, %v", scan, err)
	}
}

// namespace 안에서 워크스페이스를 bind 별칭으로 두면 binding.scratch_filesystem 이 같은 filesystem · 다른 마운트로
// 어긋난다 (답 6 — rename 이 EXDEV 로 거절되는 자리). 원래 경로는 같은 마운트다.
func TestBindAliasCheckIntegration(t *testing.T) {
	if ws, base, child := itSpec(t, "alias"); child {
		alias, scratch := filepath.Join(base, "alias"), filepath.Join(base, "scratch")
		if err := unix.Mount(ws, alias, "", unix.MS_BIND|unix.MS_REC, ""); err != nil {
			t.Fatalf("bind the alias: %v", err)
		}
		fs := Check("", alias, scratch, os.Getuid())
		t.Logf("alias: %+v", fs)
		if len(fs) != 2 || fs[0].OK || !strings.HasPrefix(fs[0].Observed, "same filesystem, different mount (") ||
			!strings.HasSuffix(fs[0].Observed, "); is the workspace a bind alias?") || !fs[1].OK {
			t.Errorf("the bind alias was not caught: %+v", fs)
		}
		fs = Check("", ws, scratch, os.Getuid())
		t.Logf("original: %+v", fs)
		if len(fs) != 2 || !fs[0].OK || fs[0].Observed != "same mount" {
			t.Errorf("the original path is not the same mount: %+v", fs)
		}
		// rename 이 정말 거절된다 — 이 판정이 막으려는 것
		probe := filepath.Join(scratch, "probe")
		if err := os.WriteFile(probe, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(probe, filepath.Join(alias, "probe")); err == nil {
			t.Error("a rename across the bind alias succeeded")
		} else {
			t.Logf("rename across the alias: %v", err)
		}
		return
	}
	base := t.TempDir()
	for _, d := range []string{"ws", "alias", "scratch"} {
		if err := os.Mkdir(filepath.Join(base, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	out, err := itRun(t, "TestBindAliasCheckIntegration", "alias|"+filepath.Join(base, "ws")+"|"+base).CombinedOutput()
	t.Logf("namespace child:\n%s", out)
	if err != nil {
		t.Fatalf("the namespace child failed: %v", err)
	}
}
