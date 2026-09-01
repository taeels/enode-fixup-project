package api_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/taeels/enode/internal/api"
	"github.com/taeels/enode/internal/config"
	"github.com/taeels/enode/internal/store"
)

// ── 오류 계약을 읽는 자리 ────────────────────────────────────────────────
//
// 경계에서 나가는 오류는 형태가 하나다 — fail(w, code, reason) 이 내는
// {"error":{"code":…,"reason":…}}. 반대편(runctl)이 그 형태를 되읽으므로
// 코드만 맞고 본문이 다르면 클라이언트에서 깨진다.

func reason(t *testing.T, body map[string]any) string {
	t.Helper()
	e, ok := body["error"].(map[string]any)
	if !ok {
		raw, _ := json.Marshal(body)
		t.Fatalf("the response is not shaped like an error: %s", raw)
	}
	r, _ := e["reason"].(string)
	return r
}

func assertFaultShape(t *testing.T, body map[string]any, wantCode int, wantReason string) {
	t.Helper()
	e, ok := body["error"].(map[string]any)
	if !ok {
		raw, _ := json.Marshal(body)
		t.Fatalf("the response is not shaped like an error: %s", raw)
	}
	c, ok := e["code"].(float64)
	if !ok {
		t.Fatalf("the error body carries no numeric code: %v", e)
	}
	if int(c) != wantCode {
		t.Errorf("error.code=%d, want %d — the body must agree with the status line", int(c), wantCode)
	}
	if r, _ := e["reason"].(string); r != wantReason {
		t.Errorf("error.reason=%q, want %q", r, wantReason)
	}
}

func readFile(p string) (string, error) {
	b, err := os.ReadFile(p)
	return string(b), err
}

// ── 광고 ─────────────────────────────────────────────────────────────────

// 못 읽는 광고는 400 이고, 무엇이 문제인지 파서의 말을 그대로 싣는다.
//
// 노드가 광고를 못 붙이면 함대에서 조용히 사라진다 — 그 원인이 응답에
// 없으면 사람이 Mediator 로그를 뒤져야 한다.
func TestAdvert_RefusesABodyItCannotRead(t *testing.T) {
	srv, _ := newServerFast(t)

	code, body := do(t, srv, "POST", "/v1/nodes", `{"node_id": `, nil)
	if code != 400 {
		t.Fatalf("code=%d, want 400", code)
	}
	if r := reason(t, body); !strings.Contains(r, "cannot parse advertisement") {
		t.Errorf("reason=%q does not say the advertisement could not be parsed", r)
	}
}

// node_id 가 없는 광고는 400 이고 그 필드를 이름으로 지적한다.
//
// 문법은 맞고 신원만 없는 경우다. "cannot parse" 로 뭉뚱그리면 보내는
// 쪽이 JSON 을 들여다보게 된다.
func TestAdvert_RefusesAnAdvertWithNoNodeID(t *testing.T) {
	srv, _ := newServerFast(t)

	code, body := do(t, srv, "POST", "/v1/nodes",
		`{"label":"box","capabilities":[{"capability":"agent.reason"}]}`, nil)
	if code != 400 {
		t.Fatalf("code=%d, want 400", code)
	}
	assertFaultShape(t, body, 400, "node_id is missing")
}

// ── 제출 ─────────────────────────────────────────────────────────────────

// 못 읽는 계약은 400 이다 — Validate 가 도는 자리까지 가지 않는다.
func TestSubmit_RefusesABodyItCannotRead(t *testing.T) {
	srv, _ := newServerFast(t)

	for _, path := range []string{"/v1/runs", "/v1/runs/dry-run"} {
		code, body := do(t, srv, "POST", path, `{"run_id": "r", `, nil)
		if code != 400 {
			t.Errorf("%s: code=%d, want 400", path, code)
		}
		if r := reason(t, body); !strings.Contains(r, "cannot parse contract") {
			t.Errorf("%s: reason=%q does not say the contract could not be parsed", path, r)
		}
	}
}

// ── 결과 보고 ────────────────────────────────────────────────────────────

// 단계 번호가 번호가 아니면 400 이고, 받은 값을 그대로 되비춘다.
func TestResult_RefusesASequenceThatIsNotAStepNumber(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "box", map[string]string{"role": "x"}), nil)
	do(t, srv, "POST", "/v1/runs", oneStepRun("rseq", "n1"), nil)

	for _, seq := range []string{"abc", "0", "-3"} {
		code, body := do(t, srv, "POST", "/v1/runs/rseq/steps/"+seq+"/result",
			`{"node":"n1","exit_code":0}`, nil)
		if code != 400 {
			t.Errorf("seq=%q: code=%d, want 400", seq, code)
		}
		r := reason(t, body)
		if !strings.Contains(r, "sequence") {
			t.Errorf("seq=%q: reason=%q does not say the sequence is the problem", seq, r)
		}
		if !strings.Contains(r, seq) {
			t.Errorf("seq=%q: reason=%q does not echo the value it refused", seq, r)
		}
	}
}

// 못 읽는 결과 본문은 400 이다 — 완주 여부를 추측하지 않는다.
//
// 여기서 조용히 "완주 못 함" 으로 치면 그 판정이 success_when 이 아니라
// 파서에서 나온 것이 되고, ADR-004 가 무너진다.
func TestResult_RefusesABodyItCannotRead(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "box", map[string]string{"role": "x"}), nil)
	do(t, srv, "POST", "/v1/runs", oneStepRun("rbody", "n1"), nil)
	do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)

	code, body := do(t, srv, "POST", "/v1/runs/rbody/steps/1/result", `{"exit_code":`, nil)
	if code != 400 {
		t.Fatalf("code=%d, want 400", code)
	}
	if r := reason(t, body); !strings.Contains(r, "cannot parse result") {
		t.Errorf("reason=%q does not say the result could not be parsed", r)
	}
	// 그리고 단계는 그대로 살아 있다 — 못 읽은 보고가 단계를 죽이지 않는다.
	_, run := do(t, srv, "GET", "/v1/runs/rbody", "", nil)
	if run["state"] != "RUNNING" {
		t.Fatalf("state=%v, want RUNNING — an unreadable report changed the run", run["state"])
	}
}

// 집히지도 않은 단계의 보고는 409 다 — 그 자리에 쓸 것이 없다.
func TestResult_AStepThatWasNeverClaimedIs409(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "box", map[string]string{"role": "x"}), nil)
	do(t, srv, "POST", "/v1/runs", oneStepRun("unclaimed", "n1"), nil)

	// 1단계를 집지 않은 채 2단계 결과를 보낸다.
	code, body := do(t, srv, "POST", "/v1/runs/unclaimed/steps/2/result",
		`{"node":"n1","exit_code":0}`, nil)
	if code != 409 {
		t.Fatalf("code=%d, want 409", code)
	}
	if r := reason(t, body); r == "" {
		t.Error("a 409 came back with no reason at all")
	}
	_, run := do(t, srv, "GET", "/v1/runs/unclaimed", "", nil)
	if run["state"] != "RUNNING" {
		t.Fatalf("state=%v, want RUNNING — a rejected report settled the run", run["state"])
	}
}

// ── 원장 ─────────────────────────────────────────────────────────────────

// 없는 Run 의 원장은 404 다 — 빈 목록이 아니다.
//
// 빈 목록으로 답하면 "아직 아무것도 안 냈다" 와 "그런 Run 이 없다" 가
// 구별되지 않고, 껍데기가 재시도를 못 정한다.
func TestLedger_UnknownRunIs404NotAnEmptyList(t *testing.T) {
	srv, _ := newServerFast(t)

	code, body := do(t, srv, "GET", "/v1/runs/never-existed/ledger", "", nil)
	if code != 404 {
		t.Fatalf("code=%d, want 404", code)
	}
	assertFaultShape(t, body, 404, "no such run")
	if _, ok := body["entries"]; ok {
		t.Error("a 404 still carried an entries list")
	}
}

// ── 기록 ─────────────────────────────────────────────────────────────────

// 없는 Run 의 기록은 404 다 — 봉인 여부를 묻기 전에 존재부터 본다.
func TestRecord_UnknownRunIs404(t *testing.T) {
	srv, _ := newServerFast(t)

	code, body := do(t, srv, "GET", "/v1/runs/never-existed/record", "", nil)
	if code != 404 {
		t.Fatalf("code=%d, want 404", code)
	}
	assertFaultShape(t, body, 404, "no such run")
}

// ── 산출물 ───────────────────────────────────────────────────────────────

// blob 도 단계 번호를 검사한다 — log 와 같은 규칙이다.
func TestBlob_RefusesASequenceThatIsNotAStepNumber(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "box", map[string]string{"role": "x"}), nil)
	do(t, srv, "POST", "/v1/runs", oneStepRun("bseq", "n1"), nil)

	for _, seq := range []string{"abc", "0"} {
		code, body := do(t, srv, "PUT", "/v1/runs/bseq/steps/"+seq+"/blob/out", "x", nil)
		if code != 400 {
			t.Errorf("seq=%q: code=%d, want 400", seq, code)
		}
		if r := reason(t, body); !strings.Contains(r, "sequence") {
			t.Errorf("seq=%q: reason=%q does not say the sequence is the problem", seq, r)
		}
	}
}

// 없는 Run 에 올리면 404 이고, 있는 Run 의 없는 단계면 404 다.
//
// 둘을 같은 404 로 두되 사유를 다르게 말한다 — 생산자가 run_id 를 틀린
// 것과 seq 를 틀린 것은 고치는 자리가 다르다.
func TestBlob_UnknownRunAndUnknownStepAreBoth404WithDifferentReasons(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "box", map[string]string{"role": "x"}), nil)
	do(t, srv, "POST", "/v1/runs", oneStepRun("bnf", "n1"), nil)

	code, body := do(t, srv, "PUT", "/v1/runs/nope/steps/1/blob/out", "x", nil)
	if code != 404 {
		t.Fatalf("unknown run: code=%d, want 404", code)
	}
	assertFaultShape(t, body, 404, "no such run")

	// 계약에 없는 단계 번호 — Run 은 있다.
	code, body = do(t, srv, "PUT", "/v1/runs/bnf/steps/99/blob/out", "x", nil)
	if code != 404 {
		t.Fatalf("unknown step: code=%d, want 404", code)
	}
	assertFaultShape(t, body, 404, "no such step")
}

// 상한을 넘는 산출물은 **거절**한다 — 잘라 저장하지 않는다.
//
// "잘린 산출물은 산출물이 아니다". 스키마가 걸린 자리와 안 걸린 자리
// 둘 다 같은 판단이어야 한다 — 한쪽만 자르면 produced 의 뜻이 갈린다.
func TestBlob_OversizeIsRefusedOnBothPaths(t *testing.T) {
	const limit = 32
	srv, _ := newServerFast(t, func(c *config.Config) { c.Artifacts.MaxBlobBytes = limit })
	do(t, srv, "POST", "/v1/nodes", advert("n1", "box", map[string]string{"role": "x"}), nil)
	if code, _ := do(t, srv, "POST", "/v1/runs", schemaRun("bigb", okSchema), nil); code != 201 {
		t.Fatalf("submit failed: %d", code)
	}
	do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)

	// 상한을 넘되 그 자체로는 스키마를 만족하는 본문 — 크기로 걸린 것임을 못박는다.
	big := `{"status":"found","reason":"` + strings.Repeat("y", limit*3) + `"}`

	// 스키마가 걸린 이름(hypothesis) — 검증하려면 먼저 읽어야 하는 길.
	code, body := do(t, srv, "PUT", "/v1/runs/bigb/steps/1/blob/hypothesis", big, nil)
	if code != 413 {
		t.Fatalf("schema path: code=%d, want 413", code)
	}
	if r := reason(t, body); !strings.Contains(r, "size limit") {
		t.Errorf("schema path: reason=%q does not say it is a size limit", r)
	}

	// 스키마가 안 걸린 이름 — 곧장 흘려 쓰는 길.
	code, body = do(t, srv, "PUT", "/v1/runs/bigb/steps/1/blob/unschemad", big, nil)
	if code != 413 {
		t.Fatalf("streaming path: code=%d, want 413", code)
	}
	if r := reason(t, body); !strings.Contains(r, "size limit") {
		t.Errorf("streaming path: reason=%q does not say it is a size limit", r)
	}

	// 상한 안쪽이면 두 길 다 지나간다 — 상한이 전부를 막는 것이 아니다.
	if code, _ := do(t, srv, "PUT", "/v1/runs/bigb/steps/1/blob/hypothesis",
		`{"status":"none"}`, nil); code != 204 {
		t.Fatalf("schema path: a blob under the limit was refused: %d", code)
	}
	if code, _ := do(t, srv, "PUT", "/v1/runs/bigb/steps/1/blob/unschemad", "small", nil); code != 204 {
		t.Fatalf("streaming path: a blob under the limit was refused: %d", code)
	}
}

// ── 되묻기 ───────────────────────────────────────────────────────────────

// 답의 단계 번호도 검사한다.
func TestAnswer_RefusesASequenceThatIsNotAStepNumber(t *testing.T) {
	srv, _ := newServerFast(t)

	for _, seq := range []string{"abc", "0"} {
		code, body := do(t, srv, "POST", "/v1/runs/any/steps/"+seq+"/answer", `{"verdict":"approve"}`, nil)
		if code != 400 {
			t.Errorf("seq=%q: code=%d, want 400", seq, code)
		}
		if r := reason(t, body); !strings.Contains(r, "sequence") {
			t.Errorf("seq=%q: reason=%q does not say the sequence is the problem", seq, r)
		}
	}
}

// 상한을 넘는 답은 413 이다 — 잘라서 저장하면 그 답이 무엇이었는지 잃는다.
func TestAnswer_OversizeIsRefused(t *testing.T) {
	const limit = 16
	srv, _ := newServerFast(t, func(c *config.Config) { c.Artifacts.MaxBlobBytes = limit })

	code, body := do(t, srv, "POST", "/v1/runs/any/steps/1/answer",
		`{"verdict":"approve","note":"`+strings.Repeat("z", limit*4)+`"}`, nil)
	if code != 413 {
		t.Fatalf("code=%d, want 413", code)
	}
	if r := reason(t, body); !strings.Contains(r, "size limit") {
		t.Errorf("reason=%q does not say it is a size limit", r)
	}
}

// can_answer 는 **보는 사람 기준**이다 — 목록 안의 사람에게는 참이다.
//
// 인박스는 모두가 같은 것을 보지만 이 필드만 관점이다. 서버가 안 채우면
// 껍데기가 answerers 목록을 스스로 해석하게 되고, 그 해석이 서버의
// 403 판정과 어긋날 수 있다.
func TestInbox_CanAnswerIsTrueForSomeoneOnTheList(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("q1", "a", map[string]string{"role": "x"}), nil)
	c := map[string]any{
		"run_id":   "inbox1",
		"requires": []map[string]any{req("b", map[string]any{"role": "x"})},
		"steps": []map[string]any{
			runStep("prep", "b"),
			askStep("gate", []string{"boss@corp", "lead@corp"}, nil),
			runStep("approve", "b"), runStep("reject", "b"),
		},
		"success_when": []map[string]any{{"step": "prep", "exit_code": 0}},
	}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if code, _ := do(t, srv, "POST", "/v1/runs", string(b), nil); code != 201 {
		t.Fatalf("submit failed: %d", code)
	}
	do(t, srv, "POST", "/v1/nodes/q1/claim", "", nil)
	do(t, srv, "POST", "/v1/runs/inbox1/steps/1/result", `{"node":"q1","exit_code":0}`, nil)

	canAnswer := func(principal string) bool {
		t.Helper()
		hdr := map[string]string{"X-Enode-Principal": principal}
		_, inbox := do(t, srv, "GET", "/v1/asks", "", hdr)
		raw, err := json.Marshal(inbox["asks"])
		if err != nil {
			t.Fatal(err)
		}
		var asks []struct {
			CanAnswer bool `json:"can_answer"`
		}
		if err := json.Unmarshal(raw, &asks); err != nil {
			t.Fatal(err)
		}
		if len(asks) != 1 {
			t.Fatalf("%s: the inbox holds %d questions, want exactly 1: %s", principal, len(asks), raw)
		}
		return asks[0].CanAnswer
	}

	for _, who := range []string{"boss@corp", "lead@corp"} {
		if !canAnswer(who) {
			t.Errorf("can_answer=false for %s, who is on the answerers list", who)
		}
	}
	if canAnswer("passer-by@corp") {
		t.Error("can_answer=true for someone outside the answerers list")
	}
	// 그리고 그 필드가 서버의 판정과 같은 말을 한다.
	if code, _ := do(t, srv, "POST", "/v1/runs/inbox1/steps/2/answer",
		`{"verdict":"approve"}`, map[string]string{"X-Enode-Principal": "passer-by@corp"}); code != 403 {
		t.Errorf("code=%d, want 403 — can_answer and the enforcement disagree", code)
	}
}

// ── Record 저장소가 없는 Mediator ────────────────────────────────────────

// Record 저장소 없이 뜬 Mediator 는 기록에 닿는 모든 표면에서 503 이다.
//
// 패닉이 아니다 — cmd/mediator 는 항상 붙이지만, 붙이는 것을 잊은 배치가
// 500 이나 크래시로 나가면 원인이 응답에 안 남는다.
func TestSurface_WithoutARecordStoreEveryRecordPathIs503(t *testing.T) {
	cfg := config.Default()
	cfg.Token = token
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	// Records 를 안 붙인 저장소. 여기 걸리는 표면은 DB 에 닿기 전에 멈춘다.
	srv := httptest.NewServer(api.New(&store.Store{}, cfg, log).Handler())
	t.Cleanup(srv.Close)

	cases := []struct {
		method, path string
	}{
		{"PUT", "/v1/runs/r/steps/1/log"},
		{"PUT", "/v1/runs/r/steps/1/blob/out"},
		{"GET", "/v1/runs/r/blob/out"},
		{"POST", "/v1/runs/r/steps/1/answer"},
		{"GET", "/v1/runs/r/record"},
	}
	for _, c := range cases {
		code, body := do(t, srv, c.method, c.path, "{}", nil)
		if code != 503 {
			t.Errorf("%s %s: code=%d, want 503", c.method, c.path, code)
			continue
		}
		assertFaultShape(t, body, 503, "record store is not configured")
	}
}

// ── 롱폴 ─────────────────────────────────────────────────────────────────

// 할 일이 없으면 시간이 다 될 때까지 기다렸다가 204 로 답한다.
//
// 즉시 204 로 답하면 enode 가 바로 다시 걸어 폴링이 된다. 기다리는 것이
// 이 표면의 일이다 (ADR-015 §5).
func TestClaim_WaitsOutTheWindowThenAnswers204(t *testing.T) {
	srv, _ := newServerFast(t, func(c *config.Config) { c.Claim.LongPollSeconds = 2 })
	do(t, srv, "POST", "/v1/nodes", advert("idle", "a", map[string]string{"role": "x"}), nil)

	start := time.Now()
	code, _ := do(t, srv, "POST", "/v1/nodes/idle/claim", "", nil)
	waited := time.Since(start)

	if code != 204 {
		t.Fatalf("code=%d, want 204 — no work means no content", code)
	}
	if waited < time.Second {
		t.Fatalf("the long poll returned after %v; it did not wait out its window", waited)
	}
}

// 클라이언트가 끊는 것은 정상이다 — 오류로 세지 않는다 (ADR-015 §5).
//
// enode 가 죽거나 네트워크가 끊기면 롱폴이 매번 이 자리로 온다. 여기서
// 오류를 내면 Mediator 로그가 정상 운영에서 오류로 가득 찬다.
func TestClaim_AClientThatHangsUpIsNotAnError(t *testing.T) {
	srv, _ := newServerFast(t, func(c *config.Config) { c.Claim.LongPollSeconds = 30 })
	do(t, srv, "POST", "/v1/nodes", advert("goner", "a", map[string]string{"role": "x"}), nil)

	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, "POST", srv.URL+"/v1/nodes/goner/claim", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	done := make(chan error, 1)
	go func() {
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			resp.Body.Close()
		}
		done <- err
	}()

	// 롱폴이 실제로 대기에 들어간 뒤에 끊는다.
	time.Sleep(1500 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("the cancelled request came back without an error on the client side")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the server held the connection after the client hung up")
	}

	// 서버는 멀쩡하다 — 끊긴 폴 하나가 표면을 죽이지 않는다.
	if code, _ := do(t, srv, "POST", "/v1/nodes",
		advert("goner", "a", map[string]string{"role": "x"}), nil); code != 200 {
		t.Fatalf("the server stopped answering after a client hung up: %d", code)
	}
}
