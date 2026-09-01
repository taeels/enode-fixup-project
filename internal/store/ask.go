package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/schema"
)

// raiseAsks 는 needs 가 찬 되묻기 단계를 ASKED 로 올린다 (ADR-032).
//
// 대기는 일급 상태다 — 조사한 15개 시스템 전부의 공통점이고, 자원을 쥔 채
// 기다리는 것이 안티패턴의 교과서였다. ASKED 단계는 노드에 안 가고(claim 이
// node_id 로 거른다) 자원을 안 잡는다. 발견은 인박스(PendingAsks)가 한다.
func (s *Store) raiseAsks(ctx context.Context, tx pgx.Tx, runID string) ([]AskEvent, error) {
	rows, err := tx.Query(ctx, `
		SELECT s.seq FROM steps s
		 WHERE s.run_id = $1 AND s.state = 'PENDING' AND s.kind = 'ask'
		   AND NOT EXISTS (
		       SELECT 1 FROM steps p
		        WHERE p.run_id = s.run_id AND p.name = ANY(s.needs)
		          AND p.state NOT IN ('DONE','SKIPPED'))`, runID)
	if err != nil {
		return nil, err
	}
	var seqs []int
	var raised []AskEvent
	for rows.Next() {
		var n int
		if err := rows.Scan(&n); err != nil {
			rows.Close()
			return nil, err
		}
		seqs = append(seqs, n)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(seqs) == 0 {
		return nil, nil
	}
	c, err := s.liveContractIn(ctx, tx, runID)
	if err != nil {
		return nil, err
	}
	for _, seq := range seqs {
		if seq < 1 || seq > len(c.Steps) || c.Steps[seq-1].Ask == nil {
			continue
		}
		var deadline *time.Time
		if t := c.Steps[seq-1].Ask.Timeout; t != nil {
			if d, err := time.ParseDuration(t.After); err == nil {
				at := time.Now().UTC().Add(d)
				deadline = &at
			}
		}
		if _, err := tx.Exec(ctx, `
			UPDATE steps SET state='ASKED', started_at=now(), ask_deadline=$3
			 WHERE run_id=$1 AND seq=$2 AND state='PENDING'`,
			runID, seq, deadline); err != nil {
			return nil, err
		}
		st := c.Steps[seq-1]
		s.log().Info("ask: awaiting an answer", "run", runID, "seq", seq, "step", st.ID)
		raised = append(raised, AskEvent{
			Event: "ask", RunID: runID, Seq: seq, Step: st.ID,
			Prompt: st.Ask.Prompt, Answerers: st.Ask.Answerers, Deadline: deadline,
			AnswerPath: s.answerPath(runID, seq),
		})
	}
	return raised, nil
}

// AskEvent 는 푸시 웹훅에 실리는 것이다 (ADR-032 §4 — 푸시는 보조).
//
// 인박스가 정본이고 푸시는 알림이다 — 유실돼도 인박스에 남는다.
// 그래서 재시도가 없다. answer_path 는 알림에서 바로 응답 지점으로 가는
// 직링크 재료다 (조사의 Airflow 모범 — 알림에 응답 페이지 직링크).
type AskEvent struct {
	Event      string     `json:"event"`
	RunID      string     `json:"run_id"`
	Seq        int        `json:"seq"`
	Step       string     `json:"step"`
	Prompt     string     `json:"prompt"`
	Answerers  []string   `json:"answerers,omitempty"`
	Deadline   *time.Time `json:"deadline,omitempty"`
	AnswerPath string     `json:"answer_path"`
}

// answerPath 는 주입된 경로 생성기를 쓴다. 없으면 빈 문자열이다 —
// 상태 층이 라우트를 지어내지 않는다 (FR3.2).
func (s *Store) answerPath(runID string, seq int) string {
	if s.AnswerPath == nil {
		return ""
	}
	return s.AnswerPath(runID, seq)
}

// PushAsks 는 올라온 질문을 웹훅으로 알린다. 커밋 뒤에만 부른다 —
// 트랜잭션 안에서 쏘면 롤백된 질문을 알리게 된다.
func (s *Store) PushAsks(events []AskEvent) {
	if s.NotifyURL == "" || len(events) == 0 {
		return
	}
	go func() {
		cl := &http.Client{Timeout: 5 * time.Second}
		for _, e := range events {
			body, err := json.Marshal(e)
			if err != nil {
				continue
			}
			resp, err := cl.Post(s.NotifyURL, "application/json", bytes.NewReader(body))
			if err != nil {
				s.log().Warn("ask notification failed; the inbox remains authoritative", "err", err)
				continue
			}
			resp.Body.Close()
		}
	}()
}

// AskView 는 인박스의 항목 하나다 (ADR-032 §4).
// 대기 중인 것만 든다 — 답한 것은 봉인에 있다.
type AskView struct {
	RunID     string          `json:"run_id"`
	Seq       int             `json:"seq"`
	Step      string          `json:"step"`
	Prompt    string          `json:"prompt"`
	Schema    json.RawMessage `json:"schema"`
	Answerers []string        `json:"answerers,omitempty"`
	AskedAt   *time.Time      `json:"asked_at,omitempty"`
	Deadline  *time.Time      `json:"deadline,omitempty"`
	// CanAnswer 는 관점 필드다 — 보는 사람 기준으로 서버가 채운다
	// (GitHub 의 current_user_can_approve 모범). 저장되는 값이 아니다.
	CanAnswer bool `json:"can_answer"`
	// Shown 은 질문과 함께 보여줄 산출물의 내용이다 (ask.show).
	// 보지 않고 답하게 만들면 안 된다 — prompt 는 계약 시점 문자열이라
	// 에이전트가 실행 중에 만든 질문 내용은 blob 에 있다. 그것을 여기 든다.
	Shown []ShownArtifact `json:"shown,omitempty"`
	// Proposes 는 이 ask 가 채택 여부를 묻는 제안된 판정 기준이다 (ADR-033).
	// 무엇을 승인하는지 보지 않고 승인하게 만들면 안 된다 — 그래서
	// 인박스가 제안을 함께 든다.
	Proposes []contract.Condition `json:"proposes,omitempty"`
}

// ShownArtifact 는 인박스에 함께 실리는 산출물 하나다.
type ShownArtifact struct {
	Name string `json:"name"`
	// Content 는 JSON 이면 그대로, 아니면 문자열로 실린다.
	Content json.RawMessage `json:"content"`
	// Truncated 는 잘렸다는 표시다 — 전문은 blob 표면으로 가져간다.
	Truncated bool `json:"truncated,omitempty"`
}

// askShowLimit 는 인박스에 인라인으로 싣는 산출물의 상한이다.
// 인박스는 목록이다 — 10MiB blob 을 통째로 실으면 원장이 목록인 이유
// (ADR-023 §6.3)와 같은 문제가 여기서 난다. 넘으면 자르고 표시한다.
const askShowLimit = 32 << 10

// shownArtifacts 는 ask.show 가 지목한 산출물들을 읽어 온다.
func (s *Store) shownArtifacts(runID string, names []string) []ShownArtifact {
	if s.Records == nil {
		return nil
	}
	out := make([]ShownArtifact, 0, len(names))
	for _, name := range names {
		rc, _, err := s.Records.OpenBlob(runID, name)
		if err != nil {
			continue // 아직 안 나온 산출물 — 없는 채로 보여준다
		}
		buf := make([]byte, askShowLimit+1)
		n, _ := io.ReadFull(rc, buf)
		rc.Close()
		a := ShownArtifact{Name: name}
		if n > askShowLimit {
			a.Truncated = true
			n = askShowLimit
		}
		body := buf[:n]
		if json.Valid(body) && !a.Truncated {
			a.Content = body
		} else {
			enc, _ := json.Marshal(string(body))
			a.Content = enc
		}
		out = append(out, a)
	}
	return out
}

// PendingAsks 는 답을 기다리는 되묻기 전부다 — 인박스의 정본 표면.
func (s *Store) PendingAsks(ctx context.Context) ([]AskView, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT s.run_id, s.seq, s.name, s.started_at, s.ask_deadline,
		       coalesce(r.contract_versions -> -1, r.contract), r.contract_versions
		  FROM steps s JOIN runs r ON r.run_id = s.run_id
		 WHERE s.state = 'ASKED' AND r.state = 'RUNNING'
		 ORDER BY s.started_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AskView{}
	for rows.Next() {
		var v AskView
		var raw, versRaw []byte
		if err := rows.Scan(&v.RunID, &v.Seq, &v.Step, &v.AskedAt, &v.Deadline,
			&raw, &versRaw); err != nil {
			return nil, err
		}
		var c contract.Contract
		if json.Unmarshal(raw, &c) != nil || v.Seq < 1 || v.Seq > len(c.Steps) {
			continue
		}
		st := c.Steps[v.Seq-1]
		if st.Ask == nil {
			continue
		}
		v.Prompt = st.Ask.Prompt
		v.Answerers = st.Ask.Answerers
		if len(st.Out) == 1 {
			if b, err := json.Marshal(st.Schema[st.Out[0]]); err == nil {
				v.Schema = b
			}
		}
		// 질문과 함께 볼 것을 든다 (ask.show).
		if len(st.Ask.Show) > 0 {
			v.Shown = s.shownArtifacts(v.RunID, st.Ask.Show)
		}
		// 무엇을 승인하는지 함께 든다 — adopts 대상의 마지막 제안.
		if st.Ask.Adopts != "" && len(versRaw) > 0 {
			var vers []ContractVersion
			if json.Unmarshal(versRaw, &vers) == nil {
				for i := len(vers) - 1; i >= 0; i-- {
					if vers[i].By == byStep(st.Ask.Adopts) && len(vers[i].Proposed) > 0 {
						v.Proposes = vers[i].Proposed
						break
					}
				}
			}
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

var (
	// ErrNoAsk 는 답을 기다리는 단계가 아니다 — 없거나, 이미 답했거나, ask 가 아니다.
	ErrNoAsk = errors.New("step is not awaiting an answer")
	// ErrNotAnswerer 는 answerers 에 없는 사람이다.
	ErrNotAnswerer = errors.New("principal is not an allowed answerer")
)

// SchemaViolation 은 답이 질문의 형태를 어겼다 — PUT blob 의 422 와 같은 자리다.
type SchemaViolation struct{ Details string }

func (e *SchemaViolation) Error() string { return "schema violation: " + e.Details }

// AnswerStep 은 답을 받는다 (ADR-032 §1②) — 주소 있는 단일 쓰기.
//
// 답은 산출물이다: 스키마 검증을 통과해야 저장되고(강제 지점 하나 — ADR-020),
// 단계가 DONE 이 되면서 answered_by · answered_at이 기록에 남고,
// dispatch · 전파 · 다음 되묻기까지 보고와 같은 트랜잭션 경로를 탄다.
func (s *Store) AnswerStep(ctx context.Context, runID string, seq int,
	principal string, body []byte, limit int64) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var state string
	var attempt int
	err = tx.QueryRow(ctx, `
		SELECT s.state, s.attempt FROM steps s JOIN runs r ON r.run_id = s.run_id
		 WHERE s.run_id=$1 AND s.seq=$2 AND s.kind='ask' AND r.state='RUNNING'
		 FOR UPDATE OF s`, runID, seq).Scan(&state, &attempt)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && state != "ASKED") {
		return false, ErrNoAsk
	}
	if err != nil {
		return false, err
	}
	c, err := s.liveContractIn(ctx, tx, runID)
	if err != nil {
		return false, err
	}
	if seq < 1 || seq > len(c.Steps) || c.Steps[seq-1].Ask == nil {
		return false, ErrNoAsk
	}
	st := c.Steps[seq-1]
	if as := st.Ask.Answerers; len(as) > 0 {
		ok := false
		for _, a := range as {
			if a == principal {
				ok = true
				break
			}
		}
		if !ok {
			return false, ErrNotAnswerer
		}
	}
	name := st.Out[0]
	if vs := schema.Validate(st.Schema[name], body); len(vs) > 0 {
		parts := make([]string, 0, len(vs))
		for _, v := range vs {
			parts = append(parts, v.String())
		}
		// 어긴 답은 저장하지 않는다 — 질문은 그대로 열려 있다. 다시 답하면 된다.
		return false, &SchemaViolation{Details: strings.Join(parts, " / ")}
	}
	if s.Records == nil {
		return false, fmt.Errorf("record store is not configured")
	}
	if _, err := s.Records.WriteBlob(runID, seq, attempt, name,
		bytes.NewReader(body), limit); err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE steps SET state='DONE', ended_at=now(), result=$3
		 WHERE run_id=$1 AND seq=$2`,
		runID, seq, mustJSON(StepResult{
			Produced: []string{name}, AnsweredBy: principal})); err != nil {
		return false, err
	}
	s.log().Info("ask: answer received", "run", runID, "step", st.ID, "by", principal)
	// 채택 — 목표 위임의 승인 지점 (ADR-033).
	//
	// 이 ask 가 expands 단계를 adopts 로 지목했고 답이 approve 면, 그 계획이
	// 제안했던 success_when 이 여기서야 효력을 얻는다 — 계약의 열에
	// by: "answer:<이 단계>" 판이 붙는다. 저자(기계)와 승인자(사람)가
	// 봉인에 각각 남는다. reject 면 아무것도 채택되지 않고, dispatch 가
	// 선언돼 있으면 그 답대로 갈린다 (재계획 경로로 보낼 수 있다).
	if st.Ask.Adopts != "" {
		var ans struct {
			Verdict string `json:"verdict"`
		}
		// 어느 답이 채택인가는 계약이 정한다 (ADR-061 §2.4) —
		// 안 적었으면 오늘 그대로 "approve" 다.
		adoptOn := st.Ask.AdoptWhen
		if adoptOn == "" {
			adoptOn = "approve"
		}
		if json.Unmarshal(body, &ans) == nil && ans.Verdict == adoptOn {
			if err := s.adoptProposal(ctx, tx, runID, seq, attempt, st, c); err != nil {
				return false, err
			}
		} else if st.Dispatch != nil || st.Ask.AdoptWhen != "" {
			// 갈 곳이 있을 때만 물린다 (ADR-061 §2.5) — 아무것도 안 적었으면
			// 거절은 채택을 안 하는 것 뿐이고 계획은 그대로 돈다(ADR-033).
			// 그 동작을 여기서 바꾸면 계약이 안 적은 결말을 Mediator 가 정한다.
			//
			// adopt_when 만 적어도 갈 곳이 있다 (ADR-062) — 그때 거절의 뜻은
			// "다시 지어라" 이고, 갈 곳은 계획을 지은 그 단계다. 계약이
			// 목적지를 지목하지 않으므로 앞단이 뒷단의 모양을 단정하지 않는다.
			if err := s.retirePlan(ctx, tx, runID, seq, attempt, st); err != nil {
				return false, err
			}
			// 거절은 분기가 아니라 되돌림이다 (ADR-062) — 계획을 지은
			// 단계로 돌아가 다시 짓는다. ADR-026 이 "반복은 뒤로 가는 간선" 으로
			// 푼 것과 같은 모양이고, 되돌리는 기계도 그것을 그대로 쓴다.
			if st.Dispatch == nil {
				back, err := s.rewindToPlanner(ctx, tx, runID, seq, st, c)
				if err != nil {
					return false, err
				}
				if back {
					// 되돌렸으면 효과를 적용하지 않는다 — 아래 rollBack ·
					// afterStep 이 도는 자리와 같은 규칙이다.
					return true, tx.Commit(ctx)
				}
			}
		}
	}
	// 보고와 같은 길이다 — 되돌림을 먼저 보고, 안 되돌리면 효과를 적용한다.
	rolled, err := s.rollBack(ctx, tx, runID, seq)
	if err != nil {
		return false, err
	}
	if rolled {
		return true, tx.Commit(ctx)
	}
	raisedAsks, err := s.afterStep(ctx, tx, runID, seq)
	if err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	// 알림은 커밋 뒤에만 — 답이 다음 질문을 올렸을 수 있다 (질문의 사슬).
	s.PushAsks(raisedAsks)
	return false, nil
}

// adoptProposal 은 계획이 제안한 success_when 을 사람의 답으로 채택한다 (ADR-033).
func (s *Store) adoptProposal(ctx context.Context, tx pgx.Tx, runID string,
	seq, attempt int, ask contract.Step, live contract.Contract) error {
	var raw []byte
	if err := tx.QueryRow(ctx,
		`SELECT contract_versions FROM runs WHERE run_id=$1`, runID).Scan(&raw); err != nil {
		return err
	}
	var vers []ContractVersion
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &vers); err != nil {
			return err
		}
	}
	// 제안은 그 단계가 지은 마지막 판에 있다.
	var proposal *ContractVersion
	for i := len(vers) - 1; i >= 0; i-- {
		if vers[i].By == byStep(ask.Ask.Adopts) && len(vers[i].Proposed) > 0 {
			proposal = &vers[i]
			break
		}
	}
	if proposal == nil {
		// 계획이 제안을 안 했으면 채택할 것이 없다 — 승인은 그냥 답이다.
		return nil
	}
	next := live
	// 이미 선 조건과 겹치는 제안은 안 더한다 — 계약 저자가 쓴 조건을
	// 계획이 다시 제안하는 것은 흔하고, 그때 그냥 붙이면 같은 조건이 두 번
	// 선다. 오늘은 사본이 같아 판정이 안 바뀌지만, Verify 가 대조된 조건의
	// 수를 세어 「전부 건너뛰었나」의 하한으로 쓰므로 그 수가 사본만큼 부풀면
	// 하한이 잘못된 근거로 판단하게 된다.
	next.SuccessWhen = append([]contract.Condition{}, live.SuccessWhen...)
	for _, c := range proposal.Proposed {
		if contract.HasCondition(next.SuccessWhen, c) {
			continue
		}
		next.SuccessWhen = append(next.SuccessWhen, c)
	}
	// 채택 시점에 전체를 다시 검증한다 — 제안이 지어진 단계를 가리켜도
	// 지금은 그 단계가 live 에 있으므로 통과한다. 그것이 P4 가 그어둔
	// 「지어진 단계의 성패는 판정에 안 들어간다」는 경계가 여기서 열리는 방식이다.
	if err := next.Validate(); err != nil {
		return fmt.Errorf("adopting the proposal would make the contract invalid: %w", err)
	}
	ver := ContractVersion{
		V:  len(vers) + 2, // v1 은 runs.contract
		At: time.Now().UTC(),
		By: byAnswer(ask.ID),
		// 무엇으로 채택했나 — 사람의 답 blob 이다.
		Evidence: fmt.Sprintf("blobs/%02d.%d-%s", seq, attempt, ask.Out[0]),
		// 무엇을 채택했나 — 그 제안이 실린 계획의 산출물이다.
		Cause:    []string{proposal.Evidence},
		Contract: next,
	}
	vers = append(vers, ver)
	nextJSON, err := json.Marshal(vers)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE runs SET contract_versions=$2 WHERE run_id=$1`, runID, nextJSON); err != nil {
		return err
	}
	s.log().Info("proposal adopted; success criteria are now in effect",
		"run", runID, "by", byAnswer(ask.ID), "conditions", len(proposal.Proposed))
	return nil
}

// liveContractIn 은 트랜잭션 안에서 지금 유효한 계약을 읽는다.
func (s *Store) liveContractIn(ctx context.Context, tx pgx.Tx, runID string) (contract.Contract, error) {
	var raw []byte
	var c contract.Contract
	if err := tx.QueryRow(ctx,
		`SELECT `+liveContract+` FROM runs WHERE run_id=$1`, runID).Scan(&raw); err != nil {
		return c, err
	}
	return c, json.Unmarshal(raw, &c)
}

// ExpireAsks 는 기한이 지난 되묻기를 실패시킨다 (ADR-032 §2 — then:"fail").
// 회수(reap) 주기에서 돈다 — 여기서도 시간이 감시자다.
func (s *Store) ExpireAsks(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		UPDATE steps s SET state='FAILED', ended_at=now(),
		       result = coalesce(result,'{}'::jsonb) || $1::jsonb
		  FROM runs r
		 WHERE r.run_id = s.run_id AND r.state = 'RUNNING'
		   AND s.state = 'ASKED' AND s.ask_deadline IS NOT NULL
		   AND s.ask_deadline <= now()
		 RETURNING s.run_id`,
		mustJSON(map[string]string{"error": "ask deadline passed"}))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	seen := map[string]bool{}
	var runs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if !seen[id] {
			seen[id] = true
			runs = append(runs, id)
		}
	}
	return runs, rows.Err()
}

// retirePlan 은 거절당한 계획을 물린다 (ADR-061 §2.5).
//
// 무엇이 문제였나 — 계획은 expands 가 끝나는 그 순간 계약에 붙는다.
// 거절은 채택만 안 할 뿐이므로 그 단계들이 유효 계약에 그대로 남고, 그래서
// 재계획이 같은 목표를 다시 지으려 하면 이름이 이미 점유돼 있다:
//
//	실측 (reject-3): replan 이 "duplicate steps[].id: make_note" 로 실패했다.
//	거절 이유는 제대로 전달돼 읽혔는데, 지을 자리가 없었다.
//
// 지우지 않는다. 더한다 (ADR-005) — 계획이 붙기 직전의 계약을
// 새 판으로 다시 적는다. 판의 목록은 여전히 앞으로만 자라고, 무엇이
// 제안됐고 무엇이 거절됐는지가 봉인에 다 남는다.
//
//	v2  by: step:plan        ← 계획이 붙었다 (제안 포함)
//	v3  by: answer:approve   ← 거절. 내용은 v1 과 같다
//	v4  by: step:replan      ← 재계획이 여기 붙는다. 이름이 비어 있다
//
// liveContract 규칙을 안 건드린다 — 여전히 "마지막 판" 이다.
func (s *Store) retirePlan(ctx context.Context, tx pgx.Tx, runID string,
	seq, attempt int, ask contract.Step) error {
	var raw []byte
	if err := tx.QueryRow(ctx,
		`SELECT contract_versions FROM runs WHERE run_id=$1`, runID).Scan(&raw); err != nil {
		return err
	}
	var vers []ContractVersion
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &vers); err != nil {
			return err
		}
	}
	// 물릴 대상은 그 단계가 붙인 마지막 판이다.
	at := -1
	for i := len(vers) - 1; i >= 0; i-- {
		if vers[i].By == byStep(ask.Ask.Adopts) {
			at = i
			break
		}
	}
	if at < 0 {
		return nil // 계획이 안 붙었으면 물릴 것이 없다
	}
	// 그 앞의 계약 — v1(runs.contract)이면 vers 에 없으므로 그때는
	// 지금 유효 계약에서 계획이 지은 단계를 걷어낸 것과 같다.
	var back contract.Contract
	if at == 0 {
		if err := tx.QueryRow(ctx,
			`SELECT contract FROM runs WHERE run_id=$1`, runID).Scan(&raw); err != nil {
			return err
		}
		if err := json.Unmarshal(raw, &back); err != nil {
			return err
		}
	} else {
		back = vers[at-1].Contract
	}
	// 그 계획이 지은 단계의 행도 함께 물린다 — 계약에서 뺐는데 행이 남으면
	// 재계획이 같은 seq 에 INSERT 하다 기본키에 걸린다 (실측에서 밟았다:
	// "current transaction is aborted" 로 다음 보고가 409 가 됐다).
	//
	// 행은 파생이다 — 계약에서 유도된 것이므로 계약이 물러나면 함께 물러난다.
	// 계약 판은 그대로 남으므로 무엇이 있었는지는 봉인이 안다.
	//
	// 안 돈 것만 지운다 — 이미 돌았으면 그 사실이 기록이고, 지우면
	// Run Record 가 "일어난 일" 을 잃는다. 그때는 남겨서 SKIPPED 로도 안 바꾼다.
	kept := map[string]bool{}
	for _, st := range back.Steps {
		kept[st.ID] = true
	}
	var retired []string
	for _, st := range vers[at].Steps {
		if !kept[st.ID] {
			retired = append(retired, st.ID)
		}
	}
	if len(retired) > 0 {
		if _, err := tx.Exec(ctx, `
			DELETE FROM steps
			 WHERE run_id=$1 AND name = ANY($2) AND state='PENDING'`,
			runID, retired); err != nil {
			return err
		}
	}
	vers = append(vers, ContractVersion{
		V:  len(vers) + 2,
		At: time.Now().UTC(),
		By: byAnswer(ask.ID),
		// 무엇으로 물렸나 — 사람의 답 blob 이다. 채택과 같은 자리다.
		Evidence: fmt.Sprintf("blobs/%02d.%d-%s", seq, attempt, ask.Out[0]),
		// 무엇을 물렸나 — 거절당한 계획이 실린 판의 근거다.
		Cause:    []string{vers[at].Evidence},
		Contract: back,
	})
	nextJSON, err := json.Marshal(vers)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE runs SET contract_versions=$2 WHERE run_id=$1`, runID, nextJSON); err != nil {
		return err
	}
	s.log().Info("plan rejected; the contract went back to what it was",
		"run", runID, "by", byAnswer(ask.ID), "retired", len(retired))
	return nil
}

// rewindToPlanner 는 거절을 되돌림으로 만든다 (ADR-062).
//
// 왜 분기가 아닌가 — 계약이 "승인하면 여기로 간다" 를 적으려면 그 자리가
// 계획 안 어디인지를 알아야 한다. 그런데 그것은 계획을 짓는 쪽만 안다:
//
//	실측 (promised-1): 계약이 dispatch.to 에 목표 단계 report 를 적었는데,
//	계획이 그 앞에 준비 단계를 두자 승인해도 못 닿는 그래프가 됐다.
//	앞단이 뒷단의 모양을 단정한 것이 모순의 원천이다.
//
// ⇒ 계약은 목적지를 안 적는다. 승인이면 계획대로 이어서 돌고, 거절이면
//
//	계획을 지은 단계로 돌아가 다시 짓는다. ADR-026 이 "반복은 뒤로 가는
//	간선" 으로 푼 것과 같은 모양이다.
//
// 회차가 상한을 준다 — 되돌릴 때마다 attempt 가 오르고, max_versions 가
// 계약의 열에 상한을 준다(ADR-031). 무한히 되돌지 않는다.
func (s *Store) rewindToPlanner(ctx context.Context, tx pgx.Tx, runID string,
	seq int, ask contract.Step, c contract.Contract) (bool, error) {
	from := 0
	for i, st := range c.Steps {
		if st.ID == ask.Ask.Adopts {
			from = i + 1
			break
		}
	}
	if from == 0 {
		return false, nil // 검증이 막았어야 한다
	}
	// 구간을 통째로 되돌린다 — 계획 단계부터 이 ask 까지. loopBack 과 같은
	// 규칙이고, 획득 단계는 안 되돌린다 (임대가 살아 있다).
	var attempt int
	if err := tx.QueryRow(ctx,
		`SELECT attempt FROM steps WHERE run_id=$1 AND seq=$2`, runID, from).
		Scan(&attempt); err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE steps SET state='PENDING', attempt=$4, result=NULL,
		       started_at=NULL, ended_at=NULL, ledger_at=NULL
		 WHERE run_id=$1 AND seq BETWEEN $2 AND $3 AND kind <> 'acquire'`,
		runID, from, seq, attempt+1); err != nil {
		return false, err
	}
	s.log().Info("plan rejected; going back to the step that builds it",
		"run", runID, "back_to", ask.Ask.Adopts, "attempt", attempt+2)
	return true, nil
}
