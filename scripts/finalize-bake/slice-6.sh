#!/usr/bin/env bash
# finalize-bake 조각 6 (굽기가 끝까지 돈다) — 팩의 scene-gates.md 2절 장면 2 · bake 유닛 business-logic-model.md 8.3 ·
# 계획 7.1 · CG 물음 4 답 (「몇 초」 는 배타를 잡은 때부터 committed 까지).
#
# 사람이 보는 조각이다. 이 스크립트는 판정하지 않는다 — 순서대로 돌리고 무엇을 볼지 출력한 뒤 Enter 를 기다린다.
# 코드 시험이 초록이라는 것으로 이 조각을 대신하지 않는다 (US-14). 입력과 허용 표지는 bake-common.sh 의 머리를 본다.
#
#   export M=http://<mediator>:8080 T=<bootstrap token>
#   export WS=<baking node workspace> SIBLING_WS=<sibling workspace: a symlink to WS (ln -s "$WS" "$SIBLING_WS")>
#   export SYNC_URL=<git repository> IR=<a tag in it> IR_B=<another tag in it>   # 굽기 A 는 IR · 굽기 B 는 IR_B
#   export BUILD_A='<your build command for config-a>' BUILD_B='<your build command for config-b>'   # 비우면 짧은 기본값
#   export SIBLING_LOG=<the sibling node's log file>   # 4 의 형제 Run 끝과 released 시각.  비우면 사람이 적는다
#   export BAKING_LOG=<the baking node's log file>     # 4 의 took 과 합친 시각.  비우면 사람이 적는다
#   scripts/finalize-bake/slice-6.sh
#
#   1      빈 lower 에 굽기 A (IR) — build · merge DONE · metadata 의 칸 · bake.run · bake.node
#   2 · 3  형제에 긴 Run 을 낸 뒤 굽기 B (IR_B) — build 는 곧바로 · merge 는 기다린다 · 단계 로그에 쥔 쪽 줄과 남은 시간 ·
#          형제는 draining (한 광고 주기 안에 켜지기를 기다려 걸린 시간을 찍는다)
#   4      형제 Run 이 끝난 때 · 형제의 released · merge 의 took · committed 의 네 시각과 기대 창 (노드 로그의 시각)
#   5      두 노드가 새 ir (IR_B) 과 repo.built.<이름> 을 광고한다 (한 광고 주기 안에 바뀌기를 기다린다)
#   6      합치기 전 merged view 목록 == 합친 lower 목록 (.enode-metadata.json 은 뺀다)
#   7      빌드 하나를 일부러 실패시킨 굽기 · sync 가 IR 에 닿지 않는 굽기 — build DONE · merge 는 합칠 것 없음 · Run FAILED
#
# 환경 변수 — SIBLING_SECONDS (형제 Run 의 길이 · 기본 300) · IR_BAD (없는 태그 · 기본은 시각이 든 이름) ·
#   ADVERT (광고 주기의 초 · 기본 60 · Mediator 의 lease.renew_seconds 와 같게 둔다)
SLICE="slice 6"
cd "$(dirname "$0")/../.."
# shellcheck source=scripts/finalize-bake/bake-common.sh
. scripts/finalize-bake/bake-common.sh
: "${IR_B:?set IR_B to another tag in SYNC_URL; bake B moves the lower from IR to IR_B}"
if [ "$IR_B" = "$IR" ]; then
  echo "$SLICE: IR_B is the same tag as IR; a new ir would not show in 5; stopping"
  exit 1
fi
SIBLING_SECONDS=${SIBLING_SECONDS:-300}
IR_BAD=${IR_BAD:-enode-slice-no-such-tag-$STAMP}
ADVERT=${ADVERT:-60}
mkdir -p "$SLICE_DIR"

check_target
check_same_lower
build_runctl

epoch() { date -d "$1" +%s.%N; }
# span 은 두 시각 사이의 초다 (b - a).
span() { awk -v a="$(epoch "$1")" -v b="$(epoch "$2")" 'BEGIN { printf "%.3f s", b - a }'; }
# log_time 은 노드 로그에서 두 글자를 모두 담은 마지막 줄의 시각이다 (time=... 칸).  없으면 빈 글자.
log_time() {
  local file=$1 a=$2 b=$3
  [ -n "$file" ] && [ -r "$file" ] || return 0
  { grep -F -- "$a" "$file" | grep -F -- "$b" | tail -1 | sed -n 's/^time=\([^ ]*\) .*/\1/p'; } || true
}
# attrs_of 는 ws 가 그 글자인 노드의 광고 속성이다.
attrs_of() {
  api "$M/v1/nodes" | jq -c --arg ws "$1" '.nodes[] | ((.capabilities // []) | map(.attrs // {}) | add // {}) as $a |
    select($a.ws == $ws) | {node: .node_id, draining} + $a'
}
draining_of() { { attrs_of "$1" | jq -r '.draining // ""' | head -1; } || true; }
ir_of() { { attrs_of "$1" | jq -r '.ir // ""' | head -1; } || true; }

echo
echo "== 1. bake A on this lower (ir $IR) =="
R_A=slice6-a-$STAMP
bake_contract "$R_A" "$WS" 4h "$BUILD_A" "$BUILD_B" | submit_json "$R_A"
wait_run "$R_A"
steps "$R_A"
state_json | jq -c '{phase, since, last_attempt}'
metadata
look "1: build and merge are DONE and the run SUCCEEDED. .enode-metadata.json has every field (source url, branch, repo_id, head, ir, pinned, sync_command, synced_at, builds, environment, workspace_target, bake run, node, merged_at, resumed, previous_ir); bake.run is $R_A and bake.node is the baking node. First checks: the build reached pending, so the rootfs has bash and git and git in the session accepted the lower's owner (no safe.directory refusal)."

echo
echo "== 2, 3. a long run on the sibling, then bake B (ir $IR_B) =="
R_S=slice6-sibling-$STAMP
plain_contract "$R_S" "$SIBLING_WS" "$SIBLING_SECONDS" | submit_json "$R_S"
for _ in $(seq 1 600); do
  [ "$(api "$M/v1/runs/$R_S" | jq -r '.steps[0].phase // empty')" = running ] && break
  sleep 0.2
done
R_B=slice6-b-$STAMP
bake_contract "$R_B" "$WS" 4h "$BUILD_A" "$BUILD_B" "$IR_B" | submit_json "$R_B"
for _ in $(seq 1 6000); do
  step_log "$R_B" 2 | grep -q "waiting for the lower lock" && break
  sleep 0.2
done
UPPER=$(state_json | jq -r '.pending_upper')
merged_view "$UPPER" > "$SLICE_DIR/$R_B.before"
echo "   merge step log so far:"
step_log "$R_B" 2 | sed 's/^/     /'
# drain 은 merge 가 기다리기 시작한 뒤의 첫 광고에 켜진다 — merge 단계의 시작부터 한 광고 주기 (+5초) 까지 기다린다.
# 시작은 Mediator 의 시계 · 지금은 이 기계의 시계다.
W0=$(api "$M/v1/runs/$R_B" | jq -r '.steps[1].started_at')
DEADLINE=$(awk -v a="$(epoch "$W0")" -v s="$ADVERT" 'BEGIN { printf "%.3f", a + s + 5 }')
DRAINED=""
while awk -v n="$(date +%s.%N)" -v d="$DEADLINE" 'BEGIN { exit !(n < d) }'; do
  if [ -n "$(draining_of "$SIBLING_WS")" ]; then
    DRAINED=$(date -u +%Y-%m-%dT%H:%M:%S.%NZ)
    break
  fi
  sleep 0.2
done
if [ -n "$DRAINED" ]; then
  echo "   the sibling was draining $(span "$W0" "$DRAINED") after the merge step started ($W0)"
else
  echo "   the sibling was not draining within $((ADVERT + 5)) s of the merge step start ($W0)"
fi
echo "   nodes on this lower:"
attrs_of "$WS" | jq -c '{node, ws, draining}' | sed 's/^/     /'
attrs_of "$SIBLING_WS" | jq -c '{node, ws, draining}' | sed 's/^/     /'
look "2, 3: the build step ended at once; the merge step waits and its log names the sibling node holding the lower for run $R_S, with a deadline in node clock and the time left (completion condition 5); the sibling is draining within one advert cycle ($ADVERT s) of the merge step start."

echo
echo "== 4. four times (after the sibling run ends) =="
wait_run "$R_B"
steps "$R_S"
steps "$R_B"
COMMITTED=$(state_json | jq -r '.since')
S_END=$(log_time "${SIBLING_LOG:-}" 'msg="step finished"' "step=$R_S#01")
S_END_FROM="the sibling node log"
if [ -z "$S_END" ]; then
  S_END=$(jq -r '.ended_at' "$SLICE_DIR/$R_S"/*/steps/01-*.json)
  S_END_FROM="the record ended_at, mediator clock"
fi
RELEASED=$(log_time "${SIBLING_LOG:-}" 'released the lower lock' 'time=')
TOOK=$(log_time "${BAKING_LOG:-}" "step=$R_B#02" 'event=took')
MERGED=$(log_time "${BAKING_LOG:-}" "step=$R_B#02" 'msg="bake: merged"')
echo "   the sibling run ended          $S_END   ($S_END_FROM)"
if [ -n "$RELEASED" ]; then
  echo "   the sibling released the lock  $RELEASED   (run end + $(span "$S_END" "$RELEASED"))"
else
  echo "   the sibling released the lock  (read it in the sibling node log: released the lower lock)"
fi
if [ -n "$TOOK" ]; then
  if [ -n "$RELEASED" ]; then
    echo "   the merge took the lower lock  $TOOK   (released + $(span "$RELEASED" "$TOOK"))"
  else
    echo "   the merge took the lower lock  $TOOK"
  fi
  echo "   merged                         ${MERGED:-(not in the baking node log)}"
  echo "   committed                      $COMMITTED   (state.json since; took -> committed $(span "$TOOK" "$COMMITTED"))"
else
  echo "   the merge took the lower lock  (read it in the baking node log: event=took for step $R_B#02)"
  echo "   committed                      $COMMITTED   (state.json since)"
fi
echo "   all four are node clock unless marked; expected windows: released within two advert cycles of the sibling run"
echo "   end ($ADVERT s each; lower-state answer 1, a design value); took within a second of released; took -> committed"
echo "   is the merge itself"
look "4: the few seconds of scene 2.4 are took -> committed. The time from the sibling run end to took is two advert cycles at most."

echo
echo "== 5. the new ir on both nodes =="
# 광고는 합친 뒤의 첫 광고에 바뀐다 — committed 부터 한 광고 주기 (+5초) 까지 두 노드가 IR_B 를 광고하기를 기다린다
DEADLINE=$(awk -v a="$(epoch "$COMMITTED")" -v s="$ADVERT" 'BEGIN { printf "%.3f", a + s + 5 }')
NEW_IR=""
while awk -v n="$(date +%s.%N)" -v d="$DEADLINE" 'BEGIN { exit !(n < d) }'; do
  if [ "$(ir_of "$WS")" = "$IR_B" ] && [ "$(ir_of "$SIBLING_WS")" = "$IR_B" ]; then
    NEW_IR=$(date -u +%Y-%m-%dT%H:%M:%S.%NZ)
    break
  fi
  sleep 0.2
done
if [ -n "$NEW_IR" ]; then
  echo "   both nodes advertised ir=$IR_B $(span "$COMMITTED" "$NEW_IR") after committed"
else
  echo "   the two nodes did not both advertise ir=$IR_B within $((ADVERT + 5)) s of committed"
fi
for ws in "$WS" "$SIBLING_WS"; do
  attrs_of "$ws" | jq -c '{node, ir, "repo.built.config-a", "repo.built.config-b", "bake.run", "bake.resumed", ws}' |
    sed 's/^/     /'
done
look "5: both nodes advertise ir=$IR_B (bake A left $IR) and repo.built.config-a, repo.built.config-b; bake.run is $R_B."

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
bake_contract "$R_F" "$WS" 4h "exit 3" "$BUILD_B" "$IR_B" | submit_json "$R_F"
wait_run "$R_F"
steps "$R_F"
state_json | jq -c '.last_attempt'
step_log "$R_F" 1 | grep '^bake: ' | sed 's/^/     /'
R_I=slice6-ir-$STAMP
bake_contract "$R_I" "$WS" 4h "$BUILD_A" "$BUILD_B" "$IR_BAD" "$IR_B" | submit_json "$R_I"
wait_run "$R_I"
steps "$R_I"
step_log "$R_I" 1 | grep '^bake: ' | sed 's/^/     /'
look "7: the failed build is DONE with exit 3, config-b is skipped, the merge is DONE with nothing to merge and the run FAILED by its produced condition; last_attempt names the run. The ir bake is DONE with reason ir_mismatch, head and head_tags in the record, no manifest; the step log ends with the sentence on why; the run FAILED (completion condition 10, US-19)."
echo
echo "slice 6: done; judge it by the looks above"
