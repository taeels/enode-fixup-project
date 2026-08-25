#!/usr/bin/env bash
# 맥에서 돌린다. 실행파일을 놓고, 설정 자리를 만들고, 프리플라이트를 잰다.
#
#   ./install.sh                 ~/.local/bin 에 설치
#   PREFIX=/usr/local ./install.sh
#
# 아무것도 기동하지 않는다 — 기동은 enodectl 의 몫이고, 그래야 설치와
# 「어떤 노드로 설 것인가」가 갈린다. 노드는 여럿일 수 있다 (설정 파일이 곧 신원이다).
set -euo pipefail

HERE=$(cd "$(dirname "$0")" && pwd)
PREFIX=${PREFIX:-$HOME/.local}
BINDIR="$PREFIX/bin"
CONFDIR=${ENODE_CONFDIR:-$HOME/.config/enode}
STATEDIR=${ENODE_STATEDIR:-$HOME/.local/state/enode}

say()  { printf '%s\n' "$*"; }
warn() { printf '  ▲ %s\n' "$*" >&2; }
die()  { printf '  ✗ %s\n' "$*" >&2; exit 1; }
ok()   { printf '  ○ %s\n' "$*"; }

say "== 1. 이 기계 =="
[ "$(uname -s)" = "Darwin" ] || die "맥이 아니다 ($(uname -s)). 이 스크립트는 맥에서 돌린다."
UNAME_M=$(uname -m)
case "$UNAME_M" in
  arm64)  GOARCH=arm64 ;;
  x86_64) GOARCH=amd64 ;;
  *) die "모르는 아키텍처다: $UNAME_M" ;;
esac
ok "macOS $(sw_vers -productVersion) · $UNAME_M → darwin/$GOARCH"

# 로제타 위에서 도는 셸을 잡아낸다 — arm64 맥에서 x86_64 터미널을 쓰면
# uname 이 x86_64 를 말해 amd64 바이너리가 깔리고, 그러면 느린데 이유가 안 보인다.
if [ "$GOARCH" = "amd64" ] && [ "$(sysctl -n sysctl.proc_translated 2>/dev/null || echo 0)" = "1" ]; then
  warn "이 셸이 로제타 위에서 돈다. 실제 CPU 는 애플 실리콘이다."
  warn "  네이티브 터미널에서 다시 돌리면 arm64 가 깔린다 (arch -arm64 zsh)."
fi

say
say "== 2. 프리플라이트 — 여기서 안 걸리면 맥에서 조용히 실패한다 =="

# ① git 이메일. 없으면 enode 가 그 자리에서 죽는다 (ADR-015 §1, ErrNoEmail).
if EMAIL=$(git config --global --get user.email 2>/dev/null) && [ -n "$EMAIL" ]; then
  ok "git 이메일: $EMAIL"
else
  die "git --global user.email 이 없다. enode 는 조용한 대체를 안 하고 죽는다.
      git config --global user.email 'you@example.com'"
fi

# ② hostname 안정성 — 맥 고유의 함정이다
#    node_id = hash(email ∥ hostname ∥ realpath(config)) 인데, 맥은 HostName 을
#    안 박아두면 gethostname() 이 Bonjour 이름·DHCP 로 흔들린다. 네트워크가 바뀌면
#    같은 기계가 다른 노드가 되고 유령 노드가 쌓인다 (ADR-015 §2 가 무작위
#    UUID 를 기각한 것과 똑같은 실패다).
HN=$(hostname)
if scutil --get HostName >/dev/null 2>&1; then
  ok "HostName 이 고정돼 있다: $(scutil --get HostName)  (hostname → $HN)"
else
  warn "HostName 이 안 박혀 있다. 지금 hostname 은 '$HN' 이고 네트워크가"
  warn "  바뀌면 따라 바뀐다 → node_id 가 바뀐다 → 유령 노드가 쌓인다."
  warn "  고정: sudo scutil --set HostName $(echo "$HN" | sed 's/\.local$//')"
fi

# ③ 하네스. 없으면 광고에 harness 가 안 실리고 → 후보에서 빠지고 → 계약이 422 다.
#    여기서 못 찾는 것과 enode 가 못 찾는 것은 다르다 — enode 의 PATH 는
#    그것을 띄운 셸의 PATH 이므로 enodectl 이 로그인 셸에서 띄우는 이유가 이것이다.
if CLAUDE=$(command -v claude 2>/dev/null); then
  ok "하네스: $CLAUDE ($(claude --version 2>/dev/null | head -1))"
else
  warn "PATH 에 claude 가 없다. 이 노드는 harness 속성을 광고하지 못한다."
  warn "  에이전트 단계를 요구하는 계약은 이 노드를 못 고른다 (422)."
fi

# ④ 시스템 잠자기 — 함대를 통째로 끊는다
#    디스플레이가 꺼지는 것은 상관없다. 시스템이 자면 광고가 멈추고
#    not_after 가 지나 그 노드를 쥔 Run 이 죽는다. 실측에서 밟았다.
#    enodectl start 가 caffeinate 를 함께 띄우지만, 영구 설정은 사람 몫이다.
if command -v caffeinate >/dev/null 2>&1; then
  ok "caffeinate 가 있다 — enodectl start 가 잠자기를 막아준다"
else
  warn "caffeinate 가 없다 — 시스템이 자면 함대에서 사라진다"
fi
SLEEP=$(pmset -g custom 2>/dev/null | awk '/^AC Power/,0' | awk '$1=="sleep"{print $2; exit}')
if [ -n "$SLEEP" ] && [ "$SLEEP" != "0" ]; then
  warn "AC 전원의 시스템 잠자기가 ${SLEEP}분이다. 상시 노드로 쓸 것이면:"
  warn "  sudo pmset -c sleep 0        # 시스템은 안 잔다"
  warn "  sudo pmset -c displaysleep 10 # 화면은 꺼도 된다"
fi
if [ "$(pmset -g 2>/dev/null | awk '$1=="tcpkeepalive"{print $2}')" = "0" ]; then
  warn "tcpkeepalive 가 0 이다 — 잠자기 중 네트워크가 통째로 끊긴다"
  warn "  sudo pmset -a tcpkeepalive 1"
fi

# ⑤ 크로스 툴체인. enode 의 자동 탐지는 arm-linux-gnueabihf-gcc /
#    aarch64-linux-gnu-gcc 둘만 본다 — 맥의 Zephyr SDK 는 여기 안 걸린다.
#    그래서 맥 노드는 설정에 arch 를 명시한다 (Local.Arch 가 그 자리다).
if command -v arm-zephyr-eabi-gcc >/dev/null 2>&1; then
  ok "Zephyr 툴체인: $(command -v arm-zephyr-eabi-gcc)"
  warn "  자동 탐지는 이 이름을 모른다. 설정에 arch: 를 손으로 적어야 한다."
elif command -v arm-linux-gnueabihf-gcc >/dev/null 2>&1 || command -v aarch64-linux-gnu-gcc >/dev/null 2>&1; then
  ok "크로스 툴체인이 자동 탐지 범위 안에 있다"
else
  warn "크로스 툴체인이 없다. 빌드 능력을 광고하려면 설정에 arch: 를 적는다."
fi

say
say "== 3. 설치 =="
mkdir -p "$BINDIR" "$CONFDIR" "$STATEDIR"

for c in enode runctl enodectl; do
  src="$HERE/bin/$c-darwin-$GOARCH"
  [ -f "$src" ] || die "묶음에 $c-darwin-$GOARCH 가 없다"
  install -m 0755 "$src" "$BINDIR/$c"
  ok "$BINDIR/$c"
done

# 예시는 하위 디렉터리에 둔다 — $CONFDIR 에 바로 풀면 안 고른 노드까지
# enodectl list 에 뜬다. 특히 colima.yaml 은 맥이 아니라 VM 안에 놓일 파일이라
# 맥의 노드 목록에 있으면 그 자체가 틀린 그림이다.
# 노드를 세우는 것은 고르는 일이지 설치의 부수효과가 아니다.
mkdir -p "$CONFDIR/examples"
for f in "$HERE"/examples/*.yaml; do
  install -m 0600 "$f" "$CONFDIR/examples/$(basename "$f")"
done
ok "$CONFDIR/examples/  ($(ls "$HERE"/examples/*.yaml | wc -l | tr -d ' ') 개)"

# 이미 만든 설정은 절대 안 건드린다 — 설정 파일이 신원이라 덮어쓰면
# node_id 는 그대로인데 내용이 바뀌어 다른 것을 광고하는 같은 노드가 된다.
for f in "$CONFDIR"/*.yaml; do
  [ -e "$f" ] || continue
  ok "그대로 둔다: $f"
done

say
say "== 4. 실행 확인 =="
# 서명이 안 붙었으면 여기서 killed: 9 로 죽는다 — 설치 시점에 걸러낸다.
#
# 종료코드가 아니라 우리 코드가 낸 메시지를 본다. 서명이 없으면 커널이
# SIGKILL 로 죽여서 stdout·stderr 가 통째로 비고, 종료코드만 보면 그것과
# "설정이 없다" 를 구별할 수 없다.
PROBE=$("$BINDIR/enode" --config /nonexistent/enode-probe.yaml 2>&1 || true)
if printf '%s' "$PROBE" | grep -q '설정을 읽을 수 없다'; then
  ok "실행된다 (서명 문제 없음)"
else
  warn "enode 가 낸 것: ${PROBE:-<아무것도 안 냈다>}"
  die "실행이 안 된다. 아무것도 안 냈으면 서명 문제다 (killed: 9):
      codesign -s - $BINDIR/enode $BINDIR/runctl $BINDIR/enodectl"
fi

case ":$PATH:" in
  *":$BINDIR:"*) ok "PATH 에 $BINDIR 가 있다" ;;
  *) warn "PATH 에 $BINDIR 가 없다. 셸 설정에 추가하라:"
     warn "  echo 'export PATH=\"$BINDIR:\$PATH\"' >> ~/.zshrc" ;;
esac

say
say "== 다음 =="
say "  ① 예시에서 하나 골라 노드를 만든다 — 파일 이름이 노드 이름이다"
say "       cp $CONFDIR/examples/zephyr.yaml $CONFDIR/zephyr.yaml"
say "       vi $CONFDIR/zephyr.yaml          mediator · token · workspace"
say "  ② enodectl id zephyr          띄우기 전에 node_id 를 안다"
say "     enodectl start zephyr"
say "  ③ enodectl status"
say "  ④ CT103 에서: runctl capabilities   ← 맥이 함대에 보이는지"
