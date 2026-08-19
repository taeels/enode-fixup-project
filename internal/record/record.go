// Package record 는 Run Record 를 만들고 봉인한다 (ADR-005).
//
// ★ Run Record 와 Evidence 는 다른 것이다 ★
// Run Record 는 하나의 Run 이 남긴 ★ 사실 ★ 의 봉인된 묶음이고 MVP 안에 있다.
// Evidence 는 거기에 "어떤 Decision 을 뒷받침한다" 는 ★ 관계 ★ 가 붙은 것이며
// Agent Office 몫, 즉 범위 밖이다.
//
// ★ 왜 DB 가 아니라 파일시스템인가 ★ (ADR-015 §3)
// I4(봉인)를 파일시스템은 강제할 수 있고 행은 못 한다. 봉인된 디렉터리는
// 권한으로 불변이 되지만 봉인된 행은 애플리케이션 규약일 뿐이다.
// 그리고 성질 4(자기충족)는 tar 하나로 건네지는 디렉터리에서 자연스럽다.
package record

import (
	"archive/tar"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Store struct{ Root string }

func New(root string) *Store { return &Store{Root: root} }

func (s *Store) dir(runID string) string {
	return filepath.Join(s.Root, "run-"+safe(runID))
}

// safe 는 run_id 를 경로 성분 하나로 만든다. run_id 는 (change-id, patchset) 에서
// 유도되므로 대체로 안전하지만, 경로 탈출은 ★ 절대 ★ 허용하지 않는다.
func safe(id string) string {
	r := strings.NewReplacer("/", "_", "\\", "_", "..", "_", string(os.PathSeparator), "_")
	return r.Replace(id)
}

// Open 은 Run 이 시작될 때 쓰기 가능한 디렉터리를 만든다.
// ★ 실행 중에는 붙이기만 한다 ★ (성질 1: append-only).
func (s *Store) Open(runID string) error {
	for _, d := range []string{"steps", "logs", "blobs"} {
		if err := os.MkdirAll(filepath.Join(s.dir(runID), d), 0o755); err != nil {
			return err
		}
	}
	return nil
}

// AppendLog 는 그 단계가 뱉은 것을 원문 그대로 남긴다 (ADR-005 의 logs/).
// blob 과 자리가 다르다 — blob 은 단계 ★ 사이 ★ 를 오가는 산출물이고
// log 는 그 단계가 ★ 뱉은 것 ★ 으로 아무도 읽지 않고 기록에만 남는다.
func (s *Store) AppendLog(runID string, seq int, name string, r io.Reader, limit int64) (int64, error) {
	p := filepath.Join(s.dir(runID), "logs", fmt.Sprintf("%02d-%s.log", seq, safe(name)))
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	n, err := io.Copy(f, io.LimitReader(r, limit))
	if err != nil {
		return n, err
	}
	// 상한을 넘으면 잘라 저장하고 ★ 잘렸음을 표시한다 ★ (ADR-015 §5)
	if n == limit {
		_, _ = fmt.Fprintf(f, "\n… 로그가 %d 바이트에서 잘렸습니다\n", limit)
	}
	return n, nil
}

// Sealed 는 이미 봉인됐는지다. 봉인은 한 번뿐이다 (I4).
func (s *Store) Sealed(runID string) bool {
	fi, err := os.Stat(filepath.Join(s.dir(runID), "verdict.json"))
	return err == nil && fi.Mode().Perm()&0o200 == 0
}

// Seal 은 Run 이 종료 상태에 이르렀을 때 기록을 완성하고 ★ 불변으로 만든다 ★.
//
//	run-<id>/
//	├── manifest.json   Run id · Work · 요청자 · ★ 계약 전문 ★ · 시각 · 최종 상태
//	├── steps/NN-*.json ★ node id + label ★ · 시각 · 결과
//	├── logs/NN-*.log   원문 그대로 (실행 중에 쌓인다)
//	├── blobs/          단계 간 산출물 (S8)
//	└── verdict.json    ⑩ 의 대조 결과
//
// 계약 전문이 manifest 에 들어가는 것이 성질 4(자기충족)의 핵심이다 —
// ADR-020 이 스키마를 ★ 인라인 ★ 으로 둔 덕에 무엇으로 검증했는지까지 함께 남는다.
func (s *Store) Seal(runID string, manifest, verdict any, steps []StepFile) error {
	if s.Sealed(runID) {
		return nil // 봉인은 멱등이다. 다시 쓰지 않는다.
	}
	if err := s.Open(runID); err != nil {
		return err
	}
	d := s.dir(runID)
	if err := writeJSON(filepath.Join(d, "manifest.json"), manifest); err != nil {
		return err
	}
	for _, st := range steps {
		p := filepath.Join(d, "steps", fmt.Sprintf("%02d-%s.json", st.Seq, safe(st.Name)))
		if err := writeJSON(p, st); err != nil {
			return err
		}
	}
	if err := writeJSON(filepath.Join(d, "verdict.json"), verdict); err != nil {
		return err
	}
	// 중단된 업로드의 임시 파일이 봉인에 섞이지 않게 한다.
	if ents, err := os.ReadDir(filepath.Join(d, "blobs")); err == nil {
		for _, e := range ents {
			if strings.HasPrefix(e.Name(), ".tmp-") {
				_ = os.Remove(filepath.Join(d, "blobs", e.Name()))
			}
		}
	}
	return seal(d)
}

// StepFile 은 steps/NN-*.json 이다.
// ★ node id 와 label 이 반드시 들어간다 ★ (성질 3) — Case D 의
// "서로 다른 기계였다" 가 여기 걸린다. label 이 없으면 사람이 못 읽는다.
type StepFile struct {
	Seq       int    `json:"seq"`
	StepID    string `json:"step_id"` // run_id#NN — 사람이 읽는 표기
	Name      string `json:"name"`
	Uses      string `json:"uses"`
	Kind      string `json:"kind"`
	Node      string `json:"node"`
	NodeLabel string `json:"node_label"`
	State     string `json:"state"`
	StartedAt string `json:"started_at,omitempty"`
	EndedAt   string `json:"ended_at,omitempty"`
	Result    any    `json:"result,omitempty"`
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

// seal 은 ★ 파일시스템이 불변을 강제하게 한다 ★ (ADR-015 §3).
// 애플리케이션이 "고치지 않기로 한다" 가 아니라 쓰기 비트를 내린다.
func seal(dir string) error {
	var files []string
	var dirs []string
	err := filepath.Walk(dir, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			dirs = append(dirs, p)
		} else {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, p := range files {
		if err := os.Chmod(p, 0o444); err != nil {
			return err
		}
	}
	// 디렉터리는 파일을 다 잠근 뒤에 잠근다 — 순서를 뒤집으면 못 들어간다.
	for i := len(dirs) - 1; i >= 0; i-- {
		if err := os.Chmod(dirs[i], 0o555); err != nil {
			return err
		}
	}
	return nil
}

// unseal 은 쓰기 비트를 되돌린다.
//
// ★ 운영상 함의 하나 ★ — 봉인은 ★ 삭제까지 막는다 ★. 그것이 I4 의 값이지만,
// 디스크가 차면 사람이 지우지도 못한다. ADR-005 가 "장기 보관 정책은 MVP 밖,
// 쌓아두기만 한다" 로 미뤄둔 자리이며, 보관 정책이 생기면 여기가 그 입구다.
// 지금은 테스트 정리에만 쓴다.
func unseal(dir string) error {
	return filepath.Walk(dir, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			return os.Chmod(p, 0o755)
		}
		return os.Chmod(p, 0o644)
	})
}

var ErrNotSealed = errors.New("봉인되지 않았다")

// Tar 는 봉인된 묶음을 통째로 내보낸다.
//
// ★ 성질 4(자기충족)가 전송 형식까지 정한다 ★ — 묶음 하나를 받아 풀면
// 그 안에 전부 있다. 외부 조회가 필요하면 실패다.
func (s *Store) Tar(runID string, w io.Writer) error {
	d := s.dir(runID)
	if !s.Sealed(runID) {
		return ErrNotSealed
	}
	tw := tar.NewWriter(w)
	defer tw.Close()
	base := filepath.Base(d)
	return filepath.Walk(d, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(d, p)
		if err != nil {
			return err
		}
		h, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return err
		}
		h.Name = filepath.ToSlash(filepath.Join(base, rel))
		if err := tw.WriteHeader(h); err != nil {
			return err
		}
		if fi.IsDir() {
			return nil
		}
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
}

// ── blob — 단계 사이를 오가는 산출물 (run-contract §4 별 모양) ────────────
//
// ★ 왜 <seq>-<name> 인가 ★
// 계약에서 parent_build 와 patch_build 가 ★ 둘 다 artifact 를 낸다 ★.
// 이름만으로 키를 잡으면 뒤엣것이 앞엣것을 덮고, 봉인된 기록에 하나만 남아
// 차분 반증의 두 아티팩트를 구분할 수 없게 된다 (성질 4 자기충족이 깨진다).
//
// 그래서 저장은 단계별로 하고, 조회는 이름으로 ★ 가장 최근 것 ★ 을 준다 —
// 생산자는 자기가 몇 번째인지 알고, 소비자는 이름만 안다.
func (s *Store) blobPath(runID string, seq int, name string) string {
	return filepath.Join(s.dir(runID), "blobs", fmt.Sprintf("%02d-%s", seq, safe(name)))
}

// WriteBlob 은 그 단계의 산출물을 저장한다. 상한을 넘으면 ★ 저장하지 않는다 ★.
func (s *Store) WriteBlob(runID string, seq int, name string, r io.Reader, limit int64) (int64, error) {
	p := s.blobPath(runID, seq, name)
	f, err := os.CreateTemp(filepath.Dir(p), ".tmp-*")
	if err != nil {
		return 0, err
	}
	tmp := f.Name()
	defer os.Remove(tmp) //nolint:errcheck // 이름이 바뀌었으면 무해하다

	// limit+1 까지 읽어 초과를 감지한다 — 자르지 않는다.
	// ★ 잘린 산출물은 산출물이 아니다 ★ (로그와 다른 점이다. 로그는 잘라 표시한다.)
	n, err := io.Copy(f, io.LimitReader(r, limit+1))
	f.Close()
	if err != nil {
		return n, err
	}
	if n > limit {
		return n, ErrTooBig
	}
	return n, os.Rename(tmp, p)
}

var (
	ErrTooBig = errors.New("산출물이 상한을 넘는다")
	ErrNoBlob = errors.New("그런 산출물이 없다")
)

// OpenBlob 은 이름으로 ★ 가장 최근 단계의 것 ★ 을 연다.
func (s *Store) OpenBlob(runID, name string) (io.ReadCloser, int64, error) {
	ents, err := os.ReadDir(filepath.Join(s.dir(runID), "blobs"))
	if err != nil {
		return nil, 0, ErrNoBlob
	}
	suffix := "-" + safe(name)
	best := ""
	for _, e := range ents {
		if strings.HasSuffix(e.Name(), suffix) && e.Name() > best {
			best = e.Name() // 접두가 %02d 라 문자열 비교가 곧 순번 비교다
		}
	}
	if best == "" {
		return nil, 0, ErrNoBlob
	}
	p := filepath.Join(s.dir(runID), "blobs", best)
	fi, err := os.Stat(p)
	if err != nil {
		return nil, 0, ErrNoBlob
	}
	f, err := os.Open(p)
	return f, fi.Size(), err
}
