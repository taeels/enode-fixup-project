package enode

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/taeels/enode/internal/scratch"
)

// 대기 자리와 초안 파일 (bake 유닛 · FD 엔티티 4절).
//
//	<scratch>/pending/<lower 키>/<이름>/     MkdirTemp (0700).  building 을 쓰기 전에 만든다
//	  upper/                                세션의 upper 를 rename 한 번으로 (Keep.Upper)
//	  bake.json                             초안 — metadata 에서 bake 칸만 빠진 것과 그 굽기의 주인
//
// 모든 플랫폼에서 빌드되는 방법만 쓴다 — syscall.O_NOFOLLOW 가 windows 에 없다 (계획 4.1 7번). 파일 주인은
// fileOwner 가 읽는다 (mergehelper_linux.go · linux 밖에서는 모른다고 돌려주고 읽기가 거절한다). Baker 는 linux
// 에서만 만들어지므로 이 파일의 함수는 실제로는 linux 에서만 불린다.

const (
	// pendingDirName 은 scratch 안의 대기 자리 이름이다. enode-runc- 로 시작하지 않으므로 기동 청소가 건드리지
	// 않는다 (trash_linux.go 의 runcSessionPrefix).
	pendingDirName = "pending"
	// draftName 은 초안의 파일 이름이다.
	draftName = "bake.json"
	// draftMax 는 초안을 읽는 상한이다.
	draftMax = 1 << 20
	// pinnedFile 은 repo 모양의 고정 manifest 이름이다 — 워크스페이스 뿌리에 남아 lower 로 합쳐진다.
	pinnedFile = ".enode-manifest.xml"
	// pinnedMax 는 고정 manifest 의 sha256 을 읽는 상한이다.
	pinnedMax = 16 << 20
)

// makePending 은 대기 자리 하나를 만든다 — <scratch>/pending/ 과 <키> 는 0700 의 MkdirAll, 그 아래 <이름> 은
// MkdirTemp (0700). 이름에 run_id 를 쓰지 않는다 — 계약이 적는 아무 글자다. 경로는 state.json 이 든다.
func makePending(scratchDir, key string) (string, error) {
	parent := filepath.Join(scratchDir, pendingDirName, key)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return "", fmt.Errorf("cannot create the pending upper directory: %w", err)
	}
	dir, err := os.MkdirTemp(parent, "bake-")
	if err != nil {
		return "", fmt.Errorf("cannot create the pending upper directory: %w", err)
	}
	return dir, nil
}

// pendingScratch 는 대기 자리 <이름> 의 scratch 다 — 세 단계 위 (이름 -> 키 -> pending -> scratch).
func pendingScratch(dir string) string {
	return filepath.Dir(filepath.Dir(filepath.Dir(dir)))
}

// discardPending 은 대기 자리 <이름> 하나를 그 자리가 있는 scratch 의 trash 로 옮긴다 — rename 한 번. 다른
// 마운트의 trash 로는 rename 이 EXDEV 다. 자리가 이미 없으면 옮겨진 것으로 본다 (결정 46) — 몸통이 옮긴 뒤
// committed 를 못 썼거나, 옮기기와 committed 쓰기 사이에 죽었다.
func discardPending(dir string) error {
	if _, err := os.Lstat(dir); errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if _, err := scratch.TrashIn(pendingScratch(dir)).Move(dir); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// sweepPending 은 자기 scratch 의 pending/<키>/ 아래에서 keep 밖의 모든 항목을 자기 trash 로 옮긴다. 굽기 잠금을
// 쥔 쪽만 부른다 — 그 lower 의 굽기가 이 프로세스 밖에서 돌지 않으므로 남은 것은 모두 버려진 자리다
// (business-rules.md 12.2). 옮긴 이름과 첫 오류를 돌려준다 — 오류가 나도 나머지를 계속 옮긴다.
func sweepPending(scratchDir, key, keep string) ([]string, error) {
	parent := filepath.Join(scratchDir, pendingDirName, key)
	ents, err := os.ReadDir(parent)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var moved []string
	var first error
	for _, e := range ents {
		p := filepath.Join(parent, e.Name())
		if p == keep {
			continue
		}
		if err := discardPending(p); err != nil {
			if first == nil {
				first = err
			}
			continue
		}
		moved = append(moved, p)
	}
	return moved, first
}

// writeDraft 는 초안을 쓴다 — 같은 디렉터리의 CreateTemp (0600) · fsync · rename · 디렉터리 fsync. pending 보다
// 먼저 디스크에 있다 — pending 이 보이면 초안이 있다.
func writeDraft(dir string, d Draft) error {
	fail := func(err error) error { return fmt.Errorf("cannot write the bake draft: %w", err) }
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return fail(err)
	}
	f, err := os.CreateTemp(dir, "."+draftName+".tmp-*")
	if err != nil {
		return fail(err)
	}
	tmp := f.Name()
	if _, err := f.Write(append(b, '\n')); err != nil {
		f.Close()
		_ = os.Remove(tmp)
		return fail(err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		_ = os.Remove(tmp)
		return fail(err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return fail(err)
	}
	if err := os.Rename(tmp, filepath.Join(dir, draftName)); err != nil {
		_ = os.Remove(tmp)
		return fail(err)
	}
	if err := syncDir(dir); err != nil {
		return fail(err)
	}
	return nil
}

// syncDir 는 디렉터리 항목을 디스크에 넣는다 — rename 이 전원이 나간 뒤에도 남게.
func syncDir(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	err = f.Sync()
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

// afterLstat 은 openPlain 의 Lstat 과 Open 사이다. 시험이 그 틈에 파일을 바꿔 끼운다 (symlink 로 바꾸기 · 키우기).
var afterLstat = func(string) {}

// openPlain 은 보통 파일 하나를 연다 — Lstat 이 보통 파일이고 max 안일 때만 열고, 연 파일이 Lstat 과 같은
// 파일일 때만 돌려준다. 사이에 symlink 로 바뀌어도 그 내용을 읽지 않는다 (계획 4.1 7 · 30번). symlink 가
// 가리키는 호스트 파일이나 FIFO 를 열지 않는다 — 계약의 명령이 그 자리를 바꿀 수 있다.
func openPlain(p string, max int64) (*os.File, os.FileInfo, error) {
	fi, err := os.Lstat(p)
	if err != nil {
		return nil, nil, err
	}
	if !fi.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("%s is not a regular file (%s)", p, fi.Mode().Type())
	}
	if fi.Size() > max {
		return nil, nil, fmt.Errorf("%s is larger than %d bytes", p, max)
	}
	afterLstat(p)
	f, err := os.Open(p)
	if err != nil {
		return nil, nil, err
	}
	now, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, nil, err
	}
	if !os.SameFile(fi, now) {
		f.Close()
		return nil, nil, fmt.Errorf("%s changed while it was opened", p)
	}
	return f, fi, nil
}

// readDraft 는 대기 자리의 초안을 읽는다 — 보통 파일 · 주인이 uid (노드 uid) · 1 MiB 까지 · schema 1. 주인
// 확인은 lower 의 readJSON 을 따른다 (perm_linux.go).
func readDraft(dir string, uid int) (Draft, error) {
	fail := func(err error) (Draft, error) { return Draft{}, fmt.Errorf("cannot read the bake draft: %w", err) }
	p := filepath.Join(dir, draftName)
	f, fi, err := openPlain(p, draftMax)
	if err != nil {
		return fail(err)
	}
	defer f.Close()
	owner, ok := fileOwner(fi)
	switch {
	case !ok:
		return fail(fmt.Errorf("%s: cannot tell its owner on this system", p))
	case owner != uid:
		return fail(fmt.Errorf("%s is owned by uid %d, not %d", p, owner, uid))
	}
	b, err := io.ReadAll(io.LimitReader(f, draftMax+1))
	if err != nil {
		return fail(err)
	}
	if len(b) > draftMax {
		return fail(fmt.Errorf("%s is larger than %d bytes", p, draftMax))
	}
	var d Draft
	if err := json.Unmarshal(b, &d); err != nil {
		return fail(fmt.Errorf("%s: %w", p, err))
	}
	if d.Schema != draftSchema {
		return fail(fmt.Errorf("%s has schema %d, want %d", p, d.Schema, draftSchema))
	}
	return d, nil
}

// readPinned 는 닫은 뒤 대기 자리의 고정 manifest 의 sha256 이다 (계획 4.1 30번). pinned 는 builds 앞에 뜨므로
// 계약의 명령이 그 파일을 symlink · FIFO · 장치로 바꿀 수 있다 — 보통 파일이고 16 MiB 안일 때만 읽는다.
func readPinned(p string) (string, error) {
	f, _, err := openPlain(p, pinnedMax)
	if err != nil {
		return "", fmt.Errorf("cannot pin the manifest: %w", err)
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, pinnedMax+1))
	if err != nil {
		return "", fmt.Errorf("cannot pin the manifest: %w", err)
	}
	if n > pinnedMax {
		return "", fmt.Errorf("cannot pin the manifest: %s is larger than %d bytes", p, pinnedMax)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
