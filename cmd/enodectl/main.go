// enodectl — 한 기계의 enode 인스턴스들을 다룬다.
//
//	enodectl list                설정 = 노드. 무엇이 있고 무엇이 도는가
//	enodectl id <이름>           node_id 를 미리 계산한다 (띄우기 전에 확인)
//	enodectl start <이름> [인자…]
//	enodectl stop  <이름>
//	enodectl logs  <이름> [-f]
//	enodectl status
//
// 왜 「이름」이 인자인가
// 설정 파일이 곧 신원이다 (ADR-015 §2). 한 기계에서 노드를 여럿 세우는 것이
// 예외가 아니라 기본이다 — 맥에 zephyr 워크스페이스 노드, colima 안의
// 노드, qemu 노드가 각각 선다. 그래서 이 도구는 하나를 다루는 형태를 안 갖는다.
//
// 왜 Go 인가 — 셸판은 node_id 를 다시 계산 했다. sha256 파이프와
// python3 realpath 로 Derive() 를 흉내 냈고, 그래서 ① python3 가 없는 기계에서
// 못 돌고 ② 흉내가 어긋나면 거짓 node_id를 말한다. 여기서는 enode 가
// 쓰는 그 함수를 그대로 부른다 — 정의상 어긋날 수 없다.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/taeels/enode/internal/build"
	"time"

	"github.com/taeels/enode/internal/enode"
)

func main() {
	args := os.Args[1:]
	cmd := "status"
	if len(args) > 0 {
		cmd, args = args[0], args[1:]
	}
	var err error
	switch cmd {
	case "list":
		err = cmdList()
	case "id":
		err = cmdID(args)
	case "start":
		err = cmdStart(args)
	case "stop":
		err = cmdStop(args)
	case "logs":
		err = cmdLogs(args)
	case "status":
		err = cmdStatus()
	case "-h", "--help", "help":
		usage()
	// 도구는 자기가 무엇인지 말할 수 있어야 한다 (ADR-056) — enode 와
	// runctl 은 답하는데 이것만 못 답했다. 실측(vm-scratch-5)에서 계획이
	// `enode --version` 에 막힌 것과 같은 종류의 구멍이다.
	case "--version", "-version", "version":
		fmt.Println(build.Version("enodectl"))
	default:
		err = fmt.Errorf("unknown command: %s  (list · id · start · stop · logs · status · version)", cmd)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Print(`enodectl — manage the enode instances on one machine.

  enodectl list                a config is a node. what exists and what runs
  enodectl id <name>           compute node_id ahead of time
  enodectl start <name> [args…]
  enodectl stop  <name>
  enodectl logs  <name> [-f]
  enodectl status

env: ENODE_CONFDIR · ENODE_STATEDIR · ENODE_BIN
`)
}

// ── 자리 ─────────────────────────────────────────────────────────────────

func home() string { h, _ := os.UserHomeDir(); return h }

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func confDir() string { return envOr("ENODE_CONFDIR", filepath.Join(home(), ".config", "enode")) }
func stateDir() string {
	return envOr("ENODE_STATEDIR", filepath.Join(home(), ".local", "state", "enode"))
}
func confOf(n string) string { return filepath.Join(confDir(), n+".yaml") }
func logOf(n string) string  { return filepath.Join(stateDir(), n+".log") }

// enodeBin 은 PATH 의 enode 를 먼저 보고, 없으면 ~/.local/bin 을 쓴다.
func enodeBin() string {
	if v := os.Getenv("ENODE_BIN"); v != "" {
		return v
	}
	if p, err := exec.LookPath("enode"); err == nil {
		return p
	}
	return filepath.Join(home(), ".local", "bin", "enode")
}

// names 는 설정 디렉터리의 <이름>.yaml 을 이름만 뽑아 정렬해 돌려준다.
func names() []string {
	ents, err := os.ReadDir(confDir())
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range ents {
		if e.IsDir() || filepath.Ext(e.Name()) != ".yaml" {
			continue
		}
		out = append(out, strings.TrimSuffix(e.Name(), ".yaml"))
	}
	sort.Strings(out)
	return out
}

// ── 도는가 ───────────────────────────────────────────────────────────────

// pidOf 는 그 설정을 열고 있는 프로세스의 pid 다. 없으면 0.
//
// 잠금 파일이 곧 상태다 — enode 가 자기 pid 를 거기 쓰고, 죽으면 커널이
// flock 을 푼다. 따로 pid 장부를 두면 그 장부가 진실과 갈라진다.
//
// 그런데 파일의 존재만으로는 아무것도 못 말한다 — enode 는 끝나도 잠금
// 파일을 안 지운다(flock 은 커널이 푼다). 그래서 그 pid 가 살아 있고 그
// 설정을 열고 있는지 를 함께 본다.
func pidOf(n string) int {
	conf := confOf(n)
	b, err := os.ReadFile(conf + ".lock")
	if err != nil {
		return 0
	}
	line := strings.TrimSpace(strings.SplitN(string(b), "\n", 2)[0])
	pid, err := strconv.Atoi(line)
	if err != nil || pid <= 0 {
		return 0
	}
	// 이름이 아니라 명령줄을 본다 — pid 는 재사용되고, 실행파일 이름은
	// 배포 방식에 따라 다르다(설치본은 enode, 묶음에서 바로 돌리면
	// enode-linux-amd64). 우리가 묻는 것은 이 설정을 열고 있는가다.
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "args=").Output()
	if err != nil || !strings.Contains(string(out), conf) {
		return 0
	}
	return pid
}

// alive 는 그 pid 가 아직 사는지다.
func alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return processAlive(pid)
}

// ── 명령 ─────────────────────────────────────────────────────────────────

func cmdList() error {
	fmt.Printf("config dir: %s\n\n", confDir())
	ns := names()
	if len(ns) == 0 {
		fmt.Printf("  (no configs. create <name>.yaml under %s)\n", confDir())
		return nil
	}
	for _, n := range ns {
		id, label := identityOf(n)
		state := "stopped"
		if pid := pidOf(n); pid > 0 {
			state = fmt.Sprintf("running pid=%d", pid)
			// 잠자기 방지가 붙어 있는지 함께 보인다 —
			// 안 붙어 있으면 시연 중에 끊긴다.
			if runtime.GOOS == "darwin" && caffeinated(pid) {
				state += " ☕"
			}
		}
		fmt.Printf("  %-14s %-14s %-24s %s\n", n, id, label, state)
	}
	return nil
}

func cmdID(args []string) error {
	n, err := oneName(args)
	if err != nil {
		return err
	}
	ident, err := enode.Derive(confOf(n))
	if err != nil {
		return err
	}
	fmt.Printf("node_id  %s\nlabel    %s\nconfig   %s\n",
		ident.NodeID, ident.Label, ident.Config)
	return nil
}

func cmdStart(args []string) error {
	n, err := oneName(args)
	if err != nil {
		return err
	}
	rest := args[1:]
	conf := confOf(n)
	bin := enodeBin()
	if st, err := os.Stat(bin); err != nil || st.IsDir() {
		return fmt.Errorf("enode binary not found: %s", bin)
	}
	if pid := pidOf(n); pid > 0 {
		return fmt.Errorf("already running (pid=%d); enode's flock rejects the second one", pid)
	}
	// PATH 에 claude 가 있는지 본다 — 맥에서 가장 잘 밟는 자리다.
	// launchd 로 띄우면 PATH 가 최소 집합이라 claude 를 못 찾고, 그러면
	// harness 가 광고에서 조용히 빠져 계약이 422 를 받는다. 원인이 안 보인다.
	if _, err := exec.LookPath("claude"); err != nil {
		fmt.Fprintln(os.Stderr, "▲ claude is not on PATH; this node will not advertise a harness.")
	}
	if err := os.MkdirAll(stateDir(), 0o755); err != nil {
		return err
	}
	log, err := os.OpenFile(logOf(n), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer log.Close() //nolint:errcheck
	fmt.Fprintf(log, "\n===== %s start =====\n", time.Now().Format("2006-01-02 15:04:05"))

	c := exec.Command(bin, append([]string{"--config", conf}, rest...)...)
	c.Stdout, c.Stderr = log, log
	// 부모에서 떼어낸다 — enodectl 이 끝나도 노드는 살아 있어야 한다.
	c.SysProcAttr = detachAttr()
	if err := c.Start(); err != nil {
		return err
	}
	_ = c.Process.Release()

	time.Sleep(time.Second)
	pid := pidOf(n)
	if pid == 0 {
		fmt.Fprintln(os.Stderr, "✗ did not come up. tail of the log:")
		tailTo(os.Stderr, logOf(n), 20)
		return errors.New("start failed")
	}
	ident, _ := enode.Derive(conf)
	fmt.Printf("up          %s  node=%s  pid=%d\n", n, ident.NodeID, pid)
	fmt.Printf("  log: %s\n", logOf(n))
	keepAwake(pid)
	return nil
}

func cmdStop(args []string) error {
	n, err := oneName(args)
	if err != nil {
		return err
	}
	pid := pidOf(n)
	if pid == 0 {
		fmt.Printf("already stopped: %s\n", n)
		return nil
	}
	// SIGTERM 이면 signal.NotifyContext 가 받아 스스로 정리하고 끝난다.
	// 윈도우에는 그 길이 없다 — proc_windows.go 가 이유를 적는다.
	if err := signalStop(pid); err != nil {
		return err
	}
	for i := 0; i < 20; i++ {
		if pidOf(n) == 0 {
			fmt.Printf("stopped: %s\n", n)
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	fmt.Fprintf(os.Stderr, "▲ did not exit within 10s; killing (pid=%d)\n", pid)
	return signalKill(pid)
}

func cmdLogs(args []string) error {
	n, err := oneName(args)
	if err != nil {
		return err
	}
	log := logOf(n)
	if _, err := os.Stat(log); err != nil {
		return fmt.Errorf("no log: %s", log)
	}
	if len(args) > 1 && args[1] == "-f" {
		c := exec.Command("tail", "-f", log)
		c.Stdout, c.Stderr = os.Stdout, os.Stderr
		return c.Run()
	}
	tailTo(os.Stdout, log, 50)
	return nil
}

func cmdStatus() error {
	if err := cmdList(); err != nil {
		return err
	}
	fmt.Println()
	for _, n := range names() {
		if pidOf(n) == 0 {
			continue
		}
		fmt.Printf("── %s ── recent log\n", n)
		tailIndent(os.Stdout, logOf(n), 5, "   ")
		fmt.Println()
	}
	return nil
}

// ── 곁 ───────────────────────────────────────────────────────────────────

func oneName(args []string) (string, error) {
	if len(args) == 0 || args[0] == "" {
		return "", errors.New("a name is required")
	}
	n := args[0]
	if _, err := os.Stat(confOf(n)); err != nil {
		return "", fmt.Errorf("no config: %s", confOf(n))
	}
	return n, nil
}

// identityOf 는 목록에 쓸 값이다. 실패해도 줄을 지우지 않는다 —
// git 이메일이 없으면 신원을 못 만들지만, 그 설정이 있다는 사실은 보여야 한다.
func identityOf(n string) (id, label string) {
	ident, err := enode.Derive(confOf(n))
	if err != nil {
		return "(no identity)", "(" + err.Error() + ")"
	}
	return ident.NodeID, ident.Label
}

// keepAwake 는 시스템 잠자기가 함대를 끊는 것을 막는다 (맥에서만).
//
// 디스플레이가 꺼지는 것은 상관없다. 시스템 잠자기가 CPU 와 네트워크를
// 멈추고, 그러면 광고가 끊기고 not_after 가 지나 Run 이 죽는다.
// 실측에서 밟았다: 승인을 기다리던 Run 이 맥이 조용해진 지 180초 만에
// "임대 만료로 Run 을 회수했다" 로 FAILED 가 됐다.
//
// enode 의 수명에 묶는다 (-w) — 껐다 잊는 일이 없고 유령이 안 남는다.
// -d 는 안 준다 — 화면은 꺼져도 된다. 우리가 막는 것은 그것이 아니다.
// sudo 를 안 쓴다 — 전원 설정을 영구히 바꾸지 않는다.
func keepAwake(pid int) {
	if runtime.GOOS != "darwin" {
		return
	}
	if _, err := exec.LookPath("caffeinate"); err != nil {
		fmt.Fprintln(os.Stderr, "  ▲ caffeinate is missing — system sleep can cut the fleet")
		return
	}
	c := exec.Command("caffeinate", "-i", "-s", "-w", strconv.Itoa(pid))
	c.SysProcAttr = detachAttr()
	if err := c.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "  ▲ could not start caffeinate: %v\n", err)
		return
	}
	_ = c.Process.Release()
	fmt.Printf("  ☕ sleep held off (while enode pid=%d lives)\n", pid)
}

// caffeinated 는 그 pid 를 지키는 caffeinate 가 붙어 있는지다.
func caffeinated(pid int) bool {
	out, err := exec.Command("ps", "-eo", "args=").Output()
	if err != nil {
		return false
	}
	want := "-w " + strconv.Itoa(pid)
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "caffeinate") && strings.Contains(line, want) {
			return true
		}
	}
	return false
}

func tailTo(w *os.File, path string, n int) { tailIndent(w, path, n, "") }

func tailIndent(w *os.File, path string, n int, indent string) {
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	for _, l := range lines {
		fmt.Fprintln(w, indent+l)
	}
}
