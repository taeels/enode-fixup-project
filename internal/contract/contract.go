// Package contract 는 Run 계약의 타입을 정의한다.
//
// 정본은 enode-design 저장소의 protocol/run-contract.md 이고 이 파일은 그 형태다.
// 의미가 어긋나면 protocol/INVARIANTS.md 가 이긴다.
package contract

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/taeels/enode/internal/schema"
)

// capability 어휘는 ★ 둘 ★ 이다 (ADR-019 · ADR-022 §5.2).
//
// ★ 왜 orchestration 을 속성이 아니라 별도 capability 로 두는가 ★
//
// 매처의 의미론 때문이다 — ★ 속성은 포함(부분집합)이고 capability 는 배제(완전일치) ★:
//
//	Capability.Satisfies:  c.Capability != r.Capability  → 탈락   ★ 배제 ★
//	                       r.Attrs 의 각 키가 c.Attrs 에  → 통과   ★ 포함 ★
//
// 속성으로 표현하면(agent.reason { role: orchestrator }) 평범한 Run 의
// requires: agent.reason 이 ★ 부분집합 매칭으로 통과해 오케스트레이터를 잡아간다 ★.
// 임대 키가 (노드)이므로(ADR-019 결정 2) 그 순간 오케스트레이션이 막힌다.
//
// ★ 두 번째 어휘의 존재 이유가 「매칭되기 위해」가 아니라 「매칭 안 되기 위해」다 ★.
// 그래서 세 번째를 더할 압력이 안 생긴다 — ADR-019 가 막으려던 어휘 팽창과 다르다.
//
// ADR-019 의 기각 사유 둘 다 안 걸린다:
//   - "결정론적 작업을 별도 cap 으로 내면 그냥 원격 실행으로 읽힌다"
//     → orchestration 은 결정론적 작업이 아니다
//   - "어휘를 하나로 줄이면 requires 가 순수 속성 매칭이 된다"
//     → 이름을 하나 더해도 매칭은 완전일치+부분집합 그대로다. 표현식이 안 생긴다.
const (
	// CapabilityAgentReason 은 ★ 일을 하는 능력 ★ 이다 — 함대의 대부분.
	// 노드는 한 종류이고 능력은 속성의 존재로 표현된다.
	CapabilityAgentReason = "agent.reason"
	// CapabilityOrchestration 은 ★ 계약을 짓는 능력 ★ 이다 (ADR-022 §5).
	//
	// ★ 이 노드는 arch 를 광고하지 않는다 ★ — Mediator 머신에 놓이는 것을
	// 전제하는데, INVARIANTS 가 "명령 단계에는 파일시스템 경계가 없다" 이므로
	// 그 기계에서 명령 단계가 돌면 ★ DB·아티팩트·토큰에 무경계 argv 가 닿는다 ★.
	// 광고를 좁히는 것으로 닫는다 (detect.go).
	CapabilityOrchestration = "orchestration"
)

// knownCapability 는 계약이 요구할 수 있는 어휘다.
// ★ 열린 어휘가 아니다 ★ — 모르는 이름은 422 로 떨어져야 계약 저자가
// 오타를 즉시 안다 (ADR-014 결정 3 의 영구 거절).
func knownCapability(name string) bool {
	return name == CapabilityAgentReason || name == CapabilityOrchestration
}

// Contract 는 runctl 이 제출하는 것 전체다.
type Contract struct {
	RunID       string      `json:"run_id"`
	Work        Work        `json:"work"`
	Requires    []Require   `json:"requires"`
	Lease       Lease       `json:"lease"`
	Steps       []Step      `json:"steps"`
	SuccessWhen []Condition `json:"success_when"`

	// Ledger 는 ★ 이 Run 이 무엇을 발견할 수 있는가 ★ 다 (ADR-023 §6).
	// ★ 없으면 scope:"run" 과 같다 ★ = 오늘 그대로.
	Ledger *Ledger `json:"ledger,omitempty"`
}

// See 는 한 단계의 시야를 줄인다 (ADR-023 §6.4 자리 3).
type See struct {
	// Ledger 는 ★ 원장 목록을 이 단계에 심을 것인가 ★ 다.
	//
	//	"" · "none"  안 심는다 (기본. ★ 오늘 동작 ★)
	//	"list"       그 시점 ★ 목록 ★ 을 심는다 — ★ 본문이 아니다 ★
	//
	// 목록만 심는 이유는 §6.3 이다: 원장 전체를 하네스에 깔면 컨텍스트가 터지고
	// 비용이 든다. 본문이 필요하면 계약이 in.from 에 이름을 적는다.
	Ledger string `json:"ledger,omitempty"`
	// From 은 ★ 자리다. 오늘은 값이 오면 400 이다 ★.
	//
	// in.from 중 일부만 $IN 에 까는 ★ 비용·권한 경계 ★ 인데, 오늘은 계약 저자가
	// in.from 을 직접 줄이면 되므로 값이 없다. ★ 값이 생기는 시점은 계획을
	// 기계가 쓸 때 ★ 다 — 좁히기는 비용 결정이라 계획을 짓는 쪽이 아는 것이
	// 자연스럽고, 그러면 시야가 계획 위임 위에 앉는다 (ADR-023 §12 의 열린 질문).
	// ★ 조용히 무시하지 않는다 ★ — 무시하면 계약 저자는 좁혀진 줄 안다.
	From []string `json:"from,omitempty"`
}

const (
	SeeNone = "none"
	SeeList = "list"
)

// Ledger 는 시야의 ★ 발견 ★ 축이다 (ADR-023 §6.2).
//
// ★ 시야를 순서에서 유도하지 않는다 ★ — 원장을 두고 정책이 줄인다. 셋을 가른다:
//
//	① ★ 발견 ★  원장     "지금 무엇이 ★ 있는가 ★"
//	② ★ 주입 ★  in.from  "그중 무엇을 ★ $IN 에 깔 것인가 ★"
//	③ ★ 축소 ★  see      "깔리는 것을 ★ 더 줄인다 ★"
//
// 초안 둘을 폐기하고 여기 왔다 — "leaf 는 볼 게 없다" 와 "시야가 순서 그래프를
// 오염시킨다" 는 ★ 전부 「유도」가 만든 결함 ★ 이라 원인째 사라졌다.
type Ledger struct {
	// Scope 는 원장이 ★ 어디까지 ★ 보이는가다.
	//
	//	""  · "run"   이 Run 이 낸 것 (기본. 오늘 동작)
	//	"work"        ★ 같은 Work 의 이전 Run 들이 낸 것까지 ★
	//	              patchset 3 의 리뷰가 patchset 2 의 산출물을 본다
	//	              ⇒ "지난번에 이 지적을 했는데 안 고쳤다" 가 성립한다
	//
	// ★ 모르는 값은 400 이다 ★ — 미구현을 조용히 무시하지 않는다 (ADR-013).
	// ★ enum 을 코드에 박지 않는다 ★ — "오늘은 둘이다" 는 관찰이지 제약이 아니다.
	// 세 번째(Work 를 넘는 축적)의 여는 조건은 ADR-023 §6.5.2 에 있다.
	Scope string `json:"scope,omitempty"`
}

// LedgerScope 는 오늘 아는 값들이다. 아래 둘뿐이지만 ★ 하한이 아니라 현재 상태다 ★.
const (
	ScopeRun  = "run"
	ScopeWork = "work"
)

func knownScope(v string) bool {
	return v == "" || v == ScopeRun || v == ScopeWork
}

// Work 는 이 Run 이 무엇에 대한 것인가를 가리킨다.
// RunID 는 (ChangeID, Patchset) 에서 결정적으로 유도되므로 재제출이 멱등이다.
//
// ★ 이 객체는 둘을 함께 든다 ★ (ADR-023 §6.5.2)
//
//	★ 키 ★         (system, change_id)                 ← Run 을 ★ 넘어 산다 ★
//	★ 이 Run 의 지점 ★ patchset · parent_rev · patch_rev  ← 이번 실행이 가리키는 곳
//
// 가르지 않으면 ★ 동일성을 물을 자리가 없다 ★ — patchset 이 안에 있으니 통째로
// 키로 쓰면 Work 가 Run 과 1:1 이 되어 execution-model §2.2 의 Work 1:N Run 과
// 어긋난다. 그것이 "patchset 2 와 3 은 같은 Work 인가" 에 답이 없던 이유다.
type Work struct {
	// ID 는 ★ Work 의 키 ★ 다. ★ 안 적으면 (system, change_id) 에서 유도한다 ★
	// = 오늘 그대로. 적는 경우는 그 시스템의 변경 단위가 change_id 와 다를 때다.
	ID        *WorkID `json:"id,omitempty"`
	System    string  `json:"system"`
	ChangeID  string  `json:"change_id"`
	Patchset  int     `json:"patchset"`
	ParentRev string  `json:"parent_rev"` // 차분 반증의 기준점
	PatchRev  string  `json:"patch_rev"`
}

// WorkID 는 ★ 동일성을 우리가 정의하지 않는다 ★ 는 결정의 형태다 (ADR-023 §6.5.2).
//
// 우리는 ★ 그 시스템이 「하나의 변경」이라 부르는 것의 식별자를 받아 적는다 ★.
// Gerrit 에서 그 단위는 ★ change number ★ 이고 Change-Id 해시가 아니다 —
// 해시는 여러 브랜치의 change 에 걸치므로 키가 아니다. cherry-pick 과 rebase 에서
// change number 는 유지되고, abandon 후 새로 올리면 새 번호 = ★ 다른 Work ★ 다.
//
// ⇒ ★ patchset 2 와 3 은 같은 Work 다 ★. 도메인이 Gerrit 밖으로 넓어져도
// 규칙이 안 바뀐다 — system 마다 그 시스템의 변경 단위를 받으면 된다.
type WorkID struct {
	System   string `json:"system"`
	ChangeID string `json:"change_id"`
}

// Key 는 이 Run 이 속한 Work 의 키다. ★ 없으면 유도한다 ★.
//
// 문자열 하나로 내는 이유는 이것이 ★ 인덱스이자 필터 ★ 이기 때문이다 —
// 두 조각을 들고 다니면 조회하는 쪽마다 합치는 규칙을 알아야 한다.
func (w Work) Key() string {
	id := WorkID{System: w.System, ChangeID: w.ChangeID}
	if w.ID != nil {
		id = *w.ID
	}
	if id.System == "" && id.ChangeID == "" {
		return ""
	}
	return id.System + ":" + id.ChangeID
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
// Dispatch 는 한 단계가 고를 수 있는 갈림길이다 (ADR-022 §7.2).
//
//	"out": ["route"],
//	"schema": { "route": { … "next": { "enum": ["full","quick","skip"] } … } },
//	"dispatch": { "from": "route.next", "to": ["full","quick","skip"] }
//
// ★ 왜 to 를 계약이 적는가 ★ — 적지 않으면 ★ 아무도 형제를 모른다 ★.
// 값만 있으면 "무엇을 골랐나" 는 알아도 "무엇을 안 골랐나" 를 알 수 없고,
// 그러면 안 간 쪽을 SKIPPED 로 못 만든다. 그리고 to 가 있어야
// ★ 검증기가 DAG 를 정적으로 확인 ★ 해서 종료를 보장할 수 있다.
//
// ★ 값은 스키마가 한 번 더 막는다 ★ — enum 을 벗어나면 PUT blob 이 422 고
// 저장되지 않아 produced 가 불만족이 된다 (ADR-020). 즉 ★ 검사가 두 겹 ★ 이다.
//
// ★ 지금은 목적지가 단계 하나다 ★ — 구간(여러 단계)으로 갈리는 분기는
// P2 의 block(repeat.body) 위에 앉는다. 그때 to 가 블록을 가리키면 된다.
type Dispatch struct {
	// From 은 산출물 안의 경로다 — "<out 이름>.<필드>[.<필드>…]".
	From string `json:"from"`
	// To 는 가능한 목적지 단계 id 들이다. ★ 전부 이 단계보다 뒤여야 한다 ★.
	To []string `json:"to"`
}

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
	// Collect 는 「산출물 이름 → ★ 워크스페이스 상대경로 ★」 다.
	//
	// $OUT 이 유일한 보편 수거 채널이므로, 관측을 정교하게 하는 것보다
	// ★ 선언을 쉽게 만드는 편이 낫다 ★. enode 가 그 경로를 $OUT 으로 옮긴다.
	// ★ 절대경로 · .. 탈출 · 심링크는 거부된다 ★ — 계약이 노드의 경계를
	// 우회하면 안 되고, 계약은 노드 주인이 아닌 사람이 낸다.
	Collect map[string]string `json:"collect,omitempty"`

	In  map[string]interface{} `json:"in,omitempty"`
	Out []string               `json:"out,omitempty"`

	// Schema 는 산출물 이름 → JSON Schema (ADR-020). ★ 인라인이다 ★ —
	// 경로면 Record 를 열었을 때 무엇으로 검증했는지 알 수 없다 (성질 4).
	Schema map[string]interface{} `json:"schema,omitempty"`

	// Expands 는 ★ 이 단계가 steps 를 늘린다 ★ 는 표시다 (ADR-022 §6.3 갈래 A).
	//
	// 계획을 짓는 것은 에이전트이고, 지어진 계약을 제출하는 것은 ★ enode ★ 다
	// (§6.2). 에이전트가 Mediator 에 직접 내면 토큰을 줘야 하고 그러면 R1 이
	// 막은 구멍이 다시 뚫린다 — ★ 에이전트는 결재 문서를 쓰고 도장은 enode 가 찍는다 ★.
	// 덤으로 파일이므로 ★ 스키마 검증을 받는다 ★ (ADR-020): 형태가 틀린 계약은
	// PUT blob 이 422 로 거절해 저장되지 않고, produced 가 불만족이 되어 그 단계가 실패한다.
	//
	// ★ 산출물의 형태는 { "steps": [ … ] } 다 ★ — 배열을 그대로 두지 않고 객체로
	// 감싼다. dispatch.from 이 "route.next" 로 객체 경로를 쓰는 것과 같은 결이고,
	// 나중에 계획에 다른 것(근거·비용)을 붙일 자리가 남는다.
	//
	// ★ 한 번만·한 단계만 ★ — 계약 하나에 expands 단계는 최대 하나이고, 그 단계는
	// 한 번만 늘린다. 여러 번은 ⑥재계획(P6)이고 그때는 깊이 상한이 따라온다.
	// ⇒ ★ 종료는 여전히 정적으로 보장된다 ★ — 늘어난 계약도 append 시점에
	//   DAG 검사를 다시 받고, 늘어나는 횟수가 유계다.
	Expands bool `json:"expands,omitempty"`

	// Needs 는 ★ 이 단계가 기다리는 단계들 ★ 이다 (ADR-023 §4).
	//
	// 오늘까지 의존은 ★ 목록에서의 위치 ★ 였다 — steps[] 가 리스트이므로
	// 리스트가 전순서를 주고, 그래서 ★ 한 번에 하나만 돌았다 ★. needs 는 그
	// 전순서를 ★ 선언된 간선 ★ 으로 바꾼다. 표현이 늘지 않는다 — steps[] 는
	// 평평한 목록 그대로이고 관계만 는다.
	//
	// ★ 없으면 [직전 단계] 다 ★ — 그래서 안 적은 계약은 오늘과 똑같이 돈다.
	// 기본값은 ★ NeedsOf 가 채운다 ★. 게이트는 ★ 한 형태만 알면 된다 ★:
	// 필드가 있는 경우와 없는 경우로 코드가 갈리지 않는다.
	//
	// ★ nil 과 빈 배열이 다르다 ★
	//	nil (필드 없음)  → [직전 단계]. 오늘 그대로
	//	[]  (빈 배열)    → ★ 아무것도 안 기다린다 ★ = 병렬 시작점
	// 그래서 첫 단계가 아니어도 시작점이 될 수 있고, 그것이 폭을 여는 방법이다.
	//
	// ★ 자기보다 앞선 seq 만 가리킨다 ★ (ADR-023 §4.3) — DAG 검사가 한 번
	// 훑는 것으로 끝나고 종료가 제출 시점에 정적으로 보장된다. 뒤로 가야 하는
	// 것은 의존이 아니라 ★ 반복 ★ 이고 그건 repeat 의 자리다 (dispatch 와 같다).
	Needs []string `json:"needs,omitempty"`

	// Release 는 ★ 여기서 놓는 역할들 ★ 이다 (ADR-022 §7.4).
	//
	// 분기가 생기면 어느 경로로 갈지 모르므로 ★ 모든 경로의 자원을 잡아야 한다 ★(I5).
	// 문제는 안 쓰는 노드를 Run 내내 묶는 것이고, 경로가 확정된 뒤 놓으면 풀린다.
	// ★ 계획 위임이 이 조건을 더 강하게 만든다 ★ — 계획을 기계가 지으면 사람은
	// ★ 무엇이 쓰일지 모른 채 ★ requires 를 선언하므로 넉넉히 잡게 된다.
	//
	// ★ 새 개념이 아니다 ★ — 임대는 원래 not_after 로 만료되고(ADR-008),
	// 조기 해제는 그 특수 경우다. 전달 경로도 ADR-016 의 「임대 목록에서 빠진다」가
	// 이미 갖고 있다 — 노드는 다음 하트비트에서 그 임대가 없어진 것을 본다.
	//
	// ★ 되돌릴 수 없다 ★ — 놓은 것은 남이 채간다. 그래서 검증이
	// ★ 그 역할을 쓰는 모든 단계가 이 단계의 조상일 것 ★ 을 요구한다(§4.3).
	Release []string `json:"release,omitempty"`

	// See 는 ★ 이 단계가 무엇을 볼지 ★ 다 — 시야의 ★ 축소 ★ 축 (ADR-023 §6.4).
	//
	// ★ 순수 축소다 ★ — see 로는 원장 밖을 못 본다. 그것이 검증 조건이다.
	// 넓히는 것은 ledger.scope 가 하고, 이쪽은 ★ 줄이기만 ★ 한다.
	// ★ 없으면 목록을 안 심는다 ★ = 오늘 그대로.
	See *See `json:"see,omitempty"`

	// Dispatch 는 ★ 분기다 — 식을 평가하지 않고 이름을 고른다 ★ (ADR-022 §7.2).
	//
	// 기각한 것은 「결과가 X 면 A」 라는 ★ 표현식 ★ 이지 분기 자체가 아니다.
	// 에이전트가 이름 하나를 산출물로 내고 Mediator 는 ★ 그 이름의 단계를
	// 찾아 실행할 뿐 ★ 이므로, ★ Mediator 는 여전히 아무것도 판정하지 않는다 ★
	// (ADR-004). ADR-011 의 라벨 매칭 계열과 같은 정신 — 표현식이 아니라 값 일치다.
	Dispatch *Dispatch `json:"dispatch,omitempty"`

	// 재시도 루프 (ADR-013). 스키마 위반도 이 루프의 입력이 된다.
	ValidateWith string   `json:"validate_with,omitempty"`
	MaxAttempts  int      `json:"max_attempts,omitempty"`
	Feedback     []string `json:"feedback,omitempty"`

	Repeat int `json:"repeat,omitempty"`
}

// ancestors 는 i 번째 단계보다 ★ 확실히 앞서는 ★ 단계들이다 — needs 간선을
// 거슬러 올라가 닿는 것 전부. 부분순서이므로 ★ 앞도 뒤도 아닌 단계가 있다 ★:
// 그것들은 동시에 돌 수 있고, 그래서 여기 안 들어온다.
func ancestors(steps []Step, i int) map[int]bool {
	index := map[string]int{}
	for k, st := range steps {
		index[st.ID] = k
	}
	seen := map[int]bool{}
	var walk func(int)
	walk = func(n int) {
		for _, name := range NeedsOf(steps, n) {
			j, ok := index[name]
			if !ok || seen[j] {
				continue
			}
			seen[j] = true
			walk(j)
		}
	}
	walk(i)
	return seen
}

// inFrom 은 in 의 from 목록을 꺼낸다. In 이 자유 형식 맵이라(프롬프트·참조 문법이
// 함께 산다) 타입으로 못 받고 여기서 읽는다.
func inFrom(in map[string]interface{}) []string {
	raw, ok := in["from"]
	if !ok {
		return nil
	}
	list, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, v := range list {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// Scope 는 이 계약의 원장 범위다. ★ 없으면 run ★ = 오늘 그대로.
func (c Contract) Scope() string {
	if c.Ledger == nil || c.Ledger.Scope == "" {
		return ScopeRun
	}
	return c.Ledger.Scope
}

// NeedsOf 는 steps 의 i 번째(0 부터) 단계가 기다리는 단계 이름들이다.
//
// ★ 기본값을 여기 한 곳에서 채운다 ★ — 계약을 읽는 쪽이 여럿이므로
// "필드가 없으면 직전 단계" 를 각자 알게 두면 언젠가 한 곳이 어긋난다.
// CreateRun 이 이 값을 steps 행에 박고, 그 뒤로는 ★ DB 가 한 형태만 든다 ★.
//
// ★ 계약 전문은 안 바꾼다 ★ — 정규화 결과를 계약에 되쓰면 봉인된 manifest 가
// 제출 전문이 아니게 되어 ADR-005 성질 4(이 묶음만 열면 무엇을 요청했는지 안다)가
// 깨진다. steps 행은 계약에서 유도된 파생이므로 거기 박는 것이 맞는 자리다.
func NeedsOf(steps []Step, i int) []string {
	if i < 0 || i >= len(steps) {
		return nil
	}
	if n := steps[i].Needs; n != nil {
		return n // ★ 빈 배열도 선언이다 ★ — "아무것도 안 기다린다"
	}
	// ★ 분기 목적지는 형제다 ★ — dispatch.to 에 이름이 올라 있다는 것 자체가
	// 이미 선언이므로, 기본값은 직전 단계가 아니라 ★ 분기를 낸 단계 ★ 다.
	//
	// ★ 이것을 빠뜨리면 형제가 사슬로 이어진다 ★ — to: ["full","quick"] 에서
	// quick 의 기본값이 [full] 이 되고, full 이 SKIPPED 가 되는 순간 전파가
	// ★ 살아 있어야 할 가지까지 죽인다 ★. 순서 의미로도 틀리다 — full 다음에
	// quick 이 오는 것이 아니다.
	for j := 0; j < i; j++ {
		d := steps[j].Dispatch
		if d == nil {
			continue
		}
		for _, t := range d.To {
			if t == steps[i].ID {
				return []string{steps[j].ID}
			}
		}
	}
	if i == 0 {
		return []string{}
	}
	return []string{steps[i-1].ID}
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
		if !knownCapability(r.Capability) {
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

	// ★ dispatch 는 단계 목록이 다 모인 뒤에 본다 ★ — 뒤를 가리킬 수 있어야 하므로.
	index := map[string]int{}
	for i, st := range c.Steps {
		index[st.ID] = i
	}
	// ★ 밑줄로 시작하는 산출물 이름은 예약이다 ★ (ADR-023 §6.3.1).
	// $IN 에는 계약이 적은 이름들이 깔리는데 원장 목록도 거기 파일로 간다.
	// 이름 공간을 안 가르면 ★ 계약이 _ledger.json 을 내는 순간 조용히 덮인다 ★.
	// 검증으로 막는 쪽을 고른다 — 런타임에 이름을 바꾸면 계약 저자가 모른다.
	for _, st := range c.Steps {
		for _, o := range st.Out {
			if strings.HasPrefix(o, "_") {
				return fmt.Errorf("step %q: 산출물 이름 %q — "+
					"밑줄로 시작하는 이름은 예약이다 ($IN 의 이름 공간)", st.ID, o)
			}
		}
	}

	// ★ in.from 의 정적 검사 — 오타를 막는 것이 전부다 ★ (ADR-023 §6.2.1).
	//
	// ★ 조상인지 아닌지는 안 본다 ★ — dispatch 와 병렬 때문에 어차피 실행 시에
	// 정해지고, 순서 안전은 ★ "그 시점 원장에 없으면 안 깔린다" ★ 로 흡수된다.
	// 여기서 보는 것은 ★ 그 이름을 내는 단계가 계약에 있는가 ★ 뿐이다.
	// 안 보면 계약 저자가 영구 거절 대신 "산출물을 못 받았다" 를 실행 중에 본다.
	produced := map[string]bool{}
	for _, st := range c.Steps {
		for _, o := range st.Out {
			produced[o] = true
		}
	}
	for _, st := range c.Steps {
		for _, name := range inFrom(st.In) {
			if !produced[name] {
				return fmt.Errorf("step %q: in.from 이 아무도 내지 않는 %q 를 가리킨다",
					st.ID, name)
			}
		}
	}

	for _, st := range c.Steps {
		if st.See == nil {
			continue
		}
		switch st.See.Ledger {
		case "", SeeNone, SeeList:
		default:
			return fmt.Errorf("step %q: see.ledger %q 를 모른다 — 아는 것은 %q 와 %q 다",
				st.ID, st.See.Ledger, SeeNone, SeeList)
		}
		if len(st.See.From) > 0 {
			return fmt.Errorf("step %q: see.from 은 아직 없다 — "+
				"오늘은 in.from 을 직접 줄인다 (ADR-023 §6.4)", st.ID)
		}
	}

	if c.Ledger != nil && !knownScope(c.Ledger.Scope) {
		return fmt.Errorf("ledger.scope %q 를 모른다 — 아는 것은 %q 와 %q 다",
			c.Ledger.Scope, ScopeRun, ScopeWork)
	}

	// ★ expands 는 한 단계만 ★ (ADR-022 §6.3) — 「Run 은 하나다」의 완화를
	// 여기서 유계로 묶는다. 임의 확장이 아니라 ★ 한 단계가 한 번 ★ 이다.
	expander := ""
	for _, st := range c.Steps {
		if !st.Expands {
			continue
		}
		if expander != "" {
			return fmt.Errorf("step %q: expands 가 이미 %q 에 있다 — "+
				"계약 하나에 하나뿐이다 (여러 번은 재계획이고 깊이 상한이 따라온다)",
				st.ID, expander)
		}
		expander = st.ID
		// ★ 계획은 산출물이다 ★ — 이름이 없으면 무엇을 읽어야 할지 모른다.
		if len(st.Out) != 1 {
			return fmt.Errorf("step %q: expands 단계는 산출물 이름이 정확히 하나여야 한다", st.ID)
		}
		// ★ 스키마가 없으면 무엇이든 계약으로 들어온다 ★ — 형태 검증이
		// PUT blob 에서 걸리게 하려면 계약이 스키마를 들고 있어야 한다 (ADR-020).
		if _, ok := st.Schema[st.Out[0]]; !ok {
			return fmt.Errorf("step %q: expands 단계의 산출물 %q 에 스키마가 없다 — "+
				"형태가 틀린 계약이 함대로 들어온다", st.ID, st.Out[0])
		}
	}

	// ★ release 는 부분순서 위에서 검사한다 ★ (ADR-022 §7.4 + ADR-023).
	//
	// ★ 폭이 열리면서 조건이 강해졌다 ★ — 순차였다면 "뒤에서 안 쓰면 된다" 로
	// 족했지만, 병렬에서는 ★ 순서가 정해지지 않은 단계가 동시에 돌 수 있다 ★.
	// 그 단계가 놓아버린 역할을 쓰면 실행 중에 임대가 사라진다.
	// ⇒ ★ 그 역할을 쓰는 모든 단계가 놓는 단계의 조상이어야 한다 ★.
	//   조상이면 이미 끝났고(게이트가 보장한다), 그래야 놓는 것이 안전하다.
	for i, st := range c.Steps {
		if len(st.Release) == 0 {
			continue
		}
		anc := ancestors(c.Steps, i)
		seen := map[string]bool{}
		for _, role := range st.Release {
			if seen[role] {
				return fmt.Errorf("step %q: release 에 %q 가 두 번 있다", st.ID, role)
			}
			seen[role] = true
			if !roles[role] {
				return fmt.Errorf("step %q: 없는 역할 %q 를 놓는다", st.ID, role)
			}
			for j, other := range c.Steps {
				if other.Uses != role || j == i {
					continue
				}
				if !anc[j] {
					return fmt.Errorf("step %q: %q 를 놓는데 step %q 가 그것을 쓴다 — "+
						"놓는 단계보다 ★ 앞선다는 보장이 없다 ★ (되돌릴 수 없으므로 거절한다)",
						st.ID, role, other.ID)
				}
			}
		}
	}

	// ★ needs 도 목록이 다 모인 뒤에 본다 ★ (ADR-023 §4.3).
	// 검사는 dispatch 의 j <= i 와 ★ 같은 모양 ★ 이고 같은 것을 지킨다 —
	// 간선이 전부 뒤를 향하면 그래프가 DAG 이므로 ★ 종료가 제출 시점에 보장된다 ★.
	for i, st := range c.Steps {
		seen := map[string]bool{}
		for _, n := range st.Needs {
			if seen[n] {
				return fmt.Errorf("step %q: needs 에 %q 가 두 번 있다", st.ID, n)
			}
			seen[n] = true
			j, ok := index[n]
			if !ok {
				return fmt.Errorf("step %q: needs 가 없는 단계 %q 를 가리킨다", st.ID, n)
			}
			// ★ 뒤로 못 간다 ★ — 자기 자신도 여기서 걸린다 (j == i).
			// 뒤로 가야 하는 것은 의존이 아니라 반복이고 그건 repeat 의 자리다.
			if j >= i {
				return fmt.Errorf("step %q: needs 의 %q 가 자기보다 뒤다 — "+
					"의존은 뒤로 못 간다 (뒤로 가야 하면 repeat 다)", st.ID, n)
			}
		}
	}

	for i, st := range c.Steps {
		d := st.Dispatch
		if d == nil {
			continue
		}
		if d.From == "" {
			return fmt.Errorf("step %q: dispatch.from 이 비었다", st.ID)
		}
		// from 의 첫 조각은 ★ 이 단계가 실제로 내는 산출물 ★ 이어야 한다.
		// 아니면 실행 시에 "고를 값이 없다" 로 조용히 죽는다.
		blob, _, _ := strings.Cut(d.From, ".")
		if !contains(st.Out, blob) {
			return fmt.Errorf("step %q: dispatch.from 이 내지 않는 산출물 %q 를 가리킨다",
				st.ID, blob)
		}
		// ★ 갈림길이 하나면 갈림길이 아니다 ★ — 순차로 쓰면 될 것을
		// 분기로 쓰면 읽는 사람이 경로가 갈린다고 오해한다.
		if len(d.To) < 2 {
			return fmt.Errorf("step %q: dispatch.to 가 둘 미만이다", st.ID)
		}
		seen := map[string]bool{}
		for _, t := range d.To {
			if seen[t] {
				return fmt.Errorf("step %q: dispatch.to 에 %q 가 두 번 있다", st.ID, t)
			}
			seen[t] = true
			j, ok := index[t]
			if !ok {
				return fmt.Errorf("step %q: dispatch.to 가 없는 단계 %q 를 가리킨다", st.ID, t)
			}
			// ★ 뒤로 못 간다 = DAG = 종료가 정적으로 보장된다 ★ (ADR-022 §7.2).
			// 뒤로 가야 하는 것은 분기가 아니라 ★ 반복 ★ 이고 그건 repeat 의 자리다.
			if j <= i {
				return fmt.Errorf("step %q: dispatch.to 의 %q 가 자기보다 앞이다 — "+
					"분기는 뒤로 못 간다 (뒤로 가야 하면 repeat 다)", st.ID, t)
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
