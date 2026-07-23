# vop

AWS credential management via 1Password. Spawn shells and run commands with AWS credentials fetched directly from your 1Password vault.

## Features

- **Shell sessions** with AWS credentials injected as environment variables
- **Exec mode** to run single commands with credentials
- **MFA support** with automatic TOTP retrieval from 1Password, or from any external command (`pass-otp`, `rbw`, `ykman`, KeePassXC, …)
- **Flexible sources** — keys can come from a 1Password item or straight from `~/.aws/credentials`
- **Rate-limit protection** — auth failures trigger a local cooldown so retries don't hammer the upstream credential source (1Password, `pass`/gpg-agent, etc.)
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
- **`.deb`** -- on Debian/Ubuntu (and derivatives), offers `apt install` of the official `.deb`
- **Binary download** -- always available as a fallback, downloads to `/usr/local/bin`

When multiple methods are available, you'll be prompted to choose. To force a specific method or override the install directory:

```bash
# Force a specific method
VOP_METHOD=brew curl -fsSL https://raw.githubusercontent.com/NodeSpy/vop/main/install.sh | bash
VOP_METHOD=aur curl -fsSL https://raw.githubusercontent.com/NodeSpy/vop/main/install.sh | bash
VOP_METHOD=deb curl -fsSL https://raw.githubusercontent.com/NodeSpy/vop/main/install.sh | bash
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

### Debian / Ubuntu (.deb)

```bash
# amd64
curl -fsSLO "https://github.com/NodeSpy/vop/releases/latest/download/vop_$(curl -fsSL https://api.github.com/repos/NodeSpy/vop/releases/latest | grep -oP '"tag_name": *"v\K[^"]+')_amd64.deb"
sudo apt install ./vop_*.deb
```

`apt install` resolves `1password-cli` (recommended) and integrates with `apt list --installed` / `apt remove`. Upgrade on each release with `sudo apt install --only-upgrade vop` or re-run the install one-liner.

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

This detects all vop installations on your system (Homebrew, AUR, .deb, standalone binary) and offers to remove them. If you have both a package-managed install and a standalone binary (e.g. from a previous one-liner install), it will warn about the conflict and default to removing only the standalone binary.

To uninstall a specific method manually:

```bash
# Homebrew
brew uninstall vop && brew untap NodeSpy/vop

# AUR
yay -R vop-bin

# Debian/Ubuntu (.deb)
sudo apt remove vop

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
| `op_item` | 1P source only | 1Password item name (e.g. `AWS - prod`) |
| `op_vault` | SDK only | 1Password vault name (for `op://vault/item/field` refs) |
| `service_account_token` | SDK only | 1Password service account token |
| `aws_credentials_profile` | File source | Profile name in `~/.aws/credentials` to read/rotate keys against (see below) |
| `aws_credentials_file` | No | Override path for the shared credentials file (defaults to `~/.aws/credentials`) |
| `credentials_command` | Command source | Shell command whose stdout is the AWS keys (2-3 lines: access key, secret, optional session token) |
| `credentials_write_command` | No | Shell command that receives new keys on stdin (line 1: access key, line 2: secret). Enables `vop rotate` for command sources. |
| `mfa_totp_item` | No | 1Password item that holds the MFA TOTP |
| `mfa_totp_command` | No | Shell command whose stdout is the current TOTP (see below) |
| `mfa_serial` | No | Explicit MFA device ARN (overrides file/1P/IAM lookup) |
| `iam_username` | No | IAM username (for key rotation) |
| `description` | No | Profile description |
| `agent_policy` | No | Custom agent instructions (default: read-only) |
| `role_arn` | Role only | ARN of an IAM role to `sts:AssumeRole` into |
| `source_profile` | Role only | Name of another vop profile whose creds are used to assume the role |
| `role_session_name` | No | Session name passed to `sts:AssumeRole` (defaults to `vop`) |
| `external_id` | No | ExternalId condition value for role assumption |

### Backend selection

Each profile independently chooses its backend:

- **No `service_account_token`** = CLI backend (uses `op` binary, interactive auth)
- **Has `service_account_token`** = SDK backend (pure Go, no external binaries)

### Alternative credential sources

By default, vop reads AWS keys from a 1Password item. Three optional fields let you source the base keys and/or the TOTP from somewhere else:

- `aws_credentials_profile` — read the keys straight from `~/.aws/credentials`
- `credentials_command` — read the keys from any shell command (e.g. `pass aws/prod`)
- `mfa_totp_command` — read the TOTP from any shell command (e.g. `pass otp aws/prod`, `ykman`, `rbw`, `keepassxc-cli`)

These are independent — mix and match freely. See the [Combining sources](#combining-sources) matrix below.

#### Reading keys from `~/.aws/credentials`

Set `aws_credentials_profile` to the section name in your shared credentials file. vop will read the access key and secret from there instead of 1Password, and `vop rotate` will write the new keys back to the same section (preserving comments, other profiles, and file permissions).

```json
{
  "profiles": {
    "ednition": {
      "aws_credentials_profile": "ednition",
      "mfa_totp_command": "pass otp aws/ednition"
    }
  }
}
```

vop discovers `mfa_serial` automatically from `~/.aws/config` (or `~/.aws/credentials`, wherever the AWS CLI put it) — no need to duplicate it in the vop profile. Override via the `mfa_serial` field only if you need something different from the AWS CLI's view. Override the credentials file path with `aws_credentials_file` if it isn't at `~/.aws/credentials`.

#### Reading keys from `pass` (or any other command)

`credentials_command` runs an arbitrary shell command and parses its stdout as the base AWS keys. vop uses `sh -c`, inherits your environment (so `gpg-agent` / DBus / TTY prompts keep working), and expects:

- Line 1: `aws_access_key_id`
- Line 2: `aws_secret_access_key`
- Line 3 (optional): `aws_session_token`

Blank lines and lines starting with `#` are ignored, so you can annotate your entries.

##### Quick start with `pass`

The recommended encrypted-at-rest setup: both the access-key pair and the TOTP live in [`pass`](https://www.passwordstore.org/), guarded by GPG. `gpg-agent`'s cache TTL keeps you from re-entering the passphrase on every call, and nothing sensitive sits on disk in plaintext.

```bash
# 1. Store the access key + secret as a two-line pass entry.
pass insert -m aws/prod
# Editor opens; paste:
#   AKIAIOSFODNN7EXAMPLE
#   wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

# 2. Store the TOTP seed (via pass-otp).
pass otp insert aws/prod
# Paste the otpauth:// URI or the base32 secret.

# 3. Point vop at both.
```

Then in `~/.config/vop/profiles.json`:

```json
{
  "profiles": {
    "prod": {
      "credentials_command": "pass aws/prod",
      "mfa_totp_command": "pass otp aws/prod",
      "mfa_serial": "arn:aws:iam::123456789012:mfa/daniel"
    }
  }
}
```

That's it — `vop prod` unlocks `pass` (once per `gpg-agent` window) and hands you an MFA-authenticated shell. No `op` CLI, no 1Password service account, no plaintext keys on disk.

##### Other tools

Any command that produces the 2-3-line format works directly. For tools that don't, wrap them:

```bash
# ~/.local/bin/aws-from-1p — mixing 1Password items with pass-otp
#!/bin/bash
op read "op://Personal/AWS $1/access key id"
op read "op://Personal/AWS $1/secret access key"
```

```json
"credentials_command": "aws-from-1p prod"
```

##### Rotation

`vop rotate` works for command sources if you also configure `credentials_write_command` — a command that accepts the new keys on stdin (line 1: access key, line 2: secret) and updates the source. For pass:

```json
{
  "prod": {
    "credentials_command": "pass aws/prod",
    "credentials_write_command": "pass insert -m -f aws/prod",
    "mfa_totp_command": "pass otp aws/prod",
    "mfa_serial": "arn:aws:iam::123456789012:mfa/daniel"
  }
}
```

Now `vop rotate prod` creates a new IAM key, pipes `aws_access_key_id\naws_secret_access_key\n` into `pass insert -m -f aws/prod` (which overwrites the entry), verifies via STS with the new key + a fresh TOTP, and deletes the old key. On any failure, vop rolls the pass entry back to the previous keys automatically.

If `credentials_write_command` isn't set, `vop rotate` refuses cleanly and points you here.

#### External TOTP command

Set `mfa_totp_command` to a shell command that prints the current 6-digit code. vop runs it via `sh -c`, inherits your environment (so `gpg-agent`, DBus, and TTY prompts keep working), and uses the trimmed last non-empty line as the code. Takes precedence over `mfa_totp_item` when both are set.

Examples:

```json
"mfa_totp_command": "pass otp aws/prod"
"mfa_totp_command": "rbw get --raw code aws-prod"
"mfa_totp_command": "ykman oath accounts code -s aws-prod"
"mfa_totp_command": "keepassxc-cli show -a otp ~/vault.kdbx aws-prod"
```

#### Combining sources

The fields are independent — mix and match freely:

| Base keys | TOTP | Config |
|---|---|---|
| 1Password | 1Password | (default) — `op_item`, `mfa_totp_item` |
| 1Password | External | `op_item` + `mfa_totp_command` |
| AWS file | 1Password | `aws_credentials_profile` + `mfa_totp_item` |
| AWS file | External | `aws_credentials_profile` + `mfa_totp_command` |
| Command | 1Password | `credentials_command` + `mfa_totp_item` |
| Command | External | `credentials_command` + `mfa_totp_command` |

Precedence when multiple sources are set: `credentials_command` > `aws_credentials_profile` > `op_item`. When a profile has no 1P interaction at all, vop skips the `op` sign-in entirely — you can use these profiles without the `op` CLI installed.

> `vop add` and `vop edit` don't prompt for these fields yet; add them by editing `~/.config/vop/profiles.json` directly.

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
| `vop unlock <profile>` | Clear a profile's failure cooldown after a fixed auth issue |
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
