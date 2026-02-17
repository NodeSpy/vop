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
		mfaTOTPItem = ui.Prompt("1Password item containing TOTP", "")
	}

	iamUsername := ui.Prompt("IAM username (blank = caller identity)", "")

	profile := &config.Profile{
		OPAccount:           opAccount,
		OPItem:              opItem,
		OPVault:             opVault,
		Description:         description,
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
