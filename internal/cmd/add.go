package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/NodeSpy/vop/internal/awsclient"
	"github.com/NodeSpy/vop/internal/config"
	"github.com/NodeSpy/vop/internal/op"
	"github.com/NodeSpy/vop/internal/ui"
	"github.com/spf13/cobra"
)

func newAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add [profile]",
		Short: "Add a new profile",
		Args:  cobra.MaximumNArgs(1),
		RunE:  cmdAdd,
	}
}

func cmdAdd(_ *cobra.Command, args []string) error {
	c, err := ensureConfig()
	if err != nil {
		return err
	}

	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	if name == "" {
		name = ui.Prompt("Profile name (e.g. mycompany-work)", "")
	}
	if name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}

	if c.ProfileExists(name) {
		return fmt.Errorf("profile '%s' already exists. Use 'vop edit %s' to modify it", name, name)
	}

	fmt.Println()

	// Choose backend: service account token (SDK) or op CLI
	useSDK := ui.PromptYN("Use a service account token? (no = use op CLI)", false)

	var client op.Client
	var serviceAccountToken, opVault, opAccount string

	if useSDK {
		serviceAccountToken = ui.Prompt("Service account token", "")
		if serviceAccountToken == "" {
			return fmt.Errorf("service account token cannot be empty")
		}
		opVault = ui.Prompt("1Password vault name", "")
		if opVault == "" {
			return fmt.Errorf("vault name is required when using a service account token")
		}
		sdkClient, sdkErr := op.NewSDK(serviceAccountToken, opVault)
		if sdkErr != nil {
			return fmt.Errorf("failed to initialize SDK client: %w", sdkErr)
		}
		client = sdkClient
	} else {
		opAccount = ui.Prompt("1Password account (e.g. my.1password.com)", "")
		if opAccount == "" {
			return fmt.Errorf("1Password account cannot be empty")
		}
		cliClient := getCLIClient()
		if !cliClient.IsInstalled() {
			return fmt.Errorf("the 1Password CLI (op) is not installed.\n  Install it: https://developer.1password.com/docs/cli/get-started/\n  Or re-run this command and choose the service account token option instead")
		}
		client = cliClient
	}

	opItem := ui.Prompt("1Password item name for AWS credentials", "")
	if opItem == "" {
		return fmt.Errorf("1Password item cannot be empty")
	}

	description := ui.Prompt("Description", "")
	iamUsername := ui.Prompt("IAM username (blank = caller identity)", "")

	fmt.Println()
	fieldOptions := []string{
		"Use vop-prefixed field names (recommended)",
		"Use standard field names (no prefix)",
		"Map custom field names on the item",
	}
	fieldChoice, fieldErr := ui.Select("How are credential fields named?", fieldOptions)
	if fieldErr != nil {
		return fieldErr
	}

	fieldPrefix := ""
	var fieldMap map[string]string

	switch fieldChoice {
	case fieldOptions[0]:
		fieldPrefix = "vop."
	case fieldOptions[1]:
		fieldPrefix = ""
	case fieldOptions[2]:
		fieldMap = make(map[string]string)
		fieldMap["access key id"] = ui.Prompt("Field name for access key ID", "access key id")
		fieldMap["secret access key"] = ui.Prompt("Field name for secret access key", "secret access key")
		mfaField := ui.Prompt("Field name for MFA serial (blank to skip)", "")
		if mfaField != "" {
			fieldMap["mfa serial"] = mfaField
		}
	}

	// Helper to resolve the effective field name for a base name.
	fn := func(base string) string {
		if fieldMap != nil {
			if mapped, ok := fieldMap[base]; ok {
				return mapped
			}
		}
		if fieldPrefix == "" {
			return base
		}
		return fieldPrefix + base
	}

	// --- MFA setup ---
	mfaTOTPItem := ""
	mfaSerial := ""

	if ui.PromptYN("Configure MFA/TOTP?", false) {
		// Sign in so we can read credentials
		if err := client.EnsureSignedIn(opAccount); err != nil {
			return err
		}

		// Read the access key to look up MFA devices via AWS IAM
		akField := fn("access key id")
		skField := fn("secret access key")

		accessKey, err := client.ReadField(opAccount, opItem, akField)
		if err != nil {
			return fmt.Errorf("failed to read access key from 1Password: %w", err)
		}
		secretKey, err := client.ReadField(opAccount, opItem, skField)
		if err != nil {
			return fmt.Errorf("failed to read secret key from 1Password: %w", err)
		}

		ui.Info("Looking up MFA devices via AWS IAM...")
		serials, mfaErr := awsclient.ListAllMFADevices(
			context.Background(),
			accessKey, secretKey,
			iamUsername,
		)
		if mfaErr != nil {
			return fmt.Errorf("failed to list MFA devices: %w", mfaErr)
		}

		if len(serials) == 1 {
			mfaSerial = serials[0]
			ui.Success("Found MFA device: %s", mfaSerial)
		} else {
			fmt.Println()
			selected, selErr := ui.Select("Select MFA device", serials)
			if selErr != nil {
				return selErr
			}
			mfaSerial = selected
		}

		// Derive IAM username from MFA serial if not already set
		if iamUsername == "" {
			parts := strings.Split(mfaSerial, "/")
			if len(parts) > 1 {
				iamUsername = parts[len(parts)-1]
			}
		}

		// Store the MFA serial on the 1Password item
		assignment := fn("mfa serial") + "[text]=" + mfaSerial
		ui.Info("Storing MFA serial on '%s'...", opItem)
		if editErr := client.EditItem(opAccount, opItem, assignment); editErr != nil {
			ui.Warn("Could not store MFA serial on 1Password item: %s", editErr)
			ui.Info("You may need to add it manually.")
		} else {
			ui.Success("MFA serial saved to 1Password item.")
		}

		// Link TOTP item — same pattern as migrate
		fmt.Println()
		totpOptions := []string{
			"Same item (" + opItem + ")",
			"Different item",
			"Skip (no TOTP configured)",
		}
		totpChoice, totpErr := ui.Select("Where is the TOTP seed for MFA?", totpOptions)
		if totpErr != nil {
			return totpErr
		}
		switch totpChoice {
		case totpOptions[0]:
			mfaTOTPItem = opItem
		case totpOptions[1]:
			// Try to list items in the vault for a picker
			items, _ := client.ListItems(opAccount, opVault)
			otherOptions := make([]string, 0, len(items))
			for _, it := range items {
				if it.Title != opItem {
					otherOptions = append(otherOptions, it.Title)
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
				mfaTOTPItem = ui.Prompt("1Password item containing TOTP", "")
			}
		}
	}

	// --- Role assumption setup ---
	roleARN := ""
	sourceProfile := ""
	roleSessionName := ""
	externalID := ""

	fmt.Println()
	if ui.PromptYN("Assume a role? (for cross-account or delegated access)", false) {
		roleARN = ui.Prompt("Role ARN (e.g. arn:aws:iam::123456789012:role/MyRole)", "")
		if roleARN == "" {
			return fmt.Errorf("role ARN cannot be empty when role assumption is enabled")
		}

		existingNames := c.ProfileNames()
		// Remove the current profile name in case it was added in a prior partial run.
		filtered := existingNames[:0]
		for _, n := range existingNames {
			if n != name {
				filtered = append(filtered, n)
			}
		}

		if len(filtered) > 0 {
			fmt.Println()
			selected, selErr := ui.Select("Source profile (holds the base credentials for AssumeRole)", filtered)
			if selErr != nil {
				return selErr
			}
			sourceProfile = selected
		} else {
			sourceProfile = ui.Prompt("Source profile name (must already exist in vop)", "")
		}
		if sourceProfile == "" {
			return fmt.Errorf("source profile cannot be empty")
		}
		if sourceProfile == name {
			return fmt.Errorf("source_profile cannot be the same as the profile being created")
		}

		roleSessionName = ui.Prompt("Session name (blank = 'vop')", "")
		externalID = ui.Prompt("External ID (blank = none)", "")
	}

	profile := &config.Profile{
		OPAccount:           opAccount,
		OPItem:              opItem,
		OPVault:             opVault,
		Description:         description,
		FieldPrefix:         fieldPrefix,
		FieldMap:            fieldMap,
		MFATOTPItem:         mfaTOTPItem,
		IAMUsername:         iamUsername,
		ServiceAccountToken: serviceAccountToken,
		RoleARN:             roleARN,
		SourceProfile:       sourceProfile,
		RoleSessionName:     roleSessionName,
		ExternalID:          externalID,
	}

	c.SetProfile(name, profile)
	if err := saveConfig(c); err != nil {
		return err
	}

	ui.Success("Profile '%s' added.", name)
	return showProfile(c, name)
}
