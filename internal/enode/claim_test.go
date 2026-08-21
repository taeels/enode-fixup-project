package enode

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// ★ 보고는 닿을 때까지 다시 보낸다 ★ (ADR-030)
//
// 완주한 단계의 보고가 유실된 채 다음 claim 을 걸면, 같은 생 재전달이 그
// 단계를 돌려줘 ★ 완주한 단계가 두 번 돈다 ★. 재시도가 그 문을 닫는다.
func TestReport_닿을_때까지_다시_보낸다(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(500) // ★ 첫 번째는 유실(서버 오류)로 친다 ★
			return
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	held := NewHeld()
	held.Add(Lease{RunID: "r1", NotAfter: time.Now().Add(time.Hour)})
	w := &Worker{
		Client:        &Client{Base: srv.URL, HTTP: srv.Client()},
		Held:          held,
		Log:           slog.New(slog.NewTextHandler(io.Discard, nil)),
		ReportBackoff: time.Millisecond,
	}
	w.report(context.Background(), &Step{RunID: "r1", Seq: 1, StepID: "r1#01"}, Result{})
	if got := calls.Load(); got != 2 {
		t.Fatalf("★ 재시도가 안 됐다 ★: %d 번 호출 — 유실된 보고는 영영 유실된다", got)
	}
}

// ★ 거절(4xx)은 유실이 아니다 ★ — 다시 보내도 같은 답이므로 재시도하지 않는다.
func TestReport_거절은_재시도하지_않는다(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(409)
	}))
	defer srv.Close()

	held := NewHeld()
	held.Add(Lease{RunID: "r1", NotAfter: time.Now().Add(time.Hour)})
	w := &Worker{
		Client:        &Client{Base: srv.URL, HTTP: srv.Client()},
		Held:          held,
		Log:           slog.New(slog.NewTextHandler(io.Discard, nil)),
		ReportBackoff: time.Millisecond,
	}
	w.report(context.Background(), &Step{RunID: "r1", Seq: 1, StepID: "r1#01"}, Result{})
	if got := calls.Load(); got != 1 {
		t.Fatalf("★ 거절을 재시도했다 ★: %d 번 호출", got)
	}
}

// ★ 임대가 끝나면 물러선다 ★ — 그때는 회수가 Run 째로 정리한다 (시간이 감시자다).
func TestReport_임대가_끝나면_물러선다(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(500)
	}))
	defer srv.Close()

	w := &Worker{
		Client:        &Client{Base: srv.URL, HTTP: srv.Client()},
		Held:          NewHeld(), // ★ 임대가 아예 없다 ★
		Log:           slog.New(slog.NewTextHandler(io.Discard, nil)),
		ReportBackoff: time.Millisecond,
	}
	w.report(context.Background(), &Step{RunID: "r1", Seq: 1, StepID: "r1#01"}, Result{})
	if got := calls.Load(); got != 1 {
		t.Fatalf("임대 없이 %d 번 던졌다", got)
	}
}
