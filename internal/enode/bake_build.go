package enode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/lower"
)

// 굽기의 build 단계다 (bake 유닛 · business-rules.md 1 ~ 6 · 15절 · business-logic-model.md 2절).
//
//	1   굽기 잠금 · state          takeForBuild — 낡은 building · pending 은 정리 · merging 은 재개를 열고 거절
//	3   previous_ir               lower 의 metadata 의 source.ir
//	4   대기 자리 · building        MkdirTemp · WriteState(building) — pending_upper 를 이때 적는다
//	6   세션 · bash 확인             sh -c "command -v bash" — 0 이 아니면 계약의 명령을 하나도 안 돌린다
//	7   sync                       bash -c <sync>
//	8   IR 대조                     sh -c <probeScript> — 어긋나면 DONE · ir_mismatch · 못 대 보면 FAILED
//	9   pinned                     repo 모양이면 bash -c "repo manifest -r -o .enode-manifest.xml"
//	10  builds                     적힌 차례로 · 첫 실패에서 멈춘다
//	11  성공 끝                     exited · Finalize · Close(Keep{Upper}) · sha256 · 초안 · pending · HoldBake · manifest
//
// 계약의 명령과 노드의 고정 한 줄은 모두 세션의 Run 으로만 돈다 — 호스트 프로세스를 띄우지 않는다 (계획 3.1).
// held 가 있는 동안의 오류 길은 모두 치우는 몸통으로 끝난다 (결정 44).

// bakeLog 는 단계 로그다 — 버퍼와 진행 청크 둘로 간다 (business-rules.md 8.6 · 16.2). host 경로를 쓰지 않는다.
type bakeLog struct {
	buf  bytes.Buffer
	sink io.Writer
	stop func()
}

func (w *Worker) newBakeLog(step *Step) *bakeLog {
	l := &bakeLog{}
	l.sink = &l.buf
	tee, stop := w.transcript(step)
	if tee != nil {
		l.sink = io.MultiWriter(&l.buf, tee)
	}
	l.stop = stop
	return l
}

func (l *bakeLog) line(s string) { _, _ = io.WriteString(l.sink, s+"\n") }

// finish 는 진행 청크의 꼬리를 비운다 — 선별본을 올리기 전에 부른다. 두 번 불러도 된다.
func (l *bakeLog) finish() {
	if l.stop != nil {
		l.stop()
		l.stop = nil
	}
}

// uploadStepLog 는 Finalize 에 닿지 않은 굽기 단계의 끝이다 — 단계 로그만 올린다. Worker 의 ctx 에 업로드 예산을 건다.
func (w *Worker) uploadStepLog(ctx context.Context, step *Step, l *bakeLog, log *slog.Logger) {
	l.finish()
	_, uploadBudget := w.budgetsFor(step)
	uctx, cancel := context.WithTimeout(ctx, uploadBudget)
	defer cancel()
	if err := w.Client.UploadLog(uctx, step.RunID, step.Seq, step.Name, l.buf.Bytes()); err != nil && ctx.Err() == nil {
		log.Warn("log upload failed", "err", err)
	}
}

// takeForBuild 는 build claim 의 1 · 2 다 (business-rules.md 3절). 굽기를 쥐었으면 held 와 building 에 실을
// last_attempt 를 돌려주고, 거절이면 reason 과 문장을 돌려준다. guard.Dir 은 Baker.mu 앞에서 부른다 (계획 4.1 12번).
func (b *Baker) takeForBuild(step *Step) (*heldBake, *lower.LastAttempt, string, string) {
	dir := b.guard.Dir()
	if dir == nil {
		return nil, nil, "", cannotOpenState(errors.New("the node has not opened it; see the node log"))
	}
	b.mu.Lock()
	if b.held != nil || b.resuming {
		b.mu.Unlock()
		st, _ := dir.ReadState()
		return nil, nil, contract.ReasonBakeInProgress, bakeInProgressText(st)
	}
	lock, ok, err := dir.TryBake()
	if err != nil {
		b.mu.Unlock()
		return nil, nil, "", cannotOpenState(err)
	}
	if !ok {
		b.mu.Unlock()
		st, _ := dir.ReadState()
		return nil, nil, contract.ReasonBakeInProgress, bakeInProgressText(st)
	}
	st, err := dir.ReadState()
	if err != nil {
		_ = lock.Release()
		b.mu.Unlock()
		return nil, nil, "", cannotOpenState(err)
	}
	if st.Phase == lower.PhaseMerging {
		// 잡은 잠금을 넘겨 재개를 배경에 연다 — 굽기 Run 의 임대가 묶이지 않는다 (되물음 4 답 A)
		b.mu.Unlock()
		b.openResume(dir, lock, st, fromBuild)
		return nil, nil, contract.ReasonBakeInProgress, resumeOpenedText(ownerRun(st))
	}
	h := &heldBake{b: b, run: step.RunID, step: step.Seq, dir: dir, lock: lock}
	b.held = h
	b.mu.Unlock()
	base := st.LastAttempt
	if st.Phase == lower.PhaseBuilding || st.Phase == lower.PhasePending {
		next, err := b.clean(dir, lock, st, fromBuild)
		if err != nil {
			b.log.Warn("bake: cannot write the lower state while cleaning a stale bake", "err", err)
		}
		base = next.LastAttempt
	}
	return h, base, "", ""
}

// previousIR 은 굽기 잠금을 잡은 뒤 읽은 lower 의 source.ir 이다 — 초안에 둔다 (결정 4). 못 읽으면 null 과 경고 한 줄.
func (b *Baker) previousIR(dir *lower.Dir) *string {
	md, err := lower.ReadMetadata(lowerRootOf(dir))
	if err != nil {
		b.log.Warn("bake: cannot read the previous metadata; previous_ir is null", "err", err)
		return nil
	}
	if md == nil || md.Source.IR == nil {
		return nil
	}
	ir := *md.Source.IR
	return &ir
}

// buildRun 은 build 단계 하나가 도는 동안의 사실이다.
type buildRun struct {
	w       *Worker
	b       *Baker
	h       *heldBake
	step    *Step
	runCtx  context.Context
	ctx     context.Context
	log     *slog.Logger
	session StepSession
	dir     string // 워크스페이스 (lower 루트)
	out     string // 세션에 준 $OUT — 올리지 않는다
	paths   RuntimePaths
	env     []string
	sl      *bakeLog
	owner   *lower.Owner
	base    *lower.LastAttempt

	sync   lower.BuildRecord
	synced bool
	builds []lower.BuildRecord
	last   lower.BuildRecord // 마지막으로 돈 계약의 명령 — exited 와 exit_code
	probe  *irProbe
	match  bool
	pinned bool
}

func (w *Worker) runBuildStep(runCtx, ctx context.Context, step *Step, dir, in, out string, log *slog.Logger) {
	// 제자리에 쓰는 노드는 거절한다 — Bake 가 있어도 본다. NativeRuntime 으로 떨어뜨리지 않는다 (계획 4.1 2번) —
	// 떨어뜨리면 계약의 명령이 호스트에서 돈다.
	b := w.Bake
	if b == nil || WorkspaceWrites(w.Runtime) != writesIsolated {
		w.report(ctx, step, Result{Node: w.Ident.NodeID, Error: refusedText})
		return
	}
	h, base, reason, text := b.takeForBuild(step)
	if h == nil {
		w.report(ctx, step, Result{Node: w.Ident.NodeID, Error: text, Reason: reason})
		return
	}
	log.Info("bake: holding the lower for run "+step.RunID, "run", step.RunID)
	r := &buildRun{w: w, b: b, h: h, step: step, runCtx: runCtx, ctx: ctx, log: log, base: base, dir: dir, out: out,
		owner: &lower.Owner{Run: step.RunID, Step: step.Seq, Node: b.node, Instance: b.instance}}
	previousIR := b.previousIR(h.dir)

	pending, err := makePending(b.scratch, h.dir.Root.Key.String())
	if err != nil {
		r.refuse(err.Error())
		return
	}
	h.setPending(pending)
	upper := filepath.Join(pending, "upper")
	building := lower.State{Phase: lower.PhaseBuilding, Owner: r.owner, PendingUpper: upper, Since: b.now().UTC(),
		LastAttempt: base}
	if err := writeLowerState(h.lock, building); err != nil {
		r.refuse("cannot write the lower state: " + err.Error())
		return
	}
	log.Info("bake: building", "run", step.RunID, "pending", pending)

	session, err := w.Runtime.Open(runCtx, RuntimeSpec{RunID: step.RunID, StepID: step.StepID, Dir: dir, In: in,
		Out: out, Record: w.RuntimeRecord})
	if err != nil {
		r.refuse("runtime open: " + err.Error())
		return
	}
	r.session = manageSession(session)
	defer r.session.Close(ctx, Keep{}) //nolint:errcheck // 첫 Close 가 오류를 결과로 옮긴다 — 이것은 안전망이다
	r.paths = r.session.Paths()
	r.env = harnessEnv(append(commandEnv, step.Env...),
		map[string]string{"OUT": r.paths.Out, "IN": r.paths.In, contract.EnvIR: step.IR}, nil)
	r.sl = w.newBakeLog(step)
	defer r.sl.finish()

	// 6' bash 확인 — sync 앞 (되물음 3 답 A · 결정 45)
	code, _, _, err := r.nodeCommand([]string{"sh", "-c", bashCheck})
	if r.interrupted(code, err) {
		return
	}
	if text := bashCheckText(code, err); text != "" {
		r.stopEarly(text)
		return
	}

	// 7 sync
	rec, code, err := r.command("sync", "sync", step.Sync)
	if r.interrupted(code, err) {
		return
	}
	r.sync, r.synced, r.last = rec, true, rec
	if code != 0 {
		for _, bld := range step.Builds {
			r.sl.line(skipLine(bld.Name, "sync failed"))
		}
		r.failEnd(exitReason("sync", code), "", "")
		return
	}

	// 8 IR 대조 — 세션 안 · IR 은 환경 변수로만
	code, stdout, stderr, err := r.nodeCommand([]string{"sh", "-c", probeScript})
	if r.interrupted(code, err) {
		return
	}
	p := parseProbe(stdout, code, stderr)
	r.probe = &p
	switch outcome, text := irVerdict(step.IR, p); outcome {
	case irMatch:
		r.match = true
		r.sl.line("bake: " + text)
	case irNotLocal, irElsewhere:
		for _, bld := range step.Builds {
			r.sl.line(skipLine(bld.Name, "ir "+step.IR+" did not match"))
		}
		r.sl.line("bake: " + text)
		r.failEnd(contract.ReasonIRMismatch, contract.ReasonIRMismatch, "")
		return
	default:
		r.failEnd(text, "", text)
		return
	}

	// 9 pinned — repo 모양만 · IR 이 맞은 바로 뒤 · builds 앞
	if p.Mode == "repo" {
		code, _, stderr, err := r.nodeCommand([]string{"bash", "-c", pinCommand})
		if r.interrupted(code, err) {
			return
		}
		if code != 0 {
			text := pinFailedText(code, lastLine(stderr))
			r.failEnd(text, "", text)
			return
		}
		r.pinned = true
		r.sl.line(pinnedLine)
	}

	// 10 builds — 첫 실패에서 멈춘다 (답 3)
	for i, bld := range step.Builds {
		rec, code, err := r.command(buildLabel(bld.Name), bld.Name, bld.Command)
		if r.interrupted(code, err) {
			return
		}
		r.builds = append(r.builds, rec)
		r.last = rec
		if code != 0 {
			for _, rest := range step.Builds[i+1:] {
				r.sl.line(skipLine(rest.Name, bld.Name+" failed"))
			}
			r.failEnd(exitReason(buildLabel(bld.Name), code), "", "")
			return
		}
	}
	r.succeed(upper, previousIR)
}

// refuse 는 세션 앞에서 held 를 둔 뒤의 노드 쪽 오류다 — 몸통 · FAILED (결정 44). 계약의 명령이 돌지 않았다.
func (r *buildRun) refuse(text string) {
	r.h.abandon(text, nil)
	r.w.report(r.ctx, r.step, Result{Node: r.w.Ident.NodeID, Error: text})
}

// command 는 계약의 명령 하나다 — ["bash", "-c", <계약의 문자열>] (되물음 3 답 A). 출력은 단계 로그와 진행 청크로,
// 머리와 끝 줄은 노드가 쓴다. 시간 상한이 없다 — 임대가 끝나면 끊긴다.
func (r *buildRun) command(label, name, command string) (lower.BuildRecord, int, error) {
	r.sl.line(startedLine(label))
	start := time.Now()
	code, err := r.session.Run(r.runCtx, ProcessSpec{Argv: []string{"bash", "-c", command}, Dir: r.paths.Dir,
		Env: r.env, Stdout: r.sl.sink, Stderr: r.sl.sink})
	end := time.Now()
	rec := lower.BuildRecord{Name: name, Command: command, StartedAt: start.UTC(), FinishedAt: end.UTC(), ExitCode: code}
	if code >= 0 && r.runCtx.Err() == nil {
		r.sl.line(exitedLine(label, code, end.Sub(start)))
	}
	return rec, code, err
}

// nodeCommand 는 노드의 고정 한 줄이다 — 단계 로그에 흘리지 않는다 (계획 4.1 9번). stdout 은 64 KiB 까지, stderr
// 는 뒤쪽만 남긴다.
func (r *buildRun) nodeCommand(argv []string) (int, string, string, error) {
	stdout, stderr := &diffBuffer{limit: probeOutMax}, &tailBuffer{max: 4 << 10}
	code, err := r.session.Run(r.runCtx, ProcessSpec{Argv: argv, Dir: r.paths.Dir, Env: r.env, Stdout: stdout,
		Stderr: stderr})
	return code, stdout.buf.String(), stderr.String(), err
}

// interrupted 는 명령이 끊겼나다 — 데몬이 멈춤 · 임대가 끝남 · 세션 오류 (exit -1). 끊겼으면 Finalize 없이 끝낸다.
func (r *buildRun) interrupted(code int, err error) bool {
	switch {
	case r.ctx.Err() != nil:
		r.stopEarly("node stopped")
	case r.runCtx.Err() != nil:
		r.stopEarly("aborted: lease expired")
	case code < 0:
		r.stopEarly(runtimeRunText(err))
	default:
		return false
	}
	return true
}

// stopEarly 는 Finalize 에 닿지 않는 끝이다 — 닫고 (upper 는 trash) · 몸통 · 단계 로그 · 보고. exited 와 exit_code 가
// 없다 (명령이 하나도 안 돌았거나 끝나지 않았다 · 계획 4.1 28번). 데몬이 멈췄으면 보고하지 않는다 (ADR-030).
func (r *buildRun) stopEarly(reason string) {
	if err := r.session.Close(r.ctx, Keep{}); err != nil {
		r.log.Warn("runtime cleanup failed", "err", err)
	}
	r.h.abandon(reason, r.builds)
	if r.ctx.Err() != nil {
		r.sl.finish()
		r.log.Warn("bake: the node stopped during the build step", "run", r.step.RunID)
		return
	}
	r.w.uploadStepLog(r.ctx, r.step, r.sl, r.log)
	r.w.report(r.ctx, r.step, Result{Node: r.w.Ident.NodeID, Error: reason, Environment: r.session.Environment(),
		Build: r.manifest()})
}

// manifest 는 result 의 build 칸이다 — 계약의 명령이 하나라도 끝났으면 늘 (business-rules.md 15절). head 와
// head_tags 는 대조가 HEAD 와 태그를 읽었을 때만 있다 — 못 대 봤으면 "" 와 null.
func (r *buildRun) manifest() *contract.BuildManifest {
	if !r.synced {
		return nil
	}
	var head string
	var tags []string
	if r.probe != nil && r.probe.Exit == 0 && r.probe.Head != "" {
		head, tags = r.probe.Head, r.probe.Tags
	}
	var ir *string
	var pinned *lower.Pinned
	if r.match {
		v := r.step.IR
		ir = &v
		if d := r.h.currentDraft(); d != nil {
			pinned = d.Source.Pinned
		}
	}
	m := buildManifestOf(r.sync, r.builds, head, ir, pinned, tags)
	return &m
}

// exited 는 종료 보고를 시작한다 — 마지막으로 돈 계약의 명령이다 (business-rules.md 5절).
func (r *buildRun) exited() (*exitReporter, time.Time) {
	at := r.last.FinishedAt
	code := r.last.ExitCode
	return r.w.startExitReport(r.ctx, r.step, contract.Exited{Node: r.w.Ident.NodeID, Instance: r.w.Client.Instance,
		Attempt: r.step.Attempt, Outcome: contract.Outcome{Kind: contract.OutcomeExit, Code: &code}, ExitedAt: at}), at
}

// failEnd 는 계약의 명령이 돈 뒤 합칠 upper 를 남기지 않는 끝이다 — 명령 실패 · IR 어긋남 (DONE) 과 대조를 못 함 ·
// pinned 실패 (FAILED). exited (마지막 명령) · Finalize · Close(Keep{}) · 몸통 · 단계 로그만 올린다 · 보고.
// attempt 는 last_attempt 의 reason, code 는 원인 코드 (ir_mismatch), errText 는 노드 쪽 오류의 문장이다.
func (r *buildRun) failEnd(attempt, code, errText string) {
	reporter, exitedAt := r.exited()
	spec := finalizeSpecFor(r.step, true, r.w.Local)
	spec.Workspace, spec.Out = r.dir, r.out
	res := Result{Node: r.w.Ident.NodeID, Environment: r.session.Environment(), ExitedAt: &exitedAt}
	after := func(time.Time, error, error) (bool, error) {
		r.h.abandon(attempt, r.builds)
		r.sl.finish()
		return false, nil
	}
	leaseEnded := r.w.closeOut(r.runCtx, r.ctx, r.step, r.session, spec, exitedAt, r.sl.buf.Bytes, false, &res,
		r.log, closing{after: after})
	reporter.Stop()
	if r.ctx.Err() != nil {
		return
	}
	res.Build = r.manifest()
	switch {
	case leaseEnded:
		res.Error = "aborted: lease expired"
	default:
		exit := r.last.ExitCode
		res.ExitCode = &exit
		if errText != "" {
			res.Error = joinErr(errText, res.Error)
		}
		if res.Reason == "" {
			res.Reason = code
		}
	}
	r.log.Info("bake: the build step left no pending upper", "run", r.step.RunID, "reason", attempt)
	r.w.report(r.ctx, r.step, res)
}

// joinErr 는 앞의 문장에 settle 의 문장을 잇는다.
func joinErr(first, rest string) string {
	if rest == "" {
		return first
	}
	return first + "; " + rest
}

// succeed 는 성공 끝이다 — exited (마지막 build) · Finalize · Close(Keep{Upper}) · pinned 의 sha256 · 초안 ·
// pending 쓰기가 Finalize 예산 안이고, 넘겼으면 pending 을 쓰지 않는다 (business-rules.md 5 · 6절 · 결정 32).
// pending 뒤에 HoldBake · manifest · 업로드 · 보고 DONE. 업로드 예산을 넘기면 곧바로 몸통이다.
func (r *buildRun) succeed(upper string, previousIR *string) {
	reporter, exitedAt := r.exited()
	spec := finalizeSpecFor(r.step, true, r.w.Local)
	spec.Workspace, spec.Out = r.dir, r.out
	res := Result{Node: r.w.Ident.NodeID, Environment: r.session.Environment(), ExitedAt: &exitedAt}
	pending := filepath.Dir(upper)
	// build 의 $OUT 은 노드의 것이다 (되물음 5 답 A · 계획 4.1 5번) — 세션의 $OUT 을 올리지 않고 노드가 만든 폴더에
	// manifest 하나를 써서 올린다. 명령이 쓴 manifest 가 build 의 조건을 참으로 만들지 못한다.
	upload, err := os.MkdirTemp("", "enode-bake-out-")
	if err == nil {
		defer func() { _ = os.RemoveAll(upload) }()
	}
	abandon := func(reason string) { r.h.abandon(reason, r.builds) }
	after := func(deadline time.Time, closeErr, finErr error) (bool, error) {
		defer r.sl.finish()
		switch {
		case r.ctx.Err() != nil:
			abandon("node stopped")
			return false, nil
		case r.runCtx.Err() != nil:
			abandon("aborted: lease expired")
			return false, nil
		case errors.Is(finErr, context.DeadlineExceeded):
			abandon(contract.ReasonFinalizeTimeout)
			return false, nil
		case finErr != nil:
			abandon("runtime finalize: " + finErr.Error())
			return false, nil
		case closeErr != nil:
			abandon("runtime cleanup: " + closeErr.Error())
			return false, nil
		case err != nil:
			abandon("cannot create the manifest directory: " + err.Error())
			return false, fmt.Errorf("cannot create the manifest directory: %w", err)
		}
		var pinned *lower.Pinned
		if r.pinned {
			sum, err := readPinned(filepath.Join(upper, pinnedFile))
			if err != nil {
				abandon(err.Error())
				return false, err
			}
			pinned = &lower.Pinned{File: pinnedFile, SHA256: sum}
		}
		p := *r.probe
		ir := r.step.IR
		d := Draft{Schema: draftSchema, Run: r.step.RunID, Node: r.b.node,
			Source: lower.Source{URL: stripPassword(p.URL), Branch: p.Branch, RepoID: repoIDOf(p), Head: p.Head,
				IR: &ir, Pinned: pinned, SyncCommand: r.step.Sync, SyncedAt: r.sync.FinishedAt},
			Builds: r.builds, PreviousIR: previousIR}
		if rec := r.w.RuntimeRecord; rec != nil {
			d.Environment, d.WorkspaceTarget = rec.PreparedEnvironment, rec.WorkspaceTarget
		}
		if err := writeDraft(pending, d); err != nil {
			abandon(err.Error())
			return false, err
		}
		r.h.setDraft(d)
		if time.Now().After(deadline) {
			abandon(contract.ReasonFinalizeTimeout)
			return true, nil
		}
		st := lower.State{Phase: lower.PhasePending, Owner: r.owner, PendingUpper: upper, Since: r.b.now().UTC(),
			LastAttempt: r.base}
		if err := writeLowerState(r.h.lock, st); err != nil {
			text := "cannot write the lower state: " + err.Error()
			abandon(text)
			return false, errors.New(text)
		}
		h := r.h
		r.b.guard.HoldBake(r.step.RunID, func() { h.abandon(reasonBakeRunEnded, h.draftBuilds()) })
		r.sl.line(pendingLine)
		m := r.manifest()
		body, _ := json.MarshalIndent(m, "", "  ")
		if err := os.WriteFile(filepath.Join(upload, contract.ArtifactManifest), append(body, '\n'), 0o644); err != nil {
			abandon("cannot write the manifest: " + err.Error())
			return false, fmt.Errorf("cannot write the manifest: %w", err)
		}
		r.log.Info("bake: the upper is pending the merge step", "run", r.step.RunID, "pending", pending)
		return false, nil
	}
	leaseEnded := r.w.closeOut(r.runCtx, r.ctx, r.step, r.session, spec, exitedAt, r.sl.buf.Bytes, true, &res,
		r.log, closing{keep: Keep{Upper: upper}, after: after, upload: upload})
	reporter.Stop()
	if res.Upload == contract.StageTimeout {
		// pending 을 쓴 뒤 업로드 예산을 넘겼다 — Run 이 FAILED 로 끝나므로 merge 가 오지 않는다. 광고가 임대가
		// 사라진 것을 볼 때까지 기다리지 않는다 (business-rules.md 6절)
		abandon(contract.ReasonUploadTimeout)
	}
	if r.ctx.Err() != nil {
		return
	}
	res.Build = r.manifest()
	if leaseEnded {
		res.Error = "aborted: lease expired"
	} else {
		exit := r.last.ExitCode
		res.ExitCode = &exit
	}
	r.w.report(r.ctx, r.step, res)
}
