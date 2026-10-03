//go:build unix

package scratch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"syscall"

	"golang.org/x/sys/unix"
)

// spoolUID 는 spool 의 주인이어야 할 uid 다 — 노드 uid. 시험이 바꿔 끼운다 (남의 것을 만들 수 없다).
var spoolUID = os.Getuid

// Open 은 기동 때 spool 자리를 본다 (business-rules.md 11절 · NFR C8).
//
//	없다                          0700 으로 만든다
//	symlink · 디렉터리 아님 · 남의 것  고치지 않고 거절한다 — 남이 놓은 symlink 를 따라가면 밖에 파일이 생긴다
//	남에게 열린 비트 (0o077)        fchmod 0700 으로 좁힌다
//
// 거절해도 오류가 아니다 — 보존만 못 하고 노드는 뜬다. 까닭은 Refused 에 남는다.
func (s *Store) Open() (SpoolCheck, error) {
	var check SpoolCheck
	fi, err := os.Lstat(s.Dir)
	if errors.Is(err, fs.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(s.Dir), 0o700); err != nil {
			return check, fmt.Errorf("create scratch: %w", err)
		}
		if err := os.Mkdir(s.Dir, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
			return check, fmt.Errorf("create spool: %w", err)
		}
		check.Created = true
		fi, err = os.Lstat(s.Dir)
	}
	if err != nil {
		return check, err
	}
	switch st, _ := fi.Sys().(*syscall.Stat_t); {
	case fi.Mode()&fs.ModeSymlink != 0:
		check.Refused = "symlink"
	case !fi.IsDir():
		check.Refused = "not a directory"
	case st != nil && int(st.Uid) != spoolUID():
		check.Refused = fmt.Sprintf("owned by uid %d", st.Uid)
	case fi.Mode().Perm() != 0o700:
		fd, err := unix.Open(s.Dir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if err != nil {
			return check, &fs.PathError{Op: "open", Path: s.Dir, Err: err}
		}
		defer unix.Close(fd) //nolint:errcheck
		if err := unix.Fchmod(fd, 0o700); err != nil {
			return check, &fs.PathError{Op: "chmod", Path: s.Dir, Err: err}
		}
		check.Narrowed = !check.Created && fi.Mode().Perm()&0o077 != 0
		check.From = fi.Mode().Perm()
	}
	s.refused = check.Refused
	return check, nil
}

// SameFilesystem 은 upper 와 spool 이 같은 filesystem 인가다 (규칙 2절 ②). spool 은 lstat — symlink 면 따라가지
// 않고 예약이 거절한다.
func (s *Store) SameFilesystem() (bool, error) {
	var a, b unix.Stat_t
	if err := unix.Stat(s.Scratch, &a); err != nil {
		return false, err
	}
	if err := unix.Lstat(s.Dir, &b); err != nil {
		return false, err
	}
	return a.Dev == b.Dev, nil
}

// Filesystem 은 scratch 의 statfs 다.
func (s *Store) Filesystem() (Filesystem, error) {
	if s.Stat != nil {
		return s.Stat()
	}
	var st unix.Statfs_t
	if err := unix.Statfs(s.Scratch, &st); err != nil {
		return Filesystem{}, err
	}
	// 칸의 타입이 OS 와 아키텍처마다 다르다 (linux/arm 은 32비트) — 모두 uint64 로 옮겨 담는다
	return Filesystem{Known: true, Free: uint64(st.Bavail) * uint64(st.Bsize),
		FilesTotal: uint64(st.Files), FilesFree: uint64(st.Ffree)}, nil
}

// Kept 는 요약의 보존 총량이다 (규칙 5절 · NFR Design D3). 없거나 못 읽으면 0 — 받아들임이 느슨할 뿐 판정이 막는다.
func (s *Store) Kept() int64 {
	sum, err := readSummary(s.Dir)
	if err != nil {
		return 0
	}
	return sum.Bytes
}

// Reserve 는 세션을 열 때의 예약이다 (NFR Design 답 1) — ID · 항목 폴더 (0700) · 항목 잠금 · reserved 기록. spool
// 은 O_NOFOLLOW 로 연다 — 기동 뒤에 바뀐 자리도 거절한다. spool 잠금은 쥐지 않는다 (규칙 5절).
func (s *Store) Reserve(e Entry) (*Reservation, error) {
	if s.refused != "" {
		return nil, ErrRefused
	}
	dfd, err := unix.Open(s.Dir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open the spool: %w", err)
	}
	defer unix.Close(dfd) //nolint:errcheck
	var id string
	for i := 0; ; i++ {
		if id, err = NewID(); err != nil {
			return nil, err
		}
		err = unix.Mkdirat(dfd, id, 0o700)
		if err == nil {
			break
		}
		if err != unix.EEXIST || i == 7 {
			return nil, fmt.Errorf("create the entry: %w", err)
		}
	}
	r := &Reservation{ID: id, Dir: filepath.Join(s.Dir, id)}
	fail := func(err error) (*Reservation, error) {
		if r.lock != nil {
			_ = r.lock.Close()
		}
		_ = os.Remove(filepath.Join(r.Dir, recordName))
		_ = os.Remove(filepath.Join(r.Dir, SessionLockName))
		_ = os.Remove(r.Dir)
		return nil, err
	}
	if err := unix.Fchmodat(dfd, id, 0o700, 0); err != nil {
		return fail(fmt.Errorf("set the entry mode: %w", err))
	}
	efd, err := unix.Openat(dfd, id, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return fail(fmt.Errorf("open the entry: %w", err))
	}
	defer unix.Close(efd) //nolint:errcheck
	lfd, err := unix.Openat(efd, SessionLockName, unix.O_RDWR|unix.O_CREAT|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0o600)
	if err != nil {
		return fail(fmt.Errorf("create the entry lock: %w", err))
	}
	r.lock = os.NewFile(uintptr(lfd), SessionLockName)
	// 기다리는 잠금이다 — 이 잠금을 시험하는 쪽 (조정 · 조회) 은 LOCK_NB 로 한순간만 쥔다
	if err := unix.Flock(lfd, unix.LOCK_EX); err != nil {
		return fail(fmt.Errorf("hold the entry lock: %w", err))
	}
	e.Schema, e.ID, e.State = recordSchema, id, EntryReserved
	if err := writeRecord(r.Dir, e); err != nil {
		return fail(fmt.Errorf("write the record: %w", err))
	}
	r.entry = e
	return r, nil
}

// Commit 은 확정이다 (규칙 2절 ⑦) — kept 기록 · 잡은 시각 · 만료 시각 (TTL · 규칙 7절) · 항목 잠금을 놓는다.
// e 는 닫을 때 모은 신원까지 담은 기록이다. 실패하면 부르는 쪽이 Abandon 한다.
func (s *Store) Commit(r *Reservation, e Entry) (Entry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := s.now()
	expires := now.Add(s.Policy.TTL)
	e.Schema, e.ID, e.State = recordSchema, r.ID, EntryKept
	e.CapturedAt, e.ExpiresAt = &now, &expires
	if err := writeRecord(r.Dir, e); err != nil {
		return Entry{}, err
	}
	r.entry = e
	r.releaseLocked()
	return e, nil
}

// Abandon 은 항목 폴더째 trash 로 옮긴다 (rename 한 번 · 규칙 2절). 옮긴 upper 가 안에 있어도 된다. 옮기지 못하면
// reserved 로 남아 기동 조정이 거둔다.
func (s *Store) Abandon(r *Reservation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.done {
		return nil
	}
	_, err := s.Trash.Move(r.Dir)
	r.releaseLocked()
	return err
}

// Dispose 는 요구하지 않은 단계의 예약을 버린다 (보고 뒤 · NFR Design 답 1). 빈 예약이면 파일 둘과 폴더를 지우고,
// 무엇이 더 들어 있으면 trash 로 옮긴다. 확정했거나 이미 버렸으면 할 일이 없다.
func (s *Store) Dispose(r *Reservation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.done {
		return nil
	}
	_ = os.Remove(filepath.Join(r.Dir, recordName))
	_ = os.Remove(filepath.Join(r.Dir, SessionLockName))
	err := os.Remove(r.Dir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		_, err = s.Trash.Move(r.Dir)
	} else {
		err = nil
	}
	r.releaseLocked()
	return err
}

// releaseLocked 는 항목 잠금을 놓는다. r.mu 안에서 부른다.
func (r *Reservation) releaseLocked() {
	if r.lock != nil {
		_ = r.lock.Close()
		r.lock = nil
	}
	r.done = true
}

// Report 는 보고의 성패를 기록에 적는다 (규칙 8절). 판정의 적기와 겹치지 않게 spool 잠금을 쥔다. 기록이 없으면
// (그새 만료되었다) 할 일이 없다.
func (s *Store) Report(id, outcome string) error {
	unlock, err := s.lockSpool()
	if err != nil {
		return err
	}
	defer unlock()
	dir := filepath.Join(s.Dir, id)
	e, err := readRecord(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	if e.State != EntryKept && e.State != EntryEvicted {
		return nil
	}
	e.Report = outcome
	return writeRecord(dir, *e)
}

// Settle 은 보고 뒤 판정이다 (규칙 6절 · NFR Design D2). spool 잠금은 기록을 읽을 때와 적을 때만 쥐고 측정은 잠금
// 밖에서 한다 — 적기 전에 기록을 다시 읽어 그새 형제가 퇴출하거나 만료한 항목의 측정값은 버린다.
func (s *Store) Settle(ctx context.Context) (SettleResult, error) {
	res := SettleResult{Failed: map[string]error{}}
	if s.refused != "" {
		return res, ErrRefused
	}
	now := s.now()

	// 1 — 읽기 · 만료 · 측정할 목록
	unlock, err := s.lockSpool()
	if err != nil {
		return res, err
	}
	entries := s.readAll()
	var unmeasured []Entry
	for _, e := range entries {
		switch {
		case e.State != EntryKept && e.State != EntryEvicted:
		case e.ExpiresAt != nil && !now.Before(*e.ExpiresAt):
			if s.moveWhole(e.ID) {
				res.Expired, res.Moved = append(res.Expired, e.ID), true
			}
		case e.State == EntryKept && e.Bytes == nil:
			unmeasured = append(unmeasured, e)
		}
	}
	unlock()
	sort.SliceStable(unmeasured, func(i, j int) bool {
		return capturedAt(unmeasured[i]).Before(capturedAt(unmeasured[j]))
	})

	// 2 — 측정 (잠금 밖)
	sizes := map[string]Size{}
	for _, e := range unmeasured {
		if ctx.Err() != nil || s.Measure == nil {
			break
		}
		size, err := s.Measure(ctx, filepath.Join(s.Dir, e.ID), upperName)
		var launch *LaunchError
		switch {
		case errors.As(err, &launch):
			res.Launch = launch
		case err != nil:
			res.Failed[e.ID] = err
		default:
			sizes[e.ID] = size
		}
		if res.Launch != nil {
			break
		}
	}

	// 3 ~ 7 — 적기 · 퇴출 · 요약
	unlock, err = s.lockSpool()
	if err != nil {
		return res, err
	}
	defer unlock()
	entries = s.readAll()
	measuredAt := s.now()
	for i, e := range entries {
		size, ok := sizes[e.ID]
		if !ok || e.State != EntryKept || e.Bytes != nil {
			continue // 그새 퇴출 · 만료되었거나 다른 판정이 이미 쟀다
		}
		bytes, inodes := size.Bytes, size.Entries
		e.Bytes, e.Inodes, e.MeasuredAt = &bytes, &inodes, &measuredAt
		if err := writeRecord(filepath.Join(s.Dir, e.ID), e); err == nil {
			entries[i] = e
		}
	}
	fsys, err := s.Filesystem()
	if err != nil {
		fsys = Filesystem{}
	}
	plan := s.Policy.PlanSettle(entries, fsys, now)
	for _, id := range plan.Expire {
		if s.moveWhole(id) {
			res.Expired, res.Moved = append(res.Expired, id), true
		}
	}
	evicted := map[string]string{}
	for _, ev := range plan.Evict {
		if _, err := s.Trash.Move(filepath.Join(s.Dir, ev.ID, upperName)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			res.Failed[ev.ID] = err
			continue
		}
		evicted[ev.ID] = ev.Reason
		res.Evicted, res.Moved = append(res.Evicted, ev), true
	}
	gone := map[string]bool{}
	for _, id := range plan.Expire {
		gone[id] = true
	}
	var sum Summary
	for _, e := range entries {
		if gone[e.ID] {
			continue
		}
		if reason, ok := evicted[e.ID]; ok {
			e.State, e.Evicted, e.EvictedAt = EntryEvicted, reason, &now
			_ = writeRecord(filepath.Join(s.Dir, e.ID), e)
			continue
		}
		if e.State != EntryKept {
			continue
		}
		res.Usage.Checkpoints++
		if e.Bytes == nil {
			res.Usage.Unsized++
			continue
		}
		sum.Bytes += *e.Bytes
		if e.Inodes != nil {
			sum.Inodes += *e.Inodes
		}
	}
	sum.At = s.now()
	res.Usage.Bytes, res.Usage.At = sum.Bytes, sum.At
	if err := writeSummary(s.Dir, sum); err != nil {
		return res, fmt.Errorf("write the spool summary: %w", err)
	}
	return res, nil
}

// moveWhole 은 항목 폴더째 trash 로 옮긴다 — 만료 (규칙 7절).
func (s *Store) moveWhole(id string) bool {
	_, err := s.Trash.Move(filepath.Join(s.Dir, id))
	return err == nil
}

// Reconcile 은 기동 조정이다 (규칙 9절). spool 잠금을 쥐고 돈다.
func (s *Store) Reconcile() (ReconcileResult, error) {
	var res ReconcileResult
	if s.refused != "" {
		return res, ErrRefused
	}
	unlock, err := s.lockSpool()
	if err != nil {
		return res, err
	}
	defer unlock()
	ents, err := os.ReadDir(s.Dir)
	if err != nil {
		return res, err
	}
	now := s.now()
	for _, de := range ents {
		f := s.found(de)
		switch PlanReconcile(f, now) {
		case TrashIt:
			if s.moveWhole(f.Name) {
				res.Trashed = append(res.Trashed, f.Name)
			}
		case MarkUnknown:
			e := *f.Record
			e.Report = ReportUnknown
			if writeRecord(filepath.Join(s.Dir, f.Name), e) == nil {
				res.Marked = append(res.Marked, f.Name)
			}
		case WarnUnknown:
			res.Unknown = append(res.Unknown, f.Name)
		}
	}
	return res, nil
}

// found 는 spool 의 이름 하나를 본다 — 항목 잠금은 LOCK_NB 로 한순간만 쥐어 본다.
func (s *Store) found(de os.DirEntry) Found {
	f := Found{Name: de.Name()}
	fi, err := de.Info()
	if err != nil || !fi.IsDir() || fi.Mode()&fs.ModeSymlink != 0 {
		return f
	}
	f.Dir = true
	f.Age = s.now().Sub(fi.ModTime())
	if !ValidID(f.Name) {
		return f
	}
	dir := filepath.Join(s.Dir, f.Name)
	f.LockFile, f.LockFree = probeLock(filepath.Join(dir, SessionLockName))
	if e, err := readRecord(dir); err == nil {
		f.Record = e
	}
	return f
}

// probeLock 은 잠금 파일이 있는가와 지금 아무도 안 쥐었는가다. 쥐어 본 잠금은 곧바로 놓는다.
func probeLock(path string) (exists, free bool) {
	fd, err := unix.Open(path, unix.O_RDWR|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return !errors.Is(err, unix.ENOENT), false
	}
	defer unix.Close(fd) //nolint:errcheck
	return true, unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB) == nil
}

// List 는 조회다 (규칙 13절). 기록만 읽고 spool 잠금을 쥐지 않는다. 항목 잠금이 쥐어진 reserved 는 도는 단계의
// 예약이라 뺀다 (NFR Design 답 1). 오래된 것부터.
func (s *Store) List() ([]Listed, error) {
	ents, err := os.ReadDir(s.Dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var out []Listed
	for _, de := range ents {
		f := s.found(de)
		if !f.Dir || !ValidID(f.Name) {
			continue
		}
		dir := filepath.Join(s.Dir, f.Name)
		l := Listed{Path: filepath.Join(dir, upperName)}
		if f.Record != nil {
			l.Entry = *f.Record
		} else {
			l.Entry = Entry{ID: f.Name, State: EntryReserved}
		}
		if l.State == EntryReserved {
			if f.LockFile && !f.LockFree {
				continue
			}
			l.Incomplete = true
		}
		if l.CapturedAt != nil {
			l.Since = *l.CapturedAt
		} else {
			l.Since = s.now().Add(-f.Age)
		}
		out = append(out, l)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Since.Before(out[j].Since) })
	return out, nil
}

// Lookup 은 조회의 show 다. 없거나 도는 단계의 예약이면 nil 이다.
func (s *Store) Lookup(id string) (*Listed, error) {
	if !ValidID(id) {
		return nil, nil
	}
	all, err := s.List()
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].ID == id {
			return &all[i], nil
		}
	}
	return nil, nil
}

// readAll 은 spool 의 읽을 수 있는 기록 전부다. spool 잠금 안에서 부른다.
func (s *Store) readAll() []Entry {
	ents, err := os.ReadDir(s.Dir)
	if err != nil {
		return nil
	}
	var out []Entry
	for _, de := range ents {
		if !de.IsDir() || !ValidID(de.Name()) {
			continue
		}
		if e, err := readRecord(filepath.Join(s.Dir, de.Name())); err == nil {
			out = append(out, *e)
		}
	}
	return out
}

// lockSpool 은 spool 잠금이다 (<spool>/.lock · 기다린다). 쥐는 것은 판정의 읽기 · 적기와 조정 · Report 다 — 모두 ms 다.
func (s *Store) lockSpool() (func(), error) {
	fd, err := unix.Open(filepath.Join(s.Dir, spoolLockName), unix.O_RDWR|unix.O_CREAT|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open the spool lock: %w", err)
	}
	if err := unix.Flock(fd, unix.LOCK_EX); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("hold the spool lock: %w", err)
	}
	return func() { _ = unix.Close(fd) }, nil
}

// readRecord 는 기록을 읽는다 — O_NOFOLLOW · 보통 파일 · 1 MiB 까지 (NFR C7). 넘거나 비거나 JSON 이 아니면 오류다.
func readRecord(dir string) (*Entry, error) {
	var e Entry
	if err := readJSON(filepath.Join(dir, recordName), &e); err != nil {
		return nil, err
	}
	if e.ID == "" {
		return nil, errors.New("the record has no id")
	}
	return &e, nil
}

func readSummary(spool string) (Summary, error) {
	var sum Summary
	err := readJSON(filepath.Join(spool, summaryName), &sum)
	return sum, err
}

func readJSON(path string, v any) error {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return &fs.PathError{Op: "open", Path: path, Err: err}
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close() //nolint:errcheck
	fi, err := f.Stat()
	if err != nil {
		return err
	}
	if !fi.Mode().IsRegular() || fi.Size() > recordLimit {
		return fmt.Errorf("%s is not a regular file of at most %d bytes", filepath.Base(path), recordLimit)
	}
	b, err := io.ReadAll(io.LimitReader(f, recordLimit+1))
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// writeRecord 는 기록을 쓴다 — 같은 폴더의 임시 파일 (0600) 뒤 rename. fsync 하지 않는다 (NFR P4) — 전원이 나가 기록을
// 잃으면 기동 조정이 주인 없는 미완료로 거둔다.
func writeRecord(dir string, e Entry) error {
	return writeJSON(dir, recordName, e)
}

func writeSummary(spool string, sum Summary) error {
	return writeJSON(spool, summaryName, sum)
}

func writeJSON(dir, name string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, "."+name+"-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, filepath.Join(dir, name)); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
