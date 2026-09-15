package enode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// $OUT 을 문자 그대로 주면 모델이 확장하지 않는다
//
// 실물 claude 에서 밟았다 — 6턴을 쓰고도 아무 파일도 안 만들었다.
// 어댑터는 경로를 아는데 모델은 모른다. 아는 쪽이 적어준다.
func TestPromptCarriesLiteralPath(t *testing.T) {
	out := "/tmp/enode-out-123"
	p := buildPrompt("point at the regression", out, []string{"hypothesis"}, nil, nil, 0, false, nil, nil, nil, nil, nil, nil, "", "")
	if !strings.Contains(p, out+"/hypothesis") {
		t.Fatalf("the real path did not go in:\n%s", p)
	}
	if strings.Contains(p, "$OUT/") {
		t.Fatalf("an unexpanded $OUT survived:\n%s", p)
	}
}

// 스키마가 있으면 못 하겠다를 값으로 말할 수 있어야 한다 (ADR-020).
func TestPromptCarriesSchemaAndHonestNone(t *testing.T) {
	sch := map[string]json.RawMessage{
		"hypothesis": json.RawMessage(`{"type":"object","required":["status"]}`),
	}
	p := buildPrompt("the request", "/o", []string{"hypothesis"}, sch, nil, 0, false, nil, nil, nil, nil, nil, nil, "", "")
	if !strings.Contains(p, `"required":["status"]`) {
		t.Fatal("the schema did not ride the prompt")
	}
	if !strings.Contains(p, "Absence cannot be told apart from a crash") {
		t.Fatal("nothing tells it to say so as a value when it has no conclusion")
	}
}

// 되먹임은 요청 바로 앞에 온다 — 무엇을 고쳐야 하는지가 가장 가깝게 놓인다.
func TestPromptPutsFeedbackJustBeforeRequest(t *testing.T) {
	p := buildPrompt("REQUEST_MARKER", "/o", []string{"x"}, nil,
		map[string]string{"build_log": "error: undefined reference"}, 1, false, nil, nil, nil, nil, nil, nil, "", "")
	fb := strings.Index(p, "error: undefined reference")
	req := strings.Index(p, "REQUEST_MARKER")
	if fb < 0 || req < 0 || fb > req {
		t.Fatalf("the feedback does not sit before the request: fb=%d req=%d", fb, req)
	}
	if !strings.Contains(p, "the previous attempt failed") {
		t.Fatal("nothing says this is a retry")
	}
	// 1회차에는 되먹임이 없다
	if strings.Contains(buildPrompt("R", "/o", []string{"x"}, nil, nil, 0, false, nil, nil, nil, nil, nil, nil, "", ""), "the previous attempt failed") {
		t.Fatal("a first attempt carries retry wording")
	}
}

// 실패 차선은 스키마가 있든 없든 항상 붙는다 (ADR-038)
func TestBuildPrompt_TheFailureLane(t *testing.T) {
	// 스키마 없는 단계
	p := buildPrompt("build it", "/o", []string{"log"}, nil, nil, 0, false, nil, nil, nil, nil, nil, nil, "", "")
	if !strings.Contains(p, "/o/_cannot") {
		t.Fatalf("a step without a schema has no lane:\n%s", p)
	}
	if !strings.Contains(p, "do not claim success") {
		t.Fatalf("nothing wards off a false success")
	}
	// 대체물이 아니라는 것도 말해야 한다 — 안 그러면 _cannot 만 내고 끝낸다
	if !strings.Contains(p, "does not stand in for the required artifact") {
		t.Fatalf("\"the step still fails\" is missing")
	}

	// 스키마 있는 단계에도 붙는다 (ADR-020 문구와 함께)
	sch := map[string]json.RawMessage{"r": json.RawMessage(`{"type":"object"}`)}
	p = buildPrompt("review it", "/o", []string{"r"}, sch, nil, 0, false, nil, nil, nil, nil, nil, nil, "", "")
	if !strings.Contains(p, "/o/_cannot") || !strings.Contains(p, "Absence cannot be told apart from a crash") {
		t.Fatalf("both must be present:\n%s", p)
	}
}

// 자백을 읽으면 ok 가 cannot 이 된다 (ADR-038)
func TestReadCannot(t *testing.T) {
	out := t.TempDir()
	if _, ok := readCannot(out); ok {
		t.Fatal("said it exists when it does not")
	}
	if err := os.WriteFile(filepath.Join(out, cannotName),
		[]byte("  no toolchain  \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	why, ok := readCannot(out)
	if !ok || why != "no toolchain" {
		t.Fatalf("could not read the reason: %q %v", why, ok)
	}
	// 빈 파일도 자백이다
	if err := os.WriteFile(filepath.Join(out, cannotName), []byte("\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if why, ok := readCannot(out); !ok || why == "" {
		t.Fatalf("an empty confession was ignored: %q %v", why, ok)
	}
}

// cannot 은 완주다 — 크래시가 아니라 정직한 보고이므로 산출물을 믿을 수 있다.
func TestReasonCannot_CountsAsCompleted(t *testing.T) {
	if !ReasonCannot.Completed() {
		t.Fatal("if cannot is not completion the artifacts are thrown away — and it is an honest report")
	}
	// 크래시 계열은 그대로 완주가 아니다.
	for _, r := range []Reason{ReasonError, ReasonTimeout} {
		if r.Completed() {
			t.Fatalf("%q became completion", r)
		}
	}
}

// 되먹임은 회차와 무관하게 실린다 (ADR-048)
//
// 예전에는 attempt > 0 일 때만 실었다. 그래서 계획이 지은 재계획 단계가
// 앞 단계 로그를 하나도 못 봤다 — expands 로 붙은 단계는 attempt 0 이다.
// 실측에서 밟았다: 재계획 에이전트가 "요청 섹션이 비어 있고 입력 디렉터리도
// 비어 있어 무엇을 고칠지 모르겠다" 며 _cannot 을 남겼다.
func TestFeedbackRidesEvenOnTheFirstAttempt(t *testing.T) {
	fb := map[string]string{"build_log": "error: something blew up"}

	// attempt 0 — 계획이 지은 재계획 단계의 자리
	got := buildPrompt("build it again", "/o", []string{"plan2"}, nil, fb, 0, true, []string{"a"}, nil, nil, nil, nil, nil, "", "")
	if !strings.Contains(got, "error: something blew up") {
		t.Fatal("no feedback on the first attempt — a replan cannot see the logs")
	}
	// 실패했다고 단정하지 않는다 — 성공한 로그를 보고 판단하는 자리이기도 하다.
	if strings.Contains(got, "the previous attempt failed") {
		t.Fatal("attempt 0, yet it says the previous attempt failed")
	}
	if !strings.Contains(got, "what the earlier steps left behind") {
		t.Fatalf("the heading is missing: %s", got)
	}

	// attempt > 0 — 재시도. 앞 시도의 나가 남긴 것이다
	got = buildPrompt("fix it", "/o", []string{"x"}, nil, fb, 2, false, nil, nil, nil, nil, nil, nil, "", "")
	if !strings.Contains(got, "the previous attempt failed (attempt 2)") {
		t.Fatalf("the retry heading is missing: %s", got)
	}
}

// 거절 이유가 프롬프트에 실린다 (ADR-062)
//
// 거절은 되돌림이므로 다시 도는 것은 계획을 지은 단계 자신이고, 그 in 은
// 처음 그대로다. 이유를 안 실으면 같은 계획을 다시 짓는다 (실측 rewind-1).
func TestTheRejectionReasonRidesThePrompt(t *testing.T) {
	rej := []Rejection{{Answer: json.RawMessage(
		`{"verdict":"again","note":"it writes straight into the workspace root"}`)}}
	p := buildPrompt("the goal", "/out", []string{"plan"}, nil, nil, 0, true,
		nil, nil, nil, nil, rej, nil, "", "")
	for _, want := range []string{"was rejected", "it writes straight into the workspace root", "Do not submit the same plan again"} {
		if !strings.Contains(p, want) {
			t.Fatalf("%q is missing from the prompt", want)
		}
	}
	if q := buildPrompt("the goal", "/out", []string{"plan"}, nil, nil, 0, true,
		nil, nil, nil, nil, nil, nil, "", ""); strings.Contains(q, "was rejected") {
		t.Fatal("the section appeared although there was no rejection")
	}
}

// agent.mcp · agent.pack 이 AgentParams 까지 오고, 타입이 틀리면
// 제출에서 받는 것과 같은 문장이 나오는지 (features.md 3.5 · US-7)
//
// 이 문구는 로그가 아니라 기록이다 — 오류가 Result.Error 로
// steps/NN-*.json 에 봉인된다. 그래서 Go 의 기본 문구면 봉인을 여는 사람이
// 계약 어휘가 아니라 Go 의 구조체 이름을 읽는다.
func TestParseAgentParams_CarriesComponentsAndNamesBadTypes(t *testing.T) {
	p, err := parseAgentParams([]byte(`{"ask":"never","mcp":["probe","serial"],"pack":"kernel-review"}`))
	if err != nil {
		t.Fatalf("a valid agent map was rejected: %v", err)
	}
	if len(p.MCP) != 2 || p.MCP[0] != "probe" || p.MCP[1] != "serial" {
		t.Fatalf("agent.mcp did not arrive: %#v", p.MCP)
	}
	if p.Pack != "kernel-review" {
		t.Fatalf("agent.pack did not arrive: %q", p.Pack)
	}

	// 안 적으면 비어 있다 — 오늘 그대로다.
	p, err = parseAgentParams([]byte(`{"ask":"never"}`))
	if err != nil {
		t.Fatal(err)
	}
	if p.MCP != nil || p.Pack != "" {
		t.Fatalf("absent keys must stay empty: %#v", p)
	}

	for _, tc := range []struct{ raw, want string }{
		{`{"mcp":"probe"}`, "agent.mcp must be an array of server names"},
		{`{"mcp":["probe",""]}`, "agent.mcp[1] must be a non-empty server name"},
		{`{"pack":3}`, "agent.pack must be a blob name"},
		{`{"pack":""}`, "agent.pack must be a blob name"},
	} {
		_, err := parseAgentParams([]byte(tc.raw))
		if err == nil {
			t.Errorf("parseAgentParams(%s) = nil, want an error", tc.raw)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("parseAgentParams(%s) = %v, want it to say %q", tc.raw, err, tc.want)
		}
	}
}

// 빈 배열과 부재가 같아지는 자리 — omitempty 가 빈 슬라이스를 뺀다.
//
// 그래서 "이름을 0 개 적었다" 와 "안 적었다" 가 왕복 뒤에 구별되지 않고,
// 허용목록이 빈다는 같은 뜻으로 남는다 (U4 의 resolveComponents 가 읽는 값).
func TestAgentParams_EmptyMCPRoundTripsToAbsent(t *testing.T) {
	p, err := parseAgentParams([]byte(`{"mcp":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "mcp") {
		t.Fatalf("an empty agent.mcp must not survive the round trip: %s", b)
	}
}
