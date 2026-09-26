package enode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
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
	// Upload 는 요청마다의 제한이 없는 client 다 (FR-3 · N3). 마감은 업로드 예산의
	// ctx 가 준다. PutBlob 과 UploadLog 가 쓴다. nil 이면 HTTP 를 쓴다 (시험 · 옛 조립).
	//
	// HTTP 의 요청마다 30초가 큰 산출물을 끊었다 — 예산 3분 안에서도 한 파일이 30초를
	// 넘으면 실패였다. 롱폴처럼 칸을 나눈다: 한쪽을 조이면 다른 쪽이 따라 조여지면 안 된다.
	Upload *http.Client
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
	// Drain 은 중앙이 받아 적은 정책이다 (ADR-063 §4.1). 통보이지 판정이 아니다 —
	// 「받았다」는 뜻뿐이고 소유자를 가리지 않는다. 이 값이 Worker 의 것이다:
	// 파일에 썼다고 중앙이 알았다는 뜻은 아니므로 파일이 아니라 응답을 본다.
	Drain string `json:"drain"`
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

	// Caps 는 지금 광고할 능력을 돌려준다 (ADR-068).
	//
	// 여기서 탐지하지 않는다 — 탐지의 일부는 외부 프로세스를 띄우고,
	// 그것이 안 돌아오면 이 루프가 함께 멈춘다. 그러면 하트비트가 끊기고
	// 노드는 프로세스도 로그도 정상인 채로 함대에서 사라진다.
	// 비싼 것은 Detector 가 자기 시계로 갱신하고 여기서는 읽기만 한다.
	Caps func() Capabilities
	// Every 는 첫 광고 전까지의 기본값이다 (ADR-028) —
	// 응답이 오면 Mediator 가 말한 주기로 바뀐다. Run 루프만 이 값을 만진다.
	Every time.Duration
	Log   *slog.Logger

	// OnLeases 는 응답의 임대 목록을 받는다. S4 가 여기에 붙는다.
	OnLeases func([]Lease)

	// Held 는 응답의 drain 과 광고 주기를 Worker 에 나르는 자리다 (ADR-063 §4).
	// Advertiser 와 Worker 는 다른 고루틴이고 둘을 잇는 것은 이것뿐이다.
	// nil 이면 안 나른다 — 정책을 모르는 호출자는 오늘 그대로 돈다.
	Held *Held

	// Local 은 여유 부족 drain 의 재료다 — 워크스페이스와 min_free_gb (trash 유닛 · ADR-063 §4).
	// 워크스페이스가 없으면 여유를 안 보고 걸지 않는다.
	Local Local

	// Status 는 상태 파일의 칸을 모으는 자리다 (ADR-068 = A). 삭제자도 같은 것에 쓴다.
	// nil 이면 첫 광고에서 스스로 만든다 — 설정 경로가 있을 때만.
	Status *StatusBook

	// policy 는 광고 직전마다 읽는 정책 파일이다. 비어 있으면 첫 광고에서
	// Ident.Config 옆의 파일로 만든다 — 설정이 없는 시험은 안 읽는다.
	policy *policyReader

	// free 는 여유를 재는 자리다. nil 이면 freeBytes. 시험이 바꿔 끼운다.
	free func(path string) (uint64, error)
	// diskHeld 는 앞 광고에서 여유 부족 drain 을 걸었나다 — 거는 선과 푸는 선이 다르다.
	diskHeld bool
	// diskWarn 은 여유를 못 잰 원인이다. 바뀔 때만 적는다.
	diskWarn string

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

		// 매번 전부 보낸다 — 보드가 빠지면 그 항목이 빠진 채로 나가고, 그것이
		// 곧 "지금은 못 한다" 다 (ADR-017 결정 3).
		//
		// 보내는 규칙은 그대로이고 알아내는 자리만 옮겼다 (ADR-068).
		// 디스크 여유는 능력이 아니라 drain 이다 (trash 유닛) — 광고마다 새로 재서,
		// 빌드가 도는 동안 디스크가 차면 다음 광고에서 노드 전체가 빠진다.
		snap := a.Caps()
		drain := a.drain(time.Now())
		// 제어판이 읽을 상태 파일에 탐지 능력과 drain 의 출처를 남긴다 (ADR-068 = A).
		// 광고 경로 옆에 두는 것은 여기서 둘을 이미 들기 때문이다 — 못 써도
		// 막지 않는다. 능력은 광고가 이미 진다.
		a.status().SetCaps(snap)
		a.status().SetDrain(drain)
		ad := contract.Advert{
			NodeID:       a.Ident.NodeID,
			Label:        a.Ident.Label,
			Instance:     a.Client.Instance, // 이번 생 (ADR-030)
			Capabilities: snap.Caps,
			// 출처를 합친 값 하나만 싣는다 (business-rules.md 6.3). 안 걸렸으면 제로값이라
			// policy 키가 안 나간다 (omitzero).
			Policy: contract.Policy{Drain: drain.Effective},
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
			// 중앙이 받아 적은 정책을 Worker 에 나른다 (ADR-063 §4.1). 바뀔 때만
			// 찍는다 — 같은 값이 광고마다 오는 것이 정상이다.
			if a.Held != nil {
				if a.Held.SetDrain(resp.Drain) {
					a.Log.Info("drain acknowledged by mediator", "mode", resp.Drain)
				}
				a.Held.SetRenew(a.Every)
			}
			// 값의 나이를 함께 찍는다 — 비싼 탐지는 자기 주기로 도므로
			// 여기 실린 harness·repo 는 「지금」이 아니다. 그 사실이 안
			// 보이면 「왜 로그아웃했는데 아직 광고에 있지」가 미궁이 된다.
			a.Log.Debug("advertise", "node", ad.NodeID, "caps", len(ad.Capabilities),
				"leases", len(resp.Leases), "caps_age", time.Since(snap.At).Round(time.Second))
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

// readPolicy 는 이번 광고에 실을 정책이다. 설정 경로가 없으면(시험) 안 읽는다.
func (a *Advertiser) readPolicy() contract.Policy {
	if a.policy == nil {
		if a.Ident.Config == "" {
			return contract.Policy{}
		}
		a.policy = &policyReader{Path: PolicyPath(a.Ident.Config), Log: a.Log}
	}
	return a.policy.Read()
}

// status 는 상태 파일의 자리다. 설정 경로가 없으면(시험) nil 이고 nil 은 아무것도 안 쓴다.
func (a *Advertiser) status() *StatusBook {
	if a.Status == nil && a.Ident.Config != "" {
		a.Status = NewStatusBook(a.Ident.Config, a.Log)
	}
	return a.Status
}

// drain 은 이번 광고에 실을 drain 과 그 출처다 (business-logic-model.md 4절). 소유자 출처는
// 정책 파일에서 · 여유 부족 출처는 워크스페이스의 여유에서 온다. 합친 값은 센 쪽 하나다.
func (a *Advertiser) drain(now time.Time) DrainStatus {
	var sources []DrainSource
	if src, ok := OwnerDrain(a.readPolicy().Drain); ok {
		sources = append(sources, src)
	}
	if src, ok := a.diskSource(); ok {
		sources = append(sources, src)
	}
	return DrainStatus{Effective: combineDrain(sources), Sources: sources, At: now.UTC()}
}

// diskSource 는 여유 부족 출처다 (business-rules.md 6.1 · 6.2). 워크스페이스가 없으면 안 잰다.
// 못 재면 넉넉한 것으로 치고 원인이 바뀔 때만 적는다. 걸기와 풀기가 바뀌면 로그 한 줄이다.
func (a *Advertiser) diskSource() (DrainSource, bool) {
	path := a.Local.Workspace
	if path == "" {
		return DrainSource{}, false
	}
	measure := a.free
	if measure == nil {
		measure = freeBytes
	}
	free, err := measure(path)
	measured := err == nil
	if err != nil {
		if reason := err.Error(); reason != a.diskWarn {
			a.diskWarn = reason
			a.Log.Warn("cannot measure free disk; assuming enough", "path", path, "err", err)
		}
		free = math.MaxUint64
	} else {
		a.diskWarn = ""
	}
	src, ok := diskDrain(free, a.Local.MinFreeGB, a.diskHeld)
	if ok != a.diskHeld {
		a.diskHeld = ok
		var freeGB any = "unknown"
		if measured {
			freeGB = free / (1 << 30)
		}
		if ok {
			a.Log.Warn("free disk below min_free_gb; draining this node", "free_gb", freeGB, "min_gb", a.Local.MinFreeGB)
		} else {
			a.Log.Info("free disk recovered; lifting the disk drain", "free_gb", freeGB, "min_gb", a.Local.MinFreeGB)
		}
	}
	return src, ok
}
