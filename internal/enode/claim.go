package enode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Step 은 claim 이 돌려주는 할 일 하나다.
type Step struct {
	StepID string `json:"step_id"` // run_id#NN — ★ 표시용이다 ★
	RunID  string `json:"run_id"`
	Seq    int    `json:"seq"` // 경로에 쓰는 것은 이쪽

	Name    string            `json:"name"`
	Uses    string            `json:"uses"`
	Kind    string            `json:"kind"`
	Agent   json.RawMessage   `json:"agent,omitempty"`
	Run     []string          `json:"run,omitempty"`
	Env     []string          `json:"env,omitempty"`
	Collect map[string]string `json:"collect,omitempty"`
	// CheckChanged 는 ★ 바뀌었는지 확인할 경로들 ★ 이다 (ADR-037).
	// Mediator 가 success_when 에서 뽑아 싣는다 — ★ 노드는 판정 조건을 모른다 ★.
	CheckChanged []string        `json:"check_changed,omitempty"`
	Workspace    json.RawMessage `json:"workspace,omitempty"`
	In           struct {
		Prompt string   `json:"prompt"`
		From   []string `json:"from"`
	} `json:"in,omitempty"`
	Out    []string                   `json:"out,omitempty"`
	Schema map[string]json.RawMessage `json:"schema,omitempty"`
	// Roles 는 ★ 계획이 uses 에 쓸 수 있는 이름들 ★ 이다 (ADR-045).
	// 문법이 「uses 를 적는다」까지만 말하고 ★ 무엇을 적는지 ★ 는 안 말하므로,
	// 어휘를 아는 쪽(Mediator)이 실어 보낸다.
	Roles []string `json:"roles,omitempty"`
	// Owed 는 ★ 계약이 약속했는데 아직 안 지어진 단계 이름 ★ 이다 (ADR-049).
	// ★ 이것이 곧 목표다 ★ — success_when 이 이미 그 이름을 가리키고 있다.
	Owed []string `json:"owed,omitempty"`
	// Goal 은 ★ 이 Run 이 처음 받은 목표 ★ 다 (ADR-049) — 재계획이 향할 곳.
	Goal string `json:"goal,omitempty"`
	// Expands 는 ★ 계약을 짓는 단계인가 ★ 다 (ADR-045) — 어댑터가 프롬프트에
	// 계약 문법을 심을지 정한다. ★ 노드는 그것으로 판정하지 않는다 ★.
	Expands   bool     `json:"expands,omitempty"`
	Attempt   int      `json:"attempt,omitempty"`
	Requester string   `json:"requester,omitempty"` // ★ 요청한 사람 ★ (R2 재료)
	Feedback  []string `json:"feedback,omitempty"`
	// Ledger 는 ★ 그 시점 원장의 목록 ★ 이다 — 계약이 see.ledger:"list" 라고
	// 했을 때만 온다 (ADR-023 §6.4). ★ 본문이 아니다 ★: $IN 에 파일 하나로 깔고,
	// 본문이 필요하면 계약이 in.from 에 이름을 적어 그 경로로 받는다.
	Ledger json.RawMessage `json:"ledger,omitempty"`
	Lease  Lease           `json:"lease"`
}

// planOutName 은 계획 산출물의 이름이다 — ★ 계획 단계가 아니면 빈 문자열 ★.
//
// 훅은 이 값이 있을 때만 계획을 검사한다 (ADR-046). 빈 값이면 그냥 안 본다 —
// ★ 모르는 것으로 막지 않는다 ★.
func planOutName(step *Step) string {
	if step == nil || !step.Expands || len(step.Out) != 1 {
		return ""
	}
	return step.Out[0]
}

// ledgerFile 은 원장 목록이 $IN 에 깔리는 이름이다.
// ★ 산출물 이름과 충돌하지 않아야 한다 ★ — $IN 에는 계약이 적은 이름들이 깔린다.
// 밑줄로 시작하는 산출물 이름은 ★ 계약 검증이 400 으로 막는다 ★ (contract.Validate).
// 런타임에 이름을 바꾸는 대신 검증으로 막는 이유는, 바꾸면 계약 저자가 모르기 때문이다.
const ledgerFile = "_ledger.json"

// poll 은 롱폴에 쓸 클라이언트다. 안 주어졌으면 일반 것을 쓴다 —
// ★ 없다고 동작이 달라지면 안 된다 ★.
func (c *Client) poll() *http.Client {
	if c.Poll != nil {
		return c.Poll
	}
	return c.HTTP
}

var errNoWork = errors.New("204")

// Claim 은 롱폴이다. 할 일이 없으면 204 이고 그건 ★ 정상 ★ 이다.
//
// ★ 롱폴 전용 클라이언트를 쓴다 ★ (ADR-029) — 짧은 타임아웃으로 걸면
// 그 시간에 끊고, 끊는 순간 서버가 집으면 ★ 응답이 유실되어 그 단계를
// 아무도 안 돌린다 ★. 수명은 ctx 가 관리한다.
func (c *Client) Claim(ctx context.Context, nodeID string) (*Step, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.Base+"/v1/nodes/"+nodeID+"/claim", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-Enode-Principal", c.Principal)
	if c.Instance != "" {
		// ★ 같은 생이 다시 물으면 들고 있던 것을 돌려받는다 ★ (ADR-030)
		req.Header.Set("X-Enode-Instance", c.Instance)
	}
	resp, err := c.poll().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case 204:
		return nil, errNoWork
	case 200:
		var s Step
		return &s, json.NewDecoder(resp.Body).Decode(&s)
	default:
		return nil, fmt.Errorf("claim rejected: %s", resp.Status)
	}
}

// UploadLog 는 그 단계가 뱉은 것을 원문 그대로 올린다 (ADR-005 의 logs/).
// ★ result 보다 먼저 올린다 ★ — 단계가 실패해도 로그는 남아야 한다.
func (c *Client) UploadLog(ctx context.Context, runID string, seq int, name string, body []byte) error {
	url := fmt.Sprintf("%s/v1/runs/%s/steps/%d/log?name=%s", c.Base, runID, seq, name)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-Enode-Principal", c.Principal)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("log upload rejected: %s", resp.Status)
	}
	return nil
}

// PutBlob 은 산출물을 Mediator 로 올린다 (run-contract §4 별 모양).
//
// ★ 노드끼리 직접 전송하지 않는다 ★ — 서로 다른 기계라 공유 작업공간이 없고,
// 직접 보내려면 enode 가 서로를 알아야 한다. Mediator 는 이미 전부와 말한다.
//
// 422 는 ★ 스키마 위반 ★ 이다 (ADR-020) — 저장되지 않았으므로 그 이름을
// produced 에 넣으면 안 된다. 어긴 산출물은 산출물이 아니다.
func (c *Client) PutBlob(ctx context.Context, runID string, seq int, name string, body []byte) error {
	url := fmt.Sprintf("%s/v1/runs/%s/steps/%d/blob/%s", c.Base, runID, seq, name)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-Enode-Principal", c.Principal)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return nil
}

// GetBlob 은 이전 단계의 산출물을 받는다. 이름으로 ★ 가장 최근 것 ★ 이 온다.
// ★ 리다이렉트는 http.Client 가 알아서 따른다 ★ — 나중에 Mediator 가 302 로
// 저장소를 가리켜도 이 코드는 안 바뀐다 (ADR-018).
func (c *Client) GetBlob(ctx context.Context, runID, name string, w io.Writer) error {
	url := fmt.Sprintf("%s/v1/runs/%s/blob/%s", c.Base, runID, name)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-Enode-Principal", c.Principal)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("cannot fetch blob: %s", resp.Status)
	}
	_, err = io.Copy(w, resp.Body)
	return err
}

type Result struct {
	Node     string         `json:"node"`
	ExitCode *int           `json:"exit_code,omitempty"`
	Produced []string       `json:"produced,omitempty"`
	Harness  *HarnessResult `json:"harness,omitempty"` // agent 단계만 (ADR-020)
	// Workspace 는 ★ 어떤 상태의 워크스페이스에서 돌았는가 ★ 다 (ADR-036).
	// "unprepared" 면 되돌리지 않은 자리에서 돌았다는 뜻이고,
	// ★ 그 사실이 봉인에 남아야 재현 실패의 원인을 찾을 수 있다 ★.
	Workspace Prep `json:"workspace,omitempty"`
	// Changed 는 ★ 계약이 지목한 경로 중 실제로 바뀐 것 ★ 이다 (ADR-037).
	// ★ 에이전트가 저작하지 않는 관찰 ★ — 대조는 Mediator 가 한다.
	Changed []string `json:"changed,omitempty"`
	// Error 는 ★ 완주하지 못한 ★ 경우에만 채운다.
	// 종료코드가 0 이 아닌 것은 완주다 — 그게 성공인지는 success_when 이 판정한다.
	Error string `json:"error,omitempty"`
}

func (c *Client) Report(ctx context.Context, runID string, seq int, res Result) error {
	body, err := json.Marshal(res)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/v1/runs/%s/steps/%d/result", c.Base, runID, seq), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-Enode-Principal", c.Principal)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		// ★ 서버가 받고서 거절했다 ★ — 다시 보내도 같은 답이다.
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return &ReportRejected{Status: resp.Status, Body: string(b)}
	}
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("report failed: %s %s", resp.Status, b)
	}
	return nil
}

// ReportRejected 는 서버가 ★ 받고서 거절한 ★ 보고다 (4xx).
// 유실이 아니므로 재시도 대상이 아니다 — 다시 보내도 같은 답이다.
type ReportRejected struct {
	Status string
	Body   string
}

func (e *ReportRejected) Error() string { return "report rejected: " + e.Status + " " + e.Body }

// Worker 는 일을 당겨가서 실행한다.
//
// ★ Advertiser 와 ★ 다른 고루틴 ★ 이다 ★ (ADR-016) — 주기가 다르고(하나는
// 시간 단위 롱폴, 하나는 초 단위) 의미가 다르다(하나는 일, 하나는 권한).
// 합치면 롱폴이 걸린 2시간 동안 갱신이 멈춘다.
type Worker struct {
	Client *Client
	Ident  Identity
	Local  Local
	Held   *Held
	Log    *slog.Logger

	// ReportBackoff 는 보고 재시도 간격이다. 0 이면 2초 — 시험이 줄인다.
	ReportBackoff time.Duration

	// Creds 는 ★ 인증 주입 자리 ★ 다 (R1). nil 이면 Transparent —
	// 머신에 이미 있는 자격증명을 그대로 쓴다. 나중에 요청자 신원 / 팀 공용
	// 신원을 넣을 때 ★ 이 필드만 갈아끼운다 ★.
	Creds Credentials
}

// report 는 보고가 ★ 닿을 때까지 ★ 다시 보낸다 (ADR-030).
//
// ★ 이것이 재전달의 안전을 받친다 ★ — 완주한 단계의 보고가 유실된 채 워커가
// 다음 claim 을 걸면, 장부에는 그 단계가 CLAIMED 로 남아 있으므로 같은 생
// 재전달이 그것을 돌려주고 ★ 완주한 단계가 두 번 돈다 ★. 그래서 완주한 단계를
// 든 채로는 물러서지 않는다: 성공하거나, 서버가 거절하거나(4xx — 받긴 받았다),
// ★ 임대가 죽을 때 ★ 까지 던진다. 임대가 죽으면 회수가 Run 을 정리하므로
// 유실된 보고도 함께 정리된다 — ★ 여기서도 시간이 감시자다 ★ (ADR-008).
func (w *Worker) report(ctx context.Context, step *Step, res Result) {
	for {
		err := w.Client.Report(ctx, step.RunID, step.Seq, res)
		if err == nil || ctx.Err() != nil {
			return
		}
		var rej *ReportRejected
		if errors.As(err, &rej) {
			w.Log.Error("report rejected; not retrying", "step", step.StepID, "err", err)
			return
		}
		if _, ok := w.Held.Valid(step.RunID); !ok {
			w.Log.Warn("lease ended before the report was delivered; the reaper will settle it",
				"step", step.StepID, "err", err)
			return
		}
		w.Log.Warn("report failed; retrying", "step", step.StepID, "err", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(w.backoff()):
		}
	}
}

func (w *Worker) backoff() time.Duration {
	if w.ReportBackoff > 0 {
		return w.ReportBackoff
	}
	return 2 * time.Second
}

func (w *Worker) creds() Credentials {
	if w.Creds == nil {
		return Transparent{}
	}
	return w.Creds
}

func (w *Worker) Run(ctx context.Context) {
	for ctx.Err() == nil {
		step, err := w.Client.Claim(ctx, w.Ident.NodeID)
		switch {
		case errors.Is(err, errNoWork):
			continue // 시간이 다 됐다. 즉시 다시 건다.
		case err != nil:
			if ctx.Err() == nil {
				// 롱폴 끊김은 ★ 정상 ★ 이다 (ADR-015 §5) — 중간 프록시가 끊는다.
				w.Log.Debug("claim interrupted; retrying", "err", err)
				select {
				case <-ctx.Done():
				case <-time.After(2 * time.Second):
				}
			}
			continue
		}
		w.Held.Add(step.Lease)
		w.safeExecute(ctx, step)
	}
}

// safeExecute 는 ★ 한 단계의 패닉이 노드를 죽이지 않게 한다 ★.
// 죽으면 그 노드가 든 다른 임대까지 갱신이 끊겨 무관한 Run 이 회수된다.
func (w *Worker) safeExecute(ctx context.Context, step *Step) {
	defer func() {
		if r := recover(); r != nil {
			w.Log.Error("panic while running step", "step", step.StepID, "panic", r)
			w.report(ctx, step, Result{
				Node: w.Ident.NodeID, Error: fmt.Sprintf("adapter panic: %v", r)})
		}
	}()
	w.execute(ctx, step)
}

func (w *Worker) execute(ctx context.Context, step *Step) {
	log := w.Log.With("step", step.StepID, "name", step.Name, "kind", step.Kind)

	// ★ 단계를 시작하기 전에 not_after 를 확인한다 ★ (ADR-010).
	// 지났으면 실행하지 않고 그 자리에서 멈춘다. Mediator 가 죽어도
	// 실행이 멈추는 장치가 이것이다.
	if _, ok := w.Held.Valid(step.RunID); !ok {
		log.Warn("lease is not valid; not running")
		return
	}

	out, err := os.MkdirTemp("", "enode-out-")
	if err != nil {
		log.Error("cannot create $OUT", "err", err)
		return
	}
	defer os.RemoveAll(out)

	// ★ ①사출 ★ (ADR-013 · ADR-017) — 순서가 있다:
	//   sanitize → 리비전 확인 → $IN 을 깐다 → 기동
	// 워크스페이스를 먼저 세워야 한다. clean 이 $IN 을 지우면 안 되므로
	// $IN 은 워크스페이스 밖의 임시 디렉터리다.
	var prep Prep
	if spec, err := parseWorkspace(step.Workspace); err != nil || spec != nil {
		if err == nil {
			prep, err = w.Prepare(ctx, spec, log)
		}
		if err != nil {
			log.Error("cannot prepare workspace", "err", err)
			w.report(ctx, step, Result{
				Node: w.Ident.NodeID, Error: "workspace: " + err.Error()})
			return
		}
	}

	// ★ 여기서 기준 시각을 잡는다 ★ (R5②' — changed.go)
	//
	// sanitize ★ 직후 ★ 여야 한다. 그래야 "원래 있던 것" 과 "이 단계가 만든 것" 이
	// 갈린다. 데워둔 빌드 캐시는 sanitize 가 남기므로 기준보다 오래됐고,
	// 이 단계가 새로 빌드한 것만 새 것이 된다.
	//
	// ★ git 이 못 보는 것을 여기서 본다 ★ — .gitignore 가 빌드 산출물을 정확히
	// 가리므로, git status 만으로는 훅이 zImage 도 .ko 도 못 본다.
	stamp := stampNow(w.Local.Workspace)

	// 이전 단계의 산출물을 $IN 에 이름별 파일로 깐다.
	// 별 모양이므로 노드끼리 직접 주고받지 않고 Mediator 를 경유한다.
	in, err := os.MkdirTemp("", "enode-in-")
	if err != nil {
		log.Error("cannot create $IN", "err", err)
		return
	}
	// ★ 지우려면 쓰기 권한을 돌려놔야 한다 ★ — sealInput 이 0555 로 잠근다.
	defer func() {
		_ = os.Chmod(in, 0o700)
		_ = os.RemoveAll(in)
	}()
	for _, name := range step.In.From {
		f, err := os.Create(filepath.Join(in, name))
		if err == nil {
			err = w.Client.GetBlob(ctx, step.RunID, name, f)
			f.Close()
		}
		if err != nil {
			log.Error("cannot fetch input blob", "name", name, "err", err)
			w.report(ctx, step, Result{
				Node: w.Ident.NodeID, Error: "blob " + name + " could not be fetched"})
			return
		}
	}
	// ★ 원장 목록을 $IN 에 깐다 ★ (ADR-023 §6.3.1 의 (가)) —
	// enode 가 받아서 파일로 깐다. ★ 토큰이 에이전트에 안 간다 ★ (R1).
	// 에이전트가 도구로 직접 부르는 (나) 안은 새 부품이라 순연했다.
	// ★ 안 오면 안 깐다 ★ = 오늘 그대로.
	if len(step.Ledger) > 0 {
		if err := os.WriteFile(filepath.Join(in, ledgerFile), step.Ledger, 0o644); err != nil {
			// ★ 막지 않는다 ★ — 원장은 발견을 돕는 것이지 단계의 성립 조건이 아니다.
			// 없으면 없는 대로 간다 (ADR-023 §6.2.1 의 "안 깔린다. 실패가 아니다").
			log.Warn("cannot write ledger listing", "err", err)
		}
	}
	if why := sealInput(in); why != "" {
		// ★ 막지는 않는다 ★ — 잠금은 방어이지 단계의 성립 조건이 아니다.
		// 다만 조용히 넘어가면 훅의 시야 밖 쓰기가 생기므로 이유를 남긴다.
		log.Warn("cannot make $IN read-only", "why", why)
	}

	dir := w.Local.Workspace
	if dir == "" {
		dir = os.TempDir()
	}
	// ★ 실행 중에도 임대를 감시한다 ★
	//
	// ADR-010 은 "단계를 시작하기 전에 not_after 를 확인한다" 고 했는데,
	// 그것만으로는 ★ 긴 단계가 임대보다 오래 산다 ★. 실측에서 임대가 회수된 뒤
	// 7초를 더 돌았고, 그동안 Mediator 는 그 자원을 새 Run 에 줄 수 있다 —
	// ADR-008 이 "옛 Run 의 flash 가 아직 돌고 있다" 로 이름 붙인 충돌이다.
	//
	// 권위는 여전히 not_after 다. 하트비트가 한 번 실패했다고 죽이지 않는다
	// (ADR-016) — not_after 는 갱신 주기의 배수이므로 여러 번 놓쳐야 지난다.
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			select {
			case <-runCtx.Done():
				return
			case <-t.C:
				if _, ok := w.Held.Valid(step.RunID); !ok {
					log.Warn("lease expired; aborting the running step")
					cancel()
					return
				}
			}
		}
	}()

	// ★ 단계는 두 종류다 ★ (ADR-019 결정 3) — 노드는 합쳤지만 단계는 안 합쳤다.
	// agent 단계는 produced 로, 명령 단계는 exit_code 로 판정한다.
	if step.Kind == "agent" {
		w.runAgentStep(runCtx, ctx, step, dir, in, out, stamp, prep, log)
		return
	}
	if len(step.Run) == 0 {
		w.report(ctx, step, Result{
			Node: w.Ident.NodeID, Error: "run step has an empty argv"})
		return
	}

	// ★ argv 의 $OUT · $IN 을 푼다 ★ (argv.go)
	//
	// 셸을 안 거치므로 그대로 두면 리터럴로 넘어간다. 이게 풀려야
	// `make modules_install INSTALL_MOD_PATH=$OUT` 처럼 ★ 빌드가 직접 $OUT 에
	// 놓게 ★ 시킬 수 있고, 그러면 ★ 아무도 산출물 경로를 미리 몰라도 된다 ★.
	argv := expandIO(step.Run, IOPaths{Dir: dir, In: in, Out: out})
	if left := unexpandedVars(argv); len(left) > 0 {
		// ★ 조용히 틀리게 두지 않는다 ★ — 셸이 없어 안 풀린 이름을 알려준다.
		// 막지는 않는다: 판정은 success_when 몫이다 (ADR-004 · I3).
		log.Warn("argv contains unexpanded variables; no shell is used", "names", left)
	}

	cmd := exec.CommandContext(runCtx, argv[0], argv[1:]...)
	cmd.Dir = dir
	// ★ 명령 단계도 화이트리스트다 ★ (R1)
	//
	// 처음엔 agent 쪽만 고쳤는데, 계약은 ★ 노드 주인이 아닌 사람 ★ 이 낼 수 있고
	// argv 는 무엇이든 될 수 있다 — `sh -c 'env > $OUT/leak'` 이면 끝난다.
	// 위협이 같으므로 규칙도 같다.
	//
	// 다만 빌드는 환경이 더 필요하다. 기본 목록(commandEnv)에 흔한 것을 담고,
	// 나머지는 ★ 계약이 이름으로 선언한다 ★ (steps[].env) — 값이 아니라 이름이라
	// 자격증명이 Run Record 에 봉인되는 일이 없다.
	// $OUT 에 이름별 파일로 배출한다 (ADR-013).
	cmd.Env = harnessEnv(append(commandEnv, step.Env...),
		map[string]string{"OUT": out, "IN": in}, nil)
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf

	start := time.Now()
	runErr := cmd.Run()
	code := cmd.ProcessState.ExitCode()

	res := Result{Node: w.Ident.NodeID, Workspace: prep,
		Produced: w.uploadProduced(ctx, step, out, stamp, log),
		Changed:  CheckChanged(stamp, step.CheckChanged)}

	// ★ 로그를 먼저 올린다 ★ — 단계가 실패해도 원문은 남아야 한다.
	// 여기서 실패해도 결과 보고는 계속한다. 로그가 없다고 Run 을 멈출 이유는 없다.
	if err := w.Client.UploadLog(ctx, step.RunID, step.Seq, step.Name, buf.Bytes()); err != nil && ctx.Err() == nil {
		log.Warn("log upload failed", "err", err)
	}

	switch {
	case runCtx.Err() != nil && ctx.Err() == nil:
		// ★ 임대가 끝나 중단됐다 — 완주가 아니다 ★
		res.Error = "aborted: lease expired"
		log.Warn("aborted: lease expired")
	case runErr != nil && code < 0:
		// 프로세스를 못 띄웠다 (실행 파일 없음 등) — 완주가 아니다
		res.Error = runErr.Error()
		log.Error("cannot execute", "err", runErr)
	default:
		// ★ 완주했다. 종료코드가 무엇이든. ★
		// exit 2 로 끝난 빌드도 완주한 것이고, 성공 여부는 success_when 이 판정한다
		// (ADR-004 · I3). 여기서 판정하면 O4 가 성립하지 않는다.
		res.ExitCode = &code
		log.Info("step finished", "exit", code, "produced", res.Produced,
			"took", time.Since(start).Round(time.Millisecond))
	}

	w.report(ctx, step, res)
}

// runAgentStep 은 ADR-013 의 어댑터 넷 중 ②기동을 부르고 ④수확으로 잇는다.
// ①사출은 위에서 이미 했다 ($IN + 프롬프트 조립).
func (w *Worker) runAgentStep(runCtx, ctx context.Context, step *Step, dir, in, out string, stamp Stamp, prep Prep, log *slog.Logger) {
	p, err := parseAgentParams(step.Agent)
	if err != nil {
		w.report(ctx, step, Result{
			Node: w.Ident.NodeID, Error: err.Error()})
		return
	}
	// ★ 어댑터를 고른다 ★ — 계약의 harness 속성이 곧 이름이다 (ADR-019:
	// capability 어휘는 agent.reason 하나뿐이고 구별은 전부 속성이 한다).
	name := p.Harness
	if name == "" {
		name = "claude"
	}
	ha, ok := harnessFor(name)
	if !ok {
		// ★ 조용히 claude 로 떨어뜨리지 않는다 ★ — 계약이 요구한 하네스가
		// 아닌 것으로 돌면 Record 가 거짓을 남긴다.
		w.report(ctx, step, Result{
			Node: w.Ident.NodeID, Error: "unknown harness: " + name})
		return
	}
	bin := w.Local.HarnessBin
	if bin == "" {
		bin = ha.Name()
	}

	// 되먹임 — 앞의 산출물을 프롬프트에 싣는다 (ADR-013 의 루프).
	// ★ 되먹이는 것이 LLM 의 의견이 아니라 검증기·빌드의 출력이다 ★
	//
	// ★ 두 갈래를 가른다 ★ (ADR-048)
	//
	//	★ 자백(_cannot) ★     ★ 재시도일 때만 ★ — 「앞 시도의 나」가 남긴 것이다.
	//	                       첫 시도에 앞 시도의 자백이 실리면 거짓이 된다.
	//	★ 계약이 적은 이름 ★  ★ 언제나 ★ — 그것은 ★ 남의 산출물 ★ 이지
	//	                       내 앞 시도가 아니다. 재시도와 아무 상관이 없다.
	//
	// 예전에는 둘 다 attempt > 0 에 묶여 있었다. 그래서 계획이 지은 재계획 단계가
	// ★ 앞 단계 로그를 하나도 못 봤다 ★ — expands 로 붙은 단계는 attempt 0 이라
	// 「첫 시도」이기 때문이다. 실측에서 밟았다: 재계획 에이전트가
	// "요청 섹션이 비어 있고 입력 디렉터리도 비어 있어 무엇을 고칠지 모르겠다" 며
	// _cannot 을 남겼다 (ADR-038 이 그 자리를 받아준 것은 옳게 동작한 것이다).
	feedback := map[string]string{}
	get := func(n string) {
		if _, dup := feedback[n]; dup {
			return
		}
		var buf bytes.Buffer
		if err := w.Client.GetBlob(ctx, step.RunID, n, &buf); err == nil {
			feedback[n] = buf.String()
		}
	}
	// ★ 계약이 지목한 것은 회차와 무관하게 싣는다 ★
	for _, n := range step.Feedback {
		get(n)
	}
	if step.Attempt > 0 {
		// ★ 자백은 계약이 안 적어도 되먹인다 ★ (ADR-038) —
		// 앞 시도가 "왜 못 했는지" 를 남겼으면 다음 시도가 그것을 봐야 한다.
		// 계약 저자가 feedback 에 _cannot 을 적을 수는 없다 — ★ 밑줄은 예약이라
		// 계약이 그 이름을 못 쓴다 ★. 그래서 여기서 붙인다.
		get(cannotName)
	}

	log.Debug("preparing agent step", "attempt", step.Attempt,
		"feedback_names", step.Feedback, "feedback_got", len(feedback),
		"out", step.Out, "schema", len(step.Schema))
	prompt := buildPrompt(step.In.Prompt, out, step.Out, step.Schema, feedback,
		step.Attempt, step.Expands, step.Roles, step.Owed, step.Goal)
	writePromptFile(out, prompt)

	// ★ R1 — 부모 환경을 통째로 물려주지 않는다 ★
	// 고치기 전에는 os.Environ() 을 그대로 얹어 ENODE_TOKEN 이 에이전트 손에 갔다.
	inject, err := w.creds().For(ctx, RunIdentity{
		RunID: step.RunID, Step: step.Name,
		Requester: step.Requester, NodeOwner: w.Ident.Principal,
	})
	if err != nil {
		// ★ 자격증명을 못 만들었으면 안 돌린다 ★ — 조용히 없는 채로 돌리면
		// 하네스가 엉뚱한 신원으로 붙거나 알 수 없는 이유로 실패한다.
		w.report(ctx, step, Result{
			Node: w.Ident.NodeID, Error: "cannot prepare credentials: " + err.Error()})
		return
	}
	if d := droppedNotable(); len(d) > 0 {
		// ★ 조용히 버리지 않는다 ★ — 하네스가 인증을 못 찾을 때
		// 사람이 이 줄을 보고 화이트리스트를 의심할 수 있어야 한다.
		log.Debug("environment variables not passed through", "names", d)
	}
	logBytes, h := runHarness(runCtx, ha, bin, Job{
		Params: p, Prompt: prompt,
		IO:     IOPaths{Dir: dir, In: in, Out: out},
		Expect: step.Out, // ★ 훅이 짚을 이름 ★ — 계약이 요구한 산출물
		// ★ 계획 단계면 훅이 모양까지 본다 ★ (ADR-046).
		// expands 단계는 산출물이 정확히 하나임을 계약 검증이 보장한다.
		Plan:   planOutName(step),
		Roles:  step.Roles,
		Stamp:  stamp, // ★ 훅이 볼 기준 시각 ★ — git 이 못 보는 것까지
		Inject: inject,
		Emit:   func(e Event) { log.Debug("harness event", "kind", e.Kind) },
	})
	_ = w.Client.UploadLog(ctx, step.RunID, step.Seq, step.Name, logBytes)

	res := Result{Node: w.Ident.NodeID, Harness: &h, Workspace: prep,
		Changed: CheckChanged(stamp, step.CheckChanged)}
	if !h.Reason.Completed() {
		// ★ 크래시는 완주가 아니다 ★ — 반쯤 쓴 파일을 믿을 수 없다
		res.Error = "harness: " + string(h.Reason) + " " + h.Message
		log.Warn("harness did not complete", "reason", h.Reason, "msg", h.Message)
		w.report(ctx, step, res)
		return
	}
	// ④수확 — 올라간 것만 produced 다. 스키마를 어긴 것은 422 로 거절된다.
	res.Produced = w.uploadProduced(ctx, step, out, stamp, log)
	log.Info("agent step finished", "reason", h.Reason, "turns", h.Turns,
		"cost_usd", h.CostUSD, "produced", res.Produced)
	w.report(ctx, step, res)
}

// uploadProduced 는 ④수확이다 — $OUT 을 걷어 올린다.
//
// ★ 올라간 것만 produced 다 ★ — 스키마를 어긴 것은 422 로 거절되어
// 저장되지 않았고, ★ 어긴 산출물은 산출물이 아니다 ★ (ADR-020).
func (w *Worker) uploadProduced(ctx context.Context, step *Step, out string, stamp Stamp, log *slog.Logger) []string {
	var collectNotes []collectNote
	// ★ R5② — 워크스페이스 변경을 걷는다 ★ (ADR-017 결정 6)
	//
	// ★ 여기 두는 이유 ★ — agent 단계와 명령 단계가 ★ 둘 다 ★ 이 함수를 지난다.
	// 처음엔 agent 쪽에만 붙였는데, 그건 ADR 문장의 「에이전트 산출물」을
	// 그대로 조건문으로 옮긴 것이었다. 명령 단계도 `git apply` 를 하거나
	// make 가 추적 파일을 재생성하면 ★ 같은 흔적을 남긴다 ★.
	//
	// $OUT 에 놓으면 아래 수확이 그대로 걷는다 — 새 전송 경로가 없다.
	//
	// ★ 실패해도 단계를 죽이지 않는다 ★ — 판정은 success_when 이 한다
	// (ADR-004 · I3). 기록 수단이 정규 경로를 무너뜨리면 안 된다.
	if w.Local.Workspace != "" && len(step.Workspace) > 0 {
		switch n, err := writeWorkspaceDiff(ctx, w.Local.Workspace, out, maxBlobBytes); {
		case err != nil:
			log.Warn("cannot collect workspace diff", "err", err)
		case n > 0:
			log.Info("workspace diff", "bytes", n)
		}
	}

	// ★ 기록이 스스로 설명하게 한다 ★ (R5②')
	//
	// ★ 명령 단계에는 훅이 없다 ★ — 되물을 상대가 스크립트다. 그런데 빌드·플래시가
	// 전부 명령 단계이고, ADR-019 로 그것들이 같은 노드의 cap 아래 들어왔다.
	// agent 단계는 훅이 모델에게 되묻지만, 명령 단계는 ★ 기록이 말해야 한다 ★:
	//
	//	"vmlinux 와 .ko 12개를 만들었는데 계약이 요구한 artifact 는 $OUT 에 없다"
	//
	// 이 한 문장이 없으면 사람이 로그를 뒤져 스스로 이어붙여야 하고,
	// 그건 우리가 없애려던 바로 그 상태다 (ADR-005 이유 2).
	//
	// diff 로는 안 된다 — .gitignore 가 빌드 산출물을 정확히 가려서
	// ★ 빌드 단계는 diff 만 보면 아무 일도 안 한 것처럼 보인다 ★.
	// ★ collect 가 먼저다 ★ — 계약이 적은 경로를 $OUT 으로 옮긴 뒤라야
	// "요구했는데 없는 것" 이 정확해진다.
	if got, notes := collectDeclared(w.Local.Workspace, out, step.Collect); len(got) > 0 || len(notes) > 0 {
		if len(got) > 0 {
			log.Info("collected", "names", got)
		}
		for _, n := range notes {
			log.Warn("collect failed", "name", n.Name, "why", n.Why)
		}
		collectNotes = notes
	}

	writeChangedNote(out, step.Out, stamp, collectNotes, log)

	var produced []string
	for _, name := range harvest(out) {
		if strings.HasPrefix(name, ".enode-") {
			continue // 어댑터가 남긴 것 (프롬프트 등) 은 산출물이 아니다
		}
		body, err := os.ReadFile(filepath.Join(out, name))
		if err != nil {
			log.Error("cannot read output", "name", name, "err", err)
			continue
		}
		if err := w.Client.PutBlob(ctx, step.RunID, step.Seq, name, body); err != nil {
			log.Warn("blob rejected; not listed in produced", "name", name, "err", err)
			continue
		}
		produced = append(produced, name)
	}
	return produced
}

// writeChangedNote 는 「무엇을 만들었고 무엇을 안 냈나」를 한 파일로 남긴다.
//
// ★ 판정하지 않는다 ★ — 판정은 success_when 이 한다 (ADR-004 · I3).
// 여기서는 사실만 적는다. 실패해도 단계를 죽이지 않는다.
func writeChangedNote(out string, want []string, stamp Stamp, cnotes []collectNote, log *slog.Logger) {
	have := map[string]bool{}
	for _, n := range harvest(out) {
		have[n] = true
	}
	var missing []string
	for _, n := range want {
		if !have[n] {
			missing = append(missing, n)
		}
	}

	var found []Changed
	var total int
	if stamp.Root != "" {
		var err error
		if found, total, err = changedSince(stamp, 2000); err != nil {
			log.Warn("cannot collect change list", "err", err)
		}
	}
	if total == 0 && len(missing) == 0 && len(cnotes) == 0 {
		return // 적을 것이 없다
	}

	var b strings.Builder
	if len(missing) > 0 {
		b.WriteString("required by the contract but missing from $OUT: ")
		b.WriteString(strings.Join(missing, ", "))
		b.WriteString("\n\n")
		log.Warn("required outputs are missing from $OUT", "missing", missing, "changed", total)
	}
	// ★ collect 가 왜 못 걷었는지 ★ — 이게 없으면 "요구했는데 없다" 만 남고
	// 사람이 계약과 트리를 대조해 스스로 알아내야 한다.
	if len(cnotes) > 0 {
		b.WriteString("collect could not gather:\n")
		for _, n := range cnotes {
			b.WriteString("  " + n.Name + " — " + n.Why + "\n")
		}
		b.WriteString("\n")
	}
	if total > 0 {
		b.WriteString(summarize(found, total, 40))
	} else {
		// ★ 이 경우가 보드 단계다 ★ — 파일시스템에 흔적이 없다.
		// 시리얼 출력이 산출물이므로 단계가 직접 $OUT 에 옮겨야 한다.
		b.WriteString("no files changed in the workspace.\n" +
			"(expected for steps whose result is not a file; the step must write to $OUT.)\n")
	}
	if err := os.WriteFile(filepath.Join(out, changedName), []byte(b.String()), 0o644); err != nil {
		log.Warn("cannot write change note", "err", err)
	}
}

// harvest 는 $OUT 에 이름별로 놓인 것을 걷는다 (ADR-013 ④수확).
func harvest(dir string) []string {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range ents {
		if !e.IsDir() {
			names = append(names, filepath.Base(e.Name()))
		}
	}
	return names
}
