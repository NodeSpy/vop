#!/usr/bin/env bash
#
# vop installer — detects OS/arch and installs the latest release.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/NodeSpy/vop/main/install.sh | bash
#
set -euo pipefail

REPO="NodeSpy/vop"
INSTALL_DIR="${VOP_INSTALL_DIR:-/usr/local/bin}"

# --- helpers ---

info() { printf '\033[0;34m>>>\033[0m %s\n' "$*"; }
error() { printf '\033[0;31m>>>\033[0m %s\n' "$*" >&2; }
ok() { printf '\033[0;32m>>>\033[0m %s\n' "$*"; }

need_cmd() {
	if ! command -v "$1" &>/dev/null; then
		error "Required command not found: $1"
		exit 1
	fi
}

# --- detect platform ---

detect_os() {
	local os
	os=$(uname -s | tr '[:upper:]' '[:lower:]')
	case "$os" in
	linux) echo "linux" ;;
	darwin) echo "darwin" ;;
	*)
		error "Unsupported OS: $os"
		exit 1
		;;
	esac
}

detect_arch() {
	local arch
	arch=$(uname -m)
	case "$arch" in
	x86_64 | amd64) echo "amd64" ;;
	aarch64 | arm64) echo "arm64" ;;
	*)
		error "Unsupported architecture: $arch"
		exit 1
		;;
	esac
}

# --- main ---

main() {
	need_cmd curl
	need_cmd uname

	local os arch binary_name download_url version
	os=$(detect_os)
	arch=$(detect_arch)
	binary_name="vop-${os}-${arch}"

	info "Detected platform: ${os}/${arch}"

	# Get latest release tag
	info "Fetching latest release..."
	version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" |
		grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')

	if [ -z "$version" ]; then
		error "Failed to determine latest version."
		error "Check https://github.com/${REPO}/releases for available versions."
		exit 1
	fi

	info "Latest version: ${version}"
	download_url="https://github.com/${REPO}/releases/download/${version}/${binary_name}"

	# Download
	local tmp
	tmp=$(mktemp)
	trap 'rm -f "$tmp"' EXIT

	info "Downloading ${binary_name}..."
	if ! curl -fsSL -o "$tmp" "$download_url"; then
		error "Download failed. Check that a release exists for ${os}/${arch}."
		error "URL: ${download_url}"
		exit 1
	fi

	chmod +x "$tmp"

	# Install
	if [ -w "$INSTALL_DIR" ]; then
		mv "$tmp" "${INSTALL_DIR}/vop"
	else
		info "Installing to ${INSTALL_DIR} (requires sudo)..."
		sudo mv "$tmp" "${INSTALL_DIR}/vop"
	fi

	ok "Installed vop ${version} to ${INSTALL_DIR}/vop"
	echo ""
	info "Run 'vop check' to verify your setup."
}

main "$@"
