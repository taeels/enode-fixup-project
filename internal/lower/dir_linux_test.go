package lower

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func readRecord(t *testing.T, d *Dir) Identity {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(d.Path, identityFile))
	if err != nil {
		t.Fatal(err)
	}
	var rec Identity
	if err := json.Unmarshal(b, &rec); err != nil {
		t.Fatal(err)
	}
	return rec
}

func writeFile(t *testing.T, p, body string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// Open 은 자리를 만들고 lower.json 을 적는다. 다른 경로로 다시 열면 paths 에 더하고, 같은 경로면 다시 쓰지 않는다.
func TestOpenRecordsAndAddsPaths(t *testing.T) {
	f := newFixture(t)
	at := time.Date(2026, 9, 27, 1, 0, 0, 0, time.UTC)
	d, err := Open(f.lowers, f.root, at)
	if err != nil {
		t.Fatal(err)
	}
	if d.Path != filepath.Join(f.lowers, f.root.Key.String()) || len(d.Notes) != 0 {
		t.Fatalf("dir = %+v", d)
	}
	rec := readRecord(t, d)
	want := Identity{Schema: 1, FSID: f.root.Key.fsidString(), Ino: f.root.Key.Ino, BirthNs: f.root.BirthNs,
		Paths: []string{f.root.Path}, OwnerUID: os.Getuid(), SeenAt: at}
	if rec.Schema != want.Schema || rec.FSID != want.FSID || rec.Ino != want.Ino || rec.BirthNs != want.BirthNs ||
		!slices.Equal(rec.Paths, want.Paths) || rec.OwnerUID != want.OwnerUID || !rec.SeenAt.Equal(at) {
		t.Fatalf("lower.json = %+v, want %+v", rec, want)
	}
	for _, name := range []string{lowerLock, bakeLock, holdersDir} {
		if _, err := os.Lstat(filepath.Join(d.Path, name)); err != nil {
			t.Errorf("Open did not make %s: %v", name, err)
		}
	}

	// 같은 경로로 다시 — 쓰지 않는다
	if _, err := Open(f.lowers, f.root, at.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if rec := readRecord(t, d); !rec.SeenAt.Equal(at) {
		t.Errorf("Open rewrote an unchanged record: %v", rec.SeenAt)
	}
	// bind 별칭 — 같은 키 · 다른 경로
	alias := f.root
	alias.Path = "/work"
	if _, err := Open(f.lowers, alias, at.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if rec := readRecord(t, d); !slices.Equal(rec.Paths, []string{f.root.Path, "/work"}) || !rec.SeenAt.Equal(at.Add(2*time.Hour)) {
		t.Errorf("paths = %v seen %v", rec.Paths, rec.SeenAt)
	}
}

// Peek 은 아무것도 만들지 않는다. 자리가 없으면 nil, nil 이다.
func TestPeekCreatesNothing(t *testing.T) {
	f := newFixture(t)
	d, err := Peek(f.lowers, f.root)
	if d != nil || err != nil {
		t.Fatalf("Peek on nothing = %v, %v", d, err)
	}
	if _, err := os.Lstat(filepath.Join(f.dir, "state")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Peek made something: %v", err)
	}
	if err := os.MkdirAll(f.lowers, 0o700); err != nil {
		t.Fatal(err)
	}
	if d, err := Peek(f.lowers, f.root); d != nil || err != nil {
		t.Fatalf("Peek without the key directory = %v, %v", d, err)
	}
	if ents, _ := os.ReadDir(f.lowers); len(ents) != 0 {
		t.Fatalf("Peek made %v", ents)
	}
	f.open(t)
	d, err = Peek(f.lowers, f.root)
	if err != nil || d == nil || d.Path != filepath.Join(f.lowers, f.root.Key.String()) || len(d.Loose) != 0 {
		t.Fatalf("Peek after Open = %+v, %v", d, err)
	}
	// holders 와 잠금 파일이 없어도 자리는 있다 — 옛 판이 만든 자리 · 데몬이 아직 안 연 자리
	if err := os.Remove(filepath.Join(d.Path, holdersDir)); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(d.Path, lowerLock)); err != nil {
		t.Fatal(err)
	}
	if d, err := Peek(f.lowers, f.root); err != nil || d == nil {
		t.Fatalf("Peek without holders and lower.lock = %v, %v", d, err)
	}
}

// 판정 다섯의 동작 — Broken 과 Foreign 은 오류이고 자리를 열지 않는다.
func TestOpenVerdicts(t *testing.T) {
	f := newFixture(t)
	d := f.open(t)
	jsonPath := filepath.Join(d.Path, identityFile)
	rec := readRecord(t, d)
	write := func(edit func(*Identity)) {
		t.Helper()
		r := rec
		edit(&r)
		b, _ := json.Marshal(r)
		writeFile(t, jsonPath, string(b))
	}

	write(func(r *Identity) { r.Ino++ })
	if _, err := Open(f.lowers, f.root, time.Now()); err == nil || err.Error() != "lower: "+jsonPath+" does not match its key" {
		t.Errorf("another ino = %v", err)
	}
	writeFile(t, jsonPath, "{not json")
	if _, err := Open(f.lowers, f.root, time.Now()); err == nil || !strings.Contains(err.Error(), jsonPath) {
		t.Errorf("broken JSON = %v", err)
	}

	// btime 이 다르고 committed — 새로 쓴다 (Reused). last_attempt 를 지운다.
	b, ok, err := d.TryBake()
	if err != nil || !ok {
		t.Fatal(err)
	}
	if err := b.WriteState(State{Phase: PhaseCommitted, Since: time.Now(), LastAttempt: &LastAttempt{Run: "R-old"}}); err != nil {
		t.Fatal(err)
	}
	_ = b.Release()
	write(func(r *Identity) { r.BirthNs = 1 })
	d2, err := Open(f.lowers, f.root, time.Now())
	if err != nil || len(d2.Notes) != 1 || !strings.Contains(d2.Notes[0], "rewrote it") {
		t.Fatalf("Reused = %+v, %v", d2, err)
	}
	if st, _ := d.ReadState(); st.LastAttempt != nil {
		t.Errorf("last_attempt is left: %+v", st)
	}
	if got := readRecord(t, d); got.BirthNs != f.root.BirthNs {
		t.Errorf("lower.json was not rewritten: %+v", got)
	}

	// 굽기 잠금을 남이 쥐고 있으면 last_attempt 를 지우지 않는다 (계획 4절 ⑬)
	b, _, _ = d.TryBake()
	if err := b.WriteState(State{Phase: PhaseCommitted, Since: time.Now(), LastAttempt: &LastAttempt{Run: "R-old"}}); err != nil {
		t.Fatal(err)
	}
	write(func(r *Identity) { r.BirthNs = 1 })
	d3, err := Open(f.lowers, f.root, time.Now())
	_ = b.Release()
	if err != nil || len(d3.Notes) != 2 || !strings.Contains(d3.Notes[1], "a bake holds") {
		t.Fatalf("Reused under a held bake lock = %+v, %v", d3, err)
	}
	if st, _ := d.ReadState(); st.LastAttempt == nil {
		t.Error("last_attempt was cleared without the bake lock")
	}

	// btime 이 다르고 pending — Foreign
	b, _, _ = d.TryBake()
	if err := b.WriteState(State{Phase: PhasePending, Owner: &Owner{Run: "R-2"}, Since: time.Now()}); err != nil {
		t.Fatal(err)
	}
	_ = b.Release()
	write(func(r *Identity) { r.BirthNs = 1 })
	want := "lower: " + jsonPath + " records a different directory (birth time differs) and its state is pending"
	if _, err := Open(f.lowers, f.root, time.Now()); err == nil || err.Error() != want {
		t.Errorf("Foreign = %v", err)
	}
	// state.json 을 못 읽으면 판정을 못 한다
	writeFile(t, filepath.Join(d.Path, stateFile), "{")
	if _, err := Open(f.lowers, f.root, time.Now()); err == nil || !strings.Contains(err.Error(), stateFile) {
		t.Errorf("unreadable state = %v", err)
	}
}

// state.json — 없으면 committed · 쓴 것을 읽는다 · 깨졌거나 모르는 phase 면 오류.
func TestState(t *testing.T) {
	f := newFixture(t)
	d := f.open(t)
	st, err := d.ReadState()
	if err != nil || st.Phase != PhaseCommitted || st.Schema != 1 {
		t.Fatalf("no state.json = %+v, %v", st, err)
	}
	b, ok, err := d.TryBake()
	if err != nil || !ok {
		t.Fatal(err)
	}
	defer func() { _ = b.Release() }()
	since := time.Date(2026, 9, 27, 2, 0, 0, 0, time.UTC)
	want := State{Schema: 1, Phase: PhasePending, Owner: &Owner{Run: "R-1", Step: 2, Node: "n", Instance: "i"},
		PendingUpper: "/s/pending", Since: since,
		LastAttempt: &LastAttempt{Run: "R-0", At: since, Reason: "x", Builds: []BuildRecord{{Name: "a", ExitCode: 2}}}}
	if err := b.WriteState(want); err != nil {
		t.Fatal(err)
	}
	got, err := d.ReadState()
	if err != nil || got.Phase != want.Phase || *got.Owner != *want.Owner || got.PendingUpper != want.PendingUpper ||
		!got.Since.Equal(since) || got.LastAttempt.Builds[0].ExitCode != 2 {
		t.Fatalf("ReadState = %+v, %v", got, err)
	}
	if fi, err := os.Stat(filepath.Join(d.Path, stateFile)); err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("state.json mode = %v, %v", fi.Mode(), err)
	}
	if err := b.WriteState(State{Phase: "done"}); err == nil || !strings.Contains(err.Error(), `unknown phase "done"`) {
		t.Errorf("WriteState of an unknown phase = %v", err)
	}
	for body, want := range map[string]string{
		"{":                                "unexpected end of JSON input",
		`{"schema":1,"phase":"merged"}`:    `unknown phase "merged"`,
		`{"schema":2,"phase":"committed"}`: "schema 2 is not 1",
	} {
		writeFile(t, filepath.Join(d.Path, stateFile), body)
		if _, err := d.ReadState(); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("ReadState of %s = %v, want %s", body, err, want)
		}
	}
	// 임시 파일은 점으로 시작하고 남지 않는다
	ents, _ := os.ReadDir(d.Path)
	for _, e := range ents {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("a temporary file is left: %s", e.Name())
		}
	}
}

// metadata — 없으면 nil · 쓴 것을 읽는다 · 상한 · symlink · schema · 옛 임시 파일.
func TestMetadata(t *testing.T) {
	f := newFixture(t)
	m, err := ReadMetadata(f.lower)
	if m != nil || err != nil {
		t.Fatalf("no metadata = %v, %v", m, err)
	}
	ir := "your-ir-tag"
	merged := time.Date(2026, 9, 27, 3, 0, 0, 0, time.UTC)
	stale := filepath.Join(f.lower, ".enode-metadata.json.tmp-123")
	writeFile(t, stale, "half")
	in := Metadata{Source: Source{URL: "u", IR: &ir, Pinned: &Pinned{File: "m.xml", SHA256: "abc"}},
		Builds: []BuildRecord{{Name: "config-a", ExitCode: 0}}, Bake: BakeRecord{Run: "R-1", MergedAt: merged, Resumed: true}}
	if err := WriteMetadata(f.lower, in); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(stale); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("an old temporary file is left: %v", err)
	}
	fi, err := os.Stat(filepath.Join(f.lower, metadataFile))
	if err != nil || fi.Mode().Perm() != 0o644 {
		t.Errorf("metadata mode = %v, %v", fi, err)
	}
	m, err = ReadMetadata(f.lower)
	if err != nil || m.Schema != 1 || *m.Source.IR != ir || m.Source.Pinned.SHA256 != "abc" || m.Builds[0].Name != "config-a" ||
		m.Marker() != (Marker{Run: "R-1", MergedAt: merged}) || !m.Bake.Resumed {
		t.Fatalf("ReadMetadata = %+v, %v", m, err)
	}
	if err := WriteMetadata(f.lower, Metadata{Schema: 3}); err == nil || !strings.Contains(err.Error(), "schema 3 is not 1") {
		t.Errorf("WriteMetadata schema 3 = %v", err)
	}
	if err := WriteMetadata(filepath.Join(f.dir, "nope"), in); err == nil {
		t.Error("WriteMetadata into a missing lower succeeded")
	}

	p := filepath.Join(f.lower, metadataFile)
	for body, want := range map[string]string{
		`{"schema":2}`: "schema 2 is not 1",
		"[":            "unexpected end of JSON input",
	} {
		writeFile(t, p, body)
		if _, err := ReadMetadata(f.lower); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("ReadMetadata of %s = %v", body, err)
		}
	}
	writeFile(t, p, `{"schema":1,"environment":"`+strings.Repeat("x", maxJSON)+`"}`)
	if _, err := ReadMetadata(f.lower); err == nil || !strings.Contains(err.Error(), "larger than 1048576 bytes") {
		t.Errorf("ReadMetadata of a large file = %v", err)
	}
	// symlink 는 따라가지 않는다 — 읽기는 거절하고, 쓰기는 링크를 파일로 바꾼다 (가리키던 것은 그대로)
	outside := filepath.Join(f.dir, "outside.json")
	writeFile(t, outside, `{"schema":1}`)
	_ = os.Remove(p)
	if err := os.Symlink(outside, p); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadMetadata(f.lower); err == nil || !strings.Contains(err.Error(), "too many levels of symbolic links") {
		t.Errorf("ReadMetadata through a symlink = %v", err)
	}
	if err := WriteMetadata(f.lower, in); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Lstat(p); err != nil || !fi.Mode().IsRegular() {
		t.Errorf("the symlink was not replaced: %v, %v", fi, err)
	}
	if b, _ := os.ReadFile(outside); string(b) != `{"schema":1}` {
		t.Errorf("the symlink target changed: %s", b)
	}
	// 디렉터리는 보통 파일이 아니다
	_ = os.Remove(p)
	if err := os.Mkdir(p, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadMetadata(f.lower); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Errorf("ReadMetadata of a directory = %v", err)
	}
}
