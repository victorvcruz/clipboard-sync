#!/bin/bash

set -e

echo "📋 Clipboard Sync - Installation"
echo "========================================"

if [[ "$OSTYPE" != "linux-gnu"* ]]; then
    echo "❌ This script is designed for Linux systems only."
    exit 1
fi

if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21 or later first."
    echo "   Visit: https://golang.org/doc/install"
    exit 1
fi

GO_VERSION=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+' || echo "0.0")
REQUIRED_VERSION="1.21"

if ! printf '%s\n%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V -C; then
    echo "❌ Go version $REQUIRED_VERSION or later is required. Current version: $GO_VERSION"
    exit 1
fi

if ! (pkg-config --exists x11 2>/dev/null || ls /usr/lib/*/libX11.so* >/dev/null 2>&1 || ls /usr/lib/libX11.so* >/dev/null 2>&1); then
    echo "📦 Installing X11 development libraries..."
    
    if command -v apt-get &> /dev/null; then
        sudo apt-get update -qq
        sudo apt-get install -y libx11-dev
    elif command -v yum &> /dev/null; then
        sudo yum install -y libX11-devel
    elif command -v pacman &> /dev/null; then
        sudo pacman -S --noconfirm libx11
    else
        echo "❌ Package manager not found. Please install X11 development libraries manually:"
        echo "   Ubuntu/Debian: sudo apt-get install libx11-dev"
        echo "   CentOS/RHEL:   sudo yum install libX11-devel"
        echo "   Arch Linux:    sudo pacman -S libx11"
        exit 1
    fi
    
    echo "✅ X11 libraries installed"
fi

echo "🔨 Building application..."
go mod tidy >/dev/null 2>&1
go build -o bin/clipboard-sync ./cmd/clipboard-sync/

echo "📍 Installing to system..."
sudo cp bin/clipboard-sync /usr/local/bin/
sudo chmod +x /usr/local/bin/clipboard-sync

echo "✅ Installation completed!"
echo ""

clipboard-sync --help