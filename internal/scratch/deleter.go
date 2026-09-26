package scratch

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Size 는 항목 하나의 크기(블록 수 x 512)와 항목 수(디렉터리 포함)다.
type Size struct {
	Bytes   int64 `json:"bytes"`
	Entries int64 `json:"entries"`
}

// Usage 는 scratch 의 양이다. 늦은 값이다 — 측정한 시각을 함께 싣는다 (Application Design Q3).
// 측정값은 삭제자의 메모리에만 있다. 데몬이 재시작하면 남은 항목은 다시 크기를 모른다.
type Usage struct {
	TrashBytes   int64     `yaml:"trash_bytes" json:"trash_bytes"`     // 측정한 항목들의 합
	TrashEntries int       `yaml:"trash_entries" json:"trash_entries"` // trash 에 남은 항목 수
	TrashUnsized int       `yaml:"trash_unsized" json:"trash_unsized"` // 그 가운데 아직 측정하지 않은 수
	Deleting     bool      `yaml:"deleting" json:"deleting"`
	MeasuredAt   time.Time `yaml:"measured_at" json:"measured_at"`
}

// LaunchError 는 helper 를 못 띄웠다는 것이다 (unshare 가 없다 · uid 매핑이 안 된다 …).
// 항목의 잘못이 아니므로 삭제자는 그 깸을 멈추고 다음 깸까지 기다린다.
type LaunchError struct {
	Err error
}

func (e *LaunchError) Error() string { return e.Err.Error() }
func (e *LaunchError) Unwrap() error { return e.Err }

// LeftError 는 helper 가 다른 filesystem 이라 들어가지 않고 남긴 경로가 있다는 것이다.
type LeftError struct {
	Left []string
}

func (e *LeftError) Error() string {
	return fmt.Sprintf("%d paths reach into another filesystem", len(e.Left))
}

// DefaultEvery 는 trash 에 항목이 남아 있을 때 다시 깨는 기본 간격이다.
const DefaultEvery = 10 * time.Minute

// Deleter 는 배경 삭제자다 (business-rules.md 4절). trash 의 항목을 하나씩 차례로 helper
// 하나로 지운다. 동시에 둘을 안 연다. 도는 단계가 있어도 멈추지 않는다 — helper 가 IO
// 우선순위 idle · CPU 우선순위 19 로 돈다.
//
//	깨는 때   Run 이 시작하면 곧바로 · Kick (결과 보고 뒤) · 항목이 남아 있으면 Every 마다
//	멈추는 때 ctx 가 끝나면.  도는 helper 는 Launch 의 ctx 로 죽는다
type Deleter struct {
	Trash Trash
	// Launch 는 항목 하나에 helper 하나를 연다 (enode 가 채운다 — namespace 를 여는 쪽).
	// measured 는 helper 가 지우기 전에 낸 측정이다. 지우기가 끝나면 돌아온다 — 다 지웠으면
	// nil · 남긴 경로가 있으면 *LeftError · helper 를 못 띄웠으면 *LaunchError.
	Launch func(ctx context.Context, entry string, measured func(Size)) error
	// Changed 는 Usage 가 바뀔 때 불린다 — 상태 파일을 쓰는 자리가 받는다. nil 이면 안 부른다.
	Changed func(Usage)
	Log     *slog.Logger
	// Every 는 trash 에 항목이 남아 있을 때 다시 깨는 간격이다. 0 이면 DefaultEvery.
	Every time.Duration

	once sync.Once
	kick chan struct{}

	mu    sync.Mutex
	names []string
	sizes map[string]Size
	usage Usage

	// launchWarn · listWarn 은 같은 원인을 다시 안 적으려고 둔다
	launchWarn string
	listWarn   string
}

// Kick 은 결과 보고 뒤에 부른다. 막지 않는다 — 이미 깨어 있으면 한 번 더 도는 것으로 합친다.
func (d *Deleter) Kick() {
	select {
	case d.kicks() <- struct{}{}:
	default:
	}
}

func (d *Deleter) kicks() chan struct{} {
	d.once.Do(func() { d.kick = make(chan struct{}, 1) })
	return d.kick
}

// Usage 는 지금 아는 scratch 의 양이다.
func (d *Deleter) Usage() Usage {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.usage
}

// Run 은 ctx 가 끝날 때까지 돈다. 곧바로 한 번 돈다 — 기동 청소가 옮긴 것을 거둔다.
func (d *Deleter) Run(ctx context.Context) {
	every := d.Every
	if every <= 0 {
		every = DefaultEvery
	}
	for {
		remaining := d.pass(ctx)
		if ctx.Err() != nil {
			return
		}
		var wake <-chan time.Time
		var timer *time.Timer
		if remaining {
			timer = time.NewTimer(every)
			wake = timer.C
		}
		select {
		case <-ctx.Done():
		case <-d.kicks():
		case <-wake:
		}
		if timer != nil {
			timer.Stop()
		}
		if ctx.Err() != nil {
			return
		}
	}
}

// pass 는 trash 를 한 번 비운다. 돌려주는 것은 항목이 남았나다.
func (d *Deleter) pass(ctx context.Context) bool {
	names, err := d.Trash.Entries()
	if err != nil {
		d.warnOnce(&d.listWarn, "cannot list trash; trash is not emptied", err)
		return true
	}
	d.listWarn = ""
	d.update(func() { d.setNames(names) }, len(names) > 0)
	if len(names) == 0 {
		return false
	}
	for _, name := range names {
		if ctx.Err() != nil {
			return true
		}
		began := time.Now()
		err := d.Launch(ctx, name, func(s Size) {
			d.update(func() { d.sizes[name] = s }, true)
		})
		if ctx.Err() != nil {
			return true
		}
		var launch *LaunchError
		if errors.As(err, &launch) {
			d.warnOnce(&d.launchWarn, "cannot start the trash helper; trash is not emptied", launch.Err)
			break
		}
		d.launchWarn = ""
		d.report(name, err, time.Since(began))
		if err == nil {
			d.update(func() { d.drop(name) }, true)
		}
	}
	names, err = d.Trash.Entries()
	if err != nil {
		d.warnOnce(&d.listWarn, "cannot list trash; trash is not emptied", err)
		d.update(func() {}, false)
		return true
	}
	d.update(func() { d.setNames(names) }, false)
	return len(names) > 0
}

// report 는 결과마다의 노드 로그다 (business-rules.md 4.3). 삭제는 판정이 아니다 — 지우지
// 못해도 단계의 결과는 안 바뀌고 항목이 trash 에 남는다.
func (d *Deleter) report(name string, err error, took time.Duration) {
	var left *LeftError
	switch {
	case err == nil:
		d.log().Debug("trash entry removed", "name", name, "bytes", d.sizeOf(name).Bytes,
			"took", took.Round(time.Millisecond))
	case errors.As(err, &left):
		d.log().Warn("trash entry left in part; it reaches into another filesystem",
			"name", name, "left", left.Left)
	default:
		d.log().Warn("cannot remove trash entry; it stays in trash", "name", name, "err", err)
	}
}

func (d *Deleter) warnOnce(last *string, msg string, err error) {
	if err.Error() == *last {
		return
	}
	*last = err.Error()
	d.log().Warn(msg, "err", err)
}

// update 는 f 로 목록과 측정을 고친 뒤 Usage 를 다시 세고 Changed 를 부른다. Changed 는
// 잠금 밖에서 부른다 — 상태 파일을 쓰는 쪽이 제 잠금을 쥔다.
func (d *Deleter) update(f func(), deleting bool) {
	d.mu.Lock()
	if d.sizes == nil {
		d.sizes = map[string]Size{}
	}
	f()
	u := Usage{TrashEntries: len(d.names), Deleting: deleting, MeasuredAt: time.Now().UTC()}
	for _, n := range d.names {
		if s, ok := d.sizes[n]; ok {
			u.TrashBytes += s.Bytes
		} else {
			u.TrashUnsized++
		}
	}
	d.usage = u
	d.mu.Unlock()
	if d.Changed != nil {
		d.Changed(u)
	}
}

// setNames 는 trash 의 목록을 새로 받는다. 없어진 이름의 측정은 버린다. 잠금 안에서 부른다.
func (d *Deleter) setNames(names []string) {
	d.names = append(d.names[:0], names...)
	keep := make(map[string]bool, len(names))
	for _, n := range names {
		keep[n] = true
	}
	for n := range d.sizes {
		if !keep[n] {
			delete(d.sizes, n)
		}
	}
}

// drop 은 지운 항목을 목록과 측정에서 뺀다. 잠금 안에서 부른다.
func (d *Deleter) drop(name string) {
	delete(d.sizes, name)
	for i, n := range d.names {
		if n == name {
			d.names = append(d.names[:i], d.names[i+1:]...)
			break
		}
	}
}

func (d *Deleter) sizeOf(name string) Size {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.sizes[name]
}

func (d *Deleter) log() *slog.Logger {
	if d.Log == nil {
		return slog.New(slog.DiscardHandler)
	}
	return d.Log
}
