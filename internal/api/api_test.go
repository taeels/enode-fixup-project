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
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/taeels/enode/internal/api"
	"github.com/taeels/enode/internal/config"
	"github.com/taeels/enode/internal/record"
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
	st.Records = newRecords(t)
	cfg := config.Default()
	cfg.Token = token
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := httptest.NewServer(api.New(st, cfg, log).Handler())
	t.Cleanup(srv.Close)
	return srv
}

// newRecords 는 봉인된 디렉터리를 t.TempDir 이 지울 수 있게 정리를 걸어둔다.
// ★ 봉인은 삭제까지 막는다 ★ — 그 자체가 I4 가 작동한다는 증거다.
func newRecords(t *testing.T) *record.Store {
	t.Helper()
	root := t.TempDir()
	t.Cleanup(func() {
		_ = filepath.Walk(root, func(p string, fi os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if fi.IsDir() {
				return os.Chmod(p, 0o755)
			}
			return os.Chmod(p, 0o644)
		})
	})
	return record.New(root)
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

// ═══ S4 — 하트비트가 임대를 나른다 · claim · 회수 ════════════════════════

func newServerFast(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	url := os.Getenv("ENODE_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("ENODE_TEST_DATABASE_URL 이 없다")
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
	if err := st.Truncate(ctx); err != nil {
		t.Fatal(err)
	}
	st.Records = newRecords(t)
	cfg := config.Default()
	cfg.Token = token
	cfg.Claim.LongPollSeconds = 0 // 테스트에서는 즉시 204
	cfg.Lease.RenewSeconds, cfg.Lease.NotAfterFactor = 1, 2
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := httptest.NewServer(api.New(st, cfg, log).Handler())
	t.Cleanup(srv.Close)
	return srv, st
}

func oneStepRun(id, node string) string {
	return contractJSON(id, []map[string]any{req("b", map[string]any{"role": "x"})},
		[]map[string]any{runStep("s1", "b"), runStep("s2", "b")})
}

// ★ ADR-016 — 하트비트 응답이 임대의 갱신이자 취소 통보다 ★
func TestHeartbeatCarriesLease(t *testing.T) {
	srv, _ := newServerFast(t)
	adv := advert("n1", "box", map[string]string{"role": "x"})

	_, body := do(t, srv, "POST", "/v1/nodes", adv, nil)
	if l, _ := body["leases"].([]any); len(l) != 0 {
		t.Fatalf("아직 아무것도 안 잡았는데 임대가 있다: %v", body)
	}

	if code, _ := do(t, srv, "POST", "/v1/runs", oneStepRun("hb", "n1"), nil); code != 201 {
		t.Fatalf("제출 실패: %d", code)
	}

	_, body = do(t, srv, "POST", "/v1/nodes", adv, nil)
	leases, _ := body["leases"].([]any)
	if len(leases) != 1 {
		t.Fatalf("★ 임대가 하트비트 응답에 안 실렸다 ★: %v", body)
	}
	first := leases[0].(map[string]any)["not_after"].(string)

	// 갱신 — not_after 가 앞으로 밀려야 한다
	time.Sleep(1100 * time.Millisecond)
	_, body = do(t, srv, "POST", "/v1/nodes", adv, nil)
	leases, _ = body["leases"].([]any)
	if len(leases) != 1 {
		t.Fatalf("갱신에서 임대가 사라졌다: %v", body)
	}
	if second := leases[0].(map[string]any)["not_after"].(string); second <= first {
		t.Fatalf("★ not_after 가 안 밀렸다 ★ %s → %s", first, second)
	}
}

// ★ ADR-015 §3 — SELECT … FOR UPDATE SKIP LOCKED 가 배분 그 자체다 ★
// 여러 노드가 동시에 당겨도 한 단계는 정확히 한 노드에만 간다.
func TestClaimGoesToExactlyOneNode(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "a", map[string]string{"role": "x"}), nil)
	if code, _ := do(t, srv, "POST", "/v1/runs", oneStepRun("c1", "n1"), nil); code != 201 {
		t.Fatal("제출 실패")
	}

	const n = 8
	got := make([]int, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) { defer wg.Done(); got[i], _ = do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil) }(i)
	}
	wg.Wait()
	won := 0
	for _, c := range got {
		if c == 200 {
			won++
		} else if c != 204 {
			t.Fatalf("claim code=%d — 200 이나 204 여야 한다", c)
		}
	}
	if won != 1 {
		t.Fatalf("★ 한 단계가 %d 번 배분됐다 ★", won)
	}
}

// Mediator 가 시퀀서다 (ADR-014 결정 1) — 앞 단계가 끝나야 다음이 나온다.
func TestClaimRespectsOrder(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "a", map[string]string{"role": "x"}), nil)
	do(t, srv, "POST", "/v1/runs", oneStepRun("ord", "n1"), nil)

	code, first := do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)
	if code != 200 || first["name"] != "s1" {
		t.Fatalf("첫 단계가 안 나왔다: %d %v", code, first)
	}
	// 1 단계가 끝나기 전에는 2 단계가 안 나온다
	if code, _ := do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil); code != 204 {
		t.Fatalf("★ 순서를 어기고 다음 단계를 줬다 ★: %d", code)
	}
	// 보고하면 나온다
	if code, _ := do(t, srv, "POST", "/v1/runs/ord/steps/1/result",
		`{"node":"n1","exit_code":0,"produced":["s1"]}`, nil); code != 200 {
		t.Fatalf("보고 실패: %d", code)
	}
	code, second := do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)
	if code != 200 || second["name"] != "s2" {
		t.Fatalf("다음 단계가 안 나왔다: %d %v", code, second)
	}
}

// ★ I2 — 종료 상태에서 점유 장부가 비어 있다 ★
func TestLeasesReleasedOnCompletion(t *testing.T) {
	srv, _ := newServerFast(t)
	adv := advert("n1", "a", map[string]string{"role": "x"})
	do(t, srv, "POST", "/v1/nodes", adv, nil)
	do(t, srv, "POST", "/v1/runs", oneStepRun("done", "n1"), nil)

	for seq := 1; seq <= 2; seq++ {
		do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)
		do(t, srv, "POST", fmt.Sprintf("/v1/runs/done/steps/%d/result", seq),
			fmt.Sprintf(`{"node":"n1","exit_code":0,"produced":["s%d"]}`, seq), nil)
	}
	_, run := do(t, srv, "GET", "/v1/runs/done", "", nil)
	if run["state"] != "SUCCEEDED" {
		t.Fatalf("state=%v 기대 SUCCEEDED", run["state"])
	}
	_, body := do(t, srv, "POST", "/v1/nodes", adv, nil)
	if l, _ := body["leases"].([]any); len(l) != 0 {
		t.Fatalf("★ I2 위반 ★ 종료했는데 임대가 남았다: %v", l)
	}
	// 자원이 다시 쓸 수 있어야 한다
	if code, _ := do(t, srv, "POST", "/v1/runs", oneStepRun("next", "n1"), nil); code != 201 {
		t.Fatalf("★ 자원이 안 풀렸다 ★: %d", code)
	}
}

// ★ O6 — 갱신이 끊기면 시간이 회수한다 (ADR-008: 시간이 감시자다) ★
func TestReapReleasesExpiredLease(t *testing.T) {
	srv, st := newServerFast(t)
	adv := advert("n1", "a", map[string]string{"role": "x"})
	do(t, srv, "POST", "/v1/nodes", adv, nil)
	do(t, srv, "POST", "/v1/runs", oneStepRun("expire", "n1"), nil)

	// 하트비트가 끊긴 상황을 만든다 — not_after 를 과거로 민다
	if err := st.ForceExpire(context.Background(), "expire"); err != nil {
		t.Fatal(err)
	}
	n, err := st.Reap(context.Background(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("회수된 Run=%d 기대 1", n)
	}
	_, run := do(t, srv, "GET", "/v1/runs/expire", "", nil)
	if run["state"] != "FAILED" {
		t.Fatalf("state=%v 기대 FAILED", run["state"])
	}
	// ★ 그리고 자원을 다시 쓸 수 있다 — 409 → 201 ★
	if code, _ := do(t, srv, "POST", "/v1/runs", oneStepRun("after", "n1"), nil); code != 201 {
		t.Fatalf("★ O6 실패 ★ 회수 뒤에도 자원이 안 풀렸다: %d", code)
	}
}

// ═══ S5 — 계약 조건 대조 ═════════════════════════════════════════════════

// ★ O4 — 시연에서 가장 설명이 필요한 장면 ★
// 단계가 exit 0 으로 완주하고 test_result 를 냈다. 그 내용이 FAIL 이어도
// 계약이 "나왔는가" 만 물었으므로 Run 은 SUCCEEDED 다.
func TestVerdictO4(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "a", map[string]string{"role": "x"}), nil)
	body, _ := json.Marshal(map[string]any{
		"run_id":   "o4",
		"requires": []map[string]any{req("b", map[string]any{"role": "x"})},
		"steps": []map[string]any{
			{"id": "observe", "uses": "b", "run": []string{"true"}, "out": []string{"test_result"}},
		},
		"success_when": []map[string]any{{"step": "observe", "produced": []string{"test_result"}}},
	})
	if code, _ := do(t, srv, "POST", "/v1/runs", string(body), nil); code != 201 {
		t.Fatal("제출 실패")
	}
	do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)
	// 테스트는 FAIL 이지만 단계는 완주했고 산출물이 나왔다
	do(t, srv, "POST", "/v1/runs/o4/steps/1/result",
		`{"node":"n1","exit_code":0,"produced":["test_result","serial_log"]}`, nil)

	_, run := do(t, srv, "GET", "/v1/runs/o4", "", nil)
	if run["state"] != "SUCCEEDED" {
		t.Fatalf("★ O4 실패 ★ state=%v — 결과값이 통과 기준에 새어들었다", run["state"])
	}
	if run["verdict"] == nil {
		t.Fatal("verdict 가 안 남았다")
	}
}

// ★ 완주와 성공은 다르다 ★
// exit 2 로 끝난 것도 완주다. 성공 여부는 success_when 이 판정한다.
func TestNonZeroExitIsStillCompletion(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "a", map[string]string{"role": "x"}), nil)
	mk := func(id string, cond map[string]any) string {
		b, _ := json.Marshal(map[string]any{
			"run_id":   id,
			"requires": []map[string]any{req("b", map[string]any{"role": "x"})},
			"steps": []map[string]any{
				{"id": "build", "uses": "b", "run": []string{"false"}, "out": []string{"build_log"}},
			},
			"success_when": []map[string]any{cond},
		})
		return string(b)
	}
	// exit_code 를 물으면 실패한다
	do(t, srv, "POST", "/v1/runs", mk("strict", map[string]any{"step": "build", "exit_code": 0}), nil)
	do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)
	do(t, srv, "POST", "/v1/runs/strict/steps/1/result",
		`{"node":"n1","exit_code":2,"produced":["build_log"]}`, nil)
	_, run := do(t, srv, "GET", "/v1/runs/strict", "", nil)
	if run["state"] != "FAILED" {
		t.Fatalf("exit 2 인데 통과했다: %v", run["state"])
	}

	// ★ 안 물으면 안 본다 ★ (I3)
	do(t, srv, "POST", "/v1/runs", mk("loose", map[string]any{"step": "build", "produced": []string{"build_log"}}), nil)
	do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)
	do(t, srv, "POST", "/v1/runs/loose/steps/1/result",
		`{"node":"n1","exit_code":2,"produced":["build_log"]}`, nil)
	_, run = do(t, srv, "GET", "/v1/runs/loose", "", nil)
	if run["state"] != "SUCCEEDED" {
		t.Fatalf("★ I3 위반 ★ 계약이 안 물은 것으로 판정했다: %v", run["state"])
	}
}

// 완주하지 못한 단계는 VERIFYING 을 안 거친다 — 대조할 재료가 없다.
func TestStepThatCouldNotRun(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "a", map[string]string{"role": "x"}), nil)
	do(t, srv, "POST", "/v1/runs", oneStepRun("broke", "n1"), nil)
	do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)
	do(t, srv, "POST", "/v1/runs/broke/steps/1/result",
		`{"node":"n1","error":"임대 만료로 중단됨"}`, nil)

	_, run := do(t, srv, "GET", "/v1/runs/broke", "", nil)
	if run["state"] != "FAILED" {
		t.Fatalf("state=%v 기대 FAILED", run["state"])
	}
	// ★ 두 번째 단계는 돌지 않는다 ★ — RUNNING → RUNNING 이 멱등이 아니므로 재개하지 않는다
	if code, _ := do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil); code != 204 {
		t.Fatalf("★ 실패한 Run 의 다음 단계가 배분됐다 ★: %d", code)
	}
}

// ═══ S7 — 취소 (ADR-009) ═════════════════════════════════════════════════

func TestCancel(t *testing.T) {
	srv, _ := newServerFast(t)
	adv := advert("n1", "a", map[string]string{"role": "x"})
	do(t, srv, "POST", "/v1/nodes", adv, nil)
	do(t, srv, "POST", "/v1/runs", oneStepRun("cx", "n1"), nil)

	code, body := do(t, srv, "POST", "/v1/runs/cx/cancel", "", nil)
	if code != 200 || body["state"] != "FAILED" {
		t.Fatalf("취소 실패: %d %v", code, body)
	}

	// ★ 멱등이다 ★ — 이미 종료면 조용히 200
	if code, _ := do(t, srv, "POST", "/v1/runs/cx/cancel", "", nil); code != 200 {
		t.Fatalf("두 번째 취소가 %d", code)
	}

	// ★ I2 ★ 그리고 이 삭제가 곧 enode 에 대한 취소 통보다 (ADR-016) —
	// 다음 하트비트 응답의 목록에서 빠지는 것이 통보다.
	_, hb := do(t, srv, "POST", "/v1/nodes", adv, nil)
	if l, _ := hb["leases"].([]any); len(l) != 0 {
		t.Fatalf("★ 취소했는데 임대가 남았다 ★: %v", l)
	}

	// 취소된 Run 의 단계는 배분되지 않는다
	if code, _ := do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil); code != 204 {
		t.Fatalf("★ 취소된 Run 의 단계가 배분됐다 ★: %d", code)
	}
	// 자원을 다시 쓸 수 있다
	if code, _ := do(t, srv, "POST", "/v1/runs", oneStepRun("after-cx", "n1"), nil); code != 201 {
		t.Fatalf("★ 취소 뒤 자원이 안 풀렸다 ★: %d", code)
	}
	// ★ 왜 끝났는지가 Record 에 남는다 ★ (ADR-005)
	_, run := do(t, srv, "GET", "/v1/runs/cx", "", nil)
	v, _ := run["verdict"].(map[string]any)
	checks, _ := v["checks"].([]any)
	if len(checks) == 0 || checks[0].(map[string]any)["what"] != "cancelled" {
		t.Fatalf("취소 사유가 안 남았다: %v", run["verdict"])
	}
}

func TestCancelUnknownRun(t *testing.T) {
	srv, _ := newServerFast(t)
	if code, _ := do(t, srv, "POST", "/v1/runs/nope/cancel", "", nil); code != 404 {
		t.Fatalf("code=%d 기대 404", code)
	}
}

// ═══ S8 — blob 별 모양 · 스키마 검증 ═════════════════════════════════════

func schemaRun(id string, sch map[string]any) string {
	b, _ := json.Marshal(map[string]any{
		"run_id":   id,
		"requires": []map[string]any{req("b", map[string]any{"role": "x"})},
		"steps": []map[string]any{{"id": "hypothesis", "uses": "b",
			"run": []string{"true"}, "out": []string{"hypothesis"},
			"schema": map[string]any{"hypothesis": sch}}},
		"success_when": []map[string]any{{"step": "hypothesis", "produced": []string{"hypothesis"}}},
	})
	return string(b)
}

var okSchema = map[string]any{
	"type": "object", "required": []string{"status"},
	"properties": map[string]any{
		"status": map[string]any{"enum": []string{"found", "none"}},
		"reason": map[string]any{"type": "string"},
	},
}

// ★ ADR-020 의 경계선이 400 이다 ★
// 산문으로 두면 새어나가고, 그 순간 ADR-004 가 스키마를 통해 무너진다.
func TestJudgmentKeywordRejectedAtSubmit(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "a", map[string]string{"role": "x"}), nil)
	bad := map[string]any{"type": "object", "properties": map[string]any{
		"confidence": map[string]any{"type": "number", "minimum": 0.8}}}
	code, body := do(t, srv, "POST", "/v1/runs", schemaRun("judge", bad), nil)
	if code != 400 {
		t.Fatalf("★ 판정 키워드가 통과했다 ★ code=%d", code)
	}
	if e, _ := body["error"].(map[string]any); e == nil ||
		!strings.Contains(e["reason"].(string), "minimum") {
		t.Fatalf("어느 키워드가 문제인지가 안 나온다: %v", body)
	}
}

// ★ 스키마를 어긴 산출물은 산출물이 아니다 ★
// 422 로 거절되고 저장되지 않으므로 produced 가 불만족이 된다 —
// success_when 에 schema_ok 같은 새 조건이 생기지 않는다.
func TestBlobSchemaViolationIsNotStored(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "a", map[string]string{"role": "x"}), nil)
	do(t, srv, "POST", "/v1/runs", schemaRun("sv", okSchema), nil)
	do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)

	code, body := do(t, srv, "PUT", "/v1/runs/sv/steps/1/blob/hypothesis",
		`{"status":"maybe"}`, nil)
	if code != 422 {
		t.Fatalf("code=%d 기대 422", code)
	}
	if e, _ := body["error"].(map[string]any); e == nil ||
		!strings.Contains(e["reason"].(string), "enum") {
		t.Fatalf("위반 내역이 부실하다: %v", body) // feedback 으로 되먹여져야 한다
	}
	// 저장되지 않았다
	if code, _ := do(t, srv, "GET", "/v1/runs/sv/blob/hypothesis", "", nil); code != 404 {
		t.Fatalf("★ 어긴 산출물이 저장됐다 ★ code=%d", code)
	}
	// 정직한 답은 통과한다 — ★ 부재가 아니라 값으로 "못 하겠다" 를 말한다 ★
	if code, _ := do(t, srv, "PUT", "/v1/runs/sv/steps/1/blob/hypothesis",
		`{"status":"none","reason":"패치가 주석만 바꾼다"}`, nil); code != 204 {
		t.Fatalf("정직한 답이 거절됐다: %d", code)
	}
}

// ★ 별 모양 ★ — 노드끼리 직접 주고받지 않고 Mediator 를 경유한다.
func TestBlobRoundTrip(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "a", map[string]string{"role": "x"}), nil)
	do(t, srv, "POST", "/v1/runs", oneStepRun("bl", "n1"), nil)

	if code, _ := do(t, srv, "PUT", "/v1/runs/bl/steps/1/blob/artifact", "ELF-parent", nil); code != 204 {
		t.Fatalf("업로드 실패: %d", code)
	}
	req, _ := http.NewRequest("GET", srv.URL+"/v1/runs/bl/blob/artifact", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if string(b) != "ELF-parent" {
		t.Fatalf("받은 것: %q", b)
	}
	// 다음 단계가 같은 이름을 다시 내면 ★ 최신이 온다 ★
	do(t, srv, "PUT", "/v1/runs/bl/steps/2/blob/artifact", "ELF-patch", nil)
	req, _ = http.NewRequest("GET", srv.URL+"/v1/runs/bl/blob/artifact", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	b, _ = io.ReadAll(resp2.Body)
	if string(b) != "ELF-patch" {
		t.Fatalf("최신이 아니다: %q", b)
	}
}

// ★ I4 — 종료된 Run 에는 못 쓴다 ★
func TestBlobRefusedAfterTerminal(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "a", map[string]string{"role": "x"}), nil)
	do(t, srv, "POST", "/v1/runs", oneStepRun("term", "n1"), nil)
	do(t, srv, "POST", "/v1/runs/term/cancel", "", nil)
	if code, _ := do(t, srv, "PUT", "/v1/runs/term/steps/1/blob/x", "late", nil); code != 410 {
		t.Fatalf("★ 봉인된 뒤에 써졌다 ★ code=%d", code)
	}
}
