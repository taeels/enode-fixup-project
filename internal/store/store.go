// Package store 는 Mediator 의 상태를 PostgreSQL 에 둔다 (ADR-015 §3).
//
// 매칭 로직은 여기 없다 — ADR-014 결정 3 이 매처를 순수 함수로 못 박았다.
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
	// DB 가 아니다 — I4(봉인)를 파일시스템은 강제할 수 있고 행은 못 한다.
	Records *record.Store
	// NotifyURL 은 되묻기 알림 웹훅이다 (ADR-032 §4). 비면 알림 없음.
	// 푸시는 보조다 — 인박스(GET /v1/asks)가 정본이고, 유실돼도 재시도 없다.
	NotifyURL string
	// AnswerPath 는 알림에 실을 응답 지점의 경로를 짓는다 (ADR-032 §4).
	//
	// 상태 층은 HTTP 를 모른다 — 라우트를 여기서 지으면 등록하는 곳과
	// 발행하는 곳이 둘로 갈려 한쪽만 바뀌어도 아무도 모른다. 그래서 경로는
	// HTTP 표면을 소유한 쪽(internal/api)이 주입한다 (FR3.2).
	//
	// 비면 알림에 직링크가 없다. 그것으로 충분하다 — 인박스가 정본이고
	// 직링크는 편의다. 없는 경로를 지어 보내는 것보다 낫다.
	AnswerPath func(runID string, seq int) string
	// MaxLeasesPerRun 은 한 Run 이 동시에 쥘 수 있는 노드 수다 (ADR-024 §4.2).
	// 폭의 상한 — 0 이면 무제한(오늘 그대로).
	MaxLeasesPerRun int
	// MaxContractVersions 는 계약의 열이 가질 수 있는 판의 개수다 (ADR-031).
	// 계약이 못 건드리는 자리 — 종료 보장을 시스템이 쥔다. 0 이면 2.
	MaxContractVersions int
	// Log 는 되돌림처럼 밖에서 안 보이는 판단을 남기는 자리다.
	// 없으면 조용히 지나간다 — 로그가 없다고 동작이 달라지면 안 된다.
	Log *slog.Logger
	// LeaseTTL 은 대기열에서 승격될 때 발급하는 임대의 수명이다 (ADR-010).
	//
	// 제출 경로의 임대는 api 가 cfg.Lease.TTLSeconds 로 만들어 넘기지만,
	// 승격은 저장 계층 안에서 일어나 그 값을 볼 수 없다. 그래서 cmd/mediator 가
	// 같은 값을 여기 채운다 (MaxLeasesPerRun 과 같은 자리). 0 이면 설정의
	// 기본값(3600초)과 같게 둔다 — 시험이 안 채워도 임대가 즉시 만료되지 않게.
	LeaseTTL time.Duration
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
		return nil, fmt.Errorf("cannot connect to database: %w", err)
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

// 광고가 나를 수 있는 drain 정책의 어휘다 (ADR-063 §6).
// 이 셋 밖은 값이 아니라 오해이고, DrainPolicy 가 "" 로 접는다.
// 정본은 contract 의 셋이다 — 노드도 같은 셋을 보므로 한 벌만 둔다.
const (
	DrainNone       = contract.DrainNone
	DrainGraceful   = contract.DrainGraceful
	DrainAtBoundary = contract.DrainAtBoundary
)

// DrainPolicy 는 광고가 실어 온 정책 값을 어휘 안으로 접는다.
//
// 순수 함수이고 로그를 안 남긴다 — 부르는 자리가 둘이기 때문이다.
// UpsertAdvert 가 저장 전에 부르고, 응답에 「저장된 값」을 실어야 하는
// internal/api 도 같은 함수를 부른다. 두 자리가 같은 함수를 쓰는 것이
// 「응답의 drain 은 언제나 DB 에 앉은 값과 같다」를 보장하는 방법이다.
// 시그니처를 넓혀 저장된 값을 함께 돌려주는 길보다 이쪽이 싸다 —
// UpsertAdvert 의 반환은 이전 값이라 자리가 이미 차 있다.
//
// 로그는 UpsertAdvert 만 남긴다. 둘 다 남기면 광고 하나에 경고가 두 줄
// 찍히고, 그러면 기록이 사실보다 커진다.
func DrainPolicy(v string) string {
	switch v {
	case DrainNone, DrainGraceful, DrainAtBoundary:
		return v
	}
	return DrainNone
}

// UpsertAdvert 는 광고를 통째로 교체하고 이전 drain 정책을 돌려준다.
//
// 델타를 받지 않는 것이 ADR-012 이고, 그래서 capability 를 빼고 보내는 것이
// 곧 "지금은 못 한다" 가 된다 (ADR-017 결정 3). 병합하면 그 뜻이 사라진다.
// 정책도 같은 규칙이다 — policy 키가 없는 광고는 "안 걸려 있다" 이지
// "변경 없음" 이 아니다.
//
// 이전 값을 돌려주는 이유는 obs 가 쓰지 않는다. WakeQueued 를 부르는 지점
// 여섯 중 하나가 「drain 해제를 받은 광고 처리」인데, 덮어쓰기만 하면 해제를
// 본 사람이 아무도 없다. W0 이 버릴 값을 세워 두는 것이고, 그래야 drain 이
// 이 파일을 다시 열지 않는다.
//
// 어휘 검사가 핸들러가 아니라 여기 있는 이유는 둘이다 — 이 열에 쓰는 자리가
// 하나뿐이라 그 자리에 붙이면 앞으로 생길 다른 쓰기도 함께 걸리고,
// 어휘 밖 문자열이 앉으면 queue 의 DrainingNodes 가 그 노드를 매칭 후보에서
// 조용히 뺀다. 증상은 「능력은 있는데 계속 QUEUED」이고 원인은 눈으로만 보인다.
// 400 을 내지 않는 이유는 광고가 하트비트를 겸하기 때문이다 (ADR-016) —
// 정책 값 하나로 광고 전체를 거절하면 노드가 함대에서 사라진다.
func (s *Store) UpsertAdvert(ctx context.Context, a contract.Advert, principal string, ttl time.Duration) (prevDrain string, err error) {
	caps, err := json.Marshal(a.Capabilities)
	if err != nil {
		return "", err
	}
	drain := DrainPolicy(a.Policy.Drain)
	if drain != a.Policy.Drain {
		// 값 원문을 안 찍는다 (SECURITY-03) — 광고 본문은 밖에서 온 것이고
		// 로그는 그것을 그대로 담는 자리가 아니다. 어느 노드가 무엇을 겪었는지는
		// node 하나로 충분히 찾아진다.
		s.log().Warn("unknown drain policy folded to none", "node", a.NodeID)
	}
	// coalesce 가 서브쿼리 밖이라는 것이 이 문장의 전부다.
	//
	// (SELECT coalesce(draining,'') FROM old) 로 안쪽에 넣으면 old 가 0행일 때
	// 여전히 SQL NULL 이 나온다 — coalesce 는 행이 있을 때의 열 값만 접는다.
	// 그리고 old 가 0행인 것은 사고가 아니라 「이 노드의 첫 광고」다.
	// 그러니 안쪽에 넣으면 노드가 함대에 처음 들어오는 경로가 통째로 죽는다.
	//
	// RETURNING OLD.* 는 PostgreSQL 18 부터다. 여기는 17 이라 CTE 로 받는다.
	// CTE 는 문장 시작 시점의 스냅숏을 보므로 같은 문장의 INSERT 가 쓴 값이
	// 아니라 그 앞의 값이 나온다 — 그것이 필요한 값이다.
	err = s.pool.QueryRow(ctx, `
		WITH old AS (SELECT draining FROM nodes WHERE node_id = $1)
		INSERT INTO nodes (node_id, label, principal, capabilities, expires_at, seen_at, instance, draining)
		VALUES ($1,$2,$3,$4, now() + $5::interval, now(), $6, $7)
		ON CONFLICT (node_id) DO UPDATE SET
			label = EXCLUDED.label, principal = EXCLUDED.principal,
			capabilities = EXCLUDED.capabilities,
			expires_at = EXCLUDED.expires_at, seen_at = now(),
			instance = EXCLUDED.instance, draining = EXCLUDED.draining
		RETURNING coalesce((SELECT draining FROM old), '')`,
		a.NodeID, a.Label, principal, caps, ttl.String(), nullable(a.Instance), drain).
		Scan(&prevDrain)
	return prevDrain, err
}

// LiveAdverts 는 만료되지 않은 광고를 돌려준다.
// 살아 있음의 신탁이 아니다 — 죽었지만 아직 만료 안 된 노드가 들어 있다.
func (s *Store) LiveAdverts(ctx context.Context) ([]contract.Advert, error) {
	return liveAdvertsIn(ctx, s.pool)
}

// querier 는 pool 과 tx 가 함께 만족하는 읽기 겉면이다.
//
// 대기열 훑기는 부르는 트랜잭션 안에서 광고를 읽어야 하고(queue.go), 제출은
// pool 로 읽는다. SQL 이 두 벌이 되면 언젠가 어긋나므로 문장을 하나로 두고
// 어디서 읽을지만 갈아 끼운다.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func liveAdvertsIn(ctx context.Context, q querier) ([]contract.Advert, error) {
	rows, err := q.Query(ctx,
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
	// Submitter 는 제출자의 표시 라벨이다 — 게스트 로그인 이름이 여기 앉는다.
	//
	// 권한이 아니다. Principal 과 같은 성격의 자기 신고이고(ADR-015 §1),
	// 그래서 RunFilter 에 이 열의 필터가 없다. 실 함대의 Run 은 언제나 "" 다.
	Submitter string
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
	// QUEUED 는 후보는 있는데 전부 점유돼 기다리는 중이다 (ADR-064).
	// 임대 0 · 노드 0 · 재기동 후에도 그대로 참이다 (INVARIANTS §1.1).
	StateQueued    = "QUEUED"
	StateRunning   = "RUNNING"
	StateVerifying = "VERIFYING"
	StateSucceeded = "SUCCEEDED"
	StateFailed    = "FAILED"
)

var ErrNotFound = errors.New("not found")

// GetRun 은 없으면 ErrNotFound 다.
// liveContract 는 지금 유효한 계약을 주는 SQL 조각이다.
//
// 계약은 실행 중에 자란다(expands · 재계획). runs.contract 는 제출 전문 그대로
// 남고(성질 4), 붙은 판은 runs.contract_versions 에 쌓인다.
// ⇒ 실행하는 쪽은 마지막 판을 봐야 한다 — 제출본만 보면 계획이 지은 단계를
// 집을 때 "명령 단계인데 run 이 비었다" 가 된다. 실측이 그렇게 밟았다.
//
// 봉인만 제출 전문을 본다 (seal.go) — v1 이 제출본이어야 하기 때문이다.
// 판이 평평하게 임베드돼 있어(ContractVersion) 그대로 계약으로 읽힌다.
const liveContract = `coalesce(contract_versions -> -1, contract)`

// nullable 은 빈 문자열을 NULL 로 보낸다 — "" 와 "not found" 를 섞지 않는다.
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

// ErrNodeTaken 은 기본키 충돌이다 — 그 사이 다른 Run 이 노드를 가져갔다.
// 호출자는 409 로 답하고 전체를 롤백한다. 그것이 I5 다.
var ErrNodeTaken = errors.New("node is already held by another run")

// CreateRun 은 하나의 트랜잭션 안에서 Run · 임대 · 단계를 만든다.
//
// I5(전부 아니면 전무)를 애플리케이션 루프가 아니라 트랜잭션으로 얻는다.
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

	// Work 의 키를 여기서 박는다 (ADR-023 §6.5.2) — 계약에 id 가 없으면
	// (system, change_id) 에서 유도한다. Run 을 넘어 사는 유일한 식별자다.
	if _, err := tx.Exec(ctx,
		`INSERT INTO runs (run_id, state, principal, contract, assigned, work_id, submitter)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		r.RunID, r.State, r.Principal, contractJSON, assignedJSON,
		nullable(r.Contract.Work.Key()), r.Submitter); err != nil {
		return err
	}
	raisedAsks, err := s.createRunIn(ctx, tx, r, grants, steps)
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	// 알림은 커밋 뒤에만 — 트랜잭션 안에서 쏘면 롤백된 질문을 알리게 된다.
	s.PushAsks(raisedAsks)
	return nil
}

// createRunIn 은 runs 행이 이미 있는 Run 에 임대 · 단계 · 첫 획득 · 첫 되묻기 ·
// Record 디렉터리를 붙인다. CreateRun 의 몸통이고, 대기열 승격(queue.go)이
// 같은 몸통을 쓴다 — 제출로 도는 Run 과 기다렸다 도는 Run 이 다른 코드로
// 만들어지면 둘은 언젠가 다르게 돈다.
//
// 커밋하지 않는다. 돌려주는 되묻기는 부르는 쪽이 커밋한 뒤 알린다 (ADR-032 §4).
func (s *Store) createRunIn(ctx context.Context, tx pgx.Tx, r Run, grants []LeaseGrant, steps []contract.Step) ([]AskEvent, error) {
	for _, g := range grants {
		_, err := tx.Exec(ctx,
			`INSERT INTO leases (node_id, run_id, not_after, nonce) VALUES ($1,$2,$3,$4)`,
			g.NodeID, r.RunID, g.NotAfter, g.Nonce)
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("%w: %s", ErrNodeTaken, g.NodeID)
		}
		if err != nil {
			return nil, err
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
			return nil, err
		}
		// 기본값을 여기서 채워 넣는다 (ADR-023 §4) — needs 를 안 적은 계약은
		// [직전 단계] 가 되어 오늘과 똑같이 돈다. 계약 전문은 안 바꾼다:
		// 정규화 결과는 파생인 steps 행에만 산다 (ADR-005 성질 4).
		if _, err := tx.Exec(ctx,
			`INSERT INTO steps (run_id, seq, name, uses, kind, state, node_id, needs)
			 VALUES ($1,$2,$3,$4,$5,'PENDING',$6,$7)`,
			r.RunID, i+1, st.ID, st.Uses, kind.String(), nodeOf[st.Uses],
			contract.NeedsOf(steps, i)); err != nil {
			return nil, err
		}
	}
	// 첫 단계가 획득일 수 있다 — 그러면 아무도 보고하기 전에 수행해야 한다.
	// 노드에 안 가므로 claim 을 기다릴 수 없고, 여기서 안 하면 Run 이 멈춘다.
	if err := s.runAcquires(ctx, tx, r.RunID); err != nil {
		return nil, err
	}
	// 첫 단계가 되묻기일 수 있다 — 제출 즉시 물을 것은 물어야 한다.
	raisedAsks, err := s.raiseAsks(ctx, tx, r.RunID)
	if err != nil {
		return nil, err
	}
	// 실행 중에는 붙이기만 한다 (성질 1: append-only) —
	// 디렉터리를 지금 열어두고 로그가 쌓이게 한다. 봉인은 종료 시 한 번뿐이다.
	//
	// 커밋 전에 연다 — 커밋 뒤에 열다 실패하면 Run 은 이미 있는데
	// 호출자는 에러를 받아 상태가 갈린다.
	if s.Records != nil {
		if err := s.Records.Open(r.RunID); err != nil {
			return nil, err
		}
	}
	return raisedAsks, nil
}

// CreateRejectedRun 은 매칭이 거절된 Run 을 기록한다.
// 거절도 남긴다 — ADR-005 가 "실패 원인이 Record 에 있다" 고 했고,
// 왜 안 돌았는지가 어디에도 없으면 껍데기가 다시 제출할지를 못 정한다.
//
// submitter 도 여기 든다. 정상 경로에만 더하면 거절된 Run 의 제출자 이름이
// 영원히 비고, 데모에서 가장 흔한 실패가 「보드가 꺼져 있어 422」다 —
// 목록에서 이름 없이 뜨는 행이 바로 그 행이 된다.
//
// assigned 는 여전히 INSERT 목록에 없다. 그것이 의도다 — 열이 SQL NULL 로
// 남아야 읽기 쪽의 coalesce 가 [] 로 접는다.
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
		`INSERT INTO runs (run_id, state, principal, contract, reject, ended_at, submitter)
		 VALUES ($1,$2,$3,$4,$5, now(), $6)`,
		r.RunID, StateFailed, r.Principal, contractJSON, rejectJSON, r.Submitter)
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

// Capabilities 는 함대의 속성 어휘를 돌려준다.
//
// 존재는 답하고 여유는 답하지 않는다 (ADR-014 결정 3) —
// nodes 는 총수이지 지금 비어 있는 수가 아니다. 여유를 알려주면
// 호출자가 그것을 보고 제출하는데 그 사이 다른 Run 이 가져가, 아무것도
// 보장하지 않는 확인이 된다. 그리고 first available 아래서는 지명도 못 한다.
//
// ADR-019 로 어휘가 agent.reason 하나가 됐으므로 읽는 것은 「속성 어휘」다.
// attrs 의 값 목록은 노드별 집합이 아니라 함대 전체의 합집합이다 —
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
