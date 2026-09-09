// Package match 는 요구를 노드에 짝지운다.
//
// 순수 함수여야 한다 (ADR-014 결정 3). POST /v1/runs 와
// POST /v1/runs/dry-run 이 같은 이 함수를 부르고, 전자만 결과로 점유한다.
// 매처가 두 벌이 되면 언젠가 어긋난다.
package match

import (
	"fmt"
	"sort"
	"strings"

	"github.com/taeels/enode/internal/contract"
)

// 거절 코드 — 와이어의 HTTP 상태 그대로다.
//
//	422  요구를 만족할 노드가 함대에 없다          → 영구. 계약을 고쳐야 한다.
//	409  후보는 있는데 전부 점유됨                 → 일시. 다시 제출하면 된다.
//
// 409 는 매처 안의 상황 이름으로 남는다 — 제출 응답은 202 이고 Run 은 QUEUED 에
// 선다 (ADR-064 · 2026-09-04). 다시 내는 것은 호출자가 아니라 Mediator 의 대기열이다.
// 영구인지 일시인지를 질의가 아니라 이 코드로 안다 (ADR-014).
const (
	CodeNoCandidate = 422
	CodeAllBusy     = 409
)

// attrCount 는 그 노드가 광고한 속성의 개수다.
//
// 희소성의 대리 지표다 (ADR-027 §4.2) — 오늘은 속성이 전부 실제 능력이라
// (harness · arch · board · repo) 개수가 특별함과 같이 간다.
// 장식용 속성이 생기면 이 가정이 깨진다. 그때의 답은 무게를 주는 것이
// 아니라 장식용을 광고에 안 싣는 것이다.
func attrCount(a contract.Advert) int {
	n := 0
	for _, c := range a.Capabilities {
		n += len(c.Attrs)
	}
	return n
}

// Assignment 는 역할 하나에 배정된 노드들이다.
type Assignment struct {
	As    string
	Nodes []string
}

// Reject 는 배정 실패다. 하나라도 실패하면 전부 실패다 (I5).
type Reject struct {
	Code   int
	As     string
	Reason string
}

func (r *Reject) Error() string {
	return fmt.Sprintf("%d %s: %s", r.Code, r.As, r.Reason)
}

// Match 는 요구 전부를 만족시키거나 아무것도 배정하지 않는다.
//
// busy 는 다른 Run 에 이미 묶인 노드다. 임대 키가 (노드) 이므로
// (ADR-019 결정 2) 노드 자체가 배타 자원이다.
//
// 같은 Run 안에서는 한 노드가 여러 역할을 맡을 수 있다 — 임대가 하나이기
// 때문이다. 다만 같은 역할의 count 안에서는 중복되지 않는다.
func Match(reqs []contract.Require, adverts []contract.Advert, busy map[string]bool) ([]Assignment, *Reject) {
	// 속성이 적은 노드를 먼저 준다 (ADR-027).
	//
	// 매칭이 부분집합이므로 속성이 많은 노드일수록 더 많은 요구에 걸린다 —
	// 보드를 가진 노드가 흔한 추론 요구에도 후보가 되고, 그것을 내주면
	// 하나뿐인 보드가 묶인다 (실측에서 밟았다).
	// 적은 쪽을 먼저 주면 희소한 것이 저절로 남는다.
	//
	// 이것이 Rank 는 아니다 — 점수도 가중치도 없고 조건이 하나이며
	// 계약이 그것을 못 건드린다. 다만 두 번째 기준이 생기는 순간
	// 그것은 Rank 이고 ADR-011 이 두 번 기각한 자리다 (ADR-027 §4.1).
	//
	// 동점은 node_id 로 가른다 — 같은 입력이면 같은 배정 (ADR-014 결정 3).
	sorted := make([]contract.Advert, len(adverts))
	copy(sorted, adverts)
	sort.Slice(sorted, func(i, j int) bool {
		ai, aj := attrCount(sorted[i]), attrCount(sorted[j])
		if ai != aj {
			return ai < aj
		}
		return sorted[i].NodeID < sorted[j].NodeID
	})

	// 1차 — 영구 불가를 전부 먼저 본다
	//
	// 순서대로 보다가 첫 실패에서 멈추면, 앞에 일시(409) 뒤에 영구(422)가 있을 때
	// 호출자가 409 를 받고 영원히 재시도한다. 코드가 존재하는 이유가
	// "재시도해도 되는지" 를 알려주는 것이므로(ADR-014 결정 3),
	// 어디든 영구 문제가 있으면 그것이 이긴다.
	for _, r := range reqs {
		if n := countSatisfying(sorted, r, nil); n < r.Wanted() {
			return nil, &Reject{
				Code: CodeNoCandidate, As: r.As,
				Reason: fmt.Sprintf("need %d, fleet has %d - %s", r.Wanted(), n, describe(r)),
			}
		}
	}

	// 2차 — 실제 배정. 여기서 나오는 실패는 전부 일시적이다.
	out := make([]Assignment, 0, len(reqs))
	for _, r := range reqs {
		want := r.Wanted()
		var free []string
		for _, a := range sorted {
			if a.Satisfies(r) && !busy[a.NodeID] {
				free = append(free, a.NodeID)
			}
		}
		if len(free) < want {
			return nil, &Reject{
				Code: CodeAllBusy, As: r.As,
				Reason: fmt.Sprintf("need %d, currently %d - %s", want, len(free), describe(r)),
			}
		}
		out = append(out, Assignment{As: r.As, Nodes: append([]string(nil), free[:want]...)})
	}
	return out, nil
}

func countSatisfying(adverts []contract.Advert, r contract.Require, busy map[string]bool) int {
	n := 0
	for _, a := range adverts {
		if a.Satisfies(r) && !busy[a.NodeID] {
			n++
		}
	}
	return n
}

// describe 는 거절 사유에 요구를 사람이 읽게 적는다.
// 속성 어휘가 창발하므로(ADR-012) 무엇을 요구했는지가 안 보이면 원인을 못 찾는다.
func describe(r contract.Require) string {
	keys := make([]string, 0, len(r.Attrs))
	for k := range r.Attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys)+1)
	parts = append(parts, r.Capability)
	for _, k := range keys {
		parts = append(parts, k+"="+r.Attrs[k])
	}
	return strings.Join(parts, " ")
}
