package creds

import (
	"encoding/json"
	"errors"
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
	// Kind classifies the last failure so the cooldown message can say
	// something useful about it. Empty in files written by older versions.
	Kind FailureKind `json:"kind,omitempty"`
	// LockedCount tracks consecutive locked-source failures separately
	// from Count. A locked keyring never reaches the upstream provider,
	// so it must not push the profile up the upstream-protection backoff
	// curve — but it still needs to stop a retry storm.
	LockedCount int `json:"locked_count,omitempty"`
}

// FailureKind classifies why a credential fetch failed.
type FailureKind string

const (
	// KindAuth is a generic auth failure: wrong TOTP, expired keys, clock skew.
	KindAuth FailureKind = "auth"
	// KindLocked means the local credential store (gpg-agent, 1Password
	// app, keyring) is locked or couldn't prompt. Nothing reached the
	// upstream provider, and no amount of retrying will help until the
	// user unlocks it.
	KindLocked FailureKind = "locked"
	// KindRateLimit means the upstream source signalled a throttle.
	KindRateLimit FailureKind = "rate_limit"
)

// CooldownError is returned by CheckCooldown when a profile is still
// within its backoff window. Callers can type-assert it to distinguish a
// cooldown from a real credential failure — notably Execute, which maps
// it to a dedicated exit code so automated callers can stop retrying.
type CooldownError struct {
	Profile   string
	Kind      FailureKind
	Remaining time.Duration
	Count     int
	Reason    string
}

func (e *CooldownError) Error() string {
	var b strings.Builder
	switch e.Kind {
	case KindLocked:
		fmt.Fprintf(&b, "credential source for %s is locked — retrying will not help until it is unlocked (waiting %s)",
			e.Profile, e.Remaining)
	case KindRateLimit:
		fmt.Fprintf(&b, "upstream rate limit detected for %s — wait %s before retrying to avoid extending the block",
			e.Profile, e.Remaining)
	default:
		fmt.Fprintf(&b, "recent auth failure for %s (%d attempts) — waiting %s before retry to avoid triggering an upstream rate limit",
			e.Profile, e.Count, e.Remaining)
	}
	if e.Reason != "" {
		fmt.Fprintf(&b, "\n  last error: %s", strings.ReplaceAll(e.Reason, "\n", "\n  "))
	}
	fmt.Fprintf(&b, "\n  run `vop unlock %s` to clear this cooldown once the cause is fixed", e.Profile)
	return b.String()
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
	return &CooldownError{
		Profile:   profileName,
		Kind:      rec.kind(),
		Remaining: remaining.Round(time.Second),
		Count:     rec.Count,
		Reason:    rec.Reason,
	}
}

// kind returns the recorded failure kind, falling back to the legacy
// SourceRateLimit flag for records written by older vop versions.
func (r *FailureRecord) kind() FailureKind {
	if r.Kind != "" {
		return r.Kind
	}
	if r.SourceRateLimit {
		return KindRateLimit
	}
	return KindAuth
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

	switch {
	case IsRateLimitError(err):
		rec.SourceRateLimit = true
		rec.Kind = KindRateLimit
		rec.Count++
	case IsLockedSourceError(err):
		// A locked store never touched the upstream provider, so this
		// failure carries no rate-limit risk. Keep it out of Count so it
		// can't escalate the profile into a 5-minute upstream backoff —
		// but do count it separately so a retry loop still gets damped.
		rec.Kind = KindLocked
		rec.LockedCount++
	default:
		rec.Kind = KindAuth
		rec.Count++
	}

	if err != nil {
		rec.Reason = truncate(err.Error(), 400)
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
	if rec.kind() == KindRateLimit {
		return 15 * time.Minute
	}
	// A locked credential store is a local, user-fixable condition: the
	// fix is "unlock it", not "wait". Use a short flat cooldown that stops
	// a tight retry loop without making the user sit out a long backoff
	// after they've unlocked. The first failure is free so an interactive
	// user who just got a pinentry prompt isn't penalised.
	if rec.kind() == KindLocked {
		if rec.LockedCount <= 1 {
			return 0
		}
		return 20 * time.Second
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

// lockedSignatures are substrings that indicate the local credential store
// couldn't be read because it is locked, or because it needed to prompt and
// had nowhere to do it. These are all conditions the user fixes locally —
// retrying without intervention is pure waste.
var lockedSignatures = []string{
	// gpg / pass
	"gpg: decryption failed",
	"no secret key",
	"secret key not available",
	"gpg-agent",
	"pinentry",
	"inappropriate ioctl for device",
	"no such device or address",
	"cannot open display",
	"operation cancelled",
	"operation canceled",
	// 1Password
	"not currently signed in",
	"session expired",
	"authorization prompt",
	"connecting to desktop app",
	// generic keyrings / vaults
	"is locked",
	"vault is locked",
	"database is locked",
}

// IsLockedSourceError reports whether err looks like a locked or
// un-promptable credential store rather than a genuine auth rejection.
func IsLockedSourceError(err error) bool {
	if err == nil {
		return false
	}
	var sce *SourceCommandError
	if errors.As(err, &sce) {
		// A timeout on a source command is almost always a prompt nobody
		// could answer — same remediation as an outright locked store.
		if sce.TimedOut {
			return true
		}
		return isLockedMessage(sce.Stderr)
	}
	return isLockedMessage(err.Error())
}

func isLockedMessage(s string) bool {
	s = strings.ToLower(s)
	for _, sig := range lockedSignatures {
		if strings.Contains(s, sig) {
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
