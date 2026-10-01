#!/usr/bin/env bash
# finalize-bake 조각 9 (보존) — 팩의 scene-gates.md 장면 3 · 조각 9 · checkpoint 유닛의 business-logic-model.md 7절.
#
# 사람이 보는 조각이다. 이 스크립트는 판정하지 않는다 — 순서대로 돌리고 무엇을 볼지 출력한 뒤 Enter 를 기다린다.
# 코드 시험이 초록이라는 것으로 이 조각을 대신하지 않는다.
#
# 이미 도는 노드를 쓴다 — 스크립트가 노드를 띄우지 않는다. 설정을 고치고 데몬을 다시 띄우는 일은 사람이 한다
# (스크립트가 멈추고 할 일을 출력한다).
#
#   export M=http://<mediator>:8080 T=<bootstrap token>
#   export NODE_CONFIG=<runc-overlay 노드의 설정 파일>   checkpoint: {ttl_hours: 1, max_gb: 1} 을 적고 띄운다
#   export WS=<그 노드의 워크스페이스>                  계약의 ws 로 그 노드를 고른다
#   export NATIVE_WS=<native 노드의 워크스페이스>        비우면 4 를 건너뛴다
#   export ENODE=<그 노드의 enode 실행 파일>            조회 명령.  기본은 PATH 의 enode
#   scripts/finalize-bake/slice-9.sh
#
# 차례 — TTL 을 먼저 건다.  8 이 한 시간과 10분을 기다려야 하므로 첫 보존본을 가장 먼저 잡고 기다리는 동안 나머지를 돈다.
#
#   1  exit 1 단계와 exit 0 단계            앞은 captured 와 ID · expires_at 이 한 시간 뒤 · 뒤는 not_requested
#   2  enode checkpoint list · show        kept · 보고 뒤 크기 · delivered · path · open · discard 줄
#   3  1 GiB 넘는 upper 를 남기고 exit 1      receipt 는 captured · 몇 초 뒤 목록에 evicted: larger than max_gb
#   4  native 노드에 exit 1 단계             unsupported (runtime) · 그 문장
#   5  주인 없는 reserved 를 놓고 재시작       그 항목이 trash 로 가고 삭제자가 지운다 · 도는 단계의 예약은 목록에 없다
#   6  policy: off 로 1 의 실패 계약을 다시   exit_code · produced · finalize 칸이 1 과 같다 · not_requested
#   7  min_free_gb 를 여유보다 1 GB 작게      단계가 2 GB 를 쓰고 exit 1 · rejected (free_space).  끝나면 되돌린다
#   8  1 의 expires_at 과 10분이 지난 뒤      목록에 없음 · show 는 exit 1 · Record 는 captured 그대로
#
# 환경 변수
#   SCRATCH    scratch 의 자리.  기본은 NODE_CONFIG 의 environment.scratch
#   SLICE_DIR  스크래치 자리.  기본 ${TMPDIR:-/tmp}/enode-slice-9
#   PAUSE      0 이면 Enter 를 기다리지 않는다 (todo 는 늘 기다린다)
set -euo pipefail

cd "$(dirname "$0")/../.."
: "${M:?set M to the mediator address}"
: "${T:?set T to the bootstrap token}"
: "${NODE_CONFIG:?set NODE_CONFIG to the config file of a running runc-overlay node}"
: "${WS:?set WS to the workspace of that node}"
ENODE=${ENODE:-enode}
SLICE_DIR=${SLICE_DIR:-${TMPDIR:-/tmp}/enode-slice-9}
BIN=$SLICE_DIR/bin
STAMP=$(date +%s)
# environment.scratch 를 설정에서 읽는다 — 들여쓴 scratch: 줄 하나 (slice-4.sh 와 같다)
SCRATCH=${SCRATCH:-$(awk '/^environment:/{e=1;next} e&&/^[^ #]/{e=0} e&&$1=="scratch:"{print $2}' "$NODE_CONFIG" | tr -d '"'"'")}
: "${SCRATCH:?cannot find environment.scratch in NODE_CONFIG; set SCRATCH}"
SPOOL=$SCRATCH/spool
TRASH=$SCRATCH/trash

for tool in go jq curl git df; do
  command -v "$tool" >/dev/null || { echo "slice 9: $tool is not installed"; exit 1; }
done
command -v "$ENODE" >/dev/null || [ -x "$ENODE" ] || { echo "slice 9: cannot run $ENODE; set ENODE"; exit 1; }
git config --get user.email >/dev/null ||
  { echo "slice 9: runctl needs git config user.email (or GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=user.email GIT_CONFIG_VALUE_0=<you>)"; exit 1; }

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

contract() {
  local id=$1 ws=$2 script=$3
  jq -n --arg id "$id" --arg ws "$ws" --arg script "$script" '{
    run_id: $id,
    work: { id: { system: "manual", change_id: $id }, system: "manual" },
    requires: [ { as: "node", capability: "agent.reason", ws: $ws } ],
    steps: [ { id: "slice", uses: "node", run: ["sh", "-c", $script] } ],
    success_when: [ { step: "slice", exit_code: 0 } ]
  }'
}

submit() {
  local id=$1 ws=$2 script=$3
  contract "$id" "$ws" "$script" > "$SLICE_DIR/$id.json"
  ENODE_MEDIATOR=$M ENODE_TOKEN=$T "$BIN/runctl" submit "$SLICE_DIR/$id.json" > /dev/null
}

wait_run() {
  local id=$1
  for _ in $(seq 1 18000); do
    case "$(api "$M/v1/runs/$id" | jq -r '.state')" in SUCCEEDED | FAILED | CANCELLED) return 0 ;; esac
    sleep 0.2
  done
  echo "slice 9: run $id did not end"
  exit 1
}

# 봉인된 단계 기록의 결과 칸 — 보존의 receipt 와 diagnostics 의 문장 · 단계의 결과 칸
step_json() {
  local id=$1
  for _ in $(seq 1 60); do
    ENODE_MEDIATOR=$M ENODE_TOKEN=$T "$BIN/runctl" record "$id" -o "$SLICE_DIR/$id.tar" > /dev/null 2>&1 && break
    sleep 1
  done
  tar -xOf "$SLICE_DIR/$id.tar" --wildcards '*/steps/01-*.json'
}

record() {
  step_json "$1" | jq '{state, exit_code: .result.exit_code, produced: .result.produced, error: .result.error,
    finalize: .result.finalize, checkpoint_capture: .result.checkpoint_capture,
    detail: .result.diagnostics.checkpoint}'
}

capture_id() { step_json "$1" | jq -r '.result.checkpoint_capture.id // empty'; }
expires_at() { step_json "$1" | jq -r '.result.checkpoint_capture.expires_at // empty'; }

list() { "$ENODE" checkpoint list --config "$NODE_CONFIG"; }
free_gb() { df -BG --output=avail "$SCRATCH" | tail -1 | tr -dc '0-9'; }

echo "== build =="
mkdir -p "$BIN"
go build -o "$BIN/runctl" ./cmd/runctl
echo "   node config  $NODE_CONFIG"
echo "   scratch      $SCRATCH"
echo "   spool        $SPOOL"
echo "   free         $(free_gb) GB"
grep -n -A6 '^checkpoint:' "$NODE_CONFIG" | sed 's/^/   /' ||
  todo "add checkpoint: {ttl_hours: 1, max_gb: 1} to $NODE_CONFIG and restart that node"

echo
echo "== 1 · a failing step and a passing step (the ttl clock starts here) =="
A=slice9-fail-$STAMP
submit "$A" "$WS" 'mkdir -p built && printf failed > built/out.bin; exit 1'
submit "slice9-pass-$STAMP" "$WS" 'printf ok > built-ok; exit 0'
wait_run "$A"
wait_run "slice9-pass-$STAMP"
record "$A"
record "slice9-pass-$STAMP"
A_ID=$(capture_id "$A")
A_EXPIRES=$(expires_at "$A")
echo "   checkpoint $A_ID expires at $A_EXPIRES"
look "1: the failing step has checkpoint_capture captured with a 12-character id, scope workspace-upper, guarantee inspect-only, this node, and expires_at one hour after the step; the passing step has not_requested and no detail."

echo
echo "== 2 · enode checkpoint list and show =="
for _ in $(seq 1 60); do
  "$ENODE" checkpoint show "$A_ID" --config "$NODE_CONFIG" | grep -q '^size  *unmeasured' || break
  sleep 1
done
list
"$ENODE" checkpoint show "$A_ID" --config "$NODE_CONFIG"
look "2: the list shows $A_ID as kept with a measured size and report delivered; show has path, the open line with unshare, and the discard line. The status file and the panel show the spool line."

echo
echo "== 3 · an upper over max_gb =="
B=slice9-big-$STAMP
submit "$B" "$WS" 'dd if=/dev/zero of=big.bin bs=1M count=1100 status=none; exit 1'
wait_run "$B"
record "$B"
B_ID=$(capture_id "$B")
for _ in $(seq 1 120); do
  list | grep -q "^$B_ID  *evicted" && break
  sleep 1
done
list
look "3: the receipt says captured (the fact at the time), and the list now says evicted: larger than max_gb for $B_ID."

echo
echo "== 4 · a native node =="
if [ -n "${NATIVE_WS:-}" ]; then
  C=slice9-native-$STAMP
  submit "$C" "$NATIVE_WS" 'exit 1'
  wait_run "$C"
  record "$C"
  look "4: unsupported with reason runtime, and the detail says the node runs steps without an isolated upper."
else
  echo "   NATIVE_WS is not set; skipped"
fi

echo
echo "== 5 · restart reconciliation and a running step's reservation =="
ORPHAN=$(od -An -tx1 -N6 /dev/urandom | tr -d ' \n')
mkdir -m 700 "$SPOOL/$ORPHAN"
printf '{"schema":1,"id":"%s","state":"reserved"}\n' "$ORPHAN" > "$SPOOL/$ORPHAN/checkpoint.json"
: > "$SPOOL/$ORPHAN/.enode-session.lock"
list
todo "restart the runc-overlay node (stop it and start it again with the same config)"
for _ in $(seq 1 60); do [ -e "$SPOOL/$ORPHAN" ] || break; sleep 1; done
echo "   spool has $ORPHAN: $([ -e "$SPOOL/$ORPHAN" ] && echo yes || echo no) · trash: $(ls -A "$TRASH" 2>/dev/null | tr '\n' ' ')"
D=slice9-long-$STAMP
submit "$D" "$WS" 'sleep 30; exit 0'
sleep 10
echo "   spool entries while the long step runs: $(ls "$SPOOL" | tr '\n' ' ')"
list
wait_run "$D"
look "5: the ownerless reservation $ORPHAN left the spool after the restart and the deleter removed it; while the long step ran its reservation was in the spool but not in the list."

echo
echo "== 6 · the same failing step with policy off =="
todo "set checkpoint: {policy: off, ttl_hours: 1, max_gb: 1} in $NODE_CONFIG and restart the node"
E=slice9-off-$STAMP
submit "$E" "$WS" 'mkdir -p built && printf failed > built/out.bin; exit 1'
wait_run "$E"
record "$A"
record "$E"
look "6: exit_code, produced and finalize are the same for $A and $E; $E has not_requested. The capture did not change the step."

echo
echo "== 7 · free space below min_free_gb =="
NOW_FREE=$(free_gb)
todo "set checkpoint back to {ttl_hours: 1, max_gb: 1} and min_free_gb: $((NOW_FREE - 1)) in $NODE_CONFIG (free is $NOW_FREE GB now), and restart the node"
F=slice9-free-$STAMP
submit "$F" "$WS" 'dd if=/dev/zero of=fill.bin bs=1M count=2048 status=none; exit 1'
wait_run "$F"
record "$F"
look "7: rejected with reason free_space and the detail names the free space and min_free_gb. Filling the disk before the step would drain the node so it could not claim the step."
todo "set min_free_gb back to its old value in $NODE_CONFIG and restart the node"

echo
echo "== 8 · the ttl =="
if [ -n "$A_EXPIRES" ]; then
  until_epoch=$(( $(date -d "$A_EXPIRES" +%s) + 600 ))
  while [ "$(date +%s)" -lt "$until_epoch" ]; do
    echo "   $(date +%T)  $(( (until_epoch - $(date +%s)) / 60 )) minutes left"
    sleep 60
  done
fi
list
set +e
"$ENODE" checkpoint show "$A_ID" --config "$NODE_CONFIG"
echo "   show exit $?"
set -e
record "$A"
look "8: $A_ID is gone from the list, show says no checkpoint and exits 1, and the sealed record of $A still says captured — the fact at the time."

echo
echo "slice 9: done — judge each look above"
