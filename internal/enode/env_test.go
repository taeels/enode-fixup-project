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

// ★ R1 의 본체 ★ — 토큰이 하네스로 안 간다.
func Test환경_토큰이_안_샌다(t *testing.T) {
	t.Setenv("ENODE_TOKEN", "비밀")
	t.Setenv("ENODE_MEDIATOR", "http://내부")

	env := harnessEnv(claudeEnv, map[string]string{"OUT": "/o", "IN": "/i"}, nil)

	for _, bad := range []string{"ENODE_TOKEN", "ENODE_MEDIATOR"} {
		if _, ok := has(env, bad); ok {
			t.Fatalf("★ %s 가 넘어갔다 ★", bad)
		}
	}
	if v, ok := has(env, "OUT"); !ok || v != "/o" {
		t.Fatalf("OUT 이 없다: %q %v", v, ok)
	}
}

// 화이트리스트가 ★ 필요한 것은 통과시킨다 ★ — 너무 조이면 하네스가 안 돈다.
func Test환경_필수는_통과한다(t *testing.T) {
	t.Setenv("PATH", "/usr/bin")
	t.Setenv("HOME", "/home/누구") // ★ transparent 인증이 여기 걸려 있다 ★
	t.Setenv("HTTPS_PROXY", "http://프록시:3128")

	env := harnessEnv(nil, nil, nil)
	for _, need := range []string{"PATH", "HOME", "HTTPS_PROXY"} {
		if _, ok := has(env, need); !ok {
			t.Fatalf("★ %s 가 안 넘어갔다 ★ — 하네스가 못 돈다", need)
		}
	}
}

// 하네스가 선언한 이름은 통과하고, ★ 선언 안 한 것은 막힌다 ★.
func Test환경_하네스_선언만_통과한다(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "키")
	t.Setenv("ANTHROPIC_SECRET_어쩌구", "안돼")

	env := harnessEnv(claudeEnv, nil, nil)
	if _, ok := has(env, "ANTHROPIC_API_KEY"); !ok {
		t.Fatal("선언한 이름이 막혔다")
	}
	if _, ok := has(env, "ANTHROPIC_SECRET_어쩌구"); ok {
		t.Fatal("★ 선언 안 한 이름이 통과했다 ★")
	}
}

// 주입이 ★ 마지막에 이긴다 ★ — 요청자 신원으로 갈아끼우는 것이 R2 의 형태다.
func Test환경_주입이_덮어쓴다(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "머신것")
	env := harnessEnv(claudeEnv, nil, map[string]string{"ANTHROPIC_API_KEY": "요청자것"})
	if v, _ := has(env, "ANTHROPIC_API_KEY"); v != "요청자것" {
		t.Fatalf("주입이 안 이겼다: %q", v)
	}
}

// Transparent 는 ★ 아무것도 안 준다 ★ — MVP 의 정직한 형태.
func Test인증_MVP는_투명하다(t *testing.T) {
	m, err := Transparent{}.For(context.Background(), RunIdentity{RunID: "r1"})
	if err != nil || len(m) != 0 {
		t.Fatalf("%v %v", m, err)
	}
}

// 걸러낸 것을 ★ 이름만 ★ 로그로 낼 수 있어야 한다 — 조용한 실패 금지.
func Test환경_걸러낸_것을_말한다(t *testing.T) {
	t.Setenv("ANTHROPIC_모르는것", "x")
	t.Setenv("ENODE_TOKEN", "비밀")

	d := droppedNotable()
	if len(d) == 0 || d[0] != "ANTHROPIC_모르는것" {
		t.Fatalf("안 넘긴 인증 이름을 못 짚었다: %v", d)
	}
	for _, k := range d {
		if strings.HasPrefix(k, "ENODE_") {
			t.Fatal("★ ENODE_ 는 매번 찍히면 안 된다 ★ — 진짜 신호가 묻힌다")
		}
	}
}

// ★ 진짜 시험 ★ — 실제로 프로세스를 띄워서 확인한다.
//
// harnessEnv 단위 테스트만으로는 ★ 누가 cmd.Env 에 os.Environ() 을 다시 얹는
// 회귀 ★ 를 못 잡는다. 그건 정확히 우리가 고친 그 한 줄이다.
func Test환경_실제_프로세스에_안_샌다(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh 가 없다")
	}
	os.Setenv("ENODE_TOKEN", "비밀단어청록") //nolint:errcheck // 자식이 볼 수 있어야 시험이 된다
	defer os.Unsetenv("ENODE_TOKEN")   //nolint:errcheck

	env := harnessEnv(nil, map[string]string{"OUT": "/o"}, nil)
	cmd := exec.Command("/bin/sh", "-c", "env")
	cmd.Env = env
	var buf bytes.Buffer
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		t.Fatalf("env 실행 실패: %v", err)
	}
	if strings.Contains(buf.String(), "비밀단어청록") {
		t.Fatalf("★ 자식 프로세스가 토큰을 봤다 ★\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "OUT=/o") {
		t.Fatalf("OUT 이 자식에 안 갔다\n%s", buf.String())
	}
}

// ★ 구멍이 있던 바로 그 줄을 막는다 ★
//
// 위 시험은 harnessEnv 가 옳다는 것만 보인다. 정작 고친 곳은 exec 지점의
//
//	cmd.Env = append(os.Environ(), …)   →   cmd.Env = env
//
// 이므로, 누가 여기에 os.Environ() 을 다시 얹는 회귀를 잡으려면
// ★ runner 를 통과시켜야 한다 ★. 가짜 하네스를 세워 그 구간을 덮는다.
//
// R3 로 exec 이 runner.go 한 곳에 모였으므로 ★ 이 시험 하나가 모든 어댑터를 덮는다 ★.
func Test환경_runner가_화이트리스트를_쓴다(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh 가 없다")
	}
	os.Setenv("ENODE_TOKEN", "비밀단어청록") //nolint:errcheck
	defer os.Unsetenv("ENODE_TOKEN")   //nolint:errcheck

	dir := t.TempDir()
	// 플래그는 전부 무시하고, 자기가 받은 환경을 파일로 뱉은 뒤 봉투를 찍는다.
	fake := dir + "/fake-harness"
	script := "#!/bin/sh\nenv > \"$OUT/env.txt\"\n" +
		`printf '{"type":"result","subtype":"success","num_turns":1,"session_id":"s1"}` + "\n'\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	_, h := runHarness(context.Background(), claudeHarness{}, fake, Job{
		Prompt: "안녕", IO: IOPaths{Dir: dir, In: dir, Out: dir}})
	if h.Reason != ReasonOK {
		t.Fatalf("가짜 하네스가 완주 안 했다: %+v", h)
	}
	if h.Session != "s1" {
		t.Fatalf("session 을 안 넘겼다: %q", h.Session)
	}

	seen, err := os.ReadFile(dir + "/env.txt")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(seen), "비밀단어청록") {
		t.Fatalf("★ 하네스가 토큰을 봤다 ★\n%s", seen)
	}
	if !strings.Contains(string(seen), "OUT="+dir) {
		t.Fatalf("OUT 이 안 갔다\n%s", seen)
	}
}

// ★ 명령 단계도 화이트리스트다 ★
//
// R1 을 처음 고칠 때 agent 쪽만 막았는데, 계약은 ★ 노드 주인이 아닌 사람 ★ 이
// 낼 수 있고 argv 는 무엇이든 된다 — `sh -c 'env > $OUT/leak'` 이면 끝난다.
// 위협이 같으므로 규칙도 같다.
func Test환경_명령단계도_막힌다(t *testing.T) {
	t.Setenv("ENODE_TOKEN", "비밀")
	t.Setenv("ARCH", "arm")
	t.Setenv("사내_비밀_변수", "새면_안됨")

	env := harnessEnv(commandEnv, map[string]string{"OUT": "/o"}, nil)
	if _, ok := has(env, "ENODE_TOKEN"); ok {
		t.Fatal("★ 명령 단계에 토큰이 갔다 ★")
	}
	if _, ok := has(env, "사내_비밀_변수"); ok {
		t.Fatal("선언 안 한 이름이 통과했다")
	}
	if _, ok := has(env, "ARCH"); !ok {
		t.Fatal("★ ARCH 가 막혔다 ★ — 크로스 빌드가 안 된다")
	}
}

// ★ 계약이 이름을 더할 수 있다 ★ — 값이 아니라 이름이다.
// 값을 계약에 적으면 Run Record 의 manifest 로 봉인되어 영구히 남는다.
func Test환경_계약이_이름을_더한다(t *testing.T) {
	t.Setenv("사내_툴체인_경로", "/opt/tc")

	if _, ok := has(harnessEnv(commandEnv, nil, nil), "사내_툴체인_경로"); ok {
		t.Fatal("선언 없이 통과했다")
	}
	declared := append(append([]string{}, commandEnv...), "사내_툴체인_경로")
	if _, ok := has(harnessEnv(declared, nil, nil), "사내_툴체인_경로"); !ok {
		t.Fatal("★ 계약이 선언했는데 안 통과했다 ★")
	}
}
