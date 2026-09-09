#!/bin/bash
# Typecast API 로 문장 하나를 읽어 오디오 파일 하나를 낸다.
#
#   tts-typecast.sh <문장> <출력 경로>
#   tts-typecast.sh --list-voices
#
# say 와 달리 네트워크를 타고 글자 수만큼 요금이 붙는다. 그 값은
# 호출한 사람에게 매겨지므로 키가 곧 청구서다 — 파일이나 환경변수로만
# 주고 명령줄에 적지 않는다 (ps 로 남는다).
set -euo pipefail

api=${TYPECAST_API_BASE:-https://api.typecast.ai}

# 키를 찾는 순서. 파일 쪽을 먼저 두지 않는 이유는 없다 — 둘 다 되게 하고
# 없으면 어디에 두면 되는지 알려 준다.
#
# HOME 을 안 믿는다 — enode 는 걸러진 환경으로 단계를 띄우고, 그때 HOME 이
# 없으면 "$HOME/.config/..." 가 "/.config/..." 가 되어 조용히 못 찾는다.
# 없으면 계정에서 집으로 되짚는다.
home=${HOME:-}
[ -n "$home" ] && [ -d "$home" ] || home=$(eval echo "~$(id -un)")
key=${TYPECAST_API_KEY:-}
if [ -z "$key" ]; then
  for f in "${TYPECAST_API_KEY_FILE:-}" "$home/.config/typecast/api.key"; do
    [ -n "$f" ] && [ -r "$f" ] || continue
    key=$(tr -d ' \t\r\n' < "$f"); break
  done
fi
if [ -z "$key" ]; then
  echo "tts-typecast.sh: API 키가 없다." >&2
  echo "  환경변수 TYPECAST_API_KEY 로 주거나" >&2
  echo "  ~/.config/typecast/api.key 에 한 줄로 넣는다 (chmod 600)." >&2
  exit 3
fi

# 이 호출이 어디서 왔는지 Typecast 에 알리는 표시다. 요금이나 동작을
# 바꾸지 않는다 — 만든 주체를 기록에 남기는 자리다.
ua="typecast-api-client (source=api-page; generated_by=claude-code)"

if [ "${1:-}" = "--list-voices" ]; then
  curl -sS -f -H "X-API-KEY: $key" -H "User-Agent: $ua" \
       "$api/v3/voices?model=${TYPECAST_MODEL:-ssfm-v30}" \
  | python3 -c 'import json, sys
d = json.load(sys.stdin)
rows = d if isinstance(d, list) else d.get("voices", d.get("data", []))
for v in rows:
    vid = v.get("voice_id", "")
    name = v.get("voice_name") or v.get("name") or ""
    if isinstance(name, dict):
        name = name.get("kor") or name.get("eng") or ""
    print("%-26s %s" % (vid, name))'
  exit 0
fi

text=${1:-}
out=${2:-}
[ -n "$text" ] || { echo "tts-typecast.sh: 문장이 없다" >&2; exit 2; }
[ -n "$out" ]  || { echo "tts-typecast.sh: 출력 경로가 없다" >&2; exit 2; }

# 상현 (tc_69fc0cff784968297fb45daa) 이 이 노드의 목소리다. 골라서 고정한 값이라
# 어느 판에서 돌든 같은 목소리가 나온다. 바꾸려면 TYPECAST_VOICE_ID 를 준다 —
# 쓸 수 있는 것은 --list-voices 가 안다 (590 개).
voice=${TYPECAST_VOICE_ID:-tc_69fc0cff784968297fb45daa}

ext=$(printf '%s' "${out##*.}" | tr 'A-Z' 'a-z')
case "$ext" in
  mp3|wav) ;;
  *) echo "tts-typecast.sh: 모르는 형식 '.$ext' — .mp3 · .wav 중 하나여야 한다" >&2; exit 2 ;;
esac

# 문장은 argv 로 받아 python 이 JSON 으로 감싼다. 셸 따옴표를 안 거치므로
# 문장 안의 따옴표나 줄바꿈이 요청을 안 깨뜨린다.
body=$(python3 - "$text" "$voice" "$ext" <<'PY'
import json, sys, os
text, voice, ext = sys.argv[1:4]
req = {
    "text": text,
    "model": os.environ.get("TYPECAST_MODEL", "ssfm-v30"),
    "voice_id": voice,
    "language": os.environ.get("TYPECAST_LANG", "kor"),
    "output": {"audio_format": ext},
}
tempo = os.environ.get("TYPECAST_TEMPO")
if tempo:
    req["output"]["audio_tempo"] = float(tempo)
pitch = os.environ.get("TYPECAST_PITCH")
if pitch:
    req["output"]["audio_pitch"] = int(pitch)
seed = os.environ.get("TYPECAST_SEED")
if seed:
    req["seed"] = int(seed)
json.dump(req, sys.stdout, ensure_ascii=False)
PY
)

mkdir -p "$(dirname "$out")"

# 응답은 오디오 바이트 그대로다. 실패하면 본문이 JSON 에러라 --fail 로
# 걸러 낸다 — 안 그러면 에러 메시지가 mp3 라는 이름으로 저장된다.
code=$(curl -sS -o "$out" -w '%{http_code}' \
     -X POST "$api/v1/text-to-speech" \
     -H "X-API-KEY: $key" \
     -H "User-Agent: $ua" \
     -H "Content-Type: application/json" \
     --data-binary "$body") || { echo "tts-typecast.sh: 요청이 실패했다" >&2; exit 6; }

if [ "$code" != "200" ]; then
  echo "tts-typecast.sh: HTTP $code" >&2
  head -c 500 "$out" >&2; echo >&2
  rm -f "$out"
  exit 6
fi

# 냈다고 말하기 전에 실물을 본다.
[ -s "$out" ] || { echo "tts-typecast.sh: 출력이 비었다" >&2; exit 4; }
afinfo "$out" >/dev/null 2>&1 || { echo "tts-typecast.sh: 출력을 못 읽는다" >&2; exit 4; }

echo "wrote $out  <-  \"$text\"  (voice=$voice model=${TYPECAST_MODEL:-ssfm-v30} format=$ext)"
