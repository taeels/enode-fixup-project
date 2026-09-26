// Package scratch 는 runc-overlay 노드의 scratch 에서 단계가 남긴 작업 폴더를 버리고
// 지우는 규칙이다 (ADR-076 §4.1 · requirements.md 5.4).
//
// 담는 것 — trash 의 자리와 rename 한 번의 옮기기 · 작업 폴더의 세션 잠금 · 지우기 전의
// 측정과 삭제의 경계 · 배경 삭제자와 scratch 의 양. 모르는 것 — namespace 를 여는 일
// (internal/enode 가 Deleter.Launch 로 채운다) · 광고 · lower 와 합치기 · 굽기 · Mediator.
//
// 표준 라이브러리와 golang.org/x/sys 만 쓴다 (경계 시험의 봉인). Mediator 는 이 패키지를
// 링크하지 않는다.
package scratch

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// TrashName 은 scratch 안의 trash 이름이다.
const TrashName = "trash"

// Trash 는 <scratch>/trash 다. 단계가 남긴 작업 폴더가 rename 한 번으로 들어온다.
type Trash struct {
	Dir string
}

// TrashIn 은 scratch 의 trash 다. 자리의 규칙이 한 곳에 있게 여기서만 짓는다.
func TrashIn(scratchDir string) Trash {
	return Trash{Dir: filepath.Join(scratchDir, TrashName)}
}

// Move 는 path 를 trash 로 옮긴다. rename 한 번이라 크기와 무관하다 (requirements.md 5.4).
//
// 이름은 path 의 마지막 조각이다. 같은 이름이 trash 에 있으면 "-1" · "-2" … 를 붙인다 —
// 작업 폴더 이름은 MkdirTemp 로 유일하지만, 지운 이름이 다시 나오는 동안 옛 항목이 아직
// trash 에 남아 있을 수 있다. trash 가 없으면 0700 으로 만든다. 돌려주는 것은 trash 안의
// 이름이다.
//
// 실패하면 지우지 않는다. 지우기로 떨어지면 그 비용을 보고 전 창에 다시 들인다 — 같은
// filesystem 안의 rename 이 실패하면 설정이나 권한이 틀린 것이고, 그 사실이 오류로 남아야 한다.
func (t Trash) Move(path string) (string, error) {
	if err := os.MkdirAll(t.Dir, 0o700); err != nil {
		return "", fmt.Errorf("create trash: %w", err)
	}
	base := filepath.Base(path)
	name := base
	for i := 1; ; i++ {
		dst := filepath.Join(t.Dir, name)
		_, err := os.Lstat(dst)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			err = os.Rename(path, dst)
			if err == nil {
				return name, nil
			}
			// 사이에 누가 같은 이름을 들였다 (ENOTEMPTY 도 ErrExist 로 읽힌다). 다음 이름으로 간다
			if !errors.Is(err, fs.ErrExist) {
				return "", err
			}
		case err != nil:
			return "", err
		}
		name = fmt.Sprintf("%s-%d", base, i)
	}
}

// Entries 는 trash 의 바로 아래 이름들이다 (이름순). 노드 uid 로 읽는다 — trash 자체는
// 노드 소유다. trash 가 아직 없으면 빈 목록이다.
func (t Trash) Entries() ([]string, error) {
	ents, err := os.ReadDir(t.Dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(ents))
	for _, e := range ents {
		names = append(names, e.Name())
	}
	return names, nil
}

// CheckEntry 는 helper 가 받은 이름을 본다. trash 의 바로 아래 이름 하나만 받는다
// (business-rules.md 5절). 빈 이름 · "." · ".." · 구분자가 든 이름은 거절한다. trash 자체가
// 디렉터리가 아니거나 symlink 면 거절한다 — lstat 으로 본다.
func CheckEntry(trashDir, entry string) error {
	if entry == "" || entry == "." || entry == ".." || strings.ContainsAny(entry, `/\`) {
		return fmt.Errorf("invalid trash entry name: %q", entry)
	}
	fi, err := os.Lstat(trashDir)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return fmt.Errorf("trash is not a plain directory: %s", trashDir)
	}
	return nil
}
