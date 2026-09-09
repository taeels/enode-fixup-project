echo "== which =="; which mpg123; readlink -f /usr/bin/mpg123
echo "== mpg123 --version =="
mpg123 --version 2>&1
echo "version_rc=$?"
echo "== card index for Headphones =="
cat /proc/asound/cards
echo "== amixer controls card 1 =="
amixer -c 1 scontrols 2>&1
echo "== clip info =="
ls -l /tmp/mpg123debs/welcome.mp3
ffprobe -v error -show_entries format=duration,bit_rate,format_name -show_entries stream=codec_name,sample_rate,channels -of default=noprint_wrappers=1 /tmp/mpg123debs/welcome.mp3 2>&1
