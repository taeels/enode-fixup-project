package enode

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeClaude 는 auth status 와 --version 에 원하는 대로 답하는 가짜다.
func fakeClaude(t *testing.T, authJSON string, authOK bool, exitNonZero ...bool) string {
	var nz bool
	if len(exitNonZero) > 0 {
		nz = exitNonZero[0]
	}
	_ = nz
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "claude")
	// authOK=false 는 하위명령이 없는 옛 CLI다.
	// 종료코드가 0 이 아니면서 JSON 을 찍는 경우는 exitNonZero 로 잰다 —
	// 실측에서 그것을 놓쳤다.
	authCase := "printf '%s\\n' " + shQuote(authJSON) + "; exit 0"
	if !authOK {
		authCase = "exit 127"
	}
	if nz {
		authCase = "printf '%s\\n' " + shQuote(authJSON) + "; exit 1"
	}
	script := "#!/bin/sh\n" +
		"case \"$1 $2\" in\n" +
		"  '--version ') echo 'fake 9.9.9 (Claude Code)'; exit 0 ;;\n" +
		"  'auth status') " + authCase + " ;;\n" +
		"esac\nexit 0\n"
	if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// shQuote 는 sh 작은따옴표 안에 안전하게 넣는다.
func shQuote(v string) string {
	return "'" + strings.ReplaceAll(v, "'", `'\''`) + "'"
}

// 「있다」와 「쓸 수 있다」를 가른다 (ADR-059)
//
// 실측 (2026-08-24) colima VM 의 노드가 harness=claude 를 광고했는데
// agent 단계가 1턴 1초에 죽었다: "Not logged in · Please run /login".
// --version 은 로그인 없이도 답한다 — 그것만 보면 못 가른다.
func Test로그인_안_된_하네스는_광고에서_빠진다(t *testing.T) {
	ctx := context.Background()

	// ① 로그인돼 있다 → 쓸 수 있다
	bin := fakeClaude(t, `{"loggedIn":true,"authMethod":"claude.ai"}`, true)
	ver, err := (claudeHarness{}).Probe(ctx, bin)
	if err != nil {
		t.Fatalf("로그인돼 있는데 거절했다: %v", err)
	}
	if ver == "" {
		t.Fatal("버전이 비었다")
	}

	// ② 로그인 안 됐다 → 못 쓴다
	bin = fakeClaude(t, `{"loggedIn":false}`, true)
	_, err = (claudeHarness{}).Probe(ctx, bin)
	if !errors.Is(err, errNotUsable) {
		t.Fatalf("로그인 안 됐는데 통과했다: %v", err)
	}

	// ③ 종료코드가 0 이 아니어도 나온 것을 읽는다
	//
	// 실측에서 놓쳤다 (2026-08-24) — colima VM 을 갱신했는데 harness 가
	// 그대로 실렸다. 그 VM 의 claude 는 {"loggedIn":false} 를 분명히 찍고
	// 있었고, 우리가 Output() 으로 종료코드를 보느라 안 읽은 것이다.
	bin = fakeClaude(t, `{"loggedIn":false}`, true, true)
	_, err = (claudeHarness{}).Probe(ctx, bin)
	if !errors.Is(err, errNotUsable) {
		t.Fatalf("종료코드가 0 이 아니라고 판정을 포기했다: %v", err)
	}

	// ④ 모르면 「쓸 수 있다」로 본다 — 옛 CLI 는 이 하위명령이 없고,
	// 형식이 바뀔 수도 있다. 노드가 통째로 사라지면 안 된다.
	for _, tc := range []struct {
		name, out string
		ok        bool
	}{
		{"하위명령이 없다", "", false},
		{"JSON 이 아니다", "그런 명령 없음", true},
		{"loggedIn 이 없다", `{"authMethod":"claude.ai"}`, true},
	} {
		bin = fakeClaude(t, tc.out, tc.ok)
		_, err := (claudeHarness{}).Probe(ctx, bin)
		if errors.Is(err, errNotUsable) {
			t.Fatalf("%s — 모르는 것을 못 쓴다고 판정했다", tc.name)
		}
	}
}

// 못 쓰는 하네스는 광고에서 빠지고, 그 사실이 로그에 남는다
//
// ADR-012: "못 하는 것을 빼고 보내는 것이 「지금은 못 한다」를 표현하는 방법".
// 그런데 조용히 빠지면 사람이 원인을 못 찾는다 — 없는 것과 못 쓰는 것은 다르다.
func Test못_쓰는_하네스는_광고에_안_실린다(t *testing.T) {
	bin := fakeClaude(t, `{"loggedIn":false}`, true)
	caps := Detect(Local{HarnessBin: bin, Arch: "arm64", Workspace: t.TempDir()},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	for _, c := range caps {
		if _, has := c.Attrs["harness"]; has {
			t.Fatalf("못 쓰는 하네스가 광고에 실렸다: %+v", c.Attrs)
		}
	}
	// 다른 능력은 남는다 — 하네스를 못 쓴다고 빌드까지 못 하는 것은 아니다.
	if len(caps) == 0 {
		t.Fatal("arch 가 있는데 광고가 통째로 비었다")
	}
}
