package contract

import (
	"fmt"
	"strings"
	"time"
)

// Build 는 굽기 build 단계의 구성 하나다 (ADR-077 §2).
//
// 명령은 제품이 해석하지 않는 문자열이다 — 내용은 계약을 쓰는 쪽의 것이고,
// 노드가 격리 runtime 안에서 돌린다 (bake 유닛).
type Build struct {
	// Name 은 a-z · 0-9 · - 로 1 ~ 64 자이고 계약 안에서 겹치지 않는다.
	// 광고 키 repo.built.<이름> 의 한 조각이 되므로 키가 한없이 길어지지 않게 한다.
	Name    string `json:"name"`
	Command string `json:"command"`
}

// Merge 는 굽기 merge 단계의 표시이자 설정이다 (ADR-077 §2).
//
// 명령이 아니라 노드의 내장 단계다. 그래서 "merge": {} 도 merge 단계다 —
// 판별 칸이 객체 하나이고, 스케치의 kind: merge 처럼 종류 이름을 따로 적지 않는다.
type Merge struct {
	// Wait 는 형제를 기다리는 상한이다. 기본 4시간이고 0 보다 커야 한다.
	Wait string `json:"wait,omitempty"`
}

const (
	// DefaultMergeWait 는 merge 가 형제를 기다리는 기본 상한이다.
	DefaultMergeWait = 4 * time.Hour

	// 굽기 두 단계의 산출물 이름은 제품이 정한다 — 계약은 build · merge 에 out 을
	// 적지 않는다. 판정은 오늘의 produced 대조가 그대로 한다 (Verify 는 안 바뀐다).
	//
	// ArtifactManifest 는 build 단계가 모든 명령이 0 으로 끝나고 IR 이 맞았을 때만 낸다.
	ArtifactManifest = "manifest"
	// ArtifactMerged 는 merge 단계가 합쳤을 때만 낸다.
	ArtifactMerged = "merged"

	// EnvIR 은 노드가 sync 와 builds 명령에 ir 값을 넘기는 환경 변수 이름이다.
	EnvIR = "ENODE_IR"
)

// MergeWait 는 merge 대기 상한을 기본값을 채워 준다.
// Validate 를 지난 계약에서만 부른다 (Budgets 와 같다).
func (s Step) MergeWait() time.Duration {
	if s.Merge != nil {
		if d, err := time.ParseDuration(s.Merge.Wait); err == nil {
			return d
		}
	}
	return DefaultMergeWait
}

// otherFields 는 판별 칸도 아니고 종류마다 따로 보는 칸도 아닌 것 가운데
// 적힌 것의 JSON 이름이다. build 와 merge 는 이 가운데 무엇도 받지 않는다.
//
// 칸이 늘면 여기도 늘어야 한다 — bake_test.go 가 Step 의 JSON 칸을 하나씩
// 채워 보며 빠진 이름을 찾는다. 빠지면 그 칸이 굽기 단계에서 조용히 무시된다.
func otherFields(s Step) []string {
	var out []string
	add := func(name string, set bool) {
		if set {
			out = append(out, name)
		}
	}
	add("workspace", s.Workspace != nil)
	add("env", len(s.Env) > 0)
	add("collect", len(s.Collect) > 0)
	add("in", len(s.In) > 0)
	add("out", len(s.Out) > 0)
	add("schema", len(s.Schema) > 0)
	add("expands", s.Expands)
	add("produces", len(s.Produces) > 0)
	add("adopt", s.Adopt != "")
	add("release", len(s.Release) > 0)
	add("see", s.See != nil)
	add("dispatch", s.Dispatch != nil)
	add("validate_with", s.ValidateWith != "")
	add("max_attempts", s.MaxAttempts != 0)
	add("feedback", len(s.Feedback) > 0)
	add("repeat", s.Repeat != 0)
	add("loop", s.Loop != nil)
	return out
}

// checkBuildStep 은 굽기 build 단계의 칸과 값을 본다.
//
// 받는 칸 — id · uses · needs · effect(prepare) · sync · builds · ir · env · budget.
// 계약이 노드 설정이나 host 경로를 지정하는 칸은 없다 (팩 보안 표) —
// workspace 를 받지 않는 것이 그 자리다. 소스는 sync 가 맞춘다.
func checkBuildStep(s Step) error {
	for _, f := range otherFields(s) {
		switch f {
		case "env":
			continue // 오늘의 뜻 그대로 — 노드 환경에서 통과시킬 이름
		case "workspace":
			return fmt.Errorf("step %q: a build step does not take workspace; sync prepares the source", s.ID)
		case "out":
			return fmt.Errorf("step %q: a build step does not take out; it produces %q when the build succeeds",
				s.ID, ArtifactManifest)
		}
		return fmt.Errorf("step %q: a build step does not take %s", s.ID, f)
	}
	if s.Uses == "" {
		return fmt.Errorf("step %q: a build step needs uses; it runs on a node", s.ID)
	}
	// 명령의 내용은 보지 않는다 — 제품은 그 문자열을 해석하지 않는다.
	if strings.TrimSpace(s.Sync) == "" {
		return fmt.Errorf("step %q: a build step needs sync", s.ID)
	}
	if len(s.Builds) == 0 {
		return fmt.Errorf("step %q: a build step needs at least one entry in builds", s.ID)
	}
	seen := map[string]bool{}
	for i, b := range s.Builds {
		if !buildName(b.Name) {
			return fmt.Errorf("step %q: builds[%d].name %q must be 1 to 64 characters of a-z, 0-9 and -",
				s.ID, i, b.Name)
		}
		if seen[b.Name] {
			return fmt.Errorf("step %q: builds name %q appears twice", s.ID, b.Name)
		}
		seen[b.Name] = true
		if strings.TrimSpace(b.Command) == "" {
			return fmt.Errorf("step %q: builds[%d].command is empty", s.ID, i)
		}
	}
	if s.IR == "" {
		return fmt.Errorf("step %q: a build step needs ir, the exact tag to bake", s.ID)
	}
	if why := irProblem(s.IR); why != "" {
		return fmt.Errorf("step %q: ir %q is not a valid tag name: %s", s.ID, s.IR, why)
	}
	return nil
}

func buildName(n string) bool {
	if n == "" || len(n) > 64 {
		return false
	}
	for _, r := range n {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
			return false
		}
	}
	return true
}

// irProblem 은 ir 이 어긴 규칙 하나를 말한다. 어긴 것이 없으면 "" 다.
//
// git 태그 이름으로 쓸 수 있는 것 가운데, 환경 변수 값과 광고 값과 셸 인용에서
// 문제가 없는 좁은 집합이다. 날짜 모양 같은 형식 규칙은 없다 — 사내 형식은
// 제품이 모른다. 조각과 끝의 . 줄은 git 의 참조 이름 규칙(git check-ref-format)이
// 거절하는 것이다. 받으면 노드가 sync 뒤 대조에서 반드시 실패한다.
func irProblem(ir string) string {
	if len(ir) > 128 {
		return "it is longer than 128 characters"
	}
	for _, r := range ir {
		if !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' ||
			r == '.' || r == '_' || r == '-' || r == '/') {
			return fmt.Sprintf("%q is not allowed; use A-Z, a-z, 0-9, '.', '_', '-' and '/'", r)
		}
	}
	switch {
	case strings.HasPrefix(ir, "-"), strings.HasPrefix(ir, "/"), strings.HasPrefix(ir, "."):
		return "it must not start with '-', '/' or '.'"
	case strings.HasSuffix(ir, "/"), strings.HasSuffix(ir, "."):
		return "it must not end with '/' or '.'"
	case strings.Contains(ir, ".."), strings.Contains(ir, "//"):
		return "it must not contain '..' or '//'"
	}
	for _, part := range strings.Split(ir, "/") {
		if strings.HasPrefix(part, ".") {
			return "no part between slashes may start with '.'"
		}
		if strings.HasSuffix(part, ".lock") {
			return "no part between slashes may end with '.lock'"
		}
	}
	return ""
}

// checkMergeStep 은 굽기 merge 단계의 칸과 값을 본다.
// 받는 칸 — id · uses · needs · merge.
func checkMergeStep(s Step) error {
	if f := otherFields(s); len(f) > 0 {
		return fmt.Errorf("step %q: a merge step does not take %s", s.ID, f[0])
	}
	if s.Uses == "" {
		return fmt.Errorf("step %q: a merge step needs uses; it runs on the node that built", s.ID)
	}
	if w := s.Merge.Wait; w != "" {
		d, err := time.ParseDuration(w)
		if err != nil {
			return fmt.Errorf("step %q: merge.wait %q is not a duration (for example \"4h\")", s.ID, w)
		}
		if d <= 0 {
			return fmt.Errorf("step %q: merge.wait must be greater than zero", s.ID)
		}
	}
	return nil
}

// checkBake 는 굽기 계약의 모양과, 계획이 짓는 굽기의 승인을 본다 (ADR-077 §2).
//
// build 나 merge 가 하나라도 있으면 굽기 계약이다. 굽기 계약은 build 하나와
// merge 하나로 끝나고, 그 앞에는 계획 단계만 올 수 있다. 계획이 지은 단계는
// 언제나 계약 끝에 붙으므로(store.applyExpands) 계획 단계가 앞에 있고 굽기가
// 뒤에 오는 모양이 저절로 나온다.
//
// needs · dispatch · adopt 를 Validate 의 앞 덩어리가 확인한 뒤에 부른다 —
// 그래야 여기서 NeedsOf 를 믿고 쓴다.
//
// stubs 는 CheckPlan 이 참조만 성립시키려고 채운 그루터기의 이름이다.
// 그루터기의 종류는 진짜가 아니므로 「build 앞은 계획 단계만」 한 줄이 건너뛴다.
func checkBake(c Contract, kinds map[string]StepKind, stubs map[string]bool) error {
	var builds, merges []int
	for i, st := range c.Steps {
		switch kinds[st.ID] {
		case KindBuild:
			builds = append(builds, i)
		case KindMerge:
			merges = append(merges, i)
		}
	}
	if len(builds) == 0 && len(merges) == 0 {
		return nil
	}
	if len(builds) > 1 {
		return fmt.Errorf("the contract has %d build steps; a bake has exactly one", len(builds))
	}
	if len(merges) > 1 {
		return fmt.Errorf("the contract has %d merge steps; a bake has exactly one", len(merges))
	}
	if len(merges) == 0 {
		return fmt.Errorf("build step %q has no merge step; the built upper would wait forever",
			c.Steps[builds[0]].ID)
	}
	if len(builds) == 0 {
		return fmt.Errorf("merge step %q has no build step to merge", c.Steps[merges[0]].ID)
	}
	b, m := builds[0], merges[0]
	build, merge := c.Steps[b], c.Steps[m]
	if m+1 < len(c.Steps) {
		return fmt.Errorf("step %q comes after merge step %q; a bake ends with its build and merge",
			c.Steps[m+1].ID, merge.ID)
	}
	if m != b+1 {
		return fmt.Errorf("step %q sits between build %q and merge %q; merge follows build directly",
			c.Steps[b+1].ID, build.ID, merge.ID)
	}
	if needs := NeedsOf(c.Steps, m); len(needs) != 1 || needs[0] != build.ID {
		return fmt.Errorf("merge step %q must need exactly [%q]", merge.ID, build.ID)
	}
	if merge.Uses != build.Uses {
		return fmt.Errorf("merge step %q must use the same role as build step %q (%q)",
			merge.ID, build.ID, build.Uses)
	}
	for _, st := range c.Steps[:b] {
		if stubs[st.ID] || planning(st, kinds[st.ID]) {
			continue
		}
		return fmt.Errorf("step %q comes before the bake but is not a planning step; only an agent "+
			"step with expands and the ask that adopts its plan may come first", st.ID)
	}
	return checkPlannedBake(c, b)
}

// planning 은 계획 단계인가다 — 계획을 짓는 agent 단계와 그 계획을 승인하는 ask 단계.
func planning(st Step, k StepKind) bool {
	return k == KindAgent && st.Expands || k == KindAsk && st.Ask.Adopts != ""
}

// checkPlannedBake 는 계획이 짓는 굽기를 사람이 승인한 뒤에만 돌게 한다.
//
// 코드가 주는 것만으로는 모자라다 — 계획의 단계는 계획 단계가 끝나면 곧바로
// 계약 끝에 붙는다. ask 의 adopts 는 계획이 제안한 success_when 을 채택할지를
// 정할 뿐이고, 거절은 adopt_when 이나 dispatch 가 있을 때만 계획을 물린다
// (store.retirePlan · rewindToPlanner). 그래서 넷이 모두 있어야 한다.
//
// 「계획이 지었나」가 아니라 「굽기 계약에 계획 단계가 있나」로 건다. Validate 는
// 어느 단계를 계획이 지었는지 모르고, 알 필요도 없다 — 굽기 계약에서 계획 단계는
// build 앞에만 올 수 있으므로, 그 계획이 무엇을 짓든 사람이 본 뒤에 굽기가 돌아야 한다.
//
// 제출 때에는 build · merge 가 아직 없으므로 걸리지 않는다. 계획이 굽기를 지어
// 붙이는 순간 늘어난 계약의 Validate 가 거절하고, 계획 단계는 오늘의 계획 거절
// 경로로 실패한다.
func checkPlannedBake(c Contract, build int) error {
	anc := ancestors(c.Steps, build)
	for _, st := range c.Steps {
		if !st.Expands {
			continue
		}
		if st.Adopt == AdoptYolo {
			return fmt.Errorf("step %q adopts its plan without asking (adopt \"yolo\"), but the contract "+
				"bakes; a bake merges into a shared lower, so a person must adopt the plan "+
				"(an ask step with adopts)", st.ID)
		}
		ask := -1
		for j, other := range c.Steps {
			if other.Ask != nil && other.Ask.Adopts == st.ID {
				ask = j
				break
			}
		}
		if ask < 0 {
			return fmt.Errorf("step %q builds a plan in a contract that bakes, but no ask step adopts it; "+
				"a person must adopt the plan before the bake runs (an ask step with adopts: %q)",
				st.ID, st.ID)
		}
		a := c.Steps[ask]
		if a.Ask.AdoptWhen == "" && a.Dispatch == nil {
			return fmt.Errorf("ask step %q adopts plan %q in a contract that bakes, but says nothing about "+
				"rejection; set adopt_when so that a rejected plan is rebuilt instead of baked",
				a.ID, st.ID)
		}
		if !anc[ask] {
			return fmt.Errorf("build step %q does not wait for ask step %q; a bake runs only after a "+
				"person adopts the plan (add it to needs)", c.Steps[build].ID, a.ID)
		}
	}
	return nil
}

// checkBakeCondition 은 build · merge 에 걸린 판정 조건을 본다.
//
// 굽기 두 단계는 고정 산출물 하나로만 판정한다. changed · within_attempts ·
// exit_code 는 그 단계에서 공허하게 참이거나 언제나 거짓이다 — 조용히 통과하면
// 계약 저자가 판정이 없는 줄 모른다.
func checkBakeCondition(cond Condition, k StepKind) error {
	var want string
	switch k {
	case KindBuild:
		want = ArtifactManifest
	case KindMerge:
		want = ArtifactMerged
	default:
		return nil
	}
	if cond.ExitCode != nil || len(cond.Changed) > 0 || cond.WithinAttempts ||
		len(cond.Produced) != 1 || cond.Produced[0] != want {
		return fmt.Errorf("success_when: step %q is a %s step; judge it by produced [%q]", cond.Step, k, want)
	}
	return nil
}
