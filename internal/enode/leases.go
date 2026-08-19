package enode

import (
	"sync"
	"time"
)

// Held 는 이 노드가 지금 들고 있는 임대다.
//
// ★ 하트비트 응답으로 통째로 교체된다 ★ (ADR-016) — 목록은 델타가 아니라
// 전부이므로, 목록에서 빠지는 것이 곧 취소 통보다. 병합하면 그 뜻이 사라진다.
type Held struct {
	mu     sync.RWMutex
	byRun  map[string]Lease
	lastOK time.Time
}

func NewHeld() *Held { return &Held{byRun: map[string]Lease{}} }

// Set 은 ★ 교체한다 ★. 없어진 임대는 없어진 것이다.
func (h *Held) Set(ls []Lease) {
	h.mu.Lock()
	defer h.mu.Unlock()
	m := make(map[string]Lease, len(ls))
	for _, l := range ls {
		m[l.RunID] = l
	}
	h.byRun = m
	h.lastOK = time.Now()
}

// Add 는 claim 응답으로 받은 아티팩트를 즉시 반영한다.
// 다음 하트비트를 기다리면 그 사이 단계를 못 시작한다.
func (h *Held) Add(l Lease) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.byRun[l.RunID] = l
}

// Valid 는 이 Run 의 단계를 지금 시작해도 되는지다.
//
// ★ 권위는 not_after 이지 응답의 유무가 아니다 ★ (ADR-016) —
// 하트비트가 한 번 실패했다고 멈추지 않는다. 갱신을 못 받는 동안 not_after 가
// 다가오고, 지나면 그때 멈춘다. 뒤집으면 네트워크가 한 번 끊길 때마다 빌드가 죽는다.
func (h *Held) Valid(runID string) (Lease, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	l, ok := h.byRun[runID]
	if !ok {
		return Lease{}, false // 목록에 없으면 없는 것이다 — 취소되었거나 회수되었다
	}
	return l, time.Now().Before(l.NotAfter)
}
