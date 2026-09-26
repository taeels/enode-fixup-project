//go:build linux

package enode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/taeels/enode/internal/scratch"
)

// TestTrashHelperProcess 는 시험이 namespace 없이 띄우는 trash-helper 다. 모드가 없으면 진짜
// RunTrashHelper 를 돈다 — 우선순위도 진짜로 내린다 (launcher 가 새 프로세스 그룹을 만든다).
func TestTrashHelperProcess(t *testing.T) {
	if os.Getenv("ENODE_TEST_TRASH_HELPER") != "1" {
		return
	}
	args := os.Args
	for i, a := range args {
		if a == "--" {
			args = args[i+1:]
			break
		}
	}
	switch os.Getenv("ENODE_TEST_TRASH_MODE") {
	case "silent":
		fmt.Fprint(os.Stderr, "newuidmap: write to uid_map failed: Operation not permitted")
		os.Exit(1)
	case "left":
		fmt.Println(`{"measured":{"bytes":4096,"entries":3}}`)
		fmt.Println(`{"removed":false,"left":["e/upper/mnt"]}`)
		os.Exit(1)
	case "no-result":
		fmt.Println(`{"measured":{"bytes":1,"entries":1}}`)
		os.Exit(0)
	case "hang":
		time.Sleep(time.Minute)
		os.Exit(0)
	}
	os.Exit(RunTrashHelper(args, os.Stdout, os.Stderr))
}

// useTestTrashHelper 는 launcher 가 namespace 대신 이 시험 바이너리를 띄우게 한다.
func useTestTrashHelper(t *testing.T, mode string) {
	t.Helper()
	t.Setenv("ENODE_TEST_TRASH_HELPER", "1")
	t.Setenv("ENODE_TEST_TRASH_MODE", mode)
	was := trashHelperCommand
	trashHelperCommand = func(trash, entry string) ([]string, error) {
		return []string{os.Args[0], "-test.run=^TestTrashHelperProcess$", "--", trash, entry}, nil
	}
	t.Cleanup(func() { trashHelperCommand = was })
}

// quietPriority 는 이 프로세스에서 부르는 RunTrashHelper 가 go test 의 그룹을 안 건드리게 한다.
func quietPriority(t *testing.T) *int {
	t.Helper()
	calls := 0
	was := lowerTrashPriority
	lowerTrashPriority = func() error { calls++; return errors.New("priority left as is") }
	t.Cleanup(func() { lowerTrashPriority = was })
	return &calls
}

func trashEntry(t *testing.T) (trash, entry string) {
	t.Helper()
	trash = filepath.Join(t.TempDir(), "trash")
	entry = "enode-runc-1"
	for _, f := range []string{"upper/a", "upper/sub/b", "work/work/c"} {
		p := filepath.Join(trash, entry, f)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	workwork := filepath.Join(trash, entry, "work", "work")
	if err := os.Chmod(workwork, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(workwork, 0o700) })
	return trash, entry
}

func trashLines(t *testing.T, out string) []trashLine {
	t.Helper()
	var lines []trashLine
	dec := json.NewDecoder(strings.NewReader(out))
	for dec.More() {
		var l trashLine
		if err := dec.Decode(&l); err != nil {
			t.Fatalf("stdout is not the line protocol: %v\n%s", err, out)
		}
		lines = append(lines, l)
	}
	return lines
}

// 보통 권한의 trash 에서 줄 둘(측정 · 지움)과 exit 0. 권한 000 디렉터리도 지운다.
func TestRunTrashHelperMeasuresThenRemoves(t *testing.T) {
	calls := quietPriority(t)
	trash, entry := trashEntry(t)
	var out, errOut bytes.Buffer
	if code := RunTrashHelper([]string{trash, entry}, &out, &errOut); code != 0 {
		t.Fatalf("exit %d\nstdout %s\nstderr %s", code, out.String(), errOut.String())
	}
	lines := trashLines(t, out.String())
	if len(lines) != 2 || lines[0].Measured == nil || lines[1].Removed == nil || !*lines[1].Removed {
		t.Fatalf("lines = %s", out.String())
	}
	if m := lines[0].Measured; m.Entries != 8 || m.Bytes <= 0 {
		t.Fatalf("measured = %+v", m)
	}
	if _, err := os.Lstat(filepath.Join(trash, entry)); !os.IsNotExist(err) {
		t.Fatalf("the entry survived: %v", err)
	}
	if *calls != 1 || !strings.Contains(errOut.String(), "priority left as is") {
		t.Fatalf("priority calls = %d stderr = %q", *calls, errOut.String())
	}
}

// 인자가 모자라거나 이름이 틀리면 오류 줄 하나와 exit 1. 아무것도 안 지운다.
func TestRunTrashHelperRejectsBadInput(t *testing.T) {
	quietPriority(t)
	trash, entry := trashEntry(t)
	for _, tc := range []struct {
		args []string
		want string
	}{
		{nil, "usage: enode trash-helper <trash> <entry>"},
		{[]string{trash}, "usage: enode trash-helper <trash> <entry>"},
		{[]string{trash, ".."}, "invalid trash entry name"},
		{[]string{trash, entry + "/upper"}, "invalid trash entry name"},
		{[]string{filepath.Join(trash, entry, "upper", "a"), "x"}, "trash is not a plain directory"},
	} {
		var out bytes.Buffer
		if code := RunTrashHelper(tc.args, &out, io.Discard); code != 1 {
			t.Errorf("%v: exit %d", tc.args, code)
		}
		lines := trashLines(t, out.String())
		if len(lines) != 1 || !strings.Contains(lines[0].Error, tc.want) {
			t.Errorf("%v: lines = %s", tc.args, out.String())
		}
	}
	if _, err := os.Stat(filepath.Join(trash, entry, "upper", "a")); err != nil {
		t.Fatalf("a rejected call removed something: %v", err)
	}
}

// 걷다가 실패하면 오류 줄이다 — 측정 줄 없이 끝날 수도 있다.
func TestRunTrashHelperCarriesAWalkFailure(t *testing.T) {
	quietPriority(t)
	trash := filepath.Join(t.TempDir(), "trash")
	if err := os.MkdirAll(trash, 0o755); err != nil {
		t.Fatal(err)
	}
	// 항목이 없으면 측정이 비고 지울 것도 없다 — 그것은 다 지운 것이다
	var out bytes.Buffer
	if code := RunTrashHelper([]string{trash, "gone"}, &out, io.Discard); code != 0 {
		t.Fatalf("exit %d: %s", code, out.String())
	}
	if os.Geteuid() == 0 {
		return
	}
	// trash 를 못 열면 측정이 실패한다
	if err := os.Chmod(trash, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(trash, 0o755) })
	out.Reset()
	if code := RunTrashHelper([]string{trash, "x"}, &out, io.Discard); code != 1 {
		t.Fatalf("exit %d", code)
	}
	if lines := trashLines(t, out.String()); len(lines) != 1 || lines[0].Error == "" {
		t.Fatalf("lines = %s", out.String())
	}
}

// helper 의 argv — runtime-helper 와 같은 uid 매핑이고 마운트 namespace 를 안 연다.
func TestTrashHelperArgvOpensNoMountNamespace(t *testing.T) {
	got := trashHelperArgv("/opt/enode", "/s/trash", "enode-runc-1")
	want := []string{"unshare", "--user", "--map-root-user", "--map-auto", "--fork", "--kill-child",
		"--", "/opt/enode", "trash-helper", "/s/trash", "enode-runc-1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("argv = %v", got)
	}
	for _, a := range got {
		if a == "--mount" {
			t.Fatal("the trash helper opens a mount namespace")
		}
	}
	runtime := runcOverlayHelperArgv("/opt/enode")
	if !reflect.DeepEqual(runtime[:4], got[:4]) {
		t.Fatalf("the uid mapping differs from the runtime helper: %v vs %v", runtime, got)
	}
	argv, err := trashHelperCommand("/s/trash", "e")
	if err != nil || argv[len(argv)-3] != "trash-helper" {
		t.Fatalf("argv = %v err = %v", argv, err)
	}
}

// launcher 가 진짜 helper 프로세스를 띄워 줄을 읽는다 — 측정은 콜백으로, 다 지웠으면 nil.
func TestTrashLauncherRunsTheHelperAndReadsItsLines(t *testing.T) {
	useTestTrashHelper(t, "")
	trash, entry := trashEntry(t)
	var got []scratch.Size
	err := TrashLauncher(scratch.Trash{Dir: trash})(context.Background(), entry, func(s scratch.Size) { got = append(got, s) })
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Entries != 8 {
		t.Fatalf("measured = %+v", got)
	}
	if _, err := os.Lstat(filepath.Join(trash, entry)); !os.IsNotExist(err) {
		t.Fatalf("the entry survived: %v", err)
	}
	// 오류 줄은 그 문구 그대로다
	err = TrashLauncher(scratch.Trash{Dir: trash})(context.Background(), "..", func(scratch.Size) {})
	var launch *scratch.LaunchError
	if err == nil || errors.As(err, &launch) || !strings.Contains(err.Error(), "invalid trash entry name") {
		t.Fatalf("err = %v", err)
	}
}

func TestTrashLauncherOutcomes(t *testing.T) {
	for _, tc := range []struct {
		mode   string
		launch bool   // LaunchError 여야 한다
		want   string // 오류 문구의 일부
	}{
		{"silent", true, "newuidmap: write to uid_map failed"},
		{"no-result", false, "trash helper ended without a result"},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			useTestTrashHelper(t, tc.mode)
			err := TrashLauncher(scratch.Trash{Dir: t.TempDir()})(context.Background(), "e", func(scratch.Size) {})
			var launch *scratch.LaunchError
			if err == nil || errors.As(err, &launch) != tc.launch || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v", err)
			}
		})
	}
	t.Run("left", func(t *testing.T) {
		useTestTrashHelper(t, "left")
		var measured scratch.Size
		err := TrashLauncher(scratch.Trash{Dir: t.TempDir()})(context.Background(), "e", func(s scratch.Size) { measured = s })
		var left *scratch.LeftError
		if !errors.As(err, &left) || !reflect.DeepEqual(left.Left, []string{"e/upper/mnt"}) || measured.Bytes != 4096 {
			t.Fatalf("err = %v measured = %+v", err, measured)
		}
	})
	t.Run("cannot start", func(t *testing.T) {
		was := trashHelperCommand
		trashHelperCommand = func(string, string) ([]string, error) {
			return []string{filepath.Join(t.TempDir(), "no-such-helper")}, nil
		}
		t.Cleanup(func() { trashHelperCommand = was })
		err := TrashLauncher(scratch.Trash{})(context.Background(), "e", func(scratch.Size) {})
		var launch *scratch.LaunchError
		if !errors.As(err, &launch) || !strings.Contains(err.Error(), "start trash helper") {
			t.Fatalf("err = %v", err)
		}
		trashHelperCommand = func(string, string) ([]string, error) { return nil, errors.New("no executable") }
		if err := TrashLauncher(scratch.Trash{})(context.Background(), "e", func(scratch.Size) {}); !errors.As(err, &launch) {
			t.Fatalf("err = %v", err)
		}
	})
}

// ctx 가 끝나면 helper 의 그룹을 죽이고 곧 돌아온다.
func TestTrashLauncherKillsTheHelperWithItsContext(t *testing.T) {
	useTestTrashHelper(t, "hang")
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	began := time.Now()
	err := TrashLauncher(scratch.Trash{Dir: t.TempDir()})(ctx, "e", func(scratch.Size) {})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v", err)
	}
	if took := time.Since(began); took > 10*time.Second {
		t.Fatalf("the helper outlived its context by %v", took)
	}
}

func TestTrashHelperFailureText(t *testing.T) {
	exit := errors.New("exit status 1")
	for _, tc := range []struct {
		err    error
		stderr string
		want   string
	}{
		{exit, "boom", "x: exit status 1: boom"},
		{exit, "", "x: exit status 1"},
		{nil, "boom", "x: boom"},
		{nil, "", "x"},
	} {
		if got := trashHelperFailure("x", tc.err, tc.stderr).Error(); got != tc.want {
			t.Errorf("got %q want %q", got, tc.want)
		}
	}
}

// 기동 청소 — 잠금을 쥔 사람이 없는 작업 폴더만 trash 로 간다.
func TestSweepOrphanSessionsMovesOnlyWhatNobodyHolds(t *testing.T) {
	scratchDir := t.TempDir()
	held := filepath.Join(scratchDir, "enode-runc-held")
	dead := filepath.Join(scratchDir, "enode-runc-dead")
	young := filepath.Join(scratchDir, "enode-runc-young")
	smoke := filepath.Join(scratchDir, "enode-smoke-io-1")
	for _, d := range []string{held, dead, young, smoke} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	lock, err := scratch.HoldSession(held)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = lock.Release() }()
	gone, err := scratch.HoldSession(dead)
	if err != nil {
		t.Fatal(err)
	}
	_ = gone.Release()
	var logs bytes.Buffer
	SweepOrphanSessions(scratchDir, slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	names, err := scratch.TrashIn(scratchDir).Entries()
	if err != nil || !reflect.DeepEqual(names, []string{"enode-runc-dead"}) {
		t.Fatalf("trash = %v err = %v", names, err)
	}
	for _, d := range []string{held, young, smoke} {
		if _, err := os.Stat(d); err != nil {
			t.Fatalf("%s was moved: %v", d, err)
		}
	}
	for _, want := range []string{"orphaned runtime session moved to trash", "too young to judge"} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("log lacks %q:\n%s", want, logs.String())
		}
	}
	// scratch 가 파일이면 찾지 못한 사실을 적고 돌아온다
	file := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	logs.Reset()
	SweepOrphanSessions(file, slog.New(slog.NewTextHandler(&logs, nil)))
	if !strings.Contains(logs.String(), "cannot look for orphaned runtime sessions") {
		t.Fatalf("log = %s", logs.String())
	}
	// 옮길 수 없으면 적고 넘어간다 — trash 자리를 파일이 쥐고 있다
	blocked := t.TempDir()
	if err := os.MkdirAll(filepath.Join(blocked, "enode-runc-x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if l, err := scratch.HoldSession(filepath.Join(blocked, "enode-runc-x")); err == nil {
		_ = l.Release()
	}
	if err := os.WriteFile(filepath.Join(blocked, scratch.TrashName), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	logs.Reset()
	SweepOrphanSessions(blocked, slog.New(slog.NewTextHandler(&logs, nil)))
	if !strings.Contains(logs.String(), "cannot move an orphaned runtime session to trash") {
		t.Fatalf("log = %s", logs.String())
	}
}
