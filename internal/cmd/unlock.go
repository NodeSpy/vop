package cmd

import (
	"fmt"

	"github.com/NodeSpy/vop/internal/creds"
	"github.com/NodeSpy/vop/internal/ui"
	"github.com/spf13/cobra"
)

func newUnlockCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unlock <profile>",
		Short: "Clear a profile's rate-limit cooldown",
		Long: `Clear the failure cooldown for a profile.

vop tracks recent auth failures on disk and enforces a backoff
before hitting 1Password again — this prevents rapid retries
(e.g. from an out-of-sync MFA) from triggering 1Password's
account-level rate limit.

Use this command if you've verified the underlying problem is
fixed (system clock synced, TOTP correct) and want to retry
immediately instead of waiting out the cooldown.`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeProfiles,
		RunE:              cmdUnlock,
	}
}

func cmdUnlock(_ *cobra.Command, args []string) error {
	profileName := args[0]

	c, err := loadConfig()
	if err != nil {
		return err
	}
	if _, ok := c.Profiles[profileName]; !ok {
		return fmt.Errorf("unknown profile: %s", profileName)
	}

	creds.ClearFailures(profileName)
	ui.Success("Cooldown cleared for '%s'", profileName)
	return nil
}
