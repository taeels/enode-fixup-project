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
	Out       []string        `json:"out,omitempty"`
	Lease     Lease           `json:"lease"`
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

type Result struct {
	Node     string   `json:"node"`
	ExitCode *int     `json:"exit_code,omitempty"`
	Produced []string `json:"produced,omitempty"`
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
		w.execute(ctx, step)
	}
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

	if step.Kind != "run" {
		// agent 단계는 S9. 지금 실행하면 판정 없이 성공으로 보이게 된다.
		log.Warn("agent 단계는 아직 구현하지 않았다")
		_ = w.Client.Report(ctx, step.RunID, step.Seq, Result{
			Node: w.Ident.NodeID, Error: "agent 단계 미구현 (S9)"})
		return
	}

	out, err := os.MkdirTemp("", "enode-out-")
	if err != nil {
		log.Error("$OUT 을 만들 수 없다", "err", err)
		return
	}
	defer os.RemoveAll(out)

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

	cmd := exec.CommandContext(runCtx, step.Run[0], step.Run[1:]...)
	cmd.Dir = dir
	// $OUT 에 이름별 파일로 배출한다 (ADR-013) — stdout JSON 은 로그와 섞이고
	// 구조화 출력 API 는 하네스마다 다르다. 파일은 어떤 하네스든 쓸 수 있다.
	cmd.Env = append(os.Environ(), "OUT="+out)
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf

	start := time.Now()
	runErr := cmd.Run()
	code := cmd.ProcessState.ExitCode()

	produced := harvest(out)
	res := Result{Node: w.Ident.NodeID, Produced: produced}

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
		log.Info("단계 끝", "exit", code, "produced", produced,
			"took", time.Since(start).Round(time.Millisecond))
	}

	if err := w.Client.Report(ctx, step.RunID, step.Seq, res); err != nil && ctx.Err() == nil {
		log.Error("결과 보고 실패", "err", err)
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
