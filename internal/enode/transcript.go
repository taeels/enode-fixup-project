package enode

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// 하네스 트랜스크립트 링 파일 (decisions §6.2 · §6.3).
//
// 하네스가 도는 동안 뱉는 글자는 지금까지 어디에도 없었다 — runner.go 가
// bytes.Buffer 로 통째로 받아 프로세스가 끝나야 Decode 했다. 이 파일이 그것을
// 노드의 고정 크기 링 파일에 흘려, 제어판이 1초로 읽어 카드에 그리게 한다.
//
// 윈도우가 이 설계를 정했다 — Go 가 FILE_SHARE_DELETE 를 안 줘서 읽는 쪽이
// 열고 있으면 rename·삭제가 막히고, Truncate 와 읽기가 겹치면 읽는 쪽이 깨진
// 것을 본다. 그래서 회전·자르기·삭제를 아예 안 하고 WriteAt 만 쓴다. 크기가
// 안 바뀌고 이름이 안 바뀌고 안 지워지고 잠금이 없다 — 윈도우와 유닉스가 같은
// 코드로 돈다. 빌드 태그 쌍이 하나도 안 는다.

const (
	ringMagic      = "ENTR"
	ringVersion    = 1
	ringHeaderSize = 32
	// DefaultTranscriptCapacity 는 몸통 바이트 수다. 80자 줄로 대략 6,400줄 (decisions §6.2).
	DefaultTranscriptCapacity = 512 * 1024
)

// TranscriptPath 는 설정 파일 옆의 링 파일이다 — <dir>/<stem>.transcript.
// PolicyPath · StatusPath 와 같은 규칙이다 (설정 경로가 곧 신원 · ADR-017).
func TranscriptPath(configPath string) string {
	return strings.TrimSuffix(configPath, filepath.Ext(configPath)) + ".transcript"
}

// Ring 은 노드의 트랜스크립트 링 파일이다. 쓰는 쪽(데몬)이 하나다 — 한 노드는
// 한 번에 한 단계라 동시에 두 곳에서 안 쓴다.
type Ring struct {
	mu       sync.Mutex
	f        *os.File
	capacity int64
	total    uint64
	gen      uint64
}

// Snapshot 은 제어판이 읽어 가는 지금의 트랜스크립트다.
type Snapshot struct {
	Data       []byte // 오래된 것 -> 새것 순서. total<=capacity 면 전부, 넘으면 최근 capacity
	Generation uint64
	Total      uint64
}

func putHeader(b []byte, capacity, total, gen uint64) {
	copy(b[0:4], ringMagic)
	binary.LittleEndian.PutUint16(b[4:6], ringVersion)
	binary.LittleEndian.PutUint64(b[8:16], capacity)
	binary.LittleEndian.PutUint64(b[16:24], total)
	binary.LittleEndian.PutUint64(b[24:32], gen)
}

// OpenRing 은 링 파일을 열거나 만든다. 머리가 없거나 매직·용량이 안 맞거나
// 크기가 틀리면 새로 초기화한다 (총량 0 · 세대 0 · 몸통 0).
func OpenRing(path string, capacity int) (*Ring, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	r := &Ring{f: f, capacity: int64(capacity)}
	want := int64(ringHeaderSize) + int64(capacity)
	hdr := make([]byte, ringHeaderSize)
	st, statErr := f.Stat()
	valid := statErr == nil && st.Size() == want
	if valid {
		if _, e := f.ReadAt(hdr, 0); e != nil {
			valid = false
		} else if string(hdr[0:4]) != ringMagic ||
			binary.LittleEndian.Uint64(hdr[8:16]) != uint64(capacity) {
			valid = false
		}
	}
	if valid {
		r.total = binary.LittleEndian.Uint64(hdr[16:24])
		r.gen = binary.LittleEndian.Uint64(hdr[24:32])
		return r, nil
	}
	// 초기화 — 파일을 want 크기로 맞추고 머리를 쓴다. 이 Truncate 는 만드는
	// 순간뿐이라(도는 중에 안 한다) 읽는 쪽과 겹치지 않는다.
	if err := f.Truncate(want); err != nil {
		f.Close() //nolint:errcheck
		return nil, err
	}
	putHeader(hdr, uint64(capacity), 0, 0)
	if _, err := f.WriteAt(hdr, 0); err != nil {
		f.Close() //nolint:errcheck
		return nil, err
	}
	return r, nil
}

// Write 는 몸통에 흘리고 감긴다. io.Writer 라 io.MultiWriter 한 겹으로 꽂힌다.
//
// 쓰기가 실패해도 오류를 안 낸다 — tee 는 보조이고, 링 쓰기 오류로 MultiWriter
// 가 멈추면 하네스 실행(cmd.Run)까지 멈춘다. 그래서 언제나 len(p), nil 을
// 돌려주고 실패는 삼킨다 (nfr-design §3). 능력은 링이 아니라 봉인이 진다.
func (r *Ring) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(p) == 0 {
		return 0, nil
	}
	cap64 := r.capacity
	// 커서는 흘린 전체 길이만큼 나아간다. 몸통에는 최근 cap 바이트만 남길 수
	// 있으므로, 한 번에 용량보다 많이 오면 꼬리만 쓴다. 꼬리의 첫 바이트의
	// 논리 위치는 (총량 + 길이 - 꼬리길이)이고, 그 자리에서 감아 쓴다 — 그래야
	// 읽는 쪽의 start=총량%용량 재구성과 어긋나지 않는다.
	keep := int64(len(p))
	tail := p
	if keep > cap64 {
		keep = cap64
		tail = p[int64(len(p))-cap64:]
	}
	off := int64((r.total + uint64(len(p)) - uint64(keep)) % uint64(cap64))
	if off+keep <= cap64 {
		r.writeAt(tail, off)
	} else {
		first := cap64 - off
		r.writeAt(tail[:first], off)
		r.writeAt(tail[first:], 0)
	}
	r.total += uint64(len(p))
	r.flushHeader()
	return len(p), nil
}

func (r *Ring) writeAt(b []byte, off int64) {
	_, _ = r.f.WriteAt(b, ringHeaderSize+off) // 실패는 삼킨다 (Write 주석)
}

func (r *Ring) flushHeader() {
	hdr := make([]byte, ringHeaderSize)
	putHeader(hdr, uint64(r.capacity), r.total, r.gen)
	_, _ = r.f.WriteAt(hdr, 0)
}

// Reset 은 새 단계가 시작할 때 총량을 0 으로 하고 세대를 올린다. 파일·몸통은
// 그대로 둔다 — 단계가 끝날 때가 아니라 시작할 때 비운다(decisions §6.2).
func (r *Ring) Reset() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.total = 0
	r.gen++
	r.flushHeader()
	return nil
}

func (r *Ring) Close() error { return r.f.Close() }

// ReadRing 은 제어판이 읽는다. 파일만 있으면 데몬이 죽어 있어도 읽힌다.
//
// 잠금이 없으므로 머리를 다시 읽어 많이 움직였으면 몸통을 한 번 더 읽는다
// (찢긴 읽기 완화). 머리를 마지막에 갱신하므로, 반쯤 쓰인 꼬리를 읽어도 total
// 이 그 꼬리를 아직 안 가리켜 화면에 안 나온다.
func ReadRing(path string) (Snapshot, error) {
	f, err := os.Open(path)
	if err != nil {
		return Snapshot{}, err
	}
	defer f.Close() //nolint:errcheck

	hdr := make([]byte, ringHeaderSize)
	if _, err := f.ReadAt(hdr, 0); err != nil {
		return Snapshot{}, err
	}
	if string(hdr[0:4]) != ringMagic {
		return Snapshot{}, os.ErrInvalid
	}
	capacity := int64(binary.LittleEndian.Uint64(hdr[8:16]))
	total1 := binary.LittleEndian.Uint64(hdr[16:24])

	body := make([]byte, capacity)
	if _, err := f.ReadAt(body, ringHeaderSize); err != nil {
		return Snapshot{}, err
	}

	// 머리를 다시 읽는다. 많이 움직였으면(한 바퀴 이상) 몸통을 한 번 더.
	if _, err := f.ReadAt(hdr, 0); err == nil {
		total2 := binary.LittleEndian.Uint64(hdr[16:24])
		if total2 >= total1+uint64(capacity) {
			_, _ = f.ReadAt(body, ringHeaderSize)
		}
	}
	total := binary.LittleEndian.Uint64(hdr[16:24])
	gen := binary.LittleEndian.Uint64(hdr[24:32])

	var data []byte
	if total <= uint64(capacity) {
		data = append([]byte(nil), body[:total]...)
	} else {
		start := total % uint64(capacity)
		data = append([]byte(nil), body[start:]...)
		data = append(data, body[:start]...)
	}
	return Snapshot{Data: data, Generation: gen, Total: total}, nil
}
