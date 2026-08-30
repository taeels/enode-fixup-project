package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// medCall 은 Mediator 가 실제로 받은 왕복 하나다.
type medCall struct {
	Method    string
	Path      string
	Auth      string
	Principal string
	CType     string
	Body      string
}

type medServer struct {
	mu    sync.Mutex
	calls []medCall
}

func (s *medServer) record(c medCall) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, c)
}

func (s *medServer) seen() []medCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]medCall, len(s.calls))
	copy(out, s.calls)
	return out
}

// newMediatorClient 는 대역을 세우고 그것을 보는 클라이언트를 준다.
func newMediatorClient(t *testing.T, principal string, h http.HandlerFunc) (*Mediator, *medServer) {
	t.Helper()
	rec := &medServer{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		rec.record(medCall{
			Method:    r.Method,
			Path:      r.URL.Path,
			Auth:      r.Header.Get("Authorization"),
			Principal: r.Header.Get("X-Enode-Principal"),
			CType:     r.Header.Get("Content-Type"),
			Body:      string(raw),
		})
		h(w, r)
	}))
	t.Cleanup(srv.Close)
	return NewMediator(srv.URL, "fleet-token", principal), rec
}

func TestMediator_NewMediatorTrimsTheTrailingSlash(t *testing.T) {
	m := NewMediator("https://mediator.test/", "tok", "taeels")
	if m.base != "https://mediator.test" {
		t.Fatalf("base = %q, want the trailing slash removed", m.base)
	}
	if m.token != "tok" || m.principal != "taeels" {
		t.Fatalf("token/principal = %q/%q", m.token, m.principal)
	}
	if m.client == nil || m.client.Timeout == 0 {
		t.Fatal("the client has no timeout - a hung Mediator would hold the adapter forever")
	}
}

// 되묻기의 답을 쓸 때 누가 답했는지가 봉인에 남는다 (ADR-033) — 그 통로가
// X-Enode-Principal 헤더 하나다. 비어 있으면 헤더 자체를 안 보낸다.
func TestMediator_SendsThePrincipalOnlyWhenItHasOne(t *testing.T) {
	for _, tc := range []struct {
		name      string
		principal string
	}{
		{"an adapter with a principal", "taeels"},
		{"an adapter with none", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, rec := newMediatorClient(t, tc.principal, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{}`))
			})
			if err := m.SubmitRun(context.Background(), []byte(`{"run_id":"r"}`)); err != nil {
				t.Fatal(err)
			}
			got := rec.seen()[0]
			if got.Auth != "Bearer fleet-token" {
				t.Fatalf("Authorization = %q", got.Auth)
			}
			if got.Principal != tc.principal {
				t.Fatalf("X-Enode-Principal = %q, want %q", got.Principal, tc.principal)
			}
			if got.CType != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json for a body", got.CType)
			}
			if got.Body != `{"run_id":"r"}` {
				t.Fatalf("the contract was altered on the way out: %q", got.Body)
			}
		})
	}
}

func TestMediator_DoSendsNoContentTypeWithoutABody(t *testing.T) {
	m, rec := newMediatorClient(t, "", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"capabilities":[]}`))
	})
	if _, err := m.Capabilities(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := rec.seen()[0]; got.CType != "" {
		t.Fatalf("Content-Type = %q on a bodyless GET", got.CType)
	}
}

// 404 는 오류가 아니라 센티널이다 — nextRunID 가 이것으로 「아직 없는 이름」을
// 찾는다. 다른 오류와 뭉개면 탐침이 첫 세대에서 멈춘다.
func TestMediator_MissingRunIsASentinelNotAnError(t *testing.T) {
	m, _ := newMediatorClient(t, "", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"no such run"}`))
	})
	run, err := m.GetRun(context.Background(), "itsaplan-EP-2-1")
	if !errors.Is(err, ErrNoRun) {
		t.Fatalf("err = %v, want ErrNoRun", err)
	}
	if run != nil {
		t.Fatalf("run = %+v, want nil", run)
	}
}

// 상태 코드를 살려 두는 이유가 하나다 — 409 와 422 를 호출자가 갈라야 한다
// (ADR-014 결정 3). error 하나로 뭉개면 그 정보가 여기서 사라진다.
func TestMediator_KeepsTheStatusCodeOnTheError(t *testing.T) {
	for _, status := range []int{400, 401, 409, 422, 500, 503} {
		m, _ := newMediatorClient(t, "", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte("  worker: need 1, currently 0  "))
		})
		err := m.SubmitRun(context.Background(), []byte(`{}`))
		var he *HTTPError
		if !errors.As(err, &he) {
			t.Fatalf("%d did not come back as *HTTPError: %v", status, err)
		}
		if he.Status != status {
			t.Fatalf("Status = %d, want %d", he.Status, status)
		}
		if he.Method != "POST" || he.Path != "/v1/runs" {
			t.Fatalf("the error lost the route: %+v", he)
		}
		if he.Body != "worker: need 1, currently 0" {
			t.Fatalf("Body = %q, want it trimmed", he.Body)
		}
		if !strings.Contains(he.Error(), "/v1/runs") {
			t.Fatalf("Error() = %q lost the route", he.Error())
		}
	}
}

func TestMediator_DoRejectsAResponseItCannotRead(t *testing.T) {
	m, _ := newMediatorClient(t, "", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"state": }`))
	})
	if _, err := m.GetRun(context.Background(), "itsaplan-EP-2-1"); err == nil {
		t.Fatal("unreadable JSON was accepted as a run lookup")
	}
}

func TestMediator_DoFailsBeforeTheWire(t *testing.T) {
	ctx := context.Background()
	t.Run("a method the request builder refuses", func(t *testing.T) {
		m, _ := newMediatorClient(t, "", func(w http.ResponseWriter, _ *http.Request) {})
		if err := m.do(ctx, "BAD METHOD", "/v1/runs", nil, nil); err == nil {
			t.Fatal("a malformed method was accepted")
		}
	})
	t.Run("a Mediator that is not listening", func(t *testing.T) {
		m := NewMediator("http://127.0.0.1:1", "t", "")
		if err := m.do(ctx, "GET", "/v1/asks", nil, nil); err == nil {
			t.Fatal("a transport failure was swallowed")
		}
	})
}

func TestMediator_GetRunReadsTheWholeView(t *testing.T) {
	m, rec := newMediatorClient(t, "", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"run_id":"itsaplan-EP-2-1","state":"RUNNING",
		  "assigned":[{"as":"worker","nodes":[{"node":"n-1","label":"host:orch"}]}],
		  "steps":[{"seq":1,"id":"plan","state":"SUCCEEDED","uses":"planner","node":"n-1",
		            "needs":[],"attempt":1,"exit_code":0}]}`))
	})
	run, err := m.GetRun(context.Background(), "itsaplan-EP-2-1")
	if err != nil {
		t.Fatal(err)
	}
	if rec.seen()[0].Path != "/v1/runs/itsaplan-EP-2-1" {
		t.Fatalf("path = %q", rec.seen()[0].Path)
	}
	if run.RunID != "itsaplan-EP-2-1" || run.State != "RUNNING" {
		t.Fatalf("run = %+v", run)
	}
	if len(run.Assigned) != 1 || run.Assigned[0].Nodes[0].Label != "host:orch" {
		t.Fatalf("the node labels were lost: %+v", run.Assigned)
	}
	if len(run.Steps) != 1 || run.Steps[0].ExitCode == nil || *run.Steps[0].ExitCode != 0 {
		t.Fatalf("the steps were lost: %+v", run.Steps)
	}
	// RUNNING 은 종료가 아니다 — follow 가 여기서 계속 돈다.
	if run.Terminal() {
		t.Fatal("a RUNNING run is reported as terminal")
	}
}

// 오케스트레이터가 아직 안 떴는데 Run 을 내면 422 다 (adapter-example §1.2 ⑤).
// 그래서 광고에 자기 이름표가 보일 때까지 기다린다 — 그 눈이 HasLabel 이다.
func TestMediator_HasLabelLooksForOneLabelOnOneCapability(t *testing.T) {
	const payload = `{"capabilities":[
	  {"capability":"orchestration","nodes":2,"attrs":{"issue":["EP-2","EP-3"]}},
	  {"capability":"agent.reason","nodes":1,"attrs":{"harness":["claude"],"issue":["EP-9"]}}]}`
	for _, tc := range []struct {
		name       string
		capability string
		key        string
		val        string
		want       bool
	}{
		{"our own orchestrator is advertised", "orchestration", "issue", "EP-2", true},
		{"a sibling issue on the same capability", "orchestration", "issue", "EP-3", true},
		{"an issue that has not appeared yet", "orchestration", "issue", "EP-9", false},
		{"the right label on the wrong capability", "orchestration", "harness", "claude", false},
		{"a capability nobody advertises", "agent.plan", "issue", "EP-2", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, rec := newMediatorClient(t, "", func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(payload))
			})
			got, err := m.HasLabel(context.Background(), tc.capability, tc.key, tc.val)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("HasLabel(%s, %s=%s) = %v, want %v",
					tc.capability, tc.key, tc.val, got, tc.want)
			}
			if rec.seen()[0].Path != "/v1/capabilities" {
				t.Fatalf("path = %q", rec.seen()[0].Path)
			}
		})
	}
}

// 광고를 못 읽는 것은 「아직 안 떴다」가 아니다 — 뭉개면 WaitAdvertised 가
// Mediator 장애를 기동 지연으로 읽고 상한까지 조용히 기다린다.
func TestMediator_HasLabelPropagatesALookupFailure(t *testing.T) {
	m, _ := newMediatorClient(t, "", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	ok, err := m.HasLabel(context.Background(), "orchestration", "issue", "EP-2")
	if err == nil {
		t.Fatal("an unreadable advertisement was reported as 'not there yet'")
	}
	if ok {
		t.Fatal("HasLabel = true on an error path")
	}
	if _, err := m.Capabilities(context.Background()); err == nil {
		t.Fatal("Capabilities swallowed a 503")
	}
}

// 인박스가 정본이다 — 알림은 보조다 (ADR-032 §4).
func TestMediator_AskForPicksThisRunsQuestion(t *testing.T) {
	const payload = `{"asks":[
	  {"run_id":"itsaplan-EP-3-1","seq":1,"step":"gate","prompt":"another issue"},
	  {"run_id":"itsaplan-EP-2-1","seq":4,"step":"gate","prompt":"ours",
	   "shown":[{"name":"plan","content":{"steps":[]}}],"proposes":{"verdict":"ok"}}]}`
	m, rec := newMediatorClient(t, "", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(payload))
	})

	ask, err := m.AskFor(context.Background(), "itsaplan-EP-2-1")
	if err != nil {
		t.Fatal(err)
	}
	if ask == nil {
		t.Fatal("our own open question was not found")
	}
	if ask.Seq != 4 || ask.Step != "gate" || ask.Prompt != "ours" {
		t.Fatalf("ask = %+v", ask)
	}
	if len(ask.Shown) != 1 || ask.Shown[0].Name != "plan" {
		t.Fatalf("what the human is approving was lost: %+v", ask.Shown)
	}
	if string(ask.Proposes) == "" || string(ask.Proposes) == "null" {
		t.Fatalf("proposes = %q", string(ask.Proposes))
	}
	if rec.seen()[0].Path != "/v1/asks" {
		t.Fatalf("path = %q", rec.seen()[0].Path)
	}

	// 남의 이슈만 열려 있으면 우리 것은 없는 것이다.
	none, err := m.AskFor(context.Background(), "itsaplan-EP-9-1")
	if err != nil {
		t.Fatal(err)
	}
	if none != nil {
		t.Fatalf("AskFor picked up another issue's question: %+v", none)
	}
}

func TestMediator_AskForPropagatesAnInboxFailure(t *testing.T) {
	m, _ := newMediatorClient(t, "", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	ask, err := m.AskFor(context.Background(), "itsaplan-EP-2-1")
	if err == nil {
		t.Fatal("an unreadable inbox was reported as 'no open question'")
	}
	if ask != nil {
		t.Fatalf("ask = %+v on an error path", ask)
	}
	if _, err := m.Asks(context.Background()); err == nil {
		t.Fatal("Asks swallowed a 500")
	}
}

// 본문이 곧 답이고 그대로 산출물이 된다 — 어댑터가 손대면 안 된다.
func TestMediator_AnswerAddressesTheStepAndPassesTheBodyThrough(t *testing.T) {
	m, rec := newMediatorClient(t, "taeels", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	body, err := json.Marshal(map[string]any{"verdict": "again", "note": "표를 더 자세히"})
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Answer(context.Background(), "itsaplan-EP-2-1", 4, body); err != nil {
		t.Fatal(err)
	}
	got := rec.seen()[0]
	if got.Method != "POST" || got.Path != "/v1/runs/itsaplan-EP-2-1/steps/4/answer" {
		t.Fatalf("%s %s is not the answer route", got.Method, got.Path)
	}
	if got.Body != string(body) {
		t.Fatalf("the answer was altered on the way out: %q", got.Body)
	}
	if got.Principal != "taeels" {
		t.Fatalf("the answer carries no principal - the seal would not say who answered")
	}
}

// 스키마 위반이면 422 이고 질문은 열린 채 남는다 — 어댑터가 그것을 사람에게
// 그대로 옮겨야 다시 답할 수 있다.
func TestMediator_AnswerSurfacesASchemaRejection(t *testing.T) {
	m, _ := newMediatorClient(t, "", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte("verdict must be one of ok|again"))
	})
	err := m.Answer(context.Background(), "itsaplan-EP-2-1", 4, []byte(`{"verdict":"maybe"}`))
	if err == nil {
		t.Fatal("a schema violation was swallowed")
	}
	if !strings.Contains(err.Error(), "verdict must be one of") {
		t.Fatalf("the reason did not reach the caller: %v", err)
	}
}

// 원장은 blobs/ 에서 유도된다 (ADR-040 §3.3 정정).
func TestMediator_LedgerReadsEveryEntry(t *testing.T) {
	m, rec := newMediatorClient(t, "", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"entries":[
		  {"seq":1,"attempt":1,"name":"plan","by":"planner","at":"2026-08-24T00:00:00Z",
		   "bytes":180,"schema_ok":true}]}`))
	})
	entries, err := m.Ledger(context.Background(), "itsaplan-EP-2-1")
	if err != nil {
		t.Fatal(err)
	}
	if rec.seen()[0].Path != "/v1/runs/itsaplan-EP-2-1/ledger" {
		t.Fatalf("path = %q", rec.seen()[0].Path)
	}
	if len(entries) != 1 || entries[0].Name != "plan" || entries[0].Bytes != 180 {
		t.Fatalf("entries = %+v", entries)
	}
	if entries[0].SchemaOK == nil || !*entries[0].SchemaOK {
		t.Fatalf("schema_ok = %v - unchecked and failed must stay distinct", entries[0].SchemaOK)
	}
	if _, err := m.Ledger(context.Background(), ""); err != nil {
		t.Fatalf("a bare ledger lookup failed: %v", err)
	}
}

func TestMediator_BlobReadsTheArtefact(t *testing.T) {
	m, rec := newMediatorClient(t, "", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("the report body"))
	})
	raw, err := m.Blob(context.Background(), "itsaplan-EP-2-1", "summary")
	if err != nil {
		t.Fatal(err)
	}
	got := rec.seen()[0]
	if got.Method != "GET" || got.Path != "/v1/runs/itsaplan-EP-2-1/blob/summary" {
		t.Fatalf("%s %s is not the blob route", got.Method, got.Path)
	}
	if got.Auth != "Bearer fleet-token" {
		t.Fatalf("the blob read carries no token: %q", got.Auth)
	}
	if string(raw) != "the report body" {
		t.Fatalf("blob = %q", string(raw))
	}
}

func TestMediator_BlobDistinguishesAbsenceFromFailure(t *testing.T) {
	t.Run("a missing artefact is the sentinel", func(t *testing.T) {
		m, _ := newMediatorClient(t, "", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		raw, err := m.Blob(context.Background(), "r", "summary")
		if !errors.Is(err, ErrNoRun) {
			t.Fatalf("err = %v, want ErrNoRun", err)
		}
		if raw != nil {
			t.Fatalf("blob = %q on a 404", string(raw))
		}
	})
	t.Run("any other rejection names the artefact", func(t *testing.T) {
		m, _ := newMediatorClient(t, "", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		})
		_, err := m.Blob(context.Background(), "r", "summary")
		if err == nil {
			t.Fatal("a 403 blob read was reported as success")
		}
		for _, want := range []string{"summary", "403"} {
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("error %q lost %q", err.Error(), want)
			}
		}
	})
	t.Run("a base URL that cannot be a request", func(t *testing.T) {
		m := NewMediator("://not-a-url", "t", "")
		if _, err := m.Blob(context.Background(), "r", "summary"); err == nil {
			t.Fatal("an unbuildable request was accepted")
		}
	})
	t.Run("a Mediator that is not listening", func(t *testing.T) {
		m := NewMediator("http://127.0.0.1:1", "t", "")
		if _, err := m.Blob(context.Background(), "r", "summary"); err == nil {
			t.Fatal("a transport failure was swallowed")
		}
	})
}

// 노드가 쓰는 것과 같은 HTTP 경로를 어댑터도 쓴다 (ADR-040 §5) — 내부 함수를
// 새로 안 뚫으므로 enode 코어가 한 줄도 안 바뀐다.
func TestMediator_PutBlobUsesTheSameRouteANodeWould(t *testing.T) {
	m, rec := newMediatorClient(t, "taeels", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	body := []byte(`{"system":"itsaplan","comment":314}`)
	if err := m.PutBlob(context.Background(), "itsaplan-EP-2-1", 4, "_outbound", body); err != nil {
		t.Fatal(err)
	}
	got := rec.seen()[0]
	if got.Method != "PUT" || got.Path != "/v1/runs/itsaplan-EP-2-1/steps/4/blob/_outbound" {
		t.Fatalf("%s %s is not the node's blob route", got.Method, got.Path)
	}
	if got.CType != "application/json" || got.Auth != "Bearer fleet-token" {
		t.Fatalf("headers = %+v", got)
	}
	if got.Principal != "taeels" {
		t.Fatal("the outbound record carries no principal")
	}
	if got.Body != string(body) {
		t.Fatalf("body = %q", got.Body)
	}
}

func TestMediator_PutBlobSurfacesEveryFailure(t *testing.T) {
	t.Run("a rejection keeps the status and the reason", func(t *testing.T) {
		m, _ := newMediatorClient(t, "", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte("  _outbound already exists  "))
		})
		err := m.PutBlob(context.Background(), "r", 4, "_outbound", []byte(`{}`))
		if err == nil {
			t.Fatal("a 409 on PutBlob was swallowed - answer matching would silently weaken")
		}
		for _, want := range []string{"_outbound", "409", "already exists"} {
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("error %q lost %q", err.Error(), want)
			}
		}
	})
	t.Run("a base URL that cannot be a request", func(t *testing.T) {
		m := NewMediator("://not-a-url", "t", "")
		if err := m.PutBlob(context.Background(), "r", 4, "_outbound", []byte(`{}`)); err == nil {
			t.Fatal("an unbuildable request was accepted")
		}
	})
	t.Run("a Mediator that is not listening", func(t *testing.T) {
		m := NewMediator("http://127.0.0.1:1", "t", "")
		if err := m.PutBlob(context.Background(), "r", 4, "_outbound", []byte(`{}`)); err == nil {
			t.Fatal("a transport failure was swallowed")
		}
	})
}
