package enode

import (
	"testing"
	"time"
)

// 임대 장부는 하트비트 응답으로 통째로 교체된다 (ADR-016).
//
// 목록은 델타가 아니라 전부이므로, 목록에서 빠지는 것이 곧 취소 통보다.
// 병합하면 그 뜻이 사라진다 — 회수된 Run 의 임대가 장부에 영원히 남고,
// 노드는 이미 남에게 넘어간 자원 위에서 계속 돈다.
func TestHeld_SetReplacesItDoesNotMerge(t *testing.T) {
	h := NewHeld()
	h.Add(Lease{RunID: "r1", NotAfter: time.Now().Add(time.Hour)})
	h.Add(Lease{RunID: "r2", NotAfter: time.Now().Add(time.Hour)})

	// 하트비트가 r2 만 돌려줬다 — r1 은 취소된 것이다.
	h.Set([]Lease{{RunID: "r2", NotAfter: time.Now().Add(time.Hour)}})

	if _, ok := h.Valid("r1"); ok {
		t.Fatal("ADR-016 violated: a lease absent from the heartbeat survived the replacement")
	}
	if _, ok := h.Valid("r2"); !ok {
		t.Fatal("ADR-016 violated: a lease the heartbeat renewed was dropped")
	}
}

// 권위는 not_after 이지 응답의 유무가 아니다 (ADR-016).
//
// 하트비트가 한 번 실패했다고 멈추지 않는다 — not_after 는 갱신 주기의
// 배수라 여러 번 놓쳐야 지난다. 뒤집으면 네트워크가 한 번 끊길 때마다
// 빌드가 죽는다.
func TestHeld_TheDeadlineDecidesNotThePresenceOfTheEntry(t *testing.T) {
	h := NewHeld()
	h.Set([]Lease{
		{RunID: "fresh", NotAfter: time.Now().Add(time.Hour)},
		{RunID: "stale", NotAfter: time.Now().Add(-time.Second)},
	})

	if _, ok := h.Valid("fresh"); !ok {
		t.Fatal("ADR-016 violated: a lease inside its not_after was judged expired")
	}
	if _, ok := h.Valid("stale"); ok {
		t.Fatal("ADR-010 violated: a lease past its not_after was still judged valid")
	}
	if _, ok := h.Valid("never-seen"); ok {
		t.Fatal("ADR-016 violated: a run that was never in the ledger holds a lease")
	}
}

// EverHeld 는 Set 이 아니라 Add 에서 센다.
//
// Set 은 광고 응답이고 주기가 길다 (기본 60초). 그보다 짧은 Run 은 광고가
// 임대를 한 번도 못 본다 — 실측에서 밟았다 (2026-08-24): --once 로 띄운
// 탄력 노드가 Run 을 마쳤는데 종료하지 않았다. claim 은 임대를 확실히 지난다.
func TestHeld_EverHeldIsCountedOnTheClaimNotTheAdvert(t *testing.T) {
	h := NewHeld()
	if h.EverHeld() {
		t.Fatal("a fresh ledger claims it already held work")
	}

	// 광고가 두 번 돌았는데 그 사이의 Run 은 이미 끝나 있었다 —
	// 빈 목록을 두 번 본다. 이것으로 세면 --once 노드가 안 죽는다.
	h.Set(nil)
	h.Set(nil)
	if h.EverHeld() {
		t.Fatal("an advert that saw no lease was counted as having held work; --once would never exit")
	}

	h.Add(Lease{RunID: "r1", NotAfter: time.Now().Add(time.Hour)})
	if !h.EverHeld() {
		t.Fatal("a claimed lease was not counted; --once would run forever")
	}

	// 그 뒤 광고가 빈 목록을 돌려줘도 「한 번이라도 집었다」는 안 지워진다.
	h.Set(nil)
	if !h.EverHeld() {
		t.Fatal("the fact that work was once held was erased by a later advert")
	}
}
