#!/usr/bin/env bash
# finalize-bake 조각 2 (보인다) — 팩의 scene-gates.md 2절 · 요구 문서 6절.
#
# 사람이 보는 조각이다. 이 스크립트는 판정하지 않는다 — 순서대로 돌리고 무엇을 볼지
# 출력한 뒤 Enter 를 기다린다. 코드 시험이 초록이라는 것으로 이 조각을 대신하지 않는다.
#
#   export M=http://<mediator>:8080 T=<bootstrap token>
#   export MEDIATOR_PID=<mediator 의 pid>      ② 에서 10초 멈춘다 (kill -STOP · -CONT)
#   export OLD_ENODE=<0c0370c 에서 빌드한 enode>  ④ 에 쓴다.  비우면 ④ 를 건너뛴다
#   scripts/finalize-bake/slice-2.sh
#
#   ①  exit 1 로 끝나고 큰 $OUT 을 올리는 단계 — 명령이 끝나는 순간 진행 조회에
#      phase finalizing 과 exit {kind: exit, code: 1} 이 보인다.  명령이 도는 동안은 running
#   ②  Mediator 를 10초 멈춘 채 같은 계약 — 종료 보고가 늦거나 재전송되어도 봉인된
#      단계 기록이 ① 과 같다 (exited_at 은 result 가 싣는다)
#   ③  같은 노드의 다른 instance 가 보낸 종료 보고는 409 다
#   ④  이 유닛 전의 노드 — phase running 에 머물고 last_phase 가 running 이다
#   ⑤  단계 기록에 exited_at 과 finalized_at 이 따로 있다
#
# 환경 변수
#   SLICE_DIR  스크래치 자리.  기본 ${TMPDIR:-/tmp}/enode-slice-2
#   OUT_MB     ① ② 의 $OUT 크기.  기본 256
#   PAUSE      0 이면 Enter 를 기다리지 않는다
set -euo pipefail

cd "$(dirname "$0")/../.."
: "${M:?set M to the mediator address}"
: "${T:?set T to the bootstrap token}"
SLICE_DIR=${SLICE_DIR:-${TMPDIR:-/tmp}/enode-slice-2}
OUT_MB=${OUT_MB:-256}
BIN=$SLICE_DIR/bin
STAMP=$(date +%s)
PIDS=()

for tool in go jq curl; do
  command -v "$tool" >/dev/null || { echo "slice 2: $tool is not installed"; exit 1; }
done

cleanup() {
  [ -n "${MEDIATOR_PID:-}" ] && kill -CONT "$MEDIATOR_PID" 2>/dev/null || true
  for pid in "${PIDS[@]}"; do kill "$pid" 2>/dev/null || true; done
  wait 2>/dev/null || true
}
trap cleanup EXIT

look() {
  echo
  echo ">> look: $*"
  if [ "${PAUSE:-1}" != 0 ]; then read -r -p "   press Enter to go on " _; fi
}

api() { curl -s -H "Authorization: Bearer $T" "$@"; }

echo "== build =="
mkdir -p "$BIN"
go build -o "$BIN/enode" ./cmd/enode
go build -o "$BIN/runctl" ./cmd/runctl

# 노드 하나 — 설정 파일의 경로가 신원을 정한다.  ready 파일에 노드 id 가 적힌다
start_node() {
  local name=$1 binary=$2 dir=$SLICE_DIR/node-$1
  rm -rf "$dir"
  mkdir -p "$dir/ws"
  cat > "$dir/enode.yaml" <<YAML
mediator: "$M"
token: "$T"
principal: "slice-2@example.invalid"
workspace: "$dir/ws"
workspace_id: "slice-2-$name"
YAML
  "$binary" --config "$dir/enode.yaml" --ready-file "$dir/ready" > "$dir/node.log" 2>&1 &
  PIDS+=($!)
  for _ in $(seq 1 60); do
    [ -s "$dir/ready" ] && return 0
    sleep 1
  done
  tail -20 "$dir/node.log"
  echo "slice 2: node $name did not become ready"
  exit 1
}

# 명령이 sleep 뒤 exit 1 로 끝나고 $OUT 에 큰 파일을 남긴다.  업로드가 길다
contract() {
  local id=$1 ws=$2 sleep=$3
  cat <<JSON
{
  "run_id": "$id",
  "work": { "id": { "system": "manual", "change_id": "$id" }, "system": "manual" },
  "requires": [ { "as": "node", "capability": "agent.reason", "ws": "$ws" } ],
  "steps": [
    {
      "id": "fail",
      "uses": "node",
      "run": ["sh", "-c", "head -c ${OUT_MB}M /dev/urandom > \$OUT/big; sleep $sleep; exit 1"],
      "out": ["big"],
      "budget": { "upload": "10m" }
    }
  ],
  "success_when": [ { "step": "fail", "exit_code": 0 } ]
}
JSON
}

submit() {
  local id=$1 ws=$2 sleep=$3
  contract "$id" "$ws" "$sleep" > "$SLICE_DIR/$id.json"
  ENODE_MEDIATOR=$M ENODE_TOKEN=$T "$BIN/runctl" submit "$SLICE_DIR/$id.json" > /dev/null 2>&1 || true
}

# watch 는 Run 이 끝날 때까지 1초마다 단계의 phase · phase_since · exit 를 찍는다
watch() {
  local id=$1
  while :; do
    local body state
    body=$(api "$M/v1/runs/$id")
    state=$(jq -r '.state' <<<"$body")
    printf '   %s  %-8s %s\n' "$(date +%T)" "$state" "$(jq -c '.steps[] | {phase, phase_since, exit}' <<<"$body")"
    case "$state" in DONE | FAILED | CANCELLED) break ;; esac
    sleep 1
  done
}

record() {
  local id=$1
  for _ in $(seq 1 60); do
    ENODE_MEDIATOR=$M ENODE_TOKEN=$T "$BIN/runctl" record "$id" -o "$SLICE_DIR/$id.tar" > /dev/null 2>&1 && break
    sleep 1
  done
  tar -xOf "$SLICE_DIR/$id.tar" --wildcards '*/steps/01-*.json' |
    jq '{state, exit, last_phase, exited_at, finalized_at,
         result: (.result | {exit_code, error, finalize, upload, reason, exited_at, finalized_at})}'
}

echo "== nodes =="
start_node new "$BIN/enode"
NODE_ID=$(cat "$SLICE_DIR/node-new/ready")
echo "   new node $NODE_ID"

echo
echo "== 1. the step ends with exit 1 and uploads ${OUT_MB} MiB =="
R1=slice2-visible-$STAMP
submit "$R1" "$SLICE_DIR/node-new/ws" 8
# ③ 을 여기서 한다 — 명령이 도는 동안 다른 instance 로 종료 보고를 보낸다
for _ in $(seq 1 30); do
  [ "$(api "$M/v1/runs/$R1" | jq -r '.steps[0].phase // empty')" = running ] && break
  sleep 1
done
echo "   3. an exit report from another instance of the same node:"
curl -s -o "$SLICE_DIR/other.json" -w '%{http_code}' -H "Authorization: Bearer $T" \
  -H 'Content-Type: application/json' -X POST "$M/v1/runs/$R1/steps/1/exited" \
  -d "{\"node\":\"$NODE_ID\",\"instance\":\"slice-2-another-life\",\"attempt\":0,\"outcome\":{\"kind\":\"exit\",\"code\":0},\"exited_at\":\"$(date -u +%FT%T.%NZ)\"}" \
  > "$SLICE_DIR/other.code" || true
echo "      HTTP $(cat "$SLICE_DIR/other.code")  $(cat "$SLICE_DIR/other.json")"
watch "$R1"
look "phase is running while the command runs, then finalizing with exit {kind: exit, code: 1} while ${OUT_MB} MiB uploads. The report from another instance above is 409."

echo
echo "== 2. the same contract with the mediator paused for 10 seconds =="
if [ -z "${MEDIATOR_PID:-}" ]; then
  echo "   MEDIATOR_PID is unset; skipped"
else
  R2=slice2-paused-$STAMP
  submit "$R2" "$SLICE_DIR/node-new/ws" 8
  for _ in $(seq 1 30); do
    [ "$(api "$M/v1/runs/$R2" | jq -r '.steps[0].phase // empty')" = running ] && break
    sleep 1
  done
  sleep 7 # 명령이 끝나기 직전
  kill -STOP "$MEDIATOR_PID"
  echo "   mediator paused at $(date +%T)"
  sleep 10
  kill -CONT "$MEDIATOR_PID"
  echo "   mediator resumed at $(date +%T)"
  watch "$R2"
  echo "   step record of run 1:"
  record "$R1"
  echo "   step record of run 2 (paused):"
  record "$R2"
  look "the two step records match apart from the times: same exit, last_phase, finalize, upload, and both carry exited_at. grep 'exit report' $SLICE_DIR/node-new/node.log shows the retries."
fi

echo
echo "== 4. a node from before this unit =="
if [ -z "${OLD_ENODE:-}" ]; then
  echo "   OLD_ENODE is unset; skipped"
else
  start_node old "$OLD_ENODE"
  R4=slice2-old-$STAMP
  submit "$R4" "$SLICE_DIR/node-old/ws" 3
  watch "$R4"
  record "$R4"
  look "the old node never reports its exit: phase stays running until the run ends, last_phase is running and exited_at is absent."
fi

echo
echo "== 5. two times in the step record =="
record "$R1"
look "exited_at and finalized_at are both present and finalized_at comes after exited_at."
echo "slice 2: walked through; the verdict is the person's"
