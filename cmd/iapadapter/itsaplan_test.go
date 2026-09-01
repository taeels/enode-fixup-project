package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// iapCall 은 서버가 실제로 받은 왕복 하나다.
//
// 경로만 보면 부족하다 — 러너 프로토콜에서 계약인 것은 메서드 · 키 헤더 ·
// 본문의 모양까지다 (itsaplan.go 머리 주석의 세 엔드포인트 + ADR-040 §1).
type iapCall struct {
	Method string
	Path   string
	Query  string
	Key    string
	CType  string
	Body   string
}

// iapServer 는 받은 왕복을 전부 기록하는 It's a Plan 대역이다.
type iapServer struct {
	mu    sync.Mutex
	calls []iapCall
}

func (s *iapServer) record(c iapCall) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, c)
}

func (s *iapServer) seen() []iapCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]iapCall, len(s.calls))
	copy(out, s.calls)
	return out
}

// newIAPClient 는 대역을 세우고 그것을 보는 클라이언트를 준다.
func newIAPClient(t *testing.T, h http.HandlerFunc) (*IAP, *iapServer) {
	t.Helper()
	rec := &iapServer{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		rec.record(iapCall{
			Method: r.Method,
			Path:   r.URL.Path,
			Query:  r.URL.RawQuery,
			Key:    r.Header.Get("x-api-key"),
			CType:  r.Header.Get("Content-Type"),
			Body:   string(raw),
		})
		h(w, r)
	}))
	t.Cleanup(srv.Close)
	return NewIAP(srv.URL, "secret-key"), rec
}

// okJSON 은 어느 라우트에나 통하는 빈 객체를 돌려준다.
func okJSON(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{}`))
}

func TestIAP_NewIAPTrimsTheTrailingSlash(t *testing.T) {
	// 꼬리 슬래시를 안 떼면 경로가 //agent-runs/claim 이 되어 라우트가 안 맞는다.
	c := NewIAP("https://example.test/api/", "k")
	if c.base != "https://example.test/api" {
		t.Fatalf("base = %q, want the trailing slash removed", c.base)
	}
	if c.key != "k" {
		t.Fatalf("key = %q, want it carried through", c.key)
	}
	if c.client == nil || c.client.Timeout == 0 {
		t.Fatal("the client has no timeout — a hung tracker would hold the adapter forever")
	}
}

// 같은 에이전트 키 하나로 전부 된다 (itsaplan.go 머리 주석) — 그리고 그
// 「전부」가 메서드와 경로까지 포함해 여기 고정된다.
func TestIAP_EveryRouteCarriesTheAgentKey(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name   string
		call   func(c *IAP) error
		method string
		path   string
	}{
		{"claim", func(c *IAP) error { _, err := c.Claim(ctx); return err },
			"POST", "/agent-runs/claim"},
		{"heartbeat", func(c *IAP) error { return c.Heartbeat(ctx, 42) },
			"POST", "/agent-runs/42/heartbeat"},
		{"result", func(c *IAP) error { return c.Result(ctx, 42, "success", "", "") },
			"POST", "/agent-runs/42/result"},
		{"comment", func(c *IAP) error { _, err := c.Comment(ctx, 7, "hi", 0); return err },
			"POST", "/issues/7/comments"},
		{"columns", func(c *IAP) error { _, err := c.Columns(ctx, "EP"); return err },
			"GET", "/projects/EP"},
		{"move issue", func(c *IAP) error { return c.MoveIssue(ctx, 7, 3) },
			"PATCH", "/issues/7"},
		{"feed", func(c *IAP) error { _, err := c.Feed(ctx, 7, 10); return err },
			"GET", "/issues/7/feed"},
		{"issue", func(c *IAP) error { _, err := c.Issue(ctx, 7); return err },
			"GET", "/issues/7"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, rec := newIAPClient(t, okJSON)
			if err := tc.call(c); err != nil {
				t.Fatalf("call failed: %v", err)
			}
			seen := rec.seen()
			if len(seen) != 1 {
				t.Fatalf("made %d round trips, want exactly 1", len(seen))
			}
			if seen[0].Method != tc.method {
				t.Fatalf("method = %q, want %q", seen[0].Method, tc.method)
			}
			if seen[0].Path != tc.path {
				t.Fatalf("path = %q, want %q", seen[0].Path, tc.path)
			}
			if seen[0].Key != "secret-key" {
				t.Fatalf("x-api-key = %q — this route would be rejected", seen[0].Key)
			}
		})
	}
}

func TestIAP_DoSetsContentTypeOnlyWithABody(t *testing.T) {
	ctx := context.Background()
	c, rec := newIAPClient(t, okJSON)

	if _, err := c.Claim(ctx); err != nil {
		t.Fatal(err)
	}
	if err := c.Result(ctx, 1, "success", "", ""); err != nil {
		t.Fatal(err)
	}

	seen := rec.seen()
	if seen[0].CType != "" {
		t.Fatalf("claim sent Content-Type %q with no body", seen[0].CType)
	}
	if seen[0].Body != "" {
		t.Fatalf("claim sent a body %q, want none", seen[0].Body)
	}
	if seen[1].CType != "application/json" {
		t.Fatalf("result sent Content-Type %q, want application/json", seen[1].CType)
	}
}

// 상태 코드와 서버가 적은 이유가 오류에 남아야 한다 — 그것이 그대로
// 이슈 코멘트로 실린다 (main.go failOut).
func TestIAP_DoKeepsTheStatusAndTheBodyInTheError(t *testing.T) {
	c, _ := newIAPClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("  agent key is not allowed here  "))
	})

	status, err := c.do(context.Background(), "POST", "/agent-runs/claim", nil, nil)
	if err == nil {
		t.Fatal("a 403 was swallowed")
	}
	if status != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 carried back to the caller", status)
	}
	for _, want := range []string{"POST", "/agent-runs/claim", "403", "agent key is not allowed here"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q lost %q", err.Error(), want)
		}
	}
	if strings.Contains(err.Error(), "  agent key") {
		t.Fatalf("error %q kept the untrimmed body", err.Error())
	}
}

func TestIAP_DoRejectsAResponseItCannotRead(t *testing.T) {
	c, _ := newIAPClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"run": {{{`))
	})

	status, err := c.do(context.Background(), "POST", "/agent-runs/claim", nil, &struct{}{})
	if err == nil {
		t.Fatal("unreadable JSON was accepted as a valid response")
	}
	// 2xx 였다는 사실을 지우지 않는다 — 왕복은 성공했고 본문만 못 읽었다.
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200 kept", status)
	}
	if !strings.Contains(err.Error(), "cannot read response") {
		t.Fatalf("error %q does not say the body was unreadable", err.Error())
	}
}

// 선을 타기 전에 실패하는 세 갈래는 상태 코드가 없다 — 0 이어야 한다.
// 0 이 아니면 호출자가 그것을 HTTP 응답으로 오독한다.
func TestIAP_DoFailsBeforeTheWire(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name   string
		method string
		body   any
		dead   bool
	}{
		{name: "a body that cannot be marshalled", method: "POST", body: make(chan int)},
		{name: "a method the request builder refuses", method: "BAD METHOD"},
		{name: "a tracker that is not listening", method: "POST", dead: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := newIAPClient(t, okJSON)
			if tc.dead {
				// 대역을 미리 닫아 전송 자체를 실패시킨다.
				c = NewIAP("http://127.0.0.1:1", "k")
			}
			status, err := c.do(ctx, tc.method, "/agent-runs/claim", tc.body, nil)
			if err == nil {
				t.Fatal("the failure was swallowed")
			}
			if status != 0 {
				t.Fatalf("status = %d, want 0 — nothing reached the tracker", status)
			}
		})
	}
}

// 클레임이 status 를 안 바꾼다 (실측 ⑤) — 큐가 비면 오류가 아니라 (nil, nil) 이다.
// 오류로 올리면 어댑터가 빈 큐마다 경고를 찍고 poll 간격이 무의미해진다.
func TestIAP_ClaimReturnsNothingWhenTheQueueIsEmpty(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{"an explicit null run", `{"run":null}`},
		{"no run key at all", `{}`},
		{"an empty response body", ""},
		{"whitespace only", "   \n  "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := newIAPClient(t, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(tc.body))
			})
			run, err := c.Claim(context.Background())
			if err != nil {
				t.Fatalf("an empty queue was reported as an error: %v", err)
			}
			if run != nil {
				t.Fatalf("run = %+v, want nil for an empty queue", run)
			}
		})
	}
}

// 일곱 개이고 전부 자연어다 (integration §5 · 실측 ③) — 하나라도 흘리면
// 계약이 목표를 잃거나 (issueIdentifier) 어댑터가 일감을 무시한다 (issueId).
func TestIAP_ClaimReadsAllSevenFields(t *testing.T) {
	c, _ := newIAPClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"run":{
			"id": 91,
			"trigger": "delegation",
			"prompt": "please do the thing",
			"systemPrompt": "you are a runner",
			"attempts": 2,
			"issueId": 2,
			"issueIdentifier": "EP-2"
		}}`))
	})

	run, err := c.Claim(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if run == nil {
		t.Fatal("a queued run came back as an empty queue")
	}
	if run.ID != 91 || run.Trigger != "delegation" || run.Attempts != 2 {
		t.Fatalf("run = %+v", run)
	}
	if run.Prompt != "please do the thing" || run.SystemPrompt != "you are a runner" {
		t.Fatalf("the natural-language fields were lost: %+v", run)
	}
	if run.IssueIdentifier != "EP-2" {
		t.Fatalf("issueIdentifier = %q — the contract loses its target", run.IssueIdentifier)
	}
	if run.IssueID == nil || *run.IssueID != 2 {
		t.Fatalf("issueId = %v — handle would drop this job", run.IssueID)
	}
}

// issueId 는 포인터다 — 없는 것과 0 이 달라야 handle 이 가른다.
func TestIAP_ClaimKeepsAMissingIssueIDDistinctFromZero(t *testing.T) {
	c, _ := newIAPClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"run":{"id":1,"trigger":"delegation"}}`))
	})
	run, err := c.Claim(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if run.IssueID != nil {
		t.Fatalf("issueId = %v, want nil for a job with no issue", *run.IssueID)
	}
}

// output 에 봉인을 붓지 않는다 (ADR-040 §1 ①) — 빈 값은 키 자체를 안 낸다.
// 빈 문자열을 실으면 agent_run.output 이 빈 값으로 덮여 UI 가 지저분해진다.
func TestIAP_ResultOmitsTheEmptyFields(t *testing.T) {
	for _, tc := range []struct {
		name    string
		output  string
		errMsg  string
		wantKey []string
		notKey  []string
	}{
		{"a status on its own", "", "",
			[]string{"status"}, []string{"output", "error"}},
		{"a one-line summary", "SUCCEEDED - 2/2 checks", "",
			[]string{"status", "output"}, []string{"error"}},
		{"a failure reason", "", "the fleet never freed up",
			[]string{"status", "error"}, []string{"output"}},
		{"both", "FAILED - 1/2 checks", "step 3 exited 1",
			[]string{"status", "output", "error"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, rec := newIAPClient(t, okJSON)
			if err := c.Result(context.Background(), 42, "success", tc.output, tc.errMsg); err != nil {
				t.Fatal(err)
			}
			var body map[string]any
			if err := json.Unmarshal([]byte(rec.seen()[0].Body), &body); err != nil {
				t.Fatalf("the result body is not JSON: %v", err)
			}
			for _, k := range tc.wantKey {
				if _, ok := body[k]; !ok {
					t.Fatalf("body %v is missing %q", body, k)
				}
			}
			for _, k := range tc.notKey {
				if _, ok := body[k]; ok {
					t.Fatalf("body %v carries an empty %q", body, k)
				}
			}
			if body["status"] != "success" {
				t.Fatalf("status = %v, want success", body["status"])
			}
		})
	}
}

// replyTo 가 0 이 아니면 그 코멘트의 답글이 된다 — 0 이면 키를 안 낸다.
// 0 을 실으면 트래커가 "0번 코멘트에 대한 답글"로 읽는다.
func TestIAP_CommentRepliesOnlyWhenAskedTo(t *testing.T) {
	for _, tc := range []struct {
		name    string
		replyTo int
		want    bool
	}{
		{"a fresh comment", 0, false},
		{"a negative reply id is not a reply", -1, false},
		{"a reply to comment 5", 5, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, rec := newIAPClient(t, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"id":314}`))
			})
			id, err := c.Comment(context.Background(), 7, "the body", tc.replyTo)
			if err != nil {
				t.Fatal(err)
			}
			if id != 314 {
				t.Fatalf("comment id = %d, want 314 — handOff records it in _outbound", id)
			}
			var body map[string]any
			if err := json.Unmarshal([]byte(rec.seen()[0].Body), &body); err != nil {
				t.Fatal(err)
			}
			if body["body"] != "the body" {
				t.Fatalf("body = %v", body["body"])
			}
			got, has := body["replyToId"]
			if has != tc.want {
				t.Fatalf("replyToId present = %v (%v), want %v", has, got, tc.want)
			}
			if tc.want && got != float64(tc.replyTo) {
				t.Fatalf("replyToId = %v, want %d", got, tc.replyTo)
			}
		})
	}
}

// GET /projects/{key}/columns 같은 라우트는 없다 (실측 §10.5.2) —
// 프로젝트 payload 안에 실려 온다.
func TestIAP_ColumnsComeFromTheProjectPayload(t *testing.T) {
	c, rec := newIAPClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":1,"key":"EP","columns":[
			{"id":11,"name":"Todo"},{"id":12,"name":"In Progress"},{"id":13,"name":"Done"}]}`))
	})

	cols, err := c.Columns(context.Background(), "EP")
	if err != nil {
		t.Fatal(err)
	}
	if rec.seen()[0].Path != "/projects/EP" {
		t.Fatalf("path = %q, want the project itself and not a columns route", rec.seen()[0].Path)
	}
	if len(cols) != 3 {
		t.Fatalf("read %d columns, want 3", len(cols))
	}
	if cols[1].ID != 12 || cols[1].Name != "In Progress" {
		t.Fatalf("column 1 = %+v — move would not find its target", cols[1])
	}
}

func TestIAP_MoveIssuePatchesTheColumn(t *testing.T) {
	c, rec := newIAPClient(t, okJSON)
	if err := c.MoveIssue(context.Background(), 7, 13); err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(rec.seen()[0].Body), &body); err != nil {
		t.Fatal(err)
	}
	if body["columnId"] != float64(13) {
		t.Fatalf("columnId = %v, want 13", body["columnId"])
	}
	if len(body) != 1 {
		t.Fatalf("the patch carries %v — a column move must not touch anything else", body)
	}
}

// limit 이 밖으로 새면 트래커가 거절하거나 조용히 자른다. 0 이하와 100 초과는
// 100 으로 닫는다.
func TestIAP_FeedClampsTheLimit(t *testing.T) {
	for _, tc := range []struct {
		name  string
		limit int
		want  string
	}{
		{"zero becomes the maximum", 0, "limit=100"},
		{"a negative limit becomes the maximum", -5, "limit=100"},
		{"above the maximum is capped", 101, "limit=100"},
		{"the maximum itself is kept", 100, "limit=100"},
		{"a small limit is kept", 7, "limit=7"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, rec := newIAPClient(t, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"items":[]}`))
			})
			if _, err := c.Feed(context.Background(), 7, tc.limit); err != nil {
				t.Fatal(err)
			}
			if rec.seen()[0].Query != tc.want {
				t.Fatalf("query = %q, want %q", rec.seen()[0].Query, tc.want)
			}
		})
	}
}

// 코멘트 전용 라우트가 없다 — 코멘트는 피드의 한 종류로 온다.
func TestIAP_FeedReadsCommentsAndActivitiesAlike(t *testing.T) {
	c, _ := newIAPClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"items":[
			{"id":9,"kind":"activity","actorName":"enode fleet","action":"delegated","createdAt":"2026-08-24T00:00:00Z"},
			{"id":8,"kind":"comment","actorName":"someone","body":"ok","replyToId":7,"createdAt":"2026-08-24T00:01:00Z"}]}`))
	})

	items, err := c.Feed(context.Background(), 7, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("read %d items, want 2", len(items))
	}
	if items[0].Kind != "activity" || items[0].Action != "delegated" {
		t.Fatalf("item 0 = %+v", items[0])
	}
	if items[1].Kind != "comment" || items[1].Body != "ok" {
		t.Fatalf("item 1 = %+v", items[1])
	}
	if items[1].ReplyToID == nil || *items[1].ReplyToID != 7 {
		t.Fatalf("replyToId = %v — the answer could not be matched to our question", items[1].ReplyToID)
	}
	if items[0].ReplyToID != nil {
		t.Fatal("an activity was given a replyToId it does not have")
	}
}

// 어댑터가 표를 안 들고 중복을 막는 방법이다 — 정본은 이슈 자신이다.
// 활동(activity)은 세면 안 된다: 그 표지를 담은 활동이 있어도 우리가 결과를
// 넘긴 것은 아니다.
func TestIAP_AlreadySaidOnlyCountsComments(t *testing.T) {
	marker := ResultMarker("itsaplan-EP-2-1")
	for _, tc := range []struct {
		name  string
		items string
		want  bool
	}{
		{"nothing on the issue yet", `[]`, false},
		{"our result comment is there", `[{"kind":"comment","body":"done ` + marker + ` tail"}]`, true},
		{"an activity carrying the marker does not count",
			`[{"kind":"activity","body":"` + marker + `"}]`, false},
		{"another run's marker does not count",
			`[{"kind":"comment","body":"enode:result:itsaplan-EP-2-2"}]`, false},
		{"the question comment alone does not count",
			`[{"kind":"comment","body":"run ` + "`itsaplan-EP-2-1`" + ` step plan"}]`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := newIAPClient(t, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"items":` + tc.items + `}`))
			})
			said, err := c.AlreadySaid(context.Background(), 7, marker)
			if err != nil {
				t.Fatal(err)
			}
			if said != tc.want {
				t.Fatalf("AlreadySaid = %v, want %v", said, tc.want)
			}
		})
	}
}

// 피드를 못 읽으면 중복 여부를 모르는 것이지 "안 썼다"가 아니다.
// false, nil 로 뭉개면 reportUnreported 가 결과를 두 번 쓴다.
func TestIAP_AlreadySaidPropagatesAFeedFailure(t *testing.T) {
	c, _ := newIAPClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	})
	said, err := c.AlreadySaid(context.Background(), 7, "enode:result:x")
	if err == nil {
		t.Fatal("a feed failure was reported as 'not said yet'")
	}
	if said {
		t.Fatal("AlreadySaid = true on an error path")
	}
}

// 제목과 본문이 계약의 목표가 된다 (BuildContract).
func TestIAP_IssueReadsTheContractTarget(t *testing.T) {
	c, _ := newIAPClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":2,"sequenceNumber":14,"title":"the title",
			"description":"the body","columnId":11}`))
	})
	issue, err := c.Issue(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if issue.ID != 2 || issue.SequenceNumber != 14 || issue.ColumnID != 11 {
		t.Fatalf("issue = %+v", issue)
	}
	if issue.Title != "the title" || issue.Description != "the body" {
		t.Fatalf("the contract target was lost: %+v", issue)
	}
}

// 어느 라우트도 거절을 삼키지 않는다 — 삼키면 어댑터가 실패를 성공으로 넘긴다.
func TestIAP_EveryRouteSurfacesARejection(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name string
		call func(c *IAP) error
	}{
		{"claim", func(c *IAP) error {
			run, err := c.Claim(ctx)
			if run != nil {
				return nil
			}
			return err
		}},
		{"heartbeat", func(c *IAP) error { return c.Heartbeat(ctx, 42) }},
		{"result", func(c *IAP) error { return c.Result(ctx, 42, "success", "o", "e") }},
		{"comment", func(c *IAP) error {
			id, err := c.Comment(ctx, 7, "hi", 3)
			if id != 0 {
				return nil
			}
			return err
		}},
		{"columns", func(c *IAP) error {
			cols, err := c.Columns(ctx, "EP")
			if cols != nil {
				return nil
			}
			return err
		}},
		{"move issue", func(c *IAP) error { return c.MoveIssue(ctx, 7, 3) }},
		{"feed", func(c *IAP) error {
			items, err := c.Feed(ctx, 7, 10)
			if items != nil {
				return nil
			}
			return err
		}},
		{"issue", func(c *IAP) error {
			issue, err := c.Issue(ctx, 7)
			if issue != nil {
				return nil
			}
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := newIAPClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadGateway)
				_, _ = w.Write([]byte("the tracker is down"))
			})
			err := tc.call(c)
			if err == nil {
				t.Fatal("a 502 was swallowed, or a value was returned alongside it")
			}
			if !strings.Contains(err.Error(), "502") {
				t.Fatalf("error %q lost the status code", err.Error())
			}
		})
	}
}
