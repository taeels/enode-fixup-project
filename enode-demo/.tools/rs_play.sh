set -u
F=/tmp/mpg123debs/welcome.mp3
ST=/proc/asound/card1/pcm0p/sub0/status
echo "== mixer before =="
amixer -c 1 sget PCM 2>&1 | tail -4
echo "== set volume max + unmute =="
amixer -c 1 sset PCM 100% unmute 2>&1 | tail -4
echo "== status file before playback =="
cat "$ST" 2>&1
echo "== sampler start =="
: > /tmp/mpg123debs/hwptr.log
(
  for i in $(seq 1 60); do
    t=$(date +%s.%N)
    s=$(cat "$ST" 2>/dev/null | tr '\n' ' ')
    echo "t=$t $s" >> /tmp/mpg123debs/hwptr.log
    sleep 0.15
  done
) &
SPID=$!
sleep 0.3
echo "== PLAYBACK =="
echo "CMD: mpg123 -o alsa -a plughw:1,0 $F"
T0=$(date +%s.%N)
mpg123 -o alsa -a plughw:1,0 "$F" 2>&1
RC=$?
T1=$(date +%s.%N)
echo "play_rc=$RC"
echo "elapsed=$(awk -v a=$T0 -v b=$T1 'BEGIN{printf "%.3f", b-a}')"
sleep 0.5
kill $SPID 2>/dev/null; wait $SPID 2>/dev/null
echo "== status file after playback =="
cat "$ST" 2>&1
echo "== sampler log (non-closed lines) =="
grep -v 'closed' /tmp/mpg123debs/hwptr.log | sed -n '1,80p'
echo "== sampler log summary: first/last closed markers =="
grep -c 'closed' /tmp/mpg123debs/hwptr.log
