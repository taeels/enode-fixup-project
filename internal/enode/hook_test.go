package enode

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func hookRun(t *testing.T, a HookArgs, in StopInput) StopOutput {
	t.Helper()
	b, _ := json.Marshal(in)
	var out bytes.Buffer
	if err := RunStopHook(a, bytes.NewReader(b), &out); err != nil {
		t.Fatal(err)
	}
	var o StopOutput
	if out.Len() > 0 {
		if err := json.Unmarshal(out.Bytes(), &o); err != nil {
			t.Fatalf("the hook output is not JSON: %q", out.String())
		}
	}
	return o
}

// 같은 이유로는 두 번 안 막는다 — 안 그러면 영원히 돈다 (ADR-051).
func TestHook_BlocksOnceForTheSameReason(t *testing.T) {
	out := t.TempDir()
	a := HookArgs{Out: out, Expect: []string{"nosuch"}}
	if o := hookRun(t, a, StopInput{}); o.Decision != "block" {
		t.Fatalf("did not block the first time: %+v", o)
	}
	if o := hookRun(t, a, StopInput{StopHookActive: true}); o.Decision != "" {
		t.Fatalf("blocked again for the same reason — that is an infinite loop: %+v", o)
	}
}

// 되묻기는 이유마다 한 번씩이다 (ADR-051)
//
// 예전에는 stop_hook_active 하나로 「한 번 물었으면 끝」이었다. 그래서
// 첫 되묻기가 산출물 누락이면 문법은 영영 안 봤다.
//
//	실측        vm-scratch-3 의 replan_1 이 11턴을 돌고 문법이 틀린 계획을
//	          냈는데 훅이 한 번도 안 짚었다. 서버가 그것을 거절해 단계가 죽었다.
func TestHook_StillChecksGrammarAfterPointingAtArtifacts(t *testing.T) {
	out := t.TempDir()
	a := HookArgs{Out: out, Expect: []string{"plan2"}, Plan: "plan2", Roles: []string{"n"}}

	// ① 아직 아무것도 안 냈다 — 산출물 누락으로 막는다.
	o := hookRun(t, a, StopInput{})
	if o.Decision != "block" || !strings.Contains(o.Reason, "plan2") {
		t.Fatalf("the missing artifact was not pointed out: %+v", o)
	}

	// ② 모델이 파일을 냈다. 그런데 문법이 틀렸다 —
	//    success_when[].produced 는 배열이어야 하는데 문자열이다.
	bad := `{"steps":[{"id":"a","uses":"n","run":["true"],"out":["x"]}],
	         "success_when":[{"step":"a","produced":"x"}]}`
	if err := os.WriteFile(filepath.Join(out, "plan2"), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	o = hookRun(t, a, StopInput{StopHookActive: true})
	if o.Decision != "block" {
		t.Fatalf("skipped the grammar because it had pointed at artifacts: %+v", o)
	}
	if !strings.Contains(o.Reason, "contract grammar") {
		t.Fatalf("it did not say this is a grammar problem: %+v", o)
	}

	// ③ 같은 문법 문제로는 한 번만 짚는다.
	if o = hookRun(t, a, StopInput{StopHookActive: true}); o.Decision != "" {
		t.Fatalf("blocked again for the same grammar problem: %+v", o)
	}

	// ④ 다른 문법 문제면 다시 짚는다 (ADR-051 §4)
	//
	// 실측 (vm-scratch-4) 훅이 문법을 한 번 짚었고 모델이 그것을 고쳤는데
	// 다른 자리에서 또 틀렸다 — produced 는 배열이 됐고 이번엔 in 이
	// 배열이었다. 이유를 종류로 세면 두 번째 위반은 못 짚는다.
	other := `{"steps":[{"id":"a","uses":"n","run":["true"],"out":["x"],"in":["oops"]}],
	           "success_when":[{"step":"a","produced":["x"]}]}`
	if err := os.WriteFile(filepath.Join(out, "plan2"), []byte(other), 0o644); err != nil {
		t.Fatal(err)
	}
	if o = hookRun(t, a, StopInput{StopHookActive: true}); o.Decision != "block" {
		t.Fatalf("a different grammar problem was not pointed out: %+v", o)
	}
}

// 되묻기에는 총 상한이 있다 — 이유를 「무엇이 틀렸나」로 세면
// 종류가 무한히 늘 수 있고, 그러면 영원히 막을 수 있다 (ADR-051).
func TestHook_PassesThroughOnceTheAskBudgetIsSpent(t *testing.T) {
	out := t.TempDir()
	a := HookArgs{Out: out, Expect: []string{"plan2"}, Plan: "plan2", Roles: []string{"n"}}
	blocked := 0
	for i := 0; i < hookBlockBudget+3; i++ {
		// 매번 다른 문법 위반 — 이름이 다르면 오류 문장도 다르다.
		bad := `{"steps":[{"id":"s` + strconv.Itoa(i) + `","uses":"norole",` +
			`"run":["true"],"out":["x"]}],"success_when":[]}`
		if err := os.WriteFile(filepath.Join(out, "plan2"), []byte(bad), 0o644); err != nil {
			t.Fatal(err)
		}
		if hookRun(t, a, StopInput{}).Decision == "block" {
			blocked++
		}
	}
	if blocked > hookBlockBudget {
		t.Fatalf("blocked %d times past the cap — that is where an infinite loop lives", blocked)
	}
	if blocked == 0 {
		t.Fatal("never blocked — the cap killed the asking itself")
	}
}

// 기억 파일은 산출물이 아니다 — .enode- 접두사가 수확에서 걸러진다.
func TestHook_TheMemoryFileDoesNotBecomeAnArtifact(t *testing.T) {
	out := t.TempDir()
	hookRun(t, HookArgs{Out: out, Expect: []string{"nosuch"}}, StopInput{})
	for _, n := range harvest(out) {
		if !strings.HasPrefix(n, ".enode-") {
			t.Fatalf("a file that would be harvested appeared in $OUT: %s", n)
		}
	}
}

// 요구된 것이 다 있으면 통과.
func TestHook_PassesWhenEverythingWasProduced(t *testing.T) {
	out := t.TempDir()
	if err := os.WriteFile(filepath.Join(out, "hypothesis"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if o := hookRun(t, HookArgs{Out: out, Expect: []string{"hypothesis"}}, StopInput{}); o.Decision != "" {
		t.Fatalf("blocked although everything was produced: %+v", o)
	}
}

// 계약이 요구한 이름 중 없는 것을 짚는다 — 막연한 물음이 아니다.
func TestHook_PointsAtTheMissingName(t *testing.T) {
	out := t.TempDir()
	if err := os.WriteFile(filepath.Join(out, "hypothesis"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	o := hookRun(t, HookArgs{Out: out, Expect: []string{"hypothesis", "test_source"}}, StopInput{})
	if o.Decision != "block" {
		t.Fatalf("something is missing yet it did not block: %+v", o)
	}
	if !strings.Contains(o.Reason, "test_source") {
		t.Fatalf("the missing name was not pointed out: %q", o.Reason)
	}
	if strings.Contains(o.Reason, "fail") {
		t.Fatalf("the hook passed judgment — success_when does the judging (ADR-004·I3): %q", o.Reason)
	}
}

// 요구가 없으면 아무 말도 안 한다 — 훅이 참견할 근거가 없다.
func TestHook_StaysQuietWhenNothingIsRequired(t *testing.T) {
	if o := hookRun(t, HookArgs{Out: t.TempDir()}, StopInput{}); o.Decision != "" {
		t.Fatalf("blocked although nothing was required: %+v", o)
	}
}

// 입력이 깨져도 통과시킨다 — 훅이 하네스를 막아 세우면 안 된다.
func TestHook_BrokenInputPassesThrough(t *testing.T) {
	var out bytes.Buffer
	if err := RunStopHook(HookArgs{Expect: []string{"x"}},
		strings.NewReader("this is not JSON"), &out); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Fatalf("blocked on broken input: %s", out.String())
	}
}

// 바뀐 파일을 참고로 보여준다 — 판정이 아니라 알림이다.
//
// git status 로 하지 않는다 — .gitignore 를 지켜서 빌드 산출물을 가린다.
// 기준 시각(stamp) 이 있어야 zImage 도 .ko 도 보인다. changed_test.go 참조.
func TestHook_ReportsWhatChanged(t *testing.T) {
	dir, inst := gitInit(t), t.TempDir()
	time.Sleep(1100 * time.Millisecond)
	s := stampNow(dir)
	time.Sleep(1100 * time.Millisecond)
	write(t, dir, "fixed.c", "int q;\n")

	sp := filepath.Join(inst, "stamp")
	if err := writeStamp(sp, s); err != nil {
		t.Fatal(err)
	}
	o := hookRun(t, HookArgs{Out: t.TempDir(), Workspace: dir, Expect: []string{"result"}, Stamp: sp},
		StopInput{})
	if !strings.Contains(o.Reason, "fixed.c") {
		t.Fatalf("what changed was not reported: %q", o.Reason)
	}
}

// 설정 파일이 $OUT 밖에 놓인다 — 안에 두면 ④수확이 산출물로 걷어 올린다.
func TestHook_TheSettingsFileSitsOutsideOUT(t *testing.T) {
	inst, out := t.TempDir(), t.TempDir()
	flags, err := WriteHookSettings(inst, "/usr/bin/enode", HookArgs{Out: out, Expect: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(harvest(out)) != 0 {
		t.Fatalf("the instrumentation file landed in $OUT: %v", harvest(out))
	}
	// 개인 설정을 차단한다 (R6) — 안 하면 노드마다 결과가 달라진다.
	j := strings.Join(flags, " ")
	if !strings.Contains(j, "--setting-sources") {
		t.Fatalf("personal settings were not blocked: %v", flags)
	}
	b, err := os.ReadFile(filepath.Join(inst, "enode-settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"Stop"`) || !strings.Contains(string(b), "hook stop") {
		t.Fatalf("the Stop hook was not planted:\n%s", b)
	}
}

// 사내 인증 게이트웨이의 인증 필드(apiKeyHelper · env)는 병합된다.
//
// 값은 값이 아니라 통과 여부만 본다 — apiKeyHelper 는 실행되는 스크립트 경로고
// env 는 그 스크립트가 인증 모드를 고르는 재료라, 병합 대상인지가 중요하다.
func TestHook_MergesGatewayAuthFieldsFromPersonalSettings(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	personal := `{
		"apiKeyHelper": "/opt/gateway/api-key-helper",
		"env": {"OIDC_ISSUER_URL": "https://example.invalid", "OIDC_CLIENT_ID": "abc123"},
		"permissions": {"allow": ["Bash"]}
	}`
	if err := os.WriteFile(filepath.Join(home, ".claude", "settings.json"), []byte(personal), 0o600); err != nil {
		t.Fatal(err)
	}

	inst, out := t.TempDir(), t.TempDir()
	if _, err := WriteHookSettings(inst, "/usr/bin/enode", HookArgs{Out: out}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(inst, "enode-settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got["apiKeyHelper"] != "/opt/gateway/api-key-helper" {
		t.Fatalf("apiKeyHelper was not merged: %v", got["apiKeyHelper"])
	}
	env, ok := got["env"].(map[string]any)
	if !ok || env["OIDC_ISSUER_URL"] != "https://example.invalid" || env["OIDC_CLIENT_ID"] != "abc123" {
		t.Fatalf("env was not merged: %v", got["env"])
	}
	// 그 밖의 개인 설정(권한 등)은 여전히 안 옮겨진다 — R6 의 나머지는 그대로다.
	if _, ok := got["permissions"]; ok {
		t.Fatalf("an unrelated personal setting leaked through: %v", got)
	}
}

// 개인 설정 파일이 없으면(개인 구독 · Bedrock · Vertex 는 흔히 없다)
// 조용히 아무것도 병합하지 않는다 — 지금까지의 동작 그대로다.
func TestHook_NoPersonalSettingsIsFine(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // .claude/settings.json 이 존재하지 않는 HOME

	inst, out := t.TempDir(), t.TempDir()
	if _, err := WriteHookSettings(inst, "/usr/bin/enode", HookArgs{Out: out}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(inst, "enode-settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if _, ok := got["apiKeyHelper"]; ok {
		t.Fatalf("apiKeyHelper appeared with no personal settings file: %v", got)
	}
}

// 공백이 든 경로가 훅 명령에서 안 깨진다 — 셸이 한 줄로 받기 때문이다.
func TestHook_APathWithSpacesSurvives(t *testing.T) {
	inst := t.TempDir()
	out := filepath.Join(t.TempDir(), "folder with spaces")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteHookSettings(inst, "/usr/bin/enode", HookArgs{Out: out}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(inst, "enode-settings.json"))
	var cfg struct {
		Hooks map[string][]struct {
			Hooks []struct{ Command string } `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		t.Fatal(err)
	}
	cmd := cfg.Hooks["Stop"][0].Hooks[0].Command
	if !strings.Contains(cmd, "'"+out+"'") {
		t.Fatalf("the path with spaces was not quoted: %s", cmd)
	}
}

// 훅이 어긴 계획을 하네스가 끝나기 전에 짚는다 (ADR-046)
//
// 이것이 없으면 어긴 계획은 계약 적용 시점 에야 거절되고, 그때는
// 하네스가 이미 끝나 고칠 기회가 없다 — 판 하나가 통째로 버려진다.
// 실측에서 두 번 밟았다 (10차 없는 역할 · 11차 schema 키잉).
func TestHook_PointsAtAPlanThatBreaksTheRules(t *testing.T) {
	dir := t.TempDir()
	// 산출물은 냈다 — missingOutputs 는 통과한다. 모양만 틀렸다.
	if err := os.WriteFile(filepath.Join(dir, "plan"),
		[]byte(`{"steps":[{"id":"r","uses":"norole","run":["true"],"out":["l"]}]}`),
		0o644); err != nil {
		t.Fatal(err)
	}
	a := HookArgs{Out: dir, Expect: []string{"plan"}, Plan: "plan",
		Roles: []string{"planner", "mac"}}
	var out bytes.Buffer
	if err := RunStopHook(a, strings.NewReader(`{"stop_hook_active":false}`), &out); err != nil {
		t.Fatal(err)
	}
	var so StopOutput
	if err := json.Unmarshal(out.Bytes(), &so); err != nil {
		t.Fatalf("did not block: %q", out.String())
	}
	if so.Decision != "block" {
		t.Fatalf("did not block: %+v", so)
	}
	// 오류 문장을 그대로 전한다 — 번역하면 ADR-045 의 실수를 되풀이한다.
	if !strings.Contains(so.Reason, "undeclared role") {
		t.Fatalf("what Validate said did not ride along: %s", so.Reason)
	}
	// 어휘도 함께 준다 — 무엇을 써야 하는지 모르면 또 추측한다.
	if !strings.Contains(so.Reason, "planner") {
		t.Fatalf("the role list did not ride along: %s", so.Reason)
	}
}

// 정당한 계획은 안 막는다 — 안전망이 정규 경로를 무너뜨리는 것이 가장 나쁘다.
func TestHook_ALegitimatePlanPasses(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "plan"),
		[]byte(`{"steps":[{"id":"w","uses":"mac","needs":["approve_plan"],`+
			`"run":["true"],"out":["l"]}],`+
			`"success_when":[{"step":"w","exit_code":0,"produced":["l"]}]}`), 0o644)
	a := HookArgs{Out: dir, Expect: []string{"plan"}, Plan: "plan",
		Roles: []string{"planner", "mac"}}
	var out bytes.Buffer
	if err := RunStopHook(a, strings.NewReader(`{"stop_hook_active":false}`), &out); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Fatalf("blocked a legitimate plan: %s", out.String())
	}
}

// 계획 단계가 아니면 안 본다 — Plan 이 비면 그냥 지나간다.
func TestHook_LooksOnlyAtAPlanStep(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "x"), []byte(`this is not a plan`), 0o644)
	a := HookArgs{Out: dir, Expect: []string{"x"}} // Plan 이 비어 있다
	var out bytes.Buffer
	if err := RunStopHook(a, strings.NewReader(`{"stop_hook_active":false}`), &out); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Fatalf("blocked something that is not a plan: %s", out.String())
	}
}
