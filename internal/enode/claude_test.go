package enode

import (
	"context"
	"os"
	"strings"
	"testing"
)

// Argv 가 순수 함수라 프로세스 없이 시험된다 — 그게 exec 을 뺀 이유다.
func TestAdapter_ArgvIsPure(t *testing.T) {
	got := claudeHarness{}.Argv(AgentParams{Model: "opus", MaxTurns: 12},
		IOPaths{Dir: "/ws", Out: "/o", In: "/i"})
	// stream-json 이다 (decisions.md 6절 ⑮) — 게이트가 재는 system/init 줄이
	// 그래야 나온다. --verbose 가 있어야 -p 아래서 사건이 흐른다.
	want := "-p --output-format stream-json --verbose --model opus --max-turns 12 " +
		"--permission-mode bypassPermissions --add-dir /o --add-dir /i"
	if strings.Join(got, " ") != want {
		t.Fatalf("\ngot:  %s\nwant: %s", strings.Join(got, " "), want)
	}
}

// 경계는 단계가 아니라 노드에 있다 (ADR-042)
//
// 이 시험은 뒤집힌 결정이다. 예전에는 "bypassPermissions 가 있으면 실패" 였다.
// 근거였던 포함관계(모델이 쓸 수 있는 곳 ⊆ 훅이 볼 수 있는 곳)는 명령 단계가
// 이미 무경계 이므로 agent 단계에만 걸어도 위협이 안 좁아진다는 것이
// 드러나 기각됐다. 좁은 쪽이 산 것은 안전이 아니라 에이전트가 명령을 못 돌린다였다.
//
// 되돌리려면 여기부터 본다 — 명령 단계에 파일시스템 경계가 생기면
// (INVARIANTS 의 그 줄이 바뀌면) 이 결정의 전제가 사라진다.
func TestAdapter_TheBoundaryLivesOnTheNode(t *testing.T) {
	got := strings.Join(claudeHarness{}.Argv(AgentParams{},
		IOPaths{Dir: "/ws", Out: "/o", In: "/i"}), " ")
	if !strings.Contains(got, "--permission-mode bypassPermissions") {
		t.Fatalf("if the agent cannot run commands the environment scenario cannot run: %s", got)
	}
	// --add-dir 은 남는다 — 권한상 불필요하지만 의도가 argv 에 남아야
	// 이 단계가 어디를 쓸 셈이었는지가 Record 의 하네스 로그에 찍힌다.
	for _, d := range []string{"--add-dir /o", "--add-dir /i"} {
		if !strings.Contains(got, d) {
			t.Fatalf("%s is missing — the intent disappears from the record: %s", d, got)
		}
	}
	// cwd 는 기본으로 열리므로 중복해서 열지 않는다.
	if strings.Contains(got, "--add-dir /ws") {
		t.Fatalf("cwd was opened twice: %s", got)
	}
}

// $IN 잠금이 이제 유일한 기계적 방어다 (ADR-042)
//
// 권한 모드가 아니라 파일시스템이 거는 것이라 이번 뒤집기에 안 묶인다.
// 이 시험이 없으면 sealInput 이 조용히 죽어도 아무도 모른다 —
// 예전에는 --add-dir 이 같이 막았지만 이제는 이것뿐이다.
func TestLock_INIsReadOnly(t *testing.T) {
	dir := t.TempDir()
	// 잠금을 되돌리는 정리를 먼저 등록한다 — t.Cleanup 은 LIFO 라
	// TempDir 이 등록한 삭제보다 나중에 등록한 이것이 먼저 돈다.
	// 안 그러면 0555 때문에 unlink 가 막혀 시험이 통과해도 FAIL 로 보인다.
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	if err := os.WriteFile(dir+"/diff", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if why := sealInput(dir); why != "" {
		t.Fatalf("could not lock: %s", why)
	}
	fi, err := os.Stat(dir + "/diff")
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm()&0o222 != 0 {
		t.Fatalf("a write bit survived: %04o", fi.Mode().Perm())
	}
}

// 도구를 나열하지 않는다 — --allowed-tools 는 제한이 아니라 자동승인 목록이라
// "Read" 를 넣으면 경로 조건 없이 승인돼 --add-dir 경계를 덮어쓴다 (실측).
func TestAdapter_DoesNotListTools(t *testing.T) {
	got := strings.Join(claudeHarness{}.Argv(AgentParams{}, IOPaths{Dir: "/ws", Out: "/o"}), " ")
	if strings.Contains(got, "--allowed-tools") {
		t.Fatal("listing tools overwrites the --add-dir boundary")
	}
}

// ask 가 비었거나 never 면 안 묻는다 — 무인 실행에서 물으면 매달린다.
func TestAdapter_UnattendedNeverAsks(t *testing.T) {
	for _, ask := range []string{"", "never"} {
		if !strings.Contains(strings.Join(claudeHarness{}.Argv(AgentParams{Ask: ask}, IOPaths{}), " "),
			"--permission-mode") {
			t.Fatalf("ask=%q yet it tries to ask for permission", ask)
		}
	}
}

// 모르는 하네스를 조용히 claude 로 떨어뜨리지 않는다
// 계약이 요구한 것과 다른 것으로 돌면 Record 가 거짓을 남긴다.
func TestAdapter_AnUnknownNameIsReportedAbsent(t *testing.T) {
	if _, ok := harnessFor("claude"); !ok {
		t.Fatal("claude must be registered")
	}
	if h, ok := harnessFor("openhands"); ok {
		t.Fatalf("found a harness that does not exist: %v", h.Name())
	}
}

// Usable 이 곧 executable resolve 다 — 없으면 err. Version 도 같은 해석을 쓴다.
func TestAdapter_UsableIsResolve(t *testing.T) {
	if err := (claudeHarness{}).Usable(context.Background(), "definitely-no-such-binary"); err == nil {
		t.Fatal("claimed a missing binary exists — riding the advert, it blows up on stage")
	}
	if _, err := (claudeHarness{}).Version(context.Background(), "definitely-no-such-binary"); err == nil {
		t.Fatal("gave a version for a binary that is not there")
	}
	if _, err := execLookPath("claude"); err != nil {
		t.Skip("claude is not on this machine")
	}
	v, err := (claudeHarness{}).Version(context.Background(), "claude")
	if err != nil || v == "" {
		t.Fatalf("could not get the version: %q %v", v, err)
	}
}

// 버전이 결과에 실린다 — 봉투 드리프트를 봉인된 기록만 보고 알기 위해서다.
func TestAdapter_TheVersionRidesTheResult(t *testing.T) {
	dir := t.TempDir()
	fake := dir + "/fake"
	// --version 이면 버전만, 아니면 봉투를 찍는다.
	script := "#!/bin/sh\n" +
		"case \"$1\" in --version) echo '9.9.9 (fake)'; exit 0;; esac\n" +
		`printf '{"type":"result","subtype":"success","num_turns":2}` + "\n'\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	_, h := runHarness(context.Background(), claudeHarness{}, fake, Job{
		Prompt: "x", IO: IOPaths{Dir: dir, In: dir, Out: dir}})
	if h.Version != "9.9.9 (fake)" {
		t.Fatalf("the version did not reach the record: %q", h.Version)
	}
	if h.Reason != ReasonOK || h.Turns != 2 {
		t.Fatalf("%+v", h)
	}
}

// 사건이 흘러나온다 — 배치는 스트리밍의 퇴화형이라 지금은 final 하나다.
func TestAdapter_EmitsEvents(t *testing.T) {
	var kinds []EventKind
	claudeHarness{}.Decode(strings.NewReader(
		`{"type":"result","subtype":"success"}`), 0, func(e Event) { kinds = append(kinds, e.Kind) })
	if len(kinds) != 1 || kinds[0] != EventFinal {
		t.Fatalf("no events came: %v", kinds)
	}
}
