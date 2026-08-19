package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/record"
)

// Manifest 는 Record 의 manifest.json 이다 (ADR-005).
//
// ★ 계약 전문이 여기 들어간다 ★ — 성질 4(자기충족)의 핵심이다.
// ADR-020 이 스키마를 인라인으로 둔 덕에 ★ 무엇으로 검증했는지 ★ 까지 함께 남는다.
type Manifest struct {
	RunID     string            `json:"run_id"`
	Work      contract.Work     `json:"work"`
	Principal string            `json:"principal"` // 요청자. ★ 식별이지 인증이 아니다 ★
	State     string            `json:"state"`
	CreatedAt time.Time         `json:"created_at"`
	EndedAt   *time.Time        `json:"ended_at,omitempty"`
	Assigned  []Assigned        `json:"assigned,omitempty"`
	Contract  contract.Contract `json:"contract"`
}

// StepFiles 는 봉인에 쓸 단계 기록을 모은다.
// ★ node id 와 label 을 함께 남긴다 ★ (성질 3) — Case D 의
// "서로 다른 기계였다" 를 사람이 읽을 수 있어야 한다.
func (s *Store) StepFiles(ctx context.Context, runID string) ([]record.StepFile, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT st.seq, st.name, st.uses, st.kind, coalesce(st.node_id,''),
		       coalesce(n.label,''), st.state, st.started_at, st.ended_at, st.result
		  FROM steps st
		  LEFT JOIN nodes n ON n.node_id = st.node_id
		 WHERE st.run_id = $1 ORDER BY st.seq`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []record.StepFile
	for rows.Next() {
		var f record.StepFile
		var started, ended *time.Time
		var raw []byte
		if err := rows.Scan(&f.Seq, &f.Name, &f.Uses, &f.Kind, &f.Node, &f.NodeLabel,
			&f.State, &started, &ended, &raw); err != nil {
			return nil, err
		}
		f.StepID = stepID(runID, f.Seq)
		if started != nil {
			f.StartedAt = started.UTC().Format(time.RFC3339Nano)
		}
		if ended != nil {
			f.EndedAt = ended.UTC().Format(time.RFC3339Nano)
		}
		if len(raw) > 0 {
			var r StepResult
			if err := json.Unmarshal(raw, &r); err == nil {
				f.Result = r
			}
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func stepID(runID string, seq int) string {
	return runID + "#" + pad2(seq)
}

func pad2(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// sealRecord 는 종료 상태에 이른 Run 의 기록을 완성하고 불변으로 만든다.
// 봉인은 멱등이다 — 이미 봉인됐으면 아무것도 하지 않는다 (I4).
func (s *Store) sealRecord(ctx context.Context, runID string, v Verdict) error {
	if s.Records == nil {
		return nil
	}
	run, err := s.GetRun(ctx, runID)
	if err != nil {
		return err
	}
	steps, err := s.StepFiles(ctx, runID)
	if err != nil {
		return err
	}
	var ended *time.Time
	if t, err := s.endedAt(ctx, runID); err == nil {
		ended = t
	}
	m := Manifest{
		RunID: run.RunID, Work: run.Contract.Work, Principal: run.Principal,
		State: v.State, CreatedAt: run.CreatedAt, EndedAt: ended,
		Assigned: run.Assigned, Contract: run.Contract,
	}
	return s.Records.Seal(runID, m, v, steps)
}

func (s *Store) endedAt(ctx context.Context, runID string) (*time.Time, error) {
	var t *time.Time
	err := s.pool.QueryRow(ctx, `SELECT ended_at FROM runs WHERE run_id=$1`, runID).Scan(&t)
	return t, err
}
