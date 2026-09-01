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
# gh 가 필요하다 — 릴리즈 자산을 받아 대조군으로 쓴다.
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

echo "== A · D  릴리즈 자산 =="
for t in "$BAD" "$GOOD"; do
  d="$TMP/rel-$t"; mkdir -p "$d"
  gh release download "$t" --repo taeels/enode \
    --pattern "enode-${t#v}-windows-amd64.zip" -D "$d" >/dev/null
  ( cd "$d" && unzip -oq ./*.zip )
  src="$(find "$d" -name enodectl.exe | head -1)"
  case "$t" in
    "$BAD")  cp "$src" "$OUT/A-${t#v}-release.exe" ;;
    "$GOOD") cp "$src" "$OUT/D-${t#v}-release.exe" ;;
  esac
done

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
