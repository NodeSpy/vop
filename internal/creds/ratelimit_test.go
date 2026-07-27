package creds

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestIsRateLimitError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"generic", errors.New("something went wrong"), false},
		{"429", errors.New("HTTP 429 from server"), true},
		{"rate limit", errors.New("rate limit exceeded"), true},
		{"too many requests", errors.New("Too Many Requests"), true},
		{"throttled", errors.New("request was throttled"), true},
		{"wrapped", fmt.Errorf("outer: %w", errors.New("rate limit exceeded")), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsRateLimitError(tc.err); got != tc.want {
				t.Errorf("IsRateLimitError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestBackoffFor(t *testing.T) {
	cases := []struct {
		rec  FailureRecord
		want time.Duration
	}{
		{FailureRecord{Count: 1}, 0},
		{FailureRecord{Count: 2}, 5 * time.Second},
		{FailureRecord{Count: 3}, 15 * time.Second},
		{FailureRecord{Count: 4}, 30 * time.Second},
		{FailureRecord{Count: 5}, 1 * time.Minute},
		{FailureRecord{Count: 10}, 5 * time.Minute},
		{FailureRecord{Count: 1, SourceRateLimit: true}, 15 * time.Minute},
		{FailureRecord{Count: 99, SourceRateLimit: true}, 15 * time.Minute},
	}
	for _, tc := range cases {
		if got := backoffFor(&tc.rec); got != tc.want {
			t.Errorf("backoffFor(%+v) = %v, want %v", tc.rec, got, tc.want)
		}
	}
}

// isolateRuntimeDir points RuntimeDir at a temp dir for the test.
func isolateRuntimeDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)
	return dir
}

func TestCheckCooldown_NoRecord(t *testing.T) {
	isolateRuntimeDir(t)
	if err := CheckCooldown("nobody"); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestRecordFailure_FirstIsFree(t *testing.T) {
	isolateRuntimeDir(t)
	RecordFailure("p", errors.New("bad totp"))
	if err := CheckCooldown("p"); err != nil {
		t.Fatalf("first failure should not gate retries, got %v", err)
	}
}

func TestRecordFailure_SecondTriggersBackoff(t *testing.T) {
	isolateRuntimeDir(t)
	RecordFailure("p", errors.New("bad totp"))
	RecordFailure("p", errors.New("bad totp"))
	err := CheckCooldown("p")
	if err == nil {
		t.Fatal("expected cooldown error after 2 failures")
	}
}

func TestRecordFailure_RateLimitEnforcesLongCooldown(t *testing.T) {
	isolateRuntimeDir(t)
	RecordFailure("p", errors.New("HTTP 429: rate limit exceeded"))
	err := CheckCooldown("p")
	if err == nil {
		t.Fatal("expected cooldown error immediately on rate limit")
	}
	// Should mention rate limit in the message (source-agnostic).
	if !strings.Contains(err.Error(), "rate limit") {
		t.Errorf("expected 'rate limit' in error, got: %v", err)
	}
}

func TestClearFailures(t *testing.T) {
	isolateRuntimeDir(t)
	RecordFailure("p", errors.New("bad"))
	RecordFailure("p", errors.New("bad"))
	if err := CheckCooldown("p"); err == nil {
		t.Fatal("setup: expected cooldown before clear")
	}
	ClearFailures("p")
	if err := CheckCooldown("p"); err != nil {
		t.Fatalf("expected no cooldown after clear, got %v", err)
	}
	if _, err := os.Stat(failureFile("p")); !os.IsNotExist(err) {
		t.Errorf("expected failure file removed, stat err = %v", err)
	}
}

func TestCooldownExpires(t *testing.T) {
	isolateRuntimeDir(t)
	RecordFailure("p", errors.New("bad"))
	RecordFailure("p", errors.New("bad"))
	// Rewind LastFailure past the backoff window (5s for count=2).
	rec := loadFailure("p")
	if rec == nil {
		t.Fatal("expected record to exist")
	}
	rec.LastFailure = time.Now().Add(-10 * time.Second)
	saveFailure("p", rec)
	if err := CheckCooldown("p"); err != nil {
		t.Fatalf("expected cooldown to have expired, got %v", err)
	}
}

func TestIsLockedSourceError(t *testing.T) {
	locked := []error{
		errors.New("gpg: decryption failed: No secret key"),
		errors.New("gpg-agent: no pinentry"),
		errors.New("Inappropriate ioctl for device"),
		errors.New("you are not currently signed in"),
		&SourceCommandError{Stderr: "gpg: decryption failed", Err: errors.New("exit status 2")},
		&SourceCommandError{TimedOut: true},
		fmt.Errorf("fetching TOTP: %w", &SourceCommandError{Stderr: "gpg-agent unavailable"}),
	}
	for _, err := range locked {
		if !IsLockedSourceError(err) {
			t.Errorf("expected locked classification for: %v", err)
		}
	}

	notLocked := []error{
		nil,
		errors.New("HTTP 429: rate limit exceeded"),
		errors.New("the security token included in the request is invalid"),
		// The cooldown message names `vop unlock` — it must not be
		// mistaken for a locked source and re-recorded as one.
		&CooldownError{Profile: "p", Kind: KindAuth, Count: 3},
		// A missing entry is a config error, not a locked store: it needs
		// a config fix, and the generic backoff is the right response.
		&SourceCommandError{Stderr: "aws/prod: passfile not found."},
	}
	for _, err := range notLocked {
		if IsLockedSourceError(err) {
			t.Errorf("expected non-locked classification for: %v", err)
		}
	}
}

func TestRecordFailure_LockedDoesNotEscalateUpstreamBackoff(t *testing.T) {
	isolateRuntimeDir(t)
	lockErr := &SourceCommandError{Stderr: "gpg: decryption failed: No secret key"}
	for range 8 {
		RecordFailure("p", lockErr)
	}

	rec := loadFailure("p")
	if rec.Kind != KindLocked {
		t.Fatalf("kind = %q, want %q", rec.Kind, KindLocked)
	}
	if rec.Count != 0 {
		t.Errorf("locked failures must not increment the upstream counter, Count = %d", rec.Count)
	}
	if got := backoffFor(rec); got != 20*time.Second {
		t.Errorf("backoff = %v, want the flat locked cooldown of 20s", got)
	}
}

func TestRecordFailure_LockedFirstIsFree(t *testing.T) {
	isolateRuntimeDir(t)
	RecordFailure("p", &SourceCommandError{Stderr: "gpg: decryption failed"})
	if err := CheckCooldown("p"); err != nil {
		t.Fatalf("first locked failure should not gate a retry, got %v", err)
	}
	RecordFailure("p", &SourceCommandError{Stderr: "gpg: decryption failed"})
	if err := CheckCooldown("p"); err == nil {
		t.Fatal("expected a cooldown after the second locked failure")
	}
}

func TestCooldownError_ReportsCauseAndRemedy(t *testing.T) {
	isolateRuntimeDir(t)
	lockErr := &SourceCommandError{
		Label:   "mfa_totp_command",
		Command: "pass otp aws/prod",
		Stderr:  "gpg: decryption failed: No secret key",
		Err:     errors.New("exit status 2"),
	}
	RecordFailure("p", lockErr)
	RecordFailure("p", lockErr)

	err := CheckCooldown("p")
	if err == nil {
		t.Fatal("expected a cooldown error")
	}
	var ce *CooldownError
	if !errors.As(err, &ce) {
		t.Fatalf("expected *CooldownError, got %T", err)
	}
	if ce.Kind != KindLocked {
		t.Errorf("kind = %q, want %q", ce.Kind, KindLocked)
	}
	// The whole point: the message has to name the real cause, not just
	// "recent auth failure", or the caller keeps retrying blindly.
	for _, want := range []string{"locked", "gpg: decryption failed", "pass otp aws/prod", "vop unlock p"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("cooldown message missing %q:\n%s", want, err)
		}
	}
}

func TestLegacyRateLimitRecordStillClassified(t *testing.T) {
	// Records written before Kind existed only set op_rate_limit.
	rec := &FailureRecord{SourceRateLimit: true, Count: 1}
	if rec.kind() != KindRateLimit {
		t.Errorf("kind = %q, want %q", rec.kind(), KindRateLimit)
	}
	if got := backoffFor(rec); got != 15*time.Minute {
		t.Errorf("backoff = %v, want 15m", got)
	}
}
