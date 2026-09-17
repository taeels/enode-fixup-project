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

	// 그리고 본문도 같다 (ADR-071). 이것이 이 ADR 이 산 것이다 —
	// 지난 것을 여는 화면이 도는 것과 같은 글자를 그린다.
	//
	// 어느 사건도 껍데기 줄기로 안 간다. 껍데기 줄기는 이제 옛 기록 몫이다.
	for i := range whole.Events {
		a, b := whole.Events[i], shelled.Events[i]
		if a.Text != b.Text {
			t.Fatalf("event %d says %q verbatim and %q through the log", i, a.Text, b.Text)
		}
		if a.ID != b.ID {
			t.Fatalf("event %d has id %q verbatim and %q through the log", i, a.ID, b.ID)
		}
	}
	for i, e := range shelled.Events {
		if e.Shell {
			t.Fatalf("event %d (%s) was read as a shell; the body should be there", i, e.Kind)
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
// 생각 블록 하나가 사라지는 것이 그 자리 전부다 (ADR-071 2절). 원문에서
// 생각과 말 둘인 줄이 로그에서 말 하나가 된다. 나머지 블록 종류는 안 준다.
//
// 고칠 것이 아니라 적을 것이다. 생각은 본문이 비고 서명만 1,186 바이트라
// 남길 값이 없다 — 본문이 실제로 차서 오는 날 이 시험이 그 자리다.
func TestLogsEvents_TheThinkingBlockIsTheOnlyOneThatGoes(t *testing.T) {
	stdout := assistantTextLine + "\n"

	whole := transcript.Parse([]byte(stdout), false)
	shelled := transcript.Parse(selectLogs([]byte(stdout), nil), false)

	if len(whole.Events) != 2 {
		t.Fatalf("the verbatim line made %d events, want 2 (thinking and text)", len(whole.Events))
	}
	if len(shelled.Events) != 1 {
		t.Fatalf("the shelled line made %d events, want 1", len(shelled.Events))
	}
	// 남은 하나는 말이고, 그 말이 원문 그대로다.
	e := shelled.Events[0]
	if e.Kind != transcript.KindText || e.Sub == "thinking" {
		t.Fatalf("the kept event is %+v", e)
	}
	if want := whole.Events[1].Text; e.Text != want {
		t.Fatalf("the kept event says %q, want %q", e.Text, want)
	}
}
