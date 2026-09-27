package enode

import (
	"fmt"
	"maps"
	"strconv"
	"strings"

	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/lower"
)

// 광고 키 — 예약이다. 라벨이 못 덮는다 (business-rules.md 11절 · ADR-077 §8 · FR-9).
//
// 탐지가 안 내고 광고 때 더한다. 오늘 라벨은 탐지한 키만 못 덮으므로(detect.go 의 capabilities) 이 다섯은
// 따로 막는다 — 라벨로 ir 을 적으면 굽지 않은 IR 을 광고하게 된다.
const (
	KeyWorkspaceWrites = "workspace.writes" // isolated | in-place.  모든 노드
	KeyIR              = "ir"               // metadata source.ir.  runc-overlay
	KeyRepoBuilt       = "repo.built."      // 뒤에 구성 이름 · 값 yes.  metadata builds[].name 마다
	KeyBakeRun         = "bake.run"         // metadata bake.run
	KeyBakeResumed     = "bake.resumed"     // true | false.  bake.run 이 있으면 늘 함께
)

// maxBakeRun 은 bake.run 을 싣는 상한이다 — IR 의 상한과 같다 (계획 4절 ⑮). metadata 1 MiB 안에서 광고 하나가
// 커지지 않게 한다.
const maxBakeRun = 128

// reservedKey 는 노드가 광고 때 정하는 키인가다. 탐지는 이 키를 안 낸다 — 능력에 있으면 라벨에서 왔다.
func reservedKey(k string) bool {
	switch k {
	case KeyWorkspaceWrites, KeyIR, KeyBakeRun, KeyBakeResumed:
		return true
	}
	return strings.HasPrefix(k, KeyRepoBuilt)
}

// metadataKeys 는 .enode-metadata.json 에서 광고 키를 짓는다. 이름과 IR 은 계약의 규칙으로 거른다 — lower 는 같은
// 사용자면 누구나 쓸 수 있는 자리다. 어긋난 것은 그 키만 빼고 까닭을 dropped 에 적는다. metadata 가 없으면 키가 없다.
func metadataKeys(md *lower.Metadata) (keys map[string]string, dropped []string) {
	if md == nil {
		return nil, nil
	}
	keys = map[string]string{}
	if ir := md.Source.IR; ir != nil && *ir != "" {
		if why := contract.IRProblem(*ir); why != "" {
			dropped = append(dropped, fmt.Sprintf("ir %q: %s", *ir, why))
		} else {
			keys[KeyIR] = *ir
		}
	}
	for _, b := range md.Builds {
		if !contract.ValidBuildName(b.Name) {
			dropped = append(dropped, fmt.Sprintf("builds name %q is not 1 to 64 characters of a-z, 0-9 and -", b.Name))
			continue
		}
		keys[KeyRepoBuilt+b.Name] = "yes"
	}
	switch run := md.Bake.Run; {
	case run == "":
	case len(run) > maxBakeRun:
		dropped = append(dropped, fmt.Sprintf("bake.run is longer than %d bytes", maxBakeRun))
	default:
		keys[KeyBakeRun] = run
		keys[KeyBakeResumed] = strconv.FormatBool(md.Bake.Resumed)
	}
	return keys, dropped
}

// advertCaps 는 탐지 능력의 복사본에 노드가 정하는 키를 싣는다 (계획 4절 ④ ⑤ ⑥). 원본은 상태 파일 장부가 들고
// 있으므로 건드리지 않는다 — 원본에 쓰면 삭제자 고루틴이 상태 파일을 쓰며 같은 map 을 읽는 경합이 된다.
//
// 라벨이 적은 예약 키는 빼고 ignored 로 알린다. 뺀 뒤 할 줄 아는 것이 없으면 그 능력을 뺀다 (detect.go 의 「할 줄
// 아는 것이 없으면 광고하지 않는다」). 남은 모든 능력에 workspace.writes 와 keys 를 싣는다.
func advertCaps(caps []contract.Capability, writes string, keys map[string]string, ignored func(key string)) []contract.Capability {
	var out []contract.Capability
	for _, c := range caps {
		attrs := maps.Clone(c.Attrs)
		for k := range attrs {
			if reservedKey(k) {
				delete(attrs, k)
				if ignored != nil {
					ignored(k)
				}
			}
		}
		if !hasCapability(attrs) {
			continue
		}
		attrs[KeyWorkspaceWrites] = writes
		maps.Copy(attrs, keys)
		out = append(out, contract.Capability{Capability: c.Capability, Attrs: attrs})
	}
	return out
}
