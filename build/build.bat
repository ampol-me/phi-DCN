@echo off
setlocal enabledelayedexpansion

REM Set working directory to script location
cd /d "%~dp0"

REM Get version from git branch
for /f "tokens=*" %%i in ('git rev-parse --abbrev-ref HEAD ^| sed "s/[^0-9.]//g"') do set VERSION=%%i
if "%VERSION%"=="" set VERSION=1.0.0

echo Building phi-DCN version %VERSION% for Windows...

REM Create win directory if it doesn't exist
if not exist "win" mkdir win

REM Build for Windows
echo Building for Windows...
set CGO_ENABLED=1
set GOOS=windows
set GOARCH=amd64
go build -tags "windows" -o win/phi-dcn-bridge.exe ..\client

REM Create zip file
echo Creating zip file...
cd win
powershell Compress-Archive -Path phi-dcn-bridge.exe -DestinationPath ..\phi-dcn-windows-%VERSION%.zip -Force

echo Build completed successfully!
pause 