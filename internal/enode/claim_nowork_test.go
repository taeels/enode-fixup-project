package enode

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/taeels/enode/internal/contract"
)

// 204 는 오류가 아니다 — 롱폴이 「지금은 할 일이 없다」로 답한 것이고,
// Worker.Run 은 그 자리에서 즉시 다시 건다 (ADR-015 §5).
//
// 이 갈래는 이 파일이 생기기 전에도 커버리지에 나타났다 나타나지 않았다 했다.
// 밟은 것이 테스트가 아니라 경합이었기 때문이다 — 다른 테스트가 Worker.Run 을
// 고루틴으로 띄운 채 취소했고, 취소가 claim 왕복의 어디에 떨어지느냐에 따라
// 이 case 를 지나기도 하고 안 지나기도 했다. AC3.4.6 이 요구하는 것은
// 「두 번 재면 같다」이므로, 여기서 확정적으로 먼저 밟는다.
//
// 덤으로 계약 하나가 단언을 얻는다 — 지금까지 「204 를 받으면 재시도한다」를
// 확인하는 테스트가 없었다. 204 를 오류로 처리하도록 바꾸면 이 테스트가 잡는다.
func TestWorkerRun_NoWorkIsNotAnErrorAndTheWorkerReclaims(t *testing.T) {
	var claims atomic.Int64
	enough := make(chan struct{})
	var once bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if claims.Add(1) == 3 && !once {
			once = true
			close(enough)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	w := &Worker{
		Client: &Client{Base: srv.URL, Token: "t", Principal: "p", HTTP: srv.Client()},
		Ident:  Identity{NodeID: "n1"},
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		w.Run(ctx)
	}()

	select {
	case <-enough:
	case <-time.After(10 * time.Second):
		cancel()
		<-done
		t.Fatalf("the worker stopped claiming after %d attempts; 204 must not end the loop", claims.Load())
	}

	cancel()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Worker.Run did not return after cancellation")
	}

	if got := claims.Load(); got < 3 {
		t.Fatalf("204 must send the worker straight back: got %d claims, want at least 3", got)
	}
}

// sortStrings 의 교환 갈래는 지금까지 맵 순회 순서에 걸려 있었다.
//
// 부르는 쪽이 맵을 훑어 목록을 만드는데 Go 는 맵 순회 순서를 무작위화하므로,
// 목록이 이미 정렬돼 들어오면 안쪽 교환이 한 번도 안 돌고 그 블록이 안 덮인다.
// 실행마다 갈리는 것이 그 이유다 — 정렬 함수를 직접 부르는 테스트가 없었다.
//
// 손으로 쓴 삽입 정렬이고 표준 라이브러리를 안 쓰는 자리이므로, 무작위가
// 아니라 이름으로 그 성질을 건다.
func TestSortStrings_OrdersInPlaceIncludingTheAlreadySortedCase(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"reversed", []string{"c", "b", "a"}, []string{"a", "b", "c"}},
		{"already sorted", []string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"one swap in the middle", []string{"a", "c", "b", "d"}, []string{"a", "b", "c", "d"}},
		{"duplicates keep their count", []string{"b", "a", "b"}, []string{"a", "b", "b"}},
		{"empty", []string{}, []string{}},
		{"single", []string{"only"}, []string{"only"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := append([]string(nil), c.in...)
			sortStrings(got)
			if len(got) != len(c.want) {
				t.Fatalf("sortStrings(%v): got %v, want %v", c.in, got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("sortStrings(%v): got %v, want %v", c.in, got, c.want)
				}
			}
		})
	}
}

// Advertise 의 전송 실패 갈래도 같은 이유로 실행마다 갈렸다.
//
// 광고자는 주기적으로 도는 고루틴이고, 다른 테스트가 그것을 띄운 채 서버를
// 닫거나 컨텍스트를 취소하면 `c.HTTP.Do` 가 실패한다. 그 실패가 취소보다
// 먼저 오느냐 나중에 오느냐가 실행마다 달랐다 — 전체 스위트를 아홉 번 재면
// `internal/enode` 가 1293 과 1294 사이를 오갔고, 갈리는 블록이 이 하나였다.
//
// 여기서 확정적으로 밟는다. 닿을 수 없는 주소로 한 번 부르면 끝이다.
// 덤으로 계약 하나가 단언을 얻는다 — 전송이 실패하면 그것을 삼키지 않고
// 호출자에게 돌려준다. 삼키면 광고자가 「광고했다」고 믿은 채 계속 돈다.
func TestAdvertise_ATransportFailureReachesTheCaller(t *testing.T) {
	// 서버를 세웠다 바로 닫는다 — 그 주소로는 아무도 응답하지 않는다.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	base := srv.URL
	srv.Close()

	c := &Client{Base: base, Token: "t", Principal: "p", HTTP: &http.Client{Timeout: 2 * time.Second}}
	out, err := c.Advertise(context.Background(), contract.Advert{NodeID: "n1"})
	if err == nil {
		t.Fatal("Advertise to a closed server: got nil error, want the transport failure surfaced")
	}
	if out != nil {
		t.Fatalf("a failed advertise must not return a response: got %+v", out)
	}
}
