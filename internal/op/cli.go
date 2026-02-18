package op

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Commander is an interface for executing op CLI commands, allowing test mocking.
type Commander interface {
	Run(args ...string) (string, error)
	RunPassthrough(args ...string) error
	// RunInteractive runs a command that may prompt on the TTY (e.g. password)
	// while capturing stdout. Stderr is passed through to the terminal.
	RunInteractive(args ...string) (string, error)
}

// ExecCommander implements Commander using the real op binary.
type ExecCommander struct{}

func (c *ExecCommander) Run(args ...string) (string, error) {
	cmd := exec.Command("op", args...)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func (c *ExecCommander) RunPassthrough(args ...string) error {
	cmd := exec.Command("op", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (c *ExecCommander) RunInteractive(args ...string) (string, error) {
	cmd := exec.Command("op", args...)
	// Connect stdin so op can read the password from the terminal.
	cmd.Stdin = os.Stdin
	// Stderr goes to the terminal so the user sees prompts/errors.
	cmd.Stderr = os.Stderr
	// Capture stdout where op writes the session token.
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// CLIClient implements Client using the 1Password CLI binary.
type CLIClient struct {
	Cmd Commander
}

// NewCLI creates a new CLIClient with the real op CLI.
func NewCLI() *CLIClient {
	return &CLIClient{Cmd: &ExecCommander{}}
}

func (c *CLIClient) IsInstalled() bool {
	_, err := exec.LookPath("op")
	return err == nil
}

func (c *CLIClient) EnsureSignedIn(account string) error {
	// Check if we already have an active session for this account.
	_, err := c.Cmd.Run("account", "get", "--account", account)
	if err == nil {
		return nil
	}

	// Try standalone signin with --raw first. This prompts for the master
	// password on the terminal and returns just the session token on stdout.
	// This is the correct path when there is no 1Password desktop app.
	out, rawErr := c.Cmd.RunInteractive("signin", "--account", account, "--raw")
	if rawErr == nil {
		token := strings.TrimSpace(out)
		if token != "" {
			// Set OP_SESSION_<shorthand> so subsequent op commands are
			// authenticated within this process.
			shorthand := account
			if idx := strings.Index(shorthand, "."); idx > 0 {
				shorthand = shorthand[:idx]
			}
			shorthand = strings.NewReplacer(".", "_", "-", "_").Replace(shorthand)
			os.Setenv("OP_SESSION_"+shorthand, token)
		}
		return nil
	}

	// --raw failed — try desktop app integration (biometric/system auth).
	// In this mode, op communicates with the 1Password app directly and
	// there is no session token to capture.
	if passthroughErr := c.Cmd.RunPassthrough("signin", "--account", account); passthroughErr == nil {
		return nil
	}

	return fmt.Errorf("failed to sign in to 1Password account %q.\n  Make sure the account is added: op account add --address %s\n  Then try again, or use a service account token instead", account, account)
}

func (c *CLIClient) ReadField(account, item, field string) (string, error) {
	out, err := c.Cmd.Run("item", "get", item, "--account", account, "--fields", field, "--reveal")
	if err != nil {
		return "", fmt.Errorf("failed to read field %q from item %q: %w", field, item, err)
	}
	if out == "" {
		return "", fmt.Errorf("empty value for field %q in item %q", field, item)
	}
	return out, nil
}

func (c *CLIClient) GetTOTP(account, item string) (string, error) {
	out, err := c.Cmd.Run("item", "get", item, "--account", account, "--otp")
	if err != nil {
		return "", fmt.Errorf("failed to get TOTP from item %q: %w", item, err)
	}
	return out, nil
}

func (c *CLIClient) EditItem(account, item string, assignments ...string) error {
	args := []string{"item", "edit", item, "--account", account}
	args = append(args, assignments...)
	_, err := c.Cmd.Run(args...)
	return err
}

func (c *CLIClient) ListAccounts() ([]OPAccount, error) {
	out, err := c.Cmd.Run("account", "list", "--format=json")
	if err != nil {
		return nil, err
	}
	var accounts []OPAccount
	if err := json.Unmarshal([]byte(out), &accounts); err != nil {
		return nil, err
	}
	return accounts, nil
}

func (c *CLIClient) ListVaults(account string) ([]OPVault, error) {
	out, err := c.Cmd.Run("vault", "list", "--account", account, "--format=json")
	if err != nil {
		return nil, err
	}
	var vaults []OPVault
	if err := json.Unmarshal([]byte(out), &vaults); err != nil {
		return nil, err
	}
	return vaults, nil
}

func (c *CLIClient) CreateItem(account, vault, category, title, tags string, assignments ...string) error {
	args := []string{
		"item", "create",
		"--account", account,
		"--vault", vault,
		"--category", category,
		"--title", title,
	}
	if tags != "" {
		args = append(args, "--tags", tags)
	}
	args = append(args, assignments...)
	_, err := c.Cmd.Run(args...)
	return err
}

func (c *CLIClient) ListItems(account, vault string) ([]OPItem, error) {
	args := []string{"item", "list", "--account", account, "--format=json"}
	if vault != "" {
		args = append(args, "--vault", vault)
	}
	out, err := c.Cmd.Run(args...)
	if err != nil {
		return nil, err
	}
	var items []OPItem
	if err := json.Unmarshal([]byte(out), &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c *CLIClient) ListFields(account, item string) ([]OPField, error) {
	out, err := c.Cmd.Run("item", "get", item, "--account", account, "--format=json")
	if err != nil {
		return nil, fmt.Errorf("failed to get item %q: %w", item, err)
	}

	// op item get --format=json returns an object with a "fields" array.
	// Each field has "label", "type", and possibly "value".
	var raw struct {
		Fields []struct {
			Label   string `json:"label"`
			Type    string `json:"type"`
			Purpose string `json:"purpose"`
		} `json:"fields"`
	}
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse item JSON: %w", err)
	}

	var fields []OPField
	for _, f := range raw.Fields {
		// Skip internal/purpose fields (username, password built-ins)
		// that have no user-assigned label, but include them if they
		// have a label the user might want to map.
		if f.Label == "" {
			continue
		}
		fields = append(fields, OPField{
			Label: f.Label,
			Type:  f.Type,
		})
	}
	return fields, nil
}
