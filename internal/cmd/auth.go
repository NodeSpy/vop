package cmd

import (
	"fmt"

	"github.com/NodeSpy/vop/internal/creds"
	"github.com/NodeSpy/vop/internal/ui"
	"github.com/spf13/cobra"
)

func newAuthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "auth <profile>",
		Short: "Authenticate and push credentials to the credential server",
		Long: `Fetch credentials from 1Password for the given profile and push
them to the running credential server. This is the primary way to
load credentials into the server for Docker containers to consume.

If no credential server is running, credentials are fetched and
displayed but not pushed.`,
		Args:              cobra.ExactArgs(1),
		RunE:              cmdAuth,
		ValidArgsFunction: completeProfiles,
	}
}

func cmdAuth(_ *cobra.Command, args []string) error {
	profileName := args[0]

	c, err := loadConfig()
	if err != nil {
		return err
	}

	profileName, profile, err := resolveProfile(c, profileName)
	if err != nil {
		return err
	}

	client, err := getClientForProfile(profile)
	if err != nil {
		return err
	}

	awsCreds, err := creds.Fetch(profile, profileName, client)
	if err != nil {
		return err
	}

	pushed := pushToServer(profileName, awsCreds)
	if pushed {
		fmt.Println()
		ui.Success("Credentials for '%s' pushed to credential server.", profileName)
		if awsCreds.Expiration != "" {
			ui.Info("Expires: %s", awsCreds.Expiration)
		}
		fmt.Println()
	} else {
		fmt.Println()
		ui.Warn("No credential server running. Start one with: vop serve start")
		ui.Info("Credentials fetched for '%s' but not pushed anywhere.", profileName)
		fmt.Println()
	}

	return nil
}
