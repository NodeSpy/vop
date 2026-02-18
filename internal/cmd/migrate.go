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
		if err := migrateSingleVault(c, v); err != nil {
			ui.Error("Failed to migrate '%s': %s", v, err)
		}
	}

	fmt.Printf("%s%sMigration complete.%s\n\n", ui.Green, ui.Bold, ui.Reset)
	return cmdLs(nil, nil)
}

func migrateSingleVault(c *config.Config, vaultName string) error {
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

	// --- Choose backend before contacting 1Password ---
	useSDK := ui.PromptYN("Use a service account token? (no = use op CLI)", false)

	var client op.Client
	var serviceAccountToken, opVault, opAccount string
	var vaults []op.OPVault

	if useSDK {
		serviceAccountToken = ui.Prompt("Service account token", "")
		if serviceAccountToken == "" {
			return fmt.Errorf("service account token cannot be empty")
		}
		opVault = ui.Prompt("1Password vault name", "")
		if opVault == "" {
			return fmt.Errorf("vault name is required when using a service account token")
		}

		sdkClient, err := op.NewSDK(serviceAccountToken, opVault)
		if err != nil {
			return fmt.Errorf("failed to initialize SDK client: %w", err)
		}
		client = sdkClient

		vaults, _ = client.ListVaults("")
	} else {
		cliClient := getCLIClient()
		if !cliClient.IsInstalled() {
			return fmt.Errorf("the 1Password CLI (op) is not installed.\n  Install it: https://developer.1password.com/docs/cli/get-started/\n  Or re-run this command and choose the service account token option instead")
		}
		client = cliClient

		// Select 1Password account
		accounts, err := cliClient.ListAccounts()
		if err != nil || len(accounts) == 0 {
			return fmt.Errorf("no 1Password accounts found. Run: op account add")
		}

		accountOptions := make([]string, len(accounts))
		for i, a := range accounts {
			accountOptions[i] = fmt.Sprintf("%s  (%s)", a.URL, a.Email)
		}

		fmt.Println()
		selected, selErr := ui.Select("1Password account", accountOptions)
		if selErr != nil {
			return selErr
		}
		// Extract the URL part (before the first space)
		opAccount = strings.Fields(selected)[0]
		// Convert URL to account address (remove https://)
		opAccount = strings.TrimPrefix(opAccount, "https://")

		if err := client.EnsureSignedIn(opAccount); err != nil {
			return err
		}

		vaults, _ = client.ListVaults(opAccount)
	}

	// --- Vault selection ---
	if !useSDK {
		if len(vaults) == 0 {
			return fmt.Errorf("no vaults found for account %s", opAccount)
		}

		vaultNames := make([]string, len(vaults))
		for i, v := range vaults {
			vaultNames[i] = v.Name
		}

		fmt.Println()
		selectedVault, selErr := ui.Select("1Password vault", vaultNames)
		if selErr != nil {
			return selErr
		}
		opVault = selectedVault
	}

	// --- Item: select existing or create new ---
	items, _ := client.ListItems(opAccount, opVault)
	itemTitles := make([]string, len(items))
	for i, it := range items {
		itemTitles[i] = it.Title
	}

	opItemName := ""
	useExisting := false

	if len(itemTitles) > 0 {
		fmt.Println()
		selected, isNew, selErr := ui.SelectOrCreate("1Password item", itemTitles, "+ Create new item")
		if selErr != nil {
			return selErr
		}
		if isNew {
			opItemName = ui.Prompt("New item name", "AWS - "+vaultName)
		} else {
			opItemName = selected
			useExisting = true
		}
	} else {
		opItemName = ui.Prompt("New 1Password item name", "AWS - "+vaultName)
	}

	profileName := ui.Prompt("vop profile name", vaultName)

	// Derive IAM username from MFA serial if available (no need to prompt).
	iamUsername := ""
	if dump.AWSKey.MFA != "" {
		parts := strings.Split(dump.AWSKey.MFA, "/")
		if len(parts) > 1 {
			iamUsername = parts[len(parts)-1]
		}
	}

	// --- Determine how credential fields are named ---
	fieldPrefix := "vop."
	var fieldMap map[string]string
	writeFields := true // whether we need to write credential values to 1Password

	if useExisting {
		fmt.Println()
		options := []string{
			"Map existing fields on this item",
			"Write new vop-prefixed fields",
			"Write new fields (standard names, no prefix)",
		}
		choice, selErr := ui.Select("How should vop find credentials on this item?", options)
		if selErr != nil {
			return selErr
		}

		switch choice {
		case options[0]:
			// --- Map existing fields ---
			writeFields = false
			fieldPrefix = ""
			fieldMap = make(map[string]string)

			fields, fErr := client.ListFields(opAccount, opItemName)
			if fErr != nil {
				ui.Warn("Could not list fields: %s", fErr)
				ui.Info("Falling back to manual entry.")

				fieldMap["access key id"] = ui.Prompt("Field name for access key ID", "")
				fieldMap["secret access key"] = ui.Prompt("Field name for secret access key", "")
				if dump.AWSKey.MFA != "" {
					mfaField := ui.Prompt("Field name for MFA serial (blank to skip)", "")
					if mfaField != "" {
						fieldMap["mfa serial"] = mfaField
					}
				}
			} else {
				fieldLabels := make([]string, len(fields))
				for i, f := range fields {
					fieldLabels[i] = f.Label
				}

				fmt.Println()
				ui.Info("Select which field contains each credential:")

				akField, akErr := ui.Select("Access Key ID", fieldLabels)
				if akErr != nil {
					return akErr
				}
				fieldMap["access key id"] = akField

				fmt.Println()
				skField, skErr := ui.Select("Secret Access Key", fieldLabels)
				if skErr != nil {
					return skErr
				}
				fieldMap["secret access key"] = skField

				if dump.AWSKey.MFA != "" {
					fmt.Println()
					mfaOptions := append([]string{"(skip)"}, fieldLabels...)
					mfaField, mfaErr := ui.Select("MFA Serial", mfaOptions)
					if mfaErr != nil {
						return mfaErr
					}
					if mfaField != "(skip)" {
						fieldMap["mfa serial"] = mfaField
					}
				}
			}

			fmt.Println()
			ui.Success("Field mapping configured:")
			for base, label := range fieldMap {
				fmt.Printf("    %s -> %s\n", base, label)
			}

		case options[1]:
			fieldPrefix = "vop."
		case options[2]:
			fieldPrefix = ""
		}
	}

	// Helper to resolve the effective field name for a base name.
	fn := func(base string) string {
		if fieldMap != nil {
			if mapped, ok := fieldMap[base]; ok {
				return mapped
			}
		}
		return fieldPrefix + base
	}

	fmt.Println()

	if writeFields {
		// Build field assignments for the 1Password item.
		// op item edit adds new fields or updates existing ones by label —
		// it does NOT overwrite other fields on the item (URLs, notes, etc.).
		assignments := []string{
			fn("access key id") + "[text]=" + dump.AWSKey.ID,
			fn("secret access key") + "[password]=" + dump.AWSKey.Secret,
		}
		if dump.AWSKey.MFA != "" {
			assignments = append(assignments, fn("mfa serial")+"[text]="+dump.AWSKey.MFA)
		}
		if dump.AWSKey.Region != "" {
			assignments = append(assignments, fn("region")+"[text]="+dump.AWSKey.Region)
		}

		if useExisting {
			ui.Info("Adding AWS credential fields to '%s'...", opItemName)
			ui.Info("Only these fields will be added/updated: %s, %s%s",
				fn("access key id"), fn("secret access key"),
				func() string {
					extra := ""
					if dump.AWSKey.MFA != "" {
						extra += ", " + fn("mfa serial")
					}
					if dump.AWSKey.Region != "" {
						extra += ", " + fn("region")
					}
					return extra
				}())
			fmt.Println()

			if !ui.PromptYN("Continue?", true) {
				ui.Info("Skipping.")
				return nil
			}

			if editErr := client.EditItem(opAccount, opItemName, assignments...); editErr != nil {
				ui.Error("Failed to update 1Password item: %s", editErr)
				ui.Warn("You may need to update it manually.")
				return nil
			}
			ui.Success("Updated 1Password item: %s", opItemName)
		} else {
			ui.Info("Creating 1Password item: %s", opItemName)
			if createErr := client.CreateItem(opAccount, opVault, "Login", opItemName, "aws,vop", assignments...); createErr != nil {
				ui.Error("Failed to create 1Password item: %s", createErr)
				ui.Warn("You may need to create it manually.")
				return nil
			}
			ui.Success("Created 1Password item: %s", opItemName)
		}
	} else {
		ui.Info("Using existing fields on '%s' — no changes written to 1Password.", opItemName)
	}

	// TOTP handling
	mfaTOTPItem := ""
	if dump.AWSKey.MFA != "" {
		fmt.Println()
		totpOptions := []string{
			"Same item (" + opItemName + ")",
			"Different item",
			"Skip (no TOTP configured)",
		}
		totpChoice, totpErr := ui.Select("Where is the TOTP seed for MFA?", totpOptions)
		if totpErr != nil {
			return totpErr
		}
		switch totpChoice {
		case totpOptions[0]:
			mfaTOTPItem = opItemName
		case totpOptions[1]:
			otherOptions := make([]string, 0, len(itemTitles))
			for _, t := range itemTitles {
				if t != opItemName {
					otherOptions = append(otherOptions, t)
				}
			}
			if len(otherOptions) > 0 {
				fmt.Println()
				sel, selErr := ui.Select("TOTP item", otherOptions)
				if selErr != nil {
					return selErr
				}
				mfaTOTPItem = sel
			} else {
				mfaTOTPItem = ui.Prompt("TOTP item name", "")
			}
		}
	}

	profile := &config.Profile{
		OPAccount:           opAccount,
		OPItem:              opItemName,
		OPVault:             opVault,
		FieldPrefix:         fieldPrefix,
		FieldMap:            fieldMap,
		MFATOTPItem:         mfaTOTPItem,
		IAMUsername:         iamUsername,
		ServiceAccountToken: serviceAccountToken,
	}

	c.SetProfile(profileName, profile)
	if err := saveConfig(c); err != nil {
		return err
	}

	ui.Success("Profile '%s' added to vop config.", profileName)
	fmt.Println()
	return nil
}
