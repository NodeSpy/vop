package creds

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/NodeSpy/vop/internal/ui"
)

// runSourceCommand executes a user-configured credential command via `sh -c`,
// returning its stdout.
//
// Stderr is tee'd: it still reaches the user's terminal (so pinentry and
// unlock prompts work) but is also captured so the diagnostic text can be
// folded into the returned error. Without this the caller only ever sees
// "exit status 1", which is useless for telling a locked gpg-agent apart
// from a typo'd item name — and useless for deciding whether retrying is
// worth anything.
//
// stdin is only wired up when the terminal is interactive. In a
// non-interactive context (an agent, a cron job) a pinentry prompt would
// otherwise block until the timeout, so we fail fast and say why.
func runSourceCommand(label, command string, stdin io.Reader) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	var stdout, stderr bytes.Buffer
	cmd.Stdin = stdin
	cmd.Stdout = &stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)

	err := cmd.Run()
	if err == nil {
		return stdout.String(), nil
	}
	if ctx.Err() == context.DeadlineExceeded {
		return "", &SourceCommandError{
			Label:    label,
			Command:  command,
			Stderr:   tailLines(stderr.String(), 3),
			TimedOut: true,
		}
	}
	return "", &SourceCommandError{
		Label:   label,
		Command: command,
		Stderr:  tailLines(stderr.String(), 3),
		Err:     err,
	}
}

// SourceCommandError describes a failed credential/TOTP command. It carries
// the command and its stderr so callers can classify the failure (locked
// keyring vs. missing entry vs. hung prompt) and show the user something
// actionable.
type SourceCommandError struct {
	Label    string // e.g. "mfa_totp_command"
	Command  string
	Stderr   string
	TimedOut bool
	Err      error
}

func (e *SourceCommandError) Error() string {
	var b strings.Builder
	if e.TimedOut {
		fmt.Fprintf(&b, "%s timed out after 30s", e.Label)
	} else {
		fmt.Fprintf(&b, "%s failed (%v)", e.Label, e.Err)
	}
	fmt.Fprintf(&b, ": %s", e.Command)
	if e.Stderr != "" {
		fmt.Fprintf(&b, "\n  %s", strings.ReplaceAll(e.Stderr, "\n", "\n  "))
	}
	if hint := e.Hint(); hint != "" {
		fmt.Fprintf(&b, "\n  hint: %s", hint)
	}
	return b.String()
}

func (e *SourceCommandError) Unwrap() error { return e.Err }

// Hint returns remediation advice for the common failure modes, or "" when
// the failure doesn't match a known shape.
func (e *SourceCommandError) Hint() string {
	var cause string
	switch {
	case e.TimedOut:
		cause = "the command was probably waiting on an interactive prompt"
	case isLockedMessage(e.Stderr):
		cause = "the credential store looks locked"
	default:
		return ""
	}
	hint := fmt.Sprintf("%s. Run `%s` in an interactive terminal to unlock it, then retry", cause, e.Command)
	if !ui.IsInteractive() {
		hint += " (vop has no terminal here, so it cannot prompt you itself)"
	}
	return hint
}

// tailLines returns at most the last n non-empty lines of s.
func tailLines(s string, n int) string {
	var lines []string
	for _, raw := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(raw); t != "" {
			lines = append(lines, t)
		}
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

// commandStdin returns os.Stdin when it's a terminal, and nil otherwise.
// Handing a non-terminal stdin to pinentry just produces a 30s hang.
func commandStdin() io.Reader {
	if ui.IsInteractive() {
		return os.Stdin
	}
	return nil
}

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

	out, err := runSourceCommand("mfa_totp_command", command, commandStdin())
	if err != nil {
		return "", err
	}

	code := strings.TrimSpace(out)
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

	ui.Info("Fetching credentials via credentials_command")
	out, err := runSourceCommand("credentials_command", command, commandStdin())
	if err != nil {
		return "", "", "", err
	}

	var lines []string
	for _, raw := range strings.Split(out, "\n") {
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

	_, err := runSourceCommand("credentials_write_command", command,
		strings.NewReader(accessKey+"\n"+secretKey+"\n"))
	return err
}
