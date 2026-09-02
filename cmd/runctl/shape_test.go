package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/taeels/enode/internal/contract"
)

// 셋은 Mediator 없이 서야 한다 (ADR-066 §4).
//
// 여기서 재는 것이 그 성질이다 — 환경을 통째로 비우고 부른다. 토큰이나
// 주소를 요구하기 시작하면 「계약을 어떻게 쓰나」를 물으려고 먼저 함대를
// 세워야 하고, 그러면 이 셋이 있어야 하는 이유가 사라진다.
func TestShapeCommands_NeedNoFleet(t *testing.T) {
	t.Setenv("ENODE_MEDIATOR", "")
	t.Setenv("ENODE_TOKEN", "")
	for _, tc := range []struct {
		name string
		call func() int
	}{
		{"example 목록", func() int { return cmdExample("") }},
		{"example command", func() int { return cmdExample("command") }},
		{"schema 목록", func() int { return cmdSchema("") }},
		{"schema steps", func() int { return cmdSchema("steps") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if code := tc.call(); code != exitOK {
				t.Errorf("= %d, want %d", code, exitOK)
			}
		})
	}
}

// 없는 이름을 물으면 무엇이 있는지 말해야 한다.
func TestShapeCommands_UnknownNameIsNotSilent(t *testing.T) {
	if code := cmdExample("nope"); code != exitRequest {
		t.Errorf("cmdExample(nope) = %d, want %d", code, exitRequest)
	}
	if code := cmdSchema("nope"); code != exitRequest {
		t.Errorf("cmdSchema(nope) = %d, want %d", code, exitRequest)
	}
}

// lint 가 잡아야 하는 것은 「유효하지만 뜻이 헐거운 계약」이다.
//
// ADR-066 의 출발점이 그것이었다 — success_when 이 없는 계약은 Validate 를
// 통과하고 dry-run 도 통과하는데, 명령이 실패해도 Run 이 성공으로 끝난다.
func TestLintWarnings_CatchesTheSilentSuccess(t *testing.T) {
	var c contract.Contract
	if err := json.Unmarshal([]byte(`{
	  "run_id": "t", "requires": [{"as":"w","capability":"agent.reason"}],
	  "steps": [{"id":"shot","uses":"w","run":["screencapture","$OUT/screen.png"],
	             "out":["screen.png"]}]
	}`), &c); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(lintWarnings(c), "\n")
	if !strings.Contains(got, "success_when") {
		t.Fatalf("경고가 success_when 을 안 짚는다:\n%s", got)
	}
	// 제안이 그대로 붙여넣어져야 한다 — 「고치라」고만 하면 다시 찾아야 한다.
	if !strings.Contains(got, `"step": "shot"`) || !strings.Contains(got, `"exit_code": 0`) ||
		!strings.Contains(got, `"produced": ["screen.png"]`) {
		t.Fatalf("제안이 붙여넣을 수 있는 모양이 아니다:\n%s", got)
	}
}

// 단계가 여럿일 때 판정에서 빠진 것을 짚어야 한다.
func TestLintWarnings_NamesTheUnjudgedStep(t *testing.T) {
	var c contract.Contract
	if err := json.Unmarshal([]byte(`{
	  "run_id": "t", "requires": [{"as":"w","capability":"agent.reason"}],
	  "steps": [{"id":"a","uses":"w","run":["true"],"out":["x"]},
	            {"id":"b","uses":"w","run":["true"],"out":["y"]}],
	  "success_when": [{"step":"a","exit_code":0}]
	}`), &c); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(lintWarnings(c), "\n")
	if !strings.Contains(got, `"b"`) {
		t.Fatalf("판정 없는 단계 b 를 안 짚는다:\n%s", got)
	}
	if strings.Contains(got, `단계 "a"`) {
		t.Fatalf("판정이 있는 a 를 짚는다:\n%s", got)
	}
}

// 없는 단계를 가리키는 조건은 영원히 안 맞는다.
func TestLintWarnings_ConditionPointingNowhere(t *testing.T) {
	var c contract.Contract
	if err := json.Unmarshal([]byte(`{
	  "run_id": "t", "requires": [{"as":"w","capability":"agent.reason"}],
	  "steps": [{"id":"a","uses":"w","run":["true"],"out":["x"]}],
	  "success_when": [{"step":"a","exit_code":0},{"step":"ghost","exit_code":0}]
	}`), &c); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(lintWarnings(c), "\n")
	if !strings.Contains(got, "ghost") {
		t.Fatalf("없는 단계를 가리키는 조건을 안 짚는다:\n%s", got)
	}
}

// 예시는 경고가 하나도 없어야 한다 — 베끼는 쪽이 같은 자리에 빠지면 안 된다.
func TestExamples_LintClean(t *testing.T) {
	for _, name := range contract.ExampleNames() {
		t.Run(name, func(t *testing.T) {
			b, err := contract.Example(name)
			if err != nil {
				t.Fatal(err)
			}
			var c contract.Contract
			if err := json.Unmarshal(b, &c); err != nil {
				t.Fatal(err)
			}
			if w := lintWarnings(c); len(w) > 0 {
				t.Errorf("예시에 경고가 있다:\n%s", strings.Join(w, "\n"))
			}
		})
	}
}

// lint 가 파일을 못 읽거나 못 파싱하면 그 자리에서 말해야 한다.
func TestCmdLint_SaysWhatIsWrongWithTheFile(t *testing.T) {
	if code := cmdLint(""); code != exitRequest {
		t.Errorf("cmdLint(\"\") = %d, want %d", code, exitRequest)
	}
	if code := cmdLint(filepath.Join(t.TempDir(), "missing.json")); code != exitRequest {
		t.Error("없는 파일인데 거절하지 않는다")
	}
	bad := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(bad, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code := cmdLint(bad); code != exitRequest {
		t.Error("깨진 JSON 인데 거절하지 않는다")
	}
}

// 거절 사유가 다음 행동을 달고 나와야 한다.
func TestNextStep_PointsAtTheCommandThatHasTheAnswer(t *testing.T) {
	for _, tc := range []struct{ reason, want string }{
		{`unknown capability: "nope"`, "runctl capabilities"},
		{"no node satisfies requires[0]", "runctl capabilities"},
		{"step \"a\" is missing required field", "runctl lint"},
	} {
		if got := nextStep(tc.reason); !strings.Contains(got, tc.want) {
			t.Errorf("nextStep(%q) = %q, want it to mention %q", tc.reason, got, tc.want)
		}
	}
	// 못 맞히면 조용하다 — 틀린 안내를 하느니 안 한다.
	if got := nextStep("database is on fire"); got != "" {
		t.Errorf("nextStep(모르는 사유) = %q, want \"\"", got)
	}
}
