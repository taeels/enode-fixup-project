package enode

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// recordingClaude 는 자기가 어떤 인자로 불렸는지 파일에 적는 가짜다.
func recordingClaude(t *testing.T, dir, callLog string) string {
	t.Helper()
	p := filepath.Join(dir, "claude")
	script := "#!/bin/sh\n" +
		"echo \"$@\" >> " + shQuote(callLog) + "\n" +
		"case \"$1 $2\" in\n" +
		"  '--version ') echo 'fake 9.9.9 (Claude Code)'; exit 0 ;;\n" +
		"  'auth status') echo '{\"loggedIn\":true}'; exit 0 ;;\n" +
		"esac\nexit 0\n"
	if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// 광고 경로가 띄우는 자식 프로세스를 센다.
//
// 왜 세나 — 이 비용은 어디서도 빨개지지 않는다. 프로세스가 하나 더 떠도
// 시험은 통과하고 로그도 조용하다. 리눅스와 맥에서는 아무 흔적이 없고,
// 윈도우에서 콘솔 창이 깜빡이는 것으로 사람이 알아차릴 때까지 남는다.
//
//	실측 (2026-09-05) 윈도우 노드가 60초마다 창을 셋 띄웠다. 둘이
//	하네스 탐지였고 그중 하나는 돌려받은 버전을 그 자리에서 버렸다.
//	버리는 값을 위해 프로세스가 매 분 하나씩 떴다.
//
// console_test.go 가 감싸지 않은 실행 자리를 세는 것과 같은 성질이다 —
// 지켜졌는지 셀 수 없는 규칙은 지켜지지 않는다.
func TestDetectSpawnsOneProcessPerHarness(t *testing.T) {
	dir := t.TempDir()
	callLog := filepath.Join(dir, "calls")
	bin := recordingClaude(t, dir, callLog)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	// 워크스페이스를 안 준다 — 여기서 세는 것은 하네스 탐지의 비용이다.
	caps := Detect(context.Background(), Local{HarnessBin: bin}, log)
	if len(caps) == 0 || caps[0].Attrs["harness"] != "claude" {
		t.Fatalf("the usable harness did not ride the advert: %+v", caps)
	}

	b, err := os.ReadFile(callLog)
	if err != nil {
		t.Fatalf("the harness was never run: %v", err)
	}
	calls := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(calls) != 1 {
		t.Fatalf("detecting one harness ran %d processes, want 1:\n  %s",
			len(calls), strings.Join(calls, "\n  "))
	}
	// 무엇을 물었는지도 본다 — 광고가 쓰는 것은 「쓸 수 있는가」뿐이다.
	if !strings.Contains(calls[0], "auth status") {
		t.Fatalf("the advert path asked something it does not use: %q", calls[0])
	}
}

// 탐지가 안 돌아와도 부르는 쪽이 함께 멈추지 않는다.
//
// 왜 중요한가 — Detect 는 광고 루프 안에서 돈다. 여기가 멈추면 하트비트가
// 안 나가고, 노드는 프로세스도 로그도 정상인 채로 함대에서 사라진다.
// ADR-028 이 그 증상을 이미 겪었고 진단에 세 번 헛짚었는데, 그때 첫 번째로
// 의심한 것이 이 자리였다. 그때는 원인이 아니었지만 구멍은 남아 있었다 —
// Detect 가 context.Background() 를 쓰고 있어서 취소가 닿지 않았다.
func TestDetectComesBackWhenItsContextIsCancelled(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "claude")
	// 무엇을 물어도 안 돌아오는 하네스.
	if err := os.WriteFile(p, []byte("#!/bin/sh\nsleep 60\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		Detect(ctx, Local{HarnessBin: p}, log)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Detect outlived its context — the advertise loop would be stuck with it")
	}
}

// 광고가 읽는 자리는 프로세스를 안 띄운다 (ADR-068).
//
// 여기가 이 갈래의 요점이다. 예전에는 광고 루프가 직접 탐지했고, 그래서
// 광고 주기가 곧 프로세스 기동 주기였다. Mediator 가 renew_seconds 를
// 낮추면 아무도 고른 적 없는 경로로 하네스 탐지까지 촘촘해졌다.
func TestReadingTheCapabilitiesSpawnsNothing(t *testing.T) {
	dir := t.TempDir()
	callLog := filepath.Join(dir, "calls")
	bin := recordingClaude(t, dir, callLog)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	d := NewDetector(context.Background(), Local{HarnessBin: bin}, time.Hour, log)
	before := countLines(t, callLog)

	// 광고가 열 번 돈 셈 친다.
	for i := 0; i < 10; i++ {
		if got := d.Capabilities(); len(got.Caps) == 0 {
			t.Fatal("the advert went out empty")
		}
	}

	if after := countLines(t, callLog); after != before {
		t.Fatalf("reading the capabilities ran %d more processes; the advert loop must spawn none",
			after-before)
	}
}

// 탐지가 안 돌아와도 광고는 마지막 값을 들고 나간다 (ADR-068).
//
// 예전에는 이 자리에서 노드가 조용히 사라졌다 — 탐지가 멈추면 하트비트가
// 함께 멈추고, 프로세스도 로그도 정상인 채로 광고가 만료된다. ADR-028 이
// 겪은 증상이 그것이고, 그때 첫 번째로 의심한 것이 하네스 탐지였다.
func TestTheAdvertDoesNotWaitForADetectionThatHangs(t *testing.T) {
	dir := t.TempDir()
	counter := filepath.Join(dir, "n")
	p := filepath.Join(dir, "claude")
	// 첫 번째는 즉시 답하고, 그 뒤로는 안 돌아온다.
	script := "#!/bin/sh\n" +
		"n=$(cat " + shQuote(counter) + " 2>/dev/null || echo 0)\n" +
		"n=$((n+1)); echo $n > " + shQuote(counter) + "\n" +
		"[ \"$n\" -gt 1 ] && sleep 60\n" +
		"echo '{\"loggedIn\":true}'\n"
	if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d := NewDetector(ctx, Local{HarnessBin: p}, 20*time.Millisecond, log)
	go d.Run(ctx)

	// 두 번째 탐지가 시작되어 걸릴 때까지 기다린다.
	deadline := time.Now().Add(3 * time.Second)
	for countLines(t, counter) < 1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	// 그 사이에도 광고는 즉시 답해야 하고, 마지막으로 알아낸 것을 들고 있어야 한다.
	done := make(chan Capabilities, 1)
	go func() { done <- d.Capabilities() }()
	select {
	case got := <-done:
		if len(got.Caps) == 0 || got.Caps[0].Attrs["harness"] != "claude" {
			t.Fatalf("the advert lost what it already knew: %+v", got.Caps)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("reading the capabilities blocked on a hanging detection")
	}
}

func countLines(t *testing.T, path string) int {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return 0
	}
	return len(strings.Split(s, "\n"))
}
