#!/usr/bin/env bash
# 맥 설치본(.pkg)을 만든다. ★ 이것만 맥에서 돈다 ★.
#
#   PKG_VERSION=1.2.3 packaging/macos/pkg.sh dist/stage
#
# 인자는 크로스 빌드 결과가 있는 디렉터리다 — 그 아래에 darwin-amd64/ 와
# darwin-arm64/ 가 있어야 한다. 즉 ★ 컴파일은 리눅스가 하고 맥은 포장만 한다 ★
# (ADR-015 가 Go 를 고른 이유를 여기서도 쓴다).
#
# ★ 왜 리눅스에서 못 만드나 ★ — pkgbuild·productbuild 가 맥 전용이다.
# 리눅스 대체품(xar + bomutils)은 데비안 저장소에 없어 소스 빌드가 필요하고,
# 그 취약함을 릴리즈 경로에 들이는 것보다 맥 러너를 한 번 쓰는 편이 낫다.
#
# ★ 서명은 안 한다 ★ — 애플 개발자 계정이 필요하다. 서명 없는 pkg 는
# 더블클릭하면 Gatekeeper 가 막지만 ★ 터미널에서는 깔린다 ★:
#   sudo installer -pkg enode-*.pkg -target /
# README 에 그 한 줄을 적어 둔다. 계정이 생기면 productsign + notarytool 을
# 이 스크립트 끝에 붙이면 된다 — ★ 자리는 여기다 ★.
set -euo pipefail

STAGE=${1:?"크로스 빌드 결과 디렉터리를 인자로 달라 (예: dist/stage)"}
cd "$(dirname "$0")/../.."
ROOT=$(pwd)
STAGE=$(cd "$STAGE" && pwd)

PKG_VERSION=${PKG_VERSION:-0.0.0}
CMDS=${CMDS:-"enode runctl enodectl"}
OUT="$ROOT/dist/installers"
mkdir -p "$OUT"

ROOTDIR=$(mktemp -d)
trap 'rm -rf "$ROOTDIR"' EXIT
mkdir -p "$ROOTDIR/usr/local/bin"

echo "== 유니버설 바이너리 =="
# ★ 하나로 합친다 ★ — 인텔 맥과 애플 실리콘에 같은 pkg 를 준다. 두 벌을 내면
# 사람이 고르다 틀리고, 틀린 것은 "killed: 9" 로만 보인다.
for c in $CMDS; do
  a="$STAGE/darwin-amd64/$c"
  b="$STAGE/darwin-arm64/$c"
  if [ -f "$a" ] && [ -f "$b" ]; then
    lipo -create -output "$ROOTDIR/usr/local/bin/$c" "$a" "$b"
    echo "   $c  (amd64 + arm64)"
  elif [ -f "$b" ]; then
    cp "$b" "$ROOTDIR/usr/local/bin/$c"; echo "   $c  (arm64 만)"
  elif [ -f "$a" ]; then
    cp "$a" "$ROOTDIR/usr/local/bin/$c"; echo "   $c  (amd64 만)"
  else
    echo "   ▲ $c 가 $STAGE 에 없다"; exit 1
  fi
  chmod +x "$ROOTDIR/usr/local/bin/$c"
done

echo "== ★ 서명 확인 ★ =="
# 애플 실리콘 커널은 ★ 서명 없는 실행파일을 거부한다 ★ (killed: 9).
# lipo 가 합치면서 ad-hoc 서명이 살아 있는지 여기서 잰다 — build.sh 가 크로스
# 빌드 결과에 대해 하는 검사와 같은 이유다.
for c in $CMDS; do
  if codesign -dv "$ROOTDIR/usr/local/bin/$c" 2>&1 | grep -q "Signature=adhoc\|Signature size"; then
    echo "   $c  서명 있음"
  else
    echo "   ▲ $c 에 서명이 없다 — 애플 실리콘에서 killed: 9 가 된다"
    codesign -s - -f "$ROOTDIR/usr/local/bin/$c"
    echo "     ad-hoc 으로 붙였다"
  fi
done

echo "== pkgbuild =="
COMPONENT="$ROOTDIR/../enode-component.pkg"
pkgbuild \
  --root "$ROOTDIR" \
  --identifier com.taeels.enode \
  --version "${PKG_VERSION#v}" \
  --install-location / \
  "$COMPONENT" >/dev/null

echo "== productbuild =="
DIST=$(mktemp)
cat > "$DIST" <<XML
<?xml version="1.0" encoding="utf-8"?>
<installer-gui-script minSpecVersion="1">
  <title>enode</title>
  <options customize="never" require-scripts="false" hostArchitectures="x86_64,arm64"/>
  <pkg-ref id="com.taeels.enode"/>
  <choices-outline><line choice="default"/></choices-outline>
  <choice id="default" title="enode"><pkg-ref id="com.taeels.enode"/></choice>
  <pkg-ref id="com.taeels.enode" version="${PKG_VERSION#v}" onConclusion="none">enode-component.pkg</pkg-ref>
</installer-gui-script>
XML

productbuild \
  --distribution "$DIST" \
  --package-path "$(dirname "$COMPONENT")" \
  "$OUT/enode-$PKG_VERSION-macos.pkg" >/dev/null
rm -f "$DIST" "$COMPONENT"

echo
echo "== 완료 =="
ls -1 "$OUT"/*.pkg
