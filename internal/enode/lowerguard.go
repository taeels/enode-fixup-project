package enode

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/lower"
)

// LowerGuard 는 이 노드의 lower 공유 잠금을 쥐고 놓는다 (lower-state 유닛 · ADR-077 §6 · business-rules.md 5절).
//
// 원칙 — 매칭된 순간부터 그 노드가 그 Run 의 임대를 놓을 때까지 lower 가 안 바뀐다. 이 lower 를 마운트하는 모든
// 자리가 공유를 쥐므로, 배타를 잡은 merge 는 「이 lower 의 overlay 마운트 0」을 안다. 지키지 못하는 경로가 생기면
// 그 Run 의 단계를 돌리지 않는다 — 바뀐 lower 위에서 조용히 돌지 않는다.
//
// 광고 주기(Advertiser)와 claim(Worker)이 나눠 쓴다. 둘은 다른 고루틴이라 mutex 하나로 묶는다. 잡은 동안 하는
// 일은 파일 호출뿐이다 (µs 에서 1 ms 아래). 막히는 호출(배타 대기)은 하지 않는다. 합치기 없이 끝난 굽기의 몸통은
// 따로 도는 고루틴이다. runc-overlay 노드만 만든다 — 그 밖의 노드는 nil 이고 nil 은 아무것도 안 한다.
type LowerGuard struct {
	mu     sync.Mutex
	lowers string // 상태 자리의 뿌리
	path   string // lower 루트 — 노드 설정의 워크스페이스
	node   string
	label  string
	log    *slog.Logger
	now    func() time.Time

	homeErr error      // home 을 못 찾았다 — 자리를 열 수 없다
	dir     *lower.Dir // nil 이면 못 열었다 — 광고 주기마다 다시 연다
	openErr error

	shared *lower.Shared // 쥐고 있나
	role   lower.Role
	run    string
	since  time.Time
	streak int // drain 이 받아 적히고 임대가 0 인 응답이 연속 몇 번 (답 1)

	released bool         // 놓았다 — 이 뒤에 보인 임대는 늦게 보인 임대다 (5.4)
	mark     lower.Marker // 놓을 때의 metadata 표지
	everHeld bool

	refused  map[string]bool // lower_changed 로 실패시킬 Run.  그 Run 의 임대가 사라지면 지운다
	baking   map[string]bool // prepare · merge 단계를 claim 한 Run — 늦게 보인 임대로 치지 않는다
	stepping int             // 도는 단계 수 — 0 이 아니면 놓지 않는다
	bake     *bakeHold       // 이 프로세스가 쥔 굽기 잠금의 Run 과 치우는 몸통 (bake 유닛이 등록)

	// 로그는 원인이나 phase 가 바뀔 때만 쓴다
	warnOpen   string
	warnPhase  lower.Phase
	warnBusy   string
	warnMeta   string
	warnDrop   string
	warnRecord string
}

type bakeHold struct {
	run     string
	abandon func()
}

// LowerChangedError 는 놓은 뒤에 보인 임대의 Run 을 이 노드가 돌리지 않는다는 뜻이다 (business-rules.md 5.4).
// Worker 가 그 단계를 reason lower_changed 로 보고한다 — 다시 내면 된다.
type LowerChangedError struct {
	Run string
}

func (e *LowerChangedError) Error() string {
	return "the lower changed after this run was matched on this node; resubmit the run"
}

// StartLowerGuard 는 노드 기동의 한 줄이다 (FD 흐름 7절 2 · 3 · 5). lower 루트가 없으면(runc-overlay 가 아닌 노드)
// nil 이다. 상태 자리를 열고 「놓은 상태」로 시작한다 — 그때의 metadata 표지를 적는다. 첫 광고가 공유를 쥔다.
// 자리를 못 열어도 nil 이 아니다 — 오류를 들고 광고마다 다시 연다.
func StartLowerGuard(lowerRoot string, ident Identity, log *slog.Logger) *LowerGuard {
	if lowerRoot == "" {
		return nil
	}
	lowers, err := LowersDir()
	g := newLowerGuard(lowers, lowerRoot, ident, log)
	g.homeErr = err
	g.start()
	return g
}

func newLowerGuard(lowers, lowerRoot string, ident Identity, log *slog.Logger) *LowerGuard {
	return &LowerGuard{lowers: lowers, path: lowerRoot, node: ident.NodeID, label: ident.Label, log: log,
		now: time.Now, refused: map[string]bool{}, baking: map[string]bool{}}
}

// start 는 자리를 열고 놓은 상태로 둔다 (business-rules.md 5.5).
func (g *LowerGuard) start() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.openLocked()
	g.released, g.mark = true, g.markerLocked()
}

// once 는 원인이 바뀔 때만 쓴다. 빈 원인은 「지금은 괜찮다」이고 기억만 되돌린다.
func once(last *string, reason string, write func()) {
	if *last == reason {
		return
	}
	*last = reason
	if reason != "" {
		write()
	}
}

// openLocked 는 자리를 연다 — 못 열었으면 다시 해 본다. Open 이 고친 것(좁힌 권한 · 새로 쓴 신원)은 한 줄씩 남긴다.
func (g *LowerGuard) openLocked() {
	if g.dir != nil || g.homeErr != nil {
		return
	}
	root, err := lower.ReadRoot(g.path)
	if err == nil {
		var d *lower.Dir
		if d, err = lower.Open(g.lowers, root, g.now()); err == nil {
			g.dir, g.openErr = d, nil
			for _, note := range d.Notes {
				g.log.Info("lower state directory fixed on open", "note", note)
			}
			return
		}
	}
	g.openErr = err
	g.warnOpenLocked(err)
}

func (g *LowerGuard) warnOpenLocked(err error) {
	once(&g.warnOpen, err.Error(), func() {
		g.log.Warn("cannot open the lower state; draining this node", "dir", g.lowers, "err", err)
	})
}

// lowerSource 는 상태 자리를 못 쓰는 노드의 drain 출처다 (business-rules.md 9절). bake 로 적지 않는다 —
// 제어판의 「합치기가 끝나면」이 거짓이 된다.
func lowerSource(detail string) DrainSource {
	return DrainSource{Kind: DrainLower, Mode: contract.DrainGraceful, Detail: detail}
}

// bakeSource 는 이 lower 의 굽기가 거는 drain 출처다. 소유자가 풀 수 없다.
func bakeSource(detail string) DrainSource {
	return DrainSource{Kind: DrainBake, Mode: contract.DrainGraceful, Detail: detail}
}

// runSuffix 는 문구 끝의 (run <Run>) 이다. 주인이 없으면 비운다.
func runSuffix(st lower.State) string {
	if st.Owner == nil || st.Owner.Run == "" {
		return ""
	}
	return " (run " + st.Owner.Run + ")"
}

// BeforeAdvert 는 광고 직전이다 (FD 흐름 2절). others 는 소유자 · 여유 부족 출처다. 이 노드가 더할 출처(bake ·
// lower)와 광고 키(ir · repo.built.* · bake.*)를 돌려준다. 실을 drain 이 없고 공유를 안 쥐었으면 여기서 쥔다.
// 공유를 먼저 정하고 metadata 를 뒤에 읽는다 — 공유를 쥔 뒤에 읽은 ir 은 그 공유를 놓을 때까지 안 바뀐다.
func (g *LowerGuard) BeforeAdvert(others []DrainSource) (own []DrainSource, keys map[string]string) {
	if g == nil {
		return nil, nil
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	own = g.sourcesLocked()
	all := append(append([]DrainSource(nil), others...), own...)
	// HoldBake 부터 DropBake 까지는 쥐지 않는다 (계획 4절 ⑪) — 한 프로세스의 fd 둘이 부딪쳐 자기 합치기를 막는다
	if combineDrain(all) == contract.DrainNone && g.shared == nil && g.bake == nil && g.dir != nil {
		ok, err := g.takeLocked(lower.RoleCandidate, "")
		switch {
		case err != nil:
			own = append(own, lowerSource("cannot open the lower state: "+err.Error()))
		case !ok:
			own = append(own, bakeSource("lower is being merged"))
		case g.everHeld:
			g.log.Info("took the lower lock again", "role", g.role)
		}
		g.everHeld = g.everHeld || ok
	}
	g.syncLocked()
	return own, g.keysLocked()
}

// sourcesLocked 는 이 노드가 스스로 거는 drain 출처다 — 자리를 못 열었다 · state.json 을 못 읽었다 · 굽기가
// pending 이나 merging 이다. 형제는 state.json 을 먼저 읽는다 — pending · merging 이면 공유를 새로 안 잡는다.
// 기다리는 배타에 우선권이 없어서다 (FD 계획 2.1 셋째 줄).
func (g *LowerGuard) sourcesLocked() []DrainSource {
	if g.homeErr != nil {
		return []DrainSource{lowerSource("cannot find the home directory: " + g.homeErr.Error())}
	}
	g.openLocked()
	if g.dir == nil {
		return []DrainSource{lowerSource("cannot open the lower state: " + g.openErr.Error())}
	}
	st, err := g.dir.ReadState()
	if err != nil {
		g.warnOpenLocked(err)
		return []DrainSource{lowerSource("cannot open the lower state: " + err.Error())}
	}
	g.warnOpen = ""
	if st.Phase != g.warnPhase {
		g.warnPhase = st.Phase
		switch st.Phase {
		case lower.PhasePending:
			g.log.Info("the lower is pending a merge; draining this node", "run", runOf(st))
		case lower.PhaseMerging:
			g.log.Info("the lower lock is held by a merge; draining this node", "run", runOf(st))
		}
	}
	switch st.Phase {
	case lower.PhasePending:
		return []DrainSource{bakeSource("lower pending a merge" + runSuffix(st))}
	case lower.PhaseMerging:
		return []DrainSource{bakeSource("lower being merged" + runSuffix(st))}
	}
	return nil
}

func runOf(st lower.State) string {
	if st.Owner == nil {
		return ""
	}
	return st.Owner.Run
}

// takeLocked 는 공유를 잡는다. 배타가 쥐어져 있으면(합치는 중) false 다.
func (g *LowerGuard) takeLocked(role lower.Role, run string) (bool, error) {
	now := g.now().UTC()
	s, ok, err := g.dir.TryShared(lower.Holder{Node: g.node, Label: g.label, Role: role, Run: run, Since: now})
	if err != nil {
		g.warnOpenLocked(err)
		return false, err
	}
	if !ok {
		once(&g.warnBusy, "busy", func() { g.log.Info("the lower lock is held by a merge; draining this node") })
		return false, nil
	}
	g.warnBusy = ""
	g.shared, g.role, g.run, g.since, g.streak, g.released = s, role, run, now, 0, false
	if !s.Recorded() {
		once(&g.warnRecord, g.node, func() {
			g.log.Warn("holding the lower lock without a holder record; a waiting merge cannot name this node", "node", g.node)
		})
	}
	return true, nil
}

// setRoleLocked 는 기록의 역할을 바꾼다 — 바뀌면 since 도 새로.
func (g *LowerGuard) setRoleLocked(role lower.Role, run string) {
	if role != g.role || run != g.run {
		g.role, g.run, g.since = role, run, g.now().UTC()
	}
}

// syncLocked 는 쥔 사람 기록을 맞춘다 — 역할 · Run · acks. 바뀐 칸이 없으면 안 쓴다.
func (g *LowerGuard) syncLocked() {
	if g.shared == nil {
		return
	}
	h := lower.Holder{Node: g.node, Label: g.label, Role: g.role, Run: g.run, Since: g.since, Acks: g.streak}
	if err := g.shared.Update(h); err != nil {
		once(&g.warnRecord, err.Error(), func() { g.log.Warn("cannot update the lower holder record", "err", err) })
	}
}

// releaseLocked 는 공유를 놓고 그때의 표지를 적는다 (5.2 · 5.3).
func (g *LowerGuard) releaseLocked() {
	if g.shared != nil {
		_ = g.shared.Release()
		g.shared = nil
	}
	g.released, g.mark, g.streak, g.role, g.run = true, g.markerLocked(), 0, "", ""
}

// markerLocked 는 지금 metadata 의 표지다. 못 읽으면 제로값이다 — 놓을 때 읽혔는데 지금 안 읽히면 다르다고 본다.
func (g *LowerGuard) markerLocked() lower.Marker {
	md, err := lower.ReadMetadata(g.path)
	if err != nil {
		return lower.Marker{}
	}
	return md.Marker()
}

// keysLocked 는 metadata 에서 광고 키를 짓는다 (business-rules.md 11절). 못 읽으면 네 키를 다 안 싣고 원인이
// 바뀔 때 한 줄 남긴다 — 노드는 계속 일한다. pending · merging 동안에는 옛 metadata 를 그대로 싣는다.
func (g *LowerGuard) keysLocked() map[string]string {
	md, err := lower.ReadMetadata(g.path)
	if err != nil {
		once(&g.warnMeta, err.Error(), func() {
			g.log.Warn("cannot read .enode-metadata.json; not advertising ir, repo.built and bake keys", "err", err)
		})
		return nil
	}
	g.warnMeta = ""
	keys, dropped := metadataKeys(md)
	once(&g.warnDrop, strings.Join(dropped, "; "), func() {
		g.log.Warn("ignoring values in .enode-metadata.json; not advertising those keys", "why", strings.Join(dropped, "; "))
	})
	return keys
}

// AfterResponse 는 광고 응답 뒤다 (FD 흐름 2절). 광고가 실패해 응답이 없으면 부르지 않는다 — 본 것이 없으므로
// 셈을 그대로 둔다 (5.2).
//
//	임대가 있다      쥐었으면 역할 run.  안 쥐었으면 prepare 가 아닌 Run 의 임대는 늦게 보인 임대다 (5.4)
//	임대가 0 이다    drain 이 받아 적혔으면 셈을 하나 올린다.  둘이 되고 도는 단계가 없으면 놓는다 (표지를 적는다).
//	                 쥔 채면 역할을 candidate 로 (계획 4절 ⑫)
//	굽기 잠금       쥐었고 pending 인데 그 Run 의 임대가 없다 — 치우는 몸통을 한 번 부른다 (business-rules.md 10절)
func (g *LowerGuard) AfterResponse(drain string, leases []Lease) {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	present := map[string]bool{}
	for _, l := range leases {
		present[l.RunID] = true
	}
	for run := range g.refused {
		if !present[run] {
			delete(g.refused, run)
		}
	}
	for run := range g.baking {
		if !present[run] {
			delete(g.baking, run)
		}
	}
	var runs []string // prepare 가 아닌 Run 의 임대
	for _, l := range leases {
		if !g.baking[l.RunID] && (g.bake == nil || g.bake.run != l.RunID) && !contains(runs, l.RunID) {
			runs = append(runs, l.RunID)
		}
	}
	switch {
	case len(leases) > 0:
		g.streak = 0
	case drain != "":
		g.streak++
	default:
		g.streak = 0
	}
	switch {
	case g.shared != nil && len(runs) > 0:
		g.setRoleLocked(lower.RoleRun, runs[0])
	case g.shared != nil && len(leases) == 0 && g.streak >= 2 && g.stepping == 0:
		g.releaseLocked()
		g.log.Info("released the lower lock: drain acknowledged twice and no lease",
			"mark_run", g.mark.Run, "mark_merged_at", g.mark.MergedAt)
	case g.shared != nil && len(leases) == 0:
		g.setRoleLocked(lower.RoleCandidate, "")
	case g.shared == nil && g.bake == nil:
		for _, run := range runs {
			if !g.refused[run] && g.lateLocked(run) == nil {
				break
			}
		}
	}
	g.syncLocked()
	g.bakeGoneLocked(present)
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// lateLocked 는 놓은 뒤에 보인 임대다 (business-rules.md 5.4).
//
//	state.json 이 merging                  거절 — lower 가 반쯤 바뀌었을 수 있다
//	metadata 표지가 놓을 때와 다르다        거절 — 그 사이 합쳐졌다
//	그 밖 (committed · building · pending)  공유를 잡는다 (역할 run).  pending 이어도 잡는다 — merge 는 그 Run 을 더 기다린다
//	  못 잡았다 (배타가 쥐어져 있다)        거절
//
// 자리를 못 열었거나 state.json 을 못 읽으면 거절하지 않고 오류다 — lower 가 바뀐 것을 안 것이 아니다.
func (g *LowerGuard) lateLocked(run string) error {
	if g.homeErr != nil {
		return fmt.Errorf("cannot open the lower state directory: %w", g.homeErr)
	}
	g.openLocked()
	if g.dir == nil {
		return fmt.Errorf("cannot open the lower state directory: %w", g.openErr)
	}
	st, err := g.dir.ReadState()
	if err != nil {
		return fmt.Errorf("cannot open the lower state directory: %w", err)
	}
	why := ""
	switch {
	case st.Phase == lower.PhaseMerging:
		why = "the lower is being merged"
	case g.markerLocked() != g.mark:
		why = "the lower was merged after the lock was released"
	default:
		ok, err := g.takeLocked(lower.RoleRun, run)
		if err != nil {
			return fmt.Errorf("cannot open the lower state directory: %w", err)
		}
		if ok {
			g.everHeld = true
			g.log.Info("a lease arrived after the lower lock was released; taking it again", "run", run)
			return nil
		}
		why = "the lower lock is held by a merge"
	}
	g.refused[run] = true
	g.log.Warn("refusing a run: the lower changed after it was matched", "run", run, "why", why)
	return &LowerChangedError{Run: run}
}

// bakeGoneLocked 는 합치기 없이 끝난 굽기다 (business-rules.md 10절). building 이면 build 단계가 끝에서 치우고,
// merging 이면 합치기 본체를 끊지 않는다 — pending 일 때만 부른다. 광고 루프를 막지 않게 따로 돈다.
func (g *LowerGuard) bakeGoneLocked(present map[string]bool) {
	if g.bake == nil || present[g.bake.run] || g.dir == nil {
		return
	}
	st, err := g.dir.ReadState()
	if err != nil || st.Phase != lower.PhasePending || st.Owner == nil || st.Owner.Run != g.bake.run {
		return
	}
	g.log.Warn("the bake run's lease is gone while the lower is pending; discarding the upper", "run", g.bake.run)
	abandon := g.bake.abandon
	g.bake = nil
	if abandon != nil {
		go abandon()
	}
}

// OnClaim 은 claim 한 단계를 돌리기 전이다 (FD 흐름 3절). 부른 쪽은 곧바로 StepDone 을 defer 로 건다 — 거절이어도
// 짝을 맞춘다.
//
//	prepare 단계 · merge 단계     공유를 곧바로 놓는다 (결정 3-9).  울타리를 세지 않고 거절하지 않는다
//	거절한 Run                    *LowerChangedError
//	자리를 못 열었다               오류 (cannot open the lower state directory) — 원인 코드를 달지 않는다
//	쥐었다                        역할 run
//	안 쥐었다 (놓은 뒤)            늦게 보인 임대 — 쥐면 nil, 못 쥐면 오류
func (g *LowerGuard) OnClaim(step *Step) error {
	if g == nil {
		return nil
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.stepping++
	if bakeStep(step) {
		g.baking[step.RunID] = true
		if g.shared != nil {
			g.releaseLocked()
		}
		return nil
	}
	if g.refused[step.RunID] {
		return &LowerChangedError{Run: step.RunID}
	}
	if g.shared != nil {
		g.setRoleLocked(lower.RoleRun, step.RunID)
		g.streak = 0
		g.syncLocked()
		return nil
	}
	return g.lateLocked(step.RunID)
}

// bakeStep 은 굽기 단계인가다 — effect 가 prepare 이거나 종류가 build · merge. 둘은 계약이 짝으로 묶는다 (effect.go).
func bakeStep(step *Step) bool {
	return step.Effect == contract.EffectPrepare || step.Kind == contract.KindBuild.String() ||
		step.Kind == contract.KindMerge.String()
}

// StepDone 은 Worker 가 단계의 세션을 닫은 뒤 부른다 — OnClaim 과 짝이다. 놓는 울타리가 도는 단계를 센다.
func (g *LowerGuard) StepDone(*Step) {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.stepping > 0 {
		g.stepping--
	}
}

// HoldBake 는 bake 유닛이 부른다 — 이 프로세스가 굽기 잠금을 쥐었고 pending 을 썼다. abandon 은 합치기 없이 끝났을 때
// 치우는 몸통이다 (upper 를 trash 로 · committed · 굽기 잠금 놓기). 쥔 공유가 있으면 놓는다 — 배타를 기다리기 전에
// 불러야 한다. DropBake 까지 공유를 쥐지 않는다 (계획 4절 ⑪).
func (g *LowerGuard) HoldBake(run string, abandon func()) {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.shared != nil {
		g.releaseLocked()
	}
	g.bake = &bakeHold{run: run, abandon: abandon}
}

// DropBake 는 bake 유닛이 committed 를 쓰고 굽기 잠금을 놓은 뒤 부른다. 이 노드가 바꾼 lower 의 표지를 새로 적는다 —
// 그 사이 이 노드에 붙은 다른 Run 은 없다 (노드마다 임대 하나).
func (g *LowerGuard) DropBake() {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.bake = nil
	if g.shared == nil {
		g.released, g.mark = true, g.markerLocked()
	}
}
