//go:build noupdate

package cmd

import "github.com/spf13/cobra"

// newUpdateCmd returns nil when built with the noupdate tag.
// Package-manager builds (Homebrew, AUR) use this tag so the
// update command does not exist at all.
func newUpdateCmd() *cobra.Command {
	return nil
}
