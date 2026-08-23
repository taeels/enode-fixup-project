package contract

import (
	"encoding/json"
	"strings"
	"testing"
)

// ★ 문법이 낡으면 여기서 깨진다 ★ (ADR-045)
//
// Grammar 는 사람이 읽는 텍스트이고 Validate() 는 기계가 강제하는 규칙이다.
// ★ 둘이 갈라지는 것이 정확히 우리가 아홉 판 동안 겪은 사고다 ★ —
// 사람이 매 판 Validate 의 일부를 자연어로 번역했고, 번역이 축약되고 모순됐다.
//
// 그래서 문법의 ★ 각 문장마다 ★ "그 규칙을 어긴 계약이 실제로 거절되는가" 를 잰다.
// Validate 가 늘었는데 Grammar 가 안 늘면 이 표에 항목이 없어 통과하지만,
// ★ Grammar 가 말한 것이 거짓이 되면 즉시 깨진다 ★.
// go.mod 의 toolchain 을 CI 가 go-version-file 로 읽는 것과 같은 장치다.

// bad 는 ★ 문법이 금지한다고 적은 것 ★ 을 실제로 어긴 계약이다.
type bad struct {
	name     string // 문법의 어느 문장인가
	mustSay  string // Grammar 에 이 문구가 있어야 한다
	contract string
	wantErr  string // 거절 사유에 이 문구가 있어야 한다
}

const okStep = `{"id":"a","uses":"n","run":["true"],"out":["log"]}`

func Test문법_금지한다고_적은_것은_실제로_거절된다(t *testing.T) {
	cases := []bad{
		{
			name:    "ask 에 uses 를 적으면 거절된다",
			mustSay: "ask 에 uses 를 적으면 거절된다",
			contract: `{"run_id":"r","requires":[{"as":"n","capability":"agent.reason"}],
			  "steps":[` + okStep + `,
			   {"id":"q","uses":"n","ask":{"prompt":"?"},"out":["ans"],
			    "schema":{"ans":{"type":"object","properties":{"v":{"type":"string"}}}}}],
			  "success_when":[{"step":"a","exit_code":0}]}`,
			wantErr: "ask 단계에는 uses 가 없다",
		},
		{
			name:    "에이전트 단계에 exit_code 를 걸면 거절된다",
			mustSay: "에이전트 단계에 exit_code 를 걸면 거절된다",
			contract: `{"run_id":"r","requires":[{"as":"n","capability":"agent.reason"}],
			  "steps":[{"id":"a","uses":"n","agent":{},"in":{"prompt":"p"},"out":["x"]}],
			  "success_when":[{"step":"a","exit_code":0}]}`,
			wantErr: "exit_code",
		},
		{
			name:    "success_when 은 실존하는 단계만 가리킬 수 있다",
			mustSay: "success_when 은 ★ 실존하는 단계 ★ 만 가리킬 수 있다",
			contract: `{"run_id":"r","requires":[{"as":"n","capability":"agent.reason"}],
			  "steps":[` + okStep + `],
			  "success_when":[{"step":"없는단계","exit_code":0}]}`,
			wantErr: "없는 단계",
		},
		{
			name:    "재계획은 out 이 정확히 하나여야 한다",
			mustSay: "out ★ 정확히 하나 ★",
			contract: `{"run_id":"r","requires":[{"as":"n","capability":"agent.reason"}],
			  "steps":[{"id":"p","uses":"n","expands":true,"agent":{},"in":{"prompt":"x"},
			            "out":["a","b"],"schema":{"a":{"type":"object"}}}],
			  "success_when":[{"step":"p","produced":["a"]}]}`,
			wantErr: "정확히 하나",
		},
		{
			name:    "재계획의 산출물에는 스키마가 필수다",
			mustSay: "schema ★ 필수 ★",
			contract: `{"run_id":"r","requires":[{"as":"n","capability":"agent.reason"}],
			  "steps":[{"id":"p","uses":"n","expands":true,"agent":{},"in":{"prompt":"x"},
			            "out":["plan"]}],
			  "success_when":[{"step":"p","produced":["plan"]}]}`,
			wantErr: "스키마가 없다",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// ① 문법이 그 말을 실제로 하고 있는가
			if !strings.Contains(Grammar, c.mustSay) {
				t.Fatalf("★ 문법에 %q 가 없다 ★ — Grammar 가 낡았거나 이 시험이 낡았다", c.mustSay)
			}
			// ② 그 말대로 실제로 거절되는가
			var ct Contract
			if err := json.Unmarshal([]byte(c.contract), &ct); err != nil {
				t.Fatalf("시험 계약이 JSON 이 아니다: %v", err)
			}
			err := ct.Validate()
			if err == nil {
				t.Fatalf("★ 문법은 거절한다고 적었는데 통과했다 ★")
			}
			if !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("다른 이유로 거절됐다: %v (기대: %q)", err, c.wantErr)
			}
		})
	}
}

// ★ 스키마 어휘 목록이 갈라지지 않는지 ★ — 문법이 나열한 이름과
// schema 패키지가 허용/거절하는 이름이 같아야 한다.
func Test문법_스키마어휘가_실제와_같다(t *testing.T) {
	for _, name := range []string{
		"type", "required", "properties", "enum", "items",
		"additionalProperties", "title", "description",
	} {
		if !strings.Contains(Grammar, name) {
			t.Errorf("★ 허용 어휘 %q 가 문법에 없다 ★", name)
		}
	}
	for _, name := range []string{
		"minimum", "maximum", "minLength", "maxLength",
		"pattern", "format", "minItems", "maxItems",
	} {
		if !strings.Contains(Grammar, name) {
			t.Errorf("★ 거절 어휘 %q 가 문법에 없다 ★ — 계획이 그것을 쓰고 422 를 받는다", name)
		}
	}
}

// ★ 빈 계획이 값이라는 것을 문법이 말해야 한다 ★ (ADR-043)
// 이 문장이 없으면 계획은 고칠 것이 없을 때도 억지로 단계를 지어낸다 — 실측에서 밟았다.
func Test문법_빈계획을_말한다(t *testing.T) {
	for _, want := range []string{`"steps": []`, "오류가 아니라 판단"} {
		if !strings.Contains(Grammar, want) {
			t.Errorf("★ 문법에 %q 가 없다 ★", want)
		}
	}
}
