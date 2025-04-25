#!/bin/bash

# กำหนดสีสำหรับข้อความ
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# รับตำแหน่งของไฟล์สคริปต์
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

# Get version from git branch
VERSION=$(git rev-parse --abbrev-ref HEAD | sed 's/[^0-9.]//g')
if [ -z "$VERSION" ]; then
    VERSION="1.0.0"
fi

echo -e "${BLUE}🔨 Building phi-DCN version ${VERSION} for Windows${NC}"

# Create win directory if it doesn't exist
mkdir -p win

# Build for Windows
echo "Building for Windows..."
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -tags "windows" -o win/phi-dcn-bridge.exe ../client

# Create zip file
echo "Creating zip file..."
cd win
zip -u ../phi-dcn-windows-$VERSION.zip phi-dcn-bridge.exe

echo -e "${GREEN}✅ Build completed successfully!${NC}"
echo -e "${BLUE}📂 Binary files are available in the build/ directory${NC}"

# แสดงไฟล์ที่สร้าง
ls -la win

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