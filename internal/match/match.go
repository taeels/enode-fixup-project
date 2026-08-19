// Package match 는 요구를 노드에 짝지운다.
//
// ★ 순수 함수여야 한다 ★ (ADR-014 결정 3). POST /v1/runs 와
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
//	422  요구를 만족할 노드가 함대에 ★ 없다 ★    → 영구. 계약을 고쳐야 한다.
//	409  후보는 있는데 ★ 전부 점유됨 ★           → 일시. 다시 제출하면 된다.
//
// 큐가 없으므로(ADR-002 가 QUEUED 를 기각) 재시도는 호출자 몫이고,
// 재시도해도 되는지를 ★ 질의가 아니라 이 코드로 ★ 안다 (ADR-014).
const (
	CodeNoCandidate = 422
	CodeAllBusy     = 409
)

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
// busy 는 ★ 다른 Run 에 이미 묶인 ★ 노드다. 임대 키가 (노드) 이므로
// (ADR-019 결정 2) 노드 자체가 배타 자원이다.
//
// 같은 Run 안에서는 한 노드가 여러 역할을 맡을 수 있다 — 임대가 하나이기
// 때문이다. 다만 같은 역할의 count 안에서는 중복되지 않는다.
func Match(reqs []contract.Require, adverts []contract.Advert, busy map[string]bool) ([]Assignment, *Reject) {
	// first available (ADR-011). 선호 표현이 없으므로 순서만 결정적이면 된다.
	// node_id 오름차순으로 고정한다 — 같은 입력이면 같은 배정이 나온다.
	sorted := make([]contract.Advert, len(adverts))
	copy(sorted, adverts)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].NodeID < sorted[j].NodeID })

	out := make([]Assignment, 0, len(reqs))
	for _, r := range reqs {
		want := r.Wanted()

		var total, free []string
		for _, a := range sorted {
			if !a.Satisfies(r) {
				continue
			}
			total = append(total, a.NodeID)
			if !busy[a.NodeID] {
				free = append(free, a.NodeID)
			}
		}

		// ★ 영구와 일시를 가른다 ★
		// 함대에 아예 없거나 총수가 모자라면 다시 제출해도 영원히 같다.
		if len(total) < want {
			return nil, &Reject{
				Code: CodeNoCandidate, As: r.As,
				Reason: fmt.Sprintf("요구 %d, 함대에 %d — %s", want, len(total), describe(r)),
			}
		}
		if len(free) < want {
			return nil, &Reject{
				Code: CodeAllBusy, As: r.As,
				Reason: fmt.Sprintf("요구 %d, 지금 %d — %s", want, len(free), describe(r)),
			}
		}
		out = append(out, Assignment{As: r.As, Nodes: append([]string(nil), free[:want]...)})
	}
	return out, nil
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
