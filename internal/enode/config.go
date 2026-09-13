package enode

import (
	"errors"
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

// Local 은 그 기계에서만 아는 것이다 (ADR-012).
// 중앙에는 아무것도 안 적는다 — 그것이 LAVA 와 갈리는 자리다.
// 열 줄을 넘기지 않는 것이 원칙이다.
type Local struct {
	Mediator string `yaml:"mediator"`
	Token    string `yaml:"token"`

	// Principal 은 이 노드가 누구의 것인가다 (ADR-015 §1). 비우면 git 을 본다.
	//
	// 왜 자리가 생겼나 — ADR-015 는 git 전역 설정의 이메일을 정본으로 삼으면서
	// "커널 개발자는 예외 없이 user.email 을 설정해 두었다. 이미 있는 것을
	// 읽는다" 를 근거로 들었다. 그 전제가 안 서는 기계가 있다.
	//
	//	실측 (2026-09-06) git 이 안 깔린 윈도우 노트북에서 노드가 신원을
	//	못 만들고 그 자리에서 죽었다. enodectl 은 "start failed" 만 냈다.
	//	그 노드는 추론만 하므로 git 이 할 일이 애초에 없었다.
	//
	// 조용한 대체가 아니다 — ADR-015 가 막은 것은 $USER 나 hostname 으로
	// 몰래 채우는 것이고, 그 비교 목록에 「사람이 적는다」가 없었다.
	// 여기 적힌 값은 사람이 고른 것이라 어긋나도 어디를 볼지가 분명하다.
	//
	// 적혀 있으면 이것이 이긴다 — repo 와 반대 방향인데 근거가 같다.
	// repo 는 기계가 관찰하는 사실이라 유도가 이기고, 신원은 사람의 것이라
	// 사람이 이긴다. 한 기계의 git 전역 설정이 그 기계 노드 전부의 신원을
	// 강제하면 팀 공용 노드를 세울 자리가 없다.
	Principal string `yaml:"principal,omitempty"`

	// 이 enode 가 서 있는 워크스페이스. 경로가 곧 신원의 일부다 (ADR-017).
	// repo canonical id 는 여기서 유도한다 — 사람이 저장소 주소를 안 적는다.
	Workspace string `yaml:"workspace"`

	// WorkspaceID 는 자동 유도가 실패할 때만 적는다.
	//
	// DetectRepo 가 .repo · .git 에서 신원을 유도하지만, 둘 다 없는 워크스페이스
	// (풀어놓은 소스 트리 · 문서 디렉터리 · 파일서버 마운트)는 기계가 알아낼 방법이
	// 없다. 그러면 repo 속성이 안 실리고 그런 노드 둘이 매처에게 똑같아 보인다.
	//
	// Board 와 같은 자리다 (ADR-012) — "포트에 무엇이 달렸는지는 기계가 모른다".
	// 그래서 「사람이 저장소 주소를 안 적는다」의 예외가 되지만, 이유가 같다.
	// 유도가 성공하면 이 값은 무시된다 — 사람이 적은 것이 기계가 본 것을 못 이긴다.
	WorkspaceID string `yaml:"workspace_id,omitempty"`

	// Labels 는 사람이 붙이는 이름표다 — 그대로 광고에 실린다.
	//
	// 왜 필요한가 — 탄력 노드가 자기 몫의 Run 만 잡으려면 구별할 것이
	// 있어야 한다. 이슈마다 오케스트레이터가 서면 열 개가 동시에 뜰 수 있고,
	// 능력만으로는 서로가 똑같아 보인다.
	//
	//	labels: { issue: PROJ-42 }
	//	→ 광고에 issue=PROJ-42 로 실린다
	//	→ requires: [{as: planner, capability: orchestration, issue: PROJ-42}]
	//	→ 다른 이슈의 오케스트레이터는 안 걸린다
	//
	// ADR-012 가 「속성 어휘는 창발한다」로 이미 열어둔 자리다 —
	// 매처는 부분집합 비교뿐이라 새 이름을 알 필요가 없다.
	//
	// 기계가 잰 것을 못 이긴다 — 탐지한 속성과 이름이 겹치면 탐지가 이긴다.
	// 사람이 적은 것이 기계가 본 것을 이기면 둘이 어긋났을 때 조용히 틀린다
	// (detect.go 가 repo 에 대해 적은 것과 같은 규칙).
	Labels map[string]string `yaml:"labels,omitempty"`

	// MCP 는 이 기계가 가진 MCP 서버다 (ADR-012 · ADR-035 §4.4).
	//
	// 중앙에 안 적는다 — 그 기계에서만 아는 것이다. 여기 없는 서버는
	// 광고되지 않고, 광고되지 않으면 계약이 그 노드를 못 고른다.
	// Board 와 같은 자리다 — 포트에 무엇이 달렸는지를 기계가 모르듯,
	// 어느 사내 MCP 가 이 기계에 물려 있는지도 기계가 모른다.
	//
	// 맵의 키가 서버의 이름이고 그 이름이 곧 광고 키의 꼬리다 —
	// probe 를 적으면 mcp.probe 로 광고된다. 계약이 그 글자를 그대로 적는다.
	MCP map[string]MCPServer `yaml:"mcp,omitempty"`

	// 자동으로 못 알아내는 것만 적는다. 포트에 무엇이 달렸는지는 기계가 모른다.
	Board *Board `yaml:"board,omitempty"`

	// 크로스 툴체인 탐지가 애매할 때만 명시한다. 비우면 자동 탐지한다.
	Arch string `yaml:"arch,omitempty"`

	// HarnessBin 은 하네스 실행 파일이다. 비우면 claude.
	// 시험용 스텁을 가리키게 할 수 있다.
	HarnessBin string `yaml:"harness_bin,omitempty"`

	// 이 값 아래로 떨어지면 빌드 능력을 광고에서 뺀다 (ADR-017 결정 3).
	// 매칭 조건이 아니라 광고 조건이다 — "할 수 있는가" 는 노드가 판단한다.
	MinFreeGB int `yaml:"min_free_gb,omitempty"`

	// Orchestration 은 이 노드가 계약을 짓는 자리 라는 선언이다 (ADR-022 §5).
	//
	// 사람이 적는 이유 — 기계가 알아낼 수 없다. 하네스가 있다는 사실만으로는
	// "이 노드에 오케스트레이션을 맡겨도 되는가" 를 판정할 수 없고, 그건
	// 노드 소유자의 결정이다. ADR-012 가 "포트에 무엇이 달렸는지는 기계가
	// 모른다 — 그 기계에만 적는다" 로 board 를 다룬 것과 같은 자리다.
	Orchestration bool `yaml:"orchestration,omitempty"`
}

type Board struct {
	SoC  string `yaml:"soc"`
	Tag  string `yaml:"tag"`
	Port string `yaml:"port"`
}

func LoadLocal(path string) (Local, error) {
	var l Local
	b, err := os.ReadFile(path)
	if err != nil {
		return l, fmt.Errorf("config %s: %w", path, err)
	}
	if err := yaml.Unmarshal(b, &l); err != nil {
		return l, fmt.Errorf("config %s: %w", path, err)
	}
	if l.MinFreeGB == 0 {
		l.MinFreeGB = 10
	}
	// 검증은 기본값 다음이다 — 둘은 무관하고, 검증이 먼저 죽으면 그 기본이
	// 안 채워진 채로 오류만 나간다.
	if err := validateMCP(l.MCP); err != nil {
		return Local{}, fmt.Errorf("config %s: %w", path, err)
	}
	return l, nil
}

// validateMCP 는 mcp: 절의 아는 키의 모양을 본다 (decisions.md 2절 · US-2).
//
// 왜 필요한가 — yaml 이 잡는 것은 종류뿐이다. command: 5 는 조용히 "5" 가 되고,
// 빈 서버도 command 와 url 이 둘 다인 선언도 그대로 통과한다. 그래서
// "아는 키의 모양은 검증한다" 를 세는 코드가 따로 있어야 한다.
//
// 여섯이 하는 일은 하나다 — 어느 종류인가를 한 값으로 만든다. mcpUp 이 그
// 종류로 갈라 판정하므로, 모호한 선언이 남으면 판정 코드가 어느 갈래로 갈지를
// 스스로 정하게 되고 그 정함은 설정 파일에 안 보인다.
//
// 여기서 오류가 나면 노드가 안 뜬다 — cmd/enode 가 LoadLocal 의 오류를
// cannot read config 로 내고 죽는다. 틀리면 닫히는 쪽으로 틀린다.
//
// 검사 순서가 문구를 정한다. 좁은 것을 먼저 본다 — args 만 적힌 서버에
// "needs command or url" 을 주면 사람이 args 를 지우려 하고, 실은 command 를
// 빠뜨린 것이다.
//
// 이름 순으로 돈다. 같은 설정이면 같은 문구가 나와야 한다 (ADR-014 결정 3 의 결).
func validateMCP(servers map[string]MCPServer) error {
	names := make([]string, 0, len(servers))
	for name := range servers {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if name == "" {
			return errors.New("mcp: a server name must not be empty")
		}
		s := servers[name]
		switch {
		case s.Command != "" && s.URL != "":
			return fmt.Errorf("mcp server %q: command and url are both set; "+
				"a server is either stdio or remote", name)
		case s.Command == "" && len(s.Args) > 0:
			return fmt.Errorf("mcp server %q: args without command", name)
		case s.URL == "" && s.Credential != "":
			return fmt.Errorf("mcp server %q: credential without url", name)
		case s.Command == "" && s.URL == "":
			return fmt.Errorf("mcp server %q: needs command (stdio) or url (remote)", name)
		}
		if err := validateEnvNames(name, s.Env); err != nil {
			return err
		}
	}
	return nil
}

// validateEnvNames 는 env 의 값이 이름인지 본다 (features.md 3.2).
//
// 값 자리에 오는 것은 노드 환경에서 그 값을 길어 올 변수의 이름이다.
// 값을 직접 적으면 그것이 계장 디렉터리의 허용목록 파일에 그대로 남는다 —
// 그 파일에는 이름만 적는다는 것이 이 팩의 값이다.
//
// 꼴 검사는 "값을 적지 말라" 를 다 못 잰다. 토큰이 우연히 이 꼴이면 통과한다.
// 그럼에도 두는 이유는 흔한 실수가 전부 걸리고, 걸린 사람이 문구에서 규칙을
// 배우기 때문이다.
//
// 키는 안 잰다 — 키는 하네스가 서버에게 줄 변수의 이름이고 그 어휘는 서버의 것이다.
func validateEnvNames(server string, env map[string]string) error {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !isEnvName(env[k]) {
			return fmt.Errorf("mcp server %q: env[%q] must name an environment variable, not a value",
				server, k)
		}
	}
	return nil
}

func isEnvName(v string) bool {
	if v == "" {
		return false
	}
	for i, r := range v {
		switch {
		case r == '_':
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9' && i > 0:
		default:
			return false
		}
	}
	return true
}

// SampleLocal 은 노드 설정의 최소 형태다. 설정을 못 찾았을 때 화면에 낸다 —
// 「없다」만 말하고 무엇을 만들어야 하는지는 안 말하면 사람이 또 찾아야 한다.
func SampleLocal() string {
	return `    mediator: http://<mediator-host>:8080
    token: "<the same token the mediator has>"
    workspace: <path to a checkout; omit it to advertise reasoning only>
    principal: <your email; omit it to read git config --global user.email>
    # mcp: { probe: { command: /opt/tools/probe-mcp } }   MCP servers only this machine has

`
}
