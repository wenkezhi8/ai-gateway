#!/bin/bash

# Build script for AI Gateway

set -e

APP_NAME="ai-gateway"
VERSION=${VERSION:-"dev"}
BUILD_DIR="./bin"

echo "Building $APP_NAME version $VERSION..."

# Create build directory
mkdir -p $BUILD_DIR

# Build for current platform
echo "Building for current platform..."
go build -ldflags="-X main.Version=$VERSION" -o $BUILD_DIR/$APP_NAME ./cmd/gateway

echo "Build complete: $BUILD_DIR/$APP_NAME"
