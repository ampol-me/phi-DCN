@echo off
setlocal enabledelayedexpansion

echo Building Phi DCN Client...

:: ตั้งค่าตัวแปร
set GOOS=windows
set GOARCH=amd64
set OUTPUT=phi-dcn-windows.exe
set VERSION=1.0.0
set MANIFEST=manifest.xml

:: สร้างโฟลเดอร์ build
if not exist build mkdir build
if not exist build\release mkdir build\release

:: Build executable
echo Building executable...
go build -o build\%OUTPUT% -ldflags "-s -w -H=windowsgui" main.go

:: Embed manifest
echo Embedding manifest...
mt.exe -manifest %MANIFEST% -outputresource:build\%OUTPUT%;1

:: Code signing (ต้องมี Code Signing Certificate)
echo Signing executable...
signtool.exe sign /tr http://timestamp.digicert.com /td sha256 /fd sha256 /a build\%OUTPUT%

:: สร้างไฟล์ ZIP
echo Creating ZIP file...
powershell Compress-Archive -Path build\%OUTPUT% -DestinationPath build\release\phi-dcn-%VERSION%-windows.zip -Force

echo Build completed successfully!
echo Output: build\release\phi-dcn-%VERSION%-windows.zip 