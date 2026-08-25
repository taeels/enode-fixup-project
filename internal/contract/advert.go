package contract

// Advert 는 enode 가 POST /v1/nodes 로 올리는 것 전체다 (ADR-012).
//
// 델타가 아니라 매번 전부다 — 그래서 capability 하나를 빼는 것이 곧
// "지금은 못 한다" 가 된다 (ADR-017 결정 3). 디스크가 모자라거나 보드가
// 빠지면 노드가 스스로 그 항목을 빼고, 후보 목록에서 사라진다.
// Mediator 는 아무것도 새로 알 필요가 없다.
type Advert struct {
	NodeID string `json:"node_id"` // hash(email ∥ hostname ∥ realpath(config))
	Label  string `json:"label"`   // 유도된 것. 사람이 안 적는다. Record 가 읽는다.

	// Instance 는 이 프로세스의 「이번 생」 표식이다 (ADR-030).
	//
	// NodeID 는 재시작해도 같다 — 신원이기 때문이다. Instance 는 기동마다
	// 새로 뽑는 난수라 재시작하면 달라진다. 이 차이가 둘을 가른다:
	//   같은 생이 다시 묻는다   → 응답이 유실됐다  → 재전달해도 안전하다
	//   다른 생이 나타났다      → 재시작했다      → 진행 중이던 단계를 믿을 수 없다
	// 비었으면 옛 enode 다 — 재전달도 재시작 판정도 하지 않는다(오늘 그대로).
	Instance string `json:"instance,omitempty"`

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
// 부분집합 일치다 — 요구가 적은 속성을 노드가 전부 같은 값으로 갖고 있으면
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
