package enode

import (
	"strings"
	"testing"

	"github.com/taeels/enode/internal/transcript"
)

// FR-3 의 수용 기준 — 같은 로그를 링에서 읽든 봉인된 logs/ 에서 읽든 같은
// 사건 열이 선다.
//
// 이 시험이 internal/enode 에 있는 이유는 selectLogs 가 비공개라서다.
// internal/transcript 는 internal/enode 를 임포트할 수 없다 (R9.1) — 방향이
// 하나이므로 두 경로를 한자리에서 재는 곳은 여기뿐이다.
//
// 재는 범위를 좁혀 적는다. 「전부 같다」는 거짓이고, 거짓인 수용 기준은
// 쓸 수 없다. 같은 것과 다른 것을 아래가 이름으로 가른다.
//
//	같다     Kind 의 열과 순서 · Name · OK
//	다르다   Text (껍데기에 본문이 없다) · Shell · ID (logShell 에 자리가 없다)
func TestLogsEvents_BothPathsGiveTheSameSpine(t *testing.T) {
	// 줄 하나가 사건 하나를 내는 넷만 쓴다. 왜 그런지는 아래 시험이 잰다.
	stdout := strings.Join([]string{
		initLine, assistantToolLine, userResultLine, resultLine,
	}, "\n") + "\n"

	whole := transcript.Parse([]byte(stdout), false)
	shelled := transcript.Parse(selectLogs([]byte(stdout), nil), false)

	if len(whole.Events) != len(shelled.Events) {
		t.Fatalf("the two paths made %d and %d events",
			len(whole.Events), len(shelled.Events))
	}

	for i := range whole.Events {
		a, b := whole.Events[i], shelled.Events[i]

		// 같은 것 — 척추다.
		if a.Kind != b.Kind {
			t.Fatalf("event %d is %q verbatim and %q through the shell", i, a.Kind, b.Kind)
		}
		if a.Name != b.Name {
			t.Fatalf("event %d has name %q verbatim and %q through the shell", i, a.Name, b.Name)
		}
		if (a.OK == nil) != (b.OK == nil) {
			t.Fatalf("event %d has ok %v verbatim and %v through the shell", i, a.OK, b.OK)
		}
		if a.OK != nil && *a.OK != *b.OK {
			t.Fatalf("event %d has ok %v verbatim and %v through the shell", i, *a.OK, *b.OK)
		}
	}

	// 다른 것 — 그 차이가 US-6 이 읽는 값이다.
	//
	// 걷힌 사건은 Shell 이 참이고 본문이 없다. 전문으로 남은 봉투 둘은
	// 양쪽에서 같으므로 Shell 이 거짓이다.
	for i, e := range shelled.Events {
		envelope := e.Kind == transcript.KindInit || e.Kind == transcript.KindResult
		if e.Shell == envelope {
			t.Fatalf("event %d (%s) has Shell=%v, which is backwards", i, e.Kind, e.Shell)
		}
		if e.Shell && e.Text != "" {
			t.Fatalf("a shelled event carried a body: %q", e.Text)
		}
		if e.Shell && e.ID != "" {
			t.Fatalf("a shelled event carried a tool_use_id: %q", e.ID)
		}
	}

	// 걷혔다는 것이 값으로 선다 — 화면이 「비어 있었다」와 가른다.
	if shelled.Elided == nil {
		t.Fatalf("the shelled path did not report what was taken")
	}
	if whole.Elided != nil {
		t.Fatalf("the verbatim path reported an elision that never happened")
	}
}

// 척추가 갈리는 자리 하나를 이름으로 잰다.
//
// 블록이 여럿인 assistant 줄은 원문에서 사건 여럿이고 껍데기에서 하나다 —
// logShell 에 블록 배열 자리가 없기 때문이다. 그래서 FR-3 의 수용 기준은
// 「사건 열이 같다」가 아니라 「줄 하나가 사건 하나인 줄에서 같다」다.
//
// 고칠 것이 아니라 적을 것이다. 껍데기에 블록 수를 실으면 본문의 모양이
// 새고, 그것이 허용목록이 막는 바로 그것이다.
func TestLogsEvents_AMultiBlockLineCollapsesIntoOneShell(t *testing.T) {
	stdout := assistantTextLine + "\n"

	whole := transcript.Parse([]byte(stdout), false)
	shelled := transcript.Parse(selectLogs([]byte(stdout), nil), false)

	if len(whole.Events) != 2 {
		t.Fatalf("the verbatim line made %d events, want 2 (thinking and text)", len(whole.Events))
	}
	if len(shelled.Events) != 1 {
		t.Fatalf("the shelled line made %d events, want 1", len(shelled.Events))
	}
	// 줄이 통째로 사라지지는 않는다 — 걷힌 것이 사건 하나로 선다.
	if shelled.Events[0].Kind != transcript.KindText || !shelled.Events[0].Shell {
		t.Fatalf("the collapsed event is %+v", shelled.Events[0])
	}
}
