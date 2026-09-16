package transcript

import (
	"os"
	"path/filepath"
	"testing"
	"unicode/utf8"
)

// FuzzParse 는 대상 하나다.
//
// FuzzParseLine 과 FuzzShell 을 안 만든다 — 코퍼스가 두 벌이 되고, 두 벌이
// 되면 한쪽만 자라 커버리지가 갈린다. Parse 가 그 둘을 다 지나므로 대상
// 하나로 같은 길을 다 밟는다.
//
// CI 는 -fuzz 를 안 붙인다. 그러면 시드만 한 번씩 도는데 그것이 스킵이
// 아니다 — 시드는 실측한 모양 전부라 회귀를 그것으로 잡는다. 변이는 사람이
// 돌린다:
//
//	go test ./internal/transcript -run=XXX -fuzz=FuzzParse -fuzztime=60s
func FuzzParse(f *testing.F) {
	// 시드는 Step 9 의 픽스처 그대로다. 두 벌이 0 이다 — 픽스처를 고치면
	// 코퍼스도 같이 바뀐다.
	dir := filepath.Join("testdata", "lines")
	entries, err := os.ReadDir(dir)
	if err != nil {
		f.Fatalf("the fixture directory is missing: %s: %v", dir, err)
	}
	var all []byte
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			f.Fatalf("fixture is unreadable: %s: %v", e.Name(), err)
		}
		f.Add(b, false)
		f.Add(b, true)
		all = append(all, b...)
	}
	// 줄이 여럿인 입력도 시드로 둔다 — 붙이기와 집계는 줄 하나로는 안 돈다.
	f.Add(all, false)
	f.Add(all, true)

	f.Fuzz(func(t *testing.T, b []byte, truncated bool) {
		// F1 — 어떤 입력에도 패닉하지 않는다. 이 호출이 돌아오는 것이
		// 그 단언이다. 오류를 안 돌려주므로 남는 실패 경로가 패닉뿐이다.
		r := Parse(b, truncated)

		// F4 — 경계 산술. 음수가 아니고 입력 길이를 안 넘는다.
		if r.Head < 0 || r.Partial < 0 {
			t.Fatalf("Head=%d Partial=%d, neither may be negative", r.Head, r.Partial)
		}
		if r.Head > len(b) || r.Partial > len(b) {
			t.Fatalf("Head=%d Partial=%d exceed the %d bytes of input", r.Head, r.Partial, len(b))
		}
		if r.Head+r.Partial > len(b) {
			t.Fatalf("Head+Partial=%d exceeds the %d bytes of input", r.Head+r.Partial, len(b))
		}

		// F3 — 줄을 안 버린다. 읽은 줄은 사건을 내거나 집계다.
		aggregate := 0
		if r.Elided != nil {
			aggregate = 1
		}
		seen := map[int]bool{}
		for _, e := range r.Events {
			if e.Line < 1 || e.Line > r.Lines {
				t.Fatalf("an event claims line %d, but %d lines were read", e.Line, r.Lines)
			}
			seen[e.Line] = true
		}
		if len(seen)+aggregate != r.Lines {
			t.Fatalf("%d lines made events and %d made the aggregate, but %d were read",
				len(seen), aggregate, r.Lines)
		}

		for _, e := range r.Events {
			// 입력의 유효성과 무관하게 참인 유일한 자르기 불변식이다.
			if e.Cut < 0 || len(e.Text)+e.Cut < 0 {
				t.Fatalf("Cut=%d with len(Text)=%d", e.Cut, len(e.Text))
			}
			// F2 — 입력이 올바른 UTF-8 일 때만 자른 Text 도 올바르다.
			//
			// 조건이 붙는 이유는 tool_use 의 Text 가 json.RawMessage 를
			// compact 한 것이라, 잘못된 UTF-8 이 표준 라이브러리를 그대로
			// 지나기 때문이다. 파서가 그것을 고치면 그것이 마스킹이다.
			if e.Cut > 0 && utf8.Valid(b) && !utf8.ValidString(e.Text) {
				t.Fatalf("a cut text is not valid utf-8 though the input was: %q", e.Text)
			}
		}

		// Raw 는 세는 값이다 — Events 안의 raw 수와 같아야 한다.
		raw := 0
		for _, e := range r.Events {
			if e.Kind == KindRaw {
				raw++
			}
		}
		if raw != r.Raw {
			t.Fatalf("Raw is %d but %d events are raw", r.Raw, raw)
		}
	})
}
