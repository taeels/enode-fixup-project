package main

import (
	"encoding/json"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// 서브커맨드 왕복. run() 하나가 이 패키지의 224 문장 중 167 을 들고 있고,
// 그 안이 전부 "무엇을 보내고 무엇을 찍고 어떤 종료코드를 내는가" 다.
//
// 경계를 셋 다 실물로 세운다 - HTTP 는 httptest, 신원은 진짜 git, 파일은
// t.TempDir(). 목을 세우지 않는 이유는 이 셋이 전부 실물로 설 수 있어서다.
//
// 종료코드가 곧 계약이다 (INVARIANTS §4) - 셸이 필요로 하는 구분이
// "일이 실패했나 / 내 요청이 틀렸나 / 시스템이 죽었나" 이므로 거의 모든
// 테스트가 그 코드를 단언한다.

const testToken = "test-token"
const testEmail = "runctl-test@example.invalid"

type cli struct {
	t   *testing.T
	srv *httptest.Server
	// seen 은 마지막 요청의 헤더다. 경계를 넘어간 것이 무엇인지 본다.
	auth, principal string
}

// newCLI 는 가짜 Mediator 를 세우고 신원과 자리를 격리한다.
//
// 왜 GIT_CONFIG_GLOBAL 과 t.Chdir 을 함께 쓰는가 - runctl.Principal 은
// `git config --get user.email` 을 쓴다(--global 이 아니다). 저장소 안에서
// 돌면 그 저장소의 local 설정이 이기므로, 기계마다 다른 값이 들어와 단언이
// 흔들린다. 저장소 밖으로 나가고 전역 설정을 우리가 만든 파일로 고정하면
// 그 값이 무엇인지 이 파일이 정한다.
func newCLI(t *testing.T, h http.HandlerFunc) *cli {
	t.Helper()
	c := &cli{t: t}
	c.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.auth = r.Header.Get("Authorization")
		c.principal = r.Header.Get("X-Enode-Principal")
		h(w, r)
	}))
	t.Cleanup(c.srv.Close)

	gitconf := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(gitconf, []byte("[user]\n\temail = "+testEmail+"\n"), 0o600); err != nil {
		t.Fatalf("write a git config for the test: %v", err)
	}
	t.Chdir(t.TempDir())
	t.Setenv("GIT_CONFIG_GLOBAL", gitconf)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("ENODE_MEDIATOR", c.srv.URL)
	t.Setenv("ENODE_TOKEN", testToken)
	return c
}

// exec 은 os.Args 를 갈아끼우고 run() 을 부른다.
//
// flag.CommandLine 을 매번 새것으로 바꾸는 이유가 둘이다. 첫째, run() 이
// 같은 이름의 플래그를 전역 집합에 다시 등록하므로 두 번째 호출이
// "flag redefined" 로 패닉한다. 둘째, 생산의 flag.CommandLine 은
// ExitOnError 라 파싱 실패가 os.Exit(2) 로 테스트 바이너리를 통째로 데리고
// 나간다. ContinueOnError 로 두면 그 자리에서 죽는 대신 실패한다 - 이
// 파일의 어떤 테스트도 잘못된 플래그를 먹이지 않으므로 그 차이가 실제로
// 쓰이지는 않는다.
//
// 새 집합의 출력을 io.Discard 로 막던 것을 걷었다. 막아 두면 usage() 가
// 부르는 flag.PrintDefaults() 의 출력만 통째로 사라져서, 이 파일은
// "CLI 가 찍은 것" 이라고 부르는 것 중 한 갈래를 못 본다. 사용법에
// 토큰이 실려 나간 결함이 그 사각지대에서 살았다. 지금은 출력을 안
// 지정하므로 flag 가 쓰는 시점의 os.Stderr, 즉 captureOutput 이 끼운
// 파이프로 간다 - 생산에서 사람이 보는 것과 같은 자리다.
func (c *cli) exec(args ...string) (code int, stdout, stderr string) {
	c.t.Helper()
	oldArgs, oldFlags := os.Args, flag.CommandLine
	fs := flag.NewFlagSet("runctl", flag.ContinueOnError)
	os.Args, flag.CommandLine = append([]string{"runctl"}, args...), fs
	defer func() { os.Args, flag.CommandLine = oldArgs, oldFlags }()
	stdout, stderr = captureOutput(c.t, func() { code = run() })
	return code, stdout, stderr
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Errorf("encoding the fake mediator response: %v", err)
	}
}

// fault 는 와이어의 오류 봉투다 - internal/api 의 fail() 이 내는 모양.
func fault(t *testing.T, w http.ResponseWriter, code int, reason string) {
	t.Helper()
	writeJSON(t, w, code, map[string]any{
		"error": map[string]any{"code": code, "reason": reason},
	})
}

func contractFile(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "contract.json")
	if err := os.WriteFile(p, []byte(`{"version":1,"steps":[]}`), 0o600); err != nil {
		t.Fatalf("write the contract file: %v", err)
	}
	return p
}

// ── 플래그 파싱보다 앞에 있는 것들 ──────────────────────────────────────

func TestRun_VersionAnswersWithoutAMediatorOrAToken(t *testing.T) {
	// ADR-056 - 토큰이 없어도 답한다. 실측(vm-scratch-5)에서 계획이
	// `enode --version` 에 막힌 것과 같은 종류의 구멍을 막는 자리다.
	t.Setenv("ENODE_MEDIATOR", "")
	t.Setenv("ENODE_TOKEN", "")
	for _, arg := range []string{"--version", "-version"} {
		c := &cli{t: t}
		code, stdout, _ := c.exec(arg)
		if code != exitOK {
			t.Fatalf("runctl %s = %d, want %d even with no mediator and no token", arg, code, exitOK)
		}
		if strings.TrimSpace(stdout) == "" {
			t.Fatalf("runctl %s printed nothing; a tool must be able to say what it is", arg)
		}
	}
}

func TestRun_IncompleteInvocationsPrintUsageAndAskForAFix(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"no subcommand at all", nil},
		{"a subcommand that needs an argument but has none", []string{"status"}},
		{"an unknown subcommand", []string{"teleport", "r1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newCLI(t, func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("the mediator was called for %q; nothing should leave the process", tc.args)
			})
			code, _, stderr := c.exec(tc.args...)
			if code != exitRequest {
				t.Fatalf("runctl %q = %d, want %d (the request itself is wrong)", tc.args, code, exitRequest)
			}
			if !strings.Contains(stderr, "runctl submit") {
				t.Fatalf("stderr = %q, want the usage text listing the subcommands", stderr)
			}
		})
	}
}

// 사용법은 토큰의 이름만 말하고 값은 말하지 않는다.
//
// 왜 이것을 재는가 - 사용법은 인자가 틀릴 때마다 나오고, 사람은 그것을
// 이슈에 붙이고 화면에 띄운다. 자격증명이 거기 한 줄이라도 실리면 그
// 순간부터 그 토큰은 공개된 것이다. 실제로 플래그 기본값에
// os.Getenv("ENODE_TOKEN") 이 들어 있었고 flag.PrintDefaults() 가 그것을
// (default "...") 로 그대로 찍었다.
//
// 재는 대상은 "환경변수를 안 읽는다" 가 아니라 "읽은 값을 안 찍는다" 다 -
// 환경변수로 준 토큰이 실제 요청에 실리는 것은
// TestRun_CapabilitiesSortsTheAttributeVocabulary 가, 플래그가 환경변수를
// 이기는 것은 TestRun_ExplicitFlagsBeatTheEnvironment 가 각각 잡고 있다.
func TestRun_UsageNamesTheTokenVariableWithoutPrintingItsValue(t *testing.T) {
	const secret = "usage-must-not-print-this-value"
	c := newCLI(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("the mediator was called for an incomplete invocation")
	})
	t.Setenv("ENODE_TOKEN", secret)

	// 인자가 없으면 사용법이 나온다. 이것이 사람이 가장 자주 보는 경로다.
	code, stdout, stderr := c.exec()
	if code != exitRequest {
		t.Fatalf("code = %d, want %d", code, exitRequest)
	}
	// 사용법에 플래그 목록이 실제로 실려 있어야 아래 단언이 의미를 갖는다.
	if !strings.Contains(stderr, "-token string") {
		t.Fatalf("stderr = %q, want the flag defaults section listing -token", stderr)
	}
	// 값을 가리는 대신 이름을 지우는 것은 고친 것이 아니다. 어디서 읽는지는
	// 계속 말해야 한다.
	if !strings.Contains(stderr, "$ENODE_TOKEN") {
		t.Fatalf("stderr = %q, want the usage to still name the variable it reads the token from", stderr)
	}
	if strings.Contains(stderr, secret) {
		t.Fatal("the usage text printed the value of $ENODE_TOKEN; it must name the variable and never its value")
	}
	if strings.Contains(stdout, secret) {
		t.Fatal("the value of $ENODE_TOKEN reached stdout; a credential must not be printed")
	}
}

func TestRun_RefusesToGuessTheMediatorOrTheToken(t *testing.T) {
	for _, tc := range []struct{ name, med, tok string }{
		{"no token", "http://127.0.0.1:1", ""},
		{"no mediator", "", testToken},
		{"neither", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ENODE_MEDIATOR", tc.med)
			t.Setenv("ENODE_TOKEN", tc.tok)
			c := &cli{t: t}
			code, _, stderr := c.exec("status", "run-1")
			if code != exitRequest {
				t.Fatalf("code = %d, want %d", code, exitRequest)
			}
			if !strings.Contains(stderr, "mediator and token are required") {
				t.Fatalf("stderr = %q, want it to name both missing inputs", stderr)
			}
		})
	}
}

func TestRun_DiesWhenGitCannotSayWhoIsAsking(t *testing.T) {
	// ADR-015 §1 - 조용한 대체는 한 사람에게 두 신원을 만든다. 그래서
	// 이메일이 없으면 그 자리에서 죽는다.
	c := newCLI(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("the mediator was called without a principal")
	})
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	code, _, stderr := c.exec("status", "run-1")
	if code != exitRequest {
		t.Fatalf("code = %d, want %d", code, exitRequest)
	}
	if !strings.Contains(stderr, "no git email") {
		t.Fatalf("stderr = %q, want it to say the email is missing and how to set it", stderr)
	}
}

// ── capabilities · asks (인자 없는 서브커맨드) ──────────────────────────

func TestRun_CapabilitiesSortsTheAttributeVocabulary(t *testing.T) {
	c := newCLI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/capabilities" {
			t.Errorf("path = %q, want /v1/capabilities", r.URL.Path)
		}
		writeJSON(t, w, 200, map[string]any{"capabilities": []map[string]any{{
			"capability": "build", "nodes": 2,
			"attrs": map[string][]string{"os": {"linux", "darwin"}, "arch": {"arm64"}},
		}}})
	})
	code, stdout, _ := c.exec("capabilities")
	if code != exitOK {
		t.Fatalf("code = %d, want %d", code, exitOK)
	}
	// 경계를 넘어간 것을 본다 - 토큰과 신원이 둘 다 실려야 한다.
	if c.auth != "Bearer "+testToken {
		t.Fatalf("Authorization = %q, want the bearer token", c.auth)
	}
	if c.principal != testEmail {
		t.Fatalf("X-Enode-Principal = %q, want %q", c.principal, testEmail)
	}
	if !strings.Contains(stdout, "build  nodes: 2") {
		t.Fatalf("stdout = %q, want the capability and its node count", stdout)
	}
	// 어휘는 광고에서 창발하므로 순서가 없다. 사람이 읽으려면 우리가 준다.
	iArch, iOS := strings.Index(stdout, "arch"), strings.Index(stdout, "os")
	if iArch < 0 || iOS < 0 || iArch > iOS {
		t.Fatalf("stdout = %q, want the attribute keys sorted (arch before os)", stdout)
	}
	if !strings.Contains(stdout, "linux, darwin") {
		t.Fatalf("stdout = %q, want the attribute values joined", stdout)
	}
	// nodes 는 총계이지 가용성이 아니다 - 그 오해를 출력이 직접 막는다.
	if !strings.Contains(stdout, "not availability") {
		t.Fatalf("stdout = %q, want the note that nodes is a total, not availability", stdout)
	}
}

func TestRun_AsksRendersTheInboxWithEnoughToAnswerIt(t *testing.T) {
	long := strings.Repeat("d", 400)
	c := newCLI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/asks" {
			t.Errorf("path = %q, want /v1/asks", r.URL.Path)
		}
		writeJSON(t, w, 200, map[string]any{"asks": []any{
			map[string]any{
				"run_id": "run-1", "seq": 3, "step": "approve", "prompt": "ship it?",
				"can_answer": true, "deadline": "2030-01-02T03:04:05Z",
				"shown": []any{
					map[string]any{"name": "diff", "content": long, "truncated": true},
					map[string]any{"name": "log", "content": "short", "truncated": false},
				},
				"proposes": map[string]any{"exit": 0},
				"schema": map[string]any{
					"required": []string{"kind"},
					"properties": map[string]any{
						"kind":  map[string]any{"enum": []string{"approve", "reject"}},
						"note":  map[string]any{"type": "string"},
						"extra": map[string]any{},
					},
				},
			},
			// 두 번째는 최소형이다 - 답할 수 없고, 기한도 산출물도 제안도
			// 없고, 스키마가 우리가 못 읽는 모양이다.
			map[string]any{
				"run_id": "run-2", "seq": 1, "step": "review", "prompt": "look?",
				"can_answer": false, "schema": "not an object",
			},
		}})
	})
	code, stdout, _ := c.exec("asks")
	if code != exitOK {
		t.Fatalf("code = %d, want %d", code, exitOK)
	}
	for _, want := range []string{"run-1 #3", "approve", "ship it?", "run-2 #1", "review"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout = %q, want %q in the inbox", stdout, want)
		}
	}
	if !strings.Contains(stdout, "(deadline ") {
		t.Fatalf("stdout = %q, want the deadline shown on the ask that has one", stdout)
	}
	// 300자를 넘으면 자른다 - 자른 사실이 보여야 한다.
	if strings.Contains(stdout, long) {
		t.Fatalf("stdout carried all 400 characters of the shown artifact; it must be cut at 300")
	}
	if !strings.Contains(stdout, "see record/blob for the full text") {
		t.Fatalf("stdout = %q, want the truncated artifact to say where the full text is", stdout)
	}
	if !strings.Contains(stdout, "log: ") || strings.Contains(stdout, "log (truncated") {
		t.Fatalf("stdout = %q, want the untruncated artifact shown without the truncation note", stdout)
	}
	if !strings.Contains(stdout, "proposed success criteria") {
		t.Fatalf("stdout = %q, want the proposed criteria - the person approves them", stdout)
	}
	// 스키마가 곧 질문의 형태다. 필수는 별표, enum 은 선택지, type 은 타입.
	if !strings.Contains(stdout, "* kind") {
		t.Fatalf("stdout = %q, want the required field starred", stdout)
	}
	if !strings.Contains(stdout, "approve | reject") {
		t.Fatalf("stdout = %q, want the enum rendered as choices", stdout)
	}
	if !strings.Contains(stdout, "  note       string") {
		t.Fatalf("stdout = %q, want the optional field unstarred with its type", stdout)
	}
	if !strings.Contains(stdout, "runctl answer <run-id> <seq>") {
		t.Fatalf("stdout = %q, want the closing line that says how to answer", stdout)
	}
}

func TestRun_AsksSaysSoWhenTheInboxIsEmpty(t *testing.T) {
	c := newCLI(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 200, map[string]any{"asks": []any{}})
	})
	code, stdout, _ := c.exec("asks")
	if code != exitOK {
		t.Fatalf("code = %d, want %d", code, exitOK)
	}
	if !strings.Contains(stdout, "no questions awaiting an answer") {
		t.Fatalf("stdout = %q, want an explicit empty-inbox line", stdout)
	}
}

func TestRun_CapabilitiesReportsAWireFault(t *testing.T) {
	c := newCLI(t, func(w http.ResponseWriter, r *http.Request) {
		fault(t, w, 503, "the fleet is draining")
	})
	code, _, stderr := c.exec("capabilities")
	if code != exitSystem {
		t.Fatalf("code = %d, want %d (5xx is the system's fault)", code, exitSystem)
	}
	if !strings.Contains(stderr, "503 the fleet is draining") {
		t.Fatalf("stderr = %q, want the wire reason kept", stderr)
	}
}

// ── answer ──────────────────────────────────────────────────────────────

func TestRun_AnswerRefusesTheMalformedInvocations(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"no sequence number", []string{"answer", "run-1"}, "runctl submit"},
		{"the sequence is not a number", []string{"answer", "run-1", "third"}, "invalid step sequence: third"},
		{"neither --set nor --json", []string{"answer", "run-1", "3"}, "answer is empty"},
		{"--json is not json", []string{"answer", "run-1", "3", "--json", "{oops"}, "cannot parse --json"},
		{"--set without an equals sign", []string{"answer", "run-1", "3", "--set", "kind"}, "--set expects field=value: kind"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newCLI(t, func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("an answer left the process for %q; it should have been refused locally", tc.args)
			})
			code, _, stderr := c.exec(tc.args...)
			if code != exitRequest {
				t.Fatalf("code = %d, want %d", code, exitRequest)
			}
			if !strings.Contains(stderr, tc.want) {
				t.Fatalf("stderr = %q, want %q", stderr, tc.want)
			}
		})
	}
}

func TestRun_AnswerAssemblesTheBodyFromJSONAndSet(t *testing.T) {
	var got map[string]any
	c := newCLI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/runs/run-1/steps/3/answer" {
			t.Errorf("path = %q, want the run and the sequence in it", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decoding the answer body: %v", err)
		}
		writeJSON(t, w, 200, map[string]any{"run_id": "run-1", "state": "RUNNING"})
	})
	code, stdout, _ := c.exec("answer", "run-1", "3",
		"--json", `{"kind":"approve","score":7}`, "--set", "note=looks right", "--set", "score=9")
	if code != exitOK {
		t.Fatalf("code = %d, want %d", code, exitOK)
	}
	if got["kind"] != "approve" {
		t.Fatalf("body kind = %v, want approve (from --json)", got["kind"])
	}
	if got["note"] != "looks right" {
		t.Fatalf("body note = %v, want the --set value", got["note"])
	}
	// --set 은 문자열 필드이고 --json 뒤에 얹힌다 - 같은 키면 --set 이 이긴다.
	if got["score"] != "9" {
		t.Fatalf("body score = %#v, want the string \"9\"; --set is applied after --json and its values are strings", got["score"])
	}
	if !strings.Contains(stdout, "answered  run-1 #3") {
		t.Fatalf("stdout = %q, want the acknowledgement naming the run and the step", stdout)
	}
	if !strings.Contains(stdout, "run is now RUNNING") {
		t.Fatalf("stdout = %q, want the new run state when the mediator reports one", stdout)
	}
}

func TestRun_AnswerStaysQuietAboutTheStateWhenThereIsNone(t *testing.T) {
	c := newCLI(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 200, map[string]any{"run_id": "run-1"})
	})
	code, stdout, _ := c.exec("answer", "run-1", "3", "--set", "kind=approve")
	if code != exitOK {
		t.Fatalf("code = %d, want %d", code, exitOK)
	}
	if strings.Contains(stdout, "run is now") {
		t.Fatalf("stdout = %q, want no state line when the mediator reported no state", stdout)
	}
}

// ── submit · dry-run ────────────────────────────────────────────────────

func TestRun_SubmitReadsTheContractFromDisk(t *testing.T) {
	c := newCLI(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("a missing contract file must never reach the mediator")
	})
	code, _, stderr := c.exec("submit", filepath.Join(t.TempDir(), "absent.json"))
	if code != exitRequest {
		t.Fatalf("code = %d, want %d", code, exitRequest)
	}
	if !strings.Contains(stderr, "absent.json") {
		t.Fatalf("stderr = %q, want the path that could not be read", stderr)
	}
}

func TestRun_DryRunAllocatesNothingAndReturnsAtOnce(t *testing.T) {
	var paths []string
	c := newCLI(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		writeJSON(t, w, 200, map[string]any{"run_id": "dry-1", "state": "MATCHED"})
	})
	code, stdout, _ := c.exec("dry-run", contractFile(t), "--wait")
	if code != exitOK {
		t.Fatalf("code = %d, want %d", code, exitOK)
	}
	if len(paths) != 1 || paths[0] != "/v1/runs/dry-run" {
		t.Fatalf("paths = %q, want exactly one call to /v1/runs/dry-run - --wait must not poll a dry run", paths)
	}
	if !strings.Contains(stdout, "dry-1  MATCHED") {
		t.Fatalf("stdout = %q, want the matched result printed", stdout)
	}
}

func TestRun_SubmitWithoutWaitReturnsAsSoonAsItIsAccepted(t *testing.T) {
	var n atomic.Int32
	c := newCLI(t, func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		writeJSON(t, w, 200, map[string]any{"run_id": "run-3", "state": "PENDING"})
	})
	code, stdout, _ := c.exec("submit", contractFile(t))
	if code != exitOK {
		t.Fatalf("code = %d, want %d - runctl is stateless; it submits and forgets", code, exitOK)
	}
	if n.Load() != 1 {
		t.Fatalf("the mediator was called %d times, want 1 without --wait", n.Load())
	}
	if !strings.Contains(stdout, "run-3  PENDING") {
		t.Fatalf("stdout = %q, want the accepted run printed", stdout)
	}
}

func TestRun_SubmitHonoursAFlagWrittenAfterThePositional(t *testing.T) {
	// 생산 주석이 이유를 적는다: 표준 flag 는 첫 위치인자에서 파싱을 멈추고,
	// 그래서 `runctl submit x.json --wait` 의 --wait 가 조용히 무시됐다.
	// 조용한 무시가 가장 나쁘다. permute 가 그것을 막는지는 여기서만
	// 끝까지 확인된다 - 플래그 목록을 베끼지 않고 실제 목록으로 본다.
	var polls atomic.Int32
	c := newCLI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			writeJSON(t, w, 200, map[string]any{"run_id": "run-4", "state": "RUNNING"})
			return
		}
		polls.Add(1)
		writeJSON(t, w, 200, map[string]any{"run_id": "run-4", "state": "SUCCEEDED"})
	})
	code, stdout, _ := c.exec("submit", contractFile(t), "--wait", "--poll", "10ms")
	if code != exitOK {
		t.Fatalf("code = %d, want %d", code, exitOK)
	}
	if polls.Load() == 0 {
		t.Fatalf("the run was never polled; --wait written after the positional was dropped")
	}
	if strings.Count(stdout, "run-4") < 2 {
		t.Fatalf("stdout = %q, want the run printed on submission and again when it settles", stdout)
	}
	if !strings.Contains(stdout, "SUCCEEDED") {
		t.Fatalf("stdout = %q, want the terminal state", stdout)
	}
}

func TestRun_SubmitAndWaitCarriesTheRunFailureIntoTheExitCode(t *testing.T) {
	c := newCLI(t, func(w http.ResponseWriter, r *http.Request) {
		state := "FAILED"
		if r.Method == http.MethodPost {
			state = "RUNNING"
		}
		writeJSON(t, w, 200, map[string]any{"run_id": "run-5", "state": state})
	})
	code, _, _ := c.exec("submit", contractFile(t), "--wait", "--poll", "10ms")
	if code != exitRunFail {
		t.Fatalf("code = %d, want %d - the request was valid, the work failed", code, exitRunFail)
	}
}

func TestRun_SubmitReportsARejectedContract(t *testing.T) {
	c := newCLI(t, func(w http.ResponseWriter, r *http.Request) {
		fault(t, w, 422, "no node advertises arch=riscv64")
	})
	code, _, stderr := c.exec("submit", contractFile(t))
	if code != exitRequest {
		t.Fatalf("code = %d, want %d - fix the contract", code, exitRequest)
	}
	if !strings.Contains(stderr, "422 no node advertises arch=riscv64") {
		t.Fatalf("stderr = %q, want the mediator's reason", stderr)
	}
}

func TestRun_SubmitAndWaitReportsAFaultRaisedWhilePolling(t *testing.T) {
	c := newCLI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			writeJSON(t, w, 200, map[string]any{"run_id": "run-6", "state": "RUNNING"})
			return
		}
		fault(t, w, 500, "the store is unreachable")
	})
	code, _, stderr := c.exec("submit", contractFile(t), "--wait", "--poll", "10ms")
	if code != exitSystem {
		t.Fatalf("code = %d, want %d", code, exitSystem)
	}
	if !strings.Contains(stderr, "500 the store is unreachable") {
		t.Fatalf("stderr = %q, want the fault raised during the wait", stderr)
	}
}

// ── status · cancel ─────────────────────────────────────────────────────

func TestRun_StatusSeparatesUnfinishedFromFailed(t *testing.T) {
	for _, tc := range []struct {
		state string
		want  int
	}{
		{"RUNNING", exitOK}, // 아직 안 끝났다 - 실패가 아니다
		{"PENDING", exitOK},
		{"SUCCEEDED", exitOK},
		{"FAILED", exitRunFail},
	} {
		t.Run(tc.state, func(t *testing.T) {
			c := newCLI(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/runs/run-1" {
					t.Errorf("path = %q, want /v1/runs/run-1", r.URL.Path)
				}
				writeJSON(t, w, 200, map[string]any{"run_id": "run-1", "state": tc.state})
			})
			code, stdout, _ := c.exec("status", "run-1")
			if code != tc.want {
				t.Fatalf("status of a %s run = %d, want %d", tc.state, code, tc.want)
			}
			if !strings.Contains(stdout, "run-1  "+tc.state) {
				t.Fatalf("stdout = %q, want the run and its state", stdout)
			}
		})
	}
}

func TestRun_StatusReportsAnUnknownRun(t *testing.T) {
	c := newCLI(t, func(w http.ResponseWriter, r *http.Request) {
		fault(t, w, 404, "no such run")
	})
	code, _, stderr := c.exec("status", "run-nope")
	if code != exitRequest {
		t.Fatalf("code = %d, want %d", code, exitRequest)
	}
	if !strings.Contains(stderr, "404 no such run") {
		t.Fatalf("stderr = %q, want the reason", stderr)
	}
}

func TestRun_CancelSucceedsBecauseItWasAskedFor(t *testing.T) {
	c := newCLI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/runs/run-1/cancel" {
			t.Errorf("path = %q, want the cancel endpoint", r.URL.Path)
		}
		writeJSON(t, w, 200, map[string]any{"run_id": "run-1", "state": "FAILED"})
	})
	code, stdout, _ := c.exec("cancel", "run-1")
	// 취소는 요청한 대로 된 것이다 - Run 이 FAILED 로 끝나도 0 이다.
	if code != exitOK {
		t.Fatalf("code = %d, want %d; a cancellation did what it was asked to do", code, exitOK)
	}
	if !strings.Contains(stdout, "run-1  FAILED") {
		t.Fatalf("stdout = %q, want the resulting state printed", stdout)
	}
}

// ── record ──────────────────────────────────────────────────────────────

func TestRun_RecordGoesToStdoutByDefault(t *testing.T) {
	c := newCLI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/runs/run-1/record" {
			t.Errorf("path = %q, want the record endpoint", r.URL.Path)
		}
		if _, err := w.Write([]byte("sealed-record-bytes")); err != nil {
			t.Errorf("writing the fake record: %v", err)
		}
	})
	code, stdout, _ := c.exec("record", "run-1")
	if code != exitOK {
		t.Fatalf("code = %d, want %d", code, exitOK)
	}
	if stdout != "sealed-record-bytes" {
		t.Fatalf("stdout = %q, want the record bytes verbatim", stdout)
	}
}

func TestRun_RecordWritesTheFileNamedByDashO(t *testing.T) {
	c := newCLI(t, func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("sealed-record-bytes")); err != nil {
			t.Errorf("writing the fake record: %v", err)
		}
	})
	out := filepath.Join(t.TempDir(), "out.tar")
	code, stdout, _ := c.exec("record", "run-1", "-o", out)
	if code != exitOK {
		t.Fatalf("code = %d, want %d", code, exitOK)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want nothing on stdout when -o names a file", stdout)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("reading %s: %v", out, err)
	}
	if string(b) != "sealed-record-bytes" {
		t.Fatalf("%s = %q, want the record bytes", out, b)
	}
}

func TestRun_RecordFailsLoudlyWhenItCannotCreateTheFile(t *testing.T) {
	c := newCLI(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("the record was fetched even though the destination could not be created")
	})
	out := filepath.Join(t.TempDir(), "no-such-dir", "out.tar")
	code, _, stderr := c.exec("record", "run-1", "-o", out)
	if code != exitSystem {
		t.Fatalf("code = %d, want %d", code, exitSystem)
	}
	if !strings.Contains(stderr, "out.tar") {
		t.Fatalf("stderr = %q, want the path it could not create", stderr)
	}
}

func TestRun_RecordReportsAWireFault(t *testing.T) {
	c := newCLI(t, func(w http.ResponseWriter, r *http.Request) {
		fault(t, w, 409, "the run is not sealed yet")
	})
	code, _, stderr := c.exec("record", "run-1")
	if code != exitRequest {
		t.Fatalf("code = %d, want %d - try again later", code, exitRequest)
	}
	if !strings.Contains(stderr, "409 the run is not sealed yet") {
		t.Fatalf("stderr = %q, want the reason", stderr)
	}
}

// ── 닿지 않는 Mediator ──────────────────────────────────────────────────

func TestRun_UnreachableMediatorIsASystemFailure(t *testing.T) {
	c := newCLI(t, func(w http.ResponseWriter, r *http.Request) {})
	// 세워 두고 바로 닫는다 - 주소는 살아 있고 아무도 안 받는다.
	dead := c.srv.URL
	c.srv.Close()
	t.Setenv("ENODE_MEDIATOR", dead)
	code, _, stderr := c.exec("status", "run-1")
	if code != exitSystem {
		t.Fatalf("code = %d, want %d (cannot reach the mediator)", code, exitSystem)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Fatalf("stderr was empty; a transport failure must say what happened")
	}
}

// ── 플래그로 준 값이 환경변수를 이긴다 ──────────────────────────────────

func TestRun_ExplicitFlagsBeatTheEnvironment(t *testing.T) {
	c := newCLI(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 200, map[string]any{"run_id": "run-1", "state": "SUCCEEDED"})
	})
	url := c.srv.URL
	t.Setenv("ENODE_MEDIATOR", "http://127.0.0.1:1")
	t.Setenv("ENODE_TOKEN", "wrong-token")
	code, _, _ := c.exec("status", "run-1", "--mediator", url, "--token", "flag-token")
	if code != exitOK {
		t.Fatalf("code = %d, want %d; --mediator and --token must win over the environment", code, exitOK)
	}
	if c.auth != "Bearer flag-token" {
		t.Fatalf("Authorization = %q, want the token given on the command line", c.auth)
	}
}

// ── 나머지 서브커맨드의 오류 경로 ───────────────────────────────────────

// 모든 서브커맨드가 같은 자리에서 같은 방식으로 오류를 종료코드로 옮긴다.
// 하나만 보면 그 대응이 서브커맨드마다 다르게 새는 것을 못 잡는다.
func TestRun_EverySubcommandCarriesTheWireFaultOut(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		code int
		want int
	}{
		{"asks cannot be listed", []string{"asks"}, 500, exitSystem},
		{"the answer is refused", []string{"answer", "run-1", "3", "--set", "kind=approve"}, 409, exitRequest},
		{"the run cannot be cancelled", []string{"cancel", "run-1"}, 409, exitRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newCLI(t, func(w http.ResponseWriter, r *http.Request) {
				fault(t, w, tc.code, "refused for the test")
			})
			code, stdout, stderr := c.exec(tc.args...)
			if code != tc.want {
				t.Fatalf("runctl %q against a %d = %d, want %d", tc.args, tc.code, code, tc.want)
			}
			if !strings.Contains(stderr, "refused for the test") {
				t.Fatalf("stderr = %q, want the mediator's reason", stderr)
			}
			if strings.TrimSpace(stdout) != "" {
				t.Fatalf("stdout = %q, want nothing printed when the call failed", stdout)
			}
		})
	}
}
