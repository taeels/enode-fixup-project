package enode

import (
	"strings"
	"testing"
)

// writeMCPConfig 는 mcp: 절 하나를 붙인 최소 설정을 쓴다.
//
// mediator 와 token 은 LoadLocal 이 안 보지만, 예시가 최소 형태로 보이는
// 것이 값이라 그대로 적는다. 파일 쓰기는 identity_nogit_test.go 의
// writeConfig 가 이미 든다 — 도구를 두 벌로 두지 않는다.
func writeMCPConfig(t *testing.T, body string) string {
	t.Helper()
	return writeConfig(t, "mediator: http://m.invalid:8080\ntoken: t\n"+body)
}

// 모양이 틀린 mcp: 는 노드를 안 띄운다 (US-2 · decisions.md 2절).
//
// 왜 설정 시점인가 — yaml 이 잡는 것은 종류뿐이다. 여섯이 하는 일은 하나다:
// 어느 종류인가를 한 값으로 만든다. 모호한 선언이 남으면 mcpUp 이 어느
// 갈래로 갈지를 스스로 정하게 되고, 그 정함은 설정 파일에 안 보인다.
//
// 문구도 함께 잰다 — 사람이 그 줄을 읽고 고친다.
func TestConfig_AMalformedMCPServerStopsTheNode(t *testing.T) {
	for _, c := range []struct {
		name    string
		body    string
		mustSay string
	}{
		{
			name:    "이름이 비었다",
			body:    "mcp:\n  \"\": { command: /bin/true }\n",
			mustSay: "a server name must not be empty",
		},
		{
			name:    "stdio 와 remote 가 둘 다",
			body:    "mcp:\n  probe: { command: /bin/true, url: https://x.invalid }\n",
			mustSay: "command and url are both set",
		},
		{
			name:    "args 만 있고 command 가 없다",
			body:    "mcp:\n  probe: { args: [--port, /dev/ttyUSB0] }\n",
			mustSay: "args without command",
		},
		{
			name:    "credential 만 있고 url 이 없다",
			body:    "mcp:\n  probe: { credential: GERRIT_TOKEN }\n",
			mustSay: "credential without url",
		},
		{
			name:    "둘 다 없다",
			body:    "mcp:\n  probe: {}\n",
			mustSay: "needs command (stdio) or url (remote)",
		},
		{
			name:    "값이 null 이다",
			body:    "mcp:\n  probe:\n",
			mustSay: "needs command (stdio) or url (remote)",
		},
		{
			name:    "env 에 이름이 아니라 값을 적었다",
			body:    "mcp:\n  probe: { command: /bin/true, env: { TOKEN: \"sk-ant-api03-abc\" } }\n",
			mustSay: "must name an environment variable, not a value",
		},
		{
			name:    "env 의 값에 공백이 있다",
			body:    "mcp:\n  probe: { command: /bin/true, env: { TOKEN: \"two words\" } }\n",
			mustSay: "must name an environment variable, not a value",
		},
		{
			name:    "env 의 값이 비었다",
			body:    "mcp:\n  probe: { command: /bin/true, env: { TOKEN: \"\" } }\n",
			mustSay: "must name an environment variable, not a value",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := LoadLocal(writeMCPConfig(t, c.body))
			if err == nil {
				t.Fatal("a malformed mcp server was accepted; the node would come up with it")
			}
			if !strings.Contains(err.Error(), c.mustSay) {
				t.Fatalf("the reason does not say what is wrong:\n  got  %v\n  want %q", err, c.mustSay)
			}
			// 어느 줄인지 말한다 — 서버가 여럿일 수 있다.
			if !strings.Contains(err.Error(), "mcp") {
				t.Fatalf("the reason does not name the mcp block: %v", err)
			}
		})
	}
}

// 성한 선언은 그대로 선다. 검증이 지나치면 여기가 빨개진다.
func TestConfig_AWellFormedMCPBlockLoads(t *testing.T) {
	path := writeMCPConfig(t, `mcp:
  gerrit: { url: https://gerrit.invalid/mcp, credential: GERRIT_TOKEN, env: { GERRIT_TOKEN: GERRIT_TOKEN } }
  serial: { command: /opt/tools/serial-mcp, args: [--port, /dev/ttyUSB0] }
  public: { url: https://open.invalid/mcp }
`)
	l, err := LoadLocal(path)
	if err != nil {
		t.Fatalf("a well-formed mcp block was rejected: %v", err)
	}
	if len(l.MCP) != 3 {
		t.Fatalf("the servers did not survive: %+v", l.MCP)
	}
	if l.MCP["serial"].Command != "/opt/tools/serial-mcp" || len(l.MCP["serial"].Args) != 2 {
		t.Fatalf("the stdio server is wrong: %+v", l.MCP["serial"])
	}
	if l.MCP["gerrit"].Credential != "GERRIT_TOKEN" {
		t.Fatalf("the remote server is wrong: %+v", l.MCP["gerrit"])
	}
}

// mcp: 가 아예 없는 설정이 그대로 선다 — 오늘 도는 노드 전부가 그렇다.
func TestConfig_NoMCPBlockIsNormal(t *testing.T) {
	for _, body := range []string{"", "mcp: {}\n"} {
		l, err := LoadLocal(writeMCPConfig(t, body))
		if err != nil {
			t.Fatalf("a config without mcp servers was rejected: %v", err)
		}
		if len(l.MCP) != 0 {
			t.Fatalf("servers appeared from nowhere: %+v", l.MCP)
		}
	}
}

// 모르는 키는 설정 시점에 안 걸린다 — 그것이 지나가는 값이다.
func TestConfig_UnknownKeysInAServerAreNotAnError(t *testing.T) {
	l, err := LoadLocal(writeMCPConfig(t,
		"mcp:\n  probe: { url: https://x.invalid/mcp, transport: sse }\n"))
	if err != nil {
		t.Fatalf("an unknown key was rejected: %v", err)
	}
	if l.MCP["probe"].Extra["transport"] != "sse" {
		t.Fatalf("the unknown key was dropped: %+v", l.MCP["probe"])
	}
}

// 설정 안내가 mcp: 를 말한다.
//
// 「없다」만 말하고 무엇을 적어야 하는지 안 말하면 사람이 또 찾는다.
func TestConfig_TheSampleMentionsMCP(t *testing.T) {
	if !strings.Contains(SampleLocal(), "mcp:") {
		t.Fatalf("the sample config does not mention the mcp block:\n%s", SampleLocal())
	}
}

// 꼴 검사는 「값을 적지 말라」를 다 못 잰다 — 그 한계를 이름으로 적는다.
//
// 토큰이 우연히 환경변수 이름의 꼴이면 통과한다. 그럼에도 검사를 두는 이유는
// 흔한 실수(하이픈이 든 키 · URL · 공백이 든 문장)가 전부 걸리고, 걸린 사람이
// 문구에서 규칙을 배우기 때문이다. 이 시험이 빨개지면 검사가 넓어진 것이고,
// 그때는 features.md 3.2 와 함께 다시 본다.
func TestConfig_TheEnvNameCheckDoesNotCatchEveryValue(t *testing.T) {
	_, err := LoadLocal(writeMCPConfig(t,
		"mcp:\n  probe: { command: /bin/true, env: { TOKEN: ghp_looks_like_a_name } }\n"))
	if err != nil {
		t.Fatalf("the check got wider than the design says: %v", err)
	}
}
