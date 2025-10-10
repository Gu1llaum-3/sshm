#!/bin/bash

INSTALL_DIR="/usr/local/bin"
EXECUTABLE_NAME=sshm
EXECUTABLE_PATH="$INSTALL_DIR/$EXECUTABLE_NAME"
USE_SUDO="false"
OS=""
ARCH=""
FORCE_INSTALL="${FORCE_INSTALL:-false}"
SSHM_VERSION="${SSHM_VERSION:-latest}"

RED='\033[0;31m'
PURPLE='\033[0;35m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

usage() {
    printf "${PURPLE}SSHM Installation Script${NC}\n\n"
    printf "Usage:\n"
    printf "  Default (latest stable):     ${GREEN}bash install.sh${NC}\n"
    printf "  Specific version:            ${GREEN}SSHM_VERSION=v1.8.0 bash install.sh${NC}\n"
    printf "  Beta/pre-release:            ${GREEN}SSHM_VERSION=v1.8.1-beta bash install.sh${NC}\n"
    printf "  Force install:               ${GREEN}FORCE_INSTALL=true bash install.sh${NC}\n"
    printf "  Custom install directory:    ${GREEN}INSTALL_DIR=/opt/bin bash install.sh${NC}\n\n"
    printf "Environment variables:\n"
    printf "  SSHM_VERSION    - Version to install (default: latest)\n"
    printf "  FORCE_INSTALL   - Skip confirmation prompts (default: false)\n"
    printf "  INSTALL_DIR     - Installation directory (default: /usr/local/bin)\n\n"
}

setSystem() {
    ARCH=$(uname -m)
    case $ARCH in
        i386|i686) ARCH="amd64" ;;
        x86_64) ARCH="amd64";;
        armv6*) ARCH="armv6" ;;
        armv7*) ARCH="armv7" ;;
        aarch64*) ARCH="arm64" ;;
        arm64) ARCH="arm64" ;;
    esac

    OS=$(echo `uname`|tr '[:upper:]' '[:lower:]')
    
    # Determine if we need sudo
    if [ "$OS" = "linux" ]; then
        USE_SUDO="true"
    fi
    if [ "$OS" = "darwin" ]; then
        USE_SUDO="true"
    fi
}

runAsRoot() {
    local CMD="$*"
    if [ "$USE_SUDO" = "true" ]; then
        printf "${PURPLE}We need sudo access to install SSHM to $INSTALL_DIR ${NC}\n"
        CMD="sudo $CMD"
    fi
    $CMD
}

getLatestVersion() {
    if [ "$SSHM_VERSION" = "latest" ]; then
        printf "${YELLOW}Fetching latest stable version...${NC}\n"
        LATEST_VERSION=$(curl -s https://api.github.com/repos/Gu1llaum-3/sshm/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
        if [ -z "$LATEST_VERSION" ]; then
            printf "${RED}Failed to fetch latest version${NC}\n"
            exit 1
        fi
    else
        printf "${YELLOW}Using specified version: $SSHM_VERSION${NC}\n"
        # Validate that the specified version exists
        RELEASE_CHECK=$(curl -s "https://api.github.com/repos/Gu1llaum-3/sshm/releases/tags/$SSHM_VERSION" | grep '"tag_name":')
        if [ -z "$RELEASE_CHECK" ]; then
            printf "${RED}Version $SSHM_VERSION not found. Available versions:${NC}\n"
            curl -s https://api.github.com/repos/Gu1llaum-3/sshm/releases | grep '"tag_name":' | head -10 | sed -E 's/.*"([^"]+)".*/  - \1/'
            exit 1
        fi
        LATEST_VERSION="$SSHM_VERSION"
    fi
    printf "${GREEN}Installing version: $LATEST_VERSION${NC}\n"
}

downloadBinary() {
    # Map OS names to match GoReleaser format
    local GORELEASER_OS="$OS"
    case $OS in
        "darwin") GORELEASER_OS="Darwin" ;;
        "linux") GORELEASER_OS="Linux" ;;
        "windows") GORELEASER_OS="Windows" ;;
    esac
    
    # Map architecture names to match GoReleaser format  
    local GORELEASER_ARCH="$ARCH"
    case $ARCH in
        "amd64") GORELEASER_ARCH="x86_64" ;;
        "arm64") GORELEASER_ARCH="arm64" ;;
        "386") GORELEASER_ARCH="i386" ;;
        "armv6") GORELEASER_ARCH="armv6" ;;
        "armv7") GORELEASER_ARCH="armv7" ;;
    esac
    
    # GoReleaser format: sshm_Linux_armv7.tar.gz
    GITHUB_FILE="sshm_${GORELEASER_OS}_${GORELEASER_ARCH}.tar.gz"
    GITHUB_URL="https://github.com/Gu1llaum-3/sshm/releases/download/$LATEST_VERSION/$GITHUB_FILE"
    
    printf "${YELLOW}Downloading $GITHUB_FILE...${NC}\n"
    curl -L "$GITHUB_URL" --progress-bar --output "sshm-tmp.tar.gz"
    
    if [ $? -ne 0 ]; then
        printf "${RED}Failed to download binary${NC}\n"
        exit 1
    fi
    
    # Extract the binary
    tar -xzf "sshm-tmp.tar.gz"
    if [ $? -ne 0 ]; then
        printf "${RED}Failed to extract binary${NC}\n"
        exit 1
    fi
    
    # GoReleaser extracts the binary as just "sshm", not with the platform suffix
    EXTRACTED_BINARY="./sshm"
    if [ ! -f "$EXTRACTED_BINARY" ]; then
        printf "${RED}Could not find extracted binary: $EXTRACTED_BINARY${NC}\n"
        exit 1
    fi
    
    mv "$EXTRACTED_BINARY" "sshm-tmp"
    rm -f "sshm-tmp.tar.gz"
}

install() {
    printf "${YELLOW}Installing SSHM...${NC}\n"
    
    # Backup old version if it exists to prevent interference during installation
    OLD_BACKUP=""
    if [ -f "$EXECUTABLE_PATH" ]; then
        OLD_BACKUP="$EXECUTABLE_PATH.backup.$$"
        runAsRoot mv "$EXECUTABLE_PATH" "$OLD_BACKUP"
    fi
    
    chmod +x "sshm-tmp"
    if [ $? -ne 0 ]; then
        printf "${RED}Failed to set permissions${NC}\n"
        # Restore backup if installation fails
        if [ -n "$OLD_BACKUP" ] && [ -f "$OLD_BACKUP" ]; then
            runAsRoot mv "$OLD_BACKUP" "$EXECUTABLE_PATH"
        fi
        exit 1
    fi

    runAsRoot mv "sshm-tmp" "$EXECUTABLE_PATH"
    if [ $? -ne 0 ]; then
        printf "${RED}Failed to install binary${NC}\n"
        # Restore backup if installation fails
        if [ -n "$OLD_BACKUP" ] && [ -f "$OLD_BACKUP" ]; then
            runAsRoot mv "$OLD_BACKUP" "$EXECUTABLE_PATH"
        fi
        exit 1
    fi
    
    # Clean up backup if installation succeeded
    if [ -n "$OLD_BACKUP" ] && [ -f "$OLD_BACKUP" ]; then
        runAsRoot rm -f "$OLD_BACKUP"
    fi
}

cleanup() {
    rm -f "sshm-tmp" "sshm-tmp.tar.gz" "sshm-${OS}-${ARCH}"
}

checkExisting() {
    if command -v sshm >/dev/null 2>&1; then
        CURRENT_VERSION=$(sshm --version 2>/dev/null | grep -o 'version.*' | cut -d' ' -f2 || echo "unknown")
        printf "${YELLOW}SSHM is already installed (version: $CURRENT_VERSION)${NC}\n"
        
        # Check if FORCE_INSTALL is set
        if [ "$FORCE_INSTALL" = "true" ]; then
            printf "${GREEN}Force install enabled, proceeding with installation...${NC}\n"
            return
        fi
        
        # Check if running via pipe (stdin is not a terminal)
        if [ ! -t 0 ]; then
            printf "${YELLOW}Running via pipe - automatically proceeding with installation...${NC}\n"
            printf "${YELLOW}Use 'FORCE_INSTALL=false bash -c \"\$(curl -sSL ...)\"' to disable auto-install${NC}\n"
            return
        fi
        
        printf "${YELLOW}Do you want to overwrite it? [y/N]: ${NC}"
        read -r response
        case "$response" in
            [yY][eE][sS]|[yY]) 
                printf "${GREEN}Proceeding with installation...${NC}\n"
                ;;
            *)
                printf "${GREEN}Installation cancelled.${NC}\n"
                exit 0
                ;;
        esac
    fi
}

main() {
    # Check for help argument
    if [ "$1" = "-h" ] || [ "$1" = "--help" ] || [ "$1" = "help" ]; then
        usage
        exit 0
    fi
    
    printf "${PURPLE}Installing SSHM - SSH Connection Manager${NC}\n\n"
    
    # Set up system detection
    setSystem
    printf "${GREEN}Detected system: $OS ($ARCH)${NC}\n"
    
    # Get and validate version FIRST (this can fail early)
    getLatestVersion
    
    # Check if already installed (this might prompt user)
    checkExisting
    
    # Download and install
    downloadBinary
    install
    cleanup
    
    printf "\n${GREEN}✅ SSHM was installed successfully to: ${NC}$EXECUTABLE_PATH\n"
    printf "${GREEN}You can now use 'sshm' command to manage your SSH connections!${NC}\n\n"
    
    # Show version
    printf "${YELLOW}Verifying installation...${NC}\n"
    if command -v sshm >/dev/null 2>&1; then
        # Use the full path to ensure we're using the newly installed version
        "$EXECUTABLE_PATH" --version 2>/dev/null || echo "Version check failed, but installation completed"
    else
        printf "${RED}Warning: 'sshm' command not found in PATH. You may need to restart your terminal or add $INSTALL_DIR to your PATH.${NC}\n"
    fi
}

# Trap to cleanup on exit
trap cleanup EXIT

main "$@"
