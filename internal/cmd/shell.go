package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/NodeSpy/vop/internal/creds"
	"github.com/NodeSpy/vop/internal/ui"
	"github.com/spf13/cobra"
)

func newShellCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "shell <profile>",
		Short: "Spawn sub-shell with credentials",
		Args:  cobra.ExactArgs(1),
		RunE:  cmdShell,
	}
}

func cmdShell(_ *cobra.Command, args []string) error {
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

	awsCreds, err := creds.Fetch(profile, profileName, client)
	if err != nil {
		return err
	}

	creds.ExportToEnv(awsCreds, profileName)

	credFile, jsonFile, err := creds.WriteFiles(awsCreds, profileName)
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
	fmt.Println()
	fmt.Printf("  %sFor external tools / AI agents:%s\n", ui.Bold, ui.Reset)
	fmt.Printf("  %sAWS credentials file:%s  %s\n", ui.Dim, ui.Reset, credFile)
	fmt.Printf("  %sJSON credentials file:%s %s\n", ui.Dim, ui.Reset, jsonFile)
	fmt.Println()
	fmt.Printf("  %sexport AWS_SHARED_CREDENTIALS_FILE=%s%s\n", ui.Dim, credFile, ui.Reset)
	fmt.Println()

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	cmd := exec.Command(shell)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	_ = cmd.Run()

	ui.Info("Shell exited. Cleaning up credentials.")
	creds.CleanupFiles(profileName)
	return nil
}
