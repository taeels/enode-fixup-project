//go:build !linux

package lower

import (
	"context"
	"time"
)

// linux 밖 — 같은 겉면이 ErrUnsupported 다. 굽기와 runc-overlay 는 linux 만 하므로 부르는 쪽이 오기 전에
// 이미 막혀 있다 (runc-overlay 런타임을 못 짓는다).

// ReadRoot 는 linux 밖에서 지원하지 않는다.
func ReadRoot(string) (Root, error) { return Root{}, ErrUnsupported }

// Open 은 linux 밖에서 지원하지 않는다.
func Open(string, Root, time.Time) (*Dir, error) { return nil, ErrUnsupported }

// Peek 은 linux 밖에서 지원하지 않는다.
func Peek(string, Root) (*Dir, error) { return nil, ErrUnsupported }

// ReadState 는 linux 밖에서 지원하지 않는다.
func (d *Dir) ReadState() (State, error) { return State{}, ErrUnsupported }

// TryShared 는 linux 밖에서 지원하지 않는다.
func (d *Dir) TryShared(Holder) (*Shared, bool, error) { return nil, false, ErrUnsupported }

// WaitShared 는 linux 밖에서 지원하지 않는다.
func (d *Dir) WaitShared(context.Context, time.Duration, func(State)) (*Shared, error) {
	return nil, ErrUnsupported
}

// Exclusive 는 linux 밖에서 지원하지 않는다.
func (d *Dir) Exclusive(context.Context, time.Duration, func(Waiting)) (*Exclusive, error) {
	return nil, ErrUnsupported
}

// TryBake 는 linux 밖에서 지원하지 않는다.
func (d *Dir) TryBake() (*Bake, bool, error) { return nil, false, ErrUnsupported }

// Holders 는 linux 밖에서 지원하지 않는다.
func (d *Dir) Holders() ([]Holder, error) { return nil, ErrUnsupported }

// Update 는 linux 밖에서 지원하지 않는다.
func (s *Shared) Update(Holder) error { return ErrUnsupported }

// Release 는 쥔 것이 없으므로 할 일이 없다.
func (s *Shared) Release() error { return nil }

// Release 는 쥔 것이 없으므로 할 일이 없다.
func (e *Exclusive) Release() error { return nil }

// WriteState 는 linux 밖에서 지원하지 않는다.
func (b *Bake) WriteState(State) error { return ErrUnsupported }

// Release 는 쥔 것이 없으므로 할 일이 없다.
func (b *Bake) Release() error { return nil }

// ReadMetadata 는 linux 밖에서 지원하지 않는다.
func ReadMetadata(string) (*Metadata, error) { return nil, ErrUnsupported }

// WriteMetadata 는 linux 밖에서 지원하지 않는다.
func WriteMetadata(string, Metadata) error { return ErrUnsupported }

// ForeignMounts 는 linux 밖에서 지원하지 않는다.
func ForeignMounts(Root) (MountScan, error) { return MountScan{}, ErrUnsupported }

// Check 는 linux 밖에서 낼 것이 없다 — runc-overlay 점검은 host.os 가 먼저 unsupported 로 막는다.
func Check(string, string, string, int) []Finding { return nil }
