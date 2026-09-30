package enode

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/store"
)

// 굽기의 모든 플랫폼 시험 — 노드가 받는 칸 넷과 결과 칸 둘 (bake 유닛 · FD 엔티티 1 · 6절).

// contractStep 이 굽기 칸 넷을 옮긴다 — merge.wait 을 MergeWait 이 읽고, 없으면 기본 4시간이다. build 의 예산은
// 종류를 안 보므로 계약이 적은 값 그대로다.
func TestContractStep_CarriesTheBakeFields(t *testing.T) {
	merge := contractStep(&Step{Kind: "merge", Merge: &contract.Merge{Wait: "30m"}})
	if got := merge.MergeWait(); got != 30*time.Minute {
		t.Fatalf("MergeWait = %s, want 30m", got)
	}
	if k, err := merge.Kind(); err != nil || k != contract.KindMerge {
		t.Fatalf("a merge step reads as %v, %v", k, err)
	}
	if got := contractStep(&Step{Kind: "merge", Merge: &contract.Merge{}}).MergeWait(); got != contract.DefaultMergeWait {
		t.Fatalf("an empty merge waits %s, want %s", got, contract.DefaultMergeWait)
	}
	build := &Step{Kind: "build", Effect: contract.EffectPrepare, Sync: "sync", IR: "ir-1",
		Builds:   []contract.Build{{Name: "config-a", Command: "make"}},
		Budget:   &contract.Budget{Finalize: "5m", Upload: "10m"},
		Run:      []string{"left", "over"},
		Discover: false}
	cs := contractStep(build)
	if k, err := cs.Kind(); err != nil || k != contract.KindBuild {
		t.Fatalf("a build step reads as %v, %v", k, err)
	}
	if cs.Run != nil || cs.Sync != "sync" || cs.IR != "ir-1" || !reflect.DeepEqual(cs.Builds, build.Builds) {
		t.Fatalf("contractStep(build) = %+v", cs)
	}
	if f, u := cs.Budgets(); f != 5*time.Minute || u != 10*time.Minute {
		t.Fatalf("build budgets = %s, %s", f, u)
	}
	if cs.EffectOrDefault() != contract.EffectPrepare {
		t.Fatalf("build effect = %q", cs.EffectOrDefault())
	}
}

// 조각 5 의 (1) — Mediator 가 싣는 모양 그대로 JSON 으로 옮겨 노드의 Step 으로 풀면 contractStep 의 MergeWait 이
// 계약의 값이다. 굽기 칸 셋도 같은 값으로 온다 (finalize 가 넘긴 틈 · 받는 일 19 · 20).
func TestClaim_MergeWaitReachesTheNode(t *testing.T) {
	builds := []contract.Build{{Name: "config-a", Command: "make a"}, {Name: "config-b", Command: "make b"}}
	sent := []store.Claimed{
		{StepID: "r#01", RunID: "r", Seq: 1, Name: "build", Kind: "build", Effect: contract.EffectPrepare,
			Sync: "git fetch", Builds: builds, IR: "ir-1"},
		{StepID: "r#02", RunID: "r", Seq: 2, Name: "merge", Kind: "merge", Merge: &contract.Merge{Wait: "2s"}},
	}
	var got []Step
	for _, c := range sent {
		b, err := json.Marshal(c)
		if err != nil {
			t.Fatal(err)
		}
		var s Step
		if err := json.Unmarshal(b, &s); err != nil {
			t.Fatal(err)
		}
		got = append(got, s)
	}
	if got[0].Sync != "git fetch" || got[0].IR != "ir-1" || !reflect.DeepEqual(got[0].Builds, builds) {
		t.Fatalf("the build step arrived as %+v", got[0])
	}
	if w := contractStep(&got[1]).MergeWait(); w != 2*time.Second {
		t.Fatalf("merge.wait arrived as %s, want 2s", w)
	}
	if w := contractStep(&got[0]).MergeWait(); w != contract.DefaultMergeWait {
		t.Fatalf("a build step waits %s", w)
	}
}

// 결과의 build · merge 칸은 store.StepResult 와 같은 JSON 이름이다. 없으면 안 나간다 — 옛 모양의 결과가 그대로다.
func TestResult_TheBakeFieldsUseTheMediatorNames(t *testing.T) {
	ir := "ir-1"
	b, _ := json.Marshal(Result{Node: "n", Build: &contract.BuildManifest{Head: "abc", IR: &ir, HeadTags: []string{}},
		Merge: &contract.MergeResult{IR: &ir}})
	for _, k := range []string{`"build":{`, `"head":"abc"`, `"head_tags":[]`, `"merge":{`, `"ops":{`} {
		if !strings.Contains(string(b), k) {
			t.Errorf("%s missing in %s", k, b)
		}
	}
	var sealed store.StepResult
	if err := json.Unmarshal(b, &sealed); err != nil {
		t.Fatal(err)
	}
	if sealed.Build == nil || sealed.Build.Head != "abc" || sealed.Merge == nil || *sealed.Merge.IR != ir {
		t.Fatalf("the Mediator read %+v", sealed)
	}
	b, _ = json.Marshal(Result{Node: "n"})
	for _, k := range []string{`"build"`, `"merge"`} {
		if strings.Contains(string(b), k) {
			t.Errorf("an old-shaped result carries %s: %s", k, b)
		}
	}
}

// 계약의 명령은 격리 실행 환경 안에서만 돈다 (계획 3.1)

// bakeSources 는 굽기 흐름의 파일 여섯이다 — 이 파일들은 호스트 프로세스를 띄우지 않는다. merge-helper 를 unshare
// 로 여는 mergehelper_linux.go 만 예외다.
var bakeSources = []string{"bake.go", "bake_build.go", "bake_merge.go", "bake_resume.go", "bakerule.go", "bakefile.go"}

// hostProcessUses 는 소스 하나에서 호스트 프로세스를 띄울 수 있는 자리를 모은다 — import 의 os/exec · syscall,
// exec. · os.StartProcess · syscall. 의 selector, 같은 패키지의 NativeRuntime · DetectRepo · detectRepoManifest ·
// gitIn · gitOut · gitOutName · child. import 만 보면 같은 패키지의 함수로 호스트 git 을 부르는 되돌림을 못 잡는다.
func hostProcessUses(t *testing.T, name string, src any) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, name, src, 0)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	at := func(n ast.Node, what string) { out = append(out, fset.Position(n.Pos()).String()+" "+what) }
	for _, imp := range f.Imports {
		if p, _ := strconv.Unquote(imp.Path.Value); p == "os/exec" || p == "syscall" {
			at(imp, "imports "+p)
		}
	}
	banned := map[string]bool{"NativeRuntime": true, "DetectRepo": true, "detectRepoManifest": true, "gitIn": true,
		"gitOut": true, "gitOutName": true, "child": true}
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			if id, ok := x.X.(*ast.Ident); ok {
				switch {
				case id.Name == "exec", id.Name == "syscall", id.Name == "os" && x.Sel.Name == "StartProcess":
					at(x, id.Name+"."+x.Sel.Name)
				}
			}
			return true
		case *ast.Ident:
			if banned[x.Name] {
				at(x, x.Name)
			}
		}
		return true
	})
	sort.Strings(out)
	return out
}

// 굽기 흐름의 파일 여섯은 호스트 프로세스를 띄우지 않는다 — 누가 그 이름을 부르게 하면 빨개진다 (경계 시험의 논리).
// 대조군 — 같은 검사가 되돌림 모양의 글에서 그 자리를 찾는다.
func TestBakeSources_StartNoHostProcess(t *testing.T) {
	for _, name := range bakeSources {
		if uses := hostProcessUses(t, name, nil); len(uses) != 0 {
			t.Errorf("%s can start a host process:\n  %s", name, strings.Join(uses, "\n  "))
		}
	}
	control := `package enode
import ("os"; "os/exec"; "syscall")
func f(ctx any) {
	exec.Command("bash", "-c", "x")
	os.StartProcess("x", nil, nil)
	_ = syscall.Getuid()
	_ = NativeRuntime{}
	DetectRepo(ctx, "/w")
	gitIn(ctx, "/w")
	child(nil)
}`
	uses := hostProcessUses(t, "control.go", control)
	if len(uses) != 9 {
		t.Fatalf("the check misses a host process in the control: %q", uses)
	}
}
