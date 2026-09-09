package enode

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/taeels/enode/internal/contract"
)

// drain (ADR-063) — 광고가 정책을 나르고, 응답이 Worker 의 태도를 바꾼다.

// echoAdvertServer 는 광고 본문의 policy.drain 을 기억하고 그대로 되돌려 준다.
func echoAdvertServer(t *testing.T) (*httptest.Server, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ad contract.Advert
		if err := json.NewDecoder(r.Body).Decode(&ad); err != nil {
			t.Errorf("cannot decode the advert: %v", err)
		}
		mu.Lock()
		seen = append(seen, ad.Policy.Drain)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"leases": []any{}, "renew_seconds": 0, "drain": ad.Policy.Drain,
		})
	}))
	return srv, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), seen...)
	}
}

// 정책 파일이 광고에 실리고, 응답이 Held 에 앉는다. 지우면 다음 광고가 해제를 나른다.
func TestDrain_PolicyFileRidesTheAdvertAndTheAckLandsInHeld(t *testing.T) {
	srv, seen := echoAdvertServer(t)
	defer srv.Close()
	cfg := filepath.Join(t.TempDir(), "local.yaml")
	policy := PolicyPath(cfg)
	if err := os.WriteFile(policy, []byte("drain: at-boundary\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	held := NewHeld()
	a := &Advertiser{
		Client: &Client{Base: srv.URL, Token: "t", Principal: "p", HTTP: srv.Client()},
		Ident:  Identity{NodeID: "n1", Label: "l", Config: cfg},
		Caps:   func() Capabilities { return Capabilities{} },
		Every:  10 * time.Millisecond,
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		Held:   held,
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for len(seen()) < 2 {
			time.Sleep(2 * time.Millisecond)
		}
		if err := os.Remove(policy); err != nil {
			t.Errorf("cannot remove the policy: %v", err)
		}
		for {
			s := seen()
			if len(s) > 0 && s[len(s)-1] == contract.DrainNone && held.Drain() == contract.DrainNone {
				break
			}
			time.Sleep(2 * time.Millisecond)
		}
		cancel()
	}()
	a.Run(ctx)

	s := seen()
	if s[0] != contract.DrainAtBoundary {
		t.Fatalf("the first advert did not carry the policy: %v", s)
	}
	if s[len(s)-1] != contract.DrainNone {
		t.Fatalf("removing the file did not release: %v", s)
	}
	if held.Renew() != 10*time.Millisecond {
		t.Fatalf("the advert interval did not reach Held: %v", held.Renew())
	}
}

// 설정 경로가 없으면 정책을 안 읽고, Held 가 없으면 안 나른다 — 오늘 그대로.
func TestDrain_NoConfigNoPolicyNoHeldIsToday(t *testing.T) {
	srv, seen := echoAdvertServer(t)
	defer srv.Close()
	a := &Advertiser{
		Client: &Client{Base: srv.URL, Token: "t", Principal: "p", HTTP: srv.Client()},
		Ident:  Identity{NodeID: "n1", Label: "l"},
		Caps:   func() Capabilities { return Capabilities{} },
		Every:  5 * time.Millisecond,
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for len(seen()) < 2 {
			time.Sleep(2 * time.Millisecond)
		}
		cancel()
	}()
	a.Run(ctx)
	for _, d := range seen() {
		if d != contract.DrainNone {
			t.Fatalf("an advert without a config carried a policy: %q", d)
		}
	}
}

// claimCounter 는 claim 을 세고 204 로 답한다 — 할 일이 없다.
func claimCounter(t *testing.T) (*httptest.Server, func() int) {
	t.Helper()
	var mu sync.Mutex
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		n++
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	return srv, func() int { mu.Lock(); defer mu.Unlock(); return n }
}

// at-boundary 를 안 순간부터 Worker 는 claim 을 안 건다. graceful 은 건다.
func TestDrain_WorkerDoesNotClaimWhileAtBoundary(t *testing.T) {
	srv, claims := claimCounter(t)
	defer srv.Close()
	held := NewHeld()
	held.SetDrain(contract.DrainAtBoundary)
	held.SetRenew(5 * time.Millisecond)
	w := &Worker{
		Client: &Client{Base: srv.URL, Token: "t", Principal: "p", HTTP: srv.Client(), Poll: srv.Client()},
		Ident:  Identity{NodeID: "n1"},
		Held:   held,
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	w.Run(ctx)
	if n := claims(); n != 0 {
		t.Fatalf("a draining node claimed %d times, want 0", n)
	}

	held.SetDrain(contract.DrainGraceful)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel2()
	w.Run(ctx2)
	if n := claims(); n == 0 {
		t.Fatal("a graceful node did not claim — graceful only blocks new leases")
	}
}
