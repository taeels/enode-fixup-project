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
	"time"

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

// UnmetName 은 ★ 목표에 못 닿았다고 말하는 자리 ★ 다 (ADR-054).
//
// ★ _cannot 과 다르다 ★ (ADR-038):
//
//	_cannot   ★ 이 단계 ★ 를 못 하겠다 — 단계가 실패하고 재시도가 남는다
//	_unmet    ★ 이 Run 이 목표에 못 닿았다 ★ — 더 해도 소용없다. Run 이 실패한다
//
// ★ 왜 필요한가 ★ — success_when 은 기계가 볼 수 있는 것만 본다: 종료코드 ·
// 파일의 존재 · 경로의 변경. 그것이 참인데 목표는 아닐 수 있다.
// "아무것도 안 깔려 있다" 를 적은 보고서도 ★ 존재하는 파일 ★ 이다.
//
//	★ 실측 ★ (vm-scratch-5) vm_node_up 이 exit 0 · produced[vm_caps] 로 두
//	조건을 다 만족했다. vm_caps 본문은 "enode binary in vm: exit status 1" 이었고
//	함대에 VM 노드는 없었다. ★ 그런데 재계획 에이전트는 그것을 읽고 목표
//	미달로 옳게 판단했다 ★ — 판정하는 쪽이 에이전트보다 둔했다.
//
// ★ 방향이 한쪽뿐이라 ADR-037 을 안 깬다 ★ — 통과할 Run 을 실패시킬 수는
// 있고, 실패할 Run 을 통과시킬 수는 ★ 없다 ★. 기준의 저자는 여전히 사람이다.
// ADR-038 이 자백을 믿은 것과 같은 비대칭이다: ★ 자기에게 불리한 신고다 ★.
//
// ★ 계획을 짓는 단계만 낼 수 있다 ★ — 목표를 판단하려면 전체 그림이 필요하고
// 그 그림은 expands 단계에만 실린다 (goal · owed · standing).
const UnmetName = "_unmet"

// Acquire 는 실행 중 획득 하나다 (ADR-022 §7.5 · ADR-024).
//
// ★ 이것 자체가 분기다 ★ — 획득 결과는 산출물이 아니라 ★ 즉시 아는 값 ★ 이라
// dispatch 를 한 겹 거칠 이유가 없다. §7.5 의 예시가 이 형태였다:
//
//	"acquire": { "want": {…}, "acquired": "on_board", "unavailable": "qemu_only" }
//
// ★ 그래서 분기의 규칙을 그대로 쓴다 ★ — 목적지는 실존해야 하고, 서로 달라야
// 하며, ★ 전부 자기보다 뒤 ★ 여야 한다(종료가 정적으로 보장된다). 안 간 쪽은
// SKIPPED 가 되고 간선을 따라 전파된다.
type Acquire struct {
	// Want 는 ★ requires 의 한 항목과 형태가 같다 ★ — 새 파서를 안 만들고
	// 같은 매처가 돈다 (ADR-014 결정 3: ★ 매처를 두 벌 만들지 않는다 ★).
	//
	// ★ 오늘은 요청 하나에 자원 하나다 ★ — 그래서 "전부 아니면 전무" 가 자명하다.
	// 여럿을 한 요청으로 잡는 형태(count)는 자리를 막지 않되 지금 안 연다:
	// 그때는 부분 점유를 남기지 않는 롤백이 필요하고 그것이 §7.5 가 말한 무게다.
	Want *Require `json:"want"`
	// Acquired · Unavailable 은 ★ 목적지 단계 ★ 다.
	// ★ 실패가 중단이 아니라 값이 되는 자리 ★ 이고, 그래서 획득 실패로
	// Run 이 죽지 않는다 — 다른 경로로 간다.
	Acquired    string `json:"acquired"`
	Unavailable string `json:"unavailable"`
}

// Ask 는 되묻기 하나다 (ADR-032).
type Ask struct {
	// Prompt 는 사람이 읽는 질문이다. 필드의 형태는 단계의 schema 가 정한다.
	Prompt string `json:"prompt"`
	// Answerers 는 ★ 답할 수 있는 사람 ★ 이다 — 선언은 스텝에, 집행은 서버에.
	// 비었으면 아무나 답한다 (오늘 신뢰 경계가 하나라 감수한다).
	// ★ 오늘 principal 은 식별이지 인증이 아니다 ★ (ADR-015 §1) — 웹 표면이
	// 생기면 인증층과 함께 이 자리를 다시 본다.
	Answerers []string `json:"answerers,omitempty"`
	// Timeout 은 선택이다. ★ 없으면 무한 대기 ★ — 사람의 시간을 시스템이
	// 짐작하지 않는다. 선언하면 행동(then)도 함께 선언한다.
	Timeout *AskTimeout `json:"timeout,omitempty"`

	// Show 는 ★ 질문과 함께 보여줄 산출물 ★ 이다 (ADR-032 §4 보강).
	//
	// prompt 는 계약 시점의 문자열이라, 에이전트가 실행 중에 만든 질문 내용
	// (예: "이 락 순서가 의도된 겁니까?" 를 담은 blob)을 사람이 보려면
	// 따로 열어야 했다. ★ 보지 않고 답하게 만들면 안 된다 ★ — 인박스가
	// 여기 적힌 산출물의 내용을 함께 든다 (큰 것은 잘라서, 잘렸음을 표시).
	Show []string `json:"show,omitempty"`

	// Adopts 는 ★ 목표 위임의 승인 지점 ★ 이다 (ADR-033).
	//
	// expands 단계 하나를 지목한다. 그 계획이 ★ 제안한 success_when ★ 은
	// 효력이 없다가, 이 ask 의 답이 verdict:"approve" 일 때 ★ 계약의 열에
	// 새 판으로 채택 ★ 된다 (by: "answer:<이 단계>").
	//
	// ★ 저자와 승인자를 가른다 ★ — 판정 기준의 저자는 기계(계획)일 수 있으나
	// 그것이 효력을 얻는 유일한 길은 ★ 목표를 준 사람의 답 ★ 이다.
	// 그래야 「판정 기준을 판정 대상이 정한다」가 안 된다 (ADR-004 개정).
	Adopts string `json:"adopts,omitempty"`
}

// AskTimeout 은 기한과 그때의 행동이다.
type AskTimeout struct {
	After string `json:"after"` // Go duration ("72h")
	// Then 은 ★ 오늘 "fail" 뿐 ★ 이다. "default"(기본 답으로 진행)와
	// "escalate"(알림 후 계속 대기)는 순연 — 여는 조건은 ADR-032 §2.
	Then string `json:"then"`
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
				return fmt.Errorf("requires[].%s must be a string: %w", k, err)
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
			return nil, fmt.Errorf("attribute name is reserved: %s", k)
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
	// KindAsk 는 ★ 사람이 수행하는 단계 ★ 다 (ADR-032).
	// 노드도 Mediator 도 아닌 네 번째 실행 주체 — 답이 오면 끝난다.
	KindAsk
	// KindAcquire 는 ★ Mediator 가 수행하는 단계 ★ 다 (ADR-022 §7.5 · ADR-024).
	//
	// ★ 이것이 ADR-019 의 「두 종류」를 뒤집지 않는다 ★ — 그 결정이 가른 것은
	// ★ 판정 방법 ★ 이다(produced 냐 exit_code 냐). 획득 단계는 결과를
	// ★ 산출물로 내므로 판정 계열이 produced 이고 ★, 다른 것은 ★ 누가 실행하나 ★ 라는
	// 별개의 축이다. 노드에 안 가므로 잡기 전에 uses 가 없어도 된다.
	KindAcquire
)

func (k StepKind) String() string {
	switch k {
	case KindAgent:
		return "agent"
	case KindRun:
		return "run"
	case KindAcquire:
		return "acquire"
	case KindAsk:
		return "ask"
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

	// Produces 는 ★ 이 계획이 반드시 지어야 할 단계 이름 ★ 이다 (ADR-049).
	//
	// ★ 왜 필요한가 — 목표를 계약이 표현할 수 없었다 ★
	//
	// success_when 은 ★ 제출 시점에 있는 단계 ★ 만 지목할 수 있다(ErrCondUnknownID).
	// 그런데 계획 위임에서는 ★ 목표를 이루는 단계를 계획이 짓는다 ★ — 그래서
	// 계약 저자가 "무엇이 되면 끝인가" 를 적을 자리가 없었다.
	//
	//	실측 (vm-scratch-1): 조사만 하고 재계획이 빈 계획을 내자 Run 이 SUCCEEDED 로
	//	끝났다. ★ VM 에는 아무것도 안 깔렸다 ★. 9차의 「성공한 일이 FAILED」를
	//	고쳤더니 이번엔 ★ 「안 한 일이 SUCCEEDED」 ★ 가 나왔다.
	//
	// ★ 이 필드가 여는 것 ★ — success_when 이 여기 적힌 이름을 ★ 미리 ★ 가리킬 수 있다.
	// 그리고 계획이 그 이름을 안 지으면 ★ 계약 적용이 거절한다 ★ (applyExpands).
	//
	// ★ 기준의 저자는 여전히 사람이다 ★ — 계획보다 먼저, 어떤 계획도 없을 때
	// 못 박힌다. ADR-037(판정하는 술어를 에이전트가 저작하지 못하게 한다)을
	// ★ 약화시키지 않고 강화한다 ★.
	//
	// ★ 빈 계획과의 관계 ★ — 약속한 이름이 아직 안 지어졌으면 빈 계획을 못 낸다.
	// ADR-043 의 「고칠 것이 없다」는 ★ 목표가 이미 선 뒤 ★ 의 판단이다.
	Produces []string `json:"produces,omitempty"`

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

	// Acquire 는 ★ 실행 중에 새 자원을 잡는다 ★ (ADR-022 §7.5 · ADR-024).
	//
	// ★ I5 를 안 깬다 ★ — ADR-024 가 「요구 자원」을 ★ 한 획득 요청의 범위 ★ 로
	// 정했다. t=0 의 requires 는 그 첫 번째 경우이고, 이것은 두 번째다.
	// 각 요청이 전부-아니면-전무이면 불변식이 살고, ★ 요청 사이에는 안 걸린다 ★ —
	// 실행 중 획득이 실패해도 이미 쥔 것은 놓지 않는다(부분 점유가 아니라 정상 점유).
	//
	// ★ 실패를 중단이 아니라 값으로 만든다 ★ — 결과를 산출물로 내고 dispatch 가
	// 그 이름을 읽는다. ⇒ ②분기의 특수 경우가 되어 ★ 새 의미가 안 생긴다 ★.
	//
	// ★ 이 단계는 Mediator 가 수행한다 ★ — 노드에 안 간다. 잡기 전이므로
	// uses 가 없고, claim 이 집지 않는다.
	Acquire *Acquire `json:"acquire,omitempty"`

	// Ask 는 ★ 사람에게 묻는 단계 ★ 다 (ADR-032).
	//
	// needs 가 차면 Mediator 가 이 단계를 ★ ASKED ★ 로 만들고, ★ 노드는 손 뗀다 ★ —
	// uses 가 없고 claim 에 안 걸리며 자원을 안 잡는다 (조사에서 자원을 쥔 채
	// 기다리는 것이 안티패턴의 교과서였다 — Jenkins 의 node 안 input).
	//
	// ★ 답은 산출물이다 ★ — POST …/answer 의 본문이 out 의 blob 이 되고,
	// PUT blob 과 같은 스키마 검증을 받으며(422 면 저장 안 됨), dispatch 가
	// 그것으로 분기한다. ★ 승인/거부가 이미 분기 문법이다 ★.
	// 질문의 형태는 ★ 단계의 schema 그대로 ★ 다 — 새 폼 언어를 만들지 않는다.
	// 다만 웹 폼 렌더를 위해 ★ 평면 부분집합 ★ 으로 제한한다 (askForm).
	Ask *Ask `json:"ask,omitempty"`

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

	// Loop 은 ★ 뒤로 가는 간선 ★ 이다 (ADR-026).
	//
	// "이 단계가 끝났는데 until 이 불만족이면 ★ back_to 부터 다시 ★" 라는 뜻이고,
	// 구간 [back_to … 이 단계] 가 ★ 블록을 안 적어도 정해진다 ★ (seq 가 위상순서다).
	//
	// ★ 새 종류가 안 는다 ★ — loop 을 든 단계는 여전히 agent 이거나 run 이다.
	// ★ 이미 도는 기계를 넓히는 것이다 ★ — validate_with 루프가 대상과 검증자
	// ★ 둘 ★ 을 되돌리는데, 여기서는 되돌릴 범위가 ★ 구간 ★ 이 된다.
	//
	// ★ repeat 과 다른 물건이다 ★ — repeat 은 같은 단계를 N 회 돌린다
	// (run-contract §4.5, requires[].count 와 짝). 이쪽은 ★ 조건이 만족될 때까지 ★ 다.
	Loop *Loop `json:"loop,omitempty"`
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

// hasVerdict 는 스키마에 verdict 필드가 있고 enum 에 approve·reject 가 있는지 본다.
// ★ 채택의 어휘를 못 박는 검사다 ★ (ADR-033) — 값 일치이지 표현식이 아니다.
func hasVerdict(sch interface{}) bool {
	m, ok := sch.(map[string]interface{})
	if !ok {
		return false
	}
	props, _ := m["properties"].(map[string]interface{})
	v, ok := props["verdict"].(map[string]interface{})
	if !ok {
		return false
	}
	enum, _ := v["enum"].([]interface{})
	has := map[string]bool{}
	for _, e := range enum {
		if s, ok := e.(string); ok {
			has[s] = true
		}
	}
	return has["approve"] && has["reject"]
}

// checkAskForm 은 ask 의 스키마가 ★ 평면 폼 부분집합 ★ 인지 본다 (ADR-032).
//
// 허용: 최상위 type:object · 속성은 원시형(string·number·integer·boolean)
// 또는 enum. 금지: 중첩 객체 · 배열. 웹 폼 렌더러의 계약이 이 부분집합이다 —
// 조사(docs/input-required-survey.md §2③)의 수렴점을 그대로 쓴다.
func checkAskForm(sch interface{}) error {
	m, ok := sch.(map[string]interface{})
	if !ok {
		return fmt.Errorf("ask schema is not an object")
	}
	if t, _ := m["type"].(string); t != "object" {
		return fmt.Errorf("ask schema must have top-level type:\"object\"")
	}
	props, _ := m["properties"].(map[string]interface{})
	for name, raw := range props {
		p, ok := raw.(map[string]interface{})
		if !ok {
			return fmt.Errorf("ask field %q: definition is not an object", name)
		}
		if _, hasEnum := p["enum"]; hasEnum {
			continue // 선택 필드
		}
		switch t, _ := p["type"].(string); t {
		case "string", "number", "integer", "boolean":
		case "object", "array":
			return fmt.Errorf("ask field %q: type %s is not allowed; only flat forms are supported "+
				"(nested objects and arrays are excluded)", name, t)
		default:
			return fmt.Errorf("ask field %q: missing or unknown type; expected "+
				"string, number, integer, boolean, or enum", name)
		}
	}
	return nil
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

// Branches 는 이 단계가 ★ 갈림길로 지목하는 단계들 ★ 이다.
//
// ★ 갈림길이 둘이다 ★ — dispatch(이름을 고른다)와 acquire(잡혔나로 고른다).
// 목적지의 성질은 같으므로 ★ 한 곳에서 계산한다 ★: 기본 needs 도, 뒤인지
// 검사도, 안 간 쪽을 닫는 것도 전부 같은 규칙을 쓴다.
func (s Step) Branches() []string {
	var out []string
	if s.Dispatch != nil {
		out = append(out, s.Dispatch.To...)
	}
	if s.Acquire != nil {
		if s.Acquire.Acquired != "" {
			out = append(out, s.Acquire.Acquired)
		}
		if s.Acquire.Unavailable != "" {
			out = append(out, s.Acquire.Unavailable)
		}
	}
	return out
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
	// ★ 분기 목적지는 형제다 ★ — 목적지로 지목됐다는 것 자체가 이미 선언이므로,
	// 기본값은 직전 단계가 아니라 ★ 갈림길을 낸 단계 ★ 다.
	//
	// ★ 이것을 빠뜨리면 형제가 사슬로 이어진다 ★ — to: ["full","quick"] 에서
	// quick 의 기본값이 [full] 이 되고, full 이 SKIPPED 가 되는 순간 전파가
	// ★ 살아 있어야 할 가지까지 죽인다 ★. 순서 의미로도 틀리다 — full 다음에
	// quick 이 오는 것이 아니다.
	//
	// ★ 갈림길이 둘이므로 Branches 한 곳에서 본다 ★ — dispatch 와 acquire.
	// 한쪽만 보면 같은 결함이 다른 문법으로 재발한다 (실제로 그렇게 밟았다).
	for j := 0; j < i; j++ {
		for _, t := range steps[j].Branches() {
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
	hasAcq := s.Acquire != nil
	hasAsk := s.Ask != nil
	n := 0
	for _, has := range []bool{hasAgent, hasRun, hasAcq, hasAsk} {
		if has {
			n++
		}
	}
	if n > 1 {
		return KindUnknown, fmt.Errorf("step %q: more than one of agent, run, acquire, ask is set", s.ID)
	}
	switch {
	case hasAgent:
		return KindAgent, nil
	case hasRun:
		return KindRun, nil
	case hasAcq:
		return KindAcquire, nil
	case hasAsk:
		return KindAsk, nil
	}
	return KindUnknown, fmt.Errorf("step %q: none of agent, run, acquire, ask is set", s.ID)
}

// Loop 은 구간 반복 하나다 (ADR-026).
type Loop struct {
	// BackTo 는 구간의 ★ 시작 ★ 이다 — 자기보다 앞선 단계.
	// ★ 뒤로 가는 유일한 간선 ★ 이고, needs 와 dispatch 는 뒤로 못 간다.
	BackTo string `json:"back_to"`
	// Max 는 ★ 회차 상한 ★ 이다. ★ 종료를 이것이 보장한다 ★ —
	// loop 은 needs 가 아니므로 DAG 검사의 대상이 아니고, 유한성은 회차가 준다.
	// 소진하면 ★ 그냥 진행한다 ★ — 성패는 success_when 이 정한다(I3).
	Max int `json:"max"`
	// Until 은 ★ 그만 돌 조건 ★ 이다. ★ 주어는 이 단계다 ★ —
	// 구간의 끝이 여기이므로 조건도 여기 걸린다. 남의 결과로 내 루프를 돌리는 것은
	// 반복이 아니라 ★ 재계획 ★ 이다(P6).
	//
	// ★ 식 언어가 아니다 ★ — exit_code · produced 는 success_when 이 이미 쓰는
	// 어휘 그대로다(ADR-004). ★ 확장이 아니라 재사용 ★ 이고, 정규식이나 값 비교를
	// 더하는 순간이 식 언어의 시작이므로 거기서 막는다.
	Until Condition `json:"until"`
}

// Condition 은 success_when 의 항목 하나다 (ADR-004).
// 계약은 "완주했는가" 만 묻는다. "결과가 무엇인가" 는 묻지 않는다.
type Condition struct {
	Step     string   `json:"step"`
	Produced []string `json:"produced,omitempty"`
	// ExitCode 는 ★ 명령 단계에만 ★ 쓸 수 있다 (ADR-019).
	// agent 단계에 쓰면 계약이 틀린 것이다 — claude 는 헛소리를 하고도 0 으로 끝난다.
	ExitCode *int `json:"exit_code,omitempty"`
	// Changed 는 ★ 이 단계 안에 실제로 바뀐 워크스페이스 경로 ★ 다 (ADR-037).
	//
	// ★ produced 와 무게가 다르다 ★ — produced 는 「에이전트가 쓴 파일이 있나」이고
	// 이것은 「★ 세상이 바뀌었나 ★」다. 에이전트가 저작하지 않는 관찰이므로
	// ★ 자기 신고에 안 갇힌다 ★. hook.go 가 세 겹을 재며 "diff 는 협조 불필요 —
	// ★ 진짜 안전망 ★" 이라고 적어둔 그 관찰을 판정으로 잇는 것이다.
	//
	// ★ agent 단계에 쓸 수 있다 ★ — exit_code 가 금지된 자리를 이것이 메운다.
	// 다만 「바뀌었다 ≠ 옳게 바뀌었다」이고, 후자는 ADR-004 가 범위 밖으로 뒀다.
	Changed        []string `json:"changed,omitempty"`
	WithinAttempts bool     `json:"within_attempts,omitempty"`
	// FleetHas 는 ★ 함대에 그런 노드가 실제로 서 있는가 ★ 다 (ADR-058).
	//
	// ★ 다른 술어와 무엇이 다른가 ★ — 대상이 이 Run 밖이다.
	//
	//	produced   에이전트가 ★ 만든 파일 ★ 이 있나        ← 저작한다
	//	exit_code  에이전트가 ★ 짠 명령 ★ 의 종료코드      ← 저작한다
	//	changed    ★ 워크스페이스가 실제로 바뀌었나 ★      ← ADR-037: 저작 못한다
	//	fleet_has  ★ 그런 노드가 광고하고 있나 ★           ← ★ 저작 못한다 ★
	//
	// ★ 왜 필요했나 ★ — vm-scratch 일곱 판의 목표가 "노드가 함대에 능력을
	// 광고하며 선다" 인데, 계약이 그것을 표현할 수단이 없었다. 그래서 대리를
	// 세 번 갈아탔고 갈 때마다 새 결함이 났다 (ADR-058 §1).
	//
	// ★ Require 와 같은 어휘다 ★ — 매처가 이미 그 형태를 푼다(Advert.Satisfies).
	// 새 술어를 짜지 않는다 (ADR-014 결정 3: 매처를 두 벌 만들지 않는다).
	//
	// ★ step 을 안 쓴다 ★ — 단계에 걸리는 조건이 아니다.
	FleetHas *Require `json:"fleet_has,omitempty"`
	// MinCount 는 ★ 그런 노드가 몇 이상이어야 하는가 ★ 다. 없으면 1.
	MinCount int `json:"min_count,omitempty"`
}

// IsFleet 은 이 조건이 ★ 단계가 아니라 함대 ★ 를 보는가다 (ADR-058).
func (c Condition) IsFleet() bool { return c.FleetHas != nil }

// Want 는 fleet_has 가 요구하는 것이다. min_count 의 기본값을 여기서 채운다 —
// ★ 읽는 쪽이 여럿이므로 기본값을 각자 알게 두면 언젠가 한 곳이 어긋난다 ★.
func (c Condition) Want() (Require, int) {
	n := c.MinCount
	if n <= 0 {
		n = 1
	}
	return *c.FleetHas, n
}

// ★ agent 와 in 이 받는 키 ★ (ADR-057) — 여기가 정본이다.
//
// ★ 어댑터의 구조체와 짝이다 ★: agent → enode.AgentParams · in → enode.Step.In.
// 한쪽이 늘면 다른 쪽도 늘어야 하고, 그것을 잊으면 ★ 계약이 거절하거나
// 어댑터가 버린다 ★ — 둘 다 조용하지 않다.
var (
	agentKeys = []string{"model", "max_turns", "max_tokens", "ask", "harness"}
	// ★ diff 는 자리다 ★ — run-contract §5 의 "@work.patch_rev 참조 해석" 이고
	// ★ 아직 런타임이 안 읽는다 ★ (seal.go 가 같은 말을 적어뒀다).
	// 시연 계약(testdata/demo.json)이 정본으로 그것을 쓰므로 허용한다.
	//
	// ★ 자리 남기기와 조용한 무시를 가른다 ★ — 문서가 정의했고 코드가
	// "아직" 이라고 적어둔 것은 자리다. 아무 데도 없는 이름은 오타다.
	// 그래서 이 목록은 ★ 문서에 있는 것 ★ 이고, 문법(Grammar)이 계획에게
	// 가르치는 것은 ★ 오늘 읽히는 것 ★ 뿐이다.
	inKeys = []string{"prompt", "from", "diff"}
)

// knownKeys 는 map 필드에 모르는 키가 있으면 ★ 어디에 적어야 하는지 ★ 와 함께 거절한다.
func knownKeys(stepID, field string, m map[string]interface{}, allowed []string) error {
	for k := range m {
		if contains(allowed, k) {
			continue
		}
		hint := ""
		if field == "agent" {
			// ★ 제일 흔한 오해를 지목한다 ★ — 실측에서 밟은 그 자리다.
			hint = "; the task for the agent goes in in.prompt, and agent carries " +
				"execution parameters only"
		}
		return fmt.Errorf("step %q: unknown field %q in %s (allowed: %s)%s",
			stepID, k, field, strings.Join(allowed, ", "), hint)
	}
	return nil
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
	ErrNoRunID       = errors.New("run_id is missing")
	ErrNoRequires    = errors.New("requires is empty")
	ErrNoSteps       = errors.New("steps is empty")
	ErrUnknownCap    = errors.New("unknown capability")
	ErrDupAs         = errors.New("duplicate requires[].as")
	ErrDupStepID     = errors.New("duplicate steps[].id")
	ErrExitOnAgent   = errors.New("exit_code condition is not allowed on an agent step")
	ErrCondUnknownID = errors.New("success_when refers to an unknown step")
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
			return fmt.Errorf("requires[].as is empty")
		}
		if roles[r.As] {
			return fmt.Errorf("%w: %q", ErrDupAs, r.As)
		}
		roles[r.As] = true
	}

	// ★ 실행 중에 잡는 역할도 이 Run 의 역할이다 ★ (ADR-022 §7.5) —
	// 뒤 단계가 그것을 uses 로 쓴다. ★ 잡기 전에 못 쓴다 ★ 는 것은 아래에서
	// 순서로 따로 본다(후손 검사). 여기서는 이름이 있는가만 본다.
	acquired := map[string]bool{}
	for _, st := range c.Steps {
		if st.Acquire == nil || st.Acquire.Want == nil || st.Acquire.Want.As == "" {
			continue
		}
		if roles[st.Acquire.Want.As] || acquired[st.Acquire.Want.As] {
			return fmt.Errorf("step %q: acquire role %q already exists", st.ID, st.Acquire.Want.As)
		}
		acquired[st.Acquire.Want.As] = true
		roles[st.Acquire.Want.As] = true
	}

	kinds := map[string]StepKind{}
	for _, s := range c.Steps {
		if s.ID == "" {
			return fmt.Errorf("steps[].id is empty")
		}
		if _, dup := kinds[s.ID]; dup {
			return fmt.Errorf("%w: %q", ErrDupStepID, s.ID)
		}
		k, err := s.Kind()
		if err != nil {
			return err
		}
		kinds[s.ID] = k
		// ★ 획득·되묻기 단계는 uses 가 없다 ★ — 노드가 수행하지 않는다.
		if k != KindAcquire && k != KindAsk && !roles[s.Uses] {
			return fmt.Errorf("step %q uses undeclared role %q", s.ID, s.Uses)
		}
		// ★ ADR-020 의 경계선을 여기서 400 으로 만든다 ★
		// "스키마는 형식만 제약한다" 를 산문으로 두면 새어나가고,
		// 그 순간 ADR-004(기계적 판정만)가 스키마를 통해 무너진다.
		for name, sch := range s.Schema {
			if err := schema.CheckBoundary(sch); err != nil {
				return fmt.Errorf("step %q: %s: %w", s.ID, name, err)
			}
			if !contains(s.Out, name) {
				return fmt.Errorf("step %q: schema declared for %q, which the step does not produce", s.ID, name)
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
				return fmt.Errorf("step %q: output name %q is invalid; "+
					"names starting with an underscore are reserved", st.ID, o)
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
				return fmt.Errorf("step %q: in.from refers to %q, which no step produces",
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
			return fmt.Errorf("step %q: unknown see.ledger %q; expected %q or %q",
				st.ID, st.See.Ledger, SeeNone, SeeList)
		}
		if len(st.See.From) > 0 {
			return fmt.Errorf("step %q: see.from is not supported; "+
				"use in.from instead", st.ID)
		}
	}

	if c.Ledger != nil && !knownScope(c.Ledger.Scope) {
		return fmt.Errorf("unknown ledger.scope %q; expected %q or %q",
			c.Ledger.Scope, ScopeRun, ScopeWork)
	}

	// ★ 한 단계가 한 번 ★ 늘린다 (ADR-022 §6.3 · ADR-031).
	//
	// ★ 계약 하나에 여러 개가 있어도 된다 ★ — 그것이 재계획이다.
	// 지어진 단계가 또 expands 를 들면 다음 판이 붙고, 그렇게 계약이 자란다.
	// ★ 유한성은 계약이 아니라 시스템이 준다 ★ — 판의 개수 상한(ADR-031).
	// 계약 안에 상한을 두면 ★ 계약을 짓는 기계가 자기 상한을 늘린다 ★.
	for _, st := range c.Steps {
		if !st.Expands {
			continue
		}
		// ★ 계획은 산출물이다 ★ — 이름이 없으면 무엇을 읽어야 할지 모른다.
		if len(st.Out) != 1 {
			return fmt.Errorf("step %q: an expands step must declare exactly one output", st.ID)
		}
		// ★ 스키마가 없으면 무엇이든 계약으로 들어온다 ★ — 형태 검증이
		// PUT blob 에서 걸리게 하려면 계약이 스키마를 들고 있어야 한다 (ADR-020).
		if _, ok := st.Schema[st.Out[0]]; !ok {
			return fmt.Errorf("step %q: expands step output %q has no schema; "+
				"a schema is required to validate the generated contract", st.ID, st.Out[0])
		}
	}

	// ★ 모르는 필드를 거절한다 ★ (ADR-057)
	//
	// agent 와 in 은 map 이라 ★ 무엇이든 받는다 ★. 어댑터는 아는 키만 읽고
	// 나머지는 ★ 조용히 사라진다 ★ — 400 도 422 도 훅도 안 난다.
	//
	//	★ 실측 ★ (vm-scratch-6) 오케스트레이터가 과제 3,308자를 agent.task 에
	//	적었다. AgentParams 에 그 필드가 없어 통째로 버려졌고, in.prompt 가
	//	비어 "### 요청" 이 빈 절로 나갔다. 실행 단계가 _cannot 을 냈다:
	//	"요청 본문이 비어 있다 … 근거 없는 추측으로 설치하는 것이 빈손보다 나쁘다".
	//
	// ★ 조용한 무시가 가장 나쁘다 ★ — 이 저장소가 runctl 의 인자 순서에 대해
	// 이미 적은 말이고, 정작 계약의 두 자리가 그것을 어기고 있었다.
	for _, st := range c.Steps {
		if err := knownKeys(st.ID, "agent", st.Agent, agentKeys); err != nil {
			return err
		}
		if err := knownKeys(st.ID, "in", st.In, inKeys); err != nil {
			return err
		}
	}

	// ★ produces 는 expands 단계에만 쓸 수 있다 ★ (ADR-049) —
	// 「계획이 지을 것」이므로 계획을 짓지 않는 단계에는 의미가 없다.
	//
	// ★ 약속한 이름은 이 단계보다 뒤에 서야 한다 ★
	//
	// 처음에는 "이미 있으면 거절" 이었다. ★ 그것이 v2 를 막았다 ★ —
	// 계획이 붙으면 그 이름은 ★ 당연히 존재한다 ★ (그것이 약속의 이행이다).
	// applyExpands 가 "약속한 이름을 지었는가" 를 확인한 ★ 바로 다음 줄에서 ★
	// 같은 계약을 Validate 하면 "이미 있다" 로 거절됐다 — ★ ①이 요구한 것을
	// ②가 금지했다 ★. vm-scratch-2 의 1판이 이것으로 죽었다: 계획은 옳았다.
	//
	// 남는 것은 ★ 순서 ★ 다. 계획이 지은 단계는 언제나 뒤에 붙으므로
	// (next.Steps = 기존 + 계획), 약속한 이름이 ★ 자신이거나 앞 ★ 이면
	// 그것은 이행이 아니라 ★ 계약 저자의 착각 ★ 이다.
	for _, st := range c.Steps {
		if len(st.Produces) == 0 {
			continue
		}
		if !st.Expands {
			return fmt.Errorf("step %q: produces is only allowed on an expands step; "+
				"a step that does not build a plan cannot promise steps", st.ID)
		}
		for _, n := range st.Produces {
			if at, dup := index[n]; dup && at <= index[st.ID] {
				return fmt.Errorf("step %q: produces names %q, which already stands at or "+
					"before this step; a promised step is built by the plan and comes after",
					st.ID, n)
			}
		}
	}

	// ★ ask 는 「사람이 수행하는」 단계다 ★ (ADR-032).
	for _, st := range c.Steps {
		a := st.Ask
		if a == nil {
			continue
		}
		if st.Uses != "" {
			return fmt.Errorf("step %q: an ask step must not set uses; it is performed by a person", st.ID)
		}
		if a.Prompt == "" {
			return fmt.Errorf("step %q: ask.prompt is empty", st.ID)
		}
		// ★ 답은 산출물이다 ★ — 이름과 형태가 있어야 검증하고 분기한다.
		if len(st.Out) != 1 {
			return fmt.Errorf("step %q: an ask step must declare exactly one output", st.ID)
		}
		sch, ok := st.Schema[st.Out[0]]
		if !ok {
			return fmt.Errorf("step %q: ask step output %q has no schema; "+
				"the schema defines the form of the question", st.ID, st.Out[0])
		}
		// ★ 평면 폼 부분집합 ★ (ADR-032 §1③) — 조사에서 여섯 시스템이 독립적으로
		// 수렴한 형태이고, MCP 는 이유까지 적었다: 클라이언트(웹 폼 렌더러)
		// 단순화를 위해 중첩을 ★ 의도적으로 ★ 뺀다.
		if err := checkAskForm(sch); err != nil {
			return fmt.Errorf("step %q: %w", st.ID, err)
		}
		for _, name := range a.Show {
			if !produced[name] {
				return fmt.Errorf("step %q: show refers to %q, which no step produces", st.ID, name)
			}
		}
		if a.Adopts != "" {
			j, ok := index[a.Adopts]
			if !ok {
				return fmt.Errorf("step %q: adopts refers to unknown step %q", st.ID, a.Adopts)
			}
			if !c.Steps[j].Expands {
				return fmt.Errorf("step %q: adopts target %q is not an expands step; "+
					"there is no proposal to adopt", st.ID, a.Adopts)
			}
			// ★ 승인의 어휘를 못 박는다 ★ — verdict 에 approve 와 reject 가 있어야
			// 답이 채택인지 아닌지가 기계적으로 갈린다 (표현식이 아니라 값 일치).
			if !hasVerdict(sch) {
				return fmt.Errorf("step %q: an adopting ask must define a verdict field "+
					"whose enum includes approve and reject", st.ID)
			}
		}
		if t := a.Timeout; t != nil {
			d, err := time.ParseDuration(t.After)
			if err != nil || d <= 0 {
				return fmt.Errorf("step %q: cannot parse ask.timeout.after %q", st.ID, t.After)
			}
			// ★ 오늘 then 은 "fail" 뿐이다 ★ — default·escalate 는 순연 (ADR-032 §2).
			// 모르는 값을 조용히 무시하지 않는다 (ADR-013 의 --interactive 와 같은 자세).
			if t.Then != "fail" {
				return fmt.Errorf("step %q: unsupported ask.timeout.then %q; only \"fail\" is supported", st.ID, t.Then)
			}
		}
	}

	// ★ loop 은 뒤로 가는 유일한 간선이다 ★ (ADR-026).
	// needs 와 dispatch 는 뒤로 못 가고, 이것만 간다. 종료는 max 가 준다.
	for i, st := range c.Steps {
		lp := st.Loop
		if lp == nil {
			continue
		}
		if st.ValidateWith != "" {
			return fmt.Errorf("step %q: loop and validate_with cannot be combined; "+
				"two retry drivers would desynchronize the attempt counter", st.ID)
		}
		j, ok := index[lp.BackTo]
		if !ok {
			return fmt.Errorf("step %q: loop.back_to refers to unknown step %q", st.ID, lp.BackTo)
		}
		if j >= i {
			return fmt.Errorf("step %q: loop.back_to target %q comes after this step; "+
				"a loop must go backward", st.ID, lp.BackTo)
		}
		if lp.Max < 2 {
			return fmt.Errorf("step %q: loop.max must be at least 2; "+
				"a single pass is not a loop", st.ID)
		}
		if lp.Until.ExitCode == nil && len(lp.Until.Produced) == 0 {
			return fmt.Errorf("step %q: loop.until is empty; "+
				"specify a stop condition with exit_code or produced", st.ID)
		}
		if lp.Until.Step != "" {
			return fmt.Errorf("step %q: loop.until must not name a step; "+
				"the condition always applies to this step", st.ID)
		}
		if lp.Until.ExitCode != nil && (kinds[st.ID] == KindAgent || kinds[st.ID] == KindAsk) {
			return fmt.Errorf("%w: %q (loop.until)", ErrExitOnAgent, st.ID)
		}
		// ★ 구간 안에서 놓으면 안 된다 ★ — 다음 회차가 그 자원을 쓰는데
		// 놓은 것은 되돌릴 수 없다.
		for k := j; k <= i; k++ {
			if len(c.Steps[k].Release) > 0 {
				return fmt.Errorf("step %q: step %q inside the loop range releases a resource; "+
					"a release cannot be undone and the next pass needs it", st.ID, c.Steps[k].ID)
			}
			// ★ 중첩은 오늘 막는다 ★ — 여는 조건은 중첩이 필요한 실물 계약이
			// 나올 때이고, 그때 깊이 상한이 따라온다 (ADR-026 §6).
			if k != i && c.Steps[k].Loop != nil {
				return fmt.Errorf("step %q: nested loop inside the loop range; "+
					"nesting is not supported", st.ID)
			}
		}
	}

	// ★ acquire 는 「잡기 전」의 단계다 ★ (ADR-022 §7.5 · ADR-024).
	for i, st := range c.Steps {
		if st.Acquire == nil {
			continue
		}
		a := st.Acquire
		if a.Want == nil || a.Want.As == "" {
			return fmt.Errorf("step %q: acquire.want.as is missing", st.ID)
		}
		if !knownCapability(a.Want.Capability) {
			return fmt.Errorf("step %q: unknown acquire capability %q",
				st.ID, a.Want.Capability)
		}
		if a.Want.Count > 1 {
			return fmt.Errorf("step %q: acquire takes one resource at a time; "+
				"acquiring several at once requires all-or-nothing rollback", st.ID)
		}
		if st.Uses != "" {
			return fmt.Errorf("step %q: an acquire step must not set uses; "+
				"it runs on the mediator before the resource is held", st.ID)
		}
		// ★ 분기의 규칙을 그대로 쓴다 ★ (ADR-022 §7.2) — 실존 · 서로 다름 ·
		// ★ 전부 뒤 ★. 뒤로 못 가면 DAG 이고 종료가 제출 시점에 보장된다.
		if a.Acquired == "" || a.Unavailable == "" {
			return fmt.Errorf("step %q: acquire requires both acquired and unavailable targets; "+
				"an unavailable resource is a value, not an abort", st.ID)
		}
		if a.Acquired == a.Unavailable {
			return fmt.Errorf("step %q: acquire targets are identical", st.ID)
		}
		for _, dst := range []string{a.Acquired, a.Unavailable} {
			j, ok := index[dst]
			if !ok {
				return fmt.Errorf("step %q: acquire refers to unknown step %q", st.ID, dst)
			}
			if j <= i {
				return fmt.Errorf("step %q: acquire target %q comes before this step; "+
					"branches must point forward", st.ID, dst)
			}
		}
		// ★ 잡기 전에는 못 쓴다 ★ — 그 역할을 쓰는 단계는 전부 이 단계의 후손이어야 한다.
		for j, other := range c.Steps {
			if other.Uses != a.Want.As {
				continue
			}
			if !ancestors(c.Steps, j)[i] {
				return fmt.Errorf("step %q: role %q acquired here is used by step %q, which "+
					"is not guaranteed to run after this step", st.ID, a.Want.As, other.ID)
			}
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
				return fmt.Errorf("step %q: duplicate %q in release", st.ID, role)
			}
			seen[role] = true
			if !roles[role] {
				return fmt.Errorf("step %q: release names undeclared role %q", st.ID, role)
			}
			for j, other := range c.Steps {
				if other.Uses != role || j == i {
					continue
				}
				if !anc[j] {
					return fmt.Errorf("step %q: releases %q but step %q uses it and "+
						"is not guaranteed to run first; a release cannot be undone",
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
				return fmt.Errorf("step %q: duplicate %q in needs", st.ID, n)
			}
			seen[n] = true
			j, ok := index[n]
			if !ok {
				return fmt.Errorf("step %q: needs refers to unknown step %q", st.ID, n)
			}
			// ★ 뒤로 못 간다 ★ — 자기 자신도 여기서 걸린다 (j == i).
			// 뒤로 가야 하는 것은 의존이 아니라 반복이고 그건 repeat 의 자리다.
			if j >= i {
				return fmt.Errorf("step %q: needs target %q comes after this step; "+
					"dependencies must point backward", st.ID, n)
			}
		}
	}

	for i, st := range c.Steps {
		d := st.Dispatch
		if d == nil {
			continue
		}
		if d.From == "" {
			return fmt.Errorf("step %q: dispatch.from is empty", st.ID)
		}
		// from 의 첫 조각은 ★ 이 단계가 실제로 내는 산출물 ★ 이어야 한다.
		// 아니면 실행 시에 "고를 값이 없다" 로 조용히 죽는다.
		blob, _, _ := strings.Cut(d.From, ".")
		if !contains(st.Out, blob) {
			return fmt.Errorf("step %q: dispatch.from refers to %q, which this step does not produce",
				st.ID, blob)
		}
		// ★ 갈림길이 하나면 갈림길이 아니다 ★ — 순차로 쓰면 될 것을
		// 분기로 쓰면 읽는 사람이 경로가 갈린다고 오해한다.
		if len(d.To) < 2 {
			return fmt.Errorf("step %q: dispatch.to needs at least two targets", st.ID)
		}
		seen := map[string]bool{}
		for _, t := range d.To {
			if seen[t] {
				return fmt.Errorf("step %q: duplicate %q in dispatch.to", st.ID, t)
			}
			seen[t] = true
			j, ok := index[t]
			if !ok {
				return fmt.Errorf("step %q: dispatch.to refers to unknown step %q", st.ID, t)
			}
			// ★ 뒤로 못 간다 = DAG = 종료가 정적으로 보장된다 ★ (ADR-022 §7.2).
			// 뒤로 가야 하는 것은 분기가 아니라 ★ 반복 ★ 이고 그건 repeat 의 자리다.
			if j <= i {
				return fmt.Errorf("step %q: dispatch.to target %q comes before this step; "+
					"branches must point forward", st.ID, t)
			}
		}
		// ★ 고른 뒤에 닿을 수 있는가 ★ (ADR-060 §2) — DAG 검사는 ★ 종료 ★ 를
		// 보장하지 ★ 도달 ★ 을 보장하지 않는다. 목적지가 안 간 쪽의 뒷단계에
		// 매달려 있으면, 그것을 골라도 SKIPPED 전파가 그 자리를 지운다.
		//
		// ★ 실측이 이것을 밟았다 ★ (third-run-1) — 계획이 재시도 루프를 loop 없이
		// 선형으로 펴고 final_verify 를 세 분기의 공통 출구로 삼았는데,
		// 그 needs 는 사슬의 끝(build_4)만 가리켜서 어느 분기로도 못 닿았다.
		// 그런데 계약은 통과했고 Run 은 SUCCEEDED 로 봉인됐다.
		for _, t := range d.To {
			if dead := skipClosure(c, index, st.Dispatch.To, t); dead[t] {
				return fmt.Errorf("step %q: dispatch.to target %q cannot be reached when it is "+
					"chosen; its needs hang off a branch this dispatch would skip. "+
					"a retry loop belongs in loop (back_to/max/until), not in a chain of "+
					"branches that share one exit", st.ID, t)
			}
		}
	}

	// ★ 계획이 짓기로 약속한 이름 ★ 은 아직 없어도 지목할 수 있다 (ADR-049).
	// 그래야 사람이 ★ 계획보다 먼저 ★ 「무엇이 되면 끝인가」를 못 박는다 —
	// 목표를 이루는 단계를 계획이 짓기 때문이다.
	promised := map[string]bool{}
	for _, st := range c.Steps {
		for _, n := range st.Produces {
			promised[n] = true
		}
	}
	for _, cond := range c.SuccessWhen {
		// ★ 함대 조건은 단계에 안 걸린다 ★ (ADR-058) — 그래서 step 을 안 본다.
		if cond.IsFleet() {
			if cond.Step != "" {
				return fmt.Errorf("success_when: fleet_has must not name a step; "+
					"it asks about the fleet, not about one step (got %q)", cond.Step)
			}
			if cond.ExitCode != nil || len(cond.Produced) > 0 ||
				len(cond.Changed) > 0 || cond.WithinAttempts {
				return errors.New("success_when: fleet_has cannot be combined with " +
					"step conditions in one entry; write them as separate entries")
			}
			if !knownCapability(cond.FleetHas.Capability) {
				return fmt.Errorf("success_when: fleet_has names unknown capability %q",
					cond.FleetHas.Capability)
			}
			if cond.MinCount < 0 {
				return errors.New("success_when: fleet_has min_count must not be negative")
			}
			continue
		}
		k, ok := kinds[cond.Step]
		if !ok {
			if promised[cond.Step] {
				// ★ 아직 안 지어졌다 ★ — 계획이 지으면 그때 종류가 정해지고,
				// 늘어난 계약이 다시 Validate 를 받으므로 검사가 안 새어나간다.
				// 그리고 안 지으면 applyExpands 가 거절한다.
				continue
			}
			return fmt.Errorf("%w: %q", ErrCondUnknownID, cond.Step)
		}
		// ★ 종료코드는 명령 단계의 것이다 ★ (ADR-019)
		//
		// 처음에는 agent 만 막았다. 그런데 ★ ask 와 acquire 도 종료코드가 없다 ★ —
		// 사람의 답과 자원 획득에는 프로세스가 없다. 조건이 조용히 통과하면
		// Verify 가 got=-1 로 비교해 ★ 언제나 거짓 ★ 이 되고, 계약 저자는
		// 자기가 무엇을 잘못 적었는지 못 본다.
		// ★ 종류를 열거하지 않고 「명령이 아니면」으로 적는다 ★ — 종류가 늘 때
		// 이 자리를 다시 안 고친다.
		if cond.ExitCode != nil && k != KindRun {
			return fmt.Errorf("%w: %q", ErrExitOnAgent, cond.Step)
		}
		// ★ changed 는 워크스페이스를 쓰는 단계에만 ★ (ADR-037) —
		// 워크스페이스가 없으면 「바뀐 것」을 잴 기준이 없다. 조용히 참이 되면
		// ★ 공허한 조건 ★ 이 하나 늘 뿐이고, 그것을 Verify 가 못 가려낸다.
		if len(cond.Changed) > 0 {
			j := index[cond.Step]
			if c.Steps[j].Workspace == nil {
				return fmt.Errorf("step %q: changed requires a workspace on that step; "+
					"there is no baseline to compare against", cond.Step)
			}
			for _, p := range cond.Changed {
				if strings.HasPrefix(p, "/") || strings.Contains(p, "..") {
					return fmt.Errorf("step %q: changed path %q must be relative to the workspace",
						cond.Step, p)
				}
			}
		}
	}
	return nil
}

// HasCondition 은 ★ 같은 조건이 이미 목록에 있는가 ★ 다 (ADR-060 §4).
//
// 계약 저자가 쓴 조건을 계획이 다시 제안하는 것은 흔하다. 그때 그냥 붙이면
// ★ 같은 조건이 두 번 선다 ★ — 오늘은 사본이 같아 판정이 안 바뀌지만,
// Verify 가 ★ 대조된 조건의 수 ★ 를 「전부 건너뛰었나」의 하한으로 쓰므로
// 그 수가 사본만큼 부풀면 하한이 잘못된 근거로 판단한다.
func HasCondition(list []Condition, want Condition) bool {
	for _, c := range list {
		if sameCondition(c, want) {
			return true
		}
	}
	return false
}

func sameCondition(a, b Condition) bool {
	if a.Step != b.Step || a.WithinAttempts != b.WithinAttempts {
		return false
	}
	if (a.ExitCode == nil) != (b.ExitCode == nil) {
		return false
	}
	if a.ExitCode != nil && *a.ExitCode != *b.ExitCode {
		return false
	}
	if !sameStrings(a.Produced, b.Produced) || !sameStrings(a.Changed, b.Changed) {
		return false
	}
	// ★ 함대 조건은 속성 맵까지 봐야 같다 ★ (ADR-058) — 어휘가 창발하므로
	// 이름만 맞춰서는 다른 요구를 같다고 부를 수 있다.
	if (a.FleetHas == nil) != (b.FleetHas == nil) {
		return false
	}
	if a.FleetHas != nil {
		wa, na := a.Want()
		wb, nb := b.Want()
		if na != nb || wa.Capability != wb.Capability || len(wa.Attrs) != len(wb.Attrs) {
			return false
		}
		for k, v := range wa.Attrs {
			if wb.Attrs[k] != v {
				return false
			}
		}
	}
	return true
}

// sameStrings 는 ★ 순서까지 같은가 ★ 다. 조건의 목록은 계약 저자가 쓴 순서를
// 그대로 지니고, 순서가 다르면 다르게 봉인되므로 여기서도 다르게 본다.
func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// skipClosure 는 ★ 이 분기가 chosen 을 골랐을 때 죽는 단계들 ★ 을 계약 위에서
// 미리 돈다 (ADR-060 §2). ★ store 의 propagateSkips 와 같은 규칙 ★ 이다 —
// needs 가 ★ 전부 ★ 죽었을 때만 죽고, 하나라도 살아 있으면 산다.
//
// ★ chosen 자신도 대상에 넣는다 ★ — applyDispatch 가 고른 것을 되살리지만,
// 되살린 뒤에도 전파는 그대로 돌기 때문이다. 되살리는 것은 ★ 분기의 판단 ★ 이지
// 도달 가능성이 아니고, needs 가 전부 죽었으면 다시 죽는 것이 맞다.
func skipClosure(c Contract, index map[string]int, to []string, chosen string) map[string]bool {
	dead := map[string]bool{}
	for _, t := range to {
		if t != chosen {
			dead[t] = true
		}
	}
	for {
		grew := false
		for i, st := range c.Steps {
			// ★ NeedsOf 로 본다 ★ — 안 적은 needs 는 [직전 단계] 또는
			// [갈림길을 낸 단계] 로 채워져 들어오므로, 여기서 st.Needs 를
			// 그대로 보면 ★ 실행 시의 그래프와 다른 그래프 ★ 를 검사하게 된다.
			needs := NeedsOf(c.Steps, i)
			if dead[st.ID] || len(needs) == 0 {
				continue
			}
			all := true
			for _, n := range needs {
				if _, ok := index[n]; !ok {
					all = false // 모르는 이름은 다른 검사가 잡는다
					break
				}
				if !dead[n] {
					all = false
					break
				}
			}
			if all {
				dead[st.ID] = true
				grew = true
			}
		}
		if !grew {
			return dead
		}
	}
}
