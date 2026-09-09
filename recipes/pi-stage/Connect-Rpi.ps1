#Requires -Version 7.0
<#
.SYNOPSIS
    Find the Raspberry Pi on a direct link and verify SSH key access.

.DESCRIPTION
    Tries the reachable paths in order and returns the first one that answers
    an actual SSH login. Runs on Windows, macOS and Linux under PowerShell 7.

.PARAMETER HostName
    mDNS name of the board. This is the portable path and is tried first.

.PARAMETER IPv4
    Static address the board holds. Only reachable when the host sits on the
    same subnet.

.PARAMETER MacPrefix
    OUI used to recognise the board during IPv6 link-local discovery.

.EXAMPLE
    ./Connect-Rpi.ps1

.EXAMPLE
    $r = ./Connect-Rpi.ps1 -Quiet -PassThru
    if ($r.Connected) { ssh -i $r.KeyPath -o IdentitiesOnly=yes $r.Target 'uptime' }
#>
[CmdletBinding()]
param(
    [string]$HostName      = 'sunnypi.local',
    [string]$IPv4          = '192.168.137.50',
    [string]$User          = 'sunny',
    [string]$KeyPath       = (Join-Path $HOME '.ssh' 'id_rpi_sunnypi'),
    [string]$MacPrefix     = 'b8:27:eb',
    [int]$TimeoutSeconds   = 8,
    [switch]$SkipLinkLocal,
    [switch]$Quiet,
    [switch]$PassThru
)

$ErrorActionPreference = 'Stop'

function Say  { param([string]$T, [string]$C = 'Gray') if (-not $Quiet) { Write-Host $T -ForegroundColor $C } }
function Ok   { param([string]$T) Say "  OK    $T" 'Green' }
function Bad  { param([string]$T) Say "  FAIL  $T" 'Red' }
function Note { param([string]$T) Say "  ..    $T" 'DarkGray' }

# 판정은 ssh 로만 한다. ICMP 는 방화벽이 흔히 막아서 연결 여부의 근거가 못 된다.
function Test-Target {
    param([string]$Target)
    $out = & ssh -T -n -i $KeyPath -o IdentitiesOnly=yes -o BatchMode=yes `
        -o PasswordAuthentication=no -o StrictHostKeyChecking=accept-new `
        -o ConnectTimeout=$TimeoutSeconds $Target 'hostname' 2>&1
    if ($LASTEXITCODE -eq 0) {
        return @{ Ok = $true; Detail = (@($out) | Select-Object -First 1) }
    }
    @{ Ok = $false; Detail = ((@($out) | Where-Object { $_ }) -join '; ') }
}

# 링크로컬 주소는 OS 마다 이웃 캐시를 읽는 도구가 다르다. 스코프 표기도 다르다 -
# Windows 는 인터페이스 인덱스를, 그 외는 인터페이스 이름을 붙인다.
function Get-PiLinkLocal {
    $hits = @()
    if ($IsWindows) {
        foreach ($a in Get-NetAdapter | Where-Object { $_.Status -eq 'Up' }) {
            & ping -6 -n 2 -w 700 "ff02::1%$($a.InterfaceIndex)" *> $null
        }
        Start-Sleep -Milliseconds 400
        $hits = Get-NetNeighbor -AddressFamily IPv6 -ErrorAction SilentlyContinue |
            Where-Object {
                $_.IPAddress -like 'fe80::*' -and
                ($_.LinkLayerAddress -replace '-', ':').ToLower().StartsWith($MacPrefix.ToLower())
            } |
            ForEach-Object { "$($_.IPAddress)%$($_.InterfaceIndex)" }
    }
    elseif ($IsMacOS) {
        foreach ($i in (& ifconfig -l) -split '\s+' | Where-Object { $_ }) {
            & ping6 -c 2 -i 0.3 "ff02::1%$i" *> $null
        }
        $hits = & ndp -an | ForEach-Object {
            $c = @(($_ -split '\s+') | Where-Object { $_ })
            if ($c.Count -ge 2 -and $c[0] -like 'fe80::*' -and
                $c[1].ToLower().StartsWith($MacPrefix.ToLower())) { $c[0] }
        }
    }
    else {
        $hits = & ip -6 neigh | ForEach-Object {
            if ($_ -match '^(fe80::\S+)\s+dev\s+(\S+).*lladdr\s+(\S+)' -and
                $Matches[3].ToLower().StartsWith($MacPrefix.ToLower())) {
                "$($Matches[1])%$($Matches[2])"
            }
        }
    }
    @($hits) | Where-Object { $_ } | Select-Object -Unique
}

$result = [ordered]@{
    Target = $null; Method = $null; Hostname = $null
    KeyPath = $KeyPath; Connected = $false; Reason = $null
}

Say ''
Say 'Raspberry Pi link check' 'White'
Say '-----------------------'

if (-not (Get-Command ssh -ErrorAction SilentlyContinue)) {
    Bad 'ssh client not found in PATH'
    $result.Reason = 'ssh missing'
    if ($PassThru) { [pscustomobject]$result }; return
}
if (-not (Test-Path $KeyPath)) {
    Bad "private key missing: $KeyPath"
    Say '  Run Initialize-RpiAccess.ps1 once on this machine to create and install it.'
    $result.Reason = 'key missing'
    if ($PassThru) { [pscustomobject]$result }; return
}

# mDNS 를 먼저 본다. 주소도 서브넷도 몰라도 되는 유일한 경로다.
$candidates = @(
    @{ Method = 'mDNS';  Target = "$User@$HostName" }
    @{ Method = 'IPv4';  Target = "$User@$IPv4" }
)

foreach ($c in $candidates) {
    Note "trying $($c.Method): $($c.Target)"
    $probe = Test-Target -Target $c.Target
    if ($probe.Ok) {
        Ok "$($c.Target) answered as '$($probe.Detail)'"
        $result.Target = $c.Target; $result.Method = $c.Method
        $result.Hostname = $probe.Detail; $result.Connected = $true
        break
    }
    Bad "$($c.Method) did not answer"
}

if (-not $result.Connected -and -not $SkipLinkLocal) {
    Note 'falling back to IPv6 link-local discovery'
    foreach ($ll in Get-PiLinkLocal) {
        $target = "$User@$ll"
        Note "trying link-local: $target"
        $probe = Test-Target -Target $target
        if ($probe.Ok) {
            Ok "$target answered as '$($probe.Detail)'"
            $result.Target = $target; $result.Method = 'IPv6 link-local'
            $result.Hostname = $probe.Detail; $result.Connected = $true
            break
        }
    }
}

Say ''
if ($result.Connected) {
    Say 'Connected.' 'Green'
    Say "  path : $($result.Method)"
    Say "  ssh  : ssh -i `"$KeyPath`" -o IdentitiesOnly=yes $($result.Target)"
} else {
    $result.Reason = 'no path answered'
    Say 'Not reachable.' 'Red'
    Say '  Check the cable and that the board has power.'
    if ($IsMacOS) {
        Say '  On macOS the static address is only reachable when this host also'
        Say '  sits on that subnet. See the README for the one-line networksetup call.'
    }
}

if ($PassThru) { [pscustomobject]$result }
