package api

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/taeels/enode/internal/store"
)

// ── GET /v1/runs — 지금 무엇이 도는가 (ADR-065) ──────────────────────────
//
// 최신이 앞이다. 지난 것을 뒤지는 자리는 Record 다.
type runsResponse struct {
	// ObservedAt 은 nodes 와 같은 규칙이다 — 행이 0 이어도 값이 선다.
	ObservedAt time.Time `json:"observed_at"`
	// Runs 는 비어 있어도 [] 다. 아무것도 안 맞는 필터와 못 읽은 것을
	// 화면이 갈라야 한다.
	Runs []store.RunRow `json:"runs"`
}

// 질의 인자 넷의 값이다.
//
//	기본 100      팩이 정한 값이다
//	상한 1000     이 표면이 답하는 것은 "지금 무엇이 도는가" 이고,
//	              그보다 큰 값을 받는 것은 지난 것을 뒤지는 일이라 Record 의 자리다
//	길이 200      state 와 work 는 등호 비교에 그대로 실린다
const (
	defaultRunLimit = 100
	maxRunLimit     = 1000
	maxFilterLen    = 200
)

// runFilter 는 질의 문자열을 좁히기 넷으로 읽는다. 나쁘면 그 이유를 낸다.
//
// "지정됐다" 의 뜻이 넷에 하나다 — 인자가 왔고 값이 비어 있지 않다.
// ?limit= 처럼 비운 것은 인자를 붙였다가 지운 흔적이지 값이 아니므로
// 미지정으로 읽는다. 필터 칸을 비우는 것이 화면의 가장 흔한 조작이고,
// 넷이 서로 다르게 굴면 화면이 인자마다 예외를 짓는다.
//
// 모르는 인자는 무시한다. 좁히기만 하는 표면이라 모르는 인자가 결과를
// 넓히지 못한다 — 이 논거가 서는 것은 필터가 넷인 이 라우트뿐이다.
//
// limit 은 넘치면 자르고 state · work 는 넘치면 거절한다. 비대칭이고
// 의도다 — limit 은 "얼마나 볼까" 라 좁혀도 뜻이 남지만, state 와 work 는
// 찾는 값 자체라 잘라 내면 다른 것을 찾은 것이 된다.
//
// 자른 것은 응답에 안 보이고 뒤로 걸어갈 페이지네이션도 없다. 그래서
// mcp 가 runs.list 도구 설명에 그 사실을 싣는다 — 서버가 말할 자리가 없다.
func runFilter(q url.Values) (store.RunFilter, string) {
	f := store.RunFilter{Limit: defaultRunLimit}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		// 0 과 음수는 거절한다. "0줄을 달라" 는 오타이지 뜻이 아니다.
		if err != nil || n <= 0 {
			return f, "invalid limit"
		}
		f.Limit = min(n, maxRunLimit)
	}
	if v := q.Get("since"); v != "" {
		// 표준시가 붙은 형태만 받는다. created_at 이 timestamptz 이므로
		// 표준시 없는 값을 서버 지역시로 짐작하면 경계가 조용히 어긋난다.
		// 포함 하한이다 — 같은 초에 만들어진 Run 이 폴링 사이에 사라지지 않는다.
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return f, "invalid since"
		}
		f.Since = t
	}
	// state 의 어휘를 여기서 검사하지 않는다. 모르는 값이 오면 빈 목록이
	// 나가고 그것이 참이다. 어휘를 박으면 QUEUED 를 더하는 queue 유닛이
	// 이 유닛을 다시 열어야 하는데, 토대 유닛이 그런 자리를 만들면 안 된다.
	f.State, f.Work = q.Get("state"), q.Get("work")
	if len(f.State) > maxFilterLen || len(f.Work) > maxFilterLen {
		return f, "invalid filter"
	}
	return f, ""
}

func (s *Server) getRuns(w http.ResponseWriter, r *http.Request) {
	f, bad := runFilter(r.URL.Query())
	if bad != "" {
		fail(w, 400, bad)
		return
	}
	runs, at, err := s.st.Runs(r.Context(), f)
	if err != nil {
		s.log.Error("cannot query runs", "err", err)
		fail(w, 503, "query failed")
		return
	}
	noStore(w)
	write(w, 200, runsResponse{ObservedAt: at, Runs: nonNil(runs)})
}
