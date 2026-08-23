package contract

import (
	"encoding/json"
	"strings"
	"testing"
)

// ★ 실측에서 실제로 밟은 계획들이 잡히는가 ★ (ADR-046)
//
// 넷 다 계획이 다 지어진 뒤에야 거절돼서 판이 통째로 버려졌다.
// 훅이 이것을 ★ 하네스가 끝나기 전에 ★ 짚으면 같은 세션에서 고친다.
func TestCheckPlan_실측에서_밟은_것들(t *testing.T) {
	roles := []string{"planner", "mac"}
	cases := []struct {
		name, plan, want string
	}{
		{
			// ★ 10차 ★ — replan_1 이 uses:"claude" 를 지어냈다
			name: "없는 역할을 쓴다",
			plan: `{"steps":[{"id":"r","uses":"claude","expands":true,"agent":{},
			         "in":{"prompt":"p"},"out":["plan2"],
			         "schema":{"plan2":{"type":"object"}}}]}`,
			want: "undeclared role",
		},
		{
			// ★ 11차 ★ — schema 를 산출물 이름으로 안 키잉했다
			name: "schema 를 산출물 이름으로 키잉하지 않는다",
			plan: `{"steps":[{"id":"r","uses":"planner","expands":true,"agent":{},
			         "in":{"prompt":"p"},"out":["plan2"],
			         "schema":{"type":"object","required":["steps"]}}]}`,
			want: "schema declared for",
		},
		{
			// ★ 6차 ★ — ask 단계에 uses 를 적었다
			name: "ask 에 uses 를 적는다",
			plan: `{"steps":[{"id":"q","uses":"mac","ask":{"prompt":"?"},"out":["a"],
			         "schema":{"a":{"type":"object","properties":{"v":{"type":"string"}}}}}]}`,
			want: "must not set uses",
		},
		{
			// ★ 8차 ★ — agent 단계에 exit_code 를 걸었다
			name: "에이전트 단계에 exit_code 를 건다",
			plan: `{"steps":[{"id":"t","uses":"mac","agent":{},"in":{"prompt":"p"},"out":["x"]}],
			        "success_when":[{"step":"t","exit_code":0}]}`,
			want: "exit_code",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := CheckPlan([]byte(c.plan), roles)
			if err == nil {
				t.Fatal("★ 잡혔어야 한다 ★")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("다른 이유로 잡혔다: %v (기대: %q)", err, c.want)
			}
		})
	}
}

// ★ 정당한 계획을 막으면 안 된다 ★ — 안전망이 정규 경로를 무너뜨리는 것이 가장 나쁘다.
func TestCheckPlan_정당한_계획은_통과한다(t *testing.T) {
	roles := []string{"planner", "mac"}
	// ★ 계획 밖(부모)의 승인 단계를 needs 로 잡는다 ★ — 실제 계획의 모양이다.
	plan := `{"steps":[
	   {"id":"work","uses":"mac","needs":["approve_plan"],"run":["true"],"out":["log"]},
	   {"id":"replan_1","uses":"planner","needs":["work"],"expands":true,"agent":{},
	    "in":{"prompt":"p"},"out":["plan2"],"schema":{"plan2":{"type":"object"}}},
	   {"id":"approve_replan_1","needs":["replan_1"],
	    "ask":{"prompt":"?","adopts":"replan_1"},"out":["approval2"],
	    "schema":{"approval2":{"type":"object","required":["verdict"],
	      "properties":{"verdict":{"enum":["approve","reject"]}}}}}],
	  "success_when":[{"step":"work","exit_code":0,"produced":["log"]},
	                  {"step":"replan_1","produced":["plan2"]},
	                  {"step":"approve_replan_1","produced":["approval2"]}]}`
	if err := CheckPlan([]byte(plan), roles); err != nil {
		t.Fatalf("★ 정당한 계획을 막았다 ★: %v", err)
	}
}

// ★ 빈 계획은 값이다 ★ (ADR-043) — 훅도 그것을 알아야 한다.
func TestCheckPlan_빈_계획은_통과한다(t *testing.T) {
	if err := CheckPlan([]byte(`{"steps":[],"success_when":[]}`), []string{"a"}); err != nil {
		t.Fatalf("★ 빈 계획을 막았다 ★: %v", err)
	}
	if err := CheckPlan([]byte(`{"steps":[],"success_when":[{"step":"x"}]}`), []string{"a"}); err == nil {
		t.Fatal("★ 단계가 없는데 판정이 있는 것은 자기모순이다 ★")
	}
}

// ★ 역할을 모르면 막지 않는다 ★ — 모르는 것으로 막는 것이 가장 나쁜 안전망이다.
func TestCheckPlan_역할을_모르면_통과시킨다(t *testing.T) {
	plan := `{"steps":[{"id":"a","uses":"뭐든","run":["true"],"out":["l"]}]}`
	if err := CheckPlan([]byte(plan), nil); err != nil {
		t.Fatalf("역할을 모르는데 막았다: %v", err)
	}
}

// ★ 계약이 약속한 단계를 계획이 안 지으면 거절된다 ★ (ADR-049)
func TestValidate_produces(t *testing.T) {
	// ① success_when 이 ★ 아직 없는 단계 ★ 를 가리켜도 통과한다 — 약속했으므로.
	ok := `{"run_id":"r","requires":[{"as":"n","capability":"agent.reason"}],
	  "steps":[{"id":"p","uses":"n","expands":true,"agent":{},"in":{"prompt":"x"},
	            "out":["plan"],"schema":{"plan":{"type":"object"}},
	            "produces":["goal_step"]}],
	  "success_when":[{"step":"goal_step","produced":["done"]}]}`
	var c Contract
	if err := json.Unmarshal([]byte(ok), &c); err != nil {
		t.Fatal(err)
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("★ 약속한 이름을 미리 못 가리켰다 ★: %v", err)
	}

	// ② 약속이 없으면 여전히 거절한다 — ErrCondUnknownID 가 살아 있어야 한다.
	bad := strings.Replace(ok, `"produces":["goal_step"]`, `"produces":[]`, 1)
	c = Contract{}
	if err := json.Unmarshal([]byte(bad), &c); err != nil {
		t.Fatal(err)
	}
	if err := c.Validate(); err == nil {
		t.Fatal("★ 약속 없이 없는 단계를 가리켰는데 통과했다 ★")
	}

	// ③ produces 는 expands 단계에만 — 계획을 안 짓는 단계는 약속할 것이 없다.
	notExpands := `{"run_id":"r","requires":[{"as":"n","capability":"agent.reason"}],
	  "steps":[{"id":"a","uses":"n","run":["true"],"out":["l"],"produces":["z"]}],
	  "success_when":[{"step":"a","exit_code":0}]}`
	c = Contract{}
	if err := json.Unmarshal([]byte(notExpands), &c); err != nil {
		t.Fatal(err)
	}
	if err := c.Validate(); err == nil ||
		!strings.Contains(err.Error(), "only allowed on an expands step") {
		t.Fatalf("★ 명령 단계의 produces 를 안 막았다 ★: %v", err)
	}

	// ④ ★ 약속이 이행된 뒤의 계약도 유효해야 한다 ★
	//
	// applyExpands 는 「약속한 이름을 지었는가」를 확인하고 ★ 바로 다음 줄에서 ★
	// 확장된 계약을 Validate 한다. 그때 그 이름은 ★ 당연히 존재한다 ★ —
	// 그것이 약속의 이행이다. 예전 규칙("이미 있으면 거절")은 ★ ①이 요구한 것을
	// ②가 금지했다 ★. vm-scratch-2 의 1판이 이것으로 죽었다: 계획은 옳았다.
	fulfilled := `{"run_id":"r","requires":[{"as":"n","capability":"agent.reason"}],
	  "steps":[{"id":"p","uses":"n","expands":true,"agent":{},"in":{"prompt":"x"},
	            "out":["plan"],"schema":{"plan":{"type":"object"}},
	            "produces":["goal_step"]},
	           {"id":"goal_step","uses":"n","needs":["p"],"run":["true"],"out":["done"]}],
	  "success_when":[{"step":"goal_step","produced":["done"]}]}`
	c = Contract{}
	if err := json.Unmarshal([]byte(fulfilled), &c); err != nil {
		t.Fatal(err)
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("★ 이행된 약속을 거절했다 ★ — 계획이 지은 단계가 있는 것이 정상이다: %v", err)
	}

	// ⑤ ★ 자기 자신은 약속할 수 없다 ★ — 계획이 지은 단계는 언제나 뒤에 붙는다.
	self := strings.Replace(ok, `"produces":["goal_step"]`, `"produces":["p"]`, 1)
	self = strings.Replace(self, `{"step":"goal_step","produced":["done"]}`,
		`{"step":"p","produced":["plan"]}`, 1)
	c = Contract{}
	if err := json.Unmarshal([]byte(self), &c); err != nil {
		t.Fatal(err)
	}
	if err := c.Validate(); err == nil ||
		!strings.Contains(err.Error(), "at or before this step") {
		t.Fatalf("★ 자기 자신을 약속했는데 통과했다 ★: %v", err)
	}
}
