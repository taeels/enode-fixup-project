package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/taeels/enode/internal/store"
)

const galleryProof = "04995a39-1fba-4c16-b2cd-dc80ff21257e"

func galleryTestJob() galleryJob {
	return galleryJob{Operation: "draft", ProjectID: "project-1", Prompt: "댓글을 써줘", OwnerHash: galleryHash(galleryProof), Submitter: "밝은 수달"}
}
func galleryTestHandler(t *testing.T) *galleryHandler {
	s := demoTestServer(t)
	return &galleryHandler{server: s, limit: newBucket(10000, 10000), projects: []galleryProject{{ID: "project-1", Title: "Other project", TeamName: "Another team"}}, cached: time.Now()}
}
func galleryCall(h http.HandlerFunc, method, path, body, proof string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Gallery-Request-ID", proof)
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 5 {
		r.SetPathValue("id", parts[4])
	}
	w := httptest.NewRecorder()
	h(w, r)
	return w
}
func galleryFixture(t *testing.T, h *galleryHandler, id string, result galleryResult) *store.Run {
	t.Helper()
	c, err := galleryContract(id, galleryTestJob())
	if err != nil {
		t.Fatal(err)
	}
	run := &store.Run{RunID: id, State: store.StateSucceeded, Contract: c, Submitter: "밝은 수달"}
	if err = h.server.st.CreateRun(context.Background(), *run, nil, nil); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(result)
	if _, err = h.server.records.WriteBlob(id, 1, 1, "gallery-result.json", strings.NewReader(string(raw)), 32768); err != nil {
		t.Fatal(err)
	}
	if err = h.server.records.Seal(id, map[string]string{}, map[string]string{}, nil); err != nil {
		t.Fatal(err)
	}
	return run
}
func galleryDraftBody() string {
	return `{"project_id":"project-1","prompt":"댓글 써줘","request_id":"` + galleryProof + `","submitter":"밝은 수달"}`
}
func TestGallery_ExactFieldsAndText(t *testing.T) {
	valid := galleryDraftBody()
	for _, raw := range []string{`null`, valid + `{}`, strings.Replace(valid, `"project_id"`, `"Project_ID"`, 1), `{"project_id":"x",` + valid[1:], `{"url":"https://evil.test",` + valid[1:], strings.Replace(valid, `"댓글 써줘"`, `null`, 1), strings.Repeat(" ", 8193)} {
		r := httptest.NewRequest("POST", "/", strings.NewReader(raw))
		r.Header.Set("Content-Type", "application/json")
		if _, status := galleryFields(httptest.NewRecorder(), r, "project_id", "prompt", "request_id", "submitter"); status == 0 {
			t.Fatalf("accepted malformed body %q", raw)
		}
	}
	if galleryText(" ", 500) || galleryText("x\x00", 500) || galleryText(strings.Repeat("가", 501), 500) || !galleryText(strings.Repeat("가", 500), 500) {
		t.Fatal("incorrect text boundary")
	}
}
func TestGallery_ContractDoesNotTurnPromptIntoCommand(t *testing.T) {
	job := galleryTestJob()
	job.Prompt = "$(touch /tmp/pwn); https://evil.test"
	id := "gallery-" + strings.Repeat("a", 64)
	c, err := galleryContract(id, job)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Steps) != 1 || len(c.Steps[0].Run) != 3 || c.Steps[0].Run[0] != "/usr/bin/python3" || c.Requires[0].Attrs["gallery"] != "comments-v1" {
		t.Fatal("overbroad command")
	}
	raw, _ := base64.RawURLEncoding.DecodeString(c.Steps[0].Run[2])
	if strings.Contains(string(raw), galleryProof) {
		t.Fatal("raw proof in contract")
	}
	run := store.Run{RunID: id, Contract: c}
	parsed, err := galleryOriginal(&run)
	if err != nil || parsed.Prompt != job.Prompt {
		t.Fatal("original contract rejected", err)
	}
	run.Contract.Steps[0].Run[0] = "sh"
	if _, err = galleryOriginal(&run); err == nil {
		t.Fatal("accepted altered command")
	}
}
func TestGallery_DefaultClosed(t *testing.T) {
	h := galleryTestHandler(t)
	for _, flags := range [][2]bool{{false, false}, {true, false}, {false, true}} {
		h.server.cfg.Demo, h.server.cfg.DemoGallery = flags[0], flags[1]
		demoTestStatus(t, demoTestCall(h.server.Handler(), "POST", "/v1/demo/gallery/runs", galleryDraftBody(), ""), 404)
	}
	h.server.cfg.Demo = true
	h.server.cfg.DemoGallery = true
	h.server.cfg.Token = ""
	demoTestStatus(t, demoTestCall(h.server.Handler(), "POST", "/v1/demo/gallery/runs", galleryDraftBody(), ""), 404)
}
func TestGallery_DraftAdmissionRetryAndOwner(t *testing.T) {
	h := galleryTestHandler(t)
	node := `{"node_id":"gallery-test-node","label":"gallery","capabilities":[{"capability":"agent.reason","attrs":{"gallery":"comments-v1","sandbox":"lima-vm","provider":"bedrock","instance":"macmini-bedrock-vm"}}]}`
	demoTestStatus(t, demoTestCall(h.server.Handler(), "POST", "/v1/nodes", node, h.server.cfg.Token), 200)
	w := galleryCall(h.draft, "POST", "/v1/demo/gallery/runs", galleryDraftBody(), "")
	demoTestStatus(t, w, 201)
	var run runView
	_ = json.Unmarshal(w.Body.Bytes(), &run)
	retry := galleryCall(h.draft, "POST", "/v1/demo/gallery/runs", galleryDraftBody(), "")
	demoTestStatus(t, retry, 200)
	rows, _, err := h.server.st.Runs(context.Background(), store.RunFilter{Work: "manual:" + galleryWork, Limit: 10})
	if err != nil || len(rows) != 1 || rows[0].Submitter != "밝은 수달" {
		t.Fatal("submitter lost", err)
	}
	path := "/v1/demo/gallery/runs/" + run.RunID
	demoTestStatus(t, galleryCall(h.result, "GET", path, "", galleryProof), 200)
	demoTestStatus(t, galleryCall(h.result, "GET", path, "", "11111111-1111-4111-8111-111111111111"), 404)
	demoTestStatus(t, galleryCall(h.result, "GET", path, "", ""), 404)
	if strings.Contains(w.Body.String(), h.server.cfg.Token) || strings.Contains(w.Body.String(), galleryProof) {
		t.Fatal("private value exposed")
	}
	for i := 0; i < 2; i++ {
		body := strings.Replace(galleryDraftBody(), "댓글 써줘", "댓글 "+strings.Repeat("가", i+1), 1)
		demoTestStatus(t, galleryCall(h.draft, "POST", "/v1/demo/gallery/runs", body, ""), 202)
	}
	limited := galleryCall(h.draft, "POST", "/v1/demo/gallery/runs", strings.Replace(galleryDraftBody(), "댓글 써줘", "새 댓글", 1), "")
	demoTestStatus(t, limited, 429)
	demoTestStatus(t, galleryCall(h.draft, "POST", "/v1/demo/gallery/runs", galleryDraftBody(), ""), 200)
}
func TestGallery_ConfirmedPublicationPinnedAndConcurrentIdempotent(t *testing.T) {
	h := galleryTestHandler(t)
	id := "gallery-" + strings.Repeat("b", 64)
	galleryFixture(t, h, id, galleryResult{Outcome: "draft_ready", Message: "초안", ProjectID: "project-1", Body: "초안 내용", Transcript: []galleryEvent{{Role: "assistant", Text: "초안 내용"}}})
	path := "/v1/demo/gallery/runs/" + id + "/publish"
	confirmation := `{"request_id":"` + galleryProof + `","body":"확인한 댓글"}`
	demoTestStatus(t, galleryCall(h.publish, "POST", path, strings.Replace(confirmation, galleryProof, "11111111-1111-4111-8111-111111111111", 1), ""), 404)
	// No matching node: existing submit records FAILED/422. A second POST still reuses that Run.
	first := galleryCall(h.publish, "POST", path, confirmation, "")
	demoTestStatus(t, first, 422)
	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() {
			w := galleryCall(h.publish, "POST", path, confirmation, "")
			if w.Code != 200 {
				t.Errorf("retry status=%d", w.Code)
			}
		})
	}
	wg.Wait()
	demoTestStatus(t, galleryCall(h.publish, "POST", path, strings.Replace(confirmation, "확인한 댓글", "바꾼 댓글", 1), ""), 409)
	stored, err := h.server.st.GetRun(context.Background(), "gallery-post-"+strings.Repeat("b", 64))
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := base64.RawURLEncoding.DecodeString(stored.Contract.Steps[0].Run[2])
	var job galleryJob
	_ = json.Unmarshal(raw, &job)
	if job.Body != "확인한 댓글" || job.ProjectID != "project-1" || job.Operation != "publish" {
		t.Fatal("confirmation changed")
	}
	parts := strings.Split(job.Ticket, ".")
	if len(parts) != 2 || parts[1] != gallerySign(h.signingKey(), parts[0]) {
		t.Fatal("invalid signature")
	}
	payload, _ := base64.RawURLEncoding.DecodeString(parts[0])
	if !strings.Contains(string(payload), "확인한 댓글") || strings.Contains(string(payload), galleryProof) {
		t.Fatal("wrong ticket")
	}
	result := galleryCall(h.result, "GET", "/v1/demo/gallery/runs/"+id, "", galleryProof)
	demoTestStatus(t, result, 200)
	if strings.Contains(result.Body.String(), job.Ticket) {
		t.Fatal("ticket exposed")
	}
}
func TestGallery_RefusalCannotPublishAndUnsafeArtifactRejected(t *testing.T) {
	h := galleryTestHandler(t)
	for i, result := range []galleryResult{{Outcome: "refused", Message: "범위 밖 요청입니다.", ProjectID: "project-1", Transcript: []galleryEvent{}}, {Outcome: "draft_ready", Message: "초안", ProjectID: "other-project", Body: "본문"}, {Outcome: "draft_ready", Message: "초안", ProjectID: "project-1", Body: "본문", Transcript: []galleryEvent{{Role: "assistant", Tool: "shell", Text: "bad"}}}} {
		id := "gallery-" + strings.Repeat(string(rune('c'+i)), 64)
		galleryFixture(t, h, id, result)
		w := galleryCall(h.publish, "POST", "/v1/demo/gallery/runs/"+id+"/publish", `{"request_id":"`+galleryProof+`","body":"댓글"}`, "")
		demoTestStatus(t, w, 409)
	}
}

type galleryTransport func(*http.Request) (*http.Response, error)

func (f galleryTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestGallery_CatalogOnlyFixedURLAndCache(t *testing.T) {
	calls := 0
	h := &galleryHandler{client: &http.Client{Transport: galleryTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != galleryOrigin+"/api/projects" || r.Method != "GET" {
			t.Fatal("unexpected request")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"projects":[{"id":"project-1","title":"Title","teamName":"Team"}]}`))}, nil
	})}}
	for range 2 {
		projects, err := h.catalog(context.Background())
		if err != nil || len(projects) != 1 {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Fatal("cache missed")
	}
}

func TestGallery_CommentMessageIsOneScopedWorkflow(t *testing.T) {
	h := galleryTestHandler(t)
	node := `{"node_id":"gallery-test-node","label":"gallery","capabilities":[{"capability":"agent.reason","attrs":{"gallery":"comments-v1","sandbox":"lima-vm","provider":"bedrock","instance":"macmini-bedrock-vm"}}]}`
	demoTestStatus(t, demoTestCall(h.server.Handler(), "POST", "/v1/nodes", node, h.server.cfg.Token), 200)
	body := `{"prompt":"다른 팀에 응원 댓글 달아줘","request_id":"` + galleryProof + `","submitter":"밝은 수달"}`
	for _, invalid := range []string{`{}`, strings.Replace(body, `"prompt"`, `"project_id":"project-1","prompt"`, 1), strings.Replace(body, "밝은 수달", "invalid", 1)} {
		demoTestStatus(t, galleryCall(h.comment, "POST", "/v1/demo/gallery/comments", invalid, ""), 400)
	}
	first := galleryCall(h.comment, "POST", "/v1/demo/gallery/comments", body, "")
	demoTestStatus(t, first, 201)
	var view runView
	if err := json.Unmarshal(first.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	run, err := h.server.st.GetRun(context.Background(), view.RunID)
	if err != nil {
		t.Fatal(err)
	}
	job, err := galleryOriginal(run)
	if err != nil || job.Operation != "comment" || job.ProjectID != "" || job.Prompt != "다른 팀에 응원 댓글 달아줘" {
		t.Fatalf("unexpected workflow: %+v %v", job, err)
	}
	parts := strings.Split(job.Ticket, ".")
	if len(parts) != 2 || parts[1] != gallerySign(h.signingKey(), parts[0]) {
		t.Fatal("invalid comment grant")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	var scope map[string]any
	if err != nil || json.Unmarshal(raw, &scope) != nil || len(scope) != 3 || scope["purpose"] != "enode-gallery-comment-v1" || scope["job_id"] != "gallery-post-"+strings.TrimPrefix(view.RunID, "gallery-") {
		t.Fatal("unexpected grant scope")
	}
	if strings.Contains(first.Body.String(), job.Ticket) || strings.Contains(first.Body.String(), galleryProof) {
		t.Fatal("private request information exposed")
	}
	demoTestStatus(t, galleryCall(h.comment, "POST", "/v1/demo/gallery/comments", body, ""), 200)
	demoTestStatus(t, galleryCall(h.result, "GET", "/v1/demo/gallery/runs/"+view.RunID, "", galleryProof), 200)
	demoTestStatus(t, galleryCall(h.publish, "POST", "/v1/demo/gallery/runs/"+view.RunID+"/publish", `{"request_id":"`+galleryProof+`","body":"override"}`, ""), 409)
	for i := range 2 {
		demoTestStatus(t, galleryCall(h.comment, "POST", "/v1/demo/gallery/comments", strings.Replace(body, "응원 댓글", strings.Repeat("응원", i+1), 1), ""), 202)
	}
	demoTestStatus(t, galleryCall(h.comment, "POST", "/v1/demo/gallery/comments", strings.Replace(body, "응원 댓글", "새 댓글", 1), ""), 429)
	demoTestStatus(t, galleryCall(h.comment, "POST", "/v1/demo/gallery/comments", body, ""), 200)
}
