package creds

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadAWSCredentialsFile(t *testing.T) {
	body := `# top comment
[default]
aws_access_key_id = AKIA_DEFAULT
aws_secret_access_key = secret_default

[work]
; comment
aws_access_key_id = AKIA_WORK
aws_secret_access_key = secret_work
aws_session_token = token_work
region = us-west-2

[legacy_key]
aws_access_key_id=AKIA_LEGACY
aws_secret_access_key=secret_legacy
`
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials")
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		profile     string
		wantAK      string
		wantSK      string
		wantToken   string
		expectError bool
	}{
		{"default", "AKIA_DEFAULT", "secret_default", "", false},
		{"work", "AKIA_WORK", "secret_work", "token_work", false},
		{"legacy_key", "AKIA_LEGACY", "secret_legacy", "", false},
		{"missing", "", "", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.profile, func(t *testing.T) {
			ak, sk, tok, err := ReadAWSCredentialsFile(path, tc.profile)
			if tc.expectError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ak != tc.wantAK {
				t.Errorf("access key: got %q, want %q", ak, tc.wantAK)
			}
			if sk != tc.wantSK {
				t.Errorf("secret key: got %q, want %q", sk, tc.wantSK)
			}
			if tok != tc.wantToken {
				t.Errorf("session token: got %q, want %q", tok, tc.wantToken)
			}
		})
	}
}

func TestReadAWSCredentialsFile_ProfileFromConfigSection(t *testing.T) {
	body := `[profile work]
aws_access_key_id = AKIA_WORK_CONFIG
aws_secret_access_key = secret_work_config
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	ak, sk, _, err := ReadAWSCredentialsFile(path, "work")
	if err != nil {
		t.Fatal(err)
	}
	if ak != "AKIA_WORK_CONFIG" || sk != "secret_work_config" {
		t.Errorf("got %q/%q", ak, sk)
	}
}

func TestUpdateSharedCredentialsBody_ReplaceExisting(t *testing.T) {
	body := `[default]
aws_access_key_id = OLD_AK
aws_secret_access_key = OLD_SK

[other]
aws_access_key_id = OTHER_AK
aws_secret_access_key = OTHER_SK
region = us-east-1
`
	out, err := updateSharedCredentialsBody(body, "default", "NEW_AK", "NEW_SK")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "aws_access_key_id = NEW_AK") {
		t.Errorf("expected new AK in output, got:\n%s", out)
	}
	if !strings.Contains(out, "aws_secret_access_key = NEW_SK") {
		t.Errorf("expected new SK in output, got:\n%s", out)
	}
	if strings.Contains(out, "OLD_AK") || strings.Contains(out, "OLD_SK") {
		t.Errorf("old values should be gone, got:\n%s", out)
	}
	// Other profile untouched
	if !strings.Contains(out, "OTHER_AK") || !strings.Contains(out, "region = us-east-1") {
		t.Errorf("other profile should be untouched, got:\n%s", out)
	}
}

func TestUpdateSharedCredentialsBody_StaleSessionTokenBlanked(t *testing.T) {
	body := `[default]
aws_access_key_id = OLD_AK
aws_secret_access_key = OLD_SK
aws_session_token = STALE_TOKEN
`
	out, err := updateSharedCredentialsBody(body, "default", "NEW_AK", "NEW_SK")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "STALE_TOKEN") {
		t.Errorf("stale session token should be removed, got:\n%s", out)
	}
}

func TestUpdateSharedCredentialsBody_CreatesMissingSection(t *testing.T) {
	body := `[default]
aws_access_key_id = X
aws_secret_access_key = Y
`
	out, err := updateSharedCredentialsBody(body, "newprofile", "NEW_AK", "NEW_SK")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "[newprofile]") {
		t.Errorf("expected new section header, got:\n%s", out)
	}
	if !strings.Contains(out, "aws_access_key_id = NEW_AK") {
		t.Errorf("expected new AK, got:\n%s", out)
	}
	// Existing profile untouched
	if !strings.Contains(out, "aws_access_key_id = X") {
		t.Errorf("original section should be preserved, got:\n%s", out)
	}
}

func TestUpdateSharedCredentialsBody_EmptyFile(t *testing.T) {
	out, err := updateSharedCredentialsBody("", "default", "AK", "SK")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "[default]") ||
		!strings.Contains(out, "aws_access_key_id = AK") ||
		!strings.Contains(out, "aws_secret_access_key = SK") {
		t.Errorf("unexpected output for empty file:\n%s", out)
	}
}

func TestLookupAWSMFASerial(t *testing.T) {
	dir := t.TempDir()
	// User's actual layout: mfa_serial lives in ~/.aws/credentials
	// alongside the keys, using [profile <name>] section headers.
	credsPath := filepath.Join(dir, "credentials")
	credsBody := `[default]
aws_access_key_id = X
aws_secret_access_key = Y

[profile ednition]
aws_access_key_id = AKIA_ED
aws_secret_access_key = secret_ed
mfa_serial = arn:aws:iam::775773658417:mfa/daniel
region = us-east-1
`
	if err := os.WriteFile(credsPath, []byte(credsBody), 0600); err != nil {
		t.Fatal(err)
	}
	// No config file present — should still resolve from credentials.
	serial, err := LookupAWSMFASerial(filepath.Join(dir, "config-does-not-exist"), credsPath, "ednition")
	if err != nil {
		t.Fatal(err)
	}
	if serial != "arn:aws:iam::775773658417:mfa/daniel" {
		t.Errorf("unexpected serial: %q", serial)
	}
}

func TestLookupAWSMFASerial_PrefersConfigFile(t *testing.T) {
	dir := t.TempDir()
	credsPath := filepath.Join(dir, "credentials")
	configPath := filepath.Join(dir, "config")
	credsBody := `[work]
aws_access_key_id = AK
aws_secret_access_key = SK
mfa_serial = arn:aws:iam::111111111111:mfa/from-creds
`
	configBody := `[profile work]
mfa_serial = arn:aws:iam::222222222222:mfa/from-config
region = us-east-1
`
	if err := os.WriteFile(credsPath, []byte(credsBody), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(configBody), 0600); err != nil {
		t.Fatal(err)
	}
	serial, err := LookupAWSMFASerial(configPath, credsPath, "work")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(serial, "from-config") {
		t.Errorf("expected config file to win, got %q", serial)
	}
}

func TestLookupAWSMFASerial_MissingFilesReturnEmpty(t *testing.T) {
	dir := t.TempDir()
	serial, err := LookupAWSMFASerial(
		filepath.Join(dir, "no-config"),
		filepath.Join(dir, "no-credentials"),
		"whatever",
	)
	if err != nil {
		t.Fatalf("missing files should not error, got: %v", err)
	}
	if serial != "" {
		t.Errorf("expected empty, got %q", serial)
	}
}

func TestWriteAWSCredentialsFileKeys_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials")
	initial := `[default]
aws_access_key_id = OLD
aws_secret_access_key = OLD_SECRET
`
	if err := os.WriteFile(path, []byte(initial), 0600); err != nil {
		t.Fatal(err)
	}

	if err := WriteAWSCredentialsFileKeys(path, "default", "NEW_AK", "NEW_SK"); err != nil {
		t.Fatal(err)
	}

	ak, sk, _, err := ReadAWSCredentialsFile(path, "default")
	if err != nil {
		t.Fatal(err)
	}
	if ak != "NEW_AK" || sk != "NEW_SK" {
		t.Errorf("round-trip mismatch: got %q/%q", ak, sk)
	}

	// Permissions should stay 0600.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected 0600 perms, got %o", info.Mode().Perm())
	}
}
