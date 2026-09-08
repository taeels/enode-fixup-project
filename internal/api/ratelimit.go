package api

import (
	"net/http"
	"sync"
	"time"
)

// ── 요청 한도 (SECURITY-11) ──────────────────────────────────────────────
//
// 데모 인스턴스에서 무인증으로 열리는 읽기 셋에만 걸린다. 실 함대는
// s.auth 를 타므로 이 파일이 도는 일이 없다.
//
// 값의 근거 — 시청자 서른이 5초마다 다섯을 부르면 시청자당 1 rps 이고
// 함대 전체가 30 rps 다. 한도는 그 넷째 배수다. 정상 폴링이 한도에 안
// 닿는 것이 첫 조건이고, 이 한도가 막는 것은 폭주 루프와 스크립트 홍수가
// 일회용 인스턴스를 넘어뜨리는 것뿐이다.
const (
	rateLimitPerSecond = 120.0
	rateLimitBurst     = 240.0
)

// bucket 은 토큰 버킷 하나다.
//
// 나누는 키가 없다 — 읽기 셋 전체에 전역 하나다. 출발지 IP 로 나누는
// 길을 버린 이유는 대회장 관객이 한 NAT 뒤에 있기 때문이다. IP 로 나누면
// 관객 전체가 한 클라이언트로 묶여 정상 폴링이 막히고, 그것은 데모 완주를
// 직접 깨는 쪽이다. 대가는 한 클라이언트가 한도를 다 먹으면 나머지가
// 굶는 것이고 그 대가를 수락한다 — 앞쪽은 정상 동작을 막고 뒤쪽은
// 비정상 하나가 남을 막는다.
//
// 상태가 프로세스 메모리에 산다. 재기동하면 비어서 시작하는데 일회용
// 인스턴스라 그것이 문제가 아니고, 그 사실이 곧 "Mediator 를 늘리면 한도도
// 나뉜다" 는 제약이다. 늘릴 일이 생기면 한도를 밖으로 내는 결정이 먼저다.
type bucket struct {
	mu     sync.Mutex
	tokens float64
	last   time.Time
	rate   float64 // 초당 채우는 토큰 수
	burst  float64
	// now 는 시험이 갈아 끼운다. 한도 시험에 DB 도 슬립도 고루틴도
	// 두지 않으려면 시계가 필드여야 한다.
	now func() time.Time
}

// newBucket 은 가득 찬 버킷을 낸다.
//
// last 를 안 채운다 — 제로값이면 첫 호출의 경과 시간이 터무니없이 크고,
// 채우기가 버스트에서 잘리므로 결과는 "가득 찬 버킷" 그대로다. 초기화
// 한 줄을 아끼려는 것이 아니라, 시계를 갈아 끼우는 시험이 생성 시각을
// 함께 맞출 필요가 없어진다.
func newBucket(rate, burst float64) *bucket {
	return &bucket{tokens: burst, rate: rate, burst: burst, now: time.Now}
}

// allow 는 토큰 하나를 쓴다. 없으면 거짓이다.
//
// 타이머도 고루틴도 없다 — 부를 때마다 경과 시간만큼 채운다. 일회용
// 인스턴스에 배경 작업을 만들지 않는 쪽이고, 배경 작업이 없으면 종료
// 순서를 신경 쓸 자리도 없다.
func (b *bucket) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now()
	b.tokens += now.Sub(b.last).Seconds() * b.rate
	if b.tokens > b.burst {
		b.tokens = b.burst
	}
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// limit 은 데모 모드의 읽기 셋을 감싸는 미들웨어다.
//
// Retry-After 를 fail 보다 먼저 세운다 — fail 이 WriteHeader 를 하고,
// 그 뒤에 헤더를 넣으면 나가지 않는다. 순서가 규칙이다.
func (s *Server) limit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.rl.allow() {
			w.Header().Set("Retry-After", "1")
			fail(w, 429, "rate limited")
			return
		}
		next(w, r)
	}
}
