@echo off
chcp 65001 >nul
title Viper - Install Acer Hardware Service

net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [Prompt] Requesting Administrator Privileges...
    powershell -Command "Start-Process '%~f0' -Verb RunAs"
    exit /b
)

cd /d "%~dp0"

echo ===================================================
echo     Viper - Install Acer Hardware Service
echo ===================================================
echo.

set "DRIVER_DIR=%~dp0driver"

if not exist "%DRIVER_DIR%\AcerAgentService.exe" (
    echo [Error] Driver files not found in driver directory!
    echo Please make sure you extracted the full Viper package.
    echo.
    pause
    exit /b 1
)

echo [1/3] Registering hardware component drivers...
if exist "%DRIVER_DIR%\PredatorService.inf" (
    pnputil /add-driver "%DRIVER_DIR%\PredatorService.inf" /install >nul 2>&1
    echo   - PredatorService.inf installed
)

echo [2/3] Registering Windows Service (AASSvc)...
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

echo [3/3] Starting hardware services...
net start AASSvc >nul 2>&1

echo.
echo ===================================================
echo [Success] Acer hardware core service is ready!
echo.
echo You can now run Viper without installing PredatorSense:
echo   bin\viperui.exe  (GUI)
echo.
echo The service is set to start automatically with Windows.
echo ===================================================
echo.
pause
