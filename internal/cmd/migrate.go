package cmd

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/NodeSpy/vop/internal/config"
	"github.com/NodeSpy/vop/internal/op"
	"github.com/NodeSpy/vop/internal/ui"
	"github.com/spf13/cobra"
)

func newMigrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate [vault]",
		Short: "Migrate from vaulted to vop",
		Args:  cobra.MaximumNArgs(1),
		RunE:  cmdMigrate,
	}
}

type vaultedDump struct {
	AWSKey *struct {
		ID     string `json:"id"`
		Secret string `json:"secret"`
		MFA    string `json:"mfa"`
		Region string `json:"region"`
	} `json:"aws_key"`
	Vars map[string]string `json:"vars"`
}

func cmdMigrate(_ *cobra.Command, args []string) error {
	if _, err := exec.LookPath("vaulted"); err != nil {
		return fmt.Errorf("'vaulted' is required but not found in PATH")
	}

	c, err := ensureConfig()
	if err != nil {
		return err
	}

	client := getCLIClient()

	// List available vaults
	out, err := exec.Command("vaulted", "ls").Output()
	if err != nil {
		return fmt.Errorf("failed to list vaulted environments: %w", err)
	}
	vaultNames := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(vaultNames) == 0 || (len(vaultNames) == 1 && vaultNames[0] == "") {
		return fmt.Errorf("no vaulted environments found")
	}

	vaultName := ""
	if len(args) > 0 {
		vaultName = args[0]
	}

	var toMigrate []string

	if vaultName == "" {
		fmt.Printf("\n%s  Vaulted environments available to migrate:%s\n\n", ui.Bold, ui.Reset)
		for _, v := range vaultNames {
			if c.ProfileExists(v) {
				fmt.Printf("  %s%-25s%s %s(already in vop)%s\n", ui.Dim, v, ui.Reset, ui.Dim, ui.Reset)
			} else {
				fmt.Printf("  %s%-25s%s\n", ui.Cyan, v, ui.Reset)
			}
		}
		fmt.Println()

		if ui.PromptYN("Migrate all vaults?", true) {
			toMigrate = vaultNames
		} else {
			vaultName = ui.Prompt("Vault name to migrate", "")
			if vaultName == "" {
				return fmt.Errorf("no vault specified")
			}
			toMigrate = []string{vaultName}
		}
	} else {
		// Validate
		found := false
		for _, v := range vaultNames {
			if v == vaultName {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("vaulted environment '%s' not found", vaultName)
		}
		toMigrate = []string{vaultName}
	}

	fmt.Println()

	for _, v := range toMigrate {
		if err := migrateSingleVault(c, client, v); err != nil {
			ui.Error("Failed to migrate '%s': %s", v, err)
		}
	}

	fmt.Printf("%s%sMigration complete.%s\n\n", ui.Green, ui.Bold, ui.Reset)
	return cmdLs(nil, nil)
}

func migrateSingleVault(c *config.Config, client op.Client, vaultName string) error {
	fmt.Printf("%s--- Migrating: %s ---%s\n\n", ui.Bold, vaultName, ui.Reset)

	if c.ProfileExists(vaultName) {
		ui.Warn("Profile '%s' already exists in vop.", vaultName)
		if !ui.PromptYN("Overwrite it?", false) {
			ui.Info("Skipping %s.", vaultName)
			fmt.Println()
			return nil
		}
	}

	ui.Info("Unlocking vault '%s'... (enter vault password when prompted)", vaultName)
	cmd := exec.Command("vaulted", "dump", vaultName)
	cmd.Stdin = nil // vaulted reads password from tty
	dumpOut, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to dump vault (wrong password?): %w", err)
	}

	var dump vaultedDump
	if err := json.Unmarshal(dumpOut, &dump); err != nil {
		return fmt.Errorf("failed to parse vault dump: %w", err)
	}

	if dump.AWSKey == nil || dump.AWSKey.ID == "" || dump.AWSKey.Secret == "" {
		ui.Warn("Vault '%s' has no AWS credentials. Skipping.", vaultName)
		fmt.Println()
		return nil
	}

	fmt.Printf("  Found AWS key: %s%s%s\n", ui.Cyan, dump.AWSKey.ID, ui.Reset)
	if dump.AWSKey.MFA != "" {
		fmt.Printf("  MFA serial:    %s%s%s\n", ui.Cyan, dump.AWSKey.MFA, ui.Reset)
	}
	if dump.AWSKey.Region != "" {
		fmt.Printf("  Region:        %s%s%s\n", ui.Cyan, dump.AWSKey.Region, ui.Reset)
	}

	if len(dump.Vars) > 0 {
		keys := make([]string, 0, len(dump.Vars))
		for k := range dump.Vars {
			keys = append(keys, k)
		}
		fmt.Printf("  Env vars:      %s%d%s %s(%s)%s\n", ui.Cyan, len(dump.Vars), ui.Reset, ui.Dim, strings.Join(keys, ", "), ui.Reset)
		ui.Warn("Environment variables are not migrated (only AWS credentials).")
	}

	fmt.Println()

	// 1Password account selection
	ui.Info("Select the 1Password account to store these credentials.")
	fmt.Println()

	accounts, err := client.ListAccounts()
	if err == nil && len(accounts) > 0 {
		fmt.Printf("  %sAvailable 1Password accounts:%s\n", ui.Bold, ui.Reset)
		for _, a := range accounts {
			fmt.Printf("  %s  (%s)\n", a.URL, a.Email)
		}
		fmt.Println()
	}

	opAccount := ui.Prompt("1Password account (e.g. my.1password.com)", "")
	if opAccount == "" {
		ui.Error("1Password account required. Skipping.")
		return nil
	}

	if err := client.EnsureSignedIn(opAccount); err != nil {
		return err
	}

	// Vault selection
	fmt.Println()
	ui.Info("Select the 1Password vault to store the item in.")
	fmt.Println()

	vaults, err := client.ListVaults(opAccount)
	if err == nil && len(vaults) > 0 {
		fmt.Printf("  %sAvailable 1Password vaults:%s\n", ui.Bold, ui.Reset)
		for _, v := range vaults {
			fmt.Printf("  %s\n", v.Name)
		}
		fmt.Println()
	}

	opVault := ui.Prompt("1Password vault name", "Private")
	opItemName := ui.Prompt("1Password item name", "AWS - "+vaultName)
	profileName := ui.Prompt("vop profile name", vaultName)

	iamUsername := ""
	if dump.AWSKey.MFA != "" {
		parts := strings.Split(dump.AWSKey.MFA, "/")
		if len(parts) > 1 {
			iamUsername = parts[len(parts)-1]
		}
	}
	iamUsername = ui.Prompt("IAM username", iamUsername)
	description := ui.Prompt("Description", "Migrated from vaulted: "+vaultName)

	fmt.Println()
	ui.Info("Creating 1Password item: %s", opItemName)

	assignments := []string{
		"access key id[text]=" + dump.AWSKey.ID,
		"secret access key[password]=" + dump.AWSKey.Secret,
	}
	if dump.AWSKey.MFA != "" {
		assignments = append(assignments, "mfa serial[text]="+dump.AWSKey.MFA)
	}
	if dump.AWSKey.Region != "" {
		assignments = append(assignments, "region[text]="+dump.AWSKey.Region)
	}

	err = client.CreateItem(opAccount, opVault, "Login", opItemName, "aws,vop,migrated", assignments...)
	if err != nil {
		ui.Error("Failed to create 1Password item: %s", err)
		ui.Warn("You may need to create it manually.")
		return nil
	}
	ui.Success("Created 1Password item: %s", opItemName)

	// TOTP handling
	mfaTOTPItem := ""
	if dump.AWSKey.MFA != "" {
		fmt.Println()
		ui.Warn("Vaulted stores the MFA serial (%s) but not the TOTP seed.", dump.AWSKey.MFA)
		ui.Warn("If you have TOTP configured in a 1Password item, provide its name below.")
		ui.Warn("Otherwise, leave blank and set it up later with 'vop edit %s'.", profileName)
		fmt.Println()
		mfaTOTPItem = ui.Prompt("1Password item with TOTP (blank to skip)", "")
	}

	profile := &config.Profile{
		OPAccount:   opAccount,
		OPItem:      opItemName,
		Description: description,
		MFATOTPItem: mfaTOTPItem,
		IAMUsername: iamUsername,
	}

	c.SetProfile(profileName, profile)
	if err := saveConfig(c); err != nil {
		return err
	}

	ui.Success("Profile '%s' added to vop config.", profileName)
	fmt.Println()
	return nil
}
