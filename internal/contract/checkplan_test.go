package contract

import (
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
			want: "없는 역할",
		},
		{
			// ★ 11차 ★ — schema 를 산출물 이름으로 안 키잉했다
			name: "schema 를 산출물 이름으로 키잉하지 않는다",
			plan: `{"steps":[{"id":"r","uses":"planner","expands":true,"agent":{},
			         "in":{"prompt":"p"},"out":["plan2"],
			         "schema":{"type":"object","required":["steps"]}}]}`,
			want: "스키마",
		},
		{
			// ★ 6차 ★ — ask 단계에 uses 를 적었다
			name: "ask 에 uses 를 적는다",
			plan: `{"steps":[{"id":"q","uses":"mac","ask":{"prompt":"?"},"out":["a"],
			         "schema":{"a":{"type":"object","properties":{"v":{"type":"string"}}}}}]}`,
			want: "ask 단계에는 uses 가 없다",
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
