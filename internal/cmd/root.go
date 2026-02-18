// Package cmd implements all vop CLI commands.
package cmd

import (
	"fmt"
	"os"

	"github.com/NodeSpy/vop/internal/config"
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
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.config/vop/profiles.json)")

	root.AddCommand(newLsCmd())
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
	root.AddCommand(newAgentCmd())
	root.AddCommand(newCatCmd())

	return root
}

// Execute runs the root command.
func Execute() {
	root := NewRootCmd()

	// Vaulted-style shorthand: if the first arg isn't a known command,
	// try treating it as a profile name for `shell`.
	if len(os.Args) > 1 {
		_, _, err := root.Find(os.Args[1:])
		if err != nil {
			// Not a known command -- check if it's a profile name
			profileName := os.Args[1]
			path := configFilePath()
			if c, loadErr := config.Load(path); loadErr == nil && c.ProfileExists(profileName) {
				// Rewrite args: vop <profile> -> vop shell <profile>
				newArgs := []string{os.Args[0], "shell"}
				newArgs = append(newArgs, os.Args[1:]...)
				os.Args = newArgs
			}
		}
	}

	if err := root.Execute(); err != nil {
		ui.Error("%s", err)
		os.Exit(1)
	}
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
// otherwise it returns a CLI client.
func getClientForProfile(profile *config.Profile) (op.Client, error) {
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

func requireProfile(c *config.Config, name string) (*config.Profile, error) {
	if name == "" {
		return nil, fmt.Errorf("profile name required")
	}
	p, ok := c.Profiles[name]
	if !ok {
		return nil, fmt.Errorf("unknown profile: %s", name)
	}
	return p, nil
}
