package store

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// 이 파일은 Reap 의 오류 갈래를 의도적으로 밟는다.
//
// 왜 따로 있나 — 실측이다. 이 갈래 넷은 이 파일이 생기기 전에도 커버리지
// 프로파일에 나타났지만, 밟은 것은 테스트가 아니라 경합이었다.
// cmd/mediator 의 바인드 실패 시험이 RunReaper 를 고루틴으로 띄운 채 컨텍스트를
// 취소하고, 그 취소가 재시작 스캔의 어느 쿼리에 떨어지느냐가 실행마다 달랐다.
// 같은 커밋을 두 번 재면 internal/store 가 1230 과 1228 사이를 오갔고,
// 어느 갈래가 덮이는지도 실행마다 바뀌었다.
//
//	AC3.4.6  같은 커밋 · 같은 CI 구성에서 두 번 재면 덮인/총 이 정확히 같다
//	NFR2     같은 명령이 같은 수를 낸다
//
// 판정(80% 하한)은 어느 실행에서도 안 뒤집혔다 — 여유가 28문장이었다.
// 뒤집힌 것은 게이트의 입력이 재현된다는 주장이고, 하한을 차단으로 올리면
// 그 주장이 게이트를 지탱한다.
//
// 고치는 방향을 둘 중에서 골랐다. 경합을 없애는 쪽(생산 코드의 종료 순서를
// 다시 손대는 것)은 U8 이 이미 확정한 stderr 계약(AC3.2.5)을 다시 열고,
// 무엇보다 그 갈래를 아무도 안 밟게 만들 뿐 밟았다는 사실을 증명하지는
// 않는다. 여기서는 반대로 간다 — **의도적으로 먼저 밟는다.** 그러면 경합이
// 나중에 같은 갈래를 밟아도 이미 덮인 것을 다시 덮는 것이라 수치가 안 움직인다.
// 덤으로, 지금까지 아무 테스트도 단언하지 않던 계약 둘이 단언을 얻는다.

// reapTestStore 는 이 테스트만의 데이터베이스를 파고 그 위에 Store 를 연다.
//
// 공용 테스트 DB 를 그대로 쓰는 안을 기각했다 — 실측으로 기각했다.
// 초안은 공용 DB 에 붙어 Truncate 로 시작했는데, go test ./... 는 패키지별
// 테스트 바이너리를 병렬로 돌리고 internal/api 의 통합 78개가 같은 DB 를
// 쓴다. 함께 돌리면 3회 중 2회가 실제로 깨졌다 —
// internal/api 의 TestAnswerPath_TheRouteHasOneOwner 가
// `deadlock detected (SQLSTATE 40P01)` 로, 이 파일의 테스트가 외래키 위반과
// 또 다른 교착으로. 두 패키지의 테스트가 서로의 결과를 바꾸는 모양이고,
// 그것은 아무도 재현하지 못하는 실패다.
//
// cmd/mediator/entrypoint_unix_test.go 의 scratchDB 가 정확히 같은 이유로
// 이미 이 모양을 하고 있다. 판 하나를 따로 파면 그 경로가 아예 없어지고,
// Truncate 도 필요 없어진다 — 새 판은 처음부터 비어 있다.
func reapTestStore(t *testing.T) *Store {
	t.Helper()
	base := os.Getenv("ENODE_TEST_DATABASE_URL")
	if base == "" {
		// 스킵하지 않는다 — CI 의 스킵 감시가 전 패키지를 보고
		// .ci-allowed-skips 는 비어 있다 (AC3.4.3 · FR4.5).
		t.Fatal("ENODE_TEST_DATABASE_URL is unset; see scripts/testdb.sh. " +
			"the reaper's failure branches cannot be measured without a real postgres")
	}
	url := scratchDB(t, base)
	ctx := context.Background()
	st, err := Open(ctx, url)
	if err != nil {
		t.Fatalf("cannot open the scratch store: %v", err)
	}
	t.Cleanup(st.Close)
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("cannot migrate the scratch store: %v", err)
	}
	return st
}

// scratchDB 는 이 테스트만의 데이터베이스를 만들고 그 URL 을 돌려준다.
func scratchDB(t *testing.T, base string) string {
	t.Helper()
	name := fmt.Sprintf("u7_reap_%d_%d", os.Getpid(), time.Now().UnixNano()%1e9)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	admin, err := pgx.Connect(ctx, base)
	if err != nil {
		t.Fatalf("cannot reach the test postgres at ENODE_TEST_DATABASE_URL: %v", err)
	}
	defer admin.Close(ctx) //nolint:errcheck // 닫기 실패는 판정을 바꾸지 않는다
	if _, err := admin.Exec(ctx, `CREATE DATABASE "`+name+`"`); err != nil {
		t.Fatalf("cannot create the scratch database %s: %v", name, err)
	}
	t.Cleanup(func() {
		dctx, dcancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer dcancel()
		drop, err := pgx.Connect(dctx, base)
		if err != nil {
			return
		}
		defer drop.Close(dctx) //nolint:errcheck // 닫기 실패는 판정을 바꾸지 않는다
		_, _ = drop.Exec(dctx, `DROP DATABASE IF EXISTS "`+name+`" WITH (FORCE)`)
	})
	return replaceDBName(t, base, name)
}

// replaceDBName 은 URL 의 데이터베이스 이름만 갈아 끼운다.
func replaceDBName(t *testing.T, raw, name string) string {
	t.Helper()
	cfg, err := pgx.ParseConfig(raw)
	if err != nil {
		t.Fatalf("cannot parse ENODE_TEST_DATABASE_URL: %v", err)
	}
	host := cfg.Host
	if !strings.HasPrefix(host, "/") {
		host = net.JoinHostPort(cfg.Host, strconv.Itoa(int(cfg.Port)))
	}
	u := "postgres://" + cfg.User
	if cfg.Password != "" {
		u += ":" + cfg.Password
	}
	return u + "@" + host + "/" + name + "?sslmode=disable"
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// 취소된 컨텍스트로 부르면 재시작 스캔이 첫 질의에서 멈춘다.
//
// 밟는 자리 셋 — ExpireAsks 의 질의 실패, Reap 가 그 실패를 로그로만 넘기는
// 자리, 그리고 그다음 회수 UPDATE 도 같은 이유로 실패하는 자리.
func TestReap_ACancelledContextStopsAtTheFirstQuery(t *testing.T) {
	st := reapTestStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	n, err := st.Reap(ctx, quietLogger())
	if err == nil {
		t.Fatal("Reap with a cancelled context: got nil error, want the cancellation surfaced")
	}
	if n != 0 {
		t.Fatalf("Reap stopped before reclaiming anything: got n=%d, want 0", n)
	}
	if ctx.Err() == nil {
		t.Fatal("the context should still report cancellation")
	}
}

// ExpireAsks 자신도 같은 갈래를 갖는다. Reap 를 거치지 않고 직접 확인한다 —
// Reap 가 그 오류를 로그로만 넘기므로 호출자에게 안 보이기 때문이다.
func TestExpireAsks_ACancelledContextSurfacesTheFailure(t *testing.T) {
	st := reapTestStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	runs, err := st.ExpireAsks(ctx)
	if err == nil {
		t.Fatal("ExpireAsks with a cancelled context: got nil error, want the cancellation surfaced")
	}
	if runs != nil {
		t.Fatalf("ExpireAsks failed before collecting anything: got %v, want nil", runs)
	}
}

// 회수는 성공했는데 그 뒤 단계 정리가 실패하면, Reap 는 회수한 수를 들고
// 오류를 함께 돌려준다. 0 을 돌려주면 부르는 쪽이 「아무것도 안 거뒀다」로
// 읽고, 실제로 FAILED 로 바뀐 Run 이 그 판단과 어긋난다.
//
// 단계 정리만 실패하게 만드는 방법 — steps 에 UPDATE 를 거부하는 트리거를
// 건다. 회수 UPDATE 는 steps 를 읽기만 하므로 통과하고, 그다음 단계 정리는
// steps 에 쓰므로 걸린다. 컨텍스트 취소로는 이 자리를 못 만든다: 취소는
// 두 질의를 가리지 않고 앞의 것부터 죽인다.
func TestReap_AStepsCleanupFailureStillCarriesTheReclaimCount(t *testing.T) {
	st := reapTestStore(t)
	ctx := context.Background()

	// 만료된 임대를 하나 심는다 — 회수 대상이다.
	if _, err := st.pool.Exec(ctx, `
		INSERT INTO runs (run_id, state, principal, contract)
		VALUES ('reap-fixture', 'RUNNING', 'test', '{}'::jsonb)`); err != nil {
		t.Fatalf("cannot seed the run: %v", err)
	}
	if _, err := st.pool.Exec(ctx, `
		INSERT INTO steps (run_id, seq, name, uses, kind, state)
		VALUES ('reap-fixture', 1, 'observe', 'worker', 'agent', 'PENDING')`); err != nil {
		t.Fatalf("cannot seed the step: %v", err)
	}
	if _, err := st.pool.Exec(ctx, `
		INSERT INTO leases (node_id, run_id, not_after, nonce)
		VALUES ('reap-node', 'reap-fixture', now() - interval '1 minute', 'n')`); err != nil {
		t.Fatalf("cannot seed the expired lease: %v", err)
	}

	if _, err := st.pool.Exec(ctx, `
		CREATE OR REPLACE FUNCTION reap_test_refuse() RETURNS trigger AS $$
		BEGIN RAISE EXCEPTION 'reap_test: steps is read-only in this test'; END;
		$$ LANGUAGE plpgsql`); err != nil {
		t.Fatalf("cannot create the refusing function: %v", err)
	}
	if _, err := st.pool.Exec(ctx, `
		CREATE TRIGGER reap_test_no_update BEFORE UPDATE ON steps
		FOR EACH ROW EXECUTE FUNCTION reap_test_refuse()`); err != nil {
		t.Fatalf("cannot create the refusing trigger: %v", err)
	}
	t.Cleanup(func() {
		cctx := context.Background()
		_, _ = st.pool.Exec(cctx, `DROP TRIGGER IF EXISTS reap_test_no_update ON steps`)
		_, _ = st.pool.Exec(cctx, `DROP FUNCTION IF EXISTS reap_test_refuse()`)
	})

	n, err := st.Reap(ctx, quietLogger())
	if err == nil {
		t.Fatal("Reap with a refusing steps table: got nil error, want the cleanup failure surfaced")
	}
	if n != 1 {
		t.Fatalf("the reclaim count must survive the later failure: got n=%d, want 1", n)
	}

	// 회수 자체는 실제로 일어났다 — n 이 지어낸 수가 아니라는 확인.
	var state string
	if err := st.pool.QueryRow(ctx,
		`SELECT state FROM runs WHERE run_id = 'reap-fixture'`).Scan(&state); err != nil {
		t.Fatalf("cannot read the run back: %v", err)
	}
	if state != StateFailed {
		t.Fatalf("the expired run should have been reclaimed: got %q, want %q", state, StateFailed)
	}
}

// RunReaper 의 루프도 같은 이유로 실행마다 갈렸다 — 타이머 갈래(t.C), 스캔
// 호출, 그리고 다음 주기 예약. 셋 다 「취소가 오기 전에 한 바퀴를 돌았는가」에
// 걸린다.
//
// 이 시험은 **두 바퀴**를 요구한다. 초안은 한 바퀴만 확인했는데, 검토가
// 변이로 그것을 깼다 — RunReaper 에서 재예약 루프를 들어내고 Reap 한 번 뒤
// 돌아가게 바꿔도 초안이 통과했다. 한 바퀴만 보면 「주기 스캔」이 아니라
// 「시작 스캔」만 검증하는 것이고, 그것은 이 함수 이름이 약속한 것이 아니다.
//
// 두 바퀴를 시간이 아니라 **결과**로 읽는다 — 첫 바퀴가 돈 것을 확인한 뒤에
// 두 번째 만료 임대를 새로 심고, 그것까지 회수되기를 기다린다. 두 번째는
// RunReaper 가 살아서 다시 돌 때만 회수되므로, 재예약이 없으면 영원히 안 온다.
// 시간으로 기다리면 지금 고치려는 그 경합을 테스트 안에 다시 만드는 것이다.
func TestRunReaper_ScansThenReschedulesUntilCancelled(t *testing.T) {
	st := reapTestStore(t)
	seed := context.Background()

	plant := func(runID, nodeID string) {
		t.Helper()
		if _, err := st.pool.Exec(seed, `
			INSERT INTO runs (run_id, state, principal, contract)
			VALUES ($1, 'RUNNING', 'test', '{}'::jsonb)`, runID); err != nil {
			t.Fatalf("cannot seed the run %s: %v", runID, err)
		}
		if _, err := st.pool.Exec(seed, `
			INSERT INTO leases (node_id, run_id, not_after, nonce)
			VALUES ($1, $2, now() - interval '1 minute', 'n')`, nodeID, runID); err != nil {
			t.Fatalf("cannot seed the expired lease for %s: %v", runID, err)
		}
	}
	// 「회수됐다」의 신호로 Run 의 상태가 아니라 **임대가 사라진 것**을 본다.
	// Reap 는 Run 을 FAILED 로 바꾸고(중간), 임대를 지우는 것으로 끝낸다(마지막).
	// 상태만 보면 그 사이에서 취소가 들어가 마지막 단계가 안 돌 수 있고,
	// 그러면 아래 I2 단언이 경합으로 깨진다. 임대가 사라졌다는 것은 그 바퀴가
	// 끝까지 돌았다는 뜻이다.
	reclaimed := func(runID string) bool {
		t.Helper()
		var state string
		var leases int
		if err := st.pool.QueryRow(seed, `
			SELECT r.state, (SELECT count(*) FROM leases l WHERE l.run_id = r.run_id)
			  FROM runs r WHERE r.run_id = $1`, runID).Scan(&state, &leases); err != nil {
			t.Fatalf("cannot read %s back: %v", runID, err)
		}
		return state == StateFailed && leases == 0
	}
	await := func(runID, what string, cancel context.CancelFunc, done <-chan struct{}) {
		t.Helper()
		deadline := time.Now().Add(20 * time.Second)
		for !reclaimed(runID) {
			if time.Now().After(deadline) {
				cancel()
				<-done
				t.Fatalf("%s: the reaper never reclaimed %s", what, runID)
			}
			time.Sleep(2 * time.Millisecond)
		}
	}

	plant("reaper-loop-1", "loop-node-1")

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		st.RunReaper(ctx, 5*time.Millisecond, quietLogger())
	}()

	// 첫 바퀴 — 재시작 스캔이다. 시작할 때 한 번 먼저 돈다.
	await("reaper-loop-1", "restart scan", cancel, done)

	// 여기서 심는 것은 첫 바퀴가 이미 지나간 뒤다. 재예약이 없으면 안 잡힌다.
	plant("reaper-loop-2", "loop-node-2")
	await("reaper-loop-2", "second cycle", cancel, done)

	// 두 바퀴를 돈 지금도 RunReaper 는 살아 있어야 한다 — 취소가 끝낸다.
	select {
	case <-done:
		t.Fatal("RunReaper returned before cancellation; the loop must outlive its scans")
	default:
	}

	cancel()
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("RunReaper did not return after cancellation")
	}

	// I2 — 종료 상태의 Run 은 점유 장부에 남지 않는다.
	var leases int
	if err := st.pool.QueryRow(seed, `SELECT count(*) FROM leases`).Scan(&leases); err != nil {
		t.Fatalf("cannot count leases: %v", err)
	}
	if leases != 0 {
		t.Fatalf("I2: the ledger must be empty after the runs ended: got %d leases, want 0", leases)
	}
}
