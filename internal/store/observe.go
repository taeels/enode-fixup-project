package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/taeels/enode/internal/contract"
)

// StepView 는 실행 중에 밖에서 보이는 단계 하나다 (ADR-025).
//
// 이것은 Record 가 아니다 — Record 는 봉인된 사실의 묶음이고 tar 이며
// 종료 후에만 있고 불변이다(I4). 이쪽은 DB 의 지금 값이고 JSON 이며
// 언제나 답하고 계속 바뀐다. 이름도 형식도 수명도 다르므로
// "봉인되지 않은 것은 Record 가 아니다" 가 그대로 산다.
//
// 그래서 본문을 안 싣는다 — 산출물 본문이나 판정을 여기 실으면
// "이걸로 충분한데 왜 Record 를 기다리나" 가 되고 그 순간 I4 가 형해화된다.
type StepView struct {
	Seq   int    `json:"seq"`
	ID    string `json:"id"`
	State string `json:"state"`
	Uses  string `json:"uses"`
	// Node 는 어느 기계에서 도는가다 — Case D 의 "서로 다른 기계였다" 를
	// 실행 중에도 읽게 한다. label 은 assigned 가 이미 든다.
	Node string `json:"node,omitempty"`
	// Needs 는 비어 있어도 내보낸다 — [] 는 "아무것도 안 기다린다" 는
	// 뜻이고 그것이 가지의 시작점을 가리킨다. 정규화된 값은 DB 에만 있으므로
	// (계약에서는 생략될 수 있다) 안 내보내면 읽는 쪽이 기본값 규칙을 추측한다.
	Needs []string `json:"needs"`
	// Attempt 는 재시도가 도는 중인지를 밖에서 알게 한다.
	// 병렬이면 회차가 가지마다 따로 돈다.
	Attempt int `json:"attempt,omitempty"`
	// Chosen 은 이 단계가 갈림길에서 골라진 적이 있는가다 (ADR-060 §3).
	//
	// false 도 싣는다 — 생략하면 SKIPPED 하나가 다시 두 가지를 뜻하게 된다.
	// 「경로가 갈려 안 갔다」와 「골랐는데 못 닿았다」를 가르려고 만든 열이므로
	// 뷰에서 기본값으로 접으면 그 열이 무의미해진다. 읽는 쪽이 키의 부재를
	// false 로 읽어 주기를 기대하는 것은 그 추측을 다시 들이는 일이다.
	Chosen    bool       `json:"chosen"`
	StartedAt *time.Time `json:"started_at,omitempty"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
}

// Steps 는 그 Run 의 단계들을 순서대로 돌려준다 (ADR-025).
//
// 폭이 1 을 넘는 순간 필요해졌다 — 순차면 "지금 ④ 단계" 한 줄로 족했지만
// 여러 가지가 동시에 살면 각각이 다른 상태에 있고, GET record 는 종료 전이면
// 409 다(I4). ⇒ 진행을 읽을 경로가 선택이 아니라 필수가 된다.
func (s *Store) Steps(ctx context.Context, runID string) ([]StepView, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT seq, name, state, uses, coalesce(node_id,''), needs, attempt,
		       chosen, started_at, ended_at
		  FROM steps WHERE run_id = $1 ORDER BY seq`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StepView{}
	for rows.Next() {
		var v StepView
		if err := rows.Scan(&v.Seq, &v.ID, &v.State, &v.Uses, &v.Node, &v.Needs,
			&v.Attempt, &v.Chosen, &v.StartedAt, &v.EndedAt); err != nil {
			return nil, err
		}
		if v.Needs == nil {
			v.Needs = []string{}
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// ── 관측 표면 (ADR-065 · ADR-069) ─────────────────────────────────────────
//
// 아래 셋은 밖에서 함대와 Run 목록을 읽는 자리다. 공통 규칙이 둘 있다.
//
//	질의가 하나다      목록과 시각이 같은 문장에서 나온다. 두 문장으로 나누면
//	                  "관측된 한 시점" 이라는 말이 그 사이에서 거짓이 된다
//	시계가 DB 것이다   lease.not_after 를 계산한 시계와 같아야 화면의
//	                  카운트다운이 안 어긋난다
//
// 매처를 안 부른다 (ADR-065) — 관측 경로에서 매칭을 돌리면 관측과 배정이
// 다시 붙고, 관측 시점과 배정 시점이 달라 그 판정은 내는 순간 낡는다.

// LeaseView 는 노드를 지금 쥐고 있는 임대다.
//
// nonce 를 안 낸다 — 허가 아티팩트의 비밀이고 관측이 쓸 값이 아니다.
// capability 도 안 낸다 — ADR-019 이후 언제나 agent.reason 이라 줄마다 같은
// 글자가 반복될 뿐이다.
type LeaseView struct {
	RunID string `json:"run_id"`
	// NotAfter 는 카운트다운이 아니다. 지나간 값이 남아 있을 수 있다 —
	// Reap 이 ASKED 단계를 가진 Run 의 임대를 회수하지 않기 때문이다(ADR-047).
	// 사람을 기다리는 동안 자원을 놓지 않는 것이 그 결정이다.
	//
	//	not_after >  observed_at   곧 풀린다. 카운트다운이 뜻을 가진다
	//	not_after <= observed_at   지났는데 남아 있다 — 사람을 기다리는 중이다
	//
	// 그 갈림을 읽는 쪽이 그린다. 응답은 저장돼 있던 사실 그대로다.
	NotAfter time.Time `json:"not_after"`
}

// NodeView 는 관측된 노드 하나다 (ADR-065 §2).
//
// principal 을 안 낸다. 사람이 읽는 식별은 Label 에 이미 있고, principal 은
// 식별이지 인증이 아니라(ADR-015 §1) 응답에 실으면 권한처럼 보인다.
type NodeView struct {
	NodeID       string                `json:"node_id"`
	Label        string                `json:"label"`
	Instance     string                `json:"instance"`
	Capabilities []contract.Capability `json:"capabilities"`
	SeenAt       time.Time             `json:"seen_at"`
	ExpiresAt    time.Time             `json:"expires_at"`
	// Lease 는 없으면 null 이다. 빈 객체를 안 낸다 —
	// "임대 없음" 과 "임대가 있는데 값이 비었다" 가 갈려야 한다.
	Lease *LeaseView `json:"lease"`
	// Draining 은 소유자 정책의 복사본이다. 걸린 노드도 만료 전이면 나온다 —
	// 「있는데 안 빌려준다」가 「죽었다」와 갈리는 자리가 이 응답이다.
	Draining string `json:"draining"`
}

// Nodes 는 지금 관측되는 함대와 그것을 관측한 시각을 돌려준다.
//
// 시각을 값으로 따로 내는 이유는 빈 함대다. observed_at 을 행에만 얹으면
// 노드가 0 일 때 값이 없는데, 이 표면은 그때도 200 과 시각을 낸다.
// 그래서 1행짜리 at CTE 를 FROM 에 놓고 노드를 거기에 LEFT JOIN 한다 —
// 행 수와 무관하게 시각이 서면서 질의는 여전히 하나다. 노드가 0 이면
// n.node_id 가 NULL 인 행 하나가 나오고 스캔이 그 행을 건너뛴다.
//
// 만료 필터도 그 시각으로 건다. now() 를 다시 부르면 필터의 기준과 응답이
// 말하는 시각이 갈리고, 그 틈만큼 not_after 대조가 뒤집힌다.
//
// 임대는 not_after 로 안 거른다. 행이 있다는 것이 곧 점유이고 BusyNodes 를
// 포함해 저장소 전체가 같은 규칙이다 (ADR-047).
func (s *Store) Nodes(ctx context.Context) ([]NodeView, time.Time, error) {
	rows, err := s.pool.Query(ctx, `
		WITH at AS (SELECT now() AS observed_at)
		SELECT at.observed_at, n.node_id, n.label, coalesce(n.instance,''), n.capabilities,
		       n.seen_at, n.expires_at, n.draining, l.run_id, l.not_after
		  FROM at
		  LEFT JOIN nodes  n ON n.expires_at > at.observed_at
		  LEFT JOIN leases l ON l.node_id = n.node_id
		 ORDER BY n.node_id`)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer rows.Close()
	out := []NodeView{}
	var observed time.Time
	for rows.Next() {
		// 노드 쪽 열은 전부 NULL 일 수 있다 — 빈 함대의 그 한 행 때문이다.
		// 열이 NOT NULL 이어도 LEFT JOIN 이 없는 쪽을 NULL 로 채운다.
		var nodeID, label, draining, runID *string
		var seenAt, expiresAt, notAfter *time.Time
		var instance string
		var caps []byte
		var v NodeView
		if err := rows.Scan(&observed, &nodeID, &label, &instance, &caps,
			&seenAt, &expiresAt, &draining, &runID, &notAfter); err != nil {
			return nil, time.Time{}, err
		}
		if nodeID == nil {
			continue // 빈 함대. 시각은 이미 받았다
		}
		v.NodeID = *nodeID
		v.Instance = instance
		if label != nil {
			v.Label = *label
		}
		if draining != nil {
			v.Draining = *draining
		}
		if seenAt != nil {
			v.SeenAt = *seenAt
		}
		if expiresAt != nil {
			v.ExpiresAt = *expiresAt
		}
		if len(caps) > 0 {
			if err := json.Unmarshal(caps, &v.Capabilities); err != nil {
				return nil, time.Time{}, err
			}
		}
		// null 로 나가지 않게 한다 — 빈 배열이 "가진 것이 없다" 이고
		// null 은 읽는 쪽에 기본값 규칙을 추측하게 만든다.
		if v.Capabilities == nil {
			v.Capabilities = []contract.Capability{}
		}
		if runID != nil && notAfter != nil {
			v.Lease = &LeaseView{RunID: *runID, NotAfter: *notAfter}
		}
		out = append(out, v)
	}
	return out, observed, rows.Err()
}

// RunRow 는 Run 목록의 한 줄이다.
//
// steps 를 안 싣는다 — 하나를 들여다보는 것은 상세이고 그쪽 라우트가 있다.
type RunRow struct {
	RunID string `json:"run_id"`
	State string `json:"state"`
	// Verdict 는 종료 전이면 null 이다. 목록에 싣는 이유는 하나다 —
	// drain 으로 닫힌 FAILED 와 진짜 실패를 목록에서 가르는 열이 그것뿐이다.
	Verdict *Verdict `json:"verdict"`
	WorkID  string   `json:"work_id"`
	// CreatedAt 은 정렬 키다.
	CreatedAt time.Time  `json:"created_at"`
	EndedAt   *time.Time `json:"ended_at"`
	// Assigned 는 값이 없으면 빈 배열이다. null 도 키 생략도 아니다 —
	// 「QUEUED 행이 어느 카드에도 안 얹혀 있다」를 화면이 이 값으로 판정한다.
	Assigned []Assigned `json:"assigned"`
	// Submitter 는 제출자 표시 라벨이고 키는 언제나 있다.
	// 실 함대 Run 은 "" 다 — verdict 와 ended_at 이 종료 전에 null 로 나가는
	// 것과 같은 관행이다.
	Submitter string `json:"submitter"`
}

// RunFilter 는 목록을 좁히기만 한다. 넓히는 인자가 없다.
//
// principal 필터도 node 필터도 없다. 앞은 ADR-065 와 같은 이유이고
// (거르는 값으로 내면 그 필드가 권한처럼 보인다), 뒤는 화면이
// assigned[].nodes[].node 로 거르기로 이미 정해져 있다.
type RunFilter struct {
	State string
	// Since 는 포함 하한이다 — created_at >= Since.
	// 열려 있으면 같은 초에 만들어진 Run 이 폴링 사이에 사라진다.
	// 빈 값의 판정은 IsZero() 다.
	Since time.Time
	Work  string
	// Limit 은 받은 수를 그대로 건다. 기본 100 과 상한 1000 은 internal/api 가
	// 질의 문자열을 읽으며 적용한다 — 두 곳에서 정하면 어긋났을 때
	// 화면이 이유 없는 결과를 받는다.
	Limit int
}

// fallbackRunsLimit 은 정책이 아니라 바닥이다. 정책은 api 가 쥔다.
const fallbackRunsLimit = 100

// Runs 는 목록과 그것을 관측한 시각을 돌려준다.
//
// 최신이 앞이다 — 이 자리는 "지금 무엇이 도는가" 를 답한다. 지난 것을
// 뒤지는 자리는 Record 다. 페이지네이션이 없다: limit 뿐이다.
//
// LIMIT 이 LATERAL 안쪽에 있어야 한다 — 밖에 걸면 시각 행까지 세어 한 줄이
// 모자란다. LATERAL 인 이유는 안쪽이 바깥 행을 봐서가 아니라 0행일 때도
// 왼쪽 한 줄이 남아야 하기 때문이고, 그것이 Nodes 와 같은 모양이다.
//
// ?work= 는 접기 전 열에 등호를 건다. 응답은 coalesce(work_id, "") 로 접어
// 내지만 필터는 raw 열이라 runs_work_idx 를 탄다. 대가는 하나다 —
// work_id 가 NULL 인 옛 행은 응답에 "" 로 보이면서 어떤 ?work= 값으로도
// 안 잡힌다. "" 로 거르는 것은 「work 가 없는 것 전부」라 뜻이 없는 질의이고,
// 그 하나를 위해 인덱스를 버릴 이유가 없다.
//
// coalesce(assigned,'[]'::jsonb) 는 SQL NULL 만 접는다. jsonb 의 null 은
// 값이라 그대로 통과해 와이어에 "assigned": null 로 나간다. 읽기 쪽에서
// 막을 수 없으므로 쓰는 쪽의 계약으로 둔다 — INSERT 는 열을 생략하거나
// [] 를 넣는다. 앞으로 생길 CreateQueuedRun 이 store.Run{} 제로값을 그대로
// 쓰면 nil 슬라이스가 null 로 마셜돼 그것이 앉는다.
func (s *Store) Runs(ctx context.Context, f RunFilter) ([]RunRow, time.Time, error) {
	// api 가 1 이상을 보장한다. 이 바닥은 그 보장 밖에서 부르는 자리를 위한
	// 것이고, 없으면 제로값 RunFilter 가 LIMIT 0 을 만들어 "Run 이 없다" 로
	// 보이는 거짓말을 내거나 음수가 SQL 오류가 된다. 수를 고르는 자리가
	// 아니므로 값은 api 의 기본과 같은 수로 둔다.
	limit := f.Limit
	if limit <= 0 {
		limit = fallbackRunsLimit
	}
	// 빈 시각은 NULL 로 보낸다 — 제로 시각을 그대로 보내면 하한이
	// 서기 1년이 되어 필터가 있는 것처럼 굴면서 아무것도 안 거른다.
	var since any
	if !f.Since.IsZero() {
		since = f.Since
	}
	rows, err := s.pool.Query(ctx, `
		WITH at AS (SELECT now() AS observed_at)
		SELECT at.observed_at, r.run_id, r.state, r.verdict, r.work_id, r.created_at,
		       r.ended_at, r.assigned, r.submitter
		  FROM at
		  LEFT JOIN LATERAL (
			SELECT run_id, state, verdict, coalesce(work_id,'') AS work_id, created_at,
			       ended_at, coalesce(assigned,'[]'::jsonb) AS assigned, submitter
			  FROM runs
			 WHERE ($1 = '' OR state = $1)
			   AND ($2::timestamptz IS NULL OR created_at >= $2)
			   AND ($3 = '' OR work_id = $3)
			 ORDER BY created_at DESC
			 LIMIT $4
		  ) r ON true`, f.State, since, f.Work, limit)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer rows.Close()
	out := []RunRow{}
	var observed time.Time
	for rows.Next() {
		// Nodes 와 같은 이유로 전부 NULL 일 수 있다 — 아무것도 안 맞는
		// 필터가 왼쪽 한 줄만 남긴다.
		var runID, state, workID, submitter *string
		var createdAt, endedAt *time.Time
		var verdictJSON, assignedJSON []byte
		if err := rows.Scan(&observed, &runID, &state, &verdictJSON, &workID,
			&createdAt, &endedAt, &assignedJSON, &submitter); err != nil {
			return nil, time.Time{}, err
		}
		if runID == nil {
			continue // 맞는 Run 이 없다. 시각은 이미 받았다
		}
		v := RunRow{RunID: *runID, EndedAt: endedAt, Assigned: []Assigned{}}
		if state != nil {
			v.State = *state
		}
		if workID != nil {
			v.WorkID = *workID
		}
		if submitter != nil {
			v.Submitter = *submitter
		}
		if createdAt != nil {
			v.CreatedAt = *createdAt
		}
		if len(verdictJSON) > 0 {
			if err := json.Unmarshal(verdictJSON, &v.Verdict); err != nil {
				return nil, time.Time{}, err
			}
		}
		if len(assignedJSON) > 0 {
			if err := json.Unmarshal(assignedJSON, &v.Assigned); err != nil {
				return nil, time.Time{}, err
			}
		}
		if v.Assigned == nil {
			v.Assigned = []Assigned{}
		}
		out = append(out, v)
	}
	return out, observed, rows.Err()
}

// RequireView 는 Run 하나가 무엇을 기다리는가의 한 줄이다 (ADR-069).
//
// 이 모양은 관측 전용이다. 계약 입력의 requires[] 는 속성을 형제 키로 편다
// (contract.Require 의 Marshal/Unmarshal) — 같은 개념을 두 모양이 지고 있고
// 변환기는 코드 어디에도 없다. 관측 응답을 그대로 계약 본문에 복사하면
// requires[].attrs must be a string 이 나는데, 그 400 은 원인을 안 가리킨다.
// 화면은 읽기만 하므로 안 밟지만 mcp 의 run.submit 은 사용자가 지은 계약을
// 그대로 나르므로 밟을 수 있다.
type RequireView struct {
	// As 는 별칭이다. steps[].uses 와 이어야 어느 단계가 무엇을 기다리는지 붙는다.
	As         string `json:"as"`
	Capability string `json:"capability"`
	// Count 는 생략되는 유일한 키다. 계약 입력이 이미 그 모양이고
	// (contract.Require 의 count 가 omitempty) "없으면 1" 이 그 자리의
	// 기본값 규칙이라 정본이 이미 정해 두었다.
	Count int `json:"count,omitempty"`
	// Attrs 는 비어도 {} 를 낸다. 키를 지우면 읽는 쪽이 기본값 규칙을
	// 추측하고, 그것이 assigned 와 chosen 을 언제나 싣기로 한 이유와 같다.
	Attrs map[string]string `json:"attrs"`
}

// RequiresOf 는 계약의 요구를 관측 모양으로 옮긴다.
//
// 출처는 지금 유효한 계약이어야 한다 (LiveContract) — 제출 전문만 보면
// 재계획이 지은 요구가 빠진다.
func RequiresOf(c contract.Contract) []RequireView {
	out := make([]RequireView, 0, len(c.Requires))
	for _, r := range c.Requires {
		v := RequireView{As: r.As, Capability: r.Capability, Count: r.Count,
			Attrs: map[string]string{}}
		for k, val := range r.Attrs {
			v.Attrs[k] = val
		}
		out = append(out, v)
	}
	return out
}

// LedgerEntry 는 원장의 항목 하나다 — 본문이 없다 (ADR-023 §6.3).
type LedgerEntry struct {
	// RunID 는 scope:"work" 일 때만 채워진다 — 이 Run 밖에서 온 것이라는 표시다.
	// 같은 Run 안의 것에 붙이면 모든 줄에 같은 값이 반복될 뿐이다.
	RunID   string `json:"run_id,omitempty"`
	Seq     int    `json:"seq"`
	Attempt int    `json:"attempt"`
	Name    string `json:"name"`
	// By 는 누가 냈나다 — "step:<id>". ADR-022 §7.6 의 by 와 같은 어휘다.
	By    string    `json:"by"`
	At    time.Time `json:"at"`
	Bytes int64     `json:"bytes"`
	// SchemaOK 는 검증했는가와 통과했는가를 가른다 (ADR-020 의 status:none 과 같은 이유).
	//	nil    계약에 그 이름의 스키마가 없었다 — 검증 안 했다
	//	true   있었고 통과했다 (통과 못 하면 애초에 저장되지 않는다 — 422)
	SchemaOK *bool `json:"schema_ok,omitempty"`
}

// Ledger 는 그 Run 이 지금 발견할 수 있는 것의 목록이다 (ADR-023 §6).
//
// 시야를 순서에서 유도하지 않는다 — 조상인지 형제인지를 묻지 않고,
// 그 시점에 원장에 있는가만 본다. 그래서 병렬이 「최신」을 안 깬다.
//
// scope:"work" 면 같은 Work 의 이전 Run 들이 낸 것까지 본다.
// 이것은 ADR-005 가 범위 밖으로 둔 「여러 Run 에 걸친 질의」가 아니다 —
// 저쪽은 봉인 묶음에 대한 질의이고, 이쪽은 work_id 인덱스로 Run 을 고른 뒤
// 각자의 blobs/ 를 나열하는 것이다. 검색 계층도 질의 언어도 안 는다.
func (s *Store) Ledger(ctx context.Context, runID string) ([]LedgerEntry, error) {
	if s.Records == nil {
		return nil, fmt.Errorf("record store is not configured")
	}
	run, err := s.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	// 이름을 붙이려면 지금 유효한 계약이 필요하다 — 계획이 지은 단계가
	// 낸 산출물은 제출 전문에 없는 이름이다.
	live, err := s.LiveContract(ctx, runID)
	if err != nil {
		return nil, err
	}
	out := []LedgerEntry{}

	// 이전 Run 들이 먼저다 — 시간 순서가 곧 목록의 순서다.
	if run.Contract.Scope() == contract.ScopeWork {
		prior, err := s.priorRuns(ctx, run)
		if err != nil {
			return nil, err
		}
		for _, p := range prior {
			es, err := s.entriesOf(p.runID, p.contract, p.runID)
			if err != nil {
				continue // 지워졌거나 못 읽는 Run 이 지금 Run 을 막지 않는다
			}
			out = append(out, es...)
		}
	}
	mine, err := s.entriesOf(runID, live, "")
	if err != nil {
		return nil, err
	}
	return append(out, mine...), nil
}

// LiveContract 는 지금 유효한 계약이다 (liveContract 참조).
// 실행 경로는 전부 이것을 본다 — 제출 전문은 봉인(v1)만 쓴다.
func (s *Store) LiveContract(ctx context.Context, runID string) (contract.Contract, error) {
	var raw []byte
	var c contract.Contract
	if err := s.pool.QueryRow(ctx,
		`SELECT `+liveContract+` FROM runs WHERE run_id=$1`, runID).Scan(&raw); err != nil {
		return c, err
	}
	return c, json.Unmarshal(raw, &c)
}

type priorRun struct {
	runID    string
	contract contract.Contract
}

// priorRuns 는 같은 Work 의 앞선 Run 들이다. 자기 자신은 뺀다.
//
// work_id 가 Run 을 넘어 사는 유일한 식별자 이므로 이 조회가 성립한다
// (ADR-023 §6.5.2). 없으면 — 계약이 work 를 안 채웠으면 — 빈 목록이다.
func (s *Store) priorRuns(ctx context.Context, run *Run) ([]priorRun, error) {
	key := run.Contract.Work.Key()
	if key == "" {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT run_id, coalesce(contract_versions -> -1, contract) FROM runs
		 WHERE work_id = $1 AND run_id <> $2
		 ORDER BY created_at`, key, run.RunID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []priorRun{}
	for rows.Next() {
		var p priorRun
		var raw []byte
		if err := rows.Scan(&p.runID, &raw); err != nil {
			return nil, err
		}
		if json.Unmarshal(raw, &p.contract) != nil {
			continue
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// entriesOf 는 Run 하나의 산출물에 계약이 아는 것을 붙인다 —
// 누가 냈는지(단계 이름)와 스키마로 검증됐는지. record 는 계약을 모르므로
// 그 둘은 여기서만 붙일 수 있다.
func (s *Store) entriesOf(runID string, c contract.Contract, label string) ([]LedgerEntry, error) {
	metas, err := s.Records.Blobs(runID)
	if err != nil {
		return nil, err
	}
	out := make([]LedgerEntry, 0, len(metas))
	for _, m := range metas {
		e := LedgerEntry{RunID: label, Seq: m.Seq, Attempt: m.Attempt,
			Name: m.Name, At: m.At, Bytes: m.Bytes}
		if m.Seq >= 1 && m.Seq <= len(c.Steps) {
			st := c.Steps[m.Seq-1]
			e.By = "step:" + st.ID
			if _, ok := st.Schema[m.Name]; ok {
				ok := true
				e.SchemaOK = &ok
			}
		}
		out = append(out, e)
	}
	return out, nil
}
