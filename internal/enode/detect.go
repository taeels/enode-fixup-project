package enode

import (
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

	// 추론 하네스가 있나
	if _, err := exec.LookPath("claude"); err == nil {
		attrs["harness"] = "claude"
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
