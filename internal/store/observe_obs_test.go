package store

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/match"
)

// 이 파일은 관측 표면(GET /v1/nodes · GET /v1/runs)이 딛는 저장소 계약을 잰다.
//
// 진짜 Postgres 위에서 돈다. 가짜로 대신할 수 없는 것이 이 유닛의 전부이기
// 때문이다 — 잰다고 말하는 것이 LEFT JOIN 이 0행에서 남기는 한 줄, CTE 가
// 문장 시작 스냅숏을 본다는 성질, coalesce 가 SQL NULL 만 접고 jsonb 의
// null 은 못 접는다는 사실이다. 셋 다 드라이버가 아니라 서버의 성질이다.
//
// 판마다 데이터베이스를 따로 판다. reap_deterministic_test.go 가 같은 이유로
// 이미 그 모양이다 — go test ./... 는 패키지별 바이너리를 병렬로 돌리고
// internal/api 의 통합 시험이 공용 테스트 DB 를 Truncate 로 비운다.
// 같이 돌면 서로의 결과를 바꾸고, 그것은 아무도 재현하지 못하는 실패다.
// 새 판은 처음부터 비어 있으므로 Truncate 도 필요 없다.

// obsStore 는 이 파일만의 데이터베이스를 파고 그 위에 Store 를 연다.
func obsStore(t *testing.T) *Store {
	t.Helper()
	base := os.Getenv("ENODE_TEST_DATABASE_URL")
	if base == "" {
		// 스킵하지 않는다 — CI 의 스킵 감시가 전 패키지를 보고
		// .ci-allowed-skips 는 비어 있다. 이 패키지의 기존 자리와 같은 판단이다.
		t.Fatal("ENODE_TEST_DATABASE_URL is unset; see scripts/testdb.sh. " +
			"the observation queries cannot be measured without a real postgres")
	}
	ctx := context.Background()
	st, err := Open(ctx, scratchDB(t, base))
	if err != nil {
		t.Fatalf("cannot open the scratch store: %v", err)
	}
	t.Cleanup(st.Close)
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("cannot migrate the scratch store: %v", err)
	}
	return st
}

// obsAdvert 는 능력 하나를 든 광고다.
func obsAdvert(nodeID, label, drain string) contract.Advert {
	return contract.Advert{
		NodeID: nodeID,
		Label:  label,
		Capabilities: []contract.Capability{{
			Capability: contract.CapabilityAgentReason,
			Attrs:      map[string]string{"harness": "claude"},
		}},
		Policy: contract.Policy{Drain: drain},
	}
}

// obsContract 는 요구 하나와 단계 하나를 든 최소 계약이다.
func obsContract() contract.Contract {
	return contract.Contract{
		Requires: []contract.Require{{
			As: "b", Capability: contract.CapabilityAgentReason,
			Attrs: map[string]string{"harness": "claude"},
		}},
		Steps: []contract.Step{{ID: "one", Uses: "b", Agent: map[string]any{}}},
	}
}

// obsRun 은 배정된 Run 하나를 만든다. grants 가 비면 임대 없이 만든다.
func obsRun(t *testing.T, st *Store, runID, state, submitter string, grants []LeaseGrant) {
	t.Helper()
	c := obsContract()
	r := Run{
		RunID: runID, State: state, Principal: "someone@example.test",
		Contract:  c,
		Assigned:  []Assigned{{As: "b", Nodes: []NodeRef{{Node: "node-a", Label: "a"}}}},
		Submitter: submitter,
	}
	if err := st.CreateRun(context.Background(), r, grants, c.Steps); err != nil {
		t.Fatalf("cannot create run %s: %v", runID, err)
	}
}

// obsBackdate 는 created_at 을 옮긴다. 정렬과 since 의 경계를 재려면
// 만든 순서가 아니라 정해진 시각이어야 한다.
func obsBackdate(t *testing.T, st *Store, runID string, at time.Time) {
	t.Helper()
	if _, err := st.pool.Exec(context.Background(),
		`UPDATE runs SET created_at = $1 WHERE run_id = $2`, at, runID); err != nil {
		t.Fatalf("cannot backdate run %s: %v", runID, err)
	}
}

// 빈 함대에서도 관측 시각이 선다.
//
// 이것이 at CTE 를 왼쪽에 두는 이유의 전부다. observed_at 을 노드 행에만
// 얹으면 노드가 0 일 때 값이 없고, 그러면 화면이 "함대가 비었다" 와
// "못 읽었다" 를 못 가른다.
func TestNodes_AnEmptyFleetStillCarriesTheObservedInstant(t *testing.T) {
	st := obsStore(t)

	nodes, observed, err := st.Nodes(context.Background())
	if err != nil {
		t.Fatalf("Nodes on an empty fleet: %v", err)
	}
	if len(nodes) != 0 {
		t.Fatalf("Nodes on an empty fleet: got %d rows, want 0", len(nodes))
	}
	if observed.IsZero() {
		t.Fatal("Nodes on an empty fleet: observed_at is the zero time, want the database clock")
	}
	// nil 이 아니라 빈 배열이어야 한다 — null 은 읽는 쪽에 기본값 규칙을 추측하게 한다.
	if b, _ := json.Marshal(nodes); string(b) != "[]" {
		t.Fatalf("empty fleet marshals to %s, want []", b)
	}
}

// 임대가 있으면 값이 실리고 없으면 null 이다.
func TestNodes_ALeaseIsFilledOnlyForTheNodeThatHoldsOne(t *testing.T) {
	st := obsStore(t)
	ctx := context.Background()
	ttl := time.Minute

	for _, id := range []string{"node-a", "node-b"} {
		if _, err := st.UpsertAdvert(ctx, obsAdvert(id, id+" label", ""), "p", ttl); err != nil {
			t.Fatalf("cannot advertise %s: %v", id, err)
		}
	}
	notAfter := time.Now().Add(2 * time.Minute).UTC().Truncate(time.Millisecond)
	obsRun(t, st, "run-1", StateRunning, "", []LeaseGrant{
		{NodeID: "node-a", NotAfter: notAfter, Nonce: "n1"}})

	nodes, observed, err := st.Nodes(ctx)
	if err != nil {
		t.Fatalf("Nodes: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("Nodes: got %d rows, want 2", len(nodes))
	}
	// node_id 오름차순이어야 한다 — 정렬이 없으면 카드가 폴링마다 자리를 바꾼다.
	if nodes[0].NodeID != "node-a" || nodes[1].NodeID != "node-b" {
		t.Fatalf("Nodes order: got %s then %s, want node-a then node-b",
			nodes[0].NodeID, nodes[1].NodeID)
	}
	if nodes[0].Lease == nil {
		t.Fatal("node-a holds a lease but lease is nil")
	}
	if nodes[0].Lease.RunID != "run-1" {
		t.Fatalf("node-a lease run_id: got %q, want run-1", nodes[0].Lease.RunID)
	}
	if !nodes[0].Lease.NotAfter.Equal(notAfter) {
		t.Fatalf("node-a lease not_after: got %s, want %s", nodes[0].Lease.NotAfter, notAfter)
	}
	if nodes[1].Lease != nil {
		t.Fatalf("node-b holds no lease but lease is %+v", nodes[1].Lease)
	}
	if nodes[0].Label != "node-a label" || nodes[0].SeenAt.IsZero() || nodes[0].ExpiresAt.IsZero() {
		t.Fatalf("node-a is missing advertised facts: %+v", nodes[0])
	}
	if len(nodes[0].Capabilities) != 1 ||
		nodes[0].Capabilities[0].Capability != contract.CapabilityAgentReason {
		t.Fatalf("node-a capabilities: got %+v, want one agent.reason", nodes[0].Capabilities)
	}
	if observed.Before(nodes[0].SeenAt) {
		t.Fatalf("observed_at %s is before seen_at %s; the two clocks disagree",
			observed, nodes[0].SeenAt)
	}
	// principal 은 응답 모양에 없다 — 마셜한 글자로 확인한다.
	b, err := json.Marshal(nodes[1])
	if err != nil {
		t.Fatalf("cannot marshal a node view: %v", err)
	}
	if strings.Contains(string(b), "principal") {
		t.Fatalf("a node view carries principal: %s", b)
	}
	if !strings.Contains(string(b), `"lease":null`) {
		t.Fatalf("an idle node marshals its lease as %s, want lease null", b)
	}
}

// 만료된 광고는 안 나온다. 필터의 기준이 응답이 말하는 시각과 같아야 한다.
func TestNodes_AnExpiredAdvertisementIsNotReturned(t *testing.T) {
	st := obsStore(t)
	ctx := context.Background()

	if _, err := st.UpsertAdvert(ctx, obsAdvert("node-gone", "gone", ""), "p", time.Minute); err != nil {
		t.Fatalf("cannot advertise: %v", err)
	}
	if _, err := st.pool.Exec(ctx,
		`UPDATE nodes SET expires_at = now() - interval '1s'`); err != nil {
		t.Fatalf("cannot expire the advertisement: %v", err)
	}

	nodes, observed, err := st.Nodes(ctx)
	if err != nil {
		t.Fatalf("Nodes: %v", err)
	}
	if len(nodes) != 0 {
		t.Fatalf("Nodes with one expired advertisement: got %d rows, want 0", len(nodes))
	}
	if observed.IsZero() {
		t.Fatal("Nodes: observed_at is the zero time even though the query ran")
	}
}

// 처음 광고하는 노드가 죽지 않는다.
//
// 이 시험이 지키는 것은 한 글자다 — coalesce 가 스칼라 서브쿼리 밖에 있다는
// 것. 안에 넣으면 old 가 0행일 때 여전히 SQL NULL 이 나오고, 0행은 사고가
// 아니라 "이 노드의 첫 광고" 다. 그러면 아무 노드도 함대에 못 들어온다.
func TestUpsertAdvert_TheFirstAdvertisementOfANodeSurvives(t *testing.T) {
	st := obsStore(t)

	prev, err := st.UpsertAdvert(context.Background(),
		obsAdvert("node-new", "new", DrainGraceful), "p", time.Minute)
	if err != nil {
		t.Fatalf("the first advertisement of a node failed: %v", err)
	}
	if prev != "" {
		t.Fatalf("the first advertisement returned a previous drain %q, want the empty string", prev)
	}
}

// 두 번째 광고는 이전 값을 돌려준다 — drain 해제를 볼 수 있게 하는 그 값이다.
func TestUpsertAdvert_TheSecondCallReturnsThePreviousDrain(t *testing.T) {
	st := obsStore(t)
	ctx := context.Background()

	if _, err := st.UpsertAdvert(ctx, obsAdvert("node-a", "a", DrainGraceful), "p", time.Minute); err != nil {
		t.Fatalf("cannot advertise a drained node: %v", err)
	}
	prev, err := st.UpsertAdvert(ctx, obsAdvert("node-a", "a", DrainNone), "p", time.Minute)
	if err != nil {
		t.Fatalf("cannot advertise the release: %v", err)
	}
	if prev != DrainGraceful {
		t.Fatalf("the release advertisement returned %q as the previous drain, want %q",
			prev, DrainGraceful)
	}
	nodes, _, err := st.Nodes(ctx)
	if err != nil {
		t.Fatalf("Nodes: %v", err)
	}
	if len(nodes) != 1 || nodes[0].Draining != DrainNone {
		t.Fatalf("after the release the stored drain is %+v, want the empty string", nodes)
	}
}

// 어휘 밖의 값은 "" 로 접혀 저장된다. 400 이 아니다 — 광고는 하트비트를 겸한다.
func TestUpsertAdvert_AnUnknownDrainPolicyIsFoldedToNone(t *testing.T) {
	st := obsStore(t)
	ctx := context.Background()
	var logged bytes.Buffer
	st.Log = slog.New(slog.NewTextHandler(&logged, nil))

	const unknown = "definitely-not-a-policy"
	prev, err := st.UpsertAdvert(ctx, obsAdvert("node-a", "a", unknown), "p", time.Minute)
	if err != nil {
		t.Fatalf("an unknown drain policy rejected the whole advertisement: %v", err)
	}
	if prev != "" {
		t.Fatalf("previous drain: got %q, want the empty string", prev)
	}
	nodes, _, err := st.Nodes(ctx)
	if err != nil {
		t.Fatalf("Nodes: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("the node left the fleet over a policy value: got %d rows, want 1", len(nodes))
	}
	if nodes[0].Draining != DrainNone {
		t.Fatalf("stored drain: got %q, want the empty string", nodes[0].Draining)
	}
	// 접었다는 사실은 남기되 값 원문은 안 찍는다 (SECURITY-03).
	if !strings.Contains(logged.String(), "unknown drain policy") {
		t.Fatalf("nothing was logged about the folded policy: %q", logged.String())
	}
	if !strings.Contains(logged.String(), "node-a") {
		t.Fatalf("the log does not name the node: %q", logged.String())
	}
	if strings.Contains(logged.String(), unknown) {
		t.Fatalf("the log echoes the raw policy value: %q", logged.String())
	}
}

// DrainPolicy 는 순수 함수다. 접는 규칙이 여기 하나로 산다.
func TestDrainPolicy_FoldsEverythingOutsideTheVocabulary(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"graceful", "graceful"},
		{"at-boundary", "at-boundary"},
		{"GRACEFUL", ""},
		{"at boundary", ""},
		{"true", ""},
	}
	for _, c := range cases {
		if got := DrainPolicy(c.in); got != c.want {
			t.Errorf("DrainPolicy(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// 목록은 최신이 앞이고 필터는 좁히기만 한다.
func TestRuns_FiltersNarrowAndTheNewestComesFirst(t *testing.T) {
	st := obsStore(t)
	ctx := context.Background()

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	obsRun(t, st, "run-old", StateRunning, "", nil)
	obsRun(t, st, "run-mid", StateSucceeded, "", nil)
	obsRun(t, st, "run-new", StateRunning, "", nil)
	obsBackdate(t, st, "run-old", base)
	obsBackdate(t, st, "run-mid", base.Add(time.Hour))
	obsBackdate(t, st, "run-new", base.Add(2*time.Hour))
	if _, err := st.pool.Exec(ctx,
		`UPDATE runs SET work_id = 'w-1' WHERE run_id = 'run-mid'`); err != nil {
		t.Fatalf("cannot set a work id: %v", err)
	}

	cases := []struct {
		name   string
		filter RunFilter
		want   []string
	}{
		{"no filter lists everything newest first", RunFilter{Limit: 10},
			[]string{"run-new", "run-mid", "run-old"}},
		{"state narrows to that state", RunFilter{State: StateRunning, Limit: 10},
			[]string{"run-new", "run-old"}},
		{"since is an inclusive lower bound", RunFilter{Since: base.Add(time.Hour), Limit: 10},
			[]string{"run-new", "run-mid"}},
		{"work narrows to that work", RunFilter{Work: "w-1", Limit: 10},
			[]string{"run-mid"}},
		{"limit cuts the newest end", RunFilter{Limit: 1}, []string{"run-new"}},
		{"a filter that matches nothing is an empty list", RunFilter{State: "NOPE", Limit: 10},
			[]string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rows, observed, err := st.Runs(ctx, c.filter)
			if err != nil {
				t.Fatalf("Runs: %v", err)
			}
			if observed.IsZero() {
				t.Fatal("Runs: observed_at is the zero time")
			}
			got := make([]string, 0, len(rows))
			for _, r := range rows {
				got = append(got, r.RunID)
			}
			if strings.Join(got, ",") != strings.Join(c.want, ",") {
				t.Fatalf("Runs: got %v, want %v", got, c.want)
			}
			if rows == nil {
				t.Fatal("Runs returned a nil slice; the wire needs an empty array")
			}
		})
	}
}

// work_id 가 NULL 인 옛 행은 "" 로 보이지만 어떤 ?work= 값으로도 안 잡힌다.
//
// 그것이 접기 전 열에 등호를 거는 대가다. 인덱스를 타려면 그렇게 해야 하고,
// "" 로 거르는 것은 「work 가 없는 것 전부」라 뜻이 없는 질의다.
func TestRuns_ALegacyNullWorkIdReadsAsEmptyAndMatchesNoFilter(t *testing.T) {
	st := obsStore(t)
	ctx := context.Background()

	obsRun(t, st, "run-legacy", StateRunning, "", nil)

	rows, _, err := st.Runs(ctx, RunFilter{Limit: 10})
	if err != nil {
		t.Fatalf("Runs: %v", err)
	}
	if len(rows) != 1 || rows[0].WorkID != "" {
		t.Fatalf("a run with no work id reads as %+v, want work_id folded to the empty string", rows)
	}
	rows, _, err = st.Runs(ctx, RunFilter{Work: "anything", Limit: 10})
	if err != nil {
		t.Fatalf("Runs with a work filter: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("a run with no work id was matched by a work filter: %+v", rows)
	}
}

// verdict 는 목록에 실린다 — drain 으로 닫힌 FAILED 와 진짜 실패를
// 목록에서 가르는 열이 그것뿐이다.
func TestRuns_TheVerdictComesBackOnTheList(t *testing.T) {
	st := obsStore(t)
	ctx := context.Background()

	obsRun(t, st, "run-1", StateFailed, "", nil)
	if _, err := st.pool.Exec(ctx,
		`UPDATE runs SET verdict = $1, ended_at = now() WHERE run_id = 'run-1'`,
		[]byte(`{"state":"FAILED","checks":[]}`)); err != nil {
		t.Fatalf("cannot store a verdict: %v", err)
	}

	rows, _, err := st.Runs(ctx, RunFilter{Limit: 10})
	if err != nil {
		t.Fatalf("Runs: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("Runs: got %d rows, want 1", len(rows))
	}
	if rows[0].Verdict == nil {
		t.Fatal("the verdict is nil on a settled run")
	}
	if rows[0].Verdict.State != StateFailed {
		t.Fatalf("verdict state: got %q, want %q", rows[0].Verdict.State, StateFailed)
	}
	if rows[0].EndedAt == nil {
		t.Fatal("ended_at is nil on a settled run")
	}
}

// 아무것도 안 맞아도 관측 시각이 선다 — 빈 함대와 같은 규칙이다.
func TestRuns_AnEmptyResultStillCarriesTheObservedInstant(t *testing.T) {
	st := obsStore(t)

	rows, observed, err := st.Runs(context.Background(), RunFilter{Limit: 10})
	if err != nil {
		t.Fatalf("Runs on an empty store: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("Runs on an empty store: got %d rows, want 0", len(rows))
	}
	if observed.IsZero() {
		t.Fatal("Runs on an empty store: observed_at is the zero time")
	}
	if b, _ := json.Marshal(rows); string(b) != "[]" {
		t.Fatalf("an empty result marshals to %s, want []", b)
	}
}

// 거절된 Run 도 목록에 이름을 남기고 assigned 는 [] 로 나간다.
//
// 거절 경로가 CP9 가 재는 행이다 — 데모에서 가장 흔한 실패가 보드가 꺼져
// 있어 나는 422 이고, 그 행의 제출자 이름이 비면 잰 것이 없다.
func TestRuns_ARejectedRunCarriesItsSubmitterAndAnEmptyAssigned(t *testing.T) {
	st := obsStore(t)
	ctx := context.Background()

	r := Run{RunID: "run-rejected", Principal: "someone@example.test",
		Contract:  obsContract(),
		Reject:    &match.Reject{Code: 422, As: "b", Reason: "no node satisfies the requirement"},
		Submitter: "brave-otter"}
	if err := st.CreateRejectedRun(ctx, r); err != nil {
		t.Fatalf("cannot record a rejected run: %v", err)
	}

	rows, _, err := st.Runs(ctx, RunFilter{Limit: 10})
	if err != nil {
		t.Fatalf("Runs: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("Runs: got %d rows, want 1", len(rows))
	}
	if rows[0].State != StateFailed {
		t.Fatalf("a rejected run is in state %q, want %q", rows[0].State, StateFailed)
	}
	if rows[0].Submitter != "brave-otter" {
		t.Fatalf("submitter: got %q, want brave-otter", rows[0].Submitter)
	}
	if rows[0].Assigned == nil || len(rows[0].Assigned) != 0 {
		t.Fatalf("assigned on a rejected run is %+v, want an empty array", rows[0].Assigned)
	}
	if rows[0].EndedAt == nil {
		t.Fatal("ended_at is nil on a rejected run")
	}
	b, err := json.Marshal(rows[0])
	if err != nil {
		t.Fatalf("cannot marshal a run row: %v", err)
	}
	for _, want := range []string{`"assigned":[]`, `"submitter":"brave-otter"`,
		`"verdict":null`, `"ended_at":`, `"work_id":""`} {
		if !strings.Contains(string(b), want) {
			t.Fatalf("a rejected run marshals to %s, want it to contain %s", b, want)
		}
	}
}

// submitter 는 두 입구 모두를 왕복한다.
func TestRuns_TheSubmitterRoundTripsThroughBothCreators(t *testing.T) {
	st := obsStore(t)
	ctx := context.Background()

	obsRun(t, st, "run-accepted", StateRunning, "quiet-heron", nil)
	if err := st.CreateRejectedRun(ctx, Run{RunID: "run-refused",
		Principal: "someone@example.test", Contract: obsContract(),
		Reject:    &match.Reject{Code: 422, As: "b", Reason: "no node"},
		Submitter: "brave-otter"}); err != nil {
		t.Fatalf("cannot record a rejected run: %v", err)
	}

	rows, _, err := st.Runs(ctx, RunFilter{Limit: 10})
	if err != nil {
		t.Fatalf("Runs: %v", err)
	}
	got := map[string]string{}
	for _, r := range rows {
		got[r.RunID] = r.Submitter
	}
	if got["run-accepted"] != "quiet-heron" {
		t.Errorf("CreateRun submitter: got %q, want quiet-heron", got["run-accepted"])
	}
	if got["run-refused"] != "brave-otter" {
		t.Errorf("CreateRejectedRun submitter: got %q, want brave-otter", got["run-refused"])
	}
}

// chosen 은 언제나 실린다. false 를 생략하면 SKIPPED 가 다시 두 가지를 뜻한다.
func TestSteps_ChosenIsAlwaysOnTheWire(t *testing.T) {
	st := obsStore(t)
	ctx := context.Background()

	obsRun(t, st, "run-1", StateRunning, "", nil)
	steps, err := st.Steps(ctx, "run-1")
	if err != nil {
		t.Fatalf("Steps: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("Steps: got %d rows, want 1", len(steps))
	}
	if steps[0].Chosen {
		t.Fatal("a fresh step is already chosen")
	}
	b, err := json.Marshal(steps[0])
	if err != nil {
		t.Fatalf("cannot marshal a step view: %v", err)
	}
	if !strings.Contains(string(b), `"chosen":false`) {
		t.Fatalf("a step marshals to %s, want it to carry chosen false", b)
	}

	if _, err := st.pool.Exec(ctx,
		`UPDATE steps SET chosen = true WHERE run_id = 'run-1'`); err != nil {
		t.Fatalf("cannot mark the step chosen: %v", err)
	}
	steps, err = st.Steps(ctx, "run-1")
	if err != nil {
		t.Fatalf("Steps: %v", err)
	}
	if !steps[0].Chosen {
		t.Fatal("the step was marked chosen in the table but the view says false")
	}
}

// requires 는 attrs 로 감싸이고 비면 {} 를 낸다. count 만 생략된다.
func TestRequiresOf_WrapsAttrsAndEmitsAnEmptyObject(t *testing.T) {
	c := contract.Contract{Requires: []contract.Require{
		{As: "b", Capability: contract.CapabilityAgentReason,
			Attrs: map[string]string{"harness": "claude", "os": "linux/amd64"}},
		{As: "bare", Capability: contract.CapabilityAgentReason, Count: 2},
	}}

	got := RequiresOf(c)
	if len(got) != 2 {
		t.Fatalf("RequiresOf: got %d views, want 2", len(got))
	}
	if got[0].As != "b" || got[0].Capability != contract.CapabilityAgentReason {
		t.Fatalf("the first view lost its identity: %+v", got[0])
	}
	if got[0].Attrs["harness"] != "claude" || got[0].Attrs["os"] != "linux/amd64" {
		t.Fatalf("attrs: got %+v, want harness and os", got[0].Attrs)
	}
	if got[1].Attrs == nil {
		t.Fatal("an empty attrs map is nil; the wire needs an object")
	}

	b, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("cannot marshal the requirement views: %v", err)
	}
	if !strings.Contains(string(b), `"attrs":{}`) {
		t.Fatalf("requires marshals to %s, want an empty attrs object", b)
	}
	if strings.Contains(string(b), `"count":0`) {
		t.Fatalf("requires marshals to %s, want count omitted when it is zero", b)
	}
	if !strings.Contains(string(b), `"count":2`) {
		t.Fatalf("requires marshals to %s, want count carried when it is set", b)
	}
	// 관측 모양은 계약 입력과 다르다 — 속성이 형제 키로 펴지지 않는다.
	if strings.Contains(string(b), `"harness":"claude","os"`) &&
		!strings.Contains(string(b), `"attrs":{"harness"`) {
		t.Fatalf("requires spread its attrs as sibling keys: %s", b)
	}

	// 요구가 없으면 빈 목록이다. nil 이어도 omitempty 가 같은 자리에서 접는다.
	if len(RequiresOf(contract.Contract{})) != 0 {
		t.Fatal("RequiresOf on a contract with no requirements returned rows")
	}
}
