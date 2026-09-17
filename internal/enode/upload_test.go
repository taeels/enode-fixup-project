package enode

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// 진행 청크를 받는 가짜 Mediator.
//
// 실제 서버의 규율만 흉내 낸다 - 이어 붙이고, 총 길이를 헤더로 돌려주고,
// 시도를 그대로 되비춘다. 나머지(시도 걷기 · 상한)는 시험마다 따로 건다.
type fakeMediator struct {
	mu      sync.Mutex
	got     bytes.Buffer
	calls   int
	queries []string

	failNext  atomic.Int32 // 이만큼의 다음 호출을 500 으로 떨어뜨린다
	capped    atomic.Bool
	sayAttemt atomic.Int32 // 응답에 실을 시도. -1 이면 받은 값 그대로
}

func newFakeMediator(t *testing.T) (*fakeMediator, *Client) {
	t.Helper()
	f := &fakeMediator{}
	f.sayAttemt.Store(-1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.calls++
		f.queries = append(f.queries, r.URL.RawQuery)
		f.mu.Unlock()

		if f.failNext.Load() > 0 {
			f.failNext.Add(-1)
			w.WriteHeader(500)
			return
		}
		f.mu.Lock()
		f.got.Write(body)
		total := f.got.Len()
		f.mu.Unlock()

		at := r.URL.Query().Get("attempt")
		if say := f.sayAttemt.Load(); say >= 0 {
			at = strconv.Itoa(int(say))
		}
		w.Header().Set("X-Enode-Log-Bytes", strconv.Itoa(total))
		w.Header().Set("X-Enode-Log-Attempt", at)
		if f.capped.Load() {
			w.Header().Set("X-Enode-Log-Capped", "1")
		}
		w.WriteHeader(200)
	}))
	t.Cleanup(srv.Close)
	return f, &Client{Base: srv.URL, HTTP: srv.Client()}
}

func (f *fakeMediator) bytesGot() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.got.String()
}

func (f *fakeMediator) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func (f *fakeMediator) lastQuery() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.queries) == 0 {
		return ""
	}
	return f.queries[len(f.queries)-1]
}

func quietLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func testStep() *Step {
	return &Step{RunID: "r1", Seq: 1, Name: "build", StepID: "r1#01", Attempt: 0}
}

// 청크가 실제로 올라가고 쿼리 셋이 맞다. Close 가 마지막으로 비우므로
// 주기를 안 기다린다.
func TestUploader_PushesTheBytesWithTheRightQuery(t *testing.T) {
	f, cl := newFakeMediator(t)
	u := newUploader(cl, testStep(), quietLog())

	if _, err := u.Write([]byte("hello\n")); err != nil {
		t.Fatal(err)
	}
	u.Close()

	if got := f.bytesGot(); got != "hello\n" {
		t.Fatalf("the mediator should hold what the harness said, got %q", got)
	}
	// attempt=0 을 그대로 보낸다. +1 을 하면 진행 파일의 숫자가 logs/ 와
	// 원장의 시도 번호에서 한 칸 어긋난다.
	want := "name=build&progress=1&attempt=0"
	if q := f.lastQuery(); q != want {
		t.Errorf("query = %q, want %q", q, want)
	}
}

// 두 번 밀면 서버의 총 길이가 늘어 있다 - 장면 게이트의 첫 줄이다.
func TestUploader_TheTotalGrowsAcrossChunks(t *testing.T) {
	f, cl := newFakeMediator(t)
	u := newUploader(cl, testStep(), quietLog())

	_, _ = u.Write(bytes.Repeat([]byte("a"), uploadNow+1)) // 즉시 문턱을 넘겨 깨운다
	waitFor(t, func() bool { return f.callCount() >= 1 }, "the first chunk never went")
	first := len(f.bytesGot())

	_, _ = u.Write(bytes.Repeat([]byte("b"), uploadNow+1))
	waitFor(t, func() bool { return len(f.bytesGot()) > first }, "the total never grew")
	u.Close()

	if got := len(f.bytesGot()); got <= first {
		t.Fatalf("the second chunk did not land: %d then %d", first, got)
	}
}

// Write 는 네트워크를 0 번 기다린다.
//
// 이 시험에 시간 상한이 있는 것이 값이다 - 동기로 보내는 변이를 걸면 실패
// 메시지가 아니라 "영영 안 끝남" 으로 빨개지고, 그 둘은 읽는 사람에게 다르다.
func TestUploader_WriteNeverWaitsForTheNetwork(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block // 서버가 영영 안 답한다
	}))
	defer srv.Close()
	defer close(block)

	u := newUploader(&Client{Base: srv.URL, HTTP: srv.Client()}, testStep(), quietLog())

	done := make(chan time.Duration, 1)
	go func() {
		start := time.Now()
		for i := 0; i < 50; i++ {
			_, _ = u.Write(bytes.Repeat([]byte("x"), uploadNow+1))
		}
		done <- time.Since(start)
	}()

	select {
	case d := <-done:
		if d > 2*time.Second {
			t.Fatalf("Write waited %v; the harness stdout stalls for that long", d)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Write blocked on the network; the harness would stall with it")
	}
}

// 서버가 잠깐 죽었다 살아나면 다음 청크가 이어 붙고 두 벌이 안 생긴다.
func TestUploader_AFailedChunkIsResentExactlyOnce(t *testing.T) {
	f, cl := newFakeMediator(t)
	u := newUploader(cl, testStep(), quietLog())

	f.failNext.Store(3) // 다음 셋은 떨어진다
	_, _ = u.Write([]byte("first\n"))
	_, _ = u.Write(bytes.Repeat([]byte("x"), uploadNow)) // 깨운다
	waitFor(t, func() bool { return f.callCount() >= 1 }, "no attempt was made")

	// 서버가 살아난다. 남은 것이 다음 주기에 한 번에 간다.
	waitFor(t, func() bool { return f.failNext.Load() == 0 }, "the failures never drained")
	_, _ = u.Write([]byte("last\n"))
	u.Close()

	got := f.bytesGot()
	if n := bytes.Count([]byte(got), []byte("first\n")); n != 1 {
		t.Fatalf("the first chunk landed %d times, want exactly 1:\n%q", n, got)
	}
	if !bytes.HasSuffix([]byte(got), []byte("last\n")) {
		t.Errorf("the tail did not land in order: %q", got)
	}
}

// 버퍼가 넘치면 멈춘다. Write 는 그래도 바로 돌아오고, 연결이 돌아와도
// 재개하지 않는다 - 안 받은 바이트가 이미 없으므로 재개하면 표시 없는
// 구멍이 생긴다.
func TestUploader_AFullBufferStopsAndDoesNotResume(t *testing.T) {
	f, cl := newFakeMediator(t)
	u := newUploader(cl, testStep(), quietLog())
	f.failNext.Store(1000) // 아무것도 안 나간다

	chunk := bytes.Repeat([]byte("y"), 64<<10)
	for i := 0; i < (uploadBufferMax/len(chunk))+4; i++ {
		start := time.Now()
		if _, err := u.Write(chunk); err != nil {
			t.Fatalf("Write must never fail: %v", err)
		}
		if d := time.Since(start); d > time.Second {
			t.Fatalf("Write took %v after the buffer filled", d)
		}
	}
	u.mu.Lock()
	over, done := u.overflowed, u.done
	u.mu.Unlock()
	if !over || !done {
		t.Fatalf("the buffer should have overflowed and stopped: overflowed=%v done=%v", over, done)
	}

	// 서버가 살아나도 재개하지 않는다.
	f.failNext.Store(0)
	before := f.callCount()
	_, _ = u.Write([]byte("after the overflow\n"))
	time.Sleep(50 * time.Millisecond)
	u.Close()
	if f.callCount() > before+1 {
		t.Errorf("a stopped uploader resumed: %d calls then %d", before, f.callCount())
	}
	if bytes.Contains([]byte(f.bytesGot()), []byte("after the overflow")) {
		t.Error("resuming after an overflow would leave an unmarked hole in the middle")
	}
}

// 서버가 상한에 닿았다고 하면 멈춘다. 실행은 계속되고 왕복이 0 이 된다.
func TestUploader_StopsWhenTheServerSaysCapped(t *testing.T) {
	f, cl := newFakeMediator(t)
	f.capped.Store(true)
	u := newUploader(cl, testStep(), quietLog())

	_, _ = u.Write(bytes.Repeat([]byte("z"), uploadNow+1))
	waitFor(t, func() bool { return f.callCount() >= 1 }, "the first chunk never went")
	after := f.callCount()

	for i := 0; i < 5; i++ {
		if _, err := u.Write(bytes.Repeat([]byte("z"), uploadNow+1)); err != nil {
			t.Fatalf("the step keeps running after the cap: %v", err)
		}
	}
	time.Sleep(50 * time.Millisecond)
	u.Close()
	if f.callCount() != after {
		t.Errorf("a capped uploader kept pushing: %d then %d round trips", after, f.callCount())
	}
}

// 서버가 나보다 큰 시도를 말하면 내 청크가 버려진 것이다 - 멈춘다.
// 새 시도는 새 업로더가 이미 돌고 있으므로, 죽은 쪽이 계속 밀면 서버가
// 계속 버리고 전역 한도만 갉는다.
func TestUploader_StopsWhenANewerAttemptIsRunning(t *testing.T) {
	f, cl := newFakeMediator(t)
	f.sayAttemt.Store(2) // 서버는 시도 2 를 들고 있다
	u := newUploader(cl, testStep(), quietLog())

	_, _ = u.Write(bytes.Repeat([]byte("q"), uploadNow+1))
	waitFor(t, func() bool { return f.callCount() >= 1 }, "the first chunk never went")
	after := f.callCount()

	_, _ = u.Write(bytes.Repeat([]byte("q"), uploadNow+1))
	time.Sleep(50 * time.Millisecond)
	u.Close()
	if f.callCount() != after {
		t.Errorf("a stale uploader kept pushing: %d then %d", after, f.callCount())
	}
}

// 링과 업로더가 서로를 안 막는다. 한쪽이 없어도 다른 쪽이 돈다.
func TestTranscript_TeesToWhicheverExists(t *testing.T) {
	_, cl := newFakeMediator(t)
	step := testStep()

	// 링만
	w := &Worker{Log: quietLog()}
	r, p := openRing(t, 4096)
	w.ring = r
	sink, stop := w.transcript(nil) // Client 가 nil 이라 업로더가 안 붙는다
	if sink == nil {
		t.Fatal("a ring alone should still be teed to")
	}
	_, _ = sink.Write([]byte("ring only\n"))
	stop()
	snap, err := ReadRing(p)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(snap.Data, []byte("ring only")) {
		t.Errorf("the ring did not get the bytes: %q", snap.Data)
	}

	// 업로더만
	w2 := &Worker{Client: cl, Log: quietLog()}
	sink2, stop2 := w2.transcript(step)
	if sink2 == nil {
		t.Fatal("an uploader alone should still be teed to")
	}
	_, _ = sink2.Write([]byte("uploader only\n"))
	stop2()

	// 둘 다 없으면 nil 과 no-op 이라 부르는 쪽에 갈래가 안 생긴다
	w3 := &Worker{Log: quietLog()}
	sink3, stop3 := w3.transcript(nil)
	if sink3 != nil {
		t.Error("with neither a ring nor an uploader the sink should be nil")
	}
	stop3()
}

func waitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal(msg)
}
