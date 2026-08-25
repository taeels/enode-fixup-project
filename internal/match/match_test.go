package match

import (
	"reflect"
	"testing"

	"github.com/taeels/enode/internal/contract"
)

func node(id string, attrs map[string]string) contract.Advert {
	return contract.Advert{
		NodeID: id, Label: id,
		Capabilities: []contract.Capability{{
			Capability: contract.CapabilityAgentReason, Attrs: attrs,
		}},
	}
}

func req(as string, count int, attrs map[string]string) contract.Require {
	return contract.Require{As: as, Capability: contract.CapabilityAgentReason, Count: count, Attrs: attrs}
}

var fleet = []contract.Advert{
	node("n03-board", map[string]string{"board": "SoC-X", "tag": "board-042"}),
	node("n01-mac", map[string]string{"harness": "claude", "repo": "corp/linux"}),
	node("n02-wsl", map[string]string{"arch": "armv7", "repo": "corp/linux"}),
}

func assigned(t *testing.T, as []Assignment, role string) []string {
	t.Helper()
	for _, a := range as {
		if a.As == role {
			return a.Nodes
		}
	}
	t.Fatalf("역할 %q 가 배정에 없다: %+v", role, as)
	return nil
}

func TestMatchHappyPath(t *testing.T) {
	reqs := []contract.Require{
		req("brain", 0, map[string]string{"harness": "claude"}),
		req("builder", 0, map[string]string{"arch": "armv7"}),
		req("board", 0, map[string]string{"board": "SoC-X"}),
	}
	got, rej := Match(reqs, fleet, nil)
	if rej != nil {
		t.Fatalf("거절됐다: %v", rej)
	}
	if n := assigned(t, got, "brain"); !reflect.DeepEqual(n, []string{"n01-mac"}) {
		t.Fatalf("brain=%v", n)
	}
	if n := assigned(t, got, "builder"); !reflect.DeepEqual(n, []string{"n02-wsl"}) {
		t.Fatalf("builder=%v", n)
	}
	if n := assigned(t, got, "board"); !reflect.DeepEqual(n, []string{"n03-board"}) {
		t.Fatalf("board=%v", n)
	}
}

// 영구와 일시를 가르는 것이 ADR-014 결정 3 의 핵심이다
// "지금 비어 있는가" 를 미리 물을 수 없는 대신 거절이 그 답을 준다.
func TestMatchRejectCodes(t *testing.T) {
	cases := []struct {
		name string
		reqs []contract.Require
		busy map[string]bool
		code int
	}{
		{
			"어휘가 함대에 없다 → 영구",
			[]contract.Require{req("v", 0, map[string]string{"machine": "qemu-virt-armv7"})},
			nil, CodeNoCandidate,
		},
		{
			"속성 값이 다르다 → 영구",
			[]contract.Require{req("b", 0, map[string]string{"arch": "riscv64"})},
			nil, CodeNoCandidate,
		},
		{
			"count 가 총수를 넘는다 → 영구",
			[]contract.Require{req("vs", 8, map[string]string{"repo": "corp/linux"})},
			nil, CodeNoCandidate,
		},
		{
			"후보는 있는데 점유됨 → 일시",
			[]contract.Require{req("board", 0, map[string]string{"board": "SoC-X"})},
			map[string]bool{"n03-board": true}, CodeAllBusy,
		},
		{
			"count 는 되는데 지금 모자람 → 일시",
			[]contract.Require{req("vs", 2, map[string]string{"repo": "corp/linux"})},
			map[string]bool{"n01-mac": true}, CodeAllBusy,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, rej := Match(c.reqs, fleet, c.busy)
			if rej == nil {
				t.Fatalf("통과해버렸다: %+v", got)
			}
			if rej.Code != c.code {
				t.Fatalf("code=%d 기대 %d (%s)", rej.Code, c.code, rej.Reason)
			}
			// I5 — 하나라도 실패하면 아무것도 배정하지 않는다.
			if got != nil {
				t.Fatalf("거절인데 배정이 남았다: %+v", got)
			}
		})
	}
}

// I5 — 전부 아니면 전무. 앞의 요구가 되더라도 뒤가 안 되면 전부 없다.
func TestMatchAllOrNothing(t *testing.T) {
	reqs := []contract.Require{
		req("brain", 0, map[string]string{"harness": "claude"}), // 된다
		req("board", 0, map[string]string{"board": "SoC-Y"}),    // 없다
	}
	got, rej := Match(reqs, fleet, nil)
	if rej == nil {
		t.Fatal("통과해버렸다")
	}
	if rej.As != "board" {
		t.Fatalf("어느 역할에서 깨졌는지가 안 나온다: %+v", rej)
	}
	if got != nil {
		t.Fatalf("I5 위반 부분 배정이 남았다: %+v", got)
	}
}

// ADR-019 결정 2 — 임대 키가 (노드) 이므로 노드가 배타 자원이다.
// 다만 같은 Run 안에서는한 노드가 여러 역할을 맡을 수 있다. 임대가 하나이기 때문이다.
func TestMatchOneNodeTwoRoles(t *testing.T) {
	both := []contract.Advert{
		node("n9", map[string]string{"arch": "armv7", "board": "SoC-X", "repo": "corp/linux"}),
	}
	reqs := []contract.Require{
		req("builder", 0, map[string]string{"arch": "armv7"}),
		req("board", 0, map[string]string{"board": "SoC-X"}),
	}
	got, rej := Match(reqs, both, nil)
	if rej != nil {
		t.Fatalf("거절됐다: %v", rej)
	}
	if assigned(t, got, "builder")[0] != "n9" || assigned(t, got, "board")[0] != "n9" {
		t.Fatalf("같은 노드가 두 역할을 못 맡았다: %+v", got)
	}
}

// 같은 역할의 count 안에서는 중복되지 않는다.
func TestMatchCountDistinct(t *testing.T) {
	reqs := []contract.Require{req("vs", 2, map[string]string{"repo": "corp/linux"})}
	got, rej := Match(reqs, fleet, nil)
	if rej != nil {
		t.Fatalf("거절됐다: %v", rej)
	}
	n := assigned(t, got, "vs")
	if len(n) != 2 || n[0] == n[1] {
		t.Fatalf("count 안에서 중복됐다: %v", n)
	}
}

// ADR-011 — first available. 선호 표현이 없으므로 순서만 결정적이면 된다.
// 같은 입력이면 같은 배정이 나와야 재현이 가능하다.
func TestMatchDeterministic(t *testing.T) {
	reqs := []contract.Require{req("any", 0, map[string]string{"repo": "corp/linux"})}
	shuffled := []contract.Advert{fleet[1], fleet[2], fleet[0]}
	a, _ := Match(reqs, fleet, nil)
	b, _ := Match(reqs, shuffled, nil)
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("입력 순서가 배정을 바꿨다: %+v vs %+v", a, b)
	}
}

// dry-run 이 같은 함수를 부른다는 것의 의미 — 이 함수는 아무것도 안 바꾼다.
func TestMatchDoesNotMutate(t *testing.T) {
	before := make([]contract.Advert, len(fleet))
	copy(before, fleet)
	busy := map[string]bool{"n01-mac": true}
	Match([]contract.Require{req("b", 0, map[string]string{"arch": "armv7"})}, fleet, busy)
	if !reflect.DeepEqual(before, fleet) {
		t.Fatal("광고 목록이 변형됐다")
	}
	if len(busy) != 1 || !busy["n01-mac"] {
		t.Fatal("점유 장부가 변형됐다")
	}
}

// 영구가 일시를 이긴다
//
// 앞의 요구가 일시(점유)이고 뒤의 요구가 영구(함대에 없음)이면,
// 순서대로 보다 멈추는 매처는 409 를 낸다 → 호출자가 영원히 재시도한다.
// 코드가 존재하는 이유가 "재시도해도 되는지" 를 알려주는 것이므로 그건 틀렸다.
func TestMatchPermanentBeatsTransient(t *testing.T) {
	reqs := []contract.Require{
		req("busy", 0, map[string]string{"harness": "claude"}), // 있는데 점유됨 → 일시
		req("never", 0, map[string]string{"board": "SoC-Z"}),   // 함대에 없음 → 영구
	}
	_, rej := Match(reqs, fleet, map[string]bool{"n01-mac": true})
	if rej == nil {
		t.Fatal("통과해버렸다")
	}
	if rej.Code != CodeNoCandidate {
		t.Fatalf("code=%d 기대 %d — 영구 문제가 있는데 재시도하라고 답했다 (%s)",
			rej.Code, CodeNoCandidate, rej.Reason)
	}
	if rej.As != "never" {
		t.Fatalf("어느 역할이 영구인지가 안 나온다: %+v", rej)
	}
}

// 속성이 적은 노드를 먼저 준다 (ADR-027)
//
// 매칭이 부분집합이라 속성이 많은 노드일수록 더 많은 요구에 걸린다.
// 그것을 흔한 요구에 내주면 하나뿐인 자원이 묶인다 — 실측에서 밟았다.
func TestMatch_희소한_노드를_아껴_고른다(t *testing.T) {
	adverts := []contract.Advert{
		// node_id 순으로는 보드 노드가 먼저다 — 옛 규칙이면 이것이 뽑힌다.
		{NodeID: "a-board", Capabilities: []contract.Capability{{
			Capability: "agent.reason",
			Attrs:      map[string]string{"harness": "claude", "board": "SoC-X", "tag": "b-042"}}}},
		{NodeID: "z-brain", Capabilities: []contract.Capability{{
			Capability: "agent.reason",
			Attrs:      map[string]string{"harness": "claude"}}}},
	}
	reqs := []contract.Require{{As: "brain", Capability: "agent.reason",
		Attrs: map[string]string{"harness": "claude"}}}

	got, rej := Match(reqs, adverts, map[string]bool{})
	if rej != nil {
		t.Fatalf("배정 실패: %v", rej)
	}
	if got[0].Nodes[0] != "z-brain" {
		t.Fatalf("희소한 노드를 내줬다: %q — 보드가 하나뿐인데 추론에 잡혔다",
			got[0].Nodes[0])
	}

	// 보드 요구는 영향이 없다 — 애초에 그 노드만 후보다.
	boardReq := []contract.Require{{As: "board", Capability: "agent.reason",
		Attrs: map[string]string{"board": "SoC-X"}}}
	got2, rej2 := Match(boardReq, adverts, map[string]bool{})
	if rej2 != nil || got2[0].Nodes[0] != "a-board" {
		t.Fatalf("보드 요구가 어긋났다: %v %v", got2, rej2)
	}

	// 동점은 node_id 가 가른다 — 같은 입력이면 같은 배정이어야 한다.
	tie := []contract.Advert{
		{NodeID: "n2", Capabilities: []contract.Capability{{
			Capability: "agent.reason", Attrs: map[string]string{"harness": "claude"}}}},
		{NodeID: "n1", Capabilities: []contract.Capability{{
			Capability: "agent.reason", Attrs: map[string]string{"harness": "claude"}}}},
	}
	for i := 0; i < 3; i++ {
		g, _ := Match(reqs, tie, map[string]bool{})
		if g[0].Nodes[0] != "n1" {
			t.Fatalf("동점 배정이 흔들린다: %q — 매처는 순수 함수여야 한다", g[0].Nodes[0])
		}
	}
}
