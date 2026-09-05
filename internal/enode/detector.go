package enode

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/taeels/enode/internal/contract"
)

// DefaultDetectEvery 는 비싼 탐지를 다시 도는 기본 주기다.
//
// 왜 광고 주기와 다른 값인가 (ADR-068)
//
// 예전에는 탐지가 광고에 딸려 있어서 한 값이 둘을 정했다. 그런데 그 둘은
// 서로 다른 것에 매여 있다 — 광고 주기는 Mediator 의 만료 계산에 맞아야 하고
// (ADR-028), 탐지 주기는 알아내려는 사실이 얼마나 빨리 변하느냐에 맞아야 한다.
// 한 손잡이가 둘을 움직이면 Mediator 가 renew_seconds 를 낮출 때 아무도 고른
// 적 없는 경로로 하네스 탐지까지 촘촘해진다.
//
// 5분인 이유 — 여기서 알아내는 것은 하네스 로그인 상태와 저장소 신원이고,
// 둘 다 사람이 손대야 바뀐다. 로그아웃한 노드가 최대 5분 동안 광고에 남는데,
// 그때 잡힌 Run 은 실행 시점에 _cannot 으로 드러난다 (ADR-038) — 조용히
// 틀리는 것이 아니라 늦게 아는 것이고, 늦게 아는 값으로 5분은 싸다.
//
// 값싼 탐지는 이 주기를 안 탄다. 디스크 여유는 광고마다 새로 본다 —
// 그것이 동적이라는 것이 ADR-017 결정 3 의 근거였다.
const DefaultDetectEvery = 5 * time.Minute

// Capabilities 는 마지막으로 알아낸 능력과 그것을 알아낸 시각이다.
//
// 시각을 함께 드는 이유 — 광고에 실리는 값이 「지금」이 아니게 되었으므로,
// 얼마나 오래됐는지 말할 수 있어야 한다. 오늘은 로그가 읽는다. 광고 봉투에는
// 안 싣는다 — 매처가 쓸 것이 아니고, 싣는 순간 프로토콜이 는다.
type Capabilities struct {
	Caps []contract.Capability
	At   time.Time
}

// Detector 는 비싼 탐지를 광고와 다른 시계로 돈다 (ADR-068).
//
// 왜 고루틴을 가르나 — 탐지의 일부는 외부 프로세스를 띄우고, 그것이 안
// 돌아오면 부르는 쪽이 함께 멈춘다. 광고 루프가 멈추면 하트비트가 끊기고
// 노드는 프로세스도 로그도 정상인 채로 함대에서 사라진다.
//
// ADR-016 이 claim 과 광고를 가른 논거가 그대로 성립한다 — 주기가 다르고
// 의미가 다른 두 일을 한 고루틴에 두면 느린 쪽이 빠른 쪽을 잡아먹는다.
type Detector struct {
	Local Local
	Log   *slog.Logger
	// Every 는 비싼 탐지를 다시 도는 주기다. 0 이면 DefaultDetectEvery.
	Every time.Duration

	mu     sync.RWMutex
	costly map[string]string
	at     time.Time
}

// NewDetector 는 비싼 탐지를 한 번 돌고 나서 돌려준다.
//
// 왜 동기로 한 번 도나 — 첫 광고가 빈 능력으로 나가면 안 된다. 광고가 곧
// 능력이므로(ADR-012) 빈 광고는 "아무것도 못 한다" 는 선언이고, 그 사이에
// 들어온 Run 은 422 로 거절된다. 노드가 뜨자마자 스스로를 못 쓰게 만드는 셈이다.
//
// 부르는 쪽이 이 결과를 기동 로그에 쓴다 — 예전에는 그 로그를 위해 Detect 를
// 한 번 더 불렀고, 그래서 기동 순간에 같은 외부 프로세스가 두 벌 떴다.
func NewDetector(ctx context.Context, l Local, every time.Duration, log *slog.Logger) *Detector {
	d := &Detector{Local: l, Log: log, Every: every}
	d.refresh(ctx)
	return d
}

// Capabilities 는 지금 광고할 능력이다.
//
// 값싼 것은 여기서 새로 보고, 비싼 것은 Run 이 갱신해 둔 것을 쓴다.
// 이 함수는 외부 프로세스를 안 띄운다 — 그것이 이 갈래의 요점이다.
func (d *Detector) Capabilities() Capabilities {
	d.mu.RLock()
	costly, at := d.costly, d.at
	d.mu.RUnlock()
	return Capabilities{
		Caps: capabilities(d.Local, d.Log, costly, cheapAttrs(d.Local, d.Log)),
		At:   at,
	}
}

// Run 은 ctx 가 끝날 때까지 비싼 탐지를 자기 주기로 다시 돈다.
//
// 첫 탐지는 NewDetector 가 이미 했으므로 여기서는 기다렸다가 시작한다.
func (d *Detector) Run(ctx context.Context) {
	every := d.Every
	if every <= 0 {
		every = DefaultDetectEvery
	}
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		d.refresh(ctx)
	}
}

func (d *Detector) refresh(ctx context.Context) {
	// 탐지는 잠금 밖에서 한다 — 여기가 오래 걸릴 수 있고, 그동안 광고가
	// 마지막 값을 읽는 것을 막으면 가른 의미가 없다.
	costly := costlyAttrs(ctx, d.Local, d.Log)
	if ctx.Err() != nil {
		// 취소 중에 얻은 결과는 「못 한다」가 아니라 「못 물어봤다」다.
		// 그것을 능력으로 쓰면 종료하는 노드가 마지막 광고에서 스스로를
		// 지운다 — ADR-012 의 「빼는 것이 못 한다는 뜻」을 거짓으로 만든다.
		return
	}
	d.mu.Lock()
	d.costly, d.at = costly, time.Now()
	d.mu.Unlock()
}
