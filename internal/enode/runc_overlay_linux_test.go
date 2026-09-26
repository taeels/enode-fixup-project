//go:build linux

package enode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	execenv "github.com/taeels/enode/internal/environment"
	"github.com/taeels/enode/internal/scratch"
)

func TestRuntimeProtocolHelperProcess(t *testing.T) {
	if os.Getenv("ENODE_TEST_RUNTIME_HELPER") != "1" {
		return
	}
	if os.Getenv("ENODE_TEST_RUNTIME_MODE") == "eof" {
		_, _ = io.WriteString(os.Stderr, "injected helper failure")
		return
	}
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	for {
		var request runtimeWireRequest
		if err := decoder.Decode(&request); err != nil {
			return
		}
		switch request.Op {
		case "open":
			if os.Getenv("ENODE_TEST_RUNTIME_MODE") == "refuse" {
				_ = encoder.Encode(runtimeWireResponse{Op: "opened", Error: "prepared rootfs target /work is missing"})
				continue
			}
			_ = encoder.Encode(runtimeWireResponse{Op: "opened"})
		case "project":
			_ = encoder.Encode(runtimeWireResponse{Op: "projected"})
		case "run":
			if len(request.Process.Argv) > 1 && request.Process.Argv[1] == "wait" {
				var cancel runtimeWireRequest
				_ = decoder.Decode(&cancel)
				_ = encoder.Encode(runtimeWireResponse{Op: "run-done", ExitCode: -1, Error: "canceled"})
				continue
			}
			_ = encoder.Encode(runtimeWireResponse{Op: "stdout", Data: append([]byte("stdout:"), request.Process.Stdin...)})
			_ = encoder.Encode(runtimeWireResponse{Op: "stderr", Data: []byte("stderr")})
			_ = encoder.Encode(runtimeWireResponse{Op: "run-done", ExitCode: 0})
		case "finalize":
			switch os.Getenv("ENODE_TEST_RUNTIME_MODE") {
			case "silent":
				// 답하지 않는다 — stdin 이 닫히거나 죽을 때까지 기다린다
				var next runtimeWireRequest
				_ = decoder.Decode(&next)
				return
			case "deadline":
				_ = encoder.Encode(runtimeWireResponse{Op: "finalized", Error: context.DeadlineExceeded.Error(),
					Finalize: &FinalizeResult{Changed: []string{"before-the-deadline"}}})
			case "broken":
				_ = encoder.Encode(runtimeWireResponse{Op: "finalized", Error: "merged view is gone",
					Finalize: &FinalizeResult{}})
			default:
				// 받은 마감을 돌려준다 — 마감이 요청에 실려 가는지 시험이 본다
				_ = encoder.Encode(runtimeWireResponse{Op: "finalized", Finalize: &FinalizeResult{
					Collected: []string{"fake"}, Changed: []string{request.Finalize.Deadline.UTC().Format(time.RFC3339)}}})
			}
		case "close":
			_ = encoder.Encode(runtimeWireResponse{Op: "closed"})
			return
		}
	}
}

func protocolTestRuntime(t *testing.T, mode string) (*RuncOverlayRuntime, RuntimeSpec, string) {
	t.Helper()
	root := t.TempDir()
	rootfs := filepath.Join(root, "rootfs")
	for _, dir := range []string{rootfs, filepath.Join(rootfs, "bin"), filepath.Join(root, "workspace"),
		filepath.Join(root, "in"), filepath.Join(root, "out"), filepath.Join(root, "scratch")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(rootfs, "bin", "sh"), []byte("stub"), 0o755); err != nil {
		t.Fatal(err)
	}
	doc := execenv.Document{SHA256: "profile", Profile: execenv.Profile{
		Runtime: execenv.RuntimePolicy{Driver: "runc-overlay", WorkspaceTarget: "/work",
			Tmp: execenv.TmpPolicy{Size: "64MiB", Executable: true}},
		RootFS: execenv.RootFSProfile{Locale: "C.UTF-8", User: execenv.RootFSUser{Name: "enode", UID: 1000, GID: 1000}},
		Host:   execenv.HostProfile{Require: execenv.HostRequirement{SubUIDSize: 65536, SubGIDSize: 65536}},
	}}
	binding := execenv.Binding{Scratch: filepath.Join(root, "scratch"), Workspace: filepath.Join(root, "workspace")}
	runtimeImpl := &RuncOverlayRuntime{doc: doc, binding: binding,
		manifest: execenv.Manifest{Profile: execenv.ManifestProfile{SHA256: doc.SHA256}}, rootfs: rootfs,
		trash: scratch.TrashIn(binding.Scratch)}
	runtimeImpl.command = func() *exec.Cmd {
		cmd := exec.Command(os.Args[0], "-test.run=^TestRuntimeProtocolHelperProcess$")
		cmd.Env = append(os.Environ(), "ENODE_TEST_RUNTIME_HELPER=1", "ENODE_TEST_RUNTIME_MODE="+mode)
		return cmd
	}
	record := &execenv.Record{Runtime: "runc-overlay"}
	return runtimeImpl, RuntimeSpec{Dir: binding.Workspace, In: filepath.Join(root, "in"), Out: filepath.Join(root, "out"), Record: record}, rootfs
}

func TestRuncOverlaySessionProtocol(t *testing.T) {
	runtimeImpl, spec, rootfs := protocolTestRuntime(t, "")
	session, err := runtimeImpl.Open(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	paths := session.Paths()
	if paths.Dir != "/work" || paths.In != runtimeInTarget || paths.Out != runtimeOutTarget {
		t.Fatalf("unexpected paths: %+v", paths)
	}
	framework := filepath.Join(t.TempDir(), "framework")
	if err := os.WriteFile(framework, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	instrumentation := t.TempDir()
	projection, err := session.Project(context.Background(), FrameworkProjectionSpec{
		HarnessExecutable: framework, EnodeExecutable: framework,
		Instrumentation: instrumentation, CredentialHelper: framework,
	})
	if err != nil {
		t.Fatal(err)
	}
	if projection.HarnessExecutable != runtimeHarnessTarget || projection.CredentialHelper != runtimeCredentialTarget {
		t.Fatalf("unexpected projection: %+v", projection)
	}
	var stdout, stderr bytes.Buffer
	code, err := session.Run(context.Background(), ProcessSpec{Argv: []string{"/bin/sh", "ok"},
		Stdin: strings.NewReader("input"), Stdout: &stdout, Stderr: &stderr})
	if err != nil || code != 0 || stdout.String() != "stdout:input" || stderr.String() != "stderr" {
		t.Fatalf("run code=%d err=%v stdout=%q stderr=%q", code, err, stdout.String(), stderr.String())
	}
	deadline := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	fctx, cancel := context.WithDeadline(context.Background(), deadline)
	finalized, err := session.Finalize(fctx, FinalizeSpec{})
	cancel()
	if err != nil || !reflect.DeepEqual(finalized.Collected, []string{"fake"}) {
		t.Fatalf("finalize=%+v err=%v", finalized, err)
	}
	if want := deadline.Format(time.RFC3339); !reflect.DeepEqual(finalized.Changed, []string{want}) {
		t.Fatalf("the deadline did not travel to the helper: got %v want %s", finalized.Changed, want)
	}
	if session.Environment() != spec.Record {
		t.Fatal("session lost its environment record")
	}
	if err := session.Close(context.Background(), Keep{}); err != nil {
		t.Fatal(err)
	}
	if err := session.Close(context.Background(), Keep{}); err != nil {
		t.Fatalf("close is not idempotent: %v", err)
	}
	if _, err := os.Stat(rootfs); err != nil {
		t.Fatalf("session removed the prepared rootfs: %v", err)
	}
}

func TestRuncOverlaySessionCancellationUsesTheHelperProtocol(t *testing.T) {
	runtimeImpl, spec, _ := protocolTestRuntime(t, "")
	session, err := runtimeImpl.Open(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	code, err := session.Run(ctx, ProcessSpec{Argv: []string{"/bin/sh", "wait"}})
	if err == nil || code != -1 {
		t.Fatalf("canceled run code=%d err=%v", code, err)
	}
	if err := session.Close(context.Background(), Keep{}); err != nil {
		t.Fatal(err)
	}
}

// 마감이 지나고 helperGrace 가 지나도 답하지 않는 helper 는 죽인다. 그 뒤의 닫기는
// helper 에 말하지 않는다. 작업 폴더는 abort 가 trash 로 옮겼다 (business-rules.md 1절 ③).
func TestRuncOverlayFinalizeAbortsAHelperThatDoesNotAnswer(t *testing.T) {
	grace := helperGrace
	helperGrace = 20 * time.Millisecond
	t.Cleanup(func() { helperGrace = grace })
	runtimeImpl, spec, _ := protocolTestRuntime(t, "silent")
	session, err := runtimeImpl.Open(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	runRoot := session.(*runcOverlaySession).runRoot
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	began := time.Now()
	_, err = session.Finalize(ctx, FinalizeSpec{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want the deadline", err)
	}
	if took := time.Since(began); took > 5*time.Second {
		t.Fatalf("finalize waited %v for a silent helper", took)
	}
	if err := session.Close(context.Background(), Keep{}); err != nil {
		t.Fatalf("close after abort failed: %v", err)
	}
	assertInTrash(t, runtimeImpl, runRoot)
}

// helper 가 제 마감에 멈춰 답하면 모은 것은 남기고 마감으로 돌려준다.
func TestRuncOverlayFinalizeKeepsWhatTheHelperGatheredBeforeItsDeadline(t *testing.T) {
	runtimeImpl, spec, _ := protocolTestRuntime(t, "deadline")
	session, err := runtimeImpl.Open(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close(context.Background(), Keep{}) //nolint:errcheck
	result, err := session.Finalize(context.Background(), FinalizeSpec{Deadline: time.Now()})
	if !errors.Is(err, context.DeadlineExceeded) || !reflect.DeepEqual(result.Changed, []string{"before-the-deadline"}) {
		t.Fatalf("result %+v err %v", result, err)
	}
}

// 마감이 아닌 helper 의 오류는 그 문구 그대로다. 앞에 머리말을 붙이는 것은 Worker 다.
func TestRuncOverlayFinalizeCarriesTheHelperError(t *testing.T) {
	runtimeImpl, spec, _ := protocolTestRuntime(t, "broken")
	session, err := runtimeImpl.Open(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close(context.Background(), Keep{}) //nolint:errcheck
	if _, err := session.Finalize(context.Background(), FinalizeSpec{}); err == nil || err.Error() != "merged view is gone" {
		t.Fatalf("err = %v", err)
	}
}

func TestRuncOverlayOpenIncludesHelperStderr(t *testing.T) {
	runtimeImpl, spec, _ := protocolTestRuntime(t, "eof")
	_, err := runtimeImpl.Open(context.Background(), spec)
	if err == nil || !strings.Contains(err.Error(), "injected helper failure") {
		t.Fatalf("missing helper diagnostic: %v", err)
	}
}

func TestRuncOverlayConstructorFailsClosed(t *testing.T) {
	rootfs := t.TempDir()
	doc := execenv.Document{SHA256: "profile", Profile: execenv.Profile{Runtime: execenv.RuntimePolicy{Driver: "native"}}}
	manifest := execenv.Manifest{Profile: execenv.ManifestProfile{SHA256: "profile"}}
	if _, err := newRuncOverlayRuntime(doc, execenv.Binding{}, manifest, rootfs); err == nil {
		t.Fatal("accepted the native driver")
	}
	doc.Profile.Runtime.Driver = "runc-overlay"
	manifest.Profile.SHA256 = "other"
	if _, err := newRuncOverlayRuntime(doc, execenv.Binding{}, manifest, rootfs); err == nil {
		t.Fatal("accepted a mismatched manifest")
	}
	manifest.Profile.SHA256 = doc.SHA256
	if _, err := newRuncOverlayRuntime(doc, execenv.Binding{}, manifest, filepath.Join(rootfs, "missing")); err == nil {
		t.Fatal("accepted a missing rootfs")
	}
	if _, err := newRuncOverlayRuntime(doc, execenv.Binding{}, manifest, rootfs); err != nil {
		t.Fatalf("rejected a valid constructor input: %v", err)
	}
}

func TestRuncOverlayHelperDoesNotCreateAnOuterPIDNamespace(t *testing.T) {
	argv := runcOverlayHelperArgv("/bin/enode")
	joined := strings.Join(argv, " ")
	if strings.Contains(joined, "--pid") {
		t.Fatalf("outer PID namespace would leave runc with the wrong /proc: %s", joined)
	}
	for _, want := range []string{"--user", "--map-auto", "--mount", "--kill-child", "/bin/enode", "runtime-helper"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("helper argv is missing %s: %s", want, joined)
		}
	}
}

func TestRuntimeInputAndProjectionValidation(t *testing.T) {
	if _, err := readRuntimeStdin(bytes.NewReader(make([]byte, maxRuntimeStdin+1))); err == nil {
		t.Fatal("accepted oversized runtime stdin")
	}
	if got, err := readRuntimeStdin(strings.NewReader("small")); err != nil || string(got) != "small" {
		t.Fatalf("stdin=%q err=%v", got, err)
	}
	if got, err := readRuntimeStdin(nil); err != nil || got != nil {
		t.Fatalf("nil stdin=%q err=%v", got, err)
	}

	rootfs := t.TempDir()
	for _, executable := range []string{"bin/sh", "usr/bin/env", "usr/bin/bash"} {
		path := filepath.Join(rootfs, executable)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("stub"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	sourceDir := t.TempDir()
	script := filepath.Join(sourceDir, "script")
	if err := os.WriteFile(script, []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	symlink := filepath.Join(sourceDir, "link")
	if err := os.Symlink(script, symlink); err != nil {
		t.Fatal(err)
	}
	if got, err := resolveProjectedExecutable(rootfs, symlink, false); err != nil || got != script {
		t.Fatalf("resolved executable=%q err=%v", got, err)
	}
	if got, err := resolveProjectedExecutable(rootfs, "", true); err != nil || got != "" {
		t.Fatalf("optional executable=%q err=%v", got, err)
	}
	if _, err := resolveProjectedExecutable(rootfs, "", false); err == nil {
		t.Fatal("accepted an empty required executable")
	}
	if got, err := resolveProjectedDirectory(sourceDir); err != nil || got != sourceDir {
		t.Fatalf("resolved directory=%q err=%v", got, err)
	}
	if _, err := resolveProjectedDirectory(""); err == nil {
		t.Fatal("accepted an empty projection directory")
	}
	if _, err := resolveProjectedDirectory(script); err == nil {
		t.Fatal("accepted a file as a projection directory")
	}
	if !rootfsExecutableExists(rootfs, "bash") || rootfsExecutableExists(rootfs, "missing") {
		t.Fatal("rootfs executable lookup is inconsistent")
	}
	badExecutable := filepath.Join(sourceDir, "not-a-program")
	if err := os.WriteFile(badExecutable, []byte("plain text"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := checkProjectedExecutableDependencies(rootfs, badExecutable); err == nil {
		t.Fatal("accepted a non-ELF executable without a shebang")
	}
	for _, dir := range []string{"lib/x86_64-linux-gnu", "lib/aarch64-linux-gnu"} {
		if err := os.MkdirAll(filepath.Join(rootfs, dir), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(rootfs, dir, "libgate.so"), []byte("stub"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if !rootfsLibraryExists(rootfs, "libgate.so") || rootfsLibraryExists(rootfs, "libmissing.so") {
		t.Fatal("rootfs library lookup is inconsistent")
	}
	if _, err := beneathRoot(rootfs, "relative"); err == nil {
		t.Fatal("accepted a relative rootfs path")
	}
	if _, err := beneathRoot(rootfs, "/"); err == nil {
		t.Fatal("accepted the rootfs root as a projection target")
	}
	if got, err := beneathRoot(rootfs, "/bin/sh"); err != nil || got != filepath.Join(rootfs, "bin/sh") {
		t.Fatalf("beneath root=%q err=%v", got, err)
	}
}

func TestRuntimeHelperProtocolRejectsIncompleteRequests(t *testing.T) {
	requests := []runtimeWireRequest{
		{Op: "open"}, {Op: "project"}, {Op: "run"}, {Op: "harvest"}, {Op: "unknown"}, {Op: "close"},
	}
	var input bytes.Buffer
	encoder := json.NewEncoder(&input)
	for _, request := range requests {
		if err := encoder.Encode(request); err != nil {
			t.Fatal(err)
		}
	}
	var output, stderr bytes.Buffer
	if code := RunRuncOverlayHelper(&input, &output, &stderr); code != 0 {
		t.Fatalf("helper code=%d stderr=%q", code, stderr.String())
	}
	decoder := json.NewDecoder(&output)
	for range requests {
		var response runtimeWireResponse
		if err := decoder.Decode(&response); err != nil {
			t.Fatal(err)
		}
		if response.Op != "closed" && response.Error == "" {
			t.Fatalf("incomplete request did not fail: %+v", response)
		}
	}
	if code := RunRuncOverlayHelper(strings.NewReader("{"), io.Discard, io.Discard); code != 1 {
		t.Fatalf("malformed helper input returned %d", code)
	}
}

func TestRuntimeHelperWritersAndValidation(t *testing.T) {
	var wire bytes.Buffer
	helper := &overlayRuntimeHelper{encoder: json.NewEncoder(&wire), errOut: io.Discard}
	writer := runtimeResponseWriter{helper: helper, op: "stdout"}
	payload := bytes.Repeat([]byte("x"), (32<<10)+1)
	if n, err := writer.Write(payload); err != nil || n != len(payload) {
		t.Fatalf("write n=%d err=%v", n, err)
	}
	decoder := json.NewDecoder(&wire)
	var first, second runtimeWireResponse
	if err := decoder.Decode(&first); err != nil {
		t.Fatal(err)
	}
	if err := decoder.Decode(&second); err != nil {
		t.Fatal(err)
	}
	if len(first.Data) != 32<<10 || len(second.Data) != 1 {
		t.Fatalf("unexpected response chunks: %d %d", len(first.Data), len(second.Data))
	}

	executable := filepath.Join(t.TempDir(), "tool")
	if err := os.WriteFile(executable, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	projection := runtimeWireProjection{Harness: executable, Enode: executable, Instrumentation: directory}
	if err := validateHelperProjection(projection); err != nil {
		t.Fatal(err)
	}
	projection.Harness = directory
	if err := validateHelperProjection(projection); err == nil {
		t.Fatal("accepted a directory as an executable projection")
	}
	if !pathsOverlap(directory, filepath.Join(directory, "child")) || pathsOverlap(directory, t.TempDir()) {
		t.Fatal("path overlap classification is inconsistent")
	}
}

// helper 의 cleanup 은 unmount 만 한다. 작업 폴더는 밖의 rename 이 trash 로 옮긴다
// (business-rules.md 1절 ④) — 부모가 죽어 최종 방어 경로로 불려도 지우지 않는다.
func TestRuntimeHelperCleanupIsIdempotentAndRemovesNothing(t *testing.T) {
	runRoot := filepath.Join(t.TempDir(), "run")
	if err := os.MkdirAll(filepath.Join(runRoot, "upper"), 0o755); err != nil {
		t.Fatal(err)
	}
	h := &overlayRuntimeHelper{open: &runtimeWireOpen{RunRoot: runRoot}}
	if err := h.cleanup(); err != nil {
		t.Fatal(err)
	}
	if err := h.cleanup(); err != nil {
		t.Fatalf("second cleanup failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(runRoot, "upper")); err != nil {
		t.Fatalf("the helper removed the runtime scratch: %v", err)
	}
}

// assertInTrash 는 작업 폴더가 원래 자리에 없고 trash 에 같은 이름으로 있으며 잠금이 풀렸는지 본다.
func assertInTrash(t *testing.T, r *RuncOverlayRuntime, runRoot string) {
	t.Helper()
	if _, err := os.Stat(runRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the work folder is still in scratch: %v", err)
	}
	moved := filepath.Join(r.trash.Dir, filepath.Base(runRoot))
	if _, err := os.Stat(filepath.Join(moved, scratch.SessionLockName)); err != nil {
		t.Fatalf("the work folder is not in trash with its lock file: %v", err)
	}
	lock, err := scratch.HoldSession(moved)
	if err != nil {
		t.Fatalf("the session lock was not released: %v", err)
	}
	_ = lock.Release()
}

// sessions 는 scratch 에 남은 작업 폴더다 — 실패한 Open 뒤에는 없어야 한다.
func sessions(t *testing.T, r *RuncOverlayRuntime) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(r.binding.Scratch, runcSessionPrefix+"*"))
	if err != nil {
		t.Fatal(err)
	}
	return matches
}

// 보통 닫기 — helper 를 닫은 뒤 작업 폴더가 trash 로 가고 잠금이 풀린다. 연 동안에는 잠금을
// 쥐고 있어 기동 청소가 남은 것으로 보지 않는다.
func TestRuncOverlayCloseMovesTheSessionToTrash(t *testing.T) {
	runtimeImpl, spec, _ := protocolTestRuntime(t, "")
	session, err := runtimeImpl.Open(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	runRoot := session.(*runcOverlaySession).runRoot
	if !strings.HasPrefix(filepath.Base(runRoot), runcSessionPrefix) {
		t.Fatalf("run root %s", runRoot)
	}
	orphans, _, err := scratch.Orphans(runtimeImpl.binding.Scratch, runcSessionPrefix, time.Now().Add(2*time.Hour))
	if err != nil || len(orphans) != 0 {
		t.Fatalf("an open session looks orphaned: %v %v", orphans, err)
	}
	if err := session.Close(context.Background(), Keep{}); err != nil {
		t.Fatal(err)
	}
	assertInTrash(t, runtimeImpl, runRoot)
}

// Open 의 실패 갈래도 trash 로 간다 — helper 가 open 을 거절했을 때 · helper 를 띄우기 전에.
func TestRuncOverlayOpenFailuresMoveTheSessionToTrash(t *testing.T) {
	t.Run("helper refuses", func(t *testing.T) {
		runtimeImpl, spec, _ := protocolTestRuntime(t, "refuse")
		_, err := runtimeImpl.Open(context.Background(), spec)
		if err == nil || !strings.Contains(err.Error(), "open runtime namespace: prepared rootfs target") {
			t.Fatalf("err = %v", err)
		}
		if left := sessions(t, runtimeImpl); len(left) != 0 {
			t.Fatalf("scratch still holds %v", left)
		}
		names, _ := runtimeImpl.trash.Entries()
		if len(names) != 1 {
			t.Fatalf("trash = %v", names)
		}
		assertInTrash(t, runtimeImpl, filepath.Join(runtimeImpl.binding.Scratch, names[0]))
	})
	t.Run("before the helper", func(t *testing.T) {
		runtimeImpl, spec, _ := protocolTestRuntime(t, "")
		runtimeImpl.doc.Profile.Runtime.Tmp.Size = "lots"
		if _, err := runtimeImpl.Open(context.Background(), spec); err == nil {
			t.Fatal("opened with a bad tmp size")
		}
		names, _ := runtimeImpl.trash.Entries()
		if len(sessions(t, runtimeImpl)) != 0 || len(names) != 1 {
			t.Fatalf("scratch %v trash %v", sessions(t, runtimeImpl), names)
		}
	})
	t.Run("helper cannot start", func(t *testing.T) {
		runtimeImpl, spec, _ := protocolTestRuntime(t, "")
		runtimeImpl.command = func() *exec.Cmd { return exec.Command(filepath.Join(t.TempDir(), "no-such-helper")) }
		if _, err := runtimeImpl.Open(context.Background(), spec); err == nil || !strings.Contains(err.Error(), "start runtime namespace helper") {
			t.Fatalf("err = %v", err)
		}
		names, _ := runtimeImpl.trash.Entries()
		if len(sessions(t, runtimeImpl)) != 0 || len(names) != 1 {
			t.Fatalf("scratch %v trash %v", sessions(t, runtimeImpl), names)
		}
	})
}

// 옮기기가 실패하면 지우지 않는다 — 오류가 닫기 오류로 남고 폴더는 scratch 에 남는다.
func TestRuncOverlayCloseDoesNotFallBackToRemoving(t *testing.T) {
	runtimeImpl, spec, _ := protocolTestRuntime(t, "")
	session, err := runtimeImpl.Open(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	runRoot := session.(*runcOverlaySession).runRoot
	// trash 자리를 파일이 쥐고 있다 — 설정이나 권한이 틀린 모양
	if err := os.WriteFile(runtimeImpl.trash.Dir, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	err = session.Close(context.Background(), Keep{})
	if err == nil || !strings.Contains(err.Error(), "move runtime session to trash") {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(runRoot); err != nil {
		t.Fatalf("the work folder was removed after a failed move: %v", err)
	}
	// 잠금은 놓았다 — 다음 기동 청소가 거둔다
	orphans, _, _ := scratch.Orphans(runtimeImpl.binding.Scratch, runcSessionPrefix, time.Now())
	if len(orphans) != 1 || orphans[0] != runRoot {
		t.Fatalf("orphans = %v", orphans)
	}
	// Open 의 실패 갈래도 같다 — 옮기기 실패가 원래 오류에 붙는다
	runtimeImpl.doc.Profile.Runtime.Tmp.Size = "lots"
	if _, err := runtimeImpl.Open(context.Background(), spec); err == nil || !strings.Contains(err.Error(), "move runtime session to trash") {
		t.Fatalf("open err = %v", err)
	}
	// 잠금을 쥐기 전의 실패도 같은 길이다 — 놓을 잠금이 없어도 원래 오류가 먼저 실린다
	err = runtimeImpl.discard(t.TempDir(), nil, errors.New("hold session lock: resource busy"))
	if err == nil || !strings.HasPrefix(err.Error(), "hold session lock") || !strings.Contains(err.Error(), "move runtime session to trash") {
		t.Fatalf("discard err = %v", err)
	}
}

func TestRuntimeHelperRunsRuncAndFramesItsOutput(t *testing.T) {
	binDir := t.TempDir()
	runc := filepath.Join(binDir, "runc")
	if err := os.WriteFile(runc, []byte(`#!/bin/sh
case "$*" in
  *" run "*) cat >/dev/null; printf runtime-out; printf runtime-err >&2; exit 0 ;;
  *) exit 0 ;;
esac
`), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	root := t.TempDir()
	bundle, state := filepath.Join(root, "bundle"), filepath.Join(root, "state")
	if err := os.MkdirAll(bundle, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(state, 0o755); err != nil {
		t.Fatal(err)
	}
	var wire bytes.Buffer
	h := &overlayRuntimeHelper{
		encoder: json.NewEncoder(&wire), errOut: io.Discard, bundle: bundle, state: state,
		merged: "/merged", inRO: "/in-ro", running: true,
		open: &runtimeWireOpen{RootFS: "/rootfs", WorkspaceTarget: "/work", In: "/in", Out: "/out",
			UserName: "enode", UID: 1000, GID: 1000, SubUIDSize: 65536, SubGIDSize: 65536,
			Locale: "C.UTF-8", TmpSize: 64 << 20},
	}
	h.runWG.Add(1)
	h.runProcess(runtimeWireProcess{Argv: []string{"/bin/sh", "-c", "true"}, Stdin: []byte("input")})
	h.waitRun()
	decoder := json.NewDecoder(&wire)
	seen := map[string]bool{}
	for {
		var response runtimeWireResponse
		if err := decoder.Decode(&response); err != nil {
			if err != io.EOF {
				t.Fatal(err)
			}
			break
		}
		seen[response.Op] = true
		if response.Op == "run-done" && (response.Error != "" || response.ExitCode != 0) {
			t.Fatalf("unexpected run response: %+v", response)
		}
	}
	for _, op := range []string{"stdout", "stderr", "run-done"} {
		if !seen[op] {
			t.Fatalf("missing %s frame: %v", op, seen)
		}
	}
	if _, err := os.Stat(filepath.Join(bundle, "config.json")); err != nil {
		t.Fatalf("OCI config was not written: %v", err)
	}
}

func TestRuntimeHelperOpenValidationFailsBeforeMounting(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	runRoot := filepath.Join(root, "run")
	rootfs := filepath.Join(root, "rootfs")
	for _, dir := range []string{workspace, runRoot, rootfs, filepath.Join(root, "in"), filepath.Join(root, "out")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	base := runtimeWireOpen{RootFS: rootfs, Workspace: workspace, In: filepath.Join(root, "in"),
		Out: filepath.Join(root, "out"), RunRoot: runRoot, WorkspaceTarget: "/work",
		UserName: "enode", UID: 1000, GID: 1000, SubUIDSize: 65536, SubGIDSize: 65536}
	h := &overlayRuntimeHelper{}
	unsafe := base
	unsafe.Workspace += ",bad"
	if err := h.openSession(unsafe); err == nil || !strings.Contains(err.Error(), "unsafe") {
		t.Fatalf("unsafe overlay path error=%v", err)
	}
	badID := base
	badID.UID = badID.SubUIDSize
	if err := h.openSession(badID); err == nil || !strings.Contains(err.Error(), "represented") {
		t.Fatalf("unmappable uid error=%v", err)
	}
	overlap := base
	overlap.RunRoot = filepath.Join(workspace, "scratch")
	if err := os.MkdirAll(overlap.RunRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := h.openSession(overlap); err == nil || !strings.Contains(err.Error(), "must not contain") {
		t.Fatalf("overlapping paths error=%v", err)
	}
	if err := h.openSession(base); err == nil || !strings.Contains(err.Error(), "target /work is missing") {
		t.Fatalf("missing rootfs target error=%v", err)
	}
}

func TestExportedRuncOverlayConstructorUsesPreparedRootFSLayout(t *testing.T) {
	store := t.TempDir()
	manifest := execenv.Manifest{PreparedEnvironmentID: "sha256:test", Profile: execenv.ManifestProfile{SHA256: "profile"}}
	rootfs := execenv.PreparedRootFS(store, manifest.PreparedEnvironmentID)
	if err := os.MkdirAll(rootfs, 0o755); err != nil {
		t.Fatal(err)
	}
	doc := execenv.Document{SHA256: "profile", Profile: execenv.Profile{Runtime: execenv.RuntimePolicy{Driver: "runc-overlay"}}}
	runtimeImpl, err := NewRuncOverlayRuntime(doc, execenv.Binding{Store: store}, manifest)
	if err != nil || runtimeImpl.rootfs != rootfs {
		t.Fatalf("runtime=%+v err=%v", runtimeImpl, err)
	}
}

func TestExecutionRuntimeVerifierNativeAndInvalidRunc(t *testing.T) {
	verifier := ExecutionRuntimeVerifier{}
	doc := execenv.Document{Profile: execenv.Profile{Runtime: execenv.RuntimePolicy{Driver: "native"}}}
	if err := verifier.Verify(context.Background(), doc, execenv.Binding{}, "", execenv.Manifest{}); err != nil {
		t.Fatal(err)
	}
	doc.Profile.Runtime.Driver = "runc-overlay"
	if err := verifier.Verify(context.Background(), doc, execenv.Binding{}, filepath.Join(t.TempDir(), "missing"), execenv.Manifest{}); err == nil {
		t.Fatal("invalid runc verifier input succeeded")
	}
}

func TestRuntimeIDMappingsExposeTheDeveloperAsTheOuterNamespaceOwner(t *testing.T) {
	want := []ociIDMapping{
		{ContainerID: 0, HostID: 1, Size: 1000},
		{ContainerID: 1000, HostID: 0, Size: 1},
		{ContainerID: 1001, HostID: 1001, Size: 64536},
	}
	if got := runtimeIDMappings(1000, 65536); !reflect.DeepEqual(got, want) {
		t.Fatalf("mapping mismatch:\n got %#v\nwant %#v", got, want)
	}
}

func TestOCIConfigKeepsHostPathsOutOfTheProcessContract(t *testing.T) {
	h := &overlayRuntimeHelper{
		open: &runtimeWireOpen{
			RootFS: "/host/rootfs", WorkspaceTarget: "/srv/workspaces/product",
			In: "/host/in", Out: "/host/out", UserName: "enode", UID: 1000, GID: 1000,
			SubUIDSize: 65536, SubGIDSize: 65536, Locale: "en_US.UTF-8", TmpSize: 256 << 20,
		},
		merged: "/host/merged", inRO: "/host/in-ro",
		projection: runtimeWireProjection{Harness: "/host/claude", Enode: "/host/enode",
			Instrumentation: "/host/instrumentation", Credential: "/host/helper"},
	}
	config, err := h.ociConfig(runtimeWireProcess{
		Argv: []string{runtimeHarnessTarget, "--print"},
		Cwd:  "/tmp",
		Env:  []string{"OUT=" + runtimeOutTarget, "PATH=/host/bin", "HOME=/host/home"},
	})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(append(append([]string{}, config.Process.Args...), config.Process.Env...), "\n")
	if strings.Contains(joined, "/host/") {
		t.Fatalf("host path leaked into process args/env:\n%s", joined)
	}
	if config.Process.Cwd != "/tmp" || config.Root.Path != "/host/rootfs" || !config.Root.Readonly {
		t.Fatalf("wrong root/cwd contract: %+v %+v", config.Process, config.Root)
	}
	if _, err := h.ociConfig(runtimeWireProcess{Argv: []string{"true"}, Cwd: "relative"}); err == nil {
		t.Fatal("accepted a relative runtime cwd")
	}
	mounts := map[string]ociMount{}
	for _, mount := range config.Mounts {
		mounts[mount.Destination] = mount
	}
	for _, target := range []string{h.open.WorkspaceTarget, runtimeInTarget, runtimeOutTarget,
		runtimeHarnessTarget, runtimeEnodeTarget, runtimeInstrumentationTarget, runtimeCredentialTarget, "/tmp"} {
		if _, ok := mounts[target]; !ok {
			t.Fatalf("missing typed mount %s: %+v", target, config.Mounts)
		}
	}
	if !hasRuntimeOption(mounts[runtimeInTarget].Options, "ro") || hasRuntimeOption(mounts[h.open.WorkspaceTarget].Options, "ro") {
		t.Fatalf("wrong workspace/input access: %+v %+v", mounts[h.open.WorkspaceTarget], mounts[runtimeInTarget])
	}
}

func hasRuntimeOption(options []string, want string) bool {
	for _, option := range options {
		if option == want {
			return true
		}
	}
	return false
}

func TestOverlayFinalizeReadsTheMergedWorkspaceAndWalksTheUpper(t *testing.T) {
	root := t.TempDir()
	merged, out, upper := filepath.Join(root, "merged"), filepath.Join(root, "out"), filepath.Join(root, "upper")
	for _, dir := range []string{merged, out, upper} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(merged, "artifact"), []byte("sealed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(upper, "written-by-the-step"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	h := &overlayRuntimeHelper{open: &runtimeWireOpen{Out: out}, merged: merged, upper: upper}
	result, err := h.finalize(FinalizeSpec{Collect: map[string]string{"result": "artifact"},
		Discover: true, Stamp: Stamp{Root: "/host/workspace"}, Deadline: time.Now().Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result.Collected, []string{"result"}) {
		t.Fatalf("unexpected finalize: %+v", result)
	}
	b, err := os.ReadFile(filepath.Join(out, "result"))
	if err != nil || string(b) != "sealed" {
		t.Fatalf("collected body=%q err=%v", b, err)
	}
	if d := result.Discovery; d == nil || d.Total != 1 || d.Paths[0].Path != "written-by-the-step" || d.Deleted != 0 {
		t.Fatalf("discover did not walk the upper: %+v", result.Discovery)
	}
}

// 문자 장치가 아닌 항목은 whiteout 이 아니다. 진짜 whiteout(문자 장치 0:0)은 보통 권한으로
// 만들 수 없어 integration 태그 시험이 확인한다.
func TestIsOverlayWhiteoutIgnoresRegularFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	if isOverlayWhiteout(entries[0]) {
		t.Fatal("a regular file was taken for a whiteout")
	}
	devs, err := os.ReadDir("/dev")
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range devs {
		if d.Name() == "null" && isOverlayWhiteout(d) {
			t.Fatal("/dev/null (1:3) was taken for a whiteout")
		}
	}
}
