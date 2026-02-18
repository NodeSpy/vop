#!/usr/bin/env bash
#
# vop installer -- installs or updates vop using the best method
# available. Auto-detects existing installations and upgrades them
# via the original package manager.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/NodeSpy/vop/main/install.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/NodeSpy/vop/main/install.sh | bash -s -- uninstall
#
# Environment variables:
#   VOP_INSTALL_DIR   Override binary install directory (default: /usr/local/bin)
#   VOP_METHOD        Force install method: brew, aur, binary
#
set -euo pipefail

REPO="NodeSpy/vop"
INSTALL_DIR="${VOP_INSTALL_DIR:-/usr/local/bin}"
_TMP="" # Global so trap can clean up.

# --- helpers ---

info() { printf '\033[0;34m>>>\033[0m %s\n' "$*"; }
warn() { printf '\033[0;33m>>>\033[0m %s\n' "$*"; }
error() { printf '\033[0;31m>>>\033[0m %s\n' "$*" >&2; }
ok() { printf '\033[0;32m>>>\033[0m %s\n' "$*"; }

has_cmd() { command -v "$1" &>/dev/null; }

need_cmd() {
	if ! has_cmd "$1"; then
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

# --- detect existing installation ---

is_arch_linux() {
	[ -f /etc/os-release ] && grep -q '^ID=arch' /etc/os-release 2>/dev/null
}

detect_aur_helper() {
	for helper in yay paru; do
		if has_cmd "$helper"; then
			echo "$helper"
			return
		fi
	done
	echo ""
}

# Returns the method that originally installed vop, or empty string.
detect_existing() {
	if ! has_cmd vop; then
		echo ""
		return
	fi

	# Check Homebrew first.
	if has_cmd brew && brew list vop &>/dev/null 2>&1; then
		echo "brew"
		return
	fi

	# Check AUR (pacman owns the file).
	if has_cmd pacman && pacman -Qo "$(command -v vop)" &>/dev/null 2>&1; then
		echo "aur"
		return
	fi

	# Installed but not via a package manager -- treat as binary.
	echo "binary"
}

# --- detect available install methods ---

detect_methods() {
	local methods=""

	if has_cmd brew; then
		methods="brew"
	fi

	if is_arch_linux; then
		local aur_helper
		aur_helper=$(detect_aur_helper)
		if [ -n "$aur_helper" ]; then
			methods="${methods:+$methods }aur"
		fi
	fi

	# Binary download is always available if curl exists.
	methods="${methods:+$methods }binary"

	echo "$methods"
}

method_label() {
	case "$1" in
	brew) echo "Homebrew (brew install vop)" ;;
	aur) echo "AUR ($(detect_aur_helper) -S vop-bin)" ;;
	binary) echo "Binary download to ${INSTALL_DIR}" ;;
	esac
}

# --- install/update methods ---

install_brew() {
	info "Installing via Homebrew..."

	if ! brew tap NodeSpy/vop https://github.com/NodeSpy/vop 2>/dev/null; then
		true
	fi

	brew install vop

	ok "Installed vop via Homebrew."
	echo ""
	info "Upgrade later with: brew upgrade vop"
}

update_brew() {
	info "Updating vop via Homebrew..."

	if ! brew tap NodeSpy/vop https://github.com/NodeSpy/vop 2>/dev/null; then
		true
	fi

	brew upgrade vop 2>&1 || {
		# "already installed" is not an error
		info "Already up to date."
	}

	ok "vop is up to date (Homebrew)."
}

install_aur() {
	local helper
	helper=$(detect_aur_helper)

	if [ -z "$helper" ]; then
		error "No AUR helper found (yay or paru required)."
		exit 1
	fi

	info "Installing via AUR using ${helper}..."
	"$helper" -S --needed vop-bin

	ok "Installed vop via AUR."
	echo ""
	info "Upgrade later with: ${helper} -Syu vop-bin"
}

update_aur() {
	local helper
	helper=$(detect_aur_helper)

	if [ -z "$helper" ]; then
		error "No AUR helper found (yay or paru required)."
		exit 1
	fi

	info "Updating vop via AUR using ${helper}..."
	"$helper" -S --needed vop-bin

	ok "vop is up to date (AUR)."
}

install_binary() {
	need_cmd curl

	local os arch binary_name download_url version
	os=$(detect_os)
	arch=$(detect_arch)
	binary_name="vop-${os}-${arch}"

	info "Detected platform: ${os}/${arch}"

	# Get latest release tag.
	info "Fetching latest release..."
	local api_response
	api_response=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>&1) || {
		error "Failed to fetch release info from GitHub."
		error "This may be due to API rate limiting. Try again later or use:"
		error "  brew install vop  (if Homebrew is available)"
		error "  Visit: https://github.com/${REPO}/releases"
		exit 1
	}

	# Parse version -- try jq first, fall back to grep.
	if has_cmd jq; then
		version=$(echo "$api_response" | jq -r '.tag_name // empty')
	else
		version=$(echo "$api_response" | grep -o '"tag_name": *"[^"]*"' | head -1 | grep -o '"v[^"]*"' | tr -d '"')
	fi

	if [ -z "${version:-}" ]; then
		error "Failed to determine latest version."
		error "Check https://github.com/${REPO}/releases for available versions."
		if echo "$api_response" | grep -qi "rate limit"; then
			error "GitHub API rate limit exceeded. Try again later."
		fi
		exit 1
	fi

	# If already installed, check if we're already up to date.
	if has_cmd vop; then
		local current
		current=$(vop version 2>/dev/null | awk '{print $1}') || true
		if [ "${current:-}" = "$version" ]; then
			ok "vop is already up to date (${version})."
			return
		fi
		info "Updating ${current:-unknown} -> ${version}"
	fi

	info "Latest version: ${version}"
	download_url="https://github.com/${REPO}/releases/download/${version}/${binary_name}"

	# Download.
	_TMP=$(mktemp)
	trap 'rm -f "$_TMP"' EXIT

	info "Downloading ${binary_name}..."
	if ! curl -fsSL -o "$_TMP" "$download_url"; then
		error "Download failed."
		error "URL: ${download_url}"
		error ""
		error "Possible causes:"
		error "  - No release binary for ${os}/${arch}"
		error "  - Network issue or GitHub outage"
		error ""
		error "Try: https://github.com/${REPO}/releases"
		exit 1
	fi

	chmod +x "$_TMP"

	# Verify it's actually an executable, not an HTML error page.
	if file "$_TMP" 2>/dev/null | grep -qi "text\|html"; then
		error "Downloaded file is not a valid binary (got HTML/text instead)."
		error "The release asset may not exist for ${os}/${arch}."
		error "URL: ${download_url}"
		exit 1
	fi

	# Install.
	mkdir -p "$INSTALL_DIR" 2>/dev/null || true
	if [ -w "$INSTALL_DIR" ]; then
		mv "$_TMP" "${INSTALL_DIR}/vop"
	else
		info "Installing to ${INSTALL_DIR} (requires sudo)..."
		sudo mkdir -p "$INSTALL_DIR"
		sudo mv "$_TMP" "${INSTALL_DIR}/vop"
	fi

	ok "Installed vop ${version} to ${INSTALL_DIR}/vop"
	echo ""
	info "Upgrade later with: vop update"
}

# --- uninstall ---

# Find all vop binaries on the system, returning "path:method" lines.
find_all_installations() {
	local found=()

	# Check Homebrew.
	if has_cmd brew && brew list vop &>/dev/null 2>&1; then
		local brew_bin
		brew_bin=$(brew --prefix)/bin/vop
		if [ -f "$brew_bin" ]; then
			found+=("${brew_bin}:brew")
		fi
	fi

	# Check AUR / pacman.
	if has_cmd pacman; then
		local pac_bin
		pac_bin=$(pacman -Ql vop-bin 2>/dev/null | awk '/\/usr\/bin\/vop$/{print $2}') || true
		if [ -n "$pac_bin" ] && [ -f "$pac_bin" ]; then
			found+=("${pac_bin}:aur")
		fi
	fi

	# Check common binary install locations.
	local dir
	for dir in /usr/local/bin /usr/bin "$HOME/.local/bin" "$HOME/bin" "${INSTALL_DIR}"; do
		local bin="${dir}/vop"
		[ -f "$bin" ] || continue

		# Skip if already accounted for by brew or pacman above.
		local dominated=false
		local f
		for f in "${found[@]:-}"; do
			if [ "${f%%:*}" = "$bin" ]; then
				dominated=true
				break
			fi
		done
		$dominated && continue

		found+=("${bin}:binary")
	done

	local f
	for f in "${found[@]:-}"; do
		[ -n "$f" ] && echo "$f"
	done
}

remove_binary() {
	local bin="$1"
	if [ ! -f "$bin" ]; then
		return
	fi
	if [ -w "$bin" ] || [ -w "$(dirname "$bin")" ]; then
		rm -f "$bin"
	else
		info "Removing ${bin} (requires sudo)..."
		sudo rm -f "$bin"
	fi
}

cmd_uninstall() {
	local installs
	installs=$(find_all_installations)

	if [ -z "$installs" ]; then
		info "vop is not installed."
		return
	fi

	echo ""
	info "Found the following vop installation(s):"
	echo ""

	local count=0
	while IFS= read -r line; do
		local path="${line%%:*}"
		local method="${line##*:}"
		count=$((count + 1))
		local ver
		ver=$("$path" version 2>/dev/null | awk '{print $1}') || ver="unknown"
		case "$method" in
		brew) printf '  %d) %s (%s, Homebrew)\n' "$count" "$path" "$ver" ;;
		aur) printf '  %d) %s (%s, AUR/pacman)\n' "$count" "$path" "$ver" ;;
		binary) printf '  %d) %s (%s, standalone binary)\n' "$count" "$path" "$ver" ;;
		esac
	done <<<"$installs"
	echo ""

	# If there's a mix, help the user understand the situation.
	local has_pkg=false has_binary=false
	while IFS= read -r line; do
		local method="${line##*:}"
		case "$method" in
		brew | aur) has_pkg=true ;;
		binary) has_binary=true ;;
		esac
	done <<<"$installs"

	if $has_pkg && $has_binary; then
		warn "You have both a package-managed and a standalone binary install."
		warn "The standalone binary may shadow the package-managed one."
		echo ""
	fi

	# Interactive: ask what to remove.
	if [ -t 0 ] || [ -e /dev/tty ]; then
		printf '\033[1mWhat would you like to remove?\033[0m\n' >/dev/tty
		echo "" >/dev/tty
		printf '  a) All installations\n' >/dev/tty
		printf '  b) Only standalone binaries (keep package-managed)\n' >/dev/tty
		printf '  c) Cancel\n' >/dev/tty
		echo "" >/dev/tty

		local choice
		while true; do
			printf 'Choose [a/b/c] (default: b): ' >/dev/tty
			read -r choice </dev/tty || choice=""
			[ -z "$choice" ] && choice="b"

			case "$choice" in
			a | A) break ;;
			b | B) break ;;
			c | C)
				info "Cancelled."
				return
				;;
			*)
				warn "Invalid choice." >/dev/tty
				;;
			esac
		done

		echo ""
		while IFS= read -r line; do
			local path="${line%%:*}"
			local method="${line##*:}"

			case "$choice" in
			b | B)
				# Only remove standalone binaries.
				if [ "$method" = "binary" ]; then
					info "Removing ${path}..."
					remove_binary "$path"
					ok "Removed ${path}"
				else
					info "Keeping ${path} (managed by ${method})"
				fi
				;;
			a | A)
				case "$method" in
				brew)
					info "Uninstalling via Homebrew..."
					brew uninstall vop 2>/dev/null || true
					brew untap NodeSpy/vop 2>/dev/null || true
					ok "Uninstalled vop from Homebrew."
					;;
				aur)
					local helper
					helper=$(detect_aur_helper)
					if [ -n "$helper" ]; then
						info "Uninstalling via ${helper}..."
						"$helper" -R vop-bin
					else
						info "Uninstalling via pacman..."
						sudo pacman -R vop-bin
					fi
					ok "Uninstalled vop-bin from AUR."
					;;
				binary)
					info "Removing ${path}..."
					remove_binary "$path"
					ok "Removed ${path}"
					;;
				esac
				;;
			esac
		done <<<"$installs"
	else
		# Non-interactive: only remove standalone binaries (safe default).
		info "Non-interactive mode: removing standalone binaries only."
		while IFS= read -r line; do
			local path="${line%%:*}"
			local method="${line##*:}"
			if [ "$method" = "binary" ]; then
				info "Removing ${path}..."
				remove_binary "$path"
				ok "Removed ${path}"
			else
				info "Keeping ${path} (managed by ${method})"
			fi
		done <<<"$installs"
	fi

	echo ""
	if has_cmd vop; then
		ok "vop is still available at $(command -v vop)"
	else
		ok "vop has been uninstalled."
	fi
}

# --- user prompt ---

pick_method() {
	local methods=("$@")
	local count=${#methods[@]}

	if [ "$count" -eq 1 ]; then
		echo "${methods[0]}"
		return
	fi

	echo "" >/dev/tty
	printf '\033[1mAvailable install methods:\033[0m\n' >/dev/tty
	echo "" >/dev/tty
	for i in "${!methods[@]}"; do
		local n=$((i + 1))
		local label
		label=$(method_label "${methods[$i]}")
		if [ "$i" -eq 0 ]; then
			printf '  \033[1;32m%d) %s (recommended)\033[0m\n' "$n" "$label" >/dev/tty
		else
			printf '  %d) %s\n' "$n" "$label" >/dev/tty
		fi
	done
	echo "" >/dev/tty

	local choice
	while true; do
		printf 'Choose [1-%d] (default: 1): ' "$count" >/dev/tty
		read -r choice </dev/tty || choice=""

		if [ -z "$choice" ]; then
			choice=1
		fi

		if [[ "$choice" =~ ^[0-9]+$ ]] && [ "$choice" -ge 1 ] && [ "$choice" -le "$count" ]; then
			echo "${methods[$((choice - 1))]}"
			return
		fi

		warn "Invalid choice. Enter a number between 1 and ${count}." >/dev/tty
	done
}

# --- main ---

main() {
	# Handle subcommands.
	case "${1:-}" in
	uninstall)
		cmd_uninstall
		return
		;;
	esac

	need_cmd uname

	local os
	os=$(detect_os)
	info "Detected OS: ${os}"

	# Detect available methods and any existing installation.
	local methods_str method_array
	methods_str=$(detect_methods)
	read -ra method_array <<<"$methods_str"

	local existing
	existing=$(detect_existing)

	# Allow forcing a method via env var (overrides existing detection).
	local method="${VOP_METHOD:-}"

	if [ -n "$method" ]; then
		local valid=false
		for m in "${method_array[@]}"; do
			if [ "$m" = "$method" ]; then
				valid=true
				break
			fi
		done
		if [ "$valid" = false ]; then
			error "Forced method '${method}' is not available on this system."
			error "Available: ${methods_str}"
			exit 1
		fi
		info "Using forced install method: ${method}"
	elif [ -n "$existing" ]; then
		# Already installed -- update via the original method.
		local current_version
		current_version=$(vop version 2>/dev/null | awk '{print $1}') || current_version="unknown"
		info "vop ${current_version} is already installed (via ${existing})."

		case "$existing" in
		brew) update_brew ;;
		aur) update_aur ;;
		binary) install_binary ;; # install_binary handles the up-to-date check
		esac

		info "Run 'vop check' to verify your setup."
		return
	elif [ "${#method_array[@]}" -gt 1 ]; then
		if [ -t 0 ] || [ -e /dev/tty ]; then
			method=$(pick_method "${method_array[@]}")
			echo "" >/dev/tty 2>/dev/null || true
		else
			method="${method_array[0]}"
			info "Non-interactive mode: using ${method}"
		fi
	else
		method="${method_array[0]}"
	fi

	case "$method" in
	brew) install_brew ;;
	aur) install_aur ;;
	binary) install_binary ;;
	*)
		error "Unknown install method: $method"
		exit 1
		;;
	esac

	info "Run 'vop check' to verify your setup."
}

main "$@"
