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

// ★ 영구와 일시를 가르는 것이 ADR-014 결정 3 의 핵심이다 ★
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
		t.Fatalf("★ I5 위반 ★ 부분 배정이 남았다: %+v", got)
	}
}

// ADR-019 결정 2 — 임대 키가 (노드) 이므로 노드가 배타 자원이다.
// 다만 ★ 같은 Run 안에서는 ★ 한 노드가 여러 역할을 맡을 수 있다. 임대가 하나이기 때문이다.
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

// ★ dry-run 이 같은 함수를 부른다는 것의 의미 ★ — 이 함수는 아무것도 안 바꾼다.
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
