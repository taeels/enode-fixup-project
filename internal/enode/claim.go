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

	Name      string            `json:"name"`
	Uses      string            `json:"uses"`
	Kind      string            `json:"kind"`
	Agent     json.RawMessage   `json:"agent,omitempty"`
	Run       []string          `json:"run,omitempty"`
	Env       []string          `json:"env,omitempty"`
	Collect   map[string]string `json:"collect,omitempty"`
	Workspace json.RawMessage   `json:"workspace,omitempty"`
	In        struct {
		Prompt string   `json:"prompt"`
		From   []string `json:"from"`
	} `json:"in,omitempty"`
	Out       []string                   `json:"out,omitempty"`
	Schema    map[string]json.RawMessage `json:"schema,omitempty"`
	Attempt   int                        `json:"attempt,omitempty"`
	Requester string                     `json:"requester,omitempty"` // ★ 요청한 사람 ★ (R2 재료)
	Feedback  []string                   `json:"feedback,omitempty"`
	Lease     Lease                      `json:"lease"`
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

	// Creds 는 ★ 인증 주입 자리 ★ 다 (R1). nil 이면 Transparent —
	// 머신에 이미 있는 자격증명을 그대로 쓴다. 나중에 요청자 신원 / 팀 공용
	// 신원을 넣을 때 ★ 이 필드만 갈아끼운다 ★.
	Creds Credentials
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
		w.runAgentStep(runCtx, ctx, step, dir, in, out, stamp, log)
		return
	}
	if len(step.Run) == 0 {
		_ = w.Client.Report(ctx, step.RunID, step.Seq, Result{
			Node: w.Ident.NodeID, Error: "명령 단계인데 run 이 비었다"})
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
		log.Warn("argv 에 안 풀린 변수가 있다 — 셸을 안 거친다", "names", left)
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

	res := Result{Node: w.Ident.NodeID, Produced: w.uploadProduced(ctx, step, out, stamp, log)}

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
func (w *Worker) runAgentStep(runCtx, ctx context.Context, step *Step, dir, in, out string, stamp Stamp, log *slog.Logger) {
	p, err := parseAgentParams(step.Agent)
	if err != nil {
		_ = w.Client.Report(ctx, step.RunID, step.Seq, Result{
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
		_ = w.Client.Report(ctx, step.RunID, step.Seq, Result{
			Node: w.Ident.NodeID, Error: "모르는 하네스: " + name})
		return
	}
	bin := w.Local.HarnessBin
	if bin == "" {
		bin = ha.Name()
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
	prompt := buildPrompt(step.In.Prompt, out, step.Out, step.Schema, feedback, step.Attempt)
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
		_ = w.Client.Report(ctx, step.RunID, step.Seq, Result{
			Node: w.Ident.NodeID, Error: "자격증명 준비 실패: " + err.Error()})
		return
	}
	if d := droppedNotable(); len(d) > 0 {
		// ★ 조용히 버리지 않는다 ★ — 하네스가 인증을 못 찾을 때
		// 사람이 이 줄을 보고 화이트리스트를 의심할 수 있어야 한다.
		log.Debug("환경변수를 안 넘겼다", "names", d)
	}
	logBytes, h := runHarness(runCtx, ha, bin, Job{
		Params: p, Prompt: prompt,
		IO:     IOPaths{Dir: dir, In: in, Out: out},
		Expect: step.Out, // ★ 훅이 짚을 이름 ★ — 계약이 요구한 산출물
		Stamp:  stamp,    // ★ 훅이 볼 기준 시각 ★ — git 이 못 보는 것까지
		Inject: inject,
		Emit:   func(e Event) { log.Debug("하네스 사건", "kind", e.Kind) },
	})
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
	res.Produced = w.uploadProduced(ctx, step, out, stamp, log)
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
			log.Warn("워크스페이스 diff 를 못 걷었다", "err", err)
		case n > 0:
			log.Info("워크스페이스 diff", "bytes", n)
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
			log.Info("collect 로 걷었다", "names", got)
		}
		for _, n := range notes {
			log.Warn("collect 실패", "name", n.Name, "why", n.Why)
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
			log.Warn("변경 목록을 못 걷었다", "err", err)
		}
	}
	if total == 0 && len(missing) == 0 && len(cnotes) == 0 {
		return // 적을 것이 없다
	}

	var b strings.Builder
	if len(missing) > 0 {
		b.WriteString("계약이 요구했는데 $OUT 에 없는 것: ")
		b.WriteString(strings.Join(missing, ", "))
		b.WriteString("\n\n")
		log.Warn("요구된 산출물이 $OUT 에 없다", "missing", missing, "changed", total)
	}
	// ★ collect 가 왜 못 걷었는지 ★ — 이게 없으면 "요구했는데 없다" 만 남고
	// 사람이 계약과 트리를 대조해 스스로 알아내야 한다.
	if len(cnotes) > 0 {
		b.WriteString("collect 가 못 걷은 것:\n")
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
		b.WriteString("워크스페이스에 바뀐 파일이 없다.\n" +
			"(보드·시리얼처럼 파일로 남지 않는 단계라면 정상이다 — 단계가 $OUT 에 옮겨야 한다.)\n")
	}
	if err := os.WriteFile(filepath.Join(out, changedName), []byte(b.String()), 0o644); err != nil {
		log.Warn("변경 기록을 못 남겼다", "err", err)
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
