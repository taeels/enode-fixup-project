package store

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/record"
)

// 이 파일은 대기열(ADR-064)이 딛는 저장소 계약을 잰다.
//
// 진짜 Postgres 위에서 돈다 — 재는 것이 advisory lock 아래의 직렬화, 한
// 트랜잭션 안의 넣기와 첫 훑기, FOR UPDATE SKIP LOCKED 의 성질이라 가짜로
// 대신할 수 없다. 판은 obsStore 처럼 이 파일만의 것을 판다.

func qAdvert(nodeID string, attrs map[string]string) contract.Advert {
	return contract.Advert{
		NodeID: nodeID, Label: nodeID + " label",
		Capabilities: []contract.Capability{{Capability: contract.CapabilityAgentReason, Attrs: attrs}},
	}
}

func qContract(attrs map[string]string) contract.Contract {
	return contract.Contract{
		Requires: []contract.Require{{As: "b", Capability: contract.CapabilityAgentReason, Attrs: attrs}},
		Steps:    []contract.Step{{ID: "one", Uses: "b", Agent: map[string]any{}}},
	}
}

func qAdvertise(t *testing.T, st *Store, nodeID string, attrs map[string]string) {
	t.Helper()
	if _, err := st.UpsertAdvert(context.Background(), qAdvert(nodeID, attrs), "p", time.Minute); err != nil {
		t.Fatalf("cannot advertise %s: %v", nodeID, err)
	}
}

// qRunning 은 노드 하나를 쥔 RUNNING Run 이다 — 제출 경로가 만드는 모양 그대로.
func qRunning(t *testing.T, st *Store, runID, nodeID string, attrs map[string]string) {
	t.Helper()
	c := qContract(attrs)
	r := Run{RunID: runID, State: StateRunning, Principal: "p", Contract: c,
		Assigned: []Assigned{{As: "b", Nodes: []NodeRef{{Node: nodeID, Label: nodeID + " label"}}}}}
	grants := []LeaseGrant{{NodeID: nodeID, NotAfter: time.Now().Add(time.Minute), Nonce: "n"}}
	if err := st.CreateRun(context.Background(), r, grants, c.Steps); err != nil {
		t.Fatalf("cannot create running run %s: %v", runID, err)
	}
}

func qEnqueue(t *testing.T, st *Store, runID string, attrs map[string]string) bool {
	t.Helper()
	promoted, err := st.CreateQueuedRun(context.Background(),
		Run{RunID: runID, Principal: "p", Contract: qContract(attrs)})
	if err != nil {
		t.Fatalf("cannot queue %s: %v", runID, err)
	}
	return promoted
}

func qState(t *testing.T, st *Store, runID string) string {
	t.Helper()
	r, err := st.GetRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("cannot read %s: %v", runID, err)
	}
	return r.State
}

func qCount(t *testing.T, st *Store, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := st.pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("count failed: %v", err)
	}
	return n
}

var claude = map[string]string{"harness": "claude"}

// testRecords 는 봉인된 디렉터리를 t.TempDir 이 지울 수 있게 정리를 걸어둔다.
// 봉인은 삭제까지 막는다 — internal/api 의 newRecords 와 같은 모양이다.
func testRecords(t *testing.T) *record.Store {
	t.Helper()
	root := t.TempDir()
	t.Cleanup(func() {
		_ = filepath.Walk(root, func(p string, fi os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if fi.IsDir() {
				return os.Chmod(p, 0o755)
			}
			return os.Chmod(p, 0o644)
		})
	})
	return record.New(root)
}

// QUEUED 는 runs 행 하나다 — 임대 0 · 단계 0 (INVARIANTS §1.1 · Q2 = A).
func TestQueue_AllBusyBecomesQueuedWithNoLeaseAndNoStep(t *testing.T) {
	st := obsStore(t)
	qAdvertise(t, st, "node-a", claude)
	qRunning(t, st, "a", "node-a", claude)

	if qEnqueue(t, st, "b", claude) {
		t.Fatal("b was promoted while node-a is held")
	}
	if s := qState(t, st, "b"); s != StateQueued {
		t.Fatalf("b state=%s, want QUEUED", s)
	}
	if n := qCount(t, st, `SELECT count(*) FROM leases WHERE run_id='b'`); n != 0 {
		t.Fatalf("a queued run holds %d leases, want 0", n)
	}
	if n := qCount(t, st, `SELECT count(*) FROM steps WHERE run_id='b'`); n != 0 {
		t.Fatalf("a queued run has %d steps before promotion, want 0", n)
	}
	r, _ := st.GetRun(context.Background(), "b")
	if r.Reject != nil || len(r.Assigned) != 0 {
		t.Fatalf("a queued run is not a rejection and has no assignment: %+v", r)
	}
}

// 넣기와 첫 훑기가 한 트랜잭션이다 — 노드가 비어 있으면 그 자리에서 RUNNING 이다.
func TestQueue_CreateQueuedRunPromotesAtOnceWhenTheNodeIsFree(t *testing.T) {
	st := obsStore(t)
	qAdvertise(t, st, "node-a", claude)

	if !qEnqueue(t, st, "b", claude) {
		t.Fatal("b was not promoted although node-a is free")
	}
	if s := qState(t, st, "b"); s != StateRunning {
		t.Fatalf("b state=%s, want RUNNING", s)
	}
	if n := qCount(t, st, `SELECT count(*) FROM leases WHERE run_id='b' AND node_id='node-a'`); n != 1 {
		t.Fatalf("promotion granted %d leases on node-a, want 1", n)
	}
	if n := qCount(t, st, `SELECT count(*) FROM steps WHERE run_id='b'`); n != 1 {
		t.Fatalf("promotion created %d steps, want 1", n)
	}
}

// 정산이 임대를 지우면 같은 트랜잭션에서 먼저 온 것이 승격된다 (FIFO · CP2).
func TestQueue_WakeOnSettlePromotesTheOldestFirst(t *testing.T) {
	st := obsStore(t)
	ctx := context.Background()
	qAdvertise(t, st, "node-a", claude)
	qRunning(t, st, "a", "node-a", claude)
	qEnqueue(t, st, "b", claude)
	qEnqueue(t, st, "c", claude)

	// a 를 끝낸다 — claim · report · settle 은 postResult 가 밟는 길 그대로다.
	cl, err := st.ClaimStep(ctx, "node-a", "")
	if err != nil || cl == nil {
		t.Fatalf("cannot claim a's step: %v %v", cl, err)
	}
	if _, err := st.ReportStep(ctx, "a", cl.Seq, "node-a", true, StepResult{}); err != nil {
		t.Fatalf("cannot report a's step: %v", err)
	}
	if state, err := st.SettleIfDone(ctx, "a"); err != nil || state == "" {
		t.Fatalf("a did not settle: %q %v", state, err)
	}

	if s := qState(t, st, "b"); s != StateRunning {
		t.Fatalf("b state=%s, want RUNNING — the oldest waiter goes first", s)
	}
	if s := qState(t, st, "c"); s != StateQueued {
		t.Fatalf("c state=%s, want QUEUED — node-a is held by b now", s)
	}
}

// 맨 앞이 못 가도 뒤의 다른 자원 요구는 간다 — 전체 훑기.
func TestQueue_WakeSkipsTheHeadAndPromotesALaterFit(t *testing.T) {
	st := obsStore(t)
	codex := map[string]string{"harness": "codex"}
	qAdvertise(t, st, "node-a", claude)
	qAdvertise(t, st, "node-b", codex)
	qRunning(t, st, "a", "node-a", claude)
	qRunning(t, st, "x", "node-b", codex)
	qEnqueue(t, st, "b", claude) // 맨 앞 — node-a 를 기다린다
	qEnqueue(t, st, "c", codex)  // 뒤 — node-b 를 기다린다

	if _, err := st.Cancel(context.Background(), "x", "test"); err != nil {
		t.Fatalf("cannot cancel x: %v", err)
	}
	if s := qState(t, st, "c"); s != StateRunning {
		t.Fatalf("c state=%s, want RUNNING — the head must not block a later fit", s)
	}
	if s := qState(t, st, "b"); s != StateQueued {
		t.Fatalf("b state=%s, want QUEUED — node-a is still held", s)
	}
}

// drain 을 건 노드는 승격 후보가 아니다. 풀리면 후보다.
func TestQueue_DrainingNodesAreNotCandidates(t *testing.T) {
	st := obsStore(t)
	ctx := context.Background()
	drained := qAdvert("node-a", claude)
	drained.Policy.Drain = DrainGraceful
	if _, err := st.UpsertAdvert(ctx, drained, "p", time.Minute); err != nil {
		t.Fatal(err)
	}
	if qEnqueue(t, st, "b", claude) {
		t.Fatal("b was promoted onto a draining node")
	}
	if d, _ := st.DrainingNodes(ctx); !d["node-a"] {
		t.Fatalf("DrainingNodes does not list node-a: %v", d)
	}
	if _, err := st.UpsertAdvert(ctx, qAdvert("node-a", claude), "p", time.Minute); err != nil {
		t.Fatal(err)
	}
	promoted, err := st.WakeQueuedNow(ctx)
	if err != nil || len(promoted) != 1 || promoted[0] != "b" {
		t.Fatalf("after the drain release WakeQueuedNow promoted %v (%v), want [b]", promoted, err)
	}
	if s := qState(t, st, "b"); s != StateRunning {
		t.Fatalf("b state=%s, want RUNNING", s)
	}
}

// 취소는 QUEUED 도 닫는다 — FAILED · cancelled · 임대 0. 단계 0 인 Record 가 봉인된다.
func TestQueue_CancelOfAQueuedRunSealsWithZeroSteps(t *testing.T) {
	st := obsStore(t)
	ctx := context.Background()
	st.Records = testRecords(t)
	qAdvertise(t, st, "node-a", claude)
	qRunning(t, st, "a", "node-a", claude)
	qEnqueue(t, st, "b", claude)

	state, err := st.Cancel(ctx, "b", "someone")
	if err != nil || state != StateFailed {
		t.Fatalf("cancel of a queued run: %q %v, want FAILED", state, err)
	}
	r, _ := st.GetRun(ctx, "b")
	if r.Verdict == nil || len(r.Verdict.Checks) == 0 || r.Verdict.Checks[0].What != "cancelled" {
		t.Fatalf("the cancel reason was not recorded: %+v", r.Verdict)
	}
	if !st.Records.Sealed("b") {
		t.Fatal("a cancelled queued run was not sealed — why it never ran must be in the Record")
	}
	if s := qState(t, st, "a"); s != StateRunning {
		t.Fatalf("cancelling b touched a: %s", s)
	}
}

// 만료 회수는 임대가 있는 Run 만 본다 — QUEUED 를 건드리지 않는다. 회수는 큐를 깨운다.
func TestQueue_ReapLeavesQueuedRunsAloneAndWakes(t *testing.T) {
	st := obsStore(t)
	ctx := context.Background()
	qAdvertise(t, st, "node-a", claude)
	qRunning(t, st, "a", "node-a", claude)
	qEnqueue(t, st, "b", claude)

	if err := st.ForceExpire(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	n, err := st.Reap(ctx, nil)
	if err != nil || n != 1 {
		t.Fatalf("reap: %d %v, want 1 reclaimed", n, err)
	}
	if s := qState(t, st, "a"); s != StateFailed {
		t.Fatalf("a state=%s, want FAILED", s)
	}
	if s := qState(t, st, "b"); s != StateRunning {
		t.Fatalf("b state=%s, want RUNNING — reaping freed node-a", s)
	}
}

// SettleIfDone 은 QUEUED 를 정산하지 않는다 — 단계 0 이 「전부 끝났다」로 읽히면 안 된다.
func TestQueue_SettleIfDoneIgnoresAQueuedRun(t *testing.T) {
	st := obsStore(t)
	ctx := context.Background()
	qAdvertise(t, st, "node-a", claude)
	qRunning(t, st, "a", "node-a", claude)
	qEnqueue(t, st, "b", claude)

	state, err := st.SettleIfDone(ctx, "b")
	if err != nil || state != "" {
		t.Fatalf("SettleIfDone on a queued run returned %q %v, want no settlement", state, err)
	}
	if s := qState(t, st, "b"); s != StateQueued {
		t.Fatalf("b state=%s, want QUEUED untouched", s)
	}
}

// 넣기와 해제가 동시에 와도 Run 이 서지 않는다 — 잠금이 둘을 줄 세운다.
func TestQueue_ConcurrentEnqueueAndReleaseNeverStrands(t *testing.T) {
	st := obsStore(t)
	ctx := context.Background()
	qAdvertise(t, st, "node-a", claude)

	for i := 0; i < 8; i++ {
		holder := "a" + string(rune('0'+i))
		waiter := "b" + string(rune('0'+i))
		qRunning(t, st, holder, "node-a", claude)

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			if _, err := st.CreateQueuedRun(ctx, Run{RunID: waiter, Principal: "p", Contract: qContract(claude)}); err != nil {
				t.Errorf("cannot queue %s: %v", waiter, err)
			}
		}()
		go func() {
			defer wg.Done()
			if _, err := st.Cancel(ctx, holder, "test"); err != nil {
				t.Errorf("cannot cancel %s: %v", holder, err)
			}
		}()
		wg.Wait()

		// 어느 순서로 갔든 끝에는 waiter 가 도는 중이어야 한다.
		if s := qState(t, st, waiter); s != StateRunning {
			t.Fatalf("round %d: %s state=%s, want RUNNING — a waiter was stranded", i, waiter, s)
		}
		if _, err := st.Cancel(ctx, waiter, "test"); err != nil {
			t.Fatal(err)
		}
	}
}
