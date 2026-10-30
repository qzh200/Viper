@echo off
chcp 65001 >nul
title Viper - 卸载原厂核心驱动服务

net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [提示] 正在请求管理员权限...
    powershell -Command "Start-Process '%~f0' -Verb RunAs"
    exit /b
)

cd /d "%~dp0"

echo ===================================================
echo     Viper - 卸载原厂核心驱动服务
echo ===================================================
echo.

echo [1/2] 正在停止服务...
net stop AASSvc >nul 2>&1
net stop AcerLightingService >nul 2>&1

echo [2/2] 正在清理系统服务...
sc delete AASSvc >nul 2>&1
sc delete AcerLightingService >nul 2>&1

echo.
echo ===================================================
echo [完成] 原厂核心服务已停止并从系统中移除。
echo ===================================================
echo.
pause
