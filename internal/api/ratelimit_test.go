package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// 이 파일에는 DB 도 슬립도 없다. 시계를 필드로 둔 이유가 그것이다 —
// 한도를 진짜 시간으로 재면 시험이 초 단위로 느려지고, 느린 시험은
// 결국 안 도는 시험이 된다.

// at 은 시험이 손으로 미는 시계다.
func at(b *bucket) func(time.Duration) {
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	b.now = func() time.Time { return now }
	return func(d time.Duration) { now = now.Add(d) }
}

func TestBucketHandsOutTheBurstAndThenStops(t *testing.T) {
	b := newBucket(rateLimitPerSecond, rateLimitBurst)
	at(b)
	for i := 0; i < int(rateLimitBurst); i++ {
		if !b.allow() {
			t.Fatalf("request %d was denied while the burst is %v", i+1, rateLimitBurst)
		}
	}
	if b.allow() {
		t.Fatal("the bucket handed out more than the burst with no time passing")
	}
}

func TestBucketRefillsWithTheTimeThatPassed(t *testing.T) {
	b := newBucket(10, 10)
	pass := at(b)
	for i := 0; i < 10; i++ {
		b.allow()
	}
	if b.allow() {
		t.Fatal("the bucket was empty and still allowed a request")
	}
	pass(500 * time.Millisecond) // 10/s for half a second is five tokens
	for i := 0; i < 5; i++ {
		if !b.allow() {
			t.Fatalf("token %d did not come back after half a second at 10 per second", i+1)
		}
	}
	if b.allow() {
		t.Fatal("more tokens came back than the elapsed time paid for")
	}
}

func TestBucketDoesNotBankMoreThanTheBurst(t *testing.T) {
	b := newBucket(10, 10)
	pass := at(b)
	b.allow()
	pass(time.Hour) // an idle hour is 36000 tokens if nothing clamps it
	for i := 0; i < 10; i++ {
		if !b.allow() {
			t.Fatalf("token %d was denied right after an idle hour", i+1)
		}
	}
	if b.allow() {
		t.Fatal("an idle hour banked more than the burst")
	}
}

// 한도는 미들웨어다 — 통과한 요청은 손대지 않고, 거절은 429 와
// Retry-After 로 다시 부를 시점을 말한다.
func TestLimitPassesUnderTheLimitAndRejectsOverIt(t *testing.T) {
	s := &Server{rl: newBucket(1, 1)}
	at(s.rl)
	h := s.limit(func(w http.ResponseWriter, r *http.Request) { write(w, 200, map[string]string{"ok": "yes"}) })

	first := httptest.NewRecorder()
	h(first, httptest.NewRequest("GET", "/v1/nodes", nil))
	if first.Code != 200 {
		t.Fatalf("code=%d, want 200 — a request under the limit did not reach the handler", first.Code)
	}

	over := httptest.NewRecorder()
	h(over, httptest.NewRequest("GET", "/v1/nodes", nil))
	if over.Code != 429 {
		t.Fatalf("code=%d, want 429", over.Code)
	}
	if got := over.Header().Get("Retry-After"); got != "1" {
		t.Fatalf("Retry-After=%q, want \"1\" — the caller is not told when to come back", got)
	}
	var b errBody
	if err := json.Unmarshal(over.Body.Bytes(), &b); err != nil {
		t.Fatalf("the rejection body is not the usual error shape: %v", err)
	}
	if b.Error.Code != 429 || b.Error.Reason != "rate limited" {
		t.Fatalf("error=%+v, want code 429 and reason \"rate limited\"", b.Error)
	}
}
