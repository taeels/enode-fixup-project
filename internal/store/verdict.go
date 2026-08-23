package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/taeels/enode/internal/contract"
)

// 단계의 상태 (schema.sql 의 steps.state).
//
// ★ SKIPPED 는 dispatch 가 쓴다 ★ (ADR-022 §7.2 · B2) — 경로가 갈리면 안 간 쪽의
// 단계는 실행되지 않는다. 그것을 ★ 「없음」으로 두면 크래시와 구분이 안 된다 ★.
// ADR-020 이 "가설 없음" 을 부재가 아니라 status:none 이라는 ★ 값 ★ 으로 만든 것과
// 같은 이유다.
const (
	StepPending = "PENDING"
	StepClaimed = "CLAIMED"
	StepDone    = "DONE"
	StepFailed  = "FAILED"
	// StepSkipped 는 ★ 종료 상태이면서 실패가 아니다 ★.
	// reap 의 집계가 PENDING·CLAIMED·ASKED 를 「남은 것」, FAILED 를 「실패」로
	// ★ 열거해서 ★ 세므로, 여기 없는 상태는 저절로 둘 다 아니게 된다.
	// ⇒ ★ 상태를 늘리면 그 열거를 반드시 다시 본다 ★ — ASKED 를 더할 때
	//   빠뜨려서 ★ 질문이 열린 채 Run 이 끝나는 결함 ★ 을 실제로 밟았다.
	StepSkipped = "SKIPPED"
	// StepAsked 는 ★ 사람의 답을 기다린다 ★ (ADR-032). 남은 것으로 센다.
	StepAsked = "ASKED"
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
	// ★ 실제로 대조된 조건의 수 ★ — 공허한 참을 막는다 (아래 참조).
	evaluated := 0
	for _, cond := range c.SuccessWhen {
		res, ran := results[cond.Step]
		// ★ 건너뛴 단계의 조건은 공허하게 참이다 ★ (ADR-022 §7.2)
		//
		// 조건이 묻는 것은 "완주했는가" 인데(ADR-004), 실행 자체가 안 됐으면
		// ★ 그 물음이 성립하지 않는다 ★. 실패로 치면 dispatch 로 경로가 갈릴 때
		// ★ 안 간 쪽의 조건이 Run 을 죽인다 ★ — 계약 저자가 경로별로 조건을
		// 나눠 쓸 방법이 없으므로(경로는 실행 시 정해진다) 이쪽이 유일한 답이다.
		if ran && res.Skipped {
			v.Checks = append(v.Checks, Check{
				Step: cond.Step, What: "skipped", OK: true,
				Note: "the step did not run; the condition is not evaluated",
			})
			continue
		}
		if !ran {
			v.Checks = append(v.Checks, Check{
				Step: cond.Step, What: "ran", OK: false,
				Note: "the step left no result",
			})
			v.State = StateFailed
			continue
		}
		evaluated++
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
		if cond.WithinAttempts {
			// ★ 재시도 소진이 verdict 로 잡힌다 ★ — 루프는 제어 흐름이고
			// 성패는 여기서 정해진다 (ADR-004: 계약에 선언된 기계적 조건으로만).
			ok := !res.Exhausted
			v.Checks = append(v.Checks, Check{
				Step: cond.Step, What: "within_attempts", Want: true, Got: ok, OK: ok,
				Note: noteIf(!ok, "attempts exhausted"),
			})
			if !ok {
				v.State = StateFailed
			}
		}
		if len(cond.Changed) > 0 {
			// ★ 세상이 바뀌었는가 ★ (ADR-037) — produced 와 무게가 다르다.
			// produced 는 에이전트가 쓴 파일이고, 이것은 ★ 에이전트가 저작하지 않는
			// 관찰 ★ 이다. agent 단계가 아무것도 안 하고 「했다」고 말해도 여기서 걸린다.
			have := map[string]bool{}
			for _, p := range res.Changed {
				have[p] = true
			}
			var missing []string
			for _, want := range cond.Changed {
				if !have[want] {
					missing = append(missing, want)
				}
			}
			ok := len(missing) == 0
			v.Checks = append(v.Checks, Check{
				Step: cond.Step, What: "changed", Want: cond.Changed, Got: res.Changed, OK: ok,
				Note: noteIf(!ok, "not changed during this step: "+strings.Join(missing, ", ")),
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
				ch.Note = fmt.Sprintf("missing: %v", missing)
				v.State = StateFailed
			}
			v.Checks = append(v.Checks, ch)
		}
	}
	// ★ 공허한 참을 막는다 ★
	//
	// 조건이 있는데 ★ 하나도 대조되지 않았다 ★ = 전부 건너뛰었다는 뜻이다.
	// 그대로 두면 ★ 아무것도 안 하고 SUCCEEDED ★ 가 된다.
	// 「검증이 불필요한 패치」도 ★ 그 경로에 보고 단계가 있어야 ★ 하고,
	// 그 단계에 조건이 걸려야 한다. 그게 이 규칙이 요구하는 것이다.
	if len(c.SuccessWhen) > 0 && evaluated == 0 {
		v.Checks = append(v.Checks, Check{
			What: "any", Want: "at least one evaluated condition", Got: 0, OK: false,
			Note: "every step with a condition was skipped; nothing was evaluated",
		})
		v.State = StateFailed
	}
	return v
}

func noteIf(cond bool, s string) string {
	if cond {
		return s
	}
	return ""
}

// StepResults 는 대조에 쓸 결과를 모은다. 이름(계약의 steps[].id)으로 색인한다.
func (s *Store) StepResults(ctx context.Context, runID string) (map[string]StepResult, error) {
	// ★ 건너뛴 단계도 싣는다 ★ — 결과는 없지만 "실행 안 됐다" 를 Verify 가
	// 알아야 한다. 안 실으면 크래시(결과 없음)와 구분이 안 된다.
	rows, err := s.pool.Query(ctx,
		`SELECT name, result, attempt, state FROM steps
		  WHERE run_id=$1 AND (result IS NOT NULL OR state='SKIPPED')`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]StepResult{}
	for rows.Next() {
		var name, state string
		var raw []byte
		var attempt int
		if err := rows.Scan(&name, &raw, &attempt, &state); err != nil {
			return nil, err
		}
		var r StepResult
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &r); err != nil {
				return nil, err
			}
		}
		r.Attempt = attempt
		r.Skipped = state == StepSkipped
		out[name] = r
	}
	return out, rows.Err()
}
