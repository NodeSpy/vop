package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/NodeSpy/vop/internal/awsclient"
	"github.com/NodeSpy/vop/internal/ui"
	"github.com/spf13/cobra"
)

func newRotateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rotate <profile>",
		Short: "Rotate AWS access keys",
		Args:  cobra.ExactArgs(1),
		RunE:  cmdRotate,
	}
}

func cmdRotate(_ *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	profileName := args[0]
	profile, err := requireProfile(c, profileName)
	if err != nil {
		return err
	}

	client, err := getClientForProfile(profile)
	if err != nil {
		return err
	}

	if err := client.EnsureSignedIn(profile.OPAccount); err != nil {
		return err
	}

	ctx := context.Background()

	// 1. Fetch current credentials
	ui.Info("Fetching current credentials from: %s", profile.OPItem)
	oldAK, err := client.ReadField(profile.OPAccount, profile.OPItem, "access key id")
	if err != nil {
		return err
	}
	oldSK, err := client.ReadField(profile.OPAccount, profile.OPItem, "secret access key")
	if err != nil {
		return err
	}
	ui.Info("Current access key: %s", oldAK)

	// 2. Create new key
	ui.Info("Creating new access key...")
	newKey, err := awsclient.CreateAccessKey(ctx, oldAK, oldSK, profile.IAMUsername)
	if err != nil {
		return err
	}

	newAK := newKey.AccessKeyID
	newSK := newKey.SecretAccessKey
	if newAK == "" || newSK == "" {
		return fmt.Errorf("AWS returned empty credentials")
	}
	ui.Info("New access key: %s", newAK)

	// 3. Update 1Password
	ui.Info("Updating 1Password item...")
	err = client.EditItem(profile.OPAccount, profile.OPItem,
		"access key id="+newAK,
		"secret access key="+newSK,
	)
	if err != nil {
		return fmt.Errorf("failed to update 1Password item: %w", err)
	}
	ui.Success("1Password updated.")

	// 4. Test new credentials
	ui.Info("Waiting for propagation...")
	time.Sleep(5 * time.Second)

	identity, testErr := awsclient.GetCallerIdentity(ctx, newAK, newSK)

	if testErr != nil {
		// ROLLBACK
		ui.Error("New credentials failed verification!")
		ui.Warn("Rolling back 1Password...")

		_ = client.EditItem(profile.OPAccount, profile.OPItem,
			"access key id="+oldAK,
			"secret access key="+oldSK,
		)

		ui.Warn("Deleting failed key: %s", newAK)
		_ = awsclient.DeleteAccessKey(ctx, oldAK, oldSK, newAK, profile.IAMUsername)

		return fmt.Errorf("rotation aborted. Old credentials restored")
	}

	ui.Success("New credentials verified:")
	fmt.Printf("    Account: %s\n", identity.Account)
	fmt.Printf("    ARN:     %s\n", identity.Arn)

	// 5. Delete old key
	ui.Info("Deleting old access key: %s", oldAK)
	if err := awsclient.DeleteAccessKey(ctx, newAK, newSK, oldAK, profile.IAMUsername); err != nil {
		ui.Warn("Failed to delete old key %s — clean up manually.", oldAK)
	}

	fmt.Println()
	fmt.Printf("%s%sRotation complete for '%s'.%s\n", ui.Green, ui.Bold, profileName, ui.Reset)
	fmt.Printf("  Old: %s%s%s (deleted)\n", ui.Red, oldAK, ui.Reset)
	fmt.Printf("  New: %s%s%s (active)\n\n", ui.Green, newAK, ui.Reset)
	return nil
}
