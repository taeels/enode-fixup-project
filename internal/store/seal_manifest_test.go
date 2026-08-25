package store

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/taeels/enode/internal/contract"
)

// manifest.contract 는 객체가 아니라 열이다 (ADR-022 §7.6 · B1)
//
// 그리고 한 판이 평평하게 읽혀야 한다 — contract[0].steps 이지
// contract[0].contract.steps 가 아니다. 임베드가 그것을 보장한다.
func TestManifest_ContractIsAColumnAndFlat(t *testing.T) {
	at := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	m := Manifest{
		RunID: "r1", Principal: "taeels@gmail.com", State: "SUCCEEDED", CreatedAt: at,
		Contract: []ContractVersion{{
			V: 1, At: at, By: ByRequester,
			Contract: contract.Contract{
				RunID: "r1",
				Steps: []contract.Step{{ID: "plan", Uses: "planner"}},
			},
		}},
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Contract []struct {
			V     int    `json:"v"`
			By    string `json:"by"`
			Steps []struct {
				ID string `json:"id"`
			} `json:"steps"` // must sit here, not one level down
		} `json:"contract"`
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("%v\n%s", err, b)
	}
	if len(got.Contract) != 1 {
		t.Fatalf("not a column: %s", b)
	}
	if got.Contract[0].V != 1 || got.Contract[0].By != ByRequester {
		t.Fatalf("v1 is not the submitted contract: %s", b)
	}
	if len(got.Contract[0].Steps) != 1 || got.Contract[0].Steps[0].ID != "plan" {
		t.Fatalf("the contract does not read flat: %s", b)
	}
}

// 길이 제약은 하한 1 뿐이다
//
// "오늘은 원소가 하나다" 를 코드에 제약으로 박으면 expands(P4)·재계획(P6)이
// 검증기부터 고쳐야 한다 — ADR-021 이 기록한 실수와 같은 모양이다.
// 이 시험은 여러 판이 그대로 실린다는 것을 고정한다.
func TestManifest_CarriesEveryVersion(t *testing.T) {
	at := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	mk := func(v int, by, ev string, cause ...string) ContractVersion {
		return ContractVersion{V: v, At: at, By: by, Evidence: ev, Cause: cause,
			Contract: contract.Contract{RunID: "r1"}}
	}
	m := Manifest{RunID: "r1", Contract: []ContractVersion{
		mk(1, ByRequester, ""),
		mk(2, ByStep("plan"), "blobs/plan-plan"),
		mk(3, ByStep("replan"), "blobs/replan-plan", "steps/02-baseline_build.json"),
	}}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Contract []struct {
			V        int      `json:"v"`
			By       string   `json:"by"`
			Evidence string   `json:"evidence"`
			Cause    []string `json:"cause"`
		} `json:"contract"`
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Contract) != 3 {
		t.Fatalf("the three versions are not carried: %s", b)
	}
	if got.Contract[1].By != "step:plan" || got.Contract[1].Evidence != "blobs/plan-plan" {
		t.Fatalf("by/evidence do not match: %s", b)
	}
	// cause 는 재계획에서만 채워진다 — 없으면 안 나가야 한다
	if len(got.Contract[1].Cause) != 0 {
		t.Fatalf("an empty cause was carried: %s", b)
	}
	if len(got.Contract[2].Cause) != 1 || got.Contract[2].Cause[0] != "steps/02-baseline_build.json" {
		t.Fatalf("cause does not match: %s", b)
	}
}
