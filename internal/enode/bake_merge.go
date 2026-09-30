package enode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/lower"
	"github.com/taeels/enode/internal/scratch"
)

// 굽기의 merge 단계다 (bake 유닛 · business-rules.md 7 · 8 · 10절 · business-logic-model.md 3절).
//
//	1   거절 확인           Baker 가 없다 · 제자리에 쓰는 노드 -> 거절.  상태 자리 · state.json -> FAILED (held 면 몸통)
//	2   claim 확인          mergeClaim (7.1 의 표)
//	3   마감                claim 을 받은 때 (노드 시계) + MergeWait
//	4   배타                Exclusive — 기다리는 동안 단계 로그에 쥔 쪽 줄
//	5   그물                ForeignMounts — 찾으면 배타를 놓고 foreignRetry 뒤 4 로
//	6   trash               <대기 upper 의 scratch>/trash 를 0700 으로
//	7   helper preflight    어긋남 · 읽기 실패 -> 배타 Release · 몸통 · FAILED
//	8   합치기 시작 표지     startMerging — 몸통이 먼저 돌았으면 배타를 놓고 끝
//	9   merging             못 쓰면 표지를 거두고 배타 Release · 몸통 · FAILED
//	10  helper apply        오류 -> merging 에 둔다 (답 2)
//	11  upper 뿌리 rmdir · 12 metadata · 13 committed · 14 대기 자리 trash · 15 놓기 · 16 merged · 업로드 · 보고
//
// merging 을 쓴 뒤에는 되돌리지 않는다 — lower 가 반쯤 바뀌었을 수 있다. 재개가 끝낸다.

var (
	// foreignMounts 는 그물이다 — 시험이 찾음 · 오류를 끼운다.
	foreignMounts = lower.ForeignMounts
	// writeMetadata 는 합치기의 마지막 동작이다 — 시험이 쓰기 실패를 끼운다.
	writeMetadata = lower.WriteMetadata
	// movePending 은 대기 자리를 그 scratch 의 trash 로 옮긴다 — root 로 도는 시험이 옮기기 실패를 끼운다.
	movePending = discardPending
)

// mergeNode 는 merge 단계 하나가 도는 동안의 사실이다.
type mergeNode struct {
	w      *Worker
	b      *Baker
	h      *heldBake
	step   *Step
	runCtx context.Context
	ctx    context.Context
	log    *slog.Logger
	sl     *bakeLog
}

func (w *Worker) runMergeStep(runCtx, ctx context.Context, step *Step, log *slog.Logger) {
	claimedAt := time.Now() // 노드 시계 — 마감을 지키는 쪽이 노드다 (business-rules.md 8.1)
	b := w.Bake
	if b == nil || WorkspaceWrites(w.Runtime) != writesIsolated {
		w.report(ctx, step, Result{Node: w.Ident.NodeID, Error: refusedText})
		return
	}
	m := &mergeNode{w: w, b: b, step: step, runCtx: runCtx, ctx: ctx, log: log, sl: w.newBakeLog(step)}
	defer m.sl.finish()
	dir := b.guard.Dir()
	b.mu.Lock()
	if b.held != nil && b.held.run == step.RunID {
		m.h = b.held
	}
	b.mu.Unlock()

	if dir == nil {
		m.fail(cannotOpenState(errors.New("the node has not opened it; see the node log")), "")
		return
	}
	st, err := dir.ReadState()
	if err != nil {
		m.fail(cannotOpenState(err), "")
		return
	}
	switch mergeClaim(m.h != nil, st, step.RunID) {
	case claimMismatch:
		// 상태가 이 굽기의 것이 아니다 — 상태를 건드리지 않고 놓는다 · last_attempt 를 남기지 않는다
		m.h.letGo()
		m.finish(Result{Error: mismatchText(st)})
		return
	case claimRestartLeft:
		m.finish(Result{Error: restartLeftText})
		return
	case claimRestartDiscarded:
		m.finish(Result{Error: restartDiscardedText})
		return
	case claimNothing:
		// 합칠 것 없음은 완주다 — 판정은 계약의 조건이 한다 (되물음 2 답 B · I3)
		m.sl.line(nothingToMergeLine)
		m.finish(Result{})
		return
	}

	wait := contractStep(step).MergeWait()
	deadline := claimedAt.Add(wait)
	dctx, cancel := context.WithDeadline(runCtx, deadline)
	defer cancel()
	ex, err := m.exclusive(dctx, dir, deadline)
	if err != nil {
		m.gaveUp(dctx, wait, err)
		return
	}

	pending := m.h.pendingDir()
	upper := filepath.Join(pending, "upper")
	trash := scratch.TrashIn(pendingScratch(pending)).Dir
	if err := os.MkdirAll(trash, 0o700); err != nil {
		_ = ex.Release()
		m.fail("cannot create the trash of the pending upper: "+err.Error(), "")
		return
	}
	req := mergeHelperRequest{Op: "preflight", Upper: upper, Lower: lowerRootOf(dir), Trash: trash}
	if _, err := callMergeHelper(req); err != nil {
		// 시작 전 확인은 아무것도 바꾸지 않는다 — lower 는 그대로 · upper 는 trash · committed (FD 규칙 10절)
		_ = ex.Release()
		m.fail(err.Error(), "")
		return
	}
	if beforeStartMerging != nil {
		beforeStartMerging()
	}
	if !m.h.startMerging() {
		// 몸통이 먼저 돌았다 — 임대가 끝났으므로 보고는 닿지 않는다 (FD 흐름 4절 표 첫째 줄)
		_ = ex.Release()
		log.Warn("bake: the bake was discarded before the merge started; not merging", "run", step.RunID)
		return
	}
	merging := lower.State{Phase: lower.PhaseMerging, Owner: st.Owner, PendingUpper: st.PendingUpper,
		Since: b.now().UTC(), LastAttempt: st.LastAttempt}
	if err := writeLowerState(m.h.lock, merging); err != nil {
		// 합치기 전이다 — 표지를 거둬야 몸통이 돈다 (계획 4.1 13번)
		m.h.cancelMerging()
		_ = ex.Release()
		m.fail("cannot write the lower state: "+err.Error(), "")
		return
	}

	began := time.Now()
	req.Op = "apply"
	result, err := callMergeHelper(req)
	if err == nil {
		// 빈 upper 뿌리를 치운다 — Apply 가 끝났다는 표지를 겸한다 (FD 규칙 12.4)
		err = os.Remove(upper)
	}
	d := m.h.currentDraft()
	mergedAt := b.now().UTC()
	if err == nil {
		err = writeMetadata(lowerRootOf(dir), metadataOf(*d, step.RunID, b.node, mergedAt, false))
	}
	if err != nil {
		m.stopped(ex, err)
		return
	}
	m.sl.line(mergedLine(result, time.Since(began)))
	log.Info("bake: merged", "run", step.RunID, "ops", result.Ops, "discarded", result.Discarded,
		"took", time.Since(began).Round(time.Millisecond))
	if err := writeLowerState(m.h.lock, lower.State{Phase: lower.PhaseCommitted, Since: b.now().UTC()}); err != nil {
		// 합치기는 끝났다 — 대기 자리를 두고 (초안이 남아 재개가 끝났나를 본다) DONE 이다 (되물음 5 답 A 의 (3))
		m.sl.line("the merge finished but cleaning up failed: " + err.Error() + "; the next resume writes committed")
		log.Warn("bake: the merge finished but committed was not written", "run", step.RunID, "err", err)
	} else if err := movePending(pending); err != nil {
		log.Warn("bake: cannot move the pending upper to trash; it is left for the start-up cleanup",
			"path", pending, "err", err)
	}
	_ = ex.Release()
	m.h.letGo()
	m.merged(contract.MergeResult{IR: d.Source.IR, PreviousIR: d.PreviousIR, MergedAt: mergedAt, Ops: mergeOpsOf(result)})
}

// exclusive 는 배타를 잡는다 (FD 규칙 8절). 막혀 있는 동안 waitLog 이 쓸 줄을 단계 로그에 쓰고, 노드 로그에는 시작 ·
// 잡음 · 마감에 한 줄씩 쓴다. 잡은 뒤 그물을 친다 — 찾으면 배타를 놓고 foreignRetry 뒤 다시 기다린다 (마감은 그대로).
// 그물이 오류면 경고 줄을 쓰고 배타만으로 합친다 — 증거는 배타 잠금이다.
func (m *mergeNode) exclusive(ctx context.Context, dir *lower.Dir, deadline time.Time) (*lower.Exclusive, error) {
	wl := &waitLog{deadline: deadline, every: waitLogEvery}
	for {
		began, waited := time.Now(), false
		ex, err := dir.Exclusive(ctx, mergeWatchEvery, func(wt lower.Waiting) {
			if !waited {
				m.log.Info("waiting for the lower lock (merge step)", "run", m.step.RunID, "event", "waiting",
					"deadline", clockText(deadline))
			}
			waited = true
			for _, line := range wl.lines(m.b.now(), wt) {
				m.sl.line(line)
			}
		})
		if err != nil {
			return nil, err
		}
		if waited {
			m.sl.line(tookLine(time.Since(began)))
			m.log.Info("waiting for the lower lock (merge step)", "run", m.step.RunID, "event", "took",
				"after", time.Since(began).Round(time.Second))
		}
		scan, err := foreignMounts(dir.Root)
		if err != nil {
			m.sl.line("cannot scan mounts of other processes: " + err.Error() + "; merging on the lower lock alone")
			return ex, nil
		}
		if scan.Unreadable > 0 {
			m.sl.line(fmt.Sprintf("%d processes could not be read while scanning mounts", scan.Unreadable))
		}
		if len(scan.Found) == 0 {
			return ex, nil
		}
		_ = ex.Release()
		for _, line := range foundLines(scan, foreignRetry) {
			m.sl.line(line)
		}
		m.log.Warn("bake: another mount namespace mounts this lower; waiting", "run", m.step.RunID,
			"found", len(scan.Found))
		if err := sleep(ctx, foreignRetry); err != nil {
			return nil, err
		}
	}
}

// foundLines 는 그물이 찾은 마운트의 줄이다 (FD 규칙 8.4). 마운트 자리를 보이는 것이 목적이라 host 경로를 쓴다.
func foundLines(scan lower.MountScan, retry time.Duration) []string {
	noun := "mount"
	if len(scan.Found) != 1 {
		noun = "mounts"
	}
	out := []string{fmt.Sprintf("found %d overlay %s of this lower in another mount namespace; releasing the lock "+
		"and waiting %gs", len(scan.Found), noun, retry.Seconds())}
	for _, f := range scan.Found {
		out = append(out, fmt.Sprintf("  pid %d namespace %s mount %s lowerdir %s", f.PID, f.Namespace, f.MountPoint,
			f.LowerDir))
	}
	return out
}

// sleep 은 ctx 를 보며 d 만큼 기다린다.
func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// gaveUp 은 배타를 못 잡고 끝난 셋이다 (FD 규칙 8.5) — 데몬 · 임대 · 마감. 셋 모두 몸통이다 (upper 는 trash ·
// committed · drain 이 다음 광고에서 풀린다).
func (m *mergeNode) gaveUp(dctx context.Context, wait time.Duration, err error) {
	switch {
	case m.ctx.Err() != nil:
		m.h.abandon("node stopped", m.h.draftBuilds())
	case m.runCtx.Err() != nil:
		m.fail("aborted: lease expired", "")
	case errors.Is(dctx.Err(), context.DeadlineExceeded):
		m.log.Warn("waiting for the lower lock (merge step)", "run", m.step.RunID, "event", "gave up",
			"wait", wait.String())
		m.h.abandon(contract.ReasonMergeWaitTimeout, m.h.draftBuilds())
		m.finish(Result{Error: mergeWaitText(wait), Reason: contract.ReasonMergeWaitTimeout})
	default:
		m.fail(cannotOpenState(err), "")
	}
}

// fail 은 합치기 전의 끝이다 — 이 프로세스가 그 굽기를 쥐었으면 몸통 (last_attempt 는 그 문장) · FAILED.
func (m *mergeNode) fail(text, reason string) {
	if m.h != nil {
		m.h.abandon(attemptReason(reason, text), m.h.draftBuilds())
	}
	m.finish(Result{Error: text, Reason: reason})
}

// stopped 는 merging 을 쓴 뒤의 오류다 — merging 에 둔다 · 두 잠금을 놓는다 · DropBake · held 를 비운다 · FAILED
// (원인 코드 없음). 재시도 간격을 걸지 않는다 — 다음 광고에서 보통 이 노드 자신이 잇는다 (답 2).
func (m *mergeNode) stopped(ex *lower.Exclusive, err error) {
	_ = ex.Release()
	m.h.letGo()
	m.log.Warn("bake: the merge stopped; the lower stays merging", "run", m.step.RunID, "err", err)
	m.finish(Result{Error: mergeStoppedText(err)})
}

// finish 는 합치지 않은 끝의 보고다 — 단계 로그만 올린다. 데몬이 멈췄으면 보고하지 않는다.
func (m *mergeNode) finish(res Result) {
	if m.ctx.Err() != nil {
		m.sl.finish()
		return
	}
	res.Node = m.w.Ident.NodeID
	m.w.uploadStepLog(m.ctx, m.step, m.sl, m.log)
	m.w.report(m.ctx, m.step, res)
}

// merged 는 합친 끝이다 — $OUT/merged 하나와 단계 로그를 업로드 예산 안에 올린다 (CG 물음 1 답 A). 넘기면 FAILED
// upload_timeout 이고 merge 칸은 싣는다 — 합치기는 끝났다. exit_code · Finalize · exited 는 없다.
func (m *mergeNode) merged(r contract.MergeResult) {
	if m.ctx.Err() != nil {
		m.sl.finish()
		return
	}
	res := Result{Node: m.w.Ident.NodeID, Merge: &r}
	_, uploadBudget := m.w.budgetsFor(m.step)
	out, err := os.MkdirTemp("", "enode-bake-out-")
	if err == nil {
		defer func() { _ = os.RemoveAll(out) }()
		body, _ := json.MarshalIndent(r, "", "  ")
		err = os.WriteFile(filepath.Join(out, contract.ArtifactMerged), append(body, '\n'), 0o644)
	}
	if err != nil {
		// 올릴 파일을 못 지었다 — 합치기는 끝났으므로 lower 는 그대로 두고 단계 로그만 올린다
		m.log.Warn("bake: cannot write the merged result", "err", err)
	}
	m.sl.finish()
	uctx, cancel := context.WithTimeout(m.runCtx, uploadBudget)
	produced, uploaded := m.w.upload(uctx, m.step, out, m.sl.buf.Bytes(), err == nil, m.log)
	cancel()
	leaseEnded := m.runCtx.Err() != nil && m.ctx.Err() == nil
	res.Produced = produced
	_, res.Upload, res.Reason, res.Error = settle(settleIn{upload: uploaded, leaseEnded: leaseEnded,
		uploadBudget: uploadBudget})
	if leaseEnded {
		res.Error = "aborted: lease expired"
	}
	m.w.report(m.ctx, m.step, res)
}
