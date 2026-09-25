package store

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/taeels/enode/internal/contract"
)

// 이 파일은 대기 사유의 후보 수(US-7)를 확인한다 — 셋이 배타인지, 줄마다 따로
// 세는지, 셈의 시각이 DB 의 것인지. 끝에 비용을 측정하는 벤치마크가 하나 있다.

func cAdvert(t testing.TB, st *Store, nodeID, drain string, attrs map[string]string, ttl time.Duration) {
	t.Helper()
	a := contract.Advert{NodeID: nodeID, Label: nodeID,
		Capabilities: []contract.Capability{{Capability: contract.CapabilityAgentReason, Attrs: attrs}},
		Policy:       contract.Policy{Drain: drain}}
	if _, err := st.UpsertAdvert(context.Background(), a, "p", ttl); err != nil {
		t.Fatalf("cannot advertise %s: %v", nodeID, err)
	}
}

// cHold 는 그 노드의 임대를 쥔 다른 Run 이다.
func cHold(t testing.TB, st *Store, runID, nodeID string) {
	t.Helper()
	c := contract.Contract{RunID: runID,
		Requires: []contract.Require{{As: "b", Capability: contract.CapabilityAgentReason}},
		Steps:    []contract.Step{{ID: "one", Uses: "b", Run: []string{"true"}}}}
	r := Run{RunID: runID, State: StateRunning, Principal: "p", Contract: c,
		Assigned: []Assigned{{As: "b", Nodes: []NodeRef{{Node: nodeID, Label: nodeID}}}}}
	grants := []LeaseGrant{{NodeID: nodeID, NotAfter: time.Now().Add(time.Hour), Nonce: "n"}}
	if err := st.CreateRun(context.Background(), r, grants, c.Steps); err != nil {
		t.Fatalf("cannot hold %s: %v", nodeID, err)
	}
}

// 셋은 배타다 — busy 를 먼저 센다. 한 노드가 두 줄을 만족하면 두 줄에 다 든다.
// 만료된 광고와 요구를 못 채우는 광고는 live 에 없다.
func TestCandidatesFor_TheCountsAreExclusive(t *testing.T) {
	st := obsStore(t)
	ctx := context.Background()
	both := map[string]string{"harness": "claude", "role": "x"}

	cAdvert(t, st, "busy", "", claude, time.Minute)
	cHold(t, st, "holder-busy", "busy")
	cAdvert(t, st, "draining", "graceful", claude, time.Minute)
	cAdvert(t, st, "busy-and-draining", "at-boundary", claude, time.Minute)
	cHold(t, st, "holder-busy-and-draining", "busy-and-draining")
	cAdvert(t, st, "free", "", both, time.Minute)
	cAdvert(t, st, "other-harness", "", map[string]string{"harness": "codex"}, time.Minute)
	cAdvert(t, st, "expired", "", claude, time.Minute)
	xExec(t, st, `UPDATE nodes SET expires_at = now() - interval '1 minute' WHERE node_id = 'expired'`)

	reqs := []contract.Require{
		{As: "a", Capability: contract.CapabilityAgentReason, Attrs: claude},
		{As: "b", Capability: contract.CapabilityAgentReason, Attrs: map[string]string{"role": "x"}},
		{As: "c", Capability: contract.CapabilityAgentReason, Attrs: map[string]string{"harness": "gemini"}},
	}
	var before, after time.Time
	if err := st.pool.QueryRow(ctx, `SELECT now()`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	got, at, err := st.CandidatesFor(ctx, reqs)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.pool.QueryRow(ctx, `SELECT now()`).Scan(&after); err != nil {
		t.Fatal(err)
	}
	want := []Candidates{
		{Live: 4, Busy: 2, Draining: 1}, // busy · busy-and-draining · draining · free
		{Live: 1},                       // free 만 role x 를 든다 — 첫 줄에도 들었다
		{},                              // 아무도 못 채운다
	}
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	if at.Before(before) || at.After(after) {
		t.Fatalf("candidates_at %v is not the database clock between %v and %v", at, before, after)
	}
}

// 요구가 없으면 빈 목록과 시각이다.
func TestCandidatesFor_NoRequirements(t *testing.T) {
	st := obsStore(t)
	got, at, err := st.CandidatesFor(context.Background(), nil)
	if err != nil || len(got) != 0 || at.IsZero() {
		t.Fatalf("got %v at %v err %v", got, at, err)
	}
}

// BenchmarkCandidatesFor 는 QUEUED 인 Run 을 한 번 조회할 때 붙는 셈의 비용이다
// (step-phase 의 Code Generation 계획 3절). 광고 50 · 요구 줄 3 · 임대 10 · drain 5.
// 기본 go test 에서는 돌지 않는다 — go test -run '^$' -bench CandidatesFor ./internal/store
func BenchmarkCandidatesFor(b *testing.B) {
	st := benchStore(b)
	for i := 0; i < 50; i++ {
		id := fmt.Sprintf("n%02d", i)
		drain := ""
		if i%10 == 5 {
			drain = "graceful"
		}
		cAdvert(b, st, id, drain, map[string]string{"harness": "claude", "role": fmt.Sprint(i % 3)}, time.Hour)
		if i%5 == 0 {
			cHold(b, st, "hold-"+id, id)
		}
	}
	reqs := []contract.Require{
		{As: "a", Capability: contract.CapabilityAgentReason, Attrs: map[string]string{"harness": "claude"}},
		{As: "b", Capability: contract.CapabilityAgentReason, Attrs: map[string]string{"role": "1"}},
		{As: "c", Capability: contract.CapabilityAgentReason, Attrs: map[string]string{"role": "2"}},
	}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := st.CandidatesFor(ctx, reqs); err != nil {
			b.Fatal(err)
		}
	}
}

// benchStore 는 obsStore 와 같은 일을 벤치마크에서 한다 — 이 벤치마크만의 데이터베이스를
// 파고 Store 를 연다. obsStore 의 도우미는 *testing.T 만 받는다.
func benchStore(b *testing.B) *Store {
	b.Helper()
	base := os.Getenv("ENODE_TEST_DATABASE_URL")
	if base == "" {
		b.Fatal("ENODE_TEST_DATABASE_URL is unset; see scripts/testdb.sh")
	}
	ctx := context.Background()
	name := fmt.Sprintf("u2_bench_%d_%d", os.Getpid(), time.Now().UnixNano()%1e9)
	admin, err := pgx.Connect(ctx, base)
	if err != nil {
		b.Fatalf("cannot reach the test postgres: %v", err)
	}
	defer admin.Close(ctx) //nolint:errcheck // 닫기 실패는 측정을 바꾸지 않는다
	if _, err := admin.Exec(ctx, `CREATE DATABASE "`+name+`"`); err != nil {
		b.Fatalf("cannot create the scratch database %s: %v", name, err)
	}
	b.Cleanup(func() {
		drop, err := pgx.Connect(context.Background(), base)
		if err != nil {
			return
		}
		defer drop.Close(context.Background()) //nolint:errcheck // 닫기 실패는 측정을 바꾸지 않는다
		_, _ = drop.Exec(context.Background(), `DROP DATABASE IF EXISTS "`+name+`" WITH (FORCE)`)
	})
	u, err := url.Parse(base)
	if err != nil {
		b.Fatalf("cannot parse ENODE_TEST_DATABASE_URL: %v", err)
	}
	u.Path = "/" + name
	st, err := Open(ctx, u.String())
	if err != nil {
		b.Fatalf("cannot open the scratch store: %v", err)
	}
	b.Cleanup(st.Close)
	if err := st.Migrate(ctx); err != nil {
		b.Fatalf("cannot migrate the scratch store: %v", err)
	}
	return st
}
