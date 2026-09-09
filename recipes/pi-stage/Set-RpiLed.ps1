#Requires -Version 7.0
<#
.SYNOPSIS
    Drive the two onboard LEDs of the Raspberry Pi over SSH.

.DESCRIPTION
    Controls ACT (green) and PWR (red) through the kernel LED class in sysfs.
    Runs on Windows, macOS and Linux under PowerShell 7. Needs no sudo on the
    board because the shipped udev rule opens the sysfs files to the gpio group.

.PARAMETER Action
    On, Off, Blink, Stop, Restore or Status.

.PARAMETER Hz
    Blink rate in full cycles per second. 2 means twice a second.

.PARAMETER Seconds
    How long to blink. Zero leaves a detached loop running on the board until
    Stop is given.

.EXAMPLE
    ./Set-RpiLed.ps1 -Action Blink -Hz 2

.EXAMPLE
    ./Set-RpiLed.ps1 -Action Blink -Hz 2 -Seconds 5 -Led ACT

.EXAMPLE
    ./Set-RpiLed.ps1 -Action Stop
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [ValidateSet('On', 'Off', 'Blink', 'Stop', 'Restore', 'Status')]
    [string]$Action,

    [ValidateSet('ACT', 'PWR', 'Both')]
    [string]$Led = 'Both',

    [ValidateRange(0.1, 20)]
    [double]$Hz = 2,

    [ValidateRange(0, 3600)]
    [int]$Seconds = 0,

    [string]$Target,
    [string]$KeyPath = (Join-Path $HOME '.ssh' 'id_rpi_sunnypi'),
    [switch]$Quiet
)

$ErrorActionPreference = 'Stop'

# 기본 트리거다. 수동 제어를 끝내면 여기로 되돌려야 SD카드 표시와 전원 표시가 살아난다.
$DefaultTrigger = @{ ACT = 'mmc0'; PWR = 'input' }
$Leds = if ($Led -eq 'Both') { @('ACT', 'PWR') } else { @($Led) }

function Say { param([string]$T, [string]$C = 'Gray') if (-not $Quiet) { Write-Host $T -ForegroundColor $C } }

# 원격 스크립트는 base64 로 실어 보낸다. 따옴표가 세 겹으로 겹치는 것을 피할 수 있고,
# 원격 명령줄에 스크립트 본문이 남지 않아서 pgrep 패턴이 자기 자신을 잡는 사고도 없다.
function Invoke-Remote {
    param([string]$Script)
    $lf  = $Script -replace "`r`n", "`n"
    $b64 = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($lf))
    $out = & ssh -T -n -i $KeyPath -o IdentitiesOnly=yes -o BatchMode=yes `
        -o StrictHostKeyChecking=accept-new -o ConnectTimeout=10 `
        $Target "echo $b64 | base64 -d | sh" 2>&1
    if ($LASTEXITCODE -ne 0) { throw "remote command failed: $((@($out) -join '; '))" }
    $out
}

function Join-Lines { param([string[]]$Lines) ($Lines -join "`n") }

$setNone  = Join-Lines ($Leds | ForEach-Object { "echo none > /sys/class/leds/$_/trigger" })
$setOn    = Join-Lines ($Leds | ForEach-Object { "echo 1 > /sys/class/leds/$_/brightness" })
$setOff   = Join-Lines ($Leds | ForEach-Object { "echo 0 > /sys/class/leds/$_/brightness" })
$restore  = Join-Lines ($Leds | ForEach-Object { "echo $($DefaultTrigger[$_]) > /sys/class/leds/$_/trigger" })
$halfWait = [Math]::Round(1.0 / (2 * $Hz), 3)

if (-not $Target) {
    $connect = Join-Path $PSScriptRoot 'Connect-Rpi.ps1'
    if (-not (Test-Path $connect)) { throw "Connect-Rpi.ps1 not found next to this script" }
    $link = & $connect -Quiet -PassThru -KeyPath $KeyPath
    if (-not $link.Connected) { throw "board not reachable: $($link.Reason)" }
    $Target  = $link.Target
    $KeyPath = $link.KeyPath
    Say "board: $($link.Hostname) via $($link.Method)" 'DarkGray'
}

switch ($Action) {

    'Status' {
        $script = @'
for d in ACT PWR; do
  t=$(sed -n 's/.*\[\(.*\)\].*/\1/p' /sys/class/leds/$d/trigger)
  printf '%-4s trigger=%-10s brightness=%s/%s\n' "$d" "$t" \
    "$(cat /sys/class/leds/$d/brightness)" "$(cat /sys/class/leds/$d/max_brightness)"
done
echo "blink loops running: $(pgrep -f "rpi-led-blink" 2>/dev/null | wc -l)"
'@
        Invoke-Remote $script | ForEach-Object { Say $_ }
    }

    'On'  { Invoke-Remote (Join-Lines @($setNone, $setOn))  | Out-Null; Say "$($Leds -join ' and ') on" 'Green' }

    'Off' { Invoke-Remote (Join-Lines @($setNone, $setOff)) | Out-Null; Say "$($Leds -join ' and ') off" 'Green' }

    'Restore' {
        Invoke-Remote $restore | Out-Null
        Say "restored: $(($Leds | ForEach-Object { "$_=$($DefaultTrigger[$_])" }) -join ' ')" 'Green'
    }

    'Stop' {
        $script = @'
n=0
for p in $(pgrep -f "rpi-led-blink" 2>/dev/null); do kill "$p" 2>/dev/null && n=$((n+1)); done
sleep 0.4
rm -f /tmp/rpi-led-*.sh
__RESTORE__
echo "stopped $n loop(s)"
'@ -replace '__RESTORE__', $restore
        Invoke-Remote $script | ForEach-Object { Say $_ 'Green' }
    }

    'Blink' {
        if ($Seconds -gt 0) {
            # 짧게 도는 동안은 붙잡고 있다가 끝나면 트리거를 되돌린다.
            $script = @'
__NONE__
end=$(( $(date +%s) + __SECONDS__ ))
while [ "$(date +%s)" -lt "$end" ]; do
__ON__
  sleep __HALF__
__OFF__
  sleep __HALF__
done
__RESTORE__
echo "blinked for __SECONDS__s at __HZ__ Hz"
'@
            $script = $script -replace '__NONE__', $setNone -replace '__ON__', $setOn `
                -replace '__OFF__', $setOff -replace '__RESTORE__', $restore `
                -replace '__HALF__', "$halfWait" -replace '__SECONDS__', "$Seconds" `
                -replace '__HZ__', "$Hz"
            Say "blinking $($Leds -join ' and ') at $Hz Hz for ${Seconds}s" 'Cyan'
            Invoke-Remote $script | ForEach-Object { Say $_ 'Green' }
        }
        else {
            # 무한 점멸은 보드에 떼어 놓고 ssh 는 빠진다. Stop 으로 세운다.
            $script = @'
for p in $(pgrep -f "rpi-led-blink" 2>/dev/null); do kill "$p" 2>/dev/null; done
sleep 0.3
rm -f /tmp/rpi-led-*.sh
cat > /tmp/rpi-led-blink.sh <<'INNER'
#!/bin/sh
__NONE__
while true; do
__ON__
  sleep __HALF__
__OFF__
  sleep __HALF__
done
INNER
chmod +x /tmp/rpi-led-blink.sh
setsid /tmp/rpi-led-blink.sh > /dev/null 2>&1 < /dev/null &
sleep 1
p=$(pgrep -f "rpi-led-blink" 2>/dev/null | head -1)
if [ -n "$p" ]; then echo "blink running pid=$p"; else echo "blink failed to start"; exit 1; fi
'@
            $script = $script -replace '__NONE__', $setNone -replace '__ON__', $setOn `
                -replace '__OFF__', $setOff -replace '__HALF__', "$halfWait"
            Say "blinking $($Leds -join ' and ') at $Hz Hz until Stop" 'Cyan'
            Invoke-Remote $script | ForEach-Object { Say $_ 'Green' }
        }
    }
}
