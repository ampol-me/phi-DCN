#!/bin/bash

echo "Building for Windows..."

# สร้างโฟลเดอร์สำหรับเก็บไฟล์ build
mkdir -p build/win

# Build client
echo "Building client..."
cd client
GOOS=windows GOARCH=amd64 go build -o ../build/win/phi-dcn-client.exe .
if [ $? -ne 0 ]; then
    echo "Error building client"
    exit 1
fi
cd ..

# Build server
echo "Building server..."
cd xmlReader
GOOS=windows GOARCH=amd64 go build -o ../build/win/phi-dcn-server.exe .
if [ $? -ne 0 ]; then
    echo "Error building server"
    exit 1
fi
cd ..

echo "Build completed successfully!"
echo "Output files are in build/win directory:"
ls -l build/win/ 