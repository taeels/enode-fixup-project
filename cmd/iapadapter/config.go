package main

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 는 어댑터가 아는 전부다.
//
// 바깥을 아는 부품은 어댑터 하나뿐이다 (ADR-040 §2) — 그래서 이슈 트래커의
// 주소도, 칸 이름도, 오케스트레이터를 어떻게 띄우는지도 전부 여기 있다.
// Mediator 는 It's a Plan 을 모르고 계약도 칸 이름을 모른다.
type Config struct {
	ItsAPlan     ItsAPlanConfig     `yaml:"itsaplan"`
	Mediator     MediatorConfig     `yaml:"mediator"`
	Orchestrator OrchestratorConfig `yaml:"orchestrator"`
	Executor     ExecutorConfig     `yaml:"executor"`
	Transition   TransitionConfig   `yaml:"transition"`
}

type ItsAPlanConfig struct {
	BaseURL    string `yaml:"base_url"`
	APIKey     string `yaml:"api_key"`
	ProjectKey string `yaml:"project_key"`
	// PollSeconds 는 claim 을 다시 거는 간격이다. 큐가 비면 그만큼 쉰다.
	PollSeconds int `yaml:"poll_seconds"`
	// HeartbeatSeconds 는 리스를 늘리는 간격이다. 실측된 리스는 300초이므로
	// 그보다 넉넉히 짧아야 한다 (integration §10.5.1 ④).
	HeartbeatSeconds int `yaml:"heartbeat_seconds"`
}

type MediatorConfig struct {
	URL   string `yaml:"url"`
	Token string `yaml:"token"`
	// Principal 은 X-Enode-Principal 로 실린다. 되묻기의 답을 쓸 때
	// 누가 답했는지가 봉인에 남는다 (ADR-033).
	Principal string `yaml:"principal"`
	// SubmitWaitSeconds 는 「지금은 전부 점유됨」(409) 을 만났을 때 다시 내며
	// 기다리는 상한이다.
	//
	// 동시성의 상한은 어댑터가 아니라 함대다 (I1 — 노드 하나는 동시에 하나의
	// Run 에만). 그래서 실행 노드가 다른 이슈를 물고 있는 것은 고장이 아니라
	// 정상이고, 그때마다 이슈를 실패로 닫으면 안 된다.
	// 422 는 이 기다림에 안 걸린다 — 다시 내도 같기 때문이다.
	SubmitWaitSeconds int `yaml:"submit_wait_seconds"`
}

type OrchestratorConfig struct {
	// Bin 은 enode 실행 파일이다. 어댑터가 이슈마다 하나 띄운다.
	Bin string `yaml:"bin"`
	// WorkspaceRoot 아래에 이슈마다 디렉터리가 생긴다. ADR-017 이
	// 워크스페이스를 노드로 정했으므로 경로가 있어야 node_id 가 선다.
	WorkspaceRoot string `yaml:"workspace_root"`
	// ReadySeconds 는 띄운 오케스트레이터가 함대에 보일 때까지 기다리는 상한이다.
	// 이 기다림이 어댑터에 있는 것은 오늘의 타협이다 —
	// adapter-example §4.1 이 "조율이 밖으로 샌다" 로 지적한 자리.
	ReadySeconds int `yaml:"ready_seconds"`
	// Keep 이 참이면 Run 이 끝나도 워크스페이스를 안 지운다 (조사용).
	Keep bool `yaml:"keep_workspace"`
}

// ExecutorConfig 는 계획이 실제 일을 시킬 노드를 어떻게 고르는가다.
//
// 계약이 이슈 트래커를 모르는 것과 같은 이유로, 이슈도 노드를 모른다 —
// 어느 노드에서 도는지는 함대의 사정이고 그것을 아는 것은 어댑터 설정이다.
type ExecutorConfig struct {
	// As 는 계약 안에서 이 역할의 이름이다. 계획이 uses 로 가리킨다.
	As string `yaml:"as"`
	// Attrs 는 requires 에 그대로 실리는 매칭 속성이다 (harness · os · ws …).
	Attrs map[string]string `yaml:"attrs"`
}

// TransitionConfig 는 보드가 읽는 것이다 (ADR-040 §2.1).
// 칸 이름이 프로젝트마다 다르므로 설정이 정하고, 빈 값이면 안 옮긴다.
type TransitionConfig struct {
	OnSuccess string `yaml:"on_success"`
	OnFailure string `yaml:"on_failure"`
	OnAsk     string `yaml:"on_ask"`
	OnStart   string `yaml:"on_start"`
}

func LoadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	if c.ItsAPlan.BaseURL == "" || c.ItsAPlan.APIKey == "" {
		return nil, fmt.Errorf("itsaplan.base_url and itsaplan.api_key are required")
	}
	if c.Mediator.URL == "" || c.Mediator.Token == "" {
		return nil, fmt.Errorf("mediator.url and mediator.token are required")
	}
	if c.Orchestrator.Bin == "" {
		return nil, fmt.Errorf("orchestrator.bin is required")
	}
	if c.Orchestrator.WorkspaceRoot == "" {
		c.Orchestrator.WorkspaceRoot = "/tmp/enode-orch"
	}
	if c.ItsAPlan.PollSeconds <= 0 {
		c.ItsAPlan.PollSeconds = 5
	}
	if c.ItsAPlan.HeartbeatSeconds <= 0 {
		c.ItsAPlan.HeartbeatSeconds = 60
	}
	if c.Orchestrator.ReadySeconds <= 0 {
		c.Orchestrator.ReadySeconds = 90
	}
	if c.Mediator.SubmitWaitSeconds <= 0 {
		c.Mediator.SubmitWaitSeconds = 600
	}
	if c.Executor.As == "" {
		c.Executor.As = "worker"
	}
	return &c, nil
}

func (c *Config) Poll() time.Duration { return time.Duration(c.ItsAPlan.PollSeconds) * time.Second }
func (c *Config) Heartbeat() time.Duration {
	return time.Duration(c.ItsAPlan.HeartbeatSeconds) * time.Second
}
func (c *Config) ReadyWait() time.Duration {
	return time.Duration(c.Orchestrator.ReadySeconds) * time.Second
}
func (c *Config) SubmitWait() time.Duration {
	return time.Duration(c.Mediator.SubmitWaitSeconds) * time.Second
}
