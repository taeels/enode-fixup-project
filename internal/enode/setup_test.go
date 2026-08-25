package enode

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mediatorSaying(t *testing.T, code int) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// --check 는 아무것도 안 쓴다. 이것이 깨지면 「보고만 받으려다 설정이
// 생겨 버리는」 상황이 되고, 경로가 곧 신원이므로 유령 노드가 하나 생긴다.
func TestSetup_CheckWritesNothing(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	code := Setup(SetupOptions{
		Name: "n", Path: filepath.Join(dir, "n.yaml"),
		Mediator: mediatorSaying(t, 200), Token: "t",
		Check: true, Out: &out, In: strings.NewReader(""),
	})
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "n.yaml")); err == nil {
		t.Fatal("--check wrote a config")
	}
}

// 토큰이 틀리면 런타임에 조용히 401 이고 노드가 함대에 안 나타난다.
// setup 이 그것을 쓰기 전에 말하지 않으면 이 도구는 존재할 이유가 없다.
func TestSetup_SaysWhenTheTokenIsRejected(t *testing.T) {
	var out bytes.Buffer
	Setup(SetupOptions{
		Name: "n", Path: filepath.Join(t.TempDir(), "n.yaml"),
		Mediator: mediatorSaying(t, 401), Token: "wrong",
		Check: true, Out: &out, In: strings.NewReader(""),
	})
	if !strings.Contains(out.String(), "rejected (401)") {
		t.Fatalf("a wrong token was not reported:\n%s", out.String())
	}
}

func TestSetup_SaysWhenTheMediatorIsUnreachable(t *testing.T) {
	var out bytes.Buffer
	Setup(SetupOptions{
		Name: "n", Path: filepath.Join(t.TempDir(), "n.yaml"),
		Mediator: "http://127.0.0.1:1", Token: "t",
		Check: true, Out: &out, In: strings.NewReader(""),
	})
	if !strings.Contains(out.String(), "unreachable") {
		t.Fatalf("an unreachable mediator was not reported:\n%s", out.String())
	}
}

// 쓴 것을 노드가 그대로 읽어야 한다. 이 왕복이 깨지면 setup 은 사람이
// 손으로 적는 것보다 나쁘다 — 통과했다고 말해 놓고 안 뜬다.
func TestSetup_WritesWhatLoadLocalReadsBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "deep", "n.yaml") // 상위 디렉터리도 없다
	var out bytes.Buffer
	code := Setup(SetupOptions{
		Name: "n", Path: path,
		Mediator: mediatorSaying(t, 200), Token: "shared-secret",
		Workspace: t.TempDir(), Arch: "arm64",
		BoardSoC: "qemu_cortex_m3", BoardTag: "qemu-probe", BoardPort: "COM3",
		Labels: map[string]string{"site": "office"},
		Yes:    true, Out: &out, In: strings.NewReader(""),
	})
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out.String())
	}

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("mode %04o, want 0600 — the file carries the shared token", fi.Mode().Perm())
	}

	l, err := LoadLocal(path)
	if err != nil {
		t.Fatalf("what setup wrote does not parse back: %v", err)
	}
	if l.Token != "shared-secret" || l.Arch != "arm64" {
		t.Fatalf("values did not survive: %+v", l)
	}
	if l.Board == nil || l.Board.SoC != "qemu_cortex_m3" || l.Board.Tag != "qemu-probe" {
		t.Fatalf("the board did not survive: %+v", l.Board)
	}
	if l.Labels["site"] != "office" {
		t.Fatalf("labels did not survive: %+v", l.Labels)
	}
}

// 사람이 답한 뒤의 광고 내용을 보여줘야 한다. 첫 화면은 arch 와 labels 를
// 묻기 전 것이라 실제로 나가는 것과 다르다 — 승인하는 대상이 그쪽이면 안 된다.
func TestSetup_ReportsTheAttributesItWillAdvertise(t *testing.T) {
	var out bytes.Buffer
	Setup(SetupOptions{
		Name: "n", Path: filepath.Join(t.TempDir(), "n.yaml"),
		Mediator: mediatorSaying(t, 200), Token: "t",
		Workspace: t.TempDir(), Arch: "arm64",
		Labels: map[string]string{"site": "office"},
		Check:  true, Out: &out, In: strings.NewReader(""),
	})
	s := out.String()
	i := strings.Index(s, "advertising")
	if i < 0 {
		t.Fatalf("it never said what it would advertise:\n%s", s)
	}
	got := advertised(s[i:])
	for _, want := range []string{"arch=arm64", "site=office"} {
		if !got[want] {
			t.Fatalf("%q is missing from the final report:\n%s", want, s[i:])
		}
	}
}

// advertised 는 최종 보고에 실린 속성 쌍을 토큰 단위로 모은다.
//
// 부분 문자열로 보면 안 된다 — "arch=amd64" 가 "host_arch=amd64" 안에
// 들어 있어서, arch 를 빼는 것을 확인하려던 시험이 통과할 수 없다 (실측).
func advertised(tail string) map[string]bool {
	out := map[string]bool{}
	for _, line := range strings.Split(tail, "\n") {
		if strings.Contains(line, "advertising") || !strings.Contains(line, "=") {
			continue
		}
		for _, tok := range strings.Split(line, "·") {
			if tok = strings.TrimSpace(tok); tok != "" {
				out[tok] = true
			}
		}
	}
	return out
}

// 오케스트레이터는 arch 를 안 광고한다 — 명령 단계가 데이터베이스와
// 아티팩트와 토큰을 쥔 기계에 내려앉으면 안 된다 (ADR-022 §5).
func TestSetup_OrchestrationDoesNotAdvertiseArch(t *testing.T) {
	var out bytes.Buffer
	Setup(SetupOptions{
		Name: "orch", Path: filepath.Join(t.TempDir(), "orch.yaml"),
		Mediator: mediatorSaying(t, 200), Token: "t",
		Workspace: t.TempDir(), Arch: "amd64", Orchestration: true,
		Check: true, Out: &out, In: strings.NewReader(""),
	})
	s := out.String()
	i := strings.Index(s, "advertising")
	if i < 0 {
		t.Fatalf("it never said what it would advertise:\n%s", s)
	}
	if advertised(s[i:])["arch=amd64"] {
		t.Fatalf("an orchestrator advertised a build arch:\n%s", s[i:])
	}
}

// 위치 인자가 플래그보다 앞에 온다 — Go 의 flag 는 거기서 파싱을 멈춘다.
// 이대로 두면 `setup win-builder --check` 가 조용히 설정을 써 버린다 (실측).
func TestSetupCLI_NameBeforeFlags(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ENODE_CONFDIR", dir)
	code := SetupCLI("setup", []string{"win-builder", "--check",
		"--mediator", mediatorSaying(t, 200), "--token", "t"})
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if _, err := os.Stat(filepath.Join(dir, "win-builder.yaml")); err == nil {
		t.Fatal("--check after the name was ignored and a config was written")
	}
}

func TestParseLabels(t *testing.T) {
	got := parseLabels(" site=office , role=builder ")
	if got["site"] != "office" || got["role"] != "builder" {
		t.Fatalf("%+v", got)
	}
	if parseLabels("") != nil {
		t.Fatal("an empty string must not make an empty map — it would advertise nothing but cost a key")
	}
}
