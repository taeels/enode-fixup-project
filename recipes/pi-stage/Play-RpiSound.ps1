#Requires -Version 7.0
<#
.SYNOPSIS
    Send an audio file to the Raspberry Pi and play it on the headphone jack.

.DESCRIPTION
    The file lives on this machine and the speaker hangs off the board, so the
    clip is copied first and played second. Runs on Windows, macOS and Linux
    under PowerShell 7.

    An exit code of zero does not mean anybody heard anything. The player can
    open the device and throw the samples away. So the run is judged on three
    things together: the exit code, the elapsed time against the clip length,
    and the kernel counter that says the card actually consumed frames.

.PARAMETER Path
    Local audio file. Extension picks the player - mp3 goes to mpg123, wav
    goes to aplay. Those two flags are named differently and aplay cannot read
    mp3 at all.

.PARAMETER Card
    ALSA card index. 1 is the 3.5mm jack, 0 is HDMI.

.PARAMETER KeepRemote
    Leave the uploaded file on the board instead of removing it.

.EXAMPLE
    ./Play-RpiSound.ps1 -Path .\welcome.mp3

.EXAMPLE
    ./Play-RpiSound.ps1 -Path .\chime.wav -Card 1 -Quiet
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [string]$Path,

    [ValidateRange(0, 7)]
    [int]$Card = 1,

    [string]$Target,
    [string]$KeyPath = (Join-Path $HOME '.ssh' 'id_rpi_sunnypi'),
    [switch]$KeepRemote,
    [switch]$NoVerify,
    [switch]$Quiet
)

$ErrorActionPreference = 'Stop'

function Say  { param([string]$T, [string]$C = 'Gray') if (-not $Quiet) { Write-Host $T -ForegroundColor $C } }
function Ok   { param([string]$T) Say "  OK    $T" 'Green' }
function Bad  { param([string]$T) Say "  FAIL  $T" 'Red' }

if (-not (Test-Path -LiteralPath $Path)) { throw "audio file not found: $Path" }
$local = (Resolve-Path -LiteralPath $Path).Path
$ext   = [IO.Path]::GetExtension($local).ToLower()

# 장치를 가리키는 플래그 이름이 재생기마다 다르다. aplay 는 mp3 를 못 읽는다.
$player = switch ($ext) {
    '.mp3' { "mpg123 -q -o alsa -a plughw:$Card,0" }
    '.wav' { "aplay -D plughw:$Card,0" }
    default { throw "unsupported extension '$ext'. Use .mp3 or .wav." }
}

if (-not $Target) {
    $connect = Join-Path $PSScriptRoot 'Connect-Rpi.ps1'
    if (-not (Test-Path $connect)) { throw 'Connect-Rpi.ps1 not found next to this script' }
    $link = & $connect -Quiet -PassThru -KeyPath $KeyPath
    if (-not $link.Connected) { throw "board not reachable: $($link.Reason)" }
    $Target  = $link.Target
    $KeyPath = $link.KeyPath
    Say "board: $($link.Hostname) via $($link.Method)" 'DarkGray'
}

# 지난 시도가 남긴 파일이 덮어쓰기를 막은 적이 있다. 이름을 매번 새로 짓는다.
$stamp  = Get-Date -Format 'yyyyMMdd-HHmmss'
$remote = "/tmp/pi-stage-$stamp$ext"

# IPv6 링크로컬로 붙은 경우 scp 는 호스트를 대괄호로 감싸야 한다.
$scpTarget = $Target
if ($Target -match '^([^@]+)@(.+:.+)$') { $scpTarget = "$($Matches[1])@[$($Matches[2])]" }

Say "upload  $([IO.Path]::GetFileName($local)) -> $remote" 'Cyan'
& scp -q -i $KeyPath -o IdentitiesOnly=yes -o BatchMode=yes `
    -o StrictHostKeyChecking=accept-new $local "${scpTarget}:$remote"
if ($LASTEXITCODE -ne 0) { throw 'scp failed' }
Ok 'uploaded'

$verify = if ($NoVerify) { '0' } else { '1' }
$keep   = if ($KeepRemote) { '1' } else { '0' }

$script = @'
set -u
F=__REMOTE__
CARD=__CARD__
ST=/proc/asound/card__CARD__/pcm0p/sub0/status

DUR=$(ffprobe -v error -show_entries format=duration -of default=noprint_wrappers=1:nokey=1 "$F" 2>/dev/null)
[ -n "$DUR" ] && echo "clip_seconds=$DUR"

# 한 번 안 들린 적이 있고 원인이 볼륨이었다. 멱등이므로 매번 넣는다.
amixer -c "$CARD" sset PCM 100% unmute > /dev/null 2>&1
echo "mixer=$(amixer -c "$CARD" sget PCM 2>/dev/null | tail -1 | tr -s ' ')"

if [ "__VERIFY__" = "1" ]; then
  LOG=$(mktemp)
  ( for i in $(seq 1 200); do
      printf '%s\n' "$(sed -n 's/^hw_ptr *: *//p' "$ST" 2>/dev/null) $(sed -n 's/^state: *//p' "$ST" 2>/dev/null)" >> "$LOG"
      sleep 0.1
    done ) &
  SPID=$!
  sleep 0.2
fi

T0=$(date +%s.%N)
__PLAYER__ "$F" 2>&1 | tail -2
RC=${PIPESTATUS[0]}
T1=$(date +%s.%N)
echo "exit_code=$RC"
echo "elapsed=$(awk -v a=$T0 -v b=$T1 'BEGIN{printf "%.3f", b-a}')"

if [ "__VERIFY__" = "1" ]; then
  sleep 0.4
  kill $SPID 2>/dev/null; wait $SPID 2>/dev/null
  # hw_ptr 이 단조 증가했으면 카드가 실제로 프레임을 먹은 것이다.
  FIRST=$(awk 'NF>1 && $1 ~ /^[0-9]+$/ {print $1; exit}' "$LOG")
  LAST=$(awk 'NF>1 && $1 ~ /^[0-9]+$/ {v=$1} END{print v}' "$LOG")
  STATES=$(awk '{print $NF}' "$LOG" | grep -E '^[A-Z]+$' | uniq | tr '\n' ' ')
  echo "hw_ptr_first=${FIRST:-none}"
  echo "hw_ptr_last=${LAST:-none}"
  echo "states=${STATES:-none}"
  rm -f "$LOG"
fi

[ "__KEEP__" = "1" ] || rm -f "$F"
'@

$script = $script -replace '__REMOTE__', $remote `
                  -replace '__CARD__',   "$Card" `
                  -replace '__PLAYER__', $player `
                  -replace '__VERIFY__', $verify `
                  -replace '__KEEP__',   $keep

$lf  = $script -replace "`r`n", "`n"
$b64 = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($lf))

Say "play    $player" 'Cyan'
$out = & ssh -T -n -i $KeyPath -o IdentitiesOnly=yes -o BatchMode=yes `
    -o StrictHostKeyChecking=accept-new -o ConnectTimeout=10 `
    $Target "echo $b64 | base64 -d | bash" 2>&1

$fields = @{}
foreach ($line in $out) {
    if ($line -match '^([a-z_]+)=(.*)$') { $fields[$Matches[1]] = $Matches[2] }
    else { Say "  $line" 'DarkGray' }
}

$rc      = [int]($fields['exit_code'] ?? 1)
$elapsed = [double]($fields['elapsed'] ?? 0)
$clip    = if ($fields['clip_seconds']) { [double]$fields['clip_seconds'] } else { $null }

Say ''
if ($rc -eq 0) { Ok "player exited 0" } else { Bad "player exited $rc" }

if ($clip) {
    # 즉시 끝났으면 장치를 열고 표본을 버린 것이다.
    $drift = [Math]::Abs($elapsed - $clip)
    if ($drift -le [Math]::Max(0.7, $clip * 0.25)) {
        Ok ("elapsed {0:N3}s matches clip {1:N3}s" -f $elapsed, $clip)
    } else {
        Bad ("elapsed {0:N3}s does not match clip {1:N3}s" -f $elapsed, $clip)
    }
} else {
    Say ("  ..    elapsed {0:N3}s, clip length unknown" -f $elapsed) 'DarkGray'
}

if (-not $NoVerify) {
    $first = $fields['hw_ptr_first']; $last = $fields['hw_ptr_last']
    $states = $fields['states']
    if ($first -and $last -and $first -ne 'none' -and [long]$last -gt [long]$first) {
        Ok "kernel counter advanced $first -> $last on card $Card"
    } else {
        Bad "kernel counter did not advance - the card never consumed frames"
    }
    if ($states -match 'DRAINING') { Ok "reached DRAINING, the clip ran to the end" }
    Say "  states seen: $states" 'DarkGray'
}

Say ''
Say 'Note: this only proves the DAC consumed the samples. Nothing here can' 'DarkGray'
Say 'hear the speaker - do not record that it was audible without a listener.' 'DarkGray'
