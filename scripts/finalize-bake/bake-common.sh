# shellcheck shell=bash
# finalize-bake 조각 6 · 7 · 8 (굽기) 의 공용 확인 — slice-6.sh · slice-7.sh · slice-8.sh 가 source 로 읽는다.
# bake 유닛 business-logic-model.md 8.3 · 계획 4.1 26번 · 물음 4 답 (운영 lower 면 멈춘다).
#
# 사람이 보는 조각이다. 스크립트는 판정하지 않는다 — 대상을 확인하고, 순서대로 돌리고, 무엇을 볼지 출력한 뒤 기다린다.
# 이미 도는 runc-overlay 노드 둘 (굽는 노드와 같은 lower 의 형제) 을 쓴다 — 스크립트가 노드를 띄우지 않는다.
#
# 입력 (환경 변수)
#   M · T          Mediator 주소와 bootstrap 토큰 (slice-4.sh 와 같은 이름)
#   WS             굽는 노드의 워크스페이스 = lower 루트.  계약의 ws 로 그 노드를 고른다
#   SIBLING_WS     같은 lower 를 다른 경로로 가리키는 형제의 워크스페이스.  글자가 WS 와 같으면 계약이 두 노드를 나누지
#                  못하므로 멈춘다 (계약은 노드가 광고하는 ws 글자로 고른다).  WS 를 가리키는 symlink 면 같은 마운트라
#                  형제가 굽는 노드의 합치기를 이을 수 있다 (조각 7).  bind 별칭 (mount --bind) 이면 다른 마운트라 잇지
#                  못한다 — 시작 전 확인의 마운트 줄 (business-rules.md 10절).  조각 8 의 4 는 bind 별칭으로 본다
#   SYNC_URL       sync 가 받아 올 git 저장소.  계약의 sync 는 git init · remote add origin · fetch · checkout 이다
#   IR             구울 태그.  SYNC_URL 에 있어야 한다
#   BUILD_A        구성 config-a 의 명령.  기본은 워크스페이스에 파일 하나를 쓰는 짧은 명령 — 자기 명령을 넣는다
#   BUILD_B        구성 config-b 의 명령.  예) BUILD_A='<your build command for config-a>'
#   SLICE_DIR      스크래치 자리.  기본 ${TMPDIR:-/tmp}/enode-slice-bake
#   PAUSE          0 이면 Enter 를 기다리지 않는다
#
# 허용 표지 — <WS>/.enode-disposable 이 보통 파일로 있어야 돈다.  없으면 운영 lower 로 보고 멈춘다.
#   touch "$WS/.enode-disposable"      버려도 되는 lower 에만 둔다

set -euo pipefail

: "${M:?set M to the mediator address}"
: "${T:?set T to the bootstrap token}"
: "${WS:?set WS to the workspace (lower root) of the baking node}"
: "${SIBLING_WS:?set SIBLING_WS to the workspace of a sibling node on the same lower (a bind alias of WS)}"
: "${SYNC_URL:?set SYNC_URL to the git repository the sync fetches from}"
: "${IR:?set IR to a tag in SYNC_URL}"
BUILD_A=${BUILD_A:-"mkdir -p .slice-bake && printf 'config-a\\n' > .slice-bake/config-a"}
BUILD_B=${BUILD_B:-"mkdir -p .slice-bake && printf 'config-b\\n' > .slice-bake/config-b"}
SLICE_DIR=${SLICE_DIR:-${TMPDIR:-/tmp}/enode-slice-bake}
BIN=$SLICE_DIR/bin
STAMP=$(date +%s)
SLICE=${SLICE:-slice}
LOWERS=${LOWERS:-$HOME/.local/state/enode/lowers}

for tool in go jq curl git stat unshare flock; do
  command -v "$tool" >/dev/null || { echo "$SLICE: $tool is not installed"; exit 1; }
done
git config --get user.email >/dev/null ||
  { echo "$SLICE: runctl needs git config user.email (or GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=user.email GIT_CONFIG_VALUE_0=<you>)"; exit 1; }

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

# 대상 — 경로 · 장치 (maj:min) · inode.  사람이 이 셋으로 lower 를 알아본다
where() { stat -L -c '%n  device %Hd:%Ld  inode %i' "$1"; }

# lower 의 상태 자리 — 키는 statfs 의 fsid (stat -f -c %i 와 같은 글자) 와 inode 다
lower_key() { printf '%s-%s' "$(stat -f -L -c %i "$1")" "$(stat -L -c %i "$1")"; }
state_dir() { printf '%s/%s' "$LOWERS" "$(lower_key "$WS")"; }
state_json() { cat "$(state_dir)/state.json" 2>/dev/null || echo '{"phase":"committed"}'; }
phase() { state_json | jq -r '.phase'; }

check_target() {
  echo "== target =="
  echo "   baking node workspace   $(where "$WS")"
  echo "   sibling workspace       $(where "$SIBLING_WS")"
  if [ "${WS%/}" = "${SIBLING_WS%/}" ]; then
    echo "$SLICE: WS and SIBLING_WS are the same path; the contract cannot tell the two nodes apart; stopping"
    exit 1
  fi
  if [ "$(stat -L -c '%d %i' "$WS")" != "$(stat -L -c '%d %i' "$SIBLING_WS")" ]; then
    echo "$SLICE: SIBLING_WS is not the same directory as WS; stopping"
    exit 1
  fi
  if [ ! -f "$WS/.enode-disposable" ] || [ -L "$WS/.enode-disposable" ]; then
    echo "$SLICE: $WS has no .enode-disposable regular file; treating it as a lower in use; stopping"
    exit 1
  fi
  echo "   the sibling workspace is $(sibling_kind)"
  echo "   the marker .enode-disposable is there.  It travels with a copy of the lower: a disposable lower copied to"
  echo "   a production place carries it too; check the device and inode above are the ones you meant"
}

# same_mount 는 SIBLING_WS 가 WS 와 같은 마운트로 lower 에 닿는지다 — symlink 면 참 · bind 별칭이면 거짓.
same_mount() { [ "$(stat -L -c %m "$WS")" = "$(stat -L -c %m "$SIBLING_WS")" ]; }
sibling_kind() {
  if same_mount; then
    echo "on the same mount as WS (a symlink or the same mount): the sibling can resume the baking node's merge"
  else
    echo "on another mount (a bind alias): the sibling cannot resume the baking node's merge (the merge preflight refuses it)"
  fi
}

# 같은 lower 의 노드가 모두 workspace.writes=isolated 여야 돈다 — 옛 판 노드는 그 키가 없고 native 는 in-place 다.
# machine 이 이 기계이거나 machine 이 없는 노드마다 ws 를 이 기계에서 stat 한다. stat 을 못 하면 같은 lower 인지 몰라
# 멈춘다.
check_same_lower() {
  echo "== nodes on this lower =="
  local host want nodes line id ws machine writes got bad=0
  host=$(hostname)
  want=$(stat -L -c '%d %i' "$WS")
  nodes=$(api "$M/v1/nodes" | jq -r '.nodes[] | ((.capabilities // []) | map(.attrs // {}) | add // {}) as $a |
    [.node_id, ($a.ws // ""), ($a.machine // ""), ($a["workspace.writes"] // "")] | @tsv')
  while IFS=$'\t' read -r id ws machine writes; do
    [ -n "$id" ] || continue
    [ -z "$machine" ] || [ "$machine" = "$host" ] || continue
    [ -n "$ws" ] || continue
    if ! got=$(stat -L -c '%d %i' "$ws" 2>/dev/null); then
      echo "$SLICE: node $id advertises ws $ws, which this machine cannot stat; cannot tell whether it is this lower; stopping"
      exit 1
    fi
    [ "$got" = "$want" ] || continue
    case $writes in
      isolated) echo "   $id  ws $ws  workspace.writes=isolated" ;;
      "") echo "   $id  ws $ws  advertises no workspace.writes: an older enode is on this lower"; bad=1 ;;
      *) echo "   $id  ws $ws  workspace.writes=$writes: a native node is on this lower"; bad=1 ;;
    esac
  done <<<"$nodes"
  if [ "$bad" != 0 ]; then
    echo "$SLICE: move every node on this lower to a build that bakes before baking; stopping"
    exit 1
  fi
  echo "   not checked here; check them yourself:"
  echo "     the build each node runs: isolated is also advertised by builds that cannot bake (the lower-state build)"
  echo "     nodes on this lower that talk to another mediator, nodes that are off, nodes started after this check"
  echo "   do not start another node on this lower while the slice runs"
}

build_runctl() {
  mkdir -p "$BIN"
  go build -o "$BIN/runctl" ./cmd/runctl
}

# bake_contract 는 runctl example bake 모양의 계약이다 — build 에 produced [manifest], merge 에 produced [merged].
# ws 로 노드를 고른다. sync 는 SYNC_URL 에서 $ENODE_IR 을 받아 detached 로 둔다 — origin 을 두어 metadata 의 url 과
# repo_id 가 찬다. 뿌리를 비울 것을 기대하지 않는다 (git clean -x 는 표지를 지운다).
#
#   bake_contract <run> <ws> <merge.wait> <build a> <build b> [ir] [tag the sync checks out]
# 마지막 칸을 주면 sync 가 $ENODE_IR 대신 그 태그로 간다 — IR 이 어긋나는 굽기를 짓는다.
bake_contract() {
  local id=$1 ws=$2 wait=$3 build_a=$4 build_b=$5 ir=${6:-$IR} to=${7:-}
  local ref='$ENODE_IR'
  [ -z "$to" ] || ref=$to
  local sync="git init -q . && { git remote get-url origin >/dev/null 2>&1 || git remote add origin '$SYNC_URL'; }"
  sync="$sync && git fetch -q origin \"+refs/tags/$ref:refs/tags/$ref\" && git checkout -q --detach \"$ref\""
  jq -n --arg id "$id" --arg ws "$ws" --arg wait "$wait" --arg sync "$sync" --arg a "$build_a" --arg b "$build_b" \
    --arg ir "$ir" '{
    run_id: $id,
    work: { id: { system: "manual", change_id: $id }, system: "manual" },
    requires: [ { as: "baker", capability: "agent.reason", "workspace.writes": "isolated", ws: $ws } ],
    steps: [
      { id: "build", uses: "baker", effect: "prepare", ir: $ir, sync: $sync,
        builds: [ { name: "config-a", command: $a }, { name: "config-b", command: $b } ] },
      { id: "merge", uses: "baker", needs: ["build"], merge: { wait: $wait } }
    ],
    success_when: [ { step: "build", produced: ["manifest"] }, { step: "merge", produced: ["merged"] } ]
  }'
}

# plain_contract 는 형제 노드에서 오래 도는 명령 단계 하나다.
plain_contract() {
  local id=$1 ws=$2 seconds=$3
  jq -n --arg id "$id" --arg ws "$ws" --arg s "sleep $seconds" '{
    run_id: $id,
    work: { id: { system: "manual", change_id: $id }, system: "manual" },
    requires: [ { as: "node", capability: "agent.reason", ws: $ws } ],
    steps: [ { id: "long", uses: "node", run: ["sh", "-c", $s] } ],
    success_when: [ { step: "long", exit_code: 0 } ]
  }'
}

submit_json() {
  local id=$1
  cat > "$SLICE_DIR/$id.json"
  ENODE_MEDIATOR=$M ENODE_TOKEN=$T "$BIN/runctl" submit "$SLICE_DIR/$id.json" > /dev/null
}

run_state() { api "$M/v1/runs/$1" | jq -r '.state'; }

wait_run() {
  local id=$1
  for _ in $(seq 1 36000); do
    case "$(run_state "$id")" in SUCCEEDED | FAILED | CANCELLED) return 0 ;; esac
    sleep 0.2
  done
  echo "$SLICE: run $id did not end"
  exit 1
}

wait_phase() {
  local want=$1 tries=${2:-3000}
  for _ in $(seq 1 "$tries"); do
    [ "$(phase)" = "$want" ] && return 0
    sleep 0.1
  done
  echo "$SLICE: the lower never reached $want (now $(phase))"
  exit 1
}

# steps 는 Record 의 단계마다 state · exit_code · error · reason 과 굽기 칸이다. lower_changed 가 보이면 다시 내면 된다.
steps() {
  local id=$1 f
  record "$id"
  echo "   run $id: $(run_state "$id")"
  for f in "$SLICE_DIR/$id"/*/steps/*.json; do
    jq -c '{step: .name, state, exit_code: .result.exit_code, error: .result.error, reason: .result.reason,
            upload: .result.upload, head: .result.build.head, head_tags: .result.build.head_tags,
            merge: .result.merge}' "$f" | sed 's/^/     /'
  done
  if cat "$SLICE_DIR/$id"/*/steps/*.json | jq -e -s 'map(.result.reason // empty) | index("lower_changed")' >/dev/null; then
    echo "   a step says lower_changed: the lower was merged after this run was matched; submit the run again"
  fi
}

# step_log 은 단계 로그다 (seq 1 은 build · 2 는 merge). 노드는 단계 이름으로 로그를 올리므로 name 을 준다 —
# 없으면 Mediator 는 기본 이름 step 의 빈 로그를 돌려준다.
step_log() {
  local name=build
  [ "$2" = 1 ] || name=merge
  api "$M/v1/runs/$1/steps/$2/log?name=$name"
}

# record 는 Run 의 Record 를 받아 풀어 둔다.
record() {
  local id=$1
  for _ in $(seq 1 60); do
    ENODE_MEDIATOR=$M ENODE_TOKEN=$T "$BIN/runctl" record "$id" -o "$SLICE_DIR/$id.tar" > /dev/null 2>&1 && break
    sleep 1
  done
  mkdir -p "$SLICE_DIR/$id" && tar -xf "$SLICE_DIR/$id.tar" -C "$SLICE_DIR/$id"
}

metadata() { jq '{bake, source, builds: [.builds[].name], fields: keys}' "$WS/.enode-metadata.json"; }

# hold_lower 는 lower 공유 잠금을 쥔다 — 기록이 없는 쥔 쪽 (smoke 모양) 이다. merge 가 배타를 기다리게 해 합치기
# 전의 merged view 를 볼 자리를 만든다. release_lower 가 놓는다.
#
# -o 로 잠금 fd 를 자식 (sleep) 에 넘기지 않는다. 넘기면 flock 만 죽여도 남은 sleep 이 공유를 계속 쥐어 merge 가
# 합치기에 들어가지 못한다. 놓을 때는 자식 sleep 도 함께 끝낸다.
hold_lower() {
  flock -s -o "$(state_dir)/lower.lock" sleep 86400 &
  HOLD_PID=$!
  sleep 0.5
}
release_lower() {
  if [ -n "${HOLD_PID:-}" ]; then
    pkill -P "$HOLD_PID" 2>/dev/null || true
    kill "$HOLD_PID" 2>/dev/null || true
    wait "$HOLD_PID" 2>/dev/null || true
    HOLD_PID=""
  fi
}

# listing 은 디렉터리의 목록이다 — 경로 · 종류 · 권한.  .enode-metadata.json 은 뺀다 (합치기의 마지막 동작으로 새로 쓰인다)
listing() {
  (cd "$1" && find . -mindepth 1 ! -path './.enode-metadata.json' ! -path './.enode-metadata.json.*' -printf '%P %y %m\n' | sort)
}

# merged_view 는 합치기 전의 merged view 목록이다 — user namespace 의 읽기 전용 overlay (upper 위 · lower 아래).
# upper 의 whiteout 과 opaque 는 overlay 가 풀어 보인다.
merged_view() {
  local upper=$1 mnt=$SLICE_DIR/view
  mkdir -p "$mnt"
  unshare --user --map-root-user --map-auto --mount sh -c '
    mount -t overlay overlay -o ro,lowerdir="$1":"$2" "$3" &&
    cd "$3" && find . -mindepth 1 ! -path "./.enode-metadata.json" ! -path "./.enode-metadata.json.*" -printf "%P %y %m\n" | sort
  ' sh "$upper" "$WS" "$mnt"
}

daemon_pid() {
  local config=$1
  head -1 "$config.lock"
}

trap release_lower EXIT
