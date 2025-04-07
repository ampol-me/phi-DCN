@echo off
REM Windows build script for phi-DCN

echo Building phi-DCN for Windows

REM สร้างโฟลเดอร์สำหรับเก็บไฟล์ที่ build
mkdir build\client\windows
mkdir build\server\windows

REM บันทึกวันที่และเวลาปัจจุบัน
for /f "tokens=2 delims==" %%a in ('wmic OS Get localdatetime /value') do set dt=%%a
set YEAR=%dt:~0,4%
set MONTH=%dt:~4,2%
set DAY=%dt:~6,2%
set HOUR=%dt:~8,2%
set MINUTE=%dt:~10,2%
set SECOND=%dt:~12,2%
set BUILD_DATE=%YEAR%-%MONTH%-%DAY%_%HOUR%-%MINUTE%-%SECOND%
set VERSION=1.0.0

echo Building phi-DCN version %VERSION% (%BUILD_DATE%)

REM Build สำหรับ Windows
echo Building client for Windows...
set GOOS=windows
set GOARCH=amd64
go build -o build\client\windows\phi-dcn-client.exe .\client

echo Building server for Windows...
go build -o build\server\windows\phi-dcn-server.exe .\server

echo Build completed successfully!
echo Binary files are available in the build\ directory

REM แสดงไฟล์ที่สร้าง
dir build\client\windows
dir build\server\windows

pause 