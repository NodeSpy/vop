// Package creds handles AWS credential fetching, STS session tokens,
// and tmpfs-backed credential file management.
package creds

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/NodeSpy/vop/internal/awsclient"
	"github.com/NodeSpy/vop/internal/config"
	"github.com/NodeSpy/vop/internal/op"
	"github.com/NodeSpy/vop/internal/ui"
)

// AWSCredentials holds a set of AWS credentials.
type AWSCredentials struct {
	AccessKeyID     string `json:"AccessKeyId"`
	SecretAccessKey string `json:"SecretAccessKey"`
	SessionToken    string `json:"SessionToken,omitempty"`
	Expiration      string `json:"Expiration,omitempty"`
}

// RuntimeDir returns the secure, per-user directory for credential files.
// On Linux this is typically a tmpfs mount at /run/user/<uid>.
// On macOS it falls back to a per-user directory under os.TempDir().
func RuntimeDir() string {
	xdg := os.Getenv("XDG_RUNTIME_DIR")
	if xdg == "" {
		xdg = runtimeDirFallback()
	}
	return filepath.Join(xdg, "vop")
}

// Fetch retrieves AWS credentials for a profile from 1Password,
// handling MFA/STS and role assumption if configured.
//
// cfg and clientFor are required when the profile has role_arn set (assumed-role
// profiles). They are used to fetch the source profile's credentials before
// calling sts:AssumeRole. Pass nil for both when calling recursively for a
// source profile (which must not itself be an assumed-role profile).
func Fetch(profile *config.Profile, profileName string, opClient op.Client, cfg *config.Config, clientFor func(*config.Profile) (op.Client, error)) (*AWSCredentials, error) {
	if err := opClient.EnsureSignedIn(profile.OPAccount); err != nil {
		return nil, fmt.Errorf("failed to sign into 1Password account %s: %w", profile.OPAccount, err)
	}

	ui.Info("Fetching credentials from: %s", profile.OPItem)

	accessKey, err := opClient.ReadField(profile.OPAccount, profile.OPItem, profile.FieldName("access key id"))
	if err != nil {
		return nil, err
	}
	secretKey, err := opClient.ReadField(profile.OPAccount, profile.OPItem, profile.FieldName("secret access key"))
	if err != nil {
		return nil, err
	}

	creds := &AWSCredentials{
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
	}

	// Handle MFA if configured
	if profile.MFATOTPItem != "" {
		ui.Info("MFA configured — fetching TOTP from: %s", profile.MFATOTPItem)

		totp, err := opClient.GetTOTP(profile.OPAccount, profile.MFATOTPItem)
		if err != nil {
			return nil, err
		}

		serialNumber, _ := opClient.ReadField(profile.OPAccount, profile.OPItem, profile.FieldName("mfa serial"))
		if serialNumber == "" {
			ui.Info("Looking up MFA serial via AWS IAM...")
			serialNumber, err = awsclient.ListMFADevices(
				context.Background(),
				creds.AccessKeyID, creds.SecretAccessKey,
				profile.IAMUsername,
			)
			if err != nil {
				return nil, err
			}
		}

		ui.Info("Requesting STS session token...")
		stsCreds, err := awsclient.GetSessionToken(
			context.Background(),
			creds.AccessKeyID, creds.SecretAccessKey,
			serialNumber, totp,
		)
		if err != nil {
			return nil, fmt.Errorf("STS get-session-token failed (TOTP may have expired): %w", err)
		}

		ui.Info("Session token obtained (expires: %s)", stsCreds.Expiration)
		return &AWSCredentials{
			AccessKeyID:     stsCreds.AccessKeyID,
			SecretAccessKey: stsCreds.SecretAccessKey,
			SessionToken:    stsCreds.SessionToken,
			Expiration:      stsCreds.Expiration,
		}, nil
	}

	// Handle role assumption if configured.
	if profile.IsAssumedRole() {
		if cfg == nil {
			return nil, fmt.Errorf("profile %q requires role assumption but no config was provided", profileName)
		}
		srcProfile, ok := cfg.Profiles[profile.SourceProfile]
		if !ok {
			return nil, fmt.Errorf("source_profile %q not found (referenced by profile %q)", profile.SourceProfile, profileName)
		}
		if srcProfile.IsAssumedRole() {
			return nil, fmt.Errorf("chained role assumption is not supported: source_profile %q is itself an assumed-role profile", profile.SourceProfile)
		}
		srcClient, err := clientFor(srcProfile)
		if err != nil {
			return nil, fmt.Errorf("failed to get client for source profile %q: %w", profile.SourceProfile, err)
		}
		// Fetch source credentials; pass nil cfg/clientFor — source cannot chain.
		srcCreds, err := Fetch(srcProfile, profile.SourceProfile, srcClient, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch source credentials for role assumption: %w", err)
		}

		ui.Info("Assuming role: %s", profile.RoleARN)
		var assumed *awsclient.SessionCredentials
		if srcCreds.SessionToken != "" {
			assumed, err = awsclient.AssumeRoleWithSession(
				context.Background(),
				srcCreds.AccessKeyID, srcCreds.SecretAccessKey, srcCreds.SessionToken,
				profile.RoleARN, profile.RoleSessionName, profile.ExternalID,
			)
		} else {
			assumed, err = awsclient.AssumeRole(
				context.Background(),
				srcCreds.AccessKeyID, srcCreds.SecretAccessKey,
				profile.RoleARN, profile.RoleSessionName, profile.ExternalID,
			)
		}
		if err != nil {
			return nil, err
		}
		ui.Info("Role assumed (expires: %s)", assumed.Expiration)
		return &AWSCredentials{
			AccessKeyID:     assumed.AccessKeyID,
			SecretAccessKey: assumed.SecretAccessKey,
			SessionToken:    assumed.SessionToken,
			Expiration:      assumed.Expiration,
		}, nil
	}

	return creds, nil
}

// ExportToEnv sets the AWS credential environment variables.
func ExportToEnv(creds *AWSCredentials, profileName string) {
	os.Setenv("AWS_ACCESS_KEY_ID", creds.AccessKeyID)
	os.Setenv("AWS_SECRET_ACCESS_KEY", creds.SecretAccessKey)
	if creds.SessionToken != "" {
		os.Setenv("AWS_SESSION_TOKEN", creds.SessionToken)
	} else {
		os.Unsetenv("AWS_SESSION_TOKEN")
	}
	os.Setenv("VOP_PROFILE", profileName)
	os.Setenv("VAULTED_ENV", profileName) // backward compat
	os.Unsetenv("AWS_DEFAULT_REGION")
}

// UnsetCredEnvVars removes the AWS credential environment variables so that
// the AWS SDK falls through to the shared credentials file. Call this in
// long-lived shell sessions where credentials are refreshed on disk via
// WriteFiles.
func UnsetCredEnvVars() {
	os.Unsetenv("AWS_ACCESS_KEY_ID")
	os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	os.Unsetenv("AWS_SESSION_TOKEN")
}

// WriteFiles writes credential files to the runtime directory.
// Returns the paths to the AWS credentials file and JSON file.
func WriteFiles(creds *AWSCredentials, profileName string) (credFile, jsonFile string, err error) {
	dir := RuntimeDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", "", fmt.Errorf("failed to create runtime dir: %w", err)
	}

	base := filepath.Join(dir, profileName)
	credFile = base + ".credentials"
	jsonFile = base + ".json"

	// AWS shared credentials format
	var sb strings.Builder
	sb.WriteString("[default]\n")
	sb.WriteString(fmt.Sprintf("aws_access_key_id = %s\n", creds.AccessKeyID))
	sb.WriteString(fmt.Sprintf("aws_secret_access_key = %s\n", creds.SecretAccessKey))
	if creds.SessionToken != "" {
		sb.WriteString(fmt.Sprintf("aws_session_token = %s\n", creds.SessionToken))
	}
	if err := os.WriteFile(credFile, []byte(sb.String()), 0600); err != nil {
		return "", "", err
	}

	// JSON format (credential_process compatible)
	jsonData := map[string]any{
		"Version":         1,
		"AccessKeyId":     creds.AccessKeyID,
		"SecretAccessKey": creds.SecretAccessKey,
		"Profile":         profileName,
	}
	if creds.SessionToken != "" {
		jsonData["SessionToken"] = creds.SessionToken
	}
	if creds.Expiration != "" {
		jsonData["Expiration"] = creds.Expiration
	}
	jBytes, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		return "", "", err
	}
	jBytes = append(jBytes, '\n')
	if err := os.WriteFile(jsonFile, jBytes, 0600); err != nil {
		return "", "", err
	}

	os.Setenv("AWS_SHARED_CREDENTIALS_FILE", credFile)
	os.Setenv("VOP_CRED_FILE", jsonFile)

	return credFile, jsonFile, nil
}

// CleanupFiles removes credential files for a profile.
func CleanupFiles(profileName string) {
	dir := RuntimeDir()
	base := filepath.Join(dir, profileName)
	os.Remove(base + ".credentials")
	os.Remove(base + ".json")
	os.Remove(dir) // only succeeds if empty
}

// ReadJSONFile reads a credential JSON file.
func ReadJSONFile(path string) (*AWSCredentials, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, "", err
	}
	creds := &AWSCredentials{
		AccessKeyID:     raw["AccessKeyId"].(string),
		SecretAccessKey: raw["SecretAccessKey"].(string),
	}
	if st, ok := raw["SessionToken"].(string); ok {
		creds.SessionToken = st
	}
	if exp, ok := raw["Expiration"].(string); ok {
		creds.Expiration = exp
	}
	profile := ""
	if p, ok := raw["Profile"].(string); ok {
		profile = p
	}
	return creds, profile, nil
}
