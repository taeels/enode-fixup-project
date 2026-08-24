package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// ★ 코멘트는 어댑터가 만든다 ★ (ADR-040 §2.2).
//
// 계약이 문구를 적으면 계약이 이슈 트래커를 알게 된다. 그래서 계약은 아무것도
// 안 적고, 어댑터가 Run · verdict · 원장을 읽어 스스로 조립한다.
//
// ★ 그리고 이 자리가 함대 병렬을 표현할 유일한 자리다 ★ — 러너의 존재감은
// last_seen_at 한 칸이라 UI 로는 초록불 하나이고, 어느 단계가 어느 노드에서
// 돌았는지는 여기 표로만 보인다.

// RenderQuestion 은 되묻기를 사람이 읽는 글로 만든다.
func RenderQuestion(ask *AskView, runID string) string {
	var b strings.Builder
	b.WriteString("### 확인이 필요합니다\n\n")
	b.WriteString(strings.TrimSpace(ask.Prompt))
	b.WriteString("\n\n")

	for _, s := range ask.Shown {
		b.WriteString(fmt.Sprintf("<details>\n<summary><b>%s</b> — 무엇을 승인하는지 여기서 보세요</summary>\n\n",
			s.Name))
		b.WriteString("```json\n")
		b.WriteString(prettyJSON(s.Content))
		b.WriteString("\n```\n\n</details>\n\n")
	}

	if len(ask.Proposes) > 0 && string(ask.Proposes) != "null" {
		b.WriteString("**이 답이 채택하면 판정 기준이 되는 것**\n\n```json\n")
		b.WriteString(prettyJSON(ask.Proposes))
		b.WriteString("\n```\n\n")
	}

	// ★ It's a Plan 에 폼이 없다 ★ (ADR-040 §4) — 스키마 강제는 우리 쪽에
	// 남으므로, 답의 형태를 ★ 말로 ★ 적는다.
	b.WriteString("**답하는 방법** — 이 코멘트에 답글을 달아 주세요.\n\n")
	b.WriteString("- 승인이면 `ok` 로 시작하는 답글\n")
	b.WriteString("- 다시 지어야 하면 `again` 으로 시작하고, 그 뒤에 이유를 적어 주세요.\n")
	b.WriteString("  그 이유는 계획을 다시 짓는 단계로 그대로 전달됩니다.\n\n")
	b.WriteString(fmt.Sprintf("<sub>run `%s` · step `%s` (seq %d)</sub>", runID, ask.Step, ask.Seq))
	return b.String()
}

// RenderResult 는 끝난 Run 을 사람이 읽는 글로 만든다.
func RenderResult(ctx context.Context, m *Mediator, run *RunView, summary string) string {
	var b strings.Builder

	head := "실패했습니다"
	switch run.State {
	case "SUCCEEDED":
		head = "끝났습니다"
	case "CANCELLED":
		head = "취소됐습니다"
	case "EXPIRED":
		head = "시간이 다 됐습니다"
	}
	fmt.Fprintf(&b, "### %s — `%s`\n\n", head, run.State)

	if strings.TrimSpace(summary) != "" {
		b.WriteString(strings.TrimSpace(summary))
		b.WriteString("\n\n")
	}

	// ★ 단계 표 — 노드별·병렬을 보여주는 자리 ★
	if len(run.Steps) > 0 {
		label := map[string]string{}
		for _, a := range run.Assigned {
			for _, n := range a.Nodes {
				label[n.Node] = n.Label
			}
		}
		b.WriteString("| # | 단계 | 상태 | 역할 | 노드 | 걸린 시간 |\n")
		b.WriteString("|---|------|------|------|------|-----------|\n")
		for _, s := range run.Steps {
			node := label[s.Node]
			if node == "" {
				node = s.Node
			}
			if node == "" {
				node = "—"
			}
			uses := s.Uses
			if uses == "" {
				uses = "(사람)"
			}
			fmt.Fprintf(&b, "| %d | `%s` | %s | %s | %s | %s |\n",
				s.Seq, s.ID, s.State, uses, node, took(s.StartedAt, s.EndedAt))
		}
		b.WriteString("\n")
	}

	// ★ verdict — 무엇을 왜 통과·실패했나 ★
	if len(run.Verdict.Checks) > 0 {
		b.WriteString("**판정**\n\n")
		for _, c := range run.Verdict.Checks {
			mark := "✗"
			if c.OK {
				mark = "✓"
			}
			fmt.Fprintf(&b, "- %s `%s` — %s", mark, c.Step, c.What)
			if w := describe(c.Want); w != "" {
				fmt.Fprintf(&b, " (요구 %s", w)
				if g := describe(c.Got); g != "" && !c.OK {
					fmt.Fprintf(&b, " · 실제 %s", g)
				}
				b.WriteString(")")
			}
			if c.Note != "" {
				fmt.Fprintf(&b, " — %s", c.Note)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// ★ 원장 — 무엇이 남았나 ★
	if entries, err := m.Ledger(ctx, run.RunID); err == nil && len(entries) > 0 {
		sort.SliceStable(entries, func(i, j int) bool {
			if entries[i].Seq != entries[j].Seq {
				return entries[i].Seq < entries[j].Seq
			}
			return entries[i].Attempt < entries[j].Attempt
		})
		b.WriteString("<details>\n<summary><b>남은 산출물</b></summary>\n\n")
		b.WriteString("| 이름 | 낸 곳 | 회차 | 바이트 | 스키마 |\n|------|-------|------|--------|--------|\n")
		for _, e := range entries {
			sch := "—"
			if e.SchemaOK != nil {
				if *e.SchemaOK {
					sch = "✓"
				} else {
					sch = "✗"
				}
			}
			fmt.Fprintf(&b, "| `%s` | %s | %d | %d | %s |\n", e.Name, e.By, e.Attempt, e.Bytes, sch)
		}
		b.WriteString("\n</details>\n\n")
	}

	// ★ 봉인의 정본은 우리 쪽에 남는다 ★ (ADR-005 성질 4) — 여기에 붓지 않고
	// 어디서 꺼내는지만 적는다.
	fmt.Fprintf(&b, "<sub>%s · 봉인된 Record 는 `GET /v1/runs/%s/record` 에 있습니다. "+
		"이 코멘트는 그 사본이 아니라 요약입니다.</sub>", ResultMarker(run.RunID), run.RunID)
	return b.String()
}

// ResultMarker 는 ★ 결과를 이미 넘겼다 ★ 는 표지다.
//
// ★ run_id 만으로는 안 된다 ★ — 질문 코멘트의 꼬리표에도 같은 run_id 가 들어
// 있어서, 그것으로 세면 ★ 질문을 쓴 것을 결과를 쓴 것으로 오인한다 ★.
// 첫 실측(EP-2)에서 밟았다: 복구 경로가 「이미 넘겼다」로 판단해 조용히 지나갔다.
func ResultMarker(runID string) string { return "enode:result:" + runID }

// summaryOf 는 report 단계가 낸 summary 산출물을 읽어 사람이 읽게 만든다.
// 없으면 빈 문자열이고, 그러면 코멘트는 표만 싣는다.
//
// ★ 산출물의 모양은 계획이 정한다 ★ — 어댑터가 필드 이름을 알면 계획의
// 스키마를 아는 것이 되고, 계획이 바뀔 때마다 어댑터를 고쳐야 한다.
// 그래서 이름이 아니라 ★ 값의 성질 ★ 로 가른다: 여러 줄인 긴 문자열은
// 사람에게 쓴 글이므로 펼치고, 나머지는 접어서 JSON 으로 둔다.
func summaryOf(ctx context.Context, m *Mediator, runID string) string {
	raw, err := m.Blob(ctx, runID, "summary")
	if err != nil || len(raw) == 0 {
		return ""
	}
	trimmed := strings.TrimSpace(string(raw))

	var obj map[string]json.RawMessage
	if json.Unmarshal([]byte(trimmed), &obj) != nil {
		// JSON 객체가 아니면 글이거나 배열이다. 글이면 그대로 싣는다.
		return clip(trimmed, 8000)
	}

	var prose, rest []string
	restObj := map[string]any{}
	names := make([]string, 0, len(obj))
	for k := range obj {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		var s string
		if json.Unmarshal(obj[k], &s) == nil && strings.Contains(s, "\n") && len(s) > 200 {
			prose = append(prose, clip(strings.TrimSpace(s), 12000))
			continue
		}
		var v any
		_ = json.Unmarshal(obj[k], &v)
		restObj[k] = v
	}
	if len(prose) > 0 {
		rest = append(rest, strings.Join(prose, "\n\n"))
	}
	if len(restObj) > 0 {
		b, err := json.MarshalIndent(restObj, "", "  ")
		if err == nil {
			label := "산출물 원본"
			if len(prose) == 0 {
				label = "report 가 낸 것"
			}
			rest = append(rest, "<details>\n<summary><b>"+label+"</b></summary>\n\n```json\n"+
				clip(string(b), 8000)+"\n```\n\n</details>")
		}
	}
	return strings.Join(rest, "\n\n")
}

func clip(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n\n…(잘렸습니다. 전문은 봉인된 Record 에 있습니다)"
}

// oneLine 은 result 의 output 에 넣을 ★ 한 줄 ★ 이다.
// ★ 봉인을 여기 붓지 않는다 ★ — agent_run.output 은 지울 수 있고 스키마도 없다.
func oneLine(run *RunView, runID string) string {
	ok, total := 0, len(run.Verdict.Checks)
	for _, c := range run.Verdict.Checks {
		if c.OK {
			ok++
		}
	}
	return fmt.Sprintf("%s — 판정 %d/%d 통과 · run %s", run.State, ok, total, runID)
}

// describe 는 다형인 want·got 을 사람이 읽는 한 조각으로 만든다.
//
// ★ 형을 단정하지 않는다 ★ — produced 는 배열이고 exit_code 는 숫자이며,
// 함대 조건(ADR-058)은 또 다른 모양일 수 있다. 어휘가 창발하므로(ADR-012)
// 여기서 열거하면 새 조건이 생길 때마다 어댑터가 조용히 못 읽는다.
func describe(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return "`" + t + "`"
	case float64:
		if t == float64(int64(t)) {
			return fmt.Sprintf("`%d`", int64(t))
		}
		return fmt.Sprintf("`%v`", t)
	case bool:
		return fmt.Sprintf("`%v`", t)
	case []any:
		if len(t) == 0 {
			return ""
		}
		parts := make([]string, 0, len(t))
		for _, e := range t {
			if s := describe(e); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, ", ")
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return "`" + string(b) + "`"
	}
}

func prettyJSON(raw json.RawMessage) string {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return strings.TrimSpace(string(raw))
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return strings.TrimSpace(string(raw))
	}
	s := string(b)
	const max = 8000
	if len(s) > max {
		s = s[:max] + "\n…(잘렸습니다)"
	}
	return s
}

func took(start, end string) string {
	if start == "" || end == "" {
		return "—"
	}
	s, err1 := parseTime(start)
	e, err2 := parseTime(end)
	if err1 != nil || err2 != nil {
		return "—"
	}
	d := e.Sub(s)
	if d < 0 {
		return "—"
	}
	return d.Round(100_000_000).String()
}
