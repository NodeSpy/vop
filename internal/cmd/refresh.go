package cmd

import (
	"fmt"
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

If no profile is specified, vop uses VOP_PROFILE (or
AGENT_DECK_VOP_PROFILE), then a .vop file in this directory or an
ancestor up to the repo root.
This re-fetches credentials from the profile's source, performs MFA/STS if
configured, and updates the tmpfs credential files.

Any tool reading AWS_SHARED_CREDENTIALS_FILE will pick up the
new credentials automatically.`,
		Args: cobra.MaximumNArgs(1),
		RunE: cmdRefresh,
	}
}

func cmdRefresh(_ *cobra.Command, args []string) error {
	resolved, err := requireDefaultProfile(args, "refresh")
	if err != nil {
		return err
	}
	profileName := resolved.Name

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
	awsCreds, err := creds.Fetch(profile, profileName, client, c, opClientFor())
	if err != nil {
		return err
	}

	// Push to credential server if running (best-effort).
	pushToServer(profileName, awsCreds)

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
