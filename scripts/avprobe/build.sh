#!/usr/bin/env bash
# 탐침 넷을 굽는다. 기본 대상은 windows/amd64 다.
#
#   bash scripts/avprobe/build.sh
#
# -trimpath 를 주는 이유는 릴리즈와 같다 — 같은 소스가 같은 바이너리를 내야
# 「받은 파일이 이 소스에서 나왔다」를 해시로 말할 수 있다.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT="${OUT:-$ROOT/dist}"
GOOS_="${GOOS:-windows}"
GOARCH_="${GOARCH:-amd64}"

ext=""; [ "$GOOS_" = "windows" ] && ext=".exe"
rm -rf "$OUT"; mkdir -p "$OUT"

for p in neither netonly proconly both; do
  ( cd "$ROOT/$p" && env GOOS="$GOOS_" GOARCH="$GOARCH_" CGO_ENABLED=0 \
      go build -trimpath -o "$OUT/$p$ext" . )
done

cd "$OUT"
sha256sum ./* > SHA256SUMS
ls -l
echo
cat SHA256SUMS
