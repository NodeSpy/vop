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

func newExecCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "exec <profile> <command...>",
		Short:              "Run command with credentials",
		Args:               cobra.MinimumNArgs(2),
		DisableFlagParsing: true,
		RunE:               cmdExec,
		ValidArgsFunction:  completeProfiles,
	}
}

func cmdExec(_ *cobra.Command, args []string) error {
	// DisableFlagParsing is set, so we handle -q/--quiet manually.
	var filtered []string
	for _, a := range args {
		if a == "-q" || a == "--quiet" {
			ui.Quiet = true
		} else {
			filtered = append(filtered, a)
		}
	}
	args = filtered

	if len(args) < 2 {
		return fmt.Errorf("usage: vop exec [-q] <profile> [--] <command...>")
	}

	c, err := loadConfig()
	if err != nil {
		return err
	}

	profileName := args[0]
	cmdArgs := args[1:]

	// Strip leading "--" separator if present (e.g. vop exec profile -- cmd).
	if len(cmdArgs) > 0 && cmdArgs[0] == "--" {
		cmdArgs = cmdArgs[1:]
	}
	if len(cmdArgs) == 0 {
		return fmt.Errorf("no command specified")
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

	// Push to credential server if running (best-effort).
	pushToServer(profileName, awsCreds)

	creds.ExportToEnv(awsCreds, profileName)

	_, _, err = creds.WriteFiles(awsCreds, profileName)
	if err != nil {
		return err
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		<-sigCh
		creds.CleanupFiles(profileName)
	}()

	ui.Info("Running: %s", cmdArgs)

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	err = cmd.Run()
	creds.CleanupFiles(profileName)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}
	return nil
}
