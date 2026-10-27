<#
.SYNOPSIS
    固件那一侧的几项设置要在管理员权限下才连得上，这个脚本逐项验证它们。

.DESCRIPTION
    原厂 AcerAgentService 管不了的几项——风扇加速（CoolBoost）、风扇表、
    超频能力、锁 Windows 键——都在固件那一侧。那一侧要管理员权限，所以
    这个脚本要**以管理员身份**运行。

    脚本会先把每一项读出来记下，改一次，读回核对，再改回原来的样子。所以
    跑完机器上的状态和跑之前是一样的。

.EXAMPLE
    以管理员身份打开 PowerShell，然后：
    powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\firmware-check.ps1
#>

$ErrorActionPreference = 'Stop'

# 管理员身份检查。
$identity = [System.Security.Principal.WindowsIdentity]::GetCurrent()
$principal = New-Object System.Security.Principal.WindowsPrincipal($identity)
if (-not $principal.IsInRole([System.Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Host '这个脚本要管理员权限：请以管理员身份打开 PowerShell 再跑一次。' -ForegroundColor Red
    Write-Host '  右键点 PowerShell，选"以管理员身份运行"。' -ForegroundColor Red
    exit 1
}

$root = Split-Path -Parent $PSScriptRoot
$viper = Join-Path $root 'bin\viper.exe'
if (-not (Test-Path $viper)) {
    Write-Host '没有找到 viper.exe，先编译：' -ForegroundColor Yellow
    Write-Host '  go build -o bin\viper.exe ./cmd/viper' -ForegroundColor Yellow
    exit 1
}

function Invoke-Viper {
    param([Parameter(ValueFromRemainingArguments = $true)][string[]]$Arguments)
    $output = & $viper @Arguments 2>&1
    return [pscustomobject]@{
        Code   = $LASTEXITCODE
        Output = ($output | Out-String).Trim()
    }
}

# 报告一项的结果。
function Report {
    param([string]$Name, $Result, [string]$Expect = '')
    $ok = $Result.Code -eq 0
    if ($ok -and $Expect -ne '') {
        $ok = $Result.Output -match [regex]::Escape($Expect)
    }
    $mark = if ($ok) { '通过' } else { '没通过' }
    $colour = if ($ok) { 'Green' } else { 'Yellow' }
    Write-Host ("  {0,-22} {1}" -f $Name, $mark) -ForegroundColor $colour
    if (-not $ok) {
        Write-Host ("      " + ($Result.Output -replace "`r?`n", "`n      ")) -ForegroundColor DarkGray
    }
}

Write-Host ''
Write-Host '固件那一侧的几项设置（管理员权限下）' -ForegroundColor Cyan

# 一、报告：这一份报告里读不到的那几项，在这里应该读得到。
$diag = Invoke-Viper diag
Write-Host ''
Write-Host '一、机器的报告' -ForegroundColor Cyan
Write-Host ($diag.Output -replace "`r?`n", "`n  ")
$hasAdmin = $diag.Output -notmatch '读不到'

# 二、风扇加速：关 -> 开 -> 核对 -> 关回去。
Write-Host ''
Write-Host '二、风扇加速（CoolBoost）' -ForegroundColor Cyan
$fans = Invoke-Viper fans
$boostOff = $fans.Output -match 'CoolBoost : off'
if (-not $hasAdmin) {
    Write-Host '  读不到固件那一侧，跳过' -ForegroundColor Yellow
} else {
    Report '打开加速' (Invoke-Viper fan coolboost on) 'CoolBoost: on'
    Report '读回加速' (Invoke-Viper fans) 'CoolBoost : on'
    Report '关掉加速' (Invoke-Viper fan coolboost off) 'CoolBoost: off'
    Report '读回加速' (Invoke-Viper fans) 'CoolBoost : off'
}

# 三、锁 Windows 键：解锁 -> 锁上 -> 核对 -> 解锁。
Write-Host ''
Write-Host '三、锁 Windows 键' -ForegroundColor Cyan
if (-not $hasAdmin) {
    Write-Host '  读不到固件那一侧，跳过' -ForegroundColor Yellow
} else {
    Report '锁上这个键' (Invoke-Viper profile winkey on) 'Windows key   : on'
    Report '读回这个键' (Invoke-Viper profile) 'Windows key   : on'
    Report '解锁' (Invoke-Viper profile winkey off) 'Windows key   : off'
    Report '读回这个键' (Invoke-Viper profile) 'Windows key   : off'
}

Write-Host ''
Write-Host '跑完了。机器上的状态和跑之前是一样的。' -ForegroundColor Cyan
