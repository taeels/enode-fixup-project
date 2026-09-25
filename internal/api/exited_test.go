package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/taeels/enode/internal/contract"
)

// 종료 보고 라우트와 진행 구간 (ADR-075 결정 7 · step-phase 유닛).
// HTTP 로 400 · 404 · 409 · 200 을 보고, 진행 조회 · 대기 사유 · 봉인에 새 칸이
// 어떻게 나오는지 본다. 저장소 쪽의 수락 표 전체는 internal/store 의 시험이 본다.

const exitedAt = "2026-09-25T09:00:00Z"

func exitedBody(node, inst string, attempt int) string {
	b, _ := json.Marshal(map[string]any{"node": node, "instance": inst, "attempt": attempt,
		"outcome": map[string]any{"kind": "exit", "code": 0}, "exited_at": exitedAt})
	return string(b)
}

func claimAs(t *testing.T, srv *httptest.Server, node, inst string) {
	t.Helper()
	req, err := http.NewRequest("POST", srv.URL+"/v1/nodes/"+node+"/claim", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Enode-Instance", inst)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("claim by %s = %d", node, resp.StatusCode)
	}
}

func reasonIn(body map[string]any) string {
	e, _ := body["error"].(map[string]any)
	r, _ := e["reason"].(string)
	return r
}

func TestExited_TheRouteAndItsAnswers(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("E1", "a", map[string]string{"role": "x"}), nil)
	if code, body := do(t, srv, "POST", "/v1/runs", oneStepRun("ex1", "E1"), nil); code != 201 {
		t.Fatalf("submit = %d %v", code, body)
	}
	claimAs(t, srv, "E1", "life-1")

	path := "/v1/runs/ex1/steps/1/exited"
	cases := []struct {
		name, path, body string
		code             int
		reason           string // 앞부분
	}{
		{"seq zero", "/v1/runs/ex1/steps/0/exited", exitedBody("E1", "life-1", 0), 400, "invalid step sequence: 0"},
		{"seq not a number", "/v1/runs/ex1/steps/x/exited", exitedBody("E1", "life-1", 0), 400, "invalid step sequence: x"},
		{"not JSON", path, "not json", 400, "cannot parse exited: "},
		{"exited_at is not RFC 3339", path,
			`{"node":"E1","instance":"life-1","outcome":{"kind":"exit","code":0},"exited_at":"yesterday"}`,
			400, "cannot parse exited: "},
		{"no node", path, exitedBody("", "life-1", 0), 400, "exited: node is empty"},
		{"unknown outcome", path,
			`{"node":"E1","instance":"life-1","outcome":{"kind":"crash"},"exited_at":"` + exitedAt + `"}`,
			400, `exited: outcome.kind "crash" is not exit, signal or timeout`},
		{"no such run", "/v1/runs/nope/steps/1/exited", exitedBody("E1", "life-1", 0), 404, "no such run"},
		{"no such step", "/v1/runs/ex1/steps/9/exited", exitedBody("E1", "life-1", 0), 404, "no such step"},
		{"a restarted node", path, exitedBody("E1", "life-2", 0), 409,
			"step ex1#01 is claimed by another instance of this node; " +
				"a restarted node cannot report the exit of an earlier life"},
		{"another node", path, exitedBody("E9", "life-1", 0), 409, "step ex1#01 is claimed by another node"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, body := do(t, srv, "POST", tc.path, tc.body, nil)
			if code != tc.code || !strings.HasPrefix(reasonIn(body), tc.reason) {
				t.Fatalf("%d %q, want %d %q", code, reasonIn(body), tc.code, tc.reason)
			}
		})
	}

	// 받는다 — 그리고 같은 보고의 재전송은 200 이지만 받지 않는다.
	for _, want := range []bool{true, false} {
		code, body := do(t, srv, "POST", path, exitedBody("E1", "life-1", 0), nil)
		if code != 200 || body["accepted"] != want || body["run_id"] != "ex1" || body["seq"] != float64(1) {
			t.Fatalf("report = %d %v, want 200 accepted %v", code, body, want)
		}
	}

	// 진행 조회 — CLAIMED 단계는 phase 와 노드 시계의 phase_since, 그리고 exit.
	// 아직 안 집힌 단계는 셋 다 없다.
	_, run := do(t, srv, "GET", "/v1/runs/ex1", "", nil)
	steps, _ := run["steps"].([]any)
	s1, _ := steps[0].(map[string]any)
	s2, _ := steps[1].(map[string]any)
	// phase_since 는 노드가 보낸 시각 그대로다. 표기의 시간대는 DB 연결의 것이라
	// 글자가 아니라 시각으로 비교한다.
	since, _ := time.Parse(time.RFC3339Nano, fmt.Sprint(s1["phase_since"]))
	want, _ := time.Parse(time.RFC3339, exitedAt)
	if s1["phase"] != "finalizing" || !since.Equal(want) {
		t.Fatalf("step 1 = %v", s1)
	}
	if e, _ := s1["exit"].(map[string]any); e["kind"] != "exit" || e["code"] != float64(0) {
		t.Fatalf("step 1 exit = %v", s1["exit"])
	}
	for _, k := range []string{"phase", "phase_since", "exit"} {
		if _, ok := s2[k]; ok {
			t.Fatalf("a pending step carries %s: %v", k, s2)
		}
	}

	// 결과 뒤 — 끝난 단계는 phase 를 안 싣고 exit 는 남는다.
	do(t, srv, "POST", "/v1/runs/ex1/steps/1/result", `{"node":"E1","exit_code":0}`, nil)
	_, run = do(t, srv, "GET", "/v1/runs/ex1", "", nil)
	steps, _ = run["steps"].([]any)
	s1, _ = steps[0].(map[string]any)
	if _, ok := s1["phase"]; ok || s1["state"] != "DONE" || s1["exit"] == nil {
		t.Fatalf("a finished step = %v", s1)
	}
	// 끝난 단계에 늦게 온 보고도 200 이고 받지 않는다.
	if code, body := do(t, srv, "POST", path, exitedBody("E1", "life-1", 0), nil); code != 200 || body["accepted"] != false {
		t.Fatalf("a report after the result = %d %v", code, body)
	}

	// 라우트가 POST 로 등록돼 있다 — 다른 메서드는 405 다.
	if code, _ := do(t, srv, "GET", path, "", nil); code != http.StatusMethodNotAllowed {
		t.Fatalf("GET %s = %d, want 405", path, code)
	}
	// 인증은 result 와 같다.
	resp, err := http.Post(srv.URL+path, "application/json", bytes.NewBufferString(exitedBody("E1", "life-1", 0)))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Fatalf("an unauthenticated report = %d, want 401", resp.StatusCode)
	}
}

// QUEUED 인 Run 의 상세 조회는 요구 줄마다 후보 수 셋과 셈의 시각을 싣는다 (US-7).
// 점유된 노드와 drain 중인 노드가 따로 보인다. RUNNING 과 제출 응답에는 없다.
func TestGetRun_AQueuedRunSaysWhyItWaits(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("Q1", "a", map[string]string{"role": "x"}), nil)
	do(t, srv, "POST", "/v1/nodes", advertWithDrain(t, "Q2", "b", map[string]string{"role": "x"}, "graceful"), nil)
	if code, _ := do(t, srv, "POST", "/v1/runs", oneStep("qa", "x"), nil); code != 201 {
		t.Fatalf("the first run was not placed: %d", code)
	}
	code, submitted := do(t, srv, "POST", "/v1/runs", oneStep("qb", "x"), nil)
	if code != 202 {
		t.Fatalf("the second run = %d %v, want 202", code, submitted)
	}
	if _, ok := submitted["candidates_at"]; ok {
		t.Fatalf("a submit response carries candidates_at: %v", submitted)
	}

	_, run := do(t, srv, "GET", "/v1/runs/qb", "", nil)
	reqs, _ := run["requires"].([]any)
	r0, _ := reqs[0].(map[string]any)
	c, _ := r0["candidates"].(map[string]any)
	if c["live"] != float64(2) || c["busy"] != float64(1) || c["draining"] != float64(1) {
		t.Fatalf("candidates = %v, want live 2 busy 1 draining 1", r0["candidates"])
	}
	if _, ok := c["free"]; ok {
		t.Fatalf("the remainder is not a field (ADR-065): %v", c)
	}
	if at, _ := run["candidates_at"].(string); at == "" {
		t.Fatalf("candidates_at is missing: %v", run)
	}

	_, running := do(t, srv, "GET", "/v1/runs/qa", "", nil)
	if _, ok := running["candidates_at"]; ok {
		t.Fatalf("a running run carries candidates_at: %v", running)
	}
	reqs, _ = running["requires"].([]any)
	if r, _ := reqs[0].(map[string]any); r["candidates"] != nil {
		t.Fatalf("a running run carries candidates: %v", r)
	}
}

// result 의 어휘 밖 값은 200 으로 받고 봉인에 그대로 남는다. 종료 보고를 받은 단계는
// Record 에 exited_at · exit · last_phase 를, 안 받은 단계는 last_phase running 만 남긴다.
func TestResult_TheNewFieldsReachTheSeal(t *testing.T) {
	srv, st := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("R1", "a", map[string]string{"role": "x"}), nil)
	c := contractJSON("rs1", []map[string]any{req("b", map[string]any{"role": "x"})},
		[]map[string]any{
			{"id": "s1", "uses": "b", "run": []string{"true"}},
			{"id": "s2", "uses": "b", "run": []string{"true"}},
		})
	if code, _ := do(t, srv, "POST", "/v1/runs", c, nil); code != 201 {
		t.Fatalf("submit = %d", code)
	}
	claimAs(t, srv, "R1", "life-1")
	if code, body := do(t, srv, "POST", "/v1/runs/rs1/steps/1/exited", exitedBody("R1", "life-1", 0), nil); code != 200 {
		t.Fatalf("exited = %d %v", code, body)
	}
	res := `{"node":"R1","exit_code":0,"finalize":"sideways","reason":"new_reason",
	  "finalized_at":"2026-09-25T09:00:30Z","diagnostics":{"changes":"maybe","effect":"build"}}`
	if code, body := do(t, srv, "POST", "/v1/runs/rs1/steps/1/result", res, nil); code != 200 {
		t.Fatalf("a result with unknown values = %d %v, want 200", code, body)
	}
	claimAs(t, srv, "R1", "life-1")
	if code, _ := do(t, srv, "POST", "/v1/runs/rs1/steps/2/result", `{"node":"R1","exit_code":0}`, nil); code != 200 {
		t.Fatalf("result 2 = %d", code)
	}

	var s1, s2 struct {
		ExitedAt    string          `json:"exited_at"`
		FinalizedAt string          `json:"finalized_at"`
		Exit        json.RawMessage `json:"exit"`
		LastPhase   string          `json:"last_phase"`
		Result      map[string]any  `json:"result"`
	}
	for _, f := range []struct {
		name string
		into any
	}{{"01-s1.json", &s1}, {"02-s2.json", &s2}} {
		raw, err := os.ReadFile(filepath.Join(st.Records.Root, "run-rs1", "steps", f.name))
		if err != nil {
			t.Fatalf("nothing was sealed: %v", err)
		}
		if err := json.Unmarshal(raw, f.into); err != nil {
			t.Fatal(err)
		}
	}
	if s1.Result["finalize"] != "sideways" || s1.Result["reason"] != "new_reason" {
		t.Fatalf("the sealed result lost the unknown values: %v", s1.Result)
	}
	if d, _ := s1.Result["diagnostics"].(map[string]any); d["changes"] != "maybe" {
		t.Fatalf("the sealed diagnostics = %v", s1.Result["diagnostics"])
	}
	var o contract.Outcome
	if err := json.Unmarshal(s1.Exit, &o); err != nil || o.Kind != "exit" {
		t.Fatalf("s1 exit = %s", s1.Exit)
	}
	if s1.ExitedAt != exitedAt || s1.FinalizedAt != "2026-09-25T09:00:30Z" || s1.LastPhase != "finalizing" {
		t.Fatalf("s1 = exited_at %q finalized_at %q last_phase %q", s1.ExitedAt, s1.FinalizedAt, s1.LastPhase)
	}
	if s2.LastPhase != "running" || s2.Exit != nil || s2.ExitedAt != "" {
		t.Fatalf("a step without an exit report = %+v", s2)
	}
}

// claim 응답이 계약의 새 칸을 JSON 이름 그대로 싣는다 — 제품의 굽기 예시로 본다.
func TestClaim_CarriesTheBakeFields(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("B1", "a",
		map[string]string{"workspace.writes": "isolated"}), nil)
	raw, err := contract.Example("bake")
	if err != nil {
		t.Fatal(err)
	}
	if code, body := do(t, srv, "POST", "/v1/runs", string(raw), nil); code != 201 {
		t.Fatalf("submit the bake example = %d %v", code, body)
	}
	code, claim := do(t, srv, "POST", "/v1/nodes/B1/claim", "", nil)
	if code != 200 || claim["kind"] != "build" || claim["effect"] != "prepare" || claim["ir"] != "your-ir-tag" {
		t.Fatalf("claim = %d %v", code, claim)
	}
	if builds, _ := claim["builds"].([]any); len(builds) != 2 {
		t.Fatalf("builds = %v", claim["builds"])
	}
	if s, _ := claim["sync"].(string); s == "" {
		t.Fatalf("sync = %v", claim["sync"])
	}
}
