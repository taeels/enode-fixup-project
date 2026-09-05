package enode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/taeels/enode/internal/contract"
)

// Client 는 Mediator 로 나가는 유일한 방향이다 (ADR-014 결정 2).
// enode 는 서버 포트를 열지 않는다 — 방화벽 구멍이 0 개다.
type Client struct {
	Base      string
	Token     string
	Principal string
	HTTP      *http.Client
	// Instance 는 이번 생의 표식이다 (ADR-030) — 기동마다 새로 뽑는 난수.
	// 광고와 claim 에 함께 실려, 유실된 claim 응답의 재전달(같은 생)과
	// 재시작 판정(다른 생)을 가른다.
	Instance string
	// Poll 은 롱폴 전용이다 (ADR-029). 없으면 HTTP 를 쓴다.
	//
	// 왜 갈라야 하나 — claim 은 서버가 몇 시간을 기다렸다 답할 수 있는데,
	// 짧은 타임아웃을 가진 클라이언트로 걸면 그 시간에 끊는다.
	// 끊는 그 순간 서버가 단계를 CLAIMED 로 만들고 응답을 쓰면
	// 응답이 유실되고 그 단계는 아무도 안 돌린다 — claim 은 비멱등이라
	// 재시도가 되찾지도 못한다. 실측에서 Run 이 영구히 멈췄다.
	Poll *http.Client
}

// AdvertResponse 는 광고의 응답이다.
//
// 이 응답이 임대의 갱신이자 취소 통보다 (ADR-016).
// 목록은 델타가 아니라 전부 이므로, 목록에 없으면 없는 것이다 —
// 취소를 알리는 별도의 신호가 필요 없다.
// (채우는 것은 S4. 자리를 지금 만들어 enode 쪽 계약을 안 바꾼다.)
type AdvertResponse struct {
	Leases []Lease `json:"leases"`
	// RenewSeconds 는 Mediator 가 말하는 주기다 (ADR-028).
	// 0 이면 안 온 것이고 그때는 우리 기본값을 쓴다 — 옛 Mediator 와도 돈다.
	RenewSeconds int `json:"renew_seconds"`
}

type Lease struct {
	RunID      string    `json:"run_id"`
	Node       string    `json:"node"`
	Capability string    `json:"capability"`
	NotAfter   time.Time `json:"not_after"`
	Nonce      string    `json:"nonce"`
}

func (c *Client) Advertise(ctx context.Context, a contract.Advert) (*AdvertResponse, error) {
	body, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.Base+"/v1/nodes", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-Enode-Principal", c.Principal)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("advertise rejected: %s", resp.Status)
	}
	var out AdvertResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Advertiser 는 주기적으로 광고한다.
//
// 하나의 요청이 셋을 겸한다 (ADR-016) — 광고 · 생존 신고 · 임대 갱신.
// 그래서 광고 주기 · 하트비트 주기 · 임대 갱신 주기가 한 값이고,
// 그 값이 취소가 여기 닿는 지연도 동시에 정한다.
type Advertiser struct {
	Client *Client
	Ident  Identity
	Local  Local
	// Every 는 첫 광고 전까지의 기본값이다 (ADR-028) —
	// 응답이 오면 Mediator 가 말한 주기로 바뀐다. Run 루프만 이 값을 만진다.
	Every time.Duration
	Log   *slog.Logger

	// OnLeases 는 응답의 임대 목록을 받는다. S4 가 여기에 붙는다.
	OnLeases func([]Lease)

	// OnReady 는 첫 광고가 성공한 뒤 한 번 불린다 (탄력 노드).
	//
	// 왜 필요한가 — 노드를 띄운 쪽은 「떴다」와 「쓸 수 있다」 사이를
	// 건너야 한다. 프로세스가 뜬 것과 함대에 등록된 것은 다르고, 그 사이에
	// Run 을 내면 매칭이 422 로 거절한다 (자원이 아직 없으므로).
	//
	// 폴링을 안 쓰는 이유 — 띄운 쪽이 「몇 초 기다려라」를 알아야 하고,
	// 그러면 조율이 우리 밖으로 샌다. 트래커마다 어댑터가 있으면
	// 그 규칙이 어댑터 수만큼 복제된다.
	//
	// 「지금 건강하다」가 아니라 「떴다」이다 — 나중에 광고가 실패해도
	// 되돌리지 않는다. 건강은 광고 만료가 이미 말한다 (ADR-012).
	OnReady func()
}

// Run 은 ctx 가 끝날 때까지 광고한다.
//
// 실패한 하트비트 하나는 중단 신호가 아니다 (ADR-016) —
// 권위는 not_after 이지 응답의 유무가 아니다. 뒤집으면 네트워크가 한 번
// 끊길 때마다 빌드가 죽는다. 실패하면 다시 걸고, 갱신을 못 받는 동안
// not_after 가 다가오다가 지나면 그때 멈춘다.
func (a *Advertiser) Run(ctx context.Context) {
	t := time.NewTimer(0)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}

		// 매번 다시 탐지한다 — 보드가 빠지거나 디스크가 차면 그 항목이
		// 빠진 채로 나가고, 그것이 곧 "지금은 못 한다" 다 (ADR-017 결정 3).
		ad := contract.Advert{
			NodeID:       a.Ident.NodeID,
			Label:        a.Ident.Label,
			Instance:     a.Client.Instance, // 이번 생 (ADR-030)
			Capabilities: Detect(ctx, a.Local, a.Log),
		}
		resp, err := a.Client.Advertise(ctx, ad)
		switch {
		case err != nil && ctx.Err() == nil:
			a.Log.Warn("advertise failed; retrying", "err", err)
		case err == nil:
			if a.OnLeases != nil {
				a.OnLeases(resp.Leases)
			}
			// 주기는 만료를 계산하는 쪽이 말한다 (ADR-028) —
			// 우리 플래그로 정하면 Mediator 의 만료 계산과 어긋날 수 있고,
			// 어긋나면 조용히 함대에서 사라진다 (claim 은 계속 도니까 안 보인다).
			if n := resp.RenewSeconds; n > 0 {
				if want := time.Duration(n) * time.Second; want != a.Every {
					a.Log.Info("adopting advertise interval from mediator",
						"was", a.Every, "now", want)
					a.Every = want
				}
			}
			a.Log.Debug("advertise", "node", ad.NodeID, "caps", len(ad.Capabilities), "leases", len(resp.Leases))
			// 첫 광고가 성공했다 — 이제 매칭이 이 노드를 고를 수 있다.
			if a.OnReady != nil {
				f := a.OnReady
				a.OnReady = nil // 한 번만
				f()
			}
		}
		t.Reset(a.Every)
	}
}
