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
// ★ 이것이 새로 그린 자리다 ★ (docs/itsaplan-adapter-example.md §1.2 ③) —
// 어댑터가 Mediator 안에 있지 않고, 밖에서 ★ 이슈마다 오케스트레이터를 띄운다 ★.
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
}

// StartOrchestrator 는 설정을 짓고 --once 로 띄운다.
//
// ★ --once 가 탄력 노드를 닫는다 ★ — Run 이 끝나면 스스로 종료하고, 광고가
// 만료되어 함대에서 저절로 빠진다 (ADR-012). 어댑터가 죽여야 할 것이 안 남는다.
func StartOrchestrator(ctx context.Context, cfg *Config, issueKey string, log *slog.Logger) (*Orchestrator, error) {
	dir := filepath.Join(cfg.Orchestrator.WorkspaceRoot, "orch-"+issueKey)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	confPath := filepath.Join(dir, "config.yaml")
	readyPath := filepath.Join(dir, "ready")
	_ = os.Remove(readyPath)

	// ★ 이름표가 하나 붙는 것이 전부다 ★ — Labels 는 detect 가 그대로 광고에
	// 싣는다 (internal/enode/config.go 의 Labels · detect.go). 계약의 requires 가
	// 그 이름표로 ★ 자기 것만 잡는다 ★.
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
	// ★ 어댑터가 죽어도 오케스트레이터는 산다 ★ — 그리고 Run 도 산다.
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
	}
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
// ★ ready 파일만으로는 부족하다 ★ — 그것은 노드 쪽 사실이고, 매칭은 Mediator 가
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

// Stop 은 오케스트레이터를 정리한다.
//
// ★ --once 라 보통은 스스로 죽는다 ★ — 이 함수는 그러지 못한 경우의 뒷정리다.
func (o *Orchestrator) Stop() {
	if o.cmd != nil && o.cmd.Process != nil {
		if o.cmd.ProcessState == nil {
			_ = o.cmd.Process.Kill()
		}
		_ = o.cmd.Wait()
	}
	if o.keep {
		o.log.Info("워크스페이스를 남긴다", "dir", o.Dir)
		return
	}
	// ★ 지우는 것은 띄운 쪽이다 ★ (adapter-example §4.2). 어댑터가 먼저 죽으면
	// 남지만, WorkspaceRoot 가 /tmp 면 재부팅이 지운다.
	if err := os.RemoveAll(o.Dir); err != nil {
		o.log.Warn("워크스페이스를 못 지웠다", "dir", o.Dir, "err", err)
	}
}

// Wait 는 오케스트레이터가 스스로 끝날 때까지 기다린다 (상한 있음).
func (o *Orchestrator) Wait(d time.Duration) {
	if o.cmd == nil || o.cmd.Process == nil {
		return
	}
	done := make(chan struct{})
	go func() { _ = o.cmd.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(d):
	}
}

func trimLine(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r' || s[len(s)-1] == ' ') {
		s = s[:len(s)-1]
	}
	return s
}
