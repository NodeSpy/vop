package cmd

import (
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
	}
}

func cmdExec(_ *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	profileName := args[0]
	cmdArgs := args[1:]

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
