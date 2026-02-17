package cmd

import (
	"github.com/NodeSpy/vop/internal/config"
	"github.com/NodeSpy/vop/internal/ui"
	"github.com/spf13/cobra"
)

func newEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit <profile>",
		Short: "Edit an existing profile",
		Args:  cobra.ExactArgs(1),
		RunE:  cmdEdit,
	}
}

func cmdEdit(_ *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	name := args[0]
	profile, err := requireProfile(c, name)
	if err != nil {
		return err
	}

	_ = showProfile(c, name)
	ui.Info("Press Enter to keep current value.")

	opAccount := ui.Prompt("1Password account", profile.OPAccount)
	opItem := ui.Prompt("1Password item", profile.OPItem)
	opVault := ui.Prompt("1Password vault", profile.OPVault)
	description := ui.Prompt("Description", profile.Description)
	mfaTOTPItem := ui.Prompt("MFA TOTP item (blank to remove)", profile.MFATOTPItem)
	iamUsername := ui.Prompt("IAM username (blank to remove)", profile.IAMUsername)
	serviceAccountToken := ui.Prompt("Service account token (blank to use op CLI)", profile.ServiceAccountToken)

	updated := &config.Profile{
		OPAccount:           opAccount,
		OPItem:              opItem,
		OPVault:             opVault,
		Description:         description,
		MFATOTPItem:         mfaTOTPItem,
		IAMUsername:         iamUsername,
		ServiceAccountToken: serviceAccountToken,
	}

	c.SetProfile(name, updated)
	if err := saveConfig(c); err != nil {
		return err
	}

	ui.Success("Profile '%s' updated.", name)
	return showProfile(c, name)
}
