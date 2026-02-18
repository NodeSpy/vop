# vop

AWS credential management via 1Password. Spawn shells and run commands with AWS credentials fetched directly from your 1Password vault.

## Features

- **Shell sessions** with AWS credentials injected as environment variables
- **Exec mode** to run single commands with credentials
- **MFA support** with automatic TOTP code retrieval from 1Password
- **Key rotation** via `vop rotate`
- **Dual backend** — each profile can independently use:
  - **CLI backend**: uses the `op` CLI binary (interactive auth, biometric)
  - **SDK backend**: uses the 1Password Go SDK with a service account token (no `op` binary needed)
- **No external dependencies** for SDK-backend profiles — the AWS SDK for Go v2 replaces the `aws` CLI, so the binary is fully self-contained
- **Credential files** in a secure per-user directory (`/run/user/$UID/vop/` on Linux, `$TMPDIR/vop-$UID/vop/` on macOS), cleaned up on exit
- **Backward compatible** with Vaulted — sets both `VOP_PROFILE` and `VAULTED_ENV`

## Install

### One-liner (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/NodeSpy/vop/main/install.sh | bash
```

The installer auto-detects available package managers and offers the best install method:

- **Homebrew** -- if `brew` is in your PATH, offers `brew install vop`
- **AUR** -- on Arch Linux with `yay` or `paru`, offers the `vop-bin` AUR package
- **Binary download** -- always available as a fallback, downloads to `/usr/local/bin`

When multiple methods are available, you'll be prompted to choose. To force a specific method or override the install directory:

```bash
# Force a specific method
VOP_METHOD=brew curl -fsSL https://raw.githubusercontent.com/NodeSpy/vop/main/install.sh | bash
VOP_METHOD=aur curl -fsSL https://raw.githubusercontent.com/NodeSpy/vop/main/install.sh | bash
VOP_METHOD=binary curl -fsSL https://raw.githubusercontent.com/NodeSpy/vop/main/install.sh | bash

# Custom install directory (binary method only)
VOP_INSTALL_DIR=~/.local/bin curl -fsSL https://raw.githubusercontent.com/NodeSpy/vop/main/install.sh | bash
```

### Homebrew (macOS and Linux)

```bash
brew tap NodeSpy/vop https://github.com/NodeSpy/vop
brew install vop
```

To upgrade:

```bash
brew upgrade vop
```

### Arch Linux (AUR)

With an AUR helper:

```bash
yay -S vop-bin
# or
paru -S vop-bin
```

### Nix

```bash
# Run directly
nix run github:NodeSpy/vop

# Install to profile
nix profile install github:NodeSpy/vop

# Or add to your flake inputs
```

A `flake.nix` is included in the repo. A dev shell is also available:

```bash
nix develop github:NodeSpy/vop
```

### Manual download

Download the binary for your platform from the [releases page](https://github.com/NodeSpy/vop/releases), make it executable, and place it in your PATH:

```bash
chmod +x vop-linux-amd64
sudo mv vop-linux-amd64 /usr/local/bin/vop
```

Available binaries:
| Platform | Binary |
|---|---|
| Linux x86_64 | `vop-linux-amd64` |
| Linux ARM64 | `vop-linux-arm64` |
| macOS x86_64 | `vop-darwin-amd64` |
| macOS ARM64 (Apple Silicon) | `vop-darwin-arm64` |

### Build from source

Requires Go 1.25+ and CGO (for the 1Password SDK):

```bash
git clone https://github.com/NodeSpy/vop.git
cd vop
go build -o vop .
sudo mv vop /usr/local/bin/
```

## Update

If installed via the install script or manual download:

```bash
vop update
```

This prompts for confirmation, then downloads and installs the latest release.

If installed via Homebrew, use `brew upgrade vop` instead.

## Uninstall

```bash
curl -fsSL https://raw.githubusercontent.com/NodeSpy/vop/main/install.sh | bash -s -- uninstall
```

This detects all vop installations on your system (Homebrew, AUR, standalone binary) and offers to remove them. If you have both a package-managed install and a standalone binary (e.g. from a previous one-liner install), it will warn about the conflict and default to removing only the standalone binary.

To uninstall a specific method manually:

```bash
# Homebrew
brew uninstall vop && brew untap NodeSpy/vop

# AUR
yay -R vop-bin

# Standalone binary
sudo rm /usr/local/bin/vop
```

## Quick start

```bash
# Check prerequisites
vop check

# Add a profile (CLI backend — uses op CLI)
vop add prod

# Add a profile (SDK backend — uses service account token)
vop add ci

# List profiles
vop ls

# Open a shell with AWS credentials
vop prod
# or equivalently:
vop shell prod

# Run a single command
vop exec prod aws s3 ls
# or with -- separator:
vop exec prod -- aws s3 ls

# Test credentials
vop test prod

# Rotate access keys
vop rotate prod

# Show profile configuration
vop show prod
```

## Prerequisites

- **CLI backend profiles**: [1Password CLI](https://developer.1password.com/docs/cli/get-started/) (`op`)
- **SDK backend profiles**: no external tools needed (service account token configured in profile)

Run `vop check` to verify your setup.

## Configuration

Profiles are stored in `~/.config/vop/profiles.json` (chmod 600). This file is intentionally excluded from version control since it may contain service account tokens.

### Profile fields

| Field | Required | Description |
|---|---|---|
| `op_account` | CLI only | 1Password account (e.g. `my.1password.com`) |
| `op_item` | Yes | 1Password item name (e.g. `AWS - prod`) |
| `op_vault` | SDK only | 1Password vault name (for `op://vault/item/field` refs) |
| `service_account_token` | SDK only | 1Password service account token |
| `iam_username` | No | IAM username (for key rotation) |
| `mfa_item` | No | Separate 1Password item for MFA TOTP |
| `description` | No | Profile description |
| `agent_policy` | No | Custom agent instructions (default: read-only) |

### Backend selection

Each profile independently chooses its backend:

- **No `service_account_token`** = CLI backend (uses `op` binary, interactive auth)
- **Has `service_account_token`** = SDK backend (pure Go, no external binaries)

### Setting up a 1Password service account

The SDK backend requires a [1Password service account](https://developer.1password.com/docs/service-accounts/) token. To create one:

1. Sign in to your 1Password account at [my.1password.com](https://my.1password.com)
2. Go to **Developer** in the sidebar (or **Settings > Developer** depending on your plan)
3. Under **Infrastructure Secrets**, click **Service Accounts**
4. Click **Create Service Account**
5. Give it a name (e.g. `vop`)
6. Grant it access to the vault(s) that contain your AWS credential items -- it needs **read** and **write** permissions (write is needed for key rotation and migration)
7. Copy the service account token (starts with `ops_`)

Then use it when adding or migrating a profile:

```bash
# When adding a new profile
vop add prod
# Answer "yes" to "Use a service account token?" and paste the token

# When migrating from Vaulted
vop migrate prod
# Answer "yes" to "Use a service account token?" and paste the token
```

The token is stored in `~/.config/vop/profiles.json` which is chmod 600 and should not be committed to version control.

## Commands

| Command | Description |
|---|---|
| `vop ls` | List configured profiles |
| `vop shell [profile]` | Open a shell with AWS credentials (picker if no arg) |
| `vop <profile>` | Shorthand for `vop shell <profile>` |
| `vop exec <profile> [--] <cmd>` | Run a command with credentials |
| `vop refresh [profile]` | Refresh credentials in an active shell |
| `vop cred-process <profile>` | Output credentials for AWS `credential_process` |
| `vop agent [profile]` | Show credential paths for AI agents / external tools |
| `vop add [profile]` | Add a new profile |
| `vop edit <profile>` | Edit an existing profile |
| `vop rm <profile>` | Remove a profile |
| `vop show <profile>` | Show profile configuration |
| `vop dump [profile]` | Dump raw credentials |
| `vop test <profile>` | Test credentials with STS |
| `vop cat` | Print profiles config (redacts tokens) |
| `vop rotate [profile]` | Rotate IAM access keys (picker if no arg) |
| `vop migrate [vault]` | Migrate from Vaulted |
| `vop check` | Check prerequisites and configuration |
| `vop version` | Print version information |
| `vop update` | Update to the latest version |

## Migrating from Vaulted

If you currently use [Vaulted](https://github.com/miquella/vaulted), vop can import your profiles:

```bash
# Migrate a single vault
vop migrate <vault-name>

# Interactive — lists all vaults and offers to migrate them
vop migrate
```

This unlocks the Vaulted vault, extracts the AWS credentials (access key, secret key, MFA serial, region), and then asks which 1Password backend to use:

- **Service account token (SDK)**: provide your token and vault name -- no `op` CLI needed
- **op CLI**: select your 1Password account and vault interactively

It then creates a new 1Password item containing those credentials and sets up a vop profile pointing to it. If the vault has MFA configured, you'll be prompted to link an existing 1Password TOTP item.

## License

MIT
