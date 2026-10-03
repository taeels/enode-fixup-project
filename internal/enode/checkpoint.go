package enode

// 실패한 단계의 보존을 Worker 에 잇는다 (checkpoint 유닛 · ADR-076 · FR-10).
//
// 규칙은 internal/scratch 에 있다 (받아들임 · 판정 · 조정). 여기는 잇는 일이다 — 세션을 열 때의 예약 · 닫을 때의
// 판정과 확정 · receipt 로 옮겨 담기 · 보고 뒤의 버림과 판정 · 기동. 보존의 성패는 단계의 결과를 바꾸지 않는다.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/taeels/enode/internal/contract"
	execenv "github.com/taeels/enode/internal/environment"
	"github.com/taeels/enode/internal/lower"
	"github.com/taeels/enode/internal/scratch"
)

// diagnostics.checkpoint 의 문장 (business-rules.md 2절). host 경로를 싣지 않는다 — 오류는 errno 의 글만 잇는다.
const (
	noteRuntime         = "this node runs steps without an isolated upper; there is nothing to keep"
	noteCrossFilesystem = "the spool is on another filesystem than the step's upper; the upper is never copied"
	noteBudgetBefore    = "the finalize budget ran out before the upper could be moved into the spool"
	noteReserve         = "reserving a place in the spool failed: "
	noteRefused         = "the spool is not a private directory of this node; nothing is kept"
	noteMove            = "moving the upper into the spool failed: "
	noteAborted         = "the session was aborted before the upper could be kept"
	noteBudgetAfter     = "the finalize budget ran out after the upper was moved; the checkpoint was discarded"
	noteRecord          = "writing the checkpoint record failed: "
)

// CheckpointKeeper 는 노드 하나의 보존이다. nil 이면 오늘과 같다 — 칸도 예약도 없다 (business-rules.md 4절).
// native 노드에도 있다 — Store 없이 정책만 들고, 요구한 단계에 unsupported(runtime) 를 싣는다.
type CheckpointKeeper struct {
	Policy    scratch.Policy
	Store     *scratch.Store // nil 이면 native — 보존할 upper 가 없다
	Support   CaptureSupport
	Node      string
	Runtime   string // 기록의 runtime 칸
	Workspace string // lower 신원과 metadata 의 자리. 비면 신원 없이 적는다
	Env       string // 준비된 실행 환경의 식별자
	Log       *slog.Logger
	Status    *StatusBook
	OnTrash   func()        // trash 에 넣었을 때 — 삭제자의 Kick
	Every     time.Duration // 만료를 거두는 주기. 0 이면 scratch.DefaultEvery
	Now       func() time.Time

	once    sync.Once
	kick    chan struct{}
	running atomic.Bool

	mu        sync.Mutex
	disposals []*scratch.Reservation
	reports   []reportNote
}

type reportNote struct{ id, outcome string }

// checkpointPause 는 시험이 예약 뒤와 확정 앞에 늦음을 끼우는 자리다 (NFR P2 · 쓰기 부하에서 예약이 초 단위로
// 멈춘 측정). 제품에서는 할 일이 없다.
var checkpointPause = func(at string) {}

// checkpointSlot 은 단계의 예약이다 — 세션을 열 때 받고 닫을 때 쓰거나 보고 뒤 버린다 (NFR Design 답 1). 열 때의
// 오류를 함께 든다 — 닫을 때 요구하면 그 errno 로 알린다.
type checkpointSlot struct {
	resv *scratch.Reservation
	err  error
}

// closeFacts 는 닫을 때 판정이 보는 사실이다 (business-rules.md 1 · 2절).
type closeFacts struct {
	step     *Step
	failed   bool // 굽기면 failEnd · 아니면 exit ≠ 0 · signal · 하네스 미완주 · 빠진 산출물 · Finalize 오류
	bake     bool
	deadline time.Time
	finErr   error
	slot     *checkpointSlot
}

// capturePlan 은 닫을 때의 판정이다. decided 가 있으면 옮기지 않고 끝났다. 없으면 keep 으로 Close 에 넘기고
// Finish 가 그 결과로 확정한다.
type capturePlan struct {
	decided  *scratch.Capture
	keep     Keep
	result   KeepResult
	slot     *checkpointSlot
	entry    scratch.Entry
	step     *Step
	deadline time.Time
}

func (k *CheckpointKeeper) clock() time.Time {
	if k.Now != nil {
		return k.Now()
	}
	return time.Now().UTC()
}

func (k *CheckpointKeeper) log() *slog.Logger {
	if k.Log == nil {
		return slog.New(slog.DiscardHandler)
	}
	return k.Log
}

// Reserve 는 세션을 연 뒤의 예약이다 (NFR Design 답 1). 정책이 off 가 아니고 runtime 이 지원하고 spool 을
// 받아들인 노드에서만 한다. 판정이 아니다 — 요구하는지는 닫을 때 정한다.
func (k *CheckpointKeeper) Reserve(step *Step) *checkpointSlot {
	if k == nil || k.Policy.Mode == scratch.ModeOff || !k.Support.Supported || k.Store == nil ||
		k.Store.Refused() != "" {
		return nil
	}
	resv, err := k.Store.Reserve(scratch.Entry{Node: k.Node, Run: step.RunID, Seq: step.Seq, Step: step.Name,
		Attempt: step.Attempt, Runtime: k.Runtime, Format: scratch.FormatOverlayUpper,
		Scope: scratch.ScopeWorkspaceUpper, Guarantee: scratch.GuaranteeInspectOnly})
	checkpointPause("reserved")
	return &checkpointSlot{resv: resv, err: err}
}

// requested 는 규칙 1절의 표다. 굽기 build 는 failEnd 만 — 성공 끝의 upper 는 merge 의 것이다 (FD 답 3).
func (k *CheckpointKeeper) requested(f closeFacts) bool {
	switch {
	case k.Policy.Mode == scratch.ModeOff:
		return false
	case f.bake:
		return f.failed
	case k.Policy.Mode == scratch.ModeAlways:
		return true
	}
	return f.failed
}

// Decide 는 닫기 앞의 판정이다 (business-rules.md 2절 ① ~ ⑤). 판정의 창 안이라 상수 시간의 일만 한다 — stat 둘 ·
// statfs · 요약 읽기 · lower 신원.
func (k *CheckpointKeeper) Decide(f closeFacts) *capturePlan {
	if k == nil {
		return nil
	}
	p := &capturePlan{slot: f.slot, step: f.step, deadline: f.deadline}
	done := func(c scratch.Capture) *capturePlan { p.decided = &c; return p }
	if !k.requested(f) {
		return done(scratch.Capture{State: scratch.StateNotRequested})
	}
	if !k.Support.Supported || k.Store == nil {
		return done(scratch.Capture{State: scratch.StateUnsupported, Reason: scratch.ReasonRuntime, Detail: noteRuntime})
	}
	if k.Store.Refused() == "" {
		if same, err := k.Store.SameFilesystem(); err == nil && !same {
			return done(scratch.Capture{State: scratch.StateUnsupported, Reason: scratch.ReasonCrossFilesystem,
				Detail: noteCrossFilesystem})
		}
	}
	if errors.Is(f.finErr, context.DeadlineExceeded) || !k.clock().Before(f.deadline) {
		return done(scratch.Capture{State: scratch.StateRejected, Reason: scratch.ReasonLeaseBudget, Detail: noteBudgetBefore})
	}
	fs, err := k.Store.Filesystem()
	if c, ok := k.Policy.Admit(scratch.Admission{Free: fs.Free, FreeKnown: err == nil && fs.Known,
		Kept: k.Store.Kept()}); !ok {
		return done(c)
	}
	if k.Store.Refused() != "" {
		return done(scratch.Capture{State: scratch.StateFailed, Reason: scratch.ReasonIO, Detail: noteRefused})
	}
	if f.slot == nil || f.slot.resv == nil {
		var cause error
		if f.slot != nil {
			cause = f.slot.err
		}
		if errors.Is(cause, scratch.ErrRefused) {
			return done(scratch.Capture{State: scratch.StateFailed, Reason: scratch.ReasonIO, Detail: noteRefused})
		}
		return done(scratch.Capture{State: scratch.StateFailed, Reason: scratch.ReasonIO, Detail: noteReserve + errnoText(cause)})
	}
	p.entry = k.identity(f.slot.resv.Entry())
	p.keep = Keep{Upper: f.slot.resv.Upper(), By: f.deadline, Result: &p.result}
	return p
}

// identity 는 옮기기 앞에 정하는 신원이다 (ADR-076 §4 의 3 · business-rules.md 11절). lower 는 읽기만 한다.
func (k *CheckpointKeeper) identity(e scratch.Entry) scratch.Entry {
	e.Environment = k.Env
	if k.Workspace == "" {
		return e
	}
	if root, err := lower.ReadRoot(k.Workspace); err == nil {
		e.Lower = root.Key.String()
	}
	if md, err := lower.ReadMetadata(k.Workspace); err == nil && md != nil {
		e.Head = md.Source.Head
		if md.Source.IR != nil {
			e.IR = *md.Source.IR
		}
	}
	return e
}

// Keep 은 Close 에 넘길 행선지다. 옮기지 않기로 정했으면 fallback 그대로다 (굽기의 대기 자리 · 버림).
func (p *capturePlan) Keep(fallback Keep) Keep {
	if p == nil || p.decided != nil {
		return fallback
	}
	return p.keep
}

// Finish 는 closedAt 뒤의 확정이다 (business-rules.md 2절 ⑤ ~ ⑦). 판정의 창 밖이라 기록 쓰기가 늦어도 단계의
// finalize 판정을 안 바꾼다 (NFR P2). receipt 의 칸과 diagnostics 의 문장을 돌려준다. nil 이면 칸이 없다.
func (k *CheckpointKeeper) Finish(p *capturePlan) (*contract.CheckpointCapture, string) {
	if k == nil || p == nil {
		return nil, ""
	}
	c := p.decided
	if c == nil {
		got := k.finishKeep(p)
		c = &got
	}
	switch c.State {
	case scratch.StateCaptured:
		k.log().Info("checkpoint captured", "id", c.ID, "run", p.step.RunID, "step", p.step.Name, "expires_at", c.ExpiresAt)
	case scratch.StateNotRequested:
	default:
		k.log().Info("checkpoint not captured", "state", c.State, "reason", c.Reason, "detail", c.Detail)
	}
	return &contract.CheckpointCapture{State: c.State, Reason: c.Reason, ID: c.ID, Scope: c.Scope,
		Guarantee: c.Guarantee, Node: c.Node, ExpiresAt: c.ExpiresAt}, c.Detail
}

func (k *CheckpointKeeper) finishKeep(p *capturePlan) scratch.Capture {
	r := p.result
	resv := p.slot.resv
	switch {
	case r.Late:
		return scratch.Capture{State: scratch.StateRejected, Reason: scratch.ReasonLeaseBudget, Detail: noteBudgetBefore}
	case errors.Is(r.Err, errKeepAborted):
		return scratch.Capture{State: scratch.StateFailed, Reason: scratch.ReasonIO, Detail: noteAborted}
	case r.Err != nil:
		return scratch.Capture{State: scratch.StateFailed, Reason: scratch.ReasonIO, Detail: noteMove + errnoText(r.Err)}
	case !r.Moved:
		return scratch.Capture{State: scratch.StateFailed, Reason: scratch.ReasonIO,
			Detail: noteMove + "the runtime did not move it"}
	}
	checkpointPause("commit")
	if !k.clock().Before(p.deadline) {
		_ = k.Store.Abandon(resv)
		return scratch.Capture{State: scratch.StateFailed, Reason: scratch.ReasonLeaseBudget, Detail: noteBudgetAfter}
	}
	e, err := k.Store.Commit(resv, p.entry)
	if err != nil {
		_ = k.Store.Abandon(resv)
		return scratch.Capture{State: scratch.StateFailed, Reason: scratch.ReasonIO, Detail: noteRecord + errnoText(err)}
	}
	return scratch.Capture{State: scratch.StateCaptured, ID: e.ID, Scope: e.Scope, Guarantee: e.Guarantee,
		Node: k.Node, ExpiresAt: e.ExpiresAt}
}

// errnoText 는 오류의 뜻만이다 — syscall.Errno 의 글 (business-rules.md 2절). PathError · LinkError 의 문장은 경로를
// 담으므로 쓰지 않는다 (NFR C2 · C3).
func errnoText(err error) string {
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return errno.Error()
	}
	if err == nil {
		return "no reservation was made"
	}
	return "unexpected error"
}

// Dispose 는 요구하지 않았거나 칸이 없는 끝의 예약을 버린다 — 보고 뒤 (NFR Design 답 1). 판정 고루틴이 돌면 그쪽에
// 넘기고, 안 돌면 (시험 · 판정 고루틴이 없는 조립) 그 자리에서 버린다. 확정했거나 이미 버렸으면 할 일이 없다.
func (k *CheckpointKeeper) Dispose(slot *checkpointSlot) {
	if k == nil || slot == nil || slot.resv == nil || k.Store == nil {
		return
	}
	if !k.running.Load() {
		_ = k.Store.Dispose(slot.resv)
		return
	}
	k.mu.Lock()
	k.disposals = append(k.disposals, slot.resv)
	k.mu.Unlock()
	k.Kick()
}

// Reported 는 보고의 성패를 기록에 적는다 (business-rules.md 8절). captured 였을 때만이다.
func (k *CheckpointKeeper) Reported(c *contract.CheckpointCapture, outcome string) {
	if k == nil || k.Store == nil || c == nil || c.State != contract.CaptureCaptured {
		return
	}
	if !k.running.Load() {
		k.writeReport(reportNote{id: c.ID, outcome: outcome})
		return
	}
	k.mu.Lock()
	k.reports = append(k.reports, reportNote{id: c.ID, outcome: outcome})
	k.mu.Unlock()
	k.Kick()
}

func (k *CheckpointKeeper) writeReport(n reportNote) {
	if err := k.Store.Report(n.id, n.outcome); err != nil {
		k.log().Warn("cannot record the report outcome of a checkpoint", "id", n.id, "err", err)
	}
}

// Kick 은 판정을 깨운다. 막지 않는다 — 이미 깨어 있으면 한 번 더 도는 것으로 합친다 (삭제자와 같다 · NFR Design D4).
func (k *CheckpointKeeper) Kick() {
	if k == nil {
		return
	}
	select {
	case k.kicks() <- struct{}{}:
	default:
	}
}

func (k *CheckpointKeeper) kicks() chan struct{} {
	k.once.Do(func() { k.kick = make(chan struct{}, 1) })
	return k.kick
}

// Run 은 ctx 가 끝날 때까지 돈다 — 조정 (기동 때 한 번) 뒤 판정, 그다음 깸 (보고 뒤 · Every) 마다 판정
// (business-rules.md 6 · 9절). spool 이 없거나 거절했으면 할 일이 없다.
func (k *CheckpointKeeper) Run(ctx context.Context) {
	if k == nil || k.Store == nil || k.Store.Refused() != "" {
		return
	}
	k.running.Store(true)
	defer k.running.Store(false)
	k.reconcile()
	every := k.Every
	if every <= 0 {
		every = scratch.DefaultEvery
	}
	for {
		k.pass(ctx)
		timer := time.NewTimer(every)
		select {
		case <-ctx.Done():
		case <-k.kicks():
		case <-timer.C:
		}
		timer.Stop()
		if ctx.Err() != nil {
			k.drain()
			return
		}
	}
}

func (k *CheckpointKeeper) reconcile() {
	res, err := k.Store.Reconcile()
	if err != nil {
		k.log().Warn("checkpoint reconcile failed", "err", err)
		return
	}
	for _, name := range res.Unknown {
		k.log().Warn("unknown entry left in the spool", "name", name)
	}
	if len(res.Trashed) > 0 && k.OnTrash != nil {
		k.OnTrash()
	}
}

// drain 은 남은 버림과 성패를 마저 한다 — 데몬이 멈출 때.
func (k *CheckpointKeeper) drain() {
	k.mu.Lock()
	disposals, reports := k.disposals, k.reports
	k.disposals, k.reports = nil, nil
	k.mu.Unlock()
	for _, r := range disposals {
		_ = k.Store.Dispose(r)
	}
	for _, n := range reports {
		k.writeReport(n)
	}
}

// pass 는 판정 한 번이다 — 남은 버림 · 보고의 성패 · 보고 뒤 판정 · 상태 파일.
func (k *CheckpointKeeper) pass(ctx context.Context) {
	k.drain()
	res, err := k.Store.Settle(ctx)
	if err != nil {
		k.log().Warn("checkpoint settle failed", "err", err)
	}
	if res.Launch != nil {
		k.log().Warn("checkpoint settle failed", "err", res.Launch)
	}
	for id, err := range res.Failed {
		k.log().Debug("cannot measure a checkpoint; the next settle measures it again", "id", id, "err", err)
	}
	for _, id := range res.Expired {
		k.log().Info("checkpoint expired", "id", id)
	}
	for _, ev := range res.Evicted {
		k.log().Info("checkpoint evicted", "id", ev.ID, "reason", ev.Reason)
	}
	if err == nil {
		k.Status.SetSpool(res.Usage)
	}
	if res.Moved && k.OnTrash != nil {
		k.OnTrash()
	}
}

// ScratchSetup 은 기동이 StartScratch 에 넘기는 것이다.
type ScratchSetup struct {
	Scratch   string // runc-overlay 의 scratch. 비면 native — 작업 폴더도 보존도 없다
	Workspace string // lower 루트 — 보존본의 lower 신원
	Config    string // 설정 파일 경로 — 기동 로그가 바꾸는 자리를 적는다
	Local     Local
	Runtime   StepRuntime
	Record    *execenv.Record
	Node      string
	Status    *StatusBook
	Log       *slog.Logger
}

// ScratchRuntime 은 기동이 지은 scratch 의 일꾼이다 — 삭제자와 보존의 판정.
type ScratchRuntime struct {
	// Keeper 는 Worker.Checkpoints 에 앉는다. native 노드에도 있다.
	Keeper *CheckpointKeeper
	// AfterReport 는 Worker.AfterReport 에 앉는다 — 삭제자와 보존의 판정을 깨운다. native 면 nil 이다.
	AfterReport func()

	wg sync.WaitGroup
}

// Wait 는 배경 고루틴이 ctx 로 끝나기를 기다린다.
func (s *ScratchRuntime) Wait() {
	if s != nil {
		s.wg.Wait()
	}
}

// StartScratch 는 scratch 의 기동이다 (trash 유닛 · checkpoint 유닛 · 흐름 4절). 차례 — 기동 청소 → 삭제자 (배경) →
// spool 자리 확인 → 보존의 조정과 판정 (배경) → 광고. 광고는 기다리지 않는다 — 둘 다 배경이다 (trash 답 10).
//
// cmd/enode 의 main 이 부르는 줄 하나가 되려고 여기 모았다 — 그 갈래는 기본 go test 에서 닿지 않는다 (계획 3.2).
func StartScratch(ctx context.Context, in ScratchSetup) *ScratchRuntime {
	log := in.Log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	policy := in.Local.CheckpointPolicy()
	status := CheckpointStatus{Policy: string(policy.Mode), TTLHours: int(policy.TTL / time.Hour),
		CapacityPercent: policy.CapacityPercent}
	support := NativeRuntime{}.Capability().Capture
	if in.Runtime != nil {
		support = in.Runtime.Capability().Capture
	}
	k := &CheckpointKeeper{Policy: policy, Support: support, Node: in.Node, Runtime: "runc-overlay",
		Workspace: in.Workspace, Log: log, Status: in.Status}
	if in.Record != nil {
		k.Env = in.Record.PreparedEnvironment
		if in.Record.Runtime != "" {
			k.Runtime = in.Record.Runtime
		}
	}
	out := &ScratchRuntime{Keeper: k}
	if in.Scratch == "" || !support.Supported {
		status.Unsupported = scratch.ReasonRuntime
		in.Status.SetCheckpoint(status)
		log.Info("checkpoint: this node runs steps without an isolated upper and keeps nothing (runtime)")
		return out
	}

	// 단계가 남긴 작업 폴더를 지우는 자리 (ADR-076 §4.1). 닫기는 작업 폴더를 trash 로 rename 한 번에 옮기기만 한다.
	// 기동 청소가 먼저 죽은 데몬이 남긴 작업 폴더를 trash 로 옮기고, 삭제자는 곧바로 한 번 · 결과 보고 뒤마다 ·
	// 항목이 남아 있으면 10분마다 깬다.
	SweepOrphanSessions(in.Scratch, log)
	trash := scratch.TrashIn(in.Scratch)
	deleter := &scratch.Deleter{Trash: trash, Launch: TrashLauncher(trash), Changed: in.Status.SetScratch, Log: log}

	// 실패한 단계의 보존 (ADR-076 §5). spool 자리를 확인하고 (NFR C8), 조정과 판정은 배경에서 돈다.
	store := &scratch.Store{Dir: scratch.SpoolIn(in.Scratch), Scratch: in.Scratch, Trash: trash, Policy: policy,
		Measure: MeasureLauncher()}
	check, err := store.Open()
	switch {
	case err != nil:
		log.Warn("checkpoint reconcile failed", "err", err)
	case check.Refused != "":
		log.Warn("the checkpoint spool is not a private directory of this node; nothing will be kept",
			"why", check.Refused)
	case check.Narrowed:
		log.Info("checkpoint spool permissions narrowed to 0700", "mode", fmt.Sprintf("%o", check.From))
	}
	k.Store, k.OnTrash = store, deleter.Kick
	in.Status.SetCheckpoint(status)
	if policy.Mode == scratch.ModeOff {
		log.Info("checkpoint policy off; failed steps leave nothing behind")
	} else {
		log.Info(fmt.Sprintf("checkpoint policy %s, ttl %dh, capacity %d percent of kept plus free space; "+
			"change it under checkpoint: in %s", policy.Mode, status.TTLHours, policy.CapacityPercent, in.Config))
	}
	out.AfterReport = func() {
		deleter.Kick()
		k.Kick()
	}
	out.wg.Add(2)
	go func() { defer out.wg.Done(); deleter.Run(ctx) }()
	go func() { defer out.wg.Done(); k.Run(ctx) }()
	return out
}
