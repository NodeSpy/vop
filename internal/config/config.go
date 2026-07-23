// Package config manages the vop profiles configuration file.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Profile represents a single AWS/1Password profile mapping.
type Profile struct {
	OPAccount   string `json:"op_account,omitempty"`
	OPItem      string `json:"op_item,omitempty"`
	OPVault     string `json:"op_vault,omitempty"`
	Description string `json:"description,omitempty"`
	MFATOTPItem string `json:"mfa_totp_item,omitempty"`
	IAMUsername string `json:"iam_username,omitempty"`

	// AWSCredentialsProfile, if set, tells vop to read the base access key
	// and secret from ~/.aws/credentials under this profile name instead of
	// from 1Password. Useful when the keys are already managed by the AWS
	// CLI and you only need vop for MFA/session/role-assumption ergonomics.
	AWSCredentialsProfile string `json:"aws_credentials_profile,omitempty"`

	// AWSCredentialsFile overrides the path to the shared credentials file.
	// Defaults to ~/.aws/credentials (or $AWS_SHARED_CREDENTIALS_FILE).
	AWSCredentialsFile string `json:"aws_credentials_file,omitempty"`

	// CredentialsCommand, if set, is executed via `sh -c` and its stdout
	// is parsed as the base AWS credentials. Output must be at least two
	// lines: line 1 is the access key ID, line 2 is the secret access key,
	// optional line 3 is a session token. Takes precedence over both
	// AWSCredentialsProfile and the 1Password item.
	//
	// Designed to work naturally with `pass` (two-line entry): store the
	// access key on line 1 and the secret on line 2, then set this to
	// `pass aws/prod`. For tools that don't produce that layout, wrap
	// them in a small shell script.
	CredentialsCommand string `json:"credentials_command,omitempty"`

	// MFATOTPCommand, if set, is executed via `sh -c` and its trimmed stdout
	// is used as the current TOTP code. Takes precedence over MFATOTPItem.
	// Lets users delegate TOTP storage to any external tool: pass-otp, rbw,
	// ykman, keepassxc-cli, etc.
	MFATOTPCommand string `json:"mfa_totp_command,omitempty"`

	// MFASerial, if set, is the ARN of the MFA device. Skips the 1P field
	// lookup and iam:ListMFADevices fallback. Handy when using an
	// AWS-credentials-file source that doesn't have MFA metadata in 1P.
	MFASerial string `json:"mfa_serial,omitempty"`

	// FieldPrefix is prepended to 1Password field labels when reading/writing
	// AWS credentials. For example, "vop." means fields are stored as
	// "vop.access key id" instead of "access key id". Empty means no prefix
	// (use the bare field names, e.g. for existing items with standard fields).
	// Ignored for any field that has an explicit mapping in FieldMap.
	FieldPrefix string `json:"field_prefix,omitempty"`

	// FieldMap stores explicit 1Password field label overrides, keyed by the
	// vop base name. For example: {"access key id": "AWS Access Key"} means
	// vop reads/writes the field labeled "AWS Access Key" instead of applying
	// the prefix. Only mapped fields are overridden; unmapped fields still
	// use FieldPrefix.
	FieldMap map[string]string `json:"field_map,omitempty"`

	// ServiceAccountToken enables the 1Password SDK backend instead of the
	// op CLI. When set, OPVault is required and OPAccount is ignored.
	ServiceAccountToken string `json:"service_account_token,omitempty"`

	// AgentPolicy contains custom instructions for AI agents provided via
	// 'vop agent'. When empty, the default read-only instructions are used.
	// Any non-empty string replaces the default instructions entirely.
	AgentPolicy string `json:"agent_policy,omitempty"`

	// RoleARN, if set, causes vop to call sts:AssumeRole after fetching the
	// source profile's credentials. Requires SourceProfile to be set.
	RoleARN string `json:"role_arn,omitempty"`

	// SourceProfile is the name of another vop profile whose credentials are
	// used to call sts:AssumeRole. Required when RoleARN is set.
	SourceProfile string `json:"source_profile,omitempty"`

	// RoleSessionName is passed to sts:AssumeRole as the session name.
	// Defaults to "vop" when empty.
	RoleSessionName string `json:"role_session_name,omitempty"`

	// ExternalID is an optional ExternalId condition value required by some
	// cross-account IAM trust policies.
	ExternalID string `json:"external_id,omitempty"`
}

// FieldName returns the 1Password field label for a given base name.
// It checks FieldMap first for an explicit override, then falls back to
// prepending FieldPrefix (if set), and finally returns the bare base name.
func (p *Profile) FieldName(base string) string {
	if p.FieldMap != nil {
		if mapped, ok := p.FieldMap[base]; ok {
			return mapped
		}
	}
	if p.FieldPrefix == "" {
		return base
	}
	return p.FieldPrefix + base
}

// UsesSDK returns true if this profile is configured with a service account
// token, meaning the 1Password SDK should be used instead of the op CLI.
func (p *Profile) UsesSDK() bool {
	return p.ServiceAccountToken != ""
}

// IsAssumedRole returns true if this profile is configured to assume a role
// via sts:AssumeRole using the credentials of another (source) profile.
func (p *Profile) IsAssumedRole() bool {
	return p.RoleARN != ""
}

// UsesCredentialsCommand returns true if base AWS keys should be fetched
// by executing an external command (e.g. `pass aws/prod`).
func (p *Profile) UsesCredentialsCommand() bool {
	return p.CredentialsCommand != ""
}

// UsesAWSCredentialsFile returns true if base AWS keys should be read from
// (and rotated into) the shared credentials file instead of 1Password.
func (p *Profile) UsesAWSCredentialsFile() bool {
	return p.AWSCredentialsProfile != "" && !p.UsesCredentialsCommand()
}

// UsesOnePassword returns true if any 1Password interaction is required
// for this profile: either the base AWS keys or the TOTP live in 1P.
func (p *Profile) UsesOnePassword() bool {
	if p.UsesCredentialsCommand() {
		// Base keys come from the command. 1P is only needed if the TOTP
		// is also in 1P.
		return p.MFATOTPItem != "" && p.MFATOTPCommand == ""
	}
	if !p.UsesAWSCredentialsFile() {
		return true // base keys are in 1P
	}
	if p.MFATOTPItem != "" && p.MFATOTPCommand == "" {
		return true // TOTP is in 1P
	}
	return false
}

// Config is the top-level configuration structure.
type Config struct {
	Profiles map[string]*Profile `json:"profiles"`
}

// DefaultConfigDir returns the default config directory path.
func DefaultConfigDir() string {
	if d := os.Getenv("VOP_CONFIG_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "vop")
}

// DefaultConfigFile returns the default config file path.
func DefaultConfigFile() string {
	if f := os.Getenv("VOP_CONFIG"); f != "" {
		return f
	}
	return filepath.Join(DefaultConfigDir(), "profiles.json")
}

// Load reads and parses the config file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid JSON in %s: %w", path, err)
	}
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]*Profile)
	}
	return &cfg, nil
}

// Save writes the config to disk with proper permissions.
func Save(path string, cfg *Config) error {
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]*Profile)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0600)
}

// EnsureConfigFile creates the config directory and an empty config file
// if they don't exist. Returns true if a new file was created.
func EnsureConfigFile(path string) (bool, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return false, err
	}
	if _, err := os.Stat(path); err == nil {
		return false, nil // already exists
	}
	cfg := &Config{Profiles: make(map[string]*Profile)}
	return true, Save(path, cfg)
}

// ProfileExists checks if a profile name exists in the config.
func (c *Config) ProfileExists(name string) bool {
	_, ok := c.Profiles[name]
	return ok
}

// ProfileNames returns a sorted list of profile names.
func (c *Config) ProfileNames() []string {
	names := make([]string, 0, len(c.Profiles))
	for k := range c.Profiles {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// ClosestProfile returns the profile name most similar to the given input,
// along with the edit distance. Returns ("", -1) if there are no profiles.
func (c *Config) ClosestProfile(input string) (string, int) {
	names := c.ProfileNames()
	if len(names) == 0 {
		return "", -1
	}

	best := names[0]
	bestDist := levenshtein(input, best)
	for _, n := range names[1:] {
		d := levenshtein(input, n)
		if d < bestDist {
			best = n
			bestDist = d
		}
	}
	return best, bestDist
}

// levenshtein computes the edit distance between two strings.
func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	// Use single-row DP to save memory.
	prev := make([]int, lb+1)
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr := make([]int, lb+1)
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, min(prev[j]+1, prev[j-1]+cost))
		}
		prev = curr
	}
	return prev[lb]
}

// SetProfile adds or updates a profile.
func (c *Config) SetProfile(name string, p *Profile) {
	c.Profiles[name] = p
}

// DeleteProfile removes a profile by name.
func (c *Config) DeleteProfile(name string) {
	delete(c.Profiles, name)
}

// DefaultAgentInstructions is the default read-only policy applied when no
// custom agent_policy is configured on a profile.
const DefaultAgentInstructions = "IMPORTANT: These AWS credentials must be used for READ-ONLY " +
	"operations only. Do NOT create, modify, or delete any AWS resources unless you have " +
	"been explicitly granted permission by the user for a specific operation. If you need " +
	"to perform a write operation, ask the user for permission first and explain what you " +
	"intend to do."

// AgentInstructions returns the agent policy instruction text for this profile.
// Returns the custom policy if set, otherwise the default read-only instructions.
func (p *Profile) AgentInstructions() string {
	if p.AgentPolicy != "" {
		return p.AgentPolicy
	}
	return DefaultAgentInstructions
}
