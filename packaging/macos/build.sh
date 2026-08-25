#!/usr/bin/env bash
# 맥용 배포 묶음을 만든다. 리눅스에서 크로스 빌드한다 (ADR-015 가 Go 를 고른 이유).
#
#   packaging/macos/build.sh            arm64 + amd64 둘 다
#   TARGETS="darwin/arm64" ...build.sh  하나만
#
# 결과는 dist/enode-macos-<commit>/ 와 그 tar.gz 다.
set -euo pipefail

cd "$(dirname "$0")/../.."
ROOT=$(pwd)

# 대상 목록을 변수로 둔다 — 오늘은 맥 둘이지만, README 의 CI 가 이미
# windows/amd64 · linux/arm 을 같은 이유로 굽고 있다. 목록이 늘 자리다.
#
# linux 가 왜 「맥용」 묶음에 들어가나
# colima 는 맥 위에 리눅스 VM을 띄운다. 그 안의 enode 는 darwin 이 아니라
# linux 다. 2차 시나리오("colima 에도 enode 를 설치해라")에서 맥의 에이전트가
# VM 안에 넣을 실행파일이 손이 닿는 곳에 이미 있어야 한다 — 없으면 그
# 에이전트가 툴체인부터 세우게 되고, 그건 이 시나리오가 재는 것이 아니다.
TARGETS=${TARGETS:-"darwin/arm64 darwin/amd64 linux/arm64 linux/amd64"}

# 세 실행파일을 함께 넣는다 — 맥은 노드이면서 동시에 사람이 앉는 자리다.
# runctl 이 없으면 맥에서 `runctl capabilities` 로 자기가 광고됐는지 볼 수 없고,
# enodectl 이 없으면 노드를 띄우고 멈출 수 없다.
#
# enodectl 이 셸이 아니라 Go 인 이유 — 셸판은 node_id 를 다시 계산 했다.
# sha256 파이프와 python3 realpath 로 Derive() 를 흉내 냈고, 그래서 python3 가
# 없는 기계에서 못 돌았다. 지금은 enode 가 쓰는 그 함수를 그대로 부른다.
CMDS=${CMDS:-"enode runctl enodectl"}

COMMIT=$(git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)
# 묶음에 들어가는 것이 커밋과 다른지 본다 — dist/ 나 문서가 지저분한 것은
# 무관하다. 무관한 것으로 -dirty 를 붙이면 그 표시가 곧 무시된다.
#
# packaging/ 도 넣는다 — 실행파일뿐 아니라 install.sh·enodectl·예시가
# 묶음에 들어간다. 그것만 고치고 다시 구우면 이름이 같은데 내용이 다른
# 묶음 이 나오고, 맥에서 어느 것이 새것인지 구별할 수 없다. 실제로 밟았다.
DIRTY=""
if [ -n "$(git status --porcelain -- '*.go' go.mod go.sum packaging 2>/dev/null)" ]; then
  DIRTY="-dirty"
fi
STAMP="${COMMIT}${DIRTY}"

# 실행파일이 자기가 무엇인지 말할 수 있게 한다 (ADR-056)
#
# 자기 갱신이 여기 걸린다 — 새 바이너리를 받아 두었을 때 그것이 새것인지
# 확인할 방법이 --version 말고는 없다. 실행해 보는 것은 노드를 하나 더 띄우는 일이다.
# 묶음 이름과 같은 값을 박는다 — 둘이 어긋나면 어느 쪽이 참인지 알 수 없다.
BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS="-X github.com/taeels/enode/internal/build.Commit=$STAMP"
LDFLAGS="$LDFLAGS -X github.com/taeels/enode/internal/build.Date=$BUILD_DATE"

PKG="$ROOT/dist/enode-macos-$STAMP"
rm -rf "$PKG"
mkdir -p "$PKG/bin" "$PKG/examples"

echo "== 크로스 빌드 =="
for t in $TARGETS; do
  goos=${t%%/*}; goarch=${t##*/}
  for c in $CMDS; do
    out="$PKG/bin/$c-$goos-$goarch"
    echo "   $goos/$goarch  $c"
    GOOS=$goos GOARCH=$goarch CGO_ENABLED=0 \
      go build -trimpath -ldflags "$LDFLAGS" -o "$out" "./cmd/$c"
  done
done

echo "== 서명 확인 =="
# 애플 실리콘 커널은 서명 없는 실행파일을 거부한다 (killed: 9).
# Go 내부 링커가 크로스 빌드에서도 ad-hoc 서명을 붙이는지 여기서 잰다 —
# 안 붙는 툴체인으로 바뀌면 맥에서 "왜 안 되는지 모르는" 실패가 된다.
for f in "$PKG"/bin/*-darwin-arm64; do
  python3 "$ROOT/packaging/macos/check-signature.py" "$f"
done

cp packaging/macos/install.sh        "$PKG/install.sh"
# 자기 갱신 스크립트도 넣는다 (ADR-056) — 다음 갱신부터는 묶음 안의 것을 쓴다.
# 첫 갱신에는 못 쓴다 — 맥에 있는 것은 이 파일이 없던 묶음이다.
cp packaging/self-update.sh         "$PKG/self-update.sh"
cp packaging/macos/README.md         "$PKG/README.md"
cp packaging/macos/examples/*.yaml   "$PKG/examples/"
chmod +x "$PKG/install.sh" "$PKG/self-update.sh"

cat > "$PKG/MANIFEST" <<MANIFEST
enode macOS 배포 묶음
commit   $STAMP
built_on $(uname -srm)
go       $(go version | awk '{print $3}')
targets  $TARGETS
cmds     $CMDS
MANIFEST

( cd "$ROOT/dist" && tar czf "enode-macos-$STAMP.tar.gz" "enode-macos-$STAMP" )

echo
echo "== 완료 =="
echo "   $PKG"
echo "   $ROOT/dist/enode-macos-$STAMP.tar.gz"
