//go:build !noupdate

package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/NodeSpy/vop/internal/ui"
	"github.com/NodeSpy/vop/internal/version"
	"github.com/spf13/cobra"
)

const installURL = "https://raw.githubusercontent.com/NodeSpy/vop/main/install.sh"

func newUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update vop to the latest version",
		RunE:  cmdUpdate,
	}
}

func cmdUpdate(_ *cobra.Command, _ []string) error {
	fmt.Printf("Current version: %s\n\n", version.Full())

	if !ui.PromptYN("Download and install the latest version?", true) {
		ui.Info("Update cancelled.")
		return nil
	}

	fmt.Println()
	ui.Info("Downloading latest version...")

	cmd := exec.Command("bash", "-c", fmt.Sprintf("curl -fsSL %s | bash", installURL))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	fmt.Println()
	ui.Success("Update complete.")
	return nil
}
