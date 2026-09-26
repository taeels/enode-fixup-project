//go:build linux && integration

package enode

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	execenv "github.com/taeels/enode/internal/environment"
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
