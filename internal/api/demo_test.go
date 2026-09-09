package api

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
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/taeels/enode/internal/config"
	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/record"
	"github.com/taeels/enode/internal/store"
)

func demoTestSubmission() demoSubmission {
	return demoSubmission{"welcome-audio", "\ubc1d\uc740 \uc218\ub2ec", "04995a39-1fba-4c16-b2cd-dc80ff21257e"}
}

func demoTestBody(d demoSubmission) string {
	b, _ := json.Marshal(d)
	return string(b)
}

func TestDemo_RejectsMalformedOrOverbroadRequests(t *testing.T) {
	valid := demoTestBody(demoTestSubmission())
	cases := []struct {
		name, body, contentType string
		want                    int
	}{
		{"valid", valid, "application/json", 0},
		{"utf8 charset", valid, "application/json; charset=UTF-8", 0},
		{"missing content type", valid, "", 415},
		{"form content type", valid, "application/x-www-form-urlencoded", 415},
		{"invalid content type", valid, "application/json; charset", 415},
		{"unsupported charset", valid, "application/json; charset=utf-16", 415},
		{"empty body", "", "application/json", 400},
		{"null body", "null", "application/json", 400},
		{"array body", "[" + valid + "]", "application/json", 400},
		{"unfinished body", valid[:len(valid)-1], "application/json", 400},
		{"trailing object", valid + "{}", "application/json", 400},
		{"trailing junk", valid + "!", "application/json", 400},
		{"duplicate key", `{"scenario_id":"led-toggle",` + valid[1:], "application/json", 400},
		{"escaped duplicate key", `{"scenario_\u0069d":"led-toggle",` + valid[1:], "application/json", 400},
		{"unknown contract key", `{"steps":[],` + valid[1:], "application/json", 400},
		{"case sensitive key", strings.Replace(valid, "scenario_id", "Scenario_ID", 1), "application/json", 400},
		{"missing field", `{"scenario_id":"led-toggle","submitter":"guest-old-name"}`, "application/json", 400},
		{"wrong field type", strings.Replace(valid, `"welcome-audio"`, "42", 1), "application/json", 400},
		{"null field", strings.Replace(valid, `"welcome-audio"`, "null", 1), "application/json", 400},
		{"invalid utf8", valid + string([]byte{0xff}), "application/json", 400},
		{"limit inclusive", valid + strings.Repeat(" ", demoBodyLimit-len(valid)), "application/json", 0},
		{"limit exceeded", valid + strings.Repeat(" ", demoBodyLimit-len(valid)+1), "application/json", 413},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/v1/demo/runs", strings.NewReader(tc.body))
			r.Header.Set("Content-Type", tc.contentType)
			_, got := readDemoSubmission(httptest.NewRecorder(), r)
			if got != tc.want {
				t.Fatalf("status=%d, want %d", got, tc.want)
			}
		})
	}
}

func TestDemo_IdentityAndFieldBoundaries(t *testing.T) {
	base := demoTestSubmission()
	for _, tc := range []struct {
		name string
		edit func(*demoSubmission)
		want int
	}{
		{"legacy name", func(d *demoSubmission) { d.Submitter = "guest-bright-otter" }, 0},
		{"maximum korean name", func(d *demoSubmission) {
			d.Submitter = strings.Repeat("\uac00", 12) + " " + strings.Repeat("\ub098", 12)
		}, 0},
		{"maximum legacy name", func(d *demoSubmission) {
			d.Submitter = "guest-" + strings.Repeat("a", 24) + "-" + strings.Repeat("b", 24)
		}, 0},
		{"long korean word", func(d *demoSubmission) { d.Submitter = strings.Repeat("\uac00", 13) + " \ub098" }, 400},
		{"long legacy word", func(d *demoSubmission) { d.Submitter = "guest-" + strings.Repeat("a", 25) + "-b" }, 400},
		{"two spaces", func(d *demoSubmission) { d.Submitter = "\ubc1d\uc740  \uc218\ub2ec" }, 400},
		{"leading whitespace", func(d *demoSubmission) { d.Submitter = " " + d.Submitter }, 400},
		{"shell input", func(d *demoSubmission) { d.Submitter = "$(touch /tmp/untrusted)" }, 400},
		{"unknown scenario", func(d *demoSubmission) { d.ScenarioID = "command" }, 400},
		{"path input", func(d *demoSubmission) { d.ScenarioID = "../command" }, 400},
		{"uppercase uuid", func(d *demoSubmission) { d.RequestID = strings.ToUpper(d.RequestID) }, 400},
		{"wrong uuid version", func(d *demoSubmission) { d.RequestID = "04995a39-1fba-5c16-b2cd-dc80ff21257e" }, 400},
		{"wrong uuid variant", func(d *demoSubmission) { d.RequestID = "04995a39-1fba-4c16-72cd-dc80ff21257e" }, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := base
			tc.edit(&d)
			r := httptest.NewRequest("POST", "/v1/demo/runs", strings.NewReader(demoTestBody(d)))
			r.Header.Set("Content-Type", "application/json")
			_, got := readDemoSubmission(httptest.NewRecorder(), r)
			if got != tc.want {
				t.Fatalf("status=%d, want %d", got, tc.want)
			}
		})
	}
	// 독립적으로 계산한 정규 JSON 배열의 SHA-256을 고정 기대값으로 둔다.
	if got := base.runID(); got != "demo-84c84edb6ce70f3a2e1d802b6c286c6f3120162c9bd8aeaa639404dd4833c689" {
		t.Fatalf("canonical run ID=%s", got)
	}
	identities := map[string]bool{base.runID(): true}
	for _, edit := range []func(*demoSubmission){
		func(d *demoSubmission) { d.ScenarioID = "led-toggle" },
		func(d *demoSubmission) { d.Submitter = "guest-bright-otter" },
		func(d *demoSubmission) { d.RequestID = "04995a39-1fba-4c16-b2cd-dc80ff21257f" },
	} {
		d := base
		edit(&d)
		if identities[d.runID()] {
			t.Fatal("distinct intent reused a run ID")
		}
		identities[d.runID()] = true
	}
}

// command 예제는 접수 시험에만 쓴다. 음원/LED의 실제 계약으로 등록하지 않는다.
func demoTestScenario() demoScenario {
	return demoScenario{example: "command", inject: func(c *contract.Contract, name string) error {
		c.Steps[0].Run = append(c.Steps[0].Run, name)
		return nil
	}}
}

func TestDemo_PreparesIndependentContractsAndFailsClosed(t *testing.T) {
	d := demoTestSubmission()
	raw, err := contract.Example("command")
	if err != nil {
		t.Fatal(err)
	}
	scenario := demoTestScenario()
	first, err := scenario.prepare(d)
	if err != nil {
		t.Fatal(err)
	}
	var original contract.Contract
	if err := json.Unmarshal(raw, &original); err != nil {
		t.Fatal(err)
	}
	if first.RunID != d.runID() || first.Work.Key() != "manual:"+d.runID() || first.Work.ChangeID != d.runID() || first.Steps[0].Run[3] != d.Submitter {
		t.Fatalf("prepared identity or name mismatch: %+v", first)
	}
	if !reflect.DeepEqual(first.Requires, original.Requires) || !reflect.DeepEqual(first.SuccessWhen, original.SuccessWhen) || !reflect.DeepEqual(first.Steps[0].Run[:3], original.Steps[0].Run) {
		t.Fatal("preparation changed requirements, success criteria, or shell program")
	}
	first.Requires[0].As = "changed"
	first.Steps[0].Run[2] = "changed"
	second, err := scenario.prepare(d)
	if err != nil || second.Requires[0].As != original.Requires[0].As || second.Steps[0].Run[2] != original.Steps[0].Run[2] {
		t.Fatalf("requests shared mutable state: %v", err)
	}
	again, err := contract.Example("command")
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal("source example changed")
	}
	for _, tc := range []struct {
		name string
		def  demoScenario
	}{
		{"unconfigured", demoScenario{}},
		{"missing injector", demoScenario{example: "command"}},
		{"missing example", demoScenario{example: "not-present", inject: scenario.inject}},
		{"failed injection", demoScenario{example: "command", inject: func(*contract.Contract, string) error { return errors.New("invalid injection point") }}},
		{"invalid contract", demoScenario{example: "command", inject: func(c *contract.Contract, _ string) error { c.Steps = nil; return nil }}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := tc.def.prepare(d); err == nil {
				t.Fatal("invalid scenario was prepared")
			}
		})
	}
}

func demoTestServer(t *testing.T) *Server {
	t.Helper()
	dsn := os.Getenv("ENODE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("ENODE_TEST_DATABASE_URL is unset; use scripts/testdb.sh")
	}
	ctx := context.Background()
	st, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := st.Truncate(ctx); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	t.Cleanup(func() {
		_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return os.Chmod(path, 0o755)
			}
			return os.Chmod(path, 0o644)
		})
	})
	st.Records = record.New(root)
	cfg := config.Default()
	cfg.Token, cfg.Demo = "demo-server-test-token", true
	return New(st, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func demoTestCall(handler http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}

func demoTestStatus(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()
	if w.Code != want {
		t.Fatalf("status=%d body=%s, want %d", w.Code, w.Body.String(), want)
	}
}

func TestDemo_RouteConfigurationAndIndependentLimit(t *testing.T) {
	s := demoTestServer(t)
	body := demoTestBody(demoTestSubmission())
	s.cfg.Demo = false
	demoTestStatus(t, demoTestCall(s.Handler(), "POST", "/v1/demo/runs", body, ""), 404)
	s.cfg.Demo = true
	public := s.Handler()
	for _, bad := range []struct {
		body string
		want int
	}{{`{"steps":[]}`, 400}, {strings.Repeat(" ", demoBodyLimit+1), 413}} {
		w := demoTestCall(s.newDemoHandler(demoScenarios()), "POST", "/v1/demo/runs", bad.body, "")
		demoTestStatus(t, w, bad.want)
		if strings.Contains(w.Body.String(), bad.body) || strings.Contains(w.Body.String(), s.cfg.Token) {
			t.Fatal("invalid request response leaked input or token")
		}
	}
	for _, id := range []string{"led-toggle", "welcome-audio"} {
		d := demoTestSubmission()
		d.ScenarioID = id
		w := demoTestCall(public, "POST", "/v1/demo/runs", demoTestBody(d), "")
		demoTestStatus(t, w, 422)
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("missing no-store response")
		}
	}
	demoTestStatus(t, demoTestCall(public, "GET", "/v1/demo/runs", "", ""), 405)
	for _, path := range []string{"/v1/runs", "/v1/nodes"} {
		demoTestStatus(t, demoTestCall(public, "POST", path, "{}", ""), 401)
	}
	demoTestStatus(t, demoTestCall(public, "GET", "/v1/asks", "", ""), 401)
	h := s.newDemoHandler(map[string]demoScenario{})
	now := time.Now()
	h.limit.now = func() time.Time { return now }
	for range 4 {
		demoTestStatus(t, demoTestCall(h, "POST", "/v1/demo/runs", body, ""), 503)
	}
	w := demoTestCall(h, "POST", "/v1/demo/runs", body, "")
	demoTestStatus(t, w, 429)
	if w.Header().Get("Retry-After") != "1" {
		t.Fatal("missing Retry-After")
	}
	demoTestStatus(t, demoTestCall(public, "GET", "/v1/runs", "", ""), 200)
	now = now.Add(time.Second)
	for range 2 {
		demoTestStatus(t, demoTestCall(h, "POST", "/v1/demo/runs", body, ""), 503)
	}
	demoTestStatus(t, demoTestCall(h, "POST", "/v1/demo/runs", body, ""), 429)
	s.cfg.Token = ""
	w = demoTestCall(s.newDemoHandler(map[string]demoScenario{"welcome-audio": demoTestScenario()}), "POST", "/v1/demo/runs", body, "browser-token")
	demoTestStatus(t, w, 503)
	rows, _, err := s.st.Runs(context.Background(), store.RunFilter{Limit: 100})
	if err != nil || len(rows) != 2 {
		t.Fatalf("unconfigured requests added runs beyond the two rejected scenarios: %v %v", rows, err)
	}
}

func TestDemo_AdmissionRetriesPromotionAndRejection(t *testing.T) {
	s := demoTestServer(t)
	public := s.Handler()
	demoTestStatus(t, demoTestCall(public, "POST", "/v1/nodes", `{"node_id":"demo-node","label":"Demo test node","capabilities":[{"capability":"agent.reason","attrs":{}}]}`, s.cfg.Token), 200)
	defs := map[string]demoScenario{"led-toggle": demoTestScenario(), "welcome-audio": demoTestScenario()}
	h := s.newDemoHandler(defs)
	// 등록 뒤 외부 map 변경은 실행 중 라우트의 매핑을 바꾸지 않는다.
	delete(defs, "welcome-audio")
	d := demoTestSubmission()
	r := httptest.NewRequest("POST", "/v1/demo/runs?browser=value", strings.NewReader(demoTestBody(d)))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer browser-token")
	r.Header.Set("X-Enode-Principal", "browser-principal")
	r.Header.Set("Cookie", "browser=value")
	before := r.Header.Clone()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	demoTestStatus(t, w, 201)
	if !reflect.DeepEqual(before, r.Header) || r.URL.Path != "/v1/demo/runs" || r.URL.RawQuery != "browser=value" {
		t.Fatal("incoming request was mutated")
	}
	if strings.Contains(w.Body.String(), s.cfg.Token) || strings.Contains(w.Body.String(), "browser-token") {
		t.Fatal("token leaked in the response")
	}
	first, err := s.st.GetRun(context.Background(), d.runID())
	if err != nil || first.Principal != "" || first.Contract.Steps[0].Run[3] != d.Submitter {
		t.Fatalf("identity or injection mismatch: %+v %v", first, err)
	}
	queued := d
	queued.RequestID = "04995a39-1fba-4c16-b2cd-dc80ff21257f"
	w = demoTestCall(h, "POST", "/v1/demo/runs", demoTestBody(queued), "")
	demoTestStatus(t, w, 202)
	// 응답을 잃었다고 가정하고 같은 세 값으로 재접수한다.
	demoTestStatus(t, demoTestCall(h, "POST", "/v1/demo/runs", demoTestBody(queued), ""), 200)
	demoTestStatus(t, demoTestCall(public, "POST", "/v1/runs/"+d.runID()+"/cancel", "", s.cfg.Token), 200)
	promoted, err := s.st.GetRun(context.Background(), queued.runID())
	if err != nil || promoted.State != store.StateRunning {
		t.Fatalf("queued run was not promoted: %+v %v", promoted, err)
	}
	// 핸들러 재생성 뒤에도 같은 의도는 같은 행으로 돌아온다.
	h = s.newDemoHandler(map[string]demoScenario{"welcome-audio": demoTestScenario()})
	demoTestStatus(t, demoTestCall(h, "POST", "/v1/demo/runs", demoTestBody(queued), ""), 200)
	rows, _, err := s.st.Runs(context.Background(), store.RunFilter{Limit: 100})
	if err != nil || len(rows) != 2 {
		t.Fatalf("retry duplicated rows: %v %v", rows, err)
	}
	for _, row := range rows {
		if row.Submitter != d.Submitter {
			t.Fatalf("submitter lost: %+v", row)
		}
	}
	different := queued
	different.Submitter = "guest-different-name"
	demoTestStatus(t, demoTestCall(h, "POST", "/v1/demo/runs", demoTestBody(different), ""), 202)

	t.Run("rejection keeps the original result", func(t *testing.T) {
		s := demoTestServer(t)
		h := s.newDemoHandler(map[string]demoScenario{"welcome-audio": demoTestScenario()})
		demoTestStatus(t, demoTestCall(h, "POST", "/v1/demo/runs", demoTestBody(d), ""), 422)
		w := demoTestCall(h, "POST", "/v1/demo/runs", demoTestBody(d), "")
		demoTestStatus(t, w, 200)
		if !strings.Contains(w.Body.String(), `"state":"FAILED"`) {
			t.Fatal("rejected retry was not FAILED")
		}
	})
}

func TestDemo_ConcurrentRetryCreatesOneRun(t *testing.T) {
	s := demoTestServer(t)
	demoTestStatus(t, demoTestCall(s.Handler(), "POST", "/v1/nodes", `{"node_id":"demo-node","label":"Demo test node","capabilities":[{"capability":"agent.reason","attrs":{}}]}`, s.cfg.Token), 200)
	h := s.newDemoHandler(map[string]demoScenario{"welcome-audio": demoTestScenario()})
	d := demoTestSubmission()
	var wg sync.WaitGroup
	responses := make(chan *httptest.ResponseRecorder, 4)
	start := make(chan struct{})
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			responses <- demoTestCall(h, "POST", "/v1/demo/runs", demoTestBody(d), "")
		}()
	}
	close(start)
	wg.Wait()
	close(responses)
	created := 0
	for w := range responses {
		if w.Code == 201 {
			created++
		} else {
			demoTestStatus(t, w, 200)
		}
		if !strings.Contains(w.Body.String(), d.runID()) {
			t.Fatal("concurrent retry returned a different run")
		}
	}
	if created != 1 {
		t.Fatalf("created responses=%d, want 1", created)
	}
	rows, _, err := s.st.Runs(context.Background(), store.RunFilter{Limit: 100})
	if err != nil || len(rows) != 1 || rows[0].Submitter != d.Submitter {
		t.Fatalf("concurrent retry rows: %v %v", rows, err)
	}
}

func TestDemo_CancelledSubmissionCanBeRetried(t *testing.T) {
	s := demoTestServer(t)
	demoTestStatus(t, demoTestCall(s.Handler(), "POST", "/v1/nodes", `{"node_id":"demo-node","label":"Demo test node","capabilities":[{"capability":"agent.reason","attrs":{}}]}`, s.cfg.Token), 200)
	h := s.newDemoHandler(map[string]demoScenario{"welcome-audio": demoTestScenario()})
	d := demoTestSubmission()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := httptest.NewRequest("POST", "/v1/demo/runs", strings.NewReader(demoTestBody(d))).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	demoTestStatus(t, w, 503)
	rows, _, err := s.st.Runs(context.Background(), store.RunFilter{Limit: 100})
	if err != nil || len(rows) != 0 {
		t.Fatalf("cancelled request created rows: %v %v", rows, err)
	}
	demoTestStatus(t, demoTestCall(h, "POST", "/v1/demo/runs", demoTestBody(d), ""), 201)
	demoTestStatus(t, demoTestCall(h, "POST", "/v1/demo/runs", demoTestBody(d), ""), 200)
}

func TestDemo_FixedFixturesOnlyChangeIdentityAndApprovedName(t *testing.T) {
	for id, definition := range demoScenarios() {
		t.Run(id, func(t *testing.T) {
			d := demoTestSubmission()
			d.ScenarioID = id
			raw, err := contract.Example(definition.example)
			if err != nil {
				t.Fatal(err)
			}
			var original contract.Contract
			if err := json.Unmarshal(raw, &original); err != nil {
				t.Fatal(err)
			}
			prepared, err := definition.prepare(d)
			if err != nil {
				t.Fatal(err)
			}
			if prepared.RunID != d.runID() || prepared.Work.Key() != "manual:"+d.runID() || prepared.Ledger != nil {
				t.Fatal("fixture did not preserve request isolation and default run ledger scope")
			}
			if id == "welcome-audio" {
				prompt := prepared.Steps[0].In["prompt"].(string)
				if !strings.Contains(prompt, d.Submitter+"\ub2d8 \ud658\uc601\ud569\ub2c8\ub2e4") || strings.Contains(prompt, "{{submitter}}") {
					t.Fatal("welcome prompt did not receive the exact submitted name")
				}
				prepared.Steps[0].In["prompt"] = original.Steps[0].In["prompt"]
			}
			prepared.RunID, prepared.Work = original.RunID, original.Work
			if !reflect.DeepEqual(prepared, original) {
				t.Fatal("fixture changed outside identity and the approved prompt field")
			}
			after, err := contract.Example(definition.example)
			if err != nil || !bytes.Equal(raw, after) {
				t.Fatal("embedded fixture changed")
			}
		})
	}
	for _, tc := range []struct {
		name string
		edit func(*contract.Contract)
	}{
		{"missing step", func(c *contract.Contract) { c.Steps[0].ID = "other" }},
		{"missing prompt", func(c *contract.Contract) { c.Steps[0].In = nil }},
		{"non-string prompt", func(c *contract.Contract) { c.Steps[0].In["prompt"] = 42 }},
		{"not an agent", func(c *contract.Contract) { c.Steps[0].Agent = nil }},
		{"missing placeholder", func(c *contract.Contract) { c.Steps[0].In["prompt"] = "No placeholder" }},
		{"repeated placeholder", func(c *contract.Contract) { c.Steps[0].In["prompt"] = "{{submitter}} {{submitter}}" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, _ := contract.Example("demo-welcome-audio")
			var c contract.Contract
			if err := json.Unmarshal(raw, &c); err != nil {
				t.Fatal(err)
			}
			tc.edit(&c)
			if err := injectDemoWelcome(&c, demoTestSubmission().Submitter); err == nil {
				t.Fatal("invalid injection point accepted")
			}
		})
	}
}

func TestDemo_PublicFixedScenariosUseQueueAndPreserveWelcomeName(t *testing.T) {
	s := demoTestServer(t)
	h := s.Handler()
	board := `{"node_id":"demo-board","label":"Demo board","capabilities":[{"capability":"agent.reason","attrs":{"device":"led","board":"rpi2b-v1.1"}}]}`
	voice := `{"node_id":"demo-voice","label":"Demo voice","capabilities":[{"capability":"agent.reason","attrs":{"harness":"claude","service":"tts","tts_typecast":"yes","voice_typecast":"Sanghyun"}}]}`
	for _, ad := range []string{board, voice} {
		demoTestStatus(t, demoTestCall(h, "POST", "/v1/nodes", ad, s.cfg.Token), 200)
	}
	led := demoTestSubmission()
	led.ScenarioID = "led-toggle"
	demoTestStatus(t, demoTestCall(h, "POST", "/v1/demo/runs", demoTestBody(led), ""), 201)
	audio := demoTestSubmission()
	demoTestStatus(t, demoTestCall(h, "POST", "/v1/demo/runs", demoTestBody(audio), ""), 202)
	demoTestStatus(t, demoTestCall(h, "POST", "/v1/demo/runs", demoTestBody(audio), ""), 200)
	demoTestStatus(t, demoTestCall(h, "POST", "/v1/runs/"+led.runID()+"/cancel", "", s.cfg.Token), 200)
	read := demoTestCall(h, "GET", "/v1/runs", "", "")
	demoTestStatus(t, read, 200)
	var list runsResponse
	if err := json.Unmarshal(read.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Runs) != 2 {
		t.Fatalf("got %d runs, want 2", len(list.Runs))
	}
	for _, row := range list.Runs {
		if row.Submitter != audio.Submitter {
			t.Fatalf("name was lost: %+v", row)
		}
	}
	run, err := s.st.GetRun(context.Background(), audio.runID())
	if err != nil || run.State != store.StateRunning {
		t.Fatalf("audio did not promote: %+v %v", run, err)
	}
	prompt, _ := run.Contract.Steps[0].In["prompt"].(string)
	if !strings.Contains(prompt, audio.Submitter+"\ub2d8 \ud658\uc601\ud569\ub2c8\ub2e4") {
		t.Fatal("stored audio input and list submitter differ")
	}
	if !reflect.DeepEqual(run.Contract.Steps[1].Needs, []string{"synthesize"}) || !reflect.DeepEqual(run.Contract.Steps[1].In["from"], []any{"welcome.wav"}) {
		t.Fatal("cross-node artifact dependency changed")
	}
}
