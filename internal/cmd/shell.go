package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/NodeSpy/vop/internal/config"
	"github.com/NodeSpy/vop/internal/creds"
	"github.com/NodeSpy/vop/internal/op"
	"github.com/NodeSpy/vop/internal/ui"
	"github.com/spf13/cobra"
)

func newShellCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shell [profile]",
		Short: "Spawn sub-shell with credentials",
		Args:  cobra.MaximumNArgs(1),
		RunE:  cmdShell,
	}
	cmd.Flags().Bool("no-refresh", false, "Disable automatic credential refresh")
	cmd.Flags().BoolP("quiet", "q", false, "Suppress all informational output")
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

	profileName := ""
	if len(args) > 0 {
		profileName = args[0]
	}

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

	profile, err := requireProfile(c, profileName)
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

	creds.ExportToEnv(awsCreds, profileName)

	_, _, err = creds.WriteFiles(awsCreds, profileName)
	if err != nil {
		return fmt.Errorf("failed to write credential files: %w", err)
	}

	// Ensure cleanup
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		<-sigCh
		creds.CleanupFiles(profileName)
		os.Exit(0)
	}()

	ui.Info("Spawning sub-shell for '%s'. Type 'exit' to leave.", profileName)
	ui.Info("Run 'vop agent' for credential paths and AI agent instructions.")
	fmt.Println()

	// Start background auto-refresh if credentials have an expiration
	noRefresh, _ := cmd.Flags().GetBool("no-refresh")
	stopRefresh := make(chan struct{})
	if !noRefresh && awsCreds.Expiration != "" {
		go autoRefresh(stopRefresh, profile, profileName, client, awsCreds.Expiration)
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
	ui.Info("Shell exited. Cleaning up credentials.")
	creds.CleanupFiles(profileName)
	return nil
}

// autoRefresh runs in a goroutine and refreshes credentials before they expire.
// It calculates the refresh time as 75% of the remaining TTL, with a minimum
// of 60 seconds and a maximum wait of 6 hours. After each refresh it
// recalculates based on the new expiration.
func autoRefresh(stop <-chan struct{}, profile *config.Profile, profileName string, client op.Client, expiration string) {
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
		newCreds, err := creds.Fetch(profile, profileName, client)
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
