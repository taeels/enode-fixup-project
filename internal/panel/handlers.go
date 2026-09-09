package panel

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/enode"
	"github.com/taeels/enode/internal/proc"
)

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.state(r.Context()))
}

// handleDrain 은 정책 파일에 drain 모드를 쓴다. 걸기와 모드 고르기다.
//
// 기존 Policy 를 읽어 Drain 만 바꿔 쓴다 — panel_token 이 보존된다.
func (s *Server) handleDrain(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = contract.DrainGraceful // 기본은 graceful (decisions §1)
	}
	if mode != contract.DrainGraceful && mode != contract.DrainAtBoundary {
		http.Error(w, "mode must be graceful or at-boundary", http.StatusBadRequest)
		return
	}
	if err := s.setDrain(mode); err != nil {
		http.Error(w, "cannot write the policy file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"drain": mode})
}

// handleUndrain 은 drain 을 푼다 — 정책 파일에 빈 값을 쓴다. 자동 복귀는 없다.
func (s *Server) handleUndrain(w http.ResponseWriter, r *http.Request) {
	if err := s.setDrain(contract.DrainNone); err != nil {
		http.Error(w, "cannot write the policy file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"drain": ""})
}

func (s *Server) setDrain(mode string) error {
	p, err := enode.ReadPolicyFile(s.cfg.ConfigPath)
	if err != nil {
		return err
	}
	p.Drain = mode
	return enode.WritePolicyFile(s.cfg.ConfigPath, p)
}

// handleStop 은 도는 Run 을 먼저 cancel 하고 데몬을 끈다 (decisions §「제어판」).
//
// cancel 을 먼저 하는 이유 — 안 그러면 그 Run 이 lease expired: renewal stopped 로
// 죽어 소유자가 껐다는 사실이 기록에 안 남고, drain 이 verdict 에 drain:<node_id>
// 를 남기는 것과 대칭이 깨진다. Mediator 가 안 닿으면 그 사실을 문구로 말하고
// 그대로 끈다.
func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	pid := proc.PidFromLock(s.cfg.ConfigPath)
	if pid == 0 {
		writeJSON(w, map[string]any{"stopped": true, "message": "already stopped"})
		return
	}
	row, _, reachable := s.thisNode(r.Context())
	runID := ""
	if row.Lease != nil {
		runID = row.Lease.RunID
	}
	cancelled := false
	if runID != "" {
		if _, err := s.client.Cancel(r.Context(), runID); err == nil {
			cancelled = true
			s.markMediator(true)
		} else {
			reachable = false
		}
	}
	sawMediator := reachable
	if err := proc.SignalStop(pid); err != nil {
		http.Error(w, "cannot stop the daemon: "+err.Error(), http.StatusInternalServerError)
		return
	}
	msg := "stopped"
	if !sawMediator {
		msg = "stopped, but the mediator was unreachable; any running run will die as lease expired: renewal stopped, with no record that you stopped it"
	} else if runID != "" && cancelled {
		msg = "cancelled the running run and stopped the daemon"
	}
	writeJSON(w, map[string]any{"stopped": true, "cancelled": cancelled, "mediator_reachable": sawMediator, "message": msg})
}

// handleStart 은 멈춘 노드를 띄운다. 제어판 자신이 enode 이므로(enode panel)
// 자기 실행파일을 --config 로 다시 띄운다. 부모에서 떼어내 제어판이 죽어도
// 노드는 산다.
func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
	if pid := proc.PidFromLock(s.cfg.ConfigPath); pid > 0 {
		writeJSON(w, map[string]any{"running": true, "pid": pid, "message": "already running"})
		return
	}
	self := s.startBin
	if self == "" {
		var err error
		if self, err = os.Executable(); err != nil {
			http.Error(w, "cannot find the enode binary: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if err := os.MkdirAll(enode.StateDir(), 0o755); err != nil {
		http.Error(w, "cannot create the state dir: "+err.Error(), http.StatusInternalServerError)
		return
	}
	logPath := s.logPath()
	log, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		http.Error(w, "cannot open the log: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer log.Close() //nolint:errcheck
	fmt.Fprintf(log, "\n===== %s start (panel) =====\n", time.Now().Format("2006-01-02 15:04:05"))

	c := exec.Command(self, "--config", s.cfg.ConfigPath)
	c.Stdout, c.Stderr = log, log
	c.SysProcAttr = proc.DetachAttr()
	if err := c.Start(); err != nil {
		http.Error(w, "cannot start the daemon: "+err.Error(), http.StatusInternalServerError)
		return
	}
	_ = c.Process.Release()

	// 떴는지 잠깐 본다 — enodectl start 와 같은 확인이다.
	time.Sleep(time.Second)
	pid := proc.PidFromLock(s.cfg.ConfigPath)
	if pid == 0 {
		writeJSON(w, map[string]any{"running": false, "message": "did not come up; check the log"})
		return
	}
	writeJSON(w, map[string]any{"running": true, "pid": pid, "message": "started"})
}

// handleLogs 는 데몬 로그의 꼬리를 보인다. 트랜스크립트 카드와 다른 물건이다
// (그건 transcript 유닛).
func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	b, err := os.ReadFile(s.logPath())
	if err != nil {
		http.Error(w, "no log", http.StatusNotFound)
		return
	}
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	if n := 200; len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(strings.Join(lines, "\n")))
}

func (s *Server) logPath() string {
	return filepath.Join(enode.StateDir(), s.cfg.Node+".log")
}
