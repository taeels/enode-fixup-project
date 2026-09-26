#!/usr/bin/env bash
# finalize-bake 조각 1 (걷지 않는다) — 팩의 scene-gates.md 2절.
#
# 파일 300만 개의 워크스페이스에 선 노드와 빈 워크스페이스에 선 노드에 같은 결정론
# no-op 단계를 낸다. 봉인된 Record 의 단계 기록에서 finalized_at - exited_at 을 읽어
# 둘을 비교한다. 차이가 1초 안쪽이면 초록이다. 지목 경로 셋만 stat 했는지는 노드
# 로그의 finalized 줄(checked=3)로 본다. $OUT 의 이름이 봉인에 있는지도 본다.
#
#   export M=http://<mediator>:8080 T=<bootstrap token>
#   scripts/finalize-bake/slice-1.sh
#
# Mediator 는 이 유닛 이후의 것이어야 한다 — result 의 exited_at · finalized_at 을
# Record 로 옮기는 것은 step-phase 유닛(main 0c0370c)이다.
#
# 세 번째 Run 은 판정에 안 넣는다. 같은 큰 워크스페이스에 "discover": true 로 같은
# 단계를 내고, 명시 훑기의 방문 수 · 걸린 시간 · 닿은 상한을 출력한다 — 이 디스크에서
# 방문 상한(2,000,000)과 시간 상한(Finalize 예산의 절반) 중 무엇이 먼저 닿나다.
#
# 환경 변수
#   SLICE_DIR    스크래치 자리.  기본 ${TMPDIR:-/tmp}/enode-slice-1.  큰 트리는 다시 안 만든다
#   DIRS FILES   큰 트리의 모양.  기본 1000 x 3000
#   REMOVE_TREE  1 이면 끝에 큰 트리를 지운다.  기본은 남긴다 — 다시 만드는 데 몇 분이 든다
set -euo pipefail

cd "$(dirname "$0")/../.."
: "${M:?set M to the mediator address}"
: "${T:?set T to the bootstrap token}"
SLICE_DIR=${SLICE_DIR:-${TMPDIR:-/tmp}/enode-slice-1}
DIRS=${DIRS:-1000}
FILES=${FILES:-3000}
BIN=$SLICE_DIR/bin
STAMP=$(date +%s)
PIDS=()

for tool in go jq curl date; do
  command -v "$tool" >/dev/null || { echo "slice 1: red - $tool is not installed"; exit 1; }
done

cleanup() {
  for pid in "${PIDS[@]}"; do kill "$pid" 2>/dev/null || true; done
  wait 2>/dev/null || true
  if [ "${REMOVE_TREE:-0}" = 1 ]; then rm -rf "$SLICE_DIR/ws-big" "$SLICE_DIR/big.done"; fi
}
trap cleanup EXIT

red() { echo "slice 1: red - $*"; exit 1; }

echo "== build =="
mkdir -p "$BIN"
go build -o "$BIN/enode" ./cmd/enode
go build -o "$BIN/runctl" ./cmd/runctl

echo "== workspaces =="
# 큰 트리 — 빈 파일이다.  크기가 아니라 항목 수가 걷기의 비용이다
if [ ! -f "$SLICE_DIR/big.done" ]; then
  rm -rf "$SLICE_DIR/ws-big"
  mkdir -p "$SLICE_DIR/ws-big"
  echo "   creating $DIRS x $FILES empty files (once) ..."
  for ((d = 0; d < DIRS; d++)); do
    mkdir "$SLICE_DIR/ws-big/d$d"
    (cd "$SLICE_DIR/ws-big/d$d" && seq -f 'f%.0f' 0 $((FILES - 1)) | xargs touch)
  done
  touch "$SLICE_DIR/big.done"
fi
rm -rf "$SLICE_DIR/ws-empty"
mkdir -p "$SLICE_DIR/ws-empty/d0" "$SLICE_DIR/ws-empty/d1" "$SLICE_DIR/ws-empty/d2"
touch "$SLICE_DIR/ws-empty/d0/f0" "$SLICE_DIR/ws-empty/d1/f1" "$SLICE_DIR/ws-empty/d2/f2"
echo "   big   $SLICE_DIR/ws-big ($((DIRS * FILES)) files)"
echo "   empty $SLICE_DIR/ws-empty (3 files)"

# 노드 둘 — 설정 파일의 경로가 신원을 정한다
start_node() {
  local name=$1 ws=$2 dir=$SLICE_DIR/node-$1
  rm -rf "$dir"
  mkdir -p "$dir"
  cat > "$dir/enode.yaml" <<YAML
mediator: "$M"
token: "$T"
principal: "slice-1@example.invalid"
workspace: "$ws"
workspace_id: "slice-1-$name"
YAML
  "$BIN/enode" --config "$dir/enode.yaml" --ready-file "$dir/ready" > "$dir/node.log" 2>&1 &
  PIDS+=($!)
  for _ in $(seq 1 60); do
    [ -s "$dir/ready" ] && return 0
    sleep 1
  done
  tail -20 "$dir/node.log"
  red "node $name did not become ready"
}

echo "== nodes =="
start_node big "$SLICE_DIR/ws-big"
start_node empty "$SLICE_DIR/ws-empty"

# 결정론 no-op — 지목 경로 셋을 건드리고 이름 하나를 $OUT 에 낸다. 셸을 안 거친다.
# changed 는 워크스페이스를 적은 단계에만 쓸 수 있다 (ADR-037). repo 를 비워 두면 노드는
# 되돌리지 않은 채로 돈다 (ADR-036) — git 을 안 부르고 트리를 안 걷는다
contract() {
  local id=$1 ws=$2 discover=$3
  cat <<JSON
{
  "run_id": "$id",
  "work": { "id": { "system": "manual", "change_id": "$id" }, "system": "manual" },
  "requires": [ { "as": "node", "capability": "agent.reason", "ws": "$ws" } ],
  "steps": [
    {
      "id": "noop",
      "uses": "node",
      "workspace": { "repo": "" },
      "run": ["touch", "d0/f0", "d1/f1", "d2/f2", "\$OUT/result"],
      "out": ["result"],
      "discover": $discover
    }
  ],
  "success_when": [
    { "step": "noop", "exit_code": 0, "produced": ["result"], "changed": ["d0/f0", "d1/f1", "d2/f2"] }
  ]
}
JSON
}

# submit 은 Run 을 내고 끝나기를 기다린 뒤 봉인된 단계 기록을 stdout 에 낸다
submit() {
  local id=$1 ws=$2 discover=$3 file=$SLICE_DIR/$1
  contract "$id" "$ws" "$discover" > "$file.json"
  ENODE_MEDIATOR=$M ENODE_TOKEN=$T "$BIN/runctl" submit "$file.json" --wait > "$file.out" 2>&1 ||
    { cat "$file.out" >&2; red "run $id did not succeed"; }
  for _ in $(seq 1 60); do
    ENODE_MEDIATOR=$M ENODE_TOKEN=$T "$BIN/runctl" record "$id" -o "$file.tar" > /dev/null 2>&1 && break
    sleep 1
  done
  [ -s "$file.tar" ] || red "run $id was not sealed"
  tar -xOf "$file.tar" --wildcards '*/steps/01-*.json'
}

# window 는 단계 기록에서 finalized_at - exited_at 을 초로 낸다
window() {
  local exited finalized
  exited=$(jq -r '.exited_at // empty' <<<"$1")
  finalized=$(jq -r '.finalized_at // empty' <<<"$1")
  [ -n "$exited" ] && [ -n "$finalized" ] || red "the step record has no exited_at or finalized_at: $1"
  echo "$(date -d "$finalized" +%s.%N) - $(date -d "$exited" +%s.%N)" | awk '{ printf "%.3f", $1 - $3 }'
}

# logged 는 그 노드 로그의 마지막 finalized 줄에서 칸 하나를 낸다
logged() {
  grep 'msg=finalized' "$SLICE_DIR/node-$1/node.log" | tail -1 | tr ' ' '\n' | sed -n "s/^$2=//p"
}

echo "== runs =="
big=$(submit "slice1-big-$STAMP" "$SLICE_DIR/ws-big" false)
empty=$(submit "slice1-empty-$STAMP" "$SLICE_DIR/ws-empty" false)
wb=$(window "$big")
we=$(window "$empty")
echo "   big   finalized_at - exited_at = ${wb}s   checked=$(logged big checked) visited=$(logged big visited)"
echo "   empty finalized_at - exited_at = ${we}s   checked=$(logged empty checked) visited=$(logged empty visited)"

echo "== discover (reported, not judged) =="
disc=$(submit "slice1-discover-$STAMP" "$SLICE_DIR/ws-big" true)
echo "   visited=$(logged big visited) limit=$(logged big limit) took=$(logged big took)"
echo "   changes=$(jq -r '.result.diagnostics.changes' <<<"$disc") discovery_limit=$(jq -r '.result.diagnostics.discovery_limit // "none"' <<<"$disc")"

echo "== verdict =="
fail=()
awk -v a="$wb" -v b="$we" 'BEGIN { d = a - b; if (d < 0) d = -d; exit !(d < 1) }' ||
  fail+=("the big workspace took ${wb}s against ${we}s for the empty one")
for n in big empty; do
  [ "$(logged "$n" checked)" = 3 ] || fail+=("node $n stat'ed $(logged "$n" checked) paths, not the 3 named ones")
done
for rec in "$big" "$empty"; do
  jq -e '.result.produced | index("result")' <<<"$rec" > /dev/null || fail+=("result was not sealed")
  jq -e '.result.diagnostics.changes == "not_measured"' <<<"$rec" > /dev/null ||
    fail+=("changes is not not_measured on a step without discover")
done
if [ ${#fail[@]} -gt 0 ]; then
  printf '   %s\n' "${fail[@]}"
  red "${fail[0]}"
fi
echo "slice 1: green"
