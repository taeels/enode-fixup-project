package panel

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
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

// 지난 것의 와이어 헤더 셋. 값을 옮겨 싣기만 하고 다시 계산하지 않는다 —
// 규칙이 두 자리에 살면 갈린다.
//
// 이름을 여기 다시 적는 이유는 경계다. 이 상수들의 정본은 internal/api 이고
// panel 은 그것을 임포트하지 않는다 (boundary_test.go 의 금지 표). 두 벌이
// 아니라 와이어 계약을 양쪽이 각자 아는 것이다 — HTTP 헤더 이름이 그렇다.
const (
	headerLogSource = "X-Enode-Log-Source"
	headerLogBytes  = "X-Enode-Log-Bytes"
	headerLogCapped = "X-Enode-Log-Capped"
)

// pastStep 은 봉투의 한 칸이다 — 단계 하나.
//
// 상태와 Chosen 을 함께 싣는 것이 값이다. 총 길이 0 이 세 가지를 뜻하고
// (경로가 갈려 안 갔다 · 골랐는데 못 닿았다 · 돌았고 아무 말도 안 했다) 그
// 셋을 가르는 것이 이 두 필드다. 한 문장으로 접으면 Step.Chosen 이 나른 값을
// 화면에서 버린다.
type pastStep struct {
	Seq    int    `json:"seq"`
	ID     string `json:"id"`
	State  string `json:"state"`
	Chosen bool   `json:"chosen"`

	// Source 는 X-Enode-Log-Source 그대로다 (progress · sealed). 봉인 전과
	// 뒤를 가르는 첫째 신호이고, 갈림을 제어판이 만들지 않는다 — GET log 가
	// 이미 갈라서 답한다.
	Source string `json:"source,omitempty"`
	// Total 은 파일의 총 길이다. 받은 길이가 아니다.
	Total int64 `json:"total"`
	// Received 는 이 응답이 실제로 받은 바이트 수다. Total 보다 작으면
	// 한 응답의 상한에서 잘린 것이고, 이 화면은 폴링이 없으므로 나머지가
	// 영영 안 온다 — 화면이 그 말을 해야 사람이 통째로 본 것으로 안 읽는다.
	//
	// 브라우저가 data 의 길이로 대신 셀 수 없다. JS 문자열의 길이는 UTF-16
	// 단위라 하네스가 한국어를 한 줄만 내도 바이트 수와 갈린다.
	Received int `json:"received"`
	// Capped 는 진행 파일이 상한에 닿았다는 뜻이다 (Received 와 다른 물건이다).
	Capped bool `json:"capped,omitempty"`

	// Transcript 는 파서가 낸 것 통째다. liveTranscript 와 같은 키 · 같은
	// 타입이라 꺼내는 식이 두 화면에서 같고, 그래서 그리는 함수가 하나다.
	// 포인터인 이유도 같다 - 실패한 단계에서 키가 통째로 빠져야 한다.
	Transcript *transcript.Result `json:"transcript,omitempty"`
	// Data 는 Transcript 를 만든 것과 같은 읽기의 같은 바이트다.
	Data string `json:"data,omitempty"`
	// Error 는 이 단계만의 실패다. 단계 하나가 실패해도 나머지는 그린다 -
	// tar 는 통째로 오거나 안 왔으므로 그 구별이 없었고, 1+N 이 되면서
	// 생긴 자리다. 화면 전체를 비우면 읽는 사람이 "기록이 없다" 로 읽는다.
	Error string `json:"error,omitempty"`
}

// pastRecord 는 Run 하나의 지난 트랜스크립트다.
//
// sealed 를 안 싣는다. 제어판에는 그 값을 낼 독립된 출처가 없다 - source 에서
// 유도하면 "둘이 같은 답을 내야 한다" 가 자동으로 참이라 재는 값이 0 이고,
// 종료 상태에서 유도하면 봉인 창에서 거짓말을 한다 (Seal 이 돌기 전에는 Run 이
// 끝났어도 진행 파일이 아직 있다). 봉인 전과 뒤는 단계마다의 Source 가 말한다.
type pastRecord struct {
	RunID string     `json:"run_id"`
	State string     `json:"state"`
	Steps []pastStep `json:"steps"`
}

// handleRecord 는 지난 트랜스크립트다 — 상세로 단계 목록을 받고 단계마다
// GET /v1/runs/{id}/steps/{seq}/log 를 읽어 봉투 하나로 묶는다.
//
// 출처가 바뀌었다. 앞 판은 GET record 의 tar 를 풀어 logs/NN-*.log 를 꺼냈고
// 이 파일에 archive/tar 가 있었다. 무르는 값이 "같은 파서의 같은 모양" 이다 -
// tar 안의 바이트를 그대로 <pre> 에 부으면 도는 것과 지난 것이 다른 화면이 된다.
// runctl 의 Client.Record 는 그대로 산다 (runctl record 가 쓴다).
//
// 모으는 일을 제어판이 한다. 브라우저의 계약이 안 바뀐다 - 오늘도
// GET /api/record 한 번이었다. 브라우저가 1+N 을 지면 실패 갈래가 N+1 개
// 생기고, 단계 셋 중 하나만 실패한 화면을 사람이 읽을 방법이 없다.
//
// 순차로 부른다. 동시성 1 이라 한도에 닿는 봉우리가 안 생기고, 누를 때 한 번이라
// 초당 요청이 0 이다 (폴링이 아니다 - 봉인된 것은 안 자란다).
func (s *Server) handleRecord(w http.ResponseWriter, r *http.Request) {
	runID := r.URL.Query().Get("run")
	if runID == "" {
		http.Error(w, "run is required", http.StatusBadRequest)
		return
	}
	run, err := s.client.Status(r.Context(), runID)
	if err != nil {
		// 404 는 "그런 Run 이 없다" 이고 빈 화면과 다르다. Mediator 는
		// 답했으므로 도달 표시를 안 건드린다.
		var f *runctl.Fail
		if errors.As(err, &f) && f.Code == http.StatusNotFound {
			s.markMediator(true)
			http.Error(w, "no such run", http.StatusNotFound)
			return
		}
		s.markMediator(false)
		http.Error(w, "cannot fetch the record", http.StatusBadGateway)
		return
	}
	s.markMediator(true)
	out := pastRecord{RunID: run.RunID, State: run.State, Steps: []pastStep{}}
	for _, st := range run.Steps {
		out.Steps = append(out.Steps, s.pastStepOf(r.Context(), runID, st))
	}
	writeJSON(w, out)
}

// pastStepOf 는 단계 하나를 봉투의 한 칸으로 만든다.
//
// 로그가 없는 단계를 목록에서 빼지 않는다. 빼면 화면의 단계 수가 계약과 안
// 맞고, 왜 없는지를 말할 자리가 사라진다. 상태로 미리 걸러 요청을 아끼지도
// 않는다 - "돌았고 아무 말도 안 했다" 는 총 길이를 실제로 봐야 알고, GET log 는
// 없는 단계에도 200 에 0 을 낸다.
func (s *Server) pastStepOf(ctx context.Context, runID string, st runctl.Step) pastStep {
	out := pastStep{Seq: st.Seq, ID: st.ID, State: st.State, Chosen: st.Chosen}
	body, h, err := s.client.StepLog(ctx, runID, st.Seq, st.ID)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	// 첫 줄을 안 버린다 - ?from= 을 안 쓰므로 이 조각이 언제나 파일의 처음이다.
	res := transcript.Parse(body, false)
	out.Transcript = &res
	out.Data = string(body)
	out.Received = len(body)
	out.Source = h.Get(headerLogSource)
	// 못 읽은 헤더는 0 이다. 파싱 실패를 오류로 올리지 않는 이유는 이 값이
	// 화면의 한 줄이지 단계의 성패가 아니기 때문이다.
	out.Total, _ = strconv.ParseInt(h.Get(headerLogBytes), 10, 64)
	out.Capped = h.Get(headerLogCapped) == "1"
	return out
}
