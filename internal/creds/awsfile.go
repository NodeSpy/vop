package creds

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DefaultAWSCredentialsFile returns the default path to the shared AWS
// credentials file (~/.aws/credentials).
//
// vop sets AWS_SHARED_CREDENTIALS_FILE itself when running under
// `vop exec`, so we deliberately do NOT consult that env var here — it
// would create a self-referential loop.
func DefaultAWSCredentialsFile() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aws", "credentials")
}

// DefaultAWSConfigFile returns the default path to the AWS config file
// (~/.aws/config), honoring $AWS_CONFIG_FILE if set.
func DefaultAWSConfigFile() string {
	if p := os.Getenv("AWS_CONFIG_FILE"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aws", "config")
}

// ResolveAWSCredentialsPath picks the credentials file path for a profile:
// explicit override wins, otherwise the default ~/.aws/credentials.
func ResolveAWSCredentialsPath(explicit string) string {
	if explicit != "" {
		return explicit
	}
	return DefaultAWSCredentialsFile()
}

// readAWSSectionKeys returns all key/value pairs under [profileName] (or
// [profile profileName] in a config file) from an AWS-style INI file.
// Keys are lowercased. Returns (nil, nil) if the file exists but the
// section doesn't, and (nil, err) if the file itself can't be read.
func readAWSSectionKeys(path, profileName string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	keys := map[string]string{}
	found := false
	inProfile := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			name := strings.TrimSpace(line[1 : len(line)-1])
			// The `config` file uses "profile <name>"; `credentials` uses bare names.
			name = strings.TrimPrefix(name, "profile ")
			inProfile = name == profileName
			if inProfile {
				found = true
			}
			continue
		}
		if !inProfile {
			continue
		}
		key, val, ok := splitKV(line)
		if !ok {
			continue
		}
		keys[strings.ToLower(key)] = val
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return keys, nil
}

// ReadAWSCredentialsFile parses the shared AWS credentials file and returns
// the access key / secret / (optional) session token for the given profile.
func ReadAWSCredentialsFile(path, profileName string) (accessKey, secretKey, sessionToken string, err error) {
	keys, err := readAWSSectionKeys(path, profileName)
	if err != nil {
		return "", "", "", err
	}
	if keys == nil {
		return "", "", "", fmt.Errorf("profile %q not found in %s", profileName, path)
	}
	accessKey = keys["aws_access_key_id"]
	secretKey = keys["aws_secret_access_key"]
	sessionToken = keys["aws_session_token"]
	if accessKey == "" || secretKey == "" {
		return "", "", "", fmt.Errorf("profile %q in %s is missing aws_access_key_id or aws_secret_access_key", profileName, path)
	}
	return accessKey, secretKey, sessionToken, nil
}

// LookupAWSMFASerial returns the mfa_serial for a profile as recorded by
// the AWS CLI. It checks ~/.aws/config first (where `aws configure set
// mfa_serial` writes it), then ~/.aws/credentials as a fallback. Missing
// files or missing keys return ("", nil) — this is a best-effort lookup.
func LookupAWSMFASerial(configPath, credentialsPath, profileName string) (string, error) {
	// Config file is the conventional home for mfa_serial.
	if serial, err := lookupKey(configPath, profileName, "mfa_serial"); err == nil && serial != "" {
		return serial, nil
	}
	// Some setups keep it alongside the keys instead.
	if serial, err := lookupKey(credentialsPath, profileName, "mfa_serial"); err == nil && serial != "" {
		return serial, nil
	}
	return "", nil
}

func lookupKey(path, profileName, key string) (string, error) {
	keys, err := readAWSSectionKeys(path, profileName)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	if keys == nil {
		return "", nil
	}
	return keys[strings.ToLower(key)], nil
}

// WriteAWSCredentialsFileKeys updates aws_access_key_id and
// aws_secret_access_key on an existing profile in the shared credentials
// file, preserving other keys, other profiles, and comments. Creates the
// profile section if it doesn't exist.
//
// The write is atomic (write to temp file, rename over) and preserves the
// original file's permissions (or falls back to 0600).
func WriteAWSCredentialsFileKeys(path, profileName, accessKey, secretKey string) error {
	// Read existing content; treat missing file as empty.
	orig, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", path, err)
	}

	perm := os.FileMode(0600)
	if info, statErr := os.Stat(path); statErr == nil {
		perm = info.Mode().Perm()
	}

	out, err := updateSharedCredentialsBody(string(orig), profileName, accessKey, secretKey)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".vop-creds-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op on success (renamed)

	if _, err := tmp.WriteString(out); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename %s: %w", path, err)
	}
	return nil
}

// updateSharedCredentialsBody performs the actual in-memory edit, so the
// logic is unit-testable without touching disk.
func updateSharedCredentialsBody(body, profileName, accessKey, secretKey string) (string, error) {
	lines := strings.Split(body, "\n")
	// Keep a trailing-newline flag so we don't spuriously add/remove one.
	trailingNewline := strings.HasSuffix(body, "\n")

	sectionStart, sectionEnd := findSection(lines, profileName)
	if sectionStart == -1 {
		// Section does not exist — append it.
		var sb strings.Builder
		sb.WriteString(strings.TrimRight(body, "\n"))
		if sb.Len() > 0 {
			sb.WriteString("\n\n")
		}
		fmt.Fprintf(&sb, "[%s]\n", profileName)
		fmt.Fprintf(&sb, "aws_access_key_id = %s\n", accessKey)
		fmt.Fprintf(&sb, "aws_secret_access_key = %s\n", secretKey)
		return sb.String(), nil
	}

	// Rewrite the section in place: keep other keys, overwrite the two we care about.
	sectionLines := lines[sectionStart:sectionEnd]
	sawAK, sawSK := false, false
	for i, line := range sectionLines {
		if i == 0 {
			continue // header
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") {
			continue
		}
		key, _, ok := splitKV(trimmed)
		if !ok {
			continue
		}
		switch strings.ToLower(key) {
		case "aws_access_key_id":
			sectionLines[i] = "aws_access_key_id = " + accessKey
			sawAK = true
		case "aws_secret_access_key":
			sectionLines[i] = "aws_secret_access_key = " + secretKey
			sawSK = true
		case "aws_session_token":
			// A stored long-lived session token is stale after rotation.
			// Blank it out rather than leaving a misleading value.
			sectionLines[i] = ""
		}
	}
	// Append any missing keys at the end of the section.
	inserted := []string{}
	if !sawAK {
		inserted = append(inserted, "aws_access_key_id = "+accessKey)
	}
	if !sawSK {
		inserted = append(inserted, "aws_secret_access_key = "+secretKey)
	}
	if len(inserted) > 0 {
		// Find the last non-empty line in the section and insert after it.
		insertAt := len(sectionLines)
		for i := len(sectionLines) - 1; i >= 0; i-- {
			if strings.TrimSpace(sectionLines[i]) != "" {
				insertAt = i + 1
				break
			}
		}
		newSection := make([]string, 0, len(sectionLines)+len(inserted))
		newSection = append(newSection, sectionLines[:insertAt]...)
		newSection = append(newSection, inserted...)
		newSection = append(newSection, sectionLines[insertAt:]...)
		sectionLines = newSection
	}

	// Reassemble.
	newLines := make([]string, 0, len(lines)+len(inserted))
	newLines = append(newLines, lines[:sectionStart]...)
	newLines = append(newLines, sectionLines...)
	newLines = append(newLines, lines[sectionEnd:]...)

	out := strings.Join(newLines, "\n")
	if trailingNewline && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return out, nil
}

// findSection returns [start, end) line indices for the named profile.
// Returns (-1, -1) if the section is not present.
func findSection(lines []string, profileName string) (int, int) {
	start := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "[") || !strings.HasSuffix(trimmed, "]") {
			continue
		}
		name := strings.TrimSpace(trimmed[1 : len(trimmed)-1])
		name = strings.TrimPrefix(name, "profile ")
		if start == -1 {
			if name == profileName {
				start = i
			}
			continue
		}
		// Next section header — end of our section.
		return start, i
	}
	if start == -1 {
		return -1, -1
	}
	return start, len(lines)
}

func splitKV(line string) (key, value string, ok bool) {
	idx := strings.Index(line, "=")
	if idx <= 0 {
		return "", "", false
	}
	return strings.TrimSpace(line[:idx]), strings.TrimSpace(line[idx+1:]), true
}
