package api

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/taeels/enode/internal/store"
	"github.com/taeels/enode/internal/transcript"
)

// GET /v1/runs/{run}/steps/{seq}/log — 진행 중인 단계의 로그를 낸다 (FR-6).
//
// 이것은 Record 가 아니다 (I4 · ADR-025 §5). GET record 는 봉인 전에 409 를
// 내고 그것은 그대로다 — 여기는 봉인 전에도 답한다. 가르는 것은 "무엇을
// 보장하느냐" 다: Record 는 불변이고 자기충족이며, 이 라우트는 지금 무슨 일이
// 일어나고 있는지만 말한다.
//
// 봉인 전후로 답의 출처가 갈린다.
//
//	봉인 전   진행 파일.  원문이다
//	봉인 뒤   logs/NN-*.log.  선별본이다.  진행 파일은 봉인 때 지워졌다
//
// 그 갈림이 헤더에 적혀 있다. 안 적으면 읽는 쪽이 두 바이트열을 같은 것으로
// 보고 from 오프셋을 그대로 이어 쓰다가 엉뚱한 자리를 읽는다.
const (
	// maxLogSliceBytes 는 한 응답이 내는 본문의 상한이다.
	//
	// 조절 손잡이가 아니라 보호다. 그래서 설정 키를 안 만든다 — 키로 두면
	// 이름과 기본값과 문서와 시험이 함께 생기고, 값을 올릴 수 있다는 것
	// 자체가 이 보호를 무르는 길이다.
	//
	// 진행 파일의 총 길이 상한(MaxBlobBytes · 기본 10 MiB)과 다른 값이고
	// 다른 단위다. 저쪽은 파일 하나가 커지는 것을 막고 이쪽은 한 요청이
	// 가져가는 양을 막는다. 화면이 여럿이면 저 천장은 곱해지고 이 천장은 안 곱해진다.
	maxLogSliceBytes = 1 << 20

	headerBytes   = "X-Enode-Log-Bytes"
	headerSource  = "X-Enode-Log-Source"
	headerAttempt = "X-Enode-Log-Attempt"
	headerCapped  = "X-Enode-Log-Capped"

	sourceProgress = "progress"
	sourceSealed   = "sealed"
)

// logSlice 는 한 응답이 실어 보낼 조각과 그 조각을 설명하는 값들이다.
type logSlice struct {
	body    []byte
	total   int64 // 파일의 총 길이. 이 조각의 길이가 아니다
	attempt int
	capped  bool
	source  string
	// headCut 은 이 조각이 줄 한가운데서 시작했다는 뜻이다. 파서가 첫 줄을
	// 버리게 하려고 넘긴다.
	headCut bool
}

// 봉인 쪽의 attempt 가 0 인 것은 값이 없어서다.
//
// logs/NN-*.log 는 이름에 시도가 없다 — 시도마다 같은 파일에 이어 붙기 때문이다
// (record.go 의 AppendLog). 그래서 봉인 뒤에는 "지금 시도" 라는 것이 아예
// 없고, 0 이 그 없음을 나른다. 화면은 출처가 sealed 인 동안 이 값으로 카드를
// 안 비운다 — 비우는 근거는 출처가 progress 인 동안의 변화다.

func (s *Server) getLog(w http.ResponseWriter, r *http.Request) {
	if !s.needRecords(w) {
		return
	}
	runID := r.PathValue("run")
	seq, err := strconv.Atoi(r.PathValue("seq"))
	if err != nil || seq <= 0 {
		fail(w, 400, "invalid step sequence")
		return
	}
	from, ok := logFrom(r)
	if !ok {
		fail(w, 400, "invalid from offset")
		return
	}
	as := r.URL.Query().Get("as")
	if as == "" {
		as = "raw"
	}
	if as != "raw" && as != "events" {
		fail(w, 400, "unknown representation")
		return
	}
	// PUT 과 같은 기본값이다 (api.go 의 putLog). 한쪽만 기본값이 있으면
	// PUT 이 쓴 것을 GET 이 못 찾는 조합이 생기고, 그 조합은 시험이 아니라
	// 운영에서 난다.
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "step"
	}

	// 404 는 이것 하나다. 없는 seq 는 404 가 아니라 200 에 총 길이 0 이다 —
	// 화면은 단계 목록을 이미 들고 폴링을 걸고, 틀린 seq 를 404 로 가르는
	// 값이 화면에 0 이다.
	if _, err := s.st.GetRun(r.Context(), runID); errors.Is(err, store.ErrNotFound) {
		fail(w, 404, "no such run")
		return
	} else if err != nil {
		fail(w, 503, "query failed")
		return
	}

	sl, err := s.logSliceOf(runID, seq, name, from)
	if err != nil {
		s.log.Error("cannot read step log", "run", runID, "seq", seq, "err", err)
		fail(w, 503, "read failed")
		return
	}

	noStore(w)
	w.Header().Set(headerBytes, strconv.FormatInt(sl.total, 10))
	w.Header().Set(headerSource, sl.source)
	w.Header().Set(headerAttempt, strconv.Itoa(sl.attempt))
	if sl.capped {
		w.Header().Set(headerCapped, "1")
	}
	if as == "events" {
		write(w, 200, transcript.Parse(sl.body, sl.headCut))
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	_, _ = w.Write(sl.body)
}

// logFrom 은 from 을 읽고 검증한다.
//
// 검증이 이 층의 일이다. internal/record 는 음수를 0 으로 보고 넘어가는데,
// 그것은 그 층의 전제를 하나로 줄이려는 것이지 잘못된 입력을 봐주려는 것이
// 아니다. 계약을 아는 쪽이 여기다.
func logFrom(r *http.Request) (int64, bool) {
	q := r.URL.Query().Get("from")
	if q == "" {
		return 0, true
	}
	n, err := strconv.ParseInt(q, 10, 64)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

// logSliceOf 는 출처를 가르고 조각을 읽는다.
//
// 가르는 것은 Sealed 한 줄이다. Run 의 상태가 아니다 — 헤더가 말하는 것은
// 어느 파일을 읽었나이고, Run 이 종료 상태인데 Seal 이 아직 안 돈 창에서는
// 진행 파일이 아직 있다 (Seal 이 DropProgress 를 가장 먼저 부른다).
func (s *Server) logSliceOf(runID string, seq int, name string, from int64) (logSlice, error) {
	// 한 바이트 앞에서 읽는다. 그 바이트가 개행인지가 곧 "이 조각이 줄
	// 경계에서 시작하는가" 이고, as=events 가 첫 줄을 버릴지를 그것으로 정한다.
	// 파일을 두 번 여는 대신 한 바이트를 더 읽는다.
	readFrom := from
	probe := from > 0
	if probe {
		readFrom--
	}

	var (
		sl  logSlice
		rc  io.ReadCloser
		err error
	)
	if s.records.Sealed(runID) {
		sl.source = sourceSealed
		var f io.ReadCloser
		var total int64
		f, total, err = s.records.OpenLog(runID, seq, name)
		if errors.Is(err, os.ErrNotExist) {
			// 그런 단계의 로그가 없다. 오류가 아니다 — 빈 본문에 총 길이 0 이다.
			return logSlice{source: sourceSealed}, nil
		}
		if err != nil {
			return logSlice{}, err
		}
		sl.total = total
		if sk, ok := f.(io.Seeker); ok && readFrom > 0 {
			if _, err := sk.Seek(readFrom, io.SeekStart); err != nil {
				f.Close() //nolint:errcheck // 이미 실패했다
				return logSlice{}, err
			}
		}
		rc = f
	} else {
		sl.source = sourceProgress
		p, body, perr := s.records.ReadProgress(runID, seq, name, readFrom)
		if perr != nil {
			return logSlice{}, perr
		}
		sl.total, sl.attempt, sl.capped = p.Total, p.Attempt, p.Capped
		rc = body
	}
	defer rc.Close() //nolint:errcheck // 읽기만 했다

	// 상한 + 1 을 읽는다. 앞의 한 바이트는 경계를 보려고 읽은 것이라 몸통에서 뺀다.
	limit := int64(maxLogSliceBytes)
	if probe {
		limit++
	}
	buf, err := io.ReadAll(io.LimitReader(rc, limit))
	if err != nil {
		return logSlice{}, err
	}
	if probe && len(buf) > 0 {
		sl.headCut = buf[0] != '\n'
		buf = buf[1:]
	}
	sl.body = cutAtNewline(buf, from+int64(len(buf)) < sl.total)
	return sl, nil
}

// cutAtNewline 은 조각이 잘렸으면 마지막 개행에서 끊는다.
//
// 이것이 없으면 상한이 폴링을 깬다. as=events 의 본문은 사건 배열이라 읽는
// 쪽이 자기가 받은 바이트를 셀 수 없고, 조각이 줄 한가운데서 끝나면 다음
// from 을 못 구한다. 개행에서 끊으면 다음 from 이 언제나 from + len(본문) 이라
// 갈래가 0 이다 — 헤더를 다섯째로 안 늘려도 닫힌다.
//
// 개행이 하나도 없으면 그대로 둔다. 한 줄이 상한보다 긴 경우이고, 거기서
// 0 바이트를 내면 폴링이 영영 안 나아간다. 그때는 Result.Partial 이 0 이
// 아니게 나가고 화면이 그 자리를 다르게 그린다.
func cutAtNewline(b []byte, truncated bool) []byte {
	if !truncated || len(b) == 0 {
		return b
	}
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] == '\n' {
			return b[:i+1]
		}
	}
	return b
}
