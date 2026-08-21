package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/record"
)

// Manifest 는 Record 의 manifest.json 이다 (ADR-005).
//
// ★ 계약 전문이 여기 들어간다 ★ — 성질 4(자기충족)의 핵심이다.
// ADR-020 이 스키마를 인라인으로 둔 덕에 ★ 무엇으로 검증했는지 ★ 까지 함께 남는다.
type Manifest struct {
	RunID     string        `json:"run_id"`
	Work      contract.Work `json:"work"`
	Principal string        `json:"principal"` // 요청자. ★ 식별이지 인증이 아니다 ★
	State     string        `json:"state"`
	CreatedAt time.Time     `json:"created_at"`
	EndedAt   *time.Time    `json:"ended_at,omitempty"`
	Assigned  []Assigned    `json:"assigned,omitempty"`
	// Contract 는 ★ 계약 하나가 아니라 계약의 열이다 ★ (ADR-022 §7.6 · B1).
	Contract []ContractVersion `json:"contract"`
}

// ContractVersion 은 계약의 한 판이다 (ADR-022 §7.6).
//
// ★ 왜 객체가 아니라 열인가 ★
//
// 계약은 실행 중에 바뀔 수 있다 — 계획이 펼쳐지고(expands), 결과를 보고 다시
// 지어진다(재계획). 그때 "무엇을 실행했나" 에 답하려면 ★ 바뀌어 온 과정 ★ 이
// 남아야 한다. 봉인은 append-only 이므로(ADR-005 성질 1) 앞 판을 고치지 않고
// 새 판을 붙인다.
//
// ★ 길이 제약을 코드에 박지 않는다 ★
//
//	불변식은 ★ len >= 1 ★ 뿐이다. 상한은 없다.
//	"오늘은 원소가 하나다" 는 ★ 관찰이지 제약이 아니다 ★ — 아직 append 하는
//	주체가 없을 뿐이다. 여기에 len == 1 을 박으면 ADR-021 이 기록한 실수와
//	같은 모양이 된다: "ADR-017 결정 6 을 구현이 그대로 조건문으로 옮겨"
//	명령 단계 산출물이 안 걷힌 그 사고. 문서의 ★ 서술 ★ 을 코드의 ★ 제약 ★ 으로
//	굳히면 ★ 자리를 남긴 게 아니라 자리를 막은 것 ★ 이 된다.
//
//	★ 하한 1 은 진짜 제약이다 ★ — Run 이 있다는 것은 누군가 무언가를 제출했다는
//	뜻이고, v1 은 ★ 제출 전문 그대로 ★ 여야 한다 (성질 4).
//
// ★ 왜 차분이 아니라 전문인가 ★
//
// 차분이면 읽는 사람이 재구성해야 한다. 성질 4 는 "봉인된 묶음만 보고 알 수
// 있어야 한다" 이므로 ★ 각 판이 그 자체로 완결 ★ 이어야 한다. 계약은 KB 단위라
// 중복 비용이 문제되지 않는다.
type ContractVersion struct {
	V  int       `json:"v"`  // 1 부터
	At time.Time `json:"at"` // 이 판이 생긴 시각
	// By 는 ★ 누가 지었나 ★ 다. 아래 By* 상수를 쓴다.
	By string `json:"by"`
	// Evidence 는 이 판의 근거가 된 산출물이다 — Record 안의 경로.
	// 계획이 펼쳐진 판이면 그 계획 blob 을 가리킨다 (예: "blobs/plan-plan").
	// ★ 스키마 검증을 통과한 것만 여기 온다 ★ (ADR-020).
	Evidence string `json:"evidence,omitempty"`
	// Cause 는 ★ 무엇을 보고 바꿨나 ★ 다 — 앞 단계의 결과 파일
	// (예: "steps/02-baseline_build.json"). 재계획에서만 채워진다.
	Cause string `json:"cause,omitempty"`
	// ★ 임베드다 ★ — JSON 이 평평해져서 한 판이 그대로 계약으로 읽힌다.
	contract.Contract
}

// 계약 한 판을 ★ 누가 지었나 ★ 의 어휘. 열린 어휘가 아니라 이 셋과
// "step:<id>" 형태뿐이다 — Record 를 읽는 쪽이 문자열을 추측하면 안 된다.
const (
	// ByRequester 는 ★ 제출본 ★ 이다. v1 은 언제나 이것이다.
	ByRequester = "requester"
	// ByMediator 는 Mediator 가 만든 판이다.
	// ★ 아직 안 쓴다 ★ — run-contract §5 의 "@work.parent_rev 참조 해석" 이
	// 여기 올 자리다. 해석본만 남기면 성질 4 위반이고 제출본만 남기면
	// 무엇이 돌았는지 모르므로, 닫을 때 ★ 둘 다 ★ 남기는 형태가 된다.
	ByMediator = "mediator"
)

// ByStep 은 어느 단계가 지었는지를 가리킨다 (ADR-022 P4·P6).
// ★ 아직 부르는 곳이 없다 ★ — expands 와 재계획이 생기면 그때 쓴다.
func ByStep(stepID string) string { return "step:" + stepID }

// StepFiles 는 봉인에 쓸 단계 기록을 모은다.
// ★ node id 와 label 을 함께 남긴다 ★ (성질 3) — Case D 의
// "서로 다른 기계였다" 를 사람이 읽을 수 있어야 한다.
func (s *Store) StepFiles(ctx context.Context, runID string) ([]record.StepFile, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT st.seq, st.name, st.uses, st.kind, coalesce(st.node_id,''),
		       coalesce(n.label,''), st.state, st.started_at, st.ended_at, st.result
		  FROM steps st
		  LEFT JOIN nodes n ON n.node_id = st.node_id
		 WHERE st.run_id = $1 ORDER BY st.seq`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []record.StepFile
	for rows.Next() {
		var f record.StepFile
		var started, ended *time.Time
		var raw []byte
		if err := rows.Scan(&f.Seq, &f.Name, &f.Uses, &f.Kind, &f.Node, &f.NodeLabel,
			&f.State, &started, &ended, &raw); err != nil {
			return nil, err
		}
		f.StepID = stepID(runID, f.Seq)
		if started != nil {
			f.StartedAt = started.UTC().Format(time.RFC3339Nano)
		}
		if ended != nil {
			f.EndedAt = ended.UTC().Format(time.RFC3339Nano)
		}
		if len(raw) > 0 {
			var r StepResult
			if err := json.Unmarshal(raw, &r); err == nil {
				f.Result = r
			}
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func stepID(runID string, seq int) string {
	return runID + "#" + pad2(seq)
}

func pad2(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// sealRecord 는 종료 상태에 이른 Run 의 기록을 완성하고 불변으로 만든다.
// 봉인은 멱등이다 — 이미 봉인됐으면 아무것도 하지 않는다 (I4).
func (s *Store) sealRecord(ctx context.Context, runID string, v Verdict) error {
	if s.Records == nil {
		return nil
	}
	run, err := s.GetRun(ctx, runID)
	if err != nil {
		return err
	}
	steps, err := s.StepFiles(ctx, runID)
	if err != nil {
		return err
	}
	var ended *time.Time
	if t, err := s.endedAt(ctx, runID); err == nil {
		ended = t
	}
	m := Manifest{
		RunID: run.RunID, Work: run.Contract.Work, Principal: run.Principal,
		State: v.State, CreatedAt: run.CreatedAt, EndedAt: ended,
		Assigned: run.Assigned,
		// ★ v1 은 언제나 제출 전문이다 ★ (성질 4 — 봉인된 묶음만 보고
		// "무엇을 요청했나" 를 알 수 있어야 한다). 실행 중에 붙은 판은
		// runs.contract_versions 에 쌓여 있고 여기서 그 뒤에 이어 붙는다.
		// ★ 아무도 안 붙였으면 판이 하나다 ★ = 오늘 그대로.
		Contract: append([]ContractVersion{{
			V: 1, At: run.CreatedAt, By: ByRequester, Contract: run.Contract,
		}}, s.contractVersions(ctx, runID)...),
	}
	return s.Records.Seal(runID, m, v, steps)
}

// contractVersions 는 실행 중에 붙은 판들이다 (P4 expands · P6 재계획).
// ★ 못 읽으면 빈 열이다 ★ — 봉인은 Run 당 한 번뿐이고 여기서 실패해 봉인을
// 통째로 막는 것보다, 제출본만이라도 남기는 편이 낫다.
func (s *Store) contractVersions(ctx context.Context, runID string) []ContractVersion {
	var raw []byte
	if err := s.pool.QueryRow(ctx,
		`SELECT contract_versions FROM runs WHERE run_id=$1`, runID).Scan(&raw); err != nil {
		return nil
	}
	var out []ContractVersion
	if json.Unmarshal(raw, &out) != nil {
		return nil
	}
	return out
}

func (s *Store) endedAt(ctx context.Context, runID string) (*time.Time, error) {
	var t *time.Time
	err := s.pool.QueryRow(ctx, `SELECT ended_at FROM runs WHERE run_id=$1`, runID).Scan(&t)
	return t, err
}
