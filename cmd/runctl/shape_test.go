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
		t.Fatalf("warnings = %q, want them to name success_when", got)
	}
	// 제안이 그대로 붙여넣어져야 한다 — 「고치라」고만 하면 다시 찾아야 한다.
	if !strings.Contains(got, `"step": "shot"`) || !strings.Contains(got, `"exit_code": 0`) ||
		!strings.Contains(got, `"produced": ["screen.png"]`) {
		t.Fatalf("warnings = %q, want a paste-ready condition", got)
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
	if !strings.Contains(got, `step "b"`) {
		t.Fatalf("warnings = %q, want them to name the unjudged step b", got)
	}
	if strings.Contains(got, `step "a"`) {
		t.Fatalf("warnings = %q, want step a left alone; it is judged", got)
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
				t.Errorf("example has warnings:\n%s", strings.Join(w, "\n"))
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
		t.Error("cmdLint on a missing file did not reject it")
	}
	bad := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(bad, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code := cmdLint(bad); code != exitRequest {
		t.Error("cmdLint on broken JSON did not reject it")
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

// 문맥 없는 에이전트가 맞혀야 했던 것들이 이제 출력에 글자로 있다.
//
// decisions.md 6절 ㉕ 의 실측이 근거다 — 빈 디렉터리에 runctl 하나만 둔
// 에이전트가 Run 스물넷을 썼고 그중 열다섯이 팩 tar 배치를 맞히는 데 들었다.
// 거절 문구는 슬롯 셋을 이름으로 말했는데, 그 슬롯이 디스크에서 무엇으로
// 불리는지는 어디에도 없었다. 그 글자들을 여기서 센다 — 지우면 빨개진다.
func TestSchema_CarriesWhatTheErrorMessagesDoNotSay(t *testing.T) {
	for _, tc := range []struct {
		section string
		want    []string
	}{
		{"pack", []string{
			"mcp.json",                         // 파일 이름.  거절 문구가 안 말하던 것
			"mcpServers",                       // 그 안의 키.  같은 자리
			"skills/<name>/SKILL.md",           // 슬롯의 디스크 배치
			"agents/<name>.md",                 //
			`"pack": "<blob>"`,                 // 두 키가 함께 있어야 한다
			`"from": ["<blob>"]`,               //
			"pack entry ./ has an unsafe name", // -C dir . 함정
			"runctl example pack",              // 다음 걸음
		}},
		{"io", []string{"$OUT", "$IN", "cwd"}},
		{"requires", []string{"runctl capabilities"}},
		{"steps", []string{"runctl schema io"}},
	} {
		t.Run(tc.section, func(t *testing.T) {
			var code int
			out, _ := captureOutput(t, func() { code = cmdSchema(tc.section) })
			if code != exitOK {
				t.Fatalf("cmdSchema(%q) = %d, want %d", tc.section, code, exitOK)
			}
			for _, w := range tc.want {
				if !strings.Contains(out, w) {
					t.Errorf("schema %s does not carry %q", tc.section, w)
				}
			}
		})
	}
}

// 목록이 새 절 둘을 알려야 한다 — 이름을 미리 아는 사람만 찾을 수 있으면
// 적어 둔 값이 절반이다.
func TestSchema_TheListNamesTheProseSections(t *testing.T) {
	out, _ := captureOutput(t, func() { cmdSchema("") }) //nolint:errcheck
	for _, w := range []string{"io", "pack"} {
		if !strings.Contains(out, w) {
			t.Errorf("schema list does not name %q", w)
		}
	}
}

// 팩 예시가 두 키를 함께 들고 있어야 한다.
//
// 하나만 적으면 그 단계는 팩 없이 돈다 — 잊은 것을 우리가 못 잡는다.
// 예시가 출발점이므로 여기서 둘이 갈리면 베끼는 쪽이 그대로 갈린다.
func TestExamplePack_CarriesBothKeys(t *testing.T) {
	b, err := contract.Example("pack")
	if err != nil {
		t.Fatal(err)
	}
	var c contract.Contract
	if err := json.Unmarshal(b, &c); err != nil {
		t.Fatal(err)
	}
	var produced, carried, laid string
	for _, s := range c.Steps {
		if len(s.Out) > 0 && s.Agent == nil {
			produced = s.Out[0]
		}
		if s.Agent != nil {
			if v, ok := s.Agent["pack"].(string); ok {
				carried = v
			}
		}
	}
	if produced == "" || carried == "" {
		t.Fatalf("the example does not show a blob being produced and carried: %q %q", produced, carried)
	}
	if produced != carried {
		t.Errorf("agent.pack is %q but the earlier step produces %q", carried, produced)
	}
	// in.from 이 같은 이름을 가리켜야 $IN 에 깔린다.
	raw := string(b)
	laid = `"from": ["` + produced + `"]`
	if !strings.Contains(raw, laid) {
		t.Errorf("the example does not lay the blob down with %s", laid)
	}
}
