package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/NodeSpy/vop/internal/config"
	"github.com/NodeSpy/vop/internal/creds"
	"github.com/NodeSpy/vop/internal/op"
)

// setupTestConfig creates a temp config file and configures the cmd package to use it.
// Returns the config path and a cleanup function.
func setupTestConfig(t *testing.T, profiles map[string]*config.Profile) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")

	c := &config.Config{Profiles: profiles}
	if c.Profiles == nil {
		c.Profiles = make(map[string]*config.Profile)
	}
	if err := config.Save(path, c); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	// Set cfgFile package var so loadConfig uses our test file
	cfgFile = path
	cfg = nil // reset cached config
	t.Cleanup(func() {
		cfgFile = ""
		cfg = nil
	})

	return path
}

func TestNewRootCmd(t *testing.T) {
	root := NewRootCmd()
	if root == nil {
		t.Fatal("NewRootCmd returned nil")
	}
	if root.Use != "vop" {
		t.Errorf("expected Use='vop', got %q", root.Use)
	}
}

func TestNewRootCmd_HasAllSubcommands(t *testing.T) {
	root := NewRootCmd()

	// Core commands that are always present.
	expected := []string{
		"ls", "shell", "exec", "add", "edit", "rm",
		"show", "dump", "rotate", "test", "migrate", "check",
		"version", "refresh", "cred-process", "cat", "profile", "skill",
	}

	commands := make(map[string]bool)
	for _, cmd := range root.Commands() {
		commands[cmd.Name()] = true
	}

	for _, name := range expected {
		if !commands[name] {
			t.Errorf("missing subcommand: %s", name)
		}
	}

	// "update" is conditionally compiled via build tag "noupdate".
	// In the default build it should be present.
	if newUpdateCmd() != nil && !commands["update"] {
		t.Errorf("newUpdateCmd() returned non-nil but 'update' subcommand not registered")
	}
	if newUpdateCmd() == nil && commands["update"] {
		t.Errorf("newUpdateCmd() returned nil but 'update' subcommand is registered")
	}
}

func TestNewRootCmd_LsAlias(t *testing.T) {
	root := NewRootCmd()
	for _, cmd := range root.Commands() {
		if cmd.Name() == "ls" {
			found := false
			for _, alias := range cmd.Aliases {
				if alias == "list" {
					found = true
				}
			}
			if !found {
				t.Error("ls command missing 'list' alias")
			}
			return
		}
	}
	t.Fatal("ls command not found")
}

func TestConfigFilePath_Default(t *testing.T) {
	old := cfgFile
	cfgFile = ""
	defer func() { cfgFile = old }()

	path := configFilePath()
	if path == "" {
		t.Fatal("configFilePath returned empty string")
	}
	if filepath.Base(path) != "profiles.json" {
		t.Errorf("expected path to end with 'profiles.json', got %q", path)
	}
}

func TestConfigFilePath_Override(t *testing.T) {
	old := cfgFile
	cfgFile = "/custom/path/config.json"
	defer func() { cfgFile = old }()

	if got := configFilePath(); got != "/custom/path/config.json" {
		t.Errorf("expected '/custom/path/config.json', got %q", got)
	}
}

func TestLoadConfig(t *testing.T) {
	setupTestConfig(t, map[string]*config.Profile{
		"prod": {OPAccount: "my.1password.com", OPItem: "AWS - prod"},
	})

	c, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}
	if !c.ProfileExists("prod") {
		t.Error("expected profile 'prod' to exist")
	}
}

func TestLoadConfig_NotFound(t *testing.T) {
	cfgFile = "/nonexistent/path/profiles.json"
	cfg = nil
	defer func() {
		cfgFile = ""
		cfg = nil
	}()

	_, err := loadConfig()
	if err == nil {
		t.Fatal("expected error for nonexistent config, got nil")
	}
}

func TestLoadConfig_Cached(t *testing.T) {
	setupTestConfig(t, map[string]*config.Profile{
		"dev": {OPAccount: "acct", OPItem: "item"},
	})

	c1, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}

	c2, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}

	// Should return the same pointer (cached)
	if c1 != c2 {
		t.Error("expected loadConfig to return cached config")
	}
}

func TestEnsureConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "profiles.json")

	cfgFile = path
	cfg = nil
	defer func() {
		cfgFile = ""
		cfg = nil
	}()

	c, err := ensureConfig()
	if err != nil {
		t.Fatalf("ensureConfig failed: %v", err)
	}
	if c == nil {
		t.Fatal("ensureConfig returned nil config")
	}

	// File should now exist
	if _, err := os.Stat(path); err != nil {
		t.Errorf("config file not created: %v", err)
	}
}

func TestSaveConfig(t *testing.T) {
	path := setupTestConfig(t, nil)

	c := &config.Config{Profiles: map[string]*config.Profile{
		"new": {OPAccount: "acct", OPItem: "item"},
	}}

	if err := saveConfig(c); err != nil {
		t.Fatalf("saveConfig failed: %v", err)
	}

	// Verify it was saved
	loaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.ProfileExists("new") {
		t.Error("expected profile 'new' to exist after save")
	}
}

func TestRequireProfile(t *testing.T) {
	c := &config.Config{
		Profiles: map[string]*config.Profile{
			"prod": {OPAccount: "acct", OPItem: "item"},
		},
	}

	p, err := requireProfile(c, "prod")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.OPAccount != "acct" {
		t.Errorf("expected OPAccount 'acct', got %q", p.OPAccount)
	}
}

func TestRequireProfile_Empty(t *testing.T) {
	c := &config.Config{Profiles: map[string]*config.Profile{}}

	_, err := requireProfile(c, "")
	if err == nil {
		t.Fatal("expected error for empty profile name, got nil")
	}
}

func TestRequireProfile_NotFound(t *testing.T) {
	c := &config.Config{Profiles: map[string]*config.Profile{}}

	_, err := requireProfile(c, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent profile, got nil")
	}
}

func TestGetCLIClient(t *testing.T) {
	cliClient = nil
	defer func() { cliClient = nil }()

	client := getCLIClient()
	if client == nil {
		t.Fatal("getCLIClient returned nil")
	}

	// Should return the same client on subsequent calls
	client2 := getCLIClient()
	if client != client2 {
		t.Error("expected getCLIClient to return cached client")
	}
}

// mockClient implements op.Client for testing.
type mockClient struct{}

func (m *mockClient) IsInstalled() bool                                  { return true }
func (m *mockClient) EnsureSignedIn(_ string) error                      { return nil }
func (m *mockClient) ReadField(_, _, _ string) (string, error)           { return "", nil }
func (m *mockClient) GetTOTP(_, _ string) (string, error)                { return "", nil }
func (m *mockClient) EditItem(_ string, _ string, _ ...string) error     { return nil }
func (m *mockClient) ListAccounts() ([]op.OPAccount, error)              { return nil, nil }
func (m *mockClient) ListVaults(_ string) ([]op.OPVault, error)          { return nil, nil }
func (m *mockClient) CreateItem(_, _, _, _, _ string, _ ...string) error { return nil }
func (m *mockClient) ListItems(_, _ string) ([]op.OPItem, error)         { return nil, nil }
func (m *mockClient) ListFields(_, _ string) ([]op.OPField, error)       { return nil, nil }

func TestGetClientForProfile_CLI(t *testing.T) {
	// Inject a mock so the test works even without op installed.
	cliClient = &mockClient{}
	defer func() { cliClient = nil }()

	profile := &config.Profile{
		OPAccount: "my.1password.com",
		OPItem:    "AWS - prod",
	}

	client, err := getClientForProfile(profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("getClientForProfile returned nil")
	}
}

func TestGetClientForProfile_CLI_NotInstalled(t *testing.T) {
	cliClient = nil
	defer func() { cliClient = nil }()

	profile := &config.Profile{
		OPAccount: "my.1password.com",
		OPItem:    "AWS - prod",
	}

	// On machines without op installed, this should return a clear error.
	client, err := getClientForProfile(profile)
	cli := getCLIClient()
	if !cli.IsInstalled() {
		// op not installed — we expect an error
		if err == nil {
			t.Fatal("expected error when op is not installed, got nil")
		}
		if client != nil {
			t.Fatal("expected nil client when op is not installed")
		}
	} else {
		// op is installed — should succeed
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestGetClientForProfile_SDK_InvalidToken(t *testing.T) {
	profile := &config.Profile{
		OPItem:              "AWS - prod",
		OPVault:             "Private",
		ServiceAccountToken: "invalid-token",
	}

	// SDK client creation will fail with an invalid token
	_, err := getClientForProfile(profile)
	if err == nil {
		t.Fatal("expected error for invalid service account token, got nil")
	}
}

func TestCmdLs_WithProfiles(t *testing.T) {
	setupTestConfig(t, map[string]*config.Profile{
		"prod": {OPAccount: "my.1password.com", OPItem: "AWS - prod", Description: "Production"},
		"dev":  {OPAccount: "my.1password.com", OPItem: "AWS - dev"},
	})

	// cmdLs writes to stdout; just verify it doesn't error
	err := cmdLs(nil, nil)
	if err != nil {
		t.Fatalf("cmdLs failed: %v", err)
	}
}

func TestCmdLs_Empty(t *testing.T) {
	setupTestConfig(t, nil)

	err := cmdLs(nil, nil)
	if err != nil {
		t.Fatalf("cmdLs with empty config failed: %v", err)
	}
}

func TestExitCodeFor(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"nil-ish generic error", errors.New("boom"), ExitFailure},
		{"locked source", errors.New("gpg: decryption failed: No secret key"), ExitLocked},
		{"wrapped locked source", fmt.Errorf("fetching TOTP: %w",
			&creds.SourceCommandError{Stderr: "gpg-agent unavailable"}), ExitLocked},
		{"auth cooldown", &creds.CooldownError{Profile: "p", Kind: creds.KindAuth}, ExitCooldown},
		{"rate-limit cooldown", &creds.CooldownError{Profile: "p", Kind: creds.KindRateLimit}, ExitCooldown},
		{"locked cooldown", &creds.CooldownError{Profile: "p", Kind: creds.KindLocked}, ExitLocked},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := exitCodeFor(tc.err); got != tc.want {
				t.Errorf("exitCodeFor(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
	}
}
