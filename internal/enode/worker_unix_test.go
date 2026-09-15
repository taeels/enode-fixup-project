//go:build !windows

package enode

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// 워커의 실행 경로 중 자식 프로세스가 실제로 떠야 재는 것들.
//
// 왜 빌드 태그로 가르는가 — 이 저장소는 OS 분기를 빌드 태그 쌍으로만 하고
// (lock_unix.go/lock_windows.go), U10 이 테스트에서도 같은 길을 골랐다.
// 런타임 t.Skip 으로 가르는 안을 기각한다: CI 의 스킵 감시가 전 패키지를
// 보고, 스킵은 「테스트가 돌아 통과했다」와 「CI 가 초록이다」를 갈라 놓는다.
// 짝이 되는 _windows 파일이 없는 것은 짝이 필요 없어서다 — 컴파일이
// 요구하는 것은 생산 코드의 짝이지 테스트의 짝이 아니다.
//
// 목을 세우지 않는다. 자식은 진짜 프로세스이고, $IN·$OUT 은 execute 가 만든
// 진짜 디렉터리이며, Mediator 는 httptest 다.

// script 는 실행 가능한 sh 스크립트를 t.TempDir() 안에 놓고 절대 경로를 준다.
//
// 절대 경로를 쓰는 이유 — argv[0] 해석은 부모 PATH 로 하지만, 시험이
// 기계의 PATH 에 무엇이 있는지에 기대면 안 된다. 자식 환경에는 PATH 가
// 간다 (harnessEnv 의 baseAllow 가 그것을 담는다) — 스크립트 안에서
// sleep · cat 을 쓸 수 있는 것이 그 때문이다.
func script(t *testing.T, name, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// waitForFile 은 자식이 실제로 떴다는 것을 파일로 기다린다.
// 시간이 아니라 상태로 끝낸다 — 타이밍에 기대면 시험이 흔들린다.
func waitForFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("the child never started: %s was not created", path)
}

// ── execute — 명령 단계 ───────────────────────────────────────

// 명령 단계는 exit_code 로 보고되고 $OUT 에 놓인 것이 produced 가 된다.
//
// argv 의 $OUT 이 풀려야 빌드가 직접 거기에 놓게 시킬 수 있다 (argv.go) —
// 그러면 아무도 산출물 경로를 미리 몰라도 된다.
func TestExecute_ACommandStepReportsItsOutputsAndItsExitCode(t *testing.T) {
	m := newMediator(t)
	w := newWorker(m)
	w.Local.Workspace = t.TempDir()
	holdLease(w)

	sh := script(t, "build", `printf 'compiling\n'
printf 'zImage\n' > "$1"
printf 'touched\n' > out.txt
`)
	step := runStep(sh, "$OUT/artifact", "$WORKSPACE")
	step.CheckChanged = []string{"out.txt"}
	w.execute(context.Background(), step)

	res := m.only(t)
	if res.Error != "" {
		t.Fatalf("a step that ran to completion reported an error: %q", res.Error)
	}
	if res.ExitCode == nil || *res.ExitCode != 0 {
		t.Fatalf("I3 violated: the exit code did not reach the report: %+v", res.ExitCode)
	}
	if !listed(res.Produced, "artifact") {
		t.Fatalf("$OUT was not expanded in argv, so nothing was harvested: %v", res.Produced)
	}
	if body, ok := m.blob("artifact"); !ok || string(body) != "zImage\n" {
		t.Fatalf("the artifact did not reach the mediator: %q", body)
	}
	if got := m.logOf("build"); !strings.Contains(got, "compiling") {
		t.Fatalf("ADR-005 violated: what the step printed is not in the log: %q", got)
	}
	// ADR-037 — 노드는 관찰만 한다. 계약이 지목한 경로가 바뀌었는지를
	// 사실로 싣고, 그것이 성공인지는 Mediator 가 판정한다.
	if !listed(res.Changed, "out.txt") {
		t.Fatalf("ADR-037 violated: the path the contract pointed at was not observed: %v", res.Changed)
	}
}

// 0 이 아닌 종료코드도 완주다 (I3).
//
// 노드가 여기서 판정하면 O4 가 성립하지 않는다 — "테스트가 실패했는데
// Run 은 성공" 을 계약이 표현할 수 없게 된다.
func TestExecute_ANonZeroExitIsStillCompletion(t *testing.T) {
	m := newMediator(t)
	w := newWorker(m)
	holdLease(w)

	sh := script(t, "failing", "printf 'tests failed\\n' >&2\nexit 3\n")
	w.execute(context.Background(), runStep(sh))

	res := m.only(t)
	if res.ExitCode == nil || *res.ExitCode != 3 {
		t.Fatalf("I3 violated: exit 3 did not reach the report as an exit code: %+v", res)
	}
	if res.Error != "" {
		t.Fatalf("I3 violated: the node judged a completed step as failed: %q", res.Error)
	}
	if got := m.logOf("build"); !strings.Contains(got, "tests failed") {
		t.Fatalf("stderr did not reach the log: %q", got)
	}
}

// 없는 입력은 값이다. 크래시가 아니다 (ADR-058 · ADR-023 §6.2.1).
//
// in.from 은 dispatch 로 안 간 가지의 산출물을 가리킬 수 있다. 404 로
// 단계를 죽이면 재계획이 실패의 증거가 없다고 죽는다 (vm-scratch-7).
func TestExecute_AnAbsentInputIsAValueNotACrash(t *testing.T) {
	m := newMediator(t)
	m.gets["prior_log"] = []byte("the earlier step said this\n")
	w := newWorker(m)
	holdLease(w)

	sh := script(t, "list-in", `for f in "$IN"/*; do printf '%s\n' "${f##*/}"; done > "$1"
`)
	step := runStep(sh, "$OUT/listing")
	step.In.From = []string{"prior_log", "vm_caps"} // vm_caps 는 아무도 안 냈다
	w.execute(context.Background(), step)

	res := m.only(t)
	if res.Error != "" {
		t.Fatalf("ADR-058 violated: an absent input killed the step: %q", res.Error)
	}
	listing, ok := m.blob("listing")
	if !ok {
		t.Fatalf("the step did not run at all: %+v", res)
	}
	if !strings.Contains(string(listing), "prior_log") {
		t.Fatalf("the input that exists was not laid into $IN: %q", listing)
	}
	if strings.Contains(string(listing), "vm_caps") {
		t.Fatalf("ADR-058 violated: a half-written file was left behind for an absent input: %q", listing)
	}
}

// 원장 목록은 $IN 에 파일로 깔린다 (ADR-023 §6.3.1 의 (가)).
//
// enode 가 받아서 깐다 — 에이전트가 도구로 직접 부르면 토큰이 그쪽에 간다 (R1).
func TestExecute_TheLedgerListingIsLaidIntoIN(t *testing.T) {
	m := newMediator(t)
	w := newWorker(m)
	holdLease(w)

	sh := script(t, "read-ledger", `cat "$IN/_ledger.json" > "$1"
`)
	step := runStep(sh, "$OUT/seen")
	step.Ledger = json.RawMessage(`[{"name":"artifact","step":"build"}]`)
	w.execute(context.Background(), step)

	res := m.only(t)
	if res.Error != "" {
		t.Fatalf("the step did not complete: %q", res.Error)
	}
	seen, ok := m.blob("seen")
	if !ok || !strings.Contains(string(seen), `"name":"artifact"`) {
		t.Fatalf("ADR-023 violated: the ledger listing did not reach $IN: %q", seen)
	}
}

// 저장소가 없으면 되돌리지 않았다는 사실을 남긴다 (ADR-036).
//
// 준비 안 함과 준비 실패와 워크스페이스 없음이 봉인에서 구분되지 않으면
// 재현 실패의 원인을 못 찾는다.
func TestExecute_AnUnpreparedWorkspaceIsRecordedAsSuch(t *testing.T) {
	m := newMediator(t)
	w := newWorker(m)
	w.Local.Workspace = t.TempDir() // git 저장소가 아니다
	holdLease(w)

	sh := script(t, "noop", "exit 0\n")
	step := runStep(sh)
	step.Workspace = json.RawMessage(`{}`) // repo 를 안 적었다
	w.execute(context.Background(), step)

	res := m.only(t)
	if res.Workspace != PrepUnprepared {
		t.Fatalf("ADR-036 violated: the workspace state was recorded as %q, want %q",
			res.Workspace, PrepUnprepared)
	}
	if res.ExitCode == nil || *res.ExitCode != 0 {
		t.Fatalf("an unprepared workspace stopped the step: %+v", res)
	}
}

// ── 임대 워치독 ───────────────────────────────────────────────

// 실행 중에 임대가 사라지면 그 단계를 끊는다 (ADR-016 · I1).
//
// ADR-010 은 "단계를 시작하기 전에 not_after 를 확인한다" 고 했는데, 그것만
// 으로는 긴 단계가 임대보다 오래 산다. 실측에서 임대가 회수된 뒤 7초를 더
// 돌았고, 그동안 Mediator 는 그 노드를 새 Run 에 줄 수 있다 — 그러면 노드
// 하나가 동시에 두 Run 에 묶인다.
//
// 임대를 Set(nil) 로 지우는 것이 하트비트 응답의 모양이다 (ADR-016) —
// 목록은 델타가 아니라 전부이므로 빠지는 것이 곧 취소 통보다.
func TestExecute_TheLeaseWatchdogAbortsARunningStep(t *testing.T) {
	m := newMediator(t)
	w := newWorker(m)
	holdLease(w)

	marker := filepath.Join(t.TempDir(), "started")
	// exec 으로 갈아탄다 — 자식을 따로 띄우면 임대가 끊겨도 그 손자가
	// stdout 파이프를 붙든 채 살아 남아 cmd.Run 이 안 돌아온다.
	sh := script(t, "long", `printf 'started\n' > "$1"
exec sleep 30
`)

	done := make(chan struct{})
	go func() {
		defer close(done)
		waitForFile(t, marker)
		w.Held.Set(nil) // 하트비트가 이 Run 을 목록에서 뺐다
	}()

	start := time.Now()
	w.execute(context.Background(), runStep(sh, marker))
	<-done

	res := m.only(t)
	if res.Error != "aborted: lease expired" {
		t.Fatalf("I1 violated: the step kept running after its lease was gone (%v elapsed): %+v",
			time.Since(start).Round(time.Millisecond), res)
	}
	if res.ExitCode != nil {
		t.Fatalf("I3 violated: an aborted step reported exit code %d as if it had completed",
			*res.ExitCode)
	}
}

// ── runAgentStep — 하네스 ─────────────────────────────────────

// stubHarness 는 봉투를 내는 가짜 하네스다.
//
// 진짜 claude 를 안 쓴다 — 기계에 하네스가 없다고 이 시험이 조용히 사라지면
// 잃는 것은 커버리지가 아니라 단언 자체다. claude_test.go 의
// TestAdapter_TheVersionRidesTheResult 가 같은 형태를 먼저 썼다.
//
// --version 과 auth status 를 따로 받는 이유 — runner 가 실행 뒤 Probe 를
// 부른다. 안 갈라두면 그 호출이 본문을 다시 돌려 $OUT 을 덮어쓴다.
func stubHarness(t *testing.T, body string) string {
	t.Helper()
	return script(t, "stub-harness", `case "$1 $2" in
  '--version ') echo '9.9.9 (fake)'; exit 0 ;;
  'auth status') echo '{"loggedIn":true}'; exit 0 ;;
esac
`+body)
}

// agent 단계는 완주하면 $OUT 이 걷힌다 (ADR-013 ②기동 → ④수확).
//
// 프롬프트는 stdin 으로 간다 — 스텁이 그것을 $OUT 에 되뱉어, 무엇이
// 실려 갔는지를 Mediator 쪽에서 읽을 수 있게 한다.
func TestRunAgentStep_AStubHarnessCompletesAndTheHarvestFollows(t *testing.T) {
	m := newMediator(t)
	w := newWorker(m)
	w.Local.HarnessBin = stubHarness(t, `cat > "$OUT/prompt"
printf 'the plan\n' > "$OUT/plan.json"
printf '{"type":"result","subtype":"success","num_turns":4,"total_cost_usd":0.5,"session_id":"s1"}\n'
`)
	holdLease(w)

	step := agentStep(`{"model":"opus","max_turns":12}`)
	step.Out = []string{"plan.json"}
	step.In.Prompt = "build a plan for the arm board"
	w.execute(context.Background(), step)

	res := m.only(t)
	if res.Error != "" {
		t.Fatalf("a harness that completed was reported as failed: %q", res.Error)
	}
	if res.Harness == nil || res.Harness.Reason != ReasonOK || res.Harness.Turns != 4 {
		t.Fatalf("ADR-020 violated: the harness envelope did not reach the record: %+v", res.Harness)
	}
	if res.Harness.Version != "9.9.9 (fake)" {
		t.Fatalf("the harness version did not reach the record: %q", res.Harness.Version)
	}
	if !listed(res.Produced, "plan.json") {
		t.Fatalf("ADR-013 violated: $OUT was not harvested: %v", res.Produced)
	}
	if body, ok := m.blob("plan.json"); !ok || string(body) != "the plan\n" {
		t.Fatalf("the artifact did not reach the mediator: %q", body)
	}
	if listed(res.Produced, ".enode-prompt.md") {
		t.Fatalf("the adapter's own prompt file was harvested as an artifact: %v", res.Produced)
	}
	prompt, ok := m.blob("prompt")
	if !ok || !strings.Contains(string(prompt), "build a plan for the arm board") {
		t.Fatalf("the request did not reach the harness on stdin: %q", prompt)
	}
}

// 완주하지 못한 하네스에서는 아무것도 안 걷는다 (ADR-020).
//
// 크래시는 반쯤 쓴 파일을 남길 수 있어 산출물을 믿을 수 없다.
func TestRunAgentStep_AHarnessThatDidNotCompleteHarvestsNothing(t *testing.T) {
	m := newMediator(t)
	w := newWorker(m)
	w.Local.HarnessBin = stubHarness(t, `printf 'half written\n' > "$OUT/plan.json"
printf '{"type":"result","subtype":"error_during_execution","is_error":true}\n'
`)
	holdLease(w)

	step := agentStep(`{}`)
	step.Out = []string{"plan.json"}
	w.execute(context.Background(), step)

	res := m.only(t)
	if res.Harness == nil || res.Harness.Reason != ReasonError {
		t.Fatalf("a crashed harness was normalised as %+v", res.Harness)
	}
	if !strings.HasPrefix(res.Error, "harness: ") {
		t.Fatalf("ADR-020 violated: a crashed harness was reported as completion: %q", res.Error)
	}
	if len(res.Produced) != 0 {
		t.Fatalf("ADR-020 violated: %v was harvested from a harness that crashed", res.Produced)
	}
	if _, ok := m.blob("plan.json"); ok {
		t.Fatal("ADR-020 violated: a half-written file was uploaded as an artifact")
	}
}

// 되먹임은 두 갈래다 (ADR-048).
//
//	계약이 적은 이름   언제나 — 남의 산출물이지 내 앞 시도가 아니다
//	자백 (_cannot)     재시도일 때만 — 첫 시도에 실리면 거짓이 된다
//
// 예전에는 둘 다 attempt > 0 에 묶여 있어, expands 로 붙은 재계획 단계가
// (attempt 0 이다) 앞 단계 로그를 하나도 못 봤다.
func TestRunAgentStep_FeedbackHasTwoLanes(t *testing.T) {
	echoPrompt := `cat > "$OUT/prompt"
printf '{"type":"result","subtype":"success","num_turns":1}\n'
`
	for _, tc := range []struct {
		name       string
		attempt    int
		wantCannot bool
	}{
		{"first attempt", 0, false},
		{"a retry", 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newMediator(t)
			m.gets["build_log"] = []byte("linker error at line 9\n")
			m.gets[cannotName] = []byte("I could not find the toolchain\n")
			w := newWorker(m)
			w.Local.HarnessBin = stubHarness(t, echoPrompt)
			holdLease(w)

			step := agentStep(`{}`)
			step.Feedback = []string{"build_log"}
			step.Attempt = tc.attempt
			w.execute(context.Background(), step)

			prompt, ok := m.blob("prompt")
			if !ok {
				t.Fatalf("the harness never ran: %+v", m.only(t))
			}
			if !strings.Contains(string(prompt), "linker error at line 9") {
				t.Fatalf("ADR-048 violated: a name the contract asked for was not fed back "+
					"on attempt %d", tc.attempt)
			}
			gotCannot := strings.Contains(string(prompt), "I could not find the toolchain")
			if gotCannot != tc.wantCannot {
				t.Fatalf("ADR-048 violated: the confession lane was %v on attempt %d, want %v",
					gotCannot, tc.attempt, tc.wantCannot)
			}
		})
	}
}

// 없는 입력은 프롬프트에 적힌다 (ADR-058 · ADR-020).
//
// 부재를 값으로 나른다 — 조용히 넘어가면 에이전트가 없는 것을 있다고 믿고
// 짓는다. 관찰하고 판단하는 것은 에이전트다.
func TestRunAgentStep_AnAbsentInputIsCarriedIntoThePrompt(t *testing.T) {
	m := newMediator(t)
	w := newWorker(m)
	w.Local.HarnessBin = stubHarness(t, `cat > "$OUT/prompt"
printf '{"type":"result","subtype":"success","num_turns":1}\n'
`)
	holdLease(w)

	step := agentStep(`{}`)
	step.In.From = []string{"vm_caps"} // 아무도 안 냈다
	w.execute(context.Background(), step)

	prompt, ok := m.blob("prompt")
	if !ok {
		t.Fatalf("ADR-058 violated: an absent input killed the agent step: %+v", m.only(t))
	}
	if !strings.Contains(string(prompt), "inputs that were requested but are absent") ||
		!strings.Contains(string(prompt), "vm_caps") {
		t.Fatalf("ADR-058 violated: the absence was not carried into the prompt:\n%s", prompt)
	}
}

func listed(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// 요청한 MCP 가 이 노드에 없으면 하네스가 안 뜬다 (U4 · R4).
//
// 배선을 잰다 — 규칙 자체는 resolve_test.go 가 잰다. 여기서 보는 것은
// 그 거절이 exec 앞이고, 문구가 res.Error 로 봉인까지 간다는 것이다.
// 게이트 CA4 의 셋째 줄이 이 줄이다.
func TestRunAgentStep_AMissingMCPServerDoesNotLaunchTheHarness(t *testing.T) {
	m := newMediator(t)
	w := newWorker(m)
	marker := filepath.Join(t.TempDir(), "launched")
	w.Local.HarnessBin = stubHarness(t, `printf 'x' > `+marker+`
printf '{"type":"result","subtype":"success"}\n'
`)
	w.Local.MCP = map[string]MCPServer{"probe": {Command: "/usr/bin/true"}}
	holdLease(w)

	step := agentStep(`{"mcp":["nope"]}`)
	w.execute(context.Background(), step)

	res := m.only(t)
	if !strings.Contains(res.Error, "mcp server nope is not available on this node") {
		t.Fatalf("the pack's sentence did not reach the record: %q", res.Error)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("the harness ran without the server the contract asked for")
	}
}

// 허용목록에는 요청된 것만 실리고 워크스페이스의 것도 거기 있다 (U4 · CA4).
//
// 스텁이 --mcp-config 가 가리키는 파일을 $OUT 으로 옮겨, 우리가 실제로 쓴
// 것을 Mediator 쪽에서 읽는다. 복제본이 아니라 그 단계가 쓴 파일이다.
func TestRunAgentStep_TheAllowlistCarriesOnlyWhatTheStepRequested(t *testing.T) {
	m := newMediator(t)
	w := newWorker(m)
	w.Local.HarnessBin = stubHarness(t, `for a in "$@"; do
  case "$a" in --mcp-config=*) cp "${a#*=}" "$OUT/allow.json" ;; esac
done
printf '{"type":"result","subtype":"success"}\n'
`)
	w.Local.Workspace = t.TempDir()
	if err := os.WriteFile(filepath.Join(w.Local.Workspace, workspaceMCPName),
		[]byte(`{"mcpServers":{"probe3":{"command":"/usr/bin/true"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	w.Local.MCP = map[string]MCPServer{
		"probe":  {Command: "/usr/bin/true"},
		"probe2": {Command: "/usr/bin/true"},
	}
	holdLease(w)

	step := agentStep(`{"mcp":["probe","probe3"]}`)
	step.Out = []string{"allow.json"}
	w.execute(context.Background(), step)

	res := m.only(t)
	if res.Error != "" {
		t.Fatalf("the step failed: %q", res.Error)
	}
	body, ok := m.blob("allow.json")
	if !ok {
		t.Fatalf("the allowlist did not reach the mediator: %v", res.Produced)
	}
	var got struct {
		MCPServers map[string]map[string]any `json:"mcpServers"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("the allowlist is not valid json: %v\n%s", err, body)
	}
	if _, ok := got.MCPServers["probe"]; !ok {
		t.Fatalf("the node declaration was not carried: %s", body)
	}
	// 워크스페이스의 것은 허용목록에는 실리고 광고에는 안 실린다 (features.md 3.4).
	if got.MCPServers["probe3"]["command"] != "/usr/bin/true" {
		t.Fatalf("the workspace declaration was not carried: %s", body)
	}
	// 종류를 우리가 채운다 — 안 채우면 하네스가 말없이 버린다 (R9).
	if got.MCPServers["probe3"]["type"] != "stdio" {
		t.Fatalf("the kind was not filled in: %s", body)
	}
	if _, ok := got.MCPServers["probe2"]; ok {
		t.Fatalf("a server the step did not request was opened: %s", body)
	}
}
