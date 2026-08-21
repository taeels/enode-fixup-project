package store

import (
	"context"
	"time"
)

// StepView 는 ★ 실행 중에 밖에서 보이는 단계 하나 ★ 다 (ADR-025).
//
// ★ 이것은 Record 가 아니다 ★ — Record 는 봉인된 사실의 묶음이고 tar 이며
// 종료 후에만 있고 불변이다(I4). 이쪽은 DB 의 ★ 지금 ★ 값이고 JSON 이며
// 언제나 답하고 ★ 계속 바뀐다 ★. 이름도 형식도 수명도 다르므로
// "봉인되지 않은 것은 Record 가 아니다" 가 그대로 산다.
//
// ★ 그래서 본문을 안 싣는다 ★ — 산출물 본문이나 판정을 여기 실으면
// "이걸로 충분한데 왜 Record 를 기다리나" 가 되고 그 순간 I4 가 형해화된다.
type StepView struct {
	Seq   int    `json:"seq"`
	ID    string `json:"id"`
	State string `json:"state"`
	Uses  string `json:"uses"`
	// Node 는 ★ 어느 기계에서 도는가 ★ 다 — Case D 의 "서로 다른 기계였다" 를
	// 실행 중에도 읽게 한다. label 은 assigned 가 이미 든다.
	Node string `json:"node,omitempty"`
	// Needs 는 ★ 비어 있어도 내보낸다 ★ — [] 는 "아무것도 안 기다린다" 는
	// 뜻이고 그것이 가지의 시작점을 가리킨다. 정규화된 값은 ★ DB 에만 있으므로 ★
	// (계약에서는 생략될 수 있다) 안 내보내면 읽는 쪽이 기본값 규칙을 추측한다.
	Needs []string `json:"needs"`
	// Attempt 는 ★ 재시도가 도는 중인지 ★ 를 밖에서 알게 한다.
	// 병렬이면 회차가 ★ 가지마다 따로 ★ 돈다.
	Attempt   int        `json:"attempt,omitempty"`
	StartedAt *time.Time `json:"started_at,omitempty"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
}

// Steps 는 그 Run 의 단계들을 순서대로 돌려준다 (ADR-025).
//
// ★ 폭이 1 을 넘는 순간 필요해졌다 ★ — 순차면 "지금 ④ 단계" 한 줄로 족했지만
// 여러 가지가 동시에 살면 각각이 다른 상태에 있고, GET record 는 종료 전이면
// 409 다(I4). ⇒ 진행을 읽을 경로가 ★ 선택이 아니라 필수 ★ 가 된다.
func (s *Store) Steps(ctx context.Context, runID string) ([]StepView, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT seq, name, state, uses, coalesce(node_id,''), needs, attempt,
		       started_at, ended_at
		  FROM steps WHERE run_id = $1 ORDER BY seq`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StepView{}
	for rows.Next() {
		var v StepView
		if err := rows.Scan(&v.Seq, &v.ID, &v.State, &v.Uses, &v.Node, &v.Needs,
			&v.Attempt, &v.StartedAt, &v.EndedAt); err != nil {
			return nil, err
		}
		if v.Needs == nil {
			v.Needs = []string{}
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
