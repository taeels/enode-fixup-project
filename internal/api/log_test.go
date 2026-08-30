package api_test

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/taeels/enode/internal/config"
)

// getRaw 는 JSON 이 아닌 응답(tar 등)을 그대로 받는다.
func getRaw(t *testing.T, srv *httptest.Server, path string) (int, http.Header, []byte) {
	t.Helper()
	req, err := http.NewRequest("GET", srv.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Enode-Principal", "taeels@gmail.com")
	resp, err := http.DefaultClient.Do(req)
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

// tarFiles 는 tar 안의 「이름 → 내용」을 뽑는다.
func tarFiles(t *testing.T, b []byte) map[string]string {
	t.Helper()
	out := map[string]string{}
	tr := tar.NewReader(bytes.NewReader(b))
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return out
		}
		if err != nil {
			t.Fatalf("the record tar is not readable: %v", err)
		}
		if h.Typeflag == tar.TypeDir {
			continue
		}
		body, err := io.ReadAll(tr)
		if err != nil {
			t.Fatalf("cannot read %s out of the tar: %v", h.Name, err)
		}
		out[h.Name] = string(body)
	}
}

// 단계가 뱉은 것이 봉인된 기록에 **원문 그대로** 남는다 (ADR-005 의 logs/).
//
// 이 한 줄이 log 표면이 존재하는 이유다. blob 과 자리가 다르다 — blob 은
// 다음 단계가 읽지만 log 는 아무도 안 읽고 기록에만 남는다. 그러므로
// "받았다" 로는 부족하고 봉인된 묶음 안에 있어야 확인된 것이다.
func TestLog_WhatTheStepSaidEndsUpInTheSealedRecord(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "box", map[string]string{"role": "x"}), nil)
	if code, _ := do(t, srv, "POST", "/v1/runs", oneStepRun("logrun", "n1"), nil); code != 201 {
		t.Fatalf("submit failed: %d", code)
	}

	const first = "  CC foo.o\n  CC bar.o\n"
	const second = "no name was given for this one\n"

	// 1단계 — 이름을 준다.
	do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)
	if code, _ := do(t, srv, "PUT", "/v1/runs/logrun/steps/1/log?name=build", first, nil); code != 204 {
		t.Fatalf("PUT log code=%d, want 204", code)
	}
	do(t, srv, "POST", "/v1/runs/logrun/steps/1/result", `{"node":"n1","exit_code":0,"produced":["s1"]}`, nil)

	// 2단계 — 이름을 안 준다. 이름 없는 로그도 사라지면 안 된다.
	do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)
	if code, _ := do(t, srv, "PUT", "/v1/runs/logrun/steps/2/log", second, nil); code != 204 {
		t.Fatalf("PUT log without a name: code=%d, want 204", code)
	}
	do(t, srv, "POST", "/v1/runs/logrun/steps/2/result", `{"node":"n1","exit_code":0,"produced":["s2"]}`, nil)

	_, run := do(t, srv, "GET", "/v1/runs/logrun", "", nil)
	if run["state"] != "SUCCEEDED" {
		t.Fatalf("state=%v, want SUCCEEDED", run["state"])
	}

	code, hdr, body := getRaw(t, srv, "/v1/runs/logrun/record")
	if code != 200 {
		t.Fatalf("GET record code=%d, want 200", code)
	}
	if ct := hdr.Get("Content-Type"); ct != "application/x-tar" {
		t.Errorf("Content-Type=%q, want %q — the record travels as one bundle", ct, "application/x-tar")
	}
	if cd := hdr.Get("Content-Disposition"); !strings.Contains(cd, "run-logrun.tar") {
		t.Errorf("Content-Disposition=%q does not name the run", cd)
	}

	files := tarFiles(t, body)
	if got := files["run-logrun/logs/01-build.log"]; got != first {
		t.Fatalf("the named log is not verbatim in the record: %q, want %q", got, first)
	}
	// 이름을 안 주면 step 이다 — 이름 없는 로그가 조용히 사라지지 않는다.
	if got := files["run-logrun/logs/02-step.log"]; got != second {
		t.Fatalf("the unnamed log did not default to step: %q, want %q\nrecord holds: %v",
			got, second, keys(files))
	}
	// 성질 4(자기충족) — 묶음 하나에 계약과 판정이 함께 있다.
	for _, want := range []string{"run-logrun/manifest.json", "run-logrun/verdict.json"} {
		if _, ok := files[want]; !ok {
			t.Errorf("%s is missing from the record: %v", want, keys(files))
		}
	}
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// 단계 번호가 번호가 아니면 400 이고, 무엇이 틀렸는지 말한다.
func TestLog_RefusesASequenceThatIsNotAStepNumber(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "box", map[string]string{"role": "x"}), nil)
	do(t, srv, "POST", "/v1/runs", oneStepRun("badseq", "n1"), nil)

	for _, seq := range []string{"abc", "0", "-1", "1.5"} {
		code, body := do(t, srv, "PUT", "/v1/runs/badseq/steps/"+seq+"/log", "x", nil)
		if code != 400 {
			t.Errorf("seq=%q: code=%d, want 400", seq, code)
		}
		if r := reason(t, body); !strings.Contains(r, "sequence") {
			t.Errorf("seq=%q: reason=%q does not say the sequence is the problem", seq, r)
		}
	}
}

// 없는 Run 에는 404 이고, 본문이 오류 계약의 모양이다.
func TestLog_UnknownRunIs404WithTheErrorShape(t *testing.T) {
	srv, _ := newServerFast(t)

	code, body := do(t, srv, "PUT", "/v1/runs/nope/steps/1/log", "x", nil)
	if code != 404 {
		t.Fatalf("code=%d, want 404", code)
	}
	assertFaultShape(t, body, 404, "no such run")
}

// 봉인된 것에는 못 쓴다 (I4) — 끝난 Run 의 기록은 더 자라지 않는다.
func TestLog_AfterTheRunFinishedIs410(t *testing.T) {
	srv, _ := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("n1", "box", map[string]string{"role": "x"}), nil)
	do(t, srv, "POST", "/v1/runs", oneStepRun("sealed", "n1"), nil)

	for seq := 1; seq <= 2; seq++ {
		do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)
		do(t, srv, "POST", fmt.Sprintf("/v1/runs/sealed/steps/%d/result", seq),
			fmt.Sprintf(`{"node":"n1","exit_code":0,"produced":["s%d"]}`, seq), nil)
	}
	_, run := do(t, srv, "GET", "/v1/runs/sealed", "", nil)
	if run["state"] != "SUCCEEDED" {
		t.Fatalf("state=%v, want SUCCEEDED", run["state"])
	}

	code, body := do(t, srv, "PUT", "/v1/runs/sealed/steps/1/log?name=late", "too late\n", nil)
	if code != 410 {
		t.Fatalf("code=%d, want 410 — a sealed record must not grow", code)
	}
	if r := reason(t, body); !strings.Contains(r, "finished") {
		t.Errorf("reason=%q does not say the run has already finished", r)
	}
	// 그리고 실제로 안 남았다.
	_, _, tarBody := getRaw(t, srv, "/v1/runs/sealed/record")
	if _, ok := tarFiles(t, tarBody)["run-sealed/logs/01-late.log"]; ok {
		t.Fatal("the refused log was written into the sealed record anyway")
	}
}

// 로그는 상한을 넘으면 **잘라서 표시**한다 — 산출물과 자리가 다르다.
//
// blob 은 잘라 저장하지 않고 413 으로 거절한다("잘린 산출물은 산출물이
// 아니다"). 로그는 아무도 안 읽고 기록에만 남으므로 반대다: 잘려도 남는
// 편이 없는 것보다 낫고, 잘렸다는 사실이 파일 안에 적힌다.
func TestLog_OversizeIsTruncatedAndMarkedNotRefused(t *testing.T) {
	const limit = 64
	srv, st := newServerFast(t, func(c *config.Config) { c.Artifacts.MaxBlobBytes = limit })
	do(t, srv, "POST", "/v1/nodes", advert("n1", "box", map[string]string{"role": "x"}), nil)
	do(t, srv, "POST", "/v1/runs", oneStepRun("bigl", "n1"), nil)
	do(t, srv, "POST", "/v1/nodes/n1/claim", "", nil)

	if code, _ := do(t, srv, "PUT", "/v1/runs/bigl/steps/1/log?name=noisy",
		strings.Repeat("x", limit*4), nil); code != 204 {
		t.Fatalf("an oversize log was refused: %d — logs truncate, they do not 413", code)
	}

	p := st.Records.Root + "/run-bigl/logs/01-noisy.log"
	b, err := readFile(p)
	if err != nil {
		t.Fatalf("nothing was written at all: %v", err)
	}
	if !strings.Contains(b, "truncated") {
		t.Fatalf("the log was cut without saying so: %q", b)
	}
	if !strings.HasPrefix(b, strings.Repeat("x", limit)) {
		t.Fatalf("the kept prefix is not the first %d bytes: %q", limit, b)
	}
}
