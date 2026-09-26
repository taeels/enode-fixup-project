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
	"path/filepath"
	"strings"
	"time"

	"github.com/taeels/enode/internal/contract"
	execenv "github.com/taeels/enode/internal/environment"
)

// Step 은 claim 이 돌려주는 할 일 하나다.
type Step struct {
	StepID string `json:"step_id"` // run_id#NN — 표시용이다
	RunID  string `json:"run_id"`
	Seq    int    `json:"seq"` // 경로에 쓰는 것은 이쪽

	Name    string            `json:"name"`
	Uses    string            `json:"uses"`
	Kind    string            `json:"kind"`
	Agent   json.RawMessage   `json:"agent,omitempty"`
	Run     []string          `json:"run,omitempty"`
	Env     []string          `json:"env,omitempty"`
	Collect map[string]string `json:"collect,omitempty"`
	// CheckChanged 는 바뀌었는지 확인할 경로들이다 (ADR-037).
	// Mediator 가 success_when 에서 뽑아 싣는다 — 노드는 판정 조건을 모른다.
	CheckChanged []string        `json:"check_changed,omitempty"`
	Workspace    json.RawMessage `json:"workspace,omitempty"`
	In           struct {
		Prompt string   `json:"prompt"`
		From   []string `json:"from"`
	} `json:"in,omitempty"`
	Out    []string                   `json:"out,omitempty"`
	Schema map[string]json.RawMessage `json:"schema,omitempty"`
	// Roles 는 계획이 uses 에 쓸 수 있는 이름들이다 (ADR-045).
	// 문법이 「uses 를 적는다」까지만 말하고 무엇을 적는지는 안 말하므로,
	// 어휘를 아는 쪽(Mediator)이 실어 보낸다.
	Roles []string `json:"roles,omitempty"`
	// RoleAttrs 는 각 역할이 앉은 기계가 무엇을 광고하는가다 (ADR-055) —
	// os · host_arch · ws · arch · board …. 계획이 명령을 지을 때 쓴다.
	RoleAttrs map[string]map[string]string `json:"role_attrs,omitempty"`
	// Owed 는 계약이 약속했는데 아직 안 지어진 단계다 (ADR-049).
	// 이것이 곧 목표다 — success_when 이 이미 그 이름을 가리키고 있다.
	// When 에 그 이름에 걸린 판정이 실려 온다 — 단계의 종류가 거기서 정해진다.
	Owed []OwedStep `json:"owed,omitempty"`
	// Standing 은 이미 계약에 서 있는 단계와 그 결말이다 (ADR-052) —
	// owed 의 반대쪽이다. 계획이 자기가 어디에 붙는지 알아야
	// 이름을 겹쳐 쓰지 않고, 이미 끝난 일을 또 짓지 않는다.
	Standing []StandingStep `json:"standing,omitempty"`

	// Goal 은 이 Run 이 처음 받은 목표다 (ADR-049) — 재계획이 향할 곳.
	Goal string `json:"goal,omitempty"`
	// Rejected 는 왜 되돌아왔는가다 (ADR-062) — 사람이 계획을 물린 답들.
	// 거절이 되돌림이므로 다시 도는 것은 이 단계 자신이고, 그 in 은
	// 처음 그대로다. 이유가 안 오면 같은 계획을 다시 짓는다 (실측 rewind-1).
	Rejected []Rejection `json:"rejected,omitempty"`
	// Expands 는 계약을 짓는 단계인가다 (ADR-045) — 어댑터가 프롬프트에
	// 계약 문법을 심을지 정한다. 노드는 그것으로 판정하지 않는다.
	Expands bool `json:"expands,omitempty"`
	// EnvelopeKey 는 되먹임 봉투의 열쇠다 (ADR-050) — Mediator 가 집을 때
	// 뽑아 실어 보낸다. 어댑터가 도구 출력을 감싸는 구분자에 붙인다.
	// 비어 있으면 열쇠 없이 봉투만 씌운다 — 옛 Mediator 와도 돈다.
	EnvelopeKey string   `json:"envelope_key,omitempty"`
	Attempt     int      `json:"attempt,omitempty"`
	Requester   string   `json:"requester,omitempty"` // 요청한 사람 (R2 재료)
	Feedback    []string `json:"feedback,omitempty"`
	// Ledger 는 그 시점 원장의 목록이다 — 계약이 see.ledger:"list" 라고
	// 했을 때만 온다 (ADR-023 §6.4). 본문이 아니다: $IN 에 파일 하나로 깔고,
	// 본문이 필요하면 계약이 in.from 에 이름을 적어 그 경로로 받는다.
	Ledger json.RawMessage `json:"ledger,omitempty"`
	Lease  Lease           `json:"lease"`

	// 아래 셋은 명령이 끝난 뒤 무엇을 거두고 얼마나 기다리나다 (ADR-075 §5 · §8 · FR-3).
	// Mediator 가 계약에서 옮겨 싣는다 — 이름은 store.Claimed 와 같다. 기본값은
	// 안 싣는다. 노드가 contract.Step 의 메서드(EffectOrDefault · Budgets)로 채운다
	// (contractStep). 옛 Mediator 는 셋을 안 싣고, 그때도 기본값으로 돈다.
	Effect   contract.Effect  `json:"effect,omitempty"`
	Budget   *contract.Budget `json:"budget,omitempty"`
	Discover bool             `json:"discover,omitempty"`
}

// planOutName 은 계획 산출물의 이름이다 — 계획 단계가 아니면 빈 문자열.
//
// 훅은 이 값이 있을 때만 계획을 검사한다 (ADR-046). 빈 값이면 그냥 안 본다 —
// 모르는 것으로 막지 않는다.
func planOutName(step *Step) string {
	if step == nil || !step.Expands || len(step.Out) != 1 {
		return ""
	}
	return step.Out[0]
}

// ledgerFile 은 원장 목록이 $IN 에 깔리는 이름이다.
// 산출물 이름과 충돌하지 않아야 한다 — $IN 에는 계약이 적은 이름들이 깔린다.
// 밑줄로 시작하는 산출물 이름은 계약 검증이 400 으로 막는다 (contract.Validate).
// 런타임에 이름을 바꾸는 대신 검증으로 막는 이유는, 바꾸면 계약 저자가 모르기 때문이다.
const ledgerFile = "_ledger.json"

// poll 은 롱폴에 쓸 클라이언트다. 안 주어졌으면 일반 것을 쓴다 —
// 없다고 동작이 달라지면 안 된다.
func (c *Client) poll() *http.Client {
	if c.Poll != nil {
		return c.Poll
	}
	return c.HTTP
}

var errNoWork = errors.New("204")

// Claim 은 롱폴이다. 할 일이 없으면 204 이고 그건 정상이다.
//
// 롱폴 전용 클라이언트를 쓴다 (ADR-029) — 짧은 타임아웃으로 걸면
// 그 시간에 끊고, 끊는 순간 서버가 집으면 응답이 유실되어 그 단계를
// 아무도 안 돌린다. 수명은 ctx 가 관리한다.
func (c *Client) Claim(ctx context.Context, nodeID string) (*Step, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.Base+"/v1/nodes/"+nodeID+"/claim", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-Enode-Principal", c.Principal)
	if c.Instance != "" {
		// 같은 생이 다시 물으면 들고 있던 것을 돌려받는다 (ADR-030)
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

// GetBlob 은 이전 단계의 산출물을 받는다. 이름으로 가장 최근 것이 온다.
// 리다이렉트는 http.Client 가 알아서 따른다 — 나중에 Mediator 가 302 로
// 저장소를 가리켜도 이 코드는 안 바뀐다 (ADR-018).
// ErrNoBlob 은 그 산출물이 이 Run 에 없다는 뜻이다 (404).
// 전송 실패와 다르다 — 없는 것은 값이고, 못 가져온 것은 사고다.
var ErrNoBlob = errors.New("no such blob")

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
	// 「없다」와 「못 가져왔다」를 가른다 (ADR-058)
	//
	// 뭉뚱그리면 미디에이터 일시 장애가 「산출물 없음」으로 위장한다 —
	// 그것이 ADR-020 이 경계한 "부재는 크래시와 구분되지 않는다" 의 반대편이다.
	if resp.StatusCode == http.StatusNotFound {
		return ErrNoBlob
	}
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
	// Workspace 는 어떤 상태의 워크스페이스에서 돌았는가다 (ADR-036).
	// "unprepared" 면 되돌리지 않은 자리에서 돌았다는 뜻이고,
	// 그 사실이 봉인에 남아야 재현 실패의 원인을 찾을 수 있다.
	Workspace Prep `json:"workspace,omitempty"`
	// Changed 는 계약이 지목한 경로 중 실제로 바뀐 것이다 (ADR-037).
	// 에이전트가 저작하지 않는 관찰 — 대조는 Mediator 가 한다.
	Changed []string `json:"changed,omitempty"`
	// Error 는 완주하지 못한 경우에만 채운다.
	// 종료코드가 0 이 아닌 것은 완주다 — 그게 성공인지는 success_when 이 판정한다.
	Error string `json:"error,omitempty"`
	// Environment는 이 단계를 실제로 실행한 준비 산출물과 runtime policy다.
	Environment *execenv.Record `json:"environment,omitempty"`

	// 아래 여섯은 명령이 끝난 뒤의 구간이 어떻게 지나갔나다 (ADR-075 §10 · FR-3).
	// 이름과 모양은 store.StepResult 와 같다. Finalize 를 돈 단계에만 있다 —
	// 명령 앞에서 실패했거나 임대가 끝나 Finalize 를 건너뛴 단계에는 없고, 칸이
	// 없다는 것이 곧 「그 구간에 닿지 않았다」다.
	//
	// 판정 재료가 아니다. 예산을 넘긴 것은 Error 를 채워 완주가 아니게 만들고,
	// 여기에는 사실만 적는다. 시각은 모두 노드 시계다.
	ExitedAt    *time.Time            `json:"exited_at,omitempty"`    // 종료 status 를 받은 순간
	FinalizedAt *time.Time            `json:"finalized_at,omitempty"` // 닫기가 끝난 순간
	Finalize    contract.Stage        `json:"finalize,omitempty"`
	Upload      contract.Stage        `json:"upload,omitempty"`
	Reason      string                `json:"reason,omitempty"`
	Diagnostics *contract.Diagnostics `json:"diagnostics,omitempty"`
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
		// 서버가 받고서 거절했다 — 다시 보내도 같은 답이다.
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return &ReportRejected{Status: resp.Status, Body: string(b)}
	}
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("report failed: %s %s", resp.Status, b)
	}
	return nil
}

// Exited 는 명령 종료 보고다 (POST /v1/runs/{run}/steps/{seq}/exited · ADR-075 §10.4).
//
// 판정이 아니다 — Mediator 는 이것으로 phase 를 finalizing 으로 옮길 뿐이고 단계를
// 끝내는 것은 result 하나다. 30초 client 를 쓴다. stop 이 참이면 다시 보내지 않는다 —
// 받았거나(2xx) 받고서 거절했다(4xx · 그때 err 는 *exitRejected). 5xx 와 끊김은
// stop 이 거짓이다.
func (c *Client) Exited(ctx context.Context, runID string, seq int, e contract.Exited) (stop bool, err error) {
	body, _ := json.Marshal(e) // 시각과 정수뿐이라 실패하지 않는다
	url := fmt.Sprintf("%s/v1/runs/%s/steps/%d/exited", c.Base, runID, seq)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return true, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-Enode-Principal", c.Principal)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	switch {
	case resp.StatusCode/100 == 2:
		return true, nil
	case exitedStop(resp.StatusCode):
		return true, &exitRejected{Code: resp.StatusCode, Body: strings.TrimSpace(string(b))}
	}
	return false, fmt.Errorf("exit report failed: %s %s", resp.Status, strings.TrimSpace(string(b)))
}

// exitRejected 는 Mediator 가 받고서 거절한 종료 보고다 (4xx). 다시 보내도 같은 답이다.
type exitRejected struct {
	Code int
	Body string
}

func (e *exitRejected) Error() string {
	return fmt.Sprintf("exit report rejected: %d %s", e.Code, e.Body)
}

// ReportRejected 는 서버가 받고서 거절한 보고다 (4xx).
// 유실이 아니므로 재시도 대상이 아니다 — 다시 보내도 같은 답이다.
type ReportRejected struct {
	Status string
	Body   string
}

func (e *ReportRejected) Error() string { return "report rejected: " + e.Status + " " + e.Body }

// Worker 는 일을 당겨가서 실행한다.
//
// Advertiser 와 다른 고루틴 이다 (ADR-016) — 주기가 다르고(하나는
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

	// Creds 는 인증 주입 자리다 (R1). nil 이면 Transparent —
	// 머신에 이미 있는 자격증명을 그대로 쓴다. 나중에 요청자 신원 / 팀 공용
	// 신원을 넣을 때 이 필드만 갈아끼운다.
	Creds Credentials

	// Runtime은 agent와 command가 함께 지나는 실행 경계다. nil이면 native다.
	Runtime       StepRuntime
	RuntimeRecord *execenv.Record

	// budgets 는 단계의 두 예산을 정한다. nil 이면 stepBudgets — 계약이 적은 값과 기본값.
	// 시험만 바꿔 끼운다: 계약의 Finalize 예산은 1분 아래로 못 내리는데 조각 3 (예산)의
	// 시험은 짧은 예산이 필요하다.
	budgets func(*Step) (finalize, upload time.Duration)
	// exitWait 는 종료 보고의 재전송 간격이다. nil 이면 exitBackoff. 시험이 줄인다.
	exitWait func(n int) time.Duration

	// AfterReport 는 결과 보고가 끝나면 불린다 — 닿았든 포기했든 (trash 유닛). 배경
	// 삭제자의 Kick 이 앉는다. 막지 않아야 한다. nil 이면 안 부른다.
	AfterReport func()

	// drainingNoted 는 「안 집는다」를 이미 찍었는가다 — 광고 주기마다 다시 안 찍는다.
	drainingNoted bool

	// ring 은 이 노드의 트랜스크립트 링 파일이다 (decisions §6). Run 이 한 번 열고,
	// 단계가 시작할 때 Reset 하며, 하네스·명령 단계의 stdout 을 여기로 tee 한다.
	// nil 이면 안 흘린다(설정 없음/열기 실패) — 능력 저하일 뿐 실행을 안 막는다.
	ring *Ring
}

// transcript 는 tee 대상을 io.Writer 로 낸다. ring 이 nil 이면 typed-nil 이 아니라
// 진짜 nil 을 돌려준다 — runner 의 nil 검사가 성립하게.
// transcript 는 이 단계의 하네스 출력이 흘러갈 자리를 낸다 - 노드의 링과
// Mediator 로 미는 업로더 둘이다. 규칙이 하나다: 링으로 가는 것은 Mediator 로도
// 간다. 그래야 중앙 화면이 제어판보다 적게 보이는 일이 없다.
//
// step 을 받는 이유 - 업로더는 runID · seq · name · attempt 넷을 알아야 한다.
// 링은 노드에 하나라 몰라도 됐다.
//
// 정리 함수를 함께 내는 이유 - 업로더는 단계 끝에 남은 꼬리를 마지막으로
// 비우고 고루틴을 멈춰야 한다. 부르는 쪽이 defer 로 든다. 둘 다 없으면
// nil 과 no-op 이라 부르는 쪽에 갈래가 안 생긴다.
func (w *Worker) transcript(step *Step) (io.Writer, func()) {
	var sinks []io.Writer
	if w.ring != nil {
		sinks = append(sinks, w.ring)
	}
	stop := func() {}
	if w.Client != nil && step != nil {
		u := newUploader(w.Client, step, w.Log)
		sinks = append(sinks, u)
		stop = u.Close
	}
	switch len(sinks) {
	case 0:
		return nil, stop
	case 1:
		return sinks[0], stop
	default:
		return io.MultiWriter(sinks...), stop
	}
}

// report 는 보고가 닿을 때까지 다시 보낸다 (ADR-030).
//
// 이것이 재전달의 안전을 받친다 — 완주한 단계의 보고가 유실된 채 워커가
// 다음 claim 을 걸면, 장부에는 그 단계가 CLAIMED 로 남아 있으므로 같은 생
// 재전달이 그것을 돌려주고 완주한 단계가 두 번 돈다. 그래서 완주한 단계를
// 든 채로는 물러서지 않는다: 성공하거나, 서버가 거절하거나(4xx — 받긴 받았다),
// 임대가 죽을 때까지 던진다. 임대가 죽으면 회수가 Run 을 정리하므로
// 유실된 보고도 함께 정리된다 — 여기서도 시간이 감시자다 (ADR-008).
//
// 끝나면 AfterReport 를 부른다 — 작업 폴더는 이미 trash 에 있고, 지우기는 보고 뒤다.
func (w *Worker) report(ctx context.Context, step *Step, res Result) {
	if w.AfterReport != nil {
		defer w.AfterReport()
	}
	if res.Environment == nil {
		res.Environment = w.RuntimeRecord
	}
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
	// 트랜스크립트 링을 한 번 연다 (decisions §6). 설정이 없거나(시험) 못 열면
	// nil 로 두고 진행한다 — tee 는 보조다.
	if w.Ident.Config != "" {
		if r, err := OpenRing(TranscriptPath(w.Ident.Config), DefaultTranscriptCapacity); err == nil {
			w.ring = r
			defer w.ring.Close() //nolint:errcheck
		} else if w.Log != nil {
			w.Log.Warn("cannot open the transcript ring; the panel will show no live output",
				"path", TranscriptPath(w.Ident.Config), "err", err)
		}
	}
	for ctx.Err() == nil {
		// 소유자가 at-boundary 로 drain 을 걸었고 중앙이 받아 적었다 (ADR-063 §2.1) —
		// 더 집지 않는다. 도는 단계는 이 분기 밖에서 이미 끝까지 간다(execute 는
		// claim 뒤다). Mediator 가 경계에서 Run 을 닫으므로 올 단계도 없지만,
		// 그 취소가 실패한 경우에도 「경계에서 놓는다」가 거짓이 되지 않게 여기서 막는다.
		// graceful 은 그대로 집는다 — 새 임대만 막는 것이고 임대는 매칭이 막는다.
		if w.Held.Drain() == contract.DrainAtBoundary {
			w.noteDraining(true)
			select {
			case <-ctx.Done():
				return
			case <-time.After(w.Held.Renew()):
			}
			continue
		}
		w.noteDraining(false)
		step, err := w.Client.Claim(ctx, w.Ident.NodeID)
		switch {
		case errors.Is(err, errNoWork):
			continue // 시간이 다 됐다. 즉시 다시 건다.
		case err != nil:
			if ctx.Err() == nil {
				// 롱폴 끊김은 정상이다 (ADR-015 §5) — 중간 프록시가 끊는다.
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

// noteDraining 은 집기를 멈추고 다시 시작하는 순간을 한 번씩만 찍는다.
func (w *Worker) noteDraining(draining bool) {
	if draining == w.drainingNoted {
		return
	}
	w.drainingNoted = draining
	if draining {
		w.Log.Info("draining at-boundary; not claiming until the owner releases it")
	} else {
		w.Log.Info("drain released; claiming again")
	}
}

// safeExecute 는 한 단계의 패닉이 노드를 죽이지 않게 한다.
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

	// 단계를 시작하기 전에 not_after 를 확인한다 (ADR-010).
	// 지났으면 실행하지 않고 그 자리에서 멈춘다. Mediator 가 죽어도
	// 실행이 멈추는 장치가 이것이다.
	if _, ok := w.Held.Valid(step.RunID); !ok {
		log.Warn("lease is not valid; not running")
		return
	}

	// 새 단계가 시작할 때 링을 비운다 (decisions §6.2) — 끝날 때가 아니다.
	// 떠나 있던 사람에게도 방금 끝난 것이 남아 있어야 하므로, 다음 단계의 첫
	// 글자에서 갈린다. 화면은 세대가 바뀐 것으로 그 순간을 안다.
	if w.ring != nil {
		_ = w.ring.Reset()
	}

	out, err := os.MkdirTemp("", "enode-out-")
	if err != nil {
		log.Error("cannot create $OUT", "err", err)
		return
	}
	defer os.RemoveAll(out)

	// ①사출 (ADR-013 · ADR-017) — 순서가 있다:
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

	// 여기서 기준 시각을 잡는다 (R5②' — changed.go)
	//
	// sanitize 직후여야 한다. 그래야 "원래 있던 것" 과 "이 단계가 만든 것" 이
	// 갈린다. 데워둔 빌드 캐시는 sanitize 가 남기므로 기준보다 오래됐고,
	// 이 단계가 새로 빌드한 것만 새 것이 된다.
	//
	// git 이 못 보는 것을 여기서 본다 — .gitignore 가 빌드 산출물을 정확히
	// 가리므로, git status 만으로는 훅이 zImage 도 .ko 도 못 본다.
	stamp := stampNow(w.Local.Workspace)

	// 이전 단계의 산출물을 $IN 에 이름별 파일로 깐다.
	// 별 모양이므로 노드끼리 직접 주고받지 않고 Mediator 를 경유한다.
	in, err := os.MkdirTemp("", "enode-in-")
	if err != nil {
		log.Error("cannot create $IN", "err", err)
		return
	}
	// 지우려면 쓰기 권한을 돌려놔야 한다 — sealInput 이 0555 로 잠근다.
	defer func() {
		_ = os.Chmod(in, 0o700)
		_ = os.RemoveAll(in)
	}()
	// 없는 입력은 값이다. 크래시가 아니다 (ADR-058 · ADR-023 §6.2.1)
	//
	// §6.2.1 은 in.from 에 대해 "없으면 안 깔린다. 실패가 아니다" 라고
	// 못 박았다 — dispatch 로 안 간 가지의 산출물을 가리킬 수 있고 그것은
	// 실행 시에나 정해지기 때문이다. 그런데 코드가 그것을 안 지켰다:
	// 404 면 단계를 FAILED 로 만들었다. (아이러니하게도 바로 아래 원장 적재는
	// 같은 §6.2.1 을 인용하며 지킨다.)
	//
	//	실측 (vm-scratch-7) VM 노드가 실제로 섰는데, 재계획이 in.from 으로
	//	앞 단계가 못 낸 vm_caps 를 요구해 그 단계가 죽었다 —
	//	재계획은 실패를 고치러 도는 단계인데 실패의 증거가 없다고 죽었다.
	//
	// 조용히 넘어가지도 않는다 (ADR-020) — 못 받은 이름을 모아 프롬프트에
	// 적는다. 에이전트가 부재를 관찰 하고 판단한다.
	var missingIn []string
	for _, name := range step.In.From {
		f, err := os.Create(filepath.Join(in, name))
		if err == nil {
			err = w.Client.GetBlob(ctx, step.RunID, name, f)
			f.Close()
		}
		if err == nil {
			continue
		}
		_ = os.Remove(filepath.Join(in, name)) // 반쯤 쓴 파일을 안 남긴다
		if errors.Is(err, ErrNoBlob) {
			log.Info("input was not produced by this run; recording it as absent",
				"name", name)
			missingIn = append(missingIn, name)
			continue
		}
		log.Error("cannot fetch input blob", "name", name, "err", err)
		w.report(ctx, step, Result{
			Node: w.Ident.NodeID, Error: "blob " + name + " could not be fetched"})
		return
	}
	// 원장 목록을 $IN 에 깐다 (ADR-023 §6.3.1 의 (가)) —
	// enode 가 받아서 파일로 깐다. 토큰이 에이전트에 안 간다 (R1).
	// 에이전트가 도구로 직접 부르는 (나) 안은 새 부품이라 순연했다.
	// 안 오면 안 깐다 = 오늘 그대로.
	if len(step.Ledger) > 0 {
		if err := os.WriteFile(filepath.Join(in, ledgerFile), step.Ledger, 0o644); err != nil {
			// 막지 않는다 — 원장은 발견을 돕는 것이지 단계의 성립 조건이 아니다.
			// 없으면 없는 대로 간다 (ADR-023 §6.2.1 의 "안 깔린다. 실패가 아니다").
			log.Warn("cannot write ledger listing", "err", err)
		}
	}
	if why := sealInput(in); why != "" {
		// 막지는 않는다 — 잠금은 방어이지 단계의 성립 조건이 아니다.
		// 다만 조용히 넘어가면 훅의 시야 밖 쓰기가 생기므로 이유를 남긴다.
		log.Warn("cannot make $IN read-only", "why", why)
	}

	dir := w.Local.Workspace
	if dir == "" {
		dir = os.TempDir()
	}
	// 실행 중에도 임대를 감시한다
	//
	// ADR-010 은 "단계를 시작하기 전에 not_after 를 확인한다" 고 했는데,
	// 그것만으로는 긴 단계가 임대보다 오래 산다. 실측에서 임대가 회수된 뒤
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

	// 단계는 두 종류다 (ADR-019 결정 3) — 노드는 합쳤지만 단계는 안 합쳤다.
	// agent 단계는 produced 로, 명령 단계는 exit_code 로 판정한다.
	if step.Kind != "agent" && len(step.Run) == 0 {
		w.report(ctx, step, Result{
			Node: w.Ident.NodeID, Error: "run step has an empty argv"})
		return
	}
	runtimeImpl := w.Runtime
	if runtimeImpl == nil {
		runtimeImpl = NativeRuntime{}
	}
	session, err := runtimeImpl.Open(runCtx, RuntimeSpec{
		RunID: step.RunID, StepID: step.StepID, Dir: dir, In: in, Out: out,
		Record: w.RuntimeRecord,
	})
	if err != nil {
		w.report(ctx, step, Result{Node: w.Ident.NodeID, Workspace: prep,
			Error: "runtime open: " + err.Error()})
		return
	}
	session = manageSession(session)
	defer session.Close(ctx, Keep{}) //nolint:errcheck // 명시 Close가 오류를 결과로 옮긴다
	runtimePaths := session.Paths()
	if step.Kind == "agent" {
		w.runAgentStep(runCtx, ctx, step, dir, in, out, runtimePaths, stamp, prep, missingIn, session, log)
		return
	}

	// argv 의 $OUT · $IN 을 푼다 (argv.go)
	//
	// 셸을 안 거치므로 그대로 두면 리터럴로 넘어간다. 이게 풀려야
	// `make modules_install INSTALL_MOD_PATH=$OUT` 처럼 빌드가 직접 $OUT 에
	// 놓게 시킬 수 있고, 그러면 아무도 산출물 경로를 미리 몰라도 된다.
	argv := expandIO(step.Run, IOPaths{Dir: runtimePaths.Dir, In: runtimePaths.In, Out: runtimePaths.Out})
	if left := unexpandedVars(argv); len(left) > 0 {
		// 조용히 틀리게 두지 않는다 — 셸이 없어 안 풀린 이름을 알려준다.
		// 막지는 않는다: 판정은 success_when 몫이다 (ADR-004 · I3).
		log.Warn("argv contains unexpanded variables; no shell is used", "names", left)
	}

	// 명령 단계도 화이트리스트다 (R1)
	//
	// 처음엔 agent 쪽만 고쳤는데, 계약은 노드 주인이 아닌 사람이 낼 수 있고
	// argv 는 무엇이든 될 수 있다 — `sh -c 'env > $OUT/leak'` 이면 끝난다.
	// 위협이 같으므로 규칙도 같다.
	//
	// 다만 빌드는 환경이 더 필요하다. 기본 목록(commandEnv)에 흔한 것을 담고,
	// 나머지는 계약이 이름으로 선언한다 (steps[].env) — 값이 아니라 이름이라
	// 자격증명이 Run Record 에 봉인되는 일이 없다.
	// $OUT 에 이름별 파일로 배출한다 (ADR-013).
	processEnv := harnessEnv(append(commandEnv, step.Env...),
		map[string]string{"OUT": runtimePaths.Out, "IN": runtimePaths.In}, nil)
	var buf bytes.Buffer
	// 명령 단계 stdout/stderr 도 링과 업로더로 tee 한다 (unit-of-work §6).
	// 로그 업로드·판정은 그대로 buf 를 읽는다.
	//
	// 에이전트 단계만 올리는 길도 있으나 그러면 중앙이 제어판보다 적게 보인다 -
	// 링에는 명령 출력이 들어가는데 Mediator 에는 안 들어가서, 화면 둘이 같은
	// 단계를 두고 다른 것을 보인다. 그 비대칭을 화면이 설명할 길이 없다.
	var sink io.Writer = &buf
	cmdTee, stopCmdTee := w.transcript(step)
	if cmdTee != nil {
		sink = io.MultiWriter(&buf, cmdTee)
	}
	start := time.Now()
	code, runErr := session.Run(runCtx, ProcessSpec{
		Argv: argv, Dir: runtimePaths.Dir, Env: processEnv, Stdout: sink, Stderr: sink,
	})
	exitedAt := time.Now().UTC()
	// 꼬리를 비우는 것이 UploadLog(선별본)보다 먼저다 - 뒤집으면 봉인 직전의
	// 마지막 줄이 중앙 화면에 안 뜬다. 명령의 출력은 Run 이 돌아온 때 이미 다 왔다.
	stopCmdTee()

	res := Result{Node: w.Ident.NodeID, Workspace: prep, Environment: session.Environment()}
	outcome, started := exitOutcome(code, runErr)
	if leaseEnded := runCtx.Err() != nil && ctx.Err() == nil; leaseEnded || !started {
		// 임대가 끝나 죽였거나 프로세스가 뜨지 않았다 — 종료 보고도 Finalize 도 없다
		// (business-rules.md 2절). 닫고 단계 로그만 올린다. 새 칸은 없다 — 칸이 없는
		// 것이 곧 「그 구간에 닿지 않았다」다.
		w.closeAndUploadLog(ctx, step, session, buf.Bytes(), log)
		switch {
		case leaseEnded:
			res.Error = "aborted: lease expired"
			log.Warn("aborted: lease expired")
		default:
			res.Error = "cannot execute"
			if runErr != nil {
				res.Error = runErr.Error()
			}
			log.Error("cannot execute", "err", runErr)
		}
		w.report(ctx, step, res)
		return
	}

	// 종료 보고는 결과 확정을 막지 않는다 (ADR-075 §10.4) — goroutine 이 보내고,
	// result 를 보내기 직전에 그만둔다. 유실돼도 result 가 같은 사실을 싣는다.
	reporter := w.startExitReport(ctx, step, contract.Exited{
		Node: w.Ident.NodeID, Instance: w.Client.Instance, Attempt: step.Attempt,
		Outcome: outcome, ExitedAt: exitedAt,
	})
	res.ExitedAt = &exitedAt
	spec := finalizeSpecFor(step, true, w.Local)
	spec.Workspace, spec.Out, spec.Stamp = dir, out, stamp
	leaseEnded := w.afterExit(runCtx, ctx, step, session, spec, exitedAt, buf.Bytes(), true, &res, log)
	reporter.Stop()

	switch {
	case leaseEnded:
		// 임대가 끝나 중단됐다 — 완주가 아니다
		res.Error = "aborted: lease expired"
		log.Warn("aborted: lease expired")
	case runErr != nil && code < 0:
		// native 에서 signal 로 죽었다 — 완주가 아니다. 종료 보고만 그것을 signal 로 알렸다
		res.Error = runErr.Error()
		log.Warn("command ended by a signal", "err", runErr)
	default:
		// 완주했다. 종료코드가 무엇이든.
		// exit 2 로 끝난 빌드도 완주한 것이고, 성공 여부는 success_when 이 판정한다
		// (ADR-004 · I3). 여기서 판정하면 O4 가 성립하지 않는다.
		// 예산을 넘긴 것은 error 에 남아 완주가 아니게 되지만 exit_code 는 그대로 남는다 —
		// 명령 실패와 원인이 나뉜다 (조각 3).
		res.ExitCode = &code
		log.Info("step finished", "exit", code, "produced", res.Produced,
			"finalize", res.Finalize, "upload", res.Upload,
			"took", time.Since(start).Round(time.Millisecond))
	}

	w.report(ctx, step, res)
}

// afterExit 는 명령이 끝난 뒤의 구간을 닫는다 — Finalize · 닫기 · 업로드 · 판정 칸
// (business-logic-model.md 1 · 2절). 명령 단계와 agent 단계가 같은 모양으로 지난다.
//
//	[Finalize 예산]  from 부터.  collect · 지목 경로 stat · diff · 명시 훑기 · 닫기
//	                 닫기는 unmount 와 rename 한 번이라 예산 안이다 (trash 유닛).  마감으로
//	                 끊지는 않는다 — 반쯤 닫은 세션은 helper 와 마운트를 남긴다.  마감 뒤에
//	                 끝났으면 finalize_timeout 이다
//	finalized_at
//	[업로드 예산]    닫기가 끝난 때부터.  단계 로그 먼저 -> $OUT 의 이름들
//
// 두 ctx 모두 runCtx 에서 딴다 — runCtx 가 임대를 보므로 임대가 끝나도 멈춘다
// (ADR-075 §9). res 에 changed · produced 와 새 칸 다섯, error 의 앞부분을 채운다.
// 돌려주는 것은 그동안 임대가 끝났나다 — 명령이 완주하지 않았다는 문구는 부르는 쪽이
// 덮어쓴다.
func (w *Worker) afterExit(runCtx, ctx context.Context, step *Step, session StepSession, spec FinalizeSpec, from time.Time, logBody []byte, blobs bool, res *Result, log *slog.Logger) bool {
	finalizeBudget, uploadBudget := w.budgetsFor(step)
	spec.DiscoverFor = discoverTime(finalizeBudget)
	spec.Deadline = from.Add(finalizeBudget)
	fctx, cancel := context.WithDeadline(runCtx, spec.Deadline)
	began := time.Now()
	fin, finErr := session.Finalize(fctx, spec)
	cancel()
	closeErr := session.Close(ctx, Keep{})
	finalizedAt := time.Now().UTC()
	closedLate := finalizedAt.After(spec.Deadline) && runCtx.Err() == nil

	diag := diagnosticsFor(step, spec.Out, spec.Effect, fin)
	logFinalize(log, spec, fin, diag, time.Since(began))
	timedOut := runCtx.Err() == nil &&
		(errors.Is(finErr, context.DeadlineExceeded) || (finErr == nil && closedLate))
	tail := logTail(step, diag, fin, timedOut, finalizeBudget)

	uctx, cancel := context.WithTimeout(runCtx, uploadBudget)
	produced, uploaded := w.upload(uctx, step, spec.Out, withTail(logBody, tail), blobs, log)
	cancel()

	leaseEnded := runCtx.Err() != nil && ctx.Err() == nil
	res.Changed, res.Produced = fin.Changed, produced
	res.FinalizedAt, res.Diagnostics = &finalizedAt, diag
	res.Finalize, res.Upload, res.Reason, res.Error = settle(settleIn{
		finalizeErr: finErr, closeErr: closeErr, closedLate: closedLate, upload: uploaded,
		leaseEnded: leaseEnded, finalizeBudget: finalizeBudget, uploadBudget: uploadBudget,
	})
	return leaseEnded
}

// closeAndUploadLog 는 Finalize 에 닿지 않은 단계의 끝이다 — 닫고 단계 로그만 올린다.
// runCtx 가 끝났을 수 있으므로 Worker 의 ctx 에 업로드 예산을 건다.
func (w *Worker) closeAndUploadLog(ctx context.Context, step *Step, session StepSession, logBody []byte, log *slog.Logger) {
	if err := session.Close(ctx, Keep{}); err != nil {
		log.Warn("runtime cleanup failed", "err", err)
	}
	_, uploadBudget := w.budgetsFor(step)
	uctx, cancel := context.WithTimeout(ctx, uploadBudget)
	defer cancel()
	if err := w.Client.UploadLog(uctx, step.RunID, step.Seq, step.Name, logBody); err != nil && ctx.Err() == nil {
		log.Warn("log upload failed", "err", err)
	}
}

// runAgentStep 은 ADR-013 의 어댑터 넷 중 ②기동을 부르고 ④수확으로 잇는다.
// ①사출은 위에서 이미 했다 ($IN + 프롬프트 조립).
// missingIn 은 계약이 요청했는데 이 Run 에 없던 입력 이름들이다 (ADR-058).
// 부재를 값으로 나른다 — 프롬프트가 그것을 적어준다.
func (w *Worker) runAgentStep(runCtx, ctx context.Context, step *Step, dir, in, out string, runtimePaths RuntimePaths, stamp Stamp, prep Prep, missingIn []string, session StepSession, log *slog.Logger) {
	report := func(res Result) {
		res.Environment = session.Environment()
		if closeErr := session.Close(ctx, Keep{}); closeErr != nil {
			res.Error = "runtime cleanup: " + closeErr.Error()
		}
		w.report(ctx, step, res)
	}
	p, err := parseAgentParams(step.Agent)
	if err != nil {
		report(Result{
			Node: w.Ident.NodeID, Error: err.Error()})
		return
	}
	// 어댑터를 고른다 — 계약의 harness 속성이 곧 이름이다 (ADR-019:
	// capability 어휘는 agent.reason 하나뿐이고 구별은 전부 속성이 한다).
	name := p.Harness
	if name == "" {
		name = "claude"
	}
	ha, ok := harnessFor(name)
	if !ok {
		// 조용히 claude 로 떨어뜨리지 않는다 — 계약이 요구한 하네스가
		// 아닌 것으로 돌면 Record 가 거짓을 남긴다.
		report(Result{
			Node: w.Ident.NodeID, Error: "unknown harness: " + name})
		return
	}
	bin := w.Local.HarnessBin
	if bin == "" {
		bin = ha.Name()
	}

	// 되먹임 — 앞의 산출물을 프롬프트에 싣는다 (ADR-013 의 루프).
	// 되먹이는 것이 LLM 의 의견이 아니라 검증기·빌드의 출력이다
	//
	// 두 갈래를 가른다 (ADR-048)
	//
	//	자백(_cannot)           재시도일 때만 — 「앞 시도의 나」가 남긴 것이다.
	//	                       첫 시도에 앞 시도의 자백이 실리면 거짓이 된다.
	//	계약이 적은 이름        언제나 — 그것은 남의 산출물이지
	//	                       내 앞 시도가 아니다. 재시도와 아무 상관이 없다.
	//
	// 예전에는 둘 다 attempt > 0 에 묶여 있었다. 그래서 계획이 지은 재계획 단계가
	// 앞 단계 로그를 하나도 못 봤다 — expands 로 붙은 단계는 attempt 0 이라
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
	// 계약이 지목한 것은 회차와 무관하게 싣는다
	for _, n := range step.Feedback {
		get(n)
	}
	if step.Attempt > 0 {
		// 자백은 계약이 안 적어도 되먹인다 (ADR-038) —
		// 앞 시도가 "왜 못 했는지" 를 남겼으면 다음 시도가 그것을 봐야 한다.
		// 계약 저자가 feedback 에 _cannot 을 적을 수는 없다 — 밑줄은 예약이라
		// 계약이 그 이름을 못 쓴다. 그래서 여기서 붙인다.
		get(cannotName)
	}

	log.Debug("preparing agent step", "attempt", step.Attempt,
		"feedback_names", step.Feedback, "feedback_got", len(feedback),
		"out", step.Out, "schema", len(step.Schema))
	prompt := buildPrompt(step.In.Prompt, runtimePaths.Out, step.Out, step.Schema, feedback,
		step.Attempt, step.Expands, step.Roles, step.RoleAttrs, step.Owed,
		step.Standing, step.Rejected, missingIn, step.Goal, step.EnvelopeKey)
	writePromptFile(out, prompt)

	// R1 — 부모 환경을 통째로 물려주지 않는다
	// 고치기 전에는 os.Environ() 을 그대로 얹어 ENODE_TOKEN 이 에이전트 손에 갔다.
	inject, err := w.creds().For(ctx, RunIdentity{
		RunID: step.RunID, Step: step.Name,
		Requester: step.Requester, NodeOwner: w.Ident.Principal,
	})
	if err != nil {
		// 자격증명을 못 만들었으면 안 돌린다 — 조용히 없는 채로 돌리면
		// 하네스가 엉뚱한 신원으로 붙거나 알 수 없는 이유로 실패한다.
		report(Result{
			Node: w.Ident.NodeID, Error: "cannot prepare credentials: " + err.Error()})
		return
	}
	if d := droppedNotable(); len(d) > 0 {
		// 조용히 버리지 않는다 — 하네스가 인증을 못 찾을 때
		// 사람이 이 줄을 보고 화이트리스트를 의심할 수 있어야 한다.
		log.Debug("environment variables not passed through", "names", d)
	}
	// 워크스페이스 선언을 여기서 읽는다 — 여는 것은 가장자리이고 고르는 것은
	// resolveComponents 다 (U4).
	//
	// 조건이 둘이다. 요청이 없으면 허용목록이 어차피 비므로 파일을 아예 안
	// 열고, 워크스페이스가 없으면 출처도 없다 — dir 은 그때 os.TempDir() 로
	// 떨어지는데 거기의 .mcp.json 은 아무나 쓸 수 있다.
	//
	// 읽기 실패를 여기서 안 다룬다. 등급과 문구는 정책이고 정책은 한 자리에 있다.
	var wsMCP map[string]map[string]any
	var wsErr error
	if len(p.MCP) > 0 && w.Local.Workspace != "" {
		wsMCP, wsErr = readWorkspaceMCP(w.Local.Workspace)
	}
	tee, stopTee := w.transcript(step)
	var exitedAt time.Time
	var reporter *exitReporter
	logBytes, h := runHarness(runCtx, ha, bin, Job{
		Params: p, Prompt: prompt,
		IO: IOPaths{Dir: dir, In: in, Out: out},
		// 출처 둘. 합치는 것은 resolveComponents 다 (U4)
		NodeMCP:         w.Local.MCP,
		WorkspaceMCP:    wsMCP,
		WorkspaceMCPErr: wsErr,
		Log:             log,
		Expect:          step.Out, // 훅이 짚을 이름 — 계약이 요구한 산출물
		// 계획 단계면 훅이 모양까지 본다 (ADR-046).
		// expands 단계는 산출물이 정확히 하나임을 계약 검증이 보장한다.
		Plan:       planOutName(step),
		Roles:      step.Roles,
		Stamp:      stamp, // 훅이 볼 기준 시각 — git 이 못 보는 것까지
		Inject:     inject,
		Emit:       func(e Event) { log.Debug("harness event", "kind", e.Kind) },
		Transcript: tee, // 하네스 stdout 을 링과 업로더로 tee
		Session:    session,
		// 하네스 프로세스가 끝난 순간 — 봉투 해석 앞이다. 프로세스가 떴고 임대가
		// 살아 있으면 종료 보고를 시작한다 (business-rules.md 4.1).
		Exited: func(code int, runErr error, at time.Time) {
			outcome, ok := exitOutcome(code, runErr)
			if !ok || runCtx.Err() != nil {
				return
			}
			exitedAt = at.UTC()
			reporter = w.startExitReport(ctx, step, contract.Exited{
				Node: w.Ident.NodeID, Instance: w.Client.Instance, Attempt: step.Attempt,
				Outcome: outcome, ExitedAt: exitedAt,
			})
		},
	})
	// 꼬리를 비우는 것이 UploadLog(선별본)보다 먼저다 - 뒤집으면 봉인 직전의
	// 마지막 문장이 중앙 화면에 안 뜬다.
	stopTee()

	res := Result{Node: w.Ident.NodeID, Harness: &h, Workspace: prep, Environment: session.Environment()}
	completed := h.Reason.Completed()
	harnessFailed := "harness: " + string(h.Reason) + " " + h.Message
	if runCtx.Err() != nil && ctx.Err() == nil {
		// 임대가 끝났다 — Finalize 와 업로드를 건너뛴다. Run 은 이미 회수되고 있다.
		reporter.Stop()
		w.closeAndUploadLog(ctx, step, session, logBytes, log)
		res.Error = "aborted: lease expired"
		if !completed {
			res.Error = harnessFailed
		}
		log.Warn("aborted: lease expired", "reason", h.Reason)
		w.report(ctx, step, res)
		return
	}

	// 하네스가 뜨지 못했으면 exited_at 이 없다. Finalize 예산은 지금부터 센다 —
	// Finalize 는 돌지만(지목 경로 stat) 종료 보고는 없다.
	from := exitedAt
	if from.IsZero() {
		from = time.Now().UTC()
	} else {
		res.ExitedAt = &exitedAt
	}
	// 완주하지 못했으면 collect 와 diff 를 안 하고 산출물을 안 올린다 — 반쯤 쓴 파일을
	// 믿을 수 없다. 단계 로그는 올린다. 오늘은 Finalize 오류에 곧바로 보고했는데, 이제
	// 명령 단계와 같은 모양으로 적고 계속한다.
	spec := finalizeSpecFor(step, completed, w.Local)
	spec.Workspace, spec.Out, spec.Stamp = dir, out, stamp
	leaseEnded := w.afterExit(runCtx, ctx, step, session, spec, from, logBytes, completed, &res, log)
	reporter.Stop()

	switch {
	case !completed:
		// 크래시는 완주가 아니다
		res.Error = harnessFailed
		log.Warn("harness did not complete", "reason", h.Reason, "msg", h.Message)
	case leaseEnded:
		res.Error = "aborted: lease expired"
		log.Warn("aborted: lease expired")
	default:
		log.Info("agent step finished", "reason", h.Reason, "turns", h.Turns,
			"cost_usd", h.CostUSD, "produced", res.Produced,
			"finalize", res.Finalize, "upload", res.Upload)
	}
	w.report(ctx, step, res)
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

// Rejection 은 거절 한 번이다 (ADR-062). Answer 가 답 전문이고,
// 이유는 그 안에 있다 — 무엇이 이유인지는 읽는 쪽이 정한다.
type Rejection struct {
	At     string          `json:"at,omitempty"`
	Answer json.RawMessage `json:"answer"`
}
