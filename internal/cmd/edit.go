package cmd

import (
	"fmt"

	"github.com/NodeSpy/vop/internal/config"
	"github.com/NodeSpy/vop/internal/ui"
	"github.com/spf13/cobra"
)

func newEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "edit <profile>",
		Short:             "Edit an existing profile",
		Args:              cobra.ExactArgs(1),
		RunE:              cmdEdit,
		ValidArgsFunction: completeProfiles,
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
	fieldPrefix := ui.Prompt("Field prefix (blank for none)", profile.FieldPrefix)
	mfaTOTPItem := ui.Prompt("MFA TOTP item (blank to remove, same item name for co-located)", profile.MFATOTPItem)
	iamUsername := ui.Prompt("IAM username (blank to remove)", profile.IAMUsername)
	serviceAccountToken := ui.Prompt("Service account token (blank to use op CLI)", profile.ServiceAccountToken)

	agentDefault := profile.AgentPolicy
	if agentDefault == "" {
		agentDefault = "readonly"
	}
	agentPolicy := ui.Prompt("Agent policy (readonly/full/custom string)", agentDefault)
	if agentPolicy == "readonly" {
		agentPolicy = "" // omit default from config
	}

	// Preserve existing field map, offer to edit it
	fieldMap := profile.FieldMap
	if len(fieldMap) > 0 {
		fmt.Println()
		ui.Info("Current field mappings:")
		for base, label := range fieldMap {
			fmt.Printf("    %s -> %s\n", base, label)
		}
		if ui.PromptYN("Edit field mappings?", false) {
			fieldMap = make(map[string]string)
			fieldMap["access key id"] = ui.Prompt("Field name for access key ID", profile.FieldMap["access key id"])
			fieldMap["secret access key"] = ui.Prompt("Field name for secret access key", profile.FieldMap["secret access key"])
			mfaDefault := profile.FieldMap["mfa serial"]
			mfaField := ui.Prompt("Field name for MFA serial (blank to skip)", mfaDefault)
			if mfaField != "" {
				fieldMap["mfa serial"] = mfaField
			}
		}
	} else if ui.PromptYN("Configure custom field name mappings?", false) {
		fieldMap = make(map[string]string)
		fieldMap["access key id"] = ui.Prompt("Field name for access key ID", "access key id")
		fieldMap["secret access key"] = ui.Prompt("Field name for secret access key", "secret access key")
		mfaField := ui.Prompt("Field name for MFA serial (blank to skip)", "")
		if mfaField != "" {
			fieldMap["mfa serial"] = mfaField
		}
	}

	updated := &config.Profile{
		OPAccount:           opAccount,
		OPItem:              opItem,
		OPVault:             opVault,
		Description:         description,
		FieldPrefix:         fieldPrefix,
		FieldMap:            fieldMap,
		MFATOTPItem:         mfaTOTPItem,
		IAMUsername:         iamUsername,
		ServiceAccountToken: serviceAccountToken,
		AgentPolicy:         agentPolicy,
	}

	c.SetProfile(name, updated)
	if err := saveConfig(c); err != nil {
		return err
	}

	ui.Success("Profile '%s' updated.", name)
	return showProfile(c, name)
}
