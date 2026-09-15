// 진행 파일 — 봉인 전의 원문이 쌓이는 자리다 (FR-5).
//
// logs/ 와 무엇이 다른가
// logs/ 는 봉인된 기록의 일부라 한 번 쌓이면 사람이 읽을 때까지 그대로 있고,
// 진행 파일은 Run 이 도는 동안에만 산다. Run 이 끝나면 봉인이 지우고(R22),
// 고아가 되면 회수기가 지운다(R24). 그래서 자리도 다르다 — 진행 트리는
// run-<id>/ 안이 아니라 그 형제인 <Root>/progress/run-<id>/ 다 (R1).
//
// 왜 형제인가
// seal 의 Walk 과 Tar 의 Walk 이 둘 다 s.dir(runID) 만 돈다 (record.go:159 ·
// record.go:229). 진행 트리를 그 안에 두면 봉인이 쓰기 비트를 내려 지울 수
// 없게 되고(R36), tar 에 원문이 실려 나가 선별본만 남긴다는 약속이 깨진다
// (R35). s.Root 를 통째로 도는 코드는 이 저장소에 없다 — s.Root 가 쓰이는
// 자리는 record.go:33 의 dir() 하나다. 형제로 두면 그 둘이 자동으로 참이다.
package record

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Progress 는 진행 파일의 지금 상태다. 세 값이 전부 폴링하는 쪽의 것이다 —
// Total 은 다음 from, Attempt 는 앞 시도의 바이트를 버릴 신호, Capped 는
// 더 기다려도 안 자란다는 신호다.
type Progress struct {
	Total   int64 // 파일의 크기. 표시 줄도 든다 (R13)
	Attempt int   // 지금 시도. 파일 이름에서 읽는다
	Capped  bool  // 마지막 줄이 표시 줄이다 (R14)
}

// ProgressMaxAge 는 살아 있는 Run 의 진행 파일이 디스크에 앉아 있을 수 있는
// 상한이다 — 마지막 쓰기부터 센다 (N2 의 V5).
//
// 이 상수가 record 에 있는 이유는 값이 지배하는 것 옆이기 때문이다. 자르는
// 것은 노출 시간이지 디스크가 아니다 (NFR Requirements 4.2) — 최악 봉투
// 10 GiB 는 이 값과 무관하게 그대로 서 있다.
const ProgressMaxAge = 6 * time.Hour

// cappedType 은 표시 줄의 type 이다.
//
// type 에 점을 넣는다 — 실측한 하네스의 type 은 전부 홑단어라
// (system · assistant · user · result · rate_limit_event) 부딪칠 수 없다.
// elidedMark 가 같은 근거로 enode.elided 를 쓴다 (internal/enode/runner.go:415).
const cappedType = "enode.capped"

// cappedMark 는 총 길이 상한에 닿았음을 표시하는 줄이다 (R12).
//
// bytes 는 상한이다. 잘린 자리가 아니다 — 멈춘 자리는 Total 이 이미 말한다.
//
// Bytes 가 int64 인 것이 elidedMark 의 int 와 다른 유일한 자리다. 상한이
// int64 로 온다 (internal/config 의 MaxBlobBytes).
type cappedMark struct {
	Type  string `json:"type"`
	Bytes int64  `json:"bytes"`
}

// cappedMarker 는 표시 줄과 그 개행을 한 버퍼로 짓는다 (R44).
// 나누면 그 사이에 다른 쓰기가 끼어 마지막 줄이 반쪽이 될 수 있다.
func cappedMarker(limit int64) []byte {
	b, err := json.Marshal(cappedMark{Type: cappedType, Bytes: limit})
	if err != nil {
		return nil
	}
	return append(b, '\n')
}

// ── 자리와 이름 ──────────────────────────────────────────────────────────

func (s *Store) progressRoot() string { return filepath.Join(s.Root, "progress") }

// progressDirName 은 Run 하나의 진행 트리 이름이다.
// safe 를 다시 짓지 않는다 — record.go:38 의 것을 그대로 쓴다.
func progressDirName(runID string) string { return "run-" + safe(runID) }

func (s *Store) progressDir(runID string) string {
	return filepath.Join(s.progressRoot(), progressDirName(runID))
}

// progressKey 는 (seq, name) 하나의 키다. 시도를 안 넣는다 (R28) —
// 같은 단계의 다른 시도가 같은 자리를 다투므로 잠금은 시도보다 굵어야 한다.
func progressKey(seq int, name string) string {
	return fmt.Sprintf("%02d-%s", seq, safe(name))
}

// progressFile 은 NN-<name>.<attempt>.log 다 (R2).
// %02d 와 %d 는 blobPath (record.go:270) 의 규칙을 따른다.
func progressFile(seq int, name string, attempt int) string {
	return fmt.Sprintf("%s.%d.log", progressKey(seq, name), attempt)
}

// parseProgressName 은 이름에서 순번과 시도와 단계 이름을 읽는다 (R6).
//
// 오른쪽에서 읽는다 — 단계 이름에 점이 들어갈 수 있다. build.step 같은 이름을
// 왼쪽에서 읽으면 시도 자리에 step 이 들어와 이름이 통째로 안 읽힌다.
//
// parseBlobName (record.go:304) 을 재사용 못 하는 이유가 여기다. 그쪽 이름은
// %02d.%d-이름 이라 순번과 시도가 둘 다 이름 앞에 있고 왼쪽에서 읽는다.
// 이쪽은 시도가 이름 뒤에 온다.
//
// 못 읽으면 ok 가 거짓이고 부르는 쪽은 조용히 건너뛴다 (R34) — 진행 트리에
// 사람이 넣어둔 것이 있어도 쓰기와 쓸기를 막지 않는다.
func parseProgressName(n string) (seq, attempt int, name string, ok bool) {
	stem, found := strings.CutSuffix(n, ".log")
	if !found {
		return 0, 0, "", false
	}
	dot := strings.LastIndexByte(stem, '.')
	if dot < 0 {
		return 0, 0, "", false
	}
	a, aok := atoiDigits(stem[dot+1:])
	head := stem[:dot]
	dash := strings.IndexByte(head, '-')
	if dash < 0 {
		return 0, 0, "", false
	}
	q, qok := atoiDigits(head[:dash])
	if !aok || !qok {
		return 0, 0, "", false
	}
	return q, a, head[dash+1:], true
}

// atoiDigits 는 숫자만으로 된 열만 받는다. strconv.Atoi 는 부호를 받지만
// 이 이름에 부호가 들어갈 자리가 없다 — 받으면 같은 시도에 이름이 둘 생긴다.
func atoiDigits(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
	}
	n, err := strconv.Atoi(s)
	return n, err == nil
}

// ── 잠금 — 단계마다 하나, Run 마다 드롭 하나 ─────────────────────────────

// progressLocks 는 Run 마다의 잠금 묶음이다. 게으르게 난다 — Store 를 여는
// 자리(New)를 안 건드린다.
//
// 키가 runID 가 아니라 디렉터리 이름인 것은 쓸기 때문이다. 쓸기는 디스크에서
// 디렉터리 이름만 보고 오고 safe 는 되돌릴 수 없다 (계획 5절 ⑤). 지키는
// 대상이 그 트리이므로 키도 그 트리의 이름이어야 양쪽이 같은 자물쇠를 잡는다.
type progressLocks struct {
	mu   sync.Mutex
	runs map[string]*runLocks
}

type runLocks struct {
	drop  sync.RWMutex           // R31. 쓰기와 읽기는 RLock · 드롭은 Lock
	mu    sync.Mutex             // steps 를 지킨다
	steps map[string]*sync.Mutex // 키는 (seq, name). 시도를 안 넣는다 (R28)
	refs  int                    // 0 이 될 때만 runs 에서 걷는다 (R30)
}

// acquireLocks 는 그 트리의 잠금 묶음을 얻고 참조를 하나 올린다.
//
// 참조 수가 R31 이 이름 붙인 함정을 막는다. 뮤텍스를 맵에서 지우는 순간
// 그것을 쥔 쓰기가 있으면 다음 호출이 새 뮤텍스를 만들고, 둘이 같은 파일에서
// 동시에 돈다. 참조가 남아 있는 동안에는 안 지워지므로 그 창이 아예 없다.
//
// Ring 이 같은 자리를 뮤텍스 하나로 풀었다 (internal/enode/transcript.go:40).
// 거기는 쓰는 쪽이 한 노드에 하나라 굵어도 됐고, 여기는 Run 이 동시에 여럿이라
// 굵게 잡으면 남의 Run 이 기다린다.
func (s *Store) acquireLocks(dirName string) *runLocks {
	s.progress.mu.Lock()
	defer s.progress.mu.Unlock()
	if s.progress.runs == nil {
		s.progress.runs = map[string]*runLocks{}
	}
	rl := s.progress.runs[dirName]
	if rl == nil {
		rl = &runLocks{steps: map[string]*sync.Mutex{}}
		s.progress.runs[dirName] = rl
	}
	rl.refs++
	return rl
}

func (s *Store) releaseLocks(dirName string, rl *runLocks) {
	s.progress.mu.Lock()
	defer s.progress.mu.Unlock()
	rl.refs--
	if rl.refs == 0 && s.progress.runs[dirName] == rl {
		delete(s.progress.runs, dirName)
	}
}

// step 은 (seq, name) 하나의 뮤텍스다. 키가 다르면 안 기다린다 (R29).
func (rl *runLocks) step(key string) *sync.Mutex {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	m := rl.steps[key]
	if m == nil {
		m = &sync.Mutex{}
		rl.steps[key] = m
	}
	return m
}

// ── 붙이기 ───────────────────────────────────────────────────────────────

// AppendProgress 는 노드가 보낸 원문 청크를 진행 파일에 붙인다.
//
// 순서가 뜻을 가진다 (business-logic-model.md 1.1) —
// ① 키를 잠그고 ② 자리를 만들고 ③ 지금 시도를 읽고 ④ 시도를 견주고
// ⑤ 이미 닿았는지 보고 ⑥ 붙이고 ⑦ 지금 상태를 낸다.
// ② ~ ⑦ 이 전부 ① 의 잠금 안이다 — 밖으로 나가는 순간 같은 단계의 두 청크가
// 서로의 크기를 보고 각자 다른 답을 쓴다.
func (s *Store) AppendProgress(runID string, seq int, name string,
	attempt int, r io.Reader, limit int64) (Progress, error) {
	dirName := progressDirName(runID)
	rl := s.acquireLocks(dirName)
	defer s.releaseLocks(dirName, rl)
	// 드롭과만 배타다. 쓰기끼리는 이 자물쇠에서 안 막힌다 (R31).
	rl.drop.RLock()
	defer rl.drop.RUnlock()
	m := rl.step(progressKey(seq, name))
	m.Lock()
	defer m.Unlock()

	// ② 게으르게 만든다 (R3). Store.Open (record.go:45) 을 안 건드린다 —
	// Seal 이 s.Open 을 부르므로 (record.go:96) 거기에 넣으면 지우기 직전에
	// 다시 만든다.
	dir := filepath.Join(s.progressRoot(), dirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return Progress{}, err
	}

	// ③ 곁파일이 0 이라 이름이 진실이다 (R5 · R6).
	cur, found := maxAttempt(dir, seq, name)
	if found {
		// ④ 늦게 온 앞 시도의 청크는 버린다 (R8). 0 바이트를 쓰고 지금
		// 상태를 낸다 — 오류가 아니다. 노드가 재시도를 이미 시작했을 뿐이다.
		if attempt < cur {
			return s.progressOf(dir, seq, name, cur)
		}
		// ④ 새 시도가 왔으면 앞 시도를 걷는다 (R9 · R16). 걷기가 실패해도
		// 진행을 안 막는다 (R25) — 걷는 범위는 같은 (seq, name) 의 더 작은
		// 시도뿐이라 다른 단계도 다른 Run 도 안 건드린다.
		if attempt > cur {
			dropOlderAttempts(dir, seq, name, attempt)
		}
	}

	p := filepath.Join(dir, progressFile(seq, name, attempt))
	// 읽기도 연다 — 마지막 줄을 봐야 하고(R14) 예산 안의 개행을 되짚어야 한다.
	f, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return Progress{}, err
	}
	fi, err := f.Stat()
	if err != nil {
		f.Close() //nolint:errcheck // 이미 실패했다
		return Progress{}, err
	}

	// ⑤ 이미 닿았으면 0 바이트를 쓴다 (R15). 판정의 근거는 표시 줄의 존재다.
	// 크기가 아니다 (R14).
	var werr error
	if !cappedTail(f, fi.Size()) {
		werr = appendWithinLimit(f, r, fi.Size(), limit) // ⑥
	}

	// ⑦ Total 은 쓰기 직후의 파일 크기다 (R13). 잠금 안에서 잰다.
	total, serr := fileSize(f)
	if serr != nil {
		f.Close() //nolint:errcheck // 이미 실패했다
		if werr != nil {
			return Progress{}, werr
		}
		return Progress{}, serr
	}
	pr := Progress{Total: total, Attempt: attempt, Capped: cappedTail(f, total)}
	if cerr := f.Close(); cerr != nil && werr == nil {
		werr = cerr
	}
	return pr, werr
}

// appendWithinLimit 은 청크를 붙이고 상한에 닿으면 표시 줄로 닫는다.
// nfr-design-patterns.md 1.1 의 순서 그대로다.
func appendWithinLimit(f *os.File, r io.Reader, size, limit int64) error {
	budget := limit - size
	// 예산이 0 이하인데 표시 줄이 없는 파일은 앞의 어떤 호출이 중간에 죽은
	// 것이다. 쓸 것이 없고 닫을 것만 남았다.
	overflow := budget <= 0
	if budget > 0 {
		// 한 바이트를 더 읽어야 넘겼는지 안다. 넘기지 않았으면 청크가 통째로
		// 들어간다 — 개행에 안 맞춘다 (R10). 맞추면 Total 이 노드가 보낸
		// 것보다 뒤처지고, 노드가 그 차이를 다시 보내 루프가 돈다.
		n, err := io.Copy(f, io.LimitReader(r, budget+1))
		if err != nil {
			return err
		}
		overflow = n > budget
		if overflow {
			// R11 — 예산 안의 마지막 개행까지만 남긴다. 반쪽 줄을 안 남긴다.
			// 예산 안에 개행이 없으면 0 바이트다.
			start := size
			cut, err := lastNewline(f, start, start+budget)
			if err != nil {
				return err
			}
			keep := start
			if cut >= 0 {
				keep = cut + 1
			}
			if err := f.Truncate(keep); err != nil {
				return err
			}
			size = keep
		} else {
			size += n
		}
	}
	if !overflow {
		return nil
	}
	// R43 — 표시 줄 앞의 개행을 보장한다.
	//
	// 없으면 무슨 일이 나는가: 반쪽 줄이 표시 줄과 붙어 한 줄이 된다. 그 줄은
	// JSON 이 아니라 파싱이 안 되고, R14 의 「마지막 줄이 표시 줄인가」가
	// 거짓이 된다. 그러면 Capped 가 영영 참이 안 되고 다음 호출마다 표시 줄이
	// 하나씩 더 붙어 무한히 쌓인다.
	if size > 0 {
		last, err := endsWithNewline(f, size)
		if err != nil {
			return err
		}
		if !last {
			if _, err := f.Write([]byte{'\n'}); err != nil {
				return err
			}
		}
	}
	// R44 — 표시 줄과 그 개행을 한 번의 쓰기로 낸다.
	//
	// 여기를 지나면 Total 이 상한보다 클 수 있다. 그것이 정상이다 (R45) —
	// 상한은 원문에 거는 것이고 표시 줄은 시스템이 더한 것이다.
	_, err := f.Write(cappedMarker(limit))
	return err
}

// lastNewline 은 [from, to) 안의 마지막 개행 자리다. 없으면 -1.
// 뒤에서부터 창으로 되짚는다 — 청크가 10 MiB 여도 메모리는 창 하나다.
func lastNewline(f *os.File, from, to int64) (int64, error) {
	const window = 64 << 10
	buf := make([]byte, window)
	for to > from {
		start := to - window
		if start < from {
			start = from
		}
		n := int(to - start)
		if _, err := f.ReadAt(buf[:n], start); err != nil {
			return -1, err
		}
		if i := bytes.LastIndexByte(buf[:n], '\n'); i >= 0 {
			return start + int64(i), nil
		}
		to = start
	}
	return -1, nil
}

func endsWithNewline(f *os.File, size int64) (bool, error) {
	var b [1]byte
	if _, err := f.ReadAt(b[:], size-1); err != nil {
		return false, err
	}
	return b[0] == '\n', nil
}

func fileSize(f *os.File) (int64, error) {
	fi, err := f.Stat()
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
}

// cappedTail 은 마지막 줄이 표시 줄인지 본다 (R14).
//
// 크기로 판정하지 않는다. 개행 없는 큰 청크 하나로 상한을 넘기면 예산 안에
// 개행이 없어 0 바이트가 남고, 그때 Total 은 상한보다 작은데 표시 줄은 이미
// 붙어 있다. 크기 비교는 그 자리에서 거짓을 내고, 반대로 표시 줄이 더해져
// Total 이 상한을 넘은 자리에서는 참을 낸다 — 양쪽으로 다 틀린다.
func cappedTail(f *os.File, size int64) bool {
	if size == 0 {
		return false
	}
	// 표시 줄은 가장 길어야 52 바이트다 (bytes 가 int64 의 큰 값일 때).
	const window = 256
	n := int64(window)
	if size < n {
		n = size
	}
	buf := make([]byte, n)
	if _, err := f.ReadAt(buf, size-n); err != nil {
		return false
	}
	line := buf
	if line[len(line)-1] == '\n' {
		line = line[:len(line)-1]
	}
	if i := bytes.LastIndexByte(line, '\n'); i >= 0 {
		line = line[i+1:]
	}
	var m cappedMark
	if err := json.Unmarshal(line, &m); err != nil {
		return false
	}
	return m.Type == cappedType
}

// maxAttempt 는 그 (seq, name) 의 가장 큰 시도다 (R5).
// 곁파일을 안 둔다 — 이름이 진실이고, 진실이 하나면 둘이 갈릴 일이 없다.
func maxAttempt(dir string, seq int, name string) (int, bool) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return 0, false
	}
	key := progressKey(seq, name)
	best, found := 0, false
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		q, a, nm, ok := parseProgressName(e.Name())
		if !ok || progressKey(q, nm) != key {
			continue
		}
		if !found || a > best {
			best, found = a, true
		}
	}
	return best, found
}

// dropOlderAttempts 는 같은 (seq, name) 의 더 작은 시도를 걷는다 (R9 · R16).
// 실패를 삼킨다 (R25) — 앞 시도의 파일이 남는 것보다 이번 청크가 안 쌓이는
// 편이 더 나쁘다. 남은 파일은 어차피 봉인이나 쓸기가 걷는다.
func dropOlderAttempts(dir string, seq int, name string, attempt int) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	key := progressKey(seq, name)
	for _, e := range ents {
		q, a, nm, ok := parseProgressName(e.Name())
		if !ok || progressKey(q, nm) != key || a >= attempt {
			continue
		}
		_ = os.Remove(filepath.Join(dir, e.Name()))
	}
}

// progressOf 는 그 시도의 지금 상태만 읽는다. 본문을 안 연다.
func (s *Store) progressOf(dir string, seq int, name string, attempt int) (Progress, error) {
	f, err := os.Open(filepath.Join(dir, progressFile(seq, name, attempt)))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Progress{}, nil
		}
		return Progress{}, err
	}
	defer f.Close() //nolint:errcheck // 읽기만 했다
	total, err := fileSize(f)
	if err != nil {
		return Progress{}, err
	}
	return Progress{Total: total, Attempt: attempt, Capped: cappedTail(f, total)}, nil
}

// ── 읽기 ─────────────────────────────────────────────────────────────────

// emptyBody 는 본문이 없는 자리의 ReadCloser 다. nil 을 안 낸다 —
// 부르는 쪽이 갈래마다 nil 검사를 하게 만들면 언젠가 하나를 빠뜨린다.
func emptyBody() io.ReadCloser { return io.NopCloser(bytes.NewReader(nil)) }

// progressBody 는 Total - from 에서 자른 본문이다.
//
// 왜 자르나 — 본문은 잠금 밖에서 읽힌다. 자르지 않으면 그 사이에 붙은
// 바이트까지 나가 본문이 Total 보다 길어지고, 폴링하는 쪽이 다음 from 을
// Total 로 잡았을 때 같은 바이트를 두 번 받는다. CB3 이 재는 것이
// 「두 벌이 안 생긴다」다.
type progressBody struct {
	io.Reader
	f *os.File
}

func (b progressBody) Close() error { return b.f.Close() }

// ReadProgress 는 from 부터의 본문과 지금 상태를 낸다.
//
// attempt 인자가 없다 (R17). 읽는 쪽은 언제나 가장 큰 시도를 본다 — 인자로
// 받으면 앞 시도를 달라는 요청이 생기고, 그 파일은 R9 가 이미 걷었다.
func (s *Store) ReadProgress(runID string, seq int, name string,
	from int64) (Progress, io.ReadCloser, error) {
	dirName := progressDirName(runID)
	rl := s.acquireLocks(dirName)
	defer s.releaseLocks(dirName, rl)
	rl.drop.RLock()
	defer rl.drop.RUnlock()
	m := rl.step(progressKey(seq, name))
	m.Lock()
	defer m.Unlock()

	dir := filepath.Join(s.progressRoot(), dirName)
	att, found := maxAttempt(dir, seq, name)
	if !found {
		// R18 — 아직 아무것도 안 온 단계다. 404 가 아니다. 노드가 늦은 것과
		// 그런 단계가 없는 것을 이 층이 못 가르고, 가르는 것은 계약을 아는
		// 쪽의 일이다.
		return Progress{}, emptyBody(), nil
	}
	f, err := os.Open(filepath.Join(dir, progressFile(seq, name, att)))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Progress{}, emptyBody(), nil
		}
		return Progress{}, nil, err
	}
	total, err := fileSize(f)
	if err != nil {
		f.Close() //nolint:errcheck // 이미 실패했다
		return Progress{}, nil, err
	}
	pr := Progress{Total: total, Attempt: att, Capped: cappedTail(f, total)}

	// from 이 음수인 자리를 0 으로 본다. 검증은 HTTP 표면의 일이고 (R33),
	// 이 층의 전제는 R32 하나다.
	if from < 0 {
		from = 0
	}
	// R19 — from 이 Total 보다 커도 오류가 아니다. 시도가 바뀌어 총 길이가
	// 뒤로 간 뒤의 폴링이 그 모양이다. 빈 본문과 함께 지금 상태를 주면
	// 폴링하는 쪽이 Attempt 가 바뀐 것을 보고 스스로 되감는다.
	if from >= total {
		f.Close() //nolint:errcheck // 읽기만 했다
		return pr, emptyBody(), nil
	}
	if _, err := f.Seek(from, io.SeekStart); err != nil {
		f.Close() //nolint:errcheck // 이미 실패했다
		return pr, nil, err
	}
	// 본문은 잠금 밖에서 읽힌다. 그 사이에 DropProgress 가 돌아도 리눅스에서
	// 열린 fd 는 살아 있어 읽기가 안 깨진다 — 지워지는 것은 이름이다.
	return pr, progressBody{Reader: io.LimitReader(f, total-from), f: f}, nil
}

// ── 지우기 ───────────────────────────────────────────────────────────────

// DropProgress 는 그 Run 의 진행 트리를 통째로 지운다 (R20).
//
// Run 단위다. 단계 단위가 아니다 — 봉인도 종료도 쓸기도 Run 단위로 일어난다.
// 멱등이다 (R21). 없으면 아무것도 안 하고 nil 을 낸다.
func (s *Store) DropProgress(runID string) error {
	return s.dropTree(progressDirName(runID))
}

// dropTree 는 진행 트리 하나를 지운다. DropProgress 와 고아 쓸기가 같은
// 경로를 쓴다 — 두 벌로 두면 한쪽만 잠금을 잡는 날이 온다.
func (s *Store) dropTree(dirName string) error {
	rl := s.acquireLocks(dirName)
	defer s.releaseLocks(dirName, rl)
	// R31 — 드롭은 그 트리의 모든 쓰기와 읽기에 배타다. 쓰는 중에 이름을
	// 지우면 그 쓰기는 지워진 inode 에 계속 붙어 디스크만 먹는다.
	rl.drop.Lock()
	defer rl.drop.Unlock()
	return os.RemoveAll(filepath.Join(s.progressRoot(), dirName))
}

// HasProgress 는 진행 트리가 하나라도 있는지다.
//
// 쓸기를 부르는 쪽이 이것을 먼저 본다 (계획 5절 ④). runs.state 에 인덱스가
// 없어 살아 있는 Run 질의가 전수 스캔이고, 진행 트리가 0 이면 그 스캔이 아예
// 안 돌아야 한다. 디렉터리 한 번 읽기이고 이름 하나만 본다.
func (s *Store) HasProgress() bool {
	f, err := os.Open(s.progressRoot())
	if err != nil {
		return false
	}
	defer f.Close() //nolint:errcheck // 읽기만 했다
	names, err := f.Readdirnames(1)
	return err == nil && len(names) > 0
}

// SweepProgress 는 고아 진행 트리를 걷는다 (N2 · R24).
//
// live 는 살아 있는 Run 의 id 다 — 부르는 쪽(internal/store)이 DB 에서 낸다.
// 정책은 store 가 알고 기제는 record 가 안다. Run 의 상태를 아는 쪽이 store
// 이고, 그 트리가 어디 있는지 아는 쪽이 record 다.
//
// 조건 둘은 배타이고 순서가 규칙이다 (R37) — 상태를 먼저 보고, 종료가 아닐
// 때만 나이를 본다. 뒤집으면 종료된 Run 의 트리가 여섯 시간 더 남아 V4 의
// 「유예 0」과 어긋난다.
//
// 지운 항목 수를 낸다 — 고아 트리 하나가 1, 늙은 파일 하나가 1 이다.
// Record 를 안 건드린다 (R39).
func (s *Store) SweepProgress(live []string, maxAge time.Duration, now time.Time) (int, error) {
	ents, err := os.ReadDir(s.progressRoot())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}
	// 앞으로 가는 사상만 쓴다 (계획 5절 ⑤). 디렉터리 이름에서 runID 를
	// 복원하지 않는다 — safe 는 되돌릴 수 없고, 틀리는 방향이 살아 있는 Run 의
	// 트리를 지우는 쪽이다. 두 runID 가 같은 이름으로 접히면 살아 있는 쪽이
	// 이긴다: 안 지우는 쪽으로 틀린다.
	alive := make(map[string]struct{}, len(live))
	for _, id := range live {
		alive[progressDirName(id)] = struct{}{}
	}
	n := 0
	var first error
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		if _, ok := alive[e.Name()]; ok {
			// 조건 ② — 살아 있는 Run 이다. 파일마다 나이를 본다.
			k, err := s.sweepAged(e.Name(), maxAge, now)
			n += k
			if err != nil && first == nil {
				first = err
			}
			continue
		}
		// 조건 ① — 살아 있는 집합에 없다. 유예 0 으로 통째로 지운다 (V4).
		// DB 에 없는 Run 도 여기로 온다 (R38) — 제출이 반쯤 죽은 자리다.
		if err := s.dropTree(e.Name()); err != nil {
			if first == nil {
				first = err
			}
			continue
		}
		n++
	}
	// 실패는 보조다 (R25). 한 항목이 실패해도 나머지를 계속 돈다.
	return n, first
}

// sweepAged 는 살아 있는 Run 의 늙은 파일만 걷는다 (V5).
// 같은 Run 의 방금 쓴 다른 단계 파일은 안 건드린다.
func (s *Store) sweepAged(dirName string, maxAge time.Duration, now time.Time) (int, error) {
	dir := filepath.Join(s.progressRoot(), dirName)
	ents, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	rl := s.acquireLocks(dirName)
	defer s.releaseLocks(dirName, rl)
	rl.drop.RLock()
	defer rl.drop.RUnlock()
	n := 0
	var first error
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		// 이름이 안 읽히면 건너뛴다 (R34) — 키를 못 만들어 잠글 수 없다.
		// 그 Run 이 끝나면 조건 ① 이 통째로 걷는다.
		seq, _, name, ok := parseProgressName(e.Name())
		if !ok {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		if now.Sub(fi.ModTime()) <= maxAge {
			continue
		}
		m := rl.step(progressKey(seq, name))
		m.Lock()
		err = os.Remove(filepath.Join(dir, e.Name()))
		m.Unlock()
		if err != nil {
			if first == nil {
				first = err
			}
			continue
		}
		n++
	}
	return n, first
}
