//go:build !windows

package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

// handleNewWork 는 오케스트레이터를 실제로 띄우므로 가짜 실행 파일이 필요하고,
// 그것을 sh 스크립트로 두는 것은 윈도우에서 성립하지 않는다. 런타임 GOOS
// 분기 대신 빌드 태그로 가른다 (orchestrator_unix_test.go 와 같은 이유).

// liveOrchestrator 는 광고를 마치고 계속 살아 있는 가짜다.
// $$ 를 남기므로 시험이 끝에서 직접 거둘 수 있다 — exec 로 바꿔치기하므로
// 그 pid 가 곧 sleep 의 pid 다.
const liveOrchestrator = "printf '%s\\n' $$ > \"$5.pid\"\nprintf 'node-abc\\n' > \"$5\"\nexec sleep 30\n"

func newWorkConfig(t *testing.T, bin string, readySeconds int) *Config {
	t.Helper()
	cfg := flowConfig()
	cfg.Executor = ExecutorConfig{As: "worker", Attrs: map[string]string{"harness": "claude"}}
	cfg.Orchestrator = OrchestratorConfig{
		Bin:           bin,
		WorkspaceRoot: t.TempDir(),
		ReadySeconds:  readySeconds,
	}
	return cfg
}

func workspaceOf(cfg *Config) string {
	return filepath.Join(cfg.Orchestrator.WorkspaceRoot, "orch-EP-2")
}

// reapAtEnd 는 detach 된 가짜 오케스트레이터를 시험 끝에서 거둔다.
func reapAtEnd(t *testing.T, cfg *Config) {
	t.Helper()
	t.Cleanup(func() {
		raw, err := os.ReadFile(filepath.Join(workspaceOf(cfg), "ready.pid"))
		if err != nil {
			return
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
		if err != nil {
			return
		}
		_ = syscall.Kill(pid, syscall.SIGKILL)
	})
}

// run_id 를 이슈에서 유도한다 — 못 정하면 그 뒤가 전부 무의미하므로
// 오케스트레이터를 띄우기 전에 멈춘다.
func TestAdapter_HandleNewWorkNeedsARunIDFirst(t *testing.T) {
	f := &iapFake{}
	m := &medFake{status: map[string]int{"/v1/runs/": 503}}
	cfg := newWorkConfig(t, "/nonexistent/enode", 1)
	a := newFlowAdapter(t, f, m, cfg)

	a.handleNewWork(context.Background(), issueRun(7, "delegation"))

	comments, _, moved, _ := f.snapshot()
	if len(comments) != 0 || len(moved) != 0 {
		t.Fatalf("it touched the issue before it had a run_id: %v %v", comments, moved)
	}
	res := lastResult(t, f)
	if res["status"] != "failed" {
		t.Fatalf("status = %v", res["status"])
	}
	if e, _ := res["error"].(string); !strings.Contains(e, "run_id") {
		t.Fatalf("error = %q", e)
	}
	if _, err := os.Stat(workspaceOf(cfg)); err == nil {
		t.Fatal("a workspace was created for a job that never got a run_id")
	}
}

// 이슈를 못 읽어도 프롬프트만으로 간다 — 위임 프롬프트에 이미 일감이 들어
// 있고, 제목·본문은 계약을 더 잘 만들 뿐 없어서 못 할 일은 아니다.
func TestAdapter_HandleNewWorkUsesThePromptWhenTheIssueCannotBeRead(t *testing.T) {
	f := &iapFake{issueStatus: 500}
	cfg := newWorkConfig(t, "/nonexistent/enode", 1)
	a := newFlowAdapter(t, f, &medFake{}, cfg)

	a.handleNewWork(context.Background(), issueRun(7, "delegation"))

	_, _, moved, _ := f.snapshot()
	// 시작 칸으로 옮겼다는 것이 「이슈를 못 읽고도 계속 갔다」의 관찰이다.
	if len(moved) == 0 || moved[0] != 14 {
		t.Fatalf("moved = %v, want it to move to the start column anyway", moved)
	}
}

// 띄우지 못하면 이슈를 실패로 닫는다 — 안 그러면 러너가 리스 만료까지
// 매달리고 사람은 이유를 못 본다.
func TestAdapter_HandleNewWorkFailsOutWhenTheOrchestratorWillNotStart(t *testing.T) {
	f := &iapFake{}
	cfg := newWorkConfig(t, filepath.Join(t.TempDir(), "absent-enode"), 1)
	a := newFlowAdapter(t, f, &medFake{}, cfg)

	a.handleNewWork(context.Background(), issueRun(7, "delegation"))

	comments, _, moved, _ := f.snapshot()
	if len(comments) != 1 || !strings.Contains(comments[0], "오케스트레이터를 못 띄웠다") {
		t.Fatalf("the reason never reached the issue: %v", comments)
	}
	// 시작 칸으로 옮겼다가 실패 칸으로 옮긴다.
	if len(moved) != 2 || moved[0] != 14 || moved[1] != 12 {
		t.Fatalf("moved = %v, want start then failure", moved)
	}
	if lastResult(t, f)["status"] != "failed" {
		t.Fatal("a job that never started was reported as success")
	}
}

// 오케스트레이터가 아직 안 떴는데 Run 을 내면 422 다 (adapter-example §1.2 ⑤) —
// 그래서 광고를 기다리고, 끝내 안 보이면 Run 을 안 낸다.
func TestAdapter_HandleNewWorkDoesNotSubmitBeforeTheFleetSeesTheNode(t *testing.T) {
	f := &iapFake{}
	m := &medFake{} // 광고에 아무것도 없다
	cfg := newWorkConfig(t, fakeEnode(t, liveOrchestrator), 1)
	reapAtEnd(t, cfg)
	a := newFlowAdapter(t, f, m, cfg)

	a.handleNewWork(context.Background(), issueRun(7, "delegation"))

	if submitted, _, _ := m.snapshot(); len(submitted) != 0 {
		t.Fatalf("a contract went out before the node was visible: %v", submitted)
	}
	comments, _, moved, _ := f.snapshot()
	if len(comments) != 1 || !strings.Contains(comments[0], "함대에 안 나타났다") {
		t.Fatalf("the reason never reached the issue: %v", comments)
	}
	if len(moved) != 2 || moved[1] != 12 {
		t.Fatalf("moved = %v, want it closed into the failure column", moved)
	}
	// 띄운 것은 거둔다 — 실패한 판마다 좀비가 남으면 안 된다.
	if _, err := os.Stat(workspaceOf(cfg)); err == nil {
		t.Fatal("the workspace of a failed start was left behind")
	}
}

func TestAdapter_HandleNewWorkFailsOutWhenTheRunIsRejected(t *testing.T) {
	f := &iapFake{}
	m := &medFake{
		caps:         `{"capabilities":[{"capability":"orchestration","nodes":1,"attrs":{"issue":["EP-2"]}}]}`,
		submitStatus: 422,
	}
	cfg := newWorkConfig(t, fakeEnode(t, liveOrchestrator), 5)
	reapAtEnd(t, cfg)
	a := newFlowAdapter(t, f, m, cfg)

	a.handleNewWork(context.Background(), issueRun(7, "delegation"))

	comments, _, moved, _ := f.snapshot()
	if len(comments) != 1 || !strings.Contains(comments[0], "Run 제출이 실패했다") {
		t.Fatalf("the rejection never reached the issue: %v", comments)
	}
	if len(moved) != 2 || moved[1] != 12 {
		t.Fatalf("moved = %v", moved)
	}
	if lastResult(t, f)["status"] != "failed" {
		t.Fatal("a rejected Run was reported as success")
	}
}

// 되묻기로 손을 뗄 때는 안 죽인다 — 답이 오면 그 자리에서 이어야 한다
// (ADR-047 이 그동안 임대를 안 죽인다). 그래도 거두기는 한다.
//
// 워크스페이스가 남아 있는지로 가른다: Stop 은 자식을 죽이고 그 자리에서
// 지우고, Detach 는 자식이 살아 있는 동안 안 지운다.
func TestAdapter_HandleNewWorkKeepsTheOrchestratorWhenItHandsOff(t *testing.T) {
	f := &iapFake{}
	m := &medFake{
		caps:     `{"capabilities":[{"capability":"orchestration","nodes":1,"attrs":{"issue":["EP-2"]}}]}`,
		onSubmit: map[string]string{"itsaplan-EP-2-1": `{"run_id":"itsaplan-EP-2-1","state":"ASKED"}`},
		asks:     `{"asks":[{"run_id":"itsaplan-EP-2-1","seq":4,"step":"gate","prompt":"진행할까요"}]}`,
	}
	cfg := newWorkConfig(t, fakeEnode(t, liveOrchestrator), 5)
	reapAtEnd(t, cfg)
	a := newFlowAdapter(t, f, m, cfg)

	a.handleNewWork(context.Background(), issueRun(7, "delegation"))

	comments, _, moved, _ := f.snapshot()
	if len(comments) != 1 || !strings.Contains(comments[0], "확인이 필요합니다") {
		t.Fatalf("the question was not relayed: %v", comments)
	}
	if len(moved) != 2 || moved[1] != 13 {
		t.Fatalf("moved = %v, want the ask column", moved)
	}
	if lastResult(t, f)["status"] != "success" {
		t.Fatal("letting go was reported as a failure - the lease would be burned")
	}
	if _, err := os.Stat(workspaceOf(cfg)); err != nil {
		t.Fatalf("the orchestrator was torn down while a human was still answering: %v", err)
	}
}

// Run 이 끝나면 코멘트 · 칸 이동 · result 이고, 오케스트레이터는 놓아준다.
func TestAdapter_HandleNewWorkSubmitsAndThenReleasesTheOrchestrator(t *testing.T) {
	f := &iapFake{issue: `{"id":2,"sequenceNumber":14,"title":"the title","description":"the body"}`}
	m := &medFake{
		caps: `{"capabilities":[{"capability":"orchestration","nodes":1,"attrs":{"issue":["EP-2"]}}]}`,
		onSubmit: map[string]string{"itsaplan-EP-2-1": `{"run_id":"itsaplan-EP-2-1","state":"SUCCEEDED",
		  "verdict":{"state":"SUCCEEDED","checks":[{"step":"plan","what":"produced","ok":true}]}}`},
	}
	cfg := newWorkConfig(t, fakeEnode(t, liveOrchestrator), 5)
	reapAtEnd(t, cfg)
	a := newFlowAdapter(t, f, m, cfg)

	a.handleNewWork(context.Background(), issueRun(7, "delegation"))

	submitted, _, _ := m.snapshot()
	if len(submitted) != 1 {
		t.Fatalf("submitted = %v, want exactly one contract", submitted)
	}
	var contract map[string]any
	if err := json.Unmarshal([]byte(submitted[0]), &contract); err != nil {
		t.Fatal(err)
	}
	if contract["run_id"] != "itsaplan-EP-2-1" {
		t.Fatalf("run_id = %v", contract["run_id"])
	}
	// 이름표가 없으면 남의 이슈의 오케스트레이터가 걸린다.
	reqs, _ := contract["requires"].([]any)
	if len(reqs) == 0 {
		t.Fatalf("requires = %v", contract["requires"])
	}
	planner, _ := reqs[0].(map[string]any)
	if planner["capability"] != "orchestration" || planner["issue"] != "EP-2" {
		t.Fatalf("planner requirement = %v", planner)
	}

	comments, _, moved, _ := f.snapshot()
	if len(comments) != 1 || !strings.Contains(comments[0], ResultMarker("itsaplan-EP-2-1")) {
		t.Fatalf("the result comment is missing or unmarked: %v", comments)
	}
	if len(moved) != 2 || moved[0] != 14 || moved[1] != 11 {
		t.Fatalf("moved = %v, want start then success", moved)
	}
	if lastResult(t, f)["status"] != "success" {
		t.Fatal("a SUCCEEDED run was reported as a failure")
	}
	// --once 가 탄력 노드를 닫지만, 이 판은 어댑터가 놓아준 것이다.
	if _, err := os.Stat(workspaceOf(cfg)); err == nil {
		t.Fatal("the workspace of a finished Run was left behind")
	}
}
