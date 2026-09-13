package enode

import (
	"encoding/json"
	"os"
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
	Command    string            // stdio
	Args       []string          // stdio
	URL        string            // remote
	Credential string            // remote. 환경변수 이름이다. 값이 아니다
	Env        map[string]string // 이름만 적는다. 값은 노드 환경이 R1 로 넘긴다
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
// 그 이름이 사는 자리는 mcpUp 의 remote 판정 하나다 (U3).
//
// 이음매 — U3 이 「그 밖의 키」를 그대로 옮긴다 (decisions.md 2절). 그 맵을
// 여기 먼저 얹고 아는 키를 그 위에 덮는다. 아는 키가 이겨야 소유자가 적은
// 선언과 우리가 읽은 선언이 안 갈린다.
func (s MCPServer) allowlistEntry() map[string]any {
	e := map[string]any{}
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
		e["env"] = s.Env
	}
	return e
}
