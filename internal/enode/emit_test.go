package enode

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/taeels/enode/internal/transcript"
)

// collector 는 사건을 모은다. emit 은 배출기의 고루틴에서 불리므로 잠근다.
type collector struct {
	mu     sync.Mutex
	kinds  []EventKind
	bodies []string
}

func (c *collector) emit(e Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.kinds = append(c.kinds, e.Kind)
	c.bodies = append(c.bodies, e.Text)
}

func (c *collector) seen() []EventKind {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]EventKind(nil), c.kinds...)
}

const textLine = `{"type":"assistant","message":{"content":[{"type":"text","text":"hello"}]}}`

// R6 — Write 는 언제나 성공한다. emit 이 패닉을 내도 그렇다.
//
// 오류를 내면 io.MultiWriter 가 거기서 멈추고, 멈추면 하네스의 stdout 이
// 막힌다. 관측하는 겹이 실행을 막으면 그것은 관측이 아니다.
func TestLineEmitter_WriteNeverFails(t *testing.T) {
	e := newLineEmitter(func(Event) { panic("the consumer blew up") })
	in := []byte(textLine + "\n" + textLine + "\n")
	n, err := e.Write(in)
	if n != len(in) || err != nil {
		t.Fatalf("Write reported a failure: %d %v", n, err)
	}
	if dropped := e.Close(); dropped != 0 {
		t.Fatalf("a panic in the consumer was counted as a drop: %d", dropped)
	}
}

// R7 — 대기열이 차면 버리고 센다. 막지 않는다.
func TestLineEmitter_DropsWhenFull(t *testing.T) {
	release := make(chan struct{})
	var once sync.Once
	e := newLineEmitter(func(Event) {
		once.Do(func() { <-release })
	})
	// 첫 줄에서 소비가 멈춘 채로 대기열보다 훨씬 많이 민다.
	for i := 0; i < emitQueueDepth*2; i++ {
		if _, err := e.Write([]byte(textLine + "\n")); err != nil {
			t.Fatalf("Write failed at line %d: %v", i, err)
		}
	}
	close(release)
	if dropped := e.Close(); dropped == 0 {
		t.Fatal("a full queue dropped nothing - the bound is not doing anything")
	}
}

// R8 — Close 는 개행 없이 끝난 꼬리를 마지막 줄로 밀고, 그 뒤에는 사건이 0 이다.
func TestLineEmitter_FlushesTailAndThenIsQuiet(t *testing.T) {
	var c collector
	e := newLineEmitter(c.emit)
	// 개행이 없다 - 꼬리로 남는다.
	if _, err := e.Write([]byte(textLine)); err != nil {
		t.Fatal(err)
	}
	if got := len(c.seen()); got != 0 {
		t.Fatalf("an unterminated line became an event too early: %d", got)
	}
	e.Close()
	after := c.seen()
	if len(after) != 1 || after[0] != transcript.KindText {
		t.Fatalf("the tail did not become the last line: %v", after)
	}
	// Close 뒤의 쓰기는 아무것도 안 낸다.
	if _, err := e.Write([]byte(textLine + "\n")); err != nil {
		t.Fatal(err)
	}
	e.Close()
	if got := c.seen(); len(got) != 1 {
		t.Fatalf("events kept coming after Close: %v", got)
	}
}

// R5 — 줄마다 나는 사건은 본문을 안 싣는다. 이것을 받는 것이 노드 로그다.
func TestLineEmitter_CarriesNoBody(t *testing.T) {
	var c collector
	e := newLineEmitter(c.emit)
	secret := `{"type":"assistant","message":{"content":[{"type":"text","text":"a token"}]}}`
	if _, err := e.Write([]byte(secret + "\n")); err != nil {
		t.Fatal(err)
	}
	e.Close()
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.bodies) == 0 {
		t.Fatal("no event came")
	}
	for i, b := range c.bodies {
		if b != "" {
			t.Fatalf("event %d carried a body: %q", i, b)
		}
	}
}

// 한 줄을 모으는 데도 상한이 있다. 없으면 이 겹이 원문을 통째로 또 든다.
func TestLineEmitter_DropsAnOverlongLine(t *testing.T) {
	var c collector
	e := newLineEmitter(c.emit)
	if _, err := e.Write([]byte(strings.Repeat("x", maxEmitLineBytes+1))); err != nil {
		t.Fatal(err)
	}
	// 그 줄이 끝나고 다음 줄이 오면 다시 읽는다.
	if _, err := e.Write([]byte("\n" + textLine + "\n")); err != nil {
		t.Fatal(err)
	}
	if dropped := e.Close(); dropped == 0 {
		t.Fatal("the overlong line was not dropped")
	}
	got := c.seen()
	if len(got) != 1 || got[0] != transcript.KindText {
		t.Fatalf("the next line did not recover: %v", got)
	}
}

// fakeHarnessScript 는 --version 에 버전을, 그 밖에는 준 줄들을 찍는다.
func fakeHarnessScript(t *testing.T, lines ...string) string {
	t.Helper()
	dir := t.TempDir()
	path := dir + "/fake"
	script := "#!/bin/sh\n" +
		"case \"$1\" in --version) echo '9.9.9 (fake)'; exit 0;; esac\n"
	for _, ln := range lines {
		script += "cat <<'EOJSON'\n" + ln + "\nEOJSON\n"
	}
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// R4 — 한 단계의 사건은 정확히 한 번 난다.
//
// 배출기와 Decode 가 둘 다 j.Emit 을 받으면 두 배가 된다. 오늘은 로그가 두
// 줄이 되는 것으로 보이지만, 이 자리에 업로더가 붙는 날 같은 바이트가 두 번
// 올라간다.
func TestRunHarness_EmitsEachEventOnce(t *testing.T) {
	var c collector
	dir := t.TempDir()
	fake := fakeHarnessScript(t, textLine, textLine,
		`{"type":"result","subtype":"success","num_turns":1}`)
	// 줄 사건을 느리게 받는다 - 순서가 우연이 아니게 된다.
	//
	// final 은 판정이 끝난 뒤에 나므로 언제나 마지막이어야 한다. 배출기를
	// Decode 뒤에 닫으면 아직 흐르고 있는 줄 사건을 final 이 앞질러 간다.
	// 늦추지 않으면 그 추월이 대개 안 보인다.
	slow := func(e Event) {
		if e.Kind != EventFinal {
			time.Sleep(10 * time.Millisecond)
		}
		c.emit(e)
	}
	_, h := runHarness(context.Background(), claudeHarness{}, fake, Job{
		Prompt: "x", IO: IOPaths{Dir: dir, In: dir, Out: dir}, Emit: slow})
	if h.Reason != ReasonOK {
		t.Fatalf("the fake harness did not complete: %+v", h)
	}
	got := c.seen()
	want := []EventKind{transcript.KindText, transcript.KindText, transcript.KindResult, EventFinal}
	if len(got) != len(want) {
		t.Fatalf("the events did not come exactly once: %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("event %d is %q, want %q (all: %v)", i, got[i], want[i], got)
		}
	}
	// final 이 마지막이다 - 판정은 흐름이 끝난 뒤의 일이다.
	if got[len(got)-1] != EventFinal {
		t.Fatalf("the envelope event overtook the lines that were still flowing: %v", got)
	}
}

// R1 — 하네스 단계의 tee 가 되살아났다. 링이 원문을 그대로 받는다.
func TestRunHarness_TeesTheRawStreamToTheRing(t *testing.T) {
	dir := t.TempDir()
	fake := fakeHarnessScript(t, textLine, `{"type":"result","subtype":"success"}`)
	var ring strings.Builder
	out, h := runHarness(context.Background(), claudeHarness{}, fake, Job{
		Prompt: "x", IO: IOPaths{Dir: dir, In: dir, Out: dir}, Transcript: &ring})
	if h.Reason != ReasonOK {
		t.Fatalf("the fake harness did not complete: %+v", h)
	}
	if !strings.Contains(ring.String(), `"type":"assistant"`) {
		t.Fatalf("the raw stream did not reach the ring: %q", ring.String())
	}
	// 선별을 링에 안 건다 - 본문이 그대로 있어야 화면이 읽을 문장이 있다.
	if !strings.Contains(ring.String(), "hello") {
		t.Fatal("the body was filtered on its way to the ring")
	}
	// 봉투 파서는 같은 바이트를 여전히 본다.
	if len(out) == 0 {
		t.Fatal("the selected log is empty")
	}
}

// R1 - Transcript 가 nil 이면 그 갈래만 빠지고 나머지는 그대로 돈다.
func TestRunHarness_WithoutARingItStillRuns(t *testing.T) {
	var c collector
	dir := t.TempDir()
	fake := fakeHarnessScript(t, `{"type":"result","subtype":"success"}`)
	_, h := runHarness(context.Background(), claudeHarness{}, fake, Job{
		Prompt: "x", IO: IOPaths{Dir: dir, In: dir, Out: dir}, Emit: c.emit})
	if h.Reason != ReasonOK {
		t.Fatalf("%+v", h)
	}
	if got := c.seen(); len(got) != 2 {
		t.Fatalf("events changed when the ring was absent: %v", got)
	}
}

// R10 — EventKind 가 transcript.Kind 다. 어휘의 정본이 하나다.
func TestEventKind_IsTheTranscriptVocabulary(t *testing.T) {
	var k EventKind = transcript.KindToolUse
	var back transcript.Kind = k
	if back != transcript.KindToolUse {
		t.Fatalf("the two vocabularies drifted: %q", back)
	}
}

// R11 — final 은 하네스가 내는 종류가 아니다. 일곱 중 어느 것과도 안 겹친다.
//
// 겹치면 화면이 봉투를 하네스 사건으로 읽는다.
func TestEventFinal_DoesNotCollideWithTheSeven(t *testing.T) {
	for _, k := range []transcript.Kind{
		transcript.KindInit, transcript.KindText, transcript.KindToolUse,
		transcript.KindToolResult, transcript.KindResult, transcript.KindRaw,
		transcript.KindCapped,
	} {
		if EventFinal == k {
			t.Fatalf("EventFinal collides with %q", k)
		}
	}
}

// 사건이 단계가 끝나기 전에 난다 — 이 유닛이 사는 값이다.
//
// 수를 세는 시험만으로는 이것이 안 잡힌다. 배출기를 통째로 들어내도 Decode
// 혼자서 같은 종류를 같은 순서로 내므로 수와 순서가 그대로다 — 실제로 한 번
// 밟았다. 다른 것은 시점뿐이고, 시점은 세서 못 잰다.
//
// 그래서 하네스가 사건을 기다리게 만든다. 첫 사건이 나야 문지기 파일이
// 생기고, 그것이 생겨야 하네스가 다음 줄을 찍고 끝난다. 배출이 끝난 뒤에만
// 난다면 하네스는 영영 안 끝나고 ctx 가 그것을 죽인다.
func TestRunHarness_EventsArriveBeforeTheStepEnds(t *testing.T) {
	dir := t.TempDir()
	gate := filepath.Join(dir, "gate")
	path := filepath.Join(dir, "fake")
	script := "#!/bin/sh\n" +
		"case \"$1\" in --version) echo '9.9.9 (fake)'; exit 0;; esac\n" +
		"cat <<'EOJSON'\n" + textLine + "\nEOJSON\n" +
		"while [ ! -f " + gate + " ]; do sleep 0.05; done\n" +
		"cat <<'EOJSON'\n" + `{"type":"result","subtype":"success"}` + "\nEOJSON\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	var once sync.Once
	var c collector
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	_, h := runHarness(ctx, claudeHarness{}, path, Job{
		Prompt: "x", IO: IOPaths{Dir: dir, In: dir, Out: dir},
		Emit: func(e Event) {
			c.emit(e)
			once.Do(func() { _ = os.WriteFile(gate, []byte("go"), 0o600) })
		}})
	if h.Reason != ReasonOK {
		t.Fatalf("no event reached the consumer while the harness was still running: %+v", h)
	}
	got := c.seen()
	if len(got) == 0 || got[0] != transcript.KindText {
		t.Fatalf("the first live event was not the first line: %v", got)
	}
}
