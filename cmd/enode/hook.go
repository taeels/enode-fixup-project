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
		fmt.Fprintln(os.Stderr, "쓰임: enode hook stop --out <dir> [--workspace <dir>] [--expect a,b]")
		return 2
	}
	fs := flag.NewFlagSet("hook stop", flag.ContinueOnError)
	out := fs.String("out", "", "$OUT 경로")
	ws := fs.String("workspace", "", "워크스페이스 경로")
	expect := fs.String("expect", "", "계약이 요구한 산출물 이름 (쉼표)")
	if err := fs.Parse(args[1:]); err != nil {
		return 0 // ★ 인자가 이상해도 하네스를 막지 않는다 ★
	}

	var names []string
	for _, n := range strings.Split(*expect, ",") {
		if n = strings.TrimSpace(n); n != "" {
			names = append(names, n)
		}
	}
	a := enode.HookArgs{Out: *out, Workspace: *ws, Expect: names}
	if err := enode.RunStopHook(a, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "훅 실패(무시하고 통과):", err)
	}
	return 0
}
