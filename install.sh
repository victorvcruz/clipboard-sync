#!/bin/bash

# Clipboard Sync Installation Script for Ubuntu/Debian

set -e

echo "📋 Clipboard Sync - Installation Script"
echo "========================================"

# Check if running on Linux
if [[ "$OSTYPE" != "linux-gnu"* ]]; then
    echo "❌ This script is designed for Linux systems only."
    exit 1
fi

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21 or later first."
    echo "   Visit: https://golang.org/doc/install"
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+' || echo "0.0")
REQUIRED_VERSION="1.21"

if ! printf '%s\n%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V -C; then
    echo "❌ Go version $REQUIRED_VERSION or later is required. Current version: $GO_VERSION"
    exit 1
fi

echo "✅ Go version $GO_VERSION detected"

# Check system dependencies
echo "📦 Checking system dependencies..."

# Check if X11 libraries are available (required for native clipboard)
if pkg-config --exists x11 2>/dev/null; then
    echo "✅ X11 development libraries found"
elif ls /usr/lib/*/libX11.so* >/dev/null 2>&1 || ls /usr/lib/libX11.so* >/dev/null 2>&1; then
    echo "✅ X11 runtime libraries found"
else
    echo "⚠️  X11 libraries not found. Installing libx11-dev..."
    
    if command -v apt-get &> /dev/null; then
        echo "   Installing libx11-dev via apt-get..."
        sudo apt-get update
        sudo apt-get install -y libx11-dev
    elif command -v yum &> /dev/null; then
        echo "   Installing libX11-devel via yum..."
        sudo yum install -y libX11-devel
    elif command -v pacman &> /dev/null; then
        echo "   Installing libx11 via pacman..."
        sudo pacman -S --noconfirm libx11
    else
        echo "❌ Package manager not found. Please install X11 development libraries manually:"
        echo "   Ubuntu/Debian: sudo apt-get install libx11-dev"
        echo "   CentOS/RHEL:   sudo yum install libX11-devel"
        echo "   Arch Linux:    sudo pacman -S libx11"
        exit 1
    fi
    
    echo "✅ X11 libraries installed successfully"
fi

echo "ℹ️  Using native X11 clipboard implementation (no external tools required)"
echo "✅ System dependencies checked"

# Build the application
echo "🔨 Building Clipboard Sync..."
go mod tidy
go build -o clipboard-sync .

echo "✅ Build completed"

# Install to system PATH
echo "📍 Installing to /usr/local/bin..."
sudo cp clipboard-sync /usr/local/bin/
sudo chmod +x /usr/local/bin/clipboard-sync

echo "✅ Installation completed!"
echo ""
echo "🎉 Clipboard Sync is now installed and ready to use!"
echo ""

# Show current device IP
echo "📍 Your current IP addresses:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Get all network interfaces with IP addresses
ip addr show | grep -E "inet [0-9]" | grep -v "127.0.0.1" | while read -r line; do
    interface=$(echo "$line" | awk '{print $NF}')
    ip_addr=$(echo "$line" | awk '{print $2}' | cut -d'/' -f1)
    echo "  Interface $interface: $ip_addr"
done

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Usage:"
echo "  clipboard-sync -target <TARGET_IP>"
echo ""
echo "Example:"
echo "  clipboard-sync -target 192.168.1.100"
echo ""
echo "Options:"
echo "  -target <IP>        Target device IP address (required)"
echo "  -port <PORT>        Local port to listen on (default: 8080)"
echo "  -target-port <PORT> Target device port (default: 8080)"
echo "  -source-id <ID>     Source device identifier (auto-generated)"
echo ""
echo "For more information, run: clipboard-sync -h"