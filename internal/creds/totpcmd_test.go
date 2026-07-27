package creds

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestRunTOTPCommand_Success(t *testing.T) {
	code, err := RunTOTPCommand("echo 123456")
	if err != nil {
		t.Fatal(err)
	}
	if code != "123456" {
		t.Errorf("got %q, want %q", code, "123456")
	}
}

func TestRunTOTPCommand_EmptyErrors(t *testing.T) {
	if _, err := RunTOTPCommand(""); err == nil {
		t.Fatal("expected error for empty command")
	}
	if _, err := RunTOTPCommand("   "); err == nil {
		t.Fatal("expected error for whitespace-only command")
	}
}

func TestRunTOTPCommand_NoOutput(t *testing.T) {
	if _, err := RunTOTPCommand("true"); err == nil {
		t.Fatal("expected error when command produces no output")
	}
}

func TestRunTOTPCommand_TakesLastLine(t *testing.T) {
	// Tools that emit chatty output before the code (e.g. warnings).
	code, err := RunTOTPCommand("printf 'warning: unlock ttl 5m\\n789012\\n'")
	if err != nil {
		t.Fatal(err)
	}
	if code != "789012" {
		t.Errorf("got %q, want %q", code, "789012")
	}
}

func TestRunTOTPCommand_TrimsWhitespace(t *testing.T) {
	code, err := RunTOTPCommand("printf '  654321  \\n'")
	if err != nil {
		t.Fatal(err)
	}
	if code != "654321" {
		t.Errorf("got %q, want %q", code, "654321")
	}
}

func TestRunTOTPCommand_FailingCommand(t *testing.T) {
	_, err := RunTOTPCommand("false")
	if err == nil {
		t.Fatal("expected error from failing command")
	}
	if !strings.Contains(err.Error(), "mfa_totp_command failed") {
		t.Errorf("expected 'mfa_totp_command failed' in error, got: %v", err)
	}
}

func TestRunCredentialsCommand_TwoLines(t *testing.T) {
	ak, sk, tok, err := RunCredentialsCommand("printf 'AKIAEXAMPLE\\nsecretExample\\n'")
	if err != nil {
		t.Fatal(err)
	}
	if ak != "AKIAEXAMPLE" || sk != "secretExample" {
		t.Errorf("got ak=%q sk=%q", ak, sk)
	}
	if tok != "" {
		t.Errorf("expected empty session token, got %q", tok)
	}
}

func TestRunCredentialsCommand_ThreeLinesWithToken(t *testing.T) {
	ak, sk, tok, err := RunCredentialsCommand("printf 'AK\\nSK\\nTOKEN\\n'")
	if err != nil {
		t.Fatal(err)
	}
	if ak != "AK" || sk != "SK" || tok != "TOKEN" {
		t.Errorf("got ak=%q sk=%q tok=%q", ak, sk, tok)
	}
}

func TestRunCredentialsCommand_SkipsBlankAndCommentLines(t *testing.T) {
	// A `pass` entry may include comments or blank lines for humans.
	script := "printf '# aws prod\\n\\nAKIAEXAMPLE\\nsecretExample\\n\\n# scratchpad\\n'"
	ak, sk, tok, err := RunCredentialsCommand(script)
	if err != nil {
		t.Fatal(err)
	}
	if ak != "AKIAEXAMPLE" || sk != "secretExample" || tok != "" {
		t.Errorf("got ak=%q sk=%q tok=%q", ak, sk, tok)
	}
}

func TestRunCredentialsCommand_TooFewLines(t *testing.T) {
	if _, _, _, err := RunCredentialsCommand("echo only_one_line"); err == nil {
		t.Fatal("expected error for single-line output")
	}
}

func TestRunCredentialsCommand_EmptyErrors(t *testing.T) {
	if _, _, _, err := RunCredentialsCommand(""); err == nil {
		t.Fatal("expected error for empty command")
	}
}

func TestRunCredentialsCommand_FailingCommand(t *testing.T) {
	_, _, _, err := RunCredentialsCommand("false")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "credentials_command failed") {
		t.Errorf("expected 'credentials_command failed' in error, got: %v", err)
	}
}

func TestRunWriteCredentialsCommand_PipesStdin(t *testing.T) {
	dir := t.TempDir()
	sink := dir + "/received.txt"
	// Command captures stdin to a file so we can assert on the exact bytes vop wrote.
	if err := RunWriteCredentialsCommand("cat > "+sink, "NEW_AK", "NEW_SK"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := os.ReadFile(sink)
	if err != nil {
		t.Fatal(err)
	}
	want := "NEW_AK\nNEW_SK\n"
	if string(got) != want {
		t.Errorf("stdin mismatch:\n got: %q\nwant: %q", string(got), want)
	}
}

func TestRunWriteCredentialsCommand_EmptyErrors(t *testing.T) {
	if err := RunWriteCredentialsCommand("", "AK", "SK"); err == nil {
		t.Fatal("expected error for empty command")
	}
}

func TestRunWriteCredentialsCommand_PropagatesFailure(t *testing.T) {
	err := RunWriteCredentialsCommand("false", "AK", "SK")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "credentials_write_command failed") {
		t.Errorf("expected 'credentials_write_command failed' in error, got: %v", err)
	}
}

func TestSourceCommandError_CapturesStderr(t *testing.T) {
	_, err := RunTOTPCommand("echo 'gpg: decryption failed: No secret key' >&2; exit 2")
	if err == nil {
		t.Fatal("expected an error")
	}
	var sce *SourceCommandError
	if !errors.As(err, &sce) {
		t.Fatalf("expected *SourceCommandError, got %T", err)
	}
	if !strings.Contains(sce.Stderr, "decryption failed") {
		t.Errorf("stderr not captured, got %q", sce.Stderr)
	}
	// The rendered error must carry the diagnostic and the remedy —
	// "exit status 2" alone tells the user nothing.
	msg := err.Error()
	for _, want := range []string{"mfa_totp_command", "decryption failed", "hint:"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error message missing %q:\n%s", want, msg)
		}
	}
}

func TestSourceCommandError_NoHintForOrdinaryFailure(t *testing.T) {
	_, err := RunTOTPCommand("echo 'aws/prod: passfile not found.' >&2; exit 1")
	if err == nil {
		t.Fatal("expected an error")
	}
	var sce *SourceCommandError
	if !errors.As(err, &sce) {
		t.Fatalf("expected *SourceCommandError, got %T", err)
	}
	if h := sce.Hint(); h != "" {
		t.Errorf("expected no unlock hint for a missing entry, got %q", h)
	}
	if !strings.Contains(err.Error(), "passfile not found") {
		t.Errorf("expected stderr in message, got:\n%s", err)
	}
}

func TestRunCredentialsCommand_ErrorCarriesStderr(t *testing.T) {
	_, _, _, err := RunCredentialsCommand("echo 'gpg-agent is not running' >&2; exit 2")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "gpg-agent is not running") {
		t.Errorf("expected stderr in message, got:\n%s", err)
	}
	if !IsLockedSourceError(err) {
		t.Error("expected a gpg-agent failure to classify as locked")
	}
}
