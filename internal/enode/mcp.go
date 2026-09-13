package enode

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// 하네스 구성요소 — 이 단계가 무엇을 열지는 exec 앞에서 정해진다
//
// 파일이 아니라 메모리다. resolveComponents 가 실패해도 아무것도 안 남고,
// 쓰는 쪽(Instrument)은 정책을 다시 판단하지 않는다 — 이미 정해진 것을 쓴다.
//
//	resolveComponents   정한다.  파일을 하나도 안 만진다
//	Instrument          쓴다.    무엇을 열지 안 정한다
//
// U1 은 「요청도 팩도 없는 경우」만 다룬다. 출처 셋을 합치는 것은 U4 이고
// 팩은 U5 다. 그래서 이 파일의 제품 경로는 언제나 빈 목록을 낸다.

// MCPServer 는 노드가 선언한 MCP 서버 하나다 (ADR-035 §4.4).
//
// stdio 와 remote 를 한 구조체가 받는다 — 하네스가 읽는 형식과 같아서
// 허용목록으로 옮겨 적는 것이 복사가 된다.
type MCPServer struct {
	Command    string   `yaml:"command,omitempty"`    // stdio
	Args       []string `yaml:"args,omitempty"`       // stdio
	URL        string   `yaml:"url,omitempty"`        // remote
	Credential string   `yaml:"credential,omitempty"` // remote. 환경변수 이름이다. 값이 아니다

	// Env 는 이름에서 이름으로 간다. 값이 아니다.
	//
	// 키는 하네스가 서버에게 줄 변수의 이름이고, 값은 노드 환경에서 그 값을
	// 길어 올 변수의 이름이다. 허용목록에는 ${이름} 참조로 나가고 실제 값은
	// R1 화이트리스트가 노드 환경으로 넘긴 것이다 — 그래서 값이 파일에 안 남는다.
	//
	// 검증은 config.go 의 validateEnvNames 가 한다. 여기서 안 하는 이유는
	// 이 형식이 노드 선언과 팩(U5) 둘 다에 쓰이는데, 팩의 것은 다른 시점에
	// 다른 문구로 거절되기 때문이다.
	Env map[string]string `yaml:"env,omitempty"`

	// Extra 는 우리가 모르는 키다 — 그대로 허용목록에 옮긴다 (decisions.md 2절).
	//
	// 왜 필드가 필요한가 — yaml 은 모르는 키를 말없이 버린다. 담을 자리가
	// 없으면 그 결정이 코드로 안 선다. 하네스가 키를 늘릴 때(transport ·
	// headers 같은) 우리 판을 갈아끼우지 않고 지나가는 길이 이 필드다.
	//
	// yaml:"-" 인 것은 읽기를 UnmarshalYAML 이 직접 하기 때문이다. 쓰기는
	// 안 짓는다 — setup 이 설정을 새로 지어 쓰기만 하고 읽어서 다시 쓰는
	// 경로가 저장소에 0 이라 오늘 잃을 것이 없다. 그 0 이 깨지는 날
	// MarshalYAML 이 함께 서야 한다.
	Extra map[string]any `yaml:"-"`
}

// knownMCPKeys 는 UnmarshalYAML 이 Extra 에서 덜어낼 이름이다.
//
// 구조체의 태그와 이 목록이 갈리면 아는 키가 Extra 로도 들어가 허용목록에
// 두 번 적힌다. 시험이 둘을 대조한다.
var knownMCPKeys = []string{"command", "args", "url", "credential", "env"}

// UnmarshalYAML 은 같은 노드를 두 번 푼다 (U3 · decisions.md 2절).
//
// 한 번만 풀면 둘 중 하나를 잃는다. 구조체로만 풀면 모르는 키가 사라지고,
// map 으로만 풀면 yaml 의 종류 검사가 사라져 args: "--x" 가 오류 대신
// 문자열로 들어온다. 그 검사는 우리가 다시 짤 것이 아니라 지킬 것이다.
func (s *MCPServer) UnmarshalYAML(value *yaml.Node) error {
	// shadow 는 이 메서드를 안 갖는 같은 모양이다 — 안 그러면 Decode 가
	// 자기 자신을 다시 불러 무한히 돈다.
	type shadow MCPServer
	var known shadow
	if err := value.Decode(&known); err != nil {
		return err
	}
	var all map[string]any
	if err := value.Decode(&all); err != nil {
		return err
	}
	for _, k := range knownMCPKeys {
		delete(all, k)
	}
	*s = MCPServer(known)
	if len(all) > 0 {
		s.Extra = all
	}
	return nil
}

// kind 는 이 선언이 어느 종류인가다. 로그가 읽는다.
//
// command 로 가른다 — validateMCP 가 둘 다 적힌 선언을 이미 거절하므로
// 이 갈래는 모호를 안 만난다.
func (s MCPServer) kind() string {
	if s.Command != "" {
		return "stdio"
	}
	return "remote"
}

// mcpUp 은 이 서버가 지금 뜨겠는가다 (FR-3 · ADR-035 §4.4).
//
// 실제 연결은 안 한다 — 광고마다 남의 서버를 두드리면 그 서버가 죽을 때
// 함대가 함께 죽는다 (requirements.md 4.4 의 순연).
//
// 「뜨나」는 존재이지 동작이 아니다. 셋을 못 잡는다 (decisions.md 6절 ⑭):
//
//	못 잡는다   자격증명이 틀렸다          환경변수 이름만 본다. 값을 안 본다
//	못 잡는다   엔드포인트가 죽었다        접근을 안 한다
//	못 잡는다   실행은 되나 MCP 가 아니다   CA2 의 가짜 서버 true 가 그 경우다
//
// 돌려주는 오류가 곧 노드 로그의 사유다 — 빠진 이유를 사람이 읽는다.
func mcpUp(s MCPServer) error {
	if s.Command != "" {
		if _, err := exec.LookPath(s.Command); err != nil {
			if strings.ContainsRune(s.Command, filepath.Separator) {
				return fmt.Errorf("command %q is not an executable file", s.Command)
			}
			return fmt.Errorf("command %q not found in PATH", s.Command)
		}
		return nil
	}
	// 인증 없는 endpoint 가 있다. 이름을 안 적었으면 볼 것이 없고, 볼 것이
	// 없는 것은 "안 뜬다" 가 아니다.
	if s.Credential == "" {
		return nil
	}
	if _, ok := os.LookupEnv(s.Credential); !ok {
		return fmt.Errorf("credential %s is not set in the node environment", s.Credential)
	}
	return nil
}

// mcpFP 는 뜨는 서버만 광고 속성으로 낸다 (FR-3).
//
// Fingerprinter 의 한 종류다. 프로세스를 안 띄우지만 호출 수가 노드 설정에
// 비례하므로 값싼 쪽에 안 둔다 (requirements.md 4.4 · components.md 2.2).
type mcpFP struct{}

func (mcpFP) Kind() string { return "mcp" }

func (mcpFP) Probe(_ context.Context, l Local, log *slog.Logger) (map[string]string, error) {
	if len(l.MCP) == 0 {
		// 선언이 없는 것은 정상이다. 로그도 안 낸다.
		return nil, nil
	}
	// 이름 순으로 돈다 — 같은 설정이면 같은 로그 순서다.
	names := make([]string, 0, len(l.MCP))
	for name := range l.MCP {
		names = append(names, name)
	}
	sort.Strings(names)

	attrs := map[string]string{}
	for _, name := range names {
		s := l.MCP[name]
		if err := mcpUp(s); err != nil {
			// 안 뜨는 서버는 이 종류의 실패가 아니라 정상적인 결과다 —
			// 빼고 보내는 것이 "지금은 못 한다" 다 (ADR-017 결정 3).
			// 그래도 조용히 빼지 않는다: 사람이 고칠 수 있는 문제다.
			log.Warn("mcp server is not up; dropping it from the advertisement",
				"server", name, "kind", s.kind(), "err", err)
			continue
		}
		// 값은 언제나 "1" 이다 — 매처가 완전 일치만 보므로 버전이나 경로를
		// 실으면 계약이 그 글자를 맞춰 적어야 한다.
		attrs["mcp."+name] = "1"
	}
	return attrs, nil
}

// Components 는 이 단계가 하네스에 실어 줄 것 전부다.
type Components struct {
	Servers map[string]MCPServer // 허용목록에 실릴 것
	Pack    *Pack                // nil 이면 팩 없음
	Notes   []string             // 로그로 낼 사실 — 이름만 담는다
}

// Pack 은 검증을 통과한 팩이다 (component-methods.md 2.1).
//
// U1 은 이름만 세운다 — 필드도 읽는 코드도 U5 가 짓는다. Components 가 이
// 형식을 가리키므로 이름이 먼저 서야 한다. 여기 필드를 미리 만들지 않는 이유는
// 상한 값과 tar 규약이 U5 의 Functional Design 이 닫을 자리이기 때문이다.
type Pack struct{}

// resolveComponents 는 이 단계가 무엇을 열지 정한다 (FR-2 · FR-4 · FR-6).
//
// 파일을 하나도 안 만진다. 그래서 시험이 하네스도 디스크도 안 쓰고,
// 오류가 곧 단계 실패다 — 부르는 쪽이 exec 전에 부른다.
//
// U1 은 요청도 팩도 없는 경우만 다루므로 언제나 빈 것을 낸다. 출처 셋
// (팩 · 노드 · 워크스페이스)과 이름이 없을 때의 거절은 U4 가, 팩은 U5 가
// 이 자리에 자란다. 빈 맵을 세워 돌려주는 것은 쓰는 쪽이 nil 과 빈 것을
// 안 가르게 하려는 것이다.
func resolveComponents(j Job) (Components, error) {
	_ = j // 출처는 U4 가 여기서 읽는다 — 오늘은 읽을 것이 없다
	return Components{Servers: map[string]MCPServer{}}, nil
}

// mcpAllowlistName 은 허용목록 파일의 이름이다 (decisions.md 2절).
//
// 가짜 홈 밖에 둔다 — 홈 안에 두면 하네스가 설정으로도 읽을 수 있고,
// 우리가 --mcp-config 로 가리키는 것과 그것이 갈릴 자리가 생긴다.
const mcpAllowlistName = "mcp.json"

// writeMCPAllowlist 는 허용목록 파일을 쓴다 (features.md 3.2).
//
// 권한은 0600 이다 — 이름만 든 파일이지만 계장 안의 다른 파일과 같은 값으로 둔다.
func writeMCPAllowlist(path string, servers map[string]MCPServer) error {
	b, err := mcpAllowlistJSON(servers)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

// mcpAllowlistJSON 은 서버 맵을 mcp.json 한 줄로 만든다.
//
// 봉투를 언제나 쓴다 — 빈 것도 {"mcpServers":{}} 이고 {} 가 아니다.
// --strict-mcp-config 는 「이 파일에 적힌 것만」이고 그 「적힌 것」의 자리가
// mcpServers 다. 키가 없는 파일을 하네스가 어떻게 읽는지는 실측이 없고,
// 봉투를 언제나 쓰면 그 물음이 사라진다. 그리고 0 이 의도임이 파일에 보인다
// (features.md 3.2 — 요청이 없을 때 0 은 의도다).
func mcpAllowlistJSON(servers map[string]MCPServer) ([]byte, error) {
	m := make(map[string]map[string]any, len(servers))
	for name, s := range servers {
		m[name] = s.allowlistEntry()
	}
	b, err := json.Marshal(struct {
		MCPServers map[string]map[string]any `json:"mcpServers"`
	}{m})
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// allowlistEntry 는 서버 하나를 mcp.json 의 객체로 만든다.
//
// 비면 키를 안 쓴다. command: "" 를 적어 보내면 하네스가 그것을 실행 경로로
// 읽을 수 있고, 그 실패는 우리 파일이 만든 것이 된다. 빈 값은 「안 적었다」이지
// 「빈 것을 적었다」가 아니다.
//
// Credential 은 안 나간다 — 하네스가 모르는 키이고(우리 어휘다), remote 인증은
// 노드 환경변수로 가므로 이름을 파일에 적어도 하네스가 그것으로 하는 일이 없다.
// 그 이름이 사는 자리는 mcpUp 의 remote 판정 하나다.
//
// Extra 를 먼저 얹고 아는 키를 그 위에 덮는다. 아는 키가 이겨야 소유자가 적은
// 선언과 우리가 읽은 선언이 안 갈린다 (decisions.md 2절).
func (s MCPServer) allowlistEntry() map[string]any {
	e := map[string]any{}
	for k, v := range s.Extra {
		e[k] = v
	}
	if s.Command != "" {
		e["command"] = s.Command
	}
	if len(s.Args) > 0 {
		e["args"] = s.Args
	}
	if s.URL != "" {
		e["url"] = s.URL
	}
	if len(s.Env) > 0 {
		e["env"] = envRefs(s.Env)
	}
	return e
}

// envRefs 는 이름을 참조로 바꾼다 (features.md 3.2).
//
// 값을 적지 않는다 — 값은 R1 화이트리스트가 노드 환경으로 넘기고 하네스가
// 그 참조를 편다. validateEnvNames 가 값 자리를 환경변수 이름으로 보장하므로
// 여기서 다시 안 본다.
func envRefs(env map[string]string) map[string]string {
	out := make(map[string]string, len(env))
	for k, v := range env {
		out[k] = "${" + v + "}"
	}
	return out
}
