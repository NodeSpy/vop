package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")

	// Save a config
	cfg := &Config{
		Profiles: map[string]*Profile{
			"prod": {
				OPAccount:   "my.1password.com",
				OPItem:      "AWS - prod",
				Description: "Production",
				IAMUsername: "deploy",
			},
			"dev": {
				OPAccount:   "my.1password.com",
				OPItem:      "AWS - dev",
				MFATOTPItem: "AWS - dev MFA",
			},
		},
	}

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file permissions
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("expected permissions 0600, got %04o", perm)
	}

	// Load it back
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded.Profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(loaded.Profiles))
	}

	prod := loaded.Profiles["prod"]
	if prod.OPAccount != "my.1password.com" {
		t.Errorf("expected OPAccount 'my.1password.com', got %q", prod.OPAccount)
	}
	if prod.OPItem != "AWS - prod" {
		t.Errorf("expected OPItem 'AWS - prod', got %q", prod.OPItem)
	}
	if prod.Description != "Production" {
		t.Errorf("expected Description 'Production', got %q", prod.Description)
	}
	if prod.IAMUsername != "deploy" {
		t.Errorf("expected IAMUsername 'deploy', got %q", prod.IAMUsername)
	}

	dev := loaded.Profiles["dev"]
	if dev.MFATOTPItem != "AWS - dev MFA" {
		t.Errorf("expected MFATOTPItem 'AWS - dev MFA', got %q", dev.MFATOTPItem)
	}
}

func TestLoadNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/profiles.json")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")
	if err := os.WriteFile(path, []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestLoadNullProfiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")
	if err := os.WriteFile(path, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Profiles == nil {
		t.Fatal("expected Profiles map to be initialized, got nil")
	}
}

func TestSaveNilProfiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")

	cfg := &Config{}
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if _, ok := raw["profiles"]; !ok {
		t.Fatal("expected 'profiles' key in output")
	}
}

func TestEnsureConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "profiles.json")

	created, err := EnsureConfigFile(path)
	if err != nil {
		t.Fatalf("EnsureConfigFile failed: %v", err)
	}
	if !created {
		t.Fatal("expected created=true for new file")
	}

	// Calling again should return created=false
	created, err = EnsureConfigFile(path)
	if err != nil {
		t.Fatalf("EnsureConfigFile second call failed: %v", err)
	}
	if created {
		t.Fatal("expected created=false for existing file")
	}

	// Verify the file is loadable
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load after EnsureConfigFile failed: %v", err)
	}
	if len(cfg.Profiles) != 0 {
		t.Fatalf("expected 0 profiles, got %d", len(cfg.Profiles))
	}
}

func TestProfileExists(t *testing.T) {
	cfg := &Config{
		Profiles: map[string]*Profile{
			"prod": {OPAccount: "a", OPItem: "b"},
		},
	}

	if !cfg.ProfileExists("prod") {
		t.Error("expected ProfileExists('prod') to be true")
	}
	if cfg.ProfileExists("dev") {
		t.Error("expected ProfileExists('dev') to be false")
	}
}

func TestProfileNames(t *testing.T) {
	cfg := &Config{
		Profiles: map[string]*Profile{
			"zebra": {},
			"alpha": {},
			"mid":   {},
		},
	}

	names := cfg.ProfileNames()
	expected := []string{"alpha", "mid", "zebra"}
	if len(names) != len(expected) {
		t.Fatalf("expected %d names, got %d", len(expected), len(names))
	}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("expected names[%d]=%q, got %q", i, expected[i], name)
		}
	}
}

func TestSetAndDeleteProfile(t *testing.T) {
	cfg := &Config{Profiles: make(map[string]*Profile)}

	cfg.SetProfile("test", &Profile{OPAccount: "acct", OPItem: "item"})
	if !cfg.ProfileExists("test") {
		t.Fatal("expected profile 'test' to exist after SetProfile")
	}

	cfg.DeleteProfile("test")
	if cfg.ProfileExists("test") {
		t.Fatal("expected profile 'test' to not exist after DeleteProfile")
	}

	// Deleting nonexistent profile should not panic
	cfg.DeleteProfile("nonexistent")
}

func TestDefaultConfigDir(t *testing.T) {
	// Test with VOP_CONFIG_DIR set
	t.Setenv("VOP_CONFIG_DIR", "/custom/path")
	if got := DefaultConfigDir(); got != "/custom/path" {
		t.Errorf("expected '/custom/path', got %q", got)
	}

	// Test without VOP_CONFIG_DIR
	t.Setenv("VOP_CONFIG_DIR", "")
	dir := DefaultConfigDir()
	if dir == "" {
		t.Fatal("DefaultConfigDir returned empty string")
	}
	if filepath.Base(dir) != "vop" {
		t.Errorf("expected dir to end with 'vop', got %q", dir)
	}
}

func TestDefaultConfigFile(t *testing.T) {
	// Test with VOP_CONFIG set
	t.Setenv("VOP_CONFIG", "/custom/config.json")
	if got := DefaultConfigFile(); got != "/custom/config.json" {
		t.Errorf("expected '/custom/config.json', got %q", got)
	}

	// Test without VOP_CONFIG
	t.Setenv("VOP_CONFIG", "")
	file := DefaultConfigFile()
	if filepath.Base(file) != "profiles.json" {
		t.Errorf("expected file to end with 'profiles.json', got %q", file)
	}
}

func TestUsesSDK(t *testing.T) {
	p := &Profile{OPAccount: "acct", OPItem: "item"}
	if p.UsesSDK() {
		t.Error("expected UsesSDK()=false without token")
	}

	p.ServiceAccountToken = "my-token"
	if !p.UsesSDK() {
		t.Error("expected UsesSDK()=true with token")
	}
}

func TestProfileWithNewFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")

	cfg := &Config{
		Profiles: map[string]*Profile{
			"sdk-profile": {
				OPItem:              "AWS - sdk",
				OPVault:             "Private",
				ServiceAccountToken: "ops_token_123",
				Description:         "SDK-based",
			},
			"cli-profile": {
				OPAccount:   "my.1password.com",
				OPItem:      "AWS - cli",
				Description: "CLI-based",
			},
		},
	}

	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	sdk := loaded.Profiles["sdk-profile"]
	if sdk.ServiceAccountToken != "ops_token_123" {
		t.Errorf("expected token 'ops_token_123', got %q", sdk.ServiceAccountToken)
	}
	if sdk.OPVault != "Private" {
		t.Errorf("expected vault 'Private', got %q", sdk.OPVault)
	}
	if !sdk.UsesSDK() {
		t.Error("expected UsesSDK()=true for sdk-profile")
	}

	cli := loaded.Profiles["cli-profile"]
	if cli.UsesSDK() {
		t.Error("expected UsesSDK()=false for cli-profile")
	}
}
