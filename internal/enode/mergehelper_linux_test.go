//go:build linux

package enode

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/taeels/enode/internal/merge"
	"golang.org/x/sys/unix"
)

// merge-helper (bake 유닛 · FD 엔티티 8절 · FD 규칙 9절 · 계획 4.1 16번). 기본 go test 는 namespace 없이 시험 바이너리를
// 다시 띄워 진짜 merge.Preflight · Apply 를 임시 폴더에 돈다. 진짜 unshare 는 integration 시험이 본다.

// TestMergeHelperProcess 는 시험이 namespace 없이 띄우는 merge-helper 다. 모드가 없으면 진짜 RunMergeHelper 를
// 돈다. 모든 갈래를 os.Exit 로 끝낸다 — return 으로 끝나면 시험 바이너리가 stdout 에 PASS 와 coverage 줄을 찍어
// 응답 한 줄을 흐린다 (계획 4.1 31번).
func TestMergeHelperProcess(t *testing.T) {
	if os.Getenv("ENODE_TEST_MERGE_HELPER") != "1" {
		return
	}
	switch os.Getenv("ENODE_TEST_MERGE_MODE") {
	case "die":
		fmt.Fprint(os.Stderr, "newuidmap: write to uid_map failed: Operation not permitted")
		os.Exit(1)
	case "garbage":
		fmt.Println("this is not json")
		os.Exit(0)
	case "noisy":
		fmt.Fprint(os.Stderr, strings.Repeat("noise on stderr\n", 2000))
	case "apply-fails":
		// preflight 는 진짜로 돌고 apply 는 호출 하나가 실패한 것처럼 답한다 — root 로 돌아도 같은 갈래다
		var req mergeHelperRequest
		if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
			os.Exit(1)
		}
		if req.Op == "apply" {
			_ = json.NewEncoder(os.Stdout).Encode(mergeHelperResponse{Error: "merge: rename a.txt: permission denied",
				Kind: "op", Result: &merge.Result{}})
			os.Exit(1)
		}
		line, _ := json.Marshal(req)
		os.Exit(RunMergeHelper(bytes.NewReader(line), os.Stdout, os.Stderr))
	}
	os.Exit(RunMergeHelper(os.Stdin, os.Stdout, os.Stderr))
}

// useTestMergeHelper 는 merge-helper 를 namespace 대신 이 시험 바이너리로 띄우게 한다. t.Setenv 를 쓰므로 부르는
// 시험은 t.Parallel 을 안 쓴다.
func useTestMergeHelper(t testing.TB, mode string) {
	t.Helper()
	t.Setenv("ENODE_TEST_MERGE_HELPER", "1")
	t.Setenv("ENODE_TEST_MERGE_MODE", mode)
	was := mergeHelperCommand
	mergeHelperCommand = func() ([]string, error) {
		return []string{os.Args[0], "-test.run=^TestMergeHelperProcess$"}, nil
	}
	t.Cleanup(func() { mergeHelperCommand = was })
}

// 제품 기본값이 unshare 로 시작하고 --mount 가 없다. 바꿔 끼운 명령만 보면 기본값을 unshare 없는 명령으로 바꿔도
// 초록이다 — 기본값을 직접 본다.
func TestMergeHelperArgv(t *testing.T) {
	got := mergeHelperArgv("/opt/enode")
	want := []string{"unshare", "--user", "--map-root-user", "--map-auto", "--fork", "--kill-child", "--",
		"/opt/enode", "merge-helper"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("argv = %q", got)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	def, err := mergeHelperCommand()
	if err != nil || !reflect.DeepEqual(def, mergeHelperArgv(exe)) || def[0] != "unshare" {
		t.Fatalf("the product default = %q %v", def, err)
	}
	for _, a := range def {
		if a == "--mount" {
			t.Fatal("the merge helper opens a mount namespace")
		}
	}
}

// helperTree 는 한 filesystem 의 upper · lower · trash 다. upper 는 a.txt 를 바꾸고 · new.txt 를 더하고 · 종류가 바뀐
// 항목 둘 (d1/same 은 파일에서 디렉터리 · d2/same 은 디렉터리에서 파일) 과 opaque 디렉터리 op 를 든다. lower 쪽 셋이
// trash 로 간다 — 끝 조각이 같은 둘은 same · same-1 이다.
func helperTree(t *testing.T) mergeHelperRequest {
	t.Helper()
	root := t.TempDir()
	r := mergeHelperRequest{Upper: filepath.Join(root, "upper"), Lower: filepath.Join(root, "lower"),
		Trash: filepath.Join(root, "trash")}
	for _, d := range []string{r.Upper, r.Lower, r.Trash} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write(t, r.Lower, "a.txt", "old a")
	write(t, r.Lower, "d1/same", "old file")
	write(t, r.Lower, "d2/same/inside", "old dir")
	write(t, r.Lower, "op/old", "old")
	write(t, r.Upper, "a.txt", "new a")
	write(t, r.Upper, "new.txt", "new")
	write(t, r.Upper, "d1/same/inside", "new dir")
	write(t, r.Upper, "d2/same", "new file")
	write(t, r.Upper, "op/fresh", "fresh")
	if err := unix.Lsetxattr(filepath.Join(r.Upper, "op"), "user.overlay.opaque", []byte("y"), 0); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestMergeHelper_Calls(t *testing.T) {
	t.Run("preflight passes", func(t *testing.T) {
		useTestMergeHelper(t, "")
		r := helperTree(t)
		r.Op = "preflight"
		if _, err := callMergeHelper(r); err != nil {
			t.Fatalf("preflight = %v", err)
		}
	})
	t.Run("preflight mismatch carries merge's sentence", func(t *testing.T) {
		useTestMergeHelper(t, "")
		r := helperTree(t)
		r.Op = "preflight"
		if err := unix.Lsetxattr(filepath.Join(r.Upper, "new.txt"), "user.overlay.metacopy", nil, 0); err != nil {
			t.Fatal(err)
		}
		want := merge.Preflight(merge.Paths{Upper: r.Upper, Lower: r.Lower, Trash: r.Trash})
		_, err := callMergeHelper(r)
		var pe *helperPreflightError
		if !errors.As(err, &pe) || pe.Check != merge.CheckMark || want == nil || err.Error() != want.Error() {
			t.Fatalf("preflight = %#v, want %v", err, want)
		}
	})
	t.Run("apply merges and moves each lower entry to trash", func(t *testing.T) {
		useTestMergeHelper(t, "")
		r := helperTree(t)
		r.Op = "apply"
		res, err := callMergeHelper(r)
		if err != nil {
			t.Fatal(err)
		}
		for name, want := range map[string]string{"a.txt": "new a", "new.txt": "new", "d1/same/inside": "new dir",
			"d2/same": "new file", "op/fresh": "fresh"} {
			if b, _ := os.ReadFile(filepath.Join(r.Lower, name)); string(b) != want {
				t.Fatalf("lower %s = %q", name, b)
			}
		}
		if _, err := os.Stat(filepath.Join(r.Lower, "op", "old")); !os.IsNotExist(err) {
			t.Fatalf("the opaque directory kept a lower entry: %v", err)
		}
		// 같은 끝 조각의 항목은 -1 로 늘어난다 — 항목마다 옮긴다 (한 디렉터리에 모으지 않는다)
		if got := entries(t, r.Trash); !reflect.DeepEqual(got, []string{"op", "same", "same-1"}) {
			t.Fatalf("trash = %v", got)
		}
		if res.Discarded != 3 || res.Added != 1 || res.Replaced != 1 || res.TypeChanged != 2 || res.OpaqueDirs != 1 {
			t.Fatalf("result = %+v", res)
		}
	})
	t.Run("an op error", func(t *testing.T) {
		useTestMergeHelper(t, "")
		r := helperTree(t)
		r.Op = "apply"
		blocker := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(blocker, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		r.Trash = filepath.Join(blocker, "trash")
		_, err := callMergeHelper(r)
		var oe *helperOpError
		if !errors.As(err, &oe) || !strings.HasPrefix(err.Error(), "merge: discard ") {
			t.Fatalf("apply = %#v", err)
		}
	})
	t.Run("an io error", func(t *testing.T) {
		useTestMergeHelper(t, "")
		r := helperTree(t)
		r.Op = "apply"
		r.Lower = filepath.Join(t.TempDir(), "missing")
		_, err := callMergeHelper(r)
		var pe *helperPreflightError
		var oe *helperOpError
		if err == nil || errors.As(err, &pe) || errors.As(err, &oe) || !strings.HasPrefix(err.Error(), "merge: open lower") {
			t.Fatalf("apply = %#v", err)
		}
	})
	t.Run("an unknown op", func(t *testing.T) {
		useTestMergeHelper(t, "")
		r := helperTree(t)
		r.Op = "shred"
		if _, err := callMergeHelper(r); err == nil || err.Error() != `merge helper: unknown op "shred"` {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("no answer carries the stderr tail", func(t *testing.T) {
		useTestMergeHelper(t, "die")
		r := helperTree(t)
		r.Op = "preflight"
		_, err := callMergeHelper(r)
		if err == nil || !strings.HasPrefix(err.Error(), "cannot run the merge helper: ") ||
			!strings.HasSuffix(err.Error(), "(newuidmap: write to uid_map failed: Operation not permitted)") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("an answer that is not json", func(t *testing.T) {
		useTestMergeHelper(t, "garbage")
		r := helperTree(t)
		r.Op = "preflight"
		if _, err := callMergeHelper(r); err == nil || !strings.HasPrefix(err.Error(), "cannot run the merge helper: ") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("the helper cannot start", func(t *testing.T) {
		was := mergeHelperCommand
		mergeHelperCommand = func() ([]string, error) { return []string{filepath.Join(t.TempDir(), "no-such")}, nil }
		t.Cleanup(func() { mergeHelperCommand = was })
		if _, err := callMergeHelper(helperTree(t)); err == nil ||
			!strings.HasPrefix(err.Error(), "cannot run the merge helper: ") {
			t.Fatalf("err = %v", err)
		}
		mergeHelperCommand = func() ([]string, error) { return nil, errors.New("no executable") }
		if _, err := callMergeHelper(helperTree(t)); err == nil || err.Error() != "cannot run the merge helper: no executable" {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("stderr does not blur the answer", func(t *testing.T) {
		useTestMergeHelper(t, "noisy")
		r := helperTree(t)
		r.Op = "preflight"
		if _, err := callMergeHelper(r); err != nil {
			t.Fatalf("preflight = %v", err)
		}
	})
}

// JSON 이 아닌 요청은 exit 1 이고 응답 한 줄이 io 오류다. 입구를 곧바로 부른다.
func TestRunMergeHelper_ABadRequest(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := RunMergeHelper(strings.NewReader("not json"), &out, &errOut); code != 1 {
		t.Fatalf("exit = %d", code)
	}
	var resp mergeHelperResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil || resp.Kind != "io" ||
		!strings.HasPrefix(resp.Error, "merge helper: cannot read the request: ") {
		t.Fatalf("response = %s %v", out.String(), err)
	}
	if lines := strings.Count(out.String(), "\n"); lines != 1 {
		t.Fatalf("the helper wrote %d lines", lines)
	}
}
