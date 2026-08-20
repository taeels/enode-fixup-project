package store

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/taeels/enode/internal/contract"
)

// applyDispatch 는 보고를 마친 단계의 분기를 적용한다 (ADR-022 §7.2).
//
// ★ Mediator 는 판정하지 않는다 ★ — 에이전트가 낸 ★ 이름 ★ 을 읽어 그 단계를
// 남기고 나머지 갈림길을 SKIPPED 로 만들 뿐이다. 표현식을 평가하지 않으므로
// ADR-004(기계적 판정만)와 ADR-011(라벨 매칭 계열)이 그대로 산다.
//
// ★ 이름을 못 고르면 그 단계가 FAILED 다 ★
//
//	값이 to 에 없으면 라우터가 라우팅을 못 한다. 그건 ★ 결과가 나쁜 것 ★ 이 아니라
//	★ 단계가 계약이 요구한 것을 못 낸 것 ★ 이므로 "완주하지 못함" 과 같은 자리다
//	(ADR-004 를 안 건드린다 — 내용의 좋고 나쁨을 안 본다).
//	보통은 여기 오기 전에 막힌다 — enum 을 벗어난 값은 PUT blob 이 422 로 거절해
//	저장되지 않는다 (ADR-020). ★ 검사가 두 겹인 것이 의도다 ★.
func (s *Store) applyDispatch(ctx context.Context, tx pgx.Tx, runID string, seq int) error {
	var raw []byte
	if err := tx.QueryRow(ctx, `SELECT contract FROM runs WHERE run_id=$1`, runID).
		Scan(&raw); err != nil {
		return err
	}
	var c contract.Contract
	if err := json.Unmarshal(raw, &c); err != nil {
		return err
	}
	if seq < 1 || seq > len(c.Steps) {
		return nil
	}
	st := c.Steps[seq-1]
	if st.Dispatch == nil {
		return nil
	}

	blob, path, _ := strings.Cut(st.Dispatch.From, ".")
	chosen, err := s.readDispatchValue(runID, blob, path)
	if err != nil {
		return fmt.Errorf("step %q: %w", st.ID, err)
	}
	var keep bool
	for _, t := range st.Dispatch.To {
		if t == chosen {
			keep = true
			break
		}
	}
	if !keep {
		return fmt.Errorf("step %q: 고른 이름 %q 가 dispatch.to 에 없다", st.ID, chosen)
	}

	// ★ 안 간 쪽만 SKIPPED 로 ★ — 갈림길 밖의 단계는 건드리지 않는다.
	// PENDING 인 것만 바꾼다: 이미 돈 것을 되돌리지 않는다.
	others := make([]string, 0, len(st.Dispatch.To))
	for _, t := range st.Dispatch.To {
		if t != chosen {
			others = append(others, t)
		}
	}
	_, err = tx.Exec(ctx, `
		UPDATE steps SET state=$3, ended_at=now()
		 WHERE run_id=$1 AND name = ANY($2) AND state='PENDING'`,
		runID, others, StepSkipped)
	return err
}

// readDispatchValue 는 산출물에서 이름 하나를 꺼낸다.
//
// ★ 경로는 점으로 끊는다 ★ — JSON 포인터를 흉내 내지 않는다. 배열 색인도 없다.
// 흉내 내기 시작하면 그게 곧 식 언어의 첫 조각이 된다 (argv 에서 셸을 안 흉내 낸
// 것과 같은 이유). ★ 필요한 것은 값 하나를 가리키는 것뿐이다 ★.
func (s *Store) readDispatchValue(runID, blob, path string) (string, error) {
	if s.Records == nil {
		return "", fmt.Errorf("기록 저장소가 없다")
	}
	rc, _, err := s.Records.OpenBlob(runID, blob)
	if err != nil {
		return "", fmt.Errorf("산출물 %q 를 못 읽었다: %w", blob, err)
	}
	defer rc.Close() //nolint:errcheck
	b, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return "", fmt.Errorf("산출물 %q 가 JSON 이 아니다: %w", blob, err)
	}
	if path != "" {
		for _, key := range strings.Split(path, ".") {
			m, ok := v.(map[string]any)
			if !ok {
				return "", fmt.Errorf("%q 를 따라가다 객체가 아닌 것을 만났다", path)
			}
			v, ok = m[key]
			if !ok {
				return "", fmt.Errorf("산출물에 %q 가 없다", path)
			}
		}
	}
	sv, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("%q 가 문자열이 아니다 — 이름이어야 한다", path)
	}
	return sv, nil
}
