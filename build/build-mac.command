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

echo -e "${BLUE}🔨 Building phi-DCN version ${VERSION} for Mac${NC}"

# Check if mac directory exists
if [ ! -d "mac" ]; then
    echo "Error: mac directory not found"
    exit 1
fi

# Build for Mac
echo "Building for Mac..."
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -o mac/phi-dcn-bridge ../client

# Create zip file
echo "Creating zip file..."
cd mac
zip -r ../phi-dcn-mac-$VERSION.zip phi-dcn-bridge

echo -e "${GREEN}✅ Build completed successfully!${NC}"
echo -e "${BLUE}📂 Binary files are available in the build/ directory${NC}" 