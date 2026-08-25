#!/usr/bin/env bash
# 설치본을 굽는다 — 전부 리눅스에서.
#
#   packaging/build-installers.sh
#   PKG_VERSION=1.2.3 packaging/build-installers.sh
#   TARGETS="linux/amd64" packaging/build-installers.sh
#
# 결과는 dist/installers/ 에 쌓인다.
#
# 두 패키지로 가른다
#
#   enode           enode · runctl · enodectl    노드는 여럿이다
#   enode-mediator  mediator                     시퀀서는 한 자리다 (ADR-002)
#
# 섞으면 노드 50대에 시퀀서 실행파일이 따라 들어가고, 더 나쁜 것은 「이 기계에
# Mediator 가 깔려 있다」가 사실이 아닌 채로 참이 된다 — 나중에 "여기서 뭐가
# 도나" 를 패키지 목록으로 물을 수 없다.
#
# 대상은 둘뿐이다 — linux/amd64 와 windows/amd64.
#
# 진짜 이유 — 맥과 arm 은 안 되는 것이 아니라 다른 통로가 이미 있다:
# packaging/macos/build.sh 가 darwin/arm64 · darwin/amd64 · linux/arm64 ·
# linux/amd64 묶음을 굽고 있고, 맥 노드와 colima 안의 노드는 그것을 쓴다.
# 여기에 또 만들면 같은 것을 두 곳에서 굽게 되고, 두 곳이 어긋나면 어느
# 쪽이 참인지 알 수 없다.
#
# 여는 조건 — 맥이나 리눅스 ARM 을 패키지 관리자로 깔아야 하는 사람이
# 생길 때. 그때 packaging/macos/pkg.sh 를 git 이력에서 되살리면 된다 —
# 지웠지만 v0.0.1-rc4 에서 실제로 통과한 물건이다.
set -euo pipefail

cd "$(dirname "$0")/.."
ROOT=$(pwd)
OUT="$ROOT/dist/installers"
STAGE="$ROOT/dist/stage"

COMMIT=$(git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)
PKG_VERSION=${PKG_VERSION:-"0.0.0"}
# v 접두사는 태그의 것이지 판 번호의 것이 아니다 — 파일 이름에 남으면
# enode-v0.0.1-... 처럼 어색하고, deb·rpm 은 어차피 nfpm 이 떼어내서
# 같은 릴리즈 안에서 이름이 두 갈래가 된다. 여기서 한 번에 뗀다.
PKG_VERSION=${PKG_VERSION#v}
# MSI 는 x.y.z 만 받는다 — 뒤에 붙은 -rc1 을 그대로 주면 거절한다.
MSI_VERSION=$(printf '%s' "$PKG_VERSION" | grep -oE '^[0-9]+(\.[0-9]+){0,2}' || true)
MSI_VERSION=${MSI_VERSION:-0.0.0}

BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS="-X github.com/taeels/enode/internal/build.Commit=$COMMIT"
LDFLAGS="$LDFLAGS -X github.com/taeels/enode/internal/build.Date=$BUILD_DATE"
# 태그에서 지은 것만 Release 를 채운다 — 안 채우면 오늘 그대로 커밋이 신원이다.
if [ "$PKG_VERSION" != "0.0.0" ]; then
  LDFLAGS="$LDFLAGS -X github.com/taeels/enode/internal/build.Release=$PKG_VERSION"
fi

CMDS_NODE=${CMDS_NODE:-"enode runctl enodectl"}
CMDS_MEDIATOR=${CMDS_MEDIATOR:-"mediator"}
TARGETS=${TARGETS:-"linux/amd64 windows/amd64"}

rm -rf "$OUT" "$STAGE"
mkdir -p "$OUT"

echo "== 크로스 빌드 =="
for t in $TARGETS; do
  goos=${t%%/*}; goarch=${t##*/}
  d="$STAGE/$goos-$goarch"
  mkdir -p "$d"
  ext=""; [ "$goos" = "windows" ] && ext=".exe"
  for c in $CMDS_NODE $CMDS_MEDIATOR; do
    echo "   $goos/$goarch  $c"
    env GOOS=$goos GOARCH=$goarch CGO_ENABLED=0 \
      go build -trimpath -ldflags "$LDFLAGS" -o "$d/$c$ext" "./cmd/$c"
  done
done

# ── 원본 묶음 ────────────────────────────────────────────────────────────
# 설치본이 아닌 것도 낸다 — 컨테이너 안에서는 패키지 관리자를 거치는 것이
# 오히려 번거롭고 손으로 밀어 넣는 자리가 있다. 여기서도 둘로 가른다.
echo "== 묶음 =="
for d in "$STAGE"/*-amd64; do
  [ -d "$d" ] || continue
  n=$(basename "$d")
  ext=""; case "$n" in windows-*) ext=".exe" ;; esac
  for pkg in enode enode-mediator; do
    case "$pkg" in
      enode)          cmds="$CMDS_NODE" ;;
      enode-mediator) cmds="$CMDS_MEDIATOR" ;;
    esac
    b="$STAGE/$pkg-$PKG_VERSION-$n"
    rm -rf "$b"; mkdir -p "$b"
    for c in $cmds; do cp "$d/$c$ext" "$b/"; done
    case "$n" in
      windows-*) ( cd "$STAGE" && zip -qr "$OUT/$(basename "$b").zip" "$(basename "$b")" ) ;;
      *)         ( cd "$STAGE" && tar czf "$OUT/$(basename "$b").tar.gz" "$(basename "$b")" ) ;;
    esac
    echo "   $pkg  $n"
  done
done

# ── deb · rpm ────────────────────────────────────────────────────────────
if command -v nfpm >/dev/null 2>&1; then
  echo "== deb · rpm =="
  a=amd64
  if [ -d "$STAGE/linux-$a" ]; then
    conf="$STAGE/nfpm-enode.yaml"
    sed -e "s|@ARCH@|$a|g" -e "s|@VERSION@|$PKG_VERSION|g" \
        -e "s|@BIN@|$STAGE/linux-$a|g" \
        -e "s|@EXAMPLES@|$ROOT/packaging/macos/examples|g" \
        "$ROOT/packaging/linux/nfpm.yaml.in" > "$conf"
    nfpm package -f "$conf" -p deb -t "$OUT" >/dev/null
    nfpm package -f "$conf" -p rpm -t "$OUT" >/dev/null
    echo "   enode           linux/$a"

    mconf="$STAGE/nfpm-mediator.yaml"
    sed -e "s|@ARCH@|$a|g" -e "s|@VERSION@|$PKG_VERSION|g" \
        -e "s|@BIN@|$STAGE/linux-$a|g" \
        -e "s|@MEDIATOR_EXAMPLE@|$ROOT/packaging/examples/mediator.yaml|g" \
        "$ROOT/packaging/linux/nfpm-mediator.yaml.in" > "$mconf"
    nfpm package -f "$mconf" -p deb -t "$OUT" >/dev/null
    nfpm package -f "$mconf" -p rpm -t "$OUT" >/dev/null
    echo "   enode-mediator  linux/$a"
  fi
else
  echo "== deb · rpm 건너뜀 (nfpm 이 없다) =="
fi

# ── msi ──────────────────────────────────────────────────────────────────
if command -v wixl >/dev/null 2>&1; then
  echo "== msi =="
  a=amd64
  if [ -d "$STAGE/windows-$a" ]; then
    # wixl 은 Source 에 절대경로를 안 받는다 (실측: g_file_get_child
    # assertion '!g_path_is_absolute'). 그래서 스테이지로 들어가 상대경로로 준다.
    ( cd "$STAGE" && wixl -a x64 \
        -D Version="$MSI_VERSION" -D BinDir="windows-$a" \
        -o "$OUT/enode-$PKG_VERSION-windows-$a.msi" \
        "$ROOT/packaging/windows/enode.wxs" )
    echo "   enode           windows/$a"
    ( cd "$STAGE" && wixl -a x64 \
        -D Version="$MSI_VERSION" -D BinDir="windows-$a" \
        -o "$OUT/enode-mediator-$PKG_VERSION-windows-$a.msi" \
        "$ROOT/packaging/windows/enode-mediator.wxs" )
    echo "   enode-mediator  windows/$a"
  fi
else
  echo "== msi 건너뜀 (wixl 이 없다 — apt install wixl) =="
fi

# ── 체크섬 ───────────────────────────────────────────────────────────────
( cd "$OUT" && sha256sum ./* > SHA256SUMS 2>/dev/null || true )

echo
echo "== 완료 =="
ls -1 "$OUT"
