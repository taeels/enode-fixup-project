package enode

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/taeels/enode/internal/lower"
	"github.com/taeels/enode/internal/merge"
	"github.com/taeels/enode/internal/scratch"
)

// 끊긴 합치기의 재개와 광고 주기의 정리다 (bake 유닛 · business-rules.md 12.3 · 12.4 · 13절 · FD 엔티티 10절).
//
// 재개는 셋이 연다 — 기동 · 광고 주기 · build claim. 몸통은 resume 하나이고 연 쪽이 TryBake 로 잡은 굽기 잠금을
// 넘겨받는다. 재개는 HoldBake · DropBake 를 부르지 않는다 — DropBake 가 표지를 합친 뒤의 것으로 적으면 합치기 전에
// 매칭된 늦은 임대가 바뀐 lower 위에서 돈다 (결정 1 · 27). 실패하면 merging 에 두고 그 노드는 10분 뒤다.

// onStale 은 광고 주기의 자리다 — LowerGuard 가 g.mu 아래에서 state 가 building · pending · merging 인 광고에서
// 곧바로 부른다. 막지 않는다 — Baker.mu 아래에서 판단만 하고 배경 일을 b.wg.Go 로 연다 (계획 4.1 18번). 한 번에
// 배경 일 하나다 — resuming 이 광고 주기의 정리도 덮는다.
//
//	멈추는 중 · 이 프로세스가 굽기를 쥠 · 재개 중 · retryAt 전     아무것도 안 한다
func (b *Baker) onStale(lower.State) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed || b.held != nil || b.resuming || b.now().Before(b.retryAt) {
		return
	}
	b.resuming = true
	b.wg.Go(b.stale)
}

// stale 은 광고 주기가 연 배경 일이다 (business-rules.md 13절).
//
//	TryBake 가 오류       로그 한 줄 (원인이 바뀔 때만) — 간격은 걸지 않는다
//	TryBake 가 안 됨      주인이 살아 있거나 다른 쪽이 쥐었다 — 아무것도 안 한다 (실패가 아니다)
//	되면                  state 를 다시 읽는다 — 광고가 읽은 것은 잠금 전의 값이다.  merging 이면 재개 ·
//	                      building · pending 이면 12.2 의 정리 · committed 면 놓는다
func (b *Baker) stale() {
	handed := false
	defer func() {
		if !handed {
			b.mu.Lock()
			b.resuming = false
			b.mu.Unlock()
		}
	}()
	dir := b.guard.Dir()
	if dir == nil {
		return
	}
	lock, ok, err := dir.TryBake()
	if err != nil {
		b.noteErr("bake: cannot take the bake lock", "", err)
		return
	}
	if !ok {
		return
	}
	st, err := dir.ReadState()
	if err != nil {
		_ = lock.Release()
		b.noteErr("bake: cannot read the lower state", "", err)
		return
	}
	switch st.Phase {
	case lower.PhaseMerging:
		b.sweepOwn(dir.Root.Key.String(), filepath.Dir(st.PendingUpper))
		handed = true
		b.resume(b.ctx, dir, lock, st, fromAdvert)
	case lower.PhaseBuilding, lower.PhasePending:
		_, err := b.clean(dir, lock, st, fromAdvert)
		_ = lock.Release()
		if err != nil {
			// committed 를 못 쓴 것만 실패로 친다 — 재개와 같은 간격이다 (결정 53)
			b.retryLater("bake: cannot write the lower state while cleaning a stale bake; trying again in 10m",
				ownerRun(st), err)
			return
		}
		b.mu.Lock()
		b.lastErr = ""
		b.mu.Unlock()
	default:
		_ = lock.Release()
	}
}

// noteErr 는 오류를 원인이 바뀔 때만 노드 로그에 쓴다 — 간격은 걸지 않는다.
func (b *Baker) noteErr(msg, run string, err error) {
	b.mu.Lock()
	changed := err.Error() != b.lastErr
	b.lastErr = err.Error()
	b.mu.Unlock()
	if changed {
		b.log.Warn(msg, "run", run, "err", err)
	}
}

// retryLater 는 실패한 재개 · 정리다 — 이 노드는 staleRetry 뒤에 다시 해 본다. 다른 노드는 자기 광고에서 해 본다.
// 같은 오류가 되풀이되면 로그는 한 번이다 (business-rules.md 13절).
func (b *Baker) retryLater(msg, run string, err error) {
	b.mu.Lock()
	b.retryAt = b.now().Add(staleRetry)
	b.mu.Unlock()
	b.noteErr(msg, run, err)
}

// resume 은 굽기 잠금을 쥔 채 도는 배경 일의 몸통이다 (business-rules.md 12.3). 연 쪽이 TryBake 로 잡은 잠금을
// 넘겨받는다. 마감이 없다 — 배타는 형제가 모두 놓을 때까지 기다린다. ctx 는 데몬의 것이다. 데몬이 멈추면 두
// 잠금을 놓고 merging 그대로 끝난다 — Apply 중이면 helper 를 기다린다 (main 이 Baker.Wait 로 기다린다).
func (b *Baker) resume(ctx context.Context, dir *lower.Dir, lock *lower.Bake, st lower.State, from string) {
	defer func() {
		b.mu.Lock()
		b.resuming = false
		b.mu.Unlock()
	}()
	run, node := ownerRun(st), ""
	if st.Owner != nil {
		node = st.Owner.Node
	}
	b.log.Info("bake: resuming an interrupted merge", "run", run, "node", node, "from", from)
	began := time.Now()
	ex, err := b.resumeExclusive(ctx, dir, run)
	if err != nil {
		_ = lock.Release()
		if ctx.Err() == nil {
			b.retryLater("resume failed; trying again in 10m", run, err)
		}
		return
	}
	result, err := b.completeMerge(dir, lock, st)
	_ = ex.Release()
	_ = lock.Release()
	if err != nil {
		b.retryLater("resume failed; trying again in 10m", run, err)
		return
	}
	b.mu.Lock()
	b.lastErr, b.retryAt = "", time.Time{}
	b.mu.Unlock()
	b.log.Info("bake: resumed the merge of run "+run, "run", run, "ops", result.Ops, "added", result.Added,
		"replaced", result.Replaced, "discarded", result.Discarded, "took", time.Since(began).Round(time.Millisecond))
}

// resumeExclusive 는 재개의 배타다 — 줄은 노드 로그로 간다 (마감 줄 대신 재개의 줄 · business-rules.md 8.6). 그물이
// 찾으면 놓고 foreignRetry 뒤 다시 기다린다. 찾은 것은 실패가 아니다 — 간격을 걸지 않는다.
func (b *Baker) resumeExclusive(ctx context.Context, dir *lower.Dir, run string) (*lower.Exclusive, error) {
	wl := &waitLog{resume: run, every: waitLogEvery}
	for {
		ex, err := dir.Exclusive(ctx, mergeWatchEvery, func(wt lower.Waiting) {
			for _, line := range wl.lines(b.now(), wt) {
				b.log.Info(line, "run", run)
			}
		})
		if err != nil {
			return nil, err
		}
		scan, err := foreignMounts(dir.Root)
		if err != nil {
			b.log.Warn("cannot scan mounts of other processes; merging on the lower lock alone", "run", run, "err", err)
			return ex, nil
		}
		if len(scan.Found) == 0 {
			return ex, nil
		}
		_ = ex.Release()
		for _, line := range foundLines(scan, foreignRetry) {
			b.log.Warn(line, "run", run)
		}
		if err := sleep(ctx, foreignRetry); err != nil {
			return nil, err
		}
	}
}

// completeMerge 는 12.3 의 3 ~ 11 이다. 끝났는지를 먼저 본다 (12.4) — run_id 한 칸이 아니라 초안의 synced_at · head
// 까지 댄다. 시작 전 확인이 어긋나도 upper 를 버리지 않는다 — lower 가 반쯤 바뀌었을 수 있다 (FD 규칙 10절 표).
func (b *Baker) completeMerge(dir *lower.Dir, lock *lower.Bake, st lower.State) (merge.Result, error) {
	var result merge.Result
	run := ownerRun(st)
	scratchDir, ok := pendingShape(st.PendingUpper, dir.Root.Key.String())
	if !ok {
		return result, errors.New(oddPendingText + ": " + st.PendingUpper)
	}
	pending, upper := filepath.Dir(st.PendingUpper), st.PendingUpper
	d, err := readDraft(pending, b.uid)
	if err != nil {
		// 초안은 pending 보다 먼저 fsync 로 디스크에 있고 committed 뒤에만 옮긴다 — 없으면 사람이 봐야 한다
		return result, err
	}
	md, _ := lower.ReadMetadata(b.lowerRoot)
	if !finished(md, run, d) {
		switch _, err := os.Lstat(upper); {
		case err == nil:
			trash := scratch.TrashIn(scratchDir).Dir
			if err := os.MkdirAll(trash, 0o700); err != nil {
				return result, fmt.Errorf("cannot create the trash of the pending upper: %w", err)
			}
			req := mergeHelperRequest{Op: "preflight", Upper: upper, Lower: b.lowerRoot, Trash: trash}
			if _, err := callMergeHelper(req); err != nil {
				return result, err
			}
			req.Op = "apply"
			if result, err = callMergeHelper(req); err != nil {
				return result, err
			}
			if err := os.Remove(upper); err != nil {
				return result, err
			}
		case !errors.Is(err, fs.ErrNotExist):
			return result, err
		}
		// Apply 와 rmdir 이 끝났다 — metadata 부터 (12.4). bake.node 는 합친 노드가 아니라 구운 노드다 (12.5)
		if err := writeMetadata(b.lowerRoot, metadataOf(d, run, d.Node, b.now(), true)); err != nil {
			return result, err
		}
	}
	if err := writeLowerState(lock, lower.State{Phase: lower.PhaseCommitted, Since: b.now().UTC()}); err != nil {
		return result, fmt.Errorf("cannot write the lower state: %w", err)
	}
	if err := movePending(pending); err != nil {
		// 합치기는 끝났고 committed 도 썼다 — 남은 자리는 버려진 것이다 (12.2)
		b.log.Warn("bake: cannot move the pending upper to trash; it is left for the start-up cleanup",
			"path", pending, "err", err)
	}
	return result, nil
}
