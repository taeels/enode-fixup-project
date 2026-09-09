package panel

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/taeels/enode/internal/enode"
	"github.com/taeels/enode/internal/runctl"
)

// 트랜스크립트 — 지금 도는 것(로컬 링 파일)과 지난 것(Mediator 가 가진 것) 둘이다
// (decisions §6.5). Mediator 를 안 고친다 — 제어판이 읽어 자기 node_id 로 거른다.

// handleTranscript 는 지금 도는 것 — 로컬 링 파일을 읽어 낸다 (1초 폴링 대상).
func (s *Server) handleTranscript(w http.ResponseWriter, r *http.Request) {
	snap, err := enode.ReadRing(enode.TranscriptPath(s.cfg.ConfigPath))
	if err != nil {
		writeJSON(w, map[string]any{"available": false})
		return
	}
	writeJSON(w, map[string]any{
		"available":  true,
		"generation": snap.Generation,
		"total":      snap.Total,
		"data":       string(snap.Data),
	})
}

type pastRun struct {
	RunID   string          `json:"run_id"`
	State   string          `json:"state"`
	EndedAt *time.Time      `json:"ended_at,omitempty"`
	Verdict json.RawMessage `json:"verdict,omitempty"`
}

// runsResp 는 GET /v1/runs 원문에서 이 유닛이 쓰는 것만 뽑는 최소 파싱이다.
type runsResp struct {
	Runs []struct {
		RunID    string          `json:"run_id"`
		State    string          `json:"state"`
		EndedAt  *time.Time      `json:"ended_at"`
		Verdict  json.RawMessage `json:"verdict"`
		Assigned []struct {
			Nodes []struct {
				Node string `json:"node"`
			} `json:"nodes"`
		} `json:"assigned"`
	} `json:"runs"`
}

// handleRuns 는 지난 것 — GET /v1/runs 를 받아 assigned 에 이 노드가 있는 Run 만
// 낸다. node= 서버 필터를 안 만든다 (decisions §6.5).
func (s *Server) handleRuns(w http.ResponseWriter, r *http.Request) {
	raw, err := s.client.Runs(r.Context(), runctl.RunsQuery{})
	if err != nil {
		s.markMediator(false)
		http.Error(w, "cannot reach the mediator", http.StatusBadGateway)
		return
	}
	s.markMediator(true)
	var resp runsResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		http.Error(w, "cannot parse runs", http.StatusBadGateway)
		return
	}
	out := []pastRun{}
	for _, run := range resp.Runs {
		here := false
		for _, a := range run.Assigned {
			for _, n := range a.Nodes {
				if n.Node == s.ident.NodeID {
					here = true
				}
			}
		}
		if !here {
			continue
		}
		out = append(out, pastRun{RunID: run.RunID, State: run.State, EndedAt: run.EndedAt, Verdict: run.Verdict})
	}
	writeJSON(w, map[string]any{"runs": out})
}

type recordLog struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// handleRecord 는 봉인된 지난 트랜스크립트다 — GET /v1/runs/{id}/record 의 tar 에서
// logs/NN-*.log 만 꺼낸다 (decisions §6.5). archive/tar 는 표준이라 새 의존 0.
func (s *Server) handleRecord(w http.ResponseWriter, r *http.Request) {
	runID := r.URL.Query().Get("run")
	if runID == "" {
		http.Error(w, "run is required", http.StatusBadRequest)
		return
	}
	var buf bytes.Buffer
	if err := s.client.Record(r.Context(), runID, &buf); err != nil {
		s.markMediator(false)
		http.Error(w, "cannot fetch the record", http.StatusBadGateway)
		return
	}
	s.markMediator(true)
	logs := []recordLog{}
	tr := tar.NewReader(&buf)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			http.Error(w, "cannot read the record tar", http.StatusBadGateway)
			return
		}
		name := h.Name
		if !strings.HasPrefix(name, "logs/") || !strings.HasSuffix(name, ".log") {
			continue
		}
		b, err := io.ReadAll(tr)
		if err != nil {
			continue
		}
		logs = append(logs, recordLog{Name: name, Content: string(b)})
	}
	writeJSON(w, map[string]any{"logs": logs})
}
