package enode

import (
	"context"
	"errors"
	"log/slog"
	"os/exec"
	"runtime"

	"github.com/taeels/enode/internal/contract"
)

// Detect 는 이 기계가 지금 무엇을 할 수 있는지 알아낸다 (ADR-012).
//
// 광고는 델타가 아니라 매번 전부다 그래서 이 함수의 결과가 곧 능력의 전부이고,
// 못 하는 것을 빼고 보내는 것이 "지금은 못 한다" 를 표현하는 방법이다
// (ADR-017 결정 3). Mediator 는 아무것도 새로 알 필요가 없다.
//
// capability 어휘는 agent.reason 하나뿐이므로(ADR-019) 구별은 전부 속성이 한다.
func Detect(l Local, log *slog.Logger) []contract.Capability {
	attrs := map[string]string{}

	// 이 기계가 무엇인가 (ADR-055) — 계약이 고르는 데도 쓰이고,
	// 계획이 명령을 짓는 데도 쓰인다.
	//
	// 없어서 밟았다 (vm-scratch-1..5): 계획이 매 판 uname · sw_vers 를
	// 돌려 알아냈고, 그 답을 보려면 판이 하나 더 필요했다. 그리고
	// enode-macos-… 묶음 안의 linux/arm64 바이너리를 끝내 못 알아봤다 —
	// 무엇이 필요한지(VM 은 linux/aarch64) 아는 통로가 없었다.
	//
	// arch 와 다르다 — 아래의 arch 는 빌드 대상 이다(크로스 툴체인이
	// 있으면 그 아키텍처를 광고한다). 이것은 이 프로세스가 도는 기계다.
	attrs["os"] = runtime.GOOS
	attrs["host_arch"] = runtime.GOARCH

	// 추론 하네스가 있나 — Probe() 가 곧 executable resolve 다 (R3).
	// 없으면 광고에 안 실리고 → 후보에서 빠지고 → 계약이 요구하면 422 다.
	// "설치 안 된 하네스는 실행 안 한다" 가 별도 코드 없이 성립한다.
	for _, h := range harnesses {
		bin := l.HarnessBin
		if h.Name() != "claude" {
			bin = "" // 지금은 claude 만 덮어쓸 수 있다
		}
		ver, err := h.Probe(context.Background(), bin)
		if err != nil {
			// 「있는데 못 쓴다」는 조용히 빠지면 안 된다 (ADR-059)
			//
			// 없는 것은 당연한 일이라 로그가 필요 없다 — 그 기계에 안 깔았을 뿐이다.
			// 그런데 깔려 있는데 못 쓰는 것은 사람이 고칠 수 있는 문제다.
			// 알려주지 않으면 "왜 매칭이 안 되지" 로 남는다.
			if errors.Is(err, errNotUsable) {
				log.Warn("harness is installed but not usable; "+
					"dropping it from the advertisement",
					"harness", h.Name(), "err", err)
			}
			continue
		}
		attrs["harness"] = h.Name()
		// 버전은 광고에 안 싣는다 — 매처는 동등 비교뿐이라
		// "2.1.236 (Claude Code)" 같은 문자열은 매칭에 못 쓰고 공간만 더럽힌다.
		// 버전이 필요한 이유는 기록 이므로 HarnessResult 로 간다
		// (ADR-005 성질 4 — 봉인된 묶음만 보고 알 수 있어야 한다).
		_ = ver
		break
	}

	// 워크스페이스가 있으면 저장소를 유도한다. 사람이 주소를 안 적는다
	//
	// 유도가 이긴다 — .repo · .git 이 있으면 workspace_id 는 무시한다.
	// 사람이 적은 것이 기계가 본 것을 이기면 둘이 어긋났을 때 조용히 틀린다.
	if l.Workspace != "" {
		// 워크스페이스가 어디인가 (ADR-055) — 계획이 「파일을 어디에
		// 둘 수 있나」를 알아야 명령을 지을 수 있다. 조사로 알아내려면
		// 판이 하나 든다.
		//
		// 매칭에도 쓰이지만 주된 값은 정보다 — 계약이 경로로 노드를
		// 고르는 일은 드물고, 계획이 그 경로를 쓰는 일은 매번 있다.
		attrs["ws"] = l.Workspace
		if repo := DetectRepo(l.Workspace); repo != "" {
			attrs["repo"] = repo
		} else if l.WorkspaceID != "" {
			// 유도할 수 없는 워크스페이스 — 사람이 적은 이름을 쓴다 (ADR-036).
			attrs["repo"] = l.WorkspaceID
		}
	}

	// 빌드 능력 — 툴체인이 있고 디스크가 남아 있을 때만 광고한다.
	if arch := detectArch(l); arch != "" {
		if ok, free := hasRoom(l, log); ok {
			attrs["arch"] = arch
		} else {
			log.Warn("not enough free disk; dropping build capability from the advertisement",
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

	// 사람이 붙인 이름표 — 탄력 노드가 자기 몫을 구별하는 자리.
	//
	// 탐지한 것을 못 덮는다 — 이미 있는 키는 건너뛴다. 사람이 적은 것이
	// 기계가 본 것을 이기면 둘이 어긋났을 때 조용히 틀린다.
	// (같은 이유로 detect 는 repo 에서도 유도를 우선한다.)
	for k, v := range l.Labels {
		if _, taken := attrs[k]; taken {
			log.Warn("label ignored; detection already set this attribute",
				"key", k, "detected", attrs[k], "label", v)
			continue
		}
		attrs[k] = v
	}

	// os · host_arch 만으로는 능력이 아니다 — 어느 기계에나 있다.
	// 하나도 할 줄 아는 것이 없으면 광고하지 않는다(오늘 그대로).
	if !hasCapability(attrs) {
		return nil
	}

	// 오케스트레이션 노드는 agent.reason 을 광고하지 않는다 (ADR-022 §5)
	//
	// 어휘를 나눈 이유가 배제 이기 때문이다 — 속성은 부분집합 매칭이라
	// 평범한 Run 의 requires: agent.reason 이 통과해 이 노드를 잡아가고,
	// 임대 키가 (노드)라 그 순간 오케스트레이션이 막힌다.
	//
	// 그리고 arch 를 뺀다 — 이 노드는 Mediator 머신에 놓이는 것을 전제하는데
	// INVARIANTS 가 "명령 단계에는 파일시스템 경계가 없다" 이므로 그 기계에서
	// 명령 단계가 돌면 DB·아티팩트·토큰에 무경계 argv 가 닿는다.
	// arch 가 없으면 빌드 계약이 이 노드를 못 고른다 — 광고를 좁히는 것으로
	// 배치 위험이 닫힌다. 새 코드 0 개인 방어다.
	if l.Orchestration {
		delete(attrs, "arch")
		if !hasCapability(attrs) {
			return nil // 하네스도 없으면 오케스트레이션도 못 한다
		}
		return []contract.Capability{{
			Capability: contract.CapabilityOrchestration, Attrs: attrs}}
	}
	return []contract.Capability{{Capability: contract.CapabilityAgentReason, Attrs: attrs}}
}

// hasCapability 는 이 속성 묶음이 무언가 할 줄 안다고 말하는가다.
//
// os · host_arch · ws 는 어느 기계에나 있는 사실이지 능력이 아니다.
// 이것들만 남으면 "아무것도 못 한다" 이고, 그때는 광고하지 않는다 —
// 광고가 곧 능력이라는 ADR-012 의 뜻을 지킨다.
func hasCapability(attrs map[string]string) bool {
	for k := range attrs {
		switch k {
		case "os", "host_arch", "ws":
		default:
			return true
		}
	}
	return false
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
		log.Warn("cannot measure free disk; assuming enough", "path", path, "err", err)
		return true, 0
	}
	freeGB := free / (1 << 30)
	return freeGB >= uint64(l.MinFreeGB), freeGB
}
