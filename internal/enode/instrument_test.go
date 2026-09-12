package enode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 계장 — 가짜 홈과 허용목록과 실패 등급
//
// 이 파일이 재는 것은 두 층이다.
//
//	Instrument   무엇이 쓰이고 무엇이 안 쓰이나.  오류의 등급이 무엇인가
//	runHarness   그 등급이 단계의 운명으로 어떻게 번역되나
//
// 하네스 실행파일이 없어도 돈다 — Instrument 는 t.TempDir() 에 쓰고 난
// 파일을 읽고, 등급의 배선은 가짜 하네스로 잰다.

// noHome 은 이 시험 동안 실제 홈을 빈 곳으로 돌린다.
//
// 안 하면 자격증명 복사가 이 기계의 진짜 ~/.claude 를 읽는다 — 시험이
// 사람의 파일에 기대면 결과가 기계마다 달라진다.
func noHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
}

// 고정값이 가짜 홈을 가리킨다 (features.md 3.1).
//
// 이름이 아니라 값이다 — 통과 목록에 넣으면 노드 환경에 그 변수가 없을 때
// 조용히 안 걸린다.
func TestAdapter_FixedPointsAtTheFakeHome(t *testing.T) {
	got := claudeHarness{}.Fixed("/tmp/inst-1")
	if got["CLAUDE_CONFIG_DIR"] != filepath.Join("/tmp/inst-1", "home") {
		t.Fatalf("the harness home was not pinned: %v", got)
	}
	if got["CLAUDE_CODE_DISABLE_AUTO_MEMORY"] != "1" {
		t.Fatalf("auto memory was left on: %v", got)
	}
}

// 계장이 쓰는 것과 그 자리 — 홈 · 훅 설정 · 허용목록.
func TestAdapter_InstrumentBuildsThePrivateWorld(t *testing.T) {
	noHome(t)
	dir, out := t.TempDir(), t.TempDir()
	flags, err := claudeHarness{}.Instrument(dir, "/usr/bin/enode",
		HookArgs{Out: out, Expect: []string{"plan.json"}}, Components{})
	if err != nil {
		t.Fatalf("instrumentation failed: %v", err)
	}
	home := filepath.Join(dir, "home")
	st, err := os.Stat(home)
	if err != nil || !st.IsDir() {
		t.Fatalf("the fake home is not there: %v", err)
	}
	if st.Mode().Perm() != 0o700 {
		t.Fatalf("the fake home is open to others: %v", st.Mode().Perm())
	}
	// 훅 설정은 홈 안이다 — 밖에 두면 하네스가 읽는 설정의 자리가 둘이 된다.
	if _, err := os.Stat(filepath.Join(home, hookSettingsName)); err != nil {
		t.Fatalf("the hook settings did not land inside the fake home: %v", err)
	}
	// 허용목록은 홈 밖이다.
	allowlist := filepath.Join(dir, mcpAllowlistName)
	b, err := os.ReadFile(allowlist)
	if err != nil {
		t.Fatalf("the allowlist was not written: %v", err)
	}
	if string(b) != `{"mcpServers":{}}`+"\n" {
		t.Fatalf("a step that requested nothing did not get an empty allowlist: %q", b)
	}
	// 팩이 없으면 skills/ 도 agents/ 도 안 만든다 — 없는 것이 오늘의 참이다.
	for _, name := range []string{"skills", "agents"} {
		if _, err := os.Stat(filepath.Join(home, name)); err == nil {
			t.Fatalf("%s was created by a unit that does not own it", name)
		}
	}
	joined := strings.Join(flags, " ")
	for _, want := range []string{
		`--setting-sources `,
		"--settings " + filepath.Join(home, hookSettingsName),
		"--strict-mcp-config",
		"--mcp-config=" + allowlist,
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("%s is missing from the flags: %v", want, flags)
		}
	}
}

// 우리 설정에는 Stop 훅 하나뿐이다 (business-rules.md 6.2 ①).
//
// 「logs/ 의 첫 줄은 init」이 서는 근거가 이것이다 — 실측에서 init 앞에 나온
// 사건 여덟은 전부 SessionStart 훅에서 왔고, 우리가 쓰는 설정에는 그 자리가
// 없다. 여기가 늘면 그 불변식이 조용히 깨진다.
func TestAdapter_OnlyTheStopHookIsPlanted(t *testing.T) {
	noHome(t)
	dir, out := t.TempDir(), t.TempDir()
	if _, err := (claudeHarness{}).Instrument(dir, "/usr/bin/enode",
		HookArgs{Out: out}, Components{}); err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Hooks map[string]any `json:"hooks"`
	}
	b, err := os.ReadFile(filepath.Join(dir, "home", hookSettingsName))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Hooks) != 1 {
		t.Fatalf("the settings carry hooks other than Stop: %v", cfg.Hooks)
	}
	if _, ok := cfg.Hooks["Stop"]; !ok {
		t.Fatalf("the Stop hook is not the one that is there: %v", cfg.Hooks)
	}
}

// self 가 비면 훅 블록만 빠진다 — 계장 전체가 빠지는 것이 아니다.
//
// 오늘은 os.Executable() 하나가 허용목록까지 떨어뜨렸다. 빈 값을 쓰는 것도
// 아니다 — shellJoin 이 앞이 빈 명령을 만들어 하네스가 매 Stop 마다 그것을
// 셸에 넘긴다.
func TestAdapter_AnEmptySelfDropsOnlyTheHooksKey(t *testing.T) {
	personalHome := t.TempDir()
	t.Setenv("HOME", personalHome)
	if err := os.MkdirAll(filepath.Join(personalHome, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	personal := `{"apiKeyHelper": "/opt/gateway/api-key-helper", "env": {"OIDC_CLIENT_ID": "abc"}}`
	if err := os.WriteFile(filepath.Join(personalHome, ".claude", "settings.json"),
		[]byte(personal), 0o600); err != nil {
		t.Fatal(err)
	}

	dir, out := t.TempDir(), t.TempDir()
	flags, err := claudeHarness{}.Instrument(dir, "", HookArgs{Out: out}, Components{})
	if err != nil {
		t.Fatalf("an empty self killed the instrumentation: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "home", hookSettingsName))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if _, ok := got["hooks"]; ok {
		t.Fatalf("a hook command with an empty program was planted: %s", b)
	}
	// 게이트웨이 인증은 그때도 얹힌다 — 사내 노드는 인증이 이 필드로 온다.
	if got["apiKeyHelper"] != "/opt/gateway/api-key-helper" {
		t.Fatalf("the gateway auth field was dropped with the hooks key: %v", got)
	}
	if !strings.Contains(strings.Join(flags, " "), "--strict-mcp-config") {
		t.Fatalf("the allowlist was dropped because self was empty: %v", flags)
	}
}

// 불변식 셋을 한 시험에서 잰다 (application-design.md 4.3).
//
//	①  <dir>/home 이 가장 먼저 만들어져 오류 뒤에도 남아 있다
//	②  보조 실패에도 이미 얻은 플래그를 돌려준다
//	③  보조 오류로 조기 반환하지 않는다 — 허용목록이 쓰인다
//
// 훅 쓰기를 실패시키는 방법은 그 이름을 디렉터리로 먼저 채우는 것이다.
func TestAdapter_TheIsolationFlagsSurviveAnAuxiliaryFailure(t *testing.T) {
	noHome(t)
	dir, out := t.TempDir(), t.TempDir()
	// 홈을 미리 만들고 그 안에 같은 이름의 디렉터리를 둔다 — WriteFile 이 진다.
	home := filepath.Join(dir, "home")
	if err := os.MkdirAll(filepath.Join(home, hookSettingsName), 0o700); err != nil {
		t.Fatal(err)
	}
	flags, err := claudeHarness{}.Instrument(dir, "/usr/bin/enode", HookArgs{Out: out}, Components{})
	if err == nil {
		t.Fatal("a hook settings write that could not happen was reported as success")
	}
	if !errors.Is(err, errAux) {
		t.Fatalf("the hook settings failure was not graded as auxiliary: %v", err)
	}
	// ①
	if st, err := os.Stat(home); err != nil || !st.IsDir() {
		t.Fatalf("the fake home is gone after an auxiliary failure: %v", err)
	}
	// ③ — 조기 반환했으면 이 파일이 없다.
	if _, err := os.Stat(filepath.Join(dir, mcpAllowlistName)); err != nil {
		t.Fatalf("the allowlist was skipped because the hook write failed: %v", err)
	}
	// ②
	joined := strings.Join(flags, " ")
	if !strings.Contains(joined, "--strict-mcp-config") || !strings.Contains(joined, "--setting-sources") {
		t.Fatalf("a layer of the isolation fell with the hook settings: %v", flags)
	}
	// --settings 는 없는 것이 맞다 — 그 파일을 못 썼다. 가리키면 하네스가 안 뜬다.
	if strings.Contains(joined, "--settings ") {
		t.Fatalf("the flags point at a settings file that was never written: %v", flags)
	}
}

// 감싸지 않은 오류는 전부 치명이다 — errAux 의 자리가 훅 설정 하나뿐이다.
//
// 기본이 보조이면 감쌀 자리를 하나 빠뜨리는 실수가 열리는 쪽으로 틀린다 —
// 허용목록이 안 쓰였는데 단계가 돌고 exit 0 으로 성공이 봉인된다.
func TestAdapter_EverythingButTheHookSettingsIsFatal(t *testing.T) {
	t.Run("the fake home cannot be made", func(t *testing.T) {
		noHome(t)
		dir := t.TempDir()
		// 같은 이름의 파일이 있으면 MkdirAll 이 진다.
		if err := os.WriteFile(filepath.Join(dir, "home"), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := claudeHarness{}.Instrument(dir, "/usr/bin/enode", HookArgs{}, Components{})
		assertFatal(t, err, "cannot create the harness home")
	})

	t.Run("the allowlist cannot be written", func(t *testing.T) {
		noHome(t)
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, mcpAllowlistName), 0o700); err != nil {
			t.Fatal(err)
		}
		_, err := claudeHarness{}.Instrument(dir, "/usr/bin/enode", HookArgs{}, Components{})
		assertFatal(t, err, "cannot write mcp allowlist")
		// 불변식 ① — 홈이 가장 먼저다. 뒤에 만들면 이 경로에서 안 남는다.
		if st, err := os.Stat(filepath.Join(dir, "home")); err != nil || !st.IsDir() {
			t.Fatalf("the fake home was not made before the writes that can fail: %v", err)
		}
	})

	t.Run("the credentials are there but unreadable", func(t *testing.T) {
		personalHome := t.TempDir()
		t.Setenv("HOME", personalHome)
		// 원본 자리를 디렉터리로 채운다 — 파일에 닿았는데 못 읽는 경우다.
		if err := os.MkdirAll(filepath.Join(personalHome, ".claude", credentialsName), 0o700); err != nil {
			t.Fatal(err)
		}
		_, err := claudeHarness{}.Instrument(t.TempDir(), "/usr/bin/enode", HookArgs{}, Components{})
		assertFatal(t, err, "cannot copy credentials")
	})
}

func assertFatal(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("a failure that must kill the step was reported as success (wanted %q)", want)
	}
	if errors.Is(err, errAux) {
		t.Fatalf("a fatal failure was graded as auxiliary: %v", err)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("the reason does not say what could not be done: %v (wanted %q)", err, want)
	}
}

// OAuth 자격증명이 가짜 홈으로 따라 들어간다 (ADR-034 §3 ①).
//
// 홈을 옮기면 로그인이 끊겨 Not logged in 이 된다. 복사본은 0600 이고
// 계장 디렉터리와 함께 단계 끝에 지워진다.
func TestAdapter_TheCredentialsRideIntoTheFakeHome(t *testing.T) {
	personalHome := t.TempDir()
	t.Setenv("HOME", personalHome)
	if err := os.MkdirAll(filepath.Join(personalHome, ".claude"), 0o700); err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"claudeAiOauth":{"accessToken":"secret"}}`)
	if err := os.WriteFile(filepath.Join(personalHome, ".claude", credentialsName), body, 0o600); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	if _, err := (claudeHarness{}).Instrument(dir, "/usr/bin/enode", HookArgs{}, Components{}); err != nil {
		t.Fatal(err)
	}
	copied := filepath.Join(dir, "home", credentialsName)
	got, err := os.ReadFile(copied)
	if err != nil {
		t.Fatalf("the login did not follow the home: %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("the credentials were not copied intact: %q", got)
	}
	st, err := os.Stat(copied)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("the copy is looser than the original: %v", st.Mode().Perm())
	}
}

// 자격증명이 없는 것은 정상이다 — 환경변수 노드와 게이트웨이 노드가 그렇다.
func TestAdapter_NoCredentialsIsNormal(t *testing.T) {
	noHome(t)
	dir := t.TempDir()
	if _, err := (claudeHarness{}).Instrument(dir, "/usr/bin/enode", HookArgs{}, Components{}); err != nil {
		t.Fatalf("a node without an oauth login was reported as a failure: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "home", credentialsName)); err == nil {
		t.Fatal("a credentials file appeared where the node had none")
	}
}

// gradeHarness 는 계장의 오류만 정하는 가짜 하네스다.
//
// 등급이 단계의 운명으로 어떻게 번역되는지는 runHarness 의 것이고, 그것을
// 재려면 어댑터가 아니라 인터페이스 쪽에서 오류를 넣어야 한다.
type gradeHarness struct {
	err error
}

func (gradeHarness) Name() string                         { return "grade" }
func (gradeHarness) Env() []string                        { return nil }
func (gradeHarness) Fixed(string) map[string]string       { return nil }
func (gradeHarness) Usable(context.Context, string) error { return nil }
func (gradeHarness) Version(context.Context, string) (string, error) {
	return "", errors.New("no version here")
}
func (h gradeHarness) Instrument(string, string, HookArgs, Components) ([]string, error) {
	return []string{"--flag-we-already-earned"}, h.err
}
func (gradeHarness) Argv(AgentParams, IOPaths) []string { return nil }
func (gradeHarness) Decode(io.Reader, int, func(Event)) HarnessResult {
	return HarnessResult{Reason: ReasonOK}
}

// 등급이 단계의 운명을 가른다 — 치명은 하네스를 안 띄우고 보조는 그대로 간다.
func TestRunHarness_TheGradeDecidesWhetherTheStepRuns(t *testing.T) {
	for _, tc := range []struct {
		name    string
		err     error
		wantRun bool
		want    Reason
	}{
		{"an auxiliary failure still runs the step", fmt.Errorf("%w: hook settings", errAux), true, ReasonOK},
		{"a fatal failure never starts the harness", errors.New("cannot write mcp allowlist: full disk"), false, ReasonError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			ran := filepath.Join(dir, "it-ran")
			bin := writeScript(t, dir, "printf 'x' > "+ran+"\n")

			_, h := runHarness(context.Background(), gradeHarness{err: tc.err}, bin,
				Job{IO: IOPaths{Dir: dir, Out: dir, In: dir}})

			_, err := os.Stat(ran)
			if (err == nil) != tc.wantRun {
				t.Fatalf("the harness ran=%v, want %v", err == nil, tc.wantRun)
			}
			if h.Reason != tc.want {
				t.Fatalf("reason=%s want %s (%s)", h.Reason, tc.want, h.Message)
			}
			if !tc.wantRun && !strings.Contains(h.Message, "cannot write mcp allowlist") {
				t.Fatalf("the reason did not reach the record: %q", h.Message)
			}
		})
	}
}

// 계장 디렉터리를 못 만들면 단계가 실패한다 (decisions.md 6절 ④).
//
// 오늘은 그 실패가 계장을 통째로 건너뛰고 그대로 기동했다 — 가짜 홈이 없고
// 개인 설정과 계정 커넥터가 다 보이는 하네스가 도는 길이다.
func TestRunHarness_AnInstrumentationDirectoryThatCannotBeMadeKillsTheStep(t *testing.T) {
	dir := t.TempDir()
	ran := filepath.Join(dir, "it-ran")
	bin := writeScript(t, dir, "printf 'x' > "+ran+"\n")
	// MkdirTemp 는 os.TempDir() 을 쓰고 그것이 TMPDIR 을 읽는다.
	t.Setenv("TMPDIR", filepath.Join(dir, "no-such-place"))

	_, h := runHarness(context.Background(), claudeHarness{}, bin,
		Job{IO: IOPaths{Dir: dir, Out: dir, In: dir}})

	if _, err := os.Stat(ran); err == nil {
		t.Fatal("the harness started with no instrumentation directory to be isolated in")
	}
	if h.Reason != ReasonError || !strings.Contains(h.Message, "cannot create instrumentation directory") {
		t.Fatalf("the step did not report what it could not do: %+v", h)
	}
}

// 하네스 단계의 링 tee 가 꺼진다 (decisions.md 6절 ⑲).
//
// 한 바이트도 안 받는 것으로 잰다 — 「도구 사건이 안 쌓인다」로 재면 링에도
// 선별을 거는 구현이 통과한다. 그 구현은 실시간 필터라 구조가 다르고,
// 이 팩이 고른 것은 끄는 쪽이다.
func TestRunHarness_TheRingGetsNothingWhileTheStreamIsRaw(t *testing.T) {
	dir := t.TempDir()
	bin := writeScript(t, dir,
		`printf '{"type":"assistant","message":{"content":[{"type":"text","text":"secret"}]}}\n'`+"\n"+
			`printf '{"type":"result","subtype":"success","num_turns":1}\n'`+"\n")

	var ring bytes.Buffer
	_, h := runHarness(context.Background(), claudeHarness{}, bin,
		Job{IO: IOPaths{Dir: dir, Out: dir, In: dir}, Transcript: &ring})

	if h.Reason != ReasonOK {
		t.Fatalf("the stub harness did not complete: %+v", h)
	}
	if ring.Len() != 0 {
		t.Fatalf("the raw stream reached the ring, which lives on the node disk: %q", ring.String())
	}
}

// 단계가 올리는 로그가 걸러진 것이다 (decisions.md 6절 ⑱).
//
// 선별 자체는 logs_test.go 가 재고 이 줄은 배선을 잰다 — 함수가 있어도
// runHarness 가 안 부르면 원문이 그대로 Record 로 올라가 봉인된다.
func TestRunHarness_TheUploadedLogIsTheFilteredOne(t *testing.T) {
	dir := t.TempDir()
	bin := writeScript(t, dir,
		`printf '{"type":"system","subtype":"init","mcp_servers":[]}\n'`+"\n"+
			`printf '{"type":"assistant","message":{"content":[{"type":"text","text":"sk-ant-secret"}]}}\n'`+"\n"+
			`printf '{"type":"result","subtype":"success","num_turns":1}\n'`+"\n")

	logBytes, h := runHarness(context.Background(), claudeHarness{}, bin,
		Job{IO: IOPaths{Dir: dir, Out: dir, In: dir}})

	if h.Reason != ReasonOK {
		t.Fatalf("the stub harness did not complete: %+v", h)
	}
	if strings.Contains(string(logBytes), "sk-ant-secret") {
		t.Fatalf("the raw stream was uploaded as the step log:\n%s", logBytes)
	}
	if !strings.Contains(string(logBytes), "enode.elided") {
		t.Fatalf("the log does not say it was filtered:\n%s", logBytes)
	}
	if !strings.HasPrefix(string(logBytes), `{"type":"system","subtype":"init"`) {
		t.Fatalf("head -1 does not read the init line:\n%s", logBytes)
	}
}

// writeScript 는 봉투를 찍는 가짜 하네스를 하나 만든다.
//
// --version 과 auth status 를 따로 받는 이유 — runner 가 실행 뒤 Version 을
// 부른다. 안 갈라두면 그 호출이 본문을 다시 돌린다.
func writeScript(t *testing.T, dir, body string) string {
	t.Helper()
	path := filepath.Join(dir, "fake-harness")
	src := "#!/bin/sh\ncase \"$1 $2\" in\n" +
		"  '--version ') echo '9.9.9 (fake)'; exit 0 ;;\n" +
		"  'auth status') echo '{\"loggedIn\":true}'; exit 0 ;;\n" +
		"esac\n" + body
	if err := os.WriteFile(path, []byte(src), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}
