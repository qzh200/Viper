$ErrorActionPreference = "Stop"

Write-Host ">>> [1/4] 编译全部 Viper 可执行程序..." -ForegroundColor Cyan
& powershell -File .\build.ps1
if ($LASTEXITCODE -ne 0) {
    throw "编译失败，停止打包。"
}

$distDir = Join-Path $PSScriptRoot "dist\Viper"
if (Test-Path $distDir) {
    Write-Host ">>> 清理旧的发行目录..." -ForegroundColor Gray
    Remove-Item -Recurse -Force $distDir
}

New-Item -ItemType Directory -Path (Join-Path $distDir "bin") -Force | Out-Null
New-Item -ItemType Directory -Path (Join-Path $distDir "driver") -Force | Out-Null

Write-Host ">>> [2/4] 复制二进制程序到 dist/Viper/bin/..." -ForegroundColor Cyan
Copy-Item (Join-Path $PSScriptRoot "bin\*.exe") (Join-Path $distDir "bin") -Force

Write-Host ">>> [3/4] 提取官方核心驱动组件 (AgentService\driver)..." -ForegroundColor Cyan
$sourceDriver = "e:\PredatorSenseGo\PredatorSense\AgentService\driver"
if (-not (Test-Path $sourceDriver)) {
    throw "未找到官方驱动源目录: $sourceDriver"
}
Copy-Item -Recurse (Join-Path $sourceDriver "*") (Join-Path $distDir "driver") -Force

$devconPath = "e:\PredatorSenseGo\PredatorSense\AgentService\devcon.exe"
if (Test-Path $devconPath) {
    Copy-Item $devconPath (Join-Path $distDir "driver") -Force
}

Write-Host ">>> [4/4] 复制一键管理脚本与说明文件..." -ForegroundColor Cyan
$templateDir = Join-Path $PSScriptRoot "dist_template"
Copy-Item (Join-Path $templateDir "*") $distDir -Force

$driverSize = (Get-ChildItem (Join-Path $distDir "driver") -Recurse | Measure-Object -Property Length -Sum).Sum / 1MB
Write-Host ">>> 打包成功！发行目录: dist\Viper" -ForegroundColor Green
Write-Host ("   - 官方核心驱动大小: {0:N2} MB (官方原完整安装包约 500MB)" -f $driverSize) -ForegroundColor Yellow
Write-Host "   - Viper 核心程序大小: 正常编译输出 (bin/)" -ForegroundColor Yellow
