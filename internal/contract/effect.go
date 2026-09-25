package contract

import (
	"fmt"
	"time"
)

// Effect 는 단계가 워크스페이스에 무엇을 하나다 (ADR-075 §5).
//
// actor 와 effect 를 나눈 결정이다 — 누가 수행하나(agent · run)와 무엇을
// 하나(읽나 · 고치나 · 짓나)는 다른 축이다. 오늘까지는 종류가 둘 다 정했고,
// 그래서 노드가 모든 명령 단계의 diff 와 전체 훑기를 거뒀다 (internal/enode/claim.go:691-694).
// effect 가 그 짐작을 계약에 적힌 값으로 바꾼다.
//
// 노드가 결과로 무엇을 거두나를 이것이 정한다 — 그 일은 finalize 유닛이 한다.
type Effect string

const (
	// EffectRead 는 읽기만 한다. 이번 회차에서는 build 와 같이 diff 와 전체
	// 훑기를 안 한다. source 를 쓴 것을 위반으로 잡는 일은 뒤 회차다.
	EffectRead Effect = "read"
	// EffectEdit 는 source 를 고친다. diff 를 거둔다. agent 단계의 기본값이다.
	EffectEdit Effect = "edit"
	// EffectBuild 는 빌드와 시험이다. diff 를 안 거둔다. run 단계의 기본값이다.
	EffectBuild Effect = "build"
	// EffectPrepare 는 굽기의 build 단계 전용이다 (ADR-077 §2).
	// upper 를 버리지 않고 merge 단계로 넘긴다.
	EffectPrepare Effect = "prepare"
)

// Budget 은 명령이 끝난 뒤 구간의 시간 상한 둘이다 (FR-3).
//
// 값은 Go duration 문자열이다 ("90s" · "5m"). 계약은 사람이 쓰는 JSON 이라
// 나노초 정수보다 이쪽이 읽힌다. 위쪽 상한은 두지 않는다 (Application Design Q6) —
// 노드 소유자의 수단은 오늘처럼 drain 이다.
type Budget struct {
	// Finalize 는 결과를 확정하는 구간이다. 기본 1분이고 늘리기만 한다 —
	// 팩(기능 3)이 「계약이 늘릴 수 있다」로 적었다.
	Finalize string `json:"finalize,omitempty"`
	// Upload 는 결과를 올리는 구간이다. 기본 3분이고 0 보다 커야 한다.
	Upload string `json:"upload,omitempty"`
}

// 예산의 기본값이다. 노드(finalize 유닛)도 이 값을 읽는다 — 기본값을 두 곳에
// 적으면 언젠가 한 곳이 어긋난다.
const (
	DefaultFinalizeBudget = time.Minute
	// MinFinalizeBudget 은 finalize 예산의 바닥이다. 기본값과 같다 — 늘리기만 한다.
	MinFinalizeBudget   = DefaultFinalizeBudget
	DefaultUploadBudget = 3 * time.Minute
)

// EffectOrDefault 는 적힌 effect 를, 없으면 종류의 기본값을 준다.
//
//	run   -> build     agent -> edit
//	build -> 적혀 있다 (Validate 가 prepare 를 요구한다)
//	merge · ask · acquire -> "" (effect 가 없는 종류)
//
// 기본값을 메서드가 채우는 것은 Condition.Want 의 min_count 와 같은 이유다 —
// 읽는 쪽이 여럿이다.
func (s Step) EffectOrDefault() Effect {
	if s.Effect != "" {
		return s.Effect
	}
	k, _ := s.Kind()
	switch k {
	case KindRun:
		return EffectBuild
	case KindAgent:
		return EffectEdit
	}
	return ""
}

// Budgets 는 두 예산을 기본값을 채워 준다.
//
// Validate 를 지난 계약에서만 부른다 — 파싱 오류를 다시 다루지 않는다.
// 지나지 않은 계약은 Mediator 에 저장되지 않으므로 노드에 오지 않는다.
func (s Step) Budgets() (finalize, upload time.Duration) {
	finalize, upload = DefaultFinalizeBudget, DefaultUploadBudget
	if s.Budget == nil {
		return finalize, upload
	}
	if d, err := time.ParseDuration(s.Budget.Finalize); err == nil {
		finalize = d
	}
	if d, err := time.ParseDuration(s.Budget.Upload); err == nil {
		upload = d
	}
	return finalize, upload
}

// checkStepFields 는 종류마다 받는 칸과 그 값을 본다 (굽기 회차).
//
// 종류 판별 바로 뒤, 역할 검사 앞에서 부른다 — 그래야 uses 없는 build 단계가
// 「undeclared role ""」 대신 「a build step needs uses」를 받는다.
// 받는 칸을 값보다 먼저 본다. 칸이 틀린 단계에 값의 문구가 나가면 원인이 아닌
// 곳을 가리킨다.
func checkStepFields(s Step, k StepKind) error {
	if err := checkEffect(s, k); err != nil {
		return err
	}
	if s.Budget != nil && k != KindRun && k != KindAgent && k != KindBuild {
		hint := ""
		if k == KindMerge {
			hint = "; merge.wait sets how long it waits"
		}
		return fmt.Errorf("step %q: %s step takes no budget%s", s.ID, aKind(k), hint)
	}
	if s.Discover && k != KindRun && k != KindAgent {
		return fmt.Errorf("step %q: discover is only for a run or agent step", s.ID)
	}
	if s.IR != "" && k != KindBuild {
		return fmt.Errorf("step %q: ir is only for a build step", s.ID)
	}
	switch k {
	case KindBuild:
		if err := checkBuildStep(s); err != nil {
			return err
		}
	case KindMerge:
		return checkMergeStep(s)
	}
	return checkBudget(s)
}

// checkEffect 는 effect 를 종류마다의 표에 댄다.
//
//	run    build (기본) · edit · read
//	agent  edit (기본) · read
//	build  prepare — 반드시 적는다
//	merge · ask · acquire  받지 않는다
func checkEffect(s Step, k StepKind) error {
	switch s.Effect {
	case "", EffectRead, EffectEdit, EffectBuild, EffectPrepare:
	default:
		return fmt.Errorf("step %q: unknown effect %q; use read, edit, build or prepare", s.ID, s.Effect)
	}
	switch k {
	case KindMerge, KindAsk, KindAcquire:
		if s.Effect != "" {
			return fmt.Errorf("step %q: %s step takes no effect", s.ID, aKind(k))
		}
	case KindRun, KindAgent:
		if s.Effect == EffectPrepare {
			return fmt.Errorf("step %q: effect prepare is only for a build step (sync and builds)", s.ID)
		}
		if k == KindAgent && s.Effect == EffectBuild {
			return fmt.Errorf("step %q: effect build is not allowed on an agent step; use read or edit", s.ID)
		}
	case KindBuild:
		// 추론하지 않는다 — upper 를 합칠지를 정하는 값이라 계약에 보여야 한다.
		if s.Effect != EffectPrepare {
			return fmt.Errorf("step %q: a build step must say effect \"prepare\"; it decides "+
				"whether the upper is merged, so it is written, not inferred", s.ID)
		}
	}
	return nil
}

// checkBudget 은 예산의 값을 본다. 받는 종류는 checkStepFields 가 먼저 걸렀다.
//
// 경계 — finalize 는 정확히 1m 이면 받는다. upload 는 1ns 도 받는다.
// "budget": {} 는 두 값 모두 기본값이다.
func checkBudget(s Step) error {
	b := s.Budget
	if b == nil {
		return nil
	}
	if b.Finalize != "" {
		d, err := time.ParseDuration(b.Finalize)
		if err != nil {
			return fmt.Errorf("step %q: budget.finalize %q is not a duration (for example \"90s\" or \"5m\")",
				s.ID, b.Finalize)
		}
		if d < MinFinalizeBudget {
			return fmt.Errorf("step %q: budget.finalize %s is below the default %s; it can only be raised",
				s.ID, b.Finalize, MinFinalizeBudget)
		}
	}
	if b.Upload != "" {
		d, err := time.ParseDuration(b.Upload)
		if err != nil {
			return fmt.Errorf("step %q: budget.upload %q is not a duration (for example \"90s\" or \"5m\")",
				s.ID, b.Upload)
		}
		if d <= 0 {
			return fmt.Errorf("step %q: budget.upload must be greater than zero", s.ID)
		}
	}
	return nil
}

// aKind 는 종류 이름에 영어 관사를 붙인다 — 거절 문구가 "a ask step" 이 되지 않게.
func aKind(k StepKind) string {
	switch k {
	case KindAgent, KindAsk, KindAcquire:
		return "an " + k.String()
	}
	return "a " + k.String()
}
