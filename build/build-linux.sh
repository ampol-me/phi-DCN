#!/bin/bash

# กำหนดสีสำหรับข้อความ
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Set working directory to script location
cd "$(dirname "$0")"

# Get version from git branch
VERSION=$(git rev-parse --abbrev-ref HEAD | sed 's/[^0-9.]//g')
if [ -z "$VERSION" ]; then
    VERSION="1.0.0"
fi

echo -e "${BLUE}🔨 Building phi-DCN version ${VERSION} for Linux${NC}"

# Check if linux directory exists
if [ ! -d "linux" ]; then
    echo "Error: linux directory not found"
    exit 1
fi

# Build for Linux
echo "Building for Linux..."
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o linux/phi-dcn-bridge ../client

# Create zip file
echo "Creating zip file..."
cd linux
zip -r ../phi-dcn-linux-$VERSION.zip phi-dcn-bridge

echo -e "${GREEN}✅ Build completed successfully!${NC}"
echo -e "${BLUE}📂 Binary files are available in the build/ directory${NC}" 