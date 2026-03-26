#!/usr/bin/env bash
set -e

REPO="sthbryan/ftm"
BINARY_NAME="ftm"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
TAG="latest"

detect_os() {
    case "$(uname -s)" in
        Linux*)     echo "linux" ;;
        Darwin*)    echo "macos" ;;
        CYGWIN*|MINGW*|MSYS*) echo "windows" ;;
        *)          echo "unsupported" ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)   echo "x64" ;;
        aarch64|arm64)  echo "arm64" ;;
        *)              echo "x64" ;;
    esac
}

get_extension() {
    if [ "$OS" = "windows" ]; then
        echo ".exe"
    else
        echo ""
    fi
}

get_filename() {
    local arch=$1
    echo "${BINARY_NAME}-${OS}-${arch}${EXT}"
}

get_download_url() {
    local arch=$1
    local filename=$(get_filename $arch)
    echo "https://github.com/${REPO}/releases/${TAG}/download/${filename}"
}

check_installed() {
    if command -v $BINARY_NAME &> /dev/null; then
        echo "$BINARY_NAME is already installed: $(which $BINARY_NAME)"
        echo "Version: $($BINARY_NAME --version 2>/dev/null || echo 'unknown')"
        return 0
    fi
    return 1
}

install() {
    local arch=$1
    local url=$(get_download_url $arch)
    local filename=$(get_filename $arch)
    local tmpfile=$(mktemp)

    echo "Downloading $BINARY_NAME for $OS ($arch)..."
    echo "URL: $url"

    curl -fSL "$url" -o "$tmpfile"

    echo "Making executable..."
    chmod +x "$tmpfile"

    mkdir -p "$INSTALL_DIR"

    echo "Installing to $INSTALL_DIR/$BINARY_NAME..."
    mv "$tmpfile" "$INSTALL_DIR/$BINARY_NAME"

    if [ "$OS" = "macos" ]; then
        xattr -d com.apple.quarantine "$INSTALL_DIR/$BINARY_NAME" 2>/dev/null || true
    fi

    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        local shell_config=""
        local shell_name="${SHELL##*/}"  # e.g., zsh, bash, fish
        
        case "$shell_name" in
            zsh)  shell_config="$HOME/.zshrc" ;;
            bash) shell_config="$HOME/.bashrc" ;;
            fish) shell_config="$HOME/.config/fish/config.fish" ;;
            *)    shell_config="$HOME/.bashrc" ;;
        esac
        
        if [ "$shell_name" = "fish" ]; then
            mkdir -p "$(dirname "$shell_config")"
        fi
        
        local export_line="export PATH=\"\$PATH:$INSTALL_DIR\""
        
        if ! grep -qF "$INSTALL_DIR" "$shell_config" 2>/dev/null; then
            echo "" >> "$shell_config"
            echo "# Added by $BINARY_NAME installer" >> "$shell_config"
            echo "$export_line" >> "$shell_config"
            echo ""
            echo "✓ Added $INSTALL_DIR to PATH in $shell_config"
            echo "  Restart your terminal or run: source $shell_config"
        else
            echo ""
            echo "✓ $INSTALL_DIR already in PATH"
        fi
    fi

    echo "✓ Installed $BINARY_NAME $OS-$arch"
}

main() {
    echo "╔══════════════════════════════════════╗"
    echo "║   Foundry Tunnel Manager Installer   ║"
    echo "╚══════════════════════════════════════╝"
    echo ""

    if ! check_installed; then
        OS=$(detect_os)
        ARCH=$(detect_arch)
        EXT=$(get_extension)

        if [ "$OS" = "unsupported" ]; then
            echo "Error: Unsupported operating system"
            exit 1
        fi

        echo "Detected: $OS ($ARCH)"
        echo ""

        install "$ARCH"
    fi

    echo ""
    echo "Run with: $BINARY_NAME"
}

main "$@"
