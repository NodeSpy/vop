package cmd

import (
	"fmt"

	"github.com/NodeSpy/vop/internal/ui"
	"github.com/spf13/cobra"
)

func newLsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "ls",
		Aliases: []string{"list"},
		Short:   "List profiles",
		RunE:    cmdLs,
	}
}

func cmdLs(_ *cobra.Command, _ []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	names := c.ProfileNames()
	if len(names) == 0 {
		fmt.Printf("\n  %sNo profiles configured.%s\n", ui.Dim, ui.Reset)
		fmt.Printf("  Run %svop add <name>%s to create one.\n\n", ui.Bold, ui.Reset)
		return nil
	}

	fmt.Println()
	fmt.Printf("  %s%-25s %-30s %s%s\n", ui.Bold, "PROFILE", "1PASSWORD ACCOUNT", "DESCRIPTION", ui.Reset)
	fmt.Printf("  %-25s %-30s %s\n", "-------", "-----------------", "-----------")
	for _, name := range names {
		p := c.Profiles[name]
		desc := p.Description
		if desc == "" {
			desc = "—"
		}
		fmt.Printf("  %s%-25s%s %-30s %s\n", ui.Cyan, name, ui.Reset, p.OPAccount, desc)
	}
	fmt.Println()
	return nil
}
