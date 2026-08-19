package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/taeels/enode/internal/api"
	"github.com/taeels/enode/internal/config"
	"github.com/taeels/enode/internal/store"
)

const token = "test-token"

// 진짜 Postgres 를 쓴다. I5 는 ★ 트랜잭션으로 얻는 것 ★ 이라
// 가짜 저장소로는 검증되지 않는다.
func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	url := os.Getenv("ENODE_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("ENODE_TEST_DATABASE_URL 이 없다 — scripts/testdb.sh 참고")
	}
	ctx := context.Background()
	st, err := store.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	// 테스트마다 깨끗한 상태에서 시작한다.
	if err := st.Truncate(ctx); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Token = token
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := httptest.NewServer(api.New(st, cfg, log).Handler())
	t.Cleanup(srv.Close)
	return srv
}

func do(t *testing.T, srv *httptest.Server, method, path, body string, hdr map[string]string) (int, map[string]any) {
	t.Helper()
	req, err := http.NewRequest(method, srv.URL+path, bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Enode-Principal", "taeels@gmail.com")
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func advert(id, label string, attrs map[string]string) string {
	a, _ := json.Marshal(map[string]any{
		"node_id": id, "label": label,
		"capabilities": []map[string]any{{"capability": "agent.reason", "attrs": attrs}},
	})
	return string(a)
}

// 계약 하나. requires 의 속성은 형제 키로 들어간다 (run-contract §1).
func contractJSON(runID string, reqs []map[string]any, steps []map[string]any) string {
	b, _ := json.Marshal(map[string]any{
		"run_id": runID, "requires": reqs, "steps": steps,
	})
	return string(b)
}

func req(as string, attrs map[string]any) map[string]any {
	m := map[string]any{"as": as, "capability": "agent.reason"}
	for k, v := range attrs {
		m[k] = v
	}
	return m
}

func runStep(id, uses string) map[string]any {
	return map[string]any{"id": id, "uses": uses, "run": []string{"true"}, "out": []string{id}}
}

// ── 인증 (ADR-015 §1) ────────────────────────────────────────────────────

func TestAuthRequired(t *testing.T) {
	srv := newServer(t)
	req, _ := http.NewRequest("POST", srv.URL+"/v1/runs", bytes.NewBufferString("{}"))
	resp, err := http.DefaultClient.Do(req) // Authorization 없음
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Fatalf("code=%d 기대 401", resp.StatusCode)
	}
}

// ── 400 — 계약이 문법적으로 틀렸다 ────────────────────────────────────────

func TestSubmitRejectsBadContract(t *testing.T) {
	srv := newServer(t)
	cases := map[string]string{
		"run_id 없음": contractJSON("", []map[string]any{req("b", nil)}, []map[string]any{runStep("s", "b")}),
		// ★ ADR-019 — 어휘는 agent.reason 하나뿐이다 ★
		"옛 capability": `{"run_id":"r1","requires":[{"as":"b","capability":"build.linux"}],
		                   "steps":[{"id":"s","uses":"b","run":["true"]}]}`,
		// ★ ADR-019 결정 3 — 단계는 두 종류. 암묵을 안 남긴다 ★
		"agent 와 run 이 둘 다": `{"run_id":"r2","requires":[{"as":"b","capability":"agent.reason"}],
		                        "steps":[{"id":"s","uses":"b","run":["true"],"agent":{}}]}`,
		// ★ ADR-004 를 지키는 한 줄 ★
		"agent 단계에 exit_code": `{"run_id":"r3","requires":[{"as":"b","capability":"agent.reason"}],
		                          "steps":[{"id":"s","uses":"b","agent":{}}],
		                          "success_when":[{"step":"s","exit_code":0}]}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			code, _ := do(t, srv, "POST", "/v1/runs", body, nil)
			if code != 400 {
				t.Fatalf("code=%d 기대 400", code)
			}
		})
	}
}

// ── 422 영구 / 409 일시 (ADR-014 결정 3) ─────────────────────────────────

func TestSubmitRejectCodes(t *testing.T) {
	srv := newServer(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "mac", map[string]string{"harness": "claude"}), nil)

	// 함대에 없다 → 영구. 다시 제출해도 영원히 같다.
	code, _ := do(t, srv, "POST", "/v1/runs",
		contractJSON("perm", []map[string]any{req("b", map[string]any{"board": "SoC-X"})},
			[]map[string]any{runStep("s", "b")}), nil)
	if code != 422 {
		t.Fatalf("code=%d 기대 422", code)
	}

	// 첫 Run 이 n1 을 잡는다
	code, _ = do(t, srv, "POST", "/v1/runs",
		contractJSON("first", []map[string]any{req("b", map[string]any{"harness": "claude"})},
			[]map[string]any{runStep("s", "b")}), nil)
	if code != 201 {
		t.Fatalf("첫 Run code=%d 기대 201", code)
	}

	// ★ O7 · I1 ★ 같은 자원을 요구하는 두 번째 Run 은 거절된다 — 일시적이므로 409
	code, body := do(t, srv, "POST", "/v1/runs",
		contractJSON("second", []map[string]any{req("b", map[string]any{"harness": "claude"})},
			[]map[string]any{runStep("s", "b")}), nil)
	if code != 409 {
		t.Fatalf("code=%d 기대 409 — I1 이 안 지켜졌다 (%v)", code, body)
	}
}

// ── 멱등 (INVARIANTS §2 첫 행) ───────────────────────────────────────────

func TestSubmitIdempotent(t *testing.T) {
	srv := newServer(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "mac", map[string]string{"harness": "claude"}), nil)
	c := contractJSON("gerrit-12345-ps3", []map[string]any{req("b", map[string]any{"harness": "claude"})},
		[]map[string]any{runStep("s", "b")})

	code, first := do(t, srv, "POST", "/v1/runs", c, nil)
	if code != 201 {
		t.Fatalf("code=%d 기대 201", code)
	}
	// ★ 재제출은 200 + 기존 Run ★ — 폴링 커서가 필요 없는 이유가 이것이다
	code, again := do(t, srv, "POST", "/v1/runs", c, nil)
	if code != 200 {
		t.Fatalf("재제출 code=%d 기대 200", code)
	}
	if first["run_id"] != again["run_id"] || again["state"] != first["state"] {
		t.Fatalf("다른 Run 이 생겼다: %v vs %v", first, again)
	}
}

// ── ★ I5 — 동시 제출이 부분 점유를 안 남긴다 ★ ────────────────────────────
//
// 노드 둘을 요구하는 Run 을 여럿 동시에 던진다. 하나만 이겨야 하고,
// 진 쪽은 ★ 잡았던 노드를 남기면 안 된다 ★.
func TestConcurrentSubmitAllOrNothing(t *testing.T) {
	srv := newServer(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "a", map[string]string{"role": "x"}), nil)
	do(t, srv, "POST", "/v1/nodes", advert("n2", "b", map[string]string{"role": "y"}), nil)

	const n = 6
	codes := make([]int, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c := contractJSON(fmt.Sprintf("race-%d", i),
				[]map[string]any{
					req("x", map[string]any{"role": "x"}),
					req("y", map[string]any{"role": "y"}),
				},
				[]map[string]any{runStep("s1", "x"), runStep("s2", "y")})
			codes[i], _ = do(t, srv, "POST", "/v1/runs", c, nil)
		}(i)
	}
	wg.Wait()

	won := 0
	for i, c := range codes {
		switch c {
		case 201:
			won++
		case 409:
		default:
			t.Fatalf("race-%d code=%d — 201 이나 409 여야 한다", i, c)
		}
	}
	if won != 1 {
		t.Fatalf("★ I1 위반 ★ %d 개가 이겼다. 하나여야 한다.", won)
	}

	// ★ 진 쪽이 부분 점유를 남겼는지 본다 ★
	// 남았다면 세 번째 자원 요구가 409 로 막힌다.
	code, body := do(t, srv, "POST", "/v1/runs",
		contractJSON("after", []map[string]any{req("x", map[string]any{"role": "x"})},
			[]map[string]any{runStep("s", "x")}), nil)
	if code != 409 {
		t.Fatalf("이긴 Run 이 n1 을 쥐고 있어야 한다: code=%d (%v)", code, body)
	}
}

// ── dry-run — 같은 매처, 점유는 안 본다 (ADR-014 결정 3) ──────────────────

func TestDryRunDoesNotAllocate(t *testing.T) {
	srv := newServer(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "mac", map[string]string{"harness": "claude"}), nil)
	c := contractJSON("dry", []map[string]any{req("b", map[string]any{"harness": "claude"})},
		[]map[string]any{runStep("s", "b")})

	code, _ := do(t, srv, "POST", "/v1/runs/dry-run", c, nil)
	if code != 200 {
		t.Fatalf("dry-run code=%d 기대 200", code)
	}
	// 점유하지 않았으므로 진짜 제출이 통과해야 한다
	if code, _ := do(t, srv, "POST", "/v1/runs", c, nil); code != 201 {
		t.Fatalf("★ dry-run 이 점유했다 ★ code=%d", code)
	}
	// ★ 이미 점유된 상태에서도 dry-run 은 409 를 내지 않는다 ★
	// 존재는 답하고 여유는 답하지 않는다.
	code, _ = do(t, srv, "POST", "/v1/runs/dry-run",
		contractJSON("dry2", []map[string]any{req("b", map[string]any{"harness": "claude"})},
			[]map[string]any{runStep("s", "b")}), nil)
	if code != 200 {
		t.Fatalf("dry-run 이 점유를 봤다: code=%d", code)
	}
	// 반면 영구 거절(422)은 dry-run 도 그대로 낸다 — 계약을 고쳐야 하기 때문이다
	code, _ = do(t, srv, "POST", "/v1/runs/dry-run",
		contractJSON("dry3", []map[string]any{req("b", map[string]any{"board": "SoC-Z"})},
			[]map[string]any{runStep("s", "b")}), nil)
	if code != 422 {
		t.Fatalf("dry-run 이 422 를 안 냈다: code=%d", code)
	}
}

// ── 광고는 통째로 교체된다 (ADR-012 · ADR-017 결정 3) ────────────────────
//
// capability 를 빼고 보내는 것이 곧 "지금은 못 한다" 이므로,
// 병합하면 그 뜻이 사라진다.
func TestAdvertReplacesNotMerges(t *testing.T) {
	srv := newServer(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "box", map[string]string{"arch": "armv7", "board": "SoC-X"}), nil)
	if code, _ := do(t, srv, "POST", "/v1/runs",
		contractJSON("b1", []map[string]any{req("b", map[string]any{"board": "SoC-X"})},
			[]map[string]any{runStep("s", "b")}), nil); code != 201 {
		t.Fatalf("처음엔 보드가 있어야 한다: %d", code)
	}
	// 보드가 빠졌다고 다시 광고한다
	do(t, srv, "POST", "/v1/nodes", advert("n1", "box", map[string]string{"arch": "armv7"}), nil)
	code, _ := do(t, srv, "POST", "/v1/runs",
		contractJSON("b2", []map[string]any{req("b", map[string]any{"board": "SoC-X"})},
			[]map[string]any{runStep("s", "b")}), nil)
	if code != 422 {
		t.Fatalf("★ 광고가 병합됐다 ★ code=%d 기대 422", code)
	}
}

func TestGetRunNotFound(t *testing.T) {
	srv := newServer(t)
	if code, _ := do(t, srv, "GET", "/v1/runs/nope", "", nil); code != 404 {
		t.Fatalf("code=%d 기대 404", code)
	}
}
