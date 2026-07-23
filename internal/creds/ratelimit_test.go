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

