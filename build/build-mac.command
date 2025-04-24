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

echo -e "${BLUE}🔨 Building phi-DCN version ${VERSION} for Mac M2${NC}"

# Clean previous builds
rm -rf build/mac
rm -f build/phi-dcn-mac
rm -f build/phi-dcn-mac.zip

# Create build directory
mkdir -p build/mac

# Build for Mac M2
GOOS=darwin GOARCH=arm64 go build -o build/mac/phi-dcn-mac ./client

# Create Info.plist
cat > build/mac/Info.plist << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>phi-dcn-mac</string>
    <key>CFBundleIdentifier</key>
    <string>com.phi.dcn</string>
    <key>CFBundleName</key>
    <string>Phi DCN</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>$VERSION</string>
    <key>CFBundleVersion</key>
    <string>$VERSION</string>
    <key>LSMinimumSystemVersion</key>
    <string>11.0</string>
</dict>
</plist>
EOF

# Create README
cat > build/mac/README.txt << EOF
Phi DCN Bridge v$VERSION
=====================

This is the Mac M2 version of Phi DCN Bridge.

Usage:
1. Open Terminal
2. Navigate to this directory
3. Run: ./phi-dcn-mac

For more information, visit: https://github.com/your-repo/phi-DCN
EOF

# Create ZIP file
cd build/mac
zip -r ../phi-dcn-mac-$VERSION.zip *
cd ../..

echo -e "${GREEN}✅ Build completed successfully!${NC}"
echo -e "${BLUE}📂 Binary files are available in the build/ directory${NC}"

# แสดงไฟล์ที่สร้าง
ls -la build/mac

# หยุดรอการกดปุ่มจากผู้ใช้
echo ""
read -n 1 -s -r -p "กดปุ่มใดก็ได้เพื่อปิดหน้าต่าง..." 