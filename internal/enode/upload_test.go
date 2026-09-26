package enode

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/taeels/enode/internal/contract"
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

// ── 단계 끝의 업로드 (FR-3 · N3) ──────────────────────────────────

// blobServer 는 단계 끝의 업로드를 받는 시험 서버다. 이름마다 답을 고른다.
type blobServer struct {
	mu      sync.Mutex
	got     map[string]int64 // 이름 -> 받은 바이트
	lengths map[string]int64 // 이름 -> 요청의 Content-Length
	order   []string
	answer  func(name string, w http.ResponseWriter, r *http.Request) bool // 참이면 답을 끝냈다
}

func newBlobServer(t *testing.T) (*blobServer, *Worker) {
	t.Helper()
	b := &blobServer{got: map[string]int64{}, lengths: map[string]int64{}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if i := strings.Index(r.URL.Path, "/blob/"); i >= 0 {
			name = r.URL.Path[i+len("/blob/"):]
		}
		if b.answer != nil && b.answer(name, w, r) {
			return
		}
		n, _ := io.Copy(io.Discard, r.Body)
		b.mu.Lock()
		b.got[name], b.lengths[name] = n, r.ContentLength
		b.order = append(b.order, name)
		b.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	w := &Worker{
		// HTTP 는 쓰면 안 되는 client 다 — 업로드는 Upload 로만 나가야 한다
		Client: &Client{Base: srv.URL, HTTP: &http.Client{Transport: failingTransport{}}, Upload: srv.Client()},
		Log:    discardLog(),
	}
	return b, w
}

// hold 는 답하지 않고 붙잡는다. 본문을 먼저 다 읽는다 — 서버는 본문을 다 읽은 뒤에야
// 끊긴 연결을 알아채고 r.Context() 를 끝낸다.
func hold(r *http.Request) {
	_, _ = io.Copy(io.Discard, r.Body)
	select {
	case <-r.Context().Done():
	case <-time.After(10 * time.Second):
	}
}

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, io.ErrUnexpectedEOF
}

func outWith(t *testing.T, files map[string]int) string {
	t.Helper()
	out := t.TempDir()
	for name, size := range files {
		if err := os.WriteFile(filepath.Join(out, name), bytes.Repeat([]byte("x"), size), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return out
}

// 흘려 보낸다 — Content-Length 가 파일 크기이고 본문이 그대로 닿는다. 단계 로그가 먼저다.
// 어댑터가 남긴 것(.enode-)과 마감에 멈춘 복사의 반쪽(.part)은 산출물이 아니다.
func TestUpload_StreamsWithTheFileSizeAndSkipsLeavings(t *testing.T) {
	b, w := newBlobServer(t)
	out := outWith(t, map[string]int{"big": 3 << 20, "empty": 0, ".enode-prompt.md": 5, "half.part": 7})
	produced, stage := w.upload(context.Background(), runStep(), out, []byte("log"), true, discardLog())
	if stage != contract.StageOK || strings.Join(produced, ",") != "big,empty" {
		t.Fatalf("produced %v stage %q", produced, stage)
	}
	if b.got["big"] != 3<<20 || b.lengths["big"] != 3<<20 {
		t.Errorf("big: got %d bytes with Content-Length %d", b.got["big"], b.lengths["big"])
	}
	if b.order[0] != "build" {
		t.Errorf("the step log must go first: %v", b.order)
	}
}

// Mediator 가 받고서 거절한 것(413 · 422)은 produced 에 없고 upload 칸을 바꾸지 않는다.
func TestUpload_ARejectionIsTheMediatorsJudgment(t *testing.T) {
	b, w := newBlobServer(t)
	b.answer = func(name string, w http.ResponseWriter, r *http.Request) bool {
		if name != "huge" {
			return false
		}
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		_, _ = io.WriteString(w, "blob exceeds the limit")
		return true
	}
	out := outWith(t, map[string]int{"huge": 10, "small": 1})
	produced, stage := w.upload(context.Background(), runStep(), out, nil, true, discardLog())
	if stage != contract.StageOK || strings.Join(produced, ",") != "small" {
		t.Fatalf("produced %v stage %q", produced, stage)
	}
}

// 업로드 예산의 마감 — 진행 중 요청을 끊고 남은 이름을 안 올린다. upload 는 timeout.
func TestUpload_TheBudgetStopsTheRest(t *testing.T) {
	b, w := newBlobServer(t)
	b.answer = func(name string, w http.ResponseWriter, r *http.Request) bool {
		if name != "b" {
			return false
		}
		hold(r) // 답하지 않는다
		return true
	}
	out := outWith(t, map[string]int{"a": 1, "b": 1, "c": 1})
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	produced, stage := w.upload(ctx, runStep(), out, nil, true, discardLog())
	if stage != contract.StageTimeout || strings.Join(produced, ",") != "a" {
		t.Fatalf("produced %v stage %q", produced, stage)
	}
	if _, ok := b.got["c"]; ok {
		t.Fatal("a name after the deadline was still uploaded")
	}
}

// 예산 안의 전송 실패는 그 이름만 빠지고 다음 이름으로 간다. upload 는 error.
func TestUpload_ATransferFailureMovesOn(t *testing.T) {
	b, w := newBlobServer(t)
	b.answer = func(name string, w http.ResponseWriter, r *http.Request) bool {
		if name != "a" {
			return false
		}
		hj, _ := w.(http.Hijacker)
		conn, _, _ := hj.Hijack()
		_ = conn.Close() // 답 없이 끊는다
		return true
	}
	out := outWith(t, map[string]int{"a": 1, "b": 1})
	produced, stage := w.upload(context.Background(), runStep(), out, nil, true, discardLog())
	if stage != contract.StageError || strings.Join(produced, ",") != "b" {
		t.Fatalf("produced %v stage %q", produced, stage)
	}
}

// 5xx 도 전송 실패다 — Mediator 가 받지 못했다.
func TestUpload_AServerErrorIsAFailure(t *testing.T) {
	b, w := newBlobServer(t)
	b.answer = func(name string, w http.ResponseWriter, r *http.Request) bool {
		if name != "build" {
			return false
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		return true
	}
	produced, stage := w.upload(context.Background(), runStep(), outWith(t, map[string]int{"a": 1}), []byte("log"), true, discardLog())
	if stage != contract.StageError || strings.Join(produced, ",") != "a" {
		t.Fatalf("produced %v stage %q", produced, stage)
	}
}

// 단계 로그에서 마감이 오면 산출물을 하나도 안 올린다. 완주 못 한 agent 단계는 로그만.
func TestUpload_LogOnlyAndADeadlineOnTheLog(t *testing.T) {
	b, w := newBlobServer(t)
	produced, stage := w.upload(context.Background(), runStep(), outWith(t, map[string]int{"a": 1}), []byte("log"), false, discardLog())
	if stage != contract.StageOK || produced != nil || len(b.order) != 1 {
		t.Fatalf("log only: produced %v stage %q order %v", produced, stage, b.order)
	}
	b.answer = func(name string, w http.ResponseWriter, r *http.Request) bool {
		hold(r)
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	produced, stage = w.upload(ctx, runStep(), outWith(t, map[string]int{"a": 1}), []byte("log"), true, discardLog())
	if stage != contract.StageTimeout || produced != nil {
		t.Fatalf("deadline on the log: produced %v stage %q", produced, stage)
	}
}

// 임대가 끝나 멈춘 업로드는 마감이 아니다 — error 로 적는다.
func TestUpload_ALeaseEndIsNotTheBudget(t *testing.T) {
	_, w := newBlobServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, stage := w.upload(ctx, runStep(), outWith(t, map[string]int{"a": 1}), []byte("log"), true, discardLog())
	if stage != contract.StageError {
		t.Fatalf("stage %q, want error", stage)
	}
}
