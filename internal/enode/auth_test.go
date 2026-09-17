package enode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeAuthSettings 는 사람의 settings.json 한 장을 쓴다.
//
// 인증 필드 둘과 그 밖의 것을 함께 담는다 — 거르는 것을 재려면 걸러질 것이
// 파일에 있어야 한다.
func writeAuthSettings(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	body := `{
	  "apiKeyHelper": "/opt/corp/issue-key.sh",
	  "env": {"CORP_OIDC_ISSUER": "https://sso.corp.invalid"},
	  "permissions": {"allow": ["Bash(rm:*)"]},
	  "model": "some-other-model"
	}`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// readSettings 는 계장이 가짜 홈에 쓴 것을 읽는다.
func readSettings(t *testing.T, home string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(home, hookSettingsName))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

// 노드가 어느 파일에서 인증을 길어올지 고른다 — 그리고 고르는 것은 파일뿐이다.
//
// 한 기계가 갈래를 둘 쓴다 (사내 실측 2026-09-17). 사람은 settings 를 둘로
// 나눠 두고 셸 별칭으로 --settings 를 갈아끼우는데, 노드에는 그 손이 없었다.
//
// 거르는 쪽이 여전히 enode 다 — 노드 주인이 가리키는 것은 파일이지 무엇을
// 읽을지가 아니다. permissions 와 model 이 그 파일에 적혀 있어도 안 실린다.
func TestTheNodeChoosesWhichFileTheAuthComesFrom(t *testing.T) {
	settings := writeAuthSettings(t, "settings-corp.json")
	home := t.TempDir()

	if _, err := WriteHookSettings(home, "/usr/bin/enode",
		HookArgs{Out: t.TempDir()}, AuthSettings(settings)); err != nil {
		t.Fatalf("writing the harness settings failed: %v", err)
	}

	got := readSettings(t, home)
	if got["apiKeyHelper"] != "/opt/corp/issue-key.sh" {
		t.Fatalf("apiKeyHelper did not come from the file the node chose: %v", got["apiKeyHelper"])
	}
	env, ok := got["env"].(map[string]any)
	if !ok || env["CORP_OIDC_ISSUER"] != "https://sso.corp.invalid" {
		t.Fatalf("env did not come from the file the node chose: %v", got["env"])
	}
	for _, k := range []string{"permissions", "model"} {
		if _, has := got[k]; has {
			t.Fatalf("%q rode along; only the auth fields may be copied", k)
		}
	}
	// 훅은 그대로 있다 — 인증 필드가 훅 블록을 밀어내지 않는다.
	if _, has := got["hooks"]; !has {
		t.Fatal("the stop hook is gone from the settings the instrumentation wrote")
	}
}

// 사람이 가리킨 인증 구성이 없으면 단계를 죽인다 — 보조가 아니다.
//
// 훅 쓰기 실패는 보조다 (errAux). 인증은 아니다: 없는 채로 돌면 하네스가
// 떠서 Not logged in 으로 늦게 죽고, 그때는 임대와 예산을 이미 썼다.
// copyCredentials 가 치명인 이유와 같다 — 실패를 앞으로 당긴다.
func TestAMissingAuthSettingsKillsTheStep(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(t.TempDir(), "settings-corp.json")

	_, err := claudeHarness{}.Instrument(dir, "/usr/bin/enode",
		HookArgs{Out: t.TempDir()}, Components{}, AuthSettings(missing))
	if err == nil {
		t.Fatal("a missing auth settings file passed silently")
	}
	if errors.Is(err, errAux) {
		t.Fatalf("the missing auth settings was graded as auxiliary: %v", err)
	}
	if !strings.Contains(err.Error(), missing) {
		t.Fatalf("the error does not say which file is missing: %v", err)
	}
}

// 아무것도 안 적은 노드는 오늘 그대로 돈다 — 기본 자리에 파일이 없어도 조용하다.
//
// 그 자리에 파일이 없는 것은 흔한 정상이다: 개인 구독 · Bedrock · Vertex 는
// 인증을 이 파일에 안 둔다. 여기서 죽이면 오늘 도는 노드 대부분이 멈춘다.
func TestTheDefaultAuthSettingsStaysQuietWhenItIsAbsent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // 윈도우의 os.UserHomeDir 은 이것을 본다

	fields, err := readAuthFields("")
	if err != nil {
		t.Fatalf("an absent default settings file was treated as an error: %v", err)
	}
	if len(fields) != 0 {
		t.Fatalf("fields appeared out of nowhere: %v", fields)
	}
}

// fakeClaudeWithSettings 는 --settings 를 받아 적고 auth status 에 답하는 가짜다.
//
// 앞의 전역 플래그를 건너뛴다 — 하위명령을 그 뒤에서 찾는다. 받은 argv 는
// 파일로 남긴다: 프로브가 실행과 같은 파일로 물었는지는 그것으로만 잰다.
func fakeClaudeWithSettings(t *testing.T, authJSON string) (bin, argvLog string) {
	t.Helper()
	dir := t.TempDir()
	bin = filepath.Join(dir, "claude")
	argvLog = filepath.Join(dir, "argv")
	script := "#!/bin/sh\n" +
		"echo \"$@\" >> " + shQuote(argvLog) + "\n" +
		"if [ \"$1\" = '--settings' ]; then shift 2; fi\n" +
		"case \"$1 $2\" in\n" +
		"  '--version ') echo 'fake 9.9.9 (Claude Code)'; exit 0 ;;\n" +
		"  'auth status') printf '%s\\n' " + shQuote(authJSON) + "; exit 0 ;;\n" +
		"esac\nexit 0\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin, argvLog
}

// 광고가 실행과 같은 파일로 묻는다 (features.md 3.1).
//
// 프로브가 기본 자리를 보고 실행이 다른 파일을 보면 「쓸 수 있다」가 다른
// 갈래의 답이 된다. 갈래가 둘인 기계에서 그 어긋남이 곧 매칭 통과 · 실행
// 시점 죽음이다 — ADR-059 가 --version 에서 닫은 모양이 이 축에서 되살아난다.
func TestTheProbeAsksWithTheSameFileTheRunWillUse(t *testing.T) {
	settings := writeAuthSettings(t, "settings-corp.json")
	bin, argvLog := fakeClaudeWithSettings(t, `{"loggedIn":true}`)

	if err := (claudeHarness{}).Usable(context.Background(), bin, AuthSettings(settings)); err != nil {
		t.Fatalf("rejected while logged in: %v", err)
	}
	b, err := os.ReadFile(argvLog)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "--settings "+settings) {
		t.Fatalf("the probe did not ask with the node's settings file: %q", string(b))
	}
}

// 사람이 가리킨 인증 구성을 못 읽으면 하네스가 광고에서 빠진다 (ADR-012 · ADR-059).
//
// 「있다」와 「쓸 수 있다」를 가르는 축이 하나 늘었다 — 설치돼 있고 로그인도
// 돼 있는데 이 노드가 쓰기로 한 구성이 없으면 그 구성으로는 못 한다.
func TestAnUnreadableAuthSettingsDropsTheHarnessFromTheAdvert(t *testing.T) {
	bin, _ := fakeClaudeWithSettings(t, `{"loggedIn":true}`)
	missing := filepath.Join(t.TempDir(), "settings-corp.json")

	caps := Detect(context.Background(), Local{
		HarnessBin: bin,
		Auth:       &HarnessAuth{Name: "corp", Settings: missing},
		Arch:       "arm64", Workspace: t.TempDir(),
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	for _, c := range caps {
		if _, has := c.Attrs["harness"]; has {
			t.Fatalf("the harness rode the advert without the auth it runs with: %+v", c.Attrs)
		}
		if _, has := c.Attrs["auth"]; has {
			t.Fatalf("the auth name rode the advert without its file: %+v", c.Attrs)
		}
	}
	// 다른 능력은 남는다 — 인증 구성이 없다고 빌드까지 못 하는 것은 아니다.
	if len(caps) == 0 {
		t.Fatal("arch exists yet the whole advert is empty")
	}
}

// 광고가 어느 갈래로 도는지 말한다 — 그리고 그것은 선언이 아니라 통과한 사실이다.
//
// labels 로 적으면 파일이 없어져도 노드가 계속 그 이름을 외친다. 여기는
// Usable() 뒤라 「그 구성으로 지금 일을 시킬 수 있다」가 이미 참이다.
// 그래서 낡은 labels 가 남아 있어도 탐지가 이긴다 (capabilities 의 규칙).
func TestTheAdvertSaysWhichAuthTheNodeRunsWith(t *testing.T) {
	settings := writeAuthSettings(t, "settings-corp.json")
	bin, _ := fakeClaudeWithSettings(t, `{"loggedIn":true}`)

	caps := Detect(context.Background(), Local{
		HarnessBin: bin,
		Auth:       &HarnessAuth{Name: "corp", Settings: settings},
		Labels:     map[string]string{"auth": "stale-bedrock"},
		Workspace:  t.TempDir(),
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if len(caps) != 1 {
		t.Fatalf("expected one capability, got %d", len(caps))
	}
	if got := caps[0].Attrs["auth"]; got != "corp" {
		t.Fatalf("the advert does not say which auth the node runs with: %q", got)
	}
	if _, has := caps[0].Attrs["harness.claude"]; !has {
		t.Fatalf("the harness is missing from the advert: %+v", caps[0].Attrs)
	}
}

// 이름만 적은 노드도 광고한다 — 기본 자리로 도는 갈래에 이름을 붙이는 것이다.
func TestAnAuthNameWithoutAFileStillRidesTheAdvert(t *testing.T) {
	bin, _ := fakeClaudeWithSettings(t, `{"loggedIn":true}`)
	caps := Detect(context.Background(), Local{
		HarnessBin: bin,
		Auth:       &HarnessAuth{Name: "bedrock"},
		Workspace:  t.TempDir(),
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if len(caps) != 1 || caps[0].Attrs["auth"] != "bedrock" {
		t.Fatalf("the default branch could not be named: %+v", caps)
	}
}

// 설정 파일의 ~ 는 홈으로 편다. 그리고 애매한 경로는 노드를 안 띄운다.
//
// yaml 에 적힌 ~ 는 셸을 안 지나므로 글자 그대로 남는다 — 사람은 셸에서
// 그것이 펴지는 것을 보고 살아서 여기에도 적는다.
func TestConfig_TheAuthSettingsPathIsResolvedWhenTheNodeLoads(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	l, err := LoadLocal(writeConfig(t, "mediator: http://m.invalid:8080\ntoken: t\n"+
		"harness_auth:\n  name: corp\n  settings: ~/.claude/settings-corp.json\n"))
	if err != nil {
		t.Fatalf("a ~ path was rejected: %v", err)
	}
	want := filepath.Join(home, ".claude", "settings-corp.json")
	if l.Auth.Settings != want {
		t.Fatalf("~ was not expanded: %q want %q", l.Auth.Settings, want)
	}
	if l.harnessAuthName() != "corp" {
		t.Fatalf("the auth name was lost: %q", l.harnessAuthName())
	}
}

// 상대경로는 거절한다 — 노드의 작업 디렉터리는 이 파일이 정하는 것이 아니라서
// 어디를 가리키는지 사람이 못 읽는다. 틀리면 닫히는 쪽으로 틀린다.
func TestConfig_ARelativeAuthSettingsPathStopsTheNode(t *testing.T) {
	_, err := LoadLocal(writeConfig(t, "mediator: http://m.invalid:8080\ntoken: t\n"+
		"harness_auth:\n  settings: settings-corp.json\n"))
	if err == nil {
		t.Fatal("a relative auth settings path was accepted")
	}
	if !strings.Contains(err.Error(), "absolute path") {
		t.Fatalf("the message does not tell the reader what to write: %v", err)
	}
}

// 아무것도 안 적은 설정은 오늘 그대로다 — 빈 값이 기본 자리를 뜻한다.
func TestConfig_ANodeWithoutHarnessAuthKeepsTodaysBehaviour(t *testing.T) {
	l, err := LoadLocal(writeConfig(t, "mediator: http://m.invalid:8080\ntoken: t\n"))
	if err != nil {
		t.Fatal(err)
	}
	if l.harnessAuth() != "" || l.harnessAuthName() != "" {
		t.Fatalf("an unset harness_auth did not stay empty: %q %q",
			l.harnessAuth(), l.harnessAuthName())
	}
}

// setup 이 쓴 harness_auth 를 LoadLocal 이 그대로 읽는다 — 경로는 펴진 채로.
//
// setup 과 LoadLocal 이 같은 resolveAuth 를 쓴다. 두 벌로 두면 한쪽이 ~ 를
// 펴고 다른 쪽이 안 펴는 날이 오고, 그때 노드는 있지도 않은 파일을 가리킨다.
func TestSetup_TheAuthProfileSurvivesTheRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	path := filepath.Join(t.TempDir(), "corp.yaml")
	var out bytes.Buffer
	code := Setup(SetupOptions{
		Name: "corp", Path: path,
		Mediator: mediatorSaying(t, 200), Token: "shared-secret",
		Workspace:    t.TempDir(),
		Arch:         "arm64",
		AuthName:     "corp",
		AuthSettings: "~/.claude/settings-corp.json",
		Yes:          true, Out: &out, In: strings.NewReader(""),
	})
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out.String())
	}

	l, err := LoadLocal(path)
	if err != nil {
		t.Fatalf("what setup wrote does not parse back: %v", err)
	}
	if l.Auth == nil || l.Auth.Name != "corp" {
		t.Fatalf("the auth name did not survive: %+v", l.Auth)
	}
	want := filepath.Join(home, ".claude", "settings-corp.json")
	if l.Auth.Settings != want {
		t.Fatalf("the auth settings path did not survive expanded: %q want %q",
			l.Auth.Settings, want)
	}
}

// 애매한 경로는 setup 도 안 쓴다. LoadLocal 과 같은 자리에서 같은 이유로 막는다.
func TestSetup_ARelativeAuthSettingsPathWritesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corp.yaml")
	var out bytes.Buffer
	code := Setup(SetupOptions{
		Name: "corp", Path: path,
		Mediator: mediatorSaying(t, 200), Token: "t",
		AuthSettings: "settings-corp.json",
		Yes:          true, Out: &out, In: strings.NewReader(""),
	})
	if code == 0 {
		t.Fatalf("a relative auth settings path was accepted: %s", out.String())
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("setup wrote a config it had already refused")
	}
	if !strings.Contains(out.String(), "absolute path") {
		t.Fatalf("the message does not tell the reader what to write: %s", out.String())
	}
}

// 아무것도 안 적은 노드는 오늘 그대로 ~/.claude/settings.json 에서 길어온다.
//
// 이 갈래가 오늘 도는 게이트웨이 노드다 (보드 노드 실측 2026-09-01/02).
// harness_auth 를 더하면서 그 자리가 조용히 옮겨가면 그 노드들이 다음 배포에서
// 인증을 잃는다.
func TestTheDefaultAuthSettingsIsStillTheHomeOne(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o700); err != nil {
		t.Fatal(err)
	}
	body := `{"apiKeyHelper":"/opt/corp/issue-key.sh","permissions":{"allow":["Bash(rm:*)"]}}`
	if err := os.WriteFile(filepath.Join(home, ".claude", defaultAuthSettings),
		[]byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	fields, err := readAuthFields("")
	if err != nil {
		t.Fatal(err)
	}
	if fields["apiKeyHelper"] != "/opt/corp/issue-key.sh" {
		t.Fatalf("the default settings file was not read: %v", fields)
	}
	if _, has := fields["permissions"]; has {
		t.Fatal("permissions rode along from the default settings file")
	}
}

// 사람이 가리킨 파일이 json 이 아니면 말해 준다 — 조용히 빈 채로 돌지 않는다.
//
// 기본 자리라면 조용한 것이 맞다 (없는 것이 흔한 정상이다). 가리킨 자리는
// 다르다: 적은 사람은 그것으로 돌릴 셈이었다.
func TestAMalformedAuthSettingsSaysSoInsteadOfRunningWithout(t *testing.T) {
	p := filepath.Join(t.TempDir(), "settings-corp.json")
	if err := os.WriteFile(p, []byte("{this is not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := readAuthFields(AuthSettings(p))
	if err == nil {
		t.Fatal("a malformed auth settings file passed silently")
	}
	if !strings.Contains(err.Error(), "not valid json") {
		t.Fatalf("the message does not say what is wrong: %v", err)
	}
}

// ~ 하나만 적은 경우도 편다. 홈 자체를 가리키는 것은 파일이 아니므로 뒤에서
// 못 읽는 것으로 걸리지만, 펴는 규칙에 구멍을 두면 그 실패가 엉뚱해 보인다.
func TestConfig_ATildeOnItsOwnExpandsToTheHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	got, err := expandHome("~")
	if err != nil {
		t.Fatal(err)
	}
	if got != home {
		t.Fatalf("~ did not expand to the home: %q want %q", got, home)
	}
	// ~ 가 앞에 없는 경로는 그대로 둔다 — 펴는 자리가 경로를 안 바꾼다.
	if got, err := expandHome("/etc/enode/settings.json"); err != nil || got != "/etc/enode/settings.json" {
		t.Fatalf("an absolute path was rewritten: %q %v", got, err)
	}
}
