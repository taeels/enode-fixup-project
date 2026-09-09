#!/bin/bash
# 파이에 물린 LED 둘을 번갈아 켠다.
#
#   submit.sh [빨강GPIO] [초록GPIO] [반복] [간격ms]
#   submit.sh 17 27 6 250
#
# 기본값은 확인 안 된 값이다 (README 참고). 처음 돌릴 때는 눈으로 보고,
# 다르면 인자로 바로잡은 뒤 그 값을 CLAUDE.md 와 README 에 적는다.
set -euo pipefail

RED=${1:-17}
GREEN=${2:-27}
CYCLES=${3:-6}
MS=${4:-250}

PI_HOST=${PI_HOST:-sunny@192.168.137.50}

for v in "$RED" "$GREEN" "$CYCLES" "$MS"; do
  case "$v" in
    ''|*[!0-9]*) echo "submit.sh: 숫자가 아니다: $v" >&2; exit 2 ;;
  esac
done

here=$(cd "$(dirname "$0")" && pwd)
runid="$(date +%Y%m%d-%H%M%S)-$$"
tmp=$(mktemp -t led-toggle).json

python3 - "$here/led-toggle.json" "$runid" "$PI_HOST" "$RED" "$GREEN" "$CYCLES" "$MS" > "$tmp" <<'PY'
import json, sys
src, runid, pihost, red, green, cycles, ms = sys.argv[1:8]
raw = open(src, encoding="utf-8").read()
for k, v in (("__RUNID__", runid), ("__PIHOST__", pihost), ("__RED__", red),
             ("__GREEN__", green), ("__CYCLES__", cycles), ("__MS__", ms)):
    raw = raw.replace(k, v)
json.dump(json.loads(raw), sys.stdout, ensure_ascii=False, indent=2)
PY

runctl lint "$tmp" >/dev/null
runctl submit "$tmp" --wait
echo "record: runctl record led-toggle-$runid -o led-toggle-$runid.tar"
