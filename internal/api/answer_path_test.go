package api_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/taeels/enode/internal/config"
	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/store"
)

// 답변 지점의 경로는 주인이 하나다 (FR3.2).
//
// 정본은 Mediator 표면이 하나라고 말하는데, 라우트 문자열은 다섯 벌로
// 흩어져 있다. 그중 internal/store/ask.go 는 성격이 다르다 — 나머지 넷은
// HTTP 를 아는 쪽(클라이언트와 어댑터)이지만 이쪽은 상태 층이고,
// team-practices.md 가 「상태 층은 HTTP 를 모른다」의 유일한 예외로
// 기록한 자리가 실은 둘이었다(아웃바운드 웹훅 + 이 라우트 문자열).
//
// 상태 층이 라우트를 스스로 지으면 등록과 발행이 두 곳에서 따로 지어져
// 한쪽만 바뀌어도 아무도 모른다. 경로를 아는 것은 HTTP 표면이고,
// 상태 층은 그것을 건네받는다.
func TestAnswerPath_TheRouteHasOneOwner(t *testing.T) {
	// newServerFast 가 DB 환경변수의 부재를 이미 걸러 준다 —
	// 이 파일은 새 스킵을 만들지 않는다.
	srv, _ := newServerFast(t, func(c *config.Config) { c.Notify.AsksURL = wiredHook(t) })

	t.Run("the state layer does not invent a route", func(t *testing.T) {
		got := make(chan []byte, 4)
		hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			got <- b
			w.WriteHeader(200)
		}))
		defer hook.Close()

		// HTTP 표면을 거치지 않은 상태 층이다 — api.New 를 안 부른다.
		ctx := context.Background()
		raw, err := store.Open(ctx, os.Getenv("ENODE_TEST_DATABASE_URL"))
		if err != nil {
			t.Fatal(err)
		}
		defer raw.Close()
		raw.NotifyURL = hook.URL

		var c contract.Contract
		if err := json.Unmarshal([]byte(askOnlyContract("bare1")), &c); err != nil {
			t.Fatal(err)
		}
		if err := raw.CreateRun(ctx,
			store.Run{RunID: c.RunID, State: store.StateRunning, Contract: c},
			nil, c.Steps); err != nil {
			t.Fatal(err)
		}

		select {
		case b := <-got:
			var e struct {
				RunID      string `json:"run_id"`
				AnswerPath string `json:"answer_path"`
			}
			if err := json.Unmarshal(b, &e); err != nil || e.RunID != "bare1" {
				t.Fatalf("the webhook body is wrong: %s", b)
			}
			if e.AnswerPath != "" {
				t.Fatalf("the state layer built an HTTP route on its own: answer_path=%q - the route belongs to the HTTP surface",
					e.AnswerPath)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("the webhook never rang")
		}
	})

	t.Run("the route it hands down is one it serves", func(t *testing.T) {
		do(t, srv, "POST", "/v1/nodes", advert("p9", "a", map[string]string{"role": "x"}), nil)
		if code, _ := do(t, srv, "POST", "/v1/runs", askOnlyContract("wired1"), nil); code != 201 {
			t.Fatalf("submit failed: %d", code)
		}
		select {
		case b := <-wiredAsks:
			var e struct {
				RunID      string `json:"run_id"`
				AnswerPath string `json:"answer_path"`
			}
			if err := json.Unmarshal(b, &e); err != nil || e.RunID != "wired1" {
				t.Fatalf("the webhook body is wrong: %s", b)
			}
			if e.AnswerPath == "" {
				t.Fatal("the HTTP surface handed down no answer_path - the direct link is gone")
			}
			// 404 가 아니면 mux 가 그 경로를 실제로 태운 것이다.
			// 본문이 비어 400 이 정상이고, 우리가 보는 것은 라우팅이다.
			code, _ := do(t, srv, "POST", e.AnswerPath, "{}", nil)
			if code == 404 {
				t.Fatalf("the answer_path the Mediator published is not a route it serves: %s", e.AnswerPath)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("the webhook never rang")
		}
	})
}

// wiredAsks 는 서버가 쏜 알림이 모이는 자리다.
var wiredAsks = make(chan []byte, 4)

func wiredHook(t *testing.T) string {
	t.Helper()
	h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		wiredAsks <- b
		w.WriteHeader(200)
	}))
	t.Cleanup(h.Close)
	return h.URL
}

// askOnlyContract 는 첫 단계가 되묻기인 계약이다 — 제출 즉시 질문이 오른다.
func askOnlyContract(runID string) string {
	gate := map[string]any{
		"id":  "gate",
		"ask": map[string]any{"prompt": "shall we start?"},
		"out": []string{"decision"},
		"schema": map[string]any{"decision": map[string]any{
			"type": "object", "required": []string{"verdict"},
			"properties": map[string]any{
				"verdict": map[string]any{"enum": []string{"approve", "reject"}}}}}}
	return contractJSON(runID, []map[string]any{req("b", map[string]any{"role": "x"})},
		[]map[string]any{gate, runStep("work", "b")})
}
