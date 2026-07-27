package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/NodeSpy/vop/internal/creds"
	"github.com/NodeSpy/vop/internal/ui"
	"github.com/spf13/cobra"
)

func newExecCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "exec [profile] <command...>",
		Short:              "Run command with credentials",
		Args:               cobra.MinimumNArgs(1),
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

	if len(args) < 1 {
		return fmt.Errorf("usage: vop exec [-q] [profile] [--] <command...>")
	}

	c, err := loadConfig()
	if err != nil {
		return err
	}

	// A leading "--" means the profile was omitted: everything after it is
	// the command, and the profile comes from the environment or a .vop
	// file. Requiring the separator keeps this unambiguous — without it,
	// `vop exec foo` would be impossible to read as either form.
	var profileName string
	var cmdArgs []string
	if args[0] == "--" {
		cmdArgs = args[1:]
		resolved, err := requireDefaultProfile(nil, "exec")
		if err != nil {
			return err
		}
		profileName = resolved.Name
	} else {
		profileName = args[0]
		cmdArgs = args[1:]
		// Strip the separator in the explicit form (vop exec profile -- cmd).
		if len(cmdArgs) > 0 && cmdArgs[0] == "--" {
			cmdArgs = cmdArgs[1:]
		}
	}

	if len(cmdArgs) == 0 {
		return fmt.Errorf("no command specified")
	}

	profileName, profile, err := resolveProfile(c, profileName)
	if err != nil {
		return err
	}

	// Use cached credentials if they are still valid — no 1Password interaction.
	awsCreds := creds.LoadCached(profileName)

	if awsCreds == nil {
		client, err := getClientForProfile(profile)
		if err != nil {
			return err
		}

		awsCreds, err = creds.Fetch(profile, profileName, client, c, opClientFor())
		if err != nil {
			return err
		}

		// Persist to disk so subsequent exec calls reuse without re-authing.
		if _, _, err := creds.WriteFiles(awsCreds, profileName); err != nil {
			return err
		}

		pushToServer(profileName, awsCreds)
	}

	creds.ExportToEnv(awsCreds, profileName)

	// Ensure AWS_SHARED_CREDENTIALS_FILE points at the on-disk copy.
	// WriteFiles already sets this when doing a fresh fetch; set it here
	// for the cached path where we skipped WriteFiles.
	credFile := creds.CredFilePath(profileName)
	os.Setenv("AWS_SHARED_CREDENTIALS_FILE", credFile)

	ui.Info("Running: %s", cmdArgs)

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}
	return nil
}
