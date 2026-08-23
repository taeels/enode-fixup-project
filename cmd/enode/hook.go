package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/taeels/enode/internal/enode"
)

// runHookCmd 는 `enode hook stop` 이다 — ★ 훅이 곧 enode 자신이다 ★.
//
//	✗ 셸 스크립트를 써서 심는다   윈도우에서 안 돈다. 인용이 위험하다. 시험이 어렵다.
//	○ enode 자신을 훅으로 심는다  ★ 이미 노드에 있는 단일 정적 바이너리다 ★ (ADR-015)
//	                              배포할 것이 늘지 않고 Go 로 시험된다.
//
// ★ 절대 실패로 끝내지 않는다 ★ — 훅이 죽으면 하네스가 멈춰 선다.
// 안전망이 정규 경로를 무너뜨리는 것이 가장 나쁘다. 그래서 무슨 일이 있어도 0 이다.
func runHookCmd(args []string) int {
	if len(args) == 0 || args[0] != "stop" {
		fmt.Fprintln(os.Stderr,
			"usage: enode hook stop --out <dir> [--workspace <dir>] [--expect a,b] "+
				"[--stamp <file>] [--plan <name>] [--roles a,b]")
		return 2
	}
	fs := flag.NewFlagSet("hook stop", flag.ContinueOnError)
	out := fs.String("out", "", "output directory ($OUT)")
	ws := fs.String("workspace", "", "workspace directory")
	expect := fs.String("expect", "", "comma-separated output names required by the contract")
	stamp := fs.String("stamp", "", "baseline timestamp file (needed to see ignored build artifacts)")
	// ★ 계획 단계의 보조 ★ (ADR-046) — 어긴 계획을 하네스가 끝나기 전에 짚는다.
	plan := fs.String("plan", "", "plan output name (expands steps only)")
	roles := fs.String("roles", "", "comma-separated roles allowed in uses")
	if err := fs.Parse(args[1:]); err != nil {
		return 0 // ★ 인자가 이상해도 하네스를 막지 않는다 ★
	}

	split := func(s string) []string {
		var out []string
		for _, n := range strings.Split(s, ",") {
			if n = strings.TrimSpace(n); n != "" {
				out = append(out, n)
			}
		}
		return out
	}
	a := enode.HookArgs{Out: *out, Workspace: *ws, Expect: split(*expect), Stamp: *stamp,
		Plan: *plan, Roles: split(*roles)}
	if err := enode.RunStopHook(a, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "hook error (ignored):", err)
	}
	return 0
}
