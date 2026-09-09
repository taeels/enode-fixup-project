package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/taeels/enode/internal/contract"
)

const demoBodyLimit = 4 * 1024

var (
	demoNamePattern   = regexp.MustCompile(`^[가-힣]{1,12} [가-힣]{1,12}$`)
	demoLegacyPattern = regexp.MustCompile(`^guest-[a-z]{1,24}-[a-z]{1,24}$`)
	demoIDPattern     = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
)

type demoSubmission struct {
	ScenarioID string `json:"scenario_id"`
	Submitter  string `json:"submitter"`
	RequestID  string `json:"request_id"`
}

func (d demoSubmission) runID() string {
	// 검증한 문자열 셋의 JSON 배열은 실패 없이 직렬화된다. 원문을 정규화하지 않는다.
	b, _ := json.Marshal([3]string{d.ScenarioID, d.RequestID, d.Submitter})
	hash := sha256.Sum256(b)
	return "demo-" + hex.EncodeToString(hash[:])
}

// demoScenario는 서버 코드만 지정한다. 요청 값을 example 이름이나 경로로 쓰지 않는다.
// inject는 검토한 데이터 위치만 바꾼다. LED처럼 이름을 쓰지 않아도 명시적 함수를 둔다.
type demoScenario struct {
	example string
	inject  func(*contract.Contract, string) error
}

func demoScenarios() map[string]demoScenario {
	// 진행자 c7a237d의 두 계약만 공개한다. LED는 이름을 실행 명령에 넣지 않는다.
	return map[string]demoScenario{
		"led-toggle":    {example: "demo-led-toggle", inject: func(*contract.Contract, string) error { return nil }},
		"welcome-audio": {example: "demo-welcome-audio", inject: injectDemoWelcome},
	}
}

func injectDemoWelcome(c *contract.Contract, name string) error {
	for i := range c.Steps {
		step := &c.Steps[i]
		if step.ID != "synthesize" {
			continue
		}
		prompt, ok := step.In["prompt"].(string)
		if !ok || step.Agent == nil || strings.Count(prompt, "{{submitter}}") != 1 {
			return errors.New("invalid welcome name injection point")
		}
		// 검토한 prompt 한 필드만 바꾼다. argv·경로·다른 텍스트는 치환하지 않는다.
		step.In["prompt"] = strings.Replace(prompt, "{{submitter}}", name, 1)
		return nil
	}
	return errors.New("missing welcome synthesis step")
}

type demoHandler struct {
	server    *Server
	scenarios map[string]demoScenario
	limit     *bucket
}

func (s *Server) newDemoHandler(scenarios map[string]demoScenario) *demoHandler {
	// 등록 시 사본을 고정한다. 읽기 버킷이나 Server 필드를 늘리지 않는다.
	fixed := make(map[string]demoScenario, len(scenarios))
	for name, scenario := range scenarios {
		fixed[name] = scenario
	}
	return &demoHandler{server: s, scenarios: fixed, limit: newBucket(2, 4)}
}

func (h *demoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	if !h.limit.allow() {
		w.Header().Set("Retry-After", "1")
		fail(w, http.StatusTooManyRequests, "rate limited")
		return
	}
	d, status := readDemoSubmission(w, r)
	if status != 0 {
		fail(w, status, "invalid demo request")
		return
	}
	if h.server.cfg.Token == "" {
		fail(w, http.StatusServiceUnavailable, "demo submission is not configured")
		return
	}
	c, err := h.scenarios[d.ScenarioID].prepare(d)
	if err != nil {
		fail(w, http.StatusServiceUnavailable, "demo scenario is not configured")
		return
	}
	body, err := json.Marshal(c)
	if err != nil {
		fail(w, http.StatusServiceUnavailable, "cannot prepare demo request")
		return
	}
	// 브라우저의 헤더·principal을 내부 요청에 넘기지 않는다. 취소 수명만 이어받는다.
	internal := r.Clone(context.WithValue(r.Context(), submitterKey, d.Submitter))
	internal.Method = http.MethodPost
	internal.URL.Path, internal.URL.RawPath, internal.URL.RawQuery = "/v1/runs", "", ""
	internal.RequestURI = "/v1/runs"
	internal.Header = http.Header{
		"Authorization": {"Bearer " + h.server.cfg.Token},
		"Content-Type":  {"application/json"},
	}
	internal.Body = io.NopCloser(bytes.NewReader(body))
	defer internal.Body.Close()
	internal.ContentLength = int64(len(body))
	internal.GetBody, internal.TransferEncoding, internal.Trailer = nil, nil, nil
	h.server.auth(h.server.postRuns)(w, internal)
}

func readDemoSubmission(w http.ResponseWriter, r *http.Request) (demoSubmission, int) {
	var d demoSubmission
	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" || (params["charset"] != "" && !strings.EqualFold(params["charset"], "utf-8")) {
		return d, http.StatusUnsupportedMediaType
	}
	body := http.MaxBytesReader(w, r.Body, demoBodyLimit)
	defer body.Close()
	raw, err := io.ReadAll(body)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return d, http.StatusRequestEntityTooLarge
		}
		return d, http.StatusBadRequest
	}
	if !utf8.Valid(raw) {
		return d, http.StatusBadRequest
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		return d, http.StatusBadRequest
	}
	fields := map[string]*string{"scenario_id": &d.ScenarioID, "submitter": &d.Submitter, "request_id": &d.RequestID}
	seen := make(map[string]bool, len(fields))
	for decoder.More() {
		key, err := decoder.Token()
		name, ok := key.(string)
		if err != nil || !ok || fields[name] == nil || seen[name] {
			return d, http.StatusBadRequest
		}
		seen[name] = true
		if err := decoder.Decode(fields[name]); err != nil {
			return d, http.StatusBadRequest
		}
	}
	closing, err := decoder.Token()
	if err != nil || closing != json.Delim('}') || len(seen) != len(fields) {
		return d, http.StatusBadRequest
	}
	if _, err := decoder.Token(); err != io.EOF {
		return d, http.StatusBadRequest
	}
	if d.ScenarioID != "led-toggle" && d.ScenarioID != "welcome-audio" {
		return d, http.StatusBadRequest
	}
	if !demoIDPattern.MatchString(d.RequestID) || len(d.Submitter) > 73 || (!demoNamePattern.MatchString(d.Submitter) && !demoLegacyPattern.MatchString(d.Submitter)) {
		return d, http.StatusBadRequest
	}
	return d, 0
}

func (s demoScenario) prepare(d demoSubmission) (contract.Contract, error) {
	var c contract.Contract
	if s.example == "" || s.inject == nil {
		return c, errors.New("demo scenario is not configured")
	}
	raw, err := contract.Example(s.example)
	if err != nil {
		return c, err
	}
	// 매번 새로 해석하므로 requires·steps·중첩 입력도 다른 요청과 공유하지 않는다.
	if err := json.Unmarshal(raw, &c); err != nil {
		return c, err
	}
	c.RunID = d.runID()
	c.Work = contract.Work{System: "manual", ChangeID: c.RunID, ID: &contract.WorkID{System: "manual", ChangeID: c.RunID}}
	if err := s.inject(&c, d.Submitter); err != nil {
		return c, err
	}
	return c, c.Validate()
}
