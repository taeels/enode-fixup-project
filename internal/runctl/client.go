// Package runctl 은 사람과 껍데기가 Mediator 에 말하는 표면이다.
//
// runctl 은 무상태다 — 제출하고 잊는다. 죽어도 Run 은 계속 돈다.
// 격자가 확인한 성질이고, ADR-013 이 --interactive 를 거절한 사유이기도 하다.
package runctl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	Base      string
	Token     string
	Principal string
	HTTP      *http.Client
}

// Principal 은 git config 의 이메일이다 (ADR-015 §1).
//
// enode 는 --global 을 쓰지만 runctl 은 해석되는 형태를 쓴다 —
// runctl 은 저장소 안에서 실행되고, 요청자는 *그 저장소에서 커밋할 신원* 과
// 같아야 자연스럽다. Gerrit 이 사람을 그 이메일로 알고 있고 ⑫ 의 코멘트가
// 그 이름으로 달린다.
//
// 없으면 그 자리에서 죽는다 — 조용한 대체는 한 사람에게 두 신원을 만든다.
func Principal() (string, error) {
	out, err := exec.Command("git", "config", "--get", "user.email").Output()
	if err != nil || strings.TrimSpace(string(out)) == "" {
		return "", fmt.Errorf("no git email: set it with git config user.email")
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
	// Requires 는 이 Run 이 무엇을 기다리는가다. Verdict 와 같은 대접이다 —
	// 원문으로 진다. 안 실으면 이 유닛이 연 값이 runctl status 와 mcp 의
	// run.get 에서만 조용히 사라진다. 정본이 run.get 을 Client.Status 로
	// 못 박았으므로 그 둘은 이 구조체로 디코드한다.
	Requires json.RawMessage `json:"requires,omitempty"`
	// Steps 는 실행 중 관측이다 (ADR-025). 폭이 1 을 넘으면 여러 가지가
	// 각각 다른 상태에 있어서 Run 상태 한 줄로는 어디까지 갔는지 안 보인다.
	Steps []Step `json:"steps,omitempty"`
	// Warnings 는 받았지만 뜻대로 안 돌 것이다 (ADR-061 §2).
	Warnings []string `json:"warnings,omitempty"`
}

type Step struct {
	Seq     int      `json:"seq"`
	ID      string   `json:"id"`
	State   string   `json:"state"`
	Uses    string   `json:"uses"`
	Node    string   `json:"node"`
	Needs   []string `json:"needs"`
	Attempt int      `json:"attempt"`
	// Chosen 은 이 단계가 갈림길에서 골라진 적이 있는가다. 언제나 싣는다 —
	// false 를 생략하면 SKIPPED 하나가 두 가지를 다시 뜻하게 된다.
	// 「경로가 갈려 안 갔다」와 「골랐는데 못 닿았다」를 가르려고 만든 열이다.
	Chosen bool `json:"chosen"`
	// StartedAt 은 이 유닛의 필요가 아니다. panel(W3)이 S3 에 단계 이름 ·
	// 회차 · 시작 시각을 그리는데 앞의 둘은 ID 와 Attempt 로 이미 서고
	// 시작 시각만 안 선다.
	//
	// 담당이 갈리는 것이 이 필드를 W0 으로 당긴 이유다 — 파일 행렬이 이
	// 파일을 단독 obs 로 세워 진행자가 직렬 대상으로 보지 않으므로, W3 에
	// 다른 담당이 이 파일을 열면 그 충돌이 늦게 보인다. 한 필드를 지금
	// 세우는 쪽이 싸다.
	StartedAt *time.Time `json:"started_at,omitempty"`
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

// Answer 는 답을 보낸다 — 본문이 곧 답이고 산출물이 된다 (ADR-032).
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
// 이것이 runctl 을 상태 있게 만들지 않는다 — 폴링일 뿐이고 중간에 죽어도
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

// Capabilities 는 계약을 쓰기 전에 어휘를 읽는다 (ADR-014 결정 3).
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

// RunsQuery 는 GET /v1/runs 를 좁히는 인자 넷이다. 넓히지 않는다 —
// 서버가 받는 것과 같은 넷이고, 이름만 갈라 둔다 (store 쪽은 RunFilter).
//
// 빈 문자열과 0 과 IsZero 인 시각은 질의에서 뺀다. 이것은 취향이 아니라
// 계약이다 — 제로값 RunsQuery{} 를 그대로 펴면 ?limit=0 이 되고 서버는
// 그것을 400 으로 거절한다. 인자 없이 부르는 것이 mcp runs.list 의 기본
// 사용법이므로, 안 빼면 그 첫 호출이 실패한다 (CP5).
//
// 클라이언트는 검증하지 않는다. 길이 상한도 limit 천장도 시각 형식도 서버가
// 진다 — 두 곳에서 검사하면 어긋났을 때 화면이 이유 없는 400 을 받는다.
type RunsQuery struct {
	State string
	Since time.Time
	Work  string
	Limit int
}

// query 는 붙일 질의 문자열을 낸다. 실을 것이 없으면 빈 문자열이라
// 물음표도 안 붙는다 — 경로가 인자 없이 부른 것과 글자까지 같아진다.
func (q RunsQuery) query() string {
	v := url.Values{}
	if q.State != "" {
		v.Set("state", q.State)
	}
	if !q.Since.IsZero() {
		v.Set("since", q.Since.Format(time.RFC3339))
	}
	if q.Work != "" {
		v.Set("work", q.Work)
	}
	if q.Limit != 0 {
		v.Set("limit", strconv.Itoa(q.Limit))
	}
	if len(v) == 0 {
		return ""
	}
	return "?" + v.Encode()
}

// raw 는 읽기 경로의 응답 본문을 원문 그대로 읽는다.
//
// 기존 do 를 그대로 탄다 — Authorization 과 X-Enode-Principal 이 붙고
// 2xx 가 아니면 Fail 이 된다. 디코드는 안 한다: io.ReadAll 로 통째로 받는다.
func (c *Client) raw(ctx context.Context, path string) (json.RawMessage, error) {
	resp, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

// Nodes 는 함대를 읽는다. 인자가 없는 것이 ADR-065 의 코드상 표현이다 —
// 여유 질의를 안 만든다.
//
// 원문을 그대로 낸다. mcp 의 fleet.list 가 GET /v1/nodes 와 글자까지 같아야
// 하고(CP5), 파싱해서 다시 마셜하면 키 순서와 생략 규칙이 그 사이에서 갈린다.
//
// 부르는 쪽 셋이 이미 정해져 있고 원하는 모양이 서로 다르다 — mcp(W1)는
// passthrough, panel(W3)은 자기 node_id 행의 lease, transcript(W4)는 assigned
// 로 거른 목록이다. 그래서 파싱 타입을 여기서 미리 내지 않는다.
func (c *Client) Nodes(ctx context.Context) (json.RawMessage, error) {
	return c.raw(ctx, "/v1/nodes")
}

// Runs 는 Run 목록을 읽는다. 반환이 원문인 이유는 Nodes 와 같다.
func (c *Client) Runs(ctx context.Context, q RunsQuery) (json.RawMessage, error) {
	return c.raw(ctx, "/v1/runs"+q.query())
}
