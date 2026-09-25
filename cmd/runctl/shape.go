package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/taeels/enode/internal/contract"
)

// 계약을 쓰는 쪽을 돕는 셋이다 (ADR-066).
//
// 셋 다 Mediator 를 안 거친다. 거치게 하면 「계약을 어떻게 쓰나」를 물으려고
// 먼저 함대를 세워야 하고, 그러면 있어야 하는 이유가 사라진다.

// cmdExample 은 붙여넣으면 도는 계약을 낸다.
//
// 이름이 없으면 목록을 낸다 — 처음 오는 쪽은 무엇을 물어야 하는지도 모른다.
func cmdExample(name string) int {
	if name == "" {
		fmt.Println("ready-to-run examples:")
		for _, n := range contract.ExampleNames() {
			fmt.Printf("  %-10s runctl example %s\n", n, n)
		}
		fmt.Println("\neach one is valid as printed. pipe it straight in:")
		fmt.Println("  runctl example command > c.json && runctl dry-run c.json")
		return exitOK
	}
	b, err := contract.Example(name)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitRequest
	}
	os.Stdout.Write(b) //nolint:errcheck
	return exitOK
}

// cmdLint 는 제출하기 전에 계약을 본다.
//
// dry-run 과 다른 것을 묻는다 — dry-run 은 「이 요구를 만족하는 노드가 함대에
// 있는가」이고, 이것은 「이 계약이 내가 원하는 것을 말하고 있는가」다. 둘은
// 갈린다: success_when 이 없는 계약도 dry-run 은 통과한다.
func cmdLint(path string) int {
	if path == "" {
		fmt.Fprintln(os.Stderr, "usage: runctl lint <contract.json>")
		return exitRequest
	}
	b, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitRequest
	}
	var c contract.Contract
	if err := json.Unmarshal(b, &c); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
		fmt.Fprintln(os.Stderr, "  runctl example   print a valid one to compare against")
		return exitRequest
	}
	if err := c.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
		return exitRequest
	}
	warnings := lintWarnings(c)
	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, w)
	}
	if len(warnings) == 0 {
		fmt.Printf("%s: ok  (%d steps, %d conditions)\n", path, len(c.Steps), len(c.SuccessWhen))
	}
	// 경고는 거절이 아니다 — 계약이 유효한데 뜻이 헐거운 것이고, 그 판단은
	// 계약 저자의 몫이다. 종료코드로 막으면 시스템이 뜻을 대신 정하게 된다.
	return exitOK
}

// lintWarnings 는 유효하지만 뜻이 헐거운 자리를 찾는다.
func lintWarnings(c contract.Contract) []string {
	return append(conditionWarnings(c), bakeWarnings(c)...)
}

// conditionWarnings 는 판정 조건이 비었거나 어느 단계를 안 보는 자리를 찾는다.
func conditionWarnings(c contract.Contract) []string {
	var out []string

	// ADR-066 의 출발점이다. success_when 이 비면 어떤 단계가 실패해도 Run 이
	// 성공한다 — 실측에서 exit_code 1 에 산출물도 없는 Run 이 SUCCEEDED 로 끝났다.
	if len(c.SuccessWhen) == 0 {
		out = append(out, "warning: success_when is empty; the run succeeds even if every step fails")
		if len(c.Steps) > 0 {
			s := c.Steps[0]
			out = append(out, "  add a condition like:")
			out = append(out, "    "+suggestCondition(s))
		}
		return out
	}

	judged := map[string]bool{}
	for _, cond := range c.SuccessWhen {
		judged[cond.Step] = true
	}
	for _, s := range c.Steps {
		if !judged[s.ID] {
			out = append(out, fmt.Sprintf("warning: no condition judges step %q; it can fail and the run still succeeds", s.ID))
			out = append(out, "  "+suggestCondition(s))
		}
	}
	// 없는 단계를 가리키는 조건은 여기서 안 본다 — Validate 가 이미 거절하므로
	// 이 자리까지 오지 않는다. 겹쳐 두면 도달하지 않는 코드가 하나 는다.
	return out
}

// suggestCondition 은 그 단계에 맞는 조건 한 줄을 만든다.
//
// exit_code 는 명령 단계에만 쓸 수 있다 (ADR-019) — 하네스는 헛소리를 하고도
// 0 으로 끝나므로 agent 단계에 걸면 판정이 아니라 장식이 된다.
func suggestCondition(s contract.Step) string {
	// 굽기 두 단계는 고정 산출물 하나로만 판정한다 — 다른 조건은 Validate 가 거절한다.
	switch k, _ := s.Kind(); k {
	case contract.KindBuild:
		return fmt.Sprintf(`{ "step": "%s", "produced": ["%s"] }`, s.ID, contract.ArtifactManifest)
	case contract.KindMerge:
		return fmt.Sprintf(`{ "step": "%s", "produced": ["%s"] }`, s.ID, contract.ArtifactMerged)
	}
	produced := ""
	if len(s.Out) > 0 {
		quoted := make([]string, 0, len(s.Out))
		for _, o := range s.Out {
			quoted = append(quoted, `"`+o+`"`)
		}
		produced = fmt.Sprintf(`, "produced": [%s]`, strings.Join(quoted, ", "))
	}
	if len(s.Run) > 0 {
		return fmt.Sprintf(`{ "step": "%s", "exit_code": 0%s }`, s.ID, produced)
	}
	if produced == "" {
		return fmt.Sprintf(`{ "step": "%s", "produced": ["<artifact name>"] }`, s.ID)
	}
	return fmt.Sprintf(`{ "step": "%s"%s }`, s.ID, strings.TrimPrefix(produced, ", "))
}

// bakeWarnings 는 굽기 계약이 뜻대로 안 돌 자리를 찾는다.
//
// 둘 다 거절이 아니라 경고다.
//
// workspace.writes — 굽기는 공유 lower 위의 격리 runtime 에서만 돈다 (ADR-077 이
// 「공유 lower 위에 native 굽기 노드」를 기각했다). Validate 가 이것을 막으면 광고
// 어휘가 문법으로 들어와 매칭의 어휘와 두 벌이 된다. 그래서 여기서 짚는다.
//
// 계획이 지을 굽기 — 제출 때에는 굽기가 아직 없으므로 Validate 가 승인 규칙을
// 못 건다. 계획이 굽기를 지어 붙이는 순간에야 거절되고, 그때는 계획 한 판이
// 버려진 뒤다. success_when 이 약속한 이름에 manifest 나 merged 를 걸었으면
// 굽기를 지을 것이라는 단서이고, 제출 때 볼 수 있는 셋을 미리 본다.
func bakeWarnings(c contract.Contract) []string {
	var out []string
	required := map[string]contract.Require{}
	for _, r := range c.Requires {
		required[r.As] = r
	}
	for _, st := range c.Steps {
		if st.Acquire != nil && st.Acquire.Want != nil {
			required[st.Acquire.Want.As] = *st.Acquire.Want
		}
	}
	for _, st := range c.Steps {
		if k, _ := st.Kind(); k != contract.KindBuild {
			continue
		}
		if r, ok := required[st.Uses]; ok && r.Attrs["workspace.writes"] != "isolated" {
			out = append(out, fmt.Sprintf("warning: build step %q uses role %q, which does not require "+
				"workspace.writes=isolated; a node that writes in place cannot bake", st.ID, st.Uses))
			out = append(out, fmt.Sprintf(`  add "workspace.writes": "isolated" to requires[] entry %q`, st.Uses))
		}
	}

	exists := map[string]bool{}
	for _, st := range c.Steps {
		exists[st.ID] = true
	}
	warned := map[string]bool{}
	for _, cond := range c.SuccessWhen {
		if exists[cond.Step] {
			continue
		}
		artifact := ""
		for _, p := range cond.Produced {
			if p == contract.ArtifactManifest || p == contract.ArtifactMerged {
				artifact = p
				break
			}
		}
		if artifact == "" {
			continue
		}
		for _, plan := range c.Steps {
			if !plan.Expands || warned[plan.ID] || !contains(plan.Produces, cond.Step) {
				continue
			}
			if why := plannedBakeGap(c, plan); why != "" {
				warned[plan.ID] = true
				out = append(out, fmt.Sprintf("warning: the plan of step %q is expected to bake "+
					"(success_when waits for %q), but %s; the plan will be rejected when it is attached",
					plan.ID, artifact, why))
			}
		}
	}
	return out
}

// plannedBakeGap 은 계획이 지은 굽기를 승인하는 설정 가운데 제출 때 볼 수 있는
// 셋에서 빠진 것 하나를 말한다. 넷째(build 가 그 ask 를 기다리나)는 계획이 짓는
// build 의 needs 에 달려 있어 제출 때 모른다.
func plannedBakeGap(c contract.Contract, plan contract.Step) string {
	if plan.Adopt == contract.AdoptYolo {
		return `it adopts the plan with adopt "yolo"`
	}
	for _, st := range c.Steps {
		if st.Ask == nil || st.Ask.Adopts != plan.ID {
			continue
		}
		if st.Ask.AdoptWhen == "" && st.Dispatch == nil {
			return fmt.Sprintf("ask step %q does not set adopt_when", st.ID)
		}
		return ""
	}
	return "no ask step adopts it"
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

// cmdSchema 는 계약의 필드 어휘를 낸다.
//
// 손으로 적지 않고 구조체에서 뽑는다 — 손으로 적으면 필드가 늘 때 같이 안
// 늘고, 그러면 이 명령이 있는 이유가 사라진다. 대신 「왜 그 필드가 있는가」는
// 못 낸다. 그 답은 예시와 protocol/run-contract.md 에 있다.
//
// 예시가 빠르고 스키마가 정확하다. 처음 오는 쪽에게는 example 을 먼저 권한다.
func cmdSchema(section string) int {
	// typ 이 nil 인 절은 글이다.
	//
	// 구조체에서 뽑는 것만으로는 안 보이는 값이 있다 — 팩 tar 의 배치와
	// $OUT 규약은 필드가 아니라 규약이고, 필드 목록에는 나타날 자리가 없다.
	// 그것을 안 적어 두면 문맥 없는 에이전트가 실물 기계 위에서 Run 을
	// 던져 가며 맞혀야 한다 (decisions.md 6절 ㉕ 의 실측 — 스물넷 중 열다섯).
	type sect struct {
		name string
		typ  reflect.Type
		note string
		text string // typ 이 nil 일 때 찍는다
		tail string // 필드 뒤에 붙는 줄
	}
	sects := []sect{
		{name: "contract", typ: reflect.TypeOf(contract.Contract{}), note: "the whole contract"},
		{name: "steps", typ: reflect.TypeOf(contract.Step{}), note: "one entry of steps[]",
			tail: "\nwhat a step gets (cwd, $OUT, $IN):\n  runctl schema io"},
		{name: "requires", typ: reflect.TypeOf(contract.Require{}),
			note: "one entry of requires[]; any other key is a matching attribute",
			tail: "\nthe attribute values you write here are the ones the fleet advertises:\n  runctl capabilities"},
		{name: "success_when", typ: reflect.TypeOf(contract.Condition{}), note: "one entry of success_when[]"},
		{name: "io", note: "what a step gets: cwd, $OUT, $IN", text: schemaIO},
		{name: "pack", note: "the pack tar layout and the two keys that carry it", text: schemaPack},
	}
	if section == "" {
		for _, s := range sects {
			fmt.Printf("%-14s %s\n", s.name, s.note)
		}
		fmt.Println("\n  runctl schema <section>   fields of that section")
		fmt.Println("  runctl example            usually faster to start here")
		return exitOK
	}
	for _, s := range sects {
		if s.name != section {
			continue
		}
		fmt.Printf("%s — %s\n\n", s.name, s.note)
		if s.typ == nil {
			fmt.Print(s.text)
			return exitOK
		}
		printFields(s.typ)
		if s.tail != "" {
			fmt.Println(s.tail)
		}
		return exitOK
	}
	names := make([]string, 0, len(sects))
	for _, s := range sects {
		names = append(names, s.name)
	}
	fmt.Fprintf(os.Stderr, "no such section: %s  (have: %s)\n", section, strings.Join(names, " · "))
	return exitRequest
}

// printFields 는 JSON 태그를 그대로 읽는다 — 와이어에 나가는 이름이 그것이다.
func printFields(t reflect.Type) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		tag := f.Tag.Get("json")
		if tag == "-" {
			continue
		}
		name, opts, _ := strings.Cut(tag, ",")
		if name == "" {
			name = f.Name // 태그가 없으면 커스텀 마샬러가 있다는 뜻이다
		}
		required := "required"
		if strings.Contains(opts, "omitempty") || f.Type.Kind() == reflect.Ptr {
			required = "optional"
		}
		fmt.Printf("  %-14s %-24s %s\n", name, typeName(f.Type), required)
	}
}

func typeName(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Ptr:
		return typeName(t.Elem())
	case reflect.Slice:
		return "[]" + typeName(t.Elem())
	case reflect.Map:
		return "map[" + typeName(t.Key()) + "]" + typeName(t.Elem())
	case reflect.Interface:
		return "any"
	}
	if n := t.Name(); n != "" {
		return n
	}
	return t.String()
}

// schemaIO 는 단계가 무엇을 받는가다.
//
// 이 값들은 오케스트레이터가 받는 계획 프롬프트에는 있었고 schema 에는 없었다.
// 그래서 손으로 계약을 쓰는 사람은 실물 기계에 프로브를 던져 알아내야 했다
// (decisions.md 6절 ㉕).
const schemaIO = `  cwd            the workspace on the node. a run step starts there
  $OUT           the only place a step's results are collected from.
                 out: ["name"] means the step wrote $OUT/name
  $IN            what earlier steps produced, laid out by blob name.
                 in: {"from": ["name"]} puts that blob at $IN/name
                 the files are read-only and so is the directory

both are absolute paths in the step's environment. nothing outside $OUT is
collected. a fresh shell runs each step, so export what you need again.
`

// schemaPack 은 팩 tar 의 배치다.
//
// 거절 문구는 슬롯 셋을 이름으로 말한다 (pack %s carries no skills, agents,
// or mcp servers). 안 말하는 것이 그 한 층 아래다 — 그 슬롯이 디스크에서
// 무엇으로 불리는지. 이 글이 그 자리다.
const schemaPack = `a pack is one tar. three slots are read; anything else is ignored and named
in the node log.

  skills/<name>/SKILL.md     a skill. files beside it travel with it
  agents/<name>.md           a subagent
  mcp.json                   {"mcpServers": {"<name>": {"command": "..."}}}
                             the same shape a workspace .mcp.json uses

two keys carry it into a step, and a step needs both:

  "agent": {"pack": "<blob>"}     run this step with that pack
  "in":    {"from": ["<blob>"]}   lay the blob down at $IN/<blob>

the blob comes from an earlier step that wrote out: ["<blob>"].

building it:

  tar -cf pack.tar skills agents mcp.json     name the slots
  tar -cf pack.tar -C dir skills mcp.json     or name them under -C
  tar -cf pack.tar -C dir .                   refused. that form writes a "./"
                                              entry and it comes back as
                                              "pack entry ./ has an unsafe name"

gzip is unwrapped for you. a pack carrying none of the three slots is refused
rather than run empty, because a harness ignores a missing pack in silence.

a skill from a pack is named pack:<name> in the session - the directory the
pack is spread into becomes the prefix.

  runctl example pack   a contract that builds one and uses it
`
