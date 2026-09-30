package api_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/store"
)

// 굽기 결과의 판정 (bake 유닛 · FD 흐름 8.1 의 Mediator 줄 · 되물음 2 답 B · 물음 1 답 B · 결정 54 · CG 물음 1 답 A).
// 노드는 완주했는지만 말하고 Run 의 성패는 계약의 조건이 정한다 (I3). 시험 DB 로 돈다.

// bakeContract 는 제품의 굽기 예시다 — run_id 를 바꾸고, conditions 가 거짓이면 판정 조건을 뺀다.
func bakeContract(t *testing.T, runID string, conditions bool) string {
	t.Helper()
	raw, err := contract.Example("bake")
	if err != nil {
		t.Fatal(err)
	}
	var c map[string]any
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatal(err)
	}
	c["run_id"] = runID
	c["work"] = map[string]any{"id": map[string]any{"system": "manual", "change_id": runID}, "system": "manual"}
	if !conditions {
		delete(c, "success_when")
	}
	b, _ := json.Marshal(c)
	return string(b)
}

// bakeRun 은 굽는 노드 하나에 굽기 계약을 내고 build 단계를 집는다.
func bakeRun(t *testing.T, runID string, conditions bool) (srvDo func(method, path, body string) (int, map[string]any), st *store.Store) {
	t.Helper()
	srv, st := newServerFast(t)
	do(t, srv, "POST", "/v1/nodes", advert("B1", "baker", map[string]string{"workspace.writes": "isolated"}), nil)
	if code, body := do(t, srv, "POST", "/v1/runs", bakeContract(t, runID, conditions), nil); code != 201 {
		t.Fatalf("submit = %d %v", code, body)
	}
	if code, claim := do(t, srv, "POST", "/v1/nodes/B1/claim", "", nil); code != 200 || claim["kind"] != "build" {
		t.Fatalf("claim build = %d %v", code, claim)
	}
	return func(method, path, body string) (int, map[string]any) { return do(t, srv, method, path, body, nil) }, st
}

// sealedStep 은 봉인된 Record 의 단계 결과다.
func sealedStep(t *testing.T, st *store.Store, runID, file string) store.StepResult {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(st.Records.Root, "run-"+runID, "steps", file))
	if err != nil {
		t.Fatalf("nothing was sealed: %v", err)
	}
	var s struct {
		Result store.StepResult `json:"result"`
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	return s.Result
}

// 판정 조건 없는 굽기 계약 — build 가 DONE exit 1 (sync 실패) 이고 merge 가 합칠 것 없음 DONE 이면 Run 은
// SUCCEEDED 다. 그대로 둔다 (되물음 2 답 B) — Record 의 build 칸이 알린다. lint 가 그 계약을 경고한다.
func TestBake_NoConditionsAndASyncFailureSucceeds(t *testing.T) {
	call, st := bakeRun(t, "bake-nc", false)
	build := `{"node":"B1","exit_code":1,"build":{"sync":{"name":"sync","command":"s","exit_code":1,
	  "started_at":"2026-09-27T05:00:00Z","finished_at":"2026-09-27T05:01:00Z"},"builds":[],"head":"","ir":null,
	  "pinned":null,"head_tags":null}}`
	if code, res := call("POST", "/v1/runs/bake-nc/steps/1/result", build); code != 200 || res["run_state"] != "" {
		t.Fatalf("build result = %d %v", code, res)
	}
	if code, claim := call("POST", "/v1/nodes/B1/claim", ""); code != 200 || claim["kind"] != "merge" {
		t.Fatalf("claim merge = %d %v", code, claim)
	}
	if code, res := call("POST", "/v1/runs/bake-nc/steps/2/result", `{"node":"B1"}`); code != 200 ||
		res["run_state"] != "SUCCEEDED" {
		t.Fatalf("merge result = %d %v", code, res)
	}
	if _, run := call("GET", "/v1/runs/bake-nc", ""); run["state"] != "SUCCEEDED" {
		t.Fatalf("run = %v", run)
	}
	b := sealedStep(t, st, "bake-nc", "01-build.json")
	if b.Build == nil || b.Build.Sync.ExitCode != 1 || b.Build.HeadTags != nil {
		t.Fatalf("the sealed build = %+v", b.Build)
	}
}

// 예시 계약 (build 에 produced manifest) — build 가 DONE reason ir_mismatch 이고 manifest 가 없으면 Run 은 FAILED
// 다 (물음 1 답 B · 결정 54). Record 에 reason 과 head_tags 가 남고 어휘 밖 값이 없다.
func TestBake_IRMismatchFailsByTheManifestCondition(t *testing.T) {
	call, st := bakeRun(t, "bake-ir", true)
	build := `{"node":"B1","exit_code":0,"reason":"ir_mismatch","build":{"sync":{"name":"sync","command":"s",
	  "exit_code":0,"started_at":"2026-09-27T05:00:00Z","finished_at":"2026-09-27T05:01:00Z"},"builds":[],
	  "head":"1111111111111111111111111111111111111111","ir":null,"pinned":null,"head_tags":["other-tag"]}}`
	if code, _ := call("POST", "/v1/runs/bake-ir/steps/1/result", build); code != 200 {
		t.Fatalf("build result = %d", code)
	}
	if code, claim := call("POST", "/v1/nodes/B1/claim", ""); code == 200 {
		// needs 가 build 를 따라 merge 가 온다 — 합칠 것 없음 DONE
		if claim["kind"] != "merge" {
			t.Fatalf("claim = %v", claim)
		}
		call("POST", "/v1/runs/bake-ir/steps/2/result", `{"node":"B1"}`)
	}
	if _, run := call("GET", "/v1/runs/bake-ir", ""); run["state"] != "FAILED" {
		t.Fatalf("run = %v", run)
	}
	b := sealedStep(t, st, "bake-ir", "01-build.json")
	if b.Reason != contract.ReasonIRMismatch || !reflect.DeepEqual(b.Build.HeadTags, []string{"other-tag"}) ||
		len(b.OutOfVocabulary()) != 0 {
		t.Fatalf("the sealed build = reason %q build %+v out of vocabulary %v", b.Reason, b.Build, b.OutOfVocabulary())
	}
}

// merge 단계가 합친 뒤 업로드 예산을 넘기면 FAILED upload_timeout 이고 Run 은 FAILED 다 — merge 칸과 reason 이
// Record 에 남는다 (CG 물음 1 답 A · 계획 4.1 15번). Mediator 는 막지 않는다.
func TestBake_MergeUploadTimeoutFailsTheRun(t *testing.T) {
	call, st := bakeRun(t, "bake-up", true)
	build := `{"node":"B1","exit_code":0,"produced":["manifest"],"build":{"sync":{"name":"sync","command":"s",
	  "exit_code":0,"started_at":"2026-09-27T05:00:00Z","finished_at":"2026-09-27T05:01:00Z"},"builds":[],
	  "head":"1111111111111111111111111111111111111111","ir":"your-ir-tag","pinned":null,"head_tags":["your-ir-tag"]}}`
	if code, _ := call("POST", "/v1/runs/bake-up/steps/1/result", build); code != 200 {
		t.Fatalf("build result = %d", code)
	}
	if code, claim := call("POST", "/v1/nodes/B1/claim", ""); code != 200 || claim["kind"] != "merge" {
		t.Fatalf("claim merge = %d %v", code, claim)
	}
	merge := `{"node":"B1","error":"upload budget of 3m0s exceeded","reason":"upload_timeout","upload":"timeout",
	  "merge":{"ir":"your-ir-tag","previous_ir":null,"merged_at":"2026-09-27T06:00:00Z","resumed":false,
	  "ops":{"replaced":1,"created":2,"dirs":0,"opaque":0,"whiteouts":0,"trashed":1,"attrs":0}}}`
	if code, res := call("POST", "/v1/runs/bake-up/steps/2/result", merge); code != 200 || res["run_state"] != "FAILED" {
		t.Fatalf("merge result = %d %v", code, res)
	}
	m := sealedStep(t, st, "bake-up", "02-merge.json")
	if m.Reason != contract.ReasonUploadTimeout || m.Merge == nil || m.Merge.Ops.Created != 2 ||
		m.Upload != contract.StageTimeout || len(m.OutOfVocabulary()) != 0 {
		t.Fatalf("the sealed merge = %+v", m)
	}
}
