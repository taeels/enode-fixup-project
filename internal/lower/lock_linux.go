package lower

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// pollEvery 는 배타를 기다리는 동안 LOCK_EX|LOCK_NB 를 다시 거는 간격이다 (business-rules.md 4절). 막힌 채
// 기다리는 flock 은 ctx 로 끊을 수 없다. 1초는 하루치 합치기(1 ~ 2초)에 견주어 작다. 시험만 줄인다.
var pollEvery = time.Second

// flockNB 는 막히지 않는 flock 이다. 막혔으면 false 와 nil 이다.
func flockNB(f *os.File, how int) (bool, error) {
	err := unix.Flock(int(f.Fd()), how|unix.LOCK_NB)
	if errors.Is(err, unix.EWOULDBLOCK) {
		return false, nil
	}
	if err != nil {
		return false, &pathError{Path: f.Name(), Err: err}
	}
	return true, nil
}

func (d *Dir) lockPath(name string) string { return filepath.Join(d.Path, name) }
func (d *Dir) holderPath(node, ext string) string {
	return filepath.Join(d.Path, holdersDir, node+ext)
}

// TryShared 는 lower.lock 을 공유로 잡고 쥔 사람 기록을 남긴다. 막히지 않는다 — 배타가 쥐어져 있으면 false 다.
//
// 차례는 lower.lock 다음 기록이다 (business-rules.md 7절) — 살아 있는 기록이 있으면 그 노드는 lower.lock 을
// 쥐고 있다. node_id 가 파일 이름이 될 수 없으면 기록 없이 공유만 쥔다 (Recorded 가 거짓).
func (d *Dir) TryShared(h Holder) (*Shared, bool, error) {
	f, _, err := openLock(d.lockPath(lowerLock), d.uid)
	if err != nil {
		return nil, false, err
	}
	ok, err := flockNB(f, unix.LOCK_SH)
	if err != nil || !ok {
		f.Close()
		return nil, false, err
	}
	s := &Shared{d: d, lock: f}
	if !validNode(h.Node) {
		s.h = h
		return s, true, nil
	}
	rec, _, err := openLock(d.holderPath(h.Node, ".lock"), d.uid)
	if err != nil {
		_ = s.Release()
		return nil, false, err
	}
	ok, err = flockNB(rec, unix.LOCK_EX)
	if err != nil {
		rec.Close()
		_ = s.Release()
		return nil, false, err
	}
	if !ok {
		// 같은 node_id 의 기록 잠금을 다른 프로세스가 쥐고 있다 — 기록은 정보이고 증거는 lower.lock 이다.
		// 공유는 그대로 쥐고 기록 없이 간다.
		rec.Close()
		s.h = h
		return s, true, nil
	}
	s.rec = rec
	if err := s.write(h); err != nil {
		_ = s.Release()
		return nil, false, err
	}
	return s, true, nil
}

// write 는 기록 파일을 임시 파일 + rename 으로 쓴다. fsync 하지 않는다 — 살아 있음은 잠금이 말한다 (계획 4절 ②).
func (s *Shared) write(h Holder) error {
	h.PID = os.Getpid()
	if err := writeJSON(filepath.Join(s.d.Path, holdersDir), h.Node+".json", h, false, 0); err != nil {
		return err
	}
	s.h = h
	return nil
}

// Update 는 기록을 바꾼다 — 역할 · Run · acks 가 바뀔 때. 바뀐 칸이 없으면 쓰지 않는다. node_id 는 못 바꾼다.
func (s *Shared) Update(h Holder) error {
	h.Node, h.PID = s.h.Node, s.h.PID
	if s.rec == nil {
		s.h = h
		return nil
	}
	if h == s.h {
		return nil
	}
	return s.write(h)
}

// Release 는 기록 다음 lower.lock 을 놓는다. 기록 파일은 지우지 않는다 — 잠금이 풀리면 죽은 기록이다.
// nil 이나 두 번 불러도 된다.
func (s *Shared) Release() error {
	if s == nil {
		return nil
	}
	var errs []error
	if s.rec != nil {
		errs = append(errs, s.rec.Close())
		s.rec = nil
	}
	if s.lock != nil {
		errs = append(errs, s.lock.Close())
		s.lock = nil
	}
	return errors.Join(errs...)
}

// WaitShared 는 smoke 가 쓴다 (답 5). 기록을 남기지 않는다 — 몇 초 쥐고 놓고, 같은 node_id 의 데몬이 이미
// 그 이름의 잠금을 쥐고 있을 수 있다. 배타가 쥐어져 있으면 every 마다 다시 보며 ctx 까지 기다리고, 다시 볼
// 때마다 notice 에 state.json 을 넘긴다 (누가 합치나를 한 줄로 쓰게). lower.lock 이 없으면 잠그지 않고
// nil, nil 이다 — 잠금이 없는 파일은 누구도 쥘 수 없으므로 그 순간 합치기도 없다 (계획 4절 ⑰).
func (d *Dir) WaitShared(ctx context.Context, every time.Duration, notice func(State)) (*Shared, error) {
	f, err := openLockRO(d.lockPath(lowerLock), d.uid)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	for {
		ok, err := flockNB(f, unix.LOCK_SH)
		if err != nil {
			f.Close()
			return nil, err
		}
		if ok {
			return &Shared{d: d, lock: f}, nil
		}
		if notice != nil {
			st, _ := d.ReadState()
			notice(st)
		}
		if err := sleep(ctx, every); err != nil {
			f.Close()
			return nil, err
		}
	}
}

// Exclusive 는 lower.lock 의 배타를 ctx 가 끝날 때까지 기다린다. pollEvery 마다 LOCK_EX|LOCK_NB 를 다시 걸고,
// 막혀 있는 동안 처음과 every 마다 watch 에 살아 있는 기록을 넘긴다. 마감이면 ctx.Err() 다.
func (d *Dir) Exclusive(ctx context.Context, every time.Duration, watch func(Waiting)) (*Exclusive, error) {
	f, _, err := openLock(d.lockPath(lowerLock), d.uid)
	if err != nil {
		return nil, err
	}
	var watched time.Time
	for {
		ok, err := flockNB(f, unix.LOCK_EX)
		if err != nil {
			f.Close()
			return nil, err
		}
		if ok {
			return &Exclusive{lock: f}, nil
		}
		if watch != nil && (watched.IsZero() || time.Since(watched) >= every) {
			watched = time.Now()
			hs, _ := d.Holders()
			watch(Waiting{Holders: hs, Unnamed: len(hs) == 0})
		}
		if err := sleep(ctx, pollEvery); err != nil {
			f.Close()
			return nil, err
		}
	}
}

// Release 는 배타를 놓는다. nil 이나 두 번 불러도 된다.
func (e *Exclusive) Release() error {
	if e == nil || e.lock == nil {
		return nil
	}
	err := e.lock.Close()
	e.lock = nil
	return err
}

// TryBake 는 bake.lock 을 배타로 잡는다. 주인이 살아 있으면 false 다 (결정 3-12).
func (d *Dir) TryBake() (*Bake, bool, error) {
	f, _, err := openLock(d.lockPath(bakeLock), d.uid)
	if err != nil {
		return nil, false, err
	}
	ok, err := flockNB(f, unix.LOCK_EX)
	if err != nil || !ok {
		f.Close()
		return nil, false, err
	}
	return &Bake{d: d, lock: f}, true, nil
}

// WriteState 는 state.json 을 쓴다 — 굽기 잠금의 주인만 부른다. 임시 파일 · fsync · rename · 자리 디렉터리
// fsync 차례다 (business-rules.md 3절). merging 은 첫 합치기 동작 전에 디스크에 있어야 한다.
func (b *Bake) WriteState(s State) error {
	if s.Schema == 0 {
		s.Schema = schema
	}
	if err := checkState(s); err != nil {
		return &pathError{Path: filepath.Join(b.d.Path, stateFile), Err: err}
	}
	return writeJSON(b.d.Path, stateFile, s, true, 0)
}

// Release 는 굽기 잠금을 놓는다. nil 이나 두 번 불러도 된다.
func (b *Bake) Release() error {
	if b == nil || b.lock == nil {
		return nil
	}
	err := b.lock.Close()
	b.lock = nil
	return err
}

// Holders 는 살아 있는 기록만 돌려준다 (business-rules.md 7절). .json 마다 짝 .lock 에 LOCK_SH|LOCK_NB 를 걸어
// 막히면 살아 있는 기록, 잡히면 죽은 기록이라 건너뛴다. 건 잠금은 곧바로 놓는다. 지우지 않는다 — 같은 노드가
// 다시 뜨면 덮어쓰고, 지우면 막 잡은 쪽과 경쟁한다. 점으로 시작하는 이름(쓰는 중인 임시 파일)은 건너뛴다.
func (d *Dir) Holders() ([]Holder, error) {
	dir := filepath.Join(d.Path, holdersDir)
	ents, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, openErr(dir, err)
	}
	var out []Holder
	for _, e := range ents {
		name := e.Name()
		node, ok := strings.CutSuffix(name, ".json")
		if !ok || strings.HasPrefix(name, ".") {
			continue
		}
		if !d.alive(node) {
			continue
		}
		var h Holder
		if err := readJSON(filepath.Join(dir, name), d.uid, &h); err != nil {
			continue
		}
		out = append(out, h)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Node < out[j].Node })
	return out, nil
}

// alive 는 기록의 짝 잠금이 쥐어져 있는가다.
func (d *Dir) alive(node string) bool {
	f, err := openLockRO(d.holderPath(node, ".lock"), d.uid)
	if err != nil {
		return false
	}
	defer f.Close()
	ok, err := flockNB(f, unix.LOCK_SH)
	return err == nil && !ok
}

// sleep 은 ctx 를 보며 d 만큼 기다린다.
func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
