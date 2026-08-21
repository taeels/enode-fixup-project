// Command runctl 은 사람과 껍데기가 Run 을 다루는 CLI 다.
//
//	runctl submit  <contract.json> [--wait]
//	runctl dry-run <contract.json>
//	runctl status  <run-id>
//	runctl record  <run-id> [-o out.tar]
//	runctl cancel  <run-id>
//
// ★ runctl 은 무상태다 ★ — 제출하고 잊는다. 죽어도 Run 은 계속 돈다.
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

	"github.com/taeels/enode/internal/runctl"
)

// ★ CLI 종료코드는 0~3 이다 ★ (INVARIANTS §4 「에러 코드 체계」)
//
// 와이어는 HTTP 를 쓰고 CLI 는 이 넷을 쓴다 — 경계가 둘이라 하나로 통일하지 않는다.
// 셸 스크립트가 필요로 하는 구분은 "일이 실패했나 / 내 요청이 틀렸나 / 시스템이 죽었나" 다.
const (
	exitOK      = 0 // Run 이 SUCCEEDED
	exitRunFail = 1 // ★ Run 이 FAILED ★ — 요청은 정상이었다
	exitRequest = 2 // ★ 요청이 거절됐다 ★ (4xx) — 계약을 고치거나 나중에 다시
	exitSystem  = 3 // ★ Mediator 에 못 닿았다 ★ 또는 내부 오류
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
	base := flag.String("mediator", os.Getenv("ENODE_MEDIATOR"), "Mediator 주소 ($ENODE_MEDIATOR)")
	token := flag.String("token", os.Getenv("ENODE_TOKEN"), "토큰 ($ENODE_TOKEN)")
	wait := flag.Bool("wait", false, "종료 상태가 될 때까지 기다린다")
	out := flag.String("o", "", "record 를 쓸 파일 (기본: 표준출력)")
	every := flag.Duration("poll", 2*time.Second, "--wait 의 폴링 주기")
	answerJSON := flag.String("json", "", "answer 의 본문 전체 (JSON)")
	var sets stringList
	flag.Var(&sets, "set", "answer 의 필드=값 (반복 가능. 값은 문자열)")
	flag.Usage = usage
	// ★ 표준 flag 는 첫 위치인자에서 파싱을 멈춘다 ★
	// 그런데 사람은 `runctl submit x.json --wait` 라고 쓴다. 그 순서를 안 받으면
	// 플래그가 조용히 무시되고, ★ 조용한 무시가 가장 나쁘다 ★.
	flag.CommandLine.Parse(permute(os.Args[1:]))

	// capabilities 는 인자가 없다.
	if flag.NArg() == 1 && (flag.Arg(0) == "capabilities" || flag.Arg(0) == "asks") {
		flag.CommandLine.Parse(append(permute(os.Args[1:]), "-"))
	} else if flag.NArg() < 2 {
		usage()
		return exitRequest
	}
	if *base == "" || *token == "" {
		fmt.Fprintln(os.Stderr, "mediator 와 token 이 필요하다 (--mediator/--token 또는 환경변수)")
		return exitRequest
	}

	// ★ 신원은 git config 에서 읽는다 ★ — 사람이 두 곳에 같은 사실을 적지 않는다.
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
		fmt.Println("\n※ nodes 는 총수(존재)다. 지금 비어 있는지는 알려주지 않는다 —")
		fmt.Println("  속성 조합으로 세려면 dry-run 을 쓴다.")
		return exitOK

	case "asks":
		asks, err := c.Asks(ctx)
		if code := report(err); code != 0 {
			return code
		}
		if len(asks) == 0 {
			fmt.Println("답을 기다리는 질문이 없다")
			return exitOK
		}
		for _, a := range asks {
			mark := " "
			if a.CanAnswer {
				mark = "★" // 내가 답할 수 있는 것
			}
			line := fmt.Sprintf("%s %s #%d %-14s %s", mark, a.RunID, a.Seq, a.Step, a.Prompt)
			if a.Deadline != nil {
				line += fmt.Sprintf("  (기한 %s)", a.Deadline.Local().Format("01-02 15:04"))
			}
			fmt.Println(line)
			// ★ 질문과 함께 볼 것 ★ (ask.show) — 보지 않고 답하게 만들지 않는다.
			for _, sh := range a.Shown {
				c := string(sh.Content)
				if len(c) > 300 {
					c = c[:300] + "…"
				}
				mark2 := ""
				if sh.Truncated {
					mark2 = " (잘림 — 전문은 record/blob 으로)"
				}
				fmt.Printf("    ┆ %s%s: %s\n", sh.Name, mark2, c)
			}
			// ★ 제안된 판정 기준 ★ (adopts) — 무엇을 승인하는지 보여준다.
			if len(a.Proposes) > 0 {
				pb, _ := json.Marshal(a.Proposes)
				fmt.Printf("    ┆ 제안된 판정 기준: %s\n", pb)
			}
			// ★ 스키마가 곧 질문의 형태다 ★ — 무엇을 적어야 하는지 보여준다.
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
		fmt.Println("답하기:  runctl answer <run-id> <seq> --set 필드=값")
		return exitOK

	case "answer":
		if flag.NArg() < 3 {
			usage()
			return exitRequest
		}
		seq, err := strconv.Atoi(flag.Arg(2))
		if err != nil {
			fmt.Fprintln(os.Stderr, "단계 순번이 이상하다:", flag.Arg(2))
			return exitRequest
		}
		// ★ 답을 조립한다 ★ — --json 이 통짜, --set k=v 가 문자열 필드.
		answer := map[string]any{}
		if *answerJSON != "" {
			if err := json.Unmarshal([]byte(*answerJSON), &answer); err != nil {
				fmt.Fprintln(os.Stderr, "--json 을 못 읽는다:", err)
				return exitRequest
			}
		}
		for _, kv := range sets {
			k, v, ok := strings.Cut(kv, "=")
			if !ok {
				fmt.Fprintln(os.Stderr, "--set 은 필드=값 형태다:", kv)
				return exitRequest
			}
			answer[k] = v
		}
		if len(answer) == 0 {
			fmt.Fprintln(os.Stderr, "답이 비었다 — --set 이나 --json 으로 채운다")
			return exitRequest
		}
		body, _ := json.Marshal(answer)
		r, err := c.Answer(ctx, arg, seq, body)
		if code := report(err); code != 0 {
			return code
		}
		fmt.Printf("답했다  %s #%d", arg, seq)
		if r.State != "" {
			fmt.Printf("  → run %s", r.State)
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
		if f.Code/100 == 4 {
			return exitRequest // 계약이 틀렸거나(400/422) 지금은 안 된다(409)
		}
		return exitSystem
	}
	fmt.Fprintln(os.Stderr, err)
	return exitSystem
}

// verdictCode 는 ★ Run 의 성패 ★ 를 종료코드로 옮긴다.
// 요청이 정상이었는데 일이 실패한 것이므로 2 가 아니라 1 이다.
func verdictCode(r *runctl.Run) int {
	if r.State == "SUCCEEDED" {
		return exitOK
	}
	return exitRunFail
}

func printRun(r *runctl.Run) {
	fmt.Printf("%s  %s\n", r.RunID, r.State)
	for _, a := range r.Assigned {
		for _, n := range a.Nodes {
			fmt.Printf("  %-10s %s  %s\n", a.As, n.Node, n.Label)
		}
	}
	// ★ 폭이 1 을 넘으면 Run 상태 한 줄로는 안 보인다 ★ (ADR-025) —
	// 어느 가지가 어디까지 갔고 지금 도는 것이 어느 기계인지를 여기서 읽는다.
	for _, st := range r.Steps {
		line := fmt.Sprintf("  %2d %-16s %-8s %s", st.Seq, st.ID, st.State, st.Node)
		if st.Attempt > 0 {
			line += fmt.Sprintf("  (%d회차)", st.Attempt+1)
		}
		// 기다리는 중이면 ★ 무엇을 기다리는지 ★ 를 같이 보여준다.
		if st.State == "PENDING" && len(st.Needs) > 0 {
			line += "  ← " + strings.Join(st.Needs, " · ")
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
					mark = "✗ "
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
	fmt.Fprint(os.Stderr, `runctl — Run 을 만들고 지켜보고 기록을 받는다

  runctl submit  <contract.json> [--wait]
  runctl dry-run <contract.json>          매칭만 해본다. 아무것도 점유하지 않는다.
  runctl status  <run-id>
  runctl record  <run-id> [-o out.tar]    ★ 봉인된 Run Record ★
  runctl cancel  <run-id>
  runctl capabilities                     ★ 함대의 속성 어휘 ★ — 계약을 쓰기 전에
  runctl asks                             ★ 답을 기다리는 질문들 ★ (인박스)
  runctl answer <run-id> <seq> --set k=v [--set …]   질문에 답한다

종료코드
  0  Run 이 SUCCEEDED (또는 아직 진행 중)
  1  ★ Run 이 FAILED ★ — 요청은 정상이었다
  2  ★ 요청이 거절됐다 ★ — 계약을 고치거나(400/422) 나중에 다시(409)
  3  ★ Mediator 에 못 닿았다 ★

옵션
`)
	flag.PrintDefaults()
}
