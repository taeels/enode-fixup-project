package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// call 은 코드와 헤더와 몸통을 그대로 낸다. do 는 JSON 만 보고 헤더를 버린다.
func call(t *testing.T, srv *httptest.Server, method, path, body string) (int, http.Header, []byte) {
	t.Helper()
	rq, err := http.NewRequest(method, srv.URL+path, bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	rq.Header.Set("Authorization", "Bearer "+token)
	rq.Header.Set("X-Enode-Principal", "taeels@gmail.com")
	resp, err := http.DefaultClient.Do(rq)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, resp.Header, b
}

// startRun 은 한 노드가 첫 단계를 집은 Run 을 만든다.
func startRun(t *testing.T, srv *httptest.Server, id string) {
	t.Helper()
	do(t, srv, "POST", "/v1/nodes", advert("n1", "box", map[string]string{"role": "x"}), nil)
	if code, _ := do(t, srv, "POST", "/v1/runs", oneStepRun(id, "n1"), nil); code/100 != 2 {
		t.Fatalf("submit failed: %d", code)
	}
	do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)
}

func jsonLine(text string) string {
	b, _ := json.Marshal(text)
	return `{"type":"assistant","message":{"content":[{"type":"text","text":` + string(b) + `}]}}` + "\n"
}

// 입력 검증이 이 층의 일이다 (SECURITY-05). 아래층은 음수를 0 으로 보고 넘어간다.
func TestGetLog_RejectsWhatItCannotRead(t *testing.T) {
	srv, _ := newServerFast(t)
	startRun(t, srv, "badin")
	for _, q := range []string{
		"/v1/runs/badin/steps/0/log",
		"/v1/runs/badin/steps/x/log",
		"/v1/runs/badin/steps/-1/log",
		"/v1/runs/badin/steps/1/log?from=-1",
		"/v1/runs/badin/steps/1/log?from=abc",
		"/v1/runs/badin/steps/1/log?as=xml",
	} {
		if code, _, _ := call(t, srv, "GET", q, ""); code != 400 {
			t.Fatalf("%s gave %d, want 400", q, code)
		}
	}
}

// 404 는 없는 Run 하나다. 없는 단계는 200 에 총 길이 0 이다.
//
// 화면은 단계 목록을 이미 들고 폴링을 건다. 틀린 seq 를 404 로 가르는 값이
// 화면에 0 이고, 갈래를 하나 더 만들면 "아직 안 왔다" 와 섞인다.
func TestGetLog_OnlyAMissingRunIs404(t *testing.T) {
	srv, _ := newServerFast(t)
	startRun(t, srv, "miss")
	if code, _, _ := call(t, srv, "GET", "/v1/runs/nosuchrun/steps/1/log", ""); code != 404 {
		t.Fatal("a missing run was not 404")
	}
	code, h, b := call(t, srv, "GET", "/v1/runs/miss/steps/99/log", "")
	if code != 200 {
		t.Fatalf("a step with no log gave %d, want 200", code)
	}
	if h.Get("X-Enode-Log-Bytes") != "0" || len(b) != 0 {
		t.Fatalf("a step with no log was not empty: %q %q", h.Get("X-Enode-Log-Bytes"), b)
	}
}

// 출처는 Sealed 한 줄이 가른다 — Run 의 상태가 아니다.
//
// 헤더가 말하는 것은 어느 파일을 읽었나다. 봉인 전에는 진행 파일의 원문이고
// 봉인 뒤에는 logs/ 의 선별본이라, 두 바이트열을 같은 것으로 읽으면 from
// 오프셋이 엉뚱한 자리를 가리킨다.
func TestGetLog_TheSourceSplitsAtTheSeal(t *testing.T) {
	srv, _ := newServerFast(t)
	startRun(t, srv, "split")

	live := jsonLine("still running")
	if code, _, _ := call(t, srv, "PUT",
		"/v1/runs/split/steps/1/log?name=build&progress=1&attempt=1", live); code != 200 {
		t.Fatal("the progress chunk was refused")
	}
	sealedBody := jsonLine("what the step said")
	call(t, srv, "PUT", "/v1/runs/split/steps/1/log?name=build", sealedBody)

	code, h, b := call(t, srv, "GET", "/v1/runs/split/steps/1/log?name=build", "")
	if code != 200 || h.Get("X-Enode-Log-Source") != "progress" {
		t.Fatalf("before the seal the source was %q", h.Get("X-Enode-Log-Source"))
	}
	if string(b) != live {
		t.Fatalf("the raw chunk did not come back: %q", b)
	}
	if h.Get("X-Enode-Log-Attempt") != "1" {
		t.Fatalf("the attempt was not reported: %q", h.Get("X-Enode-Log-Attempt"))
	}

	// Run 을 끝까지 돌려 봉인시킨다.
	do(t, srv, "POST", "/v1/runs/split/steps/1/result", `{"node":"n1","exit_code":0,"produced":["s1"]}`, nil)
	do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)
	call(t, srv, "PUT", "/v1/runs/split/steps/2/log?name=test", jsonLine("second"))
	do(t, srv, "POST", "/v1/runs/split/steps/2/result", `{"node":"n1","exit_code":0,"produced":["s2"]}`, nil)

	code, h, b = call(t, srv, "GET", "/v1/runs/split/steps/1/log?name=build", "")
	if code != 200 || h.Get("X-Enode-Log-Source") != "sealed" {
		t.Fatalf("after the seal the source was %q (code %d)", h.Get("X-Enode-Log-Source"), code)
	}
	if string(b) != sealedBody {
		t.Fatalf("the sealed log did not come back: %q", b)
	}
}

// 총 길이는 파일의 것이고 이 응답의 것이 아니다. 그리고 from 이 이어진다.
func TestGetLog_TotalIsTheFileAndFromResumes(t *testing.T) {
	srv, _ := newServerFast(t)
	startRun(t, srv, "resume")
	first, second := jsonLine("one"), jsonLine("two")
	call(t, srv, "PUT", "/v1/runs/resume/steps/1/log?name=build&progress=1&attempt=1", first)
	call(t, srv, "PUT", "/v1/runs/resume/steps/1/log?name=build&progress=1&attempt=1", second)

	_, h, b := call(t, srv, "GET", "/v1/runs/resume/steps/1/log?name=build", "")
	total, _ := strconv.ParseInt(h.Get("X-Enode-Log-Bytes"), 10, 64)
	if total != int64(len(first)+len(second)) {
		t.Fatalf("the total is %d, want %d", total, len(first)+len(second))
	}
	if string(b) != first+second {
		t.Fatalf("the body did not hold both chunks: %q", b)
	}
	// 이어 받으면 같은 바이트가 두 번 안 온다.
	_, h2, b2 := call(t, srv, "GET",
		fmt.Sprintf("/v1/runs/resume/steps/1/log?name=build&from=%d", len(first)), "")
	if string(b2) != second {
		t.Fatalf("resuming gave %q, want %q", b2, second)
	}
	if h2.Get("X-Enode-Log-Bytes") != h.Get("X-Enode-Log-Bytes") {
		t.Fatal("the total moved when only the offset did")
	}
	// 끝에서 다시 물으면 빈 본문이다. 오류가 아니다.
	_, _, b3 := call(t, srv, "GET",
		fmt.Sprintf("/v1/runs/resume/steps/1/log?name=build&from=%d", total), "")
	if len(b3) != 0 {
		t.Fatalf("polling at the end returned bytes again: %q", b3)
	}
}

// 상한에 걸린 조각은 개행에서 끝난다 — 그래야 다음 from 이 from + len(본문) 이다.
//
// 이 규칙이 없으면 as=events 의 폴링이 깨진다. 사건 배열은 읽는 쪽이 자기가
// 받은 바이트를 못 세므로, 조각이 줄 한가운데서 끝나면 다음 자리를 못 구한다.
func TestGetLog_ASlicedBodyEndsOnANewline(t *testing.T) {
	srv, _ := newServerFast(t)
	startRun(t, srv, "slice")
	// 상한(1 MiB)을 넘긴다.
	line := jsonLine(strings.Repeat("y", 900))
	var whole strings.Builder
	for whole.Len() < (1<<20)+50_000 {
		whole.WriteString(line)
	}
	call(t, srv, "PUT", "/v1/runs/slice/steps/1/log?name=build&progress=1&attempt=1", whole.String())

	var got strings.Builder
	var from int64
	slices := 0
	for i := 0; ; i++ {
		if i > 8 {
			t.Fatal("polling did not finish - the offset is not advancing")
		}
		slices++
		_, h, b := call(t, srv, "GET",
			fmt.Sprintf("/v1/runs/slice/steps/1/log?name=build&from=%d", from), "")
		total, _ := strconv.ParseInt(h.Get("X-Enode-Log-Bytes"), 10, 64)
		if int64(len(b)) > 1<<20 {
			t.Fatalf("a slice broke the cap: %d", len(b))
		}
		if from+int64(len(b)) < total && b[len(b)-1] != '\n' {
			t.Fatal("a sliced body did not end on a newline")
		}
		got.Write(b)
		from += int64(len(b))
		if from >= total {
			break
		}
	}
	if got.String() != whole.String() {
		t.Fatalf("the reassembled log differs: got %d bytes, want %d", got.Len(), whole.Len())
	}
	// 조각이 실제로 둘 이상이어야 이 시험이 무언가를 잰다. 하나면 상한이
	// 안 걸린 것이고 그때는 개행 규칙도 안 밟힌다.
	if slices < 2 {
		t.Fatalf("the cap never applied: %d slice(s)", slices)
	}
}

// as=events 로 같은 왕복을 돌아도 줄이 안 쪼개진다.
func TestGetLog_EventsSurviveTheSlicing(t *testing.T) {
	srv, _ := newServerFast(t)
	startRun(t, srv, "evslice")
	line := jsonLine(strings.Repeat("z", 900))
	var whole strings.Builder
	n := 0
	for whole.Len() < (1<<20)+50_000 {
		whole.WriteString(line)
		n++
	}
	call(t, srv, "PUT", "/v1/runs/evslice/steps/1/log?name=build&progress=1&attempt=1", whole.String())

	seen, raw, slices := 0, 0, 0
	var from int64
	for i := 0; ; i++ {
		if i > 8 {
			t.Fatal("polling did not finish")
		}
		slices++
		_, h, b := call(t, srv, "GET",
			fmt.Sprintf("/v1/runs/evslice/steps/1/log?name=build&as=events&from=%d", from), "")
		var res struct {
			Events  []map[string]any `json:"events"`
			Raw     int              `json:"raw"`
			Partial int              `json:"partial"`
			Head    int              `json:"head"`
		}
		if err := json.Unmarshal(b, &res); err != nil {
			t.Fatalf("the body was not the result envelope: %v", err)
		}
		if res.Partial != 0 || res.Head != 0 {
			t.Fatalf("a slice cut a line: head=%d partial=%d", res.Head, res.Partial)
		}
		seen += len(res.Events)
		raw += res.Raw
		total, _ := strconv.ParseInt(h.Get("X-Enode-Log-Bytes"), 10, 64)
		// 사건 배열은 바이트를 안 세므로 원문으로 한 번 더 물어 자리를 옮긴다.
		_, _, rb := call(t, srv, "GET",
			fmt.Sprintf("/v1/runs/evslice/steps/1/log?name=build&from=%d", from), "")
		from += int64(len(rb))
		if from >= total {
			break
		}
	}
	if seen != n {
		t.Fatalf("events lost across slices: got %d, want %d", seen, n)
	}
	if raw != 0 {
		t.Fatalf("%d lines fell through to raw - a line was cut", raw)
	}
	if slices < 2 {
		t.Fatalf("the cap never applied: %d slice(s)", slices)
	}
}

// PUT 의 갈래 — progress 쿼리 하나가 정한다. 라우트는 안 는다.
func TestPutLog_TheProgressBranch(t *testing.T) {
	srv, st := newServerFast(t)
	startRun(t, srv, "putp")

	code, h, _ := call(t, srv, "PUT",
		"/v1/runs/putp/steps/1/log?name=build&progress=1&attempt=2", jsonLine("chunk"))
	if code != 200 {
		t.Fatalf("the progress chunk was refused: %d", code)
	}
	if h.Get("X-Enode-Log-Attempt") != "2" {
		t.Fatalf("the attempt did not come back: %q", h.Get("X-Enode-Log-Attempt"))
	}
	if h.Get("X-Enode-Log-Bytes") == "" || h.Get("X-Enode-Log-Bytes") == "0" {
		t.Fatalf("the total was not reported: %q", h.Get("X-Enode-Log-Bytes"))
	}
	// 진행 파일은 logs/ 밖이다 — 봉인 묶음에 안 들어간다.
	if _, err := readFile(st.Records.Root + "/run-putp/logs/01-build.log"); err == nil {
		t.Fatal("the progress chunk was written into the sealed tree")
	}

	// attempt 가 없거나 규칙 밖이면 400 이다. 모르는 채로 붙이면 앞 시도를 걷는다.
	for _, q := range []string{
		"/v1/runs/putp/steps/1/log?name=build&progress=1",
		"/v1/runs/putp/steps/1/log?name=build&progress=1&attempt=0",
		"/v1/runs/putp/steps/1/log?name=build&progress=1&attempt=x",
	} {
		if code, _, _ := call(t, srv, "PUT", q, "x"); code != 400 {
			t.Fatalf("%s gave %d, want 400", q, code)
		}
	}

	// 비진행 갈래도 총 길이를 낸다 (D4).
	_, h2, _ := call(t, srv, "PUT", "/v1/runs/putp/steps/1/log?name=build", "plain\n")
	if h2.Get("X-Enode-Log-Bytes") != "6" {
		t.Fatalf("the plain branch did not report the total: %q", h2.Get("X-Enode-Log-Bytes"))
	}
}

// 봉인된 Run 에는 두 갈래 다 못 쓴다 (오용 시나리오 ⑤).
func TestPutLog_ASealedRunRefusesBothBranches(t *testing.T) {
	srv, _ := newServerFast(t)
	startRun(t, srv, "sealedput")
	do(t, srv, "POST", "/v1/runs/sealedput/steps/1/result", `{"node":"n1","exit_code":0,"produced":["s1"]}`, nil)
	do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)
	do(t, srv, "POST", "/v1/runs/sealedput/steps/2/result", `{"node":"n1","exit_code":0,"produced":["s2"]}`, nil)

	for _, q := range []string{
		"/v1/runs/sealedput/steps/1/log?name=build",
		"/v1/runs/sealedput/steps/1/log?name=build&progress=1&attempt=1",
	} {
		if code, _, _ := call(t, srv, "PUT", q, "late\n"); code != 410 {
			t.Fatalf("%s gave %d, want 410", q, code)
		}
	}
}

// 줄 한가운데의 from 은 잘린 머리로 보고된다 — 쓰레기로 안 읽는다.
//
// 폴링이 규칙대로 돌면 이 갈래를 한 번도 안 밟는다 (개행에서 끊으므로 다음
// from 이 언제나 줄 머리다). 밟는 것은 둘이다 — 사람이 손으로 from 을 줄
// 때와, 한 줄이 상한보다 길어 중간에서 끊긴 다음 폴링. 그때 서버가 앞
// 바이트를 안 보면 반쪽 줄이 raw 사건으로 화면에 그려진다.
func TestGetLog_AMidLineFromIsReportedAsACutHead(t *testing.T) {
	srv, _ := newServerFast(t)
	startRun(t, srv, "midline")
	first, second := jsonLine("aaaa"), jsonLine("bbbb")
	call(t, srv, "PUT", "/v1/runs/midline/steps/1/log?name=build&progress=1&attempt=1", first+second)

	mid := int64(len(first) / 2)
	_, _, b := call(t, srv, "GET",
		fmt.Sprintf("/v1/runs/midline/steps/1/log?name=build&as=events&from=%d", mid), "")
	var res struct {
		Events []map[string]any `json:"events"`
		Raw    int              `json:"raw"`
		Head   int              `json:"head"`
	}
	if err := json.Unmarshal(b, &res); err != nil {
		t.Fatal(err)
	}
	if res.Head == 0 {
		t.Fatal("a slice starting mid-line reported no cut head")
	}
	if res.Raw != 0 {
		t.Fatalf("%d half lines were drawn as raw events", res.Raw)
	}
	if len(res.Events) != 1 {
		t.Fatalf("the whole line after the cut did not survive: %d events", len(res.Events))
	}

	// 줄 머리에서 물으면 잘린 머리가 0 이다.
	_, _, b2 := call(t, srv, "GET",
		fmt.Sprintf("/v1/runs/midline/steps/1/log?name=build&as=events&from=%d", len(first)), "")
	res.Head, res.Raw, res.Events = 0, 0, nil
	if err := json.Unmarshal(b2, &res); err != nil {
		t.Fatal(err)
	}
	if res.Head != 0 {
		t.Fatalf("a slice starting at a line head reported a cut: %d", res.Head)
	}
}

// 한 줄이 상한보다 길면 개행 없이 끊고 그 사실이 partial 로 나간다.
//
// 여기서 0 바이트를 내면 폴링이 영영 안 나아간다. 그래서 자르되, 잘렸다는
// 것을 읽는 쪽이 알 수 있어야 한다.
func TestGetLog_ALineLongerThanTheCapStillMovesForward(t *testing.T) {
	srv, _ := newServerFast(t)
	startRun(t, srv, "longline")
	huge := jsonLine(strings.Repeat("q", (1<<20)+10_000))
	call(t, srv, "PUT", "/v1/runs/longline/steps/1/log?name=build&progress=1&attempt=1", huge)

	_, h, b := call(t, srv, "GET", "/v1/runs/longline/steps/1/log?name=build", "")
	if len(b) == 0 {
		t.Fatal("a line longer than the cap returned nothing - polling cannot advance")
	}
	if int64(len(b)) > 1<<20 {
		t.Fatalf("the cap did not apply: %d", len(b))
	}
	if b[len(b)-1] == '\n' {
		t.Fatal("the slice ended on a newline - the line was supposed to be longer than the cap")
	}
	total, _ := strconv.ParseInt(h.Get("X-Enode-Log-Bytes"), 10, 64)
	if total != int64(len(huge)) {
		t.Fatalf("total is %d, want %d", total, len(huge))
	}

	_, _, eb := call(t, srv, "GET", "/v1/runs/longline/steps/1/log?name=build&as=events", "")
	var res struct {
		Partial int `json:"partial"`
	}
	if err := json.Unmarshal(eb, &res); err != nil {
		t.Fatal(err)
	}
	if res.Partial == 0 {
		t.Fatal("the cut tail was not reported as partial")
	}
}
