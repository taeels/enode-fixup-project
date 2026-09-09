package api

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/store"
)

const galleryOrigin = "https://main.d3gkmtkue9o7ly.amplifyapp.com"
const galleryWorker = "/opt/enode/gallery/worker.py"
const galleryWork = "gallery-comments-v1"

var galleryProjectID = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
var galleryRunID = regexp.MustCompile(`^gallery-[0-9a-f]{64}$`)

type galleryProject struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	TeamName string `json:"teamName"`
}
type galleryJob struct {
	Operation string `json:"operation"`
	ProjectID string `json:"project_id"`
	Prompt    string `json:"prompt,omitempty"`
	OwnerHash string `json:"owner_hash"`
	Submitter string `json:"submitter"`
	Ticket    string `json:"ticket,omitempty"`
	Body      string `json:"body,omitempty"`
}
type galleryEvent struct {
	Role string `json:"role"`
	Tool string `json:"tool,omitempty"`
	Text string `json:"text"`
}
type galleryResult struct {
	Outcome    string         `json:"outcome"`
	Message    string         `json:"message"`
	ProjectID  string         `json:"project_id,omitempty"`
	Body       string         `json:"body,omitempty"`
	CommentID  string         `json:"comment_id,omitempty"`
	URL        string         `json:"url,omitempty"`
	Transcript []galleryEvent `json:"transcript"`
}
type galleryHandler struct {
	server   *Server
	mu       sync.Mutex // One configured demo Mediator; serial admission and confirmation.
	limit    *bucket
	client   *http.Client
	projects []galleryProject
	cached   time.Time
}

func (s *Server) registerGallery(mux *http.ServeMux) {
	if !s.cfg.Demo || !s.cfg.DemoGallery || s.cfg.Token == "" {
		return
	}
	h := &galleryHandler{server: s, limit: newBucket(2, 4), client: &http.Client{Timeout: 12 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirect blocked") }}}
	mux.HandleFunc("GET /v1/demo/gallery/projects", h.list)
	mux.HandleFunc("POST /v1/demo/gallery/runs", h.draft)
	mux.HandleFunc("GET /v1/demo/gallery/runs/{id}", h.result)
	mux.HandleFunc("POST /v1/demo/gallery/runs/{id}/publish", h.publish)
}

func galleryHash(s string) string { sum := sha256.Sum256([]byte(s)); return hex.EncodeToString(sum[:]) }
func gallerySign(key, value string) string {
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}
func (h *galleryHandler) signingKey() string {
	return gallerySign(h.server.cfg.Token, "enode-gallery-broker-key-v1")
}

// Exact string fields: reject duplicates, nulls, case folding, trailing data and extra commands.
func galleryFields(w http.ResponseWriter, r *http.Request, names ...string) (map[string]string, int) {
	media, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" || (params["charset"] != "" && !strings.EqualFold(params["charset"], "utf-8")) {
		return nil, 415
	}
	body := http.MaxBytesReader(w, r.Body, 8192)
	defer body.Close()
	raw, err := io.ReadAll(body)
	if err != nil {
		var large *http.MaxBytesError
		if errors.As(err, &large) {
			return nil, 413
		}
		return nil, 400
	}
	if !utf8.Valid(raw) {
		return nil, 400
	}
	allowed := map[string]bool{}
	for _, name := range names {
		allowed[name] = true
	}
	fields := map[string]string{}
	dec := json.NewDecoder(bytes.NewReader(raw))
	if token, e := dec.Token(); e != nil || token != json.Delim('{') {
		return nil, 400
	}
	for dec.More() {
		token, e := dec.Token()
		name, ok := token.(string)
		_, seen := fields[name]
		if e != nil || !ok || !allowed[name] || seen {
			return nil, 400
		}
		var value *string
		if dec.Decode(&value) != nil || value == nil {
			return nil, 400
		}
		fields[name] = *value
	}
	if token, e := dec.Token(); e != nil || token != json.Delim('}') || len(fields) != len(names) {
		return nil, 400
	}
	if _, e := dec.Token(); e != io.EOF {
		return nil, 400
	}
	return fields, 0
}
func galleryText(text string, max int) bool {
	if !utf8.ValidString(text) || strings.TrimSpace(text) == "" || utf8.RuneCountInString(text) > max {
		return false
	}
	for _, r := range text {
		if r < 32 && r != '\n' && r != '\t' {
			return false
		}
	}
	return true
}
func (h *galleryHandler) begin(w http.ResponseWriter) bool {
	noStore(w)
	if !h.limit.allow() {
		w.Header().Set("Retry-After", "2")
		fail(w, 429, "rate limited")
		return false
	}
	return true
}
func (h *galleryHandler) catalog(ctx context.Context) ([]galleryProject, error) {
	if time.Since(h.cached) < 5*time.Minute {
		return h.projects, nil
	}
	req, err := http.NewRequestWithContext(ctx, "GET", galleryOrigin+"/api/projects", nil)
	if err != nil {
		return nil, err
	}
	res, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, errors.New("gallery unavailable")
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, 512*1024+1))
	if err != nil || len(raw) > 512*1024 {
		return nil, errors.New("catalog too large")
	}
	var body struct {
		Projects []galleryProject `json:"projects"`
	}
	if err = json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	if len(body.Projects) > 200 {
		return nil, errors.New("catalog too large")
	}
	for _, p := range body.Projects {
		if !galleryProjectID.MatchString(p.ID) || !galleryText(p.Title, 300) || utf8.RuneCountInString(p.TeamName) > 300 {
			return nil, errors.New("invalid catalog")
		}
	}
	h.projects = body.Projects
	h.cached = time.Now()
	return h.projects, nil
}
func (h *galleryHandler) list(w http.ResponseWriter, r *http.Request) {
	if !h.begin(w) {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	projects, err := h.catalog(r.Context())
	if err != nil {
		fail(w, 503, "gallery unavailable")
		return
	}
	write(w, 200, map[string]any{"projects": projects, "account": "Run Away", "origin": galleryOrigin})
}
func galleryContract(id string, job galleryJob) (contract.Contract, error) {
	raw, err := json.Marshal(job)
	if err != nil {
		return contract.Contract{}, err
	}
	zero := 0
	c := contract.Contract{RunID: id, Work: contract.Work{System: "manual", ChangeID: galleryWork, ID: &contract.WorkID{System: "manual", ChangeID: galleryWork}},
		Requires:    []contract.Require{{As: "gallery", Capability: contract.CapabilityAgentReason, Attrs: map[string]string{"gallery": "comments-v1", "sandbox": "lima-vm", "provider": "bedrock", "instance": "macmini-bedrock-vm"}}},
		Steps:       []contract.Step{{ID: "gallery", Uses: "gallery", Run: []string{"/usr/bin/python3", galleryWorker, base64.RawURLEncoding.EncodeToString(raw)}, Out: []string{"gallery-result.json"}}},
		SuccessWhen: []contract.Condition{{Step: "gallery", ExitCode: &zero, Produced: []string{"gallery-result.json"}}}}
	return c, c.Validate()
}
func galleryOriginal(run *store.Run) (galleryJob, error) {
	var job galleryJob
	c := run.Contract
	if len(c.Steps) != 1 || len(c.Steps[0].Run) != 3 || c.Steps[0].Run[1] != galleryWorker {
		return job, errors.New("not gallery")
	}
	raw, err := base64.RawURLEncoding.DecodeString(c.Steps[0].Run[2])
	if err != nil || json.Unmarshal(raw, &job) != nil {
		return job, errors.New("invalid gallery job")
	}
	expected, err := galleryContract(run.RunID, job)
	if err != nil {
		return job, err
	}
	a, _ := json.Marshal(c)
	b, _ := json.Marshal(expected)
	if !bytes.Equal(a, b) || job.Operation != "draft" || !galleryRunID.MatchString(run.RunID) {
		return job, errors.New("invalid gallery contract")
	}
	return job, nil
}
func (h *galleryHandler) owned(r *http.Request, id, proof string) (*store.Run, galleryJob, error) {
	var job galleryJob
	if !galleryRunID.MatchString(id) || !demoIDPattern.MatchString(proof) {
		return nil, job, errors.New("invalid owner")
	}
	run, err := h.server.st.GetRun(r.Context(), id)
	if err != nil {
		return nil, job, err
	}
	job, err = galleryOriginal(run)
	if err != nil || !hmac.Equal([]byte(job.OwnerHash), []byte(galleryHash(proof))) {
		return nil, job, errors.New("invalid owner")
	}
	return run, job, nil
}
func (h *galleryHandler) admit(w http.ResponseWriter, r *http.Request) bool {
	rows, _, err := h.server.st.Runs(r.Context(), store.RunFilter{Work: "manual:" + galleryWork, Since: time.Now().Add(-24 * time.Hour), Limit: 201})
	if err != nil {
		fail(w, 503, "cannot check demo capacity")
		return false
	}
	pending := 0
	for _, row := range rows {
		if row.State == store.StateQueued || row.State == store.StateRunning || row.State == store.StateResolving || row.State == store.StateAllocating || row.State == store.StateVerifying {
			pending++
		}
	}
	if len(rows) >= 200 || pending >= 3 {
		w.Header().Set("Retry-After", "30")
		fail(w, 429, "gallery demo capacity reached")
		return false
	}
	return true
}
func (h *galleryHandler) submit(w http.ResponseWriter, r *http.Request, id string, job galleryJob) {
	c, err := galleryContract(id, job)
	if err != nil {
		fail(w, 503, "cannot prepare demo")
		return
	}
	raw, _ := json.Marshal(c)
	internal := r.Clone(context.WithValue(r.Context(), submitterKey, job.Submitter))
	internal.Method = "POST"
	internal.URL.Path = "/v1/runs"
	internal.URL.RawPath = ""
	internal.URL.RawQuery = ""
	internal.RequestURI = "/v1/runs"
	internal.Header = http.Header{"Content-Type": {"application/json"}, "Authorization": {"Bearer " + h.server.cfg.Token}}
	internal.Body = io.NopCloser(bytes.NewReader(raw))
	defer internal.Body.Close()
	internal.ContentLength = int64(len(raw))
	internal.GetBody = nil
	internal.TransferEncoding = nil
	internal.Trailer = nil
	h.server.auth(h.server.postRuns)(w, internal)
}
func (h *galleryHandler) draft(w http.ResponseWriter, r *http.Request) {
	if !h.begin(w) {
		return
	}
	fields, status := galleryFields(w, r, "project_id", "prompt", "request_id", "submitter")
	if status != 0 {
		fail(w, status, "invalid gallery request")
		return
	}
	if !galleryProjectID.MatchString(fields["project_id"]) || !galleryText(fields["prompt"], 1000) || !demoIDPattern.MatchString(fields["request_id"]) || !demoNamePattern.MatchString(fields["submitter"]) {
		fail(w, 400, "invalid gallery request")
		return
	}
	identity, _ := json.Marshal([]string{fields["project_id"], fields["prompt"], fields["request_id"], fields["submitter"]})
	id := "gallery-" + galleryHash(string(identity))
	h.mu.Lock()
	defer h.mu.Unlock()
	if existing, err := h.server.st.GetRun(r.Context(), id); err == nil {
		write(w, 200, h.server.view(r.Context(), existing))
		return
	} else if !errors.Is(err, store.ErrNotFound) {
		fail(w, 503, "cannot check request")
		return
	}
	projects, err := h.catalog(r.Context())
	if err != nil {
		fail(w, 503, "gallery unavailable")
		return
	}
	found := false
	for _, p := range projects {
		if p.ID == fields["project_id"] {
			found = true
		}
	}
	if !found {
		fail(w, 400, "unknown project")
		return
	}
	if !h.admit(w, r) {
		return
	}
	h.submit(w, r, id, galleryJob{Operation: "draft", ProjectID: fields["project_id"], Prompt: fields["prompt"], OwnerHash: galleryHash(fields["request_id"]), Submitter: fields["submitter"]})
}
func (h *galleryHandler) artifact(run *store.Run) (*galleryResult, error) {
	if run.State != store.StateSucceeded {
		return nil, nil
	}
	if !h.server.records.Sealed(run.RunID) {
		return nil, nil
	}
	file, size, err := h.server.records.OpenBlob(run.RunID, "gallery-result.json")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	if size > 32768 {
		return nil, errors.New("result too large")
	}
	raw, err := io.ReadAll(io.LimitReader(file, 32769))
	if err != nil || len(raw) > 32768 {
		return nil, errors.New("invalid result")
	}
	var result galleryResult
	if json.Unmarshal(raw, &result) != nil {
		return nil, errors.New("invalid result")
	}
	switch result.Outcome {
	case "refused", "draft_ready", "posted", "error":
	default:
		return nil, errors.New("invalid outcome")
	}
	if len(result.Transcript) > 16 || !galleryText(result.Message, 2000) {
		return nil, errors.New("invalid transcript")
	}
	for _, event := range result.Transcript {
		if (event.Role != "user" && event.Role != "assistant" && event.Role != "system" && event.Role != "tool") || !galleryText(event.Text, 3000) || (event.Tool != "" && event.Tool != "get_project" && event.Tool != "post_confirmed_comment") {
			return nil, errors.New("invalid event")
		}
	}
	if result.Outcome == "draft_ready" && !galleryText(result.Body, 500) {
		return nil, errors.New("invalid draft")
	}
	// Do not trust model-provided navigation targets.
	result.URL = ""
	if result.Outcome == "posted" && galleryProjectID.MatchString(result.ProjectID) {
		result.URL = galleryOrigin + "/projects/" + result.ProjectID
	}
	return &result, nil
}
func (h *galleryHandler) result(w http.ResponseWriter, r *http.Request) {
	if !h.begin(w) {
		return
	}
	run, _, err := h.owned(r, r.PathValue("id"), r.Header.Get("X-Gallery-Request-ID"))
	if err != nil {
		fail(w, 404, "gallery request not found")
		return
	}
	result, err := h.artifact(run)
	if err != nil {
		fail(w, 503, "gallery result unavailable")
		return
	}
	response := map[string]any{"run": h.server.view(r.Context(), run), "result": result}
	if posted, e := h.server.st.GetRun(r.Context(), "gallery-post-"+strings.TrimPrefix(run.RunID, "gallery-")); e == nil {
		publication, e := h.artifact(posted)
		if e != nil {
			fail(w, 503, "publication result unavailable")
			return
		}
		response["publication"] = map[string]any{"run": h.server.view(r.Context(), posted), "result": publication}
	} else if !errors.Is(e, store.ErrNotFound) {
		fail(w, 503, "publication unavailable")
		return
	}
	write(w, 200, response)
}
func (h *galleryHandler) publish(w http.ResponseWriter, r *http.Request) {
	if !h.begin(w) {
		return
	}
	fields, status := galleryFields(w, r, "request_id", "body")
	if status != 0 {
		fail(w, status, "invalid confirmation")
		return
	}
	if !galleryText(fields["body"], 500) {
		fail(w, 400, "invalid comment")
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	run, job, err := h.owned(r, r.PathValue("id"), fields["request_id"])
	if err != nil {
		fail(w, 404, "gallery request not found")
		return
	}
	id := "gallery-post-" + strings.TrimPrefix(run.RunID, "gallery-")
	if existing, e := h.server.st.GetRun(r.Context(), id); e == nil {
		if len(existing.Contract.Steps) != 1 || len(existing.Contract.Steps[0].Run) != 3 {
			fail(w, 409, "confirmation conflict")
			return
		}
		raw, e := base64.RawURLEncoding.DecodeString(existing.Contract.Steps[0].Run[2])
		var previous galleryJob
		if e != nil || json.Unmarshal(raw, &previous) != nil || previous.Body != fields["body"] {
			fail(w, 409, "confirmation already fixed")
			return
		}
		write(w, 200, h.server.view(r.Context(), existing))
		return
	} else if !errors.Is(e, store.ErrNotFound) {
		fail(w, 503, "cannot check confirmation")
		return
	}
	result, err := h.artifact(run)
	if err != nil || result == nil || result.Outcome != "draft_ready" || result.ProjectID != job.ProjectID {
		fail(w, 409, "draft is not ready")
		return
	}
	if !h.admit(w, r) {
		return
	}
	ticket, _ := json.Marshal(map[string]any{"purpose": "enode-gallery-publish-v1", "job_id": id, "project_id": job.ProjectID, "body": fields["body"], "expires": time.Now().Add(15 * time.Minute).Unix()})
	payload := base64.RawURLEncoding.EncodeToString(ticket)
	job.Operation = "publish"
	job.Prompt = ""
	job.Body = fields["body"]
	job.Ticket = payload + "." + gallerySign(h.signingKey(), payload)
	h.submit(w, r, id, job)
}
