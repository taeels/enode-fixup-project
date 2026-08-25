package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Orchestrator 는 이슈 하나를 위해 띄운 오케스트레이션 노드다.
//
// 이것이 새로 그린 자리다 (docs/itsaplan-adapter-example.md §1.2 ③) —
// 어댑터가 Mediator 안에 있지 않고, 밖에서 이슈마다 오케스트레이터를 띄운다.
// 그래서 이슈 둘이 동시에 오면 오케스트레이터도 둘이고, 설정 경로가 다르므로
// node_id 도 다르다 (ADR-015) — I1(노드당 한 Run)이 안 부딪힌다.
type Orchestrator struct {
	IssueKey  string
	Dir       string
	ConfigPan string
	NodeID    string
	cmd       *exec.Cmd
	log       *slog.Logger
	keep      bool
	// done 은 자식이 끝났을 때 닫힌다.
	//
	// 거두는 자리가 하나여야 한다 — Wait 를 부르는 곳이 흩어지면 어떤
	// 경로에서는 아무도 안 부르고, 그러면 자식이 좀비로 남는다.
	// 첫 실측에서 밟았다 (2026-08-24): 되묻기로 손을 뗀 판마다 좀비가 하나씩
	// 쌓였다 — 그 경로는 죽이면 안 되므로 Stop 을 안 불렀고, Wait 가 Stop 에만
	// 있었다. 그래서 시작 직후에 거두는 고루틴을 하나 띄우고 그것만 Wait 한다.
	done chan struct{}
}

// StartOrchestrator 는 설정을 짓고 --once 로 띄운다.
//
// --once 가 탄력 노드를 닫는다 — Run 이 끝나면 스스로 종료하고, 광고가
// 만료되어 함대에서 저절로 빠진다 (ADR-012). 어댑터가 죽여야 할 것이 안 남는다.
func StartOrchestrator(ctx context.Context, cfg *Config, issueKey string, log *slog.Logger) (*Orchestrator, error) {
	dir := filepath.Join(cfg.Orchestrator.WorkspaceRoot, "orch-"+issueKey)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	// 파일 이름이 곧 이름표가 된다 — 노드 라벨이 설정 파일 이름에서
	// 유도되므로 config.yaml 로 두면 함대에 taeels@호스트:config 로 뜬다.
	// 이슈 이름을 넣으면 결과 코멘트의 노드 칸이 사람에게 뜻이 있다.
	confPath := filepath.Join(dir, "orch-"+issueKey+".yaml")
	readyPath := filepath.Join(dir, "ready")
	_ = os.Remove(readyPath)

	// 이름표가 하나 붙는 것이 전부다 — Labels 는 detect 가 그대로 광고에
	// 싣는다 (internal/enode/config.go 의 Labels · detect.go). 계약의 requires 가
	// 그 이름표로 자기 것만 잡는다.
	conf := map[string]any{
		"mediator":      cfg.Mediator.URL,
		"token":         cfg.Mediator.Token,
		"workspace":     dir,
		"workspace_id":  "itsaplan-orch-" + issueKey,
		"orchestration": true,
		"min_free_gb":   1,
		"labels":        map[string]string{"issue": issueKey},
	}
	b, err := yaml.Marshal(conf)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(confPath, b, 0o600); err != nil {
		return nil, err
	}

	logPath := filepath.Join(dir, "orch.log")
	lf, err := os.Create(logPath)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(cfg.Orchestrator.Bin,
		"--config", confPath,
		"--once",
		"--ready-file", readyPath)
	cmd.Stdout = lf
	cmd.Stderr = lf
	// 어댑터가 죽어도 오케스트레이터는 산다 — 그리고 Run 도 산다.
	// Mediator 와 노드는 어댑터를 모른다 (adapter-example §2).
	if err := cmd.Start(); err != nil {
		lf.Close()
		return nil, err
	}
	o := &Orchestrator{
		IssueKey:  issueKey,
		Dir:       dir,
		ConfigPan: confPath,
		cmd:       cmd,
		log:       log,
		keep:      cfg.Orchestrator.Keep,
		done:      make(chan struct{}),
	}
	// 거두는 자리는 여기 하나다 — 어느 경로로 끝나든 자식이 좀비로 안 남는다.
	go func() {
		_ = cmd.Wait()
		lf.Close()
		close(o.done)
	}()
	log.Info("오케스트레이터를 띄웠다", "issue", issueKey, "dir", dir, "pid", cmd.Process.Pid)

	// ready 파일에는 node_id 가 들어간다.
	deadline := time.Now().Add(cfg.ReadyWait())
	for time.Now().Before(deadline) {
		if raw, err := os.ReadFile(readyPath); err == nil && len(raw) > 0 {
			o.NodeID = trimLine(string(raw))
			log.Info("오케스트레이터가 광고를 마쳤다", "issue", issueKey, "node", o.NodeID)
			return o, nil
		}
		select {
		case <-ctx.Done():
			o.Stop()
			return nil, ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
	o.Stop()
	return nil, fmt.Errorf("오케스트레이터가 %s 안에 광고하지 않았다 (로그: %s)",
		cfg.ReadyWait(), logPath)
}

// WaitAdvertised 는 Mediator 의 광고에 이 이슈의 이름표가 보일 때까지 기다린다.
//
// ready 파일만으로는 부족하다 — 그것은 노드 쪽 사실이고, 매칭은 Mediator 가
// 본 것으로 한다. 둘 사이에 지연이 있으면 Run 이 422 로 떨어진다.
func (o *Orchestrator) WaitAdvertised(ctx context.Context, m *Mediator, wait time.Duration) error {
	deadline := time.Now().Add(wait)
	for time.Now().Before(deadline) {
		ok, err := m.HasLabel(ctx, "orchestration", "issue", o.IssueKey)
		if err == nil && ok {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return fmt.Errorf("함대의 광고에 issue=%s 가 %s 안에 안 나타났다", o.IssueKey, wait)
}

// Stop 은 오케스트레이터를 끝내고 정리한다.
//
// --once 라 보통은 스스로 죽는다 — 이 함수는 그러지 못한 경우의 뒷정리다.
// 이미 끝났으면 죽이지 않고 거두기만 한다.
func (o *Orchestrator) Stop() {
	if o.cmd != nil && o.cmd.Process != nil {
		select {
		case <-o.done: // 스스로 끝났다
		default:
			_ = o.cmd.Process.Kill()
		}
	}
	o.awaitAndClean()
}

// Detach 는 죽이지 않고 거두기만 한다 — 되묻기로 손을 뗄 때 쓴다.
//
// 그 순간 오케스트레이터는 살아 있어야 한다. 사람이 답하면 계약이
// 그 자리에서 이어지기 때문이다(ADR-047 이 그동안 임대를 안 죽인다).
// 그래도 언젠가는 끝나므로 거둘 사람이 필요하다.
func (o *Orchestrator) Detach() {
	go o.awaitAndClean()
}

func (o *Orchestrator) awaitAndClean() {
	if o.done != nil {
		<-o.done
	}
	if o.keep {
		o.log.Info("워크스페이스를 남긴다", "dir", o.Dir)
		return
	}
	// 지우는 것은 띄운 쪽이다 (adapter-example §4.2). 어댑터가 먼저 죽으면
	// 남지만, WorkspaceRoot 가 /tmp 면 재부팅이 지운다.
	if err := os.RemoveAll(o.Dir); err != nil {
		o.log.Warn("워크스페이스를 못 지웠다", "dir", o.Dir, "err", err)
	}
}

func trimLine(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r' || s[len(s)-1] == ' ') {
		s = s[:len(s)-1]
	}
	return s
}
