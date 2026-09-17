package transcript

import "testing"

// 왕복 — 짓는 쪽과 읽는 쪽이 같은 글자를 본다.
//
// 이 시험이 이 패키지에 있는 이유가 이것 하나다. Shell 의 JSON 태그를 바꾸면
// 컴파일은 그대로 되고 봉인된 Run 을 화면에서 열 때 틀린다 — 몇 주 뒤다.
// 같은 패키지에 두 쪽이 있어야 그 갈림이 지금 빨개진다.
//
// enode.capped 는 이 왕복이 못 잡는다. 짓는 쪽이 internal/record (U3) 라
// 이 패키지에 없다. 그 절반은 사람이 맞댄다.
func TestRoundTrip_WhatTheShellWritesIsWhatTheParserReads(t *testing.T) {
	for _, tc := range []struct {
		name   string
		line   string
		kind   Kind
		tool   string
		ok     *bool
		tokens map[string]int
	}{
		{
			name:   "a tool call keeps its name and budget",
			line:   line("assistant-tool.json"),
			kind:   KindToolUse,
			tool:   "Read",
			tokens: map[string]int{"in": 10, "out": 3, "cache_read": 13551},
		},
		{
			name: "a failed tool result keeps its verdict",
			line: line("user-result.json"),
			kind: KindToolResult,
			ok:   boolOf(false),
		},
		{
			name: "an assistant that only talked keeps what it said",
			line: line("assistant-text.json"),
			kind: KindText,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			obj, typ, ok := ParseLine([]byte(tc.line))
			if !ok {
				t.Fatalf("the fixture was not read as an event")
			}
			shell := Shell(obj, typ)
			if shell == nil {
				t.Fatalf("the shell could not be built")
			}

			r := Parse(append(shell, '\n'), false)
			if len(r.Events) != 1 {
				t.Fatalf("the shell made %d events, want 1: %s", len(r.Events), shell)
			}
			e := r.Events[0]

			// ADR-071 뒤로 짓는 쪽이 message 를 실으므로 읽는 쪽이 원문
			// 줄기로 간다. 껍데기 줄기가 사라진 것이 아니다 — 옛 기록이
			// 그리로 가고, parse_test 의 두 줄기 시험이 그것을 잰다.
			if e.Shell {
				t.Fatalf("the reduced line was read as a shell: %s", shell)
			}
			if e.Kind != tc.kind {
				t.Fatalf("got kind %q, want %q — the two sides disagree: %s", e.Kind, tc.kind, shell)
			}
			if e.Name != tc.tool {
				t.Fatalf("got tool %q, want %q: %s", e.Name, tc.tool, shell)
			}
			if (e.OK == nil) != (tc.ok == nil) {
				t.Fatalf("got ok %v, want %v: %s", e.OK, tc.ok, shell)
			}
			if e.OK != nil && *e.OK != *tc.ok {
				t.Fatalf("got ok %v, want %v: %s", *e.OK, *tc.ok, shell)
			}
			for k, v := range tc.tokens {
				if e.Tokens[k] != v {
					t.Fatalf("token %q is %d, want %d: %s", k, e.Tokens[k], v, shell)
				}
			}
			// 안 나가야 할 것은 어느 쪽으로도 안 돌아온다 — 본문은 이제
			// 돌아오고 (ADR-071) 생각의 서명과 최상위 두 벌은 아니다.
			assertNoLeak(t, []byte(e.Text))
		})
	}
}

// 걷은 양도 왕복한다 — ElidedMarker 가 쓰고 Parse 가 읽는다.
func TestRoundTrip_TheElidedMarkerReadsBackAsTheAggregate(t *testing.T) {
	r := Parse(append(ElidedMarker(7, 4096), '\n'), false)
	if r.Elided == nil {
		t.Fatalf("the marker did not come back as the aggregate")
	}
	if r.Elided.Events != 7 || r.Elided.Bytes != 4096 {
		t.Fatalf("got %+v, want events 7 bytes 4096", *r.Elided)
	}
	// 집계는 사건이 아니다. 줄로는 센다.
	if len(r.Events) != 0 {
		t.Fatalf("the aggregate became %d events", len(r.Events))
	}
	if r.Lines != 1 {
		t.Fatalf("the aggregate line was not counted: %d", r.Lines)
	}
}

func boolOf(v bool) *bool { return &v }
