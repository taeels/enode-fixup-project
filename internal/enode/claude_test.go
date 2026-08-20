package enode

import (
	"context"
	"os"
	"strings"
	"testing"
)

// Argv 가 ★ 순수 함수 ★ 라 프로세스 없이 시험된다 — 그게 exec 을 뺀 이유다.
func Test어댑터_Argv는_순수하다(t *testing.T) {
	got := claudeHarness{}.Argv(AgentParams{Model: "opus", MaxTurns: 12}, IOPaths{})
	want := "-p --output-format json --model opus --max-turns 12 --permission-mode bypassPermissions"
	if strings.Join(got, " ") != want {
		t.Fatalf("\n얻음: %s\n원함: %s", strings.Join(got, " "), want)
	}
}

// ask 가 비었거나 never 면 ★ 안 묻는다 ★ — 무인 실행에서 물으면 매달린다.
func Test어댑터_무인이면_안_묻는다(t *testing.T) {
	for _, ask := range []string{"", "never"} {
		if !strings.Contains(strings.Join(claudeHarness{}.Argv(AgentParams{Ask: ask}, IOPaths{}), " "),
			"bypassPermissions") {
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
