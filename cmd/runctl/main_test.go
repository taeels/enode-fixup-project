package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/taeels/enode/internal/runctl"
)

// 이 파일은 main.go 의 곁 함수들을 거울처럼 따라간다 - permute · report ·
// verdictCode · printRun · stringList · usage. 서브커맨드 왕복은
// subcommand_test.go 가 본다.
//
// 왜 출력을 파이프로 가로채는가 - cmd/runctl 은 fmt.Printf 로 os.Stdout 에
// 직접 쓴다. 출력 대상을 인자로 받는 형태가 아니므로 패키지 변수를
// 갈아끼우는 것 말고는 찍힌 것을 읽을 방법이 없다. 이것을 이음매로 삼는
// 대신 생산 코드에 io.Writer 를 주입하는 안을 기각했다 - 이 단위는 커버리지
// 공백을 메우는 일이고, 그 일이 CLI 의 모양을 바꾸기 시작하면 무엇을
// 시험하는지가 흐려진다.

// captureOutput 은 os.Stdout 과 os.Stderr 를 파이프로 바꿔 fn 이 찍은 것을
// 돌려준다.
//
// 고루틴으로 비우는 이유 - 파이프 버퍼(64KiB)가 차면 fn 이 쓰기에서 멈추고,
// 멈추면 테스트가 왜 멈췄는지 출력에 아무것도 안 남는다. asks 출력이 그만큼
// 길어질 일은 없지만 그 실패 양식은 진단이 불가능하다.
func captureOutput(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe for stdout: %v", err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe for stderr: %v", err)
	}
	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outW, errW

	var ob, eb bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); _, _ = io.Copy(&ob, outR) }()
	go func() { defer wg.Done(); _, _ = io.Copy(&eb, errR) }()

	func() {
		defer func() {
			os.Stdout, os.Stderr = oldOut, oldErr
			outW.Close()
			errW.Close()
		}()
		fn()
	}()
	wg.Wait()
	outR.Close()
	errR.Close()
	return ob.String(), eb.String()
}

// permuteWith 는 주어진 플래그 집합 아래에서 permute 를 부른다.
//
// permute 는 flag.VisitAll 로 전역 집합을 읽어 "값을 받는 플래그"를 가리므로,
// 무엇이 등록되어 있는지가 곧 입력이다. 여기서 생산의 플래그 목록을 베끼지
// 않는 것은 일부러다 - 베끼면 같은 사실이 두 곳에 살고 한쪽만 낡는다.
// 이 테스트가 고정하는 것은 목록이 아니라 규칙이다: 값을 받는 플래그는 다음
// 인자를 데려가고 불리언은 안 데려간다. 목록 자체는 run() 을 통과하는
// subcommand_test.go 의 TestRun_SubmitHonoursAFlagWrittenAfterThePositional
// 이 지킨다.
func permuteWith(t *testing.T, args []string) []string {
	t.Helper()
	old := flag.CommandLine
	fs := flag.NewFlagSet("runctl", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Bool("wait", false, "boolean; takes no value")
	fs.String("o", "", "string; takes a value")
	flag.CommandLine = fs
	t.Cleanup(func() { flag.CommandLine = old })
	return permute(args)
}

func TestPermute_MovesFlagsAheadOfThePositionalArguments(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   []string
		want []string
	}{
		{
			// 생산 주석이 이유를 적는다: 표준 flag 는 첫 위치인자에서 파싱을
			// 멈추므로, 이것을 안 옮기면 --wait 가 조용히 무시된다.
			name: "a boolean written after the positional moves ahead of it",
			in:   []string{"submit", "x.json", "--wait"},
			want: []string{"--wait", "submit", "x.json"},
		},
		{
			name: "a value-taking flag drags its detached value along",
			in:   []string{"record", "r1", "-o", "out.tar"},
			want: []string{"-o", "out.tar", "record", "r1"},
		},
		{
			name: "an attached value needs no dragging",
			in:   []string{"record", "r1", "-o=out.tar"},
			want: []string{"-o=out.tar", "record", "r1"},
		},
		{
			// 불리언이 다음 인자를 데려가면 서브커맨드가 플래그 값으로
			// 먹혀 없어진다.
			name: "a boolean does not swallow the token after it",
			in:   []string{"--wait", "submit", "x.json"},
			want: []string{"--wait", "submit", "x.json"},
		},
		{
			name: "everything after the double dash stays a positional",
			in:   []string{"submit", "--", "-not-a-flag"},
			want: []string{"submit", "-not-a-flag"},
		},
		{
			name: "a bare dash is a positional, not a flag",
			in:   []string{"capabilities", "-"},
			want: []string{"capabilities", "-"},
		},
		{
			name: "nothing to permute",
			in:   nil,
			want: nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := permuteWith(t, tc.in)
			if strings.Join(got, " ") != strings.Join(tc.want, " ") {
				t.Fatalf("permute(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestReport_TranslatesTheWireFaultIntoAnExitCode(t *testing.T) {
	// 경계가 둘이라 하나로 통일하지 않는다 (INVARIANTS §4) - 이 대응이
	// 그 두 경계를 잇는 유일한 자리다.
	for _, tc := range []struct {
		name     string
		err      error
		wantCode int
		wantErr  string
	}{
		{"no error is exitOK", nil, exitOK, ""},
		{"400 is the caller's fault", &runctl.Fail{Code: 400, Reason: "bad contract"}, exitRequest, "400 bad contract"},
		{"409 is the caller's fault too - retry later", &runctl.Fail{Code: 409, Reason: "busy"}, exitRequest, "409 busy"},
		{"422 is the caller's fault", &runctl.Fail{Code: 422, Reason: "no such capability"}, exitRequest, "422 no such capability"},
		{"500 is the system's fault", &runctl.Fail{Code: 500, Reason: "boom"}, exitSystem, "500 boom"},
		{"503 is the system's fault", &runctl.Fail{Code: 503, Reason: "down"}, exitSystem, "503 down"},
		{"a transport error is the system's fault", errors.New("dial tcp: refused"), exitSystem, "dial tcp: refused"},
		{
			// errors.As 로 캐낸다 - 감싸인 Fail 도 같은 코드여야 한다.
			"a wrapped fault is still a fault",
			fmt.Errorf("submitting: %w", &runctl.Fail{Code: 422, Reason: "unmatched"}),
			exitRequest, "422 unmatched",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var code int
			_, stderr := captureOutput(t, func() { code = report(tc.err) })
			if code != tc.wantCode {
				t.Fatalf("report(%v) = %d, want %d", tc.err, code, tc.wantCode)
			}
			if tc.wantErr == "" {
				if stderr != "" {
					t.Fatalf("report(nil) wrote %q to stderr, want nothing", stderr)
				}
				return
			}
			if !strings.Contains(stderr, tc.wantErr) {
				t.Fatalf("report stderr = %q, want it to name %q", stderr, tc.wantErr)
			}
		})
	}
}

func TestVerdictCode_OnlySucceededIsZero(t *testing.T) {
	// 요청이 정상이었는데 일이 실패한 것이므로 2 가 아니라 1 이다.
	if got := verdictCode(&runctl.Run{State: "SUCCEEDED"}); got != exitOK {
		t.Fatalf("SUCCEEDED = %d, want %d", got, exitOK)
	}
	for _, s := range []string{"FAILED", "RUNNING", "PENDING", ""} {
		if got := verdictCode(&runctl.Run{State: s}); got != exitRunFail {
			t.Fatalf("state %q = %d, want %d (the request was valid; the work was not)", s, got, exitRunFail)
		}
	}
}

func TestPrintRun_PutsWarningsAboveEverythingElse(t *testing.T) {
	// ADR-061 §2 - 제출은 됐지만 뜻대로 안 도는 것이 있으면 그것부터 보여야
	// 한다. 아래 줄들에 묻히면 못 읽는다.
	r := &runctl.Run{
		RunID: "run-7", State: "RUNNING",
		Warnings: []string{"retries were requested but this step cannot retry"},
		Assigned: []runctl.Assigned{{As: "builder", Nodes: []struct {
			Node  string `json:"node"`
			Label string `json:"label"`
		}{{Node: "n1", Label: "kim@box"}}}},
		Steps: []runctl.Step{
			{Seq: 1, ID: "build", State: "SUCCEEDED", Node: "n1"},
			{Seq: 2, ID: "flash", State: "RUNNING", Node: "n1", Attempt: 1},
			{Seq: 3, ID: "check", State: "PENDING", Needs: []string{"build", "flash"}},
		},
	}
	stdout, _ := captureOutput(t, func() { printRun(r) })

	iw := strings.Index(stdout, "warning: retries were requested")
	if iw < 0 {
		t.Fatalf("printRun output = %q, want the warning in it", stdout)
	}
	for _, later := range []string{"builder", " 1 build", " 2 flash", " 3 check"} {
		if i := strings.Index(stdout, later); i < 0 || i < iw {
			t.Fatalf("printRun put %q at %d but the warning at %d; the warning must come first", later, i, iw)
		}
	}
	if !strings.Contains(stdout, "run-7  RUNNING") {
		t.Fatalf("printRun output = %q, want the run id and state on the first line", stdout)
	}
	// Attempt 는 0-based 이고 사람이 읽는 것은 1-based 다.
	if !strings.Contains(stdout, "(attempt 2)") {
		t.Fatalf("printRun output = %q, want attempt 1 shown as (attempt 2)", stdout)
	}
	// 기다리는 중이면 무엇을 기다리는지를 같이 보여준다.
	if !strings.Contains(stdout, "waits on build · flash") {
		t.Fatalf("printRun output = %q, want the pending step to name what it waits on", stdout)
	}
	if strings.Contains(stdout, "waits on") != strings.Contains(stdout, "PENDING") {
		t.Fatalf("printRun output = %q, only a PENDING step names what it waits on", stdout)
	}
}

func TestPrintRun_UnpacksTheVerdictAndSurvivesAnUnreadableOne(t *testing.T) {
	good := &runctl.Run{RunID: "run-8", State: "FAILED", Verdict: json.RawMessage(
		`{"state":"FAILED","checks":[
		  {"step":"build","what":"exit==0","ok":true,"note":""},
		  {"step":"flash","what":"device answers","ok":false,"note":"timed out"}]}`)}
	stdout, _ := captureOutput(t, func() { printRun(good) })
	if !strings.Contains(stdout, "ok  build exit==0") {
		t.Fatalf("printRun output = %q, want the passing check marked ok", stdout)
	}
	if !strings.Contains(stdout, "bad flash device answers timed out") {
		t.Fatalf("printRun output = %q, want the failing check marked bad and its note kept", stdout)
	}

	// 판정이 우리가 못 읽는 모양이어도 Run 줄은 나와야 한다 - 여기서 죽으면
	// status 가 통째로 안 보인다.
	bad := &runctl.Run{RunID: "run-9", State: "FAILED", Verdict: json.RawMessage(`["not an object"]`)}
	stdout, _ = captureOutput(t, func() { printRun(bad) })
	if !strings.Contains(stdout, "run-9  FAILED") {
		t.Fatalf("printRun output = %q, want the run line even when the verdict cannot be read", stdout)
	}
	if strings.Contains(stdout, "ok ") || strings.Contains(stdout, "bad ") {
		t.Fatalf("printRun output = %q, want no check lines from an unreadable verdict", stdout)
	}
}

func TestStringList_CollectsEveryRepeatedSet(t *testing.T) {
	var l stringList
	for _, v := range []string{"kind=approve", "note=looks right"} {
		if err := l.Set(v); err != nil {
			t.Fatalf("Set(%q) = %v, want nil", v, err)
		}
	}
	if len(l) != 2 {
		t.Fatalf("after two Set calls len = %d, want 2 (--set is repeatable)", len(l))
	}
	if got := l.String(); got != "kind=approve,note=looks right" {
		t.Fatalf("String() = %q, want the values joined by a comma", got)
	}
	var empty stringList
	if got := empty.String(); got != "" {
		t.Fatalf("empty String() = %q, want the empty string", got)
	}
}
