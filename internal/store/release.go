package store

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"github.com/taeels/enode/internal/contract"
)

// applyRelease 는 그 단계가 놓기로 한 역할들의 점유를 푼다 (ADR-022 §7.4).
//
// 왜 필요한가 — 분기가 생기면 어느 경로로 갈지 모르므로 모든 경로의
// 자원을 잡아야 한다 (I5). 문제는 안 쓰는 노드를 Run 내내 묶는 것이고,
// 경로가 확정된 뒤 놓으면 풀린다. 계획 위임이 이 조건을 더 강하게 만든다 —
// 계획을 기계가 지으면 사람은 무엇이 쓰일지 모른 채 requires 를 선언한다.
//
// 전달 경로가 이미 있다 — ADR-016 의 「임대 목록에서 빠진다」가 그것이다.
// 노드는 다음 하트비트에서 자기 임대가 없어진 것을 보고 다음 단계를 시작하지 않는다.
// ⇒ 새 통보 채널이 안 생긴다.
//
// 한 노드가 두 역할을 맡았으면 안 놓는다 — 매처가 같은 노드를 두 역할에
// 짝지을 수 있고(I1 은 Run 사이의 배타이지 Run 안의 중복이 아니다), 그때
// 한쪽을 놓는다고 임대를 지우면 남은 역할이 자기 노드를 잃는다.
//
// 되돌릴 수 없다 — 놓은 것은 남이 채간다. 그래서 계약 검증이 미리
// "그 역할을 쓰는 모든 단계가 놓는 단계의 조상 일 것" 을 요구한다.
// 폭이 열리면서 그 조건이 강해졌다 — 순차였다면 "뒤에서 안 쓰면 된다" 로
// 족했지만, 병렬에서는 순서가 안 정해진 단계가 동시에 돌 수 있다.
// 돌려주는 수는 지운 임대의 수다 — 0 보다 크면 부르는 쪽이 큐를 깨운다 (ADR-064).
func (s *Store) applyRelease(ctx context.Context, tx pgx.Tx, runID string, seq int) (int, error) {
	var raw, assignedJSON []byte
	if err := tx.QueryRow(ctx,
		`SELECT `+liveContract+`, assigned FROM runs WHERE run_id=$1`, runID).
		Scan(&raw, &assignedJSON); err != nil {
		return 0, err
	}
	var c contract.Contract
	if err := json.Unmarshal(raw, &c); err != nil {
		return 0, err
	}
	if seq < 1 || seq > len(c.Steps) || len(c.Steps[seq-1].Release) == 0 {
		return 0, nil
	}
	freed := map[string]bool{}
	for _, r := range c.Steps[seq-1].Release {
		freed[r] = true
	}
	var assigned []Assigned
	if len(assignedJSON) > 0 {
		if err := json.Unmarshal(assignedJSON, &assigned); err != nil {
			return 0, err
		}
	}
	keep, drop := map[string]bool{}, map[string]bool{}
	for _, a := range assigned {
		for _, n := range a.Nodes {
			if freed[a.As] {
				drop[n.Node] = true
			} else {
				keep[n.Node] = true
			}
		}
	}
	nodes := []string{}
	for n := range drop {
		if !keep[n] {
			nodes = append(nodes, n)
		}
	}
	if len(nodes) == 0 {
		return 0, nil
	}
	tag, err := tx.Exec(ctx, `DELETE FROM leases WHERE run_id=$1 AND node_id = ANY($2)`,
		runID, nodes)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}
