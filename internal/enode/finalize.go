package enode

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/taeels/enode/internal/contract"
)

// 명령이 끝난 뒤의 구간을 닫는 규칙이다 (ADR-075 §8 · §9 · §10 · FR-1 ~ FR-3).
//
// 규칙은 순수 함수로 떼어 둔다. 흐름(claim.go 의 execute · runAgentStep)은 이 함수들을
// 차례로 부를 뿐이고, 무엇을 거두나 · 언제 멈추나 · 칸에 무엇을 적나는 여기서 정한다.
// 그래야 규칙마다 표 시험 한 줄로 덮을 수 있다 — 이 패키지의 커버리지 여유가 얇다.

// contractStep 은 노드의 Step 에서 계약 패키지의 기본값 메서드를 부를 만큼만 옮긴다.
//
// 기본값을 여기서 다시 적지 않는다 — Mediator 와 노드가 같은 상수를 읽어야
// 한쪽만 바뀌는 일이 없다. 종류를 정하는 칸(run 이면 Run, agent 면 Agent)과
// 세 칸만 채운다.
func contractStep(step *Step) contract.Step {
	cs := contract.Step{Effect: step.Effect, Budget: step.Budget, Discover: step.Discover}
	if step.Kind == "agent" {
		cs.Agent = map[string]interface{}{}
	} else {
		cs.Run = step.Run
	}
	return cs
}

// stepBudgets 는 두 예산이다. 계약이 안 적었으면 Finalize 1분 · 업로드 3분.
func stepBudgets(step *Step) (finalize, upload time.Duration) {
	return contractStep(step).Budgets()
}

// budgetsFor 는 Worker 가 쓰는 예산이다. 시험이 budgets 를 바꿔 끼운다.
func (w *Worker) budgetsFor(step *Step) (finalize, upload time.Duration) {
	if w.budgets != nil {
		return w.budgets(step)
	}
	return stepBudgets(step)
}

// finalizeSpecFor 는 effect 표대로 무엇을 거둘지 정한다 (business-rules.md 1절).
//
//	run · build (기본) · read    diff 없음
//	run · edit · agent · edit    오늘 조건 그대로 — 노드에 워크스페이스가 있고 계약이 workspace 를 적었다
//	agent · read                 diff 없음
//	모든 단계                     지목 경로 stat 은 늘 한다 (ADR-037).  명시 훑기는 discover 일 때만
//
// completed 는 agent 하네스가 완주했나다. 명령 단계는 늘 참이다 — 종료 코드가
// 무엇이든 명령은 끝났다. 완주하지 못했으면 collect 와 diff 를 안 한다 — 반쯤
// 쓴 파일을 믿을 수 없다. 명시 훑기는 한다 — 실패한 sandbox 를 조사하는 진단이다.
//
// 경로 · 기준 시각 · 마감은 부르는 쪽이 채운다.
func finalizeSpecFor(step *Step, completed bool, local Local) FinalizeSpec {
	effect := contractStep(step).EffectOrDefault()
	spec := FinalizeSpec{
		Effect:   effect,
		Check:    step.CheckChanged,
		Discover: step.Discover,
	}
	if completed {
		spec.Collect = step.Collect
		spec.Diff = effect == contract.EffectEdit && local.Workspace != "" && len(step.Workspace) > 0
	}
	return spec
}

// ── 종료 보고 (ADR-075 §10.4 · business-rules.md 4절) ────────────────────

// exitOutcome 은 session.Run 이 돌려준 것에서 종료 보고의 outcome 을 만든다.
// ok 가 거짓이면 보내지 않는다 — 프로세스가 뜨지 않았다.
//
//	code >= 0                          exit 와 그 코드.  overlay 에서 signal 로 죽은
//	                                   프로세스는 runc 가 128+n 을 주므로 여기 걸린다
//	code < 0 · signal 로 끝났다 (native)  signal 과 신호 번호
//	그 밖                               안 뜬 프로세스 · 이유를 문자열로만 받은 것
func exitOutcome(code int, runErr error) (contract.Outcome, bool) {
	if code >= 0 {
		return contract.Outcome{Kind: contract.OutcomeExit, Code: &code}, true
	}
	var ee *exec.ExitError
	if errors.As(runErr, &ee) {
		if ws, ok := ee.Sys().(interface {
			Signaled() bool
			Signal() syscall.Signal
		}); ok && ws.Signaled() {
			n := int(ws.Signal())
			return contract.Outcome{Kind: contract.OutcomeSignal, Code: &n}, true
		}
	}
	return contract.Outcome{}, false
}

// exitedStop 은 그 응답에서 멈출지다. 2xx 는 받았고(accepted 가 false 여도 — 재전송이
// 닿았다는 뜻이다) 4xx 는 받고서 거절했다. 옛 Mediator 는 라우트가 없어 404 를 준다.
// 5xx 와 그 밖은 다시 보낸다.
func exitedStop(code int) bool { return code/100 == 2 || code/100 == 4 }

// exitReportLog 는 응답마다 노드 로그의 수준과 문구다 (business-rules.md 4.2).
// code 0 은 전송 실패다.
func exitReportLog(code int) (slog.Level, string) {
	switch {
	case code/100 == 2:
		return slog.LevelDebug, "exit reported"
	case code == 400:
		return slog.LevelError, "exit report was malformed; not retrying"
	case code == 404:
		return slog.LevelInfo, "the mediator does not accept exit reports for this step; not retrying"
	case code/100 == 4:
		return slog.LevelWarn, "exit report rejected; not retrying"
	}
	return slog.LevelDebug, "exit report failed; retrying"
}

// exitBackoff 는 n 번째 재전송 앞의 간격이다 — 1초에서 두 배씩, 최대 10초.
func exitBackoff(n int) time.Duration {
	d := time.Second << min(n, 4)
	return min(d, 10*time.Second)
}

// exitReporter 는 종료 보고 한 건을 goroutine 에서 보낸다. Finalize 를 막지 않는다 —
// 보고가 유실돼도 result 가 같은 사실을 싣는다.
type exitReporter struct {
	cancel context.CancelFunc
	done   chan struct{}
	once   sync.Once
}

// startExitReport 는 보고를 시작한다. Client 에 instance 가 없으면 보내지 않고 nil 을
// 돌려준다 — Mediator 가 400 으로 돌려줄 것을 알고 보내지 않는다. nil 의 Stop 은 할 일이 없다.
func (w *Worker) startExitReport(ctx context.Context, step *Step, e contract.Exited) *exitReporter {
	if w.Client == nil || w.Client.Instance == "" {
		w.Log.Debug("no instance id; exit report not sent", "step", step.StepID)
		return nil
	}
	wait := w.exitWait
	if wait == nil {
		wait = exitBackoff
	}
	ctx, cancel := context.WithCancel(ctx)
	r := &exitReporter{cancel: cancel, done: make(chan struct{})}
	go func() {
		defer close(r.done)
		for n := 0; ; n++ {
			stop, err := w.Client.Exited(ctx, step.RunID, step.Seq, e)
			if ctx.Err() != nil {
				return // Stop 을 받았거나 Worker 가 끝났다
			}
			code := 0
			var rej *exitRejected
			switch {
			case err == nil:
				code = 200
			case errors.As(err, &rej):
				code = rej.Code
			}
			level, msg := exitReportLog(code)
			w.Log.Log(ctx, level, msg, "step", step.StepID, "err", err)
			if stop {
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(wait(n)):
			}
		}
	}()
	return r
}

// Stop 은 result 를 보내기 직전에 부른다. 도는 요청과 재전송을 그만두고 goroutine 이
// 끝나기를 기다린다 — result 가 같은 사실을 싣는다.
func (r *exitReporter) Stop() {
	if r == nil {
		return
	}
	r.once.Do(r.cancel)
	<-r.done
}

// ── 판정 칸 · 진단 · 로그 끝 (business-rules.md 3.2 · 5.3 · 6절 · 7절) ─────

// settleIn 은 finalize · upload · reason 칸과 error 를 정하는 데 드는 사실이다.
type settleIn struct {
	finalizeErr    error          // session.Finalize 가 돌려준 것
	closeErr       error          // session.Close 가 돌려준 것
	closedLate     bool           // 닫기가 Finalize 예산의 마감 뒤에 끝났다 (임대는 살아 있다)
	upload         contract.Stage // Worker.upload 가 돌려준 것
	leaseEnded     bool           // runCtx 가 임대로 끝났다
	finalizeBudget time.Duration
	uploadBudget   time.Duration
}

// settle 은 finalize · upload · reason 칸과 error 문구를 정한다 (business-rules.md 3.2).
//
// 예산을 넘긴 것만 판정을 바꾼다 — error 가 차면 Mediator 가 완주가 아닌 것으로 받는다.
// 닫기는 Finalize 예산 안이다 (trash 유닛 · business-rules.md 2절). Finalize 가 제시간에
// 끝났어도 닫기가 마감을 넘겼으면 같은 timeout 이다. 문구는 일어난 순서대로 "; " 로 잇는다. 명령이 완주하지 않았다는 문구(임대 만료 ·
// 실행 실패 · 하네스 미완주)는 부르는 쪽이 이 위에 덮어쓴다 — 그 사실을 먼저 적는다.
func settle(in settleIn) (finalize, upload contract.Stage, reason, errText string) {
	var errs []string
	finalize = contract.StageOK
	switch {
	case (errors.Is(in.finalizeErr, context.DeadlineExceeded) || (in.finalizeErr == nil && in.closedLate)) && !in.leaseEnded:
		finalize = contract.StageTimeout
		errs = append(errs, fmt.Sprintf("finalize budget of %s exceeded", in.finalizeBudget))
	case in.finalizeErr != nil:
		finalize = contract.StageError
		errs = append(errs, "runtime finalize: "+in.finalizeErr.Error())
	}
	if in.closeErr != nil {
		if finalize == contract.StageOK {
			finalize = contract.StageError
		}
		errs = append(errs, "runtime cleanup: "+in.closeErr.Error())
	}
	upload = in.upload
	if upload == contract.StageTimeout {
		if in.leaseEnded {
			upload = contract.StageError
		} else {
			errs = append(errs, fmt.Sprintf("upload budget of %s exceeded", in.uploadBudget))
		}
	}
	switch {
	case finalize == contract.StageTimeout:
		reason = contract.ReasonFinalizeTimeout
	case upload == contract.StageTimeout:
		reason = contract.ReasonUploadTimeout
	}
	return finalize, upload, reason, strings.Join(errs, "; ")
}

// diagnosticsFor 는 result 의 진단 칸이다 (business-rules.md 6.1). 판정 재료가 아니다.
//
// missing 은 $OUT 을 직접 읽어 채운다 — Finalize 가 결과를 못 돌려줬어도 채운다.
// .part 로 끝나는 반쪽은 그 이름이 아니므로 없는 것이다.
func diagnosticsFor(step *Step, out string, effect contract.Effect, fin FinalizeResult) *contract.Diagnostics {
	d := &contract.Diagnostics{Effect: effect, Changes: contract.ChangesNotMeasured}
	have := map[string]bool{}
	for _, n := range harvest(out) {
		have[n] = true
	}
	for _, n := range step.Out {
		if !have[n] {
			d.Missing = append(d.Missing, n)
		}
	}
	for _, n := range fin.Notes {
		d.Collect = append(d.Collect, contract.CollectNote{Name: n.Name, Why: n.Why})
	}
	if disc := fin.Discovery; disc != nil && disc.Skipped == "" {
		d.Changes = contract.ChangesMeasured
		if disc.Limit != "" {
			d.Changes = contract.ChangesPartial
			d.DiscoveryLimit = disc.Limit
		}
		for _, p := range disc.Paths {
			d.Discovered = append(d.Discovered, p.Path)
		}
	}
	return d
}

// 바뀐 기본값의 안내다 (완료 조건 3 ④ ⑤ · business-rules.md 7절). 기한을 두지 않는다.
const (
	noticeRunDefault = "this run step used the default effect build, so it returns no workspace.diff " +
		"and no list of changed files. Write \"effect\": \"edit\" if the command changes source files, " +
		"or \"discover\": true to list what changed."
	noticeAgentChanged = "agent steps no longer return workspace.changed. Write \"discover\": true to list what changed."
)

// logTailList 는 단계 로그 끝에 싣는 목록의 줄 수다. 나머지는 diagnostics 칸에 있다.
const logTailList = 20

// logTail 은 단계 로그(Record 의 logs/ 에 들어가는 선별본) 끝에 붙는 줄이다
// (business-rules.md 6.3). 사람이 읽는 자리다 — 계약 작성자가 result 칸 대신 여기를 본다.
//
// 「바뀐 파일이 없다」를 쓰지 않는다. 확인하지 않은 것을 안 바뀌었다고 말하지 않는다 —
// 훑기를 안 켠 단계는 목록 줄 자체가 없다. 업로드의 결과는 여기 없다 — 이 로그를 먼저
// 올리므로 그 뒤의 일을 담을 수 없다.
func logTail(step *Step, diag *contract.Diagnostics, fin FinalizeResult, finalizeTimeout bool, finalizeBudget time.Duration) string {
	var b strings.Builder
	line := func(format string, args ...any) {
		b.WriteString("enode: ")
		fmt.Fprintf(&b, format, args...)
		b.WriteByte('\n')
	}
	if len(diag.Missing) > 0 {
		line("required by the contract but missing from $OUT: %s", strings.Join(diag.Missing, ", "))
	}
	for _, n := range diag.Collect {
		line("collect could not gather %s: %s", n.Name, n.Why)
	}
	if fin.DiffError != "" {
		line("workspace.diff was not produced: %s", fin.DiffError)
	}
	if disc := fin.Discovery; disc != nil {
		switch {
		case disc.Skipped != "":
			line("discover was requested but %s; nothing was listed", disc.Skipped)
		default:
			head := fmt.Sprintf("discover listed %d files created or modified by this step", disc.Total)
			if disc.Limit != "" {
				head += fmt.Sprintf(" before it stopped at the %s limit; the list is partial", disc.Limit)
			}
			if disc.Deleted > 0 {
				head += fmt.Sprintf(", and %d deletions", disc.Deleted)
			}
			line("%s", head)
			for i, p := range disc.Paths {
				if i >= logTailList {
					break
				}
				line("  %s  %s", p.Path, human(p.Size))
			}
		}
	}
	if finalizeTimeout {
		line("finalize budget of %s exceeded", finalizeBudget)
	}
	switch {
	case step.Kind == "agent" && !step.Discover:
		line("%s", noticeAgentChanged)
	case step.Kind != "agent" && step.Effect == "" && !step.Discover:
		line("%s", noticeRunDefault)
	}
	return b.String()
}

// withTail 은 단계 로그 끝에 줄을 붙인다. 로그가 줄바꿈 없이 끝났으면 한 줄을 띄운다.
func withTail(logBody []byte, tail string) []byte {
	out := append([]byte(nil), logBody...)
	if len(out) > 0 && out[len(out)-1] != '\n' {
		out = append(out, '\n')
	}
	return append(out, tail...)
}

// logFinalize 는 결과 확정을 노드 로그에 남긴다. finalized 한 줄은 조각 1 (걷지 않는다)의
// 스크립트가 읽는다 — checked 가 지목 경로 stat 수이고 visited 가 훑기의 방문 수다.
func logFinalize(log *slog.Logger, spec FinalizeSpec, fin FinalizeResult, diag *contract.Diagnostics, took time.Duration) {
	visited, limit := 0, ""
	if fin.Discovery != nil {
		visited, limit = fin.Discovery.Visited, fin.Discovery.Limit
	}
	log.Info("finalized", "effect", spec.Effect, "checked", len(spec.Check), "discover", spec.Discover,
		"visited", visited, "limit", limit, "took", took.Round(time.Millisecond))
	if len(diag.Missing) > 0 {
		log.Warn("required outputs are missing from $OUT", "missing", diag.Missing)
	}
	for _, n := range fin.Notes {
		log.Warn("collect failed", "name", n.Name, "why", n.Why)
	}
	switch {
	case fin.DiffError != "":
		log.Warn("cannot collect workspace diff", "err", fin.DiffError)
	case fin.DiffBytes > 0:
		log.Info("workspace diff", "bytes", fin.DiffBytes)
	}
	if len(fin.Collected) > 0 {
		log.Info("collected", "names", fin.Collected)
	}
}
