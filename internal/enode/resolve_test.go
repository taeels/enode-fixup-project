package enode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 이 파일은 resolveComponents 의 규칙을 잰다 (U4 · business-rules R1 ~ R11).
//
// 파일도 프로세스도 안 쓴다 — 출처를 Job 이 들고 오므로 Job 하나를 넣고
// 결과와 오류를 보면 된다. 디스크가 필요한 것은 readWorkspaceMCP 하나이고
// 그 넷은 이 파일 끝에 있다.

func jobWith(request []string, node map[string]MCPServer, ws map[string]map[string]any) Job {
	return Job{Params: AgentParams{MCP: request}, NodeMCP: node, WorkspaceMCP: ws}
}

func names(servers map[string]map[string]any) []string {
	return entryNames(servers)
}

// 계약이 적은 이름만 열린다 (R1).
//
// 출처가 느는 것은 선택지가 느는 것이지 권한이 느는 것이 아니다 —
// 팩(U5)도 같은 필터를 탄다.
func TestResolve_OnlyRequestedNamesAreOpened(t *testing.T) {
	c, err := resolveComponents(jobWith(
		[]string{"probe"},
		map[string]MCPServer{
			"probe":  {Command: "/usr/bin/true"},
			"probe2": {Command: "/usr/bin/true"},
		},
		map[string]map[string]any{"probe3": {"command": "/usr/bin/true"}},
	))
	if err != nil {
		t.Fatal(err)
	}
	if got := names(c.Servers); len(got) != 1 || got[0] != "probe" {
		t.Fatalf("the allowlist carries something that was not requested: %v", got)
	}
}

// 이름이 겹치면 노드가 이긴다 (R3).
//
// 저장소에 쓰는 사람이 소유자의 선언을 가로채지 못한다 — 매칭은 소유자의
// 값으로 하고 실행은 저장소의 값으로 하는 길이 닫힌다.
func TestResolve_TheNodeWinsOverTheWorkspace(t *testing.T) {
	c, err := resolveComponents(jobWith(
		[]string{"probe"},
		map[string]MCPServer{"probe": {Command: "/usr/bin/node-one"}},
		map[string]map[string]any{"probe": {"command": "/usr/bin/workspace-one"}},
	))
	if err != nil {
		t.Fatal(err)
	}
	if c.Servers["probe"]["command"] != "/usr/bin/node-one" {
		t.Fatalf("the workspace overrode the node declaration: %v", c.Servers["probe"])
	}
	if !noted(c, "declared by both") {
		t.Fatalf("the overlap was silent: %v", c.Notes)
	}
}

// 요청한 이름이 어디에도 없으면 단계가 선다 (R4 · R5).
//
// 조용히 빼고 돌면 하네스가 exit 0 으로 끝나고 성공이 봉인된다 —
// 단계는 초록인데 모델은 그 도구를 못 봤다.
func TestResolve_AMissingNameStopsTheStep(t *testing.T) {
	c, err := resolveComponents(jobWith(
		[]string{"zulu", "alpha"},
		map[string]MCPServer{"probe": {Command: "/usr/bin/true"}},
		nil,
	))
	if err == nil {
		t.Fatal("a contract that asked for servers this node does not have was accepted")
	}
	// 정렬한 첫 이름이 문구로 나간다 — 같은 계약이면 같은 문구다.
	if err.Error() != "mcp server alpha is not available on this node" {
		t.Fatalf("the rejection does not carry the pack's sentence: %q", err)
	}
	// 나머지는 Note 로 간다. 게이트가 찾는 문자열은 그대로 두고 사람은
	// 한 번에 다 고칠 수 있다.
	if !noted(c, "mcp server zulu is not available on this node") {
		t.Fatalf("the other missing name was lost: %v", c.Notes)
	}
}

// 못 읽는 워크스페이스 파일은 요청이 있을 때만 만난다 (R7).
//
// 요청이 0 이면 걸음 1 이 먼저 돌아 파일 오류에 닿지 않는다. 부르는 쪽도
// 그때는 파일을 아예 안 연다 (claim.go) — 같은 값이 두 자리에서 지켜진다.
func TestResolve_AnUnreadableWorkspaceFileStopsTheStepOnlyWhenSomethingWasRequested(t *testing.T) {
	broken := Job{
		Params:          AgentParams{MCP: []string{"probe"}},
		NodeMCP:         map[string]MCPServer{"probe": {Command: "/usr/bin/true"}},
		WorkspaceMCPErr: errBrokenFixture,
	}
	_, err := resolveComponents(broken)
	if err == nil || !strings.HasPrefix(err.Error(), "cannot read workspace .mcp.json: ") {
		t.Fatalf("a broken workspace file did not name itself: %v", err)
	}
	// 문구가 파일 경로를 안 싣는다 — res.Error 로 봉인에 들어간다.
	if strings.Contains(err.Error(), "/") && !strings.Contains(err.Error(), ".mcp.json:") {
		t.Fatalf("the rejection carries a path: %v", err)
	}

	quiet := broken
	quiet.Params = AgentParams{}
	if _, err := resolveComponents(quiet); err != nil {
		t.Fatalf("a step that asked for nothing died on someone else's file: %v", err)
	}
}

// 종류를 못 정하는 워크스페이스 항목은 거절이다 (R8).
//
// 채우지 못한 항목을 하네스가 말없이 버리므로, 계약이 이름으로 요청한 것이
// 조용히 사라진다. command: 5 도 같은 자리다 — 문자열이 아닌 값은 없는 것이다.
func TestResolve_AWorkspaceEntryWithoutCommandOrURLIsRejected(t *testing.T) {
	for name, entry := range map[string]map[string]any{
		"empty":  {},
		"number": {"command": float64(5)},
		"blank":  {"url": ""},
	} {
		_, err := resolveComponents(jobWith([]string{"ws"}, nil,
			map[string]map[string]any{"ws": entry}))
		if err == nil || !strings.Contains(err.Error(),
			"workspace mcp server ws declares neither command nor url") {
			t.Fatalf("%s: a shapeless workspace entry was accepted: %v", name, err)
		}
	}
}

// 종류는 없을 때만 채운다 (R9).
//
// claude 2.1.266 은 type 도 command 도 없는 항목을 mcp_servers 에서 말없이
// 뺀다. 이 한 줄이 그 침묵을 막는다. 이미 있으면 소유자나 저장소가 적은 것이 이긴다.
func TestResolve_TheTypeIsFilledInOnlyWhenItIsAbsent(t *testing.T) {
	c, err := resolveComponents(jobWith(
		[]string{"stdio", "remote", "declared", "ws"},
		map[string]MCPServer{
			"stdio":  {Command: "/usr/bin/true"},
			"remote": {URL: "https://gerrit.invalid/mcp", Credential: "GERRIT_TOKEN"},
			// Extra 로 들어온 type 은 우리가 안 덮는다 (U3 의 모르는 키 통과).
			"declared": {URL: "https://gerrit.invalid/mcp", Extra: map[string]any{"type": "sse"}},
		},
		map[string]map[string]any{"ws": {
			"type":    "http",
			"url":     "https://ws.invalid/mcp",
			"headers": map[string]any{"Authorization": "Bearer ${WS_TOKEN}"},
		}},
	))
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{
		"stdio": "stdio", "remote": "http", "declared": "sse", "ws": "http",
	} {
		if got := c.Servers[name]["type"]; got != want {
			t.Fatalf("%s: the kind is %v, not %q", name, got, want)
		}
	}
	// 워크스페이스 항목은 우리 어휘로 안 번역된다 — headers 가 그대로 산다.
	h, ok := c.Servers["ws"]["headers"].(map[string]any)
	if !ok || h["Authorization"] != "Bearer ${WS_TOKEN}" {
		t.Fatalf("a key we do not know was dropped: %v", c.Servers["ws"])
	}
}

// 저장소가 적었으나 이 단계가 요청 안 한 이름은 Note 로 남는다 (R10).
//
// 계약 작성자가 「적어 뒀는데 왜 없나」를 여기서 푼다.
func TestResolve_UnrequestedWorkspaceNamesBecomeNotes(t *testing.T) {
	c, err := resolveComponents(jobWith(
		[]string{"probe"},
		map[string]MCPServer{"probe": {Command: "/usr/bin/true"}},
		map[string]map[string]any{"probe3": {"command": "/usr/bin/true"}},
	))
	if err != nil {
		t.Fatal(err)
	}
	if !noted(c, "workspace .mcp.json declares probe3, which this step did not request") {
		t.Fatalf("the workspace declaration was dropped without a word: %v", c.Notes)
	}
}

// 거절로 끝나도 Notes 는 살아 나온다 (R11).
//
// 왜 실패했는지를 아는 데 필요한 사실이 실패와 함께 사라지면 안 된다.
func TestResolve_NotesSurviveARejection(t *testing.T) {
	c, err := resolveComponents(jobWith(
		[]string{"probe", "nope"},
		map[string]MCPServer{"probe": {Command: "/usr/bin/true"}},
		map[string]map[string]any{"probe": {"command": "/usr/bin/other"}},
	))
	if err == nil {
		t.Fatal("a missing name was accepted")
	}
	if !noted(c, "declared by both") {
		t.Fatalf("the notes were thrown away with the failure: %v", c.Notes)
	}
}

// Job 이 들고 온 맵을 안 고친다.
//
// 종류를 채우는 것이 맵을 고치는 일이고 그 맵은 워크스페이스 파일에서 읽은
// 그대로다. 원본을 고치면 같은 Job 을 두 번 쓰는 호출자와 시험이 조용히 갈린다.
func TestResolve_TheJobsWorkspaceEntriesAreNotModified(t *testing.T) {
	ws := map[string]map[string]any{"ws": {"command": "/usr/bin/true"}}
	if _, err := resolveComponents(jobWith([]string{"ws"}, nil, ws)); err != nil {
		t.Fatal(err)
	}
	if _, ok := ws["ws"]["type"]; ok {
		t.Fatalf("resolve wrote into the job's own map: %v", ws["ws"])
	}
}

// readWorkspaceMCP 의 넷 — 이 유닛에서 디스크가 필요한 유일한 자리다.
func TestReadWorkspaceMCP_TheFourCases(t *testing.T) {
	dir := t.TempDir()

	// ① 없는 파일은 빈 것이다. 저장소 대부분에 그 파일이 없다.
	got, err := readWorkspaceMCP(dir)
	if err != nil || got != nil {
		t.Fatalf("a missing file was not empty: %v %v", got, err)
	}

	// ② mcpServers 키가 없는 파일도 빈 것이다. 다른 목적의 파일일 수 있다.
	writeWorkspaceFile(t, dir, `{"other":{}}`)
	if got, err := readWorkspaceMCP(dir); err != nil || got != nil {
		t.Fatalf("a file without mcpServers was not empty: %v %v", got, err)
	}

	// ③ 깨진 파일은 오류다. 등급은 resolveComponents 가 정한다.
	for _, body := range []string{`{`, `{"mcpServers":[]}`, `{"mcpServers":{"ws":"nope"}}`} {
		writeWorkspaceFile(t, dir, body)
		if _, err := readWorkspaceMCP(dir); err == nil {
			t.Fatalf("a file we cannot parse was read as empty: %s", body)
		}
	}

	// ③-1 값의 종류가 틀린 것은 파일을 안 죽인다 — 원문으로 받기 때문이다.
	// MCPServer 로 받던 판에서는 command: 5 하나가 파일 전체를 죽였다.
	// 그 항목은 요청될 때 R8 이 잡는다.
	writeWorkspaceFile(t, dir, `{"mcpServers":{"ws":{"command":5}}}`)
	if got, err := readWorkspaceMCP(dir); err != nil || len(got) != 1 {
		t.Fatalf("one odd value killed the whole file: %v %v", got, err)
	}

	// ④ 성한 파일은 원문 그대로 온다 — 우리 어휘에 없는 키까지.
	writeWorkspaceFile(t, dir, `{"mcpServers":{"ws":{"type":"http","url":"https://ws.invalid/mcp",`+
		`"headers":{"Authorization":"Bearer x"},"env":{"A":"literal"}}}}`)
	got, err = readWorkspaceMCP(dir)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(got["ws"])
	for _, want := range []string{`"type":"http"`, `"headers"`, `"env"`, `"literal"`} {
		if !strings.Contains(string(b), want) {
			t.Fatalf("the workspace entry lost %s: %s", want, b)
		}
	}
}

func writeWorkspaceFile(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, workspaceMCPName), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func noted(c Components, want string) bool {
	for _, n := range c.Notes {
		if strings.Contains(n, want) {
			return true
		}
	}
	return false
}

// errBrokenFixture 는 읽기 실패를 흉내 내는 오류다. 실제 파일이 필요 없다 —
// 등급을 정하는 것이 resolveComponents 이고 그 입력은 오류 하나다.
var errBrokenFixture = errorString("unexpected end of JSON input")

type errorString string

func (e errorString) Error() string { return string(e) }
