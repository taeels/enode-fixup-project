package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// IAP 는 It's a Plan 클라이언트다.
//
// ★ 러너 프로토콜은 세 엔드포인트뿐이다 ★ (apps/api/src/modules/agents/runner/index.ts):
//
//	POST /agent-runs/claim              큐에서 하나 당긴다
//	POST /agent-runs/{id}/heartbeat     리스를 늘린다
//	POST /agent-runs/{id}/result        끝났다고 보고한다
//
// ★ 그리고 그 셋으로는 이슈에 아무것도 안 써진다 ★ (integration §2.1 실측 ⑥) —
// 활동 두 줄만 남는다. 사람이 읽을 것은 코멘트로, 보드가 읽을 것은 칸 이동으로
// ★ 어댑터가 직접 쓴다 ★. 같은 에이전트 키 하나로 전부 된다.
type IAP struct {
	base   string
	key    string
	client *http.Client
}

func NewIAP(base, key string) *IAP {
	return &IAP{
		base:   strings.TrimRight(base, "/"),
		key:    key,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// RunnerRun 은 claim 이 주는 것 전부다.
//
// ★ 일곱 개이고 전부 자연어다 ★ (integration §5 · 실측 ③) — 구조화 필드가
// 하나도 없다. 그래서 계약은 ★ 고정 템플릿 ★ 이고 추론이 0 이다.
type RunnerRun struct {
	ID              int    `json:"id"`
	Trigger         string `json:"trigger"`
	Prompt          string `json:"prompt"`
	SystemPrompt    string `json:"systemPrompt"`
	Attempts        int    `json:"attempts"`
	IssueID         *int   `json:"issueId"`
	IssueIdentifier string `json:"issueIdentifier"`
}

func (c *IAP) do(ctx context.Context, method, path string, body any, out any) (int, error) {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, rd)
	if err != nil {
		return 0, err
	}
	req.Header.Set("x-api-key", c.key)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode >= 400 {
		return resp.StatusCode, fmt.Errorf("%s %s: %d %s", method, path, resp.StatusCode,
			strings.TrimSpace(string(raw)))
	}
	if out != nil && len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return resp.StatusCode, fmt.Errorf("%s %s: 응답을 못 읽는다: %w", method, path, err)
		}
	}
	return resp.StatusCode, nil
}

// Claim 은 큐에서 하나 당긴다. 큐가 비면 (nil, nil) 이다.
//
// ★ 클레임이 status 를 안 바꾼다 ★ (실측 ⑤) — 리스만 걸린다. 그래서 어댑터가
// 죽으면 리스가 만료되고 ★ 같은 일감이 다시 나온다 ★.
func (c *IAP) Claim(ctx context.Context) (*RunnerRun, error) {
	var out struct {
		Run *RunnerRun `json:"run"`
	}
	if _, err := c.do(ctx, "POST", "/agent-runs/claim", nil, &out); err != nil {
		return nil, err
	}
	return out.Run, nil
}

func (c *IAP) Heartbeat(ctx context.Context, runID int) error {
	_, err := c.do(ctx, "POST", fmt.Sprintf("/agent-runs/%d/heartbeat", runID), nil, nil)
	return err
}

// Result 는 ★ 기계가 읽는 것 ★ 이다 (ADR-040 §1 ①).
//
// ★ output 에 봉인을 붓지 않는다 ★ — agent_run.output 은 지울 수 있고 스키마도
// 없다. 한 줄 요약과 run_id 만 넣고 정본은 우리 쪽 Record 에 남긴다 (ADR-005).
func (c *IAP) Result(ctx context.Context, runID int, status, output, errMsg string) error {
	body := map[string]any{"status": status}
	if output != "" {
		body["output"] = output
	}
	if errMsg != "" {
		body["error"] = errMsg
	}
	_, err := c.do(ctx, "POST", fmt.Sprintf("/agent-runs/%d/result", runID), body, nil)
	return err
}

// Comment 는 ★ 사람이 읽는 것 ★ 이다 (ADR-040 §1 ②).
// replyTo 가 0 이 아니면 그 코멘트의 답글이 된다.
func (c *IAP) Comment(ctx context.Context, issueID int, body string, replyTo int) (int, error) {
	req := map[string]any{"body": body}
	if replyTo > 0 {
		req["replyToId"] = replyTo
	}
	var out struct {
		ID int `json:"id"`
	}
	if _, err := c.do(ctx, "POST", fmt.Sprintf("/issues/%d/comments", issueID), req, &out); err != nil {
		return 0, err
	}
	return out.ID, nil
}

// Column 은 프로젝트의 칸 하나다.
type Column struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Columns 는 칸 이름을 id 로 바꾸기 위해 프로젝트를 읽는다.
//
// ★ GET /projects/{key}/columns 같은 라우트는 없다 ★ (실측 §10.5.2) —
// 프로젝트 payload 안에 실려 온다.
func (c *IAP) Columns(ctx context.Context, projectKey string) ([]Column, error) {
	var out struct {
		Columns []Column `json:"columns"`
	}
	if _, err := c.do(ctx, "GET", "/projects/"+projectKey, nil, &out); err != nil {
		return nil, err
	}
	return out.Columns, nil
}

// MoveIssue 는 ★ 보드가 읽는 것 ★ 이다 (ADR-040 §1 ③).
//
// ★ 실패해도 되돌릴 것이 없다 ★ (ADR-040 §6) — Run 판정은 이미 끝났고
// 코멘트는 이미 써졌다. 부분 성공이며 오늘의 최소선은 로그를 남기는 것이다.
func (c *IAP) MoveIssue(ctx context.Context, issueID, columnID int) error {
	_, err := c.do(ctx, "PATCH", fmt.Sprintf("/issues/%d", issueID),
		map[string]any{"columnId": columnID}, nil)
	return err
}

// Issue 는 이슈의 최소 형태다 — 제목과 본문이 계약의 목표가 된다.
type Issue struct {
	ID             int    `json:"id"`
	SequenceNumber int    `json:"sequenceNumber"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	ColumnID       int    `json:"columnId"`
}

// FeedItem 은 이슈 활동 하나다. 코멘트도 활동도 같은 표면으로 온다.
type FeedItem struct {
	ID        int    `json:"id"`
	Kind      string `json:"kind"` // comment | activity
	ReplyToID *int   `json:"replyToId"`
	ActorName string `json:"actorName"`
	Body      string `json:"body"`
	Action    string `json:"action"`
	CreatedAt string `json:"createdAt"`
}

// Feed 는 이슈의 최근 활동을 읽는다 (최신순).
//
// ★ 코멘트 전용 라우트가 없다 ★ — 코멘트는 피드의 한 종류로 온다.
func (c *IAP) Feed(ctx context.Context, issueID, limit int) ([]FeedItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	var out struct {
		Items []FeedItem `json:"items"`
	}
	path := fmt.Sprintf("/issues/%d/feed?limit=%d", issueID, limit)
	if _, err := c.do(ctx, "GET", path, nil, &out); err != nil {
		return nil, err
	}
	return out.Items, nil
}

// AlreadySaid 는 이 이슈에 그 표지를 담은 코멘트가 이미 있는가다.
//
// ★ 어댑터가 표를 안 들고 중복을 막는 방법이다 ★ — 재시작하면 진행 중이던
// 것을 다시 잡게 되는데, 결과 코멘트를 두 번 쓰면 이슈가 지저분해진다.
// 정본은 이슈 자신이므로 그것에 물어본다.
func (c *IAP) AlreadySaid(ctx context.Context, issueID int, marker string) (bool, error) {
	items, err := c.Feed(ctx, issueID, 100)
	if err != nil {
		return false, err
	}
	for _, it := range items {
		if it.Kind == "comment" && strings.Contains(it.Body, marker) {
			return true, nil
		}
	}
	return false, nil
}

func (c *IAP) Issue(ctx context.Context, issueID int) (*Issue, error) {
	var out Issue
	if _, err := c.do(ctx, "GET", fmt.Sprintf("/issues/%d", issueID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
