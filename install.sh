#!/usr/bin/env bash

# Kopilot installer script
# Usage: curl -fsSL https://raw.githubusercontent.com/e9169/kopilot/main/install.sh | bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# GitHub repository
REPO="e9169/kopilot"
BINARY_NAME="kopilot"

# Detect OS
OS="$(uname -s)"
case "$OS" in
    Linux*)     OS="linux";;
    Darwin*)    OS="darwin";;
    MINGW*|MSYS*|CYGWIN*) OS="windows";;
    *)
        echo -e "${RED}Error: Unsupported operating system: $OS${NC}"
        exit 1
        ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64|amd64)   ARCH="amd64";;
    aarch64|arm64)  ARCH="arm64";;
    *)
        echo -e "${RED}Error: Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

echo -e "${BLUE}╔═══════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║       Kopilot Installation Script            ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}Detected platform:${NC} ${GREEN}$OS/$ARCH${NC}"
echo ""

# Get latest release version
echo -e "${YELLOW}→${NC} Fetching latest release..."
LATEST_RELEASE=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_RELEASE" ]; then
    echo -e "${RED}Error: Could not fetch latest release version${NC}"
    exit 1
fi

echo -e "${GREEN}✓${NC} Latest version: ${GREEN}$LATEST_RELEASE${NC}"

# Construct download URL
if [ "$OS" = "windows" ]; then
    ARCHIVE_EXT="zip"
else
    ARCHIVE_EXT="tar.gz"
fi

ARCHIVE_NAME="${BINARY_NAME}_${LATEST_RELEASE}_${OS}_${ARCH}.${ARCHIVE_EXT}"
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_RELEASE/$ARCHIVE_NAME"

# Create temporary directory
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

echo -e "${YELLOW}→${NC} Downloading $ARCHIVE_NAME..."
if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/$ARCHIVE_NAME"; then
    echo -e "${RED}Error: Failed to download release${NC}"
    echo -e "${RED}URL: $DOWNLOAD_URL${NC}"
    exit 1
fi

echo -e "${GREEN}✓${NC} Download complete"

# Extract archive
echo -e "${YELLOW}→${NC} Extracting archive..."
cd "$TMP_DIR"
if [ "$OS" = "windows" ]; then
    unzip -q "$ARCHIVE_NAME"
else
    tar -xzf "$ARCHIVE_NAME"
fi

# Find binary (it might be in the archive directly or in a subdirectory)
BINARY_PATH=""
if [ -f "$BINARY_NAME" ]; then
    BINARY_PATH="$BINARY_NAME"
elif [ -f "$BINARY_NAME.exe" ]; then
    BINARY_PATH="$BINARY_NAME.exe"
else
    # Search for binary in extracted files
    BINARY_PATH=$(find . -name "$BINARY_NAME" -o -name "$BINARY_NAME.exe" | head -n 1)
fi

if [ -z "$BINARY_PATH" ]; then
    echo -e "${RED}Error: Binary not found in archive${NC}"
    exit 1
fi

echo -e "${GREEN}✓${NC} Extraction complete"

# Determine install location
if [ "$OS" = "windows" ]; then
    # For Windows, suggest adding to PATH manually
    INSTALL_DIR="$HOME/bin"
    mkdir -p "$INSTALL_DIR"
    INSTALL_PATH="$INSTALL_DIR/$BINARY_NAME.exe"
else
    # Try /usr/local/bin first, fall back to ~/.local/bin
    if [ -w "/usr/local/bin" ]; then
        INSTALL_DIR="/usr/local/bin"
    elif [ "$(id -u)" -eq 0 ]; then
        INSTALL_DIR="/usr/local/bin"
    else
        # Check if user wants to use sudo
        if command -v sudo >/dev/null 2>&1; then
            echo -e "${YELLOW}→${NC} Installation requires sudo access for /usr/local/bin"
            echo -n "  Use sudo to install to /usr/local/bin? [Y/n] "
            read -r response
            if [[ "$response" =~ ^[Nn]$ ]]; then
                INSTALL_DIR="$HOME/.local/bin"
                mkdir -p "$INSTALL_DIR"
            else
                INSTALL_DIR="/usr/local/bin"
            fi
        else
            INSTALL_DIR="$HOME/.local/bin"
            mkdir -p "$INSTALL_DIR"
        fi
    fi
    INSTALL_PATH="$INSTALL_DIR/$BINARY_NAME"
fi

# Install binary
echo -e "${YELLOW}→${NC} Installing to $INSTALL_PATH..."
if [ -w "$INSTALL_DIR" ] || [ "$(id -u)" -eq 0 ]; then
    mv "$BINARY_PATH" "$INSTALL_PATH"
    chmod +x "$INSTALL_PATH"
else
    sudo mv "$BINARY_PATH" "$INSTALL_PATH"
    sudo chmod +x "$INSTALL_PATH"
fi

echo -e "${GREEN}✓${NC} Installation complete"
echo ""

# Verify installation
if command -v "$BINARY_NAME" >/dev/null 2>&1; then
    VERSION=$("$BINARY_NAME" --version 2>&1 || true)
    echo -e "${GREEN}✓${NC} kopilot is ready to use!"
    echo -e "${BLUE}  Version:${NC} $VERSION"
else
    echo -e "${YELLOW}⚠${NC}  kopilot installed to $INSTALL_PATH"
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        echo -e "${YELLOW}⚠${NC}  $INSTALL_DIR is not in your PATH"
        echo ""
        echo -e "${BLUE}To use kopilot, add this to your shell profile:${NC}"
        echo -e "${GREEN}  export PATH=\"\$PATH:$INSTALL_DIR\"${NC}"
        echo ""
        echo -e "${BLUE}Or run directly:${NC}"
        echo -e "${GREEN}  $INSTALL_PATH${NC}"
    fi
fi

echo ""
echo -e "${BLUE}╔═══════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  Get started with: ${GREEN}kopilot${BLUE}                  ║${NC}"
echo -e "${BLUE}║  Documentation: ${YELLOW}https://kopilot.dev${BLUE}      ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════╝${NC}"
