#!/usr/bin/env bash
# finalize-bake 조각 4 (trash) — 팩의 scene-gates.md 2절 · 요구 문서 6절 · trash 유닛의
# business-logic-model.md 7절.
#
# 사람이 보는 조각이다. 이 스크립트는 판정하지 않는다 — 순서대로 돌리고 무엇을 볼지
# 출력한 뒤 Enter 를 기다린다. 코드 시험이 초록이라는 것으로 이 조각을 대신하지 않는다.
#
# 이미 도는 runc-overlay 노드를 쓴다 — 스크립트가 노드를 띄우지 않는다. 데몬을 다시 띄우는
# 일은 사람이 한다 (스크립트가 멈추고 할 일을 출력한다).
#
#   export M=http://<mediator>:8080 T=<bootstrap token>
#   export NODE_CONFIG=<그 노드의 설정 파일>   상태 파일 · 잠금 파일 · scratch 를 여기서 찾는다
#   export WS=<그 노드의 워크스페이스>          계약의 ws 로 그 노드를 고른다
#   export ROOTFS=<준비된 rootfs 경로>          ⑥ 의 Open 실패에 쓴다.  비우면 그 줄을 건너뛴다
#   export ENODE=<그 노드의 enode 실행 파일>     ⑥ 의 env check 에 쓴다.  기본은 PATH 의 enode
#   scripts/finalize-bake/slice-4.sh
#
#   ①  15만 파일 · 9 GB upper 를 남기는 단계와 작은 upper 를 남기는 단계 — finalized_at - exited_at
#      이 비슷하다 (1초 안쪽).  보고 전에는 rename 한 번이다
#   ②  보고 직후 <scratch>/trash 가 차 있고 곧 빈다.  상태 파일의 scratch 칸이 측정 · 지우는 중 ·
#      빈 것으로 바뀐다.  9 GB 를 지우는 데 든 시간과 이 기계의 IO 스케줄러를 적는다
#   ③  권한 000 인 work/work 와 whiteout 도 지워진다.  upper 의 항목은 노드 uid 소유다 — 단계 사용자
#      (컨테이너 uid 1000)가 노드 uid 로 매핑된다
#   ④  데몬을 SIGKILL 로 죽이고 다시 띄우면 남은 작업 폴더가 trash 로 가고 비워진다
#   ⑤  min_free_gb 를 여유보다 크게 두면 GET /v1/nodes 에 draining · 상태 파일에 여유 부족 출처.
#      그동안 arch 키는 광고에 남는다.  되돌리면 다음 광고에 풀린다
#   ⑥  다섯 자리 모두 trash 로 간다 — Open 실패 · 보통 닫기(①) · helper cleanup(④) ·
#      env check 의 smoke.  abort 는 기본 go test 가 가짜 helper 로 본다
#
# 환경 변수
#   SCRATCH    scratch 의 자리.  기본은 NODE_CONFIG 의 environment.scratch
#   FILES      ① 의 큰 upper 의 파일 수.  기본 150000
#   UPPER_GB   ① 의 큰 upper 의 크기.  기본 9
#   SLICE_DIR  스크래치 자리.  기본 ${TMPDIR:-/tmp}/enode-slice-4
#   PAUSE      0 이면 Enter 를 기다리지 않는다
set -euo pipefail

cd "$(dirname "$0")/../.."
: "${M:?set M to the mediator address}"
: "${T:?set T to the bootstrap token}"
: "${NODE_CONFIG:?set NODE_CONFIG to the config file of a running runc-overlay node}"
: "${WS:?set WS to the workspace of that node}"
SLICE_DIR=${SLICE_DIR:-${TMPDIR:-/tmp}/enode-slice-4}
FILES=${FILES:-150000}
UPPER_GB=${UPPER_GB:-9}
BIN=$SLICE_DIR/bin
STAMP=$(date +%s)
STATUS=${NODE_CONFIG%.*}.status.yaml
LOCK=$NODE_CONFIG.lock
# environment.scratch 를 설정에서 읽는다 — 들여쓴 scratch: 줄 하나
SCRATCH=${SCRATCH:-$(awk '/^environment:/{e=1;next} e&&/^[^ #]/{e=0} e&&$1=="scratch:"{print $2}' "$NODE_CONFIG" | tr -d '"'"'")}
: "${SCRATCH:?cannot find environment.scratch in NODE_CONFIG; set SCRATCH}"
TRASH=$SCRATCH/trash
ROOTFS_MOVED=""

for tool in go jq curl git; do
  command -v "$tool" >/dev/null || { echo "slice 4: $tool is not installed"; exit 1; }
done
# runctl 은 git 의 user.email 을 요청자로 쓴다. 없으면 submit 이 멈춘다 — 처음에 알린다
git config --get user.email >/dev/null ||
  { echo "slice 4: runctl needs git config user.email (or GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=user.email GIT_CONFIG_VALUE_0=<you>)"; exit 1; }

cleanup() {
  # ⑥ 에서 옮긴 rootfs 는 무슨 일이 있어도 되돌린다
  if [ -n "$ROOTFS_MOVED" ] && [ -d "$ROOTFS_MOVED" ] && [ ! -e "$ROOTFS" ]; then
    mv "$ROOTFS_MOVED" "$ROOTFS"
  fi
}
trap cleanup EXIT

look() {
  echo
  echo ">> look: $*"
  if [ "${PAUSE:-1}" != 0 ]; then read -r -p "   press Enter to go on " _; fi
}

todo() {
  echo
  echo ">> do: $*"
  read -r -p "   press Enter when done " _
}

api() { curl -s -H "Authorization: Bearer $T" "$@"; }

node_id() { api "$M/v1/nodes" | jq -r --arg ws "$WS" '.nodes[] | select(.capabilities[]?.attrs.ws == $ws) | .node_id' | head -1; }

trash_now() {
  printf '   %s  trash: %s\n' "$(date +%T.%3N)" "$(ls -A "$TRASH" 2>/dev/null | tr '\n' ' ')"
}

scratch_now() {
  printf '   %s  status scratch: %s\n' "$(date +%T.%3N)" \
    "$(awk '/^scratch:/{s=1;next} s&&/^[^ ]/{s=0} s{printf "%s ", $0}' "$STATUS" 2>/dev/null | tr -s ' ')"
}

# 비워질 때까지 0.2초마다 trash 와 상태 파일의 scratch 칸을 찍는다. 바뀔 때만 찍는다
watch_trash() {
  local last="" began
  began=$(date +%s%N)
  for _ in $(seq 1 3000); do
    local line
    line="$(ls -A "$TRASH" 2>/dev/null | tr '\n' ' ') | $(awk '/^scratch:/{s=1;next} s&&/^[^ ]/{s=0} s{printf "%s ", $0}' "$STATUS" 2>/dev/null | tr -s ' ')"
    if [ "$line" != "$last" ]; then
      printf '   %s  %s\n' "$(date +%T.%3N)" "$line"
      last=$line
    fi
    [ -z "$(ls -A "$TRASH" 2>/dev/null)" ] && break
    sleep 0.2
  done
  echo "   trash emptied after $((($(date +%s%N) - began) / 1000000)) ms"
}

contract() {
  local id=$1 script=$2
  jq -n --arg id "$id" --arg ws "$WS" --arg script "$script" '{
    run_id: $id,
    work: { id: { system: "manual", change_id: $id }, system: "manual" },
    requires: [ { as: "node", capability: "agent.reason", ws: $ws } ],
    steps: [ { id: "slice", uses: "node", run: ["sh", "-c", $script] } ],
    success_when: [ { step: "slice", exit_code: 0 } ]
  }'
}

submit() {
  local id=$1 script=$2
  contract "$id" "$script" > "$SLICE_DIR/$id.json"
  ENODE_MEDIATOR=$M ENODE_TOKEN=$T "$BIN/runctl" submit "$SLICE_DIR/$id.json" > /dev/null
}

wait_run() {
  local id=$1
  for _ in $(seq 1 3600); do
    case "$(api "$M/v1/runs/$id" | jq -r '.state')" in SUCCEEDED | FAILED | CANCELLED) return 0 ;; esac
    sleep 0.2
  done
  echo "slice 4: run $id did not end"
  exit 1
}

wait_phase() {
  local id=$1 phase=$2
  for _ in $(seq 1 600); do
    [ "$(api "$M/v1/runs/$id" | jq -r '.steps[0].phase // empty')" = "$phase" ] && return 0
    sleep 0.2
  done
  echo "slice 4: run $id never reached phase $phase"
  exit 1
}

# 단계 기록의 두 시각과 그 차 (ms). 시각은 노드 시계다 — 둘 다 같은 시계라 차가 뜻이 있다
record() {
  local id=$1 json exited finalized
  for _ in $(seq 1 60); do
    ENODE_MEDIATOR=$M ENODE_TOKEN=$T "$BIN/runctl" record "$id" -o "$SLICE_DIR/$id.tar" > /dev/null 2>&1 && break
    sleep 1
  done
  json=$(tar -xOf "$SLICE_DIR/$id.tar" --wildcards '*/steps/01-*.json')
  jq '{state, error: .result.error, finalize: .result.finalize, reason: .result.reason,
       exited_at: .result.exited_at, finalized_at: .result.finalized_at}' <<<"$json"
  exited=$(jq -r '.result.exited_at // empty' <<<"$json")
  finalized=$(jq -r '.result.finalized_at // empty' <<<"$json")
  if [ -n "$exited" ] && [ -n "$finalized" ]; then
    echo "   finalized_at - exited_at = $((($(date -d "$finalized" +%s%N) - $(date -d "$exited" +%s%N)) / 1000000)) ms"
  fi
}

echo "== build =="
mkdir -p "$BIN"
go build -o "$BIN/runctl" ./cmd/runctl
echo "   node config  $NODE_CONFIG"
echo "   scratch      $SCRATCH"
echo "   status file  $STATUS"
NODE=$(node_id)
[ -n "$NODE" ] || { echo "slice 4: no node advertises ws=$WS"; exit 1; }
echo "   node         $NODE"
echo "   IO schedulers of the disks (idle IO works only under bfq):"
for d in /sys/block/*; do
  case ${d##*/} in loop* | nbd* | ram* | sr* | zram*) continue ;; esac
  [ -r "$d/queue/scheduler" ] && printf '     %s  %s\n' "${d##*/}" "$(cat "$d/queue/scheduler")"
done
df -hT "$SCRATCH" | sed 's/^/     /'

# 워크스페이스에 파일 하나를 둔다 — 단계가 지우면 upper 에 whiteout 이 남는다 (③)
printf 'lower\n' > "$WS/.slice4-lower"

BIG="mkdir -p slice4-big && cd slice4-big && seq -f 'f%06g' 1 $FILES | xargs touch && \
(fallocate -l ${UPPER_GB}G blob 2>/dev/null || dd if=/dev/zero of=blob bs=1M count=$((UPPER_GB * 1024)) status=none) && \
rm -f ../.slice4-lower && id"
SMALL="printf small > slice4-small && id"

echo
echo "== 1. a step leaving a big upper and one leaving a small upper =="
R_BIG=slice4-big-$STAMP
submit "$R_BIG" "$BIG"
wait_run "$R_BIG"
trash_now
scratch_now
echo "   2. trash right after the report, until it is empty:"
watch_trash
R_SMALL=slice4-small-$STAMP
submit "$R_SMALL" "$SMALL"
wait_run "$R_SMALL"
echo "   step record of the big upper:"
record "$R_BIG"
echo "   step record of the small upper:"
record "$R_SMALL"
look "1: finalized_at - exited_at of the two records are alike and under a second; the big one no longer pays for deleting $FILES files and $UPPER_GB GB. 2: trash held the work folder right after the report and emptied; the status scratch line went through measured bytes, deleting: true and back to 0 entries. The seconds it took to empty are how long $UPPER_GB GB takes on this disk."

echo
echo "== 3. what the node uid cannot remove =="
echo "   the big step ran as the rootfs user, which maps to the node uid, so its upper held node-owned files,"
echo "   work/work with mode 000 (the node uid cannot read it) and a whiteout for .slice4-lower."
look "3: trash is empty and the node log has no 'cannot remove trash entry' line. grep -E 'trash entry|trash helper' <node log>"

echo
echo "== 4. the daemon dies while a step runs =="
R_KILL=slice4-kill-$STAMP
submit "$R_KILL" "printf x > slice4-kill && sleep 120"
wait_phase "$R_KILL" running
PID=$(head -1 "$LOCK")
echo "   killing the daemon pid $PID with SIGKILL"
kill -9 "$PID"
sleep 2
echo "   work folders left in scratch: $(ls -d "$SCRATCH"/enode-runc-* 2>/dev/null | tr '\n' ' ')"
todo "start the node again (enodectl start, or the way you start it)"
for _ in $(seq 1 60); do [ -n "$(node_id)" ] && break; sleep 1; done
echo "   work folders left in scratch: $(ls -d "$SCRATCH"/enode-runc-* 2>/dev/null | tr '\n' ' ')"
trash_now
watch_trash
look "4: the killed step's work folder was in scratch until the restart, then 'orphaned runtime session moved to trash' in the node log, and trash emptied. The helper's own cleanup did not remove it."

echo
echo "== 5. free disk below min_free_gb =="
FREE_GB=$(df -BG --output=avail "$WS" | tail -1 | tr -dc '0-9')
echo "   free on the workspace filesystem: $FREE_GB GB"
todo "set min_free_gb: $((FREE_GB + 5)) in $NODE_CONFIG and restart the node"
sleep 5
for _ in $(seq 1 120); do
  [ "$(api "$M/v1/nodes" | jq -r --arg n "$NODE" '.nodes[] | select(.node_id == $n) | .draining')" != "" ] && break
  sleep 1
done
api "$M/v1/nodes" | jq --arg n "$NODE" '.nodes[] | select(.node_id == $n) |
  {draining, arch_keys: [.capabilities[].attrs | to_entries[] | select(.key | startswith("arch")) | "\(.key)=\(.value)"]}'
awk '/^drain:/{s=1} s&&/^[^ ]/&&!/^drain:/{s=0} s' "$STATUS"
if ! api "$M/v1/nodes" | jq -e --arg n "$NODE" '.nodes[] | select(.node_id == $n) | .capabilities[].attrs | keys[] | select(startswith("arch"))' >/dev/null; then
  echo "   this node advertises no arch key (no toolchain and no arch: in its config); add arch: to see the keys stay"
fi
look "5: draining is graceful, the arch keys are still there, and the status file names the source disk with 'free N GB < min M GB'. The panel shows the same source and no lift button for it."
todo "put min_free_gb back in $NODE_CONFIG and restart the node"
for _ in $(seq 1 120); do
  [ "$(api "$M/v1/nodes" | jq -r --arg n "$NODE" '.nodes[] | select(.node_id == $n) | .draining')" = "" ] && break
  sleep 1
done
api "$M/v1/nodes" | jq --arg n "$NODE" '.nodes[] | select(.node_id == $n) | {draining}'
look "5: the drain is lifted on the next advert."

echo
echo "== 6. the other places a work folder leaves from =="
if [ -z "${ROOTFS:-}" ]; then
  echo "   ROOTFS is unset; the open failure is skipped"
else
  ROOTFS_MOVED=$ROOTFS.slice4-off
  mv "$ROOTFS" "$ROOTFS_MOVED"
  R_OPEN=slice4-open-$STAMP
  submit "$R_OPEN" "true"
  wait_run "$R_OPEN"
  mv "$ROOTFS_MOVED" "$ROOTFS"
  ROOTFS_MOVED=""
  api "$M/v1/runs/$R_OPEN" | jq '{state, steps: [.steps[] | {state, error}]}'
  trash_now
  watch_trash
  look "6: the step failed with 'runtime open:' and its work folder went to trash, then away."
fi
echo "   env check runs the same runtime smoke; its work folder goes to trash too"
"${ENODE:-enode}" env check --config "$NODE_CONFIG" > "$SLICE_DIR/env-check.txt" 2>&1 || true
tail -3 "$SLICE_DIR/env-check.txt" | sed 's/^/     /'
trash_now
R_KICK=slice4-kick-$STAMP
submit "$R_KICK" "true"
wait_run "$R_KICK"
watch_trash
look "6: the smoke's work folder waited in trash until the next report woke the deleter, then went away. No enode-runc-* is left in $SCRATCH."
ls -d "$SCRATCH"/enode-runc-* 2>/dev/null || echo "   scratch holds no work folder"
