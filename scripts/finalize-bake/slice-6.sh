#!/usr/bin/env bash
# finalize-bake 조각 6 (굽기가 끝까지 돈다) — 팩의 scene-gates.md 2절 장면 2 · bake 유닛 business-logic-model.md 8.3 ·
# 계획 7.1 · CG 물음 4 답 (「몇 초」 는 배타를 잡은 때부터 committed 까지).
#
# 사람이 보는 조각이다. 이 스크립트는 판정하지 않는다 — 순서대로 돌리고 무엇을 볼지 출력한 뒤 Enter 를 기다린다.
# 코드 시험이 초록이라는 것으로 이 조각을 대신하지 않는다 (US-14). 입력과 허용 표지는 bake-common.sh 의 머리를 본다.
#
#   export M=http://<mediator>:8080 T=<bootstrap token>
#   export WS=<baking node workspace> SIBLING_WS=<sibling workspace: a bind alias of WS>
#   export SYNC_URL=<git repository> IR=<a tag in it>
#   export BUILD_A='<your build command for config-a>' BUILD_B='<your build command for config-b>'   # 비우면 짧은 기본값
#   export SIBLING_LOG=<the sibling node's log file>   # 장면 2 의 4 의 released 시각.  비우면 사람이 적는다
#   scripts/finalize-bake/slice-6.sh
#
#   1      빈 lower 에 굽기 A — build · merge DONE · metadata 의 칸 · bake.run · bake.node
#   2 · 3  형제에 긴 Run 을 낸 뒤 굽기 B — build 는 곧바로 · merge 는 기다린다 · 단계 로그에 쥔 쪽 줄과 남은 시간 ·
#          형제는 draining
#   4      형제 Run 이 끝난 때 · 형제의 released · merge 의 took · committed 의 네 시각과 기대 창
#   5      두 노드가 새 ir 과 repo.built.<이름> 을 광고한다
#   6      합치기 전 merged view 목록 == 합친 lower 목록 (.enode-metadata.json 은 뺀다)
#   7      빌드 하나를 일부러 실패시킨 굽기 · sync 가 IR 에 닿지 않는 굽기 — build DONE · merge 는 합칠 것 없음 · Run FAILED
#
# 환경 변수 — SIBLING_SECONDS (형제 Run 의 길이 · 기본 300) · IR_BAD (없는 태그 · 기본은 시각이 든 이름)
SLICE="slice 6"
cd "$(dirname "$0")/../.."
# shellcheck source=scripts/finalize-bake/bake-common.sh
. scripts/finalize-bake/bake-common.sh
SIBLING_SECONDS=${SIBLING_SECONDS:-300}
IR_BAD=${IR_BAD:-enode-slice-no-such-tag-$STAMP}
mkdir -p "$SLICE_DIR"

check_target
check_same_lower
build_runctl

# seconds 는 Go 의 time.Duration 글자 (1h2m3.5s) 를 초로 푼다.
seconds() { awk -v d="$1" 'BEGIN { s = 0; while (match(d, /[0-9.]+(h|ms|m|s)/)) { t = substr(d, RSTART, RLENGTH);
  v = t + 0; u = t; sub(/[0-9.]+/, "", u); s += (u == "h" ? v * 3600 : u == "m" ? v * 60 : u == "ms" ? v / 1000 : v);
  d = substr(d, RSTART + RLENGTH) } printf "%.3f", s }'; }
epoch() { date -d "$1" +%s.%N; }
at() { date -u -d "@$1" +%Y-%m-%dT%H:%M:%S.%3NZ; }

echo
echo "== 1. bake A on this lower =="
R_A=slice6-a-$STAMP
bake_contract "$R_A" "$WS" 4h "$BUILD_A" "$BUILD_B" | submit_json "$R_A"
wait_run "$R_A"
steps "$R_A"
state_json | jq -c '{phase, since, last_attempt}'
metadata
look "1: build and merge are DONE and the run SUCCEEDED. .enode-metadata.json has every field (source url, branch, repo_id, head, ir, pinned, sync_command, synced_at, builds, environment, workspace_target, bake run, node, merged_at, resumed, previous_ir); bake.run is $R_A and bake.node is the baking node. First checks: the build reached pending, so the rootfs has bash and git and git in the session accepted the lower's owner (no safe.directory refusal)."

echo
echo "== 2, 3. a long run on the sibling, then bake B =="
R_S=slice6-sibling-$STAMP
plain_contract "$R_S" "$SIBLING_WS" "$SIBLING_SECONDS" | submit_json "$R_S"
for _ in $(seq 1 600); do
  [ "$(api "$M/v1/runs/$R_S" | jq -r '.steps[0].phase // empty')" = running ] && break
  sleep 0.2
done
R_B=slice6-b-$STAMP
bake_contract "$R_B" "$WS" 4h "$BUILD_A" "$BUILD_B" | submit_json "$R_B"
for _ in $(seq 1 6000); do
  step_log "$R_B" 2 | grep -q "waiting for the lower lock" && break
  sleep 0.2
done
UPPER=$(state_json | jq -r '.pending_upper')
merged_view "$UPPER" > "$SLICE_DIR/$R_B.before"
echo "   merge step log so far:"
step_log "$R_B" 2 | sed 's/^/     /'
echo "   nodes:"
api "$M/v1/nodes" | jq -c '.nodes[] | {node: .node_id, draining}' | sed 's/^/     /'
look "2, 3: the build step ended at once; the merge step waits and its log names the sibling node holding the lower for run $R_S, with a deadline in node clock and the time left (completion condition 5); the sibling is draining."

echo
echo "== 4. four times (after the sibling run ends) =="
wait_run "$R_B"
steps "$R_S"
S_END=$(jq -r '.ended_at' "$SLICE_DIR/$R_S"/*/steps/01-*.json)
steps "$R_B"
M_START=$(jq -r '.started_at' "$SLICE_DIR/$R_B"/*/steps/02-*.json)
TOOK=$(step_log "$R_B" 2 | sed -n 's/^took the lower lock after //p' | tail -1)
COMMITTED=$(state_json | jq -r '.since')
MERGED_AT=$(jq -r '.bake.merged_at' "$WS/.enode-metadata.json")
RELEASED=""
if [ -n "${SIBLING_LOG:-}" ] && [ -r "$SIBLING_LOG" ]; then
  RELEASED=$(grep 'released the lower lock' "$SIBLING_LOG" | tail -1 | sed -n 's/^time=\([^ ]*\) .*/\1/p')
fi
echo "   the sibling run ended          $S_END"
echo "   the sibling released the lock  ${RELEASED:-(read it in the sibling node log: released the lower lock)}"
if [ -n "$TOOK" ]; then
  T_TOOK=$(awk -v a="$(epoch "$M_START")" -v b="$(seconds "$TOOK")" 'BEGIN { printf "%.3f", a + b }')
  echo "   the merge took the lower lock  $(at "$T_TOOK")   (merge started $M_START + $TOOK)"
  echo "   committed                      $COMMITTED   (metadata merged_at $MERGED_AT)"
  echo "   took -> committed              $(awk -v a="$T_TOOK" -v b="$(epoch "$MERGED_AT")" 'BEGIN { printf "%.3f s", b - a }')"
fi
echo "   expected windows: released within two advert cycles of the sibling run end (default 60 s each; lower-state"
echo "   answer 1, a design value); took within a second of released; took -> committed is the merge itself"
look "4: the few seconds of scene 2.4 are took -> committed. The time from the sibling run end to took is two advert cycles at most."

echo
echo "== 5. the new ir on both nodes =="
api "$M/v1/capabilities" | jq -c '.. | objects | select(has("ir") or (keys | map(startswith("repo.built.")) | any))' 2>/dev/null | head -5
look "5: both nodes advertise ir=$IR and repo.built.config-a, repo.built.config-b."

echo
echo "== 6. the merged lower equals the merged view before the merge =="
listing "$WS" > "$SLICE_DIR/$R_B.after"
if diff "$SLICE_DIR/$R_B.before" "$SLICE_DIR/$R_B.after" > "$SLICE_DIR/$R_B.diff"; then
  echo "   no difference ($(wc -l < "$SLICE_DIR/$R_B.after") entries)"
else
  sed 's/^/     /' "$SLICE_DIR/$R_B.diff" | head -40
fi
look "6: the listings are the same (path, type, mode); .enode-metadata.json is left out."

echo
echo "== 7. a build that fails, and a sync that does not reach the ir =="
R_F=slice6-fail-$STAMP
bake_contract "$R_F" "$WS" 4h "exit 3" "$BUILD_B" | submit_json "$R_F"
wait_run "$R_F"
steps "$R_F"
state_json | jq -c '.last_attempt'
step_log "$R_F" 1 | grep '^bake: ' | sed 's/^/     /'
R_I=slice6-ir-$STAMP
bake_contract "$R_I" "$WS" 4h "$BUILD_A" "$BUILD_B" "$IR_BAD" "$IR" | submit_json "$R_I"
wait_run "$R_I"
steps "$R_I"
step_log "$R_I" 1 | grep '^bake: ' | sed 's/^/     /'
look "7: the failed build is DONE with exit 3, config-b is skipped, the merge is DONE with nothing to merge and the run FAILED by its produced condition; last_attempt names the run. The ir bake is DONE with reason ir_mismatch, head and head_tags in the record, no manifest; the step log ends with the sentence on why; the run FAILED (completion condition 10, US-19)."
echo
echo "slice 6: done; judge it by the looks above"
