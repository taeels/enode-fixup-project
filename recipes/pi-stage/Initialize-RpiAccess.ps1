#Requires -Version 7.0
<#
.SYNOPSIS
    Prepare one machine to drive the board without a password.

.DESCRIPTION
    Run this once per machine, from a real terminal. It asks for the board
    account password twice - once to install the public key, once for sudo
    while placing the udev rule. After that Connect-Rpi.ps1 and Set-RpiLed.ps1
    need no password at all.

    Existing authorized_keys entries are kept. The key is appended, never
    written over. The same holds for the local ssh config.

.PARAMETER Alias
    Short name written into the local ssh config, so that plain "ssh sunnypi"
    reaches the board with no key flags.

.PARAMETER SkipSshConfig
    Leave the local ssh config alone. Callers then have to pass -i on every
    invocation, because the key does not carry a default name.

.EXAMPLE
    ./Initialize-RpiAccess.ps1
#>
[CmdletBinding()]
param(
    [string]$HostName = 'sunnypi.local',
    [string]$IPv4     = '192.168.137.50',
    [string]$User     = 'sunny',
    [string]$Alias    = 'sunnypi',
    [string]$KeyPath  = (Join-Path $HOME '.ssh' 'id_rpi_sunnypi'),
    [switch]$SkipSshConfig
)

$ErrorActionPreference = 'Stop'
$target = "$User@$HostName"

function Step { param([string]$T) Write-Host "`n$T" -ForegroundColor Cyan }
function Ok   { param([string]$T) Write-Host "  OK    $T" -ForegroundColor Green }
function Bad  { param([string]$T) Write-Host "  FAIL  $T" -ForegroundColor Red }

foreach ($tool in 'ssh', 'ssh-keygen') {
    if (-not (Get-Command $tool -ErrorAction SilentlyContinue)) {
        throw "$tool not found in PATH. Install the OpenSSH client first."
    }
}

Step '1. Local key pair'
if (Test-Path $KeyPath) {
    Ok "already present: $KeyPath"
} else {
    $dir = Split-Path -Parent $KeyPath
    if (-not (Test-Path $dir)) { New-Item -ItemType Directory -Path $dir -Force | Out-Null }
    # 자동 제어에 쓰므로 패스프레이즈를 걸지 않는다. 이 계정이 곧 보드 접근 권한이다.
    & ssh-keygen -t ed25519 -f $KeyPath -N '""' -C 'rpi-led' -q
    if ($LASTEXITCODE -ne 0) { throw 'ssh-keygen failed' }
    Ok "created: $KeyPath"
}
$pub = (Get-Content "$KeyPath.pub" -Raw).Trim()

Step '2. Local ssh config entry'
# 키 이름이 기본값이 아니라서, config 에 안 적으면 ssh 가 후보로도 안 올린다.
# 그 상태에서 ssh 를 그냥 부르면 publickey 로 거절당한다 - 네트워크 문제처럼 보이지만
# 클라이언트가 내밀 키가 없었던 것이다.
$cfgPath = Join-Path $HOME '.ssh' 'config'
if ($SkipSshConfig) {
    Ok 'skipped by request'
}
elseif ((Test-Path $cfgPath) -and
        ((Get-Content $cfgPath -Raw) -match "(?m)^\s*Host\s+.*\b$([regex]::Escape($Alias))\b")) {
    Ok "entry for '$Alias' already present in $cfgPath"
}
else {
    # 홈 아래의 키는 물결표로 적는다. 기계마다 홈 경로가 다르고 구분자도 다르다.
    $idForCfg = $KeyPath
    if ($KeyPath.StartsWith($HOME)) {
        $idForCfg = '~' + ($KeyPath.Substring($HOME.Length) -replace '\\', '/')
    }
    $block = @"

Host $Alias
  HostName $HostName

Host $Alias $HostName $IPv4
  User $User
  IdentityFile $idForCfg
  IdentitiesOnly yes
  StrictHostKeyChecking accept-new
"@ -replace "`r`n", "`n"

    $cfgDir = Split-Path -Parent $cfgPath
    if (-not (Test-Path $cfgDir)) { New-Item -ItemType Directory -Path $cfgDir -Force | Out-Null }
    # 이어 붙인다. 다른 호스트 항목이 이미 들어 있을 수 있다.
    Add-Content -Path $cfgPath -Value $block -NoNewline
    if (-not $IsWindows) { & chmod 600 $cfgPath }
    Ok "added '$Alias' to $cfgPath"
}

Step '3. Install the public key on the board'
Write-Host '  You will be asked for the board account password.' -ForegroundColor Yellow
# 이미 있는 줄은 건드리지 않는다. 다른 기계의 키가 함께 들어 있을 수 있다.
$install = @'
mkdir -p ~/.ssh && chmod 700 ~/.ssh
k=$(cat)
if grep -qxF "$k" ~/.ssh/authorized_keys 2>/dev/null; then
  echo "key already present"
else
  printf '%s\n' "$k" >> ~/.ssh/authorized_keys
  echo "key appended"
fi
chmod 600 ~/.ssh/authorized_keys
'@ -replace "`r`n", "`n"
$b64 = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($install))
$pub | & ssh -o StrictHostKeyChecking=accept-new $target "echo $b64 | base64 -d | sh"
if ($LASTEXITCODE -ne 0) { Bad 'could not install the key'; return }

Step '4. Verify password-free login'
$who = & ssh -T -n -i $KeyPath -o IdentitiesOnly=yes -o BatchMode=yes `
    -o PasswordAuthentication=no -o StrictHostKeyChecking=accept-new $target 'hostname' 2>&1
if ($LASTEXITCODE -ne 0) { Bad "key login failed: $((@($who) -join '; '))"; return }
Ok "logged in as $target, board reports '$who'"

Step '5. Open the LED files to the gpio group'
$rulePath = Join-Path $PSScriptRoot '99-led-permissions.rules'
if (-not (Test-Path $rulePath)) { throw "99-led-permissions.rules not found next to this script" }
$rule = (Get-Content $rulePath -Raw) -replace "`r`n", "`n"
$ruleB64 = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($rule))
Write-Host '  sudo on the board will ask for the same password.' -ForegroundColor Yellow
# sudo 가 프롬프트를 띄우려면 원격에 터미널이 있어야 한다. 그래서 -t 로 붙인다.
& ssh -t -i $KeyPath -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new $target `
    "echo $ruleB64 | base64 -d | sudo tee /etc/udev/rules.d/99-led-permissions.rules > /dev/null && sudo udevadm control --reload-rules && sudo udevadm trigger --subsystem-match=leds --action=add && echo applied"
if ($LASTEXITCODE -ne 0) { Bad 'could not install the udev rule'; return }

Step '6. Confirm the LEDs are writable without sudo'
$check = & ssh -T -n -i $KeyPath -o IdentitiesOnly=yes -o BatchMode=yes $target `
    'for d in ACT PWR; do [ -w /sys/class/leds/$d/brightness ] && echo "$d writable" || echo "$d NOT writable"; done' 2>&1
$check | ForEach-Object { Write-Host "  $_" }

Write-Host "`nDone." -ForegroundColor Green
Write-Host "  ssh    ssh $Alias"
Write-Host "  leds   ./Set-RpiLed.ps1 -Action Blink -Hz 2 -Seconds 3"
