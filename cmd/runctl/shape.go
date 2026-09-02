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
		fmt.Fprintln(os.Stderr, "  runctl example  로 도는 계약을 하나 받아 견줘 본다")
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
	var out []string

	// ADR-066 의 출발점이다. success_when 이 비면 어떤 단계가 실패해도 Run 이
	// 성공한다 — 실측에서 exit_code 1 에 산출물도 없는 Run 이 SUCCEEDED 로 끝났다.
	if len(c.SuccessWhen) == 0 {
		out = append(out, "warning: success_when 이 비어 있다. 어떤 단계가 실패해도 Run 은 성공한다")
		if len(c.Steps) > 0 {
			s := c.Steps[0]
			out = append(out, "  이렇게 적을 수 있다:")
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
			out = append(out, fmt.Sprintf("warning: 단계 %q 를 판정하는 조건이 없다. 그 단계는 실패해도 Run 이 성공한다", s.ID))
			out = append(out, "  "+suggestCondition(s))
		}
	}
	// 조건이 없는 단계를 가리키면 그 조건은 영원히 안 맞는다.
	ids := map[string]bool{}
	for _, s := range c.Steps {
		ids[s.ID] = true
	}
	for _, cond := range c.SuccessWhen {
		if cond.Step != "" && !ids[cond.Step] {
			out = append(out, fmt.Sprintf("warning: success_when 이 없는 단계 %q 를 가리킨다", cond.Step))
		}
	}
	return out
}

// suggestCondition 은 그 단계에 맞는 조건 한 줄을 만든다.
//
// exit_code 는 명령 단계에만 쓸 수 있다 (ADR-019) — 하네스는 헛소리를 하고도
// 0 으로 끝나므로 agent 단계에 걸면 판정이 아니라 장식이 된다.
func suggestCondition(s contract.Step) string {
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
		return fmt.Sprintf(`{ "step": "%s", "produced": ["<산출물 이름>"] }`, s.ID)
	}
	return fmt.Sprintf(`{ "step": "%s"%s }`, s.ID, strings.TrimPrefix(produced, ", "))
}

// cmdSchema 는 계약의 필드 어휘를 낸다.
//
// 손으로 적지 않고 구조체에서 뽑는다 — 손으로 적으면 필드가 늘 때 같이 안
// 늘고, 그러면 이 명령이 있는 이유가 사라진다. 대신 「왜 그 필드가 있는가」는
// 못 낸다. 그 답은 예시와 protocol/run-contract.md 에 있다.
//
// 예시가 빠르고 스키마가 정확하다. 처음 오는 쪽에게는 example 을 먼저 권한다.
func cmdSchema(section string) int {
	type sect struct {
		name string
		typ  reflect.Type
		note string
	}
	sects := []sect{
		{"contract", reflect.TypeOf(contract.Contract{}), "계약 전체"},
		{"steps", reflect.TypeOf(contract.Step{}), "steps[] 의 한 칸"},
		{"requires", reflect.TypeOf(contract.Require{}), "requires[] 의 한 칸. 여기 없는 키는 전부 매칭 속성이 된다"},
		{"success_when", reflect.TypeOf(contract.Condition{}), "success_when[] 의 한 칸"},
	}
	if section == "" {
		for _, s := range sects {
			fmt.Printf("%-14s %s\n", s.name, s.note)
		}
		fmt.Println("\n  runctl schema <section>   그 칸의 필드를 본다")
		fmt.Println("  runctl example            먼저 이것을 보는 편이 빠르다")
		return exitOK
	}
	for _, s := range sects {
		if s.name != section {
			continue
		}
		fmt.Printf("%s — %s\n\n", s.name, s.note)
		printFields(s.typ)
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
		required := "필수"
		if strings.Contains(opts, "omitempty") || f.Type.Kind() == reflect.Ptr {
			required = "선택"
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
