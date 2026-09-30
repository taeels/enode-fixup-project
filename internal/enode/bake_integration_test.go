//go:build integration && linux

package enode

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/taeels/enode/internal/lower"
)

// 굽기의 integration 시험 (bake 유닛 · FD 흐름 8.2 · CI 밖). 진짜 unshare 로 여는 merge-helper — 제품 기본값
// (mergeHelperCommand) 그대로 이 시험 바이너리를 다시 실행한다 (runc_overlay_integration_test.go 의 TestMain 이
// merge-helper 입구를 안다). --map-auto 가 되는 기계 (SunnyVM) 에서 돈다. 버려도 되는 자리에서만 — TMPDIR 을 ext4 의
// 버릴 폴더로 둔다.
//
//	go test -c -tags integration -o enode.it ./internal/enode
//	TMPDIR=$HOME/bake-it-tmp ./enode.it -test.run 'TestMergeHelperIntegration|TestResumeAfterAKilledHelperIntegration' -test.v

const bakeItChild = "ENODE_BAKE_IT_CHILD"

// itInNamespace 는 이 바이너리를 helper 와 같은 매핑의 namespace 에서 그 시험 이름으로 다시 실행한다 — upper 에
// subordinate uid 소유 항목을 짓는 자리다. 매핑을 못 하면 실패한다 — 건너뛰지 않는다.
func itInNamespace(t *testing.T, test, spec string) {
	t.Helper()
	cmd := exec.Command("unshare", "--user", "--map-root-user", "--map-auto", os.Args[0], "-test.run=^"+test+"$",
		"-test.count=1", "-test.v")
	cmd.Env = append(os.Environ(), bakeItChild+"="+spec)
	out, err := cmd.CombinedOutput()
	t.Logf("namespace child:\n%s", out)
	if err != nil {
		t.Fatalf("the namespace child failed: %v", err)
	}
}

// 진짜 unshare 의 merge-helper 가 preflight · apply 를 한다 — upper 에 subordinate uid 소유 파일과 권한 000 디렉터리가
// 있어도 합친다. lower 쪽 항목은 항목마다 trash 로 간다.
func TestMergeHelperIntegration(t *testing.T) {
	if upper := os.Getenv(bakeItChild); upper != "" {
		// namespace 안 — uid 1000 은 subordinate 범위다
		if err := os.WriteFile(filepath.Join(upper, "owned"), []byte("sub"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chown(filepath.Join(upper, "owned"), 1000, 1000); err != nil {
			t.Fatal(err)
		}
		locked := filepath.Join(upper, "locked")
		if err := os.MkdirAll(filepath.Join(locked, "inner"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(locked, 0); err != nil {
			t.Fatal(err)
		}
		// 종류가 바뀐 항목 — lower 의 디렉터리가 trash 로 간다
		if err := os.WriteFile(filepath.Join(upper, "changed"), []byte("now a file"), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	root := t.TempDir()
	req := mergeHelperRequest{Upper: filepath.Join(root, "upper"), Lower: filepath.Join(root, "lower"),
		Trash: filepath.Join(root, "trash")}
	for _, d := range []string{req.Upper, req.Lower, req.Trash} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(req.Lower, "locked"), 0o755) })
	if err := os.WriteFile(filepath.Join(req.Lower, "owned"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(req.Lower, "locked"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(req.Lower, "locked", "old"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(req.Lower, "changed", "was"), 0o755); err != nil {
		t.Fatal(err)
	}
	itInNamespace(t, "TestMergeHelperIntegration", req.Upper)
	argv, err := mergeHelperCommand()
	if err != nil || argv[0] != "unshare" {
		t.Fatalf("the product default = %q %v", argv, err)
	}
	req.Op = "preflight"
	if _, err := callMergeHelper(req); err != nil {
		t.Fatalf("preflight = %v", err)
	}
	req.Op = "apply"
	res, err := callMergeHelper(req)
	if err != nil {
		t.Fatalf("apply = %v", err)
	}
	t.Logf("apply = %+v", res)
	var st syscall.Stat_t
	if err := syscall.Lstat(filepath.Join(req.Lower, "owned"), &st); err != nil || int(st.Uid) == os.Getuid() {
		t.Fatalf("the subordinate-owned file: uid %d err %v", st.Uid, err)
	}
	if err := syscall.Lstat(filepath.Join(req.Lower, "locked"), &st); err != nil || st.Mode&0o777 != 0 {
		t.Fatalf("the mode-000 directory: mode %o err %v", st.Mode&0o777, err)
	}
	if ents, _ := os.ReadDir(req.Trash); len(ents) != 1 || ents[0].Name() != "changed" {
		t.Fatalf("trash = %v; want the lower's changed directory", ents)
	}
	if b, _ := os.ReadFile(filepath.Join(req.Lower, "changed")); string(b) != "now a file" {
		t.Fatalf("lower changed = %q", b)
	}
}

// 끊고 잇기 — apply 도중 helper 를 SIGKILL 하면 merging 이 남고, 재개가 끝낸다 (metadata resumed true · committed).
// helper 를 여는 명령을 감싸 upper 의 절반이 옮겨진 순간에 unshare 를 죽인다 — --kill-child 가 helper 도 거둔다.
func TestResumeAfterAKilledHelperIntegration(t *testing.T) {
	const n = 100000
	f := newBakeFixture(t)
	pending := f.pendingIn(t, f.scratch, "R-dead", true)
	upperDir := filepath.Join(pending, "upper", "d")
	if err := os.MkdirAll(filepath.Join(f.ws, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(upperDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < n; i++ {
		name := "f" + strconv.Itoa(i)
		if err := os.WriteFile(filepath.Join(f.ws, "d", name), []byte("old"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(upperDir, name), []byte("new"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	f.putState(t, lower.State{Phase: lower.PhaseMerging, Owner: &lower.Owner{Run: "R-dead", Node: "node-k"},
		PendingUpper: filepath.Join(pending, "upper")})
	trash := filepath.Join(f.scratch, "trash")
	if err := os.MkdirAll(trash, 0o700); err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	was := mergeHelperCommand
	// 비동기 명령의 stdin 은 /dev/null 이 된다 — 요청 한 줄을 fd 3 으로 넘긴다
	killer := fmt.Sprintf(`exec 3<&0
%s 0<&3 &
p=$!
exec 3<&-
while [ "$(ls %s | wc -l)" -gt %d ]; do sleep 0.01; done
kill -9 $p
wait $p`, strings.Join(mergeHelperArgv(exe), " "), upperDir, n/2)
	mergeHelperCommand = func() ([]string, error) { return []string{"sh", "-c", killer}, nil }
	req := mergeHelperRequest{Op: "apply", Upper: filepath.Join(pending, "upper"), Lower: f.ws, Trash: trash}
	_, err = callMergeHelper(req)
	mergeHelperCommand = was
	if err == nil || !strings.HasPrefix(err.Error(), "cannot run the merge helper: ") {
		t.Fatalf("the killed apply = %v", err)
	}
	left, _ := os.ReadDir(upperDir)
	t.Logf("killed with %d of %d entries left in the upper", len(left), n)
	if len(left) == 0 || len(left) == n {
		t.Fatalf("the kill did not land mid-apply: %d left", len(left))
	}
	f.staleOnce(t)
	if st := f.state(t); st.Phase != lower.PhaseCommitted {
		t.Fatalf("state = %+v\n%s", st, f.nodeLog)
	}
	md, err := lower.ReadMetadata(f.ws)
	if err != nil || md == nil || !md.Bake.Resumed || md.Bake.Run != "R-dead" {
		t.Fatalf("metadata = %+v %v", md, err)
	}
	for _, i := range []int{0, n / 2, n - 1} {
		if b, _ := os.ReadFile(filepath.Join(f.ws, "d", "f"+strconv.Itoa(i))); string(b) != "new" {
			t.Fatalf("lower d/f%d = %q", i, b)
		}
	}
	t.Logf("node log:\n%s", f.nodeLog)
}
