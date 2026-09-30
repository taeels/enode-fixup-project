#!/usr/bin/env bash
# finalize-bake 조각 5 (굽기 계약 — 기계) — 팩의 scene-gates.md 2절 · bake 유닛 계획 7.1 · Step 20.
#
# 조각 5 의 시험을 이름으로 돌리고 초록인지 빨강인지 말한다. 시험 DB 를 요구한다 — ENODE_TEST_DATABASE_URL 이
# 없으면 internal/api 의 시험이 t.Skip 하고 go test 는 exit 0 이다. 그래서 목록의 시험 (하위 시험 포함) 마다 JSON 의
# pass 줄이 있어야 초록이다 — 스킵이거나 · 없거나 (-run 이 아무것도 못 맞춰도 exit 0 이다) · 실패면 빨강이다.
#
#   eval "$(scripts/testdb.sh)"
#   scripts/finalize-bake/slice-5.sh
set -uo pipefail

cd "$(dirname "$0")/../.."
if [ -z "${ENODE_TEST_DATABASE_URL:-}" ]; then
  echo "slice 5: ENODE_TEST_DATABASE_URL is unset; the api tests would skip; see scripts/testdb.sh"
  echo "slice 5: red"
  exit 1
fi
OUT=${SLICE_DIR:-${TMPDIR:-/tmp}/enode-slice-5}
mkdir -p "$OUT"

# 패키지마다 시험 이름 (하위 시험은 go test 의 이름 그대로)
declare -A want=(
  [internal/contract]="TestValidate_BakeShape/build_alone
TestGrammar_WhatItForbidsIsActuallyRejected/a_bake_has_both_steps
TestValidate_BuildStep/upper-case_name
TestValidate_BuildStep/empty_name
TestValidate_BuildStep/name_of_65
TestGrammar_WhatItForbidsIsActuallyRejected/builds_names
TestValidate_BuildStep/name_twice
TestGrammar_WhatItForbidsIsActuallyRejected/builds_names_do_not_repeat
TestMergeWait_FillsDefault
TestValidate_MergeStep/budget"
  [internal/api]="TestSubmitRejectsBadContract/bake_without_merge
TestSubmitRejectsBadContract/bake_build_name
TestSubmitRejectsBadContract/bake_build_name_twice
TestBake_NoConditionsAndASyncFailureSucceeds
TestBake_IRMismatchFailsByTheManifestCondition
TestBake_MergeUploadTimeoutFailsTheRun"
  [internal/enode]="TestContractStep_CarriesTheBakeFields
TestClaim_MergeWaitReachesTheNode
TestMergeStep_WaitsForTheContractValue
TestMergeStep_WaitsForTheContractValue/300ms
TestMergeStep_WaitsForTheContractValue/900ms"
)

red=0
for pkg in internal/contract internal/api internal/enode; do
  # 윗 시험 이름만 -run 에 싣는다 — 하위 시험은 그 안에서 돈다
  run=$(printf '%s\n' "${want[$pkg]}" | cut -d/ -f1 | sort -u | paste -sd'|')
  json=$OUT/${pkg##*/}.json
  go test -count=1 -json -run "^($run)\$" "./$pkg/" > "$json" 2>&1
  while read -r name; do
    [ -n "$name" ] || continue
    if grep -q "\"Action\":\"pass\",\"Package\":\"[^\"]*\",\"Test\":\"$name\"" "$json"; then
      echo "   pass  $pkg  $name"
    elif grep -q "\"Action\":\"skip\",\"Package\":\"[^\"]*\",\"Test\":\"$name\"" "$json"; then
      echo "   SKIP  $pkg  $name"
      red=1
    elif grep -q "\"Action\":\"fail\",\"Package\":\"[^\"]*\",\"Test\":\"$name\"" "$json"; then
      echo "   FAIL  $pkg  $name"
      red=1
    else
      echo "   NONE  $pkg  $name  (the test did not run)"
      red=1
    fi
  done <<<"${want[$pkg]}"
done
if [ "$red" != 0 ]; then
  echo "slice 5: red"
  exit 1
fi
echo "slice 5: green"
