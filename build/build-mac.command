#!/bin/bash

# รับตำแหน่งของไฟล์สคริปต์
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR/.."

# บันทึกวันที่และเวลาปัจจุบัน
BUILD_DATE=$(date +"%Y-%m-%d_%H-%M-%S")
VERSION="1.0.0"

# สร้างโฟลเดอร์สำหรับเก็บไฟล์ที่ build
mkdir -p build/client/mac
mkdir -p build/server/mac

echo "🔨 Building phi-DCN version ${VERSION} (${BUILD_DATE})"

# ตรวจสอบว่าเป็น Apple Silicon หรือ Intel
if [[ $(uname -m) == "arm64" ]]; then
  ARCH="arm64"
  echo "🍎 Detected Apple Silicon (M1/M2)"
else
  ARCH="amd64"
  echo "🍎 Detected Intel Mac"
fi

# Build สำหรับ Mac
echo "📦 Building client for macOS (${ARCH})..."
GOOS=darwin GOARCH=${ARCH} go build -o build/client/mac/phi-dcn-client ./client

echo "📦 Building server for macOS (${ARCH})..."
GOOS=darwin GOARCH=${ARCH} go build -o build/server/mac/phi-dcn-server ./server

echo "✅ Build completed successfully!"
echo "📂 Binary files are available in the build/ directory"

# แสดงไฟล์ที่สร้าง
ls -la build/client/mac build/server/mac

# หยุดรอการกดปุ่มจากผู้ใช้
echo ""
read -n 1 -s -r -p "กดปุ่มใดก็ได้เพื่อปิดหน้าต่าง..." 