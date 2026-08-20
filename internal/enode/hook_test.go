package enode

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
			t.Fatalf("훅 출력이 JSON 이 아니다: %q", out.String())
		}
	}
	return o
}

// ★ 한 번만 되묻는다 ★ — stop_hook_active 를 안 보면 영원히 돈다.
func TestHook_두번째는_통과시킨다(t *testing.T) {
	out := t.TempDir()
	o := hookRun(t, HookArgs{Out: out, Expect: []string{"없는것"}},
		StopInput{StopHookActive: true})
	if o.Decision != "" {
		t.Fatalf("★ 두 번째에도 막았다 — 무한 루프다 ★: %+v", o)
	}
}

// 요구된 것이 다 있으면 통과.
func TestHook_다_냈으면_통과(t *testing.T) {
	out := t.TempDir()
	if err := os.WriteFile(filepath.Join(out, "가설"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if o := hookRun(t, HookArgs{Out: out, Expect: []string{"가설"}}, StopInput{}); o.Decision != "" {
		t.Fatalf("다 냈는데 막았다: %+v", o)
	}
}

// ★ 계약이 요구한 이름 중 없는 것을 짚는다 ★ — 막연한 물음이 아니다.
func TestHook_빠진_이름을_짚는다(t *testing.T) {
	out := t.TempDir()
	if err := os.WriteFile(filepath.Join(out, "가설"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	o := hookRun(t, HookArgs{Out: out, Expect: []string{"가설", "테스트소스"}}, StopInput{})
	if o.Decision != "block" {
		t.Fatalf("★ 빠졌는데 안 막았다 ★: %+v", o)
	}
	if !strings.Contains(o.Reason, "테스트소스") {
		t.Fatalf("빠진 이름을 안 짚었다: %q", o.Reason)
	}
	if strings.Contains(o.Reason, "실패") {
		t.Fatalf("★ 훅이 판정했다 ★ — 판정은 success_when 이 한다 (ADR-004·I3): %q", o.Reason)
	}
}

// 요구가 없으면 ★ 아무 말도 안 한다 ★ — 훅이 참견할 근거가 없다.
func TestHook_요구가_없으면_조용하다(t *testing.T) {
	if o := hookRun(t, HookArgs{Out: t.TempDir()}, StopInput{}); o.Decision != "" {
		t.Fatalf("요구가 없는데 막았다: %+v", o)
	}
}

// ★ 입력이 깨져도 통과시킨다 ★ — 훅이 하네스를 막아 세우면 안 된다.
func TestHook_깨진_입력은_통과(t *testing.T) {
	var out bytes.Buffer
	if err := RunStopHook(HookArgs{Expect: []string{"x"}},
		strings.NewReader("이건 JSON 이 아니다"), &out); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Fatalf("★ 깨진 입력에 막았다 ★: %s", out.String())
	}
}

// 바뀐 파일을 참고로 보여준다 — ★ 판정이 아니라 알림이다 ★.
func TestHook_바뀐_것을_알려준다(t *testing.T) {
	dir := gitInit(t)
	write(t, dir, "고쳤다.c", "int q;\n")
	o := hookRun(t, HookArgs{Out: t.TempDir(), Workspace: dir, Expect: []string{"결과"}},
		StopInput{})
	if !strings.Contains(o.Reason, "고쳤다.c") {
		t.Fatalf("바뀐 것을 안 알려줬다: %q", o.Reason)
	}
}

// ★ 설정 파일이 $OUT 밖에 놓인다 ★ — 안에 두면 ④수확이 산출물로 걷어 올린다.
func TestHook_설정이_OUT밖에_놓인다(t *testing.T) {
	inst, out := t.TempDir(), t.TempDir()
	flags, err := WriteHookSettings(inst, "/usr/bin/enode", HookArgs{Out: out, Expect: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(harvest(out)) != 0 {
		t.Fatalf("★ 계장 파일이 $OUT 에 들어갔다 ★: %v", harvest(out))
	}
	// ★ 개인 설정을 차단한다 ★ (R6) — 안 하면 노드마다 결과가 달라진다.
	j := strings.Join(flags, " ")
	if !strings.Contains(j, "--setting-sources") {
		t.Fatalf("개인 설정을 안 막았다: %v", flags)
	}
	b, err := os.ReadFile(filepath.Join(inst, "enode-settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"Stop"`) || !strings.Contains(string(b), "hook stop") {
		t.Fatalf("Stop 훅이 안 심겼다:\n%s", b)
	}
}

// 공백이 든 경로가 ★ 훅 명령에서 안 깨진다 ★ — 셸이 한 줄로 받기 때문이다.
func TestHook_공백_경로가_안_깨진다(t *testing.T) {
	inst := t.TempDir()
	out := filepath.Join(t.TempDir(), "빈 칸 있는 폴더")
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
		t.Fatalf("★ 공백 경로가 인용되지 않았다 ★: %s", cmd)
	}
}
