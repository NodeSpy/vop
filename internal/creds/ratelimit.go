package creds

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FailureRecord tracks recent failed credential fetches for a profile.
// Persisted to disk so the cooldown survives across vop invocations,
// preventing rapid retries from hammering the upstream credential source
// (1Password, pass/gpg-agent, etc.) after an auth failure like an
// out-of-sync MFA code.
type FailureRecord struct {
	FirstFailure time.Time `json:"first_failure"`
	LastFailure  time.Time `json:"last_failure"`
	Count        int       `json:"count"`
	// SourceRateLimit is set when the failure looked like the upstream
	// source itself throttled us (HTTP 429, "too many requests", etc.).
	// Triggers a much longer cooldown than a generic auth failure.
	// The json tag stays `op_rate_limit` for on-disk compatibility with
	// files written by older vop versions.
	SourceRateLimit bool `json:"op_rate_limit,omitempty"`
	// Reason is a short human-readable summary of the last failure,
	// shown in the cooldown message.
	Reason string `json:"reason,omitempty"`
}

func failureFile(profileName string) string {
	return filepath.Join(RuntimeDir(), profileName+".failures.json")
}

// CheckCooldown returns nil if it's safe to hit 1Password for this profile.
// Otherwise it returns an error describing the remaining wait time.
func CheckCooldown(profileName string) error {
	rec := loadFailure(profileName)
	if rec == nil {
		return nil
	}
	wait := backoffFor(rec)
	if wait <= 0 {
		return nil
	}
	remaining := time.Until(rec.LastFailure.Add(wait))
	if remaining <= 0 {
		return nil
	}
	rounded := remaining.Round(time.Second)
	if rec.SourceRateLimit {
		return fmt.Errorf(
			"upstream rate limit detected — wait %s before retrying to avoid extending the block. Run `vop unlock %s` to override",
			rounded, profileName,
		)
	}
	return fmt.Errorf(
		"recent auth failure for %s (%d attempts) — waiting %s before retry to avoid triggering an upstream rate limit. Run `vop unlock %s` to override",
		profileName, rec.Count, rounded, profileName,
	)
}

// RecordFailure notes that a credential fetch failed. If the underlying
// error indicates 1Password itself rate limited us, a longer cooldown applies.
func RecordFailure(profileName string, err error) {
	rec := loadFailure(profileName)
	now := time.Now()
	if rec == nil {
		rec = &FailureRecord{FirstFailure: now}
	}
	rec.LastFailure = now
	rec.Count++
	if IsRateLimitError(err) {
		rec.SourceRateLimit = true
	}
	if err != nil {
		rec.Reason = truncate(err.Error(), 200)
	}
	saveFailure(profileName, rec)
}

// ClearFailures resets the failure record on a successful fetch or
// on explicit user override.
func ClearFailures(profileName string) {
	os.Remove(failureFile(profileName))
}

// backoffFor returns how long to wait after rec.LastFailure before
// allowing a retry.
//
// Normal auth failures use exponential backoff. The first failure is
// free (users mistype passwords, TOTP windows expire) — real
// protection kicks in on the second+ attempt.
//
// Upstream-signalled rate limits use a fixed longer cooldown. 1P's
// account-level throttle typically lasts ~10-15 minutes, and other
// providers are usually in that ballpark, so we wait that long before
// letting the user try again.
func backoffFor(rec *FailureRecord) time.Duration {
	if rec.SourceRateLimit {
		return 15 * time.Minute
	}
	switch {
	case rec.Count <= 1:
		return 0
	case rec.Count == 2:
		return 5 * time.Second
	case rec.Count == 3:
		return 15 * time.Second
	case rec.Count == 4:
		return 30 * time.Second
	case rec.Count == 5:
		return 1 * time.Minute
	default:
		return 5 * time.Minute
	}
}

// IsRateLimitError reports whether err smells like a 1Password (or
// upstream) rate-limit response. 1P returns 429s and messages
// containing "rate limit" / "too many requests" when throttled.
func IsRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	needles := []string{
		"rate limit",
		"rate-limit",
		"ratelimit",
		"too many requests",
		"429",
		"throttl",
	}
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

func loadFailure(profileName string) *FailureRecord {
	data, err := os.ReadFile(failureFile(profileName))
	if err != nil {
		return nil
	}
	var rec FailureRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil
	}
	return &rec
}

func saveFailure(profileName string, rec *FailureRecord) {
	dir := RuntimeDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return
	}
	_ = os.WriteFile(failureFile(profileName), data, 0600)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
