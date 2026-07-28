package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/NodeSpy/vop/internal/config"
	"github.com/NodeSpy/vop/internal/creds"
	"github.com/NodeSpy/vop/internal/op"
	"github.com/NodeSpy/vop/internal/ui"
	"github.com/spf13/cobra"
)

func newShellCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "shell [profile]",
		Short:             "Spawn sub-shell with credentials",
		Args:              cobra.MaximumNArgs(1),
		RunE:              cmdShell,
		ValidArgsFunction: completeProfiles,
	}
	cmd.Flags().Bool("no-refresh", false, "Disable automatic credential refresh")
	cmd.Flags().BoolP("quiet", "q", false, "Suppress all informational output")
	cmd.Flags().Bool("serve", false, "Start credential server if not already running")
	return cmd
}

func cmdShell(cmd *cobra.Command, args []string) error {
	quiet, _ := cmd.Flags().GetBool("quiet")
	if quiet {
		ui.Quiet = true
	}

	c, err := loadConfig()
	if err != nil {
		return err
	}

	// Start credential server if --serve is set and no server is running.
	serve, _ := cmd.Flags().GetBool("serve")
	if serve {
		if err := ensureServer(cmd); err != nil {
			ui.Warn("Failed to start credential server: %s", err)
		}
	}

	resolved := profileOrDefault(args)
	announceProfile(resolved)
	profileName := resolved.Name

	if profileName == "" {
		names := c.ProfileNames()
		if len(names) == 0 {
			return fmt.Errorf("no profiles configured. Run 'vop add' to create one")
		}
		if len(names) == 1 {
			profileName = names[0]
		} else {
			fmt.Println()
			selected, selErr := ui.Select("Profile", names)
			if selErr != nil {
				return selErr
			}
			profileName = selected
			fmt.Println()
		}
	}

	profileName, profile, err := resolveProfile(c, profileName)
	if err != nil {
		return err
	}

	// Use cached credentials if they are still valid — no 1Password interaction.
	awsCreds := creds.LoadCached(profileName)

	var client op.Client
	if awsCreds == nil {
		var err error
		client, err = getClientForProfile(profile)
		if err != nil {
			return err
		}

		awsCreds, err = creds.Fetch(profile, profileName, client, c, opClientFor())
		if err != nil {
			return err
		}

		if _, _, err := creds.WriteFiles(awsCreds, profileName); err != nil {
			return fmt.Errorf("failed to write credential files: %w", err)
		}

		pushToServer(profileName, awsCreds)
	}

	// Set profile metadata but do NOT export AWS credential env vars.
	// Credentials are served via AWS_SHARED_CREDENTIALS_FILE (set by
	// WriteFiles) so that 'vop refresh' and autoRefresh can update them
	// on disk without needing to modify the shell's environment.
	os.Setenv("VOP_PROFILE", profileName)
	os.Setenv("VAULTED_ENV", profileName)
	os.Unsetenv("AWS_DEFAULT_REGION")
	creds.UnsetCredEnvVars()
	os.Setenv("AWS_SHARED_CREDENTIALS_FILE", creds.CredFilePath(profileName))

	ui.Info("Spawning sub-shell for '%s'. Type 'exit' to leave.", profileName)
	fmt.Println()

	// Start background auto-refresh if credentials have an expiration.
	// Requires the op client; skip if we used the cache and didn't build one.
	noRefresh, _ := cmd.Flags().GetBool("no-refresh")
	stopRefresh := make(chan struct{})
	if !noRefresh && awsCreds.Expiration != "" && client != nil {
		go autoRefresh(stopRefresh, profile, profileName, client, c, awsCreds.Expiration)
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	shellCmd := exec.Command(shell)
	shellCmd.Stdin = os.Stdin
	shellCmd.Stdout = os.Stdout
	shellCmd.Stderr = os.Stderr
	shellCmd.Env = os.Environ()

	_ = shellCmd.Run()

	close(stopRefresh)
	ui.Info("Shell exited.")
	return nil
}

// autoRefresh runs in a goroutine and refreshes credentials before they expire.
// It calculates the refresh time as 75% of the remaining TTL, with a minimum
// of 60 seconds and a maximum wait of 6 hours. After each refresh it
// recalculates based on the new expiration.
func autoRefresh(stop <-chan struct{}, profile *config.Profile, profileName string, client op.Client, cfg *config.Config, expiration string) {
	for {
		refreshAt := calcRefreshTime(expiration)
		if refreshAt <= 0 {
			// Already expired or unparseable — try refreshing immediately
			refreshAt = 5 * time.Second
		}

		select {
		case <-stop:
			return
		case <-time.After(refreshAt):
		}

		// Check if we should still be running
		select {
		case <-stop:
			return
		default:
		}

		// Suppress informational output during background refresh
		ui.Quiet = true
		newCreds, err := creds.Fetch(profile, profileName, client, cfg, opClientFor())
		ui.Quiet = false

		if err != nil {
			ui.Warn("Auto-refresh failed: %s", err)
			ui.Warn("Run 'vop refresh' manually, or exit and re-enter the shell.")
			return
		}

		if _, _, err := creds.WriteFiles(newCreds, profileName); err != nil {
			ui.Warn("Auto-refresh: failed to write credential files: %s", err)
			return
		}

		// Push refreshed creds to server if running (best-effort).
		pushToServer(profileName, newCreds)

		ui.Success("Credentials auto-refreshed (expires: %s)", newCreds.Expiration)
		expiration = newCreds.Expiration
	}
}

// calcRefreshTime returns how long to wait before refreshing credentials.
// It aims for 75% of the remaining TTL.
func calcRefreshTime(expiration string) time.Duration {
	exp, err := time.Parse(time.RFC3339, expiration)
	if err != nil {
		// Try alternate format
		exp, err = time.Parse("2006-01-02T15:04:05Z", expiration)
		if err != nil {
			return 0
		}
	}

	remaining := time.Until(exp)
	if remaining <= 0 {
		return 0
	}

	refresh := time.Duration(float64(remaining) * 0.75)

	// Clamp to reasonable bounds
	const minWait = 60 * time.Second
	const maxWait = 6 * time.Hour
	if refresh < minWait {
		refresh = minWait
	}
	if refresh > maxWait {
		refresh = maxWait
	}
	return refresh
}
