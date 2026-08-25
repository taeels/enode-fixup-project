package enode

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// 첫 광고가 성공하면 한 번 알린다 (docs/elastic-nodes.md §4.4)
//
// 노드를 띄운 쪽은 「떴다」와 「쓸 수 있다」 사이를 건너야 한다.
// 프로세스가 뜬 것과 함대에 등록된 것은 다르고, 그 사이에 Run 을 내면
// 매칭이 422 로 거절한다.
func Test첫_광고가_성공하면_알린다(t *testing.T) {
	srv, calls := fakeAdvertServer(t)
	defer srv.Close()

	var mu sync.Mutex
	var ready int
	a := &Advertiser{
		Client:  &Client{Base: srv.URL, Token: "t", Principal: "p", HTTP: srv.Client()},
		Ident:   Identity{NodeID: "n1", Label: "l"},
		Local:   Local{},
		Every:   20 * time.Millisecond,
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		OnReady: func() { mu.Lock(); ready++; mu.Unlock() },
	}
	// 시간이 아니라 횟수로 끝낸다 — 타이밍에 기대면 시험이 흔들린다.
	// 광고가 세 번 돌면 ctx 를 끝낸다. 그래야 「한 번만」을 잴 수 있다.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		for calls() < 3 {
			time.Sleep(5 * time.Millisecond)
		}
		cancel()
	}()
	a.Run(ctx)

	mu.Lock()
	defer mu.Unlock()
	if ready == 0 {
		t.Fatal("광고가 성공했는데 안 알렸다")
	}
	// 한 번만 — 광고는 주기마다 도는데 ready 는 「떴다」이지 「건강하다」가 아니다.
	if ready != 1 {
		t.Fatalf("%d 번 알렸다 — 한 번이어야 한다 (광고는 %d 번 돌았다)",
			ready, calls())
	}
}

// 광고가 실패하면 안 알린다 — 안 뜬 것을 떴다고 하면 그 다음이 422 다.
func Test광고가_실패하면_안_알린다(t *testing.T) {
	srv := failingAdvertServer(t)
	defer srv.Close()

	var mu sync.Mutex
	var ready int
	a := &Advertiser{
		Client:  &Client{Base: srv.URL, Token: "t", Principal: "p", HTTP: srv.Client()},
		Ident:   Identity{NodeID: "n1"},
		Every:   20 * time.Millisecond,
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		OnReady: func() { mu.Lock(); ready++; mu.Unlock() },
	}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	a.Run(ctx)

	mu.Lock()
	defer mu.Unlock()
	if ready != 0 {
		t.Fatalf("광고가 실패했는데 %d 번 알렸다", ready)
	}
}

// ── 시험용 서버 둘 ──────────────────────────────────────────────

func fakeAdvertServer(t *testing.T) (*httptest.Server, func() int) {
	t.Helper()
	var mu sync.Mutex
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		n++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"renew_seconds":0,"leases":[]}`)
	}))
	return srv, func() int { mu.Lock(); defer mu.Unlock(); return n }
}

func failingAdvertServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
}
