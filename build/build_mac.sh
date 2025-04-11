#!/bin/bash

echo "Building for macOS..."

# สร้างโฟลเดอร์สำหรับเก็บไฟล์ build
mkdir -p build/mac

# Build client
echo "Building client..."
cd client
go build -o ../build/mac/phi-dcn-client .
if [ $? -ne 0 ]; then
    echo "Error building client"
    exit 1
fi
cd ..

# Build server
echo "Building server..."
cd xmlReader
go build -o ../build/mac/phi-dcn-server .
if [ $? -ne 0 ]; then
    echo "Error building server"
    exit 1
fi
cd ..

echo "Build completed successfully!"
echo "Output files are in build/mac directory:"
ls -l build/mac/ 