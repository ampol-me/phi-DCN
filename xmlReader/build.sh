#!/bin/bash

# สร้างโฟลเดอร์สำหรับเก็บไฟล์ที่ build
mkdir -p dist

# Build สำหรับ Mac
echo "Building for Mac..."
GOOS=darwin GOARCH=arm64 go build -o dist/dcn-client-mac tcp_client.go

# Build สำหรับ Windows
echo "Building for Windows..."
GOOS=windows GOARCH=amd64 go build -o dist/dcn-client.exe tcp_client.go

echo "✅ Build สำเร็จ!"
echo "ไฟล์ที่สร้าง:"
echo "- Mac: $(pwd)/dist/dcn-client-mac"
echo "- Windows: $(pwd)/dist/dcn-client.exe" 