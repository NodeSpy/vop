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
	OPAccount   string `json:"op_account"`
	OPItem      string `json:"op_item"`
	OPVault     string `json:"op_vault,omitempty"`
	Description string `json:"description,omitempty"`
	MFATOTPItem string `json:"mfa_totp_item,omitempty"`
	IAMUsername string `json:"iam_username,omitempty"`

	// ServiceAccountToken enables the 1Password SDK backend instead of the
	// op CLI. When set, OPVault is required and OPAccount is ignored.
	ServiceAccountToken string `json:"service_account_token,omitempty"`
}

// UsesSDK returns true if this profile is configured with a service account
// token, meaning the 1Password SDK should be used instead of the op CLI.
func (p *Profile) UsesSDK() bool {
	return p.ServiceAccountToken != ""
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

// SetProfile adds or updates a profile.
func (c *Config) SetProfile(name string, p *Profile) {
	c.Profiles[name] = p
}

// DeleteProfile removes a profile by name.
func (c *Config) DeleteProfile(name string) {
	delete(c.Profiles, name)
}
