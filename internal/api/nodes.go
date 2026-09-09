package api

import (
	"net/http"
	"time"

	"github.com/taeels/enode/internal/store"
)

// ── GET /v1/nodes — 관측된 함대 (ADR-065) ────────────────────────────────
//
// 한 시점의 스냅샷이다. 판정이 아니다 — "이 노드는 왜 안 맞나" 를 서버가
// 내지 않고, 매처도 안 돈다. 관측 경로에서 매처를 돌리면 관측과 배정이
// 다시 붙고, 두 시점이 다르므로 그 판정은 내는 순간 낡는다.
type nodesResponse struct {
	// ObservedAt 은 DB 의 시계다. 프로세스 시각으로 내면 lease.not_after
	// (DB 가 계산한 값)와 다른 시계가 되고, 화면의 카운트다운이 그 차이만큼
	// 어긋난다. 노드가 0 이어도 값이 선다.
	ObservedAt time.Time `json:"observed_at"`
	// Nodes 는 비어 있어도 [] 다. null 도 404 도 아니다 — 화면이
	// "없음" 과 "못 읽음" 을 갈라야 한다.
	Nodes []store.NodeView `json:"nodes"`
}

// nonNil 은 nil 슬라이스를 빈 슬라이스로 접는다.
//
// 분기가 아니라 append 인 이유 — nil 이 아닌 값이 오는 것이 정상 경로라
// 분기로 두면 그 몸통이 영영 안 밟히는 줄로 남는다. 여기서 접는 이유는
// 이 규약("키는 언제나 있고 목록은 빈 배열이다")이 저장소가 아니라 HTTP
// 표면의 약속이기 때문이다.
func nonNil[T any](s []T) []T { return append([]T{}, s...) }

// noStore 는 폴링 응답이 캐시에 앉지 않게 한다.
//
// 화면이 5초마다 다시 묻는데 중간 캐시가 한 번 잡아 두면 함대가 멈춘
// 것처럼 보인다. GET /v1/nodes 에 ?t=... 같은 인자를 붙여 캐시를 피하는
// 길은 이 라우트에서 400 이므로, 헤더가 그 자리를 대신한다.
func noStore(w http.ResponseWriter) { w.Header().Set("Cache-Control", "no-store") }

func (s *Server) getNodes(w http.ResponseWriter, r *http.Request) {
	// 이 라우트는 필터를 안 받는다. 하나라도 오면 거절이다 — 정본이
	// "?free=true 같은 질의는 400" 으로 적었고, 여유 질의를 만들지 않겠다는
	// 결정이 표면에서 눈에 보이는 자리가 여기다.
	//
	// GET /v1/runs 는 모르는 인자를 무시한다. 갈라 두는 것이 옳다 —
	// "좁히기만 하므로 무시해도 된다" 는 논거는 필터가 넷인 그쪽의 것이고,
	// 필터가 0 인 이 라우트로 옮겨오지 않는다.
	if len(r.URL.Query()) > 0 {
		fail(w, 400, "unknown query parameter")
		return
	}
	nodes, at, err := s.st.Nodes(r.Context())
	if err != nil {
		s.log.Error("cannot query nodes", "err", err)
		fail(w, 503, "query failed")
		return
	}
	noStore(w)
	write(w, 200, nodesResponse{ObservedAt: at, Nodes: nonNil(nodes)})
}
