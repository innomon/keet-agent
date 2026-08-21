#!/usr/bin/env bash
# ==============================================================================
# Keet ADK Gateway - Multi-Platform Build Script
# 
# Supported Targets:
#   1. macOS Apple Silicon (darwin/arm64)
#   2. Debian Linux ARM64  (linux/arm64)
#   3. Ubuntu AMD64        (linux/amd64)
# ==============================================================================

set -euo pipefail

# ------------------------------------------------------------------------------
# Configuration & Defaults
# ------------------------------------------------------------------------------
APP_NAME="gateway"
CMD_PATH="./cmd/gateway"
OUTPUT_DIR="bin"
DIST_DIR="dist"
PACKAGE=false
CLEAN=false
VERBOSE=false

# ANSI Color codes
BOLD="\033[1m"
GREEN="\033[0;32m"
BLUE="\033[0;34m"
CYAN="\033[0;36m"
YELLOW="\033[1;33m"
RED="\033[0;31m"
NC="\033[0m" # No Color

# Git build metadata
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS="-s -w -X main.Version=${VERSION} -X main.GitCommit=${GIT_COMMIT} -X main.BuildDate=${BUILD_TIME}"

# ------------------------------------------------------------------------------
# Helper Functions
# ------------------------------------------------------------------------------
log_info() {
    echo -e "${BLUE}${BOLD}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}${BOLD}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}${BOLD}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}${BOLD}[ERROR]${NC} $1"
}

print_header() {
    echo -e "${CYAN}${BOLD}"
    echo "============================================================"
    echo "       Keet ADK Gateway - Cross-Platform Builder           "
    echo "============================================================"
    echo -e "${NC}"
    echo -e "  Version:    ${BOLD}${VERSION}${NC} (${GIT_COMMIT})"
    echo -e "  Build Date: ${BOLD}${BUILD_TIME}${NC}"
    echo -e "  Output Dir: ${BOLD}${OUTPUT_DIR}${NC}"
    echo ""
}

show_help() {
    cat << EOF
Usage: $(basename "$0") [TARGET|COMMAND] [OPTIONS]

Targets:
  all               Build for all supported platforms (default)
  macos-arm64       Build for macOS Apple Silicon (darwin/arm64)
  darwin-arm64      Alias for macos-arm64
  debian-arm64      Build for Debian Linux ARM64 (linux/arm64)
  linux-arm64       Alias for debian-arm64
  ubuntu-amd64      Build for Ubuntu AMD64 (linux/amd64)
  linux-amd64       Alias for ubuntu-amd64
  host              Build for current host OS and architecture

Commands:
  clean             Remove all built binaries and distribution archives
  package           Build and package (.tar.gz + SHA256 checksums) for all platforms

Options:
  -o, --output DIR  Specify binary output directory (default: bin)
  -p, --package     Create compressed tarballs and checksums
  -c, --clean       Clean output directory before building
  -v, --verbose     Enable verbose compiler output
  -h, --help        Display this help message

Examples:
  ./build.sh                     # Build all 3 target binaries
  ./build.sh macos-arm64         # Build macOS Apple Silicon binary only
  ./build.sh debian-arm64        # Build Debian ARM64 binary only
  ./build.sh ubuntu-amd64        # Build Ubuntu AMD64 binary only
  ./build.sh all --package       # Build all targets and create tar.gz archives
  ./build.sh clean               # Clean output directories

EOF
}

check_prerequisites() {
    if ! command -v go >/dev/null 2>&1; then
        log_error "Go compiler (go) is not installed or not in PATH."
        exit 1
    fi
}

clean_artifacts() {
    log_info "Cleaning build artifacts..."
    rm -rf "${OUTPUT_DIR}" "${DIST_DIR}"
    log_success "Cleaned '${OUTPUT_DIR}' and '${DIST_DIR}'."
}

# ------------------------------------------------------------------------------
# Build Function
# ------------------------------------------------------------------------------
build_target() {
    local target_name="$1"
    local goos="$2"
    local goarch="$3"
    local output_bin="${OUTPUT_DIR}/${APP_NAME}-${goos}-${goarch}"

    log_info "Building ${BOLD}${target_name}${NC} (${goos}/${goarch}) -> ${output_bin}..."

    mkdir -p "${OUTPUT_DIR}"

    local verbose_flag=""
    if [ "${VERBOSE}" = true ]; then
        verbose_flag="-v"
    fi

    # Compile pure Go static binary (CGO_ENABLED=0 prevents glibc/system dynamic link issues)
    CGO_ENABLED=0 GOOS="${goos}" GOARCH="${goarch}" go build \
        ${verbose_flag} \
        -trimpath \
        -ldflags "${LDFLAGS}" \
        -o "${output_bin}" \
        "${CMD_PATH}"

    local bin_size
    bin_size=$(ls -lh "${output_bin}" | awk '{print $5}')
    log_success "Built ${BOLD}${target_name}${NC} (${bin_size})"

    # If packaging requested, create .tar.gz bundle with checksum
    if [ "${PACKAGE}" = true ]; then
        package_target "${target_name}" "${goos}" "${goarch}" "${output_bin}"
    fi
}

package_target() {
    local target_name="$1"
    local goos="$2"
    local goarch="$3"
    local bin_path="$4"
    local archive_name="${APP_NAME}-${VERSION}-${goos}-${goarch}.tar.gz"

    mkdir -p "${DIST_DIR}"

    log_info "Packaging ${archive_name}..."

    # Create temporary packaging directory
    local pkg_tmp
    pkg_tmp=$(mktemp -d 2>/dev/null || mktemp -d -t 'keet_pkg')
    
    # Copy binary as standard 'gateway' name inside archive
    cp "${bin_path}" "${pkg_tmp}/${APP_NAME}"
    chmod +x "${pkg_tmp}/${APP_NAME}"

    # Include documentation / config if available
    [ -f "README.md" ] && cp "README.md" "${pkg_tmp}/"
    [ -f "config.example.yaml" ] && cp "config.example.yaml" "${pkg_tmp}/config.yaml"

    # Create tar.gz archive
    tar -czf "${DIST_DIR}/${archive_name}" -C "${pkg_tmp}" .
    rm -rf "${pkg_tmp}"

    # Generate SHA-256 checksum
    (
        cd "${DIST_DIR}"
        if command -v sha256sum >/dev/null 2>&1; then
            sha256sum "${archive_name}" >> SHA256SUMS
        else
            shasum -a 256 "${archive_name}" >> SHA256SUMS
        fi
    )

    log_success "Created archive ${BOLD}${DIST_DIR}/${archive_name}${NC}"
}

# ------------------------------------------------------------------------------
# Argument Parsing
# ------------------------------------------------------------------------------
TARGETS=()

while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--help|help)
            show_help
            exit 0
            ;;
        -c|--clean)
            CLEAN=true
            shift
            ;;
        clean)
            clean_artifacts
            exit 0
            ;;
        -p|--package)
            PACKAGE=true
            shift
            ;;
        package)
            PACKAGE=true
            TARGETS+=("all")
            shift
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -o|--output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        macos-arm64|darwin-arm64|macos|apple-silicon)
            TARGETS+=("macos-arm64")
            shift
            ;;
        debian-arm64|linux-arm64|debian)
            TARGETS+=("debian-arm64")
            shift
            ;;
        ubuntu-amd64|linux-amd64|ubuntu)
            TARGETS+=("ubuntu-amd64")
            shift
            ;;
        host|native|current)
            TARGETS+=("host")
            shift
            ;;
        all)
            TARGETS+=("all")
            shift
            ;;
        *)
            log_error "Unknown argument: $1"
            echo "Run '$(basename "$0") --help' for usage."
            exit 1
            ;;
    esac
done

# Default to "all" if no specific target is provided
if [ ${#TARGETS[@]} -eq 0 ]; then
    TARGETS=("all")
fi

# ------------------------------------------------------------------------------
# Execution
# ------------------------------------------------------------------------------
check_prerequisites
print_header

if [ "${CLEAN}" = true ]; then
    clean_artifacts
fi

# If packaging is requested, reset checksums file
if [ "${PACKAGE}" = true ]; then
    mkdir -p "${DIST_DIR}"
    rm -f "${DIST_DIR}/SHA256SUMS"
fi

START_TIME=$(date +%s)

for target in "${TARGETS[@]}"; do
    case "${target}" in
        all)
            build_target "macOS Apple Silicon" "darwin" "arm64"
            build_target "Debian Linux ARM64"  "linux"  "arm64"
            build_target "Ubuntu Linux AMD64"  "linux"  "amd64"
            ;;
        macos-arm64)
            build_target "macOS Apple Silicon" "darwin" "arm64"
            ;;
        debian-arm64)
            build_target "Debian Linux ARM64" "linux" "arm64"
            ;;
        ubuntu-amd64)
            build_target "Ubuntu Linux AMD64" "linux" "amd64"
            ;;
        host)
            HOST_OS=$(go env GOOS)
            HOST_ARCH=$(go env GOARCH)
            build_target "Host Native (${HOST_OS}/${HOST_ARCH})" "${HOST_OS}" "${HOST_ARCH}"
            ;;
    esac
done

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

echo ""
echo -e "${CYAN}------------------------------------------------------------${NC}"
log_success "Build completed in ${BOLD}${DURATION}s${NC}!"
log_info "Binaries are located in: ${BOLD}${OUTPUT_DIR}/${NC}"

if [ "${PACKAGE}" = true ]; then
    log_info "Distribution packages and SHA256SUMS located in: ${BOLD}${DIST_DIR}/${NC}"
fi
echo -e "${CYAN}------------------------------------------------------------${NC}"
