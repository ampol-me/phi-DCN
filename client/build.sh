#!/bin/bash

# สร้างโฟลเดอร์สำหรับเก็บไฟล์ที่ build
mkdir -p build

# Build สำหรับ macOS
echo "Building for macOS..."
go build -o build/phi-dcn-bridge-mac main.go

# สร้าง .icns สำหรับ macOS (ต้องมี iconutil)
if [[ "$OSTYPE" == "darwin"* ]]; then
    echo "Creating .icns for macOS..."
    mkdir -p build/icon.iconset
    sips -z 16 16 assets/phi-xml-icon.png --out build/icon.iconset/icon_16x16.png
    sips -z 32 32 assets/phi-xml-icon.png --out build/icon.iconset/icon_16x16@2x.png
    sips -z 32 32 assets/phi-xml-icon.png --out build/icon.iconset/icon_32x32.png
    sips -z 64 64 assets/phi-xml-icon.png --out build/icon.iconset/icon_32x32@2x.png
    sips -z 128 128 assets/phi-xml-icon.png --out build/icon.iconset/icon_128x128.png
    sips -z 256 256 assets/phi-xml-icon.png --out build/icon.iconset/icon_128x128@2x.png
    sips -z 256 256 assets/phi-xml-icon.png --out build/icon.iconset/icon_256x256.png
    sips -z 512 512 assets/phi-xml-icon.png --out build/icon.iconset/icon_256x256@2x.png
    sips -z 512 512 assets/phi-xml-icon.png --out build/icon.iconset/icon_512x512.png
    sips -z 1024 1024 assets/phi-xml-icon.png --out build/icon.iconset/icon_512x512@2x.png
    iconutil -c icns build/icon.iconset -o build/phi-xml-icon.icns
    rm -rf build/icon.iconset
fi

# Build สำหรับ Windows
echo "Building for Windows..."

# ติดตั้ง tools ที่จำเป็น
go install github.com/akavel/rsrc@latest

# สร้าง resource file
echo "Creating Windows resource file..."
rsrc -manifest app.manifest -ico assets/phi-xml-icon.ico -o rsrc.syso

# Build with resources
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 go build -ldflags "-H windowsgui" -o build/phi-dcn-bridge.exe

# ลบไฟล์ชั่วคราว
rm -f rsrc.syso

# คัดลอกไฟล์ที่จำเป็น
echo "Copying required files..."
cp -r assets build/
cp -r web build/

echo "Build completed!"
echo "Files are in the 'build' directory:"
ls -l build/ 