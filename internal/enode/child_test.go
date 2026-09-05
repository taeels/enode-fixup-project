package enode

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// 감싸지 않은 자식 실행을 센다.
//
// 왜 소스를 훑나 — 이 결함은 윈도우에서만 티가 난다. 리눅스에서 개발하는
// 동안에는 자리를 하나 빠뜨려도 아무것도 빨개지지 않고, 사람이 윈도우에서
// 창 제목이 이상하다고 말해 줄 때까지 남는다. 그래서 「지켜졌는지 세는」
// 쪽으로 옮긴다.
//
// glyphscan 이 문자열 리터럴을 훑는 것과 같은 성질이다.
func TestEveryChildProcessIsWrapped(t *testing.T) {
	// exec.Command( 또는 exec.CommandContext( 를 찾되, 바로 앞이 child(
	// 인 것은 뺀다.
	call := regexp.MustCompile(`exec\.Command(Context)?\(`)
	wrapped := regexp.MustCompile(`child\(exec\.Command(Context)?\(`)

	ents, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	var bad []string
	for _, e := range ents {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		if name == "child.go" { // 감싸는 쪽이다
			continue
		}
		b, err := os.ReadFile(filepath.Join(".", name))
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			if !call.MatchString(line) || wrapped.MatchString(line) {
				continue
			}
			bad = append(bad, name+":"+strconv.Itoa(i+1)+"  "+strings.TrimSpace(line))
		}
	}
	if len(bad) > 0 {
		t.Fatalf("child processes not wrapped in child:\n  %s\n\n"+
			"on windows a child inherits the parent console and can change its title;\n"+
			"see childAttr in console_windows.go", strings.Join(bad, "\n  "))
	}
}

// 갈래가 실제로 갈려 있는지 — 유닉스에서는 nil 이어야 한다.
// 윈도우 값은 이 시험이 안 본다. 리눅스에서 재는 커버리지의 분모에 들어가면
// 도달할 수 없는 문장이 되기 때문이다 (.coverage-contract.yml 의 플랫폼 고정).
func TestChildAttr_IsNilOffWindows(t *testing.T) {
	if got := childAttr(); got != nil {
		t.Errorf("childAttr() = %+v, want nil off windows", got)
	}
}

// child 은 받은 cmd 를 그대로 돌려준다 — 체이닝이 성립해야 한다.
// identity.go 가 child(...).Output() 으로 쓰므로 이것이 깨지면 컴파일은
// 되지만 감싸는 효과가 사라진다.
func TestChild_ReturnsTheSameCommand(t *testing.T) {
	want := exec.Command("true")
	if got := child(want); got != want {
		t.Errorf("child returned a different command")
	}
}
