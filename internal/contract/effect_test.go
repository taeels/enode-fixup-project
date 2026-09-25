package contract

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// decodeContract 는 시험의 계약을 JSON 으로 적게 한다 — Go 리터럴의 map 은
// 스키마 하나에도 줄이 불어나 무엇을 시험하는지가 묻힌다.
func decodeContract(t *testing.T, s string) Contract {
	t.Helper()
	var c Contract
	if err := json.Unmarshal([]byte(s), &c); err != nil {
		t.Fatalf("the test contract is not JSON: %v", err)
	}
	return c
}

// wantValidate 는 want 가 비면 통과를, 아니면 그 구절이 든 거절을 기대한다.
func wantValidate(t *testing.T, c Contract, want string) {
	t.Helper()
	err := c.Validate()
	if want == "" {
		if err != nil {
			t.Fatalf("Validate = %v, want nil", err)
		}
		return
	}
	if err == nil {
		t.Fatalf("Validate = nil, want an error containing %q", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("Validate = %v, want it to contain %q", err, want)
	}
}

// oneStep 은 단계 하나짜리 계약이다. 단계는 JSON 조각으로 받는다.
func oneStep(t *testing.T, step string) Contract {
	return decodeContract(t, `{"run_id":"r","requires":[{"as":"n","capability":"agent.reason"}],
	  "steps":[`+step+`]}`)
}

// effect 는 종류마다 받는 값이 다르다 (ADR-075 §5).
func TestValidate_Effect(t *testing.T) {
	cases := []struct {
		name, step, want string
	}{
		{"run takes build", `{"id":"a","uses":"n","run":["true"],"effect":"build"}`, ""},
		{"run takes edit", `{"id":"a","uses":"n","run":["true"],"effect":"edit"}`, ""},
		{"run takes read", `{"id":"a","uses":"n","run":["true"],"effect":"read"}`, ""},
		{"agent takes edit", `{"id":"a","uses":"n","agent":{},"effect":"edit"}`, ""},
		{"agent takes read", `{"id":"a","uses":"n","agent":{},"effect":"read"}`, ""},
		{"an unknown value", `{"id":"a","uses":"n","run":["true"],"effect":"write"}`,
			`unknown effect "write"; use read, edit, build or prepare`},
		{"agent does not build", `{"id":"a","uses":"n","agent":{},"effect":"build"}`,
			"effect build is not allowed on an agent step; use read or edit"},
		{"prepare is for build steps (run)", `{"id":"a","uses":"n","run":["true"],"effect":"prepare"}`,
			"effect prepare is only for a build step (sync and builds)"},
		{"prepare is for build steps (agent)", `{"id":"a","uses":"n","agent":{},"effect":"prepare"}`,
			"effect prepare is only for a build step"},
		{"ask takes none", `{"id":"q","ask":{"prompt":"?"},"effect":"read","out":["a"],
		   "schema":{"a":{"type":"object"}}}`, `step "q": an ask step takes no effect`},
		{"acquire takes none", `{"id":"g","effect":"read","acquire":{"want":{"as":"m","capability":"agent.reason"},
		   "acquired":"x","unavailable":"y"}}`, "an acquire step takes no effect"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wantValidate(t, oneStep(t, c.step), c.want)
		})
	}
}

// budget 은 run · agent · build 만 받고, finalize 는 늘리기만 한다 (FR-3).
func TestValidate_Budget(t *testing.T) {
	run := func(budget string) string {
		return `{"id":"a","uses":"n","run":["true"],"budget":` + budget + `}`
	}
	cases := []struct {
		name, step, want string
	}{
		{"empty object is both defaults", run(`{}`), ""},
		{"finalize exactly the default", run(`{"finalize":"1m"}`), ""},
		{"finalize raised", run(`{"finalize":"90s"}`), ""},
		{"upload one nanosecond", run(`{"upload":"1ns"}`), ""},
		{"no upper bound", run(`{"finalize":"48h","upload":"48h"}`), ""},
		{"agent takes a budget", `{"id":"a","uses":"n","agent":{},"budget":{"upload":"10m"}}`, ""},
		{"finalize below the default", run(`{"finalize":"59s"}`),
			`budget.finalize 59s is below the default 1m0s; it can only be raised`},
		{"finalize negative", run(`{"finalize":"-1m"}`), "is below the default 1m0s"},
		{"finalize not a duration", run(`{"finalize":"soon"}`),
			`budget.finalize "soon" is not a duration (for example "90s" or "5m")`},
		{"upload zero", run(`{"upload":"0s"}`), "budget.upload must be greater than zero"},
		{"upload negative", run(`{"upload":"-5m"}`), "budget.upload must be greater than zero"},
		{"upload not a duration", run(`{"upload":"3"}`), `budget.upload "3" is not a duration`},
		{"ask takes none", `{"id":"q","ask":{"prompt":"?"},"budget":{},"out":["a"],
		   "schema":{"a":{"type":"object"}}}`, "an ask step takes no budget"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wantValidate(t, oneStep(t, c.step), c.want)
		})
	}
}

// discover 는 명시로 켜는 훑기다 (ADR-075 §8). run · agent 만 받는다.
func TestValidate_Discover(t *testing.T) {
	cases := []struct {
		name, step, want string
	}{
		{"run", `{"id":"a","uses":"n","run":["true"],"discover":true}`, ""},
		{"agent without a workspace", `{"id":"a","uses":"n","agent":{},"discover":true}`, ""},
		{"ask", `{"id":"q","ask":{"prompt":"?"},"discover":true,"out":["a"],
		   "schema":{"a":{"type":"object"}}}`, `step "q": discover is only for a run or agent step`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wantValidate(t, oneStep(t, c.step), c.want)
		})
	}
}

func TestValidate_IROnlyOnBuild(t *testing.T) {
	wantValidate(t, oneStep(t, `{"id":"a","uses":"n","run":["true"],"ir":"v1"}`),
		`step "a": ir is only for a build step`)
}

func TestEffectOrDefault(t *testing.T) {
	cases := []struct {
		name string
		s    Step
		want Effect
	}{
		{"run defaults to build", Step{Run: []string{"true"}}, EffectBuild},
		{"agent defaults to edit", Step{Agent: map[string]interface{}{}}, EffectEdit},
		{"written value wins", Step{Run: []string{"gofmt"}, Effect: EffectEdit}, EffectEdit},
		{"build step carries prepare", Step{Sync: "s", Effect: EffectPrepare}, EffectPrepare},
		{"merge has none", Step{Merge: &Merge{}}, ""},
		{"ask has none", Step{Ask: &Ask{}}, ""},
		{"acquire has none", Step{Acquire: &Acquire{}}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.s.EffectOrDefault(); got != c.want {
				t.Fatalf("EffectOrDefault = %q, want %q", got, c.want)
			}
		})
	}
}

func TestBudgets_FillDefaults(t *testing.T) {
	cases := []struct {
		name             string
		b                *Budget
		finalize, upload time.Duration
	}{
		{"no budget", nil, time.Minute, 3 * time.Minute},
		{"empty budget", &Budget{}, time.Minute, 3 * time.Minute},
		{"finalize only", &Budget{Finalize: "5m"}, 5 * time.Minute, 3 * time.Minute},
		{"upload only", &Budget{Upload: "10m"}, time.Minute, 10 * time.Minute},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, u := Step{Budget: c.b}.Budgets()
			if f != c.finalize || u != c.upload {
				t.Fatalf("Budgets = %v, %v, want %v, %v", f, u, c.finalize, c.upload)
			}
		})
	}
}

// 문구가 틀린 영어를 내지 않는다 — 거절 문구는 밖으로 나간다 (CONVENTIONS 2.1).
func TestAKind(t *testing.T) {
	for k, want := range map[StepKind]string{
		KindAgent: "an agent", KindAsk: "an ask", KindAcquire: "an acquire",
		KindRun: "a run", KindBuild: "a build", KindMerge: "a merge",
	} {
		if got := aKind(k); got != want {
			t.Errorf("aKind(%v) = %q, want %q", k, got, want)
		}
	}
}
