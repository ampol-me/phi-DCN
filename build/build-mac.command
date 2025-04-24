#!/bin/bash

# กำหนดสีสำหรับข้อความ
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# รับตำแหน่งของไฟล์สคริปต์
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR/.."

# บันทึกวันที่และเวลาปัจจุบัน
BUILD_DATE=$(date +"%Y-%m-%d_%H-%M-%S")
VERSION="1.0.0"

# สร้างโฟลเดอร์สำหรับเก็บไฟล์ที่ build
mkdir -p build/client/mac

echo -e "${BLUE}🔨 Building phi-DCN version ${VERSION} (${BUILD_DATE})${NC}"

# ตรวจสอบว่าเป็น Apple Silicon หรือ Intel
if [[ $(uname -m) == "arm64" ]]; then
  ARCH="arm64"
  echo -e "${GREEN}🍎 Detected Apple Silicon (M1/M2)${NC}"
else
  ARCH="amd64"
  echo -e "${GREEN}🍎 Detected Intel Mac${NC}"
fi

# Build สำหรับ Mac
echo -e "${YELLOW}📦 Building client for macOS (${ARCH})...${NC}"
GOOS=darwin GOARCH=${ARCH} go build -o build/client/mac/phi-dcn-client ./client

echo -e "${YELLOW}📦 Building server for macOS (${ARCH})...${NC}"
GOOS=darwin GOARCH=${ARCH} go build -o build/server/mac/phi-dcn-server ./server

echo -e "${GREEN}✅ Build completed successfully!${NC}"
echo -e "${BLUE}📂 Binary files are available in the build/ directory${NC}"

# แสดงไฟล์ที่สร้าง
ls -la build/client/mac build/server/mac

# หยุดรอการกดปุ่มจากผู้ใช้
echo ""
read -n 1 -s -r -p "กดปุ่มใดก็ได้เพื่อปิดหน้าต่าง..." 