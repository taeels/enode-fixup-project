package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrNoRun 은 그 run_id 가 아직 없다는 뜻이다 (404). 세대를 탐침할 때 쓴다.
var ErrNoRun = errors.New("no such run")

// HTTPError 는 Mediator 가 준 4xx·5xx 다.
//
// 상태 코드를 살려 두는 이유가 하나다 — 409 와 422 를 호출자가 갈라야 한다.
// 매처가 「후보는 있는데 전부 점유됨」(409, 일시) 과 「함대에 없다」(422, 영구)
// 를 가르는 목적이 재시도해도 되는지를 알려주는 것인데 (ADR-014 결정 3),
// error 하나로 뭉개면 그 정보가 여기서 사라진다. 실제로 사라져 있었다 —
// 실행 노드가 잠시 바쁜 것뿐인데 이슈가 실패 칸으로 갔다.
type HTTPError struct {
	Status int
	Method string
	Path   string
	Body   string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("%s %s: %d %s", e.Method, e.Path, e.Status, e.Body)
}

// IsBusy 는 「지금은 전부 점유됨」인가다 (409).
//
// 일시적이므로 다시 내면 된다. 422 는 여기 안 걸린다 — 함대에 그런 노드가
// 아예 없다는 뜻이라 다시 내도 같다.
func IsBusy(err error) bool {
	var he *HTTPError
	return errors.As(err, &he) && he.Status == http.StatusConflict
}

// Mediator 는 우리 쪽 클라이언트다.
//
// Mediator 는 It's a Plan 을 모른다 (ADR-002 의 범위를 안 넓힌다).
// 그래서 이 파일에는 이슈도 코멘트도 없고, 오직 계약·Run·되묻기만 있다.
type Mediator struct {
	base      string
	token     string
	principal string
	client    *http.Client
}

func NewMediator(base, token, principal string) *Mediator {
	return &Mediator{
		base:      strings.TrimRight(base, "/"),
		token:     token,
		principal: principal,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (m *Mediator) do(ctx context.Context, method, path string, body []byte, out any) error {
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, m.base+path, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+m.token)
	if m.principal != "" {
		req.Header.Set("X-Enode-Principal", m.principal)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode == 404 {
		return ErrNoRun
	}
	if resp.StatusCode >= 400 {
		return &HTTPError{Status: resp.StatusCode, Method: method, Path: path,
			Body: strings.TrimSpace(string(raw))}
	}
	if out != nil && len(bytes.TrimSpace(raw)) > 0 {
		return json.Unmarshal(raw, out)
	}
	return nil
}

// Check 는 계약 조건 하나의 대조 결과다 (store.Check 와 같은 모양).
//
// Want 와 Got 이 다형이다 — produced 검사는 문자열 배열이고 exit_code
// 검사는 숫자다. []string 으로 받으면 실물에서 조용히 안 읽힌다:
// 어댑터가 Run 조회에 계속 실패해 결과 코멘트를 영영 못 쓴다.
// 첫 실측이 이것을 잡았다 (2026-08-24, EP-2).
type Check struct {
	Step string `json:"step"`
	What string `json:"what"` // exit_code | produced
	Want any    `json:"want"`
	Got  any    `json:"got"`
	OK   bool   `json:"ok"`
	Note string `json:"note,omitempty"`
}

// RunView 는 GET /v1/runs/{id} 가 주는 것이다.
type RunView struct {
	RunID    string `json:"run_id"`
	State    string `json:"state"`
	Assigned []struct {
		As    string `json:"as"`
		Nodes []struct {
			Node  string `json:"node"`
			Label string `json:"label"`
		} `json:"nodes"`
	} `json:"assigned"`
	Verdict struct {
		State  string  `json:"state"`
		Checks []Check `json:"checks"`
	} `json:"verdict"`
	Steps []struct {
		Seq       int      `json:"seq"`
		ID        string   `json:"id"`
		State     string   `json:"state"`
		Uses      string   `json:"uses"`
		Node      string   `json:"node"`
		Needs     []string `json:"needs"`
		Attempt   int      `json:"attempt"`
		StartedAt string   `json:"started_at"`
		EndedAt   string   `json:"ended_at"`
		ExitCode  *int     `json:"exit_code"`
		Error     string   `json:"error"`
	} `json:"steps"`
}

// Terminal 은 Run 이 끝났는가다. 끝난 Run 만 봉인되고 Record 가 열린다 (I4).
func (r *RunView) Terminal() bool {
	switch r.State {
	case "SUCCEEDED", "FAILED", "CANCELLED", "EXPIRED":
		return true
	}
	return false
}

func (m *Mediator) GetRun(ctx context.Context, runID string) (*RunView, error) {
	var out RunView
	if err := m.do(ctx, "GET", "/v1/runs/"+runID, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SubmitRun 은 계약을 낸다. 점유는 전부 아니면 전무다 (I5).
//
// 거절이 둘로 갈린다 — 함대에 그런 노드가 없으면 422(영구) 이고, 있는데
// 지금 전부 잡혀 있으면 409(일시) 다. 호출자가 IsBusy 로 가른다.
func (m *Mediator) SubmitRun(ctx context.Context, contract []byte) error {
	return m.do(ctx, "POST", "/v1/runs", contract, nil)
}

// Capability 는 GET /v1/capabilities 의 항목 하나다.
type Capability struct {
	Capability string              `json:"capability"`
	Nodes      int                 `json:"nodes"`
	Attrs      map[string][]string `json:"attrs"`
}

func (m *Mediator) Capabilities(ctx context.Context) ([]Capability, error) {
	var out struct {
		Capabilities []Capability `json:"capabilities"`
	}
	if err := m.do(ctx, "GET", "/v1/capabilities", nil, &out); err != nil {
		return nil, err
	}
	return out.Capabilities, nil
}

// HasLabel 은 그 능력에 그 이름표가 광고돼 있는가다.
//
// 이것이 ⑤ 의 기다림이다 (adapter-example §1.2) — 오케스트레이터가 아직
// 안 떴는데 Run 을 내면 422 다. 그래서 광고에 자기 이름표가 보일 때까지 기다린다.
func (m *Mediator) HasLabel(ctx context.Context, capability, key, val string) (bool, error) {
	caps, err := m.Capabilities(ctx)
	if err != nil {
		return false, err
	}
	for _, c := range caps {
		if c.Capability != capability {
			continue
		}
		for _, v := range c.Attrs[key] {
			if v == val {
				return true, nil
			}
		}
	}
	return false, nil
}

// AskView 는 인박스의 항목 하나다 (ADR-032 §4).
type AskView struct {
	RunID    string          `json:"run_id"`
	Seq      int             `json:"seq"`
	Step     string          `json:"step"`
	Prompt   string          `json:"prompt"`
	Schema   json.RawMessage `json:"schema"`
	AskedAt  *time.Time      `json:"asked_at"`
	Deadline *time.Time      `json:"deadline"`
	Shown    []struct {
		Name    string          `json:"name"`
		Content json.RawMessage `json:"content"`
	} `json:"shown"`
	Proposes json.RawMessage `json:"proposes"`
}

// Asks 는 답을 기다리는 되묻기 전부다. 인박스가 정본이다 — 알림은 보조다.
func (m *Mediator) Asks(ctx context.Context) ([]AskView, error) {
	var out struct {
		Asks []AskView `json:"asks"`
	}
	if err := m.do(ctx, "GET", "/v1/asks", nil, &out); err != nil {
		return nil, err
	}
	return out.Asks, nil
}

// AskFor 는 그 Run 이 지금 기다리는 되묻기를 고른다.
func (m *Mediator) AskFor(ctx context.Context, runID string) (*AskView, error) {
	asks, err := m.Asks(ctx)
	if err != nil {
		return nil, err
	}
	for i := range asks {
		if asks[i].RunID == runID {
			return &asks[i], nil
		}
	}
	return nil, nil
}

// Answer 는 되묻기에 답한다. 본문이 곧 답이고 그대로 산출물이 된다.
// 스키마 위반이면 422 이고 질문은 열린 채 남는다 — 다시 답하면 된다.
func (m *Mediator) Answer(ctx context.Context, runID string, seq int, body []byte) error {
	return m.do(ctx, "POST", fmt.Sprintf("/v1/runs/%s/steps/%d/answer", runID, seq), body, nil)
}

// LedgerEntry 는 원장의 한 줄이다 — blobs/ 에서 유도된다 (ADR-040 §3.3 정정).
type LedgerEntry struct {
	Seq      int    `json:"seq"`
	Attempt  int    `json:"attempt"`
	Name     string `json:"name"`
	By       string `json:"by"`
	At       string `json:"at"`
	Bytes    int64  `json:"bytes"`
	SchemaOK *bool  `json:"schema_ok"`
}

func (m *Mediator) Ledger(ctx context.Context, runID string) ([]LedgerEntry, error) {
	var out struct {
		Entries []LedgerEntry `json:"entries"`
	}
	if err := m.do(ctx, "GET", "/v1/runs/"+runID+"/ledger", nil, &out); err != nil {
		return nil, err
	}
	return out.Entries, nil
}

// Blob 은 산출물 하나를 읽는다.
func (m *Mediator) Blob(ctx context.Context, runID, name string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		m.base+"/v1/runs/"+runID+"/blob/"+name, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+m.token)
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return nil, ErrNoRun
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("blob %s: %d", name, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 8<<20))
}

// PutBlob 은 산출물을 하나 쓴다.
//
// ADR-040 §6 이 남긴 물음의 답이다 — "_outbound 를 누가 PUT 하나".
// 노드가 쓰는 것과 같은 HTTP 경로를 어댑터도 쓴다. 내부 함수를 새로
// 뚫지 않으므로 enode 코어가 한 줄도 안 바뀐다 (ADR-040 §5).
func (m *Mediator) PutBlob(ctx context.Context, runID string, seq int, name string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, "PUT",
		fmt.Sprintf("%s/v1/runs/%s/steps/%d/blob/%s", m.base, runID, seq, name),
		bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+m.token)
	req.Header.Set("Content-Type", "application/json")
	if m.principal != "" {
		req.Header.Set("X-Enode-Principal", m.principal)
	}
	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 400 {
		return fmt.Errorf("PUT blob %s: %d %s", name, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return nil
}
