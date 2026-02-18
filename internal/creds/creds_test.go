package creds

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRuntimeDir(t *testing.T) {
	// With XDG_RUNTIME_DIR set
	t.Setenv("XDG_RUNTIME_DIR", "/tmp/test-runtime")
	got := RuntimeDir()
	if got != "/tmp/test-runtime/vop" {
		t.Errorf("expected '/tmp/test-runtime/vop', got %q", got)
	}

	// Without XDG_RUNTIME_DIR, falls back to platform-specific path
	t.Setenv("XDG_RUNTIME_DIR", "")
	got = RuntimeDir()
	switch runtime.GOOS {
	case "linux":
		if !strings.HasPrefix(got, "/run/user/") || !strings.HasSuffix(got, "/vop") {
			t.Errorf("expected /run/user/<uid>/vop pattern, got %q", got)
		}
	case "darwin":
		if !strings.Contains(got, "vop-") || !strings.HasSuffix(got, "/vop") {
			t.Errorf("expected <tmpdir>/vop-<uid>/vop pattern, got %q", got)
		}
	default:
		if !strings.HasSuffix(got, "/vop") {
			t.Errorf("expected path ending in /vop, got %q", got)
		}
	}
}

func TestWriteAndCleanupFiles(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)

	creds := &AWSCredentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		SessionToken:    "FwoGZXIvYXdzEBYaDH...",
		Expiration:      "2026-02-18T02:00:00Z",
	}

	credFile, jsonFile, err := WriteFiles(creds, "testprofile")
	if err != nil {
		t.Fatalf("WriteFiles failed: %v", err)
	}

	// Verify credential file path
	expectedDir := filepath.Join(dir, "vop")
	if !strings.HasPrefix(credFile, expectedDir) {
		t.Errorf("expected credFile to be under %q, got %q", expectedDir, credFile)
	}

	// Verify .credentials file content
	credData, err := os.ReadFile(credFile)
	if err != nil {
		t.Fatalf("failed to read credentials file: %v", err)
	}
	credStr := string(credData)
	if !strings.Contains(credStr, "[default]") {
		t.Error("credentials file missing [default] section")
	}
	if !strings.Contains(credStr, "aws_access_key_id = AKIAIOSFODNN7EXAMPLE") {
		t.Error("credentials file missing access key")
	}
	if !strings.Contains(credStr, "aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY") {
		t.Error("credentials file missing secret key")
	}
	if !strings.Contains(credStr, "aws_session_token = FwoGZXIvYXdzEBYaDH...") {
		t.Error("credentials file missing session token")
	}

	// Verify .json file content
	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("failed to read JSON file: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("JSON file is not valid JSON: %v", err)
	}
	if parsed["AccessKeyId"] != "AKIAIOSFODNN7EXAMPLE" {
		t.Errorf("unexpected AccessKeyId: %v", parsed["AccessKeyId"])
	}
	if parsed["Profile"] != "testprofile" {
		t.Errorf("unexpected Profile: %v", parsed["Profile"])
	}
	if parsed["SessionToken"] != "FwoGZXIvYXdzEBYaDH..." {
		t.Errorf("unexpected SessionToken: %v", parsed["SessionToken"])
	}
	// Version should be numeric 1
	if v, ok := parsed["Version"].(float64); !ok || v != 1 {
		t.Errorf("unexpected Version: %v", parsed["Version"])
	}

	// Verify file permissions
	info, err := os.Stat(credFile)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("expected cred file permissions 0600, got %04o", perm)
	}
	info, err = os.Stat(jsonFile)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("expected json file permissions 0600, got %04o", perm)
	}

	// Verify env vars were set
	if os.Getenv("AWS_SHARED_CREDENTIALS_FILE") != credFile {
		t.Errorf("AWS_SHARED_CREDENTIALS_FILE not set correctly")
	}
	if os.Getenv("VOP_CRED_FILE") != jsonFile {
		t.Errorf("VOP_CRED_FILE not set correctly")
	}

	// Cleanup
	CleanupFiles("testprofile")
	if _, err := os.Stat(credFile); !os.IsNotExist(err) {
		t.Error("credentials file not cleaned up")
	}
	if _, err := os.Stat(jsonFile); !os.IsNotExist(err) {
		t.Error("JSON file not cleaned up")
	}
}

func TestWriteFilesNoSessionToken(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)

	creds := &AWSCredentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}

	credFile, jsonFile, err := WriteFiles(creds, "basic")
	if err != nil {
		t.Fatalf("WriteFiles failed: %v", err)
	}

	// Verify .credentials file does NOT contain session token line
	credData, err := os.ReadFile(credFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(credData), "aws_session_token") {
		t.Error("credentials file should not contain session token when empty")
	}

	// Verify .json file does NOT contain SessionToken key
	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatal(err)
	}
	if _, ok := parsed["SessionToken"]; ok {
		t.Error("JSON file should not contain SessionToken when empty")
	}

	CleanupFiles("basic")
}

func TestReadJSONFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")

	data := map[string]any{
		"Version":         1,
		"AccessKeyId":     "AKIAEXAMPLE",
		"SecretAccessKey": "SECRET",
		"SessionToken":    "TOKEN",
		"Profile":         "myprofile",
	}
	jBytes, _ := json.MarshalIndent(data, "", "  ")
	if err := os.WriteFile(path, jBytes, 0600); err != nil {
		t.Fatal(err)
	}

	creds, profile, err := ReadJSONFile(path)
	if err != nil {
		t.Fatalf("ReadJSONFile failed: %v", err)
	}
	if creds.AccessKeyID != "AKIAEXAMPLE" {
		t.Errorf("expected AccessKeyID 'AKIAEXAMPLE', got %q", creds.AccessKeyID)
	}
	if creds.SecretAccessKey != "SECRET" {
		t.Errorf("expected SecretAccessKey 'SECRET', got %q", creds.SecretAccessKey)
	}
	if creds.SessionToken != "TOKEN" {
		t.Errorf("expected SessionToken 'TOKEN', got %q", creds.SessionToken)
	}
	if profile != "myprofile" {
		t.Errorf("expected profile 'myprofile', got %q", profile)
	}
}

func TestReadJSONFile_NoSessionToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")

	data := map[string]any{
		"Version":         1,
		"AccessKeyId":     "AKIAEXAMPLE",
		"SecretAccessKey": "SECRET",
	}
	jBytes, _ := json.MarshalIndent(data, "", "  ")
	if err := os.WriteFile(path, jBytes, 0600); err != nil {
		t.Fatal(err)
	}

	creds, profile, err := ReadJSONFile(path)
	if err != nil {
		t.Fatalf("ReadJSONFile failed: %v", err)
	}
	if creds.SessionToken != "" {
		t.Errorf("expected empty SessionToken, got %q", creds.SessionToken)
	}
	if profile != "" {
		t.Errorf("expected empty profile, got %q", profile)
	}
}

func TestReadJSONFile_NotFound(t *testing.T) {
	_, _, err := ReadJSONFile("/nonexistent/file.json")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

func TestExportToEnv(t *testing.T) {
	// Clean up env after test
	defer func() {
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		os.Unsetenv("AWS_SESSION_TOKEN")
		os.Unsetenv("VOP_PROFILE")
		os.Unsetenv("VAULTED_ENV")
	}()

	creds := &AWSCredentials{
		AccessKeyID:     "AKIA123",
		SecretAccessKey: "SECRET123",
		SessionToken:    "TOKEN123",
	}

	ExportToEnv(creds, "testprofile")

	if got := os.Getenv("AWS_ACCESS_KEY_ID"); got != "AKIA123" {
		t.Errorf("expected AWS_ACCESS_KEY_ID='AKIA123', got %q", got)
	}
	if got := os.Getenv("AWS_SECRET_ACCESS_KEY"); got != "SECRET123" {
		t.Errorf("expected AWS_SECRET_ACCESS_KEY='SECRET123', got %q", got)
	}
	if got := os.Getenv("AWS_SESSION_TOKEN"); got != "TOKEN123" {
		t.Errorf("expected AWS_SESSION_TOKEN='TOKEN123', got %q", got)
	}
	if got := os.Getenv("VOP_PROFILE"); got != "testprofile" {
		t.Errorf("expected VOP_PROFILE='testprofile', got %q", got)
	}
	if got := os.Getenv("VAULTED_ENV"); got != "testprofile" {
		t.Errorf("expected VAULTED_ENV='testprofile', got %q", got)
	}
}

func TestExportToEnv_NoSessionToken(t *testing.T) {
	defer func() {
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		os.Unsetenv("AWS_SESSION_TOKEN")
		os.Unsetenv("VOP_PROFILE")
		os.Unsetenv("VAULTED_ENV")
	}()

	// Set a session token first, then export without one
	os.Setenv("AWS_SESSION_TOKEN", "old-token")

	creds := &AWSCredentials{
		AccessKeyID:     "AKIA456",
		SecretAccessKey: "SECRET456",
	}

	ExportToEnv(creds, "nosts")

	if got := os.Getenv("AWS_SESSION_TOKEN"); got != "" {
		t.Errorf("expected AWS_SESSION_TOKEN to be unset, got %q", got)
	}
}
