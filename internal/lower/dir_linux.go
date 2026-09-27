package lower

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// Open 은 노드 데몬이 부른다 (기동 · 못 열었으면 광고 주기마다 다시). 자리 · holders/ (0700) 와
// lower.lock · bake.lock (0600) 을 만들고, lower.json 을 대조한 뒤 판정대로 쓴다 (business-rules.md 1 · 2절).
//
// lowers 의 부모는 만들 때만 0700 이고 보지 않는다. lowers 부터는 한 층씩 만들고 다시 열어 본다 —
// symlink · 주인이 다른 자리는 거절하고, 남에게 열린 비트는 좁힌다 (계획 3.1). 고친 것은 Notes 에 남는다.
func Open(lowers string, root Root, now time.Time) (*Dir, error) {
	d := &Dir{Root: root, Path: filepath.Join(lowers, root.Key.String()), uid: os.Getuid()}
	if err := os.MkdirAll(filepath.Dir(lowers), 0o700); err != nil {
		return nil, openErr(filepath.Dir(lowers), err)
	}
	top, err := ensureDir(unix.AT_FDCWD, lowers, lowers, d.uid, &d.Notes)
	if err != nil {
		return nil, err
	}
	defer func() { _ = unix.Close(top) }()
	key, err := ensureDir(top, root.Key.String(), d.Path, d.uid, &d.Notes)
	if err != nil {
		return nil, err
	}
	defer func() { _ = unix.Close(key) }()
	holders, err := ensureDir(key, holdersDir, filepath.Join(d.Path, holdersDir), d.uid, &d.Notes)
	if err != nil {
		return nil, err
	}
	_ = unix.Close(holders)
	for _, name := range []string{lowerLock, bakeLock} {
		p := filepath.Join(d.Path, name)
		f, narrowed, err := openLock(p, d.uid)
		if err != nil {
			return nil, err
		}
		f.Close()
		if narrowed {
			d.Notes = append(d.Notes, "narrowed loose permissions on "+p+" to 0600")
		}
	}
	if err := d.record(now); err != nil {
		return nil, err
	}
	return d, nil
}

// record 는 lower.json 을 대조하고 판정대로 쓴다 (business-rules.md 2절).
func (d *Dir) record(now time.Time) error {
	rec, v, st, err := d.inspect()
	if err != nil {
		return err
	}
	jsonPath := filepath.Join(d.Path, identityFile)
	fresh := &Identity{Schema: schema, FSID: d.Root.Key.fsidString(), Ino: d.Root.Key.Ino,
		BirthNs: d.Root.BirthNs, Paths: []string{d.Root.Path}, OwnerUID: d.Root.UID, SeenAt: now.UTC()}
	switch v {
	case VerdictNew:
		return writeJSON(d.Path, identityFile, fresh, true, 0)
	case VerdictMatch:
		if slices.Contains(rec.Paths, d.Root.Path) {
			return nil
		}
		rec.Paths = append(rec.Paths, d.Root.Path)
		rec.SeenAt = now.UTC()
		return writeJSON(d.Path, identityFile, rec, true, 0)
	case VerdictReused:
		if err := writeJSON(d.Path, identityFile, fresh, true, 0); err != nil {
			return err
		}
		d.Notes = append(d.Notes, jsonPath+" recorded another directory with the same inode; rewrote it")
		d.clearLastAttempt()
		return nil
	case VerdictForeign:
		return fmt.Errorf("lower: %s records a different directory (birth time differs) and its state is %s",
			jsonPath, st.Phase)
	}
	return fmt.Errorf("lower: %s does not match its key", jsonPath)
}

// clearLastAttempt 는 Reused 에서 옛 디렉터리의 실패한 굽기를 지운다 (계획 4절 ⑬). state.json 을 쓰는 쪽은
// 굽기 잠금의 주인이므로 잠깐 쥐고 쓴다. 못 쥐면 지우지 않는다 — 그 굽기가 building 을 쓰며 상태를 새로 쓴다.
func (d *Dir) clearLastAttempt() {
	st, err := d.ReadState()
	if err != nil || st.LastAttempt == nil {
		return
	}
	b, ok, err := d.TryBake()
	if err != nil || !ok {
		d.Notes = append(d.Notes, "a bake holds "+filepath.Join(d.Path, bakeLock)+
			"; left last_attempt of the old directory in state.json")
		return
	}
	defer func() { _ = b.Release() }()
	st.LastAttempt = nil
	if err := b.WriteState(st); err != nil {
		d.Notes = append(d.Notes, "cannot clear last_attempt of the old directory: "+err.Error())
	}
}

// inspect 는 lower.json 을 읽어 판정한다. Open 과 Check 가 함께 쓴다. state.json 은 btime 이 달라 phase 가
// 필요할 때만 읽는다. 못 읽으면 err 가 있고, 판정은 Broken 이다.
func (d *Dir) inspect() (*Identity, Verdict, State, error) {
	rec, err := d.readIdentity()
	if err != nil {
		return nil, VerdictBroken, State{}, err
	}
	var st State
	if rec != nil && rec.matchesKey(d.Root.Key) && rec.BirthNs != 0 && d.Root.BirthNs != 0 &&
		rec.BirthNs != d.Root.BirthNs {
		if st, err = d.ReadState(); err != nil {
			return rec, VerdictBroken, State{}, err
		}
	}
	return rec, identityVerdict(rec, d.Root, st.Phase), st, nil
}

// readIdentity 는 lower.json 이다. 없으면 nil, nil 이다.
func (d *Dir) readIdentity() (*Identity, error) {
	var rec Identity
	err := readJSON(filepath.Join(d.Path, identityFile), d.uid, &rec)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

// Peek 은 env check 와 smoke 가 부른다. 아무것도 만들지 않고 고치지 않는다 (ADR-073). 자리가 없으면 nil, nil
// 이다. symlink 와 주인이 다른 자리는 오류다. 남에게 열린 비트는 Loose 에 적는다.
func Peek(lowers string, root Root) (*Dir, error) {
	d := &Dir{Root: root, Path: filepath.Join(lowers, root.Key.String()), uid: os.Getuid()}
	top, loose, err := peekDir(unix.AT_FDCWD, lowers, lowers, d.uid)
	if err != nil || top < 0 {
		return nil, err
	}
	defer func() { _ = unix.Close(top) }()
	d.noteLoose(lowers, loose)
	key, loose, err := peekDir(top, root.Key.String(), d.Path, d.uid)
	if err != nil || key < 0 {
		return nil, err
	}
	defer func() { _ = unix.Close(key) }()
	d.noteLoose(d.Path, loose)
	hp := filepath.Join(d.Path, holdersDir)
	holders, loose, err := peekDir(key, holdersDir, hp, d.uid)
	if err != nil {
		return nil, err
	}
	if holders >= 0 {
		_ = unix.Close(holders)
		d.noteLoose(hp, loose)
	}
	for _, name := range []string{lowerLock, bakeLock} {
		p := filepath.Join(d.Path, name)
		fd, err := unix.Open(p, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if errors.Is(err, unix.ENOENT) {
			continue
		}
		if err != nil {
			return nil, &pathError{Path: p, Err: err}
		}
		loose, err := checkFd(fd, p, d.uid, false, false)
		_ = unix.Close(fd)
		if err != nil {
			return nil, err
		}
		d.noteLoose(p, loose)
	}
	return d, nil
}

func (d *Dir) noteLoose(p string, loose bool) {
	if loose {
		d.Loose = append(d.Loose, p)
	}
}

// ReadState 는 누구나 부른다 — 잠금 없이 읽는다 (파일은 rename 으로만 바뀐다). 파일이 없으면 committed 다 —
// 굽기가 한 번도 없었던 lower 다. 못 읽거나 모르는 phase 면 오류다 — 모르는 상태를 committed 로 읽지 않는다.
func (d *Dir) ReadState() (State, error) {
	var st State
	p := filepath.Join(d.Path, stateFile)
	err := readJSON(p, d.uid, &st)
	if errors.Is(err, fs.ErrNotExist) {
		return State{Schema: schema, Phase: PhaseCommitted}, nil
	}
	if err != nil {
		return State{}, err
	}
	if err := checkState(st); err != nil {
		return State{}, &pathError{Path: p, Err: err}
	}
	return st, nil
}

// ReadMetadata 는 <lowerRoot>/.enode-metadata.json 이다. 없으면 nil, nil 이다. O_NOFOLLOW · 보통 파일 ·
// 1 MiB 까지 · schema 1 (business-rules.md 11절). lower 는 같은 사용자면 누구나 쓸 수 있는 자리라 주인은 안 본다.
func ReadMetadata(lowerRoot string) (*Metadata, error) {
	var m Metadata
	p := filepath.Join(lowerRoot, metadataFile)
	err := readJSON(p, -1, &m)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := checkMetadata(&m); err != nil {
		return nil, &pathError{Path: p, Err: err}
	}
	return &m, nil
}

// WriteMetadata 는 합치기의 마지막 동작이다 (bake 유닛이 lower 배타를 쥐고 부른다). lower 뿌리의 임시 파일에
// 쓰고 rename 한다 — 그 자리가 symlink 여도 따라가지 않고 링크를 바꾼다. 옛 쓰기가 남긴 임시 파일을 먼저
// 지운다: 배타 아래라 경쟁이 없고, 합친 lower 에 남으면 다음 빌드가 그 파일을 본다. 권한은 0644 다 —
// lower 의 다른 파일과 같다.
func WriteMetadata(lowerRoot string, m Metadata) error {
	if m.Schema == 0 {
		m.Schema = schema
	}
	if err := checkMetadata(&m); err != nil {
		return &pathError{Path: filepath.Join(lowerRoot, metadataFile), Err: err}
	}
	ents, err := os.ReadDir(lowerRoot)
	if err != nil {
		return openErr(lowerRoot, err)
	}
	prefix := strings.TrimSuffix(tempPattern(metadataFile), "*")
	for _, e := range ents {
		if strings.HasPrefix(e.Name(), prefix) {
			if err := os.Remove(filepath.Join(lowerRoot, e.Name())); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return openErr(filepath.Join(lowerRoot, e.Name()), err)
			}
		}
	}
	return writeJSON(lowerRoot, metadataFile, m, true, 0o644)
}
