@echo off

:: Set working directory to script location
cd /d "%~dp0"

:: Check if windows directory exists
if not exist "windows" (
    echo Error: windows directory not found
    exit /b 1
)

:: Build for Windows
echo Building for Windows...
set GOOS=windows
set GOARCH=amd64
go build -o windows/phi-dcn-client.exe ..

:: Create zip file
echo Creating zip file...
cd windows
powershell Compress-Archive -Path phi-dcn-client.exe -DestinationPath ..\phi-dcn-windows-0.0.7.zip -Force

echo Build completed successfully! 