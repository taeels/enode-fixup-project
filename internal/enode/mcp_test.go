package enode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 빈 허용목록도 봉투를 쓴다 — {} 가 아니다.
//
// --strict-mcp-config 는 「이 파일에 적힌 것만」이고 그 「적힌 것」의 자리가
// mcpServers 다. 키가 없는 파일을 하네스가 어떻게 읽는지는 실측이 없다.
// 봉투를 언제나 쓰면 그 물음이 사라지고, 0 이 의도임이 파일에 보인다.
func TestMCP_TheEmptyAllowlistKeepsTheEnvelope(t *testing.T) {
	for _, servers := range []map[string]MCPServer{nil, {}} {
		b, err := mcpAllowlistJSON(servers)
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != `{"mcpServers":{}}`+"\n" {
			t.Fatalf("an empty allowlist is not the empty envelope: %q", b)
		}
	}
}

// 서버 하나가 그대로 허용목록의 객체가 된다 (U1 이 직렬화를 전부 진다).
//
// U1 의 제품 경로는 언제나 빈 목록이므로 여기서 직접 넣어 잰다 — 서버가
// 실린 파일은 U4 가 처음 만들지만 쓰는 코드는 이 유닛의 것이다.
func TestMCP_AServerIsCopiedIntoTheAllowlist(t *testing.T) {
	b, err := mcpAllowlistJSON(map[string]MCPServer{
		"gerrit": {URL: "https://gerrit.invalid/mcp", Credential: "GERRIT_TOKEN"},
		"fs":     {Command: "/usr/bin/mcp-fs", Args: []string{"--root", "/srv"}, Env: map[string]string{"MCP_FS_MODE": "ro"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		MCPServers map[string]map[string]any `json:"mcpServers"`
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("the allowlist is not valid json: %v\n%s", err, b)
	}
	fs, ok := got.MCPServers["fs"]
	if !ok || fs["command"] != "/usr/bin/mcp-fs" {
		t.Fatalf("the stdio server did not survive: %v", got.MCPServers)
	}
	if args, ok := fs["args"].([]any); !ok || len(args) != 2 || args[0] != "--root" {
		t.Fatalf("args did not survive: %v", fs["args"])
	}
	if env, ok := fs["env"].(map[string]any); !ok || env["MCP_FS_MODE"] != "ro" {
		t.Fatalf("env did not survive: %v", fs["env"])
	}
	remote, ok := got.MCPServers["gerrit"]
	if !ok || remote["url"] != "https://gerrit.invalid/mcp" {
		t.Fatalf("the remote server did not survive: %v", got.MCPServers)
	}
	// credential 은 안 나간다 — 하네스가 모르는 키이고, 그 이름으로 하네스가
	// 하는 일이 없다. remote 인증은 노드 환경변수로 간다.
	if _, ok := remote["credential"]; ok {
		t.Fatalf("the credential name was written into the allowlist: %v", remote)
	}
	if strings.Contains(string(b), "GERRIT_TOKEN") {
		t.Fatalf("the credential name reached the file:\n%s", b)
	}
	// 빈 값은 「안 적었다」이지 「빈 것을 적었다」가 아니다 — command: "" 를
	// 보내면 하네스가 그것을 실행 경로로 읽을 수 있다.
	if _, ok := fs["url"]; ok {
		t.Fatalf("an empty url was written for the stdio server: %v", fs)
	}
	for _, key := range []string{"command", "args", "env"} {
		if _, ok := remote[key]; ok {
			t.Fatalf("an empty %s was written for the remote server: %v", key, remote)
		}
	}
}

// 허용목록 파일은 0600 이고 한 줄이다.
func TestMCP_TheAllowlistFileIsLockedDown(t *testing.T) {
	path := filepath.Join(t.TempDir(), mcpAllowlistName)
	if err := writeMCPAllowlist(path, nil); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("the allowlist is readable by others: %v", st.Mode().Perm())
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(b), "\n") != 1 {
		t.Fatalf("the allowlist is not one line: %q", b)
	}
}

// U1 은 아무것도 안 연다 — 요청도 팩도 없는 경우만 다룬다.
//
// 빈 맵을 세워 돌려주는 것이 nil 과 빈 것을 쓰는 쪽이 안 가르게 한다.
func TestMCP_ResolveComponentsOpensNothing(t *testing.T) {
	c, err := resolveComponents(Job{})
	if err != nil {
		t.Fatal(err)
	}
	if c.Servers == nil || len(c.Servers) != 0 {
		t.Fatalf("this unit opened something: %v", c.Servers)
	}
	if c.Pack != nil || len(c.Notes) != 0 {
		t.Fatalf("a pack or a note appeared before the unit that makes them: %+v", c)
	}
}
