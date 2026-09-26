package enode

import (
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/scratch"
)

// 상태 파일 (ADR-068 = A).
//
// 탐지 능력과 그것을 알아낸 시각은 데몬 메모리(Detector 의 Capabilities{Caps,At})
// 에만 있고 광고에 안 실린다 — 매처가 쓸 것이 아니고 싣는 순간 프로토콜이 는다
// (ADR-068). 그런데 제어판은 「무엇을 할 수 있나」와 「언제 잰 값인가」를 함께
// 보여야 한다(enode-features 3.1.2). 제어판은 데몬과 다른 프로세스이고 데몬은
// 나가는 http.Client 만 있어 그 값을 읽을 통로가 없다.
//
// 그래서 데몬이 이 파일에 쓰고 제어판이 읽는다. 정책 파일과 같은 자리·같은
// 권한이라 새 개념이 없다 — 같은 기계의 제어판↔데몬 통로를 이 저장소는 이미
// 정책 파일로 정해 뒀고(ADR-063), 이것은 그 방향만 뒤집은 것이다.

// Status 는 상태 파일에 담기는 것이다. drain 과 scratch 는 trash 유닛이 더했다 — 옛 상태
// 파일(칸 둘)도 그대로 읽히고 그때 두 칸은 nil 이다.
type Status struct {
	Caps    []contract.Capability `yaml:"caps"`
	At      time.Time             `yaml:"at"`
	Drain   *DrainStatus          `yaml:"drain,omitempty"`   // 이번 광고에 실은 drain 과 출처
	Scratch *scratch.Usage        `yaml:"scratch,omitempty"` // scratch 가 없는 노드(native)는 없다
}

// StatusPath 는 설정 파일 옆의 상태 파일이다 — <dir>/<stem>.status.yaml.
// PolicyPath 와 같은 규칙이다 (설정 경로가 곧 노드의 신원이므로 · ADR-017).
func StatusPath(configPath string) string {
	return strings.TrimSuffix(configPath, filepath.Ext(configPath)) + ".status.yaml"
}

// WriteStatus 는 상태 파일 전체를 쓴다.
//
// tmp 에 쓰고 rename 으로 바꾼다 — 제어판이 반쯤 쓴 파일을 읽지 않게. 권한은
// 정책 파일과 같다(유닉스 0600 · 윈도우는 그 디렉터리의 상속 ACL 이라 모드
// 비트를 안 건다). 데몬에서 부르는 자리는 StatusBook 하나다 — 값이 바뀔 때만 쓴다.
func WriteStatus(configPath string, s Status) error {
	b, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	path := StatusPath(configPath)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ReadStatus 는 제어판이 읽는다. 없으면 그 사실을 오류로 돌려준다 —
// 부르는 쪽이 「아직 모름」으로 그린다.
func ReadStatus(configPath string) (Status, error) {
	b, err := os.ReadFile(StatusPath(configPath))
	if err != nil {
		return Status{}, err
	}
	var s Status
	if err := yaml.Unmarshal(b, &s); err != nil {
		return Status{}, err
	}
	return s, nil
}

// StatusBook 은 상태 파일의 칸을 한 자리에 모은다 (답 8 = A). 어느 칸이든 값이 바뀌면 파일
// 전체를 쓴다. 같은 값이면 안 쓴다.
//
//	SetCaps     광고 주기.  탐지 시각이 바뀔 때 (광고 60초 · 탐지 5분이라 매 광고 쓰기는 낭비다)
//	SetDrain    광고 주기.  실린 값이나 출처가 바뀔 때.  At 은 그 값을 정한 광고의 시각으로 남는다
//	SetScratch  삭제자.  목록 · 측정 · 지움 시작과 끝
//
// Advertiser 와 삭제자가 다른 고루틴에서 부르므로 잠금 하나를 쥔다 — 칸을 한 벌로 들고
// 파일을 통째로 쓰므로 한쪽이 다른 쪽의 칸을 덮지 않는다. 못 쓰면 경고를 원인이 바뀔 때만
// 적고, 다음 부름이 값이 같아도 다시 쓴다. 광고와 삭제는 안 막는다 — 능력은 광고가 이미 진다.
//
// nil 이면 아무것도 안 한다 — 설정 경로가 없는 시험이 오늘 그대로 돈다.
type StatusBook struct {
	mu     sync.Mutex
	config string
	log    *slog.Logger
	now    Status
	dirty  bool   // 마지막 쓰기가 실패했다
	warn   string // 마지막으로 적은 쓰기 실패의 원인
}

// NewStatusBook 은 설정 파일 옆의 상태 파일을 쓰는 자리다.
func NewStatusBook(configPath string, log *slog.Logger) *StatusBook {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &StatusBook{config: configPath, log: log}
}

// SetCaps 는 탐지 능력과 그 시각을 적는다. 시각이 같으면 안 쓴다.
func (b *StatusBook) SetCaps(c Capabilities) {
	b.set(func(s *Status) bool {
		if c.At.Equal(s.At) {
			return false
		}
		s.Caps, s.At = c.Caps, c.At
		return true
	})
}

// SetDrain 은 이번 광고에 실은 drain 과 출처를 적는다. 실린 값과 출처가 같으면 안 쓴다.
func (b *StatusBook) SetDrain(d DrainStatus) {
	b.set(func(s *Status) bool {
		if s.Drain != nil && s.Drain.Effective == d.Effective && slices.Equal(s.Drain.Sources, d.Sources) {
			return false
		}
		s.Drain = &d
		return true
	})
}

// SetScratch 는 scratch 의 양을 적는다.
func (b *StatusBook) SetScratch(u scratch.Usage) {
	b.set(func(s *Status) bool {
		if s.Scratch != nil && sameUsage(*s.Scratch, u) {
			return false
		}
		s.Scratch = &u
		return true
	})
}

func sameUsage(a, b scratch.Usage) bool {
	return a.TrashBytes == b.TrashBytes && a.TrashEntries == b.TrashEntries &&
		a.TrashUnsized == b.TrashUnsized && a.Deleting == b.Deleting && a.MeasuredAt.Equal(b.MeasuredAt)
}

func (b *StatusBook) set(change func(*Status) bool) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if !change(&b.now) && !b.dirty {
		return
	}
	if err := WriteStatus(b.config, b.now); err != nil {
		b.dirty = true
		if reason := err.Error(); reason != b.warn {
			b.warn = reason
			b.log.Warn("cannot write the status file; the panel will show capabilities as unknown",
				"path", StatusPath(b.config), "err", reason)
		}
		return
	}
	b.dirty, b.warn = false, ""
}
