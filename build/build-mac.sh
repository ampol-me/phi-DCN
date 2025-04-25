#!/bin/bash

# Set working directory to script location
cd "$(dirname "$0")"

# Check if mac directory exists
if [ ! -d "mac" ]; then
    echo "Error: mac directory not found"
    exit 1
fi

# Build for macOS
echo "Building for macOS..."
GOOS=darwin GOARCH=amd64 go build -o mac/phi-dcn-bridge ..

# Create zip file
echo "Creating zip file..."
cd mac
zip -r ../phi-dcn-mac-0.0.7.zip phi-dcn-bridge

echo "Build completed successfully!" 