package runctl

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"
)

func newClient(h http.HandlerFunc) (*Client, func()) {
	srv := httptest.NewServer(h)
	return &Client{Base: srv.URL, Token: "t", Principal: "a@b", HTTP: srv.Client()}, srv.Close
}

// 와이어의 HTTP 상태를 그대로 들고 온다 — CLI 종료코드로의 번역은
// cmd/runctl 이 한다. 경계가 둘이라 하나로 통일하지 않는다 (INVARIANTS §4).
func TestFailCarriesWireCode(t *testing.T) {
	for _, code := range []int{400, 409, 422, 503} {
		c, done := newClient(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(code)
			_, _ = w.Write([]byte(`{"error":{"code":` + itoa(code) + `,"reason":"why"}}`))
		})
		_, err := c.Status(context.Background(), "r")
		var f *Fail
		if !errors.As(err, &f) || f.Code != code || f.Reason != "why" {
			t.Fatalf("code=%d → %v", code, err)
		}
		done()
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// 인증과 식별이 모든 요청에 실린다 (ADR-015 §1).
func TestHeaders(t *testing.T) {
	var gotAuth, gotPrincipal string
	c, done := newClient(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotPrincipal = r.Header.Get("Authorization"), r.Header.Get("X-Enode-Principal")
		_, _ = w.Write([]byte(`{"run_id":"r","state":"RUNNING"}`))
	})
	defer done()
	if _, err := c.Status(context.Background(), "r"); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer t" || gotPrincipal != "a@b" {
		t.Fatalf("auth=%q principal=%q", gotAuth, gotPrincipal)
	}
}

// Wait 는 폴링일 뿐이다 — runctl 을 상태 있게 만들지 않는다.
// 중간에 죽어도 Run 은 계속 돈다 (ADR-013 이 --interactive 를 거절한 사유).
func TestWaitUntilTerminal(t *testing.T) {
	n := 0
	c, done := newClient(func(w http.ResponseWriter, r *http.Request) {
		n++
		state := "RUNNING"
		if n >= 3 {
			state = "SUCCEEDED"
		}
		_, _ = w.Write([]byte(`{"run_id":"r","state":"` + state + `"}`))
	})
	defer done()
	run, err := c.Wait(context.Background(), "r", time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if run.State != "SUCCEEDED" || n < 3 {
		t.Fatalf("state=%s polls=%d", run.State, n)
	}
}

func TestTerminal(t *testing.T) {
	for s, want := range map[string]bool{
		"SUCCEEDED": true, "FAILED": true,
		"RUNNING": false, "VERIFYING": false, "RESOLVING": false,
	} {
		if Terminal(s) != want {
			t.Errorf("Terminal(%s)=%v", s, !want)
		}
	}
}

// 제로값 RunsQuery 는 질의 문자열을 아예 안 만든다. 규칙이 아니라 계약이다 —
// ?limit=0 은 서버가 400 으로 거절하고, 인자 없이 부르는 것이 mcp runs.list 의
// 기본 사용법이므로 그것이 곧 첫 호출의 실패다 (CP5).
func TestRunsZeroQuerySendsNoQueryString(t *testing.T) {
	var gotPath, gotRaw string
	c, done := newClient(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotRaw = r.URL.Path, r.URL.RawQuery
		_, _ = w.Write([]byte(`{"runs":[]}`))
	})
	defer done()
	if _, err := c.Runs(context.Background(), RunsQuery{}); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/runs" {
		t.Errorf("path = %q, want /v1/runs", gotPath)
	}
	if gotRaw != "" {
		t.Errorf("raw query = %q, want empty", gotRaw)
	}
}

// 넷을 다 채우면 넷이 다 실린다. since 는 RFC 3339 다.
func TestRunsQueryCarriesAllFour(t *testing.T) {
	var got url.Values
	c, done := newClient(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		_, _ = w.Write([]byte(`{"runs":[]}`))
	})
	defer done()
	q := RunsQuery{
		State: "RUNNING",
		Since: time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC),
		Work:  "w-1",
		Limit: 50,
	}
	if _, err := c.Runs(context.Background(), q); err != nil {
		t.Fatal(err)
	}
	want := url.Values{
		"state": {"RUNNING"},
		"since": {"2026-02-03T04:05:06Z"},
		"work":  {"w-1"},
		"limit": {"50"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("query = %v, want %v", got, want)
	}
}

// 채운 것만 실린다 — 하나만 채워도 나머지 셋이 안 따라 나간다.
func TestRunsQueryOmitsEmptyFields(t *testing.T) {
	cases := []struct {
		name string
		q    RunsQuery
		want string
	}{
		{"state only", RunsQuery{State: "FAILED"}, "state=FAILED"},
		{"work only", RunsQuery{Work: "w-9"}, "work=w-9"},
		{"limit only", RunsQuery{Limit: 10}, "limit=10"},
		{"since only", RunsQuery{Since: time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)}, "since=2026-02-03T04%3A05%3A06Z"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotRaw string
			c, done := newClient(func(w http.ResponseWriter, r *http.Request) {
				gotRaw = r.URL.RawQuery
				_, _ = w.Write([]byte(`{"runs":[]}`))
			})
			defer done()
			if _, err := c.Runs(context.Background(), tc.q); err != nil {
				t.Fatal(err)
			}
			if gotRaw != tc.want {
				t.Fatalf("raw query = %q, want %q", gotRaw, tc.want)
			}
		})
	}
}

// verbatimBody 는 키 순서와 공백을 일부러 어긋나게 둔 응답이다.
// 파싱해서 다시 마셜하면 이 모양이 눈에 보이게 달라진다.
const verbatimBody = `{"observed_at":"2026-02-03T04:05:06Z",   "count":1,
  "nodes":[ {"stale":false,"node_id":"n-1","labels":{}} ]}`

// 읽기 둘은 본문을 글자 그대로 낸다 — mcp 의 fleet.list 가 GET /v1/nodes 와
// 글자까지 같아야 하고(CP5), 재마셜은 키 순서와 생략 규칙을 그 사이에서 가른다.
func TestNodesAndRunsReturnBodyVerbatim(t *testing.T) {
	for _, tc := range []struct {
		name string
		path string
		call func(*Client) (json.RawMessage, error)
	}{
		{"Nodes", "/v1/nodes", func(c *Client) (json.RawMessage, error) {
			return c.Nodes(context.Background())
		}},
		{"Runs", "/v1/runs", func(c *Client) (json.RawMessage, error) {
			return c.Runs(context.Background(), RunsQuery{})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var gotPath string
			c, done := newClient(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				_, _ = w.Write([]byte(verbatimBody))
			})
			defer done()
			raw, err := tc.call(c)
			if err != nil {
				t.Fatal(err)
			}
			if gotPath != tc.path {
				t.Errorf("path = %q, want %q", gotPath, tc.path)
			}
			if string(raw) != verbatimBody {
				t.Fatalf("body = %q, want %q", string(raw), verbatimBody)
			}
		})
	}
}

// 읽기 둘도 2xx 가 아니면 Fail 이 된다 — 기존 do 를 그대로 타기 때문이다.
func TestNodesAndRunsFailCarriesWireCode(t *testing.T) {
	for _, tc := range []struct {
		name string
		call func(*Client) (json.RawMessage, error)
	}{
		{"Nodes", func(c *Client) (json.RawMessage, error) {
			return c.Nodes(context.Background())
		}},
		{"Runs", func(c *Client) (json.RawMessage, error) {
			return c.Runs(context.Background(), RunsQuery{State: "RUNNING"})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, code := range []int{400, 429, 503} {
				c, done := newClient(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(code)
					_, _ = w.Write([]byte(`{"error":{"code":` + itoa(code) + `,"reason":"why"}}`))
				})
				raw, err := tc.call(c)
				var f *Fail
				if !errors.As(err, &f) || f.Code != code || f.Reason != "why" {
					t.Errorf("code=%d → %v", code, err)
				}
				if raw != nil {
					t.Errorf("code=%d → body %q, want nil", code, string(raw))
				}
				done()
			}
		})
	}
}

// Status 가 requires · chosen · started_at 을 푼다. 안 넓히면 이 유닛이 연 값이
// runctl status 와 mcp 의 run.get 에서만 조용히 사라진다.
func TestStatusDecodesRequiresChosenAndStartedAt(t *testing.T) {
	const requires = `[{"as":"builder","count":2,"attrs":{"os":["linux"]}}]`
	body := `{"run_id":"r-1","state":"RUNNING","requires":` + requires + `,` +
		`"steps":[` +
		`{"seq":1,"id":"build","state":"RUNNING","attempt":2,"chosen":true,"started_at":"2026-02-03T04:05:06Z"},` +
		`{"seq":2,"id":"test","state":"PENDING","attempt":0,"chosen":false}]}`
	c, done := newClient(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	})
	defer done()
	run, err := c.Status(context.Background(), "r-1")
	if err != nil {
		t.Fatal(err)
	}
	if string(run.Requires) != requires {
		t.Errorf("requires = %q, want %q", string(run.Requires), requires)
	}
	if len(run.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(run.Steps))
	}
	if !run.Steps[0].Chosen {
		t.Error("step 1 chosen = false, want true")
	}
	want := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	if run.Steps[0].StartedAt == nil || !run.Steps[0].StartedAt.Equal(want) {
		t.Errorf("step 1 started_at = %v, want %v", run.Steps[0].StartedAt, want)
	}
	if run.Steps[1].Chosen {
		t.Error("step 2 chosen = true, want false")
	}
	if run.Steps[1].StartedAt != nil {
		t.Errorf("step 2 started_at = %v, want nil", run.Steps[1].StartedAt)
	}
}

// 잘린 본문은 원문이 아니다 — 짧게 끊긴 것을 그대로 돌려주면 부르는 쪽이
// 그것을 온전한 응답으로 믿는다. 읽기 실패는 에러로 낸다.
func TestNodesTruncatedBodyIsAnError(t *testing.T) {
	c, done := newClient(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "64")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"nodes":`))
	})
	defer done()
	raw, err := c.Nodes(context.Background())
	if err == nil {
		t.Fatalf("err = nil, want a read failure; body = %q", string(raw))
	}
	if raw != nil {
		t.Errorf("body = %q, want nil", string(raw))
	}
}
