package api_test

import (
	"encoding/json"
	"testing"
)

// 대기열 (ADR-064 · CP2) — 점유 실패가 202 · QUEUED 로 받아지고, 임대가
// 풀리는 지점에서 같은 요청 안에 승격된다.

func oneStep(runID, role string) string {
	return contractJSON(runID, []map[string]any{req("b", map[string]any{"role": role})},
		[]map[string]any{runStep("s", "b")})
}

// advertWithDrain 은 광고에 정책을 얹는다 — advert() 의 본문을 그대로 두고 키 하나만 더한다.
func advertWithDrain(t *testing.T, id, label string, attrs map[string]string, drain string) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(advert(id, label, attrs)), &m); err != nil {
		t.Fatal(err)
	}
	m["policy"] = map[string]any{"drain": drain}
	b, _ := json.Marshal(m)
	return string(b)
}

// CP2 첫 줄 — 노드 하나에 Run 둘을 던지면 둘째가 202 · QUEUED 다.
func TestSubmit_SecondRunOnABusyNodeIs202Queued(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "a", map[string]string{"role": "x"}), nil)

	if code, _ := do(t, srv, "POST", "/v1/runs", oneStep("a", "x"), nil); code != 201 {
		t.Fatalf("first submit code=%d, want 201", code)
	}
	code, body := do(t, srv, "POST", "/v1/runs", oneStep("b", "x"), nil)
	if code != 202 {
		t.Fatalf("second submit code=%d, want 202 (%v)", code, body)
	}
	if body["state"] != "QUEUED" {
		t.Fatalf("state=%v, want QUEUED", body["state"])
	}
	if _, ok := body["assigned"]; ok {
		t.Fatalf("a queued run carries an assignment: %v", body["assigned"])
	}
	if _, ok := body["reject"]; ok {
		t.Fatalf("a queued run is not a rejection: %v", body["reject"])
	}
}

// 대기 Run 은 상세 조회에 QUEUED 로 뜨고, 무엇을 기다리는지(requires · obs)를 보인다.
func TestSubmit_QueuedRunAppearsInGetRunWithRequires(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "a", map[string]string{"role": "x"}), nil)
	do(t, srv, "POST", "/v1/runs", oneStep("a", "x"), nil)
	do(t, srv, "POST", "/v1/runs", oneStep("b", "x"), nil)

	code, run := do(t, srv, "GET", "/v1/runs/b", "", nil)
	if code != 200 || run["state"] != "QUEUED" {
		t.Fatalf("GET queued run: %d %v", code, run)
	}
	reqs, _ := run["requires"].([]any)
	if len(reqs) != 1 {
		t.Fatalf("requires is missing on a queued run: %v", run)
	}
	if steps, _ := run["steps"].([]any); len(steps) != 0 {
		t.Fatalf("a queued run has steps before promotion: %v", steps)
	}
	// 재제출은 200 + 기존 Run — QUEUED 여도 같다 (INVARIANTS §2 첫 행)
	if code, again := do(t, srv, "POST", "/v1/runs", oneStep("b", "x"), nil); code != 200 || again["state"] != "QUEUED" {
		t.Fatalf("resubmit of a queued run: %d %v, want 200 QUEUED", code, again)
	}
}

// CP2 셋째 줄 — 첫째가 끝나면 둘째가 RUNNING 이다. 결과 보고 요청이 돌아온 뒤 바로다 (동기).
func TestSubmit_WhenTheFirstRunEndsTheQueuedOneIsRunning(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "a", map[string]string{"role": "x"}), nil)
	do(t, srv, "POST", "/v1/runs", oneStep("a", "x"), nil)
	do(t, srv, "POST", "/v1/runs", oneStep("b", "x"), nil)

	if code, cl := do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil); code != 200 || cl["run_id"] != "a" {
		t.Fatalf("a's step did not come: %d %v", code, cl)
	}
	if code, _ := do(t, srv, "POST", "/v1/runs/a/steps/1/result",
		`{"node":"n1","exit_code":0,"produced":["s"]}`, nil); code != 200 {
		t.Fatalf("report failed: %d", code)
	}
	_, run := do(t, srv, "GET", "/v1/runs/b", "", nil)
	if run["state"] != "RUNNING" {
		t.Fatalf("b state=%v right after a ended, want RUNNING", run["state"])
	}
	if code, cl := do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil); code != 200 || cl["run_id"] != "b" {
		t.Fatalf("b's step did not come after promotion: %d %v", code, cl)
	}
}

// 취소도 임대를 풀고, 같은 요청 안에서 대기 Run 을 깨운다.
func TestSubmit_CancelReleasesAndWakes(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "a", map[string]string{"role": "x"}), nil)
	do(t, srv, "POST", "/v1/runs", oneStep("a", "x"), nil)
	do(t, srv, "POST", "/v1/runs", oneStep("b", "x"), nil)

	if code, _ := do(t, srv, "POST", "/v1/runs/a/cancel", "", nil); code != 200 {
		t.Fatalf("cancel failed: %d", code)
	}
	_, run := do(t, srv, "GET", "/v1/runs/b", "", nil)
	if run["state"] != "RUNNING" {
		t.Fatalf("b state=%v after a was cancelled, want RUNNING", run["state"])
	}
	// 대기 Run 자체의 취소도 닫힌다 — 출구는 이것뿐이다
	do(t, srv, "POST", "/v1/runs", oneStep("c", "x"), nil)
	if code, body := do(t, srv, "POST", "/v1/runs/c/cancel", "", nil); code != 200 || body["state"] != "FAILED" {
		t.Fatalf("cancel of a queued run: %d %v, want 200 FAILED", code, body)
	}
}

// dry-run 은 busy 도 draining 도 안 본다 — 202 가 나올 수 없다 (ADR-014 결정 3 · ADR-063 §6).
func TestSubmit_DryRunIgnoresBusyAndDraining(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "a", map[string]string{"role": "x"}), nil)
	do(t, srv, "POST", "/v1/nodes", advertWithDrain(t, "n2", "b", map[string]string{"role": "y"}, "graceful"), nil)
	do(t, srv, "POST", "/v1/runs", oneStep("a", "x"), nil)

	if code, body := do(t, srv, "POST", "/v1/runs/dry-run", oneStep("dry-x", "x"), nil); code != 200 || body["state"] != "DRY_RUN" {
		t.Fatalf("dry-run against a busy node: %d %v, want 200 DRY_RUN", code, body)
	}
	if code, body := do(t, srv, "POST", "/v1/runs/dry-run", oneStep("dry-y", "y"), nil); code != 200 || body["state"] != "DRY_RUN" {
		t.Fatalf("dry-run against a draining node: %d %v, want 200 DRY_RUN", code, body)
	}
	// 진짜 제출은 draining 을 본다 — 기다린다
	if code, body := do(t, srv, "POST", "/v1/runs", oneStep("real-y", "y"), nil); code != 202 || body["state"] != "QUEUED" {
		t.Fatalf("submit against a draining node: %d %v, want 202 QUEUED", code, body)
	}
}

// 함대에 없는 능력은 여전히 422 · FAILED 다 — 기다려도 안 된다.
func TestSubmit_UnmatchableIsStill422Failed(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "a", map[string]string{"role": "x"}), nil)

	if code, _ := do(t, srv, "POST", "/v1/runs", oneStep("nope", "z"), nil); code != 422 {
		t.Fatalf("code=%d, want 422", code)
	}
	if _, run := do(t, srv, "GET", "/v1/runs/nope", "", nil); run["state"] != "FAILED" {
		t.Fatalf("state=%v, want FAILED", run["state"])
	}
}

// drain 해제를 나르는 광고가 대기 Run 을 깨운다.
func TestSubmit_DrainReleaseInAdvertWakes(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advertWithDrain(t, "n1", "a", map[string]string{"role": "x"}, "at-boundary"), nil)

	if code, body := do(t, srv, "POST", "/v1/runs", oneStep("w", "x"), nil); code != 202 || body["state"] != "QUEUED" {
		t.Fatalf("submit against a drained node: %d %v, want 202 QUEUED", code, body)
	}
	_, resp := do(t, srv, "POST", "/v1/nodes", advert("n1", "a", map[string]string{"role": "x"}), nil)
	if resp["drain"] != "" {
		t.Fatalf("the release was not acknowledged: %v", resp["drain"])
	}
	_, run := do(t, srv, "GET", "/v1/runs/w", "", nil)
	if run["state"] != "RUNNING" {
		t.Fatalf("w state=%v after the drain release, want RUNNING", run["state"])
	}
}
