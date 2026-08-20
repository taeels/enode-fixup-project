// Package contract 는 Run 계약의 타입을 정의한다.
//
// 정본은 enode-design 저장소의 protocol/run-contract.md 이고 이 파일은 그 형태다.
// 의미가 어긋나면 protocol/INVARIANTS.md 가 이긴다.
package contract

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/taeels/enode/internal/schema"
)

// CapabilityAgentReason 은 MVP 의 유일한 capability 다 (ADR-019).
// 노드는 한 종류이고 능력은 속성의 존재로 표현된다.
const CapabilityAgentReason = "agent.reason"

// Contract 는 runctl 이 제출하는 것 전체다.
type Contract struct {
	RunID       string      `json:"run_id"`
	Work        Work        `json:"work"`
	Requires    []Require   `json:"requires"`
	Lease       Lease       `json:"lease"`
	Steps       []Step      `json:"steps"`
	SuccessWhen []Condition `json:"success_when"`
}

// Work 는 이 Run 이 무엇에 대한 것인가를 가리킨다.
// RunID 는 (ChangeID, Patchset) 에서 결정적으로 유도되므로 재제출이 멱등이다.
type Work struct {
	System    string `json:"system"`
	ChangeID  string `json:"change_id"`
	Patchset  int    `json:"patchset"`
	ParentRev string `json:"parent_rev"` // 차분 반증의 기준점
	PatchRev  string `json:"patch_rev"`
}

// Lease 는 점유의 수명이다 (ADR-008 · ADR-010).
// Artifact 는 계약에 쓰는 값이 아니라 Mediator 가 채워 내려보내는 자리다.
type Lease struct {
	TTLSeconds   int `json:"ttl_seconds"`
	RenewSeconds int `json:"renew_seconds"`
}

// Require 는 자원 하나에 대한 요구다.
//
// capability 는 항상 agent.reason 이고 (ADR-019) 실제 구별은 Attrs 가 한다.
// 와이어에서 속성은 as/capability/count 와 ★ 같은 층의 형제 키 ★ 로 온다:
//
//	{ "as": "builder", "capability": "agent.reason",
//	  "arch": "armv7", "repo": "gerrit.corp/kernel/linux" }
//
// 그래서 UnmarshalJSON 이 알려진 키를 뺀 나머지를 Attrs 로 걷는다.
type Require struct {
	As         string
	Capability string
	Count      int // 0 이면 1 로 읽는다
	Attrs      map[string]string
}

// requireKnown 은 속성이 아닌 키다. 여기 없는 키는 전부 매칭 속성이 된다.
var requireKnown = map[string]bool{"as": true, "capability": true, "count": true}

func (r *Require) UnmarshalJSON(b []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*r = Require{Attrs: map[string]string{}}
	for k, v := range raw {
		switch k {
		case "as":
			if err := json.Unmarshal(v, &r.As); err != nil {
				return fmt.Errorf("requires[].as: %w", err)
			}
		case "capability":
			if err := json.Unmarshal(v, &r.Capability); err != nil {
				return fmt.Errorf("requires[].capability: %w", err)
			}
		case "count":
			if err := json.Unmarshal(v, &r.Count); err != nil {
				return fmt.Errorf("requires[].count: %w", err)
			}
		default:
			// 속성 값은 문자열로 고정한다. 라벨 매칭 계열이므로 (ADR-011)
			// 표현식도 범위도 없다 — 속성 일치뿐이다.
			var s string
			if err := json.Unmarshal(v, &s); err != nil {
				return fmt.Errorf("requires[].%s 는 문자열이어야 한다: %w", k, err)
			}
			r.Attrs[k] = s
		}
	}
	return nil
}

func (r Require) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{"as": r.As, "capability": r.Capability}
	if r.Count != 0 {
		m["count"] = r.Count
	}
	for k, v := range r.Attrs {
		if requireKnown[k] {
			return nil, fmt.Errorf("속성 이름이 예약어와 겹친다: %s", k)
		}
		m[k] = v
	}
	return json.Marshal(m)
}

// Wanted 는 이 요구가 몇 개의 노드를 원하는지다. 0 은 1 로 읽는다.
func (r Require) Wanted() int {
	if r.Count <= 0 {
		return 1
	}
	return r.Count
}

// StepKind 는 단계의 두 종류다 (ADR-019 결정 3).
//
// 노드는 한 종류로 합쳤지만 단계는 합치지 않는다. 전부 agent 단계가 되면
// 하네스가 쓸모없는 소리를 하고도 종료코드 0 으로 끝나므로 성패를 기계적으로
// 물을 수 없게 되고, 그러면 ADR-004 가 무너진다.
type StepKind int

const (
	KindUnknown StepKind = iota
	KindAgent            // agent 가 있다 — 하네스를 띄운다. 판정은 produced.
	KindRun              // run 이 있다 — 어댑터가 직접 돌리고 exit_code 를 잰다.
)

func (k StepKind) String() string {
	switch k {
	case KindAgent:
		return "agent"
	case KindRun:
		return "run"
	}
	return "unknown"
}

// Workspace 는 이 단계가 어느 저장소의 어느 리비전 위에 서는가다 (ADR-017).
// 저장소는 옮기지 않는다 — 노드가 이미 갖고 있고 그것이 매칭 조건이었다.
type Workspace struct {
	Repo string `json:"repo"` // canonical id: lower(host)/path
	Rev  string `json:"rev"`
}

// Step 은 계약이 요청 시점에 전부 선언하는 단계 하나다.
// 실행 중 추가되지 않는다 — 「Run 은 하나다」.
type Step struct {
	ID        string     `json:"id"`
	Uses      string     `json:"uses"`
	Workspace *Workspace `json:"workspace,omitempty"`

	// ★ 판별자 두 개 ★ 정확히 하나만 있어야 한다 (ADR-019).
	Agent map[string]interface{} `json:"agent,omitempty"` // 하네스 실행 파라미터
	Run   []string               `json:"run,omitempty"`   // argv 배열. 셸을 거치지 않는다.
	// Env 는 노드 환경에서 ★ 통과시킬 이름 ★ 이다. ★ 값이 아니라 이름이다 ★ —
	// 값을 계약에 적으면 자격증명이 Run Record 에 봉인되어 영구히 남는다.
	//
	// 기본 목록(PATH·HOME·프록시·흔한 빌드 변수)은 노드가 갖고 있고,
	// 여기 적는 것은 ★ 그 위에 더하는 것 ★ 이다. 적지 않은 이름은 안 넘어간다.
	Env []string `json:"env,omitempty"`

	In  map[string]interface{} `json:"in,omitempty"`
	Out []string               `json:"out,omitempty"`

	// Schema 는 산출물 이름 → JSON Schema (ADR-020). ★ 인라인이다 ★ —
	// 경로면 Record 를 열었을 때 무엇으로 검증했는지 알 수 없다 (성질 4).
	Schema map[string]interface{} `json:"schema,omitempty"`

	// 재시도 루프 (ADR-013). 스키마 위반도 이 루프의 입력이 된다.
	ValidateWith string   `json:"validate_with,omitempty"`
	MaxAttempts  int      `json:"max_attempts,omitempty"`
	Feedback     []string `json:"feedback,omitempty"`

	Repeat int `json:"repeat,omitempty"`
}

// Kind 는 단계의 종류를 판별한다.
// 둘 다 있거나 둘 다 없으면 계약이 틀린 것이다 — 암묵을 남기지 않는다.
func (s Step) Kind() (StepKind, error) {
	hasAgent := s.Agent != nil
	hasRun := len(s.Run) > 0
	switch {
	case hasAgent && hasRun:
		return KindUnknown, fmt.Errorf("step %q: agent 와 run 이 둘 다 있다", s.ID)
	case hasAgent:
		return KindAgent, nil
	case hasRun:
		return KindRun, nil
	}
	return KindUnknown, fmt.Errorf("step %q: agent 도 run 도 없다", s.ID)
}

// Condition 은 success_when 의 항목 하나다 (ADR-004).
// 계약은 "완주했는가" 만 묻는다. "결과가 무엇인가" 는 묻지 않는다.
type Condition struct {
	Step     string   `json:"step"`
	Produced []string `json:"produced,omitempty"`
	// ExitCode 는 ★ 명령 단계에만 ★ 쓸 수 있다 (ADR-019).
	// agent 단계에 쓰면 계약이 틀린 것이다 — claude 는 헛소리를 하고도 0 으로 끝난다.
	ExitCode       *int `json:"exit_code,omitempty"`
	WithinAttempts bool `json:"within_attempts,omitempty"`
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

var (
	ErrNoRunID       = errors.New("run_id 가 없다")
	ErrNoRequires    = errors.New("requires 가 비어 있다")
	ErrNoSteps       = errors.New("steps 가 비어 있다")
	ErrUnknownCap    = errors.New("capability 는 agent.reason 하나뿐이다 (ADR-019)")
	ErrDupAs         = errors.New("requires[].as 가 중복이다")
	ErrDupStepID     = errors.New("steps[].id 가 중복이다")
	ErrExitOnAgent   = errors.New("agent 단계에 exit_code 조건을 쓸 수 없다 (ADR-019)")
	ErrCondUnknownID = errors.New("success_when 이 없는 단계를 가리킨다")
)

// Validate 는 계약이 문법적으로 성립하는지만 본다. 400 의 근거다.
// 자원이 있는지(422)와 지금 비어 있는지(409)는 매처가 본다.
func (c Contract) Validate() error {
	if c.RunID == "" {
		return ErrNoRunID
	}
	if len(c.Requires) == 0 {
		return ErrNoRequires
	}
	if len(c.Steps) == 0 {
		return ErrNoSteps
	}

	roles := map[string]bool{}
	for _, r := range c.Requires {
		if r.Capability != CapabilityAgentReason {
			return fmt.Errorf("%w: %q", ErrUnknownCap, r.Capability)
		}
		if r.As == "" {
			return fmt.Errorf("requires[].as 가 비었다")
		}
		if roles[r.As] {
			return fmt.Errorf("%w: %q", ErrDupAs, r.As)
		}
		roles[r.As] = true
	}

	kinds := map[string]StepKind{}
	for _, s := range c.Steps {
		if s.ID == "" {
			return fmt.Errorf("steps[].id 가 비었다")
		}
		if _, dup := kinds[s.ID]; dup {
			return fmt.Errorf("%w: %q", ErrDupStepID, s.ID)
		}
		k, err := s.Kind()
		if err != nil {
			return err
		}
		kinds[s.ID] = k
		if !roles[s.Uses] {
			return fmt.Errorf("step %q 가 없는 역할 %q 를 쓴다", s.ID, s.Uses)
		}
		// ★ ADR-020 의 경계선을 여기서 400 으로 만든다 ★
		// "스키마는 형식만 제약한다" 를 산문으로 두면 새어나가고,
		// 그 순간 ADR-004(기계적 판정만)가 스키마를 통해 무너진다.
		for name, sch := range s.Schema {
			if err := schema.CheckBoundary(sch); err != nil {
				return fmt.Errorf("step %q 의 %s: %w", s.ID, name, err)
			}
			if !contains(s.Out, name) {
				return fmt.Errorf("step %q 가 내지 않는 산출물 %q 에 스키마를 달았다", s.ID, name)
			}
		}
	}

	for _, cond := range c.SuccessWhen {
		k, ok := kinds[cond.Step]
		if !ok {
			return fmt.Errorf("%w: %q", ErrCondUnknownID, cond.Step)
		}
		if cond.ExitCode != nil && k == KindAgent {
			return fmt.Errorf("%w: %q", ErrExitOnAgent, cond.Step)
		}
	}
	return nil
}
