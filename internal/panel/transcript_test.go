package panel

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/taeels/enode/internal/enode"
	"github.com/taeels/enode/internal/transcript"
)

func TestHandleTranscript(t *testing.T) {
	s := testServer(t, nil, "n")
	// no ring yet -> available:false
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()
	resp, _ := http.Get(srv.URL + "/api/transcript")
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	resp.Body.Close()
	if out["available"] != false {
		t.Fatalf("no ring should give available=false: %+v", out)
	}

	// write a ring beside the config, then it should read
	r, err := enode.OpenRing(enode.TranscriptPath(s.cfg.ConfigPath), 4096)
	if err != nil {
		t.Fatal(err)
	}
	r.Write([]byte("agent is thinking..."))
	r.Close() //nolint:errcheck

	resp2, _ := http.Get(srv.URL + "/api/transcript")
	var out2 map[string]any
	_ = json.NewDecoder(resp2.Body).Decode(&out2)
	resp2.Body.Close()
	if out2["available"] != true || out2["data"] != "agent is thinking..." {
		t.Fatalf("ring content not served: %+v", out2)
	}
}

func TestHandleRunsFiltersByNode(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/runs", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"runs": []any{
			map[string]any{"run_id": "mine", "state": "SUCCEEDED",
				"assigned": []any{map[string]any{"nodes": []any{map[string]any{"node": "node-xyz"}}}}},
			map[string]any{"run_id": "theirs", "state": "FAILED",
				"assigned": []any{map[string]any{"nodes": []any{map[string]any{"node": "other"}}}}},
		}})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	s := testServer(t, nil, "node-xyz")
	s.client.Base = srv.URL

	psrv := httptest.NewServer(s.Handler())
	defer psrv.Close()
	resp, err := http.Get(psrv.URL + "/api/runs")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out struct {
		Runs []pastRun `json:"runs"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if len(out.Runs) != 1 || out.Runs[0].RunID != "mine" {
		t.Fatalf("expected only this node's run: %+v", out.Runs)
	}
}

func TestHandleRecordNeedsRun(t *testing.T) {
	s := testServer(t, nil, "n")
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()
	resp, _ := http.Get(srv.URL + "/api/record")
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing run should be 400, got %d", resp.StatusCode)
	}
}

// 아래는 U5 panel-live 가 더한 것이다 — 봉투 · 캐시 · 잘림 · 경과.
//
// 픽스처에 개행을 붙이는 것이 규율이다. transcript.Parse 는 개행 없이 끝나는
// 꼬리를 안 읽고 Partial 에 센다 (쓰는 중에 읽히는 링을 위한 규율이다).
// 안 붙이면 사건이 0 개이고, 그 0 은 "파서가 안 불렸다" 와 구별이 안 된다.

const txLine = `{"type":"assistant","message":{"content":[{"type":"text","text":"hello"}]}}` + "\n"

func liveGet(t *testing.T, url string) liveTranscript {
	t.Helper()
	resp, err := http.Get(url + "/api/transcript")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck
	var out liveTranscript
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestLiveTranscriptCarriesEventsAndTheSameBytes(t *testing.T) {
	s := testServer(t, nil, "n")
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	r, err := enode.OpenRing(enode.TranscriptPath(s.cfg.ConfigPath), 4096)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Write([]byte(txLine)); err != nil {
		t.Fatal(err)
	}
	r.Close() //nolint:errcheck

	out := liveGet(t, srv.URL)
	if !out.Available || out.Transcript == nil {
		t.Fatalf("ring should be readable and parsed: %+v", out)
	}
	if len(out.Transcript.Events) != 1 || out.Transcript.Events[0].Text != "hello" {
		t.Fatalf("want one text event carrying the body, got %+v", out.Transcript.Events)
	}
	// 원문 토글이 그리는 바이트가 사건을 만든 것과 같은 읽기여야 한다. 따로
	// 읽으면 링이 그사이 감겨 다른 창을 보이고, 읽는 사람은 그것을 파서의
	// 버그로 읽는다.
	if out.Data != txLine {
		t.Errorf("data must be the same bytes the events came from, got %q", out.Data)
	}
	if out.RingPath != enode.TranscriptPath(s.cfg.ConfigPath) {
		t.Errorf("ring path should be this node's own: %q", out.RingPath)
	}
}

// 링이 없으면 나머지 키가 통째로 빠진다. 있는데 비어 있는 것과 없는 것이
// 화면에서 갈려야 하므로 omitempty 가 실제로 듣는지를 선 위에서 잰다.
func TestLiveTranscriptWithoutARingCarriesNothingElse(t *testing.T) {
	s := testServer(t, nil, "n")
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/transcript")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck
	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		t.Fatal(err)
	}
	if raw["available"] != false {
		t.Fatalf("no ring should give available=false: %+v", raw)
	}
	if len(raw) != 1 {
		t.Errorf("available:false should carry nothing else, got %+v", raw)
	}
}

// R5 를 재는 줄이다 — 응답에 기간(초)이 0 개다.
//
// 왜 필드가 없는 것을 재는가: 캐시가 (generation, total, mtime) 으로 걸리므로
// 침묵이 길어지면 같은 몸통이 계속 나간다. 기간을 서버가 계산해 실으면 그 수가
// 캐시에 얼어붙어 "3초 전" 이 영영 3초 전이 된다. 침묵이 바로 이 표시가 있는
// 이유인 구간이라 거기서만 얼면 표시가 없는 것보다 나쁘다.
func TestLiveTranscriptCarriesAnInstantAndNeverADuration(t *testing.T) {
	s := testServer(t, nil, "n")
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	path := enode.TranscriptPath(s.cfg.ConfigPath)
	r, err := enode.OpenRing(path, 4096)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Write([]byte(txLine)); err != nil {
		t.Fatal(err)
	}
	r.Close() //nolint:errcheck

	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	out := liveGet(t, srv.URL)
	if out.LastWrite != st.ModTime().UTC().Format(time.RFC3339) {
		t.Errorf("last_write should be the ring file's mtime, got %q", out.LastWrite)
	}

	// 링을 과거로 늙힌다. 이 줄이 없으면 이 시험이 아무것도 안 잰다 - 방금
	// 쓴 링은 경과가 0 이라, 누가 기간 필드를 더해도 omitempty 가 그 키를
	// 지워 아래 검사를 그냥 통과한다. 변이로 실제로 밟아 확인했다.
	old := time.Now().Add(-90 * time.Second)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}

	resp, err := http.Get(srv.URL + "/api/transcript")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck
	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		t.Fatal(err)
	}
	if raw["last_write"] != old.UTC().Format(time.RFC3339) {
		t.Errorf("last_write should follow the ring's mtime, got %v", raw["last_write"])
	}
	// 봉투가 이름으로 인정한 키 밖에 수가 있으면 안 된다. 이름으로 막는 것보다
	// 이쪽이 센 이유 - 새로 더해지는 기간 필드는 이름을 뭐라 짓든 여기 걸린다.
	// total 과 capacity 도 수라서 범위로는 못 가른다 (바이트 수가 초와 같은
	// 자리에 온다). 그래서 아는 키를 빼고 나머지를 전부 본다.
	known := map[string]bool{
		"available": true, "generation": true, "total": true, "capacity": true,
		"truncated": true, "last_write": true, "ring_path": true,
		"transcript": true, "data": true,
	}
	for k, v := range raw {
		if known[k] {
			continue
		}
		if _, isNum := v.(float64); isNum {
			t.Errorf("unexpected numeric key %q = %v; the response must carry an instant, not a duration", k, v)
		}
	}
}

// 캐시가 듣는지를 파싱 결과의 동일성이 아니라 포인터로 잰다 — 같은 몸통이
// 그대로 다시 나가는 것이 이 캐시의 약속이다.
func TestLiveTranscriptReusesTheBodyUntilTheRingMoves(t *testing.T) {
	s := testServer(t, nil, "n")
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	path := enode.TranscriptPath(s.cfg.ConfigPath)
	r, err := enode.OpenRing(path, 4096)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Write([]byte(txLine)); err != nil {
		t.Fatal(err)
	}

	first := liveGet(t, srv.URL)
	s.live.mu.Lock()
	cached := s.live.body.Transcript
	s.live.mu.Unlock()
	if cached == nil {
		t.Fatal("the first read should have filled the cache")
	}

	second := liveGet(t, srv.URL)
	if second.Total != first.Total || len(second.Transcript.Events) != len(first.Transcript.Events) {
		t.Fatalf("an unmoved ring should give the same body: %+v vs %+v", first, second)
	}
	s.live.mu.Lock()
	still := s.live.body.Transcript == cached
	s.live.mu.Unlock()
	if !still {
		t.Error("an unmoved ring must not be parsed again")
	}

	// 링이 움직이면 캐시가 안 듣는다.
	if _, err := r.Write([]byte(txLine)); err != nil {
		t.Fatal(err)
	}
	r.Close() //nolint:errcheck
	third := liveGet(t, srv.URL)
	if len(third.Transcript.Events) != 2 {
		t.Fatalf("a moved ring must be parsed again, got %d events", len(third.Transcript.Events))
	}
}

// 열쇠에서 mtime 을 빼면 여기가 빨개진다. 링을 통째로 다시 만들면 (0, 0) 으로
// 돌아오므로 (generation, total) 둘로는 옛 항목에 맞는다.
func TestLiveTranscriptDoesNotServeAStaleBodyToARebuiltRing(t *testing.T) {
	s := testServer(t, nil, "n")
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	// 두 줄의 길이가 같아야 이 시험이 무엇을 잰다. 길이가 다르면 total 이
	// 갈려 (generation, total) 둘만으로도 캐시가 안 맞고, 그러면 mtime 이
	// 열쇠에 있든 없든 통과한다 - 변이로 실제로 밟아 확인했다.
	const lineA = `{"type":"assistant","message":{"content":[{"type":"text","text":"hello"}]}}` + "\n"
	const lineB = `{"type":"assistant","message":{"content":[{"type":"text","text":"world"}]}}` + "\n"
	if len(lineA) != len(lineB) {
		t.Fatalf("this test needs two lines of equal length, got %d and %d", len(lineA), len(lineB))
	}

	path := enode.TranscriptPath(s.cfg.ConfigPath)
	r, err := enode.OpenRing(path, 4096)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Write([]byte(lineA)); err != nil {
		t.Fatal(err)
	}
	r.Close() //nolint:errcheck
	first := liveGet(t, srv.URL)
	if first.Transcript == nil || first.Transcript.Events[0].Text != "hello" {
		t.Fatalf("the first read should have cached the first line: %+v", first)
	}

	// 파일을 지우고 같은 자리에 새 링을 만들어 같은 길이의 다른 줄을 쓴다.
	// generation 도 total 도 앞것과 똑같다 - 갈리는 것은 mtime 하나뿐이다.
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond) // mtime 해상도
	r2, err := enode.OpenRing(path, 4096)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r2.Write([]byte(lineB)); err != nil {
		t.Fatal(err)
	}
	r2.Close() //nolint:errcheck

	out := liveGet(t, srv.URL)
	if out.Generation != first.Generation || out.Total != first.Total {
		t.Fatalf("the rebuilt ring must land on the same key for this test to measure anything: %d/%d vs %d/%d",
			out.Generation, out.Total, first.Generation, first.Total)
	}
	if out.Transcript == nil || out.Transcript.Events[0].Text != "world" {
		t.Fatalf("a rebuilt ring must not hit the old cache entry: %+v", out.Transcript)
	}
}

// R9 — 엄격 부등호다. Total == Capacity 는 몸통이 정확히 찬 것이고 머리가
// 안 잘렸다. >= 로 바꾸면 이 시험이 빨개진다.
func TestLiveTranscriptTreatsAnExactlyFullRingAsWhole(t *testing.T) {
	const capacity = 256
	s := testServer(t, nil, "n")
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	r, err := enode.OpenRing(enode.TranscriptPath(s.cfg.ConfigPath), capacity)
	if err != nil {
		t.Fatal(err)
	}
	body := make([]byte, capacity)
	for i := range body {
		body[i] = 'a'
	}
	body[capacity-1] = '\n'
	if _, err := r.Write(body); err != nil {
		t.Fatal(err)
	}
	r.Close() //nolint:errcheck

	out := liveGet(t, srv.URL)
	if out.Total != capacity || out.Capacity != capacity {
		t.Fatalf("want a ring filled to exactly its capacity, got %d/%d", out.Total, out.Capacity)
	}
	if out.Truncated {
		t.Error("a ring filled to exactly its capacity has lost no head")
	}
	if out.Transcript.Head != 0 {
		t.Errorf("the parser should not have dropped a head, got %d", out.Transcript.Head)
	}
}

// 감긴 링은 잘렸다고 말하고, 파서가 반쪽 첫 줄을 버린다.
func TestLiveTranscriptReportsAWrappedRing(t *testing.T) {
	const capacity = 256
	s := testServer(t, nil, "n")
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	r, err := enode.OpenRing(enode.TranscriptPath(s.cfg.ConfigPath), capacity)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		if _, err := r.Write([]byte(txLine)); err != nil {
			t.Fatal(err)
		}
	}
	r.Close() //nolint:errcheck

	out := liveGet(t, srv.URL)
	if out.Total <= uint64(out.Capacity) {
		t.Fatalf("this ring should have wrapped: %d/%d", out.Total, out.Capacity)
	}
	if !out.Truncated {
		t.Error("a wrapped ring must say its head was cut")
	}
	if out.Transcript.Head == 0 {
		t.Error("the parser should have dropped the half line at the head")
	}
}

// 단계가 바뀌면 세대가 오른다 — 화면이 그것으로 펼침 상태를 비운다.
func TestLiveTranscriptGenerationRisesOnReset(t *testing.T) {
	s := testServer(t, nil, "n")
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	r, err := enode.OpenRing(enode.TranscriptPath(s.cfg.ConfigPath), 4096)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Write([]byte(txLine)); err != nil {
		t.Fatal(err)
	}
	before := liveGet(t, srv.URL)

	time.Sleep(10 * time.Millisecond)
	if err := r.Reset(); err != nil {
		t.Fatal(err)
	}
	r.Close() //nolint:errcheck

	after := liveGet(t, srv.URL)
	if after.Generation != before.Generation+1 {
		t.Errorf("reset should raise the generation: %d -> %d", before.Generation, after.Generation)
	}
	if after.Total != 0 {
		t.Errorf("reset should empty the ring, got total %d", after.Total)
	}
	if after.LastWrite == "" {
		t.Error("reset writes the header, so the step start is elapsed zero")
	}
}

// 변이 하나가 시험으로는 안 잡혔다 — 사건을 만든 읽기와 원문 토글이 그리는
// 읽기를 따로 두는 변이다. 조용한 링에서는 두 읽기가 같은 바이트라 아무 단언도
// 안 걸리고, 쓰는 고루틴을 띄워 경합으로 재 봤더니 열 번에 일곱 번만 빨갰다.
// 열에 셋을 놓치는 시험은 게이트가 아니다.
//
// 그래서 시험이 아니라 구조로 닫았다 — liveBody 는 스냅샷 하나만 받는다.
// 나눠 볼 둘이 없으므로 그 갈래가 아예 없다. 아래는 그 함수가 자기가 받은
// 하나에서만 두 자리를 채우는지를 결정적으로 잰다.
func TestLiveBodyFillsBothViewsFromTheOneSnapshotItWasGiven(t *testing.T) {
	raw := []byte(txLine + txLine)
	snap := enode.Snapshot{
		Data:       raw,
		Generation: 3,
		Total:      uint64(len(raw)),
		Capacity:   4096,
	}
	mtime := time.Date(2026, 9, 16, 7, 41, 2, 0, time.UTC)
	body := liveBody(snap, mtime, "/tmp/node.transcript")

	if body.Data != string(snap.Data) {
		t.Errorf("the raw view must be the bytes it was handed, got %q", body.Data)
	}
	if body.Transcript == nil || len(body.Transcript.Events) != 2 {
		t.Fatalf("the event view must come from the same bytes: %+v", body.Transcript)
	}
	// 두 자리가 같은 바이트에서 왔다는 것을 산수로 고정한다. 안 감긴 링에서
	// ReadRing 은 body[:total] 을 그대로 내므로 이 등식이 정확히 성립한다.
	if len(body.Data) != int(body.Total) {
		t.Errorf("total says %d bytes but the raw view carries %d", body.Total, len(body.Data))
	}
	again := transcript.Parse([]byte(body.Data), body.Truncated)
	if len(again.Events) != len(body.Transcript.Events) {
		t.Errorf("re-parsing the raw view gives %d events, the envelope carries %d",
			len(again.Events), len(body.Transcript.Events))
	}
	if body.LastWrite != "2026-09-16T07:41:02Z" {
		t.Errorf("last_write should be the instant it was handed, got %q", body.LastWrite)
	}
}

// liveBody 는 시계를 안 읽는다 — 같은 입력에 언제나 같은 봉투다.
// 그 성질이 부르는 쪽의 캐시가 성립하는 근거이고, 기간을 실으면 깨진다.
func TestLiveBodyIsPure(t *testing.T) {
	snap := enode.Snapshot{Data: []byte(txLine), Total: uint64(len(txLine)), Capacity: 4096}
	mtime := time.Date(2026, 9, 16, 7, 41, 2, 0, time.UTC)

	first, err := json.Marshal(liveBody(snap, mtime, "/tmp/r"))
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	second, err := json.Marshal(liveBody(snap, mtime, "/tmp/r"))
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Errorf("liveBody must not read a clock:\n%s\n%s", first, second)
	}
}

// 아래는 U6 panel-past 가 더한 것이다 — 지난 것의 출처가 tar 에서 GET 으로 바뀌었다.
//
// 앞 판의 시험은 tar 를 지어 먹이고 logs/NN-*.log 만 꺼내는지를 쟀다. 재던
// 것이 없어졌고 재는 것이 바뀐다 — 단계 목록을 받아 단계마다 로그 하나를
// 읽고 봉투 하나로 묶는가다.

// pastFake 는 상세 하나와 단계마다의 로그를 내는 가짜 Mediator 다.
//
// 읽기 횟수를 세는 것이 이 픽스처의 값이다. 단계마다 한 번만 읽는 것이
// 이 유닛의 규율이고 (사건 열과 원문이 같은 읽기에서 나와야 한다) 그것은
// 응답을 보는 것만으로는 안 재진다.
type pastFake struct {
	mu    sync.Mutex
	reads map[int]int // seq -> 그 단계의 로그를 몇 번 읽었나
	steps []pastFakeStep
	state string
}

type pastFakeStep struct {
	seq    int
	id     string
	state  string
	chosen bool
	// body 는 읽을 때마다 낸다. n 은 그 단계를 몇 번째로 읽는가다 (1 부터).
	body   func(n int) string
	source string
	// total 이 0 이면 본문 길이를 쓴다. 다르게 주면 잘린 응답이다.
	total  int64
	capped bool
	// code 가 0 이 아니면 그 상태로 답한다 (단계 하나의 실패).
	code int
}

func newPastFake(state string, steps ...pastFakeStep) *pastFake {
	return &pastFake{reads: map[int]int{}, steps: steps, state: state}
}

func (f *pastFake) handler(runID string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/runs/"+runID, func(w http.ResponseWriter, r *http.Request) {
		out := []map[string]any{}
		for _, st := range f.steps {
			out = append(out, map[string]any{"seq": st.seq, "id": st.id, "state": st.state, "chosen": st.chosen})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"run_id": runID, "state": f.state, "steps": out})
	})
	mux.HandleFunc("GET /v1/runs/"+runID+"/steps/{seq}/log", func(w http.ResponseWriter, r *http.Request) {
		seq, _ := strconv.Atoi(r.PathValue("seq"))
		f.mu.Lock()
		f.reads[seq]++
		n := f.reads[seq]
		f.mu.Unlock()
		for _, st := range f.steps {
			if st.seq != seq {
				continue
			}
			// 화면이 단계 이름으로 Step.ID 를 넘기는지를 여기서 잰다. 상세의
			// 단계 객체에 이름 필드가 없고 ID 가 곧 로그의 이름이다.
			if got := r.URL.Query().Get("name"); got != st.id {
				http.Error(w, fmt.Sprintf("name = %q, want the step id %q", got, st.id), http.StatusBadRequest)
				return
			}
			if st.code != 0 {
				http.Error(w, "read failed", st.code)
				return
			}
			body := ""
			if st.body != nil {
				body = st.body(n)
			}
			total := st.total
			if total == 0 {
				total = int64(len(body))
			}
			w.Header().Set("X-Enode-Log-Source", st.source)
			w.Header().Set("X-Enode-Log-Bytes", strconv.FormatInt(total, 10))
			if st.capped {
				w.Header().Set("X-Enode-Log-Capped", "1")
			}
			_, _ = w.Write([]byte(body))
			return
		}
		http.Error(w, "no such step", http.StatusNotFound)
	})
	return mux
}

// pastGet 은 제어판의 지난 것 봉투를 받는다.
func pastGet(t *testing.T, f *pastFake, runID string) (int, pastRecord, *Server) {
	t.Helper()
	med := httptest.NewServer(f.handler(runID))
	t.Cleanup(med.Close)
	s := testServer(t, med, "n")
	s.client.Base = med.URL
	psrv := httptest.NewServer(s.Handler())
	t.Cleanup(psrv.Close)

	resp, err := http.Get(psrv.URL + "/api/record?run=" + runID)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck
	var out pastRecord
	if resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatal(err)
		}
	}
	return resp.StatusCode, out, s
}

const elidedLine = `{"type":"enode.elided","events":98,"bytes":408989}` + "\n"

func TestPastRecordIsOneEnvelopePerStep(t *testing.T) {
	f := newPastFake("SUCCEEDED",
		// 봉인된 단계. 걷힌 줄이 있다 — 걷는 일은 봉인 때 일어난다.
		pastFakeStep{seq: 1, id: "survey", state: "DONE", chosen: true, source: "sealed",
			body: func(int) string { return elidedLine + txLine }},
		// 봉인 전 단계. 같은 봉투 안에서 출처가 갈린다 — Run 한 줄로 접으면
		// 이 단계가 거짓이 된다.
		pastFakeStep{seq: 2, id: "build", state: "CLAIMED", chosen: true, source: "progress",
			body: func(int) string { return txLine }},
		// 로그가 없는 단계 둘. 목록에서 안 뺀다 — 빼면 화면의 단계 수가
		// 계약과 안 맞고 왜 없는지를 말할 자리가 사라진다.
		//
		// 둘의 차이가 Chosen 하나다. 총 길이 0 이 "경로가 갈려 안 갔다" 와
		// "골랐는데 못 닿았다" 를 다시 뜻하게 되는 자리이고, Step.Chosen 이
		// 있는 이유가 정확히 그 구별이다.
		pastFakeStep{seq: 3, id: "deploy", state: "SKIPPED", chosen: false, source: "sealed"},
		pastFakeStep{seq: 4, id: "rollback", state: "SKIPPED", chosen: true, source: "sealed"},
	)
	code, out, _ := pastGet(t, f, "r1")
	if code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	if out.RunID != "r1" || out.State != "SUCCEEDED" {
		t.Errorf("run = %q state = %q", out.RunID, out.State)
	}
	if len(out.Steps) != 4 {
		t.Fatalf("steps = %d, want the detail's four (a step with no log stays in the list)", len(out.Steps))
	}

	one := out.Steps[0]
	if one.Source != "sealed" || one.Total != int64(len(elidedLine+txLine)) {
		t.Errorf("step 1 carried source=%q total=%d; the headers must be copied as they came", one.Source, one.Total)
	}
	if one.Transcript == nil || len(one.Transcript.Events) != 1 || one.Transcript.Events[0].Text != "hello" {
		t.Fatalf("step 1 events = %+v", one.Transcript)
	}
	// 걷힌 줄이 봉인된 것에만 있다. 출처 헤더가 첫째 신호이고 이것이 둘째다.
	if one.Transcript.Elided == nil || one.Transcript.Elided.Events != 98 || one.Transcript.Elided.Bytes != 408989 {
		t.Errorf("step 1 elided = %+v, want the sealed marker", one.Transcript.Elided)
	}
	if one.Data != elidedLine+txLine {
		t.Errorf("step 1 data = %q, want the same bytes the events came from", one.Data)
	}

	two := out.Steps[1]
	if two.Source != "progress" {
		t.Errorf("step 2 source = %q, want progress in the same envelope", two.Source)
	}
	// progress 면 elided 가 없다. 진행 파일에 봉인 표시가 섞이면 하나가 틀린 것이다.
	if two.Transcript == nil || two.Transcript.Elided != nil {
		t.Errorf("step 2 elided = %+v, want none before the seal", two.Transcript)
	}

	three, four := out.Steps[2], out.Steps[3]
	if three.Total != 0 || three.State != "SKIPPED" || four.Total != 0 || four.State != "SKIPPED" {
		t.Fatalf("steps 3 and 4 = %+v %+v, want kept steps with nothing in them", three, four)
	}
	// 두 단계의 차이는 Chosen 하나다. 접으면 SKIPPED 가 다시 두 가지를 뜻하고
	// 화면이 "안 갔다" 와 "갔는데 못 닿았다" 를 합친다.
	if three.Chosen || !four.Chosen {
		t.Errorf("chosen = %v %v, want false then true", three.Chosen, four.Chosen)
	}
	if three.Transcript == nil {
		t.Error("step 3 lost its transcript; an empty log is not a failure")
	}
}

// 사건 열과 원문이 같은 읽기에서 나온다.
//
// 이 시험이 이 유닛의 값을 지킨다. 설계는 단계마다 as=events 와 as=raw 를 따로
// 부르기로 적었고, 그러면 봉인 전 진행 파일이 두 읽기 사이에 자라 사건 열과
// 원문이 다른 창을 보인다. 가짜가 읽을 때마다 다른 것을 내므로 두 번 읽으면
// 이 단언이 죽는다.
func TestPastRecordReadsEachStepOnce(t *testing.T) {
	f := newPastFake("RUNNING",
		pastFakeStep{seq: 1, id: "survey", state: "CLAIMED", chosen: true, source: "progress",
			body: func(n int) string { return strings.Repeat(txLine, n) }},
	)
	_, out, _ := pastGet(t, f, "r1")
	if len(out.Steps) != 1 {
		t.Fatalf("steps = %d", len(out.Steps))
	}
	if f.reads[1] != 1 {
		t.Fatalf("the step's log was read %d times; the events and the raw bytes must come from one read", f.reads[1])
	}
	got := out.Steps[0]
	if got.Data != txLine || got.Transcript == nil || len(got.Transcript.Events) != 1 {
		t.Errorf("data = %q with %d events; they must agree", got.Data, len(got.Transcript.Events))
	}
	// 받은 길이를 브라우저가 문자열 길이로 대신 셀 수 없으므로 서버가 센다.
	if got.Received != len(txLine) {
		t.Errorf("received = %d, want %d", got.Received, len(txLine))
	}
}

// 한 응답의 상한에서 잘린 단계. 이 화면은 폴링이 없어 나머지가 영영 안 온다.
func TestPastRecordSaysWhenOnlyAPrefixCame(t *testing.T) {
	f := newPastFake("SUCCEEDED",
		pastFakeStep{seq: 1, id: "survey", state: "DONE", chosen: true, source: "sealed",
			body: func(int) string { return txLine }, total: 999999},
	)
	_, out, _ := pastGet(t, f, "r1")
	got := out.Steps[0]
	if got.Total != 999999 || got.Received != len(txLine) {
		t.Fatalf("total = %d received = %d, want the file's length and this read's length", got.Total, got.Received)
	}
}

// 단계 하나가 실패해도 나머지는 그린다.
//
// tar 는 통째로 오거나 안 왔으므로 이 구별이 없었다. 1+N 이 되면서 생긴
// 자리이고, 한 단계의 실패가 화면 전체를 비우면 읽는 사람이 그것을
// "기록이 없다" 로 읽는다.
func TestPastRecordKeepsTheRestWhenOneStepFails(t *testing.T) {
	f := newPastFake("SUCCEEDED",
		pastFakeStep{seq: 1, id: "survey", state: "DONE", chosen: true, source: "sealed",
			body: func(int) string { return txLine }},
		pastFakeStep{seq: 2, id: "build", state: "DONE", chosen: true, code: http.StatusServiceUnavailable},
		pastFakeStep{seq: 3, id: "deploy", state: "DONE", chosen: true, source: "sealed",
			body: func(int) string { return txLine }},
	)
	code, out, _ := pastGet(t, f, "r1")
	if code != http.StatusOK {
		t.Fatalf("status = %d; one step's failure must not empty the screen", code)
	}
	if len(out.Steps) != 3 {
		t.Fatalf("steps = %d", len(out.Steps))
	}
	if out.Steps[1].Error == "" || out.Steps[1].Transcript != nil {
		t.Errorf("step 2 = %+v, want the failure in its own slot and no transcript", out.Steps[1])
	}
	if out.Steps[0].Transcript == nil || out.Steps[2].Transcript == nil {
		t.Error("the steps around the failure lost their events")
	}
}

// 404 는 "그런 Run 이 없다" 이고 빈 화면과 다르다.
func TestPastRecordCarriesTheMissingRun(t *testing.T) {
	med := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"code":404,"reason":"no such run"}}`, http.StatusNotFound)
	}))
	defer med.Close()
	s := testServer(t, med, "n")
	s.client.Base = med.URL
	psrv := httptest.NewServer(s.Handler())
	defer psrv.Close()

	resp, err := http.Get(psrv.URL + "/api/record?run=gone")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 kept as 404", resp.StatusCode)
	}
}

// 상세가 죽으면 502 다. 그 Run 이 없는 것과 못 본 것이 갈린다.
func TestPastRecordFailingDetailIs502(t *testing.T) {
	med := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"code":503,"reason":"query failed"}}`, http.StatusServiceUnavailable)
	}))
	defer med.Close()
	s := testServer(t, med, "n")
	s.client.Base = med.URL
	psrv := httptest.NewServer(s.Handler())
	defer psrv.Close()

	resp, err := http.Get(psrv.URL + "/api/record?run=r1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", resp.StatusCode)
	}
}
