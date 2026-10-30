@echo off
chcp 65001 >nul
title Viper - Uninstall Acer Hardware Service

net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [Prompt] Requesting Administrator Privileges...
    powershell -Command "Start-Process '%~f0' -Verb RunAs"
    exit /b
)

cd /d "%~dp0"

echo ===================================================
echo     Viper - Uninstall Acer Hardware Service
echo ===================================================
echo.

echo [1/2] Stopping services...
net stop AASSvc >nul 2>&1
net stop AcerLightingService >nul 2>&1

echo [2/2] Removing Windows services...
sc delete AASSvc >nul 2>&1
sc delete AcerLightingService >nul 2>&1

echo.
echo ===================================================
echo [Done] Acer services have been stopped and removed.
echo ===================================================
echo.
pause
