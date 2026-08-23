package enode

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
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
//
// ★ git status 로 하지 않는다 ★ — .gitignore 를 지켜서 빌드 산출물을 가린다.
// 기준 시각(stamp) 이 있어야 zImage 도 .ko 도 보인다. changed_test.go 참조.
func TestHook_바뀐_것을_알려준다(t *testing.T) {
	dir, inst := gitInit(t), t.TempDir()
	time.Sleep(1100 * time.Millisecond)
	s := stampNow(dir)
	time.Sleep(1100 * time.Millisecond)
	write(t, dir, "고쳤다.c", "int q;\n")

	sp := filepath.Join(inst, "stamp")
	if err := writeStamp(sp, s); err != nil {
		t.Fatal(err)
	}
	o := hookRun(t, HookArgs{Out: t.TempDir(), Workspace: dir, Expect: []string{"결과"}, Stamp: sp},
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

// ★ 훅이 어긴 계획을 하네스가 끝나기 전에 짚는다 ★ (ADR-046)
//
// 이것이 없으면 어긴 계획은 ★ 계약 적용 시점 ★ 에야 거절되고, 그때는
// 하네스가 이미 끝나 고칠 기회가 없다 — 판 하나가 통째로 버려진다.
// 실측에서 두 번 밟았다 (10차 없는 역할 · 11차 schema 키잉).
func Test훅_어긴_계획을_짚는다(t *testing.T) {
	dir := t.TempDir()
	// ★ 산출물은 냈다 ★ — missingOutputs 는 통과한다. 모양만 틀렸다.
	if err := os.WriteFile(filepath.Join(dir, "plan"),
		[]byte(`{"steps":[{"id":"r","uses":"★없는역할★","run":["true"],"out":["l"]}]}`),
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
		t.Fatalf("★ 막지 않았다 ★: %q", out.String())
	}
	if so.Decision != "block" {
		t.Fatalf("★ 막지 않았다 ★: %+v", so)
	}
	// ★ 오류 문장을 그대로 전한다 ★ — 번역하면 ADR-045 의 실수를 되풀이한다.
	if !strings.Contains(so.Reason, "undeclared role") {
		t.Fatalf("Validate 의 말이 안 실렸다: %s", so.Reason)
	}
	// ★ 어휘도 함께 준다 ★ — 무엇을 써야 하는지 모르면 또 추측한다.
	if !strings.Contains(so.Reason, "planner") {
		t.Fatalf("역할 목록이 안 실렸다: %s", so.Reason)
	}
}

// ★ 정당한 계획은 안 막는다 ★ — 안전망이 정규 경로를 무너뜨리는 것이 가장 나쁘다.
func Test훅_정당한_계획은_통과시킨다(t *testing.T) {
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
		t.Fatalf("★ 정당한 계획을 막았다 ★: %s", out.String())
	}
}

// ★ 계획 단계가 아니면 안 본다 ★ — Plan 이 비면 그냥 지나간다.
func Test훅_계획단계가_아니면_안_본다(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "x"), []byte(`이건 계획이 아니다`), 0o644)
	a := HookArgs{Out: dir, Expect: []string{"x"}} // Plan 이 비어 있다
	var out bytes.Buffer
	if err := RunStopHook(a, strings.NewReader(`{"stop_hook_active":false}`), &out); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Fatalf("★ 계획도 아닌데 막았다 ★: %s", out.String())
	}
}
