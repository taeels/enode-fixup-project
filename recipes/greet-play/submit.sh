#!/bin/bash
# 문장 하나를 스피커로 재생한다.
#
#   submit.sh "<문장>"            board 노드의 기본 오디오 장치로 낸다
#   submit.sh --pi "<문장>"       그 노드에 물린 라즈베리파이 잭으로 낸다
#
# 만드는 자리와 트는 자리가 다르다. 이 스크립트는 계약을 채워 넣기만 하고,
# 어느 기계가 무엇을 맡을지는 매처가 고른다 (requires 의 속성이 정한다).
#
# --pi 가 바꾸는 것은 play 스텝 하나다. requires 는 같다 — 매칭은 노드를
# 고를 뿐이고, 소리가 어느 스피커로 나가는지는 스텝이 정한다.
set -euo pipefail

target=host
if [ "${1:-}" = "--pi" ]; then
  target=pi
  shift
fi

text=${1:-}
[ -n "$text" ] || { echo "submit.sh: 문장이 없다" >&2; exit 2; }

# 파이의 주소와 ALSA 장치. 환경변수로 덮어쓴다.
#
# 장치를 명시적으로 적는 이유 — 이 파이에는 HDMI 싱크가 붙어 있어서
# 기본 라우팅에 기대면 소리가 조용히 모니터로 간다.
PI_HOST=${PI_HOST:-sunny@192.168.137.50}
PI_DEV=${PI_DEV:-plughw:1,0}

here=$(cd "$(dirname "$0")" && pwd)
runid="$(date +%Y%m%d-%H%M%S)-$$"
tmp=$(mktemp -t greet-play).json

case "$target" in
  host) src="$here/greet-play.json";    prefix=greet-play    ;;
  pi)   src="$here/greet-play-pi.json"; prefix=greet-play-pi ;;
esac

# 문장은 argv 배열의 한 칸에 그대로 들어간다. 셸을 안 거치므로
# 공백이나 따옴표가 있어도 명령이 안 갈린다.
python3 - "$src" "$runid" "$text" "$PI_HOST" "$PI_DEV" > "$tmp" <<'PY'
import json, sys
src, runid, text, pihost, pidev = sys.argv[1:6]
raw = open(src, encoding="utf-8").read().replace("__RUNID__", runid)
raw = raw.replace("__PIHOST__", pihost).replace("__PIDEV__", pidev)
c = json.loads(raw)
for st in c["steps"]:
    st["run"] = [text if a == "__TEXT__" else a for a in st.get("run", [])]
json.dump(c, sys.stdout, ensure_ascii=False, indent=2)
PY

runctl lint "$tmp" >/dev/null
runctl submit "$tmp" --wait
echo "record: runctl record $prefix-$runid -o $prefix-$runid.tar"
