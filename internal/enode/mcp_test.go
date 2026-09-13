package enode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// 빈 허용목록도 봉투를 쓴다 — {} 가 아니다.
//
// --strict-mcp-config 는 「이 파일에 적힌 것만」이고 그 「적힌 것」의 자리가
// mcpServers 다. 키가 없는 파일을 하네스가 어떻게 읽는지는 실측이 없다.
// 봉투를 언제나 쓰면 그 물음이 사라지고, 0 이 의도임이 파일에 보인다.
func TestMCP_TheEmptyAllowlistKeepsTheEnvelope(t *testing.T) {
	for _, servers := range []map[string]map[string]any{nil, {}} {
		b, err := mcpAllowlistJSON(servers)
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != `{"mcpServers":{}}`+"\n" {
			t.Fatalf("an empty allowlist is not the empty envelope: %q", b)
		}
	}
}

// 노드 선언 하나가 그대로 허용목록의 객체가 된다 (U1 이 직렬화를 전부 진다).
//
// 항목을 짓는 것은 allowlistEntry 이고 직렬화는 mcpAllowlistJSON 이다 — U4 가
// 둘 사이를 갈랐다. 파일에 무엇이 실리는가는 여전히 이 시험이 잰다.
func TestMCP_AServerIsCopiedIntoTheAllowlist(t *testing.T) {
	b, err := mcpAllowlistJSON(map[string]map[string]any{
		"gerrit": MCPServer{URL: "https://gerrit.invalid/mcp", Credential: "GERRIT_TOKEN"}.allowlistEntry(),
		"fs": MCPServer{Command: "/usr/bin/mcp-fs", Args: []string{"--root", "/srv"},
			Env: map[string]string{"MCP_FS_MODE": "FS_MODE"}}.allowlistEntry(),
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
	// env 는 이름에서 이름으로 가고 파일에는 참조로 적힌다 (U3 · features.md 3.2).
	// 값을 적으면 그 값이 계장 디렉터리의 파일에 남는다.
	if env, ok := fs["env"].(map[string]any); !ok || env["MCP_FS_MODE"] != "${FS_MODE}" {
		t.Fatalf("env is not a reference: %v", fs["env"])
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

// 요청이 없으면 아무것도 안 연다 (R2).
//
// U1 이 세운 줄이고 U4 뒤에도 참이다 — 출처가 무엇을 선언했든 계약이 이름을
// 안 적으면 0 이다. 빈 맵을 세워 돌려주는 것이 nil 과 빈 것을 쓰는 쪽이
// 안 가르게 한다.
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

// 모르는 키가 허용목록으로 그대로 옮겨진다 (decisions.md 2절 · U3).
//
// 왜 필요한가 — 하네스가 키를 늘릴 때 우리 판을 갈아끼우지 않고 지나가는
// 길이 이것이다. yaml 이 모르는 키를 말없이 버리므로 담을 자리가 없으면
// 그 결정이 코드로 안 선다.
func TestMCP_UnknownKeysRideAlongIntoTheAllowlist(t *testing.T) {
	var s MCPServer
	if err := yaml.Unmarshal([]byte("url: https://x.invalid/mcp\ntransport: sse\nheaders: {A: b}\n"), &s); err != nil {
		t.Fatal(err)
	}
	if s.URL != "https://x.invalid/mcp" {
		t.Fatalf("a known key was lost: %+v", s)
	}
	if s.Extra["transport"] != "sse" {
		t.Fatalf("an unknown key was dropped: %v", s.Extra)
	}
	if _, ok := s.Extra["url"]; ok {
		t.Fatalf("a known key was also kept as an unknown one: %v", s.Extra)
	}
	e := s.allowlistEntry()
	if e["transport"] != "sse" {
		t.Fatalf("the unknown key did not reach the allowlist: %v", e)
	}
	if e["url"] != "https://x.invalid/mcp" {
		t.Fatalf("the known key did not reach the allowlist: %v", e)
	}
}

// 아는 키가 모르는 키를 덮는다. 순서가 뒤집히면 소유자가 적은 선언과
// 우리가 읽은 선언이 갈린다.
//
// 이 줄을 안 재면 얹는 순서가 뒤집혀도 초록이다 — 덮는 방향 자체를 잰다.
func TestMCP_AKnownKeyBeatsAnUnknownOneOfTheSameName(t *testing.T) {
	s := MCPServer{
		Command: "/usr/bin/real",
		Extra:   map[string]any{"command": "/usr/bin/impostor"},
	}
	if got := s.allowlistEntry()["command"]; got != "/usr/bin/real" {
		t.Fatalf("the unknown key won: %v", got)
	}
}

// 아는 키의 목록과 구조체의 태그가 안 갈린다.
//
// 갈리면 아는 키가 Extra 로도 들어가 허용목록에 두 번 적힌다.
func TestMCP_TheKnownKeyListMatchesTheStructTags(t *testing.T) {
	tags := map[string]bool{}
	rt := reflect.TypeOf(MCPServer{})
	for i := 0; i < rt.NumField(); i++ {
		name, _, _ := strings.Cut(rt.Field(i).Tag.Get("yaml"), ",")
		if name != "" && name != "-" {
			tags[name] = true
		}
	}
	for _, k := range knownMCPKeys {
		if !tags[k] {
			t.Errorf("knownMCPKeys has %q but no field carries that yaml tag", k)
		}
		delete(tags, k)
	}
	for k := range tags {
		t.Errorf("the struct has a yaml tag %q that knownMCPKeys does not list", k)
	}
}

// 같은 노드를 두 번 푸는 것이 yaml 의 종류 검사를 안 잃는다.
//
// map 으로만 풀면 args: "--x" 가 오류 대신 문자열로 들어온다.
func TestMCP_TheKindCheckSurvivesTheSecondPass(t *testing.T) {
	var s MCPServer
	if err := yaml.Unmarshal([]byte(`{command: /bin/true, args: "--x"}`), &s); err == nil {
		t.Fatalf("a string where a list belongs was accepted: %+v", s)
	}
}
