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
	_, err := c.Cmd.Run("account", "get", "--account", account)
	if err == nil {
		return nil
	}
	return c.Cmd.RunPassthrough("signin", "--account", account)
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
