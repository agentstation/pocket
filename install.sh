#!/usr/bin/env bash
# Pocket installation script
# This script downloads and installs the latest version of Pocket

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
GITHUB_REPO="agentstation/pocket"
BINARY_NAME="pocket"
INSTALL_DIR="${POCKET_INSTALL_DIR:-/usr/local/bin}"

# Functions
info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

success() {
    echo -e "${GREEN}✓${NC} $1"
}

warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1" >&2
}

# Detect OS and architecture
detect_os() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    case "$OS" in
        darwin) OS="darwin" ;;
        linux) OS="linux" ;;
        mingw*|msys*|cygwin*) OS="windows" ;;
        *) error "Unsupported operating system: $OS"; exit 1 ;;
    esac
}

detect_arch() {
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64|amd64) ARCH="x86_64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        i386|i686) ARCH="i386" ;;
        *) error "Unsupported architecture: $ARCH"; exit 1 ;;
    esac
}

# Get the latest release version
get_latest_version() {
    info "Fetching latest version..."
    VERSION=$(curl -s "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$VERSION" ]; then
        error "Failed to fetch latest version"
        exit 1
    fi
    success "Latest version: $VERSION"
}

# Construct download URL
construct_download_url() {
    if [ "$OS" = "windows" ]; then
        ARCHIVE_NAME="${BINARY_NAME}-${OS}-${ARCH}.zip"
    else
        ARCHIVE_NAME="${BINARY_NAME}-${OS}-${ARCH}.tar.gz"
    fi
    DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}/${ARCHIVE_NAME}"
}

# Download the binary
download_binary() {
    info "Downloading Pocket ${VERSION} for ${OS}/${ARCH}..."
    
    TEMP_DIR=$(mktemp -d)
    cd "$TEMP_DIR"
    
    if ! curl -L -f -o "$ARCHIVE_NAME" "$DOWNLOAD_URL"; then
        error "Failed to download from $DOWNLOAD_URL"
        error "Please check if this platform is supported"
        rm -rf "$TEMP_DIR"
        exit 1
    fi
    
    success "Downloaded successfully"
}

# Verify checksum
verify_checksum() {
    info "Verifying checksum..."
    
    # Download checksums file
    CHECKSUMS_URL="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}/checksums.txt"
    if ! curl -L -f -s -o checksums.txt "$CHECKSUMS_URL"; then
        warning "Could not download checksums file, skipping verification"
        return
    fi
    
    # Extract expected checksum
    EXPECTED_CHECKSUM=$(grep "$ARCHIVE_NAME" checksums.txt | awk '{print $1}')
    if [ -z "$EXPECTED_CHECKSUM" ]; then
        warning "Checksum not found for $ARCHIVE_NAME, skipping verification"
        return
    fi
    
    # Calculate actual checksum
    if command -v sha256sum >/dev/null 2>&1; then
        ACTUAL_CHECKSUM=$(sha256sum "$ARCHIVE_NAME" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
        ACTUAL_CHECKSUM=$(shasum -a 256 "$ARCHIVE_NAME" | awk '{print $1}')
    else
        warning "No checksum tool available, skipping verification"
        return
    fi
    
    # Compare checksums
    if [ "$EXPECTED_CHECKSUM" != "$ACTUAL_CHECKSUM" ]; then
        error "Checksum verification failed!"
        error "Expected: $EXPECTED_CHECKSUM"
        error "Actual:   $ACTUAL_CHECKSUM"
        rm -rf "$TEMP_DIR"
        exit 1
    fi
    
    success "Checksum verified"
}

# Extract archive
extract_archive() {
    info "Extracting archive..."
    
    if [ "$OS" = "windows" ]; then
        unzip -q "$ARCHIVE_NAME"
    else
        tar -xzf "$ARCHIVE_NAME"
    fi
    
    # Find the extracted directory
    EXTRACT_DIR=$(find . -type d -name "${BINARY_NAME}-*" | head -n 1)
    if [ -z "$EXTRACT_DIR" ]; then
        # Archive might extract directly without a directory
        EXTRACT_DIR="."
    fi
    
    success "Extracted successfully"
}

# Install binary
install_binary() {
    info "Installing to $INSTALL_DIR..."
    
    # Check if we need sudo
    if [ -w "$INSTALL_DIR" ]; then
        SUDO=""
    else
        SUDO="sudo"
        warning "Installation requires sudo privileges"
    fi
    
    # Create install directory if it doesn't exist
    $SUDO mkdir -p "$INSTALL_DIR"
    
    # Find and install the binary
    BINARY_PATH="${EXTRACT_DIR}/${BINARY_NAME}"
    if [ "$OS" = "windows" ]; then
        BINARY_PATH="${BINARY_PATH}.exe"
    fi
    
    if [ ! -f "$BINARY_PATH" ]; then
        error "Binary not found at $BINARY_PATH"
        ls -la "$EXTRACT_DIR"
        rm -rf "$TEMP_DIR"
        exit 1
    fi
    
    # Copy binary to install directory
    $SUDO cp "$BINARY_PATH" "$INSTALL_DIR/${BINARY_NAME}"
    $SUDO chmod +x "$INSTALL_DIR/${BINARY_NAME}"
    
    success "Installed to $INSTALL_DIR/${BINARY_NAME}"
}

# Verify installation
verify_installation() {
    info "Verifying installation..."
    
    if ! command -v "$BINARY_NAME" >/dev/null 2>&1; then
        warning "Pocket is installed but not in your PATH"
        warning "Add $INSTALL_DIR to your PATH or run: $INSTALL_DIR/$BINARY_NAME"
    else
        INSTALLED_VERSION=$("$BINARY_NAME" version | grep -oE 'v?[0-9]+\.[0-9]+\.[0-9]+' | head -n 1)
        success "Pocket $INSTALLED_VERSION is installed and ready to use!"
    fi
}

# Cleanup
cleanup() {
    if [ -n "$TEMP_DIR" ] && [ -d "$TEMP_DIR" ]; then
        rm -rf "$TEMP_DIR"
    fi
}

# Main installation flow
main() {
    echo -e "${BLUE}🚀 Pocket Installer${NC}"
    echo "===================="
    echo
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --version)
                VERSION="$2"
                shift 2
                ;;
            --install-dir)
                INSTALL_DIR="$2"
                shift 2
                ;;
            --help)
                echo "Usage: $0 [options]"
                echo "Options:"
                echo "  --version VERSION     Install specific version (default: latest)"
                echo "  --install-dir DIR     Installation directory (default: /usr/local/bin)"
                echo "  --help               Show this help message"
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    # Set trap for cleanup
    trap cleanup EXIT
    
    # Detect system
    detect_os
    detect_arch
    info "Detected: ${OS}/${ARCH}"
    
    # Get version if not specified
    if [ -z "$VERSION" ]; then
        get_latest_version
    else
        info "Installing specified version: $VERSION"
    fi
    
    # Download and install
    construct_download_url
    download_binary
    verify_checksum
    extract_archive
    install_binary
    verify_installation
    
    echo
    echo -e "${GREEN}✨ Installation complete!${NC}"
    echo
    echo "Get started with:"
    echo "  pocket run hello.yaml"
    echo "  pocket --help"
    echo
    echo "Documentation: https://github.com/${GITHUB_REPO}/tree/main/docs"
}

# Run main function
main "$@"