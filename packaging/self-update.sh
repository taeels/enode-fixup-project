#!/usr/bin/env bash
# enode 를 스스로 갱신한다 (ADR-056).
#
#   self-update.sh <묶음 URL> <설정 이름> [기대 커밋]
#   self-update.sh http://192.168.219.203:8088/enode-macos-abc123.tar.gz local abc123
#
# 왜 필요한가 — 맥은 ssh 가 닫혀 있고 사람이 늘 앞에 있지 않다. 그런데
# 노드를 고치려면 바이너리를 바꾸고 프로세스를 다시 띄워야 한다.
# 그 일을 그 노드 자신이 돌리는 명령 단계로 한다.
#
# 이 스크립트의 어려운 점은 교체가 아니라 되돌리기다 —
# 새 것이 안 뜨면 노드가 사라지고, 사람이 돌아올 때까지 함대가 반쪽이 된다.
set -euo pipefail

URL=${1:?묶음 URL 이 필요하다}
NAME=${2:-local}
WANT=${3:-}

CONFDIR=${ENODE_CONFDIR:-$HOME/.config/enode}
BINDIR=${ENODE_BINDIR:-$HOME/.local/bin}
STATEDIR=${ENODE_STATEDIR:-$HOME/.local/state/enode}
CFG="$CONFDIR/$NAME.yaml"
LOG="$STATEDIR/$NAME.log"

say() { printf '%s\n' "$*"; }
die() { printf '실패 %s\n' "$*" >&2; exit 1; }

# node_pid 는 그 설정으로 도는 노드의 pid다. 없으면 빈 문자열이다.
#
# pgrep 을 안 쓴다 — BSD(macOS) 의 pgrep 은 자기 조상을 기본으로 제외한다.
# 그런데 이 스크립트는 노드가 run 단계로 돌리므로 언제나 그 노드의 자식이다.
# ⇒ 맥에서는 자기를 띄운 노드를 영원히 못 찾는다.
#
#	실측 (probe-pgrep-1) ps 에는 7360(local)과 7573(qemu)이 둘 다 보이는데
#	같은 프로세스에서 부른 pgrep -f 'enode --config' 는 7573 만 돌려줬다.
#	7360 은 이 명령을 돌리는 셸의 부모였다.
#
# 리눅스 procps 는 조상 제외가 -A 로 선택 사항이라 거기서는 우연히 맞았다.
# ⇒ 우연히 맞는 것을 고쳐 쓰지 않는다. ps 는 두 곳에서 같게 동작한다.
#
# awk 자신은 안 걸린다 — command 의 첫 토큰($2)이 실행파일 경로와 같아야
# 하는데 awk 행의 첫 토큰은 awk 다. grep 의 [e]node 관용구가 필요 없다.
node_pid() {
  ps -eo pid=,command= | awk -v bin="$1" -v cfg="$2" '
    $2 == bin && index($0, "--config " cfg) { print $1; exit }'
}

[ -f "$CFG" ] || die "설정이 없다: $CFG"

# 이 기계가 무엇인지 스스로 알아낸다
#
# 처음에는 맥 전용이었다. 탄력 노드는 리눅스에 뜬다 —
# colima VM · 클라우드 장비 · 컨테이너. 같은 스크립트가 거기서도 돌아야 한다.
# 묶음이 네 플랫폼을 다 담으므로(build.sh TARGETS) 고를 것만 고르면 된다.
case "$(uname -s)" in
  Darwin) OS=darwin ;;
  Linux)  OS=linux ;;
  *) die "모르는 운영체제: $(uname -s)" ;;
esac
case "$(uname -m)" in
  arm64|aarch64) ARCH=arm64 ;;
  x86_64|amd64)  ARCH=amd64 ;;
  *) die "모르는 아키텍처: $(uname -m)" ;;
esac
PLAT="$OS-$ARCH"

# ── ① 받는다 ────────────────────────────────────────────────────────────
# 설치 경로 밖에서 푼다 — 검증을 통과하기 전에는 아무것도 안 건드린다.
WORK=$(mktemp -d "${TMPDIR:-/tmp}/enode-update.XXXXXX")
trap 'rm -rf "$WORK"' EXIT
say "== 받는다 =="
curl -fsSL --max-time 120 -o "$WORK/pkg.tar.gz" "$URL" || die "내려받지 못했다: $URL"
tar xzf "$WORK/pkg.tar.gz" -C "$WORK" || die "묶음을 못 푼다"
SRC=$(find "$WORK" -maxdepth 2 -type d -name 'bin' | head -1)
[ -n "$SRC" ] || die "묶음에 bin/ 이 없다"

# ── ② 검증한다 ──────────────────────────────────────────────────────────
# 실행해서 확인한다. 다만 --version 은 노드를 띄우지 않는다 —
# 설정도 안 읽고 미디에이터에도 안 붙는다. 그래서 안전하게 물어볼 수 있다.
say "== 검증 =="
NEW_ENODE="$SRC/enode-$PLAT"
[ -x "$NEW_ENODE" ] || die "이 기계용 실행파일이 없다: enode-$PLAT"
VER=$("$NEW_ENODE" --version 2>&1) || die "새 실행파일이 안 돈다: $VER"
say "   새 것 : $VER"
say "   지금  : $("$BINDIR/enode" --version 2>&1 || echo '(--version 이 없는 옛 것)')"

# 받아온 것이 이 기계용인가 — --version 이 "linux/amd64" 처럼 찍는다.
case "$VER" in
  *"$OS/$ARCH"*|*"$PLAT"*) ;;
  *) die "다른 기계용이다: $VER (이 기계는 $OS/$ARCH)" ;;
esac
if [ -n "$WANT" ]; then
  case "$VER" in
    *"$WANT"*) ;;
    *) die "기대한 커밋이 아니다: $WANT 를 기대했는데 $VER" ;;
  esac
fi

RUNNING=$(node_pid "$BINDIR/enode" "$CFG")
[ -n "$RUNNING" ] || say "   지금 도는 노드가 없다 — 교체만 하고 띄운다"

# ── ③ 바꾼다 ────────────────────────────────────────────────────────────
# 옛 것을 지우지 않는다 — .prev 로 남긴다. 되돌릴 것이 없으면 되돌릴 수 없다.
say "== 바꾼다 =="
mkdir -p "$BINDIR" "$STATEDIR"
for c in enode runctl enodectl; do
  [ -f "$SRC/$c-$PLAT" ] || continue
  [ -f "$BINDIR/$c" ] && cp -p "$BINDIR/$c" "$BINDIR/$c.prev"
  # mv 는 실행 중이어도 된다 — 도는 프로세스는 옛 inode 를 계속 쓴다.
  cp "$SRC/$c-$PLAT" "$BINDIR/$c.new"
  chmod +x "$BINDIR/$c.new"
  mv "$BINDIR/$c.new" "$BINDIR/$c"
  say "   $c ← $c-$PLAT"
done

# ── ④ 다시 띄운다 ── 이 단계가 끝난 뒤에 ───────────────────────────
# 지금 죽이면 이 명령 자신이 죽는다 — 그러면 단계가 실패로 보고되고,
# 무엇이 됐고 무엇이 안 됐는지가 기록에 안 남는다. 그래서 예약한다:
# 이 단계가 정상 종료하고 보고까지 마친 뒤에 교체가 일어난다.
#
# 되돌리기가 이 스크립트의 본체다 — 새 것이 안 뜨면 .prev 로 되살린다.
say "== 재시작을 예약한다 (${DELAY:=25}초 뒤) =="
cat > "$STATEDIR/$NAME-restart.sh" <<'INNER'
#!/bin/sh
set -u
NAME="$1"; CFG="$2"; BINDIR="$3"; STATEDIR="$4"; DELAY="$5"
LOG="$STATEDIR/$NAME.log"
# 여기서도 pgrep 을 안 쓴다 — 이 스크립트는 sleep 뒤에 고아가 되어
# 노드가 조상에서 빠지므로 오늘은 pgrep 으로도 맞는다. 그러나 그것은
# 타이밍에 기댄 것 이고, 같은 함정을 두 곳에 두지 않는다.
node_pid() {
  ps -eo pid=,command= | awk -v bin="$1" -v cfg="$2" '
    $2 == bin && index($0, "--config " cfg) { print $1; exit }'
}
sleep "$DELAY"
start() {
  cd "$HOME" || exit 1
  nohup "$BINDIR/enode" --config "$CFG" >> "$LOG" 2>&1 &
  echo "$!" > "$STATEDIR/$NAME.newpid"
}
{
  echo "===== $(date '+%Y-%m-%d %H:%M:%S') 자기 갱신 재시작 ====="
  OLD=$(node_pid "$BINDIR/enode" "$CFG")
  if [ -n "$OLD" ]; then
    echo "옛 프로세스 $OLD 를 멈춘다"
    kill "$OLD" 2>/dev/null
    i=0; while [ $i -lt 15 ] && kill -0 "$OLD" 2>/dev/null; do sleep 1; i=$((i+1)); done
    kill -0 "$OLD" 2>/dev/null && kill -9 "$OLD" 2>/dev/null
  fi
  start
  sleep 8
  NEW=$(cat "$STATEDIR/$NAME.newpid" 2>/dev/null)
  if [ -n "$NEW" ] && kill -0 "$NEW" 2>/dev/null; then
    echo "새 노드가 떴다 pid=$NEW        $("$BINDIR/enode" --version 2>&1)"
    exit 0
  fi
  # 되돌린다 — 새 것이 안 뜨면 노드가 사라진다. 그것이 최악이다.
  echo "새 노드가 안 떴다 — 되돌린다"
  if [ -f "$BINDIR/enode.prev" ]; then
    cp "$BINDIR/enode.prev" "$BINDIR/enode.rollback" && chmod +x "$BINDIR/enode.rollback"
    mv "$BINDIR/enode.rollback" "$BINDIR/enode"
    [ -f "$BINDIR/runctl.prev" ] && cp -p "$BINDIR/runctl.prev" "$BINDIR/runctl"
    [ -f "$BINDIR/enodectl.prev" ] && cp -p "$BINDIR/enodectl.prev" "$BINDIR/enodectl"
    start
    sleep 5
    NEW=$(cat "$STATEDIR/$NAME.newpid" 2>/dev/null)
    if [ -n "$NEW" ] && kill -0 "$NEW" 2>/dev/null; then
      echo "옛 것으로 되살렸다 pid=$NEW"
    else
      echo " 되돌리기도 실패했다 — 사람이 봐야 한다 "
    fi
  else
    echo " .prev 가 없어 되돌릴 수 없다 "
  fi
} >> "$STATEDIR/$NAME-update.log" 2>&1
INNER
chmod +x "$STATEDIR/$NAME-restart.sh"

# 부모에서 완전히 떼어낸다 — 이 단계가 끝나 프로세스 그룹이 정리돼도
# 살아남아야 한다. 그래서 setsid 대신 nohup + 리다이렉트 전부를 쓴다.
nohup "$STATEDIR/$NAME-restart.sh" "$NAME" "$CFG" "$BINDIR" "$STATEDIR" "$DELAY" \
  >/dev/null 2>&1 < /dev/null &
disown 2>/dev/null || true

say ""
say "예약됨. ${DELAY}초 뒤에 노드가 새 바이너리로 다시 뜬다."
say "확인:  $STATEDIR/$NAME-update.log"
say "되돌리기가 필요하면:  $BINDIR/enode.prev 가 남아 있다"
say "새 버전: $VER"
