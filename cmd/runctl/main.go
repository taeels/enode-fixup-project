// Command runctl 은 사람과 껍데기가 Run 을 다루는 CLI 다.
//
//	runctl submit  <contract.json> [--wait]
//	runctl dry-run <contract.json>
//	runctl status  <run-id>
//	runctl record  <run-id> [-o out.tar]
//	runctl cancel  <run-id>
//
// runctl 은 무상태다 — 제출하고 잊는다. 죽어도 Run 은 계속 돈다.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/taeels/enode/internal/build"
	"github.com/taeels/enode/internal/runctl"
)

// CLI 종료코드는 0~3 이다 (INVARIANTS §4 「에러 코드 체계」)
//
// 와이어는 HTTP 를 쓰고 CLI 는 이 넷을 쓴다 — 경계가 둘이라 하나로 통일하지 않는다.
// 셸 스크립트가 필요로 하는 구분은 "일이 실패했나 / 내 요청이 틀렸나 / 시스템이 죽었나" 다.
const (
	exitOK      = 0 // Run 이 SUCCEEDED
	exitRunFail = 1 // Run 이 FAILED — 요청은 정상이었다
	exitRequest = 2 // 요청이 거절됐다 (4xx) — 계약을 고치거나 나중에 다시
	exitSystem  = 3 // Mediator 에 못 닿았다 또는 내부 오류
)

func main() { os.Exit(run()) }

// permute 는 플래그를 앞으로 모은다. 값을 받는 플래그의 다음 인자도 함께 옮긴다.
func permute(args []string) []string {
	takesValue := map[string]bool{}
	flag.VisitAll(func(f *flag.Flag) {
		if f.DefValue != "true" && f.DefValue != "false" {
			takesValue["-"+f.Name] = true
			takesValue["--"+f.Name] = true
		}
	})
	var flags, rest []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			rest = append(rest, args[i+1:]...)
			i = len(args)
		case strings.HasPrefix(a, "-") && len(a) > 1:
			flags = append(flags, a)
			// -o out.tar 처럼 값이 떨어져 있으면 같이 옮긴다 (-o=out.tar 는 붙어 있다)
			if takesValue[a] && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				flags = append(flags, args[i])
			}
		default:
			rest = append(rest, a)
		}
	}
	return append(flags, rest...)
}

func run() int {
	// --version 은 플래그 파싱보다 앞이다 (ADR-056) — 토큰이 없어도 답한다.
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-version") {
		fmt.Println(build.Version("runctl"))
		return 0
	}
	base := flag.String("mediator", os.Getenv("ENODE_MEDIATOR"), "mediator address ($ENODE_MEDIATOR)")
	token := flag.String("token", os.Getenv("ENODE_TOKEN"), "auth token ($ENODE_TOKEN)")
	wait := flag.Bool("wait", false, "wait until the run reaches a terminal state")
	out := flag.String("o", "", "file to write the record to (default: stdout)")
	every := flag.Duration("poll", 2*time.Second, "polling interval for --wait")
	answerJSON := flag.String("json", "", "full answer body (JSON)")
	var sets stringList
	flag.Var(&sets, "set", "answer field=value (repeatable; values are strings)")
	flag.Usage = usage
	// 표준 flag 는 첫 위치인자에서 파싱을 멈춘다
	// 그런데 사람은 `runctl submit x.json --wait` 라고 쓴다. 그 순서를 안 받으면
	// 플래그가 조용히 무시되고, 조용한 무시가 가장 나쁘다.
	flag.CommandLine.Parse(permute(os.Args[1:]))

	// 함대를 안 거치는 셋은 여기서 끝낸다 (ADR-066 §4).
	//
	// Mediator 주소도 토큰도 git 신원도 요구하지 않는다 — 그것을 요구하면
	// 「계약을 어떻게 쓰나」를 물으려고 먼저 함대를 세워야 하고, 그러면
	// 이 셋이 있어야 하는 이유가 사라진다. 처음 오는 쪽이 가장 먼저 잡는
	// 것이 이 셋이다.
	switch flag.Arg(0) {
	case "example":
		return cmdExample(flag.Arg(1))
	case "schema":
		return cmdSchema(flag.Arg(1))
	case "lint":
		return cmdLint(flag.Arg(1))
	}

	// capabilities 는 인자가 없다.
	if flag.NArg() == 1 && (flag.Arg(0) == "capabilities" || flag.Arg(0) == "asks") {
		flag.CommandLine.Parse(append(permute(os.Args[1:]), "-"))
	} else if flag.NArg() < 2 {
		usage()
		return exitRequest
	}
	if *base == "" || *token == "" {
		fmt.Fprintln(os.Stderr, "mediator and token are required (--mediator/--token or environment)")
		return exitRequest
	}

	// 신원은 git config 에서 읽는다 — 사람이 두 곳에 같은 사실을 적지 않는다.
	// 없으면 그 자리에서 죽는다 (ADR-015 §1).
	principal, err := runctl.Principal()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitRequest
	}

	c := &runctl.Client{Base: *base, Token: *token, Principal: principal,
		HTTP: &http.Client{Timeout: 60 * time.Second}}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cmd, arg := flag.Arg(0), ""
	if flag.NArg() > 1 {
		arg = flag.Arg(1)
	}
	switch cmd {
	case "capabilities":
		caps, err := c.Capabilities(ctx)
		if code := report(err); code != 0 {
			return code
		}
		for _, cp := range caps {
			fmt.Printf("%s  nodes: %d\n", cp.Capability, cp.Nodes)
			keys := make([]string, 0, len(cp.Attrs))
			for k := range cp.Attrs {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Printf("  %-10s %s\n", k, strings.Join(cp.Attrs[k], ", "))
			}
		}
		fmt.Println("\nnote: nodes is a total count, not availability.")
		fmt.Println("  use dry-run to check a specific attribute combination.")
		return exitOK

	case "asks":
		asks, err := c.Asks(ctx)
		if code := report(err); code != 0 {
			return code
		}
		if len(asks) == 0 {
			fmt.Println("no questions awaiting an answer")
			return exitOK
		}
		for _, a := range asks {
			mark := " "
			if a.CanAnswer {
				mark = "" // 내가 답할 수 있는 것
			}
			line := fmt.Sprintf("%s %s #%d %-14s %s", mark, a.RunID, a.Seq, a.Step, a.Prompt)
			if a.Deadline != nil {
				line += fmt.Sprintf("  (deadline %s)", a.Deadline.Local().Format("01-02 15:04"))
			}
			fmt.Println(line)
			// 질문과 함께 볼 것 (ask.show) — 보지 않고 답하게 만들지 않는다.
			for _, sh := range a.Shown {
				c := string(sh.Content)
				if len(c) > 300 {
					c = c[:300] + "…"
				}
				mark2 := ""
				if sh.Truncated {
					mark2 = " (truncated; see record/blob for the full text)"
				}
				fmt.Printf("    ┆ %s%s: %s\n", sh.Name, mark2, c) // 괘선이라 대상이 아님 - 세로줄로 묶는다
			}
			// 제안된 판정 기준 (adopts) — 무엇을 승인하는지 보여준다.
			if len(a.Proposes) > 0 {
				pb, _ := json.Marshal(a.Proposes)
				fmt.Printf("    | proposed success criteria: %s\n", pb)
			}
			// 스키마가 곧 질문의 형태다 — 무엇을 적어야 하는지 보여준다.
			var form struct {
				Required   []string                          `json:"required"`
				Properties map[string]map[string]interface{} `json:"properties"`
			}
			if json.Unmarshal(a.Schema, &form) == nil {
				req := map[string]bool{}
				for _, r := range form.Required {
					req[r] = true
				}
				keys := make([]string, 0, len(form.Properties))
				for k := range form.Properties {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					p := form.Properties[k]
					desc := ""
					if e, ok := p["enum"].([]interface{}); ok {
						opts := make([]string, 0, len(e))
						for _, o := range e {
							opts = append(opts, fmt.Sprint(o))
						}
						desc = strings.Join(opts, " | ")
					} else if t, ok := p["type"].(string); ok {
						desc = t
					}
					star := " "
					if req[k] {
						star = "*"
					}
					fmt.Printf("    %s %-10s %s"+"\n", star, k, desc)
				}
			}
		}
		fmt.Println()
		fmt.Println("to answer: runctl answer <run-id> <seq> --set field=value")
		return exitOK

	case "answer":
		if flag.NArg() < 3 {
			usage()
			return exitRequest
		}
		seq, err := strconv.Atoi(flag.Arg(2))
		if err != nil {
			fmt.Fprintln(os.Stderr, "invalid step sequence:", flag.Arg(2))
			return exitRequest
		}
		// 답을 조립한다 — --json 이 통짜, --set k=v 가 문자열 필드.
		answer := map[string]any{}
		if *answerJSON != "" {
			if err := json.Unmarshal([]byte(*answerJSON), &answer); err != nil {
				fmt.Fprintln(os.Stderr, "cannot parse --json:", err)
				return exitRequest
			}
		}
		for _, kv := range sets {
			k, v, ok := strings.Cut(kv, "=")
			if !ok {
				fmt.Fprintln(os.Stderr, "--set expects field=value:", kv)
				return exitRequest
			}
			answer[k] = v
		}
		if len(answer) == 0 {
			fmt.Fprintln(os.Stderr, "answer is empty; use --set or --json")
			return exitRequest
		}
		body, _ := json.Marshal(answer)
		r, err := c.Answer(ctx, arg, seq, body)
		if code := report(err); code != 0 {
			return code
		}
		fmt.Printf("answered  %s #%d", arg, seq)
		if r.State != "" {
			fmt.Printf("  run is now %s", r.State)
		}
		fmt.Println()
		return exitOK

	case "submit", "dry-run":
		body, err := os.ReadFile(arg)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return exitRequest
		}
		r, err := c.Submit(ctx, body, cmd == "dry-run")
		if code := report(err); code != 0 {
			return code
		}
		printRun(r)
		if cmd == "dry-run" || !*wait {
			return exitOK
		}
		r, err = c.Wait(ctx, r.RunID, *every)
		if code := report(err); code != 0 {
			return code
		}
		printRun(r)
		return verdictCode(r)

	case "status":
		r, err := c.Status(ctx, arg)
		if code := report(err); code != 0 {
			return code
		}
		printRun(r)
		if !runctl.Terminal(r.State) {
			return exitOK // 아직 안 끝났다 — 실패가 아니다
		}
		return verdictCode(r)

	case "cancel":
		r, err := c.Cancel(ctx, arg)
		if code := report(err); code != 0 {
			return code
		}
		printRun(r)
		return exitOK // 취소는 요청한 대로 된 것이다

	case "record":
		w := os.Stdout
		if *out != "" {
			f, err := os.Create(*out)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return exitSystem
			}
			defer f.Close()
			w = f
		}
		if code := report(c.Record(ctx, arg, w)); code != 0 {
			return code
		}
		return exitOK

	default:
		usage()
		return exitRequest
	}
}

// report 는 와이어 오류를 CLI 종료코드로 번역한다.
func report(err error) int {
	if err == nil {
		return 0
	}
	var f *runctl.Fail
	if errors.As(err, &f) {
		fmt.Fprintf(os.Stderr, "%d %s\n", f.Code, f.Reason)
		// 무엇이 틀렸는지만 말하고 무엇이 맞는지는 안 말하면 왕복이 하나 는다
		// (ADR-066 §4.4). 답을 들고 있는 명령이 바로 옆에 있다.
		if next := nextStep(f.Reason); next != "" {
			fmt.Fprintln(os.Stderr, "  "+next)
		}
		if f.Code/100 == 4 {
			return exitRequest // 계약이 틀렸거나(400/422) 지금은 안 된다(409)
		}
		return exitSystem
	}
	fmt.Fprintln(os.Stderr, err)
	return exitSystem
}

// nextStep 은 거절 사유에 맞는 다음 행동 한 줄이다 (ADR-066 §4.4).
//
// 사유 문자열로 가른다 — Mediator 가 구조화된 코드를 주지 않기 때문이다.
// 못 맞히면 빈 문자열이고, 그때는 오늘과 같다. 틀린 안내를 하느니 안 한다.
func nextStep(reason string) string {
	switch {
	case strings.Contains(reason, "unknown capability"):
		return "see: runctl capabilities"
	case strings.Contains(reason, "no node"), strings.Contains(reason, "unsatisfied"),
		strings.Contains(reason, "no match"):
		return "see: runctl capabilities   (매칭되는 속성 조합이 함대에 없다)"
	case strings.Contains(reason, "duplicate"), strings.Contains(reason, "exists"):
		return "run_id 는 함대에서 유일해야 한다"
	case strings.Contains(reason, "busy"), strings.Contains(reason, "capacity"):
		return "함대가 지금 바쁘다. 잠시 뒤 다시 낸다"
	}
	// 그 밖의 계약 오류는 형식 문제일 때가 많다.
	if strings.Contains(reason, "step") || strings.Contains(reason, "contract") ||
		strings.Contains(reason, "required") {
		return "see: runctl lint <contract.json>  ·  runctl example"
	}
	return ""
}

// verdictCode 는 Run 의 성패를 종료코드로 옮긴다.
// 요청이 정상이었는데 일이 실패한 것이므로 2 가 아니라 1 이다.
func verdictCode(r *runctl.Run) int {
	if r.State == "SUCCEEDED" {
		return exitOK
	}
	return exitRunFail
}

func printRun(r *runctl.Run) {
	fmt.Printf("%s  %s\n", r.RunID, r.State)
	// 경고를 먼저 찍는다 (ADR-061 §2) — 제출은 됐지만 뜻대로 안 도는 것이
	// 있으면 그것부터 보여야 한다. 아래 줄들에 묻히면 못 읽는다.
	for _, wmsg := range r.Warnings {
		fmt.Printf("  warning: %s\n", wmsg)
	}
	for _, a := range r.Assigned {
		for _, n := range a.Nodes {
			fmt.Printf("  %-10s %s  %s\n", a.As, n.Node, n.Label)
		}
	}
	// 폭이 1 을 넘으면 Run 상태 한 줄로는 안 보인다 (ADR-025) —
	// 어느 가지가 어디까지 갔고 지금 도는 것이 어느 기계인지를 여기서 읽는다.
	for _, st := range r.Steps {
		line := fmt.Sprintf("  %2d %-16s %-8s %s", st.Seq, st.ID, st.State, st.Node)
		if st.Attempt > 0 {
			line += fmt.Sprintf("  (attempt %d)", st.Attempt+1)
		}
		// 기다리는 중이면 무엇을 기다리는지를 같이 보여준다.
		if st.State == "PENDING" && len(st.Needs) > 0 {
			line += "  waits on " + strings.Join(st.Needs, " · ")
		}
		fmt.Println(line)
	}
	if len(r.Verdict) > 0 {
		var v struct {
			State  string `json:"state"`
			Checks []struct {
				Step string `json:"step"`
				What string `json:"what"`
				OK   bool   `json:"ok"`
				Note string `json:"note"`
			} `json:"checks"`
		}
		if json.Unmarshal(r.Verdict, &v) == nil {
			for _, ch := range v.Checks {
				mark := "ok"
				if !ch.OK {
					mark = "bad"
				}
				fmt.Printf("  %-3s %s %s %s\n", mark, ch.Step, ch.What, ch.Note)
			}
		}
	}
}

// stringList 는 반복 가능한 --set 을 받는다.
type stringList []string

func (l *stringList) String() string     { return strings.Join(*l, ",") }
func (l *stringList) Set(v string) error { *l = append(*l, v); return nil }

func usage() {
	fmt.Fprint(os.Stderr, `runctl - submit, watch and fetch runs

  runctl submit  <contract.json> [--wait]
  runctl dry-run <contract.json>          match only; allocates nothing
  runctl status  <run-id>
  runctl record  <run-id> [-o out.tar]    fetch the sealed run record
  runctl cancel  <run-id>
  runctl capabilities                     attribute vocabulary of the fleet
  runctl asks                             questions awaiting an answer
  runctl answer <run-id> <seq> --set k=v [--set ...]   answer a question

Writing a contract           these three need no mediator and no token
  runctl example                          list the ready-to-run examples
  runctl example <name>                   print one; it is valid as-is
  runctl lint <contract.json>             check it before you submit it
  runctl schema [section]                 the field vocabulary

Exit codes
  0  run succeeded (or is still running)
  1  run failed; the request itself was valid
  2  request rejected; fix the contract (400/422) or retry later (409)
  3  cannot reach the mediator

Options
`)
	flag.PrintDefaults()
}
