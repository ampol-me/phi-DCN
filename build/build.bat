@echo off
setlocal enabledelayedexpansion

echo Building Phi DCN Client...

:: ตั้งค่าตัวแปร
set GOOS=windows
set GOARCH=amd64
set OUTPUT=phi-dcn-windows.exe
set VERSION=1.0.0
set MANIFEST=client\app.manifest
set SDK_PATH=C:\Program Files (x86)\Windows Kits\10\bin\10.0.26100.0\x64

:: สร้างโฟลเดอร์ build
if not exist build mkdir build
if not exist build\release mkdir build\release

:: Build executable
echo Building executable...
set GOPATH=%CD%
set GO111MODULE=on
go mod tidy
go build -o build\%OUTPUT% -ldflags "-s -w -H=windowsgui" client\main.go

:: ตรวจสอบว่าไฟล์ executable ถูกสร้างขึ้นหรือไม่
if not exist build\%OUTPUT% (
    echo Error: Failed to create executable
    exit /b 1
)

:: คัดลอกไฟล์ config.ini
echo Copying config.ini...
if exist config\config.ini (
    copy config\config.ini build\config.ini
) else (
    echo Warning: config.ini not found
)

:: Embed manifest
echo Embedding manifest...
if exist %MANIFEST% (
    :: คัดลอกไฟล์ manifest ไปยังโฟลเดอร์ build
    copy %MANIFEST% build\app.manifest
    :: ใช้ mt.exe จาก Windows SDK
    if exist "%SDK_PATH%\mt.exe" (
        "%SDK_PATH%\mt.exe" -manifest build\app.manifest -outputresource:build\%OUTPUT%;1
    ) else (
        echo Warning: mt.exe not found, skipping manifest embedding
    )
) else (
    echo Warning: manifest file not found
)

:: Code signing (ต้องมี Code Signing Certificate)
echo Signing executable...
if exist build\%OUTPUT% (
    :: ตรวจสอบว่ามี certificate หรือไม่
    if exist "%SDK_PATH%\signtool.exe" (
        "%SDK_PATH%\signtool.exe" sign /tr http://timestamp.digicert.com /td sha256 /fd sha256 /a build\%OUTPUT%
    ) else (
        echo Warning: signtool.exe not found, skipping code signing
    )
) else (
    echo Error: Executable not found
    exit /b 1
)

:: สร้างไฟล์ ZIP
echo Creating ZIP file...
if exist build\%OUTPUT% (
    if exist build\config.ini (
        powershell Compress-Archive -Path build\%OUTPUT%,build\config.ini -DestinationPath build\release\phi-dcn-%VERSION%-windows.zip -Force
    ) else (
        powershell Compress-Archive -Path build\%OUTPUT% -DestinationPath build\release\phi-dcn-%VERSION%-windows.zip -Force
    )
) else (
    echo Error: Cannot create ZIP - executable not found
    exit /b 1
)

echo Build completed successfully!
echo Output: build\release\phi-dcn-%VERSION%-windows.zip 