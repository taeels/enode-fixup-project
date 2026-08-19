package contract

// Advert 는 enode 가 POST /v1/nodes 로 올리는 것 전체다 (ADR-012).
//
// ★ 델타가 아니라 매번 전부다 ★ — 그래서 capability 하나를 빼는 것이 곧
// "지금은 못 한다" 가 된다 (ADR-017 결정 3). 디스크가 모자라거나 보드가
// 빠지면 노드가 스스로 그 항목을 빼고, 후보 목록에서 사라진다.
// Mediator 는 아무것도 새로 알 필요가 없다.
type Advert struct {
	NodeID string `json:"node_id"` // hash(email ∥ hostname ∥ realpath(config))
	Label  string `json:"label"`   // 유도된 것. 사람이 안 적는다. Record 가 읽는다.

	// ADR-019 이후 항목은 사실상 하나(agent.reason)지만 배열을 유지한다 —
	// 나중에 키를 다시 쪼갤 여지를 남기기 위한 것이고, ADR-016 이 임대 목록을
	// 배열로 남긴 것과 같은 이유다.
	Capabilities []Capability `json:"capabilities"`
}

// Capability 는 노드가 가진 것 하나다. 속성의 존재가 곧 능력이다 (ADR-019).
type Capability struct {
	Capability string            `json:"capability"`
	Attrs      map[string]string `json:"attrs"`
}

// Satisfies 는 이 capability 가 요구를 만족하는지 본다.
//
// ★ 부분집합 일치다 ★ — 요구가 적은 속성을 노드가 전부 같은 값으로 갖고 있으면
// 만족이다. 노드에 여분의 속성이 있는 것은 상관없다. 표현식도 범위도 없다.
// 라벨 매칭 계열의 순수한 형태다 (ADR-011).
func (c Capability) Satisfies(r Require) bool {
	if c.Capability != r.Capability {
		return false
	}
	for k, want := range r.Attrs {
		if got, ok := c.Attrs[k]; !ok || got != want {
			return false
		}
	}
	return true
}

// Satisfies 는 노드의 어느 항목이든 요구를 만족하면 참이다.
func (a Advert) Satisfies(r Require) bool {
	for _, c := range a.Capabilities {
		if c.Satisfies(r) {
			return true
		}
	}
	return false
}
