package cmd

import (
	"fmt"

	"github.com/NodeSpy/vop/internal/config"
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
	} else {
		opAccount = ui.Prompt("1Password account (e.g. my.1password.com)", "")
		if opAccount == "" {
			return fmt.Errorf("1Password account cannot be empty")
		}
	}

	opItem := ui.Prompt("1Password item name for AWS credentials", "")
	if opItem == "" {
		return fmt.Errorf("1Password item cannot be empty")
	}

	description := ui.Prompt("Description", "")

	mfaTOTPItem := ""
	if ui.PromptYN("Configure MFA/TOTP?", false) {
		totpOptions := []string{
			"Same item (" + opItem + ")",
			"Different item",
		}
		totpChoice, totpErr := ui.Select("Where is the TOTP seed?", totpOptions)
		if totpErr != nil {
			return totpErr
		}
		if totpChoice == totpOptions[0] {
			mfaTOTPItem = opItem
		} else {
			mfaTOTPItem = ui.Prompt("1Password item containing TOTP", "")
		}
	}

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
	}

	c.SetProfile(name, profile)
	if err := saveConfig(c); err != nil {
		return err
	}

	ui.Success("Profile '%s' added.", name)
	return showProfile(c, name)
}
