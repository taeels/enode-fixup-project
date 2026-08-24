// Command iapadapter 는 It's a Plan 과 enode 함대를 잇는 어댑터다.
//
// ★ 바깥을 아는 부품은 이것 하나뿐이다 ★ (ADR-040 §2).
// Mediator 는 It's a Plan 을 모르고 (ADR-002 의 범위를 안 넓힌다),
// 계약도 이슈 칸 이름을 모른다. 트래커가 하나 더 붙으면 어댑터가 하나 더 는다.
//
// 흐름은 docs/itsaplan-adapter-example.md §1.2 그대로다:
//
//	① 사람이 이슈의 delegate 를 이 에이전트로 바꾼다      ★ 우리 밖 ★
//	② 어댑터가 당긴다        POST /agent-runs/claim
//	③ 어댑터가 오케스트레이터를 띄운다   enode --once --ready-file, labels{issue}
//	④ 오케스트레이터가 스스로 함대에 붙는다               ★ 오늘 그대로 ★
//	⑤ 어댑터가 광고를 기다린다
//	⑥ 어댑터가 Run 을 낸다   POST /v1/runs  (고정 템플릿 · 추론 0)
//	⑦ 매칭이 그 오케스트레이터를 고른다                   ★ 오늘 그대로 ★
//	⑧ 오케스트레이터가 계획을 짓고 함대가 실행한다
//	⑨ 되묻기가 오면 어댑터가 코멘트로 옮긴다              ★ 손을 뗀다 ★
//	⑩ Run 이 끝나면 코멘트 · 칸 이동 · result
//	⑪ 오케스트레이터가 --once 로 죽고 광고가 만료된다
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

func main() {
	confPath := flag.String("config", "adapter.yaml", "어댑터 설정 파일")
	once := flag.Bool("once", false, "일감 하나를 처리하고 끝낸다 (시험용)")
	verbose := flag.Bool("v", false, "자세히 적는다")
	flag.Parse()

	lvl := slog.LevelInfo
	if *verbose {
		lvl = slog.LevelDebug
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl}))

	cfg, err := LoadConfig(*confPath)
	if err != nil {
		log.Error("설정을 못 읽는다", "path", *confPath, "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	a := &Adapter{
		cfg: cfg,
		iap: NewIAP(cfg.ItsAPlan.BaseURL, cfg.ItsAPlan.APIKey),
		med: NewMediator(cfg.Mediator.URL, cfg.Mediator.Token, cfg.Mediator.Principal),
		log: log,
	}
	if err := a.loadColumns(ctx); err != nil {
		log.Warn("칸 목록을 못 읽었다; 상태 전이를 건너뛴다", "err", err)
	}

	log.Info("어댑터를 시작한다",
		"itsaplan", cfg.ItsAPlan.BaseURL,
		"mediator", cfg.Mediator.URL,
		"project", cfg.ItsAPlan.ProjectKey,
		"poll", cfg.Poll())

	a.Run(ctx, *once)
	a.wg.Wait()
	log.Info("어댑터가 멈췄다")
}

type Adapter struct {
	cfg     *Config
	iap     *IAP
	med     *Mediator
	log     *slog.Logger
	columns map[string]int
	wg      sync.WaitGroup
}

func (a *Adapter) loadColumns(ctx context.Context) error {
	if a.cfg.ItsAPlan.ProjectKey == "" {
		return fmt.Errorf("itsaplan.project_key 가 비어 있다")
	}
	cols, err := a.iap.Columns(ctx, a.cfg.ItsAPlan.ProjectKey)
	if err != nil {
		return err
	}
	a.columns = map[string]int{}
	for _, c := range cols {
		a.columns[c.Name] = c.ID
	}
	return nil
}

// Run 은 큐를 드레인한다.
//
// ★ 이슈마다 고루틴 하나다 ★ — 이슈 둘이 동시에 오면 오케스트레이터도 둘이고,
// 설정 경로가 다르므로 node_id 도 다르다 (ADR-015). I1 이 안 부딪힌다.
func (a *Adapter) Run(ctx context.Context, once bool) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		rr, err := a.iap.Claim(ctx)
		if err != nil {
			a.log.Warn("claim 이 실패했다", "err", err)
			if !sleep(ctx, a.cfg.Poll()) {
				return
			}
			continue
		}
		if rr == nil {
			if once {
				a.log.Info("큐가 비었다 (--once)")
				return
			}
			if !sleep(ctx, a.cfg.Poll()) {
				return
			}
			continue
		}

		a.log.Info("일감을 당겼다",
			"agent_run", rr.ID, "trigger", rr.Trigger,
			"issue", rr.IssueIdentifier, "attempts", rr.Attempts)

		a.wg.Add(1)
		go func(rr *RunnerRun) {
			defer a.wg.Done()
			a.handle(ctx, rr)
		}(rr)

		if once {
			return
		}
	}
}

// handle 은 일감 하나를 끝까지 본다.
func (a *Adapter) handle(ctx context.Context, rr *RunnerRun) {
	defer func() {
		if r := recover(); r != nil {
			a.log.Error("일감 처리 중 죽었다", "agent_run", rr.ID, "panic", r)
			_ = a.iap.Result(ctx, rr.ID, "failed", "", fmt.Sprintf("어댑터 내부 오류: %v", r))
		}
	}()

	if rr.IssueID == nil || rr.IssueIdentifier == "" {
		a.log.Warn("이슈가 없는 일감은 다루지 않는다", "agent_run", rr.ID)
		_ = a.iap.Result(ctx, rr.ID, "failed", "",
			"이 어댑터는 이슈에 붙은 일감만 다룬다 (issueId 가 비어 있다)")
		return
	}

	// 하트비트를 끝까지 돌린다 — ★ 리스가 만료되면 같은 일감이 다시 나온다 ★.
	hbCtx, hbStop := context.WithCancel(ctx)
	defer hbStop()
	go a.heartbeat(hbCtx, rr.ID)

	if rr.Trigger == "mention" {
		a.handleAnswer(ctx, rr)
		return
	}
	a.handleNewWork(ctx, rr)
}

func (a *Adapter) heartbeat(ctx context.Context, runID int) {
	t := time.NewTicker(a.cfg.Heartbeat())
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := a.iap.Heartbeat(ctx, runID); err != nil {
				a.log.Debug("하트비트가 실패했다", "agent_run", runID, "err", err)
			}
		}
	}
}

// ───────────────────────────────────────────────────────────────────────────
// 새 일감 — 위임 · 커스텀 필드 · 이슈 전체를 맡는 멘션
// ───────────────────────────────────────────────────────────────────────────

func (a *Adapter) handleNewWork(ctx context.Context, rr *RunnerRun) {
	issueKey := rr.IssueIdentifier
	log := a.log.With("issue", issueKey, "agent_run", rr.ID)

	// ★ run_id 를 이슈에서 유도한다 ★ — 어댑터가 죽어도 답이 왔을 때 같은
	// 이름을 다시 계산할 수 있다. ADR-040 §3.5 가 어댑터 DB 를 기각한 이유가
	// 그것이고, 유도 가능하면 표 자체가 필요 없다.
	runID, err := a.nextRunID(ctx, issueKey)
	if err != nil {
		log.Error("run_id 를 못 정했다", "err", err)
		_ = a.iap.Result(ctx, rr.ID, "failed", "", "run_id 를 못 정했다: "+err.Error())
		return
	}

	issue, err := a.iap.Issue(ctx, *rr.IssueID)
	if err != nil {
		log.Warn("이슈를 못 읽었다; 프롬프트만 쓴다", "err", err)
	}

	a.move(ctx, *rr.IssueID, a.cfg.Transition.OnStart, log)

	orch, err := StartOrchestrator(ctx, a.cfg, issueKey, log)
	if err != nil {
		log.Error("오케스트레이터를 못 띄웠다", "err", err)
		a.failOut(ctx, rr, "오케스트레이터를 못 띄웠다: "+err.Error())
		return
	}
	// ★ 되묻기로 손을 뗄 때는 안 죽인다 ★ — 답이 오면 그 자리에서 이어야 한다.
	// ★ 그래도 거두기는 한다 ★ — 안 그러면 그 판마다 좀비가 하나씩 쌓인다.
	keepAlive := false
	defer func() {
		if keepAlive {
			orch.Detach()
			return
		}
		orch.Stop()
	}()

	if err := orch.WaitAdvertised(ctx, a.med, a.cfg.ReadyWait()); err != nil {
		log.Error("광고를 못 봤다", "err", err)
		a.failOut(ctx, rr, "오케스트레이터가 함대에 안 나타났다: "+err.Error())
		return
	}

	body, err := BuildContract(a.cfg, runID, issueKey, issue, rr)
	if err != nil {
		log.Error("계약을 못 지었다", "err", err)
		a.failOut(ctx, rr, "계약을 못 지었다: "+err.Error())
		return
	}
	if err := a.med.SubmitRun(ctx, body); err != nil {
		log.Error("Run 제출이 실패했다", "run", runID, "err", err)
		a.failOut(ctx, rr, "Run 제출이 실패했다: "+err.Error())
		return
	}
	log.Info("Run 을 냈다", "run", runID)

	keepAlive = a.follow(ctx, rr, runID, issueKey, log)
}

// follow 는 Run 을 끝까지 따라간다.
// 되묻기로 손을 떼면 true 를 돌려준다 — 그러면 오케스트레이터를 안 죽인다.
func (a *Adapter) follow(ctx context.Context, rr *RunnerRun, runID, issueKey string, log *slog.Logger) bool {
	tick := time.NewTicker(3 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return true // 어댑터가 내려간다 — Run 은 살아 있으므로 안 죽인다
		case <-tick.C:
		}

		run, err := a.med.GetRun(ctx, runID)
		if err != nil {
			log.Debug("Run 조회가 실패했다", "err", err)
			continue
		}

		if run.Terminal() {
			a.finish(ctx, rr, run, log)
			return false
		}

		// ★ 되묻기 ★ — 인박스가 정본이다 (ADR-032 §4).
		ask, err := a.med.AskFor(ctx, runID)
		if err != nil || ask == nil {
			continue
		}
		a.handOff(ctx, rr, run, ask, log)
		return true
	}
}

// handOff 는 되묻기를 이슈로 내보내고 ★ 손을 뗀다 ★.
//
// ★ 실행을 붙잡고 기다리면 실패로 끝난다 ★ (integration §6.2 의 함정) —
// It's a Plan 의 리스는 300초이고 사람은 그보다 오래 걸린다. 그래서 result 는
// success 다: ★ 실패가 아니라 손을 뗀 것 ★ (ADR-040 §4).
func (a *Adapter) handOff(ctx context.Context, rr *RunnerRun, run *RunView, ask *AskView, log *slog.Logger) {
	commentID, err := a.iap.Comment(ctx, *rr.IssueID, RenderQuestion(ask, run.RunID), 0)
	if err != nil {
		log.Error("질문 코멘트를 못 썼다", "err", err)
		_ = a.iap.Result(ctx, rr.ID, "failed", "", "질문을 이슈로 못 옮겼다: "+err.Error())
		return
	}
	log.Info("질문을 코멘트로 옮겼다", "comment", commentID, "step", ask.Step, "seq", ask.Seq)

	// ★ 밖에 남긴 것도 산출물이다 ★ (ADR-040 §3.3).
	//
	// 어댑터의 표가 아니라 blob 으로 낸다 — 그러면 원장에 저절로 나타나고
	// 봉인에 저절로 남는다. ★ 새 부품이 0 개다 ★. 밑줄 예약이 이름 충돌을
	// 막으므로 계약이 이 이름을 못 쓴다 (ADR-023 §6.3.1).
	ob, _ := json.Marshal(map[string]any{
		"system":  "itsaplan",
		"issue":   *rr.IssueID,
		"comment": commentID,
		"step":    ask.Step,
		"seq":     ask.Seq,
		"at":      time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err := a.med.PutBlob(ctx, run.RunID, ask.Seq, "_outbound", ob); err != nil {
		// ★ 막지 않는다 ★ — 질문은 이미 나갔다. 다만 답을 맞출 재료가 약해지므로
		// 크게 적는다.
		log.Error("_outbound 를 못 남겼다; 답 매칭이 약해진다", "err", err)
	}

	a.move(ctx, *rr.IssueID, a.cfg.Transition.OnAsk, log)

	out := fmt.Sprintf("사람의 답을 기다린다 — 코멘트 #%d · run %s", commentID, run.RunID)
	if err := a.iap.Result(ctx, rr.ID, "success", out, ""); err != nil {
		log.Warn("result 보고가 실패했다", "err", err)
	}
}

// finish 는 끝난 Run 을 밖으로 넘긴다 — ★ 셋으로 갈린다 ★ (ADR-040 §1).
func (a *Adapter) finish(ctx context.Context, rr *RunnerRun, run *RunView, log *slog.Logger) {
	// ② 사람이 읽는 것
	summary := summaryOf(ctx, a.med, run.RunID)
	if _, err := a.iap.Comment(ctx, *rr.IssueID, RenderResult(ctx, a.med, run, summary), 0); err != nil {
		log.Error("결과 코멘트를 못 썼다", "err", err)
	}

	// ③ 보드가 읽는 것
	col := a.cfg.Transition.OnFailure
	if run.State == "SUCCEEDED" {
		col = a.cfg.Transition.OnSuccess
	}
	a.move(ctx, *rr.IssueID, col, log)

	// ① 기계가 읽는 것 — ★ 우리 완주 판정이 그대로 사상된다 ★
	status := "failed"
	errMsg := ""
	if run.State == "SUCCEEDED" {
		status = "success"
	} else {
		errMsg = "Run 이 " + run.State + " 로 끝났다"
	}
	if err := a.iap.Result(ctx, rr.ID, status, oneLine(run, run.RunID), errMsg); err != nil {
		log.Warn("result 보고가 실패했다", "err", err)
	}
	log.Info("일감을 닫았다", "run", run.RunID, "state", run.State)
}

func (a *Adapter) failOut(ctx context.Context, rr *RunnerRun, msg string) {
	if rr.IssueID != nil {
		_, _ = a.iap.Comment(ctx, *rr.IssueID,
			"### 시작하지 못했습니다\n\n```\n"+msg+"\n```", 0)
		a.move(ctx, *rr.IssueID, a.cfg.Transition.OnFailure, a.log)
	}
	_ = a.iap.Result(ctx, rr.ID, "failed", "", msg)
}

// move 는 칸을 옮긴다. 이름이 비었거나 모르면 안 옮긴다.
//
// ★ 실패해도 되돌릴 것이 없다 ★ (ADR-040 §6) — 판정은 이미 끝났다.
func (a *Adapter) move(ctx context.Context, issueID int, columnName string, log *slog.Logger) {
	if columnName == "" || a.columns == nil {
		return
	}
	id, ok := a.columns[columnName]
	if !ok {
		log.Warn("모르는 칸 이름이다; 안 옮긴다", "column", columnName)
		return
	}
	if err := a.iap.MoveIssue(ctx, issueID, id); err != nil {
		log.Warn("칸을 못 옮겼다 (부분 성공)", "column", columnName, "err", err)
		return
	}
	log.Info("칸을 옮겼다", "column", columnName)
}

// ───────────────────────────────────────────────────────────────────────────
// 답 — 사람이 질문 코멘트에 답글을 달면 ★ 새 mention run ★ 으로 온다
// ───────────────────────────────────────────────────────────────────────────

// ★ \b 를 쓰면 안 된다 ★ — Go 의 RE2 에서 \b 는 ASCII 낱말 경계라
// "승인" 이 한 번도 안 걸린다. 시험이 그것을 잡았다.
var verdictRe = regexp.MustCompile(
	`(?i)(?:^|[^0-9A-Za-z])(ok|again|approve|reject|승인|거절)(?:[^0-9A-Za-z]|$)`)

// handleAnswer 는 답글로 온 일감을 되묻기의 답으로 옮긴다.
//
// ★ 상관관계 식별자가 구조에 없다 ★ (실측 ③) — 그래서 (이슈, 우리가 남긴 질문
// 코멘트 번호)로 맞춘다. 그 번호는 _outbound 산출물에 있고, 원장이 scope:"work"
// 이므로 ★ 이전 Run 이 낸 것까지 보인다 ★ (ADR-040 §3.4).
func (a *Adapter) handleAnswer(ctx context.Context, rr *RunnerRun) {
	issueKey := rr.IssueIdentifier
	log := a.log.With("issue", issueKey, "agent_run", rr.ID)

	runID, ask, err := a.findOpenAsk(ctx, issueKey)
	if err != nil || ask == nil {
		// ★ 열린 질문이 없다고 할 일이 없는 것은 아니다 ★ — 어댑터가 죽거나
		// 재시작하면 이미 끝난 Run 의 결과가 아직 안 나갔을 수 있다.
		// Mediator 와 노드는 어댑터를 모르므로 Run 은 어댑터 없이도 끝난다
		// (adapter-example §2). 그래서 ★ 끝난 것을 다시 찾아 넘긴다 ★.
		if a.reportUnreported(ctx, rr, issueKey, log) {
			return
		}
		log.Info("이 이슈에 열린 질문도 안 넘긴 결과도 없다", "err", err)
		_ = a.iap.Result(ctx, rr.ID, "success",
			"열린 질문이 없어 답으로 처리하지 않았다", "")
		return
	}

	verdict, note := parseAnswer(rr.Prompt)
	if verdict == "" {
		log.Info("답에서 판정을 못 읽었다", "prompt", head(rr.Prompt, 200))
		_, _ = a.iap.Comment(ctx, *rr.IssueID,
			"답을 읽지 못했습니다. `ok` 또는 `again` 으로 시작하는 답글을 달아 주세요.", 0)
		_ = a.iap.Result(ctx, rr.ID, "success", "답에서 판정을 못 읽었다", "")
		return
	}

	body, _ := json.Marshal(map[string]any{"verdict": verdict, "note": note})
	if err := a.med.Answer(ctx, runID, ask.Seq, body); err != nil {
		log.Error("답을 못 넣었다", "run", runID, "seq", ask.Seq, "err", err)
		_, _ = a.iap.Comment(ctx, *rr.IssueID,
			"답을 계약에 넣지 못했습니다:\n\n```\n"+err.Error()+"\n```", 0)
		_ = a.iap.Result(ctx, rr.ID, "failed", "", err.Error())
		return
	}
	log.Info("답을 넣었다", "run", runID, "seq", ask.Seq, "verdict", verdict)

	a.move(ctx, *rr.IssueID, a.cfg.Transition.OnStart, log)
	_ = a.follow(ctx, rr, runID, issueKey, log)
}

// reportUnreported 는 끝났는데 아직 밖으로 안 넘긴 Run 을 찾아 넘긴다.
//
// ★ 중복을 어떻게 막나 ★ — 어댑터가 표를 들면 ADR-040 §3.5 가 기각한 것으로
// 되돌아간다. 그래서 ★ 이슈 자신에게 물어본다 ★: 결과 코멘트의 꼬리표에
// run_id 가 들어 있으므로, 그것이 피드에 있으면 이미 넘긴 것이다.
func (a *Adapter) reportUnreported(ctx context.Context, rr *RunnerRun, issueKey string, log *slog.Logger) bool {
	runID, run := a.latestRun(ctx, issueKey)
	if run == nil || !run.Terminal() {
		return false
	}
	said, err := a.iap.AlreadySaid(ctx, *rr.IssueID, ResultMarker(runID))
	if err != nil {
		log.Warn("피드를 못 읽어 중복 여부를 모른다; 안 넘긴다", "err", err)
		return false
	}
	if said {
		return false
	}
	log.Info("끝났는데 안 넘긴 Run 을 찾았다; 지금 넘긴다", "run", runID, "state", run.State)
	a.finish(ctx, rr, run, log)
	return true
}

// latestRun 은 이 이슈의 가장 최근 세대 Run 을 찾는다.
func (a *Adapter) latestRun(ctx context.Context, issueKey string) (string, *RunView) {
	prefix := runIDPrefix(issueKey)
	var lastID string
	var last *RunView
	for n := 1; n <= 50; n++ {
		id := prefix + strconv.Itoa(n)
		run, err := a.med.GetRun(ctx, id)
		if err != nil {
			break
		}
		lastID, last = id, run
	}
	return lastID, last
}

// findOpenAsk 는 이 이슈의 열린 되묻기를 찾는다.
//
// ★ 어댑터가 표를 안 든다 ★ — 인박스를 훑고, run_id 가 이 이슈에서 유도된
// 이름인지로 가른다. 어댑터가 재시작해도 그대로 성립한다.
func (a *Adapter) findOpenAsk(ctx context.Context, issueKey string) (string, *AskView, error) {
	asks, err := a.med.Asks(ctx)
	if err != nil {
		return "", nil, err
	}
	prefix := runIDPrefix(issueKey)
	var best *AskView
	var bestRun string
	for i := range asks {
		if !strings.HasPrefix(asks[i].RunID, prefix) {
			continue
		}
		// 세대가 여럿이면 ★ 가장 최근 것 ★ 을 고른다.
		if bestRun == "" || asks[i].RunID > bestRun {
			best = &asks[i]
			bestRun = asks[i].RunID
		}
	}
	if best == nil {
		return "", nil, nil
	}
	return bestRun, best, nil
}

// parseAnswer 는 답글 본문에서 판정과 이유를 뽑는다.
//
// ★ It's a Plan 에 폼이 없다 ★ (ADR-040 §4) — 스키마 강제는 우리 쪽에 남으므로
// 여기서 자연어를 스키마에 맞춘다. 못 맞추면 ★ 질문은 열린 채 남는다 ★.
func parseAnswer(prompt string) (verdict, note string) {
	// 답 run 의 프롬프트는 스레드를 통째로 싣는다 (실측 §10.5.2). 우리가 찾는
	// 것은 ★ 우리를 부른 그 코멘트 ★ 이므로 그 표지 뒤를 본다.
	body := prompt
	for _, marker := range []string{
		"The comment that mentioned you:",
		"The comment that mentioned you",
		"Someone answered your comment",
	} {
		if i := strings.Index(prompt, marker); i >= 0 {
			body = prompt[i+len(marker):]
			break
		}
	}
	loc := verdictRe.FindStringSubmatchIndex(body)
	if loc == nil {
		return "", ""
	}
	word := strings.ToLower(body[loc[2]:loc[3]])
	switch word {
	case "ok", "approve", "승인":
		verdict = "ok"
	default:
		verdict = "again"
	}
	// 판정어 뒤가 이유다. 앞에 붙은 구두점은 떼어낸다.
	note = strings.TrimSpace(body[loc[3]:])
	note = strings.TrimSpace(strings.TrimLeft(note, ".,:;-—·"))
	if len(note) > 4000 {
		note = note[:4000]
	}
	return verdict, note
}

// ───────────────────────────────────────────────────────────────────────────

func runIDPrefix(issueKey string) string { return "itsaplan-" + issueKey + "-" }

// nextRunID 는 이 이슈의 다음 세대 이름을 고른다.
//
// ★ 표가 아니라 탐침이다 ★ — itsaplan-EP-2-1 부터 올려 보며 없는 첫 이름을
// 쓴다. 어댑터가 죽어도, 다시 떠도, 같은 규칙이 같은 답을 준다.
func (a *Adapter) nextRunID(ctx context.Context, issueKey string) (string, error) {
	prefix := runIDPrefix(issueKey)
	for n := 1; n <= 50; n++ {
		id := prefix + strconv.Itoa(n)
		run, err := a.med.GetRun(ctx, id)
		if err == ErrNoRun {
			return id, nil
		}
		if err != nil {
			return "", err
		}
		if !run.Terminal() {
			return "", fmt.Errorf("%s 가 아직 안 끝났다 (%s) — 이슈 하나에 열린 Run 은 하나다",
				id, run.State)
		}
	}
	return "", fmt.Errorf("%s 의 세대가 50 을 넘었다", issueKey)
}

func sleep(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

func head(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func parseTime(s string) (time.Time, error) { return time.Parse(time.RFC3339Nano, s) }
