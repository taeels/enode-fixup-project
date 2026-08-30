package enode

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// 워커의 실행 경로 — claim.go 의 「일을 받아 돌리고 보고한다」 반쪽이다.
//
// claim_test.go 는 클라이언트 반쪽(Report · GetBlob)을 이미 들고 있으므로
// 거기 얹지 않는다. 이 파일은 대응하는 원본 파일 없이 행동의 이름을 다는
// 갈래다 (elastic_test.go · pipeline_test.go 와 같은 자리).
//
// 자식 프로세스를 실제로 띄워야 하는 것들은 worker_unix_test.go 에 있다 —
// 빌드 태그로 가른다. 런타임 t.Skip 으로 가르는 안을 기각한 이유는 U10 이
// 적은 그대로다: CI 의 스킵 감시가 전 패키지를 보고, 스킵은 「테스트가 돌아
// 통과했다」와 「CI 가 초록이다」를 갈라 놓는다.

// ── 시험용 Mediator ────────────────────────────────────────────
//
// 경로마다 서버를 세우지 않고 하나로 받는다 — execute 한 번이 claim 을 뺀
// 넷(GET blob · PUT blob · PUT log · POST result)을 모두 지나기 때문이다.
// 목을 세우지 않는다: 진짜 http 왕복이고, 진짜 파일이고, 진짜 프로세스다.
type mediator struct {
	srv *httptest.Server

	mu       sync.Mutex
	results  []Result
	logs     map[string][]byte
	blobs    map[string][]byte
	gets     map[string][]byte // GET blob 이 돌려줄 본문. 없으면 404 다
	getCode  map[string]int    // 그 이름에만 따로 답할 상태코드
	putCode  map[string]int    // PUT blob 이 답할 상태코드. 없으면 200
	claims   []*Step           // claim 이 차례로 내줄 것. 비면 204 다
	instance []string          // claim 이 받은 X-Enode-Instance 값들
}

func newMediator(t *testing.T) *mediator {
	t.Helper()
	m := &mediator{
		logs: map[string][]byte{}, blobs: map[string][]byte{},
		gets: map[string][]byte{}, getCode: map[string]int{}, putCode: map[string]int{},
	}
	m.srv = httptest.NewServer(http.HandlerFunc(m.serve))
	t.Cleanup(m.srv.Close)
	return m
}

func (m *mediator) serve(w http.ResponseWriter, r *http.Request) {
	p := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	m.mu.Lock()
	defer m.mu.Unlock()
	switch {
	case len(p) == 4 && p[1] == "nodes" && p[3] == "claim":
		m.instance = append(m.instance, r.Header.Get("X-Enode-Instance"))
		if len(m.claims) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		s := m.claims[0]
		m.claims = m.claims[1:]
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(s)
	case len(p) == 5 && p[1] == "runs" && p[3] == "blob":
		name := p[4]
		if code, ok := m.getCode[name]; ok {
			w.WriteHeader(code)
			return
		}
		body, ok := m.gets[name]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(body)
	case len(p) == 6 && p[5] == "log":
		b, _ := io.ReadAll(r.Body)
		m.logs[r.URL.Query().Get("name")] = b
		w.WriteHeader(http.StatusOK)
	case len(p) == 7 && p[5] == "blob":
		name := p[6]
		b, _ := io.ReadAll(r.Body)
		if code, ok := m.putCode[name]; ok {
			w.WriteHeader(code)
			_, _ = io.WriteString(w, "schema violation in "+name)
			return
		}
		m.blobs[name] = b
		w.WriteHeader(http.StatusOK)
	case len(p) == 6 && p[5] == "result":
		var res Result
		_ = json.NewDecoder(r.Body).Decode(&res)
		m.results = append(m.results, res)
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{}`)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// only 는 보고가 정확히 하나 올라왔음을 요구하고 그것을 돌려준다.
// 둘 이상이면 재시도가 돌았다는 뜻이라 그 자체가 결함이다.
func (m *mediator) only(t *testing.T) Result {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.results) != 1 {
		t.Fatalf("want exactly one report, got %d: %+v", len(m.results), m.results)
	}
	return m.results[0]
}

func (m *mediator) reports() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.results)
}

func (m *mediator) blob(name string) ([]byte, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.blobs[name]
	return b, ok
}

func (m *mediator) logOf(name string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return string(m.logs[name])
}

func (m *mediator) enqueue(s *Step) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.claims = append(m.claims, s)
}

func (m *mediator) instances() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.instance...)
}

// ── 시험용 워커 ────────────────────────────────────────────────

func discardLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newWorker(m *mediator) *Worker {
	return &Worker{
		Client: &Client{Base: m.srv.URL, Token: "t", Principal: "p", HTTP: m.srv.Client()},
		Ident:  Identity{NodeID: "n1", Principal: "owner@example.com"},
		Held:   NewHeld(),
		Log:    discardLog(),
		// 보고 재시도를 짧게 — 이 파일의 시험은 재시도를 재는 것이 아니다.
		ReportBackoff: time.Millisecond,
	}
}

func holdLease(w *Worker) {
	w.Held.Add(Lease{RunID: "r1", Node: w.Ident.NodeID, NotAfter: time.Now().Add(time.Hour)})
}

func runStep(argv ...string) *Step {
	return &Step{RunID: "r1", Seq: 1, StepID: "r1#01", Name: "build", Kind: "run", Run: argv}
}

func agentStep(params string) *Step {
	s := &Step{RunID: "r1", Seq: 1, StepID: "r1#01", Name: "plan", Kind: "agent"}
	if params != "" {
		s.Agent = json.RawMessage(params)
	}
	return s
}

// failingCreds 는 자격증명을 못 만드는 주입 자리다 (R1).
type failingCreds struct{}

func (failingCreds) For(context.Context, RunIdentity) (map[string]string, error) {
	return nil, errors.New("no vault reachable")
}

// ── execute — 임대와 argv ──────────────────────────────────────

// 임대가 없으면 시작하지 않는다 (ADR-010).
//
// Mediator 가 죽어도 실행이 멈추는 장치가 이것이다. 이 확인이 사라지면
// 임대 없는 노드가 단계를 돌리고, 그 사이 Mediator 는 같은 노드를 새 Run 에
// 줄 수 있다 — I1 이 서 있는 전제가 무너진다.
func TestExecute_WillNotStartWithoutAValidLease(t *testing.T) {
	m := newMediator(t)
	w := newWorker(m) // 장부가 비어 있다
	w.execute(context.Background(), runStep(filepath.Join(t.TempDir(), "never-launched")))

	if n := m.reports(); n != 0 {
		t.Fatalf("ADR-010 violated: a step without a lease still reported %d times", n)
	}
}

// argv 가 비면 돌린 것이 없다 — 완주가 아니므로 exit_code 를 안 적는다.
func TestExecute_AnEmptyArgvIsReportedNotRun(t *testing.T) {
	m := newMediator(t)
	w := newWorker(m)
	holdLease(w)
	w.execute(context.Background(), runStep())

	res := m.only(t)
	if res.Error != "run step has an empty argv" {
		t.Fatalf("an empty argv was not reported as such: %q", res.Error)
	}
	if res.ExitCode != nil {
		t.Fatalf("I3 violated: nothing ran yet exit code %d was reported", *res.ExitCode)
	}
}

// 프로세스를 못 띄운 것은 완주가 아니다 — exit_code 가 아니라 error 다.
//
// 뭉뚱그리면 success_when 이 「띄우지도 못했다」를 「종료코드 -1 로 끝났다」로
// 읽는다. 판정의 재료가 거짓이 된다.
func TestExecute_ABinaryThatCannotStartIsNotCompletion(t *testing.T) {
	m := newMediator(t)
	w := newWorker(m)
	holdLease(w)
	w.execute(context.Background(), runStep(filepath.Join(t.TempDir(), "no-such-binary")))

	res := m.only(t)
	if res.ExitCode != nil {
		t.Fatalf("I3 violated: a process that never started reported exit code %d", *res.ExitCode)
	}
	if res.Error == "" {
		t.Fatal("a step that could not start reported no error at all")
	}
}

// 전송 실패는 부재가 아니다 (ADR-058).
//
// 404 는 값이고 500 은 사고다. 뭉뚱그리면 Mediator 의 일시 장애가
// 「산출물 없음」으로 위장해 단계가 잘못된 전제 위에서 돈다.
func TestExecute_ATransportFailureOnAnInputStopsTheStep(t *testing.T) {
	m := newMediator(t)
	m.getCode["qemu_log"] = http.StatusInternalServerError
	w := newWorker(m)
	holdLease(w)

	step := runStep(filepath.Join(t.TempDir(), "never-launched"))
	step.In.From = []string{"qemu_log"}
	w.execute(context.Background(), step)

	res := m.only(t)
	if !strings.Contains(res.Error, "could not be fetched") {
		t.Fatalf("ADR-058 violated: a transport failure was not reported as one: %+v", res)
	}
	if res.ExitCode != nil {
		t.Fatalf("I3 violated: the step never ran yet reported exit code %d", *res.ExitCode)
	}
}

// 워크스페이스 명세를 못 읽으면 안 돌린다 — 어느 리비전인지 모른 채
// 도는 것은 조용히 틀린 결과를 낸다.
func TestExecute_AnUnreadableWorkspaceSpecStopsTheStep(t *testing.T) {
	m := newMediator(t)
	w := newWorker(m)
	holdLease(w)

	step := runStep(filepath.Join(t.TempDir(), "never-launched"))
	step.Workspace = json.RawMessage(`{"repo":`) // 잘린 JSON
	w.execute(context.Background(), step)

	res := m.only(t)
	if !strings.HasPrefix(res.Error, "workspace: ") {
		t.Fatalf("an unreadable workspace spec was not reported as such: %+v", res)
	}
}

// 다른 저장소를 빌드하지 않는다 (ADR-017).
//
// 매칭이 이미 걸렀지만 노드가 다시 확인한다 — 어긋난 채로 돌면 조용히
// 틀린 결과가 나오고, 봉인은 그것을 옳은 것으로 기록한다.
func TestExecute_AWorkspaceFromAnotherRepositoryIsRefused(t *testing.T) {
	m := newMediator(t)
	w := newWorker(m)
	w.Local.Workspace = t.TempDir() // 이 저장소가 아니다
	holdLease(w)

	step := runStep(filepath.Join(t.TempDir(), "never-launched"))
	step.Workspace = json.RawMessage(`{"repo":"github.com/someone/else","rev":"main"}`)
	w.execute(context.Background(), step)

	res := m.only(t)
	if !strings.Contains(res.Error, "workspace repository mismatch") {
		t.Fatalf("ADR-017 violated: the step ran against the wrong repository: %+v", res)
	}
	if res.ExitCode != nil {
		t.Fatalf("I3 violated: a step that never ran reported exit code %d", *res.ExitCode)
	}
}

// ── runAgentStep — 돌리기 전에 멈추는 자리 셋 ──────────────────

// 모르는 하네스로는 안 돌린다 — 조용히 claude 로 떨어뜨리지 않는다.
//
// 계약이 요구한 것과 다른 어댑터로 돌면 Record 가 거짓을 남긴다 (ADR-019:
// capability 어휘는 하나뿐이고 구별은 전부 속성이 한다).
func TestRunAgentStep_AnUnknownHarnessIsNotSilentlyClaude(t *testing.T) {
	m := newMediator(t)
	w := newWorker(m)
	holdLease(w)
	w.execute(context.Background(), agentStep(`{"harness":"openhands"}`))

	res := m.only(t)
	if res.Error != "unknown harness: openhands" {
		t.Fatalf("ADR-019 violated: an unknown harness was not refused: %+v", res)
	}
	if res.Harness != nil {
		t.Fatalf("ADR-019 violated: a harness result exists for a harness that was never run: %+v", res.Harness)
	}
}

// agent 파라미터가 없으면 지어내지 않는다 — 없는 것은 없다고 보고한다.
func TestRunAgentStep_MissingAgentParametersAreReportedNotGuessed(t *testing.T) {
	m := newMediator(t)
	w := newWorker(m)
	holdLease(w)
	w.execute(context.Background(), agentStep(""))

	res := m.only(t)
	if res.Error != "agent parameters are missing" {
		t.Fatalf("absent agent parameters were not reported as absent: %+v", res)
	}
}

// 자격증명을 못 만들면 안 돌린다 (R1).
//
// 조용히 없는 채로 돌리면 하네스가 엉뚱한 신원으로 붙거나 알 수 없는
// 이유로 실패한다 — 어느 쪽이든 Record 가 원인을 안 말한다.
func TestRunAgentStep_WillNotRunWithoutCredentials(t *testing.T) {
	m := newMediator(t)
	w := newWorker(m)
	w.Creds = failingCreds{}
	// 실제로 띄웠다면 이 경로가 없어서 하네스 오류가 났을 것이다.
	// 자격증명 오류가 온다는 것이 곧 「띄우기 전에 멈췄다」의 증거다.
	w.Local.HarnessBin = filepath.Join(t.TempDir(), "never-launched")
	holdLease(w)
	w.execute(context.Background(), agentStep(`{}`))

	res := m.only(t)
	if !strings.HasPrefix(res.Error, "cannot prepare credentials: ") {
		t.Fatalf("R1 violated: the step ran without credentials: %+v", res)
	}
	if res.Harness != nil {
		t.Fatalf("R1 violated: the harness was launched anyway: %+v", res.Harness)
	}
}

// 자격증명 주입은 결정이지 부재가 아니다 (R1) — nil 이면 Transparent 다.
func TestCreds_TransparentIsTheDefaultAndItIsADecision(t *testing.T) {
	if _, ok := (&Worker{}).creds().(Transparent); !ok {
		t.Fatalf("R1 violated: no credentials policy at all, got %T", (&Worker{}).creds())
	}
	w := &Worker{Creds: failingCreds{}}
	if _, ok := w.creds().(failingCreds); !ok {
		t.Fatalf("the injected policy was ignored, got %T", w.creds())
	}
}

// ── claim.go 의 HTTP 경계 ─────────────────────────────────────

// 204 는 정상이다 — 할 일이 없다는 뜻이지 실패가 아니다.
func TestClaim_NoWorkIsNotAFailure(t *testing.T) {
	m := newMediator(t)
	c := &Client{Base: m.srv.URL, Token: "t", Principal: "p", HTTP: m.srv.Client()}

	if _, err := c.Claim(context.Background(), "n1"); !errors.Is(err, errNoWork) {
		t.Fatalf("204 was not read as no work: %v", err)
	}

	m.enqueue(&Step{RunID: "r7", Seq: 3, StepID: "r7#03", Name: "build", Kind: "run"})
	s, err := c.Claim(context.Background(), "n1")
	if err != nil {
		t.Fatalf("a step was on offer yet claim failed: %v", err)
	}
	if s.RunID != "r7" || s.Seq != 3 || s.Name != "build" {
		t.Fatalf("the step did not survive the round trip: %+v", s)
	}
}

// 그 밖의 상태코드는 거절이다 — 204 와 뭉뚱그리면 워커가 조용히 논다.
func TestClaim_ARejectionIsAnErrorNotSilence(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()
	c := &Client{Base: srv.URL, HTTP: srv.Client()}

	_, err := c.Claim(context.Background(), "n1")
	if err == nil || errors.Is(err, errNoWork) {
		t.Fatalf("403 was not reported as a rejection: %v", err)
	}
}

// 같은 생이 다시 물으면 들고 있던 것을 돌려받는다 (ADR-030).
//
// 그 판정의 재료가 이 헤더다. 생이 없는데 실으면 다른 생의 재전달을
// 같은 생으로 오인하고, 완주한 단계가 두 번 돈다.
func TestClaim_TheInstanceHeaderRidesOnlyWhenThereIsOne(t *testing.T) {
	m := newMediator(t)
	bare := &Client{Base: m.srv.URL, HTTP: m.srv.Client()}
	_, _ = bare.Claim(context.Background(), "n1")

	marked := &Client{Base: m.srv.URL, HTTP: m.srv.Client(), Instance: "life-2"}
	_, _ = marked.Claim(context.Background(), "n1")

	got := m.instances()
	if len(got) != 2 || got[0] != "" || got[1] != "life-2" {
		t.Fatalf("ADR-030 violated: the instance header rode as %q", got)
	}
}

// 롱폴은 전용 클라이언트로 건다 (ADR-029).
//
// 짧은 타임아웃으로 걸면 그 시간에 끊고, 끊는 순간 서버가 집으면 응답이
// 유실되어 그 단계를 아무도 안 돌린다 — claim 은 비멱등이라 재시도가
// 되찾지도 못한다. 어느 클라이언트를 썼는지를 전송 계층에서 표시해 잰다.
func TestClaim_TheLongPollClientIsUsedWhenThereIsOne(t *testing.T) {
	m := newMediator(t)

	var viaPoll, viaPlain atomic.Bool
	poll := &http.Client{Transport: marking{base: m.srv.Client().Transport, used: &viaPoll}}
	plain := &http.Client{Transport: marking{base: m.srv.Client().Transport, used: &viaPlain}}

	c := &Client{Base: m.srv.URL, HTTP: plain, Poll: poll}
	_, _ = c.Claim(context.Background(), "n1")
	if !viaPoll.Load() || viaPlain.Load() {
		t.Fatalf("ADR-029 violated: claim did not use the long-poll client (poll=%v plain=%v)",
			viaPoll.Load(), viaPlain.Load())
	}

	// Poll 이 없으면 동작이 달라지면 안 된다 — 일반 것으로 돈다.
	viaPoll.Store(false)
	c.Poll = nil
	_, _ = c.Claim(context.Background(), "n1")
	if !viaPlain.Load() {
		t.Fatal("ADR-029 violated: without a long-poll client claim went nowhere")
	}
}

// marking 은 어느 클라이언트가 요청을 냈는지 표시한다.
type marking struct {
	base http.RoundTripper
	used *atomic.Bool
}

func (t marking) RoundTrip(r *http.Request) (*http.Response, error) {
	t.used.Store(true)
	return t.base.RoundTrip(r)
}

// 로그는 원문 그대로 올라간다 (ADR-005 의 logs/) — 단계 이름이 그 열쇠다.
func TestUploadLog_TheBodyRidesVerbatimUnderTheStepName(t *testing.T) {
	m := newMediator(t)
	c := &Client{Base: m.srv.URL, Token: "t", Principal: "p", HTTP: m.srv.Client()}

	body := "make: *** [Makefile:1: all] Error 2\n"
	if err := c.UploadLog(context.Background(), "r1", 2, "build", []byte(body)); err != nil {
		t.Fatalf("the log did not land: %v", err)
	}
	if got := m.logOf("build"); got != body {
		t.Fatalf("the log was not kept verbatim:\ngot:  %q\nwant: %q", got, body)
	}
}

// 로그 업로드의 거절은 삼키지 않는다 — 부르는 쪽이 경고를 남길 수 있어야 한다.
func TestUploadLog_ARejectionIsReported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := &Client{Base: srv.URL, HTTP: srv.Client()}

	if err := c.UploadLog(context.Background(), "r1", 2, "build", []byte("x")); err == nil {
		t.Fatal("a rejected log upload was reported as success")
	}
}

// 422 는 스키마 위반이고 그 이유가 오류에 실린다 (ADR-020).
//
// 이유를 안 실으면 사람이 「거절됐다」만 보고 무엇을 어겼는지 모른다.
func TestPutBlob_A422CarriesTheReasonIntoTheError(t *testing.T) {
	m := newMediator(t)
	m.putCode["plan.json"] = http.StatusUnprocessableEntity
	c := &Client{Base: m.srv.URL, Token: "t", Principal: "p", HTTP: m.srv.Client()}

	err := c.PutBlob(context.Background(), "r1", 1, "plan.json", []byte(`{}`))
	if err == nil {
		t.Fatal("ADR-020 violated: a 422 was reported as a successful upload")
	}
	if !strings.Contains(err.Error(), "schema violation in plan.json") {
		t.Fatalf("ADR-020 violated: the reason did not reach the caller: %v", err)
	}
	if _, ok := m.blob("plan.json"); ok {
		t.Fatal("the rejected blob was stored anyway")
	}
}

// 올라간 것만 produced 다 — 저장이 성공해야 이름이 실린다.
func TestPutBlob_ASuccessfulUploadKeepsTheBodyIntact(t *testing.T) {
	m := newMediator(t)
	c := &Client{Base: m.srv.URL, Token: "t", Principal: "p", HTTP: m.srv.Client()}

	body := []byte("zImage bytes\x00\x01")
	if err := c.PutBlob(context.Background(), "r1", 1, "artifact", body); err != nil {
		t.Fatalf("the blob did not land: %v", err)
	}
	got, ok := m.blob("artifact")
	if !ok || string(got) != string(body) {
		t.Fatalf("the blob was altered on the way: %q", got)
	}
}

// ── uploadProduced — ④수확 ────────────────────────────────────

// 어댑터가 남긴 것은 산출물이 아니다 — .enode- 접두는 걷지 않는다.
func TestUploadProduced_AdapterLeavingsAreNotArtifacts(t *testing.T) {
	m := newMediator(t)
	w := newWorker(m)
	out := t.TempDir()
	writeFile(t, filepath.Join(out, ".enode-prompt.md"), "what we asked")
	writeFile(t, filepath.Join(out, "artifact"), "the real thing")

	got := w.uploadProduced(context.Background(), runStep(), out, Stamp{}, discardLog())

	if len(got) != 1 || got[0] != "artifact" {
		t.Fatalf("the adapter's own leavings were harvested as artifacts: %v", got)
	}
	if _, ok := m.blob(".enode-prompt.md"); ok {
		t.Fatal("the prompt file was uploaded as an artifact")
	}
}

// 어긴 산출물은 산출물이 아니다 (ADR-020).
//
// 422 로 거절된 것은 저장되지 않았으므로 그 이름이 produced 에 들어가면
// 다음 단계가 없는 파일을 가리키게 된다.
func TestUploadProduced_ARejectedNameIsNotListedAsProduced(t *testing.T) {
	m := newMediator(t)
	m.putCode["plan.json"] = http.StatusUnprocessableEntity
	w := newWorker(m)
	out := t.TempDir()
	writeFile(t, filepath.Join(out, "plan.json"), `{"steps":[]}`)
	writeFile(t, filepath.Join(out, "notes.txt"), "kept")

	got := w.uploadProduced(context.Background(), runStep(), out, Stamp{}, discardLog())

	for _, n := range got {
		if n == "plan.json" {
			t.Fatalf("ADR-020 violated: a rejected artifact was listed as produced: %v", got)
		}
	}
	if len(got) != 1 || got[0] != "notes.txt" {
		t.Fatalf("the accepted artifact did not survive: %v", got)
	}
}

// collect 가 못 걷어도 단계를 안 죽인다 — 판정은 success_when 이 한다
// (ADR-004 · I3). 대신 왜 못 걷었는지를 기록이 스스로 적는다.
func TestUploadProduced_ACollectFailureIsWrittenDownNotThrown(t *testing.T) {
	m := newMediator(t)
	w := newWorker(m)
	w.Local.Workspace = t.TempDir()
	out := t.TempDir()

	step := runStep()
	step.Collect = map[string]string{"artifact": "arch/arm/boot/zImage"}
	step.Out = []string{"artifact"}

	got := w.uploadProduced(context.Background(), step, out, Stamp{}, discardLog())

	note, ok := m.blob(changedName)
	if !ok {
		t.Fatalf("I3 violated: nothing was written down about the failed collect; produced=%v", got)
	}
	if !strings.Contains(string(note), "collect could not gather") ||
		!strings.Contains(string(note), "no file matches") {
		t.Fatalf("the note does not say why collect failed:\n%s", note)
	}
	if !strings.Contains(string(note), "required by the contract but missing from $OUT: artifact") {
		t.Fatalf("the note does not say what the contract wanted:\n%s", note)
	}
}

// ── 워커의 수명 ────────────────────────────────────────────────

// Run 은 일을 집어 장부에 올리고 돌린다 — claim 응답의 임대를 즉시 반영한다.
//
// 다음 하트비트를 기다리면 그 사이 단계를 못 시작한다 (Held.Add).
func TestRun_ClaimsAndRecordsTheLeaseBeforeExecuting(t *testing.T) {
	m := newMediator(t)
	w := newWorker(m)
	step := runStep() // argv 가 비어 있다 — 즉시 보고하고 끝난다
	step.Lease = Lease{RunID: "r1", Node: "n1", NotAfter: time.Now().Add(time.Hour)}
	m.enqueue(step)

	// 시간이 아니라 상태로 끝낸다 — 보고가 하나 올라오면 ctx 를 끝낸다
	// (elastic_test.go 의 관행).
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		for m.reports() == 0 {
			time.Sleep(time.Millisecond)
		}
		cancel()
	}()
	w.Run(ctx)

	if !w.Held.EverHeld() {
		t.Fatal("ADR-016 violated: the claimed lease never reached the ledger")
	}
	if _, ok := w.Held.Valid("r1"); !ok {
		t.Fatal("ADR-010 violated: the lease claim brought is not valid for the step it came with")
	}
	if got := m.only(t).Error; got != "run step has an empty argv" {
		t.Fatalf("the claimed step was not executed: %q", got)
	}
}

// 한 단계의 패닉이 노드를 죽이지 않는다.
//
// 죽으면 그 노드가 든 다른 임대까지 갱신이 끊겨 무관한 Run 이 회수된다
// (ADR-016 — 하트비트가 임대를 나른다). 그래서 패닉도 보고로 나간다.
func TestSafeExecute_APanicIsReportedAndTheNodeSurvives(t *testing.T) {
	m := newMediator(t)
	w := newWorker(m)
	// 장부 자체가 없으면 execute 의 첫 확인에서 패닉한다. 어댑터 안의
	// 어떤 패닉이든 같은 자리로 떨어지므로 이 하나가 그 문을 잰다.
	w.Held = nil

	w.safeExecute(context.Background(), &Step{RunID: "r1", Seq: 1, StepID: "r1#01"})

	res := m.only(t)
	if !strings.HasPrefix(res.Error, "adapter panic: ") {
		t.Fatalf("ADR-016 violated: a panicking step reported %q instead of an adapter panic", res.Error)
	}
}

// 계획 단계만 계획 산출물의 이름을 갖는다 (ADR-046).
//
// 훅은 이 값이 있을 때만 계획을 검사한다 — 모르는 것으로 막지 않는다.
func TestPlanOutName_OnlyAnExpandingStepWithOneOutputHasOne(t *testing.T) {
	for _, tc := range []struct {
		name string
		step *Step
		want string
	}{
		{"no step at all", nil, ""},
		{"not an expanding step", &Step{Out: []string{"plan.json"}}, ""},
		{"expanding with two outputs", &Step{Expands: true, Out: []string{"a", "b"}}, ""},
		{"expanding with none", &Step{Expands: true}, ""},
		{"expanding with exactly one", &Step{Expands: true, Out: []string{"plan.json"}}, "plan.json"},
	} {
		if got := planOutName(tc.step); got != tc.want {
			t.Fatalf("ADR-046 violated: %s gave %q, want %q", tc.name, got, tc.want)
		}
	}
}

// writeFile 은 시험 자료를 놓는다. t.TempDir() 만 쓴다 — execute 가 $IN 을
// 0555 로 잠그므로 정리를 testing 이 지게 한다.
func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
