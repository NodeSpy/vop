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
- **Credential files on tmpfs** at `/run/user/$UID/vop/` for external tool access, cleaned up on exit
- **Backward compatible** with Vaulted — sets both `VOP_PROFILE` and `VAULTED_ENV`

## Install

### One-liner (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/NodeSpy/vop/main/install.sh | bash
```

This detects your OS and architecture, downloads the correct binary, and installs it to `/usr/local/bin`. Set `VOP_INSTALL_DIR` to install elsewhere:

```bash
VOP_INSTALL_DIR=~/.local/bin curl -fsSL https://raw.githubusercontent.com/NodeSpy/vop/main/install.sh | bash
```

### Homebrew

```bash
brew tap NodeSpy/vop https://github.com/NodeSpy/vop
brew install vop
```

To upgrade:

```bash
brew upgrade vop
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

### Backend selection

Each profile independently chooses its backend:

- **No `service_account_token`** = CLI backend (uses `op` binary, interactive auth)
- **Has `service_account_token`** = SDK backend (pure Go, no external binaries)

## Commands

| Command | Description |
|---|---|
| `vop ls` | List configured profiles |
| `vop shell <profile>` | Open a shell with AWS credentials |
| `vop <profile>` | Shorthand for `vop shell <profile>` |
| `vop exec <profile> -- <cmd>` | Run a command with credentials |
| `vop add <profile>` | Add a new profile |
| `vop edit <profile>` | Edit an existing profile |
| `vop rm <profile>` | Remove a profile |
| `vop show <profile>` | Show profile configuration |
| `vop dump <profile>` | Dump raw credentials |
| `vop test <profile>` | Test credentials with STS |
| `vop rotate <profile>` | Rotate IAM access keys |
| `vop migrate <profile>` | Migrate from Vaulted |
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

This unlocks the Vaulted vault, extracts the AWS credentials (access key, secret key, MFA serial, region), creates a new 1Password item containing those credentials, and sets up a vop profile pointing to it. If the vault has MFA configured, you'll be prompted to link an existing 1Password TOTP item.

## License

MIT
