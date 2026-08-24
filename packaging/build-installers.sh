#!/usr/bin/env bash
# 설치본을 굽는다 — ★ 리눅스에서 되는 것 전부 ★.
#
#   packaging/build-installers.sh                버전은 커밋에서 짓는다
#   PKG_VERSION=1.2.3 packaging/build-installers.sh
#   TARGETS="linux/amd64" packaging/build-installers.sh
#
# 결과는 dist/installers/ 에 쌓인다.
#
# ★ 맥 .pkg 는 여기서 안 만든다 ★ — pkgbuild·productbuild 가 맥에만 있고,
# 리눅스 대체품(xar + bomutils)은 데비안 저장소에 없어 소스 빌드가 필요하다.
# 그 취약함을 CI 에 들이는 것보다 맥 러너를 쓰는 편이 낫다.
# packaging/macos/pkg.sh 가 그 자리이고 macos 러너에서 돈다.
set -euo pipefail

cd "$(dirname "$0")/.."
ROOT=$(pwd)
OUT="$ROOT/dist/installers"
STAGE="$ROOT/dist/stage"

# ★ 버전은 두 갈래다 ★ — 태그에서 지으면 판 번호가 되고, 아니면 커밋이다.
# deb·rpm·msi 는 전부 판 번호를 요구하므로 실험 레인에도 무언가는 있어야 한다.
COMMIT=$(git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)
PKG_VERSION=${PKG_VERSION:-"0.0.0"}
# ★ v 접두사는 태그의 것이지 판 번호의 것이 아니다 ★ — 파일 이름에 남으면
# enode-v0.0.1-... 처럼 어색하고, deb·rpm 은 어차피 nfpm 이 떼어내서
# ★ 같은 릴리즈 안에서 이름이 두 갈래가 된다 ★. 여기서 한 번에 뗀다.
PKG_VERSION=${PKG_VERSION#v}
# ★ MSI 는 x.y.z 만 받는다 ★ — 접두사 v 나 뒤에 붙은 -rc1 을 그대로 주면 거절한다.
MSI_VERSION=$(printf '%s' "$PKG_VERSION" | sed 's/^v//' | grep -oE '^[0-9]+(\.[0-9]+){0,2}' || true)
MSI_VERSION=${MSI_VERSION:-0.0.0}

BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS="-X github.com/taeels/enode/internal/build.Commit=$COMMIT"
LDFLAGS="$LDFLAGS -X github.com/taeels/enode/internal/build.Date=$BUILD_DATE"
# ★ 태그에서 지은 것만 Release 를 채운다 ★ — 안 채우면 오늘 그대로 커밋이 신원이다.
if [ "${PKG_VERSION}" != "0.0.0" ]; then
  LDFLAGS="$LDFLAGS -X github.com/taeels/enode/internal/build.Release=${PKG_VERSION#v}"
fi

CMDS=${CMDS:-"enode runctl enodectl"}
TARGETS=${TARGETS:-"linux/amd64 linux/arm64 linux/arm windows/amd64 windows/arm64 darwin/amd64 darwin/arm64"}

rm -rf "$OUT" "$STAGE"
mkdir -p "$OUT"

echo "== 크로스 빌드 =="
for t in $TARGETS; do
  goos=${t%%/*}; goarch=${t##*/}
  d="$STAGE/$goos-$goarch"
  mkdir -p "$d"
  ext=""; [ "$goos" = "windows" ] && ext=".exe"
  for c in $CMDS; do
    echo "   $goos/$goarch  $c"
    # ★ GOARM 은 arm 일 때만 ★ — 다른 아키텍처에 주면 조용히 무시되지만,
    # 조용히 무시되는 것을 습관으로 두면 진짜 필요할 때 안 보인다 (Pi 2 = armv7).
    if [ "$goarch" = "arm" ]; then
      env GOOS=$goos GOARCH=$goarch GOARM=7 CGO_ENABLED=0 \
        go build -trimpath -ldflags "$LDFLAGS" -o "$d/$c$ext" "./cmd/$c"
    else
      env GOOS=$goos GOARCH=$goarch CGO_ENABLED=0 \
        go build -trimpath -ldflags "$LDFLAGS" -o "$d/$c$ext" "./cmd/$c"
    fi
  done
done

# ── 원본 묶음 ────────────────────────────────────────────────────────────
# ★ 설치본이 아닌 것도 낸다 ★ — 컨테이너·VM 안에서는 패키지 관리자를 거치는
# 것이 오히려 번거롭고, colima 안의 노드처럼 ★ 손으로 밀어 넣는 자리 ★ 가 있다.
echo "== 묶음 =="
for d in "$STAGE"/*; do
  n=$(basename "$d")
  case "$n" in
    windows-*) ( cd "$STAGE" && zip -qr "$OUT/enode-$PKG_VERSION-$n.zip" "$n" ) ;;
    *)         ( cd "$STAGE" && tar czf "$OUT/enode-$PKG_VERSION-$n.tar.gz" "$n" ) ;;
  esac
  echo "   $n"
done

# ── deb · rpm ────────────────────────────────────────────────────────────
if command -v nfpm >/dev/null 2>&1; then
  echo "== deb · rpm =="
  for a in amd64 arm64; do
    [ -d "$STAGE/linux-$a" ] || continue
    conf="$STAGE/nfpm-$a.yaml"
    sed -e "s|@ARCH@|$a|g" \
        -e "s|@VERSION@|$PKG_VERSION|g" \
        -e "s|@BIN@|$STAGE/linux-$a|g" \
        -e "s|@EXAMPLES@|$ROOT/packaging/macos/examples|g" \
        "$ROOT/packaging/linux/nfpm.yaml.in" > "$conf"
    nfpm package -f "$conf" -p deb -t "$OUT" >/dev/null
    nfpm package -f "$conf" -p rpm -t "$OUT" >/dev/null
    echo "   linux/$a"
  done
else
  echo "== deb · rpm 건너뜀 (nfpm 이 없다) =="
fi

# ── msi ──────────────────────────────────────────────────────────────────
if command -v wixl >/dev/null 2>&1; then
  echo "== msi =="
  # ★ amd64 뿐이다 ★ — wixl 0.101 이 arm64 를 거절한다
  # ("arch of type 'arm64' is not supported"). 윈도우 ARM 기계는 zip 으로 받는다.
  # ★ 되돌리는 조건 ★ — msitools 가 arm64 를 받거나, 윈도우 ARM 노드가 실물로
  # 필요해지면 그때 WiX 정식 툴체인(windows 러너)으로 옮긴다.
  for a in amd64; do
    [ -d "$STAGE/windows-$a" ] || continue
    # wixl 의 아키텍처 이름은 Go 와 다르다.
    case "$a" in amd64) wa=x64 ;; esac
    # ★ wixl 은 Source 에 절대경로를 안 받는다 ★ (실측: g_file_get_child
    # assertion '!g_path_is_absolute'). 그래서 스테이지로 들어가 상대경로로 준다.
    ( cd "$STAGE" && wixl -a "$wa" \
        -D Version="$MSI_VERSION" \
        -D BinDir="windows-$a" \
        -o "$OUT/enode-$PKG_VERSION-windows-$a.msi" \
        "$ROOT/packaging/windows/enode.wxs" )
    echo "   windows/$a"
  done
else
  echo "== msi 건너뜀 (wixl 이 없다 — apt install wixl) =="
fi

# ── 체크섬 ───────────────────────────────────────────────────────────────
( cd "$OUT" && sha256sum ./* > SHA256SUMS 2>/dev/null || true )

echo
echo "== 완료 =="
ls -1 "$OUT"
