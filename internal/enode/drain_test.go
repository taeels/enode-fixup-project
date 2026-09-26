package enode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
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

// ── 여유 부족 drain (trash 유닛 · business-rules.md 6절) ───────────────────

const gib = uint64(1) << 30

// lockedBuffer 는 여러 고루틴이 쓰는 노드 로그를 모은다.
type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

// 걸고 푸는 선이 다르다 — 걸 때 < min · 풀 때 >= min + 1 GB (6.2 의 네 줄).
func TestDiskDrainHoldsUntilOneGBAboveTheMinimum(t *testing.T) {
	cases := []struct {
		name   string
		free   uint64
		held   bool
		want   bool
		detail string
	}{
		{"not held · below", 7 * gib, false, true, "free 7 GB < min 10 GB"},
		{"not held · at the minimum", 10 * gib, false, false, ""},
		{"held · still below the lifting line", 10*gib + gib/2, true, true, "free 10 GB < min 10 GB + 1 GB to lift"},
		{"held · below the minimum", 3 * gib, true, true, "free 3 GB < min 10 GB"},
		{"held · at the lifting line", 11 * gib, true, false, ""},
	}
	for _, c := range cases {
		src, ok := diskDrain(c.free, 10, c.held)
		if ok != c.want || src.Detail != c.detail {
			t.Errorf("%s: ok=%v detail=%q, want %v %q", c.name, ok, src.Detail, c.want, c.detail)
		}
		if ok && (src.Kind != DrainDisk || src.Mode != contract.DrainGraceful || src.Owner) {
			t.Errorf("%s: source = %+v", c.name, src)
		}
	}
	if _, ok := diskDrain(0, 0, true); ok {
		t.Error("a zero min_free_gb drained the node")
	}
}

// 합치기는 센 쪽 하나다 (6.3 의 표).
func TestCombineDrainTakesTheStrongest(t *testing.T) {
	owner := func(mode string) DrainSource { s, _ := OwnerDrain(mode); return s }
	disk, _ := diskDrain(0, 10, false)
	cases := []struct {
		name    string
		sources []DrainSource
		want    string
	}{
		{"none", nil, contract.DrainNone},
		{"disk only", []DrainSource{disk}, contract.DrainGraceful},
		{"owner graceful and disk", []DrainSource{owner(contract.DrainGraceful), disk}, contract.DrainGraceful},
		{"owner at-boundary and disk", []DrainSource{owner(contract.DrainAtBoundary), disk}, contract.DrainAtBoundary},
		{"owner graceful only", []DrainSource{owner(contract.DrainGraceful)}, contract.DrainGraceful},
		{"owner at-boundary only", []DrainSource{owner(contract.DrainAtBoundary)}, contract.DrainAtBoundary},
	}
	for _, c := range cases {
		if got := combineDrain(c.sources); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
	if _, ok := OwnerDrain(contract.DrainNone); ok {
		t.Error("an empty policy made an owner source")
	}
	if s := owner(contract.DrainGraceful); s.Kind != DrainOwner || !s.Owner || s.Detail != "set in the policy file" {
		t.Errorf("owner source = %+v", s)
	}
}

// fakeFree 는 광고마다 여유를 하나씩 돌려준다. 끝에 닿으면 마지막 값을 되풀이한다.
type fakeFree struct {
	mu    sync.Mutex
	vals  []uint64
	errs  []error
	calls int
}

func (f *fakeFree) free(string) (uint64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	i := min(f.calls, len(f.vals)-1)
	f.calls++
	var err error
	if i < len(f.errs) {
		err = f.errs[i]
	}
	return f.vals[i], err
}

func (f *fakeFree) count() int { f.mu.Lock(); defer f.mu.Unlock(); return f.calls }

// 광고 n 번을 돌린다 (그리고 until 이 참일 때까지). 본문의 policy.drain 을 앞에서 n 개 돌려준다.
// until 은 응답이 Advertiser 에 닿은 뒤를 기다리는 자리다 — 서버가 본 것은 응답 앞이다.
func runAdverts(t *testing.T, a *Advertiser, n int, seen func() []string, until ...func() bool) []string {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for len(seen()) < n || (len(until) > 0 && !until[0]()) {
			time.Sleep(time.Millisecond)
		}
		cancel()
	}()
	a.Run(ctx)
	return seen()[:n]
}

// 여유가 min 아래로 떨어지면 노드가 스스로 graceful 을 싣고, min + 1 GB 를 되찾으면 푼다.
// 응답의 drain 이 Held 에 앉는다 — 소유자 drain 과 같은 길이다. 걸기와 풀기는 한 줄씩 적는다.
func TestDrain_LowDiskDrainsTheNodeAndLiftsAtTheLine(t *testing.T) {
	srv, seen := echoAdvertServer(t)
	defer srv.Close()
	var logs lockedBuffer
	held := NewHeld()
	free := &fakeFree{vals: []uint64{20 * gib, 7 * gib, 10 * gib, 11 * gib}}
	cfg := filepath.Join(t.TempDir(), "local.yaml")
	a := &Advertiser{
		Client: &Client{Base: srv.URL, Token: "t", Principal: "p", HTTP: srv.Client()},
		Ident:  Identity{NodeID: "n1", Label: "l", Config: cfg},
		Caps:   func() Capabilities { return Capabilities{} },
		Every:  time.Millisecond,
		Log:    slog.New(slog.NewTextHandler(&logs, nil)),
		Held:   held,
		Local:  Local{Workspace: t.TempDir(), MinFreeGB: 10},
		free:   free.free,
	}
	// 응답의 drain 이 Held 에 앉는다 — 마지막 광고가 푼 것까지 닿아야 끝난다
	got := runAdverts(t, a, 4, seen, func() bool { return held.Drain() == contract.DrainNone })
	want := []string{contract.DrainNone, contract.DrainGraceful, contract.DrainGraceful, contract.DrainNone}
	if !slices.Equal(got, want) {
		t.Fatalf("adverts carried %q, want %q", got, want)
	}
	out := logs.String()
	if strings.Count(out, "free disk below min_free_gb; draining this node") != 1 ||
		strings.Count(out, "free disk recovered; lifting the disk drain") != 1 {
		t.Fatalf("the log does not say it once each:\n%s", out)
	}
	status, err := ReadStatus(cfg)
	if err != nil || status.Drain == nil || status.Drain.Effective != contract.DrainNone || len(status.Drain.Sources) != 0 {
		t.Fatalf("status drain = %+v err = %v", status.Drain, err)
	}
}

// 소유자 at-boundary 와 여유 부족이 겹치면 광고에는 센 쪽 하나가, 상태 파일에는 출처 둘이 남는다.
func TestDrain_TheAdvertCarriesTheCombinedValueAndTheStatusBothSources(t *testing.T) {
	srv, seen := echoAdvertServer(t)
	defer srv.Close()
	cfg := filepath.Join(t.TempDir(), "local.yaml")
	if err := os.WriteFile(PolicyPath(cfg), []byte("drain: at-boundary\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	held := NewHeld()
	a := &Advertiser{
		Client: &Client{Base: srv.URL, Token: "t", Principal: "p", HTTP: srv.Client()},
		Ident:  Identity{NodeID: "n1", Label: "l", Config: cfg},
		Caps:   func() Capabilities { return Capabilities{} },
		Every:  time.Millisecond,
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		Held:   held,
		Local:  Local{Workspace: t.TempDir(), MinFreeGB: 10},
		free:   (&fakeFree{vals: []uint64{2 * gib}}).free,
	}
	got := runAdverts(t, a, 2, seen, func() bool { return held.Drain() == contract.DrainAtBoundary })
	if got[0] != contract.DrainAtBoundary || got[1] != contract.DrainAtBoundary {
		t.Fatalf("adverts carried %q", got)
	}
	status, err := ReadStatus(cfg)
	if err != nil || status.Drain == nil {
		t.Fatalf("status = %+v err = %v", status, err)
	}
	if status.Drain.Effective != contract.DrainAtBoundary || len(status.Drain.Sources) != 2 ||
		status.Drain.Sources[0].Kind != DrainOwner || status.Drain.Sources[1].Kind != DrainDisk ||
		status.Drain.Sources[1].Detail != "free 2 GB < min 10 GB" || status.Drain.At.IsZero() {
		t.Fatalf("status drain = %+v", status.Drain)
	}
}

// 워크스페이스가 없으면 여유를 안 잰다. 못 재면 넉넉한 것으로 치고 같은 원인은 한 번만 적는다.
func TestDrain_NoWorkspaceOrNoMeasureMeansNoDiskDrain(t *testing.T) {
	srv, seen := echoAdvertServer(t)
	defer srv.Close()
	free := &fakeFree{vals: []uint64{0}}
	a := &Advertiser{
		Client: &Client{Base: srv.URL, Token: "t", Principal: "p", HTTP: srv.Client()},
		Ident:  Identity{NodeID: "n1", Label: "l"},
		Caps:   func() Capabilities { return Capabilities{} },
		Every:  time.Millisecond,
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		Local:  Local{MinFreeGB: 10},
		free:   free.free,
	}
	for _, d := range runAdverts(t, a, 2, seen) {
		if d != contract.DrainNone {
			t.Fatalf("a node without a workspace drained: %q", d)
		}
	}
	if free.count() != 0 {
		t.Fatal("measured free disk without a workspace")
	}

	srv2, seen2 := echoAdvertServer(t)
	defer srv2.Close()
	var logs lockedBuffer
	broken := errors.New("statfs: no such file or directory")
	a2 := &Advertiser{
		Client: &Client{Base: srv2.URL, Token: "t", Principal: "p", HTTP: srv2.Client()},
		Ident:  Identity{NodeID: "n1", Label: "l"},
		Caps:   func() Capabilities { return Capabilities{} },
		Every:  time.Millisecond,
		Log:    slog.New(slog.NewTextHandler(&logs, nil)),
		Local:  Local{Workspace: "/gone", MinFreeGB: 10},
		free:   (&fakeFree{vals: []uint64{0, 0, 0, 0}, errs: []error{nil, broken, broken, nil}}).free,
	}
	got := runAdverts(t, a2, 4, seen2)
	// 0 GB 를 재서 걸었다 -> 못 재면 넉넉한 것으로 쳐서 푼다 -> 다시 재서 건다
	want := []string{contract.DrainGraceful, contract.DrainNone, contract.DrainNone, contract.DrainGraceful}
	if !slices.Equal(got, want) {
		t.Fatalf("adverts carried %q, want %q", got, want)
	}
	out := logs.String()
	if strings.Count(out, "cannot measure free disk; assuming enough") != 1 {
		t.Fatalf("the measure failure is not written once:\n%s", out)
	}
	if !strings.Contains(out, "free_gb=unknown") {
		t.Fatalf("the lift after a failed measure claims a number:\n%s", out)
	}
}
