#!/bin/bash

# บันทึกวันที่และเวลาปัจจุบัน
BUILD_DATE=$(date +"%Y-%m-%d_%H-%M-%S")
VERSION="1.0.0"

# สร้างโฟลเดอร์สำหรับเก็บไฟล์ที่ build
mkdir -p build/client/mac
mkdir -p build/client/windows
mkdir -p build/server/mac
mkdir -p build/server/windows

echo "🔨 Building phi-DCN version ${VERSION} (${BUILD_DATE})"

# Build สำหรับ Mac
echo "📦 Building client for macOS (arm64)..."
GOOS=darwin GOARCH=arm64 go build -o build/client/mac/phi-dcn-client-arm64 ./client
GOOS=darwin GOARCH=amd64 go build -o build/client/mac/phi-dcn-client-amd64 ./client

echo "📦 Building server for macOS (arm64)..."
GOOS=darwin GOARCH=arm64 go build -o build/server/mac/phi-dcn-server-arm64 ./server
GOOS=darwin GOARCH=amd64 go build -o build/server/mac/phi-dcn-server-amd64 ./server

# Build สำหรับ Windows
echo "📦 Building client for Windows..."
GOOS=windows GOARCH=amd64 go build -o build/client/windows/phi-dcn-client.exe ./client

echo "📦 Building server for Windows..."
GOOS=windows GOARCH=amd64 go build -o build/server/windows/phi-dcn-server.exe ./server

echo "✅ Build completed successfully!"
echo "📂 Binary files are available in the build/ directory" 