// Package cmd implements all vop CLI commands.
package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/NodeSpy/vop/internal/config"
	"github.com/NodeSpy/vop/internal/creds"
	"github.com/NodeSpy/vop/internal/credserver"
	"github.com/NodeSpy/vop/internal/op"
	"github.com/NodeSpy/vop/internal/ui"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	cfg     *config.Config

	// cliClient is the cached CLI-based 1Password client, used when no
	// service account token is configured (and for commands that don't
	// operate on a specific profile, like check).
	cliClient op.Client
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "vop",
		Short: "AWS credential management via 1Password",
		Long:  "vop -- spawn shells and run commands with AWS credentials fetched from 1Password.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// No args: list profiles. If a profile name is given as the only
			// arg, treat it as `vop shell <profile>` (vaulted-style shorthand).
			return cmdLs(cmd, args)
		},
		ValidArgsFunction: completeProfilesForRoot,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}

	root.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.config/vop/profiles.json)")

	root.AddCommand(newLsCmd())
	root.AddCommand(newProfileCmd())
	root.AddCommand(newShellCmd())
	root.AddCommand(newExecCmd())
	root.AddCommand(newAddCmd())
	root.AddCommand(newEditCmd())
	root.AddCommand(newRmCmd())
	root.AddCommand(newShowCmd())
	root.AddCommand(newDumpCmd())
	root.AddCommand(newRotateCmd())
	root.AddCommand(newTestCmd())
	root.AddCommand(newMigrateCmd())
	root.AddCommand(newCheckCmd())
	root.AddCommand(newVersionCmd())
	if cmd := newUpdateCmd(); cmd != nil {
		root.AddCommand(cmd)
	}
	root.AddCommand(newRefreshCmd())
	root.AddCommand(newCredProcessCmd())
	root.AddCommand(newCatCmd())
	root.AddCommand(newServeCmd())
	root.AddCommand(newAuthCmd())
	root.AddCommand(newUnlockCmd())

	return root
}

// Execute runs the root command.
func Execute() {
	root := NewRootCmd()

	// Vaulted-style shorthand: if the first arg isn't a known command,
	// treat it as a profile name for `shell`. The shell command's
	// resolveProfile handles typos with suggestions and a picker.
	if len(os.Args) > 1 {
		arg := os.Args[1]
		_, _, err := root.Find(os.Args[1:])
		if err != nil && arg != "help" && arg != "completion" &&
			arg != "__complete" && !strings.HasPrefix(arg, "-") {
			// Not a known command or flag — rewrite to shell so the
			// profile resolution (with typo suggestions) handles it.
			newArgs := []string{os.Args[0], "shell"}
			newArgs = append(newArgs, os.Args[1:]...)
			os.Args = newArgs
		}
	}

	if err := root.Execute(); err != nil {
		ui.Error("%s", err)
		os.Exit(exitCodeFor(err))
	}
}

// Exit codes. Anything automated (an agent, a script, CI) can use these to
// tell "this will fix itself if you try again" apart from "stop retrying" —
// blind retry loops against a locked credential store are exactly how a
// profile ends up in a long rate-limit cooldown.
const (
	// ExitFailure is any error without a more specific classification.
	ExitFailure = 1
	// ExitCooldown means vop refused to try because the profile is inside
	// its backoff window. Retrying before it elapses will just repeat this.
	ExitCooldown = 11
	// ExitLocked means the credential source is locked or could not prompt.
	// Requires a human to unlock it; retrying changes nothing.
	ExitLocked = 12
)

func exitCodeFor(err error) int {
	var cooldown *creds.CooldownError
	if errors.As(err, &cooldown) {
		if cooldown.Kind == creds.KindLocked {
			return ExitLocked
		}
		return ExitCooldown
	}
	if creds.IsLockedSourceError(err) {
		return ExitLocked
	}
	return ExitFailure
}

func configFilePath() string {
	if cfgFile != "" {
		return cfgFile
	}
	return config.DefaultConfigFile()
}

func loadConfig() (*config.Config, error) {
	if cfg != nil {
		return cfg, nil
	}
	path := configFilePath()
	c, err := config.Load(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no config file found: %s\n\n  Run %svop add <profile>%s to get started", path, ui.Bold, ui.Reset)
		}
		return nil, err
	}
	cfg = c
	return cfg, nil
}

func ensureConfig() (*config.Config, error) {
	path := configFilePath()
	created, err := config.EnsureConfigFile(path)
	if err != nil {
		return nil, err
	}
	if created {
		ui.Info("Created config: %s", path)
	}
	return config.Load(path)
}

func saveConfig(c *config.Config) error {
	return config.Save(configFilePath(), c)
}

// getClientForProfile returns the appropriate op.Client for the given profile.
// If the profile has a service account token, it returns an SDK client;
// otherwise it returns a CLI client. Returns (nil, nil) when the profile
// doesn't need 1Password at all (aws-credentials-file source + external
// TOTP command).
func getClientForProfile(profile *config.Profile) (op.Client, error) {
	if !profile.UsesOnePassword() {
		return nil, nil
	}
	if profile.UsesSDK() {
		return op.NewSDK(profile.ServiceAccountToken, profile.OPVault)
	}
	client := getCLIClient()
	if !client.IsInstalled() {
		return nil, fmt.Errorf("the 1Password CLI (op) is not installed.\n  Install it: https://developer.1password.com/docs/cli/get-started/\n  Or configure this profile with a service account token instead")
	}
	return client, nil
}

// getCLIClient returns the cached CLI-based op.Client.
func getCLIClient() op.Client {
	if cliClient == nil {
		cliClient = op.NewCLI()
	}
	return cliClient
}

// opClientFor returns a client-resolver function suitable for passing to
// creds.Fetch as the clientFor argument. It delegates to getClientForProfile
// so assumed-role source profiles use the correct 1Password backend.
func opClientFor() func(*config.Profile) (op.Client, error) {
	return getClientForProfile
}

// resolveProfile looks up a profile by exact name, and if not found,
// suggests the closest match and falls back to the interactive picker.
// Returns the resolved profile name, profile, and any error.
func resolveProfile(c *config.Config, name string) (string, *config.Profile, error) {
	if name == "" {
		return "", nil, fmt.Errorf("profile name required")
	}
	if p, ok := c.Profiles[name]; ok {
		return name, p, nil
	}

	// Not an exact match — suggest the closest and show the picker
	names := c.ProfileNames()
	if len(names) == 0 {
		return "", nil, fmt.Errorf("unknown profile: %s (no profiles configured)", name)
	}

	closest, dist := c.ClosestProfile(name)

	// Only suggest if the match is reasonably close (within ~half the length)
	if dist > 0 && dist <= len(name)/2+1 {
		ui.Warn("Unknown profile '%s'. Did you mean '%s'?", name, closest)
	} else {
		ui.Warn("Unknown profile '%s'.", name)
	}

	fmt.Println()
	selected, selErr := ui.Select("Profile", names)
	if selErr != nil {
		return "", nil, selErr
	}
	fmt.Println()

	p, ok := c.Profiles[selected]
	if !ok {
		return "", nil, fmt.Errorf("unknown profile: %s", selected)
	}
	return selected, p, nil
}

func requireProfile(c *config.Config, name string) (*config.Profile, error) {
	_, p, err := resolveProfile(c, name)
	return p, err
}

// pushToServer sends credentials to the running credential server as a
// side effect. Returns true if the server was reached. Failures are
// silently ignored -- the push is best-effort.
func pushToServer(profileName string, awsCreds *creds.AWSCredentials) bool {
	client := credserver.NewClient()
	if client == nil {
		return false
	}
	if err := client.PushCreds(profileName, awsCreds); err != nil {
		return false
	}
	return true
}

// completeProfiles provides tab-completion for profile names.
func completeProfiles(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	path := configFilePath()
	c, err := config.Load(path)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return c.ProfileNames(), cobra.ShellCompDirectiveNoFileComp
}

// completeProfilesForRoot provides tab-completion for the root command,
// offering profile names alongside subcommands (for vop <profile> shorthand).
func completeProfilesForRoot(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	path := configFilePath()
	c, err := config.Load(path)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return c.ProfileNames(), cobra.ShellCompDirectiveNoFileComp
}
