package enode

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Setup 은 노드 설정을 하나 만든다.
//
// 두 바이너리가 같은 것을 부른다 — enodectl setup 이 제자리이지만(설정
// 디렉터리를 이미 그쪽이 쥐고 있다), 윈도우에서 enode.exe 만 보고 있을 수
// 있어 그쪽에도 별칭을 둔다. 코드가 여기 하나이므로 둘이 갈릴 일이 없다.
type SetupOptions struct {
	Name          string
	Path          string // 비면 ConfDir()/<Name>.yaml
	Mediator      string
	Token         string
	Workspace     string
	WorkspaceID   string
	Arch          string
	BoardSoC      string
	BoardTag      string
	BoardPort     string
	Labels        map[string]string
	Orchestration bool
	MinFreeGB     int

	Check bool // 아무것도 안 쓰고 상태만 본다
	Yes   bool // 묻지 않는다
	In    io.Reader
	Out   io.Writer
}

// Setup 은 0 또는 종료 코드를 돌려준다.
func Setup(o SetupOptions) int {
	if o.In == nil {
		o.In = os.Stdin
	}
	if o.Out == nil {
		o.Out = os.Stdout
	}
	p := func(f string, a ...any) { fmt.Fprintf(o.Out, f, a...) }
	in := bufio.NewReader(o.In)
	interactive := !o.Yes && !o.Check

	if o.Name == "" {
		o.Name = "local"
	}
	if o.Path == "" {
		o.Path = filepath.Join(ConfDir(), o.Name+".yaml")
	}
	abs, err := filepath.Abs(o.Path)
	if err == nil {
		o.Path = abs
	}

	p("\n  config file   %s\n", o.Path)
	p("                The absolute path of this file is part of the node id.\n")
	p("                Moving it later makes a different node.\n\n")

	if interactive {
		if o.Mediator == "" {
			o.Mediator = "http://127.0.0.1:8080"
		}
		o.Mediator = ask(in, o.Out, "mediator address", o.Mediator)
		if o.Token == "" {
			o.Token = ask(in, o.Out, "token", "")
		}
		if o.Workspace == "" {
			o.Workspace = ask(in, o.Out,
				"workspace (a checkout; empty to advertise reasoning only)", "")
		}
	}
	if o.Mediator == "" {
		p("a mediator address is required.\n")
		return 2
	}

	// ① 기계가 이미 아는 것을 보여준다. 사람이 그것을 또 적으려다 마는 일을
	// 없앤다 — 그리고 탐지가 사람이 적은 것을 이긴다는 규칙이 화면에 보인다.
	local := Local{
		Mediator: o.Mediator, Token: o.Token, Workspace: o.Workspace,
		WorkspaceID: o.WorkspaceID,
		Arch:        o.Arch, Labels: o.Labels, Orchestration: o.Orchestration,
		MinFreeGB: o.MinFreeGB,
	}
	if o.BoardSoC != "" || o.BoardTag != "" || o.BoardPort != "" {
		local.Board = &Board{SoC: o.BoardSoC, Tag: o.BoardTag, Port: o.BoardPort}
	}
	// nil 을 넘기면 안 된다 — Detect 는 「깔렸는데 못 쓴다」에서 log.Warn 을
	// 부르고, 그것이 nil 이면 그 자리에서 죽는다. 그리고 그 경고가 바로
	// setup 이 보여줘야 하는 것이다 (ADR-059): 로그인 안 한 claude 는
	// 광고에서 빠지고, 사람은 그 이유를 여기서 알아야 한다.
	dlog := slog.New(slog.NewTextHandler(o.Out, &slog.HandlerOptions{Level: slog.LevelWarn}))
	// 여기서는 Background 가 맞다 — setup 은 사람이 앞에 앉아 한 번 도는
	// 명령이고, 탐지가 걸리면 그 사람이 Ctrl-C 로 끊는다. Detect 가 ctx 를
	// 받는 이유는 광고 루프가 그것 때문에 멈추지 않게 하려는 것인데,
	// 여기에는 멈출 루프가 없다.
	caps := Detect(context.Background(), local, dlog)

	p("\n  looking at this machine…\n\n")
	if len(caps) == 0 {
		p("  nothing to advertise yet — no harness and no workspace.\n")
	}
	for _, c := range caps {
		printAttrs(o.Out, c.Attrs)
	}

	if interactive {
		p("\n  These the machine cannot know. Leave empty to skip.\n\n")
		o.Arch = ask(in, o.Out, "build target arch (arch)", o.Arch)
		o.BoardSoC = ask(in, o.Out, "board soc", o.BoardSoC)
		if o.BoardSoC != "" {
			o.BoardTag = ask(in, o.Out, "board tag", o.BoardTag)
			o.BoardPort = ask(in, o.Out, "board port", o.BoardPort)
		}
		if s := ask(in, o.Out, "labels (k=v, comma separated)", labelString(o.Labels)); s != "" {
			o.Labels = parseLabels(s)
		}
		local.Arch, local.Labels = o.Arch, o.Labels
		if o.BoardSoC != "" {
			local.Board = &Board{SoC: o.BoardSoC, Tag: o.BoardTag, Port: o.BoardPort}
		}
		caps = Detect(context.Background(), local, dlog)
	}

	// ② 쓰기 전에 Mediator 에 물어본다.
	//
	// 토큰이 틀리면 런타임에 조용히 401 이고, 노드는 그냥 함대에 안 나타난다.
	// 가장 흔한 실패를 쓰기 전에 잡는다. 광고를 보내지는 않는다 — 설정을
	// 만드는 일이 함대 상태를 바꾸면 안 된다.
	p("\n  checking the mediator…\n\n")
	reach, code := probeMediator(o.Mediator, o.Token)
	switch {
	case !reach:
		p("  unreachable  %s\n", o.Mediator)
		p("\n  The node cannot advertise until this address answers.\n")
	case code == 200:
		p("  reachable    %s\n", o.Mediator)
		p("  token        accepted\n")
	case code == 401 || code == 403:
		p("  reachable    %s\n", o.Mediator)
		p("  token        rejected (%d)\n", code)
		p("\n  The mediator has a different token. A node with the wrong token is\n")
		p("  not refused loudly at runtime — it simply never appears in the fleet.\n")
		p("  Find it in the mediator's config, or run `mediator setup` there.\n")
	default:
		p("  reachable    %s\n", o.Mediator)
		p("  token        cannot tell (%d)\n", code)
	}

	p("\n")
	if id, err := Derive(o.Path); err == nil {
		p("  node id      %s   %s\n", id.NodeID, id.Label)
	} else {
		p("  node id      cannot derive: %v\n", err)
	}
	// 속성을 여기서 한 번 더 낸다. 위의 「looking at this machine」 은 사람이
	// arch · board · labels 를 답하기 전 것이라, 답한 뒤에 실제로 나가는
	// 내용과 다르다. 사람이 확인하고 승인하는 것은 이쪽이어야 한다.
	for _, c := range caps {
		p("  advertising  %s\n", c.Capability)
		for _, line := range wrapAttrs(attrPairs(c.Attrs), 60) {
			p("               %s\n", line)
		}
	}
	if o.Orchestration {
		p("\n  note  arch is deliberately not advertised. A command step must not run\n")
		p("        on the machine that holds the database, the artifacts and the token.\n")
	}
	if len(caps) == 0 {
		p("\n  note  this node would advertise nothing, so no step can match it.\n")
		p("        Give it a workspace, or log in to a harness (ADR-059).\n")
	}
	p("\n")

	if o.Check {
		return 0
	}
	if interactive && !confirm(in, o.Out, "write it?") {
		p("nothing was written.\n")
		return 0
	}

	if err := writeLocal(o.Path, local); err != nil {
		fmt.Fprintf(o.Out, "cannot write %s: %v\n", o.Path, err)
		if os.IsPermission(err) {
			fmt.Fprintf(o.Out, "try again with sudo, or pass --config with a path you can write.\n")
		}
		return 1
	}
	p("wrote %s\n\n", o.Path)
	p("start it with:  enodectl start %s\n", o.Name)
	return 0
}

// probeMediator 는 인증만 확인한다. 광고를 보내지 않는다.
func probeMediator(addr, token string) (reachable bool, code int) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
		strings.TrimRight(addr, "/")+"/v1/capabilities", nil)
	if err != nil {
		return false, 0
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	c := &http.Client{Timeout: 5 * time.Second}
	resp, err := c.Do(req)
	if err != nil {
		return false, 0
	}
	defer resp.Body.Close() //nolint:errcheck
	return true, resp.StatusCode
}

// writeLocal 은 설정을 0600 으로 쓴다. 토큰이 그 안에 있다.
func writeLocal(path string, l Local) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := yaml.Marshal(l)
	if err != nil {
		return err
	}
	head := "# enode node config.\n" +
		"# The absolute path of this file is part of the node id (ADR-017):\n" +
		"# moving it makes a different node.\n\n"
	return os.WriteFile(path, append([]byte(head), b...), 0o600)
}

func printAttrs(w io.Writer, attrs map[string]string) {
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(w, "  %-11s %s\n", k, attrs[k])
	}
}

func labelString(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+m[k])
	}
	return strings.Join(parts, ",")
}

func parseLabels(s string) map[string]string {
	m := map[string]string{}
	for _, kv := range strings.Split(s, ",") {
		kv = strings.TrimSpace(kv)
		if kv == "" {
			continue
		}
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		m[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	if len(m) == 0 {
		return nil
	}
	return m
}

func ask(in *bufio.Reader, w io.Writer, prompt, def string) string {
	if def != "" {
		fmt.Fprintf(w, "%s [%s]: ", prompt, def)
	} else {
		fmt.Fprintf(w, "%s: ", prompt)
	}
	line, err := in.ReadString('\n')
	if err != nil && line == "" {
		return def
	}
	if v := strings.TrimSpace(line); v != "" {
		return v
	}
	return def
}

func confirm(in *bufio.Reader, w io.Writer, prompt string) bool {
	fmt.Fprintf(w, "%s [Y/n]: ", prompt)
	line, _ := in.ReadString('\n')
	v := strings.ToLower(strings.TrimSpace(line))
	return v == "" || v == "y" || v == "yes"
}

// SetupCLI 는 플래그를 읽고 Setup 을 부른다.
//
// enodectl setup 과 enode setup 이 같은 이것을 부른다. 배포 모양 때문에
// 이름이 둘이지만(윈도우에 enode.exe 만 푼 사람이 설정을 만들 수 있어야
// 한다) 코드가 하나이므로 둘이 갈릴 수 없다.
func SetupCLI(prog string, args []string) int {
	fs := flag.NewFlagSet(prog, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		path  = fs.String("config", "", "write here instead of <confdir>/<name>.yaml")
		med   = fs.String("mediator", "", "mediator address")
		tok   = fs.String("token", "", "the token the mediator shares with its nodes")
		ws    = fs.String("workspace", "", "a checkout this node stands in")
		wsid  = fs.String("workspace-id", "", "name the repo yourself when .git and .repo are both absent")
		arch  = fs.String("arch", "", "build target arch, when the machine cannot tell")
		soc   = fs.String("board-soc", "", "the chip on the port; the machine cannot see it")
		tag   = fs.String("board-tag", "", "a name for that board")
		port  = fs.String("board-port", "", "the serial port it hangs on")
		label = fs.String("label", "", "k=v pairs, comma separated")
		orch  = fs.Bool("orchestration", false, "this node builds contracts instead of running commands")
		free  = fs.Int("min-free-gb", 0, "stop advertising build capacity below this")
		check = fs.Bool("check", false, "report and write nothing")
		yes   = fs.Bool("yes", false, "ask nothing")
	)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s <name> [flags]\n\n"+
			"Creates a node config and checks it against the mediator before writing.\n"+
			"The absolute path of that file is part of the node id: moving it later\n"+
			"makes a different node.\n\nflags:\n", prog)
		fs.PrintDefaults()
	}
	// 이름을 먼저 뗀다. Go 의 flag 는 첫 비플래그 인자에서 파싱을 멈추므로,
	// 사람이 자연스럽게 치는 `setup win-builder --check` 의 플래그가 통째로
	// 무시된다 (실측). 이름은 위치 인자로 두는 편이 자연스러우니 여기서 뗀다.
	name := "local"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		name, args = args[0], args[1:]
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if rest := fs.Args(); len(rest) > 0 {
		fmt.Fprintf(os.Stderr, "unexpected argument: %s\n", rest[0])
		return 2
	}

	fmt.Printf("%s — creating a node config.\n", prog)
	if !*check && !*yes {
		fmt.Printf("Press enter to keep the value in brackets.\n")
	}

	return Setup(SetupOptions{
		Name: name, Path: *path,
		Mediator: *med, Token: *tok, Workspace: *ws, WorkspaceID: *wsid,
		Arch: *arch, BoardSoC: *soc, BoardTag: *tag, BoardPort: *port,
		Labels: parseLabels(*label), Orchestration: *orch, MinFreeGB: *free,
		Check: *check, Yes: *yes,
	})
}

func attrPairs(attrs map[string]string) []string {
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k+"="+attrs[k])
	}
	return out
}

// wrapAttrs 는 한 줄에 width 를 넘지 않게 접는다. 속성 하나가 그보다 길면
// (워크스페이스 경로가 흔히 그렇다) 자르지 않고 그 줄을 넘긴다 — 잘린 경로는
// 사람이 확인할 수 없다.
func wrapAttrs(pairs []string, width int) []string {
	var lines []string
	cur := ""
	for _, kv := range pairs {
		switch {
		case cur == "":
			cur = kv
		case len(cur)+3+len(kv) <= width:
			cur += " · " + kv
		default:
			lines = append(lines, cur)
			cur = kv
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}
