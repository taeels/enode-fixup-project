package contract

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

// 굽기 계약의 시험 재료. 시험마다 한 곳만 바꿔 틀리게 만든다.
const (
	bakeRequires = `"requires":[{"as":"baker","capability":"agent.reason","workspace.writes":"isolated"},
	                            {"as":"planner","capability":"agent.reason"}]`
	bakeBuild = `{"id":"build","uses":"baker","effect":"prepare","ir":"your-ir-tag",
	              "sync":"<sync>","builds":[{"name":"config-a","command":"<build a>"},
	                                        {"name":"config-b","command":"<build b>"}]}`
	bakeMerge = `{"id":"merge","uses":"baker","needs":["build"],"merge":{"wait":"4h"}}`
	bakeWhen  = `"success_when":[{"step":"build","produced":["manifest"]},
	                             {"step":"merge","produced":["merged"]}]`

	// 계획 단계 둘 — 계획을 짓는 agent 와 그것을 승인하는 ask.
	planStep = `{"id":"plan","uses":"planner","expands":true,"agent":{},"in":{"prompt":"p"},
	             "out":["plan"],"schema":{"plan":{"type":"object"}},"produces":["build","merge"]}`
	approveStep = `{"id":"approve","ask":{"prompt":"bake this?","adopts":"plan","adopt_when":"approve"},
	                "out":["approval"],"schema":{"approval":{"type":"object","required":["verdict"],
	                  "properties":{"verdict":{"enum":["approve","reject"]}}}}}`
)

func bakeOf(t *testing.T, steps ...string) Contract {
	return decodeContract(t, `{"run_id":"r",`+bakeRequires+`,"steps":[`+
		strings.Join(steps, ",")+`],`+bakeWhen+`}`)
}

// 사람이 처음부터 쓴 굽기와, 계획이 지어 붙인 뒤의 굽기는 받는다.
func TestValidate_BakeAccepted(t *testing.T) {
	t.Run("written by a person", func(t *testing.T) {
		wantValidate(t, bakeOf(t, bakeBuild, bakeMerge), "")
	})
	t.Run("merge {} with default needs", func(t *testing.T) {
		wantValidate(t, bakeOf(t, bakeBuild, `{"id":"merge","uses":"baker","merge":{}}`), "")
	})
	t.Run("build and merge after a plan and its approval", func(t *testing.T) {
		wantValidate(t, bakeOf(t, planStep, approveStep, bakeBuild, bakeMerge), "")
	})
	t.Run("the plan's build names the approval in needs", func(t *testing.T) {
		b := strings.Replace(bakeBuild, `"uses":"baker",`, `"uses":"baker","needs":["approve"],`, 1)
		wantValidate(t, bakeOf(t, planStep, approveStep, b, bakeMerge), "")
	})
	t.Run("env names on the build step", func(t *testing.T) {
		b := strings.Replace(bakeBuild, `"uses":"baker",`, `"uses":"baker","env":["DL_DIR"],`, 1)
		wantValidate(t, bakeOf(t, b, bakeMerge), "")
	})
}

// build 단계가 받는 칸과 값 (ADR-077 §2).
func TestValidate_BuildStep(t *testing.T) {
	name64 := strings.Repeat("a", 64)
	cases := []struct {
		name, build, want string
	}{
		{"no uses", `{"id":"build","effect":"prepare","ir":"v","sync":"s","builds":[{"name":"a","command":"c"}]}`,
			`step "build": a build step needs uses; it runs on a node`},
		{"no effect", `{"id":"build","uses":"baker","ir":"v","sync":"s","builds":[{"name":"a","command":"c"}]}`,
			`a build step must say effect "prepare"`},
		{"effect edit", `{"id":"build","uses":"baker","effect":"edit","ir":"v","sync":"s",
		   "builds":[{"name":"a","command":"c"}]}`, `a build step must say effect "prepare"`},
		{"builds without sync", `{"id":"build","uses":"baker","effect":"prepare","ir":"v",
		   "builds":[{"name":"a","command":"c"}]}`, "a build step needs sync"},
		{"blank sync", `{"id":"build","uses":"baker","effect":"prepare","ir":"v","sync":"  ",
		   "builds":[{"name":"a","command":"c"}]}`, "a build step needs sync"},
		{"sync without builds", `{"id":"build","uses":"baker","effect":"prepare","ir":"v","sync":"s"}`,
			"a build step needs at least one entry in builds"},
		{"empty builds", `{"id":"build","uses":"baker","effect":"prepare","ir":"v","sync":"s","builds":[]}`,
			"a build step needs at least one entry in builds"},
		{"upper-case name", `{"id":"build","uses":"baker","effect":"prepare","ir":"v","sync":"s",
		   "builds":[{"name":"Config","command":"c"}]}`,
			`builds[0].name "Config" must be 1 to 64 characters of a-z, 0-9 and -`},
		{"empty name", `{"id":"build","uses":"baker","effect":"prepare","ir":"v","sync":"s",
		   "builds":[{"name":"","command":"c"}]}`, `builds[0].name "" must be 1 to 64 characters`},
		{"name of 65", `{"id":"build","uses":"baker","effect":"prepare","ir":"v","sync":"s",
		   "builds":[{"name":"` + name64 + `b","command":"c"}]}`, "must be 1 to 64 characters"},
		{"name twice", `{"id":"build","uses":"baker","effect":"prepare","ir":"v","sync":"s",
		   "builds":[{"name":"a","command":"c"},{"name":"a","command":"d"}]}`, `builds name "a" appears twice`},
		{"blank command", `{"id":"build","uses":"baker","effect":"prepare","ir":"v","sync":"s",
		   "builds":[{"name":"a","command":"c"},{"name":"b","command":" "}]}`, "builds[1].command is empty"},
		{"no ir", `{"id":"build","uses":"baker","effect":"prepare","sync":"s","builds":[{"name":"a","command":"c"}]}`,
			"a build step needs ir, the exact tag to bake"},
		{"workspace", `{"id":"build","uses":"baker","effect":"prepare","ir":"v","sync":"s",
		   "builds":[{"name":"a","command":"c"}],"workspace":{"repo":"x/y"}}`,
			"a build step does not take workspace; sync prepares the source"},
		{"out", `{"id":"build","uses":"baker","effect":"prepare","ir":"v","sync":"s",
		   "builds":[{"name":"a","command":"c"}],"out":["manifest"]}`,
			`a build step does not take out; it produces "manifest" when the build succeeds`},
		{"collect", `{"id":"build","uses":"baker","effect":"prepare","ir":"v","sync":"s",
		   "builds":[{"name":"a","command":"c"}],"collect":{"x":"y"}}`, "a build step does not take collect"},
		{"loop", `{"id":"build","uses":"baker","effect":"prepare","ir":"v","sync":"s",
		   "builds":[{"name":"a","command":"c"}],"loop":{"back_to":"x","max":2}}`, "a build step does not take loop"},
		{"discover", `{"id":"build","uses":"baker","effect":"prepare","ir":"v","sync":"s",
		   "builds":[{"name":"a","command":"c"}],"discover":true}`, "discover is only for a run or agent step"},
		{"budget below the floor", `{"id":"build","uses":"baker","effect":"prepare","ir":"v","sync":"s",
		   "builds":[{"name":"a","command":"c"}],"budget":{"finalize":"10s"}}`, "is below the default 1m0s"},
		{"name of 64 and a budget", `{"id":"build","uses":"baker","effect":"prepare","ir":"v","sync":"s",
		   "builds":[{"name":"` + name64 + `","command":"c"}],"budget":{"finalize":"5m","upload":"1h"}}`, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wantValidate(t, bakeOf(t, c.build, bakeMerge), c.want)
		})
	}
}

// ir 은 git 태그 이름 가운데 좁은 집합이다. 날짜 모양 같은 형식 규칙은 없다.
func TestValidate_IRCharacters(t *testing.T) {
	ok := []string{"your-ir-tag", "yocto-5.0.1", "release/v1.2", "A_b-1.2/x", strings.Repeat("x", 128)}
	bad := map[string]string{
		strings.Repeat("x", 129): "longer than 128 characters",
		"a b":                    `' ' is not allowed`,
		"a$b":                    `'$' is not allowed`,
		"-x":                     "must not start with '-', '/' or '.'",
		"/x":                     "must not start with",
		".x":                     "must not start with",
		"x/":                     "must not end with '/' or '.'",
		"x.":                     "must not end with",
		"a..b":                   "must not contain '..' or '//'",
		"a//b":                   "must not contain",
		"a/.b":                   "no part between slashes may start with '.'",
		"a.lock":                 "no part between slashes may end with '.lock'",
		"a/b.lock/c":             "may end with '.lock'",
	}
	withIR := func(ir string) Contract {
		b := strings.Replace(bakeBuild, `"your-ir-tag"`, jsonString(t, ir), 1)
		return bakeOf(t, b, bakeMerge)
	}
	for _, ir := range ok {
		t.Run("accepts "+ir[:min(len(ir), 20)], func(t *testing.T) {
			wantValidate(t, withIR(ir), "")
		})
	}
	for ir, why := range bad {
		t.Run("rejects "+ir[:min(len(ir), 20)], func(t *testing.T) {
			wantValidate(t, withIR(ir), "is not a valid tag name: ")
			wantValidate(t, withIR(ir), why)
		})
	}
}

func jsonString(t *testing.T, s string) string {
	t.Helper()
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// merge 단계가 받는 칸과 값.
func TestValidate_MergeStep(t *testing.T) {
	cases := []struct {
		name, merge, want string
	}{
		{"no uses", `{"id":"merge","needs":["build"],"merge":{}}`,
			`step "merge": a merge step needs uses; it runs on the node that built`},
		{"wait not a duration", `{"id":"merge","uses":"baker","merge":{"wait":"soon"}}`,
			`merge.wait "soon" is not a duration (for example "4h")`},
		{"wait zero", `{"id":"merge","uses":"baker","merge":{"wait":"0s"}}`,
			"merge.wait must be greater than zero"},
		{"wait negative", `{"id":"merge","uses":"baker","merge":{"wait":"-1h"}}`,
			"merge.wait must be greater than zero"},
		{"env", `{"id":"merge","uses":"baker","merge":{},"env":["X"]}`, "a merge step does not take env"},
		{"out", `{"id":"merge","uses":"baker","merge":{},"out":["merged"]}`, "a merge step does not take out"},
		{"effect", `{"id":"merge","uses":"baker","merge":{},"effect":"read"}`, "a merge step takes no effect"},
		{"budget", `{"id":"merge","uses":"baker","merge":{},"budget":{}}`,
			"a merge step takes no budget; merge.wait sets how long it waits"},
		{"wait raised", `{"id":"merge","uses":"baker","merge":{"wait":"12h"}}`, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wantValidate(t, bakeOf(t, bakeBuild, c.merge), c.want)
		})
	}
}

// 굽기 계약은 build 하나와 merge 하나로 끝난다. 그 앞에는 계획 단계만 온다.
func TestValidate_BakeShape(t *testing.T) {
	other := `{"id":"other","uses":"baker","run":["true"]}`
	build2 := strings.Replace(bakeBuild, `"id":"build"`, `"id":"build2"`, 1)
	merge2 := `{"id":"merge2","uses":"baker","needs":["build"],"merge":{}}`
	cases := []struct {
		name  string
		steps []string
		want  string
	}{
		{"two builds", []string{bakeBuild, build2, bakeMerge},
			"the contract has 2 build steps; a bake has exactly one"},
		{"two merges", []string{bakeBuild, bakeMerge, merge2},
			"the contract has 2 merge steps; a bake has exactly one"},
		{"build alone", []string{bakeBuild},
			`build step "build" has no merge step; the built upper would wait forever`},
		{"merge alone", []string{`{"id":"merge","uses":"baker","merge":{}}`},
			`merge step "merge" has no build step to merge`},
		{"a step after merge", []string{bakeBuild, bakeMerge,
			`{"id":"other","uses":"baker","needs":["merge"],"run":["true"]}`},
			`step "other" comes after merge step "merge"; a bake ends with its build and merge`},
		{"merge before build", []string{`{"id":"merge","uses":"baker","merge":{}}`, bakeBuild},
			`step "build" comes after merge step "merge"`},
		{"a step between", []string{bakeBuild, `{"id":"other","uses":"baker","needs":["build"],"run":["true"]}`,
			bakeMerge},
			`step "other" sits between build "build" and merge "merge"; merge follows build directly`},
		{"merge waits for nothing", []string{bakeBuild, `{"id":"merge","uses":"baker","needs":[],"merge":{}}`},
			`merge step "merge" must need exactly ["build"]`},
		{"merge on another role", []string{bakeBuild, `{"id":"merge","uses":"planner","merge":{}}`},
			`merge step "merge" must use the same role as build step "build" ("baker")`},
		{"a run step before the bake", []string{other,
			strings.Replace(bakeBuild, `"uses":"baker",`, `"uses":"baker","needs":["other"],`, 1), bakeMerge},
			`step "other" comes before the bake but is not a planning step`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wantValidate(t, bakeOf(t, c.steps...), c.want)
		})
	}
}

// 계획이 지은 굽기는 사람이 승인한 뒤에만 돈다 — 넷이 모두 있어야 한다.
func TestValidate_PlannedBake(t *testing.T) {
	yolo := strings.Replace(planStep, `"expands":true,`, `"expands":true,"adopt":"yolo",`, 1)
	noAdoptWhen := strings.Replace(approveStep, `,"adopt_when":"approve"`, "", 1)
	skipsApproval := strings.Replace(bakeBuild, `"uses":"baker",`, `"uses":"baker","needs":["plan"],`, 1)
	cases := []struct {
		name  string
		steps []string
		want  string
	}{
		{"adopted with yolo", []string{yolo, bakeBuild, bakeMerge},
			`step "plan" adopts its plan without asking (adopt "yolo"), but the contract bakes`},
		{"no ask adopts the plan", []string{planStep, bakeBuild, bakeMerge},
			`step "plan" builds a plan in a contract that bakes, but no ask step adopts it; ` +
				`a person must adopt the plan before the bake runs (an ask step with adopts: "plan")`},
		{"the ask ignores rejection", []string{planStep, noAdoptWhen, bakeBuild, bakeMerge},
			`ask step "approve" adopts plan "plan" in a contract that bakes, but says nothing about rejection`},
		{"build skips the approval", []string{planStep, approveStep, skipsApproval, bakeMerge},
			`build step "build" does not wait for ask step "approve"; a bake runs only after a person adopts the plan`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wantValidate(t, bakeOf(t, c.steps...), c.want)
		})
	}
}

// 제출 때에는 굽기가 아직 없다 — 계획이 약속한 이름에 걸린 조건은 종류 검사를
// 건너뛴다 (ADR-049). 굽기 규칙은 계획이 붙은 뒤의 Validate 가 건다.
func TestValidate_PlannedBakeAtSubmission(t *testing.T) {
	wantValidate(t, bakeOf(t, planStep, approveStep), "")
}

// 굽기 두 단계는 고정 산출물 하나로만 판정한다.
func TestValidate_BakeConditions(t *testing.T) {
	cases := []struct {
		name, when, want string
	}{
		{"exit_code on build", `{"step":"build","exit_code":0}`,
			`exit_code condition is allowed only on a run step: "build"`},
		{"another name on build", `{"step":"build","produced":["log"]}`,
			`success_when: step "build" is a build step; judge it by produced ["manifest"]`},
		{"two names on build", `{"step":"build","produced":["manifest","log"]}`,
			`judge it by produced ["manifest"]`},
		{"within_attempts on build", `{"step":"build","produced":["manifest"],"within_attempts":true}`,
			`judge it by produced ["manifest"]`},
		{"step only", `{"step":"build"}`, `judge it by produced ["manifest"]`},
		{"manifest on merge", `{"step":"merge","produced":["manifest"]}`,
			`success_when: step "merge" is a merge step; judge it by produced ["merged"]`},
		{"changed on merge", `{"step":"merge","produced":["merged"],"changed":["x"]}`,
			`judge it by produced ["merged"]`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ct := decodeContract(t, `{"run_id":"r",`+bakeRequires+`,"steps":[`+bakeBuild+`,`+bakeMerge+`],
			  "success_when":[`+c.when+`]}`)
			wantValidate(t, ct, c.want)
		})
	}
}

// exit_code 의 거절은 이름(ErrExitOnAgent)을 그대로 둔다 — errors.Is 로 부르는 쪽이 있다.
func TestValidate_ExitCodeOnAskSaysRunStep(t *testing.T) {
	c := oneStep(t, `{"id":"q","ask":{"prompt":"?"},"out":["a"],"schema":{"a":{"type":"object"}}}`)
	c.SuccessWhen = []Condition{{Step: "q", ExitCode: zero()}}
	wantValidate(t, c, `exit_code condition is allowed only on a run step: "q"`)
}

// otherFields 가 Step 의 칸을 빠짐없이 덮는가.
//
// 칸이 늘었는데 여기가 안 늘면 그 칸이 build · merge 에서 조용히 무시된다.
// 칸을 하나씩 채워 보고, 판별 칸이나 종류마다 따로 보는 칸이 아닌데
// otherFields 가 이름을 안 대면 깨진다.
func TestOtherFields_CoverEveryStepField(t *testing.T) {
	handledElsewhere := map[string]bool{
		"id": true, "uses": true, "needs": true, // build · merge 가 받는다
		"agent": true, "run": true, "ask": true, "acquire": true, // 판별 칸 — Kind 가 본다
		"sync": true, "builds": true, "merge": true, // 판별 칸 — Kind 가 본다
		"effect": true, "budget": true, "discover": true, "ir": true, // checkStepFields 가 먼저 본다
	}
	typ := reflect.TypeOf(Step{})
	for i := 0; i < typ.NumField(); i++ {
		name, _, _ := strings.Cut(typ.Field(i).Tag.Get("json"), ",")
		if handledElsewhere[name] {
			continue
		}
		var s Step
		setNonZero(t, reflect.ValueOf(&s).Elem().Field(i))
		if got := otherFields(s); len(got) != 1 || got[0] != name {
			t.Errorf("field %q is set but otherFields = %v; add it to otherFields "+
				"or to the handled list of this test", name, got)
		}
	}
}

func setNonZero(t *testing.T, v reflect.Value) {
	t.Helper()
	switch v.Kind() {
	case reflect.Pointer:
		v.Set(reflect.New(v.Type().Elem()))
	case reflect.Slice:
		v.Set(reflect.MakeSlice(v.Type(), 1, 1))
	case reflect.Map:
		m := reflect.MakeMap(v.Type())
		m.SetMapIndex(reflect.New(v.Type().Key()).Elem(), reflect.New(v.Type().Elem()).Elem())
		v.Set(m)
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int:
		v.SetInt(1)
	case reflect.String:
		v.SetString("x")
	default:
		t.Fatalf("setNonZero does not know %v; teach it", v.Kind())
	}
}

// 안 적은 계약은 오늘과 같은 JSON 이다 — 새 칸 일곱이 다시 묶을 때 안 나온다.
func TestNewFields_AbsentFromExistingExamples(t *testing.T) {
	added := []string{"effect", "budget", "discover", "sync", "builds", "ir", "merge"}
	for _, name := range ExampleNames() {
		if name == "bake" {
			continue // 새 칸을 쓰는 예시다
		}
		t.Run(name, func(t *testing.T) {
			raw, _ := Example(name)
			c := decodeContract(t, string(raw))
			for _, st := range c.Steps {
				b, err := json.Marshal(st)
				if err != nil {
					t.Fatal(err)
				}
				var keys map[string]interface{}
				if err := json.Unmarshal(b, &keys); err != nil {
					t.Fatal(err)
				}
				for _, k := range added {
					if _, ok := keys[k]; ok {
						t.Errorf("step %q marshals %q although the example does not write it", st.ID, k)
					}
				}
			}
		})
	}
}

func TestMergeWait_FillsDefault(t *testing.T) {
	cases := []struct {
		name string
		m    *Merge
		want time.Duration
	}{
		{"no merge", nil, 4 * time.Hour},
		{"empty merge", &Merge{}, 4 * time.Hour},
		{"written", &Merge{Wait: "6h"}, 6 * time.Hour},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := (Step{Merge: c.m}).MergeWait(); got != c.want {
				t.Fatalf("MergeWait = %v, want %v", got, c.want)
			}
		})
	}
}
