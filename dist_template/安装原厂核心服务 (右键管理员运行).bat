@echo off
chcp 65001 >nul
title Viper - 安装原厂核心驱动服务

net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [提示] 正在请求管理员权限...
    powershell -Command "Start-Process '%~f0' -Verb RunAs"
    exit /b
)

cd /d "%~dp0"

echo ===================================================
echo     Viper - 宏碁原厂核心驱动服务一键安装器
echo ===================================================
echo.

set "DRIVER_DIR=%~dp0driver"

if not exist "%DRIVER_DIR%\AcerAgentService.exe" (
    echo [错误] 未在 driver 目录中找到核心驱动文件！
    echo 请确认完整解压了 Viper 发行包。
    echo.
    pause
    exit /b 1
)

echo [1/3] 正在注册底层内核驱动与设备组件...
if exist "%DRIVER_DIR%\PredatorService.inf" (
    pnputil /add-driver "%DRIVER_DIR%\PredatorService.inf" /install >nul 2>&1
    echo   - PredatorService.inf 驱动注册完毕
)

echo [2/3] 正在注册并配置 Windows 系统服务 (AASSvc)...
sc query AASSvc >nul 2>&1
if %errorlevel% neq 0 (
    sc create AASSvc binPath= "\"%DRIVER_DIR%\AcerCentralService.exe\"" start= auto DisplayName= "Acer Agent Service" >nul
) else (
    sc config AASSvc binPath= "\"%DRIVER_DIR%\AcerCentralService.exe\"" start= auto >nul
)

sc query AcerLightingService >nul 2>&1
if %errorlevel% neq 0 (
    if exist "%DRIVER_DIR%\AcerLightingService.exe" (
        sc create AcerLightingService binPath= "\"%DRIVER_DIR%\AcerLightingService.exe\"" start= auto DisplayName= "Acer Lighting Service" >nul
    )
)

echo [3/3] 正在启动原厂核心服务...
net start AASSvc >nul 2>&1

echo.
echo ===================================================
echo [成功] 原厂核心服务已安装并启动完毕！
echo.
echo 您现在可以免安装任何官方程序，直接双击运行：
echo   bin\viperui.exe  (图形控制中心)
echo.
echo 服务已设为开机自动启动，后续无需重复运行此安装脚本。
echo ===================================================
echo.
pause
