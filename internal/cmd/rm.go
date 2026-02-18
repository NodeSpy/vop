package cmd

import (
	"github.com/NodeSpy/vop/internal/ui"
	"github.com/spf13/cobra"
)

func newRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "rm <profile>",
		Aliases:           []string{"remove"},
		Short:             "Remove a profile",
		Args:              cobra.ExactArgs(1),
		RunE:              cmdRm,
		ValidArgsFunction: completeProfiles,
	}
}

func cmdRm(_ *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	name := args[0]
	if _, err := requireProfile(c, name); err != nil {
		return err
	}

	_ = showProfile(c, name)

	if !ui.PromptYN("Delete profile '"+name+"'?", false) {
		ui.Info("Cancelled.")
		return nil
	}

	c.DeleteProfile(name)
	if err := saveConfig(c); err != nil {
		return err
	}

	ui.Success("Profile '%s' deleted.", name)
	return nil
}
