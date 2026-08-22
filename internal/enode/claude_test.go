package enode

import (
	"context"
	"os"
	"strings"
	"testing"
)

// Argv 가 ★ 순수 함수 ★ 라 프로세스 없이 시험된다 — 그게 exec 을 뺀 이유다.
func Test어댑터_Argv는_순수하다(t *testing.T) {
	got := claudeHarness{}.Argv(AgentParams{Model: "opus", MaxTurns: 12},
		IOPaths{Dir: "/ws", Out: "/o", In: "/i"})
	want := "-p --output-format json --model opus --max-turns 12 " +
		"--permission-mode bypassPermissions --add-dir /o --add-dir /i"
	if strings.Join(got, " ") != want {
		t.Fatalf("\n얻음: %s\n원함: %s", strings.Join(got, " "), want)
	}
}

// ★ 경계는 단계가 아니라 노드에 있다 ★ (ADR-042)
//
// 이 시험은 ★ 뒤집힌 결정 ★ 이다. 예전에는 "bypassPermissions 가 있으면 실패" 였다.
// 근거였던 포함관계(모델이 쓸 수 있는 곳 ⊆ 훅이 볼 수 있는 곳)는 명령 단계가
// ★ 이미 무경계 ★ 이므로 agent 단계에만 걸어도 위협이 안 좁아진다는 것이
// 드러나 기각됐다. 좁은 쪽이 산 것은 안전이 아니라 ★ 에이전트가 명령을 못 돌린다 ★ 였다.
//
// ★ 되돌리려면 여기부터 본다 ★ — 명령 단계에 파일시스템 경계가 생기면
// (INVARIANTS 의 그 줄이 바뀌면) 이 결정의 전제가 사라진다.
func Test어댑터_경계는_노드에_있다(t *testing.T) {
	got := strings.Join(claudeHarness{}.Argv(AgentParams{},
		IOPaths{Dir: "/ws", Out: "/o", In: "/i"}), " ")
	if !strings.Contains(got, "--permission-mode bypassPermissions") {
		t.Fatalf("★ 에이전트가 명령을 못 돌리면 환경 구성 시나리오가 안 돈다 ★: %s", got)
	}
	// ★ --add-dir 은 남는다 ★ — 권한상 불필요하지만 ★ 의도가 argv 에 남아야 ★
	// 이 단계가 어디를 쓸 셈이었는지가 Record 의 하네스 로그에 찍힌다.
	for _, d := range []string{"--add-dir /o", "--add-dir /i"} {
		if !strings.Contains(got, d) {
			t.Fatalf("★ %s 가 없다 — 의도가 기록에서 사라진다 ★: %s", d, got)
		}
	}
	// cwd 는 기본으로 열리므로 중복해서 열지 않는다.
	if strings.Contains(got, "--add-dir /ws") {
		t.Fatalf("cwd 를 중복으로 열었다: %s", got)
	}
}

// ★ $IN 잠금이 이제 유일한 기계적 방어다 ★ (ADR-042)
//
// 권한 모드가 아니라 ★ 파일시스템 ★ 이 거는 것이라 이번 뒤집기에 안 묶인다.
// 이 시험이 없으면 sealInput 이 조용히 죽어도 아무도 모른다 —
// 예전에는 --add-dir 이 같이 막았지만 ★ 이제는 이것뿐 ★ 이다.
func Test잠금_IN은_읽기전용이다(t *testing.T) {
	dir := t.TempDir()
	// ★ 잠금을 되돌리는 정리를 먼저 등록한다 ★ — t.Cleanup 은 LIFO 라
	// TempDir 이 등록한 삭제보다 ★ 나중에 등록한 이것이 먼저 ★ 돈다.
	// 안 그러면 0555 때문에 unlink 가 막혀 ★ 시험이 통과해도 FAIL 로 보인다 ★.
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	if err := os.WriteFile(dir+"/diff", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if why := sealInput(dir); why != "" {
		t.Fatalf("잠그지 못했다: %s", why)
	}
	fi, err := os.Stat(dir + "/diff")
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm()&0o222 != 0 {
		t.Fatalf("★ 쓰기 비트가 남아 있다 ★: %04o", fi.Mode().Perm())
	}
}

// ★ 도구를 나열하지 않는다 ★ — --allowed-tools 는 제한이 아니라 자동승인 목록이라
// "Read" 를 넣으면 경로 조건 없이 승인돼 --add-dir 경계를 덮어쓴다 (실측).
func Test어댑터_도구를_나열하지_않는다(t *testing.T) {
	got := strings.Join(claudeHarness{}.Argv(AgentParams{}, IOPaths{Dir: "/ws", Out: "/o"}), " ")
	if strings.Contains(got, "--allowed-tools") {
		t.Fatal("★ 도구를 나열하면 --add-dir 경계가 덮어써진다 ★")
	}
}

// ask 가 비었거나 never 면 ★ 안 묻는다 ★ — 무인 실행에서 물으면 매달린다.
func Test어댑터_무인이면_안_묻는다(t *testing.T) {
	for _, ask := range []string{"", "never"} {
		if !strings.Contains(strings.Join(claudeHarness{}.Argv(AgentParams{Ask: ask}, IOPaths{}), " "),
			"--permission-mode") {
			t.Fatalf("ask=%q 인데 권한을 물으려 한다", ask)
		}
	}
}

// ★ 모르는 하네스를 조용히 claude 로 떨어뜨리지 않는다 ★
// 계약이 요구한 것과 다른 것으로 돌면 Record 가 거짓을 남긴다.
func Test어댑터_모르는_이름은_없다고_한다(t *testing.T) {
	if _, ok := harnessFor("claude"); !ok {
		t.Fatal("claude 가 등록돼 있어야 한다")
	}
	if h, ok := harnessFor("openhands"); ok {
		t.Fatalf("★ 없는 하네스를 찾아줬다 ★: %v", h.Name())
	}
}

// Probe 가 ★ 곧 executable resolve 다 ★ — 없으면 err, 있으면 버전.
func Test어댑터_Probe가_resolve다(t *testing.T) {
	if _, err := (claudeHarness{}).Probe(context.Background(), "이런건-없다-확실히"); err == nil {
		t.Fatal("★ 없는 실행파일을 있다고 했다 ★ — 광고에 실리면 시연장에서 터진다")
	}
	if _, err := execLookPath("claude"); err != nil {
		t.Skip("claude 가 이 기계에 없다")
	}
	v, err := (claudeHarness{}).Probe(context.Background(), "claude")
	if err != nil || v == "" {
		t.Fatalf("버전을 못 얻었다: %q %v", v, err)
	}
}

// 버전이 ★ 결과에 실린다 ★ — 봉투 드리프트를 봉인된 기록만 보고 알기 위해서다.
func Test어댑터_버전이_결과에_실린다(t *testing.T) {
	dir := t.TempDir()
	fake := dir + "/fake"
	// --version 이면 버전만, 아니면 봉투를 찍는다.
	script := "#!/bin/sh\n" +
		"case \"$1\" in --version) echo '9.9.9 (가짜)'; exit 0;; esac\n" +
		`printf '{"type":"result","subtype":"success","num_turns":2}` + "\n'\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	_, h := runHarness(context.Background(), claudeHarness{}, fake, Job{
		Prompt: "x", IO: IOPaths{Dir: dir, In: dir, Out: dir}})
	if h.Version != "9.9.9 (가짜)" {
		t.Fatalf("★ 버전이 기록에 안 실렸다 ★: %q", h.Version)
	}
	if h.Reason != ReasonOK || h.Turns != 2 {
		t.Fatalf("%+v", h)
	}
}

// 사건이 ★ 흘러나온다 ★ — 배치는 스트리밍의 퇴화형이라 지금은 final 하나다.
func Test어댑터_사건을_낸다(t *testing.T) {
	var kinds []EventKind
	claudeHarness{}.Decode(strings.NewReader(
		`{"type":"result","subtype":"success"}`), 0, func(e Event) { kinds = append(kinds, e.Kind) })
	if len(kinds) != 1 || kinds[0] != EventFinal {
		t.Fatalf("사건이 안 나왔다: %v", kinds)
	}
}
