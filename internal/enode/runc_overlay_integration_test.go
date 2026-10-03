//go:build linux && integration

package enode

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/taeels/enode/internal/contract"
	execenv "github.com/taeels/enode/internal/environment"
	"github.com/taeels/enode/internal/lower"
	"github.com/taeels/enode/internal/scratch"
)

func TestRuncOverlayRuntimeIntegration(t *testing.T) {
	rootfs := os.Getenv("ENODE_RUNC_ROOTFS")
	testRoot := os.Getenv("ENODE_RUNC_TEST_ROOT")
	helper := os.Getenv("ENODE_RUNC_HELPER")
	if rootfs == "" || testRoot == "" || helper == "" {
		t.Skip("set ENODE_RUNC_ROOTFS, ENODE_RUNC_TEST_ROOT, and ENODE_RUNC_HELPER for the real namespace gate")
	}
	// rootfs 의 사용자 이름 — 준비된 rootfs 마다 다르다 (profile 의 rootfs.user.name)
	user := os.Getenv("ENODE_RUNC_USER")
	if user == "" {
		user = "sunny"
	}
	subIDSize := 65536
	if value := os.Getenv("ENODE_RUNC_SUBID_SIZE"); value != "" {
		var err error
		subIDSize, err = strconv.Atoi(value)
		if err != nil || subIDSize <= 1000 {
			t.Fatalf("invalid ENODE_RUNC_SUBID_SIZE %q", value)
		}
	}
	doc, err := execenv.Parse([]byte(fmt.Sprintf(`api_version: enode.dev/v1alpha1
kind: execution-environment
name: runc-integration
host:
  provider: apt
  packages: [debootstrap, runc, uidmap, util-linux]
  require: {subuid_size: %d, subgid_size: %d, unprivileged_userns: true}
rootfs:
  builder: debootstrap
  release: noble
  arch: amd64
  apt: {components: [main], packages: [bash]}
  locale: C.UTF-8
  user: {name: %s, uid: 1000, gid: 1000}
runtime:
  driver: runc-overlay
  workspace_target: /work
  tmp: {size: 64MiB, executable: true}
  credentials: {ssh: readonly}
verify: {executables: [bash], locale: C.UTF-8}
`, subIDSize, subIDSize, user)))
	if err != nil {
		t.Fatal(err)
	}
	binding := execenv.Binding{
		Scratch: filepath.Join(testRoot, "scratch"), Workspace: filepath.Join(testRoot, "workspace"),
		Store: filepath.Join(testRoot, "store"), SSHDir: filepath.Join(testRoot, "ssh"),
	}
	instrumentation := filepath.Join(testRoot, "instrumentation")
	for _, dir := range []string{binding.Scratch, binding.Workspace, binding.SSHDir, instrumentation} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(binding.SSHDir, "config"), []byte("gate\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// 단계가 지우는 lower 의 파일 — upper 에 whiteout(문자 장치 0:0)으로 남는다
	if err := os.WriteFile(filepath.Join(binding.Workspace, "gate-lower"), []byte("lower\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := execenv.Manifest{Profile: execenv.ManifestProfile{Name: doc.Profile.Name, SHA256: doc.SHA256}}
	runtimeImpl, err := newRuncOverlayRuntime(doc, binding, manifest, rootfs)
	if err != nil {
		t.Fatal(err)
	}
	runtimeImpl.helper = helper
	in, out := filepath.Join(testRoot, "in"), filepath.Join(testRoot, "out")
	for _, dir := range []string{in, out} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	probeInput := filepath.Join(in, "probe")
	_ = os.Remove(probeInput)
	if err := os.WriteFile(probeInput, []byte("input\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	session, err := runtimeImpl.Open(context.Background(), RuntimeSpec{Dir: binding.Workspace, In: in, Out: out})
	if err != nil {
		t.Fatal(err)
	}
	concrete := session.(*runcOverlaySession)
	defer session.Close(context.Background(), Keep{}) //nolint:errcheck
	projection, err := session.Project(context.Background(), FrameworkProjectionSpec{
		HarnessExecutable: helper, EnodeExecutable: helper,
		Instrumentation: instrumentation, CredentialHelper: helper,
	})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code, runErr := session.Run(context.Background(), ProcessSpec{
		Argv: []string{"/bin/sh", "-c", `set -eu
test "$(id -u)" = 1000
test "$(id -g)" = 1000
test "$PWD" = /work
test "$(cat "$IN/probe")" = input
! (printf bad > "$IN/probe") 2>/dev/null
test "$(cat "$HOME/.ssh/config")" = gate
! (printf bad > "$HOME/.ssh/config") 2>/dev/null
"$HARNESS" --version >/dev/null
test -x "$CREDENTIAL"
printf instrument > "$INSTRUMENTATION/probe"
printf workspace > .gate-marker
rm gate-lower
printf output > "$OUT/probe"
printf '#!/bin/sh\nexit 0\n' > /tmp/gate-exec
chmod +x /tmp/gate-exec
/tmp/gate-exec
`},
		Env: []string{
			"IN=" + runtimeInTarget, "OUT=" + runtimeOutTarget,
			"HARNESS=" + projection.HarnessExecutable,
			"CREDENTIAL=" + projection.CredentialHelper,
			"INSTRUMENTATION=" + projection.Instrumentation,
		},
		Dir: "/work", Stdout: &stdout, Stderr: &stderr,
	})
	if runErr != nil || code != 0 {
		_ = session.Close(context.Background(), Keep{})
		t.Fatalf("run code=%d err=%v stdout=%q stderr=%q", code, runErr, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(binding.Workspace, ".gate-marker")); !os.IsNotExist(err) {
		_ = session.Close(context.Background(), Keep{})
		t.Fatalf("overlay write reached the original workspace: %v", err)
	}
	if b, err := os.ReadFile(filepath.Join(out, "probe")); err != nil || string(b) != "output" {
		_ = session.Close(context.Background(), Keep{})
		t.Fatalf("$OUT projection body=%q err=%v", b, err)
	}
	if b, err := os.ReadFile(filepath.Join(instrumentation, "probe")); err != nil || string(b) != "instrument" {
		t.Fatalf("instrumentation projection body=%q err=%v", b, err)
	}
	_ = os.Remove(filepath.Join(out, "gate-artifact"))
	finalized, err := session.Finalize(context.Background(), FinalizeSpec{
		Collect:  map[string]string{"gate-artifact": ".gate-marker"},
		Discover: true, Stamp: Stamp{Root: binding.Workspace}, Deadline: time.Now().Add(time.Minute),
	})
	if err != nil || len(finalized.Collected) != 1 || finalized.Collected[0] != "gate-artifact" {
		t.Fatalf("merged finalize=%+v err=%v", finalized, err)
	}
	// 명시 훑기는 upper 만 걷는다 — 쓴 것은 목록에, 지운 것은 whiteout 으로 센다
	if d := finalized.Discovery; d == nil || d.Deleted != 1 || !discovered(d, ".gate-marker") {
		t.Fatalf("upper discovery=%+v", finalized.Discovery)
	}
	// helper 여유 5초의 측정 (계획 3절 ②) — 마감이 지난 요청에 helper 가 돌아오기까지
	began := time.Now()
	if _, err := session.Finalize(context.Background(), FinalizeSpec{
		Discover: true, Stamp: Stamp{Root: binding.Workspace}, Deadline: time.Now(),
	}); err == nil {
		t.Fatal("a finalize past its deadline returned no error")
	}
	t.Logf("helper answered %v after its deadline (grace %v)", time.Since(began), helperGrace)
	if b, err := os.ReadFile(filepath.Join(out, "gate-artifact")); err != nil || string(b) != "workspace" {
		t.Fatalf("collected artifact body=%q err=%v", b, err)
	}

	cancelCtx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	started := time.Now()
	code, runErr = session.Run(cancelCtx, ProcessSpec{Argv: []string{"/bin/sh", "-c", "sleep 60"}, Dir: "/work"})
	if cancelCtx.Err() == nil || runErr == nil || code == 0 {
		t.Fatalf("canceled run code=%d err=%v context=%v", code, runErr, cancelCtx.Err())
	}
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("canceled runtime took %s", elapsed)
	}
	if err := session.Close(context.Background(), Keep{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(concrete.runRoot); !os.IsNotExist(err) {
		t.Fatalf("runtime session state remains at %s: %v", concrete.runRoot, err)
	}

	// 닫은 작업 폴더는 trash 에 있다 — 권한 000 인 work/work 와 whiteout 째로 (trash 유닛).
	// 노드 uid 로는 못 지우는 모양이다. trash-helper 가 같은 uid 매핑의 namespace 안에서 지운다
	// (business-logic-model.md 7절 · 조각 4 ③).
	trash := runtimeImpl.trash
	entry := filepath.Base(concrete.runRoot)
	var st syscall.Stat_t
	if err := syscall.Lstat(filepath.Join(trash.Dir, entry, "work", "work"), &st); err != nil || st.Mode&0o777 != 0 {
		t.Fatalf("work/work in trash: mode %o err %v", st.Mode&0o777, err)
	}
	if err := syscall.Lstat(filepath.Join(trash.Dir, entry, "upper", "gate-lower"), &st); err != nil ||
		st.Mode&syscall.S_IFMT != syscall.S_IFCHR || st.Rdev != 0 {
		t.Fatalf("the whiteout did not travel to trash: mode %o rdev %d err %v", st.Mode, st.Rdev, err)
	}
	// 소유자를 적는다 — 단계 사용자(컨테이너 uid 1000)는 노드 uid 로 매핑된다 (runtimeIDMappings).
	// 2026-09-26 SunnyVM 에서 upper · work 의 항목은 모두 노드 uid 소유였다
	for _, rel := range []string{"upper", "upper/.gate-marker", "upper/gate-lower", "work", "work/work"} {
		var own syscall.Stat_t
		if err := syscall.Lstat(filepath.Join(trash.Dir, entry, rel), &own); err == nil {
			t.Logf("trash %s: uid %d gid %d mode %o", rel, own.Uid, own.Gid, own.Mode&0o7777)
		}
	}
	// 노드 uid 로는 권한 000 인 work/work 안을 못 읽는다. helper 가 항목 전체를 지운다
	was := trashHelperCommand
	trashHelperCommand = func(dir, name string) ([]string, error) { return trashHelperArgv(helper, dir, name), nil }
	defer func() { trashHelperCommand = was }()
	var measured scratch.Size
	began = time.Now()
	if err := TrashLauncher(trash)(context.Background(), entry, func(s scratch.Size) { measured = s }); err != nil {
		t.Fatalf("trash helper: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(trash.Dir, entry)); !os.IsNotExist(err) {
		t.Fatalf("the trash entry survived the helper: %v", err)
	}
	t.Logf("trash helper removed %d entries (%d bytes) in %v", measured.Entries, measured.Bytes, time.Since(began))

	record := &execenv.Record{
		ProfileSHA256: doc.SHA256, PreparedEnvironment: "integration-rootfs", Runtime: "runc-overlay",
	}
	t.Run("worker command", func(t *testing.T) {
		m := newMediator(t)
		w := newWorker(m)
		w.Local.Workspace = binding.Workspace
		w.Runtime, w.RuntimeRecord = runtimeImpl, record
		holdLease(w)
		step := runStep("/bin/sh", "-c", `printf 'command result\n' > "$OUT/command"`)
		step.Out = []string{"command"}
		w.execute(context.Background(), step)
		result := m.only(t)
		if result.Error != "" || result.ExitCode == nil || *result.ExitCode != 0 {
			t.Fatalf("command result=%+v", result)
		}
		if body, ok := m.blob("command"); !ok || string(body) != "command result\n" {
			t.Fatalf("command artifact=%q present=%v", body, ok)
		}
		if result.Environment == nil || result.Environment.ProfileSHA256 != record.ProfileSHA256 ||
			result.Environment.PreparedEnvironment != record.PreparedEnvironment ||
			result.Environment.Runtime != record.Runtime {
			t.Fatalf("command runtime record=%+v", result.Environment)
		}
	})

	t.Run("worker agent", func(t *testing.T) {
		m := newMediator(t)
		w := newWorker(m)
		w.Local.Workspace = binding.Workspace
		w.Local.HarnessBin = stubHarness(t, `cat >/dev/null
printf 'agent result\n' > "$OUT/plan.json"
printf '{"type":"result","subtype":"success","num_turns":1}\n'
`)
		w.Runtime, w.RuntimeRecord = runtimeImpl, record
		holdLease(w)
		step := agentStep(`{}`)
		step.Out = []string{"plan.json"}
		w.execute(context.Background(), step)
		result := m.only(t)
		if result.Error != "" || result.Harness == nil || result.Harness.Reason != ReasonOK {
			t.Fatalf("agent result=%+v", result)
		}
		if body, ok := m.blob("plan.json"); !ok || string(body) != "agent result\n" {
			t.Fatalf("agent artifact=%q present=%v", body, ok)
		}
		if result.Environment == nil || result.Environment.ProfileSHA256 != record.ProfileSHA256 ||
			result.Environment.PreparedEnvironment != record.PreparedEnvironment ||
			result.Environment.Runtime != record.Runtime {
			t.Fatalf("agent runtime record=%+v", result.Environment)
		}
	})

	left, err := filepath.Glob(filepath.Join(binding.Scratch, "enode-runc-*"))
	if err != nil || len(left) != 0 {
		t.Fatalf("runtime scratch remains after Worker scenes: %v err=%v", left, err)
	}

	// Worker 장면이 trash 에 남긴 것을 배경 삭제자가 비운다
	deleter := &scratch.Deleter{Trash: trash, Launch: TrashLauncher(trash)}
	dctx, dcancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { deleter.Run(dctx); close(done) }()
	deadline := time.Now().Add(time.Minute)
	for u := deleter.Usage(); u.TrashEntries != 0 || u.Deleting || u.MeasuredAt.IsZero(); u = deleter.Usage() {
		if time.Now().After(deadline) {
			dcancel()
			<-done
			t.Fatalf("the deleter did not empty trash: %+v", deleter.Usage())
		}
		time.Sleep(50 * time.Millisecond)
	}
	dcancel()
	<-done
}

func discovered(d *Discovery, path string) bool {
	for _, p := range d.Paths {
		if p.Path == path {
			return true
		}
	}
	return false
}

// TestMain 은 이 시험 바이너리가 runtime-helper 로도 돌게 한다 — Verify 는 os.Executable() 을 helper 로 쓰므로,
// smoke 를 진짜로 돌리려면 시험 바이너리가 그 입구를 알아야 한다 (cmd/enode 의 run 과 같은 갈래).
//
// merge-helper 입구도 안다 — 굽기의 integration 시험이 제품 기본값 (unshare 로 이 바이너리를 다시 실행) 그대로
// 합친다 (bake 유닛).
func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "runtime-helper" {
		os.Exit(RunRuncOverlayHelper(os.Stdin, os.Stdout, os.Stderr))
	}
	if len(os.Args) > 1 && os.Args[1] == "merge-helper" {
		os.Exit(RunMergeHelper(os.Stdin, os.Stdout, os.Stderr))
	}
	// 보존의 측정 helper 가 이 시험 바이너리를 다시 실행한다 (checkpoint 유닛 · MeasureLauncher)
	if len(os.Args) > 1 && os.Args[1] == "trash-helper" {
		os.Exit(RunTrashHelper(os.Args[2:], os.Stdout, os.Stderr))
	}
	os.Exit(m.Run())
}

// 다른 Dir 이 lower 의 배타를 쥔 동안 smoke 는 기다리고, 놓으면 돈다 (lower-state 답 5 · 계획 4절 ⑰). HOME 을 버려도
// 되는 자리로 옮겨 상태 자리도 그 안에 둔다. rootfs 는 읽기 전용으로 쓴다 — /bin/sh 와 id · cat · printf · test ·
// chmod 가 있으면 된다 (busybox 하나로 지은 것도 된다).
func TestRuncOverlaySmokeWaitsForAMerge(t *testing.T) {
	rootfs := os.Getenv("ENODE_RUNC_ROOTFS")
	testRoot := os.Getenv("ENODE_RUNC_TEST_ROOT")
	if rootfs == "" || testRoot == "" {
		t.Skip("set ENODE_RUNC_ROOTFS and ENODE_RUNC_TEST_ROOT for the real namespace gate")
	}
	doc, err := execenv.Parse([]byte(`api_version: enode.dev/v1alpha1
kind: execution-environment
name: runc-smoke-lock
host:
  provider: apt
  packages: [debootstrap, runc, uidmap, util-linux]
  require: {subuid_size: 65536, subgid_size: 65536, unprivileged_userns: true}
rootfs:
  builder: debootstrap
  release: noble
  arch: amd64
  apt: {components: [main], packages: [bash]}
  locale: C.UTF-8
  user: {name: enode, uid: 1000, gid: 1000}
runtime:
  driver: runc-overlay
  workspace_target: /work
  tmp: {size: 16MiB, executable: true}
  credentials: {}
verify: {executables: [sh], locale: C.UTF-8}
`))
	if err != nil {
		t.Fatal(err)
	}
	base, err := os.MkdirTemp(testRoot, "smoke-lock-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(base)
	binding := execenv.Binding{Scratch: filepath.Join(base, "scratch"), Workspace: filepath.Join(base, "workspace"),
		Store: filepath.Join(base, "store")}
	if err := os.MkdirAll(binding.Workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", filepath.Join(base, "home"))
	lowers, err := LowersDir()
	if err != nil {
		t.Fatal(err)
	}
	root, err := lower.ReadRoot(binding.Workspace)
	if err != nil {
		t.Fatal(err)
	}
	merge, err := lower.Open(lowers, root, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	bake, _, err := merge.TryBake()
	if err != nil {
		t.Fatal(err)
	}
	if err := bake.WriteState(lower.State{Phase: lower.PhaseMerging, Owner: &lower.Owner{Run: "R-M", Node: "baker"},
		Since: time.Now()}); err != nil {
		t.Fatal(err)
	}
	ex, err := merge.Exclusive(context.Background(), time.Hour, nil)
	if err != nil {
		t.Fatal(err)
	}
	was := smokePoll
	smokePoll = 50 * time.Millisecond
	defer func() { smokePoll = was }()

	var notice lockedBuffer
	manifest := execenv.Manifest{Profile: execenv.ManifestProfile{Name: doc.Profile.Name, SHA256: doc.SHA256}}
	done := make(chan error, 1)
	go func() {
		done <- ExecutionRuntimeVerifier{Notice: &notice}.Verify(context.Background(), doc, binding, rootfs, manifest)
	}()
	select {
	case err := <-done:
		t.Fatalf("the smoke did not wait for the merge: %v", err)
	case <-time.After(time.Second):
	}
	if !strings.Contains(notice.String(), "env check: waiting for the lower merge to finish (run R-M on node baker, since ") {
		t.Fatalf("notice = %q", notice.String())
	}
	released := time.Now()
	ex.Release()
	bake.Release()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("the smoke after the merge: %v", err)
		}
	case <-time.After(2 * time.Minute):
		t.Fatal("the smoke did not finish")
	}
	t.Logf("the smoke ran %v after the merge let go; notice %q", time.Since(released), notice.String())
	// smoke 는 세션을 닫은 뒤 공유를 놓았다 — 배타가 곧바로 잡힌다
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	again, err := merge.Exclusive(ctx, time.Hour, nil)
	if err != nil {
		t.Fatalf("the smoke kept the lower lock: %v", err)
	}
	again.Release()
}

// itRuntime 은 진짜 runc-overlay 런타임이다 — ENODE_RUNC_ROOTFS · ENODE_RUNC_TEST_ROOT 가 있어야 한다. rootfs 의 사용자는
// ENODE_RUNC_USER (기본 enode · uid 1000). SSH 투영은 없다.
func itRuntime(t *testing.T, name string) (*RuncOverlayRuntime, execenv.Binding, string) {
	t.Helper()
	rootfs, testRoot := os.Getenv("ENODE_RUNC_ROOTFS"), os.Getenv("ENODE_RUNC_TEST_ROOT")
	if rootfs == "" || testRoot == "" {
		t.Skip("set ENODE_RUNC_ROOTFS and ENODE_RUNC_TEST_ROOT for the real namespace gate")
	}
	user := os.Getenv("ENODE_RUNC_USER")
	if user == "" {
		user = "enode"
	}
	doc, err := execenv.Parse([]byte(fmt.Sprintf(`api_version: enode.dev/v1alpha1
kind: execution-environment
name: %s
host:
  provider: apt
  packages: [debootstrap, runc, uidmap, util-linux]
  require: {subuid_size: 65536, subgid_size: 65536, unprivileged_userns: true}
rootfs:
  builder: debootstrap
  release: noble
  arch: amd64
  apt: {components: [main], packages: [bash, git]}
  locale: C.UTF-8
  user: {name: %s, uid: 1000, gid: 1000}
runtime:
  driver: runc-overlay
  workspace_target: /work
  tmp: {size: 64MiB, executable: true}
  credentials: {}
verify: {executables: [sh], locale: C.UTF-8}
`, name, user)))
	if err != nil {
		t.Fatal(err)
	}
	base, err := os.MkdirTemp(testRoot, name+"-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(base) })
	binding := execenv.Binding{Scratch: filepath.Join(base, "scratch"), Workspace: filepath.Join(base, "workspace"),
		Store: filepath.Join(base, "store")}
	for _, dir := range []string{binding.Scratch, binding.Workspace} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	manifest := execenv.Manifest{Profile: execenv.ManifestProfile{Name: doc.Profile.Name, SHA256: doc.SHA256}}
	runtimeImpl, err := newRuncOverlayRuntime(doc, binding, manifest, rootfs)
	if err != nil {
		t.Fatal(err)
	}
	return runtimeImpl, binding, base
}

// 진짜 세션의 단계 사용자가 쓴 파일이 Close(Keep{Upper}) 로 옮겨지고 호스트에서 노드 uid 소유다 — 나머지 작업 폴더는
// trash 로 간다 (bake 유닛 · 계획 4.1 6번).
func TestRuncOverlayCloseKeepsTheUpperIntegration(t *testing.T) {
	runtimeImpl, binding, base := itRuntime(t, "bake-keep")
	in, out := filepath.Join(base, "in"), filepath.Join(base, "out")
	for _, dir := range []string{in, out, filepath.Join(base, "pending")} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	session, err := runtimeImpl.Open(context.Background(), RuntimeSpec{Dir: binding.Workspace, In: in, Out: out})
	if err != nil {
		t.Fatal(err)
	}
	runRoot := session.(*runcOverlaySession).runRoot
	var stderr bytes.Buffer
	code, err := session.Run(context.Background(), ProcessSpec{Argv: []string{"/bin/sh", "-c",
		"mkdir -p built && printf kept > built/out.bin"}, Dir: "/work", Stderr: &stderr})
	if err != nil || code != 0 {
		_ = session.Close(context.Background(), Keep{})
		t.Fatalf("run = %d %v %s", code, err, stderr.String())
	}
	keep := filepath.Join(base, "pending", "upper")
	if err := session.Close(context.Background(), Keep{Upper: keep}); err != nil {
		t.Fatal(err)
	}
	var st syscall.Stat_t
	if err := syscall.Lstat(filepath.Join(keep, "built", "out.bin"), &st); err != nil || int(st.Uid) != os.Getuid() {
		t.Fatalf("the kept file: uid %d err %v (node uid %d)", st.Uid, err, os.Getuid())
	}
	if _, err := os.Stat(filepath.Join(binding.Workspace, "built")); !os.IsNotExist(err) {
		t.Fatalf("the step wrote the lower: %v", err)
	}
	moved := filepath.Join(runtimeImpl.trash.Dir, filepath.Base(runRoot))
	if _, err := os.Stat(filepath.Join(moved, "work")); err != nil {
		t.Fatalf("the rest of the session is not in trash: %v", err)
	}
	if _, err := os.Stat(filepath.Join(moved, "upper")); !os.IsNotExist(err) {
		t.Fatalf("the upper also went to trash: %v", err)
	}
}

// 격리 노드의 diff 는 세션 안의 git 이 만든다 — edit 단계가 git 워크스페이스의 추적 파일을 고치고 .git/config 에
// core.fsmonitor 로 호스트 절대 경로 (컨테이너에는 없다) 의 표지 스크립트를 적어도, workspace.diff 에 그 고친 줄이 있고
// 호스트 표지가 없다 (bake 유닛 · 계획 4.1 34번). rootfs 에 git 이 있어야 한다.
func TestRuncOverlayFinalizeDiffInTheSessionIntegration(t *testing.T) {
	runtimeImpl, binding, base := itRuntime(t, "bake-diff")
	for k, v := range map[string]string{"GIT_CONFIG_GLOBAL": "/dev/null", "GIT_CONFIG_NOSYSTEM": "1",
		"GIT_AUTHOR_NAME": "t", "GIT_AUTHOR_EMAIL": "t@e", "GIT_COMMITTER_NAME": "t", "GIT_COMMITTER_EMAIL": "t@e"} {
		t.Setenv(k, v)
	}
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"config", "commit.gpgsign", "false"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = binding.Workspace
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}
	if err := os.WriteFile(filepath.Join(binding.Workspace, "tracked.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "tracked.txt"}, {"commit", "-q", "-m", "one"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = binding.Workspace
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}
	marker := filepath.Join(base, "host-marker")
	hook := filepath.Join(base, "hook")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\necho ran >> "+marker+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	in, out := filepath.Join(base, "in"), filepath.Join(base, "out")
	for _, dir := range []string{in, out} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	session, err := runtimeImpl.Open(context.Background(), RuntimeSpec{Dir: binding.Workspace, In: in, Out: out})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close(context.Background(), Keep{}) //nolint:errcheck
	var stderr bytes.Buffer
	code, err := session.Run(context.Background(), ProcessSpec{Argv: []string{"/bin/sh", "-c",
		"printf 'two\\n' > tracked.txt && printf '[core]\\n\\tfsmonitor = " + hook + "\\n' >> .git/config"},
		Dir: "/work", Stderr: &stderr})
	if err != nil || code != 0 {
		t.Fatalf("the edit step = %d %v %s", code, err, stderr.String())
	}
	res, err := session.Finalize(context.Background(), FinalizeSpec{Diff: true, Out: out,
		Deadline: time.Now().Add(time.Minute)})
	if err != nil || res.DiffError != "" || res.DiffBytes == 0 {
		t.Fatalf("finalize = %+v %v", res, err)
	}
	b, err := os.ReadFile(filepath.Join(out, diffName))
	if err != nil || !strings.Contains(string(b), "+two") || !strings.Contains(string(b), "-one") {
		t.Fatalf("workspace.diff = %q %v", b, err)
	}
	if got, err := os.ReadFile(marker); !os.IsNotExist(err) {
		t.Fatalf("the host ran the tree's fsmonitor: %q", got)
	}
	t.Logf("workspace.diff (%d bytes):\n%s", res.DiffBytes, b)
}

// 보존 (checkpoint 유닛) — 진짜 세션의 upper 가 spool 의 예약 자리로 옮겨지고, helper 가 재고, 소유자가 안내대로 연다.
// 사람이 SunnyVM 에서 돈다 (CI 밖). 판정의 순수한 규칙은 기본 go test 가 본다.

// itCapture 는 진짜 세션 하나를 열고 명령을 돌린 뒤 Keeper 의 판정 · Close · 확정을 지난다. 돌려주는 것은 receipt 의
// 칸 · 그 spool · 걸린 시간 (판정의 창 — Decide 부터 closedAt 까지) 이다.
func itCapture(t *testing.T, name, script string, lowerFiles map[string]string) (*contract.CheckpointCapture, *scratch.Store, time.Duration) {
	t.Helper()
	runtimeImpl, binding, base := itRuntime(t, name)
	for file, body := range lowerFiles {
		if err := os.WriteFile(filepath.Join(binding.Workspace, file), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	in, out := filepath.Join(base, "in"), filepath.Join(base, "out")
	for _, dir := range []string{in, out} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	st := &scratch.Store{Dir: scratch.SpoolIn(binding.Scratch), Scratch: binding.Scratch,
		Trash: scratch.TrashIn(binding.Scratch), Measure: MeasureLauncher(),
		Policy: scratch.Policy{Mode: scratch.ModeOnFailure, TTL: time.Hour, CapacityPercent: 20, MaxBytes: 32 << 30}}
	if _, err := st.Open(); err != nil {
		t.Fatal(err)
	}
	k := &CheckpointKeeper{Policy: st.Policy, Store: st, Support: runtimeImpl.Capability().Capture, Node: "n-it",
		Runtime: "runc-overlay", Workspace: binding.Workspace}
	step := &Step{RunID: "r-it", Seq: 1, Name: "build"}
	slot := k.Reserve(step)
	session, err := runtimeImpl.Open(context.Background(), RuntimeSpec{Dir: binding.Workspace, In: in, Out: out})
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	code, err := session.Run(context.Background(), ProcessSpec{Argv: []string{"/bin/sh", "-c", script}, Dir: "/work",
		Stderr: &stderr})
	if err != nil {
		_ = session.Close(context.Background(), Keep{})
		t.Fatalf("run = %d %v %s", code, err, stderr.String())
	}
	began := time.Now()
	plan := k.Decide(closeFacts{step: step, failed: true, deadline: time.Now().Add(time.Minute), slot: slot})
	if err := session.Close(context.Background(), plan.Keep(Keep{})); err != nil {
		t.Fatal(err)
	}
	window := time.Since(began)
	c, detail := k.Finish(plan)
	if c == nil || c.State != contract.CaptureCaptured {
		t.Fatalf("capture = %+v %q", c, detail)
	}
	return c, st, window
}

// 단계가 쓴 파일과 lower 를 지운 whiteout 이 spool 에 그대로 있다. 작업 폴더의 나머지는 trash 로 갔다.
func TestIntegrationCheckpoint_CapturesSubuidAndWhiteouts(t *testing.T) {
	c, st, _ := itCapture(t, "checkpoint-capture", "mkdir -p built && printf kept > built/out.bin && rm gone.txt",
		map[string]string{"gone.txt": "lower"})
	upper := filepath.Join(st.Dir, c.ID, "upper")
	if b, err := os.ReadFile(filepath.Join(upper, "built", "out.bin")); err != nil || string(b) != "kept" {
		t.Fatalf("the written file: %q %v", b, err)
	}
	var w syscall.Stat_t
	if err := syscall.Lstat(filepath.Join(upper, "gone.txt"), &w); err != nil || w.Mode&syscall.S_IFMT != syscall.S_IFCHR || w.Rdev != 0 {
		t.Fatalf("the whiteout did not survive the move: mode %o rdev %d err %v", w.Mode, w.Rdev, err)
	}
	l, err := st.Lookup(c.ID)
	if err != nil || l == nil || l.State != scratch.EntryKept || l.Lower == "" {
		t.Fatalf("record = %+v %v", l, err)
	}
}

// helper 가 spool 의 항목을 namespace 안에서 측정한다 — 판정이 그 값을 기록에 적는다.
func TestIntegrationCheckpoint_MeasureInHelper(t *testing.T) {
	c, st, _ := itCapture(t, "checkpoint-measure", "mkdir -p a/b && head -c 65536 /dev/zero > a/b/z", nil)
	size, err := MeasureLauncher()(context.Background(), filepath.Join(st.Dir, c.ID), "upper")
	if err != nil || size.Entries < 3 || size.Bytes < 65536 {
		t.Fatalf("measure = %+v %v", size, err)
	}
	res, err := st.Settle(context.Background())
	if err != nil || res.Usage.Checkpoints != 1 || res.Usage.Bytes != size.Bytes || res.Usage.Unsized != 0 {
		t.Fatalf("settle = %+v %v (helper said %+v)", res.Usage, err, size)
	}
}

// NFR C4 — show 의 안내 줄 (unshare --user --map-root-user --map-auto) 로 subordinate uid 소유의 0600 파일이 읽힌다.
// 노드 uid 로는 못 읽는다.
func TestIntegrationCheckpoint_UnshareReadsRootFiles(t *testing.T) {
	c, st, _ := itCapture(t, "checkpoint-open", "printf mine > mine.txt", nil)
	upper := filepath.Join(st.Dir, c.ID, "upper")
	ns := []string{"unshare", "--user", "--map-root-user", "--map-auto"}
	mk := exec.Command(ns[0], append(ns[1:], "sh", "-c",
		"printf secret > root.txt && chown 1 root.txt && chmod 600 root.txt")...)
	mk.Dir = upper
	if out, err := mk.CombinedOutput(); err != nil {
		t.Fatalf("cannot make a subordinate-uid file: %v %s", err, out)
	}
	if _, err := os.ReadFile(filepath.Join(upper, "root.txt")); err == nil {
		t.Fatal("the node uid read a 0600 file of a subordinate uid; the open line would be needless")
	}
	read := exec.Command(ns[0], append(ns[1:], "cat", filepath.Join(upper, "root.txt"))...)
	if out, err := read.Output(); err != nil || string(out) != "secret" {
		t.Fatalf("the open line did not read it: %q %v", out, err)
	}
}

// NFR P1 · P3 — 보존이 판정의 창에 더하는 일은 upper 크기와 무관하다. 큰 upper 와 빈 upper 의 창을 적는다.
func TestIntegrationCheckpoint_WindowIndependentOfSize(t *testing.T) {
	_, _, small := itCapture(t, "checkpoint-small", "true", nil)
	_, _, big := itCapture(t, "checkpoint-big", "mkdir -p many && cd many && i=0; while [ $i -lt 20000 ]; do : > f$i; i=$((i+1)); done", nil)
	t.Logf("window: empty upper %v · 20000-file upper %v", small, big)
	if big > small+2*time.Second {
		t.Fatalf("the window grew with the upper: %v vs %v", big, small)
	}
}
