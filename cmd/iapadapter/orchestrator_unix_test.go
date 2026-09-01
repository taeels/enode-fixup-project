//go:build !windows

package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// 짝 파일이 아니라 갈래 파일이다 — StartOrchestrator 를 재려면 실행 파일
// 하나를 띄워야 하고, 가짜 실행 파일을 sh 스크립트로 두는 것은 윈도우에서
// 성립하지 않는다. 런타임 GOOS 분기 대신 빌드 태그로 가른다
// (cmd/enode/entrypoint_unix_test.go · cmd/enodectl/lifecycle_unix_test.go 선례).

// fakeEnode 는 --ready-file 에 node_id 를 적는 가짜 오케스트레이터다.
// 받은 인자를 <ready>.args 에 먼저 적으므로, ready 가 보이면 인자도 이미 있다.
func fakeEnode(t *testing.T, body string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "fake-enode")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$5.args\"\n" + body
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin
}

func orchConfig(t *testing.T, bin string, readySeconds int, keep bool) *Config {
	t.Helper()
	return &Config{
		Mediator: MediatorConfig{URL: "https://mediator.test", Token: "fleet-token"},
		Orchestrator: OrchestratorConfig{
			Bin:           bin,
			WorkspaceRoot: t.TempDir(),
			ReadySeconds:  readySeconds,
			Keep:          keep,
		},
	}
}

// 이름표가 하나 붙는 것이 전부다 — Labels 는 detect 가 그대로 광고에 싣고,
// 계약의 requires 가 그 이름표로 자기 것만 잡는다. 이것이 I1 이 안 부딪히는
// 이유다 (ADR-015): 이슈마다 설정 경로가 달라 node_id 도 다르다.
func TestStartOrchestrator_WritesAConfigThatCarriesTheIssueLabel(t *testing.T) {
	bin := fakeEnode(t, "printf 'node-abc\\n' > \"$5\"\nsleep 5\n")
	cfg := orchConfig(t, bin, 10, true)

	o, err := StartOrchestrator(context.Background(), cfg, "EP-2", quietLog())
	if err != nil {
		t.Fatal(err)
	}
	defer o.Stop()

	if o.NodeID != "node-abc" {
		t.Fatalf("node id = %q, want the ready file's content with its newline stripped", o.NodeID)
	}
	if o.IssueKey != "EP-2" {
		t.Fatalf("issue key = %q", o.IssueKey)
	}
	if filepath.Base(o.Dir) != "orch-EP-2" {
		t.Fatalf("dir = %q, want one workspace per issue", o.Dir)
	}
	// 파일 이름이 곧 이름표가 된다 — 노드 라벨이 설정 파일 이름에서 유도되므로
	// config.yaml 로 두면 함대에 taeels@호스트:config 로 뜬다.
	if filepath.Base(o.ConfigPan) != "orch-EP-2.yaml" {
		t.Fatalf("config file = %q, want the issue in its name", o.ConfigPan)
	}

	raw, err := os.ReadFile(o.ConfigPan)
	if err != nil {
		t.Fatal(err)
	}
	var conf map[string]any
	if err := yaml.Unmarshal(raw, &conf); err != nil {
		t.Fatal(err)
	}
	if conf["mediator"] != "https://mediator.test" || conf["token"] != "fleet-token" {
		t.Fatalf("the orchestrator cannot reach the fleet: %v", conf)
	}
	if conf["workspace"] != o.Dir {
		t.Fatalf("workspace = %v, want %q", conf["workspace"], o.Dir)
	}
	// ADR-017 이 워크스페이스를 노드로 정했으므로 경로가 있어야 node_id 가 선다.
	if conf["workspace_id"] != "itsaplan-orch-EP-2" {
		t.Fatalf("workspace_id = %v", conf["workspace_id"])
	}
	if conf["orchestration"] != true {
		t.Fatalf("orchestration = %v, want true - it would advertise the wrong capability", conf["orchestration"])
	}
	labels, ok := conf["labels"].(map[string]any)
	if !ok || labels["issue"] != "EP-2" {
		t.Fatalf("labels = %v, want issue=EP-2 - another issue's contract would match this node", conf["labels"])
	}

	// --once 가 탄력 노드를 닫는다 (ADR-012) — 그 인자가 실제로 실려야 한다.
	args, err := os.ReadFile(filepath.Join(o.Dir, "ready.args"))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--config", o.ConfigPan, "--once", "--ready-file", filepath.Join(o.Dir, "ready")}
	got := strings.Split(strings.TrimRight(string(args), "\n"), "\n")
	if len(got) != len(want) {
		t.Fatalf("argv = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("argv[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// 앞 세대가 남긴 ready 파일을 지우고 시작한다 — 안 지우면 이번 세대가
// 뜨기도 전에 지난 세대의 node_id 를 자기 것으로 읽는다.
func TestStartOrchestrator_IgnoresAStaleReadyFile(t *testing.T) {
	bin := fakeEnode(t, "sleep 0.6\nprintf 'node-new\\n' > \"$5\"\nsleep 5\n")
	cfg := orchConfig(t, bin, 10, true)

	dir := filepath.Join(cfg.Orchestrator.WorkspaceRoot, "orch-EP-2")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ready"), []byte("node-stale\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	o, err := StartOrchestrator(context.Background(), cfg, "EP-2", quietLog())
	if err != nil {
		t.Fatal(err)
	}
	defer o.Stop()

	if o.NodeID == "node-stale" {
		t.Fatal("a previous generation's node id was adopted as this one's")
	}
	if o.NodeID != "node-new" {
		t.Fatalf("node id = %q, want node-new", o.NodeID)
	}
}

// 광고를 못 하고 상한을 넘기면 띄운 것을 거두고 실패로 닫는다 — 그리고
// 어디를 보면 되는지(로그 경로)를 오류에 적는다.
func TestStartOrchestrator_GivesUpAndReapsWhenItNeverAdvertises(t *testing.T) {
	bin := fakeEnode(t, "sleep 30\n")
	cfg := orchConfig(t, bin, 0, false)

	start := time.Now()
	o, err := StartOrchestrator(context.Background(), cfg, "EP-2", quietLog())
	if err == nil {
		o.Stop()
		t.Fatal("an orchestrator that never advertised was accepted")
	}
	if o != nil {
		t.Fatalf("a failed start still returned an orchestrator: %+v", o)
	}
	if !strings.Contains(err.Error(), "did not advertise") {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(err.Error(), "orch.log") {
		t.Fatalf("error %q does not say where to look", err.Error())
	}
	// Stop 이 살아 있는 자식을 죽이고 거둔다 — 안 그러면 sleep 30 이 남는다.
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Fatalf("the failed start took %s - the live child was not killed", elapsed)
	}
	if _, err := os.Stat(filepath.Join(cfg.Orchestrator.WorkspaceRoot, "orch-EP-2")); err == nil {
		t.Fatal("the workspace of a failed start was left behind")
	}
}

func TestStartOrchestrator_StopsWhenTheAdapterIsGoingDown(t *testing.T) {
	bin := fakeEnode(t, "sleep 30\n")
	cfg := orchConfig(t, bin, 30, false)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	o, err := StartOrchestrator(ctx, cfg, "EP-2", quietLog())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want the cancellation", err)
	}
	if o != nil {
		t.Fatalf("a cancelled start still returned an orchestrator: %+v", o)
	}
}

// 띄우기 전에 실패하는 자리들 — 어느 것도 조용히 지나가면 안 된다.
// 지나가면 어댑터가 오케스트레이터 없이 Run 을 내고 422 로 떨어진다.
func TestStartOrchestrator_RefusesWhatItCannotSetUp(t *testing.T) {
	t.Run("the workspace root is not a directory", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "a-file")
		if err := os.WriteFile(root, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg := orchConfig(t, fakeEnode(t, "exit 0\n"), 5, false)
		cfg.Orchestrator.WorkspaceRoot = root
		if _, err := StartOrchestrator(context.Background(), cfg, "EP-2", quietLog()); err == nil {
			t.Fatal("an unusable workspace root was accepted")
		}
	})

	t.Run("the config path is taken by a directory", func(t *testing.T) {
		cfg := orchConfig(t, fakeEnode(t, "exit 0\n"), 5, false)
		blocked := filepath.Join(cfg.Orchestrator.WorkspaceRoot, "orch-EP-2", "orch-EP-2.yaml")
		if err := os.MkdirAll(blocked, 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := StartOrchestrator(context.Background(), cfg, "EP-2", quietLog()); err == nil {
			t.Fatal("a config that could not be written was accepted")
		}
	})

	t.Run("the log path is taken by a directory", func(t *testing.T) {
		cfg := orchConfig(t, fakeEnode(t, "exit 0\n"), 5, false)
		blocked := filepath.Join(cfg.Orchestrator.WorkspaceRoot, "orch-EP-2", "orch.log")
		if err := os.MkdirAll(blocked, 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := StartOrchestrator(context.Background(), cfg, "EP-2", quietLog()); err == nil {
			t.Fatal("a log file that could not be created was accepted")
		}
	})

	t.Run("the orchestrator binary does not exist", func(t *testing.T) {
		cfg := orchConfig(t, filepath.Join(t.TempDir(), "absent-enode"), 5, false)
		if _, err := StartOrchestrator(context.Background(), cfg, "EP-2", quietLog()); err == nil {
			t.Fatal("a missing orchestrator binary was accepted")
		}
	})
}

// --once 라 보통은 스스로 죽는다 — Stop 은 그러지 못한 경우의 뒷정리다.
func TestOrchestrator_StopKillsAChildThatWillNotLeave(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "orch-EP-2")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("/bin/sh", "-c", "sleep 30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	o := &Orchestrator{IssueKey: "EP-2", Dir: dir, cmd: cmd, log: quietLog(), done: make(chan struct{})}
	go func() { _ = cmd.Wait(); close(o.done) }()

	done := make(chan struct{})
	go func() { o.Stop(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Stop did not kill a child that outlives its Run")
	}
	if _, err := os.Stat(dir); err == nil {
		t.Fatal("the workspace was left behind")
	}
}
