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

	Name      string          `json:"name"`
	Uses      string          `json:"uses"`
	Kind      string          `json:"kind"`
	Agent     json.RawMessage `json:"agent,omitempty"`
	Run       []string        `json:"run,omitempty"`
	Workspace json.RawMessage `json:"workspace,omitempty"`
	In        struct {
		Prompt string   `json:"prompt"`
		From   []string `json:"from"`
	} `json:"in,omitempty"`
	Out      []string                   `json:"out,omitempty"`
	Schema   map[string]json.RawMessage `json:"schema,omitempty"`
	Attempt  int                        `json:"attempt,omitempty"`
	Feedback []string                   `json:"feedback,omitempty"`
	Lease    Lease                      `json:"lease"`
}

var errNoWork = errors.New("204")

// Claim 은 롱폴이다. 할 일이 없으면 204 이고 그건 ★ 정상 ★ 이다.
func (c *Client) Claim(ctx context.Context, nodeID string) (*Step, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.Base+"/v1/nodes/"+nodeID+"/claim", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-Enode-Principal", c.Principal)
	resp, err := c.HTTP.Do(req)
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
		return nil, fmt.Errorf("claim 거절: %s", resp.Status)
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
		return fmt.Errorf("로그 거절: %s", resp.Status)
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
		return fmt.Errorf("산출물을 못 받았다: %s", resp.Status)
	}
	_, err = io.Copy(w, resp.Body)
	return err
}

type Result struct {
	Node     string         `json:"node"`
	ExitCode *int           `json:"exit_code,omitempty"`
	Produced []string       `json:"produced,omitempty"`
	Harness  *HarnessResult `json:"harness,omitempty"` // agent 단계만 (ADR-020)
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
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("보고 거절: %s %s", resp.Status, b)
	}
	return nil
}

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
				w.Log.Debug("claim 끊김 — 다시 건다", "err", err)
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
			w.Log.Error("단계 실행 중 패닉", "step", step.StepID, "panic", r)
			_ = w.Client.Report(ctx, step.RunID, step.Seq, Result{
				Node: w.Ident.NodeID, Error: fmt.Sprintf("어댑터 패닉: %v", r)})
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
		log.Warn("임대가 유효하지 않다 — 실행하지 않는다")
		return
	}

	out, err := os.MkdirTemp("", "enode-out-")
	if err != nil {
		log.Error("$OUT 을 만들 수 없다", "err", err)
		return
	}
	defer os.RemoveAll(out)

	// ★ ①사출 ★ (ADR-013 · ADR-017) — 순서가 있다:
	//   sanitize → 리비전 확인 → $IN 을 깐다 → 기동
	// 워크스페이스를 먼저 세워야 한다. clean 이 $IN 을 지우면 안 되므로
	// $IN 은 워크스페이스 밖의 임시 디렉터리다.
	if spec, err := parseWorkspace(step.Workspace); err != nil || spec != nil {
		if err == nil {
			err = w.Prepare(ctx, spec, log)
		}
		if err != nil {
			log.Error("워크스페이스를 세울 수 없다", "err", err)
			_ = w.Client.Report(ctx, step.RunID, step.Seq, Result{
				Node: w.Ident.NodeID, Error: "워크스페이스: " + err.Error()})
			return
		}
	}

	// 이전 단계의 산출물을 $IN 에 이름별 파일로 깐다.
	// 별 모양이므로 노드끼리 직접 주고받지 않고 Mediator 를 경유한다.
	in, err := os.MkdirTemp("", "enode-in-")
	if err != nil {
		log.Error("$IN 을 만들 수 없다", "err", err)
		return
	}
	defer os.RemoveAll(in)
	for _, name := range step.In.From {
		f, err := os.Create(filepath.Join(in, name))
		if err == nil {
			err = w.Client.GetBlob(ctx, step.RunID, name, f)
			f.Close()
		}
		if err != nil {
			log.Error("이전 단계 산출물을 못 받았다", "name", name, "err", err)
			_ = w.Client.Report(ctx, step.RunID, step.Seq, Result{
				Node: w.Ident.NodeID, Error: "산출물 " + name + " 을 못 받았다"})
			return
		}
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
					log.Warn("임대가 끝났다 — 실행 중인 단계를 중단한다")
					cancel()
					return
				}
			}
		}
	}()

	// ★ 단계는 두 종류다 ★ (ADR-019 결정 3) — 노드는 합쳤지만 단계는 안 합쳤다.
	// agent 단계는 produced 로, 명령 단계는 exit_code 로 판정한다.
	if step.Kind == "agent" {
		w.runAgentStep(runCtx, ctx, step, dir, in, out, log)
		return
	}
	if len(step.Run) == 0 {
		_ = w.Client.Report(ctx, step.RunID, step.Seq, Result{
			Node: w.Ident.NodeID, Error: "명령 단계인데 run 이 비었다"})
		return
	}

	cmd := exec.CommandContext(runCtx, step.Run[0], step.Run[1:]...)
	cmd.Dir = dir
	// $OUT 에 이름별 파일로 배출한다 (ADR-013) — stdout JSON 은 로그와 섞이고
	// 구조화 출력 API 는 하네스마다 다르다. 파일은 어떤 하네스든 쓸 수 있다.
	cmd.Env = append(os.Environ(), "OUT="+out, "IN="+in)
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf

	start := time.Now()
	runErr := cmd.Run()
	code := cmd.ProcessState.ExitCode()

	res := Result{Node: w.Ident.NodeID, Produced: w.uploadProduced(ctx, step, out, log)}

	// ★ 로그를 먼저 올린다 ★ — 단계가 실패해도 원문은 남아야 한다.
	// 여기서 실패해도 결과 보고는 계속한다. 로그가 없다고 Run 을 멈출 이유는 없다.
	if err := w.Client.UploadLog(ctx, step.RunID, step.Seq, step.Name, buf.Bytes()); err != nil && ctx.Err() == nil {
		log.Warn("로그 업로드 실패", "err", err)
	}

	switch {
	case runCtx.Err() != nil && ctx.Err() == nil:
		// ★ 임대가 끝나 중단됐다 — 완주가 아니다 ★
		res.Error = "임대 만료로 중단됨"
		log.Warn("임대 만료로 중단됨")
	case runErr != nil && code < 0:
		// 프로세스를 못 띄웠다 (실행 파일 없음 등) — 완주가 아니다
		res.Error = runErr.Error()
		log.Error("실행할 수 없다", "err", runErr)
	default:
		// ★ 완주했다. 종료코드가 무엇이든. ★
		// exit 2 로 끝난 빌드도 완주한 것이고, 성공 여부는 success_when 이 판정한다
		// (ADR-004 · I3). 여기서 판정하면 O4 가 성립하지 않는다.
		res.ExitCode = &code
		log.Info("단계 끝", "exit", code, "produced", res.Produced,
			"took", time.Since(start).Round(time.Millisecond))
	}

	if err := w.Client.Report(ctx, step.RunID, step.Seq, res); err != nil && ctx.Err() == nil {
		log.Error("결과 보고 실패", "err", err)
	}
}

// runAgentStep 은 ADR-013 의 어댑터 넷 중 ②기동을 부르고 ④수확으로 잇는다.
// ①사출은 위에서 이미 했다 ($IN + 프롬프트 조립).
func (w *Worker) runAgentStep(runCtx, ctx context.Context, step *Step, dir, in, out string, log *slog.Logger) {
	p, err := parseAgentParams(step.Agent)
	if err != nil {
		_ = w.Client.Report(ctx, step.RunID, step.Seq, Result{
			Node: w.Ident.NodeID, Error: err.Error()})
		return
	}
	bin := w.Local.HarnessBin
	if bin == "" {
		bin = "claude"
	}

	// 되먹임 — 앞 시도의 산출물을 프롬프트에 싣는다 (ADR-013 의 루프).
	// ★ 되먹이는 것이 LLM 의 의견이 아니라 검증기·빌드의 출력이다 ★
	feedback := map[string]string{}
	if step.Attempt > 0 {
		for _, n := range step.Feedback {
			var buf bytes.Buffer
			if err := w.Client.GetBlob(ctx, step.RunID, n, &buf); err == nil {
				feedback[n] = buf.String()
			}
		}
	}

	log.Debug("agent 단계 준비", "attempt", step.Attempt,
		"feedback_names", step.Feedback, "feedback_got", len(feedback),
		"out", step.Out, "schema", len(step.Schema))
	prompt := buildPrompt(step.In.Prompt, step.Out, step.Schema, feedback, step.Attempt)
	writePromptFile(out, prompt)

	logBytes, h := runAgent(runCtx, bin, p, prompt, dir, in, out)
	_ = w.Client.UploadLog(ctx, step.RunID, step.Seq, step.Name, logBytes)

	res := Result{Node: w.Ident.NodeID, Harness: &h}
	if !h.Reason.Completed() {
		// ★ 크래시는 완주가 아니다 ★ — 반쯤 쓴 파일을 믿을 수 없다
		res.Error = "하네스: " + string(h.Reason) + " " + h.Message
		log.Warn("하네스가 완주하지 못했다", "reason", h.Reason, "msg", h.Message)
		_ = w.Client.Report(ctx, step.RunID, step.Seq, res)
		return
	}
	// ④수확 — 올라간 것만 produced 다. 스키마를 어긴 것은 422 로 거절된다.
	res.Produced = w.uploadProduced(ctx, step, out, log)
	log.Info("agent 단계 끝", "reason", h.Reason, "turns", h.Turns,
		"cost_usd", h.CostUSD, "produced", res.Produced)
	if err := w.Client.Report(ctx, step.RunID, step.Seq, res); err != nil && ctx.Err() == nil {
		log.Error("결과 보고 실패", "err", err)
	}
}

// uploadProduced 는 ④수확이다 — $OUT 을 걷어 올린다.
//
// ★ 올라간 것만 produced 다 ★ — 스키마를 어긴 것은 422 로 거절되어
// 저장되지 않았고, ★ 어긴 산출물은 산출물이 아니다 ★ (ADR-020).
func (w *Worker) uploadProduced(ctx context.Context, step *Step, out string, log *slog.Logger) []string {
	var produced []string
	for _, name := range harvest(out) {
		if strings.HasPrefix(name, ".enode-") {
			continue // 어댑터가 남긴 것 (프롬프트 등) 은 산출물이 아니다
		}
		body, err := os.ReadFile(filepath.Join(out, name))
		if err != nil {
			log.Error("산출물을 못 읽었다", "name", name, "err", err)
			continue
		}
		if err := w.Client.PutBlob(ctx, step.RunID, step.Seq, name, body); err != nil {
			log.Warn("산출물이 거절됐다 — produced 에 넣지 않는다", "name", name, "err", err)
			continue
		}
		produced = append(produced, name)
	}
	return produced
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
