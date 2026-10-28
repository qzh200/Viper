# 构建 Viper 的全部程序。
#
# 命令行是给人用的，剩下那几个是取证与真机验证的工具。它们都在默认的构建
# 与测试链里（没有构建标签藏着），所以 `go build ./...` 与 `go test ./...`
# 都会把它们算进去——以前它们被一个标签挡在默认构建之外，改坏了不会红。
#
# 用法：
#   .\build.ps1
$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
Push-Location $root
try {
    New-Item -ItemType Directory -Force -Path 'bin' | Out-Null

    $commands = @(
        'viper',      # 命令行
        'agcheck',    # 真机验证：读、写、整机读、逐项改一次再还原
        'fanproto',   # 风扇三个模式的取证
        'winkey',     # WIN_KEY 的语义与能不能写
        'gpucross',   # 显卡遥测与原厂服务的读数对照
        'smbiosdump' # 机器身份信息的原始结构
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
