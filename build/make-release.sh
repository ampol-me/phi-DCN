#!/bin/bash

# รับตำแหน่งของไฟล์สคริปต์
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR/.."

# บันทึกวันที่และเวลาปัจจุบัน
BUILD_DATE=$(date +"%Y-%m-%d")
VERSION="1.0.0"

# ตรวจสอบการมีอยู่ของไฟล์ก่อนสร้าง release
if [ ! -f "build/client/mac/phi-dcn-client-arm64" ] || [ ! -f "build/server/mac/phi-dcn-server-arm64" ] || [ ! -f "build/client/windows/phi-dcn-client.exe" ] || [ ! -f "build/server/windows/phi-dcn-server.exe" ]; then
    echo "❌ ไม่พบไฟล์ที่ถูก build ครบทุกไฟล์ โปรดทำการ build ก่อนสร้าง release"
    echo "   โดยใช้คำสั่ง ./build/build.sh"
    exit 1
fi

echo "🔨 สร้าง release สำหรับ phi-DCN เวอร์ชัน ${VERSION} (${BUILD_DATE})"

# สร้างโฟลเดอร์สำหรับเก็บไฟล์ release
RELEASE_DIR="build/release"
mkdir -p "$RELEASE_DIR"

# --------------------------------------------------------
# แพ็คเกจสำหรับ macOS (Apple Silicon - M1/M2)
# --------------------------------------------------------
echo "📦 สร้างแพ็คเกจสำหรับ macOS (Apple Silicon)..."
MAC_ARM_DIR="$RELEASE_DIR/phi-dcn-${VERSION}-mac-arm64"
mkdir -p "$MAC_ARM_DIR"

# คัดลอกไฟล์ไปยังโฟลเดอร์ชั่วคราว
cp "build/client/mac/phi-dcn-client-arm64" "$MAC_ARM_DIR/phi-dcn-client"
cp "build/server/mac/phi-dcn-server-arm64" "$MAC_ARM_DIR/phi-dcn-server"
cp README.md "$MAC_ARM_DIR/"

# สร้างไฟล์ ZIP
cd "$RELEASE_DIR"
zip -r "phi-dcn-${VERSION}-mac-arm64.zip" "phi-dcn-${VERSION}-mac-arm64"
cd -
rm -rf "$MAC_ARM_DIR"

# --------------------------------------------------------
# แพ็คเกจสำหรับ macOS (Intel)
# --------------------------------------------------------
echo "📦 สร้างแพ็คเกจสำหรับ macOS (Intel)..."
MAC_INTEL_DIR="$RELEASE_DIR/phi-dcn-${VERSION}-mac-amd64"
mkdir -p "$MAC_INTEL_DIR"

# คัดลอกไฟล์ไปยังโฟลเดอร์ชั่วคราว
cp "build/client/mac/phi-dcn-client-amd64" "$MAC_INTEL_DIR/phi-dcn-client"
cp "build/server/mac/phi-dcn-server-amd64" "$MAC_INTEL_DIR/phi-dcn-server"
cp README.md "$MAC_INTEL_DIR/"

# สร้างไฟล์ ZIP
cd "$RELEASE_DIR"
zip -r "phi-dcn-${VERSION}-mac-amd64.zip" "phi-dcn-${VERSION}-mac-amd64"
cd -
rm -rf "$MAC_INTEL_DIR"

# --------------------------------------------------------
# แพ็คเกจสำหรับ Windows
# --------------------------------------------------------
echo "📦 สร้างแพ็คเกจสำหรับ Windows..."
WIN_DIR="$RELEASE_DIR/phi-dcn-${VERSION}-windows"
mkdir -p "$WIN_DIR"

# คัดลอกไฟล์ไปยังโฟลเดอร์ชั่วคราว
cp "build/client/windows/phi-dcn-client.exe" "$WIN_DIR/"
cp "build/server/windows/phi-dcn-server.exe" "$WIN_DIR/"
cp README.md "$WIN_DIR/"

# สร้างไฟล์ ZIP
cd "$RELEASE_DIR"
zip -r "phi-dcn-${VERSION}-windows.zip" "phi-dcn-${VERSION}-windows"
cd -
rm -rf "$WIN_DIR"

echo "✅ สร้าง release เสร็จสมบูรณ์!"
echo "📂 ไฟล์ ZIP พร้อมสำหรับการแจกจ่ายอยู่ในโฟลเดอร์ $RELEASE_DIR"
ls -la "$RELEASE_DIR" 