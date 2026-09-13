package enode

import (
	"context"
	"errors"
	"log/slog"
	"os/exec"
	"runtime"
	"strings"

	"github.com/taeels/enode/internal/contract"
)

// Detect 는 이 기계가 지금 무엇을 할 수 있는지 알아낸다 (ADR-012).
//
// 광고는 델타가 아니라 매번 전부다 그래서 이 함수의 결과가 곧 능력의 전부이고,
// 못 하는 것을 빼고 보내는 것이 "지금은 못 한다" 를 표현하는 방법이다
// (ADR-017 결정 3). Mediator 는 아무것도 새로 알 필요가 없다.
//
// capability 어휘는 agent.reason 하나뿐이므로(ADR-019) 구별은 전부 속성이 한다.
//
// 두 갈래를 한 번에 한다 (ADR-068)
//
// 알아내는 비용이 속성마다 다르다 — 어떤 것은 상수이고 어떤 것은 외부
// 프로세스를 띄운다. 광고 루프는 비싼 쪽을 직접 부르면 안 되므로(Detector 가
// 대신 든다), 이 함수는 둘을 함께 하는 형태로 남는다. 한 번만 도는 자리가
// 쓴다 — setup 과 Detector 의 첫 탐지가 그것이다.
func Detect(ctx context.Context, l Local, log *slog.Logger) []contract.Capability {
	return capabilities(l, log, costlyAttrs(ctx, l, log), cheapAttrs(l, log))
}

// cheapAttrs 는 외부 프로세스를 안 띄우고 알아내는 것이다 (ADR-068 §4 비용 축).
//
// 상수 · syscall · 설정값뿐이라 광고마다 새로 봐도 된다. 오히려 새로 봐야
// 한다 — 디스크 여유가 여기 있고, 그것이 동적이라는 것이 ADR-017 결정 3 의
// 근거였다. 빌드가 도는 동안 디스크가 차면 다음 광고에서 바로 빠져야 한다.
func cheapAttrs(l Local, log *slog.Logger) map[string]string {
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

	if l.Workspace != "" {
		// 워크스페이스가 어디인가 (ADR-055) — 계획이 「파일을 어디에
		// 둘 수 있나」를 알아야 명령을 지을 수 있다. 조사로 알아내려면
		// 판이 하나 든다.
		//
		// 매칭에도 쓰이지만 주된 값은 정보다 — 계약이 경로로 노드를
		// 고르는 일은 드물고, 계획이 그 경로를 쓰는 일은 매번 있다.
		attrs["ws"] = l.Workspace
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
	//
	// 여기는 탐지가 아니라 선언이다 — 사람이 적은 것을 그대로 싣는다.
	// 보드가 뽑혀도 광고에 남으므로, 매칭은 통과하고 실행 시점에 죽는다.
	// ADR-059 가 하네스에서 고친 것과 같은 모양이 여기 남아 있다.
	// 고치려면 살아있음을 물어봐야 하는데, 그 확인은 지금 일하는 Run 이 쥔
	// 포트를 여는 일이라 임대와 맞물려야 한다 (ADR-068 §4.1 · §5.1 순연).
	if l.Board != nil && l.Board.SoC != "" {
		attrs["board"] = l.Board.SoC
		if l.Board.Tag != "" {
			attrs["tag"] = l.Board.Tag
		}
	}
	return attrs
}

// Fingerprinter 는 비싼 사실 한 종류다 (ADR-035 §4.2).
//
// 왜 종류로 가르나 — 탐색기를 하네스 밖에 세우고 하네스도 그 한 종류로
// 만든다. ADR-035:246-248 이 「ADR-034 를 구현하면서 detect.go 를 탐지기
// 순회로 바꾸는 것까지는 오늘 동작을 안 바꾸고 된다」로 이 시점을 지목했다.
//
// 이름이 Detector 가 아닌 이유 — 그 이름은 ADR-068 의 주기 장치가 쓴다.
// Nomad 의 fingerprint 가 §4.2 가 든 유비이고 그 단어를 따른다.
type Fingerprinter interface {
	Kind() string // "harness" | "repo" | "mcp"

	// Probe 는 logger 를 받는다. costlyAttrs 안에서 돌던 log.Warn(ADR-059)과
	// FR-3 이 요구하는 서버별 누락 사유가 갈 자리다.
	//
	// 오류는 「못 한다」가 아니라 「못 물어봤다」다. 부르는 쪽이 그 종류가
	// 낸 것만 버리고 순회를 계속한다 — 못 물어본 것을 빼고 보내면 노드가
	// 스스로를 지운다 (ADR-012 의 「빼는 것이 못 한다는 뜻」이 거짓이 된다).
	Probe(ctx context.Context, l Local, log *slog.Logger) (map[string]string, error)
}

// fingerprinters 는 오늘 costlyAttrs 가 실제로 도는 비싼 사실 셋이다.
//
// ADR-035 §4.2 는 셋째로 toolchain 을 들었는데 여기는 repo 다 — 크로스
// 툴체인 탐지(detectArch)는 프로세스를 안 띄우고 호출 수도 상수라 값싼 쪽에
// 이미 앉아 있다 (decisions.md 6절 ⑤).
var fingerprinters = []Fingerprinter{harnessFP{}, repoFP{}, mcpFP{}}

// harnessFP 는 추론 하네스를 잰다 — Usable() 이 곧 executable resolve 다 (R3).
//
// 없으면 광고에 안 실리고 → 후보에서 빠지고 → 계약이 요구하면 422 다.
// "설치 안 된 하네스는 실행 안 한다" 가 별도 코드 없이 성립한다.
type harnessFP struct{}

func (harnessFP) Kind() string { return "harness" }

func (harnessFP) Probe(ctx context.Context, l Local, log *slog.Logger) (map[string]string, error) {
	attrs := map[string]string{}
	for _, h := range harnesses {
		bin := l.HarnessBin
		if h.Name() != "claude" {
			bin = "" // 지금은 claude 만 덮어쓸 수 있다
		}
		if err := h.Usable(ctx, bin); err != nil {
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
		// 버전은 광고에 안 싣는다 — 매처는 동등 비교뿐이라
		// "2.1.236 (Claude Code)" 같은 문자열은 매칭에 못 쓰고 공간만 더럽힌다.
		// 버전이 필요한 이유는 기록 이므로 HarnessResult 로 간다
		// (ADR-005 성질 4 — 봉인된 묶음만 보고 알 수 있어야 한다).
		//
		// 그래서 여기서 Version 을 안 부른다. 예전에는 Probe 하나가 둘을
		// 겸했고 돌려받은 버전을 그 자리에서 버렸는데, 버리는 값을 위해
		// 프로세스는 광고마다 그대로 떴다.
		attrs["harness."+h.Name()] = "1"

		// 옛 키는 첫 것으로 남긴다 (ADR-035 §4.3 · §6).
		//
		// break 가 여기서 걷혔다 — 종류로 가르면서 「하나만 보고 멈춘다」는
		// 제약이 사라졌다. 그래도 옛 키는 하나뿐이라 첫 것이 이긴다.
		// 걷는 날은 이 회차가 안 정한다 — 그날 매처의 attrCount 가 노드마다
		// 1 씩 줄어 정렬이 움직인다.
		if _, taken := attrs["harness"]; !taken {
			attrs["harness"] = h.Name()
		}
	}
	return attrs, nil
}

// repoFP 는 워크스페이스에서 저장소를 유도한다. 사람이 주소를 안 적는다.
//
// 유도가 이긴다 — .repo · .git 이 있으면 workspace_id 는 무시한다.
// 사람이 적은 것이 기계가 본 것을 이기면 둘이 어긋났을 때 조용히 틀린다.
type repoFP struct{}

func (repoFP) Kind() string { return "repo" }

func (repoFP) Probe(ctx context.Context, l Local, _ *slog.Logger) (map[string]string, error) {
	if l.Workspace == "" {
		return nil, nil
	}
	if repo := DetectRepo(ctx, l.Workspace); repo != "" {
		return map[string]string{"repo": repo}, nil
	}
	if l.WorkspaceID != "" {
		// 유도할 수 없는 워크스페이스 — 사람이 적은 이름을 쓴다 (ADR-036).
		// 이 갈래를 안 옮기면 git 없는 노드에서 repo 가 사라져 중립이 깨진다.
		return map[string]string{"repo": l.WorkspaceID}, nil
	}
	return nil, nil
}

// costlyAttrs 는 외부 프로세스를 띄워야 알아내는 것이다 (ADR-068 §4 비용 축).
//
// 광고 루프가 이것을 직접 부르면 안 된다 — 여기서 멈추면 하트비트가 함께
// 멈추고 노드가 조용히 함대에서 사라진다. Detector 가 자기 시계로 갱신한다.
//
// 종류를 순회하는 것으로 바뀌었고 동작은 중립이다 (ADR-035 §4.2).
// 한 종류가 실패해도 나머지는 실린다 — 부분 광고가 빈 광고보다 참에 가깝다.
func costlyAttrs(ctx context.Context, l Local, log *slog.Logger) map[string]string {
	attrs := map[string]string{}
	for _, fp := range fingerprinters {
		part, err := fp.Probe(ctx, l, log)
		if err != nil {
			log.Warn("fingerprint failed; its attributes are missing from this advertisement",
				"kind", fp.Kind(), "err", err)
			continue
		}
		// 키가 겹치면 나중 것이 이긴다. 셋의 키 공간이 안 겹치므로 오늘
		// 이 규칙이 걸리는 자리는 0 이고, 적어 두는 것은 종류가 느는 날
		// 조용히 갈리지 않게 하려는 것이다.
		for k, v := range part {
			attrs[k] = v
		}
	}
	return attrs
}

// capabilities 는 알아낸 조각들을 광고에 실을 모양으로 합친다.
//
// 조각을 나눈 것은 알아내는 비용 때문이고(ADR-068), 합치고 나면 광고가
// 무엇을 싣는지는 하나도 안 바뀐다 — ADR-012 의 「매번 전부」도,
// ADR-017 결정 3 의 「못 하면 뺀다」도 그대로다.
func capabilities(l Local, log *slog.Logger, parts ...map[string]string) []contract.Capability {
	attrs := map[string]string{}
	for _, part := range parts {
		for k, v := range part {
			attrs[k] = v
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
		warnMCPWithoutCapability(l, log)
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
			warnMCPWithoutCapability(l, log)
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
//
// mcp.<이름> 도 그것만으로는 능력이 아니다 (U3)
//
// MCP 서버는 하네스가 물어야 쓸 수 있는 것이다. 하네스가 없는 기계가
// 선언만으로 광고를 내면 계약이 requires: mcp.<이름> 으로 그 노드를 잡고,
// 실행 시점에 하네스가 없어 죽는다. 광고가 곧 능력인 곳에서 그것은
// 할 줄 모르는 것을 광고한 것이다.
//
// harness.<이름> 은 능력이다 — 그것이 물 수 있는 쪽이다.
func hasCapability(attrs map[string]string) bool {
	for k := range attrs {
		switch {
		case k == "os", k == "host_arch", k == "ws":
		case strings.HasPrefix(k, "mcp."):
		default:
			return true
		}
	}
	return false
}

// warnMCPWithoutCapability 는 선언은 있는데 능력이 0 일 때 그 사유를 낸다 (FR-3).
//
// 침묵이 이 결정의 가장 큰 대가다. 소유자는 mcp: 를 적었는데 함대에서 자기
// 노드가 아무것도 못 하는 것으로 보이고, 이유를 말해 주는 줄이 없으면 그
// 자리에서 멈춘다. 「없음이 실패보다 나쁘다」(ADR-035 §3)가 여기에도 걸린다.
//
// 광고 주기마다 나온다 — capabilities 가 광고마다 불린다. 탐지 주기(5분)보다
// 잦지만, 이것은 사람이 설정을 고쳐야 끝나는 상태다.
func warnMCPWithoutCapability(l Local, log *slog.Logger) {
	if len(l.MCP) > 0 {
		log.Warn("node declares mcp servers but advertises no capability; is a harness installed?",
			"mcp", len(l.MCP))
	}
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
