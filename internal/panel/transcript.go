package panel

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/taeels/enode/internal/enode"
	"github.com/taeels/enode/internal/runctl"
	"github.com/taeels/enode/internal/transcript"
)

// 트랜스크립트 — 지금 도는 것(로컬 링 파일)과 지난 것(Mediator 가 가진 것) 둘이다
// (decisions §6.5). Mediator 를 안 고친다 — 제어판이 읽어 자기 node_id 로 거른다.

// liveTranscript 는 도는 단계의 폴링 응답이다.
//
// 기간(초)을 안 싣는다 - 시각만 싣고 브라우저가 뺀다 (business-rules R5).
// 이 구조체에 초 단위 수 필드가 없는 것이 그 규칙을 재는 법이다.
//
// 왜 기간이면 안 되나 - 아래 캐시가 (generation, total, mtime) 으로 걸린다.
// 침묵이 길어지면 셋이 다 그대로라 캐시가 나가는데, 기간을 서버가 계산해
// 실으면 그 수가 캐시에 얼어붙어 "3초 전" 이 영영 3초 전이 된다. 침묵이
// 바로 이 표시가 있는 이유인 구간이므로, 거기서만 얼면 표시가 없는 것보다
// 나쁘다 - 읽는 사람이 시계를 믿는다.
//
// Transcript 가 포인터인 이유 - Available 이 거짓일 때 키가 통째로 빠져야
// 한다. 값 타입이면 omitempty 가 구조체에 안 듣는다.
type liveTranscript struct {
	Available  bool   `json:"available"`
	Generation uint64 `json:"generation,omitempty"`
	Total      uint64 `json:"total,omitempty"`
	Capacity   int64  `json:"capacity,omitempty"`
	// Truncated 는 링이 감겨 앞이 잘렸다는 뜻이다. 브라우저가 다시 계산하지
	// 않는다 - 규칙이 두 자리에 살면 갈린다.
	Truncated bool `json:"truncated,omitempty"`
	// LastWrite 는 링 파일의 마지막 쓰기 시각이다. RFC 3339 이고 기간이 아니다.
	LastWrite string `json:"last_write,omitempty"`
	// RingPath 는 이 노드의 링 파일 자리다. 노드마다 다르므로 서버가 낸다 -
	// 화면에 고정 문구를 박으면 어떤 기계에서는 틀린 줄이 되고, 틀린 줄은
	// 없는 줄보다 나쁘다 (읽는 사람이 엉뚱한 자리를 지우러 간다).
	RingPath string `json:"ring_path,omitempty"`
	// Transcript 는 파서가 낸 것 통째다. 펼치지 않는다 - 봉인된 것을 내는
	// 쪽(GET log 의 as=events)이 이미 이 타입을 몸통으로 낸다. 같은 키에
	// 같은 타입으로 담아야 화면 함수가 하나로 선다.
	Transcript *transcript.Result `json:"transcript,omitempty"`
	// Data 는 Transcript 를 만든 것과 같은 읽기의 같은 바이트다. 원문 토글이
	// 그린다. 토글을 켤 때 따로 읽으면 링이 그사이 감겨 다른 창을 보이고,
	// 읽는 사람은 그것을 파서의 버그로 읽는다.
	Data string `json:"data,omitempty"`
}

// liveCache 는 한 칸짜리 캐시다. 링이 하나라 한 칸이면 족하다.
//
// 열쇠가 셋인 이유 - (generation, total) 둘로 줄이면 링이 통째로 다시
// 만들어져 (0, 0) 으로 돌아온 자리가 옛 항목에 맞는다 (OpenRing 의 재초기화).
// mtime 이 그 자리를 가른다.
type liveCache struct {
	mu    sync.Mutex
	gen   uint64
	total uint64
	mtime time.Time
	body  liveTranscript
	valid bool
}

// handleTranscript 는 지금 도는 것 — 로컬 링 파일을 읽어 낸다 (1초 폴링 대상).
//
// stat 이 ReadRing 보다 먼저인 것이 값이다 - mtime 을 먼저 얻어야 캐시 열쇠가
// 서고, 몸통 512 KiB 를 읽기 전에 건너뛸 수 있다. 하네스가 수십 초 말이 없으면
// 그 구간의 파싱 비용이 0 이다.
func (s *Server) handleTranscript(w http.ResponseWriter, r *http.Request) {
	path := enode.TranscriptPath(s.cfg.ConfigPath)
	st, err := os.Stat(path)
	if err != nil {
		writeJSON(w, liveTranscript{Available: false})
		return
	}
	mtime := st.ModTime()

	snap, err := enode.ReadRing(path)
	if err != nil {
		writeJSON(w, liveTranscript{Available: false})
		return
	}

	if body, ok := s.live.get(snap.Generation, snap.Total, mtime); ok {
		writeJSON(w, body)
		return
	}

	body := liveBody(snap, mtime, path)
	s.live.put(snap.Generation, snap.Total, mtime, body)
	writeJSON(w, body)
}

// liveBody 는 스냅샷 하나를 봉투 하나로 바꾼다.
//
// 인자가 스냅샷 하나인 것이 이 함수의 값이다. 사건 열과 원문 토글이 같은
// 읽기에서 나와야 하는데, 그것을 시험으로 지키려 하면 지킬 수 없다 - 링이
// 조용하면 두 읽기가 같은 바이트라 어떤 단언도 안 걸리고, 움직이는 링으로
// 재면 경합이라 열 번에 서너 번만 걸린다 (실측했다). 여기서는 나눠 볼
// 스냅샷이 애초에 하나뿐이라 그 갈래가 구조상 없다.
//
// 시계를 안 읽는다 - mtime 을 인자로 받는다. 그래서 같은 입력에 언제나 같은
// 봉투가 나오고, 부르는 쪽의 캐시가 성립한다.
func liveBody(snap enode.Snapshot, mtime time.Time, path string) liveTranscript {
	// 엄격 부등호다. Total == Capacity 면 몸통이 정확히 찬 것이고 머리가
	// 안 잘렸다 - ReadRing 이 total <= capacity 에서 body[:total] 을 그대로 낸다.
	truncated := snap.Total > uint64(snap.Capacity)
	res := transcript.Parse(snap.Data, truncated)
	return liveTranscript{
		Available:  true,
		Generation: snap.Generation,
		Total:      snap.Total,
		Capacity:   snap.Capacity,
		Truncated:  truncated,
		LastWrite:  mtime.UTC().Format(time.RFC3339),
		RingPath:   path,
		Transcript: &res,
		Data:       string(snap.Data),
	}
}

func (c *liveCache) get(gen, total uint64, mtime time.Time) (liveTranscript, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid && c.gen == gen && c.total == total && c.mtime.Equal(mtime) {
		return c.body, true
	}
	return liveTranscript{}, false
}

func (c *liveCache) put(gen, total uint64, mtime time.Time, body liveTranscript) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gen, c.total, c.mtime, c.body, c.valid = gen, total, mtime, body, true
}

type pastRun struct {
	RunID   string          `json:"run_id"`
	State   string          `json:"state"`
	EndedAt *time.Time      `json:"ended_at,omitempty"`
	Verdict json.RawMessage `json:"verdict,omitempty"`
}

// runsResp 는 GET /v1/runs 원문에서 이 유닛이 쓰는 것만 뽑는 최소 파싱이다.
type runsResp struct {
	Runs []struct {
		RunID    string          `json:"run_id"`
		State    string          `json:"state"`
		EndedAt  *time.Time      `json:"ended_at"`
		Verdict  json.RawMessage `json:"verdict"`
		Assigned []struct {
			Nodes []struct {
				Node string `json:"node"`
			} `json:"nodes"`
		} `json:"assigned"`
	} `json:"runs"`
}

// handleRuns 는 지난 것 — GET /v1/runs 를 받아 assigned 에 이 노드가 있는 Run 만
// 낸다. node= 서버 필터를 안 만든다 (decisions §6.5).
func (s *Server) handleRuns(w http.ResponseWriter, r *http.Request) {
	raw, err := s.client.Runs(r.Context(), runctl.RunsQuery{})
	if err != nil {
		s.markMediator(false)
		http.Error(w, "cannot reach the mediator", http.StatusBadGateway)
		return
	}
	s.markMediator(true)
	var resp runsResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		http.Error(w, "cannot parse runs", http.StatusBadGateway)
		return
	}
	out := []pastRun{}
	for _, run := range resp.Runs {
		here := false
		for _, a := range run.Assigned {
			for _, n := range a.Nodes {
				if n.Node == s.ident.NodeID {
					here = true
				}
			}
		}
		if !here {
			continue
		}
		out = append(out, pastRun{RunID: run.RunID, State: run.State, EndedAt: run.EndedAt, Verdict: run.Verdict})
	}
	writeJSON(w, map[string]any{"runs": out})
}

type recordLog struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// handleRecord 는 봉인된 지난 트랜스크립트다 — GET /v1/runs/{id}/record 의 tar 에서
// logs/NN-*.log 만 꺼낸다 (decisions §6.5). archive/tar 는 표준이라 새 의존 0.
func (s *Server) handleRecord(w http.ResponseWriter, r *http.Request) {
	runID := r.URL.Query().Get("run")
	if runID == "" {
		http.Error(w, "run is required", http.StatusBadRequest)
		return
	}
	var buf bytes.Buffer
	if err := s.client.Record(r.Context(), runID, &buf); err != nil {
		s.markMediator(false)
		http.Error(w, "cannot fetch the record", http.StatusBadGateway)
		return
	}
	s.markMediator(true)
	logs := []recordLog{}
	tr := tar.NewReader(&buf)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			http.Error(w, "cannot read the record tar", http.StatusBadGateway)
			return
		}
		name := h.Name
		if !strings.HasPrefix(name, "logs/") || !strings.HasSuffix(name, ".log") {
			continue
		}
		b, err := io.ReadAll(tr)
		if err != nil {
			continue
		}
		logs = append(logs, recordLog{Name: name, Content: string(b)})
	}
	writeJSON(w, map[string]any{"logs": logs})
}
