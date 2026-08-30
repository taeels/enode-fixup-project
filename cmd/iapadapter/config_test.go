package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// 최소 설정 — 이 다섯 값이 없으면 어댑터가 아무것도 못 한다.
const minimalConfig = `
itsaplan:
  base_url: https://iap.test/api/
  api_key: agent-key
mediator:
  url: https://mediator.test
  token: fleet-token
orchestrator:
  bin: /usr/local/bin/enode
`

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "adapter.yaml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// 안 적은 것에는 기본값이 선다 — 0 이 그대로 남으면 poll 이 바쁜 고리가 되고
// 하트비트가 안 돌아 리스가 만료된다 (integration §10.5.1 ④).
func TestLoadConfig_FillsInTheDefaults(t *testing.T) {
	c, err := LoadConfig(writeConfig(t, minimalConfig))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		got  int
		want int
	}{
		{"itsaplan.poll_seconds", c.ItsAPlan.PollSeconds, 5},
		{"itsaplan.heartbeat_seconds", c.ItsAPlan.HeartbeatSeconds, 60},
		{"orchestrator.ready_seconds", c.Orchestrator.ReadySeconds, 90},
		{"mediator.submit_wait_seconds", c.Mediator.SubmitWaitSeconds, 600},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %d, want the default %d", tc.name, tc.got, tc.want)
		}
	}
	if c.Orchestrator.WorkspaceRoot != "/tmp/enode-orch" {
		t.Errorf("workspace_root = %q, want the default", c.Orchestrator.WorkspaceRoot)
	}
	// 계약의 requires 가 이 이름으로 실행 노드를 가리킨다 (ADR-058).
	if c.Executor.As != "worker" {
		t.Errorf("executor.as = %q, want the default worker", c.Executor.As)
	}
	// 하트비트는 리스보다 넉넉히 짧아야 한다 — 실측된 리스가 300초다.
	if c.Heartbeat() >= 300*time.Second {
		t.Errorf("the default heartbeat %s is not shorter than the 300s lease", c.Heartbeat())
	}
}

func TestLoadConfig_KeepsWhatIsWritten(t *testing.T) {
	c, err := LoadConfig(writeConfig(t, `
itsaplan:
  base_url: https://iap.test
  api_key: agent-key
  project_key: EP
  poll_seconds: 2
  heartbeat_seconds: 30
mediator:
  url: https://mediator.test
  token: fleet-token
  principal: taeels
  submit_wait_seconds: 120
orchestrator:
  bin: /opt/enode
  workspace_root: /var/lib/enode
  ready_seconds: 15
  keep_workspace: true
executor:
  as: builder
  attrs:
    harness: claude
    os: linux
transition:
  on_success: Done
  on_failure: Blocked
  on_ask: Waiting
  on_start: In Progress
`))
	if err != nil {
		t.Fatal(err)
	}
	if c.ItsAPlan.ProjectKey != "EP" || c.ItsAPlan.PollSeconds != 2 || c.ItsAPlan.HeartbeatSeconds != 30 {
		t.Errorf("itsaplan = %+v", c.ItsAPlan)
	}
	// principal 이 X-Enode-Principal 로 실려 봉인에 남는다 (ADR-033).
	if c.Mediator.Principal != "taeels" || c.Mediator.SubmitWaitSeconds != 120 {
		t.Errorf("mediator = %+v", c.Mediator)
	}
	if c.Orchestrator.WorkspaceRoot != "/var/lib/enode" || c.Orchestrator.ReadySeconds != 15 || !c.Orchestrator.Keep {
		t.Errorf("orchestrator = %+v", c.Orchestrator)
	}
	if c.Executor.As != "builder" || c.Executor.Attrs["harness"] != "claude" || c.Executor.Attrs["os"] != "linux" {
		t.Errorf("executor = %+v", c.Executor)
	}
	// 칸 이름이 프로젝트마다 다르므로 설정이 정한다 (ADR-040 §2.1).
	want := TransitionConfig{OnSuccess: "Done", OnFailure: "Blocked", OnAsk: "Waiting", OnStart: "In Progress"}
	if c.Transition != want {
		t.Errorf("transition = %+v, want %+v", c.Transition, want)
	}
}

// 빠진 것이 있으면 뜨기 전에 거절한다 — 뜨고 나서 매 왕복마다 401 을 받으면
// 원인이 로그 깊숙이 묻힌다.
func TestLoadConfig_RefusesWhatItCannotRun(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{"no base_url", strings.Replace(minimalConfig, "base_url: https://iap.test/api/", "", 1),
			"itsaplan.base_url"},
		{"no api_key", strings.Replace(minimalConfig, "api_key: agent-key", "", 1),
			"itsaplan.api_key"},
		{"no mediator url", strings.Replace(minimalConfig, "url: https://mediator.test", "", 1),
			"mediator.url"},
		{"no mediator token", strings.Replace(minimalConfig, "token: fleet-token", "", 1),
			"mediator.token"},
		{"no orchestrator bin", strings.Replace(minimalConfig, "bin: /usr/local/bin/enode", "", 1),
			"orchestrator.bin"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadConfig(writeConfig(t, tc.body))
			if err == nil {
				t.Fatal("an unusable config was accepted")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not name %q", err.Error(), tc.want)
			}
		})
	}
}

func TestLoadConfig_RefusesAFileItCannotRead(t *testing.T) {
	if _, err := LoadConfig(filepath.Join(t.TempDir(), "absent.yaml")); err == nil {
		t.Fatal("a missing config file was accepted")
	}
	if _, err := LoadConfig(writeConfig(t, "itsaplan: [this is not a mapping")); err == nil {
		t.Fatal("malformed YAML was accepted")
	}
}

// 설정은 초로 적고 코드는 Duration 으로 쓴다 — 그 변환이 한 자리에 있다.
func TestConfig_DurationsComeFromSeconds(t *testing.T) {
	c := &Config{
		ItsAPlan: ItsAPlanConfig{PollSeconds: 3, HeartbeatSeconds: 45},
		Mediator: MediatorConfig{SubmitWaitSeconds: 90},
		Orchestrator: OrchestratorConfig{
			ReadySeconds: 12,
		},
	}
	for _, tc := range []struct {
		name string
		got  time.Duration
		want time.Duration
	}{
		{"Poll", c.Poll(), 3 * time.Second},
		{"Heartbeat", c.Heartbeat(), 45 * time.Second},
		{"ReadyWait", c.ReadyWait(), 12 * time.Second},
		{"SubmitWait", c.SubmitWait(), 90 * time.Second},
	} {
		if tc.got != tc.want {
			t.Errorf("%s() = %s, want %s", tc.name, tc.got, tc.want)
		}
	}
}
