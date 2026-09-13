package enode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
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
// U4 가 출처 둘(노드 선언 · 워크스페이스)을 합쳤다. 남은 것은 팩이고 U5 다.
//
//	readWorkspaceMCP    읽는다.  가장자리다 — 정하지 않는다

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
//
// Servers 는 MCPServer 가 아니라 허용목록 항목이다 (U4).
//
// 출처마다 어휘가 다르기 때문이다 — 노드 선언은 우리가 정의한 MCPServer 이고,
// 워크스페이스 .mcp.json 과 팩의 mcp.json 은 하네스가 정의한 형식이다. 그 둘을
// MCPServer 로 받으면 우리가 모르는 키(type · headers)가 사라지고 env 의 뜻이
// 뒤집힌다 — 노드 선언의 env 는 이름에서 이름으로 가는데 그 파일들의 env 는 값이다.
//
// 최종 항목으로 올리면 합치는 자리에서 어휘가 하나가 된다. 노드 것은
// allowlistEntry() 를 지나 들어오고 그 밖의 출처는 원문 그대로 들어온다 —
// 번역하는 코드가 없으므로 번역이 못 틀린다.
type Components struct {
	Servers map[string]map[string]any // 이름 -> 허용목록 항목
	Pack    *Pack                     // nil 이면 팩 없음
	Notes   []string                  // 로그로 낼 사실 — 이름만 담는다
}

// Pack 은 검증을 통과한 팩이다 (component-methods.md 2.1).
//
// U1 은 이름만 세운다 — 필드도 읽는 코드도 U5 가 짓는다. Components 가 이
// 형식을 가리키므로 이름이 먼저 서야 한다. 여기 필드를 미리 만들지 않는 이유는
// 상한 값과 tar 규약이 U5 의 Functional Design 이 닫을 자리이기 때문이다.
type Pack struct{}

// resolveComponents 는 이 단계가 무엇을 열지 정한다 (FR-2 · FR-4 · FR-6).
//
// 파일을 하나도 안 만진다. 출처는 Job 이 들고 온다 — 여는 것은 가장자리
// (claim.go)이고 고르는 것은 여기다. 그래서 시험이 하네스도 디스크도 안 쓰고,
// 오류가 곧 단계 실패다 — 부르는 쪽이 exec 전에 부른다.
//
// 걸음 여섯이고 순서가 뜻을 가진다:
//
//	1  요청 집합을 만든다.  비면 빈 것을 낸다 — 출처를 아예 안 본다
//	2  워크스페이스 파일을 못 읽었으면 거절한다.  3 보다 앞이라 깨진 파일이
//	   「없는 이름」으로 둔갑하지 않는다
//	3  이름 순으로 출처를 뒤진다.  노드가 워크스페이스를 이긴다
//	4  집은 항목마다 종류를 채운다.  3 보다 뒤라 우리가 채운 것과 출처가 안 섞인다
//	5  요청 안 한 워크스페이스 이름을 Notes 에 남긴다
//	6  못 찾은 이름이 있으면 거절한다.  맨 뒤라 Notes 가 다 채워진 뒤다
//
// 거절로 끝날 때도 그때까지의 Components 를 함께 돌려준다 — 왜 실패했는지를
// 아는 데 필요한 사실이 실패와 함께 사라지면 안 된다. 부르는 쪽이 Notes 를
// 먼저 찍고 실패를 낸다 (runner.go 의 ②).
//
// 팩은 U5 가 걸음 3 앞에 더한다 — 팩이 노드 선언 이름을 덮으면 거절이기 때문이다.
//
// 빈 맵을 세워 돌려주는 것은 쓰는 쪽이 nil 과 빈 것을 안 가르게 하려는 것이다.
func resolveComponents(j Job) (Components, error) {
	c := Components{Servers: map[string]map[string]any{}}

	// 1 — 요청이 없으면 허용목록이 빈다. 출처가 무엇을 선언했든 그렇다
	// (features.md 3.2 — 요청이 없을 때 0 은 의도다).
	want := wantedMCP(j.Params.MCP)
	if len(want) == 0 {
		return c, nil
	}

	// 2 — 원인이 파일이면 문구도 파일을 가리킨다. 파일 경로는 안 싣는다:
	// 이 문구가 res.Error 로 봉인에 들어가고, 노드의 디렉터리 구조는 계약
	// 작성자가 알 것이 아니다. 파일은 하나뿐이라 이름으로 충분하다.
	if j.WorkspaceMCPErr != nil {
		return c, fmt.Errorf("cannot read workspace .mcp.json: %w", j.WorkspaceMCPErr)
	}

	// 3
	var missing []string
	for _, name := range want {
		if s, ok := j.NodeMCP[name]; ok {
			if _, dup := j.WorkspaceMCP[name]; dup {
				// 노드가 이긴다. 저장소에 쓰는 사람이 소유자가 선언한
				// 이름을 가로채지 못한다 — 매칭은 소유자의 값으로 하고
				// 실행은 저장소의 값으로 하는 길이 닫힌다.
				c.Notes = append(c.Notes, "mcp server "+name+" is declared by both "+
					"the node and the workspace; the node declaration wins")
			}
			c.Servers[name] = s.allowlistEntry()
			continue
		}
		if e, ok := j.WorkspaceMCP[name]; ok {
			// 원본을 안 고친다 — 4 가 키를 하나 더하는데 그 맵은 Job 의 것이다.
			e = copyEntry(e)
			if !hasText(e, "command") && !hasText(e, "url") {
				// 종류를 못 정하는 항목은 하네스가 말없이 버린다. 계약이
				// 이름으로 요청한 것이 조용히 사라지는 것은 아래 6 이 막으려는
				// 바로 그 실패다.
				return c, fmt.Errorf(
					"workspace mcp server %s declares neither command nor url", name)
			}
			c.Servers[name] = e
			continue
		}
		missing = append(missing, name)
	}

	// 4 — 종류를 채우는 자리가 하나다. 두 출처가 같은 규칙을 받는다.
	for _, e := range c.Servers {
		ensureType(e)
	}

	// 5 — 계약 작성자가 「적어 뒀는데 왜 없나」를 여기서 푼다.
	requested := make(map[string]bool, len(want))
	for _, name := range want {
		requested[name] = true
	}
	for _, name := range entryNames(j.WorkspaceMCP) {
		if !requested[name] {
			c.Notes = append(c.Notes, "workspace .mcp.json declares "+name+
				", which this step did not request")
		}
	}

	// 6 — 조용히 빼고 돌면 하네스가 exit 0 으로 끝나고 성공이 봉인된다.
	// 단계는 초록인데 모델은 그 도구를 못 봤다 (ADR-035 §3).
	if len(missing) > 0 {
		for _, name := range missing[1:] {
			c.Notes = append(c.Notes, notAvailable(name))
		}
		return c, errors.New(notAvailable(missing[0]))
	}
	return c, nil
}

// notAvailable 은 팩이 박은 문구다 (features.md 3.2 · scene-gates.md CA4).
//
// 게이트가 부분 문자열로 찾으므로 이름 하나짜리 단수형을 유지한다. 없는
// 이름이 여럿이면 첫 이름이 문구로 나가고 나머지는 Notes 로 간다.
func notAvailable(name string) string {
	return "mcp server " + name + " is not available on this node"
}

// wantedMCP 는 계약이 요청한 이름을 정렬된 유일한 목록으로 만든다.
//
// 같은 이름을 두 번 적은 계약을 거절하지 않는다 — 이름의 집합이 같으므로
// 결과가 같다. 정렬하는 이유는 거절 문구와 Notes 의 순서가 같은 계약이면
// 같아야 하기 때문이다 (ADR-014 결정 3 의 결).
func wantedMCP(names []string) []string {
	seen := make(map[string]bool, len(names))
	out := make([]string, 0, len(names))
	for _, n := range names {
		if seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// entryNames 는 항목 맵의 이름을 정렬해 낸다. 같은 파일이면 같은 순서다.
func entryNames(m map[string]map[string]any) []string {
	out := make([]string, 0, len(m))
	for name := range m {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// copyEntry 는 항목을 얕게 복사한다.
//
// ensureType 이 맵을 고치는데 그 맵은 Job 의 것이다 — 워크스페이스 파일에서
// 읽은 그대로다. 원본을 고치면 같은 Job 을 두 번 쓰는 호출자와 시험이 조용히
// 갈린다. 값은 안 복사한다 — 우리는 키 하나만 더하고 값은 안 만진다.
func copyEntry(e map[string]any) map[string]any {
	out := make(map[string]any, len(e)+1)
	for k, v := range e {
		out[k] = v
	}
	return out
}

// hasText 는 그 키에 빈 문자열이 아닌 문자열이 있는가다.
//
// 문자열이 아닌 값은 「없다」로 본다 — {"command": 5} 는 종류를 못 정하므로
// 우리가 채울 수도 없고, 사람이 고칠 자리는 같은 줄이다.
func hasText(e map[string]any, key string) bool {
	s, ok := e[key].(string)
	return ok && s != ""
}

// ensureType 은 항목에 종류를 채운다 (features.md 3.2 · decisions.md 6절 ㉓).
//
// 실측이 근거다 — claude 2.1.266 은 type 도 command 도 없는 항목을 init 줄의
// mcp_servers 에서 말없이 뺀다. failed 로도 안 나타난다. ADR-035 §4.4 의
// 예시(url + credential)가 그대로 그 모양이라, 그 선언은 광고는 서고 서버는
// 안 열리는 조합을 조용히 만든다 — ADR-035 §3 의 「없음이 실패보다 나쁘다」다.
//
// 이미 있으면 안 덮는다. 소유자가 type: sse 를 적었거나 저장소가 http 를
// 적었으면 그것이 이긴다 — 우리는 비어 있는 자리만 채운다.
//
// 종류를 잘못 채우는 경우는 남는다 (sse 서버를 url 만으로 적으면 http 가 된다).
// 그때 그 서버는 failed 로 목록에 나타난다 — 침묵이 아니라 실패다.
func ensureType(e map[string]any) {
	if _, ok := e["type"]; ok {
		return
	}
	switch {
	case hasText(e, "command"):
		e["type"] = "stdio"
	case hasText(e, "url"):
		e["type"] = "http"
	}
}

// workspaceMCPName 은 워크스페이스가 하네스에게 쓰는 파일이다 (features.md 3.4).
const workspaceMCPName = ".mcp.json"

// readWorkspaceMCP 는 워크스페이스의 선언을 원문 그대로 읽는다.
//
// 가장자리다 — 정하는 자리가 아니다. 등급도 문구도 resolveComponents 가
// 정하므로 오류를 그대로 올린다.
//
// 원문 그대로인 이유는 이 파일이 우리 형식이 아니기 때문이다. 저장소가
// 하네스에게 쓴 것이고 type · headers 처럼 우리 어휘에 없는 키가 있다.
// 우리는 전달자이고 번역자가 아니다.
//
//	없는 파일               nil.  저장소 대부분에 그 파일이 없다
//	mcpServers 가 없는 파일   nil.  다른 목적의 파일일 수 있다
//	그 밖의 실패             오류.  요청이 있을 때만 만난다 — claim.go 가
//	                       그때만 부른다
//
// 크기 상한을 안 둔다 — 같은 워크스페이스를 collectDeclared 와 워크스페이스
// diff 가 이미 통째로 읽는다. 여기만 상한을 두면 규율이 갈린다. 푼 뒤의
// 크기를 지는 것은 팩이다 (U5).
func readWorkspaceMCP(dir string) (map[string]map[string]any, error) {
	b, err := os.ReadFile(filepath.Join(dir, workspaceMCPName))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var f struct {
		MCPServers map[string]map[string]any `json:"mcpServers"`
	}
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	return f.MCPServers, nil
}

// mcpAllowlistName 은 허용목록 파일의 이름이다 (decisions.md 2절).
//
// 가짜 홈 밖에 둔다 — 홈 안에 두면 하네스가 설정으로도 읽을 수 있고,
// 우리가 --mcp-config 로 가리키는 것과 그것이 갈릴 자리가 생긴다.
const mcpAllowlistName = "mcp.json"

// writeMCPAllowlist 는 허용목록 파일을 쓴다 (features.md 3.2).
//
// 권한은 0600 이다 — 이름만 든 파일이지만 계장 안의 다른 파일과 같은 값으로 둔다.
func writeMCPAllowlist(path string, servers map[string]map[string]any) error {
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
func mcpAllowlistJSON(servers map[string]map[string]any) ([]byte, error) {
	// nil 도 빈 봉투다 — 아래 Marshal 이 nil 맵을 null 로 적는다.
	if servers == nil {
		servers = map[string]map[string]any{}
	}
	b, err := json.Marshal(struct {
		MCPServers map[string]map[string]any `json:"mcpServers"`
	}{servers})
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
