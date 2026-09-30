package enode

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/taeels/enode/internal/lower"
)

// 노드의 굽기다 (bake 유닛 · ADR-077 · FD 엔티티 2 · 3 · 10절). build · merge 단계 · 기동 정리 · 광고 주기의
// 정리와 재개가 이 파일의 Baker 를 나눠 쓴다. 규칙은 bakerule.go 의 순수 함수이고 파일 일은 bakefile.go 다.
//
// 잠금의 차례 (계획 4.1 12번) — heldBake.mu 를 먼저, Baker.mu 를 나중에 잡는다. LowerGuard.mu 다음에 Baker.mu
// 인 차례가 하나 있다 (guard 가 g.mu 아래에서 onStale 을 부른다). 그 반대 — Baker.mu 나 heldBake.mu 를 쥔 채
// g.mu 를 잡는 것 (guard.Dir · HoldBake · DropBake) — 은 없다. abandon 은 Baker.mu 를 쥔 채 부르지 않는다.

// 시험이 바꾸는 간격들 (계획 4.1 11번). 바꾸는 시험은 t.Parallel 을 안 쓰고 t.Cleanup 으로 되돌린다.
var (
	// mergeWatchEvery 는 배타를 기다리는 동안 쥔 쪽을 보는 간격이다 (Exclusive 의 every).
	mergeWatchEvery = 10 * time.Second
	// waitLogEvery 는 쥔 쪽의 모습이 그대로일 때 대기 줄을 다시 쓰는 간격이다.
	waitLogEvery = 5 * time.Minute
	// foreignRetry 는 그물이 다른 namespace 의 마운트를 찾았을 때 배타를 놓고 다시 기다리기까지다.
	foreignRetry = 60 * time.Second
	// staleRetry 는 재개나 광고 주기의 정리가 실패한 뒤 이 노드가 다시 해 보기까지다.
	staleRetry = 10 * time.Minute
	// beforeStartMerging 은 merge 단계의 시작 전 확인과 startMerging 사이의 시험 훅이다. 제품은 nil 이다.
	beforeStartMerging func()
	// writeLowerState 는 state.json 쓰기다. root 로 도는 시험이 쓰기 실패를 끼운다 — 자리 디렉터리 0500 이 root 를
	// 막지 못한다.
	writeLowerState = func(b *lower.Bake, st lower.State) error { return b.WriteState(st) }
)

// 낡은 상태를 치우거나 재개를 여는 쪽이다 — 노드 로그의 from.
const (
	fromStart  = "start"
	fromAdvert = "advert"
	fromBuild  = "build claim"
)

// Baker 는 이 노드의 굽기다 — build · merge 단계 · 기동 정리 · 재개. Worker(단계)와 LowerGuard(광고 주기)가 나눠
// 쓴다. runc-overlay 노드에만 있다 (LowerGuard 와 같은 조건). 그 밖의 노드는 nil 이고 굽기 단계를 거절한다.
//
// lower 뿌리의 파일 일 (metadata 읽기와 쓰기 · merge-helper 요청의 Lower) 은 설정의 ws 글자가 아니라 lowerRootOf 로
// 한다 — 아래 lowerRootOf 의 주석.
type Baker struct {
	ctx      context.Context // 데몬의 ctx — 배경 재개가 쓴다
	scratch  string
	node     string
	label    string
	instance string
	uid      int // 노드 uid — 초안의 주인
	log      *slog.Logger
	now      func() time.Time
	guard    *LowerGuard

	mu       sync.Mutex
	held     *heldBake // 이 프로세스가 쥔 굽기 (build · merge 단계) — 노드마다 임대 하나 (I1)
	resuming bool      // 배경 일 (정리 · 재개) 이 굽기 잠금을 쥐고 도는 중 — 한 번에 하나
	retryAt  time.Time // 재개나 광고 주기의 정리가 실패한 뒤 이 노드가 다시 해 볼 때
	lastErr  string    // 마지막 재개 · 정리 오류 — 바뀔 때만 로그
	closed   bool      // Wait 가 불렸다 — 배경 일을 더 열지 않는다
	wg       sync.WaitGroup
}

// StartBaker 는 노드 기동의 한 줄이다 (cmd/enode/main.go · StartLowerGuard 뒤 · 광고 시작 전). guard 가 nil 이면
// nil 이다.
//
// 첫 줄에서 guard.OnStale(b.onStale) 를 등록한다 — 굽기 잠금을 못 잡아도, 자리를 못 열어도 광고 주기의 정리와
// 재개는 열려 있어야 한다. 그다음 굽기 잠금을 해 보고, 되면 낡은 상태를 정리하거나 재개를 배경에 연다
// (business-rules.md 12절). 안 되면 다른 노드가 쥐었다 — 그쪽이 한다.
func StartBaker(ctx context.Context, guard *LowerGuard, scratchDir string, ident Identity, instance string,
	log *slog.Logger) *Baker {
	if guard == nil {
		return nil
	}
	b := &Baker{ctx: ctx, scratch: scratchDir, node: ident.NodeID, label: ident.Label,
		instance: instance, uid: os.Getuid(), log: log, now: time.Now, guard: guard}
	guard.OnStale(b.onStale)
	b.startUp()
	return b
}

// lowerRootOf 는 굽기가 lower 뿌리의 파일을 다루는 경로다 — 상태 자리를 연 때 (LowerGuard 가 lower.ReadRoot 로)
// symlink 를 푼 진짜 경로 (lower.Dir 의 Root.Path) 다. 설정의 ws 글자를 쓰지 않는다: 워크스페이스가 lower 를
// 가리키는 symlink 인 노드에서 lower.WriteMetadata 는 뿌리를 O_NOFOLLOW 로 열어 fsync 하므로 합친 뒤에 ENOTDIR 로
// 멈춘다 (조각 7 에서 찾았다). 상태 자리의 키 · 그물 (ForeignMounts) · merge-helper 의 시작 전 확인이 모두 푼 경로를
// 보므로 한 lower 를 한 글자로 가리킨다 — 두 번 풀지 않으므로 그 사이에 symlink 가 바뀌어도 키와 파일이 다른 lower 를
// 보지 않는다.
// 자리를 못 열었으면 (뿌리를 못 풀었으면) 굽기 흐름은 dir 이 nil 인 갈래로 가서 이 함수에 닿지 않는다.
func lowerRootOf(dir *lower.Dir) string {
	return dir.Root.Path
}

// startUp 은 기동 정리다 (business-rules.md 12.1 · 12.2).
func (b *Baker) startUp() {
	dir := b.guard.Dir()
	if dir == nil {
		b.log.Warn("bake: the lower state directory is not open; skipping the start-up cleanup")
		return
	}
	lock, ok, err := dir.TryBake()
	if err != nil {
		b.log.Warn("bake: cannot take the bake lock; skipping the start-up cleanup", "err", err)
		return
	}
	if !ok {
		return // 다른 노드가 쥐었다 — 그쪽이 한다
	}
	st, err := dir.ReadState()
	if err != nil {
		_ = lock.Release()
		b.log.Warn("bake: cannot read the lower state; skipping the start-up cleanup", "err", err)
		return
	}
	if st.Phase == lower.PhaseMerging {
		b.openResume(dir, lock, st, fromStart)
		return
	}
	if _, err := b.clean(dir, lock, st, fromStart); err != nil {
		b.log.Warn("bake: cannot write the lower state while cleaning a stale bake", "err", err)
	}
	_ = lock.Release()
}

// Wait 는 배경 일 (재개 · 광고 주기의 정리) 이 끝나기를 기다린다 — 데몬이 멈출 때 main 이 부른다. 그 뒤에는 배경
// 일을 더 열지 않는다. nil 이면 할 일이 없다.
func (b *Baker) Wait() {
	if b == nil {
		return
	}
	b.mu.Lock()
	b.closed = true
	b.mu.Unlock()
	b.wg.Wait()
}

// clean 은 굽기 잠금을 쥔 채 FD 규칙 12.2 의 표를 한 벌로 돈다 — 기동 · 광고 주기 · build claim 이 부른다. merging
// 줄은 재개라 여기서 하지 않는다. 잠금을 놓지 않는다 — 부르는 쪽이 놓거나 이어 쓴다.
//
//	committed           자기 scratch 의 pending/<키>/ 아래 버려진 자리를 trash 로 — 기동일 때만
//	building · pending  기록된 대기 자리를 그 자리의 scratch 의 trash 로 · 자기 scratch 의 나머지를 자기 trash 로 ·
//	                    committed + last_attempt (abandoned)
//
// 기록된 자리의 모양이 틀리면 옮기지 않고 경로를 로그에 남긴다 — 상태만 committed 다. 옮기지 못해도 committed 를
// 쓴다 (결정 53) — 남은 자리는 그 scratch 를 쓰는 노드의 기동 정리가 거둔다. 돌려주는 것은 쓴 (쓰려던) state 와
// 그 쓰기의 오류다 — build claim 이 building 에 last_attempt 를 옮겨 싣는다.
func (b *Baker) clean(dir *lower.Dir, lock *lower.Bake, st lower.State, from string) (lower.State, error) {
	key := dir.Root.Key.String()
	switch st.Phase {
	case lower.PhaseCommitted:
		if from == fromStart {
			b.sweepOwn(key, "")
		}
		return st, nil
	case lower.PhaseBuilding, lower.PhasePending:
	default:
		return st, nil
	}
	recorded, builds := "", []lower.BuildRecord(nil)
	if st.PendingUpper != "" {
		if _, ok := pendingShape(st.PendingUpper, key); ok {
			recorded = filepath.Dir(st.PendingUpper)
			builds = b.draftBuildsOf(recorded)
			if err := movePending(recorded); err != nil {
				b.log.Warn("bake: cannot move the pending upper to trash; it is left for the start-up cleanup",
					"path", recorded, "err", err)
			}
		} else {
			b.log.Warn("bake: "+oddPendingText+"; leaving it", "path", st.PendingUpper)
		}
	}
	b.sweepOwn(key, recorded)
	now := b.now().UTC()
	run := ""
	if st.Owner != nil {
		run = st.Owner.Run
	}
	next := lower.State{Phase: lower.PhaseCommitted, Since: now,
		LastAttempt: &lower.LastAttempt{Run: run, At: now, Reason: abandonedReason(st.Phase), Builds: builds}}
	b.log.Info("bake: cleaned a stale bake", "run", run, "phase", st.Phase, "path", st.PendingUpper, "from", from)
	return next, writeLowerState(lock, next)
}

// draftBuildsOf 는 정리의 last_attempt 에 실을 builds 다 — 초안이 있으면 그 builds, 없으면 비운다 (계획 4.1 20번).
// 초안이 아직 없는 것 (building) 은 조용히 비우고, 못 읽은 것만 한 줄 남긴다.
func (b *Baker) draftBuildsOf(dir string) []lower.BuildRecord {
	d, err := readDraft(dir, b.uid)
	if err == nil {
		return d.Builds
	}
	if !errors.Is(err, fs.ErrNotExist) {
		b.log.Warn("bake: cannot read the bake draft; last_attempt has no builds", "path", dir, "err", err)
	}
	return nil
}

// sweepOwn 은 자기 scratch 의 pending/<키>/ 아래에서 keep 밖을 자기 trash 로 옮긴다 — 굽기 잠금을 쥔 쪽만 부른다.
func (b *Baker) sweepOwn(key, keep string) {
	if b.scratch == "" {
		return
	}
	moved, err := sweepPending(b.scratch, key, keep)
	for _, p := range moved {
		b.log.Info("bake: cleaned a stale bake", "phase", lower.PhaseCommitted, "path", p)
	}
	if err != nil {
		b.log.Warn("bake: cannot move the pending upper to trash; it is left for the start-up cleanup",
			"path", filepath.Join(b.scratch, pendingDirName, key), "err", err)
	}
}

// openResume 은 쥔 굽기 잠금을 넘겨 재개를 배경에 연다 — 기동 · build claim 이 부른다 (광고 주기는 이미 배경
// 일이라 곧바로 부른다). 기록된 자리 밖의 자기 scratch 의 버려진 자리를 먼저 치운다 (business-rules.md 12.2 의
// merging 줄). 멈추는 중이면 열지 않고 잠금을 놓는다.
func (b *Baker) openResume(dir *lower.Dir, lock *lower.Bake, st lower.State, from string) bool {
	b.sweepOwn(dir.Root.Key.String(), filepath.Dir(st.PendingUpper))
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		_ = lock.Release()
		return false
	}
	b.resuming = true
	b.wg.Go(func() { b.resume(b.ctx, dir, lock, st, from) })
	return true
}

// heldBake 는 이 프로세스가 쥔 굽기 하나다 (FD 엔티티 3절). build 단계가 굽기 잠금을 잡을 때 만들고, committed 를
// 쓰고 잠금을 놓을 때 끝난다. 치우는 몸통이 굽기 하나에 한 번만 돌게 하는 자리다.
type heldBake struct {
	b    *Baker
	run  string // 굽기 Run
	step int    // build 단계의 seq
	dir  *lower.Dir
	// lock 은 bake.lock 이다 — building 부터 committed 까지. 쥔 동안 이 참조를 붙들어 둔다 — os.File 은 참조가
	// 끊기면 GC 의 finalizer 가 fd 를 닫아 flock 이 풀린다 (결정 44).
	lock *lower.Bake

	mu      sync.Mutex
	pending string // 대기 자리 <scratch>/pending/<lower 키>/<이름> — building 때 만든다
	draft   *Draft // pending 을 쓴 뒤에 있다
	merging bool   // 「합치기 시작」 표지 — merge 단계가 merging 을 쓰기 전에 mu 아래에서 적는다
	done    bool   // 몸통이 돌았거나 놓았다 — 더 할 일이 없다
}

// abandon 은 치우는 몸통이다 (business-rules.md 11절). 몇 번 불러도 한 번만 돈다. merging 표지가 있으면 아무것도
// 안 한다 — 합치기 본체를 끊지 않는다. 차례는 옮기기 · committed + last_attempt · 굽기 Release · DropBake · held
// 비움이다. 쓰기가 실패하면 할 수 있는 데까지 한다 — 옮기기가 실패해도 committed 를 쓰고, committed 가 실패해도
// 잠금은 놓는다 (결정 28). Baker.mu 를 쥔 채 부르지 않는다.
func (h *heldBake) abandon(reason string, builds []lower.BuildRecord) {
	h.mu.Lock()
	if h.done || h.merging {
		h.mu.Unlock()
		return
	}
	h.done = true
	pending := h.pending
	h.mu.Unlock()

	b := h.b
	if pending != "" {
		if err := movePending(pending); err != nil {
			b.log.Warn("bake: cannot move the pending upper to trash; it is left for the start-up cleanup",
				"path", pending, "err", err)
		}
	}
	now := b.now().UTC()
	st := lower.State{Phase: lower.PhaseCommitted, Since: now,
		LastAttempt: &lower.LastAttempt{Run: h.run, At: now, Reason: reason, Builds: builds}}
	if err := writeLowerState(h.lock, st); err != nil {
		b.log.Warn("bake: cannot write the lower state while discarding the bake", "run", h.run, "err", err)
	}
	b.log.Info("bake: discarded the bake", "run", h.run, "reason", reason)
	h.letGo()
}

// letGo 는 굽기를 놓는다 — 굽기 Release · DropBake · held 비움. 몸통 · 합침 · merging 뒤 멈춤 · 7.1 의 어긋남이 부른다.
// 상태를 쓰지 않는다.
func (h *heldBake) letGo() {
	h.mu.Lock()
	h.done = true
	h.mu.Unlock()
	_ = h.lock.Release()
	h.b.guard.DropBake()
	h.b.mu.Lock()
	if h.b.held == h {
		h.b.held = nil
	}
	h.b.mu.Unlock()
}

// draftBuilds 는 쥔 초안의 builds 다 — HoldBake 에 넘기는 몸통이 last_attempt 에 싣는다.
func (h *heldBake) draftBuilds() []lower.BuildRecord {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.draft == nil {
		return nil
	}
	return h.draft.Builds
}

// startMerging 은 merge 단계가 merging 을 쓰기 직전에 부른다. 몸통이 먼저 돌았으면 false — merge 단계는 거기서
// 멈춘다. true 면 그 뒤의 몸통은 아무것도 안 한다.
func (h *heldBake) startMerging() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.done {
		return false
	}
	h.merging = true
	return true
}

// cancelMerging 은 merging 을 못 쓴 merge 단계가 표지를 거둔다 — 합치기 전이다. 거두지 않으면 몸통이 표지를 보고
// 아무것도 안 해 잠금과 held 가 남는다 (계획 4.1 13번).
func (h *heldBake) cancelMerging() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.merging = false
}

// setPending · setDraft 는 build 단계가 자리와 초안을 적는다 — 몸통이 다른 고루틴에서 읽는다.
func (h *heldBake) setPending(dir string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pending = dir
}

func (h *heldBake) setDraft(d Draft) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.draft = &d
}

func (h *heldBake) pendingDir() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.pending
}

func (h *heldBake) currentDraft() *Draft {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.draft
}
