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

// ═══ S9 — ⑥ 의 재시도 루프 (ADR-013 · ADR-014 결정 1) ════════════════════

func loopRun(id string, maxAttempts int) string {
	b, _ := json.Marshal(map[string]any{
		"run_id":   id,
		"requires": []map[string]any{req("b", map[string]any{"role": "x"})},
		"steps": []map[string]any{
			{"id": "write_test", "uses": "b", "agent": map[string]any{"ask": "never"},
				"out": []string{"test_source"}, "validate_with": "build",
				"max_attempts": maxAttempts, "feedback": []string{"build_log"}},
			{"id": "build", "uses": "b", "run": []string{"true"},
				"in": map[string]any{"from": []string{"test_source"}}, "out": []string{"build_log"}},
		},
		"success_when": []map[string]any{
			{"step": "write_test", "within_attempts": true, "produced": []string{"test_source"}},
			{"step": "build", "exit_code": 0},
		},
	})
	return string(b)
}

// ★ 루프를 도는 주체는 Mediator 다 ★ — agent 노드와 build 노드는 서로를 모른다.
func TestRetryLoopIsDrivenByMediator(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "a", map[string]string{"role": "x"}), nil)
	do(t, srv, "POST", "/v1/runs", loopRun("loop", 3), nil)

	// 1회차 — agent 는 내고, 검증자가 실패한다
	do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)
	do(t, srv, "POST", "/v1/runs/loop/steps/1/result", `{"node":"n1","produced":["test_source"]}`, nil)
	do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)
	code, body := do(t, srv, "POST", "/v1/runs/loop/steps/2/result",
		`{"node":"n1","exit_code":2,"produced":["build_log"]}`, nil)
	if code != 200 || body["retried"] != true {
		t.Fatalf("★ 되먹여 재시도하지 않았다 ★: %d %v", code, body)
	}

	// ★ agent 단계가 다시 나와야 하고 회차가 올라 있어야 한다 ★
	code, step := do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)
	if code != 200 || step["name"] != "write_test" {
		t.Fatalf("재시도 단계가 안 나왔다: %d %v", code, step)
	}
	if a, _ := step["attempt"].(float64); a != 1 {
		t.Fatalf("attempt=%v 기대 1 — 되먹임을 프롬프트에 실을 근거가 없다", step["attempt"])
	}
	if fb, _ := step["feedback"].([]any); len(fb) != 1 {
		t.Fatalf("feedback 이름이 안 실렸다: %v", step["feedback"])
	}

	// 2회차 — 이번엔 검증자가 통과한다
	do(t, srv, "POST", "/v1/runs/loop/steps/1/result", `{"node":"n1","produced":["test_source"]}`, nil)
	do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)
	do(t, srv, "POST", "/v1/runs/loop/steps/2/result",
		`{"node":"n1","exit_code":0,"produced":["build_log"]}`, nil)

	_, run := do(t, srv, "GET", "/v1/runs/loop", "", nil)
	if run["state"] != "SUCCEEDED" {
		t.Fatalf("state=%v 기대 SUCCEEDED · %v", run["state"], run["verdict"])
	}
}

// ★ 소진은 verdict 가 잡는다 ★ — 루프는 제어 흐름이고 성패는 success_when 이 정한다.
func TestRetryExhaustionFailsViaWithinAttempts(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "a", map[string]string{"role": "x"}), nil)
	do(t, srv, "POST", "/v1/runs", loopRun("ex", 2), nil)

	for i := 0; i < 2; i++ {
		do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)
		do(t, srv, "POST", "/v1/runs/ex/steps/1/result", `{"node":"n1","produced":["test_source"]}`, nil)
		do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)
		do(t, srv, "POST", "/v1/runs/ex/steps/2/result",
			`{"node":"n1","exit_code":2,"produced":["build_log"]}`, nil)
	}
	_, run := do(t, srv, "GET", "/v1/runs/ex", "", nil)
	if run["state"] != "FAILED" {
		t.Fatalf("state=%v 기대 FAILED", run["state"])
	}
	v, _ := run["verdict"].(map[string]any)
	checks, _ := v["checks"].([]any)
	found := false
	for _, c := range checks {
		m := c.(map[string]any)
		if m["what"] == "within_attempts" && m["ok"] == false {
			found = true
		}
	}
	if !found {
		t.Fatalf("★ 소진이 verdict 에 안 잡혔다 ★: %v", v)
	}
}

// ═══ S10 — GET /v1/capabilities ══════════════════════════════════════════

// ★ 이것이 우리 층의 tools/list 다 ★ — ADR-012 가 어휘를 창발시켰으므로
// 읽는 경로가 없으면 계약을 쓰는 쪽이 문자열을 추측한다.
func TestCapabilities(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "a",
		map[string]string{"harness": "claude", "repo": "corp/linux", "arch": "armv7"}), nil)
	do(t, srv, "POST", "/v1/nodes", advert("n2", "b",
		map[string]string{"board": "SoC-X", "tag": "board-042"}), nil)

	_, body := do(t, srv, "GET", "/v1/capabilities", "", nil)
	caps, _ := body["capabilities"].([]any)
	if len(caps) != 1 {
		t.Fatalf("★ ADR-019 이후 어휘는 하나다 ★: %v", body)
	}
	c := caps[0].(map[string]any)
	if c["capability"] != "agent.reason" {
		t.Fatalf("capability=%v", c["capability"])
	}
	// ★ nodes 는 총수(존재)다 ★ — 여유가 아니다
	if n, _ := c["nodes"].(float64); n != 2 {
		t.Fatalf("nodes=%v 기대 2", c["nodes"])
	}
	attrs, _ := c["attrs"].(map[string]any)
	for _, k := range []string{"harness", "repo", "arch", "board", "tag"} {
		if _, ok := attrs[k]; !ok {
			t.Fatalf("속성 %q 가 안 보인다 — 계약 작성자가 어휘를 못 읽는다: %v", k, attrs)
		}
	}
}

// ★ 존재는 답하고 여유는 답하지 않는다 ★ (ADR-014 결정 3)
func TestCapabilitiesDoesNotLeakAvailability(t *testing.T) {
	srv, _ := newServerFast(t)
	adv := advert("n1", "a", map[string]string{"role": "x"})
	do(t, srv, "POST", "/v1/nodes", adv, nil)
	do(t, srv, "POST", "/v1/runs", oneStepRun("busy", "n1"), nil) // 점유한다

	_, body := do(t, srv, "GET", "/v1/capabilities", "", nil)
	caps, _ := body["capabilities"].([]any)
	c := caps[0].(map[string]any)
	if n, _ := c["nodes"].(float64); n != 1 {
		t.Fatalf("★ 점유 상태가 어휘에 샜다 ★ nodes=%v — 총수여야 한다", c["nodes"])
	}
	if _, leaked := c["free"]; leaked {
		t.Fatal("★ 여유를 노출했다 ★ 확인-후-행동 경쟁을 부른다")
	}
}

// ★ claim 이 요청자를 싣는다 ★ (R1/R2)
//
// 지금은 아무도 이 값을 안 본다 — MVP 인증은 transparent 다. 그래서
// ★ 시험이 없으면 비어 있어도 아무도 모른다 ★. R2 에서 「요청자 신원으로
// 하네스를 돌린다」를 켤 때 그제서야 발견되는 종류의 구멍이다.
func TestClaim이_요청자를_싣는다(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "a", map[string]string{"role": "x"}), nil)
	if code, _ := do(t, srv, "POST", "/v1/runs", oneStepRun("c-req", "n1"), nil); code != 201 {
		t.Fatal("제출 실패")
	}
	code, body := do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)
	if code != 200 {
		t.Fatalf("claim code=%d", code)
	}
	if got, _ := body["requester"].(string); got != "taeels@gmail.com" {
		t.Fatalf("★ 요청자가 안 실렸다 ★: %q (runs.principal 이어야 한다)", got)
	}
}

// ═══ 분기 — ★ 식을 평가하지 않고 이름을 고른다 ★ (ADR-022 §7.2) ═══════════
//
// 에이전트가 이름 하나를 산출물로 내고, Mediator 는 ★ 그 이름의 단계를 찾아
// 실행할 뿐 ★ 이다. 안 간 쪽은 SKIPPED 가 되고 뒤 단계는 그것을 넘어 진행한다.
func TestDispatch_이름을_골라_한쪽만_돈다(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "a", map[string]string{"role": "x"}), nil)

	triage := map[string]any{
		"id": "triage", "uses": "b",
		"agent": map[string]any{"ask": "never"},
		"out":   []string{"route"},
		// ★ enum 이 어휘를 못 박는다 ★ — 벗어나면 PUT blob 이 422 다 (ADR-020).
		"schema": map[string]any{"route": map[string]any{
			"type": "object", "required": []string{"next"},
			"properties": map[string]any{
				"next": map[string]any{"enum": []string{"full", "quick"}}}}},
		"dispatch": map[string]any{"from": "route.next", "to": []string{"full", "quick"}},
	}
	body := contractJSON("br", []map[string]any{req("b", map[string]any{"role": "x"})},
		[]map[string]any{triage, runStep("full", "b"), runStep("quick", "b")})
	if code, _ := do(t, srv, "POST", "/v1/runs", body, nil); code != 201 {
		t.Fatalf("제출 실패: %d", code)
	}

	if code, c := do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil); code != 200 || c["name"] != "triage" {
		t.Fatalf("triage 가 안 나왔다: %d %v", code, c)
	}
	if code, _ := do(t, srv, "PUT", "/v1/runs/br/steps/1/blob/route",
		`{"next":"quick"}`, nil); code != 200 && code != 201 && code != 204 {
		t.Fatalf("산출물 저장 실패: %d", code)
	}
	if code, _ := do(t, srv, "POST", "/v1/runs/br/steps/1/result",
		`{"node":"n1","produced":["route"]}`, nil); code != 200 {
		t.Fatalf("보고 실패: %d", code)
	}

	// ★ 고른 쪽이 나온다 — 안 간 쪽(full, seq 2)을 넘어서 ★
	code, next := do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)
	if code != 200 || next["name"] != "quick" {
		t.Fatalf("★ 고른 경로가 안 나왔다 ★: %d %v", code, next)
	}
}

// ★ 이름을 못 고르면 그 단계가 FAILED 다 ★
//
// 결과가 나쁜 것이 아니라 ★ 계약이 요구한 것을 못 낸 것 ★ 이므로
// "완주하지 못함" 과 같은 자리다 (ADR-004 를 안 건드린다).
// 그리고 ★ 안 간 쪽이 열린 채로 남지 않는다 ★ — 아무 경로도 안 돈다.
func TestDispatch_이름을_못_고르면_그_단계가_실패한다(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "a", map[string]string{"role": "x"}), nil)

	triage := map[string]any{
		"id": "triage", "uses": "b",
		"agent": map[string]any{"ask": "never"},
		"out":   []string{"route"},
		// ★ 스키마를 일부러 안 단다 ★ — 그래야 엉뚱한 값이 PUT 을 통과해
		// 라우터까지 온다. 두 겹 중 ★ 두 번째 겹 ★ 만 시험하는 것이다.
		"dispatch": map[string]any{"from": "route.next", "to": []string{"full", "quick"}},
	}
	body := contractJSON("bad", []map[string]any{req("b", map[string]any{"role": "x"})},
		[]map[string]any{triage, runStep("full", "b"), runStep("quick", "b")})
	do(t, srv, "POST", "/v1/runs", body, nil)
	do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)
	do(t, srv, "PUT", "/v1/runs/bad/steps/1/blob/route", `{"next":"없는것"}`, nil)
	if code, _ := do(t, srv, "POST", "/v1/runs/bad/steps/1/result",
		`{"node":"n1","produced":["route"]}`, nil); code != 200 {
		t.Fatalf("보고는 받아야 한다: %d", code)
	}
	// 그 단계가 실패했으므로 ★ 아무 경로도 안 나온다 ★
	if code, c := do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil); code == 200 {
		t.Fatalf("★ 라우팅이 실패했는데 경로가 돌았다 ★: %v", c)
	}
}

// ═══ 폭 — ★ 리스트가 강제하던 전순서를 간선으로 푼다 ★ (ADR-023) ═════════
//
// 오늘까지 의존은 ★ 목록에서의 위치 ★ 였다. steps[] 가 리스트이므로 리스트가
// 전순서를 주고, 그래서 한 번에 하나만 돌았다. needs 는 그것을 선언된 간선으로
// 바꾼다 — ★ 표현이 늘지 않고 관계만 는다 ★.

// ★ 안 적으면 오늘 그대로 ★ — 기본값이 [직전 단계] 라 순차 계약이 안 바뀐다.
func TestNeeds_안_적으면_오늘과_같은_순서로_돈다(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "a", map[string]string{"role": "x"}), nil)
	body := contractJSON("seq3", []map[string]any{req("b", map[string]any{"role": "x"})},
		[]map[string]any{runStep("one", "b"), runStep("two", "b"), runStep("three", "b")})
	if code, _ := do(t, srv, "POST", "/v1/runs", body, nil); code != 201 {
		t.Fatalf("제출 실패: %d", code)
	}
	for i, want := range []string{"one", "two", "three"} {
		code, c := do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)
		if code != 200 || c["name"] != want {
			t.Fatalf("%d 번째로 %v 가 나왔다 — %q 여야 한다", i+1, c["name"], want)
		}
		if code, _ := do(t, srv, "POST",
			fmt.Sprintf("/v1/runs/seq3/steps/%d/result", i+1),
			`{"node":"n1","exit_code":0}`, nil); code != 200 {
			t.Fatalf("보고 실패: %d", code)
		}
	}
}

// ★ 의존이 없는 두 단계가 동시에 집힌다 ★ — 여기가 폭이 실물이 되는 자리다.
//
// 오늘(seq 게이트)이면 두 번째 노드는 ★ 앞 단계가 안 끝났다 ★ 는 이유로 204 를
// 받는다. needs 가 열리면 서로 안 가리키므로 ★ 둘 다 집힌다 ★.
func TestNeeds_의존_없는_둘이_동시에_집힌다(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("p1", "a", map[string]string{"role": "x"}), nil)
	do(t, srv, "POST", "/v1/nodes", advert("p2", "a", map[string]string{"role": "y"}), nil)

	left := runStep("left", "b")
	left["needs"] = []string{}
	right := runStep("right", "c")
	right["needs"] = []string{}
	body := contractJSON("par", []map[string]any{
		req("b", map[string]any{"role": "x"}), req("c", map[string]any{"role": "y"})},
		[]map[string]any{left, right})
	if code, _ := do(t, srv, "POST", "/v1/runs", body, nil); code != 201 {
		t.Fatalf("제출 실패: %d", code)
	}
	code1, c1 := do(t, srv, "POST", "/v1/nodes/p1/claim", "", nil)
	code2, c2 := do(t, srv, "POST", "/v1/nodes/p2/claim", "", nil)
	if code1 != 200 || c1["name"] != "left" {
		t.Fatalf("p1 이 left 를 못 집었다: %d %v", code1, c1)
	}
	if code2 != 200 || c2["name"] != "right" {
		t.Fatalf("★ 두 번째가 동시에 안 집혔다 ★: %d %v — "+
			"전순서가 아직 남아 있다는 뜻이다", code2, c2)
	}
}

// ★ join 은 needs 가 전부 끝난 뒤에만 집힌다 ★ — 폭을 열어도 합류는 기다린다.
func TestNeeds_조인은_전부_끝난_뒤에_집힌다(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("j1", "a", map[string]string{"role": "x"}), nil)
	do(t, srv, "POST", "/v1/nodes", advert("j2", "a", map[string]string{"role": "y"}), nil)

	left := runStep("left", "b")
	left["needs"] = []string{}
	right := runStep("right", "c")
	right["needs"] = []string{}
	join := runStep("join", "b")
	join["needs"] = []string{"left", "right"}
	body := contractJSON("joi", []map[string]any{
		req("b", map[string]any{"role": "x"}), req("c", map[string]any{"role": "y"})},
		[]map[string]any{left, right, join})
	if code, _ := do(t, srv, "POST", "/v1/runs", body, nil); code != 201 {
		t.Fatalf("제출 실패: %d", code)
	}
	do(t, srv, "POST", "/v1/nodes/j1/claim", "", nil)                                     // left
	do(t, srv, "POST", "/v1/nodes/j2/claim", "", nil)                                     // right
	do(t, srv, "POST", "/v1/runs/joi/steps/1/result", `{"node":"j1","exit_code":0}`, nil) // left 만 끝난다
	if code, c := do(t, srv, "POST", "/v1/nodes/j1/claim", "", nil); code == 200 {
		t.Fatalf("★ 한쪽만 끝났는데 join 이 집혔다 ★: %v", c)
	}
	do(t, srv, "POST", "/v1/runs/joi/steps/2/result", `{"node":"j2","exit_code":0}`, nil) // right 도 끝난다
	if code, c := do(t, srv, "POST", "/v1/nodes/j1/claim", "", nil); code != 200 || c["name"] != "join" {
		t.Fatalf("★ 전부 끝났는데 join 이 안 집혔다 ★: %d %v", code, c)
	}
}

// ★ SKIPPED 는 간선을 따라 전파한다 ★ (ADR-023 §7).
//
// dispatch.to 만 SKIPPED 로 바꾸면 갈림길이 ★ 단계 하나짜리일 때만 ★ 맞다.
// 안 간 경로가 두 단계 이상이면 그 뒷단계가 PENDING 으로 남아 ★ 그대로 실행된다 ★.
func TestSkipped_간선을_따라_전파된다(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("s1", "a", map[string]string{"role": "x"}), nil)

	triage := map[string]any{
		"id": "triage", "uses": "b",
		"agent": map[string]any{"ask": "never"},
		"out":   []string{"route"},
		"schema": map[string]any{"route": map[string]any{
			"type": "object", "required": []string{"next"},
			"properties": map[string]any{
				"next": map[string]any{"enum": []string{"full", "quick"}}}}},
		"dispatch": map[string]any{"from": "route.next", "to": []string{"full", "quick"}},
	}
	// full 다음에 full2 가 매달린다. quick 을 고르면 ★ 둘 다 죽어야 한다 ★.
	full2 := runStep("full2", "b")
	full2["needs"] = []string{"full"}
	body := contractJSON("prop", []map[string]any{req("b", map[string]any{"role": "x"})},
		[]map[string]any{triage, runStep("full", "b"), full2, runStep("quick", "b")})
	if code, _ := do(t, srv, "POST", "/v1/runs", body, nil); code != 201 {
		t.Fatalf("제출 실패: %d", code)
	}
	do(t, srv, "POST", "/v1/nodes/s1/claim", "", nil) // triage
	do(t, srv, "PUT", "/v1/runs/prop/steps/1/blob/route", `{"next":"quick"}`, nil)
	if code, _ := do(t, srv, "POST", "/v1/runs/prop/steps/1/result",
		`{"node":"s1","produced":["route"]}`, nil); code != 200 {
		t.Fatalf("보고 실패: %d", code)
	}
	// claim 은 seq 순이므로 full2(seq 3)가 살아 있으면 quick(seq 4)보다 먼저 나온다.
	code, next := do(t, srv, "POST", "/v1/nodes/s1/claim", "", nil)
	if code != 200 || next["name"] != "quick" {
		t.Fatalf("★ 안 간 경로의 뒷단계가 살아남았다 ★: %d %v — "+
			"전파가 간선을 안 따라갔다", code, next)
	}
}

// ═══ 계획 위임 — ★ 오케스트레이터가 나머지 단계를 짓는다 ★ (ADR-022 §6.3) ═══
//
// 에이전트가 계약을 ★ 파일로 쓰고 ★ enode 가 제출한다 — 토큰은 enode 에만 남아
// R1 이 안 깨진다. Mediator 는 그것을 ★ 검증해서 받는다 ★: 무엇이 좋은 계획인지
// 안 보고 ★ 유효한 계약인지만 ★ 본다 (ADR-004 를 안 건드린다).

// planStep 은 expands 단계 하나다. 스키마는 ★ 형식만 ★ 제약한다 (ADR-020).
func planStep(uses string) map[string]any {
	return map[string]any{
		"id": "plan", "uses": uses,
		"agent": map[string]any{"ask": "never"},
		"out":   []string{"plan"},
		"schema": map[string]any{"plan": map[string]any{
			"type": "object", "required": []string{"steps"}}},
		"expands": true,
	}
}

// ★ 계획이 단계를 늘리고, 늘어난 단계가 집힌다 ★
func TestExpands_계획이_단계를_늘린다(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("e1", "a", map[string]string{"role": "x"}), nil)
	body := contractJSON("exp", []map[string]any{req("b", map[string]any{"role": "x"})},
		[]map[string]any{planStep("b")})
	if code, _ := do(t, srv, "POST", "/v1/runs", body, nil); code != 201 {
		t.Fatalf("제출 실패: %d", code)
	}
	if code, c := do(t, srv, "POST", "/v1/nodes/e1/claim", "", nil); code != 200 || c["name"] != "plan" {
		t.Fatalf("plan 이 안 나왔다: %d %v", code, c)
	}
	// ★ 계획은 산출물이다 ★ — 스키마 검증을 통과해야 저장된다.
	if code, _ := do(t, srv, "PUT", "/v1/runs/exp/steps/1/blob/plan",
		`{"steps":[{"id":"built","uses":"b","run":["true"],"out":["built"]}]}`, nil); code >= 300 {
		t.Fatalf("계획 저장 실패: %d", code)
	}
	if code, _ := do(t, srv, "POST", "/v1/runs/exp/steps/1/result",
		`{"node":"e1","produced":["plan"]}`, nil); code != 200 {
		t.Fatalf("보고 실패: %d", code)
	}
	code, next := do(t, srv, "POST", "/v1/nodes/e1/claim", "", nil)
	if code != 200 || next["name"] != "built" {
		t.Fatalf("★ 지어진 단계가 안 집혔다 ★: %d %v", code, next)
	}
}

// ★ 자원은 여전히 사용자가 선언한다 ★ (갈래 A) — 계획이 requires 에 없는
// 역할을 쓰면 계약 검증이 거절하고, 그 단계가 FAILED 다.
//
// 자원까지 위임하는 것은 P5(acquire)이고 그것은 I5 를 건드린다.
func TestExpands_없는_역할을_쓰면_그_단계가_실패한다(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("e2", "a", map[string]string{"role": "x"}), nil)
	body := contractJSON("exp2", []map[string]any{req("b", map[string]any{"role": "x"})},
		[]map[string]any{planStep("b")})
	if code, _ := do(t, srv, "POST", "/v1/runs", body, nil); code != 201 {
		t.Fatalf("제출 실패: %d", code)
	}
	do(t, srv, "POST", "/v1/nodes/e2/claim", "", nil)
	do(t, srv, "PUT", "/v1/runs/exp2/steps/1/blob/plan",
		`{"steps":[{"id":"built","uses":"★없는역할★","run":["true"],"out":["built"]}]}`, nil)
	if code, _ := do(t, srv, "POST", "/v1/runs/exp2/steps/1/result",
		`{"node":"e2","produced":["plan"]}`, nil); code != 200 {
		t.Fatalf("보고 자체는 받아야 한다: %d", code)
	}
	// 늘어나지 않았으므로 집을 것이 없다.
	if code, c := do(t, srv, "POST", "/v1/nodes/e2/claim", "", nil); code == 200 {
		t.Fatalf("★ 유효하지 않은 계획이 함대로 들어왔다 ★: %v", c)
	}
}

// ★ 계획이 판정 기준을 짓지 못한다 ★ (ADR-022 §7.7)
//
// 계획을 짓는 것과 성패의 기준을 짓는 것은 다른 권한이다. 후자를 넘기면
// ★ 판정 기준을 판정 대상이 정하게 되어 ★ ADR-004 의 뿌리가 흔들린다.
// ⇒ 조용히 무시하지 않고 그 단계를 실패시킨다.
func TestExpands_계획은_판정_기준을_못_짓는다(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("e3", "a", map[string]string{"role": "x"}), nil)
	body := contractJSON("exp3", []map[string]any{req("b", map[string]any{"role": "x"})},
		[]map[string]any{planStep("b")})
	if code, _ := do(t, srv, "POST", "/v1/runs", body, nil); code != 201 {
		t.Fatalf("제출 실패: %d", code)
	}
	do(t, srv, "POST", "/v1/nodes/e3/claim", "", nil)
	do(t, srv, "PUT", "/v1/runs/exp3/steps/1/blob/plan",
		`{"steps":[{"id":"built","uses":"b","run":["true"],"out":["built"]}],`+
			`"success_when":[{"step":"built","exit_code":0}]}`, nil)
	do(t, srv, "POST", "/v1/runs/exp3/steps/1/result", `{"node":"e3","produced":["plan"]}`, nil)
	if code, c := do(t, srv, "POST", "/v1/nodes/e3/claim", "", nil); code == 200 {
		t.Fatalf("★ 계획이 지은 판정 기준이 들어왔다 ★: %v", c)
	}
}

// ★ 봉인에 v2 가 남는다 ★ (ADR-005 성질 1·4)
//
// v1 은 언제나 ★ 제출 전문 ★ 이고 v2 는 계획이 지은 판이다. 앞 판을 고치지 않고
// 붙이므로, ★ 봉인된 묶음만 열어서 「무엇을 요청했고 무엇이 지어졌나」 ★ 를 안다.
func TestExpands_봉인에_계약의_열이_남는다(t *testing.T) {
	srv, st := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("e4", "a", map[string]string{"role": "x"}), nil)
	c := map[string]any{
		"run_id":   "exp4",
		"requires": []map[string]any{req("b", map[string]any{"role": "x"})},
		"steps":    []map[string]any{planStep("b")},
		// ★ 판정은 제출 시점 계약이 든다 ★ — 지어질 단계를 가리킬 수 없다.
		"success_when": []map[string]any{{"step": "plan", "produced": []string{"plan"}}},
	}
	b, _ := json.Marshal(c)
	if code, _ := do(t, srv, "POST", "/v1/runs", string(b), nil); code != 201 {
		t.Fatalf("제출 실패: %d", code)
	}
	do(t, srv, "POST", "/v1/nodes/e4/claim", "", nil)
	do(t, srv, "PUT", "/v1/runs/exp4/steps/1/blob/plan",
		`{"steps":[{"id":"built","uses":"b","run":["true"],"out":["built"]}]}`, nil)
	do(t, srv, "POST", "/v1/runs/exp4/steps/1/result", `{"node":"e4","produced":["plan"]}`, nil)
	if code, next := do(t, srv, "POST", "/v1/nodes/e4/claim", "", nil); code != 200 || next["name"] != "built" {
		t.Fatalf("지어진 단계가 안 집혔다: %d %v", code, next)
	}
	do(t, srv, "POST", "/v1/runs/exp4/steps/2/result", `{"node":"e4","exit_code":0}`, nil)

	raw, err := os.ReadFile(filepath.Join(st.Records.Root, "run-exp4", "manifest.json"))
	if err != nil {
		t.Fatalf("★ 봉인이 안 됐다 ★: %v", err)
	}
	var m struct {
		Contract []struct {
			V        int    `json:"v"`
			By       string `json:"by"`
			Evidence string `json:"evidence"`
			Steps    []struct {
				ID string `json:"id"`
			} `json:"steps"`
		} `json:"contract"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if len(m.Contract) != 2 {
		t.Fatalf("★ 판이 %d 개다 ★ — v1(제출본)과 v2(계획)로 둘이어야 한다", len(m.Contract))
	}
	if m.Contract[0].By != "requester" || len(m.Contract[0].Steps) != 1 {
		t.Fatalf("★ v1 이 제출 전문이 아니다 ★: by=%q steps=%d — 성질 4 가 깨진다",
			m.Contract[0].By, len(m.Contract[0].Steps))
	}
	if m.Contract[1].By != "step:plan" || len(m.Contract[1].Steps) != 2 {
		t.Fatalf("★ v2 가 틀렸다 ★: by=%q steps=%d", m.Contract[1].By, len(m.Contract[1].Steps))
	}
	if m.Contract[1].Evidence == "" {
		t.Fatal("★ 무엇을 보고 지었는지가 안 남았다 ★ — evidence 가 비었다")
	}
}

// ═══ 실행 중 관측 — ★ Record 가 아니다 ★ (ADR-025) ═══════════════════════
//
// 폭이 1 일 때는 "지금 ④ 단계" 한 줄로 족했다. 여러 가지가 동시에 살면
// 각각이 다른 상태에 있고, GET record 는 종료 전이면 409 다 (I4).
// ⇒ 진행을 읽을 경로가 ★ 선택이 아니라 필수 ★ 가 된다.

func TestObserve_병렬로_도는_가지들이_각각_보인다(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("o1", "왼쪽기계", map[string]string{"role": "x"}), nil)
	do(t, srv, "POST", "/v1/nodes", advert("o2", "오른쪽기계", map[string]string{"role": "y"}), nil)

	left := runStep("left", "b")
	left["needs"] = []string{}
	right := runStep("right", "c")
	right["needs"] = []string{}
	join := runStep("join", "b")
	join["needs"] = []string{"left", "right"}
	body := contractJSON("obs", []map[string]any{
		req("b", map[string]any{"role": "x"}), req("c", map[string]any{"role": "y"})},
		[]map[string]any{left, right, join})
	if code, _ := do(t, srv, "POST", "/v1/runs", body, nil); code != 201 {
		t.Fatalf("제출 실패: %d", code)
	}
	do(t, srv, "POST", "/v1/nodes/o1/claim", "", nil)
	do(t, srv, "POST", "/v1/nodes/o2/claim", "", nil)

	// ★ 종료 전인데 답한다 ★ — record 는 409 인 시점이다.
	if code, _ := do(t, srv, "GET", "/v1/runs/obs/record", "", nil); code != 409 {
		t.Fatalf("★ 봉인 전 Record 가 409 가 아니다 ★: %d — I4 가 흔들린다", code)
	}
	code, v := do(t, srv, "GET", "/v1/runs/obs", "", nil)
	if code != 200 {
		t.Fatalf("관측 실패: %d", code)
	}
	raw, _ := json.Marshal(v["steps"])
	var steps []struct {
		Seq   int      `json:"seq"`
		ID    string   `json:"id"`
		State string   `json:"state"`
		Node  string   `json:"node"`
		Needs []string `json:"needs"`
	}
	if err := json.Unmarshal(raw, &steps); err != nil || len(steps) != 3 {
		t.Fatalf("★ 단계가 안 보인다 ★: %v (%d 개)", string(raw), len(steps))
	}
	// ★ 두 가지가 동시에 돌고 서로 다른 기계에 있다 ★ — Case D 를 실행 중에 읽는다.
	if steps[0].State != "CLAIMED" || steps[1].State != "CLAIMED" {
		t.Fatalf("★ 동시에 도는 것이 안 보인다 ★: %q %q", steps[0].State, steps[1].State)
	}
	if steps[0].Node == steps[1].Node || steps[0].Node == "" {
		t.Fatalf("★ 어느 기계에서 도는지가 안 보인다 ★: %q %q", steps[0].Node, steps[1].Node)
	}
	// ★ 빈 needs 도 내보낸다 ★ — [] 는 "안 기다린다" 는 뜻이고 가지의 시작점이다.
	if steps[0].Needs == nil || len(steps[0].Needs) != 0 {
		t.Fatalf("★ 빈 needs 가 안 나왔다 ★: %v — 읽는 쪽이 기본값을 추측하게 된다", steps[0].Needs)
	}
	if len(steps[2].Needs) != 2 {
		t.Fatalf("★ 합류의 간선이 안 보인다 ★: %v", steps[2].Needs)
	}
	if steps[2].State != "PENDING" {
		t.Fatalf("아직 안 집힌 단계가 %q 다", steps[2].State)
	}
}

// ★ 건너뛴 가지가 관측에서도 보인다 ★ — 결과가 없는 것과 다르다는 것이
// 실행 중에도 읽혀야 한다 (그러지 않으면 크래시와 구분이 안 된다).
func TestObserve_건너뛴_가지가_보인다(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("o3", "a", map[string]string{"role": "x"}), nil)
	triage := map[string]any{
		"id": "triage", "uses": "b",
		"agent": map[string]any{"ask": "never"},
		"out":   []string{"route"},
		"schema": map[string]any{"route": map[string]any{
			"type": "object", "required": []string{"next"},
			"properties": map[string]any{
				"next": map[string]any{"enum": []string{"full", "quick"}}}}},
		"dispatch": map[string]any{"from": "route.next", "to": []string{"full", "quick"}},
	}
	body := contractJSON("obs2", []map[string]any{req("b", map[string]any{"role": "x"})},
		[]map[string]any{triage, runStep("full", "b"), runStep("quick", "b")})
	do(t, srv, "POST", "/v1/runs", body, nil)
	do(t, srv, "POST", "/v1/nodes/o3/claim", "", nil)
	do(t, srv, "PUT", "/v1/runs/obs2/steps/1/blob/route", `{"next":"quick"}`, nil)
	do(t, srv, "POST", "/v1/runs/obs2/steps/1/result", `{"node":"o3","produced":["route"]}`, nil)

	_, v := do(t, srv, "GET", "/v1/runs/obs2", "", nil)
	raw, _ := json.Marshal(v["steps"])
	var steps []struct {
		ID    string `json:"id"`
		State string `json:"state"`
	}
	_ = json.Unmarshal(raw, &steps)
	got := map[string]string{}
	for _, st := range steps {
		got[st.ID] = st.State
	}
	if got["full"] != "SKIPPED" {
		t.Fatalf("★ 안 간 가지가 SKIPPED 로 안 보인다 ★: %q", got["full"])
	}
	if got["quick"] != "PENDING" {
		t.Fatalf("고른 가지가 %q 다", got["quick"])
	}
}

// ═══ 원장 — ★ 시야를 순서에서 유도하지 않는다 ★ (ADR-023 §6) ═════════════
//
// 조상인지 형제인지를 묻지 않고 ★ 그 시점 원장에 있는가 ★ 만 본다.
// 그래서 병렬이 「최신」을 안 깨고, leaf 라고 볼 것이 없지도 않다.

// workContract 는 work 를 채운 계약이다 — ★ Work 의 키가 있어야 ★ 원장이
// Run 을 넘을 수 있다 (ADR-023 §6.5.2).
func workContract(runID, changeID string, patchset int, scope string,
	steps []map[string]any) string {
	c := map[string]any{
		"run_id": runID,
		"work": map[string]any{
			"system": "gerrit", "change_id": changeID, "patchset": patchset,
			"parent_rev": "aaa", "patch_rev": "bbb"},
		"requires": []map[string]any{req("b", map[string]any{"role": "x"})},
		"steps":    steps,
	}
	if scope != "" {
		c["ledger"] = map[string]any{"scope": scope}
	}
	b, _ := json.Marshal(c)
	return string(b)
}

// ★ 원장은 목록이다. 본문이 아니다 ★ (ADR-023 §6.3)
func TestLedger_목록을_준다_본문은_아니다(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("L1", "a", map[string]string{"role": "x"}), nil)
	step := map[string]any{"id": "note", "uses": "b",
		"agent": map[string]any{"ask": "never"}, "out": []string{"note"}}
	do(t, srv, "POST", "/v1/runs", workContract("led1", "555", 1, "", []map[string]any{step}), nil)
	do(t, srv, "POST", "/v1/nodes/L1/claim", "", nil)
	do(t, srv, "PUT", "/v1/runs/led1/steps/1/blob/note", `{"말":"이 줄은 본문이다"}`, nil)
	do(t, srv, "POST", "/v1/runs/led1/steps/1/result", `{"node":"L1","produced":["note"]}`, nil)

	code, v := do(t, srv, "GET", "/v1/runs/led1/ledger", "", nil)
	if code != 200 {
		t.Fatalf("원장 조회 실패: %d", code)
	}
	raw, _ := json.Marshal(v["entries"])
	if strings.Contains(string(raw), "이 줄은 본문이다") {
		t.Fatalf("★ 원장에 본문이 들어갔다 ★: %s — 목록이어야 한다", raw)
	}
	var es []struct {
		Name  string `json:"name"`
		By    string `json:"by"`
		Bytes int64  `json:"bytes"`
		RunID string `json:"run_id"`
	}
	_ = json.Unmarshal(raw, &es)
	if len(es) != 1 || es[0].Name != "note" {
		t.Fatalf("원장이 %v 다", string(raw))
	}
	if es[0].By != "step:note" {
		t.Fatalf("★ 누가 냈는지가 안 보인다 ★: %q", es[0].By)
	}
	if es[0].Bytes == 0 {
		t.Fatal("크기가 안 보인다 — 본문을 받을지 정할 재료가 없다")
	}
	if es[0].RunID != "" {
		t.Fatalf("★ 같은 Run 것에 run_id 가 붙었다 ★: %q — 모든 줄에 같은 값이 반복된다", es[0].RunID)
	}
}

// ★ 같은 Work 의 이전 Run 이 낸 것이 보인다 ★ (ADR-023 §6.5)
//
// "지난번에 이 지적을 했는데 안 고쳤다" 가 성립하는 자리다.
// patchset 2 와 3 은 ★ 같은 Work ★ 이므로 3 이 2 의 산출물을 발견한다.
func TestLedger_이전_패치셋이_낸_것이_보인다(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("L2", "a", map[string]string{"role": "x"}), nil)
	step := map[string]any{"id": "review", "uses": "b",
		"agent": map[string]any{"ask": "never"}, "out": []string{"review"}}

	// patchset 2 — 지난번 리뷰. 판정 조건이 없어 FAILED 로 끝나지만
	// ★ 낸 것은 남는다 ★ (원장은 성패와 무관하다).
	do(t, srv, "POST", "/v1/runs", workContract("ps2", "777", 2, "", []map[string]any{step}), nil)
	do(t, srv, "POST", "/v1/nodes/L2/claim", "", nil)
	do(t, srv, "PUT", "/v1/runs/ps2/steps/1/blob/review", `{"지적":"락을 안 풀었다"}`, nil)
	do(t, srv, "POST", "/v1/runs/ps2/steps/1/result", `{"node":"L2","produced":["review"]}`, nil)

	// patchset 3 — 이번 리뷰. ★ scope:"work" ★
	do(t, srv, "POST", "/v1/runs", workContract("ps3", "777", 3, "work", []map[string]any{step}), nil)

	code, v := do(t, srv, "GET", "/v1/runs/ps3/ledger", "", nil)
	if code != 200 {
		t.Fatalf("원장 조회 실패: %d", code)
	}
	raw, _ := json.Marshal(v["entries"])
	var es []struct {
		Name  string `json:"name"`
		RunID string `json:"run_id"`
	}
	_ = json.Unmarshal(raw, &es)
	if len(es) != 1 {
		t.Fatalf("★ 이전 패치셋이 낸 것이 안 보인다 ★: %s", raw)
	}
	if es[0].RunID != "ps2" {
		t.Fatalf("★ 어느 Run 에서 왔는지가 안 보인다 ★: %q", es[0].RunID)
	}

	// ★ scope 를 안 적으면 안 보인다 ★ — 기본은 오늘 동작이다.
	do(t, srv, "POST", "/v1/runs", workContract("ps4", "777", 4, "", []map[string]any{step}), nil)
	_, v4 := do(t, srv, "GET", "/v1/runs/ps4/ledger", "", nil)
	raw4, _ := json.Marshal(v4["entries"])
	var e4 []any
	_ = json.Unmarshal(raw4, &e4)
	if len(e4) != 0 {
		t.Fatalf("★ scope 없이 Run 을 넘어 봤다 ★: %s — 기본은 run 이어야 한다", raw4)
	}
}

// ★ 심는 것은 계약이 그러라고 할 때뿐이다 ★ (ADR-023 §6.4 자리 3)
func TestLedger_see가_있어야_목록이_실린다(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("L3", "a", map[string]string{"role": "x"}), nil)
	first := map[string]any{"id": "first", "uses": "b",
		"agent": map[string]any{"ask": "never"}, "out": []string{"first"}}
	quiet := map[string]any{"id": "quiet", "uses": "b", "run": []string{"true"}}
	seeing := map[string]any{"id": "seeing", "uses": "b", "run": []string{"true"},
		"see": map[string]any{"ledger": "list"}}
	do(t, srv, "POST", "/v1/runs",
		workContract("see1", "888", 1, "", []map[string]any{first, quiet, seeing}), nil)

	do(t, srv, "POST", "/v1/nodes/L3/claim", "", nil)
	do(t, srv, "PUT", "/v1/runs/see1/steps/1/blob/first", `{"a":1}`, nil)
	do(t, srv, "POST", "/v1/runs/see1/steps/1/result", `{"node":"L3","produced":["first"]}`, nil)

	// ★ 안 적은 단계에는 안 실린다 ★ = 오늘 그대로.
	_, c2 := do(t, srv, "POST", "/v1/nodes/L3/claim", "", nil)
	if _, got := c2["ledger"]; got {
		t.Fatalf("★ 안 적었는데 목록이 실렸다 ★: %v", c2["ledger"])
	}
	do(t, srv, "POST", "/v1/runs/see1/steps/2/result", `{"node":"L3","exit_code":0}`, nil)

	// ★ 적은 단계에는 실린다 ★
	_, c3 := do(t, srv, "POST", "/v1/nodes/L3/claim", "", nil)
	raw, _ := json.Marshal(c3["ledger"])
	if !strings.Contains(string(raw), `"first"`) {
		t.Fatalf("★ see:list 인데 목록이 안 실렸다 ★: %s", raw)
	}
}

// ★ 워터마크가 봉인에 남는다 ★ (ADR-023 §6.4 자리 2)
//
// 성질 4(자기충족)를 지키는 장치다 — 봉인된 묶음만 열어서 ★ "무엇을 볼 수
// 있었나" ★ 를 알 수 있어야 한다. ★ 안 깔린 것도 남는다 ★: 원장에 있었는데
// 이 단계가 안 가져간 것과 애초에 없었던 것은 다르다.
func TestLedger_워터마크가_봉인에_남는다(t *testing.T) {
	srv, st := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("L4", "a", map[string]string{"role": "x"}), nil)
	first := map[string]any{"id": "first", "uses": "b",
		"agent": map[string]any{"ask": "never"}, "out": []string{"first"}}
	later := map[string]any{"id": "later", "uses": "b", "run": []string{"true"}}
	c := map[string]any{
		"run_id":   "wm1",
		"work":     map[string]any{"system": "gerrit", "change_id": "999", "patchset": 1},
		"requires": []map[string]any{req("b", map[string]any{"role": "x"})},
		"steps":    []map[string]any{first, later},
		"success_when": []map[string]any{
			{"step": "first", "produced": []string{"first"}},
			{"step": "later", "exit_code": 0}},
	}
	b, _ := json.Marshal(c)
	if code, _ := do(t, srv, "POST", "/v1/runs", string(b), nil); code != 201 {
		t.Fatalf("제출 실패: %d", code)
	}
	do(t, srv, "POST", "/v1/nodes/L4/claim", "", nil)
	do(t, srv, "PUT", "/v1/runs/wm1/steps/1/blob/first", `{"a":1}`, nil)
	do(t, srv, "POST", "/v1/runs/wm1/steps/1/result", `{"node":"L4","produced":["first"]}`, nil)
	do(t, srv, "POST", "/v1/nodes/L4/claim", "", nil) // later — 이때 원장에 first 가 있다
	do(t, srv, "POST", "/v1/runs/wm1/steps/2/result", `{"node":"L4","exit_code":0}`, nil)

	raw, err := os.ReadFile(filepath.Join(st.Records.Root, "run-wm1", "steps", "02-later.json"))
	if err != nil {
		t.Fatalf("★ 봉인이 안 됐다 ★: %v", err)
	}
	var f struct {
		LedgerAt []string `json:"ledger_at"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	if len(f.LedgerAt) != 1 || !strings.HasSuffix(f.LedgerAt[0], "-first") {
		t.Fatalf("★ 볼 수 있었던 것이 안 남았다 ★: %v — "+
			"성질 4 는 봉인된 묶음만 보고 알 수 있기를 요구한다", f.LedgerAt)
	}

	// ★ 첫 단계는 볼 것이 없었다 ★ — 그것도 사실이고 그대로 남아야 한다.
	raw1, _ := os.ReadFile(filepath.Join(st.Records.Root, "run-wm1", "steps", "01-first.json"))
	var f1 struct {
		LedgerAt []string `json:"ledger_at"`
	}
	_ = json.Unmarshal(raw1, &f1)
	if len(f1.LedgerAt) != 0 {
		t.Fatalf("첫 단계가 무언가를 봤다: %v", f1.LedgerAt)
	}
}
