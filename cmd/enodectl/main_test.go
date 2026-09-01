package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// 이 파일은 자리와 이름과 신원 - OS 를 안 타는 것들만 본다. 프로세스를
// 다루는 것(pidOf · cmdStart · cmdStop · cmdLogs -f · cmdStatus ·
// caffeinated)은 lifecycle_unix_test.go 가 본다. 그쪽이 ps 와 SIGTERM 과
// 떼어낸 자식에 매여 있어서다.
//
// 격리는 생산 코드를 안 바꾸고 얻는다 - internal/enode 의 ConfDir 과
// StateDir 이 ENODE_CONFDIR · ENODE_STATEDIR 을 먼저 보고, enodectl 의
// confDir · stateDir 이 그 둘에 위임한다. 그래서 t.TempDir() 하나로
// 설정 · 잠금 · 로그가 통째로 갈린다.

// captureOutput 은 os.Stdout 과 os.Stderr 를 파이프로 바꿔 fn 이 찍은 것을
// 돌려준다.
//
// enodectl 은 fmt.Printf 로 os.Stdout 에 직접 쓰고, tailIndent 는 *os.File
// 을 받는다. 출력 대상을 io.Writer 로 바꾸는 안을 기각했다 - 이 단위는
// 커버리지 공백을 메우는 일이고, 그 일이 CLI 의 모양을 바꾸기 시작하면
// 무엇을 시험하는지가 흐려진다.
//
// 고루틴으로 비우는 이유 - 파이프 버퍼가 차면 fn 이 쓰기에서 멈추고,
// 멈추면 왜 멈췄는지 출력에 아무것도 안 남는다.
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

// isolate 는 이 테스트만의 설정 · 상태 디렉터리를 세운다.
func isolate(t *testing.T) (conf, state string) {
	t.Helper()
	conf, state = t.TempDir(), t.TempDir()
	t.Setenv("ENODE_CONFDIR", conf)
	t.Setenv("ENODE_STATEDIR", state)
	return conf, state
}

// gitIdentity 는 enode.Derive 가 읽는 전역 git 이메일을 고정한다.
//
// email 이 비면 이메일이 없는 기계를 만든다 - ADR-015 §1 이 조용한 대체를
// 금했으므로 그 자리에서 실패하는 것이 계약이고, 그 계약도 시험해야 한다.
// GIT_CONFIG_NOSYSTEM 까지 거는 이유는 /etc/gitconfig 가 있는 기계에서
// 이 테스트가 다르게 도는 것을 막기 위해서다.
func gitIdentity(t *testing.T, email string) {
	t.Helper()
	path := os.DevNull
	if email != "" {
		path = filepath.Join(t.TempDir(), "gitconfig")
		if err := os.WriteFile(path, []byte("[user]\n\temail = "+email+"\n"), 0o600); err != nil {
			t.Fatalf("write a git config for the test: %v", err)
		}
	}
	t.Setenv("GIT_CONFIG_GLOBAL", path)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
}

// writeConfig 는 <confdir>/<name>.yaml 을 만든다. 설정 파일이 곧 노드다
// (ADR-015 §2) - 내용은 이 패키지가 안 읽으므로 최소형으로 둔다.
func writeConfig(t *testing.T, name string) string {
	t.Helper()
	p := confOf(name)
	if err := os.WriteFile(p, []byte("mediator: http://127.0.0.1:8080\n"), 0o600); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

// ── 자리 ─────────────────────────────────────────────────────────────────

func TestPaths_TheEnvironmentDecidesWhereTheNodesLive(t *testing.T) {
	conf, state := isolate(t)
	if got := confDir(); got != conf {
		t.Fatalf("confDir() = %q, want %q from ENODE_CONFDIR", got, conf)
	}
	if got := stateDir(); got != state {
		t.Fatalf("stateDir() = %q, want %q from ENODE_STATEDIR", got, state)
	}
	// enode 와 enodectl 이 서로 다른 자리를 보면 enodectl start 로 띄운
	// 노드를 enode 가 혼자서는 못 찾는다. 이름을 붙이는 규칙이 그 접점이다.
	if got, want := confOf("win-builder"), filepath.Join(conf, "win-builder.yaml"); got != want {
		t.Fatalf("confOf = %q, want %q", got, want)
	}
	if got, want := logOf("win-builder"), filepath.Join(state, "win-builder.log"); got != want {
		t.Fatalf("logOf = %q, want %q", got, want)
	}
}

// ── 이름 ─────────────────────────────────────────────────────────────────

func TestNames_CountsTheYamlFilesAndSortsThem(t *testing.T) {
	conf, _ := isolate(t)
	for _, n := range []string{"zephyr", "alpha", "mid"} {
		writeConfig(t, n)
	}
	// 설정이 아닌 것들. 이것들이 섞여 나오면 목록이 노드 목록이 아니게 된다.
	if err := os.WriteFile(filepath.Join(conf, "notes.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write notes.txt: %v", err)
	}
	if err := os.Mkdir(filepath.Join(conf, "archive.yaml"), 0o755); err != nil {
		t.Fatalf("mkdir archive.yaml: %v", err)
	}
	got := names()
	if strings.Join(got, ",") != "alpha,mid,zephyr" {
		t.Fatalf("names() = %q, want the three yaml files sorted and nothing else", got)
	}
}

func TestNames_IsEmptyWhenThereIsNoConfigDirectory(t *testing.T) {
	// 설정 디렉터리가 아직 없는 기계에서도 enodectl 은 죽지 않고 답해야
	// 한다 - list 가 처음 쓰는 사람이 가장 먼저 치는 명령이다.
	t.Setenv("ENODE_CONFDIR", filepath.Join(t.TempDir(), "not-created-yet"))
	t.Setenv("ENODE_STATEDIR", t.TempDir())
	if got := names(); got != nil {
		t.Fatalf("names() = %q, want nil when the config directory does not exist", got)
	}
}

func TestOneName_RequiresExactlyOneNameThatAlreadyHasAConfig(t *testing.T) {
	isolate(t)
	writeConfig(t, "present")
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"no argument at all", nil, "a name is required"},
		{"an empty name", []string{""}, "a name is required"},
		{"a name with no config", []string{"absent"}, "no config:"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := oneName(tc.args)
			if err == nil {
				t.Fatalf("oneName(%q) = %q with no error, want an error", tc.args, got)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("oneName(%q) error = %q, want it to contain %q", tc.args, err, tc.want)
			}
		})
	}
	// 뒤에 붙는 인자는 노드로 넘어갈 것이라 이름 판정에 영향을 주지 않는다.
	got, err := oneName([]string{"present", "--verbose"})
	if err != nil || got != "present" {
		t.Fatalf("oneName([present --verbose]) = %q, %v; want present with no error", got, err)
	}
}

// ── 신원 ─────────────────────────────────────────────────────────────────

func TestIdentityOf_KeepsTheRowWhenTheIdentityCannotBeDerived(t *testing.T) {
	isolate(t)
	writeConfig(t, "orphan")

	// git 이메일이 없으면 신원을 못 만들지만, 그 설정이 있다는 사실은
	// 보여야 한다 - 목록에서 줄이 사라지면 사람은 설정을 잃었다고 읽는다.
	gitIdentity(t, "")
	id, label := identityOf("orphan")
	if id != "(no identity)" {
		t.Fatalf("identityOf id = %q, want the placeholder that keeps the row", id)
	}
	if !strings.Contains(label, "git") {
		t.Fatalf("identityOf label = %q, want it to say why the identity is missing", label)
	}

	gitIdentity(t, "node-test@example.invalid")
	id, label = identityOf("orphan")
	if len(id) != 12 {
		t.Fatalf("identityOf id = %q, want the 12-character node id (ADR-015 §2)", id)
	}
	if !strings.Contains(label, "node-test@") || !strings.Contains(label, "orphan") {
		t.Fatalf("identityOf label = %q, want the local part and the config name in it", label)
	}
}

func TestCmdID_ComputesTheNodeIdBeforeAnythingIsStarted(t *testing.T) {
	isolate(t)
	writeConfig(t, "pi2")
	gitIdentity(t, "node-test@example.invalid")

	stdout, _ := captureOutput(t, func() {
		if err := cmdID([]string{"pi2"}); err != nil {
			t.Errorf("cmdID = %v, want nil", err)
		}
	})
	for _, want := range []string{"node_id", "label", "config", "pi2"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("cmdID output = %q, want %q in it", stdout, want)
		}
	}
	// 설정 파일의 절대경로가 node_id 의 일부다 (ADR-017) - 그래서 그 경로를
	// 같이 찍는다. 사람이 옮겨도 되는지 판단하는 근거가 이 줄이다.
	if !strings.Contains(stdout, confOf("pi2")) {
		t.Fatalf("cmdID output = %q, want the absolute config path", stdout)
	}

	if err := cmdID(nil); err == nil {
		t.Fatal("cmdID(nil) = nil, want an error asking for a name")
	}
	gitIdentity(t, "")
	if err := cmdID([]string{"pi2"}); err == nil {
		t.Fatal("cmdID with no git email = nil, want the derivation error surfaced")
	}
}

// ── 목록 ─────────────────────────────────────────────────────────────────

func TestCmdList_SaysWhereToPutTheFirstConfigWhenThereIsNone(t *testing.T) {
	conf, _ := isolate(t)
	stdout, _ := captureOutput(t, func() {
		if err := cmdList(); err != nil {
			t.Errorf("cmdList = %v, want nil", err)
		}
	})
	if !strings.Contains(stdout, conf) {
		t.Fatalf("cmdList output = %q, want the config directory named", stdout)
	}
	if !strings.Contains(stdout, "no configs") {
		t.Fatalf("cmdList output = %q, want it to say the directory is empty and what to put there", stdout)
	}
}

func TestCmdList_ShowsAStoppedNodeAsStopped(t *testing.T) {
	isolate(t)
	gitIdentity(t, "node-test@example.invalid")
	writeConfig(t, "zephyr")
	stdout, _ := captureOutput(t, func() {
		if err := cmdList(); err != nil {
			t.Errorf("cmdList = %v, want nil", err)
		}
	})
	if !strings.Contains(stdout, "zephyr") {
		t.Fatalf("cmdList output = %q, want the config listed", stdout)
	}
	if !strings.Contains(stdout, "stopped") {
		t.Fatalf("cmdList output = %q, want a config with no live process shown as stopped", stdout)
	}
	if strings.Contains(stdout, "running") {
		t.Fatalf("cmdList output = %q, want nothing reported as running", stdout)
	}
}

// ── 로그 꼬리 ────────────────────────────────────────────────────────────

func TestTailIndent_ShowsAtMostTheLastNLinesAndIndentsThem(t *testing.T) {
	dir := t.TempDir()
	long := filepath.Join(dir, "long.log")
	if err := os.WriteFile(long, []byte("l1\nl2\nl3\nl4\nl5\n"), 0o600); err != nil {
		t.Fatalf("write long.log: %v", err)
	}
	stdout, _ := captureOutput(t, func() { tailIndent(os.Stdout, long, 2, "   ") })
	if strings.Contains(stdout, "l3") {
		t.Fatalf("tailIndent(n=2) output = %q, want only the last two lines", stdout)
	}
	if !strings.Contains(stdout, "   l4\n   l5\n") {
		t.Fatalf("tailIndent output = %q, want the last two lines each carrying the indent", stdout)
	}

	// n 보다 짧으면 있는 만큼만 낸다.
	short := filepath.Join(dir, "short.log")
	if err := os.WriteFile(short, []byte("only\n"), 0o600); err != nil {
		t.Fatalf("write short.log: %v", err)
	}
	stdout, _ = captureOutput(t, func() { tailTo(os.Stdout, short, 50) })
	if stdout != "only\n" {
		t.Fatalf("tailTo output = %q, want the single line with no indent", stdout)
	}

	// 로그가 아직 없는 것은 오류가 아니다 - 노드가 한 번도 안 떴을 뿐이다.
	stdout, stderr := captureOutput(t, func() {
		tailIndent(os.Stdout, filepath.Join(dir, "absent.log"), 5, "")
	})
	if stdout != "" || stderr != "" {
		t.Fatalf("tailIndent on a missing file wrote %q / %q, want silence", stdout, stderr)
	}
}
