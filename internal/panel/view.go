package panel

import (
	"context"
	"encoding/json"
	"time"

	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/enode"
	"github.com/taeels/enode/internal/proc"
	"github.com/taeels/enode/internal/scratch"
)

// State 는 화면이 읽는 값이다. 다섯 묶음이고 출처가 저마다 달라, 하나가 막혀도
// 나머지는 채워진다 (business-rules §2).
type State struct {
	Node     string   `json:"node"`
	Identity Identity `json:"identity"`
	Caps     CapsView `json:"caps"`
	Process  ProcView `json:"process"`
	Work     WorkView `json:"work"`
	Drain    string   `json:"drain"` // "" | graceful | at-boundary — 실린 값
	// DrainSources 는 누가 건 drain 인가다 (완료 조건 1). DrainFrom 은 어디서 읽은 값인가 —
	// "status" 는 데몬이 광고에 실은 값 · "policy" 는 정책 파일의 값이다.
	DrainSources []enode.DrainSource `json:"drain_sources"`
	DrainFrom    string              `json:"drain_from"`
	// Scratch 는 곧 지워질 trash 의 양이다 (완료 조건 2). scratch 가 없는 노드는 없다.
	Scratch  *scratch.Usage `json:"scratch,omitempty"`
	Mediator MediatorView   `json:"mediator"`
}

// drain 칸의 출처 (business-rules.md 8.1).
const (
	drainFromStatus = "status"
	drainFromPolicy = "policy"
)

type Identity struct {
	NodeID    string `json:"node_id"`
	Label     string `json:"label"`
	Principal string `json:"principal"`
	Instance  string `json:"instance"` // Mediator 광고 행에서. 멈춘 노드는 빈다
}

type CapsView struct {
	Known bool                  `json:"known"` // 상태 파일이 있고 읽혔나
	Caps  []contract.Capability `json:"caps"`
	At    *time.Time            `json:"at"` // 마지막 탐지 시각. 없으면 nil
}

type ProcView struct {
	Running bool `json:"running"`
	Pid     int  `json:"pid"`
}

type WorkView struct {
	HasLease  bool       `json:"has_lease"`
	RunID     string     `json:"run_id,omitempty"`
	NotAfter  *time.Time `json:"not_after,omitempty"`
	Step      string     `json:"step,omitempty"`
	Attempt   int        `json:"attempt,omitempty"`
	StartedAt *time.Time `json:"started_at,omitempty"`
}

type MediatorView struct {
	Reachable    bool       `json:"reachable"`
	LastResponse *time.Time `json:"last_response,omitempty"`
}

// nodesResp · nodeRow · lease 는 GET /v1/nodes 원문에서 이 노드의 행만 뽑는
// 최소 파싱이다 (obs 가 원문으로 남긴 것을 panel 이 자기 쓰임에 맞게 좁힌다).
type nodesResp struct {
	Nodes []nodeRow `json:"nodes"`
}

type nodeRow struct {
	NodeID   string `json:"node_id"`
	Instance string `json:"instance"`
	Lease    *lease `json:"lease"`
}

type lease struct {
	RunID    string     `json:"run_id"`
	NotAfter *time.Time `json:"not_after"`
}

// state 는 지금의 화면 값을 모은다. 각 묶음은 독립 실패한다.
func (s *Server) state(ctx context.Context) State {
	st := State{Node: s.cfg.Node}

	// 신원 — 로컬. Derive 가 실패했으면 빈 채로 둔다 (New 가 이미 시도했다).
	st.Identity = Identity{
		NodeID:    s.ident.NodeID,
		Label:     s.ident.Label,
		Principal: s.ident.Principal,
	}

	// 탐지 능력 · scratch 의 양 — 로컬 상태 파일. 없으면 「아직 모름」.
	var status *enode.Status
	if got, err := enode.ReadStatus(s.cfg.ConfigPath); err == nil {
		status = &got
		at := got.At
		st.Caps = CapsView{Known: true, Caps: got.Caps, At: &at}
		st.Scratch = got.Scratch
	}

	// 프로세스 — 로컬 잠금 파일 + proc.
	if pid := proc.PidFromLock(s.cfg.ConfigPath); pid > 0 {
		st.Process = ProcView{Running: true, Pid: pid}
	}

	// drain — 데몬이 돌면 상태 파일(실린 값과 출처) · 아니면 로컬 정책 파일.
	var policy *enode.Policy
	if p, err := enode.ReadPolicyFile(s.cfg.ConfigPath); err == nil {
		policy = &p
	}
	st.Drain, st.DrainSources, st.DrainFrom = drainView(st.Process.Running, status, policy)

	// 현재 작업 · instance — Mediator 조회 (두 홉). 불통이면 로컬 묶음은 이미 섰다.
	// 도달 여부는 「응답했나」다 — 멈춘 노드는 목록에 없어도 Mediator 는 살아 있다.
	row, found, reachable := s.thisNode(ctx)
	st.Mediator.Reachable = reachable
	if found {
		st.Identity.Instance = row.Instance
		if row.Lease != nil && row.Lease.RunID != "" {
			st.Work = WorkView{HasLease: true, RunID: row.Lease.RunID, NotAfter: row.Lease.NotAfter}
			s.fillStep(ctx, &st.Work)
		}
	}
	if last := s.mediatorLast(); !last.IsZero() {
		st.Mediator.LastResponse = &last
	}
	return st
}

// drainView 는 drain 칸을 정한다 (business-rules.md 8.1).
//
// 데몬이 돌고 상태 파일에 drain 칸이 있으면 데몬이 광고에 실은 값과 그 출처다 — 소유자가
// 풀어도 여유 부족이 남아 노드가 빠져 있는 사실은 거기에만 있다. 아니면 오늘처럼 정책 파일의
// 값이고 출처는 소유자 하나다. 데몬이 멈췄으면 광고가 없으므로 스스로 건 drain 도 없다.
// 정책 파일을 못 읽었으면(nil) 빈 값이다.
func drainView(running bool, status *enode.Status, policy *enode.Policy) (string, []enode.DrainSource, string) {
	if running && status != nil && status.Drain != nil {
		return status.Drain.Effective, status.Drain.Sources, drainFromStatus
	}
	if policy == nil {
		return "", nil, drainFromPolicy
	}
	var sources []enode.DrainSource
	if src, ok := enode.OwnerDrain(policy.Drain); ok {
		sources = append(sources, src)
	}
	return policy.Drain, sources, drainFromPolicy
}

// thisNode 는 GET /v1/nodes 원문에서 이 노드의 행을 뽑는다.
//
// 셋을 가른다 — 행(row) · 찾았나(found) · Mediator 가 응답했나(reachable).
// 멈춘 노드는 광고 만료로 목록에 없지만 Mediator 는 여전히 도달 가능하다.
func (s *Server) thisNode(ctx context.Context) (row nodeRow, found, reachable bool) {
	raw, err := s.client.Nodes(ctx)
	if err != nil {
		s.markMediator(false)
		return nodeRow{}, false, false
	}
	s.markMediator(true)
	var resp nodesResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nodeRow{}, false, true
	}
	for _, n := range resp.Nodes {
		if n.NodeID == s.ident.NodeID {
			return n, true, true
		}
	}
	return nodeRow{}, false, true
}

// fillStep 은 임대한 Run 의 CLAIMED 단계 이름·회차·시작 시각을 채운다 (두 번째 홉).
func (s *Server) fillStep(ctx context.Context, w *WorkView) {
	run, err := s.client.Status(ctx, w.RunID)
	if err != nil {
		return
	}
	s.markMediator(true)
	for _, step := range run.Steps {
		if step.State == "CLAIMED" {
			w.Step = step.ID
			w.Attempt = step.Attempt
			w.StartedAt = step.StartedAt
			return
		}
	}
}
