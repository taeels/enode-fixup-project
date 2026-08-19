package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/taeels/enode/internal/contract"
)

// Check 는 계약 조건 하나의 대조 결과다. verdict.json 이 된다 (ADR-005).
type Check struct {
	Step string      `json:"step"`
	What string      `json:"what"` // exit_code | produced
	Want interface{} `json:"want"`
	Got  interface{} `json:"got"`
	OK   bool        `json:"ok"`
	Note string      `json:"note,omitempty"`
}

type Verdict struct {
	State  string  `json:"state"`
	Checks []Check `json:"checks"`
}

// Verify 는 ⑩ 이다 — ★ 계약에 선언된 기계적 조건만 대조한다 ★ (ADR-004 · I3).
//
// ★ 여기서 하지 않는 것 ★
//   - 결과값의 좋고 나쁨. test_result 가 FAIL 이어도 조건에 없으면 안 본다.
//   - 단계 간 관계 (⑦.result != ⑨.result). 쓰려면 식 언어가 필요해지고
//     차분 분류는 ⑫ 에서 껍데기가 2비트 조회표로 한다.
//
// ★ 그래서 회귀를 증명한 Run 이 SUCCEEDED 다 ★ — 결과값을 통과 기준에 넣으면
// 회귀를 찾아낸 Run 이 FAILED 가 되어 ADR-004 의 네 결과표가 뒤집힌다.
func Verify(c contract.Contract, results map[string]StepResult) Verdict {
	v := Verdict{State: StateSucceeded}
	for _, cond := range c.SuccessWhen {
		res, ran := results[cond.Step]
		if !ran {
			v.Checks = append(v.Checks, Check{
				Step: cond.Step, What: "ran", OK: false,
				Note: "그 단계가 결과를 남기지 않았다",
			})
			v.State = StateFailed
			continue
		}
		if cond.ExitCode != nil {
			got := -1
			if res.ExitCode != nil {
				got = *res.ExitCode
			}
			ok := res.ExitCode != nil && *res.ExitCode == *cond.ExitCode
			v.Checks = append(v.Checks, Check{
				Step: cond.Step, What: "exit_code", Want: *cond.ExitCode, Got: got, OK: ok,
			})
			if !ok {
				v.State = StateFailed
			}
		}
		if len(cond.Produced) > 0 {
			have := map[string]bool{}
			for _, p := range res.Produced {
				have[p] = true
			}
			var missing []string
			for _, want := range cond.Produced {
				if !have[want] {
					missing = append(missing, want)
				}
			}
			ok := len(missing) == 0
			ch := Check{
				Step: cond.Step, What: "produced",
				Want: cond.Produced, Got: res.Produced, OK: ok,
			}
			if !ok {
				ch.Note = fmt.Sprintf("없는 것: %v", missing)
				v.State = StateFailed
			}
			v.Checks = append(v.Checks, ch)
		}
	}
	return v
}

// StepResults 는 대조에 쓸 결과를 모은다. 이름(계약의 steps[].id)으로 색인한다.
func (s *Store) StepResults(ctx context.Context, runID string) (map[string]StepResult, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT name, result FROM steps WHERE run_id=$1 AND result IS NOT NULL`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]StepResult{}
	for rows.Next() {
		var name string
		var raw []byte
		if err := rows.Scan(&name, &raw); err != nil {
			return nil, err
		}
		var r StepResult
		if err := json.Unmarshal(raw, &r); err != nil {
			return nil, err
		}
		out[name] = r
	}
	return out, rows.Err()
}
