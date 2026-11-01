$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
Push-Location $root
try {
    New-Item -ItemType Directory -Force -Path 'bin' | Out-Null

    $commands = @(
        'viper',
        'viperui',
        'agcheck',
        'fanproto',
        'winkey'
    )

    foreach ($name in $commands) {
        go build -o "bin/$name.exe" "./cmd/$name"
        if ($LASTEXITCODE -ne 0) {
            throw "构建 $name 失败"
        }
        Write-Host "built bin/$name.exe"
    }
}
finally {
    Pop-Location
}
