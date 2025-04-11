@echo off

REM สร้างโฟลเดอร์สำหรับเก็บไฟล์ที่ build
if not exist dist mkdir dist

REM Build สำหรับ Windows
echo Building for Windows...
set GOOS=windows
set GOARCH=amd64
go build -o dist\dcn-client.exe tcp_client.go

REM Build สำหรับ Mac
echo Building for Mac...
set GOOS=darwin
set GOARCH=arm64
go build -o dist\dcn-client-mac tcp_client.go

echo ✅ Build สำเร็จ!
echo ไฟล์ที่สร้าง:
echo - Windows: %CD%\dist\dcn-client.exe
echo - Mac: %CD%\dist\dcn-client-mac 