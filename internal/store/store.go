// Package store 는 Mediator 의 상태를 PostgreSQL 에 둔다 (ADR-015 §3).
//
// ★ 매칭 로직은 여기 없다 ★ — ADR-014 결정 3 이 매처를 순수 함수로 못 박았다.
// 이 패키지는 광고와 점유를 읽어주고, 결정된 배정을 트랜잭션으로 굳힐 뿐이다.
package store

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/match"
)

//go:embed schema.sql
var schemaSQL string

type Store struct{ pool *pgxpool.Pool }

func Open(ctx context.Context, url string) (*Store, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("DB 연결 실패: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

// Migrate 는 스키마를 멱등하게 적용한다.
// MVP 는 마이그레이션 도구를 안 쓴다 — 스키마가 굳으면 그때 넣는다.
func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, schemaSQL)
	return err
}

// ── 노드 광고 ────────────────────────────────────────────────────────────

// UpsertAdvert 는 광고를 ★ 통째로 교체한다 ★.
// 델타를 받지 않는 것이 ADR-012 이고, 그래서 capability 를 빼고 보내는 것이
// 곧 "지금은 못 한다" 가 된다 (ADR-017 결정 3). 병합하면 그 뜻이 사라진다.
func (s *Store) UpsertAdvert(ctx context.Context, a contract.Advert, principal string, ttl time.Duration) error {
	caps, err := json.Marshal(a.Capabilities)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO nodes (node_id, label, principal, capabilities, expires_at, seen_at)
		VALUES ($1,$2,$3,$4, now() + $5::interval, now())
		ON CONFLICT (node_id) DO UPDATE SET
			label = EXCLUDED.label, principal = EXCLUDED.principal,
			capabilities = EXCLUDED.capabilities,
			expires_at = EXCLUDED.expires_at, seen_at = now()`,
		a.NodeID, a.Label, principal, caps, ttl.String())
	return err
}

// LiveAdverts 는 ★ 만료되지 않은 ★ 광고를 돌려준다.
// 살아 있음의 신탁이 아니다 — 죽었지만 아직 만료 안 된 노드가 들어 있다.
func (s *Store) LiveAdverts(ctx context.Context) ([]contract.Advert, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT node_id, label, capabilities FROM nodes WHERE expires_at > now() ORDER BY node_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []contract.Advert
	for rows.Next() {
		var a contract.Advert
		var caps []byte
		if err := rows.Scan(&a.NodeID, &a.Label, &caps); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(caps, &a.Capabilities); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ── Run ──────────────────────────────────────────────────────────────────

type Run struct {
	RunID     string
	State     string
	Principal string
	Contract  contract.Contract
	Assigned  []Assigned
	Reject    *match.Reject
	Verdict   *Verdict
	CreatedAt time.Time
}

type Assigned struct {
	As    string    `json:"as"`
	Nodes []NodeRef `json:"nodes"`
}

type NodeRef struct {
	Node  string `json:"node"`
	Label string `json:"label"`
}

// 상태 이름은 INVARIANTS §1.1 그대로다.
const (
	StateResolving  = "RESOLVING"
	StateAllocating = "ALLOCATING"
	StateRunning    = "RUNNING"
	StateVerifying  = "VERIFYING"
	StateSucceeded  = "SUCCEEDED"
	StateFailed     = "FAILED"
)

var ErrNotFound = errors.New("없다")

// GetRun 은 없으면 ErrNotFound 다.
func (s *Store) GetRun(ctx context.Context, runID string) (*Run, error) {
	var r Run
	var contractJSON, assignedJSON, rejectJSON, verdictJSON []byte
	err := s.pool.QueryRow(ctx,
		`SELECT run_id, state, principal, contract, assigned, reject, verdict, created_at
		 FROM runs WHERE run_id = $1`, runID).
		Scan(&r.RunID, &r.State, &r.Principal, &contractJSON, &assignedJSON, &rejectJSON,
			&verdictJSON, &r.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(contractJSON, &r.Contract); err != nil {
		return nil, err
	}
	if len(assignedJSON) > 0 {
		if err := json.Unmarshal(assignedJSON, &r.Assigned); err != nil {
			return nil, err
		}
	}
	if len(rejectJSON) > 0 {
		if err := json.Unmarshal(rejectJSON, &r.Reject); err != nil {
			return nil, err
		}
	}
	if len(verdictJSON) > 0 {
		if err := json.Unmarshal(verdictJSON, &r.Verdict); err != nil {
			return nil, err
		}
	}
	return &r, nil
}

// LeaseGrant 는 배정 하나에 발급할 임대다.
type LeaseGrant struct {
	NodeID   string
	NotAfter time.Time
	Nonce    string
}

// ErrNodeTaken 은 ★ 기본키 충돌 ★ 이다 — 그 사이 다른 Run 이 노드를 가져갔다.
// 호출자는 409 로 답하고 전체를 롤백한다. 그것이 I5 다.
var ErrNodeTaken = errors.New("노드가 이미 다른 Run 에 묶여 있다")

// CreateRun 은 ★ 하나의 트랜잭션 ★ 안에서 Run · 임대 · 단계를 만든다.
//
// I5(전부 아니면 전무)를 애플리케이션 루프가 아니라 ★ 트랜잭션 ★ 으로 얻는다.
// leases 의 기본키가 node_id 이므로(ADR-019 결정 2) 경쟁하는 Run 은 INSERT 에서
// 충돌하고, 롤백이 이미 잡은 것을 전부 되돌린다 — 손으로 해제할 것이 없다.
//
// ※ claim(S4)의 SKIP LOCKED 와 다른 기계다. 저쪽은 대기열에서 하나를 집는 것이고
//
//	이쪽은 여러 자원을 한꺼번에 잡거나 전부 포기하는 것이다.
func (s *Store) CreateRun(ctx context.Context, r Run, grants []LeaseGrant, steps []contract.Step) error {
	contractJSON, err := json.Marshal(r.Contract)
	if err != nil {
		return err
	}
	assignedJSON, err := json.Marshal(r.Assigned)
	if err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // 커밋됐으면 무해하다

	if _, err := tx.Exec(ctx,
		`INSERT INTO runs (run_id, state, principal, contract, assigned) VALUES ($1,$2,$3,$4,$5)`,
		r.RunID, r.State, r.Principal, contractJSON, assignedJSON); err != nil {
		return err
	}
	for _, g := range grants {
		_, err := tx.Exec(ctx,
			`INSERT INTO leases (node_id, run_id, not_after, nonce) VALUES ($1,$2,$3,$4)`,
			g.NodeID, r.RunID, g.NotAfter, g.Nonce)
		if isUniqueViolation(err) {
			return fmt.Errorf("%w: %s", ErrNodeTaken, g.NodeID)
		}
		if err != nil {
			return err
		}
	}
	// 역할 → 노드. 매처가 이미 정했으므로 단계 행에 박아둔다.
	// claim 이 jsonb 를 뒤지지 않아도 되고, Record 의 노드 귀속이 처음부터 산다.
	nodeOf := map[string]string{}
	for _, a := range r.Assigned {
		if len(a.Nodes) > 0 {
			nodeOf[a.As] = a.Nodes[0].Node // count>1 은 아직 (run-contract §4.5)
		}
	}
	for i, st := range steps {
		kind, err := st.Kind()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO steps (run_id, seq, name, uses, kind, state, node_id)
			 VALUES ($1,$2,$3,$4,$5,'PENDING',$6)`,
			r.RunID, i+1, st.ID, st.Uses, kind.String(), nodeOf[st.Uses]); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// CreateRejectedRun 은 매칭이 거절된 Run 을 기록한다.
// ★ 거절도 남긴다 ★ — ADR-005 가 "실패 원인이 Record 에 있다" 고 했고,
// 왜 안 돌았는지가 어디에도 없으면 껍데기가 다시 제출할지를 못 정한다.
func (s *Store) CreateRejectedRun(ctx context.Context, r Run) error {
	contractJSON, err := json.Marshal(r.Contract)
	if err != nil {
		return err
	}
	rejectJSON, err := json.Marshal(r.Reject)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO runs (run_id, state, principal, contract, reject, ended_at)
		 VALUES ($1,$2,$3,$4,$5, now())`,
		r.RunID, StateFailed, r.Principal, contractJSON, rejectJSON)
	return err
}

// BusyNodes 는 지금 다른 Run 에 묶인 노드다. 매처의 입력이 된다.
func (s *Store) BusyNodes(ctx context.Context) (map[string]bool, error) {
	rows, err := s.pool.Query(ctx, `SELECT node_id FROM leases`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	busy := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		busy[id] = true
	}
	return busy, rows.Err()
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// Truncate 는 테스트 전용이다. 순서는 외래키를 따른다.
func (s *Store) Truncate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `TRUNCATE steps, leases, runs, nodes`)
	return err
}

// ForceExpire 는 테스트 전용이다 — 하트비트가 끊긴 상황을 만든다.
func (s *Store) ForceExpire(ctx context.Context, runID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE leases SET not_after = now() - interval '1s' WHERE run_id=$1`, runID)
	return err
}
