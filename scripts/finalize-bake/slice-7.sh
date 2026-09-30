#!/usr/bin/env bash
# finalize-bake 조각 7 (끊겨도 된다) — 팩의 scene-gates.md 2절 · bake 유닛 business-logic-model.md 8.3 · 결정 21.
#
# 사람이 보는 조각이다. 이 스크립트는 판정하지 않는다. 입력과 허용 표지는 bake-common.sh 의 머리를 본다.
#
#   export M=... T=... WS=... SIBLING_WS=... SYNC_URL=... IR=...
#   export NODE_CONFIG=<the baking node's config file>   # 그 노드의 pid 를 설정 옆의 잠금 파일에서 읽는다
#
# 형제는 굽는 노드의 합치기를 이어야 하므로 WS 와 같은 마운트로 lower 에 닿아야 한다 — SIBLING_WS 는 WS 를 가리키는 symlink
# 다 (ln -s "$WS" "$SIBLING_WS").  bind 별칭이면 멈춘다 (business-rules.md 10절 — 시작 전 확인의 마운트 줄).
#   scripts/finalize-bake/slice-7.sh
#
#   첫째 경우  같은 lower 의 다른 노드를 멈춘 채 merge 단계를 여러 자리에서 SIGKILL -> 다른 노드를 시작 ->
#             committed · metadata 의 resumed true 와 원래 Run · 합친 목록이 한 번에 끝낸 것과 같다
#   둘째 경우  형제를 띄워 둔 채 SIGKILL -> 형제가 한 광고 주기 안에 잇는다
#
# 합치기가 끊길 틈이 있게 구성 하나가 파일 MANY 개를 쓴다 (기본 50,000). 처음 한 번은 새 디렉터리라 rename 한 번에
# 들어가므로 먼저 한 번 굽고, 그 뒤의 굽기가 같은 파일을 모두 다시 써 항목마다 합친다. 합치기 전의 merged view 를
# 보려고 lower 의 공유 잠금을 잠깐 쥔다 (기록 없는 쥔 쪽 — smoke 와 같은 모양).
#
# 환경 변수 — KILL_AFTER (merging 을 본 뒤 SIGKILL 까지의 초 · 기본 "0 0.3 1") · MANY (기본 50000)
SLICE="slice 7"
cd "$(dirname "$0")/../.."
# shellcheck source=scripts/finalize-bake/bake-common.sh
. scripts/finalize-bake/bake-common.sh
: "${NODE_CONFIG:?set NODE_CONFIG to the config file of the baking node}"
KILL_AFTER=${KILL_AFTER:-"0 0.3 1"}
MANY=${MANY:-50000}
MANY_BUILD="mkdir -p .slice7 && cd .slice7 && seq -f 'f%06g' 1 $MANY | xargs touch"
mkdir -p "$SLICE_DIR"

check_target
if ! same_mount; then
  echo "$SLICE: SIBLING_WS reaches the lower through another mount (a bind alias); a sibling there cannot resume the"
  echo "$SLICE: baking node's merge (the merge preflight refuses a bind alias); point SIBLING_WS at a symlink to WS; stopping"
  exit 1
fi
check_same_lower
build_runctl

echo
echo "== prime: one bake so the next ones merge entry by entry =="
R_P=slice7-prime-$STAMP
bake_contract "$R_P" "$WS" 4h "$MANY_BUILD" "$BUILD_B" | submit_json "$R_P"
wait_run "$R_P"
steps "$R_P"

# kill_mid_merge 는 굽기 하나를 merging 까지 끌고 가 SIGKILL 한다. 합치기 전의 merged view 목록을 파일로 남긴다.
kill_mid_merge() {
  local id=$1 delay=$2
  hold_lower
  bake_contract "$id" "$WS" 4h "$MANY_BUILD" "$BUILD_B" | submit_json "$id"
  wait_phase pending 36000
  for _ in $(seq 1 600); do
    step_log "$id" 2 | grep -q "waiting for the lower lock" && break
    sleep 0.2
  done
  merged_view "$(state_json | jq -r '.pending_upper')" > "$SLICE_DIR/$id.before"
  release_lower
  wait_phase merging 600
  sleep "$delay"
  local pid
  pid=$(daemon_pid "$NODE_CONFIG")
  kill -9 "$pid"
  echo "   killed the baking daemon (pid $pid) ${delay}s after merging; the lower is $(phase)"
}

echo
echo "== case 1: the other node is stopped =="
i=0
for delay in $KILL_AFTER; do
  i=$((i + 1))
  todo "stop the sibling node (ws $SIBLING_WS) and make sure the baking node (ws $WS) is running"
  R=slice7-kill$i-$STAMP
  kill_mid_merge "$R" "$delay"
  todo "start the sibling node now; leave the baking node stopped"
  wait_phase committed 36000
  metadata | jq -c '{bake}'
  listing "$WS" > "$SLICE_DIR/$R.after"
  if diff "$SLICE_DIR/$R.before" "$SLICE_DIR/$R.after" > "$SLICE_DIR/$R.diff"; then
    echo "   the merged lower equals the merged view before the merge ($(wc -l < "$SLICE_DIR/$R.after") entries)"
  else
    sed 's/^/     /' "$SLICE_DIR/$R.diff" | head -40
  fi
  look "case 1 ($delay s): the sibling resumed at its start ('bake: resuming an interrupted merge' from=start, then 'bake: resumed the merge of run $R' in its log); state.json is committed; metadata bake.resumed is true and bake.run is $R (the original run); the listing matches a merge done in one go."
  todo "start the baking node again"
done

echo
echo "== case 2: the sibling stays up =="
todo "make sure both nodes are running"
R=slice7-live-$STAMP
kill_mid_merge "$R" 0.3
T0=$(date +%s)
wait_phase committed 36000
echo "   the sibling finished the merge $(($(date +%s) - T0)) s after the kill"
metadata | jq -c '{bake}'
look "case 2: a sibling that was running picked it up within one advert cycle (60 s by default); 'bake: resuming an interrupted merge' from=advert in its log; metadata bake.resumed is true and bake.run is $R."
todo "start the baking node again"
echo
echo "slice 7: done; judge it by the looks above"
