#!/usr/bin/env bash
# 실물 네 벌을 나란히 굽는다. 탐침 넷이 모두 통과한 다음 단계다.
#
#   bash scripts/avprobe/bisect.sh
#
# 왜 인공 탐침이 아니라 실물인가 — avprobe 의 both 는 rc13 의 enodectl 과
# 같은 API 를 링크했는데도 지워지지 않았다. 그러므로 방아쇠는 임포트하는
# API 집합이 아니고, 남은 후보는 실제 코드량과 문자열과 구조다. 그것들은
# 흉내 낼 수 없으므로 실물을 반으로 가른다.
#
# 릴리즈 자산 둘이 대조군이다. 저장소가 비공개라 익명 다운로드가 안 되므로
# github.com 에 로그인된 gh 로 받는다. 사내 GHES 에만 붙은 기계처럼 그것이
# 안 되는 자리에서는 손으로 받은 zip 을 건네면 gh 를 건너뛴다.
#
#   ZIP_BAD=~/Downloads/enode-0.0.1-rc13-windows-amd64.zip \
#   ZIP_GOOD=~/Downloads/enode-0.0.1-rc9-windows-amd64.zip \
#     bash scripts/avprobe/bisect.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUT="${OUT:-$ROOT/scripts/avprobe/dist-bisect}"
TMP="$(mktemp -d)"
trap 'git -C "$ROOT" worktree remove --force "$TMP/wt13" 2>/dev/null || true; rm -rf "$TMP"' EXIT

BAD="${BAD:-v0.0.1-rc13}"   # 지워지는 판
GOOD="${GOOD:-v0.0.1-rc9}"  # 멀쩡한 판
rm -rf "$OUT"; mkdir -p "$OUT"

build() { # <워크트리> <출력이름>
  ( cd "$1" && env GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath \
      -ldflags "-X github.com/taeels/enode/internal/build.Release=${BAD#v}" \
      -o "$OUT/$2" ./cmd/enodectl )
}

# 미리 받아 둔 zip 이 있으면 그것을 쓰고, 없으면 gh 로 받는다.
fetch() { # <태그> <출력이름> <미리받은zip 또는 빈값>
  local tag="$1" out="$2" given="${3:-}" d="$TMP/rel-$1"
  mkdir -p "$d"
  if [ -n "$given" ]; then
    [ -f "$given" ] || { echo "없는 파일이다: $given" >&2; exit 1; }
    cp "$given" "$d/asset.zip"
  else
    if ! gh auth status --hostname github.com >/dev/null 2>&1; then
      echo "gh 가 github.com 에 로그인돼 있지 않다." >&2
      echo "릴리즈 페이지에서 zip 둘을 받아 ZIP_BAD 와 ZIP_GOOD 로 건넨다:" >&2
      echo "  https://github.com/taeels/enode/releases" >&2
      exit 1
    fi
    gh release download "$tag" --repo taeels/enode \
      --pattern "enode-${tag#v}-windows-amd64.zip" -O "$d/asset.zip"
  fi
  ( cd "$d" && unzip -oq asset.zip )
  local src; src="$(find "$d" -name enodectl.exe | head -1)"
  [ -n "$src" ] || { echo "묶음 안에 enodectl.exe 가 없다: $tag" >&2; exit 1; }
  cp "$src" "$OUT/$out"
}

echo "== A · D  릴리즈 자산 =="
fetch "$BAD"  "A-${BAD#v}-release.exe"  "${ZIP_BAD:-}"
fetch "$GOOD" "D-${GOOD#v}-release.exe" "${ZIP_GOOD:-}"

echo "== B  같은 소스를 여기서 굽는다 =="
git -C "$ROOT" worktree add -q --detach "$TMP/wt13" "$BAD"
build "$TMP/wt13" "B-${BAD#v}-source.exe"

echo "== C  거기서 setup 만 뺀다 =="
cd "$TMP/wt13"
rm -f cmd/enodectl/setup.go
sed -i '/case "setup":/,+1d' cmd/enodectl/main.go
sed -i '/enodectl setup <name>/d' cmd/enodectl/main.go
sed -i 's/(setup · list · id/(list · id/' cmd/enodectl/main.go
gofmt -w cmd/enodectl/main.go
build "$TMP/wt13" "C-${BAD#v}-nosetup.exe"

cd "$OUT"
sha256sum ./* > SHA256SUMS
for f in *.exe; do
  printf "  %-26s %10d bytes  crypto/tls=%s\n" "$f" "$(stat -c%s "$f")" \
    "$(strings -a "$f" | grep -c crypto/tls || true)"
done
