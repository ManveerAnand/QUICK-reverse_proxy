#!/bin/bash

# Build and test script for QUIC Reverse Proxy

set -e

echo "🔨 Building QUIC Reverse Proxy..."

# Clean previous builds
echo "🧹 Cleaning previous builds..."
rm -rf build/ tmp/

# Download dependencies
echo "📦 Downloading dependencies..."
go mod download
go mod tidy

# Run tests
echo "🧪 Running tests..."
go test -v -race ./...

# Run linting if available
if command -v golangci-lint &> /dev/null; then
    echo "🔍 Running linter..."
    golangci-lint run
else
    echo "⚠️  golangci-lint not found, skipping linting"
fi

# Build the binary
echo "🔨 Building binary..."
mkdir -p build
go build -ldflags="-w -s" -o build/quic-proxy ./cmd/proxy

# Verify build
if [[ -f build/quic-proxy ]]; then
    echo "✅ Build successful!"
    echo "📁 Binary location: build/quic-proxy"
    
    # Show version info
    echo "ℹ️  Version info:"
    ./build/quic-proxy -version 2>/dev/null || echo "Version flag not implemented yet"
    
    echo ""
    echo "🚀 Ready to run! Try:"
    echo "   ./build/quic-proxy -config configs/proxy.yaml"
    echo ""
    echo "🐳 Or build Docker image:"
    echo "   make docker-build"
else
    echo "❌ Build failed!"
    exit 1
fi