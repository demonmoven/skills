#!/bin/bash
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Logging functions
log_info() { echo -e "${BLUE}[INFO]${NC} $1" >&2; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1" >&2; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1" >&2; }
log_error() { echo -e "${RED}[ERROR]${NC} $1" >&2; }

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Darwin*) echo "darwin" ;;
        Linux*) echo "linux" ;;
        *) echo "unknown" ;;
    esac
}

# Detect architecture
detect_arch() {
    case $(uname -m) in
        x86_64) echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *) uname -m ;;
    esac
}

# Add cli path to PATH
add_to_path() {
    local path="$1"
    local shell_type
    local profile_file
    
    # Detect current shell
    if [ -n "$ZSH_VERSION" ]; then
        shell_type="zsh"
    elif [ -n "$BASH_VERSION" ]; then
        shell_type="bash"
    elif [ -n "$FISH_VERSION" ]; then
        shell_type="fish"
    else
        shell_type="$(basename "$SHELL" 2>/dev/null || echo "bash")"
    fi
    
    # Determine profile file
    case "$shell_type" in
        zsh)
            profile_file="$HOME/.zshrc"
            ;;
        fish)
            profile_file="$HOME/.config/fish/config.fish"
            ;;
        bash|*)
            if [ -f "$HOME/.bashrc" ]; then
                profile_file="$HOME/.bashrc"
            elif [ -f "$HOME/.bash_profile" ]; then
                profile_file="$HOME/.bash_profile"
            else
                profile_file="$HOME/.bashrc"
            fi
            ;;
    esac
    
    # Add to current session PATH
    if [[ ! "$PATH" =~ (^|:)"${path}"(:|$) ]]; then
        export PATH="${path}:${PATH}"
        log_info "Added ${path} to current session PATH"
    fi
    
    # Check if path already exists in profile file
    path_exists_in_profile() {
        local file="$1"
        local check_path="$2"
        if [ ! -f "$file" ]; then
            return 1
        fi
        if [ "$shell_type" = "fish" ]; then
            grep -v "^\s*#" "$file" 2>/dev/null | grep -q "fish_add_path.*\"${check_path}\"" || \
            grep -v "^\s*#" "$file" 2>/dev/null | grep -q "fish_add_path.*'${check_path}'" || \
            grep -v "^\s*#" "$file" 2>/dev/null | grep -q "fish_add_path.*${check_path}\b"
        else
            grep -v "^\s*#" "$file" 2>/dev/null | grep -q "export.*PATH.*\"${check_path}\"" || \
            grep -v "^\s*#" "$file" 2>/dev/null | grep -q "export.*PATH.*'${check_path}'" || \
            grep -v "^\s*#" "$file" 2>/dev/null | grep -q "export.*PATH.*${check_path}\b"
        fi
    }
    
    # Add to profile file if not already present
    if ! path_exists_in_profile "$profile_file" "$path"; then
        local profile_dir
        profile_dir="$(dirname "$profile_file")"
        if [ -f "$profile_file" ] && [ ! -w "$profile_file" ]; then
            log_warning "Skip updating ${profile_file} (permission denied). Please add ${path} to PATH manually."
            return 0
        fi
        if [ ! -f "$profile_file" ] && [ ! -w "$profile_dir" ]; then
            log_warning "Skip creating ${profile_file} (permission denied). Please add ${path} to PATH manually."
            return 0
        fi

        if [ "$shell_type" = "fish" ]; then
            echo "fish_add_path \"${path}\"" >> "$profile_file"
        else
            local export_line="export PATH=\"${path}:\$PATH\""
            echo "" >> "$profile_file"
            echo "$export_line" >> "$profile_file"
        fi
        log_success "Added ${path} to ${profile_file}"
    fi
}

# Main
main() {
    INSTALL_DIR="${HOME}/.local/share/bits-env"
    mkdir -p "$INSTALL_DIR"

    local target_in_scripts
    target_in_scripts="${INSTALL_DIR}/bits_env_cli"

    log_info "Installed:"
    log_info "  ${target_in_scripts}"
    
    # Add script directory to PATH
    add_to_path "$INSTALL_DIR"

    if ! command_exists "curl"; then
        log_error "curl is required but not installed."
        exit 1
    fi

    local os="$(detect_os)"
    local arch="$(detect_arch)"

    if [ "$os" = "unknown" ]; then
        log_error "Unsupported operating system: $(uname -s)"
        exit 1
    fi

    log_info "Detected platform: ${os}_${arch}"

    local filename="bits_env_cli_${os}_${arch}"

    local download_url
    if [ -n "${BYTE_ENV_CLI_URL:-}" ]; then
        download_url="${BYTE_ENV_CLI_URL}"
    else
        local base_url="${BYTE_ENV_CLI_BASE_URL:-https://tosv.byted.org/obj/boe-data-build/env_skills}"
        base_url="${base_url%/}"

        local last_segment="${base_url##*/}"

        local resolved_base_url
        if [[ "$last_segment" =~ ^[0-9]+(\.[0-9]+){2,3}$ ]]; then
            resolved_base_url="$base_url"
        else
            local version="$(curl -fsSL "${base_url}/version.txt" 2>/dev/null | tr -d '\r' | head -n 1 | xargs || true)"
            if [ -n "$version" ]; then
                resolved_base_url="${base_url}/${version}"
            else
                resolved_base_url=""
            fi
        fi

        if [ -n "$resolved_base_url" ]; then
            download_url="${resolved_base_url}/${filename}"
        else
            download_url=""
        fi
    fi

    if [ -n "$download_url" ]; then
        log_info "Downloading: ${download_url}"
    else
        log_warning "Failed to resolve remote download URL, fallback to bundled binary"
    fi

    local tmp_file="$(mktemp -t bits_env_cli.XXXXXX)"
    if [ -n "$download_url" ] && curl -fL -o "$tmp_file" "$download_url" 2>/dev/null; then
        :
    else
        if [ -n "$download_url" ]; then
            log_warning "Download failed, fallback to bundled binary"
        fi
        local source_path="${script_dir}/bits_env_${os}_${arch}"
        if [ ! -f "$source_path" ]; then
            log_error "No bundled binary for platform: ${os}_${arch}"
            log_info "Available binaries in ${script_dir}:"
            ls -1 "${script_dir}"/bits_env_* 2>/dev/null || true
            rm -f "$tmp_file" 2>/dev/null || true
            exit 1
        fi
        cp -f "$source_path" "$tmp_file"
    fi

    if command_exists "install"; then
        install -m 755 "$tmp_file" "$target_in_scripts"
    else
        cp -f "$tmp_file" "$target_in_scripts"
        chmod 755 "$target_in_scripts"
    fi
    rm -f "$tmp_file" 2>/dev/null || true

    if [ "$os" = "darwin" ] && command_exists "xattr"; then
        xattr -rd com.apple.quarantine "$target_in_scripts" 2>/dev/null || true
    fi

    log_success "Installed:"
    log_success "  ${target_in_scripts}"
}

main "$@"
