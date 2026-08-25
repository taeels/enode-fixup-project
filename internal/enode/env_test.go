package enode

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func has(env []string, name string) (string, bool) {
	for _, kv := range env {
		if k, v, ok := strings.Cut(kv, "="); ok && k == name {
			return v, true
		}
	}
	return "", false
}

// R1 의 본체 — 토큰이 하네스로 안 간다.
func TestEnv_TheTokenDoesNotLeak(t *testing.T) {
	t.Setenv("ENODE_TOKEN", "secret")
	t.Setenv("ENODE_MEDIATOR", "http://internal")

	env := harnessEnv(claudeEnv, map[string]string{"OUT": "/o", "IN": "/i"}, nil)

	for _, bad := range []string{"ENODE_TOKEN", "ENODE_MEDIATOR"} {
		if _, ok := has(env, bad); ok {
			t.Fatalf("%s got through", bad)
		}
	}
	if v, ok := has(env, "OUT"); !ok || v != "/o" {
		t.Fatalf("OUT is missing: %q %v", v, ok)
	}
}

// 화이트리스트가 필요한 것은 통과시킨다 — 너무 조이면 하네스가 안 돈다.
func TestEnv_TheRequiredOnesPass(t *testing.T) {
	t.Setenv("PATH", "/usr/bin")
	t.Setenv("HOME", "/home/someone") // transparent auth hangs on this
	t.Setenv("HTTPS_PROXY", "http://proxy:3128")

	env := harnessEnv(nil, nil, nil)
	for _, need := range []string{"PATH", "HOME", "HTTPS_PROXY"} {
		if _, ok := has(env, need); !ok {
			t.Fatalf("%s did not get through — the harness cannot run", need)
		}
	}
}

// 하네스가 선언한 이름은 통과하고, 선언 안 한 것은 막힌다.
func TestEnv_OnlyWhatTheHarnessDeclaresPasses(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "key")
	t.Setenv("ANTHROPIC_SECRET_WHATEVER", "nope")

	env := harnessEnv(claudeEnv, nil, nil)
	if _, ok := has(env, "ANTHROPIC_API_KEY"); !ok {
		t.Fatal("a declared name was blocked")
	}
	if _, ok := has(env, "ANTHROPIC_SECRET_WHATEVER"); ok {
		t.Fatal("an undeclared name passed")
	}
}

// 주입이 마지막에 이긴다 — 요청자 신원으로 갈아끼우는 것이 R2 의 형태다.
func TestEnv_InjectionWins(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "from-machine")
	env := harnessEnv(claudeEnv, nil, map[string]string{"ANTHROPIC_API_KEY": "from-requester"})
	if v, _ := has(env, "ANTHROPIC_API_KEY"); v != "from-requester" {
		t.Fatalf("injection did not win: %q", v)
	}
}

// Transparent 는 아무것도 안 준다 — MVP 의 정직한 형태.
func TestAuth_TheMVPIsTransparent(t *testing.T) {
	m, err := Transparent{}.For(context.Background(), RunIdentity{RunID: "r1"})
	if err != nil || len(m) != 0 {
		t.Fatalf("%v %v", m, err)
	}
}

// 걸러낸 것을 이름만 로그로 낼 수 있어야 한다 — 조용한 실패 금지.
func TestEnv_SaysWhatItFilteredOut(t *testing.T) {
	t.Setenv("ANTHROPIC_UNKNOWN", "x")
	t.Setenv("ENODE_TOKEN", "secret")

	d := droppedNotable()
	if len(d) == 0 || d[0] != "ANTHROPIC_UNKNOWN" {
		t.Fatalf("could not name the auth variable it withheld: %v", d)
	}
	for _, k := range d {
		if strings.HasPrefix(k, "ENODE_") {
			t.Fatal("ENODE_ must not print every time — it buries the real signal")
		}
	}
}

// 진짜 시험 — 실제로 프로세스를 띄워서 확인한다.
//
// harnessEnv 단위 테스트만으로는 누가 cmd.Env 에 os.Environ() 을 다시 얹는
// 회귀 를 못 잡는다. 그건 정확히 우리가 고친 그 한 줄이다.
func TestEnv_DoesNotLeakIntoARealProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh is missing")
	}
	os.Setenv("ENODE_TOKEN", "secret-word-teal") //nolint:errcheck // the child must be able to see it for the test to mean anything
	defer os.Unsetenv("ENODE_TOKEN")             //nolint:errcheck

	env := harnessEnv(nil, map[string]string{"OUT": "/o"}, nil)
	cmd := exec.Command("/bin/sh", "-c", "env")
	cmd.Env = env
	var buf bytes.Buffer
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		t.Fatalf("running env failed: %v", err)
	}
	if strings.Contains(buf.String(), "secret-word-teal") {
		t.Fatalf("the child process saw the token\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "OUT=/o") {
		t.Fatalf("OUT did not reach the child\n%s", buf.String())
	}
}

// 구멍이 있던 바로 그 줄을 막는다
//
// 위 시험은 harnessEnv 가 옳다는 것만 보인다. 정작 고친 곳은 exec 지점의
//
//	cmd.Env = append(os.Environ(), …)   →   cmd.Env = env
//
// 이므로, 누가 여기에 os.Environ() 을 다시 얹는 회귀를 잡으려면
// runner 를 통과시켜야 한다. 가짜 하네스를 세워 그 구간을 덮는다.
//
// R3 로 exec 이 runner.go 한 곳에 모였으므로 이 시험 하나가 모든 어댑터를 덮는다.
func TestEnv_TheRunnerUsesTheWhitelist(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh is missing")
	}
	os.Setenv("ENODE_TOKEN", "secret-word-teal") //nolint:errcheck
	defer os.Unsetenv("ENODE_TOKEN")             //nolint:errcheck

	dir := t.TempDir()
	// 플래그는 전부 무시하고, 자기가 받은 환경을 파일로 뱉은 뒤 봉투를 찍는다.
	fake := dir + "/fake-harness"
	script := "#!/bin/sh\nenv > \"$OUT/env.txt\"\n" +
		`printf '{"type":"result","subtype":"success","num_turns":1,"session_id":"s1"}` + "\n'\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	_, h := runHarness(context.Background(), claudeHarness{}, fake, Job{
		Prompt: "hello", IO: IOPaths{Dir: dir, In: dir, Out: dir}})
	if h.Reason != ReasonOK {
		t.Fatalf("the fake harness did not complete: %+v", h)
	}
	if h.Session != "s1" {
		t.Fatalf("the session was not passed through: %q", h.Session)
	}

	seen, err := os.ReadFile(dir + "/env.txt")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(seen), "secret-word-teal") {
		t.Fatalf("the harness saw the token\n%s", seen)
	}
	if !strings.Contains(string(seen), "OUT="+dir) {
		t.Fatalf("OUT did not go through\n%s", seen)
	}
}

// 자동 메모리를 값으로 끈다
//
// --setting-sources ” 는 설정만 막고 ~/.claude/projects/<cwd>/memory/ 는 못 막는다
// (Agent SDK 호스팅 문서: 자동 메모리는 settingSources 와 무관하게 로드된다).
// R1 이 HOME 을 통과시키므로 노드 주인의 메모리가 시스템 프롬프트에 섞이고,
// 노드마다 결과가 달라지는데 Record 에는 안 남는다.
//
// 통과 이름이 아니라 값이어야 한다 — 이름이면 노드 환경에 없을 때 안 걸린다.
func TestEnv_TurnsOffAutoMemoryByValue(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh is missing")
	}
	// 노드 환경에 그 변수가 아예 없어도 걸려야 한다.
	os.Unsetenv("CLAUDE_CODE_DISABLE_AUTO_MEMORY") //nolint:errcheck

	dir := t.TempDir()
	fake := dir + "/fake-harness"
	script := "#!/bin/sh\nenv > \"$OUT/env.txt\"\n" +
		`printf '{"type":"result","subtype":"success","num_turns":1,"session_id":"s1"}` + "\n'\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, h := runHarness(context.Background(), claudeHarness{}, fake, Job{
		Prompt: "hello", IO: IOPaths{Dir: dir, In: dir, Out: dir}}); h.Reason != ReasonOK {
		t.Fatalf("the fake harness did not complete: %+v", h)
	}
	seen, err := os.ReadFile(dir + "/env.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(seen), "CLAUDE_CODE_DISABLE_AUTO_MEMORY=1") {
		t.Fatalf("auto memory was not turned off\n%s", seen)
	}
}

// 명령 단계도 화이트리스트다
//
// R1 을 처음 고칠 때 agent 쪽만 막았는데, 계약은 노드 주인이 아닌 사람이
// 낼 수 있고 argv 는 무엇이든 된다 — `sh -c 'env > $OUT/leak'` 이면 끝난다.
// 위협이 같으므로 규칙도 같다.
func TestEnv_CommandStepsAreBlockedToo(t *testing.T) {
	t.Setenv("ENODE_TOKEN", "secret")
	t.Setenv("ARCH", "arm")
	t.Setenv("CORP_SECRET_VAR", "must-not-leak")

	env := harnessEnv(commandEnv, map[string]string{"OUT": "/o"}, nil)
	if _, ok := has(env, "ENODE_TOKEN"); ok {
		t.Fatal("the token reached a command step")
	}
	if _, ok := has(env, "CORP_SECRET_VAR"); ok {
		t.Fatal("an undeclared name passed")
	}
	if _, ok := has(env, "ARCH"); !ok {
		t.Fatal("ARCH was blocked — cross builds cannot work")
	}
}

// 계약이 이름을 더할 수 있다 — 값이 아니라 이름이다.
// 값을 계약에 적으면 Run Record 의 manifest 로 봉인되어 영구히 남는다.
func TestEnv_TheContractAddsNames(t *testing.T) {
	t.Setenv("CORP_TOOLCHAIN_PATH", "/opt/tc")

	if _, ok := has(harnessEnv(commandEnv, nil, nil), "CORP_TOOLCHAIN_PATH"); ok {
		t.Fatal("passed without a declaration")
	}
	declared := append(append([]string{}, commandEnv...), "CORP_TOOLCHAIN_PATH")
	if _, ok := has(harnessEnv(declared, nil, nil), "CORP_TOOLCHAIN_PATH"); !ok {
		t.Fatal("the contract declared it yet it did not pass")
	}
}
