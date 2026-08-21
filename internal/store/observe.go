package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/taeels/enode/internal/contract"
)

// StepView 는 ★ 실행 중에 밖에서 보이는 단계 하나 ★ 다 (ADR-025).
//
// ★ 이것은 Record 가 아니다 ★ — Record 는 봉인된 사실의 묶음이고 tar 이며
// 종료 후에만 있고 불변이다(I4). 이쪽은 DB 의 ★ 지금 ★ 값이고 JSON 이며
// 언제나 답하고 ★ 계속 바뀐다 ★. 이름도 형식도 수명도 다르므로
// "봉인되지 않은 것은 Record 가 아니다" 가 그대로 산다.
//
// ★ 그래서 본문을 안 싣는다 ★ — 산출물 본문이나 판정을 여기 실으면
// "이걸로 충분한데 왜 Record 를 기다리나" 가 되고 그 순간 I4 가 형해화된다.
type StepView struct {
	Seq   int    `json:"seq"`
	ID    string `json:"id"`
	State string `json:"state"`
	Uses  string `json:"uses"`
	// Node 는 ★ 어느 기계에서 도는가 ★ 다 — Case D 의 "서로 다른 기계였다" 를
	// 실행 중에도 읽게 한다. label 은 assigned 가 이미 든다.
	Node string `json:"node,omitempty"`
	// Needs 는 ★ 비어 있어도 내보낸다 ★ — [] 는 "아무것도 안 기다린다" 는
	// 뜻이고 그것이 가지의 시작점을 가리킨다. 정규화된 값은 ★ DB 에만 있으므로 ★
	// (계약에서는 생략될 수 있다) 안 내보내면 읽는 쪽이 기본값 규칙을 추측한다.
	Needs []string `json:"needs"`
	// Attempt 는 ★ 재시도가 도는 중인지 ★ 를 밖에서 알게 한다.
	// 병렬이면 회차가 ★ 가지마다 따로 ★ 돈다.
	Attempt   int        `json:"attempt,omitempty"`
	StartedAt *time.Time `json:"started_at,omitempty"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
}

// Steps 는 그 Run 의 단계들을 순서대로 돌려준다 (ADR-025).
//
// ★ 폭이 1 을 넘는 순간 필요해졌다 ★ — 순차면 "지금 ④ 단계" 한 줄로 족했지만
// 여러 가지가 동시에 살면 각각이 다른 상태에 있고, GET record 는 종료 전이면
// 409 다(I4). ⇒ 진행을 읽을 경로가 ★ 선택이 아니라 필수 ★ 가 된다.
func (s *Store) Steps(ctx context.Context, runID string) ([]StepView, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT seq, name, state, uses, coalesce(node_id,''), needs, attempt,
		       started_at, ended_at
		  FROM steps WHERE run_id = $1 ORDER BY seq`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StepView{}
	for rows.Next() {
		var v StepView
		if err := rows.Scan(&v.Seq, &v.ID, &v.State, &v.Uses, &v.Node, &v.Needs,
			&v.Attempt, &v.StartedAt, &v.EndedAt); err != nil {
			return nil, err
		}
		if v.Needs == nil {
			v.Needs = []string{}
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// LedgerEntry 는 원장의 항목 하나다 — ★ 본문이 없다 ★ (ADR-023 §6.3).
type LedgerEntry struct {
	// RunID 는 ★ scope:"work" 일 때만 채워진다 ★ — 이 Run 밖에서 온 것이라는 표시다.
	// 같은 Run 안의 것에 붙이면 모든 줄에 같은 값이 반복될 뿐이다.
	RunID   string `json:"run_id,omitempty"`
	Seq     int    `json:"seq"`
	Attempt int    `json:"attempt"`
	Name    string `json:"name"`
	// By 는 ★ 누가 냈나 ★ 다 — "step:<id>". ADR-022 §7.6 의 by 와 ★ 같은 어휘 ★ 다.
	By    string    `json:"by"`
	At    time.Time `json:"at"`
	Bytes int64     `json:"bytes"`
	// SchemaOK 는 ★ 검증했는가와 통과했는가를 가른다 ★ (ADR-020 의 status:none 과 같은 이유).
	//	nil    계약에 그 이름의 스키마가 ★ 없었다 ★ — 검증 안 했다
	//	true   있었고 통과했다 (통과 못 하면 애초에 저장되지 않는다 — 422)
	SchemaOK *bool `json:"schema_ok,omitempty"`
}

// Ledger 는 그 Run 이 ★ 지금 발견할 수 있는 것 ★ 의 목록이다 (ADR-023 §6).
//
// ★ 시야를 순서에서 유도하지 않는다 ★ — 조상인지 형제인지를 묻지 않고,
// 그 시점에 원장에 ★ 있는가 ★ 만 본다. 그래서 병렬이 「최신」을 안 깬다.
//
// ★ scope:"work" 면 같은 Work 의 이전 Run 들이 낸 것까지 ★ 본다.
// 이것은 ADR-005 가 범위 밖으로 둔 「여러 Run 에 걸친 질의」가 ★ 아니다 ★ —
// 저쪽은 ★ 봉인 묶음에 대한 질의 ★ 이고, 이쪽은 work_id 인덱스로 Run 을 고른 뒤
// 각자의 blobs/ 를 나열하는 것이다. ★ 검색 계층도 질의 언어도 안 는다 ★.
func (s *Store) Ledger(ctx context.Context, runID string) ([]LedgerEntry, error) {
	if s.Records == nil {
		return nil, fmt.Errorf("기록 저장소가 없다")
	}
	run, err := s.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	// ★ 이름을 붙이려면 지금 유효한 계약이 필요하다 ★ — 계획이 지은 단계가
	// 낸 산출물은 제출 전문에 없는 이름이다.
	live, err := s.LiveContract(ctx, runID)
	if err != nil {
		return nil, err
	}
	out := []LedgerEntry{}

	// ★ 이전 Run 들이 먼저다 ★ — 시간 순서가 곧 목록의 순서다.
	if run.Contract.Scope() == contract.ScopeWork {
		prior, err := s.priorRuns(ctx, run)
		if err != nil {
			return nil, err
		}
		for _, p := range prior {
			es, err := s.entriesOf(p.runID, p.contract, p.runID)
			if err != nil {
				continue // ★ 지워졌거나 못 읽는 Run 이 지금 Run 을 막지 않는다 ★
			}
			out = append(out, es...)
		}
	}
	mine, err := s.entriesOf(runID, live, "")
	if err != nil {
		return nil, err
	}
	return append(out, mine...), nil
}

// LiveContract 는 지금 유효한 계약이다 (liveContract 참조).
// ★ 실행 경로는 전부 이것을 본다 ★ — 제출 전문은 봉인(v1)만 쓴다.
func (s *Store) LiveContract(ctx context.Context, runID string) (contract.Contract, error) {
	var raw []byte
	var c contract.Contract
	if err := s.pool.QueryRow(ctx,
		`SELECT `+liveContract+` FROM runs WHERE run_id=$1`, runID).Scan(&raw); err != nil {
		return c, err
	}
	return c, json.Unmarshal(raw, &c)
}

type priorRun struct {
	runID    string
	contract contract.Contract
}

// priorRuns 는 ★ 같은 Work 의 앞선 Run 들 ★ 이다. 자기 자신은 뺀다.
//
// work_id 가 ★ Run 을 넘어 사는 유일한 식별자 ★ 이므로 이 조회가 성립한다
// (ADR-023 §6.5.2). 없으면 — 계약이 work 를 안 채웠으면 — 빈 목록이다.
func (s *Store) priorRuns(ctx context.Context, run *Run) ([]priorRun, error) {
	key := run.Contract.Work.Key()
	if key == "" {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT run_id, coalesce(contract_versions -> -1, contract) FROM runs
		 WHERE work_id = $1 AND run_id <> $2
		 ORDER BY created_at`, key, run.RunID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []priorRun{}
	for rows.Next() {
		var p priorRun
		var raw []byte
		if err := rows.Scan(&p.runID, &raw); err != nil {
			return nil, err
		}
		if json.Unmarshal(raw, &p.contract) != nil {
			continue
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// entriesOf 는 Run 하나의 산출물에 ★ 계약이 아는 것 ★ 을 붙인다 —
// 누가 냈는지(단계 이름)와 스키마로 검증됐는지. record 는 계약을 모르므로
// 그 둘은 여기서만 붙일 수 있다.
func (s *Store) entriesOf(runID string, c contract.Contract, label string) ([]LedgerEntry, error) {
	metas, err := s.Records.Blobs(runID)
	if err != nil {
		return nil, err
	}
	out := make([]LedgerEntry, 0, len(metas))
	for _, m := range metas {
		e := LedgerEntry{RunID: label, Seq: m.Seq, Attempt: m.Attempt,
			Name: m.Name, At: m.At, Bytes: m.Bytes}
		if m.Seq >= 1 && m.Seq <= len(c.Steps) {
			st := c.Steps[m.Seq-1]
			e.By = "step:" + st.ID
			if _, ok := st.Schema[m.Name]; ok {
				ok := true
				e.SchemaOK = &ok
			}
		}
		out = append(out, e)
	}
	return out, nil
}
