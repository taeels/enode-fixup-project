package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func quietLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// ready 파일에는 node_id 가 들어간다 — 파일에 실려 오므로 꼬리 개행이 붙는다.
// 그것을 안 떼면 node_id 가 광고의 것과 안 맞는다.
func TestTrimLine_StripsWhatAFileAppends(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want string
	}{
		{"a bare id", "node-abc", "node-abc"},
		{"a trailing newline", "node-abc\n", "node-abc"},
		{"a windows line ending", "node-abc\r\n", "node-abc"},
		{"trailing spaces", "node-abc  ", "node-abc"},
		{"all of them at once", "node-abc \r\n \n", "node-abc"},
		{"nothing but padding", " \n\r ", ""},
		{"an empty file", "", ""},
		// 안쪽 공백은 안 건드린다 — id 의 일부일 수 있다.
		{"inner spaces survive", "node abc\n", "node abc"},
		// 탭은 안 뗀다 — 뗄 것을 세 글자로 못박은 것이 이 함수의 계약이다.
		{"a tab is not padding", "node-abc\t", "node-abc\t"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := trimLine(tc.in); got != tc.want {
				t.Fatalf("trimLine(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// ready 파일만으로는 부족하다 — 그것은 노드 쪽 사실이고, 매칭은 Mediator 가
// 본 것으로 한다. 둘 사이에 지연이 있으면 Run 이 422 로 떨어진다.
func TestWaitAdvertised_WaitsForTheFleetNotTheNode(t *testing.T) {
	advertised := `{"capabilities":[{"capability":"orchestration","nodes":1,
	  "attrs":{"issue":["EP-2"]}}]}`

	t.Run("it returns as soon as the label shows up", func(t *testing.T) {
		m := renderMediator(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(advertised))
		})
		o := &Orchestrator{IssueKey: "EP-2", log: quietLog()}
		if err := o.WaitAdvertised(context.Background(), m, 5*time.Second); err != nil {
			t.Fatalf("the advertisement was there and it still waited: %v", err)
		}
	})

	t.Run("it keeps looking after a lookup failure", func(t *testing.T) {
		var mu sync.Mutex
		calls := 0
		m := renderMediator(t, func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			calls++
			n := calls
			mu.Unlock()
			if n == 1 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			_, _ = w.Write([]byte(advertised))
		})
		o := &Orchestrator{IssueKey: "EP-2", log: quietLog()}
		if err := o.WaitAdvertised(context.Background(), m, 10*time.Second); err != nil {
			t.Fatalf("one unreadable advertisement ended the wait: %v", err)
		}
		mu.Lock()
		defer mu.Unlock()
		if calls < 2 {
			t.Fatalf("asked %d times, want it to look again after the failure", calls)
		}
	})

	t.Run("it gives up at the deadline and says whose label was missing", func(t *testing.T) {
		m := renderMediator(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"capabilities":[]}`))
		})
		o := &Orchestrator{IssueKey: "EP-2", log: quietLog()}
		err := o.WaitAdvertised(context.Background(), m, 0)
		if err == nil {
			t.Fatal("it waited past its own deadline")
		}
		if !strings.Contains(err.Error(), "issue=EP-2") {
			t.Fatalf("error %q does not name the label it waited for", err.Error())
		}
	})

	t.Run("it stops when the adapter is going down", func(t *testing.T) {
		m := renderMediator(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"capabilities":[]}`))
		})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		o := &Orchestrator{IssueKey: "EP-2", log: quietLog()}
		err := o.WaitAdvertised(ctx, m, time.Minute)
		if err == nil {
			t.Fatal("a cancelled adapter kept waiting")
		}
		if !strings.Contains(err.Error(), "context canceled") {
			t.Fatalf("error = %v, want the cancellation", err)
		}
	})
}

// 지우는 것은 띄운 쪽이다 (adapter-example §4.2) — 다만 keep_workspace 면
// 조사할 수 있게 남긴다.
func TestOrchestrator_CleansTheWorkspaceUnlessAskedToKeepIt(t *testing.T) {
	for _, tc := range []struct {
		name string
		keep bool
	}{
		{"the workspace is removed by default", false},
		{"keep_workspace leaves it for inspection", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "orch-EP-2")
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "orch.log"), []byte("x"), 0o600); err != nil {
				t.Fatal(err)
			}
			o := &Orchestrator{
				IssueKey: "EP-2", Dir: dir, log: quietLog(),
				keep: tc.keep, done: make(chan struct{}),
			}
			close(o.done)

			o.Stop()

			_, err := os.Stat(dir)
			if tc.keep && err != nil {
				t.Fatalf("keep_workspace still removed %s: %v", dir, err)
			}
			if !tc.keep && err == nil {
				t.Fatalf("%s was left behind", dir)
			}
		})
	}
}
