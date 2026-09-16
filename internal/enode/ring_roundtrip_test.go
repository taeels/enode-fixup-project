package enode

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/taeels/enode/internal/transcript"
)

// 링에 쓴 것을 파서가 읽는다 — 짓는 쪽과 읽는 쪽이 다른 패키지라 여기서만 잡힌다.
//
// 이 시험이 여기 있는 이유가 W-a 에서 배운 것이다. 한 패키지 안에서 짓고 읽는
// 값은 그 패키지의 왕복 시험이 잡지만, 링(internal/enode)과 파서
// (internal/transcript)처럼 갈라져 있으면 양쪽이 각자 초록인 채로 갈릴 수 있다.
// 틀리는 순간은 제어판을 여는 날이고 그때는 원인이 셋으로 갈린다.
//
// 감긴 경우가 핵심이다 — 링은 줄 경계를 모르고 감으므로 첫 줄이 반쪽이 된다.
// 파서가 그것을 읽을 수 있어야 CB1 의 "첫 줄이 잘린 채로도 파싱된다" 가 선다.
func TestRingRoundTrip_TheParserReadsWhatTheRingWrote(t *testing.T) {
	line := func(i int) string {
		return fmt.Sprintf(
			`{"type":"assistant","message":{"content":[{"type":"text","text":"line %04d"}]}}`, i) + "\n"
	}

	for _, tc := range []struct {
		name      string
		capacity  int
		lines     int
		wantWrap  bool
		wantEvent bool
	}{
		{"안 감긴다", 64 << 10, 40, false, true},
		{"감긴다", 8 << 10, 400, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "node.transcript")
			r, err := OpenRing(path, tc.capacity)
			if err != nil {
				t.Fatal(err)
			}
			var whole strings.Builder
			for i := 0; i < tc.lines; i++ {
				s := line(i)
				whole.WriteString(s)
				if _, err := r.Write([]byte(s)); err != nil {
					t.Fatal(err)
				}
			}
			if err := r.Close(); err != nil {
				t.Fatal(err)
			}

			snap, err := ReadRing(path)
			if err != nil {
				t.Fatal(err)
			}
			// 제어판이 쓰는 식이다. 스냅숏 안에 용량이 안 실려 있으므로
			// 「흘린 총량이 남은 바이트보다 크다」가 곧 감겼다는 뜻이다.
			truncated := snap.Total > uint64(len(snap.Data))
			if truncated != tc.wantWrap {
				t.Fatalf("wrap detection is wrong: total=%d data=%d", snap.Total, len(snap.Data))
			}

			res := transcript.Parse(snap.Data, truncated)
			if len(res.Events) == 0 {
				t.Fatal("the parser got nothing out of the ring")
			}
			if truncated && res.Head == 0 {
				t.Fatal("a wrapped ring reported no cut head - the half line was read as a whole one")
			}
			if !truncated && res.Head != 0 {
				t.Fatalf("an unwrapped ring reported a cut head: %d", res.Head)
			}
			// 마지막 줄은 언제나 온전하다 — 감기는 것은 머리다.
			last := res.Events[len(res.Events)-1]
			if last.Kind != transcript.KindText ||
				!strings.Contains(last.Text, fmt.Sprintf("line %04d", tc.lines-1)) {
				t.Fatalf("the newest line did not survive: %+v", last)
			}
			// raw 로 떨어진 줄이 0 이다 — 반쪽 줄을 사건으로 세면 화면이
			// 깨진 JSON 을 원문으로 그린다.
			if res.Raw != 0 {
				t.Fatalf("%d lines fell through to raw", res.Raw)
			}
		})
	}
}
