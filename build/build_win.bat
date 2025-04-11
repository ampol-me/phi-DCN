@echo off
echo Building for Windows...

REM สร้างโฟลเดอร์สำหรับเก็บไฟล์ build
if not exist build\win mkdir build\win

REM Build client
echo Building client...
cd client
go build -o ..\build\win\phi-dcn-client.exe main.go
if errorlevel 1 (
    echo Error building client
    exit /b 1
)
cd ..

REM Build server
echo Building server...
cd xmlReader
go build -o ..\build\win\phi-dcn-server.exe main.go
if errorlevel 1 (
    echo Error building server
    exit /b 1
)
cd ..

echo Build completed successfully!
echo Output files are in build\win directory:
dir build\win 