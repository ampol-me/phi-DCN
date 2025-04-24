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

# Get version from git branch
VERSION=$(git rev-parse --abbrev-ref HEAD | sed 's/[^0-9.]//g')
if [ -z "$VERSION" ]; then
    VERSION="1.0.0"
fi

echo -e "${BLUE}🔨 Building phi-DCN version ${VERSION} for Windows${NC}"

# Clean previous builds
rm -rf build/win
rm -f build/phi-dcn-windows.exe
rm -f build/phi-dcn-windows-$VERSION.zip

# Create build directory
mkdir -p build/win

# Build for Windows
GOOS=windows GOARCH=amd64 go build -o build/win/phi-dcn-windows.exe ./client

# Create README
cat > build/win/README.txt << EOF
Phi DCN Bridge v$VERSION
=====================

This is the Windows version of Phi DCN Bridge.

Usage:
1. Open Command Prompt
2. Navigate to this directory
3. Run: phi-dcn-windows.exe

For more information, visit: https://github.com/your-repo/phi-DCN
EOF

# Create ZIP file
cd build/win
zip -r ../phi-dcn-windows-$VERSION.zip *
cd ../..

echo -e "${GREEN}✅ Build completed successfully!${NC}"
echo -e "${BLUE}📂 Binary files are available in the build/ directory${NC}"

# แสดงไฟล์ที่สร้าง
ls -la build/win

# หยุดรอการกดปุ่มจากผู้ใช้
echo ""
read -n 1 -s -r -p "กดปุ่มใดก็ได้เพื่อปิดหน้าต่าง..." 