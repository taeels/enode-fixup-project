#!/bin/bash
# 문장 하나를 보드 노드의 스피커로 재생한다.
#
#   submit.sh "<문장>"
#
# 만드는 자리와 트는 자리가 다르다. 이 스크립트는 계약을 채워 넣기만 하고,
# 어느 기계가 무엇을 맡을지는 매처가 고른다 (requires 의 속성이 정한다).
set -euo pipefail

text=${1:-}
[ -n "$text" ] || { echo "submit.sh: 문장이 없다" >&2; exit 2; }

here=$(cd "$(dirname "$0")" && pwd)
runid="$(date +%Y%m%d-%H%M%S)-$$"
tmp=$(mktemp -t greet-play).json

# 문장은 argv 배열의 한 칸에 그대로 들어간다. 셸을 안 거치므로
# 공백이나 따옴표가 있어도 명령이 안 갈린다.
python3 - "$here/greet-play.json" "$runid" "$text" > "$tmp" <<'PY'
import json, sys
src, runid, text = sys.argv[1:4]
c = json.loads(open(src, encoding="utf-8").read().replace("__RUNID__", runid))
for st in c["steps"]:
    st["run"] = [text if a == "__TEXT__" else a for a in st.get("run", [])]
json.dump(c, sys.stdout, ensure_ascii=False, indent=2)
PY

runctl lint "$tmp" >/dev/null
runctl submit "$tmp" --wait
echo "record: runctl record greet-play-$runid -o greet-play-$runid.tar"
