package enode

import (
	"context"
	"log/slog"
	"os/exec"

	"github.com/taeels/enode/internal/contract"
)

// Detect 는 이 기계가 지금 무엇을 할 수 있는지 알아낸다 (ADR-012).
//
// ★ 광고는 델타가 아니라 매번 전부다 ★ 그래서 이 함수의 결과가 곧 능력의 전부이고,
// 못 하는 것을 ★ 빼고 ★ 보내는 것이 "지금은 못 한다" 를 표현하는 방법이다
// (ADR-017 결정 3). Mediator 는 아무것도 새로 알 필요가 없다.
//
// capability 어휘는 agent.reason 하나뿐이므로(ADR-019) 구별은 전부 속성이 한다.
func Detect(l Local, log *slog.Logger) []contract.Capability {
	attrs := map[string]string{}

	// 추론 하네스가 있나 — ★ Probe() 가 곧 executable resolve 다 ★ (R3).
	// 없으면 광고에 안 실리고 → 후보에서 빠지고 → 계약이 요구하면 422 다.
	// "설치 안 된 하네스는 실행 안 한다" 가 ★ 별도 코드 없이 ★ 성립한다.
	for _, h := range harnesses {
		bin := l.HarnessBin
		if h.Name() != "claude" {
			bin = "" // 지금은 claude 만 덮어쓸 수 있다
		}
		ver, err := h.Probe(context.Background(), bin)
		if err != nil {
			continue
		}
		attrs["harness"] = h.Name()
		// ★ 버전은 광고에 안 싣는다 ★ — 매처는 동등 비교뿐이라
		// "2.1.236 (Claude Code)" 같은 문자열은 매칭에 못 쓰고 공간만 더럽힌다.
		// 버전이 필요한 이유는 ★ 기록 ★ 이므로 HarnessResult 로 간다
		// (ADR-005 성질 4 — 봉인된 묶음만 보고 알 수 있어야 한다).
		_ = ver
		break
	}

	// 워크스페이스가 있으면 저장소를 유도한다. ★ 사람이 주소를 안 적는다 ★
	if l.Workspace != "" {
		if repo := DetectRepo(l.Workspace); repo != "" {
			attrs["repo"] = repo
		}
	}

	// 빌드 능력 — 툴체인이 있고 ★ 디스크가 남아 있을 때만 ★ 광고한다.
	if arch := detectArch(l); arch != "" {
		if ok, free := hasRoom(l, log); ok {
			attrs["arch"] = arch
		} else {
			log.Warn("디스크가 모자라 빌드 능력을 광고에서 뺀다",
				"free_gb", free, "min_gb", l.MinFreeGB)
		}
	}

	// 포트에 무엇이 달렸는지는 기계가 모른다 — 그 기계에만 적는다 (ADR-012).
	if l.Board != nil && l.Board.SoC != "" {
		attrs["board"] = l.Board.SoC
		if l.Board.Tag != "" {
			attrs["tag"] = l.Board.Tag
		}
	}

	if len(attrs) == 0 {
		return nil // 아무것도 못 하면 아무것도 광고하지 않는다
	}

	// ★ 오케스트레이션 노드는 agent.reason 을 광고하지 않는다 ★ (ADR-022 §5)
	//
	// 어휘를 나눈 이유가 ★ 배제 ★ 이기 때문이다 — 속성은 부분집합 매칭이라
	// 평범한 Run 의 requires: agent.reason 이 통과해 이 노드를 잡아가고,
	// 임대 키가 (노드)라 그 순간 오케스트레이션이 막힌다.
	//
	// ★ 그리고 arch 를 뺀다 ★ — 이 노드는 Mediator 머신에 놓이는 것을 전제하는데
	// INVARIANTS 가 "명령 단계에는 파일시스템 경계가 없다" 이므로 그 기계에서
	// 명령 단계가 돌면 ★ DB·아티팩트·토큰에 무경계 argv 가 닿는다 ★.
	// arch 가 없으면 빌드 계약이 이 노드를 ★ 못 고른다 ★ — 광고를 좁히는 것으로
	// 배치 위험이 닫힌다. ★ 새 코드 0 개 ★ 인 방어다.
	if l.Orchestration {
		delete(attrs, "arch")
		if len(attrs) == 0 {
			return nil // 하네스도 없으면 오케스트레이션도 못 한다
		}
		return []contract.Capability{{
			Capability: contract.CapabilityOrchestration, Attrs: attrs}}
	}
	return []contract.Capability{{Capability: contract.CapabilityAgentReason, Attrs: attrs}}
}

func detectArch(l Local) string {
	if l.Arch != "" {
		return l.Arch
	}
	for arch, cc := range map[string]string{
		"armv7": "arm-linux-gnueabihf-gcc",
		"arm64": "aarch64-linux-gnu-gcc",
	} {
		if _, err := exec.LookPath(cc); err == nil {
			return arch
		}
	}
	return ""
}

func hasRoom(l Local, log *slog.Logger) (bool, uint64) {
	path := l.Workspace
	if path == "" {
		return true, 0
	}
	free, err := freeBytes(path)
	if err != nil {
		log.Warn("디스크 여유를 못 재서 있다고 본다", "path", path, "err", err)
		return true, 0
	}
	freeGB := free / (1 << 30)
	return freeGB >= uint64(l.MinFreeGB), freeGB
}
