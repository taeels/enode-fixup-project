package runctl

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newClient(h http.HandlerFunc) (*Client, func()) {
	srv := httptest.NewServer(h)
	return &Client{Base: srv.URL, Token: "t", Principal: "a@b", HTTP: srv.Client()}, srv.Close
}

// 와이어의 HTTP 상태를 그대로 들고 온다 — CLI 종료코드로의 번역은
// cmd/runctl 이 한다. 경계가 둘이라 하나로 통일하지 않는다 (INVARIANTS §4).
func TestFailCarriesWireCode(t *testing.T) {
	for _, code := range []int{400, 409, 422, 503} {
		c, done := newClient(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(code)
			_, _ = w.Write([]byte(`{"error":{"code":` + itoa(code) + `,"reason":"이유"}}`))
		})
		_, err := c.Status(context.Background(), "r")
		var f *Fail
		if !errors.As(err, &f) || f.Code != code || f.Reason != "이유" {
			t.Fatalf("code=%d → %v", code, err)
		}
		done()
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// 인증과 식별이 모든 요청에 실린다 (ADR-015 §1).
func TestHeaders(t *testing.T) {
	var gotAuth, gotPrincipal string
	c, done := newClient(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotPrincipal = r.Header.Get("Authorization"), r.Header.Get("X-Enode-Principal")
		_, _ = w.Write([]byte(`{"run_id":"r","state":"RUNNING"}`))
	})
	defer done()
	if _, err := c.Status(context.Background(), "r"); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer t" || gotPrincipal != "a@b" {
		t.Fatalf("auth=%q principal=%q", gotAuth, gotPrincipal)
	}
}

// Wait 는 폴링일 뿐이다 — runctl 을 상태 있게 만들지 않는다.
// 중간에 죽어도 Run 은 계속 돈다 (ADR-013 이 --interactive 를 거절한 사유).
func TestWaitUntilTerminal(t *testing.T) {
	n := 0
	c, done := newClient(func(w http.ResponseWriter, r *http.Request) {
		n++
		state := "RUNNING"
		if n >= 3 {
			state = "SUCCEEDED"
		}
		_, _ = w.Write([]byte(`{"run_id":"r","state":"` + state + `"}`))
	})
	defer done()
	run, err := c.Wait(context.Background(), "r", time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if run.State != "SUCCEEDED" || n < 3 {
		t.Fatalf("state=%s polls=%d", run.State, n)
	}
}

func TestTerminal(t *testing.T) {
	for s, want := range map[string]bool{
		"SUCCEEDED": true, "FAILED": true,
		"RUNNING": false, "VERIFYING": false, "RESOLVING": false,
	} {
		if Terminal(s) != want {
			t.Errorf("Terminal(%s)=%v", s, !want)
		}
	}
}
