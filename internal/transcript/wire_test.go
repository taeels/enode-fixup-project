package transcript_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/taeels/enode/internal/transcript"
)

// 선 위의 키 이름이 여기서 굳는다.
//
// 이 시험이 있는 이유 - as=events 가 이 구조체를 그대로 마샬해 내보낸다.
// 태그가 없으면 키가 Kind · OK 로 나가고, 있다가 지워지면 화면 셋이 같은 날
// 조용히 빈 칸을 그린다. 화면이 읽는 이름이 코드의 필드 이름과 같아야 할
// 이유가 없으므로, 그 사상을 문서가 아니라 시험이 진다.
func TestWire_TheKeysAreFrozen(t *testing.T) {
	yes := true
	ev := transcript.Event{
		Kind: transcript.KindToolResult, Sub: "x", Line: 3,
		Text: "body", Name: "Read", ID: "tu_1", OK: &yes, Cut: 7, Shell: true,
		Tokens: map[string]int{"in": 1},
		Info:   transcript.Info{Model: "m", Turns: 2, CostUSD: 1.5, Bytes: 9},
	}
	b, err := json.Marshal(ev)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	for _, k := range []string{
		`"kind":"tool_result"`, `"sub":"x"`, `"line":3`, `"text":"body"`,
		`"name":"Read"`, `"id":"tu_1"`, `"ok":true`, `"cut":7`, `"shell":true`,
		`"tokens":{"in":1}`, `"model":"m"`, `"turns":2`, `"cost_usd":1.5`, `"bytes":9`,
	} {
		if !strings.Contains(got, k) {
			t.Fatalf("the wire lost %s: %s", k, got)
		}
	}
	// 필드 이름이 새어 나가면 안 된다 - 태그가 빠진 자리를 이것이 잡는다.
	for _, k := range []string{`"Kind"`, `"OK"`, `"CostUSD"`, `"Info"`} {
		if strings.Contains(got, k) {
			t.Fatalf("a Go field name reached the wire (%s): %s", k, got)
		}
	}
}

// OK 의 셋째 값이 선 위에서도 산다.
//
// nil 은 "도구를 안 불렀다" 이고 false 는 "불렀는데 실패했다" 다. 키가 빠지는
// 것과 false 로 오는 것이 그 둘을 가른다 - 합치면 화면이 없는 실패를 그린다.
func TestWire_AbsentIsNotFalse(t *testing.T) {
	no := false
	absent, _ := json.Marshal(transcript.Event{Kind: transcript.KindText})
	failed, _ := json.Marshal(transcript.Event{Kind: transcript.KindToolResult, OK: &no})
	if strings.Contains(string(absent), `"ok"`) {
		t.Fatalf("an absent OK became a key: %s", absent)
	}
	if !strings.Contains(string(failed), `"ok":false`) {
		t.Fatalf("a false OK was dropped: %s", failed)
	}
}

// 값이 없는 Info 는 키가 통째로 빠진다 - 사건 대부분이 그 모양이다.
func TestWire_AnEmptyInfoIsNotShipped(t *testing.T) {
	b, _ := json.Marshal(transcript.Event{Kind: transcript.KindText, Line: 1})
	if strings.Contains(string(b), `"info"`) {
		t.Fatalf("an empty info rode along on every event: %s", b)
	}
}

// Result 의 키도 함께 굳는다 - as=events 의 몸통이 이것이다.
func TestWire_TheResultEnvelopeIsFrozen(t *testing.T) {
	in := `{"type":"assistant","message":{"content":[{"type":"text","text":"hi"}]}}` + "\n" +
		`{"type":"enode.elided","events":3,"bytes":40}` + "\n"
	b, err := json.Marshal(transcript.Parse([]byte(in), false))
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	for _, k := range []string{`"events":[`, `"raw":0`, `"lines":2`, `"head":0`, `"partial":0`,
		`"elided":{"events":3,"bytes":40}`} {
		if !strings.Contains(got, k) {
			t.Fatalf("the envelope lost %s: %s", k, got)
		}
	}
}
