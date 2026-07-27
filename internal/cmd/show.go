package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/NodeSpy/vop/internal/config"
	"github.com/NodeSpy/vop/internal/ui"
	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "show [profile]",
		Short:             "Show profile details",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeProfiles,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := loadConfig()
			if err != nil {
				return err
			}
			resolved, err := requireDefaultProfile(args, "show")
			if err != nil {
				return err
			}
			name, _, err := resolveProfile(c, resolved.Name)
			if err != nil {
				return err
			}
			return showProfile(c, name)
		},
	}
}

func showProfile(c *config.Config, name string) error {
	p := c.Profiles[name]
	data, err := json.MarshalIndent(p, "  ", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("\n  %s%s%s\n", ui.Bold, name, ui.Reset)
	fmt.Printf("  %s\n\n", string(data))
	return nil
}
