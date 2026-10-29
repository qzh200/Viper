# 构建 Viper 工具套件中的所有可执行程序到 bin 目录
# 包含主命令行客户端 viper 以及各硬件探测与取证诊断工具

$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
Push-Location $root
try {
    # 确保输出目录 bin 存在
    New-Item -ItemType Directory -Force -Path 'bin' | Out-Null

    # 定义待编译的子程序列表
    $commands = @(
        'viper',
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
