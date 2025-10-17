#!/bin/bash
set -e

# Get git information
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)

# Generate version based on git tag or build timestamp
if GIT_TAG=$(git describe --tags --exact-match 2>/dev/null); then
    # Use exact git tag if available
    VERSION=${1:-$GIT_TAG}
else
    # Generate version from timestamp if no tag
    VERSION=${1:-"v0.1.0-$(date -u +%Y%m%d-%H%M%S)"}
fi

echo "Building mcpproxy"
echo "  Version:    $VERSION"
echo "  Commit:     $GIT_COMMIT"
echo "  Branch:     $GIT_BRANCH"
echo "  Build Time: $BUILD_TIME"

LDFLAGS="-X main.version=$VERSION -X main.buildTime=$BUILD_TIME -X main.gitCommit=$GIT_COMMIT -X main.gitBranch=$GIT_BRANCH -s -w"

# Build for current platform (with CGO for tray support if needed)
echo "Building for current platform..."
go build -ldflags "$LDFLAGS" -o mcpproxy ./cmd/mcpproxy

# Build for Linux (with CGO disabled to avoid systray issues)
echo "Building for Linux..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$LDFLAGS" -o mcpproxy-linux-amd64 ./cmd/mcpproxy

# Build for Windows (with CGO disabled to avoid systray issues)
echo "Building for Windows..."
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$LDFLAGS" -o mcpproxy-windows-amd64.exe ./cmd/mcpproxy

# Build for macOS (only when running on macOS due to systray dependencies)
if [[ "$OSTYPE" == "darwin"* ]]; then
    echo "Building for macOS..."
    # Build for current architecture (native)
    go build -ldflags "$LDFLAGS" -o mcpproxy-darwin-$(uname -m) ./cmd/mcpproxy
    
    # Try cross-compilation for other macOS architectures (may fail due to systray)
    if [[ "$(uname -m)" == "arm64" ]]; then
        echo "Attempting to build for macOS amd64..."
        GOOS=darwin GOARCH=amd64 go build -ldflags "$LDFLAGS" -o mcpproxy-darwin-amd64 ./cmd/mcpproxy 2>/dev/null || echo "macOS amd64 cross-compilation failed (expected due to systray dependencies)"
    else
        echo "Attempting to build for macOS arm64..."
        GOOS=darwin GOARCH=arm64 go build -ldflags "$LDFLAGS" -o mcpproxy-darwin-arm64 ./cmd/mcpproxy 2>/dev/null || echo "macOS arm64 cross-compilation failed (expected due to systray dependencies)"
    fi
else
    echo "Skipping macOS builds (not running on macOS - systray dependencies prevent cross-compilation)"
fi

echo "Build complete!"
echo "Available binaries:"
ls -la mcpproxy*

echo ""
echo "Test version info:"
./mcpproxy --version 