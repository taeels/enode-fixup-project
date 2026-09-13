package enode

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// capturingLog 는 노드 로그를 버퍼로 받는다.
//
// FR-3 이 요구하는 것은 「빠진 이유가 노드 로그에 남는다」이고, 그것은
// 속성으로는 안 잰다 — 빠진 것은 attrs 에 없기 때문이다.
func capturingLog() (*slog.Logger, *bytes.Buffer) {
	var b bytes.Buffer
	return slog.New(slog.NewTextHandler(&b, nil)), &b
}

// 실행 가능한 스크립트 하나를 PATH 에 안 넣고 절대경로로 둔다.
func execFile(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// 뜨나 판정은 존재까지다 — 프로세스를 안 띄우고 연결도 안 한다.
func TestMCPUp_ResolvesWithoutRunningAnything(t *testing.T) {
	abs := execFile(t, "serial-mcp")

	t.Run("절대경로가 있다", func(t *testing.T) {
		if err := mcpUp(MCPServer{Command: abs}); err != nil {
			t.Fatalf("an executable at an absolute path did not count as up: %v", err)
		}
	})
	t.Run("절대경로가 없다", func(t *testing.T) {
		err := mcpUp(MCPServer{Command: filepath.Join(t.TempDir(), "gone")})
		if err == nil {
			t.Fatal("a missing executable counted as up")
		}
		if !strings.Contains(err.Error(), "not an executable file") {
			t.Fatalf("the reason does not say what is wrong: %v", err)
		}
	})
	t.Run("PATH 에 있다", func(t *testing.T) {
		t.Setenv("PATH", filepath.Dir(abs))
		if err := mcpUp(MCPServer{Command: "serial-mcp"}); err != nil {
			t.Fatalf("an executable on PATH did not count as up: %v", err)
		}
	})
	t.Run("PATH 에 없다", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		err := mcpUp(MCPServer{Command: "serial-mcp"})
		if err == nil {
			t.Fatal("a command that does not resolve counted as up")
		}
		if !strings.Contains(err.Error(), "not found in PATH") {
			t.Fatalf("the reason does not say what is wrong: %v", err)
		}
	})
	t.Run("자격증명이 있다", func(t *testing.T) {
		t.Setenv("GERRIT_TOKEN_TEST", "x")
		s := MCPServer{URL: "https://gerrit.invalid/mcp", Credential: "GERRIT_TOKEN_TEST"}
		if err := mcpUp(s); err != nil {
			t.Fatalf("a remote server with its credential set did not count as up: %v", err)
		}
	})
	t.Run("자격증명이 없다", func(t *testing.T) {
		os.Unsetenv("GERRIT_TOKEN_TEST") //nolint:errcheck
		s := MCPServer{URL: "https://gerrit.invalid/mcp", Credential: "GERRIT_TOKEN_TEST"}
		err := mcpUp(s)
		if err == nil {
			t.Fatal("a remote server without its credential counted as up")
		}
		if !strings.Contains(err.Error(), "is not set in the node environment") {
			t.Fatalf("the reason does not say what is wrong: %v", err)
		}
	})
	t.Run("자격증명을 안 적었다", func(t *testing.T) {
		// 인증 없는 endpoint 가 있다. 볼 것이 없는 것은 안 뜨는 것이 아니다.
		if err := mcpUp(MCPServer{URL: "https://open.invalid/mcp"}); err != nil {
			t.Fatalf("a remote server without a credential name did not count as up: %v", err)
		}
	})
}

// 뜨는 것만 광고에 실리고 빠진 것의 사유가 로그에 남는다 (FR-3 · CA2).
func TestMCPFingerprint_AdvertisesOnlyWhatIsUp(t *testing.T) {
	abs := execFile(t, "serial-mcp")
	log, out := capturingLog()

	attrs, err := mcpFP{}.Probe(context.Background(), Local{MCP: map[string]MCPServer{
		"serial": {Command: abs},
		"gone":   {Command: filepath.Join(t.TempDir(), "missing")},
	}}, log)
	if err != nil {
		t.Fatalf("one server being down made the whole kind fail: %v", err)
	}
	if attrs["mcp.serial"] != "1" {
		t.Fatalf("the server that is up did not ride the advert: %v", attrs)
	}
	if _, ok := attrs["mcp.gone"]; ok {
		t.Fatalf("a server that is not up rode the advert: %v", attrs)
	}
	// 조용히 빼지 않는다 — 사람이 고칠 수 있는 문제다.
	if !strings.Contains(out.String(), "mcp server is not up") {
		t.Fatalf("the drop left no reason in the node log:\n%s", out)
	}
	if !strings.Contains(out.String(), "server=gone") {
		t.Fatalf("the reason does not name the server:\n%s", out)
	}
	if strings.Contains(out.String(), "server=serial") {
		t.Fatalf("a server that is up was reported as dropped:\n%s", out)
	}
}

// 선언이 없는 것은 정상이다 — 로그도 안 낸다.
func TestMCPFingerprint_NoDeclarationIsSilent(t *testing.T) {
	log, out := capturingLog()
	attrs, err := mcpFP{}.Probe(context.Background(), Local{}, log)
	if err != nil || len(attrs) != 0 {
		t.Fatalf("a node without mcp servers produced something: %v %v", attrs, err)
	}
	if out.Len() != 0 {
		t.Fatalf("a node without mcp servers wrote to the log:\n%s", out)
	}
}

// 한 종류가 실패해도 나머지는 실린다 (R7).
//
// 못 물어본 것을 「못 한다」로 광고하면 노드가 스스로를 지운다.
func TestCostlyAttrs_AFailingKindDoesNotTakeTheOthersDown(t *testing.T) {
	saved := fingerprinters
	t.Cleanup(func() { fingerprinters = saved })
	fingerprinters = []Fingerprinter{failingFP{}, stubFP{}}

	log, out := capturingLog()
	attrs := costlyAttrs(context.Background(), Local{}, log)
	if attrs["stub"] != "1" {
		t.Fatalf("a failing kind took the others down: %v", attrs)
	}
	if !strings.Contains(out.String(), "kind=boom") {
		t.Fatalf("the failing kind left no reason in the log:\n%s", out)
	}
}

type failingFP struct{}

func (failingFP) Kind() string { return "boom" }
func (failingFP) Probe(context.Context, Local, *slog.Logger) (map[string]string, error) {
	return map[string]string{"never": "1"}, errNotUsable
}

type stubFP struct{}

func (stubFP) Kind() string { return "stub" }
func (stubFP) Probe(context.Context, Local, *slog.Logger) (map[string]string, error) {
	return map[string]string{"stub": "1"}, nil
}

// mcp.<이름> 만으로는 광고하지 않는다 (R10).
//
// MCP 서버는 하네스가 물어야 쓸 수 있는 것이다. 선언만으로 광고를 내면
// 계약이 그 노드를 잡고 실행 시점에 하네스가 없어 죽는다.
func TestCapabilities_MCPAloneIsNotACapability(t *testing.T) {
	log, out := capturingLog()
	caps := capabilities(Local{MCP: map[string]MCPServer{"probe": {Command: "/bin/true"}}},
		log, map[string]string{"mcp.probe": "1"}, map[string]string{"os": "linux"})
	if len(caps) != 0 {
		t.Fatalf("a node with no harness advertised itself: %+v", caps)
	}
	// 침묵이 이 결정의 가장 큰 대가다 — 소유자가 이유를 알아야 한다.
	if !strings.Contains(out.String(), "declares mcp servers but advertises no capability") {
		t.Fatalf("the silent node left no reason in the log:\n%s", out)
	}
}

// 하네스가 함께 있으면 광고가 나가고 mcp.<이름> 도 함께 실린다.
func TestCapabilities_MCPRidesAlongWhenAHarnessIsThere(t *testing.T) {
	log, out := capturingLog()
	caps := capabilities(Local{MCP: map[string]MCPServer{"probe": {Command: "/bin/true"}}}, log,
		map[string]string{"mcp.probe": "1", "harness.claude": "1", "harness": "claude"},
		map[string]string{"os": "linux"})
	if len(caps) != 1 {
		t.Fatalf("a node with a harness did not advertise: %+v", caps)
	}
	for _, key := range []string{"mcp.probe", "harness.claude", "harness"} {
		if _, ok := caps[0].Attrs[key]; !ok {
			t.Fatalf("%s is missing from the advert: %v", key, caps[0].Attrs)
		}
	}
	if strings.Contains(out.String(), "advertises no capability") {
		t.Fatalf("a node that does advertise was reported as silent:\n%s", out)
	}
}

// 하네스는 새 키와 옛 키를 함께 싣는다 (ADR-035 §4.3 · §6).
//
// 옛 키를 걷으면 오늘 도는 계약(requires: harness: claude)의 매칭이
// 같은 배포에서 끊긴다. 걷는 날은 이 회차가 안 정한다.
func TestHarnessFingerprint_CarriesBothTheNewAndTheOldKey(t *testing.T) {
	dir := t.TempDir()
	bin := recordingClaude(t, dir, filepath.Join(dir, "calls"))
	log, _ := capturingLog()

	attrs, err := harnessFP{}.Probe(context.Background(), Local{HarnessBin: bin}, log)
	if err != nil {
		t.Fatal(err)
	}
	if attrs["harness.claude"] != "1" {
		t.Fatalf("the new key is missing: %v", attrs)
	}
	if attrs["harness"] != "claude" {
		t.Fatalf("the old key is missing; contracts that say harness: claude would stop matching: %v", attrs)
	}
}

// 저장소 유도가 안 되면 사람이 적은 이름을 쓴다 (ADR-036).
//
// 이 갈래를 순회로 옮기면서 빠뜨리면 git 없는 노드에서 repo 가 사라진다 —
// 그것이 이 리팩터가 깨뜨릴 수 있는 중립성이다.
func TestRepoFingerprint_KeepsTheWorkspaceIDFallback(t *testing.T) {
	log, _ := capturingLog()
	l := Local{Workspace: t.TempDir(), WorkspaceID: "hand-written"}

	attrs, err := repoFP{}.Probe(context.Background(), l, log)
	if err != nil {
		t.Fatal(err)
	}
	if attrs["repo"] != "hand-written" {
		t.Fatalf("the workspace_id fallback is gone: %v", attrs)
	}
}
