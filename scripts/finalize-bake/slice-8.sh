#!/usr/bin/env bash
# finalize-bake 조각 8 (배타와 대기) — 팩의 scene-gates.md 2절 · requirements.md 6절 · bake 유닛 business-logic-model.md 8.3 ·
# 되물음 1 답 A (pending 에서 죽으면 떠 있는 형제의 광고 주기가 치운다).
#
# 사람이 보는 조각이다. 이 스크립트는 판정하지 않는다. 입력과 허용 표지는 bake-common.sh 의 머리를 본다.
#
#   export M=... T=... WS=... SIBLING_WS=... SYNC_URL=... IR=...
#   export NODE_CONFIG=<the baking node's config file>
#
# 4 는 bind 별칭의 형제로 본다 — SIBLING_WS 가 WS 의 bind 별칭이어야 한다 (mount --bind · 형제의 scratch 도 그 마운트에).
# 나머지는 형제의 모양을 가리지 않는다 (형제는 합치지 않고 정리만 한다 · 정리는 대기 자리를 그 scratch 의 trash 로 옮긴다).
#   scripts/finalize-bake/slice-8.sh
#
#   1  굽기 중 두 번째 굽기 (형제 노드에) -> 곧바로 FAILED bake_in_progress 와 주인 문장 (US-16)
#   2  pending 에서 굽는 노드를 SIGKILL -> 떠 있는 형제가 두 광고 주기 안에 정리하고 drain 이 풀린다
#   3  merge.wait 1m 굽기 + 형제에 5분 Run -> merge_wait_timeout · Record 에서 build DONE 과 merge 의 reason 이 따로
#   4  bind 별칭 두 노드가 같은 상태 자리
#   5  env check 의 not ready 사유 셋
#   6  늦게 매칭된 형제 Run 의 lower_changed — 보이면 다시 내면 된다
SLICE="slice 8"
cd "$(dirname "$0")/../.."
# shellcheck source=scripts/finalize-bake/bake-common.sh
. scripts/finalize-bake/bake-common.sh
: "${NODE_CONFIG:?set NODE_CONFIG to the config file of the baking node}"
mkdir -p "$SLICE_DIR"

check_target
check_same_lower
build_runctl

sibling_draining() {
  api "$M/v1/nodes" | jq -r --arg ws "$SIBLING_WS" '.nodes[] | ((.capabilities // []) | map(.attrs // {}) | add // {}) as $a |
    select($a.ws == $ws) | .draining // ""'
}

echo
echo "== 1. a second bake while one bakes =="
R1=slice8-first-$STAMP
bake_contract "$R1" "$WS" 4h "sleep 60" "$BUILD_B" | submit_json "$R1"
wait_phase building 3000
R2=slice8-second-$STAMP
bake_contract "$R2" "$SIBLING_WS" 4h "$BUILD_A" "$BUILD_B" | submit_json "$R2"
wait_run "$R2"
steps "$R2"
look "1: the second bake FAILED at once with reason bake_in_progress and 'another bake holds this lower: run $R1 on node <baking node> since <time> (building); submit the bake again later'."
wait_run "$R1"

echo
echo "== 2. the baking node dies while pending =="
todo "make sure both nodes are running"
hold_lower
R3=slice8-pending-$STAMP
bake_contract "$R3" "$WS" 4h "$BUILD_A" "$BUILD_B" | submit_json "$R3"
wait_phase pending 36000
PID=$(daemon_pid "$NODE_CONFIG")
kill -9 "$PID"
release_lower
T0=$(date +%s)
echo "   killed the baking daemon (pid $PID) while pending; the sibling drain is '$(sibling_draining)'"
wait_phase committed 3600
echo "   committed $(($(date +%s) - T0)) s after the kill"
state_json | jq -c '.last_attempt'
for _ in $(seq 1 600); do [ -z "$(sibling_draining)" ] && break; sleep 1; done
echo "   the sibling drain lifted $(($(date +%s) - T0)) s after the kill"
look "2: the sibling cleaned the stale pending ('bake: cleaned a stale bake' in its log) within two advert cycles (120 s by default); last_attempt is 'abandoned: no process held the bake while it was pending' for $R3; the pending directory went to the trash of its scratch; the drain lifted."
todo "start the baking node again"

echo
echo "== 3. merge.wait 1m against a 5 minute sibling run =="
R_S=slice8-sibling-$STAMP
plain_contract "$R_S" "$SIBLING_WS" 300 | submit_json "$R_S"
for _ in $(seq 1 600); do
  [ "$(api "$M/v1/runs/$R_S" | jq -r '.steps[0].phase // empty')" = running ] && break
  sleep 0.2
done
R4=slice8-wait-$STAMP
bake_contract "$R4" "$WS" 1m "$BUILD_A" "$BUILD_B" | submit_json "$R4"
wait_run "$R4"
steps "$R4"
state_json | jq -c '{phase, last_attempt}'
look "3: the build step is DONE and the merge step FAILED with reason merge_wait_timeout ('gave up waiting for the lower lock after 1m0s (merge.wait)'); the record shows the two apart (completion condition 9, US-17); the upper went to trash; the lower is committed; the drain lifts on the next advert."
wait_run "$R_S"

echo
echo "== 4. two nodes on a bind alias share one state directory =="
echo "   the sibling workspace is $(sibling_kind)"
echo "   key of WS          $(lower_key "$WS")"
echo "   key of SIBLING_WS  $(lower_key "$SIBLING_WS")"
ls -l "$(state_dir)" "$(state_dir)/holders" | sed 's/^/     /'
if same_mount; then
  echo "   SIBLING_WS is not a bind alias here; run this slice again with a bind alias sibling to see this item"
fi
look "4: with a bind alias sibling, the two keys are the same and the one state directory holds both nodes' holder records."

echo
echo "== 5. env check names what is not ready =="
echo "   run enodectl env check on a node set up in each of these ways (on a disposable lower):"
echo "     a node running as another uid than the lower's owner          lower.owner_uid"
echo "     scratch on another filesystem or mount than the workspace     binding.scratch_filesystem"
echo "     a lower.json that records another directory                   lower.identity"
todo "run the three env checks and read the not ready reasons"
look "5: each check says not ready and names the fact above."

echo
echo "== 6. a late match on the sibling =="
hold_lower
R5=slice8-late-bake-$STAMP
bake_contract "$R5" "$WS" 4h "$BUILD_A" "$BUILD_B" | submit_json "$R5"
wait_phase pending 36000
R6=slice8-late-run-$STAMP
plain_contract "$R6" "$SIBLING_WS" 1 | submit_json "$R6"
release_lower
wait_run "$R5"
wait_run "$R6"
steps "$R6"
look "6: if the sibling was matched to $R6 before the merge and ran it after, that run says lower_changed and must be submitted again; if it was matched after, it SUCCEEDED on the new lower. Both are right; a run on a changed lower without lower_changed is wrong."
echo
echo "slice 8: done; judge it by the looks above"
