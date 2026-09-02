package contract

import (
	"encoding/json"
	"strings"
	"testing"
)

// 예시를 시험으로 건다 (ADR-066 §5).
//
// 문서는 낡아도 아무것도 빨개지지 않는다. 예시가 시험에 걸려 있으면 계약
// 어휘가 바뀔 때 함께 깨지므로 둘이 갈릴 수 없다. 이것이 「문서를 더 잘
// 쓴다」와 갈리는 지점이다.
func TestExamples_ParseAndValidate(t *testing.T) {
	names := ExampleNames()
	if len(names) == 0 {
		t.Fatal("no examples; the embed is empty or the files are gone")
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			b, err := Example(name)
			if err != nil {
				t.Fatalf("Example(%q) = %v", name, err)
			}
			var c Contract
			if err := json.Unmarshal(b, &c); err != nil {
				t.Fatalf("example must parse as printed: %v", err)
			}
			if err := c.Validate(); err != nil {
				t.Fatalf("Validate = %v", err)
			}
		})
	}
}

// 예시가 바로 그 실수를 재현하면 안 된다.
//
// ADR-066 의 출발점이 success_when 을 빠뜨린 계약이었고, 그 계약은 명령이
// 실패했는데도 SUCCEEDED 로 끝났다. 예시를 보고 베끼는 쪽이 같은 자리에
// 빠지지 않아야 한다.
func TestExamples_EveryStepHasASuccessCondition(t *testing.T) {
	for _, name := range ExampleNames() {
		t.Run(name, func(t *testing.T) {
			b, _ := Example(name)
			var c Contract
			if err := json.Unmarshal(b, &c); err != nil {
				t.Fatal(err)
			}
			if len(c.SuccessWhen) == 0 {
				t.Fatal("success_when is empty; the run would succeed even if every step failed")
			}
			judged := map[string]bool{}
			for _, cond := range c.SuccessWhen {
				judged[cond.Step] = true
			}
			for _, s := range c.Steps {
				if !judged[s.ID] {
					t.Errorf("no condition judges step %q", s.ID)
				}
			}
		})
	}
}

// 이름이 없으면 무엇이 있는지 말해야 한다 — 없는 것을 물었을 때 다음 행동을
// 알려주는 것이 이 ADR 의 §4.4 와 같은 성질이다.
func TestExample_UnknownNameListsWhatExists(t *testing.T) {
	_, err := Example("nope")
	if err == nil {
		t.Fatal("Example(\"nope\") = nil, want an error")
	}
	for _, name := range ExampleNames() {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error = %v, want it to list %q", err, name)
		}
	}
}
