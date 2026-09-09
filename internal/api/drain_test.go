package api_test

import (
	"strings"
	"testing"
)

// drain (ADR-063 · CP3) — 소유자가 at-boundary 로 걸면 Mediator 가 단계 경계에서
// Run 을 닫고, graceful 이면 끝까지 두며, 풀면 기다리던 Run 이 그 노드로 간다.

func drainNote(run map[string]any) string {
	v, _ := run["verdict"].(map[string]any)
	checks, _ := v["checks"].([]any)
	if len(checks) == 0 {
		return ""
	}
	c, _ := checks[0].(map[string]any)
	note, _ := c["note"].(string)
	return note
}

// 단계 둘 계약이 도는 노드에 at-boundary 를 건다 — 첫 단계의 보고 뒤 Run 이 닫힌다.
func TestDrain_AtBoundaryClosesTheRunAfterTheStepReport(t *testing.T) {
	srv, _ := newServerFast(t)
	plain := advert("n1", "a", map[string]string{"role": "x"})
	do(t, srv, "POST", "/v1/nodes", plain, nil)
	if code, _ := do(t, srv, "POST", "/v1/runs", oneStepRun("d1", "n1"), nil); code != 201 {
		t.Fatalf("submit failed: %d", code)
	}
	// 소유자가 파일에 썼다 → 다음 광고가 나른다
	_, hb := do(t, srv, "POST", "/v1/nodes", advertWithDrain(t, "n1", "a", map[string]string{"role": "x"}, "at-boundary"), nil)
	if hb["drain"] != "at-boundary" {
		t.Fatalf("the advert response did not acknowledge the policy: %v", hb)
	}
	_, nodes := do(t, srv, "GET", "/v1/nodes", "", nil)
	if list, _ := nodes["nodes"].([]any); len(list) != 1 || list[0].(map[string]any)["draining"] != "at-boundary" {
		t.Fatalf("GET /v1/nodes does not show draining: %v", nodes)
	}

	// 도는 단계는 끝까지 간다 — 집고 · 돌고 · 보고한다
	if code, cl := do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil); code != 200 || cl["name"] != "s1" {
		t.Fatalf("the running step did not come: %d %v", code, cl)
	}
	code, res := do(t, srv, "POST", "/v1/runs/d1/steps/1/result", `{"node":"n1","exit_code":0,"produced":["s1"]}`, nil)
	if code != 200 || res["run_state"] != "FAILED" {
		t.Fatalf("the boundary did not close the run: %d %v", code, res)
	}

	_, run := do(t, srv, "GET", "/v1/runs/d1", "", nil)
	if run["state"] != "FAILED" {
		t.Fatalf("state=%v, want FAILED", run["state"])
	}
	if note := drainNote(run); !strings.Contains(note, "drain:n1") {
		t.Fatalf("the verdict does not carry drain:<node_id>: %q", note)
	}
	// 끝난 단계는 DONE 으로 남고, 남은 단계는 닫혔다
	steps, _ := run["steps"].([]any)
	if len(steps) != 2 || steps[0].(map[string]any)["state"] != "DONE" || steps[1].(map[string]any)["state"] != "FAILED" {
		t.Fatalf("steps after the boundary: %v", steps)
	}
	// 임대가 풀렸다 — 다음 하트비트에 없다
	_, hb = do(t, srv, "POST", "/v1/nodes", advertWithDrain(t, "n1", "a", map[string]string{"role": "x"}, "at-boundary"), nil)
	if l, _ := hb["leases"].([]any); len(l) != 0 {
		t.Fatalf("a lease survived the boundary: %v", l)
	}
	// 봉인됐다 — 끝난 단계의 산출이 Record 에 남는다 (I4)
	if code, _ := do(t, srv, "GET", "/v1/runs/d1/record", "", nil); code != 200 {
		t.Fatalf("the record of a drained run is not sealed: %d", code)
	}
}

// graceful 은 도는 Run 을 끝까지 둔다 — 새 임대만 막는다.
func TestDrain_GracefulLetsTheRunFinish(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "a", map[string]string{"role": "x"}), nil)
	do(t, srv, "POST", "/v1/runs", oneStepRun("g1", "n1"), nil)
	do(t, srv, "POST", "/v1/nodes", advertWithDrain(t, "n1", "a", map[string]string{"role": "x"}, "graceful"), nil)

	do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)
	if _, res := do(t, srv, "POST", "/v1/runs/g1/steps/1/result", `{"node":"n1","exit_code":0,"produced":["s1"]}`, nil); res["run_state"] != "" {
		t.Fatalf("graceful closed the run at the boundary: %v", res)
	}
	if code, cl := do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil); code != 200 || cl["name"] != "s2" {
		t.Fatalf("the next step did not come under graceful: %d %v", code, cl)
	}
	if _, res := do(t, srv, "POST", "/v1/runs/g1/steps/2/result", `{"node":"n1","exit_code":0,"produced":["s2"]}`, nil); res["run_state"] != "SUCCEEDED" {
		t.Fatalf("the run did not finish under graceful: %v", res)
	}
	// 새 임대는 막힌다 — 기다린다
	if code, body := do(t, srv, "POST", "/v1/runs", oneStep("g2", "x"), nil); code != 202 || body["state"] != "QUEUED" {
		t.Fatalf("a new run went to a graceful-draining node: %d %v", code, body)
	}
}

// 마지막 단계의 보고면 정산이 먼저 닫는다 — 취소가 아니다.
func TestDrain_LastStepIsSettledNotCancelled(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "a", map[string]string{"role": "x"}), nil)
	do(t, srv, "POST", "/v1/runs", oneStep("l1", "x"), nil)
	do(t, srv, "POST", "/v1/nodes", advertWithDrain(t, "n1", "a", map[string]string{"role": "x"}, "at-boundary"), nil)

	do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)
	if _, res := do(t, srv, "POST", "/v1/runs/l1/steps/1/result", `{"node":"n1","exit_code":0,"produced":["s"]}`, nil); res["run_state"] != "SUCCEEDED" {
		t.Fatalf("the last step was not settled: %v", res)
	}
	if _, run := do(t, srv, "GET", "/v1/runs/l1", "", nil); drainNote(run) != "" {
		t.Fatalf("a settled run carries a cancel note: %q", drainNote(run))
	}
}

// CP3 의 끝 — 닫힌 뒤 새 계약은 기다리고, 소유자가 풀면 그 노드로 간다.
func TestDrain_ReleaseInAdvertPromotesTheWaiter(t *testing.T) {
	srv, _ := newServerFast(t)
	plain := advert("n1", "a", map[string]string{"role": "x"})
	do(t, srv, "POST", "/v1/nodes", plain, nil)
	do(t, srv, "POST", "/v1/runs", oneStepRun("r1", "n1"), nil)
	do(t, srv, "POST", "/v1/nodes", advertWithDrain(t, "n1", "a", map[string]string{"role": "x"}, "at-boundary"), nil)
	do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)
	do(t, srv, "POST", "/v1/runs/r1/steps/1/result", `{"node":"n1","exit_code":0,"produced":["s1"]}`, nil)

	// 닫혔고 노드는 비었지만 draining 이라 후보가 아니다
	if code, body := do(t, srv, "POST", "/v1/runs", oneStep("r2", "x"), nil); code != 202 || body["state"] != "QUEUED" {
		t.Fatalf("a new run went to a draining node: %d %v", code, body)
	}
	// 소유자가 푼다 — 정책 없는 광고
	do(t, srv, "POST", "/v1/nodes", plain, nil)
	if _, run := do(t, srv, "GET", "/v1/runs/r2", "", nil); run["state"] != "RUNNING" {
		t.Fatalf("the waiter did not take the released node: %v", run["state"])
	}
}

// 되돌림(재시도 루프) 뒤에도 경계는 경계다 — at-boundary 면 다시 돌리지 않고 닫는다.
func TestDrain_RolledBackStepStillClosesAtBoundary(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "a", map[string]string{"role": "x"}), nil)
	do(t, srv, "POST", "/v1/runs", loopRun("rd", 3), nil)

	do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)
	do(t, srv, "POST", "/v1/runs/rd/steps/1/result", `{"node":"n1","produced":["test_source"]}`, nil)
	// 검증 단계가 도는 사이 소유자가 at-boundary 를 건다
	do(t, srv, "POST", "/v1/nodes", advertWithDrain(t, "n1", "a", map[string]string{"role": "x"}, "at-boundary"), nil)
	do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)
	code, body := do(t, srv, "POST", "/v1/runs/rd/steps/2/result",
		`{"node":"n1","exit_code":2,"produced":["build_log"]}`, nil)
	if code != 200 || body["rolled_back"] != true {
		t.Fatalf("the failed check did not roll back: %d %v", code, body)
	}
	if body["run_state"] != "FAILED" {
		t.Fatalf("the boundary after a rollback did not close the run: %v", body)
	}
	_, run := do(t, srv, "GET", "/v1/runs/rd", "", nil)
	if run["state"] != "FAILED" || !strings.Contains(drainNote(run), "drain:n1") {
		t.Fatalf("rolled-back run after the boundary: %v %q", run["state"], drainNote(run))
	}
	// 되돌린 단계가 다시 나오지 않는다
	if code, _ := do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil); code != 204 {
		t.Fatalf("a step of a drained run was handed out again: %d", code)
	}
}
