package cmd

import (
	"context"
	"fmt"

	"github.com/NodeSpy/vop/internal/awsclient"
	"github.com/NodeSpy/vop/internal/ui"
	"github.com/spf13/cobra"
)

func newTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test <profile>",
		Short: "Test 1Password + AWS connectivity",
		Args:  cobra.ExactArgs(1),
		RunE:  cmdTest,
	}
}

func cmdTest(_ *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	name := args[0]
	profile, err := requireProfile(c, name)
	if err != nil {
		return err
	}

	client, err := getClientForProfile(profile)
	if err != nil {
		return err
	}

	fmt.Println()
	ui.Info("Testing profile: %s", name)

	if profile.UsesSDK() {
		ui.Info("Backend: 1Password SDK (service account)")
	} else {
		ui.Info("Backend: op CLI")
	}

	// 1Password sign-in
	if err := client.EnsureSignedIn(profile.OPAccount); err != nil {
		return fmt.Errorf("sign-in failed for %s: %w", profile.OPAccount, err)
	}
	ui.Success("Signed into 1Password: %s", profile.OPAccount)

	// Item access
	ak, err := client.ReadField(profile.OPAccount, profile.OPItem, "access key id")
	if err != nil {
		ui.Error("Cannot read item '%s' from '%s'", profile.OPItem, profile.OPAccount)
		ui.Warn("Check the item name and that it has an 'access key id' field.")
		return err
	}
	ui.Success("Can read item: %s", profile.OPItem)

	// MFA
	if profile.MFATOTPItem != "" {
		_, err := client.GetTOTP(profile.OPAccount, profile.MFATOTPItem)
		if err != nil {
			ui.Error("Cannot get TOTP from '%s'", profile.MFATOTPItem)
			ui.Warn("Check item name and TOTP configuration.")
		} else {
			ui.Success("Can read TOTP from: %s", profile.MFATOTPItem)
		}
	}

	// AWS credentials
	ui.Info("Testing AWS credentials...")
	sk, err := client.ReadField(profile.OPAccount, profile.OPItem, "secret access key")
	if err != nil {
		return err
	}

	identity, err := awsclient.GetCallerIdentity(context.Background(), ak, sk)
	if err != nil {
		ui.Error("AWS credentials failed")
		ui.Warn("Credentials in 1Password may be expired or invalid.")
		return err
	}

	ui.Success("AWS credentials valid:")
	fmt.Printf("    Account: %s\n", identity.Account)
	fmt.Printf("    ARN:     %s\n", identity.Arn)
	fmt.Println()

	return nil
}
