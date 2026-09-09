package main

import (
	"net"
	"os"
	"strings"
	"testing"

	"github.com/taeels/enode/internal/enode"
)

// 제어판 하위명령은 데몬을 시작하지 않고 설정·바인딩 실패를 종료 코드로 전한다.
func TestRun_PanelRejectsInvalidStartup(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		code int
		want string
	}{
		{"unknown flag", []string{"--unknown"}, 2, "flag provided but not defined"},
		{"missing config", nil, 1, "no config file found"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolateNode(t)
			noSystemNodeConfig(t)
			code, _, stderr := callRun(t, append([]string{"panel"}, tc.args...)...)
			if code != tc.code || !strings.Contains(stderr, tc.want) {
				t.Fatalf("panel: code=%d stderr=%q, want %d and %q", code, stderr, tc.code, tc.want)
			}
		})
	}
	for _, tc := range []struct {
		name, config, policy, listen, want string
	}{
		{"bad config", "invalid: [", "", "127.0.0.1:8081", "cannot read config"},
		{"bad policy", "mediator: http://127.0.0.1:1\ntoken: test\n", "drain: [", "127.0.0.1:8081", "cannot read the policy file"},
		{"public bind without panel token", "mediator: http://127.0.0.1:1\ntoken: test\n", "", "0.0.0.0:8081", "cannot start the panel"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolateNode(t)
			config := nodeConfig(t, tc.config)
			if tc.policy != "" {
				if err := os.WriteFile(enode.PolicyPath(config), []byte(tc.policy), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			code, _, stderr := callRun(t, "panel", "--config", config, "--listen", tc.listen)
			if code != 1 || !strings.Contains(stderr, tc.want) {
				t.Fatalf("panel: code=%d stderr=%q, want failure and %q", code, stderr, tc.want)
			}
		})
	}
}

func TestRun_PanelReportsOccupiedPort(t *testing.T) {
	isolateNode(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	config := nodeConfig(t, "mediator: http://127.0.0.1:1\ntoken: test\n")
	code, _, stderr := callRun(t, "panel", "--config", config, "--listen", listener.Addr().String(), "--node", "test-panel")
	if code != 1 || !strings.Contains(stderr, "bind:") || !strings.Contains(stderr, "test-panel") {
		t.Fatalf("occupied panel port: code=%d stderr=%q", code, stderr)
	}
}
