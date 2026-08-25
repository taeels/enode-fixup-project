package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// BuildContract 는 고정 템플릿이다 — 추론이 0 개다.
//
// integration §5 가 지적한 구멍이 여기다: claim 이 주는 일곱 필드는 전부
// 자연어라서 계약을 유도할 수 없다. 그래서 유도하지 않는다 — 목표 위임
// (ADR-033)이 정확히 이 자리이고, 계약은 목표를 담는 그릇 일 뿐이다.
// 무엇을 할지는 오케스트레이터가 계획으로 짓는다.
//
// 계약이 목적지를 안 적는다 (ADR-062) — 승인이면 계획대로 이어 돌고,
// 거절이면 계획을 지은 단계로 되돌아가 다시 짓는다. dispatch 를 안 쓴다.
func BuildContract(cfg *Config, runID, issueKey string, issue *Issue, rr *RunnerRun) ([]byte, error) {
	planPrompt := planPrompt(cfg, issueKey, issue, rr)

	planner := map[string]any{
		"as":         "planner",
		"capability": "orchestration",
		// 자기 것만 잡는다 — 이 이름표가 없으면 남의 이슈를 위해 뜬
		// 오케스트레이터가 걸린다 (adapter-example §1.2 ⑥⑦).
		"issue": issueKey,
	}
	worker := map[string]any{
		"as":         cfg.Executor.As,
		"capability": "agent.reason",
	}
	for k, v := range cfg.Executor.Attrs {
		worker[k] = v
	}

	steps := []any{
		map[string]any{
			"id":   "plan",
			"uses": "planner",
			"agent": map[string]any{
				"ask":        "never",
				"max_tokens": 300000,
				"max_turns":  60,
			},
			"in":     map[string]any{"prompt": planPrompt},
			"out":    []string{"plan"},
			"schema": map[string]any{"plan": planSchema()},
			// 계획이 단계로 펼쳐지는 자리 (ADR-022 §6).
			"expands": true,
			// 이름을 약속한다 (ADR-049) — 계획이 어떻게 짓든 "report" 라는
			// 단계가 있어야 하고, success_when 이 그 이름을 가리킨다.
			// 그래프의 자리가 아니라 끝난 뒤의 사실을 묻는다 (ADR-062 §1.2).
			"produces": []string{"report"},
		},
		map[string]any{
			"id":    "gate",
			"uses":  "",
			"needs": []string{"plan"},
			"out":   []string{"approval"},
			"schema": map[string]any{
				"approval": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"verdict": map[string]any{
							"title": "승인 여부",
							"enum":  []string{"ok", "again"},
						},
						"note": map[string]any{
							"title": "이유",
							"type":  "string",
						},
					},
					"required": []string{"verdict"},
				},
			},
			// 승인의 자리를 계약이 정한다 (ADR-061).
			// adopts 는 어느 단계의 제안을 채택하는지, adopt_when 은 어느 답이
			// 채택인지를 적는다. 거절이면 계획을 지은 단계로 되돌아간다.
			"ask": map[string]any{
				"prompt": "이 계획으로 진행할까요. 좋으면 ok 로, 다시 지어야 하면 " +
					"again 으로 답하고 note 에 무엇이 문제인지 구체적으로 적어 주세요 — " +
					"그 글이 다시 지을 때 계획 단계로 전달됩니다.",
				"show":       []string{"plan"},
				"adopts":     "plan",
				"adopt_when": "ok",
			},
		},
	}

	c := map[string]any{
		"run_id": runID,
		"work": map[string]any{
			"system":    "itsaplan",
			"change_id": issueKey,
		},
		"requires": []any{planner, worker},
		"steps":    steps,
		"success_when": []any{
			map[string]any{"step": "plan", "produced": []string{"plan"}},
			map[string]any{"step": "report", "produced": []string{"summary"}},
		},
		// 원장을 Work 범위로 연다 (ADR-023 §6 · ADR-040 §3.4).
		// 되묻기는 It's a Plan 쪽 run 을 닫으므로 답은 항상 다른 agent run
		// 으로 온다. 같은 Work 의 이전 Run 이 낸 _outbound 가 보여야 어느
		// 질문의 답인지 맞출 수 있다.
		"ledger": map[string]any{"scope": "work"},
	}
	return json.MarshalIndent(c, "", "  ")
}

// planSchema 는 계획의 모양이다.
//
// 규칙과 함께 모양을 준다 (ADR-057) — 이름에서 유추하게 두면 에이전트가
// "agent 단계니까 agent 에 할 일을 적는다" 같은 오해를 한다.
func planSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"steps", "success_when"},
		"properties": map[string]any{
			"steps": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":     "object",
					"required": []string{"id"},
					"properties": map[string]any{
						"id":       map[string]any{"type": "string"},
						"uses":     map[string]any{"type": "string"},
						"needs":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
						"run":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
						"env":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
						"out":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
						"feedback": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
						"agent":    map[string]any{"type": "object"},
						"ask":      map[string]any{"type": "object"},
						"in":       map[string]any{"type": "object"},
						"schema":   map[string]any{"type": "object"},
						"expands":  map[string]any{"type": "boolean"},
					},
				},
			},
			"success_when": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":     "object",
					"required": []string{"step"},
					"properties": map[string]any{
						"step":      map[string]any{"type": "string"},
						"produced":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
						"exit_code": map[string]any{"type": "integer"},
					},
				},
			},
		},
	}
}

// planPrompt 는 이슈를 목표로 바꾼다.
//
// 여기가 유일하게 It's a Plan 의 말이 우리 쪽으로 들어오는 통로다 —
// 그리고 그것은 목표 문자열 하나다. 구조는 전부 템플릿이 준다.
func planPrompt(cfg *Config, issueKey string, issue *Issue, rr *RunnerRun) string {
	var b strings.Builder
	b.WriteString("You are the orchestration node. You do not execute; you build the remaining steps.\n\n")

	b.WriteString("# Goal\n\n")
	if issue != nil {
		fmt.Fprintf(&b, "issue %s — %s\n\n", issueKey, issue.Title)
		if strings.TrimSpace(issue.Description) != "" {
			b.WriteString(strings.TrimSpace(issue.Description))
			b.WriteString("\n\n")
		}
	}
	if strings.TrimSpace(rr.Prompt) != "" {
		b.WriteString("What the issue tracker passed along:\n\n")
		b.WriteString(strings.TrimSpace(rr.Prompt))
		b.WriteString("\n\n")
	}

	b.WriteString("# Roles\n\n")
	fmt.Fprintf(&b, "%s — an execution node. It can run a shell in a run step and reason in an agent step.\n",
		cfg.Executor.As)
	for k, v := range cfg.Executor.Attrs {
		fmt.Fprintf(&b, "  · %s = %s\n", k, v)
	}
	b.WriteString("\n")

	b.WriteString(`# What the contract has already fixed

- The last step must be named exactly "report", and it must produce $OUT/summary.
  The verdict looks only at that fact — how many steps come before report does not matter.
- After you submit the plan a person answers whether they approve it. On a rejection
  you come back to this step and build again, and the reason is passed to you.

# Rules for building the plan

- Keep the steps to a minimum. Two or three are enough.
- Each run step must put its result under $OUT/ to be harvested.
- Every step gets a fresh shell — write the environment variables again each time.
- Never touch anything outside the workspace. Reading is free; writing goes inside $OUT.
- Do not use destructive commands (rm -rf, overwriting config files, killing processes).
`)
	return b.String()
}
