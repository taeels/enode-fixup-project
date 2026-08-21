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
	"io"
	"log/slog"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/match"
	"github.com/taeels/enode/internal/record"
)

//go:embed schema.sql
var schemaSQL string

type Store struct {
	pool *pgxpool.Pool
	// Records 는 봉인된 Run Record 가 사는 곳이다 (ADR-015 §3).
	// ★ DB 가 아니다 ★ — I4(봉인)를 파일시스템은 강제할 수 있고 행은 못 한다.
	Records *record.Store
	// NotifyURL 은 되묻기 알림 웹훅이다 (ADR-032 §4). 비면 알림 없음.
	// ★ 푸시는 보조다 ★ — 인박스(GET /v1/asks)가 정본이고, 유실돼도 재시도 없다.
	NotifyURL string
	// MaxLeasesPerRun 은 한 Run 이 동시에 쥘 수 있는 노드 수다 (ADR-024 §4.2).
	// ★ 폭의 상한 ★ — 0 이면 무제한(오늘 그대로).
	MaxLeasesPerRun int
	// MaxContractVersions 는 계약의 열이 가질 수 있는 판의 개수다 (ADR-031).
	// ★ 계약이 못 건드리는 자리 ★ — 종료 보장을 시스템이 쥔다. 0 이면 2.
	MaxContractVersions int
	// Log 는 ★ 되돌림처럼 밖에서 안 보이는 판단 ★ 을 남기는 자리다.
	// 없으면 조용히 지나간다 — 로그가 없다고 동작이 달라지면 안 된다.
	Log *slog.Logger
}

func (s *Store) log() *slog.Logger {
	if s.Log == nil {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return s.Log
}

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
		INSERT INTO nodes (node_id, label, principal, capabilities, expires_at, seen_at, instance)
		VALUES ($1,$2,$3,$4, now() + $5::interval, now(), $6)
		ON CONFLICT (node_id) DO UPDATE SET
			label = EXCLUDED.label, principal = EXCLUDED.principal,
			capabilities = EXCLUDED.capabilities,
			expires_at = EXCLUDED.expires_at, seen_at = now(),
			instance = EXCLUDED.instance`,
		a.NodeID, a.Label, principal, caps, ttl.String(), nullable(a.Instance))
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
// liveContract 는 ★ 지금 유효한 계약 ★ 을 주는 SQL 조각이다.
//
// 계약은 실행 중에 자란다(expands · 재계획). ★ runs.contract 는 제출 전문 그대로 ★
// 남고(성질 4), 붙은 판은 runs.contract_versions 에 쌓인다.
// ⇒ ★ 실행하는 쪽은 마지막 판을 봐야 한다 ★ — 제출본만 보면 계획이 지은 단계를
// 집을 때 "명령 단계인데 run 이 비었다" 가 된다. ★ 실측이 그렇게 밟았다 ★.
//
// ★ 봉인만 제출 전문을 본다 ★ (seal.go) — v1 이 제출본이어야 하기 때문이다.
// 판이 평평하게 임베드돼 있어(ContractVersion) 그대로 계약으로 읽힌다.
const liveContract = `coalesce(contract_versions -> -1, contract)`

// nullable 은 빈 문자열을 NULL 로 보낸다 — ★ "" 와 "없다" 를 섞지 않는다 ★.
func nullable(v string) any {
	if v == "" {
		return nil
	}
	return v
}

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

	// ★ Work 의 키를 여기서 박는다 ★ (ADR-023 §6.5.2) — 계약에 id 가 없으면
	// (system, change_id) 에서 유도한다. Run 을 넘어 사는 유일한 식별자다.
	if _, err := tx.Exec(ctx,
		`INSERT INTO runs (run_id, state, principal, contract, assigned, work_id)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		r.RunID, r.State, r.Principal, contractJSON, assignedJSON,
		nullable(r.Contract.Work.Key())); err != nil {
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
		// ★ 기본값을 여기서 채워 넣는다 ★ (ADR-023 §4) — needs 를 안 적은 계약은
		// [직전 단계] 가 되어 오늘과 똑같이 돈다. 계약 전문은 안 바꾼다:
		// 정규화 결과는 파생인 steps 행에만 산다 (ADR-005 성질 4).
		if _, err := tx.Exec(ctx,
			`INSERT INTO steps (run_id, seq, name, uses, kind, state, node_id, needs)
			 VALUES ($1,$2,$3,$4,$5,'PENDING',$6,$7)`,
			r.RunID, i+1, st.ID, st.Uses, kind.String(), nodeOf[st.Uses],
			contract.NeedsOf(steps, i)); err != nil {
			return err
		}
	}
	// ★ 첫 단계가 획득일 수 있다 ★ — 그러면 아무도 보고하기 전에 수행해야 한다.
	// 노드에 안 가므로 claim 을 기다릴 수 없고, 여기서 안 하면 Run 이 멈춘다.
	if err := s.runAcquires(ctx, tx, r.RunID); err != nil {
		return err
	}
	// ★ 첫 단계가 되묻기일 수 있다 ★ — 제출 즉시 물을 것은 물어야 한다.
	raisedAsks, err := s.raiseAsks(ctx, tx, r.RunID)
	if err != nil {
		return err
	}
	// ★ 실행 중에는 붙이기만 한다 ★ (성질 1: append-only) —
	// 디렉터리를 지금 열어두고 로그가 쌓이게 한다. 봉인은 종료 시 한 번뿐이다.
	//
	// ★ 커밋 전에 연다 ★ — 커밋 뒤에 열다 실패하면 Run 은 이미 있는데
	// 호출자는 에러를 받아 상태가 갈린다.
	if s.Records != nil {
		if err := s.Records.Open(r.RunID); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	// ★ 알림은 커밋 뒤에만 ★ — 트랜잭션 안에서 쏘면 롤백된 질문을 알리게 된다.
	s.PushAsks(raisedAsks)
	return nil
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

// CapabilityView 는 GET /v1/capabilities 의 한 줄이다 (ADR-014 결정 3).
type CapabilityView struct {
	Capability string              `json:"capability"`
	Nodes      int                 `json:"nodes"`
	Attrs      map[string][]string `json:"attrs"`
}

// Capabilities 는 ★ 함대의 속성 어휘 ★ 를 돌려준다.
//
// ★ 존재는 답하고 여유는 답하지 않는다 ★ (ADR-014 결정 3) —
// nodes 는 ★ 총수 ★ 이지 지금 비어 있는 수가 아니다. 여유를 알려주면
// 호출자가 그것을 보고 제출하는데 그 사이 다른 Run 이 가져가, 아무것도
// 보장하지 않는 확인이 된다. 그리고 first available 아래서는 지명도 못 한다.
//
// ADR-019 로 어휘가 agent.reason 하나가 됐으므로 ★ 읽는 것은 「속성 어휘」 ★ 다.
// attrs 의 값 목록은 노드별 집합이 아니라 ★ 함대 전체의 합집합 ★ 이다 —
// "보드가 있고 armv7 을 빌드하는 노드가 몇 개인가" 는 여기서 못 센다.
// 그 답은 POST /v1/runs/dry-run 이 준다 (매처를 두 벌 만들지 않는다).
func (s *Store) Capabilities(ctx context.Context) ([]CapabilityView, error) {
	adverts, err := s.LiveAdverts(ctx)
	if err != nil {
		return nil, err
	}
	byCap := map[string]*CapabilityView{}
	seen := map[string]map[string]map[string]bool{} // cap → key → value
	for _, a := range adverts {
		for _, c := range a.Capabilities {
			v, ok := byCap[c.Capability]
			if !ok {
				v = &CapabilityView{Capability: c.Capability, Attrs: map[string][]string{}}
				byCap[c.Capability] = v
				seen[c.Capability] = map[string]map[string]bool{}
			}
			v.Nodes++
			for k, val := range c.Attrs {
				if seen[c.Capability][k] == nil {
					seen[c.Capability][k] = map[string]bool{}
				}
				if !seen[c.Capability][k][val] {
					seen[c.Capability][k][val] = true
					v.Attrs[k] = append(v.Attrs[k], val)
				}
			}
		}
	}
	out := make([]CapabilityView, 0, len(byCap))
	names := make([]string, 0, len(byCap))
	for n := range byCap {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		v := byCap[n]
		for k := range v.Attrs {
			sort.Strings(v.Attrs[k])
		}
		out = append(out, *v)
	}
	return out, nil
}
