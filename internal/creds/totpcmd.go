package creds

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/NodeSpy/vop/internal/ui"
)

// RunTOTPCommand executes the configured shell command and returns its
// trimmed stdout as a TOTP code. Runs via `sh -c` so users can pipe/quote
// naturally (e.g. `pass otp aws/prod`, `ykman oath accounts code prod`,
// `rbw get --raw code aws-prod`).
//
// Inherits the caller's environment so tools that rely on
// GPG_TTY/DBUS_SESSION_BUS_ADDRESS/etc. (pass, secret-service clients)
// keep working. Stderr is passed through so unlock prompts still reach
// the user's terminal.
func RunTOTPCommand(command string) (string, error) {
	if strings.TrimSpace(command) == "" {
		return "", fmt.Errorf("mfa_totp_command is empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	var stdout bytes.Buffer
	cmd.Stdin = os.Stdin
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("mfa_totp_command timed out after 30s: %s", command)
		}
		return "", fmt.Errorf("mfa_totp_command failed: %w", err)
	}

	code := strings.TrimSpace(stdout.String())
	// Some tools (rbw, pass-otp) emit only the code; others may include
	// extra formatting. Grab the last non-empty line as a mild convenience,
	// which matches how humans usually copy-paste TOTPs.
	if strings.Contains(code, "\n") {
		lines := strings.Split(code, "\n")
		for i := len(lines) - 1; i >= 0; i-- {
			if s := strings.TrimSpace(lines[i]); s != "" {
				code = s
				break
			}
		}
	}
	if code == "" {
		return "", fmt.Errorf("mfa_totp_command produced no output")
	}
	return code, nil
}

// RunCredentialsCommand executes the configured shell command and parses
// its stdout as base AWS credentials.
//
// Output format (line-based):
//
//	line 1: aws_access_key_id
//	line 2: aws_secret_access_key
//	line 3: aws_session_token (optional)
//
// Blank lines and lines starting with '#' are ignored so users can annotate
// their `pass` entries. Runs via `sh -c` and inherits the caller's
// environment (so gpg-agent, TTY prompts, etc. work).
func RunCredentialsCommand(command string) (accessKey, secretKey, sessionToken string, err error) {
	if strings.TrimSpace(command) == "" {
		return "", "", "", fmt.Errorf("credentials_command is empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	var stdout bytes.Buffer
	cmd.Stdin = os.Stdin
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	ui.Info("Fetching credentials via credentials_command")
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", "", "", fmt.Errorf("credentials_command timed out after 30s: %s", command)
		}
		return "", "", "", fmt.Errorf("credentials_command failed: %w", err)
	}

	var lines []string
	for _, raw := range strings.Split(stdout.String(), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines = append(lines, line)
	}

	if len(lines) < 2 {
		return "", "", "", fmt.Errorf("credentials_command output must be at least 2 lines (access key, secret); got %d", len(lines))
	}
	accessKey = lines[0]
	secretKey = lines[1]
	if len(lines) >= 3 {
		sessionToken = lines[2]
	}
	return accessKey, secretKey, sessionToken, nil
}

// RunWriteCredentialsCommand executes the configured write-back command
// with the new access key on stdin line 1 and secret on stdin line 2.
// Used by `vop rotate` to persist newly created keys into a command-based
// source (e.g. `pass insert -m -f aws/prod`).
//
// The command receives exactly two lines followed by EOF. Tools that
// expect a specific format (JSON, key=value, etc.) should be wrapped in
// a small shell script.
func RunWriteCredentialsCommand(command, accessKey, secretKey string) error {
	if strings.TrimSpace(command) == "" {
		return fmt.Errorf("credentials_write_command is empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Stdin = strings.NewReader(accessKey + "\n" + secretKey + "\n")
	// Pass stderr through so unlock prompts / warnings reach the user.
	cmd.Stderr = os.Stderr
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("credentials_write_command timed out after 30s: %s", command)
		}
		return fmt.Errorf("credentials_write_command failed: %w", err)
	}
	return nil
}
