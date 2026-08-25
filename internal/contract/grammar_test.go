package contract

import (
	"encoding/json"
	"strings"
	"testing"
)

// 문법이 낡으면 여기서 깨진다 (ADR-045)
//
// Grammar 는 사람이 읽는 텍스트이고 Validate() 는 기계가 강제하는 규칙이다.
// 둘이 갈라지는 것이 정확히 우리가 아홉 판 동안 겪은 사고다 —
// 사람이 매 판 Validate 의 일부를 자연어로 번역했고, 번역이 축약되고 모순됐다.
//
// 그래서 문법의 각 문장마다 "그 규칙을 어긴 계약이 실제로 거절되는가" 를 잰다.
// Validate 가 늘었는데 Grammar 가 안 늘면 이 표에 항목이 없어 통과하지만,
// Grammar 가 말한 것이 거짓이 되면 즉시 깨진다.
// go.mod 의 toolchain 을 CI 가 go-version-file 로 읽는 것과 같은 장치다.

// bad 는 문법이 금지한다고 적은 것을 실제로 어긴 계약이다.
type bad struct {
	name     string // which sentence of the grammar
	mustSay  string // Grammar must contain this phrase
	contract string
	wantErr  string // the rejection reason must contain this phrase
}

const okStep = `{"id":"a","uses":"n","run":["true"],"out":["log"]}`

func TestGrammar_WhatItForbidsIsActuallyRejected(t *testing.T) {
	cases := []bad{
		{
			// 사람의 답에는 프로세스가 없다 — 조건이 통과하면 Verify 가
			// got=-1 로 비교해 언제나 거짓이 되고, 저자는 왜인지 못 본다.
			name:    "exit_code is allowed only on a run step (ask)",
			mustSay: "exit_code is allowed only on a run step",
			contract: `{"run_id":"r","requires":[{"as":"n","capability":"agent.reason"}],
			  "steps":[` + okStep + `,
			   {"id":"q","needs":["a"],"ask":{"prompt":"?"},"out":["ans"],
			    "schema":{"ans":{"type":"object","properties":{"v":{"type":"string"}}}}}],
			  "success_when":[{"step":"q","exit_code":0}]}`,
			wantErr: "exit_code",
		},
		{
			// 같은 곳을 두 번 가리키면 갈림길이 아니다
			name:    "dispatch.to needs at least two targets, all distinct",
			mustSay: "all distinct",
			contract: `{"run_id":"r","requires":[{"as":"n","capability":"agent.reason"}],
			  "steps":[{"id":"s","uses":"n","run":["true"],"out":["found"],
			    "schema":{"found":{"type":"object","required":["next"],
			      "properties":{"next":{"enum":["a"]}}}},
			    "dispatch":{"from":"found.next","to":["a","a"]}},
			   {"id":"a","uses":"n","needs":["s"],"run":["true"],"out":["l"]}],
			  "success_when":[{"step":"s","exit_code":0}]}`,
			wantErr: "duplicate",
		},
		{
			// 없는 곳으로는 갈 수 없다
			name:    "every branch target must exist",
			mustSay: "every branch target must exist and come after this step",
			contract: `{"run_id":"r","requires":[{"as":"n","capability":"agent.reason"}],
			  "steps":[{"id":"s","uses":"n","run":["true"],"out":["found"],
			    "schema":{"found":{"type":"object","required":["next"],
			      "properties":{"next":{"enum":["a","nosuch"]}}}},
			    "dispatch":{"from":"found.next","to":["a","nosuch"]}},
			   {"id":"a","uses":"n","needs":["s"],"run":["true"],"out":["l"]}],
			  "success_when":[{"step":"s","exit_code":0}]}`,
			wantErr: "unknown step",
		},
		{
			// capability 어휘는 닫혀 있다 — 문법이 acquire 예시에서
			// build.zephyr 를 가르쳤다가 이 시험이 잡았다 (실제로 밟았다).
			name:    "capability is a closed vocabulary",
			mustSay: "capability is a closed vocabulary",
			contract: `{"run_id":"r","requires":[{"as":"n","capability":"agent.reason"}],
			  "steps":[{"id":"g","acquire":{"want":{"as":"b","capability":"build.zephyr"},
			     "acquired":"a","unavailable":"b2"}},
			   {"id":"a","uses":"n","needs":["g"],"run":["true"],"out":["l"]},
			   {"id":"b2","uses":"n","needs":["g"],"run":["true"],"out":["l2"]}],
			  "success_when":[{"step":"a","exit_code":0}]}`,
			wantErr: "capability",
		},
		{
			// 획득도 갈림길이다 — 목적지 규칙이 dispatch 와 같다.
			name:    "every branch target must exist and come after this step (acquire)",
			mustSay: "every branch target must exist and come after this step",
			contract: `{"run_id":"r","requires":[{"as":"n","capability":"agent.reason"}],
			  "steps":[{"id":"a","uses":"n","run":["true"],"out":["l"]},
			   {"id":"g","needs":["a"],"acquire":{"want":{"as":"b","capability":"agent.reason"},
			     "acquired":"a","unavailable":"z"}},
			   {"id":"z","uses":"n","needs":["g"],"run":["true"],"out":["l3"]}],
			  "success_when":[{"step":"a","exit_code":0}]}`,
			wantErr: "forward",
		},
		{
			// 갈림길이 하나면 갈림길이 아니다 (ADR-053 이 문법에 적었다)
			name:    "dispatch.to needs at least two targets",
			mustSay: "dispatch.to needs at least two targets",
			contract: `{"run_id":"r","requires":[{"as":"n","capability":"agent.reason"}],
			  "steps":[{"id":"s","uses":"n","run":["true"],"out":["found"],
			    "schema":{"found":{"type":"object","required":["next"],
			      "properties":{"next":{"enum":["a"]}}}},
			    "dispatch":{"from":"found.next","to":["a"]}},
			   {"id":"a","uses":"n","needs":["s"],"run":["true"],"out":["l"]}],
			  "success_when":[{"step":"s","exit_code":0}]}`,
			wantErr: "at least two targets",
		},
		{
			// 뒤로 못 간다 = DAG = 종료가 정적으로 보장된다
			name:    "every branch target must exist and come after this step",
			mustSay: "every branch target must exist and come after this step",
			contract: `{"run_id":"r","requires":[{"as":"n","capability":"agent.reason"}],
			  "steps":[{"id":"a","uses":"n","run":["true"],"out":["l"]},
			   {"id":"b","uses":"n","needs":["a"],"run":["true"],"out":["l2"]},
			   {"id":"s","uses":"n","needs":["b"],"run":["true"],"out":["found"],
			    "schema":{"found":{"type":"object","required":["next"],
			      "properties":{"next":{"enum":["a","b"]}}}},
			    "dispatch":{"from":"found.next","to":["a","b"]}}],
			  "success_when":[{"step":"s","exit_code":0}]}`,
			wantErr: "branches must point forward",
		},
		{
			// dispatch.from 은 그 단계가 내는 산출물을 가리켜야 한다
			name:    "dispatch.from must name a field inside an output this step produces",
			mustSay: "dispatch.from must name a field inside an output this step produces",
			contract: `{"run_id":"r","requires":[{"as":"n","capability":"agent.reason"}],
			  "steps":[{"id":"s","uses":"n","run":["true"],"out":["found"],
			    "schema":{"found":{"type":"object","properties":{"next":{"enum":["a","b"]}}}},
			    "dispatch":{"from":"nosuch.next","to":["a","b"]}},
			   {"id":"a","uses":"n","needs":["s"],"run":["true"],"out":["l"]},
			   {"id":"b","uses":"n","needs":["s"],"run":["true"],"out":["l2"]}],
			  "success_when":[{"step":"s","exit_code":0}]}`,
			wantErr: "does not produce",
		},
		{
			name:    "An ask step must not set uses",
			mustSay: "An ask step must not set uses",
			contract: `{"run_id":"r","requires":[{"as":"n","capability":"agent.reason"}],
			  "steps":[` + okStep + `,
			   {"id":"q","uses":"n","ask":{"prompt":"?"},"out":["ans"],
			    "schema":{"ans":{"type":"object","properties":{"v":{"type":"string"}}}}}],
			  "success_when":[{"step":"a","exit_code":0}]}`,
			wantErr: "must not set uses",
		},
		{
			name:    "exit_code is allowed only on a run step (agent)",
			mustSay: "exit_code is allowed only on a run step",
			contract: `{"run_id":"r","requires":[{"as":"n","capability":"agent.reason"}],
			  "steps":[{"id":"a","uses":"n","agent":{},"in":{"prompt":"p"},"out":["x"]}],
			  "success_when":[{"step":"a","exit_code":0}]}`,
			wantErr: "exit_code",
		},
		{
			name:    "success_when may only point at steps that exist",
			mustSay: "success_when may only refer to steps that exist",
			contract: `{"run_id":"r","requires":[{"as":"n","capability":"agent.reason"}],
			  "steps":[` + okStep + `],
			  "success_when":[{"step":"nostep","exit_code":0}]}`,
			wantErr: "unknown step",
		},
		{
			name:    "a replan must have exactly one out",
			mustSay: "out (exactly one)",
			contract: `{"run_id":"r","requires":[{"as":"n","capability":"agent.reason"}],
			  "steps":[{"id":"p","uses":"n","expands":true,"agent":{},"in":{"prompt":"x"},
			            "out":["a","b"],"schema":{"a":{"type":"object"}}}],
			  "success_when":[{"step":"p","produced":["a"]}]}`,
			wantErr: "exactly one output",
		},
		{
			name:    "a replan artifact requires a schema",
			mustSay: "schema (required)",
			contract: `{"run_id":"r","requires":[{"as":"n","capability":"agent.reason"}],
			  "steps":[{"id":"p","uses":"n","expands":true,"agent":{},"in":{"prompt":"x"},
			            "out":["plan"]}],
			  "success_when":[{"step":"p","produced":["plan"]}]}`,
			wantErr: "has no schema",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// ① 문법이 그 말을 실제로 하고 있는가
			if !strings.Contains(Grammar, c.mustSay) {
				t.Fatalf("the grammar lacks %q — either Grammar or this test is stale", c.mustSay)
			}
			// ② 그 말대로 실제로 거절되는가
			var ct Contract
			if err := json.Unmarshal([]byte(c.contract), &ct); err != nil {
				t.Fatalf("the test contract is not JSON: %v", err)
			}
			err := ct.Validate()
			if err == nil {
				t.Fatalf("the grammar says it rejects this, yet it passed")
			}
			if !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("rejected for a different reason: %v (want: %q)", err, c.wantErr)
			}
		})
	}
}

// 스키마 어휘 목록이 갈라지지 않는지 — 문법이 나열한 이름과
// schema 패키지가 허용/거절하는 이름이 같아야 한다.
func TestGrammar_SchemaVocabularyMatchesReality(t *testing.T) {
	for _, name := range []string{
		"type", "required", "properties", "enum", "items",
		"additionalProperties", "title", "description",
	} {
		if !strings.Contains(Grammar, name) {
			t.Errorf("allowed keyword %q is missing from the grammar", name)
		}
	}
	for _, name := range []string{
		"minimum", "maximum", "minLength", "maxLength",
		"pattern", "format", "minItems", "maxItems",
	} {
		if !strings.Contains(Grammar, name) {
			t.Errorf("rejected keyword %q is missing from the grammar — a plan would use it and get a 422", name)
		}
	}
}

// 빈 계획이 값이라는 것을 문법이 말해야 한다 (ADR-043)
// 이 문장이 없으면 계획은 고칠 것이 없을 때도 억지로 단계를 지어낸다 — 실측에서 밟았다.
func TestGrammar_MentionsTheEmptyPlan(t *testing.T) {
	for _, want := range []string{`"steps": []`, "a judgment, not an error"} {
		if !strings.Contains(Grammar, want) {
			t.Errorf("the grammar lacks %q", want)
		}
	}
}
