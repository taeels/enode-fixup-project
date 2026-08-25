package store

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/taeels/enode/internal/contract"
)

// 단계의 상태 (schema.sql 의 steps.state).
//
// SKIPPED 는 dispatch 가 쓴다 (ADR-022 §7.2 · B2) — 경로가 갈리면 안 간 쪽의
// 단계는 실행되지 않는다. 그것을 「없음」으로 두면 크래시와 구분이 안 된다.
// ADR-020 이 "가설 없음" 을 부재가 아니라 status:none 이라는 값으로 만든 것과
// 같은 이유다.
const (
	StepPending = "PENDING"
	StepClaimed = "CLAIMED"
	StepDone    = "DONE"
	StepFailed  = "FAILED"
	// StepSkipped 는 종료 상태이면서 실패가 아니다.
	// reap 의 집계가 PENDING·CLAIMED·ASKED 를 「남은 것」, FAILED 를 「실패」로
	// 열거해서 세므로, 여기 없는 상태는 저절로 둘 다 아니게 된다.
	// ⇒ 상태를 늘리면 그 열거를 반드시 다시 본다 — ASKED 를 더할 때
	//   빠뜨려서 질문이 열린 채 Run 이 끝나는 결함을 실제로 밟았다.
	StepSkipped = "SKIPPED"
	// StepAsked 는 사람의 답을 기다린다 (ADR-032). 남은 것으로 센다.
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

// describeWant 는 함대 조건을 Check 에 사람이 읽게 적는다 —
// 속성 어휘가 창발하므로(ADR-012) 무엇을 요구했는지가 안 보이면 원인을 못 찾는다.
// 순서를 고정한다 — 맵을 그대로 돌면 같은 조건이 매번 다르게 봉인된다.
func describeWant(r contract.Require) string {
	keys := make([]string, 0, len(r.Attrs))
	for k := range r.Attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := "fleet_has " + r.Capability
	for _, k := range keys {
		out += " " + k + "=" + r.Attrs[k]
	}
	return out
}

// hasName 은 목록에 그 이름이 있는지다.
func hasName(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

type Verdict struct {
	State  string  `json:"state"`
	Checks []Check `json:"checks"`
	// Fleet 은 판정 시점에 관측한 함대다 (ADR-058) — fleet_has 를 쓴
	// 계약에만 담긴다.
	//
	// 왜 봉인하는가 — Verify 는 순수 함수이고 함대는 인자다. 그 인자를
	// 안 남기면 봉인된 묶음만 보고 판정을 재현할 수 없다 (ADR-005 성질 4).
	// 그리고 "왜 실패했나" 의 답이 대개 여기 있다: 그 노드가 그때 없었다.
	Fleet []contract.Advert `json:"fleet,omitempty"`
}

// Verify 는 ⑩ 이다 — 계약에 선언된 기계적 조건만 대조한다 (ADR-004 · I3).
//
// 여기서 하지 않는 것
//   - 결과값의 좋고 나쁨. test_result 가 FAIL 이어도 조건에 없으면 안 본다.
//   - 단계 간 관계 (⑦.result != ⑨.result). 쓰려면 식 언어가 필요해지고
//     차분 분류는 ⑫ 에서 껍데기가 2비트 조회표로 한다.
//
// 그래서 회귀를 증명한 Run 이 SUCCEEDED 다 — 결과값을 통과 기준에 넣으면
// 회귀를 찾아낸 Run 이 FAILED 가 되어 ADR-004 의 네 결과표가 뒤집힌다.
// Verify 는 순수 함수다 — DB 도 시계도 안 본다.
//
// 함대를 인자로 받는다 (ADR-058) — fleet_has 가 보는 것은 이 Run 밖의
// 사실이지만, 그것을 여기서 읽으면 순수 함수가 아니게 되고 Record 로
// 재현할 수 없다. 그래서 호출자가 읽어 넘기고, 그 스냅샷이 봉인된다.
// nil 이면 fleet_has 조건은 「관측하지 못했다」로 실패한다 — 조용히 참이
// 되지 않는다 (공허한 참을 막는 것은 이 파일의 오래된 규칙이다).
func Verify(c contract.Contract, results map[string]StepResult, fleet []contract.Advert) Verdict {
	v := Verdict{State: StateSucceeded}
	// 「목표에 못 닿았다」는 기준을 이긴다 (ADR-054)
	//
	// success_when 은 기계가 볼 수 있는 것만 본다. 그것이 참인데 목표는
	// 아닐 수 있다 — "아무것도 안 깔려 있다" 를 적은 보고서도 존재하는
	// 파일 이다 (vm-scratch-5 가 그렇게 통과할 뻔했다).
	//
	// 방향이 한쪽뿐이다 — 통과할 Run 을 실패시킬 수는 있고, 실패할 Run 을
	// 통과시킬 수는 없다. 그래서 ADR-037(판정 술어를 에이전트가 저작하지
	// 못하게 한다)을 약화시키지 않는다. 기준의 저자는 여전히 사람이다.
	//
	// 나머지 대조를 건너뛰지 않는다 — Checks 는 기록이고, 무엇이
	// 맞았고 무엇이 틀렸는지가 Record 에 다 남아야 한다.
	for _, st := range c.Steps {
		res, ran := results[st.ID]
		if !ran || !hasName(res.Produced, contract.UnmetName) {
			continue
		}
		if !st.Expands {
			// 목표를 판단하려면 전체 그림이 필요하다 (ADR-054 §2.2) —
			// goal · owed · standing 은 계획을 짓는 단계에만 실린다. 평범한
			// 단계는 자기 일만 알므로, 그 자리를 주면 자기가 막힌 것을
			// Run 전체의 실패로 선언한다. 그 말은 _cannot 의 자리다.
			//
			// 무시하되 기록한다 — 조용히 버리면 왜 안 먹혔는지 알 수 없다.
			v.Checks = append(v.Checks, Check{
				Step: st.ID, What: contract.UnmetName, Want: false, Got: true, OK: true,
				Note: "only a step that builds a plan may report the goal; ignored here",
			})
			continue
		}
		v.Checks = append(v.Checks, Check{
			Step: st.ID, What: contract.UnmetName, Want: false, Got: true, OK: false,
			Note: "the step reported that the goal was not reached",
		})
		v.State = StateFailed
	}
	// 실제로 대조된 조건의 수 — 공허한 참을 막는다 (아래 참조).
	evaluated := 0
	for _, cond := range c.SuccessWhen {
		// 함대 조건 (ADR-058) — 단계가 아니라 그 시점 광고를 본다.
		if cond.IsFleet() {
			evaluated++
			want, min := cond.Want()
			n := 0
			for _, a := range fleet {
				if a.Satisfies(want) {
					n++
				}
			}
			ok := fleet != nil && n >= min
			note := ""
			if fleet == nil {
				note = "the fleet was not observed"
			}
			v.Checks = append(v.Checks, Check{
				Step: "(fleet)", What: describeWant(want), Want: min, Got: n, OK: ok,
				Note: note,
			})
			if !ok {
				v.State = StateFailed
			}
			v.Fleet = fleet
			continue
		}
		res, ran := results[cond.Step]
		// 건너뛴 단계의 조건은 공허하게 참이다 (ADR-022 §7.2)
		//
		// 조건이 묻는 것은 "완주했는가" 인데(ADR-004), 실행 자체가 안 됐으면
		// 그 물음이 성립하지 않는다. 실패로 치면 dispatch 로 경로가 갈릴 때
		// 안 간 쪽의 조건이 Run 을 죽인다 — 계약 저자가 경로별로 조건을
		// 나눠 쓸 방법이 없으므로(경로는 실행 시 정해진다) 이쪽이 유일한 답이다.
		if ran && res.Skipped {
			// 골랐는데도 SKIPPED 면 공허하게 참이 아니다 (ADR-060 §3).
			//
			// 갈림길이 이 단계를 목적지로 골랐다는 것은 "이 경로로 간다" 는
			// 선언이다. 그런데도 안 돌았다면 경로가 갈린 것이 아니라
			// 그 자리에 못 닿은 것이고, 계약이 그 단계에 조건을 건 이상
			// 목표 판정이 통째로 빠진 것이다. 실측이 이것을 밟았다
			// (third-run-1: 목표 단계가 SKIPPED 인데 Run 이 SUCCEEDED 로 봉인됐다).
			//
			// 방향이 한쪽뿐이라 I3 을 안 깬다 — 통과할 Run 을 실패시킬 뿐
			// 실패할 Run 을 통과시키지 못한다. ADR-054 의 _unmet 과 같은 비대칭이다.
			if res.Chosen {
				v.Checks = append(v.Checks, Check{
					Step: cond.Step, What: "skipped", Want: false, Got: true, OK: false,
					Note: "a branch chose this step, yet it never ran; the goal was not judged",
				})
				v.State = StateFailed
				continue
			}
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
			// 재시도 소진이 verdict 로 잡힌다 — 루프는 제어 흐름이고
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
			// 세상이 바뀌었는가 (ADR-037) — produced 와 무게가 다르다.
			// produced 는 에이전트가 쓴 파일이고, 이것은 에이전트가 저작하지 않는
			// 관찰 이다. agent 단계가 아무것도 안 하고 「했다」고 말해도 여기서 걸린다.
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
	// 공허한 참을 막는다
	//
	// 조건이 있는데 하나도 대조되지 않았다 = 전부 건너뛰었다는 뜻이다.
	// 그대로 두면 아무것도 안 하고 SUCCEEDED가 된다.
	// 「검증이 불필요한 패치」도 그 경로에 보고 단계가 있어야 하고,
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
	// 건너뛴 단계도 싣는다 — 결과는 없지만 "실행 안 됐다" 를 Verify 가
	// 알아야 한다. 안 실으면 크래시(결과 없음)와 구분이 안 된다.
	rows, err := s.pool.Query(ctx,
		`SELECT name, result, attempt, state, chosen FROM steps
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
		var chosen bool
		if err := rows.Scan(&name, &raw, &attempt, &state, &chosen); err != nil {
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
		r.Chosen = chosen
		out[name] = r
	}
	return out, rows.Err()
}
