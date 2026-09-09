package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/taeels/enode/internal/config"
)

// 관측 표면 셋의 시험이다 — GET /v1/nodes · GET /v1/runs · GET /v1/runs/{id}.
//
// 진짜 Postgres 를 쓴다. observed_at 이 DB 시계이고 assigned 의 빈 배열이
// SQL 의 coalesce 에서 나오므로, 가짜 저장소로는 재는 것이 없다.
// 서버는 기존 newServerFast 를 그대로 쓴다 — DB 헬퍼를 두 벌로 두면
// 정리 규칙이 갈린다.

// demoServer 는 데모 인스턴스로 뜬 Mediator 다. 읽기 셋이 무인증이고
// 그 셋에만 한도가 걸린다.
func demoServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv, _ := newServerFast(t, func(c *config.Config) { c.Demo = true })
	return srv
}

// obsGet 은 헤더까지 돌려준다 — do 는 본문만 내는데 이 셋은 Cache-Control 도
// 규약이다. 토큰을 붙일지도 고른다: 데모 모드의 판정이 그것이다.
func obsGet(t *testing.T, srv *httptest.Server, path string, auth bool) (int, http.Header, map[string]any) {
	t.Helper()
	req, err := http.NewRequest("GET", srv.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if auth {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, resp.Header, out
}

func reasonOf(body map[string]any) string {
	e, _ := body["error"].(map[string]any)
	s, _ := e["reason"].(string)
	return s
}

func listOf(t *testing.T, body map[string]any, key string) []any {
	t.Helper()
	v, ok := body[key].([]any)
	if !ok {
		t.Fatalf("%s is not an array but %#v — null and an empty list must not be the same bytes", key, body[key])
	}
	return v
}

// ── GET /v1/nodes ────────────────────────────────────────────────────────

// 노드가 하나도 없어도 시각은 선다. 빈 함대와 못 읽은 것이 갈려야 한다.
func TestNodes_AnEmptyFleetStillCarriesTheObservationTime(t *testing.T) {
	srv, _ := newServerFast(t)
	code, hdr, body := obsGet(t, srv, "/v1/nodes", true)
	if code != 200 {
		t.Fatalf("code=%d, want 200 — an empty fleet is a fleet", code)
	}
	if n := listOf(t, body, "nodes"); len(n) != 0 {
		t.Fatalf("nodes=%v, want an empty list", n)
	}
	if at, _ := body["observed_at"].(string); at == "" {
		t.Fatalf("observed_at is missing: %v", body)
	}
	if got := hdr.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control=%q, want no-store", got)
	}
}

// 임대는 있으면 값이고 없으면 null 이다. 빈 객체를 내면 화면이
// "임대 없음" 과 "임대가 있는데 값이 비었다" 를 못 가른다.
func TestNodes_ALeasedNodeCarriesTheLeaseAndAFreeOneCarriesNull(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "busy-box", map[string]string{"role": "x"}), nil)
	do(t, srv, "POST", "/v1/nodes", advert("n2", "free-box", map[string]string{"role": "y"}), nil)
	if code, _ := do(t, srv, "POST", "/v1/runs", oneStepRun("obs-lease", "n1"), nil); code != 201 {
		t.Fatalf("submit failed: %d", code)
	}

	_, _, body := obsGet(t, srv, "/v1/nodes", true)
	byID := map[string]map[string]any{}
	for _, n := range listOf(t, body, "nodes") {
		row := n.(map[string]any)
		byID[row["node_id"].(string)] = row
	}
	if len(byID) != 2 {
		t.Fatalf("the fleet has %d nodes, want 2: %v", len(byID), body)
	}
	lease, ok := byID["n1"]["lease"].(map[string]any)
	if !ok {
		t.Fatalf("the leased node carries no lease: %v", byID["n1"])
	}
	if lease["run_id"] != "obs-lease" || lease["not_after"] == nil {
		t.Fatalf("lease=%v, want the run that took it and when it lapses", lease)
	}
	if byID["n2"]["lease"] != nil {
		t.Fatalf("a free node carries a lease: %v", byID["n2"]["lease"])
	}
	// 키는 언제나 있다 — 값이 없으면 빈 문자열이다.
	if _, ok := byID["n2"]["draining"]; !ok {
		t.Fatalf("draining is missing from a node that is not draining: %v", byID["n2"])
	}
}

// 이 라우트는 필터가 0 이다. 여유 질의를 만들지 않겠다는 결정이
// 표면에서 눈에 보이는 자리가 여기다.
func TestNodes_AnyQueryParameterIsRefused(t *testing.T) {
	srv, _ := newServerFast(t)
	for _, q := range []string{"?free=true", "?t=1757300000", "?limit=5", "?state="} {
		code, _, body := obsGet(t, srv, "/v1/nodes"+q, true)
		if code != 400 || reasonOf(body) != "unknown query parameter" {
			t.Errorf("%s: code=%d reason=%q, want 400 and \"unknown query parameter\"",
				q, code, reasonOf(body))
		}
	}
}

// ── GET /v1/runs ─────────────────────────────────────────────────────────

// rejectedRuns 는 노드가 없는 함대에 Run 을 밀어 넣는다.
// 422 로 거절되지만 행은 남는다 — 그것이 CP9 가 재는 바로 그 행이다.
func rejectedRuns(t *testing.T, srv *httptest.Server, ids ...string) {
	t.Helper()
	for _, id := range ids {
		if code, _ := do(t, srv, "POST", "/v1/runs", oneStepRun(id, "none"), nil); code == 201 {
			t.Fatalf("%s was accepted although the fleet is empty", id)
		}
	}
}

// limit 은 넘치면 자르고 나쁘면 거절한다. 자른 것은 응답에 안 보인다 —
// 회복 수단이 없으므로 그 사실은 mcp 의 도구 설명이 진다.
func TestRuns_TheLimitIsClampedButBadValuesAreRefused(t *testing.T) {
	srv, _ := newServerFast(t)
	rejectedRuns(t, srv, "lim-1", "lim-2", "lim-3")

	for _, c := range []struct {
		query string
		code  int
		rows  int // -1 이면 안 센다
	}{
		{"", 200, 3},
		{"?limit=", 200, 3},     // 인자를 붙였다가 지운 흔적이지 값이 아니다
		{"?limit=2", 200, 2},    // 좁힌다
		{"?limit=2000", 200, 3}, // 상한으로 자르고 거절하지 않는다
		{"?limit=0", 400, -1},
		{"?limit=-1", 400, -1},
		{"?limit=abc", 400, -1},
		{"?limit=1.5", 400, -1},
	} {
		code, _, body := obsGet(t, srv, "/v1/runs"+c.query, true)
		if code != c.code {
			t.Errorf("%q: code=%d, want %d (%v)", c.query, code, c.code, body)
			continue
		}
		if c.code == 400 {
			if reasonOf(body) != "invalid limit" {
				t.Errorf("%q: reason=%q, want \"invalid limit\"", c.query, reasonOf(body))
			}
			continue
		}
		if got := len(listOf(t, body, "runs")); got != c.rows {
			t.Errorf("%q: %d rows, want %d", c.query, got, c.rows)
		}
	}
}

// since 는 표준시가 붙은 RFC 3339 만 받고 하한을 포함한다.
// 포함이 아니면 같은 초에 만들어진 Run 이 폴링 사이에 사라진다.
func TestRuns_SinceIsAnInclusiveLowerBoundInRFC3339(t *testing.T) {
	srv, _ := newServerFast(t)
	rejectedRuns(t, srv, "since-1")

	_, _, body := obsGet(t, srv, "/v1/runs", true)
	row := listOf(t, body, "runs")[0].(map[string]any)
	created, err := time.Parse(time.RFC3339Nano, row["created_at"].(string))
	if err != nil {
		t.Fatalf("created_at is not a time: %v", err)
	}

	for _, c := range []struct {
		name  string
		since string
		rows  int
	}{
		{"its own creation time is included", created.Format(time.RFC3339Nano), 1},
		{"a moment later excludes it", created.Add(time.Second).Format(time.RFC3339Nano), 0},
		{"an empty value narrows nothing", "", 1},
		// 같은 순간을 다른 표준시로 적어도 같은 순간이다.
		{"another offset for the same instant",
			created.In(time.FixedZone("KST", 9*60*60)).Format(time.RFC3339Nano), 1},
	} {
		code, _, got := obsGet(t, srv, "/v1/runs?since="+url.QueryEscape(c.since), true)
		if code != 200 {
			t.Errorf("%s: code=%d, want 200 (%v)", c.name, code, got)
			continue
		}
		if n := len(listOf(t, got, "runs")); n != c.rows {
			t.Errorf("%s: %d rows, want %d", c.name, n, c.rows)
		}
	}

	// 표준시가 없는 값을 서버 지역시로 짐작하지 않는다 — 경계가 조용히 어긋난다.
	for _, bad := range []string{"yesterday", "2026-09-08", "2026-09-08 12:00:00", "1757300000",
		"2026-09-08T12:00:00"} { // 표준시가 없는 값이다
		code, _, got := obsGet(t, srv, "/v1/runs?since="+url.QueryEscape(bad), true)
		if code != 400 || reasonOf(got) != "invalid since" {
			t.Errorf("%q: code=%d reason=%q, want 400 and \"invalid since\"", bad, code, reasonOf(got))
		}
	}
}

// state 와 work 는 찾는 값 자체라 잘라 내면 다른 것을 찾은 것이 된다.
// 그래서 상한을 넘으면 자르지 않고 거절한다 — limit 과 반대다.
func TestRuns_TheFiltersAreCappedAndUnknownParametersAreIgnored(t *testing.T) {
	srv, _ := newServerFast(t)
	rejectedRuns(t, srv, "filter-1")
	over, edge := strings.Repeat("a", 201), strings.Repeat("a", 200)

	for _, c := range []struct {
		query string
		code  int
		rows  int
	}{
		{"?state=" + over, 400, -1},
		{"?work=" + over, 400, -1},
		{"?state=" + edge, 200, 0}, // 상한 자체는 통과한다. 그런 상태가 없을 뿐이다
		{"?state=FAILED", 200, 1},
		{"?state=NOPE", 200, 0}, // 어휘를 검사하지 않는다. 빈 목록이 참이다
		{"?state=", 200, 1},     // 비운 것은 미지정이다
		{"?work=nothing", 200, 0},
		{"?free=true&nonsense=1", 200, 1}, // 모르는 인자는 무시한다
	} {
		code, _, body := obsGet(t, srv, "/v1/runs"+c.query, true)
		if code != c.code {
			t.Errorf("%q: code=%d, want %d (%v)", c.query, code, c.code, body)
			continue
		}
		if c.code == 400 {
			if reasonOf(body) != "invalid filter" {
				t.Errorf("%q: reason=%q, want \"invalid filter\"", c.query, reasonOf(body))
			}
			continue
		}
		if got := len(listOf(t, body, "runs")); got != c.rows {
			t.Errorf("%q: %d rows, want %d", c.query, got, c.rows)
		}
	}
}

// 거절된 Run 도 목록의 규약을 지킨다 — submitter 키가 있고 assigned 가
// 빈 배열이다. 데모에서 가장 흔한 실패가 이 행이고, 여기서 키가 빠지면
// 그 Run 만 화면에서 이름 없이 뜬다.
func TestRuns_EveryRowCarriesSubmitterAndAnEmptyAssigned(t *testing.T) {
	srv, _ := newServerFast(t)
	rejectedRuns(t, srv, "row-1")

	_, hdr, body := obsGet(t, srv, "/v1/runs", true)
	row := listOf(t, body, "runs")[0].(map[string]any)
	if v, ok := row["submitter"]; !ok || v != "" {
		t.Fatalf("submitter=%#v, want the key to be there and empty on a real fleet run", v)
	}
	assigned, ok := row["assigned"].([]any)
	if !ok || len(assigned) != 0 {
		t.Fatalf("assigned=%#v, want an empty array — a row on no card must not be null", row["assigned"])
	}
	for _, key := range []string{"run_id", "state", "verdict", "work_id", "created_at", "ended_at"} {
		if _, ok := row[key]; !ok {
			t.Errorf("%s is missing from the row: %v", key, row)
		}
	}
	if got := hdr.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control=%q, want no-store", got)
	}
}

// ── GET /v1/runs/{id} ────────────────────────────────────────────────────

// 무엇을 기다리는가를 낸다. 판정은 안 낸다 — 화면이 GET /v1/nodes 와 대조한다.
func TestGetRun_CarriesWhatItWaitsForAndWhetherEachStepWasChosen(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "box", map[string]string{"role": "x"}), nil)
	if code, _ := do(t, srv, "POST", "/v1/runs", oneStepRun("obs-req", "n1"), nil); code != 201 {
		t.Fatal("submit failed")
	}

	code, hdr, body := obsGet(t, srv, "/v1/runs/obs-req", true)
	if code != 200 {
		t.Fatalf("code=%d, want 200 (%v)", code, body)
	}
	want := listOf(t, body, "requires")
	if len(want) != 1 {
		t.Fatalf("requires=%v, want one entry", want)
	}
	first := want[0].(map[string]any)
	if first["as"] != "b" || first["capability"] != "agent.reason" {
		t.Fatalf("requires[0]=%v, want the alias and the capability", first)
	}
	attrs, ok := first["attrs"].(map[string]any)
	if !ok || attrs["role"] != "x" {
		t.Fatalf("the attributes are not wrapped in attrs: %v", first)
	}
	step := listOf(t, body, "steps")[0].(map[string]any)
	if _, ok := step["chosen"]; !ok {
		t.Fatalf("chosen is missing from a step: %v", step)
	}
	if got := hdr.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control=%q, want no-store", got)
	}
}

// 계약을 못 읽어도 Run 상태는 준다. 다만 조용히 빠지지 않는다 —
// requires 의 생략과 읽기 실패가 한 글자가 되면 화면은 빈 칸을 그리고
// 원인은 서버 로그에만 남는다.
func TestGetRun_SaysSoWhenTheContractCannotBeRead(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "box", map[string]string{"role": "x"}), nil)
	if code, _ := do(t, srv, "POST", "/v1/runs", oneStepRun("obs-warn", "n1"), nil); code != 201 {
		t.Fatal("submit failed")
	}
	// 살아 있는 판을 계약이 아닌 것으로 바꾼다. 제출 열은 그대로이므로
	// Run 자체는 여전히 읽힌다 — 갈리는 것은 계약을 푸는 자리뿐이다.
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv("ENODE_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx,
		`UPDATE runs SET contract_versions = '["not a contract"]'::jsonb WHERE run_id = $1`,
		"obs-warn"); err != nil {
		t.Fatal(err)
	}

	code, _, body := obsGet(t, srv, "/v1/runs/obs-warn", true)
	if code != 200 {
		t.Fatalf("code=%d, want 200 — observation must not block the lookup", code)
	}
	if _, ok := body["requires"]; ok {
		t.Fatalf("requires is present although the contract could not be read: %v", body["requires"])
	}
	warnings := listOf(t, body, "warnings")
	if len(warnings) != 1 || !strings.Contains(warnings[0].(string), "contract could not be read") {
		t.Fatalf("warnings=%v, want one line saying the contract could not be read", warnings)
	}
}

// ── POST /v1/nodes 의 정책 왕복 ──────────────────────────────────────────

// 받아 적은 값을 되돌려 보여준다. 통보이지 판정이 아니다.
// 언제나 실린다 — 생략하면 "풀렸다" 와 "이 필드를 모르는 중앙이다" 가
// 한 글자가 된다.
func TestAdvert_TheStoredDrainComesBackAndOutOfVocabularyFoldsToEmpty(t *testing.T) {
	srv, _ := newServerFast(t)
	for _, c := range []struct{ sent, want string }{
		{"graceful", "graceful"},
		{"at-boundary", "at-boundary"},
		{"", ""},
		{"whatever", ""}, // 어휘 밖은 접는다. 광고 전체를 거절하지 않는다
	} {
		body := `{"node_id":"n1","label":"box","capabilities":[],"policy":{"drain":"` + c.sent + `"}}`
		code, got := do(t, srv, "POST", "/v1/nodes", body, nil)
		if code != 200 {
			t.Fatalf("%q: code=%d, want 200 — a policy value must not cost the heartbeat", c.sent, code)
		}
		if v, ok := got["drain"]; !ok || v != c.want {
			t.Errorf("%q: drain=%#v, want %q and the key always present", c.sent, v, c.want)
		}
		// 그리고 관측이 같은 값을 낸다.
		_, _, fleet := obsGet(t, srv, "/v1/nodes", true)
		node := listOf(t, fleet, "nodes")[0].(map[string]any)
		if node["draining"] != c.want {
			t.Errorf("%q: draining=%#v, want %q", c.sent, node["draining"], c.want)
		}
	}
}

// ── 데모 모드 ────────────────────────────────────────────────────────────

// 데모 인스턴스는 읽기 셋만 연다. 쓰기와 인박스는 잠긴 채다.
func TestDemoMode_OpensTheReadsAndLeavesEverythingElseLocked(t *testing.T) {
	srv := demoServer(t)
	for _, c := range []struct {
		path string
		code int
	}{
		{"/v1/nodes", 200},
		{"/v1/runs", 200},
		{"/v1/runs/nope", 404}, // 라우트가 답했다는 뜻이다. 401 이 아니다
		{"/v1/asks", 401},      // 확정이 반쪽이라 무인증으로 낼 값이 아니다
		{"/v1/capabilities", 401},
	} {
		if code, _, body := obsGet(t, srv, c.path, false); code != c.code {
			t.Errorf("%s without a token: code=%d, want %d (%v)", c.path, code, c.code, body)
		}
	}
	req, err := http.NewRequest("POST", srv.URL+"/v1/runs", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Fatalf("a write without a token: code=%d, want 401 — writes stay locked", resp.StatusCode)
	}
}

// 기본값은 거짓이다. 실 함대는 오늘 그대로다.
func TestReads_StillNeedATokenWhenDemoModeIsOff(t *testing.T) {
	srv, _ := newServerFast(t)
	for _, p := range []string{"/v1/nodes", "/v1/runs", "/v1/runs/nope"} {
		if code, _, body := obsGet(t, srv, p, false); code != 401 {
			t.Errorf("%s without a token: code=%d, want 401 (%v)", p, code, body)
		}
	}
}

// 스위치는 환경변수 하나다. 데모 인스턴스는 컨테이너로 뜨므로
// 설정 파일을 굽지 않는다.
func TestDemoMode_ComesFromTheEnvironment(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("ENODE_MEDIATOR_CONFIG", "")
	for _, c := range []struct {
		value string
		want  bool
	}{
		{"1", true}, {"true", true}, {"YES", true},
		{"0", false}, {"maybe", false},
	} {
		t.Setenv("ENODE_DEMO_MODE", c.value)
		cfg, err := config.Load("")
		if err != nil {
			t.Fatalf("%q: %v", c.value, err)
		}
		if cfg.Demo != c.want {
			t.Errorf("ENODE_DEMO_MODE=%q gives Demo=%v, want %v", c.value, cfg.Demo, c.want)
		}
	}
}
