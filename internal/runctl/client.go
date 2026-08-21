// Package runctl 은 사람과 껍데기가 Mediator 에 말하는 표면이다.
//
// ★ runctl 은 무상태다 ★ — 제출하고 잊는다. 죽어도 Run 은 계속 돈다.
// 격자가 확인한 성질이고, ADR-013 이 --interactive 를 거절한 사유이기도 하다.
package runctl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

type Client struct {
	Base      string
	Token     string
	Principal string
	HTTP      *http.Client
}

// Principal 은 ★ git config 의 이메일 ★ 이다 (ADR-015 §1).
//
// enode 는 --global 을 쓰지만 runctl 은 ★ 해석되는 형태 ★ 를 쓴다 —
// runctl 은 저장소 안에서 실행되고, 요청자는 *그 저장소에서 커밋할 신원* 과
// 같아야 자연스럽다. Gerrit 이 사람을 그 이메일로 알고 있고 ⑫ 의 코멘트가
// 그 이름으로 달린다.
//
// 없으면 ★ 그 자리에서 죽는다 ★ — 조용한 대체는 한 사람에게 두 신원을 만든다.
func Principal() (string, error) {
	out, err := exec.Command("git", "config", "--get", "user.email").Output()
	if err != nil || strings.TrimSpace(string(out)) == "" {
		return "", fmt.Errorf("git 이메일이 없다: git config user.email 을 설정하라")
	}
	return strings.TrimSpace(string(out)), nil
}

// Fail 은 와이어의 HTTP 상태를 그대로 들고 온다.
// CLI 종료코드로의 번역은 cmd/runctl 이 한다 — 경계가 둘이라 하나로 통일하지 않는다
// (INVARIANTS §4 「에러 코드 체계」).
type Fail struct {
	Code   int
	Reason string
}

func (f *Fail) Error() string { return fmt.Sprintf("%d %s", f.Code, f.Reason) }

func (c *Client) do(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.Base+path, r)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-Enode-Principal", c.Principal)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 == 2 {
		return resp, nil
	}
	defer resp.Body.Close()
	var e struct {
		Error struct {
			Code   int    `json:"code"`
			Reason string `json:"reason"`
		} `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&e)
	reason := e.Error.Reason
	if reason == "" {
		reason = resp.Status
	}
	return nil, &Fail{Code: resp.StatusCode, Reason: reason}
}

type Run struct {
	RunID    string          `json:"run_id"`
	State    string          `json:"state"`
	Assigned []Assigned      `json:"assigned,omitempty"`
	Verdict  json.RawMessage `json:"verdict,omitempty"`
	// Steps 는 ★ 실행 중 관측 ★ 이다 (ADR-025). 폭이 1 을 넘으면 여러 가지가
	// 각각 다른 상태에 있어서 Run 상태 한 줄로는 어디까지 갔는지 안 보인다.
	Steps []Step `json:"steps,omitempty"`
}

type Step struct {
	Seq     int      `json:"seq"`
	ID      string   `json:"id"`
	State   string   `json:"state"`
	Uses    string   `json:"uses"`
	Node    string   `json:"node"`
	Needs   []string `json:"needs"`
	Attempt int      `json:"attempt"`
}

type Assigned struct {
	As    string `json:"as"`
	Nodes []struct {
		Node  string `json:"node"`
		Label string `json:"label"`
	} `json:"nodes"`
}

func (c *Client) Submit(ctx context.Context, contract []byte, dry bool) (*Run, error) {
	path := "/v1/runs"
	if dry {
		path += "/dry-run"
	}
	resp, err := c.do(ctx, "POST", path, contract)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var r Run
	return &r, json.NewDecoder(resp.Body).Decode(&r)
}

// AskItem 은 인박스의 항목 하나다 (ADR-032 §4).
type AskItem struct {
	RunID     string          `json:"run_id"`
	Seq       int             `json:"seq"`
	Step      string          `json:"step"`
	Prompt    string          `json:"prompt"`
	Schema    json.RawMessage `json:"schema"`
	Answerers []string        `json:"answerers"`
	Deadline  *time.Time      `json:"deadline"`
	CanAnswer bool            `json:"can_answer"`
	Shown     []ShownArtifact `json:"shown"`
	Proposes  json.RawMessage `json:"proposes"`
}

// ShownArtifact 는 질문과 함께 실려 온 산출물이다.
type ShownArtifact struct {
	Name      string          `json:"name"`
	Content   json.RawMessage `json:"content"`
	Truncated bool            `json:"truncated"`
}

// Asks 는 답을 기다리는 되묻기 전부다 — 답한 것은 봉인에 있다.
func (c *Client) Asks(ctx context.Context) ([]AskItem, error) {
	resp, err := c.do(ctx, "GET", "/v1/asks", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out struct {
		Asks []AskItem `json:"asks"`
	}
	return out.Asks, json.NewDecoder(resp.Body).Decode(&out)
}

// Answer 는 답을 보낸다 — ★ 본문이 곧 답이고 산출물이 된다 ★ (ADR-032).
func (c *Client) Answer(ctx context.Context, runID string, seq int, body []byte) (*Run, error) {
	resp, err := c.do(ctx, "POST", fmt.Sprintf("/v1/runs/%s/steps/%d/answer", runID, seq), body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var r Run
	return &r, json.NewDecoder(resp.Body).Decode(&r)
}

func (c *Client) Status(ctx context.Context, runID string) (*Run, error) {
	resp, err := c.do(ctx, "GET", "/v1/runs/"+runID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var r Run
	return &r, json.NewDecoder(resp.Body).Decode(&r)
}

func (c *Client) Cancel(ctx context.Context, runID string) (*Run, error) {
	resp, err := c.do(ctx, "POST", "/v1/runs/"+runID+"/cancel", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var r Run
	return &r, json.NewDecoder(resp.Body).Decode(&r)
}

func (c *Client) Record(ctx context.Context, runID string, w io.Writer) error {
	resp, err := c.do(ctx, "GET", "/v1/runs/"+runID+"/record", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(w, resp.Body)
	return err
}

// Terminal 은 종료 상태인지다 (INVARIANTS §1.1).
func Terminal(state string) bool { return state == "SUCCEEDED" || state == "FAILED" }

// Wait 는 종료 상태가 될 때까지 기다린다.
//
// ★ 이것이 runctl 을 상태 있게 만들지 않는다 ★ — 폴링일 뿐이고 중간에 죽어도
// Run 은 계속 돈다. 되묻기를 올리는 것(ADR-013 --interactive)과는 다르다.
func (c *Client) Wait(ctx context.Context, runID string, every time.Duration) (*Run, error) {
	for {
		r, err := c.Status(ctx, runID)
		if err != nil {
			return nil, err
		}
		if Terminal(r.State) {
			return r, nil
		}
		select {
		case <-ctx.Done():
			return r, ctx.Err()
		case <-time.After(every):
		}
	}
}

// Capability 는 함대의 속성 어휘 한 줄이다.
type Capability struct {
	Capability string              `json:"capability"`
	Nodes      int                 `json:"nodes"`
	Attrs      map[string][]string `json:"attrs"`
}

// Capabilities 는 ★ 계약을 쓰기 전에 어휘를 읽는다 ★ (ADR-014 결정 3).
// 어휘가 enode 광고에서 창발하므로 어디에도 선언되어 있지 않다.
func (c *Client) Capabilities(ctx context.Context) ([]Capability, error) {
	resp, err := c.do(ctx, "GET", "/v1/capabilities", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out struct {
		Capabilities []Capability `json:"capabilities"`
	}
	return out.Capabilities, json.NewDecoder(resp.Body).Decode(&out)
}
