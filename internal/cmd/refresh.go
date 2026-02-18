package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/NodeSpy/vop/internal/creds"
	"github.com/NodeSpy/vop/internal/ui"
	"github.com/spf13/cobra"
)

func newRefreshCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "refresh [profile]",
		Short:             "Refresh credentials in an active vop shell",
		ValidArgsFunction: completeProfiles,
		Long: `Refresh AWS credentials for the active vop session.

If no profile is specified, the current VOP_PROFILE is used.
This re-fetches credentials from 1Password, performs MFA/STS if
configured, and updates the tmpfs credential files.

Any tool reading AWS_SHARED_CREDENTIALS_FILE will pick up the
new credentials automatically.`,
		Args: cobra.MaximumNArgs(1),
		RunE: cmdRefresh,
	}
}

func cmdRefresh(_ *cobra.Command, args []string) error {
	profileName := ""
	if len(args) > 0 {
		profileName = args[0]
	}
	if profileName == "" {
		profileName = os.Getenv("VOP_PROFILE")
	}
	if profileName == "" {
		return fmt.Errorf("not in a vop shell (VOP_PROFILE not set).\n  Specify a profile: vop refresh <profile>")
	}

	c, err := loadConfig()
	if err != nil {
		return err
	}

	profile, err := requireProfile(c, profileName)
	if err != nil {
		return err
	}

	client, err := getClientForProfile(profile)
	if err != nil {
		return err
	}

	start := time.Now()
	awsCreds, err := creds.Fetch(profile, profileName, client)
	if err != nil {
		return err
	}

	// Update tmpfs credential files
	credFile, jsonFile, err := creds.WriteFiles(awsCreds, profileName)
	if err != nil {
		return fmt.Errorf("failed to write credential files: %w", err)
	}

	elapsed := time.Since(start).Round(time.Millisecond)

	fmt.Println()
	ui.Success("Credentials refreshed for '%s' (%s)", profileName, elapsed)
	fmt.Printf("  %sCredentials file:%s %s\n", ui.Dim, ui.Reset, credFile)
	fmt.Printf("  %sJSON file:%s       %s\n", ui.Dim, ui.Reset, jsonFile)

	if awsCreds.Expiration != "" {
		fmt.Printf("  %sExpires:%s         %s\n", ui.Dim, ui.Reset, awsCreds.Expiration)
	}
	fmt.Println()

	return nil
}
