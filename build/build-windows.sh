#!/bin/bash

# กำหนดสีสำหรับข้อความ
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# รับตำแหน่งของไฟล์สคริปต์
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$ROOT_DIR"

# Get version from git branch
VERSION=$(git rev-parse --abbrev-ref HEAD | sed 's/[^0-9.]//g')
if [ -z "$VERSION" ]; then
    VERSION="1.0.0"
fi

echo -e "${BLUE}🔨 Building phi-DCN version ${VERSION} for Windows${NC}"

# ตั้งค่าตัวแปร
APP_NAME="phi-dcn-bridge"
PLATFORM="windows"
ARCH="amd64"
OUTPUT_DIR="$SCRIPT_DIR/dist/windows"
ICON_SRC="$ROOT_DIR/assets/icon.ico"
ICON_DST="$OUTPUT_DIR/assets/icon.ico"

# สร้างโฟลเดอร์ output ถ้ายังไม่มี
mkdir -p "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR/assets"

# คัดลอกไฟล์ไอคอน
if [ -f "$ICON_SRC" ]; then
    cp "$ICON_SRC" "$ICON_DST"
    echo "✅ Copied icon file to $ICON_DST"
else
    echo -e "${YELLOW}⚠️ Warning: Icon file not found at $ICON_SRC${NC}"
fi

# คัดลอกไฟล์ config.ini
if [ -f "$ROOT_DIR/config/config.ini" ]; then
    cp "$ROOT_DIR/config/config.ini" "$OUTPUT_DIR/config/config.ini"
    echo "✅ Copied config.ini to $OUTPUT_DIR/config/config.ini"
else
    echo -e "${YELLOW}⚠️ Warning: config.ini not found at $ROOT_DIR/config/config.ini${NC}"
fi

# คัดลอกไฟล์ license จาก backup
BACKUP_DIR="$ROOT_DIR/config/backup"
if [ -d "$BACKUP_DIR" ]; then
    # หาไฟล์ license ล่าสุด
    LATEST_LICENSE=$(ls -t "$BACKUP_DIR"/license_*.json 2>/dev/null | head -n1)
    if [ -n "$LATEST_LICENSE" ]; then
        cp "$LATEST_LICENSE" "$OUTPUT_DIR/config/backup/license.json"
        echo "✅ Copied latest license from backup to $OUTPUT_DIR/config/backup/license.json"
    else
        echo -e "${YELLOW}⚠️ Warning: No license files found in $BACKUP_DIR${NC}"
    fi
else
    echo -e "${YELLOW}⚠️ Warning: backup directory not found at $BACKUP_DIR${NC}"
fi

# คัดลอกไฟล์ README.md
if [ -f "$ROOT_DIR/README.md" ]; then
    cp "$ROOT_DIR/README.md" "$OUTPUT_DIR/"
    echo "✅ Copied README.md to $OUTPUT_DIR/"
else
    echo -e "${YELLOW}⚠️ Warning: README.md not found at $ROOT_DIR/README.md${NC}"
fi

# สร้างไฟล์ .exe
echo "🚀 Building Windows executable..."
cd "$ROOT_DIR"
GOOS=$PLATFORM GOARCH=$ARCH go build -o "$OUTPUT_DIR/$APP_NAME.exe" ./client

# ตรวจสอบว่าสร้างไฟล์สำเร็จหรือไม่
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Build successful!${NC}"
    echo "📦 Output directory: $OUTPUT_DIR"
    echo "📄 Files:"
    ls -la "$OUTPUT_DIR"
else
    echo -e "${RED}❌ Build failed!${NC}"
    exit 1
fi

# Create zip file
echo "Creating zip file..."
cd "$OUTPUT_DIR"
zip -r "$SCRIPT_DIR/phi-dcn-windows-$VERSION.zip" .

echo -e "${GREEN}✅ Build completed successfully!${NC}"
echo -e "${BLUE}📂 Binary files are available in the build/dist/windows directory${NC}"

# แสดงไฟล์ที่สร้าง
ls -la "$SCRIPT_DIR/phi-dcn-windows-$VERSION.zip"

# หยุดรอการกดปุ่มจากผู้ใช้
echo ""
read -n 1 -s -r -p "กดปุ่มใดก็ได้เพื่อปิดหน้าต่าง..." 

# Set working directory to script location
cd "$(dirname "$0")"

# Check if win directory exists
if [ ! -d "win" ]; then
    echo "Error: win directory not found"
    exit 1
fi

# Build for Windows
echo "Building for Windows..."
GOOS=windows GOARCH=amd64 go build -o win/phi-dcn-bridge.exe ..

# Create zip file
echo "Creating zip file..."
cd win
zip -u ../phi-dcn-windows-0.0.7.zip phi-dcn-bridge.exe

echo "Build completed successfully!" 